package gpt

import (
	"bytes"
	"fmt"
	"net/http"
)

const apiBaseUri = "https://api.openai.com/v1"

// makeRequest creates an HTTP request with the given method, URI, body, and
// headers.
func (c *OpenAIClient) makeRequest(method string, uri string, body []byte, headers map[string]string) (*http.Request, error) {
	req, err := http.NewRequest(method, uri, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set common headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.OpenAIApiKey)
	req.Header.Set("OpenAI-Beta", "assistants=v2")

	// Set user headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return req, nil
}
