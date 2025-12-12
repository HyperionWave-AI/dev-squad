package mcp

import (
	"context"
	"fmt"
	"time"

	"hyper/internal/ai-service/tools"
	"hyper/internal/mcp/storage"
)

// QueryKnowledgeTool implements the ToolExecutor interface
type QueryKnowledgeTool struct {
	storage storage.KnowledgeStorage
}

func (t *QueryKnowledgeTool) Name() string {
	return "query_knowledge"
}

func (t *QueryKnowledgeTool) Description() string {
	return "Query the coordinator knowledge base for relevant information. Returns top matches with similarity scores. Limit: 10 results max. Use to find existing solutions, patterns, or context before implementing."
}

func (t *QueryKnowledgeTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"collection": map[string]interface{}{
				"type":        "string",
				"description": "Collection name to query (e.g., 'technical-knowledge', 'task:hyperion://task/human/{taskId}')",
			},
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query text (natural language)",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results (default: 5, max: 10)",
			},
		},
		"required": []string{"collection", "query"},
	}
}

func (t *QueryKnowledgeTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
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
		if limit > 10 {
			limit = 10 // Enforce max limit per task context
		}
	}

	// Query knowledge storage
	results, err := t.storage.Query(collection, query, limit, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query knowledge: %w", err)
	}

	// Convert results to structured format for AI decision-making
	// Step 1: Convert storage results to raw format
	rawResults := make([]map[string]interface{}, len(results))
	for i, result := range results {
		rawResults[i] = map[string]interface{}{
			"id":       result.Entry.ID,
			"filePath": result.Entry.Collection, // Use collection as file path for grouping
			"text":     result.Entry.Text,
			"score":    result.Score,
		}
	}

	// Step 2: Use ResultFormatter to create structured presentation
	formatter := tools.NewResultFormatter()
	startTime := time.Now()
	structuredResponse := formatter.FormatSearchResults(
		rawResults,
		query,
		int64(time.Since(startTime).Milliseconds()),
	)

	// Step 3: Add metadata from original results
	// Attach original metadata to recommendations for reference
	for i := range structuredResponse.Recommendations {
		if i < len(results) {
			// Store original metadata in a metadata field if needed
			// This can be extended for more detailed metadata handling
			_ = results[i].Entry.Metadata
		}
	}

	return structuredResponse, nil
}

// UpsertKnowledgeTool implements the ToolExecutor interface
type UpsertKnowledgeTool struct {
	storage storage.KnowledgeStorage
}

func (t *UpsertKnowledgeTool) Name() string {
	return "coordinator_upsert_knowledge"
}

func (t *UpsertKnowledgeTool) Description() string {
	return "Store knowledge in the coordinator knowledge base. IMPORTANT: MAX 1000 tokens (~750 words, ~4000 characters) per entry. Keep entries focused and granular - ONE concept per entry. Use for storing task context, ADRs, data contracts, and coordination information. For large documents, split into multiple focused entries. Returns entry ID and collection."
}

func (t *UpsertKnowledgeTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"collection": map[string]interface{}{
				"type":        "string",
				"description": "Collection name (e.g., 'task:taskURI', 'adr', 'data-contracts')",
			},
			"text": map[string]interface{}{
				"type":        "string",
				"description": "Content to store (MAX 1000 tokens ≈ 4000 characters)",
			},
			"metadata": map[string]interface{}{
				"type":        "object",
				"description": "Optional metadata (taskId, agentName, timestamp, etc.)",
			},
		},
		"required": []string{"collection", "text"},
	}
}

func (t *UpsertKnowledgeTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	collection, ok := input["collection"].(string)
	if !ok || collection == "" {
		return nil, fmt.Errorf("collection is required and must be a string")
	}

	text, ok := input["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("text is required and must be a string")
	}

	// Validate token count (approximate: 1 token ≈ 4 characters)
	const maxTokens = 1000
	const maxChars = maxTokens * 4 // ~4000 characters

	if len(text) > maxChars {
		estimatedTokens := len(text) / 4
		return nil, fmt.Errorf(
			"Entry too large: ~%d tokens (max: %d tokens ≈ %d characters).\n\n"+
				"📝 This knowledge base is designed for AI retrieval with focused, granular entries.\n\n"+
				"Please split your content into multiple entries, each containing:\n"+
				"• ONE specific concept, pattern, or procedure\n"+
				"• Clear, concise information (aim for 200-800 tokens)\n"+
				"• Descriptive metadata for easy retrieval\n\n"+
				"Example: Instead of storing an entire API documentation, create separate entries for:\n"+
				"- Authentication flow\n"+
				"- Rate limiting rules\n"+
				"- Error handling patterns\n"+
				"- Each endpoint specification",
			estimatedTokens, maxTokens, maxChars,
		)
	}

	var metadata map[string]interface{}
	if m, ok := input["metadata"].(map[string]interface{}); ok {
		metadata = m
	}

	entry, err := t.storage.Upsert(collection, text, metadata, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert knowledge: %w", err)
	}

	return map[string]interface{}{
		"id":         entry.ID,
		"collection": entry.Collection,
		"createdAt":  entry.CreatedAt,
	}, nil
}

// ListCollectionsTool implements the ToolExecutor interface
type ListCollectionsTool struct {
	storage storage.KnowledgeStorage
}

func (t *ListCollectionsTool) Name() string {
	return "knowledge_list_collections"
}

func (t *ListCollectionsTool) Description() string {
	return "List available knowledge collections with entry counts. Use this to discover which collections exist before calling knowledge_find or knowledge_store. Returns collection names with entry counts sorted by popularity."
}

func (t *ListCollectionsTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of collections to return (default: 5)",
			},
		},
	}
}

func (t *ListCollectionsTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	limit := 5
	if l, ok := input["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	stats, err := t.storage.GetPopularCollections(limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular collections: %w", err)
	}

	if stats == nil || len(stats) == 0 {
		return map[string]interface{}{
			"collections":  []interface{}{},
			"message":      "No collections with entries yet",
			"totalDefined": 14,
		}, nil
	}

	return stats, nil
}
