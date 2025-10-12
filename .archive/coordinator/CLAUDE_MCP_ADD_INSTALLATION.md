# Hyperion Coordinator MCP - Installation with `claude mcp add`

Official installation guide using Claude Code's `claude mcp add` command.

---

## ✅ Quick Install (Recommended)

### One-Command Installation

```bash
cd /Users/maxmednikov/MaxSpace/dev-squad/coordinator
./install-mcp-to-claude-code.sh
```

This script will:
1. ✅ Check Docker is running
2. ✅ Start the MCP container (if not running)
3. ✅ Verify health endpoint
4. ✅ Run `claude mcp add` command
5. ✅ Display next steps

---

## 🔧 Manual Installation

If you prefer to run the commands manually:

### Step 1: Start the Docker Container

```bash
cd /Users/maxmednikov/MaxSpace/dev-squad/coordinator
docker-compose -f docker-compose.mcp-only.yml up -d
```

### Step 2: Verify Container is Running

```bash
# Check container status
docker ps | grep hyper-mcp

# Test health endpoint
curl http://localhost:7778/health
```

**Expected**: Container running and health check returns `OK`

### Step 3: Add to Claude Code

```bash
claude mcp add --transport http hyper http://localhost:7778/mcp
```

**What this does**:
- `--transport http` - Uses HTTP transport instead of stdio
- `hyper` - Server name (will appear in `/mcp` list)
- `http://localhost:7778/mcp` - MCP server endpoint URL

### Step 4: Restart Claude Code

**CRITICAL**: You must restart Claude Code for the server to appear.

```bash
# Quit Claude Code completely, then restart
```

### Step 5: Verify Installation

After restarting Claude Code:

```bash
/mcp
```

**Expected**: `hyper` appears in the list with 9 tools

---

## 📋 Command Breakdown

```bash
claude mcp add --transport http hyper http://localhost:7778/mcp
│              │              │                      │
│              │              │                      └─ MCP endpoint URL
│              │              └─ Server name (identifier)
│              └─ Transport type (http vs stdio)
└─ Claude Code CLI command
```

### Parameters Explained

| Parameter | Value | Description |
|-----------|-------|-------------|
| `--transport` | `http` | Use HTTP streaming transport (required for Docker) |
| Server name | `hyper` | Identifier for the MCP server |
| URL | `http://localhost:7778/mcp` | HTTP endpoint for MCP protocol |

---

## 🎯 Why HTTP Transport?

### stdio vs HTTP

**stdio (default)**:
- ❌ Doesn't work with `docker exec` (creates new process each time)
- ❌ Can't maintain persistent connection
- ❌ Server never appears in `/mcp` list

**HTTP (our solution)**:
- ✅ Container runs continuously with HTTP server
- ✅ Claude Code connects via URL
- ✅ Persistent connection maintained
- ✅ Works reliably with Docker

---

## 📊 What Gets Installed

### MCP Server Details
- **Name**: `hyper`
- **Transport**: HTTP
- **Endpoint**: http://localhost:7778/mcp
- **Health Check**: http://localhost:7778/health
- **Storage**: MongoDB Atlas (coordinator_db)

### Available Tools (9 total)

#### Task Management (6 tools)
1. `coordinator_create_human_task` - Create user-initiated task
2. `coordinator_create_agent_task` - Create agent task with TODOs
3. `coordinator_list_human_tasks` - List all human tasks
4. `coordinator_list_agent_tasks` - List agent tasks (filterable)
5. `coordinator_update_task_status` - Update task status/notes
6. `coordinator_update_todo_status` - Update TODO item status

#### Knowledge Management (2 tools)
7. `coordinator_upsert_knowledge` - Store knowledge in collections
8. `coordinator_query_knowledge` - Search knowledge by query

#### Maintenance (1 tool)
9. `coordinator_clear_task_board` - Clear all tasks/knowledge

### Resources (2 URIs)
- `hyperion://task/human/{taskId}` - Human task details
- `hyperion://task/agent/{agentName}/{taskId}` - Agent task details

---

## 🔍 Verification Checklist

After installation, verify each step:

- [ ] Docker container running: `docker ps | grep hyper-mcp`
- [ ] Health check works: `curl http://localhost:7778/health` → `OK`
- [ ] MCP endpoint responds: `curl http://localhost:7778/mcp` (should return MCP response)
- [ ] Claude Code restarted
- [ ] `/mcp` shows `hyper`
- [ ] Test tool works: `mcp__hyper__coordinator_list_human_tasks({})`

---

## 🛠️ Management Commands

### View MCP Servers
```bash
claude mcp list
```

Shows all configured MCP servers including `hyper`

### Get Server Details
```bash
claude mcp get hyper
```

Shows configuration details for the server

### Remove Server
```bash
claude mcp remove hyper
```

Removes the server from Claude Code configuration

### Update Server Configuration
```bash
# Remove old configuration
claude mcp remove hyper

# Add with new settings
claude mcp add --transport http hyper http://localhost:7778/mcp
```

---

## 🐳 Docker Management

### View Container Logs
```bash
docker logs hyper-mcp -f
```

### Check HTTP Server Started
```bash
docker logs hyper-mcp | grep "HTTP server listening"
```

**Expected output**:
```
INFO	HTTP server listening	{"address": ":7778", "mcp_endpoint": "/mcp", "health_endpoint": "/health"}
```

### Restart Container
```bash
docker restart hyper-mcp
```

### Stop Container
```bash
docker-compose -f coordinator/docker-compose.mcp-only.yml stop
```

### Start Container
```bash
docker-compose -f coordinator/docker-compose.mcp-only.yml start
```

### Rebuild Container (after code changes)
```bash
cd coordinator
docker-compose -f docker-compose.mcp-only.yml up -d --build
```

---

## 🔧 Troubleshooting

### "claude: command not found"

**Problem**: Claude Code CLI not in PATH

**Solution**:
```bash
# Add to ~/.zshrc or ~/.bashrc
export PATH="$PATH:/Applications/Claude.app/Contents/Resources/app/bin"

# Reload shell
source ~/.zshrc
```

### Container Not Running

**Check status**:
```bash
docker ps -a | grep hyper-mcp
```

**Start container**:
```bash
cd /Users/maxmednikov/MaxSpace/dev-squad/coordinator
docker-compose -f docker-compose.mcp-only.yml up -d
```

**Check logs for errors**:
```bash
docker logs hyper-mcp
```

### Health Check Fails

**Test manually**:
```bash
curl -v http://localhost:7778/health
```

**If connection refused**:
- Container might not be running: `docker start hyper-mcp`
- Port might be in use: `lsof -i :7778`
- Check logs: `docker logs hyper-mcp`

### Server Not in `/mcp` List

**Common causes**:
1. ❌ Forgot to restart Claude Code → **Restart now**
2. ❌ Container not running → `docker start hyper-mcp`
3. ❌ Wrong URL in command → Check URL is `http://localhost:7778/mcp`
4. ❌ Health check failing → Test with curl

**Debug steps**:
```bash
# 1. Verify container is running
docker ps | grep hyper-mcp

# 2. Test health endpoint
curl http://localhost:7778/health

# 3. Test MCP endpoint
curl -X POST http://localhost:7778/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}'

# 4. Check Claude Code logs
tail -f ~/Library/Logs/Claude/mcp.log | grep hyperion

# 5. List configured servers
claude mcp list
```

### MongoDB Connection Errors

**Check logs**:
```bash
docker logs hyper-mcp | grep MongoDB
```

**Expected**:
```
INFO	Successfully connected to MongoDB Atlas
```

**If connection fails**:
- Check MongoDB Atlas cluster is accessible
- Verify credentials in `.env` file
- Test network connectivity

---

## ⚙️ Configuration

### Environment Variables

Edit `/Users/maxmednikov/MaxSpace/dev-squad/.env`:

```bash
# MongoDB Configuration
MONGODB_URI=mongodb+srv://dev:fvOKzv9enD8CSVwD@devdb.yqf8f8r.mongodb.net/?retryWrites=true&w=majority&appName=devDB
MONGODB_DATABASE=coordinator_db

# MCP Server Configuration (DO NOT CHANGE for HTTP mode)
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
docker restart hyper-mcp

# 3. No need to re-add to Claude Code
```

### Change MCP Port

```bash
# 1. Edit docker-compose.mcp-only.yml
ports:
  - "7779:7779"
environment:
  - MCP_PORT=7779

# 2. Rebuild container
docker-compose -f coordinator/docker-compose.mcp-only.yml up -d --build

# 3. Remove and re-add to Claude Code
claude mcp remove hyper
claude mcp add --transport http hyper http://localhost:7779/mcp

# 4. Restart Claude Code
```

---

## 📚 Documentation

- **MCP Specification**: [SPECIFICATION.md](./SPECIFICATION.md)
- **Full Stack Setup**: [FULL_STACK_SETUP.md](./FULL_STACK_SETUP.md)
- **Docker Installation**: [DOCKER_INSTALLATION.md](./DOCKER_INSTALLATION.md)
- **Integration Guide**: [INTEGRATION_COMPLETE.md](./INTEGRATION_COMPLETE.md)

---

## 🎓 Usage Examples

### Create a Human Task

```typescript
mcp__hyper__coordinator_create_human_task({
  prompt: "Review codebase for security vulnerabilities"
})
```

### List Tasks

```typescript
// List all human tasks
mcp__hyper__coordinator_list_human_tasks({})

// List agent tasks for specific agent
mcp__hyper__coordinator_list_agent_tasks({
  agentName: "backend-services-specialist"
})
```

### Update Task Status

```typescript
mcp__hyper__coordinator_update_task_status({
  taskId: "task-uuid-here",
  status: "in_progress",
  notes: "Started working on security review"
})
```

### Store Knowledge

```typescript
mcp__hyper__coordinator_upsert_knowledge({
  collection: "technical-knowledge",
  text: "JWT tokens should use HS256 algorithm with 512-bit keys",
  metadata: {
    category: "security",
    topic: "authentication"
  }
})
```

### Query Knowledge

```typescript
mcp__hyper__coordinator_query_knowledge({
  collection: "technical-knowledge",
  query: "JWT security best practices",
  limit: 5
})
```

---

## ✅ Success Criteria

Installation is successful when:

1. ✅ `docker ps` shows `hyper-mcp` running
2. ✅ `curl http://localhost:7778/health` returns `OK`
3. ✅ `claude mcp list` shows `hyper`
4. ✅ `/mcp` in Claude Code shows the server
5. ✅ `coordinator_list_human_tasks({})` returns response (may be empty)

---

## 🚀 Quick Start Summary

```bash
# 1. Install (one command)
cd /Users/maxmednikov/MaxSpace/dev-squad/coordinator
./install-mcp-to-claude-code.sh

# 2. Restart Claude Code

# 3. Verify
/mcp

# 4. Test
mcp__hyper__coordinator_list_human_tasks({})
```

---

**Installation complete!** 🎉

The `hyper` MCP server is now available in Claude Code with 9 tools for task coordination and knowledge management.
