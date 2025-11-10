package handlers

import (
	"context"
	"encoding/json"
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

func (m *MockQdrantClient) DeletePoint(collectionName string, pointID string) error {
	if m.shouldError {
		return &mockError{msg: m.errorMsg}
	}
	if points, ok := m.points[collectionName]; ok {
		delete(points, pointID)
	}
	return nil
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

	// Add test data
	mockClient.EnsureCollection("test-collection", 1536)
	mockClient.StorePoint("test-collection", "test-id-1", "This is test content for searching", map[string]interface{}{
		"author": "test-user",
		"tags":   []string{"testing", "qdrant"},
	})

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
	assert.Contains(t, textContent.Text, "Found 1 results")
	assert.Contains(t, textContent.Text, "This is test content")
}

// Test knowledge_find with missing collectionName
func TestKnowledgeFind_MissingCollectionName(t *testing.T) {
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

// Test knowledge_find with missing query
func TestKnowledgeFind_MissingQuery(t *testing.T) {
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

// Test knowledge_find with limit > 20 (should use max 20)
func TestKnowledgeFind_LimitExceedsMax(t *testing.T) {
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

// Test knowledge_find with no results
func TestKnowledgeFind_NoResults(t *testing.T) {
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

// Test knowledge_store with valid parameters
func TestKnowledgeStore_ValidParams(t *testing.T) {
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
