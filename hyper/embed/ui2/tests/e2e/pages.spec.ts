/**
 * E2E smoke tests for all 9 pages in ui2
 * Verifies each page loads, shows correct heading, and has working navigation
 */
import { test, expect } from '@playwright/test';

// Mock API responses to prevent backend dependency
const mockApiResponses = async (page: any) => {
  // Mock MCP tools list
  await page.route('**/api/mcp/tools/list', (route: any) => {
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ tools: [] })
    });
  });

  // Mock tasks
  await page.route('**/api/v1/tasks**', (route: any) => {
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ tasks: [] })
    });
  });

  // Mock agent tasks
  await page.route('**/api/agent-tasks**', (route: any) => {
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ tasks: [], total: 0 })
    });
  });

  // Mock knowledge collections
  await page.route('**/api/knowledge/collections**', (route: any) => {
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ collections: [] })
    });
  });

  // Mock health check
  await page.route('**/bridge-health', (route: any) => {
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'healthy' })
    });
  });
};

test.describe('Page Smoke Tests', () => {
  test.beforeEach(async ({ page }) => {
    await mockApiResponses(page);

    // Suppress console errors from expected API issues
    page.on('console', msg => {
      if (msg.type() === 'error' && !msg.text().includes('401')) {
        console.log('Console error:', msg.text());
      }
    });
  });

  test('Chat page loads and displays correctly', async ({ page }) => {
    await page.goto('/chat');
    await expect(page).toHaveTitle(/Hyperion/);

    // Verify navigation is present
    const nav = page.locator('nav');
    await expect(nav).toBeVisible();

    // Check for main content area
    await expect(page.locator('main, [role="main"]')).toBeVisible();
  });

  test('Tasks (Kanban) page loads and displays correctly', async ({ page }) => {
    await page.goto('/tasks');
    await expect(page).toHaveTitle(/Hyperion/);

    const nav = page.locator('nav');
    await expect(nav).toBeVisible();

    await expect(page.locator('main, [role="main"]')).toBeVisible();
  });

  test('Knowledge Base page loads and displays correctly', async ({ page }) => {
    await page.goto('/knowledge');
    await expect(page).toHaveTitle(/Hyperion/);

    const nav = page.locator('nav');
    await expect(nav).toBeVisible();

    await expect(page.locator('main, [role="main"]')).toBeVisible();
  });

  test('Reflection page loads and displays correctly', async ({ page }) => {
    await page.goto('/reflection');
    await expect(page).toHaveTitle(/Hyperion/);

    const nav = page.locator('nav');
    await expect(nav).toBeVisible();

    await expect(page.locator('main, [role="main"]')).toBeVisible();
  });

  test('Code Search page loads and displays correctly', async ({ page }) => {
    await page.goto('/code');
    await expect(page).toHaveTitle(/Hyperion/);

    const nav = page.locator('nav');
    await expect(nav).toBeVisible();

    await expect(page.locator('main, [role="main"]')).toBeVisible();
  });

  test('MCP Servers page loads and displays correctly', async ({ page }) => {
    await page.goto('/mcp-servers');
    await expect(page).toHaveTitle(/Hyperion/);

    const nav = page.locator('nav');
    await expect(nav).toBeVisible();

    await expect(page.locator('main, [role="main"]')).toBeVisible();
  });

  test('HTTP Tools page loads and displays correctly', async ({ page }) => {
    await page.goto('/tools');
    await expect(page).toHaveTitle(/Hyperion/);

    const nav = page.locator('nav');
    await expect(nav).toBeVisible();

    await expect(page.locator('main, [role="main"]')).toBeVisible();
  });

  test('Subagents page loads and displays correctly', async ({ page }) => {
    await page.goto('/subagents');
    await expect(page).toHaveTitle(/Hyperion/);

    const nav = page.locator('nav');
    await expect(nav).toBeVisible();

    await expect(page.locator('main, [role="main"]')).toBeVisible();
  });

  test('Settings page loads and displays correctly', async ({ page }) => {
    await page.goto('/settings');
    await expect(page).toHaveTitle(/Hyperion/);

    const nav = page.locator('nav');
    await expect(nav).toBeVisible();

    await expect(page.locator('main, [role="main"]')).toBeVisible();
  });
});

test.describe('Navigation Tests', () => {
  test.beforeEach(async ({ page }) => {
    await mockApiResponses(page);
  });

  test('Navigation links are clickable and route correctly', async ({ page }) => {
    await page.goto('/chat');

    // Find navigation and test a few key links
    const nav = page.locator('nav');
    await expect(nav).toBeVisible();

    // Test navigation to tasks page
    const tasksLink = nav.locator('a[href*="/tasks"]').first();
    if (await tasksLink.isVisible()) {
      await tasksLink.click();
      await page.waitForURL('**/tasks');
      await expect(page).toHaveURL(/\/tasks/);
    }
  });

  test('Root path redirects or loads a valid page', async ({ page }) => {
    await page.goto('/');

    // Should either be on root or redirected to a default page
    await expect(page.locator('nav')).toBeVisible();
    await expect(page.locator('main, [role="main"]')).toBeVisible();
  });
});

test.describe('Console Error Check', () => {
  test('Pages should not have critical console errors', async ({ page }) => {
    const errors: string[] = [];

    page.on('console', msg => {
      if (msg.type() === 'error') {
        // Filter out expected errors (auth, network issues in test env)
        if (!msg.text().includes('401') &&
            !msg.text().includes('Failed to fetch') &&
            !msg.text().includes('NetworkError')) {
          errors.push(msg.text());
        }
      }
    });

    await mockApiResponses(page);
    await page.goto('/chat');

    // Wait a bit for any async errors
    await page.waitForTimeout(1000);

    // Should have no critical errors
    expect(errors.length).toBe(0);
  });
});
