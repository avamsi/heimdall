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

	"github.com/avamsi/eclipse"
	"golang.org/x/term"

	"github.com/avamsi/heimdal/notify"
)

func check0(err error) {
	if err != nil {
		panic(err)
	}
}

func check1[T any](arg T, err error) T {
	check0(err)
	return arg
}

var cfgPath string = filepath.Join(check1(user.Current()).HomeDir, ".config/heimdal")

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
	whURLParsed := check1(url.Parse(whURL))
	parts := strings.Split(whURLParsed.Path, "/")
	if !(parts[1] == "v1" && parts[2] == "spaces" && parts[4] == "messages") {
		panic(fmt.Sprintf("want: /v1/spaces/{spaceID}/messages; got: %v", whURLParsed.Path))
	}
	return whURLParsed.Query().Get("key"), whURLParsed.Query().Get("token"), parts[3]
}

type Heimdal struct{}

func (Heimdal) Execute(flags struct{ Reset bool }) {
	whURL, err := parseCfg()
	if errors.Is(err, os.ErrNotExist) || flags.Reset {
		fmt.Print("Please enter the webhook URL (this will be saved to ~/.config/heimdal): ")
		whURL = strings.TrimSpace(string(check1(term.ReadPassword(syscall.Stdin))))
		fmt.Println()
		check1(
			os.OpenFile(cfgPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.ModePerm),
		).WriteString(whURL)
	} else {
		check0(err)
	}
	parseWebhookURL(whURL)
	fmt.Println("All good to go! " +
		"Please add `source <(heimdal sh)` to your shell config if you haven't already.")
}

//go:embed heimdal.sh
var sh string

func (Heimdal) Sh() {
	fmt.Print(sh)
}

func (Heimdal) Notify(flags struct {
	Cmd string
	T   int
}) {
	apiKey, token, spaceID := parseWebhookURL(check1(parseCfg()))
	// Only notify if a command runs longer than 42 seconds.
	if time.Now().Unix() < int64(flags.T)+42 {
		return
	}
	msg := fmt.Sprintf("`$ %v` completed running", flags.Cmd)
	if err := notify.OnChat(context.Background(), apiKey, token, spaceID, msg); err != nil {
		log.Println(err)
		check0(exec.Command("tput", "bel").Run())
	}
}

func main() {
	eclipse.Execute(Heimdal{})
}
