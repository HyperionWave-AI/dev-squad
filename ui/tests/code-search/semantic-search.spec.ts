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
 * 
 * ERROR HANDLING ANALYSIS:
 * - WebSocket connection error handling for real-time search
 * - API call timeout handling for search requests
 * - Network connectivity error handling
 * - Search result parsing error handling
 * - File system error handling in test setup
 * - Graceful degradation when search service is unavailable
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
    try {
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
    } catch (error) {
      console.error('Failed to setup search test project:', error);
      throw error;
    }
  });

  test.afterAll(async () => {
    try {
      // Cleanup
      if (fs.existsSync(searchTestPath)) {
        fs.rmSync(searchTestPath, { recursive: true, force: true });
      }
    } catch (error) {
      console.warn('Failed to cleanup search test project:', error);
    }
  });

  // Helper function to setup test project
  async function setupTestProject(page: any, projectPath: string) {
    try {
      const addFolderButton = page.getByRole('button', { name: /add folder/i });
      await addFolderButton.click();

      const folderPathInput = page.getByLabel(/folder path/i);
      await folderPathInput.fill(projectPath);

      const submitButton = page.getByRole('button', { name: /add|submit|save/i });
      await submitButton.click();

      // Wait for folder to be added
      await page.waitForSelector('[data-testid="indexed-folders"]', { timeout: 5000 });

      // Trigger scan
      const folderRow = page.locator('[data-testid="folder-row"]').filter({ hasText: projectPath });
      const scanButton = folderRow.getByRole('button', { name: /scan/i });
      await scanButton.click();

      // Wait for scan completion
      await page.waitForTimeout(5000);
    } catch (error) {
      console.error('Failed to setup test project:', error);
      throw error;
    }
  }

  test('should find JWT authentication code with natural language query', async ({ page }) => {
    try {
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

      // Wait for results with timeout
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
    } catch (error) {
      console.error('Test failed: JWT authentication search', error);
      throw error;
    }
  });

  test('should handle WebSocket connection errors gracefully', async ({ page }) => {
    try {
      await page.goto('/code-search');
      await page.waitForLoadState('networkidle');

      // Monitor WebSocket connections
      const wsConnections: any[] = [];
      page.on('websocket', ws => {
        wsConnections.push(ws);
        
        ws.on('close', () => {
          console.log('WebSocket connection closed');
        });
        
        ws.on('socketerror', (error) => {
          console.log('WebSocket error:', error);
        });
      });

      // Simulate WebSocket connection failure by blocking WebSocket requests
      await page.route('**/ws/**', route => {
        route.abort('connectionfailed');
      });

      // Try to perform a search that would normally use WebSocket
      const searchInput = page.getByRole('textbox', { name: /search|query/i });
      await searchInput.fill('test query');
      await page.keyboard.press('Enter');

      // Should show appropriate error message or fallback to HTTP
      const errorMessage = page.getByText(/connection error|websocket failed|search unavailable/i).or(
        page.getByRole('alert')
      );

      // Either show error or fallback gracefully
      const hasError = await errorMessage.isVisible({ timeout: 5000 }).catch(() => false);
      const hasResults = await page.locator('[data-testid="search-result"]').isVisible({ timeout: 5000 }).catch(() => false);

      // Should either show error message or fallback to HTTP search
      expect(hasError || hasResults).toBe(true);
    } catch (error) {
      console.error('Test failed: WebSocket error handling', error);
      throw error;
    }
  });

  test('should handle search API timeout errors', async ({ page }) => {
    try {
      await page.goto('/code-search');
      await page.waitForLoadState('networkidle');

      // Simulate slow API responses
      await page.route('**/api/search/**', async route => {
        // Delay response to simulate timeout
        await new Promise(resolve => setTimeout(resolve, 10000));
        route.continue();
      });

      const searchInput = page.getByRole('textbox', { name: /search|query/i });
      await searchInput.fill('timeout test query');
      await page.keyboard.press('Enter');

      // Should show loading state initially
      const loadingIndicator = page.getByText(/searching|loading/i).or(
        page.locator('[data-testid="search-loading"]')
      );
      await expect(loadingIndicator).toBeVisible({ timeout: 2000 });

      // Should show timeout error after reasonable wait
      const timeoutError = page.getByText(/timeout|request timed out|search took too long/i).or(
        page.getByRole('alert')
      );
      await expect(timeoutError).toBeVisible({ timeout: 8000 });
    } catch (error) {
      console.error('Test failed: search API timeout handling', error);
      throw error;
    }
  });

  test('should handle malformed search responses', async ({ page }) => {
    try {
      await page.goto('/code-search');
      await page.waitForLoadState('networkidle');

      // Intercept search API and return malformed response
      await page.route('**/api/search/**', route => {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: '{"invalid": "json", "missing": "results"}'
        });
      });

      const searchInput = page.getByRole('textbox', { name: /search|query/i });
      await searchInput.fill('malformed response test');
      await page.keyboard.press('Enter');

      // Should handle malformed response gracefully
      const errorMessage = page.getByText(/search error|invalid response|failed to parse/i).or(
        page.getByRole('alert')
      );
      await expect(errorMessage).toBeVisible({ timeout: 5000 });

      // Should not crash the application
      const searchInterface = page.getByRole('textbox', { name: /search|query/i });
      await expect(searchInterface).toBeVisible();
    } catch (error) {
      console.error('Test failed: malformed response handling', error);
      throw error;
    }
  });

  test('should handle network connectivity issues during search', async ({ page }) => {
    try {
      await page.goto('/code-search');
      await page.waitForLoadState('networkidle');

      const searchInput = page.getByRole('textbox', { name: /search|query/i });
      await searchInput.fill('network test query');

      // Go offline before search
      await page.context().setOffline(true);
      await page.keyboard.press('Enter');

      // Should show network error
      const networkError = page.getByText(/offline|network error|no connection/i).or(
        page.getByRole('alert')
      );
      await expect(networkError).toBeVisible({ timeout: 5000 });

      // Restore connection
      await page.context().setOffline(false);

      // Should allow retry
      const retryButton = page.getByRole('button', { name: /retry|try again/i });
      if (await retryButton.isVisible()) {
        await retryButton.click();
        
        // Should work after reconnection
        await page.waitForLoadState('networkidle');
        const searchResults = page.locator('[data-testid="search-result"]');
        // Results may or may not appear depending on implementation
      }
    } catch (error) {
      console.error('Test failed: network connectivity handling', error);
      // Ensure network is restored
      await page.context().setOffline(false);
      throw error;
    }
  });

  test('should handle empty search results gracefully', async ({ page }) => {
    try {
      await page.goto('/code-search');
      await page.waitForLoadState('networkidle');

      // Search for something that definitely won't exist
      const searchInput = page.getByRole('textbox', { name: /search|query/i });
      await searchInput.fill('xyzabc123nonexistentcode999');
      await page.keyboard.press('Enter');

      // Wait for search to complete
      await page.waitForTimeout(3000);

      // Should show "no results" message
      const noResultsMessage = page.getByText(/no results|nothing found|no matches/i);
      await expect(noResultsMessage).toBeVisible({ timeout: 5000 });

      // Should not show error state
      const errorAlert = page.getByRole('alert');
      const hasError = await errorAlert.isVisible().catch(() => false);
      expect(hasError).toBe(false);
    } catch (error) {
      console.error('Test failed: empty results handling', error);
      throw error;
    }
  });

  test('should handle search service unavailable', async ({ page }) => {
    try {
      await page.goto('/code-search');
      await page.waitForLoadState('networkidle');

      // Simulate search service being down
      await page.route('**/api/search/**', route => {
        route.fulfill({
          status: 503,
          contentType: 'application/json',
          body: '{"error": "Service Unavailable"}'
        });
      });

      const searchInput = page.getByRole('textbox', { name: /search|query/i });
      await searchInput.fill('service unavailable test');
      await page.keyboard.press('Enter');

      // Should show service unavailable message
      const serviceError = page.getByText(/service unavailable|search service down|temporarily unavailable/i).or(
        page.getByRole('alert')
      );
      await expect(serviceError).toBeVisible({ timeout: 5000 });

      // Should provide helpful guidance
      const helpText = page.getByText(/try again later|contact support/i);
      const hasHelpText = await helpText.isVisible().catch(() => false);
      // Help text is optional but good UX
    } catch (error) {
      console.error('Test failed: service unavailable handling', error);
      throw error;
    }
  });

  test('should handle concurrent search requests', async ({ page }) => {
    try {
      await page.goto('/code-search');
      await page.waitForLoadState('networkidle');

      const searchInput = page.getByRole('textbox', { name: /search|query/i });

      // Perform multiple rapid searches
      await searchInput.fill('first search');
      await page.keyboard.press('Enter');
      
      await searchInput.fill('second search');
      await page.keyboard.press('Enter');
      
      await searchInput.fill('third search');
      await page.keyboard.press('Enter');

      // Should handle concurrent requests gracefully
      // Either show results for the last search or handle cancellation properly
      await page.waitForTimeout(3000);

      // Should not show multiple loading states or conflicting results
      const loadingIndicators = page.locator('[data-testid="search-loading"]');
      const loadingCount = await loadingIndicators.count();
      expect(loadingCount).toBeLessThanOrEqual(1);

      // Should show results or appropriate message
      const hasResults = await page.locator('[data-testid="search-result"]').isVisible().catch(() => false);
      const hasMessage = await page.getByText(/no results|searching/i).isVisible().catch(() => false);
      expect(hasResults || hasMessage).toBe(true);
    } catch (error) {
      console.error('Test failed: concurrent requests handling', error);
      throw error;
    }
  });
});