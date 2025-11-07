# Ollama Setup Guide for Hyperion

## Overview

This guide explains how to set up Ollama with Hyperion for local, GPU-accelerated embeddings. Ollama is the **recommended default** for development because it's free, fast, and runs locally.

**Why Ollama?**
- ✅ **Free** - No API costs
- ✅ **Fast** - GPU-accelerated (1,000-2,000 embeddings/sec on M3 Max)
- ✅ **Private** - All data stays local
- ✅ **Easy** - Works out of the box with `hyper init`

---

## Quick Start (Recommended)

### Option 1: Using `hyper init` (Easiest)

```bash
# Initialize Hyperion with Ollama (default)
hyper init

# Start services (includes Ollama + auto model download)
docker compose up -d

# Wait for model download (happens automatically)
docker compose logs -f ollama-pull

# Run Hyper
hyper --mode=http

# Access UI
open http://localhost:7095
```

**That's it!** Ollama is configured and the embedding model is downloaded automatically.

---

### Option 2: Manual Docker Setup

```bash
# Start Ollama container
docker run -d \
  --name ollama \
  -p 11434:11434 \
  -v ollama:/root/.ollama \
  --gpus all \
  ollama/ollama:latest

# Pull embedding model
docker exec ollama ollama pull nomic-embed-text

# Verify
curl http://localhost:11434/api/tags
```

---

## Choosing an Embedding Model

Ollama supports multiple embedding models. Here are the best options for code and technical content:

### 🏆 Recommended: nomic-embed-text (Default)

**Best for:** General use, code, technical documentation

```bash
# Pull the model
ollama pull nomic-embed-text

# Or via Docker
docker exec ollama ollama pull nomic-embed-text
```

**Specifications:**
- **Dimensions:** 768
- **Context window:** 2,048 tokens
- **Model size:** 274MB
- **Performance:** Beats OpenAI text-embedding-ada-002 and text-embedding-3-small
- **Speed:** 1,000-2,000 embeddings/sec (GPU)
- **Use case:** General-purpose, works well with code

**Why nomic-embed-text?**
- Most popular (44.9M pulls)
- Excellent general performance
- Good balance of speed and quality
- Large context window (2K tokens)
- Works well with code and technical content

---

### 🥈 Alternative: mxbai-embed-large

**Best for:** Maximum quality, when performance matters most

```bash
# Pull the model
ollama pull mxbai-embed-large
```

**Specifications:**
- **Dimensions:** 1024
- **Context window:** 512 tokens
- **Model size:** 670MB (335M parameters)
- **Performance:** SOTA for Bert-large sized models, beats OpenAI text-embedding-3-large
- **Speed:** 800-1,500 embeddings/sec (GPU)
- **Use case:** When you need the best quality

**Why mxbai-embed-large?**
- State-of-the-art performance
- Outperforms models 20x its size
- Trained without MTEB overlap (generalizes well)
- Excellent for semantic search

**Trade-offs:**
- Larger size (670MB vs 274MB)
- Smaller context window (512 vs 2,048 tokens)
- Slightly slower

---

### 🥉 Alternative: snowflake-arctic-embed

**Best for:** Flexibility, multiple size options

```bash
# Pull specific size (default is 335m)
ollama pull snowflake-arctic-embed:335m
ollama pull snowflake-arctic-embed:137m
ollama pull snowflake-arctic-embed:110m
ollama pull snowflake-arctic-embed:33m
ollama pull snowflake-arctic-embed:22m
```

**Specifications:**
- **Sizes:** 22M, 33M, 110M, 137M, 335M parameters
- **Context window:** 512 tokens (2K for 137m variant)
- **Performance:** Optimized for retrieval tasks

**Why snowflake-arctic-embed?**
- Multiple size options
- 137m variant has 2K context
- Good retrieval performance

---

## Model Comparison Table

| Model | Size | Dimensions | Context | Speed | Best For |
|-------|------|------------|---------|-------|----------|
| **nomic-embed-text** ⭐ | 274MB | 768 | 2,048 | ⭐⭐⭐⭐⭐ | Default (code + general) |
| **mxbai-embed-large** | 670MB | 1024 | 512 | ⭐⭐⭐⭐ | Maximum quality |
| **snowflake-arctic-embed:335m** | 669MB | ? | 512 | ⭐⭐⭐⭐ | Retrieval tasks |
| **snowflake-arctic-embed:137m** | 274MB | ? | 2,048 | ⭐⭐⭐⭐ | Balance (2K context) |
| **all-minilm** | 45MB | 384 | 256 | ⭐⭐⭐⭐⭐ | Speed, small size |

---

## Configuration

### Using `hyper init` (Automatic)

When you run `hyper init`, these settings are automatically configured in `.env.hyper`:

```bash
# Embedding configuration (automatically created by hyper init)
EMBEDDING=ollama
OLLAMA_URL=http://localhost:7335
OLLAMA_MODEL=nomic-embed-text
```

**Note:** Port 7335 is used to avoid conflicts with system Ollama (11434).

---

### Manual Configuration

If you're using system Ollama or need custom settings:

```bash
# .env.hyper
EMBEDDING=ollama
OLLAMA_URL=http://localhost:11434
OLLAMA_MODEL=nomic-embed-text
```

**For different models:**
```bash
# Use mxbai-embed-large
OLLAMA_MODEL=mxbai-embed-large

# Use snowflake-arctic-embed (default size)
OLLAMA_MODEL=snowflake-arctic-embed

# Use specific snowflake size
OLLAMA_MODEL=snowflake-arctic-embed:137m
```

---

## Installation Methods

### Method 1: Docker Compose (Recommended)

**Using `hyper init`:**

```bash
# Initialize (creates docker-compose.yml with Ollama)
hyper init

# Start services
docker compose up -d

# Check status
docker compose ps
```

**Generated docker-compose.yml includes:**
```yaml
ollama:
  image: ollama/ollama:latest
  ports:
    - "7335:11434"
  volumes:
    - ollama-data:/root/.ollama
  deploy:
    resources:
      reservations:
        devices:
          - driver: nvidia
            count: all
            capabilities: [gpu]

ollama-pull:
  image: ollama/ollama:latest
  depends_on:
    - ollama
  command: pull nomic-embed-text
```

---

### Method 2: System Ollama (macOS/Linux)

**macOS:**
```bash
# Download from https://ollama.com/download
# Or install via Homebrew
brew install ollama

# Start Ollama
ollama serve &

# Pull embedding model
ollama pull nomic-embed-text

# Verify
ollama list
```

**Linux:**
```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Pull embedding model
ollama pull nomic-embed-text

# Verify
ollama list
```

**Configure Hyperion to use system Ollama:**
```bash
# .env.hyper
EMBEDDING=ollama
OLLAMA_URL=http://localhost:11434
OLLAMA_MODEL=nomic-embed-text
```

---

### Method 3: Docker (Manual)

```bash
# Start Ollama container
docker run -d \
  --name ollama \
  -p 11434:11434 \
  -v ollama:/root/.ollama \
  --gpus all \
  ollama/ollama:latest

# Pull model
docker exec ollama ollama pull nomic-embed-text

# Verify
docker exec ollama ollama list
```

---

## GPU Acceleration

### macOS (Metal)

Ollama automatically uses Metal GPU acceleration on Apple Silicon (M1/M2/M3):

```bash
# No additional configuration needed
ollama pull nomic-embed-text
ollama run nomic-embed-text "test"

# Metal acceleration is automatic
# Speed: 1,000-2,000 embeddings/sec on M3 Max
```

---

### Linux (NVIDIA CUDA)

**Prerequisites:**
- NVIDIA GPU
- CUDA drivers installed
- nvidia-docker2 installed

```bash
# Install nvidia-docker2
distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
curl -s -L https://nvidia.github.io/nvidia-docker/gpgkey | sudo apt-key add -
curl -s -L https://nvidia.github.io/nvidia-docker/$distribution/nvidia-docker.list | \
  sudo tee /etc/apt/sources.list.d/nvidia-docker.list

sudo apt-get update
sudo apt-get install -y nvidia-docker2
sudo systemctl restart docker

# Run Ollama with GPU
docker run -d \
  --name ollama \
  --gpus all \
  -p 11434:11434 \
  -v ollama:/root/.ollama \
  ollama/ollama:latest
```

---

### Linux (AMD ROCm)

**Prerequisites:**
- AMD GPU
- ROCm drivers installed

```bash
# Run Ollama with ROCm
docker run -d \
  --name ollama \
  --device /dev/kfd \
  --device /dev/dri \
  -p 11434:11434 \
  -v ollama:/root/.ollama \
  ollama/ollama:rocm
```

---

## Testing & Verification

### 1. Check Ollama is Running

```bash
# Health check
curl http://localhost:11434/api/tags

# Or with hyper init ports
curl http://localhost:7335/api/tags
```

**Expected response:**
```json
{
  "models": [
    {
      "name": "nomic-embed-text:latest",
      "size": 274015488
    }
  ]
}
```

---

### 2. Test Embedding Generation

```bash
# Generate test embedding
curl http://localhost:11434/api/embeddings -d '{
  "model": "nomic-embed-text",
  "prompt": "function calculateSum(a, b) { return a + b; }"
}'
```

**Expected response:**
```json
{
  "embedding": [0.123, -0.456, 0.789, ...]
}
```

---

### 3. Test with Hyperion

```bash
# Start Hyperion
hyper --mode=http

# Check logs for embedding initialization
# Should see: "Initialized Ollama embedding client"

# Index some code
# Use Code Search page in UI: http://localhost:7095
# Add folder → Watch embedding progress
```

---

## Performance Benchmarks

### Embedding Generation Speed

| Hardware | Model | Speed | Notes |
|----------|-------|-------|-------|
| **M3 Max (Metal)** | nomic-embed-text | 1,500-2,000/sec | GPU accelerated |
| **M3 Max (Metal)** | mxbai-embed-large | 800-1,200/sec | Larger model |
| **M2 Pro (Metal)** | nomic-embed-text | 1,000-1,500/sec | GPU accelerated |
| **M1 (Metal)** | nomic-embed-text | 600-1,000/sec | GPU accelerated |
| **NVIDIA RTX 4090** | nomic-embed-text | 3,000-5,000/sec | High-end GPU |
| **NVIDIA RTX 3080** | nomic-embed-text | 2,000-3,000/sec | Mid-range GPU |
| **CPU (16 cores)** | nomic-embed-text | 50-100/sec | No GPU |

---

### Indexing Performance

| Task | Model | Time (10K files) | Notes |
|------|-------|------------------|-------|
| **Initial indexing** | nomic-embed-text | 5-10 min | M3 Max |
| **Incremental update** | nomic-embed-text | 10-30 sec | Changed files only |
| **Search query** | nomic-embed-text | 100-200ms | Vector search |

---

## Troubleshooting

### Issue: Ollama not found

**Error:**
```
Error: Failed to connect to Ollama at http://localhost:11434
```

**Solutions:**

1. **Check if Ollama is running:**
```bash
curl http://localhost:11434/api/tags
```

2. **Start Ollama:**
```bash
# Docker
docker compose up -d ollama

# System Ollama
ollama serve &
```

3. **Check port configuration:**
```bash
# If using hyper init, use port 7335
curl http://localhost:7335/api/tags

# Update .env.hyper
OLLAMA_URL=http://localhost:7335
```

---

### Issue: Model not found

**Error:**
```
Error: model "nomic-embed-text" not found
```

**Solution:**
```bash
# Docker
docker exec ollama ollama pull nomic-embed-text

# System Ollama
ollama pull nomic-embed-text

# Verify
ollama list
```

---

### Issue: Slow embedding generation

**Symptoms:**
- Embeddings take 5+ seconds each
- Indexing very slow
- CPU usage high

**Causes & Solutions:**

1. **CPU mode (no GPU):**
```bash
# Check if GPU is detected
# macOS
system_profiler SPDisplaysDataType | grep Metal

# Linux
nvidia-smi
```

2. **Wrong Ollama version:**
```bash
# Ensure using GPU-enabled image
docker pull ollama/ollama:latest
```

3. **Resource limits:**
```bash
# Check Docker resources
docker stats ollama

# Increase if needed (Docker Desktop → Settings → Resources)
```

---

### Issue: Out of memory

**Error:**
```
Error: failed to allocate memory
```

**Solutions:**

1. **Use smaller model:**
```bash
# Switch to smaller model
ollama pull all-minilm
# Update .env.hyper
OLLAMA_MODEL=all-minilm
```

2. **Increase Docker memory:**
```bash
# Docker Desktop → Settings → Resources
# Increase memory to 8GB+
```

3. **Reduce batch size:**
```bash
# .env.hyper
EMBEDDING_BATCH_SIZE=10  # Default is 50
```

---

### Issue: Port conflict

**Error:**
```
Error: port 11434 already in use
```

**Solution:**

1. **Using `hyper init` (automatic):**
```bash
# hyper init uses port 7335 by default (no conflict)
hyper init
docker compose up -d
```

2. **Manual fix:**
```bash
# Stop conflicting service
pkill ollama

# Or use different port
docker run -d -p 11435:11434 ollama/ollama

# Update .env.hyper
OLLAMA_URL=http://localhost:11435
```

---

## Switching Models

### From nomic-embed-text to mxbai-embed-large

```bash
# 1. Pull new model
docker exec ollama ollama pull mxbai-embed-large

# 2. Update .env.hyper
OLLAMA_MODEL=mxbai-embed-large

# 3. Restart Hyper
pkill hyper
hyper --mode=http

# 4. Re-index (embeddings are different dimensions)
# Use UI to clear and re-index code
```

**Important:** Different models have different embedding dimensions. You must **re-index** all content when switching models.

---

### From Ollama to Cloud Provider

```bash
# Switch to Voyage AI
# 1. Update .env.hyper
EMBEDDING=voyage
VOYAGE_API_KEY=pa-your-key
VOYAGE_MODEL=voyage-3

# 2. Restart Hyper
pkill hyper
hyper --mode=http

# 3. Re-index content
```

---

## Advanced Configuration

### Custom Ollama Installation

```bash
# Custom port
docker run -d \
  --name ollama-custom \
  -p 12345:11434 \
  -v ollama-data:/root/.ollama \
  --gpus all \
  ollama/ollama:latest

# Update .env.hyper
OLLAMA_URL=http://localhost:12345
```

---

### Multiple Models

```bash
# Pull multiple models
ollama pull nomic-embed-text
ollama pull mxbai-embed-large
ollama pull all-minilm

# Switch between them by updating .env.hyper
OLLAMA_MODEL=nomic-embed-text
# Or
OLLAMA_MODEL=mxbai-embed-large
```

---

### Remote Ollama

```bash
# Run Ollama on server
# Server: 192.168.1.100
ollama serve

# Client: Update .env.hyper
OLLAMA_URL=http://192.168.1.100:11434
OLLAMA_MODEL=nomic-embed-text
```

---

## Best Practices

### For Development

```bash
# Use default nomic-embed-text
EMBEDDING=ollama
OLLAMA_MODEL=nomic-embed-text
```

**Why:**
- Fast enough (1,000-2,000/sec)
- Good quality
- Large context window (2K)
- Works well with code

---

### For Maximum Quality

```bash
# Use mxbai-embed-large
EMBEDDING=ollama
OLLAMA_MODEL=mxbai-embed-large
```

**Why:**
- Best quality
- State-of-the-art performance
- Worth the slowdown for critical apps

---

### For Speed

```bash
# Use all-minilm
EMBEDDING=ollama
OLLAMA_MODEL=all-minilm
```

**Why:**
- Smallest size (45MB)
- Fastest (2,000+ embeddings/sec)
- Good enough for most tasks

---

## FAQ

### Q: Which model should I use?

**A:** Start with **nomic-embed-text** (default). It's the best balance of speed, quality, and context window.

Only switch if:
- You need maximum quality → **mxbai-embed-large**
- You need maximum speed → **all-minilm**
- You need 2K context + smaller size → **snowflake-arctic-embed:137m**

---

### Q: Can I use Ollama for both embeddings and AI?

**A:** Yes, but Hyperion currently only uses Ollama for embeddings. For AI/chat, you need OpenAI, Anthropic, or other LLM provider.

```bash
# Example: Anthropic for AI, Ollama for embeddings
AI_PROVIDER=anthropic
ANTHROPIC_API_KEY=sk-ant-...

EMBEDDING=ollama
OLLAMA_MODEL=nomic-embed-text
```

---

### Q: Do I need to re-index when switching models?

**A:** Yes. Different models produce different embedding dimensions and values. You must re-index all content.

---

### Q: Can I run Ollama without GPU?

**A:** Yes, but it will be much slower (50-100 embeddings/sec vs 1,000-2,000/sec).

---

### Q: How much disk space do models need?

**A:**
- nomic-embed-text: 274MB
- mxbai-embed-large: 670MB
- all-minilm: 45MB
- snowflake-arctic-embed:335m: 669MB

---

### Q: Can I use custom/fine-tuned models?

**A:** Yes, if they're compatible with Ollama. Create a Modelfile and import:

```bash
# Create Modelfile
cat > Modelfile <<EOF
FROM ./your-model.gguf
EOF

# Import
ollama create custom-model -f Modelfile

# Use
OLLAMA_MODEL=custom-model
```

---

## Summary

**Recommended Setup:**

1. **Use `hyper init`** - Automatic Ollama setup
2. **Use nomic-embed-text** - Best default model
3. **Use Docker Compose** - Easiest management
4. **Enable GPU** - 10-20x faster

**Quick Start Command:**
```bash
hyper init && docker compose up -d && hyper --mode=http
```

**Performance:**
- M3 Max: 1,500-2,000 embeddings/sec
- 10K files indexed in 5-10 minutes
- Search queries in 100-200ms

**Cost:** Free, runs locally

---

## Additional Resources

- **Ollama Documentation:** https://ollama.com/docs
- **Model Library:** https://ollama.com/library
- **nomic-embed-text:** https://ollama.com/library/nomic-embed-text
- **mxbai-embed-large:** https://ollama.com/library/mxbai-embed-large
- **GitHub:** https://github.com/ollama/ollama

---

**Version:** 1.0.0
**Last Updated:** 2025-11-06
**Default Model:** nomic-embed-text
**Recommended For:** Development, local use, privacy
