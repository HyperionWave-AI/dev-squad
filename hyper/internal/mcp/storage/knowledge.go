package storage

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// Collection represents a knowledge collection with immutable ID
// Solves the Qdrant rename problem by using immutable IDs
type Collection struct {
	ID          primitive.ObjectID `json:"id" bson:"_id"`
	Name        string             `json:"name" bson:"name"`                     // User-facing, renamable
	QdrantName  string             `json:"qdrantName" bson:"qdrantName"`         // Immutable, always matches Qdrant collection
	Category    string             `json:"category" bson:"category"`
	Description string             `json:"description" bson:"description"`
	Tags        []string           `json:"tags" bson:"tags"`
	EntryCount  int                `json:"entryCount" bson:"entryCount"`         // Cached count
	CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// KnowledgeEntry represents a stored knowledge item
type KnowledgeEntry struct {
	ID           string                 `json:"id" bson:"entryId"`
	CollectionID primitive.ObjectID     `json:"collectionId,omitempty" bson:"collectionId,omitempty"` // References Collection._id
	Collection   string                 `json:"collection" bson:"collection"`                         // DEPRECATED: Keep for backward compatibility
	TaskId       string                 `json:"taskId,omitempty" bson:"taskId,omitempty"`             // Optional task ID for task-scoped filtering
	Text         string                 `json:"text" bson:"text"`
	Metadata     map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"createdAt" bson:"createdAt"`
}

// Vote represents a user vote on a knowledge entry
type Vote struct {
	EntryID   string    `json:"entryId" bson:"entryId"`
	UserID    string    `json:"userId" bson:"userId"`
	Vote      string    `json:"vote" bson:"vote"` // "+" or "-"
	Reason    string    `json:"reason" bson:"reason"`
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

// VoteSummary represents voting statistics for a knowledge entry
type VoteSummary struct {
	Upvotes   int    `json:"upvotes"`
	Downvotes int    `json:"downvotes"`
	NetScore  int    `json:"netScore"`
	UserVote  string `json:"userVote,omitempty"` // User's current vote ("+", "-", or empty)
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
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Count       int      `json:"count"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// CollectionMetadata represents metadata for a collection
type CollectionMetadata struct {
	CollectionName string    `json:"collectionName" bson:"collectionName"`
	Description    string    `json:"description" bson:"description"`
	Tags           []string  `json:"tags" bson:"tags"`
	Category       string    `json:"category" bson:"category"`
	CreatedAt      time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt" bson:"updatedAt"`
}

// KnowledgeStorage provides storage interface for knowledge entries
type KnowledgeStorage interface {
	Upsert(collection, text string, metadata map[string]interface{}, taskId *string) (*KnowledgeEntry, error)
	UpdateEntry(id, text string, metadata map[string]interface{}) (*KnowledgeEntry, error)
	DeleteEntry(id string) error
	GetEntryByID(id string) (*KnowledgeEntry, error)
	GetEntriesByCollection(collectionName string) ([]*KnowledgeEntry, error)
	Query(collection, query string, limit int, taskId *string, voteBoost ...float64) ([]*QueryResult, error)
	ListCollections() []string
	CreateCollection(name, category, description string, tags []string) (*Collection, error)
	DeleteCollection(id string) (string, int64, error)
	GetPopularCollections(limit int) ([]*CollectionStats, error)
	GetCollectionStatsWithMetadata() ([]*CollectionWithMetadata, error)
	ListKnowledge(collection string, limit int) ([]*KnowledgeEntry, error)
	UpdateCollectionMetadata(collectionName, description string, tags []string, category string) (*CollectionMetadata, error)
	RenameCollection(oldName, newName string) (int64, error)
	VoteOnEntry(entryID, userID, vote, reason string) (*Vote, error)
	GetEntryVotes(entryID, userID string) (*VoteSummary, error)
	BatchSyncVotesToQdrant(collectionName string) (int, error)
}

// MongoKnowledgeStorage implements KnowledgeStorage using MongoDB + Qdrant
type MongoKnowledgeStorage struct {
	knowledgeCollection   *mongo.Collection
	collectionsCollection *mongo.Collection // NEW: Collection objects
	metadataCollection    *mongo.Collection
	votesCollection       *mongo.Collection
	qdrantClient          QdrantClientInterface
	vectorDimension       int
	logger                *zap.Logger
}

// NewMongoKnowledgeStorage creates a new MongoDB + Qdrant knowledge storage
func NewMongoKnowledgeStorage(db *mongo.Database, qdrantClient QdrantClientInterface, logger *zap.Logger) (*MongoKnowledgeStorage, error) {
	storage := &MongoKnowledgeStorage{
		knowledgeCollection:   db.Collection("knowledge_entries"),
		collectionsCollection: db.Collection("collections"),
		metadataCollection:    db.Collection("collection_metadata"),
		votesCollection:       db.Collection("knowledge_votes"),
		qdrantClient:          qdrantClient,
		vectorDimension:       qdrantClient.GetDimensions(), // Get dimensions from shared embedding client
		logger:                logger,
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

	// Sparse index on taskId for efficient task-scoped filtering
	_, err = storage.knowledgeCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "taskId", Value: 1}},
		Options: options.Index().SetSparse(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create taskId index: %w", err)
	}

	// Index on collection_metadata.collectionName (unique)
	_, err = storage.metadataCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "collectionName", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create collection metadata index: %w", err)
	}

	// Index on collections.name (unique)
	_, err = storage.collectionsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create collections name index: %w", err)
	}

	// Index on collections.qdrantName (unique) - ensures 1:1 mapping to Qdrant
	_, err = storage.collectionsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "qdrantName", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create collections qdrantName index: %w", err)
	}

	// Index on collections.createdAt for sorting
	_, err = storage.collectionsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "createdAt", Value: -1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create collections createdAt index: %w", err)
	}

	// Compound unique index on knowledge_votes (entryId + userId) for upsert pattern
	_, err = storage.votesCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "entryId", Value: 1},
			{Key: "userId", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create votes compound index: %w", err)
	}

	// Index on entryId for efficient vote retrieval
	_, err = storage.votesCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "entryId", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create votes entryId index: %w", err)
	}

	return storage, nil
}

// CreateCollection creates a new Collection object with immutable ID and QdrantName
func (s *MongoKnowledgeStorage) CreateCollection(name, category, description string, tags []string) (*Collection, error) {
	ctx := context.Background()

	// Validate name is unique
	count, err := s.collectionsCollection.CountDocuments(ctx, bson.M{"name": name})
	if err != nil {
		return nil, fmt.Errorf("failed to check collection name: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("collection already exists: %s", name)
	}

	// Ensure tags is not nil
	if tags == nil {
		tags = []string{}
	}

	now := time.Now().UTC()
	newID := primitive.NewObjectID()
	collection := &Collection{
		ID:          newID,
		Name:        name,
		QdrantName:  newID.Hex(), // Generate unique Qdrant name from ObjectID
		Category:    category,
		Description: description,
		Tags:        tags,
		EntryCount:  0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Create Qdrant collection
	if s.qdrantClient != nil {
		if err := s.qdrantClient.EnsureCollection(collection.QdrantName, s.vectorDimension); err != nil {
			return nil, fmt.Errorf("failed to create Qdrant collection: %w", err)
		}
	}

	// Insert into MongoDB
	_, err = s.collectionsCollection.InsertOne(ctx, collection)
	if err != nil {
		return nil, fmt.Errorf("failed to insert collection: %w", err)
	}

	s.logger.Info("Created new collection",
		zap.String("name", name),
		zap.String("id", collection.ID.Hex()),
		zap.String("qdrantName", collection.QdrantName))

	return collection, nil
}

// GetCollection retrieves a collection by name
func (s *MongoKnowledgeStorage) GetCollection(name string) (*Collection, error) {
	ctx := context.Background()
	var collection Collection
	err := s.collectionsCollection.FindOne(ctx, bson.M{"name": name}).Decode(&collection)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("collection not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}
	return &collection, nil
}

// GetCollectionByID retrieves a collection by its ObjectID
func (s *MongoKnowledgeStorage) GetCollectionByID(id primitive.ObjectID) (*Collection, error) {
	ctx := context.Background()
	var collection Collection
	err := s.collectionsCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&collection)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("collection not found by ID")
		}
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}
	return &collection, nil
}

// ListCollectionsObjects returns all Collection objects sorted by name
func (s *MongoKnowledgeStorage) ListCollectionsObjects() ([]*Collection, error) {
	ctx := context.Background()
	cursor, err := s.collectionsCollection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	defer cursor.Close(ctx)

	collections := make([]*Collection, 0)
	if err := cursor.All(ctx, &collections); err != nil {
		return nil, fmt.Errorf("failed to decode collections: %w", err)
	}

	return collections, nil
}

// DeleteCollection deletes an entire collection including:
// - All knowledge entries in MongoDB (both collectionId and old collection field)
// - The Qdrant collection (vector data)
// - The Collection object itself
// Returns the collection name and count of deleted entries
func (s *MongoKnowledgeStorage) DeleteCollection(id string) (string, int64, error) {
	ctx := context.Background()

	// Parse collection ID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return "", 0, fmt.Errorf("invalid collection ID: %w", err)
	}

	// Find Collection object by ID
	collectionObj, err := s.GetCollectionByID(objectID)
	if err != nil {
		return "", 0, fmt.Errorf("collection not found: %w", err)
	}

	s.logger.Info("Deleting collection",
		zap.String("id", id),
		zap.String("name", collectionObj.Name),
		zap.String("qdrantName", collectionObj.QdrantName))

	// Delete all knowledge entries with this collection
	// Support both collectionId (new) and collection (old) for backward compatibility
	entryFilter := bson.M{
		"$or": []bson.M{
			{"collectionId": collectionObj.ID},
			{"collection": collectionObj.Name},
		},
	}

	result, err := s.knowledgeCollection.DeleteMany(ctx, entryFilter)
	if err != nil {
		return collectionObj.Name, 0, fmt.Errorf("failed to delete knowledge entries: %w", err)
	}

	entriesDeleted := result.DeletedCount
	s.logger.Info("Deleted knowledge entries",
		zap.String("id", id),
		zap.String("name", collectionObj.Name),
		zap.Int64("count", entriesDeleted))

	// Delete Qdrant collection if client is available
	if s.qdrantClient != nil {
		err := s.qdrantClient.DeleteCollection(collectionObj.QdrantName)
		if err != nil {
			s.logger.Warn("Failed to delete Qdrant collection (continuing anyway)",
				zap.String("qdrantName", collectionObj.QdrantName),
				zap.Error(err))
		} else {
			s.logger.Info("Deleted Qdrant collection",
				zap.String("qdrantName", collectionObj.QdrantName))
		}
	}

	// Delete the Collection object itself
	_, err = s.collectionsCollection.DeleteOne(ctx, bson.M{"_id": collectionObj.ID})
	if err != nil {
		return collectionObj.Name, entriesDeleted, fmt.Errorf("failed to delete collection object: %w", err)
	}

	s.logger.Info("Collection deleted successfully",
		zap.String("id", id),
		zap.String("name", collectionObj.Name),
		zap.Int64("entriesDeleted", entriesDeleted))

	return collectionObj.Name, entriesDeleted, nil
}

// RebuildCollectionCounts recalculates and updates entry counts for all collections
// This is useful for fixing collections with incorrect cached counts
func (s *MongoKnowledgeStorage) RebuildCollectionCounts() (map[string]interface{}, error) {
	ctx := context.Background()

	// Get all Collection objects
	collections, err := s.ListCollectionsObjects()
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	stats := map[string]interface{}{
		"collectionsUpdated": 0,
		"totalEntries":       0,
		"details":            []map[string]interface{}{},
		"errors":             []string{},
	}

	for _, collection := range collections {
		// Count actual entries for this collection
		// Support both collectionId (new) and collection (old) for backward compatibility
		filter := bson.M{
			"$or": []bson.M{
				{"collectionId": collection.ID},
				{"collection": collection.Name},
			},
		}

		actualCount, err := s.knowledgeCollection.CountDocuments(ctx, filter)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to count entries for collection %s: %v", collection.Name, err)
			stats["errors"] = append(stats["errors"].([]string), errMsg)
			continue
		}

		// Update the collection's entryCount if it differs from actual count
		if int(actualCount) != collection.EntryCount {
			_, err := s.collectionsCollection.UpdateOne(ctx,
				bson.M{"_id": collection.ID},
				bson.M{"$set": bson.M{"entryCount": int(actualCount)}},
			)
			if err != nil {
				errMsg := fmt.Sprintf("Failed to update count for collection %s: %v", collection.Name, err)
				stats["errors"] = append(stats["errors"].([]string), errMsg)
				continue
			}

			stats["collectionsUpdated"] = stats["collectionsUpdated"].(int) + 1

			s.logger.Info("Rebuilt collection entry count",
				zap.String("collection", collection.Name),
				zap.String("id", collection.ID.Hex()),
				zap.Int("oldCount", collection.EntryCount),
				zap.Int64("actualCount", actualCount))
		}

		stats["totalEntries"] = stats["totalEntries"].(int) + int(actualCount)

		// Add collection details
		details := stats["details"].([]map[string]interface{})
		details = append(details, map[string]interface{}{
			"name":        collection.Name,
			"id":          collection.ID.Hex(),
			"oldCount":    collection.EntryCount,
			"actualCount": int(actualCount),
			"updated":     int(actualCount) != collection.EntryCount,
		})
		stats["details"] = details
	}

	return stats, nil
}

// MigrateToCollectionObjects creates Collection objects from existing knowledge entries
// This is a one-time migration function that converts string-based collections to Collection objects
func (s *MongoKnowledgeStorage) MigrateToCollectionObjects() (map[string]interface{}, error) {
	ctx := context.Background()

	// Get distinct collection names from knowledge entries
	collectionNames, err := s.knowledgeCollection.Distinct(ctx, "collection", bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to get distinct collections: %w", err)
	}

	stats := map[string]interface{}{
		"collectionsCreated": 0,
		"entriesUpdated":     0,
		"errors":             []string{},
	}

	// Get existing metadata
	metadataMap := make(map[string]*CollectionMetadata)
	cursor, err := s.metadataCollection.Find(ctx, bson.M{})
	if err == nil {
		defer cursor.Close(ctx)
		for cursor.Next(ctx) {
			var metadata CollectionMetadata
			if err := cursor.Decode(&metadata); err == nil {
				metadataMap[metadata.CollectionName] = &metadata
			}
		}
	}

	// Create Collection object for each unique collection name
	for _, collectionNameRaw := range collectionNames {
		collectionName, ok := collectionNameRaw.(string)
		if !ok {
			continue
		}

		// Check if Collection already exists
		existing, _ := s.GetCollection(collectionName)
		if existing != nil {
			s.logger.Info("Collection object already exists, skipping", zap.String("name", collectionName))
			continue
		}

		// Get metadata if it exists
		metadata := metadataMap[collectionName]
		category := "Other"
		description := ""
		tags := []string{}

		if metadata != nil {
			category = metadata.Category
			description = metadata.Description
			tags = metadata.Tags
		}

		now := time.Now().UTC()
		collection := &Collection{
			ID:          primitive.NewObjectID(),
			Name:        collectionName,
			QdrantName:  collectionName, // IMPORTANT: Keep old name for existing Qdrant collections
			Category:    category,
			Description: description,
			Tags:        tags,
			EntryCount:  0,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// Insert Collection document
		_, err := s.collectionsCollection.InsertOne(ctx, collection)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to create collection %s: %v", collectionName, err)
			stats["errors"] = append(stats["errors"].([]string), errMsg)
			continue
		}

		stats["collectionsCreated"] = stats["collectionsCreated"].(int) + 1

		// Update knowledge entries to reference new Collection ID
		filter := bson.M{"collection": collectionName}
		update := bson.M{"$set": bson.M{"collectionId": collection.ID}}
		result, err := s.knowledgeCollection.UpdateMany(ctx, filter, update)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to update entries for collection %s: %v", collectionName, err)
			stats["errors"] = append(stats["errors"].([]string), errMsg)
		} else {
			stats["entriesUpdated"] = stats["entriesUpdated"].(int) + int(result.ModifiedCount)
		}

		s.logger.Info("Migrated collection",
			zap.String("name", collectionName),
			zap.String("id", collection.ID.Hex()),
			zap.String("qdrantName", collection.QdrantName),
			zap.Int64("entriesUpdated", result.ModifiedCount))
	}

	return stats, nil
}

// Upsert stores or updates a knowledge entry in both MongoDB and Qdrant
func (s *MongoKnowledgeStorage) Upsert(collection, text string, metadata map[string]interface{}, taskId *string) (*KnowledgeEntry, error) {
	ctx := context.Background()

	// Lookup Collection object by name
	collectionObj, err := s.GetCollection(collection)
	if err != nil {
		return nil, fmt.Errorf("collection not found, please create it first: %w", err)
	}

	entry := &KnowledgeEntry{
		ID:           uuid.New().String(),
		CollectionID: collectionObj.ID,      // NEW: Set collection ID
		Collection:   collection,            // Keep for backward compatibility
		Text:         text,
		Metadata:     metadata,
		CreatedAt:    time.Now().UTC(),
	}

	// Set taskId if provided
	if taskId != nil && *taskId != "" {
		entry.TaskId = *taskId
	}

	// Store in MongoDB for metadata and audit trail
	_, err = s.knowledgeCollection.InsertOne(ctx, entry)
	if err != nil {
		return nil, fmt.Errorf("failed to insert knowledge entry in MongoDB: %w", err)
	}

	// Increment the collection's entry count atomically
	_, err = s.collectionsCollection.UpdateOne(ctx,
		bson.M{"_id": collectionObj.ID},
		bson.M{"$inc": bson.M{"entryCount": 1}},
	)
	if err != nil {
		s.logger.Warn("Failed to increment collection entry count",
			zap.String("collectionId", collectionObj.ID.Hex()),
			zap.Error(err))
	}

	// Store in Qdrant using QdrantName (immutable)
	if s.qdrantClient != nil {
		// Ensure collection exists using immutable QdrantName
		if err := s.qdrantClient.EnsureCollection(collectionObj.QdrantName, s.vectorDimension); err != nil {
			// Log error but don't fail - MongoDB has the data
			s.logger.Warn("Failed to ensure Qdrant collection",
				zap.String("qdrantName", collectionObj.QdrantName),
				zap.Error(err))
		} else {
			// Prepare Qdrant payload with taskId if provided
			qdrantPayload := metadata
			if qdrantPayload == nil {
				qdrantPayload = make(map[string]interface{})
			}
			if taskId != nil && *taskId != "" {
				qdrantPayload["taskId"] = *taskId
			}

			// Store vector point using QdrantName
			if err := s.qdrantClient.StorePoint(collectionObj.QdrantName, entry.ID, text, qdrantPayload); err != nil {
				// Log error but don't fail - MongoDB has the data
				s.logger.Warn("Failed to store point in Qdrant",
					zap.String("qdrantName", collectionObj.QdrantName),
					zap.String("entryId", entry.ID),
					zap.Error(err))
			}
		}
	}

	return entry, nil
}

// UpdateEntry updates an existing knowledge entry in both MongoDB and Qdrant
func (s *MongoKnowledgeStorage) UpdateEntry(id, text string, metadata map[string]interface{}) (*KnowledgeEntry, error) {
	ctx := context.Background()

	// Find existing entry to get its collection name
	var existingEntry KnowledgeEntry
	err := s.knowledgeCollection.FindOne(ctx, bson.M{"entryId": id}).Decode(&existingEntry)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("knowledge entry not found: %s", id)
		}
		return nil, fmt.Errorf("failed to find knowledge entry: %w", err)
	}

	// Update MongoDB entry
	update := bson.M{
		"$set": bson.M{
			"text":     text,
			"metadata": metadata,
		},
	}

	_, err = s.knowledgeCollection.UpdateOne(ctx, bson.M{"entryId": id}, update)
	if err != nil {
		return nil, fmt.Errorf("failed to update knowledge entry in MongoDB: %w", err)
	}

	// Update Qdrant point with new embedding
	if s.qdrantClient != nil {
		// Delete old point
		if err := s.qdrantClient.DeletePoint(existingEntry.Collection, id); err != nil {
			s.logger.Warn("Failed to delete old Qdrant point",
				zap.String("collection", existingEntry.Collection),
				zap.String("entryId", id),
				zap.Error(err))
		}

		// Store new point with updated embedding
		if err := s.qdrantClient.StorePoint(existingEntry.Collection, id, text, metadata); err != nil {
			s.logger.Warn("Failed to update point in Qdrant",
				zap.String("collection", existingEntry.Collection),
				zap.String("entryId", id),
				zap.Error(err))
		}
	}

	// Fetch and return updated entry
	var updatedEntry KnowledgeEntry
	err = s.knowledgeCollection.FindOne(ctx, bson.M{"entryId": id}).Decode(&updatedEntry)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updated entry: %w", err)
	}

	return &updatedEntry, nil
}

// DeleteEntry deletes a knowledge entry from both MongoDB and Qdrant
func (s *MongoKnowledgeStorage) DeleteEntry(id string) error {
	ctx := context.Background()

	// Find existing entry to get its collection name
	var existingEntry KnowledgeEntry
	err := s.knowledgeCollection.FindOne(ctx, bson.M{"entryId": id}).Decode(&existingEntry)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("knowledge entry not found: %s", id)
		}
		return fmt.Errorf("failed to find knowledge entry: %w", err)
	}

	// Delete from MongoDB
	_, err = s.knowledgeCollection.DeleteOne(ctx, bson.M{"entryId": id})
	if err != nil {
		return fmt.Errorf("failed to delete knowledge entry from MongoDB: %w", err)
	}

	// Decrement the collection's entry count atomically
	if existingEntry.CollectionID != primitive.NilObjectID {
		_, err = s.collectionsCollection.UpdateOne(ctx,
			bson.M{"_id": existingEntry.CollectionID},
			bson.M{"$inc": bson.M{"entryCount": -1}},
		)
		if err != nil {
			s.logger.Warn("Failed to decrement collection entry count",
				zap.String("collectionId", existingEntry.CollectionID.Hex()),
				zap.Error(err))
		}
	}

	// Delete from Qdrant
	if s.qdrantClient != nil {
		if err := s.qdrantClient.DeletePoint(existingEntry.Collection, id); err != nil {
			s.logger.Warn("Failed to delete point from Qdrant",
				zap.String("collection", existingEntry.Collection),
				zap.String("entryId", id),
				zap.Error(err))
		}
	}

	return nil
}

// GetEntryByID retrieves a knowledge entry by its ID
func (s *MongoKnowledgeStorage) GetEntryByID(id string) (*KnowledgeEntry, error) {
	ctx := context.Background()

	var entry KnowledgeEntry
	err := s.knowledgeCollection.FindOne(ctx, bson.M{"entryId": id}).Decode(&entry)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("knowledge entry not found: %s", id)
		}
		return nil, fmt.Errorf("failed to find knowledge entry: %w", err)
	}

	return &entry, nil
}

// GetEntriesByCollection retrieves all knowledge entries for a collection from MongoDB
// This is used for re-embedding data when recreating Qdrant collections after dimension changes
func (s *MongoKnowledgeStorage) GetEntriesByCollection(collectionName string) ([]*KnowledgeEntry, error) {
	ctx := context.Background()

	// Lookup Collection object by name
	collectionObj, err := s.GetCollection(collectionName)
	if err != nil {
		return nil, fmt.Errorf("collection not found: %w", err)
	}

	// Build filter - support both collectionId (new) and collection (old) for backward compatibility
	filter := bson.M{
		"$or": []bson.M{
			{"collectionId": collectionObj.ID},
			{"collection": collectionName},
		},
	}

	// Query MongoDB for all entries
	cursor, err := s.knowledgeCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch entries: %w", err)
	}
	defer cursor.Close(ctx)

	// Decode results
	var entries []*KnowledgeEntry
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, fmt.Errorf("failed to decode entries: %w", err)
	}

	// Return empty slice (not nil) if no results
	if entries == nil {
		entries = make([]*KnowledgeEntry, 0)
	}

	s.logger.Info("Retrieved entries from MongoDB for reindexing",
		zap.String("collection", collectionName),
		zap.Int("count", len(entries)))

	return entries, nil
}

// Query searches for knowledge entries using Qdrant vector search with optional vote boosting
// taskId is an optional parameter for filtering by task
// voteBoost is an optional parameter (default: 0.0 for no boosting)
func (s *MongoKnowledgeStorage) Query(collection, query string, limit int, taskId *string, voteBoost ...float64) ([]*QueryResult, error) {
	ctx := context.Background()

	// Lookup Collection object by name
	collectionObj, err := s.GetCollection(collection)
	if err != nil {
		return nil, fmt.Errorf("collection not found: %w", err)
	}

	// Use Qdrant for semantic vector search if available (using immutable QdrantName)
	if s.qdrantClient != nil {
		// Build filter for Qdrant if taskId is provided
		var qdrantFilter map[string]interface{}
		if taskId != nil && *taskId != "" {
			qdrantFilter = map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"key":   "taskId",
						"match": map[string]interface{}{"value": *taskId},
					},
				},
			}
		}

		results, err := s.qdrantClient.SearchSimilarWithFilter(collectionObj.QdrantName, query, limit, qdrantFilter, voteBoost...)
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

		// Check if error is a dimension mismatch - trigger auto-recovery
		if err != nil {
			if dimErr, ok := err.(*DimensionMismatchError); ok {
				s.logger.Warn("Detected dimension mismatch, initiating auto-recovery",
					zap.String("collection", collection),
					zap.String("qdrantName", collectionObj.QdrantName),
					zap.Int("expectedDim", dimErr.ExpectedDim),
					zap.Int("gotDim", dimErr.GotDim))

				// Step 1: Get all entries from MongoDB (source of truth)
				entries, err := s.GetEntriesByCollection(collection)
				if err != nil {
					s.logger.Error("Auto-recovery failed: could not retrieve entries from MongoDB",
						zap.String("collection", collection),
						zap.Error(err))
					return nil, fmt.Errorf("auto-recovery failed: %w", err)
				}

				// Step 2: Recreate collection with correct dimensions and reindex
				currentDimensions := s.qdrantClient.GetDimensions()
				reindexedCount, err := s.qdrantClient.RecreateCollectionWithReindex(
					collectionObj.QdrantName,
					entries,
					currentDimensions,
				)
				if err != nil {
					s.logger.Error("Auto-recovery failed: could not recreate collection",
						zap.String("collection", collection),
						zap.String("qdrantName", collectionObj.QdrantName),
						zap.Int("dimensions", currentDimensions),
						zap.Error(err))
					return nil, fmt.Errorf("auto-recovery failed: %w", err)
				}

				s.logger.Info("Auto-recovery completed successfully",
					zap.String("collection", collection),
					zap.String("qdrantName", collectionObj.QdrantName),
					zap.Int("entriesReindexed", reindexedCount),
					zap.Int("totalEntries", len(entries)),
					zap.Int("newDimensions", currentDimensions))

				// Step 3: Retry the search
				results, err = s.qdrantClient.SearchSimilarWithFilter(collectionObj.QdrantName, query, limit, qdrantFilter, voteBoost...)
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

				// If retry still fails, log and fall back to MongoDB
				if err != nil {
					s.logger.Warn("Search failed after auto-recovery, falling back to MongoDB",
						zap.String("collection", collection),
						zap.Error(err))
				}
			} else {
				// Not a dimension mismatch - log as regular error and fall back
				s.logger.Warn("Qdrant search failed, falling back to MongoDB",
					zap.String("qdrantName", collectionObj.QdrantName),
					zap.String("query", query),
					zap.Error(err))
			}
		}
	}

	// Fallback to MongoDB text search (support both collectionId and old collection field)
	filter := bson.M{
		"$or": []bson.M{
			{"collectionId": collectionObj.ID},
			{"collection": collection},
		},
		"$text": bson.M{"$search": query},
	}

	// Add taskId filter if provided
	if taskId != nil && *taskId != "" {
		filter["taskId"] = *taskId
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
		return s.fallbackQuery(ctx, collectionObj, query, limit, taskId)
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

// fallbackQuery performs simple similarity matching when text search fails
func (s *MongoKnowledgeStorage) fallbackQuery(ctx context.Context, collectionObj *Collection, query string, limit int, taskId *string) ([]*QueryResult, error) {
	// Support both collectionId (new) and collection (old) for backward compatibility
	filter := bson.M{
		"$or": []bson.M{
			{"collectionId": collectionObj.ID},
			{"collection": collectionObj.Name},
		},
	}

	// Add taskId filter if provided
	if taskId != nil && *taskId != "" {
		filter["taskId"] = *taskId
	}
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

// GetCollectionStatsWithMetadata returns all collections with stats and category metadata
// Now simplified: reads directly from Collection objects which contain all metadata
func (s *MongoKnowledgeStorage) GetCollectionStatsWithMetadata() ([]*CollectionWithMetadata, error) {
	// Get all Collection objects (already contains name, category, description, tags, entryCount)
	collections, err := s.ListCollectionsObjects()
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	// Convert Collection objects to CollectionWithMetadata
	results := make([]*CollectionWithMetadata, 0, len(collections))
	for _, col := range collections {
		results = append(results, &CollectionWithMetadata{
			ID:          col.ID.Hex(),
			Name:        col.Name,
			Category:    col.Category,
			Count:       col.EntryCount,
			Description: col.Description,
			Tags:        col.Tags,
		})
	}

	return results, nil
}

// ListKnowledge retrieves knowledge entries from a collection without search (browse mode)
// Returns entries sorted by creation date (newest first)
func (s *MongoKnowledgeStorage) ListKnowledge(collection string, limit int) ([]*KnowledgeEntry, error) {
	ctx := context.Background()

	// Lookup Collection object by name
	collectionObj, err := s.GetCollection(collection)
	if err != nil {
		return nil, fmt.Errorf("collection not found: %w", err)
	}

	// Build filter - support both collectionId (new) and collection (old) for backward compatibility
	filter := bson.M{
		"$or": []bson.M{
			{"collectionId": collectionObj.ID},
			{"collection": collection},
		},
	}

	// Set options: sort by creation date descending, limit results
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	// Query MongoDB
	cursor, err := s.knowledgeCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list knowledge entries: %w", err)
	}
	defer cursor.Close(ctx)

	// Decode results
	var entries []*KnowledgeEntry
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, fmt.Errorf("failed to decode knowledge entries: %w", err)
	}

	// Return empty slice (not nil) if no results
	if entries == nil {
		entries = make([]*KnowledgeEntry, 0)
	}

	return entries, nil
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

// UpdateCollectionMetadata updates or creates metadata for a collection
func (s *MongoKnowledgeStorage) UpdateCollectionMetadata(collectionName, description string, tags []string, category string) (*CollectionMetadata, error) {
	ctx := context.Background()

	// Verify collection exists by checking if it has any entries
	count, err := s.knowledgeCollection.CountDocuments(ctx, bson.M{"collection": collectionName})
	if err != nil {
		return nil, fmt.Errorf("failed to verify collection exists: %w", err)
	}
	if count == 0 {
		return nil, fmt.Errorf("collection does not exist: %s", collectionName)
	}

	// Ensure tags is not nil
	if tags == nil {
		tags = []string{}
	}

	now := time.Now()

	// Upsert metadata
	filter := bson.M{"collectionName": collectionName}
	update := bson.M{
		"$set": bson.M{
			"description": description,
			"tags":        tags,
			"category":    category,
			"updatedAt":   now,
		},
		"$setOnInsert": bson.M{
			"createdAt": now,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err = s.metadataCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to update collection metadata: %w", err)
	}

	// Fetch and return the updated metadata
	var metadata CollectionMetadata
	err = s.metadataCollection.FindOne(ctx, filter).Decode(&metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updated metadata: %w", err)
	}

	return &metadata, nil
}

// RenameCollection renames a collection by updating only Collection.Name (instant operation)
// No entry updates needed since entries reference CollectionID (immutable)
// QdrantName stays the same (immutable) so Qdrant collections are unaffected
func (s *MongoKnowledgeStorage) RenameCollection(oldName, newName string) (int64, error) {
	ctx := context.Background()

	// Find Collection object by old name
	collectionObj, err := s.GetCollection(oldName)
	if err != nil {
		return 0, fmt.Errorf("collection not found: %w", err)
	}

	// Verify new collection name doesn't already exist
	existingCount, err := s.collectionsCollection.CountDocuments(ctx, bson.M{"name": newName})
	if err != nil {
		return 0, fmt.Errorf("failed to verify new collection name: %w", err)
	}
	if existingCount > 0 {
		return 0, fmt.Errorf("collection already exists: %s", newName)
	}

	// Update ONLY the Collection.Name field (no entry updates needed!)
	// QdrantName stays the same (immutable), entries reference CollectionID (unchanged)
	filter := bson.M{"_id": collectionObj.ID}
	update := bson.M{"$set": bson.M{
		"name":      newName,
		"updatedAt": time.Now().UTC(),
	}}

	result, err := s.collectionsCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, fmt.Errorf("failed to rename collection: %w", err)
	}

	s.logger.Info("Collection renamed (instant operation - no entry updates needed)",
		zap.String("oldName", oldName),
		zap.String("newName", newName),
		zap.String("collectionId", collectionObj.ID.Hex()),
		zap.String("qdrantName", collectionObj.QdrantName))

	return result.ModifiedCount, nil
}

// VoteOnEntry records or updates a user's vote on a knowledge entry
// Uses upsert pattern: one vote per user per entry
func (s *MongoKnowledgeStorage) VoteOnEntry(entryID, userID, vote, reason string) (*Vote, error) {
	ctx := context.Background()

	// Validate vote is "+" or "-"
	if vote != "+" && vote != "-" {
		return nil, fmt.Errorf("vote must be '+' or '-', got: %s", vote)
	}

	// Validate reason is max 10 words
	words := strings.Fields(reason)
	if len(words) > 10 {
		return nil, fmt.Errorf("reason must be maximum 10 words, got %d words", len(words))
	}

	// Verify entry exists
	var entry KnowledgeEntry
	err := s.knowledgeCollection.FindOne(ctx, bson.M{"entryId": entryID}).Decode(&entry)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("knowledge entry not found: %s", entryID)
		}
		return nil, fmt.Errorf("failed to verify entry exists: %w", err)
	}

	now := time.Now().UTC()

	// Upsert vote (one vote per user per entry)
	filter := bson.M{
		"entryId": entryID,
		"userId":  userID,
	}

	update := bson.M{
		"$set": bson.M{
			"vote":      vote,
			"reason":    reason,
			"updatedAt": now,
		},
		"$setOnInsert": bson.M{
			"createdAt": now,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err = s.votesCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert vote: %w", err)
	}

	// Fetch and return the vote
	var voteRecord Vote
	err = s.votesCollection.FindOne(ctx, filter).Decode(&voteRecord)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch vote record: %w", err)
	}

	// Sync vote data to Qdrant (non-blocking - log errors only)
	if err := s.SyncVoteDataToQdrant(entryID); err != nil {
		s.logger.Warn("Failed to sync vote data to Qdrant",
			zap.String("entryId", entryID),
			zap.Error(err))
	}

	return &voteRecord, nil
}

// GetEntryVotes retrieves voting summary for a knowledge entry
// If userID is provided, includes the user's current vote
func (s *MongoKnowledgeStorage) GetEntryVotes(entryID, userID string) (*VoteSummary, error) {
	ctx := context.Background()

	// Verify entry exists
	var entry KnowledgeEntry
	err := s.knowledgeCollection.FindOne(ctx, bson.M{"entryId": entryID}).Decode(&entry)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("knowledge entry not found: %s", entryID)
		}
		return nil, fmt.Errorf("failed to verify entry exists: %w", err)
	}

	// Get all votes for this entry
	cursor, err := s.votesCollection.Find(ctx, bson.M{"entryId": entryID})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch votes: %w", err)
	}
	defer cursor.Close(ctx)

	var votes []Vote
	if err := cursor.All(ctx, &votes); err != nil {
		return nil, fmt.Errorf("failed to decode votes: %w", err)
	}

	// Calculate summary
	summary := &VoteSummary{
		Upvotes:   0,
		Downvotes: 0,
		NetScore:  0,
		UserVote:  "",
	}

	for _, v := range votes {
		if v.Vote == "+" {
			summary.Upvotes++
		} else if v.Vote == "-" {
			summary.Downvotes++
		}

		// Check if this is the current user's vote
		if userID != "" && v.UserID == userID {
			summary.UserVote = v.Vote
		}
	}

	summary.NetScore = summary.Upvotes - summary.Downvotes

	return summary, nil
}

// SyncVoteDataToQdrant calculates vote aggregates and syncs them to Qdrant payload
// This enables vote-weighted search without re-embedding the entry
func (s *MongoKnowledgeStorage) SyncVoteDataToQdrant(entryID string) error {
	ctx := context.Background()

	// Get entry to find its collection
	var entry KnowledgeEntry
	err := s.knowledgeCollection.FindOne(ctx, bson.M{"entryId": entryID}).Decode(&entry)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("knowledge entry not found: %s", entryID)
		}
		return fmt.Errorf("failed to find entry: %w", err)
	}

	// Calculate vote aggregates
	cursor, err := s.votesCollection.Find(ctx, bson.M{"entryId": entryID})
	if err != nil {
		return fmt.Errorf("failed to fetch votes: %w", err)
	}
	defer cursor.Close(ctx)

	var votes []Vote
	if err := cursor.All(ctx, &votes); err != nil {
		return fmt.Errorf("failed to decode votes: %w", err)
	}

	upvotes := 0
	downvotes := 0
	for _, v := range votes {
		if v.Vote == "+" {
			upvotes++
		} else if v.Vote == "-" {
			downvotes++
		}
	}

	netScore := upvotes - downvotes

	// Update Qdrant payload
	if s.qdrantClient != nil {
		payload := map[string]interface{}{
			"upvotes":        upvotes,
			"downvotes":      downvotes,
			"voteScore":      netScore,
			"lastVoteUpdate": time.Now().UTC().Format(time.RFC3339),
		}

		if err := s.qdrantClient.UpdatePointPayload(entry.Collection, entryID, payload); err != nil {
			return fmt.Errorf("failed to update Qdrant payload: %w", err)
		}
	}

	return nil
}

// BatchSyncVotesToQdrant syncs vote data for all knowledge entries with votes
// This is useful for migrating existing vote data or performing maintenance
// Returns count of entries synced and any errors encountered
func (s *MongoKnowledgeStorage) BatchSyncVotesToQdrant(collectionName string) (int, error) {
	ctx := context.Background()

	// Build filter for entries (optionally filter by collection)
	filter := bson.M{}
	if collectionName != "" {
		filter["collection"] = collectionName
	}

	// Get all knowledge entries
	cursor, err := s.knowledgeCollection.Find(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch entries: %w", err)
	}
	defer cursor.Close(ctx)

	var entries []KnowledgeEntry
	if err := cursor.All(ctx, &entries); err != nil {
		return 0, fmt.Errorf("failed to decode entries: %w", err)
	}

	// Sync each entry that has votes
	syncedCount := 0
	var syncErrors []string

	for _, entry := range entries {
		// Check if entry has any votes
		voteCount, err := s.votesCollection.CountDocuments(ctx, bson.M{"entryId": entry.ID})
		if err != nil {
			syncErrors = append(syncErrors, fmt.Sprintf("Entry %s: failed to count votes: %v", entry.ID, err))
			continue
		}

		// Skip entries with no votes
		if voteCount == 0 {
			continue
		}

		// Sync vote data to Qdrant
		if err := s.SyncVoteDataToQdrant(entry.ID); err != nil {
			syncErrors = append(syncErrors, fmt.Sprintf("Entry %s: %v", entry.ID, err))
			continue
		}

		syncedCount++
	}

	if len(syncErrors) > 0 {
		// Return partial success with error details
		errorMsg := fmt.Sprintf("Synced %d entries, but encountered %d errors: %v", syncedCount, len(syncErrors), syncErrors)
		return syncedCount, fmt.Errorf("%s", errorMsg)
	}

	return syncedCount, nil
}