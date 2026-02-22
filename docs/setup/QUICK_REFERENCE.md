# Hyperion Quick Reference Card

## 🚀 Common Commands

### Initialize New Project
```bash
# Default (Ollama - local, free)
hyper init

# With OpenAI (validates token)
hyper init -provider openai -model gpt-4 -token sk-proj-your-key

# With Anthropic/Claude (validates token)
hyper init -provider anthropic -model claude-sonnet-4 -token sk-ant-your-key

# With Voyage AI (validates token)
hyper init -provider voyage -model voyage-3 -token pa-your-key

# With custom API URL
hyper init -provider openai -model gpt-4 -token sk-key -api-url https://custom.api.com/v1
```

### Build & Run
```bash
# Build production binary
make native

# Build and run (development with hot reload)
make dev-hot

# Run production binary
make run

# Clean build artifacts
make clean
```

### Docker Compose Profiles
```bash
# Ollama (default - GPU-accelerated, free, RECOMMENDED)
docker compose -f docker-compose.full.yml up -d

# TEI (CPU-based, slower)
docker compose -f docker-compose.full.yml --profile tei up -d

# Cloud only (MongoDB + Qdrant for Voyage/OpenAI)
docker compose -f docker-compose.full.yml --profile cloud up -d

# Everything + Admin UI
docker compose -f docker-compose.full.yml --profile all --profile admin up -d

# Stop all services
docker compose -f docker-compose.full.yml down

# Stop and remove volumes
docker compose -f docker-compose.full.yml down -v
```

---

## 🔧 Embedding Configuration

### Ollama (GPU-Accelerated Local) ⭐ RECOMMENDED
```bash
# Start service
docker compose -f docker-compose.full.yml --profile ollama up -d

# Configure
export EMBEDDING="ollama"
export OLLAMA_URL="http://localhost:11434"
export OLLAMA_MODEL="nomic-embed-text"
```
**Speed:** 1,000-2,000 embeddings/s | **Cost:** Free | **Dims:** 768

---

### Voyage AI (Cloud - Production) ⭐ RECOMMENDED
```bash
# Get API key: https://www.voyageai.com/
export EMBEDDING="voyage"
export VOYAGE_API_KEY="pa-your-key"
export VOYAGE_MODEL="voyage-3"
```
**Speed:** 500-1,000 embeddings/s | **Cost:** $0.06/1M tokens | **Dims:** 1024

---

### OpenAI (Cloud)
```bash
# Get API key: https://platform.openai.com/api-keys
export EMBEDDING="openai"
export OPENAI_API_KEY="sk-your-key"
export OPENAI_MODEL="text-embedding-3-small"
```
**Speed:** 500-1,000 embeddings/s | **Cost:** $0.02/1M tokens | **Dims:** 1536

---

### TEI (Local CPU - Slowest)
```bash
# Start service
docker compose -f docker-compose.full.yml up -d

# Configure
export EMBEDDING="ollama"
export TEI_URL="http://localhost:8080"
```
**Speed:** 15-30s per chunk | **Cost:** Free | **Dims:** 768

---

## 📦 Complete Setup

### Option 1: Quick Start (Using `hyper init`)
```bash
# 1. Create a new directory and initialize
mkdir my-hyper-project && cd my-hyper-project
hyper init

# 2. Start all services
docker compose up -d

# 3. Wait for model download
docker compose logs -f ollama-pull

# 4. Run hyper
hyper --mode=http

# 5. Open browser
open http://localhost:7095
```

### Option 2: Manual Setup (Using full compose file)
```bash
# 1. Start all services (Ollama is default profile)
docker compose -f docker-compose.full.yml up -d

# 2. Wait for model download
docker compose -f docker-compose.full.yml logs -f ollama-pull

# 3. Configure
cat > .env.hyper <<EOF
MONGODB_URI="mongodb://localhost:27017"
QDRANT_URL="http://localhost:6333"
EMBEDDING="ollama"
OLLAMA_URL="http://localhost:11434"
OLLAMA_MODEL="nomic-embed-text"
HTTP_PORT="7095"
EOF

# 4. Build and run
make native
./bin/hyper --mode=http
```

---

### Option 2: Production (Voyage AI)
```bash
# 1. Start dependencies
docker compose -f docker-compose.full.yml --profile cloud up -d

# 2. Configure
cat > .env.hyper <<EOF
MONGODB_URI="mongodb+srv://user:pass@cluster.mongodb.net"
QDRANT_URL="https://your-cluster.qdrant.io:6333"
QDRANT_API_KEY="your-key"
EMBEDDING="voyage"
VOYAGE_API_KEY="pa-your-key"
VOYAGE_MODEL="voyage-3"
HTTP_PORT="7095"
EOF

# 3. Build and run
make native
./bin/hyper --mode=http
```

---

## 🛠️ Troubleshooting

### Check Service Health
```bash
# MongoDB
docker ps | grep mongodb
mongosh "mongodb://localhost:27017"

# Qdrant
curl http://localhost:6333/health

# TEI
curl http://localhost:8080/health

# Ollama
curl http://localhost:11434/api/tags

# Hyper
curl http://localhost:7095/health
```

### View Logs
```bash
# All services
docker compose -f docker-compose.full.yml logs -f

# Specific service
docker logs hyperion-mongodb -f
docker logs hyperion-qdrant -f
docker logs hyperion-ollama -f
docker logs hyperion-embedding-tei -f
```

### Reset Everything
```bash
# Stop and remove all data
docker compose -f docker-compose.full.yml down -v

# Clean build artifacts
make clean

# Start fresh
docker compose -f docker-compose.full.yml --profile ollama up -d
make native
```

---

## 🌐 URLs & Ports

| Service | URL | Port |
|---------|-----|------|
| **Hyper UI** | http://localhost:7095 | 7095 |
| **Hyper API** | http://localhost:7095/api/v1 | 7095 |
| **MongoDB** | mongodb://localhost:27017 | 27017 |
| **Qdrant HTTP** | http://localhost:6333 | 6333 |
| **Qdrant gRPC** | http://localhost:6334 | 6334 |
| **TEI** | http://localhost:8080 | 8080 |
| **Ollama** | http://localhost:11434 | 11434 |
| **Mongo Express** | http://localhost:8081 | 8081 |

---

## 📊 Comparison Table

| Provider | Speed | Cost | Dimensions | GPU | Internet | Best For |
|----------|-------|------|------------|-----|----------|----------|
| **Ollama** | ⭐⭐⭐⭐⭐ | Free | 768 | ✅ | ❌ | Local dev (default) |
| **Voyage AI** | ⭐⭐⭐⭐ | $0.06/1M | 1024 | ❌ | ✅ | Production (best quality) |
| **OpenAI** | ⭐⭐⭐⭐ | $0.02/1M | 1536 | ❌ | ✅ | OpenAI users |
| **TEI** | ⭐ | Free | 768 | ❌ | ❌ | Testing only |

---

## 🎯 Recommended Setup

### For Development (Default)
```bash
EMBEDDING="ollama"  # GPU-accelerated, free, fast
```

### For Production
```bash
EMBEDDING="voyage"  # Best quality, cloud-hosted
```

### For Testing (No GPU)
```bash
EMBEDDING="ollama"  # TEI, CPU-only, slower
```

---

## 📚 Full Documentation

For detailed explanations, see:
- **MAKEFILE_AND_DOCKER_GUIDE.md** - Complete reference
- **README.md** - Project overview
- **README-HYPER.md** - Native binary guide
- **.env.example** - Configuration template

---

**Version:** 1.0.0
**Last Updated:** 2025-01-06
