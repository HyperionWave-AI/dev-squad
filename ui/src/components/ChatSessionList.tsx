import React, { Fragment, useState } from 'react';
import {
  TrashIcon,
  PencilIcon,
  EllipsisVerticalIcon,
} from '@heroicons/react/24/outline';
import {
  Menu,
  MenuButton,
  MenuItem,
  MenuItems,
  Transition,
  Dialog,
  DialogPanel,
  DialogTitle,
} from '@headlessui/react';

interface ChatSession {
  id: string;
  title: string;
  lastMessage: string;
  timestamp: string;
  messageCount: number;
}

interface ChatSessionListProps {
  sessions: ChatSession[];
  activeSessionId?: string;
  onSessionSelect: (sessionId: string) => void;
  onSessionDelete: (sessionId: string) => void;
  onSessionRename: (sessionId: string, newTitle: string) => void;
}

const ChatSessionList: React.FC<ChatSessionListProps> = ({
  sessions,
  activeSessionId,
  onSessionSelect,
  onSessionDelete,
  onSessionRename,
}) => {
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [sessionToDelete, setSessionToDelete] = useState<string | null>(null);
  const [isRenameDialogOpen, setIsRenameDialogOpen] = useState(false);
  const [sessionToRename, setSessionToRename] = useState<string | null>(null);
  const [newTitle, setNewTitle] = useState('');

  const handleDeleteClick = (sessionId: string) => {
    setSessionToDelete(sessionId);
    setIsDeleteDialogOpen(true);
  };

  const handleRenameClick = (sessionId: string, currentTitle: string) => {
    setSessionToRename(sessionId);
    setNewTitle(currentTitle);
    setIsRenameDialogOpen(true);
  };

  const confirmDelete = () => {
    if (sessionToDelete) {
      onSessionDelete(sessionToDelete);
      setIsDeleteDialogOpen(false);
      setSessionToDelete(null);
    }
  };

  const confirmRename = () => {
    if (sessionToRename && newTitle.trim()) {
      onSessionRename(sessionToRename, newTitle.trim());
      setIsRenameDialogOpen(false);
      setSessionToRename(null);
      setNewTitle('');
    }
  };

  return (
    <div className="flex flex-col h-full">
      <div className="flex-shrink-0 p-4 border-b border-gray-200 dark:border-gray-700">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Chat Sessions</h2>
        <button
          type="button"
          className="mt-3 w-full flex items-center justify-center px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm bg-white dark:bg-gray-800 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 transition-colors duration-200"
          aria-label="Create new chat session"
          onClick={() => {
            // Placeholder functionality - to be implemented
            console.log('New session button clicked');
          }}
        >
          <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
          </svg>
          New Session
        </button>
      </div>
      <div className="flex-1 overflow-y-auto">
        {sessions.length === 0 ? (
          <div className="p-4 text-center text-gray-500 dark:text-gray-400">
            No chat sessions yet
          </div>
        ) : (
          <div className="p-2">
            {sessions.map((session) => (
              <div
                key={session.id}
                className={`group relative p-3 mb-2 rounded-lg cursor-pointer transition-colors ${
                  activeSessionId === session.id
                    ? 'bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800'
                    : 'hover:bg-gray-50 dark:hover:bg-gray-800 border border-transparent'
                }`}
                onClick={() => onSessionSelect(session.id)}
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1 min-w-0">
                    <h3 className="text-sm font-medium text-gray-900 dark:text-white truncate">
                      {session.title}
                    </h3>
                    <p className="text-xs text-gray-500 dark:text-gray-400 truncate mt-1">
                      {session.lastMessage}
                    </p>
                    <div className="flex items-center mt-2 text-xs text-gray-400 dark:text-gray-500">
                      <span>{session.timestamp}</span>
                      <span className="mx-1">•</span>
                      <span>{session.messageCount} messages</span>
                    </div>
                  </div>
                  <Menu as="div" className="relative ml-2">
                    <MenuButton className="opacity-0 group-hover:opacity-100 p-1 rounded-md hover:bg-gray-200 dark:hover:bg-gray-700 transition-opacity">
                      <EllipsisVerticalIcon className="h-4 w-4 text-gray-500 dark:text-gray-400" />
                    </MenuButton>
                    <Transition
                      as={Fragment}
                      enter="transition ease-out duration-100"
                      enterFrom="transform opacity-0 scale-95"
                      enterTo="transform opacity-100 scale-100"
                      leave="transition ease-in duration-75"
                      leaveFrom="transform opacity-100 scale-100"
                      leaveTo="transform opacity-0 scale-95"
                    >
                      <MenuItems className="absolute right-0 mt-1 w-32 bg-white dark:bg-gray-800 rounded-md shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none z-10">
                        <MenuItem>
                          {({ active }) => (
                            <button
                              className={`${
                                active ? 'bg-gray-100 dark:bg-gray-700' : ''
                              } group flex items-center w-full px-3 py-2 text-sm text-gray-700 dark:text-gray-300`}
                              onClick={(e) => {
                                e.stopPropagation();
                                handleRenameClick(session.id, session.title);
                              }}
                            >
                              <PencilIcon className="h-4 w-4 mr-2" />
                              Rename
                            </button>
                          )}
                        </MenuItem>
                        <MenuItem>
                          {({ active }) => (
                            <button
                              className={`${
                                active ? 'bg-gray-100 dark:bg-gray-700' : ''
                              } group flex items-center w-full px-3 py-2 text-sm text-red-600 dark:text-red-400`}
                              onClick={(e) => {
                                e.stopPropagation();
                                handleDeleteClick(session.id);
                              }}
                            >
                              <TrashIcon className="h-4 w-4 mr-2" />
                              Delete
                            </button>
                          )}
                        </MenuItem>
                      </MenuItems>
                    </Transition>
                  </Menu>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Delete Confirmation Dialog */}
      <Transition appear show={isDeleteDialogOpen} as={Fragment}>
        <Dialog as="div" className="relative z-50" onClose={() => setIsDeleteDialogOpen(false)}>
          <Transition.Child
            as={Fragment}
            enter="ease-out duration-300"
            enterFrom="opacity-0"
            enterTo="opacity-100"
            leave="ease-in duration-200"
            leaveFrom="opacity-100"
            leaveTo="opacity-0"
          >
            <div className="fixed inset-0 bg-black bg-opacity-25" />
          </Transition.Child>

          <div className="fixed inset-0 overflow-y-auto">
            <div className="flex min-h-full items-center justify-center p-4 text-center">
              <Transition.Child
                as={Fragment}
                enter="ease-out duration-300"
                enterFrom="opacity-0 scale-95"
                enterTo="opacity-100 scale-100"
                leave="ease-in duration-200"
                leaveFrom="opacity-100 scale-100"
                leaveTo="opacity-0 scale-95"
              >
                <DialogPanel className="w-full max-w-md transform overflow-hidden rounded-2xl bg-white dark:bg-gray-800 p-6 text-left align-middle shadow-xl transition-all">
                  <DialogTitle
                    as="h3"
                    className="text-lg font-medium leading-6 text-gray-900 dark:text-white"
                  >
                    Delete Chat Session
                  </DialogTitle>
                  <div className="mt-2">
                    <p className="text-sm text-gray-500 dark:text-gray-400">
                      Are you sure you want to delete this chat session? This action cannot be undone.
                    </p>
                  </div>

                  <div className="mt-4 flex space-x-3">
                    <button
                      type="button"
                      className="inline-flex justify-center rounded-md border border-transparent bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-red-500 focus-visible:ring-offset-2"
                      onClick={confirmDelete}
                    >
                      Delete
                    </button>
                    <button
                      type="button"
                      className="inline-flex justify-center rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
                      onClick={() => setIsDeleteDialogOpen(false)}
                    >
                      Cancel
                    </button>
                  </div>
                </DialogPanel>
              </Transition.Child>
            </div>
          </div>
        </Dialog>
      </Transition>

      {/* Rename Dialog */}
      <Transition appear show={isRenameDialogOpen} as={Fragment}>
        <Dialog as="div" className="relative z-50" onClose={() => setIsRenameDialogOpen(false)}>
          <Transition.Child
            as={Fragment}
            enter="ease-out duration-300"
            enterFrom="opacity-0"
            enterTo="opacity-100"
            leave="ease-in duration-200"
            leaveFrom="opacity-100"
            leaveTo="opacity-0"
          >
            <div className="fixed inset-0 bg-black bg-opacity-25" />
          </Transition.Child>

          <div className="fixed inset-0 overflow-y-auto">
            <div className="flex min-h-full items-center justify-center p-4 text-center">
              <Transition.Child
                as={Fragment}
                enter="ease-out duration-300"
                enterFrom="opacity-0 scale-95"
                enterTo="opacity-100 scale-100"
                leave="ease-in duration-200"
                leaveFrom="opacity-100 scale-100"
                leaveTo="opacity-0 scale-95"
              >
                <DialogPanel className="w-full max-w-md transform overflow-hidden rounded-2xl bg-white dark:bg-gray-800 p-6 text-left align-middle shadow-xl transition-all">
                  <DialogTitle
                    as="h3"
                    className="text-lg font-medium leading-6 text-gray-900 dark:text-white"
                  >
                    Rename Chat Session
                  </DialogTitle>
                  <div className="mt-2">
                    <input
                      type="text"
                      value={newTitle}
                      onChange={(e) => setNewTitle(e.target.value)}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-white"
                      placeholder="Enter new title"
                    />
                  </div>

                  <div className="mt-4 flex space-x-3">
                    <button
                      type="button"
                      className="inline-flex justify-center rounded-md border border-transparent bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
                      onClick={confirmRename}
                    >
                      Rename
                    </button>
                    <button
                      type="button"
                      className="inline-flex justify-center rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
                      onClick={() => setIsRenameDialogOpen(false)}
                    >
                      Cancel
                    </button>
                  </div>
                </DialogPanel>
              </Transition.Child>
            </div>
          </div>
        </Dialog>
      </Transition>
    </div>
  );
};

export default ChatSessionList;