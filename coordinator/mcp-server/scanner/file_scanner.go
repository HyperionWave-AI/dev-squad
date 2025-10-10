package scanner

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"hyperion-coordinator-mcp/storage"
)

// FileScanner scans directories for code files
type FileScanner struct {
	supportedExtensions map[string]string // extension -> language
	maxFileSize         int64             // max file size in bytes
	chunkSize           int               // lines per chunk
}

// NewFileScanner creates a new file scanner
func NewFileScanner() *FileScanner {
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
		chunkSize:   200,               // 200 lines per chunk
	}
}

// ScanDirectory scans a directory and returns file information
func (fs *FileScanner) ScanDirectory(folderPath string) ([]*storage.IndexedFile, error) {
	var files []*storage.IndexedFile

	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Skip common directories to ignore
			dirName := filepath.Base(path)
			if dirName == ".git" || dirName == "node_modules" || dirName == "vendor" ||
				dirName == "dist" || dirName == "build" || dirName == ".vscode" ||
				dirName == ".idea" || dirName == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file extension is supported
		ext := filepath.Ext(path)
		language, supported := fs.supportedExtensions[ext]
		if !supported {
			return nil
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
func (fs *FileScanner) CreateFileChunks(fileID, filePath string) ([]*storage.FileChunk, error) {
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
	lineCount := 0

	for scanner.Scan() {
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return 0, err
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
