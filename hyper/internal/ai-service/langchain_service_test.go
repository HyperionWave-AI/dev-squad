package aiservice

import (
	"testing"
	"time"
)

// TestContextTrackerCreation tests that ContextTracker is created correctly
func TestContextTrackerCreation(t *testing.T) {
	ct := NewContextTracker()
	
	if ct == nil {
		t.Fatal("ContextTracker should not be nil")
	}
	
	if ct.InputTokens != 0 {
		t.Errorf("InputTokens should be 0, got %d", ct.InputTokens)
	}
	
	if ct.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}
}

// TestContextTrackerTokenRecording tests token recording methods
func TestContextTrackerTokenRecording(t *testing.T) {
	ct := NewContextTracker()
	
	ct.RecordInputTokens(100)
	ct.RecordOutputTokens(50)
	ct.RecordToolResultTokens(200)
	ct.RecordProcessedTokens(60)
	
	if ct.InputTokens != 100 {
		t.Errorf("InputTokens should be 100, got %d", ct.InputTokens)
	}
	
	if ct.OutputTokens != 50 {
		t.Errorf("OutputTokens should be 50, got %d", ct.OutputTokens)
	}
	
	if ct.ToolResultTokens != 200 {
		t.Errorf("ToolResultTokens should be 200, got %d", ct.ToolResultTokens)
	}
	
	if ct.ProcessedTokens != 60 {
		t.Errorf("ProcessedTokens should be 60, got %d", ct.ProcessedTokens)
	}
}

// TestContextTrackerTokenReduction tests token reduction calculation
func TestContextTrackerTokenReduction(t *testing.T) {
	ct := NewContextTracker()
	
	ct.RecordToolResultTokens(200)
	ct.RecordProcessedTokens(60)
	
	reduction := ct.GetTokenReduction()
	expectedReduction := 70.0 // (200-60)/200 * 100 = 70%
	
	if reduction != expectedReduction {
		t.Errorf("Token reduction should be %.1f%%, got %.1f%%", expectedReduction, reduction)
	}
}

// TestContextTrackerTokenReductionZero tests token reduction when no tool results
func TestContextTrackerTokenReductionZero(t *testing.T) {
	ct := NewContextTracker()
	
	reduction := ct.GetTokenReduction()
	
	if reduction != 0 {
		t.Errorf("Token reduction should be 0 when no tool results, got %.1f%%", reduction)
	}
}

// TestContextTrackerDuration tests duration calculation
func TestContextTrackerDuration(t *testing.T) {
	ct := NewContextTracker()
	
	// Sleep a bit to ensure duration is measurable
	time.Sleep(10 * time.Millisecond)
	
	duration := ct.GetDuration()
	
	if duration < 10*time.Millisecond {
		t.Errorf("Duration should be at least 10ms, got %v", duration)
	}
}

// TestContextTrackerComplete tests completion and duration
func TestContextTrackerComplete(t *testing.T) {
	ct := NewContextTracker()
	
	time.Sleep(10 * time.Millisecond)
	ct.Complete()
	
	if ct.EndTime.IsZero() {
		t.Error("EndTime should be set after Complete()")
	}
	
	duration := ct.GetDuration()
	if duration < 10*time.Millisecond {
		t.Errorf("Duration should be at least 10ms, got %v", duration)
	}
}

// TestContextTrackerContextSize tests context size recording
func TestContextTrackerContextSize(t *testing.T) {
	ct := NewContextTracker()
	
	ct.RecordContextSize(5000)
	
	if ct.ContextSize != 5000 {
		t.Errorf("ContextSize should be 5000, got %d", ct.ContextSize)
	}
}

// TestContextTrackerIsContextSizeExceeded tests context size threshold check
func TestContextTrackerIsContextSizeExceeded(t *testing.T) {
	ct := NewContextTracker()
	ct.RecordContextSize(5000)
	
	if !ct.IsContextSizeExceeded(4000) {
		t.Error("IsContextSizeExceeded should return true when size > threshold")
	}
	
	if ct.IsContextSizeExceeded(6000) {
		t.Error("IsContextSizeExceeded should return false when size < threshold")
	}
}

// TestContextTrackerGetContextSizePercentage tests context size percentage
func TestContextTrackerGetContextSizePercentage(t *testing.T) {
	ct := NewContextTracker()
	ct.RecordContextSize(5000)
	
	percentage := ct.GetContextSizePercentage(10000)
	expectedPercentage := 50.0
	
	if percentage != expectedPercentage {
		t.Errorf("Context size percentage should be %.1f%%, got %.1f%%", expectedPercentage, percentage)
	}
}

// TestContextTrackerShouldApplySlidingWindow tests sliding window decision
func TestContextTrackerShouldApplySlidingWindow(t *testing.T) {
	ct := NewContextTracker()
	ct.RecordContextSize(6000)
	
	// Should apply when at 60% of max (> 50%)
	if !ct.ShouldApplySlidingWindow(10000) {
		t.Error("ShouldApplySlidingWindow should return true when context is > 50% of max")
	}
	
	// Should not apply when at 40% of max (< 50%)
	ct.RecordContextSize(4000)
	if ct.ShouldApplySlidingWindow(10000) {
		t.Error("ShouldApplySlidingWindow should return false when context is < 50% of max")
	}
}

// TestContextTrackerGetMetricsMap tests metrics map generation
func TestContextTrackerGetMetricsMap(t *testing.T) {
	ct := NewContextTracker()
	ct.RecordInputTokens(100)
	ct.RecordOutputTokens(50)
	ct.RecordToolResultTokens(200)
	ct.RecordProcessedTokens(60)
	ct.RecordContextSize(5000)
	
	metrics := ct.GetMetricsMap()
	
	if metrics["input_tokens"] != 100 {
		t.Errorf("input_tokens should be 100, got %v", metrics["input_tokens"])
	}
	
	if metrics["output_tokens"] != 50 {
		t.Errorf("output_tokens should be 50, got %v", metrics["output_tokens"])
	}
	
	if metrics["context_size"] != 5000 {
		t.Errorf("context_size should be 5000, got %v", metrics["context_size"])
	}
	
	if metrics["token_reduction"] != 70.0 {
		t.Errorf("token_reduction should be 70.0, got %v", metrics["token_reduction"])
	}
}

// TestCalculateContextSize tests context size calculation from messages
func TestCalculateContextSize(t *testing.T) {
	messages := []Message{
		{
			Role:    "user",
			Content: "Hello, this is a test message",
		},
		{
			Role:    "assistant",
			Content: "This is a response",
		},
	}
	
	size := CalculateContextSize(messages)
	
	if size == 0 {
		t.Error("CalculateContextSize should return non-zero size")
	}
	
	// Verify it includes both messages
	expectedMinSize := len("Hello, this is a test message") + len("This is a response")
	if size < expectedMinSize {
		t.Errorf("CalculateContextSize should be at least %d, got %d", expectedMinSize, size)
	}
}

// TestCalculateContextSizeWithToolCalls tests context size with tool calls
func TestCalculateContextSizeWithToolCalls(t *testing.T) {
	messages := []Message{
		{
			Role:    "user",
			Content: "Call a tool",
		},
		{
			Role:    "tool_call",
			Content: "Calling tool",
			ToolCall: &ToolCall{
				ID:   "tool-1",
				Name: "test_tool",
				Args: map[string]interface{}{
					"param1": "value1",
					"param2": 42,
				},
			},
		},
	}
	
	size := CalculateContextSize(messages)
	
	if size == 0 {
		t.Error("CalculateContextSize should return non-zero size with tool calls")
	}
}

// TestContextTrackerIntegration tests full integration workflow
func TestContextTrackerIntegration(t *testing.T) {
	ct := NewContextTracker()
	
	// Simulate token tracking
	ct.RecordInputTokens(150)
	ct.RecordOutputTokens(75)
	ct.RecordToolResultTokens(300)
	ct.RecordProcessedTokens(90)
	ct.RecordContextSize(8000)
	
	// Simulate processing
	time.Sleep(5 * time.Millisecond)
	ct.Complete()
	
	// Verify all metrics
	if ct.InputTokens != 150 {
		t.Errorf("InputTokens should be 150, got %d", ct.InputTokens)
	}
	
	reduction := ct.GetTokenReduction()
	if reduction != 70.0 {
		t.Errorf("Token reduction should be 70%%, got %.1f%%", reduction)
	}
	
	if !ct.IsContextSizeExceeded(7000) {
		t.Error("Context size should exceed 7000")
	}
	
	if ct.IsContextSizeExceeded(9000) {
		t.Error("Context size should not exceed 9000")
	}
	
	metrics := ct.GetMetricsMap()
	if len(metrics) == 0 {
		t.Error("Metrics map should not be empty")
	}
}


// TestScenarioHelper provides utilities for testing real chat scenarios
type TestScenarioHelper struct {
	tracker *ContextTracker
	messages []Message
}

// NewTestScenarioHelper creates a new test scenario helper
func NewTestScenarioHelper() *TestScenarioHelper {
	return &TestScenarioHelper{
		tracker:  NewContextTracker(),
		messages: make([]Message, 0),
	}
}

// AddMessage adds a message to the scenario
func (tsh *TestScenarioHelper) AddMessage(role, content string) {
	tsh.messages = append(tsh.messages, Message{
		Role:    role,
		Content: content,
	})
}

// SimulateToolExecution simulates a tool execution with metrics
func (tsh *TestScenarioHelper) SimulateToolExecution(toolName string, inputTokens, outputTokens, toolResultTokens int) {
	tsh.tracker.RecordInputTokens(inputTokens)
	tsh.tracker.RecordOutputTokens(outputTokens)
	tsh.tracker.RecordToolResultTokens(toolResultTokens)
	
	// Simulate 70% reduction for normal tools
	processedTokens := int(float64(toolResultTokens) * 0.3)
	tsh.tracker.RecordProcessedTokens(processedTokens)
	
	// Update context size
	contextSize := CalculateContextSize(tsh.messages)
	tsh.tracker.RecordContextSize(contextSize)
}

// GetMetrics returns the current metrics
func (tsh *TestScenarioHelper) GetMetrics() map[string]interface{} {
	return tsh.tracker.GetMetricsMap()
}

// VerifyTokenReduction verifies token reduction is within expected range
func (tsh *TestScenarioHelper) VerifyTokenReduction(minReduction, maxReduction float64) bool {
	reduction := tsh.tracker.GetTokenReduction()
	return reduction >= minReduction && reduction <= maxReduction
}

// VerifyContextSize verifies context size is within expected range
func (tsh *TestScenarioHelper) VerifyContextSize(minSize, maxSize int) bool {
	return tsh.tracker.ContextSize >= minSize && tsh.tracker.ContextSize <= maxSize
}

// TestScenario1SimpleToolExecution tests simple tool execution scenario
func TestScenario1SimpleToolExecution(t *testing.T) {
	helper := NewTestScenarioHelper()
	
	// User asks for a simple tool execution
	helper.AddMessage("user", "Read the file config.json")
	helper.AddMessage("assistant", "I'll read the file for you")
	
	// Simulate tool execution
	helper.SimulateToolExecution("read_file", 50, 30, 300)
	
	// Verify metrics
	if !helper.VerifyTokenReduction(60, 80) {
		t.Errorf("Token reduction should be 60-80%%, got %.1f%%", helper.tracker.GetTokenReduction())
	}
	
	metrics := helper.GetMetrics()
	if metrics["tool_result_tokens"] != 300 {
		t.Errorf("Tool result tokens should be 300, got %v", metrics["tool_result_tokens"])
	}
}

// TestScenario2MultipleToolCalls tests multiple sequential tool calls
func TestScenario2MultipleToolCalls(t *testing.T) {
	helper := NewTestScenarioHelper()
	
	// First tool call
	helper.AddMessage("user", "Search for authentication code")
	helper.SimulateToolExecution("code_index_search", 100, 50, 500)
	firstReduction := helper.tracker.GetTokenReduction()
	
	// Verify first tool execution
	if firstReduction < 60 || firstReduction > 80 {
		t.Errorf("First tool token reduction should be 60-80%%, got %.1f%%", firstReduction)
	}
}

// TestScenario3LargeToolResults tests large tool results scenario
func TestScenario3LargeToolResults(t *testing.T) {
	helper := NewTestScenarioHelper()
	
	// Large tool result
	helper.AddMessage("user", "Search for all API endpoints")
	helper.SimulateToolExecution("code_index_search", 50, 30, 5000)
	
	// Verify token reduction for large results (70% with default 30% retention)
	if !helper.VerifyTokenReduction(60, 80) {
		t.Errorf("Token reduction for large results should be 60-80%%, got %.1f%%", helper.tracker.GetTokenReduction())
	}
	
	metrics := helper.GetMetrics()
	if metrics["tool_result_tokens"] != 5000 {
		t.Errorf("Tool result tokens should be 5000, got %v", metrics["tool_result_tokens"])
	}
}

// TestScenario4ContextWindowManagement tests context window management
func TestScenario4ContextWindowManagement(t *testing.T) {
	helper := NewTestScenarioHelper()
	
	// Simulate long conversation with longer messages
	for i := 0; i < 5; i++ {
		helper.AddMessage("user", "This is a longer question to increase context size "+string(rune(i)))
		helper.AddMessage("assistant", "This is a longer answer to increase context size "+string(rune(i)))
		helper.SimulateToolExecution("read_file", 100, 50, 300)
	}
	
	// Verify context size grows appropriately
	contextSize := helper.tracker.ContextSize
	if contextSize < 100 {
		t.Errorf("Context size should be > 100, got %d", contextSize)
	}
	
	// Verify sliding window decision
	maxContextSize := 50000
	shouldApply := helper.tracker.ShouldApplySlidingWindow(maxContextSize)
	// Should not apply if context is less than 50% of max
	if shouldApply && contextSize < maxContextSize/2 {
		t.Error("Sliding window should not be applied when context < 50% of max")
	}
}
