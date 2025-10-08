# Docker Installation Guide

> **Complete guide to installing and running Hyperion Coordinator MCP with Docker**

## Architecture Overview

The Docker deployment uses a **combined container approach**:
- **Container name:** `hyperion-http-bridge`
- **Contains:** Both the HTTP Bridge (port 7095) and the MCP Server binary (`/app/hyper-mcp`)
- **MCP Clients:** Connect via `docker exec -i hyperion-http-bridge /app/hyper-mcp`

This single container serves both HTTP requests (for the UI) and stdio MCP connections (for Claude Code and other MCP clients).

---

## Quick Start

```bash
# 1. Clone the repository
git clone <repository-url>
cd hyper-mcp

# 2. Run the installation script
./install.sh

# 3. Start all services
docker-compose up -d

# 4. Access services
# - UI Dashboard: http://localhost:5173
# - HTTP API: http://localhost:7095/health

# 5. View logs
docker-compose logs -f
```

That's it! All services are now running:
- **HTTP Bridge + MCP Server** (port 7095) - API backend
- **React UI** (port 5173) - Kanban dashboard

---

## Prerequisites

### Required
- **Docker Desktop** (macOS/Windows) or **Docker Engine** (Linux)
  - macOS: https://docs.docker.com/desktop/install/mac-install/
  - Windows: https://docs.docker.com/desktop/install/windows-install/
  - Linux: https://docs.docker.com/engine/install/
- **Docker Compose** (included with Docker Desktop)

### Optional
- **Claude Code** - For automatic MCP client configuration

---

## Installation Steps

### Step 1: Clone Repository

```bash
git clone <repository-url>
cd hyper-mcp
```

### Step 2: Run Installation Script

The `install.sh` script automates the entire setup:

```bash
./install.sh
```

**What it does:**
- ✅ Checks Docker and Docker Compose are installed
- ✅ Creates `.env` configuration from template
- ✅ Builds all Docker images (HTTP Bridge, UI)
- ✅ Detects Claude Code configuration location
- ✅ Automatically configures Claude Code MCP server (macOS/Linux)
- ✅ Provides manual configuration instructions if needed

**Expected output:**
```
═══════════════════════════════════════════════════════
  Hyperion Coordinator MCP - Docker Installation
═══════════════════════════════════════════════════════

ℹ Checking prerequisites...
✓ Docker is installed
✓ Docker Compose is installed

ℹ Creating .env file from template...
✓ .env file created (using default MongoDB Atlas dev cluster)

ℹ Building Hyperion Coordinator MCP Docker image...
✓ Docker image built successfully

ℹ Configuring Claude Code...
✓ Claude Code configured successfully
ℹ Restart Claude Code to load the MCP server

═══════════════════════════════════════════════════════
✓ Installation complete!
═══════════════════════════════════════════════════════
```

### Step 3: Start the Server

```bash
docker-compose up -d
```

**Flags explained:**
- `-d` - Detached mode (runs in background)

**What happens:**
- Docker builds all images (HTTP Bridge with MCP Server, React UI)
- HTTP Bridge starts and spawns MCP Server process
- Connects to MongoDB Atlas
- Registers all MCP tools and resources
- UI starts and serves on port 5173
- CORS configured for cross-origin requests

### Step 4: Verify Installation

**Check logs:**
```bash
# All services
docker-compose logs -f

# Just HTTP Bridge
docker-compose logs -f hyperion-http-bridge

# Just UI
docker-compose logs -f hyperion-ui
```

**Expected HTTP Bridge output:**
```
hyperion-http-bridge  | MCP server started with PID: 14
hyperion-http-bridge  | Starting Hyperion Coordinator MCP Server
hyperion-http-bridge  | Connecting to MongoDB Atlas database=coordinator_db
hyperion-http-bridge  | Successfully connected to MongoDB Atlas
hyperion-http-bridge  | Task storage initialized with MongoDB
hyperion-http-bridge  | Knowledge storage initialized with MongoDB
hyperion-http-bridge  | All handlers registered successfully tools=9 resources=2
hyperion-http-bridge  | MCP connection initialized successfully
hyperion-http-bridge  | HTTP bridge listening on port 7095
```

**Check container status:**
```bash
docker-compose ps
```

**Expected output:**
```
NAME                    STATUS                    PORTS
hyperion-http-bridge    Up 2 minutes (healthy)    0.0.0.0:7095->7095/tcp
hyperion-ui             Up 2 minutes (healthy)    0.0.0.0:5173->80/tcp
```

**Test services:**
```bash
# Test HTTP API
curl http://localhost:7095/health

# Test UI (should return HTML)
curl -I http://localhost:5173
```

### Step 5: Configure MCP Client

#### Automatic Configuration (Claude Code)

If you ran `./install.sh` on macOS or Linux, Claude Code should already be configured. Just **restart Claude Code**.

**Verify configuration:**
- macOS: Check `~/Library/Application Support/Claude/claude_desktop_config.json`
- Linux: Check `~/.config/Claude/claude_desktop_config.json`

#### Manual Configuration (Other MCP Clients)

Add this to your MCP client configuration:

```json
{
  "mcpServers": {
    "hyper": {
      "type": "stdio",
      "command": "/usr/local/bin/docker",
      "args": [
        "exec",
        "-i",
        "hyperion-http-bridge",
        "/app/hyper-mcp"
      ],
      "env": {}
    }
  }
}
```

**Note:** This connects to the running `hyperion-http-bridge` container. Ensure the container is running with `docker-compose up -d` before starting your MCP client.

---

## Configuration

### Environment Variables

Edit `.env` to customize configuration:

```bash
# MongoDB Connection (default uses dev cluster)
MONGODB_URI=mongodb+srv://dev:fvOKzv9enD8CSVwD@devdb.yqf8f8r.mongodb.net/?retryWrites=true&w=majority&appName=devDB

# Database name
MONGODB_DATABASE=coordinator_db

# Log level (debug, info, warn, error)
LOG_LEVEL=info
```

**After editing `.env`, restart the container:**
```bash
docker-compose restart
```

### Custom MongoDB

To use your own MongoDB instance:

1. **Edit `.env`:**
   ```bash
   MONGODB_URI=mongodb+srv://username:password@your-cluster.mongodb.net/?retryWrites=true&w=majority
   ```

2. **Restart:**
   ```bash
   docker-compose restart
   ```

3. **Verify connection:**
   ```bash
   docker-compose logs hyperion-http-bridge | grep MongoDB
   ```

---

## Usage

### Start Server
```bash
docker-compose up -d
```

### Stop Server
```bash
docker-compose down
```

### Restart Server
```bash
docker-compose restart
```

### View Logs
```bash
# Follow all logs in real-time
docker-compose logs -f

# Follow specific service logs
docker-compose logs -f hyperion-http-bridge
docker-compose logs -f hyperion-ui

# View last 100 lines
docker-compose logs --tail=100 hyperion-http-bridge

# View logs since 10 minutes ago
docker-compose logs --since=10m
```

### Check Status
```bash
docker-compose ps
```

### Access Container Shell
```bash
# HTTP Bridge container
docker-compose exec hyperion-http-bridge /bin/sh

# UI container
docker-compose exec hyperion-ui /bin/sh
```

### Rebuild Image
```bash
# After code changes
docker-compose build

# Or rebuild without cache
docker-compose build --no-cache
```

---

## Testing with Docker

### Test HTTP API

```bash
# Test health endpoint
curl http://localhost:7095/health

# List available tools
curl http://localhost:7095/api/mcp/tools

# Call a tool
curl -X POST http://localhost:7095/api/mcp/tools/call \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: test-1" \
  -d '{
    "name": "coordinator_list_human_tasks",
    "arguments": {}
  }'

# List resources
curl http://localhost:7095/api/mcp/resources

# Test CORS
curl -v -H "Origin: http://localhost:5173" \
  -H "Access-Control-Request-Method: POST" \
  -X OPTIONS http://localhost:7095/api/mcp/tools/call
```

### Run Integration Tests

```bash
# From the test directory
cd coordinator/mcp-server
go test -v
```

---

## Troubleshooting

### Container Won't Start

**Check logs:**
```bash
docker-compose logs hyperion-http-bridge
docker-compose logs hyperion-ui
```

**Common issues:**
- MongoDB connection failed → Check `MONGODB_URI` in `.env`
- Port 7095 already in use → Stop other services using the port
- Port 5173 already in use → Stop other services or change port mapping
- Image build failed → Run `docker-compose build --no-cache`
- Health check failing → Check logs for startup errors

### MongoDB Connection Errors

**Verify environment variables:**
```bash
docker-compose exec hyperion-http-bridge env | grep MONGODB
```

**Test MongoDB connection:**
```bash
# Check if MongoDB Atlas is accessible
curl -v https://devdb.yqf8f8r.mongodb.net/
```

**Common fixes:**
- Verify MongoDB URI is correct in `.env`
- Check MongoDB Atlas network access (whitelist your IP)
- Ensure MongoDB Atlas cluster is running

### Claude Code Not Detecting Server

1. **Verify Docker is running:**
   ```bash
   docker ps
   ```

2. **Check Claude Code config:**
   ```bash
   # macOS
   cat ~/Library/Application\ Support/Claude/claude_desktop_config.json

   # Linux
   cat ~/.config/Claude/claude_desktop_config.json
   ```

3. **Verify paths are absolute:**
   - The `-f` argument must point to the full path of `docker-compose.yml`

4. **Restart Claude Code completely:**
   - Quit Claude Code (not just close window)
   - Reopen Claude Code

5. **Check Claude Code logs:**
   - macOS: `~/Library/Logs/Claude/`
   - Linux: `~/.config/Claude/logs/`

### Permission Errors

```bash
# Make install script executable
chmod +x install.sh

# Reset Docker volumes
docker-compose down -v
docker-compose up -d
```

### Image Build Fails

```bash
# Clear Docker cache
docker system prune -a

# Rebuild from scratch
docker-compose build --no-cache
```

### CORS Errors in Browser

**Symptoms:**
- UI can't connect to API
- Browser console shows CORS errors
- OPTIONS preflight requests failing

**Solutions:**

1. **Verify CORS configuration:**
```bash
# Test CORS headers
curl -v -H "Origin: http://localhost:5173" \
  -H "Access-Control-Request-Method: POST" \
  -X OPTIONS http://localhost:7095/api/mcp/tools/call
```

2. **Check allowed origins in main.go:**
```go
// Should include:
"http://localhost:5173",  // UI origin
"http://localhost",       // Docker UI
"http://hyperion-ui",     // Docker internal
```

3. **Rebuild after CORS changes:**
```bash
docker-compose build hyperion-http-bridge
docker-compose up -d
```

### Container Stops Immediately

**Check for startup errors:**
```bash
docker-compose logs hyperion-http-bridge
docker-compose logs hyperion-ui
```

**Verify Dockerfile:**
```bash
# Test HTTP Bridge build manually
docker build -f coordinator/mcp-http-bridge/Dockerfile.combined \
  -t hyperion-test \
  coordinator/
docker run --rm -it hyperion-test

# Test UI build manually
docker build -t hyperion-ui-test coordinator/ui/
docker run --rm -p 8080:80 hyperion-ui-test
```

---

## Advanced Usage

### Custom Docker Compose

Create `docker-compose.override.yml` for local customizations:

```yaml
version: '3.8'

services:
  hyper-mcp:
    environment:
      - LOG_LEVEL=debug
    volumes:
      - ./logs:/app/logs
```

**Apply changes:**
```bash
docker-compose up -d
```

### Multi-Container Setup

Add additional services to `docker-compose.yml`:

```yaml
services:
  hyper-mcp:
    # ... existing config ...

  mongodb:
    image: mongo:7
    ports:
      - "27017:27017"
    volumes:
      - mongodb_data:/data/db

  qdrant:
    image: qdrant/qdrant:latest
    ports:
      - "6333:6333"
    volumes:
      - qdrant_storage:/qdrant/storage

volumes:
  mongodb_data:
  qdrant_storage:
```

### Health Checks

Monitor container health:

```bash
# Check health status
docker inspect hyper-mcp | grep -A 10 Health

# Auto-restart on failure (already configured)
# See docker-compose.yml: restart: unless-stopped
```

---

## Production Deployment

### Deployment Checklist

- [ ] Set production MongoDB URI in `.env`
- [ ] Change default credentials/keys
- [ ] Configure log rotation
- [ ] Set up monitoring (Prometheus, Grafana)
- [ ] Configure backups for MongoDB
- [ ] Set up reverse proxy for HTTPS
- [ ] Implement rate limiting
- [ ] Configure firewall rules

### Docker Compose Production

```yaml
version: '3.8'

services:
  hyper-mcp:
    build:
      context: ./coordinator/mcp-server
      dockerfile: Dockerfile
    container_name: hyper-mcp-prod
    environment:
      - MONGODB_URI=${MONGODB_URI}
      - MONGODB_DATABASE=${MONGODB_DATABASE}
      - LOG_LEVEL=warn
    restart: always
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
    networks:
      - hyperion-prod-network
    healthcheck:
      test: ["CMD", "pgrep", "-f", "hyper-mcp"]
      interval: 30s
      timeout: 10s
      retries: 3

networks:
  hyperion-prod-network:
    driver: bridge
```

### Monitoring

**View resource usage:**
```bash
docker stats hyper-mcp
```

**Export logs:**
```bash
docker-compose logs --no-color hyper-mcp > logs.txt
```

---

## Cleanup

### Remove Container & Image
```bash
# Stop and remove container
docker-compose down

# Remove image
docker rmi hyper-mcp
```

### Full Cleanup
```bash
# Stop containers
docker-compose down -v

# Remove images
docker rmi $(docker images -q hyper-mcp)

# Prune system (careful!)
docker system prune -a --volumes
```

---

## FAQ

### Q: Can I run this on Windows?

**A:** Yes! Docker Desktop for Windows supports both Windows and Linux containers. Follow the same installation steps.

### Q: Do I need MongoDB Atlas?

**A:** The default configuration uses MongoDB Atlas dev cluster. You can use your own MongoDB by editing `.env`.

### Q: How do I update to the latest version?

**A:**
```bash
git pull origin main
docker-compose build
docker-compose up -d
```

### Q: Can I run multiple instances?

**A:** Yes, but you'll need to modify `docker-compose.yml` to use different container names and configurations.

### Q: How do I backup data?

**A:** Data is stored in MongoDB Atlas. Use MongoDB's built-in backup features or export data via the MCP tools.

### Q: What's the performance impact?

**A:** Docker adds minimal overhead (<5%) for CPU/memory. The MCP server is lightweight and designed for concurrent requests.

---

## Support

- **Documentation**: See main [README.md](./README.md)
- **MCP Reference**: [HYPERION_COORDINATOR_MCP_REFERENCE.md](./HYPERION_COORDINATOR_MCP_REFERENCE.md)
- **Issues**: GitHub Issues

---

**Built with ❤️ for seamless AI agent coordination**
