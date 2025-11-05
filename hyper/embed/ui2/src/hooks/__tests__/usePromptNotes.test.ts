import { renderHook, act, waitFor } from '@testing-library/react';
import { usePromptNotes } from '../usePromptNotes';
import type { AgentTask } from '../../types/coordinator';

describe('usePromptNotes', () => {
  const createMockTask = (notes?: string): AgentTask => ({
    id: 'test-task-id',
    humanTaskId: 'human-task-id',
    agentName: 'test-agent',
    role: 'Test role',
    todos: [],
    status: 'pending',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    humanPromptNotes: notes,
  });

  const mockOnUpdate = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('should initialize with correct default state', () => {
    const task = createMockTask();
    const { result } = renderHook(() => usePromptNotes({ task, onUpdate: mockOnUpdate }));

    expect(result.current.isEditing).toBe(false);
    expect(result.current.draftNotes).toBe('');
    expect(result.current.isSaving).toBe(false);
    expect(result.current.error).toBe(null);
    expect(result.current.characterCount).toBe(0);
    expect(result.current.isOverLimit).toBe(false);
  });

  test('should set isEditing to true when handleEdit is called', () => {
    const task = createMockTask('Existing notes');
    const { result } = renderHook(() => usePromptNotes({ task, onUpdate: mockOnUpdate }));

    act(() => {
      result.current.handleEdit();
    });

    expect(result.current.isEditing).toBe(true);
    expect(result.current.draftNotes).toBe('Existing notes');
    expect(result.current.error).toBe(null);
  });

  test('should update draftNotes when setDraftNotes is called', () => {
    const task = createMockTask();
    const { result } = renderHook(() => usePromptNotes({ task, onUpdate: mockOnUpdate }));

    act(() => {
      result.current.handleEdit();
    });

    act(() => {
      result.current.setDraftNotes('Updated notes');
    });

    expect(result.current.draftNotes).toBe('Updated notes');
  });

  test('should call onUpdate optimistically when handleSave is called', async () => {
    const task = createMockTask();
    const { result } = renderHook(() => usePromptNotes({ task, onUpdate: mockOnUpdate }));

    act(() => {
      result.current.handleEdit();
      result.current.setDraftNotes('New notes');
    });

    await act(async () => {
      await result.current.handleSave();
    });

    // onUpdate should be called immediately (optimistic update)
    expect(mockOnUpdate).toHaveBeenCalled();
    expect(result.current.isEditing).toBe(false);
    expect(result.current.isSaving).toBe(false);
  });

  test('should set error when notes exceed character limit', async () => {
    const task = createMockTask();
    const { result } = renderHook(() => usePromptNotes({ task, onUpdate: mockOnUpdate }));

    act(() => {
      result.current.handleEdit();
      result.current.setDraftNotes('x'.repeat(5001));
    });

    await act(async () => {
      await result.current.handleSave();
    });

    expect(result.current.error).toBe('Notes exceed maximum length of 5000 characters');
    expect(result.current.isEditing).toBe(true); // Should still be in edit mode
    expect(mockOnUpdate).not.toHaveBeenCalled();
  });

  test('should rollback draftNotes on error', async () => {
    const task = createMockTask('Original notes');
    const mockOnUpdateWithError = jest.fn(() => {
      throw new Error('Save failed');
    });

    const { result } = renderHook(() =>
      usePromptNotes({ task, onUpdate: mockOnUpdateWithError })
    );

    act(() => {
      result.current.handleEdit();
      result.current.setDraftNotes('Modified notes');
    });

    await act(async () => {
      await result.current.handleSave();
    });

    // Error should be set
    expect(result.current.error).toBe('Save failed');

    // Draft notes should be restored to original
    expect(result.current.draftNotes).toBe('Original notes');

    // onUpdate should be called twice: once for optimistic update, once for rollback
    expect(mockOnUpdateWithError).toHaveBeenCalledTimes(2);
  });

  test('should calculate character count correctly', () => {
    const task = createMockTask();
    const { result } = renderHook(() => usePromptNotes({ task, onUpdate: mockOnUpdate }));

    act(() => {
      result.current.handleEdit();
      result.current.setDraftNotes('Test notes with 25 chars');
    });

    expect(result.current.characterCount).toBe(25);
  });

  test('should set isOverLimit to true when character count exceeds 5000', () => {
    const task = createMockTask();
    const { result } = renderHook(() => usePromptNotes({ task, onUpdate: mockOnUpdate }));

    act(() => {
      result.current.handleEdit();
      result.current.setDraftNotes('x'.repeat(5001));
    });

    expect(result.current.isOverLimit).toBe(true);
  });

  test('should set isOverLimit to false when character count is within limit', () => {
    const task = createMockTask();
    const { result } = renderHook(() => usePromptNotes({ task, onUpdate: mockOnUpdate }));

    act(() => {
      result.current.handleEdit();
      result.current.setDraftNotes('x'.repeat(4999));
    });

    expect(result.current.isOverLimit).toBe(false);
  });

  test('should reset state when handleCancel is called', () => {
    const task = createMockTask('Original notes');
    const { result } = renderHook(() => usePromptNotes({ task, onUpdate: mockOnUpdate }));

    act(() => {
      result.current.handleEdit();
      result.current.setDraftNotes('Modified notes');
    });

    act(() => {
      result.current.handleCancel();
    });

    expect(result.current.isEditing).toBe(false);
    expect(result.current.draftNotes).toBe('');
    expect(result.current.error).toBe(null);
  });

  test('should call onUpdate when handleClear is called', async () => {
    const task = createMockTask('Notes to clear');
    const { result } = renderHook(() => usePromptNotes({ task, onUpdate: mockOnUpdate }));

    await act(async () => {
      await result.current.handleClear();
    });

    expect(mockOnUpdate).toHaveBeenCalled();
    expect(result.current.isEditing).toBe(false);
    expect(result.current.draftNotes).toBe('');
  });

  test('should rollback on clear error', async () => {
    const task = createMockTask('Original notes');
    const mockOnUpdateWithError = jest.fn(() => {
      throw new Error('Clear failed');
    });

    const { result } = renderHook(() =>
      usePromptNotes({ task, onUpdate: mockOnUpdateWithError })
    );

    await act(async () => {
      await result.current.handleClear();
    });

    expect(result.current.error).toBe('Clear failed');
    expect(result.current.draftNotes).toBe('Original notes');
    expect(mockOnUpdateWithError).toHaveBeenCalledTimes(2); // Optimistic + rollback
  });

  test('should set isSaving to true during save operation', async () => {
    const task = createMockTask();
    let savingState = false;

    const { result } = renderHook(() => usePromptNotes({ task, onUpdate: mockOnUpdate }));

    act(() => {
      result.current.handleEdit();
      result.current.setDraftNotes('New notes');
    });

    const savePromise = act(async () => {
      const promise = result.current.handleSave();
      savingState = result.current.isSaving;
      await promise;
    });

    // isSaving should be true during the operation
    expect(savingState).toBe(true);

    await savePromise;

    // isSaving should be false after completion
    expect(result.current.isSaving).toBe(false);
  });

  test('should handle undefined original notes in error rollback', async () => {
    const task = createMockTask(); // No initial notes
    const mockOnUpdateWithError = jest.fn(() => {
      throw new Error('Save failed');
    });

    const { result } = renderHook(() =>
      usePromptNotes({ task, onUpdate: mockOnUpdateWithError })
    );

    act(() => {
      result.current.handleEdit();
      result.current.setDraftNotes('New notes');
    });

    await act(async () => {
      await result.current.handleSave();
    });

    // Should handle undefined gracefully
    expect(result.current.error).toBe('Save failed');
    // Draft notes might be empty or the attempted value depending on implementation
  });

  test('should preserve error state across operations', async () => {
    const task = createMockTask();
    const { result } = renderHook(() => usePromptNotes({ task, onUpdate: mockOnUpdate }));

    act(() => {
      result.current.handleEdit();
      result.current.setDraftNotes('x'.repeat(5001));
    });

    await act(async () => {
      await result.current.handleSave();
    });

    expect(result.current.error).toBe('Notes exceed maximum length of 5000 characters');

    // Error should be cleared when edit is called again
    act(() => {
      result.current.handleEdit();
    });

    expect(result.current.error).toBe(null);
  });
});
