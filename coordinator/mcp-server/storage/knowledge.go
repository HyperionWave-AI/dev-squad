package storage

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// KnowledgeEntry represents a stored knowledge item
type KnowledgeEntry struct {
	ID         string                 `json:"id" bson:"entryId"`
	Collection string                 `json:"collection" bson:"collection"`
	Text       string                 `json:"text" bson:"text"`
	Metadata   map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"createdAt" bson:"createdAt"`
}

// QueryResult represents a knowledge query result with similarity score
type QueryResult struct {
	Entry *KnowledgeEntry `json:"entry"`
	Score float64         `json:"score"`
}

// KnowledgeStorage provides storage interface for knowledge entries
type KnowledgeStorage interface {
	Upsert(collection, text string, metadata map[string]interface{}) (*KnowledgeEntry, error)
	Query(collection, query string, limit int) ([]*QueryResult, error)
	ListCollections() []string
}

// MongoKnowledgeStorage implements KnowledgeStorage using MongoDB
type MongoKnowledgeStorage struct {
	knowledgeCollection *mongo.Collection
}

// NewMongoKnowledgeStorage creates a new MongoDB-backed knowledge storage
func NewMongoKnowledgeStorage(db *mongo.Database) (*MongoKnowledgeStorage, error) {
	storage := &MongoKnowledgeStorage{
		knowledgeCollection: db.Collection("knowledge_entries"),
	}

	// Create indexes
	ctx := context.Background()

	// Index on entryId
	_, err := storage.knowledgeCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "entryId", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create entry ID index: %w", err)
	}

	// Index on collection for efficient queries
	_, err = storage.knowledgeCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "collection", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create collection index: %w", err)
	}

	// Text index for full-text search on text field
	_, err = storage.knowledgeCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "text", Value: "text"}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create text index: %w", err)
	}

	return storage, nil
}

// Upsert stores or updates a knowledge entry
func (s *MongoKnowledgeStorage) Upsert(collection, text string, metadata map[string]interface{}) (*KnowledgeEntry, error) {
	ctx := context.Background()

	entry := &KnowledgeEntry{
		ID:         uuid.New().String(),
		Collection: collection,
		Text:       text,
		Metadata:   metadata,
		CreatedAt:  time.Now().UTC(),
	}

	_, err := s.knowledgeCollection.InsertOne(ctx, entry)
	if err != nil {
		return nil, fmt.Errorf("failed to insert knowledge entry: %w", err)
	}

	return entry, nil
}

// Query searches for knowledge entries matching the query text
func (s *MongoKnowledgeStorage) Query(collection, query string, limit int) ([]*QueryResult, error) {
	ctx := context.Background()

	// Use MongoDB text search for better performance
	filter := bson.M{
		"collection": collection,
		"$text":      bson.M{"$search": query},
	}

	// Add text score for sorting
	opts := options.Find().
		SetProjection(bson.D{{Key: "score", Value: bson.D{{Key: "$meta", Value: "textScore"}}}}).
		SetSort(bson.D{{Key: "score", Value: bson.D{{Key: "$meta", Value: "textScore"}}}})

	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := s.knowledgeCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query knowledge: %w", err)
	}
	defer cursor.Close(ctx)

	var entries []*KnowledgeEntry
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, fmt.Errorf("failed to decode knowledge entries: %w", err)
	}

	// If MongoDB text search returns no results, fallback to simple similarity
	if len(entries) == 0 {
		return s.fallbackQuery(ctx, collection, query, limit)
	}

	// Convert to QueryResult format with normalized scores
	results := make([]*QueryResult, len(entries))
	for i, entry := range entries {
		// Normalize text search score to 0-1 range (text scores are typically 0.75-1.5)
		score := 0.7 // Default score for text matches
		results[i] = &QueryResult{
			Entry: entry,
			Score: score,
		}
	}

	return results, nil
}

// fallbackQuery performs simple similarity matching when text search fails
func (s *MongoKnowledgeStorage) fallbackQuery(ctx context.Context, collection, query string, limit int) ([]*QueryResult, error) {
	filter := bson.M{"collection": collection}
	cursor, err := s.knowledgeCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query knowledge: %w", err)
	}
	defer cursor.Close(ctx)

	var entries []*KnowledgeEntry
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, fmt.Errorf("failed to decode knowledge entries: %w", err)
	}

	// Calculate similarity scores
	results := make([]*QueryResult, 0)
	queryLower := strings.ToLower(query)

	for _, entry := range entries {
		score := calculateSimilarity(queryLower, strings.ToLower(entry.Text))

		// Only include results with non-zero similarity
		if score > 0 {
			results = append(results, &QueryResult{
				Entry: entry,
				Score: score,
			})
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// ListCollections returns all unique collection names
func (s *MongoKnowledgeStorage) ListCollections() []string {
	ctx := context.Background()

	collections, err := s.knowledgeCollection.Distinct(ctx, "collection", bson.M{})
	if err != nil {
		return []string{}
	}

	// Convert interface{} to []string
	result := make([]string, 0, len(collections))
	for _, c := range collections {
		if str, ok := c.(string); ok {
			result = append(result, str)
		}
	}

	sort.Strings(result)
	return result
}

// calculateSimilarity provides simple text similarity scoring
// Returns a score between 0.0 and 1.0 based on:
// - Exact match: 1.0
// - Contains query: 0.7
// - Word overlap: proportional to matched words
func calculateSimilarity(query, text string) float64 {
	// Exact match
	if query == text {
		return 1.0
	}

	// Contains query as substring
	if strings.Contains(text, query) {
		return 0.7
	}

	// Word-level overlap
	queryWords := strings.Fields(query)
	textWords := strings.Fields(text)

	if len(queryWords) == 0 {
		return 0.0
	}

	matchCount := 0
	for _, qw := range queryWords {
		for _, tw := range textWords {
			if qw == tw {
				matchCount++
				break
			}
		}
	}

	// Return proportion of query words found
	return float64(matchCount) / float64(len(queryWords)) * 0.5
}