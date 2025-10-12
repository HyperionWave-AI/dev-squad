package mcp

import (
	"context"
	"fmt"
	"strings"

	"hyper/internal/ai-service"
	"hyper/internal/mcp/storage"
)

// QdrantFindTool implements the ToolExecutor interface for searching Qdrant
type QdrantFindTool struct {
	qdrantClient storage.QdrantClientInterface
}

func (t *QdrantFindTool) Name() string {
	return "qdrant_find"
}

func (t *QdrantFindTool) Description() string {
	return "Search for knowledge in Qdrant by semantic similarity. Returns top matches with scores and metadata. Use for discovering patterns, solutions, and related knowledge. Limit: 5 results default (max: 20). If embedding service is down, use query_knowledge tool as fallback."
}

func (t *QdrantFindTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"collection": map[string]interface{}{
				"type":        "string",
				"description": "Collection name to search (e.g., 'technical-knowledge', 'code-patterns')",
			},
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query text (natural language)",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results (default: 5, max: 20)",
			},
		},
		"required": []string{"collection", "query"},
	}
}

func (t *QdrantFindTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Extract and validate required fields
	collection, ok := input["collection"].(string)
	if !ok || collection == "" {
		return nil, fmt.Errorf("collection is required and must be a string")
	}

	query, ok := input["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query is required and must be a string")
	}

	// Extract optional limit
	limit := 5
	if l, ok := input["limit"].(float64); ok && l > 0 {
		limit = int(l)
		if limit > 20 {
			limit = 20
		}
	}

	// Ensure collection exists (768 dimensions for default embedding model)
	if err := t.qdrantClient.EnsureCollection(collection, 768); err != nil {
		// Check if embedding service is unavailable
		errMsg := err.Error()
		if containsAny(errMsg, []string{"connection", "dial", "lookup"}) {
			return nil, fmt.Errorf("Qdrant embedding service unavailable. For task-specific knowledge, use query_knowledge tool with task URI instead. Original error: %w", err)
		}
		return nil, fmt.Errorf("failed to ensure collection exists: %w (try query_knowledge as fallback)", err)
	}

	// Search for similar entries
	results, err := t.qdrantClient.SearchSimilar(collection, query, limit)
	if err != nil {
		// Check if search service is unavailable
		errMsg := err.Error()
		if containsAny(errMsg, []string{"connection", "dial", "lookup", "timeout"}) {
			return nil, fmt.Errorf("Qdrant search unavailable. Use query_knowledge tool as fallback for task-specific knowledge. Original error: %w", err)
		}
		return nil, fmt.Errorf("search failed: %w (try query_knowledge as alternative)", err)
	}

	// Format results
	type QdrantResult struct {
		Text     string                 `json:"text"`
		Score    float64                `json:"score"`
		Metadata map[string]interface{} `json:"metadata,omitempty"`
	}

	formattedResults := make([]QdrantResult, len(results))
	for i, result := range results {
		formattedResults[i] = QdrantResult{
			Text:     result.Entry.Text,
			Score:    result.Score,
			Metadata: result.Entry.Metadata,
		}
	}

	// Return empty array if no results (not an error)
	return formattedResults, nil
}

// QdrantStoreTool implements the ToolExecutor interface for storing in Qdrant
type QdrantStoreTool struct {
	qdrantClient storage.QdrantClientInterface
}

func (t *QdrantStoreTool) Name() string {
	return "qdrant_store"
}

func (t *QdrantStoreTool) Description() string {
	return "Store knowledge in Qdrant with automatic embedding generation. Returns storage confirmation. Use to persist reusable patterns, solutions, and learnings for semantic search. If embedding service is down, use coordinator upsert_knowledge tool for MongoDB storage (no semantic search)."
}

func (t *QdrantStoreTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"collection": map[string]interface{}{
				"type":        "string",
				"description": "Collection name (e.g., 'technical-knowledge', 'code-patterns')",
			},
			"information": map[string]interface{}{
				"type":        "string",
				"description": "Text content to store (will be embedded automatically)",
			},
			"metadata": map[string]interface{}{
				"type":        "object",
				"description": "Optional metadata to attach (e.g., tags, source, author)",
			},
		},
		"required": []string{"collection", "information"},
	}
}

func (t *QdrantStoreTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Extract and validate required fields
	collection, ok := input["collection"].(string)
	if !ok || collection == "" {
		return nil, fmt.Errorf("collection is required and must be a string")
	}

	information, ok := input["information"].(string)
	if !ok || information == "" {
		return nil, fmt.Errorf("information is required and must be a string")
	}

	// Extract optional metadata
	metadata, _ := input["metadata"].(map[string]interface{})

	// Ensure collection exists (768 dimensions for default embedding model)
	if err := t.qdrantClient.EnsureCollection(collection, 768); err != nil {
		// Check if embedding service is unavailable
		errMsg := err.Error()
		if containsAny(errMsg, []string{"connection", "dial", "lookup"}) {
			return nil, fmt.Errorf("Qdrant embedding service unavailable. Cannot store vector embeddings. Use coordinator upsert_knowledge tool to store in MongoDB (metadata only, no semantic search). Original error: %w", err)
		}
		return nil, fmt.Errorf("failed to ensure collection exists: %w (try coordinator upsert_knowledge as fallback)", err)
	}

	// Generate ID
	id := storage.GenerateID()

	// Store point with embedding
	if err := t.qdrantClient.StorePoint(collection, id, information, metadata); err != nil {
		// Check if storage service is unavailable
		errMsg := err.Error()
		if containsAny(errMsg, []string{"connection", "dial", "lookup", "timeout"}) {
			return nil, fmt.Errorf("Qdrant storage unavailable. Use coordinator upsert_knowledge tool to store in MongoDB instead. Original error: %w", err)
		}
		return nil, fmt.Errorf("failed to store knowledge: %w (try coordinator upsert_knowledge as alternative)", err)
	}

	// Format response
	return map[string]interface{}{
		"id":         id,
		"collection": collection,
		"status":     "stored",
	}, nil
}

// containsAny checks if a string contains any of the substrings
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// RegisterQdrantTools registers all Qdrant tools with the tool registry
func RegisterQdrantTools(registry *aiservice.ToolRegistry, qdrantClient storage.QdrantClientInterface) error {
	tools := []aiservice.ToolExecutor{
		&QdrantFindTool{qdrantClient: qdrantClient},
		&QdrantStoreTool{qdrantClient: qdrantClient},
	}

	for _, tool := range tools {
		if err := registry.Register(tool); err != nil {
			return fmt.Errorf("failed to register %s: %w", tool.Name(), err)
		}
	}

	return nil
}
