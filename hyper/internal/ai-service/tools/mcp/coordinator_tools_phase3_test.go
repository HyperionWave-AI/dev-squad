package mcp

import (
	"context"
	"testing"

	"hyper/internal/mcp/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestPhase3ComplexityAnalysis tests the complexity analysis tool
func TestPhase3ComplexityAnalysis(t *testing.T) {
	// Setup logger
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	// Create mock storage (you'll need to provide actual storage connection)
	// For now, this shows the structure
	t.Skip("Skipping - requires actual storage connection")

	var taskStorage storage.TaskStorage
	// taskStorage = ... initialize with your MongoDB connection

	ctx := context.Background()
	tool := &CoordinatorAnalyzeTaskComplexityTool{storage: taskStorage}

	args := map[string]interface{}{
		"title": "Implement Advanced User Management System",
		"contextSummary": "Create a comprehensive user management system with authentication, role-based access control, profile management, and audit logging across frontend and backend",
		"todos": []string{
			"Implement JWT token generation and validation",
			"Create user CRUD operations with MongoDB",
			"Build React profile management UI with form validation",
			"Implement role-based access control middleware",
			"Add audit logging for all user actions",
			"Create password reset flow with email verification",
		},
		"filesModified": []string{
			"auth/jwt.go",
			"handlers/users.go",
			"models/user.go",
			"middleware/rbac.go",
			"services/audit.go",
			"frontend/src/components/UserProfile.tsx",
			"frontend/src/components/PasswordReset.tsx",
		},
	}

	result, err := tool.Execute(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, result)

	resultMap := result.(map[string]interface{})

	// Check that we got complexity score
	score, ok := resultMap["score"].(float64)
	require.True(t, ok, "Score should be a float64")
	assert.Greater(t, score, 0.5, "Complex task should have score > 0.5")

	// Check complexity level
	level, ok := resultMap["level"].(string)
	require.True(t, ok, "Level should be a string")
	assert.NotEmpty(t, level)

	// Check recommendation
	recommendation, ok := resultMap["recommendation"].(string)
	require.True(t, ok, "Recommendation should be a string")
	assert.NotEmpty(t, recommendation)

	t.Logf("Complexity Analysis Result: score=%.2f, level=%s, recommendation=%s",
		score, level, recommendation)
}

// TestPhase3TaskHierarchy tests the task hierarchy retrieval
func TestPhase3TaskHierarchy(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	t.Skip("Skipping - requires actual storage connection with task data")

	var taskStorage storage.TaskStorage
	// taskStorage = ... initialize with your MongoDB connection

	ctx := context.Background()
	tool := &CoordinatorGetTaskHierarchyTool{storage: taskStorage}

	args := map[string]interface{}{
		"includeProgress": true,
		"maxDepth":        5,
	}

	result, err := tool.Execute(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, result)

	resultMap := result.(map[string]interface{})

	// Check that we got root tasks
	rootTasks, ok := resultMap["rootTasks"].([]interface{})
	require.True(t, ok, "rootTasks should be an array")

	t.Logf("Task Hierarchy: Found %d root tasks", len(rootTasks))
}

// TestPhase3NextExecutableTask tests finding executable tasks
func TestPhase3NextExecutableTask(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	t.Skip("Skipping - requires actual storage connection with task data")

	var taskStorage storage.TaskStorage
	// taskStorage = ... initialize with your MongoDB connection

	ctx := context.Background()
	tool := &CoordinatorGetNextExecutableTaskTool{storage: taskStorage}

	args := map[string]interface{}{
		"maxResults":     3,
		"includeDetails": true,
		"sortBy":         "createdAt",
	}

	result, err := tool.Execute(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, result)

	resultMap := result.(map[string]interface{})

	// Check that we got executable tasks
	executableTasks, ok := resultMap["executableTasks"].([]interface{})
	require.True(t, ok, "executableTasks should be an array")

	totalFound, ok := resultMap["totalFound"].(int)
	require.True(t, ok, "totalFound should be an int")

	t.Logf("Executable Tasks: Found %d tasks", totalFound)

	if len(executableTasks) > 0 {
		// Check first task structure
		firstTask := executableTasks[0].(map[string]interface{})
		assert.Contains(t, firstTask, "taskId")
		assert.Contains(t, firstTask, "title")
		assert.Contains(t, firstTask, "status")
	}
}
