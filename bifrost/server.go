package bifrost

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/avamsi/checks"
	"github.com/avamsi/heimdall/config"
	"github.com/avamsi/heimdall/notifier"
)

func anyOf[T any](slice []T, predicate func(T) bool) bool {
	for _, i := range slice {
		if predicate(i) {
			return true
		}
	}
	return false
}

type server struct {
	*http.Server
	chat *notifier.Chat
}

func (s *server) notifyHandlerAsync(r *NotifyRequest) {
	// Don't notify if the command was interrupted by the user.
	if r.Code == 130 {
		return
	}
	isPrefixOfCmd := func(prefix string) bool {
		return strings.HasPrefix(r.Cmd, prefix)
	}
	// Don't notify if the command ran for less than 42 seconds / user requested to never be
	// notified for it (unless the user requested to always be notified for it).
	alwaysNotify := anyOf(config.AlwaysNotifyCommands(), isPrefixOfCmd)
	neverNotify := anyOf(config.NeverNotifyCommands(), isPrefixOfCmd)
	if !alwaysNotify && (time.Now().Unix() < int64(r.StartTime)+42 || neverNotify) {
		return
	}
	msg := fmt.Sprintf("`$ %s` completed running", r.Cmd)
	if err := s.chat.Notify(msg); err != nil {
		log.Println(err)
		checks.Check0(exec.Command("tput", "bel").Run())
	}
}

func (s *server) notifyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}
	nr := &NotifyRequest{}
	if err := json.NewDecoder(r.Body).Decode(nr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	go s.notifyHandlerAsync(nr)
}

func NewServer(port int, chat *notifier.Chat) *server {
	s := &server{&http.Server{Addr: fmt.Sprintf("localhost:%d", port)}, chat}
	http.HandleFunc("/notify", s.notifyHandler)
	return s
}
