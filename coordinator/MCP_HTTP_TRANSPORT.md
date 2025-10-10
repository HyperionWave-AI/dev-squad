# MCP HTTP Transport Implementation

## Overview

The Hyperion Coordinator now implements the official Model Context Protocol (MCP) HTTP transport using the go-sdk v1.0.0. This enables Claude Code and other MCP clients to communicate with the coordinator over HTTP using the Streamable HTTP transport.

## Implementation Details

### Transport Type: StreamableHTTP

The implementation uses `mcp.NewStreamableHTTPHandler` which implements the MCP Streamable HTTP transport specification (2024-11-05). This is the modern, recommended approach that replaces the older SSE-only transport.

**Key Features:**
- **POST-based communication**: All MCP requests use HTTP POST with JSON-RPC 2.0 payloads
- **SSE streaming**: Responses use Server-Sent Events (text/event-stream) for real-time updates
- **Session management**: Maintains stateful sessions via `Mcp-Session-Id` headers
- **Full MCP compliance**: Implements the complete MCP specification including:
  - Tools (tools/list, tools/call)
  - Resources (resources/list, resources/read)
  - Prompts (prompts/list, prompts/get)
  - Logging and notifications

### Code Location

**Primary Implementation:**
- `coordinator/internal/server/http_server.go` - MCP HTTP endpoint setup
- `coordinator/cmd/coordinator/main.go:191` - MCP server instance creation

**Test Coverage:**
- `coordinator/internal/server/http_server_test.go` - Comprehensive unit and integration tests

## HTTP Endpoint

**Endpoint:** `POST /mcp`

**Required Headers:**
```http
Content-Type: application/json
Accept: application/json, text/event-stream
```

**Optional Headers:**
```http
Mcp-Session-Id: <session-id>  # For maintaining session state
```

### Example: Initialize Request

```bash
curl -X POST http://localhost:7095/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
      "protocolVersion": "2024-11-05",
      "capabilities": {},
      "clientInfo": {
        "name": "claude-code",
        "version": "1.0.0"
      }
    }
  }'
```

**Response (SSE format):**
```
event: message
id: <message-id>
data: {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"logging":{},"prompts":{"listChanged":true},"resources":{"listChanged":true},"tools":{"listChanged":true}},"protocolVersion":"2024-11-05","serverInfo":{"name":"hyperion-coordinator-unified","version":"2.0.0"}}}
```

### Example: List Tools

```bash
curl -X POST http://localhost:7095/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Mcp-Session-Id: <session-id-from-initialize>" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/list",
    "params": {}
  }'
```

## Configuration

The MCP HTTP transport is configured with the following options:

```go
mcpHandler := mcp.NewStreamableHTTPHandler(
    func(req *http.Request) *mcp.Server {
        return mcpServer  // Same server instance for all requests
    },
    &mcp.StreamableHTTPOptions{
        Stateless:    false,  // Validates Mcp-Session-Id headers
        JSONResponse: false,  // Uses text/event-stream for responses
    },
)
```

**Configuration Explanation:**

- **`Stateless: false`**:
  - Validates the `Mcp-Session-Id` header on each request
  - Maintains proper session state across requests
  - Required for server-to-client notifications and requests

- **`JSONResponse: false`**:
  - Responses use `text/event-stream` (SSE format)
  - Enables real-time streaming of responses
  - Follows MCP specification for Streamable HTTP

## Server Modes

The coordinator supports multiple server modes via the `--mode` flag:

```bash
# HTTP only (REST API + MCP HTTP + UI)
./coordinator --mode=http

# MCP stdio only (for command-line MCP clients)
./coordinator --mode=mcp

# Both HTTP and stdio (default)
./coordinator --mode=both
```

## MCP Server Capabilities

The coordinator exposes the following MCP capabilities:

- **Tools** (`HasTools: true`)
  - Create and manage tasks (human and agent tasks)
  - Update task status and TODOs
  - Query and store knowledge
  - Code indexing and search
  - Qdrant vector operations

- **Resources** (`HasResources: true`)
  - Documentation resources (standards, architecture, guides)
  - Workflow resources (active agents, task queue, dependencies)
  - Knowledge resources (collections, recent learnings)
  - Metrics resources (squad velocity, context efficiency)
  - Dynamic task resources (human and agent tasks)

- **Prompts** (`HasPrompts: true`)
  - Task planning prompts
  - Knowledge query optimization
  - Cross-squad coordination
  - Documentation generation

## Testing

Run the MCP HTTP transport tests:

```bash
cd coordinator/internal/server
go test -v -run TestMCPHTTPTransport
go test -v -run TestMCPServerIntegration
```

**Test Coverage:**
- ✅ POST /mcp with initialize request
- ✅ POST /mcp with tools/list request
- ✅ Session ID management
- ✅ SSE response format validation
- ✅ End-to-end integration testing

## Connecting from Claude Code

To connect Claude Code to the coordinator via MCP HTTP:

1. **Start the coordinator:**
   ```bash
   cd coordinator
   ./coordinator --mode=http
   ```

2. **Configure Claude Code MCP settings:**
   ```json
   {
     "mcpServers": {
       "hyperion-coordinator": {
         "transport": "http",
         "url": "http://localhost:7095/mcp"
       }
     }
   }
   ```

3. **Verify connection:**
   - Claude Code will send an `initialize` request
   - Check coordinator logs for "MCP HTTP transport initialized"
   - Tools should appear in Claude Code's MCP tool list

## Architecture Benefits

**Using Official go-sdk v1.0.0:**
- ✅ Full MCP 2024-11-05 specification compliance
- ✅ Automatic protocol handling (initialization, session management)
- ✅ Built-in SSE streaming support
- ✅ Standard error handling and validation
- ✅ Future-proof against spec updates

**StreamableHTTPHandler Advantages:**
- Implements http.Handler interface (works with any HTTP framework)
- Handles all MCP protocol details automatically
- Supports both stateless and stateful modes
- Efficient session management
- Proper content negotiation

## Troubleshooting

### Issue: "Accept must contain both 'application/json' and 'text/event-stream'"

**Solution:** Ensure your HTTP client sends the correct Accept header:
```
Accept: application/json, text/event-stream
```

### Issue: "405 Method Not Allowed" on GET requests

**Explanation:** StreamableHTTP transport only supports POST method. This is correct behavior per MCP spec.

### Issue: Session not found errors

**Solution:** Include the `Mcp-Session-Id` header from the initialize response in subsequent requests.

## References

- [MCP Specification - Transports](https://modelcontextprotocol.io/docs/concepts/transports)
- [go-sdk v1.0.0 Documentation](https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk@v1.0.0/mcp)
- [Streamable HTTP Transport Spec](https://modelcontextprotocol.io/2025/03/26/streamable-http-transport.html)

## Migration Notes

**Previous Implementation:**
- Placeholder `/mcp` endpoints with friendly error messages
- No actual MCP protocol support
- Required manual JSON-RPC handling

**Current Implementation:**
- Full MCP protocol support via official go-sdk
- Automatic session management
- Standard SSE streaming
- Complete tools/resources/prompts capabilities

**Breaking Changes:**
- None - this is a new feature addition
- REST API endpoints remain unchanged
- Existing stdio MCP mode continues to work
