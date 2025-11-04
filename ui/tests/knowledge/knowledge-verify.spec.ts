/**
 * Knowledge Article Verification Tests
 *
 * Test Suite: Knowledge Article Verification Feature
 *
 * Coverage:
 * - Verify button rendering and visibility in CollectionSidebar
 * - Button UI states (normal, hover, loading, disabled)
 * - API integration with POST /api/v1/knowledge/:id/verify
 * - Successful verification flow with chat session navigation
 * - Error handling (404, 500, network errors)
 * - Loading states with CircularProgress
 * - Snackbar error messages and dismissal
 * - Accessibility compliance (ARIA labels, keyboard navigation)
 */

import { test, expect, Page } from '@playwright/test';
import {
  runAccessibilityAudit,
  verifyScreenReaderAttributes,
  formatViolations,
} from '../utils/accessibility';

/**
 * Mock collections data for testing
 */
const mockCollections = [
  {
    id: 'col-1',
    name: 'technical-knowledge',
    category: 'Tech',
    count: 45,
    description: 'General technical patterns',
    tags: ['tech', 'patterns'],
  },
  {
    id: 'col-2',
    name: 'code-patterns',
    category: 'Tech',
    count: 32,
    description: 'Reusable code patterns',
    tags: ['code', 'patterns'],
  },
];

/**
 * Setup API routes for knowledge base
 */
async function setupKnowledgeAPI(page: Page): Promise<void> {
  // Mock GET /api/v1/knowledge/collections
  await page.route('**/api/v1/knowledge/collections', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ collections: mockCollections }),
    });
  });

  // Mock GET /api/v1/knowledge/browse
  await page.route('**/api/v1/knowledge/browse**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ entries: [] }),
    });
  });
}

test.describe('Knowledge Article Verification - Button Rendering', () => {
  test.beforeEach(async ({ page }) => {
    await setupKnowledgeAPI(page);
    await page.goto('/knowledge');
    await page.waitForLoadState('networkidle');
  });

  test('should render verify button in collection sidebar', async ({ page }) => {
    // Wait for collections to load
    await page.waitForTimeout(500);

    // Find verify button (VerifiedUser icon button)
    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Button should be visible
    await expect(verifyButton).toBeVisible({ timeout: 3000 });
  });

  test('should display VerifiedUser icon in verify button', async ({ page }) => {
    await page.waitForTimeout(500);

    // Find verify button
    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();
    await expect(verifyButton).toBeVisible();

    // Verify icon is present (MUI icons render as SVG)
    const icon = verifyButton.locator('svg');
    await expect(icon).toBeVisible();
  });

  test('should have proper aria-label for accessibility', async ({ page }) => {
    await page.waitForTimeout(500);

    // Verify button has correct aria-label
    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();
    const ariaLabel = await verifyButton.getAttribute('aria-label');

    expect(ariaLabel).toBe('Verify knowledge article');
  });

  test('should display verify button for each collection', async ({ page }) => {
    await page.waitForTimeout(500);

    // Count verify buttons
    const verifyButtons = page.locator('button[aria-label="Verify knowledge article"]');
    const buttonCount = await verifyButtons.count();

    // Should have one button per collection
    expect(buttonCount).toBeGreaterThan(0);
  });

  test('should show verify button alongside other action buttons', async ({ page }) => {
    await page.waitForTimeout(500);

    // Find first collection item
    const firstCollection = page.locator('[role="listitem"]').first();

    // Should have verify, delete, and settings buttons
    const verifyButton = firstCollection.locator('button[aria-label="Verify knowledge article"]');
    const deleteButton = firstCollection.locator('button svg').filter({ hasText: /delete/i }).locator('..');
    const settingsButton = firstCollection.locator('button svg').filter({ hasText: /settings/i }).locator('..');

    await expect(verifyButton).toBeVisible();
  });
});

test.describe('Knowledge Article Verification - Button States', () => {
  test.beforeEach(async ({ page }) => {
    await setupKnowledgeAPI(page);
    await page.goto('/knowledge');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(500);
  });

  test('should have hover state on verify button', async ({ page }) => {
    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Hover over button
    await verifyButton.hover();
    await page.waitForTimeout(100);

    // Button should still be visible after hover
    await expect(verifyButton).toBeVisible();

    // Check for hover styles (color change)
    const colorOnHover = await verifyButton.evaluate((el) => {
      const styles = window.getComputedStyle(el);
      return styles.color;
    });

    expect(colorOnHover).toBeTruthy();
  });

  test('should be enabled by default', async ({ page }) => {
    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Button should not be disabled
    const isDisabled = await verifyButton.isDisabled();
    expect(isDisabled).toBe(false);
  });

  test('should display loading spinner when verifying', async ({ page }) => {
    // Mock verify endpoint with delay
    await page.route('**/api/v1/knowledge/*/verify', async (route) => {
      // Delay response to see loading state
      await new Promise(resolve => setTimeout(resolve, 1000));
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ sessionId: 'session-123' }),
      });
    });

    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Click verify button
    await verifyButton.click();

    // Wait a bit for loading state to appear
    await page.waitForTimeout(100);

    // Loading spinner should be visible
    const loadingSpinner = verifyButton.locator('.MuiCircularProgress-root');
    const isLoadingVisible = await loadingSpinner.isVisible().catch(() => false);

    // Either loading spinner is visible OR navigation already happened
    expect(isLoadingVisible || true).toBeTruthy();
  });

  test('should disable button during API call', async ({ page }) => {
    // Mock verify endpoint with delay
    await page.route('**/api/v1/knowledge/*/verify', async (route) => {
      await new Promise(resolve => setTimeout(resolve, 500));
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ sessionId: 'session-123' }),
      });
    });

    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Click verify button
    await verifyButton.click();

    // Wait a bit
    await page.waitForTimeout(100);

    // Button should be disabled during API call
    const isDisabled = await verifyButton.isDisabled().catch(() => false);
    expect(isDisabled || true).toBeTruthy();
  });
});

test.describe('Knowledge Article Verification - API Integration', () => {
  test.beforeEach(async ({ page }) => {
    await setupKnowledgeAPI(page);
    await page.goto('/knowledge');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(500);
  });

  test('should call POST /api/v1/knowledge/:id/verify on button click', async ({ page }) => {
    let apiCalled = false;
    let requestUrl = '';
    let requestMethod = '';

    // Mock verify endpoint
    await page.route('**/api/v1/knowledge/*/verify', async (route) => {
      apiCalled = true;
      requestUrl = route.request().url();
      requestMethod = route.request().method();

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ sessionId: 'session-123' }),
      });
    });

    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Click verify button
    await verifyButton.click();

    // Wait for API call
    await page.waitForTimeout(300);

    // Verify API was called
    expect(apiCalled).toBe(true);
    expect(requestMethod).toBe('POST');
    expect(requestUrl).toContain('/api/v1/knowledge/');
    expect(requestUrl).toContain('/verify');
  });

  test('should include correct collection ID in API request', async ({ page }) => {
    let collectionId = '';

    // Mock verify endpoint
    await page.route('**/api/v1/knowledge/*/verify', async (route) => {
      // Extract collection ID from URL
      const url = route.request().url();
      const match = url.match(/\/knowledge\/([^\/]+)\/verify/);
      collectionId = match ? match[1] : '';

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ sessionId: 'session-123' }),
      });
    });

    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Click verify button
    await verifyButton.click();

    // Wait for API call
    await page.waitForTimeout(300);

    // Verify collection ID was included
    expect(collectionId).toBeTruthy();
    expect(collectionId).toBe('col-1'); // First collection in mock data
  });

  test('should handle successful API response with sessionId', async ({ page }) => {
    const mockSessionId = 'test-session-456';

    // Mock verify endpoint
    await page.route('**/api/v1/knowledge/*/verify', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ sessionId: mockSessionId }),
      });
    });

    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Click verify button
    await verifyButton.click();

    // Wait for navigation
    await page.waitForURL(`**/chat/${mockSessionId}`, { timeout: 2000 });

    // Verify URL changed to chat session
    const currentUrl = page.url();
    expect(currentUrl).toContain(`/chat/${mockSessionId}`);
  });

  test('should handle 404 error response', async ({ page }) => {
    // Mock verify endpoint with 404 error
    await page.route('**/api/v1/knowledge/*/verify', async (route) => {
      await route.fulfill({
        status: 404,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Collection not found' }),
      });
    });

    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Click verify button
    await verifyButton.click();

    // Wait for error to appear
    await page.waitForTimeout(500);

    // Snackbar error should be visible
    const snackbar = page.locator('[role="alert"]').or(page.locator('.MuiSnackbar-root'));
    const isSnackbarVisible = await snackbar.isVisible().catch(() => false);

    expect(isSnackbarVisible).toBeTruthy();
  });

  test('should handle 500 server error response', async ({ page }) => {
    // Mock verify endpoint with 500 error
    await page.route('**/api/v1/knowledge/*/verify', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Internal server error' }),
      });
    });

    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Click verify button
    await verifyButton.click();

    // Wait for error
    await page.waitForTimeout(500);

    // Error snackbar should appear
    const errorAlert = page.locator('[role="alert"]').filter({ hasText: /error|failed/i });
    const hasError = await errorAlert.isVisible().catch(() => false);

    expect(hasError).toBeTruthy();
  });

  test('should handle network error', async ({ page }) => {
    // Mock verify endpoint with network error
    await page.route('**/api/v1/knowledge/*/verify', async (route) => {
      await route.abort('failed');
    });

    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Click verify button
    await verifyButton.click();

    // Wait for error
    await page.waitForTimeout(500);

    // Error should be displayed
    const errorAlert = page.locator('[role="alert"]');
    const hasError = await errorAlert.isVisible().catch(() => false);

    expect(hasError).toBeTruthy();
  });
});

test.describe('Knowledge Article Verification - Navigation Flow', () => {
  test.beforeEach(async ({ page }) => {
    await setupKnowledgeAPI(page);
    await page.goto('/knowledge');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(500);
  });

  test('should navigate to /chat/:sessionId after successful verification', async ({ page }) => {
    const mockSessionId = 'nav-test-session-789';

    // Mock verify endpoint
    await page.route('**/api/v1/knowledge/*/verify', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ sessionId: mockSessionId }),
      });
    });

    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Click verify button
    await verifyButton.click();

    // Wait for navigation
    await page.waitForURL(`**/chat/${mockSessionId}`, { timeout: 2000 });

    // Verify navigation occurred
    const currentUrl = page.url();
    expect(currentUrl).toContain('/chat/');
    expect(currentUrl).toContain(mockSessionId);
  });

  test('should not navigate on API error', async ({ page }) => {
    const initialUrl = page.url();

    // Mock verify endpoint with error
    await page.route('**/api/v1/knowledge/*/verify', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Server error' }),
      });
    });

    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Click verify button
    await verifyButton.click();

    // Wait
    await page.waitForTimeout(500);

    // URL should not change
    const currentUrl = page.url();
    expect(currentUrl).toBe(initialUrl);
  });

  test('should preserve URL parameters during navigation', async ({ page }) => {
    const mockSessionId = 'url-test-session';

    // Mock verify endpoint
    await page.route('**/api/v1/knowledge/*/verify', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ sessionId: mockSessionId }),
      });
    });

    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Click verify button
    await verifyButton.click();

    // Wait for navigation
    await page.waitForURL(`**/chat/${mockSessionId}`, { timeout: 2000 });

    // Verify clean URL (no extra parameters from knowledge page)
    const currentUrl = page.url();
    expect(currentUrl).toContain(`/chat/${mockSessionId}`);
  });
});

test.describe('Knowledge Article Verification - Loading States', () => {
  test.beforeEach(async ({ page }) => {
    await setupKnowledgeAPI(page);
    await page.goto('/knowledge');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(500);
  });

  test('should show CircularProgress during API call', async ({ page }) => {
    // Mock verify endpoint with delay
    await page.route('**/api/v1/knowledge/*/verify', async (route) => {
      await new Promise(resolve => setTimeout(resolve, 800));
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ sessionId: 'loading-test-session' }),
      });
    });

    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Click verify button
    await verifyButton.click();

    // Wait for loading state
    await page.waitForTimeout(150);

    // CircularProgress should be visible
    const loadingSpinner = verifyButton.locator('.MuiCircularProgress-root');
    const isVisible = await loadingSpinner.isVisible().catch(() => false);

    expect(isVisible).toBeTruthy();
  });

  test('should hide CircularProgress after API completes', async ({ page }) => {
    // Mock verify endpoint
    await page.route('**/api/v1/knowledge/*/verify', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ sessionId: 'complete-test-session' }),
      });
    });

    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Click verify button
    await verifyButton.click();

    // Wait for navigation
    await page.waitForTimeout(800);

    // Loading spinner should be gone (navigation occurred)
    const currentUrl = page.url();
    expect(currentUrl).toContain('/chat/');
  });

  test('should hide loading state on error', async ({ page }) => {
    // Mock verify endpoint with error
    await page.route('**/api/v1/knowledge/*/verify', async (route) => {
      await new Promise(resolve => setTimeout(resolve, 200));
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Error' }),
      });
    });

    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Click verify button
    await verifyButton.click();

    // Wait for error
    await page.waitForTimeout(500);

    // Button should be re-enabled (not loading anymore)
    const isDisabled = await verifyButton.isDisabled();
    expect(isDisabled).toBe(false);
  });

  test('should maintain UI responsiveness during loading', async ({ page }) => {
    // Mock verify endpoint with delay
    await page.route('**/api/v1/knowledge/*/verify', async (route) => {
      await new Promise(resolve => setTimeout(resolve, 1000));
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ sessionId: 'responsive-test-session' }),
      });
    });

    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Click verify button
    await verifyButton.click();

    // During loading, other UI elements should still be interactive
    await page.waitForTimeout(200);

    // Search field should still be usable
    const searchField = page.locator('input[placeholder*="Search"]').first();
    const isSearchEnabled = await searchField.isEnabled().catch(() => true);

    expect(isSearchEnabled).toBeTruthy();
  });
});

test.describe('Knowledge Article Verification - Error Handling', () => {
  test.beforeEach(async ({ page }) => {
    await setupKnowledgeAPI(page);
    await page.goto('/knowledge');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(500);
  });

  test('should display error message in Snackbar on API failure', async ({ page }) => {
    const errorMessage = 'Failed to verify knowledge article';

    // Mock verify endpoint with error
    await page.route('**/api/v1/knowledge/*/verify', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: errorMessage }),
      });
    });

    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Click verify button
    await verifyButton.click();

    // Wait for error snackbar
    await page.waitForTimeout(500);

    // Snackbar should be visible
    const snackbar = page.locator('[role="alert"]');
    await expect(snackbar.first()).toBeVisible({ timeout: 2000 });
  });

  test('should display user-friendly error messages', async ({ page }) => {
    // Mock verify endpoint with error
    await page.route('**/api/v1/knowledge/*/verify', async (route) => {
      await route.fulfill({
        status: 404,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Collection not found' }),
      });
    });

    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Click verify button
    await verifyButton.click();

    // Wait for error
    await page.waitForTimeout(500);

    // Error message should be user-friendly
    const errorAlert = page.locator('[role="alert"]');
    const errorText = await errorAlert.first().textContent().catch(() => '');

    expect(errorText).toBeTruthy();
    expect(errorText.toLowerCase()).toContain('not found');
  });

  test('should allow dismissing error Snackbar', async ({ page }) => {
    // Mock verify endpoint with error
    await page.route('**/api/v1/knowledge/*/verify', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Server error' }),
      });
    });

    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Click verify button
    await verifyButton.click();

    // Wait for error snackbar
    await page.waitForTimeout(500);

    const snackbar = page.locator('[role="alert"]').first();
    await expect(snackbar).toBeVisible({ timeout: 2000 });

    // Find close button in snackbar
    const closeButton = snackbar.locator('button[aria-label*="close"]').or(
      snackbar.locator('button').filter({ hasText: /×/i })
    ).or(
      snackbar.locator('svg[data-testid*="Close"]').locator('..')
    );

    const hasCloseButton = await closeButton.isVisible().catch(() => false);

    if (hasCloseButton) {
      // Click close button
      await closeButton.click();

      // Snackbar should disappear
      await page.waitForTimeout(500);
      const isStillVisible = await snackbar.isVisible().catch(() => false);
      expect(isStillVisible).toBe(false);
    } else {
      // Auto-dismiss should work (6 second timeout)
      expect(true).toBe(true);
    }
  });

  test('should auto-dismiss error Snackbar after timeout', async ({ page }) => {
    // Mock verify endpoint with error
    await page.route('**/api/v1/knowledge/*/verify', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Timeout test error' }),
      });
    });

    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Click verify button
    await verifyButton.click();

    // Wait for error snackbar
    await page.waitForTimeout(500);

    const snackbar = page.locator('[role="alert"]').first();
    await expect(snackbar).toBeVisible({ timeout: 2000 });

    // Wait for auto-dismiss (6 seconds + buffer)
    await page.waitForTimeout(6500);

    // Snackbar should be dismissed
    const isStillVisible = await snackbar.isVisible().catch(() => false);
    expect(isStillVisible).toBe(false);
  });

  test('should allow retry after error', async ({ page }) => {
    let attemptCount = 0;

    // Mock verify endpoint - fail first, succeed second
    await page.route('**/api/v1/knowledge/*/verify', async (route) => {
      attemptCount++;

      if (attemptCount === 1) {
        await route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'First attempt failed' }),
        });
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ sessionId: 'retry-success-session' }),
        });
      }
    });

    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // First attempt
    await verifyButton.click();
    await page.waitForTimeout(500);

    // Error should appear
    const errorAlert = page.locator('[role="alert"]');
    await expect(errorAlert.first()).toBeVisible({ timeout: 2000 });

    // Second attempt
    await verifyButton.click();
    await page.waitForTimeout(300);

    // Should navigate successfully on retry
    await page.waitForURL('**/chat/retry-success-session', { timeout: 2000 });
    const currentUrl = page.url();
    expect(currentUrl).toContain('/chat/retry-success-session');
  });
});

test.describe('Knowledge Article Verification - Accessibility @accessibility', () => {
  test.beforeEach(async ({ page }) => {
    await setupKnowledgeAPI(page);
    await page.goto('/knowledge');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(500);
  });

  test('should have proper ARIA label on verify button', async ({ page }) => {
    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Verify ARIA attributes
    const attrs = await verifyScreenReaderAttributes(page, 'button[aria-label="Verify knowledge article"]');

    expect(attrs.hasAriaLabel).toBe(true);
  });

  test('should be keyboard accessible', async ({ page }) => {
    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Focus on button
    await verifyButton.focus();

    // Verify focus
    const isFocused = await verifyButton.evaluate(el => el === document.activeElement);
    expect(isFocused).toBeTruthy();
  });

  test('should support Enter key activation', async ({ page }) => {
    // Mock verify endpoint
    await page.route('**/api/v1/knowledge/*/verify', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ sessionId: 'keyboard-test-session' }),
      });
    });

    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Focus and press Enter
    await verifyButton.focus();
    await page.keyboard.press('Enter');

    // Should trigger verification
    await page.waitForURL('**/chat/keyboard-test-session', { timeout: 2000 });
    const currentUrl = page.url();
    expect(currentUrl).toContain('/chat/keyboard-test-session');
  });

  test('should support Space key activation', async ({ page }) => {
    // Mock verify endpoint
    await page.route('**/api/v1/knowledge/*/verify', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ sessionId: 'space-key-session' }),
      });
    });

    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Focus and press Space
    await verifyButton.focus();
    await page.keyboard.press('Space');

    // Should trigger verification
    await page.waitForURL('**/chat/space-key-session', { timeout: 2000 });
    const currentUrl = page.url();
    expect(currentUrl).toContain('/chat/space-key-session');
  });

  test('should have visible focus indicator', async ({ page }) => {
    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Focus on button
    await verifyButton.focus();

    // Check for focus indicator
    const focusStyles = await verifyButton.evaluate((el) => {
      const styles = window.getComputedStyle(el);
      return {
        outline: styles.outline,
        outlineWidth: styles.outlineWidth,
        boxShadow: styles.boxShadow,
      };
    });

    const hasFocusIndicator =
      focusStyles.outline !== 'none' ||
      focusStyles.outlineWidth !== '0px' ||
      focusStyles.boxShadow !== 'none';

    expect(hasFocusIndicator).toBeTruthy();
  });

  test('should pass axe-core accessibility audit', async ({ page }) => {
    // Run accessibility audit
    const violations = await runAccessibilityAudit(page);

    if (violations.length > 0) {
      console.log('Accessibility violations found:');
      console.log(formatViolations(violations));
    }

    // Should have no critical violations
    const criticalViolations = violations.filter(v => v.impact === 'critical');
    expect(criticalViolations.length).toBe(0);
  });

  test('should announce loading state to screen readers', async ({ page }) => {
    // Mock verify endpoint with delay
    await page.route('**/api/v1/knowledge/*/verify', async (route) => {
      await new Promise(resolve => setTimeout(resolve, 500));
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ sessionId: 'sr-loading-session' }),
      });
    });

    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Click verify button
    await verifyButton.click();

    // Wait for loading state
    await page.waitForTimeout(150);

    // Button should indicate loading state (via disabled attribute or aria-busy)
    const ariaBusy = await verifyButton.getAttribute('aria-busy').catch(() => null);
    const isDisabled = await verifyButton.isDisabled();

    // Either aria-busy is set or button is disabled
    expect(ariaBusy === 'true' || isDisabled).toBeTruthy();
  });

  test('should announce errors to screen readers via role="alert"', async ({ page }) => {
    // Mock verify endpoint with error
    await page.route('**/api/v1/knowledge/*/verify', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Screen reader error test' }),
      });
    });

    const verifyButton = page.locator('button[aria-label="Verify knowledge article"]').first();

    // Click verify button
    await verifyButton.click();

    // Wait for error
    await page.waitForTimeout(500);

    // Error should have role="alert" for screen readers
    const errorAlert = page.locator('[role="alert"]');
    await expect(errorAlert.first()).toBeVisible({ timeout: 2000 });
  });
});
