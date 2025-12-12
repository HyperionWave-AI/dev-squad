package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"hyper/internal/mcp/storage"

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
	pingError   error
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

func (m *MockQdrantClient) SearchSimilar(collectionName string, query string, limit int, voteBoost ...float64) ([]*storage.QdrantQueryResult, error) {
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

func (m *MockQdrantClient) DeletePoint(collectionName string, pointID string) error {
	if m.shouldError {
		return &mockError{msg: m.errorMsg}
	}
	if points, ok := m.points[collectionName]; ok {
		delete(points, pointID)
	}
	return nil
}

func (m *MockQdrantClient) DeleteCollection(collectionName string) error {
	if m.shouldError {
		return &mockError{msg: m.errorMsg}
	}
	delete(m.collections, collectionName)
	delete(m.points, collectionName)
	return nil
}

func (m *MockQdrantClient) UpdatePointPayload(collectionName string, pointID string, payload map[string]interface{}) error {
	return nil
}

func (m *MockQdrantClient) RecreateCollectionWithReindex(collectionName string, entries []*storage.KnowledgeEntry, dimensions int) (int, error) {
	return 0, nil
}

func (m *MockQdrantClient) GetDimensions() int {
	return 384
}

func (m *MockQdrantClient) SearchSimilarWithFilter(collectionName string, query string, limit int, filter map[string]interface{}, voteBoost ...float64) ([]*storage.QdrantQueryResult, error) {
	return nil, nil
}

func (m *MockQdrantClient) SearchWithVoteFilter(collectionName string, query string, limit int, minVoteScore int, voteBoost ...float64) ([]*storage.QdrantQueryResult, error) {
	return nil, nil
}

func (m *MockQdrantClient) Ping(ctx context.Context) error {
	if m.pingError != nil {
		return m.pingError
	}
	return nil
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

// Test knowledge_find with valid parameters
func TestKnowledgeFind_ValidParams(t *testing.T) {
	mockClient := NewMockQdrantClient()
	handler := NewQdrantToolHandler(mockClient)
	mockStorage := NewMockVotingStorage()
	handler.SetKnowledgeStorage(mockStorage)

	// Add test data via storage (handler uses storage, not client directly)
	mockStorage.Upsert("test-collection", "This is test content for searching", map[string]interface{}{
		"author": "test-user",
		"tags":   []string{"testing", "qdrant"},
	}, nil)

	// Test knowledge_find
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
	assert.Contains(t, textContent.Text, "Found 1 total results")
	assert.Contains(t, textContent.Text, "test content")
}

// Test knowledge_find with missing collectionName
func TestKnowledgeFind_MissingCollectionName(t *testing.T) {
	mockClient := NewMockQdrantClient()
	handler := NewQdrantToolHandler(mockClient)
	mockStorage := NewMockVotingStorage()
	handler.SetKnowledgeStorage(mockStorage)

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

// Test knowledge_find with missing query
func TestKnowledgeFind_MissingQuery(t *testing.T) {
	mockClient := NewMockQdrantClient()
	handler := NewQdrantToolHandler(mockClient)
	mockStorage := NewMockVotingStorage()
	handler.SetKnowledgeStorage(mockStorage)

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

// Test knowledge_find with limit > 20 (should use max 20)
func TestKnowledgeFind_LimitExceedsMax(t *testing.T) {
	mockClient := NewMockQdrantClient()
	handler := NewQdrantToolHandler(mockClient)
	mockStorage := NewMockVotingStorage()
	handler.SetKnowledgeStorage(mockStorage)

	// Add multiple test points via storage
	for i := 0; i < 25; i++ {
		mockStorage.Upsert("test-collection", fmt.Sprintf("test content %d", i), nil, nil)
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
	results, ok := data.([]*storage.QueryResult)
	require.True(t, ok)
	assert.LessOrEqual(t, len(results), 20, "Results should be capped at 20")
}

// Test knowledge_find with no results
func TestKnowledgeFind_NoResults(t *testing.T) {
	mockClient := NewMockQdrantClient()
	handler := NewQdrantToolHandler(mockClient)
	mockStorage := NewMockVotingStorage()
	handler.SetKnowledgeStorage(mockStorage)

	// No data added - collection is empty

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
	// The handler returns "No results at offset X" message when there are no results
	assert.Contains(t, textContent.Text, "No results")
}

// Test knowledge_store with valid parameters
func TestKnowledgeStore_ValidParams(t *testing.T) {
	mockClient := NewMockQdrantClient()
	handler := NewQdrantToolHandler(mockClient)
	mockStorage := NewMockVotingStorage()
	handler.SetKnowledgeStorage(mockStorage)

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
	assert.Contains(t, textContent.Text, "✓ Knowledge stored successfully")
	assert.Contains(t, textContent.Text, "Collection: test-collection")

	// Verify data structure
	dataMap, ok := data.(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, dataMap, "id")
	assert.Equal(t, "test-collection", dataMap["collection"])
}

// Test knowledge_store with missing collectionName
func TestKnowledgeStore_MissingCollectionName(t *testing.T) {
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

// Test knowledge_store with empty information
func TestKnowledgeStore_EmptyInformation(t *testing.T) {
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

// Test knowledge_store without metadata (optional parameter)
func TestKnowledgeStore_NoMetadata(t *testing.T) {
	mockClient := NewMockQdrantClient()
	handler := NewQdrantToolHandler(mockClient)
	mockStorage := NewMockVotingStorage()
	handler.SetKnowledgeStorage(mockStorage)

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

// Test knowledge_store with storage failure
func TestKnowledgeStore_StorageFailure(t *testing.T) {
	mockClient := NewMockQdrantClient()
	mockClient.shouldError = true
	mockClient.errorMsg = "storage failure"
	handler := NewQdrantToolHandler(mockClient)
	mockStorage := NewMockVotingStorage()
	handler.SetKnowledgeStorage(mockStorage)

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

// Test knowledge_find with collection creation failure
func TestKnowledgeFind_CollectionFailure(t *testing.T) {
	mockClient := NewMockQdrantClient()
	mockClient.shouldError = true
	mockClient.errorMsg = "collection creation failed"
	handler := NewQdrantToolHandler(mockClient)
	mockStorage := NewMockVotingStorage()
	handler.SetKnowledgeStorage(mockStorage)

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

		// Create the request properly - CallToolRequest is ServerRequest[*CallToolParamsRaw]
		params := &mcp.CallToolParamsRaw{
			Name:      "test_tool",
			Arguments: argsJSON,
		}
		req := mcp.ServerRequest[*mcp.CallToolParamsRaw]{
			Params: params,
		}

		result, err := extractArguments(&req)
		require.NoError(t, err)
		assert.Equal(t, "value1", result["key1"])
		assert.Equal(t, float64(123), result["key2"])
	})

	t.Run("nil arguments", func(t *testing.T) {
		params := &mcp.CallToolParamsRaw{
			Name:      "test_tool",
			Arguments: nil,
		}
		req := mcp.ServerRequest[*mcp.CallToolParamsRaw]{
			Params: params,
		}

		result, err := extractArguments(&req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		params := &mcp.CallToolParamsRaw{
			Name:      "test_tool",
			Arguments: json.RawMessage(`{invalid json}`),
		}
		req := mcp.ServerRequest[*mcp.CallToolParamsRaw]{
			Params: params,
		}

		_, err := extractArguments(&req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "arguments must be a valid JSON object")
	})
}

// Test response format verification
func TestKnowledgeFind_ResponseFormat(t *testing.T) {
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

// MockVotingStorage implements a mock storage for testing voting functionality
type MockVotingStorage struct {
	votes      map[string]*storage.Vote // key: entryID+userID
	voteCounts map[string]*storage.VoteSummary
	entries    map[string]*storage.KnowledgeEntry
}

func NewMockVotingStorage() *MockVotingStorage {
	return &MockVotingStorage{
		votes:      make(map[string]*storage.Vote),
		voteCounts: make(map[string]*storage.VoteSummary),
		entries:    make(map[string]*storage.KnowledgeEntry),
	}
}

func (m *MockVotingStorage) VoteOnEntry(entryID, userID, vote, reason string) (*storage.Vote, error) {
	// Check if entry exists
	if _, exists := m.entries[entryID]; !exists {
		return nil, &mockError{msg: "entry not found"}
	}

	key := entryID + userID
	voteRecord := &storage.Vote{
		EntryID: entryID,
		UserID:  userID,
		Vote:    vote,
		Reason:  reason,
	}
	m.votes[key] = voteRecord

	// Update vote counts
	if m.voteCounts[entryID] == nil {
		m.voteCounts[entryID] = &storage.VoteSummary{}
	}
	summary := m.voteCounts[entryID]

	// Recalculate counts
	upvotes := 0
	downvotes := 0
	for k, v := range m.votes {
		if v.EntryID == entryID {
			if v.Vote == "+" {
				upvotes++
			} else if v.Vote == "-" {
				downvotes++
			}
		}
		if k == key {
			summary.UserVote = vote
		}
	}
	summary.Upvotes = upvotes
	summary.Downvotes = downvotes
	summary.NetScore = upvotes - downvotes

	return voteRecord, nil
}

func (m *MockVotingStorage) GetEntryVotes(entryID, userID string) (*storage.VoteSummary, error) {
	// Check if entry exists
	if _, exists := m.entries[entryID]; !exists {
		return nil, &mockError{msg: "entry not found"}
	}

	summary := m.voteCounts[entryID]
	if summary == nil {
		summary = &storage.VoteSummary{}
	}

	// Get user's vote
	key := entryID + userID
	if vote, exists := m.votes[key]; exists {
		summary.UserVote = vote.Vote
	}

	return summary, nil
}

// Implement remaining KnowledgeStorage methods as stubs
func (m *MockVotingStorage) Upsert(collection, text string, metadata map[string]interface{}, taskId *string) (*storage.KnowledgeEntry, error) {
	// Create a knowledge entry
	id := fmt.Sprintf("%s-%d", collection, len(m.entries))
	entry := &storage.KnowledgeEntry{
		ID:         id,
		Collection: collection,
		Text:       text,
		Metadata:   metadata,
	}
	m.entries[id] = entry
	return entry, nil
}

func (m *MockVotingStorage) UpdateEntry(id, text string, metadata map[string]interface{}) (*storage.KnowledgeEntry, error) {
	entry, exists := m.entries[id]
	if !exists {
		return nil, &mockError{msg: "entry not found"}
	}
	entry.Text = text
	entry.Metadata = metadata
	return entry, nil
}

func (m *MockVotingStorage) DeleteEntry(id string) error {
	delete(m.entries, id)
	return nil
}

func (m *MockVotingStorage) GetEntryByID(id string) (*storage.KnowledgeEntry, error) {
	entry, exists := m.entries[id]
	if !exists {
		return nil, &mockError{msg: "entry not found"}
	}
	return entry, nil
}

func (m *MockVotingStorage) GetEntriesByCollection(collectionName string) ([]*storage.KnowledgeEntry, error) {
	var results []*storage.KnowledgeEntry
	for _, entry := range m.entries {
		if entry.Collection == collectionName {
			results = append(results, entry)
		}
	}
	return results, nil
}

func (m *MockVotingStorage) Query(collection, query string, limit int, taskId *string, voteBoost ...float64) ([]*storage.QueryResult, error) {
	// Simple mock implementation: return all entries from the collection that contain the query string
	var results []*storage.QueryResult
	for _, entry := range m.entries {
		if entry.Collection == collection {
			// Simple contains check for testing
			if query == "" || containsText(entry.Text, query) {
				results = append(results, &storage.QueryResult{
					Entry: entry,
					Score: 0.9, // Mock score
				})
			}
		}
	}

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// Helper function for simple string contains check
func containsText(text, substr string) bool {
	if substr == "" {
		return true
	}
	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (m *MockVotingStorage) ListCollections() []string {
	return []string{}
}

func (m *MockVotingStorage) CreateCollection(name, category, description string, tags []string) (*storage.Collection, error) {
	return nil, nil
}

func (m *MockVotingStorage) DeleteCollection(id string) (string, int64, error) {
	return "", 0, nil
}

func (m *MockVotingStorage) GetPopularCollections(limit int) ([]*storage.CollectionStats, error) {
	return nil, nil
}

func (m *MockVotingStorage) GetCollectionStatsWithMetadata() ([]*storage.CollectionWithMetadata, error) {
	return nil, nil
}

func (m *MockVotingStorage) ListKnowledge(collection string, limit int) ([]*storage.KnowledgeEntry, error) {
	return nil, nil
}

func (m *MockVotingStorage) UpdateCollectionMetadata(collectionName, description string, tags []string, category string) (*storage.CollectionMetadata, error) {
	return nil, nil
}

func (m *MockVotingStorage) RenameCollection(oldName, newName string) (int64, error) {
	return 0, nil
}

func (m *MockVotingStorage) BatchSyncVotesToQdrant(collectionName string) (int, error) {
	return 0, nil
}

// Test knowledge_vote_on_entry with successful upvote
func TestKnowledgeVoteOnEntry_SuccessfulUpvote(t *testing.T) {
	mockStorage := NewMockVotingStorage()
	handler := &QdrantToolHandler{
		knowledgeStorage: mockStorage,
	}

	// Add test entry
	mockStorage.entries["test-entry-1"] = &storage.KnowledgeEntry{
		ID:         "test-entry-1",
		Collection: "test-collection",
		Text:       "Test knowledge entry",
	}

	args := map[string]interface{}{
		"entryId": "test-entry-1",
		"vote":    "+",
		"reason":  "This is very helpful information",
	}

	result, data, err := handler.handleKnowledgeVoteOnEntry(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.NotNil(t, data)

	// Verify result contains expected information
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "✓ Vote recorded successfully")
	assert.Contains(t, textContent.Text, "upvote")
	assert.Contains(t, textContent.Text, "This is very helpful information")
	assert.Contains(t, textContent.Text, "Upvotes: 1")
	assert.Contains(t, textContent.Text, "Net Score: 1")

	// Verify data structure
	dataMap, ok := data.(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, dataMap, "vote")
	assert.Contains(t, dataMap, "summary")
}

// Test knowledge_vote_on_entry with successful downvote
func TestKnowledgeVoteOnEntry_SuccessfulDownvote(t *testing.T) {
	mockStorage := NewMockVotingStorage()
	handler := &QdrantToolHandler{
		knowledgeStorage: mockStorage,
	}

	// Add test entry
	mockStorage.entries["test-entry-1"] = &storage.KnowledgeEntry{
		ID:         "test-entry-1",
		Collection: "test-collection",
		Text:       "Test knowledge entry",
	}

	args := map[string]interface{}{
		"entryId": "test-entry-1",
		"vote":    "-",
		"reason":  "Information is outdated",
	}

	result, data, err := handler.handleKnowledgeVoteOnEntry(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.NotNil(t, data)

	// Verify result contains downvote information
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "✓ Vote recorded successfully")
	assert.Contains(t, textContent.Text, "downvote")
	assert.Contains(t, textContent.Text, "Information is outdated")
	assert.Contains(t, textContent.Text, "Downvotes: 1")
	assert.Contains(t, textContent.Text, "Net Score: -1")
}

// Test knowledge_vote_on_entry with invalid vote type
func TestKnowledgeVoteOnEntry_InvalidVoteType(t *testing.T) {
	mockStorage := NewMockVotingStorage()
	handler := &QdrantToolHandler{
		knowledgeStorage: mockStorage,
	}

	args := map[string]interface{}{
		"entryId": "test-entry-1",
		"vote":    "invalid",
		"reason":  "test reason",
	}

	result, _, err := handler.handleKnowledgeVoteOnEntry(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "vote must be '+' or '-'")
	assert.Contains(t, textContent.Text, "invalid")
}

// Test knowledge_vote_on_entry with missing entryId
func TestKnowledgeVoteOnEntry_MissingEntryId(t *testing.T) {
	mockStorage := NewMockVotingStorage()
	handler := &QdrantToolHandler{
		knowledgeStorage: mockStorage,
	}

	args := map[string]interface{}{
		"vote":   "+",
		"reason": "test reason",
	}

	result, _, err := handler.handleKnowledgeVoteOnEntry(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "entryId parameter is required")
}

// Test knowledge_vote_on_entry with missing vote
func TestKnowledgeVoteOnEntry_MissingVote(t *testing.T) {
	mockStorage := NewMockVotingStorage()
	handler := &QdrantToolHandler{
		knowledgeStorage: mockStorage,
	}

	args := map[string]interface{}{
		"entryId": "test-entry-1",
		"reason":  "test reason",
	}

	result, _, err := handler.handleKnowledgeVoteOnEntry(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "vote parameter is required")
}

// Test knowledge_vote_on_entry with missing reason
func TestKnowledgeVoteOnEntry_MissingReason(t *testing.T) {
	mockStorage := NewMockVotingStorage()
	handler := &QdrantToolHandler{
		knowledgeStorage: mockStorage,
	}

	args := map[string]interface{}{
		"entryId": "test-entry-1",
		"vote":    "+",
	}

	result, _, err := handler.handleKnowledgeVoteOnEntry(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "reason parameter is required")
}

// Test knowledge_vote_on_entry with entry not found
func TestKnowledgeVoteOnEntry_EntryNotFound(t *testing.T) {
	mockStorage := NewMockVotingStorage()
	handler := &QdrantToolHandler{
		knowledgeStorage: mockStorage,
	}

	args := map[string]interface{}{
		"entryId": "nonexistent-entry",
		"vote":    "+",
		"reason":  "test reason",
	}

	result, _, err := handler.handleKnowledgeVoteOnEntry(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "Entry with ID 'nonexistent-entry' not found")
	assert.Contains(t, textContent.Text, "Use knowledge_find or knowledge_get_by_id")
}

// Test knowledge_vote_on_entry without knowledge storage
func TestKnowledgeVoteOnEntry_NoStorage(t *testing.T) {
	handler := &QdrantToolHandler{
		knowledgeStorage: nil,
	}

	args := map[string]interface{}{
		"entryId": "test-entry-1",
		"vote":    "+",
		"reason":  "test reason",
	}

	result, _, err := handler.handleKnowledgeVoteOnEntry(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "Knowledge storage not initialized")
}

// Test knowledge_get_entry_votes with successful retrieval
func TestKnowledgeGetEntryVotes_Success(t *testing.T) {
	mockStorage := NewMockVotingStorage()
	handler := &QdrantToolHandler{
		knowledgeStorage: mockStorage,
	}

	// Add test entry and votes
	mockStorage.entries["test-entry-1"] = &storage.KnowledgeEntry{
		ID:         "test-entry-1",
		Collection: "test-collection",
		Text:       "Test knowledge entry",
	}

	// Add multiple votes
	mockStorage.VoteOnEntry("test-entry-1", "user1", "+", "helpful")
	mockStorage.VoteOnEntry("test-entry-1", "user2", "+", "useful")
	mockStorage.VoteOnEntry("test-entry-1", "user3", "-", "not accurate")

	args := map[string]interface{}{
		"entryId": "test-entry-1",
	}

	result, data, err := handler.handleKnowledgeGetEntryVotes(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.NotNil(t, data)

	// Verify result contains vote summary
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "Vote Summary for Entry: test-entry-1")
	assert.Contains(t, textContent.Text, "Upvotes: 2")
	assert.Contains(t, textContent.Text, "Downvotes: 1")
	assert.Contains(t, textContent.Text, "Net Score: 1")

	// Verify data structure
	summary, ok := data.(*storage.VoteSummary)
	require.True(t, ok)
	assert.Equal(t, 2, summary.Upvotes)
	assert.Equal(t, 1, summary.Downvotes)
	assert.Equal(t, 1, summary.NetScore)
}

// Test knowledge_get_entry_votes with no votes
func TestKnowledgeGetEntryVotes_NoVotes(t *testing.T) {
	mockStorage := NewMockVotingStorage()
	handler := &QdrantToolHandler{
		knowledgeStorage: mockStorage,
	}

	// Add test entry but no votes
	mockStorage.entries["test-entry-1"] = &storage.KnowledgeEntry{
		ID:         "test-entry-1",
		Collection: "test-collection",
		Text:       "Test knowledge entry",
	}

	args := map[string]interface{}{
		"entryId": "test-entry-1",
	}

	result, data, err := handler.handleKnowledgeGetEntryVotes(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.NotNil(t, data)

	// Verify result shows zero votes
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "Upvotes: 0")
	assert.Contains(t, textContent.Text, "Downvotes: 0")
	assert.Contains(t, textContent.Text, "Net Score: 0")
	assert.Contains(t, textContent.Text, "Your vote: None")
}

// Test knowledge_get_entry_votes with missing entryId
func TestKnowledgeGetEntryVotes_MissingEntryId(t *testing.T) {
	mockStorage := NewMockVotingStorage()
	handler := &QdrantToolHandler{
		knowledgeStorage: mockStorage,
	}

	args := map[string]interface{}{}

	result, _, err := handler.handleKnowledgeGetEntryVotes(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "entryId parameter is required")
}

// Test knowledge_get_entry_votes with entry not found
func TestKnowledgeGetEntryVotes_EntryNotFound(t *testing.T) {
	mockStorage := NewMockVotingStorage()
	handler := &QdrantToolHandler{
		knowledgeStorage: mockStorage,
	}

	args := map[string]interface{}{
		"entryId": "nonexistent-entry",
	}

	result, _, err := handler.handleKnowledgeGetEntryVotes(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "Entry with ID 'nonexistent-entry' not found")
	assert.Contains(t, textContent.Text, "Use knowledge_find or knowledge_get_by_id")
}

// Test knowledge_get_entry_votes without knowledge storage
func TestKnowledgeGetEntryVotes_NoStorage(t *testing.T) {
	handler := &QdrantToolHandler{
		knowledgeStorage: nil,
	}

	args := map[string]interface{}{
		"entryId": "test-entry-1",
	}

	result, _, err := handler.handleKnowledgeGetEntryVotes(args)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "Knowledge storage not initialized")
}
