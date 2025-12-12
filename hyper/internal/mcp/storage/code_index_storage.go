package storage

import (
	"context"
	"fmt"
	"strings"
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
	pathMappingsCol *mongo.Collection
}

// NewCodeIndexStorage creates a new MongoDB storage instance
func NewCodeIndexStorage(db *mongo.Database) (*CodeIndexStorage, error) {
	storage := &CodeIndexStorage{
		db:              db,
		foldersCol:      db.Collection("indexed_folders"),
		filesCol:        db.Collection("indexed_files"),
		chunksCol:       db.Collection("file_chunks"),
		pathMappingsCol: db.Collection("code_index_map"),
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

	// Path mappings indexes
	_, err = s.pathMappingsCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "path", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "qdrantCollection", Value: 1}}},
	})
	if err != nil {
		return fmt.Errorf("failed to create path mapping indexes: %w", err)
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
		{Keys: bson.D{{Key: "chunkType", Value: 1}}},
		{Keys: bson.D{{Key: "nodeType", Value: 1}}},
		{Keys: bson.D{{Key: "nodeName", Value: 1}}},
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

// AddFolderWithConfig adds a new folder with custom configuration
func (s *CodeIndexStorage) AddFolderWithConfig(path, description string, includePatterns, excludePatterns []string, chunkSize string) (*IndexedFolder, error) {
	folder := &IndexedFolder{
		Path:            path,
		Description:     description,
		AddedAt:         time.Now(),
		Status:          "active",
		FileCount:       0,
		IncludePatterns: includePatterns,
		ExcludePatterns: excludePatterns,
		ChunkSize:       chunkSize,
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

	// Delete the folder - convert string ID to ObjectID
	objectID, err := primitive.ObjectIDFromHex(folderID)
	if err != nil {
		return fmt.Errorf("invalid folder ID format: %s", folderID)
	}

	_, err = s.foldersCol.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return fmt.Errorf("failed to delete folder: %w", err)
	}

	return nil
}

// GetFolder retrieves a folder by ID
func (s *CodeIndexStorage) GetFolder(folderID string) (*IndexedFolder, error) {
	// Convert string ID to ObjectID
	objectID, err := primitive.ObjectIDFromHex(folderID)
	if err != nil {
		return nil, fmt.Errorf("invalid folder ID format: %s", folderID)
	}

	var folder IndexedFolder
	err = s.foldersCol.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&folder)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("folder not found: %s", folderID)
		}
		return nil, fmt.Errorf("failed to get folder: %w", err)
	}

	// Populate the ID field from MongoDB's _id
	folder.ID = objectID.Hex()
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
	// Convert string ID to ObjectID
	objectID, err := primitive.ObjectIDFromHex(folderID)
	if err != nil {
		return fmt.Errorf("invalid folder ID format: %s", folderID)
	}

	update := bson.M{
		"$set": bson.M{
			"status": status,
		},
	}

	if errorMsg != "" {
		update["$set"].(bson.M)["error"] = errorMsg
	}

	_, err = s.foldersCol.UpdateOne(
		context.Background(),
		bson.M{"_id": objectID},
		update,
	)
	if err != nil {
		return fmt.Errorf("failed to update folder status: %w", err)
	}

	return nil
}

// UpdateFolderScanTime updates the last scanned time for a folder
func (s *CodeIndexStorage) UpdateFolderScanTime(folderID string, fileCount int) error {
	// Convert string ID to ObjectID
	objectID, err := primitive.ObjectIDFromHex(folderID)
	if err != nil {
		return fmt.Errorf("invalid folder ID format: %s", folderID)
	}

	_, err = s.foldersCol.UpdateOne(
		context.Background(),
		bson.M{"_id": objectID},
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

// GetChunksByFileIDAndLineRange retrieves chunks that overlap with a specified line range
// Returns chunks where startLine <= targetEnd AND endLine >= targetStart
func (s *CodeIndexStorage) GetChunksByFileIDAndLineRange(fileID string, targetStart, targetEnd int) ([]*FileChunk, error) {
	filter := bson.M{
		"fileId": fileID,
		"$and": []bson.M{
			{"startLine": bson.M{"$lte": targetEnd}},
			{"endLine": bson.M{"$gte": targetStart}},
		},
	}

	cursor, err := s.chunksCol.Find(
		context.Background(),
		filter,
		options.Find().SetSort(bson.D{{Key: "chunkNum", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query chunks by line range: %w", err)
	}
	defer cursor.Close(context.Background())

	var chunks []*FileChunk
	if err := cursor.All(context.Background(), &chunks); err != nil {
		return nil, fmt.Errorf("failed to decode chunks: %w", err)
	}

	return chunks, nil
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

// CodeIndexMapping represents a path-to-Qdrant-collection mapping
type CodeIndexMapping struct {
	Path             string    `bson:"path" json:"path"`
	QdrantCollection string    `bson:"qdrantCollection" json:"qdrantCollection"`
	CreatedAt        time.Time `bson:"createdAt" json:"createdAt"`
	LastIndexed      time.Time `bson:"lastIndexed" json:"lastIndexed"`
}

// AddPathMapping adds or updates a path-to-collection mapping
func (s *CodeIndexStorage) AddPathMapping(path, qdrantCollection string) error {
	mapping := &CodeIndexMapping{
		Path:             path,
		QdrantCollection: qdrantCollection,
		CreatedAt:        time.Now(),
		LastIndexed:      time.Now(),
	}

	opts := options.Update().SetUpsert(true)
	filter := bson.M{"path": path}
	update := bson.M{
		"$set": bson.M{
			"qdrantCollection": qdrantCollection,
			"lastIndexed":      mapping.LastIndexed,
		},
		"$setOnInsert": bson.M{
			"createdAt": mapping.CreatedAt,
		},
	}

	_, err := s.pathMappingsCol.UpdateOne(context.Background(), filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to add path mapping: %w", err)
	}

	return nil
}

// GetPathMapping retrieves a mapping by path
func (s *CodeIndexStorage) GetPathMapping(path string) (*CodeIndexMapping, error) {
	var mapping CodeIndexMapping
	err := s.pathMappingsCol.FindOne(context.Background(), bson.M{"path": path}).Decode(&mapping)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get path mapping: %w", err)
	}
	return &mapping, nil
}

// GetPathMappingByPath is an alias for GetPathMapping for clarity
func (s *CodeIndexStorage) GetPathMappingByPath(path string) (*CodeIndexMapping, error) {
	return s.GetPathMapping(path)
}

// ListPathMappings returns all path mappings
func (s *CodeIndexStorage) ListPathMappings() ([]*CodeIndexMapping, error) {
	cursor, err := s.pathMappingsCol.Find(context.Background(), bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to list path mappings: %w", err)
	}
	defer cursor.Close(context.Background())

	var mappings []*CodeIndexMapping
	if err := cursor.All(context.Background(), &mappings); err != nil {
		return nil, fmt.Errorf("failed to decode path mappings: %w", err)
	}

	return mappings, nil
}

// RemovePathMapping deletes a path-to-collection mapping
func (s *CodeIndexStorage) RemovePathMapping(path string) error {
	_, err := s.pathMappingsCol.DeleteOne(context.Background(), bson.M{"path": path})
	if err != nil {
		return fmt.Errorf("failed to remove path mapping: %w", err)
	}
	return nil
}

// StructuralFilter represents structural search criteria for code chunks
type StructuralFilter struct {
	FunctionName string   // Function name pattern (supports regex)
	ClassName    string   // Exact class name match
	NodeType     string   // Node type: function, class, method, interface, import
	Symbols      []string // Symbols that must be present
	Imports      []string // Imports that must be present
	HasDocstring *bool    // Filter by documentation presence (nil = no filter)
}

// FindChunksWithFilters performs MongoDB filtering to find chunks matching structural criteria
// Returns chunk IDs (vectorId) that match the filter for use in Qdrant search
func (s *CodeIndexStorage) FindChunksWithFilters(filter StructuralFilter) ([]string, error) {
	ctx := context.Background()

	// Build MongoDB filter
	mongoFilter := bson.M{}

	// NodeType filter (exact match)
	if filter.NodeType != "" {
		mongoFilter["nodeType"] = filter.NodeType
	}

	// ClassName filter (exact match)
	if filter.ClassName != "" {
		mongoFilter["nodeName"] = filter.ClassName
		// Also ensure it's a class type
		mongoFilter["nodeType"] = "class"
	}

	// FunctionName filter (regex pattern match on nodeName)
	if filter.FunctionName != "" {
		// Support both exact match and regex patterns
		mongoFilter["nodeName"] = bson.M{"$regex": filter.FunctionName, "$options": "i"}
		// Filter to function/method types
		if filter.NodeType == "" {
			mongoFilter["nodeType"] = bson.M{"$in": []string{"function", "method"}}
		}
	}

	// Symbols filter (must contain all specified symbols)
	if len(filter.Symbols) > 0 {
		mongoFilter["symbols"] = bson.M{"$all": filter.Symbols}
	}

	// Imports filter (must contain all specified imports)
	if len(filter.Imports) > 0 {
		mongoFilter["imports"] = bson.M{"$all": filter.Imports}
	}

	// HasDocstring filter
	if filter.HasDocstring != nil {
		mongoFilter["hasDocstring"] = *filter.HasDocstring
	}

	// Query chunks collection
	cursor, err := s.chunksCol.Find(ctx, mongoFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to find chunks with filters: %w", err)
	}
	defer cursor.Close(ctx)

	// Extract vectorIds from matching chunks
	var chunkIDs []string
	for cursor.Next(ctx) {
		var chunk FileChunk
		if err := cursor.Decode(&chunk); err != nil {
			continue
		}
		// Only include chunks that have been indexed in Qdrant (have vectorId)
		if chunk.VectorID != "" {
			chunkIDs = append(chunkIDs, chunk.VectorID)
		}
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return chunkIDs, nil
}

// ListAllPathMappings returns all path->collection mappings
func (s *CodeIndexStorage) ListAllPathMappings() ([]*CodeIndexMapping, error) {
	ctx := context.Background()
	cursor, err := s.pathMappingsCol.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var mappings []*CodeIndexMapping
	if err := cursor.All(ctx, &mappings); err != nil {
		return nil, err
	}
	return mappings, nil
}

// CountAllFiles returns total file count
func (s *CodeIndexStorage) CountAllFiles() (int, error) {
	count, err := s.filesCol.CountDocuments(context.Background(), bson.M{})
	return int(count), err
}

// CountAllChunks returns total chunk count
func (s *CodeIndexStorage) CountAllChunks() (int, error) {
	count, err := s.chunksCol.CountDocuments(context.Background(), bson.M{})
	return int(count), err
}

// ClearAllIndexData removes all indexed data but preserves folder configurations
// This allows users to reindex after clearing without needing to re-add folders

// GetContentByFilePathAndLineRange retrieves code content for a file by path and optional line range
// If startLine and endLine are 0, returns the entire file content
// Returns file metadata, content, and actual line range retrieved
func (s *CodeIndexStorage) GetContentByFilePathAndLineRange(filePath string, startLine, endLine int) (*ContentResult, error) {
	fmt.Printf("🗄️ PHASE2-STORAGE: GetContentByFilePathAndLineRange called - filePath=%s, startLine=%d, endLine=%d\n", filePath, startLine, endLine)

	// Validate file path
	if filePath == "" {
		fmt.Printf("🗄️ PHASE2-STORAGE: ERROR - filePath is empty\n")
		return nil, fmt.Errorf("filePath is required")
	}

	// Reject archived files
	if strings.Contains(filePath, "/.archived/") || strings.Contains(filePath, "/.archive/") {
		fmt.Printf("🗄️ PHASE2-STORAGE: ERROR - archived file rejected: %s\n", filePath)
		return nil, fmt.Errorf("cannot fetch content from archived file: %s", filePath)
	}

	// Get file metadata by path
	fmt.Printf("🗄️ PHASE2-STORAGE: Looking up file in MongoDB by path: %s\n", filePath)
	file, err := s.GetFileByPath(filePath)
	if err != nil {
		fmt.Printf("🗄️ PHASE2-STORAGE: ERROR - failed to get file by path: %v\n", err)
		return nil, fmt.Errorf("failed to get file by path: %w", err)
	}
	if file == nil {
		fmt.Printf("🗄️ PHASE2-STORAGE: ERROR - file not found in index: %s\n", filePath)
		return nil, fmt.Errorf("file not found in index: %s", filePath)
	}
	fmt.Printf("🗄️ PHASE2-STORAGE: File found - ID=%s, Language=%s, LineCount=%d, ChunkCount=%d\n", file.ID, file.Language, file.LineCount, file.ChunkCount)

	// Fetch all chunks for this file
	fmt.Printf("🗄️ PHASE2-STORAGE: Fetching chunks for fileID: %s\n", file.ID)
	chunks, err := s.GetChunksByFileID(file.ID)
	if err != nil {
		fmt.Printf("🗄️ PHASE2-STORAGE: ERROR - failed to fetch chunks: %v\n", err)
		return nil, fmt.Errorf("failed to fetch chunks: %w", err)
	}
	if len(chunks) == 0 {
		fmt.Printf("🗄️ PHASE2-STORAGE: ERROR - no chunks found for file: %s\n", filePath)
		return nil, fmt.Errorf("no chunks found for file: %s", filePath)
	}
	fmt.Printf("🗄️ PHASE2-STORAGE: Found %d chunks for file\n", len(chunks))

	// Build full file content from chunks
	var fullContent strings.Builder
	for i, chunk := range chunks {
		fullContent.WriteString(chunk.Content)
		if i == 0 {
			fmt.Printf("🗄️ PHASE2-STORAGE: Chunk 0: startLine=%d, endLine=%d, contentLen=%d\n", chunk.StartLine, chunk.EndLine, len(chunk.Content))
		}
	}
	totalContentLen := fullContent.Len()
	fmt.Printf("🗄️ PHASE2-STORAGE: Built full content from chunks - totalBytes=%d\n", totalContentLen)

	// Determine actual line range to return
	actualStartLine := startLine
	actualEndLine := endLine

	// If no line range specified, return entire file
	if startLine == 0 && endLine == 0 {
		actualStartLine = 1
		actualEndLine = file.LineCount
		fmt.Printf("🗄️ PHASE2-STORAGE: No line range specified - returning entire file (lines 1-%d)\n", file.LineCount)
	} else {
		// Validate line range
		if startLine < 1 {
			actualStartLine = 1
		}
		if endLine > file.LineCount {
			actualEndLine = file.LineCount
		}
		if actualStartLine > actualEndLine {
			fmt.Printf("🗄️ PHASE2-STORAGE: ERROR - invalid line range: startLine (%d) > endLine (%d)\n", actualStartLine, actualEndLine)
			return nil, fmt.Errorf("invalid line range: startLine (%d) > endLine (%d)", actualStartLine, actualEndLine)
		}
		fmt.Printf("🗄️ PHASE2-STORAGE: Line range validated - requested=%d-%d, actual=%d-%d\n", startLine, endLine, actualStartLine, actualEndLine)
	}

	// Extract requested line range from full content
	allLines := strings.Split(fullContent.String(), "\n")
	fmt.Printf("🗄️ PHASE2-STORAGE: Split content into %d lines\n", len(allLines))

	// Convert to 0-based indexing for extraction
	startIdx := actualStartLine - 1
	endIdx := actualEndLine

	// Validate indices
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx > len(allLines) {
		endIdx = len(allLines)
	}

	// Extract lines
	var extractedLines []string
	if startIdx < len(allLines) {
		extractedLines = allLines[startIdx:endIdx]
	}
	fmt.Printf("🗄️ PHASE2-STORAGE: Extracted %d lines (indices %d-%d)\n", len(extractedLines), startIdx, endIdx)

	// Build result
	result := &ContentResult{
		FilePath:       file.Path,
		RelativePath:   file.RelativePath,
		Language:       file.Language,
		LineCount:      file.LineCount,
		Content:        strings.Join(extractedLines, "\n"),
		StartLine:      actualStartLine,
		EndLine:        actualEndLine,
		RequestedStart: startLine,
		RequestedEnd:   endLine,
		Size:           file.Size,
		IndexedAt:      file.IndexedAt,
	}

	fmt.Printf("🗄️ PHASE2-STORAGE: SUCCESS - returning result: filePath=%s, lines=%d-%d, contentLen=%d\n", result.FilePath, result.StartLine, result.EndLine, len(result.Content))
	return result, nil
}

// ContentResult represents the result of fetching code content
type ContentResult struct {
	FilePath       string    `json:"filePath"`
	RelativePath   string    `json:"relativePath"`
	Language       string    `json:"language"`
	LineCount      int       `json:"lineCount"`
	Content        string    `json:"content"`
	StartLine      int       `json:"startLine"`      // Actual start line returned
	EndLine        int       `json:"endLine"`        // Actual end line returned
	RequestedStart int       `json:"requestedStart"` // Requested start line (may differ if out of range)
	RequestedEnd   int       `json:"requestedEnd"`   // Requested end line (may differ if out of range)
	Size           int64     `json:"size"`
	IndexedAt      time.Time `json:"indexedAt"`
}

func (s *CodeIndexStorage) ClearAllIndexData() error {
	ctx := context.Background()

	// Delete in order: chunks -> files -> mappings
	// NOTE: We preserve folder configurations so users can reindex
	if _, err := s.chunksCol.DeleteMany(ctx, bson.M{}); err != nil {
		return fmt.Errorf("failed to clear chunks: %w", err)
	}

	if _, err := s.filesCol.DeleteMany(ctx, bson.M{}); err != nil {
		return fmt.Errorf("failed to clear files: %w", err)
	}

	if _, err := s.pathMappingsCol.DeleteMany(ctx, bson.M{}); err != nil {
		return fmt.Errorf("failed to clear path mappings: %w", err)
	}

	return nil
}
