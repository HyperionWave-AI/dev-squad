# Port Configuration Guide

## Default Ports

By default, the Hyperion Coordinator MCP uses:
- **UI**: `http://localhost:5173`
- **HTTP Bridge**: `http://localhost:8095`

## Changing Ports (If Defaults Conflict)

If ports 5173 or 8095 are already in use on your machine, you can easily change them:

### Method 1: Using docker-compose.override.yml (Recommended)

1. **Copy the example file:**
   ```bash
   cp docker-compose.override.yml.example docker-compose.override.yml
   ```

2. **Edit `docker-compose.override.yml`:**
   ```yaml
   services:
     hyperion-http-bridge:
       ports:
         - "YOUR_PORT:8095"  # Change YOUR_PORT to any available port
     hyperion-ui:
       ports:
         - "YOUR_PORT:80"    # Change YOUR_PORT to any available port
   ```

3. **Restart containers:**
   ```bash
   docker-compose down
   docker-compose up -d
   ```

4. **Access on new ports:**
   - UI: `http://localhost:YOUR_UI_PORT`
   - Bridge: `http://localhost:YOUR_BRIDGE_PORT`

### Example Configurations

**Example 1: Ports 9095 and 9173**
```yaml
services:
  hyperion-http-bridge:
    ports:
      - "9095:8095"
  hyperion-ui:
    ports:
      - "9173:80"
```

**Example 2: Ports 10095 and 10173**
```yaml
services:
  hyperion-http-bridge:
    ports:
      - "10095:8095"
  hyperion-ui:
    ports:
      - "10173:80"
```

## CORS and Port Changes

**Good news:** Changing ports does **NOT** cause CORS issues!

### Why?
- **Nginx handles CORS at the edge** - The nginx reverse proxy manages CORS headers for ANY localhost port
- **Port-agnostic configuration** - Works with localhost:5173, localhost:9173, localhost:10173, or any port
- **No code changes needed** - CORS logic is in nginx config, not application code
- **Regex-based validation** - Only allows `http://localhost:*` origins for security

### How It Works

1. Browser makes request from `http://localhost:YOUR_PORT/`
2. Request includes `Origin: http://localhost:YOUR_PORT` header
3. Nginx validates origin matches `http://localhost:(any_port)` pattern
4. Nginx strips Origin header before forwarding to bridge (internal network)
5. Nginx adds CORS headers to response with the original origin
6. Browser receives response with proper CORS headers

### Internal vs External Ports

- **External (Host) Port**: The port you access from your browser (customizable)
- **Internal (Container) Port**: The port inside Docker (must stay fixed)

When you change `"9173:80"`:
- `9173` = External port (change this to avoid conflicts)
- `80` = Internal port (NEVER change this)

Docker handles the mapping automatically. The browser only sees the external port, and nginx handles everything internally.

### Security

The nginx CORS configuration only allows origins matching:
```
^https?://localhost(:[0-9]+)?$
```

This means:
- ✅ `http://localhost` (no port)
- ✅ `http://localhost:5173`
- ✅ `http://localhost:9173`
- ✅ `http://localhost:ANY_PORT`
- ✅ `https://localhost:ANY_PORT`
- ❌ `http://example.com` (rejected)
- ❌ `http://192.168.1.1` (rejected)

## Troubleshooting

### Check if ports are in use:
```bash
# macOS/Linux
lsof -iTCP -sTCP:LISTEN -P | grep -E ":(5173|8095)"

# Or check specific port
lsof -i :5173
```

### View current port mappings:
```bash
docker-compose ps
# or
docker ps --format "table {{.Names}}\t{{.Ports}}"
```

### Test new ports are working:
```bash
# Test bridge directly
curl http://localhost:YOUR_BRIDGE_PORT/health

# Test bridge via UI proxy
curl http://localhost:YOUR_UI_PORT/bridge-health

# Test MCP tools list
curl http://localhost:YOUR_UI_PORT/api/mcp/tools
```

## Important Notes

1. **Don't edit docker-compose.yml directly** - use the override file instead
2. **Keep internal ports unchanged** - only change the first port number in the mapping
3. **Restart containers** after changing ports for changes to take effect
4. **No code changes needed** - port changes are purely configuration
5. **CORS-free** - nginx proxy ensures all requests remain same-origin

## For Teams

Each team member can have their own `docker-compose.override.yml` with custom ports. The file is typically git-ignored, so everyone can use different ports without conflicts.

Add to `.gitignore`:
```
docker-compose.override.yml
```

Keep `docker-compose.override.yml.example` in git for reference.
