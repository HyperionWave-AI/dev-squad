import { renderHook } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { useKeyboardShortcuts } from '../useKeyboardShortcuts';
import type { KeyboardShortcut } from '../useKeyboardShortcuts';

describe('useKeyboardShortcuts', () => {
  let handlers: {
    handler1: ReturnType<typeof vi.fn>;
    handler2: ReturnType<typeof vi.fn>;
    handler3: ReturnType<typeof vi.fn>;
  };

  beforeEach(() => {
    handlers = {
      handler1: vi.fn(),
      handler2: vi.fn(),
      handler3: vi.fn(),
    };
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('should call handler when matching key is pressed', () => {
    const shortcuts: KeyboardShortcut[] = [
      { key: 'k', handler: handlers.handler1 },
    ];

    renderHook(() => useKeyboardShortcuts(shortcuts));

    // Simulate 'k' key press
    const event = new KeyboardEvent('keydown', { key: 'k' });
    window.dispatchEvent(event);

    expect(handlers.handler1).toHaveBeenCalledTimes(1);
  });

  it('should call handler when key + metaKey combination is pressed', () => {
    const shortcuts: KeyboardShortcut[] = [
      { key: 'k', metaKey: true, handler: handlers.handler1 },
    ];

    renderHook(() => useKeyboardShortcuts(shortcuts));

    // Simulate 'Cmd+k' (or Ctrl+k on Windows/Linux)
    const event = new KeyboardEvent('keydown', { key: 'k', metaKey: true });
    window.dispatchEvent(event);

    expect(handlers.handler1).toHaveBeenCalledTimes(1);
  });

  it('should call handler when key + ctrlKey combination is pressed', () => {
    const shortcuts: KeyboardShortcut[] = [
      { key: 's', ctrlKey: true, handler: handlers.handler1 },
    ];

    renderHook(() => useKeyboardShortcuts(shortcuts));

    // Simulate 'Ctrl+s'
    const event = new KeyboardEvent('keydown', { key: 's', ctrlKey: true });
    window.dispatchEvent(event);

    expect(handlers.handler1).toHaveBeenCalledTimes(1);
  });

  it('should not call handler when key does not match', () => {
    const shortcuts: KeyboardShortcut[] = [
      { key: 'k', handler: handlers.handler1 },
    ];

    renderHook(() => useKeyboardShortcuts(shortcuts));

    // Simulate 'j' key press (different key)
    const event = new KeyboardEvent('keydown', { key: 'j' });
    window.dispatchEvent(event);

    expect(handlers.handler1).not.toHaveBeenCalled();
  });

  it('should not call handler when modifier key does not match', () => {
    const shortcuts: KeyboardShortcut[] = [
      { key: 'k', metaKey: true, handler: handlers.handler1 },
    ];

    renderHook(() => useKeyboardShortcuts(shortcuts));

    // Simulate 'k' without metaKey
    const event = new KeyboardEvent('keydown', { key: 'k' });
    window.dispatchEvent(event);

    expect(handlers.handler1).not.toHaveBeenCalled();
  });

  it('should handle multiple shortcuts', () => {
    const shortcuts: KeyboardShortcut[] = [
      { key: 'k', metaKey: true, handler: handlers.handler1 },
      { key: 'Escape', handler: handlers.handler2 },
      { key: 's', ctrlKey: true, handler: handlers.handler3 },
    ];

    renderHook(() => useKeyboardShortcuts(shortcuts));

    // Test first shortcut
    let event = new KeyboardEvent('keydown', { key: 'k', metaKey: true });
    window.dispatchEvent(event);
    expect(handlers.handler1).toHaveBeenCalledTimes(1);
    expect(handlers.handler2).not.toHaveBeenCalled();
    expect(handlers.handler3).not.toHaveBeenCalled();

    // Reset mocks
    vi.clearAllMocks();

    // Test second shortcut
    event = new KeyboardEvent('keydown', { key: 'Escape' });
    window.dispatchEvent(event);
    expect(handlers.handler1).not.toHaveBeenCalled();
    expect(handlers.handler2).toHaveBeenCalledTimes(1);
    expect(handlers.handler3).not.toHaveBeenCalled();

    // Reset mocks
    vi.clearAllMocks();

    // Test third shortcut
    event = new KeyboardEvent('keydown', { key: 's', ctrlKey: true });
    window.dispatchEvent(event);
    expect(handlers.handler1).not.toHaveBeenCalled();
    expect(handlers.handler2).not.toHaveBeenCalled();
    expect(handlers.handler3).toHaveBeenCalledTimes(1);
  });

  it('should be case-insensitive for key matching', () => {
    const shortcuts: KeyboardShortcut[] = [
      { key: 'k', handler: handlers.handler1 },
    ];

    renderHook(() => useKeyboardShortcuts(shortcuts));

    // Simulate 'K' (uppercase) key press
    const event = new KeyboardEvent('keydown', { key: 'K' });
    window.dispatchEvent(event);

    expect(handlers.handler1).toHaveBeenCalledTimes(1);
  });

  it('should handle Escape key', () => {
    const shortcuts: KeyboardShortcut[] = [
      { key: 'Escape', handler: handlers.handler1 },
    ];

    renderHook(() => useKeyboardShortcuts(shortcuts));

    const event = new KeyboardEvent('keydown', { key: 'Escape' });
    window.dispatchEvent(event);

    expect(handlers.handler1).toHaveBeenCalledTimes(1);
  });

  it('should remove event listener on unmount', () => {
    const shortcuts: KeyboardShortcut[] = [
      { key: 'k', handler: handlers.handler1 },
    ];

    const { unmount } = renderHook(() => useKeyboardShortcuts(shortcuts));

    // Verify handler works before unmount
    let event = new KeyboardEvent('keydown', { key: 'k' });
    window.dispatchEvent(event);
    expect(handlers.handler1).toHaveBeenCalledTimes(1);

    // Unmount
    unmount();

    // Clear mock
    vi.clearAllMocks();

    // Verify handler no longer works after unmount
    event = new KeyboardEvent('keydown', { key: 'k' });
    window.dispatchEvent(event);
    expect(handlers.handler1).not.toHaveBeenCalled();
  });

  it('should handle ctrlKey on Windows/Linux for metaKey shortcuts', () => {
    const shortcuts: KeyboardShortcut[] = [
      { key: 'k', metaKey: true, handler: handlers.handler1 },
    ];

    renderHook(() => useKeyboardShortcuts(shortcuts));

    // Simulate 'Ctrl+k' on Windows/Linux (ctrlKey instead of metaKey)
    const event = new KeyboardEvent('keydown', { key: 'k', ctrlKey: true });
    window.dispatchEvent(event);

    expect(handlers.handler1).toHaveBeenCalledTimes(1);
  });

  it('should handle shift + key combinations', () => {
    const shortcuts: KeyboardShortcut[] = [
      { key: 'K', shiftKey: true, handler: handlers.handler1 },
    ];

    renderHook(() => useKeyboardShortcuts(shortcuts));

    const event = new KeyboardEvent('keydown', { key: 'K', shiftKey: true });
    window.dispatchEvent(event);

    expect(handlers.handler1).toHaveBeenCalledTimes(1);
  });

  it('should handle alt + key combinations', () => {
    const shortcuts: KeyboardShortcut[] = [
      { key: 'a', altKey: true, handler: handlers.handler1 },
    ];

    renderHook(() => useKeyboardShortcuts(shortcuts));

    const event = new KeyboardEvent('keydown', { key: 'a', altKey: true });
    window.dispatchEvent(event);

    expect(handlers.handler1).toHaveBeenCalledTimes(1);
  });
});
