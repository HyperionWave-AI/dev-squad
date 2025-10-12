import { test, expect } from '@playwright/test';

test.describe('Human Prompt Notes Feature', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the application
    await page.goto('/');
    // Wait for the task board to load
    await page.waitForSelector('[data-testid="task-board"]', { timeout: 10000 });
  });

  test('should add, edit, and clear task-level prompt notes', async ({ page }) => {
    // Mock MCP client responses
    await page.route('**/api/mcp/tools/call', async (route) => {
      const request = route.request();
      const postData = request.postDataJSON();

      if (postData.name === 'coordinator_add_task_prompt_notes') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            content: [{ type: 'text', text: '✓ Task prompt notes added successfully' }]
          })
        });
      } else if (postData.name === 'coordinator_update_task_prompt_notes') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            content: [{ type: 'text', text: '✓ Task prompt notes updated successfully' }]
          })
        });
      } else if (postData.name === 'coordinator_clear_task_prompt_notes') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            content: [{ type: 'text', text: '✓ Task prompt notes cleared successfully' }]
          })
        });
      } else {
        await route.continue();
      }
    });

    // Find and click on a pending agent task
    const agentTaskCard = page.locator('[data-testid="agent-task-card"]').first();
    await agentTaskCard.click();

    // Wait for task detail dialog to open
    await page.waitForSelector('[role="dialog"]');

    // Find the Human Guidance Notes accordion
    const notesAccordion = page.locator('text=Add Human Guidance Notes').or(page.locator('text=Human Guidance Notes')).first();
    await expect(notesAccordion).toBeVisible();

    // Click to expand if not already expanded
    if (await notesAccordion.locator('..').locator('[data-testid="ExpandMoreIcon"]').isVisible()) {
      await notesAccordion.click();
    }

    // Click Add Notes button
    const addNotesButton = page.locator('button:has-text("Add Notes")');
    await addNotesButton.click();

    // Type 100 characters in the text field
    const notesTextField = page.locator('textarea[placeholder*="guidance"]');
    const testNotes = 'This is a test note with exactly 100 characters to verify the character counter works correctly!!!!';
    await notesTextField.fill(testNotes);

    // Verify character counter shows 100/5000
    const characterCounter = page.locator('text=/100\\/5000/');
    await expect(characterCounter).toBeVisible();

    // Verify counter is green (success color)
    await expect(characterCounter).toHaveCSS('color', /rgb\(22, 163, 74\)/); // success.main color

    // Click Save button
    const saveButton = page.locator('button:has-text("Save")');
    await saveButton.click();

    // Wait for save to complete
    await page.waitForTimeout(500);

    // Verify notes are rendered with ReactMarkdown
    await expect(page.locator('text=' + testNotes)).toBeVisible();

    // Verify "Has Notes" chip appears
    const hasNotesChip = page.locator('[role="dialog"]').locator('text=Has Notes');
    await expect(hasNotesChip).toBeVisible();

    // Click Edit button
    const editButton = page.locator('button:has-text("Edit")').first();
    await editButton.click();

    // Modify to 200 characters
    const modifiedNotes = testNotes + ' Adding more text to reach 200 characters exactly for this test to verify edit functionality works!!';
    await notesTextField.fill(modifiedNotes);

    // Verify character counter shows 200/5000
    await expect(page.locator('text=/200\\/5000/')).toBeVisible();

    // Save the update
    await saveButton.click();
    await page.waitForTimeout(500);

    // Verify updated notes are rendered
    await expect(page.locator('text=' + modifiedNotes.substring(0, 50))).toBeVisible();

    // Click Clear button
    const clearButton = page.locator('button:has-text("Clear")');
    await clearButton.click();

    // Confirm the dialog
    page.once('dialog', dialog => {
      expect(dialog.message()).toContain('Are you sure');
      dialog.accept();
    });

    await page.waitForTimeout(500);

    // Verify notes are removed
    await expect(page.locator('text=' + modifiedNotes)).not.toBeVisible();
    await expect(page.locator('text=No notes added yet')).toBeVisible();
  });

  test('should add and display TODO-level prompt notes', async ({ page }) => {
    // Mock MCP client responses for TODO notes
    await page.route('**/api/mcp/tools/call', async (route) => {
      const request = route.request();
      const postData = request.postDataJSON();

      if (postData.name === 'coordinator_add_todo_prompt_notes') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            content: [{ type: 'text', text: '✓ TODO prompt notes added successfully' }]
          })
        });
      } else if (postData.name === 'coordinator_update_todo_prompt_notes') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            content: [{ type: 'text', text: '✓ TODO prompt notes updated successfully' }]
          })
        });
      } else {
        await route.continue();
      }
    });

    // Open task detail dialog
    const agentTaskCard = page.locator('[data-testid="agent-task-card"]').first();
    await agentTaskCard.click();
    await page.waitForSelector('[role="dialog"]');

    // Find first TODO item
    const todoItem = page.locator('[data-testid="todo-item"]').first();
    await expect(todoItem).toBeVisible();

    // Click expand button on TODO notes
    const expandButton = todoItem.locator('[aria-label*="Expand"]').or(todoItem.locator('[data-testid="ExpandMoreIcon"]')).first();
    await expandButton.click();

    // Click Add Notes button for TODO
    const addTodoNotesButton = page.locator('button:has-text("Add Notes")').last();
    await addTodoNotesButton.click();

    // Enter 50 character note
    const todoNotesField = page.locator('textarea[placeholder*="TODO"]').last();
    const todoNote = 'This TODO note has exactly 50 characters here!!';
    await todoNotesField.fill(todoNote);

    // Save TODO note
    const saveTodoButton = page.locator('button:has-text("Save")').last();
    await saveTodoButton.click();
    await page.waitForTimeout(500);

    // Verify 📝 badge appears
    const notesBadge = todoItem.locator('text=📝');
    await expect(notesBadge).toBeVisible();

    // Verify note preview shows first 50 chars
    await expect(todoItem.locator('text=' + todoNote)).toBeVisible();

    // Click expand to see full markdown
    const expandNotesButton = todoItem.locator('[aria-label*="Expand"]').last();
    await expandNotesButton.click();

    // Verify full markdown is rendered
    await expect(page.locator('text=' + todoNote)).toBeVisible();

    // Test multiple TODOs can have independent notes
    const secondTodo = page.locator('[data-testid="todo-item"]').nth(1);
    if (await secondTodo.isVisible()) {
      await secondTodo.locator('[aria-label*="Expand"]').first().click();
      const secondAddButton = secondTodo.locator('button:has-text("Add Notes")');
      if (await secondAddButton.isVisible()) {
        await secondAddButton.click();
        await secondTodo.locator('textarea').fill('Second TODO note');
        await secondTodo.locator('button:has-text("Save")').click();
        await page.waitForTimeout(500);

        // Verify second TODO has its own badge
        await expect(secondTodo.locator('text=📝')).toBeVisible();
      }
    }
  });

  test('should enforce 5000 character limit with visual feedback', async ({ page }) => {
    // Open task detail dialog
    const agentTaskCard = page.locator('[data-testid="agent-task-card"]').first();
    await agentTaskCard.click();
    await page.waitForSelector('[role="dialog"]');

    // Open notes editor
    const notesAccordion = page.locator('text=Add Human Guidance Notes').or(page.locator('text=Human Guidance Notes')).first();
    await notesAccordion.click();

    const addNotesButton = page.locator('button:has-text("Add Notes")');
    await addNotesButton.click();

    const notesTextField = page.locator('textarea[placeholder*="guidance"]');

    // Type 4400 characters - should be green
    const text4400 = 'x'.repeat(4400);
    await notesTextField.fill(text4400);

    let counter = page.locator('text=/4400\\/5000/');
    await expect(counter).toBeVisible();
    await expect(counter).toHaveCSS('color', /rgb\(22, 163, 74\)/); // green

    // Type to 4600 characters - should be orange
    const text4600 = 'x'.repeat(4600);
    await notesTextField.fill(text4600);

    counter = page.locator('text=/4600\\/5000/');
    await expect(counter).toBeVisible();
    await expect(counter).toHaveCSS('color', /rgb\(234, 88, 12\)/); // orange/warning

    // Type to 5001 characters - should be red and Save disabled
    const text5001 = 'x'.repeat(5001);
    await notesTextField.fill(text5001);

    counter = page.locator('text=/5001\\/5000/');
    await expect(counter).toBeVisible();
    await expect(counter).toHaveCSS('color', /rgb\(220, 38, 38\)/); // red/error

    // Verify Save button is disabled
    const saveButton = page.locator('button:has-text("Save")');
    await expect(saveButton).toBeDisabled();

    // Delete back to 4999 - Save should be enabled
    const text4999 = 'x'.repeat(4999);
    await notesTextField.fill(text4999);

    counter = page.locator('text=/4999\\/5000/');
    await expect(counter).toBeVisible();
    await expect(saveButton).toBeEnabled();
  });

  test('should disable editing when task status is in_progress', async ({ page }) => {
    // Mock task data with in_progress status
    await page.route('**/api/mcp/tools/call', async (route) => {
      const request = route.request();
      const postData = request.postDataJSON();

      if (postData.name === 'coordinator_list_agent_tasks') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            tasks: [{
              id: 'test-task-1',
              humanTaskId: 'human-1',
              agentName: 'test-agent',
              role: 'Test Role',
              status: 'in_progress',
              humanPromptNotes: 'Existing notes that should be read-only',
              humanPromptNotesAddedAt: new Date().toISOString(),
              todos: [],
              createdAt: new Date().toISOString(),
              updatedAt: new Date().toISOString()
            }]
          })
        });
      } else {
        await route.continue();
      }
    });

    // Open task detail dialog
    const agentTaskCard = page.locator('[data-testid="agent-task-card"]').first();
    await agentTaskCard.click();
    await page.waitForSelector('[role="dialog"]');

    // Open notes accordion
    const notesAccordion = page.locator('text=Human Guidance Notes').first();
    await notesAccordion.click();

    // Verify notes are rendered as markdown
    await expect(page.locator('text=Existing notes that should be read-only')).toBeVisible();

    // Verify Edit and Clear buttons are not visible or disabled
    const editButton = page.locator('button:has-text("Edit")').first();
    const clearButton = page.locator('button:has-text("Clear")').first();

    await expect(editButton).not.toBeVisible();
    await expect(clearButton).not.toBeVisible();

    // Verify "Notes locked" message appears
    await expect(page.locator('text=/Notes locked.*task in progress/i')).toBeVisible();

    // Attempt to click the notes area - verify no input field appears
    await page.locator('text=Existing notes').click();
    await expect(page.locator('textarea[placeholder*="guidance"]')).not.toBeVisible();
  });

  test('should meet WCAG 2.1 AA accessibility standards', async ({ page }) => {
    // Open task detail dialog
    const agentTaskCard = page.locator('[data-testid="agent-task-card"]').first();
    await agentTaskCard.click();
    await page.waitForSelector('[role="dialog"]');

    // Open notes editor
    const addNotesButton = page.locator('button:has-text("Add Notes")');
    await addNotesButton.click();

    // Verify all buttons have aria-label
    await expect(page.locator('button[aria-label="Edit notes"]').or(page.locator('button[aria-label="Add notes"]'))).toHaveCount(1);
    await expect(page.locator('button[aria-label="Save notes"]')).toHaveCount(1);
    await expect(page.locator('button[aria-label="Cancel editing notes"]')).toHaveCount(1);

    // Test keyboard navigation: Tab through controls
    await page.keyboard.press('Tab'); // Focus on TextField
    await expect(page.locator('textarea[placeholder*="guidance"]')).toBeFocused();

    await page.keyboard.press('Tab'); // Focus on Cancel button
    await expect(page.locator('button:has-text("Cancel")')).toBeFocused();

    await page.keyboard.press('Tab'); // Focus on Save button
    await expect(page.locator('button:has-text("Save")')).toBeFocused();

    // Test Enter key saves notes (when focused on Save button)
    const notesTextField = page.locator('textarea[placeholder*="guidance"]');
    await notesTextField.fill('Test accessibility note');
    await page.keyboard.press('Tab');
    await page.keyboard.press('Tab');
    await page.keyboard.press('Enter');

    await page.waitForTimeout(500);
    await expect(page.locator('text=Test accessibility note')).toBeVisible();

    // Test Escape key cancels editing
    const editButton = page.locator('button:has-text("Edit")').first();
    await editButton.click();
    await page.keyboard.press('Escape');
    await expect(page.locator('textarea[placeholder*="guidance"]')).not.toBeVisible();

    // Verify character counter has aria-live for screen reader updates
    await addNotesButton.click();
    const characterCounter = page.locator('text=/\\/5000/');
    // Note: aria-live might be on parent element
    const counterElement = characterCounter.locator('..');
    const ariaLive = await counterElement.getAttribute('aria-live');
    expect(ariaLive).toBe('polite');

    // Test color contrast for character counter states
    // Green state (< 4500)
    await notesTextField.fill('x'.repeat(100));
    let counter = page.locator('text=/100\\/5000/');
    let color = await counter.evaluate((el) => window.getComputedStyle(el).color);
    expect(color).toMatch(/rgb\(22, 163, 74\)/); // Ensure it's the expected green

    // Orange state (4500-4900)
    await notesTextField.fill('x'.repeat(4600));
    counter = page.locator('text=/4600\\/5000/');
    color = await counter.evaluate((el) => window.getComputedStyle(el).color);
    expect(color).toMatch(/rgb\(234, 88, 12\)/); // Orange

    // Red state (> 4900)
    await notesTextField.fill('x'.repeat(4950));
    counter = page.locator('text=/4950\\/5000/');
    color = await counter.evaluate((el) => window.getComputedStyle(el).color);
    expect(color).toMatch(/rgb\(220, 38, 38\)/); // Red

    // Run axe-core accessibility scan
    // Note: Requires @axe-core/playwright to be installed
    // await injectAxe(page);
    // const results = await checkA11y(page);
    // expect(results.violations).toHaveLength(0);
  });
});
