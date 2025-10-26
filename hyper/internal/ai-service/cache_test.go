package aiservice

import (
	"testing"
)

func TestToolResultCache_DeletePrefix(t *testing.T) {
	cache := NewToolResultCache()

	// Add test entries with various prefixes
	testCases := []struct {
		signature string
		result    *ToolResult
	}{
		{
			signature: "coordinator_list_human_tasks:{}",
			result:    &ToolResult{Name: "coordinator_list_human_tasks", Output: []interface{}{}, Error: ""},
		},
		{
			signature: "coordinator_list_human_tasks:{\"status\":\"pending\"}",
			result:    &ToolResult{Name: "coordinator_list_human_tasks", Output: []interface{}{}, Error: ""},
		},
		{
			signature: "coordinator_list_agent_tasks:{}",
			result:    &ToolResult{Name: "coordinator_list_agent_tasks", Output: []interface{}{}, Error: ""},
		},
		{
			signature: "coordinator_list_agent_tasks:{\"humanTaskId\":\"123\"}",
			result:    &ToolResult{Name: "coordinator_list_agent_tasks", Output: []interface{}{}, Error: ""},
		},
		{
			signature: "coordinator_get_agent_task:{\"taskId\":\"456\"}",
			result:    &ToolResult{Name: "coordinator_get_agent_task", Output: map[string]interface{}{}, Error: ""},
		},
		{
			signature: "coordinator_get_agent_task:{\"taskId\":\"789\"}",
			result:    &ToolResult{Name: "coordinator_get_agent_task", Output: map[string]interface{}{}, Error: ""},
		},
		{
			signature: "some_other_tool:{}",
			result:    &ToolResult{Name: "some_other_tool", Output: "test", Error: ""},
		},
	}

	// Populate cache
	for _, tc := range testCases {
		cache.Set(tc.signature, tc.result)
	}

	// Verify all entries are cached
	if len(cache.cache) != 7 {
		t.Errorf("Expected 7 cache entries, got %d", len(cache.cache))
	}

	// Test DeletePrefix for coordinator_list_human_tasks
	t.Run("DeletePrefix_list_human_tasks", func(t *testing.T) {
		count := cache.DeletePrefix("coordinator_list_human_tasks:")
		if count != 2 {
			t.Errorf("Expected to delete 2 entries, deleted %d", count)
		}

		// Verify the entries are gone
		result, exists := cache.Get("coordinator_list_human_tasks:{}")
		if exists {
			t.Errorf("Expected entry to be deleted, but it still exists: %+v", result)
		}

		result, exists = cache.Get("coordinator_list_human_tasks:{\"status\":\"pending\"}")
		if exists {
			t.Errorf("Expected entry to be deleted, but it still exists: %+v", result)
		}

		// Verify other entries remain
		if len(cache.cache) != 5 {
			t.Errorf("Expected 5 remaining cache entries, got %d", len(cache.cache))
		}
	})

	// Test DeletePrefix for coordinator_list_agent_tasks
	t.Run("DeletePrefix_list_agent_tasks", func(t *testing.T) {
		count := cache.DeletePrefix("coordinator_list_agent_tasks:")
		if count != 2 {
			t.Errorf("Expected to delete 2 entries, deleted %d", count)
		}

		// Verify remaining count
		if len(cache.cache) != 3 {
			t.Errorf("Expected 3 remaining cache entries, got %d", len(cache.cache))
		}
	})

	// Test DeletePrefix for coordinator_get_agent_task
	t.Run("DeletePrefix_get_agent_task", func(t *testing.T) {
		count := cache.DeletePrefix("coordinator_get_agent_task:")
		if count != 2 {
			t.Errorf("Expected to delete 2 entries, deleted %d", count)
		}

		// Verify remaining count
		if len(cache.cache) != 1 {
			t.Errorf("Expected 1 remaining cache entry, got %d", len(cache.cache))
		}

		// Verify the unrelated entry remains
		result, exists := cache.Get("some_other_tool:{}")
		if !exists {
			t.Error("Expected unrelated entry to remain, but it was deleted")
		}
		if result.Name != "some_other_tool" {
			t.Errorf("Expected remaining entry to be 'some_other_tool', got '%s'", result.Name)
		}
	})
}

func TestToolResultCache_DeletePrefix_EmptyCache(t *testing.T) {
	cache := NewToolResultCache()

	// Test DeletePrefix on empty cache
	count := cache.DeletePrefix("coordinator_list_human_tasks:")
	if count != 0 {
		t.Errorf("Expected to delete 0 entries from empty cache, deleted %d", count)
	}
}

func TestToolResultCache_DeletePrefix_NoMatches(t *testing.T) {
	cache := NewToolResultCache()

	// Add unrelated entries
	cache.Set("some_tool:{}", &ToolResult{Name: "some_tool", Output: "test", Error: ""})
	cache.Set("another_tool:{}", &ToolResult{Name: "another_tool", Output: "test", Error: ""})

	// Test DeletePrefix with no matching entries
	count := cache.DeletePrefix("coordinator_list_human_tasks:")
	if count != 0 {
		t.Errorf("Expected to delete 0 entries, deleted %d", count)
	}

	// Verify original entries remain
	if len(cache.cache) != 2 {
		t.Errorf("Expected 2 cache entries to remain, got %d", len(cache.cache))
	}
}

func TestToolResultCache_DeletePrefix_ConcurrentAccess(t *testing.T) {
	cache := NewToolResultCache()

	// Populate cache with many entries
	for i := 0; i < 100; i++ {
		signature := "coordinator_list_human_tasks:{\"id\":\"" + string(rune(i)) + "\"}"
		cache.Set(signature, &ToolResult{Name: "coordinator_list_human_tasks", Output: []interface{}{}, Error: ""})
	}

	// Test concurrent DeletePrefix calls
	done := make(chan bool)
	go func() {
		count := cache.DeletePrefix("coordinator_list_human_tasks:")
		if count != 100 {
			t.Errorf("Expected to delete 100 entries, deleted %d", count)
		}
		done <- true
	}()

	// Wait for completion
	<-done

	// Verify all entries are gone
	if len(cache.cache) != 0 {
		t.Errorf("Expected empty cache, got %d entries", len(cache.cache))
	}
}
