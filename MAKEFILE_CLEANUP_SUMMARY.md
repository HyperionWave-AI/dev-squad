# Makefile Cleanup Summary

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

## Conclusion

✅ **Makefile cleanup complete**
✅ **Redundant binaries archived**
✅ **Build system unified**
✅ **All targets tested and working**

The build system now focuses on the **single unified hyper binary** approach. All redundant targets removed, all working targets preserved.
