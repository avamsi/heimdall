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
	return filepath.Join(ergo.Must1(os.UserHomeDir()), ".config")
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
	return bifrost.NewService(ergo.Must1(config.Load(cfgDir)))
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
	Cmd  string
	Time int64 // seconds from epoch
}

func (h Heimdall) Preexec(opts PreexecOpts) (string, error) {
	c := ergo.Must1(config.Load(h.Config()))
	resp, err := bifrost.NewClient(c).Preexec(context.TODO(), &bpb.PreexecRequest{
		Command: &bpb.Command{
			Command:     opts.Cmd,
			PreexecTime: timestamppb.New(time.UnixMilli(1000 * opts.Time)),
		},
	})
	if err != nil {
		return "", err
	}
	return resp.GetId(), nil
}

type PrecmdOpts struct {
	Cmd         string
	PreexecTime int64 // seconds from epoch
	Code        int32 // return code of the command
}

func (h Heimdall) Precmd(opts PrecmdOpts) error {
	c := ergo.Must1(config.Load(h.Config()))
	return ergo.Error1(bifrost.NewClient(c).Precmd(context.TODO(), &bpb.PrecmdRequest{
		Command: &bpb.Command{
			Command:     opts.Cmd,
			PreexecTime: timestamppb.New(time.UnixMilli(1000 * opts.PreexecTime)),
		},
		ReturnCode:  opts.Code,
		ForceNotify: ergo.Must1(c.EnvAsBool("HEIMDALL_FORCE_NOTIFY")),
	}))
}

func (h Heimdall) List() error {
	c := ergo.Must1(config.Load(h.Config()))
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

//go:generate eclipse docs --out=eclipse.docs
//go:embed eclipse.docs
var docs []byte

func main() {
	eclipse.Execute(docs, Heimdall{}, Bifrost{})
}
