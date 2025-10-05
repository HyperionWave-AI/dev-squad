package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

// QdrantClientInterface defines the interface for Qdrant operations
type QdrantClientInterface interface {
	EnsureCollection(collectionName string, vectorSize int) error
	StorePoint(collectionName string, id string, text string, metadata map[string]interface{}) error
	SearchSimilar(collectionName string, query string, limit int) ([]*QdrantQueryResult, error)
	Ping(ctx context.Context) error
}

// QdrantClient provides direct access to Qdrant vector database
type QdrantClient struct {
	baseURL         string
	httpClient      *http.Client
	embeddingFunc   func(string) ([]float64, error)
	openAIAPIKey    string
	openAIBaseURL   string
	embeddingModel  string
	vectorDimension int
}

// QdrantPoint represents a point to store in Qdrant
type QdrantPoint struct {
	ID      string                 `json:"id"`
	Vector  []float64              `json:"vector"`
	Payload map[string]interface{} `json:"payload"`
}

// QdrantSearchResult represents a search result from Qdrant
type QdrantSearchResult struct {
	ID      string                 `json:"id"`
	Score   float64                `json:"score"`
	Payload map[string]interface{} `json:"payload"`
}

// QdrantQueryResult wraps a search result with the knowledge entry
type QdrantQueryResult struct {
	Entry *KnowledgeEntry
	Score float64
}

// NewQdrantClient creates a new Qdrant client with OpenAI embeddings
func NewQdrantClient(baseURL string) *QdrantClient {
	apiKey := os.Getenv("OPENAI_API_KEY")

	client := &QdrantClient{
		baseURL:         baseURL,
		openAIAPIKey:    apiKey,
		openAIBaseURL:   "https://api.openai.com/v1",
		embeddingModel:  "text-embedding-3-small",
		vectorDimension: 1536,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Set embedding function based on API key availability
	if apiKey != "" {
		client.embeddingFunc = client.generateOpenAIEmbedding
	} else {
		// Fallback to simple embedding for testing
		client.embeddingFunc = func(text string) ([]float64, error) {
			return generateSimpleEmbedding(text, client.vectorDimension), nil
		}
	}

	return client
}

// NewQdrantClientWithEmbedding creates a Qdrant client with custom embedding function (for testing)
func NewQdrantClientWithEmbedding(baseURL string, embeddingFunc func(string) ([]float64, error), vectorDim int) *QdrantClient {
	return &QdrantClient{
		baseURL:         baseURL,
		embeddingFunc:   embeddingFunc,
		vectorDimension: vectorDim,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// EnsureCollection ensures a Qdrant collection exists
func (c *QdrantClient) EnsureCollection(collectionName string, vectorSize int) error {
	// Check if collection exists
	checkURL := fmt.Sprintf("%s/collections/%s", c.baseURL, collectionName)
	resp, err := c.httpClient.Get(checkURL)
	if err != nil {
		return fmt.Errorf("failed to check collection: %w", err)
	}
	defer resp.Body.Close()

	// If collection exists (200), return
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	// Create collection
	createPayload := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     vectorSize,
			"distance": "Cosine",
		},
	}

	payloadBytes, err := json.Marshal(createPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal create payload: %w", err)
	}

	createURL := fmt.Sprintf("%s/collections/%s", c.baseURL, collectionName)
	req, err := http.NewRequest("PUT", createURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err = c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create collection: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// StorePoint stores a point in Qdrant with text embedding
func (c *QdrantClient) StorePoint(collectionName string, id string, text string, metadata map[string]interface{}) error {
	// Generate embedding using configured function
	vector, err := c.embeddingFunc(text)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Create payload with text and metadata
	payload := make(map[string]interface{})
	payload["text"] = text
	payload["id"] = id
	payload["createdAt"] = time.Now().UTC().Format(time.RFC3339)

	// Merge metadata
	if metadata != nil {
		for k, v := range metadata {
			payload[k] = v
		}
	}

	// Create point
	point := QdrantPoint{
		ID:      id,
		Vector:  vector,
		Payload: payload,
	}

	// Upsert point
	upsertPayload := map[string]interface{}{
		"points": []QdrantPoint{point},
	}

	payloadBytes, err := json.Marshal(upsertPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal upsert payload: %w", err)
	}

	upsertURL := fmt.Sprintf("%s/collections/%s/points", c.baseURL, collectionName)
	req, err := http.NewRequest("PUT", upsertURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upsert point: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upsert point: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SearchSimilar searches for similar points in Qdrant
func (c *QdrantClient) SearchSimilar(collectionName string, query string, limit int) ([]*QdrantQueryResult, error) {
	// Generate query embedding using configured function
	queryVector, err := c.embeddingFunc(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Create search request
	searchPayload := map[string]interface{}{
		"vector": queryVector,
		"limit":  limit,
		"with_payload": true,
	}

	payloadBytes, err := json.Marshal(searchPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search payload: %w", err)
	}

	searchURL := fmt.Sprintf("%s/collections/%s/points/search", c.baseURL, collectionName)
	req, err := http.NewRequest("POST", searchURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var searchResponse struct {
		Result []QdrantSearchResult `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	// Convert to QueryResult format
	results := make([]*QdrantQueryResult, len(searchResponse.Result))
	for i, result := range searchResponse.Result {
		entry := &KnowledgeEntry{
			ID:         result.ID,
			Collection: collectionName,
			Text:       getStringFromPayload(result.Payload, "text"),
			Metadata:   result.Payload,
			CreatedAt:  parseTimeFromPayload(result.Payload, "createdAt"),
		}

		results[i] = &QdrantQueryResult{
			Entry: entry,
			Score: result.Score,
		}
	}

	return results, nil
}

// Helper functions

// generateSimpleEmbedding generates a simple hash-based embedding (placeholder)
// In production, this should use OpenAI or another embedding service
func generateSimpleEmbedding(text string, dimensions int) []float64 {
	vector := make([]float64, dimensions)

	// Simple hash-based embedding for testing
	// This is NOT a real embedding - just for demonstration
	for i := 0; i < dimensions; i++ {
		// Use text bytes to generate pseudo-random but deterministic values
		if i < len(text) {
			vector[i] = float64(text[i]) / 255.0
		} else {
			// Pad with values based on text length
			vector[i] = float64((i + len(text)) % 256) / 255.0
		}
	}

	return vector
}

// getStringFromPayload safely extracts a string from payload
func getStringFromPayload(payload map[string]interface{}, key string) string {
	if val, ok := payload[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// parseTimeFromPayload safely parses time from payload
func parseTimeFromPayload(payload map[string]interface{}, key string) time.Time {
	if val, ok := payload[key]; ok {
		if str, ok := val.(string); ok {
			if t, err := time.Parse(time.RFC3339, str); err == nil {
				return t
			}
		}
	}
	return time.Now().UTC()
}

// GenerateID generates a new UUID for Qdrant points
func GenerateID() string {
	return uuid.New().String()
}

// Ping checks Qdrant connectivity
func (c *QdrantClient) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/", c.baseURL), nil)
	if err != nil {
		return fmt.Errorf("failed to create ping request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to ping Qdrant: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Qdrant ping failed with status: %d", resp.StatusCode)
	}

	return nil
}
