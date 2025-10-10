# UI MCP Cleanup - Complete Migration to REST API

## Summary

All direct MCP calls have been removed from the UI. The architecture now correctly follows:

```
UI → REST API → Storage Layer (NO MCP proxying)
```

## Files Changed

### 1. **services/codeClient.ts** - Replaced MCP calls with REST
**Before:** 168 lines of MCP tool calls for code indexing
**After:** 10 lines re-exporting restCodeClient

```typescript
// OLD: Direct MCP calls (FORBIDDEN)
await (mcpClient as any).callTool('code_index_add_folder', {...})
await (mcpClient as any).callTool('code_search_semantic', {...})

// NEW: REST API calls (CORRECT)
export { restCodeClient as codeClient } from './restCodeClient';
```

### 2. **services/mcpClient.ts** - Updated deprecation notice
- Added clear warning that this file violates architecture
- Documented correct REST clients to use instead
- File kept only for legacy compatibility (will be removed)

### 3. **services/restCodeClient.ts** - Fixed API contracts
- Updated `addFolder()` to use POST body instead of query params
- Fixed `removeFolder()` to use path parameter: `DELETE /api/code-index/remove-folder/:configId`
- Fixed `scan()` to use JSON body with required `folderPath`
- Added all search options (fileTypes, minScore, retrieve)

### 4. **types/codeIndex.ts** - Updated type definitions
- Added `SearchResult` fields from backend: fileId, relativePath, chunkNum, content, etc.
- Added `configId` to `IndexStatus.folders` for delete operations
- Maintained backward compatibility with optional legacy fields

### 5. **components/code/CodeIndexConfig.tsx** - Fixed integration
- Pass folderPath to scan() after adding folder
- Use folder.configId when deleting folders (not folderPath)
- Fixed refresh button to reload status (not trigger full rescan)

### 6. **components/code/CodeResults.tsx** - Updated display logic
- Fixed TypeScript error: handle optional fileName field
- Display content instead of excerpt (new format)
- Show line numbers with startingLineNumber from result
- Show "Full File" vs "Code Chunk" based on fullFileRetrieved flag

## Verification

### Build Status
✅ UI builds successfully with no errors
✅ TypeScript compilation passes
✅ No MCP client imports in components

### Architecture Compliance

**✅ CORRECT - All UI code now uses REST:**
- Tasks/Todos: `restClient.ts` → `/api/tasks`, `/api/agent-tasks`
- Code Index: `restCodeClient.ts` → `/api/code-index/*`
- Knowledge: `knowledgeApi.ts` → `/api/knowledge/*`

**✅ NO VIOLATIONS:**
- Zero direct MCP tool calls from UI
- Zero `mcpClient` imports (except deprecated file itself)
- Zero `callTool` usage in active code

### Files Using REST (Correct)
- ✅ `restClient.ts` - Tasks, todos, knowledge operations
- ✅ `restCodeClient.ts` - Code index operations
- ✅ `knowledgeApi.ts` - Knowledge base queries
- ✅ All React components using above clients

### Files NOT Used (Deprecated)
- ⚠️ `mcpClient.ts` - Deprecated, has warning header, not imported
- ⚠️ Old `codeClient.ts` implementation - Replaced with REST re-export

## Testing Checklist

When testing the unified coordinator:

1. **Code Index Operations:**
   - [ ] Add folder via UI (POST /api/code-index/add-folder)
   - [ ] Scan folder (POST /api/code-index/scan)
   - [ ] Search code (POST /api/code-index/search)
   - [ ] Remove folder (DELETE /api/code-index/remove-folder/:configId)
   - [ ] Check status (GET /api/code-index/status)

2. **Verify No MCP Usage:**
   - [ ] Open browser DevTools Network tab
   - [ ] Confirm all requests go to `/api/*` endpoints
   - [ ] Confirm NO requests to `/mcp/*` endpoints
   - [ ] Verify response format matches REST API DTOs

## Architecture Diagram

```
┌─────────────────┐
│   React UI      │
│                 │
│  - restClient   │  ──────┐
│  - restCodeClient│       │
│  - knowledgeApi │        │
└─────────────────┘        │
                           │ HTTP/JSON
                           │ (REST API)
                           ▼
┌──────────────────────────────────────┐
│   Unified Coordinator                │
│   (Port 7095)                        │
│                                      │
│   internal/api/rest_handler.go       │
│   - TaskStorage                      │
│   - CodeIndexStorage                 │
│   - KnowledgeStorage                 │
│   - QdrantClient                     │
│   - EmbeddingClient                  │
└──────────────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────┐
│   Data Layer                         │
│   - MongoDB (tasks, files, chunks)   │
│   - Qdrant (vectors)                 │
└──────────────────────────────────────┘
```

## Benefits

1. **Clean Architecture:** UI → REST → Storage (no MCP proxying)
2. **Better Performance:** Direct storage access, no JSON-RPC overhead
3. **Type Safety:** Proper TypeScript DTOs matching backend
4. **Maintainability:** Single source of truth for API contracts
5. **Testability:** Easy to mock REST endpoints

## Next Steps

1. Start unified coordinator: `./bin/coordinator --mode=http`
2. Test all code index operations through UI
3. Remove deprecated `mcpClient.ts` after confirming no usage
4. Consider removing MCP HTTP bridge if no longer needed

---

**Status:** ✅ Complete - No MCP calls in UI, all REST API
**Date:** 2025-10-10
**Architecture:** UI → REST API → Storage Layer
