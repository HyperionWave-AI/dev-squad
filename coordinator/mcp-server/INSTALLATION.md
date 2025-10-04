# Hyperion Coordinator MCP - Installation Guide

Multiple installation options to fit different user needs and technical comfort levels.

---

## 🚀 Quick Install (Recommended)

### Option 1: NPM Global Install (Easiest)

**Prerequisites:**
- Node.js 18+ (already installed for Claude Code)
- Go 1.25+ ([download](https://go.dev/dl/))

**Installation:**

```bash
npm install -g @hyperion/coordinator-mcp
```

**What happens:**
1. ✅ Downloads package from npm
2. ✅ Builds Go binary for your platform
3. ✅ Automatically configures Claude Code
4. ✅ Ready to use!

**Verification:**

Restart Claude Code, then test:

```typescript
mcp__hyperion-coordinator__coordinator_list_human_tasks({})
```

**Updates:**

```bash
npm update -g @hyperion/coordinator-mcp
```

**Uninstall:**

```bash
npm uninstall -g @hyperion/coordinator-mcp
```

---

## 🛠️ Manual Installation Options

### Option 2: NPX (No Global Install)

Run without installing:

```bash
npx @hyperion/coordinator-mcp
```

Then manually add to Claude Code config:

**macOS:**
`~/Library/Application Support/Claude/claude_desktop_config.json`

**Linux:**
`~/.config/Claude/claude_desktop_config.json`

**Windows:**
`%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "hyperion-coordinator": {
      "command": "npx",
      "args": ["-y", "@hyperion/coordinator-mcp"]
    }
  }
}
```

---

### Option 3: Pre-built Binary Download

**Download for your platform:**

- **macOS (Intel):** [hyperion-coordinator-mcp-darwin-amd64](https://github.com/yourorg/hyperion-coordinator-mcp/releases)
- **macOS (Apple Silicon):** [hyperion-coordinator-mcp-darwin-arm64](https://github.com/yourorg/hyperion-coordinator-mcp/releases)
- **Linux (x64):** [hyperion-coordinator-mcp-linux-amd64](https://github.com/yourorg/hyperion-coordinator-mcp/releases)
- **Windows (x64):** [hyperion-coordinator-mcp-windows-amd64.exe](https://github.com/yourorg/hyperion-coordinator-mcp/releases)

**Installation:**

```bash
# macOS/Linux
curl -L https://github.com/yourorg/hyperion-coordinator-mcp/releases/latest/download/hyperion-coordinator-mcp-$(uname -s)-$(uname -m) -o hyperion-coordinator-mcp
chmod +x hyperion-coordinator-mcp
sudo mv hyperion-coordinator-mcp /usr/local/bin/

# Windows (PowerShell)
Invoke-WebRequest -Uri "https://github.com/yourorg/hyperion-coordinator-mcp/releases/latest/download/hyperion-coordinator-mcp-windows-amd64.exe" -OutFile "hyperion-coordinator-mcp.exe"
Move-Item hyperion-coordinator-mcp.exe "$env:ProgramFiles\hyperion-coordinator-mcp.exe"
```

**Claude Code configuration:**

```json
{
  "mcpServers": {
    "hyperion-coordinator": {
      "command": "/usr/local/bin/hyperion-coordinator-mcp"
    }
  }
}
```

---

### Option 4: Build from Source

**Prerequisites:**
- Go 1.25+
- Git

**Steps:**

```bash
# Clone repository
git clone https://github.com/yourorg/hyperion-coordinator-mcp.git
cd hyperion-coordinator-mcp

# Install dependencies
go mod download

# Build binary
go build -o hyperion-coordinator-mcp main.go

# Make executable (macOS/Linux)
chmod +x hyperion-coordinator-mcp

# Add to PATH (optional)
sudo mv hyperion-coordinator-mcp /usr/local/bin/
```

**Auto-configure Claude Code:**

```bash
./add-to-claude-code.sh
```

**Manual configuration:**

Edit Claude Code config file and add:

```json
{
  "mcpServers": {
    "hyperion-coordinator": {
      "command": "/path/to/hyperion-coordinator-mcp",
      "env": {
        "MONGODB_URI": "your_mongodb_uri",
        "MONGODB_DATABASE": "coordinator_db"
      }
    }
  }
}
```

---

## 🐳 Docker Installation

### Option 5: Docker Container

**Prerequisites:**
- Docker installed

**Run as container:**

```bash
docker run -d \
  --name hyperion-coordinator-mcp \
  -e MONGODB_URI="your_mongodb_uri" \
  -e MONGODB_DATABASE="coordinator_db" \
  hyperion/coordinator-mcp:latest
```

**Claude Code configuration:**

```json
{
  "mcpServers": {
    "hyperion-coordinator": {
      "command": "docker",
      "args": ["exec", "-i", "hyperion-coordinator-mcp", "/app/hyperion-coordinator-mcp"]
    }
  }
}
```

**Build Docker image:**

```bash
cd hyperion-coordinator-mcp
docker build -t hyperion/coordinator-mcp:latest .
```

**Dockerfile:**

```dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o hyperion-coordinator-mcp main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /build/hyperion-coordinator-mcp .

ENTRYPOINT ["/app/hyperion-coordinator-mcp"]
```

---

## ⚙️ Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MONGODB_URI` | Dev cluster URI | MongoDB connection string |
| `MONGODB_DATABASE` | `coordinator_db` | Database name |
| `LOG_LEVEL` | `info` | Logging level (debug/info/warn/error) |

### MongoDB Setup

**Option A: Use Default Dev Cluster**

The MCP server includes a default MongoDB Atlas dev cluster - no setup needed!

**Option B: Use Your Own MongoDB**

```bash
# Local MongoDB
export MONGODB_URI="mongodb://localhost:27017"

# MongoDB Atlas (free tier)
export MONGODB_URI="mongodb+srv://username:password@cluster.mongodb.net/?retryWrites=true&w=majority"
```

Create MongoDB Atlas account: https://www.mongodb.com/cloud/atlas/register

---

## 🧪 Testing Installation

### Verify MCP Server is Running

```bash
# Test with echo
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | hyperion-coordinator-mcp
```

**Expected output:**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "tools": [
      {
        "name": "coordinator_list_human_tasks",
        "description": "List all human tasks..."
      },
      ...
    ]
  }
}
```

### Test in Claude Code

```typescript
// List available tools
mcp__hyperion-coordinator__coordinator_list_human_tasks({})

// Create a test task
mcp__hyperion-coordinator__coordinator_create_human_task({
  prompt: "Test task"
})

// Query knowledge
mcp__hyperion-coordinator__coordinator_query_knowledge({
  collection: "technical-knowledge",
  query: "test",
  limit: 5
})
```

---

## 🔧 Troubleshooting

### "command not found: hyperion-coordinator-mcp"

**Solution:** Binary not in PATH. Either:
1. Use full path: `/usr/local/bin/hyperion-coordinator-mcp`
2. Add to PATH: `export PATH=$PATH:/path/to/binary`

### "Go is not installed"

**Solution:** Install Go from https://go.dev/dl/

```bash
# Verify installation
go version
# Should output: go version go1.25.x ...
```

### "Failed to connect to MongoDB"

**Solution:** Check MongoDB URI and network connectivity:

```bash
# Test MongoDB connection
mongosh "your_mongodb_uri"
```

### "MCP server not appearing in Claude Code"

**Solution:**

1. Restart Claude Code completely
2. Check config file location matches your platform
3. Verify JSON syntax in config file
4. Check Claude Code logs for errors

**macOS logs:**
```bash
tail -f ~/Library/Logs/Claude/mcp*.log
```

### "Permission denied" errors

**Solution:**

```bash
# Make binary executable
chmod +x hyperion-coordinator-mcp

# If installing to /usr/local/bin
sudo mv hyperion-coordinator-mcp /usr/local/bin/
```

---

## 📦 Installation Comparison

| Method | Ease of Use | Auto-Update | Isolation | Best For |
|--------|-------------|-------------|-----------|----------|
| **NPM Global** | ⭐⭐⭐⭐⭐ | ✅ `npm update` | ❌ Global | Most users |
| **NPX** | ⭐⭐⭐⭐ | ✅ Always latest | ✅ Per-run | Testing |
| **Pre-built Binary** | ⭐⭐⭐ | ❌ Manual | ❌ Global | Air-gapped systems |
| **Build from Source** | ⭐⭐ | ❌ Git pull | ❌ Global | Developers |
| **Docker** | ⭐⭐⭐ | ✅ Docker pull | ✅ Container | Containerized environments |

---

## 🆘 Support

- **Documentation:** https://github.com/yourorg/hyperion-coordinator-mcp
- **Issues:** https://github.com/yourorg/hyperion-coordinator-mcp/issues
- **Discussions:** https://github.com/yourorg/hyperion-coordinator-mcp/discussions

---

## 📝 Next Steps

After installation:

1. ✅ Read the [Getting Started Guide](./README.md#usage-examples)
2. ✅ Review [MCP Tool Reference](./HYPERION_COORDINATOR_MCP_REFERENCE.md)
3. ✅ Check [Agent Integration Guide](./CLAUDE.md)
4. ✅ Explore [Example Workflows](./examples/)

---

## 🔄 Migration from Manual Install

If you previously installed manually, migrate to npm:

```bash
# Remove old binary
sudo rm /usr/local/bin/hyperion-coordinator-mcp

# Install via npm
npm install -g @hyperion/coordinator-mcp

# Config is automatically updated!
```

---

**Questions?** Open an issue or discussion on GitHub!
