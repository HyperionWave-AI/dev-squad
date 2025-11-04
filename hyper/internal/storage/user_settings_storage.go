package storage

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// ConversationMode represents the display mode for conversations
type ConversationMode string

const (
	ConversationModeDebug   ConversationMode = "debug"
	ConversationModeDefault ConversationMode = "default"
)

// UserSettings represents user preferences and settings
type UserSettings struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID           string             `bson:"userId" json:"userId"`
	CompanyID        string             `bson:"companyId" json:"companyId"`
	ConversationMode ConversationMode   `bson:"conversationMode" json:"conversationMode"`
	CreatedAt        time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// UserSettingsStorage handles user settings persistence
type UserSettingsStorage struct {
	db         *mongo.Database
	collection *mongo.Collection
	logger     *zap.Logger
}

// NewUserSettingsStorage creates a new UserSettingsStorage instance
func NewUserSettingsStorage(db *mongo.Database, logger *zap.Logger) (*UserSettingsStorage, error) {
	collection := db.Collection("user_settings")

	storage := &UserSettingsStorage{
		db:         db,
		collection: collection,
		logger:     logger,
	}

	// Create indexes
	if err := storage.createIndexes(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return storage, nil
}

// createIndexes creates necessary indexes for the user settings collection
func (s *UserSettingsStorage) createIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "companyId", Value: 1},
			},
			Options: options.Index().SetUnique(true).SetName("unique_user_company"),
		},
		{
			Keys:    bson.D{{Key: "updatedAt", Value: -1}},
			Options: options.Index().SetName("updated_at_desc"),
		},
	}

	_, err := s.collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	s.logger.Info("User settings indexes created successfully")
	return nil
}

// GetUserSettings retrieves user settings or creates default settings if not found
func (s *UserSettingsStorage) GetUserSettings(ctx context.Context, userID, companyID string) (*UserSettings, error) {
	filter := bson.M{
		"userId":    userID,
		"companyId": companyID,
	}

	var settings UserSettings
	err := s.collection.FindOne(ctx, filter).Decode(&settings)

	if err == mongo.ErrNoDocuments {
		// Create default settings
		settings = UserSettings{
			ID:               primitive.NewObjectID(),
			UserID:           userID,
			CompanyID:        companyID,
			ConversationMode: ConversationModeDefault, // Default to friendly mode
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		_, insertErr := s.collection.InsertOne(ctx, settings)
		if insertErr != nil {
			return nil, fmt.Errorf("failed to create default settings: %w", insertErr)
		}

		s.logger.Info("Created default user settings",
			zap.String("userId", userID),
			zap.String("companyId", companyID),
			zap.String("conversationMode", string(ConversationModeDefault)))

		return &settings, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get user settings: %w", err)
	}

	return &settings, nil
}

// UpdateConversationMode updates the user's conversation mode preference
func (s *UserSettingsStorage) UpdateConversationMode(ctx context.Context, userID, companyID string, mode ConversationMode) error {
	// Validate mode
	if mode != ConversationModeDebug && mode != ConversationModeDefault {
		return fmt.Errorf("invalid conversation mode: %s (must be 'debug' or 'default')", mode)
	}

	filter := bson.M{
		"userId":    userID,
		"companyId": companyID,
	}

	update := bson.M{
		"$set": bson.M{
			"conversationMode": mode,
			"updatedAt":        time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	result, err := s.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to update conversation mode: %w", err)
	}

	if result.UpsertedCount > 0 {
		s.logger.Info("Created new user settings with conversation mode",
			zap.String("userId", userID),
			zap.String("companyId", companyID),
			zap.String("mode", string(mode)))
	} else {
		s.logger.Info("Updated conversation mode",
			zap.String("userId", userID),
			zap.String("companyId", companyID),
			zap.String("mode", string(mode)))
	}

	return nil
}

// UpdateUserSettings updates user settings
func (s *UserSettingsStorage) UpdateUserSettings(ctx context.Context, settings *UserSettings) error {
	filter := bson.M{
		"userId":    settings.UserID,
		"companyId": settings.CompanyID,
	}

	settings.UpdatedAt = time.Now()

	update := bson.M{
		"$set": settings,
	}

	opts := options.Update().SetUpsert(true)
	_, err := s.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to update user settings: %w", err)
	}

	return nil
}

// DeleteUserSettings deletes user settings
func (s *UserSettingsStorage) DeleteUserSettings(ctx context.Context, userID, companyID string) error {
	filter := bson.M{
		"userId":    userID,
		"companyId": companyID,
	}

	result, err := s.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete user settings: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("user settings not found")
	}

	s.logger.Info("Deleted user settings",
		zap.String("userId", userID),
		zap.String("companyId", companyID))

	return nil
}
