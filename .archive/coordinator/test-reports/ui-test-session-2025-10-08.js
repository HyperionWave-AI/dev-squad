/**
 * UI Test Session: Hyperion Coordinator UI
 * Date: 2025-10-08
 * Browser: Chromium
 * Viewport: Desktop (1440x900)
 */

const { chromium } = require('playwright');
const fs = require('fs');
const path = require('path');

const SCREENSHOTS_DIR = path.join(__dirname, 'screenshots-2025-10-08');

// Ensure screenshots directory exists
if (!fs.existsSync(SCREENSHOTS_DIR)) {
  fs.mkdirSync(SCREENSHOTS_DIR, { recursive: true });
}

async function runTests() {
  const browser = await chromium.launch({ headless: false });
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 }
  });
  const page = await context.newPage();

  const results = {
    testSession: 'Hyperion Coordinator UI - 2025-10-08',
    environment: 'Local Development (http://localhost:7777)',
    viewport: '1440x900 (Desktop)',
    browser: 'Chromium',
    tests: []
  };

  try {
    // Test 1: Navigation and Initial Load
    console.log('\n=== Test 1: Navigation and Initial Load ===');
    const test1 = {
      name: 'Navigation and Initial Load',
      status: 'PASS',
      issues: []
    };

    await page.goto('http://localhost:7777', { waitUntil: 'networkidle' });

    // Wait a bit for content to render
    await page.waitForTimeout(2000);

    // Check if page title/header is present
    const header = await page.locator('text=/Hyperion/i').first();
    if (await header.count() === 0) {
      test1.status = 'FAIL';
      test1.issues.push('Header "Hyperion Coordinator" not found');
    } else {
      console.log('✅ Header found');
    }

    // Check for Dashboard button
    const dashboardBtn = await page.locator('text=/Dashboard/i').first();
    if (await dashboardBtn.count() === 0) {
      test1.status = 'FAIL';
      test1.issues.push('Dashboard button not found');
    } else {
      console.log('✅ Dashboard button found');
    }

    // Check for Knowledge button
    const knowledgeBtn = await page.locator('text=/Knowledge/i').first();
    if (await knowledgeBtn.count() === 0) {
      test1.status = 'FAIL';
      test1.issues.push('Knowledge button not found');
    } else {
      console.log('✅ Knowledge button found');
    }

    await page.screenshot({
      path: path.join(SCREENSHOTS_DIR, '01-initial-load.png'),
      fullPage: true
    });
    console.log('📸 Screenshot saved: 01-initial-load.png');

    results.tests.push(test1);

    // Test 2: Dashboard View
    console.log('\n=== Test 2: Dashboard View ===');
    const test2 = {
      name: 'Dashboard View - Task List',
      status: 'PASS',
      issues: []
    };

    // Click Dashboard if not already active
    await dashboardBtn.click();
    await page.waitForTimeout(2000);

    // Check for task cards or task list
    const taskElements = await page.locator('[class*="task"], [class*="card"], [data-testid*="task"]').count();
    if (taskElements === 0) {
      // Try to find any content that indicates tasks loaded
      const pageContent = await page.content();
      if (pageContent.includes('task') || pageContent.includes('Task')) {
        console.log('✅ Task-related content found');
      } else {
        test2.issues.push('No task elements visible - tasks may not be loading');
        console.log('⚠️ No obvious task elements found');
      }
    } else {
      console.log(`✅ Found ${taskElements} task-related elements`);
    }

    // Check for refresh button
    const refreshBtn = await page.locator('text=/🔄|Refresh/i').first();
    if (await refreshBtn.count() === 0) {
      test2.issues.push('Refresh button not found in header');
      console.log('⚠️ Refresh button not found');
    } else {
      console.log('✅ Refresh button found');
    }

    await page.screenshot({
      path: path.join(SCREENSHOTS_DIR, '02-dashboard-view.png'),
      fullPage: true
    });
    console.log('📸 Screenshot saved: 02-dashboard-view.png');

    results.tests.push(test2);

    // Test 3: Knowledge Browser
    console.log('\n=== Test 3: Knowledge Browser ===');
    const test3 = {
      name: 'Knowledge Browser - Navigation and Search',
      status: 'PASS',
      issues: []
    };

    // Click Knowledge button
    await knowledgeBtn.click();
    await page.waitForTimeout(2000);

    // Check for "Quick Start" section
    const quickStart = await page.locator('text=/Quick Start/i').first();
    if (await quickStart.count() === 0) {
      test3.status = 'FAIL';
      test3.issues.push('Quick Start section not found');
      console.log('❌ Quick Start section not found');
    } else {
      console.log('✅ Quick Start section found');
    }

    await page.screenshot({
      path: path.join(SCREENSHOTS_DIR, '03-knowledge-browser-initial.png'),
      fullPage: true
    });
    console.log('📸 Screenshot saved: 03-knowledge-browser-initial.png');

    // Try to find and click an example search button
    const exampleButtons = await page.locator('button:has-text("JWT"), button:has-text("authentication")').first();
    if (await exampleButtons.count() > 0) {
      console.log('✅ Example search buttons found, clicking...');
      await exampleButtons.click();
      await page.waitForTimeout(3000); // Wait for search results

      // Check for search results
      const resultsContainer = await page.locator('[class*="result"], [data-testid*="result"]').count();
      if (resultsContainer === 0) {
        // Check page content for results
        const content = await page.content();
        if (content.includes('result') || content.includes('Result') || content.includes('Found')) {
          console.log('✅ Search results appeared (detected in content)');
        } else {
          test3.issues.push('Search executed but no results displayed');
          console.log('⚠️ Search may not have returned results');
        }
      } else {
        console.log(`✅ Found ${resultsContainer} result elements`);
      }

      await page.screenshot({
        path: path.join(SCREENSHOTS_DIR, '04-knowledge-search-results.png'),
        fullPage: true
      });
      console.log('📸 Screenshot saved: 04-knowledge-search-results.png');
    } else {
      test3.issues.push('Example search buttons not found');
      console.log('⚠️ Could not find example search buttons');
    }

    results.tests.push(test3);

    // Test 4: Refresh Button Functionality
    console.log('\n=== Test 4: Refresh Button ===');
    const test4 = {
      name: 'Refresh Button Functionality',
      status: 'PASS',
      issues: []
    };

    // Go back to dashboard
    await dashboardBtn.click();
    await page.waitForTimeout(1000);

    const refreshButton = await page.locator('text=/🔄|Refresh/i').first();
    if (await refreshButton.count() > 0) {
      await refreshButton.click();
      console.log('✅ Refresh button clicked');
      await page.waitForTimeout(2000);

      // Check if page is still functional after refresh
      const stillWorks = await page.locator('text=/Hyperion/i').first();
      if (await stillWorks.count() > 0) {
        console.log('✅ Page still functional after refresh');
      } else {
        test4.status = 'FAIL';
        test4.issues.push('Page may have broken after refresh');
      }
    } else {
      test4.status = 'FAIL';
      test4.issues.push('Refresh button not accessible');
    }

    await page.screenshot({
      path: path.join(SCREENSHOTS_DIR, '05-after-refresh.png'),
      fullPage: true
    });
    console.log('📸 Screenshot saved: 05-after-refresh.png');

    results.tests.push(test4);

    // Test 5: Console Errors Check
    console.log('\n=== Test 5: Console Errors ===');
    const test5 = {
      name: 'Console Errors Check',
      status: 'PASS',
      issues: []
    };

    const consoleMessages = [];
    page.on('console', msg => {
      consoleMessages.push({ type: msg.type(), text: msg.text() });
    });

    const errors = [];
    page.on('pageerror', error => {
      errors.push(error.message);
      test5.status = 'FAIL';
      test5.issues.push(`JavaScript Error: ${error.message}`);
    });

    // Navigate around to trigger any errors
    await dashboardBtn.click();
    await page.waitForTimeout(1000);
    await knowledgeBtn.click();
    await page.waitForTimeout(1000);

    if (errors.length > 0) {
      console.log('❌ JavaScript errors detected:', errors);
    } else {
      console.log('✅ No JavaScript errors detected');
    }

    results.tests.push(test5);

  } catch (error) {
    console.error('\n❌ CRITICAL ERROR during testing:', error.message);
    results.criticalError = error.message;

    await page.screenshot({
      path: path.join(SCREENSHOTS_DIR, 'error-state.png'),
      fullPage: true
    });
  } finally {
    // Generate summary
    console.log('\n=== TEST SUMMARY ===');
    const passed = results.tests.filter(t => t.status === 'PASS').length;
    const failed = results.tests.filter(t => t.status === 'FAIL').length;

    console.log(`Total Tests: ${results.tests.length}`);
    console.log(`✅ Passed: ${passed}`);
    console.log(`❌ Failed: ${failed}`);

    results.summary = {
      total: results.tests.length,
      passed,
      failed,
      passRate: `${((passed / results.tests.length) * 100).toFixed(1)}%`
    };

    // Save results to JSON
    fs.writeFileSync(
      path.join(SCREENSHOTS_DIR, 'test-results.json'),
      JSON.stringify(results, null, 2)
    );
    console.log('\n📄 Test results saved to test-results.json');

    await browser.close();
  }

  return results;
}

// Run tests
runTests().then(results => {
  console.log('\n✅ Testing complete!');
  console.log(`Screenshots saved to: ${SCREENSHOTS_DIR}`);
  process.exit(results.summary.failed > 0 ? 1 : 0);
}).catch(err => {
  console.error('\n❌ Test execution failed:', err);
  process.exit(1);
});
