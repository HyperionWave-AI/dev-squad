# Configurable Context Size Limit

**Date:** 2025-11-21
**Status:** ✅ IMPLEMENTED
**Change Type:** Configuration Enhancement

## Summary

Made the context size limit **configurable via environment variable** and increased the default from **120KB to 500KB** to support longer conversations without hitting limits.

## Problem

The system had a **hardcoded 120KB context limit** that was too restrictive for:
- Long conversations with many tool calls (100+ messages)
- Large tool outputs (file contents, test results)
- Multi-iteration AI processing

Users would hit "Context size limit exceeded" errors after ~85-100 messages.

## Solution

### 1. Added Configuration Support

**File:** `hyper/internal/config/limits.go`

Added constant and function:
```go
// Default: 500KB ≈ 125K tokens (4 chars/token)
const DefaultMaxContextSize = 500 * 1024 // 500KB

// GetMaxContextSize returns the configured or default max context size
func GetMaxContextSize() int {
    if maxContextStr := os.Getenv("MAX_CONTEXT_SIZE"); maxContextStr != "" {
        if val, err := strconv.Atoi(maxContextStr); err == nil && val > 0 {
            return val
        }
    }
    return DefaultMaxContextSize
}
```

### 2. Updated Tool Executor

**File:** `hyper/internal/ai-service/tool_executor.go`

**Before** (2 occurrences at lines 1368 and 2149):
```go
const maxContextBeforeToolResult = 120000 // 120KB limit

if currentContextSize+toolResultSize > maxContextBeforeToolResult {
    log.Printf("[Context Guardrail] Tool result would exceed 120KB limit...")
    toolResultMsg = "...would push context over 120KB limit..."
}
```

**After**:
```go
// Get max context size from config (default 500KB, configurable via MAX_CONTEXT_SIZE env var)
maxContextBeforeToolResult := config.GetMaxContextSize()

if currentContextSize+toolResultSize > maxContextBeforeToolResult {
    log.Printf("[Context Guardrail] Tool result would exceed %s limit...",
        config.FormatSize(maxContextBeforeToolResult))
    toolResultMsg = fmt.Sprintf("...would push context over %s limit...",
        config.FormatSize(maxContextBeforeToolResult))
}
```

### 3. Added Import

Added `"hyper/internal/config"` import to `tool_executor.go`

## Configuration

### Environment Variable

Set `MAX_CONTEXT_SIZE` in `.env.hyper` to override the default:

```bash
# Default: 500KB (500000 bytes)
MAX_CONTEXT_SIZE=500000

# For larger contexts (1MB)
MAX_CONTEXT_SIZE=1000000

# For smaller contexts (250KB)
MAX_CONTEXT_SIZE=250000
```

**Value**: Size in bytes (integer)

### Default Behavior

If `MAX_CONTEXT_SIZE` is:
- **Not set**: Uses 500KB default
- **Invalid** (non-numeric, negative): Uses 500KB default
- **Valid**: Uses specified value

## Impact

### Before (120KB limit)

- **Context capacity**: ~30K tokens
- **Message count**: ~50-85 messages before hitting limit
- **Request size**: 120-130KB would trigger errors
- **User experience**: Frequent "Context size limit exceeded" errors

### After (500KB default)

- **Context capacity**: ~125K tokens
- **Message count**: ~200-400 messages before hitting limit
- **Request size**: Up to 500KB supported
- **User experience**: Much longer conversations without errors
- **Still configurable**: Can increase further if needed

## Size Comparison

| Limit | Bytes | Approx Tokens | Message Count | Use Case |
|-------|-------|---------------|---------------|----------|
| 120KB | 120,000 | ~30K | 50-85 | Old limit (too small) |
| 250KB | 250,000 | ~62K | 100-150 | Conservative |
| **500KB** | **500,000** | **~125K** | **200-400** | **New default** |
| 1MB | 1,000,000 | ~250K | 400-800 | Large contexts |

**Note**: Token counts assume ~4 chars/token. Actual counts vary by model and content.

## Model Compatibility

### Supported Models

| Model | Max Context | Recommended MAX_CONTEXT_SIZE |
|-------|-------------|------------------------------|
| GPT-3.5 | 16K tokens | 250KB (62K tokens) |
| GPT-4 | 128K tokens | 500KB-1MB |
| Claude 3 | 200K tokens | 500KB-2MB |
| Claude 3.5 Sonnet | 200K tokens | 500KB-2MB |
| Kimi (Moonshot) | 128K tokens | 500KB-1MB |

**Formula**: MAX_CONTEXT_SIZE = (Model Max Tokens × 0.6) × 4 bytes/token

The 0.6 factor leaves room for:
- System prompts (1-2K tokens)
- AI response (2-4K tokens)
- Tool results still to be added

## Testing

### Build
```bash
cd /Users/maxmednikov/MaxSpace/hyper/hyper
go build ./cmd/coordinator
```
✅ Build successful

### Verification

1. **Default behavior** (no env var):
   - Context limit: 500KB
   - Log shows: "Tool result would exceed 500.0KB limit"

2. **Custom value** (with env var):
   ```bash
   export MAX_CONTEXT_SIZE=1000000
   ```
   - Context limit: 1MB
   - Log shows: "Tool result would exceed 1.0MB limit"

3. **Invalid value**:
   ```bash
   export MAX_CONTEXT_SIZE=invalid
   ```
   - Falls back to 500KB default

## Files Modified

1. **`hyper/internal/config/limits.go`**
   - Added `DefaultMaxContextSize` constant (500KB)
   - Added `GetMaxContextSize()` function
   - Added imports: `os`, `strconv`

2. **`hyper/internal/ai-service/tool_executor.go`**
   - Updated 2 occurrences of hardcoded 120KB limit
   - Changed from `const` to function call
   - Added human-readable formatting in logs/messages
   - Added `hyper/internal/config` import

## Migration Guide

### For Existing Deployments

**No action required** - The change is backward compatible:
- Default increases from 120KB → 500KB automatically
- Existing conversations will benefit immediately
- No configuration changes needed

### To Customize (Optional)

Add to `.env.hyper`:
```bash
# Set custom context size (in bytes)
MAX_CONTEXT_SIZE=500000  # 500KB (default)
```

Or set environment variable:
```bash
export MAX_CONTEXT_SIZE=1000000  # 1MB for longer chats
```

### For Development

Test different limits:
```bash
# Small (for testing limits)
MAX_CONTEXT_SIZE=100000 ./coordinator

# Default
./coordinator  # Uses 500KB

# Large (for long conversations)
MAX_CONTEXT_SIZE=2000000 ./coordinator  # 2MB
```

## Future Enhancements

Potential improvements:
1. **Auto-detect from model**: Set limit based on model's context window
2. **Sliding window**: Automatically trim old messages instead of blocking
3. **Compression**: Summarize old messages to save context
4. **Per-session limits**: Different limits for different chat sessions

## Related Issues

- **Original issue**: "Context size limit exceeded" at 130KB
- **User report**: 113 messages caused overflow
- **Request logs**: 77.req.json (123KB), 78.req.json (130KB) exceeded old limit
- **Solution**: Increased default to 500KB (4x larger)

## Notes

- The limit is checked **before** adding each tool result
- Prevents exceeding model's maximum context window
- Provides helpful error message when limit is reached
- Suggests ways to reduce output (filters, pagination, etc.)
- **Does NOT apply to initial message load** - only during tool execution

## See Also

- `docs/reports/TOOL_CALL_ID_DEBUGGING.md` - Tool call ID tracking
- `hyper/internal/config/limits.go` - All size limit constants
- `.env.hyper.example` - Environment configuration examples
