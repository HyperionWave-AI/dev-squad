# Archive Summary - Deprecated Go Projects

**Date:** 2025-10-12
**Action:** Archived all unused Go projects
**Location:** `hyper/.archived/coordinator-old/`

---

## 📦 What Was Archived

All deprecated Go code from the old `coordinator/` directory has been moved to `hyper/.archived/coordinator-old/`.

### Archived Projects

| Project | Original Location | Archive Location | Size | Status |
|---------|------------------|------------------|------|--------|
| **Old Coordinator** | `coordinator/cmd/coordinator/` | `hyper/.archived/coordinator-old/cmd/` | 22MB binary | ❌ Deprecated |
| **Old MCP Server** | `coordinator/mcp-server/` | `hyper/.archived/coordinator-old/mcp-server/` | 38 files | ❌ Deprecated |
| **Old HTTP Bridge** | `coordinator/mcp-http-bridge/` | `hyper/.archived/coordinator-old/mcp-http-bridge/` | 26 files | ❌ Deprecated |
| **Old Internal Packages** | `coordinator/internal/` | `hyper/.archived/coordinator-old/internal/` | 8 dirs | ❌ Deprecated |
| **Old AI Service** | `coordinator/ai-service/` | `hyper/.archived/coordinator-old/ai-service/` | 11 files | ❌ Deprecated |
| **Old Embed** | `coordinator/embed/` | `hyper/.archived/coordinator-old/embed/` | 4 files | ❌ Deprecated |
| **Go Modules** | `coordinator/go.mod`, `go.sum` | `hyper/.archived/coordinator-old/` | - | ❌ Deprecated |
| **Binaries** | `coordinator/coordinator`, `coordinator/bin/` | `hyper/.archived/coordinator-old/` | 22MB | ❌ Deprecated |
| **Build Config** | `coordinator/.air.toml`, Dockerfiles | `hyper/.archived/coordinator-old/` | - | ❌ Deprecated |

**Total Archived:** 15 items (directories, files, binaries)

---

## 📊 Before & After

### Before Archiving

```
coordinator/
├── cmd/                  # ❌ Old coordinator main
├── mcp-server/           # ❌ Old MCP server
├── mcp-http-bridge/      # ❌ Old HTTP bridge
├── internal/             # ❌ Old internal packages
├── ai-service/           # ❌ Old AI service
├── embed/                # ❌ Old embed
├── go.mod, go.sum        # ❌ Old Go modules
├── coordinator (binary)  # ❌ Old compiled binary
├── bin/                  # ❌ Old binaries
├── .air.toml             # ❌ Old Air config
├── Dockerfile*           # ❌ Old Docker configs
├── ui/                   # ✅ Active UI source
└── *.md                  # ✅ Documentation

PROBLEM: Confusing mix of deprecated Go code + active UI
```

### After Archiving

```
coordinator/
├── ui/                   # ✅ Active UI source (React/TypeScript)
├── *.md                  # ✅ Documentation
├── scripts/              # ✅ Utility scripts
├── test-reports/         # ✅ Test results
├── docker-compose.yml    # ✅ Docker config
└── Makefile              # ✅ Build commands

hyper/.archived/coordinator-old/
├── cmd/                  # 📦 Archived old coordinator
├── mcp-server/           # 📦 Archived old MCP server
├── mcp-http-bridge/      # 📦 Archived old HTTP bridge
├── internal/             # 📦 Archived old packages
├── ai-service/           # 📦 Archived old AI service
├── embed/                # 📦 Archived old embed
├── go.mod, go.sum        # 📦 Archived modules
├── coordinator (binary)  # 📦 Archived binary
├── bin/                  # 📦 Archived binaries
├── .air.toml             # 📦 Archived Air config
└── Dockerfile*           # 📦 Archived Docker configs

CLEAR: Only active code in main dirs, all deprecated code archived
```

---

## ✅ What Remains Active

### In `coordinator/` (Active)

| Item | Purpose | Status |
|------|---------|--------|
| `ui/` | React/TypeScript UI source | ✅ Active - Built and embedded into hyper |
| `*.md` | Documentation files | ✅ Active - Reference materials |
| `scripts/` | Utility scripts | ✅ Active - May be useful |
| `test-reports/` | Test coverage reports | ✅ Active - Recent test results |
| `docker-compose.yml` | Docker configuration | ✅ Active - May be useful |
| `Makefile` | Build commands | ✅ Active - May be useful |

### In `hyper/` (Active)

| Item | Purpose | Status |
|------|---------|--------|
| `hyper/cmd/coordinator/main.go` | Unified entry point | ✅ Active - Only binary to use |
| `hyper/internal/` | All internal packages | ✅ Active - Production code |
| `hyper/embed/` | Embedded UI bundle | ✅ Active - Production UI |
| `hyper/go.mod` | Go module definition | ✅ Active - Only module |
| `bin/hyper` | Unified binary (17MB) | ✅ Active - Only binary |

---

## 🎯 Why Archive Instead of Delete?

### Reasons to Keep Archived Code

1. **Historical Reference** - May need to reference old implementation
2. **Migration Verification** - Can compare old vs new if issues arise
3. **Code Archaeology** - Understand why certain decisions were made
4. **Recovery** - In case something was missed in migration
5. **Documentation** - Shows evolution of the codebase

### Disk Space

- **Archived Size:** ~23MB (mostly one compiled binary)
- **Impact:** Negligible on modern systems
- **Storage:** All in one directory for easy cleanup later

---

## 📝 Archived Project Details

### 1. Old Coordinator (`cmd/coordinator/`)
**Purpose:** Original coordinator service
**Replaced By:** `hyper/cmd/coordinator/main.go`
**Size:** 22MB binary + source
**Status:** Fully deprecated

**Functionality (now in unified binary):**
- Task management (human tasks, agent tasks)
- Knowledge base operations
- MongoDB integration
- HTTP server
- No MCP server (ran separately)

### 2. Old MCP Server (`mcp-server/`)
**Purpose:** Standalone MCP server
**Replaced By:** Integrated into `hyper/cmd/coordinator/main.go`
**Size:** 38 files, ~12MB binary
**Status:** Fully deprecated

**Functionality (now integrated):**
- MCP stdio transport
- MCP HTTP transport
- Tool handlers (33 tools)
- Resource handlers (12 resources)
- Prompt handlers (7 prompts)

### 3. Old HTTP Bridge (`mcp-http-bridge/`)
**Purpose:** Bridge HTTP to MCP subprocess
**Replaced By:** `hyper/internal/server/http_server.go`
**Size:** 26 files, ~12MB binary
**Status:** Fully deprecated

**Functionality (now integrated):**
- HTTP-to-MCP translation
- Concurrent request handling
- Background response routing
- No subprocess needed (direct calls)

### 4. Old Internal Packages (`internal/`)
**Purpose:** Shared internal code
**Replaced By:** `hyper/internal/`
**Size:** 8 directories
**Status:** Fully deprecated

**Contents:**
- handlers/, models/, services/, storage/
- All duplicated in hyper/internal/

### 5. Old AI Service (`ai-service/`)
**Purpose:** AI chat service
**Replaced By:** `hyper/internal/ai-service/`
**Size:** 11 files
**Status:** Fully deprecated

**Contents:**
- Claude/GPT streaming
- Tool registry
- Chat service

### 6. Old Embed (`embed/`)
**Purpose:** UI embedding
**Replaced By:** `hyper/embed/`
**Size:** 4 files
**Status:** Fully deprecated

### 7. Go Modules (`go.mod`, `go.sum`)
**Purpose:** Old coordinator Go module
**Replaced By:** `hyper/go.mod`
**Status:** Fully deprecated

**Old Module:** `hyperion-coordinator-mcp`
**New Module:** `hyper`

### 8. Compiled Binaries
**Files:**
- `coordinator` (22MB) - Old coordinator binary
- `bin/hyper` - Old binary (different from current bin/hyper)

**Replaced By:** `bin/hyper` (17MB, unified)

### 9. Build Configs
**Files:**
- `.air.toml` - Old Air hot-reload config
- `Dockerfile` - Old coordinator Dockerfile
- `Dockerfile.dev` - Old dev Dockerfile

**Replaced By:**
- Root `.air.toml` (builds hyper/)
- Root `Dockerfile` (builds hyper/)

---

## 🔍 Verification

### Confirm Active Binary

```bash
# Check active binary
ls -lh bin/hyper
# Should show: 17MB bin/hyper

# Check source
ls hyper/cmd/coordinator/main.go
# Should exist

# Check no old binaries
ls coordinator/coordinator 2>/dev/null
# Should show: No such file or directory

# Check archived
ls hyper/.archived/coordinator-old/coordinator
# Should show: 22MB archived binary
```

### Confirm No Old Go Modules

```bash
# Check active module
cat hyper/go.mod | head -1
# Should show: module hyper

# Check no old module
cat coordinator/go.mod 2>/dev/null
# Should show: No such file or directory

# Check archived module
cat hyper/.archived/coordinator-old/go.mod | head -1
# Should show: module hyperion-coordinator-mcp
```

### Confirm UI Still Active

```bash
# Check active UI source
ls coordinator/ui/src/
# Should show React components

# Check UI is built and embedded
ls hyper/embed/ui/
# Should show compiled UI assets
```

---

## 📖 Migration History

### Timeline

1. **Phase 1:** Created unified hyper binary (hyper/cmd/coordinator/)
2. **Phase 2:** Integrated MCP server into unified binary
3. **Phase 3:** Integrated HTTP bridge into HTTP server
4. **Phase 4:** Updated all make targets to use unified binary
5. **Phase 5:** Added all 9 missing MCP tools (filesystem, discovery, subagent)
6. **Phase 6:** Archived all deprecated Go code ✅ **(Current)**

### What Changed

| Aspect | Before | After |
|--------|--------|-------|
| **Entry Points** | 3 separate main.go files | 1 unified main.go |
| **Binaries** | 3 binaries (~48MB total) | 1 binary (17MB) |
| **Go Modules** | 3 go.mod files | 1 go.mod |
| **Processes** | 3 processes | 1 process |
| **Communication** | Inter-process (stdio/HTTP) | In-memory function calls |
| **Build** | Build 3 projects | Build 1 project |
| **Deploy** | Deploy 3 services | Deploy 1 service |

---

## 🚀 Usage

### ✅ CORRECT: Use Unified Binary

```bash
# Build
make build

# Run in different modes
./bin/hyper --mode=http    # REST API + UI + MCP HTTP
./bin/hyper --mode=mcp     # MCP stdio for Claude Code
./bin/hyper --mode=both    # Dual mode

# Development
make dev                   # Hot reload (Go)
make dev-hot              # Hot reload (Go + UI)
```

### ❌ WRONG: Don't Use Archived Code

```bash
# ❌ DON'T DO THIS - Uses old deprecated binary
cd coordinator && go build ./cmd/coordinator

# ❌ DON'T DO THIS - Old MCP server
cd coordinator/mcp-server && go build

# ❌ DON'T DO THIS - Old HTTP bridge
cd coordinator/mcp-http-bridge && go build
```

---

## 🗑️ Future Cleanup (Optional)

When you're confident the unified binary is stable (after 30-90 days):

```bash
# Option 1: Delete archived code
rm -rf hyper/.archived/coordinator-old/

# Option 2: Compress and archive externally
tar -czf coordinator-old-backup-2025-10-12.tar.gz hyper/.archived/coordinator-old/
mv coordinator-old-backup-2025-10-12.tar.gz ~/backups/
rm -rf hyper/.archived/coordinator-old/

# Option 3: Keep indefinitely (only ~23MB)
# Leave in hyper/.archived/ for historical reference
```

---

## 📊 Archive Statistics

| Metric | Value |
|--------|-------|
| **Projects Archived** | 3 (coordinator, mcp-server, mcp-http-bridge) |
| **Directories Archived** | 8 |
| **Files Archived** | ~100+ files |
| **Binaries Archived** | 3 binaries |
| **Total Size** | ~23MB |
| **Archive Location** | `hyper/.archived/coordinator-old/` |
| **Remaining Active UI** | `coordinator/ui/` |
| **Active Binary** | `bin/hyper` (17MB) |

---

## ✅ Verification Checklist

After archiving, verify:

- [ ] `bin/hyper` exists and is 17MB
- [ ] `hyper/cmd/coordinator/main.go` exists
- [ ] `hyper/go.mod` is the only active module
- [ ] `coordinator/ui/` still exists (active UI source)
- [ ] `coordinator/cmd/` does NOT exist (archived)
- [ ] `coordinator/mcp-server/` does NOT exist (archived)
- [ ] `coordinator/mcp-http-bridge/` does NOT exist (archived)
- [ ] `coordinator/internal/` does NOT exist (archived)
- [ ] `coordinator/go.mod` does NOT exist (archived)
- [ ] `hyper/.archived/coordinator-old/` contains all archived code
- [ ] `make build` still works
- [ ] `make dev` still works
- [ ] UI at `coordinator/ui/` can still be built

---

## 🎓 Lessons Learned

1. **Archive, Don't Delete** - Keeps history without cluttering workspace
2. **Single Source of Truth** - One binary, one codebase
3. **Clear Separation** - Active code vs archived code
4. **Disk Space Minimal** - ~23MB for complete history
5. **Migration Safety** - Can reference old code if needed

---

**Archive Date:** 2025-10-12
**Archived By:** Claude Code AI Assistant
**Status:** ✅ Complete
**Location:** `hyper/.archived/coordinator-old/`
**Active Binary:** `bin/hyper` (17MB)
**Active UI:** `coordinator/ui/` (React/TypeScript)
