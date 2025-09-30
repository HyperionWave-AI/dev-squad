# Hyperion Coordinator MCP Server

An MCP (Model Context Protocol) server for coordinating tasks and knowledge across AI agents in the Hyperion system.

## Overview

This MCP server provides:
- **Task Management**: Create and track human and agent tasks with hierarchical relationships
- **Knowledge Base**: Store and query coordination knowledge using semantic similarity
- **Resource Access**: URI-based access to task details via MCP resources
- **Tool Operations**: Five coordination tools for task and knowledge management

## Architecture

### Storage
- **MongoDB Atlas**: Persistent cloud storage using MongoDB Atlas (dev cluster)
- **Task Storage**: Separate collections for human tasks and agent tasks with parent-child relationships
- **Knowledge Storage**: Collection-based knowledge entries with MongoDB text search and similarity matching
- **Indexes**: Optimized indexes on taskId, agentName, collection, and text fields for fast queries

### Resources

The server exposes tasks as MCP resources with these URI patterns:

1. **Human Tasks**: `hyperion://task/human/{taskId}`
   - Original user prompt
   - Creation timestamp
   - Current status
   - Optional notes

2. **Agent Tasks**: `hyperion://task/agent/{agentName}/{taskId}`
   - Parent human task reference
   - Agent name and role
   - TODO list
   - Status and notes

### Tools

The server provides five coordination tools:

#### 1. `coordinator_upsert_knowledge`
Store knowledge in the coordinator knowledge base.

**Parameters:**
- `collection` (string, required): Collection name (e.g., "task:taskURI", "adr", "data-contracts")
- `text` (string, required): Content to store
- `metadata` (object, optional): Additional metadata (taskId, agentName, timestamp, etc.)

**Returns:** Success confirmation with entry ID

**Example:**
```json
{
  "collection": "task:hyperion://task/human/abc-123",
  "text": "Implemented authentication middleware using JWT tokens",
  "metadata": {
    "taskId": "abc-123",
    "agentName": "security-specialist",
    "timestamp": "2025-09-30T10:00:00Z"
  }
}
```

#### 2. `coordinator_query_knowledge`
Query the knowledge base with semantic similarity scoring.

**Parameters:**
- `collection` (string, required): Collection name to search
- `query` (string, required): Text to search for
- `limit` (number, optional): Maximum results (default: 5)

**Returns:** Array of knowledge entries with similarity scores

**Example:**
```json
{
  "collection": "adr",
  "query": "authentication middleware patterns",
  "limit": 3
}
```

#### 3. `coordinator_create_human_task`
Create a new human task with the original user request.

**Parameters:**
- `prompt` (string, required): Original human request/prompt

**Returns:** Task ID (UUID format)

**Example:**
```json
{
  "prompt": "Build a custom MCP server for task coordination"
}
```

#### 4. `coordinator_create_agent_task`
Create a new agent task linked to a parent human task.

**Parameters:**
- `humanTaskId` (string, required): Parent human task ID
- `agentName` (string, required): Name of assigned agent
- `role` (string, required): Agent's role/responsibility
- `todos` (array, required): List of TODO items

**Returns:** Task ID (UUID format)

**Example:**
```json
{
  "humanTaskId": "abc-123",
  "agentName": "go-mcp-dev",
  "role": "MCP server implementation",
  "todos": [
    "Design storage layer",
    "Implement resource handlers",
    "Create tool handlers",
    "Write tests"
  ]
}
```

#### 5. `coordinator_update_task_status`
Update the status of any task (human or agent).

**Parameters:**
- `taskId` (string, required): Task ID to update
- `status` (string, required): New status (pending, in_progress, completed, blocked)
- `notes` (string, optional): Progress notes

**Returns:** Success confirmation

**Example:**
```json
{
  "taskId": "def-456",
  "status": "completed",
  "notes": "Successfully implemented all MCP handlers"
}
```

## Setup

### Prerequisites
- Go 1.25+
- Official MCP Go SDK
- MongoDB Atlas access (dev cluster)

### Installation

1. Clone the repository and navigate to the server directory:
```bash
cd ./development/coordinator/mcp-server/
```

2. Install dependencies:
```bash
go mod download
```

3. Build the server:
```bash
go build -o hyperion-coordinator-mcp
```

### Configuration

The server connects to MongoDB Atlas for persistent storage. Configuration via environment variables:

```bash
# Optional: Override default MongoDB connection
export MONGODB_URI="mongodb+srv://dev:fvOKzv9enD8CSVwD@devdb.yqf8f8r.mongodb.net/?retryWrites=true&w=majority&appName=devDB"

# Optional: Override default database name (default: coordinator_db)
export MONGODB_DATABASE="coordinator_db"
```

**Default Behavior**: If not set, the server uses the dev MongoDB Atlas cluster automatically.

### Running the Server

The server uses stdio transport for MCP communication:

```bash
./hyperion-coordinator-mcp
```

On startup, you'll see:
```
Starting Hyperion Coordinator MCP Server
Connecting to MongoDB Atlas database=coordinator_db
Successfully connected to MongoDB Atlas
Task storage initialized with MongoDB
Knowledge storage initialized with MongoDB
All handlers registered successfully tools=5 resources=2
Starting MCP server with stdio transport
```

### MCP Client Configuration

#### Quick Setup (Claude Code)

Run the provided setup script:

```bash
./add-to-claude-code.sh
```

This automatically adds the server to Claude Code's configuration.

#### Manual Setup (Other MCP Clients)

Add to your MCP client configuration (e.g., Claude Desktop):

```json
{
  "mcpServers": {
    "hyperion-coordinator": {
      "command": "/Users/maxmednikov/MaxSpace/Hyperion/development/coordinator/mcp-server/hyperion-coordinator-mcp"
    }
  }
}
```

## Usage Examples

### Creating a Task Workflow

1. **Create human task:**
```bash
coordinator_create_human_task(prompt="Implement user authentication")
# Returns: taskId="abc-123"
```

2. **Create agent tasks:**
```bash
coordinator_create_agent_task(
  humanTaskId="abc-123",
  agentName="security-specialist",
  role="JWT middleware implementation",
  todos=["Design JWT schema", "Implement middleware", "Add tests"]
)
# Returns: taskId="def-456"

coordinator_create_agent_task(
  humanTaskId="abc-123",
  agentName="frontend-specialist",
  role="Login UI components",
  todos=["Create login form", "Add auth context", "Handle token storage"]
)
# Returns: taskId="ghi-789"
```

3. **Store knowledge:**
```bash
coordinator_upsert_knowledge(
  collection="task:hyperion://task/human/abc-123",
  text="Using bcrypt for password hashing with cost factor 12",
  metadata={"agentName": "security-specialist", "taskId": "def-456"}
)
```

4. **Query knowledge:**
```bash
coordinator_query_knowledge(
  collection="task:hyperion://task/human/abc-123",
  query="password hashing approach",
  limit=5
)
```

5. **Update task status:**
```bash
coordinator_update_task_status(
  taskId="def-456",
  status="completed",
  notes="JWT middleware fully implemented and tested"
)
```

### Knowledge Collections

Recommended collection naming patterns:

- **Task-specific**: `task:hyperion://task/human/{taskId}`
- **Architecture Decisions**: `adr`
- **Data Contracts**: `data-contracts`
- **Agent Coordination**: `agent-coordination`
- **Technical Debt**: `technical-debt`
- **Code Patterns**: `code-patterns`

### Resource Access

Access task details via MCP resources:

```bash
# Read human task
resources/read URI="hyperion://task/human/abc-123"

# Read agent task
resources/read URI="hyperion://task/agent/security-specialist/def-456"

# List all tasks
resources/list
```

## Development

### Project Structure

```
./development/coordinator/mcp-server/
├── main.go              # Server entry point and initialization
├── go.mod               # Go module definition
├── go.sum               # Dependency checksums
├── handlers/
│   ├── resources.go     # MCP resource handlers (list, read)
│   └── tools.go         # MCP tool handlers (5 coordination tools)
├── storage/
│   ├── tasks.go         # Task storage (human + agent tasks)
│   └── knowledge.go     # Knowledge storage with similarity search
└── README.md            # This file
```

### Storage Details

**Task Storage:**
- Human tasks: Map of taskId → HumanTask
- Agent tasks: Map of taskId → AgentTask
- Relationships: Map of humanTaskId → []agentTaskIds

**Knowledge Storage:**
- Entries: Map of entryId → KnowledgeEntry
- Collections: Map of collection → []entryIds
- Similarity: Simple text matching with word overlap scoring

### Similarity Scoring

The knowledge query uses a simple similarity algorithm:
- **Exact match**: Score 1.0
- **Contains substring**: Score 0.7
- **Word overlap**: Score based on proportion of matched words (max 0.5)

For production use, consider integrating with a real vector database (e.g., Qdrant, Weaviate).

## Testing

Test the server using an MCP client or by sending JSON-RPC requests via stdio:

```bash
# Example JSON-RPC request (via stdin)
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | ./hyperion-coordinator-mcp
```

## Limitations

- **In-Memory Only**: Data is lost on server restart
- **No Persistence**: No database backend
- **Simple Similarity**: Text matching only, no embeddings
- **No Authentication**: Local development use only
- **Single Process**: No distributed coordination

## Future Enhancements

- [ ] Add Qdrant integration for real vector similarity
- [ ] Implement task dependencies and blocking relationships
- [ ] Add task assignment and reassignment capabilities
- [ ] Create task history and audit trail
- [ ] Add WebSocket notifications for task updates
- [ ] Implement task search and filtering
- [ ] Add metrics and observability
- [ ] Create CLI for local task management

## License

Part of the Hyperion monorepo. See root LICENSE file.

## Contact

Part of the Hyperion Parallel Squad System. See main CLAUDE.md for agent coordination guidelines.