# File Watcher in Docker - Solutions

## Problem

The file watcher monitors host filesystem paths (e.g., `/Users/max/project`), but Docker containers don't have access to these paths unless explicitly mounted. When running in Docker:

1. Paths stored in MongoDB are host paths
2. Container filesystem is isolated from host
3. `fsnotify` watcher fails to watch non-existent paths
4. No file change notifications reach the service

## Solutions

### ✅ **Solution 1: Path Mapping with Volume Mounts (RECOMMENDED)**

Map host paths to container paths using environment variables and volume mounts.

#### Configuration

**Add to `.env`:**
```bash
# Path mappings: HOST_PATH:CONTAINER_PATH
CODE_INDEX_PATH_MAPPINGS=/Users/max/projects:/workspace/projects,/Users/max/hyperion:/workspace/hyperion

# Enable/disable file watching (set to false in production)
ENABLE_FILE_WATCHER=true
```

**Add to `docker-compose.yml`:**
```yaml
services:
  hyperion-mcp-server:
    environment:
      - CODE_INDEX_PATH_MAPPINGS=${CODE_INDEX_PATH_MAPPINGS}
      - ENABLE_FILE_WATCHER=${ENABLE_FILE_WATCHER:-false}
    volumes:
      # Mount host directories you want to index
      - /Users/max/projects:/workspace/projects:ro
      - /Users/max/hyperion:/workspace/hyperion:ro
```

**Implementation:**
```go
// watcher/path_mapper.go
type PathMapper struct {
    mappings map[string]string  // host -> container
    reverse  map[string]string  // container -> host
}

func NewPathMapper(mappingsEnv string) *PathMapper {
    // Parse: "/host/path1:/container/path1,/host/path2:/container/path2"
    mappings := make(map[string]string)
    reverse := make(map[string]string)

    if mappingsEnv == "" {
        return &PathMapper{mappings, reverse}
    }

    pairs := strings.Split(mappingsEnv, ",")
    for _, pair := range pairs {
        parts := strings.Split(pair, ":")
        if len(parts) == 2 {
            host := strings.TrimSpace(parts[0])
            container := strings.TrimSpace(parts[1])
            mappings[host] = container
            reverse[container] = host
        }
    }

    return &PathMapper{mappings, reverse}
}

func (pm *PathMapper) ToContainerPath(hostPath string) string {
    for host, container := range pm.mappings {
        if strings.HasPrefix(hostPath, host) {
            return strings.Replace(hostPath, host, container, 1)
        }
    }
    return hostPath  // No mapping found
}

func (pm *PathMapper) ToHostPath(containerPath string) string {
    for container, host := range pm.reverse {
        if strings.HasPrefix(containerPath, container) {
            return strings.Replace(containerPath, container, host, 1)
        }
    }
    return containerPath
}
```

**Modify `watcher/file_watcher.go`:**
```go
type FileWatcher struct {
    // ... existing fields
    pathMapper *PathMapper
}

func NewFileWatcher(..., pathMapper *PathMapper) (*FileWatcher, error) {
    // ... existing code
    fw.pathMapper = pathMapper
    return fw, nil
}

func (fw *FileWatcher) AddFolder(folder *storage.IndexedFolder) error {
    // Translate host path to container path
    containerPath := fw.pathMapper.ToContainerPath(folder.Path)

    // Check if path exists in container
    if _, err := os.Stat(containerPath); os.IsNotExist(err) {
        return fmt.Errorf("path not mounted in container: %s (host: %s)",
            containerPath, folder.Path)
    }

    // Watch container path
    if err := fw.watcher.Add(containerPath); err != nil {
        return fmt.Errorf("failed to watch folder: %w", err)
    }

    // ... rest of existing code
}
```

**Modify `main.go`:**
```go
// Check if file watcher should be enabled
enableWatcher := os.Getenv("ENABLE_FILE_WATCHER")
if enableWatcher == "" {
    enableWatcher = "true"  // Default enabled for backward compatibility
}

if enableWatcher == "true" {
    // Initialize path mapper
    pathMappings := os.Getenv("CODE_INDEX_PATH_MAPPINGS")
    pathMapper := watcher.NewPathMapper(pathMappings)

    // Initialize file watcher with path mapper
    fileWatcher, err := watcher.NewFileWatcher(
        codeIndexStorage,
        qdrantClient,
        embeddingClient,
        pathMapper,
        logger,
    )
    if err != nil {
        logger.Warn("Failed to create file watcher", zap.Error(err))
    } else {
        if err := fileWatcher.Start(); err != nil {
            logger.Warn("Failed to start file watcher", zap.Error(err))
        } else {
            logger.Info("File watcher started successfully")
        }
    }
} else {
    logger.Info("File watcher disabled via ENABLE_FILE_WATCHER=false")
}
```

**Pros:**
- ✅ Flexible - works for any number of projects
- ✅ Secure - read-only mounts prevent accidental modifications
- ✅ Explicit - clearly shows what's being watched
- ✅ Production-ready - can disable in production

**Cons:**
- ❌ Requires configuration for each project
- ❌ Large volume mounts can be slow on some platforms (Mac/Windows with Docker Desktop)

---

### ✅ **Solution 2: Workspace Volume with Relative Paths**

Mount a single workspace directory and use relative paths.

**Add to `docker-compose.yml`:**
```yaml
services:
  hyperion-mcp-server:
    environment:
      - CODE_INDEX_WORKSPACE=/workspace
    volumes:
      # Mount your entire workspace
      - /Users/max/workspace:/workspace:ro
```

**When adding folders via MCP:**
```typescript
// User adds with host path
mcp__hyper__code_index_add_folder({
  folderPath: "/Users/max/workspace/hyperion"
})

// Service translates to container path
// Stores in MongoDB: { hostPath: "/Users/max/workspace/hyperion", containerPath: "/workspace/hyperion" }
// Watches: /workspace/hyperion
```

**Implementation:**
```go
// In code_tools.go
func (h *CodeToolsHandler) handleAddFolder(...) {
    workspaceEnv := os.Getenv("CODE_INDEX_WORKSPACE")

    if workspaceEnv != "" {
        // Running in Docker with workspace
        // Check if folderPath is under workspace
        if !strings.HasPrefix(folderPath, workspaceEnv) {
            return &mcp.CallToolResult{
                IsError: true,
                Content: []interface{}{&mcp.TextContent{
                    Text: fmt.Sprintf(
                        "Path must be under workspace: %s\nGot: %s",
                        workspaceEnv, folderPath,
                    ),
                }},
            }, nil
        }
    }

    // Store both host and container paths
    folder := &storage.IndexedFolder{
        HostPath:      folderPath,      // Original path
        ContainerPath: containerPath,   // Translated path
        // ...
    }
}
```

**Pros:**
- ✅ Simple configuration (single mount)
- ✅ Works well for monorepo or single workspace
- ✅ Less configuration overhead

**Cons:**
- ❌ Limited to single workspace directory
- ❌ All projects must be under one root

---

### ✅ **Solution 3: Disable File Watcher, Use Manual Scans**

Disable automatic watching and provide manual/periodic scanning.

**Add to `docker-compose.yml`:**
```yaml
services:
  hyperion-mcp-server:
    environment:
      - ENABLE_FILE_WATCHER=false
      - CODE_INDEX_SCAN_INTERVAL=300  # Scan every 5 minutes (optional)
```

**Implementation:**
```go
// In main.go
enableWatcher := os.Getenv("ENABLE_FILE_WATCHER")
if enableWatcher != "true" {
    logger.Info("File watcher disabled - using manual scan mode")

    // Optional: periodic scanning
    scanInterval := os.Getenv("CODE_INDEX_SCAN_INTERVAL")
    if scanInterval != "" {
        interval, _ := strconv.Atoi(scanInterval)
        go startPeriodicScanner(codeIndexStorage, qdrantClient, interval, logger)
    }
} else {
    // Start file watcher
}

func startPeriodicScanner(storage *storage.CodeIndexStorage, ..., intervalSec int) {
    ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        folders, _ := storage.ListFolders()
        for _, folder := range folders {
            if folder.Status == "active" {
                logger.Info("Periodic scan", zap.String("folder", folder.Path))
                // Trigger scan via code_index_scan tool
            }
        }
    }
}
```

**Manual trigger via MCP:**
```typescript
// User manually triggers scan
mcp__hyper__code_index_scan({
  folderPath: "/workspace/hyperion"
})
```

**Pros:**
- ✅ No volume mount requirements
- ✅ Works with any storage (NFS, network drives)
- ✅ Predictable resource usage (scans at intervals)
- ✅ Production-ready

**Cons:**
- ❌ No real-time updates
- ❌ Requires manual or periodic scans
- ❌ Potential delay in index freshness

---

### ✅ **Solution 4: Hybrid - Watch Mounted, Poll Unmounted**

Combine file watching for mounted paths and periodic scanning for others.

**Implementation:**
```go
func (fw *FileWatcher) AddFolder(folder *storage.IndexedFolder) error {
    containerPath := fw.pathMapper.ToContainerPath(folder.Path)

    // Try to watch if path exists
    if _, err := os.Stat(containerPath); err == nil {
        // Path exists - use fsnotify watcher
        if err := fw.watcher.Add(containerPath); err == nil {
            fw.logger.Info("Watching folder with fsnotify",
                zap.String("path", containerPath))
            return nil
        }
    }

    // Path not mounted - use polling
    fw.logger.Info("Path not mounted, will use periodic scanning",
        zap.String("hostPath", folder.Path))
    fw.addToPollingQueue(folder)
    return nil
}
```

**Pros:**
- ✅ Best of both worlds
- ✅ Flexible - works with any combination
- ✅ Optimal performance where possible

**Cons:**
- ❌ More complex implementation
- ❌ Mixed behavior (some real-time, some delayed)

---

## Recommendation

**For Development:** Use **Solution 1 (Path Mapping)** with volume mounts
- Provides real-time file watching
- Explicit and configurable
- Works for multiple projects

**For Production:** Use **Solution 3 (Disabled Watcher)** with manual scans
- More predictable and controllable
- No volume mount overhead
- Trigger scans via API or periodic schedule

---

## Implementation Priority

1. **Phase 1 (Immediate):** Add `ENABLE_FILE_WATCHER` environment variable to disable watcher in Docker
2. **Phase 2 (Short-term):** Implement path mapping with volume mounts
3. **Phase 3 (Future):** Add periodic scanning option and hybrid mode

---

## Example Docker Compose Configuration

```yaml
# docker-compose.yml
services:
  hyperion-mcp-server:
    environment:
      # Disable file watcher in production
      - ENABLE_FILE_WATCHER=false
    # No volumes needed if watcher disabled

# docker-compose.dev.yml
services:
  hyperion-mcp-server:
    environment:
      # Enable file watcher in development
      - ENABLE_FILE_WATCHER=true
      - CODE_INDEX_PATH_MAPPINGS=/Users/max/projects:/workspace
    volumes:
      # Mount projects directory
      - /Users/max/projects:/workspace:ro
```

---

## Testing

```bash
# Test with watcher disabled
docker-compose up -d
docker logs hyperion-mcp-server | grep "File watcher disabled"

# Test with watcher enabled + volume mount
docker-compose -f docker-compose.yml -f docker-compose.dev.yml up -d
docker logs hyperion-mcp-server | grep "File watcher started"

# Test path mapping
docker exec hyperion-mcp-server ls -la /workspace
```
