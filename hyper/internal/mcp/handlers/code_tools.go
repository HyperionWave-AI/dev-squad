package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"hyper/internal/ai-service/tools"
	"hyper/internal/mcp/embeddings"
	"hyper/internal/mcp/scanner"
	"hyper/internal/mcp/storage"
	"hyper/internal/mcp/watcher"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// CodeToolsHandler handles MCP tool requests for code indexing
type CodeToolsHandler struct {
	codeIndexStorage *storage.CodeIndexStorage
	qdrantClient     *storage.QdrantClient
	embeddingClient  embeddings.EmbeddingClient
	fileScanner      *scanner.FileScanner
	fileWatcher      *watcher.FileWatcher
	logger           *zap.Logger
	metadataRegistry *ToolMetadataRegistry
}

// NewCodeToolsHandler creates a new code tools handler
func NewCodeToolsHandler(
	codeIndexStorage *storage.CodeIndexStorage,
	qdrantClient *storage.QdrantClient,
	embeddingClient embeddings.EmbeddingClient,
	fileWatcher *watcher.FileWatcher,
	logger *zap.Logger,
) *CodeToolsHandler {
	return &CodeToolsHandler{
		codeIndexStorage: codeIndexStorage,
		qdrantClient:     qdrantClient,
		embeddingClient:  embeddingClient,
		fileScanner:      scanner.NewFileScanner(),
		fileWatcher:      fileWatcher,
		logger:           logger,
	}
}

// SetMetadataRegistry sets the metadata registry for tool indexing
func (h *CodeToolsHandler) SetMetadataRegistry(registry *ToolMetadataRegistry) {
	h.metadataRegistry = registry
}

// addToolWithMetadata adds a tool to the server and registers it for indexing
func (h *CodeToolsHandler) addToolWithMetadata(server *mcp.Server, tool *mcp.Tool, handler mcp.ToolHandler) {
	server.AddTool(tool, handler)
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
}

// RegisterCodeIndexTools registers all code indexing MCP tools
func (h *CodeToolsHandler) RegisterCodeIndexTools(server *mcp.Server) error {
	if err := h.registerScan(server); err != nil {
		return fmt.Errorf("failed to register code_index_scan tool: %w", err)
	}

	if err := h.registerSearch(server); err != nil {
		return fmt.Errorf("failed to register code_index_search tool: %w", err)
	}

	if err := h.registerStatus(server); err != nil {
		return fmt.Errorf("failed to register code_index_status tool: %w", err)
	}

	h.logger.Info("Registered code indexing MCP tools", zap.Int("count", 3))
	return nil
}

// registerScan registers the code_index_scan tool
func (h *CodeToolsHandler) registerScan(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "code_index_scan",
		Description: "Scan or rescan a folder to update the code index. This will detect new/modified/deleted files and update the index accordingly. If folderPath is not provided, uses INDEX_SOURCE_PATH environment variable or current working directory.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"folderPath": {
					Type:        "string",
					Description: "Absolute path to the folder to scan (optional: defaults to INDEX_SOURCE_PATH env var or current directory)",
				},
			},
			Required: []string{},
		},
	}

	h.addToolWithMetadata(server, tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createCodeIndexErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		return h.handleScan(ctx, args)
	})

	return nil
}

// registerSearch registers the code_index_search tool
func (h *CodeToolsHandler) registerSearch(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "code_index_search",
		Description: "Search for code using natural language queries. Returns relevant code snippets with file paths and line numbers. Content can be retrieved as chunks (default) or full files.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"query": {
					Type:        "string",
					Description: "Natural language search query (e.g., 'authentication logic', 'error handling for API calls')",
				},
				"limit": {
					Type:        "number",
					Description: "Maximum number of results to return (default: 10, max: 50)",
				},
				"folderPath": {
					Type:        "string",
					Description: "Optional: filter results to a specific folder path",
				},
				"retrieve": {
					Type:        "string",
					Description: "Content retrieval mode: 'chunk' (default - return matching chunk only) or 'full' (return entire file content)",
					Enum:        []interface{}{"chunk", "full"},
				},
			},
			Required: []string{"query"},
		},
	}

	h.addToolWithMetadata(server, tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createCodeIndexErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		return h.handleSearch(ctx, args)
	})

	return nil
}

// registerStatus registers the code_index_status tool
func (h *CodeToolsHandler) registerStatus(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "code_index_status",
		Description: "Get the current status of the code index, including indexed folders, file counts, and last scan times.",
		InputSchema: &jsonschema.Schema{
			Type:       "object",
			Properties: map[string]*jsonschema.Schema{},
		},
	}

	h.addToolWithMetadata(server, tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return h.handleStatus(ctx)
	})

	return nil
}

// handleScan handles the code_index_scan tool
func (h *CodeToolsHandler) handleScan(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	// Always use project root (no manual folderPath parameter)
	projectRoot := tools.GetProjectRoot()

	// Lookup collection name from code_index_map
	mapping, err := h.codeIndexStorage.GetPathMapping(projectRoot)
	if err != nil {
		return createCodeIndexErrorResult(fmt.Sprintf("failed to lookup collection mapping: %s", err.Error())), nil
	}
	if mapping == nil {
		return createCodeIndexErrorResult(fmt.Sprintf("no code index found for project root '%s' - please restart coordinator to auto-index", projectRoot)), nil
	}

	collectionName := mapping.QdrantCollection

	// Get folder (for legacy compatibility with MongoDB storage)
	folder, err := h.codeIndexStorage.GetFolderByPath(projectRoot)
	if err != nil || folder == nil {
		return createCodeIndexErrorResult(fmt.Sprintf("folder metadata not found for: %s", projectRoot)), nil
	}

	// Update folder status to scanning
	if err := h.codeIndexStorage.UpdateFolderStatus(folder.ID, "scanning", ""); err != nil {
		return createCodeIndexErrorResult(fmt.Sprintf("failed to update folder status: %s", err.Error())), nil
	}

	// Scan directory for files
	scannedFiles, err := h.fileScanner.ScanDirectory(projectRoot)
	if err != nil {
		h.codeIndexStorage.UpdateFolderStatus(folder.ID, "error", err.Error())
		return createCodeIndexErrorResult(fmt.Sprintf("failed to scan directory: %s", err.Error())), nil
	}

	filesIndexed := 0
	filesUpdated := 0
	filesSkipped := 0

	// Process each file
	for _, scannedFile := range scannedFiles {
		scannedFile.FolderID = folder.ID

		// Check if file already exists
		existingFile, _ := h.codeIndexStorage.GetFileByPath(scannedFile.Path)

		if existingFile != nil {
			// Check if file has changed
			if existingFile.SHA256 == scannedFile.SHA256 {
				filesSkipped++
				continue
			}
			filesUpdated++
			scannedFile.ID = existingFile.ID
		} else {
			filesIndexed++
			scannedFile.ID = uuid.New().String()
		}

		// Create chunks
		chunks, err := h.fileScanner.CreateFileChunks(scannedFile.ID, scannedFile.Path)
		if err != nil {
			h.logger.Warn("Failed to create chunks", zap.String("file", scannedFile.Path), zap.Error(err))
			continue
		}

		// Generate embeddings for chunks
		var qdrantPoints []storage.CodeIndexPoint
		for _, chunk := range chunks {
			// Generate embedding
			embedding, err := h.embeddingClient.CreateEmbedding(chunk.Content)
			if err != nil {
				h.logger.Warn("Failed to create embedding",
					zap.String("file", scannedFile.Path),
					zap.Int("chunk", chunk.ChunkNum),
					zap.Error(err))
				continue
			}

			// Create Qdrant point
			pointID := fmt.Sprintf("%s_%d", scannedFile.ID, chunk.ChunkNum)
			chunk.VectorID = pointID

			point := storage.CodeIndexPoint{
				ID:     pointID,
				Vector: embedding,
				Payload: map[string]interface{}{
					"fileId":       scannedFile.ID,
					"folderId":     folder.ID,
					"folderPath":   folder.Path,
					"filePath":     scannedFile.Path,
					"relativePath": scannedFile.RelativePath,
					"language":     scannedFile.Language,
					"chunkNum":     chunk.ChunkNum,
					"startLine":    chunk.StartLine,
					"endLine":      chunk.EndLine,
					"content":      chunk.Content,
				},
			}
			qdrantPoints = append(qdrantPoints, point)

			// Save chunk to MongoDB
			if err := h.codeIndexStorage.UpsertChunk(chunk); err != nil {
				h.logger.Warn("Failed to save chunk", zap.Error(err))
			}
		}

		// Upload vectors to Qdrant (using the correct collection for this path)
		if len(qdrantPoints) > 0 {
			if err := h.qdrantClient.UpsertCodeIndexPoints(collectionName, qdrantPoints); err != nil {
				h.logger.Warn("Failed to upsert vectors", zap.String("file", scannedFile.Path), zap.Error(err))
			}
		}

		// Save file metadata to MongoDB
		if err := h.codeIndexStorage.UpsertFile(scannedFile); err != nil {
			h.logger.Warn("Failed to save file", zap.Error(err))
		}
	}

	// Update folder status and scan time
	if err := h.codeIndexStorage.UpdateFolderStatus(folder.ID, "active", ""); err != nil {
		h.logger.Warn("Failed to update folder status", zap.Error(err))
	}

	if err := h.codeIndexStorage.UpdateFolderScanTime(folder.ID, len(scannedFiles)); err != nil {
		h.logger.Warn("Failed to update scan time", zap.Error(err))
	}

	h.logger.Info("Completed folder scan",
		zap.String("folderID", folder.ID),
		zap.Int("filesIndexed", filesIndexed),
		zap.Int("filesUpdated", filesUpdated),
		zap.Int("filesSkipped", filesSkipped))

	jsonData, _ := json.Marshal(map[string]interface{}{
		"success":      true,
		"filesIndexed": filesIndexed,
		"filesUpdated": filesUpdated,
		"filesSkipped": filesSkipped,
		"totalFiles":   len(scannedFiles),
	})

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, nil
}

// handleSearch handles the code_index_search tool
func (h *CodeToolsHandler) handleSearch(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	query, ok := args["query"].(string)
	if !ok {
		return createCodeIndexErrorResult("query is required and must be a string"), nil
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}
	if limit > 50 {
		limit = 50
	}

	// Get retrieve mode (default: "chunk")
	retrieveMode := "chunk"
	if mode, ok := args["retrieve"].(string); ok {
		if mode == "full" || mode == "chunk" {
			retrieveMode = mode
		}
	}

	// Get current project root
	projectRoot := tools.GetProjectRoot()

	// Lookup collection name from code_index_map
	mapping, err := h.codeIndexStorage.GetPathMapping(projectRoot)
	if err != nil {
		return createCodeIndexErrorResult(fmt.Sprintf("failed to lookup collection mapping: %s", err.Error())), nil
	}
	if mapping == nil {
		return createCodeIndexErrorResult(fmt.Sprintf("no code index found for project root '%s' - please restart coordinator to auto-index, or the path has not been indexed yet", projectRoot)), nil
	}

	collectionName := mapping.QdrantCollection

	// Generate embedding for query
	queryEmbedding, err := h.embeddingClient.CreateEmbedding(query)
	if err != nil {
		return createCodeIndexErrorResult(fmt.Sprintf("failed to create query embedding: %s", err.Error())), nil
	}

	// Search in Qdrant using the correct collection
	searchResp, err := h.qdrantClient.SearchCodeIndex(collectionName, queryEmbedding, limit)
	if err != nil {
		return createCodeIndexErrorResult(fmt.Sprintf("failed to search in collection '%s': %s", collectionName, err.Error())), nil
	}

	// Build results
	var results []storage.SearchResult
	for _, hit := range searchResp.Result {
		result := storage.SearchResult{
			Score: hit.Score,
		}

		if fileID, ok := hit.Payload["fileId"].(string); ok {
			result.FileID = fileID
		}
		if folderID, ok := hit.Payload["folderId"].(string); ok {
			result.FolderID = folderID
		}
		if folderPath, ok := hit.Payload["folderPath"].(string); ok {
			result.FolderPath = folderPath
		}
		if filePath, ok := hit.Payload["filePath"].(string); ok {
			result.FilePath = filePath
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
				allChunks, err := h.codeIndexStorage.GetChunksByFileID(result.FileID)
				if err != nil {
					h.logger.Warn("Failed to fetch full file content",
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

	h.logger.Info("Code search completed",
		zap.String("query", query),
		zap.String("retrieveMode", retrieveMode),
		zap.Int("results", len(results)))

	jsonData, _ := json.Marshal(map[string]interface{}{
		"success":      true,
		"query":        query,
		"retrieveMode": retrieveMode,
		"results":      results,
		"count":        len(results),
	})

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, nil
}

// handleStatus handles the code_index_status tool
func (h *CodeToolsHandler) handleStatus(ctx context.Context) (*mcp.CallToolResult, error) {
	// Get index status
	status, err := h.codeIndexStorage.GetIndexStatus()
	if err != nil {
		return createCodeIndexErrorResult(fmt.Sprintf("failed to get index status: %s", err.Error())), nil
	}

	// Get folder details
	folders, err := h.codeIndexStorage.ListFolders()
	if err != nil {
		return createCodeIndexErrorResult(fmt.Sprintf("failed to list folders: %s", err.Error())), nil
	}

	// Calculate total size from all files
	totalSize := int64(0)
	for _, folder := range folders {
		files, _ := h.codeIndexStorage.ListFiles(folder.ID)
		for _, file := range files {
			totalSize += file.Size
		}
	}

	// Determine watcher status (running if file watcher exists and has active folders)
	watcherStatus := "stopped"
	if h.fileWatcher != nil && status.ActiveFolders > 0 {
		watcherStatus = "running"
	}

	// Transform folders to UI format
	uiFolders := make([]map[string]interface{}, 0, len(folders))
	for _, folder := range folders {
		uiFolders = append(uiFolders, map[string]interface{}{
			"folderPath": folder.Path,
			"fileCount":  folder.FileCount,
			"enabled":    folder.Status == "active",
		})
	}

	// Return in UI-expected format
	jsonData, _ := json.Marshal(map[string]interface{}{
		"totalFolders":  status.TotalFolders,
		"totalFiles":    status.TotalFiles,
		"totalSize":     totalSize,
		"watcherStatus": watcherStatus,
		"folders":       uiFolders,
	})

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, nil
}

// extractArguments safely extracts arguments from CallToolRequest
func (h *CodeToolsHandler) extractArguments(req *mcp.CallToolRequest) (map[string]interface{}, error) {
	if req.Params.Arguments == nil || len(req.Params.Arguments) == 0 {
		return make(map[string]interface{}), nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(req.Params.Arguments, &result); err != nil {
		return nil, fmt.Errorf("arguments must be a valid JSON object: %w", err)
	}

	return result, nil
}

// validateSafeIndexPath validates that a path is safe to index
// Prevents indexing system-critical directories that would destroy the system
func validateSafeIndexPath(path string) error {
	// Clean the path
	cleanPath := filepath.Clean(path)

	// CRITICAL: Reject root filesystem
	if cleanPath == "/" {
		return fmt.Errorf("cannot index root filesystem '/' - would destroy system")
	}

	// CRITICAL: Reject common system directories
	dangerousPaths := []string{
		"/bin", "/sbin", "/usr", "/lib", "/lib64",
		"/etc", "/var", "/sys", "/proc", "/dev", "/boot",
		"/System", "/Library", "/Applications", "/Volumes",
		"/private", "/cores", "/tmp", "/var/tmp",
	}

	for _, dangerous := range dangerousPaths {
		if cleanPath == dangerous {
			return fmt.Errorf("cannot index system directory '%s'", cleanPath)
		}
		if strings.HasPrefix(cleanPath, dangerous+"/") &&
		   dangerous != "/opt" && dangerous != "/tmp" {
			return fmt.Errorf("cannot index system subdirectory '%s' under '%s'", cleanPath, dangerous)
		}
	}

	// CRITICAL: Require minimum path depth
	pathSegments := strings.Split(strings.Trim(cleanPath, "/"), "/")
	if len(pathSegments) < 2 {
		return fmt.Errorf("path too shallow '%s' - must be at least 2 levels deep (e.g., /Users/name/project)", cleanPath)
	}

	// CRITICAL: Require paths to be within safe user directories
	allowedPrefixes := []string{
		"/Users/",     // macOS user directories
		"/home/",      // Linux user directories
		"/opt/",       // Optional software (containers, projects)
		"/workspace/", // Common container workspace
		"/app/",       // Common container app directory
	}

	isAllowed := false
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(cleanPath, prefix) {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		return fmt.Errorf("path '%s' must be within allowed directories: /Users/, /home/, /opt/, /workspace/, or /app/", cleanPath)
	}

	return nil
}

// createCodeIndexErrorResult creates an error result with the given message
func createCodeIndexErrorResult(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("❌ Error: %s", message)},
		},
		IsError: true,
	}
}
