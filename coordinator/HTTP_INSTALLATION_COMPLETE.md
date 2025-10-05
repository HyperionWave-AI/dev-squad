# ✅ Hyperion Coordinator MCP - HTTP Installation Complete

The Hyperion Coordinator MCP server is now running in **HTTP streaming mode** and configured for Claude Code CLI.

---

## What Was Set Up

### 1. Docker Container (HTTP Mode)
- **Container**: `hyperion-coordinator-mcp`
- **Transport**: HTTP Streamable (not stdio)
- **Port**: 7778 (exposed to host)
- **Endpoint**: http://localhost:7778/mcp
- **Health Check**: http://localhost:7778/health
- **MongoDB**: Connected to coordinator_db

### 2. Claude Code Configuration
- **Location**: `~/.claude/settings.json`
- **Server Name**: `hyperion-coordinator`
- **Connection**: HTTP URL (not command/args)
- **URL**: `http://localhost:7778/mcp`

### 3. Configuration Files
- **Docker Compose**: `coordinator/docker-compose.mcp-only.yml`
- **Environment Variables**:
  - `TRANSPORT_MODE=http`
  - `MCP_PORT=7778`
  - MongoDB connection configured

---

## 🚨 IMPORTANT: Restart Claude Code

**You MUST restart Claude Code** for the HTTP MCP server to appear in `/mcp` list.

```bash
# Quit Claude Code completely
# Then restart it
```

After restart, run `/mcp` to verify `hyperion-coordinator` appears in the list.

---

## ✅ Verification Steps

### 1. Check Container is Running
```bash
docker ps | grep hyperion-coordinator-mcp
```

**Expected**: Container status shows `Up X minutes` with port `7778:7778`

### 2. Test Health Endpoint
```bash
curl http://localhost:7778/health
```

**Expected**: `OK`

### 3. Test MCP Endpoint
```bash
curl -X POST http://localhost:7778/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}'
```

**Expected**: JSON response with server info:
```json
{
  "jsonrpc":"2.0",
  "id":1,
  "result":{
    "serverInfo":{
      "name":"hyperion-coordinator-mcp",
      "version":"1.0.0"
    },
    ...
  }
}
```

### 4. After Restarting Claude Code
Run in Claude Code:
```
/mcp
```

**Expected**: `hyperion-coordinator` appears in the list

### 5. Test MCP Tool
```typescript
mcp__hyperion-coordinator__coordinator_list_human_tasks({})
```

**Expected**: Returns list of tasks (may be empty)

---

## 🔧 Management Commands

### View Container Logs
```bash
docker logs hyperion-coordinator-mcp -f
```

### Check HTTP Server Status
```bash
docker logs hyperion-coordinator-mcp | grep "HTTP server listening"
```

**Expected**:
```
INFO	HTTP server listening	{"address": ":7778", "mcp_endpoint": "/mcp", "health_endpoint": "/health"}
```

### Restart Container
```bash
cd /Users/maxmednikov/MaxSpace/dev-squad/coordinator
docker-compose -f docker-compose.mcp-only.yml restart
```

### Stop Container
```bash
docker-compose -f docker-compose.mcp-only.yml stop
```

### Start Container
```bash
docker-compose -f docker-compose.mcp-only.yml start
```

### Rebuild After Code Changes
```bash
docker-compose -f docker-compose.mcp-only.yml up -d --build
```

---

## 📊 Available MCP Tools (9 total)

Once Claude Code is restarted, you'll have access to:

### Task Management
1. `coordinator_create_human_task` - Create user task
2. `coordinator_create_agent_task` - Create agent task
3. `coordinator_list_human_tasks` - List all human tasks
4. `coordinator_list_agent_tasks` - List agent tasks (filterable by agentName/humanTaskId)
5. `coordinator_update_task_status` - Update task status and notes
6. `coordinator_update_todo_status` - Update TODO item within agent task

### Knowledge Management
7. `coordinator_upsert_knowledge` - Store knowledge in collection
8. `coordinator_query_knowledge` - Search knowledge by collection and query

### Maintenance
9. `coordinator_clear_task_board` - Clear all tasks (optionally clear knowledge)

### MCP Resources (2 total)
- `hyperion://task/human/{taskId}` - Human task details
- `hyperion://task/agent/{agentName}/{taskId}` - Agent task details

---

## 🎯 Why HTTP Streaming Mode?

### Problem with Docker + stdio
- `docker exec -i` creates a **new process** each time
- Claude Code expects a **persistent process**
- Connection couldn't be maintained
- Server never appeared in `/mcp` list

### Solution: HTTP Streaming
- Container runs continuously with HTTP server
- Claude Code connects via HTTP URL
- Persistent connection maintained
- ✅ Works reliably with Docker

---

## 🔍 Troubleshooting

### Container Not Running
```bash
docker ps -a | grep hyperion-coordinator-mcp
docker logs hyperion-coordinator-mcp
```

If container exited, check logs for MongoDB connection errors.

### Port Already in Use
```bash
# Check what's using port 7778
lsof -i :7778

# Change port in docker-compose.mcp-only.yml
ports:
  - "7779:7778"  # Map host 7779 to container 7778

# Update settings.json URL
"url": "http://localhost:7779/mcp"
```

### MCP Server Not in `/mcp` List
1. ✅ Verify container is running: `docker ps`
2. ✅ Test health endpoint: `curl http://localhost:7778/health`
3. ✅ Test MCP endpoint with curl (see above)
4. ✅ Check settings.json syntax is valid
5. ✅ **RESTART Claude Code** (most common fix)
6. ✅ Check Claude logs: `tail -f ~/Library/Logs/Claude/mcp.log`

### MongoDB Connection Errors
```bash
# Check logs
docker logs hyperion-coordinator-mcp | grep MongoDB

# Test MongoDB connection from container
docker exec -it hyperion-coordinator-mcp wget -O- http://localhost:7778/health
```

If MongoDB connection fails:
1. Check MongoDB Atlas cluster is accessible
2. Verify MONGODB_URI in `.env` file
3. Check network connectivity

### HTTP Connection Refused
```bash
# Verify port is exposed
docker port hyperion-coordinator-mcp

# Expected output:
7778/tcp -> 0.0.0.0:7778
```

---

## ⚙️ Configuration

### Environment Variables

Edit `/Users/maxmednikov/MaxSpace/dev-squad/.env`:

```bash
# MongoDB Configuration
MONGODB_URI=mongodb+srv://...
MONGODB_DATABASE=coordinator_db_max

# MCP Transport (DO NOT CHANGE - required for HTTP mode)
TRANSPORT_MODE=http
MCP_PORT=7778

# Logging
LOG_LEVEL=info
```

### Change MongoDB Database
```bash
# 1. Edit .env
MONGODB_DATABASE=your_database_name

# 2. Restart container
cd coordinator
docker-compose -f docker-compose.mcp-only.yml restart
```

### Change MCP Port
```bash
# 1. Edit docker-compose.mcp-only.yml
ports:
  - "7779:7779"
environment:
  - MCP_PORT=7779

# 2. Edit settings.json
"url": "http://localhost:7779/mcp"

# 3. Rebuild container
docker-compose -f docker-compose.mcp-only.yml up -d --build

# 4. Restart Claude Code
```

---

## 🏗️ Architecture

```
┌─────────────────────────────────────┐
│     Claude Code CLI                 │
│     (macOS Host)                    │
└───────────────┬─────────────────────┘
                │ HTTP (port 7778)
                │ URL: http://localhost:7778/mcp
                ▼
┌─────────────────────────────────────┐
│  Container: hyperion-coordinator-mcp│
│  ┌───────────────────────────────┐  │
│  │ HTTP Server (port 7778)       │  │
│  │ ├─ /mcp (MCP endpoint)        │  │
│  │ └─ /health (health check)     │  │
│  │                               │  │
│  │ MCP Server                    │  │
│  │ - 9 tools                     │  │
│  │ - 2 resources                 │  │
│  │ - MongoDB client              │  │
│  └───────────────────────────────┘  │
└───────────────┬─────────────────────┘
                │ MongoDB Driver (TLS)
                ▼
┌─────────────────────────────────────┐
│   MongoDB Atlas                     │
│   Database: coordinator_db_max      │
│   Collections:                      │
│   - human_tasks                     │
│   - agent_tasks                     │
│   - knowledge_entries               │
└─────────────────────────────────────┘
```

**Transport**: HTTP Streamable (MCP 2024-11-05 protocol)
**Benefits**:
- ✅ Works reliably with Docker
- ✅ Easy debugging (curl, browser, Postman)
- ✅ Health check endpoint
- ✅ Container auto-restarts
- ✅ Port exposure for monitoring

---

## 📝 Next Steps

1. ✅ Container running on HTTP mode
2. ✅ Claude Code configured with HTTP URL
3. ✅ Health endpoint responding
4. ✅ MCP endpoint responding
5. 🚨 **RESTART Claude Code** (do this now!)
6. 🎯 Run `/mcp` to verify server appears
7. 🎯 Test tool: `coordinator_list_human_tasks({})`
8. 📖 Read [SPECIFICATION.md](./SPECIFICATION.md) for full tool reference

---

## 🔄 Updates

### Update MCP Server Code
```bash
# 1. Make code changes in coordinator/mcp-server/

# 2. Rebuild container
cd /Users/maxmednikov/MaxSpace/dev-squad/coordinator
docker-compose -f docker-compose.mcp-only.yml up -d --build

# 3. Container will auto-restart with new code
# 4. No need to restart Claude Code unless settings changed
```

---

## 📦 Alternative: Full Stack with UI

To run the **full UI + HTTP Bridge + MCP Server** stack:

```bash
cd /Users/maxmednikov/MaxSpace/dev-squad
docker-compose up -d
```

This starts:
- MCP Server (HTTP mode on port 7778)
- HTTP Bridge (port 8095 - for web UI)
- React UI (port 5173)

Access web UI: http://localhost:5173

**Note**: For Claude Code CLI, you only need the MCP-only container (port 7778).

---

## ✅ Installation Checklist

- [x] Docker container built and running
- [x] HTTP server listening on port 7778
- [x] Health endpoint responding (OK)
- [x] MCP endpoint responding (initialize works)
- [x] Claude Code settings.json updated with HTTP URL
- [x] MongoDB connection successful
- [ ] **Claude Code restarted** ← DO THIS NOW!
- [ ] `/mcp` shows hyperion-coordinator
- [ ] Test tool works

---

**Installation complete!** 🎉

**REMEMBER**: You must **restart Claude Code** for the server to appear in `/mcp` list.

After restart, the `hyperion-coordinator` MCP server will be available with 9 tools for task coordination and knowledge management.
