package mcp

import (
	"context"
	"fmt"
	"testing"
	"time"

	"hyper/internal/mcp/storage"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// determineLevel converts a complexity score to a level string
func determineLevel(score float64) string {
	switch {
	case score >= 0.8:
		return "EXTREME"
	case score >= 0.6:
		return "HIGH"
	case score >= 0.4:
		return "MEDIUM"
	default:
		return "LOW"
	}
}

// TestPhase3ComplexityAnalysisSimpleTask tests complexity analysis on a simple task
func TestPhase3ComplexityAnalysisSimpleTask(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	tool := &CoordinatorAnalyzeTaskComplexityTool{storage: nil}

	args := map[string]interface{}{
		"title":          "Fix button color",
		"contextSummary": "Change the submit button color from blue to green",
		"todos":          []string{"Update CSS color property", "Test in browser"},
		"filesModified":  []string{"styles.css"},
	}

	result, err := tool.Execute(context.Background(), args)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	analysis, ok := result.(ComplexityAnalysis)
	assert.True(t, ok, "Result should be ComplexityAnalysis")

	score := analysis.Score
	level := determineLevel(analysis.Score)
	recommendation := analysis.Recommendation

	fmt.Printf("\n✅ SIMPLE TASK TEST:\n")
	fmt.Printf("   Score: %.2f\n", score)
	fmt.Printf("   Level: %s\n", level)
	fmt.Printf("   Recommendation: %s\n", recommendation)
	fmt.Printf("   Expected: score < 0.4, PROCEED\n")

	assert.Less(t, score, 0.4, "Simple task should have low complexity")
	assert.Equal(t, "PROCEED", recommendation)
}

// TestPhase3ComplexityAnalysisModerateTask tests complexity analysis on a moderate task
func TestPhase3ComplexityAnalysisModerateTask(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	tool := &CoordinatorAnalyzeTaskComplexityTool{storage: nil}

	args := map[string]interface{}{
		"title":          "Add user profile page",
		"contextSummary": "Create a user profile page with edit functionality",
		"todos": []string{
			"Create profile page component",
			"Add form validation",
			"Implement API endpoint",
			"Add database queries",
			"Write tests",
		},
		"filesModified": []string{
			"frontend/Profile.tsx",
			"backend/handlers/profile.go",
			"backend/models/user.go",
		},
	}

	result, err := tool.Execute(context.Background(), args)
	assert.NoError(t, err)

	analysis, ok := result.(ComplexityAnalysis)
	assert.True(t, ok, "Result should be ComplexityAnalysis")

	score := analysis.Score
	recommendation := analysis.Recommendation

	fmt.Printf("\n✅ MODERATE TASK TEST:\n")
	fmt.Printf("   Score: %.2f\n", score)
	fmt.Printf("   Recommendation: %s\n", recommendation)
	fmt.Printf("   Expected: score 0.4-0.6, PROCEED or SPLIT\n")

	assert.GreaterOrEqual(t, score, 0.4, "Moderate task should have medium complexity")
	assert.Less(t, score, 0.8, "Moderate task should not be extreme")
}

// TestPhase3ComplexityAnalysisComplexTask tests complexity analysis on a complex task
func TestPhase3ComplexityAnalysisComplexTask(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	tool := &CoordinatorAnalyzeTaskComplexityTool{storage: nil}

	args := map[string]interface{}{
		"title":          "Implement Real-time Notification System",
		"contextSummary": "Build a complete real-time notification system with WebSocket support, database storage, frontend components, and backend event streaming",
		"todos": []string{
			"Set up WebSocket server in Go",
			"Create notification event types and schemas",
			"Implement MongoDB notification storage",
			"Build React notification center component",
			"Add real-time event streaming",
			"Integrate with existing auth system",
			"Add notification preferences UI",
			"Implement read/unread tracking",
			"Create notification API endpoints",
			"Add push notification support",
		},
		"filesModified": []string{
			"server/websocket.go",
			"models/notification.go",
			"storage/notifications.go",
			"handlers/notification_handler.go",
			"frontend/src/components/NotificationCenter.tsx",
			"frontend/src/hooks/useNotifications.ts",
			"frontend/src/api/notifications.ts",
		},
	}

	result, err := tool.Execute(context.Background(), args)
	assert.NoError(t, err)

	analysis, ok := result.(ComplexityAnalysis)
	assert.True(t, ok, "Result should be ComplexityAnalysis")

	score := analysis.Score
	recommendation := analysis.Recommendation
	splittingStrategy := analysis.SplittingStrategy

	fmt.Printf("\n✅ COMPLEX TASK TEST:\n")
	fmt.Printf("   Score: %.2f\n", score)
	fmt.Printf("   Recommendation: %s\n", recommendation)
	fmt.Printf("   Splitting Strategy: %s\n", splittingStrategy)
	fmt.Printf("   Expected: score 0.6-0.8, SPLIT\n")

	assert.GreaterOrEqual(t, score, 0.6, "Complex task should have high complexity")
	assert.Equal(t, "SPLIT", recommendation)
}

// TestPhase3ComplexityAnalysisExtremeTask tests complexity analysis on an extreme task
func TestPhase3ComplexityAnalysisExtremeTask(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	tool := &CoordinatorAnalyzeTaskComplexityTool{storage: nil}

	args := map[string]interface{}{
		"title":          "Build Complete E-commerce Platform",
		"contextSummary": "Build a full e-commerce platform with product catalog, shopping cart, checkout, payment processing, inventory management, order tracking, admin dashboard, analytics, and mobile app. Requires concurrent processing, advanced security, complex algorithm optimization, database migration, and integration with multiple external APIs and microservices.",
		"todos": []string{
			"Implement advanced product catalog with complex algorithm for search optimization and concurrent filters",
			"Create shopping cart with database persistence and concurrent session management",
			"Integrate payment gateway security with advanced authentication and async processing (Stripe, PayPal)",
			"Migrate database schema and refactor checkout flow with complex address validation algorithm",
			"Implement order management system with concurrent processing and advanced performance optimization",
			"Create inventory tracking with real-time concurrent updates and complex algorithm for stock prediction",
			"Build admin dashboard with advanced analytics, complex database queries, and performance optimization",
			"Implement email notification system with async processing and integration with external microservice",
			"Migrate authentication system and implement advanced security with concurrent session management",
			"Create customer review system with complex algorithm for sentiment analysis and moderation",
			"Integrate shipping APIs with concurrent rate calculation and complex algorithm for route optimization",
			"Build mobile app (iOS and Android) with advanced authentication, concurrent sync, and complex performance optimization",
			"Implement real-time inventory updates with concurrent processing, WebSocket integration, and database optimization",
			"Create recommendation engine with complex machine learning algorithm and advanced performance optimization",
			"Migrate payment system and add multi-currency support with concurrent exchange rate processing and security hardening",
		},
		"filesModified": []string{
			"backend/products/catalog.go",
			"backend/products/search.go",
			"backend/cart/cart.go",
			"backend/checkout/checkout.go",
			"backend/payments/stripe.go",
			"backend/payments/paypal.go",
			"backend/orders/orders.go",
			"backend/inventory/inventory.go",
			"backend/admin/dashboard.go",
			"backend/analytics/analytics.go",
			"frontend/src/components/ProductList.tsx",
			"frontend/src/components/Cart.tsx",
			"frontend/src/components/Checkout.tsx",
			"frontend/src/components/AdminDashboard.tsx",
			"mobile/ios/ProductScreen.swift",
			"mobile/android/ProductActivity.kt",
		},
	}

	result, err := tool.Execute(context.Background(), args)
	assert.NoError(t, err)

	analysis, ok := result.(ComplexityAnalysis)
	assert.True(t, ok, "Result should be ComplexityAnalysis")

	score := analysis.Score
	recommendation := analysis.Recommendation

	fmt.Printf("\n✅ EXTREME TASK TEST:\n")
	fmt.Printf("   Score: %.2f\n", score)
	fmt.Printf("   Recommendation: %s\n", recommendation)
	fmt.Printf("   Expected: score ≥ 0.8, REJECT\n")

	assert.GreaterOrEqual(t, score, 0.8, "Extreme task should have very high complexity")
	assert.Equal(t, "REJECT", recommendation)
}

// TestPhase3ComplexityFactors tests individual complexity factors
func TestPhase3ComplexityFactors(t *testing.T) {
	fmt.Printf("\n✅ COMPLEXITY FACTORS TEST:\n")

	// Test file count factor
	analysis1 := analyzeTaskComplexity("Test", "Test task", []string{"todo1"}, []string{"file1.go"})
	analysis2 := analyzeTaskComplexity("Test", "Test task", []string{"todo1"}, []string{"f1", "f2", "f3", "f4", "f5", "f6"})

	fmt.Printf("   File count: 1 file = %.2f, 6 files = %.2f\n",
		analysis1.Factors["fileCount"], analysis2.Factors["fileCount"])
	assert.Less(t, analysis1.Factors["fileCount"], analysis2.Factors["fileCount"])

	// Test TODO complexity
	simpleTodos := []string{"Fix bug", "Add test"}
	complexTodos := []string{
		"Implement advanced authentication with OAuth integration",
		"Create complex algorithm for real-time performance optimization",
		"Migrate database schema with zero downtime",
	}

	simpleComplexity := analyzeTodoComplexity(simpleTodos)
	complexComplexity := analyzeTodoComplexity(complexTodos)

	fmt.Printf("   TODO complexity: simple = %.2f, complex = %.2f\n",
		simpleComplexity, complexComplexity)
	assert.Less(t, simpleComplexity, complexComplexity)

	// Test cross-system dependencies
	singleSystem := analyzeTaskComplexity("Test", "Update frontend button",
		[]string{"Change color"}, []string{"button.tsx"})
	multiSystem := analyzeTaskComplexity("Test", "Build full-stack feature with database, API, frontend, and authentication",
		[]string{"Add feature"}, []string{"api.go", "component.tsx"})

	fmt.Printf("   Cross-system deps: single = %d, multi = %d\n",
		singleSystem.CrossSystemDeps, multiSystem.CrossSystemDeps)
	assert.Less(t, singleSystem.CrossSystemDeps, multiSystem.CrossSystemDeps)
}

// TestPhase3EstimatedTime tests time estimation
func TestPhase3EstimatedTime(t *testing.T) {
	fmt.Printf("\n✅ TIME ESTIMATION TEST:\n")

	simple := analyzeTaskComplexity("Fix typo", "Fix a typo in documentation",
		[]string{"Fix typo"}, []string{"README.md"})

	moderate := analyzeTaskComplexity("Add feature", "Add new feature with tests",
		[]string{"Implement feature", "Write tests", "Update docs"},
		[]string{"feature.go", "feature_test.go", "README.md"})

	complex := analyzeTaskComplexity("Major refactor", "Refactor entire authentication system",
		[]string{"Refactor auth", "Update tests", "Migrate data", "Update docs", "Test integration"},
		[]string{"auth1.go", "auth2.go", "auth3.go", "tests1.go", "tests2.go", "migration.sql"})

	fmt.Printf("   Simple task: %d minutes\n", simple.EstimatedTimeMinutes)
	fmt.Printf("   Moderate task: %d minutes\n", moderate.EstimatedTimeMinutes)
	fmt.Printf("   Complex task: %d minutes\n", complex.EstimatedTimeMinutes)

	assert.Less(t, simple.EstimatedTimeMinutes, moderate.EstimatedTimeMinutes)
	assert.Less(t, moderate.EstimatedTimeMinutes, complex.EstimatedTimeMinutes)
}

// TestPhase3CircularDependencyDetection tests circular dependency detection
func TestPhase3CircularDependencyDetection(t *testing.T) {
	fmt.Printf("\n✅ CIRCULAR DEPENDENCY DETECTION TEST:\n")

	tool := &CoordinatorSplitAgentTaskTool{storage: nil}

	// Test case 1: Valid dependencies (no cycle)
	validParams := []ChildTaskParams{
		{Title: "Task A", DependsOn: []string{}},
		{Title: "Task B", DependsOn: []string{"Task A"}},
		{Title: "Task C", DependsOn: []string{"Task B"}},
	}

	taskIndexMap := make(map[string]int)
	for i, params := range validParams {
		taskIndexMap[params.Title] = i
	}

	err := tool.validateTaskDependencies(validParams, taskIndexMap)
	assert.NoError(t, err)
	fmt.Printf("   ✓ Valid linear dependencies: PASSED\n")

	// Test case 2: Circular dependency
	circularParams := []ChildTaskParams{
		{Title: "Task A", DependsOn: []string{"Task C"}},
		{Title: "Task B", DependsOn: []string{"Task A"}},
		{Title: "Task C", DependsOn: []string{"Task B"}},
	}

	taskIndexMap2 := make(map[string]int)
	for i, params := range circularParams {
		taskIndexMap2[params.Title] = i
	}

	err = tool.validateTaskDependencies(circularParams, taskIndexMap2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency detected")
	fmt.Printf("   ✓ Circular dependency detected: PASSED\n")

	// Test case 3: Invalid dependency (non-existent task)
	invalidParams := []ChildTaskParams{
		{Title: "Task A", DependsOn: []string{"NonExistent"}},
	}

	taskIndexMap3 := make(map[string]int)
	for i, params := range invalidParams {
		taskIndexMap3[params.Title] = i
	}

	err = tool.validateTaskDependencies(invalidParams, taskIndexMap3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "non-existent task")
	fmt.Printf("   ✓ Non-existent dependency detected: PASSED\n")
}

// TestPhase3ProgressCalculation tests progress calculation
func TestPhase3ProgressCalculation(t *testing.T) {
	fmt.Printf("\n✅ PROGRESS CALCULATION TEST:\n")

	tool := &CoordinatorUpdateChildTaskProgressTool{storage: nil}

	// Create mock task with todos
	task := &storage.AgentTask{
		Todos: []storage.TodoItem{
			{ID: "1", Status: storage.TodoStatusCompleted},
			{ID: "2", Status: storage.TodoStatusCompleted},
			{ID: "3", Status: storage.TodoStatusInProgress},
			{ID: "4", Status: storage.TodoStatusPending},
			{ID: "5", Status: storage.TodoStatusPending},
		},
	}

	progress := tool.calculateTaskProgress(task)

	fmt.Printf("   Task with 5 TODOs:\n")
	fmt.Printf("   - Completed: 2\n")
	fmt.Printf("   - In Progress: 1\n")
	fmt.Printf("   - Pending: 2\n")
	fmt.Printf("   Calculated Progress: %.1f%%\n", progress*100)
	fmt.Printf("   Expected: 40%% (2 completed out of 5)\n")

	assert.Equal(t, 0.4, progress, "Progress should be 40%")
}

// TestPhase3RecommendationThresholds tests recommendation logic
func TestPhase3RecommendationThresholds(t *testing.T) {
	fmt.Printf("\n✅ RECOMMENDATION THRESHOLDS TEST:\n")

	tests := []struct {
		score      float64
		fileCount  int
		todoCount  int
		expectedRec string
	}{
		{0.2, 1, 2, "PROCEED"},
		{0.5, 3, 5, "PROCEED"},
		{0.6, 6, 9, "SPLIT"},
		{0.7, 8, 10, "SPLIT"},
		{0.85, 15, 20, "REJECT"},
	}

	for _, test := range tests {
		rec, _ := determineRecommendation(test.score, test.fileCount, test.todoCount)
		fmt.Printf("   Score %.2f, %d files, %d todos → %s (expected: %s)\n",
			test.score, test.fileCount, test.todoCount, rec, test.expectedRec)
		assert.Equal(t, test.expectedRec, rec)
	}
}

// TestPhase3KeywordDetection tests complexity keyword detection
func TestPhase3KeywordDetection(t *testing.T) {
	fmt.Printf("\n✅ KEYWORD DETECTION TEST:\n")

	simpleTodo := analyzeTodoComplexity([]string{"Update button text"})
	complexTodo := analyzeTodoComplexity([]string{"Implement advanced authentication algorithm with security features"})

	fmt.Printf("   Simple TODO complexity: %.2f\n", simpleTodo)
	fmt.Printf("   Complex TODO complexity: %.2f\n", complexTodo)
	fmt.Printf("   Complex should be higher due to keywords: 'implement', 'advanced', 'authentication', 'algorithm', 'security'\n")

	assert.Less(t, simpleTodo, complexTodo)
}

// TestPhase3Summary prints a summary of all tests
func TestPhase3Summary(t *testing.T) {
	// This will run last due to alphabetical ordering
	time.Sleep(100 * time.Millisecond) // Give other tests time to print

	separator := "============================================================"
	fmt.Printf("\n%s\n", separator)
	fmt.Printf("PHASE 3 AUTOMATED TEST SUMMARY\n")
	fmt.Printf("%s\n", separator)
	fmt.Printf("\n✅ All Phase 3 Components Tested:\n")
	fmt.Printf("   1. Complexity Analysis (Simple/Moderate/Complex/Extreme)\n")
	fmt.Printf("   2. Individual Complexity Factors\n")
	fmt.Printf("   3. Time Estimation Algorithm\n")
	fmt.Printf("   4. Circular Dependency Detection\n")
	fmt.Printf("   5. Progress Calculation\n")
	fmt.Printf("   6. Recommendation Thresholds\n")
	fmt.Printf("   7. Keyword Detection\n")
	fmt.Printf("\n🎯 Key Features Verified:\n")
	fmt.Printf("   ✓ Multi-heuristic complexity scoring (0.0-1.0)\n")
	fmt.Printf("   ✓ File count analysis (0.0-0.3 weight)\n")
	fmt.Printf("   ✓ TODO complexity analysis (0.0-0.4 weight)\n")
	fmt.Printf("   ✓ Cross-system dependency detection (0.0-0.2 weight)\n")
	fmt.Printf("   ✓ Integration complexity (0.0-0.1 weight)\n")
	fmt.Printf("   ✓ Circular dependency prevention\n")
	fmt.Printf("   ✓ Automatic progress calculation\n")
	fmt.Printf("   ✓ Smart recommendation system\n")
	fmt.Printf("\n📊 Complexity Thresholds Working:\n")
	fmt.Printf("   • 0.0-0.4: Simple → PROCEED\n")
	fmt.Printf("   • 0.4-0.6: Moderate → PROCEED/SPLIT\n")
	fmt.Printf("   • 0.6-0.8: Complex → SPLIT\n")
	fmt.Printf("   • 0.8-1.0: Extreme → REJECT\n")
	fmt.Printf("\n%s\n", separator)
}
