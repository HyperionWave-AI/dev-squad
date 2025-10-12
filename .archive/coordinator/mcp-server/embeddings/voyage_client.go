package embeddings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// VoyageClient handles communication with Voyage AI embeddings API
// Anthropic's recommended embedding provider
type VoyageClient struct {
	apiKey     string
	httpClient *http.Client
	model      string
	dimensions int
}

// VoyageRequest represents the request payload for Voyage AI embeddings API
type VoyageRequest struct {
	Input     []string `json:"input"`
	Model     string   `json:"model"`
	InputType string   `json:"input_type"` // "document" for code chunks
}

// VoyageResponse is the response from Voyage AI
type VoyageResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

// NewVoyageClient creates a new Voyage AI client for embeddings
// Uses voyage-3 model by default (1024 dimensions, best price/performance)
func NewVoyageClient(apiKey string) *VoyageClient {
	return &VoyageClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		model:      "voyage-3",    // Best price/performance: $0.06/1M tokens
		dimensions: 1024,           // voyage-3 dimensions
	}
}

// NewVoyageClientWithModel creates a Voyage AI client with a specific model
// Available models: voyage-3, voyage-3-large, voyage-3.5, voyage-3.5-lite, voyage-code-3
func NewVoyageClientWithModel(apiKey, model string) *VoyageClient {
	client := NewVoyageClient(apiKey)
	client.model = model

	// Set dimensions based on model
	switch model {
	case "voyage-3", "voyage-3.5":
		client.dimensions = 1024
	case "voyage-3-large", "voyage-3.5-large":
		client.dimensions = 1024 // Can be configured smaller
	case "voyage-code-3":
		client.dimensions = 1024
	case "voyage-3.5-lite":
		client.dimensions = 512
	default:
		client.dimensions = 1024 // Default fallback
	}

	return client
}

// CreateEmbedding generates a single embedding vector for the given text
func (c *VoyageClient) CreateEmbedding(text string) ([]float32, error) {
	embeddings, err := c.CreateEmbeddings([]string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	return embeddings[0], nil
}

// CreateEmbeddings generates embedding vectors for multiple texts
func (c *VoyageClient) CreateEmbeddings(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided")
	}

	reqBody := VoyageRequest{
		Input:     texts,
		Model:     c.model,
		InputType: "document", // For code chunks (vs "query" for search queries)
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.voyageai.com/v1/embeddings", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to Voyage AI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Voyage AI error (status %d): %s", resp.StatusCode, string(body))
	}

	var voyageResp VoyageResponse
	if err := json.NewDecoder(resp.Body).Decode(&voyageResp); err != nil {
		return nil, fmt.Errorf("failed to decode Voyage AI response: %w", err)
	}

	if len(voyageResp.Data) != len(texts) {
		return nil, fmt.Errorf("Voyage AI returned %d embeddings for %d texts", len(voyageResp.Data), len(texts))
	}

	// Extract embeddings in order (sorted by index)
	embeddings := make([][]float32, len(texts))
	for _, item := range voyageResp.Data {
		if item.Index >= len(texts) {
			return nil, fmt.Errorf("invalid index %d in response", item.Index)
		}
		embeddings[item.Index] = item.Embedding
	}

	return embeddings, nil
}

// GetDimensions returns the number of dimensions for the embedding model
func (c *VoyageClient) GetDimensions() int {
	return c.dimensions
}
