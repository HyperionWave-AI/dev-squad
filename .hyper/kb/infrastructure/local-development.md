# Local development

**Collection:** infrastructure
**Created:** 2025-11-20

---

## Overview

Hyperion is configured entirely through environment variables. This document provides a comprehensive reference for all configuration options organized by category.

---

## Database Configuration

### MongoDB

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MONGODB_URI` | **Yes** | - | MongoDB connection string |
| `MONGODB_DATABASE` | No | `hyper_db` | Database name |

**Example Values:**

```bash
# Local development
MONGODB_URI=mongodb://admin:admin123@localhost:27017/?authSource=admin
MONGODB_DATABASE=hyper_db

# MongoDB Atlas (Cloud)
MONGODB_URI=mongodb+srv://username:password@cluster0.mongodb.net/hyper_db?retryWrites=true&w=majority

# Replica Set
MONGODB_URI=mongodb://user:password@mongo1:27017,mongo2:27017,mongo3:27017/hyper_db?replicaSet=rs0&authSource=admin
```

### Qdrant

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `QDRANT_URL` | Yes | `http://localhost:7333` | Qdrant HTTP API endpoint |
| `QDRANT_API_KEY` | No | - | API key for authentication |
| `QDRANT_CODE_COLLECTION` | No | `code_index` | Collection name for code indexing |
| `QDRANT_KNOWLEDGE_COLLECTION` | No | `hyper-knowledge-base` | Collection name for knowledge base |

**Example Values:**

```bash
# Local development
QDRANT_URL=http://localhost:7333
QDRANT_CODE_COLLECTION=code_index
QDRANT_KNOWLEDGE_COLLECTION=hyper-knowledge-base

# Production with authentication
QDRANT_URL=https://qdrant.your-domain.com
QDRANT_API_KEY=your-api-key-here

# Qdrant Cloud
QDRANT_URL=https://your-cluster.cloud.qdrant.io:6333
QDRANT_API_KEY=your-cloud-api-key
```

---

## Embedding Providers

### Provider Selection

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `EMBEDDING` | No | `local` | Embedding provider: `local`, `ollama`, `voyage`, `openai` |

### TEI (Text Embeddings Inference) - Local/Default

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `TEI_URL` | No | `http://embedding-service:8080` | TEI service endpoint |

**Model:** `nomic-embed-text-v1.5`
**Dimensions:** 768

**Example:**

```bash
# Local TEI service
TEI_URL=http://localhost:8080
EMBEDDING=local

# Docker service name
TEI_URL=http://embedding-service:8080
```

### Ollama - Local GPU

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `OLLAMA_URL` | No | `http://localhost:7335` | Ollama service endpoint |
| `OLLAMA_MODEL` | No | `nomic-embed-text` | Embedding model name |

**Example:**

```bash
EMBEDDING=ollama
OLLAMA_URL=http://localhost:7335
OLLAMA_MODEL=nomic-embed-text
```

### Voyage AI - Cloud

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `VOYAGE_API_KEY` | **Yes** | - | Voyage AI API key |
| `VOYAGE_MODEL` | No | `voyage-3` | Voyage model name |

**Available Models:**
- `voyage-3` - Latest model (1024 dimensions)
- `voyage-2` - Previous generation
- `voyage-code-2` - Optimized for code

**Example:**

```bash
EMBEDDING=voyage
VOYAGE_API_KEY=pa-your-voyage-api-key
VOYAGE_MODEL=voyage-3
```

### OpenAI - Cloud

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `OPENAI_API_KEY` | **Yes** | - | OpenAI API key |
| `OPENAI_MODEL` | No | `text-embedding-3-small` | OpenAI embedding model |

**Available Models:**
- `text-embedding-3-small` - 1536 dimensions, cost-effective
- `text-embedding-3-large` - 3072 dimensions, highest quality
- `text-embedding-ada-002` - Legacy model

**Example:**

```bash
EMBEDDING=openai
OPENAI_API_KEY=sk-your-openai-api-key
OPENAI_MODEL=text-embedding-3-small
```

---

## AI Provider Configuration

### Claude (Anthropic)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ANTHROPIC_API_KEY` | **Yes** | - | Anthropic API key |
| `ANTHROPIC_MODEL` | No | `claude-sonnet-4-5` | Claude model name |

**Available Models:**
- `claude-sonnet-4-5` - Latest balanced model (recommended)
- `claude-opus-4` - Most capable model
- `claude-haiku-3-5` - Fastest, most cost-effective

**Example:**

```bash
ANTHROPIC_API_KEY=sk-ant-your-anthropic-api-key
ANTHROPIC_MODEL=claude-sonnet-4-5
```

### OpenAI (Chat)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `OPENAI_API_KEY` | **Yes** | - | OpenAI API key |
| `OPENAI_CHAT_MODEL` | No | `gpt-4-turbo` | Chat model name |

**Available Models:**
- `gpt-4-turbo` - Latest GPT-4 variant
- `gpt-4` - Standard GPT-4
- `gpt-3.5-turbo` - Faster, more economical

**Example:**

```bash
OPENAI_API_KEY=sk-your-openai-api-key
OPENAI_CHAT_MODEL=gpt-4-turbo
```

---

## Code Indexing Configuration

### Indexing Behavior

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `INDEX_SOURCE_PATH` | No | - | Comma-separated paths to auto-index on startup |
| `ENABLE_FILE_WATCHER` | No | `false` | Enable automatic reindexing on file changes |
| `CODE_INDEX_AUTO_RECREATE` | No | `false` | Auto-recreate index on dimension mismatch |
| `KB_DOCS_PATH` | No | `.hyper/kb` | Path to knowledge base markdown files |

**Example:**

```bash
# Auto-index multiple paths
INDEX_SOURCE_PATH=/path/to/backend,/path/to/frontend

# Enable file watching
ENABLE_FILE_WATCHER=true

# Auto-handle dimension mismatches (development only)
CODE_INDEX_AUTO_RECREATE=true

# Custom KB docs location
KB_DOCS_PATH=/custom/kb/path
```

### Chunking Configuration

**Note:** Chunk sizes are configured per-folder via API, not environment variables.

**Chunk Size Options:**
- `xs` - 50 lines
- `s` - 100 lines
- `m` - 200 lines (default)
- `l` - 400 lines
- `xl` - 800 lines

---

## Application Settings

### HTTP Server

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `HTTP_PORT` | No | `7095` | HTTP server port |
| `GIN_MODE` | No | `debug` | Gin mode: `debug`, `release`, `test` |

**Example:**

```bash
# Development
HTTP_PORT=7095
GIN_MODE=debug

# Production
HTTP_PORT=8080
GIN_MODE=release
```

### Logging

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `LOG_LEVEL` | No | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | No | `json` | Log format: `json`, `console` |

**Example:**

```bash
# Development
LOG_LEVEL=debug
LOG_FORMAT=console

# Production
LOG_LEVEL=info
LOG_FORMAT=json
```

### Security

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `JWT_SECRET` | **Yes** | - | Secret key for JWT signing |
| `DEV_MODE` | No | `false` | Bypass JWT validation (development only) |

**Example:**

```bash
# Production
JWT_SECRET=your-super-secret-jwt-key-min-32-chars

# Development (bypass auth)
DEV_MODE=true
```

### Rate Limiting

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `RATE_LIMIT_ENABLED` | No | `true` | Enable rate limiting |
| `RATE_LIMIT_REQUESTS` | No | `100` | Requests per window |
| `RATE_LIMIT_WINDOW` | No | `60` | Window duration in seconds |

**Example:**

```bash
# Strict rate limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=50
RATE_LIMIT_WINDOW=60

# Disable (development only)
RATE_LIMIT_ENABLED=false
```

---

## Frontend Configuration

### Vite Environment Variables

**Note:** Vite requires `VITE_` prefix for client-side access.

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `VITE_BACKEND_URL` | No | `http://localhost:7095` | Backend API URL (dev proxy) |

**Example:**

```bash
# Development
VITE_BACKEND_URL=http://localhost:7095

# Production (use relative paths, no proxy)
# Leave unset - frontend uses /api/v1 directly
```

---

## Docker Compose Configuration

### Sample docker-compose.yml

```yaml
version: '3.8'

services:
  mongodb:
    image: mongo:7.0
    environment:
      MONGO_INITDB_ROOT_USERNAME: admin
      MONGO_INITDB_ROOT_PASSWORD: admin123
    ports:
      - "27017:27017"
    volumes:
      - mongodb_data:/data/db

  qdrant:
    image: qdrant/qdrant:v1.7.4
    ports:
      - "7333:6333"
    volumes:
      - qdrant_data:/qdrant/storage

  tei:
    image: ghcr.io/huggingface/text-embeddings-inference:cpu-1.2
    command: --model-id nomic-ai/nomic-embed-text-v1.5
    ports:
      - "8080:80"

  coordinator:
    build: ./hyper
    environment:
      # Database
      MONGODB_URI: mongodb://admin:admin123@mongodb:27017/?authSource=admin
      MONGODB_DATABASE: hyper_db
      QDRANT_URL: http://qdrant:6333

      # Embedding
      EMBEDDING: local
      TEI_URL: http://tei:80

      # AI Provider
      ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}

      # Application
      HTTP_PORT: 7095
      LOG_LEVEL: info
      GIN_MODE: release

      # Security
      JWT_SECRET: ${JWT_SECRET}
    ports:
      - "7095:7095"
    depends_on:
      - mongodb
      - qdrant
      - tei

volumes:
  mongodb_data:
  qdrant_data:
```

---

## Environment File Templates

### Development (.env.development)

```bash
# Database
MONGODB_URI=mongodb://admin:admin123@localhost:27017/?authSource=admin
MONGODB_DATABASE=hyper_db
QDRANT_URL=http://localhost:7333

# Embedding (Local TEI)
EMBEDDING=local
TEI_URL=http://localhost:8080

# AI Provider
ANTHROPIC_API_KEY=sk-ant-your-key-here

# Code Indexing
INDEX_SOURCE_PATH=/path/to/your/code
ENABLE_FILE_WATCHER=true
CODE_INDEX_AUTO_RECREATE=true

# Application
HTTP_PORT=7095
GIN_MODE=debug
LOG_LEVEL=debug
LOG_FORMAT=console

# Security
JWT_SECRET=dev-jwt-secret-key-change-in-production
DEV_MODE=true

# Rate Limiting
RATE_LIMIT_ENABLED=false

# Frontend
VITE_BACKEND_URL=http://localhost:7095
```

### Production (.env.production)

```bash
# Database
MONGODB_URI=mongodb+srv://prod-user:prod-pass@cluster.mongodb.net/hyper_prod?retryWrites=true&w=majority
MONGODB_DATABASE=hyper_prod
QDRANT_URL=https://qdrant.your-domain.com
QDRANT_API_KEY=your-production-qdrant-key

# Embedding (Voyage AI Cloud)
EMBEDDING=voyage
VOYAGE_API_KEY=pa-your-voyage-key
VOYAGE_MODEL=voyage-3

# AI Provider
ANTHROPIC_API_KEY=sk-ant-production-key

# Code Indexing
ENABLE_FILE_WATCHER=false
CODE_INDEX_AUTO_RECREATE=false
KB_DOCS_PATH=/app/kb

# Application
HTTP_PORT=8080
GIN_MODE=release
LOG_LEVEL=info
LOG_FORMAT=json

# Security
JWT_SECRET=super-secret-production-jwt-key-min-32-characters
DEV_MODE=false

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=60
```

---

## Kubernetes ConfigMap Example

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: hyperion-config
  namespace: hyperion-prod
data:
  MONGODB_DATABASE: "hyper_prod"
  QDRANT_URL: "http://qdrant-service:6333"
  QDRANT_CODE_COLLECTION: "code_index"
  QDRANT_KNOWLEDGE_COLLECTION: "hyper-knowledge-base"
  EMBEDDING: "voyage"
  VOYAGE_MODEL: "voyage-3"
  HTTP_PORT: "8080"
  GIN_MODE: "release"
  LOG_LEVEL: "info"
  LOG_FORMAT: "json"
  ENABLE_FILE_WATCHER: "false"
  CODE_INDEX_AUTO_RECREATE: "false"
  RATE_LIMIT_ENABLED: "true"
  RATE_LIMIT_REQUESTS: "100"
  RATE_LIMIT_WINDOW: "60"
---
apiVersion: v1
kind: Secret
metadata:
  name: hyperion-secrets
  namespace: hyperion-prod
type: Opaque
stringData:
  MONGODB_URI: "mongodb+srv://user:pass@cluster.mongodb.net/hyper_prod"
  QDRANT_API_KEY: "your-qdrant-key"
  ANTHROPIC_API_KEY: "sk-ant-your-key"
  VOYAGE_API_KEY: "pa-your-voyage-key"
  JWT_SECRET: "super-secret-jwt-key"
```

---

## Validation and Defaults

### Required Variables by Environment

**Development:**
- `MONGODB_URI`
- `ANTHROPIC_API_KEY`

**Production (Additional):**
- `JWT_SECRET` (must be strong)
- `QDRANT_API_KEY` (if using cloud Qdrant)
- `VOYAGE_API_KEY` or `OPENAI_API_KEY` (for embeddings)

### Default Values Summary

| Category | Variable | Default |
|----------|----------|---------|
| Database | `MONGODB_DATABASE` | `hyper_db` |
| Database | `QDRANT_URL` | `http://localhost:7333` |
| Database | `QDRANT_CODE_COLLECTION` | `code_index` |
| Database | `QDRANT_KNOWLEDGE_COLLECTION` | `hyper-knowledge-base` |
| Embedding | `EMBEDDING` | `local` |
| Embedding | `TEI_URL` | `http://embedding-service:8080` |
| Embedding | `OLLAMA_URL` | `http://localhost:7335` |
| Embedding | `OLLAMA_MODEL` | `nomic-embed-text` |
| Embedding | `VOYAGE_MODEL` | `voyage-3` |
| Embedding | `OPENAI_MODEL` | `text-embedding-3-small` |
| AI | `ANTHROPIC_MODEL` | `claude-sonnet-4-5` |
| AI | `OPENAI_CHAT_MODEL` | `gpt-4-turbo` |
| Code | `ENABLE_FILE_WATCHER` | `false` |
| Code | `CODE_INDEX_AUTO_RECREATE` | `false` |
| Code | `KB_DOCS_PATH` | `.hyper/kb` |
| App | `HTTP_PORT` | `7095` |
| App | `GIN_MODE` | `debug` |
| App | `LOG_LEVEL` | `info` |
| App | `LOG_FORMAT` | `json` |
| App | `DEV_MODE` | `false` |
| App | `RATE_LIMIT_ENABLED` | `true` |
| App | `RATE_LIMIT_REQUESTS` | `100` |
| App | `RATE_LIMIT_WINDOW` | `60` |

---

## Security Best Practices

### 1. Never Commit Secrets

```bash
# ✅ GOOD - .gitignore
.env
.env.*
!.env.example
*.key
*.pem
```

### 2. Use Strong Secrets

```bash
# ❌ BAD
JWT_SECRET=secret

# ✅ GOOD
JWT_SECRET=$(openssl rand -base64 32)
```

### 3. Rotate Credentials Regularly

- API keys: Every 90 days
- JWT secrets: Every 180 days
- Database passwords: Every 365 days

### 4. Use Environment-Specific Files

```bash
# Development
.env.development

# Staging
.env.staging

# Production
.env.production
```

### 5. Kubernetes Secrets

```bash
# Create from file
kubectl create secret generic hyperion-secrets \
  --from-env-file=.env.production \
  --namespace=hyperion-prod

# Or from literals
kubectl create secret generic hyperion-secrets \
  --from-literal=JWT_SECRET=$(openssl rand -base64 32) \
  --namespace=hyperion-prod
```

---

## Troubleshooting

### Issue: "Cannot connect to MongoDB"

**Check:**
```bash
echo $MONGODB_URI
mongosh "$MONGODB_URI"
```

**Solution:** Verify URI format, credentials, and network connectivity

### Issue: "Qdrant dimension mismatch"

**Check:**
```bash
echo $EMBEDDING
echo $CODE_INDEX_AUTO_RECREATE
```

**Solution:** Set `CODE_INDEX_AUTO_RECREATE=true` (dev only) or manually reindex

### Issue: "Embedding generation timeout"

**Check:**
```bash
echo $TEI_URL
curl $TEI_URL/health
```

**Solution:** Verify embedding service is running and accessible

### Issue: "Rate limit exceeded"

**Check:**
```bash
echo $RATE_LIMIT_ENABLED
echo $RATE_LIMIT_REQUESTS
echo $RATE_LIMIT_WINDOW
```

**Solution:** Increase limits or disable for development

---

## Related Documents

- [MongoDB Integration](./mongodb-integration.md) - Database setup
- [Qdrant Integration](./qdrant-integration.md) - Vector database setup
- [Component Architecture](./component-architecture.md) - System overview
- [API Service Layer](./api-service-layer.md) - API endpoints

## Configuration Checklist

### Development Setup
- [ ] MongoDB running locally or accessible
- [ ] Qdrant running locally or accessible
- [ ] TEI/Ollama running (or use cloud embedding)
- [ ] `ANTHROPIC_API_KEY` set
- [ ] `MONGODB_URI` configured
- [ ] `DEV_MODE=true` (optional, bypasses JWT)

### Production Deployment
- [ ] Strong `JWT_SECRET` generated
- [ ] MongoDB Atlas or production database configured
- [ ] Qdrant cloud or production instance configured
- [ ] Cloud embedding provider configured (Voyage/OpenAI)
- [ ] `GIN_MODE=release`
- [ ] `LOG_FORMAT=json`
- [ ] Rate limiting enabled
- [ ] File watcher disabled
- [ ] All secrets stored in Kubernetes Secrets or secret manager
- [ ] Environment variables validated
- [ ] Health checks configured
- [ ] Monitoring and alerting set up
