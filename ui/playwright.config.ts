import { defineConfig, devices } from '@playwright/test';

const webPort = Number(process.env.PLAYWRIGHT_WEB_PORT || 7878);
const webURL = `http://localhost:${webPort}`;

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
    baseURL: webURL,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  // Auto-start unified hyper binary before running tests
  webServer: {
    command: 'cd .. && ./bin/hyper --mode=http --config=.env.hyper.hot',
    url: `${webURL}/health`,
    reuseExistingServer: !process.env.CI,
    timeout: 30000, // 30 seconds to start
    stdout: 'pipe',
    stderr: 'pipe',
  },
});
