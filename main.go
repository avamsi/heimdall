package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "embed"

	"github.com/avamsi/eclipse"
	"github.com/avamsi/ergo"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/avamsi/heimdall/bifrost"
	"github.com/avamsi/heimdall/config"

	bpb "github.com/avamsi/heimdall/bifrost/proto"
)

type Heimdall struct{}

func (Heimdall) Config() string {
	return filepath.Join(ergo.Check1(os.UserHomeDir()), ".config")
}

type Bifrost struct {
	H      Heimdall
	Config string
}

func (b Bifrost) newService() bifrost.Service {
	cfgDir := b.Config
	if cfgDir == "" {
		cfgDir = b.H.Config()
	}
	return bifrost.NewService(ergo.Check1(config.Load(cfgDir)))
}

func (b Bifrost) Run() error {
	return b.newService().Run()
}

func (b Bifrost) Install() error {
	return b.newService().Install()
}

func (b Bifrost) Uninstall() error {
	return b.newService().Uninstall()
}

func (b Bifrost) Start() error {
	return b.newService().Start()
}

func (b Bifrost) Stop() error {
	return b.newService().Stop()
}

//go:embed heimdall.sh
var sh string

func (Heimdall) Sh() {
	fmt.Print(sh)
}

type PreexecOpts struct {
	Cmd string
	// Seconds from Epoch.
	Time int64
}

func (h Heimdall) Preexec(opts PreexecOpts) (string, error) {
	c := ergo.Check1(config.Load(h.Config()))
	resp, err := bifrost.NewClient(c).Preexec(context.TODO(), &bpb.PreexecRequest{
		Command: &bpb.Command{
			Command:     opts.Cmd,
			PreexecTime: timestamppb.New(time.UnixMilli(1000 * opts.Time)),
		},
	})
	if err != nil {
		return "", err
	}
	return resp.GetId(), err
}

type PrecmdOpts struct {
	Cmd string
	// Seconds from Epoch.
	PreexecTime int64
	// Return code of the command.
	Code int32
}

func (h Heimdall) Precmd(opts PrecmdOpts) error {
	c := ergo.Check1(config.Load(h.Config()))
	_, err := bifrost.NewClient(c).Precmd(context.TODO(), &bpb.PrecmdRequest{
		Command: &bpb.Command{
			Command:     opts.Cmd,
			PreexecTime: timestamppb.New(time.UnixMilli(1000 * opts.PreexecTime)),
		},
		ReturnCode:  opts.Code,
		ForceNotify: ergo.Check1(c.EnvAsBool("HEIMDALL_FORCE_NOTIFY")),
	})
	return err
}

func (h Heimdall) List() error {
	c := ergo.Check1(config.Load(h.Config()))
	resp, err := bifrost.NewClient(c).ListCommands(context.TODO(), &bpb.ListCommandsRequest{})
	if err != nil {
		return err
	}
	cmds := resp.GetCommands()
	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i].GetPreexecTime().AsTime().Before(cmds[j].GetPreexecTime().AsTime())
	})
	for _, cmd := range cmds {
		fmt.Println(cmd)
	}
	return nil
}

func main() {
	eclipse.Execute(Heimdall{}, Bifrost{})
}
