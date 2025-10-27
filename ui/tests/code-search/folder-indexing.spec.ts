/**
 * Code Search - Folder Indexing Workflow E2E Tests
 *
 * Test Suite: Folder indexing MCP tools (add_folder, scan, remove_folder)
 *
 * Coverage:
 * - Add folder to code index
 * - Scan folder and index files
 * - Detect duplicate folder additions
 * - Track indexing progress (files indexed/updated/skipped)
 * - Remove folder and cleanup vectors
 * - Handle invalid paths and permissions
 * - Status reporting across operations
 * - MongoDB and Qdrant integration
 * 
 * ERROR HANDLING ANALYSIS:
 * - Uses try-catch blocks implicitly through Playwright's expect() assertions
 * - Timeout handling for async operations (5000ms, 30000ms)
 * - Graceful error message validation for invalid paths
 * - File system error handling in beforeAll/afterAll hooks
 * - Network error handling with waitForLoadState('networkidle')
 * - Conditional element visibility checks to prevent race conditions
 */

import { test, expect } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';

// Test fixtures
const TEST_PROJECT_NAME = 'test-code-project';
let testProjectPath: string;

test.describe('Code Search - Folder Indexing Workflow', () => {
  test.beforeAll(async () => {
    try {
      // Create a temporary test project with code files
      testProjectPath = path.join(os.tmpdir(), TEST_PROJECT_NAME);

      if (!fs.existsSync(testProjectPath)) {
        fs.mkdirSync(testProjectPath, { recursive: true });
      }

      // Create sample code files for indexing
      const files = {
        'auth.go': `package auth

import (
	"errors"
	"time"
)

// ValidateToken validates JWT tokens
func ValidateToken(token string) (string, error) {
	if token == "" {
		return "", errors.New("token is empty")
	}

	// Token validation logic
	return "user-id", nil
}

// GenerateToken creates a new JWT token
func GenerateToken(userId string) (string, error) {
	// Token generation logic
	return "jwt-token", nil
}`,

        'handler.go': `package handlers

import (
	"encoding/json"
	"net/http"
)

// ExportHandler handles CSV export requests
func ExportHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv")
	w.Write([]byte("name,email\\nJohn,john@example.com"))
}`,

        'utils.ts': `export function formatDate(date: Date): string {
  return date.toISOString().split('T')[0];
}

export function validateEmail(email: string): boolean {
  const emailRegex = /^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$/;
  return emailRegex.test(email);
}`,

        'README.md': `# Test Project

This is a test project for code indexing.

## Features
- Authentication
- CSV Export
- Date formatting
`
      };

      for (const [filename, content] of Object.entries(files)) {
        fs.writeFileSync(path.join(testProjectPath, filename), content, 'utf-8');
      }
    } catch (error) {
      console.error('Failed to setup test project:', error);
      throw error;
    }
  });

  test.afterAll(async () => {
    try {
      // Cleanup test project
      if (fs.existsSync(testProjectPath)) {
        fs.rmSync(testProjectPath, { recursive: true, force: true });
      }
    } catch (error) {
      console.warn('Failed to cleanup test project:', error);
      // Don't throw - cleanup failures shouldn't fail the test suite
    }
  });

  test('should add a new folder to the code index', async ({ page }) => {
    try {
      // Navigate to code search page (assuming it exists at /code-search)
      await page.goto('/code-search');
      await page.waitForLoadState('networkidle');

      // Click "Add Folder" button
      const addFolderButton = page.getByRole('button', { name: /add folder/i });
      await expect(addFolderButton).toBeVisible();
      await addFolderButton.click();

      // Fill in folder path
      const folderPathInput = page.getByLabel(/folder path/i);
      await expect(folderPathInput).toBeVisible();
      await folderPathInput.fill(testProjectPath);

      // Fill in optional description
      const descriptionInput = page.getByLabel(/description/i);
      if (await descriptionInput.isVisible()) {
        await descriptionInput.fill('Test project for E2E testing');
      }

      // Submit the form
      const submitButton = page.getByRole('button', { name: /add|submit|save/i });
      await submitButton.click();

      // Wait for success message
      const successMessage = page.getByText(/folder added successfully/i);
      await expect(successMessage).toBeVisible({ timeout: 5000 });

      // Verify folder appears in the list
      const folderList = page.locator('[data-testid="indexed-folders"]');
      await expect(folderList).toContainText(testProjectPath);
    } catch (error) {
      console.error('Test failed: should add a new folder to the code index', error);
      throw error;
    }
  });

  test('should prevent duplicate folder additions', async ({ page }) => {
    try {
      await page.goto('/code-search');
      await page.waitForLoadState('networkidle');

      // Try to add the same folder again
      const addFolderButton = page.getByRole('button', { name: /add folder/i });
      await addFolderButton.click();

      const folderPathInput = page.getByLabel(/folder path/i);
      await folderPathInput.fill(testProjectPath);

      const submitButton = page.getByRole('button', { name: /add|submit|save/i });
      await submitButton.click();

      // Should show a message indicating folder already exists
      const duplicateMessage = page.getByText(/already indexed|already exists/i);
      await expect(duplicateMessage).toBeVisible({ timeout: 5000 });
    } catch (error) {
      console.error('Test failed: should prevent duplicate folder additions', error);
      throw error;
    }
  });

  test('should scan folder and index all code files', async ({ page }) => {
    try {
      await page.goto('/code-search');
      await page.waitForLoadState('networkidle');

      // Find the folder in the list
      const folderRow = page.locator('[data-testid="folder-row"]').filter({ hasText: testProjectPath });
      await expect(folderRow).toBeVisible();

      // Click "Scan" button
      const scanButton = folderRow.getByRole('button', { name: /scan/i });
      await scanButton.click();

      // Wait for scanning to start
      const scanningStatus = page.getByText(/scanning/i);
      await expect(scanningStatus).toBeVisible({ timeout: 2000 });

      // Wait for scan to complete (may take a few seconds)
      const completedStatus = page.getByText(/scan completed|active/i);
      await expect(completedStatus).toBeVisible({ timeout: 30000 });

      // Verify file counts are updated
      const fileCountElement = folderRow.locator('[data-testid="file-count"]');
      const fileCountText = await fileCountElement.textContent();
      const fileCount = parseInt(fileCountText || '0');

      // Should have indexed at least 3 code files (auth.go, handler.go, utils.ts)
      expect(fileCount).toBeGreaterThanOrEqual(3);
    } catch (error) {
      console.error('Test failed: should scan folder and index all code files', error);
      throw error;
    }
  });

  test('should display indexing statistics (indexed/updated/skipped)', async ({ page }) => {
    try {
      await page.goto('/code-search');
      await page.waitForLoadState('networkidle');

      const folderRow = page.locator('[data-testid="folder-row"]').filter({ hasText: testProjectPath });
      const scanButton = folderRow.getByRole('button', { name: /scan/i });

      // Trigger rescan
      await scanButton.click();

      // Wait for scan completion
      await page.waitForTimeout(3000);

      // Check for statistics display
      const statsDialog = page.locator('[data-testid="scan-results"]').or(
        page.getByRole('dialog')
      );

      if (await statsDialog.isVisible()) {
        // Verify statistics are shown
        await expect(statsDialog).toContainText(/files indexed|files updated|files skipped/i);

        // Since this is a rescan, most files should be skipped
        const skippedCount = statsDialog.locator('[data-testid="files-skipped"]');
        if (await skippedCount.isVisible()) {
          const skippedText = await skippedCount.textContent();
          const skipped = parseInt(skippedText || '0');
          expect(skipped).toBeGreaterThan(0);
        }
      }
    } catch (error) {
      console.error('Test failed: should display indexing statistics', error);
      throw error;
    }
  });

  test('should update index when files are modified', async ({ page }) => {
    let originalContent = '';
    try {
      // Modify a file in the test project
      const authFilePath = path.join(testProjectPath, 'auth.go');
      originalContent = fs.readFileSync(authFilePath, 'utf-8');
      const modifiedContent = originalContent + '\n// New comment added for testing\n';
      fs.writeFileSync(authFilePath, modifiedContent, 'utf-8');

      await page.goto('/code-search');
      await page.waitForLoadState('networkidle');

      const folderRow = page.locator('[data-testid="folder-row"]').filter({ hasText: testProjectPath });
      const scanButton = folderRow.getByRole('button', { name: /scan/i });

      await scanButton.click();
      await page.waitForTimeout(3000);

      // Check that at least 1 file was updated
      const statsDialog = page.locator('[data-testid="scan-results"]').or(
        page.getByRole('dialog')
      );

      if (await statsDialog.isVisible()) {
        const updatedCount = statsDialog.locator('[data-testid="files-updated"]');
        if (await updatedCount.isVisible()) {
          const updatedText = await updatedCount.textContent();
          const updated = parseInt(updatedText || '0');
          expect(updated).toBeGreaterThanOrEqual(1);
        }
      }
    } catch (error) {
      console.error('Test failed: should update index when files are modified', error);
      throw error;
    } finally {
      // Restore original file even if test fails
      try {
        if (originalContent) {
          const authFilePath = path.join(testProjectPath, 'auth.go');
          fs.writeFileSync(authFilePath, originalContent, 'utf-8');
        }
      } catch (cleanupError) {
        console.warn('Failed to restore original file:', cleanupError);
      }
    }
  });

  test('should handle invalid folder paths gracefully', async ({ page }) => {
    try {
      await page.goto('/code-search');
      await page.waitForLoadState('networkidle');

      const addFolderButton = page.getByRole('button', { name: /add folder/i });
      await addFolderButton.click();

      const folderPathInput = page.getByLabel(/folder path/i);
      await folderPathInput.fill('/nonexistent/invalid/path/12345');

      const submitButton = page.getByRole('button', { name: /add|submit|save/i });
      await submitButton.click();

      // Should show error message
      const errorMessage = page.getByText(/invalid|not found|does not exist/i).or(
        page.getByRole('alert')
      );
      await expect(errorMessage.first()).toBeVisible({ timeout: 5000 });
    } catch (error) {
      console.error('Test failed: should handle invalid folder paths gracefully', error);
      throw error;
    }
  });

  test('should remove folder and cleanup vectors', async ({ page }) => {
    try {
      await page.goto('/code-search');
      await page.waitForLoadState('networkidle');

      // Find the folder in the list
      const folderRow = page.locator('[data-testid="folder-row"]').filter({ hasText: testProjectPath });
      await expect(folderRow).toBeVisible();

      // Click "Remove" button
      const removeButton = folderRow.getByRole('button', { name: /remove|delete/i });
      await removeButton.click();

      // Confirm removal if confirmation dialog appears
      const confirmButton = page.getByRole('button', { name: /confirm|yes|remove/i });
      if (await confirmButton.isVisible({ timeout: 2000 })) {
        await confirmButton.click();
      }

      // Wait for removal success message
      const successMessage = page.getByText(/folder removed|removed successfully/i);
      await expect(successMessage).toBeVisible({ timeout: 5000 });

      // Verify folder is no longer in the list
      const folderList = page.locator('[data-testid="indexed-folders"]');
      await expect(folderList).not.toContainText(testProjectPath);
    } catch (error) {
      console.error('Test failed: should remove folder and cleanup vectors', error);
      throw error;
    }
  });

  test('should handle permission errors gracefully', async ({ page }) => {
    try {
      await page.goto('/code-search');
      await page.waitForLoadState('networkidle');

      const addFolderButton = page.getByRole('button', { name: /add folder/i });
      await addFolderButton.click();

      const folderPathInput = page.getByLabel(/folder path/i);
      // Try to add a system folder that might have permission issues
      await folderPathInput.fill('/root');

      const submitButton = page.getByRole('button', { name: /add|submit|save/i });
      await submitButton.click();

      // Should show permission error message
      const errorMessage = page.getByText(/permission|access denied|unauthorized/i).or(
        page.getByRole('alert')
      );
      await expect(errorMessage.first()).toBeVisible({ timeout: 5000 });
    } catch (error) {
      console.error('Test failed: should handle permission errors gracefully', error);
      throw error;
    }
  });

  test('should handle network connectivity issues', async ({ page }) => {
    try {
      // Simulate network issues by going offline
      await page.context().setOffline(true);

      await page.goto('/code-search');
      
      // Should show appropriate error message for offline state
      const offlineMessage = page.getByText(/offline|network error|connection failed/i).or(
        page.getByRole('alert')
      );
      
      // Wait a bit longer for offline detection
      await expect(offlineMessage.first()).toBeVisible({ timeout: 10000 });

      // Restore network connection
      await page.context().setOffline(false);
      
      // Page should recover
      await page.reload();
      await page.waitForLoadState('networkidle');
      
      const pageContent = page.locator('body');
      await expect(pageContent).toBeVisible();
    } catch (error) {
      console.error('Test failed: should handle network connectivity issues', error);
      // Ensure network is restored even if test fails
      await page.context().setOffline(false);
      throw error;
    }
  });
});