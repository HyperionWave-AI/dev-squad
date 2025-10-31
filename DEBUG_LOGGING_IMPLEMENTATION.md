# Debug Logging Implementation for MCP Tools Discovery

## Summary
Added comprehensive debug logging to the three discovery functions in `tools_discovery.go` to help diagnose why `hyperion-storage-api` returns 0 tools.

## Changes Made

### 1. Added Logger Field to ToolsDiscoveryHandler
**File:** `hyper/internal/mcp/handlers/tools_discovery.go`

- Added `logger *zap.Logger` field to the `ToolsDiscoveryHandler` struct (line 21)
- Updated `NewToolsDiscoveryHandler` constructor to accept logger parameter (line 61)
- Added zap import (line 13)

### 2. Enhanced discoverServerTools Function (lines 763-835)
Added debug logging at key points:
- **Before connection:** Logs server URL, header count, and header keys (no values for security)
- **After SDK client creation:** Logs client name and endpoint
- **Before connection attempt:** Logs connection attempt
- **Connection failure:** Logs failure with error details
- **Connection success:** Logs successful connection
- **Before ListTools call:** Logs API call attempt
- **ListTools failure:** Logs API call failure with error
- **ListTools success:** Logs result count
- **Per-tool discovery:** Logs each discovered tool name
- **Completion:** Logs total tools discovered

### 3. Enhanced discoverServerResources Function (lines 838-914)
Added same comprehensive logging pattern:
- URL and header information (keys only)
- SDK connection lifecycle
- ListResources API call status
- Per-resource discovery details
- Total resources discovered

### 4. Enhanced discoverServerPrompts Function (lines 917-1001)
Added same comprehensive logging pattern:
- URL and header information (keys only)
- SDK connection lifecycle
- ListPrompts API call status
- Per-prompt discovery details
- Total prompts discovered

### 5. Updated Call Sites
**Files modified:**
- `hyper/cmd/coordinator/main.go` (line 573): Added logger parameter to NewToolsDiscoveryHandler call
- `hyper/internal/server/http_server.go` (line 180): Added logger parameter to NewToolsDiscoveryHandler call

### 6. Updated Tests
**File:** `hyper/internal/mcp/handlers/tools_discovery_test.go`

- Added zap import
- Added `zap.NewNop()` logger to all test cases
- Fixed method names from lowercase (handleDiscoverTools) to uppercase (HandleDiscoverTools)

## Security Considerations
- **Header keys logged, NOT values:** The logging explicitly extracts only header key names to avoid exposing sensitive data like authentication tokens
- **Example:** If headers contain `{"Authorization": "Bearer secret"}`, only `"Authorization"` is logged

## Debug Output Example
When calling `mcp_add_server` for hyperion-storage-api, you will now see logs like:

```
DEBUG  Starting tool discovery  serverURL=http://hyperion-storage-api:8084/mcp headerCount=1 headerKeys=[Authorization]
DEBUG  Created MCP client and transport  clientName=hyperion-mcp-discovery endpoint=http://hyperion-storage-api:8084/mcp
DEBUG  Attempting SDK connection to MCP server
DEBUG  SDK connection successful, calling ListTools
DEBUG  ListTools API call successful  serverURL=http://hyperion-storage-api:8084/mcp toolCount=5
DEBUG  Discovered tool  serverURL=http://hyperion-storage-api:8084/mcp toolName=storage_list_buckets
DEBUG  Discovered tool  serverURL=http://hyperion-storage-api:8084/mcp toolName=storage_upload_file
...
DEBUG  Tool discovery complete  serverURL=http://hyperion-storage-api:8084/mcp totalTools=5
```

## Testing
To test the new debug logging with hyperion-storage-api:

```bash
# 1. Enable debug logging (set LOG_LEVEL=debug in .env.hyper.hot)
echo "LOG_LEVEL=debug" >> .env.hyper.hot

# 2. Restart coordinator
make dev-hot

# 3. Add hyperion-storage-api server
curl -X POST http://localhost:4097/api/mcp/servers \
  -H "Content-Type: application/json" \
  -d '{
    "serverName": "hyperion-storage-api",
    "serverUrl": "http://hyperion-storage-api:8084/mcp",
    "description": "Storage API MCP server"
  }'

# 4. Check coordinator logs for debug output
# You should see detailed logs showing:
# - Connection attempts
# - API call results
# - Number of tools discovered
# - Individual tool names
```

## Compilation Status
✅ Code compiles successfully
✅ Binary created: `/tmp/hyper_debug` (25MB)
✅ No compilation errors

## Next Steps for Debugging
1. Set `LOG_LEVEL=debug` in coordinator environment
2. Restart coordinator to pick up new binary with debug logging
3. Call `mcp_add_server` for hyperion-storage-api
4. Examine logs to identify:
   - Does SDK connection succeed?
   - Does ListTools API call succeed?
   - What is the actual tool count returned?
   - Are any errors occurring during discovery?

## Knowledge Base Storage
This implementation and findings should be stored in coordinator knowledge base:
```bash
mcp__hyper__coordinator_upsert_knowledge \
  collection="hyperion_project" \
  text="MCP Tools Discovery Debug Logging: Added comprehensive debug logging to discoverServerTools, discoverServerResources, and discoverServerPrompts functions. Logs header keys (not values), SDK connection status, API call results, and per-item discovery. Helps diagnose 0-tools issue with hyperion-storage-api."
```
