# Architecture Enforcement - ESLint Rules

## Overview

This document describes the ESLint rules that enforce the Hyperion UI architecture, preventing violations of the REST API abstraction layer.

## Rule: No Direct MCP Usage

**Rule:** `no-restricted-imports` for `mcpClient.ts`

**Location:** `eslint.config.js`

### What It Does

Prevents React components from importing and using `mcpClient.ts` directly. All coordinator operations (tasks, knowledge, agent tasks) must go through the REST API layer.

### Architecture

```
✅ CORRECT:
UI Components → restClient.ts → REST API (/api/*) → MCP Bridge → MCP Server

❌ FORBIDDEN:
UI Components → mcpClient.ts → MCP Bridge → MCP Server
```

### Error Message

```
'../../services/mcpClient' import is restricted from being used.
Direct MCP calls are prohibited. Use ../../services/restClient instead
for task and knowledge operations. Only codeClient.ts should use MCP
tools for code indexing.
```

### Exceptions

1. **`mcpClient.ts` itself** - Allowed to exist (deprecated, will be removed)
2. **`codeClient.ts`** - Allowed to use MCP for code indexing operations

### What to Use Instead

| Old (mcpClient) | New (restClient) |
|----------------|------------------|
| `mcpClient.listHumanTasks()` | `restClient.listHumanTasks()` |
| `mcpClient.listAgentTasks()` | `restClient.listAgentTasks()` |
| `mcpClient.createHumanTask()` | `restClient.createHumanTask()` |
| `mcpClient.updateTaskStatus()` | `restClient.updateTaskStatus()` |
| `mcpClient.updateTodoStatus()` | `restClient.updateTodoStatus()` |
| `mcpClient.queryKnowledge()` | `restClient.queryKnowledge()` |
| `mcpClient.upsertKnowledge()` | `restClient.upsertKnowledge()` |
| `mcpClient.addTaskPromptNotes()` | `restClient.addTaskPromptNotes()` |
| `mcpClient.addTodoPromptNotes()` | `restClient.addTodoPromptNotes()` |

### Running the Linter

```bash
# Check for violations
npm run lint

# Auto-fix where possible
npm run lint -- --fix
```

### How It Works

The rule uses ESLint's `no-restricted-imports` with multiple patterns:

1. **Exact paths:** `./services/mcpClient`, `../services/mcpClient`, `../../services/mcpClient`
2. **Glob pattern:** `**/services/mcpClient`

This catches all import variations across the component tree.

### Testing

The rule is tested in CI/CD:
- Pre-commit hooks run linter
- Build process includes lint check
- Pull requests must pass linting

### When to Update

If you add new coordinator operations:

1. ✅ Add to `restClient.ts` (REST API calls)
2. ✅ Add corresponding REST endpoint in `mcp-http-bridge/main.go`
3. ✅ Update this documentation with the new method mapping
4. ❌ DO NOT add to `mcpClient.ts` - it's deprecated

### Historical Context

**Problem:** UI was calling MCP tools directly via `/mcp/tools/call`, violating architecture.

**Solution:**
- Created REST API layer in `mcp-http-bridge/main.go`
- Created `restClient.ts` for UI to use
- Added ESLint rule to prevent future violations
- Deprecated `mcpClient.ts` with clear warnings

**Date Implemented:** 2025-10-10

### Related Documentation

- `restClient.ts` - REST API client implementation
- `../mcp-http-bridge/main.go` - REST API endpoints
- `CLAUDE.md` - General project guidelines
