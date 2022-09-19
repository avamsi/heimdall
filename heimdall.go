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

	"github.com/avamsi/ergo"
	"github.com/djherbis/atime"
	"github.com/erikgeiser/promptkit/selection"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/avamsi/heimdall/bifrost"
	"github.com/avamsi/heimdall/config"
	"github.com/avamsi/heimdall/notifiers"

	bpb "github.com/avamsi/heimdall/bifrost/proto"
)

type Heimdall struct{}

//go:embed heimdall.sh
var sh string

// Sh prints a shell script to be sourced into your favorite shell.
//
//	source <(heimdall sh)
func (Heimdall) Sh() {
	fmt.Print(sh)
}

// Config prints the directory heimdall uses to read the config from.
func (Heimdall) Config() string {
	return filepath.Join(ergo.Must1(os.UserHomeDir()), ".config")
}

func (h Heimdall) config() *config.Config {
	return ergo.Must1(config.Load(h.Config()))
}

// Usage: notify [message]
func (h Heimdall) Notify(args []string) {
	chat := ergo.Must1(notifiers.NewChat(ergo.Must3(h.config().ChatOptions())))
	ergo.Must0(chat.Notify(context.Background(), strings.Join(args, " ")))
}

type StartOpts struct {
	Cmd  string
	Time int64  // seconds from epoch
	ID   string // (id of the command)
}

// Starts adds a command to the list of currently running commands.
func (h Heimdall) Start(opts StartOpts) string {
	client := ergo.Must1(bifrost.NewClient(h.config()))
	req := &bpb.CommandStartRequest{
		Command: &bpb.Command{
			Command: opts.Cmd,
			Id:      opts.ID,
		},
	}
	if opts.Time != 0 {
		req.Command.StartTime = &timestamppb.Timestamp{Seconds: opts.Time}
	} else {
		req.Command.StartTime = timestamppb.Now()
	}
	return ergo.Must1(client.CommandStart(context.Background(), req)).GetId()
}

type EndOpts struct {
	Cmd       string
	StartTime int64  // seconds from epoch
	Code      int32  // return code of the command
	ID        string // id of the command (from start)
}

// End removes a command from the list of currently running commands.
//
// This may also result in a notification to the user if the command was
// configured to be always notified on or if the start / last interacted with
// time is outside of the configured duration.
func (h Heimdall) End(opts EndOpts) error {
	c := h.config()
	client := ergo.Must1(bifrost.NewClient(c))
	req := &bpb.CommandEndRequest{
		Command: &bpb.Command{
			Command: opts.Cmd,
			Id:      strings.TrimSpace(opts.ID),
		},
		ReturnCode:          opts.Code,
		ForceNotify:         ergo.Must1(c.EnvAsBool("HEIMDALL_FORCE_NOTIFY")),
		LastInteractionTime: timestamppb.New(atime.Get(ergo.Must1(os.Stdin.Stat()))),
	}
	if opts.StartTime != 0 {
		req.Command.StartTime = &timestamppb.Timestamp{Seconds: opts.StartTime}
	}
	return ergo.Error1(client.CommandEnd(context.Background(), req))
}

func (h Heimdall) list(ctx context.Context) []*bpb.Command {
	// TODO: filter out the current command from this list.
	client := ergo.Must1(bifrost.NewClient(h.config()))
	resp := ergo.Must1(client.ListCommands(ctx, &bpb.ListCommandsRequest{}))
	cmds := resp.GetCommands()
	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i].GetStartTime().AsTime().Before(cmds[j].GetStartTime().AsTime())
	})
	return cmds
}

// List lists heimdall aware currently running commands.
func (h Heimdall) List() error {
	for _, cmd := range h.list(context.Background()) {
		t := cmd.GetStartTime().AsTime().Local()
		fmt.Printf("[%s: %s] $ %s\n", t.Format(time.Kitchen), cmd.GetId(), cmd.GetCommand())
	}
	return nil
}

func (h Heimdall) chooseFromList() (id string, err error) {
	choices := []*selection.Choice{}
	for _, cmd := range h.list(context.Background()) {
		t := cmd.GetStartTime().AsTime().Local()
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
	ID string // id of the command (either from start or list)
}

// Wait waits on a heimdall aware command till it's done running and exits 0.
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
	// acceptable duration (in seconds) since the cached run
	Within int32 `default:"420"`
	// returns failed runs (only successful runs are returned by default)
	Any bool `default:"false"`
}

// Cache pass-through executes the command (i.e., command is run iff a cached
// run isn't available / acceptable, depending on the flags) and returns its
// stdout, stderr and return code.
//
// Cache also triggers a cron to run and cache the command in the background if
// it isn't already the case (i.e., Cache is being called for the first time).
//
// Cache doesn't work with compound commands or shell aliases. Consider wrapping
// the command with your favorite shell in that case. For example,
//
//	$ heimdall cache zsh -ic 'echo hello && echo world'
//
// Short: Cache pass-through executes the command
// Usage: cache command [args]
func (h Heimdall) Cache(opts CacheOpts, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "please pass a command to be run, see --help")
		os.Exit(1)
	}
	client := ergo.Must1(bifrost.NewClient(h.config()))
	resp := ergo.Must1(client.CacheCommand(context.Background(), &bpb.CacheCommandRequest{
		Command: args[0],
		Args:    args[1:],
		Within:  opts.Within,
		Any:     opts.Any,
	}))
	fmt.Print(resp.GetStdout())
	fmt.Fprint(os.Stderr, resp.GetStderr())
	os.Exit(int(resp.ReturnCode))
}
