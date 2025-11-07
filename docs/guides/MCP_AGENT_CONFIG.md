# MCP Configuration for Agent Access to Hyperion Coordinator

## Problem
Sub-agents launched via the Task tool cannot access the hyper MCP server, preventing them from updating task statuses in real-time.

## Required Configuration

All agents working on the AI Band Manager project (and any Hyperion project) need access to:

### 1. Hyperion Coordinator MCP Server
```json
{
  "mcpServers": {
    "hyper": {
      "command": "node",
      "args": [
        "/Users/alcwynparker/Documents/2025/2025-09-30-dev-ex-mcp/coordinator/mcp-http-bridge/dist/index.js"
      ],
      "env": {
        "COORDINATOR_API_URL": "http://localhost:3000",
        "LOG_LEVEL": "info"
      }
    }
  }
}
```

### 2. Qdrant MCP Server (for knowledge sharing)
```json
{
  "mcpServers": {
    "qdrant-mcp": {
      "command": "npx",
      "args": [
        "-y",
        "@qdrant/mcp-server-qdrant"
      ],
      "env": {
        "QDRANT_URL": "http://localhost:6333",
        "QDRANT_API_KEY": ""
      }
    }
  }
}
```

## Agent Type MCP Access Matrix

According to CLAUDE.md, each agent type should have access to:

### Backend Services Specialist
- ✅ hyper (MANDATORY)
- ✅ qdrant-mcp (MANDATORY)
- ✅ @modelcontextprotocol/server-filesystem
- ✅ @modelcontextprotocol/server-github
- ✅ @modelcontextprotocol/server-fetch
- ✅ mcp-server-mongodb

### Frontend Experience Specialist
- ✅ hyper (MANDATORY)
- ✅ qdrant-mcp (MANDATORY)
- ✅ @modelcontextprotocol/server-filesystem
- ✅ @modelcontextprotocol/server-github
- ✅ playwright-mcp
- ✅ @modelcontextprotocol/server-fetch

### ui-dev
- ✅ hyper (MANDATORY)
- ✅ qdrant-mcp (MANDATORY)
- ✅ @modelcontextprotocol/server-filesystem
- ✅ @modelcontextprotocol/server-github
- ✅ @modelcontextprotocol/server-fetch
- ✅ playwright-mcp

### Event Systems Specialist
- ✅ hyper (MANDATORY)
- ✅ qdrant-mcp (MANDATORY)
- ✅ @modelcontextprotocol/server-filesystem
- ✅ @modelcontextprotocol/server-github
- ✅ @modelcontextprotocol/server-fetch

### Data Platform Specialist
- ✅ hyper (MANDATORY)
- ✅ qdrant-mcp (MANDATORY)
- ✅ @modelcontextprotocol/server-filesystem
- ✅ @modelcontextprotocol/server-github
- ✅ @modelcontextprotocol/server-fetch
- ✅ mcp-server-mongodb

## Current Issue

The Task tool in Claude Code does not automatically pass MCP server configurations to sub-agents. This means:

1. Main Claude Code process ✅ Can access hyper and qdrant
2. Sub-agents via Task tool ❌ Cannot access these MCPs

## Workaround Options

### Option 1: Manual Task Updates (Current Approach)
The main Claude Code process manually updates coordinator tasks after each step.

**Pros**: Works immediately
**Cons**: Not truly parallel, no real agent autonomy

### Option 2: Configure Claude Code Agent Settings
Update Claude Code's agent system configuration to include MCP server access for all agent types.

**Location**: This requires modifying Claude Code's internal agent configuration, which may be in:
- Application preferences
- User config directory
- Command-line flags

**Required**: Documentation from Anthropic on how to configure MCP access for sub-agents

### Option 3: Use HTTP Bridge for All Agents
Instead of relying on MCP tool inheritance, agents could call the coordinator HTTP bridge directly via fetch/curl.

**Example**:
```bash
curl -X POST http://localhost:3000/api/v1/tasks/status \
  -H "Content-Type: application/json" \
  -d '{"taskId": "123", "status": "in_progress"}'
```

**Pros**: Works around MCP inheritance issues
**Cons**: Bypasses MCP abstraction, requires agents to know HTTP endpoints

### Option 4: Shared MCP Configuration File
Create a shared MCP configuration that all agent processes can reference.

**Location**: `~/.config/claude-code/mcp-servers.json`

```json
{
  "mcpServers": {
    "hyper": {
      "command": "node",
      "args": ["/path/to/mcp-http-bridge/dist/index.js"],
      "env": {
        "COORDINATOR_API_URL": "http://localhost:3000"
      }
    },
    "qdrant-mcp": {
      "command": "npx",
      "args": ["-y", "@qdrant/mcp-server-qdrant"],
      "env": {
        "QDRANT_URL": "http://localhost:6333"
      }
    }
  }
}
```

## Recommended Solution

**For this demo**: Use Option 3 (HTTP Bridge) or Option 1 (Manual Updates)

**For production**: Wait for Claude Code to support MCP inheritance in sub-agents, or use Option 4 if a shared config file is supported.

## Testing

To verify an agent has MCP access:

```typescript
// Agent should be able to call:
mcp__hyperion_coordinator__coordinator_list_human_tasks({})

// If this fails with "tool not found", MCP access is not configured
```

## Next Steps

1. ✅ Document the issue (this file)
2. ⏳ Test if Claude Code supports shared MCP config files
3. ⏳ Implement HTTP bridge fallback for agents
4. ⏳ Request feature from Anthropic: MCP inheritance for sub-agents
