package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	aiservice "hyper/internal/ai-service"
	"hyper/internal/ai-service/executor"
	"hyper/internal/ai-service/tools"
	"hyper/internal/config"
	"hyper/internal/mcp/storage"
	"hyper/internal/metrics"
	"hyper/internal/middleware"
	"hyper/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// chatServiceAdapter adapts ChatServiceInterface to executor.ChatServiceInterface.
// This is needed because the actual service returns *models.ChatMessage but executor expects *interface{}.
type chatServiceAdapter struct {
	service ChatServiceInterface
}

func (a *chatServiceAdapter) SaveMessage(ctx context.Context, sessionID primitive.ObjectID, role, content, companyID string) (*interface{}, error) {
	msg, err := a.service.SaveMessage(ctx, sessionID, role, content, companyID)
	if err != nil {
		return nil, err
	}
	var result interface{} = msg
	return &result, nil
}

func (a *chatServiceAdapter) SaveToolCall(ctx context.Context, sessionID primitive.ObjectID, toolCallID, toolName string, args map[string]interface{}, companyID string) (*interface{}, error) {
	msg, err := a.service.SaveToolCall(ctx, sessionID, toolCallID, toolName, args, companyID)
	if err != nil {
		return nil, err
	}
	var result interface{} = msg
	return &result, nil
}

func (a *chatServiceAdapter) SaveToolResult(ctx context.Context, sessionID primitive.ObjectID, toolCallID, toolName, output, errorMsg string, durationMs int64, companyID string) (*interface{}, error) {
	msg, err := a.service.SaveToolResult(ctx, sessionID, toolCallID, toolName, output, errorMsg, durationMs, companyID)
	if err != nil {
		return nil, err
	}
	var result interface{} = msg
	return &result, nil
}

// SECURITY PHASE 3: Redundant session ownership validation
// validateSessionOwnership performs a belt-and-suspenders validation of session ownership
// This is called AFTER the initial validation in HandleChatWebSocket for defense in depth
// Returns error if validation fails (session not found, ownership mismatch, or company mismatch)
func (h *ChatWebSocketHandler) validateSessionOwnership(ctx context.Context, sessionID primitive.ObjectID, userID, companyID string) error {
	// Fresh database query (not from cache) to verify session still exists and belongs to user
	session, err := h.chatService.GetSession(ctx, sessionID, companyID)
	if err != nil {
		// SECURITY: Log detailed error server-side, return generic error
		h.logger.Warn("SECURITY: Session ownership validation failed - session fetch error",
			zap.String("sessionId", sessionID.Hex()),
			zap.String("userId", userID),
			zap.Error(err))
		return fmt.Errorf("session validation failed")
	}

	// Verify user still owns this session
	if session.UserID != userID {
		// SECURITY: Potential unauthorized access - log as warning
		h.logger.Warn("SECURITY: Session ownership validation failed - user mismatch",
			zap.String("sessionId", sessionID.Hex()),
			zap.String("requestUserId", userID),
			zap.String("sessionOwnerId", session.UserID))
		metrics.WebSocketOwnershipViolations.Inc()
		return fmt.Errorf("session ownership mismatch")
	}

	// Verify company still matches
	if session.CompanyID != companyID {
		// SECURITY: Cross-company access attempt - log as warning
		h.logger.Warn("SECURITY: Session ownership validation failed - company mismatch",
			zap.String("sessionId", sessionID.Hex()),
			zap.String("requestCompanyId", companyID),
			zap.String("sessionCompanyId", session.CompanyID))
		metrics.WebSocketOwnershipViolations.Inc()
		return fmt.Errorf("session company mismatch")
	}

	return nil
}

// WebSocket upgrader configuration (Production Hardened)
var upgrader = websocket.Upgrader{
	ReadBufferSize:  8192,  // 8KB - efficient for large messages
	WriteBufferSize: 32768, // 32KB - handles large streaming AI responses (increased to prevent broken pipe)
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			// No origin header (non-browser clients like curl) - allow for dev/testing
			return true
		}

		// Read allowed origins from environment (same as CORS config)
		corsOriginsEnv := r.Context().Value("allowedOrigins")
		if corsOriginsEnv != nil {
			if allowedOrigins, ok := corsOriginsEnv.([]string); ok {
				for _, allowed := range allowedOrigins {
					if origin == allowed {
						return true
					}
				}
			}
		}

		// Fallback: check environment variable directly
		allowedOriginsStr := os.Getenv("CORS_ALLOWED_ORIGINS")
		if allowedOriginsStr != "" {
			origins := strings.Split(allowedOriginsStr, ",")
			for _, allowed := range origins {
				if strings.TrimSpace(allowed) == origin {
					return true
				}
			}
		}

		// Log rejected origin for security monitoring
		// Note: This will log to stdout since we don't have logger in this context
		fmt.Printf("[WebSocket CORS] Rejected origin: %q (allowed: %q)\n", origin, allowedOriginsStr)
		return false
	},
}

// PHASE 1 Backpressure: Write timeout constants
const (
	// WriteTimeout is the maximum time to wait for a WebSocket write to complete
	// If exceeded, the write fails and the client is considered slow
	WriteTimeout = 10 * time.Second

	// SlowWriteThreshold is the duration above which a write is logged as "slow"
	SlowWriteThreshold = 1 * time.Second
)

// SECURITY: Generic error messages to prevent information leakage
// These constants ensure consistent, non-revealing error responses
const (
	// errInvalidRequest is returned for all request validation failures
	// Intentionally vague to prevent session ID enumeration attacks
	errInvalidRequest = "Invalid request"

	// errUnauthorized is returned for authentication failures
	errUnauthorized = "Unauthorized"

	// errServiceUnavailable is returned for capacity issues
	errServiceUnavailable = "Service temporarily unavailable"

	// errTooManyRequests is returned for rate limiting
	errTooManyRequests = "Too many requests"
)

// Production Rate Limiting & Connection Management
var (
	// Global connection counter (atomic for thread-safety)
	activeConnections int64
	maxConnections    int64 = 1000 // Configurable via env: WS_MAX_CONNECTIONS

	// Per-user connection tracking
	// Note: Set to 5 to allow for transient connection overlaps during:
	// - React StrictMode double-mounts in development
	// - Session switching (old connection closing while new one opens)
	// - Browser tab refresh (old connections may linger briefly)
	userConnections       = sync.Map{} // map[userID]int
	maxConnectionsPerUser = 5          // Max 5 concurrent connections per user

	// Per-user message rate limiting
	userMessageRates = sync.Map{} // map[userID]*messageRateLimit
)

// messageRateLimit tracks message rate per user
type messageRateLimit struct {
	lastMessage  time.Time
	messageCount int
	mu           sync.Mutex
}

// checkRateLimit returns true if user is within rate limits
func checkRateLimit(userID string) bool {
	// Load or create rate limit entry
	val, _ := userMessageRates.LoadOrStore(userID, &messageRateLimit{
		lastMessage:  time.Now(),
		messageCount: 0,
	})
	rateLimit := val.(*messageRateLimit)

	rateLimit.mu.Lock()
	defer rateLimit.mu.Unlock()

	now := time.Now()
	// Reset counter every minute
	if now.Sub(rateLimit.lastMessage) > time.Minute {
		rateLimit.messageCount = 0
		rateLimit.lastMessage = now
	}

	// Limit: 10 messages per minute per user
	if rateLimit.messageCount >= 10 {
		return false // Rate limit exceeded
	}

	rateLimit.messageCount++
	return true
}

// incrementUserConnection increments connection count for user
func incrementUserConnection(userID string) bool {
	val, _ := userConnections.LoadOrStore(userID, new(int32))
	count := val.(*int32)

	// Check current count BEFORE incrementing to avoid leak
	currentCount := atomic.LoadInt32(count)
	if int(currentCount) >= maxConnectionsPerUser {
		return false // Already at limit, don't increment
	}

	// Increment and verify we're still under limit
	// (race condition: another goroutine might have incremented meanwhile)
	newCount := atomic.AddInt32(count, 1)
	if int(newCount) > maxConnectionsPerUser {
		// Rollback - we went over limit due to race
		atomic.AddInt32(count, -1)
		return false
	}

	return true
}

// decrementUserConnection decrements connection count for user
func decrementUserConnection(userID string) {
	val, exists := userConnections.Load(userID)
	if exists {
		count := val.(*int32)
		atomic.AddInt32(count, -1)
	}
}

// ChatServiceInterface defines the interface for chat service operations
type ChatServiceInterface interface {
	GetSession(ctx context.Context, sessionID primitive.ObjectID, companyID string) (*models.ChatSession, error)
	GetSessionMessages(ctx context.Context, sessionID primitive.ObjectID) ([]models.ChatMessage, error)
	SaveMessage(ctx context.Context, sessionID primitive.ObjectID, role, content, companyID string) (*models.ChatMessage, error)
	SaveToolCall(ctx context.Context, sessionID primitive.ObjectID, id, name string, args map[string]interface{}, companyID string) (*models.ChatMessage, error)
	SaveToolResult(ctx context.Context, sessionID primitive.ObjectID, id, name string, output interface{}, errorMsg string, durationMs int64, companyID string) (*models.ChatMessage, error)
	ArchiveMessages(ctx context.Context, sessionID primitive.ObjectID, messageIDs []primitive.ObjectID) error
}

// AIServiceInterface defines the interface for AI service operations
type AIServiceInterface interface {
	StreamChatWithTools(ctx context.Context, messages []aiservice.Message, maxToolCalls int) (<-chan aiservice.StreamEvent, error)
	StreamChatWithToolsFiltered(ctx context.Context, messages []aiservice.Message, maxToolCalls int, allowedToolNames []string) (<-chan aiservice.StreamEvent, error)
	GetConfig() *aiservice.AIConfig
	GetAllowedToolsForDirectSubagent() []string
}

// AISettingsServiceInterface defines the interface for AI settings service operations
type AISettingsServiceInterface interface {
	GetSubagent(ctx context.Context, id primitive.ObjectID, companyID string) (*models.Subagent, error)
	GetSystemPrompt(ctx context.Context, userID, companyID string) (string, error)
}

// ChatWebSocketHandler handles WebSocket connections for real-time chat streaming
type ChatWebSocketHandler struct {
	chatService            ChatServiceInterface
	aiService              AIServiceInterface
	aiSettingsService      AISettingsServiceInterface
	compactionOrchestrator *CompactionOrchestrator
	subchatStorage         SubchatStorageInterface
	logger                 *zap.Logger
	toolResultProcessor    executor.ToolResultProcessorFunc
	resultInterceptor      *ToolResultInterceptor // Token-based result deflection
	writeMutex             sync.Mutex             // Protects concurrent WebSocket writes (ping + message streaming)
}

// SubchatStorageInterface defines the interface for subchat storage operations (system subagents)
type SubchatStorageInterface interface {
	GetSubagent(name string) (*storage.Subagent, error)
}

// NewChatWebSocketHandler creates a new WebSocket handler with ai-service integration
func NewChatWebSocketHandler(chatService ChatServiceInterface, aiService AIServiceInterface, aiSettingsService AISettingsServiceInterface, subchatStorage SubchatStorageInterface, logger *zap.Logger) *ChatWebSocketHandler {
	// Create default tool result processor
	defaultProcessor := func(toolName string, output interface{}) (string, bool, bool) {
		// Default: stream and save all results
		outputStr := fmt.Sprintf("%v", output)
		return outputStr, true, true
	}

	// Initialize token-based result interceptor for intelligent deflection
	resultInterceptor := NewToolResultInterceptor(logger)

	// Initialize compaction orchestrator for adaptive context management
	compactionOrchestrator := NewCompactionOrchestrator(
		DefaultCompactionConfig(),
		aiService,
		chatService,
		logger,
	)

	return &ChatWebSocketHandler{
		chatService:            chatService,
		compactionOrchestrator: compactionOrchestrator,
		aiService:              aiService,
		aiSettingsService:      aiSettingsService,
		subchatStorage:         subchatStorage,
		logger:                 logger,
		toolResultProcessor:    defaultProcessor,
		resultInterceptor:      resultInterceptor,
	}
}

// safeWriteJSON safely writes JSON to WebSocket with mutex protection and timeout
// PHASE 1 Backpressure: Added write deadline to prevent indefinite blocking on slow clients
// Prevents race condition between ping goroutine and message streaming goroutine
func (h *ChatWebSocketHandler) safeWriteJSON(conn *websocket.Conn, msg interface{}) error {
	// Call the monitored version without session ID (no buffer tracking)
	return h.safeWriteJSONWithMonitoring(conn, msg, "")
}

// ErrCircuitOpen is returned when the circuit breaker is open and blocking requests
var ErrCircuitOpen = fmt.Errorf("circuit breaker open: client too slow")

// safeWriteJSONWithMonitoring safely writes JSON to WebSocket with optional buffer monitoring
// PHASE 3 Buffer Monitoring: When sessionID is provided, records write metrics for buffer estimation
// PHASE 5 Circuit Breaker: Integrates circuit breaker to protect against slow/stuck clients
func (h *ChatWebSocketHandler) safeWriteJSONWithMonitoring(conn *websocket.Conn, msg interface{}, sessionID string) error {
	// PHASE 5: Check circuit breaker before attempting write
	var circuitBreaker *CircuitBreaker
	if sessionID != "" {
		registry := GetCircuitBreakerRegistry(h.logger)
		circuitBreaker = registry.Get(sessionID)

		if !circuitBreaker.AllowRequest() {
			h.logger.Debug("Circuit breaker open - blocking write",
				zap.String("sessionId", sessionID),
				zap.String("state", circuitBreaker.State().String()))
			metrics.WebSocketCircuitBreakerTrips.Inc()
			return ErrCircuitOpen
		}
	}

	h.writeMutex.Lock()
	defer h.writeMutex.Unlock()

	// PHASE 3: Get health monitor for buffer tracking (if session ID provided)
	var monitor *ConnectionHealthMonitor
	if sessionID != "" {
		healthPool := GetHealthMonitorPool(h.logger)
		if val, ok := healthPool.monitors.Load(sessionID); ok {
			monitor = val.(*ConnectionHealthMonitor)
			monitor.RecordWriteStart()
		}
	}

	// PHASE 1: Set write deadline to prevent indefinite blocking on slow clients
	if err := conn.SetWriteDeadline(time.Now().Add(WriteTimeout)); err != nil {
		h.logger.Warn("Failed to set write deadline", zap.Error(err))
		// Continue anyway - deadline failure shouldn't block writes
	}
	defer conn.SetWriteDeadline(time.Time{}) // Clear deadline after write

	start := time.Now()
	err := conn.WriteJSON(msg)
	duration := time.Since(start)

	// PHASE 3: Record write end for buffer monitoring
	if monitor != nil {
		monitor.RecordWriteEnd(duration)
	}

	// Record write latency metric
	metrics.WebSocketWriteLatency.Observe(duration.Seconds())

	// PHASE 5: Record result in circuit breaker
	if circuitBreaker != nil {
		if err != nil {
			if isTimeoutError(err) {
				circuitBreaker.RecordTimeout()
			} else {
				circuitBreaker.RecordFailure(err)
			}
		} else if duration > SlowWriteThreshold {
			// Slow but successful - track for slow call rate
			circuitBreaker.RecordSlowCall(duration)
		} else {
			circuitBreaker.RecordSuccess(duration)
		}
	}

	if err == nil {
		metrics.WebSocketMessagesSent.Inc()
		// Log slow writes for monitoring
		if duration > SlowWriteThreshold {
			metrics.WebSocketSlowWrites.Inc()
			h.logger.Warn("Slow WebSocket write detected",
				zap.Duration("duration", duration),
				zap.Duration("threshold", SlowWriteThreshold))
		}
	} else if isTimeoutError(err) {
		// PHASE 1: Track write timeouts separately
		metrics.WebSocketWriteTimeouts.Inc()
		h.logger.Warn("WebSocket write timeout - client too slow",
			zap.Duration("timeout", WriteTimeout),
			zap.Error(err))
	}

	return err
}

// isTimeoutError checks if an error is a timeout error (net.Error with Timeout())
// PHASE 1 Backpressure: Helper to identify write deadline violations
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	netErr, ok := err.(net.Error)
	return ok && netErr.Timeout()
}

// safeWriteControl safely writes control frame to WebSocket with mutex protection
// Prevents race condition between ping goroutine and message streaming goroutine
func (h *ChatWebSocketHandler) safeWriteControl(conn *websocket.Conn, messageType int, data []byte, deadline time.Time) error {
	h.writeMutex.Lock()
	defer h.writeMutex.Unlock()
	return conn.WriteControl(messageType, data, deadline)
}

// sendSystemNotification sends a system notification to the WebSocket client
// Used for compaction, deflection, and summarization events
func (h *ChatWebSocketHandler) sendSystemNotification(conn *websocket.Conn, notification models.SystemNotification) {
	if conn == nil {
		return
	}
	msg := models.StreamMessage{
		Type:         "system_notification",
		Notification: &notification,
	}
	if err := h.safeWriteJSON(conn, msg); err != nil {
		h.logger.Debug("Failed to send system notification (client may have disconnected)",
			zap.String("category", notification.Category),
			zap.Error(err))
	}
}

// extractAuthFromContext extracts authentication from Gin context (set by JWT middleware)
// Falls back to query parameters for backward compatibility
// GET /api/v1/chat/stream?sessionId=xxx
func (h *ChatWebSocketHandler) extractAuthFromContext(c *gin.Context) (string, string, error) {
	// First try to get from context (set by OptionalJWTMiddleware)
	if userIDVal, exists := c.Get("userId"); exists {
		if companyIDVal, exists := c.Get("companyId"); exists {
			userID, ok1 := userIDVal.(string)
			companyID, ok2 := companyIDVal.(string)
			if ok1 && ok2 && userID != "" && companyID != "" {
				return userID, companyID, nil
			}
		}
	}

	// Fallback to query parameters for backward compatibility
	userID := c.Query("userId")
	companyID := c.Query("companyId")

	if userID == "" || companyID == "" {
		return "", "", fmt.Errorf("missing authentication parameters")
	}

	return userID, companyID, nil
}

// HandleChatWebSocket handles WebSocket connections for chat streaming
// GET /api/v1/chat/stream?sessionId=xxx
func (h *ChatWebSocketHandler) HandleChatWebSocket(c *gin.Context) {
	// Extract authentication from context (set by middleware)
	userID, companyID, err := h.extractAuthFromContext(c)
	if err != nil {
		// SECURITY: Log detailed error server-side, return generic error to client
		h.logger.Warn("WebSocket auth failed",
			zap.Error(err),
			zap.String("remoteAddr", c.ClientIP()))
		c.JSON(http.StatusUnauthorized, gin.H{"error": errUnauthorized})
		return
	}

	// Get session ID from query
	// SECURITY: All session validation failures return the same generic error
	// to prevent session ID enumeration attacks
	sessionIDStr := c.Query("sessionId")
	if sessionIDStr == "" {
		h.logger.Debug("WebSocket request missing sessionId",
			zap.String("userId", userID),
			zap.String("remoteAddr", c.ClientIP()))
		c.JSON(http.StatusBadRequest, gin.H{"error": errInvalidRequest})
		return
	}

	sessionID, err := primitive.ObjectIDFromHex(sessionIDStr)
	if err != nil {
		// SECURITY: Don't reveal that the format was invalid
		h.logger.Debug("WebSocket request with invalid sessionId format",
			zap.String("sessionIdStr", sessionIDStr),
			zap.String("userId", userID),
			zap.String("remoteAddr", c.ClientIP()))
		c.JSON(http.StatusBadRequest, gin.H{"error": errInvalidRequest})
		return
	}

	// Verify session exists and user has access
	session, err := h.chatService.GetSession(c.Request.Context(), sessionID, companyID)
	if err != nil {
		// SECURITY: Don't reveal whether session exists or access denied
		h.logger.Debug("WebSocket session access failed",
			zap.String("sessionId", sessionID.Hex()),
			zap.String("userId", userID),
			zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": errInvalidRequest})
		return
	}

	// Verify session belongs to user
	if session.UserID != userID {
		// SECURITY: Log potential unauthorized access attempt, return generic error
		h.logger.Warn("WebSocket session ownership mismatch - potential unauthorized access",
			zap.String("sessionId", sessionID.Hex()),
			zap.String("requestUserId", userID),
			zap.String("sessionOwnerId", session.UserID),
			zap.String("remoteAddr", c.ClientIP()))
		c.JSON(http.StatusBadRequest, gin.H{"error": errInvalidRequest})
		return
	}

	// Production Rate Limiting: Check global connection limit
	currentConnections := atomic.LoadInt64(&activeConnections)
	if currentConnections >= maxConnections {
		h.logger.Warn("Connection rejected - global limit reached",
			zap.Int64("activeConnections", currentConnections),
			zap.Int64("maxConnections", maxConnections))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": errServiceUnavailable})
		return
	}

	// Production Rate Limiting: Check per-user connection limit
	if !incrementUserConnection(userID) {
		h.logger.Warn("Connection rejected - user limit reached",
			zap.String("userId", userID),
			zap.Int("maxPerUser", maxConnectionsPerUser))
		c.JSON(http.StatusTooManyRequests, gin.H{"error": errTooManyRequests})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade to WebSocket", zap.Error(err))
		decrementUserConnection(userID) // Rollback on failure
		return
	}
	defer conn.Close()

	// Track connection counts
	atomic.AddInt64(&activeConnections, 1)
	defer func() {
		atomic.AddInt64(&activeConnections, -1)
		decrementUserConnection(userID)
	}()

	// Handle pointer field for logging
	activeSubagentName := ""
	if session.ActiveSubagentName != nil {
		activeSubagentName = *session.ActiveSubagentName
	}
	h.logger.Info("🔌 WebSocket CONNECTED",
		zap.String("sessionId", sessionID.Hex()),
		zap.String("userId", userID),
		zap.Int64("totalConnections", atomic.LoadInt64(&activeConnections)),
		zap.String("sessionTitle", session.Title),
		zap.Bool("hasActiveSubagent", session.ActiveSubagentID != nil || (session.ActiveSubagentName != nil && *session.ActiveSubagentName != "")),
		zap.String("activeSubagentName", activeSubagentName))

	// Register WebSocket connection for broadcasting (e.g., session_created events)
	broadcaster := GetWebSocketBroadcaster(h.logger)
	broadcaster.RegisterConnection(sessionID, conn, &h.writeMutex)
	defer broadcaster.UnregisterConnection(sessionID)

	// Record WebSocket connection metrics
	connectionStart := time.Now()
	metrics.RecordWebSocketConnection()
	defer func() {
		metrics.RecordWebSocketDisconnection()
		metrics.WebSocketConnectionDuration.Observe(time.Since(connectionStart).Seconds())
	}()

	// PHASE 1 Context Lifecycle: Get HTTP context first (parent of all contexts)
	httpCtx := c.Request.Context()

	// PHASE 1: Create AI context as CHILD of HTTP context
	// This ensures AI processing cancels when:
	// 1. HTTP connection closes (client disconnect) - cancels immediately
	// 2. 10-minute timeout expires - whichever happens first
	// Previously this was context.Background() which didn't cancel on HTTP disconnect!
	aiCtx, aiCancel := context.WithTimeout(httpCtx, 10*time.Minute)
	defer aiCancel()

	h.logger.Debug("Context hierarchy established",
		zap.String("sessionId", sessionID.Hex()),
		zap.String("hierarchy", "HTTP -> AI (10min timeout)"))

	// Pass both contexts to handleMessages (httpCtx still needed for explicit checks)
	h.handleMessages(aiCtx, httpCtx, conn, sessionID, userID, companyID)
}

// StreamCleanup manages channel lifecycle and goroutine coordination
type StreamCleanup struct {
	doneOnce   sync.Once
	done       chan struct{}
	wg         sync.WaitGroup
	streamCtx  context.Context
	cancelFunc context.CancelFunc
}

// Close safely closes the done channel and waits for all goroutines
func (sc *StreamCleanup) Close() {
	sc.doneOnce.Do(func() {
		close(sc.done)
		sc.cancelFunc()
		sc.wg.Wait() // Block until all goroutines exit
	})
}

// handleMessages manages the WebSocket message loop with processing state
func (h *ChatWebSocketHandler) handleMessages(aiCtx context.Context, httpCtx context.Context, conn *websocket.Conn, sessionID primitive.ObjectID, userID, companyID string) {
	// SECURITY PHASE 3: Redundant session ownership validation (belt and suspenders)
	// This is called AFTER the initial validation in HandleChatWebSocket for defense in depth
	// Even though we validated above, validate again before starting message processing
	if err := h.validateSessionOwnership(httpCtx, sessionID, userID, companyID); err != nil {
		h.logger.Warn("SECURITY: Redundant session validation failed - closing connection",
			zap.String("sessionId", sessionID.Hex()),
			zap.String("userId", userID),
			zap.Error(err))
		// Send error to client and close (using generic error for security)
		errMsg := models.StreamMessage{
			Type:  "error",
			Error: errInvalidRequest,
		}
		h.safeWriteJSON(conn, errMsg)
		return
	}

	// Set read deadline for ping/pong (5 minutes to allow users time to review responses)
	conn.SetReadDeadline(time.Now().Add(30 * time.Second)) // Reduced from 5 minutes to 30 seconds
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(30 * time.Second)) // Reduced from 5 minutes to 30 seconds
		// Record pong received in health monitor
		healthPool := GetHealthMonitorPool(h.logger)
		if monitor, ok := healthPool.monitors.Load(sessionID.Hex()); ok {
			monitor.(*ConnectionHealthMonitor).RecordPongReceived()
		}
		return nil
	})
	ticker := time.NewTicker(30 * time.Second)

	// PHASE 2 Context Lifecycle: Create stream context as CHILD of AI context
	// Context hierarchy: HTTP -> AI (10min timeout) -> Stream
	// This ensures stream cleanup cancels when AI or HTTP cancels
	// Previously this was context.Background() which wasn't tied to HTTP lifecycle!
	streamCtx, streamCancel := context.WithCancel(aiCtx)
	cleanup := &StreamCleanup{
		done:       make(chan struct{}),
		streamCtx:  streamCtx,
		cancelFunc: streamCancel,
	}

	// Ordered defer chain (LIFO execution):
	// Note: WebSocket close is handled by parent function's defer
	// 1. Stop ticker (after goroutines exit)
	defer ticker.Stop()
	// 2. Wait for goroutines and close done channel
	defer cleanup.Close()
	// 3. Signal all goroutines to exit
	defer func() {
		// Ensure cleanup on panic
		if r := recover(); r != nil {
			h.logger.Error("Panic in handleMessages",
				zap.String("sessionId", sessionID.Hex()),
				zap.Any("panic", r))
			panic(r) // Re-panic after cleanup
		}
	}()

	// Processing state to prevent concurrent messages during AI response (using atomic for panic safety)
	var isProcessing atomic.Bool

	// Cancel function for current AI execution (allows stop button to cancel)
	var currentAICancelMu sync.Mutex
	var currentAICancel context.CancelFunc

	// PHASE 7: Register health monitor with disconnect callback for this connection
	// The callback triggers cleanup when health monitor detects unhealthy connection
	healthDisconnectOnce := sync.Once{}
	healthDisconnectCallback := func(reason string) {
		healthDisconnectOnce.Do(func() {
			h.logger.Warn("Health monitor triggered disconnection",
				zap.String("sessionId", sessionID.Hex()),
				zap.String("reason", reason))
			// Signal cleanup to all goroutines
			cleanup.Close()
		})
	}
	healthMonitor := NewConnectionHealthMonitorWithCallback(conn, h.logger, &h.writeMutex, sessionID.Hex(), healthDisconnectCallback)
	healthPool := GetHealthMonitorPool(h.logger)
	healthPool.Register(sessionID.Hex(), healthMonitor)
	defer func() {
		healthPool.Unregister(sessionID.Hex())
	}()
	h.logger.Info("Connection health monitor registered with disconnect callback",
		zap.String("sessionId", sessionID.Hex()))

	// PHASE 5: Register circuit breaker for this connection (cleanup on disconnect)
	circuitBreakerRegistry := GetCircuitBreakerRegistry(h.logger)
	defer circuitBreakerRegistry.Remove(sessionID.Hex())

	// Goroutine for sending pings (tracked with WaitGroup)
	cleanup.wg.Add(1)
	middleware.SafeGo(h.logger, func() {
		defer cleanup.wg.Done()
		for {
			select {
			case <-ticker.C:
				if err := h.safeWriteControl(conn, websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
					h.logger.Warn("Failed to send ping", zap.Error(err))
					return
				}
			case <-httpCtx.Done():
				return
			case <-cleanup.done:
				return
			}
		}
	})

	// Channel for incoming messages (allows concurrent reading while AI processes)
	type wsMessage struct {
		data []byte
		err  error
	}
	messageChan := make(chan wsMessage, 10)

	// SECURITY: Frame rate limiter to prevent DoS attacks via high-frequency frames
	frameRateLimiter := NewFrameRateLimiter(sessionID.Hex(), h.logger)

	// Goroutine for reading WebSocket messages (runs concurrently with message processing)
	cleanup.wg.Add(1)
	middleware.SafeGo(h.logger, func() {
		defer cleanup.wg.Done()
		defer close(messageChan)
		for {
			select {
			case <-httpCtx.Done():
				return
			case <-cleanup.done:
				return
			default:
				_, messageData, err := conn.ReadMessage()

				// SECURITY: Apply frame rate limiting
				if !frameRateLimiter.Allow() {
					// Check if we should disconnect due to repeated violations
					if frameRateLimiter.ShouldDisconnect() {
						h.logger.Warn("SECURITY: Disconnecting client due to repeated frame rate violations",
							zap.String("sessionId", sessionID.Hex()),
							zap.Int("violations", frameRateLimiter.GetViolations()))
						cleanup.Close()
						return
					}
					// Skip this frame but continue reading (allow client to recover)
					continue
				}

				select {
				case messageChan <- wsMessage{data: messageData, err: err}:
				case <-httpCtx.Done():
					return
				case <-cleanup.done:
					return
				}
				if err != nil {
					return // Exit read goroutine on error
				}
			}
		}
	})

	// Main message processing loop
	for {
		select {
		case <-httpCtx.Done():
			h.logger.Info("🔌 WebSocket DISCONNECTING - HTTP context cancelled",
				zap.String("sessionId", sessionID.Hex()),
				zap.Bool("wasProcessing", isProcessing.Load()),
				zap.String("reason", "HTTP context cancelled (page navigation/refresh)"))
			// done channel will be closed by defer
			return

		case msg, ok := <-messageChan:
			if !ok {
				// Channel closed, exit
				return
			}

			// Handle read error
			if msg.err != nil {
				err := msg.err
				// Check if this is an idle timeout (expected after task completion)
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					h.logger.Info("🔌 WebSocket DISCONNECTING - idle timeout",
						zap.String("sessionId", sessionID.Hex()),
						zap.Duration("timeout", 300*time.Second),
						zap.Bool("wasProcessing", isProcessing.Load()),
						zap.String("reason", "No activity from client (user reviewing response)"))
					// done channel will be closed by defer
					return
				}

				// Record WebSocket error (only for non-timeout errors)
				if !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived) {
					metrics.WebSocketErrors.WithLabelValues("read_error").Inc()
				}

				// Check if this is a normal disconnection
				if websocket.IsCloseError(err,
					websocket.CloseGoingAway,          // 1001: browser navigation
					websocket.CloseAbnormalClosure,    // 1006: abnormal closure
					websocket.CloseNormalClosure,      // 1000: normal closure
					websocket.CloseNoStatusReceived) { // 1005: no status (browser refresh/close)
					closeCode := "unknown"
					if websocket.IsCloseError(err, websocket.CloseGoingAway) {
						closeCode = "1001-GoingAway (navigation)"
					} else if websocket.IsCloseError(err, websocket.CloseAbnormalClosure) {
						closeCode = "1006-AbnormalClosure"
					} else if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
						closeCode = "1000-NormalClosure"
					} else if websocket.IsCloseError(err, websocket.CloseNoStatusReceived) {
						closeCode = "1005-NoStatus (refresh/close)"
					}
					h.logger.Info("🔌 WebSocket DISCONNECTED - client closed",
						zap.String("sessionId", sessionID.Hex()),
						zap.String("closeCode", closeCode),
						zap.Bool("wasProcessing", isProcessing.Load()),
						zap.String("rawError", err.Error()))
				} else {
					// Truly unexpected network error
					h.logger.Warn("🔌 WebSocket DISCONNECTED - network error",
						zap.String("sessionId", sessionID.Hex()),
						zap.Bool("wasProcessing", isProcessing.Load()),
						zap.Error(err),
						zap.String("errorType", fmt.Sprintf("%T", err)))
				}
				// done channel will be closed by defer
				return
			}

			messageData := msg.data

			// Record message received and size
			metrics.WebSocketMessagesReceived.Inc()
			metrics.WebSocketMessageSize.Observe(float64(len(messageData)))

			// Layer 1: Validate raw message size (fail fast before JSON parsing)
			if len(messageData) > config.MaxMessageBytes {
				h.logger.Warn("Message rejected - size exceeds limit",
					zap.String("sessionId", sessionID.Hex()),
					zap.Int("messageSize", len(messageData)),
					zap.Int("maxSize", config.MaxMessageBytes))
				// Record validation rejection metrics
				metrics.RecordValidationRejection("websocket")
				metrics.RecordMessageSizeExceeded("content")
				h.sendError(conn, fmt.Sprintf("Message too large: %d bytes (max %d bytes / 1MB)", len(messageData), config.MaxMessageBytes))
				continue
			}

			// Parse user message
			var userMsg models.SendMessageRequest
			if err := json.Unmarshal(messageData, &userMsg); err != nil {
				h.sendError(conn, "Invalid message format")
				continue
			}

			// Handle stop execution request (can be received while AI is processing!)
			if userMsg.IsStopRequest() {
				h.logger.Info("🛑 Stop execution request received",
					zap.String("sessionId", sessionID.Hex()),
					zap.Bool("isProcessing", isProcessing.Load()))

				if isProcessing.Load() {
					// Cancel the current AI execution context
					currentAICancelMu.Lock()
					if currentAICancel != nil {
						currentAICancel()
						h.logger.Info("🛑 Stop execution - cancelled AI context",
							zap.String("sessionId", sessionID.Hex()))
					}
					currentAICancelMu.Unlock()

					// Also trigger interrupt via message notifier (for backward compatibility)
					notifier := GetMessageNotifier(h.logger)
					notifier.NotifyNewMessage(sessionID)

					h.logger.Info("🛑 Stop execution requested by user",
						zap.String("sessionId", sessionID.Hex()),
						zap.String("userId", userID))

					// Send stop confirmation notification
					stopNotification := models.StreamMessage{
						Type: "system_notification",
						Notification: &models.SystemNotification{
							Category: "execution_stopped",
							Title:    "Execution Stopped",
							Message:  "AI execution has been stopped by user request.",
							Severity: "info",
						},
					}
					if err := h.safeWriteJSON(conn, stopNotification); err != nil {
						h.logger.Warn("Failed to send stop notification",
							zap.String("sessionId", sessionID.Hex()),
							zap.Error(err))
					}
				} else {
					h.logger.Debug("Stop requested but no execution in progress",
						zap.String("sessionId", sessionID.Hex()))
				}
				continue
			}

			// Layer 2: Validate actual content size (after JSON overhead)
			if len(userMsg.Content) > config.MaxContentBytes {
				h.logger.Warn("Message content rejected - size exceeds limit",
					zap.String("sessionId", sessionID.Hex()),
					zap.Int("contentSize", len(userMsg.Content)),
					zap.Int("maxSize", config.MaxContentBytes))
				// Record validation rejection metrics
				metrics.RecordValidationRejection("content")
				metrics.RecordMessageSizeExceeded("content")
				h.sendError(conn, fmt.Sprintf("Message content too large: %d bytes (max %d bytes / 1MB)", len(userMsg.Content), config.MaxContentBytes))
				continue
			}

			// Production Rate Limiting: Check message rate limit (10 messages/minute per user)
			if !checkRateLimit(userID) {
				h.logger.Warn("Message rejected - rate limit exceeded",
					zap.String("sessionId", sessionID.Hex()),
					zap.String("userId", userID))
				metrics.RecordValidationRejection("rate_limit")
				h.sendError(conn, "Rate limit exceeded: maximum 10 messages per minute")
				continue
			}

			// Check if already processing a message (atomic compare-and-swap)
			if !isProcessing.CompareAndSwap(false, true) {
				h.logger.Warn("Message rejected - AI response in progress",
					zap.String("sessionId", sessionID.Hex()),
					zap.String("userId", userID))
				h.sendError(conn, "Please wait for current response to complete before sending another message")
				continue
			}

			// 🎬 LOG: Starting message processing
			h.logger.Info("🎬 STREAMING STARTED - processing user message",
				zap.String("sessionId", sessionID.Hex()),
				zap.String("userId", userID),
				zap.Int("contentLength", len(userMsg.Content)),
				zap.Bool("isProcessing", true))

			// Process message in a goroutine to allow reading stop messages concurrently
			// PHASE 1: Track goroutine with WaitGroup for orderly shutdown
			cleanup.wg.Add(1)
			go func(userMsg models.SendMessageRequest) {
				// PHASE 4: Track goroutine with unique ID for debugging
				goroutineID := fmt.Sprintf("msg-%d", time.Now().UnixNano())
				goroutineStart := time.Now()
				exitReason := "completed" // Default exit reason

				h.logger.Info("🚀 Message processing goroutine STARTED",
					zap.String("goroutineId", goroutineID),
					zap.String("sessionId", sessionID.Hex()),
					zap.Int("contentLength", len(userMsg.Content)))

				// PHASE 1: Ensure WaitGroup.Done is called on exit
				defer cleanup.wg.Done()
				defer func() {
					isProcessing.Store(false)
					// PHASE 4: Log goroutine exit with duration and reason
					h.logger.Info("🏁 Message processing goroutine ENDED",
						zap.String("goroutineId", goroutineID),
						zap.String("sessionId", sessionID.Hex()),
						zap.String("exitReason", exitReason),
						zap.Duration("duration", time.Since(goroutineStart)),
						zap.Bool("isProcessing", false))
				}()

				// PHASE 3: Check if already cancelled before starting any work
				// This prevents wasted effort if client disconnected during queue wait
				select {
				case <-httpCtx.Done():
					exitReason = "http_context_cancelled_early"
					h.logger.Info("⏭️ Skipping message processing - HTTP context already cancelled",
						zap.String("goroutineId", goroutineID),
						zap.String("sessionId", sessionID.Hex()),
						zap.String("reason", "client disconnected before processing started"))
					return
				case <-cleanup.done:
					exitReason = "cleanup_in_progress_early"
					h.logger.Info("⏭️ Skipping message processing - cleanup already in progress",
						zap.String("goroutineId", goroutineID),
						zap.String("sessionId", sessionID.Hex()),
						zap.String("reason", "cleanup signal received"))
					return
				default:
					// Context still valid, continue processing
				}

				// Emit user message to WebSocket immediately (before database save)
				userMsgEvent := models.StreamMessage{
					Type:    "user_message",
					Content: userMsg.Content,
				}
				if err := h.safeWriteJSON(conn, userMsgEvent); err != nil {
					h.logger.Warn("Failed to emit user message to WebSocket",
						zap.String("sessionId", sessionID.Hex()),
						zap.Error(err))
					// Continue processing even if emit fails
				}

				// Save user message to database
				savedUserMsg, err := h.chatService.SaveMessage(aiCtx, sessionID, "user", userMsg.Content, companyID)
				if err != nil {
					h.logger.Error("Failed to save user message", zap.Error(err))
					h.sendError(conn, "Failed to save message")
					return // Defer will reset isProcessing
				}

				// FIX: Emit saved message with database ID for frontend reconciliation
				// This allows frontend to update optimistic message with correct database ID
				savedMsgEvent := models.StreamMessage{
					Type:    "message_saved",
					Content: savedUserMsg.ID.Hex(), // Send database ID as content
				}
				if err := h.safeWriteJSON(conn, savedMsgEvent); err != nil {
					h.logger.Warn("Failed to emit saved message ID",
						zap.String("sessionId", sessionID.Hex()),
						zap.String("messageId", savedUserMsg.ID.Hex()),
						zap.Error(err))
				}

				// Notify any running subagents about new message (for subchats)
				notifier := GetMessageNotifier(h.logger)
				notifier.NotifyNewMessage(sessionID)

				// Check if this message is interrupting an active subchat
				isInterrupting := notifier.IsSessionRegistered(sessionID)
				if isInterrupting {
					h.logger.Info("User message sent to active subchat - delegating to subchat interrupt handler",
						zap.String("sessionId", sessionID.Hex()),
						zap.String("userId", userID))

					// CRITICAL FIX: Do NOT handle interruptions in main chat!
					// The subchat's own interrupt handler (in coordinator_tools.go:2873)
					// will pick up the notification via notifyCh and handle it properly.
					// This prevents the "I'm a coordinator" bug where main chat responds
					// to subchat interruptions with the coordinator system prompt.

					// NotifyNewMessage already called above - subchat will receive via <-notifyCh
					// Let the subchat maintain its execution context and agent identity

					// FIX: Send 'done' event to properly close the WebSocket stream
					// This prevents frontend from staying in isStreaming=true state
					doneEvent := models.StreamMessage{
						Type: "done",
					}
					if err := h.safeWriteJSON(conn, doneEvent); err != nil {
						h.logger.Warn("Failed to send done event for interrupt",
							zap.String("sessionId", sessionID.Hex()),
							zap.Error(err))
					}
					return // Defer will reset isProcessing; skip to next message, don't call streamAIResponse
				}

				// ONLY stream response if NOT interrupting a subchat (i.e., this is main chat)
				// Create cancellable context for this AI execution (allows stop button to cancel)
				// PHASE 3 Context Lifecycle: aiExecCtx is child of aiCtx which is child of httpCtx
				// So HTTP cancellation propagates automatically through context hierarchy!
				aiExecCtx, aiExecCancel := context.WithCancel(aiCtx)

				// Store the cancel function so stop handler can call it
				currentAICancelMu.Lock()
				currentAICancel = aiExecCancel
				currentAICancelMu.Unlock()

				// PHASE 3 Context Lifecycle: Simplified cancellation propagation
				// HTTP cancellation now flows automatically through context hierarchy:
				//   HTTP -> AI (10min) -> aiExecCtx
				// We only need to handle explicit cleanup.done signal
				cleanupDone := make(chan struct{})
				go func() {
					defer close(cleanupDone)
					select {
					case <-cleanup.done:
						h.logger.Info("🛑 Cleanup signal received - cancelling AI execution",
							zap.String("sessionId", sessionID.Hex()),
							zap.String("reason", "cleanup requested"))
						aiExecCancel()
					case <-aiExecCtx.Done():
						// Context cancelled (HTTP disconnect, timeout, or stop button)
						// Log the reason for debugging
						if aiExecCtx.Err() == context.Canceled {
							h.logger.Debug("AI execution context cancelled",
								zap.String("sessionId", sessionID.Hex()))
						} else if aiExecCtx.Err() == context.DeadlineExceeded {
							h.logger.Info("AI execution context deadline exceeded",
								zap.String("sessionId", sessionID.Hex()))
						}
					}
				}()

				// Ensure we clean up the cancel function after execution
				defer func() {
					currentAICancelMu.Lock()
					currentAICancel = nil
					currentAICancelMu.Unlock()
					aiExecCancel() // Always call cancel to release resources
					// Wait for cleanup goroutine to exit (prevents goroutine leak)
					<-cleanupDone
				}()

				h.streamAIResponse(aiExecCtx, conn, sessionID, userMsg.Content, companyID, cleanup)

				// Defer will reset isProcessing after response complete
			}(userMsg) // Pass userMsg to goroutine
		}
	}
}

// streamAIResponse streams AI response with tool execution events back to client using ai-service
func (h *ChatWebSocketHandler) streamAIResponse(ctx context.Context, conn *websocket.Conn, sessionID primitive.ObjectID, userMessage, companyID string, cleanup *StreamCleanup) {
	h.logger.Info("Streaming AI response via ai-service",
		zap.String("sessionId", sessionID.Hex()),
		zap.String("userMessage", userMessage))

	// Send "streaming_started" event to trigger "AI is thinking" indicator immediately
	// This tells the frontend to show the thinking indicator before any content arrives
	streamingStartedMsg := models.StreamMessage{
		Type:    "streaming_started",
		Content: sessionID.Hex(), // Send session ID so frontend knows which session is streaming
	}
	if err := h.safeWriteJSON(conn, streamingStartedMsg); err != nil {
		h.logger.Debug("Failed to send streaming_started event",
			zap.String("sessionId", sessionID.Hex()),
			zap.Error(err))
	}

	// Step 1: Get session to check for active subagent
	session, err := h.chatService.GetSession(ctx, sessionID, companyID)
	if err != nil {
		h.logger.Error("Failed to retrieve session", zap.Error(err))
		h.sendError(conn, "Failed to retrieve session")
		return
	}

	// Register for progress notifications (for subchat execution)
	progressCh := GetProgressNotifier(h.logger).RegisterSession(sessionID)
	defer GetProgressNotifier(h.logger).UnregisterSession(sessionID)

	// Launch goroutine to stream progress notifications to WebSocket (tracked with WaitGroup)
	cleanup.wg.Add(1)
	middleware.SafeGo(h.logger, func() {
		defer cleanup.wg.Done()
		for progress := range progressCh {
			progressMsg := models.StreamMessage{
				Type:    "token",
				Content: "\n\n" + progress.Message + "\n\n",
			}
			if err := h.safeWriteJSON(conn, progressMsg); err != nil {
				h.logger.Debug("Failed to send progress notification (client may have disconnected)",
					zap.String("sessionId", sessionID.Hex()),
					zap.Error(err))
				return
			}
		}
	})

	// Step 2: Determine active agent and fetch system prompt
	var systemPromptText string
	// Check for system subagent first (has priority)
	if session.ActiveSubagentName != nil && *session.ActiveSubagentName != "" {
		// Using system subagent - fetch from SubchatStorage
		subagent, err := h.subchatStorage.GetSubagent(*session.ActiveSubagentName)
		if err == nil && subagent != nil {
			systemPromptText = subagent.SystemPrompt
			h.logger.Info("Using system subagent prompt",
				zap.String("sessionId", sessionID.Hex()),
				zap.String("subagentName", *session.ActiveSubagentName),
				zap.String("promptPrefix", systemPromptText[:min(200, len(systemPromptText))]))
		} else {
			h.logger.Warn("Failed to fetch system subagent, falling back to default prompt",
				zap.String("subagentName", *session.ActiveSubagentName),
				zap.Error(err))
		}
	} else if session.ActiveSubagentID != nil {
		// Using user-created subagent - fetch subagent's prompt from AI settings
		subagent, err := h.aiSettingsService.GetSubagent(ctx, *session.ActiveSubagentID, companyID)
		if err == nil && subagent != nil {
			systemPromptText = storage.BaseSubagentPrompt + "\n\n## YOUR SPECIALIZATION\n\n" + subagent.SystemPrompt
			h.logger.Info("Using user subagent prompt",
				zap.String("subagentId", session.ActiveSubagentID.Hex()),
				zap.String("subagentName", subagent.Name),
				zap.String("promptPrefix", systemPromptText[:min(200, len(systemPromptText))]))
		} else {
			h.logger.Warn("Failed to fetch user subagent, falling back to system prompt", zap.Error(err))
		}
	}

	// If no subagent or subagent fetch failed, use global system prompt
	if systemPromptText == "" {
		h.logger.Debug("Attempting to retrieve global system prompt",
			zap.String("userId", session.UserID),
			zap.String("companyId", companyID),
			zap.String("sessionId", sessionID.Hex()))

		var promptErr error
		systemPromptText, promptErr = h.aiSettingsService.GetSystemPrompt(ctx, session.UserID, companyID)

		if promptErr != nil {
			h.logger.Warn("Failed to retrieve system prompt",
				zap.Error(promptErr),
				zap.String("userId", session.UserID),
				zap.String("companyId", companyID))
		} else if systemPromptText != "" {
			h.logger.Info("Using global system prompt",
				zap.String("userId", session.UserID),
				zap.Int("promptLength", len(systemPromptText)))
		} else {
			// No custom prompt configured - detect model and use appropriate default
			aiConfig := h.aiService.GetConfig()
			isClaudeModel := strings.Contains(strings.ToLower(aiConfig.Model), "claude") ||
				strings.Contains(strings.ToLower(aiConfig.Provider), "anthropic")

			if isClaudeModel {
				systemPromptText = ClaudeSystemPrompt
				h.logger.Info("Using Claude-optimized system prompt",
					zap.String("userId", session.UserID),
					zap.String("model", aiConfig.Model),
					zap.String("provider", aiConfig.Provider),
					zap.Int("promptLength", len(systemPromptText)))
			} else {
				systemPromptText = DefaultSystemPrompt
				h.logger.Info("Using default (GPT) system prompt",
					zap.String("userId", session.UserID),
					zap.String("model", aiConfig.Model),
					zap.String("provider", aiConfig.Provider),
					zap.Int("promptLength", len(systemPromptText)))
			}
		}
	}

	// ALWAYS append critical system guidance (filesystem context + anti-loop rules + session context)
	// This is appended regardless of custom prompts to ensure consistent behavior
	// Note: For direct subagent chats, we provide autonomous execution guidance instead of delegation instructions
	projectRoot := tools.GetProjectRoot()
	isDirectSubagentChat := (session.ActiveSubagentName != nil && *session.ActiveSubagentName != "") || session.ActiveSubagentID != nil

	var criticalGuidance string
	if isDirectSubagentChat {
		// Direct subagent mode: Autonomous execution without delegation
		criticalGuidance = fmt.Sprintf(`

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
CRITICAL SYSTEM BEHAVIOR (NON-OVERRIDABLE)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🔬 SURGICAL EDIT MODE - ULTRA-STRICT (HIGHEST PRIORITY):
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
You are in SURGICAL EDIT MODE. Make MINIMAL changes ONLY.

1. CHANGE ONLY WHAT'S EXPLICITLY REQUESTED
   ✅ If asked to "fix button color", change ONLY the color property
   ❌ Do NOT refactor the component
   ❌ Do NOT rename variables
   ❌ Do NOT reorganize imports
   ❌ Do NOT change formatting/indentation
   ❌ Do NOT add features or improvements
   ❌ Do NOT fix other bugs you notice

2. PRECISE, TARGETED EDITS
   - Use Edit tool for line-specific changes
   - Change the MINIMUM number of lines
   - Keep surrounding code EXACTLY as-is
   - Preserve existing style and formatting

   ⚠️ JSX/TSX FILES - EXTRA CAREFUL:
   - JSX is FRAGILE - one wrong bracket breaks everything
   - ALWAYS include complete JSX structures in old_string
   - Count opening/closing tags - they MUST match
   - Preserve ALL whitespace/indentation exactly
   - If editing JSX, include parent/sibling elements for context
   - Example: To change text in <div>Hello</div>, include the full <div> tags
   - NEVER edit just part of a JSX element - edit the whole element

3. WHEN IN DOUBT, DO LESS
   - Better to do too little than too much
   - If unsure if a change is needed, DON'T make it
   - If tempted to "improve" something, ASK first

4. BEFORE EVERY CHANGE, ASK YOURSELF:
   ✓ "Did the user EXPLICITLY ask for this?"
   ✓ "Is this ABSOLUTELY necessary to solve the stated problem?"
   ✓ "Can I solve this with FEWER changes?"
   If ANY answer is NO → Don't make that change

EXAMPLES:
✅ GOOD: User asks "fix button color to blue" → Change 1 line: color: 'blue'
❌ BAD: User asks "fix button color to blue" → Change color + refactor component + rename vars

BEFORE COMPLETING TASK:
- Review your changes
- Count lines modified
- If you changed >10 lines for a simple fix, you probably over-engineered
- If unsure, explain your changes to the user and ask if it looks correct

🔍 MANDATORY SYNTAX VALIDATION:
- After editing TypeScript/TSX files, ALWAYS run: npx tsc --noEmit
- If compilation fails, FIX IT before marking task complete
- Pay special attention to JSX syntax errors (mismatched tags)
- If you see "Expected corresponding JSX closing tag", you broke JSX structure
- Read the error message carefully and fix the exact issue

EXAMPLES OF COMMON JSX MISTAKES:
❌ BAD - Incomplete edit (breaks structure):
old_string: "<div>"
new_string: "<div className='foo'>"
Problem: Missing closing </div>, breaks everything after

✅ GOOD - Complete element edit:
old_string: "<div>Hello</div>"
new_string: "<div className='foo'>Hello</div>"
Result: Complete structure, nothing breaks

REMEMBER: You are a SURGEON, not a RENOVATOR. Make precise incisions only.
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

DIRECT SUBAGENT MODE - AUTONOMOUS EXECUTION WITH TRACKING:
- **You are communicating directly with the user** - work autonomously
- **DO NOT delegate tasks or create subchats** - execute work yourself
- **If a task is outside your capability**, inform the user and ask if they want you to delegate
- **Only after user confirmation** should you suggest bringing in another specialist
- **USER is the orchestrator** in this mode, not you

TASK TRACKING (IMPORTANT):
- **ALWAYS create a human task** for the user's request (coordinator_create_human_task)
- **ALWAYS create an agent task** for yourself (coordinator_create_agent_task) with detailed context
- **Break work into todos** with clear descriptions and file paths
- **Update todo status** as you complete each step (coordinator_update_todo_status)
- This provides visibility into your progress and helps track completed work

SESSION CONTEXT:
- **CURRENT CHAT SESSION ID**: %s
- DO NOT ask the user for the session ID - it is provided above

FILESYSTEM CONTEXT:
- **PROJECT ROOT**: %s
- **PATH FORMAT**: ALWAYS use Unix/Mac forward slashes (/) - NEVER backslashes (\)
- **CORRECT**: %s/ui/src/file.tsx OR ./ui/src/file.tsx
- **FORBIDDEN**: C:\Users\... OR C:\\Users\... (Windows paths)
- Prefer relative paths from project root: ./ui/src/main.tsx
- Bash working directory: %s (automatically set)
- System directories BLOCKED: /etc, /var, /sys, /usr

TOOL USAGE RULES - PREVENT INFINITE LOOPS:
1. **NEVER call the same tool with identical arguments consecutively**
2. **If a tool returns a result, USE it** - don't re-call expecting different output
3. **If stuck, change approach** - try different tool or different arguments
4. **Circuit breaker**: System stops you after 3 identical calls in 5 attempts

❌ BAD PATTERN (causes circuit breaker):
  list_directory(./components) → list_directory(./components) → list_directory(./components)

✅ GOOD PATTERN (smart exploration):
  list_directory(./components) → find what you need → read_file(specific_file)

✅ If stuck, try different approach:
  list_directory fails → try bash("find . -name pattern") OR code_index_search

**When user gives you an explicit file path, just read it - don't explore directories!**

EDIT TOOL USAGE - CRITICAL FOR AVOIDING SYNTAX ERRORS:
1. **ALWAYS read the file first** before using Edit tool
2. **Copy exact text** from file output (including whitespace) for old_string
3. **For JSX/TSX edits:**
   - Match COMPLETE elements: <tag>content</tag>
   - Include surrounding context (lines before/after)
   - Count opening/closing tags carefully
   - Test: Does old_string appear exactly once in the file? (should be unique)
4. **After Edit, verify:**
   - Run: npx tsc --noEmit (for TS/TSX files)
   - Run: make lint (if available)
   - If errors appear, READ them and FIX immediately
5. **If Edit fails:**
   - Don't try again with same old_string
   - Read the file again to see current state
   - Find the correct unique match
   - Try with more surrounding context

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`, sessionID.Hex(), projectRoot, projectRoot, projectRoot)
	} else {
		// Coordinator mode: Standard delegation workflow
		criticalGuidance = fmt.Sprintf(`

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
CRITICAL SYSTEM BEHAVIOR (NON-OVERRIDABLE)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🔬 SURGICAL EDIT MODE - ULTRA-STRICT (HIGHEST PRIORITY):
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
When creating agent tasks, instruct agents to make MINIMAL changes ONLY.

TASK CREATION GUIDELINES:
- Include explicit "DO NOT CHANGE" section listing what should NOT be modified
- Specify exact files and lines to change when possible
- Estimate expected line changes (e.g., "Expected: ~5 lines changed")
- Set clear scope boundaries
- Emphasize minimal, surgical edits over comprehensive refactors
- If agent changes >3x expected lines, review carefully for scope creep

AGENT INSTRUCTIONS TO INCLUDE IN TASKS:
✅ Change ONLY what's explicitly requested
❌ Do NOT refactor or improve unrelated code
❌ Do NOT rename variables unless specifically asked
❌ Do NOT reorganize imports or fix formatting
❌ Do NOT add features beyond the stated requirement

Example Task Context:
"Fix button color in LoginButton.tsx
EXACT CHANGE: Line 45, change color: 'red' to color: 'blue'
DO NOT CHANGE: button size, layout, hover states, variable names, imports"
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

SESSION CONTEXT:
- **CURRENT CHAT SESSION ID**: %s
- **IMPORTANT**: When using execute_subagent tool, ALWAYS use parentChatId: "%s"
- DO NOT ask the user for the session ID - it is provided above
- This session ID links subagent work back to this conversation

FILESYSTEM CONTEXT:
- **PROJECT ROOT**: %s
- **PATH FORMAT**: ALWAYS use Unix/Mac forward slashes (/) - NEVER backslashes (\)
- **CORRECT**: %s/ui/src/file.tsx OR ./ui/src/file.tsx
- **FORBIDDEN**: C:\Users\... OR C:\\Users\... (Windows paths)
- Prefer relative paths from project root: ./ui/src/main.tsx
- Bash working directory: %s (automatically set)
- System directories BLOCKED: /etc, /var, /sys, /usr

TOOL USAGE RULES - PREVENT INFINITE LOOPS:
1. **NEVER call the same tool with identical arguments consecutively**
2. **If a tool returns a result, USE it** - don't re-call expecting different output
3. **If stuck, change approach** - try different tool or different arguments
4. **Circuit breaker**: System stops you after 3 identical calls in 5 attempts

❌ BAD PATTERN (causes circuit breaker):
  list_directory(./components) → list_directory(./components) → list_directory(./components)

✅ GOOD PATTERN (smart exploration):
  list_directory(./components) → find what you need → read_file(specific_file)

✅ If stuck, try different approach:
  list_directory fails → try bash("find . -name pattern") OR code_index_search

**When user gives you an explicit file path, just read it - don't explore directories!**

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`, sessionID.Hex(), sessionID.Hex(), projectRoot, projectRoot, projectRoot)
	}
	systemPromptText += criticalGuidance

	// Step 3: Get conversation history for context
	messages, err := h.chatService.GetSessionMessages(ctx, sessionID)
	if err != nil {
		h.logger.Error("Failed to retrieve conversation history", zap.Error(err))
		h.sendError(conn, "Failed to retrieve conversation history")
		return
	}

	h.logger.Debug("Retrieved conversation history",
		zap.String("sessionId", sessionID.Hex()),
		zap.Int("messageCount", len(messages)))

	// Step 3.5: Check if compaction is needed and perform if necessary
	compactionResult, compactionErr := h.compactionOrchestrator.CompactIfNeeded(ctx, sessionID, messages, companyID)
	if compactionErr != nil {
		h.logger.Warn("Compaction check failed (continuing without compaction)",
			zap.String("sessionId", sessionID.Hex()),
			zap.Error(compactionErr))
	} else if compactionResult.WasCompacted {
		h.logger.Info("Context compaction performed",
			zap.String("sessionId", sessionID.Hex()),
			zap.Int("messagesCompacted", compactionResult.MessagesCompacted),
			zap.Int("originalTokens", compactionResult.OriginalTokens),
			zap.Int("compactedTokens", compactionResult.CompactedTokens))

		// Send compaction notification to frontend
		if compactionResult.Notification != nil {
			h.sendSystemNotification(conn, *compactionResult.Notification)
		}

		// Re-fetch messages after compaction
		messages, err = h.chatService.GetSessionMessages(ctx, sessionID)
		if err != nil {
			h.logger.Error("Failed to retrieve messages after compaction", zap.Error(err))
			h.sendError(conn, "Failed to retrieve conversation after compaction")
			return
		}
		h.logger.Debug("Retrieved compacted conversation history",
			zap.String("sessionId", sessionID.Hex()),
			zap.Int("messageCount", len(messages)))
	}

	// Step 4: Convert MongoDB messages to LangChain format
	langchainMessages := aiservice.ConvertToLangChainMessages(messages)

	// Step 5: Inject system prompt as first message (if exists)
	if systemPromptText != "" {
		// Step 5: Inject session ID, company ID, error prevention mode, complexity analysis mode, and AI config into context for tool access
		ctxWithSession := context.WithValue(ctx, aiservice.SessionIDKey, sessionID.Hex())
		ctxWithCompany := context.WithValue(ctxWithSession, aiservice.CompanyIDKey, companyID)
		ctxWithErrorPrevention := context.WithValue(ctxWithCompany, aiservice.ErrorPreventionModeKey, session.ErrorPreventionMode)
		ctxWithComplexityAnalysis := context.WithValue(ctxWithErrorPrevention, aiservice.ComplexityAnalysisModeKey, session.ComplexityAnalysisMode)
		// Inject AI config for code summarizer to use current session's provider/model
		ctxWithAIConfig := context.WithValue(ctxWithComplexityAnalysis, aiservice.AIConfigKey, h.aiService.GetConfig())

		h.logger.Info("Context prepared for AI execution",
			zap.String("sessionID", sessionID.Hex()),
			zap.Bool("errorPreventionMode", session.ErrorPreventionMode),
			zap.Bool("complexityAnalysisMode", session.ComplexityAnalysisMode),
			zap.String("aiModel", h.aiService.GetConfig().Model),
			zap.String("aiProvider", h.aiService.GetConfig().Provider))

		// Step 6: Register for interrupt notifications (for prioritized interrupt handling)
		notifier := GetMessageNotifier(h.logger)
		interruptCh := notifier.RegisterSession(sessionID)
		defer notifier.UnregisterSession(sessionID)

		// Step 7: Create WebSocket sink for streaming output (using local adapter to avoid import cycles)
		// PHASE 3 Buffer Monitoring: Use session-aware sink for buffer usage tracking
		outputSink := newWebSocketSinkWithSession(conn, h, h.logger, sessionID.Hex())

		// Step 8: Determine allowed tools based on mode
		var allowedTools []string
		if isDirectSubagentChat {
			// Direct subagent mode: Use filtered tools (exclude delegation tools)
			allowedTools = h.aiService.GetAllowedToolsForDirectSubagent()
			h.logger.Info("Using direct subagent mode with filtered tools",
				zap.String("sessionId", sessionID.Hex()),
				zap.Int("allowedToolsCount", len(allowedTools)),
				zap.String("subagentName", func() string {
					if session.ActiveSubagentName != nil {
						return *session.ActiveSubagentName
					}
					if session.ActiveSubagentID != nil {
						return session.ActiveSubagentID.Hex()
					}
					return "unknown"
				}()))
		} else {
			// Coordinator mode: Use all tools (includes delegation)
			h.logger.Info("Using coordinator mode with full tool access",
				zap.String("sessionId", sessionID.Hex()))
			// nil = all tools
			allowedTools = nil
		}

		// Step 9: Create tool result processor with token-based deflection + byte-based limits
		// This combines:
		// 1. Token-based interceptor (context-aware, per-tool limits, metrics)
		// 2. Byte-based processor (hard size limits, truncation tiers)
		remainingContext := h.calculateRemainingContext(systemPromptText, messages)
		h.logger.Info("📊 Context budget calculated for tool result processing",
			zap.Int("remainingContextTokens", remainingContext),
			zap.Int("messageCount", len(messages)))

		toolResultProcessor := func(toolName string, output interface{}) (processedOutput string, shouldSave bool, shouldStream bool) {
			// PHASE 0: Check for code search summarization metadata and send notification
			if toolName == "code_index_search" {
				if resultMap, ok := output.(map[string]interface{}); ok {
					if summarization, exists := resultMap["summarization"].(map[string]interface{}); exists {
						if enabled, _ := summarization["enabled"].(bool); enabled {
							resultsSummarized, _ := summarization["resultsSummarized"].(int)
							// Handle float64 from JSON unmarshaling
							if rs, ok := summarization["resultsSummarized"].(float64); ok {
								resultsSummarized = int(rs)
							}
							tokensUsed, _ := summarization["tokensUsed"].(int)
							if tu, ok := summarization["tokensUsed"].(float64); ok {
								tokensUsed = int(tu)
							}

							h.logger.Info("📋 Code search results summarized",
								zap.Int("resultsSummarized", resultsSummarized),
								zap.Int("tokensUsed", tokensUsed))

							outputSink.SendSystemNotification(models.SystemNotification{
								Category: "summarization",
								Title:    "Search Results Summarized",
								Message:  fmt.Sprintf("Summarized %d code search results", resultsSummarized),
								Severity: "info",
								Metadata: map[string]interface{}{
									"toolName":          toolName,
									"resultsSummarized": resultsSummarized,
									"tokensUsed":        tokensUsed,
								},
							})
						}
					}
				}
			}

			// PHASE 1: Token-based deflection (context-aware, per-tool limits)
			if h.resultInterceptor != nil {
				processedResult, deflection := h.resultInterceptor.CheckResult(toolName, output, remainingContext)
				if deflection.WasDeflected {
					// Record metrics
					metrics.RecordToolResultDeflection(toolName)
					metrics.RecordToolResultTokens(toolName, deflection.OriginalSize, true)

					h.logger.Info("🛑 Tool result deflected by token-based interceptor",
						zap.String("tool", toolName),
						zap.Int("originalTokens", deflection.OriginalSize),
						zap.Int("maxAllowed", deflection.MaxAllowed),
						zap.Int("remainingContext", remainingContext))

					// Send deflection notification to frontend
					overagePercent := float64(deflection.OriginalSize-deflection.MaxAllowed) / float64(deflection.MaxAllowed) * 100
					outputSink.SendSystemNotification(models.SystemNotification{
						Category: "deflection",
						Title:    "Tool Result Deflected",
						Message:  fmt.Sprintf("%s result exceeded token limit (%d tokens)", toolName, deflection.OriginalSize),
						Severity: "warning",
						Metadata: map[string]interface{}{
							"toolName":   toolName,
							"tokenCount": deflection.OriginalSize,
							"limit":      deflection.MaxAllowed,
							"overage":    fmt.Sprintf("%.1f%%", overagePercent),
						},
					})

					// Return deflection message
					return deflection.Message, false, true // Don't save full, do stream the message
				}

				// Record non-deflected result metrics
				estimator := NewTokenEstimator()
				tokens := estimator.EstimateTokens(processedResult)
				metrics.RecordToolResultTokens(toolName, tokens, false)

				// Update output for next phase
				output = processedResult
			}

			// PHASE 2: Byte-based processing (hard size limits, truncation tiers)
			processed := h.processToolResultWithSizeLimit(toolName, output)

			// Send summarization notification for suppressed or truncated results
			if processed.Tier == "suppressed" || processed.Tier == "truncated" {
				outputSink.SendSystemNotification(models.SystemNotification{
					Category: "summarization",
					Title:    "Tool Result Condensed",
					Message:  fmt.Sprintf("%s result condensed (%s tier)", toolName, processed.Tier),
					Severity: "info",
					Metadata: map[string]interface{}{
						"toolName":     toolName,
						"tier":         processed.Tier,
						"originalSize": processed.OriginalSize,
						"isTruncated":  processed.IsTruncated,
					},
				})
			}

			return processed.OutputStr, processed.ShouldSaveFull, processed.ShouldStream
		}

		// Step 10: Create executor config
		// Callback for when message is saved despite WebSocket disconnection
		onMessageSavedWhileDisconnected := func(sessID primitive.ObjectID) {
			broadcaster := GetWebSocketBroadcaster(h.logger)
			broadcaster.BroadcastToSession(sessID, models.StreamMessage{
				Type:    "message_saved",
				Content: "AI response saved - please refresh to see the full message",
			})
		}

		execConfig := executor.StreamConfig{
			SessionID:                       sessionID,
			CompanyID:                       companyID,
			SystemPrompt:                    systemPromptText,
			AllowedTools:                    allowedTools,
			OutputSink:                      outputSink,
			InterruptCh:                     interruptCh,
			ToolResultProcessor:             toolResultProcessor,
			OnMessageSavedWhileDisconnected: onMessageSavedWhileDisconnected,
			Logger:                          h.logger,
		}

		// Step 11: Create and execute the stream executor (with adapted chat service)
		chatServiceAdapter := &chatServiceAdapter{service: h.chatService}
		exec := executor.NewStreamExecutor(execConfig, chatServiceAdapter, h.aiService)
		fullResponse, err := exec.Execute(ctxWithAIConfig, langchainMessages)

		if err != nil {
			h.logger.Error("AI execution failed", zap.Error(err))
			if !outputSink.IsDisconnected() {
				h.sendError(conn, "AI execution failed: "+err.Error())
			}
			return
		}

		h.logger.Info("AI execution completed successfully",
			zap.String("sessionId", sessionID.Hex()),
			zap.Int("responseLength", len(fullResponse)))
	}
}

// streamToolResult streams tool result to WebSocket with chunking for large outputs
// Results larger than 10KB are split into chunks to prevent WebSocket message size limits
func (h *ChatWebSocketHandler) streamToolResult(conn *websocket.Conn, result models.ToolResultEvent) error {
	// Serialize result to JSON to check size
	resultJSON, err := json.Marshal(result.Result)
	if err != nil {
		return fmt.Errorf("failed to marshal tool result: %w", err)
	}

	const maxChunkSize = 10 * 1024 // 10KB

	// If result is small enough, send as single message
	if len(resultJSON) <= maxChunkSize {
		streamMsg := models.StreamMessage{
			Type:       "tool_result",
			ToolResult: &result,
		}
		if err := h.safeWriteJSON(conn, streamMsg); err != nil {
			return fmt.Errorf("failed to send tool result: %w", err)
		}
		return nil
	}

	// Large result - split into chunks
	h.logger.Info("Chunking large tool result",
		zap.String("toolId", result.ID),
		zap.Int("totalBytes", len(resultJSON)))

	resultStr := string(resultJSON)
	totalChunks := (len(resultStr) + maxChunkSize - 1) / maxChunkSize

	for i := 0; i < totalChunks; i++ {
		start := i * maxChunkSize
		end := start + maxChunkSize
		if end > len(resultStr) {
			end = len(resultStr)
		}

		chunk := models.StreamMessage{
			Type: "tool_result_chunk",
			ToolResult: &models.ToolResultEvent{
				ID: result.ID,
				Result: models.ToolResultChunk{
					ID:    result.ID,
					Chunk: resultStr[start:end],
					Index: i,
					Total: totalChunks,
					Done:  i == totalChunks-1,
				},
				Error:      result.Error,
				DurationMs: result.DurationMs,
			},
		}

		if err := h.safeWriteJSON(conn, chunk); err != nil {
			return fmt.Errorf("failed to send chunk %d/%d: %w", i+1, totalChunks, err)
		}

		h.logger.Debug("Sent tool result chunk",
			zap.String("toolId", result.ID),
			zap.Int("chunk", i+1),
			zap.Int("total", totalChunks))
	}

	return nil
}

// sendError sends an error message to the WebSocket client
func (h *ChatWebSocketHandler) sendError(conn *websocket.Conn, errorMsg string) {
	errMsg := models.StreamMessage{
		Type:  "error",
		Error: errorMsg,
	}
	if err := h.safeWriteJSON(conn, errMsg); err != nil {
		h.logger.Error("Failed to send error message", zap.Error(err))
	}
}

// generateSuppressedToolResultMessage creates Claude-style helpful message for oversized results
func (h *ChatWebSocketHandler) generateSuppressedToolResultMessage(
	toolName string,
	size int,
	output interface{},
) string {
	// Extract metadata/summary (tool-specific logic)
	summary := extractToolResultSummary(toolName, output)

	// Build helpful message
	msg := fmt.Sprintf(`⚠️ Tool Result Too Large

The output from '%s' is too large to display (%s).

%s

**Suggested Alternatives:**

`, toolName, config.FormatSize(size), summary)

	// Add tool-specific suggestions
	switch {
	case strings.Contains(toolName, "read_file") || strings.Contains(toolName, "file_read"):
		msg += `- Use 'grep' or 'search' to find specific content instead of reading entire file
- Read the file in smaller chunks using offset/limit parameters
- Use 'file_info' to get metadata without content
- Apply filters or patterns to reduce output size`

	case strings.Contains(toolName, "grep") || strings.Contains(toolName, "search"):
		msg += `- Add more specific search patterns to narrow results
- Use file type filters (e.g., glob: "*.go")
- Limit results with head_limit parameter
- Search in a specific subdirectory instead of entire codebase`

	case strings.Contains(toolName, "bash") || strings.Contains(toolName, "execute"):
		msg += `- Pipe output through 'head' or 'tail' (e.g., '| head -100')
- Use grep to filter relevant lines (e.g., '| grep ERROR')
- Redirect large output to a file for later inspection
- Add flags to reduce verbosity (e.g., --quiet, --summary)`

	case strings.Contains(toolName, "list_files") || strings.Contains(toolName, "glob"):
		msg += `- Use more specific glob patterns to narrow results
- Search in subdirectories instead of root
- Filter by file type or extension
- Use 'find' with -maxdepth to limit recursion`

	default:
		msg += `- Use more specific parameters or filters
- Request a subset of the data using pagination
- Ask for a summary instead of full details
- Consider breaking the operation into smaller steps`
	}

	msg += "\n\nPlease retry with adjusted parameters."

	return msg
}

// calculateRemainingContext estimates remaining context tokens based on conversation history.
// Uses provider capabilities for accurate context window sizing across different AI providers.
// The provider and model can be determined from session config for provider-specific limits.
func (h *ChatWebSocketHandler) calculateRemainingContext(systemPrompt string, messages []models.ChatMessage) int {
	// Get provider capabilities (defaults work for any provider)
	// TODO: In the future, pass provider/model from session config for provider-specific limits
	// e.g., config.GetProviderCapabilities("anthropic", "claude-3-opus")
	caps := config.DefaultProviderCapabilities()

	// Calculate bytes for system prompt
	systemPromptBytes := len(systemPrompt)

	// Calculate bytes for message history
	messagesBytes := 0
	for _, msg := range messages {
		messagesBytes += len(msg.Content)
		// Add overhead for role/metadata (~40 bytes per message)
		messagesBytes += 40
	}

	// Use provider capabilities to calculate remaining context
	remaining := caps.CalculateRemainingContext(systemPromptBytes, messagesBytes)

	return remaining
}

// processToolResultWithSizeLimit checks tool result size and applies appropriate handling
func (h *ChatWebSocketHandler) processToolResultWithSizeLimit(
	toolName string,
	output interface{},
) ToolResultProcessed {
	// Step 1: Calculate original size
	var originalSize int
	var outputStr string

	if output == nil {
		return ToolResultProcessed{
			OutputStr:      "",
			ShouldStream:   true,
			ShouldSaveFull: true,
			Tier:           "normal",
			OriginalSize:   0,
			IsTruncated:    false,
		}
	}

	// Convert to string and calculate size
	if str, ok := output.(string); ok {
		outputStr = str
		originalSize = len(str)
	} else {
		outputBytes, err := json.Marshal(output)
		if err != nil {
			h.logger.Error("Failed to marshal tool result for size check",
				zap.String("tool", toolName),
				zap.Error(err))
			outputStr = fmt.Sprintf("Error: failed to process tool result: %v", err)
			return ToolResultProcessed{
				OutputStr:      outputStr,
				ShouldStream:   true,
				ShouldSaveFull: false,
				Tier:           "error",
				OriginalSize:   0,
				IsTruncated:    false,
			}
		}
		outputStr = string(outputBytes)
		originalSize = len(outputBytes)
	}

	// Step 2: Apply tier-based logic
	if originalSize <= config.MaxToolResultNormalBytes {
		// Tier 1: Normal - stream and save fully
		h.logger.Debug("Tool result within normal size limit",
			zap.String("tool", toolName),
			zap.Int("size", originalSize),
			zap.String("tier", "normal"))

		return ToolResultProcessed{
			OutputStr:      outputStr,
			ShouldStream:   true,
			ShouldSaveFull: true,
			Tier:           "normal",
			OriginalSize:   originalSize,
			IsTruncated:    false,
		}

	} else if originalSize <= config.MaxToolResultTruncatedBytes {
		// Tier 2: Truncated - stream preview + metadata, save full
		preview := outputStr
		if len(outputStr) > config.ToolResultPreviewBytes {
			preview = outputStr[:config.ToolResultPreviewBytes]
		}

		metadata := fmt.Sprintf(
			"\n\n[Output truncated: %s / %s shown. Full result saved to database.]",
			config.FormatSize(config.ToolResultPreviewBytes),
			config.FormatSize(originalSize),
		)

		h.logger.Info("Tool result truncated for display",
			zap.String("tool", toolName),
			zap.Int("originalSize", originalSize),
			zap.Int("previewSize", len(preview)),
			zap.String("tier", "truncated"))

		return ToolResultProcessed{
			OutputStr:      preview + metadata,
			ShouldStream:   true,
			ShouldSaveFull: true, // Save full content to DB
			Tier:           "truncated",
			OriginalSize:   originalSize,
			IsTruncated:    true,
		}

	} else if originalSize <= config.MaxToolResultSuppressedBytes {
		// Tier 3: Suppressed - stream helpful message, DON'T save full content
		suppressedMsg := h.generateSuppressedToolResultMessage(
			toolName,
			originalSize,
			output,
		)

		h.logger.Warn("Tool result suppressed due to size",
			zap.String("tool", toolName),
			zap.Int("size", originalSize),
			zap.String("tier", "suppressed"))

		return ToolResultProcessed{
			OutputStr:      suppressedMsg,
			ShouldStream:   true,
			ShouldSaveFull: false, // Save only the message, not full content
			Tier:           "suppressed",
			OriginalSize:   originalSize,
			IsTruncated:    true,
		}

	} else {
		// Beyond hard limit - error
		errorMsg := fmt.Sprintf(
			"Tool result size (%s) exceeds maximum allowed (%s). Tool: %s",
			config.FormatSize(originalSize),
			config.FormatSize(config.MaxToolResultSuppressedBytes),
			toolName,
		)

		h.logger.Error("Tool result exceeded hard limit",
			zap.String("tool", toolName),
			zap.Int("size", originalSize),
			zap.Int("maxSize", config.MaxToolResultSuppressedBytes))

		return ToolResultProcessed{
			OutputStr:      errorMsg,
			ShouldStream:   true,
			ShouldSaveFull: false,
			Tier:           "error",
			OriginalSize:   originalSize,
			IsTruncated:    true,
		}
	}
}
