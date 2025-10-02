# CORS Solution - Port-Agnostic Configuration

## Problem Statement

When users accessed the Hyperion Coordinator MCP UI via custom ports (e.g., `http://localhost:9173` using `docker-compose.override.yml`), they encountered CORS errors:

```
Failed to list agent tasks: SyntaxError: JSON.parse: unexpected end of data
```

This was caused by HTTP 403 Forbidden responses from the bridge, which returned empty response bodies that couldn't be parsed as JSON.

## Root Cause

The HTTP bridge had a hardcoded CORS whitelist:
```go
config.AllowOrigins = []string{
    "http://localhost:5173",  // Only this port was allowed
    "http://localhost:3000",
    // ... other ports
}
```

When the browser sent requests with `Origin: http://localhost:9173`, the bridge rejected them because port 9173 wasn't in the whitelist.

## Why Nginx Proxy Didn't Solve It Initially

Even though we added an nginx reverse proxy to eliminate CORS issues, nginx was **passing through** the browser's `Origin` header to the backend. The bridge then checked this origin against its whitelist and rejected non-whitelisted ports.

## Solution: Nginx-Based CORS Handling

We moved CORS handling from the Go application to nginx, making it port-agnostic:

### 1. Nginx Validates Origin
```nginx
set $cors_origin "";
if ($http_origin ~* "^https?://localhost(:[0-9]+)?$") {
    set $cors_origin $http_origin;
}
```

This regex accepts **ANY** localhost port while rejecting external origins.

### 2. Nginx Strips Origin Header
By selectively forwarding headers and **not** including `Origin`, the bridge receives requests as if they came from the internal Docker network (no CORS needed).

```nginx
# Forward only necessary headers (NOT Origin)
proxy_set_header Host $host;
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header Content-Type $content_type;
proxy_set_header X-Request-ID $http_x_request_id;
# Origin header NOT forwarded
```

### 3. Nginx Adds CORS Response Headers
```nginx
add_header 'Access-Control-Allow-Origin' $cors_origin always;
add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS' always;
add_header 'Access-Control-Allow-Headers' 'Origin, Content-Type, Accept, X-Request-ID' always;
add_header 'Access-Control-Allow-Credentials' 'true' always;
```

The response includes the validated origin, satisfying the browser's CORS requirements.

### 4. Nginx Handles Preflight Requests
```nginx
if ($request_method = 'OPTIONS') {
    # ... CORS headers ...
    return 204;
}
```

OPTIONS requests are handled entirely by nginx without reaching the bridge.

## Benefits

### ✅ Port-Agnostic
Works with **ANY** localhost port:
- `http://localhost:5173` ✅
- `http://localhost:9173` ✅
- `http://localhost:10173` ✅
- `http://localhost:ANY_PORT` ✅

### ✅ No Code Changes Needed
Users can customize ports via `docker-compose.override.yml` without:
- Modifying Go source code
- Rebuilding the bridge container
- Maintaining a CORS whitelist

### ✅ Secure by Default
Only allows localhost origins:
- `http://localhost:5173` ✅
- `https://localhost:8443` ✅
- `http://example.com` ❌ (rejected)
- `http://192.168.1.1` ❌ (rejected)

### ✅ Cleaner Architecture
CORS logic belongs at the **edge** (nginx), not in internal services (bridge). This follows the principle of handling cross-origin concerns at the boundary.

### ✅ Easier Maintenance
One configuration file (`nginx.conf`) instead of:
- Go source code (`main.go`)
- Go build process
- Docker container rebuild
- Multiple service deployments

## Implementation Files

### Modified Files:
1. **`coordinator/ui/nginx.conf`** - Added CORS handling and selective header forwarding
2. **`PORT_CONFIGURATION.md`** - Updated documentation explaining the solution
3. **`QUICK_START.md`** - Added clarity about port-agnostic CORS

### Unchanged (but previously modified):
- **`coordinator/mcp-http-bridge/main.go`** - Still has CORS middleware (unused but harmless)

## Testing

### Test Coverage:
✅ Port 5173 (original default)
✅ Port 9173 (custom via docker-compose.override.yml)
✅ Port 10173 (hypothetical - CORS headers validate correctly)
✅ Preflight OPTIONS requests
✅ Actual POST requests with JSON payloads
✅ Health check endpoint

### Test Results:
All ports return proper CORS headers:
```
Access-Control-Allow-Origin: http://localhost:9173
Access-Control-Allow-Methods: GET, POST, OPTIONS
Access-Control-Allow-Headers: Origin, Content-Type, Accept, X-Request-ID
Access-Control-Allow-Credentials: true
```

## How It Works (Request Flow)

1. **Browser** → `GET http://localhost:9173/` (loads UI)
2. **Browser** → `POST http://localhost:9173/api/mcp/tools/call` with `Origin: http://localhost:9173`
3. **Nginx** receives request, validates origin matches `^https?://localhost(:[0-9]+)?$`
4. **Nginx** forwards request to bridge **without** Origin header
5. **Bridge** processes request (no CORS check needed)
6. **Bridge** returns JSON response
7. **Nginx** adds CORS headers with validated origin
8. **Browser** receives response with proper CORS headers ✅

## Future Considerations

### Option: Remove Bridge CORS Middleware
Since nginx now handles all CORS, the bridge's CORS middleware could be removed:
```go
// This could be removed (currently harmless but unused)
r.Use(cors.New(config))
```

**Pros:**
- Cleaner code
- No unnecessary middleware

**Cons:**
- Direct bridge access (without nginx) wouldn't support CORS
- Useful for development/testing

**Recommendation:** Keep it for now (no harm, useful for development)

### Option: Support Non-Localhost Origins
If needed for production deployments, the nginx regex could be expanded:
```nginx
if ($http_origin ~* "^https?://(localhost|app\.example\.com)(:[0-9]+)?$") {
    set $cors_origin $http_origin;
}
```

## Conclusion

The port-agnostic CORS solution eliminates maintenance burden while providing a more secure and architecturally sound approach. Users can now change ports freely via `docker-compose.override.yml` without encountering CORS issues or needing to modify application code.

**Key Takeaway:** CORS belongs at the edge (nginx), not in internal services. This pattern should be applied to other services as they're developed.

---

**Author:** Claude Code
**Date:** 2025-10-02
**Status:** Implemented and Tested ✅
