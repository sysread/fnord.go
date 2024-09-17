package gpt

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/sysread/fnord/pkg/debug"
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
	debug.Log("[gpt] Starting new thread")

	endpoint := threadsApiUri

	// Build a request to create new thread
	req, err := c.makeRequest("POST", endpoint, nil, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %#v", err)
	}

	// Perform the request
	resp, err := c.http.Do(req)
	if err != nil {
		debug.Log("[gpt] Failed to make request: %#v", err)
		return "", fmt.Errorf("failed to make request: %#v", err)
	}

	// Ensure the response body is closed
	defer resp.Body.Close()

	// Read and parse the body of the response. We only care about the thread
	// "id" field.
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		debug.Log("[gpt] Failed to parse response body: %#v", err)
		return "", fmt.Errorf("failed to parse response body: %#v", err)
	}

	threadId, ok := response["id"].(string)
	if !ok {
		debug.Log("[gpt] Response did not contain a thread id")
		return "", fmt.Errorf("response did not contain a thread id")
	}

	debug.Log("[gpt] Thread created: %s", threadId)

	return threadId, nil
}

// AddMessage adds a message to a previously created thread in the OpenAI API.
func (c *OpenAIClient) AddMessage(threadID string, content string) error {
	// Truncate the content for logging, and handle the case where the content
	// is fewer than 100 characters.
	debug.Log("[gpt] Adding message to thread %s: %.100s", threadID, content)

	endpoint := threadsApiUri + "/" + threadID + "/messages"

	// Build our request body
	body := map[string]string{
		"role":    "user",
		"content": content,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		debug.Log("[gpt] Body could not be serialized as json: %#v", err)
		return fmt.Errorf("body could not be serialized as json: %#v", err)
	}

	// Build a request to add a message to the thread
	req, err := c.makeRequest("POST", endpoint, jsonBody, nil)
	if err != nil {
		debug.Log("[gpt] Failed to create request: %#v", err)
		return fmt.Errorf("failed to create request: %#v", err)
	}

	// Perform the request
	resp, err := c.http.Do(req)
	if err != nil {
		debug.Log("[gpt] Failed to make request: %#v", err)
		return fmt.Errorf("failed to make request: %#v", err)
	}

	// Ensure the response body is closed
	defer resp.Body.Close()

	debug.Log("[gpt] Message added to thread %s", threadID)

	return nil
}

// CreateRun starts a new run in a previously created thread in the OpenAI API,
// and returns a channel that will receive the response content in string
// chunks.
func (c *OpenAIClient) CreateRun(threadID string) (io.ReadCloser, error) {
	debug.Log("[gpt] Creating thread run %s", threadID)

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
		debug.Log("[gpt] Body could not be serialized as json: %#v", err)
		return nil, fmt.Errorf("body could not be serialized as json: %#v", err)
	}

	// Build a request to create a new run
	req, err := c.makeRequest("POST", endpoint, jsonBody, nil)
	if err != nil {
		debug.Log("[gpt] Failed to create request: %#v", err)
		return nil, fmt.Errorf("failed to create request: %#v", err)
	}

	// Perform the request
	resp, err := c.http.Do(req)
	if err != nil {
		debug.Log("[gpt] Failed to make request: %#v", err)
		return nil, fmt.Errorf("failed to make request: %#v", err)
	}
	if resp.StatusCode != 200 {
		msg, _ := io.ReadAll(resp.Body)
		debug.Log("[gpt] Failed to start thread run: %#v - %s", resp.Status, msg)
		return nil, fmt.Errorf("failed to start thread run: %#v - %s", resp.Status, msg)
	}

	debug.Log("[gpt] Thread run created for thread %s", threadID)
	return resp.Body, nil
}

func (c *OpenAIClient) submitToolOutputs(threadID string, runID string, outputs []toolOutput) (io.ReadCloser, error) {
	debug.Log("[gpt] Submitting tool outputs for thread %s run %s", threadID, runID)

	endpoint := threadsApiUri + "/" + threadID + "/runs/" + runID + "/submit_tool_outputs"

	// Build the request body
	body := struct {
		Stream      bool         `json:"stream"`
		ToolOutputs []toolOutput `json:"tool_outputs"`
	}{
		Stream:      true,
		ToolOutputs: outputs,
	}

	// Serialize the body
	jsonBody, err := json.Marshal(body)
	if err != nil {
		debug.Log("[gpt] Body could not be serialized as json: %#v", err)
		return nil, fmt.Errorf("body could not be serialized as json: %#v", err)
	}

	// Build a request to submit tool outputs
	req, err := c.makeRequest("POST", endpoint, jsonBody, nil)
	if err != nil {
		debug.Log("[gpt] Failed to create request: %#v", err)
		return nil, fmt.Errorf("failed to create request: %#v", err)
	}

	// Perform the request
	resp, err := c.http.Do(req)
	if err != nil {
		debug.Log("[gpt] Failed to make request: %#v", err)
		return nil, fmt.Errorf("failed to make request: %#v", err)
	}
	if resp.StatusCode != 200 {
		msg, _ := io.ReadAll(resp.Body)
		debug.Log("[gpt] Failed to submit tool outputs: %#v - %s", resp.Status, msg)
		return nil, fmt.Errorf("failed to submit tool outputs: %#v - %s", resp.Status, msg)
	}

	debug.Log("[gpt] Tool outputs submitted for thread %s run %s", threadID, runID)
	return resp.Body, nil
}
