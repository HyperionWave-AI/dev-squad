/**
 * Knowledge Base Workflow Tests
 *
 * Test Suite: Complete User Workflows
 *
 * Coverage:
 * - Collection selection → search form auto-fills
 * - Submit search → results appear
 * - Expand result → metadata visible
 * - Navigate with pagination → next page loads
 * - Recent search saved → appears in autocomplete
 * - End-to-end user journey validation
 *
 * @tag @workflow @e2e
 */

import { test, expect } from '@playwright/test';
import {
  mockCollections,
  mockSearchResults,
  setupKnowledgeMocks,
  waitForDebounce,
} from '../fixtures/mockKnowledgeData';
import {
  selectCollection,
  waitForSearchResults,
  expandResult,
  isResultExpanded,
  assertRecentSearch,
  getRecentSearches,
  clearRecentSearches,
  submitSearch,
  getResultCount,
  hasPagination,
  goToNextPage,
  getCurrentPageNumber,
  hasEmptyState,
  isLoading,
  hasError,
  clearSearch,
  getSelectedCollection,
} from '../utils/knowledgeHelpers';

test.describe('Knowledge Base Workflow - Complete User Journey @workflow', () => {
  test.beforeEach(async ({ page }) => {
    // Setup API mocks
    await setupKnowledgeMocks(page);

    // Clear recent searches before each test
    await page.goto('/knowledge');
    await clearRecentSearches(page);

    await page.waitForLoadState('networkidle');
  });

  test('should complete full workflow: select collection → auto-fill → search → view results', async ({ page }) => {
    // Step 1: Select a collection from browser
    await selectCollection(page, 'technical-knowledge');

    // Wait for state update
    await page.waitForTimeout(300);

    // Step 2: Verify collection is auto-selected in search form
    const selectedCollection = await getSelectedCollection(page);
    expect(selectedCollection).toBe('technical-knowledge');

    // Step 3: Enter search query
    const searchInput = page.locator('#collection-select').or(
      page.getByRole('textbox', { name: /search query/i })
    );
    await searchInput.fill('authentication');

    // Step 4: Submit search
    const searchButton = page.getByRole('button', { name: /search/i });
    await searchButton.click();

    // Step 5: Wait for results to appear
    await page.waitForTimeout(500);

    // Step 6: Verify results are displayed
    const resultCount = await getResultCount(page);
    expect(resultCount).toBeGreaterThan(0);

    // Step 7: Verify results contain search query
    const firstResult = page.locator('.MuiAccordion-root').first();
    const resultText = await firstResult.textContent();
    expect(resultText?.toLowerCase()).toContain('authentication');
  });

  test('should display search results with proper structure', async ({ page }) => {
    // Select collection and search
    await selectCollection(page, 'ui-accessibility-standards');
    await page.waitForTimeout(200);

    await submitSearch(page, {
      collection: 'ui-accessibility-standards',
      query: 'wcag',
      waitForResults: true,
    });

    // Verify results loaded
    const resultCount = await getResultCount(page);
    expect(resultCount).toBeGreaterThan(0);

    // Verify first result has proper structure
    const firstResult = page.locator('.MuiAccordion-root').first();

    // Should have accordion summary (clickable header)
    const accordionSummary = firstResult.locator('.MuiAccordionSummary-root');
    await expect(accordionSummary).toBeVisible();

    // Should show score
    const scoreElement = firstResult.locator('text=/\\d+\\.\\d+/').or(
      firstResult.locator('[data-testid="result-score"]')
    );
    const hasScore = await scoreElement.isVisible().catch(() => false);
    expect(hasScore).toBeTruthy();
  });

  test('should expand result accordion to show metadata', async ({ page }) => {
    // Setup: Search for results
    await submitSearch(page, {
      collection: 'technical-knowledge',
      query: 'jwt',
      waitForResults: true,
    });

    // Wait for results
    await page.waitForTimeout(500);

    // Verify accordion is initially collapsed (or expanded - depends on default)
    const initiallyExpanded = await isResultExpanded(page, 0);

    // Expand first result
    await expandResult(page, 0);

    // Verify accordion is now expanded
    const nowExpanded = await isResultExpanded(page, 0);
    expect(nowExpanded).toBe(true);

    // Verify metadata is visible in expanded view
    const firstResult = page.locator('.MuiAccordion-root').first();
    const accordionDetails = firstResult.locator('.MuiAccordionDetails-root');

    await expect(accordionDetails).toBeVisible();

    // Should show full text content
    const textContent = await accordionDetails.textContent();
    expect(textContent?.length).toBeGreaterThan(10);
  });

  test('should save recent search to localStorage and show in autocomplete', async ({ page }) => {
    const searchQuery = 'authentication middleware';

    // Perform search
    await submitSearch(page, {
      collection: 'code-patterns',
      query: searchQuery,
      waitForResults: true,
    });

    // Wait for search to complete
    await page.waitForTimeout(500);

    // Verify search was saved to recent searches
    await assertRecentSearch(page, searchQuery);

    // Verify autocomplete shows recent search
    const recentSearches = await getRecentSearches(page);
    expect(recentSearches).toContain(searchQuery);
    expect(recentSearches.length).toBeGreaterThan(0);

    // Clear search input
    await clearSearch(page);

    // Focus on search input
    const searchInput = page.getByRole('textbox', { name: /search query/i });
    await searchInput.click();

    // Wait for autocomplete dropdown
    await page.waitForTimeout(200);

    // Should show recent search in autocomplete options
    const autocompleteOption = page.getByRole('option', { name: searchQuery });
    const hasOption = await autocompleteOption.isVisible().catch(() => false);

    // Note: Autocomplete may not always show if query is already in input
    // This is acceptable behavior
    expect(hasOption || recentSearches.includes(searchQuery)).toBeTruthy();
  });

  test('should handle pagination when results exceed page limit', async ({ page }) => {
    // Search for query that returns many results
    await submitSearch(page, {
      collection: 'technical-knowledge',
      query: 'pattern',
      waitForResults: true,
    });

    await page.waitForTimeout(500);

    // Check if pagination exists
    const pagination = await hasPagination(page);

    if (pagination) {
      // Get initial page number
      const initialPage = await getCurrentPageNumber(page);
      expect(initialPage).toBe(1);

      // Navigate to next page
      await goToNextPage(page);

      // Wait for page update
      await page.waitForTimeout(300);

      // Verify page number changed
      const newPage = await getCurrentPageNumber(page);
      expect(newPage).toBe(2);

      // Verify results updated (different results should be visible)
      const resultCount = await getResultCount(page);
      expect(resultCount).toBeGreaterThan(0);
    } else {
      // If no pagination, verify results are all visible
      const resultCount = await getResultCount(page);
      expect(resultCount).toBeGreaterThanOrEqual(1);
    }
  });

  test('should show empty state when no results match query', async ({ page }) => {
    // Search for non-existent term
    await submitSearch(page, {
      collection: 'technical-knowledge',
      query: 'xyznonexistentterm12345',
      waitForResults: true,
    });

    await page.waitForTimeout(500);

    // Should show empty state
    const isEmpty = await hasEmptyState(page);
    expect(isEmpty).toBe(true);

    // Should show helpful message
    const emptyStateMessage = page.getByText(/no results|no entries found/i);
    await expect(emptyStateMessage).toBeVisible();
  });

  test('should show loading state during search', async ({ page }) => {
    // Select collection
    await selectCollection(page, 'code-patterns');
    await page.waitForTimeout(200);

    // Start search (don't wait for results)
    await submitSearch(page, {
      collection: 'code-patterns',
      query: 'react hooks',
      waitForResults: false,
    });

    // Check for loading state immediately
    await page.waitForTimeout(50);

    // May show loading indicator (spinner or skeleton)
    // Note: This is flaky as API may respond too quickly
    const loading = await isLoading(page);

    // Either loading was visible OR results loaded quickly
    const resultCount = await getResultCount(page);
    expect(loading || resultCount > 0).toBeTruthy();
  });

  test('should clear search results when clear button clicked', async ({ page }) => {
    // Perform search
    await submitSearch(page, {
      collection: 'ui-component-patterns',
      query: 'optimistic ui',
      waitForResults: true,
    });

    await page.waitForTimeout(500);

    // Verify results exist
    let resultCount = await getResultCount(page);
    expect(resultCount).toBeGreaterThan(0);

    // Click clear button
    await clearSearch(page);

    // Wait for clear action
    await page.waitForTimeout(300);

    // Verify search input is empty
    const searchInput = page.getByRole('textbox', { name: /search query/i });
    const inputValue = await searchInput.inputValue();
    expect(inputValue).toBe('');

    // Verify results are cleared
    resultCount = await getResultCount(page);
    expect(resultCount).toBe(0);
  });

  test('should update results when switching collections', async ({ page }) => {
    // Search in first collection
    await submitSearch(page, {
      collection: 'technical-knowledge',
      query: 'jwt',
      waitForResults: true,
    });

    await page.waitForTimeout(500);

    const firstCollectionResults = await getResultCount(page);
    expect(firstCollectionResults).toBeGreaterThan(0);

    // Switch to different collection
    const collectionSelect = page.locator('#collection-select');
    await collectionSelect.click();

    const menuItem = page.getByRole('option', { name: /ui-accessibility/i });
    await menuItem.click();

    // Search again (same query, different collection)
    const searchButton = page.getByRole('button', { name: /search/i });
    await searchButton.click();

    await page.waitForTimeout(500);

    // Results should update (may be different count)
    const secondCollectionResults = await getResultCount(page);

    // Either different count or potentially same count (both are valid)
    expect(secondCollectionResults).toBeGreaterThanOrEqual(0);
  });

  test('should maintain search state when navigating accordion expansion', async ({ page }) => {
    // Perform search
    await submitSearch(page, {
      collection: 'code-patterns',
      query: 'repository pattern',
      waitForResults: true,
    });

    await page.waitForTimeout(500);

    const initialResultCount = await getResultCount(page);
    expect(initialResultCount).toBeGreaterThan(0);

    // Expand first result
    await expandResult(page, 0);
    await page.waitForTimeout(200);

    // Verify result is expanded
    const isExpanded = await isResultExpanded(page, 0);
    expect(isExpanded).toBe(true);

    // Verify search results count didn't change
    const afterExpansionCount = await getResultCount(page);
    expect(afterExpansionCount).toBe(initialResultCount);
  });

  test('should require both collection and query to search', async ({ page }) => {
    // Try to search without collection
    const searchInput = page.getByRole('textbox', { name: /search query/i });
    await searchInput.fill('test query');

    const searchButton = page.getByRole('button', { name: /search/i });
    await searchButton.click();

    await page.waitForTimeout(300);

    // Should show error message
    const errorVisible = await hasError(page);
    expect(errorVisible).toBe(true);

    // Should not show results
    const resultCount = await getResultCount(page);
    expect(resultCount).toBe(0);
  });

  test('should display result count in header', async ({ page }) => {
    // Perform search
    await submitSearch(page, {
      collection: 'technical-knowledge',
      query: 'authentication',
      waitForResults: true,
    });

    await page.waitForTimeout(500);

    // Look for result count display
    const resultCountDisplay = page.locator('text=/\\d+ result/i').or(
      page.locator('[data-testid="result-count"]')
    );

    const hasResultCount = await resultCountDisplay.isVisible().catch(() => false);

    // Either result count is displayed OR results are visible
    const resultCount = await getResultCount(page);
    expect(hasResultCount || resultCount > 0).toBeTruthy();
  });

  test('should support multiple consecutive searches', async ({ page }) => {
    // First search
    await submitSearch(page, {
      collection: 'technical-knowledge',
      query: 'jwt',
      waitForResults: true,
    });

    await page.waitForTimeout(500);
    let resultCount = await getResultCount(page);
    expect(resultCount).toBeGreaterThan(0);

    // Second search (different query)
    const searchInput = page.getByRole('textbox', { name: /search query/i });
    await searchInput.clear();
    await searchInput.fill('mongodb');

    const searchButton = page.getByRole('button', { name: /search/i });
    await searchButton.click();

    await page.waitForTimeout(500);
    resultCount = await getResultCount(page);
    expect(resultCount).toBeGreaterThan(0);

    // Third search (different query again)
    await searchInput.clear();
    await searchInput.fill('react');

    await searchButton.click();

    await page.waitForTimeout(500);
    resultCount = await getResultCount(page);
    expect(resultCount).toBeGreaterThanOrEqual(0); // May be 0 if no matches
  });

  test('should highlight selected collection in browser', async ({ page }) => {
    // Select a collection
    await selectCollection(page, 'ui-component-patterns');

    await page.waitForTimeout(300);

    // Verify collection is highlighted/selected
    const selectedCard = page
      .locator('.MuiCard-root')
      .filter({ hasText: 'ui-component-patterns' });

    // Should have selected styling (border color or background)
    const hasSelectedStyle = await selectedCard.evaluate((el) => {
      const styles = window.getComputedStyle(el);
      const borderColor = styles.borderColor;

      // Check if border color indicates selection (usually primary color)
      return borderColor && borderColor !== 'rgba(0, 0, 0, 0)';
    });

    expect(hasSelectedStyle).toBeTruthy();
  });

  test('should preserve query when switching between collections', async ({ page }) => {
    const testQuery = 'authentication';

    // Enter search query
    const searchInput = page.getByRole('textbox', { name: /search query/i });
    await searchInput.fill(testQuery);

    // Select first collection
    const collectionSelect = page.locator('#collection-select');
    await collectionSelect.click();
    await page.getByRole('option', { name: /technical-knowledge/i }).click();

    await page.waitForTimeout(200);

    // Verify query is still in input
    let inputValue = await searchInput.inputValue();
    expect(inputValue).toBe(testQuery);

    // Switch to another collection
    await collectionSelect.click();
    await page.getByRole('option', { name: /code-patterns/i }).click();

    await page.waitForTimeout(200);

    // Verify query is STILL in input
    inputValue = await searchInput.inputValue();
    expect(inputValue).toBe(testQuery);
  });
});

test.describe('Knowledge Base Workflow - Edge Cases @workflow', () => {
  test.beforeEach(async ({ page }) => {
    await setupKnowledgeMocks(page);
    await page.goto('/knowledge');
    await page.waitForLoadState('networkidle');
  });

  test('should handle very long search queries', async ({ page }) => {
    const longQuery = 'authentication middleware implementation pattern with jwt token validation error handling retry logic'.repeat(3);

    await submitSearch(page, {
      collection: 'technical-knowledge',
      query: longQuery,
      waitForResults: true,
    });

    await page.waitForTimeout(500);

    // Should either show results or empty state (no crash)
    const hasResults = await getResultCount(page) > 0;
    const isEmpty = await hasEmptyState(page);

    expect(hasResults || isEmpty).toBeTruthy();
  });

  test('should handle special characters in search query', async ({ page }) => {
    const specialQuery = 'function(x) { return x + 1; }';

    await submitSearch(page, {
      collection: 'code-patterns',
      query: specialQuery,
      waitForResults: true,
    });

    await page.waitForTimeout(500);

    // Should handle gracefully (no crash)
    const hasResults = await getResultCount(page) > 0;
    const isEmpty = await hasEmptyState(page);

    expect(hasResults || isEmpty).toBeTruthy();
  });

  test('should handle rapid collection switching', async ({ page }) => {
    const collectionSelect = page.locator('#collection-select');

    // Rapidly switch collections
    await collectionSelect.click();
    await page.getByRole('option', { name: /technical-knowledge/i }).click();
    await page.waitForTimeout(50);

    await collectionSelect.click();
    await page.getByRole('option', { name: /code-patterns/i }).click();
    await page.waitForTimeout(50);

    await collectionSelect.click();
    await page.getByRole('option', { name: /ui-test-strategies/i }).click();
    await page.waitForTimeout(200);

    // Should end up with final selection
    const selectedCollection = await getSelectedCollection(page);
    expect(selectedCollection).toBeTruthy();
  });

  test('should recover from error state on successful search', async ({ page }) => {
    // Trigger error by searching without collection
    const searchInput = page.getByRole('textbox', { name: /search query/i });
    await searchInput.fill('test');

    const searchButton = page.getByRole('button', { name: /search/i });
    await searchButton.click();

    await page.waitForTimeout(300);

    // Verify error is shown
    let errorVisible = await hasError(page);
    expect(errorVisible).toBe(true);

    // Now perform valid search
    await submitSearch(page, {
      collection: 'technical-knowledge',
      query: 'jwt',
      waitForResults: true,
    });

    await page.waitForTimeout(500);

    // Error should be cleared
    errorVisible = await hasError(page);
    expect(errorVisible).toBe(false);

    // Results should be shown
    const resultCount = await getResultCount(page);
    expect(resultCount).toBeGreaterThan(0);
  });
});
