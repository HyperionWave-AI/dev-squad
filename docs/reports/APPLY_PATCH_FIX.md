# apply_patch Tool Fix - Path Extraction from Patch Headers

## Problem

The `apply_patch` MCP tool was returning "path cannot be empty" errors when patches were submitted without an explicit `filePath` parameter. The tool required both `filePath` and `patch` parameters, but standard unified diff patches already contain the file path in their headers.

### Error Logs
```
Tool call: apply_patch
Tool error: apply_patch - path cannot be empty
```

## Root Cause

1. **Location**: `hyper/internal/mcp/handlers/filesystem_tools.go`
2. **Issue**: The tool required `filePath` as a mandatory parameter in the schema (line 484)
3. **Problem**: Standard unified diff format already includes file paths in headers like:
   - `--- a/test.txt`
   - `+++ b/test.txt`
   - `*** Update File: test.txt` (custom format)

## Solution

### Changes Made

#### 1. Made `filePath` Optional
**File**: `hyper/internal/mcp/handlers/filesystem_tools.go:463-486`

**Before**:
```go
Required: []string{"filePath", "patch"},
```

**After**:
```go
Required: []string{"patch"},
```

Updated description to:
> "Optional: Absolute or relative path to the file to patch. If not provided, path will be extracted from patch headers"

#### 2. Added Path Extraction Logic
**File**: `hyper/internal/mcp/handlers/filesystem_tools.go:499-549`

Added new function `extractFilePathFromPatch()` that:
- Parses patch headers to extract file path
- Supports multiple formats:
  - `--- a/file` or `--- file`
  - `+++ b/file` or `+++ file`
  - `*** Update File: file` (custom format)
- Handles timestamps and `/dev/null` entries
- Returns clear error if no path found

#### 3. Updated Handler Logic
**File**: `hyper/internal/mcp/handlers/filesystem_tools.go:551-567`

**Before**:
```go
filePath, ok := args["filePath"].(string)
if !ok || filePath == "" {
    return createFilesystemErrorResult("filePath is required..."), nil
}
```

**After**:
```go
filePath, ok := args["filePath"].(string)
if !ok || filePath == "" {
    // Extract file path from patch headers
    extractedPath, err := h.extractFilePathFromPatch(patch)
    if err != nil {
        return createFilesystemErrorResult(fmt.Sprintf("filePath not provided and could not extract from patch: %s", err.Error())), nil
    }
    filePath = extractedPath
    h.logger.Info("Extracted file path from patch", zap.String("path", filePath))
}
```

## How It Works Now

### Option 1: Explicit File Path
```json
{
  "filePath": "test.txt",
  "patch": "--- a/test.txt\n+++ b/test.txt\n..."
}
```

### Option 2: Extract from Patch (NEW!)
```json
{
  "patch": "--- a/test.txt\n+++ b/test.txt\n@@ -1,3 +1,3 @@\n Line1\n-Line2\n+Line2-modified\n Line3\n"
}
```

The tool will extract `test.txt` from the `--- a/test.txt` header.

### Supported Patch Formats

#### Standard Unified Diff
```diff
--- a/test.txt
+++ b/test.txt
@@ -1,3 +1,3 @@
 Line1
-Line2
+Line2-modified
 Line3
```

#### Simple Format
```diff
--- test.txt
+++ test.txt
@@ -1,3 +1,3 @@
 Line1
-Line2
+Line2-modified
 Line3
```

#### Custom Format
```
*** Begin Patch
*** Update File: test.txt
@@
-Line2
+Line2-modified
*** End Patch
```

## Testing

### Test Case 1: Extract from --- header
```bash
curl -X POST http://localhost:7095/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "apply_patch",
      "arguments": {
        "patch": "--- a/test.txt\n+++ b/test.txt\n@@ -1,1 +1,1 @@\n-old\n+new\n"
      }
    }
  }'
```

### Test Case 2: Extract from +++ header
```bash
curl -X POST http://localhost:7095/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "apply_patch",
      "arguments": {
        "patch": "+++ b/test.txt\n@@ -1,1 +1,1 @@\n-old\n+new\n"
      }
    }
  }'
```

### Test Case 3: Explicit file path (backwards compatible)
```bash
curl -X POST http://localhost:7095/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "apply_patch",
      "arguments": {
        "filePath": "test.txt",
        "patch": "@@ -1,1 +1,1 @@\n-old\n+new\n"
      }
    }
  }'
```

## Benefits

1. **✅ Backwards Compatible**: Explicit `filePath` still works
2. **✅ Standards Compliant**: Accepts standard unified diff format
3. **✅ Flexible**: Supports multiple patch formats
4. **✅ User Friendly**: No need to specify path twice (in parameter and in patch)
5. **✅ Error Handling**: Clear error messages if path cannot be extracted

## Files Modified

- `hyper/internal/mcp/handlers/filesystem_tools.go`
  - Updated `registerApplyPatchTool()` to make `filePath` optional
  - Added `extractFilePathFromPatch()` helper function
  - Updated `handleApplyPatch()` to extract path from patch if not provided

## Next Steps

1. **Rebuild Binary**: `make clean && make native`
2. **Restart Server**: `./bin/hyper --mode=both`
3. **Test with Claude Code**: Try using apply_patch tool
4. **Verify Logs**: Check that path extraction works correctly

## Related Issues

This fix resolves the "path cannot be empty" errors that were occurring when using the `apply_patch` tool with standard unified diff format patches.

## Compatibility

- ✅ MCP Protocol version: 2024-11-05
- ✅ go-sdk version: github.com/modelcontextprotocol/go-sdk
- ✅ Unified diff format: Standard (git diff, diff -u)
- ✅ Custom formats: Supported with *** Update File: syntax
