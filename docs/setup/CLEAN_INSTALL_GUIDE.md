# Clean Installation Guide

## Overview

This guide explains how to perform a clean installation of Hyperion from scratch.

## Why Clean Install?

A clean install is needed when:
- Starting fresh on a new machine
- Build artifacts are corrupted
- Dependencies are out of sync
- Troubleshooting build issues
- Switching between build modes

---

## Quick Start (Recommended)

### One-Command Install

```bash
./clean-install.sh
```

**What it does:**
1. ✅ Cleans all build artifacts
2. ✅ Optionally removes node_modules
3. ✅ Optionally cleans Go cache
4. ✅ Installs Go dependencies
5. ✅ Installs Node.js dependencies
6. ✅ Builds hyper binary (without UI embedding)
7. ✅ Verifies installation

**Time:** 2-5 minutes (depending on internet speed and options chosen)

---

## Interactive Prompts

The script will ask you:

### 1. Confirmation to Start
```
This script will:
  1. Clean all build artifacts
  2. Remove node_modules
  3. Clean Go module cache (optional)
  4. Install dependencies
  5. Build the hyper binary

Continue? (yes/no):
```

### 2. Remove node_modules?
```
Remove node_modules? (yes/no):
```
- **yes**: Removes ui/node_modules and ui2/node_modules (~200MB)
- **no**: Keeps node_modules (faster if they're already correct)

### 3. Clean Go Cache?
```
Clean Go module cache? (yes/no):
```
- **yes**: Removes Go module cache (~500MB-1GB) - forces re-download
- **no**: Keeps Go cache (faster, recommended unless troubleshooting)

---

## Build Method

The clean-install.sh script builds the Go binary **without UI embedding** using:

```bash
cd hyper && go build -tags dev -o ../bin/hyper ./cmd/coordinator
```

### Why `-tags dev`?

The `-tags dev` flag tells Go to use `embed/ui_dev.go` instead of `embed/ui.go`:

**With `-tags dev` (Development Mode):**
- Uses `embed/ui_dev.go`
- No UI files embedded in binary
- Binary size: ~38MB
- Build time: 10-30 seconds
- UI served from Vite dev server or filesystem

**Without `-tags dev` (Production Mode):**
- Uses `embed/ui.go`
- Requires `ui2/dist` to be built first
- Embeds UI into binary
- Binary size: ~42MB
- Build time: 2-5 minutes (includes UI build)

---

## Manual Installation (Alternative)

If you prefer manual control:

### Step 1: Clean Build Artifacts

```bash
# Remove bins and build outputs
rm -rf bin/ hyper/bin/
rm -rf ui/dist ui2/dist
rm -rf hyper/embed/ui hyper/embed/ui2
```

### Step 2: Clean Dependencies (Optional)

```bash
# Remove node_modules
rm -rf ui/node_modules ui2/node_modules

# Clean Go cache
cd hyper && go clean -modcache && cd ..
```

### Step 3: Install Dependencies

```bash
# Install Go dependencies
cd hyper && go mod download && cd ..

# Install Node dependencies
cd ui && npm install && cd ..
```

### Step 4: Build Binary

```bash
# Build Go binary (without UI embedding)
cd hyper && go build -tags dev -o ../bin/hyper ./cmd/coordinator && cd ..
```

### Step 5: Verify

```bash
# Check binary
ls -lh bin/hyper

# Test binary
./bin/hyper --help
```

---

## Using Makefile Commands

You can also use Makefile targets for cleaning:

### Option 1: Clean (Keeps node_modules)
```bash
make clean
```

**Removes:**
- bin/hyper
- bin/hyper2
- hyper/bin/
- ui/dist
- ui2/dist
- hyper/embed/ui
- hyper/embed/ui2

**Keeps:**
- ui/node_modules
- ui2/node_modules
- Go module cache

### Option 2: Clean All (Removes Everything)
```bash
make clean-all
```

**⚠️ Warning:** This removes node_modules and Go cache. Requires confirmation.

**Removes:**
- Everything from `make clean`
- ui/node_modules
- ui2/node_modules
- Go module cache (~500MB-1GB)

### Option 3: Clean Install Script
```bash
make clean-install
```

**Runs:** `./clean-install.sh` - interactive clean installation

---

## After Installation

### Initialize a New Project

```bash
# Default (Ollama)
./bin/hyper init

# With OpenAI
./bin/hyper init -provider openai -token sk-proj-your-key

# With Anthropic
./bin/hyper init -provider anthropic -token sk-ant-your-key
```

### Start Services

```bash
# Start Docker services
docker compose up -d

# Wait for Ollama model download
docker compose logs -f ollama-pull
```

### Run Hyper

```bash
# Run in HTTP mode
./bin/hyper --mode=http

# Access UI
open http://localhost:7095
```

---

## Troubleshooting

### Issue: Permission Denied

**Error:**
```bash
./clean-install.sh: Permission denied
```

**Fix:**
```bash
chmod +x clean-install.sh
./clean-install.sh
```

---

### Issue: Build Fails with Embed Error

**Error:**
```
embed/ui.go:16:12: pattern all:ui2/dist: no matching files found
```

**Cause:** Building without `-tags dev` flag

**Fix:** The clean-install.sh script now includes `-tags dev` automatically. If building manually:
```bash
cd hyper && go build -tags dev -o ../bin/hyper ./cmd/coordinator
```

---

### Issue: Go Version Too Old

**Error:**
```
go: go.mod requires go >= 1.21
```

**Fix:**
```bash
# Download from https://golang.org/dl/
# Or use gvm:
gvm install go1.21
gvm use go1.21
```

---

### Issue: Node Version Too Old

**Error:**
```
npm ERR! node version not supported
```

**Fix:**
```bash
# Download from https://nodejs.org/
# Or use nvm:
nvm install 18
nvm use 18
```

---

### Issue: Dependencies Fail to Install

**Error:**
```
npm ERR! network timeout
```

**Fix:**
```bash
# Clear npm cache
npm cache clean --force

# Try again
cd ui && npm install
```

---

## Clean Install vs Native Build

| Method | UI Embedded | Build Time | Binary Size | Use Case |
|--------|------------|------------|-------------|----------|
| **Clean Install** (`clean-install.sh`) | ❌ No | 2-5 min | 38MB | Development, testing |
| **Native Build** (`make native`) | ✅ Yes | 5-10 min | 42MB | Distribution, production |

### When to Use Clean Install
- Development work
- Testing changes
- Quick setup
- Troubleshooting builds
- Clean state needed

### When to Use Native Build
- Production deployment
- Distribution to users
- Single-binary requirement
- Embedded UI needed

---

## Files Created/Modified

After clean installation, you'll have:

```
hyper/
├── bin/
│   └── hyper              # Built binary (38MB)
├── hyper/
│   └── (Go source files, no changes)
├── ui/
│   └── node_modules/      # Node dependencies (if installed)
└── (documentation files)
```

**No embedded UI files** are created in `hyper/embed/` because `-tags dev` skips embedding.

---

## Performance

### Clean Install Benchmark

| Step | Time | Notes |
|------|------|-------|
| **1. Clean artifacts** | <1s | Fast (just file deletion) |
| **2. Remove node_modules** | 1-5s | Optional, depends on size |
| **3. Clean Go cache** | 5-10s | Optional, depends on cache size |
| **4. Install Go deps** | 10-30s | Depends on internet speed |
| **5. Install Node deps** | 30-120s | Depends on internet speed |
| **6. Build binary** | 10-30s | Fast with cached deps |
| **Total** | **2-5 min** | Most time is npm install |

---

## Disk Space

### Before Clean Install
- node_modules: ~200MB
- Go cache: ~500MB-1GB
- Build artifacts: ~50MB
- **Total:** ~750MB-1.25GB

### After Clean Install
- node_modules: ~200MB (if reinstalled)
- Go cache: ~500MB-1GB (if kept)
- Build artifacts: ~40MB (just binary)
- **Total:** ~740MB-1.24GB (similar)

**Note:** Clean install doesn't save disk space - it ensures a fresh, working state.

---

## Next Steps

After clean installation:

1. **Read Documentation**
   - INSTALLATION.md - Full installation guide
   - QUICK_REFERENCE.md - Quick commands
   - HYPER_INIT_WITH_PROVIDER.md - Provider setup

2. **Initialize Project**
   ```bash
   ./bin/hyper init
   ```

3. **Start Services**
   ```bash
   docker compose up -d
   ```

4. **Run Hyper**
   ```bash
   ./bin/hyper --mode=http
   ```

5. **Access UI**
   ```
   http://localhost:7095
   ```

---

## FAQ

### Q: Should I remove node_modules?

**A:** Only if you're having npm issues or want a completely fresh install. Otherwise, keeping them saves time.

### Q: Should I clean Go cache?

**A:** Usually no. Only clean if you're troubleshooting dependency issues.

### Q: Can I use the clean install for production?

**A:** The clean install creates a development binary. For production, use `make native` to embed the UI.

### Q: How often should I clean install?

**A:** Only when needed:
- After pulling major changes
- When build artifacts are corrupted
- When dependencies are out of sync
- When troubleshooting issues

### Q: What's the difference between clean-install.sh and make clean-install?

**A:** They're the same - `make clean-install` just calls `./clean-install.sh`.

---

## Summary

**Clean installation is:**
- ✅ Fast (2-5 minutes)
- ✅ Interactive (prompts for options)
- ✅ Safe (doesn't remove uncommitted code)
- ✅ Thorough (ensures fresh dependencies)
- ✅ Verified (tests binary after build)

**Use it when:**
- Starting fresh
- Troubleshooting builds
- Dependencies are broken
- Need clean state

**Avoid when:**
- Just small code changes
- Dependencies are fine
- Time is critical

---

**Version:** 1.0.0
**Last Updated:** 2025-11-06
**Script:** clean-install.sh
**Build Method:** Go binary with `-tags dev` (no UI embedding)
