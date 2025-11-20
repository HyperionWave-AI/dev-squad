# Hyperion Deployment & Infrastructure Architecture

**Collection:** infrastructure
**Tags:** infrastructure, deployment, Docker, Kubernetes
**Technology:** Docker Compose, K8s, GKE
**Version:** 1.0

---

HYPERION DEPLOYMENT & INFRASTRUCTURE ARCHITECTURE

DOCKER COMPOSE ARCHITECTURE (docker-compose.yml, lines 1-124):
Services:
1. hyperion-coordinator: Unified service combining REST API + MCP + UI
   - Port: 7095 (HTTP) or stdio (MCP mode)
   - Health check: GET /health endpoint
   - Depends on: MongoDB
   - Modes: http, mcp, or both

2. MongoDB 7.0 (internal service)
   - Default: mongodb://admin:admin123@mongodb:27017
   - Database: coordinator_db
   - Health check: mongosh ping
   - Volumes: mongodb_data, mongodb_config
   - No external ports (internal network only)

3. Qdrant Vector Database (optional profile: local-qdrant)
   - Image: qdrant/qdrant:latest
   - Internal port: 6333
   - Volumes: qdrant_data for persistence
   - Production: Use managed Qdrant Cloud instead

4. Embedding Service (Hugging Face TEI)
   - Image: ghcr.io/huggingface/text-embeddings-inference:cpu-latest
   - Model: nomic-ai/nomic-embed-text-v1.5
   - CPU-only, platform: linux/amd64
   - Health check: /health endpoint
   - No external port (internal communication)

Networks: hyperion-network (bridge driver)

BUILD & DEPLOYMENT:
Root Makefile (/Makefile, lines 1-227):
- make native: Build unified binary with embedded UI (build-native.sh)
- make install: Install Go + Node dependencies
- make dev: Backend hot-reload (Air)
- make dev-hot: Full-stack hot-reload (Vite + Go)
- make run: Execute compiled binary
- make run-stdio: MCP stdio mode for Claude Code
- make configure-native: Register with Claude Code (stdio)
- make run-mcp-http: HTTP mode on port 7095
- make test: Run all tests
- make clean: Remove artifacts (preserve node_modules)
- make clean-all: Deep clean with confirmation

Development Workflow:
- Hot reload via Air (.air.toml)
- Frontend dev: Vite on port 5173
- Backend: Go service with live recompilation
- Config: .env.hyper or .env.native

KUBERNETES DEPLOYMENT:
- GKE cluster (Google Cloud)
- dev namespace: Development environment
- prod namespace: Production environment
- CI/CD: GitHub Actions pipeline
- Deployment configs in docs/ directory

CONFIGURATION:
Environment variables (.env files):
- MONGODB_URI, MONGODB_DATABASE
- QDRANT_URL, QDRANT_API_KEY
- EMBEDDING: ollama|openai|voyage
- TEI_URL, OLLAMA_URL
- OPENAI_API_KEY, VOYAGE_API_KEY
- HTTP_PORT, LOG_LEVEL
- CODE_INDEX_FOLDERS, CODE_INDEX_AUTO_SCAN

HEALTHCHECKS:
- Coordinator: GET /health (10s interval, 3 retries)
- MongoDB: mongosh ping (10s interval)
- Qdrant: GET /health (10s interval)
- TEI: curl /health (60s startup grace, 10s interval)
