package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ToolMetadata represents metadata about an MCP tool
type ToolMetadata struct {
	ID          string                 `json:"id" bson:"toolId"`
	ToolName    string                 `json:"toolName" bson:"toolName"`
	Description string                 `json:"description" bson:"description"`
	Schema      map[string]interface{} `json:"schema" bson:"schema"`
	ServerName  string                 `json:"serverName" bson:"serverName"`
	CreatedAt   time.Time              `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt" bson:"updatedAt"`
}

// ToolMatch represents a search result from tools discovery
type ToolMatch struct {
	ToolName    string  `json:"toolName"`
	Description string  `json:"description"`
	ServerName  string  `json:"serverName"`
	Score       float64 `json:"score"`
}

// ToolsStorageInterface defines the interface for MCP tools storage operations
type ToolsStorageInterface interface {
	StoreToolMetadata(ctx context.Context, toolName, description string, schema map[string]interface{}, serverName string) error
	SearchTools(ctx context.Context, query string, limit int) ([]*ToolMatch, error)
	GetToolSchema(ctx context.Context, toolName string) (*ToolMetadata, error)
}

// ToolsStorage provides storage interface for MCP tools metadata
type ToolsStorage struct {
	toolsCollection *mongo.Collection
	qdrantClient    QdrantClientInterface
}

// NewToolsStorage creates a new tools storage instance
func NewToolsStorage(db *mongo.Database, qdrantClient QdrantClientInterface) (*ToolsStorage, error) {
	storage := &ToolsStorage{
		toolsCollection: db.Collection("tools"),
		qdrantClient:    qdrantClient,
	}

	// Create indexes
	ctx := context.Background()

	// Index on toolId
	_, err := storage.toolsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "toolId", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create toolId index: %w", err)
	}

	// Index on toolName for fast lookup
	_, err = storage.toolsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "toolName", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create toolName index: %w", err)
	}

	// Index on serverName for filtering
	_, err = storage.toolsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "serverName", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create serverName index: %w", err)
	}

	// Text index for description search fallback
	_, err = storage.toolsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "description", Value: "text"}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create description text index: %w", err)
	}

	return storage, nil
}

// StoreToolMetadata stores tool metadata in both MongoDB and Qdrant
func (s *ToolsStorage) StoreToolMetadata(ctx context.Context, toolName, description string, schema map[string]interface{}, serverName string) error {
	metadata := &ToolMetadata{
		ID:          uuid.New().String(),
		ToolName:    toolName,
		Description: description,
		Schema:      schema,
		ServerName:  serverName,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Store in MongoDB for full schema storage
	filter := bson.M{"toolName": toolName}
	update := bson.M{
		"$set": bson.M{
			"toolId":      metadata.ID,
			"toolName":    metadata.ToolName,
			"description": metadata.Description,
			"schema":      metadata.Schema,
			"serverName":  metadata.ServerName,
			"updatedAt":   metadata.UpdatedAt,
		},
		"$setOnInsert": bson.M{
			"createdAt": metadata.CreatedAt,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := s.toolsCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to store tool metadata in MongoDB: %w", err)
	}

	// Store in Qdrant for semantic search (description + tool name for better matching)
	if s.qdrantClient != nil {
		// Ensure collection exists
		if err := s.qdrantClient.EnsureCollection("mcp-tools", 768); err != nil {
			// Log error but don't fail - MongoDB has the data
			fmt.Printf("Warning: failed to ensure Qdrant collection 'mcp-tools': %v\n", err)
		} else {
			// Create searchable text combining tool name and description
			searchableText := fmt.Sprintf("%s: %s", toolName, description)

			// Store vector point with metadata
			pointMetadata := map[string]interface{}{
				"toolName":   toolName,
				"serverName": serverName,
			}

			if err := s.qdrantClient.StorePoint("mcp-tools", metadata.ID, searchableText, pointMetadata); err != nil {
				// Log error but don't fail - MongoDB has the data
				fmt.Printf("Warning: failed to store tool in Qdrant: %v\n", err)
			}
		}
	}

	return nil
}

// SearchTools searches for tools using semantic similarity via Qdrant
func (s *ToolsStorage) SearchTools(ctx context.Context, query string, limit int) ([]*ToolMatch, error) {
	// Apply limit constraints
	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}

	// Try Qdrant vector search first
	if s.qdrantClient != nil {
		results, err := s.qdrantClient.SearchSimilar("mcp-tools", query, limit)
		if err == nil && len(results) > 0 {
			// Convert QdrantQueryResult to ToolMatch
			matches := make([]*ToolMatch, 0, len(results))
			for _, result := range results {
				match := &ToolMatch{
					ToolName:    getStringFromPayload(result.Entry.Metadata, "toolName"),
					Description: result.Entry.Text,
					ServerName:  getStringFromPayload(result.Entry.Metadata, "serverName"),
					Score:       result.Score,
				}

				// Extract description (text may include "toolName: description" format)
				if match.Description != "" && match.ToolName != "" {
					// Remove "toolName: " prefix if present
					prefix := match.ToolName + ": "
					if len(match.Description) > len(prefix) {
						match.Description = match.Description[len(prefix):]
					}
				}

				matches = append(matches, match)
			}
			return matches, nil
		}
		// Log error but continue to MongoDB fallback
		if err != nil {
			fmt.Printf("Warning: Qdrant search failed, falling back to MongoDB: %v\n", err)
		}
	}

	// Fallback to MongoDB text search
	filter := bson.M{
		"$text": bson.M{"$search": query},
	}

	opts := options.Find().
		SetProjection(bson.D{{Key: "score", Value: bson.D{{Key: "$meta", Value: "textScore"}}}}).
		SetSort(bson.D{{Key: "score", Value: bson.D{{Key: "$meta", Value: "textScore"}}}}).
		SetLimit(int64(limit))

	cursor, err := s.toolsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search tools in MongoDB: %w", err)
	}
	defer cursor.Close(ctx)

	matches := make([]*ToolMatch, 0)
	for cursor.Next(ctx) {
		var metadata ToolMetadata
		if err := cursor.Decode(&metadata); err != nil {
			continue
		}

		match := &ToolMatch{
			ToolName:    metadata.ToolName,
			Description: metadata.Description,
			ServerName:  metadata.ServerName,
			Score:       0.7, // Default score for MongoDB text matches
		}
		matches = append(matches, match)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	// Return empty slice (not nil) if no results
	if matches == nil {
		matches = make([]*ToolMatch, 0)
	}

	return matches, nil
}

// GetToolSchema fetches the full tool schema from MongoDB by exact tool name match
func (s *ToolsStorage) GetToolSchema(ctx context.Context, toolName string) (*ToolMetadata, error) {
	var metadata ToolMetadata

	filter := bson.M{"toolName": toolName}
	err := s.toolsCollection.FindOne(ctx, filter).Decode(&metadata)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("tool not found: %s", toolName)
		}
		return nil, fmt.Errorf("failed to get tool schema: %w", err)
	}

	return &metadata, nil
}
