package mcp

import (
	"context"
	"fmt"

	"hyper/internal/ai-service"
	"hyper/internal/mcp/storage"
)

// NOTE: Code index tools have complex dependencies (embedding client, file scanner, file watcher)
// that require full handler initialization. For MVP, these tools provide basic functionality.
// For full code search capabilities, use MCP tools directly via /mcp endpoint.

// CodeIndexSearchTool implements the ToolExecutor interface for code search
type CodeIndexSearchTool struct {
	codeIndexStorage *storage.CodeIndexStorage
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

	// NOTE: Full code search requires:
	// - Embedding client to generate query embeddings
	// - Qdrant client to search vector store
	// - Complex result formatting and chunking logic
	//
	// This is a simplified wrapper that checks if folders are indexed.
	// For full functionality, use code_index_search MCP tool via /mcp endpoint.

	// Check if any folders are indexed
	status, err := t.codeIndexStorage.GetIndexStatus()
	if err != nil {
		return nil, fmt.Errorf("failed to get index status: %w", err)
	}

	if status.TotalFolders == 0 {
		return nil, fmt.Errorf("no folders indexed. Use code_index_add_folder MCP tool to index a project directory first")
	}

	// Return guidance to use MCP endpoint
	return map[string]interface{}{
		"error":   "code_search_requires_mcp_endpoint",
		"message": "Code search requires direct MCP tool access for embedding generation and vector search. Use code_index_search MCP tool via /mcp endpoint instead.",
		"indexed_folders": status.TotalFolders,
		"indexed_files":   status.TotalFiles,
		"query":           query,
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
	return "Add a folder to the code index for semantic search. The folder will be monitored and code files will be indexed. Use this to enable code search for a project directory. After adding, use code_index_scan MCP tool to index existing files."
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
	return "Scan or rescan a folder to update the code index. This will detect new/modified/deleted files and update the index accordingly. Use after adding a folder or to refresh the index."
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
	return "Get the current status of the code index, including indexed folders, file counts, and last scan times. Use to verify what's indexed."
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
	return "Remove a folder from the code index. This will delete all indexed files and their vectors. Use to clean up when a project is no longer needed."
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
func RegisterCodeIndexTools(registry *aiservice.ToolRegistry, codeIndexStorage *storage.CodeIndexStorage) error {
	tools := []aiservice.ToolExecutor{
		&CodeIndexSearchTool{codeIndexStorage: codeIndexStorage},
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
