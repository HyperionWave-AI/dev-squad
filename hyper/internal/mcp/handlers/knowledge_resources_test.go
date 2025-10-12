package handlers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"hyper/internal/mcp/storage"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockKnowledgeStorage for testing
type MockKnowledgeStorage struct {
	entries     []*storage.KnowledgeEntry
	collections []string
}

func (m *MockKnowledgeStorage) Upsert(collection, text string, metadata map[string]interface{}) (*storage.KnowledgeEntry, error) {
	entry := &storage.KnowledgeEntry{
		ID:         "test-id",
		Collection: collection,
		Text:       text,
		Metadata:   metadata,
		CreatedAt:  time.Now().UTC(),
	}
	m.entries = append(m.entries, entry)
	return entry, nil
}

func (m *MockKnowledgeStorage) Query(collection, query string, limit int) ([]*storage.QueryResult, error) {
	results := make([]*storage.QueryResult, 0)
	for _, entry := range m.entries {
		if entry.Collection == collection {
			results = append(results, &storage.QueryResult{
				Entry: entry,
				Score: 0.8,
			})
		}
		if limit > 0 && len(results) >= limit {
			break
		}
	}
	return results, nil
}

func (m *MockKnowledgeStorage) ListCollections() []string {
	return m.collections
}

func (m *MockKnowledgeStorage) GetPopularCollections(limit int) ([]*storage.CollectionStats, error) {
	// Return mock collection stats for testing
	result := make([]*storage.CollectionStats, 0)
	for i, col := range m.collections {
		if limit > 0 && i >= limit {
			break
		}
		result = append(result, &storage.CollectionStats{
			Collection: col,
			Count:      len(m.entries),
		})
	}
	return result, nil
}

func TestKnowledgeResourceHandler_CollectionsResource(t *testing.T) {
	mockStorage := &MockKnowledgeStorage{
		collections: []string{"technical-knowledge", "code-patterns", "team-coordination"},
	}

	handler := NewKnowledgeResourceHandler(mockStorage)
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0.0"}, &mcp.ServerOptions{HasResources: true})

	err := handler.RegisterKnowledgeResources(server)
	require.NoError(t, err)

	// Test collections resource
	result, err := handler.handleCollectionsResource(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "hyperion://knowledge/collections",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Contents, 1)
	assert.Equal(t, "hyperion://knowledge/collections", result.Contents[0].URI)
	assert.Equal(t, "application/json", result.Contents[0].MIMEType)

	// Parse and verify JSON response
	var response map[string]interface{}
	err = json.Unmarshal([]byte(result.Contents[0].Text), &response)
	require.NoError(t, err)

	// Verify structure
	assert.Contains(t, response, "collections")
	assert.Contains(t, response, "totalDefined")
	assert.Contains(t, response, "totalWithData")

	// Verify collections are present
	collections := response["collections"].([]interface{})
	assert.Greater(t, len(collections), 0)

	// Check first collection has expected fields
	firstCollection := collections[0].(map[string]interface{})
	assert.Contains(t, firstCollection, "name")
	assert.Contains(t, firstCollection, "purpose")
	assert.Contains(t, firstCollection, "exampleQuery")
	assert.Contains(t, firstCollection, "useCases")
	assert.Contains(t, firstCollection, "category")
	assert.Contains(t, firstCollection, "hasData")
}

func TestKnowledgeResourceHandler_RecentLearningsResource(t *testing.T) {
	now := time.Now().UTC()

	mockStorage := &MockKnowledgeStorage{
		collections: []string{"technical-knowledge", "team-coordination"},
		entries: []*storage.KnowledgeEntry{
			{
				ID:         "entry-1",
				Collection: "technical-knowledge",
				Text:       "JWT authentication pattern implementation",
				Metadata: map[string]interface{}{
					"title":     "JWT Auth Pattern",
					"agentName": "backend-dev",
				},
				CreatedAt: now.Add(-1 * time.Hour), // 1 hour ago
			},
			{
				ID:         "entry-2",
				Collection: "team-coordination",
				Text:       "API contract change for task endpoints",
				Metadata: map[string]interface{}{
					"title":     "Task API Contract",
					"agentName": "api-dev",
				},
				CreatedAt: now.Add(-12 * time.Hour), // 12 hours ago
			},
			{
				ID:         "entry-3",
				Collection: "technical-knowledge",
				Text:       "Old entry that should not appear",
				Metadata: map[string]interface{}{
					"title": "Old Pattern",
				},
				CreatedAt: now.Add(-48 * time.Hour), // 48 hours ago (too old)
			},
		},
	}

	handler := NewKnowledgeResourceHandler(mockStorage)
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0.0"}, &mcp.ServerOptions{HasResources: true})

	err := handler.RegisterKnowledgeResources(server)
	require.NoError(t, err)

	// Test recent learnings resource
	result, err := handler.handleRecentLearningsResource(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "hyperion://knowledge/recent-learnings",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Contents, 1)
	assert.Equal(t, "hyperion://knowledge/recent-learnings", result.Contents[0].URI)
	assert.Equal(t, "application/json", result.Contents[0].MIMEType)

	// Parse and verify JSON response
	var response map[string]interface{}
	err = json.Unmarshal([]byte(result.Contents[0].Text), &response)
	require.NoError(t, err)

	// Verify structure
	assert.Contains(t, response, "timeRange")
	assert.Contains(t, response, "totalEntries")
	assert.Contains(t, response, "byCollection")
	assert.Contains(t, response, "allEntries")
	assert.Contains(t, response, "collectionsWithActivity")

	// Verify time range
	timeRange := response["timeRange"].(map[string]interface{})
	assert.Contains(t, timeRange, "start")
	assert.Contains(t, timeRange, "end")

	// Verify only recent entries are included (2 entries within 24h)
	allEntries := response["allEntries"].([]interface{})
	assert.Equal(t, 2, len(allEntries))

	// Verify entries have expected fields
	if len(allEntries) > 0 {
		firstEntry := allEntries[0].(map[string]interface{})
		assert.Contains(t, firstEntry, "id")
		assert.Contains(t, firstEntry, "collection")
		assert.Contains(t, firstEntry, "topic")
		assert.Contains(t, firstEntry, "text")
		assert.Contains(t, firstEntry, "metadata")
		assert.Contains(t, firstEntry, "createdAt")
	}

	// Verify grouping by collection
	byCollection := response["byCollection"].(map[string]interface{})
	assert.Contains(t, byCollection, "technical-knowledge")
	assert.Contains(t, byCollection, "team-coordination")
}

func TestKnowledgeResourceHandler_CollectionCategories(t *testing.T) {
	mockStorage := &MockKnowledgeStorage{
		collections: []string{},
	}

	handler := NewKnowledgeResourceHandler(mockStorage)

	// Test collections resource
	result, err := handler.handleCollectionsResource(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "hyperion://knowledge/collections",
		},
	})

	require.NoError(t, err)

	// Parse JSON response
	var response map[string]interface{}
	err = json.Unmarshal([]byte(result.Contents[0].Text), &response)
	require.NoError(t, err)

	collections := response["collections"].([]interface{})

	// Verify all expected categories are present
	categories := make(map[string]bool)
	for _, c := range collections {
		collection := c.(map[string]interface{})
		category := collection["category"].(string)
		categories[category] = true
	}

	assert.True(t, categories["Task"], "Should have Task category")
	assert.True(t, categories["Tech"], "Should have Tech category")
	assert.True(t, categories["UI"], "Should have UI category")
	assert.True(t, categories["Ops"], "Should have Ops category")

	// Verify specific collections exist
	collectionNames := make(map[string]bool)
	for _, c := range collections {
		collection := c.(map[string]interface{})
		name := collection["name"].(string)
		collectionNames[name] = true
	}

	expectedCollections := []string{
		"technical-knowledge",
		"code-patterns",
		"team-coordination",
		"ui-component-patterns",
		"mcp-operations",
	}

	for _, expected := range expectedCollections {
		assert.True(t, collectionNames[expected], "Should have collection: %s", expected)
	}
}
