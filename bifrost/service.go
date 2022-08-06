package bifrost

import (
	"context"
	"net/http"
	"time"

	"github.com/avamsi/ergo"
	"github.com/kardianos/service"
)

type serverShim struct {
	*http.Server
}

func (srvr serverShim) Start(srvc service.Service) error {
	ergo.Check1(srvc.Logger(nil)).Infof("Listening on %s..", srvr.Addr)
	go ergo.Check0(srvr.ListenAndServe())
	return nil
}

func (srvr serverShim) Stop(srvc service.Service) error {
	ergo.Check1(srvc.Logger(nil)).Warning("Stopping..")
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()
	return srvr.Shutdown(ctx)
}

func NewService(s *http.Server, cfgPath string) (_ service.Service, err error) {
	defer ergo.Annotate(&err, "failed to create a new service")
	cfg := &service.Config{
		Name:      "com.github.io.avamsi.heimdall.bifrost",
		Arguments: []string{"bifrost", "run", "--config=" + cfgPath},
		Option:    service.KeyValue{"RunAtLoad": true},
	}
	return service.New(serverShim{s}, cfg)
}
