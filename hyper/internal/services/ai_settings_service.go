package services

import (
	"context"
	"fmt"
	"time"

	"hyper/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// AISettingsService manages system prompts and subagents with MongoDB storage
type AISettingsService struct {
	systemPromptsCollection *mongo.Collection
	subagentsCollection     *mongo.Collection
	logger                  *zap.Logger
}

// NewAISettingsService creates a new AI settings service instance
func NewAISettingsService(db *mongo.Database, logger *zap.Logger) (*AISettingsService, error) {
	service := &AISettingsService{
		systemPromptsCollection: db.Collection("system_prompts"),
		subagentsCollection:     db.Collection("subagents"),
		logger:                  logger,
	}

	// Create indexes
	ctx := context.Background()

	// Index on system_prompts: {userId, companyId} for user prompt queries
	_, err := service.systemPromptsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "userId", Value: 1},
			{Key: "companyId", Value: 1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create system_prompts user index: %w", err)
	}

	// Index on subagents: {userId, companyId} for user subagent queries
	_, err = service.subagentsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "userId", Value: 1},
			{Key: "companyId", Value: 1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create subagents user index: %w", err)
	}

	// Index on subagents: {companyId} for company-level isolation
	_, err = service.subagentsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "companyId", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create subagents company index: %w", err)
	}

	logger.Info("AI settings service initialized with MongoDB indexes")
	return service, nil
}

// GetSystemPrompt retrieves the system prompt for a user
func (s *AISettingsService) GetSystemPrompt(ctx context.Context, userID, companyID string) (string, error) {
	var systemPrompt models.SystemPrompt
	filter := bson.M{
		"userId":    userID,
		"companyId": companyID,
	}

	err := s.systemPromptsCollection.FindOne(ctx, filter).Decode(&systemPrompt)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Return empty string if no custom prompt exists (not an error)
			return "", nil
		}
		return "", fmt.Errorf("failed to retrieve system prompt: %w", err)
	}

	return systemPrompt.Prompt, nil
}

// UpdateSystemPrompt updates or creates the system prompt for a user
func (s *AISettingsService) UpdateSystemPrompt(ctx context.Context, userID, companyID, prompt string) error {
	filter := bson.M{
		"userId":    userID,
		"companyId": companyID,
	}

	update := bson.M{
		"$set": bson.M{
			"userId":    userID,
			"companyId": companyID,
			"prompt":    prompt,
			"updatedAt": time.Now().UTC(),
		},
	}

	opts := options.Update().SetUpsert(true)

	result, err := s.systemPromptsCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to update system prompt: %w", err)
	}

	if result.UpsertedCount > 0 {
		s.logger.Info("System prompt created",
			zap.String("userId", userID),
			zap.String("companyId", companyID))
	} else {
		s.logger.Info("System prompt updated",
			zap.String("userId", userID),
			zap.String("companyId", companyID))
	}

	return nil
}

// ListSubagents retrieves all subagents for a user within their company
func (s *AISettingsService) ListSubagents(ctx context.Context, userID, companyID string) ([]models.Subagent, error) {
	filter := bson.M{
		"userId":    userID,
		"companyId": companyID,
	}

	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}) // Latest first

	cursor, err := s.subagentsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query subagents: %w", err)
	}
	defer cursor.Close(ctx)

	var subagents []models.Subagent
	if err := cursor.All(ctx, &subagents); err != nil {
		return nil, fmt.Errorf("failed to decode subagents: %w", err)
	}

	return subagents, nil
}

// GetSubagent retrieves a specific subagent by ID
func (s *AISettingsService) GetSubagent(ctx context.Context, id primitive.ObjectID, companyID string) (*models.Subagent, error) {
	var subagent models.Subagent
	filter := bson.M{
		"_id":       id,
		"companyId": companyID, // Company-level isolation
	}

	err := s.subagentsCollection.FindOne(ctx, filter).Decode(&subagent)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("subagent not found or access denied")
		}
		return nil, fmt.Errorf("failed to retrieve subagent: %w", err)
	}

	return &subagent, nil
}

// CreateSubagent creates a new subagent for a user
func (s *AISettingsService) CreateSubagent(ctx context.Context, userID, companyID, name, description, systemPrompt string) (*models.Subagent, error) {
	now := time.Now().UTC()
	subagent := &models.Subagent{
		ID:           primitive.NewObjectID(),
		UserID:       userID,
		CompanyID:    companyID,
		Name:         name,
		Description:  description,
		SystemPrompt: systemPrompt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	_, err := s.subagentsCollection.InsertOne(ctx, subagent)
	if err != nil {
		return nil, fmt.Errorf("failed to create subagent: %w", err)
	}

	s.logger.Info("Subagent created",
		zap.String("subagentId", subagent.ID.Hex()),
		zap.String("name", name),
		zap.String("userId", userID),
		zap.String("companyId", companyID))

	return subagent, nil
}

// UpdateSubagent updates an existing subagent
func (s *AISettingsService) UpdateSubagent(ctx context.Context, id primitive.ObjectID, userID, companyID, name, description, systemPrompt string) (*models.Subagent, error) {
	// Verify subagent exists and belongs to user
	existingSubagent, err := s.GetSubagent(ctx, id, companyID)
	if err != nil {
		return nil, err
	}

	if existingSubagent.UserID != userID {
		return nil, fmt.Errorf("unauthorized: subagent does not belong to user")
	}

	update := bson.M{
		"$set": bson.M{
			"name":         name,
			"description":  description,
			"systemPrompt": systemPrompt,
			"updatedAt":    time.Now().UTC(),
		},
	}

	result, err := s.subagentsCollection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return nil, fmt.Errorf("failed to update subagent: %w", err)
	}

	if result.MatchedCount == 0 {
		return nil, fmt.Errorf("subagent not found")
	}

	s.logger.Info("Subagent updated",
		zap.String("subagentId", id.Hex()),
		zap.String("userId", userID))

	// Retrieve updated subagent
	return s.GetSubagent(ctx, id, companyID)
}

// DeleteSubagent deletes a subagent
func (s *AISettingsService) DeleteSubagent(ctx context.Context, id primitive.ObjectID, userID, companyID string) error {
	// Verify subagent belongs to user and company (authorization)
	subagent, err := s.GetSubagent(ctx, id, companyID)
	if err != nil {
		return err
	}

	if subagent.UserID != userID {
		return fmt.Errorf("unauthorized: subagent does not belong to user")
	}

	// Delete the subagent
	result, err := s.subagentsCollection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete subagent: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("subagent not found")
	}

	s.logger.Info("Subagent deleted",
		zap.String("subagentId", id.Hex()),
		zap.String("userId", userID))

	return nil
}
