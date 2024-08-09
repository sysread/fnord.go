package gpt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

type OpenAIClient struct {
	client *openai.Client
}

func NewOpenAIClient() *OpenAIClient {
	apiKey := os.Getenv("OPENAI_API_KEY")
	client := openai.NewClient(apiKey)
	return &OpenAIClient{client: client}
}

func (c *OpenAIClient) GetCompletion(conversation Conversation) (string, error) {
	res, err := c.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT4o,
			Messages: conversation.ChatCompletionMessages(),
		},
	)

	if err != nil {
		errorMessage := fmt.Sprintf("Failed to get completion: %v", err)
		return errorMessage, err
	}

	return fmt.Sprintf(res.Choices[0].Message.Content), nil
}

func (c *OpenAIClient) GetCompletionStream(conversation Conversation) chan string {
	out := make(chan string)

	go func() {
		stream, err := c.client.CreateChatCompletionStream(
			context.Background(),
			openai.ChatCompletionRequest{
				Model:    openai.GPT3Dot5Turbo,
				Messages: conversation.ChatCompletionMessages(),
				Stream:   true,
			},
		)

		if err != nil {
			fmt.Printf("Failed to get completion stream: %v\n", err)
			close(out)
			return
		}

		defer stream.Close()
		defer close(out)

		for {
			var response openai.ChatCompletionStreamResponse

			response, err = stream.Recv()

			// response stream complete
			if errors.Is(err, io.EOF) {
				return
			}

			// actual error
			if err != nil {
				fmt.Printf("Stream error: %v\n", err)
				return
			}

			// Send the content to the channel
			out <- response.Choices[0].Delta.Content
		}
	}()

	return out
}
