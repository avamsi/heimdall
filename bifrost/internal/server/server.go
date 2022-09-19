package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/rs/xid"
	"golang.org/x/exp/constraints"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/avamsi/ergo"

	pb "github.com/avamsi/heimdall/bifrost/proto"
)

func anyOf[T any](s []T, pred func(T) bool) bool {
	for _, i := range s {
		if pred(i) {
			return true
		}
	}
	return false
}

func max[T constraints.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
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
	sync.Cond
	any     *pb.CacheCommandResponse
	success *pb.CacheCommandResponse
	ttl     chan time.Duration
}

type bifrost struct {
	pb.UnimplementedBifrostServer
	config          Config
	notifier        Notifier
	msgs            chan string
	syncRunningCmds struct {
		sync.Mutex
		m map[string]command // string is the command ID
	}
	syncCachedCmds struct {
		sync.Mutex
		m map[string]*syncCachedCommand // string is the full command
	}
}

func (b *bifrost) commandStartAsync(req *pb.CommandStartRequest, id string) {
	cmd := req.GetCommand()
	cmd.Id = id
	b.syncRunningCmds.Lock()
	defer b.syncRunningCmds.Unlock()
	b.syncRunningCmds.m[id] = command{cmd, make(chan nothing)}
}

func (b *bifrost) CommandStart(todo context.Context, req *pb.CommandStartRequest) (*pb.CommandStartResponse, error) {
	id := req.GetCommand().GetId()
	if id == "" {
		id = xid.New().String()
	}
	go b.commandStartAsync(req, id)
	return &pb.CommandStartResponse{Id: id}, nil
}

func (b *bifrost) startTime(cmd *pb.Command) *timestamppb.Timestamp {
	if !cmd.ProtoReflect().Has(cmd.ProtoReflect().Descriptor().Fields().ByNumber(2)) {
		b.syncRunningCmds.Lock()
		defer b.syncRunningCmds.Unlock()
		if cmd, ok := b.syncRunningCmds.m[cmd.GetId()]; ok {
			return cmd.GetStartTime()
		}
		return nil
	}
	return cmd.GetStartTime()
}

func (b *bifrost) commandEndAsync(req *pb.CommandEndRequest) {
	defer func() {
		b.syncRunningCmds.Lock()
		defer b.syncRunningCmds.Unlock()
		if cmd, ok := b.syncRunningCmds.m[req.GetCommand().GetId()]; ok {
			// This unblocks any goroutines waiting in WaitForCommand below.
			close(cmd.done)
			delete(b.syncRunningCmds.m, cmd.GetId())
		}
	}()
	// Don't notify if the command was interrupted by the user.
	if req.GetReturnCode() == 130 {
		return
	}
	cmd := req.GetCommand()
	if cmd.GetCommand() == "" {
		return
	}
	isPrefixOfCmd := func(prefix string) bool {
		return strings.HasPrefix(cmd.GetCommand(), prefix)
	}
	// Don't notify if the user requested to never be notified for the command
	// (unless the user requested to always be notified for it).
	alwaysNotify := anyOf(b.config.AlwaysNotifyCommands(), isPrefixOfCmd)
	if !alwaysNotify && anyOf(b.config.NeverNotifyCommands(), isPrefixOfCmd) {
		return
	}
	// Don't notify if the command ran or the user interacted with it (i.e.,
	// the command accessed stdin) in the last 42 seconds.
	t := b.startTime(cmd)
	if t == nil {
		return
	}
	start := t.AsTime().Local()
	interaction := time.Since(start).Round(time.Second)
	if i := req.GetLastInteractionTime().AsTime(); i.After(start) {
		interaction = time.Since(start).Round(time.Second)
	}
	// TODO: this "42" should be configurable and not a magic number.
	if interaction < 42*time.Second {
		return
	}
	ts := start.Format(time.Kitchen)
	ds := time.Since(start).Round(time.Second).String()
	b.msgs <- fmt.Sprintf("```[%s + %s -> %d] $ %s```", ts, ds, req.GetReturnCode(), cmd.GetCommand())
}

func (b *bifrost) CommandEnd(todo context.Context, req *pb.CommandEndRequest) (*pb.CommandEndResponse, error) {
	go b.commandEndAsync(req)
	return &pb.CommandEndResponse{}, nil
}

func (b *bifrost) ListCommands(todo context.Context, _ *pb.ListCommandsRequest) (*pb.ListCommandsResponse, error) {
	b.syncRunningCmds.Lock()
	defer b.syncRunningCmds.Unlock()
	cmds := []*pb.Command{}
	for _, cmd := range b.syncRunningCmds.m {
		cmds = append(cmds, cmd.Command)
	}
	return &pb.ListCommandsResponse{Commands: cmds}, nil
}

func (b *bifrost) WaitForCommand(ctx context.Context, req *pb.WaitForCommandRequest) (*pb.WaitForCommandResponse, error) {
	b.syncRunningCmds.Lock()
	var done chan nothing
	if cmd, ok := b.syncRunningCmds.m[req.GetId()]; ok {
		done = cmd.done
	}
	b.syncRunningCmds.Unlock()
	if done != nil {
		select {
		// Block till caller gives up or this command is done running (i.e., next CommandEnd call).
		case <-done:
		case <-ctx.Done():
		}
	}
	return &pb.WaitForCommandResponse{}, nil
}

func runCommand(cmd exec.Cmd) (*pb.CacheCommandResponse, error) {
	out, err := cmd.Output()
	resp := &pb.CacheCommandResponse{Stdout: string(out), ReturnTime: timestamppb.Now()}
	exitErr := &exec.ExitError{}
	if err != nil && errors.As(err, &exitErr) {
		err = nil
		resp.Stderr = string(exitErr.Stderr)
		resp.ReturnCode = int32(exitErr.ExitCode())
	}
	// TODO: should we panic on other non-exit errors?
	return resp, err
}

func (b *bifrost) cacheCommandAsync(cmd *exec.Cmd, syncCachedCmd *syncCachedCommand) {
	minTTL := time.Duration(4.2 * float64(time.Second))
	ttl := max(<-syncCachedCmd.ttl, minTTL)
	for {
		resp, err := runCommand(*cmd)
		syncCachedCmd.Lock()
		syncCachedCmd.any = resp
		if err == nil {
			syncCachedCmd.success = resp
		}
		syncCachedCmd.Broadcast()
		syncCachedCmd.Unlock()
		select {
		case <-time.After(ttl):
			continue
		case newTTL, ok := <-syncCachedCmd.ttl:
			if !ok {
				return
			}
			ttl = min(ttl, max(newTTL, minTTL))
			// TODO: ideally we'd continue sleeping here?
		}
	}
}

func (b *bifrost) CacheCommand(todo context.Context, req *pb.CacheCommandRequest) (*pb.CacheCommandResponse, error) {
	cmd := exec.Command(req.GetCommand(), req.GetArgs()...)
	cmdKey := cmd.String()
	b.syncCachedCmds.Lock()
	syncCachedCmd, ok := b.syncCachedCmds.m[cmdKey]
	ttl := time.Duration(req.GetWithin()) * time.Second
	if !ok {
		syncCachedCmd = &syncCachedCommand{}
		syncCachedCmd.Cond.L = &syncCachedCmd.Mutex
		syncCachedCmd.ttl = make(chan time.Duration)
		b.syncCachedCmds.m[cmdKey] = syncCachedCmd
		go b.cacheCommandAsync(cmd, syncCachedCmd)
	}
	b.syncCachedCmds.Unlock()
	syncCachedCmd.Lock()
	defer syncCachedCmd.Unlock()
	syncCachedCmd.ttl <- ttl
	if time.Since(syncCachedCmd.success.GetReturnTime().AsTime()) <= ttl {
		return syncCachedCmd.success, nil
	} else if req.GetAny() && time.Since(syncCachedCmd.any.GetReturnTime().AsTime()) <= ttl {
		return syncCachedCmd.any, nil
	}
	syncCachedCmd.Wait()
	return syncCachedCmd.any, nil
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
	b.syncRunningCmds.m = map[string]command{}
	b.syncCachedCmds.m = map[string]*syncCachedCommand{}
	gs := grpc.NewServer()
	pb.RegisterBifrostServer(gs, b)
	return &server{addr: fmt.Sprintf("localhost:%d", c.BifrostPort()), b: b, gs: gs}
}
