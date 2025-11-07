# Hyperion Installation Guide

## Overview

This guide covers clean installation of Hyperion from scratch.

---

## Prerequisites

### Required

- **Go 1.21+** - [Download](https://golang.org/dl/)
- **Node.js 18+** - [Download](https://nodejs.org/)
- **Git** - For cloning the repository

### Optional

- **Docker** - For running dependencies (MongoDB, Qdrant, Ollama)
- **Make** - For using Makefile commands (usually pre-installed)

### Verify Prerequisites

```bash
# Check Go version
go version  # Should be 1.21 or higher

# Check Node version
node --version  # Should be 18 or higher

# Check npm
npm --version

# Check Docker (optional)
docker --version
docker compose version
```

---

## Installation Methods

### Method 1: Clean Install Script (Recommended)

**Fastest and easiest way to get started.**

```bash
# 1. Clone repository
git clone <repository-url>
cd hyper

# 2. Run clean install script
./clean-install.sh
```

**What it does:**
- ✅ Cleans all build artifacts
- ✅ Optionally removes node_modules
- ✅ Optionally cleans Go cache
- ✅ Installs all dependencies
- ✅ Builds hyper binary
- ✅ Verifies installation

**Time:** 2-5 minutes (depending on internet speed)

---

### Method 2: Makefile Commands

**For developers who prefer manual control.**

#### Quick Install (Recommended)
```bash
# Install dependencies and build
make install
cd hyper && go build -o ../bin/hyper ./cmd/coordinator
```

#### Clean Install
```bash
# Clean everything
make clean

# Install dependencies
make install

# Build binary (Go only, no UI embedding)
cd hyper && go build -tags dev -o ../bin/hyper ./cmd/coordinator
```

#### Deep Clean Install
```bash
# Clean everything including node_modules
make clean-all

# Install dependencies
make install

# Build binary (Go only, no UI embedding)
cd hyper && go build -tags dev -o ../bin/hyper ./cmd/coordinator
```

---

### Method 3: Manual Installation

**For complete control over the process.**

```bash
# 1. Clone repository
git clone <repository-url>
cd hyper

# 2. Clean (optional)
rm -rf bin/ hyper/bin/ ui/dist ui2/dist hyper/embed/ui hyper/embed/ui2

# 3. Install Go dependencies
cd hyper
go mod download
cd ..

# 4. Install Node dependencies
cd ui
npm install
cd ..

# 5. Build Go binary (without UI embedding)
cd hyper
go build -tags dev -o ../bin/hyper ./cmd/coordinator
cd ..

# 6. Verify
./bin/hyper init --help
```

---

## Build Options

### Option 1: Go Binary Only (Fast, Recommended)

**Best for:** Quick setup, command-line usage, clean installations

```bash
cd hyper
go build -tags dev -o ../bin/hyper ./cmd/coordinator
```

**Result:**
- Binary size: ~38MB
- Build time: 10-30 seconds
- No UI embedded (hyper proxies to Vite dev server or serves from filesystem)
- Uses `-tags dev` to skip UI embedding

---

### Option 2: Full Native Build (Slow, UI Included)

**Best for:** Distribution, embedded UI

```bash
make native
```

**Result:**
- Binary size: ~42MB
- Build time: 2-5 minutes
- UI embedded in binary

**Note:** Currently fails due to pre-existing UI TypeScript errors. Use Option 1 instead.

---

## Post-Installation

### Verify Installation

```bash
# Check binary exists
ls -lh bin/hyper

# Test binary
./bin/hyper init --help
```

### Add to PATH (Optional)

**Linux/macOS:**
```bash
# Add to current session
export PATH="$(pwd)/bin:$PATH"

# Add permanently to ~/.bashrc or ~/.zshrc
echo 'export PATH="/path/to/hyper/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc

# Or create symlink
sudo ln -s $(pwd)/bin/hyper /usr/local/bin/hyper
```

**Windows (PowerShell):**
```powershell
# Add to current session
$env:PATH = "$(pwd)\bin;$env:PATH"

# Add permanently (run as Administrator)
[Environment]::SetEnvironmentVariable("PATH", "$env:PATH;C:\path\to\hyper\bin", "Machine")
```

---

## First Project Setup

### Quick Start

```bash
# 1. Create project directory
mkdir my-project
cd my-project

# 2. Initialize Hyperion
hyper init

# 3. Start services
docker compose up -d

# 4. Wait for Ollama model download
docker compose logs -f ollama-pull

# 5. Run Hyper
hyper --mode=http

# 6. Access UI
open http://localhost:7095
```

---

### With Provider Configuration

```bash
# Initialize with OpenAI
hyper init -provider openai -model gpt-4 -token sk-proj-your-key

# Or with Anthropic
hyper init -provider anthropic -model claude-sonnet-4 -token sk-ant-your-key

# Or with Voyage AI
hyper init -provider voyage -model voyage-3 -token pa-your-key
```

---

## Troubleshooting

### Issue: Go version too old

**Error:**
```
go: go.mod requires go >= 1.21
```

**Solution:**
```bash
# Update Go
# Download from https://golang.org/dl/
# Or use version manager like gvm
```

---

### Issue: Node version too old

**Error:**
```
npm ERR! node version not supported
```

**Solution:**
```bash
# Update Node.js
# Download from https://nodejs.org/
# Or use nvm:
nvm install 18
nvm use 18
```

---

### Issue: Permission denied

**Error:**
```bash
./clean-install.sh: Permission denied
```

**Solution:**
```bash
chmod +x clean-install.sh
./clean-install.sh
```

---

### Issue: Binary not found

**Error:**
```bash
bin/hyper: No such file or directory
```

**Solution:**
```bash
# Build the binary (with dev tags to skip UI embedding)
cd hyper
go build -tags dev -o ../bin/hyper ./cmd/coordinator
cd ..

# Verify
ls -lh bin/hyper
```

---

### Issue: Make command not found

**Error:**
```bash
make: command not found
```

**Solution:**

**macOS:**
```bash
xcode-select --install
```

**Ubuntu/Debian:**
```bash
sudo apt-get install build-essential
```

**Windows:**
```bash
# Use Git Bash or WSL
# Or install via chocolatey:
choco install make
```

---

### Issue: Docker not running

**Error:**
```bash
Cannot connect to Docker daemon
```

**Solution:**
```bash
# Start Docker Desktop (macOS/Windows)
# Or start Docker daemon (Linux):
sudo systemctl start docker
```

---

## Uninstallation

### Remove Hyperion

```bash
# From project directory
cd /path/to/hyper

# Remove binary
rm -rf bin/

# Remove dependencies (optional)
rm -rf ui/node_modules ui2/node_modules

# Remove entire project
cd ..
rm -rf hyper/
```

### Remove Docker Containers

```bash
# Stop and remove containers
docker compose down -v

# Remove images
docker rmi hyperion-coordinator hyperion-ollama hyperion-qdrant
```

---

## Development Setup

### For Contributors

```bash
# 1. Fork and clone
git clone <your-fork-url>
cd hyper

# 2. Install dependencies
make install

# 3. Install Air (hot reload)
make install-air

# 4. Start development mode
make dev-hot

# 5. Make changes and test
cd hyper
go test ./...
```

---

## Build Targets Summary

| Command | What it does | Time | Size |
|---------|-------------|------|------|
| `./clean-install.sh` | Clean install (interactive) | 2-5 min | 38MB |
| `make install` | Install dependencies only | 1-2 min | - |
| `make clean` | Remove build artifacts | <1 sec | - |
| `make clean-all` | Remove everything | <1 sec | - |
| `make native` | Build with embedded UI | 2-5 min | 42MB |
| `cd hyper && go build -tags dev ...` | Build Go binary (no UI) | 10-30 sec | 38MB |

---

## Next Steps

After installation:

1. **Read Quick Reference**
   ```bash
   cat QUICK_REFERENCE.md
   ```

2. **Initialize Project**
   ```bash
   hyper init
   ```

3. **Configure Provider** (optional)
   ```bash
   hyper init -provider openai -token sk-...
   ```

4. **Start Services**
   ```bash
   docker compose up -d
   ```

5. **Run Hyper**
   ```bash
   hyper --mode=http
   ```

6. **Access UI**
   ```
   http://localhost:7095
   ```

---

## Documentation

- **QUICK_REFERENCE.md** - Quick command reference
- **HYPER_INIT_GUIDE.md** - Project initialization guide
- **HYPER_INIT_WITH_PROVIDER.md** - Provider configuration guide
- **MAKEFILE_AND_DOCKER_GUIDE.md** - Complete Makefile and Docker guide
- **TEST_SUMMARY.md** - Build and test status

---

## Support

### Common Issues

- UI TypeScript errors → Use Go binary only build
- Port conflicts → Use `hyper init` with custom ports
- Provider validation fails → Check API key and network

### Getting Help

1. Check documentation in project root
2. Review troubleshooting section above
3. Check issue tracker (if available)
4. Review logs: `hyper --mode=http` output

---

**Version:** 1.0.0
**Last Updated:** 2025-01-06
**Installation Time:** 2-5 minutes
**Minimum Requirements:** Go 1.21+, Node 18+
