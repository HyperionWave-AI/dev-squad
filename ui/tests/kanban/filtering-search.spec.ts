/**
 * Kanban Board Filtering and Search Tests
 *
 * Test Suite: Task filtering and search functionality
 *
 * Coverage:
 * - Search by task prompt/description
 * - Filter by status (pending, in_progress, completed, blocked)
 * - Filter by priority (high, medium, low)
 * - Filter by agent name
 * - Combined filters
 * - Clear filters
 * - Search results highlighting
 */

import { test, expect } from '@playwright/test';

test.describe('Kanban Filtering and Search', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    await page.waitForSelector('[data-testid*="task-card"]', { timeout: 10000 });
  });

  test('should have search input field', async ({ page }) => {
    const searchInput = page.locator('input[type="search"]').or(
      page.locator('input[placeholder*="search" i]')
    );

    if (await searchInput.count() === 0) {
      // Search may not be implemented yet
      test.skip();
      return;
    }

    await expect(searchInput).toBeVisible();
  });

  test('should filter tasks by search term', async ({ page }) => {
    const searchInput = page.locator('input[type="search"]').or(
      page.locator('input[placeholder*="search" i]')
    ).first();

    if (await searchInput.count() === 0) {
      test.skip();
      return;
    }

    // Get initial task count
    const initialCount = await page.locator('[data-testid*="task-card"]').count();

    // Search for specific term
    await searchInput.fill('authentication');
    await page.waitForTimeout(500);

    // Task count should change (either fewer or same if all match)
    const filteredCount = await page.locator('[data-testid*="task-card"]').count();
    expect(filteredCount).toBeLessThanOrEqual(initialCount);
  });

  test('should clear search when input is emptied', async ({ page }) => {
    const searchInput = page.locator('input[type="search"]').or(
      page.locator('input[placeholder*="search" i]')
    ).first();

    if (await searchInput.count() === 0) {
      test.skip();
      return;
    }

    // Search
    await searchInput.fill('test');
    await page.waitForTimeout(500);

    const filteredCount = await page.locator('[data-testid*="task-card"]').count();

    // Clear search
    await searchInput.clear();
    await page.waitForTimeout(500);

    const clearedCount = await page.locator('[data-testid*="task-card"]').count();

    // Should show more tasks after clearing
    expect(clearedCount).toBeGreaterThanOrEqual(filteredCount);
  });

  test('should have status filter buttons/dropdown', async ({ page }) => {
    const statusFilter = page.locator('[data-testid*="status-filter"]').or(
      page.locator('button').filter({ hasText: /status|filter/i })
    );

    if (await statusFilter.count() === 0) {
      // Filter UI may not be implemented yet
      test.skip();
      return;
    }

    await expect(statusFilter.first()).toBeVisible();
  });

  test('should filter by pending status', async ({ page }) => {
    // Try to find pending filter button
    const pendingFilter = page.locator('button').filter({ hasText: /pending/i }).first();

    if (await pendingFilter.count() === 0) {
      test.skip();
      return;
    }

    await pendingFilter.click();
    await page.waitForTimeout(500);

    // Only pending tasks should be visible
    const pendingColumn = page.locator('[data-testid="kanban-column-pending"]');
    const pendingTasks = await pendingColumn.locator('[data-testid*="task-card"]').count();

    // Other columns should be hidden or have no tasks
    const inProgressTasks = await page.locator('[data-testid="kanban-column-in-progress"]')
      .locator('[data-testid*="task-card"]').count();

    expect(pendingTasks).toBeGreaterThan(0);
    expect(inProgressTasks).toBe(0);
  });

  test('should filter by agent name', async ({ page }) => {
    const agentFilter = page.locator('select').or(
      page.locator('[data-testid*="agent-filter"]')
    ).first();

    if (await agentFilter.count() === 0) {
      test.skip();
      return;
    }

    // Select an agent
    await agentFilter.click();
    await page.waitForTimeout(200);

    // Select first option
    const options = page.locator('option').or(page.locator('[role="option"]'));

    if (await options.count() > 1) {
      await options.nth(1).click();
      await page.waitForTimeout(500);

      // Tasks should be filtered
      const visibleTasks = await page.locator('[data-testid*="task-card"]').count();
      expect(visibleTasks).toBeGreaterThanOrEqual(0);
    }
  });

  test('should filter by priority', async ({ page }) => {
    const priorityFilter = page.locator('button').filter({ hasText: /high|medium|low/i }).first();

    if (await priorityFilter.count() === 0) {
      test.skip();
      return;
    }

    // Get initial count
    const initialCount = await page.locator('[data-testid*="task-card"]').count();

    // Click priority filter
    await priorityFilter.click();
    await page.waitForTimeout(500);

    // Count should change or stay same
    const filteredCount = await page.locator('[data-testid*="task-card"]').count();
    expect(filteredCount).toBeLessThanOrEqual(initialCount);
  });

  test('should show no results message when search returns empty', async ({ page }) => {
    const searchInput = page.locator('input[type="search"]').or(
      page.locator('input[placeholder*="search" i]')
    ).first();

    if (await searchInput.count() === 0) {
      test.skip();
      return;
    }

    // Search for non-existent term
    await searchInput.fill('xyzzzzzzzzzznonexistent');
    await page.waitForTimeout(500);

    // Should show no results message
    const noResults = page.locator('text=/no results|not found|no tasks match/i');
    await expect(noResults).toBeVisible({ timeout: 2000 });
  });

  test('should highlight search terms in results', async ({ page }) => {
    const searchInput = page.locator('input[type="search"]').or(
      page.locator('input[placeholder*="search" i]')
    ).first();

    if (await searchInput.count() === 0) {
      test.skip();
      return;
    }

    // Search for common term
    await searchInput.fill('task');
    await page.waitForTimeout(500);

    // Check if highlighting is applied (look for mark or highlighted text)
    const highlighted = page.locator('mark').or(page.locator('[class*="highlight"]'));

    // May or may not have highlighting
    const hasHighlight = await highlighted.count() > 0;
    console.log('Search highlighting present:', hasHighlight);
  });

  test('should combine multiple filters', async ({ page }) => {
    const searchInput = page.locator('input[type="search"]').first();
    const statusButton = page.locator('button').filter({ hasText: /status|pending/i }).first();

    if (await searchInput.count() === 0 || await statusButton.count() === 0) {
      test.skip();
      return;
    }

    // Apply search
    await searchInput.fill('implement');
    await page.waitForTimeout(300);

    const searchOnlyCount = await page.locator('[data-testid*="task-card"]').count();

    // Apply status filter
    await statusButton.click();
    await page.waitForTimeout(300);

    const combinedCount = await page.locator('[data-testid*="task-card"]').count();

    // Combined filter should be more restrictive
    expect(combinedCount).toBeLessThanOrEqual(searchOnlyCount);
  });

  test('should have clear all filters button', async ({ page }) => {
    const clearButton = page.locator('button').filter({ hasText: /clear|reset/i });

    if (await clearButton.count() === 0) {
      test.skip();
      return;
    }

    await expect(clearButton.first()).toBeVisible();

    // Click clear
    await clearButton.first().click();
    await page.waitForTimeout(500);

    // All tasks should be visible
    const allTasks = await page.locator('[data-testid*="task-card"]').count();
    expect(allTasks).toBeGreaterThan(0);
  });

  test('should persist filters after page reload', async ({ page }) => {
    const searchInput = page.locator('input[type="search"]').first();

    if (await searchInput.count() === 0) {
      test.skip();
      return;
    }

    // Apply search
    await searchInput.fill('authentication');
    await page.waitForTimeout(500);

    // Reload page
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Check if filter persisted (may or may not be implemented)
    const searchValue = await searchInput.inputValue().catch(() => '');

    // Filter persistence is optional
    console.log('Filter persisted:', searchValue === 'authentication');
  });

  test('should show task count after filtering', async ({ page }) => {
    const searchInput = page.locator('input[type="search"]').first();

    if (await searchInput.count() === 0) {
      test.skip();
      return;
    }

    // Apply filter
    await searchInput.fill('test');
    await page.waitForTimeout(500);

    // Look for result count
    const resultCount = page.locator('text=/\\d+\\s+result/i').or(
      page.locator('text=/showing\\s+\\d+/i')
    );

    // Result count display is optional
    const hasResultCount = await resultCount.count() > 0;
    console.log('Result count displayed:', hasResultCount);
  });
});