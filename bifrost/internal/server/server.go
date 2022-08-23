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
	"golang.org/x/exp/maps"
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
	Notify(msg string) (err error)
}

type bifrost struct {
	pb.UnimplementedBifrostServer
	config Config
	sync   struct {
		sync.Mutex
		notifier Notifier
		cmds     map[string]*pb.Command
	}
}

func (b *bifrost) preexecAsync(todo context.Context, req *pb.PreexecRequest, id string) {
	b.sync.Lock()
	defer b.sync.Unlock()
	cmd := req.GetCommand()
	cmd.Id = id
	b.sync.cmds[id] = cmd
}

func (b *bifrost) Preexec(todo context.Context, req *pb.PreexecRequest) (*pb.PreexecResponse, error) {
	id := xid.New().String()
	go b.preexecAsync(todo, req, id)
	return &pb.PreexecResponse{Id: id}, nil
}

func (b *bifrost) precmdAsync(todo context.Context, req *pb.PrecmdRequest) {
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
	if !alwaysNotify && (neverNotify || time.Since(cmd.GetPreexecTime().AsTime()) < 42*time.Second) {
		return
	}
	t := cmd.GetPreexecTime().AsTime()
	// TODO: add duration of the command to msg here.
	msg := fmt.Sprintf("[%s] `$ %s` -> %d in %s", t.Format(time.Kitchen), cmd.GetCommand(), req.GetReturnCode(), time.Since(t))
	b.sync.Lock()
	defer b.sync.Unlock()
	if err := b.sync.notifier.Notify(msg); err != nil {
		log.Println(err)
		ergo.Must0(exec.Command("tput", "bel").Run())
	}
}

func (b *bifrost) Precmd(todo context.Context, req *pb.PrecmdRequest) (*pb.PrecmdResponse, error) {
	go b.precmdAsync(todo, req)
	return nil, nil
}

func (b *bifrost) ListCommands(todo context.Context, _ *pb.ListCommandsRequest) (*pb.ListCommandsResponse, error) {
	b.sync.Lock()
	defer b.sync.Unlock()
	return &pb.ListCommandsResponse{Commands: maps.Values(b.sync.cmds)}, nil
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
	b.sync.cmds = map[string]*pb.Command{}
	gs := grpc.NewServer()
	pb.RegisterBifrostServer(gs, b)
	return &server{fmt.Sprintf(":%d", c.BifrostPort()), gs}
}
