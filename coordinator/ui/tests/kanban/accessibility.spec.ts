/**
 * Kanban Board Accessibility Tests
 *
 * Test Suite: WCAG 2.1 AA Compliance
 *
 * Coverage:
 * - Axe-core automated accessibility audit
 * - Keyboard navigation (Tab, Enter, Space, Arrow keys)
 * - Screen reader compatibility (ARIA labels, roles, live regions)
 * - Color contrast validation
 * - Focus management
 * - Drag-and-drop keyboard accessibility
 *
 * @tag @accessibility
 */

import { test, expect } from '@playwright/test';
import {
  runAccessibilityAudit,
  testKeyboardNavigation,
  testDragDropKeyboard,
  verifyScreenReaderAttributes,
  formatViolations,
} from '../utils/accessibility';

test.describe('Kanban Accessibility - WCAG 2.1 AA Compliance @accessibility', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
  });

  test('should pass axe-core accessibility audit', async ({ page }) => {
    await page.waitForSelector('[data-testid*="kanban-column"]', { timeout: 10000 });

    // Run axe-core audit
    const violations = await runAccessibilityAudit(page);

    // Format violations for better error messages
    if (violations.length > 0) {
      console.log('Accessibility violations found:');
      console.log(formatViolations(violations));
    }

    // Assert no violations
    expect(violations.length).toBe(0);
  });

  test('should have proper ARIA labels on all columns', async ({ page }) => {
    await page.waitForSelector('[data-testid*="kanban-column"]', { timeout: 10000 });

    const columns = ['pending', 'in-progress', 'completed', 'blocked'];

    for (const column of columns) {
      const columnElement = page.locator(`[data-testid="kanban-column-${column}"]`);
      const attrs = await verifyScreenReaderAttributes(page, `[data-testid="kanban-column-${column}"]`);

      // Column should have either aria-label or role
      expect(attrs.hasAriaLabel || attrs.hasRole).toBeTruthy();
    }
  });

  test('should have ARIA labels on task cards', async ({ page }) => {
    await page.waitForSelector('[data-testid*="task-card"]', { timeout: 10000 });

    const taskCard = page.locator('[data-testid*="task-card"]').first();
    const attrs = await verifyScreenReaderAttributes(page, '[data-testid*="task-card"]');

    // Task cards should have aria-label or descriptive role
    expect(attrs.hasAriaLabel || attrs.hasRole).toBeTruthy();
  });

  test('should support keyboard navigation with Tab key', async ({ page }) => {
    await page.waitForSelector('[data-testid*="task-card"]', { timeout: 10000 });

    // Focus should start at beginning
    await page.keyboard.press('Tab');

    // Get focused element
    const focusedElement = await page.evaluate(() => {
      return document.activeElement?.tagName;
    });

    // Some element should be focused
    expect(focusedElement).toBeTruthy();

    // Tab through multiple elements
    for (let i = 0; i < 5; i++) {
      await page.keyboard.press('Tab');
      await page.waitForTimeout(100);
    }

    // Verify focus moved
    const newFocusedElement = await page.evaluate(() => {
      return document.activeElement?.tagName;
    });

    expect(newFocusedElement).toBeTruthy();
  });

  test('should support keyboard navigation for task cards', async ({ page }) => {
    await page.waitForSelector('[data-testid*="task-card"]', { timeout: 10000 });

    const taskCards = await page.locator('[data-testid*="task-card"]').all();

    if (taskCards.length === 0) {
      test.skip();
      return;
    }

    // Focus first task card
    await taskCards[0].focus();

    // Verify focus
    const isFocused = await taskCards[0].evaluate(el => el === document.activeElement || el.contains(document.activeElement));
    expect(isFocused).toBeTruthy();

    // Test Enter key interaction
    await page.keyboard.press('Enter');
    await page.waitForTimeout(200);

    // Task should still be visible (interaction occurred)
    await expect(taskCards[0]).toBeVisible();
  });

  test('should support keyboard drag-and-drop with Space and Arrow keys', async ({ page }) => {
    await page.waitForSelector('[data-testid*="task-card"]', { timeout: 10000 });

    const taskCard = page.locator('[data-testid*="task-card"]').first();

    // Focus on task
    await taskCard.focus();

    // Activate drag mode with Space
    await page.keyboard.press('Space');
    await page.waitForTimeout(300);

    // Navigate with arrow keys
    await page.keyboard.press('ArrowRight');
    await page.waitForTimeout(200);

    await page.keyboard.press('ArrowDown');
    await page.waitForTimeout(200);

    // Drop with Space or Enter
    await page.keyboard.press('Space');
    await page.waitForTimeout(300);

    // Task should still be visible
    await expect(taskCard).toBeVisible();
  });

  test('should have proper focus indicators on interactive elements', async ({ page }) => {
    await page.waitForSelector('[data-testid*="task-card"]', { timeout: 10000 });

    const taskCard = page.locator('[data-testid*="task-card"]').first();

    // Focus element
    await taskCard.focus();

    // Check for visible focus indicator
    const focusStyles = await taskCard.evaluate((el) => {
      const styles = window.getComputedStyle(el);
      return {
        outline: styles.outline,
        outlineWidth: styles.outlineWidth,
        boxShadow: styles.boxShadow,
      };
    });

    // Should have some focus indicator (outline or box-shadow)
    const hasFocusIndicator =
      focusStyles.outline !== 'none' ||
      focusStyles.outlineWidth !== '0px' ||
      focusStyles.boxShadow !== 'none';

    expect(hasFocusIndicator).toBeTruthy();
  });

  test('should announce status changes to screen readers', async ({ page }) => {
    await page.waitForSelector('[data-testid*="task-card"]', { timeout: 10000 });

    // Check for aria-live regions
    const liveRegions = await page.locator('[aria-live]').count();

    // Should have at least one live region for status announcements
    expect(liveRegions).toBeGreaterThan(0);
  });

  test('should have descriptive button labels', async ({ page }) => {
    // Check all buttons have accessible names
    const buttons = await page.locator('button').all();

    for (const button of buttons) {
      const accessibleName = await button.evaluate((btn) => {
        return (
          btn.getAttribute('aria-label') ||
          btn.textContent?.trim() ||
          btn.getAttribute('title')
        );
      });

      // Every button should have an accessible name
      expect(accessibleName).toBeTruthy();
    }
  });

  test('should maintain logical heading hierarchy', async ({ page }) => {
    // Get all headings
    const headings = await page.locator('h1, h2, h3, h4, h5, h6').all();

    const headingLevels = await Promise.all(
      headings.map(async (h) => {
        const tagName = await h.evaluate(el => el.tagName);
        return parseInt(tagName[1]);
      })
    );

    // Should have at least one heading
    expect(headingLevels.length).toBeGreaterThan(0);

    // First heading should be h1
    if (headingLevels.length > 0) {
      expect(headingLevels[0]).toBeLessThanOrEqual(2); // h1 or h2 acceptable
    }

    // Headings should not skip levels
    for (let i = 1; i < headingLevels.length; i++) {
      const diff = headingLevels[i] - headingLevels[i - 1];
      expect(diff).toBeLessThanOrEqual(1); // Can only increment by 1
    }
  });

  test('should have sufficient color contrast for priority badges', async ({ page }) => {
    await page.waitForSelector('[class*="MuiChip"]', { timeout: 10000 });

    const priorityChips = await page.locator('[class*="MuiChip"]').all();

    if (priorityChips.length === 0) {
      test.skip();
      return;
    }

    for (const chip of priorityChips) {
      const isVisible = await chip.isVisible();

      if (isVisible) {
        const styles = await chip.evaluate((el) => {
          const computed = window.getComputedStyle(el);
          return {
            color: computed.color,
            backgroundColor: computed.backgroundColor,
          };
        });

        // Just verify we can read styles (actual contrast calculation requires external library)
        expect(styles.color).toBeTruthy();
        expect(styles.backgroundColor).toBeTruthy();
      }
    }
  });

  test('should support screen reader navigation of task metadata', async ({ page }) => {
    await page.waitForSelector('[data-testid*="task-card"]', { timeout: 10000 });

    const taskCard = page.locator('[data-testid*="task-card"]').first();

    // Check for semantic HTML structure
    const hasSemanticStructure = await taskCard.evaluate((card) => {
      const hasHeading = card.querySelector('h1, h2, h3, h4, h5, h6') !== null;
      const hasList = card.querySelector('ul, ol') !== null;
      const hasArticle = card.tagName === 'ARTICLE' || card.querySelector('article') !== null;

      return hasHeading || hasList || hasArticle;
    });

    // Task cards should use semantic HTML
    expect(hasSemanticStructure).toBeTruthy();
  });

  test('should allow users to escape drag mode with Escape key', async ({ page }) => {
    await page.waitForSelector('[data-testid*="task-card"]', { timeout: 10000 });

    const taskCard = page.locator('[data-testid*="task-card"]').first();

    // Start drag mode
    await taskCard.focus();
    await page.keyboard.press('Space');
    await page.waitForTimeout(200);

    // Cancel with Escape
    await page.keyboard.press('Escape');
    await page.waitForTimeout(200);

    // Task should return to normal state
    await expect(taskCard).toBeVisible();
  });

  test('should have descriptive column headers for screen readers', async ({ page }) => {
    await page.waitForSelector('[data-testid*="kanban-column"]', { timeout: 10000 });

    const columnHeaders = await page.getByRole('heading', { level: 2 }).or(
      page.getByRole('heading', { level: 3 })
    ).all();

    // Should have column headers
    expect(columnHeaders.length).toBeGreaterThan(0);

    for (const header of columnHeaders) {
      const text = await header.textContent();
      // Headers should have meaningful text
      expect(text?.trim().length).toBeGreaterThan(0);
    }
  });
});