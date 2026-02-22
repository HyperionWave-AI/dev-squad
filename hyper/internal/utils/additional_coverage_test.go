package utils

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"hyper/internal/models"
)

func TestContextHealthString(t *testing.T) {
	if got := ContextHealthWarning.String(); got != "warning" {
		t.Fatalf("ContextHealthWarning.String() = %q, want %q", got, "warning")
	}
}

func TestCanAddMessageBranches(t *testing.T) {
	cm := NewContextManager(ContextLimitConfig{
		MaxTokens:              10,
		WarningThreshold:       80,
		CriticalThreshold:      90,
		AutoSummarizeThreshold: 80,
	}, zap.NewNop())

	ok, usage := cm.CanAddMessage("missing-session", "hello")
	if !ok || usage != nil {
		t.Fatalf("expected allow for unknown session, got ok=%v usage=%v", ok, usage)
	}

	cm.sessionUsage["session-1"] = &ContextUsage{
		SessionID:   "session-1",
		TotalTokens: 9,
		MaxTokens:   10,
	}
	ok, _ = cm.CanAddMessage("session-1", "this message should exceed")
	if ok {
		t.Fatal("expected rejection when projected tokens exceed max")
	}
}

func TestCheckContextHealthBranches(t *testing.T) {
	cm := NewContextManager(DefaultContextLimitConfig(), zap.NewNop())

	if got := cm.CheckContextHealth("unknown"); got != ContextHealthHealthy {
		t.Fatalf("expected healthy for unknown session, got %s", got)
	}

	cm.sessionUsage["warn"] = &ContextUsage{IsWarning: true}
	if got := cm.CheckContextHealth("warn"); got != ContextHealthWarning {
		t.Fatalf("expected warning, got %s", got)
	}

	cm.sessionUsage["critical"] = &ContextUsage{IsCritical: true}
	if got := cm.CheckContextHealth("critical"); got != ContextHealthCritical {
		t.Fatalf("expected critical, got %s", got)
	}
}

func TestUpdateContextUsageTriggersSummarizationOnlyBranch(t *testing.T) {
	cfg := ContextLimitConfig{
		MaxTokens:              100,
		WarningThreshold:       90,
		CriticalThreshold:      95,
		AutoSummarizeThreshold: 50,
	}
	cm := NewContextManager(cfg, zap.NewNop())

	msgs := []models.ChatMessage{
		{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      "user",
			Content:   "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			Timestamp: time.Now(),
		},
	}

	usage := cm.UpdateContextUsage(context.Background(), "session-summarize", msgs)
	if !usage.NeedsSummarization {
		t.Fatal("expected NeedsSummarization to be true")
	}
	if usage.IsWarning || usage.IsCritical {
		t.Fatalf("expected summarization-only path, got warning=%v critical=%v", usage.IsWarning, usage.IsCritical)
	}
}

func TestNewContextErrorBranches(t *testing.T) {
	warnUsage := &ContextUsage{
		TotalTokens:    800,
		MaxTokens:      1000,
		PercentageUsed: 80,
		IsWarning:      true,
	}
	warnErr := NewContextError("WARN", "warning state", warnUsage)
	if !warnErr.CanArchiveMessages || !warnErr.CanSummarize {
		t.Fatalf("expected warning recovery flags enabled, got %+v", warnErr)
	}
	if len(warnErr.RecoveryOptions) == 0 {
		t.Fatal("expected warning recovery options")
	}

	healthyUsage := &ContextUsage{
		TotalTokens:    100,
		MaxTokens:      1000,
		PercentageUsed: 10,
	}
	healthyErr := NewContextError("INFO", "healthy state", healthyUsage)
	if len(healthyErr.RecoveryOptions) != 0 {
		t.Fatalf("expected no recovery options for healthy usage, got %+v", healthyErr.RecoveryOptions)
	}
}

func TestCountMessageTokensWithToolDataBranches(t *testing.T) {
	tc := NewTokenCounter()
	msg := &models.ChatMessage{
		Role:    "assistant",
		Content: "response",
		ToolCall: &models.ToolCallData{
			ID:   "call-1",
			Name: "tool_a",
			Args: map[string]interface{}{
				"query": "hello",
			},
		},
		ToolResult: &models.ToolResultData{
			ID:     "result-1",
			Name:   "tool_a",
			Output: map[string]interface{}{"ok": true},
			Error:  "",
		},
	}

	if got := tc.CountMessageTokens(msg); got <= 0 {
		t.Fatalf("expected positive token count, got %d", got)
	}

	// Exercise JSON marshal error paths for tool args/output.
	msg.ToolCall.Args = map[string]interface{}{"bad": make(chan int)}
	msg.ToolResult.Output = map[string]interface{}{"bad": make(chan int)}
	if got := tc.CountMessageTokens(msg); got <= 0 {
		t.Fatalf("expected positive token count even with marshal errors, got %d", got)
	}
}

func TestMessageSummarizerAdditionalBranches(t *testing.T) {
	ms := NewMessageSummarizer(zap.NewNop())
	now := time.Now()

	var messages []models.ChatMessage
	for i := 0; i < 25; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		messages = append(messages, models.ChatMessage{
			ID:        primitive.NewObjectID(),
			SessionID: primitive.NewObjectID(),
			Role:      role,
			Content:   "message content",
			Timestamp: now.Add(time.Duration(i) * time.Minute),
		})
	}

	// len <= keepRecentCount branch
	if got := ms.IdentifyMessagesForSummarization(messages[:2], 2); len(got) != 0 {
		t.Fatalf("expected empty summarize list, got %d", len(got))
	}

	// StrategyOldestFirst branch
	oldest, err := ms.SummarizeMessages(context.Background(), messages, StrategyOldestFirst)
	if err != nil {
		t.Fatalf("StrategyOldestFirst returned error: %v", err)
	}
	if oldest == nil {
		t.Fatal("expected summarization result")
	}

	// StrategyByRole branch
	byRole, err := ms.SummarizeMessages(context.Background(), messages, StrategyByRole)
	if err != nil {
		t.Fatalf("StrategyByRole returned error: %v", err)
	}
	if byRole == nil {
		t.Fatal("expected summarization result for by-role strategy")
	}

	// Unknown strategy branch
	if _, err := ms.SummarizeMessages(context.Background(), messages, SummarizationStrategy("unknown")); err == nil {
		t.Fatal("expected error for unknown summarization strategy")
	}

	// Empty input branch
	empty, err := ms.SummarizeMessages(context.Background(), nil, StrategyTimeWindow)
	if err != nil {
		t.Fatalf("empty summarize returned error: %v", err)
	}
	if empty.TotalOriginalTokens != 0 || empty.SummarizedMessageCount != 0 {
		t.Fatalf("expected zeroed empty result, got %+v", empty)
	}

	// CalculateSummarizationImpact no-op branch
	saved, count := ms.CalculateSummarizationImpact(messages[:2], 2)
	if saved != 0 || count != 0 {
		t.Fatalf("expected no impact when nothing to summarize, got saved=%d count=%d", saved, count)
	}
}
