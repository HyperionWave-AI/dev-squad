package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"hyper/internal/mcp/storage"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// QdrantToolHandler manages MCP Qdrant tool operations
type QdrantToolHandler struct {
	qdrantClient      storage.QdrantClientInterface
	knowledgeStorage  storage.KnowledgeStorage
	metadataRegistry  *ToolMetadataRegistry
}

// NewQdrantToolHandler creates a new Qdrant tool handler
func NewQdrantToolHandler(client storage.QdrantClientInterface) *QdrantToolHandler {
	return &QdrantToolHandler{
		qdrantClient: client,
	}
}

// SetKnowledgeStorage sets the knowledge storage for vote syncing operations
func (h *QdrantToolHandler) SetKnowledgeStorage(storage storage.KnowledgeStorage) {
	h.knowledgeStorage = storage
}

// SetMetadataRegistry sets the metadata registry for tool indexing
func (h *QdrantToolHandler) SetMetadataRegistry(registry *ToolMetadataRegistry) {
	h.metadataRegistry = registry
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

	// Register knowledge_get_by_id tool
	if err := h.registerKnowledgeGetByID(server); err != nil {
		return fmt.Errorf("failed to register knowledge_get_by_id tool: %w", err)
	}

	// Register knowledge_vote_on_entry tool
	if err := h.registerKnowledgeVoteOnEntry(server); err != nil {
		return fmt.Errorf("failed to register knowledge_vote_on_entry tool: %w", err)
	}

	// Register knowledge_get_entry_votes tool
	if err := h.registerKnowledgeGetEntryVotes(server); err != nil {
		return fmt.Errorf("failed to register knowledge_get_entry_votes tool: %w", err)
	}

	return nil
}

// registerKnowledgeFind registers the knowledge_find tool
func (h *QdrantToolHandler) registerKnowledgeFind(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "knowledge_find",
		Description: "Search for knowledge by semantic similarity. Returns top N results with scores and metadata. Supports pagination and smart token limiting to prevent response overflow. Use offset for pagination (e.g., offset=10 for second page of results). MANDATORY WORKFLOW: Before implementing any feature/fix, search relevant collections for existing patterns and solutions. After reading results, vote on usefulness using knowledge_vote_on_entry. Example queries: 'golang error handling patterns', 'React state management best practices', 'MongoDB indexing strategies'. IMPORTANT: Use knowledge_list_collections first to discover available collections and avoid 'collection not found' errors.",
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
					Description: "Maximum number of results per page (default: 5, max: 20)",
				},
				"offset": {
					Type:        "number",
					Description: "Number of results to skip for pagination (default: 0). Use for fetching additional pages beyond token limit.",
				},
				"retrieveMode": {
					Type:        "string",
					Description: "Content retrieval mode: 'full' (entire document) or 'chunk' (partial content). Default: 'chunk' to avoid token overflow",
					Enum:        []interface{}{"full", "chunk"},
				},
				"chunkSize": {
					Type:        "number",
					Description: "Maximum characters to return per result when retrieveMode is 'chunk' (default: 800, min: 100, max: 2000)",
				},
				"maxTokens": {
					Type:        "number",
					Description: "Maximum tokens in response (default: 20000, max: 24000). Response truncated if limit reached. Use offset to fetch remaining results.",
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

	// Report tool to metadata registry for indexing
	if h.metadataRegistry != nil {
		h.metadataRegistry.RegisterTool(
			tool.Name,
			tool.Description,
			map[string]interface{}{
				"type":        "mcp-tool",
				"name":        tool.Name,
				"description": tool.Description,
				"inputSchema": tool.InputSchema,
			},
		)
	}

	return nil
}

// registerKnowledgeStore registers the knowledge_store tool
func (h *QdrantToolHandler) registerKnowledgeStore(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "knowledge_store",
		Description: "Store knowledge with automatic embedding generation. Collections are auto-created if they don't exist. IMPORTANT: MAX 1000 tokens (~750 words, ~4000 characters) per entry. Write concise, AI-optimized entries. ONE concept per entry - split large documents into focused entries. Use knowledge_list_collections first to find the right collection. This knowledge base is designed for AI retrieval - keep entries focused and granular. Each entry should contain ONE specific concept, pattern, or procedure. For large documents, split into multiple focused entries. Returns storage confirmation with ID and collection.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"collectionName": {
					Type:        "string",
					Description: "Collection name (e.g., 'technical-knowledge', 'code-patterns')",
				},
				"information": {
					Type:        "string",
					Description: "Text content to store (MAX 1000 tokens ≈ 4000 characters)",
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

	// Report tool to metadata registry for indexing
	if h.metadataRegistry != nil {
		h.metadataRegistry.RegisterTool(
			tool.Name,
			tool.Description,
			map[string]interface{}{
				"type":        "mcp-tool",
				"name":        tool.Name,
				"description": tool.Description,
				"inputSchema": tool.InputSchema,
			},
		)
	}

	return nil
}

// handleQdrantFind handles the qdrant_find tool call with pagination and token limiting
func (h *QdrantToolHandler) handleQdrantFind(args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	// Check if knowledge storage is available
	if h.knowledgeStorage == nil {
		return createErrorResult("Knowledge storage not initialized. Cannot search knowledge."), nil, nil
	}

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

	// Extract offset (optional, default 0)
	offset := 0
	if o, ok := args["offset"].(float64); ok {
		offset = int(o)
		if offset < 0 {
			offset = 0
		}
	}

	// Extract retrieveMode (optional, default "chunk" to avoid token overflow)
	retrieveMode := "chunk"
	if mode, ok := args["retrieveMode"].(string); ok {
		if mode == "chunk" || mode == "full" {
			retrieveMode = mode
		}
	}

	// Extract chunkSize (optional, default 800, min 100, max 2000)
	chunkSize := 800
	if size, ok := args["chunkSize"].(float64); ok {
		chunkSize = int(size)
		if chunkSize < 100 {
			chunkSize = 100
		}
		if chunkSize > 2000 {
			chunkSize = 2000
		}
	}

	// Extract maxTokens (optional, default 20000, max 24000)
	maxTokens := 20000
	if mt, ok := args["maxTokens"].(float64); ok {
		maxTokens = int(mt)
		if maxTokens > 24000 {
			maxTokens = 24000
		}
		if maxTokens < 1000 {
			maxTokens = 1000
		}
	}

	// Fetch more results than needed to handle offset
	// Use knowledgeStorage.Query() which uses unified collection with collectionId filtering
	fetchLimit := limit + offset
	if fetchLimit > 50 {
		fetchLimit = 50 // Cap total fetch to prevent excessive queries
	}

	results, err := h.knowledgeStorage.Query(collectionName, query, fetchLimit, nil)
	if err != nil {
		// Provide helpful recovery guidance based on error type
		errMsg := err.Error()
		if strings.Contains(errMsg, "collection not found") {
			return createErrorResult(fmt.Sprintf("Collection '%s' not found. Use knowledge_store to create it first, or try coordinator_query_knowledge as fallback.", collectionName)), nil, nil
		}
		if strings.Contains(errMsg, "connection") || strings.Contains(errMsg, "dial") || strings.Contains(errMsg, "lookup") || strings.Contains(errMsg, "timeout") {
			return createErrorResult(fmt.Sprintf("Search service unavailable. Use coordinator_query_knowledge as fallback for task-specific knowledge. Original error: %s", errMsg)), nil, nil
		}
		return createErrorResult(fmt.Sprintf("Search failed: %s. Try coordinator_query_knowledge as alternative.", errMsg)), nil, nil
	}

	// Apply offset by slicing results
	totalResults := len(results)
	if offset >= totalResults {
		// Return empty result with helpful message
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("No results at offset %d (total results: %d). Try a lower offset value.", offset, totalResults)},
			},
		}, []interface{}{}, nil
	}

	// Slice to get paginated results
	results = results[offset:]
	if len(results) > limit {
		results = results[:limit]
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

	// Format results with smart token limiting
	// Estimate ~4 characters per token (conservative)
	maxChars := maxTokens * 4
	currentChars := 0
	truncated := false
	resultsIncluded := 0

	headerText := fmt.Sprintf("Found %d total results (showing from offset %d, retrieveMode: %s):\n\n", totalResults, offset, retrieveMode)
	currentChars += len(headerText)
	resultText := headerText

	for i, result := range results {
		// Build this result's text
		resultLine := fmt.Sprintf("Result %d (Score: %.2f, ID: %s)\n", offset+i+1, result.Score, result.Entry.ID)

		// Apply chunking logic based on retrieveMode
		text := result.Entry.Text
		if retrieveMode == "chunk" && len(text) > chunkSize {
			text = text[:chunkSize] + "..."
			resultLine += fmt.Sprintf("Text (truncated to %d chars): %s\n", chunkSize, text)
		} else if retrieveMode == "full" {
			resultLine += fmt.Sprintf("Text: %s\n", text)
		} else {
			// Default fallback
			resultLine += fmt.Sprintf("Text: %s\n", text)
		}

		// Show minimal metadata in chunk mode, full in full mode
		if len(result.Entry.Metadata) > 0 {
			if retrieveMode == "chunk" {
				// Minimal metadata - just collection name and ID
				resultLine += fmt.Sprintf("ID: %s | Use knowledge_get_by_id for full details\n", result.Entry.ID)
			} else {
				metadataJSON, _ := json.MarshalIndent(result.Entry.Metadata, "", "  ")
				resultLine += fmt.Sprintf("Metadata: %s\n", string(metadataJSON))
			}
		}

		resultLine += "\n---\n\n"

		// Check if adding this result would exceed token limit
		estimatedChars := len(resultLine)
		if currentChars+estimatedChars > maxChars && resultsIncluded > 0 {
			// Would exceed limit, stop here
			truncated = true
			break
		}

		// Add this result
		resultText += resultLine
		currentChars += estimatedChars
		resultsIncluded++
	}

	// Add pagination hint if truncated
	if truncated {
		nextOffset := offset + resultsIncluded
		resultText += fmt.Sprintf("\n⚠️ Response truncated at %d results due to token limit (%d tokens).\n", resultsIncluded, maxTokens)
		resultText += fmt.Sprintf("Use offset=%d to fetch next page, or use knowledge_get_by_id with specific IDs.\n", nextOffset)
	} else if offset+resultsIncluded < totalResults {
		// More results available but not fetched yet
		nextOffset := offset + resultsIncluded
		resultText += fmt.Sprintf("\nℹ️ Showing %d results. %d more available starting at offset=%d\n", resultsIncluded, totalResults-nextOffset, nextOffset)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, results, nil
}

// registerKnowledgeGetByID registers the knowledge_get_by_id tool
func (h *QdrantToolHandler) registerKnowledgeGetByID(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "knowledge_get_by_id",
		Description: "Retrieve a specific knowledge entry by its ID. Returns full content and metadata without token limits. Use this after knowledge_find to get complete details of specific entries.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"id": {
					Type:        "string",
					Description: "Entry ID to retrieve (obtained from knowledge_find results)",
				},
				"retrieveMode": {
					Type:        "string",
					Description: "Content retrieval mode: 'full' (entire document) or 'chunk' (partial content). Default: 'full'",
					Enum:        []interface{}{"full", "chunk"},
				},
				"chunkSize": {
					Type:        "number",
					Description: "Maximum characters to return when retrieveMode is 'chunk' (default: 2000, min: 100, max: 5000)",
				},
			},
			Required: []string{"id"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleKnowledgeGetByID(args)
		return result, err
	})

	// Report tool to metadata registry for indexing
	if h.metadataRegistry != nil {
		h.metadataRegistry.RegisterTool(
			tool.Name,
			tool.Description,
			map[string]interface{}{
				"type":        "mcp-tool",
				"name":        tool.Name,
				"description": tool.Description,
				"inputSchema": tool.InputSchema,
			},
		)
	}

	return nil
}

// handleKnowledgeGetByID retrieves a specific knowledge entry by ID
func (h *QdrantToolHandler) handleKnowledgeGetByID(args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	// Check if knowledge storage is available
	if h.knowledgeStorage == nil {
		return createErrorResult("Knowledge storage not initialized. Cannot retrieve knowledge."), nil, nil
	}

	// Extract id (required)
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return createErrorResult("id parameter is required and must be a non-empty string"), nil, nil
	}

	// Extract retrieveMode (optional, default "full")
	retrieveMode := "full"
	if mode, ok := args["retrieveMode"].(string); ok {
		if mode == "chunk" || mode == "full" {
			retrieveMode = mode
		}
	}

	// Extract chunkSize (optional, default 2000, min 100, max 5000)
	chunkSize := 2000
	if size, ok := args["chunkSize"].(float64); ok {
		chunkSize = int(size)
		if chunkSize < 100 {
			chunkSize = 100
		}
		if chunkSize > 5000 {
			chunkSize = 5000
		}
	}

	// Retrieve entry by ID
	entry, err := h.knowledgeStorage.GetEntryByID(id)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			return createErrorResult(fmt.Sprintf("Entry with ID '%s' not found. Verify the ID from knowledge_find results.", id)), nil, nil
		}
		return createErrorResult(fmt.Sprintf("Failed to retrieve entry: %s", errMsg)), nil, nil
	}

	// Format result
	resultText := fmt.Sprintf("Knowledge Entry (ID: %s)\n", entry.ID)
	resultText += fmt.Sprintf("Collection: %s\n", entry.Collection)
	resultText += fmt.Sprintf("Created: %s\n\n", entry.CreatedAt.Format("2006-01-02 15:04:05"))

	// Apply chunking if requested
	text := entry.Text
	if retrieveMode == "chunk" && len(text) > chunkSize {
		text = text[:chunkSize] + "..."
		resultText += fmt.Sprintf("Text (truncated to %d chars):\n%s\n\n", chunkSize, text)
		resultText += fmt.Sprintf("ℹ️ Full text is %d characters. Use retrieveMode='full' to get complete content.\n\n", len(entry.Text))
	} else {
		resultText += fmt.Sprintf("Text (%d chars):\n%s\n\n", len(entry.Text), text)
	}

	// Show full metadata
	if len(entry.Metadata) > 0 {
		metadataJSON, _ := json.MarshalIndent(entry.Metadata, "", "  ")
		resultText += fmt.Sprintf("Metadata:\n%s\n", string(metadataJSON))
	} else {
		resultText += "Metadata: None\n"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, entry, nil
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

	// Validate token count (approximate: 1 token ≈ 4 characters)
	const maxTokens = 1000
	const maxChars = maxTokens * 4 // ~4000 characters

	if len(information) > maxChars {
		estimatedTokens := len(information) / 4
		return createErrorResult(fmt.Sprintf(
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
		)), nil, nil
	}

	// Extract metadata (optional)
	var metadata map[string]interface{}
	if m, ok := args["metadata"].(map[string]interface{}); ok {
		metadata = m
	}

	// Check if knowledge storage is available
	if h.knowledgeStorage == nil {
		return createErrorResult("Knowledge storage not initialized. Cannot store knowledge."), nil, nil
	}

	// Store in both MongoDB and Qdrant using the knowledge storage layer (no taskId for this tool)
	entry, err := h.knowledgeStorage.Upsert(collectionName, information, metadata, nil)
	if err != nil {
		// Provide helpful error message
		errMsg := err.Error()
		if strings.Contains(errMsg, "connection") || strings.Contains(errMsg, "dial") || strings.Contains(errMsg, "lookup") || strings.Contains(errMsg, "timeout") {
			return createErrorResult(fmt.Sprintf("Storage service unavailable. Original error: %s", errMsg)), nil, nil
		}
		return createErrorResult(fmt.Sprintf("Failed to store knowledge: %s", errMsg)), nil, nil
	}

	resultText := fmt.Sprintf("✓ Knowledge stored successfully\n\nID: %s\nCollection: %s\nStored in: MongoDB + Qdrant (vector embeddings)",
		entry.ID, collectionName)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, map[string]interface{}{
		"id":         entry.ID,
		"collection": collectionName,
	}, nil
}

// registerKnowledgeVoteOnEntry registers the knowledge_vote_on_entry tool
func (h *QdrantToolHandler) registerKnowledgeVoteOnEntry(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "knowledge_vote_on_entry",
		Description: "Record or update a vote on a knowledge entry. IMPORTANT: Vote on every article you read. Vote + if it helped solve your problem or provided useful context. Vote - if it was outdated, incorrect, or unhelpful. Your votes improve search ranking and help other agents find quality knowledge. Votes are used to improve search ranking and identify high-quality content.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"entryId": {
					Type:        "string",
					Description: "Knowledge entry ID to vote on (from knowledge_find or knowledge_get_by_id)",
				},
				"vote": {
					Type:        "string",
					Description: "Vote type: '+' for upvote or '-' for downvote",
					Enum:        []interface{}{"+", "-"},
				},
				"reason": {
					Type:        "string",
					Description: "Reason for the vote (required, helps improve content quality)",
				},
			},
			Required: []string{"entryId", "vote", "reason"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleKnowledgeVoteOnEntry(args)
		return result, err
	})

	// Report tool to metadata registry for indexing
	if h.metadataRegistry != nil {
		h.metadataRegistry.RegisterTool(
			tool.Name,
			tool.Description,
			map[string]interface{}{
				"type":        "mcp-tool",
				"name":        tool.Name,
				"description": tool.Description,
				"inputSchema": tool.InputSchema,
				"category":    "knowledge",
				"tags":        []string{"voting", "feedback"},
			},
		)
	}

	return nil
}

// handleKnowledgeVoteOnEntry handles the knowledge_vote_on_entry tool call
func (h *QdrantToolHandler) handleKnowledgeVoteOnEntry(args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	// Check if knowledge storage is available
	if h.knowledgeStorage == nil {
		return createErrorResult("Knowledge storage not initialized. Cannot record vote."), nil, nil
	}

	// Extract entryId (required)
	entryID, ok := args["entryId"].(string)
	if !ok || entryID == "" {
		return createErrorResult("entryId parameter is required and must be a non-empty string"), nil, nil
	}

	// Extract vote (required)
	vote, ok := args["vote"].(string)
	if !ok || vote == "" {
		return createErrorResult("vote parameter is required and must be a non-empty string"), nil, nil
	}

	// Validate vote value
	if vote != "+" && vote != "-" {
		return createErrorResult(fmt.Sprintf("vote must be '+' or '-', got: '%s'", vote)), nil, nil
	}

	// Extract reason (required)
	reason, ok := args["reason"].(string)
	if !ok || reason == "" {
		return createErrorResult("reason parameter is required and must be a non-empty string"), nil, nil
	}

	// Get userID from context, fallback to "system" like HTTP handler
	userID := "system"
	// TODO: Extract userID from context when MCP authentication is implemented

	// Record vote
	voteRecord, err := h.knowledgeStorage.VoteOnEntry(entryID, userID, vote, reason)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			return createErrorResult(fmt.Sprintf("Entry with ID '%s' not found. Use knowledge_find or knowledge_get_by_id to verify the entry exists.", entryID)), nil, nil
		}
		return createErrorResult(fmt.Sprintf("Failed to record vote: %s", errMsg)), nil, nil
	}

	// Get updated vote summary
	summary, err := h.knowledgeStorage.GetEntryVotes(entryID, userID)
	if err != nil {
		// Vote was recorded but failed to get summary - still return success
		resultText := fmt.Sprintf("✓ Vote recorded successfully\n\nEntry ID: %s\nVote: %s\nReason: %s\n\n⚠️ Could not retrieve vote summary: %s",
			entryID, vote, reason, err.Error())
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: resultText},
			},
		}, voteRecord, nil
	}

	// Format success response with vote summary
	voteType := "upvote"
	if vote == "-" {
		voteType = "downvote"
	}
	resultText := fmt.Sprintf("✓ Vote recorded successfully\n\nEntry ID: %s\nYour vote: %s (%s)\nReason: %s\n\nVote Summary:\n• Upvotes: %d\n• Downvotes: %d\n• Net Score: %d",
		entryID, voteType, vote, reason, summary.Upvotes, summary.Downvotes, summary.NetScore)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, map[string]interface{}{
		"vote":    voteRecord,
		"summary": summary,
	}, nil
}

// registerKnowledgeGetEntryVotes registers the knowledge_get_entry_votes tool
func (h *QdrantToolHandler) registerKnowledgeGetEntryVotes(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "knowledge_get_entry_votes",
		Description: "Retrieve voting statistics for a knowledge entry. Returns upvote/downvote counts, net score, and the user's current vote if any. Use this to check article quality before relying on it. Articles with high net scores are more trusted by the community. Consider voting after reading.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"entryId": {
					Type:        "string",
					Description: "Knowledge entry ID to get votes for (from knowledge_find or knowledge_get_by_id)",
				},
			},
			Required: []string{"entryId"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleKnowledgeGetEntryVotes(args)
		return result, err
	})

	// Report tool to metadata registry for indexing
	if h.metadataRegistry != nil {
		h.metadataRegistry.RegisterTool(
			tool.Name,
			tool.Description,
			map[string]interface{}{
				"type":        "mcp-tool",
				"name":        tool.Name,
				"description": tool.Description,
				"inputSchema": tool.InputSchema,
				"category":    "knowledge",
				"tags":        []string{"voting", "stats"},
			},
		)
	}

	return nil
}

// handleKnowledgeGetEntryVotes handles the knowledge_get_entry_votes tool call
func (h *QdrantToolHandler) handleKnowledgeGetEntryVotes(args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	// Check if knowledge storage is available
	if h.knowledgeStorage == nil {
		return createErrorResult("Knowledge storage not initialized. Cannot retrieve votes."), nil, nil
	}

	// Extract entryId (required)
	entryID, ok := args["entryId"].(string)
	if !ok || entryID == "" {
		return createErrorResult("entryId parameter is required and must be a non-empty string"), nil, nil
	}

	// Get userID from context, fallback to "system" like HTTP handler
	userID := "system"
	// TODO: Extract userID from context when MCP authentication is implemented

	// Get vote summary
	summary, err := h.knowledgeStorage.GetEntryVotes(entryID, userID)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			return createErrorResult(fmt.Sprintf("Entry with ID '%s' not found. Use knowledge_find or knowledge_get_by_id to verify the entry exists.", entryID)), nil, nil
		}
		return createErrorResult(fmt.Sprintf("Failed to retrieve votes: %s", errMsg)), nil, nil
	}

	// Format response
	resultText := fmt.Sprintf("Vote Summary for Entry: %s\n\n• Upvotes: %d\n• Downvotes: %d\n• Net Score: %d",
		entryID, summary.Upvotes, summary.Downvotes, summary.NetScore)

	if summary.UserVote != "" {
		voteType := "upvote"
		if summary.UserVote == "-" {
			voteType = "downvote"
		}
		resultText += fmt.Sprintf("\n• Your vote: %s (%s)", voteType, summary.UserVote)
	} else {
		resultText += "\n• Your vote: None"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, summary, nil
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
