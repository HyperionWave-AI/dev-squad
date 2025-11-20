package executor_test

import (
	"context"
	"testing"
	"time"

	aiservice "hyper/internal/ai-service"
	"hyper/internal/ai-service/executor"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// ============================================================================
// Mock Implementations
// ============================================================================

type MockChatService struct {
	mock.Mock
}

func (m *MockChatService) SaveMessage(ctx context.Context, sessionID primitive.ObjectID, role, content, companyID string) (*interface{}, error) {
	args := m.Called(ctx, sessionID, role, content, companyID)
	if args.Get(0) != nil {
		return args.Get(0).(*interface{}), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockChatService) SaveToolCall(ctx context.Context, sessionID primitive.ObjectID, toolCallID, toolName string, args map[string]interface{}, companyID string) (*interface{}, error) {
	callArgs := m.Called(ctx, sessionID, toolCallID, toolName, args, companyID)
	if callArgs.Get(0) != nil {
		return callArgs.Get(0).(*interface{}), callArgs.Error(1)
	}
	return nil, callArgs.Error(1)
}

func (m *MockChatService) SaveToolResult(ctx context.Context, sessionID primitive.ObjectID, toolCallID, toolName, output, errorMsg string, durationMs int64, companyID string) (*interface{}, error) {
	args := m.Called(ctx, sessionID, toolCallID, toolName, output, errorMsg, durationMs, companyID)
	if args.Get(0) != nil {
		return args.Get(0).(*interface{}), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockAIService struct {
	mock.Mock
}

func (m *MockAIService) StreamChatWithTools(ctx context.Context, messages []aiservice.Message, maxToolCalls int) (<-chan aiservice.StreamEvent, error) {
	args := m.Called(ctx, messages, maxToolCalls)
	if args.Get(0) != nil {
		return args.Get(0).(<-chan aiservice.StreamEvent), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAIService) StreamChatWithToolsFiltered(ctx context.Context, messages []aiservice.Message, maxToolCalls int, allowedTools []string) (<-chan aiservice.StreamEvent, error) {
	args := m.Called(ctx, messages, maxToolCalls, allowedTools)
	if args.Get(0) != nil {
		return args.Get(0).(<-chan aiservice.StreamEvent), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAIService) GetConfig() *aiservice.AIConfig {
	args := m.Called()
	return args.Get(0).(*aiservice.AIConfig)
}

type MockOutputSink struct {
	mock.Mock
	tokens      []string
	toolCalls   []string
	toolResults []string
	errors      []string
	doneCalled  bool
	disconnected bool
}

func (m *MockOutputSink) SendToken(content string) error {
	m.tokens = append(m.tokens, content)
	args := m.Called(content)
	return args.Error(0)
}

func (m *MockOutputSink) SendToolCall(toolName, toolID string, args map[string]interface{}) error {
	m.toolCalls = append(m.toolCalls, toolName)
	callArgs := m.Called(toolName, toolID, args)
	return callArgs.Error(0)
}

func (m *MockOutputSink) SendToolResult(toolID, result, errorMsg string, durationMs int) error {
	m.toolResults = append(m.toolResults, result)
	args := m.Called(toolID, result, errorMsg, durationMs)
	return args.Error(0)
}

func (m *MockOutputSink) SendDone() error {
	m.doneCalled = true
	args := m.Called()
	return args.Error(0)
}

func (m *MockOutputSink) SendError(errorMsg string) error {
	m.errors = append(m.errors, errorMsg)
	args := m.Called(errorMsg)
	return args.Error(0)
}

func (m *MockOutputSink) IsDisconnected() bool {
	args := m.Called()
	return args.Bool(0)
}

// ============================================================================
// Tests
// ============================================================================

func TestStreamExecutor_BasicTokenStreaming(t *testing.T) {
	// Setup
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

	// Create event channel with test data
	eventCh := make(chan aiservice.StreamEvent, 3)
	eventCh <- aiservice.StreamEvent{
		Type:    aiservice.StreamEventToken,
		Content: "Hello ",
	}
	eventCh <- aiservice.StreamEvent{
		Type:    aiservice.StreamEventToken,
		Content: "world!",
	}
	close(eventCh)

	// Setup expectations
	mockAI.On("StreamChatWithTools", ctx, mock.Anything, 10).Return((<-chan aiservice.StreamEvent)(eventCh), nil)
	mockSink.On("SendToken", "Hello ").Return(nil)
	mockSink.On("SendToken", "world!").Return(nil)
	mockSink.On("IsDisconnected").Return(false)
	mockSink.On("SendDone").Return(nil)
	mockChat.On("SaveMessage", ctx, sessionID, "assistant", "Hello world!", companyID).Return(nil, nil)

	// Create executor
	config := executor.StreamConfig{
		SessionID:    sessionID,
		CompanyID:    companyID,
		SystemPrompt: "",
		AllowedTools: nil,
		OutputSink:   mockSink,
		Logger:       logger,
	}
	exec := executor.NewStreamExecutor(config, mockChat, mockAI)

	// Execute
	fullResponse, err := exec.Execute(ctx, []aiservice.Message{})

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, "Hello world!", fullResponse)
	assert.Equal(t, 2, len(mockSink.tokens))
	assert.True(t, mockSink.doneCalled)
	mockChat.AssertExpectations(t)
	mockAI.AssertExpectations(t)
	mockSink.AssertExpectations(t)
}

func TestStreamExecutor_ToolCallAndResult(t *testing.T) {
	// Setup
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

	// Create event channel with test data
	eventCh := make(chan aiservice.StreamEvent, 5)
	eventCh <- aiservice.StreamEvent{
		Type:    aiservice.StreamEventToken,
		Content: "Let me check that...",
	}
	eventCh <- aiservice.StreamEvent{
		Type: aiservice.StreamEventToolCall,
		ToolCall: &aiservice.ToolCall{
			ID:   "call-123",
			Name: "read_file",
			Args: map[string]interface{}{"path": "/test/file.txt"},
		},
	}
	eventCh <- aiservice.StreamEvent{
		Type: aiservice.StreamEventToolResult,
		ToolResult: &aiservice.ToolResult{
			ID:         "call-123",
			Name:       "read_file",
			Output:     "File contents here",
			DurationMs: 50,
		},
	}
	eventCh <- aiservice.StreamEvent{
		Type:    aiservice.StreamEventToken,
		Content: "Done!",
	}
	close(eventCh)

	// Setup expectations
	mockAI.On("StreamChatWithTools", ctx, mock.Anything, 10).Return((<-chan aiservice.StreamEvent)(eventCh), nil)
	mockSink.On("SendToken", "Let me check that...").Return(nil)
	mockSink.On("SendToolCall", "read_file", "call-123", mock.Anything).Return(nil)
	mockSink.On("SendToolResult", "call-123", "File contents here", "", 50).Return(nil)
	mockSink.On("SendToken", "Done!").Return(nil)
	mockSink.On("IsDisconnected").Return(false)
	mockSink.On("SendDone").Return(nil)

	// Expect database saves
	mockChat.On("SaveMessage", ctx, sessionID, "assistant", "Let me check that...", companyID).Return(nil, nil)
	mockChat.On("SaveToolCall", ctx, sessionID, "call-123", "read_file", mock.Anything, companyID).Return(nil, nil)
	mockChat.On("SaveToolResult", ctx, sessionID, "call-123", "read_file", "File contents here", "", int64(50), companyID).Return(nil, nil)
	mockChat.On("SaveMessage", ctx, sessionID, "assistant", "Done!", companyID).Return(nil, nil)

	// Create executor
	config := executor.StreamConfig{
		SessionID:    sessionID,
		CompanyID:    companyID,
		SystemPrompt: "",
		AllowedTools: nil,
		OutputSink:   mockSink,
		Logger:       logger,
	}
	exec := executor.NewStreamExecutor(config, mockChat, mockAI)

	// Execute
	fullResponse, err := exec.Execute(ctx, []aiservice.Message{})

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, "Done!", fullResponse) // Only final text after tool call
	assert.Equal(t, 2, len(mockSink.tokens))
	assert.Equal(t, 1, len(mockSink.toolCalls))
	assert.Equal(t, 1, len(mockSink.toolResults))
	assert.True(t, mockSink.doneCalled)
	mockChat.AssertExpectations(t)
	mockAI.AssertExpectations(t)
	mockSink.AssertExpectations(t)
}

func TestStreamExecutor_FilteredTools(t *testing.T) {
	// Setup
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

	// Create event channel
	eventCh := make(chan aiservice.StreamEvent, 1)
	eventCh <- aiservice.StreamEvent{
		Type:    aiservice.StreamEventToken,
		Content: "Working...",
	}
	close(eventCh)

	// Define allowed tools
	allowedTools := []string{"read_file", "write_file"}

	// Setup expectations - should call filtered streaming
	mockAI.On("StreamChatWithToolsFiltered", ctx, mock.Anything, 10, allowedTools).Return((<-chan aiservice.StreamEvent)(eventCh), nil)
	mockSink.On("SendToken", "Working...").Return(nil)
	mockSink.On("IsDisconnected").Return(false)
	mockSink.On("SendDone").Return(nil)
	mockChat.On("SaveMessage", ctx, sessionID, "assistant", "Working...", companyID).Return(nil, nil)

	// Create executor with filtered tools
	config := executor.StreamConfig{
		SessionID:    sessionID,
		CompanyID:    companyID,
		SystemPrompt: "",
		AllowedTools: allowedTools,
		OutputSink:   mockSink,
		Logger:       logger,
	}
	exec := executor.NewStreamExecutor(config, mockChat, mockAI)

	// Execute
	fullResponse, err := exec.Execute(ctx, []aiservice.Message{})

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, "Working...", fullResponse)
	mockChat.AssertExpectations(t)
	mockAI.AssertExpectations(t)
	mockSink.AssertExpectations(t)
}

func TestStreamExecutor_ClientDisconnect(t *testing.T) {
	// Setup
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

	// Create event channel
	eventCh := make(chan aiservice.StreamEvent, 2)
	eventCh <- aiservice.StreamEvent{
		Type:    aiservice.StreamEventToken,
		Content: "First ",
	}
	eventCh <- aiservice.StreamEvent{
		Type:    aiservice.StreamEventToken,
		Content: "Second",
	}
	close(eventCh)

	// Setup expectations - client disconnects after first token
	mockAI.On("StreamChatWithTools", ctx, mock.Anything, 10).Return((<-chan aiservice.StreamEvent)(eventCh), nil)
	mockSink.On("SendToken", "First ").Return(nil)
	mockSink.On("IsDisconnected").Return(false).Once()    // First token succeeds
	mockSink.On("IsDisconnected").Return(true).Times(999) // Then disconnected
	// No SendDone expected because client is disconnected
	mockChat.On("SaveMessage", ctx, sessionID, "assistant", "First Second", companyID).Return(nil, nil)

	// Create executor
	config := executor.StreamConfig{
		SessionID:    sessionID,
		CompanyID:    companyID,
		SystemPrompt: "",
		AllowedTools: nil,
		OutputSink:   mockSink,
		Logger:       logger,
	}
	exec := executor.NewStreamExecutor(config, mockChat, mockAI)

	// Execute
	fullResponse, err := exec.Execute(ctx, []aiservice.Message{})

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, "First Second", fullResponse)
	// Message should still be saved even though client disconnected
	mockChat.AssertExpectations(t)
	mockAI.AssertExpectations(t)
}

func TestStreamExecutor_InterruptDetection(t *testing.T) {
	// Setup
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

	// Create interrupt channel
	interruptCh := make(chan struct{}, 1)

	// Create event channel (will be interrupted)
	eventCh := make(chan aiservice.StreamEvent, 2)
	eventCh <- aiservice.StreamEvent{
		Type:    aiservice.StreamEventToken,
		Content: "Starting...",
	}
	// Simulate slow streaming to allow interrupt
	go func() {
		time.Sleep(50 * time.Millisecond)
		eventCh <- aiservice.StreamEvent{
			Type:    aiservice.StreamEventToken,
			Content: "Should not reach",
		}
		close(eventCh)
	}()

	// Setup expectations
	mockAI.On("StreamChatWithTools", ctx, mock.Anything, 10).Return((<-chan aiservice.StreamEvent)(eventCh), nil)
	mockSink.On("SendToken", "Starting...").Return(nil)
	mockSink.On("SendToken", "\n\n⏸️ _Interrupt detected - processing your message..._\n\n").Return(nil)
	mockSink.On("IsDisconnected").Return(false)
	mockChat.On("SaveMessage", ctx, sessionID, "assistant", "Starting...", companyID).Return(nil, nil)

	// Create executor with interrupt channel
	config := executor.StreamConfig{
		SessionID:    sessionID,
		CompanyID:    companyID,
		SystemPrompt: "",
		AllowedTools: nil,
		OutputSink:   mockSink,
		InterruptCh:  interruptCh,
		Logger:       logger,
	}
	exec := executor.NewStreamExecutor(config, mockChat, mockAI)

	// Trigger interrupt after first token
	go func() {
		time.Sleep(25 * time.Millisecond)
		interruptCh <- struct{}{}
	}()

	// Execute
	fullResponse, err := exec.Execute(ctx, []aiservice.Message{})

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, "Starting...", fullResponse)
	mockChat.AssertExpectations(t)
	mockAI.AssertExpectations(t)
	mockSink.AssertExpectations(t)
}

func TestStreamExecutor_CustomCompletion(t *testing.T) {
	// Setup
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

	// Create event channel (3 tokens, but validator stops after 2)
	eventCh := make(chan aiservice.StreamEvent, 3)
	eventCh <- aiservice.StreamEvent{Type: aiservice.StreamEventToken, Content: "One "}
	eventCh <- aiservice.StreamEvent{Type: aiservice.StreamEventToken, Content: "Two "}
	eventCh <- aiservice.StreamEvent{Type: aiservice.StreamEventToken, Content: "Three"}
	close(eventCh)

	// Custom completion validator - stop after 2 tokens
	tokensSeen := 0
	validator := func(fullResponse string, toolCallCount int) bool {
		tokensSeen++
		return tokensSeen >= 2
	}

	// Setup expectations
	mockAI.On("StreamChatWithTools", ctx, mock.Anything, 10).Return((<-chan aiservice.StreamEvent)(eventCh), nil)
	mockSink.On("SendToken", "One ").Return(nil)
	mockSink.On("SendToken", "Two ").Return(nil)
	mockSink.On("IsDisconnected").Return(false)
	mockSink.On("SendDone").Return(nil)
	mockChat.On("SaveMessage", ctx, sessionID, "assistant", "One Two ", companyID).Return(nil, nil)

	// Create executor with custom validator
	config := executor.StreamConfig{
		SessionID:           sessionID,
		CompanyID:           companyID,
		SystemPrompt:        "",
		AllowedTools:        nil,
		OutputSink:          mockSink,
		CompletionValidator: validator,
		Logger:              logger,
	}
	exec := executor.NewStreamExecutor(config, mockChat, mockAI)

	// Execute
	fullResponse, err := exec.Execute(ctx, []aiservice.Message{})

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, "One Two ", fullResponse) // Only 2 tokens, not 3
	mockChat.AssertExpectations(t)
	mockAI.AssertExpectations(t)
	mockSink.AssertExpectations(t)
}
