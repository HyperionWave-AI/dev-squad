# list_directory Tool Fix - Absolute Path Support

## Problem

The `list_directory` tool was returning **relative paths** in the file entries, which caused issues when trying to use those paths with `read_file`:

```json
// Input
{"path": "."}

// Output (BEFORE FIX) ❌
{
  "entries": [
    {
      "name": "README.md",
      "path": "./README.md",  // ❌ Relative path
      ...
    }
  ]
}

// Trying to read the file fails:
read_file({"path": "./README.md"})  // ❌ Error: path not found
```

## Root Cause

When given a relative path input (like `"."`), the tool was using that relative path directly with `filepath.Join()`, resulting in relative paths in the output.

## Solution

Modified `ListDirectoryTool.Call()` to **convert input path to absolute path** before listing:

```go
// Convert to absolute path to ensure consistent paths in output
absPath, err := filepath.Abs(listInput.Path)
if err != nil {
    return "", fmt.Errorf("failed to get absolute path: %w", err)
}
listInput.Path = absPath
```

Now both **recursive** and **non-recursive** modes use absolute paths.

## After Fix ✅

```json
// Input (can still be relative)
{"path": "."}

// Output (AFTER FIX) ✅
{
  "path": "/Users/maxmednikov/MaxSpace/dev-squad/hyper",
  "entries": [
    {
      "name": "README.md",
      "path": "/Users/maxmednikov/MaxSpace/dev-squad/hyper/README.md",  // ✅ Absolute path
      "size": 12345,
      "isDir": false,
      ...
    }
  ]
}

// Reading the file now works:
read_file({"path": "/Users/.../hyper/README.md"})  // ✅ Success
```

## Changes Made

### 1. **file_tool.go** - Core Logic
- Added absolute path conversion in `ListDirectoryTool.Call()`
- Updated `FileInfo` struct documentation to clarify absolute paths
- Updated tool description to mention absolute paths

### 2. **filesystem_registration.go** - Tool Executor
- Updated description to clarify absolute paths are returned

## Benefits

✅ **Consistent paths** - All file paths are absolute, regardless of input  
✅ **Works with read_file** - Paths can be directly used with other tools  
✅ **Cross-platform** - Absolute paths work on Windows, macOS, Linux  
✅ **Backwards compatible** - Still accepts both relative and absolute input  

## Files Modified

- `internal/ai-service/tools/file_tool.go`
- `internal/ai-service/tools/filesystem_registration.go`

## Testing

```bash
# All binaries rebuild successfully
make clean && make build  # ✅

# Test output
bin/
├── hyper-coordinator    24M  ✅
├── hyper-mcp-server     17M  ✅
├── hyper-indexer        15M  ✅
└── hyper-bridge        5.8M  ✅
```

## Example Usage

```json
// List current directory
{
  "path": ".",
  "recursive": false,
  "showHidden": false
}

// Returns entries with absolute paths:
{
  "path": "/absolute/path/to/directory",
  "entries": [
    {
      "name": "file.txt",
      "path": "/absolute/path/to/directory/file.txt",  // ✅ Can use with read_file
      "size": 1234,
      "isDir": false,
      "modTime": "2025-10-12 10:42:00",
      "permissions": "-rw-r--r--"
    }
  ],
  "count": 1
}
```

## Status

✅ **FIXED** - list_directory now returns absolute paths for all entries
