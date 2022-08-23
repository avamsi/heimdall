package service

import (
	"github.com/avamsi/ergo"
	"github.com/kardianos/service"
)

type Server interface {
	Addr() string
	Start() error
	Stop()
}

type server struct {
	s Server
}

func (srvr server) Start(srvc service.Service) error {
	logger, err := srvc.Logger(nil)
	if err != nil {
		return err
	}
	logger.Infof("Listening on %s..", srvr.s.Addr())
	go func() {
		ergo.Must0(srvr.s.Start())
	}()
	return nil
}

func (srvr server) Stop(srvc service.Service) error {
	logger, err := srvc.Logger(nil)
	if err != nil {
		return err
	}
	logger.Warning("Stopping..")
	go srvr.s.Stop()
	return nil
}

func New(s Server, cfgDir string) (_ service.Service, err error) {
	defer ergo.Annotate(&err, "failed to create a new service")
	cfg := &service.Config{
		Name:      "com.github.io.avamsi.heimdall.bifrost",
		Arguments: []string{"bifrost", "run", "--config=" + cfgDir},
		Option:    service.KeyValue{"RunAtLoad": true},
	}
	return service.New(server{s}, cfg)
}
