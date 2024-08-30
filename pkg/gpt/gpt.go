package gpt

import (
	"net/http"

	"github.com/sysread/fnord/pkg/config"
)

type Client interface {
	GetCompletion(systemPrompt string, userPrompt string) (string, error)
	GetEmbedding(text string) ([]float32, error)
	CreateThread() (string, error)
	AddMessage(threadID string, content string) error
	RunThread(threadID string) (chan string, error)
}

type OpenAIClient struct {
	config *config.Config
	http   *http.Client
}

func NewOpenAIClient(conf *config.Config) *OpenAIClient {
	return &OpenAIClient{
		config: conf,
		http:   &http.Client{},
	}
}
