package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
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
	return ergo.Must1(client.Preexec(context.Background(), &bpb.PreexecRequest{
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
	req := &bpb.PrecmdRequest{
		Command: &bpb.Command{
			Command:     opts.Cmd,
			PreexecTime: &timestamppb.Timestamp{Seconds: opts.PreexecTime},
			Id:          strings.TrimSpace(opts.ID),
		},
		ReturnCode:  opts.Code,
		ForceNotify: ergo.Must1(c.EnvAsBool("HEIMDALL_FORCE_NOTIFY")),
	}
	if stat, ok := ergo.Must1(os.Stdin.Stat()).Sys().(*syscall.Stat_t); ok {
		req.LastInteractionTime = timestamppb.New(time.Unix(stat.Atimespec.Unix()))
	}
	return ergo.Error1(client.Precmd(context.Background(), req))
}

func (h Heimdall) list(ctx context.Context) []*bpb.Command {
	// TODO: filter out the current command from this list.
	client := ergo.Must1(bifrost.NewClient(h.config()))
	resp := ergo.Must1(client.ListCommands(ctx, &bpb.ListCommandsRequest{}))
	cmds := resp.GetCommands()
	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i].GetPreexecTime().AsTime().Before(cmds[j].GetPreexecTime().AsTime())
	})
	return cmds
}

func (h Heimdall) List() error {
	for _, cmd := range h.list(context.Background()) {
		t := cmd.GetPreexecTime().AsTime().Local()
		fmt.Printf("[%s: %s] $ %s\n", t.Format(time.Kitchen), cmd.GetId(), cmd.GetCommand())
	}
	return nil
}

func (h Heimdall) chooseFromList() (id string, err error) {
	choices := []*selection.Choice{}
	for _, cmd := range h.list(context.Background()) {
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
			// TODO: should we log something here?
			return
		}
	}
	client := ergo.Must1(bifrost.NewClient(h.config()))
	ergo.Must1(client.WaitForCommand(context.Background(), &bpb.WaitForCommandRequest{
		Id: strings.TrimSpace(opts.ID),
	}))
}

type CacheOpts struct {
	// (rough) ttl (in seconds) of the cached results
	TTL int32 `default:"420"`
}

// Cache runs and caches the command's stdout, stderr and return code for ttl
// seconds (before rerunning it in the background).
//
// It doesn't work with compound commands or shell aliases. Consider wrapping
// the command with your favorite shell in that case. For example,
//
//	$ heimdall cache zsh -ic 'echo hello && echo world'
//
// Usage: cache command [args]
func (h Heimdall) Cache(opts CacheOpts, args []string) {
	// TODO: this doesn't work with shell aliases or compound commands.
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "please pass a command to be cached / run --help")
		os.Exit(1)
	}
	client := ergo.Must1(bifrost.NewClient(h.config()))
	resp := ergo.Must1(client.CacheCommand(context.Background(), &bpb.CacheCommandRequest{
		Command: args[0],
		Args:    args[1:],
		Ttl:     opts.TTL,
	}))
	if resp.GetStdout() != "" {
		fmt.Println(resp.GetStdout())
	}
	if resp.GetStderr() != "" {
		fmt.Fprintln(os.Stderr, resp.GetStderr())
	}
	os.Exit(int(resp.ReturnCode))
}

//go:generate eclipse docs --out=eclipse.docs
//go:embed eclipse.docs
var docs []byte

func main() {
	eclipse.Execute(docs, Heimdall{}, Bifrost{})
}
