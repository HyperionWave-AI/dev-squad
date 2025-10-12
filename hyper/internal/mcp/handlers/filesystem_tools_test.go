package handlers

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestFilesystemToolHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewFilesystemToolHandler(logger)
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0.0"}, &mcp.ServerOptions{HasTools: true})

	// Register tools
	err := handler.RegisterFilesystemTools(server)
	require.NoError(t, err)
}

func TestValidatePath(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewFilesystemToolHandler(logger)

	tests := []struct {
		name      string
		path      string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "valid absolute path",
			path:      "/tmp/test.txt",
			expectErr: false,
		},
		{
			name:      "valid relative path",
			path:      "test.txt",
			expectErr: false,
		},
		{
			name:      "directory traversal attempt with ..",
			path:      "../../../etc/passwd",
			expectErr: true,
			errMsg:    "directory traversal detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := handler.validatePath(tt.path)
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSanitizeCommand(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewFilesystemToolHandler(logger)

	tests := []struct {
		name      string
		command   string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "simple command",
			command:   "echo hello",
			expectErr: false,
		},
		{
			name:      "command with semicolon",
			command:   "echo hello; rm -rf /",
			expectErr: true,
			errMsg:    "dangerous pattern",
		},
		{
			name:      "command with pipe",
			command:   "cat file | grep pattern",
			expectErr: true,
			errMsg:    "dangerous pattern",
		},
		{
			name:      "command with &&",
			command:   "echo hello && rm file",
			expectErr: true,
			errMsg:    "dangerous pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := handler.sanitizeCommand(tt.command)
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHandleFileReadWrite(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewFilesystemToolHandler(logger)

	// Create temp directory
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"

	// Test file write
	t.Run("file_write", func(t *testing.T) {
		// Encode content as base64
		encodedContent := base64.StdEncoding.EncodeToString([]byte(testContent))

		args := map[string]interface{}{
			"path":    testFile,
			"content": encodedContent,
			"append":  false,
		}

		result, err := handler.handleFileWrite(context.Background(), args)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)

		// Verify file exists
		_, err = os.Stat(testFile)
		assert.NoError(t, err)
	})

	// Test file read
	t.Run("file_read", func(t *testing.T) {
		args := map[string]interface{}{
			"path":      testFile,
			"chunkSize": 4096,
		}

		result, err := handler.handleFileRead(context.Background(), args)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)

		// Content should be base64-encoded
		assert.NotEmpty(t, result.Content)
	})

	// Test file append
	t.Run("file_write_append", func(t *testing.T) {
		appendContent := " Appended text."
		encodedContent := base64.StdEncoding.EncodeToString([]byte(appendContent))

		args := map[string]interface{}{
			"path":    testFile,
			"content": encodedContent,
			"append":  true,
		}

		result, err := handler.handleFileWrite(context.Background(), args)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)

		// Read file and verify appended content
		fileContent, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, testContent+appendContent, string(fileContent))
	})
}

func TestHandleBash(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewFilesystemToolHandler(logger)

	t.Run("simple command", func(t *testing.T) {
		args := map[string]interface{}{
			"command": "echo hello",
		}

		result, err := handler.handleBash(context.Background(), args)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)
	})

	t.Run("command with timeout", func(t *testing.T) {
		args := map[string]interface{}{
			"command": "echo test",
			"timeout": float64(5),
		}

		result, err := handler.handleBash(context.Background(), args)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)
	})

	t.Run("invalid command", func(t *testing.T) {
		args := map[string]interface{}{
			"command": "echo hello; rm -rf /",
		}

		result, err := handler.handleBash(context.Background(), args)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.IsError)
	})
}

func TestHandleApplyPatch(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewFilesystemToolHandler(logger)

	// Create temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := "line 1\nline 2\nline 3\n"
	err := os.WriteFile(testFile, []byte(originalContent), 0644)
	require.NoError(t, err)

	t.Run("parse valid patch", func(t *testing.T) {
		patch := `--- a/test.txt
+++ b/test.txt
@@ -1,3 +1,3 @@
 line 1
-line 2
+line 2 modified
 line 3
`

		args := map[string]interface{}{
			"path":   testFile,
			"patch":  patch,
			"dryRun": true,
		}

		result, err := handler.handleApplyPatch(context.Background(), args)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)
	})

	t.Run("dry run mode", func(t *testing.T) {
		patch := `@@ -1,1 +1,1 @@
-line 1
+line 1 changed
`

		args := map[string]interface{}{
			"path":   testFile,
			"patch":  patch,
			"dryRun": true,
		}

		result, err := handler.handleApplyPatch(context.Background(), args)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)

		// Verify original file unchanged
		content, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, originalContent, string(content))
	})
}

func TestHandleFileReadLargeFile(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewFilesystemToolHandler(logger)

	// Create temp file with > 10KB content
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")

	// Generate 20KB of content
	largeContent := make([]byte, 20*1024)
	for i := range largeContent {
		largeContent[i] = byte('A' + (i % 26))
	}
	err := os.WriteFile(testFile, largeContent, 0644)
	require.NoError(t, err)

	t.Run("read large file with default chunk size", func(t *testing.T) {
		args := map[string]interface{}{
			"path": testFile,
		}

		result, err := handler.handleFileRead(context.Background(), args)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)
	})

	t.Run("read large file with custom chunk size", func(t *testing.T) {
		args := map[string]interface{}{
			"path":      testFile,
			"chunkSize": float64(8192),
		}

		result, err := handler.handleFileRead(context.Background(), args)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)
	})

	t.Run("read with offset", func(t *testing.T) {
		args := map[string]interface{}{
			"path":   testFile,
			"offset": float64(1024),
		}

		result, err := handler.handleFileRead(context.Background(), args)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)
	})
}
