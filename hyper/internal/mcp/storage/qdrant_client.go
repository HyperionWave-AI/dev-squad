package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"hyper/internal/mcp/embeddings"

	"github.com/google/uuid"
)

// QdrantClientInterface defines the interface for Qdrant operations
type QdrantClientInterface interface {
	EnsureCollection(collectionName string, vectorSize int) error
	StorePoint(collectionName string, id string, text string, metadata map[string]interface{}) error
	SearchSimilar(collectionName string, query string, limit int, voteBoost ...float64) ([]*QdrantQueryResult, error)
	SearchSimilarWithFilter(collectionName string, query string, limit int, filter map[string]interface{}, voteBoost ...float64) ([]*QdrantQueryResult, error)
	SearchWithVoteFilter(collectionName string, query string, limit int, minVoteScore int, voteBoost ...float64) ([]*QdrantQueryResult, error)
	DeletePoint(collectionName string, pointID string) error
	DeleteCollection(collectionName string) error
	UpdatePointPayload(collectionName string, pointID string, payload map[string]interface{}) error
	RecreateCollectionWithReindex(collectionName string, entries []*KnowledgeEntry, dimensions int) (int, error)
	GetDimensions() int
	Ping(ctx context.Context) error
}

// QdrantClient provides direct access to Qdrant vector database
type QdrantClient struct {
	baseURL                  string
	httpClient               *http.Client
	embeddingFunc            func(string) ([]float64, error)
	qdrantAPIKey             string
	teiClient                *embeddings.TEIClient
	vectorDimension          int
	knowledgeCollectionName  string // Configurable knowledge collection name
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

// NewQdrantClient creates a new Qdrant client with TEI embeddings
func NewQdrantClient(baseURL string, knowledgeCollectionName string) *QdrantClient {
	qdrantKey := os.Getenv("QDRANT_API_KEY")
	teiURL := os.Getenv("TEI_URL")
	if teiURL == "" {
		teiURL = "http://embedding-service:8080"
	}

	// Default collection name if not provided
	if knowledgeCollectionName == "" {
		knowledgeCollectionName = "dev_squad_knowledge"
	}

	teiClient := embeddings.NewTEIClient(teiURL)

	client := &QdrantClient{
		baseURL:                 baseURL,
		qdrantAPIKey:            qdrantKey,
		teiClient:               teiClient,
		vectorDimension:         768, // TEI nomic-embed-text-v1.5 dimension
		knowledgeCollectionName: knowledgeCollectionName,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Set embedding function to use TEI
	client.embeddingFunc = client.generateTEIEmbedding

	return client
}

// NewQdrantClientWithEmbedding creates a Qdrant client with custom embedding function (for testing)
func NewQdrantClientWithEmbedding(baseURL string, embeddingFunc func(string) ([]float64, error), vectorDim int) *QdrantClient {
	qdrantKey := os.Getenv("QDRANT_API_KEY")

	return &QdrantClient{
		baseURL:         baseURL,
		qdrantAPIKey:    qdrantKey,
		embeddingFunc:   embeddingFunc,
		vectorDimension: vectorDim,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewQdrantClientWithEmbeddingClient creates a Qdrant client using an external embedding client
// This allows using the same embedding client (Ollama, OpenAI, Voyage, etc.) for both code indexing and knowledge/tools storage
func NewQdrantClientWithEmbeddingClient(baseURL string, knowledgeCollectionName string, embeddingClient embeddings.EmbeddingClient) *QdrantClient {
	qdrantKey := os.Getenv("QDRANT_API_KEY")

	// Default collection name if not provided
	if knowledgeCollectionName == "" {
		knowledgeCollectionName = "dev_squad_knowledge"
	}

	client := &QdrantClient{
		baseURL:                 baseURL,
		qdrantAPIKey:            qdrantKey,
		vectorDimension:         embeddingClient.GetDimensions(),
		knowledgeCollectionName: knowledgeCollectionName,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Set embedding function to use the provided embedding client
	// Convert float32 embeddings to float64 for Qdrant compatibility
	client.embeddingFunc = func(text string) ([]float64, error) {
		embedding32, err := embeddingClient.CreateEmbedding(text)
		if err != nil {
			return nil, err
		}

		// Convert []float32 to []float64
		embedding64 := make([]float64, len(embedding32))
		for i, v := range embedding32 {
			embedding64[i] = float64(v)
		}
		return embedding64, nil
	}

	return client
}

// GetDimensions returns the vector dimension configured for this client
func (c *QdrantClient) GetDimensions() int {
	return c.vectorDimension
}

// addAuthHeader adds the Qdrant API key header if available
func (c *QdrantClient) addAuthHeader(req *http.Request) {
	if c.qdrantAPIKey != "" {
		req.Header.Set("api-key", c.qdrantAPIKey)
	}
}

// EnsureCollection ensures a Qdrant collection exists
func (c *QdrantClient) EnsureCollection(collectionName string, vectorSize int) error {
	// Check if collection exists
	checkURL := fmt.Sprintf("%s/collections/%s", c.baseURL, collectionName)
	req, err := http.NewRequest("GET", checkURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create check request: %w", err)
	}
	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
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
	req, err = http.NewRequest("PUT", createURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.addAuthHeader(req)

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
	c.addAuthHeader(req)

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

// SearchSimilar searches for similar points in Qdrant with optional vote-based ranking
// voteBoost is an optional parameter (default: 0.0 for no boosting)
// Formula: finalScore = semanticScore * (1 + voteBoost * normalizedVoteScore)
// normalizedVoteScore = voteScore / (1 + abs(voteScore)) to keep it in [-1, 1] range
func (c *QdrantClient) SearchSimilar(collectionName string, query string, limit int, voteBoost ...float64) ([]*QdrantQueryResult, error) {
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
	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		// Check for dimension mismatch error - return special error type for auto-recovery
		if dimErr := parseDimensionMismatchError(resp.StatusCode, bodyStr, collectionName); dimErr != nil {
			return nil, dimErr
		}

		return nil, fmt.Errorf("search failed: status %d, body: %s", resp.StatusCode, bodyStr)
	}

	// Parse response
	var searchResponse struct {
		Result []QdrantSearchResult `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	// Get vote boost factor (default 0.0 = no boosting)
	boostFactor := 0.0
	if len(voteBoost) > 0 {
		boostFactor = voteBoost[0]
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

		finalScore := result.Score

		// Apply vote boosting if enabled
		if boostFactor > 0 {
			voteScore := getIntFromPayload(result.Payload, "voteScore")
			// Normalize vote score to [-1, 1] range using voteScore / (1 + abs(voteScore))
			normalizedVote := 0.0
			if voteScore != 0 {
				normalizedVote = float64(voteScore) / (1.0 + float64(abs(voteScore)))
			}
			// Apply boost: finalScore = semanticScore * (1 + boostFactor * normalizedVote)
			finalScore = result.Score * (1.0 + boostFactor*normalizedVote)
		}

		results[i] = &QdrantQueryResult{
			Entry: entry,
			Score: finalScore,
		}
	}

	// Re-sort by final score if vote boosting was applied
	if boostFactor > 0 {
		sort.Slice(results, func(i, j int) bool {
			return results[i].Score > results[j].Score
		})
	}

	return results, nil
}

// SearchSimilarWithFilter searches for similar points with custom filter
// Filter can be used for taskId filtering or other metadata filtering
// Supports optional vote boosting like SearchSimilar
func (c *QdrantClient) SearchSimilarWithFilter(collectionName string, query string, limit int, filter map[string]interface{}, voteBoost ...float64) ([]*QdrantQueryResult, error) {
	// Generate query embedding using configured function
	queryVector, err := c.embeddingFunc(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Create search request with filter
	searchPayload := map[string]interface{}{
		"vector":       queryVector,
		"limit":        limit,
		"with_payload": true,
	}

	// Add filter if provided
	if filter != nil {
		searchPayload["filter"] = filter
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
	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		// Check for dimension mismatch error - return special error type for auto-recovery
		if dimErr := parseDimensionMismatchError(resp.StatusCode, bodyStr, collectionName); dimErr != nil {
			return nil, dimErr
		}

		return nil, fmt.Errorf("search failed: status %d, body: %s", resp.StatusCode, bodyStr)
	}

	// Parse response
	var searchResponse struct {
		Result []QdrantSearchResult `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	// Get vote boost factor (default 0.0 = no boosting)
	boostFactor := 0.0
	if len(voteBoost) > 0 {
		boostFactor = voteBoost[0]
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

		finalScore := result.Score

		// Apply vote boosting if enabled
		if boostFactor > 0 {
			voteScore := getIntFromPayload(result.Payload, "voteScore")
			// Normalize vote score to [-1, 1] range using voteScore / (1 + abs(voteScore))
			normalizedVote := 0.0
			if voteScore != 0 {
				normalizedVote = float64(voteScore) / (1.0 + float64(abs(voteScore)))
			}
			// Apply boost: finalScore = semanticScore * (1 + boostFactor * normalizedVote)
			finalScore = result.Score * (1.0 + boostFactor*normalizedVote)
		}

		results[i] = &QdrantQueryResult{
			Entry: entry,
			Score: finalScore,
		}
	}

	// Re-sort by final score if vote boosting was applied
	if boostFactor > 0 {
		sort.Slice(results, func(i, j int) bool {
			return results[i].Score > results[j].Score
		})
	}

	return results, nil
}

// SearchWithVoteFilter searches for similar points with vote score filtering
// Only returns entries with voteScore >= minVoteScore
// Supports optional vote boosting like SearchSimilar
func (c *QdrantClient) SearchWithVoteFilter(collectionName string, query string, limit int, minVoteScore int, voteBoost ...float64) ([]*QdrantQueryResult, error) {
	// Generate query embedding using configured function
	queryVector, err := c.embeddingFunc(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Create search request with vote score filter
	// Qdrant filter format: { "must": [{ "key": "voteScore", "range": { "gte": minVoteScore } }] }
	searchPayload := map[string]interface{}{
		"vector":       queryVector,
		"limit":        limit,
		"with_payload": true,
		"filter": map[string]interface{}{
			"must": []map[string]interface{}{
				{
					"key": "voteScore",
					"range": map[string]interface{}{
						"gte": minVoteScore,
					},
				},
			},
		},
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
	c.addAuthHeader(req)

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

	// Get vote boost factor (default 0.0 = no boosting)
	boostFactor := 0.0
	if len(voteBoost) > 0 {
		boostFactor = voteBoost[0]
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

		finalScore := result.Score

		// Apply vote boosting if enabled
		if boostFactor > 0 {
			voteScore := getIntFromPayload(result.Payload, "voteScore")
			// Normalize vote score to [-1, 1] range using voteScore / (1 + abs(voteScore))
			normalizedVote := 0.0
			if voteScore != 0 {
				normalizedVote = float64(voteScore) / (1.0 + float64(abs(voteScore)))
			}
			// Apply boost: finalScore = semanticScore * (1 + boostFactor * normalizedVote)
			finalScore = result.Score * (1.0 + boostFactor*normalizedVote)
		}

		results[i] = &QdrantQueryResult{
			Entry: entry,
			Score: finalScore,
		}
	}

	// Re-sort by final score if vote boosting was applied
	if boostFactor > 0 {
		sort.Slice(results, func(i, j int) bool {
			return results[i].Score > results[j].Score
		})
	}

	return results, nil
}

// DeletePoint deletes a single point from a collection
func (c *QdrantClient) DeletePoint(collectionName string, pointID string) error {
	requestBody := map[string]interface{}{
		"points": []string{pointID},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/delete?wait=true", c.baseURL, collectionName)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete point: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete point (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdatePointPayload updates only the payload of an existing point without re-embedding
// This is efficient for syncing metadata like vote scores without regenerating vectors
func (c *QdrantClient) UpdatePointPayload(collectionName string, pointID string, payload map[string]interface{}) error {
	requestBody := map[string]interface{}{
		"points": []string{pointID},
		"payload": payload,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal payload update request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/payload", c.baseURL, collectionName)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update payload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update payload (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
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

// getIntFromPayload safely extracts an int from payload
func getIntFromPayload(payload map[string]interface{}, key string) int {
	if val, ok := payload[key]; ok {
		// Handle different numeric types
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		case float32:
			return int(v)
		}
	}
	return 0
}

// abs returns the absolute value of an int
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
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

// GenerateCollectionName generates a unique, valid Qdrant collection name from a path
// Returns a name like "code_index_a1b2c3d4" (alphanumeric + underscore only)
func GenerateCollectionName(path string) string {
	// Create SHA256 hash of the path
	hash := sha256.Sum256([]byte(path))
	hashStr := hex.EncodeToString(hash[:])

	// Take first 8 characters of hash for uniqueness
	shortHash := hashStr[:8]

	// Construct valid collection name: prefix + hash
	collectionName := fmt.Sprintf("code_index_%s", shortHash)

	// Ensure it's valid Qdrant collection name (alphanumeric + underscore only)
	// This regex replaces any non-alphanumeric/underscore chars with underscore
	validNameRegex := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	collectionName = validNameRegex.ReplaceAllString(collectionName, "_")

	// Convert to lowercase for consistency
	collectionName = strings.ToLower(collectionName)

	return collectionName
}

// Ping checks Qdrant connectivity
func (c *QdrantClient) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/", c.baseURL), nil)
	if err != nil {
		return fmt.Errorf("failed to create ping request: %w", err)
	}

	// Add API key header if available
	if c.qdrantAPIKey != "" {
		req.Header.Set("api-key", c.qdrantAPIKey)
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

// Code Indexing specific methods (using float32 vectors)

const (
	DefaultCodeIndexCollection = "code_index"
	CodeIndexVectorSize        = 768 // TEI nomic-embed-text-v1.5 dimension (default - may be overridden)
)

var (
	// CodeIndexCollection is the collection name used for code indexing (configurable via QDRANT_CODE_COLLECTION env var)
	CodeIndexCollection = DefaultCodeIndexCollection
)

// InitCodeIndexCollection initializes the code index collection name from environment
func InitCodeIndexCollection() {
	if collectionName := os.Getenv("QDRANT_CODE_COLLECTION"); collectionName != "" {
		CodeIndexCollection = collectionName
	}
}

// DimensionMismatchError represents a dimension mismatch error from Qdrant
type DimensionMismatchError struct {
	ExpectedDim int
	GotDim      int
	Collection  string
	OriginalErr error
}

func (e *DimensionMismatchError) Error() string {
	return fmt.Sprintf("dimension mismatch in collection '%s': expected %d, got %d", e.Collection, e.ExpectedDim, e.GotDim)
}

// CodeIndexPoint represents a code indexing point with float32 vectors
type CodeIndexPoint struct {
	ID      string                 `json:"id"`
	Vector  []float32              `json:"vector"`
	Payload map[string]interface{} `json:"payload"`
}

// CodeIndexSearchResponse represents a search response for code indexing
type CodeIndexSearchResponse struct {
	Result []struct {
		ID      string                 `json:"id"`
		Score   float32                `json:"score"`
		Payload map[string]interface{} `json:"payload"`
		Vector  []float32              `json:"vector,omitempty"`
	} `json:"result"`
}

// parseDimensionMismatchError parses a Qdrant error response for dimension mismatch
func parseDimensionMismatchError(statusCode int, body string, collection string) error {
	if statusCode != http.StatusBadRequest {
		return nil
	}

	// Check for dimension mismatch in error message
	if !bytes.Contains([]byte(body), []byte("Vector dimension error")) &&
		!bytes.Contains([]byte(body), []byte("Wrong input")) {
		return nil
	}

	// Try to parse expected and got dimensions from error message
	// Example: "Vector dimension error: expected dim: 1024, got 768"
	var expectedDim, gotDim int
	_, err := fmt.Sscanf(body, `{"status":{"error":"Wrong input: Vector dimension error: expected dim: %d, got %d"}`, &expectedDim, &gotDim)
	if err != nil {
		// Alternative format
		_, err = fmt.Sscanf(body, `Vector dimension error: expected dim: %d, got %d`, &expectedDim, &gotDim)
	}

	if err == nil && expectedDim > 0 && gotDim > 0 {
		return &DimensionMismatchError{
			ExpectedDim: expectedDim,
			GotDim:      gotDim,
			Collection:  collection,
		}
	}

	// Generic dimension error without parsing
	return &DimensionMismatchError{
		ExpectedDim: 0,
		GotDim:      0,
		Collection:  collection,
		OriginalErr: fmt.Errorf("%s", body),
	}
}

// DeleteCollection deletes a Qdrant collection
func (c *QdrantClient) DeleteCollection(collectionName string) error {
	url := fmt.Sprintf("%s/collections/%s", c.baseURL, collectionName)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}
	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete collection (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// RecreateCollectionWithReindex deletes and recreates a Qdrant collection with new dimensions, then re-embeds all entries
// This is used for auto-recovery when dimension mismatch is detected (e.g., embedding provider changed)
// Returns the count of successfully reindexed entries
func (c *QdrantClient) RecreateCollectionWithReindex(collectionName string, entries []*KnowledgeEntry, dimensions int) (int, error) {
	// Step 1: Delete old collection
	if err := c.DeleteCollection(collectionName); err != nil {
		return 0, fmt.Errorf("failed to delete old collection: %w", err)
	}

	// Step 2: Recreate collection with new dimensions
	if err := c.EnsureCollection(collectionName, dimensions); err != nil {
		return 0, fmt.Errorf("failed to recreate collection with dimensions %d: %w", dimensions, err)
	}

	// Step 3: Re-embed and store each entry
	reindexedCount := 0
	for _, entry := range entries {
		// Re-embed the text with current embedding function
		if err := c.StorePoint(collectionName, entry.ID, entry.Text, entry.Metadata); err != nil {
			// Log error but continue with other entries
			fmt.Printf("Warning: failed to reindex entry %s: %v\n", entry.ID, err)
			continue
		}
		reindexedCount++
	}

	return reindexedCount, nil
}

// RecreateCodeIndexCollection deletes and recreates the code index collection with new dimensions
func (c *QdrantClient) RecreateCodeIndexCollection(vectorSize int) error {
	// Delete existing collection
	if err := c.DeleteCollection(CodeIndexCollection); err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	// Create collection with new dimensions
	collectionConfig := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     vectorSize,
			"distance": "Cosine",
		},
	}

	jsonBody, err := json.Marshal(collectionConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal collection config: %w", err)
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/collections/%s", c.baseURL, CodeIndexCollection), bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
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

// EnsureCodeIndexCollection creates the code index collection if it doesn't exist
// If expectedDimensions > 0, it also verifies the collection has matching dimensions
func (c *QdrantClient) EnsureCodeIndexCollection(expectedDimensions ...int) error {
	// Check if collection exists
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/collections/%s", c.baseURL, CodeIndexCollection), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to check collection: %w", err)
	}
	defer resp.Body.Close()

	// Collection exists - check dimensions if expectedDimensions provided
	if resp.StatusCode == http.StatusOK {
		// If expectedDimensions provided, verify vector size matches
		if len(expectedDimensions) > 0 && expectedDimensions[0] > 0 {
			// Get collection info to check vector size
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read collection info: %w", err)
			}

			var collectionInfo struct {
				Result struct {
					Config struct {
						Params struct {
							Vectors struct {
								Size int `json:"size"`
							} `json:"vectors"`
						} `json:"params"`
					} `json:"config"`
				} `json:"result"`
			}

			if err := json.Unmarshal(body, &collectionInfo); err != nil {
				return fmt.Errorf("failed to parse collection info: %w", err)
			}

			actualDim := collectionInfo.Result.Config.Params.Vectors.Size
			if actualDim != expectedDimensions[0] {
				return &DimensionMismatchError{
					ExpectedDim: actualDim,
					GotDim:      expectedDimensions[0],
					Collection:  CodeIndexCollection,
				}
			}
		}
		return nil
	}

	// Create collection
	collectionConfig := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     CodeIndexVectorSize,
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
	c.addAuthHeader(req)

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

// GetCollectionPointCount returns the number of points in a collection
func (c *QdrantClient) GetCollectionPointCount(collectionName string) (int, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/collections/%s", c.baseURL, collectionName), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to get collection info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to get collection info (status %d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	var collectionInfo struct {
		Result struct {
			PointsCount int `json:"points_count"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &collectionInfo); err != nil {
		return 0, fmt.Errorf("failed to parse collection info: %w", err)
	}

	return collectionInfo.Result.PointsCount, nil
}

// UpsertCodeIndexPoints upserts code indexing points into the specified collection
func (c *QdrantClient) UpsertCodeIndexPoints(collectionName string, points []CodeIndexPoint) error {
	requestBody := map[string]interface{}{
		"points": points,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal points: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points?wait=true", c.baseURL, collectionName)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upsert points: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		// Check for dimension mismatch
		if dimErr := parseDimensionMismatchError(resp.StatusCode, bodyStr, collectionName); dimErr != nil {
			return dimErr
		}

		return fmt.Errorf("failed to upsert points (status %d): %s", resp.StatusCode, bodyStr)
	}

	return nil
}

// SearchCodeIndex performs a vector similarity search for code in the specified collection
func (c *QdrantClient) SearchCodeIndex(collectionName string, vector []float32, limit int) (*CodeIndexSearchResponse, error) {
	return c.SearchCodeIndexWithFilter(collectionName, vector, limit, nil)
}

// SearchCodeIndexWithFilter performs a vector similarity search with optional filtering
// Supports both general filters and chunk ID filtering for structural search
func (c *QdrantClient) SearchCodeIndexWithFilter(collectionName string, vector []float32, limit int, filter map[string]interface{}, chunkIDs ...[]string) (*CodeIndexSearchResponse, error) {
	searchReq := map[string]interface{}{
		"vector":       vector,
		"limit":        limit,
		"with_payload": true,
		"with_vector":  false,
	}

	// Build composite filter if needed
	var compositeFilter map[string]interface{}

	// Handle chunk ID filtering (structural pre-filter from MongoDB)
	if len(chunkIDs) > 0 && len(chunkIDs[0]) > 0 {
		// Build Qdrant filter for chunk IDs: { "should": [{ "key": "id", "match": { "value": "id1" }}, ...] }
		shouldConditions := make([]map[string]interface{}, 0, len(chunkIDs[0]))
		for _, chunkID := range chunkIDs[0] {
			shouldConditions = append(shouldConditions, map[string]interface{}{
				"has_id": []string{chunkID},
			})
		}

		// If we have chunk IDs, use them as the primary filter
		compositeFilter = map[string]interface{}{
			"should": shouldConditions,
		}

		// If additional filters provided, combine with chunk ID filter using "must"
		if filter != nil {
			compositeFilter = map[string]interface{}{
				"must": []map[string]interface{}{
					{"should": shouldConditions},
					filter,
				},
			}
		}
	} else if filter != nil {
		// No chunk IDs, just use the provided filter
		compositeFilter = filter
	}

	// Add filter if we built one
	if compositeFilter != nil {
		searchReq["filter"] = compositeFilter
	}

	jsonBody, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/search", c.baseURL, collectionName)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		// Check for dimension mismatch
		if dimErr := parseDimensionMismatchError(resp.StatusCode, bodyStr, collectionName); dimErr != nil {
			return nil, dimErr
		}

		return nil, fmt.Errorf("search failed (status %d): %s", resp.StatusCode, bodyStr)
	}

	var searchResp CodeIndexSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	return &searchResp, nil
}

// DeleteCodeIndexByFilter deletes code index points matching a filter in the specified collection
func (c *QdrantClient) DeleteCodeIndexByFilter(collectionName string, filter map[string]interface{}) error {
	requestBody := map[string]interface{}{
		"filter": filter,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/delete?wait=true", c.baseURL, collectionName)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.addAuthHeader(req)

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

// UpsertCodeIndexPoint upserts a single code index point (helper for file watcher)
// Note: This uses the default CodeIndexCollection - use UpsertCodeIndexPoints for custom collections
func (c *QdrantClient) UpsertCodeIndexPoint(id string, vector []float32, payload map[string]interface{}) error {
	point := CodeIndexPoint{
		ID:      id,
		Vector:  vector,
		Payload: payload,
	}
	return c.UpsertCodeIndexPoints(CodeIndexCollection, []CodeIndexPoint{point})
}

// DeleteCodeIndexPoint deletes a single code index point (helper for file watcher)
func (c *QdrantClient) DeleteCodeIndexPoint(pointID string) error {
	requestBody := map[string]interface{}{
		"points": []string{pointID},
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
	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete point: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete point (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// EnsureCollectionForPath ensures a Qdrant collection exists for a specific path
// Checks code_index_map for existing mapping, or creates new collection and mapping
// Returns the collection name to use for this path
func (c *QdrantClient) EnsureCollectionForPath(path string, codeIndexStorage *CodeIndexStorage) (string, error) {
	// Check if mapping already exists
	mapping, err := codeIndexStorage.GetPathMapping(path)
	if err != nil {
		return "", fmt.Errorf("failed to check path mapping: %w", err)
	}

	if mapping != nil {
		// Mapping exists, return the collection name
		return mapping.QdrantCollection, nil
	}

	// No mapping exists, create new collection
	collectionName := GenerateCollectionName(path)

	// Create Qdrant collection with 768 dimensions (default for code indexing)
	collectionConfig := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     c.vectorDimension,
			"distance": "Cosine",
		},
	}

	jsonBody, err := json.Marshal(collectionConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal collection config: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s", c.baseURL, collectionName)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create collection: %w", err)
	}
	defer resp.Body.Close()

	// Collection already exists is OK (status 200), or newly created (201)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create collection (status %d): %s", resp.StatusCode, string(body))
	}

	// Save mapping to MongoDB
	if err := codeIndexStorage.AddPathMapping(path, collectionName); err != nil {
		return "", fmt.Errorf("failed to save path mapping: %w", err)
	}

	return collectionName, nil
}
