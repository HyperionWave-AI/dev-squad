/**
 * useChatState Hook
 * 
 * React hook for integrating ChatStateManager with React components.
 * Provides state and setState functions that sync with the centralized state manager.
 * 
 * Usage:
 * const { state, setState, setField } = useChatState();
 */

import { useEffect, useState, useCallback } from 'react';
import { type ChatState, getChatStateManager } from '@/services/ChatStateManager';

export function useChatState() {
  const manager = getChatStateManager();
  const [state, setState] = useState<ChatState>(manager.getState() as ChatState);

  // Subscribe to state changes
  useEffect(() => {
    const unsubscribe = manager.subscribe((newState) => {
      setState(newState as ChatState);
    });

    return unsubscribe;
  }, [manager]);

  // Provide setState wrapper
  const updateState = useCallback(
    (updates: Partial<ChatState>) => {
      manager.setState(updates);
    },
    [manager]
  );

  // Provide setField wrapper
  const setField = useCallback(
    <K extends keyof ChatState>(key: K, value: ChatState[K]) => {
      manager.setField(key, value);
    },
    [manager]
  );

  // Provide rollback function
  const rollback = useCallback(() => {
    manager.rollback();
  }, [manager]);

  // Provide reset function
  const reset = useCallback(() => {
    manager.reset();
  }, [manager]);

  return {
    state,
    setState: updateState,
    setField,
    rollback,
    reset,
    manager,
  };
}
