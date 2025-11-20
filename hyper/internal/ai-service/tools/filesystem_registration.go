package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	aiservice "hyper/internal/ai-service"
	"hyper/internal/mcp/storage"
	"hyper/internal/validation"

	"go.uber.org/zap"
)

// BashToolExecutor adapts BashTool to ToolExecutor interface
type BashToolExecutor struct {
	tool *BashTool
}

func (b *BashToolExecutor) Name() string {
	return "bash"
}

func (b *BashToolExecutor) Description() string {
	return "Execute shell commands and return stdout/stderr. Supports timeout (default 30s). Use for system operations, file checks, script execution."
}

func (b *BashToolExecutor) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "Shell command to execute",
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"description": "Timeout in seconds (default: 30)",
			},
		},
		"required": []string{"command"},
	}
}

func (b *BashToolExecutor) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Convert map to JSON string for BashTool.Call()
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	// Call the underlying tool
	result, err := b.tool.Call(ctx, string(inputJSON))
	if err != nil {
		return nil, err
	}

	// Parse result back to interface{} for ToolExecutor
	var output interface{}
	if err := json.Unmarshal([]byte(result), &output); err != nil {
		return nil, fmt.Errorf("failed to parse tool output: %w", err)
	}

	return output, nil
}

// ReadFileToolExecutor adapts ReadFileTool to ToolExecutor interface
type ReadFileToolExecutor struct {
	tool *ReadFileTool
}

func (r *ReadFileToolExecutor) Name() string {
	return "read_file"
}

func (r *ReadFileToolExecutor) Description() string {
	return "Read file contents from the filesystem. Returns file content as string. Max 10MB file size. Supports text and binary files with encoding detection."
}

func (r *ReadFileToolExecutor) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Absolute or relative file path to read",
			},
		},
		"required": []string{"path"},
	}
}

func (r *ReadFileToolExecutor) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	result, err := r.tool.Call(ctx, string(inputJSON))
	if err != nil {
		return nil, err
	}

	var output interface{}
	if err := json.Unmarshal([]byte(result), &output); err != nil {
		return nil, fmt.Errorf("failed to parse tool output: %w", err)
	}

	return output, nil
}

// WriteFileToolExecutor adapts WriteFileTool to ToolExecutor interface
type WriteFileToolExecutor struct {
	tool        *WriteFileTool
	validator   *validation.CodeValidator
	taskStorage storage.TaskStorage
}

func (w *WriteFileToolExecutor) Name() string {
	return "write_file"
}

func (w *WriteFileToolExecutor) Description() string {
	return "Write content to a file on the filesystem. Creates parent directories if needed. Max 5MB content size. Atomic write (temp + rename)."
}

func (w *WriteFileToolExecutor) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Absolute or relative file path to write",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Content to write to file",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (w *WriteFileToolExecutor) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	result, err := w.tool.Call(ctx, string(inputJSON))
	if err != nil {
		return nil, err
	}

	var output interface{}
	if err := json.Unmarshal([]byte(result), &output); err != nil {
		return nil, fmt.Errorf("failed to parse tool output: %w", err)
	}

	// Run SYNCHRONOUS validation after successful write - BLOCKS until validation completes
	if filePath, ok := input["path"].(string); ok && filePath != "" && w.validator != nil {
		// Filter to only validate TypeScript/Go files
		ext := filepath.Ext(filePath)
		if ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx" || ext == ".go" {
			zap.L().Info("🔍 Running SYNCHRONOUS post-write validation", zap.String("file", filePath))

			// Check for incomplete/placeholder code FIRST
			fileContent, readErr := os.ReadFile(filePath)
			if readErr == nil {
				contentStr := string(fileContent)

				// Detect placeholder comments
				placeholderPatterns := []string{
					"// rest of",
					"// ... (",
					"...rest of",
					"// (rest",
					"/* rest of",
					"# rest of",
				}

				for _, pattern := range placeholderPatterns {
					if strings.Contains(strings.ToLower(contentStr), strings.ToLower(pattern)) {
						zap.L().Error("❌ INCOMPLETE CODE - Placeholder detected",
							zap.String("file", filePath),
							zap.String("pattern", pattern))
						return nil, fmt.Errorf("❌ INCOMPLETE CODE - PLACEHOLDER DETECTED:\nFile contains placeholder '%s'.\n\nWrite the COMPLETE file, not fragments.", pattern)
					}
				}

				// For React components, check required structure
				if ext == ".tsx" || ext == ".jsx" {
					hasImport := strings.Contains(contentStr, "import")
					hasExport := strings.Contains(contentStr, "export")
					hasReact := strings.Contains(contentStr, "React") || strings.Contains(contentStr, "react")

					if !hasImport || !hasExport || !hasReact {
						zap.L().Error("❌ INCOMPLETE REACT COMPONENT",
							zap.String("file", filePath),
							zap.Bool("hasImport", hasImport),
							zap.Bool("hasExport", hasExport),
							zap.Bool("hasReact", hasReact))
						return nil, fmt.Errorf("❌ INCOMPLETE REACT COMPONENT:\nMissing: imports=%v, exports=%v, React=%v\n\nWrite COMPLETE component with all imports and exports.", !hasImport, !hasExport, !hasReact)
					}
				}
			}

			// Create independent context
			validationCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Run compilation validation
			validationResult, err := w.validator.ValidateFiles(validationCtx, []string{filePath})
			if err != nil {
				zap.L().Warn("⚠️  Validation failed to run", zap.Error(err))
			} else if !validationResult.Passed {
				zap.L().Error("❌ COMPILATION ERRORS DETECTED",
					zap.Int("errorCount", len(validationResult.Errors)),
					zap.String("file", filePath))

				errorMsg := w.validator.FormatErrorsForAgent(validationResult)
				zap.L().Error("🚨 COMPILATION ERRORS", zap.String("errors", errorMsg))

				return nil, fmt.Errorf("❌ COMPILATION FAILED:\n%s\n\nFix these errors before proceeding.", errorMsg)
			} else {
				zap.L().Info("✅ Validation passed", zap.String("file", filePath))
			}
		}
	}

	return output, nil
}

// ListDirectoryToolExecutor adapts ListDirectoryTool to ToolExecutor interface
type ListDirectoryToolExecutor struct {
	tool *ListDirectoryTool
}

func (l *ListDirectoryToolExecutor) Name() string {
	return "list_directory"
}

func (l *ListDirectoryToolExecutor) Description() string {
	return "List directory contents with file metadata. Returns absolute paths for all entries. Supports recursive mode. Max 1000 entries. Returns: name, path (absolute), size, isDir, modTime, permissions."
}

func (l *ListDirectoryToolExecutor) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Directory path to list",
			},
			// "recursive": map[string]interface{}{
			// 	"type":        "boolean",
			// 	"description": "List subdirectories recursively (default: false)",
			// },
		},
		"required": []string{"path"},
	}
}

func (l *ListDirectoryToolExecutor) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	result, err := l.tool.Call(ctx, string(inputJSON))
	if err != nil {
		return nil, err
	}

	var output interface{}
	if err := json.Unmarshal([]byte(result), &output); err != nil {
		return nil, fmt.Errorf("failed to parse tool output: %w", err)
	}

	return output, nil
}

// ApplyPatchToolExecutor adapts ApplyPatchTool to ToolExecutor interface
type ApplyPatchToolExecutor struct {
	tool        *ApplyPatchTool
	validator   *validation.CodeValidator
	taskStorage storage.TaskStorage
}

func (a *ApplyPatchToolExecutor) Name() string {
	return "apply_patch"
}

func (a *ApplyPatchToolExecutor) Description() string {
	return "Apply unified diff patches to files. Supports dry-run mode for validation. Handles multi-file patches and line-by-line hunk application."
}

func (a *ApplyPatchToolExecutor) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Absolute or relative file path to apply patch to",
			},
			"patch": map[string]interface{}{
				"type":        "string",
				"description": "Unified diff format patch content",
			},
			"dryRun": map[string]interface{}{
				"type":        "boolean",
				"description": "Validate patch without applying (default: false)",
			},
		},
		"required": []string{"path", "patch"},
	}
}

func (a *ApplyPatchToolExecutor) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	result, err := a.tool.Call(ctx, string(inputJSON))
	if err != nil {
		return nil, err
	}

	var output interface{}
	if err := json.Unmarshal([]byte(result), &output); err != nil {
		return nil, fmt.Errorf("failed to parse tool output: %w", err)
	}

	// Run SYNCHRONOUS validation after successful patch (if not dry-run) - BLOCKS until validation completes
	dryRun, _ := input["dryRun"].(bool)
	if !dryRun {
		if filePath, ok := input["path"].(string); ok && filePath != "" && a.validator != nil {
			// Filter to only validate TypeScript/Go files
			ext := filepath.Ext(filePath)
			if ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx" || ext == ".go" {
				zap.L().Info("🔍 Running SYNCHRONOUS post-patch validation", zap.String("file", filePath))

				// Create independent context that doesn't die with HTTP request
				validationCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				// Run validation synchronously
				validationResult, err := a.validator.ValidateFiles(validationCtx, []string{filePath})
				if err != nil {
					zap.L().Warn("⚠️  Validation failed to run", zap.Error(err))
					// Don't block on validation infrastructure errors
				} else if !validationResult.Passed {
					zap.L().Error("❌ COMPILATION ERRORS DETECTED",
						zap.Int("errorCount", len(validationResult.Errors)),
						zap.String("file", filePath))

					// Format errors for agent
					errorMsg := a.validator.FormatErrorsForAgent(validationResult)
					zap.L().Error("🚨 COMPILATION ERRORS", zap.String("errors", errorMsg))

					// BLOCK the tool call - return error so agent must fix before proceeding
					return nil, fmt.Errorf("❌ COMPILATION FAILED:\n%s\n\nYou MUST fix these errors before proceeding. The patch was applied but contains compilation errors", errorMsg)
				} else {
					zap.L().Info("✅ Compilation validation passed", zap.String("file", filePath))
				}
			}
		}
	}

	return output, nil
}

// RegisterFilesystemTools registers all filesystem tools with the tool registry
// Tools: bash, read_file, write_file, list_directory, apply_patch
func RegisterFilesystemTools(registry *aiservice.ToolRegistry, validator *validation.CodeValidator, taskStorage storage.TaskStorage) error {
	tools := []aiservice.ToolExecutor{
		&BashToolExecutor{tool: &BashTool{}},
		&ReadFileToolExecutor{tool: &ReadFileTool{}},
		&WriteFileToolExecutor{
			tool:        &WriteFileTool{},
			validator:   validator,
			taskStorage: taskStorage,
		},
		&ListDirectoryToolExecutor{tool: &ListDirectoryTool{}},
		&ApplyPatchToolExecutor{
			tool:        &ApplyPatchTool{},
			validator:   validator,
			taskStorage: taskStorage,
		},
	}

	for _, tool := range tools {
		if err := registry.Register(tool); err != nil {
			return fmt.Errorf("failed to register %s: %w", tool.Name(), err)
		}
	}

	return nil
}
