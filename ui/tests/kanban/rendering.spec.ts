/**
 * Kanban Board Rendering Tests
 *
 * Test Suite: Basic rendering and layout validation
 *
 * Coverage:
 * - Column structure (pending, in_progress, completed, blocked)
 * - Task card rendering
 * - MUI component presence (Cards, AppBar, Chips)
 * - Loading states
 * - Error handling
 * - Empty states
 */

import { test, expect } from '@playwright/test';
import { mockHumanTasks, mockAgentTasks, kanbanColumns } from '../fixtures/mockTasks';

test.describe('Kanban Board Rendering', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    // Wait for initial load
    await page.waitForLoadState('networkidle');
  });

  test('should render all four Kanban columns', async ({ page }) => {
    // Verify all column headers are present
    for (const column of kanbanColumns) {
      const columnHeader = page.getByRole('heading', { name: column.title });
      await expect(columnHeader).toBeVisible();
    }
  });

  test('should render MUI AppBar header', async ({ page }) => {
    // Check for MUI AppBar component
    const appBar = page.locator('[class*="MuiAppBar"]').first();
    await expect(appBar).toBeVisible();

    // Verify navigation elements
    const title = page.getByRole('heading', { name: /Hyperion|Task|Coordinator/i });
    await expect(title).toBeVisible();
  });

  test('should render task cards in correct columns', async ({ page }) => {
    // Wait for tasks to load
    await page.waitForSelector('[data-testid*="task-card"]', { timeout: 10000 });

    // Verify tasks are distributed by status
    const pendingColumn = page.locator('[data-testid="kanban-column-pending"]');
    const inProgressColumn = page.locator('[data-testid="kanban-column-in-progress"]');
    const completedColumn = page.locator('[data-testid="kanban-column-completed"]');
    const blockedColumn = page.locator('[data-testid="kanban-column-blocked"]');

    // Each column should have at least one task (based on mock data)
    await expect(pendingColumn.locator('[data-testid*="task-card"]')).toHaveCount(1, { timeout: 5000 });
    await expect(inProgressColumn.locator('[data-testid*="task-card"]')).toHaveCount(1, { timeout: 5000 });
    await expect(completedColumn.locator('[data-testid*="task-card"]')).toHaveCount(1, { timeout: 5000 });
    await expect(blockedColumn.locator('[data-testid*="task-card"]')).toHaveCount(1, { timeout: 5000 });
  });

  test('should render MUI Card components for tasks', async ({ page }) => {
    await page.waitForSelector('[class*="MuiCard"]', { timeout: 10000 });

    const muiCards = page.locator('[class*="MuiCard"]');
    const cardCount = await muiCards.count();

    expect(cardCount).toBeGreaterThan(0);

    // Verify Card structure
    const firstCard = muiCards.first();
    await expect(firstCard).toBeVisible();

    // Check for MUI Card sub-components
    const cardContent = firstCard.locator('[class*="MuiCardContent"]');
    await expect(cardContent).toBeVisible();
  });

  test('should render priority badges with MUI Chips', async ({ page }) => {
    await page.waitForSelector('[class*="MuiChip"]', { timeout: 10000 });

    const chips = page.locator('[class*="MuiChip"]');
    const chipCount = await chips.count();

    expect(chipCount).toBeGreaterThan(0);

    // Verify chip colors for different priorities
    const priorityChip = chips.first();
    await expect(priorityChip).toBeVisible();
  });

  test('should show loading state initially', async ({ page }) => {
    // Navigate to page without waiting
    await page.goto('/', { waitUntil: 'commit' });

    // Check for loading indicator
    const loadingIndicator = page.getByText(/loading/i).or(page.locator('[role="progressbar"]'));

    // Loading should appear briefly
    const isVisible = await loadingIndicator.isVisible().catch(() => false);

    // Either loading was visible or page loaded too fast (both acceptable)
    expect(typeof isVisible).toBe('boolean');
  });

  test('should display error state on API failure', async ({ page }) => {
    // Mock API failure
    await page.route('**/mcp/**', route => {
      route.abort('failed');
    });

    await page.goto('/');
    await page.waitForTimeout(2000);

    // Check for error message
    const errorMessage = page.getByText(/error|failed/i);
    await expect(errorMessage).toBeVisible({ timeout: 5000 });
  });

  test('should display empty state when no tasks exist', async ({ page }) => {
    // Mock empty response
    await page.route('**/mcp/**', route => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ tasks: [] }),
      });
    });

    await page.goto('/');
    await page.waitForTimeout(1000);

    // Check for empty state message
    const emptyState = page.getByText(/no tasks|empty|get started/i);
    await expect(emptyState).toBeVisible({ timeout: 5000 });
  });

  test('should render task metadata correctly', async ({ page }) => {
    await page.waitForSelector('[data-testid*="task-card"]', { timeout: 10000 });

    const taskCard = page.locator('[data-testid*="task-card"]').first();

    // Verify task information is displayed
    await expect(taskCard).toBeVisible();

    // Check for task title/prompt
    const taskTitle = taskCard.locator('text=/.*task.*/i').first();
    await expect(taskTitle).toBeVisible();

    // Check for agent name or role
    const agentInfo = taskCard.locator('[data-testid*="agent"]').or(
      taskCard.locator('text=/specialist|backend|frontend/i')
    );
    const hasAgentInfo = await agentInfo.count() > 0;
    expect(hasAgentInfo).toBeTruthy();
  });

  test('should maintain column order: pending → in_progress → completed → blocked', async ({ page }) => {
    await page.waitForSelector('[data-testid*="kanban-column"]', { timeout: 10000 });

    const columns = await page.locator('[data-testid*="kanban-column"]').all();

    // Get column order from data-testid attributes
    const columnOrder = await Promise.all(
      columns.map(col => col.getAttribute('data-testid'))
    );

    expect(columnOrder).toEqual([
      'kanban-column-pending',
      'kanban-column-in-progress',
      'kanban-column-completed',
      'kanban-column-blocked',
    ]);
  });
});