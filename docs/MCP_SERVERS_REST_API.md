# MCP Servers REST API Documentation

## Overview

The MCP Servers REST API provides HTTP endpoints for managing external MCP (Model Context Protocol) servers in the Hyperion platform. These endpoints complement the existing MCP tools (`mcp_add_server`, `mcp_rediscover_server`, `mcp_remove_server`) by providing a REST interface for UI and other HTTP clients.

## Base URL

```
http://localhost:4097/api/v1/mcp/servers
```

## Endpoints

### 1. List All MCP Servers

**Endpoint:** `GET /api/v1/mcp/servers`

**Description:** Retrieves a list of all registered MCP servers with their metadata.

**Request:**
```bash
curl http://localhost:4097/api/v1/mcp/servers
```

**Response:**
```json
{
  "servers": [
    {
      "serverName": "example-mcp",
      "serverUrl": "http://localhost:9999/mcp",
      "description": "Example MCP server",
      "toolCount": 5,
      "createdAt": "2025-10-30T06:00:00Z",
      "updatedAt": "2025-10-30T06:00:00Z"
    }
  ],
  "total": 1
}
```

**Response Fields:**
- `servers` (array): List of server objects
  - `serverName` (string): Unique identifier for the server
  - `serverUrl` (string): HTTP/HTTPS URL of the MCP server
  - `description` (string): Human-readable description
  - `toolCount` (integer): Number of tools discovered from this server
  - `createdAt` (string): ISO 8601 timestamp when server was added
  - `updatedAt` (string): ISO 8601 timestamp of last modification
- `total` (integer): Total number of servers

**Status Codes:**
- `200 OK`: Success
- `500 Internal Server Error`: Database or system error

---

### 2. Add New MCP Server

**Endpoint:** `POST /api/v1/mcp/servers`

**Description:** Registers a new MCP server and discovers its tools.

**Request Body:**
```json
{
  "serverName": "my-mcp-server",
  "serverUrl": "http://localhost:3000/mcp",
  "description": "My custom MCP server"
}
```

**Request Fields:**
- `serverName` (string, required): Unique name for the server
  - Must contain only alphanumeric characters, dashes, and underscores
  - Maximum 100 characters
- `serverUrl` (string, required): Full HTTP/HTTPS URL to the MCP endpoint
- `description` (string, optional): Human-readable description

**Example:**
```bash
curl -X POST http://localhost:4097/api/v1/mcp/servers \
  -H "Content-Type: application/json" \
  -d '{
    "serverName": "github-mcp",
    "serverUrl": "https://mcp.github.com/v1",
    "description": "GitHub MCP integration"
  }'
```

**Response:**
```json
{
  "success": true,
  "message": "MCP server 'github-mcp' added successfully"
}
```

**Status Codes:**
- `200 OK`: Server added successfully
- `400 Bad Request`: Invalid request body or server name validation failed
- `500 Internal Server Error`: Failed to add server or discover tools

**Validation Rules:**
- Server name must be unique
- Server name cannot contain spaces or special characters (except `-` and `_`)
- Server URL must be a valid HTTP/HTTPS URL

---

### 3. Remove MCP Server

**Endpoint:** `DELETE /api/v1/mcp/servers/:serverName`

**Description:** Removes an MCP server and all its associated tools from the registry.

**Path Parameters:**
- `serverName` (string): Name of the server to remove

**Example:**
```bash
curl -X DELETE http://localhost:4097/api/v1/mcp/servers/github-mcp
```

**Response:**
```json
{
  "success": true,
  "message": "MCP server 'github-mcp' removed successfully"
}
```

**Status Codes:**
- `200 OK`: Server removed successfully
- `400 Bad Request`: Server name not provided
- `500 Internal Server Error`: Failed to remove server

**Notes:**
- This operation removes all tools associated with the server from both MongoDB and Qdrant
- The operation is permanent and cannot be undone

---

### 4. Rediscover Server Tools

**Endpoint:** `POST /api/v1/mcp/servers/:serverName/rediscover`

**Description:** Rediscovers and refreshes the tool list from an existing MCP server. This is useful when the server has been updated with new tools.

**Path Parameters:**
- `serverName` (string): Name of the server to rediscover

**Example:**
```bash
curl -X POST http://localhost:4097/api/v1/mcp/servers/github-mcp/rediscover
```

**Response:**
```json
{
  "success": true,
  "message": "Tools from MCP server 'github-mcp' rediscovered successfully (12 tools)"
}
```

**Status Codes:**
- `200 OK`: Tools rediscovered successfully
- `400 Bad Request`: Server name not provided
- `500 Internal Server Error`: Failed to rediscover tools or server not found

**Notes:**
- Old tools are removed before discovering new ones
- The server URL must be accessible at the time of rediscovery
- Tool count in the response indicates how many tools were successfully discovered and stored

---

## Implementation Details

### File Structure

```
hyper/
├── internal/
│   ├── handlers/
│   │   └── mcp_servers.go          # REST API handlers
│   ├── mcp/
│   │   ├── handlers/
│   │   │   └── tools_discovery.go  # MCP tool handlers
│   │   └── storage/
│   │       └── tools_storage.go    # Storage interface and implementation
│   └── server/
│       └── http_server.go          # Route registration
```

### Key Components

1. **MCPServersHandler** (`internal/handlers/mcp_servers.go`)
   - Implements REST endpoints
   - Validates input and handles HTTP responses
   - Delegates business logic to ToolsDiscoveryHandler

2. **ToolsDiscoveryHandler** (`internal/mcp/handlers/tools_discovery.go`)
   - Implements core MCP server management logic
   - Handles tool discovery via HTTP calls to MCP servers
   - Manages tool metadata storage

3. **ToolsStorage** (`internal/mcp/storage/tools_storage.go`)
   - Provides interface for MongoDB and Qdrant operations
   - Stores server metadata and tool schemas
   - Enables semantic search over tool metadata

### Data Flow

```
REST Client → MCPServersHandler → ToolsDiscoveryHandler → ToolsStorage → MongoDB/Qdrant
                                           ↓
                                    External MCP Server (for discovery)
```

### Error Handling

All endpoints return consistent error responses:

```json
{
  "error": "Error message",
  "details": "Detailed error information"
}
```

Common error scenarios:
- Invalid JSON in request body → `400 Bad Request`
- Server name validation failure → `400 Bad Request`
- Server not found → `500 Internal Server Error` (with descriptive message)
- Database connection issues → `500 Internal Server Error`
- External MCP server unreachable → `500 Internal Server Error`

---

## Testing

### Automated Test Suite

A comprehensive test script is provided at `/Users/maxmednikov/MaxSpace/hyper/test_mcp_servers_api.sh`:

```bash
# Run all tests
./test_mcp_servers_api.sh

# Run with custom base URL
BASE_URL=http://localhost:4097 ./test_mcp_servers_api.sh
```

The test suite covers:
1. Listing servers (empty state)
2. Adding a new server
3. Verifying server was added
4. Rediscovering tools
5. Removing server
6. Verifying server was removed
7. Input validation (invalid server name)

### Manual Testing Examples

**List existing servers:**
```bash
curl http://localhost:4097/api/v1/mcp/servers | jq
```

**Add a test server:**
```bash
curl -X POST http://localhost:4097/api/v1/mcp/servers \
  -H "Content-Type: application/json" \
  -d '{
    "serverName": "test-server",
    "serverUrl": "http://localhost:9999/mcp",
    "description": "Test MCP server"
  }' | jq
```

**Rediscover tools:**
```bash
curl -X POST http://localhost:4097/api/v1/mcp/servers/test-server/rediscover | jq
```

**Remove server:**
```bash
curl -X DELETE http://localhost:4097/api/v1/mcp/servers/test-server | jq
```

---

## Integration with MCP Tools

The REST API complements the existing MCP tools. Both interfaces share the same backend:

| REST Endpoint | MCP Tool | Function |
|--------------|----------|----------|
| `POST /api/v1/mcp/servers` | `mcp_add_server` | Add server |
| `GET /api/v1/mcp/servers` | *(none)* | List servers |
| `DELETE /api/v1/mcp/servers/:name` | `mcp_remove_server` | Remove server |
| `POST /api/v1/mcp/servers/:name/rediscover` | `mcp_rediscover_server` | Rediscover tools |

**Use REST API when:**
- Building a UI for MCP server management
- Integrating with web applications
- Automating server management via scripts

**Use MCP tools when:**
- Working within an AI agent context
- Need semantic search over tools
- Executing tools via the MCP protocol

---

## Security Considerations

### Current Implementation

- No authentication required (runs on localhost)
- Server name validation prevents injection attacks
- HTTP client has 30-second timeout to prevent hanging
- Input validation on all fields

### Production Recommendations

For production deployments, consider:

1. **Authentication:** Add JWT authentication middleware
2. **Authorization:** Implement role-based access control
3. **Rate Limiting:** Prevent abuse of discovery endpoints
4. **HTTPS:** Use TLS for all communications
5. **Input Validation:** Additional URL validation for serverUrl
6. **Audit Logging:** Log all server additions/removals

---

## Future Enhancements

Potential improvements for future versions:

1. **Pagination:** Add pagination to GET /api/v1/mcp/servers for large server lists
2. **Filtering:** Add query parameters to filter servers by description or tool count
3. **Batch Operations:** Support adding/removing multiple servers in one request
4. **Server Health Checks:** Periodic health checks with status reporting
5. **Tool Search:** REST endpoint for semantic tool search
6. **Webhooks:** Notify external systems when servers are added/removed
7. **Update Endpoint:** PATCH endpoint to update server description/URL
8. **Export/Import:** Bulk export/import of server configurations

---

## Troubleshooting

### Common Issues

**Server returns 500 when adding server:**
- Check if external MCP server URL is accessible
- Verify MCP server is running and responding
- Check MongoDB connection

**Server added but toolCount is 0:**
- External MCP server may not be implementing tools/list correctly
- Check server logs for discovery errors
- Verify MCP server is using correct protocol version

**Rediscover fails with 500:**
- Server may have been removed
- External MCP server may be down
- Check coordinator logs for detailed error

### Debug Mode

Enable debug logging in coordinator:
```bash
LOG_LEVEL=debug make dev-hot
```

Check logs for:
- HTTP request/response details
- MongoDB query results
- MCP protocol communication
- Tool discovery progress

---

## Version History

- **v1.0** (2025-10-30): Initial implementation
  - All four CRUD endpoints
  - Complete test suite
  - Documentation

---

## Support

For issues or questions:
- Check coordinator logs: `docker logs hyperion-coordinator`
- Review test script for usage examples
- Consult MCP protocol documentation at https://modelcontextprotocol.io
