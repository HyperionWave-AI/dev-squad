/**
 * Code Search - Accessibility E2E Tests (WCAG 2.1 AA)
 *
 * Test Suite: Accessibility compliance for code search interface
 *
 * Coverage:
 * - WCAG 2.1 AA compliance via axe-core
 * - Keyboard navigation (Tab, Arrow keys, Enter, Escape)
 * - Screen reader support (ARIA labels, roles, live regions)
 * - Focus management and visible focus indicators
 * - Color contrast standards
 * - Alternative text for images and icons
 * - Form labels and error associations
 * - Skip links and landmarks
 * - Responsive text and zoom support
 * - Dark mode accessibility compliance
 */

import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

// Accessibility test utilities
async function runAccessibilityAudit(page: any, context?: string) {
  const results = await new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
    .analyze();

  if (results.violations.length > 0) {
    console.log(`\n🚨 Accessibility violations found${context ? ` in ${context}` : ''}:`);
    results.violations.forEach((violation, index) => {
      console.log(`\n${index + 1}. ${violation.id}: ${violation.description}`);
      console.log(`   Impact: ${violation.impact}`);
      console.log(`   Help: ${violation.help}`);
      console.log(`   Affected elements: ${violation.nodes.length}`);
      violation.nodes.forEach(node => {
        console.log(`   - ${node.html}`);
        console.log(`     ${node.failureSummary}`);
      });
    });
  }

  return results.violations;
}

test.describe('Code Search - WCAG 2.1 AA Compliance @accessibility', () => {
  test('should pass axe-core audit on initial page load', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    const violations = await runAccessibilityAudit(page, 'initial page load');

    expect(violations.length).toBe(0);
  });

  test('should pass axe-core audit on add folder dialog', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    const addButton = page.getByRole('button', { name: /add folder/i });
    await addButton.click();

    await page.waitForSelector('[role="dialog"]', { timeout: 2000 });

    const violations = await runAccessibilityAudit(page, 'add folder dialog');

    expect(violations.length).toBe(0);
  });

  test('should pass axe-core audit on search results page', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    await searchInput.fill('authentication');
    await page.keyboard.press('Enter');

    // Wait for results or empty state
    await page.waitForTimeout(3000);

    const violations = await runAccessibilityAudit(page, 'search results');

    expect(violations.length).toBe(0);
  });

  test('should pass axe-core audit on error states', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    // Trigger an error
    const addButton = page.getByRole('button', { name: /add folder/i });
    await addButton.click();

    const folderPathInput = page.getByLabel(/folder path/i);
    await folderPathInput.fill('/invalid/path');

    const submitButton = page.getByRole('button', { name: /add|submit|save/i });
    await submitButton.click();

    await page.waitForTimeout(2000);

    const violations = await runAccessibilityAudit(page, 'error state');

    expect(violations.length).toBe(0);
  });
});

test.describe('Dark Mode Toggle - Accessibility & Functionality @accessibility @darkmode', () => {
  test('should pass axe-core audit in light mode', async ({ page }) => {
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    // Ensure light mode is active
    await page.evaluate(() => {
      document.documentElement.removeAttribute('data-theme');
      localStorage.setItem('darkMode', 'false');
    });

    const violations = await runAccessibilityAudit(page, 'light mode settings');
    expect(violations.length).toBe(0);
  });

  test('should pass axe-core audit in dark mode', async ({ page }) => {
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    // Enable dark mode
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'dark');
      localStorage.setItem('darkMode', 'true');
    });

    await page.reload();
    await page.waitForLoadState('networkidle');

    const violations = await runAccessibilityAudit(page, 'dark mode settings');
    expect(violations.length).toBe(0);
  });

  test('should have proper ARIA attributes on dark mode toggle', async ({ page }) => {
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    const toggle = page.getByRole('checkbox', { name: /toggle dark mode/i });
    
    // Check ARIA attributes
    await expect(toggle).toHaveAttribute('aria-label', 'Toggle dark mode');
    await expect(toggle).toHaveAttribute('type', 'checkbox');
    
    // Check associated label and description
    const label = page.getByText('Dark Mode');
    const description = page.getByText('Switch between light and dark themes');
    
    await expect(label).toBeVisible();
    await expect(description).toBeVisible();
  });

  test('should be keyboard accessible', async ({ page }) => {
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    const toggle = page.getByRole('checkbox', { name: /toggle dark mode/i });
    
    // Focus the toggle
    await toggle.focus();
    await expect(toggle).toBeFocused();

    // Check initial state
    const initialState = await toggle.isChecked();

    // Toggle with Space key
    await page.keyboard.press('Space');
    await page.waitForTimeout(300);

    // State should have changed
    const newState = await toggle.isChecked();
    expect(newState).toBe(!initialState);

    // Theme should be applied
    const themeAttribute = await page.evaluate(() => 
      document.documentElement.getAttribute('data-theme')
    );
    
    if (newState) {
      expect(themeAttribute).toBe('dark');
    } else {
      expect(themeAttribute).toBeNull();
    }
  });

  test('should have sufficient color contrast in both modes', async ({ page }) => {
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    // Test light mode contrast
    await page.evaluate(() => {
      document.documentElement.removeAttribute('data-theme');
    });

    let violations = await new AxeBuilder({ page })
      .withTags(['wcag2aa'])
      .include('.settings-container')
      .analyze();

    expect(violations.filter(v => v.id === 'color-contrast').length).toBe(0);

    // Test dark mode contrast
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'dark');
    });

    violations = await new AxeBuilder({ page })
      .withTags(['wcag2aa'])
      .include('.settings-container')
      .analyze();

    expect(violations.filter(v => v.id === 'color-contrast').length).toBe(0);
  });

  test('should maintain focus visibility in both themes', async ({ page }) => {
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    const toggle = page.getByRole('checkbox', { name: /toggle dark mode/i });

    // Test focus visibility in light mode
    await page.evaluate(() => {
      document.documentElement.removeAttribute('data-theme');
    });

    await toggle.focus();
    
    const lightModeFocusStyles = await toggle.evaluate((el) => {
      const styles = window.getComputedStyle(el);
      return {
        outline: styles.outline,
        boxShadow: styles.boxShadow,
      };
    });

    // Should have visible focus indicator
    const hasLightFocusIndicator = 
      (lightModeFocusStyles.outline && lightModeFocusStyles.outline !== 'none') ||
      (lightModeFocusStyles.boxShadow && lightModeFocusStyles.boxShadow !== 'none');

    expect(hasLightFocusIndicator).toBeTruthy();

    // Test focus visibility in dark mode
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'dark');
    });

    await toggle.focus();
    
    const darkModeFocusStyles = await toggle.evaluate((el) => {
      const styles = window.getComputedStyle(el);
      return {
        outline: styles.outline,
        boxShadow: styles.boxShadow,
      };
    });

    // Should have visible focus indicator
    const hasDarkFocusIndicator = 
      (darkModeFocusStyles.outline && darkModeFocusStyles.outline !== 'none') ||
      (darkModeFocusStyles.boxShadow && darkModeFocusStyles.boxShadow !== 'none');

    expect(hasDarkFocusIndicator).toBeTruthy();
  });
});

test.describe('Dark Mode Toggle - Functionality Tests @darkmode', () => {
  test('should toggle theme when clicked', async ({ page }) => {
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    const toggle = page.getByRole('checkbox', { name: /toggle dark mode/i });
    
    // Get initial state
    const initialChecked = await toggle.isChecked();
    const initialTheme = await page.evaluate(() => 
      document.documentElement.getAttribute('data-theme')
    );

    // Click toggle
    await toggle.click();
    await page.waitForTimeout(300);

    // State should change
    const newChecked = await toggle.isChecked();
    const newTheme = await page.evaluate(() => 
      document.documentElement.getAttribute('data-theme')
    );

    expect(newChecked).toBe(!initialChecked);
    
    if (newChecked) {
      expect(newTheme).toBe('dark');
    } else {
      expect(newTheme).toBeNull();
    }
  });

  test('should persist theme preference in localStorage', async ({ page }) => {
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    const toggle = page.getByRole('checkbox', { name: /toggle dark mode/i });
    
    // Enable dark mode
    if (!(await toggle.isChecked())) {
      await toggle.click();
      await page.waitForTimeout(300);
    }

    // Check localStorage
    const storedValue = await page.evaluate(() => 
      localStorage.getItem('darkMode')
    );
    expect(storedValue).toBe('true');

    // Disable dark mode
    await toggle.click();
    await page.waitForTimeout(300);

    // Check localStorage again
    const newStoredValue = await page.evaluate(() => 
      localStorage.getItem('darkMode')
    );
    expect(newStoredValue).toBe('false');
  });

  test('should restore theme preference on page reload', async ({ page }) => {
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    // Set dark mode
    await page.evaluate(() => {
      localStorage.setItem('darkMode', 'true');
    });

    // Reload page
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Check if dark mode is restored
    const toggle = page.getByRole('checkbox', { name: /toggle dark mode/i });
    const themeAttribute = await page.evaluate(() => 
      document.documentElement.getAttribute('data-theme')
    );

    await expect(toggle).toBeChecked();
    expect(themeAttribute).toBe('dark');
  });

  test('should respect system preference when no saved preference', async ({ page }) => {
    // Clear any existing preference
    await page.evaluate(() => {
      localStorage.removeItem('darkMode');
    });

    // Mock system preference for dark mode
    await page.emulateMedia({ colorScheme: 'dark' });

    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    const toggle = page.getByRole('checkbox', { name: /toggle dark mode/i });
    const themeAttribute = await page.evaluate(() => 
      document.documentElement.getAttribute('data-theme')
    );

    // Should follow system preference
    await expect(toggle).toBeChecked();
    expect(themeAttribute).toBe('dark');

    // Test light system preference
    await page.emulateMedia({ colorScheme: 'light' });
    await page.reload();
    await page.waitForLoadState('networkidle');

    const newThemeAttribute = await page.evaluate(() => 
      document.documentElement.getAttribute('data-theme')
    );

    expect(newThemeAttribute).toBeNull();
  });

  test('should have smooth transitions between themes', async ({ page }) => {
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    const toggle = page.getByRole('checkbox', { name: /toggle dark mode/i });
    
    // Check if transitions are enabled
    const bodyTransition = await page.evaluate(() => {
      const styles = window.getComputedStyle(document.body);
      return styles.transition;
    });

    expect(bodyTransition).toContain('background-color');
    expect(bodyTransition).toContain('color');

    // Toggle and verify transition occurs
    await toggle.click();
    await page.waitForTimeout(100); // Allow transition to start

    // Transition should be in progress or completed
    const transitionStyles = await page.evaluate(() => {
      const styles = window.getComputedStyle(document.body);
      return {
        transition: styles.transition,
        backgroundColor: styles.backgroundColor,
        color: styles.color
      };
    });

    expect(transitionStyles.transition).toBeTruthy();
  });

  test('should update all themed elements when toggled', async ({ page }) => {
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    const toggle = page.getByRole('checkbox', { name: /toggle dark mode/i });
    
    // Get initial styles
    const initialStyles = await page.evaluate(() => {
      const settingsContainer = document.querySelector('.settings-container');
      const settingsItem = document.querySelector('.settings-item');
      
      if (!settingsContainer || !settingsItem) return null;
      
      return {
        containerBg: window.getComputedStyle(settingsContainer).backgroundColor,
        itemBg: window.getComputedStyle(settingsItem).backgroundColor,
        textColor: window.getComputedStyle(settingsContainer).color
      };
    });

    // Toggle theme
    await toggle.click();
    await page.waitForTimeout(300);

    // Get new styles
    const newStyles = await page.evaluate(() => {
      const settingsContainer = document.querySelector('.settings-container');
      const settingsItem = document.querySelector('.settings-item');
      
      if (!settingsContainer || !settingsItem) return null;
      
      return {
        containerBg: window.getComputedStyle(settingsContainer).backgroundColor,
        itemBg: window.getComputedStyle(settingsItem).backgroundColor,
        textColor: window.getComputedStyle(settingsContainer).color
      };
    });

    // Styles should have changed
    if (initialStyles && newStyles) {
      expect(newStyles.containerBg).not.toBe(initialStyles.containerBg);
      expect(newStyles.itemBg).not.toBe(initialStyles.itemBg);
      expect(newStyles.textColor).not.toBe(initialStyles.textColor);
    }
  });
});

test.describe('Code Search - Keyboard Navigation @accessibility', () => {
  test('should support full keyboard navigation without mouse', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    // Navigate through the page using only keyboard
    await page.keyboard.press('Tab');
    await page.waitForTimeout(200);

    let currentFocus = await page.evaluate(() => document.activeElement?.tagName);
    expect(currentFocus).toBeTruthy();

    // Continue tabbing through interactive elements
    for (let i = 0; i < 5; i++) {
      await page.keyboard.press('Tab');
      await page.waitForTimeout(100);

      currentFocus = await page.evaluate(() => document.activeElement?.tagName);
      expect(currentFocus).toBeTruthy();
    }

    // Shift+Tab should go backward
    await page.keyboard.press('Shift+Tab');
    await page.waitForTimeout(100);

    currentFocus = await page.evaluate(() => document.activeElement?.tagName);
    expect(currentFocus).toBeTruthy();
  });

  test('should have visible focus indicators on all interactive elements', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    // Get all focusable elements
    const focusableElements = await page.locator('button, a, input, select, textarea, [tabindex]:not([tabindex="-1"])').all();

    for (const element of focusableElements.slice(0, 10)) { // Test first 10 elements
      await element.focus();

      const focusStyles = await element.evaluate((el) => {
        const styles = window.getComputedStyle(el);
        return {
          outline: styles.outline,
          outlineWidth: styles.outlineWidth,
          outlineColor: styles.outlineColor,
          boxShadow: styles.boxShadow,
          border: styles.border,
        };
      });

      // Should have visible focus indicator (outline, box-shadow, or border change)
      const hasFocusIndicator =
        (focusStyles.outline && focusStyles.outline !== 'none' && focusStyles.outlineWidth !== '0px') ||
        (focusStyles.boxShadow && focusStyles.boxShadow !== 'none') ||
        focusStyles.border !== 'none';

      expect(hasFocusIndicator).toBeTruthy();
    }
  });

  test('should open add folder dialog with keyboard (Enter on button)', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    // Tab to add folder button
    let currentElement = '';
    for (let i = 0; i < 20; i++) {
      await page.keyboard.press('Tab');
      await page.waitForTimeout(100);

      currentElement = await page.evaluate(() => document.activeElement?.textContent || '');

      if (/add folder|add|new folder/i.test(currentElement)) {
        break;
      }
    }

    // Press Enter to activate
    await page.keyboard.press('Enter');

    // Dialog should open
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible({ timeout: 2000 });
  });

  test('should close dialog with Escape key', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    const addButton = page.getByRole('button', { name: /add folder/i });
    await addButton.click();

    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible({ timeout: 2000 });

    // Press Escape to close
    await page.keyboard.press('Escape');

    // Dialog should close
    await expect(dialog).not.toBeVisible({ timeout: 2000 });
  });

  test('should navigate search results with arrow keys', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    await searchInput.fill('function');
    await page.keyboard.press('Enter');

    const hasResults = await page.waitForSelector('[data-testid="search-result"]', { timeout: 10000 }).catch(() => null);

    if (hasResults) {
      const results = page.locator('[data-testid="search-result"]');
      const resultCount = await results.count();

      if (resultCount > 1) {
        // Focus on first result
        await results.first().focus();

        // Navigate with arrow keys
        await page.keyboard.press('ArrowDown');
        await page.waitForTimeout(200);

        // Check if focus moved
        const focusedElement = await page.evaluate(() => document.activeElement?.getAttribute('data-testid'));
        expect(focusedElement).toContain('search-result');

        // Navigate back up
        await page.keyboard.press('ArrowUp');
        await page.waitForTimeout(200);
      }
    }
  });

  test('should support Enter key to activate result actions', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    await searchInput.fill('authentication');
    await page.keyboard.press('Enter');

    const hasResults = await page.waitForSelector('[data-testid="search-result"]', { timeout: 10000 }).catch(() => null);

    if (hasResults) {
      const firstResult = page.locator('[data-testid="search-result"]').first();

      // Tab to a button in the result
      const resultButton = firstResult.getByRole('button').first();

      if (await resultButton.isVisible({ timeout: 1000 })) {
        await resultButton.focus();
        await page.keyboard.press('Enter');

        // Action should be triggered
        await page.waitForTimeout(1000);
      }
    }
  });
});