# Hyperion Unified Coordinator - Deployment Guide

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                   Unified Coordinator                        │
│                  (Single Go Binary)                          │
│                                                              │
│  Port 7095:                                                  │
│  ┌────────────────────────────────────────────────┐         │
│  │  REST API (/api/*)        UI Serving (/)       │         │
│  │                                                 │         │
│  │  - Tasks & TODOs          - React SPA          │         │
│  │  - Code Index             - Static Assets      │         │
│  │  - Knowledge Base         - HTML/CSS/JS        │         │
│  └────────────────────────────────────────────────┘         │
│                                                              │
│  Storage Layer:                                              │
│  ┌────────────────────────────────────────────────┐         │
│  │  TaskStorage  │  CodeIndexStorage  │  Qdrant   │         │
│  │               │                    │           │         │
│  │  MongoDB      │  MongoDB           │  Vectors  │         │
│  └────────────────────────────────────────────────┘         │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
        ┌───────────────────────────────────┐
        │  External Services                │
        │  - MongoDB (tasks, files, chunks) │
        │  - Qdrant (vector search)         │
        │  - OpenAI (embeddings)            │
        └───────────────────────────────────┘
```

**Key Principles:**
- UI → REST API → Storage Layer (NO MCP proxying)
- Single service, single deployment
- Clean data contracts and interfaces
- Direct storage access for all operations

---

## Quick Start (Local Development)

### Prerequisites
- Go 1.25+
- Node.js 18+
- Docker & Docker Compose
- MongoDB (local or Atlas)

### Option 1: Quick Start Script (Recommended)

```bash
cd /Users/maxmednikov/MaxSpace/dev-squad/coordinator

# Build everything and start
make quick-start

# Or use the script directly
./start-coordinator.sh
```

**What it does:**
1. Builds coordinator binary (if not exists)
2. Builds UI (if not exists)
3. Starts coordinator on port 7095
4. Serves REST API at `/api/*`
5. Serves UI at `/`

**Access:**
- UI: http://localhost:7095
- REST API: http://localhost:7095/api/tasks
- Health: http://localhost:7095/health

### Option 2: Manual Build & Run

```bash
# Build UI
cd ui
npm install
npm run build
cd ..

# Build coordinator
go build -o bin/coordinator ./cmd/coordinator

# Set environment variables
export MONGODB_URI="mongodb://admin:admin123@localhost:27017/coordinator_db?authSource=admin"
export QDRANT_URL="http://localhost:6333"
export OPENAI_API_KEY="your-api-key"

# Run
./bin/coordinator --mode=http
```

### Option 3: Docker Compose (Full Stack)

```bash
# Start all services (coordinator + MongoDB + Qdrant + embeddings)
make docker-up

# Or manually
docker-compose up -d

# View logs
make docker-logs

# Stop services
make docker-down
```

---

## Environment Variables

### Required
| Variable | Description | Example |
|----------|-------------|---------|
| `MONGODB_URI` | MongoDB connection string | `mongodb://admin:admin123@mongodb:27017/coordinator_db?authSource=admin` |
| `OPENAI_API_KEY` | OpenAI API key for embeddings | `sk-...` |

### Optional
| Variable | Description | Default |
|----------|-------------|---------|
| `HTTP_PORT` | REST API + UI port | `7095` |
| `MONGODB_DATABASE` | Database name | `coordinator_db` |
| `QDRANT_URL` | Qdrant server URL | `http://qdrant:6333` |
| `QDRANT_API_KEY` | Qdrant API key (if cloud) | `""` |
| `LOG_LEVEL` | Logging level | `info` |
| `CODE_INDEX_FOLDERS` | Auto-index folders on startup | `""` |
| `CODE_INDEX_AUTO_SCAN` | Auto-scan on start | `false` |

### Environment File

Create `.env` in project root:

```bash
# MongoDB (Local)
MONGODB_URI=mongodb://admin:admin123@localhost:27017/coordinator_db?authSource=admin
MONGODB_DATABASE=coordinator_db

# MongoDB (Atlas Cloud)
# MONGODB_URI=mongodb+srv://user:pass@cluster.mongodb.net/coordinator_db?retryWrites=true&w=majority

# Qdrant (Local)
QDRANT_URL=http://localhost:6333

# Qdrant (Cloud)
# QDRANT_URL=https://your-cluster.qdrant.io
# QDRANT_API_KEY=your-api-key

# OpenAI
OPENAI_API_KEY=sk-your-openai-api-key

# Server
HTTP_PORT=7095
LOG_LEVEL=info

# Code Index (Optional)
CODE_INDEX_FOLDERS=/path/to/code1,/path/to/code2
CODE_INDEX_AUTO_SCAN=false
```

---

## Production Deployment

### Docker (Recommended)

**1. Build production image:**
```bash
docker build -t hyperion-coordinator:latest -f Dockerfile .
```

**2. Push to registry:**
```bash
# Google Container Registry
docker tag hyperion-coordinator:latest us-central1-docker.pkg.dev/production-471918/hyperion-public/hyperion-coordinator:latest
docker push us-central1-docker.pkg.dev/production-471918/hyperion-public/hyperion-coordinator:latest
```

**3. Deploy with docker-compose:**
```bash
# Production stack (uses cloud MongoDB + Qdrant)
docker-compose up -d

# With local Qdrant
docker-compose --profile local-qdrant up -d
```

### Kubernetes

**1. Create secrets:**
```bash
kubectl create secret generic hyperion-coordinator-secrets \
  --from-literal=mongodb-uri="mongodb+srv://..." \
  --from-literal=openai-api-key="sk-..." \
  --from-literal=qdrant-api-key="your-key"
```

**2. Deploy coordinator:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hyperion-coordinator
spec:
  replicas: 3
  selector:
    matchLabels:
      app: hyperion-coordinator
  template:
    metadata:
      labels:
        app: hyperion-coordinator
    spec:
      containers:
      - name: coordinator
        image: us-central1-docker.pkg.dev/production-471918/hyperion-public/hyperion-coordinator:latest
        ports:
        - containerPort: 7095
          name: http
        env:
        - name: MONGODB_URI
          valueFrom:
            secretKeyRef:
              name: hyperion-coordinator-secrets
              key: mongodb-uri
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: hyperion-coordinator-secrets
              key: openai-api-key
        - name: QDRANT_URL
          value: "https://your-cluster.qdrant.io"
        - name: QDRANT_API_KEY
          valueFrom:
            secretKeyRef:
              name: hyperion-coordinator-secrets
              key: qdrant-api-key
        - name: HTTP_PORT
          value: "7095"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 7095
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 7095
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: hyperion-coordinator
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 7095
    name: http
  selector:
    app: hyperion-coordinator
```

**3. Apply:**
```bash
kubectl apply -f k8s/coordinator-deployment.yaml
```

---

## REST API Usage

### Tasks API

**Create task:**
```bash
curl -X POST http://localhost:7095/api/tasks \
  -H 'Content-Type: application/json' \
  -d '{
    "prompt": "Implement CSV export feature",
    "metadata": {
      "priority": "high",
      "assignee": "backend-team"
    }
  }'
```

**List tasks:**
```bash
curl http://localhost:7095/api/tasks
```

**Get task details:**
```bash
curl http://localhost:7095/api/tasks/{taskId}
```

### Code Index API

**Add folder:**
```bash
curl -X POST http://localhost:7095/api/code-index/add-folder \
  -H 'Content-Type: application/json' \
  -d '{
    "folderPath": "/path/to/code",
    "description": "Main application code"
  }'
```

**Scan folder:**
```bash
curl -X POST http://localhost:7095/api/code-index/scan \
  -H 'Content-Type: application/json' \
  -d '{
    "folderPath": "/path/to/code"
  }'
```

**Search code:**
```bash
curl -X POST http://localhost:7095/api/code-index/search \
  -H 'Content-Type: application/json' \
  -d '{
    "query": "JWT authentication middleware",
    "limit": 10,
    "minScore": 0.7,
    "fileTypes": ["go", "ts"],
    "retrieve": "chunk"
  }'
```

**Remove folder:**
```bash
curl -X DELETE http://localhost:7095/api/code-index/remove-folder/{configId}
```

**Get status:**
```bash
curl http://localhost:7095/api/code-index/status
```

### Knowledge Base API

**Query knowledge:**
```bash
curl -X POST http://localhost:7095/api/knowledge/query \
  -H 'Content-Type: application/json' \
  -d '{
    "query": "how to implement streaming CSV export",
    "limit": 5
  }'
```

---

## Monitoring & Health Checks

### Health Endpoint
```bash
curl http://localhost:7095/health
```

**Response (healthy):**
```json
{
  "status": "healthy",
  "timestamp": "2025-10-10T12:00:00Z",
  "services": {
    "mongodb": "connected",
    "qdrant": "connected"
  }
}
```

### Logs

**Development:**
```bash
# Coordinator logs to stdout
./bin/coordinator --mode=http
```

**Docker Compose:**
```bash
make docker-logs

# Or
docker-compose logs -f hyperion-coordinator
```

**Kubernetes:**
```bash
kubectl logs -f deployment/hyperion-coordinator
kubectl logs -f deployment/hyperion-coordinator --tail=100
```

---

## Troubleshooting

### UI not loading
- **Check binary built with UI:** `ls -la ui/dist`
- **Rebuild UI:** `cd ui && npm run build`
- **Check logs:** Look for "Serving UI from ui/dist"

### MongoDB connection failed
- **Verify connection string:** Check MONGODB_URI format
- **Test connection:** `mongosh "$MONGODB_URI"`
- **Check network:** Ensure MongoDB is accessible from coordinator

### Qdrant not responding
- **Check Qdrant URL:** Verify QDRANT_URL is correct
- **Test connection:** `curl http://localhost:6333/health`
- **Start Qdrant:** `docker run -p 6333:6333 qdrant/qdrant:latest`

### Code search returns no results
- **Check folders added:** `curl http://localhost:7095/api/code-index/status`
- **Trigger scan:** `curl -X POST http://localhost:7095/api/code-index/scan -d '{"folderPath":"/path"}'`
- **Check embeddings:** Verify OPENAI_API_KEY is set

### Build errors
- **Go version:** Ensure Go 1.25+ (`go version`)
- **Clean build:** `make clean && make build`
- **Dependencies:** `go mod download && cd ui && npm install`

---

## Development Workflow

### Hot-reload Development

**Backend (Air):**
```bash
cd cmd/coordinator
air  # Hot-reload on Go file changes
```

**Frontend (Vite):**
```bash
cd ui
npm run dev  # Dev server on port 5173
```

**Full stack with Docker:**
```bash
make dev
# Starts both services with hot-reload
```

### Running Tests

```bash
# All tests
make test

# Go tests only
make test-go

# UI tests only
make test-ui
```

### Code Quality

```bash
# Lint
make lint

# Format
make format
```

---

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make build` | Build coordinator binary and UI |
| `make run` | Run coordinator in production mode |
| `make dev` | Run with hot-reload (Docker Compose) |
| `make test` | Run all tests |
| `make clean` | Remove build artifacts |
| `make docker-build` | Build Docker images |
| `make docker-up` | Start Docker Compose services |
| `make docker-down` | Stop Docker Compose services |
| `make docker-logs` | View Docker Compose logs |
| `make quick-start` | Build and start coordinator |
| `make status` | Show service status |

---

## Migration from Old Architecture

### Old Architecture (Deprecated)
```
UI → MCP HTTP Bridge (7095) → MCP Server (stdio) → Storage
```

### New Architecture (Current)
```
UI → REST API (7095) → Storage Layer
```

### Breaking Changes

1. **No MCP proxying:** REST API uses storage layer directly
2. **Unified service:** Single binary replaces mcp-server + mcp-http-bridge
3. **Clean contracts:** REST API has its own DTOs, not MCP results
4. **UI changes:** All UI code uses REST clients, not MCP client

### Migration Steps

1. **Stop old services:**
   ```bash
   docker-compose -f docker-compose.old.yml down
   ```

2. **Update environment variables:**
   - Remove MCP-related vars
   - Keep storage configs (MongoDB, Qdrant)

3. **Deploy new coordinator:**
   ```bash
   docker-compose up -d
   ```

4. **Verify health:**
   ```bash
   curl http://localhost:7095/health
   ```

---

## Security

### Authentication
- JWT tokens required for all REST endpoints
- Generate test token: `./scripts/generate-test-jwt.js`
- Pass in Authorization header: `Bearer <token>`

### Best Practices
- Never commit `.env` files
- Use secrets management (Kubernetes Secrets, AWS Secrets Manager)
- Rotate API keys regularly
- Use HTTPS in production
- Enable MongoDB authentication
- Use Qdrant API keys in production

---

## Performance Tuning

### Code Index
- **Batch size:** Adjust chunk size in scanner (default: 500 lines)
- **Embedding cache:** Reuse embeddings for unchanged chunks
- **File filters:** Exclude large binary files, node_modules, vendor

### MongoDB
- **Indexes:** Ensure indexes on taskId, agentName, status
- **Connection pool:** Adjust maxPoolSize based on load
- **Query optimization:** Use projection to limit fields

### Qdrant
- **Vector dimensions:** 768 (nomic-embed-text-v1.5)
- **HNSW parameters:** Tune for speed vs accuracy
- **Quantization:** Enable scalar quantization for storage savings

---

## Support & Resources

- **Architecture:** See `ARCHITECTURE.md`
- **Code quality:** See `CLAUDE.md`
- **API reference:** See `API.md` (autogenerated from OpenAPI)
- **UI cleanup:** See `UI_MCP_CLEANUP_SUMMARY.md`

**Common Issues:**
- Check logs first: `make docker-logs`
- Verify environment: `make status`
- Test endpoints: See REST API examples above

---

**Version:** v1.0 Unified Coordinator
**Updated:** 2025-10-10
**Architecture:** UI → REST API → Storage Layer (NO MCP proxying)
