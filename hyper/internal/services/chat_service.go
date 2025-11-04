package services

import (
	"context"
	"fmt"
	"time"

	"hyper/internal/config"
	"hyper/internal/metrics"
	"hyper/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// ChatService manages chat sessions and messages with MongoDB storage
type ChatService struct {
	mongoClient        *mongo.Client
	sessionsCollection *mongo.Collection
	messagesCollection *mongo.Collection
	logger             *zap.Logger
}

// NewChatService creates a new chat service instance
func NewChatService(db *mongo.Database, logger *zap.Logger) (*ChatService, error) {
	service := &ChatService{
		mongoClient:        db.Client(),
		sessionsCollection: db.Collection("chat_sessions"),
		messagesCollection: db.Collection("chat_messages"),
		logger:             logger,
	}

	// Create indexes
	ctx := context.Background()

	// Index on sessions: {userId, companyId} for user session queries
	_, err := service.sessionsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "userId", Value: 1},
			{Key: "companyId", Value: 1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create sessions user index: %w", err)
	}

	// Index on sessions: {companyId} for company-level isolation
	_, err = service.sessionsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "companyId", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create sessions company index: %w", err)
	}

	// Index on messages: {sessionId, timestamp} for efficient message retrieval
	_, err = service.messagesCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "sessionId", Value: 1},
			{Key: "timestamp", Value: -1}, // Descending for latest messages first
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create messages session index: %w", err)
	}

	logger.Info("Chat service initialized with MongoDB indexes")
	return service, nil
}

// executeInTransaction executes a function within a MongoDB transaction
// Handles session management, transaction lifecycle, and error handling
func (s *ChatService) executeInTransaction(ctx context.Context, fn func(sessCtx mongo.SessionContext) error) error {
	session, err := s.mongoClient.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start MongoDB session: %w", err)
	}
	defer session.EndSession(ctx)

	// Execute the function within a transaction
	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		return nil, fn(sessCtx)
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

// CreateSession creates a new chat session for a user
func (s *ChatService) CreateSession(ctx context.Context, userID, companyID, title string) (*models.ChatSession, error) {
	return s.CreateSessionWithParent(ctx, userID, companyID, title, nil)
}

// CreateSessionWithParent creates a new chat session with an optional parent chat ID (for subchats)
func (s *ChatService) CreateSessionWithParent(ctx context.Context, userID, companyID, title string, parentChatID *primitive.ObjectID) (*models.ChatSession, error) {
	now := time.Now().UTC()
	session := &models.ChatSession{
		ID:           primitive.NewObjectID(),
		UserID:       userID,
		CompanyID:    companyID,
		Title:        title,
		ParentChatID: parentChatID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	_, err := s.sessionsCollection.InsertOne(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat session: %w", err)
	}

	// Record session creation metric
	metrics.ChatSessionsCreated.Inc()

	logFields := []zap.Field{
		zap.String("sessionId", session.ID.Hex()),
		zap.String("userId", userID),
		zap.String("companyId", companyID),
	}

	if parentChatID != nil {
		logFields = append(logFields, zap.String("parentChatId", parentChatID.Hex()))
		s.logger.Info("Subchat session created", logFields...)
	} else {
		s.logger.Info("Chat session created", logFields...)
	}

	return session, nil
}

// GetUserSessions retrieves all chat sessions for a user within their company
func (s *ChatService) GetUserSessions(ctx context.Context, userID, companyID string) ([]models.ChatSession, error) {
	filter := bson.M{
		"userId":    userID,
		"companyId": companyID,
	}

	opts := options.Find().SetSort(bson.D{{Key: "updatedAt", Value: -1}}) // Latest first

	cursor, err := s.sessionsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query chat sessions: %w", err)
	}
	defer cursor.Close(ctx)

	var sessions []models.ChatSession
	if err := cursor.All(ctx, &sessions); err != nil {
		return nil, fmt.Errorf("failed to decode chat sessions: %w", err)
	}

	return sessions, nil
}

// GetSession retrieves a specific chat session by ID
func (s *ChatService) GetSession(ctx context.Context, sessionID primitive.ObjectID, companyID string) (*models.ChatSession, error) {
	var session models.ChatSession
	filter := bson.M{
		"_id":       sessionID,
		"companyId": companyID, // Company-level isolation
	}

	err := s.sessionsCollection.FindOne(ctx, filter).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("chat session not found or access denied")
		}
		return nil, fmt.Errorf("failed to retrieve chat session: %w", err)
	}

	return &session, nil
}

// DeleteSession deletes a chat session and all its messages
// Uses transaction to ensure both operations succeed or fail together (prevents orphaned messages)
func (s *ChatService) DeleteSession(ctx context.Context, sessionID primitive.ObjectID, userID, companyID string) error {
	// Verify session belongs to user and company (authorization - outside transaction)
	session, err := s.GetSession(ctx, sessionID, companyID)
	if err != nil {
		return err
	}

	if session.UserID != userID {
		return fmt.Errorf("unauthorized: session does not belong to user")
	}

	// Execute both delete operations in a transaction
	err = s.executeInTransaction(ctx, func(sessCtx mongo.SessionContext) error {
		// Delete all messages
		_, err := s.messagesCollection.DeleteMany(sessCtx, bson.M{"sessionId": sessionID})
		if err != nil {
			return fmt.Errorf("failed to delete session messages: %w", err)
		}

		// Delete the session
		result, err := s.sessionsCollection.DeleteOne(sessCtx, bson.M{"_id": sessionID})
		if err != nil {
			return fmt.Errorf("failed to delete session: %w", err)
		}

		if result.DeletedCount == 0 {
			return fmt.Errorf("session not found")
		}

		return nil
	})

	if err != nil {
		return err
	}

	s.logger.Info("Chat session deleted",
		zap.String("sessionId", sessionID.Hex()),
		zap.String("userId", userID))

	return nil
}

// GetMessages retrieves messages for a session with pagination
func (s *ChatService) GetMessages(ctx context.Context, sessionID primitive.ObjectID, companyID string, limit, offset int) (*models.GetMessagesResponse, error) {
	// Verify session exists and user has access
	_, err := s.GetSession(ctx, sessionID, companyID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"sessionId": sessionID}

	// Count total messages
	total, err := s.messagesCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count messages: %w", err)
	}

	// Query messages with pagination
	opts := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: 1}}). // Ascending for chronological order
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := s.messagesCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []models.ChatMessage
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, fmt.Errorf("failed to decode messages: %w", err)
	}

	response := &models.GetMessagesResponse{
		Messages: messages,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
		HasMore:  int64(offset+len(messages)) < total,
	}

	return response, nil
}

// SaveMessage saves a message to the database
// Uses transaction to ensure message insert and session update are atomic
func (s *ChatService) SaveMessage(ctx context.Context, sessionID primitive.ObjectID, role, content string, companyID string) (*models.ChatMessage, error) {
	// Track operation timing for metrics
	startTime := time.Now()
	defer func() {
		// Record metrics using helper function
		metrics.RecordChatMessage(role, time.Since(startTime).Seconds())
	}()

	// Layer 3: Validate content size (defense in depth)
	if len(content) > config.MaxContentBytes {
		return nil, fmt.Errorf("message content exceeds maximum size of %d bytes (got %d bytes)", config.MaxContentBytes, len(content))
	}

	// Verify session exists and user has access (outside transaction)
	_, err := s.GetSession(ctx, sessionID, companyID)
	if err != nil {
		return nil, err
	}

	// Prepare message
	message := &models.ChatMessage{
		ID:        primitive.NewObjectID(),
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		Timestamp: time.Now().UTC(),
	}

	// Execute both operations in a transaction
	err = s.executeInTransaction(ctx, func(sessCtx mongo.SessionContext) error {
		// Insert message
		_, err := s.messagesCollection.InsertOne(sessCtx, message)
		if err != nil {
			return fmt.Errorf("failed to save message: %w", err)
		}

		// Update session's updatedAt timestamp (must succeed or transaction aborts)
		_, err = s.sessionsCollection.UpdateOne(
			sessCtx,
			bson.M{"_id": sessionID},
			bson.M{"$set": bson.M{"updatedAt": time.Now().UTC()}},
		)
		if err != nil {
			return fmt.Errorf("failed to update session timestamp: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return message, nil
}

// GetSessionMessages retrieves all messages for a session (for AI context)
func (s *ChatService) GetSessionMessages(ctx context.Context, sessionID primitive.ObjectID) ([]models.ChatMessage, error) {
	filter := bson.M{"sessionId": sessionID}
	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}}) // Chronological order

	cursor, err := s.messagesCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query session messages: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []models.ChatMessage
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, fmt.Errorf("failed to decode messages: %w", err)
	}

	return messages, nil
}

// SaveToolCall saves a tool call message to the database
// Uses transaction to ensure message insert and session update are atomic
func (s *ChatService) SaveToolCall(ctx context.Context, sessionID primitive.ObjectID, toolCallID, toolName string, args map[string]interface{}, companyID string) (*models.ChatMessage, error) {
	// Layer 3: Validate tool call args size (defense in depth)
	// Tool results can be larger, but tool calls (user input) should be limited
	argsBytes, err := bson.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to validate tool call args: %w", err)
	}
	if len(argsBytes) > config.MaxContentBytes {
		return nil, fmt.Errorf("tool call args exceed maximum size of %d bytes (got %d bytes)", config.MaxContentBytes, len(argsBytes))
	}

	// Verify session exists and user has access (outside transaction)
	_, err = s.GetSession(ctx, sessionID, companyID)
	if err != nil {
		return nil, err
	}

	// Prepare message
	message := &models.ChatMessage{
		ID:        primitive.NewObjectID(),
		SessionID: sessionID,
		Role:      "tool_call",
		Content:   fmt.Sprintf("Tool call: %s", toolName),
		Timestamp: time.Now().UTC(),
		ToolCall: &models.ToolCallData{
			ID:   toolCallID,
			Name: toolName,
			Args: args,
		},
	}

	// Execute both operations in a transaction
	err = s.executeInTransaction(ctx, func(sessCtx mongo.SessionContext) error {
		// Insert tool call message
		_, err := s.messagesCollection.InsertOne(sessCtx, message)
		if err != nil {
			return fmt.Errorf("failed to save tool call: %w", err)
		}

		// Update session's updatedAt timestamp (must succeed or transaction aborts)
		_, err = s.sessionsCollection.UpdateOne(
			sessCtx,
			bson.M{"_id": sessionID},
			bson.M{"$set": bson.M{"updatedAt": time.Now().UTC()}},
		)
		if err != nil {
			return fmt.Errorf("failed to update session timestamp: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return message, nil
}

// SaveToolResult saves a tool result message to the database
// Uses transaction to ensure message insert and session update are atomic
func (s *ChatService) SaveToolResult(ctx context.Context, sessionID primitive.ObjectID, toolCallID, toolName string, output interface{}, errorMsg string, durationMs int64, companyID string) (*models.ChatMessage, error) {
	// Layer 3: Validate tool result output size (defense in depth)
	// Tool results can be larger than user messages (10MB limit vs 1MB)
	if output != nil {
		outputBytes, err := bson.Marshal(output)
		if err != nil {
			return nil, fmt.Errorf("failed to validate tool result output: %w", err)
		}
		if len(outputBytes) > config.MaxToolResultBytes {
			return nil, fmt.Errorf("tool result output exceeds maximum size of %d bytes (got %d bytes)", config.MaxToolResultBytes, len(outputBytes))
		}
	}

	// Verify session exists and user has access (outside transaction)
	_, err := s.GetSession(ctx, sessionID, companyID)
	if err != nil {
		return nil, err
	}

	// Prepare message content
	content := fmt.Sprintf("Tool result: %s", toolName)
	if errorMsg != "" {
		content = fmt.Sprintf("Tool error: %s - %s", toolName, errorMsg)
	}

	// Prepare message
	message := &models.ChatMessage{
		ID:        primitive.NewObjectID(),
		SessionID: sessionID,
		Role:      "tool_result",
		Content:   content,
		Timestamp: time.Now().UTC(),
		ToolResult: &models.ToolResultData{
			ID:         toolCallID,
			Name:       toolName,
			Output:     output,
			Error:      errorMsg,
			DurationMs: durationMs,
		},
	}

	// Execute both operations in a transaction
	err = s.executeInTransaction(ctx, func(sessCtx mongo.SessionContext) error {
		// Insert tool result message
		_, err := s.messagesCollection.InsertOne(sessCtx, message)
		if err != nil {
			return fmt.Errorf("failed to save tool result: %w", err)
		}

		// Update session's updatedAt timestamp (must succeed or transaction aborts)
		_, err = s.sessionsCollection.UpdateOne(
			sessCtx,
			bson.M{"_id": sessionID},
			bson.M{"$set": bson.M{"updatedAt": time.Now().UTC()}},
		)
		if err != nil {
			return fmt.Errorf("failed to update session timestamp: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return message, nil
}

// UpdateSession updates a chat session's title
func (s *ChatService) UpdateSession(ctx context.Context, sessionID primitive.ObjectID, userID, companyID, title string) (*models.ChatSession, error) {
	// Verify session belongs to user and company (authorization)
	session, err := s.GetSession(ctx, sessionID, companyID)
	if err != nil {
		return nil, err
	}

	if session.UserID != userID {
		return nil, fmt.Errorf("unauthorized: session does not belong to user")
	}

	// Validate title
	if title == "" {
		return nil, fmt.Errorf("title cannot be empty")
	}
	if len(title) > 100 {
		return nil, fmt.Errorf("title too long (max 100 characters)")
	}

	// Update the session
	now := time.Now().UTC()
	update := bson.M{
		"$set": bson.M{
			"title":     title,
			"updatedAt": now,
		},
	}

	result, err := s.sessionsCollection.UpdateOne(ctx, bson.M{"_id": sessionID}, update)
	if err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	if result.MatchedCount == 0 {
		return nil, fmt.Errorf("session not found")
	}

	s.logger.Info("Chat session updated",
		zap.String("sessionId", sessionID.Hex()),
		zap.String("userId", userID),
		zap.String("newTitle", title))

	// Return updated session
	session.Title = title
	session.UpdatedAt = now
	return session, nil
}

// SetSessionSubagent sets or clears the active subagent for a chat session
func (s *ChatService) SetSessionSubagent(ctx context.Context, sessionID primitive.ObjectID, subagentID *primitive.ObjectID, companyID string) error {
	filter := bson.M{
		"_id":       sessionID,
		"companyId": companyID,
	}

	update := bson.M{
		"$set": bson.M{
			"activeSubagentId": subagentID,
			"updatedAt":        time.Now().UTC(),
		},
	}

	result, err := s.sessionsCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update session subagent: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("session not found or access denied")
	}

	if subagentID != nil {
		s.logger.Info("Session subagent set",
			zap.String("sessionId", sessionID.Hex()),
			zap.String("subagentId", subagentID.Hex()))
	} else {
		s.logger.Info("Session subagent cleared",
			zap.String("sessionId", sessionID.Hex()))
	}

	return nil
}
