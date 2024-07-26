package gpt

import (
	openai "github.com/sashabaranov/go-openai"
)

type OpenAIClient struct {
	client *openai.Client
}

func NewOpenAIClient(apiKey string) *OpenAIClient {
	client := openai.NewClient(apiKey)
	return &OpenAIClient{client: client}
}
