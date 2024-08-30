package gpt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	openai "github.com/sashabaranov/go-openai"

	"github.com/sysread/fnord/pkg/config"
	"github.com/sysread/fnord/pkg/debug"
)

const (
	threadModel     string                = openai.GPT4o
	completionModel string                = openai.GPT4o
	summaryModel    string                = openai.GPT4oMini
	embeddingsModel openai.EmbeddingModel = openai.LargeEmbedding3
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

func (c *OpenAIClient) GetCompletion(msgList []openai.ChatCompletionMessage) (string, error) {
	res, err := c.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    completionModel,
			Messages: msgList,
		},
	)

	if err != nil {
		errorMessage := fmt.Sprintf("Failed to get completion: %v", err)
		debug.Log("GPT: %s", errorMessage)
		return errorMessage, err
	}

	return fmt.Sprintf(res.Choices[0].Message.Content), nil
}

func (c *OpenAIClient) GetCompletionStream(msgList []openai.ChatCompletionMessage) chan string {
	out := make(chan string)

	go func() {
		stream, err := c.client.CreateChatCompletionStream(
			context.Background(),
			openai.ChatCompletionRequest{
				Model:    openai.GPT3Dot5Turbo,
				Messages: msgList,
				Stream:   true,
			},
		)

		if err != nil {
			errorMessage := fmt.Sprintf("Failed to get completion stream: %v", err)
			debug.Log("GPT: %s", errorMessage)
			out <- fmt.Sprintf("[red:-:-]%s[-:-:-]", errorMessage)
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

func (c *OpenAIClient) GetEmbedding(text string) ([]float32, error) {
	request := openai.EmbeddingRequest{
		Input:          text,
		Model:          embeddingsModel,
		Dimensions:     1536,
		EncodingFormat: openai.EmbeddingEncodingFormatFloat,
	}

	response, err := c.client.CreateEmbeddings(context.Background(), request)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to get embeddings: %v", err)
		debug.Log("GPT: %s", errorMessage)
		return nil, err
	}

	return response.Data[0].Embedding, nil
}
