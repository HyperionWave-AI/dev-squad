/**
 * SessionList Organism
 *
 * Chat session management component with:
 * - List of chat sessions with active highlighting
 * - New session dialog (Radix Dialog)
 * - Session actions via dropdown menu (rename, delete)
 * - Delete all sessions confirmation
 * - Subchat indicator (parentChatId)
 */

import React, { useState } from 'react';
import * as Dialog from '@radix-ui/react-dialog';
import * as DropdownMenu from '@radix-ui/react-dropdown-menu';
import { Plus, MoreVertical, Trash2, Edit2, MessageSquare, X } from 'lucide-react';
import { cn } from '@/utils';
import { Button } from '@/components/atoms/Button';
import { Input } from '@/components/atoms/Input';
import { Label } from '@/components/atoms/Label';
import { Badge } from '@/components/atoms/Badge';
import type { ChatSession } from '@/services/chatService';

export interface SessionListProps {
  sessions: ChatSession[];
  activeSessionId: string | null;
  onSessionSelect: (sessionId: string) => void;
  onNewChat: () => Promise<void>;
  onDeleteSession: (sessionId: string) => Promise<void>;
  onDeleteAllSessions: () => Promise<void>;
  onRenameSession: (sessionId: string, newTitle: string) => Promise<void>;
  className?: string;
}

// Helper to organize sessions into parent-child hierarchy
const organizeSessionsHierarchy = (sessions: ChatSession[]) => {
  const sessionMap = new Map<string, ChatSession & { children: ChatSession[] }>();
  const rootSessions: (ChatSession & { children: ChatSession[] })[] = [];

  // Create map with all sessions
  sessions.forEach(session => {
    sessionMap.set(session.id, { ...session, children: [] });
  });

  // Build parent-child relationships
  sessions.forEach(session => {
    const node = sessionMap.get(session.id)!;

    if (session.parentChatId && sessionMap.has(session.parentChatId)) {
      // This is a subchat with a valid parent
      const parentNode = sessionMap.get(session.parentChatId)!;
      parentNode.children.push(node);
    } else {
      // This is a root session
      rootSessions.push(node);
    }
  });

  // Sort root sessions by creation date (newest first)
  rootSessions.sort((a, b) =>
    new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
  );

  // Sort children by creation date (oldest first - chronological)
  rootSessions.forEach(parent => {
    parent.children.sort((a, b) =>
      new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime()
    );
  });

  return rootSessions;
};

export const SessionList: React.FC<SessionListProps> = ({
  sessions,
  activeSessionId,
  onSessionSelect,
  onNewChat,
  onDeleteSession,
  onDeleteAllSessions,
  onRenameSession,
  className,
}) => {
  const [isNewDialogOpen, setIsNewDialogOpen] = useState(false);
  const [isDeleteAllDialogOpen, setIsDeleteAllDialogOpen] = useState(false);
  const [renameSessionId, setRenameSessionId] = useState<string | null>(null);
  const [renameValue, setRenameValue] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  // Organize sessions into hierarchy
  const organizedSessions = React.useMemo(
    () => organizeSessionsHierarchy(sessions),
    [sessions]
  );

  const handleNewChat = async () => {
    setIsLoading(true);
    try {
      await onNewChat();
      setIsNewDialogOpen(false);
    } catch (error) {
      console.error('[SessionList] Error creating new chat:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleDeleteAll = async () => {
    setIsLoading(true);
    try {
      await onDeleteAllSessions();
      setIsDeleteAllDialogOpen(false);
    } catch (error) {
      console.error('[SessionList] Error deleting all sessions:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleRename = async (sessionId: string) => {
    if (!renameValue.trim()) return;

    setIsLoading(true);
    try {
      await onRenameSession(sessionId, renameValue.trim());
      setRenameSessionId(null);
      setRenameValue('');
    } catch (error) {
      console.error('[SessionList] Error renaming session:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleDelete = async (sessionId: string) => {
    setIsLoading(true);
    try {
      await onDeleteSession(sessionId);
    } catch (error) {
      console.error('[SessionList] Error deleting session:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const startRename = (session: ChatSession) => {
    setRenameSessionId(session.id);
    setRenameValue(session.title);
  };

  return (
    <div
      className={cn(
        'flex flex-col h-full border-r border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800',
        className
      )}
    >
      {/* Header with New Chat Button */}
      <div className="p-4 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Chat Sessions
          </h2>
          <Button
            onClick={() => setIsNewDialogOpen(true)}
            variant="primary"
            size="sm"
            aria-label="New chat"
          >
            <Plus className="w-4 h-4" />
          </Button>
        </div>

        {sessions.length > 1 && (
          <Button
            onClick={() => setIsDeleteAllDialogOpen(true)}
            variant="outline"
            size="sm"
            className="w-full text-red-600 hover:text-red-700 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20"
          >
            <Trash2 className="w-4 h-4 mr-2" />
            Delete All
          </Button>
        )}
      </div>

      {/* Session List */}
      <div className="flex-1 overflow-y-auto">
        {sessions.length === 0 ? (
          <div className="p-4 text-center text-gray-500 dark:text-gray-400">
            <MessageSquare className="w-12 h-12 mx-auto mb-2 opacity-50" />
            <p className="text-sm">No chat sessions yet</p>
            <p className="text-xs mt-1">Create a new chat to get started</p>
          </div>
        ) : (
          <div className="p-2 space-y-1">
            {organizedSessions.map((parentSession) => {
              // Render helper for a single session
              const renderSession = (
                session: ChatSession & { children?: ChatSession[] },
                isChild: boolean = false
              ) => {
                const isActive = session.id === activeSessionId;
                const isSubchat = !!session.parentChatId || session.title.startsWith('Subchat:');
                const isRenaming = renameSessionId === session.id;

                return (
                  <div
                    key={session.id}
                    className={cn(
                      'group relative rounded-lg transition-colors',
                      isChild && 'ml-6 border-l-2 border-gray-300 dark:border-gray-600 pl-3',
                      isActive
                        ? 'bg-primary-100 dark:bg-primary-900/30'
                        : 'hover:bg-gray-100 dark:hover:bg-gray-700/50'
                    )}
                  >
                  {isRenaming ? (
                    // Rename Input
                    <div className="p-2">
                      <Input
                        value={renameValue}
                        onChange={(e) => setRenameValue(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') {
                            handleRename(session.id);
                          } else if (e.key === 'Escape') {
                            setRenameSessionId(null);
                            setRenameValue('');
                          }
                        }}
                        placeholder="Session name"
                        className="text-sm"
                        autoFocus
                        disabled={isLoading}
                      />
                      <div className="flex gap-2 mt-2">
                        <Button
                          onClick={() => handleRename(session.id)}
                          variant="primary"
                          size="sm"
                          disabled={isLoading || !renameValue.trim()}
                        >
                          Save
                        </Button>
                        <Button
                          onClick={() => {
                            setRenameSessionId(null);
                            setRenameValue('');
                          }}
                          variant="ghost"
                          size="sm"
                          disabled={isLoading}
                        >
                          Cancel
                        </Button>
                      </div>
                    </div>
                  ) : (
                    // Session Item
                    <div
                      className="flex items-center gap-2 p-3 cursor-pointer"
                      onClick={() => !isLoading && onSessionSelect(session.id)}
                      role="button"
                      tabIndex={0}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault();
                          !isLoading && onSessionSelect(session.id);
                        }
                      }}
                      aria-label={`Select ${session.title}`}
                      aria-current={isActive ? 'true' : undefined}
                    >
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <p
                            className={cn(
                              'text-sm font-medium truncate',
                              isActive
                                ? 'text-primary-700 dark:text-primary-300'
                                : 'text-gray-900 dark:text-gray-100'
                            )}
                          >
                            {session.title}
                          </p>
                          {isSubchat && (
                            <Badge variant="outline" className="text-xs shrink-0">
                              Subchat
                            </Badge>
                          )}
                        </div>
                        <p className="text-xs text-gray-500 dark:text-gray-400">
                          {new Date(session.createdAt).toLocaleDateString()}
                        </p>
                      </div>

                      {/* Actions Dropdown */}
                      <DropdownMenu.Root>
                        <DropdownMenu.Trigger asChild>
                          <button
                            className={cn(
                              'p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors',
                              'opacity-0 group-hover:opacity-100 focus:opacity-100'
                            )}
                            onClick={(e) => e.stopPropagation()}
                            aria-label="Session actions"
                          >
                            <MoreVertical className="w-4 h-4" />
                          </button>
                        </DropdownMenu.Trigger>

                        <DropdownMenu.Portal>
                          <DropdownMenu.Content
                            className="min-w-[180px] bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 p-1 z-50"
                            sideOffset={5}
                            onClick={(e) => e.stopPropagation()}
                          >
                            <DropdownMenu.Item
                              className="flex items-center gap-2 px-3 py-2 text-sm rounded hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer outline-none"
                              onSelect={() => startRename(session)}
                            >
                              <Edit2 className="w-4 h-4" />
                              Rename
                            </DropdownMenu.Item>

                            <DropdownMenu.Item
                              className="flex items-center gap-2 px-3 py-2 text-sm text-red-600 dark:text-red-400 rounded hover:bg-red-50 dark:hover:bg-red-900/20 cursor-pointer outline-none"
                              onSelect={() => handleDelete(session.id)}
                            >
                              <Trash2 className="w-4 h-4" />
                              Delete
                            </DropdownMenu.Item>
                          </DropdownMenu.Content>
                        </DropdownMenu.Portal>
                      </DropdownMenu.Root>
                    </div>
                  )}
                </div>
              );
            };

            // Return parent session and its children
            return (
              <React.Fragment key={parentSession.id}>
                {renderSession(parentSession, false)}
                {parentSession.children?.map((childSession) =>
                  renderSession(childSession, true)
                )}
              </React.Fragment>
            );
          })}
          </div>
        )}
      </div>

      {/* New Chat Dialog */}
      <Dialog.Root open={isNewDialogOpen} onOpenChange={setIsNewDialogOpen}>
        <Dialog.Portal>
          <Dialog.Overlay className="fixed inset-0 bg-black/50 z-40 animate-in fade-in" />
          <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md p-6 z-50 animate-in fade-in zoom-in-95">
            <Dialog.Title className="text-xl font-semibold mb-4 text-gray-900 dark:text-gray-100">
              Create New Chat
            </Dialog.Title>
            <Dialog.Description className="text-sm text-gray-600 dark:text-gray-400 mb-4">
              Start a new chat session with the AI assistant.
            </Dialog.Description>

            <div className="flex gap-3 justify-end mt-6">
              <Dialog.Close asChild>
                <Button variant="ghost" disabled={isLoading}>
                  Cancel
                </Button>
              </Dialog.Close>
              <Button onClick={handleNewChat} variant="primary" disabled={isLoading}>
                {isLoading ? 'Creating...' : 'Create Chat'}
              </Button>
            </div>

            <Dialog.Close asChild>
              <button
                className="absolute top-4 right-4 p-1 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
                aria-label="Close"
              >
                <X className="w-5 h-5" />
              </button>
            </Dialog.Close>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>

      {/* Delete All Confirmation Dialog */}
      <Dialog.Root open={isDeleteAllDialogOpen} onOpenChange={setIsDeleteAllDialogOpen}>
        <Dialog.Portal>
          <Dialog.Overlay className="fixed inset-0 bg-black/50 z-40 animate-in fade-in" />
          <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md p-6 z-50 animate-in fade-in zoom-in-95">
            <Dialog.Title className="text-xl font-semibold mb-4 text-gray-900 dark:text-gray-100">
              Delete All Sessions?
            </Dialog.Title>
            <Dialog.Description className="text-sm text-gray-600 dark:text-gray-400 mb-4">
              This will permanently delete all {sessions.length} chat sessions and their messages.
              This action cannot be undone.
            </Dialog.Description>

            <div className="flex gap-3 justify-end mt-6">
              <Dialog.Close asChild>
                <Button variant="ghost" disabled={isLoading}>
                  Cancel
                </Button>
              </Dialog.Close>
              <Button onClick={handleDeleteAll} variant="danger" disabled={isLoading}>
                {isLoading ? 'Deleting...' : 'Delete All'}
              </Button>
            </div>

            <Dialog.Close asChild>
              <button
                className="absolute top-4 right-4 p-1 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
                aria-label="Close"
              >
                <X className="w-5 h-5" />
              </button>
            </Dialog.Close>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </div>
  );
};

SessionList.displayName = 'SessionList';
