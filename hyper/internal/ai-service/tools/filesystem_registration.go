package tools

import (
	"context"
	"encoding/json"
	"fmt"

	aiservice "hyper/internal/ai-service"
	"hyper/internal/validation"

	"go.uber.org/zap"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// extractModifiedFilesFromBashCommand detects which files are being modified by a bash command
func extractModifiedFilesFromBashCommand(command string) []string {
	var modifiedFiles []string
	seenFiles := make(map[string]bool)

	// Pattern 1: sed -i (in-place edit)
	// Examples: sed -i 's/foo/bar/' file.ts, sed -i.bak 's/foo/bar/' file.go
	sedPattern := regexp.MustCompile(`sed\s+(?:-[a-zA-Z]*i[a-zA-Z]*|--in-place(?:=[^\s]*)?)(?:\s+[^/\s][^\s]*)*\s+([^\s]+\.(?:ts|tsx|js|jsx|go))`)
	if matches := sedPattern.FindAllStringSubmatch(command, -1); matches != nil {
		for _, match := range matches {
			if len(match) > 1 {
				file := match[1]
				if !seenFiles[file] {
					modifiedFiles = append(modifiedFiles, file)
					seenFiles[file] = true
				}
			}
		}
	}

	// Pattern 2: Output redirection (>, >>)
	// Examples: echo "code" > file.ts, cat file1 > file2.go
	redirectPattern := regexp.MustCompile(`(?:^|[\s;&|])[^>]*>\s*([^\s;&|]+\.(?:ts|tsx|js|jsx|go))`)
	if matches := redirectPattern.FindAllStringSubmatch(command, -1); matches != nil {
		for _, match := range matches {
			if len(match) > 1 {
				file := strings.TrimSpace(match[1])
				// Skip if it's part of a comparison (e.g., if [ "$x" > "$y" ])
				if !strings.Contains(match[0], "[") && !seenFiles[file] {
					modifiedFiles = append(modifiedFiles, file)
					seenFiles[file] = true
				}
			}
		}
	}

	// Pattern 3: tee command
	// Examples: echo "code" | tee file.ts
	teePattern := regexp.MustCompile(`tee\s+(?:-a\s+)?([^\s;&|]+\.(?:ts|tsx|js|jsx|go))`)
	if matches := teePattern.FindAllStringSubmatch(command, -1); matches != nil {
		for _, match := range matches {
			if len(match) > 1 {
				file := match[1]
				if !seenFiles[file] {
					modifiedFiles = append(modifiedFiles, file)
					seenFiles[file] = true
				}
			}
		}
	}

	// Pattern 4: cat with heredoc (cat << EOF > file.ts)
	heredocPattern := regexp.MustCompile(`cat\s*<<[^\s]+\s*>\s*([^\s;&|]+\.(?:ts|tsx|js|jsx|go))`)
	if matches := heredocPattern.FindAllStringSubmatch(command, -1); matches != nil {
		for _, match := range matches {
			if len(match) > 1 {
				file := match[1]
				if !seenFiles[file] {
					modifiedFiles = append(modifiedFiles, file)
					seenFiles[file] = true
				}
			}
		}
	}

	// Pattern 5: Common text editors (vim, nano, emacs - less common but possible)
	editorPattern := regexp.MustCompile(`(?:vim|nano|emacs)\s+([^\s;&|]+\.(?:ts|tsx|js|jsx|go))`)
	if matches := editorPattern.FindAllStringSubmatch(command, -1); matches != nil {
		for _, match := range matches {
			if len(match) > 1 {
				file := match[1]
				if !seenFiles[file] {
					modifiedFiles = append(modifiedFiles, file)
					seenFiles[file] = true
				}
			}
		}
	}

	// Convert relative paths to absolute paths based on project root
	projectRoot := GetProjectRoot()
	for i, file := range modifiedFiles {
		if !filepath.IsAbs(file) {
			modifiedFiles[i] = filepath.Join(projectRoot, file)
		}
	}

	return modifiedFiles
}

// BashToolExecutor adapts BashTool to ToolExecutor interface
type BashToolExecutor struct {
	tool      *BashTool
	validator *validation.CodeValidator
	logger    *zap.Logger
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
	// Extract command for validation detection
	command, _ := input["command"].(string)

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

	// SYNCHRONOUS post-execution validation (only if error prevention mode is enabled)
	errorPreventionMode := ctx.Value(aiservice.ErrorPreventionModeKey)
	isErrorPreventionEnabled := errorPreventionMode != nil && errorPreventionMode.(bool)

	if b.logger != nil {
		b.logger.Debug("Bash command executed",
			zap.String("command", command),
			zap.Bool("errorPreventionMode", isErrorPreventionEnabled))
	}

	if isErrorPreventionEnabled && b.validator != nil && b.logger != nil {
		// Detect file modifications in the command
		modifiedFiles := extractModifiedFilesFromBashCommand(command)

		if len(modifiedFiles) > 0 {
			b.logger.Info("🔍 Detected file modifications in bash command - running validation",
				zap.Strings("files", modifiedFiles),
				zap.String("command", command))

			validationCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			validationResult, validationErr := b.validator.ValidateFiles(validationCtx, modifiedFiles)
			if validationErr != nil {
				b.logger.Warn("⚠️  Validation failed to run", zap.Error(validationErr))
			} else if !validationResult.Passed {
				b.logger.Error("❌ COMPILATION ERRORS DETECTED after bash command",
					zap.Int("errorCount", len(validationResult.Errors)),
					zap.Strings("files", modifiedFiles),
					zap.String("command", command))

				errorMsg := b.validator.FormatErrorsForAgent(validationResult)
				b.logger.Error("🚨 COMPILATION ERRORS", zap.String("errors", errorMsg))

				return "", fmt.Errorf("❌ COMPILATION FAILED after bash command:\n%s\n\nYou MUST fix these errors before proceeding. Command was: %s", errorMsg, command)
			} else {
				b.logger.Info("✅ Compilation validation passed after bash command",
					zap.Strings("files", modifiedFiles))
			}
		}
	}

	// Return raw stdout string directly (bash now returns raw output, not JSON)
	return result, nil
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

	// Return raw file content directly (read_file returns raw content, not JSON)
	// This matches the tool's design: "makes it easier for AI to process file contents directly"
	return result, nil
}

// WriteFileToolExecutor adapts WriteFileTool to ToolExecutor interface
type WriteFileToolExecutor struct {
	tool *WriteFileTool
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
	tool      *ApplyPatchTool
	validator *validation.CodeValidator
	logger    *zap.Logger
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
	// Extract path and dryRun for validation
	filePath, _ := input["path"].(string)
	dryRun, _ := input["dryRun"].(bool)

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

	// SYNCHRONOUS post-patch validation (only if error prevention mode is enabled and not a dry-run)
	errorPreventionMode := ctx.Value(aiservice.ErrorPreventionModeKey)
	isErrorPreventionEnabled := errorPreventionMode != nil && errorPreventionMode.(bool)

	if a.logger != nil {
		a.logger.Debug("Patch applied",
			zap.String("path", filePath),
			zap.Bool("dryRun", dryRun),
			zap.Bool("errorPreventionMode", isErrorPreventionEnabled))
	}

	// Only validate if:
	// 1. Error prevention is enabled
	// 2. Not a dry-run (dry-run doesn't modify files)
	// 3. File is a code file (.ts, .tsx, .js, .jsx, .go)
	if isErrorPreventionEnabled && !dryRun && a.validator != nil && a.logger != nil && filePath != "" {
		ext := filepath.Ext(filePath)
		if ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx" || ext == ".go" {
			a.logger.Info("🔍 Running post-patch validation (Error Prevention Mode: ON)",
				zap.String("file", filePath))

			// Convert relative path to absolute if needed
			projectRoot := GetProjectRoot()
			absPath := filePath
			if !filepath.IsAbs(filePath) {
				absPath = filepath.Join(projectRoot, filePath)
			}

			validationCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			validationResult, validationErr := a.validator.ValidateFiles(validationCtx, []string{absPath})
			if validationErr != nil {
				a.logger.Warn("⚠️  Validation failed to run", zap.Error(validationErr))
			} else if !validationResult.Passed {
				a.logger.Error("❌ COMPILATION ERRORS DETECTED after patch",
					zap.Int("errorCount", len(validationResult.Errors)),
					zap.String("file", filePath))

				errorMsg := a.validator.FormatErrorsForAgent(validationResult)
				a.logger.Error("🚨 COMPILATION ERRORS", zap.String("errors", errorMsg))

				return nil, fmt.Errorf("❌ COMPILATION FAILED after applying patch:\n%s\n\nYou MUST fix these errors before proceeding. File: %s", errorMsg, filePath)
			} else {
				a.logger.Info("✅ Compilation validation passed after patch",
					zap.String("file", filePath))
			}
		}
	}

	return output, nil
}

// RegisterFilesystemTools registers all filesystem tools with the tool registry
// Tools: bash, read_file, write_file, list_directory, apply_patch
func RegisterFilesystemTools(registry *aiservice.ToolRegistry, validator *validation.CodeValidator, logger *zap.Logger) error {
	tools := []aiservice.ToolExecutor{
		&BashToolExecutor{tool: &BashTool{}, validator: validator, logger: logger},
		&ReadFileToolExecutor{tool: &ReadFileTool{}},
		&WriteFileToolExecutor{tool: &WriteFileTool{validator: validator, logger: logger}},
		&ListDirectoryToolExecutor{tool: &ListDirectoryTool{}},
		&ApplyPatchToolExecutor{tool: &ApplyPatchTool{}, validator: validator, logger: logger},
	}

	for _, tool := range tools {
		if err := registry.Register(tool); err != nil {
			return fmt.Errorf("failed to register %s: %w", tool.Name(), err)
		}
	}

	return nil
}
