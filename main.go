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

func anyOf[T any](slice []T, predicate func(T) bool) bool {
	for _, i := range slice {
		if predicate(i) {
			return true
		}
	}
	return false
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

func init() {
	viper.SetConfigName("heimdall")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.config")
	viper.BindEnv("force_heimdall")
}

func (Heimdall) Execute(flags struct{ Reset bool }) {
	cfg404 := false
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			cfg404 = true
		} else {
			panic(err)
		}
	}
	if cfg404 || flags.Reset {
		fmt.Print("Please enter the Chat webhook URL " +
			"(this will be saved to ~/.config/heimdall.yaml): ")
		viper.Set("chat.webhook_url", string(checks.Check1(term.ReadPassword(syscall.Stdin))))
		viper.Set("commands.always", []string{"_github_io_avamsi_heimdall_replace_me"})
		viper.Set("commands.never", []string{"_github_io_avamsi_heimdall_replace_me"})
		fmt.Println()
		if cfg404 {
			checks.Check0(viper.SafeWriteConfig())
		} else {
			checks.Check0(viper.WriteConfig())
		}
	}
	// Make sure we're able to extract the info we need from the Chat webhook URL.
	parseWebhookURL(viper.GetString("chat.webhook_url"))
	fmt.Println("All good to go! " +
		"Please add `source <(heimdall sh)` to your shell config if you haven't already.")
}

//go:embed heimdall.sh
var sh string

func (Heimdall) Sh() {
	fmt.Print(sh)
}

func bifrost(cmd string) {
	apiKey, token, spaceID := parseWebhookURL(viper.GetString("chat.webhook_url"))
	msg := fmt.Sprintf("`$ %v` completed running", cmd)
	if err := notify.OnChat(context.Background(), apiKey, token, spaceID, msg); err != nil {
		log.Println(err)
		checks.Check0(exec.Command("tput", "bel").Run())
	}
}

func (Heimdall) Notify(flags struct {
	Cmd       string
	StartTime int
	Code      int
}) {
	checks.Check0(viper.ReadInConfig())
	isPrefixOfCmd := func(prefix string) bool {
		return strings.HasPrefix(flags.Cmd, prefix)
	}
	// Don't worry about start time / return code since the user wants be notified explicitly.
	if viper.GetBool("force_heimdall") ||
		anyOf(viper.GetStringSlice("commands.always"), isPrefixOfCmd) {
		bifrost(flags.Cmd)
		return
	}
	// Don't notify if the command was interrupted by the user / ran for less than 42 seconds /
	// explicitly blocklisted by the user.
	if flags.Code == 130 || time.Now().Unix() < int64(flags.StartTime)+42 ||
		anyOf(viper.GetStringSlice("commands.never"), isPrefixOfCmd) {
		return
	}
	bifrost(flags.Cmd)
}

func main() {
	eclipse.Execute(Heimdall{})
}
