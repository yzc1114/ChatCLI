package api

import (
	"context"
	"errors"
	"github.com/otiai10/openaigo"
	"strings"
)

type Role string

const (
	User   Role = "user"
	System Role = "system"
)

type Msg struct {
	Role    Role
	Content string
}

func (m *Msg) Pack() openaigo.ChatMessage {
	return openaigo.ChatMessage{
		Role:    string(m.Role),
		Content: m.Content,
	}
}

func ChatApi(model string, openaiApiKey string, msgs []Msg) (string, error) {
	client := openaigo.NewClient(openaiApiKey)
	openaiMsgs := make([]openaigo.ChatMessage, 0, len(msgs))
	for _, msg := range msgs {
		openaiMsgs = append(openaiMsgs, msg.Pack())
	}
	request := openaigo.ChatCompletionRequestBody{
		Model:    model,
		Messages: openaiMsgs,
	}
	ctx := context.Background()
	response, err := client.Chat(ctx, request)
	if err != nil {
		return "", err
	}
	if len(response.Choices) == 0 {
		return "", errors.New("empty response")
	}
	return strings.TrimSpace(response.Choices[0].Message.Content), nil
}
