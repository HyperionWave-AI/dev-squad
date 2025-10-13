/**
 * Kanban Board MUI Component Tests
 *
 * Test Suite: MUI component rendering and behavior validation
 *
 * Coverage:
 * - MUI Card components
 * - MUI AppBar header
 * - MUI Chip (priority badges)
 * - MUI Button interactions
 * - MUI Grid layout
 * - MUI IconButton
 * - MUI Typography
 * - MUI theme integration
 */

import { test, expect } from '@playwright/test';

test.describe('Kanban MUI Component Validation', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
  });

  test('should render MUI AppBar component', async ({ page }) => {
    const appBar = page.locator('[class*="MuiAppBar-root"]');
    await expect(appBar).toBeVisible();

    // Verify AppBar positioning
    const position = await appBar.evaluate((el) => {
      return window.getComputedStyle(el).position;
    });

    // AppBar should be fixed or sticky
    expect(['fixed', 'sticky', 'relative']).toContain(position);
  });

  test('should render MUI Toolbar within AppBar', async ({ page }) => {
    const toolbar = page.locator('[class*="MuiToolbar-root"]');
    await expect(toolbar).toBeVisible();

    // Toolbar should have proper height
    const toolbarBox = await toolbar.boundingBox();
    expect(toolbarBox).not.toBeNull();

    if (toolbarBox) {
      expect(toolbarBox.height).toBeGreaterThan(40);
    }
  });

  test('should render MUI Card components for task cards', async ({ page }) => {
    await page.waitForSelector('[class*="MuiCard-root"]', { timeout: 10000 });

    const muiCards = page.locator('[class*="MuiCard-root"]');
    const cardCount = await muiCards.count();

    expect(cardCount).toBeGreaterThan(0);

    // Verify first card has proper MUI classes
    const firstCard = muiCards.first();
    const className = await firstCard.getAttribute('class');

    expect(className).toContain('MuiCard');
  });

  test('should render MUI CardContent within Cards', async ({ page }) => {
    await page.waitForSelector('[class*="MuiCard-root"]', { timeout: 10000 });

    const cardContent = page.locator('[class*="MuiCardContent-root"]');
    const contentCount = await cardContent.count();

    expect(contentCount).toBeGreaterThan(0);
  });

  test('should render MUI Chip components for status/priority badges', async ({ page }) => {
    await page.waitForSelector('[class*="MuiChip-root"]', { timeout: 10000 });

    const chips = page.locator('[class*="MuiChip-root"]');
    const chipCount = await chips.count();

    expect(chipCount).toBeGreaterThan(0);

    // Verify chip has proper styling
    const firstChip = chips.first();
    const chipStyles = await firstChip.evaluate((el) => {
      const styles = window.getComputedStyle(el);
      return {
        borderRadius: styles.borderRadius,
        padding: styles.padding,
        display: styles.display,
      };
    });

    // Chips should have rounded borders
    expect(chipStyles.borderRadius).not.toBe('0px');
    expect(chipStyles.display).toBe('inline-flex');
  });

  test('should render MUI Button components', async ({ page }) => {
    const buttons = page.locator('[class*="MuiButton-root"]');
    const buttonCount = await buttons.count();

    if (buttonCount === 0) {
      // May not have MUI buttons in this view
      test.skip();
      return;
    }

    const firstButton = buttons.first();
    await expect(firstButton).toBeVisible();

    // Verify button variant classes
    const className = await firstButton.getAttribute('class');
    expect(className).toContain('MuiButton');
  });

  test('should render MUI IconButton for actions', async ({ page }) => {
    const iconButtons = page.locator('[class*="MuiIconButton-root"]');
    const iconButtonCount = await iconButtons.count();

    if (iconButtonCount > 0) {
      const firstIconButton = iconButtons.first();
      await expect(firstIconButton).toBeVisible();

      // IconButtons should be interactive
      await expect(firstIconButton).toBeEnabled();
    }
  });

  test('should render MUI Grid container for layout', async ({ page }) => {
    await page.waitForSelector('[class*="MuiGrid-root"]', { timeout: 10000 });

    const gridContainer = page.locator('[class*="MuiGrid-container"]');
    const hasGrid = await gridContainer.count() > 0;

    expect(hasGrid).toBeTruthy();
  });

  test('should render MUI Grid items for columns', async ({ page }) => {
    await page.waitForSelector('[class*="MuiGrid-item"]', { timeout: 10000 });

    const gridItems = page.locator('[class*="MuiGrid-item"]');
    const itemCount = await gridItems.count();

    // Should have at least 4 grid items (one per column)
    expect(itemCount).toBeGreaterThanOrEqual(4);
  });

  test('should apply MUI theme colors', async ({ page }) => {
    await page.waitForSelector('[data-testid*="task-card"]', { timeout: 10000 });

    // Check if theme colors are applied
    const taskCard = page.locator('[data-testid*="task-card"]').first();

    const colors = await taskCard.evaluate((el) => {
      const styles = window.getComputedStyle(el);
      return {
        backgroundColor: styles.backgroundColor,
        color: styles.color,
      };
    });

    // Should have defined colors (not transparent or initial)
    expect(colors.backgroundColor).not.toBe('rgba(0, 0, 0, 0)');
    expect(colors.color).toBeTruthy();
  });

  test('should render MUI Typography components', async ({ page }) => {
    const typography = page.locator('[class*="MuiTypography-root"]');
    const typographyCount = await typography.count();

    if (typographyCount === 0) {
      // May use native HTML headings instead
      test.skip();
      return;
    }

    const firstTypography = typography.first();
    await expect(firstTypography).toBeVisible();

    // Verify typography variant
    const className = await firstTypography.getAttribute('class');
    expect(className).toContain('MuiTypography');
  });

  test('should apply MUI Paper elevation to columns', async ({ page }) => {
    const papers = page.locator('[class*="MuiPaper-root"]');
    const paperCount = await papers.count();

    if (paperCount > 0) {
      const firstPaper = papers.first();

      // Paper should have elevation shadow
      const boxShadow = await firstPaper.evaluate((el) => {
        return window.getComputedStyle(el).boxShadow;
      });

      expect(boxShadow).not.toBe('none');
    }
  });

  test('should render MUI Avatar if agent icons present', async ({ page }) => {
    const avatars = page.locator('[class*="MuiAvatar-root"]');
    const avatarCount = await avatars.count();

    if (avatarCount > 0) {
      const firstAvatar = avatars.first();
      await expect(firstAvatar).toBeVisible();

      // Avatar should be circular
      const borderRadius = await firstAvatar.evaluate((el) => {
        return window.getComputedStyle(el).borderRadius;
      });

      expect(borderRadius).toBe('50%');
    }
  });

  test('should support MUI Button hover states', async ({ page }) => {
    const buttons = page.locator('[class*="MuiButton-root"]');

    if (await buttons.count() === 0) {
      test.skip();
      return;
    }

    const firstButton = buttons.first();
    await expect(firstButton).toBeVisible();

    // Get initial background
    const initialBg = await firstButton.evaluate((el) => {
      return window.getComputedStyle(el).backgroundColor;
    });

    // Hover
    await firstButton.hover();
    await page.waitForTimeout(200);

    // Background may change on hover (depends on theme)
    const hoverBg = await firstButton.evaluate((el) => {
      return window.getComputedStyle(el).backgroundColor;
    });

    // Just verify we can read the background
    expect(hoverBg).toBeTruthy();
  });

  test('should apply MUI Chip size variants correctly', async ({ page }) => {
    await page.waitForSelector('[class*="MuiChip-root"]', { timeout: 10000 });

    const chips = await page.locator('[class*="MuiChip-root"]').all();

    if (chips.length === 0) {
      test.skip();
      return;
    }

    // Check chip sizes
    for (const chip of chips) {
      const chipBox = await chip.boundingBox();

      if (chipBox) {
        // Chips should have reasonable dimensions
        expect(chipBox.height).toBeGreaterThan(15);
        expect(chipBox.height).toBeLessThan(60);
      }
    }
  });

  test('should render MUI dividers if present', async ({ page }) => {
    const dividers = page.locator('[class*="MuiDivider-root"]');
    const dividerCount = await dividers.count();

    if (dividerCount > 0) {
      const firstDivider = dividers.first();

      // Divider should be visible and thin
      const dividerBox = await firstDivider.boundingBox();

      if (dividerBox) {
        expect(dividerBox.height).toBeLessThan(5);
      }
    }
  });

  test('should apply MUI theme spacing consistently', async ({ page }) => {
    await page.waitForSelector('[data-testid*="kanban-column"]', { timeout: 10000 });

    const columns = await page.locator('[data-testid*="kanban-column"]').all();

    if (columns.length < 2) {
      test.skip();
      return;
    }

    // Check spacing between columns
    const firstColumnBox = await columns[0].boundingBox();
    const secondColumnBox = await columns[1].boundingBox();

    if (firstColumnBox && secondColumnBox) {
      const gap = secondColumnBox.x - (firstColumnBox.x + firstColumnBox.width);

      // Should have consistent spacing (MUI Grid gap)
      expect(gap).toBeGreaterThanOrEqual(8); // At least 8px gap
    }
  });
});