# CRITICAL BUG: read_file Tool Schema Mismatch

**Date:** 2025-10-12
**Severity:** CRITICAL
**Status:** ROOT CAUSE IDENTIFIED

## Problem Summary

AI makes 26+ consecutive `read_file` calls with empty `path` parameter. All calls fail with "path cannot be empty" error. AI does not learn from errors and keeps retrying.

## Root Cause

**Schema Mismatch Between Registration and Implementation**

### What's Declared to AI (LangChain Schema)
File: `coordinator/ai-service/tools/filesystem_registration.go:76-87`
```go
func (r *ReadFileToolExecutor) InputSchema() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "path": map[string]interface{}{  // ← AI is told parameter is "path"
                "type": "string",
                "description": "Absolute or relative file path to read",
            },
        },
        "required": []string{"path"},  // ← Required parameter is "path"
    }
}
```

### What's Actually Expected (Implementation)
File: `coordinator/ai-service/tools/file_tool.go:26-29`
```go
type ReadFileInput struct {
    FilePath string `json:"filePath"`  // ← Implementation expects "filePath" (camelCase)
    MaxBytes int    `json:"maxBytes,omitempty"`
}
```

### What Happens

1. **AI sends:** `{"path": "/some/file.txt"}`
2. **JSON unmarshal:** `json.Unmarshal()` attempts to map to `ReadFileInput`
3. **Mismatch:** JSON key "path" doesn't match struct field "filePath"
4. **Result:** `readInput.FilePath` is **empty string** (zero value)
5. **Validation fails:** Line 344 validates `readInput.FilePath` → `"path cannot be empty"`
6. **AI confusion:** "I provided path, why does it say empty?" → Retry with same approach

### Why AI Doesn't Learn

The error message is **misleading**:
```
Error: "path cannot be empty"
```

AI interprets this as:
- "The path value I provided is invalid"
- "Maybe I should try a different path format"
- "Let me retry with the same approach"

AI doesn't realize:
- The **parameter name itself is wrong**
- Schema says "path" but implementation expects "filePath"
- The value is discarded during unmarshaling

## Impact Analysis

**Affected Tools:**
- `read_file` - Schema says "path", expects "filePath" ❌
- `write_file` - Schema says "path", expects "filePath" ❌
- `list_directory` - Schema says "path", expects "Path" ❌

**Severity:**
- **CRITICAL** - Core filesystem tools completely broken
- AI cannot read ANY files
- AI cannot write ANY files
- AI cannot list ANY directories
- Infinite retry loops waste tokens and time

## Recommended Fix

**Option A: Fix Schema to Match Implementation** (RECOMMENDED)

Change `filesystem_registration.go` to use `filePath`:

```go
// ReadFileToolExecutor - FIX SCHEMA
func (r *ReadFileToolExecutor) InputSchema() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "filePath": map[string]interface{}{  // ← Changed from "path"
                "type": "string",
                "description": "Absolute or relative file path to read",
            },
        },
        "required": []string{"filePath"},  // ← Changed from "path"
    }
}

// WriteFileToolExecutor - FIX SCHEMA
func (w *WriteFileToolExecutor) InputSchema() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "filePath": map[string]interface{}{  // ← Changed from "path"
                "type": "string",
                "description": "Absolute or relative file path to write",
            },
            "content": map[string]interface{}{
                "type": "string",
                "description": "Content to write to file",
            },
        },
        "required": []string{"filePath", "content"},  // ← Changed "path" to "filePath"
    }
}

// ListDirectoryToolExecutor - FIX SCHEMA
func (l *ListDirectoryToolExecutor) InputSchema() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "path": map[string]interface{}{  // ← CORRECT (matches ListDirectoryInput.Path)
                "type": "string",
                "description": "Directory path to list",
            },
            "recursive": map[string]interface{}{
                "type": "boolean",
                "description": "List subdirectories recursively (default: false)",
            },
        },
        "required": []string{"path"},
    }
}
```

**Why Option A:**
- ✅ Follows camelCase convention (filePath is correct)
- ✅ No breaking changes to existing code
- ✅ Schema accurately reflects implementation contract
- ✅ Fixes the root cause directly

**Option B: Change Implementation to Match Schema** (NOT RECOMMENDED)

Change `file_tool.go` JSON tags to lowercase:

```go
type ReadFileInput struct {
    FilePath string `json:"path"` // ← Change from "filePath"
    MaxBytes int    `json:"maxBytes,omitempty"`
}

type WriteFileInput struct {
    FilePath   string `json:"path"` // ← Change from "filePath"
    Content    string `json:"content"`
    CreateDirs bool   `json:"createDirs,omitempty"`
}
```

**Why NOT Option B:**
- ❌ Breaks camelCase naming convention
- ❌ Inconsistent with other tools (maxBytes is camelCase)
- ❌ Breaking change if anything else calls these tools
- ❌ Doesn't fix the fundamental design issue

## Verification Steps

After applying fix:

1. **Test read_file:**
   ```json
   {"filePath": "/tmp/test.txt"}
   ```
   Expected: File contents returned

2. **Test write_file:**
   ```json
   {"filePath": "/tmp/test.txt", "content": "hello"}
   ```
   Expected: File created successfully

3. **Test list_directory:**
   ```json
   {"path": "/tmp"}
   ```
   Expected: Directory listing returned (already correct)

## Prevention

**Code Review Checklist:**
- [ ] Schema parameter names EXACTLY match JSON struct tags
- [ ] Test each tool with actual AI calls before merge
- [ ] Add integration tests that validate schema matches implementation
- [ ] Use type-safe schema generation from structs (future improvement)

**Automated Testing:**
```go
// Add to tools_test.go
func TestSchemaMatchesImplementation(t *testing.T) {
    // For each tool, verify InputSchema() required fields
    // match the JSON struct tags exactly

    // Example:
    readTool := &ReadFileToolExecutor{tool: &ReadFileTool{}}
    schema := readTool.InputSchema()
    required := schema["required"].([]string)

    // Verify "filePath" is in required (not "path")
    assert.Contains(t, required, "filePath")
    assert.NotContains(t, required, "path")
}
```

## Files to Modify

1. **coordinator/ai-service/tools/filesystem_registration.go**
   - Lines 80-85: Change "path" → "filePath" in ReadFileToolExecutor.InputSchema()
   - Lines 125-134: Change "path" → "filePath" in WriteFileToolExecutor.InputSchema()
   - Lines 174-183: KEEP "path" in ListDirectoryToolExecutor.InputSchema() (already correct)

2. **coordinator/ai-service/tools/tools_test.go** (create if missing)
   - Add schema validation tests

## Estimated Fix Time

- Code changes: **2 minutes**
- Testing: **5 minutes**
- Verification with AI: **3 minutes**
- **Total: 10 minutes**

## Priority

**IMMEDIATE** - This blocks ALL filesystem operations for AI agents.

---

**Analyst:** Backend Services Specialist
**Task ID:** 981ebb2c-f57d-4632-afe8-d5cdda3277f8
**Next Action:** Apply fix to `filesystem_registration.go` immediately
