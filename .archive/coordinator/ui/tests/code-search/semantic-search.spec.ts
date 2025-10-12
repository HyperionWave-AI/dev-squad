/**
 * Code Search - Semantic Search Accuracy E2E Tests
 *
 * Test Suite: Semantic code search with natural language queries
 *
 * Coverage:
 * - Natural language query processing
 * - Semantic similarity scoring
 * - Result ranking by relevance
 * - Multi-language code search
 * - Context-aware search (file path, language filters)
 * - Search result quality and precision
 * - Edge cases (empty results, typos, ambiguous queries)
 */

import { test, expect } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';

// Test project setup
const SEARCH_TEST_PROJECT = 'search-accuracy-test';
let searchTestPath: string;

test.describe('Code Search - Semantic Search Accuracy', () => {
  test.beforeAll(async () => {
    // Create test project with diverse code samples
    searchTestPath = path.join(os.tmpdir(), SEARCH_TEST_PROJECT);

    if (!fs.existsSync(searchTestPath)) {
      fs.mkdirSync(searchTestPath, { recursive: true });
    }

    // Create code files with known patterns for testing search accuracy
    const codeFiles = {
      'authentication.go': `package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

// AuthService handles user authentication
type AuthService struct {
	secretKey string
}

// ValidateJWT validates JSON Web Tokens
func (a *AuthService) ValidateJWT(token string) (string, error) {
	if token == "" {
		return "", errors.New("token is empty")
	}

	// JWT validation logic
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", errors.New("invalid token format")
	}

	return "user-id", nil
}

// GenerateJWT creates a new JWT token for authenticated users
func (a *AuthService) GenerateJWT(userId string, expiration time.Duration) (string, error) {
	// Token generation with HMAC SHA256
	hash := sha256.New()
	hash.Write([]byte(userId + a.secretKey))
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// HashPassword hashes user passwords using SHA256
func HashPassword(password string) string {
	hash := sha256.New()
	hash.Write([]byte(password))
	return hex.EncodeToString(hash.Sum(nil))
}`,

      'export_handler.go': `package handlers

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CSVExportHandler streams large CSV exports efficiently
func CSVExportHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=export.csv")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"ID", "Name", "Email", "Created"})

	// Stream data in chunks
	for i := 0; i < 10000; i++ {
		record := []string{
			fmt.Sprintf("%d", i),
			fmt.Sprintf("User %d", i),
			fmt.Sprintf("user%d@example.com", i),
			time.Now().Format(time.RFC3339),
		}
		writer.Write(record)

		// Flush after every 100 records
		if i%100 == 0 {
			writer.Flush()
		}
	}
}

// JSONExportHandler exports data as JSON
func JSONExportHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// JSON export logic
}`,

      'react_components.tsx': `import React, { useState, useEffect } from 'react';
import { Card, Button, TextField } from '@mui/material';

interface UserFormProps {
  onSubmit: (email: string, password: string) => void;
  loading: boolean;
}

// LoginForm component with email and password validation
export const LoginForm: React.FC<UserFormProps> = ({ onSubmit, loading }) => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [errors, setErrors] = useState({ email: '', password: '' });

  const validateEmail = (email: string): boolean => {
    const emailRegex = /^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$/;
    return emailRegex.test(email);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    // Validate inputs
    const newErrors = { email: '', password: '' };

    if (!validateEmail(email)) {
      newErrors.email = 'Invalid email format';
    }

    if (password.length < 8) {
      newErrors.password = 'Password must be at least 8 characters';
    }

    if (newErrors.email || newErrors.password) {
      setErrors(newErrors);
      return;
    }

    onSubmit(email, password);
  };

  return (
    <Card>
      <form onSubmit={handleSubmit}>
        <TextField
          label="Email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          error={!!errors.email}
          helperText={errors.email}
        />
        <TextField
          label="Password"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          error={!!errors.password}
          helperText={errors.password}
        />
        <Button type="submit" disabled={loading}>
          Login
        </Button>
      </form>
    </Card>
  );
};

// DataTable component for displaying tabular data
export const DataTable: React.FC = () => {
  const [data, setData] = useState([]);

  useEffect(() => {
    fetch('/api/data')
      .then(res => res.json())
      .then(setData);
  }, []);

  return <div>{/* Table rendering */}</div>;
};`,

      'database.py': `"""
Database connection and query utilities
Provides connection pooling and query execution
"""

import psycopg2
from psycopg2 import pool
import logging

logger = logging.getLogger(__name__)


class DatabaseConnection:
    """PostgreSQL database connection with connection pooling"""

    def __init__(self, host, port, database, user, password):
        self.connection_pool = psycopg2.pool.SimpleConnectionPool(
            minconn=1,
            maxconn=10,
            host=host,
            port=port,
            database=database,
            user=user,
            password=password
        )

    def execute_query(self, query, params=None):
        """Execute a SQL query and return results"""
        conn = self.connection_pool.getconn()
        try:
            with conn.cursor() as cursor:
                cursor.execute(query, params)
                if cursor.description:
                    return cursor.fetchall()
                conn.commit()
                return None
        except Exception as e:
            conn.rollback()
            logger.error(f"Query execution failed: {e}")
            raise
        finally:
            self.connection_pool.putconn(conn)

    def execute_transaction(self, queries):
        """Execute multiple queries in a transaction"""
        conn = self.connection_pool.getconn()
        try:
            with conn.cursor() as cursor:
                for query, params in queries:
                    cursor.execute(query, params)
                conn.commit()
        except Exception as e:
            conn.rollback()
            logger.error(f"Transaction failed: {e}")
            raise
        finally:
            self.connection_pool.putconn(conn)


def create_connection(config):
    """Factory function to create database connection"""
    return DatabaseConnection(**config)
`
    };

    for (const [filename, content] of Object.entries(codeFiles)) {
      fs.writeFileSync(path.join(searchTestPath, filename), content, 'utf-8');
    }
  });

  test.afterAll(async () => {
    // Cleanup
    if (fs.existsSync(searchTestPath)) {
      fs.rmSync(searchTestPath, { recursive: true, force: true });
    }
  });

  test('should find JWT authentication code with natural language query', async ({ page }) => {
    // First, index the test project
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    // Add and scan folder
    await setupTestProject(page, searchTestPath);

    // Perform semantic search
    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    await expect(searchInput).toBeVisible();

    await searchInput.fill('JWT token validation and authentication');
    await page.keyboard.press('Enter');

    // Wait for results
    await page.waitForSelector('[data-testid="search-result"]', { timeout: 10000 });

    const results = page.locator('[data-testid="search-result"]');
    const resultCount = await results.count();

    expect(resultCount).toBeGreaterThan(0);

    // First result should be from authentication.go
    const firstResult = results.first();
    await expect(firstResult).toContainText(/authentication\.go|ValidateJWT|GenerateJWT/i);

    // Check result score (should be high relevance)
    const scoreElement = firstResult.locator('[data-testid="result-score"]');
    if (await scoreElement.isVisible()) {
      const scoreText = await scoreElement.textContent();
      const score = parseFloat(scoreText || '0');
      expect(score).toBeGreaterThan(0.7); // High relevance threshold
    }
  });

  test('should rank results by relevance (highest scores first)', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    // Search for CSV export functionality
    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    await searchInput.fill('CSV export with streaming for large datasets');
    await page.keyboard.press('Enter');

    await page.waitForSelector('[data-testid="search-result"]', { timeout: 10000 });

    const results = page.locator('[data-testid="search-result"]');
    const resultCount = await results.count();

    if (resultCount > 1) {
      // Extract scores from all results
      const scores: number[] = [];

      for (let i = 0; i < Math.min(resultCount, 5); i++) {
        const result = results.nth(i);
        const scoreElement = result.locator('[data-testid="result-score"]');

        if (await scoreElement.isVisible()) {
          const scoreText = await scoreElement.textContent();
          const score = parseFloat(scoreText || '0');
          scores.push(score);
        }
      }

      // Verify scores are in descending order (highest first)
      for (let i = 1; i < scores.length; i++) {
        expect(scores[i - 1]).toBeGreaterThanOrEqual(scores[i]);
      }
    }
  });

  test('should find React form validation code', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    await searchInput.fill('email validation in React login form');
    await page.keyboard.press('Enter');

    await page.waitForSelector('[data-testid="search-result"]', { timeout: 10000 });

    const results = page.locator('[data-testid="search-result"]');
    await expect(results.first()).toBeVisible();

    // Should find the React component file
    const firstResult = results.first();
    await expect(firstResult).toContainText(/react_components\.tsx|LoginForm|validateEmail/i);
  });

  test('should search across multiple programming languages', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    await searchInput.fill('password hashing and security');
    await page.keyboard.press('Enter');

    await page.waitForSelector('[data-testid="search-result"]', { timeout: 10000 });

    const results = page.locator('[data-testid="search-result"]');
    const resultCount = await results.count();

    // Should find results from Go (authentication.go has HashPassword)
    expect(resultCount).toBeGreaterThan(0);

    // Check if results include different languages
    const languages: Set<string> = new Set();

    for (let i = 0; i < Math.min(resultCount, 5); i++) {
      const result = results.nth(i);
      const langElement = result.locator('[data-testid="result-language"]');

      if (await langElement.isVisible()) {
        const lang = await langElement.textContent();
        if (lang) languages.add(lang.trim().toLowerCase());
      }
    }

    // Should find at least one language
    expect(languages.size).toBeGreaterThan(0);
  });

  test('should filter results by folder path', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    // Use folder filter if available
    const folderFilter = page.locator('[data-testid="folder-filter"]').or(
      page.getByLabel(/filter by folder/i)
    );

    if (await folderFilter.isVisible()) {
      await folderFilter.click();
      await page.getByRole('option', { name: new RegExp(SEARCH_TEST_PROJECT) }).click();
    }

    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    await searchInput.fill('database connection');
    await page.keyboard.press('Enter');

    await page.waitForSelector('[data-testid="search-result"]', { timeout: 10000 });

    const results = page.locator('[data-testid="search-result"]');

    // All results should be from the selected folder
    const resultCount = await results.count();

    for (let i = 0; i < resultCount; i++) {
      const result = results.nth(i);
      const filePathElement = result.locator('[data-testid="result-filepath"]');
      const filePath = await filePathElement.textContent();

      expect(filePath).toContain(SEARCH_TEST_PROJECT);
    }
  });

  test('should filter results by programming language', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    // Use language filter
    const langFilter = page.locator('[data-testid="language-filter"]').or(
      page.getByLabel(/filter by language/i)
    );

    if (await langFilter.isVisible()) {
      await langFilter.click();
      await page.getByRole('option', { name: /go/i }).click();
    }

    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    await searchInput.fill('error handling');
    await page.keyboard.press('Enter');

    await page.waitForSelector('[data-testid="search-result"]', { timeout: 10000 });

    const results = page.locator('[data-testid="search-result"]');
    const resultCount = await results.count();

    // All results should be Go files
    for (let i = 0; i < resultCount; i++) {
      const result = results.nth(i);
      const langElement = result.locator('[data-testid="result-language"]');
      const lang = await langElement.textContent();

      expect(lang?.toLowerCase()).toContain('go');
    }
  });

  test('should display code context with syntax highlighting', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    await searchInput.fill('CSV writer flush');
    await page.keyboard.press('Enter');

    await page.waitForSelector('[data-testid="search-result"]', { timeout: 10000 });

    const firstResult = page.locator('[data-testid="search-result"]').first();

    // Check for code snippet display
    const codeSnippet = firstResult.locator('[data-testid="code-snippet"]').or(
      firstResult.locator('pre').or(firstResult.locator('code'))
    );

    await expect(codeSnippet).toBeVisible();

    // Check for line numbers
    const lineNumbers = firstResult.locator('[data-testid="line-numbers"]');
    if (await lineNumbers.isVisible()) {
      const lineNumText = await lineNumbers.textContent();
      expect(lineNumText).toMatch(/\d+-\d+/); // Format like "15-30"
    }
  });

  test('should handle empty search results gracefully', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    await searchInput.fill('xyznonexistentcode123abcqueryterm');
    await page.keyboard.press('Enter');

    await page.waitForTimeout(3000);

    // Should show empty state
    const emptyState = page.getByText(/no results found|no matches/i);
    await expect(emptyState).toBeVisible();

    // Should not show any results
    const results = page.locator('[data-testid="search-result"]');
    await expect(results).toHaveCount(0);
  });

  test('should handle typos with semantic similarity', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    // Intentional typo: "athentication" instead of "authentication"
    await searchInput.fill('athentication tokn validashun');
    await page.keyboard.press('Enter');

    await page.waitForSelector('[data-testid="search-result"]', { timeout: 10000 });

    // Should still find authentication-related code due to semantic understanding
    const results = page.locator('[data-testid="search-result"]');
    const resultCount = await results.count();

    // Might have lower scores but should still find relevant code
    expect(resultCount).toBeGreaterThan(0);
  });

  test('should limit results and support pagination', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    await searchInput.fill('function'); // Generic query to get many results
    await page.keyboard.press('Enter');

    await page.waitForSelector('[data-testid="search-result"]', { timeout: 10000 });

    const results = page.locator('[data-testid="search-result"]');
    const resultCount = await results.count();

    // Should limit to a reasonable number (e.g., 10-20)
    expect(resultCount).toBeLessThanOrEqual(20);

    // Check for pagination controls
    const nextButton = page.getByRole('button', { name: /next|more results/i });
    if (await nextButton.isVisible()) {
      await nextButton.click();
      await page.waitForTimeout(2000);

      // Should load more results
      const newResults = page.locator('[data-testid="search-result"]');
      const newCount = await newResults.count();
      expect(newCount).toBeGreaterThan(0);
    }
  });

  test('should show relevant file metadata in results', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    await searchInput.fill('database connection pooling');
    await page.keyboard.press('Enter');

    await page.waitForSelector('[data-testid="search-result"]', { timeout: 10000 });

    const firstResult = page.locator('[data-testid="search-result"]').first();

    // Check for essential metadata
    const filePath = firstResult.locator('[data-testid="result-filepath"]');
    await expect(filePath).toBeVisible();

    const language = firstResult.locator('[data-testid="result-language"]');
    await expect(language).toBeVisible();

    const lineRange = firstResult.locator('[data-testid="result-lines"]');
    if (await lineRange.isVisible()) {
      const lineText = await lineRange.textContent();
      expect(lineText).toMatch(/lines?\s+\d+/i);
    }
  });
});

// Helper function to set up test project
async function setupTestProject(page: any, projectPath: string) {
  const addFolderButton = page.getByRole('button', { name: /add folder/i });
  await addFolderButton.click();

  const folderPathInput = page.getByLabel(/folder path/i);
  await folderPathInput.fill(projectPath);

  const submitButton = page.getByRole('button', { name: /add|submit|save/i });
  await submitButton.click();

  await page.waitForTimeout(1000);

  // Trigger scan
  const folderRow = page.locator('[data-testid="folder-row"]').filter({ hasText: projectPath });
  const scanButton = folderRow.getByRole('button', { name: /scan/i });
  await scanButton.click();

  // Wait for scan to complete
  await page.waitForTimeout(10000);
}
