# Hyperion Native Binary

A single self-contained binary with embedded UI - no Docker, no file mappings, no external dependencies.

## Quick Start

### 1. Build the Native Binary

```bash
make native
```

This will:
- Build the React UI and embed it into the Go binary
- Create a single executable at `bin/hyper`
- Display installation instructions

### 2. Configure Environment

Copy `.env.native` and update with your settings:

```bash
cp .env.native .env.native
# Edit .env.native with your MongoDB URI, API keys, etc.
```

**Important:** The binary uses `godotenv.Overload()` which means `.env.native` values **override** system environment variables. This allows the native binary to have its own configuration without affecting your system.

### 3. Run the Binary

```bash
make run
```

Or run directly:

```bash
./bin/hyper --mode=http
```

The binary will:
1. Look for `.env.native` in the same directory as the binary
2. Fall back to `.env.native` in the current working directory
3. Load and override environment variables from the file
4. Start the HTTP server with embedded UI

### 4. Access the UI

Open your browser to: http://localhost:7095/ui

## MCP Stdio Mode (Claude Code Integration)

### Configure for Claude Code

The native binary can run in stdio mode for Claude Code integration:

```bash
make configure-native
```

This will:
1. Build the native binary (if not already built)
2. Configure Claude Code to use the binary in MCP stdio mode
3. Set it to user scope (available globally in all projects)

Or configure manually:

```bash
claude mcp add hyper "$(pwd)/bin/hyper" --args "--mode=mcp" --scope user
```

### Run in Stdio Mode

To test the stdio mode manually:

```bash
make run-stdio
```

Or run directly:

```bash
./bin/hyper --mode=mcp
```

### Verify Configuration

```bash
claude mcp list | grep hyper
```

You should see:
```
hyper    /path/to/your/bin/hyper --mode=mcp    user
```

### Configuration File Location for Stdio Mode

When running in stdio mode (MCP), the binary looks for `.env.native` in:

1. **Same directory as the binary** (recommended for system install)
   ```bash
   # If you installed to /usr/local/bin/
   sudo cp .env.native /usr/local/bin/.env.native
   ```

2. **Current working directory** (when running from project)
   ```bash
   # Place .env.native in your project root
   cp .env.native /path/to/your/project/.env.native
   ```

**Note:** The binary will try both locations and use the first one it finds.

## Configuration

The `.env.native` file supports:

### MongoDB
```bash
MONGODB_URI=your-mongodb-uri
MONGODB_DATABASE=coordinator_db
```

### Qdrant Vector Database
```bash
QDRANT_URL=http://localhost:6333
QDRANT_API_KEY=
```

### Embedding Options

**Voyage AI (Recommended for Production)**
```bash
EMBEDDING=voyage
VOYAGE_API_KEY=your-key
VOYAGE_MODEL=voyage-3
```

**Ollama (Local GPU-accelerated)**
```bash
EMBEDDING=ollama
OLLAMA_URL=http://localhost:11434
OLLAMA_MODEL=nomic-embed-text
```

**Embedded Llama.cpp (No external service)**
```bash
EMBEDDING=llama
LLAMA_MODEL_PATH=models/nomic-embed-text-v1.5.Q4_K_M.gguf
```

**OpenAI**
```bash
EMBEDDING=openai
OPENAI_API_KEY=your-key
```

### HTTP Server
```bash
HTTP_PORT=7095
```

### Code Indexing
```bash
CODE_INDEX_FOLDERS=/path/to/code
CODE_INDEX_AUTO_SCAN=true
```

## Installation

Install to system path for global access:

**macOS:**
```bash
sudo mv bin/hyper /usr/local/bin/
```

**Linux:**
```bash
sudo mv bin/hyper /usr/bin/
```

Then run from anywhere:
```bash
hyper --mode=http
```

## Development Mode

For development with hot-reload:

```bash
make run-dev
```

This uses Air to automatically rebuild on file changes.

## Build Options

### Cross-Platform Builds

Build for different platforms:

```bash
# macOS Apple Silicon
./build-native.sh --platform darwin-arm64

# macOS Intel
./build-native.sh --platform darwin-amd64

# Windows
./build-native.sh --platform windows-amd64

# Linux
./build-native.sh --platform linux-amd64
```

### Modes

The binary supports three modes:

- `--mode=http` - HTTP server only (REST API + UI)
- `--mode=mcp` - MCP server only (stdio protocol)
- `--mode=both` - Both HTTP and MCP (default)

## Architecture

- **Single Binary**: Go + embedded React UI
- **GPU Acceleration**: Metal (macOS), CUDA (Windows/Linux NVIDIA), Vulkan (AMD/Intel)
- **No Dependencies**: Everything needed is in the binary
- **Configuration**: `.env.native` file (overrides system env vars)

## Troubleshooting

### "MONGODB_URI environment variable is required"

Create `.env.native` file with your MongoDB URI:
```bash
cp .env.native .env.native
# Edit and add your MONGODB_URI
```

### "Binary not found"

Build the binary first:
```bash
make native
```

### Configuration Not Loading

The binary looks for `.env.native` in:
1. Same directory as the binary
2. Current working directory

Make sure the file exists in one of these locations.

## Environment Variable Priority

1. `.env.native` file (highest priority - uses `godotenv.Overload()`)
2. System environment variables (lowest priority)

This means `.env.native` will **override** any system environment variables with the same name.

## Quick Reference

### Build & Run Commands

```bash
# Build native binary
make native

# Run HTTP server (with embedded UI)
make run

# Run in stdio mode (for MCP/Claude Code)
make run-stdio

# Run with hot-reload (development)
make run-dev

# Configure Claude Code
make configure-native
```

### Configuration

```bash
# Create config file
cp .env.native .env.native

# Edit configuration
vim .env.native

# Install binary globally (macOS)
sudo mv bin/hyper /usr/local/bin/

# Install config globally (macOS)
sudo cp .env.native /usr/local/bin/.env.native
```

### Modes

- `--mode=http` - HTTP server + embedded UI (port 7095)
- `--mode=mcp` - MCP stdio protocol (for Claude Code)
- `--mode=both` - Both HTTP and MCP (default)

### Usage Examples

```bash
# HTTP mode with custom port
HTTP_PORT=8080 ./bin/hyper --mode=http

# Stdio mode (reads from stdin, writes to stdout)
./bin/hyper --mode=mcp

# Both modes (HTTP + MCP)
./bin/hyper --mode=both
```
