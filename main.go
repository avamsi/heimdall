package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/avamsi/checks"
	"github.com/avamsi/eclipse"
	"github.com/spf13/viper"
	"golang.org/x/term"

	"github.com/avamsi/heimdall/notify"
)

func parseWebhookURL(whURL string) (apiKey, token, spaceID string) {
	whURLParsed := checks.Check1(url.Parse(whURL))
	parts := strings.Split(whURLParsed.Path, "/")
	if !(parts[1] == "v1" && parts[2] == "spaces" && parts[4] == "messages") {
		panic(fmt.Sprintf("want: /v1/spaces/{spaceID}/messages; got: %v", whURLParsed.Path))
	}
	return whURLParsed.Query().Get("key"), whURLParsed.Query().Get("token"), parts[3]
}

type Heimdall struct{}

func init() {
	viper.SetConfigName("heimdall")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.config")
	viper.BindEnv("force_heimdall")
}

func (Heimdall) Execute(flags struct{ Reset bool }) {
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Print("Please enter the Chat webhook URL " +
				"(this will be saved to ~/.config/heimdall): ")
			viper.Set("chat.webhook_url", string(checks.Check1(term.ReadPassword(syscall.Stdin))))
			viper.SafeWriteConfig()
		} else {
			panic(err)
		}
	}
	parseWebhookURL(viper.GetString("chat.webhook_url"))
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
	apiKey, token, spaceID := parseWebhookURL(viper.GetString("chat.webhook_url"))
	msg := fmt.Sprintf("`$ %v` completed running", flags.Cmd)
	if err := notify.OnChat(context.Background(), apiKey, token, spaceID, msg); err != nil {
		log.Println(err)
		checks.Check0(exec.Command("tput", "bel").Run())
	}
}

func main() {
	eclipse.Execute(Heimdall{})
}
