import React, { useState, useEffect } from 'react';
import { Add, Delete, ChatBubble, MoreVert, Edit } from '@mui/icons-material';

// Map Material-UI icons to the variable names used in the template
const Plus = Add;
const Trash2 = Delete;
const MessageSquare = ChatBubble;
const MoreVertical = MoreVert;
const Edit2 = Edit;

interface Chat {
  id: string;
  name: string;
  created_at: string;
}

interface SubchatListProps {
  parentChatId: string;
  onSubchatClick: (subchatId: string) => Promise<void>;
  onSubchatCreated: () => Promise<void>;
}

export const SubchatList: React.FC<SubchatListProps> = ({
  parentChatId,
  onSubchatClick,
  onSubchatCreated
}) => {
  const [chats, setChats] = useState<Chat[]>([]);
  const [editingChatId, setEditingChatId] = useState<string | null>(null);
  const [editingName, setEditingName] = useState('');
  const [dropdownOpen, setDropdownOpen] = useState<string | null>(null);

  // Load subchats for the parent chat
  const loadSubchats = async () => {
    try {
      // This would typically fetch subchats from an API
      // For now, using empty array as placeholder
      setChats([]);
    } catch (error) {
      console.error('Failed to load subchats:', error);
    }
  };

  // Auto-refresh when tab becomes visible
  useEffect(() => {
    loadSubchats();

    const handleVisibilityChange = () => {
      if (!document.hidden) {
        loadSubchats(); // Background refresh when tab becomes visible
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => document.removeEventListener('visibilitychange', handleVisibilityChange);
  }, [parentChatId]);

  const handleRename = (chatId: string, currentName: string) => {
    setEditingChatId(chatId);
    setEditingName(currentName);
    setDropdownOpen(null);
  };

  const handleRenameSubmit = (chatId: string) => {
    if (editingName.trim() && editingName !== chats.find(c => c.id === chatId)?.name) {
      // Handle rename logic here
      console.log('Renaming chat', chatId, 'to', editingName.trim());
    }
    setEditingChatId(null);
    setEditingName('');
  };

  const handleRenameCancel = () => {
    setEditingChatId(null);
    setEditingName('');
  };

  const handleDeleteChat = (chatId: string) => {
    // Handle delete logic here
    console.log('Deleting chat', chatId);
    setChats(prev => prev.filter(chat => chat.id !== chatId));
  };

  const handleDeleteAll = () => {
    if (chats.length === 0) return;
    
    if (window.confirm(`Are you sure you want to delete all ${chats.length} chats? This action cannot be undone.`)) {
      // Delete all chats
      chats.forEach(chat => handleDeleteChat(chat.id));
    }
  };

  const handleNewChat = async () => {
    try {
      // Create new subchat logic would go here
      console.log('Creating new subchat for parent:', parentChatId);
      await onSubchatCreated();
    } catch (error) {
      console.error('Failed to create new chat:', error);
    }
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffTime = Math.abs(now.getTime() - date.getTime());
    const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));

    if (diffDays === 1) {
      return 'Today';
    } else if (diffDays === 2) {
      return 'Yesterday';
    } else if (diffDays <= 7) {
      return `${diffDays - 1} days ago`;
    } else {
      return date.toLocaleDateString();
    }
  };

  return (
    <div className="w-80 bg-gray-50 border-r border-gray-200 flex flex-col h-full">
      <div className="p-4 border-b border-gray-200">
        <div className="flex gap-2">
          <button
            onClick={handleNewChat}
            className="flex items-center gap-2 px-4 py-2 bg-yellow-400 hover:bg-yellow-500 text-black rounded-lg font-medium transition-colors"
          >
            <Plus className="w-4 h-4" />
            New Chat
          </button>
          <button
            onClick={handleDeleteAll}
            disabled={chats.length === 0}
            className="flex items-center gap-2 px-4 py-2 bg-yellow-400 hover:bg-yellow-500 disabled:bg-yellow-200 disabled:cursor-not-allowed text-black rounded-lg font-medium transition-colors"
          >
            <Trash2 className="w-4 h-4" />
            Delete All
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        {chats.length === 0 ? (
          <div className="p-4 text-center text-gray-500">
            <MessageSquare className="w-12 h-12 mx-auto mb-2 opacity-50" />
            <p>No chats yet</p>
            <p className="text-sm">Start a new conversation</p>
          </div>
        ) : (
          <div className="p-2">
            {chats.map((chat) => (
              <div
                key={chat.id}
                className="group relative p-3 mb-2 rounded-lg cursor-pointer transition-colors hover:bg-gray-100"
                onClick={() => onSubchatClick(chat.id)}
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1 min-w-0">
                    {editingChatId === chat.id ? (
                      <input
                        type="text"
                        value={editingName}
                        onChange={(e) => setEditingName(e.target.value)}
                        onBlur={() => handleRenameSubmit(chat.id)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') {
                            handleRenameSubmit(chat.id);
                          } else if (e.key === 'Escape') {
                            handleRenameCancel();
                          }
                        }}
                        className="w-full px-2 py-1 text-sm font-medium bg-white border border-blue-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
                        autoFocus
                        onClick={(e) => e.stopPropagation()}
                      />
                    ) : (
                      <h3 className="font-medium text-gray-900 truncate">
                        {chat.name}
                      </h3>
                    )}
                    <p className="text-sm text-gray-500 mt-1">
                      {formatDate(chat.created_at)}
                    </p>
                  </div>
                  
                  <div className="relative ml-2">
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        setDropdownOpen(dropdownOpen === chat.id ? null : chat.id);
                      }}
                      className="p-1 rounded hover:bg-gray-200 opacity-0 group-hover:opacity-100 transition-opacity"
                    >
                      <MoreVertical className="w-4 h-4" />
                    </button>
                    
                    {dropdownOpen === chat.id && (
                      <div className="absolute right-0 top-8 bg-white border border-gray-200 rounded-lg shadow-lg z-10 min-w-[120px]">
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleRename(chat.id, chat.name);
                          }}
                          className="flex items-center gap-2 w-full px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 rounded-t-lg"
                        >
                          <Edit2 className="w-3 h-3" />
                          Rename
                        </button>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleDeleteChat(chat.id);
                            setDropdownOpen(null);
                          }}
                          className="flex items-center gap-2 w-full px-3 py-2 text-sm text-red-600 hover:bg-red-50 rounded-b-lg"
                        >
                          <Trash2 className="w-3 h-3" />
                          Delete
                        </button>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};