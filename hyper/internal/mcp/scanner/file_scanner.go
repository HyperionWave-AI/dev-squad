package scanner

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"hyper/internal/mcp/parser"
	"hyper/internal/mcp/storage"
)

// FileScanner scans directories for code files
type FileScanner struct {
	supportedExtensions map[string]string // extension -> language
	maxFileSize         int64             // max file size in bytes
	chunkSize           int               // lines per chunk
	includePatterns     []string          // custom include patterns (e.g., ["*.go", "*.ts"])
	excludePatterns     []string          // custom exclude patterns (e.g., ["node_modules", "dist"])
}

// NewFileScanner creates a new file scanner
func NewFileScanner() *FileScanner {
	// Initialize parsers (safe to call multiple times due to sync.Once)
	parser.InitializeParsers()

	// Get chunk size from ENV var (default: 200 lines)
	chunkSize := 200
	if envChunkSize := os.Getenv("CODE_INDEX_CHUNK_SIZE"); envChunkSize != "" {
		if parsedSize, err := strconv.Atoi(envChunkSize); err == nil && parsedSize > 0 {
			chunkSize = parsedSize
		}
	}

	return &FileScanner{
		supportedExtensions: map[string]string{
			".go":   "go",
			".js":   "javascript",
			".ts":   "typescript",
			".jsx":  "javascript",
			".tsx":  "typescript",
			".py":   "python",
			".java": "java",
			".c":    "c",
			".cpp":  "cpp",
			".h":    "c",
			".hpp":  "cpp",
			".cs":   "csharp",
			".rb":   "ruby",
			".php":  "php",
			".rs":   "rust",
			".swift": "swift",
			".kt":   "kotlin",
			".m":    "objective-c",
			".scala": "scala",
			".r":    "r",
			".sql":  "sql",
			".sh":   "shell",
			".bash": "shell",
			".yaml": "yaml",
			".yml":  "yaml",
			".json": "json",
			".xml":  "xml",
			".html": "html",
			".css":  "css",
			".scss": "scss",
			".less": "less",
			".vue":  "vue",
			".md":   "markdown",
		},
		maxFileSize: 10 * 1024 * 1024, // 10 MB
		chunkSize:   chunkSize,         // Configurable via CODE_INDEX_CHUNK_SIZE env var
	}
}

// NewFileScannerWithConfig creates a file scanner with custom configuration
func NewFileScannerWithConfig(includePatterns, excludePatterns []string, chunkSize string) *FileScanner {
	// Initialize parsers (safe to call multiple times due to sync.Once)
	parser.InitializeParsers()

	// Convert t-shirt size to line count
	chunkLines := storage.ChunkSizeToLines(chunkSize)

	fs := &FileScanner{
		supportedExtensions: map[string]string{
			".go":   "go",
			".js":   "javascript",
			".ts":   "typescript",
			".jsx":  "javascript",
			".tsx":  "typescript",
			".py":   "python",
			".java": "java",
			".c":    "c",
			".cpp":  "cpp",
			".h":    "c",
			".hpp":  "cpp",
			".cs":   "csharp",
			".rb":   "ruby",
			".php":  "php",
			".rs":   "rust",
			".swift": "swift",
			".kt":   "kotlin",
			".m":    "objective-c",
			".scala": "scala",
			".r":    "r",
			".sql":  "sql",
			".sh":   "shell",
			".bash": "shell",
			".yaml": "yaml",
			".yml":  "yaml",
			".json": "json",
			".xml":  "xml",
			".html": "html",
			".css":  "css",
			".scss": "scss",
			".less": "less",
			".vue":  "vue",
			".md":   "markdown",
		},
		maxFileSize:     10 * 1024 * 1024, // 10 MB
		chunkSize:       chunkLines,
		includePatterns: includePatterns,
		excludePatterns: excludePatterns,
	}

	return fs
}

// shouldExcludePath checks if a path should be excluded based on patterns
func (fs *FileScanner) shouldExcludePath(path string, basePath string) bool {
	// If no custom exclude patterns, use defaults
	excludePatterns := fs.excludePatterns
	if len(excludePatterns) == 0 {
		excludePatterns = []string{"node_modules", "dist", "build", ".git", "vendor", ".next", "coverage", "__pycache__", ".vscode", ".idea", "test-results", "out"}
	}

	// Get relative path for pattern matching
	relativePath, err := filepath.Rel(basePath, path)
	if err != nil {
		relativePath = path
	}

	// Check each exclude pattern
	for _, pattern := range excludePatterns {
		// Check if path contains the pattern (simple substring match for directories)
		if strings.Contains(relativePath, pattern) {
			return true
		}
		// Also try glob match
		matched, _ := filepath.Match(pattern, filepath.Base(path))
		if matched {
			return true
		}
	}

	return false
}

// shouldIncludeFile checks if a file should be included based on patterns
func (fs *FileScanner) shouldIncludeFile(filePath string) bool {
	// If custom patterns are set, use them
	if len(fs.includePatterns) > 0 {
		fileName := filepath.Base(filePath)
		for _, pattern := range fs.includePatterns {
			matched, _ := filepath.Match(pattern, fileName)
			if matched {
				return true
			}
		}
		return false
	}

	// Otherwise, use extension-based matching
	ext := filepath.Ext(filePath)
	_, supported := fs.supportedExtensions[ext]
	return supported
}

// ScanDirectory scans a directory and returns file information
func (fs *FileScanner) ScanDirectory(folderPath string) ([]*storage.IndexedFile, error) {
	var files []*storage.IndexedFile

	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories that match exclude patterns
		if info.IsDir() {
			if fs.shouldExcludePath(path, folderPath) {
				return filepath.SkipDir
			}

			// Skip archived directories (CRITICAL FIX)
			dirName := filepath.Base(path)
			if dirName == ".archived" || dirName == ".archive" || dirName == "archived" {
				return filepath.SkipDir
			}

			// Skip paths containing archived directories
			if strings.Contains(path, "/.archived/") || strings.Contains(path, "/.archive/") {
				return filepath.SkipDir
			}

			return nil
		}

		// Check if file should be excluded
		if fs.shouldExcludePath(path, folderPath) {
			return nil
		}

		// Check if file should be included based on patterns
		if !fs.shouldIncludeFile(path) {
			return nil
		}

		// Determine language from extension
		ext := filepath.Ext(path)
		language, supported := fs.supportedExtensions[ext]
		if !supported {
			// For custom patterns, default to file extension without dot
			language = strings.TrimPrefix(ext, ".")
			if language == "" {
				language = "unknown"
			}
		}

		// Check file size
		if info.Size() > fs.maxFileSize {
			return nil
		}

		// Calculate SHA-256 hash
		hash, err := fs.calculateSHA256(path)
		if err != nil {
			return fmt.Errorf("failed to calculate hash for %s: %w", path, err)
		}

		// Count lines
		lineCount, err := fs.countLines(path)
		if err != nil {
			return fmt.Errorf("failed to count lines for %s: %w", path, err)
		}

		// Calculate relative path
		relativePath, err := filepath.Rel(folderPath, path)
		if err != nil {
			relativePath = path
		}

		// Calculate chunk count
		chunkCount := (lineCount + fs.chunkSize - 1) / fs.chunkSize
		if chunkCount == 0 {
			chunkCount = 1
		}

		file := &storage.IndexedFile{
			Path:         path,
			RelativePath: relativePath,
			Language:     language,
			SHA256:       hash,
			Size:         info.Size(),
			LineCount:    lineCount,
			ChunkCount:   chunkCount,
		}

		files = append(files, file)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return files, nil
}

// ReadFileChunks reads a file and returns it in chunks
func (fs *FileScanner) ReadFileChunks(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var chunks []string
	var currentChunk strings.Builder
	var lineCount int

	scanner := bufio.NewScanner(file)

	// Increase buffer size to 1MB to handle minified files with very long lines
	const maxCapacity = 1024 * 1024 // 1 MB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		currentChunk.WriteString(scanner.Text())
		currentChunk.WriteString("\n")
		lineCount++

		if lineCount >= fs.chunkSize {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
			lineCount = 0
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	// Add remaining content as last chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	// If no chunks, add empty chunk
	if len(chunks) == 0 {
		chunks = append(chunks, "")
	}

	return chunks, nil
}

// CreateFileChunks creates FileChunk objects for a file
// Uses AST parsing if available, with fallback to line-based chunking
func (fs *FileScanner) CreateFileChunks(fileID, filePath string) ([]*storage.FileChunk, error) {
	// Check if AST parsing is enabled (default: true)
	useAST := true
	if envUseAST := os.Getenv("CODE_INDEX_USE_AST"); envUseAST == "false" {
		useAST = false
	}

	// Check if AST fallback is enabled (default: true)
	astFallback := true
	if envFallback := os.Getenv("CODE_INDEX_AST_FALLBACK"); envFallback == "false" {
		astFallback = false
	}

	// Try AST parsing first if enabled
	if useAST {
		registry := parser.GetRegistry()
		if registry.HasParserForFile(filePath) {
			astChunks, err := fs.createASTChunks(fileID, filePath)
			if err == nil && len(astChunks) > 0 {
				// AST parsing succeeded
				return astChunks, nil
			}
			// AST parsing failed - fall through to line-based if fallback enabled
			if !astFallback {
				return nil, fmt.Errorf("AST parsing failed and fallback disabled: %w", err)
			}
		}
	}

	// Fallback to line-based chunking
	return fs.createLineBasedChunks(fileID, filePath)
}

// createASTChunks creates chunks using AST parsing with enhanced metadata
func (fs *FileScanner) createASTChunks(fileID, filePath string) ([]*storage.FileChunk, error) {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Get parser for this file
	registry := parser.GetRegistry()
	astParser, err := registry.GetParserForFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("no parser available: %w", err)
	}

	// Parse the file
	nodes, err := astParser.Parse(filePath, content)
	if err != nil {
		return nil, fmt.Errorf("AST parsing failed: %w", err)
	}

	// Convert AST nodes to file chunks with enhanced metadata
	var fileChunks []*storage.FileChunk
	for i, node := range nodes {
		chunk := &storage.FileChunk{
			FileID:    fileID,
			ChunkNum:  i,
			Content:   node.Content,
			StartLine: node.StartLine,
			EndLine:   node.EndLine,
			ChunkType: "ast",
			NodeType:  string(node.Type),
			NodeName:  node.Name,
			Signature: node.Signature,
		}

		// Extract enhanced metadata from node.Metadata map
		if node.Metadata != nil {
			// Extract symbols
			if symbols, ok := node.Metadata["symbols"].([]string); ok {
				chunk.Symbols = symbols
			}

			// Extract imports
			if imports, ok := node.Metadata["imports"].([]string); ok {
				chunk.Imports = imports
			}

			// Extract docstring information
			if hasDoc, ok := node.Metadata["hasDocstring"].(bool); ok && hasDoc {
				chunk.HasDocstring = true
				if docContent, ok := node.Metadata["docContent"].(string); ok {
					chunk.DocContent = docContent
				}
			}
		}

		fileChunks = append(fileChunks, chunk)
	}

	return fileChunks, nil
}

// createLineBasedChunks creates chunks using traditional line-based chunking
func (fs *FileScanner) createLineBasedChunks(fileID, filePath string) ([]*storage.FileChunk, error) {
	chunks, err := fs.ReadFileChunks(filePath)
	if err != nil {
		return nil, err
	}

	var fileChunks []*storage.FileChunk
	currentLine := 1

	for i, content := range chunks {
		lines := strings.Count(content, "\n")
		if lines == 0 && content != "" {
			lines = 1
		}

		chunk := &storage.FileChunk{
			FileID:    fileID,
			ChunkNum:  i,
			Content:   content,
			StartLine: currentLine,
			EndLine:   currentLine + lines - 1,
			ChunkType: "line-based",
		}

		fileChunks = append(fileChunks, chunk)
		currentLine += lines
	}

	return fileChunks, nil
}

// calculateSHA256 calculates the SHA-256 hash of a file
func (fs *FileScanner) calculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// countLines counts the number of lines in a file
func (fs *FileScanner) countLines(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Increase buffer size to 1MB to handle minified files with very long lines
	const maxCapacity = 1024 * 1024 // 1 MB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	lineCount := 0

	for scanner.Scan() {
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		// If still failing, skip this file instead of crashing the entire scan
		return 0, fmt.Errorf("file contains lines too long to process: %w", err)
	}

	return lineCount, nil
}

// IsFileChanged checks if a file has changed based on SHA-256 hash
func (fs *FileScanner) IsFileChanged(filePath string, oldHash string) (bool, error) {
	newHash, err := fs.calculateSHA256(filePath)
	if err != nil {
		return false, err
	}

	return newHash != oldHash, nil
}

// ChunkContent represents a chunk of file content with line numbers
type ChunkContent struct {
	Content   string
	StartLine int
	EndLine   int
}

// FileInfo represents scanned file information with chunks
type FileInfo struct {
	Path         string
	RelativePath string
	Language     string
	SHA256       string
	Size         int64
	LineCount    int
	Chunks       []ChunkContent
}

// ScanFile scans a single file and returns its information with chunks
func ScanFile(filePath string, basePath string) (*FileInfo, error) {
	fs := NewFileScanner()

	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Check if file extension is supported
	ext := filepath.Ext(filePath)
	language, supported := fs.supportedExtensions[ext]
	if !supported {
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}

	// Check file size
	if info.Size() > fs.maxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes (max %d)", info.Size(), fs.maxFileSize)
	}

	// Calculate SHA-256 hash
	hash, err := fs.calculateSHA256(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate hash: %w", err)
	}

	// Count lines
	lineCount, err := fs.countLines(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to count lines: %w", err)
	}

	// Calculate relative path
	relativePath, err := filepath.Rel(basePath, filePath)
	if err != nil {
		relativePath = filePath
	}

	// Read file chunks
	chunkTexts, err := fs.ReadFileChunks(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read chunks: %w", err)
	}

	// Create chunk content with line numbers
	var chunks []ChunkContent
	currentLine := 1

	for _, chunkText := range chunkTexts {
		lines := strings.Count(chunkText, "\n")
		if lines == 0 && chunkText != "" {
			lines = 1
		}

		chunks = append(chunks, ChunkContent{
			Content:   chunkText,
			StartLine: currentLine,
			EndLine:   currentLine + lines - 1,
		})

		currentLine += lines
	}

	return &FileInfo{
		Path:         filePath,
		RelativePath: relativePath,
		Language:     language,
		SHA256:       hash,
		Size:         info.Size(),
		LineCount:    lineCount,
		Chunks:       chunks,
	}, nil
}

// IsCodeFile checks if a file is a supported code file
func IsCodeFile(filePath string) bool {
	fs := NewFileScanner()
	ext := filepath.Ext(filePath)
	_, supported := fs.supportedExtensions[ext]
	return supported
}
