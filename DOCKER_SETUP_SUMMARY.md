# Docker Setup Summary

## Architecture Overview

The Hyperion Coordinator runs as a multi-container Docker setup with three main components:

### Services

1. **hyperion-http-bridge** (Port 8095)
   - Combines MCP Server + HTTP Bridge in one container
   - Built from `coordinator/mcp-http-bridge/Dockerfile.combined`
   - Spawns MCP server as subprocess
   - Provides REST API for web clients
   - Handles CORS for UI access

2. **hyperion-ui** (Port 5173)
   - React + TypeScript dashboard
   - Built from `coordinator/ui/Dockerfile`
   - Nginx serves static files
   - Production-optimized build

## Quick Start

```bash
# Install and start
./install.sh
docker-compose up -d

# Access services
open http://localhost:5173              # UI Dashboard
curl http://localhost:8095/health       # API Health

# View logs
docker-compose logs -f
```

## Docker Compose Structure

```yaml
services:
  hyperion-http-bridge:
    ports: ["8095:8095"]
    healthcheck: wget http://localhost:8095/health

  hyperion-ui:
    ports: ["5173:80"]
    depends_on: hyperion-http-bridge (healthy)
```

## Build Process

### HTTP Bridge (Multi-Stage)
1. **Stage 1**: Build MCP Server (Go 1.25)
2. **Stage 2**: Build HTTP Bridge (Go 1.25)
3. **Stage 3**: Alpine runtime with both binaries

### UI (Multi-Stage)
1. **Stage 1**: Node.js build (`npm run build`)
2. **Stage 2**: Nginx runtime with static files

## CORS Configuration

The HTTP Bridge allows requests from:
- `http://localhost:5173` - Vite dev server
- `http://localhost` - Docker UI
- `http://hyperion-ui` - Internal Docker network

Location: `coordinator/mcp-http-bridge/main.go:362-375`

## Environment Variables

Edit `.env` to configure:

```bash
MONGODB_URI=mongodb+srv://...           # MongoDB connection
MONGODB_DATABASE=coordinator_db         # Database name
LOG_LEVEL=info                          # Logging level
VITE_MCP_BRIDGE_URL=http://localhost:8095  # UI API endpoint
```

## Port Mapping

| Service | Container Port | Host Port | Purpose |
|---------|---------------|-----------|---------|
| HTTP Bridge | 8095 | 8095 | REST API |
| UI | 80 | 5173 | Web Dashboard |

## Health Checks

Both containers have health checks:
- **HTTP Bridge**: `wget http://localhost:8095/health`
- **UI**: `wget http://localhost/health`

Status visible via: `docker-compose ps`

## Common Commands

```bash
# Start services
docker-compose up -d

# Stop services
docker-compose down

# Rebuild after code changes
docker-compose build
docker-compose up -d

# View logs
docker-compose logs -f hyperion-http-bridge
docker-compose logs -f hyperion-ui

# Check status
docker-compose ps

# Access container shell
docker-compose exec hyperion-http-bridge /bin/sh

# Test API
curl http://localhost:8095/api/mcp/tools
curl -X POST http://localhost:8095/api/mcp/tools/call \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: test-1" \
  -d '{"name":"coordinator_list_human_tasks","arguments":{}}'
```

## Troubleshooting

### CORS Errors
**Symptom**: Browser can't connect to API

**Solution**:
```bash
# Test CORS
curl -v -H "Origin: http://localhost:5173" \
  -H "Access-Control-Request-Method: POST" \
  -X OPTIONS http://localhost:8095/api/mcp/tools/call

# Should see: Access-Control-Allow-Origin: http://localhost:5173
```

### Port Conflicts
**Symptom**: Container won't start, port already in use

**Solution**:
```bash
# Check what's using the port
lsof -i :8095
lsof -i :5173

# Stop conflicting service or change port in docker-compose.yml
```

### Health Check Failing
**Symptom**: Container stays unhealthy

**Solution**:
```bash
# Check logs for startup errors
docker-compose logs hyperion-http-bridge

# Common issues:
# - MongoDB connection failed (check MONGODB_URI in .env)
# - MCP server binary not found (rebuild: docker-compose build)
```

### UI Can't Reach API
**Symptom**: UI loads but shows "No tasks"

**Solution**:
```bash
# Verify both containers are healthy
docker-compose ps

# Test API directly
curl http://localhost:8095/health

# Check browser console for CORS errors
# Rebuild if CORS config changed:
docker-compose build hyperion-http-bridge
docker-compose up -d
```

## Development Workflow

### Making Changes

1. **Backend (Go) Changes**:
```bash
# Edit files in coordinator/mcp-http-bridge/ or coordinator/mcp-server/
docker-compose build hyperion-http-bridge
docker-compose up -d hyperion-http-bridge
docker-compose logs -f hyperion-http-bridge
```

2. **Frontend (React) Changes**:
```bash
# Edit files in coordinator/ui/
docker-compose build hyperion-ui
docker-compose up -d hyperion-ui
docker-compose logs -f hyperion-ui
```

3. **Environment Changes**:
```bash
# Edit .env
docker-compose restart
```

### Local Development (Non-Docker)

For faster iteration:
```bash
cd coordinator
./start-coordinator.sh
```

This runs:
- HTTP Bridge: http://localhost:8095
- UI: http://localhost:5173 (Vite dev server with hot reload)

## File Structure

```
.
├── docker-compose.yml                  # Multi-service orchestration
├── .env                                # Environment configuration
├── install.sh                          # Automated setup script
├── coordinator/
│   ├── mcp-server/                     # MCP protocol server
│   │   ├── main.go
│   │   ├── handlers/
│   │   └── storage/
│   ├── mcp-http-bridge/               # HTTP ↔ MCP adapter
│   │   ├── main.go
│   │   ├── Dockerfile                 # Single-service build
│   │   └── Dockerfile.combined        # Multi-stage build (used)
│   └── ui/                            # React dashboard
│       ├── src/
│       ├── Dockerfile                 # Production build
│       └── nginx.conf                 # Nginx configuration
```

## Production Deployment

For production:

1. Update `.env` with production MongoDB URI
2. Configure CORS for production domains in `main.go`
3. Set up reverse proxy (nginx/Caddy) for HTTPS
4. Configure monitoring and log rotation
5. Set up automated backups for MongoDB

See `DOCKER.md` for full production deployment guide.

## Next Steps

- **View UI**: http://localhost:5173
- **Test API**: http://localhost:8095/health
- **Read Docs**: [DOCKER.md](./DOCKER.md)
- **MCP Tools**: [HYPERION_COORDINATOR_MCP_REFERENCE.md](./HYPERION_COORDINATOR_MCP_REFERENCE.md)

---

**Last Updated**: 2025-10-02
**Docker Compose Version**: 3.8 (v2 syntax)
**Go Version**: 1.25
**Node Version**: 20
