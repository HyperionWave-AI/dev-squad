package storage

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
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

// ResourceMetadata represents metadata about an MCP resource
type ResourceMetadata struct {
	ID          string                 `json:"id" bson:"resourceId"`
	URI         string                 `json:"uri" bson:"uri"`
	Name        string                 `json:"name" bson:"name"`
	Description string                 `json:"description" bson:"description"`
	MimeType    string                 `json:"mimeType,omitempty" bson:"mimeType,omitempty"`
	ServerName  string                 `json:"serverName" bson:"serverName"`
	CreatedAt   time.Time              `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt" bson:"updatedAt"`
}

// PromptMetadata represents metadata about an MCP prompt
type PromptMetadata struct {
	ID          string                 `json:"id" bson:"promptId"`
	Name        string                 `json:"name" bson:"name"`
	Description string                 `json:"description" bson:"description"`
	Arguments   []map[string]interface{} `json:"arguments,omitempty" bson:"arguments,omitempty"`
	ServerName  string                 `json:"serverName" bson:"serverName"`
	CreatedAt   time.Time              `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt" bson:"updatedAt"`
}

// ServerMetadata represents metadata about an MCP server
type ServerMetadata struct {
	ServerName    string                 `json:"serverName" bson:"serverName"`
	ServerURL     string                 `json:"serverUrl" bson:"serverUrl"`
	Description   string                 `json:"description" bson:"description"`
	Headers       map[string]interface{} `json:"headers,omitempty" bson:"headers,omitempty"`
	ToolCount     int                    `json:"toolCount" bson:"toolCount"`
	ResourceCount int                    `json:"resourceCount" bson:"resourceCount"`
	PromptCount   int                    `json:"promptCount" bson:"promptCount"`
	CreatedAt     time.Time              `json:"createdAt" bson:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt" bson:"updatedAt"`
}

// ToolsStorageInterface defines the interface for MCP tools storage operations
type ToolsStorageInterface interface {
	// Tool operations
	StoreToolMetadata(ctx context.Context, toolName, description string, schema map[string]interface{}, serverName string) error
	SearchTools(ctx context.Context, query string, limit int) ([]*ToolMatch, error)
	GetToolSchema(ctx context.Context, toolName string) (*ToolMetadata, error)

	// Resource operations
	StoreResourceMetadata(ctx context.Context, uri, name, description, mimeType, serverName string) error
	GetServerResources(ctx context.Context, serverName string) ([]*ResourceMetadata, error)

	// Prompt operations
	StorePromptMetadata(ctx context.Context, name, description string, arguments []map[string]interface{}, serverName string) error
	GetServerPrompts(ctx context.Context, serverName string) ([]*PromptMetadata, error)

	// Server management
	AddServer(ctx context.Context, serverName, serverURL, description string, headers map[string]interface{}) error
	UpdateServer(ctx context.Context, serverName, serverURL, description string, headers map[string]interface{}) error
	UpdateServerCounts(ctx context.Context, serverName string, toolCount, resourceCount, promptCount int) error
	RemoveServer(ctx context.Context, serverName string) error
	GetServer(ctx context.Context, serverName string) (*ServerMetadata, error)
	GetServerTools(ctx context.Context, serverName string) ([]*ToolMetadata, error)
	ListServers(ctx context.Context) ([]*ServerMetadata, error)
	RemoveServerTools(ctx context.Context, serverName string) error
	RemoveServerResources(ctx context.Context, serverName string) error
	RemoveServerPrompts(ctx context.Context, serverName string) error
}

// ToolsStorage provides storage interface for MCP tools metadata
type ToolsStorage struct {
	toolsCollection     *mongo.Collection
	resourcesCollection *mongo.Collection
	promptsCollection   *mongo.Collection
	serversCollection   *mongo.Collection
	qdrantClient        QdrantClientInterface
	toolsCollectionName string
	logger              *zap.Logger
}

// NewToolsStorage creates a new tools storage instance
func NewToolsStorage(db *mongo.Database, qdrantClient QdrantClientInterface, logger *zap.Logger) (*ToolsStorage, error) {
	// Read base collection name from environment variable with default fallback
	baseCollectionName := os.Getenv("MCP_TOOLS_COLLECTION_NAME")
	if baseCollectionName == "" {
		baseCollectionName = "mcp-tools"
	}

	// Get vector dimensions from qdrant client to create dimension-specific collection name
	// This prevents dimension mismatch errors when switching embedding providers
	vectorDim := qdrantClient.GetDimensions() // Get from shared embedding client

	// Append dimension to collection name (e.g., "mcp-tools_1024")
	collectionName := fmt.Sprintf("%s_%d", baseCollectionName, vectorDim)

	logger.Info("Initializing MCP tools storage",
		zap.String("collection", collectionName),
		zap.Int("vectorDimension", vectorDim))

	storage := &ToolsStorage{
		toolsCollection:     db.Collection("tools"),
		resourcesCollection: db.Collection("mcp_resources"),
		promptsCollection:   db.Collection("mcp_prompts"),
		serversCollection:   db.Collection("mcp_servers"),
		qdrantClient:        qdrantClient,
		toolsCollectionName: collectionName,
		logger:              logger,
	}

	// Create indexes
	ctx := context.Background()

	// Tools collection indexes
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

	// Resources collection indexes
	// Index on resourceId (unique)
	_, err = storage.resourcesCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "resourceId", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create resourceId index: %w", err)
	}

	// Index on uri for fast lookup
	_, err = storage.resourcesCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "uri", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create uri index: %w", err)
	}

	// Index on serverName for filtering resources by server
	_, err = storage.resourcesCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "serverName", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create resource serverName index: %w", err)
	}

	// Prompts collection indexes
	// Index on promptId (unique)
	_, err = storage.promptsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "promptId", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create promptId index: %w", err)
	}

	// Index on name for fast lookup
	_, err = storage.promptsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "name", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt name index: %w", err)
	}

	// Index on serverName for filtering prompts by server
	_, err = storage.promptsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "serverName", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt serverName index: %w", err)
	}

	// Servers collection indexes
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
		vectorDim := s.qdrantClient.GetDimensions() // Get from shared embedding client

		// Ensure collection exists with correct dimensions
		if err := s.qdrantClient.EnsureCollection(s.toolsCollectionName, vectorDim); err != nil {
			// Log error but don't fail - MongoDB has the data
			s.logger.Warn("Failed to ensure Qdrant collection",
				zap.String("collection", s.toolsCollectionName),
				zap.Int("vectorDim", vectorDim),
				zap.Error(err))
		} else {
			// Create searchable text combining tool name and description
			searchableText := fmt.Sprintf("%s: %s", toolName, description)

			// Store vector point with metadata
			pointMetadata := map[string]interface{}{
				"toolName":   toolName,
				"serverName": serverName,
			}

			if err := s.qdrantClient.StorePoint(s.toolsCollectionName, metadata.ID, searchableText, pointMetadata); err != nil {
				// Log error but don't fail - MongoDB has the data
				s.logger.Warn("Failed to store tool in Qdrant",
					zap.String("collection", s.toolsCollectionName),
					zap.String("toolName", toolName),
					zap.String("toolId", metadata.ID),
					zap.Error(err))
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
		results, err := s.qdrantClient.SearchSimilar(s.toolsCollectionName, query, limit)
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
			s.logger.Warn("Qdrant search failed, falling back to MongoDB",
				zap.String("collection", s.toolsCollectionName),
				zap.String("query", query),
				zap.Int("limit", limit),
				zap.Error(err))
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
func (s *ToolsStorage) AddServer(ctx context.Context, serverName, serverURL, description string, headers map[string]interface{}) error {
	server := &ServerMetadata{
		ServerName:    serverName,
		ServerURL:     serverURL,
		Description:   description,
		Headers:       headers,
		ToolCount:     0,
		ResourceCount: 0,
		PromptCount:   0,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	filter := bson.M{"serverName": serverName}
	update := bson.M{
		"$set": bson.M{
			"serverName":  server.ServerName,
			"serverUrl":   server.ServerURL,
			"description": server.Description,
			"headers":     server.Headers,
			"updatedAt":   server.UpdatedAt,
		},
		"$setOnInsert": bson.M{
			"toolCount":     0,
			"resourceCount": 0,
			"promptCount":   0,
			"createdAt":     server.CreatedAt,
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
			if err := s.qdrantClient.DeletePoint(s.toolsCollectionName, toolID); err != nil {
				// Log error but don't fail
				s.logger.Warn("Failed to delete tool from Qdrant",
					zap.String("collection", s.toolsCollectionName),
					zap.String("toolId", toolID),
					zap.String("serverName", serverName),
					zap.Error(err))
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
		s.logger.Warn("Failed to update server tool count",
			zap.String("serverName", serverName),
			zap.Error(err))
	}

	s.logger.Info("Removed tools for server",
		zap.String("serverName", serverName),
		zap.Int64("toolCount", result.DeletedCount))
	return nil
}

// UpdateServer updates an existing MCP server's metadata
func (s *ToolsStorage) UpdateServer(ctx context.Context, serverName, serverURL, description string, headers map[string]interface{}) error {
	// Check if server exists
	_, err := s.GetServer(ctx, serverName)
	if err != nil {
		return fmt.Errorf("server not found: %s", serverName)
	}

	// Update server metadata
	filter := bson.M{"serverName": serverName}
	update := bson.M{
		"$set": bson.M{
			"serverUrl":   serverURL,
			"description": description,
			"headers":     headers,
			"updatedAt":   time.Now().UTC(),
		},
	}

	_, err = s.serversCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update server: %w", err)
	}

	s.logger.Info("Server updated successfully",
		zap.String("serverName", serverName),
		zap.String("serverUrl", serverURL))

	return nil
}

// UpdateServerCounts updates the tool, resource, and prompt counts for an MCP server
func (s *ToolsStorage) UpdateServerCounts(ctx context.Context, serverName string, toolCount, resourceCount, promptCount int) error {
	// Check if server exists
	_, err := s.GetServer(ctx, serverName)
	if err != nil {
		return fmt.Errorf("server not found: %s", serverName)
	}

	// Update server counts
	filter := bson.M{"serverName": serverName}
	update := bson.M{
		"$set": bson.M{
			"toolCount":     toolCount,
			"resourceCount": resourceCount,
			"promptCount":   promptCount,
			"updatedAt":     time.Now().UTC(),
		},
	}

	_, err = s.serversCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update server counts: %w", err)
	}

	s.logger.Info("Server counts updated successfully",
		zap.String("serverName", serverName),
		zap.Int("toolCount", toolCount),
		zap.Int("resourceCount", resourceCount),
		zap.Int("promptCount", promptCount))

	return nil
}

// GetServerTools retrieves all tools for a specific server
func (s *ToolsStorage) GetServerTools(ctx context.Context, serverName string) ([]*ToolMetadata, error) {
	filter := bson.M{"serverName": serverName}
	cursor, err := s.toolsCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find server tools: %w", err)
	}
	defer cursor.Close(ctx)

	tools := make([]*ToolMetadata, 0)
	for cursor.Next(ctx) {
		var tool ToolMetadata
		if err := cursor.Decode(&tool); err != nil {
			s.logger.Warn("Failed to decode tool metadata",
				zap.String("serverName", serverName),
				zap.Error(err))
			continue
		}
		tools = append(tools, &tool)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return tools, nil
}

// StoreResourceMetadata stores resource metadata in MongoDB
func (s *ToolsStorage) StoreResourceMetadata(ctx context.Context, uri, name, description, mimeType, serverName string) error {
	metadata := &ResourceMetadata{
		ID:          uuid.New().String(),
		URI:         uri,
		Name:        name,
		Description: description,
		MimeType:    mimeType,
		ServerName:  serverName,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Store in MongoDB - upsert by URI to avoid duplicates
	filter := bson.M{"uri": uri, "serverName": serverName}
	update := bson.M{
		"$set": bson.M{
			"resourceId":  metadata.ID,
			"uri":         metadata.URI,
			"name":        metadata.Name,
			"description": metadata.Description,
			"mimeType":    metadata.MimeType,
			"serverName":  metadata.ServerName,
			"updatedAt":   metadata.UpdatedAt,
		},
		"$setOnInsert": bson.M{
			"createdAt": metadata.CreatedAt,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := s.resourcesCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to store resource metadata in MongoDB: %w", err)
	}

	return nil
}

// GetServerResources retrieves all resources for a specific server
func (s *ToolsStorage) GetServerResources(ctx context.Context, serverName string) ([]*ResourceMetadata, error) {
	filter := bson.M{"serverName": serverName}
	cursor, err := s.resourcesCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find server resources: %w", err)
	}
	defer cursor.Close(ctx)

	resources := make([]*ResourceMetadata, 0)
	for cursor.Next(ctx) {
		var resource ResourceMetadata
		if err := cursor.Decode(&resource); err != nil {
			s.logger.Warn("Failed to decode resource metadata",
				zap.String("serverName", serverName),
				zap.Error(err))
			continue
		}
		resources = append(resources, &resource)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return resources, nil
}

// RemoveServerResources removes all resources associated with a server
func (s *ToolsStorage) RemoveServerResources(ctx context.Context, serverName string) error {
	filter := bson.M{"serverName": serverName}
	result, err := s.resourcesCollection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to remove server resources from MongoDB: %w", err)
	}

	s.logger.Info("Removed resources for server",
		zap.String("serverName", serverName),
		zap.Int64("resourceCount", result.DeletedCount))
	return nil
}

// StorePromptMetadata stores prompt metadata in MongoDB
func (s *ToolsStorage) StorePromptMetadata(ctx context.Context, name, description string, arguments []map[string]interface{}, serverName string) error {
	metadata := &PromptMetadata{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Arguments:   arguments,
		ServerName:  serverName,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Store in MongoDB - upsert by name and serverName to avoid duplicates
	filter := bson.M{"name": name, "serverName": serverName}
	update := bson.M{
		"$set": bson.M{
			"promptId":    metadata.ID,
			"name":        metadata.Name,
			"description": metadata.Description,
			"arguments":   metadata.Arguments,
			"serverName":  metadata.ServerName,
			"updatedAt":   metadata.UpdatedAt,
		},
		"$setOnInsert": bson.M{
			"createdAt": metadata.CreatedAt,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := s.promptsCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to store prompt metadata in MongoDB: %w", err)
	}

	return nil
}

// GetServerPrompts retrieves all prompts for a specific server
func (s *ToolsStorage) GetServerPrompts(ctx context.Context, serverName string) ([]*PromptMetadata, error) {
	filter := bson.M{"serverName": serverName}
	cursor, err := s.promptsCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find server prompts: %w", err)
	}
	defer cursor.Close(ctx)

	prompts := make([]*PromptMetadata, 0)
	for cursor.Next(ctx) {
		var prompt PromptMetadata
		if err := cursor.Decode(&prompt); err != nil {
			s.logger.Warn("Failed to decode prompt metadata",
				zap.String("serverName", serverName),
				zap.Error(err))
			continue
		}
		prompts = append(prompts, &prompt)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return prompts, nil
}

// RemoveServerPrompts removes all prompts associated with a server
func (s *ToolsStorage) RemoveServerPrompts(ctx context.Context, serverName string) error {
	filter := bson.M{"serverName": serverName}
	result, err := s.promptsCollection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to remove server prompts from MongoDB: %w", err)
	}

	s.logger.Info("Removed prompts for server",
		zap.String("serverName", serverName),
		zap.Int64("promptCount", result.DeletedCount))
	return nil
}
