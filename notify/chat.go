package notify

import (
	"context"
	"fmt"

	"google.golang.org/api/chat/v1"
	"google.golang.org/api/option"
)

type simpleCallOption struct {
	key, value string
}

func (s simpleCallOption) Get() (string, string) {
	return s.key, s.value
}

func OnChat(ctx context.Context, apiKey, token, spaceID, msg string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to notify on chat: %w", err)
		}
	}()
	service, err := chat.NewService(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return err
	}
	call := service.Spaces.Messages.Create("spaces/"+spaceID, &chat.Message{Text: msg})
	if _, err := call.Context(ctx).Do(simpleCallOption{"token", token}); err != nil {
		return err
	}
	return nil
}
