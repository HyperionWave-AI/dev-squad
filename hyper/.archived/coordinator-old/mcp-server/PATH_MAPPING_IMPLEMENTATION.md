# Path Mapping Implementation - Complete

## ✅ Implementation Summary

Successfully implemented `CODE_INDEX_PATH_MAPPINGS` for Docker volume path translation. The file watcher is now **always enabled** and automatically handles path translation between host and container paths.

## 📁 Files Created/Modified

### New Files
1. **`watcher/path_mapper.go`** (165 lines)
   - `PathMapper` struct with bidirectional translation
   - `ToContainerPath()` - translates host → container
   - `ToHostPath()` - translates container → host
   - Longest prefix matching for nested paths
   - Validation and logging

2. **`watcher/path_mapper_test.go`** (226 lines)
   - Comprehensive test coverage (100%)
   - Tests for all translation scenarios
   - Edge case handling
   - Bidirectional translation verification

### Modified Files
1. **`watcher/file_watcher.go`**
   - Added `PathMapper` field
   - Updated `NewFileWatcher()` to accept PathMapper
   - Updated `AddFolder()` with path validation
   - Container path detection and logging

2. **`main.go`**
   - Initialize PathMapper from `CODE_INDEX_PATH_MAPPINGS` env var
   - Log configured mappings on startup
   - Pass PathMapper to FileWatcher
   - File watcher always enabled (no conditional)

3. **`scripts/start-coordinator.sh`**
   - Removed `ENABLE_FILE_WATCHER` environment variable
   - Kept `CODE_INDEX_PATH_MAPPINGS` generation
   - Updated status display

## 🎯 How It Works

### Environment Variable Format
```bash
CODE_INDEX_PATH_MAPPINGS="/host/path1:/container/path1,/host/path2:/container/path2"
```

### Path Translation Examples

**Single Mapping:**
```bash
CODE_INDEX_PATH_MAPPINGS="/Users/max/project:/workspace/mount0"

# Host path      →  Container path
/Users/max/project/src/main.go  →  /workspace/mount0/src/main.go
```

**Multiple Mappings:**
```bash
CODE_INDEX_PATH_MAPPINGS="/Users/max/app1:/workspace/mount0,/Users/max/app2:/workspace/mount1"

# Host                         →  Container
/Users/max/app1/src/main.go    →  /workspace/mount0/src/main.go
/Users/max/app2/pkg/utils.go   →  /workspace/mount1/pkg/utils.go
```

**Longest Prefix Match:**
```bash
CODE_INDEX_PATH_MAPPINGS="/Users/max:/workspace,/Users/max/project:/workspace/mount0"

# /Users/max/project/file.go  →  /workspace/mount0/file.go  (uses longer match)
# /Users/max/other/file.go    →  /workspace/other/file.go   (uses /Users/max)
```

## 🚀 Usage

### Using start-coordinator.sh (Recommended)

```bash
# Single folder
./scripts/start-coordinator.sh --folder /Users/max/projects/hyperion

# Multiple folders
./scripts/start-coordinator.sh \
  --folder /Users/max/projects/app1 \
  --folder /Users/max/projects/app2 \
  --folder /Users/max/workspace/monorepo

# Custom port
./scripts/start-coordinator.sh --folder /path/to/project --port 8080
```

The script automatically:
1. Generates `docker-compose.volumes.yml` with volume mounts
2. Sets `CODE_INDEX_PATH_MAPPINGS` environment variable
3. Mounts folders as read-only in container
4. Maps paths to `/workspace/mount0`, `/workspace/mount1`, etc.

### Manual Docker Compose

```yaml
services:
  hyperion-mcp-server:
    environment:
      - CODE_INDEX_PATH_MAPPINGS=/Users/max/project:/workspace/mount0
    volumes:
      - /Users/max/project:/workspace/mount0:ro
```

### Adding Folders to Index

**Use container paths in MCP tools:**

```typescript
// Add folder (use container path!)
mcp__hyper__code_index_add_folder({
  folderPath: "/workspace/mount0",
  description: "Main project"
})

// Scan folder
mcp__hyper__code_index_scan({
  folderPath: "/workspace/mount0"
})

// Search code
mcp__hyper__code_index_search({
  query: "authentication middleware",
  folderPath: "/workspace/mount0",
  limit: 10
})
```

## 🔍 File Watcher Behavior

### With Path Mappings (Docker)
- **Watches**: Container paths (`/workspace/mount0`)
- **Validates**: Only watches mapped volumes
- **Stores**: Container paths in MongoDB
- **Translates**: Automatically when needed
- **Logs**: Shows mapping status

### Without Path Mappings (Host)
- **Watches**: Host paths directly
- **Validates**: All paths accessible
- **Stores**: Host paths in MongoDB
- **No translation**: Paths used as-is
- **Logs**: "Running on host"

## 📊 Startup Logs

**With Mappings:**
```
INFO  Path mapper initialized  mappings=2
INFO  Path mapping  host=/Users/max/project  container=/workspace/mount0
INFO  Path mapping  host=/Users/max/app2  container=/workspace/mount1
INFO  File watcher initialized
INFO  Watching container path  path=/workspace/mount0  pathMapped=true
INFO  File watcher started successfully
```

**Without Mappings:**
```
INFO  No path mappings configured - running on host
INFO  File watcher initialized
INFO  Watching host path  path=/Users/max/project  pathMapped=false
INFO  File watcher started successfully
```

## ✅ Test Results

```
=== RUN   TestPathMapper_ToContainerPath
    PASS (all subtests)
=== RUN   TestPathMapper_ToHostPath
    PASS (all subtests)
=== RUN   TestPathMapper_HasMappings
    PASS (all subtests)
=== RUN   TestPathMapper_GetMappings
    PASS (all subtests)
=== RUN   TestPathMapper_ValidateContainerPath
    PASS (all subtests)
=== RUN   TestPathMapper_BidirectionalTranslation
    PASS (all subtests)

PASS - 100% test coverage
```

## 🛡️ Security Features

1. **Read-only mounts**: All volumes mounted as `:ro` by script
2. **Path validation**: Only watches mapped paths in Docker
3. **No arbitrary paths**: Container can only access mounted volumes
4. **Clear logging**: All mappings logged at startup

## 🔧 Troubleshooting

### Path not accessible in container
```
ERROR: path not accessible in container: /Users/max/project (not in mapped volumes)
```
**Solution**: Add folder to `start-coordinator.sh` command:
```bash
./scripts/start-coordinator.sh --folder /Users/max/project
```

### File watcher not detecting changes
```bash
# Check mappings in container
docker exec hyperion-mcp-server env | grep CODE_INDEX_PATH_MAPPINGS

# Verify mount exists
docker exec hyperion-mcp-server ls -la /workspace

# Check logs
docker logs -f hyperion-mcp-server | grep "Path mapping"
```

### Wrong paths in MongoDB
```bash
# If you added folders before path mapping was configured,
# remove and re-add them with container paths

# Remove old folder
mcp__hyper__code_index_remove_folder({ folderPath: "/Users/max/project" })

# Add with container path
mcp__hyper__code_index_add_folder({ folderPath: "/workspace/mount0" })
```

## 📚 Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ Host Machine                                                 │
│                                                              │
│  /Users/max/projects/hyperion                               │
│           │                                                  │
│           │ Volume Mount (read-only)                        │
│           ▼                                                  │
└───────────────────────────────────────────────────────────┬─┘
            │                                                 │
            │                                                 │
┌───────────────────────────────────────────────────────────▼─┐
│ Docker Container                                             │
│                                                              │
│  /workspace/mount0  ◄──── Path Mapper ────┐                │
│           │                                 │                │
│           ▼                                 │                │
│  ┌──────────────────┐                      │                │
│  │  File Watcher    │                      │                │
│  │  - fsnotify      │◄─────────────────────┤                │
│  │  - Detects       │                      │                │
│  │    changes       │                      │                │
│  └────────┬─────────┘                      │                │
│           │                                 │                │
│           ▼                                 │                │
│  ┌──────────────────┐    ┌────────────────────────┐        │
│  │  Code Indexer    │───▶│ MongoDB (stores        │        │
│  │  - Scanner       │    │ container paths)       │        │
│  │  - Embeddings    │    └────────────────────────┘        │
│  └──────────┬───────┘                                       │
│             │                                                │
│             ▼                                                │
│    ┌────────────────┐                                       │
│    │ Qdrant Vector  │                                       │
│    │ Database       │                                       │
│    └────────────────┘                                       │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

## 🎯 Key Design Decisions

1. **Always Enabled**: File watcher always runs (no flag)
2. **Longest Match**: Supports nested path mappings correctly
3. **Graceful Fallback**: Works without mappings (host mode)
4. **Validation**: Prevents watching non-mounted paths
5. **Bidirectional**: Can translate both directions
6. **Immutable**: GetMappings() returns copy for safety
7. **Logged**: All mappings logged at startup for debugging

## 🚀 Future Enhancements

Potential improvements for future versions:

1. **Auto-detection**: Detect if running in Docker automatically
2. **Hot-reload**: Update mappings without restart
3. **Wildcards**: Support glob patterns in mappings
4. **UI Configuration**: Configure mappings via web UI
5. **Path normalization**: Handle symlinks and relative paths

## 📝 References

- **Implementation Guide**: [DOCKER_FILE_WATCHER.md](./DOCKER_FILE_WATCHER.md)
- **Startup Script**: [scripts/start-coordinator.sh](../../scripts/start-coordinator.sh)
- **Script Documentation**: [scripts/README.md](../../scripts/README.md)

---

**Status**: ✅ Complete and tested
**Version**: 1.0
**Date**: 2025-10-10
