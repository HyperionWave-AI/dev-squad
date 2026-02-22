# Hyperion - Makefile Commands & Docker Compose Guide

## Table of Contents
1. [Makefile Commands Explained](#makefile-commands-explained)
2. [Docker Compose Setup](#docker-compose-setup)
3. [Embedding Options](#embedding-options)
4. [Configuration Guide](#configuration-guide)

---

## Makefile Commands Explained

### Help & Information
```bash
make help
```
Shows all available make targets with descriptions.

---

### Build Commands

#### `make native` (or `make build`)
**What it does:**
- Builds the unified `hyper` binary with embedded UI
- Creates a single self-contained executable (~16MB)
- Embeds the React UI inside the Go binary (no external files needed)
- Output: `bin/hyper`

**Usage:**
```bash
make native
```

**When to use:** Building for production or when you want a single binary to distribute.

---

#### `make native2`
**What it does:**
- Same as `native`, but uses UI2 (alternative UI)
- Output: `bin/hyper2`

**Usage:**
```bash
make native2
```

---

### Installation Commands

#### `make install`
**What it does:**
- Installs Go dependencies (`go mod download`)
- Installs Node.js dependencies for UI (`npm install`)

**Usage:**
```bash
make install
```

**When to use:** First time setup or after pulling changes that update dependencies.

---

#### `make install2`
**What it does:**
- Same as `install`, but for UI2 dependencies

---

#### `make install-air`
**What it does:**
- Installs Air (hot-reload tool for Go)
- Allows automatic rebuilds when code changes

**Usage:**
```bash
make install-air
```

**When to use:** Setting up development environment for the first time.

---

### Development Commands

#### `make dev`
**What it does:**
- Starts Go backend with hot-reload (Air)
- Watches for Go file changes and rebuilds automatically
- Does **NOT** compile UI (uses pre-built UI from `embed/ui`)
- Requires `.air.toml` and `.env.hyper` files

**Usage:**
```bash
make dev
```

**When to use:** Backend-only development (UI is already built).

---

#### `make dev-hot`
**What it does:**
- Starts full-stack development with hot-reload
- Launches Vite dev server for UI (port 5173)
- Launches Go backend with Air (hot-reload)
- Both UI and backend reload on changes

**Usage:**
```bash
make dev-hot
```

**When to use:** Full-stack development (UI + backend changes).

---

#### `make run-dev`
**What it does:**
- Runs with Air hot-reload
- Similar to `make dev` but different script

**Usage:**
```bash
make run-dev
```

---

### Runtime Commands

#### `make run`
**What it does:**
- Runs the pre-built native binary in HTTP mode
- Requires `bin/hyper` to exist (run `make native` first)
- Uses `.env.native` for configuration
- Starts REST API + Web UI on port 7095

**Usage:**
```bash
make native  # Build first
make run     # Then run
```

**When to use:** Testing the production binary locally.

---

#### `make run-stdio`
**What it does:**
- Runs binary in **stdio mode** (MCP protocol)
- Used for Claude Code integration
- No HTTP server (only stdio communication)

**Usage:**
```bash
make run-stdio
```

**When to use:** Testing MCP integration with Claude Code.

---

#### `make run-mcp-http`
**What it does:**
- Runs unified binary in HTTP mode
- REST API: `http://localhost:7095/api/v1`
- Web UI: `http://localhost:7095`
- Health check: `http://localhost:7095/api/v1/health`

**Usage:**
```bash
make run-mcp-http
```

---

### Claude Code Integration

#### `make configure-native`
**What it does:**
- Configures Claude Code to use the native `hyper` binary
- Removes old configurations
- Adds new configuration with `--mode=mcp` (stdio mode)
- Uses `claude mcp add` command

**Usage:**
```bash
make native              # Build first
make configure-native    # Configure Claude Code
```

**When to use:** Setting up or updating Claude Code integration.

---

### Desktop App Commands

#### `make desktop` (or `make desktop-dev`)
**What it does:**
- Builds and runs desktop app in development mode
- Starts `hyper` server in background
- Launches Tauri desktop app

**Usage:**
```bash
make desktop
```

---

#### `make desktop-build`
**What it does:**
- Builds desktop app for distribution
- Creates platform-specific installers:
  - **macOS**: `.dmg` and `.app` bundle
  - **Windows**: `.msi` installer
  - **Linux**: `.AppImage`

**Usage:**
```bash
make desktop-build
```

**Output locations:**
- macOS: `desktop-app/src-tauri/target/release/bundle/dmg/`
- Windows: `desktop-app/src-tauri/target/release/bundle/msi/`
- Linux: `desktop-app/src-tauri/target/release/bundle/appimage/`

---

### Utility Commands

#### `make test`
**What it does:**
- Runs all Go tests
- Executes `go test ./...` in the `hyper/` directory

**Usage:**
```bash
make test
```

---

#### `make clean`
**What it does:**
- Removes build artifacts:
  - `bin/hyper` and `bin/hyper2`
  - `hyper/bin/`
  - `ui/dist` and `ui2/dist`
  - `hyper/embed/ui` and `hyper/embed/ui2`
- **Preserves** `node_modules` (doesn't delete)

**Usage:**
```bash
make clean
```

**When to use:** Starting fresh, fixing build issues, or before committing.

---

## Docker Compose Setup

### Complete Docker Compose File for All Dependencies

Below is a **complete** Docker Compose file that includes:
- MongoDB (with authentication)
- Qdrant (vector database)
- Embedding services (TEI, Ollama)
- Multiple profiles for different use cases

Create a file `docker-compose.full.yml`:

```yaml
version: '3.8'

# Complete Hyperion Development Stack
# Supports multiple embedding options:
#   - TEI (Text Embeddings Inference) - CPU-based, free, local
#   - Ollama - GPU-accelerated, free, local
#   - Voyage AI - Cloud-based (requires API key)
#   - OpenAI - Cloud-based (requires API key)
#
# Profiles:
#   - default: MongoDB + Qdrant + TEI
#   - ollama: MongoDB + Qdrant + Ollama
#   - cloud: MongoDB + Qdrant (use cloud embedding providers)

services:
  # ==========================================
  # MongoDB - Primary Database
  # ==========================================
  mongodb:
    image: mongo:7.0
    container_name: hyperion-mongodb
    ports:
      - "${MONGODB_PORT:-27017}:27017"
    environment:
      # Root credentials for initial setup
      MONGO_INITDB_ROOT_USERNAME: ${MONGO_ROOT_USERNAME:-admin}
      MONGO_INITDB_ROOT_PASSWORD: ${MONGO_ROOT_PASSWORD:-admin123}
      MONGO_INITDB_DATABASE: ${MONGODB_DATABASE:-coordinator_db}
    volumes:
      - mongodb_data:/data/db
      - mongodb_config:/data/configdb
    networks:
      - hyperion-network
    healthcheck:
      test: ["CMD", "mongosh", "--eval", "db.adminCommand('ping')"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s
    restart: unless-stopped

  # ==========================================
  # Qdrant - Vector Database
  # ==========================================
  qdrant:
    image: qdrant/qdrant:latest
    container_name: hyperion-qdrant
    ports:
      - "${QDRANT_HTTP_PORT:-6333}:6333"  # HTTP API
      - "${QDRANT_GRPC_PORT:-6334}:6334"  # gRPC API
    volumes:
      - qdrant_data:/qdrant/storage
    networks:
      - hyperion-network
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "-O", "/dev/null", "http://localhost:6333/health"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s
    restart: unless-stopped

  # ==========================================
  # TEI (Text Embeddings Inference) - Default
  # Hugging Face embedding service (CPU-based)
  # ==========================================
  embedding-tei:
    image: ghcr.io/huggingface/text-embeddings-inference:cpu-latest
    container_name: hyperion-embedding-tei
    platform: linux/amd64  # Required for Apple Silicon
    profiles:
      - default
      - tei
    command: --model-id nomic-ai/nomic-embed-text-v1.5 --port 8080
    ports:
      - "${TEI_PORT:-8080}:8080"
    environment:
      HF_HUB_CACHE: /data
      # Optional: Set number of threads for CPU optimization
      OMP_NUM_THREADS: ${OMP_NUM_THREADS:-4}
    volumes:
      - tei_cache:/data
    networks:
      - hyperion-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 15s
      timeout: 10s
      retries: 5
      start_period: 90s  # TEI takes longer to download model
    restart: unless-stopped

  # ==========================================
  # Ollama - GPU-Accelerated Embeddings
  # Supports Metal (macOS), CUDA (NVIDIA), Vulkan
  # ==========================================
  ollama:
    image: ollama/ollama:latest
    container_name: hyperion-ollama
    profiles:
      - ollama
    ports:
      - "${OLLAMA_PORT:-11434}:11434"
    environment:
      # GPU acceleration (uncomment for NVIDIA GPUs)
      # NVIDIA_VISIBLE_DEVICES: all
      OLLAMA_HOST: 0.0.0.0:11434
    volumes:
      - ollama_data:/root/.ollama
    networks:
      - hyperion-network
    # GPU support (uncomment for NVIDIA)
    # deploy:
    #   resources:
    #     reservations:
    #       devices:
    #         - driver: nvidia
    #           count: 1
    #           capabilities: [gpu]
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:11434/api/tags"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s
    restart: unless-stopped

  # ==========================================
  # Ollama Model Puller (Optional)
  # Automatically downloads embedding model
  # ==========================================
  ollama-pull:
    image: ollama/ollama:latest
    container_name: hyperion-ollama-pull
    profiles:
      - ollama
    depends_on:
      ollama:
        condition: service_healthy
    entrypoint: /bin/sh
    command: >
      -c "
      echo 'Pulling nomic-embed-text model...';
      ollama pull nomic-embed-text;
      echo 'Model pulled successfully!';
      "
    environment:
      OLLAMA_HOST: ollama:11434
    networks:
      - hyperion-network

  # ==========================================
  # Llama.cpp Server (Alternative)
  # Embedded in Hyper binary but can run standalone
  # ==========================================
  llama-cpp:
    image: ghcr.io/ggerganov/llama.cpp:server
    container_name: hyperion-llama-cpp
    profiles:
      - llama
    ports:
      - "${LLAMA_PORT:-8081}:8080"
    command: >
      --model /models/nomic-embed-text-v1.5.Q4_K_M.gguf
      --host 0.0.0.0
      --port 8080
      --embedding
    volumes:
      - ./models:/models  # Mount model directory
    networks:
      - hyperion-network
    # GPU support for NVIDIA (uncomment if needed)
    # deploy:
    #   resources:
    #     reservations:
    #       devices:
    #         - driver: nvidia
    #           count: 1
    #           capabilities: [gpu]
    restart: unless-stopped

# ==========================================
# Networks
# ==========================================
networks:
  hyperion-network:
    driver: bridge
    name: hyperion-network

# ==========================================
# Volumes (Persistent Storage)
# ==========================================
volumes:
  mongodb_data:
    driver: local
  mongodb_config:
    driver: local
  qdrant_data:
    driver: local
  tei_cache:
    driver: local
  ollama_data:
    driver: local
```

---

### Usage Examples

#### 1. Default Setup (MongoDB + Qdrant + TEI)
```bash
# Start with default profile (TEI embeddings)
docker compose -f docker-compose.full.yml up -d

# Configure Hyper to use TEI
export EMBEDDING="ollama"
export TEI_URL="http://localhost:8080"
```

#### 2. Ollama Setup (GPU-Accelerated)
```bash
# Start with Ollama profile
docker compose -f docker-compose.full.yml --profile ollama up -d

# Wait for model download (check logs)
docker compose -f docker-compose.full.yml logs -f ollama-pull

# Configure Hyper to use Ollama
export EMBEDDING="ollama"
export OLLAMA_URL="http://localhost:11434"
export OLLAMA_MODEL="nomic-embed-text"
```

#### 3. Cloud Setup (Voyage AI or OpenAI)
```bash
# Start only MongoDB + Qdrant
docker compose -f docker-compose.full.yml --profile cloud up -d

# Configure Hyper to use Voyage AI
export EMBEDDING="voyage"
export VOYAGE_API_KEY="pa-your-key-here"
export VOYAGE_MODEL="voyage-3"

# Or configure for OpenAI
export EMBEDDING="openai"
export OPENAI_API_KEY="sk-your-key-here"
```

#### 4. Local Development (All Services)
```bash
# Start everything
docker compose -f docker-compose.full.yml --profile ollama --profile tei up -d

# Now you can switch between embedding providers by changing environment variables
```

---

## Embedding Options

Hyperion supports **multiple embedding providers** with different trade-offs:

### 1. **Ollama - ⭐ RECOMMENDED for Local Development**

**What it is:**
- Local embedding service running `llama.cpp` as a server
- GPU-accelerated (Metal/CUDA/Vulkan)
- Separate process (not embedded in binary)

**Performance:**
- **Speed**: 1,000-2,000 embeddings/second (depends on GPU)
- **Latency**: 20-50ms per batch
- **Cost**: **FREE**

**Setup:**
```bash
# Option 1: Install Ollama natively
brew install ollama
ollama serve
ollama pull nomic-embed-text

# Option 2: Use Docker Compose
docker compose -f docker-compose.full.yml --profile ollama up -d

# Configure Hyper
export EMBEDDING="ollama"
export OLLAMA_URL="http://localhost:11434"
export OLLAMA_MODEL="nomic-embed-text"
```

**Pros:**
- ✅ Fast (GPU-accelerated)
- ✅ No API costs
- ✅ Easy to use
- ✅ Can be shared across multiple apps

**Cons:**
- ❌ Requires separate service
- ❌ Uses GPU/CPU resources
- ❌ Network overhead (localhost)

**Best for:** Local development, multiple applications sharing embeddings

---

### 2. **Voyage AI - ⭐ RECOMMENDED for Production**

**What it is:**
- Cloud-based embedding API
- Recommended by Anthropic (makers of Claude)
- High-quality embeddings optimized for semantic search

**Performance:**
- **Speed**: 500-1,000 embeddings/second (network limited)
- **Latency**: 50-200ms per batch (depends on network)
- **Cost**: $0.06 per 1M tokens
- **Quality**: 9.7% better than OpenAI on benchmarks

**Setup:**
```bash
# Get API key from: https://www.voyageai.com/
export EMBEDDING="voyage"
export VOYAGE_API_KEY="pa-your-key-here"
export VOYAGE_MODEL="voyage-3"
```

**Pros:**
- ✅ High quality (best accuracy)
- ✅ No local resources needed
- ✅ Scales automatically
- ✅ 1024 dimensions

**Cons:**
- ❌ API costs ($0.06/1M tokens)
- ❌ Requires internet connection
- ❌ Network latency
- ❌ Rate limits

**Best for:** Production deployments, cloud-native apps, quality-critical use cases

---

### 3. **OpenAI - Cloud**

**What it is:**
- OpenAI's embedding API
- Widely used, high quality

**Performance:**
- **Speed**: 500-1,000 embeddings/second
- **Cost**: $0.02/1M tokens (text-embedding-3-small)
- **Quality**: Excellent

**Setup:**
```bash
# Get API key from: https://platform.openai.com/api-keys
export EMBEDDING="openai"
export OPENAI_API_KEY="sk-your-key-here"
export OPENAI_MODEL="text-embedding-3-small"
```

**Pros:**
- ✅ High quality
- ✅ Reliable service
- ✅ 1536 dimensions

**Cons:**
- ❌ More expensive than Voyage ($0.02 vs $0.06)
- ❌ Requires internet
- ❌ Network latency

**Best for:** When already using OpenAI, need 1536-dim embeddings

---

### 4. **TEI (Text Embeddings Inference) - Local CPU**

**What it is:**
- Hugging Face's embedding inference engine
- CPU-based (no GPU required)
- Free and open-source

**Performance:**
- **Speed**: 15-30 seconds per chunk (very slow)
- **Cost**: **FREE**

**Setup:**
```bash
# Use Docker Compose
docker compose -f docker-compose.full.yml up -d

# Configure Hyper
export EMBEDDING="ollama"
export TEI_URL="http://localhost:8080"
```

**Pros:**
- ✅ Free
- ✅ No API keys
- ✅ Works offline
- ✅ 768 dimensions

**Cons:**
- ❌ Very slow (30-50x slower than llama.cpp)
- ❌ CPU-only (no GPU acceleration)
- ❌ Long startup time (model download)

**Best for:** Testing, development when GPU not available

---

### Comparison Table

| Provider | Speed | Cost | Dimensions | GPU | Internet | Best For |
|----------|-------|------|------------|-----|----------|----------|
| **Ollama** | ⭐⭐⭐⭐⭐ | Free | 768 | ✅ | ❌ | Local dev (default) |
| **Voyage AI** | ⭐⭐⭐⭐ | $0.06/1M | 1024 | ❌ | ✅ | Production (best quality) |
| **OpenAI** | ⭐⭐⭐⭐ | $0.02/1M | 1536 | ❌ | ✅ | OpenAI users |
| **TEI** | ⭐ (slowest) | Free | 768 | ❌ | ❌ | Testing only |

---

## Configuration Guide

### Environment File (`.env.hyper`)

Create a `.env.hyper` file in your project root:

```bash
# ==========================================
# DATABASE
# ==========================================
MONGODB_URI="mongodb://localhost:27017"
MONGODB_DATABASE="coordinator_db"

# ==========================================
# VECTOR DATABASE
# ==========================================
QDRANT_URL="http://localhost:6333"
QDRANT_API_KEY=""
QDRANT_CODE_COLLECTION="code_index"
QDRANT_KNOWLEDGE_COLLECTION="knowledge"

# ==========================================
# EMBEDDING PROVIDER
# Options: ollama, voyage, openai
# ==========================================
EMBEDDING="ollama"

# Llama.cpp (embedded)
LLAMA_MODEL_PATH="models/nomic-embed-text-v1.5.Q4_K_M.gguf"

# Ollama
OLLAMA_URL="http://localhost:11434"
OLLAMA_MODEL="nomic-embed-text"

# Voyage AI
VOYAGE_API_KEY="pa-your-key-here"
VOYAGE_MODEL="voyage-3"

# OpenAI
OPENAI_API_KEY="sk-your-key-here"
OPENAI_MODEL="text-embedding-3-small"

# TEI (local)
TEI_URL="http://localhost:8080"

# ==========================================
# APPLICATION
# ==========================================
HTTP_PORT="7095"
LOG_LEVEL="info"

# ==========================================
# CODE INDEXING (Optional)
# ==========================================
INDEX_SOURCE_PATH="/path/to/your/code"
ENABLE_FILE_WATCHER="false"
CODE_INDEX_AUTO_RECREATE="true"
```

---

### Quick Start Workflows

#### Workflow 0: Super Quick Start (`hyper init`) - ⭐ EASIEST
```bash
# 1. Create a new directory for your project
mkdir my-hyper-project && cd my-hyper-project

# 2. Initialize Hyperion (creates docker-compose.yml + .env.hyper)
hyper init

# 3. Start services
docker compose up -d

# 4. Wait for Ollama model download (check logs)
docker compose logs -f ollama-pull

# 5. Run Hyper
hyper --mode=http

# 6. Open browser
open http://localhost:7095
```

**What `hyper init` creates:**
- `docker-compose.yml` - MongoDB (27017) + Qdrant (7333-7334) + Ollama (7335)
- `.env.hyper` - Pre-configured for local Docker services
- `HYPER_README.md` - Quick reference guide

**Ports used:**
- Qdrant: **7333-7334** (not 6333-6334) to avoid collisions
- Ollama: **7335** (not 11434) to avoid collisions
- MongoDB: 27017 (standard)
- Hyper: 7095 (standard)

---

#### Workflow 1: Local Development (Manual Setup)
```bash
# 1. Start dependencies with Ollama
docker compose -f docker-compose.full.yml --profile ollama up -d

# 2. Wait for model download (check logs)
docker compose -f docker-compose.full.yml logs -f ollama-pull

# 3. Configure
cat > .env.hyper <<EOF
MONGODB_URI="mongodb://localhost:27017"
QDRANT_URL="http://localhost:6333"
EMBEDDING="ollama"
OLLAMA_URL="http://localhost:11434"
OLLAMA_MODEL="nomic-embed-text"
EOF

# 4. Run Hyper
make native
./bin/hyper --mode=http
```

#### Workflow 2: Production (Voyage AI)
```bash
# 1. Start dependencies
docker compose -f docker-compose.full.yml up -d

# 2. Configure
cat > .env.hyper <<EOF
MONGODB_URI="mongodb+srv://user:pass@cluster.mongodb.net"
QDRANT_URL="https://your-cluster.qdrant.io:6333"
QDRANT_API_KEY="your-key"
EMBEDDING="voyage"
VOYAGE_API_KEY="pa-your-key"
EOF

# 3. Run Hyper
make native
./bin/hyper --mode=http
```

---

### Troubleshooting

#### MongoDB Connection Issues
```bash
# Test connection
mongosh "mongodb://localhost:27017"

# Check if running
docker ps | grep mongodb

# View logs
docker logs hyperion-mongodb
```

#### Qdrant Issues
```bash
# Test connection
curl http://localhost:6333/health

# View logs
docker logs hyperion-qdrant
```

#### Ollama Issues
```bash
# Check if running
curl http://localhost:11434/api/tags

# Pull model manually
docker exec hyperion-ollama ollama pull nomic-embed-text

# View logs
docker logs hyperion-ollama
```

#### TEI Issues
```bash
# Check health
curl http://localhost:8080/health

# View logs (model download can take 5-10 minutes)
docker logs hyperion-embedding-tei -f
```

---

## Summary

### Key Takeaways

1. **Makefile Commands**:
   - `make native` - Build production binary
   - `make dev-hot` - Full-stack development
   - `make test` - Run tests
   - `make clean` - Clean artifacts

2. **Docker Compose**:
   - Supports multiple profiles (tei, ollama, cloud)
   - All dependencies in one file
   - Persistent volumes for data

3. **Embedding Options**:
   - **Llama.cpp**: Best for M3 Max, fastest, free
   - **Ollama**: Good for local dev, free
   - **Voyage AI**: Best for production, paid
   - **OpenAI**: Good quality, paid
   - **TEI**: Slowest, free

4. **Recommended Setup**:
   - **Development**: Ollama (GPU-accelerated, free)
   - **Production**: Voyage AI (cloud, best quality)
   - **Testing**: TEI (CPU-only, no GPU required)

---

**Version:** 1.0.0
**Last Updated:** 2025-01-06
**Author:** Hyperion Team
