package common

import (
	openai "github.com/sashabaranov/go-openai"
)

type Sender string

const (
	You       Sender = "You"
	Assistant Sender = "Assistant"
)

type ChatMessage struct {
	From    Sender
	Content string
}

func (m ChatMessage) Role() string {
	role := openai.ChatMessageRoleUser

	if m.From != You {
		role = openai.ChatMessageRoleAssistant
	}

	return role
}

func (m ChatMessage) ApiMessage() openai.ChatCompletionMessage {
	return openai.ChatCompletionMessage{
		Role:    m.Role(),
		Content: m.Content,
	}
}
