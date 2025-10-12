package storage

import (
	"time"
)

// IndexedFolder represents a folder that is being tracked for code indexing
type IndexedFolder struct {
	ID          string    `bson:"_id,omitempty" json:"id"`
	Path        string    `bson:"path" json:"path"`                               // Absolute path to the folder
	Description string    `bson:"description,omitempty" json:"description"`       // Optional description
	AddedAt     time.Time `bson:"addedAt" json:"addedAt"`                         // When folder was added
	LastScanned time.Time `bson:"lastScanned,omitempty" json:"lastScanned"`       // Last scan timestamp
	FileCount   int       `bson:"fileCount" json:"fileCount"`                     // Number of indexed files
	Status      string    `bson:"status" json:"status"`                           // active, scanning, error
	Error       string    `bson:"error,omitempty" json:"error,omitempty"`         // Last error if any
}

// IndexedFile represents a single file in the code index
type IndexedFile struct {
	ID           string    `bson:"_id,omitempty" json:"id"`
	FolderID     string    `bson:"folderId" json:"folderId"`                     // Reference to IndexedFolder
	Path         string    `bson:"path" json:"path"`                             // Absolute path to file
	RelativePath string    `bson:"relativePath" json:"relativePath"`             // Path relative to folder
	Language     string    `bson:"language" json:"language"`                     // Programming language
	SHA256       string    `bson:"sha256" json:"sha256"`                         // SHA-256 hash of content
	Size         int64     `bson:"size" json:"size"`                             // File size in bytes
	LineCount    int       `bson:"lineCount" json:"lineCount"`                   // Number of lines
	IndexedAt    time.Time `bson:"indexedAt" json:"indexedAt"`                   // When file was indexed
	UpdatedAt    time.Time `bson:"updatedAt" json:"updatedAt"`                   // Last update time
	VectorID     string    `bson:"vectorId,omitempty" json:"vectorId,omitempty"` // Qdrant point ID
	ChunkCount   int       `bson:"chunkCount" json:"chunkCount"`                 // Number of chunks
}

// FileChunk represents a chunk of a file (for large files)
type FileChunk struct {
	ID        string    `bson:"_id,omitempty" json:"id"`
	FileID    string    `bson:"fileId" json:"fileId"`                 // Reference to IndexedFile
	ChunkNum  int       `bson:"chunkNum" json:"chunkNum"`             // Chunk number (0-based)
	Content   string    `bson:"content" json:"content"`               // Chunk content
	StartLine int       `bson:"startLine" json:"startLine"`           // Starting line number
	EndLine   int       `bson:"endLine" json:"endLine"`               // Ending line number
	VectorID  string    `bson:"vectorId,omitempty" json:"vectorId"`   // Qdrant point ID
	IndexedAt time.Time `bson:"indexedAt" json:"indexedAt"`           // When chunk was indexed
}

// SearchResult represents a search result from the code index
type SearchResult struct {
	FileID       string  `json:"fileId"`
	FilePath     string  `json:"filePath"`
	RelativePath string  `json:"relativePath"`
	Language     string  `json:"language"`
	ChunkNum     int     `json:"chunkNum,omitempty"`
	StartLine    int     `json:"startLine,omitempty"`
	EndLine      int     `json:"endLine,omitempty"`
	Content      string  `json:"content"`
	Score        float32 `json:"score"`        // Similarity score
	FolderID     string  `json:"folderId"`
	FolderPath   string  `json:"folderPath"`
}

// IndexStatus represents the current status of the code index
type IndexStatus struct {
	TotalFolders   int       `json:"totalFolders"`
	TotalFiles     int       `json:"totalFiles"`
	TotalChunks    int       `json:"totalChunks"`
	LastScanTime   time.Time `json:"lastScanTime,omitempty"`
	ActiveFolders  int       `json:"activeFolders"`
	ScanningFolders int       `json:"scanningFolders"`
	ErrorFolders   int       `json:"errorFolders"`
}
