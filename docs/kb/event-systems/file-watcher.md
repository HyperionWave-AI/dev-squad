# Hyperion File Watcher - Filesystem Event Processing

**Collection:** event-systems
**Tags:** file-watching, fsnotify, event-queue, debouncing
**File Reference:** internal/mcp/watcher/file_watcher.go:21-81
**Version:** 1.0

---

HYPERION FILE WATCHER - FILESYSTEM EVENT PROCESSING

Real-time code indexing via filesystem monitoring with debouncing and worker pool pattern.

FILE WATCHER (internal/mcp/watcher/file_watcher.go):
Components:
- watcher: *fsnotify.Watcher (monitors OS-level changes)
- eventQueue: Buffered channel (size 100) for async event processing
- workerCount: 3 concurrent workers process events
- debounceTime: 500ms prevents rapid re-indexing of same file
- watchedFolders: Map tracks active indexed folders

EVENT PIPELINE:
1. FSNotify detects: create, modify, delete operations
2. Event enqueued to eventQueue (non-blocking)
3. Worker pool dequeues and processes (3 goroutines)
4. Debounce timer: Skip if processed <500ms ago
5. Actions: Re-index file → Update MongoDB → Refresh Qdrant vectors

INITIALIZATION (main.go:464-468):
```go
fileWatcher, err := watcher.NewFileWatcher(
  codeIndexStorage, qdrantClient, embeddingClient, pathMapper, logger)
```

AUTO-INDEXING AT STARTUP (main.go:585-626):
- Loads indexed folders from MongoDB
- Checks if project root needs indexing:
  - New mapping: needsIndexing = true
  - Collection empty: needsIndexing = true
  - FORCE_REINDEX=true env: needsIndexing = true
- Launches background goroutine to index project root
- Non-blocking: Server continues while indexing

LIFECYCLE:
Start(): Load folders → Add to watch list → Begin event processing
Stop(): Close watcher → Cancel context → Wait for workers

CONFIGURATION:
- ENABLE_FILE_WATCHER: true (default) | false (disable)
- PATH_MAPPINGS: Map project paths to Qdrant collections
- CODE_INDEX_AUTO_SCAN: Trigger auto-scan on startup

PERFORMANCE:
- Debouncing: Batch rapid changes (e.g., file save + IDE format)
- Worker pool: 3 concurrent workers prevent event queue overflow
- Buffered channel: Non-blocking enqueue
- Background indexing: Doesn't block server startup
