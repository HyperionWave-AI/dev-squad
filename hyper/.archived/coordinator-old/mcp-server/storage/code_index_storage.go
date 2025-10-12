package storage

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CodeIndexStorage handles MongoDB operations for code indexing
type CodeIndexStorage struct {
	db              *mongo.Database
	foldersCol      *mongo.Collection
	filesCol        *mongo.Collection
	chunksCol       *mongo.Collection
}

// NewCodeIndexStorage creates a new MongoDB storage instance
func NewCodeIndexStorage(db *mongo.Database) (*CodeIndexStorage, error) {
	storage := &CodeIndexStorage{
		db:         db,
		foldersCol: db.Collection("indexed_folders"),
		filesCol:   db.Collection("indexed_files"),
		chunksCol:  db.Collection("file_chunks"),
	}

	// Create indexes
	if err := storage.createIndexes(); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return storage, nil
}

// createIndexes creates necessary indexes for efficient querying
func (s *CodeIndexStorage) createIndexes() error {
	ctx := context.Background()

	// Folders indexes
	_, err := s.foldersCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "path", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "status", Value: 1}}},
	})
	if err != nil {
		return fmt.Errorf("failed to create folder indexes: %w", err)
	}

	// Files indexes
	_, err = s.filesCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "folderId", Value: 1}}},
		{Keys: bson.D{{Key: "path", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "sha256", Value: 1}}},
		{Keys: bson.D{{Key: "language", Value: 1}}},
	})
	if err != nil {
		return fmt.Errorf("failed to create file indexes: %w", err)
	}

	// Chunks indexes
	_, err = s.chunksCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "fileId", Value: 1}, {Key: "chunkNum", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "vectorId", Value: 1}}},
	})
	if err != nil {
		return fmt.Errorf("failed to create chunk indexes: %w", err)
	}

	return nil
}

// AddFolder adds a new folder to the index
func (s *CodeIndexStorage) AddFolder(path, description string) (*IndexedFolder, error) {
	folder := &IndexedFolder{
		Path:        path,
		Description: description,
		AddedAt:     time.Now(),
		Status:      "active",
		FileCount:   0,
	}

	result, err := s.foldersCol.InsertOne(context.Background(), folder)
	if err != nil {
		return nil, fmt.Errorf("failed to insert folder: %w", err)
	}

	// Convert ObjectID to string
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		folder.ID = oid.Hex()
	} else {
		folder.ID = fmt.Sprintf("%v", result.InsertedID)
	}
	return folder, nil
}

// RemoveFolder removes a folder and all its associated files and chunks
func (s *CodeIndexStorage) RemoveFolder(folderID string) error {
	ctx := context.Background()

	// Get all file IDs for this folder
	cursor, err := s.filesCol.Find(ctx, bson.M{"folderId": folderID})
	if err != nil {
		return fmt.Errorf("failed to find files: %w", err)
	}
	defer cursor.Close(ctx)

	var fileIDs []string
	for cursor.Next(ctx) {
		var file IndexedFile
		if err := cursor.Decode(&file); err != nil {
			continue
		}
		fileIDs = append(fileIDs, file.ID)
	}

	// Delete all chunks for these files
	if len(fileIDs) > 0 {
		_, err = s.chunksCol.DeleteMany(ctx, bson.M{"fileId": bson.M{"$in": fileIDs}})
		if err != nil {
			return fmt.Errorf("failed to delete chunks: %w", err)
		}
	}

	// Delete all files for this folder
	_, err = s.filesCol.DeleteMany(ctx, bson.M{"folderId": folderID})
	if err != nil {
		return fmt.Errorf("failed to delete files: %w", err)
	}

	// Delete the folder
	_, err = s.foldersCol.DeleteOne(ctx, bson.M{"_id": folderID})
	if err != nil {
		return fmt.Errorf("failed to delete folder: %w", err)
	}

	return nil
}

// GetFolder retrieves a folder by ID
func (s *CodeIndexStorage) GetFolder(folderID string) (*IndexedFolder, error) {
	var folder IndexedFolder
	err := s.foldersCol.FindOne(context.Background(), bson.M{"_id": folderID}).Decode(&folder)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("folder not found: %s", folderID)
		}
		return nil, fmt.Errorf("failed to get folder: %w", err)
	}
	return &folder, nil
}

// GetFolderByPath retrieves a folder by path
func (s *CodeIndexStorage) GetFolderByPath(path string) (*IndexedFolder, error) {
	var folder IndexedFolder
	err := s.foldersCol.FindOne(context.Background(), bson.M{"path": path}).Decode(&folder)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get folder: %w", err)
	}
	return &folder, nil
}

// ListFolders returns all indexed folders
func (s *CodeIndexStorage) ListFolders() ([]*IndexedFolder, error) {
	cursor, err := s.foldersCol.Find(context.Background(), bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}
	defer cursor.Close(context.Background())

	var folders []*IndexedFolder
	if err := cursor.All(context.Background(), &folders); err != nil {
		return nil, fmt.Errorf("failed to decode folders: %w", err)
	}

	return folders, nil
}

// UpdateFolderStatus updates the status of a folder
func (s *CodeIndexStorage) UpdateFolderStatus(folderID, status, errorMsg string) error {
	update := bson.M{
		"$set": bson.M{
			"status": status,
		},
	}

	if errorMsg != "" {
		update["$set"].(bson.M)["error"] = errorMsg
	}

	_, err := s.foldersCol.UpdateOne(
		context.Background(),
		bson.M{"_id": folderID},
		update,
	)
	if err != nil {
		return fmt.Errorf("failed to update folder status: %w", err)
	}

	return nil
}

// UpdateFolderScanTime updates the last scanned time for a folder
func (s *CodeIndexStorage) UpdateFolderScanTime(folderID string, fileCount int) error {
	_, err := s.foldersCol.UpdateOne(
		context.Background(),
		bson.M{"_id": folderID},
		bson.M{
			"$set": bson.M{
				"lastScanned": time.Now(),
				"fileCount":   fileCount,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to update folder scan time: %w", err)
	}

	return nil
}

// UpsertFile inserts or updates a file in the index
func (s *CodeIndexStorage) UpsertFile(file *IndexedFile) error {
	file.UpdatedAt = time.Now()
	if file.IndexedAt.IsZero() {
		file.IndexedAt = time.Now()
	}

	opts := options.Update().SetUpsert(true)
	filter := bson.M{"path": file.Path}

	// Build update document without _id field to avoid immutable field error
	update := bson.M{
		"$set": bson.M{
			"folderId":     file.FolderID,
			"path":         file.Path,
			"relativePath": file.RelativePath,
			"language":     file.Language,
			"sha256":       file.SHA256,
			"size":         file.Size,
			"lineCount":    file.LineCount,
			"indexedAt":    file.IndexedAt,
			"updatedAt":    file.UpdatedAt,
			"chunkCount":   file.ChunkCount,
		},
	}

	// Only set vectorId if it's not empty
	if file.VectorID != "" {
		update["$set"].(bson.M)["vectorId"] = file.VectorID
	}

	// If this is an insert (upsert creating new doc), set the ID only on insert
	if file.ID != "" {
		update["$setOnInsert"] = bson.M{"_id": file.ID}
	}

	_, err := s.filesCol.UpdateOne(context.Background(), filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert file: %w", err)
	}

	return nil
}

// GetFile retrieves a file by ID
func (s *CodeIndexStorage) GetFile(fileID string) (*IndexedFile, error) {
	var file IndexedFile
	err := s.filesCol.FindOne(context.Background(), bson.M{"_id": fileID}).Decode(&file)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("file not found: %s", fileID)
		}
		return nil, fmt.Errorf("failed to get file: %w", err)
	}
	return &file, nil
}

// GetFileByPath retrieves a file by path
func (s *CodeIndexStorage) GetFileByPath(path string) (*IndexedFile, error) {
	var file IndexedFile
	err := s.filesCol.FindOne(context.Background(), bson.M{"path": path}).Decode(&file)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get file: %w", err)
	}
	return &file, nil
}

// ListFiles returns all indexed files for a folder
func (s *CodeIndexStorage) ListFiles(folderID string) ([]*IndexedFile, error) {
	cursor, err := s.filesCol.Find(context.Background(), bson.M{"folderId": folderID})
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	defer cursor.Close(context.Background())

	var files []*IndexedFile
	if err := cursor.All(context.Background(), &files); err != nil {
		return nil, fmt.Errorf("failed to decode files: %w", err)
	}

	return files, nil
}

// UpsertChunk inserts or updates a file chunk
func (s *CodeIndexStorage) UpsertChunk(chunk *FileChunk) error {
	chunk.IndexedAt = time.Now()

	opts := options.Update().SetUpsert(true)
	filter := bson.M{"fileId": chunk.FileID, "chunkNum": chunk.ChunkNum}
	update := bson.M{"$set": chunk}

	_, err := s.chunksCol.UpdateOne(context.Background(), filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert chunk: %w", err)
	}

	return nil
}

// GetChunk retrieves a chunk by file ID and chunk number
func (s *CodeIndexStorage) GetChunk(fileID string, chunkNum int) (*FileChunk, error) {
	var chunk FileChunk
	err := s.chunksCol.FindOne(context.Background(), bson.M{
		"fileId":   fileID,
		"chunkNum": chunkNum,
	}).Decode(&chunk)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get chunk: %w", err)
	}
	return &chunk, nil
}

// ListChunks returns all chunks for a file
func (s *CodeIndexStorage) ListChunks(fileID string) ([]*FileChunk, error) {
	cursor, err := s.chunksCol.Find(
		context.Background(),
		bson.M{"fileId": fileID},
		options.Find().SetSort(bson.D{{Key: "chunkNum", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list chunks: %w", err)
	}
	defer cursor.Close(context.Background())

	var chunks []*FileChunk
	if err := cursor.All(context.Background(), &chunks); err != nil {
		return nil, fmt.Errorf("failed to decode chunks: %w", err)
	}

	return chunks, nil
}

// DeleteFile deletes a file and all its associated chunks
func (s *CodeIndexStorage) DeleteFile(ctx context.Context, fileID string) error {
	// Delete all chunks for this file
	_, err := s.chunksCol.DeleteMany(ctx, bson.M{"fileId": fileID})
	if err != nil {
		return fmt.Errorf("failed to delete chunks: %w", err)
	}

	// Delete the file
	_, err = s.filesCol.DeleteOne(ctx, bson.M{"_id": fileID})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// GetChunksByFileID retrieves all chunks for a file (alias for ListChunks for clarity)
func (s *CodeIndexStorage) GetChunksByFileID(fileID string) ([]*FileChunk, error) {
	return s.ListChunks(fileID)
}

// GetIndexStatus returns the current status of the code index
func (s *CodeIndexStorage) GetIndexStatus() (*IndexStatus, error) {
	ctx := context.Background()

	// Count folders by status
	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$status"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}

	cursor, err := s.foldersCol.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate folders: %w", err)
	}
	defer cursor.Close(ctx)

	status := &IndexStatus{}
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}

		status.TotalFolders += result.Count
		switch result.ID {
		case "active":
			status.ActiveFolders = result.Count
		case "scanning":
			status.ScanningFolders = result.Count
		case "error":
			status.ErrorFolders = result.Count
		}
	}

	// Count total files
	totalFiles, err := s.filesCol.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to count files: %w", err)
	}
	status.TotalFiles = int(totalFiles)

	// Count total chunks
	totalChunks, err := s.chunksCol.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to count chunks: %w", err)
	}
	status.TotalChunks = int(totalChunks)

	// Get last scan time
	var lastFolder IndexedFolder
	opts := options.FindOne().SetSort(bson.D{{Key: "lastScanned", Value: -1}})
	err = s.foldersCol.FindOne(ctx, bson.M{}, opts).Decode(&lastFolder)
	if err == nil {
		status.LastScanTime = lastFolder.LastScanned
	}

	return status, nil
}
