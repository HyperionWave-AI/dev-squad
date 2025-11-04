package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// Reflection represents a metacognitive record (decision, outcome, or lesson)
type Reflection struct {
	ID                 string                 `json:"id" bson:"_id"`
	Type               string                 `json:"type" bson:"type"` // "decision", "outcome", "lesson", "causal_link"
	ChatID             string                 `json:"chatId" bson:"chatId"`
	TaskID             string                 `json:"taskId,omitempty" bson:"taskId,omitempty"`
	Timestamp          time.Time              `json:"timestamp" bson:"timestamp"`
	Data               map[string]interface{} `json:"data" bson:"data"`
	Confidence         float64                `json:"confidence,omitempty" bson:"confidence,omitempty"`
	Tags               []string               `json:"tags" bson:"tags"`
	RelatedReflections []string               `json:"relatedReflections,omitempty" bson:"relatedReflections,omitempty"`
}

// ExperienceIndex represents a pattern for fast lookup
type ExperienceIndex struct {
	ID            string    `json:"id" bson:"_id"`
	Pattern       string    `json:"pattern" bson:"pattern"`
	Contexts      []string  `json:"contexts" bson:"contexts"`
	AvgConfidence float64   `json:"avgConfidence" bson:"avgConfidence"`
	Occurrences   int       `json:"occurrences" bson:"occurrences"`
	LastSeen      time.Time `json:"lastSeen" bson:"lastSeen"`
	RelatedLessons []string `json:"relatedLessons" bson:"relatedLessons"`
}

// ErrorInstance represents a single error occurrence with context
type ErrorInstance struct {
	Timestamp  time.Time              `json:"timestamp" bson:"timestamp"`
	Message    string                 `json:"message" bson:"message"`
	StackTrace string                 `json:"stackTrace,omitempty" bson:"stackTrace,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty" bson:"context,omitempty"`
}

// ErrorPattern tracks recurring error patterns for lesson extraction
type ErrorPattern struct {
	ID             string          `json:"id" bson:"_id"`
	Signature      string          `json:"signature" bson:"signature"` // Hash of error type + normalized message
	ErrorType      string          `json:"errorType" bson:"errorType"`
	MessagePattern string          `json:"messagePattern" bson:"messagePattern"`
	Occurrences    int             `json:"occurrences" bson:"occurrences"`
	FirstSeen      time.Time       `json:"firstSeen" bson:"firstSeen"`
	LastSeen       time.Time       `json:"lastSeen" bson:"lastSeen"`
	RecentErrors   []ErrorInstance `json:"recentErrors" bson:"recentErrors"` // Keep last 5
	LessonExtracted bool           `json:"lessonExtracted" bson:"lessonExtracted"`
	RelatedLesson  string          `json:"relatedLesson,omitempty" bson:"relatedLesson,omitempty"`
}

// ReflectionStorage provides storage for metacognitive data
type ReflectionStorage struct {
	reflectionsCollection     *mongo.Collection
	experienceIndexCollection *mongo.Collection
	errorPatternsCollection   *mongo.Collection
	logger                    *zap.Logger
}

// NewReflectionStorage creates a new reflection storage instance
func NewReflectionStorage(db *mongo.Database, logger *zap.Logger) (*ReflectionStorage, error) {
	storage := &ReflectionStorage{
		reflectionsCollection:     db.Collection("reflections"),
		experienceIndexCollection: db.Collection("experience_index"),
		errorPatternsCollection:   db.Collection("error_patterns"),
		logger:                    logger,
	}

	// Create indexes
	ctx := context.Background()

	// MongoDB automatically creates a unique index on _id, so we don't need to create it manually

	// Index on type for filtering
	_, err := storage.reflectionsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "type", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create type index: %w", err)
	}

	// Index on chatId for filtering
	_, err = storage.reflectionsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "chatId", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create chatId index: %w", err)
	}

	// Index on taskId for filtering
	_, err = storage.reflectionsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "taskId", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create taskId index: %w", err)
	}

	// Index on pattern for experience index
	_, err = storage.experienceIndexCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "pattern", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pattern index: %w", err)
	}

	// Index on signature for error patterns (unique - one entry per error pattern)
	_, err = storage.errorPatternsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "signature", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create error signature index: %w", err)
	}

	logger.Info("Reflection storage initialized successfully")
	return storage, nil
}

// StoreReflection stores a reflection in MongoDB
func (rs *ReflectionStorage) StoreReflection(reflection *Reflection) (string, error) {
	ctx := context.Background()

	if reflection.ID == "" {
		reflection.ID = uuid.New().String()
	}

	if reflection.Timestamp.IsZero() {
		reflection.Timestamp = time.Now().UTC()
	}

	_, err := rs.reflectionsCollection.InsertOne(ctx, reflection)
	if err != nil {
		return "", fmt.Errorf("failed to store reflection: %w", err)
	}

	rs.logger.Info("Stored reflection",
		zap.String("id", reflection.ID),
		zap.String("type", reflection.Type),
		zap.String("chatId", reflection.ChatID))

	return reflection.ID, nil
}

// GetReflectionsByType retrieves all reflections of a specific type
func (rs *ReflectionStorage) GetReflectionsByType(reflectionType string) ([]*Reflection, error) {
	ctx := context.Background()

	filter := bson.M{"type": reflectionType}
	cursor, err := rs.reflectionsCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query reflections by type: %w", err)
	}
	defer cursor.Close(ctx)

	var reflections []*Reflection
	if err := cursor.All(ctx, &reflections); err != nil {
		return nil, fmt.Errorf("failed to decode reflections: %w", err)
	}

	return reflections, nil
}

// GetReflectionsByTask retrieves all reflections for a specific task
func (rs *ReflectionStorage) GetReflectionsByTask(taskID string) ([]*Reflection, error) {
	ctx := context.Background()

	filter := bson.M{"taskId": taskID}
	cursor, err := rs.reflectionsCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query reflections by task: %w", err)
	}
	defer cursor.Close(ctx)

	var reflections []*Reflection
	if err := cursor.All(ctx, &reflections); err != nil {
		return nil, fmt.Errorf("failed to decode reflections: %w", err)
	}

	return reflections, nil
}

// GetReflectionsByChat retrieves all reflections for a specific chat
func (rs *ReflectionStorage) GetReflectionsByChat(chatID string) ([]*Reflection, error) {
	ctx := context.Background()

	filter := bson.M{"chatId": chatID}
	cursor, err := rs.reflectionsCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query reflections by chat: %w", err)
	}
	defer cursor.Close(ctx)

	var reflections []*Reflection
	if err := cursor.All(ctx, &reflections); err != nil {
		return nil, fmt.Errorf("failed to decode reflections: %w", err)
	}

	return reflections, nil
}

// GetReflectionByID retrieves a specific reflection by ID
func (rs *ReflectionStorage) GetReflectionByID(id string) (*Reflection, error) {
	ctx := context.Background()

	var reflection Reflection
	err := rs.reflectionsCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&reflection)
	if err != nil {
		return nil, fmt.Errorf("failed to get reflection by ID: %w", err)
	}

	return &reflection, nil
}

// LinkReflections creates a bidirectional link between two reflections (e.g., decision and outcome)
func (rs *ReflectionStorage) LinkReflections(reflection1ID, reflection2ID string) error {
	ctx := context.Background()

	// Add reflection2 to reflection1's related list
	_, err := rs.reflectionsCollection.UpdateOne(
		ctx,
		bson.M{"_id": reflection1ID},
		bson.M{"$addToSet": bson.M{"relatedReflections": reflection2ID}},
	)
	if err != nil {
		return fmt.Errorf("failed to link reflection1 to reflection2: %w", err)
	}

	// Add reflection1 to reflection2's related list
	_, err = rs.reflectionsCollection.UpdateOne(
		ctx,
		bson.M{"_id": reflection2ID},
		bson.M{"$addToSet": bson.M{"relatedReflections": reflection1ID}},
	)
	if err != nil {
		return fmt.Errorf("failed to link reflection2 to reflection1: %w", err)
	}

	rs.logger.Info("Linked reflections",
		zap.String("reflection1", reflection1ID),
		zap.String("reflection2", reflection2ID))

	return nil
}

// QueryPatterns searches experience index for matching patterns
func (rs *ReflectionStorage) QueryPatterns(pattern string) ([]*ExperienceIndex, error) {
	ctx := context.Background()

	// Case-insensitive regex search on pattern field
	filter := bson.M{"pattern": bson.M{"$regex": pattern, "$options": "i"}}
	cursor, err := rs.experienceIndexCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query patterns: %w", err)
	}
	defer cursor.Close(ctx)

	var patterns []*ExperienceIndex
	if err := cursor.All(ctx, &patterns); err != nil {
		return nil, fmt.Errorf("failed to decode patterns: %w", err)
	}

	return patterns, nil
}

// UpsertExperienceIndex updates or inserts a pattern in the experience index
func (rs *ReflectionStorage) UpsertExperienceIndex(pattern string, contextStr string, lessonID string, confidence float64) error {
	ctx := context.Background()

	filter := bson.M{"pattern": pattern}
	update := bson.M{
		"$set": bson.M{
			"lastSeen": time.Now().UTC(),
		},
		"$setOnInsert": bson.M{
			"_id":           uuid.New().String(),
			"avgConfidence": confidence,
		},
		"$inc": bson.M{
			"occurrences": 1,
		},
		"$addToSet": bson.M{
			"contexts":       contextStr,
			"relatedLessons": lessonID,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := rs.experienceIndexCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert experience index: %w", err)
	}

	rs.logger.Info("Updated experience index",
		zap.String("pattern", pattern),
		zap.String("context", contextStr))

	return nil
}

// SearchLessonsBy Text performs text-based search across lesson content
func (rs *ReflectionStorage) SearchLessonsByText(query string, limit int) ([]*Reflection, error) {
	ctx := context.Background()

	// Create a regex pattern for case-insensitive search across multiple fields
	regexPattern := fmt.Sprintf("(?i)%s", query)

	filter := bson.M{
		"type": "lesson",
		"$or": []bson.M{
			{"data.patternName": bson.M{"$regex": regexPattern}},
			{"data.problem": bson.M{"$regex": regexPattern}},
			{"data.solution": bson.M{"$regex": regexPattern}},
			{"data.context": bson.M{"$regex": regexPattern}},
			{"data.antipattern": bson.M{"$regex": regexPattern}},
		},
	}

	opts := options.Find().SetLimit(int64(limit))
	cursor, err := rs.reflectionsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search lessons: %w", err)
	}
	defer cursor.Close(ctx)

	var lessons []*Reflection
	if err := cursor.All(ctx, &lessons); err != nil {
		return nil, fmt.Errorf("failed to decode lessons: %w", err)
	}

	rs.logger.Info("Text search completed",
		zap.String("query", query),
		zap.Int("results", len(lessons)))

	return lessons, nil
}

// RecordError tracks an error occurrence for automatic lesson extraction
func (rs *ReflectionStorage) RecordError(errorType, message, stackTrace string, errorContext map[string]interface{}) (string, bool, error) {
	ctx := context.Background()

	// Create error signature (simple hash based on error type and normalized message)
	// Normalize by removing variable parts (IDs, timestamps, etc.)
	signature := fmt.Sprintf("%s:%s", errorType, normalizeErrorMessage(message))

	// Create error instance
	instance := ErrorInstance{
		Timestamp:  time.Now().UTC(),
		Message:    message,
		StackTrace: stackTrace,
		Context:    errorContext,
	}

	// Try to update existing pattern, or insert new one
	filter := bson.M{"signature": signature}
	update := bson.M{
		"$set": bson.M{
			"lastSeen":       time.Now().UTC(),
			"messagePattern": message,
			"errorType":      errorType,
		},
		"$setOnInsert": bson.M{
			"_id":             uuid.New().String(),
			"firstSeen":       time.Now().UTC(),
			"lessonExtracted": false,
		},
		"$inc": bson.M{
			"occurrences": 1,
		},
		"$push": bson.M{
			"recentErrors": bson.M{
				"$each":  []ErrorInstance{instance},
				"$slice": -5, // Keep only last 5 errors
			},
		},
	}

	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var result ErrorPattern
	err := rs.errorPatternsCollection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&result)
	if err != nil {
		return "", false, fmt.Errorf("failed to record error: %w", err)
	}

	// Check if we should suggest lesson extraction
	shouldSuggest := result.Occurrences >= 2 && !result.LessonExtracted

	rs.logger.Info("Error recorded",
		zap.String("signature", signature),
		zap.Int("occurrences", result.Occurrences),
		zap.Bool("shouldSuggestLesson", shouldSuggest))

	return result.ID, shouldSuggest, nil
}

// normalizeErrorMessage removes variable parts from error messages for pattern matching
func normalizeErrorMessage(message string) string {
	// TODO: Implement more sophisticated normalization
	// For now, just use the first 100 chars or until we hit something that looks like a variable
	if len(message) > 100 {
		return message[:100]
	}
	return message
}

// ShouldSuggestLesson checks if an error pattern warrants lesson extraction
func (rs *ReflectionStorage) ShouldSuggestLesson(errorPatternID string) (bool, error) {
	ctx := context.Background()

	var pattern ErrorPattern
	err := rs.errorPatternsCollection.FindOne(ctx, bson.M{"_id": errorPatternID}).Decode(&pattern)
	if err != nil {
		return false, fmt.Errorf("failed to get error pattern: %w", err)
	}

	return pattern.Occurrences >= 2 && !pattern.LessonExtracted, nil
}

// GetErrorSuggestion retrieves error pattern data for auto-populating lesson fields
func (rs *ReflectionStorage) GetErrorSuggestion(errorPatternID string) (*ErrorPattern, error) {
	ctx := context.Background()

	var pattern ErrorPattern
	err := rs.errorPatternsCollection.FindOne(ctx, bson.M{"_id": errorPatternID}).Decode(&pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to get error pattern: %w", err)
	}

	return &pattern, nil
}

// MarkLessonExtracted marks that a lesson has been extracted for an error pattern
func (rs *ReflectionStorage) MarkLessonExtracted(errorPatternID, lessonID string) error {
	ctx := context.Background()

	_, err := rs.errorPatternsCollection.UpdateOne(
		ctx,
		bson.M{"_id": errorPatternID},
		bson.M{
			"$set": bson.M{
				"lessonExtracted": true,
				"relatedLesson":   lessonID,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to mark lesson extracted: %w", err)
	}

	rs.logger.Info("Lesson extraction marked",
		zap.String("errorPattern", errorPatternID),
		zap.String("lesson", lessonID))

	return nil
}
