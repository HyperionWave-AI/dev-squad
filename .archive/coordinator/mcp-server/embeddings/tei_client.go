package embeddings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TEIClient handles communication with Hugging Face Text Embeddings Inference (TEI) service
// Compatible with models like nomic-ai/nomic-embed-text-v1.5
type TEIClient struct {
	baseURL    string
	httpClient *http.Client
	dimensions int
}

// TEIRequest represents the request payload for TEI embeddings API
type TEIRequest struct {
	Inputs []string `json:"inputs"`
}

// TEIResponse is the direct response from TEI - array of embeddings
type TEIResponse [][]float32

// NewTEIClient creates a new TEI client for embeddings
// baseURL should be like "http://embedding-service:8080" (no /embed suffix)
func NewTEIClient(baseURL string) *TEIClient {
	return &TEIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // Increased for CPU-based TEI inference (can take 30s+ with queue)
		},
		dimensions: 768, // nomic-embed-text-v1.5 dimension
	}
}

// CreateEmbedding generates an embedding vector for the given text
func (c *TEIClient) CreateEmbedding(text string) ([]float32, error) {
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
func (c *TEIClient) CreateEmbeddings(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided")
	}

	reqBody := TEIRequest{
		Inputs: texts,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/embed", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to TEI service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TEI service error (status %d): %s", resp.StatusCode, string(body))
	}

	var embeddings TEIResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddings); err != nil {
		return nil, fmt.Errorf("failed to decode TEI response: %w", err)
	}

	if len(embeddings) != len(texts) {
		return nil, fmt.Errorf("TEI returned %d embeddings for %d texts", len(embeddings), len(texts))
	}

	return [][]float32(embeddings), nil
}

// GetDimensions returns the number of dimensions for the embedding model
func (c *TEIClient) GetDimensions() int {
	return c.dimensions
}
