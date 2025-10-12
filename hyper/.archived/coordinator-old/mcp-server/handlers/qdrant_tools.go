package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"hyperion-coordinator-mcp/storage"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// QdrantToolHandler manages MCP Qdrant tool operations
type QdrantToolHandler struct {
	qdrantClient storage.QdrantClientInterface
}

// NewQdrantToolHandler creates a new Qdrant tool handler
func NewQdrantToolHandler(client storage.QdrantClientInterface) *QdrantToolHandler {
	return &QdrantToolHandler{
		qdrantClient: client,
	}
}

// RegisterQdrantTools registers Qdrant tools with the MCP server
func (h *QdrantToolHandler) RegisterQdrantTools(server *mcp.Server) error {
	// Register knowledge_find tool
	if err := h.registerKnowledgeFind(server); err != nil {
		return fmt.Errorf("failed to register knowledge_find tool: %w", err)
	}

	// Register knowledge_store tool
	if err := h.registerKnowledgeStore(server); err != nil {
		return fmt.Errorf("failed to register knowledge_store tool: %w", err)
	}

	return nil
}

// registerKnowledgeFind registers the knowledge_find tool
func (h *QdrantToolHandler) registerKnowledgeFind(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "knowledge_find",
		Description: "Search for knowledge by semantic similarity. Returns top N results with scores and metadata. Supports full or chunked text retrieval.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"collectionName": {
					Type:        "string",
					Description: "Collection name to search (e.g., 'technical-knowledge', 'code-patterns')",
				},
				"query": {
					Type:        "string",
					Description: "Search query text",
				},
				"limit": {
					Type:        "number",
					Description: "Maximum number of results (default: 5, max: 20)",
				},
				"retrieveMode": {
					Type:        "string",
					Description: "Content retrieval mode: 'full' (entire document) or 'chunk' (partial content). Default: 'full'",
					Enum:        []interface{}{"full", "chunk"},
				},
				"chunkSize": {
					Type:        "number",
					Description: "Maximum characters to return per result when retrieveMode is 'chunk' (default: 500, min: 100, max: 2000)",
				},
			},
			Required: []string{"collectionName", "query"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleQdrantFind(args)
		return result, err
	})

	return nil
}

// registerKnowledgeStore registers the knowledge_store tool
func (h *QdrantToolHandler) registerKnowledgeStore(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "knowledge_store",
		Description: "Store knowledge with automatic embedding generation. Returns storage confirmation with ID and collection.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"collectionName": {
					Type:        "string",
					Description: "Collection name (e.g., 'technical-knowledge', 'code-patterns')",
				},
				"information": {
					Type:        "string",
					Description: "Text content to store",
				},
				"metadata": {
					Type:        "object",
					Description: "Optional metadata to attach (e.g., tags, source, author)",
				},
			},
			Required: []string{"collectionName", "information"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleQdrantStore(args)
		return result, err
	})

	return nil
}

// handleQdrantFind handles the qdrant_find tool call
func (h *QdrantToolHandler) handleQdrantFind(args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	// Extract collectionName (required)
	collectionName, ok := args["collectionName"].(string)
	if !ok || collectionName == "" {
		return createErrorResult("collectionName parameter is required and must be a non-empty string"), nil, nil
	}

	// Extract query (required)
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return createErrorResult("query parameter is required and must be a non-empty string"), nil, nil
	}

	// Extract limit (optional, default 5, max 20)
	limit := 5
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
		if limit > 20 {
			limit = 20
		}
		if limit < 1 {
			limit = 1
		}
	}

	// Extract retrieveMode (optional, default "full")
	retrieveMode := "full"
	if mode, ok := args["retrieveMode"].(string); ok {
		if mode == "chunk" || mode == "full" {
			retrieveMode = mode
		}
	}

	// Extract chunkSize (optional, default 500, min 100, max 2000)
	chunkSize := 500
	if size, ok := args["chunkSize"].(float64); ok {
		chunkSize = int(size)
		if chunkSize < 100 {
			chunkSize = 100
		}
		if chunkSize > 2000 {
			chunkSize = 2000
		}
	}

	// Ensure collection exists (with 768 dimensions for TEI embeddings)
	if err := h.qdrantClient.EnsureCollection(collectionName, 768); err != nil {
		// Provide helpful recovery guidance based on error type
		errMsg := err.Error()
		if strings.Contains(errMsg, "connection") || strings.Contains(errMsg, "dial") || strings.Contains(errMsg, "lookup") {
			return createErrorResult(fmt.Sprintf("Qdrant embedding service unavailable. For task-specific knowledge, use coordinator_query_knowledge with task URI (e.g., collection='task:hyperion://task/human/{taskId}'). Original error: %s", errMsg)), nil, nil
		}
		return createErrorResult(fmt.Sprintf("Failed to ensure collection exists: %s. Try coordinator_query_knowledge as fallback.", errMsg)), nil, nil
	}

	// Search for similar entries
	results, err := h.qdrantClient.SearchSimilar(collectionName, query, limit)
	if err != nil {
		// Provide helpful recovery guidance based on error type
		errMsg := err.Error()
		if strings.Contains(errMsg, "connection") || strings.Contains(errMsg, "dial") || strings.Contains(errMsg, "lookup") || strings.Contains(errMsg, "timeout") {
			return createErrorResult(fmt.Sprintf("Qdrant search unavailable. Use coordinator_query_knowledge as fallback for task-specific knowledge. Original error: %s", errMsg)), nil, nil
		}
		return createErrorResult(fmt.Sprintf("Search failed: %s. Try coordinator_query_knowledge as alternative.", errMsg)), nil, nil
	}

	if len(results) == 0 {
		// Return empty JSON array for UI compatibility
		emptyArrayJSON, _ := json.Marshal([]interface{}{})
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(emptyArrayJSON)},
			},
		}, results, nil
	}

	// Format results with chunking if requested
	resultText := fmt.Sprintf("Found %d results (retrieveMode: %s):\n\n", len(results), retrieveMode)
	for i, result := range results {
		resultText += fmt.Sprintf("Result %d (Score: %.2f)\n", i+1, result.Score)

		// Apply chunking logic based on retrieveMode
		text := result.Entry.Text
		if retrieveMode == "chunk" && len(text) > chunkSize {
			text = text[:chunkSize] + "..."
			resultText += fmt.Sprintf("Text (truncated to %d chars): %s\n", chunkSize, text)
		} else if retrieveMode == "full" {
			resultText += fmt.Sprintf("Text: %s\n", text)
		} else {
			// Default fallback
			resultText += fmt.Sprintf("Text: %s\n", text)
		}

		// Show metadata if present
		if len(result.Entry.Metadata) > 0 {
			metadataJSON, _ := json.MarshalIndent(result.Entry.Metadata, "", "  ")
			resultText += fmt.Sprintf("Metadata: %s\n", string(metadataJSON))
		}

		resultText += "\n---\n\n"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, results, nil
}

// handleQdrantStore handles the qdrant_store tool call
func (h *QdrantToolHandler) handleQdrantStore(args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	// Extract collectionName (required)
	collectionName, ok := args["collectionName"].(string)
	if !ok || collectionName == "" {
		return createErrorResult("collectionName parameter is required and must be a non-empty string"), nil, nil
	}

	// Validate collection name format
	if len(collectionName) < 1 {
		return createErrorResult("collectionName must be at least 1 character long"), nil, nil
	}

	// Extract information (required)
	information, ok := args["information"].(string)
	if !ok || information == "" {
		return createErrorResult("information parameter is required and must be a non-empty string"), nil, nil
	}

	// Extract metadata (optional)
	var metadata map[string]interface{}
	if m, ok := args["metadata"].(map[string]interface{}); ok {
		metadata = m
	}

	// Ensure collection exists (with 768 dimensions for TEI embeddings)
	if err := h.qdrantClient.EnsureCollection(collectionName, 768); err != nil {
		// Provide helpful recovery guidance based on error type
		errMsg := err.Error()
		if strings.Contains(errMsg, "connection") || strings.Contains(errMsg, "dial") || strings.Contains(errMsg, "lookup") {
			return createErrorResult(fmt.Sprintf("Qdrant embedding service unavailable. Cannot store vector embeddings. Use coordinator_upsert_knowledge to store in MongoDB (metadata only, no semantic search). Original error: %s", errMsg)), nil, nil
		}
		return createErrorResult(fmt.Sprintf("Failed to ensure collection exists: %s. Try coordinator_upsert_knowledge as fallback.", errMsg)), nil, nil
	}

	// Generate ID
	id := storage.GenerateID()

	// Store point with embedding
	if err := h.qdrantClient.StorePoint(collectionName, id, information, metadata); err != nil {
		// Provide helpful recovery guidance based on error type
		errMsg := err.Error()
		if strings.Contains(errMsg, "connection") || strings.Contains(errMsg, "dial") || strings.Contains(errMsg, "lookup") || strings.Contains(errMsg, "timeout") {
			return createErrorResult(fmt.Sprintf("Qdrant storage unavailable. Use coordinator_upsert_knowledge to store in MongoDB instead. Original error: %s", errMsg)), nil, nil
		}
		return createErrorResult(fmt.Sprintf("Failed to store knowledge: %s. Try coordinator_upsert_knowledge as alternative.", errMsg)), nil, nil
	}

	resultText := fmt.Sprintf("✓ Knowledge stored in Qdrant\n\nID: %s\nCollection: %s\nVector dimensions: 768 (TEI nomic-embed-text-v1.5)",
		id, collectionName)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, map[string]interface{}{
		"id":         id,
		"collection": collectionName,
	}, nil
}

// extractArguments safely extracts arguments from CallToolRequest
func extractArguments(req *mcp.CallToolRequest) (map[string]interface{}, error) {
	if req.Params.Arguments == nil || len(req.Params.Arguments) == 0 {
		return make(map[string]interface{}), nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(req.Params.Arguments, &result); err != nil {
		return nil, fmt.Errorf("arguments must be a valid JSON object: %w", err)
	}

	return result, nil
}
