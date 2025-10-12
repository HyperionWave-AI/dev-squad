package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"hyper/internal/mcp/storage"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MetricsResourceHandler manages performance metrics resources
type MetricsResourceHandler struct {
	taskStorage storage.TaskStorage
}

// NewMetricsResourceHandler creates a new metrics resource handler
func NewMetricsResourceHandler(taskStorage storage.TaskStorage) *MetricsResourceHandler {
	return &MetricsResourceHandler{
		taskStorage: taskStorage,
	}
}

// SquadVelocityMetrics represents task completion rates by squad
type SquadVelocityMetrics struct {
	SquadName         string    `json:"squadName"`
	TodayCompleted    int       `json:"todayCompleted"`
	WeekCompleted     int       `json:"weekCompleted"`
	AllTimeCompleted  int       `json:"allTimeCompleted"`
	TodayTaskCount    int       `json:"todayTaskCount"`
	WeekTaskCount     int       `json:"weekTaskCount"`
	AllTimeTaskCount  int       `json:"allTimeTaskCount"`
	CompletionRate    float64   `json:"completionRate"`
	AverageTodoDuration float64 `json:"averageTodoDuration"` // in minutes
	LastActivity      time.Time `json:"lastActivity"`
}

// ContextEfficiencyMetrics represents agent context usage statistics
type ContextEfficiencyMetrics struct {
	OverallStats        OverallEfficiencyStats   `json:"overallStats"`
	SquadStats          []SquadEfficiencyStats   `json:"squadStats"`
	TrendData           TrendData                `json:"trendData"`
	TaskComplexity      TaskComplexityMetrics    `json:"taskComplexity"`
}

// OverallEfficiencyStats contains platform-wide metrics
type OverallEfficiencyStats struct {
	AverageCompletionTime float64 `json:"averageCompletionTime"` // in hours
	TasksPerDay           float64 `json:"tasksPerDay"`
	TasksPerWeek          float64 `json:"tasksPerWeek"`
	TotalTasksCompleted   int     `json:"totalTasksCompleted"`
	ActiveSquads          int     `json:"activeSquads"`
	EfficiencyScore       float64 `json:"efficiencyScore"` // 0-100 scale
}

// SquadEfficiencyStats contains per-squad efficiency data
type SquadEfficiencyStats struct {
	SquadName             string  `json:"squadName"`
	AverageCompletionTime float64 `json:"averageCompletionTime"` // in hours
	TasksCompleted        int     `json:"tasksCompleted"`
	AverageTodoCount      float64 `json:"averageTodoCount"`
	SuccessRate           float64 `json:"successRate"` // percentage
}

// TrendData contains time-series performance data
type TrendData struct {
	DailyCompletions  []DailyMetric `json:"dailyCompletions"`
	WeeklyCompletions []WeeklyMetric `json:"weeklyCompletions"`
}

// DailyMetric represents daily completion metrics
type DailyMetric struct {
	Date       string `json:"date"`
	Completed  int    `json:"completed"`
	InProgress int    `json:"inProgress"`
	Blocked    int    `json:"blocked"`
}

// WeeklyMetric represents weekly completion metrics
type WeeklyMetric struct {
	WeekStart  string  `json:"weekStart"`
	Completed  int     `json:"completed"`
	AvgPerDay  float64 `json:"avgPerDay"`
}

// TaskComplexityMetrics analyzes task difficulty
type TaskComplexityMetrics struct {
	AverageTodoCount    float64            `json:"averageTodoCount"`
	TodoDistribution    map[string]int     `json:"todoDistribution"`
	ComplexTasksPercent float64            `json:"complexTasksPercent"` // tasks with >5 TODOs
	SimpleTasksPercent  float64            `json:"simpleTasksPercent"`  // tasks with ≤3 TODOs
}

// RegisterMetricsResources registers all metrics resources with the MCP server
func (h *MetricsResourceHandler) RegisterMetricsResources(server *mcp.Server) error {
	// Register squad-velocity resource
	squadVelocityResource := &mcp.Resource{
		URI:         "hyperion://metrics/squad-velocity",
		Name:        "Squad Velocity Metrics",
		Description: "Task completion rates by squad (today, this week, all time)",
		MIMEType:    "application/json",
	}
	server.AddResource(squadVelocityResource, h.handleSquadVelocity)

	// Register context-efficiency resource
	contextEfficiencyResource := &mcp.Resource{
		URI:         "hyperion://metrics/context-efficiency",
		Name:        "Context Efficiency Metrics",
		Description: "Agent context usage and performance statistics",
		MIMEType:    "application/json",
	}
	server.AddResource(contextEfficiencyResource, h.handleContextEfficiency)

	return nil
}

// handleSquadVelocity computes and returns squad velocity metrics
func (h *MetricsResourceHandler) handleSquadVelocity(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	allAgentTasks := h.taskStorage.ListAllAgentTasks()

	// Define time windows
	now := time.Now().UTC()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	weekStart := todayStart.AddDate(0, 0, -7)

	// Group tasks by squad (agent name)
	squadTaskMap := make(map[string][]*storage.AgentTask)
	for _, task := range allAgentTasks {
		squadTaskMap[task.AgentName] = append(squadTaskMap[task.AgentName], task)
	}

	// Calculate metrics for each squad
	squadMetrics := make([]SquadVelocityMetrics, 0, len(squadTaskMap))
	for squadName, tasks := range squadTaskMap {
		metrics := SquadVelocityMetrics{
			SquadName:        squadName,
			TodayCompleted:   0,
			WeekCompleted:    0,
			AllTimeCompleted: 0,
			TodayTaskCount:   0,
			WeekTaskCount:    0,
			AllTimeTaskCount: len(tasks),
			LastActivity:     time.Time{},
		}

		totalTodoDuration := 0.0
		todoCompletionCount := 0

		for _, task := range tasks {
			// Update last activity
			if task.UpdatedAt.After(metrics.LastActivity) {
				metrics.LastActivity = task.UpdatedAt
			}

			// Count completed tasks by time window
			if task.Status == storage.TaskStatusCompleted {
				metrics.AllTimeCompleted++

				if task.UpdatedAt.After(weekStart) {
					metrics.WeekCompleted++
					metrics.WeekTaskCount++
				}

				if task.UpdatedAt.After(todayStart) {
					metrics.TodayCompleted++
					metrics.TodayTaskCount++
				}

				// Calculate average TODO completion duration
				for _, todo := range task.Todos {
					if todo.Status == storage.TodoStatusCompleted && todo.CompletedAt != nil {
						duration := todo.CompletedAt.Sub(todo.CreatedAt).Minutes()
						totalTodoDuration += duration
						todoCompletionCount++
					}
				}
			} else {
				// Count non-completed tasks
				if task.CreatedAt.After(weekStart) {
					metrics.WeekTaskCount++
				}
				if task.CreatedAt.After(todayStart) {
					metrics.TodayTaskCount++
				}
			}
		}

		// Calculate completion rate (all time)
		if metrics.AllTimeTaskCount > 0 {
			metrics.CompletionRate = float64(metrics.AllTimeCompleted) / float64(metrics.AllTimeTaskCount) * 100.0
		}

		// Calculate average TODO duration
		if todoCompletionCount > 0 {
			metrics.AverageTodoDuration = totalTodoDuration / float64(todoCompletionCount)
		}

		squadMetrics = append(squadMetrics, metrics)
	}

	// Sort by all-time completed count (descending)
	sort.Slice(squadMetrics, func(i, j int) bool {
		return squadMetrics[i].AllTimeCompleted > squadMetrics[j].AllTimeCompleted
	})

	jsonData, err := json.MarshalIndent(map[string]interface{}{
		"squads":    squadMetrics,
		"timestamp": now,
		"windows": map[string]string{
			"today": todayStart.Format(time.RFC3339),
			"week":  weekStart.Format(time.RFC3339),
		},
	}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal squad velocity metrics: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      "hyperion://metrics/squad-velocity",
				MIMEType: "application/json",
				Text:     string(jsonData),
			},
		},
	}, nil
}

// handleContextEfficiency computes and returns context efficiency metrics
func (h *MetricsResourceHandler) handleContextEfficiency(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	allAgentTasks := h.taskStorage.ListAllAgentTasks()

	// Calculate overall stats
	overallStats := h.calculateOverallStats(allAgentTasks)

	// Calculate per-squad stats
	squadStats := h.calculateSquadStats(allAgentTasks)

	// Calculate trend data
	trendData := h.calculateTrendData(allAgentTasks)

	// Calculate task complexity metrics
	complexityMetrics := h.calculateComplexityMetrics(allAgentTasks)

	metrics := ContextEfficiencyMetrics{
		OverallStats:   overallStats,
		SquadStats:     squadStats,
		TrendData:      trendData,
		TaskComplexity: complexityMetrics,
	}

	jsonData, err := json.MarshalIndent(map[string]interface{}{
		"metrics":   metrics,
		"timestamp": time.Now().UTC(),
		"analysis": map[string]string{
			"efficiency": h.getEfficiencyAnalysis(overallStats.EfficiencyScore),
			"trend":      h.getTrendAnalysis(trendData),
		},
	}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal context efficiency metrics: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      "hyperion://metrics/context-efficiency",
				MIMEType: "application/json",
				Text:     string(jsonData),
			},
		},
	}, nil
}

// calculateOverallStats computes platform-wide efficiency statistics
func (h *MetricsResourceHandler) calculateOverallStats(tasks []*storage.AgentTask) OverallEfficiencyStats {
	stats := OverallEfficiencyStats{
		TotalTasksCompleted: 0,
		ActiveSquads:        0,
	}

	if len(tasks) == 0 {
		return stats
	}

	totalCompletionTime := 0.0
	completedCount := 0
	squadSet := make(map[string]bool)

	now := time.Now().UTC()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	weekStart := todayStart.AddDate(0, 0, -7)

	todayCompleted := 0
	weekCompleted := 0

	for _, task := range tasks {
		squadSet[task.AgentName] = true

		if task.Status == storage.TaskStatusCompleted {
			stats.TotalTasksCompleted++
			completedCount++

			// Calculate completion time in hours
			completionTime := task.UpdatedAt.Sub(task.CreatedAt).Hours()
			totalCompletionTime += completionTime

			// Count daily/weekly completions
			if task.UpdatedAt.After(todayStart) {
				todayCompleted++
			}
			if task.UpdatedAt.After(weekStart) {
				weekCompleted++
			}
		}
	}

	stats.ActiveSquads = len(squadSet)

	// Calculate averages
	if completedCount > 0 {
		stats.AverageCompletionTime = totalCompletionTime / float64(completedCount)
		stats.TasksPerDay = float64(todayCompleted)
		stats.TasksPerWeek = float64(weekCompleted)
	}

	// Calculate efficiency score (0-100)
	// Formula: weighted combination of completion rate, speed, and task throughput
	completionRate := 0.0
	if len(tasks) > 0 {
		completionRate = float64(stats.TotalTasksCompleted) / float64(len(tasks)) * 100.0
	}

	speedScore := 100.0
	if stats.AverageCompletionTime > 0 {
		// Target: <2 hours per task = 100%, 24 hours = 0%
		speedScore = max(0, 100.0-((stats.AverageCompletionTime-2.0)/22.0)*100.0)
	}

	throughputScore := min(100.0, stats.TasksPerDay*10.0) // 10 tasks/day = 100%

	stats.EfficiencyScore = (completionRate*0.4 + speedScore*0.3 + throughputScore*0.3)

	return stats
}

// calculateSquadStats computes per-squad efficiency metrics
func (h *MetricsResourceHandler) calculateSquadStats(tasks []*storage.AgentTask) []SquadEfficiencyStats {
	squadTaskMap := make(map[string][]*storage.AgentTask)
	for _, task := range tasks {
		squadTaskMap[task.AgentName] = append(squadTaskMap[task.AgentName], task)
	}

	stats := make([]SquadEfficiencyStats, 0, len(squadTaskMap))
	for squadName, squadTasks := range squadTaskMap {
		squadStat := SquadEfficiencyStats{
			SquadName:      squadName,
			TasksCompleted: 0,
		}

		totalCompletionTime := 0.0
		totalTodoCount := 0
		completedCount := 0

		for _, task := range squadTasks {
			totalTodoCount += len(task.Todos)

			if task.Status == storage.TaskStatusCompleted {
				squadStat.TasksCompleted++
				completedCount++

				completionTime := task.UpdatedAt.Sub(task.CreatedAt).Hours()
				totalCompletionTime += completionTime
			}
		}

		// Calculate averages
		if completedCount > 0 {
			squadStat.AverageCompletionTime = totalCompletionTime / float64(completedCount)
		}

		if len(squadTasks) > 0 {
			squadStat.AverageTodoCount = float64(totalTodoCount) / float64(len(squadTasks))
			squadStat.SuccessRate = float64(squadStat.TasksCompleted) / float64(len(squadTasks)) * 100.0
		}

		stats = append(stats, squadStat)
	}

	// Sort by tasks completed (descending)
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].TasksCompleted > stats[j].TasksCompleted
	})

	return stats
}

// calculateTrendData generates time-series performance data
func (h *MetricsResourceHandler) calculateTrendData(tasks []*storage.AgentTask) TrendData {
	trend := TrendData{
		DailyCompletions:  make([]DailyMetric, 0),
		WeeklyCompletions: make([]WeeklyMetric, 0),
	}

	// Daily metrics for last 7 days
	now := time.Now().UTC()
	dailyMetrics := make(map[string]*DailyMetric)

	for i := 6; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		dailyMetrics[dateStr] = &DailyMetric{
			Date:       dateStr,
			Completed:  0,
			InProgress: 0,
			Blocked:    0,
		}
	}

	// Weekly metrics for last 4 weeks
	weeklyMetrics := make(map[string]*WeeklyMetric)
	for i := 3; i >= 0; i-- {
		weekStart := now.AddDate(0, 0, -7*i-int(now.Weekday()))
		weekStr := weekStart.Format("2006-01-02")
		weeklyMetrics[weekStr] = &WeeklyMetric{
			WeekStart: weekStr,
			Completed: 0,
		}
	}

	// Populate metrics from tasks
	for _, task := range tasks {
		// Daily metrics
		dateStr := task.UpdatedAt.Format("2006-01-02")
		if daily, exists := dailyMetrics[dateStr]; exists {
			switch task.Status {
			case storage.TaskStatusCompleted:
				daily.Completed++
			case storage.TaskStatusInProgress:
				daily.InProgress++
			case storage.TaskStatusBlocked:
				daily.Blocked++
			}
		}

		// Weekly metrics
		taskWeekStart := task.UpdatedAt.AddDate(0, 0, -int(task.UpdatedAt.Weekday()))
		weekStr := taskWeekStart.Format("2006-01-02")
		if weekly, exists := weeklyMetrics[weekStr]; exists {
			if task.Status == storage.TaskStatusCompleted {
				weekly.Completed++
			}
		}
	}

	// Convert maps to sorted slices
	for _, metric := range dailyMetrics {
		trend.DailyCompletions = append(trend.DailyCompletions, *metric)
	}
	sort.Slice(trend.DailyCompletions, func(i, j int) bool {
		return trend.DailyCompletions[i].Date < trend.DailyCompletions[j].Date
	})

	for _, metric := range weeklyMetrics {
		metric.AvgPerDay = float64(metric.Completed) / 7.0
		trend.WeeklyCompletions = append(trend.WeeklyCompletions, *metric)
	}
	sort.Slice(trend.WeeklyCompletions, func(i, j int) bool {
		return trend.WeeklyCompletions[i].WeekStart < trend.WeeklyCompletions[j].WeekStart
	})

	return trend
}

// calculateComplexityMetrics analyzes task difficulty based on TODO counts
func (h *MetricsResourceHandler) calculateComplexityMetrics(tasks []*storage.AgentTask) TaskComplexityMetrics {
	metrics := TaskComplexityMetrics{
		TodoDistribution: make(map[string]int),
	}

	if len(tasks) == 0 {
		return metrics
	}

	totalTodoCount := 0
	complexTasks := 0
	simpleTasks := 0

	for _, task := range tasks {
		todoCount := len(task.Todos)
		totalTodoCount += todoCount

		// Distribution buckets
		bucket := ""
		switch {
		case todoCount <= 3:
			bucket = "1-3"
			simpleTasks++
		case todoCount <= 5:
			bucket = "4-5"
		case todoCount <= 7:
			bucket = "6-7"
			complexTasks++
		default:
			bucket = "8+"
			complexTasks++
		}
		metrics.TodoDistribution[bucket]++
	}

	metrics.AverageTodoCount = float64(totalTodoCount) / float64(len(tasks))
	metrics.ComplexTasksPercent = float64(complexTasks) / float64(len(tasks)) * 100.0
	metrics.SimpleTasksPercent = float64(simpleTasks) / float64(len(tasks)) * 100.0

	return metrics
}

// getEfficiencyAnalysis provides textual analysis of efficiency score
func (h *MetricsResourceHandler) getEfficiencyAnalysis(score float64) string {
	switch {
	case score >= 90:
		return "Excellent - squad operating at peak efficiency"
	case score >= 75:
		return "Good - squad performance is above target"
	case score >= 60:
		return "Fair - squad performance is adequate but has room for improvement"
	case score >= 40:
		return "Poor - squad performance is below target, investigate bottlenecks"
	default:
		return "Critical - squad performance requires immediate attention"
	}
}

// getTrendAnalysis provides textual analysis of performance trends
func (h *MetricsResourceHandler) getTrendAnalysis(trend TrendData) string {
	if len(trend.WeeklyCompletions) < 2 {
		return "Insufficient data for trend analysis"
	}

	// Compare last two weeks
	current := trend.WeeklyCompletions[len(trend.WeeklyCompletions)-1]
	previous := trend.WeeklyCompletions[len(trend.WeeklyCompletions)-2]

	change := ((current.AvgPerDay - previous.AvgPerDay) / previous.AvgPerDay) * 100.0

	switch {
	case change > 20:
		return fmt.Sprintf("Strong upward trend (+%.1f%% week-over-week)", change)
	case change > 5:
		return fmt.Sprintf("Improving trend (+%.1f%% week-over-week)", change)
	case change > -5:
		return "Stable performance"
	case change > -20:
		return fmt.Sprintf("Declining trend (%.1f%% week-over-week)", change)
	default:
		return fmt.Sprintf("Significant decline (%.1f%% week-over-week)", change)
	}
}

// Helper functions
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
