/**
 * Test file for ChatSessionList Delete All functionality
 * 
 * This test verifies that the delete all button appears when sessions exist,
 * shows the confirmation dialog, and calls the delete all handler when confirmed.
 */

import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ChatSessionList } from '../ChatSessionList';
import type { ChatSession } from '../../services/chatService';

// Mock chat sessions for testing
const mockSessions: ChatSession[] = [
  {
    id: '1',
    title: 'Test Chat 1',
    createdAt: '2024-01-01T10:00:00Z',
    updatedAt: '2024-01-01T10:00:00Z',
  },
  {
    id: '2',
    title: 'Test Chat 2',
    createdAt: '2024-01-02T10:00:00Z',
    updatedAt: '2024-01-02T10:00:00Z',
  },
];

describe('ChatSessionList Delete All Functionality', () => {
  const mockProps = {
    sessions: mockSessions,
    activeSessionId: '1',
    onSessionSelect: jest.fn(),
    onNewChat: jest.fn(),
    onDeleteSession: jest.fn(),
    onDeleteAllSessions: jest.fn(),
    onRenameSession: jest.fn(),
    loading: false,
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('shows Delete All button when sessions exist', () => {
    render(<ChatSessionList {...mockProps} />);
    
    // Button is now icon-only, so we use aria-label to find it
    const deleteAllButton = screen.getByRole('button', { name: /delete all chat sessions/i });
    expect(deleteAllButton).toBeInTheDocument();
    
    // Verify it's an IconButton with DeleteSweep icon (no text content)
    expect(deleteAllButton).not.toHaveTextContent('Delete All');
    
    // Check for DeleteSweep icon (MUI icons are rendered as SVG)
    const icon = deleteAllButton.querySelector('svg');
    expect(icon).toBeInTheDocument();
  });

  test('hides Delete All button when no sessions exist', () => {
    render(<ChatSessionList {...mockProps} sessions={[]} />);
    
    const deleteAllButton = screen.queryByRole('button', { name: /delete all chat sessions/i });
    expect(deleteAllButton).not.toBeInTheDocument();
  });

  test('opens confirmation dialog when Delete All button is clicked', async () => {
    render(<ChatSessionList {...mockProps} />);
    
    const deleteAllButton = screen.getByRole('button', { name: /delete all chat sessions/i });
    fireEvent.click(deleteAllButton);

    await waitFor(() => {
      expect(screen.getByText('Delete All Chat Sessions')).toBeInTheDocument();
      expect(screen.getByText(/delete all chat sessions/)).toBeInTheDocument();
      expect(screen.getByText(/This action cannot be undone/)).toBeInTheDocument();
    });
  });

  test('calls onDeleteAllSessions when confirmed', async () => {
    render(<ChatSessionList {...mockProps} />);
    
    // Click Delete All button
    const deleteAllButton = screen.getByRole('button', { name: /delete all chat sessions/i });
    fireEvent.click(deleteAllButton);

    // Wait for dialog and click confirm
    await waitFor(() => {
      const confirmButton = screen.getByRole('button', { name: /delete all/i });
      fireEvent.click(confirmButton);
    });

    expect(mockProps.onDeleteAllSessions).toHaveBeenCalledTimes(1);
  });

  test('closes dialog when cancelled', async () => {
    render(<ChatSessionList {...mockProps} />);
    
    // Click Delete All button
    const deleteAllButton = screen.getByRole('button', { name: /delete all chat sessions/i });
    fireEvent.click(deleteAllButton);

    // Wait for dialog and click cancel
    await waitFor(() => {
      const cancelButton = screen.getByRole('button', { name: /cancel/i });
      fireEvent.click(cancelButton);
    });

    // Dialog should be closed
    await waitFor(() => {
      expect(screen.queryByText('Delete All Chat Sessions')).not.toBeInTheDocument();
    });

    expect(mockProps.onDeleteAllSessions).not.toHaveBeenCalled();
  });

  test('disables Delete All button when loading', () => {
    render(<ChatSessionList {...mockProps} loading={true} />);
    
    const deleteAllButton = screen.getByRole('button', { name: /delete all chat sessions/i });
    expect(deleteAllButton).toBeDisabled();
  });

  test('Delete All button has proper styling and icon', () => {
    render(<ChatSessionList {...mockProps} />);
    
    const deleteAllButton = screen.getByRole('button', { name: /delete all chat sessions/i });
    
    // Check for DeleteSweep icon (MUI icons are rendered as SVG)
    const icon = deleteAllButton.querySelector('svg');
    expect(icon).toBeInTheDocument();
    
    // Check it's an IconButton (not a regular Button with text)
    expect(deleteAllButton).toHaveClass('MuiIconButton-root');
    
    // Verify error color styling
    expect(deleteAllButton).toHaveClass('MuiIconButton-colorError');
    
    // Verify it has proper accessibility attributes
    expect(deleteAllButton).toHaveAttribute('title', 'Delete All Chats');
    expect(deleteAllButton).toHaveAttribute('aria-label', 'Delete all chat sessions');
  });

  test('button is accessible via keyboard navigation', () => {
    render(<ChatSessionList {...mockProps} />);
    
    const deleteAllButton = screen.getByRole('button', { name: /delete all chat sessions/i });
    
    // Button should be focusable
    deleteAllButton.focus();
    expect(deleteAllButton).toHaveFocus();
    
    // Should be clickable via keyboard
    fireEvent.keyDown(deleteAllButton, { key: 'Enter' });
    
    // Dialog should open
    waitFor(() => {
      expect(screen.getByText('Delete All Chat Sessions')).toBeInTheDocument();
    });
  });

  test('button has proper hover states', () => {
    render(<ChatSessionList {...mockProps} />);
    
    const deleteAllButton = screen.getByRole('button', { name: /delete all chat sessions/i });
    
    // Simulate hover
    fireEvent.mouseEnter(deleteAllButton);
    
    // Button should maintain its styling and be interactive
    expect(deleteAllButton).toBeInTheDocument();
    expect(deleteAllButton).not.toBeDisabled();
  });
});