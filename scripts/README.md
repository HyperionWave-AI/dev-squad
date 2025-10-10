# Hyperion Coordinator Scripts

Scripts for managing Hyperion Coordinator services with Docker.

## Quick Start

### 🌟 Interactive Wizard (Recommended)

The easiest way to get started - just run without arguments:

```bash
./scripts/start-coordinator.sh
```

The wizard will guide you through:
1. **Folder selection** - Choose project folders to index (defaults to current folder)
2. **Port configuration** - Select UI port (default: 5173)
3. **Project naming** - Set unique name for multiple instances (defaults to current folder name)
4. **Development mode** - Enable hot-reload if needed
5. **Confirmation** - Review settings before starting

**Smart Defaults:**
- First folder: Current directory (just press Enter)
- Project name: Current folder name (e.g., `my-app` from `/Users/max/projects/my-app`)
- Port: 5173
- Mode: Production

### Command-Line Mode

For automation or when you know exactly what you need:

```bash
./scripts/start-coordinator.sh --folder /path/to/your/project
```

### Start with custom port
```bash
./scripts/start-coordinator.sh --folder /path/to/project --port 8080
```

### Start with multiple folders
```bash
./scripts/start-coordinator.sh \
  --folder /Users/max/projects/hyperion \
  --folder /Users/max/projects/app2 \
  --folder /Users/max/workspace/monorepo
```

### Development mode (hot-reload)
```bash
./scripts/start-coordinator.sh --folder /path/to/project --dev
```

### Stop services
```bash
./scripts/start-coordinator.sh --stop
```

### Stop and clean volumes
```bash
./scripts/start-coordinator.sh --clean
```

## Interactive Wizard Workflow

When you run the script without arguments, you'll see a friendly setup wizard:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Hyperion Coordinator Setup Wizard
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📁 Folder Configuration
   Enter folders to index for code search and file watching.
   Tip: You can add multiple folders

Enter folder path [default: current folder]: ↵
[INFO] Using current folder: /Users/max/projects/my-app

Add another folder? [y/N]: n

🌐 Port Configuration
   The UI will be accessible at http://localhost:<PORT>

Enter public port [default: 5173]: ↵
[SUCCESS] Using port: 5173

📦 Project Configuration
   Project name is used for Docker Compose containers and volumes.
   Tip: Use unique names if running multiple instances

Enter project name [default: my-app]: ↵
[SUCCESS] Project name: my-app

⚙️  Development Mode
   Development mode enables hot-reload for code changes.

Enable development mode? [y/N]: ↵
[INFO] Production mode selected

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Configuration Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📁 Folders (1):
   /Users/max/projects/my-app

🌐 Port:         5173
📦 Project:      my-app
⚙️  Mode:         Production

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Start Hyperion Coordinator with this configuration? [Y/n]: y
```

**💡 Quickest Setup (just press Enter 5 times!):**
```bash
cd /path/to/your/project
./scripts/start-coordinator.sh
# Press Enter, Enter, Enter, Enter, Enter → Done! 🚀
```

## What It Does

1. **Validates** - Checks that all specified folders exist
2. **Mounts** - Maps host folders to Docker volumes (read-only)
3. **Configures** - Sets up file watcher path mappings
4. **Starts** - Launches coordinator services
5. **Reports** - Shows access URLs and next steps

## File Watcher

The script automatically enables the file watcher with proper path mappings:

**Host Path** → **Container Path**
- `/Users/max/project1` → `/workspace/mount0`
- `/Users/max/project2` → `/workspace/mount1`
- `/Users/max/project3` → `/workspace/mount2`

When adding folders to the code index via MCP, use the **container paths**:

```typescript
// Via MCP tool
mcp__hyper__code_index_add_folder({
  folderPath: "/workspace/mount0"
})
```

## Configuration

### Generated Files

The script creates `docker-compose.volumes.yml` with:
- Volume mounts for each folder
- Path mapping environment variables
- Port configuration

**Example:**
```yaml
services:
  hyperion-mcp-server:
    environment:
      - ENABLE_FILE_WATCHER=true
      - CODE_INDEX_PATH_MAPPINGS=/Users/max/project:/workspace/mount0
    volumes:
      - /Users/max/project:/workspace/mount0:ro
```

### Environment Variables

You can also set these in `.env`:
```bash
# Custom port
PUBLIC_PORT=8080

# OpenAI API key for embeddings
OPENAI_API_KEY=sk-...
```

## Access Points

After starting, access the coordinator via:

| Service | URL | Purpose |
|---------|-----|---------|
| **UI** | `http://localhost:5173` | Main interface |
| **Health** | `http://localhost:5173/health` | UI health check |
| **Bridge Health** | `http://localhost:5173/bridge-health` | Backend health |

## Examples

### First-Time Setup (Wizard)
```bash
# Navigate to your project folder
cd ~/projects/my-app

# Run interactive wizard
./scripts/start-coordinator.sh

# Just press Enter 5 times to accept all defaults:
# ✓ Folder: current directory (automatically detected)
# ✓ Port: 5173 (default)
# ✓ Name: my-app (from folder name)
# ✓ Dev mode: no (default)
# ✓ Confirm: yes

# Access UI after startup
open http://localhost:5173

# View logs
docker logs -f my-app-hyperion-mcp-server-1
```

### Quick Development Setup (Command-Line)
```bash
# Start coordinator with your project
./scripts/start-coordinator.sh --folder ~/projects/my-app

# Access UI
open http://localhost:5173

# View logs
docker logs -f hyperion-coordinator-hyperion-mcp-server-1
```

### Multiple Projects (Wizard Recommended)
```bash
# Terminal 1: Start first project
./scripts/start-coordinator.sh
# → Add folders, port 5173, name "frontend-app"

# Terminal 2: Start second project
./scripts/start-coordinator.sh
# → Add folders, port 5174, name "backend-api"

# Access both UIs
open http://localhost:5173  # frontend-app
open http://localhost:5174  # backend-api
```

### Production Setup
```bash
# Start with custom port and multiple projects
./scripts/start-coordinator.sh \
  --folder /opt/projects/prod-app \
  --folder /opt/projects/shared-lib \
  --port 9000

# Check status
docker-compose -f docker-compose.yml -f docker-compose.volumes.yml ps

# View logs
docker-compose -f docker-compose.yml -f docker-compose.volumes.yml logs -f
```

### Adding Code Index
```bash
# 1. Start coordinator
./scripts/start-coordinator.sh --folder ~/projects/hyperion

# 2. Via MCP (use container path!)
mcp__hyper__code_index_add_folder({
  folderPath: "/workspace/mount0",
  description: "Hyperion main project"
})

# 3. Scan folder
mcp__hyper__code_index_scan({
  folderPath: "/workspace/mount0"
})

# 4. Search code
mcp__hyper__code_index_search({
  query: "authentication middleware",
  limit: 10
})
```

## Troubleshooting

### Port already in use
```bash
# Check what's using the port
lsof -i :5173

# Use different port
./scripts/start-coordinator.sh --folder /path/to/project --port 8080
```

### Services not starting
```bash
# Check logs
docker-compose -f docker-compose.yml -f docker-compose.volumes.yml logs

# Restart services
./scripts/start-coordinator.sh --stop
./scripts/start-coordinator.sh --folder /path/to/project
```

### File watcher not working
```bash
# Check MCP server logs
docker logs -f hyperion-mcp-server

# Verify path mappings
docker exec hyperion-mcp-server env | grep CODE_INDEX_PATH_MAPPINGS

# Check mounted folders
docker exec hyperion-mcp-server ls -la /workspace
```

### Clean restart
```bash
# Stop and remove all volumes
./scripts/start-coordinator.sh --clean

# Start fresh
./scripts/start-coordinator.sh --folder /path/to/project
```

## Architecture

```
Host Machine                    Docker Network
┌─────────────────┐            ┌────────────────────────┐
│                 │            │                        │
│  /Users/max/    │────────────│  /workspace/mount0     │
│  projects/app   │   mount    │                        │
│                 │   (ro)     │  hyperion-mcp-server   │
└─────────────────┘            │  - File watcher        │
                               │  - Code indexing       │
┌─────────────────┐            │  - Path mapping        │
│                 │            │                        │
│  Browser        │────────────│  nginx (port 5173)     │
│  localhost:5173 │   access   │  - UI (React)          │
│                 │            │  - /api/mcp/ → bridge  │
└─────────────────┘            └────────────────────────┘
```

## Security

- **Read-only mounts**: Host folders mounted as `:ro` to prevent modifications
- **Internal network**: Backend services not exposed to host
- **Single port**: Only UI port exposed (5173 by default)
- **CORS protection**: nginx validates Origins

## Advanced Usage

### Custom docker-compose files
```bash
# Use with custom compose file
./scripts/start-coordinator.sh --folder /path/to/project

# Then manually override
docker-compose -f docker-compose.yml \
               -f docker-compose.volumes.yml \
               -f my-custom.yml \
               up -d
```

### Volume caching (Mac/Windows performance)
Edit `docker-compose.volumes.yml` after generation:
```yaml
volumes:
  - /Users/max/project:/workspace/mount0:ro,cached  # Add ,cached
```

### Selective watching
Exclude paths in file watcher by editing container:
```bash
docker exec -it hyperion-mcp-server vi /app/.gitignore
```

## See Also

- [Code Indexing Documentation](../coordinator/mcp-server/CLAUDE.md)
- [Docker File Watcher Guide](../coordinator/mcp-server/DOCKER_FILE_WATCHER.md)
- [MCP Protocol Reference](https://modelcontextprotocol.io/)
