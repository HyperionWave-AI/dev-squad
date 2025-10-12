package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	CodeIndexCollection = "code_index"
	VectorSize          = 1536 // OpenAI text-embedding-3-small dimension
)

// QdrantClient handles communication with Qdrant vector database
type QdrantClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// QdrantPoint represents a point in Qdrant
type QdrantPoint struct {
	ID      string                 `json:"id"`
	Vector  []float32              `json:"vector"`
	Payload map[string]interface{} `json:"payload"`
}

// QdrantSearchRequest represents a search request
type QdrantSearchRequest struct {
	Vector      []float32 `json:"vector"`
	Limit       int       `json:"limit"`
	WithPayload bool      `json:"with_payload"`
	WithVector  bool      `json:"with_vector"`
}

// QdrantSearchResponse represents a search response
type QdrantSearchResponse struct {
	Result []struct {
		ID      string                 `json:"id"`
		Score   float32                `json:"score"`
		Payload map[string]interface{} `json:"payload"`
		Vector  []float32              `json:"vector,omitempty"`
	} `json:"result"`
}

// NewQdrantClient creates a new Qdrant client
func NewQdrantClient(baseURL, apiKey string) *QdrantClient {
	return &QdrantClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// EnsureCollection creates the collection if it doesn't exist
func (c *QdrantClient) EnsureCollection() error {
	// Check if collection exists
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/collections/%s", c.baseURL, CodeIndexCollection), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if c.apiKey != "" {
		req.Header.Set("api-key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to check collection: %w", err)
	}
	defer resp.Body.Close()

	// Collection exists
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	// Create collection
	collectionConfig := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     VectorSize,
			"distance": "Cosine",
		},
	}

	jsonBody, err := json.Marshal(collectionConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal collection config: %w", err)
	}

	req, err = http.NewRequest("PUT", fmt.Sprintf("%s/collections/%s", c.baseURL, CodeIndexCollection), bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("api-key", c.apiKey)
	}

	resp, err = c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create collection (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpsertPoints upserts points into the collection
func (c *QdrantClient) UpsertPoints(points []QdrantPoint) error {
	requestBody := map[string]interface{}{
		"points": points,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal points: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points?wait=true", c.baseURL, CodeIndexCollection)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("api-key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upsert points: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upsert points (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Search performs a vector similarity search
func (c *QdrantClient) Search(vector []float32, limit int) (*QdrantSearchResponse, error) {
	searchReq := QdrantSearchRequest{
		Vector:      vector,
		Limit:       limit,
		WithPayload: true,
		WithVector:  false,
	}

	jsonBody, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/search", c.baseURL, CodeIndexCollection)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("api-key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed (status %d): %s", resp.StatusCode, string(body))
	}

	var searchResp QdrantSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	return &searchResp, nil
}

// DeletePoints deletes points by IDs
func (c *QdrantClient) DeletePoints(pointIDs []string) error {
	requestBody := map[string]interface{}{
		"points": pointIDs,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/delete?wait=true", c.baseURL, CodeIndexCollection)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("api-key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete points: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete points (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpsertPoint upserts a single point into the collection
func (c *QdrantClient) UpsertPoint(id string, vector []float32, payload map[string]interface{}) error {
	point := QdrantPoint{
		ID:      id,
		Vector:  vector,
		Payload: payload,
	}
	return c.UpsertPoints([]QdrantPoint{point})
}

// DeletePoint deletes a single point by ID
func (c *QdrantClient) DeletePoint(pointID string) error {
	return c.DeletePoints([]string{pointID})
}

// DeleteByFilter deletes points matching a filter
func (c *QdrantClient) DeleteByFilter(filter map[string]interface{}) error {
	requestBody := map[string]interface{}{
		"filter": filter,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/delete?wait=true", c.baseURL, CodeIndexCollection)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("api-key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete by filter: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete by filter (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
