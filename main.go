package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"

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

func parseWebhookURL(whURL []byte) (apiKey, token, spaceID string) {
	whURLParsed := check1(url.Parse(strings.TrimSpace(string(whURL))))
	parts := strings.Split(whURLParsed.Path, "/")
	if !(parts[1] == "v1" && parts[2] == "spaces" && parts[4] == "messages") {
		panic(fmt.Sprintf("want: /v1/spaces/{spaceID}/messages; got: %v", whURLParsed.Path))
	}
	return whURLParsed.Query().Get("key"), whURLParsed.Query().Get("token"), parts[3]
}

var (
	cfgPath string = filepath.Join(check1(user.Current()).HomeDir, ".config/heimdal")
	//go:embed heimdal.sh
	sh string
)

type Heimdal struct{}

func (Heimdal) Sh() {
	fmt.Print(sh)
}

func (Heimdal) Watch(flags struct {
	Cmd string
	T   int
}) {
	whURL, err := os.ReadFile(cfgPath)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Print("Please enter the webhook URL (this will be saved to ~/.config/heimdal): ")
		whURL = check1(term.ReadPassword(syscall.Stdin))
		fmt.Println()
		check1(os.OpenFile(cfgPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.ModePerm)).Write(whURL)
	} else {
		check0(err)
	}
	apiKey, token, spaceID := parseWebhookURL(whURL)
	msg := fmt.Sprintf("%v @ %v", flags.Cmd, flags.T) // TODO(avamsi): make this actually useful.
	check0(notify.OnChat(context.Background(), apiKey, token, spaceID, msg))
}

func main() {
	eclipse.Execute(Heimdal{})
}
