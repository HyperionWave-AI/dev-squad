package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// SubchatStatus represents the status of a subchat
type SubchatStatus string

const (
	SubchatStatusActive    SubchatStatus = "active"
	SubchatStatusCompleted SubchatStatus = "completed"
	SubchatStatusFailed    SubchatStatus = "failed"
)

// Subchat represents a parallel workflow session
type Subchat struct {
	ID             string        `bson:"_id" json:"id"`
	ParentChatID   string        `bson:"parentChatId" json:"parentChatId"`
	SubagentName   string        `bson:"subagentName" json:"subagentName"`
	AssignedTaskID *string       `bson:"assignedTaskId,omitempty" json:"assignedTaskId,omitempty"`
	AssignedTodoID *string       `bson:"assignedTodoId,omitempty" json:"assignedTodoId,omitempty"`
	Status         SubchatStatus `bson:"status" json:"status"`
	CreatedAt      time.Time     `bson:"createdAt" json:"createdAt"`
	UpdatedAt      time.Time     `bson:"updatedAt" json:"updatedAt"`
}

// Subagent represents an available specialist agent
type Subagent struct {
	ID           string   `bson:"_id" json:"id"`
	Name         string   `bson:"name" json:"name"`
	Description  string   `bson:"description" json:"description"`
	SystemPrompt string   `bson:"systemPrompt" json:"systemPrompt"`
	Tools        []string `bson:"tools,omitempty" json:"tools,omitempty"`
	Category     string   `bson:"category,omitempty" json:"category,omitempty"`
}

// SubchatStorage handles subchat persistence
type SubchatStorage struct {
	collection        *mongo.Collection
	subagentCollection *mongo.Collection
	logger            *zap.Logger
}

// NewSubchatStorage creates a new subchat storage
func NewSubchatStorage(db *mongo.Database, logger *zap.Logger) *SubchatStorage {
	return &SubchatStorage{
		collection:        db.Collection("subchats"),
		subagentCollection: db.Collection("subagents"),
		logger:            logger,
	}
}

// CreateSubchat creates a new subchat
func (s *SubchatStorage) CreateSubchat(parentChatID, subagentName string, taskID, todoID *string) (*Subchat, error) {
	subchat := &Subchat{
		ID:             uuid.New().String(),
		ParentChatID:   parentChatID,
		SubagentName:   subagentName,
		AssignedTaskID: taskID,
		AssignedTodoID: todoID,
		Status:         SubchatStatusActive,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := s.collection.InsertOne(ctx, subchat)
	if err != nil {
		s.logger.Error("Failed to create subchat", zap.Error(err))
		return nil, fmt.Errorf("failed to create subchat: %w", err)
	}

	s.logger.Info("Created subchat",
		zap.String("subchatId", subchat.ID),
		zap.String("parentChatId", parentChatID),
		zap.String("subagentName", subagentName))

	return subchat, nil
}

// GetSubchat retrieves a subchat by ID
func (s *SubchatStorage) GetSubchat(id string) (*Subchat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var subchat Subchat
	err := s.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&subchat)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("subchat not found: %s", id)
		}
		s.logger.Error("Failed to get subchat", zap.String("id", id), zap.Error(err))
		return nil, fmt.Errorf("failed to get subchat: %w", err)
	}

	return &subchat, nil
}

// GetSubchatsByParent retrieves all subchats for a parent chat
func (s *SubchatStorage) GetSubchatsByParent(parentChatID string) ([]*Subchat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := s.collection.Find(ctx, bson.M{"parentChatId": parentChatID})
	if err != nil {
		s.logger.Error("Failed to query subchats", zap.String("parentChatId", parentChatID), zap.Error(err))
		return nil, fmt.Errorf("failed to query subchats: %w", err)
	}
	defer cursor.Close(ctx)

	var subchats []*Subchat
	if err := cursor.All(ctx, &subchats); err != nil {
		s.logger.Error("Failed to decode subchats", zap.Error(err))
		return nil, fmt.Errorf("failed to decode subchats: %w", err)
	}

	return subchats, nil
}

// UpdateSubchatStatus updates the status of a subchat
func (s *SubchatStorage) UpdateSubchatStatus(id string, status SubchatStatus) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := s.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$set": bson.M{
				"status":    status,
				"updatedAt": time.Now(),
			},
		},
	)
	if err != nil {
		s.logger.Error("Failed to update subchat status", zap.String("id", id), zap.Error(err))
		return fmt.Errorf("failed to update subchat status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("subchat not found: %s", id)
	}

	s.logger.Info("Updated subchat status",
		zap.String("subchatId", id),
		zap.String("status", string(status)))

	return nil
}

// ListSubagents retrieves all available subagents
func (s *SubchatStorage) ListSubagents() ([]*Subagent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := s.subagentCollection.Find(ctx, bson.M{})
	if err != nil {
		s.logger.Error("Failed to query subagents", zap.Error(err))
		return nil, fmt.Errorf("failed to query subagents: %w", err)
	}
	defer cursor.Close(ctx)

	var subagents []*Subagent
	if err := cursor.All(ctx, &subagents); err != nil {
		s.logger.Error("Failed to decode subagents", zap.Error(err))
		return nil, fmt.Errorf("failed to decode subagents: %w", err)
	}

	return subagents, nil
}

// GetSubagent retrieves a subagent by name
func (s *SubchatStorage) GetSubagent(name string) (*Subagent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var subagent Subagent
	err := s.subagentCollection.FindOne(ctx, bson.M{"name": name}).Decode(&subagent)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("subagent not found: %s", name)
		}
		s.logger.Error("Failed to get subagent", zap.String("name", name), zap.Error(err))
		return nil, fmt.Errorf("failed to get subagent: %w", err)
	}

	return &subagent, nil
}
