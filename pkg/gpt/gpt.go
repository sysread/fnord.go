package gpt

import (
	"net/http"

	openai "github.com/sashabaranov/go-openai"

	"github.com/sysread/fnord/pkg/config"
)

type Client interface {
	GetCompletion(msgList []openai.ChatCompletionMessage) (string, error)
	GetEmbedding(text string) ([]float32, error)
	CreateThread() (string, error)
	AddMessage(threadID string, content string) error
	RunThread(threadID string) (chan string, error)
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
