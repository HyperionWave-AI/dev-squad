import { test, expect } from '@playwright/test';
import { injectAxe, checkA11y } from 'axe-playwright';

test.describe('Tasks Page Accessibility Testing', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await injectAxe(page);
  });

  test('should meet WCAG 2.1 AA standards on tasks page', async ({ page }) => {
    // Navigate to tasks/kanban page
    await page.click('[data-testid="tasks-nav"]');
    await page.waitForLoadState('networkidle');
    
    // Run accessibility audit
    await checkA11y(page, null, {
      detailedReport: true,
      detailedReportOptions: { html: true },
    });
  });

  test('should have proper heading hierarchy', async ({ page }) => {
    await page.click('[data-testid="tasks-nav"]');
    
    // Check for proper heading structure
    const headings = await page.locator('h1, h2, h3, h4, h5, h6').all();
    expect(headings.length).toBeGreaterThan(0);
    
    // Verify main page heading exists
    await expect(page.locator('h1')).toBeVisible();
  });

  test('should support keyboard navigation', async ({ page }) => {
    await page.click('[data-testid="tasks-nav"]');
    await page.waitForLoadState('networkidle');
    
    // Test tab navigation through interactive elements
    await page.keyboard.press('Tab');
    let focusedElement = await page.locator(':focus');
    await expect(focusedElement).toBeVisible();
    
    // Continue tabbing through elements
    for (let i = 0; i < 10; i++) {
      await page.keyboard.press('Tab');
      focusedElement = await page.locator(':focus');
      await expect(focusedElement).toBeVisible();
    }
  });

  test('should have proper ARIA labels on task cards', async ({ page }) => {
    await page.click('[data-testid="tasks-nav"]');
    await page.waitForLoadState('networkidle');
    
    // Check task cards have proper ARIA labels
    const taskCards = await page.locator('[data-testid="task-card"]').all();
    
    for (const card of taskCards) {
      // Each task card should have accessible name
      const ariaLabel = await card.getAttribute('aria-label');
      const ariaLabelledBy = await card.getAttribute('aria-labelledby');
      
      expect(ariaLabel || ariaLabelledBy).toBeTruthy();
    }
  });

  test('should have proper color contrast ratios', async ({ page }) => {
    await page.click('[data-testid="tasks-nav"]');
    
    // Run color contrast specific checks
    await checkA11y(page, null, {
      rules: {
        'color-contrast': { enabled: true }
      }
    });
  });

  test('should support screen reader navigation', async ({ page }) => {
    await page.click('[data-testid="tasks-nav"]');
    await page.waitForLoadState('networkidle');
    
    // Check for proper landmarks
    await expect(page.locator('[role="main"]')).toBeVisible();
    await expect(page.locator('[role="navigation"]')).toBeVisible();
    
    // Check for proper list structure in columns
    const columns = await page.locator('[data-testid="kanban-column"]').all();
    
    for (const column of columns) {
      // Each column should have a heading
      const heading = column.locator('h2, h3, h4');
      await expect(heading).toBeVisible();
      
      // Task lists should have proper role
      const taskList = column.locator('[role="list"]');
      if (await taskList.count() > 0) {
        await expect(taskList).toBeVisible();
      }
    }
  });

  test('should handle focus management during drag operations', async ({ page }) => {
    await page.click('[data-testid="tasks-nav"]');
    await page.waitForLoadState('networkidle');
    
    // Find a draggable task
    const firstTask = page.locator('[data-testid="task-card"]').first();
    await expect(firstTask).toBeVisible();
    
    // Focus on the task
    await firstTask.focus();
    await expect(firstTask).toBeFocused();
    
    // Test keyboard drag initiation (Space key)
    await page.keyboard.press('Space');
    
    // Verify focus is maintained during drag state
    const focusedElement = await page.locator(':focus');
    await expect(focusedElement).toBeVisible();
  });

  test('should provide proper error announcements', async ({ page }) => {
    await page.click('[data-testid="tasks-nav"]');
    
    // Look for ARIA live regions for error announcements
    const liveRegions = await page.locator('[aria-live]').all();
    expect(liveRegions.length).toBeGreaterThan(0);
    
    // Check for error alert accessibility
    const errorAlert = page.locator('[role="alert"]');
    if (await errorAlert.count() > 0) {
      await expect(errorAlert).toHaveAttribute('aria-live', 'assertive');
    }
  });

  test('should have accessible form controls', async ({ page }) => {
    await page.click('[data-testid="tasks-nav"]');
    
    // Check search input accessibility
    const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]');
    if (await searchInput.count() > 0) {
      await expect(searchInput).toHaveAttribute('aria-label');
      
      // Check for associated label
      const inputId = await searchInput.getAttribute('id');
      if (inputId) {
        const label = page.locator(`label[for="${inputId}"]`);
        if (await label.count() > 0) {
          await expect(label).toBeVisible();
        }
      }
    }
  });

  test('should support high contrast mode', async ({ page }) => {
    // Enable high contrast mode simulation
    await page.emulateMedia({ colorScheme: 'dark', forcedColors: 'active' });
    
    await page.click('[data-testid="tasks-nav"]');
    await page.waitForLoadState('networkidle');
    
    // Verify elements are still visible and accessible
    await expect(page.locator('[data-testid="kanban-column"]').first()).toBeVisible();
    await expect(page.locator('[data-testid="task-card"]').first()).toBeVisible();
    
    // Run accessibility check in high contrast mode
    await checkA11y(page);
  });

  test('should handle reduced motion preferences', async ({ page }) => {
    // Simulate reduced motion preference
    await page.emulateMedia({ reducedMotion: 'reduce' });
    
    await page.click('[data-testid="tasks-nav"]');
    await page.waitForLoadState('networkidle');
    
    // Verify page still functions with reduced motion
    const taskCard = page.locator('[data-testid="task-card"]').first();
    await expect(taskCard).toBeVisible();
    
    // Check that animations are disabled or reduced
    const computedStyle = await taskCard.evaluate((el) => {
      return window.getComputedStyle(el).getPropertyValue('animation-duration');
    });
    
    // Should be 0s or very short duration when reduced motion is preferred
    expect(computedStyle === '0s' || parseFloat(computedStyle) <= 0.1).toBeTruthy();
  });
});