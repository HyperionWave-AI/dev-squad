# 🚀 Hyperion Coordinator MCP

> **Model Context Protocol server for AI agent task coordination**

[![Docker](https://img.shields.io/badge/Docker-ready-blue.svg)](https://www.docker.com/)
[![Go](https://img.shields.io/badge/Go-1.25-blue.svg)](https://golang.org/)
[![MCP](https://img.shields.io/badge/MCP-compatible-green.svg)](https://modelcontextprotocol.io/)
[![MongoDB](https://img.shields.io/badge/MongoDB-Atlas-green.svg)](https://www.mongodb.com/cloud/atlas)

**Orchestrate AI agents with hierarchical task management, real-time progress tracking, and intelligent knowledge coordination.**

## 🎯 What is Hyperion Coordinator?

A production-ready MCP server that enables AI agents to coordinate complex workflows through:

- **Hierarchical Tasks** - Human tasks → Agent tasks → TODO tracking
- **Knowledge Base** - Store and query coordination knowledge with semantic search
- **MCP Native** - 33 tools for complete task lifecycle management
- **MongoDB Persistence** - Cloud-based storage with real-time sync
- **Kanban UI** - Visual progress tracking with drag-and-drop

Perfect for multi-agent systems, autonomous coding agents, and AI workflow orchestration.

## ⚡ Quick Start

### Option 1: Desktop App (Easiest - Native UI!)

```bash
# Build and run as native desktop app
make desktop

# OR build for distribution
make desktop-build
# Creates .dmg (macOS), .msi (Windows), .AppImage (Linux)
```

**🖥️ Native desktop app with system tray!** See [desktop-app/README.md](./desktop-app/README.md) for details.

### Option 2: Native Binary (Single File!)

```bash
# Build self-contained binary with embedded UI
make native

# Configure (one-time setup)
cp .env.native .env.native
# Edit .env.native with your MongoDB URI and settings

# Run HTTP server with UI
make run
# Access UI: http://localhost:7095/ui

# OR configure for Claude Code (MCP stdio mode)
make configure-native
# Restart Claude Code - hyper MCP tools are now available!
```

**✨ Zero Docker, zero dependencies, single binary!** See [README-NATIVE.md](./README-NATIVE.md) for details.

### Option 3: Docker (Full Stack)

```bash
# 1. Clone and install
git clone <repository-url>
cd hyper-mcp
./install.sh

# 2. Start all services (HTTP Bridge + UI)
docker-compose up -d

# 3. Access services
# - UI Dashboard: http://localhost:5173
# - HTTP API: http://localhost:7095/health

# 4. For Claude Code integration
# Restart Claude Code - The MCP server is now available!
```

**That's it!** All services are now running in Docker with proper CORS configuration.

## 📚 Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
- [MCP Tools](#mcp-tools)
- [Development](#development)
- [Deployment](#deployment)
- [Documentation](#documentation)

## ✨ Features

**🤖 Multi-Agent Coordination**
- Task decomposition (human → agent workflows)
- TODO-level progress tracking
- Role-based agent assignment
- Status tracking (pending/in_progress/completed/blocked)

**📊 Kanban Dashboard**
- 4-column board with drag-and-drop
- Priority color-coding
- Real-time updates
- Responsive design (desktop/tablet/mobile)

**🧠 Knowledge Management**
- Vector-based semantic search
- Task-specific collections
- MongoDB + Qdrant integration
- Context preservation across agents

**🔧 MCP Integration**
- 36 tools (coordinator, code index, knowledge, filesystem, discovery, server management)
- 12 dynamic resources (`hyperion://task/*`, `hyperion://docs/*`, etc.)
- 7 AI assistance prompts
- HTTP + stdio transports
- Official MCP Go SDK v0.3.0

---

## 🛠️ Technology Stack

### Backend

| Component | Technology | Version |
|-----------|-----------|---------|
| **Language** | Go | 1.25 |
| **MCP SDK** | `modelcontextprotocol/go-sdk` | 0.3.0 |
| **Database** | MongoDB Atlas | Cloud |
| **Vector Store** | Qdrant | Cloud |
| **HTTP Server** | Gin | Latest |
| **Testing** | Go Testing + Benchmarks | - |

### Frontend

| Component | Technology | Version |
|-----------|-----------|---------|
| **Framework** | React | 19.1.1 |
| **Language** | TypeScript | 5.8.3 |
| **UI Library** | Material-UI (MUI) | 7.3.2 |
| **Build Tool** | Vite | 7.1.7 |
| **Drag & Drop** | @hello-pangea/dnd | 18.0.1 |
| **Testing** | Playwright | 1.55.1 |

### Infrastructure

- **HTTP Bridge**: Go-based MCP-to-HTTP adapter (Port 7095)
- **MCP Server**: Stdio-based protocol server
- **Development**: Single-script startup for full stack

---

## 🏗️ Architecture

### System Components

```
┌─────────────────────────────────────────────────────────┐
│                   Client Applications                    │
│            (Claude Code, Web UI, API Clients)           │
└────────────────────┬────────────────────────────────────┘
                     │ HTTP/REST
                     ▼
┌─────────────────────────────────────────────────────────┐
│              MCP HTTP Bridge (Port 7095)                │
│  • CORS handling for web clients                        │
│  • HTTP → stdio request translation                     │
│  • Concurrent request routing                           │
│  • Health monitoring                                    │
└────────────────────┬────────────────────────────────────┘
                     │ stdio (JSON-RPC)
                     ▼
┌─────────────────────────────────────────────────────────┐
│       Unified Hyper Binary (bin/hyper)                  │
│  ┌───────────────────────────────────────────────────┐  │
│  │ 36 MCP Tools                                      │  │
│  │ • Coordinator (19): Tasks, TODOs, Knowledge       │  │
│  │ • Code Index (5): Semantic code search            │  │
│  │ • Qdrant (2): Vector knowledge storage            │  │
│  │ • Filesystem (4): File operations, bash, patches  │  │
│  │ • Discovery (3): Dynamic tool discovery           │  │
│  │ • Server Mgmt (3): MCP server management          │  │
│  │                                                   │  │
│  │ 12 MCP Resources (docs, workflow, metrics)        │  │
│  │ 7 MCP Prompts (planning, knowledge, diagnostics)  │  │
│  └───────────────────────────────────────────────────┘  │
└──────────────┬────────────────────┬─────────────────────┘
               │                    │
               ▼                    ▼
    ┌─────────────────┐  ┌─────────────────┐
    │ MongoDB Atlas   │  │ Qdrant Vector   │
    │ • human_tasks   │  │ • knowledge     │
    │ • agent_tasks   │  │   collections   │
    └─────────────────┘  └─────────────────┘
```

### Data Flow

1. **Web UI** → HTTP request (GET/POST)
2. **HTTP Bridge** → Translate to MCP JSON-RPC via stdin
3. **MCP Server** → Process request, query MongoDB/Qdrant
4. **MCP Server** → Return JSON-RPC response via stdout
5. **HTTP Bridge** → Translate to HTTP JSON response
6. **Web UI** → Update Kanban board

---

## 🚀 Installation

### Option 1: Docker (Recommended)

**Prerequisites:** Docker & Docker Compose ([Install Docker](https://docs.docker.com/get-docker/))

```bash
# Clone repository
git clone <repository-url>
cd hyper-mcp

# Install and start
./install.sh
docker-compose up -d

# Verify
docker-compose logs -f hyper-mcp

# Restart Claude Code
# MCP server is now available!
```

**What you get:**
- ✅ Automatic Claude Code configuration (macOS/Linux)
- ✅ MongoDB Atlas connection (dev cluster included)
- ✅ All 9 MCP tools ready to use
- ✅ Auto-restart on failure

**Common commands:**
```bash
docker-compose up -d                          # Start all services
docker-compose down                           # Stop all services
docker-compose logs -f                        # View all logs
docker-compose logs -f hyperion-http-bridge   # View bridge logs
docker-compose logs -f hyperion-ui            # View UI logs
docker-compose restart                        # Restart all services
docker-compose build                          # Rebuild images
```

**Services running:**
- `hyperion-http-bridge` - HTTP API + MCP Server (port 7095)
- `hyperion-ui` - React dashboard (port 5173)

### Option 2: Native (Development)

**Prerequisites:** Go 1.25+, Node.js 18+, MongoDB Atlas

```bash
# Clone and setup
git clone <repository-url>
cd dev-squad
export MONGODB_URI="mongodb+srv://user:pass@cluster.mongodb.net/coordinator_db"

# Build unified binary
make build

# Run in HTTP mode (REST API + UI + MCP HTTP)
./bin/hyper --mode=http

# OR run in MCP stdio mode (for Claude Code)
./bin/hyper --mode=mcp

# OR run in dual mode (both HTTP and MCP)
./bin/hyper --mode=both
```

**Service URLs:**
- MCP Server: stdio (for MCP clients)
- HTTP Bridge: http://localhost:7095
- React UI: http://localhost:5173

## ⚙️ Configuration

### Environment Variables

Edit `.env` (created by `install.sh`):

```bash
# MongoDB (required)
MONGODB_URI=mongodb+srv://dev:pass@cluster.mongodb.net/?retryWrites=true&w=majority
MONGODB_DATABASE=coordinator_db

# Qdrant (optional - for knowledge features)
QDRANT_URL=https://your-cluster.cloud.qdrant.io
QDRANT_API_KEY=your-api-key

# Logging
LOG_LEVEL=info  # debug, info, warn, error
```

**After editing `.env`:**
```bash
docker-compose restart  # Docker
# or
./start-coordinator.sh  # Native
```

### Custom MongoDB

1. Create MongoDB Atlas cluster
2. Get connection string
3. Update `MONGODB_URI` in `.env`
4. Restart server

## 📖 Usage

### Using with Claude Code

After installation and restarting Claude Code, the MCP server provides 9 tools:

```javascript
// Create human task
coordinator_create_human_task({
  prompt: "Build user authentication system"
})

// Create agent task
coordinator_create_agent_task({
  humanTaskId: "task-id",
  agentName: "backend-specialist",
  role: "JWT middleware",
  todos: ["Design schema", "Implement", "Test"]
})

// Update status
coordinator_update_task_status({
  taskId: "task-id",
  status: "in_progress",
  notes: "Started implementation"
})
```

### Using the HTTP API

The HTTP bridge (port 7095) provides REST access:

```bash
# List tools
curl http://localhost:7095/api/mcp/tools

# Call tool
curl -X POST http://localhost:7095/api/mcp/tools/call \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: req-1" \
  -d '{
    "name": "coordinator_list_human_tasks",
    "arguments": {}
  }'

# Read resource
curl "http://localhost:7095/api/mcp/resources/read?uri=hyperion://task/human/abc-123"
```

### Using the Kanban UI

Visit http://localhost:5173 for visual task management:

- **Drag & drop** tasks between columns
- **Click** cards to view details
- **Real-time** updates every 5 seconds
- **Filter** by priority, agent, status

## 🔧 MCP Tools

The unified hyper binary provides **36 MCP tools** across 6 categories:

### Coordinator Tools (19 tools)
Task management, knowledge, and coordination:
- `coordinator_create_human_task` - Create user-level task
- `coordinator_create_agent_task` - Assign task to specialist agent
- `coordinator_list_human_tasks` - List all human tasks
- `coordinator_list_agent_tasks` - List agent tasks (with pagination)
- `coordinator_get_agent_task` - Get full task details (untruncated)
- `coordinator_update_task_status` - Update task progress
- `coordinator_update_todo_status` - Mark TODO items complete
- `coordinator_add_task_prompt_notes` - Add human guidance to tasks
- `coordinator_update_task_prompt_notes` - Update task guidance
- `coordinator_clear_task_prompt_notes` - Remove task guidance
- `coordinator_add_todo_prompt_notes` - Add guidance to TODOs
- `coordinator_update_todo_prompt_notes` - Update TODO guidance
- `coordinator_clear_todo_prompt_notes` - Remove TODO guidance
- `coordinator_upsert_knowledge` - Store knowledge in MongoDB
- `coordinator_query_knowledge` - Query task-specific knowledge
- `coordinator_get_popular_collections` - Get most-used collections
- `coordinator_clear_task_board` - Clear all tasks (destructive)
- `list_subagents` - Query available specialist agents
- `set_current_subagent` - Associate subagent with chat

### Code Indexing Tools (5 tools)
Semantic code search and indexing:
- `code_index_add_folder` - Add folder to semantic index
- `code_index_remove_folder` - Remove folder from index
- `code_index_scan` - Scan folder for changes
- `code_index_search` - Natural language code search
- `code_index_status` - Get indexing status

### Knowledge Tools (2 tools)
Vector-based knowledge storage:
- `knowledge_find` - Semantic similarity search
- `knowledge_store` - Store with embeddings

### Filesystem Tools (4 tools)
File operations and command execution:
- `file_read` - Read files with chunked streaming
- `file_write` - Write files with chunked streaming
- `bash` - Execute bash commands with streaming
- `apply_patch` - Apply unified diff patches

### Discovery Tools (3 tools)
Dynamic tool discovery:
- `discover_tools` - Natural language tool search
- `get_tool_schema` - Get tool JSON schema
- `execute_tool` - Execute tools dynamically

### Server Management Tools (3 tools)
External MCP server management:
- `mcp_add_server` - Register external MCP servers and discover tools
- `mcp_rediscover_server` - Refresh tools from registered servers
- `mcp_remove_server` - Remove servers and cleanup tool data

**📖 Complete reference:** [HYPERION_COORDINATOR_MCP_REFERENCE.md](./HYPERION_COORDINATOR_MCP_REFERENCE.md)

### Example Workflow

```javascript
// 1. Create human task
const humanTaskId = await coordinator_create_human_task({
  prompt: "Implement user authentication"
})

// 2. Create agent tasks
await coordinator_create_agent_task({
  humanTaskId,
  agentName: "backend-specialist",
  role: "JWT middleware implementation",
  todos: [
    "Design JWT schema",
    "Implement token generation",
    "Add validation middleware",
    "Write unit tests"
  ]
})

await coordinator_create_agent_task({
  humanTaskId,
  agentName: "frontend-specialist",
  role: "Login UI",
  todos: [
    "Create login form component",
    "Add authentication context",
    "Handle token storage"
  ]
})

// 3. Store implementation knowledge
await coordinator_upsert_knowledge({
  collection: `task:hyperion://task/human/${humanTaskId}`,
  text: "Using bcrypt for password hashing with cost factor 12. JWT tokens expire after 24 hours.",
  metadata: {
    agentName: "backend-specialist",
    taskId: humanTaskId
  }
})

// 4. Query knowledge later
const results = await coordinator_query_knowledge({
  collection: `task:hyperion://task/human/${humanTaskId}`,
  query: "password security approach",
  limit: 5
})
```

---

## 📁 Project Structure

```
dev-squad/
├── hyper/                       # ✅ UNIFIED GO BINARY (ACTIVE)
│   ├── cmd/coordinator/         # Main entry point
│   │   └── main.go              # Unified binary with HTTP + MCP
│   ├── internal/                # Internal packages
│   │   ├── mcp/                 # MCP protocol layer
│   │   │   ├── handlers/        # 36 MCP tools
│   │   │   │   ├── tools.go                # Coordinator (19)
│   │   │   │   ├── code_tools.go           # Code index (5)
│   │   │   │   ├── qdrant_tools.go         # Knowledge (2)
│   │   │   │   ├── filesystem_tools.go     # Filesystem (4)
│   │   │   │   └── tools_discovery.go      # Discovery (3) + Server Mgmt (3)
│   │   │   ├── storage/         # MongoDB + Qdrant clients
│   │   │   └── embeddings/      # Embedding providers
│   │   ├── server/              # HTTP server
│   │   │   └── http_server.go   # REST API + MCP HTTP + UI
│   │   ├── ai-service/          # AI chat streaming
│   │   └── services/            # Business logic
│   ├── embed/                   # Embedded UI bundle
│   │   └── ui/                  # Production UI assets
│   └── go.mod                   # Go dependencies
│
├── coordinator/                 # ✅ ACTIVE UI SOURCE
│   ├── ui/                      # React frontend (source)
│   │   ├── src/
│   │   │   ├── App.tsx          # Main application
│   │   │   ├── components/      # React components
│   │   │   ├── services/        # API clients
│   │   │   └── types/           # TypeScript types
│   │   ├── tests/               # Playwright E2E tests (109 tests)
│   │   ├── package.json         # Node dependencies
│   │   └── vite.config.ts       # Build configuration
│   └── *.md                     # Documentation
│
├── bin/                         # ✅ COMPILED BINARY
│   └── hyper                    # 17MB unified binary (http|mcp|both)
│
├── scripts/                     # Development scripts
│   ├── dev-hot.sh               # Full stack hot reload
│   ├── dev-native.sh            # Native dev mode
│   └── air-build.sh             # Air build script
│
├── .air.toml                    # Air hot reload config
├── Makefile                     # Build targets
├── CLAUDE.md                    # Agent coordination guide
├── README.md                    # This file
└── HYPERION_COORDINATOR_MCP_REFERENCE.md  # MCP tool reference
```

**⚠️ Deprecated (Archived in hyper/.archived/coordinator-old/):**
- Old coordinator binary (24MB, http only)
- Old MCP server (12MB, separate process)
- Old HTTP bridge (12MB, subprocess)
- Duplicate internal packages

**✅ Use unified binary only:** `bin/hyper` (17MB, all features, single process)

## 🔧 Development

### Building from Source

**Unified Binary:**
```bash
# Build from hyper/cmd/coordinator
make build
# Output: bin/hyper (17MB)

# OR use native build script
./build-native.sh
```

**Frontend (UI source):**
```bash
cd coordinator/ui
npm install
npm run build
# Output: hyper/embed/ui/ (embedded in binary)
```

**Docker:**
```bash
docker-compose build
```

### Development Modes

```bash
# Hot reload (Go only)
make dev

# Full stack hot reload (Go + UI)
make dev-hot

# Native development
make run-native
```

### Code Quality Standards

- **Go**: Handlers ≤300 lines, Services ≤400, coverage >60%
- **TypeScript**: ESLint + strict mode, React Hooks compliance
- **Testing**: Unit + integration + E2E (109 Playwright tests)

---

## 🧪 Testing

### Test Coverage

| Component | Coverage | Test Count |
|-----------|----------|------------|
| **HTTP Bridge** | 60.3% | 9 unit tests + 6 benchmarks |
| **React UI** | ~85% | 109 Playwright tests (8 suites) |
| **MCP Server** | TBD | Integration tests pending |

### Critical Test Scenarios

✅ **Concurrent Requests**: 20+ simultaneous HTTP requests
✅ **Response Routing**: Out-of-order response handling
✅ **Drag & Drop**: Kanban card movement across columns
✅ **Accessibility**: WCAG 2.1 AA compliance
✅ **Visual Regression**: Component rendering validation
✅ **Memory Leaks**: Pending request cleanup
✅ **Error Handling**: MCP error propagation

### Performance Benchmarks

```
BenchmarkHighLoad-8              1000    1.2 ms/op
BenchmarkConcurrentToolCalls-8    500    2.5 ms/op
BenchmarkUIPollingSimulation-8    200    5.8 ms/op
BenchmarkStressTest-8              50   22.1 ms/op
```

---

## 🚢 Deployment

### Docker Deployment (Recommended)

**Production deployment with Docker:**

1. **Clone repository on server:**
   ```bash
   git clone <repository-url>
   cd hyper-mcp
   ```

2. **Configure production environment:**
   ```bash
   cp .env.example .env
   # Edit .env with production MongoDB URI and settings
   ```

3. **Build and start:**
   ```bash
   docker-compose up -d
   ```

4. **Set up reverse proxy** (nginx example):
   ```nginx
   location /mcp {
       proxy_pass http://localhost:7095;
       proxy_http_version 1.1;
       proxy_set_header Upgrade $http_upgrade;
       proxy_set_header Connection 'upgrade';
       proxy_set_header Host $host;
       proxy_cache_bypass $http_upgrade;
   }
   ```

5. **Enable monitoring:**
   ```bash
   docker-compose logs -f hyper-mcp
   ```

### Production Checklist

- [ ] Set production `MONGODB_URI` in `.env`
- [ ] Set `QDRANT_URL` and `QDRANT_API_KEY` (if using knowledge features)
- [ ] Configure CORS origins in docker-compose.yml if needed
- [ ] Set up reverse proxy (nginx/Caddy) for HTTPS
- [ ] Configure firewall rules (port 7095 internal only)
- [ ] Set up monitoring and logging (Docker logs)
- [ ] Configure backup strategy for MongoDB
- [ ] Set up automatic restarts: `restart: unless-stopped` (already in docker-compose.yml)
- [ ] Configure log rotation for Docker logs

### Native Binary Deployment

For deployment without Docker:

1. Build binaries: `go build` in both Go directories
2. Build frontend: `npm run build` in ui/
3. Set environment variables
4. Run binaries as systemd services

---

## 🤝 Contributing

Contributions are welcome! Please follow these guidelines:

1. **Fork the repository**
2. **Create a feature branch**: `git checkout -b feature/amazing-feature`
3. **Run tests**: Ensure all tests pass
4. **Follow code style**: Go formatting, TypeScript ESLint
5. **Write tests**: Coverage for new features
6. **Commit changes**: `git commit -m 'Add amazing feature'`
7. **Push to branch**: `git push origin feature/amazing-feature`
8. **Open a Pull Request**

### Development Workflow

1. Check existing issues or create a new one
2. Discuss approach in the issue
3. Implement with tests
4. Update documentation
5. Submit PR with detailed description

---

## 📚 Documentation

| Document | Description |
|----------|-------------|
| **[desktop-app/README.md](./desktop-app/README.md)** | ⭐ Desktop app guide (Tauri native app) |
| **[README-NATIVE.md](./README-NATIVE.md)** | ⭐ Native binary guide (single file deployment) |
| **[DOCKER.md](./DOCKER.md)** | Complete Docker installation & usage guide |
| **[HYPERION_COORDINATOR_MCP_REFERENCE.md](./HYPERION_COORDINATOR_MCP_REFERENCE.md)** | ⭐ Complete MCP tool reference (33 tools) |
| **[CLAUDE.md](./CLAUDE.md)** | Multi-agent coordination patterns |
| **[GO_CODE_CONSOLIDATION_STATUS.md](./GO_CODE_CONSOLIDATION_STATUS.md)** | Go code consolidation details |
| **[ARCHIVE_SUMMARY.md](./ARCHIVE_SUMMARY.md)** | Archive information |
| **[SPECIFICATION.md](./SPECIFICATION.md)** | Full technical specification |

## 🧪 Testing

### Backend Tests
```bash
cd mcp-http-bridge
go test -v              # All tests
go test -cover          # With coverage (60.3%)
go test -bench=.        # Benchmarks
```

### Frontend Tests
```bash
cd ui
npm run test            # Headless tests (109 tests)
npm run test:headed     # Headed mode
npm run test:ui         # Interactive UI
```

**Coverage:**
- HTTP Bridge: 60.3% (9 unit tests + 6 benchmarks)
- React UI: ~85% (109 Playwright tests)
- Concurrent requests: Tested up to 20 simultaneous

## 🚢 Deployment

### Docker Production

```bash
# On production server
git clone <repository-url>
cd hyper-mcp
cp .env.example .env
# Edit .env with production MongoDB URI
docker-compose up -d
```

**Production checklist:**
- [ ] Set production `MONGODB_URI` in `.env`
- [ ] Configure reverse proxy (nginx/Caddy) for HTTPS
- [ ] Set up monitoring and log rotation
- [ ] Configure MongoDB backups
- [ ] Verify auto-restart is enabled

### Native Deployment

Build binaries and deploy as systemd services. See [README sections](#installation) for build instructions.

## 🤝 Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create feature branch: `git checkout -b feature/amazing-feature`
3. Run tests: Ensure all tests pass
4. Follow code style: Go formatting, TypeScript ESLint
5. Write tests: Coverage for new features
6. Submit PR with detailed description

## 🙏 Acknowledgments

Built with:
- [Model Context Protocol](https://modelcontextprotocol.io) - MCP specification
- [Go MCP SDK](https://github.com/modelcontextprotocol/go-sdk) - Official Go SDK
- [MongoDB Atlas](https://www.mongodb.com/cloud/atlas) - Cloud database
- [Material-UI](https://mui.com) - React UI components
- [Qdrant](https://qdrant.tech) - Vector database

## 📄 License

Part of the Hyperion AI Platform. See LICENSE file for details.

---

**Built with ❤️ for AI agent coordination**

*Need help? Check [DOCKER.md](./DOCKER.md) for troubleshooting or open an [issue](https://github.com/your-org/hyper/issues)*
