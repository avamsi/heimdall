package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/avamsi/checks"
	"github.com/avamsi/eclipse"
	"golang.org/x/term"

	"github.com/avamsi/heimdall/notify"
)

var cfgPath string = filepath.Join(checks.Check1(user.Current()).HomeDir, ".config/heimdall")

func parseCfg() (whURL string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to parse config: %w", err)
		}
	}()
	whURLBytes, err := os.ReadFile(cfgPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(whURLBytes)), nil
}

func parseWebhookURL(whURL string) (apiKey, token, spaceID string) {
	whURLParsed := checks.Check1(url.Parse(whURL))
	parts := strings.Split(whURLParsed.Path, "/")
	if !(parts[1] == "v1" && parts[2] == "spaces" && parts[4] == "messages") {
		panic(fmt.Sprintf("want: /v1/spaces/{spaceID}/messages; got: %v", whURLParsed.Path))
	}
	return whURLParsed.Query().Get("key"), whURLParsed.Query().Get("token"), parts[3]
}

type Heimdall struct{}

func (Heimdall) Execute(flags struct{ Reset bool }) {
	whURL, err := parseCfg()
	if errors.Is(err, os.ErrNotExist) || flags.Reset {
		fmt.Print("Please enter the webhook URL (this will be saved to ~/.config/heimdall): ")
		whURL = strings.TrimSpace(string(checks.Check1(term.ReadPassword(syscall.Stdin))))
		fmt.Println()
		checks.Check1(
			os.OpenFile(cfgPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.ModePerm),
		).WriteString(whURL)
	} else {
		checks.Check0(err)
	}
	parseWebhookURL(whURL)
	fmt.Println("All good to go! " +
		"Please add `source <(heimdall sh)` to your shell config if you haven't already.")
}

//go:embed heimdall.sh
var sh string

func (Heimdall) Sh() {
	fmt.Print(sh)
}

func (Heimdall) Notify(flags struct {
	Cmd       string
	StartTime int
	Code      int
}) {
	// Only notify if a command isn't interrupted by the user and runs for longer than 42 seconds.
	if flags.Code != 130 && time.Now().Unix() < int64(flags.StartTime)+42 {
		return
	}
	apiKey, token, spaceID := parseWebhookURL(checks.Check1(parseCfg()))
	msg := fmt.Sprintf("`$ %v` completed running", flags.Cmd)
	if err := notify.OnChat(context.Background(), apiKey, token, spaceID, msg); err != nil {
		log.Println(err)
		checks.Check0(exec.Command("tput", "bel").Run())
	}
}

func main() {
	eclipse.Execute(Heimdall{})
}
