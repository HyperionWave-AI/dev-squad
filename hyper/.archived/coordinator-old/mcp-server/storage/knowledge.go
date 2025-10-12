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

// CollectionStats represents collection popularity statistics
type CollectionStats struct {
	Collection string `json:"collection"`
	Count      int    `json:"count"`
}

// CollectionWithMetadata represents a collection with stats and category metadata
type CollectionWithMetadata struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// KnowledgeStorage provides storage interface for knowledge entries
type KnowledgeStorage interface {
	Upsert(collection, text string, metadata map[string]interface{}) (*KnowledgeEntry, error)
	Query(collection, query string, limit int) ([]*QueryResult, error)
	ListKnowledge(collection string, limit int) ([]*KnowledgeEntry, error)
	ListCollections() []string
	GetPopularCollections(limit int) ([]*CollectionStats, error)
	GetCollectionStatsWithMetadata() ([]*CollectionWithMetadata, error)
}

// MongoKnowledgeStorage implements KnowledgeStorage using MongoDB + Qdrant
type MongoKnowledgeStorage struct {
	knowledgeCollection *mongo.Collection
	qdrantClient        QdrantClientInterface
	vectorDimension     int
}

// NewMongoKnowledgeStorage creates a new MongoDB + Qdrant knowledge storage
func NewMongoKnowledgeStorage(db *mongo.Database, qdrantClient QdrantClientInterface) (*MongoKnowledgeStorage, error) {
	storage := &MongoKnowledgeStorage{
		knowledgeCollection: db.Collection("knowledge_entries"),
		qdrantClient:        qdrantClient,
		vectorDimension:     768, // TEI nomic-embed-text-v1.5 dimension
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

// Upsert stores or updates a knowledge entry in both MongoDB and Qdrant
func (s *MongoKnowledgeStorage) Upsert(collection, text string, metadata map[string]interface{}) (*KnowledgeEntry, error) {
	ctx := context.Background()

	entry := &KnowledgeEntry{
		ID:         uuid.New().String(),
		Collection: collection,
		Text:       text,
		Metadata:   metadata,
		CreatedAt:  time.Now().UTC(),
	}

	// Store in MongoDB for metadata and audit trail
	_, err := s.knowledgeCollection.InsertOne(ctx, entry)
	if err != nil {
		return nil, fmt.Errorf("failed to insert knowledge entry in MongoDB: %w", err)
	}

	// Store in Qdrant for vector search
	if s.qdrantClient != nil {
		// Ensure collection exists
		if err := s.qdrantClient.EnsureCollection(collection, s.vectorDimension); err != nil {
			// Log error but don't fail - MongoDB has the data
			fmt.Printf("Warning: failed to ensure Qdrant collection: %v\n", err)
		} else {
			// Store vector point
			if err := s.qdrantClient.StorePoint(collection, entry.ID, text, metadata); err != nil {
				// Log error but don't fail - MongoDB has the data
				fmt.Printf("Warning: failed to store in Qdrant: %v\n", err)
			}
		}
	}

	return entry, nil
}

// Query searches for knowledge entries using Qdrant vector search
func (s *MongoKnowledgeStorage) Query(collection, query string, limit int) ([]*QueryResult, error) {
	ctx := context.Background()

	// Use Qdrant for semantic vector search if available
	if s.qdrantClient != nil {
		results, err := s.qdrantClient.SearchSimilar(collection, query, limit)
		if err == nil && len(results) > 0 {
			// Convert QdrantQueryResult to QueryResult
			queryResults := make([]*QueryResult, len(results))
			for i, r := range results {
				queryResults[i] = &QueryResult{
					Entry: r.Entry,
					Score: r.Score,
				}
			}
			return queryResults, nil
		}
		// Log error but continue to MongoDB fallback
		if err != nil {
			fmt.Printf("Warning: Qdrant search failed, falling back to MongoDB: %v\n", err)
		}
	}

	// Fallback to MongoDB text search
	filter := bson.M{
		"collection": collection,
		"$text":      bson.M{"$search": query},
	}

	opts := options.Find().
		SetProjection(bson.D{{Key: "score", Value: bson.D{{Key: "$meta", Value: "textScore"}}}}).
		SetSort(bson.D{{Key: "score", Value: bson.D{{Key: "$meta", Value: "textScore"}}}})

	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := s.knowledgeCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query knowledge in MongoDB: %w", err)
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

	// Convert to QueryResult format
	results := make([]*QueryResult, len(entries))
	for i, entry := range entries {
		results[i] = &QueryResult{
			Entry: entry,
			Score: 0.7, // Default score for MongoDB text matches
		}
	}

	return results, nil
}

// ListKnowledge retrieves knowledge entries from a collection without search (browse mode)
// Returns entries sorted by creation date (newest first)
func (s *MongoKnowledgeStorage) ListKnowledge(collection string, limit int) ([]*KnowledgeEntry, error) {
	ctx := context.Background()

	// Filter by collection
	filter := bson.M{"collection": collection}

	// Sort by creation date descending (newest first)
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})

	// Apply limit
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := s.knowledgeCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list knowledge entries: %w", err)
	}
	defer cursor.Close(ctx)

	// Initialize empty slice (never return nil)
	entries := make([]*KnowledgeEntry, 0)
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, fmt.Errorf("failed to decode knowledge entries: %w", err)
	}

	return entries, nil
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

// GetPopularCollections returns top N collections by entry count
func (s *MongoKnowledgeStorage) GetPopularCollections(limit int) ([]*CollectionStats, error) {
	ctx := context.Background()

	// MongoDB aggregation pipeline:
	// 1. Group by collection and count
	// 2. Sort by count descending
	// 3. Limit to top N
	pipeline := []bson.M{
		{
			"$group": bson.M{
				"_id":   "$collection",
				"count": bson.M{"$sum": 1},
			},
		},
		{
			"$sort": bson.M{"count": -1},
		},
	}

	if limit > 0 {
		pipeline = append(pipeline, bson.M{"$limit": limit})
	}

	cursor, err := s.knowledgeCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate collections: %w", err)
	}
	defer cursor.Close(ctx)

	// CRITICAL: Initialize empty slice instead of nil - never return null
	results := make([]*CollectionStats, 0)

	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		results = append(results, &CollectionStats{
			Collection: result.ID,
			Count:      result.Count,
		})
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	// Return empty slice (not nil) if no results
	return results, nil
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

// GetCollectionStatsWithMetadata returns all collections with stats and category metadata
func (s *MongoKnowledgeStorage) GetCollectionStatsWithMetadata() ([]*CollectionWithMetadata, error) {
	// Hardcoded metadata mapping (from MCP resource handler)
	categoryMap := map[string]string{
		"team-coordination":              "Task",
		"agent-coordination":             "Task",
		"technical-knowledge":            "Tech",
		"code-patterns":                  "Tech",
		"adr":                            "Tech",
		"data-contracts":                 "Tech",
		"technical-debt-registry":        "Tech",
		"ui-component-patterns":          "UI",
		"ui-test-strategies":             "UI",
		"ui-accessibility-standards":     "UI",
		"ui-visual-regression-baseline":  "UI",
		"mcp-operations":                 "Ops",
		"code-quality-violations":        "Ops",
	}

	// Get collection stats from MongoDB
	stats, err := s.GetPopularCollections(0) // 0 = no limit, get all
	if err != nil {
		return nil, err
	}

	// Merge with metadata
	results := make([]*CollectionWithMetadata, 0, len(stats))
	for _, stat := range stats {
		category := categoryMap[stat.Collection]

		// Handle dynamic task collections
		if category == "" {
			if len(stat.Collection) > 5 && stat.Collection[:5] == "task:" {
				category = "Task"
			} else {
				category = "Other"
			}
		}

		results = append(results, &CollectionWithMetadata{
			Name:     stat.Collection,
			Category: category,
			Count:    stat.Count,
		})
	}

	return results, nil
}