# Human Prompt Notes Persistence Fix

## Problem
`humanPromptNotes`, `humanPromptNotesAddedAt`, and `humanPromptNotesUpdatedAt` fields were being saved to MongoDB correctly but were not persisting in the UI after page refresh or 30-second auto-reload.

## Root Cause
The MCP client's `listAgentTasks()` method was not mapping the `humanPromptNotes` fields when parsing the JSON response from the backend. The fields existed in the MongoDB data and backend response, but were dropped during the UI parsing step.

**Affected File**: `coordinator/ui/src/services/mcpClient.ts`

## Fix Applied

### 1. Agent Task Level (Lines 201-203)
Added mapping for agent task humanPromptNotes fields:
```typescript
humanPromptNotes: task.humanPromptNotes,
humanPromptNotesAddedAt: task.humanPromptNotesAddedAt,
humanPromptNotesUpdatedAt: task.humanPromptNotesUpdatedAt
```

### 2. TODO Item Level (Lines 182-195)
Explicitly mapped all TODO fields including humanPromptNotes to ensure nested fields are preserved:
```typescript
todos: (task.todos || []).map((todo: any) => ({
  id: todo.id,
  description: todo.description,
  status: todo.status,
  createdAt: todo.createdAt,
  completedAt: todo.completedAt,
  notes: todo.notes,
  filePath: todo.filePath,
  functionName: todo.functionName,
  contextHint: todo.contextHint,
  humanPromptNotes: todo.humanPromptNotes,              // ✅ ADDED
  humanPromptNotesAddedAt: todo.humanPromptNotesAddedAt,     // ✅ ADDED
  humanPromptNotesUpdatedAt: todo.humanPromptNotesUpdatedAt,  // ✅ ADDED
})),
```

## Testing Instructions

### Manual Test
1. **Start the services**:
   ```bash
   # Terminal 1: Start coordinator MCP server
   cd coordinator/mcp-server
   go run main.go

   # Terminal 2: Start UI
   cd coordinator/ui
   npm run dev
   ```

2. **Test Agent Task Notes**:
   - Open UI at http://localhost:5173
   - Create a human task (or use existing)
   - Create an agent task with TODOs
   - Open the agent task detail dialog
   - Expand "Add Human Guidance Notes"
   - Add notes: "Test notes for agent task"
   - Click "Save"
   - **BEFORE FIX**: Wait 30s or refresh → Notes disappear ❌
   - **AFTER FIX**: Wait 30s or refresh → Notes persist ✅

3. **Test TODO Item Notes**:
   - In the same agent task detail dialog
   - Find a TODO item
   - Click expand icon on the TODO
   - Click "Add Notes"
   - Add notes: "Test notes for TODO item"
   - Click "Save"
   - **BEFORE FIX**: Wait 30s or refresh → Notes disappear ❌
   - **AFTER FIX**: Wait 30s or refresh → Notes persist ✅

4. **Verify Timestamps**:
   - After saving notes, check browser console Network tab
   - Look for `coordinator_list_agent_tasks` response
   - Verify response includes:
     - `humanPromptNotes`: "your notes text"
     - `humanPromptNotesAddedAt`: ISO timestamp
     - `humanPromptNotesUpdatedAt`: ISO timestamp

### Automated Test (Future)
Create Playwright E2E test:
```typescript
test('human prompt notes persist after refresh', async ({ page }) => {
  // 1. Create task and add notes
  // 2. Refresh page
  // 3. Assert notes are still visible
});
```

## Verification Checklist
- [x] TypeScript compilation passes (`npm run build` ✅)
- [x] No console errors in browser
- [ ] Manual test: Agent task notes persist after refresh
- [ ] Manual test: TODO notes persist after refresh
- [ ] Manual test: Timestamps display correctly

## Related Files
- **Fix Applied**: `coordinator/ui/src/services/mcpClient.ts`
- **Type Definitions**: `coordinator/ui/src/types/coordinator.ts` (already correct)
- **Backend Storage**: `coordinator/mcp-server/storage/tasks.go` (already correct)
- **UI Components**:
  - `coordinator/ui/src/components/PromptNotesEditor.tsx` (agent task notes)
  - `coordinator/ui/src/components/TodoPromptNotes.tsx` (TODO notes)

## Impact
- **Risk**: Low (only adding missing field mappings)
- **Breaking Changes**: None
- **Database Changes**: None (fields already in DB)
- **API Changes**: None (backend already returning fields)

## Before & After

### Before Fix
```typescript
// mcpClient.ts - humanPromptNotes fields missing
return tasks.map((task: any) => ({
  id: task.id,
  // ... other fields ...
  todos: task.todos || [],  // ❌ TODOs not explicitly mapped
  // ❌ humanPromptNotes missing
  // ❌ humanPromptNotesAddedAt missing
  // ❌ humanPromptNotesUpdatedAt missing
}));
```

### After Fix
```typescript
// mcpClient.ts - all fields mapped
return tasks.map((task: any) => ({
  id: task.id,
  // ... other fields ...
  todos: (task.todos || []).map((todo: any) => ({
    // All todo fields explicitly mapped including humanPromptNotes
  })),
  humanPromptNotes: task.humanPromptNotes,              // ✅ ADDED
  humanPromptNotesAddedAt: task.humanPromptNotesAddedAt,     // ✅ ADDED
  humanPromptNotesUpdatedAt: task.humanPromptNotesUpdatedAt, // ✅ ADDED
}));
```

## Additional Notes

### Why This Happened
The original implementation used:
```typescript
todos: task.todos || []
```

This shallow copy kept the array reference but didn't guarantee all nested object properties were preserved during the mapping transformation. When combined with the fact that the parent task object was being explicitly mapped field-by-field, any fields not explicitly listed were dropped.

### Prevention
For future fields:
1. **Always explicitly map all fields** in `listAgentTasks()` and `listHumanTasks()`
2. **Add E2E tests** for persistence after refresh
3. **Validate type completeness** between backend Go structs and frontend TypeScript types
4. **Consider using code generation** to auto-sync types from backend to frontend

---

**Fixed by**: Claude Code
**Date**: 2025-10-03
**Ticket**: humanPromptNotes persistence issue
