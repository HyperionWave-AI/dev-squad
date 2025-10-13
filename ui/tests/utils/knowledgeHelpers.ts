/**
 * Knowledge Base Test Helpers
 *
 * Reusable helper functions for knowledge base E2E testing:
 * - Collection selection
 * - Search interactions
 * - Result navigation
 * - Recent search validation
 * - Accessibility checks (reuses existing accessibility utils)
 */

import { Page, Locator, expect } from '@playwright/test';

/**
 * Select a collection by clicking its card
 * Works with MUI Card components
 */
export async function selectCollection(
  page: Page,
  collectionName: string
): Promise<void> {
  // Find collection card by name
  const collectionCard = page
    .locator('[role="button"]')
    .filter({ hasText: collectionName })
    .or(
      page.getByRole('button', { name: new RegExp(collectionName, 'i') })
    )
    .or(
      page.locator('h6, h5, h4').filter({ hasText: collectionName }).locator('..')
    );

  await collectionCard.first().click();
  await page.waitForTimeout(200); // Allow for state update
}

/**
 * Wait for search results to appear
 * Returns the results container
 */
export async function waitForSearchResults(
  page: Page,
  timeout: number = 5000
): Promise<Locator> {
  // Wait for results accordion/container to appear
  const resultsContainer = page
    .locator('[data-testid="search-results"]')
    .or(page.locator('.MuiAccordion-root'))
    .or(page.locator('[role="region"]').filter({ hasText: /result/i }));

  await expect(resultsContainer.first()).toBeVisible({ timeout });

  return resultsContainer;
}

/**
 * Expand a search result accordion by index
 * Works with MUI Accordion components
 */
export async function expandResult(
  page: Page,
  index: number
): Promise<void> {
  // Find accordion summary buttons
  const accordions = page.locator('.MuiAccordion-root').or(
    page.locator('[data-testid^="result-"]')
  );

  const targetAccordion = accordions.nth(index);

  // Click accordion summary to expand
  const accordionButton = targetAccordion
    .locator('.MuiAccordionSummary-root')
    .or(targetAccordion.locator('[role="button"]').first());

  await accordionButton.click();
  await page.waitForTimeout(300); // Allow for expansion animation
}

/**
 * Check if a result is expanded
 */
export async function isResultExpanded(
  page: Page,
  index: number
): Promise<boolean> {
  const accordions = page.locator('.MuiAccordion-root');
  const targetAccordion = accordions.nth(index);

  // Check if accordion has expanded class or aria-expanded
  const isExpanded = await targetAccordion.evaluate((el) => {
    return (
      el.classList.contains('Mui-expanded') ||
      el.querySelector('[aria-expanded="true"]') !== null
    );
  });

  return isExpanded;
}

/**
 * Assert that a search query is saved in localStorage as recent search
 */
export async function assertRecentSearch(
  page: Page,
  query: string
): Promise<void> {
  const recentSearches = await page.evaluate(() => {
    const stored = localStorage.getItem('recentKnowledgeSearches');
    return stored ? JSON.parse(stored) : [];
  });

  expect(recentSearches).toContain(query);
}

/**
 * Get recent searches from localStorage
 */
export async function getRecentSearches(page: Page): Promise<string[]> {
  return await page.evaluate(() => {
    const stored = localStorage.getItem('recentKnowledgeSearches');
    return stored ? JSON.parse(stored) : [];
  });
}

/**
 * Clear recent searches from localStorage
 */
export async function clearRecentSearches(page: Page): Promise<void> {
  await page.evaluate(() => {
    localStorage.removeItem('recentKnowledgeSearches');
  });
}

/**
 * Submit search form with collection and query
 */
export async function submitSearch(
  page: Page,
  options: {
    collection?: string;
    query: string;
    waitForResults?: boolean;
  }
): Promise<void> {
  const { collection, query, waitForResults = true } = options;

  // Select collection if provided
  if (collection) {
    const collectionSelect = page.locator('#collection').or(
      page.getByRole('combobox', { name: /collection/i })
    );

    await collectionSelect.click();
    await collectionSelect.selectOption(collection);
    await page.waitForTimeout(100);
  }

  // Enter search query
  const searchInput = page.locator('#query').or(
    page.getByRole('textbox', { name: /search|query/i })
  );

  await searchInput.fill(query);

  // Submit form
  const submitButton = page.getByRole('button', { name: /search/i });
  await submitButton.click();

  // Wait for results if requested
  if (waitForResults) {
    await page.waitForTimeout(500); // Allow for API call and render
  }
}

/**
 * Get count of visible search results
 */
export async function getResultCount(page: Page): Promise<number> {
  const results = page
    .locator('[data-testid^="result-"]')
    .or(page.locator('.MuiAccordion-root'));

  return await results.count();
}

/**
 * Get result metadata by index
 */
export async function getResultMetadata(
  page: Page,
  index: number
): Promise<{
  score?: string;
  collection?: string;
  tags?: string[];
}> {
  const result = page.locator('.MuiAccordion-root').nth(index);

  // Extract score
  const scoreElement = result.locator('[data-testid="result-score"]').or(
    result.locator('text=/Score:|\\d+\\.\\d+/')
  );
  const score = await scoreElement.textContent().catch(() => null);

  // Extract collection
  const collectionElement = result.locator('[data-testid="result-collection"]').or(
    result.locator('.MuiChip-label').first()
  );
  const collection = await collectionElement.textContent().catch(() => null);

  // Extract tags
  const tagElements = result.locator('.MuiChip-label').or(
    result.locator('[data-testid="result-tag"]')
  );
  const tagCount = await tagElements.count();
  const tags: string[] = [];

  for (let i = 0; i < tagCount; i++) {
    const tagText = await tagElements.nth(i).textContent();
    if (tagText) tags.push(tagText.trim());
  }

  return {
    score: score?.trim() || undefined,
    collection: collection?.trim() || undefined,
    tags: tags.length > 0 ? tags : undefined,
  };
}

/**
 * Navigate to next page of results (if pagination exists)
 */
export async function goToNextPage(page: Page): Promise<void> {
  const nextButton = page
    .getByRole('button', { name: /next/i })
    .or(page.locator('[aria-label*="next"]'))
    .or(page.locator('.MuiPagination-root button[aria-label*="next"]'));

  await nextButton.click();
  await page.waitForTimeout(300); // Allow for page update
}

/**
 * Navigate to previous page of results
 */
export async function goToPreviousPage(page: Page): Promise<void> {
  const prevButton = page
    .getByRole('button', { name: /previous/i })
    .or(page.locator('[aria-label*="previous"]'))
    .or(page.locator('.MuiPagination-root button[aria-label*="previous"]'));

  await prevButton.click();
  await page.waitForTimeout(300); // Allow for page update
}

/**
 * Check if pagination is visible
 */
export async function hasPagination(page: Page): Promise<boolean> {
  const pagination = page.locator('.MuiPagination-root').or(
    page.locator('[data-testid="pagination"]')
  );

  return await pagination.isVisible().catch(() => false);
}

/**
 * Get current page number from pagination
 */
export async function getCurrentPageNumber(page: Page): Promise<number> {
  const activeButton = page
    .locator('.MuiPagination-root button[aria-current="true"]')
    .or(page.locator('[data-testid="pagination"] button.active'));

  const pageText = await activeButton.textContent();
  return pageText ? parseInt(pageText.trim()) : 1;
}

/**
 * Check for empty results state
 */
export async function hasEmptyState(page: Page): Promise<boolean> {
  const emptyState = page
    .getByText(/no results|no entries found/i)
    .or(page.locator('[data-testid="empty-state"]'));

  return await emptyState.isVisible().catch(() => false);
}

/**
 * Check for loading state
 */
export async function isLoading(page: Page): Promise<boolean> {
  const loadingIndicator = page
    .locator('.MuiCircularProgress-root')
    .or(page.locator('.MuiSkeleton-root'))
    .or(page.locator('[data-testid="loading"]'));

  return await loadingIndicator.first().isVisible().catch(() => false);
}

/**
 * Check for error message
 */
export async function hasError(page: Page): Promise<boolean> {
  const errorAlert = page
    .locator('[role="alert"]')
    .or(page.locator('.MuiAlert-standardError'))
    .or(page.getByText(/error|failed/i));

  return await errorAlert.first().isVisible().catch(() => false);
}

/**
 * Get error message text
 */
export async function getErrorMessage(page: Page): Promise<string | null> {
  const errorAlert = page
    .locator('[role="alert"]')
    .or(page.locator('.MuiAlert-standardError'));

  return await errorAlert.first().textContent().catch(() => null);
}

/**
 * Wait for debounced search input (300ms)
 */
export async function waitForSearchDebounce(page: Page): Promise<void> {
  await page.waitForTimeout(350); // 300ms debounce + 50ms buffer
}

/**
 * Focus on search input using keyboard shortcut (Cmd+K / Ctrl+K)
 */
export async function focusSearchWithShortcut(page: Page): Promise<void> {
  // Detect OS
  const isMac = await page.evaluate(() => navigator.platform.toLowerCase().includes('mac'));

  if (isMac) {
    await page.keyboard.press('Meta+K');
  } else {
    await page.keyboard.press('Control+K');
  }

  await page.waitForTimeout(100);
}

/**
 * Check if search input is focused
 */
export async function isSearchInputFocused(page: Page): Promise<boolean> {
  const searchInput = page.locator('#query').or(
    page.getByRole('textbox', { name: /search|query/i })
  );

  return await searchInput.evaluate(
    (el) => el === document.activeElement
  );
}

/**
 * Clear search input and results
 */
export async function clearSearch(page: Page): Promise<void> {
  const clearButton = page.getByRole('button', { name: /clear/i });

  if (await clearButton.isVisible()) {
    await clearButton.click();
  } else {
    // Fallback: clear input manually
    const searchInput = page.locator('#query').or(
      page.getByRole('textbox', { name: /search|query/i })
    );
    await searchInput.clear();
  }

  await page.waitForTimeout(200);
}

/**
 * Get selected collection name
 */
export async function getSelectedCollection(page: Page): Promise<string | null> {
  const collectionSelect = page.locator('#collection').or(
    page.getByRole('combobox', { name: /collection/i })
  );

  return await collectionSelect.inputValue().catch(() => null);
}

/**
 * Get all available collection options
 */
export async function getAvailableCollections(page: Page): Promise<string[]> {
  const collectionSelect = page.locator('#collection').or(
    page.getByRole('combobox', { name: /collection/i })
  );

  const options = await collectionSelect.locator('option').allTextContents();

  // Filter out placeholder option
  return options.filter(opt => opt.trim() && !opt.toLowerCase().includes('select'));
}

/**
 * Check if result contains code block
 */
export async function resultHasCodeBlock(
  page: Page,
  index: number
): Promise<boolean> {
  const result = page.locator('.MuiAccordion-root').nth(index);

  const codeBlock = result.locator('code, pre').or(
    result.locator('[class*="syntax-highlight"]')
  );

  return await codeBlock.isVisible().catch(() => false);
}

/**
 * Get code block content from result
 */
export async function getCodeBlockContent(
  page: Page,
  index: number
): Promise<string | null> {
  const result = page.locator('.MuiAccordion-root').nth(index);

  const codeBlock = result.locator('code, pre').first();

  return await codeBlock.textContent().catch(() => null);
}
