package bifrost

import (
	"fmt"

	"github.com/avamsi/ergo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/avamsi/heimdall/bifrost/internal/server"
	"github.com/avamsi/heimdall/bifrost/internal/service"
	"github.com/avamsi/heimdall/notifiers"

	pb "github.com/avamsi/heimdall/bifrost/proto"
)

type Config interface {
	BifrostPort() int
	ChatOptions() (apiKey string, token string, spaceID string, err error)
	Dir() string
	AlwaysNotifyCommands() []string
	NeverNotifyCommands() []string
}

func NewClient(c Config) pb.BifrostClient {
	addr := fmt.Sprintf("localhost:%d", c.BifrostPort())
	creds := grpc.WithTransportCredentials(insecure.NewCredentials())
	conn := ergo.Check1(grpc.Dial(addr, creds))
	return pb.NewBifrostClient(conn)
}

type Service interface {
	Run() error
	Install() error
	Uninstall() error
	Start() error
	Stop() error
}

func NewService(c Config) Service {
	chat := ergo.Check1(notifiers.NewChat(ergo.Check3(c.ChatOptions())))
	return ergo.Check1(service.New(server.New(c, chat), c.Dir()))
}
