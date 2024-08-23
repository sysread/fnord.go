package gpt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	openai "github.com/sashabaranov/go-openai"

	"github.com/sysread/fnord/pkg/messages"
)

const (
	completionModel string                = openai.GPT4o
	summaryModel    string                = openai.GPT4oMini
	embeddingsModel openai.EmbeddingModel = openai.LargeEmbedding3
)

const systemSummaryPrompt = `
Your job is to summarize a conversation.
It is essential that you identify all significant facts in the conversation transcript.
You will assemble a nested an outline of this conversation in markdown format.
If there is file content present, be sure to include the file path and an individual summary of the file content as a distinct set of nested list items.
If there is command output, include the command, a the relevance of its output, and then VERY tersely summarize how it relates to the conversation.
Respond ONLY with a summary of the discussion, followed by your outline of ALL facts identified in the conversation.
`

type Client interface {
	GetCompletion(conversation messages.Conversation) (string, error)
	GetCompletionStream(conversation messages.Conversation) chan string
	GetEmbedding(text string) ([]float32, error)
	GetSummary(conversation messages.Conversation) (string, error)
	QuickCompletion(systemPrompt string, userPrompt string) (string, error)
}

type OpenAIClient struct {
	client *openai.Client
}

func NewOpenAIClient() *OpenAIClient {
	apiKey := os.Getenv("OPENAI_API_KEY")
	client := openai.NewClient(apiKey)
	return &OpenAIClient{client: client}
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
		return errorMessage, err
	}

	return fmt.Sprintf(res.Choices[0].Message.Content), nil
}

func (c *OpenAIClient) GetSummary(conversation messages.Conversation) (string, error) {
	userPrompt := conversation.ChatTranscript()
	return c.QuickCompletion(systemSummaryPrompt, userPrompt)
}

func (c *OpenAIClient) GetCompletion(conversation messages.Conversation) (string, error) {
	res, err := c.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    completionModel,
			Messages: conversation.ChatCompletionMessages(),
		},
	)

	if err != nil {
		errorMessage := fmt.Sprintf("Failed to get completion: %v", err)
		return errorMessage, err
	}

	return fmt.Sprintf(res.Choices[0].Message.Content), nil
}

func (c *OpenAIClient) GetCompletionStream(conversation messages.Conversation) chan string {
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

func (c *OpenAIClient) GetEmbedding(text string) ([]float32, error) {
	request := openai.EmbeddingRequest{
		Input:          text,
		Model:          embeddingsModel,
		Dimensions:     1536,
		EncodingFormat: openai.EmbeddingEncodingFormatFloat,
	}

	response, err := c.client.CreateEmbeddings(context.Background(), request)
	if err != nil {
		return nil, fmt.Errorf("error generating embeddings: %w", err)
	}

	return response.Data[0].Embedding, nil
}
