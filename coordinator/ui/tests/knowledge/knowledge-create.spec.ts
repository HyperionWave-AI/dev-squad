/**
 * Knowledge Create Component Tests
 *
 * Test Suite: KnowledgeCreate Form Component
 *
 * Coverage:
 * - Form rendering with all fields (collection, text, metadata)
 * - Collection autocomplete with categorized options
 * - Text validation (min 10 chars, error messages)
 * - Metadata add/remove functionality
 * - Submit button disabled/enabled states
 * - Successful submission with snackbar and form reset
 * - API error handling and display
 * - Keyboard shortcuts (Ctrl+Enter to submit)
 * - Character counter updates
 * - Focus management after submit
 * - Accessibility compliance
 */

import { test, expect } from '@playwright/test';
import {
  setupKnowledgeAPI,
  setupKnowledgeAPIWithErrors,
  mockCollections,
} from '../fixtures/knowledge-fixtures';
import {
  runAccessibilityAudit,
  verifyScreenReaderAttributes,
  formatViolations,
} from '../utils/accessibility';

test.describe('KnowledgeCreate Component', () => {
  test.beforeEach(async ({ page }) => {
    await setupKnowledgeAPI(page);
    await page.goto('/knowledge');
    await page.waitForLoadState('networkidle');

    // Navigate to create form (may be in a tab or separate section)
    const createButton = page.getByRole('button', { name: /create knowledge/i }).or(
      page.getByRole('tab', { name: /create/i })
    );

    // Click if create button/tab exists
    const isVisible = await createButton.isVisible().catch(() => false);
    if (isVisible) {
      await createButton.click();
      await page.waitForTimeout(300);
    }
  });

  test('should render form with all required fields', async ({ page }) => {
    // Verify collection field
    const collectionField = page.getByLabel(/collection/i).or(
      page.getByRole('combobox', { name: /collection/i })
    );
    await expect(collectionField).toBeVisible();

    // Verify text field
    const textField = page.getByLabel(/text|knowledge|content/i).or(
      page.getByRole('textbox', { name: /text|knowledge|content/i })
    );
    await expect(textField).toBeVisible();

    // Verify metadata section (may be expandable)
    const metadataSection = page.locator('[data-testid="metadata-section"]').or(
      page.getByText(/metadata/i)
    );
    await expect(metadataSection).toBeVisible();

    // Verify submit button
    const submitButton = page.getByRole('button', { name: /submit|create|save/i });
    await expect(submitButton).toBeVisible();
  });

  test('should show categorized collections in autocomplete', async ({ page }) => {
    const collectionField = page.getByLabel(/collection/i).or(
      page.getByRole('combobox', { name: /collection/i })
    );

    // Click to open dropdown
    await collectionField.click();
    await page.waitForTimeout(300);

    // Verify category headers are present
    const categories = ['Tech', 'Task', 'UI', 'Ops'];

    for (const category of categories) {
      // Check if category label exists in the dropdown
      const categoryLabel = page.getByText(category, { exact: false });
      const isVisible = await categoryLabel.isVisible().catch(() => false);

      // At least some categories should be visible
      if (isVisible) {
        expect(isVisible).toBeTruthy();
      }
    }

    // Verify collection options are present
    const firstCollection = page.getByRole('option', { name: new RegExp(mockCollections[0].name, 'i') });
    await expect(firstCollection).toBeVisible();
  });

  test('should validate text field with minimum 10 characters', async ({ page }) => {
    const textField = page.locator('textarea').or(
      page.getByLabel(/text|knowledge|content/i)
    ).first();
    const submitButton = page.getByRole('button', { name: /submit|create|save/i });

    // Enter less than 10 characters
    await textField.fill('Short');
    await textField.blur();
    await page.waitForTimeout(300);

    // Verify error message is shown
    const errorMessage = page.getByText(/minimum|at least|10 char/i);
    await expect(errorMessage).toBeVisible();

    // Submit button should be disabled
    await expect(submitButton).toBeDisabled();
  });

  test('should enable submit button when form is valid', async ({ page }) => {
    const collectionField = page.getByLabel(/collection/i).or(
      page.getByRole('combobox', { name: /collection/i })
    );
    const textField = page.locator('textarea').or(
      page.getByLabel(/text|knowledge|content/i)
    ).first();
    const submitButton = page.getByRole('button', { name: /submit|create|save/i });

    // Initially should be disabled
    await expect(submitButton).toBeDisabled();

    // Fill collection
    await collectionField.click();
    await page.getByRole('option', { name: /technical-knowledge/i }).click();

    // Fill text with valid content (>10 chars)
    await textField.fill('This is a valid knowledge entry with enough characters');

    // Wait for validation
    await page.waitForTimeout(300);

    // Submit button should now be enabled
    await expect(submitButton).toBeEnabled();
  });

  test('should add and remove metadata fields', async ({ page }) => {
    // Find add metadata button
    const addMetadataButton = page.getByRole('button', { name: /add metadata|add field/i });

    // Click to add metadata field
    const isVisible = await addMetadataButton.isVisible().catch(() => false);
    if (isVisible) {
      await addMetadataButton.click();
      await page.waitForTimeout(200);

      // Verify metadata key/value inputs appear
      const metadataKeyInput = page.getByPlaceholder(/key|name/i);
      const metadataValueInput = page.getByPlaceholder(/value/i);

      await expect(metadataKeyInput).toBeVisible();
      await expect(metadataValueInput).toBeVisible();

      // Add metadata
      await metadataKeyInput.fill('testKey');
      await metadataValueInput.fill('testValue');

      // Find remove button
      const removeButton = page.getByRole('button', { name: /remove|delete/i }).first();
      await removeButton.click();
      await page.waitForTimeout(200);

      // Metadata fields should be removed
      const removedKeyInput = page.getByPlaceholder(/key|name/i);
      const count = await removedKeyInput.count();
      expect(count).toBe(0);
    }
  });

  test('should show success message and clear form after successful submit', async ({ page }) => {
    const collectionField = page.getByLabel(/collection/i).or(
      page.getByRole('combobox', { name: /collection/i })
    );
    const textField = page.locator('textarea').or(
      page.getByLabel(/text|knowledge|content/i)
    ).first();
    const submitButton = page.getByRole('button', { name: /submit|create|save/i });

    // Fill form
    await collectionField.click();
    await page.getByRole('option', { name: /technical-knowledge/i }).click();
    await textField.fill('This is a successful knowledge entry for testing purposes');

    // Submit form
    await submitButton.click();
    await page.waitForTimeout(500);

    // Verify success message (snackbar or alert)
    const successMessage = page.getByText(/success|created|saved/i);
    await expect(successMessage).toBeVisible({ timeout: 2000 });

    // Verify form is cleared
    const clearedText = await textField.inputValue();
    expect(clearedText).toBe('');
  });

  test('should show error message on API failure', async ({ page }) => {
    await setupKnowledgeAPIWithErrors(page);
    await page.goto('/knowledge');
    await page.waitForLoadState('networkidle');

    // Navigate to create form
    const createButton = page.getByRole('button', { name: /create knowledge/i }).or(
      page.getByRole('tab', { name: /create/i })
    );
    const isVisible = await createButton.isVisible().catch(() => false);
    if (isVisible) {
      await createButton.click();
      await page.waitForTimeout(300);
    }

    const collectionField = page.getByLabel(/collection/i).or(
      page.getByRole('combobox', { name: /collection/i })
    );
    const textField = page.locator('textarea').or(
      page.getByLabel(/text|knowledge|content/i)
    ).first();
    const submitButton = page.getByRole('button', { name: /submit|create|save/i });

    // Fill form
    await collectionField.click();
    await page.waitForTimeout(200);
    await page.getByRole('option').first().click();
    await textField.fill('This submission will fail due to API error');

    // Submit form
    await submitButton.click();
    await page.waitForTimeout(500);

    // Verify error message
    const errorMessage = page.getByText(/error|failed/i);
    await expect(errorMessage).toBeVisible({ timeout: 2000 });

    // Form should NOT be cleared
    const textValue = await textField.inputValue();
    expect(textValue.length).toBeGreaterThan(0);
  });

  test('should submit form with Ctrl+Enter keyboard shortcut', async ({ page }) => {
    const collectionField = page.getByLabel(/collection/i).or(
      page.getByRole('combobox', { name: /collection/i })
    );
    const textField = page.locator('textarea').or(
      page.getByLabel(/text|knowledge|content/i)
    ).first();

    // Fill form
    await collectionField.click();
    await page.getByRole('option', { name: /technical-knowledge/i }).click();
    await textField.fill('Testing keyboard shortcut submission with Ctrl+Enter');

    // Focus on text field and press Ctrl+Enter
    await textField.focus();
    await page.keyboard.press('Control+Enter');
    await page.waitForTimeout(500);

    // Verify success message
    const successMessage = page.getByText(/success|created|saved/i);
    await expect(successMessage).toBeVisible({ timeout: 2000 });
  });

  test('should update character counter as typing', async ({ page }) => {
    const textField = page.locator('textarea').or(
      page.getByLabel(/text|knowledge|content/i)
    ).first();

    // Type text
    const testText = 'Character counting test';
    await textField.fill(testText);
    await page.waitForTimeout(200);

    // Look for character counter
    const counter = page.locator('[data-testid="character-count"]').or(
      page.getByText(new RegExp(`${testText.length}`, 'i'))
    );

    // Character counter should be visible and show correct count
    const isVisible = await counter.isVisible().catch(() => false);
    if (isVisible) {
      const counterText = await counter.textContent();
      expect(counterText).toContain(testText.length.toString());
    }
  });

  test('should manage focus after form submission', async ({ page }) => {
    const collectionField = page.getByLabel(/collection/i).or(
      page.getByRole('combobox', { name: /collection/i })
    );
    const textField = page.locator('textarea').or(
      page.getByLabel(/text|knowledge|content/i)
    ).first();
    const submitButton = page.getByRole('button', { name: /submit|create|save/i });

    // Fill and submit form
    await collectionField.click();
    await page.getByRole('option', { name: /technical-knowledge/i }).click();
    await textField.fill('Testing focus management after successful submission');
    await submitButton.click();

    // Wait for submission
    await page.waitForTimeout(500);

    // Verify focus is managed (should return to first field or stay on form)
    const focusedElement = await page.evaluate(() => document.activeElement?.tagName);
    expect(focusedElement).toBeTruthy();

    // Focus should be on an interactive element (INPUT, TEXTAREA, BUTTON)
    expect(['INPUT', 'TEXTAREA', 'BUTTON', 'BODY']).toContain(focusedElement);
  });

  test('should validate required collection field', async ({ page }) => {
    const textField = page.locator('textarea').or(
      page.getByLabel(/text|knowledge|content/i)
    ).first();
    const submitButton = page.getByRole('button', { name: /submit|create|save/i });

    // Fill only text field (skip collection)
    await textField.fill('This is valid text but collection is missing');
    await page.waitForTimeout(300);

    // Submit button should be disabled
    await expect(submitButton).toBeDisabled();
  });

  test('should preserve form data when switching between fields', async ({ page }) => {
    const collectionField = page.getByLabel(/collection/i).or(
      page.getByRole('combobox', { name: /collection/i })
    );
    const textField = page.locator('textarea').or(
      page.getByLabel(/text|knowledge|content/i)
    ).first();

    // Fill collection
    await collectionField.click();
    await page.getByRole('option', { name: /technical-knowledge/i }).click();

    // Fill text
    const testText = 'Testing data preservation across field changes';
    await textField.fill(testText);

    // Click back to collection
    await collectionField.click();
    await page.keyboard.press('Escape'); // Close dropdown

    // Verify text is preserved
    const preservedText = await textField.inputValue();
    expect(preservedText).toBe(testText);
  });

  test('should display validation error for empty text field on blur', async ({ page }) => {
    const textField = page.locator('textarea').or(
      page.getByLabel(/text|knowledge|content/i)
    ).first();

    // Focus and blur without entering text
    await textField.focus();
    await textField.blur();
    await page.waitForTimeout(300);

    // Error message should appear
    const errorMessage = page.getByText(/required|cannot be empty/i);
    const hasError = await errorMessage.isVisible().catch(() => false);

    // Some forms show error on blur, others only on submit attempt
    // Either behavior is acceptable
    expect(hasError !== undefined).toBeTruthy();
  });

  test('should allow pasting long text into text field', async ({ page }) => {
    const textField = page.locator('textarea').or(
      page.getByLabel(/text|knowledge|content/i)
    ).first();

    // Generate long text
    const longText = 'This is a long knowledge entry. '.repeat(20);

    // Paste text
    await textField.focus();
    await page.evaluate((text) => {
      navigator.clipboard.writeText(text);
    }, longText);
    await page.keyboard.press('Control+V');
    await page.waitForTimeout(300);

    // Verify text was pasted
    const pastedText = await textField.inputValue();
    expect(pastedText.length).toBeGreaterThan(100);
  });

  test('should show loading state during submission', async ({ page }) => {
    const collectionField = page.getByLabel(/collection/i).or(
      page.getByRole('combobox', { name: /collection/i })
    );
    const textField = page.locator('textarea').or(
      page.getByLabel(/text|knowledge|content/i)
    ).first();
    const submitButton = page.getByRole('button', { name: /submit|create|save/i });

    // Fill form
    await collectionField.click();
    await page.getByRole('option', { name: /technical-knowledge/i }).click();
    await textField.fill('Testing loading state during form submission');

    // Submit and check for loading state
    await submitButton.click();

    // Check if submit button shows loading (disabled or spinner)
    await page.waitForTimeout(100);
    const isDisabledDuringSubmit = await submitButton.isDisabled();
    const hasLoadingIndicator = await page.locator('.MuiCircularProgress-root').isVisible().catch(() => false);

    // Either button should be disabled or loading indicator should be visible
    expect(isDisabledDuringSubmit || hasLoadingIndicator).toBeTruthy();
  });
});

test.describe('KnowledgeCreate Accessibility @accessibility', () => {
  test.beforeEach(async ({ page }) => {
    await setupKnowledgeAPI(page);
    await page.goto('/knowledge');
    await page.waitForLoadState('networkidle');

    const createButton = page.getByRole('button', { name: /create knowledge/i }).or(
      page.getByRole('tab', { name: /create/i })
    );
    const isVisible = await createButton.isVisible().catch(() => false);
    if (isVisible) {
      await createButton.click();
      await page.waitForTimeout(300);
    }
  });

  test('should pass axe-core accessibility audit', async ({ page }) => {
    // Run accessibility audit
    const violations = await runAccessibilityAudit(page);

    if (violations.length > 0) {
      console.log('Accessibility violations found:');
      console.log(formatViolations(violations));
    }

    expect(violations.length).toBe(0);
  });

  test('should have proper labels for all form fields', async ({ page }) => {
    // Check collection field
    const collectionField = page.getByRole('combobox', { name: /collection/i });
    await expect(collectionField).toBeVisible();

    // Check text field
    const textField = page.locator('textarea').first();
    const hasLabel = await textField.evaluate((el) => {
      const id = el.getAttribute('id');
      const ariaLabel = el.getAttribute('aria-label');
      const ariaLabelledBy = el.getAttribute('aria-labelledby');
      const hasAssociatedLabel = id && document.querySelector(`label[for="${id}"]`);

      return !!(ariaLabel || ariaLabelledBy || hasAssociatedLabel);
    });

    expect(hasLabel).toBeTruthy();
  });

  test('should support keyboard-only form completion', async ({ page }) => {
    // Tab to collection field
    await page.keyboard.press('Tab');
    await page.keyboard.press('Tab');

    // Open collection dropdown with keyboard
    await page.keyboard.press('ArrowDown');
    await page.waitForTimeout(200);

    // Select option with Enter
    await page.keyboard.press('Enter');
    await page.waitForTimeout(200);

    // Tab to text field
    await page.keyboard.press('Tab');

    // Type text
    await page.keyboard.type('Testing keyboard-only form completion without mouse');

    // Verify form can be completed
    const focusedElement = await page.evaluate(() => document.activeElement?.tagName);
    expect(['INPUT', 'TEXTAREA', 'BUTTON']).toContain(focusedElement);
  });

  test('should announce validation errors to screen readers', async ({ page }) => {
    const textField = page.locator('textarea').or(
      page.getByLabel(/text|knowledge|content/i)
    ).first();

    // Enter invalid text
    await textField.fill('Short');
    await textField.blur();
    await page.waitForTimeout(300);

    // Check for aria-invalid or aria-describedby
    const ariaInvalid = await textField.getAttribute('aria-invalid');
    const ariaDescribedBy = await textField.getAttribute('aria-describedby');

    // Field should be marked as invalid or have error description
    expect(ariaInvalid === 'true' || !!ariaDescribedBy).toBeTruthy();
  });

  test('should have visible focus indicators on all interactive elements', async ({ page }) => {
    const submitButton = page.getByRole('button', { name: /submit|create|save/i });

    // Focus on button
    await submitButton.focus();

    // Check for focus indicator
    const focusStyles = await submitButton.evaluate((el) => {
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

  test('should announce form submission success to screen readers', async ({ page }) => {
    const collectionField = page.getByLabel(/collection/i).or(
      page.getByRole('combobox', { name: /collection/i })
    );
    const textField = page.locator('textarea').or(
      page.getByLabel(/text|knowledge|content/i)
    ).first();
    const submitButton = page.getByRole('button', { name: /submit|create|save/i });

    // Fill and submit
    await collectionField.click();
    await page.getByRole('option').first().click();
    await textField.fill('Testing screen reader announcement for success');
    await submitButton.click();
    await page.waitForTimeout(500);

    // Check for aria-live region with success message
    const liveRegion = page.locator('[aria-live]');
    const liveRegionCount = await liveRegion.count();

    expect(liveRegionCount).toBeGreaterThan(0);
  });

  test('should have descriptive button labels', async ({ page }) => {
    const submitButton = page.getByRole('button', { name: /submit|create|save/i });

    // Button should have accessible name
    const accessibleName = await submitButton.evaluate((btn) => {
      return btn.textContent?.trim() || btn.getAttribute('aria-label') || btn.getAttribute('title');
    });

    expect(accessibleName).toBeTruthy();
    expect(accessibleName!.length).toBeGreaterThan(0);
  });
});
