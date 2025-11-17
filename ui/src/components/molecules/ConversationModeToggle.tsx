/**
 * ConversationModeToggle Component
 *
 * Toggle button to switch between default and debug conversation modes.
 * - Default mode: Shows clean, user-friendly messages (hides tool calls)
 * - Debug mode: Shows technical details, tool calls, and JSON payloads
 */

import React from 'react';
import { Bug, MessageSquare } from 'lucide-react';
import { useConversationMode } from '@/contexts/ConversationModeContext';
import { Button } from '@/components/atoms/Button';
import { cn } from '@/utils';

export interface ConversationModeToggleProps {
  className?: string;
  showLabel?: boolean;
}

export const ConversationModeToggle: React.FC<ConversationModeToggleProps> = ({
  className,
  showLabel = false,
}) => {
  const { mode, setMode, isLoading } = useConversationMode();

  const handleToggle = async () => {
    const newMode = mode === 'default' ? 'debug' : 'default';
    await setMode(newMode);
  };

  const isDebugMode = mode === 'debug';

  return (
    <Button
      variant={isDebugMode ? 'secondary' : 'outline'}
      size="sm"
      onClick={handleToggle}
      disabled={isLoading}
      className={cn('flex items-center gap-2', isLoading && 'opacity-50 cursor-wait', className)}
      title={isDebugMode ? 'Switch to Default Mode' : 'Switch to Debug Mode'}
    >
      {isDebugMode ? (
        <>
          <Bug className="h-4 w-4" />
          {showLabel && <span>Debug</span>}
        </>
      ) : (
        <>
          <MessageSquare className="h-4 w-4" />
          {showLabel && <span>Default</span>}
        </>
      )}
    </Button>
  );
};

ConversationModeToggle.displayName = 'ConversationModeToggle';
