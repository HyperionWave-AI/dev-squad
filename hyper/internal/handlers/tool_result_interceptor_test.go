package handlers

import (
	"strings"
	"testing"

	"go.uber.org/zap"
)

// TestNewToolResultInterceptor tests the default constructor
func TestNewToolResultInterceptor(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewToolResultInterceptor(logger)

	if interceptor == nil {
		t.Error("NewToolResultInterceptor returned nil")
	}
	if interceptor.estimator == nil {
		t.Error("estimator is nil")
	}
	if interceptor.limits == nil {
		t.Error("limits is nil")
	}
	if interceptor.logger == nil {
		t.Error("logger is nil")
	}
}

// TestNewToolResultInterceptorWithLimits tests the custom limits constructor
func TestNewToolResultInterceptorWithLimits(t *testing.T) {
	logger := zap.NewNop()
	customLimits := &ToolResultLimits{
		DefaultMaxTokens: 2000,
		ToolSpecificLimits: map[string]int{
			"test_tool": 1000,
		},
		ContextPercentLimit: 0.15,
		MinTokenThreshold:   100,
	}

	interceptor := NewToolResultInterceptorWithLimits(customLimits, logger)

	if interceptor == nil {
		t.Error("NewToolResultInterceptorWithLimits returned nil")
	}
	if interceptor.limits != customLimits {
		t.Error("limits not set correctly")
	}
}

// TestCheckResult_SmallResult tests that small results are not deflected
func TestCheckResult_SmallResult(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewToolResultInterceptor(logger)

	result := "small result"
	processedResult, deflection := interceptor.CheckResult("test_tool", result, 5000)

	if deflection.WasDeflected {
		t.Error("Small result should not be deflected")
	}
	if processedResult != result {
		t.Error("Processed result should be the original result")
	}
}

// TestCheckResult_LargeResult tests that large results are deflected
func TestCheckResult_LargeResult(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewToolResultInterceptor(logger)

	// Create a large result (> 5000 tokens)
	largeResult := strings.Repeat("x", 25000)

	processedResult, deflection := interceptor.CheckResult("code_index_search", largeResult, 10000)

	if !deflection.WasDeflected {
		t.Error("Large result should be deflected")
	}
	if processedResult != nil {
		t.Error("Processed result should be nil when deflected")
	}
	if deflection.Message == "" {
		t.Error("Deflection message should not be empty")
	}
	if deflection.OriginalSize == 0 {
		t.Error("Original size should be set")
	}
	if deflection.MaxAllowed == 0 {
		t.Error("Max allowed should be set")
	}
}

// TestCheckResult_BelowMinThreshold tests that results below min threshold are not deflected
func TestCheckResult_BelowMinThreshold(t *testing.T) {
	logger := zap.NewNop()
	limits := &ToolResultLimits{
		DefaultMaxTokens:    5000,
		ToolSpecificLimits:  map[string]int{},
		ContextPercentLimit: 0.25,
		MinTokenThreshold:   1000, // High threshold
	}
	interceptor := NewToolResultInterceptorWithLimits(limits, logger)

	// Create a result that's 500 tokens (below 1000 threshold)
	result := strings.Repeat("x", 2000) // ~500 tokens

	processedResult, deflection := interceptor.CheckResult("test_tool", result, 10000)

	if deflection.WasDeflected {
		t.Error("Result below min threshold should not be deflected")
	}
	if processedResult != result {
		t.Error("Processed result should be the original result")
	}
}

// TestCheckResult_WithinLimit tests that results within limit are not deflected
func TestCheckResult_WithinLimit(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewToolResultInterceptor(logger)

	// Create a result that's within limits
	result := strings.Repeat("x", 4000) // ~1000 tokens

	processedResult, deflection := interceptor.CheckResult("test_tool", result, 10000)

	if deflection.WasDeflected {
		t.Error("Result within limit should not be deflected")
	}
	if processedResult != result {
		t.Error("Processed result should be the original result")
	}
}

// TestCheckResult_ContextPercentLimit tests that context percent limit is respected
func TestCheckResult_ContextPercentLimit(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewToolResultInterceptor(logger)

	// With 1000 remaining context, 25% = 250 tokens
	// Create a result that's 500 tokens (> 250)
	result := strings.Repeat("x", 2000) // ~500 tokens

	processedResult, deflection := interceptor.CheckResult("test_tool", result, 1000)

	if !deflection.WasDeflected {
		t.Error("Result exceeding context percent limit should be deflected")
	}
	if processedResult != nil {
		t.Error("Processed result should be nil when deflected")
	}
}

// TestCheckResult_ToolSpecificLimit tests that tool-specific limits are applied
func TestCheckResult_ToolSpecificLimit(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewToolResultInterceptor(logger)

	// code_index_search has a specific limit of 3000
	// Create a result that's 4000 tokens
	result := strings.Repeat("x", 16000) // ~4000 tokens

	processedResult, deflection := interceptor.CheckResult("code_index_search", result, 100000)

	if !deflection.WasDeflected {
		t.Error("Result exceeding tool-specific limit should be deflected")
	}
	if processedResult != nil {
		t.Error("Processed result should be nil when deflected")
	}
}

// TestBuildDeflectionMessage_CodeIndexSearch tests deflection message for code_index_search
func TestBuildDeflectionMessage_CodeIndexSearch(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewToolResultInterceptor(logger)

	message := interceptor.buildDeflectionMessage("code_index_search", 5000, 3000, 10000)

	if !strings.Contains(message, "code_index_search") {
		t.Error("Message should contain tool name")
	}
	if !strings.Contains(message, "5000") {
		t.Error("Message should contain result tokens")
	}
	if !strings.Contains(message, "3000") {
		t.Error("Message should contain max allowed")
	}
	if !strings.Contains(message, "summary mode") {
		t.Error("Message should contain code_index_search-specific suggestion")
	}
}

// TestBuildDeflectionMessage_ReadFile tests deflection message for read_file
func TestBuildDeflectionMessage_ReadFile(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewToolResultInterceptor(logger)

	message := interceptor.buildDeflectionMessage("read_file", 10000, 8000, 20000)

	if !strings.Contains(message, "read_file") {
		t.Error("Message should contain tool name")
	}
	if !strings.Contains(message, "line range") {
		t.Error("Message should contain read_file-specific suggestion")
	}
}

// TestBuildDeflectionMessage_Bash tests deflection message for bash
func TestBuildDeflectionMessage_Bash(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewToolResultInterceptor(logger)

	message := interceptor.buildDeflectionMessage("bash", 6000, 4000, 15000)

	if !strings.Contains(message, "bash") {
		t.Error("Message should contain tool name")
	}
	if !strings.Contains(message, "head") {
		t.Error("Message should contain bash-specific suggestion")
	}
}

// TestBuildDeflectionMessage_UnknownTool tests deflection message for unknown tool
func TestBuildDeflectionMessage_UnknownTool(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewToolResultInterceptor(logger)

	message := interceptor.buildDeflectionMessage("unknown_tool", 5000, 3000, 10000)

	if !strings.Contains(message, "unknown_tool") {
		t.Error("Message should contain tool name")
	}
	if !strings.Contains(message, "General suggestions") {
		t.Error("Message should contain general suggestions for unknown tool")
	}
}

// TestGetToolSpecificSuggestions_AllTools tests that all known tools have suggestions
func TestGetToolSpecificSuggestions_AllTools(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewToolResultInterceptor(logger)

	tools := []string{
		"code_index_search",
		"code_index_get_full_content",
		"read_file",
		"bash",
		"knowledge_find",
		"coordinator_list_agent_tasks",
		"grep",
	}

	for _, tool := range tools {
		suggestion := interceptor.getToolSpecificSuggestions(tool)
		if suggestion == "" {
			t.Errorf("No suggestion for tool: %s", tool)
		}
		// Check for common patterns in suggestions
		if !strings.Contains(suggestion, "Try") && !strings.Contains(suggestion, "Reduce") {
			t.Errorf("Suggestion for %s should contain actionable advice", tool)
		}
	}
}

// TestDeflectionResult_Structure tests the DeflectionResult struct
func TestDeflectionResult_Structure(t *testing.T) {
	result := &DeflectionResult{
		WasDeflected: true,
		Message:      "Test message",
		OriginalSize: 5000,
		MaxAllowed:   3000,
	}

	if !result.WasDeflected {
		t.Error("WasDeflected should be true")
	}
	if result.Message != "Test message" {
		t.Error("Message not set correctly")
	}
	if result.OriginalSize != 5000 {
		t.Error("OriginalSize not set correctly")
	}
	if result.MaxAllowed != 3000 {
		t.Error("MaxAllowed not set correctly")
	}
}

// TestCheckResult_MultipleTools tests deflection with different tools
func TestCheckResult_MultipleTools(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewToolResultInterceptor(logger)

	tests := []struct {
		toolName         string
		resultSize       int
		remainingContext int
		shouldDeflect    bool
		name             string
	}{
		{"code_index_search", 4000, 20000, true, "code_index_search over limit"},
		{"read_file", 10000, 20000, true, "read_file over limit"},
		{"bash", 5000, 20000, true, "bash over limit"},
		{"knowledge_find", 3000, 20000, true, "knowledge_find over limit"},
		{"unknown_tool", 6000, 20000, true, "unknown_tool over limit"},
		{"test_tool", 1000, 20000, false, "small result"},
	}

	for _, tt := range tests {
		result := strings.Repeat("x", tt.resultSize*4)
		_, deflection := interceptor.CheckResult(tt.toolName, result, tt.remainingContext)

		if deflection.WasDeflected != tt.shouldDeflect {
			t.Errorf("%s: expected deflect=%v, got %v", tt.name, tt.shouldDeflect, deflection.WasDeflected)
		}
	}
}

// TestCheckResult_EdgeCases tests edge cases
func TestCheckResult_EdgeCases(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewToolResultInterceptor(logger)

	tests := []struct {
		result           interface{}
		remainingContext int
		shouldDeflect    bool
		name             string
	}{
		{nil, 5000, false, "nil result"},
		{"", 5000, false, "empty string"},
		{[]string{}, 5000, false, "empty array"},
		{map[string]interface{}{}, 5000, false, "empty map"},
		{0, 5000, false, "zero value"},
	}

	for _, tt := range tests {
		_, deflection := interceptor.CheckResult("test_tool", tt.result, tt.remainingContext)

		if deflection.WasDeflected != tt.shouldDeflect {
			t.Errorf("%s: expected deflect=%v, got %v", tt.name, tt.shouldDeflect, deflection.WasDeflected)
		}
	}
}

// TestCheckResult_PercentageCalculation tests that percentage calculations are correct
func TestCheckResult_PercentageCalculation(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewToolResultInterceptor(logger)

	// Create a result that's 5000 tokens
	result := strings.Repeat("x", 20000)

	_, deflection := interceptor.CheckResult("test_tool", result, 10000)

	if deflection.WasDeflected {
		// Check that message contains percentage information
		if !strings.Contains(deflection.Message, "%") {
			t.Error("Message should contain percentage information")
		}
	}
}

// TestCheckResult_LowRemainingContext tests behavior with very low remaining context
func TestCheckResult_LowRemainingContext(t *testing.T) {
	logger := zap.NewNop()
	// Use custom limits with lower DefaultMaxTokens to test low context scenario
	limits := &ToolResultLimits{
		DefaultMaxTokens:    100,
		ToolSpecificLimits:  map[string]int{},
		ContextPercentLimit: 0.25,
		MinTokenThreshold:   50,
	}
	interceptor := NewToolResultInterceptorWithLimits(limits, logger)

	// With 100 tokens remaining, 25% = 25 tokens, but DefaultMaxTokens = 100
	// So limit = min(100, 25) = 25
	// Create a result that's 100 tokens (> 25)
	result := strings.Repeat("x", 400)

	_, deflection := interceptor.CheckResult("test_tool", result, 100)

	if !deflection.WasDeflected {
		t.Error("Result should be deflected with very low remaining context")
	}
}

// TestCheckResult_ZeroRemainingContext tests behavior with zero remaining context
func TestCheckResult_ZeroRemainingContext(t *testing.T) {
	logger := zap.NewNop()
	// Use custom limits to test zero context scenario
	limits := &ToolResultLimits{
		DefaultMaxTokens:    100,
		ToolSpecificLimits:  map[string]int{},
		ContextPercentLimit: 0.25,
		MinTokenThreshold:   50,
	}
	interceptor := NewToolResultInterceptorWithLimits(limits, logger)

	// With 0 remaining context, limit = DefaultMaxTokens = 100
	// Create a result that's 200 tokens (> 100)
	result := strings.Repeat("x", 800)

	_, deflection := interceptor.CheckResult("test_tool", result, 0)

	if !deflection.WasDeflected {
		t.Error("Result should be deflected with zero remaining context")
	}
}

// TestCheckResult_NegativeRemainingContext tests behavior with negative remaining context
func TestCheckResult_NegativeRemainingContext(t *testing.T) {
	logger := zap.NewNop()
	// Use custom limits to test negative context scenario
	limits := &ToolResultLimits{
		DefaultMaxTokens:    100,
		ToolSpecificLimits:  map[string]int{},
		ContextPercentLimit: 0.25,
		MinTokenThreshold:   50,
	}
	interceptor := NewToolResultInterceptorWithLimits(limits, logger)

	// With negative remaining context, limit = DefaultMaxTokens = 100
	// Create a result that's 200 tokens (> 100)
	result := strings.Repeat("x", 800)

	_, deflection := interceptor.CheckResult("test_tool", result, -1000)

	if !deflection.WasDeflected {
		t.Error("Result should be deflected with negative remaining context")
	}
}

// TestCheckResult_ComplexDataStructures tests with complex data structures
func TestCheckResult_ComplexDataStructures(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewToolResultInterceptor(logger)

	// Test with map
	largeMap := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeMap[strings.Repeat("k", 10)] = strings.Repeat("v", 10)
	}

	_, deflection := interceptor.CheckResult("test_tool", largeMap, 5000)

	if deflection.WasDeflected {
		if deflection.OriginalSize == 0 {
			t.Error("OriginalSize should be set for complex structures")
		}
	}
}

// TestCheckResult_StringArray tests with string arrays
func TestCheckResult_StringArray(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewToolResultInterceptor(logger)

	// Create a large array of strings
	largeArray := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		largeArray[i] = strings.Repeat("x", 20)
	}

	_, deflection := interceptor.CheckResult("test_tool", largeArray, 5000)

	if deflection.WasDeflected {
		if deflection.OriginalSize == 0 {
			t.Error("OriginalSize should be set for arrays")
		}
	}
}

// TestDeflectionMessage_Format tests that deflection messages are well-formatted
func TestDeflectionMessage_Format(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewToolResultInterceptor(logger)

	message := interceptor.buildDeflectionMessage("code_index_search", 5000, 3000, 10000)

	// Check for required components
	if !strings.Contains(message, "🛑") {
		t.Error("Message should contain warning emoji")
	}
	if !strings.Contains(message, "**") {
		t.Error("Message should contain markdown formatting")
	}
	if !strings.Contains(message, "Try") {
		t.Error("Message should contain actionable suggestions")
	}
}

// BenchmarkCheckResult benchmarks the CheckResult method
func BenchmarkCheckResult(b *testing.B) {
	logger := zap.NewNop()
	interceptor := NewToolResultInterceptor(logger)
	result := strings.Repeat("x", 20000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		interceptor.CheckResult("code_index_search", result, 10000)
	}
}

// BenchmarkBuildDeflectionMessage benchmarks the buildDeflectionMessage method
func BenchmarkBuildDeflectionMessage(b *testing.B) {
	logger := zap.NewNop()
	interceptor := NewToolResultInterceptor(logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		interceptor.buildDeflectionMessage("code_index_search", 5000, 3000, 10000)
	}
}
