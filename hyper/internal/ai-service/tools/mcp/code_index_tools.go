package mcp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"hyper/internal/ai-service"
	"hyper/internal/ai-service/tools"
	"hyper/internal/mcp/embeddings"
	"hyper/internal/mcp/storage"

	"go.uber.org/zap"
)

// Global cache for last code_index_search result (for auto-populating filesModified)
var (
	lastCodeSearchCache     []string
	lastCodeSearchTimestamp time.Time
	codeSearchCacheMutex    sync.RWMutex
	codeSearchCacheTTL      = 5 * time.Minute
)

// GetLastCodeSearchPaths returns the cached file paths from the most recent code_index_search
// Returns nil if cache is expired (older than 5 minutes) or empty
func GetLastCodeSearchPaths() []string {
	codeSearchCacheMutex.RLock()
	defer codeSearchCacheMutex.RUnlock()

	// Check if cache is expired
	if time.Since(lastCodeSearchTimestamp) > codeSearchCacheTTL {
		return nil
	}

	// Return a copy to avoid external modification
	if len(lastCodeSearchCache) == 0 {
		return nil
	}
	result := make([]string, len(lastCodeSearchCache))
	copy(result, lastCodeSearchCache)
	return result
}

// CodeIndexSearchTool implements the ToolExecutor interface for code search
// NOW FULLY FUNCTIONAL with embedding and Qdrant clients
type CodeIndexSearchTool struct {
	codeIndexStorage *storage.CodeIndexStorage
	embeddingClient  embeddings.EmbeddingClient
	qdrantClient     *storage.QdrantClient
	logger           *zap.Logger
}

// NewCodeIndexSearchTool creates a new code search tool instance
func NewCodeIndexSearchTool(codeIndexStorage *storage.CodeIndexStorage, embeddingClient embeddings.EmbeddingClient, qdrantClient *storage.QdrantClient, logger *zap.Logger) *CodeIndexSearchTool {
	return &CodeIndexSearchTool{
		codeIndexStorage: codeIndexStorage,
		embeddingClient:  embeddingClient,
		qdrantClient:     qdrantClient,
		logger:           logger,
	}
}

func (t *CodeIndexSearchTool) Name() string {
	return "code_index_search"
}

func (t *CodeIndexSearchTool) Description() string {
	return "Search for code using natural language queries. Returns relevant code snippets with file paths and line numbers. Default limit: 10 results (max: 50). Use to find examples, patterns, or specific implementations. NOTE: Requires code index to be populated via MCP endpoint first."
}

func (t *CodeIndexSearchTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Natural language search query (e.g., 'authentication logic', 'error handling for API calls')",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results (default: 10, max: 50)",
			},
			"folderPath": map[string]interface{}{
				"type":        "string",
				"description": "Optional: filter results to a specific folder path",
			},
		},
		"required": []string{"query"},
	}
}

func (t *CodeIndexSearchTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Validate required fields
	query, ok := input["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query is required and must be a string")
	}

	// Parse limit parameter (default: 10, max: 50)
	limit := 10
	if l, ok := input["limit"].(float64); ok {
		limit = int(l)
		if limit > 50 {
			limit = 50
		}
	}

	// Get retrieve mode (default: "chunk")
	retrieveMode := "chunk"
	if mode, ok := input["retrieve"].(string); ok {
		if mode == "full" || mode == "chunk" {
			retrieveMode = mode
		}
	}

	// Get current project root
	projectRoot := tools.GetProjectRoot()

	// Lookup collection name from code_index_map
	mapping, err := t.codeIndexStorage.GetPathMapping(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup collection mapping: %w", err)
	}
	if mapping == nil {
		return nil, fmt.Errorf("no code index found for project root '%s' - please restart coordinator to auto-index", projectRoot)
	}

	collectionName := mapping.QdrantCollection
	t.logger.Info("Code search: using collection",
		zap.String("collection", collectionName),
		zap.String("projectRoot", projectRoot),
		zap.String("query", query))

	// Generate embedding for query
	queryEmbedding, err := t.embeddingClient.CreateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("failed to create query embedding: %w", err)
	}
	t.logger.Info("Code search: embedding generated",
		zap.Int("dimensions", len(queryEmbedding)),
		zap.String("collection", collectionName))

	// Search in Qdrant using the correct collection
	searchResp, err := t.qdrantClient.SearchCodeIndex(collectionName, queryEmbedding, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search in collection '%s': %w", collectionName, err)
	}
	t.logger.Info("Code search: Qdrant search response",
		zap.Int("resultCount", len(searchResp.Result)),
		zap.String("collection", collectionName))

	// Build results (filter out archived paths)
	var results []storage.SearchResult
	for _, hit := range searchResp.Result {
		// Extract file path first to check for archived directories
		var filePath string
		if fp, ok := hit.Payload["filePath"].(string); ok {
			filePath = fp
		}

		// FILTER: Skip archived/deprecated files
		if strings.Contains(filePath, "/.archived/") || strings.Contains(filePath, "/.archive/") {
			t.logger.Debug("Skipping archived file from code search results",
				zap.String("filePath", filePath))
			continue
		}

		result := storage.SearchResult{
			Score:    hit.Score,
			FilePath: filePath,
		}

		// Extract payload fields
		if fileID, ok := hit.Payload["fileId"].(string); ok {
			result.FileID = fileID
		}
		if folderID, ok := hit.Payload["folderId"].(string); ok {
			result.FolderID = folderID
		}
		if folderPath, ok := hit.Payload["folderPath"].(string); ok {
			result.FolderPath = folderPath
		}
		if relativePath, ok := hit.Payload["relativePath"].(string); ok {
			result.RelativePath = relativePath
		}
		if language, ok := hit.Payload["language"].(string); ok {
			result.Language = language
		}
		if chunkNum, ok := hit.Payload["chunkNum"].(float64); ok {
			result.ChunkNum = int(chunkNum)
		}
		if startLine, ok := hit.Payload["startLine"].(float64); ok {
			result.StartLine = int(startLine)
		}
		if endLine, ok := hit.Payload["endLine"].(float64); ok {
			result.EndLine = int(endLine)
		}

		// Handle content based on retrieve mode
		if retrieveMode == "chunk" {
			// Default: return just the matching chunk content from Qdrant
			if content, ok := hit.Payload["content"].(string); ok {
				result.Content = content
			}
		} else if retrieveMode == "full" {
			// Fetch entire file content from MongoDB
			if result.FileID != "" {
				allChunks, err := t.codeIndexStorage.GetChunksByFileID(result.FileID)
				if err != nil {
					t.logger.Warn("Failed to fetch full file content",
						zap.String("fileID", result.FileID),
						zap.Error(err))
					// Fallback to chunk content
					if content, ok := hit.Payload["content"].(string); ok {
						result.Content = content
					}
				} else {
					// Concatenate all chunks to build full file content
					var fullContent strings.Builder
					for _, chunk := range allChunks {
						fullContent.WriteString(chunk.Content)
					}
					result.Content = fullContent.String()
					result.FullFileRetrieved = true
				}
			}
		}

		results = append(results, result)
	}

	t.logger.Info("Code search completed",
		zap.String("query", query),
		zap.String("retrieveMode", retrieveMode),
		zap.Int("results", len(results)))

	// CRITICAL: Extract file paths into a prominent list so AI can't miss them
	filePaths := make([]string, 0, len(results))
	for _, result := range results {
		filePaths = append(filePaths, result.FilePath)
	}

	// CACHE: Store file paths for auto-populating filesModified in create_agent_task
	codeSearchCacheMutex.Lock()
	lastCodeSearchCache = make([]string, len(filePaths))
	copy(lastCodeSearchCache, filePaths)
	lastCodeSearchTimestamp = time.Now()
	codeSearchCacheMutex.Unlock()

	t.logger.Info("Code search: cached file paths for auto-population",
		zap.Int("cachedPaths", len(filePaths)),
		zap.Time("cachedAt", lastCodeSearchTimestamp))

	return map[string]interface{}{
		"success":     true,
		"query":       query,
		"FILE_PATHS_TO_USE": filePaths, // PROMINENT - AI must use these exact paths
		"results":     results,
		"resultCount": len(results),
		"INSTRUCTIONS": "USE THE EXACT FILE PATHS FROM 'FILE_PATHS_TO_USE' ARRAY - DO NOT GUESS OR MODIFY THEM",
	}, nil
}

// CodeIndexAddFolderTool implements the ToolExecutor interface for adding folders
type CodeIndexAddFolderTool struct {
	codeIndexStorage *storage.CodeIndexStorage
}

func (t *CodeIndexAddFolderTool) Name() string {
	return "code_index_add_folder"
}

func (t *CodeIndexAddFolderTool) Description() string {
	return "DO NOT USE - Project root is already indexed automatically on server startup. This tool is for admin use only via MCP endpoint. Use code_index_search directly to search the already-indexed codebase."
}

func (t *CodeIndexAddFolderTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"folderPath": map[string]interface{}{
				"type":        "string",
				"description": "Absolute path to the folder to index",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Optional description of the folder/project",
			},
		},
		"required": []string{"folderPath"},
	}
}

func (t *CodeIndexAddFolderTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Validate required fields
	folderPath, ok := input["folderPath"].(string)
	if !ok || folderPath == "" {
		return nil, fmt.Errorf("folderPath is required and must be a string")
	}

	description, _ := input["description"].(string)

	// Security: Validate folder path exists and user has access
	// TODO: Implement access control (check ALLOWED_DIRS or similar whitelist)

	// Check if folder already exists
	existing, err := t.codeIndexStorage.GetFolderByPath(folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing folder: %w", err)
	}
	if existing != nil {
		return map[string]interface{}{
			"status":  "already_indexed",
			"message": "Folder already indexed. File watcher is monitoring changes.",
			"folder":  existing,
		}, nil
	}

	// Add folder to storage
	folder, err := t.codeIndexStorage.AddFolder(folderPath, description)
	if err != nil {
		return nil, fmt.Errorf("failed to add folder: %w", err)
	}

	// NOTE: File watcher and scanning require additional dependencies
	// Users should use code_index_scan MCP tool via /mcp endpoint to index files

	return map[string]interface{}{
		"status":  "added",
		"message": "Folder added successfully. Use code_index_scan MCP tool via /mcp endpoint to index existing files.",
		"folder":  folder,
		"next_step": "Call code_index_scan MCP tool with folderPath to start indexing files",
	}, nil
}

// CodeIndexScanTool implements the ToolExecutor interface for scanning folders
type CodeIndexScanTool struct {
	codeIndexStorage *storage.CodeIndexStorage
}

func (t *CodeIndexScanTool) Name() string {
	return "code_index_scan"
}

func (t *CodeIndexScanTool) Description() string {
	return "DO NOT USE - Code is already indexed automatically on server startup. This tool requires direct MCP access and will fail if called. Use code_index_search directly to search the already-indexed codebase."
}

func (t *CodeIndexScanTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"folderPath": map[string]interface{}{
				"type":        "string",
				"description": "Absolute path to the folder to scan",
			},
		},
		"required": []string{"folderPath"},
	}
}

func (t *CodeIndexScanTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	folderPath, ok := input["folderPath"].(string)
	if !ok || folderPath == "" {
		return nil, fmt.Errorf("folderPath is required and must be a string")
	}

	// NOTE: Full scanning requires:
	// - File system scanner to detect code files
	// - Embedding client to generate embeddings
	// - Qdrant client to store vectors
	//
	// Return an actual error to prevent AI from retrying
	return nil, fmt.Errorf("code scanning requires direct MCP tool access for file system scanning and embedding generation - this tool cannot perform scanning operations in this context")
}

// CodeIndexStatusTool implements the ToolExecutor interface for index status
type CodeIndexStatusTool struct {
	codeIndexStorage *storage.CodeIndexStorage
}

func (t *CodeIndexStatusTool) Name() string {
	return "code_index_status"
}

func (t *CodeIndexStatusTool) Description() string {
	return "Get the status of the code index to verify what folders and files are already indexed. The project root is auto-indexed on startup - use this to confirm indexing is complete before using code_index_search."
}

func (t *CodeIndexStatusTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (t *CodeIndexStatusTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	status, err := t.codeIndexStorage.GetIndexStatus()
	if err != nil {
		return nil, fmt.Errorf("failed to get index status: %w", err)
	}

	return status, nil
}

// CodeIndexRemoveFolderTool implements the ToolExecutor interface for removing folders
type CodeIndexRemoveFolderTool struct {
	codeIndexStorage *storage.CodeIndexStorage
}

func (t *CodeIndexRemoveFolderTool) Name() string {
	return "code_index_remove_folder"
}

func (t *CodeIndexRemoveFolderTool) Description() string {
	return "DO NOT USE - Destructive operation for admin use only. This will delete indexed data. The project root is already indexed - use code_index_search to search it."
}

func (t *CodeIndexRemoveFolderTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"folderPath": map[string]interface{}{
				"type":        "string",
				"description": "Absolute path to the folder to remove (must match the path used when adding)",
			},
		},
		"required": []string{"folderPath"},
	}
}

func (t *CodeIndexRemoveFolderTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	folderPath, ok := input["folderPath"].(string)
	if !ok || folderPath == "" {
		return nil, fmt.Errorf("folderPath is required and must be a string")
	}

	// Check if folder exists
	folder, err := t.codeIndexStorage.GetFolderByPath(folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check folder: %w", err)
	}
	if folder == nil {
		return nil, fmt.Errorf("folder not found in index: %s", folderPath)
	}

	// Remove folder
	err = t.codeIndexStorage.RemoveFolder(folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to remove folder: %w", err)
	}

	return map[string]interface{}{
		"status":     "removed",
		"message":    "Folder removed successfully",
		"folderPath": folderPath,
	}, nil
}

// RegisterCodeIndexTools registers code index tools with the tool registry
// NOW REQUIRES: embedding client and Qdrant client for full search functionality
func RegisterCodeIndexTools(
	registry *aiservice.ToolRegistry,
	codeIndexStorage *storage.CodeIndexStorage,
	embeddingClient embeddings.EmbeddingClient,
	qdrantClient *storage.QdrantClient,
	logger *zap.Logger,
) error {
	tools := []aiservice.ToolExecutor{
		&CodeIndexSearchTool{
			codeIndexStorage: codeIndexStorage,
			embeddingClient:  embeddingClient,
			qdrantClient:     qdrantClient,
			logger:           logger,
		},
		&CodeIndexAddFolderTool{codeIndexStorage: codeIndexStorage},
		&CodeIndexScanTool{codeIndexStorage: codeIndexStorage},
		&CodeIndexStatusTool{codeIndexStorage: codeIndexStorage},
		&CodeIndexRemoveFolderTool{codeIndexStorage: codeIndexStorage},
	}

	for _, tool := range tools {
		if err := registry.Register(tool); err != nil {
			return fmt.Errorf("failed to register %s: %w", tool.Name(), err)
		}
	}

	return nil
}
