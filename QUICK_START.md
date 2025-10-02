# Quick Start - Hyperion Coordinator MCP

## Default Setup

### 1. Start the services
```bash
docker-compose up -d
```

### 2. Access the UI
Open your browser to: **http://localhost:5173**

### 3. Verify it's working
```bash
# Check health
curl http://localhost:5173/bridge-health

# List available MCP tools
curl http://localhost:5173/api/mcp/tools
```

## If Ports Are Already in Use

If you see errors like "port already in use", you can easily change to different ports:

### Quick Fix
```bash
# 1. Copy the override template
cp docker-compose.override.yml.example docker-compose.override.yml

# 2. Edit docker-compose.override.yml and change the ports:
#    - Change "9095:8095" to "YOUR_PORT:8095"
#    - Change "9173:80" to "YOUR_PORT:80"

# 3. Restart
docker-compose down
docker-compose up -d

# 4. Access on new port
# Open: http://localhost:YOUR_PORT
```

### Example
If ports 5173 and 8095 are taken, use 9173 and 9095:
```yaml
# docker-compose.override.yml
services:
  hyperion-http-bridge:
    ports:
      - "9095:8095"
  hyperion-ui:
    ports:
      - "9173:80"
```

Access at: **http://localhost:9173**

## No CORS Issues!

The nginx reverse proxy handles CORS at the edge with port-agnostic configuration:
- ✅ Works with **ANY localhost port** (5173, 9173, 10173, etc.)
- ✅ **No code changes needed** when changing ports
- ✅ **Secure by default** - only allows localhost origins
- ✅ **Automatic configuration** - nginx validates and reflects the origin

Change ports freely without worrying about CORS whitelists or rebuilding containers!

## More Information

- **Full port configuration guide**: See [PORT_CONFIGURATION.md](PORT_CONFIGURATION.md)
- **Architecture details**: See [DOCKER.md](DOCKER.md)
- **MCP server docs**: See [coordinator/mcp-server/README.md](coordinator/mcp-server/README.md)
