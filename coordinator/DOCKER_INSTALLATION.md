# Hyperion Coordinator MCP - Docker Installation for Claude Code

This guide shows how to run the Hyperion Coordinator MCP server in Docker and connect it to Claude Code CLI.

---

## ✅ Installation Complete

The Hyperion Coordinator MCP server is now installed and configured:

### What Was Set Up

1. **Docker Container**: `hyperion-coordinator-mcp`
   - Running MCP server in isolated container
   - Connected to MongoDB Atlas (coordinator_db_max)
   - Automatically restarts on failure

2. **Claude Code Configuration**: `~/.claude/settings.json`
   - MCP server: `hyperion-coordinator`
   - Communication: Docker exec stdin/stdout
   - Environment: MongoDB URI and database configured

3. **Docker Compose**: `coordinator/docker-compose.mcp-only.yml`
   - Standalone MCP server configuration
   - Network isolation
   - Environment variable support

---

## 🚀 Quick Start

### 1. Start the Container (if not running)

```bash
cd /Users/maxmednikov/MaxSpace/dev-squad/coordinator
docker-compose -f docker-compose.mcp-only.yml up -d
```

### 2. Verify Container is Running

```bash
docker ps | grep hyperion-coordinator-mcp
```

**Expected output:**
```
hyperion-coordinator-mcp   ... Up X seconds
```

### 3. Restart Claude Code

**IMPORTANT**: You MUST restart Claude Code for the new MCP server to be available.

```bash
# If running in terminal, quit and restart
# If using VS Code extension, reload window
```

### 4. Test MCP Tools

After restarting Claude Code, test the coordinator:

```typescript
// List human tasks
mcp__hyperion-coordinator__coordinator_list_human_tasks({})

// Create a test task
mcp__hyperion-coordinator__coordinator_create_human_task({
  prompt: "Test task for Docker installation"
})
```

---

## 🔧 Management Commands

### View Container Logs

```bash
docker logs hyperion-coordinator-mcp --tail 50 -f
```

### Restart Container

```bash
docker restart hyperion-coordinator-mcp
```

### Stop Container

```bash
docker stop hyperion-coordinator-mcp
```

### Start Container

```bash
docker start hyperion-coordinator-mcp
```

### Rebuild After Code Changes

```bash
cd /Users/maxmednikov/MaxSpace/dev-squad/coordinator
docker-compose -f docker-compose.mcp-only.yml up -d --build
```

### Remove Container and Clean Up

```bash
docker-compose -f docker-compose.mcp-only.yml down
```

---

## 📊 Available MCP Tools

Once Claude Code is restarted, you'll have access to **9 MCP tools**:

### Task Management
- `coordinator_create_human_task` - Create user task
- `coordinator_create_agent_task` - Create agent task
- `coordinator_list_human_tasks` - List all human tasks
- `coordinator_list_agent_tasks` - List agent tasks (filterable)
- `coordinator_update_task_status` - Update task status
- `coordinator_update_todo_status` - Update TODO item

### Knowledge Management
- `coordinator_upsert_knowledge` - Store knowledge
- `coordinator_query_knowledge` - Search knowledge

### Maintenance
- `coordinator_clear_task_board` - Clear all tasks

### MCP Resources
- `hyperion://task/human/{taskId}` - Human task details
- `hyperion://task/agent/{agentName}/{taskId}` - Agent task details

---

## 🔍 Troubleshooting

### Container Not Running

```bash
# Check container status
docker ps -a | grep hyperion-coordinator-mcp

# Check logs for errors
docker logs hyperion-coordinator-mcp

# Restart container
docker restart hyperion-coordinator-mcp
```

### MCP Server Not Appearing in Claude Code

**Solution:**
1. Verify container is running: `docker ps | grep hyperion-coordinator-mcp`
2. Check settings.json syntax is valid
3. **RESTART Claude Code** (this is critical!)
4. Check Claude Code logs for MCP connection errors

### MongoDB Connection Errors

**Check logs:**
```bash
docker logs hyperion-coordinator-mcp | grep -i mongo
```

**Expected output:**
```
Successfully connected to MongoDB Atlas
```

If you see connection errors, check:
1. MongoDB URI is correct in `.env` or `docker-compose.mcp-only.yml`
2. Network connectivity to MongoDB Atlas
3. MongoDB credentials are valid

### "method is invalid during session initialization"

This is **NORMAL** when testing with raw JSON-RPC. The MCP server requires proper initialization handshake.

Claude Code handles initialization automatically - just use the tools through Claude Code after restarting.

---

## ⚙️ Configuration

### Environment Variables

Edit `/Users/maxmednikov/MaxSpace/dev-squad/.env`:

```bash
# MongoDB Configuration
MONGODB_URI=mongodb+srv://dev:fvOKzv9enD8CSVwD@devdb.yqf8f8r.mongodb.net/?retryWrites=true&w=majority&appName=devDB
MONGODB_DATABASE=coordinator_db_max

# Logging
LOG_LEVEL=info
```

### Change MongoDB Database

```bash
# Edit .env
MONGODB_DATABASE=your_database_name

# Rebuild container
docker-compose -f coordinator/docker-compose.mcp-only.yml up -d --build
```

### Use Different MongoDB Cluster

```bash
# Edit .env
MONGODB_URI=mongodb+srv://user:pass@your-cluster.mongodb.net/...

# Rebuild container
docker-compose -f coordinator/docker-compose.mcp-only.yml up -d --build
```

---

## 🐳 Docker Architecture

```
┌─────────────────────────────────────┐
│     Claude Code CLI                 │
│     (macOS Host)                    │
└───────────────┬─────────────────────┘
                │ docker exec -i
                ▼
┌─────────────────────────────────────┐
│  Container: hyperion-coordinator-mcp│
│  ┌───────────────────────────────┐  │
│  │ MCP Server (stdio)            │  │
│  │ - 9 tools                     │  │
│  │ - 2 resources                 │  │
│  │ - MongoDB connection          │  │
│  └───────────────────────────────┘  │
└───────────────┬─────────────────────┘
                │ MongoDB Driver
                ▼
┌─────────────────────────────────────┐
│   MongoDB Atlas                     │
│   Database: coordinator_db_max      │
└─────────────────────────────────────┘
```

**Benefits:**
- ✅ Isolated environment
- ✅ Easy updates (rebuild container)
- ✅ Consistent across machines
- ✅ No Go installation required on host
- ✅ Container auto-restarts on failure

---

## 📝 Next Steps

1. ✅ Container running
2. ✅ Claude Code configured
3. ✅ **RESTART Claude Code** (if you haven't already)
4. 🎯 Test MCP tools in Claude Code
5. 📖 Read [SPECIFICATION.md](./SPECIFICATION.md) for tool reference
6. 🚀 Start using coordinator for task management

---

## 🔄 Updates

### Update MCP Server Code

```bash
# 1. Make code changes in coordinator/mcp-server/

# 2. Rebuild container
cd /Users/maxmednikov/MaxSpace/dev-squad/coordinator
docker-compose -f docker-compose.mcp-only.yml up -d --build

# 3. Restart Claude Code to pick up changes
```

### Update to Latest Version

```bash
# Pull latest code
cd /Users/maxmednikov/MaxSpace/dev-squad
git pull

# Rebuild container
cd coordinator
docker-compose -f docker-compose.mcp-only.yml up -d --build

# Restart Claude Code
```

---

## 📦 Full Stack Installation

If you want to run the **full UI + HTTP Bridge + MCP Server** stack:

```bash
cd /Users/maxmednikov/MaxSpace/dev-squad
docker-compose up -d
```

This will start:
- MCP Server (stdio)
- HTTP Bridge (port 8095)
- React UI (port 5173)

Access the web UI at: http://localhost:5173

---

## ✅ Verification Checklist

- [x] Container built successfully
- [x] Container is running (`docker ps`)
- [x] Claude Code settings.json updated
- [ ] **Claude Code restarted** ← DO THIS NOW!
- [ ] MCP tools available in Claude Code
- [ ] Successfully created test task
- [ ] MongoDB connection working

---

**Installation complete!** 🎉

Remember to **restart Claude Code** to see the new MCP server.
