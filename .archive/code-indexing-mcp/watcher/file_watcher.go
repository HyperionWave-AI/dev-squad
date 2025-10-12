package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"code-indexing-mcp/embeddings"
	"code-indexing-mcp/scanner"
	"code-indexing-mcp/storage"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

// FileWatcher monitors file system changes and triggers re-indexing
type FileWatcher struct {
	watcher         *fsnotify.Watcher
	mongoStorage    *storage.MongoStorage
	qdrantClient    *storage.QdrantClient
	embeddingClient *embeddings.OpenAIClient
	logger          *zap.Logger

	// Debouncing
	debounceTime    time.Duration
	debounceMutex   sync.Mutex
	debounceTimers  map[string]*time.Timer

	// Watched folders
	watchedFolders  map[string]*storage.IndexedFolder
	foldersMutex    sync.RWMutex

	// Control
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

// NewFileWatcher creates a new file watcher instance
func NewFileWatcher(
	mongoStorage *storage.MongoStorage,
	qdrantClient *storage.QdrantClient,
	embeddingClient *embeddings.OpenAIClient,
	logger *zap.Logger,
) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	fw := &FileWatcher{
		watcher:         watcher,
		mongoStorage:    mongoStorage,
		qdrantClient:    qdrantClient,
		embeddingClient: embeddingClient,
		logger:          logger,
		debounceTime:    500 * time.Millisecond,
		debounceTimers:  make(map[string]*time.Timer),
		watchedFolders:  make(map[string]*storage.IndexedFolder),
		ctx:             ctx,
		cancel:          cancel,
	}

	return fw, nil
}

// Start begins watching all indexed folders
func (fw *FileWatcher) Start() error {
	// Load all indexed folders from MongoDB
	folders, err := fw.mongoStorage.ListFolders()
	if err != nil {
		return fmt.Errorf("failed to list folders: %w", err)
	}

	// Add folders to watch list
	for _, folder := range folders {
		if folder.Status == "active" {
			if err := fw.AddFolder(folder); err != nil {
				fw.logger.Error("Failed to watch folder",
					zap.String("path", folder.Path),
					zap.Error(err))
				continue
			}
		}
	}

	// Start event processing goroutine
	fw.wg.Add(1)
	go fw.processEvents()

	fw.logger.Info("File watcher started",
		zap.Int("watchedFolders", len(fw.watchedFolders)))

	return nil
}

// Stop stops the file watcher
func (fw *FileWatcher) Stop() error {
	fw.cancel()
	fw.wg.Wait()

	if err := fw.watcher.Close(); err != nil {
		return fmt.Errorf("failed to close watcher: %w", err)
	}

	fw.logger.Info("File watcher stopped")
	return nil
}

// AddFolder adds a folder to the watch list
func (fw *FileWatcher) AddFolder(folder *storage.IndexedFolder) error {
	fw.foldersMutex.Lock()
	defer fw.foldersMutex.Unlock()

	// Check if already watching
	if _, exists := fw.watchedFolders[folder.Path]; exists {
		return nil
	}

	// Add folder to watcher
	if err := fw.watcher.Add(folder.Path); err != nil {
		return fmt.Errorf("failed to watch folder: %w", err)
	}

	// Walk directory and add all subdirectories (excluding ignored ones)
	err := filepath.Walk(folder.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() {
			// Skip ignored directories
			if fw.shouldIgnore(path) {
				return filepath.SkipDir
			}

			// Add subdirectory to watcher
			if err := fw.watcher.Add(path); err != nil {
				fw.logger.Debug("Failed to watch subdirectory",
					zap.String("path", path),
					zap.Error(err))
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	fw.watchedFolders[folder.Path] = folder

	fw.logger.Info("Added folder to watch list",
		zap.String("path", folder.Path),
		zap.String("folderId", folder.ID))

	return nil
}

// RemoveFolder removes a folder from the watch list
func (fw *FileWatcher) RemoveFolder(folderPath string) error {
	fw.foldersMutex.Lock()
	defer fw.foldersMutex.Unlock()

	folder, exists := fw.watchedFolders[folderPath]
	if !exists {
		return nil
	}

	// Remove folder from watcher
	if err := fw.watcher.Remove(folderPath); err != nil {
		fw.logger.Debug("Failed to remove folder from watcher",
			zap.String("path", folderPath),
			zap.Error(err))
	}

	// Remove all subdirectories
	filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info != nil && info.IsDir() {
			fw.watcher.Remove(path)
		}
		return nil
	})

	delete(fw.watchedFolders, folderPath)

	fw.logger.Info("Removed folder from watch list",
		zap.String("path", folderPath),
		zap.String("folderId", folder.ID))

	return nil
}

// processEvents processes file system events
func (fw *FileWatcher) processEvents() {
	defer fw.wg.Done()

	for {
		select {
		case <-fw.ctx.Done():
			return

		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			fw.handleEvent(event)

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			fw.logger.Error("Watcher error", zap.Error(err))
		}
	}
}

// handleEvent processes a single file system event
func (fw *FileWatcher) handleEvent(event fsnotify.Event) {
	// Skip ignored files
	if fw.shouldIgnore(event.Name) {
		return
	}

	fw.logger.Debug("File event received",
		zap.String("path", event.Name),
		zap.String("op", event.Op.String()))

	// Debounce the event
	fw.debounceEvent(event)
}

// debounceEvent implements debouncing for file events
func (fw *FileWatcher) debounceEvent(event fsnotify.Event) {
	fw.debounceMutex.Lock()
	defer fw.debounceMutex.Unlock()

	// Cancel existing timer for this file
	if timer, exists := fw.debounceTimers[event.Name]; exists {
		timer.Stop()
	}

	// Create new timer
	fw.debounceTimers[event.Name] = time.AfterFunc(fw.debounceTime, func() {
		fw.processFileEvent(event)

		// Clean up timer
		fw.debounceMutex.Lock()
		delete(fw.debounceTimers, event.Name)
		fw.debounceMutex.Unlock()
	})
}

// processFileEvent processes a debounced file event
func (fw *FileWatcher) processFileEvent(event fsnotify.Event) {
	// Find which folder this file belongs to
	folder := fw.findFolder(event.Name)
	if folder == nil {
		fw.logger.Debug("File does not belong to any watched folder",
			zap.String("path", event.Name))
		return
	}

	fw.logger.Info("Processing file event",
		zap.String("path", event.Name),
		zap.String("op", event.Op.String()),
		zap.String("folderId", folder.ID))

	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		fw.handleCreate(event.Name, folder)

	case event.Op&fsnotify.Write == fsnotify.Write:
		fw.handleUpdate(event.Name, folder)

	case event.Op&fsnotify.Remove == fsnotify.Remove:
		fw.handleDelete(event.Name, folder)

	case event.Op&fsnotify.Rename == fsnotify.Rename:
		fw.handleDelete(event.Name, folder)
	}
}

// handleCreate handles file creation events
func (fw *FileWatcher) handleCreate(path string, folder *storage.IndexedFolder) {
	info, err := os.Stat(path)
	if err != nil {
		fw.logger.Debug("Failed to stat created file",
			zap.String("path", path),
			zap.Error(err))
		return
	}

	// If it's a directory, add it to watcher
	if info.IsDir() {
		if err := fw.watcher.Add(path); err != nil {
			fw.logger.Error("Failed to watch new directory",
				zap.String("path", path),
				zap.Error(err))
		}
		return
	}

	// If it's a code file, index it
	if scanner.IsCodeFile(path) {
		fw.indexFile(path, folder)
	}
}

// handleUpdate handles file modification events
func (fw *FileWatcher) handleUpdate(path string, folder *storage.IndexedFolder) {
	// Only process code files
	if !scanner.IsCodeFile(path) {
		return
	}

	// Re-index the file
	fw.indexFile(path, folder)
}

// handleDelete handles file deletion events
func (fw *FileWatcher) handleDelete(path string, folder *storage.IndexedFolder) {
	// Remove from watcher if it was a directory
	fw.watcher.Remove(path)

	// Get file from MongoDB
	file, err := fw.mongoStorage.GetFileByPath(path)
	if err != nil {
		fw.logger.Debug("File not found in index",
			zap.String("path", path),
			zap.Error(err))
		return
	}

	if file == nil {
		return
	}

	// Get all chunks for this file
	chunks, err := fw.mongoStorage.ListChunks(file.ID)
	if err != nil {
		fw.logger.Error("Failed to list chunks for deleted file",
			zap.String("path", path),
			zap.Error(err))
		return
	}

	// Delete vectors from Qdrant
	for _, chunk := range chunks {
		if err := fw.qdrantClient.DeletePoint(chunk.VectorID); err != nil {
			fw.logger.Error("Failed to delete vector",
				zap.String("vectorId", chunk.VectorID),
				zap.Error(err))
		}
	}

	// Delete from MongoDB (chunks will cascade due to foreign key)
	ctx := context.Background()
	if err := fw.mongoStorage.DeleteFile(ctx, file.ID); err != nil {
		fw.logger.Error("Failed to delete file from MongoDB",
			zap.String("path", path),
			zap.Error(err))
		return
	}

	// Update folder file count
	files, _ := fw.mongoStorage.ListFiles(folder.ID)
	fw.mongoStorage.UpdateFolderScanTime(folder.ID, len(files))

	fw.logger.Info("Deleted file from index",
		zap.String("path", path),
		zap.String("fileId", file.ID))
}

// indexFile indexes or re-indexes a single file
func (fw *FileWatcher) indexFile(path string, folder *storage.IndexedFolder) {
	fw.logger.Info("Indexing file",
		zap.String("path", path),
		zap.String("folderId", folder.ID))

	// Scan the file
	fileInfo, err := scanner.ScanFile(path, folder.Path)
	if err != nil {
		fw.logger.Error("Failed to scan file",
			zap.String("path", path),
			zap.Error(err))
		return
	}

	// Check if file already exists
	existingFile, err := fw.mongoStorage.GetFileByPath(path)
	if err != nil {
		fw.logger.Error("Failed to check existing file",
			zap.String("path", path),
			zap.Error(err))
		return
	}

	// Skip if file hasn't changed
	if existingFile != nil && existingFile.SHA256 == fileInfo.SHA256 {
		fw.logger.Debug("File unchanged, skipping",
			zap.String("path", path))
		return
	}

	// Delete old chunks and vectors if file exists
	if existingFile != nil {
		chunks, _ := fw.mongoStorage.ListChunks(existingFile.ID)
		for _, chunk := range chunks {
			fw.qdrantClient.DeletePoint(chunk.VectorID)
		}
	}

	// Create or update file record
	fileID := ""
	if existingFile != nil {
		fileID = existingFile.ID
	}

	file := &storage.IndexedFile{
		ID:           fileID,
		FolderID:     folder.ID,
		Path:         fileInfo.Path,
		RelativePath: fileInfo.RelativePath,
		Language:     fileInfo.Language,
		SHA256:       fileInfo.SHA256,
		Size:         fileInfo.Size,
		LineCount:    fileInfo.LineCount,
		ChunkCount:   len(fileInfo.Chunks),
	}

	if err := fw.mongoStorage.UpsertFile(file); err != nil {
		fw.logger.Error("Failed to upsert file",
			zap.String("path", path),
			zap.Error(err))
		return
	}

	// Index chunks
	for i, chunkContent := range fileInfo.Chunks {
		// Generate embedding
		embedding, err := fw.embeddingClient.CreateEmbedding(chunkContent.Content)
		if err != nil {
			fw.logger.Error("Failed to create embedding",
				zap.String("path", path),
				zap.Int("chunk", i),
				zap.Error(err))
			continue
		}

		vectorID := fmt.Sprintf("%s_%d", file.ID, i)

		// Store in Qdrant
		payload := map[string]interface{}{
			"fileId":       file.ID,
			"folderId":     folder.ID,
			"folderPath":   folder.Path,
			"filePath":     file.Path,
			"relativePath": file.RelativePath,
			"language":     file.Language,
			"chunkNum":     i,
			"startLine":    chunkContent.StartLine,
			"endLine":      chunkContent.EndLine,
			"content":      chunkContent.Content,
		}

		if err := fw.qdrantClient.UpsertPoint(vectorID, embedding, payload); err != nil {
			fw.logger.Error("Failed to upsert vector",
				zap.String("vectorId", vectorID),
				zap.Error(err))
			continue
		}

		// Store chunk in MongoDB
		chunk := &storage.FileChunk{
			FileID:    file.ID,
			ChunkNum:  i,
			Content:   chunkContent.Content,
			StartLine: chunkContent.StartLine,
			EndLine:   chunkContent.EndLine,
			VectorID:  vectorID,
		}

		if err := fw.mongoStorage.UpsertChunk(chunk); err != nil {
			fw.logger.Error("Failed to upsert chunk",
				zap.String("path", path),
				zap.Int("chunk", i),
				zap.Error(err))
		}
	}

	// Update folder file count
	files, _ := fw.mongoStorage.ListFiles(folder.ID)
	fw.mongoStorage.UpdateFolderScanTime(folder.ID, len(files))

	fw.logger.Info("File indexed successfully",
		zap.String("path", path),
		zap.Int("chunks", len(fileInfo.Chunks)))
}

// findFolder finds which indexed folder a file belongs to
func (fw *FileWatcher) findFolder(filePath string) *storage.IndexedFolder {
	fw.foldersMutex.RLock()
	defer fw.foldersMutex.RUnlock()

	for folderPath, folder := range fw.watchedFolders {
		if strings.HasPrefix(filePath, folderPath) {
			return folder
		}
	}
	return nil
}

// shouldIgnore checks if a path should be ignored
func (fw *FileWatcher) shouldIgnore(path string) bool {
	ignoredDirs := []string{
		".git", "node_modules", "vendor", "dist", "build",
		".vscode", ".idea", "__pycache__", ".next", "out",
	}

	base := filepath.Base(path)
	for _, ignored := range ignoredDirs {
		if base == ignored {
			return true
		}
	}

	// Ignore hidden files (except .go, .c, etc.)
	if strings.HasPrefix(base, ".") && !scanner.IsCodeFile(path) {
		return true
	}

	return false
}
