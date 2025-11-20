package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// MockQdrantClient is a mock implementation of QdrantClientInterface for testing
type MockQdrantClient struct {
	mock.Mock
}

func (m *MockQdrantClient) EnsureCollection(collectionName string, vectorSize int) error {
	args := m.Called(collectionName, vectorSize)
	return args.Error(0)
}

func (m *MockQdrantClient) StorePoint(collectionName string, id string, text string, metadata map[string]interface{}) error {
	args := m.Called(collectionName, id, text, metadata)
	return args.Error(0)
}

func (m *MockQdrantClient) SearchSimilar(collectionName string, query string, limit int, voteBoost ...float64) ([]*QdrantQueryResult, error) {
	args := m.Called(collectionName, query, limit, voteBoost)
	if args.Get(0) != nil {
		return args.Get(0).([]*QdrantQueryResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockQdrantClient) SearchSimilarWithFilter(collectionName string, query string, limit int, filter map[string]interface{}, voteBoost ...float64) ([]*QdrantQueryResult, error) {
	args := m.Called(collectionName, query, limit, filter, voteBoost)
	if args.Get(0) != nil {
		return args.Get(0).([]*QdrantQueryResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockQdrantClient) SearchWithVoteFilter(collectionName string, query string, limit int, minVoteScore int, voteBoost ...float64) ([]*QdrantQueryResult, error) {
	args := m.Called(collectionName, query, limit, minVoteScore, voteBoost)
	if args.Get(0) != nil {
		return args.Get(0).([]*QdrantQueryResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockQdrantClient) DeletePoint(collectionName string, pointID string) error {
	args := m.Called(collectionName, pointID)
	return args.Error(0)
}

func (m *MockQdrantClient) DeleteCollection(collectionName string) error {
	args := m.Called(collectionName)
	return args.Error(0)
}

func (m *MockQdrantClient) UpdatePointPayload(collectionName string, pointID string, payload map[string]interface{}) error {
	args := m.Called(collectionName, pointID, payload)
	return args.Error(0)
}

func (m *MockQdrantClient) RecreateCollectionWithReindex(collectionName string, entries []*KnowledgeEntry, dimensions int) (int, error) {
	args := m.Called(collectionName, entries, dimensions)
	return args.Int(0), args.Error(1)
}

func (m *MockQdrantClient) GetDimensions() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockQdrantClient) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// setupTestStorage creates a test MongoDB instance with MongoKnowledgeStorage
func setupTestStorage(t *testing.T) (*MongoKnowledgeStorage, *mongo.Database, *MockQdrantClient, func()) {
	// Use in-memory MongoDB for testing
	client, err := mongo.Connect(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create MongoDB client: %v", err)
	}

	db := client.Database("test_knowledge_" + uuid.New().String())
	logger := zap.NewNop()

	// Create mock Qdrant client
	mockQdrant := new(MockQdrantClient)
	mockQdrant.On("GetDimensions").Return(768)
	mockQdrant.On("EnsureCollection", mock.Anything, mock.Anything).Return(nil)

	storage, err := NewMongoKnowledgeStorage(db, mockQdrant, logger)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	cleanup := func() {
		db.Drop(context.Background())
		client.Disconnect(context.Background())
	}

	return storage, db, mockQdrant, cleanup
}

// TestKnowledgeStorage_Upsert tests creating and updating knowledge entries
func TestKnowledgeStorage_Upsert(t *testing.T) {
	storage, db, mockQdrant, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	collectionName := "test_collection"

	// Create collection first
	_, err := storage.CreateCollection(collectionName, "test", "Test collection", []string{"test"})
	assert.NoError(t, err)

	tests := []struct {
		name        string
		collection  string
		text        string
		metadata    map[string]interface{}
		taskId      *string
		setupMock   func(*MockQdrantClient)
		wantErr     bool
		errContains string
		validate    func(*testing.T, *KnowledgeEntry)
	}{
		{
			name:       "Create new entry with all fields",
			collection: collectionName,
			text:       "Test knowledge entry",
			metadata: map[string]interface{}{
				"author": "test_user",
				"tags":   []string{"test", "knowledge"},
			},
			taskId: stringPtr("task-123"),
			setupMock: func(m *MockQdrantClient) {
				m.On("StorePoint", mock.Anything, mock.Anything, "Test knowledge entry", mock.Anything).Return(nil).Once()
			},
			wantErr: false,
			validate: func(t *testing.T, entry *KnowledgeEntry) {
				assert.NotEmpty(t, entry.ID)
				assert.Equal(t, collectionName, entry.Collection)
				assert.Equal(t, "Test knowledge entry", entry.Text)
				assert.Equal(t, "test_user", entry.Metadata["author"])
				assert.Equal(t, "task-123", entry.TaskId)
				assert.NotZero(t, entry.CreatedAt)
			},
		},
		{
			name:       "Create entry without metadata",
			collection: collectionName,
			text:       "Entry without metadata",
			metadata:   nil,
			taskId:     nil,
			setupMock: func(m *MockQdrantClient) {
				m.On("StorePoint", mock.Anything, mock.Anything, "Entry without metadata", mock.Anything).Return(nil).Once()
			},
			wantErr: false,
			validate: func(t *testing.T, entry *KnowledgeEntry) {
				assert.NotEmpty(t, entry.ID)
				assert.Nil(t, entry.Metadata)
				assert.Empty(t, entry.TaskId)
			},
		},
		{
			name:       "Create entry without taskId",
			collection: collectionName,
			text:       "Entry without taskId",
			metadata:   map[string]interface{}{"key": "value"},
			taskId:     nil,
			setupMock: func(m *MockQdrantClient) {
				m.On("StorePoint", mock.Anything, mock.Anything, "Entry without taskId", mock.Anything).Return(nil).Once()
			},
			wantErr: false,
			validate: func(t *testing.T, entry *KnowledgeEntry) {
				assert.Empty(t, entry.TaskId)
			},
		},
		{
			name:       "Auto-create collection if not exists",
			collection: "new_collection",
			text:       "Entry in new collection",
			metadata:   nil,
			taskId:     nil,
			setupMock: func(m *MockQdrantClient) {
				m.On("StorePoint", mock.Anything, mock.Anything, "Entry in new collection", mock.Anything).Return(nil).Once()
			},
			wantErr: false,
			validate: func(t *testing.T, entry *KnowledgeEntry) {
				assert.Equal(t, "new_collection", entry.Collection)
				// Verify collection was created
				col, err := storage.GetCollection("new_collection")
				assert.NoError(t, err)
				assert.NotNil(t, col)
				assert.Equal(t, "new_collection", col.Name)
			},
		},
		{
			name:       "Create entry with empty text - allowed by MongoDB",
			collection: collectionName,
			text:       "",
			metadata:   nil,
			taskId:     nil,
			setupMock: func(m *MockQdrantClient) {
				m.On("StorePoint", mock.Anything, mock.Anything, "", mock.Anything).Return(nil).Once()
			},
			wantErr: false,
			validate: func(t *testing.T, entry *KnowledgeEntry) {
				assert.Equal(t, "", entry.Text)
			},
		},
		{
			name:       "Auto-create collection with empty name",
			collection: "",
			text:       "Some text",
			metadata:   nil,
			taskId:     nil,
			setupMock: func(m *MockQdrantClient) {
				m.On("StorePoint", mock.Anything, mock.Anything, "Some text", mock.Anything).Return(nil).Once()
			},
			wantErr: false,
			validate: func(t *testing.T, entry *KnowledgeEntry) {
				assert.Equal(t, "", entry.Collection)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mockQdrant)

			entry, err := storage.Upsert(tt.collection, tt.text, tt.metadata, tt.taskId)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, entry)
				if tt.validate != nil {
					tt.validate(t, entry)
				}

				// Verify entry in MongoDB
				var storedEntry KnowledgeEntry
				err = db.Collection("knowledge_entries").FindOne(ctx, bson.M{"entryId": entry.ID}).Decode(&storedEntry)
				assert.NoError(t, err)
				assert.Equal(t, entry.ID, storedEntry.ID)
			}
		})
	}
}

// TestKnowledgeStorage_GetEntryByID tests retrieving knowledge entries by ID
func TestKnowledgeStorage_GetEntryByID(t *testing.T) {
	storage, _, mockQdrant, cleanup := setupTestStorage(t)
	defer cleanup()

	collectionName := "test_collection"
	_, err := storage.CreateCollection(collectionName, "test", "Test collection", []string{})
	assert.NoError(t, err)

	// Create test entry
	mockQdrant.On("StorePoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	metadata := map[string]interface{}{"key": "value"}
	entry, err := storage.Upsert(collectionName, "Test entry", metadata, nil)
	assert.NoError(t, err)

	tests := []struct {
		name        string
		entryID     string
		wantErr     bool
		errContains string
		validate    func(*testing.T, *KnowledgeEntry)
	}{
		{
			name:    "Retrieve existing entry",
			entryID: entry.ID,
			wantErr: false,
			validate: func(t *testing.T, retrieved *KnowledgeEntry) {
				assert.Equal(t, entry.ID, retrieved.ID)
				assert.Equal(t, entry.Text, retrieved.Text)
				assert.Equal(t, entry.Collection, retrieved.Collection)
				assert.Equal(t, "value", retrieved.Metadata["key"])
			},
		},
		{
			name:        "Non-existent entry returns error",
			entryID:     "non-existent-id",
			wantErr:     true,
			errContains: "not found",
		},
		{
			name:    "Entry with all fields populated",
			entryID: entry.ID,
			wantErr: false,
			validate: func(t *testing.T, retrieved *KnowledgeEntry) {
				assert.NotEmpty(t, retrieved.ID)
				assert.NotEmpty(t, retrieved.Text)
				assert.NotEmpty(t, retrieved.Collection)
				assert.NotNil(t, retrieved.Metadata)
				assert.NotZero(t, retrieved.CreatedAt)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrieved, err := storage.GetEntryByID(tt.entryID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, retrieved)
				if tt.validate != nil {
					tt.validate(t, retrieved)
				}
			}
		})
	}
}

// TestKnowledgeStorage_UpdateEntry tests updating existing knowledge entries
func TestKnowledgeStorage_UpdateEntry(t *testing.T) {
	storage, _, mockQdrant, cleanup := setupTestStorage(t)
	defer cleanup()

	collectionName := "test_collection"
	_, err := storage.CreateCollection(collectionName, "test", "Test collection", []string{})
	assert.NoError(t, err)

	// Create initial entry
	mockQdrant.On("StorePoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockQdrant.On("DeletePoint", mock.Anything, mock.Anything).Return(nil)
	originalMetadata := map[string]interface{}{"version": "1"}
	entry, err := storage.Upsert(collectionName, "Original text", originalMetadata, nil)
	assert.NoError(t, err)

	tests := []struct {
		name        string
		entryID     string
		newText     string
		newMetadata map[string]interface{}
		wantErr     bool
		errContains string
		validate    func(*testing.T, *KnowledgeEntry)
	}{
		{
			name:        "Update text only",
			entryID:     entry.ID,
			newText:     "Updated text",
			newMetadata: originalMetadata,
			wantErr:     false,
			validate: func(t *testing.T, updated *KnowledgeEntry) {
				assert.Equal(t, "Updated text", updated.Text)
				assert.Equal(t, originalMetadata["version"], updated.Metadata["version"])
			},
		},
		{
			name:        "Update metadata only",
			entryID:     entry.ID,
			newText:     "Updated text", // Using previous update
			newMetadata: map[string]interface{}{"version": "2", "author": "test"},
			wantErr:     false,
			validate: func(t *testing.T, updated *KnowledgeEntry) {
				assert.Equal(t, "2", updated.Metadata["version"])
				assert.Equal(t, "test", updated.Metadata["author"])
			},
		},
		{
			name:        "Update both text and metadata",
			entryID:     entry.ID,
			newText:     "Completely new text",
			newMetadata: map[string]interface{}{"version": "3"},
			wantErr:     false,
			validate: func(t *testing.T, updated *KnowledgeEntry) {
				assert.Equal(t, "Completely new text", updated.Text)
				assert.Equal(t, "3", updated.Metadata["version"])
			},
		},
		{
			name:        "Non-existent entry returns error",
			entryID:     "non-existent-id",
			newText:     "New text",
			newMetadata: map[string]interface{}{},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name:        "Preserve unchanged fields",
			entryID:     entry.ID,
			newText:     "Final text",
			newMetadata: map[string]interface{}{"version": "4"},
			wantErr:     false,
			validate: func(t *testing.T, updated *KnowledgeEntry) {
				// Collection should remain unchanged
				assert.Equal(t, collectionName, updated.Collection)
				// CreatedAt should remain unchanged (MongoDB truncates to milliseconds)
				assert.WithinDuration(t, entry.CreatedAt, updated.CreatedAt, time.Millisecond)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated, err := storage.UpdateEntry(tt.entryID, tt.newText, tt.newMetadata)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, updated)
				if tt.validate != nil {
					tt.validate(t, updated)
				}
			}
		})
	}
}

// TestKnowledgeStorage_DeleteEntry tests deleting knowledge entries
func TestKnowledgeStorage_DeleteEntry(t *testing.T) {
	storage, db, mockQdrant, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	collectionName := "test_collection"
	_, err := storage.CreateCollection(collectionName, "test", "Test collection", []string{})
	assert.NoError(t, err)

	tests := []struct {
		name        string
		setupEntry  func() string // Returns entry ID
		entryID     string
		setupMock   func(*MockQdrantClient, string)
		wantErr     bool
		errContains string
		validate    func(*testing.T, string)
	}{
		{
			name: "Successful deletion from MongoDB",
			setupEntry: func() string {
				mockQdrant.On("StorePoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
				entry, _ := storage.Upsert(collectionName, "Entry to delete", nil, nil)
				return entry.ID
			},
			setupMock: func(m *MockQdrantClient, entryID string) {
				m.On("DeletePoint", mock.Anything, entryID).Return(nil).Once()
			},
			wantErr: false,
			validate: func(t *testing.T, entryID string) {
				// Verify entry is deleted from MongoDB
				var entry KnowledgeEntry
				err := db.Collection("knowledge_entries").FindOne(ctx, bson.M{"entryId": entryID}).Decode(&entry)
				assert.Error(t, err)
				assert.Equal(t, mongo.ErrNoDocuments, err)
			},
		},
		{
			name: "Successful deletion from Qdrant",
			setupEntry: func() string {
				mockQdrant.On("StorePoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
				entry, _ := storage.Upsert(collectionName, "Another entry", nil, nil)
				return entry.ID
			},
			setupMock: func(m *MockQdrantClient, entryID string) {
				m.On("DeletePoint", mock.Anything, entryID).Return(nil).Once()
			},
			wantErr: false,
			validate: func(t *testing.T, entryID string) {
				// Mock should have been called
				mockQdrant.AssertCalled(t, "DeletePoint", mock.Anything, entryID)
			},
		},
		{
			name:       "Deleting non-existent entry",
			setupEntry: func() string { return "" },
			entryID:    "non-existent-id",
			setupMock:  func(m *MockQdrantClient, entryID string) {},
			wantErr:    true,
			errContains: "not found",
		},
		{
			name: "Cascade effects - votes remain for audit",
			setupEntry: func() string {
				mockQdrant.On("StorePoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
				mockQdrant.On("UpdatePointPayload", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				entry, _ := storage.Upsert(collectionName, "Entry with votes", nil, nil)
				// Add a vote
				storage.VoteOnEntry(entry.ID, "user-1", "+", "helpful")
				return entry.ID
			},
			setupMock: func(m *MockQdrantClient, entryID string) {
				m.On("DeletePoint", mock.Anything, entryID).Return(nil).Once()
			},
			wantErr: false,
			validate: func(t *testing.T, entryID string) {
				// Verify votes still exist in database (for audit trail)
				count, err := db.Collection("knowledge_votes").CountDocuments(ctx, bson.M{"entryId": entryID})
				assert.NoError(t, err)
				assert.Equal(t, int64(1), count, "Votes should remain for audit trail")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var entryID string
			if tt.setupEntry != nil {
				entryID = tt.setupEntry()
			}
			if tt.entryID != "" {
				entryID = tt.entryID
			}

			if tt.setupMock != nil {
				tt.setupMock(mockQdrant, entryID)
			}

			err := storage.DeleteEntry(entryID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, entryID)
				}
			}
		})
	}
}

// TestKnowledgeStorage_ListKnowledge tests browsing knowledge entries by collection
func TestKnowledgeStorage_ListKnowledge(t *testing.T) {
	storage, _, mockQdrant, cleanup := setupTestStorage(t)
	defer cleanup()

	collectionName := "test_collection"
	_, err := storage.CreateCollection(collectionName, "test", "Test collection", []string{})
	assert.NoError(t, err)

	// Create test entries
	mockQdrant.On("StorePoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	var entries []*KnowledgeEntry
	for i := 0; i < 5; i++ {
		entry, err := storage.Upsert(collectionName, "Entry "+string(rune('A'+i)), nil, nil)
		assert.NoError(t, err)
		entries = append(entries, entry)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	tests := []struct {
		name       string
		collection string
		limit      int
		wantErr    bool
		validate   func(*testing.T, []*KnowledgeEntry)
	}{
		{
			name:       "Browse by collection with limit",
			collection: collectionName,
			limit:      3,
			wantErr:    false,
			validate: func(t *testing.T, results []*KnowledgeEntry) {
				assert.Len(t, results, 3)
				// Should be ordered by creation date (newest first)
				assert.True(t, results[0].CreatedAt.After(results[1].CreatedAt) || results[0].CreatedAt.Equal(results[1].CreatedAt))
			},
		},
		{
			name:       "Empty collection returns empty array",
			collection: "empty_collection",
			limit:      10,
			wantErr:    true, // Collection doesn't exist
		},
		{
			name:       "Ordering recent first",
			collection: collectionName,
			limit:      10,
			wantErr:    false,
			validate: func(t *testing.T, results []*KnowledgeEntry) {
				assert.Len(t, results, 5)
				// Verify descending order by creation time
				for i := 0; i < len(results)-1; i++ {
					assert.True(t, results[i].CreatedAt.After(results[i+1].CreatedAt) || results[i].CreatedAt.Equal(results[i+1].CreatedAt))
				}
			},
		},
		{
			name:       "Limit enforcement",
			collection: collectionName,
			limit:      2,
			wantErr:    false,
			validate: func(t *testing.T, results []*KnowledgeEntry) {
				assert.Len(t, results, 2, "Should respect limit parameter")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := storage.ListKnowledge(tt.collection, tt.limit)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, results)
				if tt.validate != nil {
					tt.validate(t, results)
				}
			}
		})
	}
}

// TestKnowledgeStorage_Query tests semantic search functionality
func TestKnowledgeStorage_Query(t *testing.T) {
	storage, _, mockQdrant, cleanup := setupTestStorage(t)
	defer cleanup()

	collectionName := "test_collection"
	col, err := storage.CreateCollection(collectionName, "test", "Test collection", []string{})
	assert.NoError(t, err)

	// Create test entries
	mockQdrant.On("StorePoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	entry1, err := storage.Upsert(collectionName, "Machine learning algorithms", nil, nil)
	assert.NoError(t, err)
	entry2, err := storage.Upsert(collectionName, "Database design patterns", nil, stringPtr("task-1"))
	assert.NoError(t, err)

	tests := []struct {
		name       string
		collection string
		query      string
		limit      int
		taskId     *string
		setupMock  func(*MockQdrantClient)
		wantErr    bool
		validate   func(*testing.T, []*QueryResult)
	}{
		{
			name:       "Semantic search with query",
			collection: collectionName,
			query:      "machine learning",
			limit:      5,
			taskId:     nil,
			setupMock: func(m *MockQdrantClient) {
				results := []*QdrantQueryResult{
					{Entry: entry1, Score: 0.95},
				}
				m.On("SearchSimilarWithFilter", mock.Anything, "machine learning", 5, mock.MatchedBy(func(filter map[string]interface{}) bool {
					must, ok := filter["must"].([]map[string]interface{})
					if !ok || len(must) == 0 {
						return false
					}
					// Check collectionId filter exists
					return must[0]["key"] == "collectionId"
				}), mock.Anything).Return(results, nil).Once()
			},
			wantErr: false,
			validate: func(t *testing.T, results []*QueryResult) {
				assert.Len(t, results, 1)
				assert.Equal(t, entry1.ID, results[0].Entry.ID)
				assert.Greater(t, results[0].Score, 0.9)
			},
		},
		{
			name:       "Limit parameter",
			collection: collectionName,
			query:      "database",
			limit:      1,
			taskId:     nil,
			setupMock: func(m *MockQdrantClient) {
				results := []*QdrantQueryResult{
					{Entry: entry2, Score: 0.85},
				}
				m.On("SearchSimilarWithFilter", mock.Anything, "database", 1, mock.Anything, mock.Anything).Return(results, nil).Once()
			},
			wantErr: false,
			validate: func(t *testing.T, results []*QueryResult) {
				assert.LessOrEqual(t, len(results), 1, "Should respect limit")
			},
		},
		{
			name:       "Collection filtering",
			collection: collectionName,
			query:      "patterns",
			limit:      5,
			taskId:     nil,
			setupMock: func(m *MockQdrantClient) {
				m.On("SearchSimilarWithFilter", mock.Anything, "patterns", 5, mock.MatchedBy(func(filter map[string]interface{}) bool {
					must := filter["must"].([]map[string]interface{})
					matchFilter := must[0]["match"].(map[string]interface{})
					return matchFilter["value"] == col.ID.Hex()
				}), mock.Anything).Return([]*QdrantQueryResult{}, nil).Once()
			},
			wantErr: false,
		},
		{
			name:       "Empty results",
			collection: collectionName,
			query:      "nonexistent topic",
			limit:      5,
			taskId:     nil,
			setupMock: func(m *MockQdrantClient) {
				m.On("SearchSimilarWithFilter", mock.Anything, "nonexistent topic", 5, mock.Anything, mock.Anything).Return([]*QdrantQueryResult{}, nil).Once()
			},
			wantErr: false,
			validate: func(t *testing.T, results []*QueryResult) {
				assert.NotNil(t, results, "Should return empty array, not nil")
			},
		},
		{
			name:       "TaskId filtering",
			collection: collectionName,
			query:      "database",
			limit:      5,
			taskId:     stringPtr("task-1"),
			setupMock: func(m *MockQdrantClient) {
				results := []*QdrantQueryResult{
					{Entry: entry2, Score: 0.85},
				}
				m.On("SearchSimilarWithFilter", mock.Anything, "database", 5, mock.MatchedBy(func(filter map[string]interface{}) bool {
					must := filter["must"].([]map[string]interface{})
					// Should have both collectionId and taskId filters
					if len(must) != 2 {
						return false
					}
					return must[1]["key"] == "taskId"
				}), mock.Anything).Return(results, nil).Once()
			},
			wantErr: false,
			validate: func(t *testing.T, results []*QueryResult) {
				assert.Len(t, results, 1)
				assert.Equal(t, "task-1", results[0].Entry.TaskId)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock(mockQdrant)
			}

			results, err := storage.Query(tt.collection, tt.query, tt.limit, tt.taskId)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, results)
				if tt.validate != nil {
					tt.validate(t, results)
				}
			}
		})
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
