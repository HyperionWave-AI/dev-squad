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
	FilePath string `json:"path"`
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
	return "Read file contents. Use RELATIVE PATHS (e.g., ./src/main.go) or virtual paths (/ = project root). Supports files up to 10MB. System directories not accessible."
}

// Call reads the file
func (r *ReadFileTool) Call(ctx context.Context, input string) (string, error) {
	var readInput ReadFileInput
	if err := json.Unmarshal([]byte(input), &readInput); err != nil {
		return "", fmt.Errorf("invalid input format: %w", err)
	}

	// Validate path and convert to absolute
	absPath, err := validatePath(readInput.FilePath)
	if err != nil {
		return "", err
	}
	readInput.FilePath = absPath

	// Check file exists and get info
	info, err := os.Stat(readInput.FilePath)
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
	content, err := os.ReadFile(readInput.FilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Detect encoding
	encoding := "binary"
	if utf8.Valid(content) {
		encoding = "utf8"
	}

	output := ReadFileOutput{
		Path:     StripProjectRoot(readInput.FilePath),
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
	FilePath   string `json:"path"`
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
	return "Create or overwrite file. Use RELATIVE PATHS (e.g., ./test.txt) or virtual paths (/ = project root). Atomic writes. Max 5MB."
}

// Call writes the file
func (w *WriteFileTool) Call(ctx context.Context, input string) (string, error) {
	var writeInput WriteFileInput
	if err := json.Unmarshal([]byte(input), &writeInput); err != nil {
		return "", fmt.Errorf("invalid input format: %w", err)
	}

	// Validate path and convert to absolute (don't check existence for write)
	absPath, err := validatePathForWrite(writeInput.FilePath)
	if err != nil {
		return "", err
	}
	writeInput.FilePath = absPath

	// Check content size
	contentBytes := []byte(writeInput.Content)
	if len(contentBytes) > maxWriteSize {
		return "", fmt.Errorf("content size %d exceeds limit %d bytes", len(contentBytes), maxWriteSize)
	}

	// Create parent directories if needed
	if writeInput.CreateDirs {
		dir := filepath.Dir(writeInput.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directories: %w", err)
		}
	}

	// Atomic write: write to temp file then rename
	tempFile := writeInput.FilePath + ".tmp"
	if err := os.WriteFile(tempFile, contentBytes, 0644); err != nil {
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tempFile, writeInput.FilePath); err != nil {
		os.Remove(tempFile) // cleanup on error
		return "", fmt.Errorf("failed to rename temp file: %w", err)
	}

	output := WriteFileOutput{
		Path:         StripProjectRoot(writeInput.FilePath),
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
	Recursive  bool   `json:"recursive,omitempty"`  // Recursive listing (default: false)
	ShowHidden bool   `json:"showHidden,omitempty"` // Show hidden files starting with . (default: false)
	Offset     int    `json:"offset,omitempty"`     // Starting index for pagination (default: 0)
	MaxResults int    `json:"maxResults,omitempty"` // Maximum number of results to return (default: 100, max: 1000)
	FileMask   string `json:"fileMask,omitempty"`   // Filter pattern (e.g., "*.js", "*.md", "test*") (default: all files)
}

// FileInfo represents a file or directory entry
type FileInfo struct {
	Name        string `json:"name"`        // File or directory name (without path)
	Path        string `json:"path"`        // Absolute path to the file or directory
	Size        int64  `json:"size"`        // Size in bytes (0 for directories)
	IsDir       bool   `json:"isDir"`       // True if entry is a directory
	ModTime     string `json:"modTime"`     // Modification time (YYYY-MM-DD HH:MM:SS)
	Permissions string `json:"permissions"` // Unix permissions string (e.g., drwxr-xr-x)
}

// ListDirectoryOutput represents the output from directory listing
type ListDirectoryOutput struct {
	Directory string   `json:"directory"` // Absolute path to the directory
	Count     int      `json:"count"`     // Total number of files/directories
	Summary   string   `json:"summary"`   // Human-readable summary (e.g., "Showing 1-100 of 523 files")
	Files     []string `json:"files"`     // Array of file/directory names (paginated)
}

// Name returns the tool name
func (l *ListDirectoryTool) Name() string {
	return "listDirectory"
}

// Description returns the tool description
func (l *ListDirectoryTool) Description() string {
	return "List files. Use RELATIVE PATHS (e.g., ./src) or virtual paths (/ = project root). Returns file/directory names only (compact format). Supports pagination with 'offset' and 'maxResults' (default: 100, max: 1000). Optional 'fileMask' filter (e.g., '*.js', '*.md', 'test*') to match specific files. Recursive defaults to false. Use offset for pagination (0, 100, 200, etc.)."
}

// Call lists the directory
func (l *ListDirectoryTool) Call(ctx context.Context, input string) (string, error) {
	var listInput ListDirectoryInput
	if err := json.Unmarshal([]byte(input), &listInput); err != nil {
		return "", fmt.Errorf("invalid input format: %w", err)
	}

	// Set defaults for pagination
	if listInput.MaxResults <= 0 {
		listInput.MaxResults = 100
	}
	if listInput.MaxResults > maxDirEntries {
		listInput.MaxResults = maxDirEntries
	}
	if listInput.Offset < 0 {
		listInput.Offset = 0
	}

	// Validate path and convert to absolute
	absPath, err := validatePath(listInput.Path)
	if err != nil {
		return "", err
	}
	listInput.Path = absPath

	// Check path exists and is directory
	info, err := os.Stat(listInput.Path)
	if err != nil {
		return "", fmt.Errorf("path not found: %w", err)
	}

	if !info.IsDir() {
		parentDir := filepath.Dir(listInput.Path)
		return "", fmt.Errorf("path '%s' is a FILE, not a directory. To read file contents, use read_file tool with this path. To list directory, provide the parent directory path: %s", listInput.Path, parentDir)
	}

	// Collect all file names (not full metadata)
	var allNames []string

	if listInput.Recursive {
		// Recursive walk - collect up to maxDirEntries total
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

			// Apply file mask filter if provided
			if listInput.FileMask != "" {
				matched, err := filepath.Match(listInput.FileMask, d.Name())
				if err != nil || !matched {
					return nil // Skip non-matching files
				}
			}

			// Add name only
			allNames = append(allNames, d.Name())

			// Hard limit to prevent memory issues
			if len(allNames) >= maxDirEntries {
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

			// Apply file mask filter if provided
			if listInput.FileMask != "" {
				matched, err := filepath.Match(listInput.FileMask, d.Name())
				if err != nil || !matched {
					continue // Skip non-matching files
				}
			}

			allNames = append(allNames, d.Name())

			// Hard limit to prevent memory issues
			if len(allNames) >= maxDirEntries {
				break
			}
		}
	}

	// Sort alphabetically
	sort.Strings(allNames)

	// Apply pagination
	totalCount := len(allNames)
	start := listInput.Offset
	if start > totalCount {
		start = totalCount
	}

	end := start + listInput.MaxResults
	if end > totalCount {
		end = totalCount
	}

	paginatedNames := allNames[start:end]

	// Create summary message
	summary := fmt.Sprintf("Showing %d-%d of %d files", start+1, end, totalCount)
	if totalCount == 0 {
		summary = "Directory is empty"
	} else if start >= totalCount {
		summary = fmt.Sprintf("Offset %d exceeds total count %d", start, totalCount)
	}

	output := ListDirectoryOutput{
		Directory: StripProjectRoot(listInput.Path),
		Count:     totalCount,
		Summary:   summary,
		Files:     paginatedNames,
	}

	result, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("failed to marshal output: %w", err)
	}

	return string(result), nil
}

// validatePath validates file paths for security and returns the absolute path
// This version checks that the path exists (for read/list operations)
func validatePath(path string) (string, error) {
	// Path must not be empty
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("path parameter is required. Provide absolute or relative file/directory path (e.g., /path/to/file.txt or ./relative/path)")
	}

	// Check for path traversal before conversion
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("path traversal (..) not allowed for security. Use absolute paths instead (e.g., /full/path/to/file)")
	}

	// Map absolute paths to project-relative
	mappedPath := MapPath(path)

	// Convert to absolute path (now relative to project root)
	absPath, err := filepath.Abs(mappedPath)
	if err != nil {
		return "", fmt.Errorf("invalid path format: %w. Provide a valid file or directory path", err)
	}

	// Check if path exists
	if _, err := os.Stat(absPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("path does not exist: '%s'. Verify the path is correct and the file/directory exists", StripProjectRoot(absPath))
		}
		return "", fmt.Errorf("cannot access path '%s': %w", StripProjectRoot(absPath), err)
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
			return "", fmt.Errorf("path '%s' is outside allowed directories. Access restricted to: %s", StripProjectRoot(absPath), allowedDirs)
		}
	}

	return absPath, nil
}

// validatePathForWrite validates file paths for write operations
// Unlike validatePath, this does NOT check existence (files may not exist yet)
func validatePathForWrite(path string) (string, error) {
	// Path must not be empty
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("path parameter is required. Provide absolute or relative file path (e.g., /path/to/file.txt or ./relative/path)")
	}

	// Check for path traversal before conversion
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("path traversal (..) not allowed for security. Use absolute paths instead (e.g., /full/path/to/file)")
	}

	// Map absolute paths to project-relative
	mappedPath := MapPath(path)

	// Convert to absolute path (now relative to project root)
	absPath, err := filepath.Abs(mappedPath)
	if err != nil {
		return "", fmt.Errorf("invalid path format: %w. Provide a valid file path", err)
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
			return "", fmt.Errorf("path '%s' is outside allowed directories. Access restricted to: %s", StripProjectRoot(absPath), allowedDirs)
		}
	}

	return absPath, nil
}
