# MCP HTTP Bridge - Service Documentation

## Overview

The MCP HTTP Bridge provides HTTP/REST access to the Hyperion Coordinator MCP server, which communicates via stdio using the Model Context Protocol (MCP) JSON-RPC format.

**Purpose:** Enable web frontends to interact with MCP tools without requiring native MCP client support.

## Architecture

### Communication Flow

```
Frontend (HTTP)  →  HTTP Bridge (Gin)  →  MCP Server (stdio)
                ←                       ←
```

1. **Frontend** sends HTTP POST/GET requests to bridge
2. **Bridge** translates HTTP → MCP JSON-RPC via stdin
3. **MCP Server** processes and returns JSON-RPC response via stdout
4. **Bridge** translates MCP response → HTTP JSON response
5. **Frontend** receives HTTP response

### Key Components

#### HTTPBridge struct
- Manages MCP server subprocess lifecycle
- Maintains stdin/stdout pipes for communication
- Routes responses to correct pending requests
- Thread-safe request/response handling

#### Background Response Handler
- Dedicated goroutine continuously reads MCP server stdout
- Routes responses by request ID to waiting HTTP handlers
- Critical for concurrent request support

#### Pending Requests Map
- Maps request ID → response channel
- Enables concurrent request handling
- Automatic cleanup after response delivery

## Critical Bug Fix: Concurrent Requests

### The Problem

Original implementation had race conditions when handling concurrent requests:
- UI polls every 3 seconds with 2 simultaneous requests
- Responses arriving out of order caused "broken pipe" errors
- No mechanism to route responses to correct request handlers

### The Solution

1. **Background Response Handler:** Dedicated goroutine reads all responses
2. **Pending Requests Map:** Maps request IDs to response channels
3. **Mutex Protection:** Thread-safe access to pending requests
4. **ID-Based Routing:** Responses matched to requests by JSON-RPC ID

### Implementation Details

```go
// Background goroutine continuously reads responses
func (b *HTTPBridge) handleResponses() {
    for {
        var resp MCPResponse
        b.responseReader.Decode(&resp) // Read from stdout

        // Find waiting request handler
        respChan := b.pendingReqs[resp.ID]

        // Deliver response
        respChan <- resp

        // Cleanup
        delete(b.pendingReqs, resp.ID)
        close(respChan)
    }
}

// Each request registers for its response
func (b *HTTPBridge) sendRequest(req MCPRequest) {
    respChan := make(chan MCPResponse, 1)
    b.pendingReqs[req.ID] = respChan // Register

    b.stdin.Write(reqJSON) // Send request

    resp := <-respChan // Wait for response
    return resp.Result
}
```

## API Endpoints

### Health Check
```
GET /health
```
Returns bridge status and version.

### List Tools
```
GET /api/mcp/tools
Header: X-Request-ID: <unique-id>
```
Returns list of available MCP tools.

### Call Tool
```
POST /api/mcp/tools/call
Header: X-Request-ID: <unique-id>
Content-Type: application/json

{
  "name": "tool_name",
  "arguments": {
    "param1": "value1"
  }
}
```
Executes MCP tool and returns result.

### List Resources
```
GET /api/mcp/resources
Header: X-Request-ID: <unique-id>
```
Returns list of available MCP resources.

### Read Resource
```
GET /api/mcp/resources/read?uri=<resource-uri>
Header: X-Request-ID: <unique-id>
```
Reads specific MCP resource content.

## Configuration

### Environment Variables

- `MCP_SERVER_PATH`: Path to MCP server binary (default: `../mcp-server/hyper-mcp`)
- `PORT`: HTTP server port (default: `7095`)

### CORS Configuration

Allows origins:
- `http://localhost:5173` (Vite dev server)
- `http://localhost:3000` (React dev server)

## Testing

### Test Suite

Comprehensive test suite in `main_test.go` and `benchmark_test.go`:

**Unit Tests:**
- ✅ TestConcurrentRequests - 20+ simultaneous requests
- ✅ TestRequestResponseMatching - Out-of-order response handling
- ✅ TestTimeoutHandling - Request timeout behavior
- ✅ TestBrokenPipeRecovery - MCP server crash scenarios (documented)
- ✅ TestErrorPropagation - MCP errors to HTTP clients
- ✅ TestInvalidRequestHandling - Malformed request handling
- ✅ TestResourceEndpoints - Resource list/read functionality
- ✅ TestToolCallEndpoint - Tool execution
- ✅ TestStdinWriteLocking - Concurrent stdin write safety
- ✅ TestPendingRequestCleanup - Memory leak prevention

**Benchmark Tests:**
- ⚡ BenchmarkHighLoad - Sequential request throughput
- ⚡ BenchmarkConcurrentToolCalls - Concurrent tool execution
- ⚡ BenchmarkUIPollingSimulation - Real UI behavior (5 clients)
- ⚡ BenchmarkStressTest - Maximum concurrent load (50 goroutines)
- ⚡ BenchmarkResourceRead - Resource read performance
- ⚡ BenchmarkParallelToolCalls - Parallel execution benchmark

**Coverage: 60.3%** (target: >80%)

### Running Tests

```bash
# All tests
go test -v -timeout=120s

# With coverage
go test -cover -coverprofile=coverage.out

# Benchmarks
go test -bench=. -benchtime=10s

# Specific test
go test -v -run TestConcurrentRequests
```

See `TEST_README.md` for detailed testing documentation.

## Development

### Building

```bash
go build -o mcp-http-bridge
```

### Running Locally

```bash
# Ensure MCP server binary exists
ls ../mcp-server/hyper-mcp

# Start bridge
./mcp-http-bridge
```

### Debug Mode

Set Gin to debug mode:
```bash
export GIN_MODE=debug
./mcp-http-bridge
```

## Deployment

The bridge is typically deployed alongside the coordinator dashboard UI:

1. Build MCP server: `cd ../mcp-server && go build`
2. Build HTTP bridge: `go build`
3. Start bridge: `./mcp-http-bridge`
4. Start UI: `cd ../../dashboard-ui && npm run dev`

## Performance Characteristics

### Throughput
- **Sequential:** >1000 req/sec (real MCP server)
- **Concurrent:** Handles 50+ simultaneous requests
- **UI Polling:** Sustains 5+ clients polling every 3 seconds

### Latency
- **Initialize:** 30s timeout (MongoDB connection)
- **Regular requests:** 10s timeout
- **Response overhead:** <1ms (ID-based routing)

### Resource Usage
- **Memory:** ~10MB baseline + 1KB per pending request
- **Goroutines:** 1 main + 1 response handler + N request handlers
- **File Descriptors:** stdin/stdout pipes to MCP server

## Troubleshooting

### "Broken Pipe" Errors
**Symptoms:** Errors when UI polls, especially with multiple concurrent requests

**Root Cause:** Pre-fix race condition in response handling

**Solution:** Update to version with background response handler (current version)

**Verification:** Run `TestConcurrentRequests` - should pass with 100% success

### Request Timeouts
**Symptoms:** Requests timing out after 10 seconds

**Possible Causes:**
1. MCP server slow to respond (check MongoDB connection)
2. MCP server crashed (check stderr logs)
3. Too many concurrent requests (check pending request count)

**Debugging:**
```bash
# Check MCP server stderr
./mcp-http-bridge 2>&1 | grep "MCP"

# Monitor pending requests
# Add logging to pendingReqs map size
```

### Response Mismatch
**Symptoms:** Wrong response returned for request

**Root Cause:** Request ID collision or routing bug

**Verification:** Run `TestRequestResponseMatching`

**Solution:** Ensure unique request IDs (use `X-Request-ID` header)

### Memory Leaks
**Symptoms:** Memory growing over time, pending request map not cleaned up

**Verification:** Run `TestPendingRequestCleanup`

**Debugging:**
```bash
# Check goroutine count
go tool pprof http://localhost:7095/debug/pprof/goroutine

# Check heap allocations
go tool pprof http://localhost:7095/debug/pprof/heap
```

## Future Enhancements

1. **WebSocket Support:** Enable bidirectional streaming for long-running tools
2. **Request Queueing:** Rate limiting and backpressure handling
3. **Metrics Endpoint:** Expose Prometheus metrics for monitoring
4. **Health Checks:** Deep health checks including MCP server responsiveness
5. **Multiple MCP Servers:** Load balancing across multiple MCP server instances
6. **Caching Layer:** Cache tool list and resource list responses

## Related Documentation

- **MCP Server:** `../mcp-server/README.md`
- **Test Suite:** `TEST_README.md`
- **Coordinator Specification:** `../SPECIFICATION.md`
- **Dashboard UI:** `../../dashboard-ui/README.md`

## Version History

### v1.0.0 (Current)
- ✅ Fixed broken pipe bug with background response handler
- ✅ Added concurrent request support with pendingReqs map
- ✅ Comprehensive test suite (60.3% coverage)
- ✅ Benchmark tests for performance validation
- ✅ CORS configuration for frontend development

### v0.1.0 (Initial)
- ❌ Basic HTTP → MCP translation
- ❌ Sequential request handling only
- ❌ Race conditions with concurrent requests
- ❌ No test coverage