package gpt

import (
	"encoding/json"
	"fmt"
	"io"
)

const threadsApiUri = apiBaseUri + "/threads"
const threadModel = "gpt-4o"

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
		return "", fmt.Errorf("failed to create request: %#v", err)
	}

	// Perform the request
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %#v", err)
	}

	// Ensure the response body is closed
	defer resp.Body.Close()

	// Read and parse the body of the response. We only care about the thread
	// "id" field.
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", fmt.Errorf("failed to parse response body: %#v", err)
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
		return fmt.Errorf("body could not be serialized as json: %#v", err)
	}

	// Build a request to add a message to the thread
	req, err := c.makeRequest("POST", endpoint, jsonBody, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %#v", err)
	}

	// Perform the request
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %#v", err)
	}

	// Ensure the response body is closed
	defer resp.Body.Close()

	return nil
}

// CreateRun starts a new run in a previously created thread in the OpenAI API,
// and returns a channel that will receive the response content in string
// chunks.
func (c *OpenAIClient) CreateRun(threadID string) (io.ReadCloser, error) {
	endpoint := threadsApiUri + "/" + threadID + "/runs"

	// Build our request body
	body := struct {
		AssistantID string `json:"assistant_id"`
		Stream      bool   `json:"stream"`
	}{
		AssistantID: AssistantID,
		Stream:      true,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("body could not be serialized as json: %#v", err)
	}

	// Build a request to create a new run
	req, err := c.makeRequest("POST", endpoint, jsonBody, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %#v", err)
	}

	// Perform the request
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %#v", err)
	}
	if resp.StatusCode != 200 {
		msg, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to start thread run: %#v - %s", resp.Status, msg)
	}

	return resp.Body, nil
}

func (c *OpenAIClient) submitToolOutputs(threadID string, runID string, outputs []toolOutput) (io.ReadCloser, error) {
	endpoint := threadsApiUri + "/" + threadID + "/runs/" + runID + "/submit_tool_outputs"

	// Build the request body
	body := struct {
		Stream      bool          `json:"stream"`
		ToolOutputs []toolOutput `json:"tool_outputs"`
	}{
		Stream:      true,
		ToolOutputs: outputs,
	}

	// Serialize the body
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("body could not be serialized as json: %#v", err)
	}

	// Build a request to submit tool outputs
	req, err := c.makeRequest("POST", endpoint, jsonBody, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %#v", err)
	}

	// Perform the request
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %#v", err)
	}
	if resp.StatusCode != 200 {
		msg, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to submit tool outputs: %#v - %s", resp.Status, msg)
	}

	return resp.Body, nil
}
