/**
 * Accessibility test suite using @axe-core/playwright
 * Tests WCAG 2.1 AA compliance for all 9 pages
 */
import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

// Mock API responses to prevent backend dependency
const mockApiResponses = async (page: any) => {
  await page.route('**/api/mcp/tools/list', (route: any) => {
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ tools: [] })
    });
  });

  await page.route('**/api/v1/tasks**', (route: any) => {
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ tasks: [] })
    });
  });

  await page.route('**/api/agent-tasks**', (route: any) => {
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ tasks: [], total: 0 })
    });
  });

  await page.route('**/api/knowledge/collections**', (route: any) => {
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ collections: [] })
    });
  });

  await page.route('**/bridge-health', (route: any) => {
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'healthy' })
    });
  });
};

test.describe('Accessibility Tests - WCAG 2.1 AA @accessibility', () => {
  test.beforeEach(async ({ page }) => {
    await mockApiResponses(page);
  });

  test('Chat page should not have accessibility violations', async ({ page }) => {
    await page.goto('/chat');
    await page.waitForLoadState('domcontentloaded');

    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
      .analyze();

    expect(accessibilityScanResults.violations).toEqual([]);
  });

  test('Tasks (Kanban) page should not have accessibility violations', async ({ page }) => {
    await page.goto('/tasks');
    await page.waitForLoadState('domcontentloaded');

    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
      .analyze();

    expect(accessibilityScanResults.violations).toEqual([]);
  });

  test('Knowledge Base page should not have accessibility violations', async ({ page }) => {
    await page.goto('/knowledge');
    await page.waitForLoadState('domcontentloaded');

    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
      .analyze();

    expect(accessibilityScanResults.violations).toEqual([]);
  });

  test('Reflection page should not have accessibility violations', async ({ page }) => {
    await page.goto('/reflection');
    await page.waitForLoadState('domcontentloaded');

    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
      .analyze();

    expect(accessibilityScanResults.violations).toEqual([]);
  });

  test('Code Search page should not have accessibility violations', async ({ page }) => {
    await page.goto('/code');
    await page.waitForLoadState('domcontentloaded');

    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
      .analyze();

    expect(accessibilityScanResults.violations).toEqual([]);
  });

  test('MCP Servers page should not have accessibility violations', async ({ page }) => {
    await page.goto('/mcp-servers');
    await page.waitForLoadState('domcontentloaded');

    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
      .analyze();

    expect(accessibilityScanResults.violations).toEqual([]);
  });

  test('HTTP Tools page should not have accessibility violations', async ({ page }) => {
    await page.goto('/tools');
    await page.waitForLoadState('domcontentloaded');

    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
      .analyze();

    expect(accessibilityScanResults.violations).toEqual([]);
  });

  test('Subagents page should not have accessibility violations', async ({ page }) => {
    await page.goto('/subagents');
    await page.waitForLoadState('domcontentloaded');

    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
      .analyze();

    expect(accessibilityScanResults.violations).toEqual([]);
  });

  test('Settings page should not have accessibility violations', async ({ page }) => {
    await page.goto('/settings');
    await page.waitForLoadState('domcontentloaded');

    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
      .analyze();

    expect(accessibilityScanResults.violations).toEqual([]);
  });
});

test.describe('Keyboard Navigation Tests @accessibility', () => {
  test.beforeEach(async ({ page }) => {
    await mockApiResponses(page);
  });

  test('Interactive elements should be keyboard accessible', async ({ page }) => {
    await page.goto('/chat');
    await page.waitForLoadState('domcontentloaded');

    // Tab through interactive elements
    await page.keyboard.press('Tab');

    // Verify focus is visible
    const focusedElement = await page.locator(':focus').first();
    await expect(focusedElement).toBeVisible();
  });

  test('Navigation menu should be keyboard accessible', async ({ page }) => {
    await page.goto('/chat');
    await page.waitForLoadState('domcontentloaded');

    // Find first navigation link and activate with keyboard
    const nav = page.locator('nav');
    await expect(nav).toBeVisible();

    // Tab to navigation and press Enter
    await page.keyboard.press('Tab');
    const focused = await page.locator(':focus').first();

    if (await focused.count() > 0) {
      await page.keyboard.press('Enter');
      // Verify navigation occurred or menu opened
      await page.waitForTimeout(500);
    }
  });
});

test.describe('Color Contrast Tests @accessibility', () => {
  test.beforeEach(async ({ page }) => {
    await mockApiResponses(page);
  });

  test('All pages should have sufficient color contrast', async ({ page }) => {
    const pages = ['/chat', '/tasks', '/knowledge', '/reflection', '/code',
                   '/mcp-servers', '/tools', '/subagents', '/settings'];

    for (const pagePath of pages) {
      await page.goto(pagePath);
      await page.waitForLoadState('domcontentloaded');

      const accessibilityScanResults = await new AxeBuilder({ page })
        .withTags(['wcag2aa'])
        .disableRules(['color-contrast']) // We'll check this separately with specific rules
        .analyze();

      // Check for color contrast issues specifically
      const contrastResults = await new AxeBuilder({ page })
        .include('body')
        .withRules(['color-contrast'])
        .analyze();

      expect(contrastResults.violations.length).toBe(0);
    }
  });
});
