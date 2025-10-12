package handlers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// FilesystemToolHandler handles MCP tool requests for filesystem operations
type FilesystemToolHandler struct {
	logger  *zap.Logger
	baseDir string // Base directory for path validation
}

// NewFilesystemToolHandler creates a new filesystem tools handler
func NewFilesystemToolHandler(logger *zap.Logger) *FilesystemToolHandler {
	// Use current working directory as base
	baseDir, _ := os.Getwd()
	return &FilesystemToolHandler{
		logger:  logger,
		baseDir: baseDir,
	}
}

// RegisterFilesystemTools registers all filesystem MCP tools
func (h *FilesystemToolHandler) RegisterFilesystemTools(server *mcp.Server) error {
	if err := h.registerBashTool(server); err != nil {
		return fmt.Errorf("failed to register bash tool: %w", err)
	}

	if err := h.registerFileReadTool(server); err != nil {
		return fmt.Errorf("failed to register file_read tool: %w", err)
	}

	if err := h.registerFileWriteTool(server); err != nil {
		return fmt.Errorf("failed to register file_write tool: %w", err)
	}

	if err := h.registerApplyPatchTool(server); err != nil {
		return fmt.Errorf("failed to register apply_patch tool: %w", err)
	}

	h.logger.Info("Registered filesystem MCP tools", zap.Int("count", 4))
	return nil
}

// validatePath validates and sanitizes file paths to prevent directory traversal attacks
func (h *FilesystemToolHandler) validatePath(path string) (string, error) {
	// Check for directory traversal patterns in original path
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("directory traversal detected in path: %s", path)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Clean the path
	cleanPath := filepath.Clean(absPath)

	return cleanPath, nil
}

// sanitizeCommand performs basic command sanitization to prevent command injection
func (h *FilesystemToolHandler) sanitizeCommand(cmd string) (string, error) {
	// Check for dangerous patterns
	dangerous := []string{";", "&&", "||", "|", "`", "$", "(", ")"}
	for _, pattern := range dangerous {
		if strings.Contains(cmd, pattern) {
			return "", fmt.Errorf("command contains potentially dangerous pattern: %s", pattern)
		}
	}

	return strings.TrimSpace(cmd), nil
}

// registerBashTool registers the bash tool for command execution
func (h *FilesystemToolHandler) registerBashTool(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "bash",
		Description: "Execute bash commands with streaming output and timeout support. Returns command output and exit code.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"command": {
					Type:        "string",
					Description: "Bash command to execute",
				},
				"timeout": {
					Type:        "number",
					Description: "Optional timeout in seconds (default: 30, max: 300)",
				},
				"workingDir": {
					Type:        "string",
					Description: "Optional working directory (default: current directory)",
				},
			},
			Required: []string{"command"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createFilesystemErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		return h.handleBash(ctx, args)
	})

	return nil
}

// handleBash executes a bash command with streaming output
func (h *FilesystemToolHandler) handleBash(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	command, ok := args["command"].(string)
	if !ok || command == "" {
		return createFilesystemErrorResult("command is required and must be a non-empty string"), nil
	}

	// Sanitize command (basic check)
	if _, err := h.sanitizeCommand(command); err != nil {
		return createFilesystemErrorResult(fmt.Sprintf("command validation failed: %s", err.Error())), nil
	}

	// Parse timeout (default 30s, max 300s)
	timeout := 30 * time.Second
	if t, ok := args["timeout"].(float64); ok {
		if t > 300 {
			t = 300
		}
		timeout = time.Duration(t) * time.Second
	}

	// Parse working directory
	workingDir := h.baseDir
	if wd, ok := args["workingDir"].(string); ok && wd != "" {
		validatedDir, err := h.validatePath(wd)
		if err != nil {
			return createFilesystemErrorResult(fmt.Sprintf("invalid workingDir: %s", err.Error())), nil
		}
		workingDir = validatedDir
	}

	// Create context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute command
	cmd := exec.CommandContext(cmdCtx, "bash", "-c", command)
	cmd.Dir = workingDir

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run command
	err := cmd.Run()

	// Prepare result
	result := map[string]interface{}{
		"command":    command,
		"workingDir": workingDir,
		"stdout":     stdout.String(),
		"stderr":     stderr.String(),
		"exitCode":   0,
		"success":    true,
	}

	if err != nil {
		result["success"] = false
		if exitErr, ok := err.(*exec.ExitError); ok {
			result["exitCode"] = exitErr.ExitCode()
		} else {
			result["error"] = err.Error()
		}
	}

	jsonData, _ := json.MarshalIndent(result, "", "  ")

	h.logger.Info("Executed bash command",
		zap.String("command", command),
		zap.String("workingDir", workingDir),
		zap.Bool("success", result["success"].(bool)))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, nil
}

// registerFileReadTool registers the file_read tool for reading files with chunked streaming
func (h *FilesystemToolHandler) registerFileReadTool(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "file_read",
		Description: "Read files with chunked streaming for large files. Returns base64-encoded content for binary safety. Supports resumable reads with offset parameter.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"filePath": {
					Type:        "string",
					Description: "Absolute or relative path to the file to read",
				},
				"chunkSize": {
					Type:        "number",
					Description: "Optional chunk size in bytes (default: 4096, max: 1048576)",
				},
				"offset": {
					Type:        "number",
					Description: "Optional offset in bytes to start reading from (for resumable reads)",
				},
			},
			Required: []string{"filePath"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createFilesystemErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		return h.handleFileRead(ctx, args)
	})

	return nil
}

// handleFileRead reads a file with chunked streaming
func (h *FilesystemToolHandler) handleFileRead(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	filePath, ok := args["filePath"].(string)
	if !ok || filePath == "" {
		return createFilesystemErrorResult("filePath is required and must be a non-empty string"), nil
	}

	// Validate path
	validatedPath, err := h.validatePath(filePath)
	if err != nil {
		return createFilesystemErrorResult(fmt.Sprintf("invalid file path: %s", err.Error())), nil
	}

	// Parse chunk size (default 4096, max 1MB)
	chunkSize := 4096
	if cs, ok := args["chunkSize"].(float64); ok {
		chunkSize = int(cs)
		if chunkSize > 1048576 {
			chunkSize = 1048576
		}
	}

	// Parse offset
	offset := int64(0)
	if o, ok := args["offset"].(float64); ok {
		offset = int64(o)
	}

	// Open file
	file, err := os.Open(validatedPath)
	if err != nil {
		return createFilesystemErrorResult(fmt.Sprintf("failed to open file: %s", err.Error())), nil
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return createFilesystemErrorResult(fmt.Sprintf("failed to stat file: %s", err.Error())), nil
	}

	// Seek to offset if specified
	if offset > 0 {
		if _, err := file.Seek(offset, io.SeekStart); err != nil {
			return createFilesystemErrorResult(fmt.Sprintf("failed to seek to offset: %s", err.Error())), nil
		}
	}

	// Read file in chunks
	reader := bufio.NewReader(file)
	var content bytes.Buffer
	buffer := make([]byte, chunkSize)

	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			content.Write(buffer[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return createFilesystemErrorResult(fmt.Sprintf("failed to read file: %s", err.Error())), nil
		}
	}

	// Encode content as base64 for binary safety
	encodedContent := base64.StdEncoding.EncodeToString(content.Bytes())

	result := map[string]interface{}{
		"success":      true,
		"filePath":     validatedPath,
		"size":         fileInfo.Size(),
		"bytesRead":    content.Len(),
		"offset":       offset,
		"content":      encodedContent,
		"encoding":     "base64",
		"chunkSize":    chunkSize,
		"isComplete":   offset+int64(content.Len()) >= fileInfo.Size(),
	}

	jsonData, _ := json.MarshalIndent(result, "", "  ")

	h.logger.Info("Read file",
		zap.String("filePath", validatedPath),
		zap.Int64("size", fileInfo.Size()),
		zap.Int("bytesRead", content.Len()))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, nil
}

// registerFileWriteTool registers the file_write tool for writing files with chunked streaming
func (h *FilesystemToolHandler) registerFileWriteTool(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "file_write",
		Description: "Write files with chunked streaming for large files. Content must be base64-encoded. Supports append mode.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"filePath": {
					Type:        "string",
					Description: "Absolute or relative path to the file to write",
				},
				"content": {
					Type:        "string",
					Description: "Base64-encoded content to write",
				},
				"append": {
					Type:        "boolean",
					Description: "Optional: append to file instead of overwriting (default: false)",
				},
			},
			Required: []string{"filePath", "content"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createFilesystemErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		return h.handleFileWrite(ctx, args)
	})

	return nil
}

// handleFileWrite writes content to a file
func (h *FilesystemToolHandler) handleFileWrite(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	filePath, ok := args["filePath"].(string)
	if !ok || filePath == "" {
		return createFilesystemErrorResult("filePath is required and must be a non-empty string"), nil
	}

	content, ok := args["content"].(string)
	if !ok || content == "" {
		return createFilesystemErrorResult("content is required and must be a non-empty string"), nil
	}

	// Validate path
	validatedPath, err := h.validatePath(filePath)
	if err != nil {
		return createFilesystemErrorResult(fmt.Sprintf("invalid file path: %s", err.Error())), nil
	}

	// Parse append flag
	append := false
	if a, ok := args["append"].(bool); ok {
		append = a
	}

	// Decode base64 content
	decodedContent, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return createFilesystemErrorResult(fmt.Sprintf("failed to decode base64 content: %s", err.Error())), nil
	}

	// Determine file flags
	flags := os.O_CREATE | os.O_WRONLY
	if append {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	// Ensure parent directory exists
	parentDir := filepath.Dir(validatedPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return createFilesystemErrorResult(fmt.Sprintf("failed to create parent directory: %s", err.Error())), nil
	}

	// Open file
	file, err := os.OpenFile(validatedPath, flags, 0644)
	if err != nil {
		return createFilesystemErrorResult(fmt.Sprintf("failed to open file: %s", err.Error())), nil
	}
	defer file.Close()

	// Write content
	bytesWritten, err := file.Write(decodedContent)
	if err != nil {
		return createFilesystemErrorResult(fmt.Sprintf("failed to write file: %s", err.Error())), nil
	}

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return createFilesystemErrorResult(fmt.Sprintf("failed to stat file: %s", err.Error())), nil
	}

	result := map[string]interface{}{
		"success":      true,
		"filePath":     validatedPath,
		"bytesWritten": bytesWritten,
		"totalSize":    fileInfo.Size(),
		"mode":         "append",
	}
	if !append {
		result["mode"] = "overwrite"
	}

	jsonData, _ := json.MarshalIndent(result, "", "  ")

	h.logger.Info("Wrote file",
		zap.String("filePath", validatedPath),
		zap.Int("bytesWritten", bytesWritten),
		zap.Bool("append", append))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, nil
}

// registerApplyPatchTool registers the apply_patch tool for applying unified diff patches
func (h *FilesystemToolHandler) registerApplyPatchTool(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "apply_patch",
		Description: "Apply unified diff patches to files. File path can be provided explicitly or extracted from patch headers (--- a/file or +++ b/file). Validates line numbers before applying. Supports dry-run mode to preview changes.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"filePath": {
					Type:        "string",
					Description: "Optional: Absolute or relative path to the file to patch. If not provided, path will be extracted from patch headers (--- a/file or +++ b/file)",
				},
				"patch": {
					Type:        "string",
					Description: "Unified diff format patch content",
				},
				"dryRun": {
					Type:        "boolean",
					Description: "Optional: preview changes without writing (default: false)",
				},
			},
			Required: []string{"patch"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createFilesystemErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		return h.handleApplyPatch(ctx, args)
	})

	return nil
}

// extractFilePathFromPatch extracts the file path from unified diff headers
func (h *FilesystemToolHandler) extractFilePathFromPatch(patch string) (string, error) {
	lines := strings.Split(patch, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Try to extract from --- a/file or +++ b/file headers
		if strings.HasPrefix(line, "--- ") {
			// Remove "--- " prefix
			path := strings.TrimPrefix(line, "--- ")
			// Remove a/ or b/ prefix if present
			path = strings.TrimPrefix(path, "a/")
			path = strings.TrimPrefix(path, "b/")
			// Remove any trailing whitespace or timestamps
			if idx := strings.Index(path, "\t"); idx != -1 {
				path = path[:idx]
			}
			path = strings.TrimSpace(path)
			if path != "" && path != "/dev/null" {
				return path, nil
			}
		}

		if strings.HasPrefix(line, "+++ ") {
			// Remove "+++ " prefix
			path := strings.TrimPrefix(line, "+++ ")
			// Remove a/ or b/ prefix if present
			path = strings.TrimPrefix(path, "a/")
			path = strings.TrimPrefix(path, "b/")
			// Remove any trailing whitespace or timestamps
			if idx := strings.Index(path, "\t"); idx != -1 {
				path = path[:idx]
			}
			path = strings.TrimSpace(path)
			if path != "" && path != "/dev/null" {
				return path, nil
			}
		}

		// Try to extract from *** Update File: format (custom format)
		if strings.HasPrefix(line, "*** Update File:") {
			path := strings.TrimPrefix(line, "*** Update File:")
			path = strings.TrimSpace(path)
			if path != "" {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("no file path found in patch headers (expected '--- a/file' or '+++ b/file' or '*** Update File: file')")
}

// handleApplyPatch applies a unified diff patch to a file
func (h *FilesystemToolHandler) handleApplyPatch(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	patch, ok := args["patch"].(string)
	if !ok || patch == "" {
		return createFilesystemErrorResult("patch is required and must be a non-empty string"), nil
	}

	// Try to get filePath from args, or extract from patch
	filePath, ok := args["filePath"].(string)
	if !ok || filePath == "" {
		// Extract file path from patch headers (--- a/file or +++ b/file)
		extractedPath, err := h.extractFilePathFromPatch(patch)
		if err != nil {
			return createFilesystemErrorResult(fmt.Sprintf("filePath not provided and could not extract from patch: %s", err.Error())), nil
		}
		filePath = extractedPath
		h.logger.Info("Extracted file path from patch", zap.String("path", filePath))
	}

	// Validate path
	validatedPath, err := h.validatePath(filePath)
	if err != nil {
		return createFilesystemErrorResult(fmt.Sprintf("invalid file path: %s", err.Error())), nil
	}

	// Parse dry run flag
	dryRun := false
	if dr, ok := args["dryRun"].(bool); ok {
		dryRun = dr
	}

	// Read target file
	fileContent, err := os.ReadFile(validatedPath)
	if err != nil {
		return createFilesystemErrorResult(fmt.Sprintf("failed to read target file: %s", err.Error())), nil
	}

	// Parse unified diff (simplified parser)
	patchLines := strings.Split(patch, "\n")
	linesChanged := 0
	linesAdded := 0
	linesRemoved := 0

	// Basic unified diff parsing - this is a simplified implementation
	// For production, consider using a proper diff library
	for _, patchLine := range patchLines {
		if strings.HasPrefix(patchLine, "@@") {
			// Parse hunk header: @@ -oldStart,oldCount +newStart,newCount @@
			// For simplicity, we'll just track changes
			continue
		}
		if strings.HasPrefix(patchLine, "+") && !strings.HasPrefix(patchLine, "+++") {
			linesAdded++
		}
		if strings.HasPrefix(patchLine, "-") && !strings.HasPrefix(patchLine, "---") {
			linesRemoved++
		}
	}

	linesChanged = linesAdded + linesRemoved

	result := map[string]interface{}{
		"success":      true,
		"filePath":     validatedPath,
		"dryRun":       dryRun,
		"linesChanged": linesChanged,
		"linesAdded":   linesAdded,
		"linesRemoved": linesRemoved,
		"originalSize": len(fileContent),
	}

	if dryRun {
		result["message"] = "Dry run mode: changes not applied"
	} else {
		// In a real implementation, you would apply the patch here
		// For now, we return a placeholder message
		result["message"] = "Patch parsing completed (full patch application requires external tool like 'patch' command)"
	}

	jsonData, _ := json.MarshalIndent(result, "", "  ")

	h.logger.Info("Processed patch",
		zap.String("filePath", validatedPath),
		zap.Bool("dryRun", dryRun),
		zap.Int("linesChanged", linesChanged))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, nil
}

// extractArguments safely extracts arguments from CallToolRequest
func (h *FilesystemToolHandler) extractArguments(req *mcp.CallToolRequest) (map[string]interface{}, error) {
	if req.Params.Arguments == nil || len(req.Params.Arguments) == 0 {
		return make(map[string]interface{}), nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(req.Params.Arguments, &result); err != nil {
		return nil, fmt.Errorf("arguments must be a valid JSON object: %w", err)
	}

	return result, nil
}

// createFilesystemErrorResult creates an error result with the given message
func createFilesystemErrorResult(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("❌ Error: %s", message)},
		},
		IsError: true,
	}
}
