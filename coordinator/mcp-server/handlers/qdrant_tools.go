package handlers

import (
	"context"
	"encoding/json"
	"fmt"

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
	// Register qdrant_find tool
	if err := h.registerQdrantFind(server); err != nil {
		return fmt.Errorf("failed to register qdrant_find tool: %w", err)
	}

	// Register qdrant_store tool
	if err := h.registerQdrantStore(server); err != nil {
		return fmt.Errorf("failed to register qdrant_store tool: %w", err)
	}

	return nil
}

// registerQdrantFind registers the qdrant_find tool
func (h *QdrantToolHandler) registerQdrantFind(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "qdrant_find",
		Description: "Search for knowledge in Qdrant by semantic similarity. Returns top N results with scores and metadata.",
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

// registerQdrantStore registers the qdrant_store tool
func (h *QdrantToolHandler) registerQdrantStore(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "qdrant_store",
		Description: "Store knowledge in Qdrant with automatic embedding generation. Returns storage confirmation with ID and collection.",
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

	// Ensure collection exists (with 1536 dimensions for OpenAI embeddings)
	if err := h.qdrantClient.EnsureCollection(collectionName, 1536); err != nil {
		return createErrorResult(fmt.Sprintf("failed to ensure collection exists: %s", err.Error())), nil, nil
	}

	// Search for similar entries
	results, err := h.qdrantClient.SearchSimilar(collectionName, query, limit)
	if err != nil {
		return createErrorResult(fmt.Sprintf("search failed: %s", err.Error())), nil, nil
	}

	if len(results) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("No results found in collection '%s' for query: %s", collectionName, query)},
			},
		}, results, nil
	}

	// Format results
	resultText := fmt.Sprintf("Found %d results:\n\n", len(results))
	for i, result := range results {
		resultText += fmt.Sprintf("Result %d (Score: %.2f)\n", i+1, result.Score)

		// Show first 200 chars of text
		text := result.Entry.Text
		if len(text) > 200 {
			text = text[:200] + "..."
		}
		resultText += fmt.Sprintf("Text: %s\n", text)

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

	// Ensure collection exists (with 1536 dimensions for OpenAI embeddings)
	if err := h.qdrantClient.EnsureCollection(collectionName, 1536); err != nil {
		return createErrorResult(fmt.Sprintf("failed to ensure collection exists: %s", err.Error())), nil, nil
	}

	// Generate ID
	id := storage.GenerateID()

	// Store point with embedding
	if err := h.qdrantClient.StorePoint(collectionName, id, information, metadata); err != nil {
		return createErrorResult(fmt.Sprintf("failed to store knowledge: %s", err.Error())), nil, nil
	}

	resultText := fmt.Sprintf("✓ Knowledge stored in Qdrant\n\nID: %s\nCollection: %s\nVector dimensions: 1536",
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
