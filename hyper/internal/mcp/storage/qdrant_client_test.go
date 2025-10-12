package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockEmbeddingFuncFactory creates a mock embedding function with specified dimensions
func mockEmbeddingFuncFactory(dimensions int) func(string) ([]float64, error) {
	return func(text string) ([]float64, error) {
		// Return fixed-size vector with simple deterministic values
		vector := make([]float64, dimensions)
		for i := 0; i < dimensions; i++ {
			if i < len(text) {
				vector[i] = float64(text[i]) / 255.0
			} else {
				vector[i] = 0.5
			}
		}
		return vector, nil
	}
}

func TestNewQdrantClient(t *testing.T) {
	client := NewQdrantClient("http://localhost:6333", "test_knowledge")

	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:6333", client.baseURL)
	assert.Equal(t, 768, client.vectorDimension) // TEI nomic-embed-text-v1.5 dimension
	assert.NotNil(t, client.httpClient)
	assert.NotNil(t, client.embeddingFunc)
}

func TestNewQdrantClientWithEmbedding(t *testing.T) {
	mockFunc := mockEmbeddingFuncFactory(128)
	client := NewQdrantClientWithEmbedding("http://localhost:6333", mockFunc, 128)

	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:6333", client.baseURL)
	assert.Equal(t, 128, client.vectorDimension)
	assert.NotNil(t, client.embeddingFunc)

	// Test embedding function works
	vec, err := client.embeddingFunc("test")
	assert.NoError(t, err)
	assert.Equal(t, 128, len(vec))
}

func TestGenerateSimpleEmbedding(t *testing.T) {
	// Test simple hash-based embedding
	vec := generateSimpleEmbedding("hello world", 1536)

	assert.Equal(t, 1536, len(vec))

	// All values should be in range [0, 1]
	for _, v := range vec {
		assert.GreaterOrEqual(t, v, 0.0)
		assert.LessOrEqual(t, v, 1.0)
	}

	// Same input should produce same output (deterministic)
	vec2 := generateSimpleEmbedding("hello world", 1536)
	assert.Equal(t, vec, vec2)

	// Different input should produce different output
	vec3 := generateSimpleEmbedding("different text", 1536)
	assert.NotEqual(t, vec, vec3)
}

func TestQdrantPoint(t *testing.T) {
	point := QdrantPoint{
		ID:     "test-id",
		Vector: []float64{0.1, 0.2, 0.3},
		Payload: map[string]interface{}{
			"text": "test text",
			"key":  "value",
		},
	}

	assert.Equal(t, "test-id", point.ID)
	assert.Equal(t, 3, len(point.Vector))
	assert.Equal(t, "test text", point.Payload["text"])
	assert.Equal(t, "value", point.Payload["key"])
}

func TestQdrantSearchResult(t *testing.T) {
	result := QdrantSearchResult{
		ID:    "result-id",
		Score: 0.95,
		Payload: map[string]interface{}{
			"text": "matched text",
		},
	}

	assert.Equal(t, "result-id", result.ID)
	assert.Equal(t, 0.95, result.Score)
	assert.Equal(t, "matched text", result.Payload["text"])
}

func TestQdrantQueryResult(t *testing.T) {
	entry := &KnowledgeEntry{
		ID:         "entry-1",
		Collection: "test-collection",
		Text:       "knowledge text",
		Metadata:   map[string]interface{}{"tag": "important"},
	}

	queryResult := &QdrantQueryResult{
		Entry: entry,
		Score: 0.88,
	}

	assert.Equal(t, "entry-1", queryResult.Entry.ID)
	assert.Equal(t, "test-collection", queryResult.Entry.Collection)
	assert.Equal(t, 0.88, queryResult.Score)
}

func TestGetStringFromPayload(t *testing.T) {
	payload := map[string]interface{}{
		"text":   "hello",
		"number": 123,
		"bool":   true,
	}

	// Valid string
	assert.Equal(t, "hello", getStringFromPayload(payload, "text"))

	// Non-string value returns empty
	assert.Equal(t, "", getStringFromPayload(payload, "number"))
	assert.Equal(t, "", getStringFromPayload(payload, "bool"))

	// Missing key returns empty
	assert.Equal(t, "", getStringFromPayload(payload, "missing"))
}

func TestParseTimeFromPayload(t *testing.T) {
	payload := map[string]interface{}{
		"createdAt": "2025-01-15T10:30:00Z",
		"invalid":   "not-a-time",
		"number":    123,
	}

	// Valid time
	parsedTime := parseTimeFromPayload(payload, "createdAt")
	assert.Equal(t, 2025, parsedTime.Year())
	assert.Equal(t, 1, int(parsedTime.Month()))
	assert.Equal(t, 15, parsedTime.Day())

	// Invalid time returns current time (non-zero)
	invalidTime := parseTimeFromPayload(payload, "invalid")
	assert.False(t, invalidTime.IsZero())

	// Missing key returns current time
	missingTime := parseTimeFromPayload(payload, "missing")
	assert.False(t, missingTime.IsZero())
}

func TestGenerateID(t *testing.T) {
	id1 := GenerateID()
	id2 := GenerateID()

	// IDs should be non-empty
	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)

	// IDs should be unique
	assert.NotEqual(t, id1, id2)

	// IDs should be valid UUIDs (36 chars with hyphens)
	assert.Equal(t, 36, len(id1))
	assert.Contains(t, id1, "-")
}

func TestPing(t *testing.T) {
	// Note: This test requires a running Qdrant instance
	// In CI/CD, this should be skipped or use testcontainers
	t.Skip("Requires running Qdrant instance")

	client := NewQdrantClient("http://localhost:6333", "test_knowledge")
	ctx := context.Background()

	err := client.Ping(ctx)
	assert.NoError(t, err)
}

func TestEmbeddingFuncFallback(t *testing.T) {
	// Uses TEI embeddings (not OpenAI fallback)
	client := NewQdrantClient("http://localhost:6333", "test_knowledge")

	vec, err := client.embeddingFunc("test text")
	// TEI service may not be available in test env, skip assertion if error
	if err != nil {
		t.Skip("TEI service not available in test environment")
	}
	assert.Equal(t, 768, len(vec))

	// Should be deterministic
	vec2, err := client.embeddingFunc("test text")
	assert.NoError(t, err)
	assert.Equal(t, vec, vec2)
}

// Integration tests (require running Qdrant)

func TestEnsureCollection_Integration(t *testing.T) {
	t.Skip("Integration test - requires running Qdrant instance")

	mockFunc := mockEmbeddingFuncFactory(128)
	client := NewQdrantClientWithEmbedding("http://localhost:6333", mockFunc, 128)

	// Create collection
	err := client.EnsureCollection("test_collection", 128)
	assert.NoError(t, err)

	// Creating same collection should be idempotent
	err = client.EnsureCollection("test_collection", 128)
	assert.NoError(t, err)
}

func TestStorePoint_Integration(t *testing.T) {
	t.Skip("Integration test - requires running Qdrant instance")

	mockFunc := mockEmbeddingFuncFactory(128)
	client := NewQdrantClientWithEmbedding("http://localhost:6333", mockFunc, 128)

	// Ensure collection exists
	err := client.EnsureCollection("test_collection", 128)
	assert.NoError(t, err)

	// Store point
	err = client.StorePoint("test_collection", "point-1", "test knowledge", map[string]interface{}{
		"category": "testing",
		"priority": "high",
	})
	assert.NoError(t, err)
}

func TestSearchSimilar_Integration(t *testing.T) {
	t.Skip("Integration test - requires running Qdrant instance")

	mockFunc := mockEmbeddingFuncFactory(128)
	client := NewQdrantClientWithEmbedding("http://localhost:6333", mockFunc, 128)

	// Setup: Create collection and add points
	err := client.EnsureCollection("test_collection", 128)
	assert.NoError(t, err)

	err = client.StorePoint("test_collection", "point-1", "kubernetes deployment patterns", map[string]interface{}{
		"topic": "infrastructure",
	})
	assert.NoError(t, err)

	err = client.StorePoint("test_collection", "point-2", "react component testing", map[string]interface{}{
		"topic": "frontend",
	})
	assert.NoError(t, err)

	// Search
	results, err := client.SearchSimilar("test_collection", "kubernetes", 5)
	assert.NoError(t, err)
	assert.NotEmpty(t, results)

	// First result should have highest score
	assert.GreaterOrEqual(t, results[0].Score, 0.0)
	assert.LessOrEqual(t, results[0].Score, 1.0)

	// Results should contain our data
	assert.Contains(t, results[0].Entry.Text, "kubernetes")
}
