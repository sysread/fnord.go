package gpt

import (
	"context"
	"fmt"
	"os"

	openai "github.com/sashabaranov/go-openai"

	"github.com/sysread/fnord/pkg/common"
)

type OpenAIClient struct {
	client *openai.Client
}

func NewOpenAIClient() *OpenAIClient {
	apiKey := os.Getenv("OPENAI_API_KEY")
	client := openai.NewClient(apiKey)
	return &OpenAIClient{client: client}
}

func (c *OpenAIClient) GetCompletion(conversation []common.ChatMessage) (string, error) {
	messages := []openai.ChatCompletionMessage{}
	for _, message := range conversation {
		messages = append(messages, message.ApiMessage())
	}

	res, err := c.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT4o,
			Messages: messages,
		},
	)

	if err != nil {
		errorMessage := fmt.Sprintf("Failed to get completion: %v", err)
		return errorMessage, err
	}

	return fmt.Sprintf(res.Choices[0].Message.Content), nil
}
