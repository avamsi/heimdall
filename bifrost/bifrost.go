package bifrost

import (
	"fmt"

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

func NewClient(c Config) (pb.BifrostClient, error) {
	addr := fmt.Sprintf("localhost:%d", c.BifrostPort())
	creds := grpc.WithTransportCredentials(insecure.NewCredentials())
	if conn, err := grpc.Dial(addr, creds); err != nil {
		return nil, err
	} else {
		return pb.NewBifrostClient(conn), nil
	}
}

type Service interface {
	Run() error
	Install() error
	Uninstall() error
	Start() error
	Stop() error
}

func NewService(c Config) (Service, error) {
	apiKey, token, spaceID, err := c.ChatOptions()
	if err != nil {
		return nil, err
	}
	if chat, err := notifiers.NewChat(apiKey, token, spaceID); err != nil {
		return nil, err
	} else {
		return service.New(server.New(c, chat), c.Dir())
	}
}
