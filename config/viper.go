package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"syscall"
	"unicode"

	"github.com/avamsi/ergo"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

func create() error {
	fmt.Print("Please enter the Chat webhook URL: ")
	url, err := term.ReadPassword(syscall.Stdin)
	fmt.Println()
	if err != nil {
		return err
	}
	viper.Set("bifrost.port", 54351)
	viper.Set("chat.webhook_url", string(url))
	viper.Set("commands.always_notify", []string{"_github_io_avamsi_heimdall_replace_me"})
	viper.Set("commands.never_notify", []string{"_github_io_avamsi_heimdall_replace_me"})
	return viper.SafeWriteConfig()
}

func init() {
	viper.SetConfigName("heimdall")
	viper.SetConfigType("yaml")
}

func Load(cfgPath string) (err error) {
	defer ergo.Annotate(&err, "failed to load config")
	viper.AddConfigPath(cfgPath)
	if err := viper.ReadInConfig(); err != nil {
		cfg404 := viper.ConfigFileNotFoundError{}
		if errors.As(err, &cfg404) {
			fmt.Printf("%s; Creating anew..\n", cfg404.Error())
			if err := create(); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return ergo.Error3(ChatOptions())
}

func OnConfigChange(run func()) {
	viper.OnConfigChange(func(fsnotify.Event) {
		run()
	})
}

func EnvAsBool(s string) (bool, error) {
	for _, c := range s {
		if c != '_' && !unicode.IsUpper(c) {
			return false, fmt.Errorf("want: CONSTANT_CASE; got: %s", s)
		}
	}
	return viper.GetBool(s), nil
}

func BifrostPort() int {
	return viper.GetInt("bifrost.port")
}

func ChatOptions() (apiKey, token, spaceID string, err error) {
	defer ergo.Annotate(&err, "failed to parse chat webhook URL")
	raw := viper.GetString("chat.webhook_url")
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", "", "", err
	}
	parts := strings.Split(parsed.Path, "/")
	if parts[1] != "v1" || parts[2] != "spaces" || parts[4] != "messages" {
		return "", "", "", fmt.Errorf("want: /v1/spaces/{spaceID}/messages; got: %s", parsed.Path)
	}
	return parsed.Query().Get("key"), parsed.Query().Get("token"), parts[3], nil
}

func AlwaysNotifyCommands() []string {
	return viper.GetStringSlice("commands.always")
}

func NeverNotifyCommands() []string {
	return viper.GetStringSlice("commands.never")
}
