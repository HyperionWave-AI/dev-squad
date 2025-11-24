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

  // Context management helpers
  const updateContextMetadata = useCallback(
    (metadata: typeof state.contextMetadata) => {
      manager.setState({
        contextMetadata: metadata,
        lastContextUpdate: Date.now(),
      });
    },
    [manager]
  );

  const setContextWarningShown = useCallback(
    (shown: boolean) => {
      manager.setField('contextWarningShown', shown);
    },
    [manager]
  );

  const setContextCriticalShown = useCallback(
    (shown: boolean) => {
      manager.setField('contextCriticalShown', shown);
    },
    [manager]
  );

  const setContextFull = useCallback(
    (isFull: boolean) => {
      manager.setField('isContextFull', isFull);
    },
    [manager]
  );

  const getContextPercentage = useCallback(() => {
    if (!state.contextMetadata) return 0;
    return state.contextMetadata.percentageUsed || 0;
  }, [state.contextMetadata]);

  const isContextNearLimit = useCallback(() => {
    const percentage = getContextPercentage();
    return percentage >= 80;
  }, [getContextPercentage]);

  const isContextAtLimit = useCallback(() => {
    const percentage = getContextPercentage();
    return percentage >= 100;
  }, [getContextPercentage]);

  return {
    state,
    setState: updateState,
    setField,
    rollback,
    reset,
    manager,
    // Context helpers
    updateContextMetadata,
    setContextWarningShown,
    setContextCriticalShown,
    setContextFull,
    getContextPercentage,
    isContextNearLimit,
    isContextAtLimit,
  };
}
