package indexer

import (
	"context"
	"fmt"
	"path/filepath"

	"hyper/internal/mcp/embeddings"
	"hyper/internal/mcp/scanner"
	"hyper/internal/mcp/storage"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AutoIndexer handles automatic code indexing for project roots
type AutoIndexer struct {
	codeIndexStorage *storage.CodeIndexStorage
	qdrantClient     *storage.QdrantClient
	embeddingClient  embeddings.EmbeddingClient
	fileScanner      *scanner.FileScanner
	logger           *zap.Logger
}

// NewAutoIndexer creates a new auto-indexer instance
func NewAutoIndexer(
	codeIndexStorage *storage.CodeIndexStorage,
	qdrantClient     *storage.QdrantClient,
	embeddingClient embeddings.EmbeddingClient,
	logger *zap.Logger,
) *AutoIndexer {
	return &AutoIndexer{
		codeIndexStorage: codeIndexStorage,
		qdrantClient:     qdrantClient,
		embeddingClient:  embeddingClient,
		fileScanner:      scanner.NewFileScanner(),
		logger:           logger,
	}
}

// IndexResult contains the results of an indexing operation
type IndexResult struct {
	FilesIndexed int
	FilesUpdated int
	FilesSkipped int
	TotalFiles   int
	Error        error
}

// IndexProjectRoot indexes a project root directory
// This is the same logic as the MCP code_index_scan tool, but callable from main.go
func (a *AutoIndexer) IndexProjectRoot(ctx context.Context, projectRoot string, collectionName string) *IndexResult {
	result := &IndexResult{}

	// Get or create folder metadata
	folder, err := a.codeIndexStorage.GetFolderByPath(projectRoot)
	if err != nil || folder == nil {
		// Create folder if doesn't exist
		folder, err = a.codeIndexStorage.AddFolder(projectRoot, "Auto-indexed project root")
		if err != nil {
			result.Error = fmt.Errorf("failed to create folder metadata: %w", err)
			return result
		}
	}

	// Update folder status to scanning
	if err := a.codeIndexStorage.UpdateFolderStatus(folder.ID, "scanning", ""); err != nil {
		a.logger.Warn("Failed to update folder status", zap.Error(err))
	}

	// Scan directory for files
	a.logger.Info("Scanning directory for code files...",
		zap.String("path", projectRoot),
		zap.String("collection", collectionName))

	scannedFiles, err := a.fileScanner.ScanDirectory(projectRoot)
	if err != nil {
		a.codeIndexStorage.UpdateFolderStatus(folder.ID, "error", err.Error())
		result.Error = fmt.Errorf("failed to scan directory: %w", err)
		return result
	}

	result.TotalFiles = len(scannedFiles)
	a.logger.Info("Files discovered",
		zap.Int("count", len(scannedFiles)),
		zap.String("path", projectRoot))

	// Process each file
	for i, scannedFile := range scannedFiles {
		scannedFile.FolderID = folder.ID

		// Log progress every 10 files
		if (i+1)%10 == 0 || i+1 == len(scannedFiles) {
			a.logger.Info("Indexing progress",
				zap.Int("processed", i+1),
				zap.Int("total", len(scannedFiles)),
				zap.String("currentFile", scannedFile.RelativePath))
		}

		// Check if file already exists
		existingFile, _ := a.codeIndexStorage.GetFileByPath(scannedFile.Path)

		if existingFile != nil {
			// Check if file has changed
			if existingFile.SHA256 == scannedFile.SHA256 {
				result.FilesSkipped++
				continue
			}
			result.FilesUpdated++
			scannedFile.ID = existingFile.ID
		} else {
			result.FilesIndexed++
			scannedFile.ID = uuid.New().String()
		}

		// Create chunks
		chunks, err := a.fileScanner.CreateFileChunks(scannedFile.ID, scannedFile.Path)
		if err != nil {
			a.logger.Warn("Failed to create chunks",
				zap.String("file", scannedFile.Path),
				zap.Error(err))
			continue
		}

		// Generate embeddings for chunks WITH METADATA CONTEXT
		var qdrantPoints []storage.CodeIndexPoint
		for _, chunk := range chunks {
			// CRITICAL FIX: Add file metadata to embedding for better semantic search
			// Format: "File: TaskDashboard.tsx\nPath: ui/src/components/TaskDashboard.tsx\nLanguage: typescript\n\n[code]"
			fileName := filepath.Base(scannedFile.Path)
			contentWithContext := fmt.Sprintf("File: %s\nPath: %s\nLanguage: %s\n\n%s",
				fileName,
				scannedFile.RelativePath,
				scannedFile.Language,
				chunk.Content)

			// Generate embedding with context
			embedding, err := a.embeddingClient.CreateEmbedding(contentWithContext)
			if err != nil {
				a.logger.Warn("Failed to create embedding",
					zap.String("file", scannedFile.Path),
					zap.Int("chunk", chunk.ChunkNum),
					zap.Error(err))
				continue
			}

			// Create Qdrant point with unique UUID (Qdrant requires pure UUID or integer)
			pointID := uuid.New().String()
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
			if err := a.codeIndexStorage.UpsertChunk(chunk); err != nil {
				a.logger.Warn("Failed to save chunk", zap.Error(err))
			}
		}

		// Upload vectors to Qdrant
		if len(qdrantPoints) > 0 {
			if err := a.qdrantClient.UpsertCodeIndexPoints(collectionName, qdrantPoints); err != nil {
				a.logger.Warn("Failed to upsert vectors",
					zap.String("file", scannedFile.Path),
					zap.Error(err))
			}
		}

		// Save file metadata to MongoDB
		if err := a.codeIndexStorage.UpsertFile(scannedFile); err != nil {
			a.logger.Warn("Failed to save file", zap.Error(err))
		}
	}

	// Update folder status and scan time
	if err := a.codeIndexStorage.UpdateFolderStatus(folder.ID, "active", ""); err != nil {
		a.logger.Warn("Failed to update folder status", zap.Error(err))
	}

	if err := a.codeIndexStorage.UpdateFolderScanTime(folder.ID, len(scannedFiles)); err != nil {
		a.logger.Warn("Failed to update scan time", zap.Error(err))
	}

	a.logger.Info("Indexing complete",
		zap.String("path", projectRoot),
		zap.String("collection", collectionName),
		zap.Int("filesIndexed", result.FilesIndexed),
		zap.Int("filesUpdated", result.FilesUpdated),
		zap.Int("filesSkipped", result.FilesSkipped),
		zap.Int("totalFiles", result.TotalFiles))

	return result
}

// CheckCollectionEmpty checks if a Qdrant collection is empty
func (a *AutoIndexer) CheckCollectionEmpty(ctx context.Context, collectionName string) (bool, error) {
	// Get the actual point count from Qdrant
	pointCount, err := a.qdrantClient.GetCollectionPointCount(collectionName)
	if err != nil {
		return false, fmt.Errorf("failed to get collection point count: %w", err)
	}

	a.logger.Info("Collection point count check",
		zap.String("collection", collectionName),
		zap.Int("pointCount", pointCount))

	return pointCount == 0, nil
}
