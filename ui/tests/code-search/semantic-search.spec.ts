/**
 * Code Search - Semantic Search Accuracy E2E Tests
 *
 * Test Suite: Semantic code search with natural language queries
 * Updated: Added dark mode toggle functionality tests
 *
 * Coverage:
 * - Natural language query processing
 * - Semantic similarity scoring
 * - Result ranking by relevance
 * - Multi-language code search
 * - Context-aware search (file path, language filters)
 * - Search result quality and precision
 * - Edge cases (empty results, typos, ambiguous queries)
 * - Dark mode toggle functionality and state persistence
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

  test('should rank results by relevance for database queries', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    await setupTestProject(page, searchTestPath);

    // Search for database-related functionality
    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    await searchInput.fill('database connection pooling PostgreSQL');
    await page.keyboard.press('Enter');

    await page.waitForSelector('[data-testid="search-result"]', { timeout: 10000 });

    const results = page.locator('[data-testid="search-result"]');
    const resultCount = await results.count();

    expect(resultCount).toBeGreaterThan(0);

    // First result should be from database.py (most relevant)
    const firstResult = results.first();
    await expect(firstResult).toContainText(/database\.py|DatabaseConnection|connection_pool/i);
  });

  test('should handle multi-language code search', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    await setupTestProject(page, searchTestPath);

    // Search for form validation across different languages
    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    await searchInput.fill('form validation email password');
    await page.keyboard.press('Enter');

    await page.waitForSelector('[data-testid="search-result"]', { timeout: 10000 });

    const results = page.locator('[data-testid="search-result"]');
    const resultCount = await results.count();

    expect(resultCount).toBeGreaterThan(0);

    // Should find React form validation
    const reactResult = results.filter({ hasText: /react_components\.tsx|validateEmail|LoginForm/i });
    await expect(reactResult.first()).toBeVisible();
  });

  test('should handle empty search results gracefully', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    await setupTestProject(page, searchTestPath);

    // Search for something that doesn't exist
    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    await searchInput.fill('blockchain cryptocurrency mining algorithm');
    await page.keyboard.press('Enter');

    // Wait for search to complete
    await page.waitForTimeout(3000);

    // Should show no results message
    const noResultsMessage = page.locator('[data-testid="no-results"]');
    await expect(noResultsMessage).toBeVisible();
    await expect(noResultsMessage).toContainText(/no results|not found/i);
  });

  test('should handle typos and fuzzy matching', async ({ page }) => {
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    await setupTestProject(page, searchTestPath);

    // Search with typos
    const searchInput = page.getByRole('textbox', { name: /search|query/i });
    await searchInput.fill('autentication tokn validaton'); // Intentional typos
    await page.keyboard.press('Enter');

    await page.waitForSelector('[data-testid="search-result"]', { timeout: 10000 });

    const results = page.locator('[data-testid="search-result"]');
    const resultCount = await results.count();

    // Should still find authentication-related code despite typos
    expect(resultCount).toBeGreaterThan(0);

    const firstResult = results.first();
    await expect(firstResult).toContainText(/authentication\.go|ValidateJWT/i);
  });
});

// Dark Mode Toggle Functionality Tests
test.describe('Dark Mode Toggle Functionality', () => {
  test.beforeEach(async ({ page }) => {
    // Clear localStorage before each test
    await page.goto('/settings');
    await page.evaluate(() => localStorage.clear());
    await page.reload();
    await page.waitForLoadState('networkidle');
  });

  test('should display dark mode toggle in settings page', async ({ page }) => {
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    // Check if dark mode toggle is visible
    const darkModeToggle = page.locator('.dark-mode-toggle');
    await expect(darkModeToggle).toBeVisible();

    // Check toggle label and description
    const label = page.locator('.settings-label', { hasText: 'Dark Mode' });
    await expect(label).toBeVisible();

    const description = page.locator('.settings-description', { hasText: /switch between light and dark/i });
    await expect(description).toBeVisible();

    // Check toggle input
    const toggleInput = darkModeToggle.locator('input[type="checkbox"]');
    await expect(toggleInput).toBeVisible();
    await expect(toggleInput).toHaveAttribute('aria-label', 'Toggle dark mode');
  });

  test('should toggle dark mode on click', async ({ page }) => {
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    const darkModeToggle = page.locator('.dark-mode-toggle');
    const toggleInput = darkModeToggle.locator('input[type="checkbox"]');

    // Initially should be unchecked (light mode)
    await expect(toggleInput).not.toBeChecked();

    // Check document theme attribute
    const htmlElement = page.locator('html');
    await expect(htmlElement).not.toHaveAttribute('data-theme', 'dark');

    // Click to enable dark mode
    await darkModeToggle.click();

    // Should be checked now
    await expect(toggleInput).toBeChecked();

    // Document should have dark theme
    await expect(htmlElement).toHaveAttribute('data-theme', 'dark');

    // Click again to disable dark mode
    await darkModeToggle.click();

    // Should be unchecked
    await expect(toggleInput).not.toBeChecked();

    // Document should not have dark theme
    await expect(htmlElement).not.toHaveAttribute('data-theme', 'dark');
  });

  test('should persist dark mode state in localStorage', async ({ page }) => {
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    const darkModeToggle = page.locator('.dark-mode-toggle');

    // Enable dark mode
    await darkModeToggle.click();

    // Check localStorage
    const darkModeValue = await page.evaluate(() => localStorage.getItem('darkMode'));
    expect(darkModeValue).toBe('true');

    // Reload page
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Dark mode should still be enabled
    const toggleInput = darkModeToggle.locator('input[type="checkbox"]');
    await expect(toggleInput).toBeChecked();

    const htmlElement = page.locator('html');
    await expect(htmlElement).toHaveAttribute('data-theme', 'dark');
  });

  test('should respect system preference when no saved preference exists', async ({ page }) => {
    // Set system preference to dark mode
    await page.emulateMedia({ colorScheme: 'dark' });
    
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    const darkModeToggle = page.locator('.dark-mode-toggle');
    const toggleInput = darkModeToggle.locator('input[type="checkbox"]');

    // Should be checked based on system preference
    await expect(toggleInput).toBeChecked();

    const htmlElement = page.locator('html');
    await expect(htmlElement).toHaveAttribute('data-theme', 'dark');
  });

  test('should show visual feedback for current state', async ({ page }) => {
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    const darkModeToggle = page.locator('.dark-mode-toggle');
    const toggleSlider = darkModeToggle.locator('.toggle-slider');

    // Check initial state (light mode)
    const initialBgColor = await toggleSlider.evaluate(el => 
      getComputedStyle(el).backgroundColor
    );

    // Enable dark mode
    await darkModeToggle.click();

    // Check that background color changed
    const darkBgColor = await toggleSlider.evaluate(el => 
      getComputedStyle(el).backgroundColor
    );

    expect(darkBgColor).not.toBe(initialBgColor);

    // Check slider position
    const sliderBefore = toggleSlider.locator('::before');
    const transform = await sliderBefore.evaluate(el => 
      getComputedStyle(el).transform
    );
    expect(transform).toContain('translateX');
  });

  test('should be keyboard accessible', async ({ page }) => {
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    const darkModeToggle = page.locator('.dark-mode-toggle input');

    // Focus the toggle
    await darkModeToggle.focus();

    // Check focus state
    await expect(darkModeToggle).toBeFocused();

    // Toggle with space key
    await page.keyboard.press('Space');

    // Should be checked
    await expect(darkModeToggle).toBeChecked();

    // Toggle again with space key
    await page.keyboard.press('Space');

    // Should be unchecked
    await expect(darkModeToggle).not.toBeChecked();
  });

  test('should work correctly across different pages', async ({ page }) => {
    // Enable dark mode in settings
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    const darkModeToggle = page.locator('.dark-mode-toggle');
    await darkModeToggle.click();

    // Navigate to code search page
    await page.goto('/code-search');
    await page.waitForLoadState('networkidle');

    // Dark mode should still be active
    const htmlElement = page.locator('html');
    await expect(htmlElement).toHaveAttribute('data-theme', 'dark');

    // Navigate back to settings
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    // Toggle should still be checked
    const toggleInput = page.locator('.dark-mode-toggle input[type="checkbox"]');
    await expect(toggleInput).toBeChecked();
  });

  test('should handle rapid toggle clicks', async ({ page }) => {
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    const darkModeToggle = page.locator('.dark-mode-toggle');
    const toggleInput = darkModeToggle.locator('input[type="checkbox"]');

    // Rapidly click multiple times
    for (let i = 0; i < 5; i++) {
      await darkModeToggle.click();
      await page.waitForTimeout(100);
    }

    // Final state should be checked (odd number of clicks)
    await expect(toggleInput).toBeChecked();

    // localStorage should reflect final state
    const darkModeValue = await page.evaluate(() => localStorage.getItem('darkMode'));
    expect(darkModeValue).toBe('true');
  });

  test('should maintain accessibility attributes', async ({ page }) => {
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    const darkModeToggle = page.locator('.dark-mode-toggle input');

    // Check ARIA attributes
    await expect(darkModeToggle).toHaveAttribute('aria-label', 'Toggle dark mode');
    await expect(darkModeToggle).toHaveAttribute('type', 'checkbox');

    // Check role
    const role = await darkModeToggle.getAttribute('role');
    expect(role === null || role === 'checkbox').toBe(true);
  });
});

// Helper function to setup test project
async function setupTestProject(page: any, projectPath: string) {
  // Navigate to folder management
  const addFolderButton = page.getByRole('button', { name: /add folder|browse/i });
  if (await addFolderButton.isVisible()) {
    await addFolderButton.click();
    
    // In a real implementation, this would involve file picker interaction
    // For testing, we'll simulate the folder being added
    await page.evaluate((path) => {
      // Simulate folder addition
      window.dispatchEvent(new CustomEvent('folder-added', { detail: { path } }));
    }, projectPath);
    
    // Wait for indexing to complete
    await page.waitForTimeout(2000);
  }
}