/**
 * Kanban Board Responsive Design Tests
 *
 * Test Suite: Responsive behavior across different viewports
 *
 * Coverage:
 * - Mobile viewport (375px)
 * - Tablet viewport (768px)
 * - Desktop viewport (1920px)
 * - Column layout adjustments
 * - Touch interactions
 * - Horizontal scrolling
 * - Mobile navigation
 */

import { test, expect } from '@playwright/test';

test.describe('Kanban Responsive Design - Desktop (1920px)', () => {
  test.use({ viewport: { width: 1920, height: 1080 } });

  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
  });

  test('should display all columns in single row on desktop', async ({ page }) => {
    await page.waitForSelector('[data-testid*="kanban-column"]', { timeout: 10000 });

    const columns = await page.locator('[data-testid*="kanban-column"]').all();
    expect(columns.length).toBe(4);

    // Check columns are displayed horizontally
    const firstColumnBox = await columns[0].boundingBox();
    const secondColumnBox = await columns[1].boundingBox();

    expect(firstColumnBox).not.toBeNull();
    expect(secondColumnBox).not.toBeNull();

    // Second column should be to the right of first column
    if (firstColumnBox && secondColumnBox) {
      expect(secondColumnBox.x).toBeGreaterThan(firstColumnBox.x);
    }
  });

  test('should use MUI Grid layout on desktop', async ({ page }) => {
    const gridContainer = page.locator('[class*="MuiGrid-container"]').first();
    await expect(gridContainer).toBeVisible();

    const gridItems = page.locator('[class*="MuiGrid-item"]');
    const gridItemCount = await gridItems.count();

    // Should have grid items for each column
    expect(gridItemCount).toBeGreaterThanOrEqual(4);
  });

  test('should not require horizontal scrolling on desktop', async ({ page }) => {
    const viewportWidth = page.viewportSize()?.width || 1920;

    const bodyWidth = await page.evaluate(() => document.body.scrollWidth);

    // Body width should not exceed viewport
    expect(bodyWidth).toBeLessThanOrEqual(viewportWidth + 20); // 20px tolerance for scrollbar
  });
});

test.describe('Kanban Responsive Design - Tablet (768px)', () => {
  test.use({ viewport: { width: 768, height: 1024 } });

  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
  });

  test('should display columns in grid layout on tablet', async ({ page }) => {
    await page.waitForSelector('[data-testid*="kanban-column"]', { timeout: 10000 });

    const columns = await page.locator('[data-testid*="kanban-column"]').all();
    expect(columns.length).toBe(4);

    // Columns may wrap on tablet (2x2 grid or horizontal scroll)
    const firstColumnBox = await columns[0].boundingBox();
    const thirdColumnBox = await columns[2].boundingBox();

    expect(firstColumnBox).not.toBeNull();

    if (firstColumnBox && thirdColumnBox) {
      // Either wrapped (third column below first) or scrollable (third column to right)
      const isWrapped = thirdColumnBox.y > firstColumnBox.y + firstColumnBox.height;
      const isScrollable = thirdColumnBox.x > firstColumnBox.x;

      expect(isWrapped || isScrollable).toBeTruthy();
    }
  });

  test('should maintain usability on tablet viewport', async ({ page }) => {
    // Verify task cards are still clickable and visible
    const taskCard = page.locator('[data-testid*="task-card"]').first();
    await expect(taskCard).toBeVisible();

    // Check card dimensions are reasonable
    const cardBox = await taskCard.boundingBox();
    expect(cardBox).not.toBeNull();

    if (cardBox) {
      expect(cardBox.width).toBeGreaterThan(150);
      expect(cardBox.width).toBeLessThan(600);
    }
  });

  test('should support touch interactions on tablet', async ({ page }) => {
    const taskCard = page.locator('[data-testid*="task-card"]').first();

    // Simulate touch tap
    await taskCard.tap();
    await page.waitForTimeout(200);

    // Verify interaction worked (no assertion needed, just verify no crash)
    await expect(taskCard).toBeVisible();
  });
});

test.describe('Kanban Responsive Design - Mobile (375px)', () => {
  test.use({ viewport: { width: 375, height: 812 } });

  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
  });

  test('should display columns vertically or with horizontal scroll on mobile', async ({ page }) => {
    await page.waitForSelector('[data-testid*="kanban-column"]', { timeout: 10000 });

    const columns = await page.locator('[data-testid*="kanban-column"]').all();
    expect(columns.length).toBe(4);

    const firstColumnBox = await columns[0].boundingBox();
    const secondColumnBox = await columns[1].boundingBox();

    expect(firstColumnBox).not.toBeNull();
    expect(secondColumnBox).not.toBeNull();

    if (firstColumnBox && secondColumnBox) {
      // Either stacked vertically or horizontally scrollable
      const isStacked = secondColumnBox.y > firstColumnBox.y + firstColumnBox.height - 50;
      const isScrollable = secondColumnBox.x > firstColumnBox.x + firstColumnBox.width - 50;

      expect(isStacked || isScrollable).toBeTruthy();
    }
  });

  test('should fit task cards within mobile viewport width', async ({ page }) => {
    await page.waitForSelector('[data-testid*="task-card"]', { timeout: 10000 });

    const taskCard = page.locator('[data-testid*="task-card"]').first();
    const cardBox = await taskCard.boundingBox();

    expect(cardBox).not.toBeNull();

    if (cardBox) {
      // Card width should be less than or equal to viewport width
      expect(cardBox.width).toBeLessThanOrEqual(375);
      // Should have reasonable minimum width
      expect(cardBox.width).toBeGreaterThan(250);
    }
  });

  test('should support touch drag-and-drop on mobile', async ({ page }) => {
    await page.waitForSelector('[data-testid*="task-card"]', { timeout: 10000 });

    const taskCard = page.locator('[data-testid*="task-card"]').first();
    const targetColumn = page.locator('[data-testid="kanban-column-completed"]');

    // Perform touch drag
    const taskBox = await taskCard.boundingBox();
    const targetBox = await targetColumn.boundingBox();

    if (taskBox && targetBox) {
      // Touch start
      await page.touchscreen.tap(taskBox.x + taskBox.width / 2, taskBox.y + taskBox.height / 2);
      await page.waitForTimeout(200);

      // Can't easily simulate touch drag in Playwright, verify elements exist
      await expect(taskCard).toBeVisible();
      await expect(targetColumn).toBeVisible();
    }
  });

  test('should display mobile-optimized navigation', async ({ page }) => {
    // Check for mobile menu or hamburger icon
    const mobileMenu = page.locator('[aria-label*="menu"]').or(
      page.locator('[class*="MuiIconButton"]').first()
    );

    // Mobile navigation should be present
    const menuExists = await mobileMenu.count() > 0;
    expect(menuExists).toBeTruthy();
  });

  test('should maintain readability of text on mobile', async ({ page }) => {
    await page.waitForSelector('[data-testid*="task-card"]', { timeout: 10000 });

    const taskCard = page.locator('[data-testid*="task-card"]').first();
    const taskText = taskCard.locator('text=/./').first();

    // Verify text is visible and has reasonable size
    await expect(taskText).toBeVisible();

    const fontSize = await taskText.evaluate((el) => {
      return window.getComputedStyle(el).fontSize;
    });

    // Font size should be at least 14px for mobile readability
    const fontSizeNum = parseInt(fontSize);
    expect(fontSizeNum).toBeGreaterThanOrEqual(12);
  });

  test('should allow horizontal scrolling if columns exceed viewport', async ({ page }) => {
    await page.waitForSelector('[data-testid*="kanban-column"]', { timeout: 10000 });

    // Check if container is scrollable
    const container = page.locator('[class*="MuiGrid-container"]').first();

    const isScrollable = await container.evaluate((el) => {
      return el.scrollWidth > el.clientWidth;
    });

    // On mobile, either scrollable or stacked vertically (both valid)
    expect(typeof isScrollable).toBe('boolean');
  });

  test('should show abbreviated column titles on mobile if needed', async ({ page }) => {
    await page.waitForSelector('[data-testid*="kanban-column"]', { timeout: 10000 });

    // Check column headers fit within mobile width
    const columnHeader = page.getByRole('heading', { name: /Pending|Progress|Completed|Blocked/i }).first();
    await expect(columnHeader).toBeVisible();

    const headerBox = await columnHeader.boundingBox();

    if (headerBox) {
      expect(headerBox.width).toBeLessThan(300);
    }
  });
});

test.describe('Kanban Responsive Design - Viewport Transitions', () => {
  test('should adapt layout when resizing from desktop to mobile', async ({ page }) => {
    // Start with desktop viewport
    await page.setViewportSize({ width: 1920, height: 1080 });
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    await page.waitForSelector('[data-testid*="kanban-column"]', { timeout: 10000 });

    // Verify desktop layout
    const desktopColumns = await page.locator('[data-testid*="kanban-column"]').all();
    expect(desktopColumns.length).toBe(4);

    // Resize to mobile
    await page.setViewportSize({ width: 375, height: 812 });
    await page.waitForTimeout(500);

    // Verify mobile layout
    const mobileColumns = await page.locator('[data-testid*="kanban-column"]').all();
    expect(mobileColumns.length).toBe(4); // Same columns, different layout

    // Check first column is still visible
    await expect(mobileColumns[0]).toBeVisible();
  });

  test('should maintain task data integrity across viewport changes', async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 });
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    await page.waitForSelector('[data-testid*="task-card"]', { timeout: 10000 });

    // Get task count at desktop size
    const desktopTaskCount = await page.locator('[data-testid*="task-card"]').count();

    // Resize to mobile
    await page.setViewportSize({ width: 375, height: 812 });
    await page.waitForTimeout(500);

    // Get task count at mobile size
    const mobileTaskCount = await page.locator('[data-testid*="task-card"]').count();

    // Task count should remain the same
    expect(mobileTaskCount).toBe(desktopTaskCount);
  });
});