package services

import (
	"context"
	"fmt"
	"time"

	"hyperion-coordinator/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// ChatService manages chat sessions and messages with MongoDB storage
type ChatService struct {
	sessionsCollection *mongo.Collection
	messagesCollection *mongo.Collection
	logger             *zap.Logger
}

// NewChatService creates a new chat service instance
func NewChatService(db *mongo.Database, logger *zap.Logger) (*ChatService, error) {
	service := &ChatService{
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

// CreateSession creates a new chat session for a user
func (s *ChatService) CreateSession(ctx context.Context, userID, companyID, title string) (*models.ChatSession, error) {
	now := time.Now().UTC()
	session := &models.ChatSession{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		CompanyID: companyID,
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err := s.sessionsCollection.InsertOne(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat session: %w", err)
	}

	s.logger.Info("Chat session created",
		zap.String("sessionId", session.ID.Hex()),
		zap.String("userId", userID),
		zap.String("companyId", companyID))

	return session, nil
}

// CreateSessionWithID creates a new chat session with a specific ID (for session recovery)
func (s *ChatService) CreateSessionWithID(ctx context.Context, sessionID primitive.ObjectID, userID, companyID, title string) (*models.ChatSession, error) {
	now := time.Now().UTC()
	session := &models.ChatSession{
		ID:        sessionID, // Use provided ID instead of generating new one
		UserID:    userID,
		CompanyID: companyID,
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err := s.sessionsCollection.InsertOne(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat session with ID: %w", err)
	}

	s.logger.Info("Chat session created with specific ID",
		zap.String("sessionId", session.ID.Hex()),
		zap.String("userId", userID),
		zap.String("companyId", companyID))

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
func (s *ChatService) DeleteSession(ctx context.Context, sessionID primitive.ObjectID, userID, companyID string) error {
	// Verify session belongs to user and company (authorization)
	session, err := s.GetSession(ctx, sessionID, companyID)
	if err != nil {
		return err
	}

	if session.UserID != userID {
		return fmt.Errorf("unauthorized: session does not belong to user")
	}

	// Delete all messages first
	_, err = s.messagesCollection.DeleteMany(ctx, bson.M{"sessionId": sessionID})
	if err != nil {
		return fmt.Errorf("failed to delete session messages: %w", err)
	}

	// Delete the session
	result, err := s.sessionsCollection.DeleteOne(ctx, bson.M{"_id": sessionID})
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("session not found")
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
func (s *ChatService) SaveMessage(ctx context.Context, sessionID primitive.ObjectID, role, content string, companyID string) (*models.ChatMessage, error) {
	// Verify session exists and user has access
	_, err := s.GetSession(ctx, sessionID, companyID)
	if err != nil {
		return nil, err
	}

	message := &models.ChatMessage{
		ID:        primitive.NewObjectID(),
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		Timestamp: time.Now().UTC(),
	}

	_, err = s.messagesCollection.InsertOne(ctx, message)
	if err != nil {
		return nil, fmt.Errorf("failed to save message: %w", err)
	}

	// Update session's updatedAt timestamp
	_, err = s.sessionsCollection.UpdateOne(
		ctx,
		bson.M{"_id": sessionID},
		bson.M{"$set": bson.M{"updatedAt": time.Now().UTC()}},
	)
	if err != nil {
		s.logger.Warn("Failed to update session timestamp", zap.Error(err))
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
func (s *ChatService) SaveToolCall(ctx context.Context, sessionID primitive.ObjectID, toolCallID, toolName string, args map[string]interface{}, companyID string) (*models.ChatMessage, error) {
	// Verify session exists and user has access
	_, err := s.GetSession(ctx, sessionID, companyID)
	if err != nil {
		return nil, err
	}

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

	_, err = s.messagesCollection.InsertOne(ctx, message)
	if err != nil {
		return nil, fmt.Errorf("failed to save tool call: %w", err)
	}

	// Update session's updatedAt timestamp
	_, err = s.sessionsCollection.UpdateOne(
		ctx,
		bson.M{"_id": sessionID},
		bson.M{"$set": bson.M{"updatedAt": time.Now().UTC()}},
	)
	if err != nil {
		s.logger.Warn("Failed to update session timestamp", zap.Error(err))
	}

	return message, nil
}

// SaveToolResult saves a tool result message to the database
func (s *ChatService) SaveToolResult(ctx context.Context, sessionID primitive.ObjectID, toolCallID, toolName string, output interface{}, errorMsg string, durationMs int64, companyID string) (*models.ChatMessage, error) {
	// Verify session exists and user has access
	_, err := s.GetSession(ctx, sessionID, companyID)
	if err != nil {
		return nil, err
	}

	content := fmt.Sprintf("Tool result: %s", toolName)
	if errorMsg != "" {
		content = fmt.Sprintf("Tool error: %s - %s", toolName, errorMsg)
	}

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

	_, err = s.messagesCollection.InsertOne(ctx, message)
	if err != nil {
		return nil, fmt.Errorf("failed to save tool result: %w", err)
	}

	// Update session's updatedAt timestamp
	_, err = s.sessionsCollection.UpdateOne(
		ctx,
		bson.M{"_id": sessionID},
		bson.M{"$set": bson.M{"updatedAt": time.Now().UTC()}},
	)
	if err != nil {
		s.logger.Warn("Failed to update session timestamp", zap.Error(err))
	}

	return message, nil
}
