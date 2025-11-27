# Adaptive Token-Based Compactor - Implementation Plan

## Overview

Implement an adaptive context compaction system that automatically summarizes older messages when approaching Claude's 128K token limit, preserving recent context and tool call pairs.

### Strategy: 90%/60% Threshold

| Threshold | Tokens | Percentage | Purpose |
|-----------|--------|------------|---------|
| Trigger | 115,200 | 90% | Start compaction |
| Target | 76,800 | 60% | Post-compaction goal |
| Buffer | 2,000 | - | Reserve for AI summary |
| Per-message max | 8,000 | - | Individual message limit |

---

## Existing Infrastructure (Reuse)

| Component | Location | Status |
|-----------|----------|--------|
| `TokenEstimator` | `handlers/token_estimator.go` | ✅ Exists |
| `ChatMessage.TokenCount` | `models/chat.go:33` | ✅ Exists |
| `ChatMessage.IsSummary` | `models/chat.go:35` | ✅ Exists |
| `ChatMessage.IsArchived` | `models/chat.go:34` | ✅ Exists |
| `ChatMessage.OriginalMessageCount` | `models/chat.go:36` | ✅ Exists |
| Summarizer service | `mcp/summarizer/` | ✅ Exists (for code) |

---

## Phase 1: Configuration & Types (Day 1)

**Goal:** Define compaction configuration and types

**Files to create:**
- `internal/handlers/compaction_config.go`

### Implementation

```go
package handlers

// CompactionConfig defines thresholds for context compaction
type CompactionConfig struct {
    // Context window settings
    MaxContextTokens        int     // 128,000 (Claude's limit)
    TriggerThreshold        float64 // 0.90 (90%)
    TargetThreshold         float64 // 0.60 (60%)

    // Buffer settings
    SummaryBufferTokens     int     // 2,000 (reserve for AI summary)
    PerMessageMaxTokens     int     // 8,000 (truncate large messages)

    // Behavior settings
    PreserveToolPairs       bool    // Keep tool_call + tool_result together
    PreserveRecentCount     int     // Minimum recent messages to keep (e.g., 4)
    AggressiveMode          bool    // Force compaction of all but last message
    ValidateAfterCompaction bool    // Verify token count post-compaction
}

// DefaultCompactionConfig returns production-ready defaults
func DefaultCompactionConfig() *CompactionConfig {
    return &CompactionConfig{
        MaxContextTokens:        128000,
        TriggerThreshold:        0.90,
        TargetThreshold:         0.60,
        SummaryBufferTokens:     2000,
        PerMessageMaxTokens:     8000,
        PreserveToolPairs:       true,
        PreserveRecentCount:     4,
        AggressiveMode:          false,
        ValidateAfterCompaction: true,
    }
}

// Computed thresholds
func (c *CompactionConfig) TriggerTokens() int {
    return int(float64(c.MaxContextTokens) * c.TriggerThreshold)
}

func (c *CompactionConfig) TargetTokens() int {
    return int(float64(c.MaxContextTokens) * c.TargetThreshold)
}

// CompactionResult holds the result of a compaction operation
type CompactionResult struct {
    WasCompacted        bool
    OriginalTokens      int
    CompactedTokens     int
    MessagesCompacted   int
    MessagesKept        int
    SummaryGenerated    bool
    Error               error
}
```

**Effort:** ~1 hour

---

## Phase 2: Enhanced Token Estimator (Day 1-2)

**Goal:** Add caching and message-level estimation

**Files to modify:**
- `internal/handlers/token_estimator.go`

### Implementation

```go
// Add to token_estimator.go

// MessageTokenCache caches token counts per message
type MessageTokenCache struct {
    cache map[string]cachedTokenCount // messageID -> count
    mu    sync.RWMutex
}

type cachedTokenCount struct {
    tokens    int
    version   int64 // For invalidation
    timestamp time.Time
}

// NewMessageTokenCache creates a new cache
func NewMessageTokenCache() *MessageTokenCache {
    return &MessageTokenCache{
        cache: make(map[string]cachedTokenCount),
    }
}

// GetOrEstimate returns cached count or estimates and caches
func (c *MessageTokenCache) GetOrEstimate(msg *models.ChatMessage, estimator *TokenEstimator) int {
    c.mu.RLock()
    if cached, ok := c.cache[msg.ID.Hex()]; ok {
        c.mu.RUnlock()
        return cached.tokens
    }
    c.mu.RUnlock()

    // Estimate tokens for message
    tokens := estimator.EstimateMessageTokens(msg)

    c.mu.Lock()
    c.cache[msg.ID.Hex()] = cachedTokenCount{
        tokens:    tokens,
        timestamp: time.Now(),
    }
    c.mu.Unlock()

    return tokens
}

// EstimateMessageTokens estimates tokens for a ChatMessage
func (te *TokenEstimator) EstimateMessageTokens(msg *models.ChatMessage) int {
    // Use cached value if available
    if msg.TokenCount > 0 {
        return msg.TokenCount
    }

    total := te.EstimateTokens(msg.Content)

    // Add overhead for role/metadata (~10 tokens)
    total += 10

    // Add tool call tokens if present
    if msg.ToolCall != nil {
        total += te.EstimateTokens(msg.ToolCall.Args) + 20 // tool name + structure
    }

    // Add tool result tokens if present
    if msg.ToolResult != nil {
        total += te.EstimateTokens(msg.ToolResult.Output) + 20
    }

    return total
}

// EstimateTotalTokens estimates total tokens for a message slice
func (te *TokenEstimator) EstimateTotalTokens(messages []models.ChatMessage) int {
    total := 0
    for i := range messages {
        total += te.EstimateMessageTokens(&messages[i])
    }
    return total
}
```

**Effort:** ~2 hours

---

## Phase 3: Core Compactor Logic (Day 2-3)

**Goal:** Implement the backward sliding window algorithm

**Files to create:**
- `internal/handlers/context_compactor.go`

### Implementation

```go
package handlers

import (
    "context"
    "hyper/internal/models"
    "go.uber.org/zap"
)

// ContextCompactor handles adaptive context window management
type ContextCompactor struct {
    config     *CompactionConfig
    estimator  *TokenEstimator
    tokenCache *MessageTokenCache
    logger     *zap.Logger
}

// NewContextCompactor creates a new compactor
func NewContextCompactor(config *CompactionConfig, logger *zap.Logger) *ContextCompactor {
    if config == nil {
        config = DefaultCompactionConfig()
    }
    return &ContextCompactor{
        config:     config,
        estimator:  NewTokenEstimator(),
        tokenCache: NewMessageTokenCache(),
        logger:     logger,
    }
}

// ShouldCompact checks if compaction is needed
func (c *ContextCompactor) ShouldCompact(messages []models.ChatMessage) bool {
    totalTokens := c.estimator.EstimateTotalTokens(messages)
    return totalTokens > c.config.TriggerTokens()
}

// CalculateSplitPoint finds where to split messages (keep recent, compact old)
// Uses backward sliding window algorithm
func (c *ContextCompactor) CalculateSplitPoint(messages []models.ChatMessage) (compactCount int, keepCount int) {
    if len(messages) == 0 {
        return 0, 0
    }

    // Calculate available tokens for kept messages
    availableTokens := c.config.TargetTokens() - c.config.SummaryBufferTokens

    // Backward sliding window: keep recent messages that fit
    recentTokens := 0
    keptCount := 0

    for i := len(messages) - 1; i >= 0; i-- {
        msgTokens := c.estimator.EstimateMessageTokens(&messages[i])

        // Cap individual message tokens
        if msgTokens > c.config.PerMessageMaxTokens {
            msgTokens = c.config.PerMessageMaxTokens
        }

        if recentTokens + msgTokens <= availableTokens {
            recentTokens += msgTokens
            keptCount++
        } else {
            break
        }
    }

    // Ensure minimum recent messages kept
    if keptCount < c.config.PreserveRecentCount && len(messages) >= c.config.PreserveRecentCount {
        keptCount = c.config.PreserveRecentCount
    }

    compactCount = len(messages) - keptCount

    // Adjust for tool pairs if needed
    if c.config.PreserveToolPairs {
        compactCount = c.adjustForToolPairs(messages, compactCount)
    }

    return compactCount, len(messages) - compactCount
}

// adjustForToolPairs ensures tool_call and tool_result stay together
func (c *ContextCompactor) adjustForToolPairs(messages []models.ChatMessage, splitIndex int) int {
    if splitIndex <= 0 || splitIndex >= len(messages) {
        return splitIndex
    }

    maxIterations := 10 // Prevent infinite loops

    for i := 0; i < maxIterations; i++ {
        if splitIndex >= len(messages) {
            break
        }

        // Check if we're splitting a tool pair
        splitMsg := messages[splitIndex]

        // If split is at tool_result, move split back to include tool_call
        if splitMsg.Role == "tool_result" && splitIndex > 0 {
            prevMsg := messages[splitIndex-1]
            if prevMsg.Role == "tool_call" || prevMsg.ToolCall != nil {
                splitIndex-- // Keep pair together
                continue
            }
        }

        // If split is at tool_call, move split forward to include tool_result
        if (splitMsg.Role == "tool_call" || splitMsg.ToolCall != nil) && splitIndex < len(messages)-1 {
            nextMsg := messages[splitIndex+1]
            if nextMsg.Role == "tool_result" {
                splitIndex++ // Keep pair together
                continue
            }
        }

        break // No adjustment needed
    }

    return splitIndex
}

// GetMessagesToCompact returns the older messages that should be summarized
func (c *ContextCompactor) GetMessagesToCompact(messages []models.ChatMessage) []models.ChatMessage {
    compactCount, _ := c.CalculateSplitPoint(messages)
    if compactCount <= 0 {
        return nil
    }
    return messages[:compactCount]
}

// GetMessagesToKeep returns the recent messages that should be preserved
func (c *ContextCompactor) GetMessagesToKeep(messages []models.ChatMessage) []models.ChatMessage {
    compactCount, _ := c.CalculateSplitPoint(messages)
    return messages[compactCount:]
}
```

**Effort:** ~4 hours

---

## Phase 4: AI-Powered Summarization (Day 3-4)

**Goal:** Generate intelligent summaries of compacted messages

**Files to create:**
- `internal/handlers/compaction_summarizer.go`

### Implementation

```go
package handlers

import (
    "context"
    "fmt"
    "strings"
    "hyper/internal/models"
)

// CompactionSummarizer generates summaries of compacted messages
type CompactionSummarizer struct {
    aiService AIServiceInterface
    logger    *zap.Logger
}

// SummaryPrompt is the system prompt for generating compaction summaries
const SummaryPrompt = `You are a conversation summarizer. Your task is to create a concise summary of the following conversation segment that will replace these messages in the context window.

REQUIREMENTS:
1. Preserve ALL key decisions, code changes, and technical details
2. Maintain tool call outcomes and their results
3. Keep file paths, function names, and important identifiers
4. Note any errors encountered and how they were resolved
5. Be concise but complete - this summary replaces the original messages

FORMAT:
- Start with "**Conversation Summary** (X messages):"
- Use bullet points for key events
- Include code snippets only if critical
- Maximum 500 words

CONVERSATION TO SUMMARIZE:`

// GenerateSummary creates an AI-powered summary of messages
func (s *CompactionSummarizer) GenerateSummary(
    ctx context.Context,
    messages []models.ChatMessage,
) (string, error) {
    if len(messages) == 0 {
        return "", nil
    }

    // Build conversation text for summarization
    var conversationBuilder strings.Builder
    for _, msg := range messages {
        conversationBuilder.WriteString(fmt.Sprintf("\n[%s]: %s", msg.Role, msg.Content))

        if msg.ToolCall != nil {
            conversationBuilder.WriteString(fmt.Sprintf("\n  Tool: %s", msg.ToolCall.Name))
        }
        if msg.ToolResult != nil {
            output := fmt.Sprintf("%v", msg.ToolResult.Output)
            if len(output) > 500 {
                output = output[:500] + "..."
            }
            conversationBuilder.WriteString(fmt.Sprintf("\n  Result: %s", output))
        }
    }

    // Call AI to generate summary
    summary, err := s.aiService.GenerateCompletion(ctx, SummaryPrompt, conversationBuilder.String())
    if err != nil {
        // Fallback to simple summary
        return s.generateFallbackSummary(messages), nil
    }

    return summary, nil
}

// generateFallbackSummary creates a basic summary without AI
func (s *CompactionSummarizer) generateFallbackSummary(messages []models.ChatMessage) string {
    var summary strings.Builder
    summary.WriteString(fmt.Sprintf("**Conversation Summary** (%d messages):\n", len(messages)))

    toolCalls := 0
    userMessages := 0

    for _, msg := range messages {
        switch msg.Role {
        case "user":
            userMessages++
        case "tool_call", "tool_result":
            toolCalls++
        }
    }

    summary.WriteString(fmt.Sprintf("- %d user messages\n", userMessages))
    summary.WriteString(fmt.Sprintf("- %d tool interactions\n", toolCalls/2))

    // Include first user message as context
    for _, msg := range messages {
        if msg.Role == "user" && len(msg.Content) > 0 {
            content := msg.Content
            if len(content) > 200 {
                content = content[:200] + "..."
            }
            summary.WriteString(fmt.Sprintf("- Started with: \"%s\"\n", content))
            break
        }
    }

    return summary.String()
}
```

**Effort:** ~3 hours

---

## Phase 5: Compaction Orchestration (Day 4-5)

**Goal:** Coordinate compaction flow and database updates

**Files to create:**
- `internal/handlers/compaction_orchestrator.go`

### Implementation

```go
package handlers

import (
    "context"
    "time"
    "hyper/internal/models"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

// CompactionOrchestrator coordinates the full compaction workflow
type CompactionOrchestrator struct {
    compactor   *ContextCompactor
    summarizer  *CompactionSummarizer
    chatService ChatServiceInterface
    logger      *zap.Logger
}

// NewCompactionOrchestrator creates a new orchestrator
func NewCompactionOrchestrator(
    config *CompactionConfig,
    aiService AIServiceInterface,
    chatService ChatServiceInterface,
    logger *zap.Logger,
) *CompactionOrchestrator {
    return &CompactionOrchestrator{
        compactor:   NewContextCompactor(config, logger),
        summarizer:  &CompactionSummarizer{aiService: aiService, logger: logger},
        chatService: chatService,
        logger:      logger,
    }
}

// CompactIfNeeded checks and performs compaction if necessary
func (o *CompactionOrchestrator) CompactIfNeeded(
    ctx context.Context,
    sessionID primitive.ObjectID,
    messages []models.ChatMessage,
) (*CompactionResult, error) {
    result := &CompactionResult{
        OriginalTokens: o.compactor.estimator.EstimateTotalTokens(messages),
    }

    // Check if compaction needed
    if !o.compactor.ShouldCompact(messages) {
        o.logger.Debug("Compaction not needed",
            zap.Int("tokens", result.OriginalTokens),
            zap.Int("threshold", o.compactor.config.TriggerTokens()))
        return result, nil
    }

    o.logger.Info("🗜️ Starting context compaction",
        zap.String("sessionId", sessionID.Hex()),
        zap.Int("originalTokens", result.OriginalTokens),
        zap.Int("messageCount", len(messages)))

    // Calculate split point
    toCompact := o.compactor.GetMessagesToCompact(messages)
    toKeep := o.compactor.GetMessagesToKeep(messages)

    if len(toCompact) == 0 {
        return result, nil
    }

    // Generate summary of compacted messages
    summary, err := o.summarizer.GenerateSummary(ctx, toCompact)
    if err != nil {
        o.logger.Error("Failed to generate summary", zap.Error(err))
        return result, err
    }

    // Create summary message
    summaryMsg := &models.ChatMessage{
        SessionID:            sessionID,
        Role:                 "system",
        Content:              summary,
        Timestamp:            time.Now(),
        IsSummary:            true,
        OriginalMessageCount: len(toCompact),
        TokenCount:           o.compactor.estimator.EstimateTokens(summary),
    }

    // Archive compacted messages and save summary
    err = o.chatService.ArchiveMessages(ctx, sessionID, toCompact)
    if err != nil {
        return result, err
    }

    err = o.chatService.SaveSummaryMessage(ctx, summaryMsg)
    if err != nil {
        return result, err
    }

    // Update result
    result.WasCompacted = true
    result.MessagesCompacted = len(toCompact)
    result.MessagesKept = len(toKeep)
    result.SummaryGenerated = true
    result.CompactedTokens = o.compactor.estimator.EstimateTotalTokens(toKeep) + summaryMsg.TokenCount

    o.logger.Info("✅ Context compaction complete",
        zap.Int("compactedMessages", result.MessagesCompacted),
        zap.Int("keptMessages", result.MessagesKept),
        zap.Int("originalTokens", result.OriginalTokens),
        zap.Int("newTokens", result.CompactedTokens),
        zap.Float64("reduction", float64(result.OriginalTokens-result.CompactedTokens)/float64(result.OriginalTokens)*100))

    return result, nil
}
```

**Effort:** ~3 hours

---

## Phase 6: WebSocket Integration (Day 5-6)

**Goal:** Integrate compactor into chat execution flow

**Files to modify:**
- `internal/handlers/chat_websocket.go`

### Implementation

```go
// Add field to ChatWebSocketHandler
type ChatWebSocketHandler struct {
    // ... existing fields ...
    compactionOrchestrator *CompactionOrchestrator
}

// Update constructor
func NewChatWebSocketHandler(...) *ChatWebSocketHandler {
    // ... existing code ...

    compactionOrchestrator := NewCompactionOrchestrator(
        DefaultCompactionConfig(),
        aiService,
        chatService,
        logger,
    )

    return &ChatWebSocketHandler{
        // ... existing fields ...
        compactionOrchestrator: compactionOrchestrator,
    }
}

// Add compaction check before AI execution (in handleUserMessage)
func (h *ChatWebSocketHandler) handleUserMessage(...) {
    // ... existing code to get messages ...

    // Check and perform compaction if needed
    compactionResult, err := h.compactionOrchestrator.CompactIfNeeded(ctx, sessionID, messages)
    if err != nil {
        h.logger.Error("Compaction failed", zap.Error(err))
        // Continue with original messages - non-fatal
    }

    if compactionResult.WasCompacted {
        // Notify client about compaction
        h.safeWriteJSON(conn, models.StreamMessage{
            Type: "context_compacted",
            Content: fmt.Sprintf("Context compacted: %d messages summarized, %d%% token reduction",
                compactionResult.MessagesCompacted,
                int((float64(compactionResult.OriginalTokens-compactionResult.CompactedTokens)/
                    float64(compactionResult.OriginalTokens))*100)),
        })

        // Reload messages after compaction
        messages, _ = h.chatService.GetSessionMessages(ctx, sessionID)
        langchainMessages = aiservice.ConvertToLangChainMessages(messages)
    }

    // ... continue with AI execution ...
}
```

**Effort:** ~2 hours

---

## Phase 7: Size-Based Compaction (Day 6-7)

**Goal:** Add MongoDB document size limit protection

**Files to create:**
- `internal/handlers/size_based_compactor.go`

### Implementation

```go
package handlers

const (
    MongoDBMaxDocSize = 16 * 1024 * 1024 // 16MB
    SizeTriggerThreshold = 0.80          // 80% = 12.8MB
    SizeTargetThreshold  = 0.20          // 20% = 3.2MB
)

// SizeBasedCompactor monitors BSON document size
type SizeBasedCompactor struct {
    config *CompactionConfig
    logger *zap.Logger
}

// EstimateSessionBSONSize estimates the BSON size of a session's messages
func (c *SizeBasedCompactor) EstimateSessionBSONSize(messages []models.ChatMessage) int {
    totalSize := 0
    for _, msg := range messages {
        // Base message overhead (~200 bytes for BSON structure)
        totalSize += 200
        totalSize += len(msg.Content)

        if msg.ToolCall != nil {
            // Estimate tool call size
            totalSize += 100 // structure overhead
            if args, err := json.Marshal(msg.ToolCall.Args); err == nil {
                totalSize += len(args)
            }
        }

        if msg.ToolResult != nil {
            totalSize += 100
            if output, err := json.Marshal(msg.ToolResult.Output); err == nil {
                totalSize += len(output)
            }
        }
    }
    return totalSize
}

// ShouldCompactBySize checks if size-based compaction is needed
func (c *SizeBasedCompactor) ShouldCompactBySize(messages []models.ChatMessage) bool {
    size := c.EstimateSessionBSONSize(messages)
    threshold := int(float64(MongoDBMaxDocSize) * SizeTriggerThreshold)
    return size > threshold
}
```

**Effort:** ~2 hours

---

## Phase 8: Metrics & Monitoring (Day 7)

**Goal:** Add observability for compaction operations

**Files to modify:**
- `internal/metrics/registry.go`

### Implementation

```go
// Add to registry.go

var (
    CompactionOperations = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "hyperion_context_compaction_total",
            Help: "Total context compaction operations",
        },
        []string{"session_type", "trigger"}, // trigger: tokens, size, emergency
    )

    CompactionTokensReduced = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "hyperion_compaction_tokens_reduced",
            Help:    "Tokens reduced by compaction",
            Buckets: []float64{1000, 5000, 10000, 25000, 50000, 100000},
        },
        []string{"session_type"},
    )

    CompactionMessagesCompacted = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "hyperion_compaction_messages_compacted",
            Help:    "Messages compacted per operation",
            Buckets: []float64{5, 10, 20, 50, 100, 200},
        },
        []string{"session_type"},
    )

    ContextTokenUsage = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "hyperion_context_token_usage",
            Help: "Current token usage by session",
        },
        []string{"session_id"},
    )
)

func RecordCompaction(sessionType, trigger string, tokensReduced, messagesCompacted int) {
    CompactionOperations.WithLabelValues(sessionType, trigger).Inc()
    CompactionTokensReduced.WithLabelValues(sessionType).Observe(float64(tokensReduced))
    CompactionMessagesCompacted.WithLabelValues(sessionType).Observe(float64(messagesCompacted))
}
```

**Effort:** ~1 hour

---

## Phase 9: Testing (Day 7-8)

**Goal:** Comprehensive test suite

**Files to create:**
- `internal/handlers/context_compactor_test.go`
- `internal/handlers/compaction_orchestrator_test.go`

### Key Test Cases

```go
func TestShouldCompact(t *testing.T) {
    // Test trigger threshold detection
}

func TestCalculateSplitPoint(t *testing.T) {
    // Test backward sliding window
}

func TestToolPairPreservation(t *testing.T) {
    // Test tool_call + tool_result stay together
}

func TestAggressiveMode(t *testing.T) {
    // Test emergency compaction
}

func TestSizeBasedCompaction(t *testing.T) {
    // Test MongoDB size limits
}

func TestCompactionE2E(t *testing.T) {
    // Full workflow test
}
```

**Effort:** ~4 hours

---

## Summary

| Phase | Description | Files | Effort |
|-------|-------------|-------|--------|
| 1 | Configuration & Types | `compaction_config.go` | 1 hr |
| 2 | Enhanced Token Estimator | `token_estimator.go` | 2 hrs |
| 3 | Core Compactor Logic | `context_compactor.go` | 4 hrs |
| 4 | AI-Powered Summarization | `compaction_summarizer.go` | 3 hrs |
| 5 | Compaction Orchestration | `compaction_orchestrator.go` | 3 hrs |
| 6 | WebSocket Integration | `chat_websocket.go` | 2 hrs |
| 7 | Size-Based Compaction | `size_based_compactor.go` | 2 hrs |
| 8 | Metrics & Monitoring | `registry.go` | 1 hr |
| 9 | Testing | `*_test.go` | 4 hrs |

**Total Estimated Effort: ~22 hours (3-4 days)**

---

## Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                     User Sends Message                          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                  Load Session Messages                          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│              Check: ShouldCompact(messages)?                    │
│                 Token count > 90% of 128K?                      │
└─────────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┴───────────────┐
              │ NO                            │ YES
              ▼                               ▼
┌─────────────────────┐     ┌─────────────────────────────────────┐
│ Continue normally   │     │  Calculate Split Point              │
│                     │     │  (Backward Sliding Window)          │
└─────────────────────┘     └─────────────────────────────────────┘
                                              │
                                              ▼
                            ┌─────────────────────────────────────┐
                            │  Adjust for Tool Pairs              │
                            │  (Keep tool_call + tool_result)     │
                            └─────────────────────────────────────┘
                                              │
                                              ▼
                            ┌─────────────────────────────────────┐
                            │  Generate AI Summary                │
                            │  (Of messages to compact)           │
                            └─────────────────────────────────────┘
                                              │
                                              ▼
                            ┌─────────────────────────────────────┐
                            │  Archive Old Messages               │
                            │  Save Summary Message               │
                            │  Notify Client                      │
                            └─────────────────────────────────────┘
                                              │
                                              ▼
                            ┌─────────────────────────────────────┐
                            │  Continue with Compacted Context    │
                            └─────────────────────────────────────┘
```

---

## Dependencies

- `ChatServiceInterface` needs new methods:
  - `ArchiveMessages(ctx, sessionID, messages)`
  - `SaveSummaryMessage(ctx, message)`

- `AIServiceInterface` needs:
  - `GenerateCompletion(ctx, systemPrompt, userPrompt) (string, error)`

---

## Future Enhancements

1. **Streaming Compaction**: Compact during streaming without blocking
2. **Selective Preservation**: Mark important messages as "never compact"
3. **Differential Summaries**: Only summarize what changed since last summary
4. **User Control**: Allow users to trigger manual compaction
5. **Multi-level Summaries**: Summary of summaries for very long sessions
