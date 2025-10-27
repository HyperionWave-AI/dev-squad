import { test, expect } from '@playwright/test';

test.describe('Kanban Drag and Drop Functionality', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    // Navigate to tasks/kanban page
    await page.click('[data-testid="tasks-nav"]');
    await page.waitForLoadState('networkidle');
  });

  test('should drag task between columns successfully', async ({ page }) => {
    // Wait for tasks to load
    await page.waitForSelector('[data-testid="task-card"]');
    
    // Find first task in Pending column
    const pendingColumn = page.locator('[data-testid="kanban-column"]:has-text("Pending")');
    const taskToDrag = pendingColumn.locator('[data-testid="task-card"]').first();
    
    // Find In Progress column
    const inProgressColumn = page.locator('[data-testid="kanban-column"]:has-text("In Progress")');
    
    // Get initial task counts
    const initialPendingCount = await pendingColumn.locator('[data-testid="task-count"]').textContent();
    const initialInProgressCount = await inProgressColumn.locator('[data-testid="task-count"]').textContent();
    
    // Perform drag and drop
    await taskToDrag.dragTo(inProgressColumn);
    
    // Wait for optimistic update
    await page.waitForTimeout(500);
    
    // Verify task moved to new column
    const taskInNewColumn = inProgressColumn.locator('[data-testid="task-card"]').first();
    await expect(taskInNewColumn).toBeVisible();
    
    // Verify task counts updated
    const newPendingCount = await pendingColumn.locator('[data-testid="task-count"]').textContent();
    const newInProgressCount = await inProgressColumn.locator('[data-testid="task-count"]').textContent();
    
    expect(parseInt(newPendingCount || '0')).toBe(parseInt(initialPendingCount || '0') - 1);
    expect(parseInt(newInProgressCount || '0')).toBe(parseInt(initialInProgressCount || '0') + 1);
  });

  test('should handle drag cancellation', async ({ page }) => {
    await page.waitForSelector('[data-testid="task-card"]');
    
    const pendingColumn = page.locator('[data-testid="kanban-column"]:has-text("Pending")');
    const taskToDrag = pendingColumn.locator('[data-testid="task-card"]').first();
    
    // Start drag
    await taskToDrag.hover();
    await page.mouse.down();
    
    // Move slightly but not to a valid drop zone
    await page.mouse.move(100, 100);
    
    // Cancel drag by pressing Escape
    await page.keyboard.press('Escape');
    
    // Verify task remains in original position
    await expect(pendingColumn.locator('[data-testid="task-card"]').first()).toBeVisible();
  });

  test('should provide visual feedback during drag', async ({ page }) => {
    await page.waitForSelector('[data-testid="task-card"]');
    
    const taskToDrag = page.locator('[data-testid="task-card"]').first();
    const targetColumn = page.locator('[data-testid="kanban-column"]').nth(1);
    
    // Start drag
    await taskToDrag.hover();
    await page.mouse.down();
    
    // Check for drag visual feedback
    await expect(taskToDrag).toHaveClass(/dragging|drag-preview/);
    
    // Move over target column
    await targetColumn.hover();
    
    // Check for drop zone visual feedback
    await expect(targetColumn).toHaveClass(/drag-over|drop-zone-active/);
    
    // Complete drag
    await page.mouse.up();
  });

  test('should handle keyboard drag and drop', async ({ page }) => {
    await page.waitForSelector('[data-testid="task-card"]');
    
    const taskToDrag = page.locator('[data-testid="task-card"]').first();
    
    // Focus on task
    await taskToDrag.focus();
    await expect(taskToDrag).toBeFocused();
    
    // Initiate keyboard drag with Space
    await page.keyboard.press('Space');
    
    // Navigate to target column with arrow keys
    await page.keyboard.press('ArrowRight');
    await page.keyboard.press('ArrowRight');
    
    // Drop with Space
    await page.keyboard.press('Space');
    
    // Verify task moved
    const inProgressColumn = page.locator('[data-testid="kanban-column"]:has-text("In Progress")');
    await expect(inProgressColumn.locator('[data-testid="task-card"]').first()).toBeVisible();
  });

  test('should handle drag to same column (no-op)', async ({ page }) => {
    await page.waitForSelector('[data-testid="task-card"]');
    
    const pendingColumn = page.locator('[data-testid="kanban-column"]:has-text("Pending")');
    const taskToDrag = pendingColumn.locator('[data-testid="task-card"]').first();
    
    const initialCount = await pendingColumn.locator('[data-testid="task-count"]').textContent();
    
    // Drag task to same column
    await taskToDrag.dragTo(pendingColumn);
    
    // Verify count unchanged
    const finalCount = await pendingColumn.locator('[data-testid="task-count"]').textContent();
    expect(finalCount).toBe(initialCount);
  });

  test('should handle multiple rapid drags', async ({ page }) => {
    await page.waitForSelector('[data-testid="task-card"]');
    
    const tasks = await page.locator('[data-testid="task-card"]').all();
    const targetColumn = page.locator('[data-testid="kanban-column"]').nth(1);
    
    // Perform multiple rapid drags
    for (let i = 0; i < Math.min(3, tasks.length); i++) {
      await tasks[i].dragTo(targetColumn);
      await page.waitForTimeout(100); // Small delay between drags
    }
    
    // Verify all tasks moved
    const movedTasks = await targetColumn.locator('[data-testid="task-card"]').count();
    expect(movedTasks).toBeGreaterThanOrEqual(3);
  });

  test('should handle drag with network error', async ({ page }) => {
    // Intercept API calls and simulate failure
    await page.route('**/api/tasks/**', route => {
      route.abort('failed');
    });
    
    await page.waitForSelector('[data-testid="task-card"]');
    
    const pendingColumn = page.locator('[data-testid="kanban-column"]:has-text("Pending")');
    const taskToDrag = pendingColumn.locator('[data-testid="task-card"]').first();
    const targetColumn = page.locator('[data-testid="kanban-column"]').nth(1);
    
    // Perform drag
    await taskToDrag.dragTo(targetColumn);
    
    // Wait for error handling
    await page.waitForTimeout(2000);
    
    // Verify error message appears
    await expect(page.locator('[role="alert"]')).toBeVisible();
    
    // Verify task rolled back to original position
    await expect(pendingColumn.locator('[data-testid="task-card"]').first()).toBeVisible();
  });

  test('should handle drag with slow network', async ({ page }) => {
    // Simulate slow network
    await page.route('**/api/tasks/**', async route => {
      await new Promise(resolve => setTimeout(resolve, 3000));
      route.continue();
    });
    
    await page.waitForSelector('[data-testid="task-card"]');
    
    const taskToDrag = page.locator('[data-testid="task-card"]').first();
    const targetColumn = page.locator('[data-testid="kanban-column"]').nth(1);
    
    // Perform drag
    await taskToDrag.dragTo(targetColumn);
    
    // Verify optimistic update happened immediately
    await expect(targetColumn.locator('[data-testid="task-card"]').first()).toBeVisible();
    
    // Verify loading indicator appears
    await expect(page.locator('[data-testid="loading-indicator"]')).toBeVisible();
    
    // Wait for network request to complete
    await page.waitForTimeout(4000);
    
    // Verify loading indicator disappears
    await expect(page.locator('[data-testid="loading-indicator"]')).not.toBeVisible();
  });

  test('should preserve task order within columns', async ({ page }) => {
    await page.waitForSelector('[data-testid="task-card"]');
    
    const sourceColumn = page.locator('[data-testid="kanban-column"]').first();
    const tasks = await sourceColumn.locator('[data-testid="task-card"]').all();
    
    if (tasks.length < 2) {
      test.skip('Need at least 2 tasks for order testing');
    }
    
    // Get initial order
    const initialOrder = await Promise.all(
      tasks.map(task => task.locator('[data-testid="task-title"]').textContent())
    );
    
    // Drag first task to different column and back
    const targetColumn = page.locator('[data-testid="kanban-column"]').nth(1);
    await tasks[0].dragTo(targetColumn);
    await page.waitForTimeout(500);
    await targetColumn.locator('[data-testid="task-card"]').first().dragTo(sourceColumn);
    await page.waitForTimeout(500);
    
    // Verify order preserved
    const finalTasks = await sourceColumn.locator('[data-testid="task-card"]').all();
    const finalOrder = await Promise.all(
      finalTasks.map(task => task.locator('[data-testid="task-title"]').textContent())
    );
    
    expect(finalOrder).toEqual(initialOrder);
  });

  test('should handle drag with invalid drop zones', async ({ page }) => {
    await page.waitForSelector('[data-testid="task-card"]');
    
    const taskToDrag = page.locator('[data-testid="task-card"]').first();
    const invalidDropZone = page.locator('header'); // Header is not a valid drop zone
    
    // Attempt to drag to invalid zone
    await taskToDrag.dragTo(invalidDropZone);
    
    // Verify task returns to original position
    const originalColumn = page.locator('[data-testid="kanban-column"]').first();
    await expect(originalColumn.locator('[data-testid="task-card"]').first()).toBeVisible();
  });

  test('should handle concurrent drags from different users', async ({ page, context }) => {
    // Create second page to simulate another user
    const page2 = await context.newPage();
    await page2.goto('/');
    await page2.click('[data-testid="tasks-nav"]');
    await page2.waitForLoadState('networkidle');
    
    await page.waitForSelector('[data-testid="task-card"]');
    await page2.waitForSelector('[data-testid="task-card"]');
    
    // Simulate concurrent drags
    const task1 = page.locator('[data-testid="task-card"]').first();
    const task2 = page2.locator('[data-testid="task-card"]').nth(1);
    
    const targetColumn1 = page.locator('[data-testid="kanban-column"]').nth(1);
    const targetColumn2 = page2.locator('[data-testid="kanban-column"]').nth(2);
    
    // Perform concurrent drags
    await Promise.all([
      task1.dragTo(targetColumn1),
      task2.dragTo(targetColumn2)
    ]);
    
    // Wait for both operations to complete
    await page.waitForTimeout(2000);
    await page2.waitForTimeout(2000);
    
    // Verify both operations succeeded or handled gracefully
    // At minimum, no errors should be thrown
    await expect(page.locator('[role="alert"]')).not.toBeVisible();
    await expect(page2.locator('[role="alert"]')).not.toBeVisible();
    
    await page2.close();
  });

  test('should maintain accessibility during drag operations', async ({ page }) => {
    await page.waitForSelector('[data-testid="task-card"]');
    
    const taskToDrag = page.locator('[data-testid="task-card"]').first();
    
    // Focus on task
    await taskToDrag.focus();
    
    // Verify ARIA attributes during drag
    await page.keyboard.press('Space'); // Start keyboard drag
    
    // Check for proper ARIA states
    await expect(taskToDrag).toHaveAttribute('aria-grabbed', 'true');
    
    // Check for live region announcements
    const liveRegion = page.locator('[aria-live="assertive"]');
    await expect(liveRegion).toContainText(/grabbed|selected|dragging/i);
    
    // Complete drag
    await page.keyboard.press('ArrowRight');
    await page.keyboard.press('Space');
    
    // Verify completion announcement
    await expect(liveRegion).toContainText(/dropped|moved|completed/i);
  });
});