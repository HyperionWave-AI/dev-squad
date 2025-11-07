# Docker Deployment Options

## Configuration Modes

### 1. Remote Qdrant (Cloud) + Local MongoDB (Default)
Best for: Development with Qdrant Cloud

```bash
# Configure .env
export QDRANT_URL=https://your-cluster.qdrant.io:6333
export QDRANT_API_KEY=your-api-key

# Start services (MongoDB local, Qdrant remote)
docker-compose up -d
```

**Services started:**
- ✅ MongoDB (local container)
- ✅ MCP Server (connects to Qdrant Cloud)
- ✅ HTTP Bridge (connects to Qdrant Cloud)
- ✅ UI
- ❌ Qdrant (skipped, using cloud)

---

### 2. Full Local Development (MongoDB + Qdrant)
Best for: Complete offline development

```bash
# Start with local Qdrant
docker-compose -f docker-compose.yml -f docker-compose.local.yml up -d

# OR use profile
docker-compose --profile local-qdrant up -d
```

**Services started:**
- ✅ MongoDB (local container)
- ✅ Qdrant (local container)
- ✅ MCP Server (connects to local Qdrant)
- ✅ HTTP Bridge (connects to local Qdrant)
- ✅ UI

---

### 3. Full Remote (MongoDB Atlas + Qdrant Cloud)
Best for: Production deployment

```bash
# Configure .env
export MONGODB_URI=mongodb+srv://user:pass@cluster.mongodb.net/
export MONGODB_DATABASE=coordinator_db_prod
export QDRANT_URL=https://your-cluster.qdrant.io:6333
export QDRANT_API_KEY=your-api-key

# Start services (no databases)
docker-compose up -d
```

**Services started:**
- ❌ MongoDB (using Atlas)
- ❌ Qdrant (using cloud)
- ✅ MCP Server (connects to remote)
- ✅ HTTP Bridge (connects to remote)
- ✅ UI

---

## Environment Variables

| Variable | Default | Override Example |
|----------|---------|------------------|
| `MONGODB_URI` | `mongodb://admin:admin123@mongodb:27017/...` | `mongodb+srv://user:pass@cluster.mongodb.net/` |
| `MONGODB_DATABASE` | `coordinator_db` | `coordinator_db_prod` |
| `QDRANT_URL` | `http://qdrant:6333` | `https://cluster-id.qdrant.io:6333` |
| `OPENAI_API_KEY` | (required) | `sk-proj-...` |
| `LOG_LEVEL` | `info` | `debug` |

---

## Quick Commands

```bash
# Start with current .env configuration
docker-compose up -d

# Start with local Qdrant (ignore .env QDRANT_URL)
docker-compose -f docker-compose.yml -f docker-compose.local.yml up -d

# Stop all services
docker-compose down

# Stop and remove volumes (WARNING: deletes data)
docker-compose down -v

# View logs
docker-compose logs -f hyperion-mcp-server
docker-compose logs -f hyperion-http-bridge

# Check service health
docker-compose ps
```

---

## Ports

| Service | Port | Access |
|---------|------|--------|
| MongoDB | 27017 | `mongodb://localhost:27017` |
| Qdrant HTTP | 6333 | http://localhost:6333 |
| Qdrant gRPC | 6334 | `localhost:6334` |
| MCP Server | 7778 | http://localhost:7778 |
| HTTP Bridge | 7095 | http://localhost:7095 |
| UI | 5173 | http://localhost:5173 |
