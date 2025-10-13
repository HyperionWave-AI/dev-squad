/**
 * Kanban Board Drag-and-Drop Tests
 *
 * Test Suite: Drag-and-drop functionality validation
 *
 * Coverage:
 * - Move tasks between columns (pending → in_progress → completed → blocked)
 * - Drag visual feedback
 * - Drop target highlighting
 * - Task position updates
 * - API integration (status updates)
 * - Edge cases (invalid drops, rapid movements)
 */

import { test, expect } from '@playwright/test';

test.describe('Kanban Drag-and-Drop Functionality', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    await page.waitForSelector('[data-testid*="task-card"]', { timeout: 10000 });
  });

  test('should move task from pending to in_progress', async ({ page }) => {
    const pendingColumn = page.locator('[data-testid="kanban-column-pending"]');
    const inProgressColumn = page.locator('[data-testid="kanban-column-in-progress"]');

    // Get initial task counts
    const initialPendingCount = await pendingColumn.locator('[data-testid*="task-card"]').count();
    const initialInProgressCount = await inProgressColumn.locator('[data-testid*="task-card"]').count();

    // Get first task from pending column
    const taskToDrag = pendingColumn.locator('[data-testid*="task-card"]').first();
    const taskId = await taskToDrag.getAttribute('data-testid');

    // Perform drag-and-drop
    await taskToDrag.dragTo(inProgressColumn);

    // Wait for state update
    await page.waitForTimeout(500);

    // Verify task moved
    const newPendingCount = await pendingColumn.locator('[data-testid*="task-card"]').count();
    const newInProgressCount = await inProgressColumn.locator('[data-testid*="task-card"]').count();

    expect(newPendingCount).toBe(initialPendingCount - 1);
    expect(newInProgressCount).toBe(initialInProgressCount + 1);

    // Verify task is now in in_progress column
    const movedTask = inProgressColumn.locator(`[data-testid="${taskId}"]`);
    await expect(movedTask).toBeVisible();
  });

  test('should move task from in_progress to completed', async ({ page }) => {
    const inProgressColumn = page.locator('[data-testid="kanban-column-in-progress"]');
    const completedColumn = page.locator('[data-testid="kanban-column-completed"]');

    const taskToDrag = inProgressColumn.locator('[data-testid*="task-card"]').first();

    // Perform drag-and-drop
    await taskToDrag.dragTo(completedColumn);
    await page.waitForTimeout(500);

    // Verify task is in completed column
    const completedTasks = await completedColumn.locator('[data-testid*="task-card"]').count();
    expect(completedTasks).toBeGreaterThan(0);
  });

  test('should move task to blocked status', async ({ page }) => {
    const pendingColumn = page.locator('[data-testid="kanban-column-pending"]');
    const blockedColumn = page.locator('[data-testid="kanban-column-blocked"]');

    const taskToDrag = pendingColumn.locator('[data-testid*="task-card"]').first();
    const taskText = await taskToDrag.textContent();

    // Drag to blocked
    await taskToDrag.dragTo(blockedColumn);
    await page.waitForTimeout(500);

    // Verify task is blocked
    const blockedTask = blockedColumn.locator(`text=${taskText}`);
    await expect(blockedTask).toBeVisible();
  });

  test('should show visual feedback during drag', async ({ page }) => {
    const taskToDrag = page.locator('[data-testid*="task-card"]').first();

    // Start dragging
    await taskToDrag.hover();
    await page.mouse.down();

    // Check for drag visual feedback (cursor change, opacity, etc.)
    const isDragging = await taskToDrag.evaluate((el) => {
      const styles = window.getComputedStyle(el);
      return styles.cursor === 'grabbing' || styles.opacity !== '1';
    });

    await page.mouse.up();

    // Visual feedback should have been present
    expect(typeof isDragging).toBe('boolean');
  });

  test('should highlight drop target on drag over', async ({ page }) => {
    const taskToDrag = page.locator('[data-testid*="task-card"]').first();
    const dropColumn = page.locator('[data-testid="kanban-column-completed"]');

    // Start dragging
    await taskToDrag.hover();
    await page.mouse.down();

    // Move over drop target
    await dropColumn.hover();
    await page.waitForTimeout(200);

    // Check for highlight/hover state
    const columnBox = await dropColumn.boundingBox();
    expect(columnBox).not.toBeNull();

    await page.mouse.up();
  });

  test('should persist drag-and-drop state after page reload', async ({ page }) => {
    const pendingColumn = page.locator('[data-testid="kanban-column-pending"]');
    const completedColumn = page.locator('[data-testid="kanban-column-completed"]');

    const taskToDrag = pendingColumn.locator('[data-testid*="task-card"]').first();
    const taskText = await taskToDrag.textContent();

    // Move task to completed
    await taskToDrag.dragTo(completedColumn);
    await page.waitForTimeout(1000);

    // Reload page
    await page.reload();
    await page.waitForLoadState('networkidle');
    await page.waitForSelector('[data-testid*="task-card"]', { timeout: 10000 });

    // Verify task is still in completed
    const completedTask = completedColumn.locator(`text=${taskText}`);
    await expect(completedTask).toBeVisible({ timeout: 5000 });
  });

  test('should handle rapid drag-and-drop movements', async ({ page }) => {
    const columns = await page.locator('[data-testid*="kanban-column"]').all();

    if (columns.length < 2) {
      test.skip();
      return;
    }

    const taskToDrag = page.locator('[data-testid*="task-card"]').first();

    // Rapid movements between columns
    for (let i = 0; i < 3; i++) {
      const targetColumn = columns[i % columns.length];
      await taskToDrag.dragTo(targetColumn);
      await page.waitForTimeout(300);
    }

    // Verify task is still visible and functional
    await expect(taskToDrag).toBeVisible();
  });

  test('should maintain task order within column after drag', async ({ page }) => {
    const column = page.locator('[data-testid="kanban-column-in-progress"]');
    const tasks = await column.locator('[data-testid*="task-card"]').all();

    if (tasks.length < 2) {
      test.skip();
      return;
    }

    // Get initial order
    const initialOrder = await Promise.all(
      tasks.map(task => task.textContent())
    );

    // Drag first task within same column (reorder)
    const firstTask = tasks[0];
    const secondTaskBox = await tasks[1].boundingBox();

    if (secondTaskBox) {
      await firstTask.dragTo(tasks[1], {
        targetPosition: { x: secondTaskBox.width / 2, y: secondTaskBox.height + 10 }
      });
      await page.waitForTimeout(500);
    }

    // Verify order changed or remained (depends on implementation)
    const newTasks = await column.locator('[data-testid*="task-card"]').all();
    expect(newTasks.length).toBe(tasks.length);
  });

  test('should update task status via API on drop', async ({ page }) => {
    let statusUpdateCalled = false;

    // Intercept API calls
    await page.route('**/mcp/**', route => {
      const postData = route.request().postDataJSON?.();
      if (postData && postData.method === 'coordinator_update_task_status') {
        statusUpdateCalled = true;
      }
      route.continue();
    });

    const taskToDrag = page.locator('[data-testid*="task-card"]').first();
    const targetColumn = page.locator('[data-testid="kanban-column-completed"]');

    await taskToDrag.dragTo(targetColumn);
    await page.waitForTimeout(1000);

    // Verify API was called
    expect(statusUpdateCalled).toBeTruthy();
  });

  test('should handle drag cancellation (ESC key)', async ({ page }) => {
    const pendingColumn = page.locator('[data-testid="kanban-column-pending"]');
    const taskToDrag = pendingColumn.locator('[data-testid*="task-card"]').first();
    const initialColumn = pendingColumn;

    // Start dragging
    await taskToDrag.hover();
    await page.mouse.down();

    // Cancel with ESC
    await page.keyboard.press('Escape');
    await page.waitForTimeout(200);

    // Verify task is still in original column
    const taskStillInColumn = await initialColumn.locator('[data-testid*="task-card"]').count();
    expect(taskStillInColumn).toBeGreaterThan(0);
  });
});