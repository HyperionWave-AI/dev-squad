import React, { useState, useEffect } from 'react';
import { Button } from '../atoms/Button';
import { Plus, MessageSquare, Trash2, Calendar, Clock, MoreVertical, Edit2 } from 'lucide-react';
import * as Dialog from '@radix-ui/react-dialog';
import { formatDistanceToNow } from 'date-fns';

interface ChatSession {
  id: string;
  title: string;
  lastMessage?: string;
  timestamp: Date;
  messageCount: number;
}

interface SessionListProps {
  sessions: ChatSession[];
  currentSessionId?: string;
  onSessionSelect: (sessionId: string) => void;
  onNewChat: () => void;
  onDeleteSession: (sessionId: string) => void;
  onDeleteAllSessions: () => void;
  onRenameSession: (sessionId: string, newTitle: string) => void;
  isLoading?: boolean;
}

export const SessionList: React.FC<SessionListProps> = ({
  sessions,
  currentSessionId,
  onSessionSelect,
  onNewChat,
  onDeleteSession,
  onDeleteAllSessions,
  onRenameSession,
  isLoading = false,
}) => {
  const [isNewDialogOpen, setIsNewDialogOpen] = useState(false);
  const [isDeleteAllDialogOpen, setIsDeleteAllDialogOpen] = useState(false);
  const [editingSessionId, setEditingSessionId] = useState<string | null>(null);
  const [editingTitle, setEditingTitle] = useState('');
  const [dropdownSessionId, setDropdownSessionId] = useState<string | null>(null);

  const handleNewChat = () => {
    setIsNewDialogOpen(false);
    onNewChat();
  };

  const handleDeleteAll = () => {
    setIsDeleteAllDialogOpen(false);
    onDeleteAllSessions();
  };

  const handleRename = (sessionId: string, currentTitle: string) => {
    setEditingSessionId(sessionId);
    setEditingTitle(currentTitle);
    setDropdownSessionId(null);
  };

  const handleRenameSubmit = (sessionId: string) => {
    if (editingTitle.trim()) {
      onRenameSession(sessionId, editingTitle.trim());
    }
    setEditingSessionId(null);
    setEditingTitle('');
  };

  const handleRenameCancel = () => {
    setEditingSessionId(null);
    setEditingTitle('');
  };

  const handleDeleteSession = (sessionId: string) => {
    onDeleteSession(sessionId);
    setDropdownSessionId(null);
  };

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = () => {
      setDropdownSessionId(null);
    };

    if (dropdownSessionId) {
      document.addEventListener('click', handleClickOutside);
      return () => document.removeEventListener('click', handleClickOutside);
    }
  }, [dropdownSessionId]);

  return (
    <div className="flex flex-col h-full bg-white dark:bg-gray-900 border-r border-gray-200 dark:border-gray-700">
      {/* Header */}
      <div className="p-4 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
            Chat Sessions
          </h2>
          <span className="text-sm text-gray-500 dark:text-gray-400">
            {sessions.length}
          </span>
        </div>

        {/* Action Buttons */}
        <div className="flex flex-col gap-2">
          <Button
            onClick={() => setIsNewDialogOpen(true)}
            variant="primary"
            size="sm"
            aria-label="New chat"
          >
            <Plus className="w-4 h-4 mr-2" />
            New Chat
          </Button>
          
          {sessions.length > 0 && (
            <Button
              onClick={() => setIsDeleteAllDialogOpen(true)}
              variant="outline"
              size="sm"
              className="text-red-600 hover:text-red-700 hover:bg-red-50"
              aria-label="Delete all chats"
            >
              <Trash2 className="w-4 h-4 mr-2" />
              Delete All
            </Button>
          )}
        </div>
      </div>

      {/* Session List */}
      <div className="flex-1 overflow-y-auto">
        {sessions.length === 0 ? (
          <div className="p-4 text-center text-gray-500 dark:text-gray-400">
            <MessageSquare className="w-12 h-12 mx-auto mb-2 opacity-50" />
            <p className="text-sm">No chat sessions yet</p>
            <p className="text-xs mt-1">Create your first chat to get started</p>
          </div>
        ) : (
          <div className="p-2">
            {sessions.map((session) => (
              <div
                key={session.id}
                className={`relative group rounded-lg p-3 mb-2 cursor-pointer transition-colors ${
                  currentSessionId === session.id
                    ? 'bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-700'
                    : 'hover:bg-gray-50 dark:hover:bg-gray-800'
                }`}
                onClick={() => onSessionSelect(session.id)}
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1 min-w-0">
                    {editingSessionId === session.id ? (
                      <div className="flex items-center gap-2" onClick={(e) => e.stopPropagation()}>
                        <input
                          type="text"
                          value={editingTitle}
                          onChange={(e) => setEditingTitle(e.target.value)}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter') {
                              handleRenameSubmit(session.id);
                            } else if (e.key === 'Escape') {
                              handleRenameCancel();
                            }
                          }}
                          onBlur={() => handleRenameSubmit(session.id)}
                          className="flex-1 px-2 py-1 text-sm border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
                          autoFocus
                        />
                      </div>
                    ) : (
                      <>
                        <h3 className="font-medium text-gray-900 dark:text-white truncate">
                          {session.title}
                        </h3>
                        {session.lastMessage && (
                          <p className="text-sm text-gray-500 dark:text-gray-400 truncate mt-1">
                            {session.lastMessage}
                          </p>
                        )}
                        <div className="flex items-center gap-3 mt-2 text-xs text-gray-400 dark:text-gray-500">
                          <div className="flex items-center gap-1">
                            <Clock className="w-3 h-3" />
                            {formatDistanceToNow(session.timestamp, { addSuffix: true })}
                          </div>
                          <div className="flex items-center gap-1">
                            <MessageSquare className="w-3 h-3" />
                            {session.messageCount}
                          </div>
                        </div>
                      </>
                    )}
                  </div>

                  {editingSessionId !== session.id && (
                    <div className="relative">
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          setDropdownSessionId(
                            dropdownSessionId === session.id ? null : session.id
                          );
                        }}
                        className="opacity-0 group-hover:opacity-100 p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-700 transition-opacity"
                        aria-label="Session options"
                      >
                        <MoreVertical className="w-4 h-4" />
                      </button>

                      {dropdownSessionId === session.id && (
                        <div className="absolute right-0 top-8 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-md shadow-lg z-10 min-w-[120px]">
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              handleRename(session.id, session.title);
                            }}
                            className="w-full px-3 py-2 text-left text-sm hover:bg-gray-50 dark:hover:bg-gray-700 flex items-center gap-2"
                          >
                            <Edit2 className="w-3 h-3" />
                            Rename
                          </button>
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              handleDeleteSession(session.id);
                            }}
                            className="w-full px-3 py-2 text-left text-sm text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 flex items-center gap-2"
                          >
                            <Trash2 className="w-3 h-3" />
                            Delete
                          </button>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* New Chat Dialog */}
      <Dialog.Root open={isNewDialogOpen} onOpenChange={setIsNewDialogOpen}>
        <Dialog.Portal>
          <Dialog.Overlay className="fixed inset-0 bg-black/50 z-50" />
          <Dialog.Content className="fixed top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 bg-white dark:bg-gray-800 rounded-lg shadow-xl z-50 w-full max-w-md mx-4">
            <div className="p-6">
              <Dialog.Title className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
                Start New Chat
              </Dialog.Title>
              <Dialog.Description className="text-sm text-gray-600 dark:text-gray-300 mb-6">
                Create a new chat session to start a fresh conversation.
              </Dialog.Description>
              <div className="flex justify-end gap-3">
                <Button variant="ghost" disabled={isLoading}>
                  Cancel
                </Button>
                <Button onClick={handleNewChat} variant="primary" disabled={isLoading}>
                  {isLoading ? 'Creating...' : 'Create Chat'}
                </Button>
              </div>
            </div>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>

      {/* Delete All Confirmation Dialog */}
      <Dialog.Root open={isDeleteAllDialogOpen} onOpenChange={setIsDeleteAllDialogOpen}>
        <Dialog.Portal>
          <Dialog.Overlay className="fixed inset-0 bg-black/50 z-50" />
          <Dialog.Content className="fixed top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 bg-white dark:bg-gray-800 rounded-lg shadow-xl z-50 w-full max-w-md mx-4">
            <div className="p-6">
              <Dialog.Title className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
                Delete All Sessions
              </Dialog.Title>
              <Dialog.Description className="text-sm text-gray-600 dark:text-gray-300 mb-6">
                This will permanently delete all {sessions.length} chat sessions and their messages.
                This action cannot be undone.
              </Dialog.Description>
              <div className="flex justify-end gap-3">
                <Button variant="ghost" disabled={isLoading}>
                  Cancel
                </Button>
                <Button onClick={handleDeleteAll} variant="danger" disabled={isLoading}>
                  {isLoading ? 'Deleting...' : 'Delete All'}
                </Button>
              </div>
            </div>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </div>
  );
};