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

type Config struct {
	dir string
	v   *viper.Viper
}

func (c *Config) validate() error {
	return ergo.Error3(c.ChatOptions())
}

func (c *Config) createFile() error {
	fmt.Print("Please enter the Chat webhook URL: ")
	url, err := term.ReadPassword(syscall.Stdin)
	fmt.Println()
	if err != nil {
		return err
	}
	c.v.Set("bifrost.port", 54351)
	c.v.Set("chat.webhook_url", string(url))
	c.v.Set("commands.always_notify", []string{"_github_io_avamsi_heimdall_replace_me"})
	c.v.Set("commands.never_notify", []string{"_github_io_avamsi_heimdall_replace_me"})
	return c.v.SafeWriteConfig()
}

func (c *Config) loadOrCreateFile() (err error) {
	defer func() {
		if err == nil {
			if err = c.validate(); err != nil {
				// TODO: WatchConfig?
			}
		}
	}()
	if err = c.v.ReadInConfig(); err == nil {
		return nil
	}
	cfg404 := viper.ConfigFileNotFoundError{}
	if !errors.As(err, &cfg404) {
		return err
	}
	fmt.Printf("%s; Creating anew..\n", cfg404.Error())
	return c.createFile()
}

func Load(dir string) (c *Config, err error) {
	defer ergo.Annotate(&err, "failed to load config")
	v := viper.New()
	v.SetConfigName("heimdall")
	v.SetConfigType("yaml")
	v.AddConfigPath(dir)
	c = &Config{dir, v}
	return c, c.loadOrCreateFile()
}

func (c *Config) Dir() string {
	return c.dir
}

func (c *Config) OnChange(run func()) {
	c.v.OnConfigChange(func(fsnotify.Event) {
		run()
	})
}

func (c *Config) EnvAsBool(s string) (bool, error) {
	for _, r := range s {
		if r != '_' && !unicode.IsUpper(r) {
			return false, fmt.Errorf("want: CONSTANT_CASE; got: %s", s)
		}
	}
	return c.v.GetBool(s), nil
}

func (c *Config) BifrostPort() int {
	return c.v.GetInt("bifrost.port")
}

func (c *Config) ChatOptions() (apiKey, token, spaceID string, err error) {
	defer ergo.Annotate(&err, "failed to parse chat webhook URL")
	raw := c.v.GetString("chat.webhook_url")
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

func (c *Config) AlwaysNotifyCommands() []string {
	return c.v.GetStringSlice("commands.always")
}

func (c *Config) NeverNotifyCommands() []string {
	return c.v.GetStringSlice("commands.never")
}
