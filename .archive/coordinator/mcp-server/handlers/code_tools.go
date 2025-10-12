package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"hyperion-coordinator-mcp/embeddings"
	"hyperion-coordinator-mcp/scanner"
	"hyperion-coordinator-mcp/storage"
	"hyperion-coordinator-mcp/watcher"

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

// RegisterCodeIndexTools registers all code indexing MCP tools
func (h *CodeToolsHandler) RegisterCodeIndexTools(server *mcp.Server) error {
	if err := h.registerAddFolder(server); err != nil {
		return fmt.Errorf("failed to register code_index_add_folder tool: %w", err)
	}

	if err := h.registerRemoveFolder(server); err != nil {
		return fmt.Errorf("failed to register code_index_remove_folder tool: %w", err)
	}

	if err := h.registerScan(server); err != nil {
		return fmt.Errorf("failed to register code_index_scan tool: %w", err)
	}

	if err := h.registerSearch(server); err != nil {
		return fmt.Errorf("failed to register code_index_search tool: %w", err)
	}

	if err := h.registerStatus(server); err != nil {
		return fmt.Errorf("failed to register code_index_status tool: %w", err)
	}

	h.logger.Info("Registered code indexing MCP tools", zap.Int("count", 5))
	return nil
}

// registerAddFolder registers the code_index_add_folder tool
func (h *CodeToolsHandler) registerAddFolder(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "code_index_add_folder",
		Description: "Add a folder to the code index for semantic search. The folder will be scanned and all supported code files will be indexed.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"folderPath": {
					Type:        "string",
					Description: "Absolute path to the folder to index",
				},
				"description": {
					Type:        "string",
					Description: "Optional description of the folder/project",
				},
			},
			Required: []string{"folderPath"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createCodeIndexErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		return h.handleAddFolder(ctx, args)
	})

	return nil
}

// registerRemoveFolder registers the code_index_remove_folder tool
func (h *CodeToolsHandler) registerRemoveFolder(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "code_index_remove_folder",
		Description: "Remove a folder from the code index. This will delete all indexed files and their vectors.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"folderPath": {
					Type:        "string",
					Description: "Absolute path to the folder to remove (must match the path used when adding)",
				},
			},
			Required: []string{"folderPath"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createCodeIndexErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		return h.handleRemoveFolder(ctx, args)
	})

	return nil
}

// registerScan registers the code_index_scan tool
func (h *CodeToolsHandler) registerScan(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "code_index_scan",
		Description: "Scan or rescan a folder to update the code index. This will detect new/modified/deleted files and update the index accordingly.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"folderPath": {
					Type:        "string",
					Description: "Absolute path to the folder to scan",
				},
			},
			Required: []string{"folderPath"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return h.handleStatus(ctx)
	})

	return nil
}

// handleAddFolder handles the code_index_add_folder tool
func (h *CodeToolsHandler) handleAddFolder(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	folderPath, ok := args["folderPath"].(string)
	if !ok {
		return createCodeIndexErrorResult("folderPath is required and must be a string"), nil
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(folderPath)
	if err != nil {
		return createCodeIndexErrorResult(fmt.Sprintf("invalid folder path: %s", err.Error())), nil
	}

	description := ""
	if desc, ok := args["description"].(string); ok {
		description = desc
	}

	// Check if folder already exists
	existing, err := h.codeIndexStorage.GetFolderByPath(absPath)
	if err != nil {
		return createCodeIndexErrorResult(fmt.Sprintf("failed to check existing folder: %s", err.Error())), nil
	}
	if existing != nil {
		jsonData, _ := json.Marshal(map[string]interface{}{
			"success": true,
			"message": "Folder already indexed. File watcher is monitoring changes.",
			"folder":  existing,
		})
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(jsonData)},
			},
		}, nil
	}

	// Add folder to storage
	folder, err := h.codeIndexStorage.AddFolder(absPath, description)
	if err != nil {
		return createCodeIndexErrorResult(fmt.Sprintf("failed to add folder: %s", err.Error())), nil
	}

	// Add folder to file watcher
	if h.fileWatcher != nil {
		if err := h.fileWatcher.AddFolder(folder); err != nil {
			h.logger.Warn("Failed to add folder to file watcher", zap.Error(err))
		} else {
			h.logger.Info("Added folder to file watcher", zap.String("path", absPath))
		}
	}

	h.logger.Info("Added folder to code index",
		zap.String("folderID", folder.ID),
		zap.String("path", absPath))

	jsonData, _ := json.Marshal(map[string]interface{}{
		"success": true,
		"message": "Folder added successfully. File watcher is now monitoring changes. Use code_index_scan to index existing files.",
		"folder":  folder,
	})

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, nil
}

// handleRemoveFolder handles the code_index_remove_folder tool
func (h *CodeToolsHandler) handleRemoveFolder(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	folderPath, ok := args["folderPath"].(string)
	if !ok {
		return createCodeIndexErrorResult("folderPath is required and must be a string"), nil
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(folderPath)
	if err != nil {
		return createCodeIndexErrorResult(fmt.Sprintf("invalid folder path: %s", err.Error())), nil
	}

	// Get folder
	folder, err := h.codeIndexStorage.GetFolderByPath(absPath)
	if err != nil || folder == nil {
		return createCodeIndexErrorResult(fmt.Sprintf("folder not found: %s", absPath)), nil
	}

	// Get all files to delete their vectors
	files, err := h.codeIndexStorage.ListFiles(folder.ID)
	if err != nil {
		return createCodeIndexErrorResult(fmt.Sprintf("failed to list files: %s", err.Error())), nil
	}

	// Delete vectors from Qdrant
	if len(files) > 0 {
		err = h.qdrantClient.DeleteCodeIndexByFilter(map[string]interface{}{
			"must": []map[string]interface{}{
				{"key": "folderId", "match": map[string]interface{}{"value": folder.ID}},
			},
		})
		if err != nil {
			h.logger.Warn("Failed to delete vectors from Qdrant", zap.Error(err))
		}
	}

	// Remove folder from file watcher
	if h.fileWatcher != nil {
		if err := h.fileWatcher.RemoveFolder(absPath); err != nil {
			h.logger.Warn("Failed to remove folder from file watcher", zap.Error(err))
		} else {
			h.logger.Info("Removed folder from file watcher", zap.String("path", absPath))
		}
	}

	// Remove folder from MongoDB (cascades to files and chunks)
	if err := h.codeIndexStorage.RemoveFolder(folder.ID); err != nil {
		return createCodeIndexErrorResult(fmt.Sprintf("failed to remove folder: %s", err.Error())), nil
	}

	h.logger.Info("Removed folder from code index",
		zap.String("folderID", folder.ID),
		zap.String("path", absPath),
		zap.Int("filesRemoved", len(files)))

	jsonData, _ := json.Marshal(map[string]interface{}{
		"success":      true,
		"message":      "Folder removed successfully",
		"filesRemoved": len(files),
	})

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, nil
}

// handleScan handles the code_index_scan tool
func (h *CodeToolsHandler) handleScan(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	folderPath, ok := args["folderPath"].(string)
	if !ok {
		return createCodeIndexErrorResult("folderPath is required and must be a string"), nil
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(folderPath)
	if err != nil {
		return createCodeIndexErrorResult(fmt.Sprintf("invalid folder path: %s", err.Error())), nil
	}

	// Get folder
	folder, err := h.codeIndexStorage.GetFolderByPath(absPath)
	if err != nil || folder == nil {
		return createCodeIndexErrorResult(fmt.Sprintf("folder not found. Use code_index_add_folder first: %s", absPath)), nil
	}

	// Update folder status to scanning
	if err := h.codeIndexStorage.UpdateFolderStatus(folder.ID, "scanning", ""); err != nil {
		return createCodeIndexErrorResult(fmt.Sprintf("failed to update folder status: %s", err.Error())), nil
	}

	// Scan directory for files
	scannedFiles, err := h.fileScanner.ScanDirectory(absPath)
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

		// Upload vectors to Qdrant
		if len(qdrantPoints) > 0 {
			if err := h.qdrantClient.UpsertCodeIndexPoints(qdrantPoints); err != nil {
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

	// Generate embedding for query
	queryEmbedding, err := h.embeddingClient.CreateEmbedding(query)
	if err != nil {
		return createCodeIndexErrorResult(fmt.Sprintf("failed to create query embedding: %s", err.Error())), nil
	}

	// Search in Qdrant
	searchResp, err := h.qdrantClient.SearchCodeIndex(queryEmbedding, limit)
	if err != nil {
		return createCodeIndexErrorResult(fmt.Sprintf("failed to search: %s", err.Error())), nil
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

// createCodeIndexErrorResult creates an error result with the given message
func createCodeIndexErrorResult(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("❌ Error: %s", message)},
		},
		IsError: true,
	}
}
