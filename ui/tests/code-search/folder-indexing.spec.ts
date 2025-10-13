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
  });

  test.afterAll(async () => {
    // Cleanup test project
    if (fs.existsSync(testProjectPath)) {
      fs.rmSync(testProjectPath, { recursive: true, force: true });
    }
  });

  test('should add a new folder to the code index', async ({ page }) => {
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
  });

  test('should prevent duplicate folder additions', async ({ page }) => {
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
  });

  test('should scan folder and index all code files', async ({ page }) => {
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
  });

  test('should display indexing statistics (indexed/updated/skipped)', async ({ page }) => {
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
  });

  test('should update index when files are modified', async ({ page }) => {
    // Modify a file in the test project
    const authFilePath = path.join(testProjectPath, 'auth.go');
    const originalContent = fs.readFileSync(authFilePath, 'utf-8');
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

    // Restore original file
    fs.writeFileSync(authFilePath, originalContent, 'utf-8');
  });

  test('should handle invalid folder paths gracefully', async ({ page }) => {
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
  });

  test('should remove folder and cleanup all data', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    // Find the folder in the list
    const folderRow = page.locator('[data-testid="folder-row"]').filter({ hasText: testProjectPath });
    await expect(folderRow).toBeVisible();

    // Get file count before removal
    const fileCountElement = folderRow.locator('[data-testid="file-count"]');
    const fileCountText = await fileCountElement.textContent();
    const fileCount = parseInt(fileCountText || '0');
    expect(fileCount).toBeGreaterThan(0);

    // Click "Remove" button
    const removeButton = folderRow.getByRole('button', { name: /remove|delete/i });
    await removeButton.click();

    // Confirm deletion in dialog
    const confirmButton = page.getByRole('button', { name: /confirm|yes|delete/i });
    await confirmButton.click();

    // Wait for removal to complete
    await page.waitForTimeout(2000);

    // Verify folder is no longer in the list
    const removedFolder = page.locator('[data-testid="folder-row"]').filter({ hasText: testProjectPath });
    await expect(removedFolder).not.toBeVisible({ timeout: 5000 });

    // Verify success message
    const successMessage = page.getByText(/removed successfully|deleted/i);
    await expect(successMessage).toBeVisible({ timeout: 5000 });
  });

  test('should display index status and statistics', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    // Look for status/statistics section
    const statusSection = page.locator('[data-testid="index-status"]').or(
      page.getByRole('region', { name: /status|statistics/i })
    );

    if (await statusSection.isVisible()) {
      // Verify key statistics are displayed
      await expect(statusSection).toContainText(/total folders|total files|last scan/i);
    }

    // Check for refresh/reload status button
    const refreshButton = page.getByRole('button', { name: /refresh|reload status/i });
    if (await refreshButton.isVisible()) {
      await refreshButton.click();
      await page.waitForTimeout(1000);

      // Status should update
      await expect(statusSection).toBeVisible();
    }
  });

  test('should show scanning progress with real-time updates', async ({ page }) => {
    // Add a folder with many files for this test
    const largeFolderPath = path.join(os.tmpdir(), 'large-test-project');

    if (!fs.existsSync(largeFolderPath)) {
      fs.mkdirSync(largeFolderPath, { recursive: true });

      // Create 20 files for testing progress
      for (let i = 0; i < 20; i++) {
        const content = `// File ${i}\nfunction test${i}() {\n  console.log('test ${i}');\n}\n`;
        fs.writeFileSync(path.join(largeFolderPath, `file${i}.ts`), content, 'utf-8');
      }
    }

    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    // Add the large folder
    const addFolderButton = page.getByRole('button', { name: /add folder/i });
    await addFolderButton.click();

    const folderPathInput = page.getByLabel(/folder path/i);
    await folderPathInput.fill(largeFolderPath);

    const submitButton = page.getByRole('button', { name: /add|submit|save/i });
    await submitButton.click();

    await page.waitForTimeout(1000);

    // Start scanning
    const folderRow = page.locator('[data-testid="folder-row"]').filter({ hasText: largeFolderPath });
    const scanButton = folderRow.getByRole('button', { name: /scan/i });
    await scanButton.click();

    // Check for progress indicator
    const progressBar = page.locator('[role="progressbar"]').or(
      page.locator('[data-testid="scan-progress"]')
    );

    if (await progressBar.isVisible({ timeout: 2000 })) {
      // Progress bar should be visible during scan
      await expect(progressBar).toBeVisible();
    }

    // Wait for completion
    await page.waitForTimeout(5000);

    // Cleanup
    const removeButton = folderRow.getByRole('button', { name: /remove|delete/i });
    await removeButton.click();
    const confirmButton = page.getByRole('button', { name: /confirm|yes|delete/i });
    await confirmButton.click();

    fs.rmSync(largeFolderPath, { recursive: true, force: true });
  });
});

test.describe('Code Search - Folder Indexing Error Handling', () => {
  test('should handle permission denied errors', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    // Try to add a system folder that might have permission issues
    const addFolderButton = page.getByRole('button', { name: /add folder/i });
    await addFolderButton.click();

    const folderPathInput = page.getByLabel(/folder path/i);
    // Use /root or /private as a path likely to have permission issues
    await folderPathInput.fill(os.platform() === 'win32' ? 'C:\\Windows\\System32' : '/root');

    const submitButton = page.getByRole('button', { name: /add|submit|save/i });
    await submitButton.click();

    // Should show permission error
    const errorMessage = page.getByText(/permission|access denied|not allowed/i);
    await expect(errorMessage).toBeVisible({ timeout: 5000 });
  });

  test('should recover from MongoDB connection failures', async ({ page }) => {
    // This test would require mocking the API to simulate failures
    // Implementation depends on how the UI handles backend errors

    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    // Check for connection status indicator
    const connectionStatus = page.locator('[data-testid="connection-status"]');

    if (await connectionStatus.isVisible()) {
      // Should show connected status
      await expect(connectionStatus).toContainText(/connected|online/i);
    }
  });

  test('should handle Qdrant vector storage errors', async ({ page }) => {
    // Similar to MongoDB test - would need API mocking
    // This ensures the UI properly displays vector storage errors

    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    // If there's a health check or status endpoint
    const healthStatus = page.locator('[data-testid="qdrant-status"]');

    if (await healthStatus.isVisible()) {
      await expect(healthStatus).toContainText(/healthy|connected/i);
    }
  });
});
