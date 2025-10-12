package services

import (
	"context"
	"fmt"
	"time"

	"hyperion-coordinator/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// SubchatService handles subchat operations
type SubchatService struct {
	collection *mongo.Collection
}

// NewSubchatService creates a new subchat service
func NewSubchatService(db *mongo.Database) *SubchatService {
	return &SubchatService{
		collection: db.Collection("chat_sessions"),
	}
}

// CreateSubchat creates a new subchat linked to a parent chat
func (s *SubchatService) CreateSubchat(ctx context.Context, userID, companyID, title, parentChatID, subagentName, taskID, todoID string) (*models.ChatSession, error) {
	// Validate parent chat exists
	parentObjID, err := primitive.ObjectIDFromHex(parentChatID)
	if err != nil {
		return nil, fmt.Errorf("invalid parent chat ID format: %w", err)
	}

	var parentChat models.ChatSession
	err = s.collection.FindOne(ctx, bson.M{
		"_id":       parentObjID,
		"companyId": companyID, // Company isolation
	}).Decode(&parentChat)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("parent chat not found with ID '%s'", parentChatID)
		}
		return nil, fmt.Errorf("failed to verify parent chat: %w", err)
	}

	// Create subchat document
	now := time.Now()
	subchat := &models.ChatSession{
		ID:             primitive.NewObjectID(),
		UserID:         userID,
		CompanyID:      companyID,
		Title:          title,
		CreatedAt:      now,
		UpdatedAt:      now,
		ParentChatID:   &parentObjID,
		SubagentName:   subagentName,
		AssignedTaskID: taskID,
		AssignedTodoID: todoID,
	}

	_, err = s.collection.InsertOne(ctx, subchat)
	if err != nil {
		return nil, fmt.Errorf("failed to create subchat: %w", err)
	}

	return subchat, nil
}

// GetSubchatsByParent retrieves all subchats for a parent chat
func (s *SubchatService) GetSubchatsByParent(ctx context.Context, parentChatID, companyID string) ([]*models.ChatSession, error) {
	parentObjID, err := primitive.ObjectIDFromHex(parentChatID)
	if err != nil {
		return nil, fmt.Errorf("invalid parent chat ID format: %w", err)
	}

	filter := bson.M{
		"parentChatId": parentObjID,
		"companyId":    companyID, // Company isolation
	}

	cursor, err := s.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query subchats: %w", err)
	}
	defer cursor.Close(ctx)

	var subchats []*models.ChatSession
	if err = cursor.All(ctx, &subchats); err != nil {
		return nil, fmt.Errorf("failed to decode subchats: %w", err)
	}

	return subchats, nil
}

// ListSubchats retrieves all subchats for a company (with optional filtering)
func (s *SubchatService) ListSubchats(ctx context.Context, companyID string) ([]*models.ChatSession, error) {
	filter := bson.M{
		"companyId":    companyID, // Company isolation
		"parentChatId": bson.M{"$exists": true, "$ne": nil}, // Only subchats
	}

	cursor, err := s.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query subchats: %w", err)
	}
	defer cursor.Close(ctx)

	var subchats []*models.ChatSession
	if err = cursor.All(ctx, &subchats); err != nil {
		return nil, fmt.Errorf("failed to decode subchats: %w", err)
	}

	return subchats, nil
}

// GetSubchatByID retrieves a single subchat by ID
func (s *SubchatService) GetSubchatByID(ctx context.Context, subchatID, companyID string) (*models.ChatSession, error) {
	objID, err := primitive.ObjectIDFromHex(subchatID)
	if err != nil {
		return nil, fmt.Errorf("invalid subchat ID format: %w", err)
	}

	var subchat models.ChatSession
	err = s.collection.FindOne(ctx, bson.M{
		"_id":          objID,
		"companyId":    companyID, // Company isolation
		"parentChatId": bson.M{"$exists": true, "$ne": nil}, // Must be a subchat
	}).Decode(&subchat)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("subchat not found with ID '%s'", subchatID)
		}
		return nil, fmt.Errorf("failed to get subchat: %w", err)
	}

	return &subchat, nil
}
