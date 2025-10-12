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

// ServerMetadata represents metadata about an MCP server
type ServerMetadata struct {
	ServerName  string    `json:"serverName" bson:"serverName"`
	ServerURL   string    `json:"serverUrl" bson:"serverUrl"`
	Description string    `json:"description" bson:"description"`
	ToolCount   int       `json:"toolCount" bson:"toolCount"`
	CreatedAt   time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt" bson:"updatedAt"`
}

// ToolsStorageInterface defines the interface for MCP tools storage operations
type ToolsStorageInterface interface {
	StoreToolMetadata(ctx context.Context, toolName, description string, schema map[string]interface{}, serverName string) error
	SearchTools(ctx context.Context, query string, limit int) ([]*ToolMatch, error)
	GetToolSchema(ctx context.Context, toolName string) (*ToolMetadata, error)

	// Server management
	AddServer(ctx context.Context, serverName, serverURL, description string) error
	RemoveServer(ctx context.Context, serverName string) error
	GetServer(ctx context.Context, serverName string) (*ServerMetadata, error)
	ListServers(ctx context.Context) ([]*ServerMetadata, error)
	RemoveServerTools(ctx context.Context, serverName string) error
}

// ToolsStorage provides storage interface for MCP tools metadata
type ToolsStorage struct {
	toolsCollection   *mongo.Collection
	serversCollection *mongo.Collection
	qdrantClient      QdrantClientInterface
}

// NewToolsStorage creates a new tools storage instance
func NewToolsStorage(db *mongo.Database, qdrantClient QdrantClientInterface) (*ToolsStorage, error) {
	storage := &ToolsStorage{
		toolsCollection:   db.Collection("tools"),
		serversCollection: db.Collection("mcp_servers"),
		qdrantClient:      qdrantClient,
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

	// Create indexes for servers collection
	// Index on serverName (unique)
	_, err = storage.serversCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "serverName", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create serverName index: %w", err)
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
		// Get vector dimensions from qdrant client (uses configured embedding client dimensions)
		// This ensures we use the correct dimensions (Ollama:768, OpenAI:1536, Voyage:1024, etc.)
		vectorDim := 768 // Default fallback
		if qdrantClientTyped, ok := s.qdrantClient.(*QdrantClient); ok {
			vectorDim = qdrantClientTyped.vectorDimension
		}

		// Ensure collection exists with correct dimensions
		if err := s.qdrantClient.EnsureCollection("mcp-tools", vectorDim); err != nil {
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

// AddServer adds a new MCP server to the registry
func (s *ToolsStorage) AddServer(ctx context.Context, serverName, serverURL, description string) error {
	server := &ServerMetadata{
		ServerName:  serverName,
		ServerURL:   serverURL,
		Description: description,
		ToolCount:   0,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	filter := bson.M{"serverName": serverName}
	update := bson.M{
		"$set": bson.M{
			"serverName":  server.ServerName,
			"serverUrl":   server.ServerURL,
			"description": server.Description,
			"updatedAt":   server.UpdatedAt,
		},
		"$setOnInsert": bson.M{
			"toolCount": 0,
			"createdAt": server.CreatedAt,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := s.serversCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to add server: %w", err)
	}

	return nil
}

// RemoveServer removes an MCP server from the registry
func (s *ToolsStorage) RemoveServer(ctx context.Context, serverName string) error {
	filter := bson.M{"serverName": serverName}
	result, err := s.serversCollection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to remove server: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("server not found: %s", serverName)
	}

	return nil
}

// GetServer retrieves server metadata
func (s *ToolsStorage) GetServer(ctx context.Context, serverName string) (*ServerMetadata, error) {
	var server ServerMetadata

	filter := bson.M{"serverName": serverName}
	err := s.serversCollection.FindOne(ctx, filter).Decode(&server)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("server not found: %s", serverName)
		}
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	return &server, nil
}

// ListServers lists all registered MCP servers
func (s *ToolsStorage) ListServers(ctx context.Context) ([]*ServerMetadata, error) {
	cursor, err := s.serversCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}
	defer cursor.Close(ctx)

	servers := make([]*ServerMetadata, 0)
	for cursor.Next(ctx) {
		var server ServerMetadata
		if err := cursor.Decode(&server); err != nil {
			continue
		}
		servers = append(servers, &server)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return servers, nil
}

// RemoveServerTools removes all tools associated with a server
func (s *ToolsStorage) RemoveServerTools(ctx context.Context, serverName string) error {
	// First, get all tool IDs for this server to remove from Qdrant
	filter := bson.M{"serverName": serverName}
	cursor, err := s.toolsCollection.Find(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to find server tools: %w", err)
	}
	defer cursor.Close(ctx)

	// Collect tool IDs for Qdrant deletion
	var toolIDs []string
	for cursor.Next(ctx) {
		var metadata ToolMetadata
		if err := cursor.Decode(&metadata); err != nil {
			continue
		}
		toolIDs = append(toolIDs, metadata.ID)
	}

	// Delete from MongoDB
	result, err := s.toolsCollection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to remove server tools from MongoDB: %w", err)
	}

	// Delete from Qdrant
	if s.qdrantClient != nil && len(toolIDs) > 0 {
		for _, toolID := range toolIDs {
			if err := s.qdrantClient.DeletePoint("mcp-tools", toolID); err != nil {
				// Log error but don't fail
				fmt.Printf("Warning: failed to delete tool %s from Qdrant: %v\n", toolID, err)
			}
		}
	}

	// Update server tool count
	update := bson.M{
		"$set": bson.M{
			"toolCount": 0,
			"updatedAt": time.Now().UTC(),
		},
	}
	_, err = s.serversCollection.UpdateOne(ctx, bson.M{"serverName": serverName}, update)
	if err != nil {
		// Log error but don't fail - tools are deleted
		fmt.Printf("Warning: failed to update server tool count: %v\n", err)
	}

	fmt.Printf("Removed %d tools for server %s\n", result.DeletedCount, serverName)
	return nil
}
