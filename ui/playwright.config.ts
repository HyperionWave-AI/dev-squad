import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright configuration for Code Search UI e2e testing
 * Supports chromium, webkit, mobile, tablet, and desktop viewports
 */
export default defineConfig({
  testDir: './tests',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [
    ['html', { outputFolder: 'test-reports/playwright-html' }],
    ['json', { outputFile: 'test-reports/playwright-results.json' }],
    ['list']
  ],

  use: {
    baseURL: 'http://localhost:5173/ui',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },

    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },

    {
      name: 'mobile',
      use: { ...devices['iPhone 13'] },
    },

    {
      name: 'tablet',
      use: { ...devices['iPad Pro'] },
    },

    {
      name: 'desktop',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1920, height: 1080 }
      },
    },
  ],

  webServer: {
    command: 'cd /home/avshall/work/dev-squad/ui && npm run dev',
    url: 'http://localhost:5173/ui',
    reuseExistingServer: !process.env.CI,
    timeout: 120000,
  },
});
