package gpt

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"

	"github.com/sysread/fnord/pkg/debug"
)

const asstApiUri = apiBaseUri + "/assistants"
const assistantName = "Fnord Prefect"

// The version of the assistant. Stored in the assistant's metadata.
// Note that this must correspond to the version in assistant.json
const asstVersion = "1.0.0"

//go:embed assistant.json
var assistantFiles embed.FS
var assistantJSON []byte

var AssistantID string
var AssistantVersion string

type assistantInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Metadata map[string]string `json:"metadata"`
}

type findAssistantResponse struct {
	Data []assistantInfo `json:"data"`
}

func (c *OpenAIClient) initAssistant() error {
	var err error

	// Load the assistant JSON file
	assistantJSON, err = assistantFiles.ReadFile("assistant.json")
	if err != nil {
		panic(fmt.Errorf("failed to read assistant JSON file: %v", err))
	}

	if err := c.findAssistant(); err != nil {
		return c.createAssistant()
	}

	debug.Log("Found Assistant: %s", AssistantID)
	debug.Log("      - Version: %s", AssistantVersion)

	if AssistantVersion != asstVersion {
		debug.Log("Updating Assistant to version %s", asstVersion)
		return c.updateAssistant()
	}

	return nil
}

func (c *OpenAIClient) findAssistant() error {
	uri := asstApiUri + "?limit=100"

	// Build a request to list assistants
	req, err := c.makeRequest("GET", uri, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Perform the request
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %v", err)
	}
	if resp.StatusCode != 200 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to list assistants: %v - %s", resp.Status, msg)
	}

	// Read the response body
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	// Parse the response body
	var asstResp findAssistantResponse
	if err := json.Unmarshal(body, &asstResp); err != nil {
		return fmt.Errorf("failed to parse response body: %v", err)
	}

	// Find our assistant's ID
	for _, asst := range asstResp.Data {
		if asst.Name == assistantName {
			AssistantID = asst.ID
			AssistantVersion = asst.Metadata["version"]
			return nil
		}
	}

	return fmt.Errorf("assistant not found: %s", assistantName)
}

func (c *OpenAIClient) createAssistant() error {
	uri := asstApiUri

	// Build a request to create an assistant
	req, err := c.makeRequest("POST", uri, assistantJSON, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Perform the request
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %v", err)
	}
	if resp.StatusCode != 200 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create assistant: %v - %s", resp.Status, msg)
	}

	// Ensure the response body is closed
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	// Parse the response body
	var assistant assistantInfo
	if err := json.Unmarshal(body, &assistant); err != nil {
		return fmt.Errorf("failed to parse response body: %v", err)
	}

	// Find our assistant's ID
	AssistantID = assistant.ID

	return nil
}

func (c *OpenAIClient) updateAssistant() error {
	uri := asstApiUri + "/" + AssistantID

	// Build a request to update the assistant
	req, err := c.makeRequest("POST", uri, assistantJSON, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Perform the request
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %v", err)
	}
	if resp.StatusCode != 200 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update assistant: %v - %s", resp.Status, msg)
	}

	// Ensure the response body is closed
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	// Parse the response body
	var assistant assistantInfo
	if err := json.Unmarshal(body, &assistant); err != nil {
		return fmt.Errorf("failed to parse response body: %v", err)
	}

	return nil
}
