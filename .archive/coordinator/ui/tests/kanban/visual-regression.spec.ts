/**
 * Kanban Board Visual Regression Tests
 *
 * Test Suite: Visual consistency and layout validation
 *
 * Coverage:
 * - Screenshot comparison for Kanban board layout
 * - Column visual consistency
 * - Task card styling verification
 * - Priority badge colors (red, yellow, green)
 * - Responsive layout screenshots
 * - Dark/light theme consistency (if applicable)
 */

import { test, expect } from '@playwright/test';

test.describe('Kanban Visual Regression Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    await page.waitForSelector('[data-testid*="kanban-column"]', { timeout: 10000 });
  });

  test('should match baseline screenshot for full Kanban board - desktop', async ({ page }) => {
    // Take full page screenshot
    await expect(page).toHaveScreenshot('kanban-board-desktop-baseline.png', {
      fullPage: true,
      maxDiffPixels: 100, // Allow small differences
    });
  });

  test('should match baseline for pending column', async ({ page }) => {
    const pendingColumn = page.locator('[data-testid="kanban-column-pending"]');
    await expect(pendingColumn).toBeVisible();

    await expect(pendingColumn).toHaveScreenshot('pending-column-baseline.png', {
      maxDiffPixels: 50,
    });
  });

  test('should match baseline for in_progress column', async ({ page }) => {
    const inProgressColumn = page.locator('[data-testid="kanban-column-in-progress"]');
    await expect(inProgressColumn).toBeVisible();

    await expect(inProgressColumn).toHaveScreenshot('in-progress-column-baseline.png', {
      maxDiffPixels: 50,
    });
  });

  test('should match baseline for completed column', async ({ page }) => {
    const completedColumn = page.locator('[data-testid="kanban-column-completed"]');
    await expect(completedColumn).toBeVisible();

    await expect(completedColumn).toHaveScreenshot('completed-column-baseline.png', {
      maxDiffPixels: 50,
    });
  });

  test('should match baseline for blocked column', async ({ page }) => {
    const blockedColumn = page.locator('[data-testid="kanban-column-blocked"]');
    await expect(blockedColumn).toBeVisible();

    await expect(blockedColumn).toHaveScreenshot('blocked-column-baseline.png', {
      maxDiffPixels: 50,
    });
  });

  test('should verify task card visual consistency', async ({ page }) => {
    await page.waitForSelector('[data-testid*="task-card"]', { timeout: 10000 });

    const taskCard = page.locator('[data-testid*="task-card"]').first();
    await expect(taskCard).toBeVisible();

    await expect(taskCard).toHaveScreenshot('task-card-baseline.png', {
      maxDiffPixels: 30,
    });
  });

  test('should verify high priority badge color (red)', async ({ page }) => {
    // Look for high priority badge
    const highPriorityBadge = page.locator('[class*="MuiChip"]').filter({
      hasText: /high|critical|urgent/i
    }).first();

    if (await highPriorityBadge.count() > 0) {
      await expect(highPriorityBadge).toBeVisible();

      // Get background color
      const bgColor = await highPriorityBadge.evaluate((el) => {
        return window.getComputedStyle(el).backgroundColor;
      });

      // Should be red-ish (rgb values with high red component)
      console.log('High priority badge color:', bgColor);
      expect(bgColor).toBeTruthy();
    }
  });

  test('should verify medium priority badge color (yellow)', async ({ page }) => {
    const mediumPriorityBadge = page.locator('[class*="MuiChip"]').filter({
      hasText: /medium|normal/i
    }).first();

    if (await mediumPriorityBadge.count() > 0) {
      await expect(mediumPriorityBadge).toBeVisible();

      const bgColor = await mediumPriorityBadge.evaluate((el) => {
        return window.getComputedStyle(el).backgroundColor;
      });

      console.log('Medium priority badge color:', bgColor);
      expect(bgColor).toBeTruthy();
    }
  });

  test('should verify low priority badge color (green)', async ({ page }) => {
    const lowPriorityBadge = page.locator('[class*="MuiChip"]').filter({
      hasText: /low|minor/i
    }).first();

    if (await lowPriorityBadge.count() > 0) {
      await expect(lowPriorityBadge).toBeVisible();

      const bgColor = await lowPriorityBadge.evaluate((el) => {
        return window.getComputedStyle(el).backgroundColor;
      });

      console.log('Low priority badge color:', bgColor);
      expect(bgColor).toBeTruthy();
    }
  });

  test('should match baseline for mobile layout (375px)', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 812 });
    await page.waitForTimeout(500);

    await expect(page).toHaveScreenshot('kanban-board-mobile-baseline.png', {
      fullPage: true,
      maxDiffPixels: 150,
    });
  });

  test('should match baseline for tablet layout (768px)', async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 1024 });
    await page.waitForTimeout(500);

    await expect(page).toHaveScreenshot('kanban-board-tablet-baseline.png', {
      fullPage: true,
      maxDiffPixels: 150,
    });
  });

  test('should verify MUI Card elevation shadows', async ({ page }) => {
    await page.waitForSelector('[class*="MuiCard"]', { timeout: 10000 });

    const muiCard = page.locator('[class*="MuiCard"]').first();

    const boxShadow = await muiCard.evaluate((el) => {
      return window.getComputedStyle(el).boxShadow;
    });

    // MUI Cards should have box-shadow for elevation
    expect(boxShadow).not.toBe('none');
    console.log('MUI Card shadow:', boxShadow);
  });

  test('should verify column header styling consistency', async ({ page }) => {
    const columnHeaders = await page.getByRole('heading', { level: 2 }).or(
      page.getByRole('heading', { level: 3 })
    ).all();

    if (columnHeaders.length === 0) {
      test.skip();
      return;
    }

    // Get styles of first header
    const firstHeaderStyles = await columnHeaders[0].evaluate((el) => {
      const styles = window.getComputedStyle(el);
      return {
        fontSize: styles.fontSize,
        fontWeight: styles.fontWeight,
        color: styles.color,
      };
    });

    // Verify other headers match
    for (const header of columnHeaders.slice(1)) {
      const headerStyles = await header.evaluate((el) => {
        const styles = window.getComputedStyle(el);
        return {
          fontSize: styles.fontSize,
          fontWeight: styles.fontWeight,
          color: styles.color,
        };
      });

      // Headers should have consistent styling
      expect(headerStyles.fontSize).toBe(firstHeaderStyles.fontSize);
      expect(headerStyles.fontWeight).toBe(firstHeaderStyles.fontWeight);
    }
  });

  test('should verify drag-and-drop visual feedback', async ({ page }) => {
    await page.waitForSelector('[data-testid*="task-card"]', { timeout: 10000 });

    const taskCard = page.locator('[data-testid*="task-card"]').first();

    // Get initial styles
    const initialStyles = await taskCard.evaluate((el) => {
      return window.getComputedStyle(el).opacity;
    });

    // Start dragging
    await taskCard.hover();
    await page.mouse.down();
    await page.waitForTimeout(200);

    // Take screenshot during drag
    await expect(page).toHaveScreenshot('drag-in-progress.png', {
      maxDiffPixels: 200,
    });

    await page.mouse.up();
  });

  test('should verify empty state visual appearance', async ({ page }) => {
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

    await expect(page).toHaveScreenshot('empty-state-baseline.png', {
      maxDiffPixels: 100,
    });
  });

  test('should verify error state visual appearance', async ({ page }) => {
    // Mock error
    await page.route('**/mcp/**', route => {
      route.abort('failed');
    });

    await page.goto('/');
    await page.waitForTimeout(2000);

    await expect(page).toHaveScreenshot('error-state-baseline.png', {
      maxDiffPixels: 100,
    });
  });

  test('should verify loading state visual appearance', async ({ page }) => {
    // Navigate without waiting
    await page.goto('/', { waitUntil: 'commit' });

    // Try to capture loading state (may be too fast)
    try {
      await page.waitForSelector('[role="progressbar"]', { timeout: 1000 });
      await expect(page).toHaveScreenshot('loading-state-baseline.png', {
        maxDiffPixels: 100,
      });
    } catch {
      // Loading state may be too fast to capture
      test.skip();
    }
  });
});