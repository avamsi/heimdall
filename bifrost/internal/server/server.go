package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/rs/xid"
	"google.golang.org/grpc"

	"github.com/avamsi/ergo"

	pb "github.com/avamsi/heimdall/bifrost/proto"
)

func anyOf[T any](slice []T, predicate func(T) bool) bool {
	for _, i := range slice {
		if predicate(i) {
			return true
		}
	}
	return false
}

type Config interface {
	BifrostPort() int
	AlwaysNotifyCommands() []string
	NeverNotifyCommands() []string
}

type Notifier interface {
	Notify(ctx context.Context, msg string) (err error)
}

type nothing struct{}

type command struct {
	*pb.Command
	done chan nothing
}

type syncCachedCommand struct {
	sync.Mutex
	r   *pb.CacheCommandResponse
	err error
}

type bifrost struct {
	pb.UnimplementedBifrostServer
	config   Config
	notifier Notifier
	msgs     chan string
	syncCmds struct {
		sync.Mutex
		m map[string]command // string is the ID returned by Preexec
	}
	syncCmdsCache struct {
		sync.Mutex
		m map[string]*syncCachedCommand // string is the full command
	}
}

func (b *bifrost) preexecAsync(req *pb.PreexecRequest, id string) {
	cmd := req.GetCommand()
	cmd.Id = id
	b.syncCmds.Lock()
	defer b.syncCmds.Unlock()
	b.syncCmds.m[id] = command{cmd, make(chan nothing)}
}

func (b *bifrost) Preexec(todo context.Context, req *pb.PreexecRequest) (*pb.PreexecResponse, error) {
	id := xid.New().String()
	go b.preexecAsync(req, id)
	return &pb.PreexecResponse{Id: id}, nil
}

func (b *bifrost) precmdAsync(req *pb.PrecmdRequest) {
	defer func() {
		b.syncCmds.Lock()
		defer b.syncCmds.Unlock()
		if cmd, ok := b.syncCmds.m[req.GetCommand().GetId()]; ok {
			// This unblocks any goroutines waiting in WaitForCommand below.
			close(cmd.done)
			delete(b.syncCmds.m, cmd.GetId())
		}
	}()
	// Don't notify if the command was interrupted by the user.
	if req.GetReturnCode() == 130 {
		return
	}
	cmd := req.GetCommand()
	isPrefixOfCmd := func(prefix string) bool {
		return strings.HasPrefix(cmd.GetCommand(), prefix)
	}
	// Don't notify if the command ran for less than 42 seconds / user requested to never be
	// notified for it (unless the user requested to always be notified for it).
	alwaysNotify := anyOf(b.config.AlwaysNotifyCommands(), isPrefixOfCmd)
	neverNotify := anyOf(b.config.NeverNotifyCommands(), isPrefixOfCmd)
	t := cmd.GetPreexecTime().AsTime().Local()
	d := time.Since(t).Round(time.Second)
	if !alwaysNotify && (neverNotify || d < 42*time.Second) {
		return
	}
	b.msgs <- fmt.Sprintf("```[%s + %s] $ %s -> %d```", t.Format(time.Kitchen), d, cmd.GetCommand(), req.GetReturnCode())
}

func (b *bifrost) Precmd(todo context.Context, req *pb.PrecmdRequest) (*pb.PrecmdResponse, error) {
	go b.precmdAsync(req)
	return &pb.PrecmdResponse{}, nil
}

func (b *bifrost) ListCommands(todo context.Context, _ *pb.ListCommandsRequest) (*pb.ListCommandsResponse, error) {
	b.syncCmds.Lock()
	defer b.syncCmds.Unlock()
	cmds := []*pb.Command{}
	for _, cmd := range b.syncCmds.m {
		cmds = append(cmds, cmd.Command)
	}
	return &pb.ListCommandsResponse{Commands: cmds}, nil
}

func (b *bifrost) WaitForCommand(ctx context.Context, req *pb.WaitForCommandRequest) (*pb.WaitForCommandResponse, error) {
	b.syncCmds.Lock()
	var done chan nothing
	if cmd, ok := b.syncCmds.m[req.GetId()]; ok {
		done = cmd.done
	}
	b.syncCmds.Unlock()
	if done != nil {
		select {
		// Block till caller gives up or this command is done running (i.e., next Precmd call).
		case <-done:
		case <-ctx.Done():
		}
	}
	return &pb.WaitForCommandResponse{}, nil
}

func runCommand(req *pb.CacheCommandRequest) (*pb.CacheCommandResponse, error) {
	out, err := exec.Command(req.GetCommand(), req.GetArgs()...).Output()
	resp := &pb.CacheCommandResponse{Stdout: string(out)}
	exitErr := &exec.ExitError{}
	if err != nil && errors.As(err, &exitErr) {
		err = nil
		resp.Stderr = string(exitErr.Stderr)
		resp.ReturnCode = int32(exitErr.ExitCode())
	}
	return resp, err
}

func (b *bifrost) cacheCommandAsync(req *pb.CacheCommandRequest) {
	for {
		time.Sleep(time.Duration(math.Max(4.2, float64(req.GetTtl()))) * time.Second)
		resp, err := runCommand(req)
		cmd := exec.Command(req.GetCommand(), req.GetArgs()...).String()
		b.syncCmdsCache.Lock()
		syncCachedCmd, ok := b.syncCmdsCache.m[cmd]
		if !ok {
			b.syncCmdsCache.Unlock()
			return
		}
		syncCachedCmd.Lock()
		b.syncCmdsCache.Unlock()
		syncCachedCmd.r, syncCachedCmd.err = resp, err
		syncCachedCmd.Unlock()
	}
}

func (b *bifrost) CacheCommand(todo context.Context, req *pb.CacheCommandRequest) (*pb.CacheCommandResponse, error) {
	cmd := exec.Command(req.GetCommand(), req.GetArgs()...).String()
	b.syncCmdsCache.Lock()
	syncCachedCmd, ok := b.syncCmdsCache.m[cmd]
	if !ok {
		syncCachedCmd = &syncCachedCommand{}
	}
	b.syncCmdsCache.m[cmd] = syncCachedCmd
	syncCachedCmd.Lock()
	b.syncCmdsCache.Unlock()
	defer syncCachedCmd.Unlock()
	if !ok {
		syncCachedCmd.r, syncCachedCmd.err = runCommand(req)
		go b.cacheCommandAsync(req)
	}
	return syncCachedCmd.r, syncCachedCmd.err
}

type server struct {
	addr string
	b    *bifrost
	gs   *grpc.Server
}

func (s *server) Addr() string {
	return s.addr
}

func (s *server) notify() {
	for {
		msg, ok := <-s.b.msgs
		if !ok {
			return
		}
		if err := s.b.notifier.Notify(context.TODO(), msg); err != nil {
			log.Println(err.Error())
			ergo.Must0(exec.Command("tput", "bel").Run())
		}
	}
}

func (s *server) Start() (err error) {
	defer ergo.Annotate(&err, "failed to start the server")
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	go s.notify()
	defer close(s.b.msgs)
	return s.gs.Serve(lis)
}

func (s *server) Stop() {
	s.gs.GracefulStop()
}

func New(c Config, notifier Notifier) *server {
	b := &bifrost{config: c, notifier: notifier, msgs: make(chan string, 42)}
	b.syncCmds.m = map[string]command{}
	b.syncCmdsCache.m = map[string]*syncCachedCommand{}
	gs := grpc.NewServer()
	pb.RegisterBifrostServer(gs, b)
	return &server{addr: fmt.Sprintf("localhost:%d", c.BifrostPort()), b: b, gs: gs}
}
