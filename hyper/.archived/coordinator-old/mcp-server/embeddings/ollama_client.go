package embeddings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaClient provides embeddings using Ollama
// Ollama runs llama.cpp with GPU acceleration (Metal/CUDA/Vulkan) as a local service
// No CGO required - uses REST API
type OllamaClient struct {
	baseURL    string
	model      string
	dimensions int
	httpClient *http.Client
}

// OllamaEmbeddingRequest represents the request to Ollama embeddings API
type OllamaEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// OllamaEmbeddingResponse represents the response from Ollama embeddings API
type OllamaEmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

// NewOllamaClient creates a new Ollama embedding client
// baseURL: Ollama API endpoint (default: http://localhost:11434)
// model: embedding model to use (default: nomic-embed-text)
func NewOllamaClient(baseURL, model string) (*OllamaClient, error) {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "nomic-embed-text"
	}

	client := &OllamaClient{
		baseURL:    baseURL,
		model:      model,
		dimensions: 0, // Will be auto-detected
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Test connection to Ollama
	if err := client.testConnection(); err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama: %w\nMake sure Ollama is running: brew services start ollama", err)
	}

	// Auto-detect embedding dimensions by making a test embedding call
	if err := client.detectDimensions(); err != nil {
		return nil, fmt.Errorf("failed to detect embedding dimensions: %w", err)
	}

	return client, nil
}

// testConnection verifies Ollama is running and accessible
func (c *OllamaClient) testConnection() error {
	resp, err := c.httpClient.Get(c.baseURL + "/api/tags")
	if err != nil {
		return fmt.Errorf("Ollama not reachable at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	return nil
}

// detectDimensions auto-detects embedding dimensions by making a test call
func (c *OllamaClient) detectDimensions() error {
	// Create a test embedding with a simple text
	testEmbedding, err := c.CreateEmbedding("test")
	if err != nil {
		return fmt.Errorf("failed to create test embedding: %w", err)
	}

	c.dimensions = len(testEmbedding)
	if c.dimensions == 0 {
		return fmt.Errorf("detected 0 dimensions for model %s", c.model)
	}

	return nil
}

// CreateEmbedding generates a single embedding vector for the given text
func (c *OllamaClient) CreateEmbedding(text string) ([]float32, error) {
	reqBody := OllamaEmbeddingRequest{
		Model:  c.model,
		Prompt: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/api/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Ollama API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result OllamaEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Embedding) == 0 {
		return nil, fmt.Errorf("Ollama returned empty embedding")
	}

	return result.Embedding, nil
}

// CreateEmbeddings generates embedding vectors for multiple texts
func (c *OllamaClient) CreateEmbeddings(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided")
	}

	results := make([][]float32, len(texts))

	// Process each text sequentially
	// Ollama handles batching and GPU optimization internally
	for i, text := range texts {
		embedding, err := c.CreateEmbedding(text)
		if err != nil {
			return nil, fmt.Errorf("failed to create embedding for text %d: %w", i, err)
		}
		results[i] = embedding
	}

	return results, nil
}

// GetDimensions returns the number of dimensions for the embedding model
// Dimensions are auto-detected during client initialization:
// - nomic-embed-text: 768 dimensions
// - all-minilm: 384 dimensions
// - embeddinggemma: 768 dimensions (Google Gemma-based)
func (c *OllamaClient) GetDimensions() int {
	return c.dimensions
}
