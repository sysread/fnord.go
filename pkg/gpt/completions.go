package gpt

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const completionsApiUri = apiBaseUri + "/chat/completions"
const completionModel = "gpt-4o-mini"

type completionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type completionRequest struct {
	Model    string              `json:"model"`
	Messages []completionMessage `json:"messages"`
}

type completionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (c *OpenAIClient) GetCompletion(systemPrompt string, userPrompt string) (string, error) {
	endpoint := completionsApiUri

	// Build the request body
	body := completionRequest{
		Model:    completionModel,
		Messages: []completionMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("body could not be serialized as json: %v", err)
	}

	// Build a request to get the completion
	req, err := c.makeRequest("POST", endpoint, jsonBody, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// Perform the request
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	// Ensure the response body is closed
	defer resp.Body.Close()

	// Read and parse the body of the response. We only care about the completion
	// "content" field.
	var response completionResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", fmt.Errorf("failed to parse response body: %v", err)
	}

	return response.Choices[0].Message.Content, nil
}
