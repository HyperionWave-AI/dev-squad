# Development Setup with Hot-Reload

This guide explains how to run the Hyperion Coordinator with hot-reload for rapid development.

## Prerequisites

- Docker and Docker Compose installed
- Go 1.23+ (for local development without Docker)
- Node.js 20+ (for local UI development)

## Quick Start (Recommended)

Start all services with hot-reload using the development compose file:

```bash
# Start all services with hot-reload
docker-compose -f docker-compose.dev.yml up

# Or start specific services
docker-compose -f docker-compose.dev.yml up hyperion-mcp-server
docker-compose -f docker-compose.dev.yml up hyperion-http-bridge
docker-compose -f docker-compose.dev.yml up hyperion-ui
```

## What's Included

### MCP Server (Go with Air)
- **Hot-reload**: Code changes automatically rebuild and restart the server
- **Port**: 7778 (HTTP mode)
- **Volumes**: `./coordinator/mcp-server` mounted to `/app`
- **Config**: `.air.toml` for Air configuration

### HTTP Bridge (Go with Air)
- **Hot-reload**: Code changes automatically rebuild and restart the bridge
- **Port**: 8095
- **Volumes**: `./coordinator/mcp-http-bridge` mounted to `/app`
- **Config**: `.air.toml` for Air configuration

### React UI (Vite)
- **Hot-reload**: Built-in Vite HMR (Hot Module Replacement)
- **Port**: 5173
- **Volumes**: `./coordinator/ui` mounted to `/app`
- **Fast**: Instant updates without full page refresh

## Local Development (Without Docker)

### MCP Server

```bash
cd coordinator/mcp-server

# Install Air (first time only)
go install github.com/air-verse/air@latest

# Run with Air
air

# Or run directly
go run main.go
```

### HTTP Bridge

```bash
cd coordinator/mcp-http-bridge

# Install Air (first time only)
go install github.com/air-verse/air@latest

# Run with Air
air

# Or run directly
go run main.go
```

### React UI

```bash
cd coordinator/ui

# Install dependencies (first time only)
npm install

# Run dev server with hot-reload
npm run dev

# Open browser to http://localhost:5173
```

## Environment Variables

### MCP Server
```bash
TRANSPORT_MODE=http          # Use HTTP instead of stdio
MCP_PORT=7778               # Port for MCP server
MONGODB_URI=mongodb+srv://... # MongoDB connection string
MONGODB_DATABASE=coordinator_db
LOG_LEVEL=debug             # Verbose logging for development
```

### HTTP Bridge
```bash
PORT=8095                   # HTTP bridge port
MONGODB_URI=mongodb+srv://... # MongoDB connection string
MONGODB_DATABASE=coordinator_db
LOG_LEVEL=debug
```

### React UI
```bash
VITE_MCP_BRIDGE_URL=http://localhost:8095  # HTTP bridge URL
```

## Air Configuration

Air watches for file changes and automatically rebuilds your Go applications. Configuration is in `.air.toml`:

### Key Settings
- **Watched extensions**: `.go`, `.tpl`, `.tmpl`, `.html`
- **Excluded**: `_test.go`, `tmp/`, `vendor/`
- **Build delay**: 1000ms (prevents rapid rebuilds)
- **Build command**: `go build -o ./tmp/main .`

### Customizing Air

Edit `.air.toml` in the respective service directory:

```toml
[build]
  cmd = "go build -o ./tmp/main ."
  delay = 1000
  exclude_regex = ["_test.go"]
  include_ext = ["go", "tpl", "tmpl", "html"]
```

## Development Workflow

### 1. Start Services
```bash
docker-compose -f docker-compose.dev.yml up
```

### 2. Make Code Changes
Edit any `.go` file in:
- `coordinator/mcp-server/`
- `coordinator/mcp-http-bridge/`

Or any React file in:
- `coordinator/ui/src/`

### 3. See Changes Immediately
- **Go services**: Air detects changes → rebuilds → restarts (1-2 seconds)
- **React UI**: Vite HMR updates browser instantly (<100ms)

### 4. Check Logs
```bash
# View all logs
docker-compose -f docker-compose.dev.yml logs -f

# View specific service
docker-compose -f docker-compose.dev.yml logs -f hyperion-mcp-server
docker-compose -f docker-compose.dev.yml logs -f hyperion-http-bridge
docker-compose -f docker-compose.dev.yml logs -f hyperion-ui
```

### 5. Rebuild After Dependency Changes
```bash
# If you change go.mod or package.json
docker-compose -f docker-compose.dev.yml build

# Then restart
docker-compose -f docker-compose.dev.yml up
```

## Troubleshooting

### Air not rebuilding

**Problem**: Code changes don't trigger rebuild

**Solution**:
```bash
# Check Air is watching files
docker-compose -f docker-compose.dev.yml logs hyperion-mcp-server | grep "watching"

# Verify volume mount
docker-compose -f docker-compose.dev.yml exec hyperion-mcp-server ls -la /app

# Force rebuild
docker-compose -f docker-compose.dev.yml restart hyperion-mcp-server
```

### Build errors not showing

**Problem**: Air swallows build errors

**Solution**:
```bash
# Check build-errors.log in tmp directory
docker-compose -f docker-compose.dev.yml exec hyperion-mcp-server cat tmp/build-errors.log

# Or build manually
docker-compose -f docker-compose.dev.yml exec hyperion-mcp-server go build -o ./tmp/main .
```

### Go module cache issues

**Problem**: Dependency changes not detected

**Solution**:
```bash
# Clear Go module cache and rebuild
docker-compose -f docker-compose.dev.yml down
docker volume rm hyperion_mcp-server-cache
docker-compose -f docker-compose.dev.yml build --no-cache hyperion-mcp-server
docker-compose -f docker-compose.dev.yml up
```

### UI hot-reload not working

**Problem**: Browser doesn't update on file changes

**Solution**:
```bash
# Check Vite is running
docker-compose -f docker-compose.dev.yml logs hyperion-ui | grep "Local:"

# Ensure WebSocket connection
# Open browser console, look for WebSocket errors

# Restart UI service
docker-compose -f docker-compose.dev.yml restart hyperion-ui
```

### Port conflicts

**Problem**: "Port already in use" error

**Solution**:
```bash
# Check what's using the port
lsof -i :7778  # MCP server
lsof -i :8095  # HTTP bridge
lsof -i :5173  # React UI

# Kill the process or change port in docker-compose.dev.yml
```

## Performance Tips

### 1. Use File System Notifications (Linux/Mac)
Air uses native file system notifications for instant detection. No polling needed.

### 2. Exclude Large Directories
Update `.air.toml` to exclude directories you don't need to watch:

```toml
[build]
  exclude_dir = ["assets", "tmp", "vendor", "testdata", "node_modules"]
```

### 3. Optimize Docker Volumes
Use named volumes for Go module cache to speed up rebuilds:

```yaml
volumes:
  - mcp-server-cache:/go/pkg/mod
```

### 4. Parallel Builds
Air builds are sequential. If you have multiple services, run them in separate terminals for parallel builds.

## Production Deployment

**IMPORTANT**: Development setup is for local development only. For production:

```bash
# Use production compose file (without hot-reload)
docker-compose up

# Or build production images
docker-compose build
```

Production images:
- Don't include Air (smaller size)
- Use multi-stage builds
- Run compiled binaries directly
- Don't mount source code volumes

## IDE Integration

### VS Code

Add to `.vscode/tasks.json`:

```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "Start Dev Environment",
      "type": "shell",
      "command": "docker-compose -f docker-compose.dev.yml up",
      "problemMatcher": []
    },
    {
      "label": "Stop Dev Environment",
      "type": "shell",
      "command": "docker-compose -f docker-compose.dev.yml down",
      "problemMatcher": []
    }
  ]
}
```

### GoLand / IntelliJ IDEA

1. Install Air plugin
2. Configure Run Configuration → Go Build → Before launch → Run Air
3. Set working directory to service root

## Further Reading

- [Air Documentation](https://github.com/air-verse/air)
- [Vite HMR API](https://vitejs.dev/guide/api-hmr.html)
- [Docker Compose Development Best Practices](https://docs.docker.com/compose/compose-file/develop/)
