# Hyper Init Command - Quick Project Setup

## Overview

The `hyper init` command is the **fastest way** to set up a new Hyperion project. It creates all necessary files for a local development environment with MongoDB, Qdrant, and Ollama.

## What It Does

Running `hyper init` in any directory creates three files:

1. **`docker-compose.yml`** - Container definitions for all dependencies
2. **`.env.hyper`** - Pre-configured environment variables
3. **`HYPER_README.md`** - Quick reference guide for the setup

## Quick Start

### 1. Create a New Project

```bash
# Create a new directory
mkdir my-hyper-project && cd my-hyper-project

# Initialize Hyperion
hyper init
```

### 2. Start Services

```bash
# Start all services (MongoDB + Qdrant + Ollama)
docker compose up -d

# Watch Ollama model download (first time only, 2-5 minutes)
docker compose logs -f ollama-pull
```

### 3. Run Hyper

```bash
# Start Hyper in HTTP mode
hyper --mode=http
```

### 4. Access UI

Open your browser to: **http://localhost:7095**

---

## Port Configuration

To **avoid port collisions**, `hyper init` uses custom ports for some services:

| Service | Standard Port | Custom Port | Why Custom? |
|---------|---------------|-------------|-------------|
| **MongoDB** | 27017 | 27017 | Standard (no collision risk) |
| **Qdrant HTTP** | 6333 | **7333** | Avoid conflicts |
| **Qdrant gRPC** | 6334 | **7334** | Avoid conflicts |
| **Ollama** | 11434 | **7335** | Avoid conflicts |
| **Hyper** | 7095 | 7095 | Standard |

**Why custom ports?**
- Port 6333-6334 may be used by other services
- Port 11434 may conflict with existing Ollama installations
- Ports 7333-7335 are less commonly used

---

## Generated Files

### 1. docker-compose.yml

Creates a complete local development stack:

```yaml
services:
  - mongodb:      Port 27017 (standard)
  - qdrant:       Ports 7333-7334 (custom)
  - ollama:       Port 7335 (custom)
  - ollama-pull:  One-time model downloader
```

**Features:**
- Health checks for all services
- Persistent volumes for data
- Auto-download of nomic-embed-text model
- Ready for GPU acceleration (Metal/CUDA)

### 2. .env.hyper

Pre-configured environment variables pointing to Docker services:

```bash
# MongoDB
MONGODB_URI=mongodb://admin:admin123@localhost:27017/?authSource=admin
MONGODB_DATABASE=hyper_db

# Qdrant (custom port 7333)
QDRANT_URL=http://localhost:7333
QDRANT_API_KEY=

# Ollama (custom port 7335)
EMBEDDING=ollama
OLLAMA_URL=http://localhost:7335
OLLAMA_MODEL=nomic-embed-text
```

### 3. HYPER_README.md

Quick reference guide with:
- Setup instructions
- Common commands
- Troubleshooting tips
- Service URLs
- Configuration examples

---

## Customization

### Using Cloud Services

You can easily switch to cloud services by editing `.env.hyper`:

#### MongoDB Atlas
```bash
MONGODB_URI=mongodb+srv://user:pass@cluster.mongodb.net/?retryWrites=true&w=majority
MONGODB_DATABASE=hyper_db
```

#### Qdrant Cloud
```bash
QDRANT_URL=https://your-cluster.qdrant.io:6333
QDRANT_API_KEY=your-api-key
```

#### Voyage AI (Embeddings)
```bash
EMBEDDING=voyage
VOYAGE_API_KEY=pa-your-key-here
VOYAGE_MODEL=voyage-3
```

### Changing Ports

If you need to change the custom ports, edit both:

1. **`docker-compose.yml`** - Update the port mappings
2. **`.env.hyper`** - Update the URLs

Example (change Qdrant to 8333):
```yaml
# docker-compose.yml
qdrant:
  ports:
    - "8333:6333"  # Change 7333 -> 8333
```

```bash
# .env.hyper
QDRANT_URL=http://localhost:8333  # Change 7333 -> 8333
```

---

## Common Commands

### View Service Logs
```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f ollama
docker compose logs -f mongodb
docker compose logs -f qdrant
```

### Check Service Health
```bash
# MongoDB
docker exec hyper-mongodb mongosh --eval "db.adminCommand('ping')"

# Qdrant
curl http://localhost:7333/health

# Ollama
curl http://localhost:7335/api/tags
```

### Restart Services
```bash
# Restart all
docker compose restart

# Restart specific service
docker compose restart ollama
```

### Stop Services
```bash
# Stop (preserves data)
docker compose down

# Stop and remove all data
docker compose down -v
```

---

## Troubleshooting

### Port Already in Use

If you see port conflicts:

```bash
# Check what's using the port
lsof -i :7333  # For Qdrant
lsof -i :7335  # For Ollama

# Kill the process or change ports in docker-compose.yml
```

### Ollama Model Not Downloaded

```bash
# Check pull logs
docker compose logs ollama-pull

# Manually trigger download
docker exec hyper-ollama ollama pull nomic-embed-text

# Verify model exists
docker exec hyper-ollama ollama list
```

### MongoDB Connection Failed

```bash
# Check if container is running
docker ps | grep hyper-mongodb

# Check MongoDB logs
docker logs hyper-mongodb

# Test connection
mongosh "mongodb://admin:admin123@localhost:27017/?authSource=admin"
```

### Qdrant Not Responding

```bash
# Check if container is running
docker ps | grep hyper-qdrant

# Check Qdrant logs
docker logs hyper-qdrant

# Test API
curl http://localhost:7333/health
```

---

## Make Command

You can also use the Makefile:

```bash
# From the hyper repository root
make init
```

This runs:
1. `make native` - Builds the hyper binary
2. `./bin/hyper init` - Runs the init command

---

## Comparison: init vs docker-compose.full.yml

| Feature | `hyper init` | `docker-compose.full.yml` |
|---------|-------------|---------------------------|
| **Services** | MongoDB + Qdrant + Ollama | All options (TEI, Ollama, Admin UI) |
| **Ports** | Custom (7333-7335) | Standard (6333-6334, 11434) |
| **Profiles** | None (simple) | Multiple (tei, cloud, all, admin) |
| **Use Case** | Quick local setup | Advanced/production |
| **Files** | 3 files | 1 file |
| **Setup Time** | 30 seconds | 2-5 minutes |

**When to use each:**
- **`hyper init`**: Starting a new project, quick local development
- **`docker-compose.full.yml`**: Testing different embedding providers, production setup

---

## Next Steps After Init

### 1. Index Your Code

Edit `.env.hyper` and add your code paths:

```bash
INDEX_SOURCE_PATH=/path/to/your/code,/path/to/another/project
```

Restart Hyper, and it will auto-scan on startup.

### 2. Enable File Watcher

For automatic re-indexing when files change:

```bash
ENABLE_FILE_WATCHER=true
```

### 3. Configure Claude Code Integration

```bash
# From the hyper repository
make configure-native
```

Or manually add to Claude Code settings.

---

## Example Workflow

Complete workflow from scratch:

```bash
# 1. Create project directory
mkdir my-ai-project && cd my-ai-project

# 2. Initialize Hyperion
hyper init

# 3. Start services
docker compose up -d

# 4. Wait for Ollama model (first time)
docker compose logs -f ollama-pull
# Press Ctrl+C when you see "Model downloaded successfully!"

# 5. Edit .env.hyper to add your code path
echo "INDEX_SOURCE_PATH=$(pwd)" >> .env.hyper

# 6. Start Hyper
hyper --mode=http

# 7. Open browser
open http://localhost:7095

# 8. Your code is now indexed and searchable!
```

---

## Summary

### ✅ Pros
- **Fast**: 30-second setup
- **Simple**: No complex configuration
- **Collision-free**: Custom ports avoid conflicts
- **Complete**: All dependencies included
- **Documented**: Auto-generated README

### 📝 Created Files
- `docker-compose.yml` - Service definitions
- `.env.hyper` - Configuration
- `HYPER_README.md` - Quick reference

### 🚀 Default Setup
- **Database**: MongoDB (local)
- **Vectors**: Qdrant (local, ports 7333-7334)
- **Embeddings**: Ollama (local, port 7335)
- **Model**: nomic-embed-text (768 dims, GPU-accelerated)

### 🎯 Perfect For
- New projects
- Local development
- Quick testing
- Learning Hyperion

---

**Version:** 1.0.0
**Last Updated:** 2025-01-06
**Command:** `hyper init`
