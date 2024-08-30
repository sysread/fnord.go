package gpt

import (
	"context"
	"fmt"
	"net/http"

	openai "github.com/sashabaranov/go-openai"

	"github.com/sysread/fnord/pkg/config"
	"github.com/sysread/fnord/pkg/debug"
)

const (
	threadModel     = openai.GPT4o
	completionModel = openai.GPT4o
	summaryModel    = openai.GPT4oMini
)

type Client interface {
	GetCompletion(msgList []openai.ChatCompletionMessage) (string, error)
	GetCompletionStream(msgList []openai.ChatCompletionMessage) chan string
	GetEmbedding(text string) ([]float32, error)
	QuickCompletion(systemPrompt string, userPrompt string) (string, error)
}

type OpenAIClient struct {
	config *config.Config
	client *openai.Client
	http   *http.Client
}

func NewOpenAIClient(conf *config.Config) *OpenAIClient {
	return &OpenAIClient{
		config: conf,
		client: openai.NewClient(conf.OpenAIApiKey),
		http:   &http.Client{},
	}
}

func (c *OpenAIClient) QuickCompletion(systemPrompt string, userPrompt string) (string, error) {
	res, err := c.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: summaryModel,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
				{Role: openai.ChatMessageRoleUser, Content: userPrompt},
			},
		},
	)

	if err != nil {
		errorMessage := fmt.Sprintf("Failed to get completion: %v", err)
		debug.Log("GPT: %s", errorMessage)
		return errorMessage, err
	}

	return fmt.Sprintf(res.Choices[0].Message.Content), nil
}
