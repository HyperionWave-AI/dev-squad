package handlers

import (
	"fmt"

	"go.uber.org/zap"
	"hyper/internal/config"
	"hyper/internal/metrics"
)

const (
	// UniversalMaxResultBytes is a hard limit that works for ANY AI provider.
	// 50KB is approximately 12-15k tokens, safe for all context windows.
	// This provides a provider-agnostic safeguard before token-based checks.
	UniversalMaxResultBytes = 50 * 1024
)

// DeflectionResult contains the result of interception
type DeflectionResult struct {
	WasDeflected bool   // Whether the result was deflected
	Message      string // Deflection message with suggestions
	OriginalSize int    // Original token count (or bytes for byte-based deflection)
	MaxAllowed   int    // Maximum allowed tokens (or bytes for byte-based deflection)
}

// ToolResultInterceptor checks tool results and deflects oversized ones
type ToolResultInterceptor struct {
	estimator *TokenEstimator
	limits    *ToolResultLimits
	logger    *zap.Logger
}

// NewToolResultInterceptor creates a new interceptor with default limits
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

// CheckResult evaluates if a tool result should be deflected.
// It performs two checks in order:
// 1. Universal byte-based check (provider-agnostic, hard limit)
// 2. Token-based check (provider-specific, configurable)
//
// Returns: (processedResult, deflectionInfo)
// If deflected, processedResult will be nil and deflectionInfo.WasDeflected will be true
func (i *ToolResultInterceptor) CheckResult(
	toolName string,
	result interface{},
	remainingContext int,
) (interface{}, *DeflectionResult) {

	// Convert result to string for size check
	resultStr := fmt.Sprintf("%v", result)
	resultBytes := len(resultStr)

	// UNIVERSAL BYTE CHECK - works for ANY provider
	// This is the first line of defense before token-based checks
	if resultBytes > UniversalMaxResultBytes {
		i.logger.Info("🛑 Tool result deflected - exceeds universal byte limit",
			zap.String("tool", toolName),
			zap.Int("resultBytes", resultBytes),
			zap.Int("maxBytes", UniversalMaxResultBytes))

		// Record metrics for deflected result
		metrics.RecordToolResultDeflection(toolName)

		// Return summarized result with byte-based message
		return nil, &DeflectionResult{
			WasDeflected: true,
			Message:      i.buildByteLimitMessage(toolName, resultBytes, UniversalMaxResultBytes),
			OriginalSize: resultBytes / 4, // Rough token estimate (4 chars per token)
			MaxAllowed:   UniversalMaxResultBytes / 4,
		}
	}

	// Estimate tokens for token-based check
	tokens := i.estimator.EstimateTokens(result)

	// Skip deflection for small results
	if tokens < i.limits.MinTokenThreshold {
		return result, &DeflectionResult{WasDeflected: false}
	}

	// Get limit for this tool
	maxAllowed := i.limits.GetLimit(toolName, remainingContext)

	// Check if deflection needed based on tokens
	if tokens > maxAllowed {
		message := i.buildDeflectionMessage(toolName, tokens, maxAllowed, remainingContext)

		i.logger.Info("🛑 Tool result deflected - exceeds token limit",
			zap.String("tool", toolName),
			zap.Int("resultTokens", tokens),
			zap.Int("maxAllowed", maxAllowed),
			zap.Int("remainingContext", remainingContext))

		// Record metrics for deflected result
		metrics.RecordToolResultDeflection(toolName)
		metrics.RecordToolResultTokens(toolName, tokens, true)

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

	// Record metrics for accepted result
	metrics.RecordToolResultTokens(toolName, tokens, false)

	return result, &DeflectionResult{WasDeflected: false}
}

// buildByteLimitMessage creates a helpful message when the universal byte limit is exceeded.
// This provides provider-agnostic guidance since byte limits work across all AI providers.
func (i *ToolResultInterceptor) buildByteLimitMessage(toolName string, resultBytes, maxBytes int) string {
	return fmt.Sprintf(
		"🛑 **Result Too Large** (%s, limit: %s)\n\n"+
			"The result from `%s` exceeds the universal size limit.\n"+
			"This limit ensures compatibility with all AI providers (Claude, GPT-4, Llama, etc.).\n\n"+
			"%s",
		config.FormatSize(resultBytes),
		config.FormatSize(maxBytes),
		toolName,
		i.getToolSpecificSuggestions(toolName),
	)
}

// buildDeflectionMessage creates a helpful deflection message with tool-specific suggestions
func (i *ToolResultInterceptor) buildDeflectionMessage(
	toolName string,
	resultTokens int,
	maxAllowed int,
	remainingContext int,
) string {

	percentOver := ((resultTokens - maxAllowed) * 100) / maxAllowed
	contextPercent := 0
	if remainingContext > 0 {
		contextPercent = (resultTokens * 100) / remainingContext
	}

	baseMessage := fmt.Sprintf(
		"🛑 **Result Too Large** (%d tokens, %d%% over limit)\n\n"+
			"The result from `%s` is too large (%d tokens) and would consume %d%% of remaining context.\n"+
			"Maximum allowed: %d tokens\n\n",
		resultTokens, percentOver, toolName, resultTokens, contextPercent, maxAllowed)

	// Add tool-specific suggestions
	suggestions := i.getToolSpecificSuggestions(toolName)

	return baseMessage + suggestions
}

// getToolSpecificSuggestions returns tool-specific guidance for more efficient queries
func (i *ToolResultInterceptor) getToolSpecificSuggestions(toolName string) string {
	suggestions := map[string]string{
		"code_index_search": `**Try these parameters to reduce result size:**

1. **Use summary mode** (recommended):
   - Returns ~100 tokens per result instead of ~500-2000
   - Example: { "responseMode": "summary", "limit": 10 }

2. **Reduce limit**:
   - Example: { "limit": 5 }

3. **Add filters** to narrow results:
   - By function: { "functionName": "handleAuth.*", "nodeType": "function" }
   - By class: { "className": "UserService" }
   - By folder: { "folderPath": "./handlers" }

4. **Use preview mode** for moderate detail:
   - Returns summary + first 20 lines
   - Example: { "responseMode": "preview" }

5. **Be more specific** in your query:
   - ❌ Bad: "search for user management"
   - ✅ Good: "search for UserService authentication handler"`,

		"code_index_get_full_content": `**Try specifying a smaller line range:**

1. **Add line range** (most effective):
   - Example: { "filePath": "handlers/user.go", "startLine": 50, "endLine": 100 }
   - This reads only 50 lines instead of the entire file

2. **Search first** to find exact lines:
   - Use code_index_search with summary mode to locate the function
   - Then use code_index_get_full_content with specific line range

3. **Check file size** before requesting full content:
   - Large files (>1000 lines) should always use line ranges
   - Example: { "filePath": "...", "startLine": 1, "endLine": 50 }`,

		"read_file": `**Try these approaches to read less content:**

1. **Use line range parameters** (most effective):
   - Example: { "path": "file.go", "offset": 100, "limit": 50 }
   - Reads only lines 100-150 instead of entire file

2. **Use code_index_search first** to find relevant sections:
   - Use summary mode to locate the exact lines
   - Then read only those specific lines

3. **Use code_index_get_full_content** with line range:
   - Example: { "filePath": "file.go", "startLine": 50, "endLine": 100 }

4. **Check file size** before reading:
   - Large files should always use line ranges
   - ❌ Bad: Read entire 10MB log file
   - ✅ Good: Read lines 1-100 of the file`,

		"bash": `**Try limiting command output:**

1. **Pipe through head** (most effective):
   - Example: command | head -100
   - Shows only first 100 lines

2. **Pipe through tail** for recent output:
   - Example: command | tail -50
   - Shows only last 50 lines

3. **Add grep filter**:
   - Example: find . -type f | grep "pattern"
   - Filters results to matching items

4. **Use quiet/summary flags** if available:
   - Example: command --quiet or command --summary

5. **Count instead of showing all**:
   - Example: find . -type f | wc -l
   - Shows count instead of listing all files`,

		"knowledge_find": `**Try limiting knowledge search:**

1. **Reduce limit** (most effective):
   - Example: { "limit": 3 }
   - Returns only top 3 results instead of 10

2. **Be more specific** in your query:
   - ❌ Bad: "find knowledge about error handling"
   - ✅ Good: "find error handling patterns for HTTP requests"

3. **Use exact phrases**:
   - Example: { "query": "JWT authentication validation" }
   - More specific than broad concepts

4. **Filter by collection** if available:
   - Example: { "collection": "code-patterns", "query": "..." }`,

		"coordinator_list_agent_tasks": `**Try filtering task lists:**

1. **Reduce limit** (most effective):
   - Example: { "limit": 5 }
   - Returns only 5 tasks instead of 20

2. **Filter by status**:
   - Example: { "status": "pending" }
   - Only list pending tasks, not completed ones

3. **Filter by agent**:
   - Example: { "agentName": "Developer" }
   - Only list tasks for specific agent

4. **Combine filters**:
   - Example: { "status": "pending", "agentName": "Developer", "limit": 5 }`,

		"grep": `**Try narrowing grep results:**

1. **Reduce output with head**:
   - Example: grep "pattern" file | head -20
   - Shows only first 20 matches

2. **Use files_with_matches mode** (just file names):
   - Example: grep -l "pattern" *.go
   - Shows only filenames, not content

3. **Add more specific pattern**:
   - Example: grep "exactFunctionName" *.go
   - More specific than broad patterns

4. **Limit files searched**:
   - Example: grep "pattern" --include="*.go" .
   - Only search Go files`,

		"file_read": `**Try these approaches to read less content:**

1. **Use line range parameters** (most effective):
   - Example: { "path": "file.go", "offset": 100, "limit": 50 }
   - Reads only lines 100-150 instead of entire file

2. **Use code_index_search first** to find relevant sections:
   - Use summary mode to locate the exact lines
   - Then read only those specific lines

3. **Check file size** before reading:
   - Large files should always use line ranges`,

		"coordinator_list_human_tasks": `**Try filtering task lists:**

1. **Reduce limit** (most effective):
   - Example: { "limit": 5 }
   - Returns only 5 tasks instead of 20

2. **Filter by status**:
   - Example: { "status": "pending" }
   - Only list pending tasks

3. **Filter by assignee**:
   - Example: { "assignedTo": "user_id" }
   - Only list tasks for specific person`,
	}

	if suggestion, exists := suggestions[toolName]; exists {
		return suggestion
	}

	// Default suggestion for unknown tools
	return `**General suggestions:**

1. **Narrow your query**: Be more specific about what you're looking for
2. **Use filters**: Apply available filters to reduce results
3. **Limit results**: Request fewer results (use limit parameter)
4. **Try a different approach**: Consider using a different tool that might return less data`
}
