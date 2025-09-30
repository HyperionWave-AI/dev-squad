# Full Stack Coordinator Setup Guide

Complete guide to running the Hyperion Coordinator with MongoDB backend, MCP server, HTTP bridge, and React UI.

---

## Architecture Overview

```
┌─────────────────────┐
│   React UI          │  Port 5173
│   (Browser)         │
└──────────┬──────────┘
           │ HTTP
           ↓
┌─────────────────────┐
│  HTTP Bridge        │  Port 8095
│  (Go Server)        │
└──────────┬──────────┘
           │ stdio
           ↓
┌─────────────────────┐
│  MCP Server         │  stdio transport
│  (Go Server)        │
└──────────┬──────────┘
           │ MongoDB Driver
           ↓
┌─────────────────────┐
│  MongoDB Atlas      │  Cloud Database
│  coordinator_db     │
└─────────────────────┘
```

---

## Prerequisites

- **Go 1.25+** for MCP server and HTTP bridge
- **Node.js 18+** for React UI
- **MongoDB Atlas** account (already configured with credentials in MCP server)

---

## Step 1: Build MCP Server

```bash
cd development/coordinator/mcp-server
go mod download
go build -o hyperion-coordinator-mcp
```

**Test the MCP server:**
```bash
./hyperion-coordinator-mcp
# Should output initialization logs and wait for stdin
# Press Ctrl+C to exit
```

---

## Step 2: Build HTTP Bridge

```bash
cd ../mcp-http-bridge
go mod download
go build -o hyperion-coordinator-bridge
```

---

## Step 3: Install UI Dependencies

```bash
cd ../ui
npm install
```

---

## Step 4: Start All Services

### Terminal 1: MCP Server (via HTTP Bridge)

The HTTP bridge will automatically start the MCP server as a child process:

```bash
cd development/coordinator/mcp-http-bridge
./hyperion-coordinator-bridge
```

**Expected output:**
```
MCP server started with PID: 12345
MCP connection initialized successfully
HTTP bridge listening on port 8095
MCP server path: ../mcp-server/hyperion-coordinator-mcp
Frontend CORS enabled for: http://localhost:5173, http://localhost:3000
```

### Terminal 2: React UI

```bash
cd development/coordinator/ui
npm run dev
```

**Expected output:**
```
  VITE v7.1.7  ready in 500 ms

  ➜  Local:   http://localhost:5173/
  ➜  Network: use --host to expose
```

---

## Step 5: Access the UI

Open your browser to: **http://localhost:5173**

You should see the Coordinator UI with:
- Task Dashboard (showing all tasks from MongoDB)
- Knowledge Browser
- Real-time updates every 3 seconds

---

## Testing the Integration

### 1. Create a Test Human Task (via CLI)

You can test the MCP server directly using the HTTP bridge:

```bash
curl -X POST http://localhost:8095/api/mcp/tools/call \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: test-1" \
  -d '{
    "name": "coordinator_create_human_task",
    "arguments": {
      "prompt": "Test task from curl - add authentication to tasks-api"
    }
  }'
```

**Expected response:**
```json
{
  "content": [
    {
      "type": "text",
      "text": "✓ Human task created successfully\n\nTask ID: abc-123-def-456\nCreated: 2025-09-30 12:00:00 UTC\nStatus: pending\n\nPrompt: Test task from curl - add authentication to tasks-api"
    }
  ]
}
```

### 2. Verify in UI

Refresh the UI at http://localhost:5173 - you should see the new task appear in the dashboard.

### 3. Create an Agent Task

```bash
curl -X POST http://localhost:8095/api/mcp/tools/call \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: test-2" \
  -d '{
    "name": "coordinator_create_agent_task",
    "arguments": {
      "humanTaskId": "abc-123-def-456",
      "agentName": "backend-services-specialist",
      "role": "Implement JWT middleware in tasks-api",
      "todos": [
        "Design JWT validation logic",
        "Implement middleware function",
        "Add tests",
        "Update API documentation"
      ]
    }
  }'
```

### 4. Update Task Status

```bash
curl -X POST http://localhost:8095/api/mcp/tools/call \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: test-3" \
  -d '{
    "name": "coordinator_update_task_status",
    "arguments": {
      "taskId": "abc-123-def-456",
      "status": "in_progress",
      "notes": "Started working on JWT implementation"
    }
  }'
```

### 5. Verify in MongoDB Atlas

You can also check the data directly in MongoDB Atlas:
1. Go to https://cloud.mongodb.com
2. Navigate to your cluster: `devdb.yqf8f8r.mongodb.net`
3. Browse Collections → `coordinator_db`
4. Collections: `human_tasks`, `agent_tasks`, `knowledge_entries`

---

## Environment Variables

### HTTP Bridge (Optional)

```bash
# Override MCP server path (default: ../mcp-server/hyperion-coordinator-mcp)
export MCP_SERVER_PATH="/path/to/hyperion-coordinator-mcp"

# Override port (default: 8095)
export PORT=9000
```

### MCP Server (Optional)

```bash
# Override MongoDB URI (default: hardcoded Atlas connection)
export MONGODB_URI="mongodb+srv://..."

# Override database name (default: coordinator_db)
export MONGODB_DATABASE="my_custom_db"
```

### React UI (Optional)

Create `.env.local` in `development/coordinator/ui/`:

```bash
# Override bridge URL (default: http://localhost:8095)
VITE_MCP_BRIDGE_URL=http://localhost:9000
```

---

## Troubleshooting

### MCP Server Won't Start

**Error:** `Failed to connect to MongoDB`

**Solution:** Check MongoDB Atlas credentials and network access:
1. Verify the connection string in `mcp-server/main.go:32`
2. Ensure your IP is whitelisted in MongoDB Atlas Network Access
3. Test connection: `mongosh "mongodb+srv://dev:fvOKzv9enD8CSVwD@devdb.yqf8f8r.mongodb.net/"`

---

### HTTP Bridge Connection Issues

**Error:** `MCP server not found`

**Solution:** Build the MCP server first:
```bash
cd development/coordinator/mcp-server
go build -o hyperion-coordinator-mcp
```

**Error:** `failed to initialize MCP connection`

**Solution:** The MCP server might be crashing on startup. Check stderr output for errors.

---

### UI Can't Connect to Bridge

**Error:** `Failed to connect to MCP HTTP bridge`

**Solution:**
1. Verify the bridge is running: `curl http://localhost:8095/health`
2. Check browser console for CORS errors
3. Ensure the bridge URL matches in UI env vars

**Expected health response:**
```json
{
  "status": "healthy",
  "service": "hyperion-coordinator-http-bridge",
  "version": "1.0.0"
}
```

---

### UI Shows No Tasks

**Possible causes:**
1. No tasks in database yet - create a test task using curl (see above)
2. Bridge not connecting to MCP server - check bridge logs
3. MCP server not connecting to MongoDB - check MCP server logs

**Debug:**
```bash
# Check bridge logs (Terminal 1)
# Check UI console (Browser DevTools)
# List all resources via bridge
curl http://localhost:8095/api/mcp/resources
```

---

## Development Workflow

### Hot Reload Setup

The React UI supports hot module replacement (HMR), so changes to React components will auto-refresh.

For Go code changes, you need to restart the services:

**MCP Server changes:**
```bash
cd development/coordinator/mcp-server
go build -o hyperion-coordinator-mcp
# Then restart HTTP bridge (which will restart MCP server)
```

**HTTP Bridge changes:**
```bash
cd development/coordinator/mcp-http-bridge
go build -o hyperion-coordinator-bridge
./hyperion-coordinator-bridge
```

---

## Production Deployment

### 1. Build Everything

```bash
# MCP Server
cd development/coordinator/mcp-server
go build -o hyperion-coordinator-mcp

# HTTP Bridge
cd ../mcp-http-bridge
go build -o hyperion-coordinator-bridge

# UI
cd ../ui
npm run build
# Output in: ui/dist/
```

### 2. Deploy Strategy

**Option A: Docker Compose**
- Create `docker-compose.yml` with all three services
- Use environment variables for configuration
- Mount MongoDB credentials as secrets

**Option B: Kubernetes**
- Deploy MCP server as a Deployment
- Deploy HTTP bridge as a Service (exposed internally)
- Deploy UI as static files via Nginx/CDN
- Use ConfigMaps for non-sensitive config
- Use Secrets for MongoDB credentials

**Option C: Single Go Binary**
- Embed the UI dist files in the HTTP bridge binary
- Serve static files from Go (using `embed` package)
- Single binary deployment with MCP server as dependency

---

## API Reference

### HTTP Bridge Endpoints

All endpoints respond with JSON.

#### GET /health
Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "service": "hyperion-coordinator-http-bridge",
  "version": "1.0.0"
}
```

#### GET /api/mcp/tools
List all available MCP tools.

**Response:**
```json
{
  "tools": [
    {
      "name": "coordinator_create_human_task",
      "description": "Create a new human task...",
      "inputSchema": {...}
    },
    ...
  ]
}
```

#### POST /api/mcp/tools/call
Call an MCP tool.

**Request:**
```json
{
  "name": "coordinator_create_human_task",
  "arguments": {
    "prompt": "Add feature X"
  }
}
```

**Response:** MCP tool result (varies by tool)

#### GET /api/mcp/resources
List all MCP resources (tasks).

**Response:**
```json
{
  "resources": [
    {
      "uri": "hyperion://task/human/abc-123",
      "name": "Human Task: ...",
      "description": "...",
      "mimeType": "application/json"
    },
    ...
  ]
}
```

#### GET /api/mcp/resources/read?uri=...
Read a specific MCP resource.

**Request:** `?uri=hyperion://task/human/abc-123`

**Response:**
```json
{
  "contents": [
    {
      "uri": "hyperion://task/human/abc-123",
      "mimeType": "application/json",
      "text": "{\"id\":\"abc-123\",\"prompt\":\"...\",\"status\":\"pending\",...}"
    }
  ]
}
```

---

## Next Steps

1. **Add Authentication**: Protect the HTTP bridge with JWT tokens
2. **WebSocket Support**: Replace polling with real-time WebSocket updates
3. **Task Creation UI**: Add forms for creating tasks directly from the UI
4. **Agent Dashboard**: Dedicated views for each agent showing their task queue
5. **Qdrant Integration**: Connect Qdrant MCP for knowledge management

---

**Last Updated:** 2025-09-30
**Version:** 1.0
**Maintainer:** AI & Experience Squad