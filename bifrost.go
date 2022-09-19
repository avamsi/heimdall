package main

import (
	"github.com/avamsi/ergo"
	"github.com/avamsi/heimdall/bifrost"
	"github.com/avamsi/heimdall/config"
)

// Bifrost suite of sub-commands deal with bifrost service ops.
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
