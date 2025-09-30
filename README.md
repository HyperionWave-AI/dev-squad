# 🚀 Hyperion Coordinator

> **AI-Powered Task Coordination System with Model Context Protocol (MCP)**

A modern, full-stack task management system designed for AI agent coordination, featuring hierarchical task decomposition, real-time progress tracking, and intelligent knowledge management.

![Hyperion Coordinator Dashboard](readme-resources/screenshot.png)

---

## 📋 Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Technology Stack](#technology-stack)
- [Architecture](#architecture)
- [Getting Started](#getting-started)
- [Project Structure](#project-structure)
- [API Documentation](#api-documentation)
- [Development](#development)
- [Testing](#testing)
- [Deployment](#deployment)
- [Contributing](#contributing)

---

## 🎯 Overview

The Hyperion Coordinator is a production-ready MCP (Model Context Protocol) server that orchestrates complex AI agent workflows through hierarchical task management. It bridges human-level planning with agent-level execution, providing a unified coordination layer for multi-agent systems.

### Key Capabilities

- **Hierarchical Task Management**: Human tasks decompose into agent tasks with fine-grained TODO tracking
- **Real-Time Progress Tracking**: Live status updates across distributed agent workflows
- **Knowledge Management**: Vector-based knowledge storage for contextual task execution
- **Modern UI**: Kanban-style dashboard built with Material-UI and React 19
- **MCP-Native**: First-class support for Model Context Protocol with 9 tools and dynamic resources
- **Production-Grade**: Comprehensive test coverage, concurrent request handling, and resilient architecture

---

## ✨ Features

### 🎨 Kanban Dashboard

- **4-Column Board**: PENDING → IN PROGRESS → BLOCKED → COMPLETED
- **Priority Color-Coding**: Visual priority indicators (High/Medium/Low)
- **Drag & Drop**: Intuitive task state management
- **Real-Time Updates**: Automatic polling for live coordination
- **Responsive Design**: Desktop, tablet, and mobile optimized

### 🤖 AI Agent Coordination

- **Task Decomposition**: Human tasks break down into agent-specific subtasks
- **Role Assignment**: Agents receive tasks with clear roles and responsibilities
- **TODO Management**: Granular progress tracking at the sub-task level
- **Status Tracking**: Multi-state workflow (pending/in_progress/completed/blocked)
- **Knowledge Integration**: Task-specific knowledge stored and queried via Qdrant

### 🔧 MCP Integration

- **9 Coordination Tools**: Complete task lifecycle management via MCP
- **Dynamic Resources**: URI-based access to task data (`hyperion://task/*`)
- **HTTP Bridge**: RESTful API gateway for web clients
- **Concurrent Request Handling**: Production-tested for high-concurrency scenarios
- **Protocol Compliance**: Official MCP Go SDK v0.3.0 implementation

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

- **HTTP Bridge**: Go-based MCP-to-HTTP adapter (Port 8095)
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
│              MCP HTTP Bridge (Port 8095)                │
│  • CORS handling for web clients                        │
│  • HTTP → stdio request translation                     │
│  • Concurrent request routing                           │
│  • Health monitoring                                    │
└────────────────────┬────────────────────────────────────┘
                     │ stdio (JSON-RPC)
                     ▼
┌─────────────────────────────────────────────────────────┐
│           MCP Server (hyperion-coordinator)             │
│  ┌───────────────────────────────────────────────────┐  │
│  │ Tools (9)                                         │  │
│  │ • coordinator_create_human_task                   │  │
│  │ • coordinator_create_agent_task                   │  │
│  │ • coordinator_list_human_tasks                    │  │
│  │ • coordinator_list_agent_tasks                    │  │
│  │ • coordinator_update_task_status                  │  │
│  │ • coordinator_update_todo_status                  │  │
│  │ • coordinator_clear_task_board                    │  │
│  │ • coordinator_upsert_knowledge                    │  │
│  │ • coordinator_query_knowledge                     │  │
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

## 🚀 Getting Started

### Prerequisites

- **Go 1.25+**: [Download](https://go.dev/dl/)
- **Node.js 18+**: [Download](https://nodejs.org/)
- **MongoDB Atlas Account**: [Sign Up](https://www.mongodb.com/cloud/atlas)
- **Qdrant Cloud Account**: [Sign Up](https://cloud.qdrant.io/) (optional for knowledge features)

### Environment Setup

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd coordinator
   ```

2. **Configure MongoDB**

   Set your MongoDB Atlas connection string:
   ```bash
   export MONGODB_URI="mongodb+srv://username:password@cluster.mongodb.net/coordinator_db?retryWrites=true&w=majority"
   ```

3. **Configure Qdrant (Optional)**

   For knowledge management features:
   ```bash
   export QDRANT_URL="https://your-cluster.cloud.qdrant.io"
   export QDRANT_API_KEY="your-api-key"
   ```

### Quick Start

The entire stack (MCP server + HTTP bridge + React UI) starts with a single command:

```bash
./start-coordinator.sh
```

This will:
- ✅ Build the Go MCP server
- ✅ Build the Go HTTP bridge
- ✅ Install UI dependencies
- ✅ Start all services in the background

**Service URLs:**
- 🌉 **HTTP Bridge**: http://localhost:8095
- 🎨 **React UI**: http://localhost:5173
- 💾 **MongoDB**: MongoDB Atlas (cloud)

### Manual Start (Alternative)

If you prefer to run components individually:

```bash
# Terminal 1: Start MCP Server + HTTP Bridge
cd mcp-http-bridge
go build -o mcp-http-bridge
./mcp-http-bridge

# Terminal 2: Start React UI
cd ui
npm install
npm run dev
```

### Verify Installation

1. **Check HTTP Bridge Health**
   ```bash
   curl http://localhost:8095/health
   # Expected: {"status":"healthy","service":"hyperion-coordinator-http-bridge","version":"1.0.0"}
   ```

2. **Create a Test Task**
   ```bash
   curl -X POST http://localhost:8095/api/mcp/tools/call \
     -H "Content-Type: application/json" \
     -H "X-Request-ID: test-1" \
     -d '{
       "name": "coordinator_create_human_task",
       "arguments": {
         "prompt": "Test task from README"
       }
     }'
   ```

3. **Open the UI**

   Visit http://localhost:5173 and verify the Kanban board displays your test task.

---

## 📁 Project Structure

```
coordinator/
├── mcp-server/                  # MCP protocol server (Go)
│   ├── main.go                  # Server entry point
│   ├── handlers/                # MCP tool handlers
│   │   ├── tools.go             # Tool implementations
│   │   └── resources.go         # Resource implementations
│   ├── storage/                 # Database layer
│   │   ├── tasks.go             # Task storage (MongoDB)
│   │   └── knowledge.go         # Knowledge storage (Qdrant)
│   └── go.mod                   # Go dependencies
│
├── mcp-http-bridge/             # HTTP ↔ MCP adapter (Go)
│   ├── main.go                  # Bridge server + routing
│   ├── main_test.go             # Unit tests (60.3% coverage)
│   ├── benchmark_test.go        # Performance benchmarks
│   ├── CLAUDE.md                # Architecture documentation
│   └── TEST_README.md           # Testing guide
│
├── ui/                          # React frontend
│   ├── src/
│   │   ├── App.tsx              # Main application
│   │   ├── theme.ts             # MUI theme configuration
│   │   ├── components/          # React components
│   │   │   ├── KanbanBoard.tsx  # Kanban board container
│   │   │   ├── KanbanTaskCard.tsx  # Task card component
│   │   │   └── KnowledgeBrowser.tsx  # Knowledge UI (future)
│   │   ├── services/            # API clients
│   │   │   └── mcpClient.ts     # MCP HTTP client
│   │   └── types/               # TypeScript types
│   │       └── coordinator.ts   # Task/Agent types
│   ├── tests/                   # Playwright E2E tests
│   │   ├── kanban-rendering.spec.ts
│   │   ├── drag-drop.spec.ts
│   │   ├── accessibility.spec.ts
│   │   └── ... (8 test suites, 109 tests)
│   ├── package.json             # Node dependencies
│   └── vite.config.ts           # Build configuration
│
├── start-coordinator.sh         # One-command startup script
├── SPECIFICATION.md             # Technical specification
├── FULL_STACK_SETUP.md         # Integration guide
└── README.md                    # This file
```

---

## 📚 API Documentation

### HTTP Bridge Endpoints

#### Health Check
```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "service": "hyperion-coordinator-http-bridge",
  "version": "1.0.0"
}
```

#### List MCP Tools
```http
GET /api/mcp/tools
X-Request-ID: <unique-id>
```

**Response:**
```json
{
  "tools": [
    {
      "name": "coordinator_create_human_task",
      "description": "Create a new human task",
      "inputSchema": { ... }
    },
    ...
  ]
}
```

#### Call MCP Tool
```http
POST /api/mcp/tools/call
Content-Type: application/json
X-Request-ID: <unique-id>

{
  "name": "coordinator_list_human_tasks",
  "arguments": {}
}
```

**Response:**
```json
{
  "content": [
    {
      "type": "text",
      "text": "✓ Retrieved 2 human tasks\n\nTasks:\n[...]"
    }
  ]
}
```

#### Read Resource
```http
GET /api/mcp/resources/read?uri=hyperion://task/human/{taskId}
X-Request-ID: <unique-id>
```

### MCP Tools

| Tool | Purpose | Parameters |
|------|---------|------------|
| `coordinator_create_human_task` | Create user task | `prompt: string` |
| `coordinator_create_agent_task` | Create agent task | `humanTaskId, agentName, role, todos` |
| `coordinator_list_human_tasks` | List all human tasks | None |
| `coordinator_list_agent_tasks` | List agent tasks | `agentName?`, `humanTaskId?` |
| `coordinator_update_task_status` | Update task status | `taskId, status, notes?` |
| `coordinator_update_todo_status` | Update TODO status | `agentTaskId, todoId, status, notes?` |
| `coordinator_clear_task_board` | Clear all tasks | `confirm: true` |
| `coordinator_upsert_knowledge` | Store knowledge | `collection, text, metadata?` |
| `coordinator_query_knowledge` | Query knowledge | `collection, query, limit?` |

---

## 🔧 Development

### Running Tests

**Backend Tests:**
```bash
cd mcp-http-bridge
go test -v -timeout=120s                    # All tests
go test -cover                              # With coverage
go test -bench=. -benchtime=10s             # Benchmarks
go test -run TestConcurrentRequests         # Specific test
```

**Frontend Tests:**
```bash
cd ui
npm run test                                # Headless tests
npm run test:headed                         # Headed mode
npm run test:ui                             # Interactive UI
npm run test:accessibility                  # A11y tests only
```

### Building

**Backend:**
```bash
cd mcp-server && go build -o hyperion-coordinator-mcp
cd mcp-http-bridge && go build -o mcp-http-bridge
```

**Frontend:**
```bash
cd ui && npm run build
```

### Code Quality

**Go:**
- File size limits: Handlers ≤300 lines, Services ≤400 lines
- Test coverage target: >80%
- Concurrent request handling validated

**TypeScript:**
- ESLint configured
- React Hooks rules enforced
- TypeScript strict mode enabled

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

### Production Checklist

- [ ] Set `MONGODB_URI` environment variable
- [ ] Set `QDRANT_URL` and `QDRANT_API_KEY` (if using knowledge features)
- [ ] Configure CORS origins in `mcp-http-bridge/main.go`
- [ ] Build binaries: `go build` in both Go directories
- [ ] Build frontend: `npm run build` in ui/
- [ ] Set up reverse proxy (nginx/Caddy) for HTTPS
- [ ] Configure firewall rules (port 8095 internal only)
- [ ] Set up monitoring and logging
- [ ] Configure backup strategy for MongoDB

### Docker (Future)

Docker support is planned for containerized deployments.

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

## 📄 License

This project is part of the Hyperion AI Platform. See LICENSE file for details.

---

## 🙏 Acknowledgments

- **Model Context Protocol**: [MCP Specification](https://modelcontextprotocol.io)
- **Material-UI**: [MUI Docs](https://mui.com)
- **Go MCP SDK**: [GitHub](https://github.com/modelcontextprotocol/go-sdk)
- **MongoDB Atlas**: Cloud database hosting
- **Qdrant**: Vector database for knowledge management

---

## 📞 Support

- **Documentation**: See `SPECIFICATION.md` for technical details
- **Testing Guide**: See `mcp-http-bridge/TEST_README.md`
- **Architecture**: See `mcp-http-bridge/CLAUDE.md`
- **Issues**: [GitHub Issues](https://github.com/your-org/hyperion-coordinator/issues)

---

**Built with ❤️ for AI Agent Coordination**

*Hyperion Coordinator - Orchestrating Intelligence, One Task at a Time*
