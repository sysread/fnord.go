package gpt

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const threadsApiUri = apiBaseUri + "/threads"

// Represents the response body of a streaming request.
type threadResponse struct {
	Delta threadStreamingResponseDelta `json:"delta"`
}

// Represents one chunk of the response body of a streaming request.
type threadStreamingResponseDelta struct {
	Content []struct {
		Index int    `json:"index"`
		Type  string `json:"type"`
		Text  struct {
			Value       string        `json:"value"`
			Annotations []interface{} `json:"annotations,omitempty"`
		} `json:"text"`
	} `json:"content"`
}

// CreateThread creates a new thread in the OpenAI API and returns the thread ID.
func (c *OpenAIClient) CreateThread() (string, error) {
	endpoint := threadsApiUri

	// Build a request to create new thread
	req, err := c.makeRequest("POST", endpoint, nil, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// Perform the request
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %v", err)
	}

	// Ensure the response body is closed
	defer resp.Body.Close()

	// Read and parse the body of the response. We only care about the thread
	// "id" field.
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", fmt.Errorf("failed to parse response body: %v", err)
	}

	threadId, ok := response["id"].(string)
	if !ok {
		return "", fmt.Errorf("response did not contain a thread id")
	}

	return threadId, nil
}

// AddMessage adds a message to a previously created thread in the OpenAI API.
func (c *OpenAIClient) AddMessage(threadID string, content string) error {
	endpoint := threadsApiUri + "/" + threadID + "/messages"

	// Build our request body
	body := map[string]string{
		"role":    "user",
		"content": content,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("body could not be serialized as json: %v", err)
	}

	// Build a request to add a message to the thread
	req, err := c.makeRequest("POST", endpoint, jsonBody, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Perform the request
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %v", err)
	}

	// Ensure the response body is closed
	defer resp.Body.Close()

	return nil
}

// RunThread starts a new run in a previously created thread in the OpenAI API,
// and returns a channel that will receive the response content in string
// chunks.
func (c *OpenAIClient) RunThread(threadID string) (chan string, error) {
	endpoint := threadsApiUri + "/" + threadID + "/runs"

	// Build our request body
	body := struct {
		AssistantID string `json:"assistant_id"`
		Stream      bool   `json:"stream"`
	}{
		AssistantID: c.config.OpenAIAsstId,
		Stream:      true,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("body could not be serialized as json: %v", err)
	}

	// Build a request to create a new run
	req, err := c.makeRequest("POST", endpoint, jsonBody, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Perform the request
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}

	responseChan := make(chan string)

	// Start a goroutine to stream the response body
	go streamThreadRunResponse(resp.Body, responseChan)

	return responseChan, nil
}

// streamResponse reads the response body of a streaming request and sends each
// piece of content to the provided channel.
func streamThreadRunResponse(body io.ReadCloser, deltaChan chan string) {
	defer close(deltaChan)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	var currentEvent string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event:") {
			currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

			if currentEvent == "thread.message.delta" {
				// Parse the JSON-encoded delta
				var response threadResponse

				if err := json.Unmarshal([]byte(data), &response); err != nil {
					// Handle JSON parse error (optional)
					fmt.Println("error parsing delta:", err)
					continue
				}

				delta := response.Delta

				// Send each piece of content from the delta to the channel
				for _, content := range delta.Content {
					deltaChan <- content.Text.Value
				}
			} else if currentEvent == "done" && data == "[DONE]" {
				// If the "done" event is received, break the loop to close the
				// channel.
				break
			}
		}
	}
}
