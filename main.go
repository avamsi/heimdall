package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/avamsi/eclipse"
	"github.com/avamsi/ergo"
	"github.com/kardianos/service"

	"github.com/avamsi/heimdall/bifrost"
	"github.com/avamsi/heimdall/config"
	"github.com/avamsi/heimdall/notifiers"
)

type Heimdall struct {
	_ *Bifrost
}

func (Heimdall) Config() string {
	return filepath.Join(ergo.Check1(os.UserHomeDir()), ".config")
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
	ergo.Check0(config.Load(cfgPath))
	chat := ergo.Check1(notifiers.NewChat(ergo.Check3(config.ChatOptions())))
	server := bifrost.NewServer(config.BifrostPort(), chat).Server
	return ergo.Check1(bifrost.NewService(server, cfgPath))
}

func (b Bifrost) Run() {
	ergo.Check0(b.bifrostService().Run())
}

func (b Bifrost) Install() {
	ergo.Check0(b.bifrostService().Install())
}

func (b Bifrost) Uninstall() {
	ergo.Check0(b.bifrostService().Uninstall())
}

func (b Bifrost) Start() {
	ergo.Check0(b.bifrostService().Start())
}

func (b Bifrost) Stop() {
	ergo.Check0(b.bifrostService().Stop())
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
		ergo.Check1(config.EnvAsBool("HEIMDALL_FORCE_NOTIFY"))
	ergo.Check0(config.Load(h.Config()))
	ergo.Check1(http.Post(
		fmt.Sprintf("http://localhost:%d/notify", config.BifrostPort()),
		"application/json",
		bytes.NewBuffer(ergo.Check1(json.Marshal(bifrost.NotifyRequest(flags)))),
	))
}

func main() {
	eclipse.Execute(Heimdall{})
}
