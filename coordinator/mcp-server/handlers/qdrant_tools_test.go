package handlers

import (
	"encoding/json"
	"testing"

	"hyperion-coordinator-mcp/storage"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockQdrantClient implements a mock Qdrant client for testing
type MockQdrantClient struct {
	collections map[string]bool
	points      map[string]map[string]*storage.QdrantQueryResult
	shouldError bool
	errorMsg    string
}

func NewMockQdrantClient() *MockQdrantClient {
	return &MockQdrantClient{
		collections: make(map[string]bool),
		points:      make(map[string]map[string]*storage.QdrantQueryResult),
	}
}

func (m *MockQdrantClient) EnsureCollection(collectionName string, vectorSize int) error {
	if m.shouldError {
		return &mockError{msg: m.errorMsg}
	}
	m.collections[collectionName] = true
	if m.points[collectionName] == nil {
		m.points[collectionName] = make(map[string]*storage.QdrantQueryResult)
	}
	return nil
}

func (m *MockQdrantClient) StorePoint(collectionName string, id string, text string, metadata map[string]interface{}) error {
	if m.shouldError {
		return &mockError{msg: m.errorMsg}
	}
	if m.points[collectionName] == nil {
		m.points[collectionName] = make(map[string]*storage.QdrantQueryResult)
	}
	m.points[collectionName][id] = &storage.QdrantQueryResult{
		Entry: &storage.KnowledgeEntry{
			ID:         id,
			Collection: collectionName,
			Text:       text,
			Metadata:   metadata,
		},
		Score: 1.0,
	}
	return nil
}

func (m *MockQdrantClient) SearchSimilar(collectionName string, query string, limit int) ([]*storage.QdrantQueryResult, error) {
	if m.shouldError {
		return nil, &mockError{msg: m.errorMsg}
	}

	// Return points from the collection
	results := make([]*storage.QdrantQueryResult, 0)
	if points, ok := m.points[collectionName]; ok {
		for _, point := range points {
			results = append(results, point)
			if len(results) >= limit {
				break
			}
		}
	}

	return results, nil
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

// Test qdrant_find with valid parameters
func TestQdrantFind_ValidParams(t *testing.T) {
	mockClient := NewMockQdrantClient()
	handler := NewQdrantToolHandler(mockClient)

	// Add test data
	mockClient.EnsureCollection("test-collection", 1536)
	mockClient.StorePoint("test-collection", "test-id-1", "This is test content for searching", map[string]interface{}{
		"author": "test-user",
		"tags":   []string{"testing", "qdrant"},
	})

	// Test qdrant_find
	args := map[string]interface{}{
		"collectionName": "test-collection",
		"query":          "test content",
		"limit":          float64(5),
	}

	result, data, err := handler.handleQdrantFind(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.NotNil(t, data)

	// Verify result contains expected text
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "Found 1 results")
	assert.Contains(t, textContent.Text, "This is test content")
}

// Test qdrant_find with missing collectionName
func TestQdrantFind_MissingCollectionName(t *testing.T) {
	mockClient := NewMockQdrantClient()
	handler := NewQdrantToolHandler(mockClient)

	args := map[string]interface{}{
		"query": "test query",
	}

	result, _, err := handler.handleQdrantFind(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "collectionName parameter is required")
}

// Test qdrant_find with missing query
func TestQdrantFind_MissingQuery(t *testing.T) {
	mockClient := NewMockQdrantClient()
	handler := NewQdrantToolHandler(mockClient)

	args := map[string]interface{}{
		"collectionName": "test-collection",
	}

	result, _, err := handler.handleQdrantFind(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "query parameter is required")
}

// Test qdrant_find with limit > 20 (should use max 20)
func TestQdrantFind_LimitExceedsMax(t *testing.T) {
	mockClient := NewMockQdrantClient()
	handler := NewQdrantToolHandler(mockClient)

	// Add multiple test points
	mockClient.EnsureCollection("test-collection", 1536)
	for i := 0; i < 25; i++ {
		mockClient.StorePoint("test-collection", string(rune(i)), "test content", nil)
	}

	args := map[string]interface{}{
		"collectionName": "test-collection",
		"query":          "test",
		"limit":          float64(30), // Request 30, should be capped at 20
	}

	result, data, err := handler.handleQdrantFind(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)

	// Verify data is limited
	results, ok := data.([]*storage.QdrantQueryResult)
	require.True(t, ok)
	assert.LessOrEqual(t, len(results), 20, "Results should be capped at 20")
}

// Test qdrant_find with no results
func TestQdrantFind_NoResults(t *testing.T) {
	mockClient := NewMockQdrantClient()
	handler := NewQdrantToolHandler(mockClient)

	// Create empty collection
	mockClient.EnsureCollection("empty-collection", 1536)

	args := map[string]interface{}{
		"collectionName": "empty-collection",
		"query":          "test",
		"limit":          float64(5),
	}

	result, _, err := handler.handleQdrantFind(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "No results found")
}

// Test qdrant_store with valid parameters
func TestQdrantStore_ValidParams(t *testing.T) {
	mockClient := NewMockQdrantClient()
	handler := NewQdrantToolHandler(mockClient)

	args := map[string]interface{}{
		"collectionName": "test-collection",
		"information":    "This is important knowledge to store",
		"metadata": map[string]interface{}{
			"author": "test-user",
			"tags":   []interface{}{"testing", "knowledge"},
		},
	}

	result, data, err := handler.handleQdrantStore(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.NotNil(t, data)

	// Verify result contains expected information
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "✓ Knowledge stored in Qdrant")
	assert.Contains(t, textContent.Text, "Collection: test-collection")
	assert.Contains(t, textContent.Text, "Vector dimensions: 1536")

	// Verify data structure
	dataMap, ok := data.(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, dataMap, "id")
	assert.Equal(t, "test-collection", dataMap["collection"])
}

// Test qdrant_store with missing collectionName
func TestQdrantStore_MissingCollectionName(t *testing.T) {
	mockClient := NewMockQdrantClient()
	handler := NewQdrantToolHandler(mockClient)

	args := map[string]interface{}{
		"information": "test information",
	}

	result, _, err := handler.handleQdrantStore(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "collectionName parameter is required")
}

// Test qdrant_store with empty information
func TestQdrantStore_EmptyInformation(t *testing.T) {
	mockClient := NewMockQdrantClient()
	handler := NewQdrantToolHandler(mockClient)

	args := map[string]interface{}{
		"collectionName": "test-collection",
		"information":    "",
	}

	result, _, err := handler.handleQdrantStore(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "information parameter is required")
}

// Test qdrant_store without metadata (optional parameter)
func TestQdrantStore_NoMetadata(t *testing.T) {
	mockClient := NewMockQdrantClient()
	handler := NewQdrantToolHandler(mockClient)

	args := map[string]interface{}{
		"collectionName": "test-collection",
		"information":    "Knowledge without metadata",
	}

	result, data, err := handler.handleQdrantStore(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.NotNil(t, data)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "✓ Knowledge stored in Qdrant")
}

// Test qdrant_store with storage failure
func TestQdrantStore_StorageFailure(t *testing.T) {
	mockClient := NewMockQdrantClient()
	mockClient.shouldError = true
	mockClient.errorMsg = "storage failure"
	handler := NewQdrantToolHandler(mockClient)

	args := map[string]interface{}{
		"collectionName": "test-collection",
		"information":    "test information",
	}

	result, _, err := handler.handleQdrantStore(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "failed to ensure collection exists")
}

// Test qdrant_find with collection creation failure
func TestQdrantFind_CollectionFailure(t *testing.T) {
	mockClient := NewMockQdrantClient()
	mockClient.shouldError = true
	mockClient.errorMsg = "collection creation failed"
	handler := NewQdrantToolHandler(mockClient)

	args := map[string]interface{}{
		"collectionName": "test-collection",
		"query":          "test",
	}

	result, _, err := handler.handleQdrantFind(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "failed to ensure collection exists")
}

// Test RegisterQdrantTools
func TestRegisterQdrantTools(t *testing.T) {
	mockClient := NewMockQdrantClient()
	handler := NewQdrantToolHandler(mockClient)

	impl := &mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}

	opts := &mcp.ServerOptions{
		HasTools: true,
	}

	server := mcp.NewServer(impl, opts)

	err := handler.RegisterQdrantTools(server)
	require.NoError(t, err)

	// Verify tools were registered
	// Note: The MCP SDK doesn't expose a way to list registered tools,
	// but we can verify no error occurred during registration
}

// Test extractArguments helper function
func TestExtractArguments(t *testing.T) {
	t.Run("valid JSON arguments", func(t *testing.T) {
		args := map[string]interface{}{
			"key1": "value1",
			"key2": float64(123),
		}
		argsJSON, _ := json.Marshal(args)

		req := &mcp.CallToolRequest{}
		req.Params.Arguments = argsJSON

		result, err := extractArguments(req)
		require.NoError(t, err)
		assert.Equal(t, "value1", result["key1"])
		assert.Equal(t, float64(123), result["key2"])
	})

	t.Run("nil arguments", func(t *testing.T) {
		req := &mcp.CallToolRequest{}
		req.Params.Arguments = nil

		result, err := extractArguments(req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := &mcp.CallToolRequest{}
		req.Params.Arguments = json.RawMessage(`{invalid json}`)

		_, err := extractArguments(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "arguments must be a valid JSON object")
	})
}

// Test response format verification
func TestQdrantFind_ResponseFormat(t *testing.T) {
	mockClient := NewMockQdrantClient()
	handler := NewQdrantToolHandler(mockClient)

	// Create test data with long text
	longText := "This is a very long piece of text that should be truncated in the response because it exceeds 200 characters. " +
		"This additional text ensures we go well over the 200 character limit so we can verify truncation is working correctly in our implementation."

	mockClient.EnsureCollection("test-collection", 1536)
	mockClient.StorePoint("test-collection", "test-id", longText, map[string]interface{}{
		"key": "value",
	})

	args := map[string]interface{}{
		"collectionName": "test-collection",
		"query":          "test",
		"limit":          float64(5),
	}

	result, _, err := handler.handleQdrantFind(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)

	// Verify format: "Result N (Score: 0.XX)"
	assert.Contains(t, textContent.Text, "Result 1 (Score:")
	assert.Contains(t, textContent.Text, "Text:")
	assert.Contains(t, textContent.Text, "Metadata:")
	assert.Contains(t, textContent.Text, "---")

	// Verify truncation (first 200 chars + "...")
	assert.Contains(t, textContent.Text, "...")
}
