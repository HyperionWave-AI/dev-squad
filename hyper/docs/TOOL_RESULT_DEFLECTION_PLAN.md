# Tool Result Deflection - Implementation Plan

## Overview

When a tool result exceeds a target context threshold, return a helpful message instead of the full result, guiding the AI to use different parameters for more efficient token usage.

### Problem Statement

Large tool results consume excessive context tokens, leading to:
- Context window exhaustion
- Increased latency and costs
- Degraded AI performance (too much noise)

### Solution

Intercept tool results before adding to context. If result exceeds threshold:
1. Discard the large result
2. Return a helpful deflection message
3. Include tool-specific suggestions for getting smaller results

---

## Phase 1: Token Estimation Infrastructure

**Goal:** Add ability to estimate token count of tool results

**Files to create/modify:**
- `internal/handlers/token_estimator.go` (new file)

**Implementation:**

```go
package handlers

import (
    "encoding/json"
)

// TokenEstimator estimates tokens for content
// Uses approximation: ~4 characters per token for code/JSON
type TokenEstimator struct {
    charsPerToken int
}

// NewTokenEstimator creates a new token estimator
func NewTokenEstimator() *TokenEstimator {
    return &TokenEstimator{
        charsPerToken: 4, // Conservative estimate for code
    }
}

// EstimateTokens estimates token count for a string
func (e *TokenEstimator) EstimateTokens(content string) int {
    if content == "" {
        return 0
    }
    return len(content) / e.charsPerToken
}

// EstimateToolResultTokens estimates tokens for any tool result
func (e *TokenEstimator) EstimateToolResultTokens(result interface{}) int {
    if result == nil {
        return 0
    }

    // Handle string results directly
    if str, ok := result.(string); ok {
        return e.EstimateTokens(str)
    }

    // JSON serialize complex results
    jsonBytes, err := json.Marshal(result)
    if err != nil {
        return 0
    }
    return len(jsonBytes) / e.charsPerToken
}
```

**Effort:** ~30 minutes

**Testing:**
```go
func TestTokenEstimator(t *testing.T) {
    e := NewTokenEstimator()

    // ~1000 chars = ~250 tokens
    content := strings.Repeat("a", 1000)
    tokens := e.EstimateTokens(content)
    assert.Equal(t, 250, tokens)
}
```

---

## Phase 2: Threshold Configuration

**Goal:** Define configurable thresholds for different tools

**Files to create/modify:**
- `internal/handlers/tool_result_limits.go` (new file)

**Implementation:**

```go
package handlers

// ToolResultLimits defines token limits for tool results
type ToolResultLimits struct {
    // DefaultMaxTokens is the default limit for any tool
    DefaultMaxTokens int

    // ToolSpecificLimits overrides for specific tools
    ToolSpecificLimits map[string]int

    // ContextPercentLimit is max percentage of remaining context a single result can use
    // e.g., 0.25 = max 25% of remaining context
    ContextPercentLimit float64

    // MinTokenThreshold is the minimum tokens before deflection kicks in
    // Results smaller than this are always allowed
    MinTokenThreshold int
}

// DefaultToolResultLimits returns sensible defaults
func DefaultToolResultLimits() *ToolResultLimits {
    return &ToolResultLimits{
        DefaultMaxTokens: 5000,
        ToolSpecificLimits: map[string]int{
            // Code search tools - prefer summaries
            "code_index_search":           3000,
            "code_index_get_full_content": 6000,

            // File operations
            "read_file":  8000,
            "file_read":  8000,

            // Shell commands can be verbose
            "bash":       4000,

            // Knowledge/search tools
            "knowledge_find":    2000,
            "coordinator_list_agent_tasks": 2000,

            // Grep results
            "grep": 3000,
        },
        ContextPercentLimit: 0.25, // Max 25% of remaining context
        MinTokenThreshold:   500,  // Don't deflect tiny results
    }
}

// GetLimit returns the token limit for a specific tool
func (l *ToolResultLimits) GetLimit(toolName string, remainingContext int) int {
    // Check tool-specific limit first
    if limit, exists := l.ToolSpecificLimits[toolName]; exists {
        return min(limit, l.calculateContextLimit(remainingContext))
    }

    // Fall back to default
    return min(l.DefaultMaxTokens, l.calculateContextLimit(remainingContext))
}

// calculateContextLimit returns max tokens based on remaining context
func (l *ToolResultLimits) calculateContextLimit(remainingContext int) int {
    if remainingContext <= 0 {
        return l.DefaultMaxTokens
    }
    return int(float64(remainingContext) * l.ContextPercentLimit)
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

**Effort:** ~20 minutes

---

## Phase 3: Result Interceptor

**Goal:** Intercept tool results before adding to message history

**Files to create/modify:**
- `internal/handlers/tool_result_interceptor.go` (new file)

**Implementation:**

```go
package handlers

import (
    "go.uber.org/zap"
)

// DeflectionResult contains the result of interception
type DeflectionResult struct {
    WasDeflected bool
    Message      string
    OriginalSize int
    MaxAllowed   int
}

// ToolResultInterceptor checks tool results and deflects oversized ones
type ToolResultInterceptor struct {
    estimator *TokenEstimator
    limits    *ToolResultLimits
    logger    *zap.Logger
}

// NewToolResultInterceptor creates a new interceptor
func NewToolResultInterceptor(logger *zap.Logger) *ToolResultInterceptor {
    return &ToolResultInterceptor{
        estimator: NewTokenEstimator(),
        limits:    DefaultToolResultLimits(),
        logger:    logger,
    }
}

// NewToolResultInterceptorWithLimits creates an interceptor with custom limits
func NewToolResultInterceptorWithLimits(limits *ToolResultLimits, logger *zap.Logger) *ToolResultInterceptor {
    return &ToolResultInterceptor{
        estimator: NewTokenEstimator(),
        limits:    limits,
        logger:    logger,
    }
}

// CheckResult evaluates if a tool result should be deflected
// Returns: (processedResult, deflectionInfo)
func (i *ToolResultInterceptor) CheckResult(
    toolName string,
    result interface{},
    remainingContext int,
) (interface{}, *DeflectionResult) {

    // Estimate tokens
    tokens := i.estimator.EstimateToolResultTokens(result)

    // Skip deflection for small results
    if tokens < i.limits.MinTokenThreshold {
        return result, &DeflectionResult{WasDeflected: false}
    }

    // Get limit for this tool
    maxAllowed := i.limits.GetLimit(toolName, remainingContext)

    // Check if deflection needed
    if tokens > maxAllowed {
        message := i.buildDeflectionMessage(toolName, tokens, maxAllowed)

        i.logger.Info("🛑 Tool result deflected - exceeds token limit",
            zap.String("tool", toolName),
            zap.Int("resultTokens", tokens),
            zap.Int("maxAllowed", maxAllowed),
            zap.Int("remainingContext", remainingContext))

        return nil, &DeflectionResult{
            WasDeflected: true,
            Message:      message,
            OriginalSize: tokens,
            MaxAllowed:   maxAllowed,
        }
    }

    // Result is within limits
    i.logger.Debug("Tool result within limits",
        zap.String("tool", toolName),
        zap.Int("resultTokens", tokens),
        zap.Int("maxAllowed", maxAllowed))

    return result, &DeflectionResult{WasDeflected: false}
}
```

**Effort:** ~1 hour

---

## Phase 4: Deflection Messages

**Goal:** Create helpful, tool-specific deflection messages with actionable suggestions

**Files to modify:**
- `internal/handlers/tool_result_interceptor.go` (add methods)

**Implementation:**

```go
// buildDeflectionMessage creates a helpful message for deflected results
func (i *ToolResultInterceptor) buildDeflectionMessage(
    toolName string,
    actualTokens int,
    maxAllowed int,
) string {

    header := fmt.Sprintf(
        "⚠️ **Result too large** (%d tokens, limit: %d)\n\n",
        actualTokens, maxAllowed,
    )

    suggestions := i.getSuggestionsForTool(toolName)

    return header + suggestions
}

// getSuggestionsForTool returns tool-specific suggestions for reducing result size
func (i *ToolResultInterceptor) getSuggestionsForTool(toolName string) string {

    switch toolName {

    case "code_index_search":
        return `**Try these parameters to reduce result size:**

1. **Use summary mode** (recommended):
   \`\`\`json
   { "responseMode": "summary", "limit": 10 }
   \`\`\`
   Returns ~100 tokens per result instead of ~500-2000

2. **Reduce limit**:
   \`\`\`json
   { "limit": 5 }
   \`\`\`

3. **Add filters** to narrow results:
   \`\`\`json
   { "functionName": "handleAuth.*", "nodeType": "function" }
   \`\`\`

4. **Use preview mode** for moderate detail:
   \`\`\`json
   { "responseMode": "preview" }
   \`\`\`
   Returns summary + first 20 lines`

    case "read_file", "file_read":
        return `**Try these approaches to read less content:**

1. **Use line range parameters**:
   \`\`\`json
   { "offset": 100, "limit": 50 }
   \`\`\`
   Reads only lines 100-150

2. **Use code_index_search first** to find relevant sections:
   \`\`\`json
   code_index_search({ "query": "function name", "responseMode": "summary" })
   \`\`\`
   Then read only the specific lines you need

3. **Use code_index_get_full_content** with line range:
   \`\`\`json
   { "filePath": "...", "startLine": 50, "endLine": 100 }
   \`\`\``

    case "code_index_get_full_content":
        return `**Try specifying a smaller line range:**

1. **Add line range**:
   \`\`\`json
   { "filePath": "...", "startLine": 1, "endLine": 50 }
   \`\`\`

2. **Search first** to find exact lines:
   \`\`\`json
   code_index_search({ "query": "...", "responseMode": "summary" })
   \`\`\`
   Use startLine/endLine from results`

    case "bash":
        return `**Try limiting command output:**

1. **Pipe through head**:
   \`\`\`bash
   command | head -100
   \`\`\`

2. **Pipe through tail** for recent output:
   \`\`\`bash
   command | tail -50
   \`\`\`

3. **Add grep filter**:
   \`\`\`bash
   command | grep "pattern"
   \`\`\`

4. **Use quiet/summary flags** if available:
   \`\`\`bash
   command --quiet
   command --summary
   \`\`\``

    case "grep", "Grep":
        return `**Try narrowing grep results:**

1. **Reduce output with head_limit**:
   \`\`\`json
   { "head_limit": 20 }
   \`\`\`

2. **Use files_with_matches mode** (just file names):
   \`\`\`json
   { "output_mode": "files_with_matches" }
   \`\`\`

3. **Add more specific pattern**:
   \`\`\`json
   { "pattern": "exactFunctionName", "type": "go" }
   \`\`\``

    case "knowledge_find":
        return `**Try limiting knowledge search:**

1. **Reduce limit**:
   \`\`\`json
   { "limit": 5 }
   \`\`\`

2. **Be more specific** in your query`

    case "coordinator_list_agent_tasks", "coordinator_list_human_tasks":
        return `**Try filtering task lists:**

1. **Reduce limit**:
   \`\`\`json
   { "limit": 5 }
   \`\`\`

2. **Filter by status**:
   \`\`\`json
   { "status": "pending" }
   \`\`\``

    default:
        return `**General suggestions:**

1. Use more specific parameters
2. Add filters to narrow results
3. Request smaller page sizes (limit parameter)
4. Try a more targeted query`
    }
}
```

**Effort:** ~45 minutes

---

## Phase 5: Integration Points

**Goal:** Wire interceptor into the tool execution flow

**Files to modify:**
- `internal/ai-service/tool_executor.go`

**Implementation:**

```go
// Add to ToolExecutorService struct
type ToolExecutorService struct {
    // ... existing fields ...
    resultInterceptor *handlers.ToolResultInterceptor
}

// Update constructor
func NewToolExecutorService(...) *ToolExecutorService {
    return &ToolExecutorService{
        // ... existing fields ...
        resultInterceptor: handlers.NewToolResultInterceptor(logger),
    }
}

// Add method to execute with deflection
func (s *ToolExecutorService) executeToolWithDeflection(
    ctx context.Context,
    toolCall ToolCall,
    remainingContext int,
) (string, error) {

    // Execute the tool normally
    result, err := s.executeTool(ctx, toolCall)
    if err != nil {
        return "", err
    }

    // Check if result needs deflection
    processedResult, deflection := s.resultInterceptor.CheckResult(
        toolCall.Name,
        result,
        remainingContext,
    )

    if deflection.WasDeflected {
        // Log metric
        s.metrics.ToolResultDeflections.WithLabelValues(toolCall.Name).Inc()

        // Return deflection message instead of result
        return deflection.Message, nil
    }

    // Return original result
    return s.formatResult(processedResult), nil
}

// Update the main execution loop to use deflection
func (s *ToolExecutorService) ExecuteToolsWithStreaming(...) {
    // ... existing code ...

    // Calculate remaining context
    remainingContext := s.calculateRemainingContext(currentMessages)

    // Execute with deflection check
    for _, toolCall := range toolCalls {
        result, err := s.executeToolWithDeflection(ctx, toolCall, remainingContext)
        // ... handle result ...
    }
}

// Helper to calculate remaining context
func (s *ToolExecutorService) calculateRemainingContext(messages []Message) int {
    totalContextWindow := 100000 // or from config

    usedTokens := 0
    for _, msg := range messages {
        usedTokens += s.resultInterceptor.estimator.EstimateTokens(msg.Content)
    }

    return totalContextWindow - usedTokens
}
```

**Effort:** ~1 hour

---

## Phase 6: Logging & Metrics

**Goal:** Track deflection events for observability and tuning

**Files to modify:**
- `internal/metrics/recorder.go`

**Implementation:**

```go
// Add new metrics
var (
    ToolResultDeflections = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "hyperion_tool_result_deflections_total",
            Help: "Total number of tool results deflected due to size",
        },
        []string{"tool_name"},
    )

    ToolResultTokens = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "hyperion_tool_result_tokens",
            Help:    "Token count of tool results",
            Buckets: []float64{100, 500, 1000, 2000, 5000, 10000, 20000},
        },
        []string{"tool_name", "deflected"},
    )
)

// Usage in interceptor
func (i *ToolResultInterceptor) CheckResult(...) {
    // ... existing code ...

    // Record metrics
    deflectedLabel := "false"
    if tokens > maxAllowed {
        deflectedLabel = "true"
        metrics.ToolResultDeflections.WithLabelValues(toolName).Inc()
    }
    metrics.ToolResultTokens.WithLabelValues(toolName, deflectedLabel).Observe(float64(tokens))
}
```

**Effort:** ~20 minutes

---

## Summary

| Phase | Description | Effort | Files |
|-------|-------------|--------|-------|
| 1 | Token Estimation | 30 min | `token_estimator.go` |
| 2 | Threshold Config | 20 min | `tool_result_limits.go` |
| 3 | Result Interceptor | 1 hour | `tool_result_interceptor.go` |
| 4 | Deflection Messages | 45 min | `tool_result_interceptor.go` |
| 5 | Integration | 1 hour | `tool_executor.go` |
| 6 | Logging/Metrics | 20 min | `metrics/recorder.go` |

**Total Estimated Effort: ~4 hours**

---

## Example Flow

```
┌─────────────────────────────────────────────────────────────────┐
│ User: "Show me all the authentication code"                     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ AI calls: code_index_search({ query: "authentication",          │
│                               limit: 50 })                      │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ Tool Executor: Executes search, gets result                     │
│ Result size: 15,000 tokens                                      │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ Interceptor Check:                                              │
│   - Result: 15,000 tokens                                       │
│   - Max allowed: 3,000 tokens (code_index_search limit)         │
│   - Decision: DEFLECT ❌                                         │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ Deflection Message returned to AI:                              │
│                                                                 │
│ ⚠️ **Result too large** (15000 tokens, limit: 3000)             │
│                                                                 │
│ **Try these parameters to reduce result size:**                 │
│                                                                 │
│ 1. **Use summary mode** (recommended):                          │
│    { "responseMode": "summary", "limit": 10 }                   │
│                                                                 │
│ 2. **Reduce limit**:                                            │
│    { "limit": 5 }                                               │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ AI retries with better params:                                  │
│ code_index_search({                                             │
│     query: "authentication",                                    │
│     limit: 10,                                                  │
│     responseMode: "summary"                                     │
│ })                                                              │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ Interceptor Check:                                              │
│   - Result: 800 tokens                                          │
│   - Max allowed: 3,000 tokens                                   │
│   - Decision: ALLOW ✅                                           │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ Result passed to AI context                                     │
│ AI can now work with manageable result                          │
└─────────────────────────────────────────────────────────────────┘
```

---

## Future Enhancements

1. **Smart Truncation**: Instead of full deflection, truncate results intelligently
2. **Streaming Results**: For very large results, stream in chunks
3. **Adaptive Limits**: Adjust limits based on conversation length
4. **Result Caching**: Cache deflected results for later retrieval
5. **User Preferences**: Allow users to configure their own limits

---

## Related Documents

- `docs/CONTEXT_OPTIMIZATION.md` - Context window management
- `docs/CODE_INDEX_SEARCH.md` - Code search tool documentation
- `CLAUDE.md` - Coordinator system prompt with tool guidance
