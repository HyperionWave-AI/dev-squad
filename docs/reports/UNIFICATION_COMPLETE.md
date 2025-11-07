# Go Projects Unification - COMPLETE ✅

## Summary

The unification of all Go projects under `./hyper` is **already implemented**. The `hyper/cmd/coordinator/main.go` serves as the unified entry point for all services.

## Current Architecture

### Single Unified Binary: `hyper/cmd/coordinator/main.go`

**Supports 3 modes via `--mode` flag:**

```bash
# REST API + Web UI only
./hyper --mode=http

# MCP stdio server only (for Claude Code integration)
./hyper --mode=mcp

# Both HTTP and MCP simultaneously (default)
./hyper --mode=both
```

### Features Already Implemented

✅ **Single binary** with embedded UI (~16MB)
✅ **Concurrent server management** using goroutines and channels
✅ **Shared resources:**
  - MongoDB connection pool (coordinator tasks)
  - Qdrant client (knowledge base)
  - Embedding client (TEI/Voyage/Ollama/OpenAI)
  - File watcher (code indexing)

✅ **Graceful shutdown** with signal handling (SIGTERM/SIGINT)
✅ **Cross-platform** support (macOS, Windows, Linux)
✅ **Hot reload** support for development
✅ **Environment-based configuration** (.env.hyper)

## Redundant Binaries (Can be Removed)

### 1. `hyper/cmd/bridge/main.go`
**Status:** Placeholder only
**Reason:** HTTP mode already provides MCP HTTP endpoints via `/mcp` route
**Action:** Can be deleted

### 2. `hyper/cmd/mcp-server/main.go`
**Status:** Duplicate of MCP mode
**Reason:** `coordinator --mode=mcp` provides same functionality
**Action:** Keep for now as standalone option, or delete if not needed

### 3. `hyper/cmd/indexer/main.go`
**Status:** Separate indexing service
**Reason:** File watcher is integrated into unified binary
**Action:** Keep if used for batch indexing, otherwise delete

## Recommended Next Steps

### 1. Rename for Clarity (Optional)
```bash
# Rename cmd/coordinator to cmd/hyper for clarity
mv hyper/cmd/coordinator hyper/cmd/hyper
```

Update build scripts to reflect new name.

### 2. Update Documentation
- Update README-HYPER.md to emphasize single binary approach
- Document the `--mode` flag usage
- Remove references to separate binaries

### 3. Clean Up Build System

**Current `build-native.sh`:**
```bash
#!/bin/bash
# Builds the unified hyper binary with embedded UI

# Build UI
cd coordinator/ui && npm run build && cd ../..

# Embed UI and build binary
cd hyper && go build -o ../bin/hyper ./cmd/coordinator
```

**Updated version (if renamed):**
```bash
cd hyper && go build -o ../bin/hyper ./cmd/hyper
```

### 4. Update docker-compose.yml

**Current:** Separate services
```yaml
services:
  coordinator:
    ...
  mcp-server:
    ...
  http-bridge:
    ...
```

**Unified:**
```yaml
services:
  hyper:
    build: ./hyper
    command: ["--mode=both"]
    ports:
      - "7095:7095"  # HTTP + UI
    environment:
      - MONGODB_URI=${MONGODB_URI}
      - QDRANT_URL=http://qdrant:6333
      - EMBEDDING=ollama
      - OLLAMA_URL=http://host.docker.internal:11434
    depends_on:
      - mongodb
      - qdrant
```

### 5. Claude Code Integration

**MCP Server Configuration** (`~/.claude/config.json`):
```json
{
  "mcpServers": {
    "hyper-dev": {
      "command": "/path/to/bin/hyper",
      "args": ["--mode=mcp"],
      "env": {
        "MONGODB_URI": "mongodb+srv://...",
        "QDRANT_URL": "https://...",
        "EMBEDDING": "voyage",
        "VOYAGE_API_KEY": "..."
      }
    }
  }
}
```

## File Structure (Final)

```
hyper/
├── cmd/
│   └── hyper/              # Single unified entry point
│       └── main.go         # --mode flag: http|mcp|both
├── internal/
│   ├── server/             # HTTP server implementation
│   ├── mcp/                # MCP protocol implementation
│   │   ├── handlers/       # MCP tool handlers
│   │   ├── storage/        # Qdrant + MongoDB
│   │   ├── embeddings/     # TEI/Voyage/Ollama/OpenAI
│   │   └── watcher/        # File watching for code index
│   ├── middleware/         # HTTP middleware (auth, CORS)
│   └── ai-service/         # AI provider integration
├── embed/                  # Embedded UI (from coordinator/ui build)
├── go.mod
└── go.sum
```

## What coordinator/ Directory Contains

The `coordinator/` directory still exists with:
- `coordinator/cmd/coordinator/main.go` - Old separate binary (duplicates hyper)
- `coordinator/mcp-server/` - Old MCP server (duplicates hyper --mode=mcp)
- `coordinator/mcp-http-bridge/` - Old HTTP bridge (duplicates hyper --mode=http MCP endpoint)
- `coordinator/ui/` - React UI source (gets built and embedded into hyper binary)

**Recommendation:** The coordinator/ directory can be:
1. **Archived** - Keep for reference but don't build
2. **Deleted** - If all functionality confirmed in hyper/
3. **UI only** - Keep only coordinator/ui/ for UI development

## Migration Path for Developers

### From Old Architecture (3 binaries):
```bash
# Old way
./coordinator/cmd/coordinator/coordinator &
./coordinator/mcp-server/mcp-server &
./coordinator/mcp-http-bridge/mcp-http-bridge &
```

### To New Architecture (1 binary):
```bash
# New way - single binary with all features
./bin/hyper --mode=both

# Or specific modes
./bin/hyper --mode=http   # Just REST API + UI
./bin/hyper --mode=mcp    # Just MCP stdio for Claude Code
```

## Build Commands

### Production Build (Single Binary):
```bash
./build-native.sh
```

Creates: `bin/hyper` (~16MB with embedded UI)

### Development Mode (Hot Reload):
```bash
make dev-hot
```

Uses Air to watch files and rebuild on changes.

### Local Development (No Docker):
```bash
# Terminal 1: Start dependencies
brew services start mongodb-community
brew services start ollama
ollama pull nomic-embed-text

# Terminal 2: Run unified binary
./bin/hyper --mode=both
```

## Environment Configuration

**File:** `.env.hyper` (placed next to binary or in current directory)

```bash
# Database
MONGODB_URI=mongodb+srv://...
MONGODB_DATABASE=coordinator_db

# Vector Store
QDRANT_URL=https://...
QDRANT_API_KEY=...
QDRANT_KNOWLEDGE_COLLECTION=dev_squad_knowledge

# Embeddings
EMBEDDING=voyage  # Options: ollama, local, openai, voyage
VOYAGE_API_KEY=...
# or
OLLAMA_URL=http://localhost:11434
OLLAMA_MODEL=nomic-embed-text

# Server Mode
HTTP_PORT=7095

# Code Indexing
CODE_INDEX_FOLDERS=/path/to/project1,/path/to/project2
CODE_INDEX_AUTO_SCAN=true
```

## Testing

### Test REST API:
```bash
./bin/hyper --mode=http &
curl http://localhost:7095/api/v1/health
curl http://localhost:7095/api/v1/tasks/human
```

### Test MCP Server:
```bash
./bin/hyper --mode=mcp
# MCP server runs on stdio - test with Claude Code
```

### Test Both:
```bash
./bin/hyper --mode=both
# Access UI at http://localhost:7095
# MCP available on stdio
```

## Conclusion

**Status:** ✅ **UNIFICATION COMPLETE**

The unified `hyper` binary already exists and works. No additional code changes needed for core unification.

**Optional improvements:**
1. Rename `cmd/coordinator` → `cmd/hyper` for clarity
2. Clean up redundant binaries in `cmd/bridge` and `cmd/mcp-server`
3. Archive or remove old `coordinator/` directory structure
4. Update documentation and build scripts

**Current working state:**
- Single binary: ✅
- Concurrent servers: ✅
- Shared resources: ✅
- Embedded UI: ✅
- Mode selection: ✅
- Cross-platform: ✅
- Hot reload: ✅

No further implementation required - the architecture you requested is already in place!
