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

// FeedbackStats represents aggregated feedback statistics
type FeedbackStats struct {
	TotalCount      int                `json:"totalCount"`
	IssueCount      int                `json:"issueCount"`
	SuccessCount    int                `json:"successCount"`
	SuggestionCount int                `json:"suggestionCount"`
	Groups          []FeedbackGroup    `json:"groups"`
	Recommendations []string           `json:"recommendations,omitempty"`
}

// FeedbackGroup represents a group of feedback (by agent, category, or type)
type FeedbackGroup struct {
	Name            string   `json:"name"`
	Count           int      `json:"count"`
	IssueCount      int      `json:"issueCount"`
	SuccessCount    int      `json:"successCount"`
	SuggestionCount int      `json:"suggestionCount"`
	AvgSeverity     float64  `json:"avgSeverity"`
	TopIssues       []string `json:"topIssues,omitempty"`
}

// ReflectionStorage provides storage for metacognitive data
type ReflectionStorage struct {
	reflectionsCollection     *mongo.Collection
	experienceIndexCollection *mongo.Collection
	errorPatternsCollection   *mongo.Collection
	qdrantClient              QdrantClientInterface
	vectorDimension           int
	lessonsCollectionName     string
	logger                    *zap.Logger
}

// NewReflectionStorage creates a new reflection storage instance with optional Qdrant support
func NewReflectionStorage(db *mongo.Database, qdrantClient QdrantClientInterface, logger *zap.Logger) (*ReflectionStorage, error) {
	// Determine vector dimension and collection name
	vectorDim := 768 // Default dimension
	lessonsCollection := "reflection_lessons"

	if qdrantClient != nil {
		vectorDim = qdrantClient.GetDimensions()
		// Create dimension-specific collection name to avoid conflicts
		lessonsCollection = fmt.Sprintf("reflection_lessons_%d", vectorDim)
		logger.Info("Reflection storage using Qdrant for semantic search",
			zap.String("collection", lessonsCollection),
			zap.Int("dimensions", vectorDim))
	} else {
		logger.Warn("Reflection storage initialized without Qdrant - semantic search will use MongoDB regex only")
	}

	storage := &ReflectionStorage{
		reflectionsCollection:     db.Collection("reflections"),
		experienceIndexCollection: db.Collection("experience_index"),
		errorPatternsCollection:   db.Collection("error_patterns"),
		qdrantClient:              qdrantClient,
		vectorDimension:           vectorDim,
		lessonsCollectionName:     lessonsCollection,
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

// StoreReflection stores a reflection in MongoDB and optionally in Qdrant (for lessons)
func (rs *ReflectionStorage) StoreReflection(reflection *Reflection) (string, error) {
	ctx := context.Background()

	if reflection.ID == "" {
		reflection.ID = uuid.New().String()
	}

	if reflection.Timestamp.IsZero() {
		reflection.Timestamp = time.Now().UTC()
	}

	// Store in MongoDB (source of truth)
	_, err := rs.reflectionsCollection.InsertOne(ctx, reflection)
	if err != nil {
		return "", fmt.Errorf("failed to store reflection: %w", err)
	}

	rs.logger.Info("Stored reflection",
		zap.String("id", reflection.ID),
		zap.String("type", reflection.Type),
		zap.String("chatId", reflection.ChatID))

	// If this is a lesson and we have Qdrant, store for semantic search
	if reflection.Type == "lesson" && rs.qdrantClient != nil {
		// Build searchable text from lesson data
		lessonText := rs.buildLessonSearchText(reflection)

		// Ensure Qdrant collection exists with dimension checking
		if err := rs.qdrantClient.EnsureCollection(rs.lessonsCollectionName, rs.vectorDimension); err != nil {
			// Check for dimension mismatch
			if dimErr, ok := err.(*DimensionMismatchError); ok {
				rs.logger.Warn("Dimension mismatch detected for lessons - triggering migration",
					zap.String("collection", rs.lessonsCollectionName),
					zap.Int("expectedDim", dimErr.ExpectedDim),
					zap.Int("gotDim", dimErr.GotDim))

				// Fetch all lessons from MongoDB
				lessons, fetchErr := rs.GetReflectionsByType("lesson")
				if fetchErr != nil {
					rs.logger.Error("Failed to fetch lessons for migration", zap.Error(fetchErr))
				} else {
					// Convert to format needed for re-indexing
					entries := make([]*KnowledgeEntry, len(lessons))
					for i, lesson := range lessons {
						entries[i] = &KnowledgeEntry{
							ID:       lesson.ID,
							Text:     rs.buildLessonSearchText(lesson),
							Metadata: map[string]interface{}{
								"type":      "lesson",
								"timestamp": lesson.Timestamp,
								"tags":      lesson.Tags,
							},
						}
					}

					// Recreate collection with new dimensions
					reindexedCount, migrateErr := rs.qdrantClient.RecreateCollectionWithReindex(
						rs.lessonsCollectionName,
						entries,
						rs.vectorDimension,
					)
					if migrateErr != nil {
						rs.logger.Error("Failed to migrate lessons collection", zap.Error(migrateErr))
					} else {
						rs.logger.Info("Successfully migrated lessons to new dimensions",
							zap.String("collection", rs.lessonsCollectionName),
							zap.Int("oldDim", dimErr.ExpectedDim),
							zap.Int("newDim", dimErr.GotDim),
							zap.Int("entriesMigrated", reindexedCount))
					}
				}
			} else {
				rs.logger.Warn("Failed to ensure Qdrant collection for lessons", zap.Error(err))
			}
		}

		// Store lesson in Qdrant
		metadata := map[string]interface{}{
			"type":      "lesson",
			"timestamp": reflection.Timestamp,
			"tags":      reflection.Tags,
		}
		if err := rs.qdrantClient.StorePoint(rs.lessonsCollectionName, reflection.ID, lessonText, metadata); err != nil {
			rs.logger.Warn("Failed to store lesson in Qdrant (MongoDB has it)",
				zap.String("lessonId", reflection.ID),
				zap.Error(err))
		} else {
			rs.logger.Debug("Stored lesson in Qdrant for semantic search",
				zap.String("lessonId", reflection.ID))
		}
	}

	return reflection.ID, nil
}

// buildLessonSearchText builds searchable text from lesson data for semantic search
func (rs *ReflectionStorage) buildLessonSearchText(reflection *Reflection) string {
	if reflection.Type != "lesson" {
		return ""
	}

	data := reflection.Data
	patternName, _ := data["patternName"].(string)
	problem, _ := data["problem"].(string)
	solution, _ := data["solution"].(string)
	context, _ := data["context"].(string)
	antipattern, _ := data["antipattern"].(string)

	// Build comprehensive searchable text
	text := fmt.Sprintf("Pattern: %s\nProblem: %s\nSolution: %s", patternName, problem, solution)
	if context != "" {
		text += fmt.Sprintf("\nContext: %s", context)
	}
	if antipattern != "" {
		text += fmt.Sprintf("\nAntipattern: %s", antipattern)
	}

	return text
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

// SearchLessonsByText performs semantic search across lesson content using Qdrant (if available)
// Falls back to MongoDB regex search if Qdrant is not configured
func (rs *ReflectionStorage) SearchLessonsByText(query string, limit int) ([]*Reflection, error) {
	ctx := context.Background()

	// If Qdrant is available, use semantic search
	if rs.qdrantClient != nil {
		rs.logger.Debug("Using Qdrant semantic search for lessons",
			zap.String("query", query),
			zap.Int("limit", limit))

		// Search in Qdrant using semantic similarity
		results, err := rs.qdrantClient.SearchSimilar(rs.lessonsCollectionName, query, limit)
		if err != nil {
			// Log warning but fall back to MongoDB regex search
			rs.logger.Warn("Qdrant search failed, falling back to MongoDB regex",
				zap.Error(err))
		} else if len(results) > 0 {
			// Convert Qdrant results to Reflection objects
			lessons := make([]*Reflection, 0, len(results))
			for _, result := range results {
				// Fetch full reflection from MongoDB by ID
				var lesson Reflection
				err := rs.reflectionsCollection.FindOne(ctx, bson.M{"_id": result.Entry.ID}).Decode(&lesson)
				if err != nil {
					rs.logger.Warn("Failed to fetch lesson from MongoDB",
						zap.String("lessonId", result.Entry.ID),
						zap.Error(err))
					continue
				}
				lessons = append(lessons, &lesson)
			}

			rs.logger.Info("Semantic search completed",
				zap.String("query", query),
				zap.Int("results", len(lessons)))

			return lessons, nil
		}
	}

	// Fallback: MongoDB regex search
	rs.logger.Debug("Using MongoDB regex search for lessons",
		zap.String("query", query),
		zap.Int("limit", limit))

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

	rs.logger.Info("Regex search completed",
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

// GetFeedbackStats retrieves aggregated feedback statistics
func (rs *ReflectionStorage) GetFeedbackStats(groupBy, filterAgent, filterCategory, filterType string, days, limit int) (*FeedbackStats, error) {
	ctx := context.Background()

	// Build filter for feedback type and time range
	cutoff := time.Now().UTC().AddDate(0, 0, -days)
	filter := bson.M{
		"type":      "feedback",
		"timestamp": bson.M{"$gte": cutoff},
	}

	// Add optional filters
	if filterAgent != "" {
		filter["data.agentType"] = filterAgent
	}
	if filterCategory != "" {
		filter["data.category"] = filterCategory
	}
	if filterType != "" {
		filter["data.feedbackType"] = filterType
	}

	// Determine group field based on groupBy parameter
	groupField := "$data.agentType"
	switch groupBy {
	case "category":
		groupField = "$data.category"
	case "type":
		groupField = "$data.feedbackType"
	}

	// Build aggregation pipeline
	pipeline := mongo.Pipeline{
		// Match feedback entries within time range
		{{Key: "$match", Value: filter}},
		// Group by the specified field
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: groupField},
			{Key: "count", Value: bson.M{"$sum": 1}},
			{Key: "issueCount", Value: bson.M{"$sum": bson.M{
				"$cond": bson.A{bson.M{"$eq": bson.A{"$data.feedbackType", "issue"}}, 1, 0},
			}}},
			{Key: "successCount", Value: bson.M{"$sum": bson.M{
				"$cond": bson.A{bson.M{"$eq": bson.A{"$data.feedbackType", "success"}}, 1, 0},
			}}},
			{Key: "suggestionCount", Value: bson.M{"$sum": bson.M{
				"$cond": bson.A{bson.M{"$eq": bson.A{"$data.feedbackType", "suggestion"}}, 1, 0},
			}}},
			{Key: "avgSeverity", Value: bson.M{"$avg": "$data.severity"}},
			{Key: "issues", Value: bson.M{"$push": bson.M{
				"$cond": bson.A{
					bson.M{"$eq": bson.A{"$data.feedbackType", "issue"}},
					"$data.summary",
					"$$REMOVE",
				},
			}}},
		}}},
		// Sort by count descending
		{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}}}},
		// Limit results
		{{Key: "$limit", Value: limit}},
	}

	cursor, err := rs.reflectionsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate feedback: %w", err)
	}
	defer cursor.Close(ctx)

	// Parse results
	var results []struct {
		ID              string   `bson:"_id"`
		Count           int      `bson:"count"`
		IssueCount      int      `bson:"issueCount"`
		SuccessCount    int      `bson:"successCount"`
		SuggestionCount int      `bson:"suggestionCount"`
		AvgSeverity     float64  `bson:"avgSeverity"`
		Issues          []string `bson:"issues"`
	}

	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode feedback stats: %w", err)
	}

	// Build response
	stats := &FeedbackStats{
		Groups:          make([]FeedbackGroup, 0, len(results)),
		Recommendations: []string{},
	}

	for _, r := range results {
		name := r.ID
		if name == "" {
			name = "unknown"
		}

		// Get top 3 issues for this group
		topIssues := r.Issues
		if len(topIssues) > 3 {
			topIssues = topIssues[:3]
		}

		stats.Groups = append(stats.Groups, FeedbackGroup{
			Name:            name,
			Count:           r.Count,
			IssueCount:      r.IssueCount,
			SuccessCount:    r.SuccessCount,
			SuggestionCount: r.SuggestionCount,
			AvgSeverity:     r.AvgSeverity,
			TopIssues:       topIssues,
		})

		stats.TotalCount += r.Count
		stats.IssueCount += r.IssueCount
		stats.SuccessCount += r.SuccessCount
		stats.SuggestionCount += r.SuggestionCount
	}

	// Generate recommendations based on patterns
	stats.Recommendations = rs.generateRecommendations(stats)

	rs.logger.Info("Retrieved feedback stats",
		zap.String("groupBy", groupBy),
		zap.Int("totalCount", stats.TotalCount),
		zap.Int("groupCount", len(stats.Groups)))

	return stats, nil
}

// generateRecommendations creates actionable recommendations based on feedback patterns
func (rs *ReflectionStorage) generateRecommendations(stats *FeedbackStats) []string {
	recommendations := []string{}

	// Find groups with high issue rates
	for _, group := range stats.Groups {
		if group.Count < 3 {
			continue // Skip groups with too few data points
		}

		issueRate := float64(group.IssueCount) / float64(group.Count)
		successRate := float64(group.SuccessCount) / float64(group.Count)

		// High issue rate recommendation
		if issueRate > 0.6 {
			recommendations = append(recommendations,
				fmt.Sprintf("🔴 %s has %.0f%% issue rate - investigate common failure patterns and update agent prompts",
					group.Name, issueRate*100))
		}

		// High severity issues
		if group.AvgSeverity >= 4.0 && group.IssueCount > 0 {
			recommendations = append(recommendations,
				fmt.Sprintf("⚠️ %s has high severity issues (avg %.1f/5) - prioritize for immediate attention",
					group.Name, group.AvgSeverity))
		}

		// Success patterns to reinforce
		if successRate > 0.7 && group.SuccessCount >= 5 {
			recommendations = append(recommendations,
				fmt.Sprintf("✅ %s has %.0f%% success rate - document patterns as best practices",
					group.Name, successRate*100))
		}
	}

	// Overall recommendations
	if stats.TotalCount > 0 {
		overallIssueRate := float64(stats.IssueCount) / float64(stats.TotalCount)
		if overallIssueRate > 0.5 {
			recommendations = append(recommendations,
				fmt.Sprintf("📊 Overall issue rate is %.0f%% - consider systematic review of agent configurations",
					overallIssueRate*100))
		}
	}

	return recommendations
}

// GetRecentFeedback retrieves recent feedback entries for UI display
func (rs *ReflectionStorage) GetRecentFeedback(filterAgent, filterCategory, filterType string, limit int) ([]*Reflection, error) {
	ctx := context.Background()

	filter := bson.M{"type": "feedback"}

	// Add optional filters
	if filterAgent != "" {
		filter["data.agentType"] = filterAgent
	}
	if filterCategory != "" {
		filter["data.category"] = filterCategory
	}
	if filterType != "" {
		filter["data.feedbackType"] = filterType
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := rs.reflectionsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent feedback: %w", err)
	}
	defer cursor.Close(ctx)

	var feedback []*Reflection
	if err := cursor.All(ctx, &feedback); err != nil {
		return nil, fmt.Errorf("failed to decode feedback: %w", err)
	}

	return feedback, nil
}
