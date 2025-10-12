# Filesystem Tools Usage Examples

## Quick Start

All tools implement the LangChain Go `tools.Tool` interface and accept JSON input.

### BashTool

```go
import "hyperion-coordinator/ai-service/tools"

// Create tool
bashTool := &tools.BashTool{}

// Execute command
ctx := context.Background()
input := `{"command":"ls -la /tmp","timeout":10}`
output, err := bashTool.Call(ctx, input)

// Parse output
var result tools.BashOutput
json.Unmarshal([]byte(output), &result)

fmt.Printf("Exit Code: %d\n", result.ExitCode)
fmt.Printf("Stdout: %s\n", result.Stdout)
fmt.Printf("Duration: %dms\n", result.Duration)
```

**Output Example**:
```json
{
  "stdout": "total 16\ndrwxr-xr-x  4 user  wheel  128 Oct 12 05:00 .\ndrwxr-xr-x  3 user  wheel   96 Oct 12 04:59 ..\n",
  "stderr": "",
  "exitCode": 0,
  "durationMs": 15
}
```

### ReadFileTool

```go
readTool := &tools.ReadFileTool{}

input := `{"filePath":"/tmp/test.txt","maxBytes":1024}`
output, err := readTool.Call(ctx, input)

var result tools.ReadFileOutput
json.Unmarshal([]byte(output), &result)

fmt.Printf("Encoding: %s\n", result.Encoding)  // "utf8" or "binary"
fmt.Printf("Size: %d bytes\n", result.Size)
fmt.Printf("Content: %s\n", result.Content)
```

**Output Example**:
```json
{
  "path": "/tmp/test.txt",
  "content": "Hello, World!\n",
  "size": 14,
  "encoding": "utf8"
}
```

### WriteFileTool

```go
writeTool := &tools.WriteFileTool{}

input := `{
  "filePath": "/tmp/output.txt",
  "content": "This is a test\nLine 2\n",
  "createDirs": true
}`
output, err := writeTool.Call(ctx, input)

var result tools.WriteFileOutput
json.Unmarshal([]byte(output), &result)

fmt.Printf("Wrote %d bytes to %s\n", result.BytesWritten, result.Path)
```

**Output Example**:
```json
{
  "path": "/tmp/output.txt",
  "bytesWritten": 27
}
```

### ListDirectoryTool

```go
listTool := &tools.ListDirectoryTool{}

input := `{
  "path": "/tmp",
  "recursive": true,
  "showHidden": false
}`
output, err := listTool.Call(ctx, input)

var result tools.ListDirectoryOutput
json.Unmarshal([]byte(output), &result)

for _, entry := range result.Entries {
    fmt.Printf("%s %10d %s\n",
        entry.Permissions,
        entry.Size,
        entry.Name)
}
```

**Output Example**:
```json
{
  "path": "/tmp",
  "entries": [
    {
      "name": "test.txt",
      "path": "/tmp/test.txt",
      "size": 14,
      "isDir": false,
      "modTime": "2025-10-12 05:15:30",
      "permissions": "-rw-r--r--"
    },
    {
      "name": "subdir",
      "path": "/tmp/subdir",
      "size": 128,
      "isDir": true,
      "modTime": "2025-10-12 05:10:00",
      "permissions": "drwxr-xr-x"
    }
  ],
  "count": 2
}
```

### ApplyPatchTool

```go
patchTool := &tools.ApplyPatchTool{}

patch := `@@ -2,3 +2,3 @@
 line 2
-old line 3
+new line 3
 line 4`

input := fmt.Sprintf(`{
  "filePath": "/tmp/file.txt",
  "patch": %q,
  "dryRun": false
}`, patch)

output, err := patchTool.Call(ctx, input)

var result tools.ApplyPatchOutput
json.Unmarshal([]byte(output), &result)

if result.Success {
    fmt.Printf("Applied patch: %d lines changed\n", result.LinesChanged)
} else {
    fmt.Printf("Patch failed: %v\n", result.Errors)
}
```

**Output Example (Success)**:
```json
{
  "success": true,
  "linesChanged": 1,
  "errors": null
}
```

**Output Example (Failure)**:
```json
{
  "success": false,
  "linesChanged": 0,
  "errors": [
    "hunk at line 2: delete line mismatch at line 3: expected 'old line 3', got 'different line 3'"
  ]
}
```

## Security Best Practices

### Path Validation

```go
// ✅ GOOD - Absolute paths
input := `{"filePath":"/tmp/safe/file.txt"}`

// ❌ BAD - Path traversal blocked
input := `{"filePath":"../../../etc/passwd"}`  // Error: path traversal not allowed
```

### Command Safety

```go
// ✅ GOOD - Safe commands
input := `{"command":"ls -la /tmp"}`

// ❌ BAD - Dangerous commands blocked
input := `{"command":"rm -rf /"}`      // Error: dangerous command blocked
input := `{"command":"dd if=/dev/zero"}` // Error: dangerous command blocked
```

### File Size Limits

```go
// ✅ GOOD - Within limits
readInput := `{"filePath":"/tmp/5mb.txt"}` // OK if file < 10MB
writeInput := `{"filePath":"/tmp/out.txt","content":"..."}` // OK if < 5MB

// ❌ BAD - Exceeds limits
// 15MB file
readInput := `{"filePath":"/tmp/15mb.txt"}` // Error: file size exceeds limit

// 8MB content
writeInput := `{"content":"..."}`  // Error: content size exceeds limit
```

## Integration with LangChain

### Register Tools

```go
import (
    "github.com/tmc/langchaingo/tools"
    fstools "hyperion-coordinator/ai-service/tools"
)

// Create tool instances
bashTool := &fstools.BashTool{}
readTool := &fstools.ReadFileTool{}
writeTool := &fstools.WriteFileTool{}
listTool := &fstools.ListDirectoryTool{}
patchTool := &fstools.ApplyPatchTool{}

// Register with LangChain agent
allTools := []tools.Tool{
    bashTool,
    readTool,
    writeTool,
    listTool,
    patchTool,
}

// Use in agent
agent, err := agents.NewOpenAIAgent(
    llm,
    allTools,
    agents.WithMaxIterations(10),
)
```

### Tool Discovery

```go
// AI can discover tools by name and description
for _, tool := range allTools {
    fmt.Printf("Tool: %s\n", tool.Name())
    fmt.Printf("  Description: %s\n", tool.Description())
}

// Output:
// Tool: bash
//   Description: Execute shell commands and return stdout/stderr...
// Tool: readFile
//   Description: Read file contents with automatic encoding detection...
// Tool: writeFile
//   Description: Create or overwrite file with content...
// Tool: listDirectory
//   Description: List files and directories with metadata...
// Tool: applyPatch
//   Description: Apply unified diff patches to files...
```

## Environment Configuration

### ALLOWED_DIRS (Optional)

Restrict file operations to specific directories:

```bash
export ALLOWED_DIRS="/tmp:/home/user/workspace:/var/app"
```

When set, all file tools will reject paths outside these directories:

```go
// With ALLOWED_DIRS="/tmp:/home/user"

// ✅ Allowed
input := `{"filePath":"/tmp/file.txt"}`        // OK
input := `{"filePath":"/home/user/doc.txt"}`   // OK

// ❌ Blocked
input := `{"filePath":"/etc/passwd"}`          // Error: path outside allowed directories
input := `{"filePath":"/var/log/app.log"}`     // Error: path outside allowed directories
```

## Error Handling

### Timeout Errors

```go
input := `{"command":"sleep 60","timeout":5}`
output, err := bashTool.Call(ctx, input)

if err != nil {
    if strings.Contains(err.Error(), "timed out") {
        fmt.Println("Command exceeded timeout")
    }
}
// Error: command timed out after 5000ms
```

### File Not Found

```go
input := `{"filePath":"/tmp/nonexistent.txt"}`
output, err := readTool.Call(ctx, input)

// Error: file not found: stat /tmp/nonexistent.txt: no such file or directory
```

### Invalid JSON

```go
input := `{invalid json}`
output, err := bashTool.Call(ctx, input)

// Error: invalid input format: invalid character 'i' looking for beginning of object key string
```

## Advanced Usage

### Dry-Run Patch Validation

```go
// Validate patch without applying
input := `{
  "filePath": "/tmp/code.go",
  "patch": "...",
  "dryRun": true
}`
output, err := patchTool.Call(ctx, input)

var result tools.ApplyPatchOutput
json.Unmarshal([]byte(output), &result)

if result.Success {
    fmt.Println("Patch is valid, safe to apply")

    // Now apply for real
    input = `{"filePath":"/tmp/code.go","patch":"...","dryRun":false}`
    patchTool.Call(ctx, input)
}
```

### Recursive Directory Search

```go
// Find all .go files recursively
input := `{
  "path": "/Users/user/project",
  "recursive": true,
  "showHidden": false
}`
output, err := listTool.Call(ctx, input)

var result tools.ListDirectoryOutput
json.Unmarshal([]byte(output), &result)

for _, entry := range result.Entries {
    if !entry.IsDir && strings.HasSuffix(entry.Name, ".go") {
        fmt.Printf("Found: %s (%d bytes)\n", entry.Path, entry.Size)
    }
}
```

### Command with Timeout

```go
// Run long command with custom timeout
input := `{
  "command": "find / -name '*.log' 2>/dev/null",
  "timeout": 120
}`
output, err := bashTool.Call(ctx, input)

// Will timeout after 120 seconds instead of default 30
```

## Testing

See `tools_test.go` for comprehensive test examples covering:
- Success cases
- Error cases
- Timeout scenarios
- Security validation
- Edge cases
- Performance benchmarks
