package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	maxReadSize  = 10 * 1024 * 1024 // 10MB
	maxWriteSize = 5 * 1024 * 1024  // 5MB
	maxDirEntries = 1000
)

// ReadFileTool reads file contents with encoding detection
type ReadFileTool struct{}

// ReadFileInput represents the input schema for file reading
type ReadFileInput struct {
	Path     string `json:"path"`
	MaxBytes int    `json:"maxBytes,omitempty"` // max bytes to read, default 10MB
}

// ReadFileOutput represents the output from file reading
type ReadFileOutput struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Size     int64  `json:"size"`
	Encoding string `json:"encoding"` // utf8 or binary
}

// Name returns the tool name
func (r *ReadFileTool) Name() string {
	return "readFile"
}

// Description returns the tool description
func (r *ReadFileTool) Description() string {
	return "Read file contents with automatic encoding detection. Supports files up to 10MB. Returns file metadata including size and encoding (UTF-8 or binary)."
}

// Call reads the file
func (r *ReadFileTool) Call(ctx context.Context, input string) (string, error) {
	var readInput ReadFileInput
	if err := json.Unmarshal([]byte(input), &readInput); err != nil {
		return "", fmt.Errorf("invalid input format: %w", err)
	}

	// Validate path
	if err := validatePath(readInput.Path); err != nil {
		return "", err
	}

	// Check file exists and get info
	info, err := os.Stat(readInput.Path)
	if err != nil {
		return "", fmt.Errorf("file not found: %w", err)
	}

	if info.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file")
	}

	// Set max bytes
	maxBytes := readInput.MaxBytes
	if maxBytes == 0 || maxBytes > maxReadSize {
		maxBytes = maxReadSize
	}

	// Check size limit
	if info.Size() > int64(maxBytes) {
		return "", fmt.Errorf("file size %d exceeds limit %d bytes", info.Size(), maxBytes)
	}

	// Read file
	content, err := os.ReadFile(readInput.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Detect encoding
	encoding := "binary"
	if utf8.Valid(content) {
		encoding = "utf8"
	}

	output := ReadFileOutput{
		Path:     readInput.Path,
		Content:  string(content),
		Size:     info.Size(),
		Encoding: encoding,
	}

	result, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("failed to marshal output: %w", err)
	}

	return string(result), nil
}

// WriteFileTool writes file contents with safety checks
type WriteFileTool struct{}

// WriteFileInput represents the input schema for file writing
type WriteFileInput struct {
	Path       string `json:"path"`
	Content    string `json:"content"`
	CreateDirs bool   `json:"createDirs,omitempty"` // create parent directories if they don't exist
}

// WriteFileOutput represents the output from file writing
type WriteFileOutput struct {
	Path         string `json:"path"`
	BytesWritten int    `json:"bytesWritten"`
}

// Name returns the tool name
func (w *WriteFileTool) Name() string {
	return "writeFile"
}

// Description returns the tool description
func (w *WriteFileTool) Description() string {
	return "Create or overwrite file with content. Supports atomic writes (temp file + rename). Max content size 5MB. Can create parent directories if needed."
}

// Call writes the file
func (w *WriteFileTool) Call(ctx context.Context, input string) (string, error) {
	var writeInput WriteFileInput
	if err := json.Unmarshal([]byte(input), &writeInput); err != nil {
		return "", fmt.Errorf("invalid input format: %w", err)
	}

	// Validate path
	if err := validatePath(writeInput.Path); err != nil {
		return "", err
	}

	// Check content size
	contentBytes := []byte(writeInput.Content)
	if len(contentBytes) > maxWriteSize {
		return "", fmt.Errorf("content size %d exceeds limit %d bytes", len(contentBytes), maxWriteSize)
	}

	// Create parent directories if needed
	if writeInput.CreateDirs {
		dir := filepath.Dir(writeInput.Path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directories: %w", err)
		}
	}

	// Atomic write: write to temp file then rename
	tempFile := writeInput.Path + ".tmp"
	if err := os.WriteFile(tempFile, contentBytes, 0644); err != nil {
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tempFile, writeInput.Path); err != nil {
		os.Remove(tempFile) // cleanup on error
		return "", fmt.Errorf("failed to rename temp file: %w", err)
	}

	output := WriteFileOutput{
		Path:         writeInput.Path,
		BytesWritten: len(contentBytes),
	}

	result, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("failed to marshal output: %w", err)
	}

	return string(result), nil
}

// ListDirectoryTool lists files and folders with metadata
type ListDirectoryTool struct{}

// ListDirectoryInput represents the input schema for directory listing
type ListDirectoryInput struct {
	Path       string `json:"path"`
	Recursive  bool   `json:"recursive,omitempty"`
	ShowHidden bool   `json:"showHidden,omitempty"`
}

// FileInfo represents a file or directory entry
type FileInfo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Size        int64  `json:"size"`
	IsDir       bool   `json:"isDir"`
	ModTime     string `json:"modTime"`
	Permissions string `json:"permissions"`
}

// ListDirectoryOutput represents the output from directory listing
type ListDirectoryOutput struct {
	Path    string     `json:"path"`
	Entries []FileInfo `json:"entries"`
	Count   int        `json:"count"`
}

// Name returns the tool name
func (l *ListDirectoryTool) Name() string {
	return "listDirectory"
}

// Description returns the tool description
func (l *ListDirectoryTool) Description() string {
	return "List files and directories with metadata (name, size, type, permissions, modification time). Supports recursive listing and hidden files. Max 1000 entries."
}

// Call lists the directory
func (l *ListDirectoryTool) Call(ctx context.Context, input string) (string, error) {
	var listInput ListDirectoryInput
	if err := json.Unmarshal([]byte(input), &listInput); err != nil {
		return "", fmt.Errorf("invalid input format: %w", err)
	}

	// Validate path
	if err := validatePath(listInput.Path); err != nil {
		return "", err
	}

	// Check path exists and is directory
	info, err := os.Stat(listInput.Path)
	if err != nil {
		return "", fmt.Errorf("path not found: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory")
	}

	var entries []FileInfo

	if listInput.Recursive {
		// Recursive walk
		err = filepath.WalkDir(listInput.Path, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// Skip hidden files if not requested
			if !listInput.ShowHidden && strings.HasPrefix(d.Name(), ".") && path != listInput.Path {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// Get file info
			info, err := d.Info()
			if err != nil {
				return err
			}

			entries = append(entries, FileInfo{
				Name:        d.Name(),
				Path:        path,
				Size:        info.Size(),
				IsDir:       d.IsDir(),
				ModTime:     info.ModTime().Format("2006-01-02 15:04:05"),
				Permissions: info.Mode().String(),
			})

			// Limit entries
			if len(entries) >= maxDirEntries {
				return io.EOF
			}

			return nil
		})

		if err != nil && err != io.EOF {
			return "", fmt.Errorf("failed to walk directory: %w", err)
		}
	} else {
		// Flat listing
		dirEntries, err := os.ReadDir(listInput.Path)
		if err != nil {
			return "", fmt.Errorf("failed to read directory: %w", err)
		}

		for _, d := range dirEntries {
			// Skip hidden files if not requested
			if !listInput.ShowHidden && strings.HasPrefix(d.Name(), ".") {
				continue
			}

			info, err := d.Info()
			if err != nil {
				continue
			}

			fullPath := filepath.Join(listInput.Path, d.Name())
			entries = append(entries, FileInfo{
				Name:        d.Name(),
				Path:        fullPath,
				Size:        info.Size(),
				IsDir:       d.IsDir(),
				ModTime:     info.ModTime().Format("2006-01-02 15:04:05"),
				Permissions: info.Mode().String(),
			})

			// Limit entries
			if len(entries) >= maxDirEntries {
				break
			}
		}
	}

	// Sort alphabetically
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	output := ListDirectoryOutput{
		Path:    listInput.Path,
		Entries: entries,
		Count:   len(entries),
	}

	result, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("failed to marshal output: %w", err)
	}

	return string(result), nil
}

// validatePath validates file paths for security
func validatePath(path string) error {
	// Path must not be empty
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Check for path traversal before conversion
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal not allowed")
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check allowed directories (if ALLOWED_DIRS env var is set)
	allowedDirs := os.Getenv("ALLOWED_DIRS")
	if allowedDirs != "" {
		allowed := false
		for _, dir := range strings.Split(allowedDirs, ":") {
			if strings.HasPrefix(absPath, dir) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("path outside allowed directories")
		}
	}

	return nil
}
