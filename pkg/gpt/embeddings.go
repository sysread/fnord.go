package gpt

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sysread/fnord/pkg/debug"
)

const embeddingsModel = "text-embedding-3-large"
const embeddingsEndpoint = apiBaseUri + "/embeddings"

type embeddingResponse struct {
	Data   []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

type embeddingRequest struct {
	Input           string `json:"input"`
	Model           string `json:"model"`
	EncodingFormat  string `json:"encoding_format"`
	Dimensions      int    `json:"dimensions"`
}

func (c *OpenAIClient) GetEmbedding(text string) ([]float32, error) {
	endpoint := embeddingsEndpoint

	// Build the request body
	body := embeddingRequest{
		Input:          text,
		Model:          embeddingsModel,
		EncodingFormat: "float",
		Dimensions:     1536,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("body could not be serialized as json: %v", err)
	}

	// Build a request to get the embedding
	req, err := c.makeRequest("POST", endpoint, jsonBody, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Perform the request
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	// Ensure the response body is closed
	defer resp.Body.Close()

	// Read and parse the body of the response. We care about data.embedding,
	// which contains an array of floats.
	var response embeddingResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response body: %v", err)
	}

	return response.Data[0].Embedding, nil
}
