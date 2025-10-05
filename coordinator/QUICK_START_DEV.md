# Quick Start - Development with Hot-Reload

## 🚀 TL;DR

```bash
# Start everything with hot-reload
make dev

# Or start in background
make dev-up

# Follow logs
make dev-logs

# Stop everything
make dev-down
```

## What You Get

✅ **MCP Server** - Code changes auto-rebuild and restart (1-2 sec)
✅ **HTTP Bridge** - Code changes auto-rebuild and restart (1-2 sec)
✅ **React UI** - Instant hot module replacement (<100ms)
✅ **All running** - MCP: http://localhost:7778, Bridge: http://localhost:8095, UI: http://localhost:5173

## Make Commands

```bash
make help          # Show all available commands
make dev           # Start with hot-reload (foreground)
make dev-up        # Start with hot-reload (background)
make dev-down      # Stop all services
make dev-logs      # View logs
make dev-build     # Rebuild images
make dev-mcp       # Start only MCP server
make dev-bridge    # Start only HTTP bridge
make dev-ui        # Start only React UI
make install-air   # Install Air locally
```

## How It Works

### Go Services (Air Hot-Reload)
1. You edit a `.go` file
2. Air detects the change
3. Rebuilds the binary (1-2 seconds)
4. Restarts the service automatically
5. You see the changes immediately

### React UI (Vite HMR)
1. You edit a `.tsx` or `.ts` file
2. Vite detects the change instantly
3. Hot Module Replacement updates the browser
4. No page refresh needed
5. Component state preserved

## File Structure

```
coordinator/
├── mcp-server/
│   ├── .air.toml           # Air configuration
│   ├── Dockerfile.dev      # Development Dockerfile with Air
│   └── main.go             # Your code here
├── mcp-http-bridge/
│   ├── .air.toml           # Air configuration
│   ├── Dockerfile.dev      # Development Dockerfile with Air
│   └── main.go             # Your code here
└── ui/
    ├── Dockerfile.dev      # Development Dockerfile with Vite
    └── src/                # Your React code here

docker-compose.dev.yml      # Development compose file
```

## Development Workflow

### 1. Start Services
```bash
make dev-up
```

### 2. Make Changes
Edit any file in:
- `coordinator/mcp-server/**/*.go`
- `coordinator/mcp-http-bridge/**/*.go`
- `coordinator/ui/src/**/*`

### 3. See Changes
- **Go**: Wait 1-2 seconds for rebuild
- **React**: Changes appear instantly

### 4. Check Logs
```bash
# All services
make dev-logs

# Specific service
docker-compose -f docker-compose.dev.yml logs -f hyperion-mcp-server
```

### 5. Stop Services
```bash
make dev-down
```

## Troubleshooting

### Changes not detected?
```bash
# Restart the service
make dev-down && make dev-up

# Or rebuild images
make dev-build && make dev-up
```

### Build errors?
```bash
# Check Air build log
docker-compose -f docker-compose.dev.yml exec hyperion-mcp-server cat tmp/build-errors.log
```

### Need to add dependencies?
```bash
# After changing go.mod or package.json
make dev-build && make dev-up
```

## Production vs Development

| Feature | Development (`make dev`) | Production (`docker-compose up`) |
|---------|--------------------------|----------------------------------|
| Hot-reload | ✅ Yes | ❌ No |
| Build time | Slower (includes Air) | Faster (optimized) |
| Image size | Larger | Smaller (multi-stage) |
| Source code | Mounted as volume | Compiled into image |
| Use for | Local development | Deployment |

## Environment Variables

Set in `.env` or `docker-compose.dev.yml`:

```bash
MONGODB_URI=mongodb+srv://...
MONGODB_DATABASE=coordinator_db
LOG_LEVEL=debug
MCP_PORT=7778
```

## Next Steps

- Read [DEV_SETUP.md](./DEV_SETUP.md) for detailed documentation
- See [Makefile](../Makefile) for all available commands
- Check `.air.toml` files to customize hot-reload behavior

## Common Issues

**Port already in use?**
```bash
# Check what's using the port
lsof -i :7778

# Or change port in docker-compose.dev.yml
```

**Go modules not found?**
```bash
# Clear cache and rebuild
docker volume rm dev-squad_mcp-server-cache
make dev-build && make dev-up
```

**UI not updating?**
```bash
# Check Vite is running
docker-compose -f docker-compose.dev.yml logs hyperion-ui | grep "Local:"

# Restart UI
docker-compose -f docker-compose.dev.yml restart hyperion-ui
```

## Tips

💡 Use `make dev-up` to run in background while you work
💡 Use `make dev-logs` to monitor all services at once
💡 Air only rebuilds changed packages (faster subsequent builds)
💡 Vite HMR preserves React component state across changes
💡 Check `tmp/build-errors.log` if Air swallows errors
