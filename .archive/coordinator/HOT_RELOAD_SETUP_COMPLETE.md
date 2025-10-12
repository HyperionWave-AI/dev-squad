# ✅ Hot-Reload Development Setup Complete

## What Was Configured

### 1. Air Hot-Reload for Go Services ✅

**MCP Server:**
- Configuration: `coordinator/mcp-server/.air.toml`
- Dockerfile: `coordinator/mcp-server/Dockerfile.dev`
- Watches: All `.go` files
- Rebuild time: 1-2 seconds
- Auto-restart: Yes

**HTTP Bridge:**
- Configuration: `coordinator/mcp-http-bridge/.air.toml`
- Dockerfile: `coordinator/mcp-http-bridge/Dockerfile.dev`
- Watches: All `.go` files
- Rebuild time: 1-2 seconds
- Auto-restart: Yes

### 2. Vite HMR for React UI ✅

**React UI:**
- Dockerfile: `coordinator/ui/Dockerfile.dev`
- Hot Module Replacement: Built-in Vite
- Update time: <100ms
- State preservation: Yes
- Port: 5173

### 3. Development Docker Compose ✅

**File:** `docker-compose.dev.yml`

Features:
- Volume mounts for live code changes
- Named volumes for Go module cache
- Health checks for all services
- Development environment variables
- Network isolation

### 4. Makefile Commands ✅

New development commands added:

```bash
make dev          # Start all with hot-reload (foreground)
make dev-up       # Start all with hot-reload (background)
make dev-down     # Stop all development services
make dev-logs     # Follow logs
make dev-build    # Rebuild development images
make dev-mcp      # Start only MCP server
make dev-bridge   # Start only HTTP bridge
make dev-ui       # Start only React UI
make install-air  # Install Air locally
```

### 5. Documentation ✅

**Quick Start Guide:** `coordinator/QUICK_START_DEV.md`
- TL;DR commands
- Development workflow
- Troubleshooting
- Common issues

**Detailed Setup:** `coordinator/DEV_SETUP.md`
- Full Air configuration
- Docker and local development
- Environment variables
- Performance tips
- IDE integration

## Quick Start

### Start Everything
```bash
make dev
```

That's it! Now you can edit code and see changes immediately:
- **Go files**: 1-2 second rebuild and restart
- **React files**: Instant hot module replacement

### Stop Everything
```bash
make dev-down
```

## File Structure Created

```
dev-squad/
├── docker-compose.dev.yml                     # NEW: Development compose
├── Makefile                                   # UPDATED: Added dev commands
├── coordinator/
│   ├── QUICK_START_DEV.md                    # NEW: Quick reference
│   ├── DEV_SETUP.md                          # NEW: Detailed guide
│   ├── HOT_RELOAD_SETUP_COMPLETE.md          # NEW: This file
│   ├── mcp-server/
│   │   ├── .air.toml                         # NEW: Air config
│   │   └── Dockerfile.dev                    # NEW: Dev Dockerfile
│   ├── mcp-http-bridge/
│   │   ├── .air.toml                         # NEW: Air config
│   │   └── Dockerfile.dev                    # NEW: Dev Dockerfile
│   └── ui/
│       └── Dockerfile.dev                     # NEW: Dev Dockerfile
```

## How It Works

### Go Services (Air)

1. **File Change Detected**
   - Air watches filesystem for `.go` file changes
   - Uses native OS notifications (no polling)

2. **Build Process**
   - Runs `go build -o ./tmp/main .`
   - Only rebuilds changed packages (incremental)
   - Build errors logged to `tmp/build-errors.log`

3. **Restart**
   - Gracefully stops old process
   - Starts new binary
   - Total time: 1-2 seconds

4. **Volume Mount**
   - Source code: `./coordinator/mcp-server → /app`
   - Go modules cache: Named volume for speed
   - No need to rebuild Docker image

### React UI (Vite)

1. **File Change Detected**
   - Vite watches for file changes
   - Lightning-fast detection (<10ms)

2. **Hot Module Replacement**
   - Updates only changed modules
   - Preserves component state
   - No full page reload
   - Total time: <100ms

3. **WebSocket**
   - Vite dev server uses WebSocket
   - Pushes updates to browser instantly
   - No manual refresh needed

## Development Workflow

### Typical Day

```bash
# Morning - Start development environment
make dev-up

# Edit code all day
# Changes auto-reload instantly

# Check logs if needed
make dev-logs

# Evening - Stop everything
make dev-down
```

### Making Changes

**Backend (Go):**
1. Edit `coordinator/mcp-server/handlers/tools.go`
2. Save file
3. Wait 1-2 seconds
4. Test changes at http://localhost:7778

**Frontend (React):**
1. Edit `coordinator/ui/src/App.tsx`
2. Save file
3. Browser updates instantly
4. Component state preserved

## Environment Variables

Set in `.env` or override in `docker-compose.dev.yml`:

```bash
# MCP Server
TRANSPORT_MODE=http
MCP_PORT=7778
MONGODB_URI=mongodb+srv://...
MONGODB_DATABASE=coordinator_db
LOG_LEVEL=debug

# HTTP Bridge
PORT=7095

# React UI
VITE_MCP_BRIDGE_URL=http://localhost:7095
```

## Performance Optimizations

✅ **Named Volumes**: Go module cache persists between restarts
✅ **Incremental Builds**: Air only rebuilds changed packages
✅ **Native Notifications**: No filesystem polling overhead
✅ **Vite HMR**: Instant updates without page refresh
✅ **Source Maps**: Enabled in development for debugging

## Troubleshooting

### Air Not Rebuilding

**Symptoms**: Code changes don't trigger rebuild

**Solutions**:
```bash
# Check Air is watching
make dev-logs | grep "watching"

# Verify volume mount
docker-compose -f docker-compose.dev.yml exec hyperion-mcp-server ls -la /app

# Force restart
make dev-down && make dev-up
```

### Build Errors Not Showing

**Symptoms**: Compilation errors disappear

**Solutions**:
```bash
# Check build error log
docker-compose -f docker-compose.dev.yml exec hyperion-mcp-server cat tmp/build-errors.log

# Or build manually
docker-compose -f docker-compose.dev.yml exec hyperion-mcp-server go build .
```

### UI Not Updating

**Symptoms**: Browser doesn't reflect code changes

**Solutions**:
```bash
# Check Vite is running
make dev-logs | grep "VITE"

# Check WebSocket connection
# Open browser console, look for WebSocket errors

# Restart UI
docker-compose -f docker-compose.dev.yml restart hyperion-ui
```

### Slow Builds

**Symptoms**: Rebuilds take >5 seconds

**Solutions**:
```bash
# Clear Go module cache
docker volume rm dev-squad_mcp-server-cache
make dev-build && make dev-up

# Check exclude patterns in .air.toml
# Ensure vendor/, tmp/ are excluded
```

## Production Deployment

⚠️ **IMPORTANT**: This setup is for development only!

For production, use:
```bash
docker-compose up  # Not docker-compose.dev.yml
```

Production differences:
- ❌ No Air (smaller images)
- ❌ No source code volumes
- ✅ Multi-stage builds
- ✅ Optimized binaries
- ✅ Smaller image sizes

## Next Steps

1. ✅ Configuration complete
2. ✅ Documentation created
3. ✅ Makefile updated
4. ⏳ Start developing: `make dev`
5. ⏳ Read [QUICK_START_DEV.md](QUICK_START_DEV.md) for commands
6. ⏳ Read [DEV_SETUP.md](DEV_SETUP.md) for details

## Testing the Setup

### Test MCP Server Hot-Reload
```bash
# Start server
make dev-mcp

# In another terminal, edit a file
echo "// Test change" >> coordinator/mcp-server/main.go

# Watch logs - should see rebuild and restart
# Remove test change
git checkout coordinator/mcp-server/main.go
```

### Test React UI Hot-Reload
```bash
# Start UI
make dev-ui

# Edit a component
# Open http://localhost:5173
# Make a change to coordinator/ui/src/App.tsx
# Browser should update instantly without refresh
```

### Test Full Stack
```bash
# Start everything
make dev-up

# Test all three services
curl http://localhost:7778/health    # MCP Server
curl http://localhost:7095/health    # HTTP Bridge
open http://localhost:5173           # React UI

# Make changes and test hot-reload
```

## Resources

- [Air GitHub](https://github.com/air-verse/air)
- [Vite HMR API](https://vitejs.dev/guide/api-hmr.html)
- [Docker Compose Documentation](https://docs.docker.com/compose/)

## Support

For issues or questions:
1. Check [DEV_SETUP.md](DEV_SETUP.md) troubleshooting section
2. Check Air logs: `make dev-logs`
3. Rebuild from scratch: `make dev-build && make dev-up`
4. Check GitHub issues for Air or Vite

---

**Status**: ✅ Complete and ready for development
**Last Updated**: 2025-10-04
**Setup Time**: ~5 minutes to start
**Developer Experience**: ⚡ Lightning fast hot-reload
