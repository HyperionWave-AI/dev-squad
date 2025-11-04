package handlers

import (
	"net/http"
	"strconv"

	"hyper/internal/mcp/storage"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ReflectionHandler handles HTTP REST requests for reflection/metacognition operations
type ReflectionHandler struct {
	reflectionStorage *storage.ReflectionStorage
	logger            *zap.Logger
}

// NewReflectionHandler creates a new reflection handler
func NewReflectionHandler(
	reflectionStorage *storage.ReflectionStorage,
	logger *zap.Logger,
) *ReflectionHandler {
	return &ReflectionHandler{
		reflectionStorage: reflectionStorage,
		logger:            logger,
	}
}

// GetDecisions retrieves decision reflections
// GET /api/v1/reflection/decisions?chatId=X&taskId=Y&limit=20
func (h *ReflectionHandler) GetDecisions(c *gin.Context) {
	chatId := c.Query("chatId")
	taskId := c.Query("taskId")

	// Parse limit parameter (default 20, max 100)
	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
			if limit > 100 {
				limit = 100 // Max limit
			}
		}
	}

	var reflections []*storage.Reflection
	var err error

	// Filter by chatId or taskId if provided
	if chatId != "" {
		reflections, err = h.reflectionStorage.GetReflectionsByChat(chatId)
	} else if taskId != "" {
		reflections, err = h.reflectionStorage.GetReflectionsByTask(taskId)
	} else {
		reflections, err = h.reflectionStorage.GetReflectionsByType("decision")
	}

	if err != nil {
		h.logger.Error("Failed to get decisions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve decisions"})
		return
	}

	// Filter to only decision type
	decisions := make([]*storage.Reflection, 0)
	for _, r := range reflections {
		if r.Type == "decision" {
			decisions = append(decisions, r)
			if len(decisions) >= limit {
				break
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"decisions": decisions,
		"count":     len(decisions),
	})
}

// GetOutcomes retrieves outcome reflections
// GET /api/v1/reflection/outcomes?decisionId=X&limit=20
func (h *ReflectionHandler) GetOutcomes(c *gin.Context) {
	decisionId := c.Query("decisionId")

	// Parse limit parameter (default 20, max 100)
	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
			if limit > 100 {
				limit = 100
			}
		}
	}

	var reflections []*storage.Reflection
	var err error

	if decisionId != "" {
		// Get the decision and its related reflections
		decision, err := h.reflectionStorage.GetReflectionByID(decisionId)
		if err != nil {
			h.logger.Error("Failed to get decision", zap.Error(err))
			c.JSON(http.StatusNotFound, gin.H{"error": "Decision not found"})
			return
		}

		// Get related reflections (outcomes)
		relatedReflections := make([]*storage.Reflection, 0)
		for _, relatedId := range decision.RelatedReflections {
			related, err := h.reflectionStorage.GetReflectionByID(relatedId)
			if err != nil {
				h.logger.Warn("Failed to get related reflection", zap.String("id", relatedId), zap.Error(err))
				continue
			}
			if related.Type == "outcome" {
				relatedReflections = append(relatedReflections, related)
			}
		}
		reflections = relatedReflections
	} else {
		reflections, err = h.reflectionStorage.GetReflectionsByType("outcome")
		if err != nil {
			h.logger.Error("Failed to get outcomes", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve outcomes"})
			return
		}
	}

	// Apply limit
	if len(reflections) > limit {
		reflections = reflections[:limit]
	}

	c.JSON(http.StatusOK, gin.H{
		"outcomes": reflections,
		"count":    len(reflections),
	})
}

// GetLessons retrieves lesson reflections
// GET /api/v1/reflection/lessons?pattern=X&tag=Y&limit=20
func (h *ReflectionHandler) GetLessons(c *gin.Context) {
	pattern := c.Query("pattern")
	tag := c.Query("tag")

	// Parse limit parameter (default 20, max 100)
	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
			if limit > 100 {
				limit = 100
			}
		}
	}

	var reflections []*storage.Reflection
	var err error

	if pattern != "" {
		// Query experience index for pattern
		patterns, err := h.reflectionStorage.QueryPatterns(pattern)
		if err != nil {
			h.logger.Error("Failed to query patterns", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query patterns"})
			return
		}

		// Get lessons from patterns
		reflections = make([]*storage.Reflection, 0)
		for _, p := range patterns {
			for _, lessonId := range p.RelatedLessons {
				lesson, err := h.reflectionStorage.GetReflectionByID(lessonId)
				if err != nil {
					h.logger.Warn("Failed to get lesson", zap.String("id", lessonId), zap.Error(err))
					continue
				}
				reflections = append(reflections, lesson)
				if len(reflections) >= limit {
					break
				}
			}
			if len(reflections) >= limit {
				break
			}
		}
	} else {
		// Get all lessons
		reflections, err = h.reflectionStorage.GetReflectionsByType("lesson")
		if err != nil {
			h.logger.Error("Failed to get lessons", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve lessons"})
			return
		}
	}

	// Filter by tag if provided
	if tag != "" {
		filtered := make([]*storage.Reflection, 0)
		for _, r := range reflections {
			for _, t := range r.Tags {
				if t == tag {
					filtered = append(filtered, r)
					break
				}
			}
		}
		reflections = filtered
	}

	// Apply limit
	if len(reflections) > limit {
		reflections = reflections[:limit]
	}

	c.JSON(http.StatusOK, gin.H{
		"lessons": reflections,
		"count":   len(reflections),
	})
}

// PostDecision records a new decision reflection
// POST /api/v1/reflection/decision
func (h *ReflectionHandler) PostDecision(c *gin.Context) {
	var req struct {
		ChatId      string                 `json:"chatId" binding:"required"`
		TaskId      string                 `json:"taskId"`
		Context     map[string]interface{} `json:"context" binding:"required"`
		Decision    map[string]interface{} `json:"decision" binding:"required"`
		Predictions map[string]interface{} `json:"predictions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Extract confidence
	confidence := 0.5
	if conf, ok := req.Decision["confidence"].(float64); ok {
		confidence = conf
	}

	// Create reflection
	reflection := &storage.Reflection{
		Type:       "decision",
		ChatID:     req.ChatId,
		TaskID:     req.TaskId,
		Data: map[string]interface{}{
			"context":     req.Context,
			"decision":    req.Decision,
			"predictions": req.Predictions,
		},
		Confidence: confidence,
		Tags:       []string{"decision", "metacognition"},
	}

	decisionId, err := h.reflectionStorage.StoreReflection(reflection)
	if err != nil {
		h.logger.Error("Failed to store decision", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store decision"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"decisionId": decisionId,
		"stored":     true,
	})
}

// PostOutcome records a new outcome reflection
// POST /api/v1/reflection/outcome
func (h *ReflectionHandler) PostOutcome(c *gin.Context) {
	var req struct {
		DecisionId string                 `json:"decisionId" binding:"required"`
		Outcome    map[string]interface{} `json:"outcome" binding:"required"`
		Analysis   map[string]interface{} `json:"analysis" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Get the original decision
	decision, err := h.reflectionStorage.GetReflectionByID(req.DecisionId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Decision not found"})
		return
	}

	// Create outcome reflection
	outcome := &storage.Reflection{
		Type:   "outcome",
		ChatID: decision.ChatID,
		TaskID: decision.TaskID,
		Data: map[string]interface{}{
			"decisionId": req.DecisionId,
			"outcome":    req.Outcome,
			"analysis":   req.Analysis,
		},
		Tags: []string{"outcome", "metacognition"},
	}

	outcomeId, err := h.reflectionStorage.StoreReflection(outcome)
	if err != nil {
		h.logger.Error("Failed to store outcome", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store outcome"})
		return
	}

	// Link decision and outcome
	err = h.reflectionStorage.LinkReflections(req.DecisionId, outcomeId)
	if err != nil {
		h.logger.Error("Failed to link reflections", zap.Error(err))
	}

	// Extract calibration
	calibration := "unknown"
	if cal, ok := req.Analysis["confidenceCalibration"].(string); ok {
		calibration = cal
	}

	c.JSON(http.StatusOK, gin.H{
		"outcomeId":   outcomeId,
		"linked":      true,
		"calibration": calibration,
	})
}

// PostLesson records a new lesson reflection
// POST /api/v1/reflection/lesson
func (h *ReflectionHandler) PostLesson(c *gin.Context) {
	var req struct {
		PatternName  string   `json:"patternName" binding:"required"`
		Context      string   `json:"context"`
		Problem      string   `json:"problem" binding:"required"`
		Solution     string   `json:"solution" binding:"required"`
		Antipattern  string   `json:"antipattern"`
		ApplicableTo []string `json:"applicableTo"`
		Confidence   float64  `json:"confidence"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Default confidence
	if req.Confidence == 0 {
		req.Confidence = 0.8
	}

	// Create lesson reflection
	lesson := &storage.Reflection{
		Type: "lesson",
		Data: map[string]interface{}{
			"patternName":  req.PatternName,
			"context":      req.Context,
			"problem":      req.Problem,
			"solution":     req.Solution,
			"antipattern":  req.Antipattern,
			"applicableTo": req.ApplicableTo,
		},
		Confidence: req.Confidence,
		Tags:       []string{"lesson", "pattern", req.PatternName},
	}

	lessonId, err := h.reflectionStorage.StoreReflection(lesson)
	if err != nil {
		h.logger.Error("Failed to store lesson", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store lesson"})
		return
	}

	// Update experience index
	err = h.reflectionStorage.UpsertExperienceIndex(req.PatternName, req.Context, lessonId, req.Confidence)
	if err != nil {
		h.logger.Error("Failed to update experience index", zap.Error(err))
	}

	c.JSON(http.StatusOK, gin.H{
		"lessonId": lessonId,
		"indexed":  true,
		"pattern":  req.PatternName,
	})
}

// SearchLessons handles GET /api/v1/reflection/search
func (h *ReflectionHandler) SearchLessons(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' is required"})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	lessons, err := h.reflectionStorage.SearchLessonsByText(query, limit)
	if err != nil {
		h.logger.Error("Failed to search lessons", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search lessons"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"query":   query,
		"count":   len(lessons),
		"lessons": lessons,
	})
}

// PostTestError is a test endpoint for recording errors (for automatic lesson extraction testing)
// POST /api/v1/reflection/test-error
func (h *ReflectionHandler) PostTestError(c *gin.Context) {
	var req struct {
		ErrorType  string                 `json:"errorType" binding:"required"`
		Message    string                 `json:"message" binding:"required"`
		StackTrace string                 `json:"stackTrace"`
		Context    map[string]interface{} `json:"context"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Record error in storage
	errorPatternId, shouldSuggest, err := h.reflectionStorage.RecordError(
		req.ErrorType,
		req.Message,
		req.StackTrace,
		req.Context,
	)
	if err != nil {
		h.logger.Error("Failed to record error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record error"})
		return
	}

	response := gin.H{
		"errorPatternId":     errorPatternId,
		"shouldSuggestLesson": shouldSuggest,
		"recorded":           true,
	}

	if shouldSuggest {
		response["message"] = "💡 This error has occurred multiple times. Consider extracting a lesson using the reflection_suggest_lesson_from_error MCP tool."
	}

	c.JSON(http.StatusOK, response)
}
