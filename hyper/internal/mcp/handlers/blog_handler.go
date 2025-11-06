package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	aiservice "hyper/internal/ai-service"
	"hyper/internal/mcp/storage"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type BlogHandler struct {
	taskStorage      storage.TaskStorage
	knowledgeStorage storage.KnowledgeStorage
	aiChatService    *aiservice.ChatService
	logger           *zap.Logger
}

func NewBlogHandler(
	taskStorage storage.TaskStorage,
	knowledgeStorage storage.KnowledgeStorage,
	aiChatService *aiservice.ChatService,
	logger *zap.Logger,
) *BlogHandler {
	return &BlogHandler{
		taskStorage:      taskStorage,
		knowledgeStorage: knowledgeStorage,
		aiChatService:    aiChatService,
		logger:           logger,
	}
}

type GenerateBlogEntryResponse struct {
	Entry storage.KnowledgeEntry `json:"entry"`
}

func (h *BlogHandler) GenerateEntry(c *gin.Context) {
	h.logger.Info("Generating blog entry from completed tasks")

	// 0. Ensure progress-blog collection exists
	_, err := h.knowledgeStorage.GetEntriesByCollection("progress-blog")
	if err != nil {
		// Collection doesn't exist, create it
		h.logger.Info("Creating progress-blog collection")
		_, err = h.knowledgeStorage.CreateCollection(
			"progress-blog",
			"documentation",
			"AI-generated progress reports and blog posts tracking development achievements",
			[]string{"blog", "progress", "ai-generated", "reports"},
		)
		if err != nil {
			// Ignore error if collection already exists (race condition)
			if !strings.Contains(err.Error(), "already exists") {
				h.logger.Error("Failed to create progress-blog collection", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize blog storage"})
				return
			}
		}
	}

	// 1. Find last blog entry to get timestamp
	lastEntries, err := h.knowledgeStorage.ListKnowledge("progress-blog", 1)
	var sinceTime time.Time
	if err != nil || len(lastEntries) == 0 {
		// No previous entries, use 7 days ago as default
		sinceTime = time.Now().AddDate(0, 0, -7)
		h.logger.Info("No previous blog entries, using 7 days ago as baseline", zap.Time("since", sinceTime))
	} else {
		sinceTime = lastEntries[0].CreatedAt
		h.logger.Info("Found last blog entry", zap.Time("since", sinceTime))
	}

	// 2. Query completed tasks since last entry
	humanTasks, err := h.taskStorage.ListHumanTasks(nil)
	if err != nil {
		h.logger.Error("Failed to get human tasks", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query tasks"})
		return
	}

	agentTasks, _, err := h.taskStorage.ListAgentTasks(nil, 0, 1000)
	if err != nil {
		h.logger.Error("Failed to get agent tasks", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query tasks"})
		return
	}

	// Filter completed tasks since sinceTime
	var completedHumanTasks []*storage.HumanTask
	var completedAgentTasks []*storage.AgentTask

	for _, task := range humanTasks {
		if task.Status == "completed" && task.UpdatedAt.After(sinceTime) {
			completedHumanTasks = append(completedHumanTasks, task)
		}
	}

	for _, task := range agentTasks {
		if task.Status == "completed" && task.UpdatedAt.After(sinceTime) {
			completedAgentTasks = append(completedAgentTasks, task)
		}
	}

	h.logger.Info("Found completed tasks",
		zap.Int("humanTasks", len(completedHumanTasks)),
		zap.Int("agentTasks", len(completedAgentTasks)))

	if len(completedHumanTasks) == 0 && len(completedAgentTasks) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"error": "No completed tasks found since last blog entry",
		})
		return
	}

	// 3. Build task summary for AI
	taskSummary := buildTaskSummary(completedHumanTasks, completedAgentTasks)

	// 4. Use AI to generate blog post
	aiPrompt := fmt.Sprintf(`You are a professional technical writer. Based on the following completed tasks, create a beautiful, insightful progress blog post in markdown format.

%s

Create a blog post that:
- Has an engaging title (use # heading)
- Summarizes key achievements
- Highlights important milestones
- Identifies patterns or trends
- Is positive and motivational
- Uses proper markdown formatting
- Includes statistics at the end

Make it read like a professional progress report, not a dry task list. Keep it concise but informative (aim for 300-500 words).`, taskSummary)

	// Call AI via chat service
	blogContent, err := h.callAIForBlogGeneration(aiPrompt)
	if err != nil {
		h.logger.Error("Failed to generate blog content with AI", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate blog content"})
		return
	}

	// 5. Store to knowledge base
	metadata := map[string]interface{}{
		"generated_at": time.Now().Format(time.RFC3339),
		"human_tasks":  len(completedHumanTasks),
		"agent_tasks":  len(completedAgentTasks),
		"generated_by": "ai",
	}

	entry, err := h.knowledgeStorage.Upsert("progress-blog", blogContent, metadata, nil)
	if err != nil {
		h.logger.Error("Failed to store blog entry", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store blog entry"})
		return
	}

	h.logger.Info("Blog entry generated and stored successfully", zap.String("id", entry.ID))

	c.JSON(http.StatusOK, GenerateBlogEntryResponse{
		Entry: *entry,
	})
}

func buildTaskSummary(humanTasks []*storage.HumanTask, agentTasks []*storage.AgentTask) string {
	summary := "Completed Tasks:\n\n"

	// Group agent tasks by human task
	agentTasksByHuman := make(map[string][]*storage.AgentTask)
	for _, agentTask := range agentTasks {
		agentTasksByHuman[agentTask.HumanTaskID] = append(agentTasksByHuman[agentTask.HumanTaskID], agentTask)
	}

	for _, humanTask := range humanTasks {
		summary += fmt.Sprintf("### %s\n", humanTask.Prompt)
		summary += fmt.Sprintf("- Status: Completed\n")
		summary += fmt.Sprintf("- Completed: %s\n", humanTask.UpdatedAt.Format("2006-01-02 15:04"))

		// Add related agent tasks
		relatedAgentTasks := agentTasksByHuman[humanTask.ID]
		if len(relatedAgentTasks) > 0 {
			summary += "\nAgent Work:\n"
			for _, agentTask := range relatedAgentTasks {
				summary += fmt.Sprintf("- **%s** (%s): %s\n", agentTask.AgentName, agentTask.Role, agentTask.Status)

				// Count completed todos
				completedTodos := 0
				for _, todo := range agentTask.Todos {
					if todo.Status == "completed" {
						completedTodos++
					}
				}
				if completedTodos > 0 {
					summary += fmt.Sprintf("  - Completed %d/%d todos\n", completedTodos, len(agentTask.Todos))
				}
			}
		}
		summary += "\n"
	}

	// Add statistics
	totalTodos := 0
	completedTodos := 0
	for _, agentTask := range agentTasks {
		totalTodos += len(agentTask.Todos)
		for _, todo := range agentTask.Todos {
			if todo.Status == "completed" {
				completedTodos++
			}
		}
	}

	summary += fmt.Sprintf("\n**Statistics:**\n")
	summary += fmt.Sprintf("- Human Tasks: %d\n", len(humanTasks))
	summary += fmt.Sprintf("- Agent Tasks: %d\n", len(agentTasks))
	summary += fmt.Sprintf("- Total TODOs Completed: %d/%d\n", completedTodos, totalTodos)

	return summary
}

func (h *BlogHandler) callAIForBlogGeneration(prompt string) (string, error) {
	// Use the AI chat service to call Claude
	ctx := context.Background()

	// Create message for AI
	messages := []aiservice.Message{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Stream the response
	streamChan, err := h.aiChatService.StreamChat(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to call AI: %w", err)
	}

	// Collect all tokens from stream
	var result strings.Builder
	for token := range streamChan {
		result.WriteString(token)
	}

	content := result.String()
	if content == "" {
		return "", fmt.Errorf("AI returned empty response")
	}

	return content, nil
}
