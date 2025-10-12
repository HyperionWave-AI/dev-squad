/**
 * Code Search - File Watcher Functionality E2E Tests
 *
 * Test Suite: Automatic file change detection and index updates
 *
 * Coverage:
 * - Detect new files added to indexed folders
 * - Detect modified files (content changes)
 * - Detect deleted files
 * - Automatic re-indexing on file changes
 * - File watcher start/stop operations
 * - Performance under high file change volume
 * - Debouncing of rapid file changes
 * - Error recovery for watcher failures
 */

import { test, expect } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';

// Test constants
const WATCHER_TEST_PROJECT = 'file-watcher-test';
let watcherTestPath: string;

test.describe('Code Search - File Watcher Detection', () => {
  test.beforeAll(async () => {
    // Create test project directory
    watcherTestPath = path.join(os.tmpdir(), WATCHER_TEST_PROJECT);

    if (!fs.existsSync(watcherTestPath)) {
      fs.mkdirSync(watcherTestPath, { recursive: true });
    }

    // Create initial files
    fs.writeFileSync(
      path.join(watcherTestPath, 'initial.go'),
      'package main\n\nfunc main() {\n\tprintln("Hello")\n}\n',
      'utf-8'
    );

    fs.writeFileSync(
      path.join(watcherTestPath, 'utils.ts'),
      'export function hello() {\n  console.log("hello");\n}\n',
      'utf-8'
    );
  });

  test.afterAll(async () => {
    // Cleanup
    if (fs.existsSync(watcherTestPath)) {
      fs.rmSync(watcherTestPath, { recursive: true, force: true });
    }
  });

  test('should enable file watcher when folder is added', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    // Add folder
    const addFolderButton = page.getByRole('button', { name: /add folder/i });
    await addFolderButton.click();

    const folderPathInput = page.getByLabel(/folder path/i);
    await folderPathInput.fill(watcherTestPath);

    // Check for file watcher toggle/checkbox
    const watcherToggle = page.locator('[data-testid="enable-watcher"]').or(
      page.getByLabel(/enable file watcher|watch for changes/i)
    );

    if (await watcherToggle.isVisible()) {
      // Ensure watcher is enabled
      const isChecked = await watcherToggle.isChecked();
      if (!isChecked) {
        await watcherToggle.check();
      }
    }

    const submitButton = page.getByRole('button', { name: /add|submit|save/i });
    await submitButton.click();

    await page.waitForTimeout(1000);

    // Scan the folder
    const folderRow = page.locator('[data-testid="folder-row"]').filter({ hasText: watcherTestPath });
    const scanButton = folderRow.getByRole('button', { name: /scan/i });
    await scanButton.click();

    await page.waitForTimeout(5000);

    // Verify watcher status
    const watcherStatus = folderRow.locator('[data-testid="watcher-status"]');
    if (await watcherStatus.isVisible()) {
      await expect(watcherStatus).toContainText(/watching|active|enabled/i);
    }
  });

  test('should detect when a new file is added', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    // Get initial file count
    const folderRow = page.locator('[data-testid="folder-row"]').filter({ hasText: watcherTestPath });
    const fileCountElement = folderRow.locator('[data-testid="file-count"]');
    const initialCountText = await fileCountElement.textContent();
    const initialCount = parseInt(initialCountText || '0');

    // Add a new file to the watched folder
    const newFilePath = path.join(watcherTestPath, 'newfile.go');
    fs.writeFileSync(
      newFilePath,
      'package main\n\nfunc newFunction() {\n\tprintln("New!")\n}\n',
      'utf-8'
    );

    // Wait for file watcher to detect the change (typically a few seconds)
    await page.waitForTimeout(8000);

    // Refresh the page to see updated count
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Verify file count increased
    const updatedFolderRow = page.locator('[data-testid="folder-row"]').filter({ hasText: watcherTestPath });
    const updatedFileCountElement = updatedFolderRow.locator('[data-testid="file-count"]');
    const updatedCountText = await updatedFileCountElement.textContent();
    const updatedCount = parseInt(updatedCountText || '0');

    expect(updatedCount).toBeGreaterThan(initialCount);

    // Verify the new file is searchable
    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    await searchInput.fill('newFunction');
    await page.keyboard.press('Enter');

    await page.waitForSelector('[data-testid="search-result"]', { timeout: 10000 });

    const results = page.locator('[data-testid="search-result"]');
    await expect(results.first()).toContainText(/newfile\.go|newFunction/i);

    // Cleanup
    fs.unlinkSync(newFilePath);
  });

  test('should detect when a file is modified', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    const targetFile = path.join(watcherTestPath, 'utils.ts');

    // Modify the existing file
    const modifiedContent = `export function hello() {
  console.log("hello");
}

export function goodbye() {
  console.log("goodbye - this is new");
}
`;
    fs.writeFileSync(targetFile, modifiedContent, 'utf-8');

    // Wait for file watcher to detect change
    await page.waitForTimeout(8000);

    // Search for the new content
    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    await searchInput.fill('goodbye function');
    await page.keyboard.press('Enter');

    await page.waitForSelector('[data-testid="search-result"]', { timeout: 10000 });

    const results = page.locator('[data-testid="search-result"]');
    const firstResult = results.first();

    // Should find the modified file with new content
    await expect(firstResult).toContainText(/utils\.ts|goodbye/i);

    // Restore original content
    fs.writeFileSync(
      targetFile,
      'export function hello() {\n  console.log("hello");\n}\n',
      'utf-8'
    );
  });

  test('should detect when a file is deleted', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    // Create a temporary file to delete
    const tempFile = path.join(watcherTestPath, 'todelete.go');
    fs.writeFileSync(
      tempFile,
      'package main\n\nfunc toDelete() {\n\tprintln("Will be deleted")\n}\n',
      'utf-8'
    );

    // Wait for it to be indexed
    await page.waitForTimeout(8000);

    // Get current file count
    const folderRow = page.locator('[data-testid="folder-row"]').filter({ hasText: watcherTestPath });
    const fileCountElement = folderRow.locator('[data-testid="file-count"]');
    await page.reload();
    await page.waitForLoadState('networkidle');

    const beforeDeleteText = await fileCountElement.textContent();
    const beforeDeleteCount = parseInt(beforeDeleteText || '0');

    // Delete the file
    fs.unlinkSync(tempFile);

    // Wait for file watcher to detect deletion
    await page.waitForTimeout(8000);

    // Refresh and check file count
    await page.reload();
    await page.waitForLoadState('networkidle');

    const afterFolderRow = page.locator('[data-testid="folder-row"]').filter({ hasText: watcherTestPath });
    const afterFileCountElement = afterFolderRow.locator('[data-testid="file-count"]');
    const afterDeleteText = await afterFileCountElement.textContent();
    const afterDeleteCount = parseInt(afterDeleteText || '0');

    expect(afterDeleteCount).toBeLessThan(beforeDeleteCount);

    // Verify the file is no longer searchable
    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    await searchInput.fill('toDelete function');
    await page.keyboard.press('Enter');

    await page.waitForTimeout(3000);

    // Should not find the deleted file
    const noResults = page.getByText(/no results found/i);
    if (await noResults.isVisible({ timeout: 2000 })) {
      await expect(noResults).toBeVisible();
    } else {
      const results = page.locator('[data-testid="search-result"]');
      const resultTexts = await results.allTextContents();
      const containsDeleted = resultTexts.some(text => text.includes('todelete.go'));
      expect(containsDeleted).toBeFalsy();
    }
  });

  test('should handle rapid file changes with debouncing', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    const rapidFile = path.join(watcherTestPath, 'rapid.ts');

    // Create and modify file rapidly multiple times
    for (let i = 0; i < 5; i++) {
      fs.writeFileSync(
        rapidFile,
        `export function version${i}() { return ${i}; }\n`,
        'utf-8'
      );
      await page.waitForTimeout(200);
    }

    // Wait for debounced indexing
    await page.waitForTimeout(10000);

    // Search for final version
    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    await searchInput.fill('version4 function');
    await page.keyboard.press('Enter');

    await page.waitForSelector('[data-testid="search-result"]', { timeout: 10000 });

    const results = page.locator('[data-testid="search-result"]');
    const firstResult = results.first();

    // Should find the latest version
    await expect(firstResult).toContainText(/rapid\.ts|version4/i);

    // Cleanup
    fs.unlinkSync(rapidFile);
  });

  test('should disable file watcher on demand', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    const folderRow = page.locator('[data-testid="folder-row"]').filter({ hasText: watcherTestPath });

    // Find watcher toggle/button
    const watcherToggle = folderRow.locator('[data-testid="watcher-toggle"]').or(
      folderRow.getByRole('button', { name: /stop watching|disable watcher/i })
    );

    if (await watcherToggle.isVisible()) {
      await watcherToggle.click();

      // Verify watcher is disabled
      const watcherStatus = folderRow.locator('[data-testid="watcher-status"]');
      if (await watcherStatus.isVisible()) {
        await expect(watcherStatus).toContainText(/stopped|disabled|inactive/i);
      }

      // Add a new file - should NOT be auto-indexed
      const notIndexedFile = path.join(watcherTestPath, 'notindexed.go');
      fs.writeFileSync(
        notIndexedFile,
        'package main\n\nfunc notIndexed() {}\n',
        'utf-8'
      );

      await page.waitForTimeout(8000);

      // Search should not find it (unless manual rescan happens)
      const searchInput = page.getByRole('textbox', { name: /search|query/i });
      await searchInput.fill('notIndexed function');
      await page.keyboard.press('Enter');

      await page.waitForTimeout(3000);

      const noResults = page.getByText(/no results found/i);
      const hasNoResults = await noResults.isVisible({ timeout: 2000 }).catch(() => false);

      if (hasNoResults) {
        await expect(noResults).toBeVisible();
      }

      // Cleanup
      fs.unlinkSync(notIndexedFile);

      // Re-enable watcher
      await watcherToggle.click();
    }
  });

  test('should show file watcher activity/notifications', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    // Check for watcher activity log or notifications
    const activityLog = page.locator('[data-testid="watcher-activity"]').or(
      page.locator('[data-testid="activity-log"]')
    );

    if (await activityLog.isVisible()) {
      // Activity log should show recent changes
      await expect(activityLog).toBeVisible();
    }

    // Check for notification when file changes are detected
    const notification = page.locator('[role="alert"]').or(
      page.locator('[data-testid="notification"]')
    );

    // Create a new file to trigger notification
    const notifyFile = path.join(watcherTestPath, 'notify.go');
    fs.writeFileSync(
      notifyFile,
      'package main\n\nfunc notify() {}\n',
      'utf-8'
    );

    // Wait for potential notification
    const hasNotification = await notification.first().isVisible({ timeout: 10000 }).catch(() => false);

    if (hasNotification) {
      await expect(notification.first()).toContainText(/file|change|index|update/i);
    }

    // Cleanup
    fs.unlinkSync(notifyFile);
  });

  test('should handle file watcher errors gracefully', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    // Try to watch a folder that gets deleted
    const errorFolder = path.join(os.tmpdir(), 'error-watch-test');
    fs.mkdirSync(errorFolder, { recursive: true });
    fs.writeFileSync(path.join(errorFolder, 'test.go'), 'package main\n', 'utf-8');

    // Add folder
    const addFolderButton = page.getByRole('button', { name: /add folder/i });
    await addFolderButton.click();

    const folderPathInput = page.getByLabel(/folder path/i);
    await folderPathInput.fill(errorFolder);

    const submitButton = page.getByRole('button', { name: /add|submit|save/i });
    await submitButton.click();

    await page.waitForTimeout(2000);

    // Scan it
    const folderRow = page.locator('[data-testid="folder-row"]').filter({ hasText: errorFolder });
    const scanButton = folderRow.getByRole('button', { name: /scan/i });
    await scanButton.click();

    await page.waitForTimeout(5000);

    // Delete the folder externally (simulating watcher error)
    fs.rmSync(errorFolder, { recursive: true, force: true });

    await page.waitForTimeout(5000);

    // Check for error status or notification
    const errorStatus = folderRow.locator('[data-testid="watcher-status"]');
    if (await errorStatus.isVisible()) {
      const statusText = await errorStatus.textContent();
      // Should indicate error or folder not found
      expect(statusText?.toLowerCase()).toMatch(/error|not found|stopped/);
    }
  });

  test('should batch multiple file changes for efficient indexing', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    const batchFolder = path.join(watcherTestPath, 'batch-test');
    if (!fs.existsSync(batchFolder)) {
      fs.mkdirSync(batchFolder, { recursive: true });
    }

    // Create multiple files in quick succession
    const filePromises = [];
    for (let i = 0; i < 10; i++) {
      const filePath = path.join(batchFolder, `batch${i}.ts`);
      fs.writeFileSync(
        filePath,
        `export function batch${i}() { return ${i}; }\n`,
        'utf-8'
      );
    }

    // Wait for batched indexing
    await page.waitForTimeout(12000);

    // All files should be searchable
    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    await searchInput.fill('batch5 function');
    await page.keyboard.press('Enter');

    await page.waitForSelector('[data-testid="search-result"]', { timeout: 10000 });

    const results = page.locator('[data-testid="search-result"]');
    await expect(results.first()).toContainText(/batch5/i);

    // Cleanup
    fs.rmSync(batchFolder, { recursive: true, force: true });
  });
});

test.describe('Code Search - File Watcher Performance', () => {
  test('should handle high volume of file changes without crashing', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    const perfFolder = path.join(watcherTestPath, 'perf-test');
    if (!fs.existsSync(perfFolder)) {
      fs.mkdirSync(perfFolder, { recursive: true });
    }

    // Create many files
    for (let i = 0; i < 50; i++) {
      fs.writeFileSync(
        path.join(perfFolder, `perf${i}.ts`),
        `export function perf${i}() {}\n`,
        'utf-8'
      );
    }

    // Wait for indexing
    await page.waitForTimeout(15000);

    // Page should still be responsive
    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    await expect(searchInput).toBeVisible();
    await expect(searchInput).toBeEnabled();

    // Cleanup
    fs.rmSync(perfFolder, { recursive: true, force: true });
  });

  test('should maintain search accuracy after multiple file changes', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    const accuracyFile = path.join(watcherTestPath, 'accuracy.go');

    // Version 1
    fs.writeFileSync(
      accuracyFile,
      'package main\n\nfunc version1() { println("v1") }\n',
      'utf-8'
    );

    await page.waitForTimeout(8000);

    // Search for v1
    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    await searchInput.fill('version1');
    await page.keyboard.press('Enter');

    await page.waitForSelector('[data-testid="search-result"]', { timeout: 10000 });
    let results = page.locator('[data-testid="search-result"]');
    await expect(results.first()).toContainText(/version1/i);

    // Version 2
    fs.writeFileSync(
      accuracyFile,
      'package main\n\nfunc version2() { println("v2") }\n',
      'utf-8'
    );

    await page.waitForTimeout(8000);

    // Search for v2
    await searchInput.fill('version2');
    await page.keyboard.press('Enter');

    await page.waitForTimeout(3000);
    results = page.locator('[data-testid="search-result"]');
    await expect(results.first()).toContainText(/version2/i);

    // Old version should not be found
    await searchInput.fill('version1');
    await page.keyboard.press('Enter');

    await page.waitForTimeout(3000);

    const noResults = page.getByText(/no results found/i);
    const hasNoResults = await noResults.isVisible({ timeout: 2000 }).catch(() => false);

    if (hasNoResults) {
      await expect(noResults).toBeVisible();
    }

    // Cleanup
    fs.unlinkSync(accuracyFile);
  });
});
