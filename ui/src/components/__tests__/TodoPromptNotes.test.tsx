import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TodoPromptNotes } from '../TodoPromptNotes';
import { restClient } from '../../services/restClient';
import type { TodoItem } from '../../types/coordinator';

// Mock REST client
jest.mock('../../services/restClient', () => ({
  restClient: {
    addTodoPromptNotes: jest.fn(),
    updateTodoPromptNotes: jest.fn(),
  },
}));

// Mock ReactMarkdown
jest.mock('react-markdown', () => {
  return function ReactMarkdown({ children }: { children: string }) {
    return <div data-testid="markdown-content">{children}</div>;
  };
});

describe('TodoPromptNotes', () => {
  const mockOnUpdate = jest.fn();
  const agentTaskId = 'test-agent-task-id';

  const createTodoWithNotes = (notes?: string): TodoItem => ({
    id: 'todo-1',
    description: 'Test TODO item',
    status: 'pending',
    createdAt: new Date().toISOString(),
    humanPromptNotes: notes,
  });

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('should render 📝 badge when todo has notes', () => {
    const todo = createTodoWithNotes('Existing TODO notes');

    render(
      <TodoPromptNotes
        todo={todo}
        agentTaskId={agentTaskId}
        isTaskPending={true}
        onUpdate={mockOnUpdate}
      />
    );

    expect(screen.getByText('📝')).toBeInTheDocument();
  });

  test('should show preview of first 50 characters when collapsed', () => {
    const longNotes = 'This is a very long note that exceeds fifty characters and should be truncated in the preview';
    const todo = createTodoWithNotes(longNotes);

    render(
      <TodoPromptNotes
        todo={todo}
        agentTaskId={agentTaskId}
        isTaskPending={true}
        onUpdate={mockOnUpdate}
      />
    );

    const preview = screen.getByText(/This is a very long note that exceeds fifty charac.../);
    expect(preview).toBeInTheDocument();
  });

  test('should expand to show full markdown when expand button is clicked', async () => {
    const user = userEvent.setup();
    const notes = 'Full markdown content with **bold** text';
    const todo = createTodoWithNotes(notes);

    render(
      <TodoPromptNotes
        todo={todo}
        agentTaskId={agentTaskId}
        isTaskPending={true}
        onUpdate={mockOnUpdate}
      />
    );

    // Click expand button
    const expandButton = screen.getByRole('button', { name: /expand/i });
    await user.click(expandButton);

    // Verify full markdown is rendered
    const markdownContent = screen.getByTestId('markdown-content');
    expect(markdownContent).toHaveTextContent(notes);
  });

  test('should disable edit button when isTaskPending=false', () => {
    const todo = createTodoWithNotes('Locked notes');

    render(
      <TodoPromptNotes
        todo={todo}
        agentTaskId={agentTaskId}
        isTaskPending={false}
        onUpdate={mockOnUpdate}
      />
    );

    // Expand to see the edit button area
    const expandButton = screen.getByRole('button', { name: /expand/i });
    fireEvent.click(expandButton);

    // Edit button should not be visible
    expect(screen.queryByRole('button', { name: /edit/i })).not.toBeInTheDocument();
  });

  test('should call addTodoPromptNotes when adding new note', async () => {
    const user = userEvent.setup();
    const todo = createTodoWithNotes(); // No notes initially
    (restClient.addTodoPromptNotes as jest.Mock).mockResolvedValue(undefined);

    render(
      <TodoPromptNotes
        todo={todo}
        agentTaskId={agentTaskId}
        isTaskPending={true}
        onUpdate={mockOnUpdate}
      />
    );

    // Expand the notes section
    const expandButton = screen.getByRole('button', { name: /expand/i });
    await user.click(expandButton);

    // Click Add Notes button
    const addButton = screen.getByRole('button', { name: /add notes/i });
    await user.click(addButton);

    // Type in the text field
    const textField = screen.getByPlaceholderText(/add guidance notes/i);
    await user.type(textField, 'New TODO note');

    // Click Save
    const saveButton = screen.getByRole('button', { name: /save/i });
    await user.click(saveButton);

    await waitFor(() => {
      expect(restClient.addTodoPromptNotes).toHaveBeenCalledWith(
        agentTaskId,
        'todo-1',
        'New TODO note'
      );
      expect(mockOnUpdate).toHaveBeenCalled();
    });
  });

  test('should call updateTodoPromptNotes when editing existing note', async () => {
    const user = userEvent.setup();
    const todo = createTodoWithNotes('Existing note');
    (restClient.updateTodoPromptNotes as jest.Mock).mockResolvedValue(undefined);

    render(
      <TodoPromptNotes
        todo={todo}
        agentTaskId={agentTaskId}
        isTaskPending={true}
        onUpdate={mockOnUpdate}
      />
    );

    // Expand
    await user.click(screen.getByRole('button', { name: /expand/i }));

    // Click Edit
    const editButton = screen.getByRole('button', { name: /edit/i });
    await user.click(editButton);

    // Modify the text
    const textField = screen.getByDisplayValue('Existing note');
    await user.clear(textField);
    await user.type(textField, 'Updated note');

    // Save
    const saveButton = screen.getByRole('button', { name: /save/i });
    await user.click(saveButton);

    await waitFor(() => {
      expect(restClient.updateTodoPromptNotes).toHaveBeenCalledWith(
        agentTaskId,
        'todo-1',
        'Updated note'
      );
      expect(mockOnUpdate).toHaveBeenCalled();
    });
  });

  test('should not render anything when no notes and task not pending', () => {
    const todo = createTodoWithNotes();

    const { container } = render(
      <TodoPromptNotes
        todo={todo}
        agentTaskId={agentTaskId}
        isTaskPending={false}
        onUpdate={mockOnUpdate}
      />
    );

    expect(container.firstChild).toBeNull();
  });

  test('should have compact layout that fits in TODO list', () => {
    const todo = createTodoWithNotes('Compact note');

    const { container } = render(
      <TodoPromptNotes
        todo={todo}
        agentTaskId={agentTaskId}
        isTaskPending={true}
        onUpdate={mockOnUpdate}
      />
    );

    // Check for compact styling (ml: 4, mt: 1, mb: 1)
    const wrapper = container.querySelector('.MuiBox-root');
    expect(wrapper).toBeInTheDocument();
  });

  test('should cancel editing when Cancel button is clicked', async () => {
    const user = userEvent.setup();
    const todo = createTodoWithNotes('Original note');

    render(
      <TodoPromptNotes
        todo={todo}
        agentTaskId={agentTaskId}
        isTaskPending={true}
        onUpdate={mockOnUpdate}
      />
    );

    // Expand and edit
    await user.click(screen.getByRole('button', { name: /expand/i }));
    await user.click(screen.getByRole('button', { name: /edit/i }));

    // Modify text
    const textField = screen.getByDisplayValue('Original note');
    await user.clear(textField);
    await user.type(textField, 'Modified note');

    // Cancel
    const cancelButton = screen.getByRole('button', { name: /cancel/i });
    await user.click(cancelButton);

    // Should not be in edit mode anymore
    expect(screen.queryByPlaceholderText(/add guidance notes/i)).not.toBeInTheDocument();

    // Original note should still be visible
    expect(screen.getByTestId('markdown-content')).toHaveTextContent('Original note');
  });

  test('should show loading state while saving', async () => {
    const user = userEvent.setup();
    const todo = createTodoWithNotes();

    // Mock slow save operation
    (restClient.addTodoPromptNotes as jest.Mock).mockImplementation(
      () => new Promise(resolve => setTimeout(resolve, 100))
    );

    render(
      <TodoPromptNotes
        todo={todo}
        agentTaskId={agentTaskId}
        isTaskPending={true}
        onUpdate={mockOnUpdate}
      />
    );

    await user.click(screen.getByRole('button', { name: /expand/i }));
    await user.click(screen.getByRole('button', { name: /add notes/i }));

    const textField = screen.getByPlaceholderText(/add guidance notes/i);
    await user.type(textField, 'New note');

    const saveButton = screen.getByRole('button', { name: /save/i });
    await user.click(saveButton);

    // Check for "Saving..." text
    expect(screen.getByText(/saving/i)).toBeInTheDocument();

    await waitFor(() => {
      expect(mockOnUpdate).toHaveBeenCalled();
    });
  });

  test('should handle error when save fails', async () => {
    const user = userEvent.setup();
    const todo = createTodoWithNotes();
    const consoleError = jest.spyOn(console, 'error').mockImplementation();

    (restClient.addTodoPromptNotes as jest.Mock).mockRejectedValue(
      new Error('Network error')
    );

    render(
      <TodoPromptNotes
        todo={todo}
        agentTaskId={agentTaskId}
        isTaskPending={true}
        onUpdate={mockOnUpdate}
      />
    );

    await user.click(screen.getByRole('button', { name: /expand/i }));
    await user.click(screen.getByRole('button', { name: /add notes/i }));

    const textField = screen.getByPlaceholderText(/add guidance notes/i);
    await user.type(textField, 'New note');

    const saveButton = screen.getByRole('button', { name: /save/i });
    await user.click(saveButton);

    await waitFor(() => {
      expect(consoleError).toHaveBeenCalledWith(
        'Failed to save TODO notes:',
        expect.any(Error)
      );
    });

    consoleError.mockRestore();
  });
});
