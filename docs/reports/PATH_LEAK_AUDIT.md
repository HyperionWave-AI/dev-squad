# Path Leak Audit Report

**Date:** 2025-10-25
**Issue:** File tools may expose full filesystem paths instead of virtual paths

---

## 🔍 Findings

### ✅ Already Secure (No Changes Needed)

#### 1. AI Service File Tools
**File:** `hyper/internal/ai-service/tools/file_tool.go`

- ✅ `ReadFileOutput.Path` uses `StripProjectRoot()` (line 97)
- ✅ `WriteFileOutput.Path` uses `StripProjectRoot()` (line 178)
- ✅ `ListDirectoryOutput.Directory` uses `StripProjectRoot()` (line 363)
- ✅ Error messages use `StripProjectRoot()` (lines 402, 404, 418, 458)

### ⚠️ Path Leaks Found

#### 1. ListDirectoryTool Error Message
**File:** `hyper/internal/ai-service/tools/file_tool.go:264`

**Issue:**
```go
return "", fmt.Errorf("path '%s' is a FILE, not a directory. To read file contents, use read_file tool with this path. To list directory, provide the parent directory path: %s", listInput.Path, parentDir)
```

**Problem:**
- `listInput.Path` is the absolute path after validation
- `parentDir` is also absolute
- Both leak full filesystem paths in error message

**Fix:**
```go
return "", fmt.Errorf("path '%s' is a FILE, not a directory. To read file contents, use read_file tool with this path. To list directory, provide the parent directory path: %s",
    StripProjectRoot(listInput.Path),
    StripProjectRoot(parentDir))
```

---

#### 2. MCP file_read Tool
**File:** `hyper/internal/mcp/handlers/filesystem_tools.go:349`

**Issue:**
```go
result := map[string]interface{}{
    "success":      true,
    "filePath":     validatedPath,  // ⚠️ ABSOLUTE PATH LEAKED
    "size":         fileInfo.Size(),
    // ...
}
```

**Fix:**
```go
"filePath":     tools.StripProjectRoot(validatedPath),
```

---

#### 3. MCP file_write Tool
**File:** `hyper/internal/mcp/handlers/filesystem_tools.go:474`

**Issue:**
```go
result := map[string]interface{}{
    "success":      true,
    "filePath":     validatedPath,  // ⚠️ ABSOLUTE PATH LEAKED
    "bytesWritten": bytesWritten,
    // ...
}
```

**Fix:**
```go
"filePath":     tools.StripProjectRoot(validatedPath),
```

---

#### 4. MCP apply_patch Tool
**File:** `hyper/internal/mcp/handlers/filesystem_tools.go:648`

**Issue:**
```go
result := map[string]interface{}{
    "success":      true,
    "filePath":     validatedPath,  // ⚠️ ABSOLUTE PATH LEAKED
    "dryRun":       dryRun,
    // ...
}
```

**Fix:**
```go
"filePath":     tools.StripProjectRoot(validatedPath),
```

---

#### 5. MCP bash Tool
**File:** `hyper/internal/mcp/handlers/filesystem_tools.go:212`

**Issue:**
```go
result := map[string]interface{}{
    "command":    command,
    "workingDir": workingDir,  // ⚠️ ABSOLUTE PATH LEAKED
    "stdout":     stdout.String(),
    "stderr":     stderr.String(),
    // ...
}
```

**Partial Fix:**
```go
"workingDir": tools.StripProjectRoot(workingDir),
```

**Note:** `stdout` and `stderr` contain actual command output which may include paths. These should ideally be sanitized but that's complex since they're arbitrary command output.

---

## 📊 Summary

| Tool | Location | Status | Severity |
|------|----------|--------|----------|
| ReadFileTool (AI) | file_tool.go:97 | ✅ Secure | N/A |
| WriteFileTool (AI) | file_tool.go:178 | ✅ Secure | N/A |
| ListDirectoryTool (AI) | file_tool.go:363 | ✅ Secure | N/A |
| ListDirectoryTool Error | file_tool.go:264 | ⚠️ Leak | Medium |
| MCP file_read | filesystem_tools.go:349 | ⚠️ Leak | High |
| MCP file_write | filesystem_tools.go:474 | ⚠️ Leak | High |
| MCP apply_patch | filesystem_tools.go:648 | ⚠️ Leak | High |
| MCP bash (workingDir) | filesystem_tools.go:212 | ⚠️ Leak | Medium |
| MCP bash (stdout/stderr) | filesystem_tools.go:213-214 | ⚠️ Leak | Low* |

*Note: stdout/stderr path leaks are expected since they contain arbitrary command output

---

## 🔧 Fixes Required

### 5 Files Need Updates

1. `hyper/internal/ai-service/tools/file_tool.go` (1 fix)
2. `hyper/internal/mcp/handlers/filesystem_tools.go` (4 fixes)

---

## 💡 Recommendations

### Priority 1 (High) - Security

Fix MCP tool path leaks:
- file_read, file_write, apply_patch all leak `validatedPath`
- These are user-facing outputs
- Easy to fix with `tools.StripProjectRoot()`

### Priority 2 (Medium) - Error Messages

Fix error message path leaks:
- ListDirectoryTool error message
- bash tool workingDir

### Priority 3 (Low) - Bash Output

Consider sanitizing bash stdout/stderr:
- Replace project root with `.` in output
- More complex, may break some use cases
- Leave as-is for now

---

## 🎯 Impact

**Before Fix:**
```json
{
  "success": true,
  "filePath": "/Users/maxmednikov/MaxSpace/hyper/src/main.go"
}
```

**After Fix:**
```json
{
  "success": true,
  "filePath": "./src/main.go"
}
```

---

## ✅ Test Plan

After applying fixes, verify:

1. **Read file:**
   ```json
   {"path": "./README.md"}
   ```
   Expected: `"filePath": "./README.md"`

2. **Write file:**
   ```json
   {"path": "./test.txt", "content": "hello"}
   ```
   Expected: `"filePath": "./test.txt"`

3. **List directory (error case):**
   ```json
   {"path": "./README.md"}
   ```
   Expected error: `"path './README.md' is a FILE..."`

4. **Bash:**
   ```json
   {"command": "pwd"}
   ```
   Expected: `"workingDir": "."`

---

**Total Fixes:** 5 path leaks
**Estimated Time:** 10 minutes
**Risk:** Low (simple string transformation)
