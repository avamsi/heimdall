package notifier

import (
	"context"
	"fmt"

	"github.com/avamsi/ergo"
	"google.golang.org/api/chat/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

type Chat struct {
	service *chat.Service
	token   googleapi.CallOption
	spaceID string
}

func (c *Chat) Notify(msg string) (err error) {
	defer ergo.Annotate(&err, "failed to notify on chat")
	call := c.service.Spaces.Messages.Create("spaces/"+c.spaceID, &chat.Message{Text: msg})
	return ergo.Error1(call.Context(context.TODO()).Do(c.token))
}

type simpleCallOption struct {
	key, value string
}

func (s *simpleCallOption) Get() (string, string) {
	return s.key, s.value
}

func NewChat(apiKey, token, spaceID string) (c *Chat, err error) {
	s, err := chat.NewService(context.TODO(), option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create new chat notifier: %w", err)
	}
	return &Chat{service: s, token: &simpleCallOption{"token", token}, spaceID: spaceID}, nil
}
