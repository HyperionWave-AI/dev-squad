package executor

// This file contains USAGE EXAMPLES for integrating the executor package.
// DO NOT COMPILE - these are reference implementations for migration.

/*
╔═══════════════════════════════════════════════════════════════════════════╗
║                        USAGE EXAMPLE 1: PARENT CHAT                       ║
║                   (WebSocket streaming to UI client)                      ║
╚═══════════════════════════════════════════════════════════════════════════╝

LOCATION: hyper/internal/handlers/chat_websocket.go
REPLACES:  streamAIResponse() function (lines 1364-2061)

func (h *ChatWebSocketHandler) streamAIResponse(
	ctx context.Context,
	conn *websocket.Conn,
	sessionID primitive.ObjectID,
	userMessage, companyID string,
	cleanup *StreamCleanup,
) {
	// Step 1: Get session and determine system prompt
	session, err := h.chatService.GetSession(ctx, sessionID, companyID)
	if err != nil {
		h.logger.Error("Failed to retrieve session", zap.Error(err))
		h.sendError(conn, "Failed to retrieve session")
		return
	}

	// Step 2: Determine system prompt (subagent vs. global)
	var systemPromptText string
	// ... existing prompt logic ...

	// Step 3: Get conversation history
	messages, err := h.chatService.GetSessionMessages(ctx, sessionID)
	if err != nil {
		h.logger.Error("Failed to retrieve conversation history", zap.Error(err))
		h.sendError(conn, "Failed to retrieve conversation history")
		return
	}

	// Step 4: Convert messages to LangChain format
	langchainMessages := aiservice.ConvertToLangChainMessages(messages)

	// Step 5: Set up progress notifications for subchats
	progressCh := GetProgressNotifier(h.logger).RegisterSession(sessionID)
	defer GetProgressNotifier(h.logger).UnregisterSession(sessionID)

	cleanup.wg.Add(1)
	middleware.SafeGo(h.logger, func() {
		defer cleanup.wg.Done()
		for progress := range progressCh {
			progressMsg := models.StreamMessage{
				Type:    "token",
				Content: "\n\n" + progress.Message + "\n\n",
			}
			h.safeWriteJSON(conn, progressMsg)
		}
	})

	// Step 6: Set up interrupt handling
	notifier := GetMessageNotifier(h.logger)
	interruptCh := notifier.RegisterSession(sessionID)
	defer notifier.UnregisterSession(sessionID)

	// Step 7: Determine tool filter based on mode
	var allowedTools []string
	isDirectSubagentChat := (session.ActiveSubagentName != nil && *session.ActiveSubagentName != "") || session.ActiveSubagentID != nil
	if isDirectSubagentChat {
		allowedTools = h.aiService.GetAllowedToolsForDirectSubagent()
	} else {
		allowedTools = nil // All tools for coordinator
	}

	// ═════════════════════════════════════════════════════════════════════
	// NEW: Use executor package instead of manual streaming loop
	// ═════════════════════════════════════════════════════════════════════

	// Create WebSocket sink
	sink := NewWebSocketSink(conn, h.logger)

	// Configure executor
	config := StreamConfig{
		SessionID:    sessionID,
		CompanyID:    companyID,
		SystemPrompt: systemPromptText,
		AllowedTools: allowedTools,
		OutputSink:   sink,
		InterruptCh:  interruptCh,
		Logger:       h.logger,
		// Optional: Custom tool result processing
		ToolResultProcessor: h.processToolResultWithSizeLimit,
	}

	// Create and execute
	exec := NewStreamExecutor(config, h.chatService, h.aiService)
	fullResponse, err := exec.Execute(ctx, langchainMessages)

	if err != nil {
		h.logger.Error("Streaming failed", zap.Error(err))
		// Error already sent to client by executor
		return
	}

	h.logger.Info("AI response completed",
		zap.String("sessionId", sessionID.Hex()),
		zap.Int("responseLength", len(fullResponse)))
}

╔═══════════════════════════════════════════════════════════════════════════╗
║                        USAGE EXAMPLE 2: SUBCHAT                           ║
║              (Background execution with progress notifications)           ║
╚═══════════════════════════════════════════════════════════════════════════╝

LOCATION: hyper/internal/ai-service/tools/mcp/coordinator_tools.go
REPLACES:  executeSubagentInBackground() function (lines 2868-3400)

func (t *ExecuteSubagentTool) executeSubagentInBackground(
	subchatID string,
	agentTask *storage.AgentTask,
	parentChatID string,
	companyID string,
) {
	// Create background context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Get parent session
	parentSessionID, err := primitive.ObjectIDFromHex(parentChatID)
	if err != nil {
		t.logger.Error("Failed to parse parent chat ID", zap.Error(err))
		t.handleExecutionFailure(agentTask.ID, fmt.Sprintf("Invalid parent chat ID: %v", err))
		return
	}

	// Emit start notification
	handlers.GetProgressNotifier(t.logger).EmitProgress(
		parentSessionID,
		fmt.Sprintf("🤖 Starting subchat: %s", agentTask.AgentName),
	)

	// Create chat session for subchat
	userID := "..." // Extract from parent session
	sessionTitle := fmt.Sprintf("Subchat: %s - %s", agentTask.AgentName, agentTask.Role)
	chatSession, err := t.chatService.CreateSessionWithParent(ctx, userID, companyID, sessionTitle, &parentSessionID)
	if err != nil {
		t.logger.Error("Failed to create chat session for subchat", zap.Error(err))
		t.handleExecutionFailure(agentTask.ID, fmt.Sprintf("Failed to create chat session: %v", err))
		return
	}

	// Update subchat with session ID
	sessionIDHex := chatSession.ID.Hex()
	t.subchatStorage.UpdateSubchatSessionID(subchatID, sessionIDHex)

	// Build prompts
	systemPrompt := t.buildExecutionPhaseSystemPrompt()
	taskPrompt := t.buildSubagentTaskPrompt(agentTask)

	// Save initial user message
	t.chatService.SaveMessage(ctx, chatSession.ID, "user", taskPrompt, companyID)

	// Register for interrupts
	notifier := handlers.GetMessageNotifier(t.logger)
	interruptCh := notifier.RegisterSession(chatSession.ID)
	defer notifier.UnregisterSession(chatSession.ID)

	// Define allowed tools for subagents (implementation tools only)
	allowedTools := []string{
		"read_file",
		"write_file",
		"apply_patch",
		"bash",
		"coordinator_update_todo_status",
		"coordinator_upsert_knowledge",
	}

	// Prepare messages
	messages := []aiservice.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: taskPrompt},
	}

	// ═════════════════════════════════════════════════════════════════════
	// NEW: Use executor package instead of manual streaming loop
	// ═════════════════════════════════════════════════════════════════════

	// Create progress notification sink
	progressNotifier := handlers.GetProgressNotifier(t.logger)
	sink := NewProgressNotificationSink(parentSessionID, progressNotifier, t.logger)

	// Configure executor
	config := StreamConfig{
		SessionID:    chatSession.ID,
		CompanyID:    companyID,
		SystemPrompt: "", // Already in messages
		AllowedTools: allowedTools,
		OutputSink:   sink,
		InterruptCh:  interruptCh,
		Logger:       t.logger,
	}

	// Create and execute
	exec := NewStreamExecutor(config, t.chatService, t.aiService)
	fullResponse, err := exec.Execute(ctx, messages)

	if err != nil {
		t.logger.Error("Subchat execution failed", zap.Error(err))
		handlers.GetProgressNotifier(t.logger).EmitProgress(
			parentSessionID,
			fmt.Sprintf("⚠️ Subchat failed: %s", agentTask.AgentName),
		)
		t.handleExecutionFailure(agentTask.ID, fmt.Sprintf("Execution failed: %v", err))
		return
	}

	// Mark task as complete
	t.logger.Info("Subchat execution completed",
		zap.String("subchatId", subchatID),
		zap.String("agentName", agentTask.AgentName),
		zap.Int("responseLength", len(fullResponse)))

	handlers.GetProgressNotifier(t.logger).EmitProgress(
		parentSessionID,
		fmt.Sprintf("✅ Subchat complete: %s", agentTask.AgentName),
	)

	t.handleExecutionSuccess(agentTask.ID)
}

╔═══════════════════════════════════════════════════════════════════════════╗
║                    USAGE EXAMPLE 3: CUSTOM PROCESSING                     ║
║              (Tool result size limits and custom completion)              ║
╚═══════════════════════════════════════════════════════════════════════════╝

func (h *ChatWebSocketHandler) streamAIResponseWithCustomProcessing(
	ctx context.Context,
	conn *websocket.Conn,
	sessionID primitive.ObjectID,
	companyID string,
	messages []aiservice.Message,
) {
	// Custom tool result processor (size-aware)
	toolResultProcessor := func(toolName string, output interface{}) (string, bool, bool) {
		outputStr := fmt.Sprintf("%v", output)
		size := len(outputStr)

		// Tier 1: Normal (< 120KB)
		if size <= config.MaxToolResultNormalBytes {
			return outputStr, true, true // Save and stream
		}

		// Tier 2: Truncated (120KB - 500KB)
		if size <= config.MaxToolResultTruncatedBytes {
			preview := outputStr[:config.ToolResultPreviewBytes]
			message := fmt.Sprintf("%s\n\n_[Truncated: %s of %s shown]_",
				preview,
				config.FormatSize(config.ToolResultPreviewBytes),
				config.FormatSize(size))
			return message, true, true // Save full, stream truncated
		}

		// Tier 3: Suppressed (> 500KB)
		message := fmt.Sprintf("⚠️ Large output suppressed (%s)\n\nTool: %s\nSize: %s\n\nOutput saved to database but not displayed to prevent UI overload.",
			config.FormatSize(size),
			toolName,
			config.FormatSize(size))
		return message, true, false // Save full, stream message only
	}

	// Custom completion validator (stop after 10 tool calls)
	completionValidator := func(fullResponse string, toolCallCount int) bool {
		if toolCallCount >= 10 {
			h.logger.Info("Stopping stream - max tool calls reached",
				zap.Int("toolCallCount", toolCallCount))
			return true
		}
		return false
	}

	// Register for interrupts
	notifier := GetMessageNotifier(h.logger)
	interruptCh := notifier.RegisterSession(sessionID)
	defer notifier.UnregisterSession(sessionID)

	// Create sink
	sink := NewWebSocketSink(conn, h.logger)

	// Configure executor with custom processing
	config := StreamConfig{
		SessionID:           sessionID,
		CompanyID:           companyID,
		SystemPrompt:        "",
		AllowedTools:        nil, // All tools
		OutputSink:          sink,
		InterruptCh:         interruptCh,
		ToolResultProcessor: toolResultProcessor,
		CompletionValidator: completionValidator,
		Logger:              h.logger,
	}

	// Create and execute
	exec := NewStreamExecutor(config, h.chatService, h.aiService)
	fullResponse, err := exec.Execute(ctx, messages)

	if err != nil {
		h.logger.Error("Streaming failed", zap.Error(err))
		return
	}

	h.logger.Info("Custom processing completed",
		zap.String("sessionId", sessionID.Hex()),
		zap.Int("responseLength", len(fullResponse)))
}

╔═══════════════════════════════════════════════════════════════════════════╗
║                     USAGE EXAMPLE 4: TESTING                              ║
║                  (Unit tests with mocked services)                        ║
╚═══════════════════════════════════════════════════════════════════════════╝

func TestMyStreamingFeature(t *testing.T) {
	// Setup test environment
	logger, _ := zap.NewDevelopment()
	ctx := context.Background()
	sessionID := primitive.NewObjectID()
	companyID := "test-company"

	// Create mock services
	mockChat := new(MockChatService)
	mockAI := new(MockAIService)
	mockSink := new(MockOutputSink)

	// Setup AI config
	mockAI.On("GetConfig").Return(&aiservice.AIConfig{MaxToolCalls: 10})

	// Create test event stream
	eventCh := make(chan aiservice.StreamEvent, 2)
	eventCh <- aiservice.StreamEvent{
		Type:    aiservice.StreamEventToken,
		Content: "Test response",
	}
	close(eventCh)

	// Setup mock expectations
	mockAI.On("StreamChatWithTools", ctx, mock.Anything, 10).
		Return((<-chan aiservice.StreamEvent)(eventCh), nil)
	mockSink.On("SendToken", "Test response").Return(nil)
	mockSink.On("IsDisconnected").Return(false)
	mockSink.On("SendDone").Return(nil)
	mockChat.On("SaveMessage", ctx, sessionID, "assistant", "Test response", companyID).
		Return(nil, nil)

	// Create executor config
	config := StreamConfig{
		SessionID:    sessionID,
		CompanyID:    companyID,
		SystemPrompt: "",
		AllowedTools: nil,
		OutputSink:   mockSink,
		Logger:       logger,
	}

	// Create and execute
	exec := NewStreamExecutor(config, mockChat, mockAI)
	fullResponse, err := exec.Execute(ctx, []aiservice.Message{})

	// Assert results
	assert.NoError(t, err)
	assert.Equal(t, "Test response", fullResponse)
	mockChat.AssertExpectations(t)
	mockAI.AssertExpectations(t)
	mockSink.AssertExpectations(t)
}

╔═══════════════════════════════════════════════════════════════════════════╗
║                           MIGRATION CHECKLIST                             ║
╚═══════════════════════════════════════════════════════════════════════════╝

For chat_websocket.go:
□ Import executor package
□ Replace streamAIResponse() body with executor usage
□ Remove manual streaming loop (lines 1793-2061)
□ Keep progress notification goroutine setup
□ Keep interrupt channel setup
□ Preserve existing error handling patterns
□ Test with WebSocket client

For coordinator_tools.go:
□ Import executor package
□ Replace executeSubagentInBackground() streaming section
□ Remove manual streaming loop (lines 3050-3400)
□ Keep session creation logic
□ Keep prompt building logic
□ Keep interrupt handling setup
□ Preserve execution scoring logic (if needed)
□ Test with subchat execution

Common:
□ Run tests: go test ./internal/ai-service/executor/...
□ Verify compilation: go build ./...
□ Check integration tests
□ Monitor metrics in production
□ Verify interrupt handling works
□ Verify disconnect handling works
□ Check database saves are consistent

═══════════════════════════════════════════════════════════════════════════
*/
