package server

import (
	"context"
	"fmt"
	"log"
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

type command struct {
	*pb.Command
	done chan struct{}
}

type bifrost struct {
	pb.UnimplementedBifrostServer
	config Config
	sync   struct {
		sync.Mutex
		notifier Notifier
		cmds     map[string]command
	}
}

func (b *bifrost) preexecAsync(req *pb.PreexecRequest, id string) {
	cmd := req.GetCommand()
	cmd.Id = id
	b.sync.Lock()
	defer b.sync.Unlock()
	b.sync.cmds[id] = command{cmd, make(chan struct{})}
}

func (b *bifrost) Preexec(todo context.Context, req *pb.PreexecRequest) (*pb.PreexecResponse, error) {
	id := xid.New().String()
	go b.preexecAsync(req, id)
	return &pb.PreexecResponse{Id: id}, nil
}

func (b *bifrost) precmdAsync(req *pb.PrecmdRequest) {
	defer func() {
		b.sync.Lock()
		defer b.sync.Unlock()
		if cmd, ok := b.sync.cmds[req.GetCommand().GetId()]; ok {
			// This unblocks any goroutines waiting in WaitForCommand below.
			close(cmd.done)
			delete(b.sync.cmds, cmd.GetId())
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
	msg := fmt.Sprintf("```[%s + %s] $ %s -> %d```", t.Format(time.Kitchen), d, cmd.GetCommand(), req.GetReturnCode())
	b.sync.Lock()
	err := b.sync.notifier.Notify(context.TODO(), msg)
	b.sync.Unlock()
	if err != nil {
		log.Println(err)
		ergo.Must0(exec.Command("tput", "bel").Run())
	}
}

func (b *bifrost) Precmd(todo context.Context, req *pb.PrecmdRequest) (*pb.PrecmdResponse, error) {
	go b.precmdAsync(req)
	return &pb.PrecmdResponse{}, nil
}

func (b *bifrost) ListCommands(todo context.Context, _ *pb.ListCommandsRequest) (*pb.ListCommandsResponse, error) {
	b.sync.Lock()
	defer b.sync.Unlock()
	cmds := []*pb.Command{}
	for _, cmd := range b.sync.cmds {
		cmds = append(cmds, cmd.Command)
	}
	return &pb.ListCommandsResponse{Commands: cmds}, nil
}

func (b *bifrost) WaitForCommand(ctx context.Context, req *pb.WaitForCommandRequest) (*pb.WaitForCommandResponse, error) {
	b.sync.Lock()
	var done chan struct{}
	if cmd, ok := b.sync.cmds[req.GetId()]; ok {
		done = cmd.done
	}
	b.sync.Unlock()
	if done != nil {
		select {
		// Block till this command is done running (i.e., next Precmd call).
		case <-done:
		case <-ctx.Done():
		}
	}
	return &pb.WaitForCommandResponse{}, nil
}

type server struct {
	addr string
	gs   *grpc.Server
}

func (s *server) Addr() string {
	return s.addr
}

func (s *server) Start() (err error) {
	defer ergo.Annotate(&err, "failed to start the server")
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	return s.gs.Serve(lis)
}

func (s *server) Stop() {
	s.gs.GracefulStop()
}

func New(c Config, notifier Notifier) *server {
	b := &bifrost{config: c}
	b.sync.notifier = notifier
	b.sync.cmds = map[string]command{}
	gs := grpc.NewServer()
	pb.RegisterBifrostServer(gs, b)
	return &server{fmt.Sprintf("localhost:%d", c.BifrostPort()), gs}
}
