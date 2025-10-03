import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { PromptNotesEditor } from '../PromptNotesEditor';

// Mock ReactMarkdown
jest.mock('react-markdown', () => {
  return function ReactMarkdown({ children }: { children: string }) {
    return <div data-testid="markdown-content">{children}</div>;
  };
});

describe('PromptNotesEditor', () => {
  const mockOnSave = jest.fn();
  const mockOnClear = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('should render with Edit button when isEditable=true', () => {
    render(
      <PromptNotesEditor
        notes="Existing notes"
        isEditable={true}
        onSave={mockOnSave}
        onClear={mockOnClear}
        placeholder="Test placeholder"
      />
    );

    expect(screen.getByText('Human Guidance Notes')).toBeInTheDocument();
    expect(screen.getByText('Existing notes')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /edit/i })).toBeInTheDocument();
  });

  test('should show TextField when Edit button is clicked', async () => {
    const user = userEvent.setup();

    render(
      <PromptNotesEditor
        notes="Existing notes"
        isEditable={true}
        onSave={mockOnSave}
        onClear={mockOnClear}
        placeholder="Test placeholder"
      />
    );

    const editButton = screen.getByRole('button', { name: /edit/i });
    await user.click(editButton);

    expect(screen.getByPlaceholderText('Test placeholder')).toBeInTheDocument();
    expect(screen.getByDisplayValue('Existing notes')).toBeInTheDocument();
  });

  test('should update character counter when typing', async () => {
    const user = userEvent.setup();

    render(
      <PromptNotesEditor
        isEditable={true}
        onSave={mockOnSave}
        onClear={mockOnClear}
        placeholder="Test placeholder"
      />
    );

    // Click Add Notes button
    const addButton = screen.getByRole('button', { name: /add notes/i });
    await user.click(addButton);

    const textField = screen.getByPlaceholderText('Test placeholder');

    // Type 100 characters
    const text100 = 'x'.repeat(100);
    await user.type(textField, text100);

    await waitFor(() => {
      expect(screen.getByText(/100\/5000/)).toBeInTheDocument();
    });
  });

  test('should call onSave with correct text when Save is clicked', async () => {
    const user = userEvent.setup();
    mockOnSave.mockResolvedValue(undefined);

    render(
      <PromptNotesEditor
        notes="Old notes"
        isEditable={true}
        onSave={mockOnSave}
        onClear={mockOnClear}
        placeholder="Test placeholder"
      />
    );

    // Click Edit
    await user.click(screen.getByRole('button', { name: /edit/i }));

    const textField = screen.getByPlaceholderText('Test placeholder');
    await user.clear(textField);
    await user.type(textField, 'New notes');

    // Click Save
    const saveButton = screen.getByRole('button', { name: /save/i });
    await user.click(saveButton);

    await waitFor(() => {
      expect(mockOnSave).toHaveBeenCalledWith('New notes');
    });
  });

  test('should restore original notes when Cancel is clicked', async () => {
    const user = userEvent.setup();

    render(
      <PromptNotesEditor
        notes="Original notes"
        isEditable={true}
        onSave={mockOnSave}
        onClear={mockOnClear}
        placeholder="Test placeholder"
      />
    );

    // Click Edit
    await user.click(screen.getByRole('button', { name: /edit/i }));

    const textField = screen.getByPlaceholderText('Test placeholder');
    await user.clear(textField);
    await user.type(textField, 'Modified notes');

    // Click Cancel
    const cancelButton = screen.getByRole('button', { name: /cancel/i });
    await user.click(cancelButton);

    // Should no longer be in edit mode
    expect(screen.queryByPlaceholderText('Test placeholder')).not.toBeInTheDocument();

    // Original notes should still be visible
    expect(screen.getByText('Original notes')).toBeInTheDocument();
  });

  test('should hide Edit button when isEditable=false', () => {
    render(
      <PromptNotesEditor
        notes="Locked notes"
        isEditable={false}
        onSave={mockOnSave}
        onClear={mockOnClear}
        placeholder="Test placeholder"
      />
    );

    expect(screen.queryByRole('button', { name: /edit/i })).not.toBeInTheDocument();
    expect(screen.getByText(/notes locked.*task in progress/i)).toBeInTheDocument();
  });

  test('should render markdown with ReactMarkdown', () => {
    const markdownText = '**Bold text** and *italic text*';

    render(
      <PromptNotesEditor
        notes={markdownText}
        isEditable={true}
        onSave={mockOnSave}
        onClear={mockOnClear}
        placeholder="Test placeholder"
      />
    );

    const markdownContent = screen.getByTestId('markdown-content');
    expect(markdownContent).toHaveTextContent(markdownText);
  });

  test('should disable Save button when over character limit', async () => {
    const user = userEvent.setup();

    render(
      <PromptNotesEditor
        isEditable={true}
        onSave={mockOnSave}
        onClear={mockOnClear}
        placeholder="Test placeholder"
      />
    );

    // Click Add Notes
    await user.click(screen.getByRole('button', { name: /add notes/i }));

    const textField = screen.getByPlaceholderText('Test placeholder');

    // Type over 5000 characters
    const text5001 = 'x'.repeat(5001);
    fireEvent.change(textField, { target: { value: text5001 } });

    await waitFor(() => {
      expect(screen.getByText(/5001\/5000/)).toBeInTheDocument();
    });

    const saveButton = screen.getByRole('button', { name: /save/i });
    expect(saveButton).toBeDisabled();
  });

  test('should show color-coded character counter', async () => {
    const user = userEvent.setup();

    render(
      <PromptNotesEditor
        isEditable={true}
        onSave={mockOnSave}
        onClear={mockOnClear}
        placeholder="Test placeholder"
      />
    );

    await user.click(screen.getByRole('button', { name: /add notes/i }));
    const textField = screen.getByPlaceholderText('Test placeholder');

    // Test green (< 4500)
    fireEvent.change(textField, { target: { value: 'x'.repeat(100) } });
    await waitFor(() => {
      const counter = screen.getByText(/100\/5000/);
      const styles = window.getComputedStyle(counter);
      // Green color - success.main
      expect(styles.color).toMatch(/rgb\(22, 163, 74\)/);
    });

    // Test orange (4500-4900)
    fireEvent.change(textField, { target: { value: 'x'.repeat(4600) } });
    await waitFor(() => {
      const counter = screen.getByText(/4600\/5000/);
      const styles = window.getComputedStyle(counter);
      // Orange color - warning.main
      expect(styles.color).toMatch(/rgb\(234, 88, 12\)/);
    });

    // Test red (> 4900)
    fireEvent.change(textField, { target: { value: 'x'.repeat(4950) } });
    await waitFor(() => {
      const counter = screen.getByText(/4950\/5000/);
      const styles = window.getComputedStyle(counter);
      // Red color - error.main
      expect(styles.color).toMatch(/rgb\(220, 38, 38\)/);
    });
  });

  test('should display timestamp when notesAddedAt is provided', () => {
    const addedAt = '2025-10-03T10:00:00Z';

    render(
      <PromptNotesEditor
        notes="Notes with timestamp"
        notesAddedAt={addedAt}
        isEditable={true}
        onSave={mockOnSave}
        onClear={mockOnClear}
        placeholder="Test placeholder"
      />
    );

    expect(screen.getByText(/added:/i)).toBeInTheDocument();
  });

  test('should confirm before clearing notes', async () => {
    const user = userEvent.setup();
    mockOnClear.mockResolvedValue(undefined);
    global.confirm = jest.fn(() => true);

    render(
      <PromptNotesEditor
        notes="Notes to clear"
        isEditable={true}
        onSave={mockOnSave}
        onClear={mockOnClear}
        placeholder="Test placeholder"
      />
    );

    const clearButton = screen.getByRole('button', { name: /clear/i });
    await user.click(clearButton);

    expect(global.confirm).toHaveBeenCalledWith(expect.stringContaining('Are you sure'));
    expect(mockOnClear).toHaveBeenCalled();
  });

  test('should not clear notes if confirmation is cancelled', async () => {
    const user = userEvent.setup();
    global.confirm = jest.fn(() => false);

    render(
      <PromptNotesEditor
        notes="Notes to keep"
        isEditable={true}
        onSave={mockOnSave}
        onClear={mockOnClear}
        placeholder="Test placeholder"
      />
    );

    const clearButton = screen.getByRole('button', { name: /clear/i });
    await user.click(clearButton);

    expect(global.confirm).toHaveBeenCalled();
    expect(mockOnClear).not.toHaveBeenCalled();
  });

  test('should show no notes message when notes are empty', () => {
    render(
      <PromptNotesEditor
        isEditable={true}
        onSave={mockOnSave}
        onClear={mockOnClear}
        placeholder="Test placeholder"
      />
    );

    expect(screen.getByText(/no notes added yet/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /add notes/i })).toBeInTheDocument();
  });
});
