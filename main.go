package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/avamsi/checks"
	"github.com/avamsi/eclipse"
	"github.com/kardianos/service"

	"github.com/avamsi/heimdall/bifrost"
	"github.com/avamsi/heimdall/config"
	"github.com/avamsi/heimdall/notifier"
)

type Heimdall struct {
	_ *Bifrost
}

func (Heimdall) Config() string {
	return filepath.Join(checks.Check1(os.UserHomeDir()), ".config")
}

type Bifrost struct {
	Heimdall Heimdall
	Config   string
}

func (b Bifrost) bifrostService() service.Service {
	cfgPath := b.Config
	if cfgPath == "" {
		cfgPath = b.Heimdall.Config()
	}
	checks.Check0(config.Load(cfgPath))
	chat := checks.Check1(notifier.NewChat(checks.Check3(config.ChatOptions())))
	server := bifrost.NewServer(config.BifrostPort(), chat).Server
	return checks.Check1(bifrost.NewService(server, cfgPath))
}

func (b Bifrost) Run() {
	checks.Check0(b.bifrostService().Run())
}

func (b Bifrost) Install() {
	checks.Check0(b.bifrostService().Install())
}

func (b Bifrost) Uninstall() {
	checks.Check0(b.bifrostService().Uninstall())
}

func (b Bifrost) Start() {
	checks.Check0(b.bifrostService().Start())
}

func (b Bifrost) Stop() {
	checks.Check0(b.bifrostService().Stop())
}

//go:embed heimdall.sh
var sh string

func (Heimdall) Sh() {
	fmt.Print(sh)
}

func (h Heimdall) Notify(flags struct {
	Cmd         string
	StartTime   int
	Code        int
	ForceNotify bool
}) {
	flags.ForceNotify = flags.ForceNotify ||
		checks.Check1(config.EnvAsBool("HEIMDALL_FORCE_NOTIFY"))
	checks.Check0(config.Load(h.Config()))
	checks.Check1(http.Post(
		fmt.Sprintf("http://localhost:%d/notify", config.BifrostPort()),
		"application/json",
		bytes.NewBuffer(checks.Check1(json.Marshal(bifrost.NotifyRequest(flags)))),
	))
}

func main() {
	eclipse.Execute(Heimdall{})
}
