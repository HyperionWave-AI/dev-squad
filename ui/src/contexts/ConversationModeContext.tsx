/**
 * Conversation Mode Context
 *
 * Manages the user's conversation display preference (debug vs default mode)
 * - Debug mode: Shows technical details, tool calls, JSON payloads
 * - Default mode: Shows natural language explanations, friendly UI
 *
 * The preference is persisted to the backend and loaded on mount
 */

import React, { createContext, useState, useEffect, useContext, useCallback } from 'react';
import type { ReactNode } from 'react';
import { userSettingsService, type ConversationMode } from '../services/userSettingsService';

interface ConversationModeContextValue {
  mode: ConversationMode;
  setMode: (mode: ConversationMode) => Promise<void>;
  isLoading: boolean;
  error: string | null;
}

const ConversationModeContext = createContext<ConversationModeContextValue | undefined>(undefined);

interface ConversationModeProviderProps {
  children: ReactNode;
}

export const ConversationModeProvider: React.FC<ConversationModeProviderProps> = ({ children }) => {
  const [mode, setModeState] = useState<ConversationMode>('default');
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Load user settings on mount
  useEffect(() => {
    const loadSettings = async () => {
      try {
        setIsLoading(true);
        setError(null);
        const settings = await userSettingsService.getSettings();
        setModeState(settings.conversationMode);
        console.log('[ConversationMode] Loaded user settings:', {
          mode: settings.conversationMode,
          timestamp: new Date().toISOString()
        });
      } catch (err) {
        console.error('[ConversationMode] Failed to load settings:', err);
        setError(err instanceof Error ? err.message : 'Failed to load settings');
        // Default to 'default' mode on error
        setModeState('default');
      } finally {
        setIsLoading(false);
      }
    };

    loadSettings();
  }, []);

  // Update mode and persist to backend
  const setMode = useCallback(async (newMode: ConversationMode) => {
    try {
      setError(null);
      console.log('[ConversationMode] Toggling mode:', {
        from: mode,
        to: newMode,
        timestamp: new Date().toISOString()
      });

      // Optimistically update UI
      setModeState(newMode);

      // Persist to backend
      await userSettingsService.updateConversationMode(newMode);
      console.log('[ConversationMode] Mode updated successfully:', {
        mode: newMode,
        timestamp: new Date().toISOString()
      });
    } catch (err) {
      console.error('[ConversationMode] Failed to update mode:', err);
      setError(err instanceof Error ? err.message : 'Failed to update mode');

      // Revert on error
      try {
        const settings = await userSettingsService.getSettings();
        setModeState(settings.conversationMode);
      } catch {
        // If reload fails, stay with optimistic value
      }
    }
  }, []);

  const value: ConversationModeContextValue = {
    mode,
    setMode,
    isLoading,
    error,
  };

  return (
    <ConversationModeContext.Provider value={value}>
      {children}
    </ConversationModeContext.Provider>
  );
};

/**
 * Hook to access conversation mode context
 * @throws Error if used outside ConversationModeProvider
 */
export const useConversationMode = (): ConversationModeContextValue => {
  const context = useContext(ConversationModeContext);
  if (context === undefined) {
    throw new Error('useConversationMode must be used within a ConversationModeProvider');
  }
  return context;
};
