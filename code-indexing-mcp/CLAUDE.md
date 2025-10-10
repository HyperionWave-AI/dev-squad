# Code Indexing MCP Server

## Overview

The Code Indexing MCP Server provides semantic code search capabilities using OpenAI embeddings and Qdrant vector database. It allows indexing of code repositories and performing natural language searches to find relevant code snippets.

## Architecture

### Components

1. **OpenAI Embedding Client** (`embeddings/openai_client.go`)
   - Uses OpenAI's `text-embedding-3-small` model (1536 dimensions)
   - Generates vector embeddings for code chunks
   - Batch processing support for multiple texts

2. **MongoDB Storage** (`storage/mongo_storage.go`)
   - Stores folder metadata (path, description, status, file count)
   - Stores file metadata (path, language, SHA-256 hash, size, line count)
   - Stores file chunks (content, line ranges, vector IDs)
   - Provides efficient querying with indexed fields

3. **Qdrant Vector Database** (`storage/qdrant_client.go`)
   - Stores vector embeddings for code chunks
   - Performs similarity search
   - Collection: `code_index` with cosine distance

4. **File Scanner** (`scanner/file_scanner.go`)
   - Scans directories for supported code files
   - Calculates SHA-256 hashes for change detection
   - Splits large files into 200-line chunks
   - Supports 30+ programming languages

5. **File Watcher** (`watcher/file_watcher.go`) **NEW**
   - Real-time monitoring of indexed folders using fsnotify
   - Automatic re-indexing on file changes (create, update, delete)
   - 500ms debouncing to prevent redundant indexing
   - Handles directory operations and removals
   - Integrates seamlessly with existing indexing pipeline

6. **MCP Tools** (`handlers/tools.go`)
   - 5 MCP tools for code indexing operations
   - JSON-based request/response format
   - Error handling with descriptive messages

## Database Schema

### MongoDB Collections

#### `indexed_folders`
```json
{
  "_id": "uuid",
  "path": "/absolute/path/to/folder",
  "description": "Optional description",
  "addedAt": "2025-01-10T...",
  "lastScanned": "2025-01-10T...",
  "fileCount": 150,
  "status": "active|scanning|error",
  "error": "Error message if status is error"
}
```

#### `indexed_files`
```json
{
  "_id": "uuid",
  "folderId": "folder-uuid",
  "path": "/absolute/path/to/file.go",
  "relativePath": "src/file.go",
  "language": "go",
  "sha256": "hash-value",
  "size": 5432,
  "lineCount": 234,
  "indexedAt": "2025-01-10T...",
  "updatedAt": "2025-01-10T...",
  "chunkCount": 2
}
```

#### `file_chunks`
```json
{
  "_id": "uuid",
  "fileId": "file-uuid",
  "chunkNum": 0,
  "content": "chunk content...",
  "startLine": 1,
  "endLine": 200,
  "vectorId": "file-uuid_0",
  "indexedAt": "2025-01-10T..."
}
```

### Qdrant Collection

**Collection Name:** `code_index`
**Vector Size:** 1536 (OpenAI text-embedding-3-small)
**Distance:** Cosine

**Point Payload:**
```json
{
  "fileId": "file-uuid",
  "folderId": "folder-uuid",
  "folderPath": "/path/to/folder",
  "filePath": "/path/to/file",
  "relativePath": "relative/path",
  "language": "go",
  "chunkNum": 0,
  "startLine": 1,
  "endLine": 200,
  "content": "code chunk..."
}
```

## MCP Tools

### 1. code_index_add_folder

Add a folder to the code index.

**Parameters:**
- `folderPath` (string, required): Absolute path to the folder
- `description` (string, optional): Description of the folder/project

**Response:**
```json
{
  "success": true,
  "message": "Folder added successfully. Use code_index_scan to index the files.",
  "folder": {
    "id": "uuid",
    "path": "/path/to/folder",
    "description": "My project",
    "addedAt": "2025-01-10T...",
    "status": "active",
    "fileCount": 0
  }
}
```

### 2. code_index_remove_folder

Remove a folder from the code index.

**Parameters:**
- `folderPath` (string, required): Absolute path to the folder (must match original path)

**Response:**
```json
{
  "success": true,
  "message": "Folder removed successfully",
  "filesRemoved": 150
}
```

### 3. code_index_scan

Scan or rescan a folder to update the code index.

**Parameters:**
- `folderPath` (string, required): Absolute path to the folder

**Response:**
```json
{
  "success": true,
  "filesIndexed": 100,
  "filesUpdated": 30,
  "filesSkipped": 20,
  "totalFiles": 150
}
```

**Process:**
1. Scans directory for supported code files
2. Calculates SHA-256 hash for each file
3. Compares with existing files (skips unchanged)
4. Splits files into 200-line chunks
5. Generates embeddings for each chunk
6. Stores vectors in Qdrant
7. Updates MongoDB metadata

### 4. code_index_search

Search for code using natural language queries.

**Parameters:**
- `query` (string, required): Natural language search query
- `limit` (number, optional): Maximum results (default: 10, max: 50)
- `folderPath` (string, optional): Filter to specific folder

**Response:**
```json
{
  "success": true,
  "query": "authentication logic",
  "results": [
    {
      "fileId": "uuid",
      "filePath": "/path/to/auth.go",
      "relativePath": "src/auth.go",
      "language": "go",
      "chunkNum": 0,
      "startLine": 1,
      "endLine": 200,
      "content": "func Authenticate()...",
      "score": 0.89,
      "folderId": "folder-uuid",
      "folderPath": "/path/to/folder"
    }
  ],
  "count": 1
}
```

### 5. code_index_status

Get the current status of the code index.

**Parameters:** None

**Response:**
```json
{
  "success": true,
  "status": {
    "totalFolders": 5,
    "totalFiles": 543,
    "totalChunks": 1234,
    "lastScanTime": "2025-01-10T...",
    "activeFolders": 4,
    "scanningFolders": 1,
    "errorFolders": 0
  },
  "folders": [
    {
      "id": "uuid",
      "path": "/path/to/folder",
      "description": "My project",
      "addedAt": "2025-01-10T...",
      "lastScanned": "2025-01-10T...",
      "fileCount": 150,
      "status": "active"
    }
  ]
}
```

## Supported Languages

The scanner supports 30+ programming languages:
- **Systems:** Go, C, C++, Rust, Swift
- **Web:** JavaScript, TypeScript, HTML, CSS, SCSS
- **Scripting:** Python, Ruby, PHP, Shell, Bash
- **JVM:** Java, Kotlin, Scala
- **Data:** SQL, R, JSON, YAML, XML
- **Mobile:** Objective-C, Swift, Kotlin
- **Documentation:** Markdown

## Configuration

### Environment Variables

```bash
# MongoDB Configuration
MONGODB_URI=mongodb+srv://user:pass@host/
MONGODB_DATABASE=code_index_db

# Qdrant Configuration
QDRANT_URL=https://your-qdrant-instance:6333
QDRANT_API_KEY=your-api-key

# OpenAI Configuration
OPENAI_API_KEY=sk-...

# Logging
LOG_LEVEL=info
```

### File Limits

- **Max File Size:** 10 MB
- **Chunk Size:** 200 lines per chunk
- **Ignored Directories:** `.git`, `node_modules`, `vendor`, `dist`, `build`, `.vscode`, `.idea`, `__pycache__`

## Usage Example

### 1. Add a Folder

```json
{
  "tool": "code_index_add_folder",
  "arguments": {
    "folderPath": "/Users/dev/my-project",
    "description": "Main project repository"
  }
}
```

### 2. Scan the Folder

```json
{
  "tool": "code_index_scan",
  "arguments": {
    "folderPath": "/Users/dev/my-project"
  }
}
```

### 3. Search for Code

```json
{
  "tool": "code_index_search",
  "arguments": {
    "query": "JWT token validation middleware",
    "limit": 5
  }
}
```

### 4. Check Status

```json
{
  "tool": "code_index_status",
  "arguments": {}
}
```

## Building and Running

### Build

```bash
cd code-indexing-mcp
go build -o code-indexing-mcp main.go
```

### Run

```bash
# Set environment variables
export MONGODB_URI="mongodb+srv://..."
export MONGODB_DATABASE="code_index_db"
export QDRANT_URL="https://..."
export QDRANT_API_KEY="..."
export OPENAI_API_KEY="sk-..."

# Run the server (stdio transport for MCP)
./code-indexing-mcp
```

## Performance Characteristics

### Indexing Performance

- **Small files (<100 lines):** ~100ms per file
- **Medium files (100-500 lines):** ~200ms per file
- **Large files (>500 lines):** ~50ms per chunk

### Search Performance

- **Vector search:** <100ms for top 10 results
- **Result processing:** <10ms

### Storage

- **MongoDB:** ~1KB per file metadata, ~500 bytes per chunk
- **Qdrant:** ~6KB per vector (1536 dimensions)

## Error Handling

All tools return structured errors:

```json
{
  "error": "Error message",
  "isError": true
}
```

Common errors:
- Invalid folder path
- Folder not found (must call add_folder first)
- Failed to connect to MongoDB/Qdrant
- Failed to generate embeddings (check OpenAI API key)
- Failed to scan directory (permission issues)

## Testing

Run tests:

```bash
go test ./... -v -cover
```

Expected coverage: 90%+

## File Watcher Features

### Real-Time Monitoring

The file watcher automatically monitors all indexed folders for changes and keeps the index up-to-date:

**Supported Operations:**
- **File Create:** Automatically indexes new code files
- **File Update:** Re-indexes modified files (detects changes via SHA-256 hash)
- **File Delete:** Removes file from index and cleans up vectors
- **Directory Create:** Adds new directories to watch list
- **Directory Delete:** Removes directory and all contained files from index

**Debouncing:**
- 500ms debounce period prevents redundant indexing
- Batches multiple rapid changes to the same file
- Efficient handling of save-on-type editors

**Ignored Paths:**
- Standard ignored directories: `.git`, `node_modules`, `vendor`, `dist`, `build`, `.vscode`, `.idea`, `__pycache__`, `.next`, `out`
- Hidden files (except code files like `.go`, `.c`, etc.)

### Automatic Activation

The file watcher:
- Starts automatically when the MCP server starts
- Loads all indexed folders from MongoDB
- Monitors folders marked as "active" status
- Handles graceful shutdown on server stop

### Integration with MCP Tools

**code_index_add_folder:**
- Adds folder to index
- **Automatically** starts monitoring for changes
- No manual activation required

**code_index_remove_folder:**
- Removes folder from index
- **Automatically** stops monitoring
- Cleans up all associated data

**code_index_scan:**
- Initial scan still available for bulk indexing
- File watcher handles incremental updates after initial scan

### Performance Impact

- Minimal CPU usage (event-driven, not polling)
- Memory efficient (only tracks folder paths and debounce timers)
- Network efficient (only re-indexes changed files)
- Storage efficient (SHA-256 comparison prevents unnecessary updates)

## Future Enhancements

1. ~~**Incremental Updates:** Watch file system for changes~~ ✅ **IMPLEMENTED**
2. **Language-Specific Parsing:** AST-based chunking for better context
3. **Multi-Language Support:** Hybrid embeddings for polyglot codebases
4. **Advanced Filters:** Filter by language, file type, date range
5. **Batch Operations:** Index multiple folders in parallel
6. **Web UI:** Visual interface for search and management

## License

Copyright © 2025 Hyperion Platform
