# Makefile Cleanup Summary

## Project Overview: Hyperion (hyper)

**Hyperion** is an AI-powered code analysis platform that integrates with Claude Code. It provides:
- **Code Indexing**: Semantic understanding of code files
- **Vector Search**: Find similar code patterns
- **Claude Integration**: Works as MCP plugin for Claude Code
- **REST API**: HTTP endpoints for programmatic access
- **Web UI**: React-based interface
- **Real-time Monitoring**: Auto-indexes file changes

### Single Binary Architecture
The project uses a **single unified binary** (`hyper`) with three runtime modes instead of separate binaries:

```
./bin/hyper --mode=http   → REST API + Web UI (port 7095)
./bin/hyper --mode=mcp    → MCP stdio (Claude Code)
./bin/hyper --mode=both   → Both modes (default)
```

---

## Changes Made

### 1. Root Makefile (`/Makefile`) - ✅ Cleaned Up

**Removed:**
- Old `build` target that compiled `coordinator/mcp-server`
- Old `run-mcp-local` target for HTTP MCP server
- Old `configure-claude-local` target for HTTP transport
- Old `test-connection` target for separate mcp-server

**Updated:**
- `build` → Now alias for `native` (unified binary)
- `native` → Builds unified hyper binary via `./build-native.sh`
- `install` → Now installs deps from `hyper/` instead of `coordinator/mcp-server`
- `test` → Runs tests in `hyper/` directory
- `clean` → Cleans unified binary artifacts

**Added:**
- `run-mcp-http` → Run unified binary in HTTP mode on port 7095

**Kept (Working):**
- `native` → Build unified hyper binary with embedded UI
- `dev` / `dev-hot` → Development with hot reload
- `run` / `run-dev` / `run-stdio` → Run modes for unified binary
- `configure-native` → Configure Claude Code for MCP stdio
- `desktop` / `desktop-build` → Desktop app targets
- `install-air` → Install Air hot-reload tool

### 2. hyper/Makefile (`/hyper/Makefile`) - ✅ Simplified

**Before:** Multiple binaries (coordinator, mcp-server, bridge, indexer)

**After:** Single unified binary

**Targets:**
- `build` → Build unified `bin/hyper` from `cmd/coordinator`
- `clean` → Clean build artifacts
- `test` → Run all tests with coverage
- `install-tools` → Install Air for hot reload
- `dev` → Run with Air hot reload
- `install` → Install Go dependencies

### 3. coordinator/Makefile (`/coordinator/Makefile`) - ⚠️ Legacy

**Status:** Kept for reference, but **not used** by main build

This Makefile still references the old architecture:
- Separate UI build
- Separate coordinator binary
- Docker compose targets

**Recommendation:** Can be deleted or archived as it's superseded by root Makefile + build-native.sh

### 4. build-native.sh - ✅ Updated

**Changed:**
- Build source: `hyper/cmd/coordinator` (was `coordinator/cmd/coordinator`)
- UI source: Still `coordinator/ui` (UI code hasn't moved)
- Embed target: `hyper/embed/ui` (was `coordinator/embed/ui`)

**Result:** Builds single unified binary at `bin/hyper`

### 5. Redundant Binaries - ✅ Archived

**Archived to `hyper/.archived/`:**
- `cmd/bridge/` → Placeholder, HTTP bridge already in unified binary
- `cmd/mcp-server/` → Duplicate of `--mode=mcp` in unified binary
- `cmd/indexer/` → Code indexing already in unified binary
- `cmd/hyper/` → Empty directory (was placeholder)

**Remaining:**
- `cmd/coordinator/` → The unified binary (contains all features)

## Technology Stack

### Backend (Go 1.25)
| Component | Technology | Purpose |
|-----------|-----------|---------|
| Framework | Gin Web Framework | HTTP server & routing |
| Protocol | MCP Go SDK | Claude Code integration |
| Database | MongoDB | Metadata, tasks, history |
| Vector DB | Qdrant | Semantic search |
| File Watching | fsnotify | Real-time monitoring |
| Embeddings | Multiple providers | Vector generation |
| Logging | Uber Zap | Structured logging |
| LLM Chain | LangChain Go | AI orchestration |
| Auth | golang-jwt | JWT tokens |
| WebSocket | Gorilla WebSocket | Real-time updates |

### Frontend (React)
- **Framework**: React 18+
- **Build Tool**: Vite
- **Embedded**: In Go binary via `embed` package

### Embedding Providers
- **Ollama**: Local, GPU-accelerated (default)
- **OpenAI**: Cloud-based, high quality
- **Voyage AI**: Specialized embeddings
- **TEI**: Self-hosted embeddings

## Verified Working Commands

### Build Commands
```bash
make build           # Build unified hyper binary with embedded UI
make native          # Same as build
make clean           # Clean all build artifacts
```

### Development Commands
```bash
make dev             # Hot reload (Air) without UI dev server
make dev-hot         # Hot reload with Vite UI dev server
make install         # Install Go and Node dependencies
make install-air     # Install Air tool
```

### Run Commands
```bash
make run             # Run in HTTP mode (REST API + UI)
make run-stdio       # Run in MCP stdio mode (for Claude Code)
make run-mcp-http    # Run in HTTP mode (explicit)
make run-dev         # Run with Air hot reload
```

### MCP Configuration
```bash
make configure-native  # Configure Claude Code to use unified binary
```

### Test Commands
```bash
make test            # Run all tests
```

### Desktop App
```bash
make desktop         # Run desktop app (dev mode)
make desktop-build   # Build desktop app for distribution
```

## Architecture Summary

### Before Cleanup
```
coordinator/
├── cmd/coordinator/main.go      (REST API)
├── mcp-server/main.go           (MCP stdio)
└── mcp-http-bridge/main.go      (MCP HTTP)

hyper/
├── cmd/coordinator/main.go      (Unified)
├── cmd/mcp-server/main.go       (Duplicate)
├── cmd/bridge/main.go           (Placeholder)
└── cmd/indexer/main.go          (Duplicate)
```

### After Cleanup
```
hyper/
└── cmd/coordinator/main.go      (Unified: REST + MCP stdio + MCP HTTP)
    --mode=http    → REST API + UI
    --mode=mcp     → MCP stdio
    --mode=both    → Both (default)

.archived/
├── cmd/bridge/
├── cmd/mcp-server/
├── cmd/indexer/
└── cmd/hyper/
```

## Build Flow

```
User runs: make native
    ↓
Calls: ./build-native.sh
    ↓
1. Build UI: coordinator/ui → coordinator/ui/dist
2. Embed UI: Copy dist to hyper/embed/ui/
3. Build Go: hyper/cmd/coordinator → bin/hyper
4. Result: Single ~16MB binary with embedded UI
```

## Core Features

### Code Indexing & Analysis
- Real-time file watching using fsnotify
- Automatic code parsing and tokenization
- Semantic indexing with embeddings
- Incremental updates for performance
- Multi-language support

### Semantic Search
- Vector-based similarity search via Qdrant
- Code snippet retrieval by semantic meaning
- Context-aware search using embeddings
- Filtering and ranking capabilities

### Claude Code Integration (MCP)
- stdio protocol for direct Claude integration
- Tool definitions in JSON Schema format
- Real-time code analysis from Claude
- Bi-directional communication

### REST API + Web UI
- RESTful endpoints for all operations
- React-based web interface on port 7095
- Real-time updates via WebSocket
- JWT authentication
- CORS support

### File Watching & Auto-Indexing
- Recursive directory monitoring
- Automatic re-indexing on file changes
- Batch processing for efficiency
- Configurable watch patterns

## Testing

### Build Test
```bash
make clean
make native
./bin/hyper --version  # Should show version info
```

### Run Test (HTTP Mode)
```bash
export MONGODB_URI="mongodb+srv://..."
export QDRANT_URL="https://..."
export EMBEDDING="voyage"
export VOYAGE_API_KEY="..."
./bin/hyper --mode=http
# Visit: http://localhost:7095
```

### Run Test (MCP Mode)
```bash
./bin/hyper --mode=mcp
# Should start stdio server for Claude Code
```

## Configuration

### Environment Variables
```bash
# MongoDB
MONGODB_URI="mongodb+srv://user:pass@cluster.mongodb.net"
MONGODB_DATABASE="coordinator_db1"

# Qdrant Vector Database
QDRANT_URL="https://qdrant-instance.com"
QDRANT_KNOWLEDGE_COLLECTION="dev_squad_knowledge"

# Embedding Provider (ollama|openai|voyage|tei)
EMBEDDING="ollama"

# Ollama Configuration
OLLAMA_URL="http://localhost:11434"
OLLAMA_MODEL="nomic-embed-text"

# OpenAI Configuration
OPENAI_API_KEY="sk-..."

# Voyage AI Configuration
VOYAGE_API_KEY="pa-..."

# Server Configuration
PORT="7095"
LOG_LEVEL="info"
```

### Configuration File
- **Location**: `.env.hyper` (in executable directory or current directory)
- **Priority**: Custom config path > executable dir > current dir
- **Format**: Standard `.env` format

## Recommendations

### Optional Cleanup (Future)
1. **Rename cmd/coordinator → cmd/hyper** for clarity
2. **Archive coordinator/ directory** (keep only coordinator/ui)
3. **Update README** to emphasize single binary approach
4. **Delete coordinator/Makefile** (redundant)

### Keep As-Is
- Root Makefile ✅
- hyper/Makefile ✅
- build-native.sh ✅
- hyper/cmd/coordinator/ ✅

## Directory Structure

```
hyper/
├── cmd/
│   └── coordinator/          # Unified binary source
│       └── main.go          # --mode flag: http|mcp|both
├── internal/
│   ├── server/              # HTTP server (Gin)
│   ├── mcp/                 # MCP protocol
│   │   ├── handlers/        # MCP tools
│   │   ├── storage/         # MongoDB + Qdrant
│   │   ├── embeddings/      # TEI/Voyage/Ollama/OpenAI
│   │   ├── indexer/         # Code indexing
│   │   └── watcher/         # File watching
│   ├── middleware/          # HTTP middleware
│   └── ai-service/          # AI integration
├── embed/                   # Embedded UI (auto-generated)
├── go.mod                   # Go dependencies
└── Makefile                 # Build targets

coordinator/
└── ui/                      # React UI source
    ├── src/
    ├── dist/                # Built UI (auto-generated)
    └── package.json
```

## Conclusion

✅ **Makefile cleanup complete**
✅ **Redundant binaries archived**
✅ **Build system unified**
✅ **All targets tested and working**

The build system now focuses on the **single unified hyper binary** approach. All redundant targets removed, all working targets preserved.

## Quick Reference

**One binary, three modes:**
```bash
./bin/hyper --mode=http   # REST API + UI (port 7095)
./bin/hyper --mode=mcp    # MCP stdio (Claude Code)
./bin/hyper --mode=both   # Both (default)
```

**Build:**
```bash
make native  # Single command builds everything
```

**Develop:**
```bash
make dev-hot  # Hot reload for Go + UI
```

**Key Capabilities:**
- AI-powered code analysis
- Semantic code search
- Claude Code integration
- REST API + Web UI
- Real-time file monitoring
- Vector embeddings
- Multiple embedding providers

---

*Last Updated: 2024*
*Project: Hyperion (hyper)*
