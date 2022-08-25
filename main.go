package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "embed"

	"github.com/avamsi/eclipse"
	"github.com/avamsi/ergo"
	"github.com/erikgeiser/promptkit/selection"
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
	return ergo.Must1(bifrost.NewService(ergo.Must1(config.Load(cfgDir))))
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

func (h Heimdall) config() *config.Config {
	return ergo.Must1(config.Load(h.Config()))
}

type PreexecOpts struct {
	Cmd  string
	Time int64 // seconds from epoch
}

func (h Heimdall) Preexec(opts PreexecOpts) string {
	client := ergo.Must1(bifrost.NewClient(h.config()))
	return ergo.Must1(client.Preexec(context.TODO(), &bpb.PreexecRequest{
		Command: &bpb.Command{
			Command:     opts.Cmd,
			PreexecTime: &timestamppb.Timestamp{Seconds: opts.Time},
		},
	})).GetId()
}

type PrecmdOpts struct {
	Cmd         string
	PreexecTime int64  // seconds from epoch
	Code        int32  // return code of the command
	ID          string // id of the command (as originally returned by preexec)
}

func (h Heimdall) Precmd(opts PrecmdOpts) error {
	c := h.config()
	client := ergo.Must1(bifrost.NewClient(c))
	return ergo.Error1(client.Precmd(context.TODO(), &bpb.PrecmdRequest{
		Command: &bpb.Command{
			Command:     opts.Cmd,
			PreexecTime: &timestamppb.Timestamp{Seconds: opts.PreexecTime},
			Id:          strings.TrimSpace(opts.ID),
		},
		ReturnCode:  opts.Code,
		ForceNotify: ergo.Must1(c.EnvAsBool("HEIMDALL_FORCE_NOTIFY")),
	}))
}

func (h Heimdall) list() []*bpb.Command {
	// TODO: filter out the current command from this list.
	client := ergo.Must1(bifrost.NewClient(h.config()))
	resp := ergo.Must1(client.ListCommands(context.TODO(), &bpb.ListCommandsRequest{}))
	cmds := resp.GetCommands()
	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i].GetPreexecTime().AsTime().Before(cmds[j].GetPreexecTime().AsTime())
	})
	return cmds
}

func (h Heimdall) List() error {
	for _, cmd := range h.list() {
		t := cmd.GetPreexecTime().AsTime().Local()
		fmt.Printf("[%s: %s] $ %s\n", t.Format(time.Kitchen), cmd.GetId(), cmd.GetCommand())
	}
	return nil
}

func (h Heimdall) chooseFromList() (id string, err error) {
	choices := []*selection.Choice{}
	for _, cmd := range h.list() {
		t := cmd.GetPreexecTime().AsTime().Local()
		s := fmt.Sprintf("[%s] $ %s", t.Format(time.Kitchen), cmd.GetCommand())
		choices = append(choices, &selection.Choice{String: s, Value: cmd})
	}
	s := selection.New("", selection.Choices(choices))
	s.FilterPlaceholder = ""
	var c *selection.Choice
	if c, err = s.RunPrompt(); err == nil {
		return c.Value.(*bpb.Command).GetId(), nil
	}
	return "", err
}

type WaitOpts struct {
	ID string // id of the command (either from preexec or list)
}

func (h Heimdall) Wait(opts WaitOpts) {
	if opts.ID == "" {
		var err error
		if opts.ID, err = h.chooseFromList(); err != nil {
			return
		}
	}
	client := ergo.Must1(bifrost.NewClient(h.config()))
	ergo.Must1(client.WaitForCommand(context.TODO(), &bpb.WaitForCommandRequest{
		Id: strings.TrimSpace(opts.ID),
	}))
}

//go:generate eclipse docs --out=eclipse.docs
//go:embed eclipse.docs
var docs []byte

func main() {
	eclipse.Execute(docs, Heimdall{}, Bifrost{})
}
