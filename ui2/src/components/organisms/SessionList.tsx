import React, { useState, useEffect } from 'react';
import { Button } from '../atoms/Button';
import { Plus, MessageSquare, Trash2, Clock, MoreVertical, Edit2, Trash } from 'lucide-react';
import * as Dialog from '@radix-ui/react-dialog';
import { formatDistanceToNow } from 'date-fns';

interface ChatSession {
  id: string;
  title: string;
  parentSessionId?: string; // UI2 naming convention
  parentChatId?: string;    // chatService naming convention
  isSubchat?: boolean;
  lastMessage?: string;
  timestamp: Date | string;
  messageCount: number;
  activeSubagentId?: string;
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

// Helper function to get parent ID from either field name
export const getParentId = (session: ChatSession): string | undefined => {
  return session.parentSessionId || session.parentChatId;
};

// Helper function to determine if session is a subchat
const isSubchatSession = (session: ChatSession): boolean => {
  return session.isSubchat || !!getParentId(session);
};

// Tree node structure for hierarchical rendering
interface ChatTreeNode {
  session: ChatSession;
  children: ChatTreeNode[];
  depth: number;
}

// Build hierarchical tree structure from flat sessions array
const buildChatTree = (sessions: ChatSession[]): ChatTreeNode[] => {
  const sessionMap = new Map<string, ChatTreeNode>();
  const rootNodes: ChatTreeNode[] = [];

  // First pass: create all nodes
  sessions.forEach(session => {
    sessionMap.set(session.id, { session, children: [], depth: 0 });
  });

  // Second pass: build parent-child relationships
  sessions.forEach(session => {
    const node = sessionMap.get(session.id)!;
    const parentId = getParentId(session);

    if (parentId && sessionMap.has(parentId)) {
      // This is a subchat with a valid parent
      const parentNode = sessionMap.get(parentId)!;
      parentNode.children.push(node);
    } else {
      // This is a root chat
      rootNodes.push(node);
    }
  });

  // Set depths for all nodes recursively
  const setDepth = (node: ChatTreeNode, depth: number) => {
    node.depth = depth;
    node.children.forEach(child => setDepth(child, depth + 1));
  };

  rootNodes.forEach(node => setDepth(node, 0));

  // Sort root nodes by timestamp (newest first for main sessions)
  rootNodes.sort((a, b) => 
    new Date(b.session.timestamp).getTime() - new Date(a.session.timestamp).getTime()
  );

  // Sort children by timestamp (oldest first for subchats)
  sessionMap.forEach(node => {
    node.children.sort((a, b) => 
      new Date(a.session.timestamp).getTime() - new Date(b.session.timestamp).getTime()
    );
  });

  return rootNodes;
};

// Flatten tree structure for rendering
const flattenTree = (tree: ChatTreeNode[]): { session: ChatSession; depth: number }[] => {
  const result: { session: ChatSession; depth: number }[] = [];

  const traverse = (node: ChatTreeNode) => {
    result.push({ session: node.session, depth: node.depth });
    node.children.forEach(child => traverse(child));
  };

  tree.forEach(node => traverse(node));
  return result;
};

// Helper function to organize sessions into hierarchy (enhanced version)
export const organizeSessionsHierarchy = (sessions: ChatSession[]) => {
  const tree = buildChatTree(sessions);
  const flatSessions = flattenTree(tree);
  
  // Legacy support - maintain old interface for backward compatibility
  const mainSessions: ChatSession[] = [];
  const subchatsMap = new Map<string, ChatSession[]>();
  
  sessions.forEach(session => {
    const parentId = getParentId(session);
    if (isSubchatSession(session) && parentId) {
      if (!subchatsMap.has(parentId)) {
        subchatsMap.set(parentId, []);
      }
      subchatsMap.get(parentId)!.push(session);
    } else {
      mainSessions.push(session);
    }
  });
  
  // Sort subchats by timestamp for each parent
  subchatsMap.forEach(subchats => subchats.sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()));
  
  return { 
    mainSessions, 
    subchatsMap,
    tree,
    flatSessions
  };
};

// Helper function to calculate hierarchy depth for a session
const calculateHierarchyDepth = (sessionId: string, sessions: ChatSession[], visited = new Set<string>()): number => {
  if (visited.has(sessionId)) return 0; // Prevent infinite loops
  visited.add(sessionId);
  
  const session = sessions.find(s => s.id === sessionId);
  if (!session) return 0;
  
  const parentId = getParentId(session);
  if (!parentId) return 0;
  
  return 1 + calculateHierarchyDepth(parentId, sessions, visited);
};

// Helper function to get indentation class based on depth
const getIndentationClass = (depth: number): string => {
  if (depth === 0) return '';
  // Use padding-left with incremental values: pl-6, pl-12, pl-18, pl-24 (max)
  const paddingValue = Math.min(depth * 6, 24);
  return `pl-${paddingValue}`;
};

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

  // Organize sessions into hierarchy using enhanced tree structure
  const { flatSessions } = organizeSessionsHierarchy(sessions);

  // Function to render a single session with proper indentation
  const renderSession = (session: ChatSession, depth = 0) => {
    const indentationClass = getIndentationClass(depth);
    
    return (
      <div
        key={session.id}
        className={`relative group rounded-lg p-3 mb-2 cursor-pointer transition-colors ${
          indentationClass
        } ${
          currentSessionId === session.id
            ? 'bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-700'
            : 'hover:bg-gray-50 dark:hover:bg-gray-800'
        }`}
        onClick={() => onSessionSelect(session.id)}
      >
        {/* Add visual hierarchy indicator for subchats */}
        {depth > 0 && (
          <div 
            className="absolute left-0 top-0 bottom-0 w-0.5 bg-gradient-to-b from-transparent via-blue-300 to-transparent opacity-60"
            style={{ left: `${(depth - 1) * 24 + 12}px` }}
          />
        )}
        
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
                <h3 className={`font-medium truncate ${
                  depth > 0 
                    ? 'text-gray-700 dark:text-gray-300 text-sm' 
                    : 'text-gray-900 dark:text-white'
                }`}>
                  {depth > 0 && '└ '}{session.title}
                </h3>
                {session.lastMessage && (
                  <p className="text-sm text-gray-500 dark:text-gray-400 truncate mt-1">
                    {session.lastMessage}
                  </p>
                )}
                <div className="flex items-center gap-3 mt-2 text-xs text-gray-400 dark:text-gray-500">
                  <div className="flex items-center gap-1">
                    <Clock className="w-3 h-3" />
                    {(() => {
                      try {
                        const date = typeof session.timestamp === 'string'
                          ? new Date(session.timestamp)
                          : session.timestamp;
                        return isNaN(date.getTime())
                          ? 'Invalid date'
                          : formatDistanceToNow(date, { addSuffix: true });
                      } catch {
                        return 'Invalid date';
                      }
                    })()}
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
    );
  };

  const handleNewChat = () => {
    setIsNewDialogOpen(false);
    onNewChat();
  };

  const handleDeleteAll = () => {
    setIsDeleteAllDialogOpen(false);
    onDeleteAllSessions();
  };

  const handleDeleteSession = (sessionId: string) => {
    setDropdownSessionId(null);
    onDeleteSession(sessionId);
  };

  const handleRename = (sessionId: string, currentTitle: string) => {
    setDropdownSessionId(null);
    setEditingSessionId(sessionId);
    setEditingTitle(currentTitle);
  };

  const handleRenameSubmit = (sessionId: string) => {
    if (editingTitle.trim() && editingTitle !== sessions.find(s => s.id === sessionId)?.title) {
      onRenameSession(sessionId, editingTitle.trim());
    }
    handleRenameCancel();
  };

  const handleRenameCancel = () => {
    setEditingSessionId(null);
    setEditingTitle('');
  };

  // Close dropdown when clicking outside
  useEffect(() => {
    if (dropdownSessionId) {
      const handleClickOutside = () => setDropdownSessionId(null);
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

        <Dialog.Root open={isNewDialogOpen} onOpenChange={setIsNewDialogOpen}>
          <Dialog.Trigger asChild>
            <Button
              variant="primary"
              size="sm"
              className="w-full"
              disabled={isLoading}
            >
              <Plus className="w-4 h-4 mr-2" />
              New Chat
            </Button>
          </Dialog.Trigger>
          <Dialog.Portal>
            <Dialog.Overlay className="fixed inset-0 bg-black/50 z-50" />
            <Dialog.Content className="fixed top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 bg-white dark:bg-gray-800 rounded-lg p-6 w-96 z-50">
              <Dialog.Title className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
                Create New Chat
              </Dialog.Title>
              <p className="text-sm text-gray-600 dark:text-gray-400 mb-6">
                Start a new conversation. You can rename it later.
              </p>
              <div className="flex gap-3 justify-end">
                <Dialog.Close asChild>
                  <Button variant="secondary" size="sm">
                    Cancel
                  </Button>
                </Dialog.Close>
                <Button
                  variant="primary"
                  size="sm"
                  onClick={handleNewChat}
                >
                  Create Chat
                </Button>
              </div>
            </Dialog.Content>
          </Dialog.Portal>
        </Dialog.Root>
      </div>

      {/* Sessions List */}
      <div className="flex-1 overflow-y-auto p-4">
        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <div className="text-sm text-gray-500 dark:text-gray-400">
              Loading sessions...
            </div>
          </div>
        ) : sessions.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <MessageSquare className="w-12 h-12 text-gray-300 dark:text-gray-600 mb-4" />
            <p className="text-sm text-gray-500 dark:text-gray-400 mb-2">
              No chat sessions yet
            </p>
            <p className="text-xs text-gray-400 dark:text-gray-500">
              Create your first chat to get started
            </p>
          </div>
        ) : (
          <div className="space-y-1">
            {flatSessions.map(({ session, depth }) => 
              renderSession(session, depth)
            )}
          </div>
        )}
      </div>

      {/* Footer */}
      {sessions.length > 0 && (
        <div className="p-4 border-t border-gray-200 dark:border-gray-700">
          <Dialog.Root open={isDeleteAllDialogOpen} onOpenChange={setIsDeleteAllDialogOpen}>
            <Dialog.Trigger asChild>
              <Button
                variant="secondary"
                size="sm"
                className="w-full text-red-600 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-900/20"
                disabled={isLoading}
              >
                <Trash className="w-4 h-4 mr-2" />
                Delete All Sessions
              </Button>
            </Dialog.Trigger>
            <Dialog.Portal>
              <Dialog.Overlay className="fixed inset-0 bg-black/50 z-50" />
              <Dialog.Content className="fixed top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 bg-white dark:bg-gray-800 rounded-lg p-6 w-96 z-50">
                <Dialog.Title className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
                  Delete All Sessions
                </Dialog.Title>
                <p className="text-sm text-gray-600 dark:text-gray-400 mb-6">
                  Are you sure you want to delete all chat sessions? This action cannot be undone.
                </p>
                <div className="flex gap-3 justify-end">
                  <Dialog.Close asChild>
                    <Button variant="secondary" size="sm">
                      Cancel
                    </Button>
                  </Dialog.Close>
                  <Button
                    variant="primary"
                    size="sm"
                    onClick={handleDeleteAll}
                    className="bg-red-600 hover:bg-red-700 text-white"
                  >
                    Delete All
                  </Button>
                </div>
              </Dialog.Content>
            </Dialog.Portal>
          </Dialog.Root>
        </div>
      )}
    </div>
  );
};