package api

import (
	"context"
	"errors"
	"github.com/otiai10/openaigo"
	"strings"
)

func ChatApi(model string, openaiApiKey string, message string) (string, error) {
	client := openaigo.NewClient(openaiApiKey)
	request := openaigo.ChatCompletionRequestBody{
		Model: model,
		Messages: []openaigo.ChatMessage{
			{Role: "user", Content: message},
		},
	}
	ctx := context.Background()
	response, err := client.Chat(ctx, request)
	//fmt.Println(response, err)
	if err != nil {
		return "", err
	}
	if len(response.Choices) == 0 {
		return "", errors.New("empty response")
	}
	return strings.TrimSpace(response.Choices[0].Message.Content), nil
}
