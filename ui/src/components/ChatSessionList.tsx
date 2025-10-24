/**
 * ChatSessionList Component
 *
 * Displays list of chat sessions in left sidebar.
 * Features: session selection, new chat creation, delete with confirmation, delete all functionality.
 */

import { useState } from 'react';
import {
  Box,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  IconButton,
  Button,
  Typography,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  DialogContentText,
  TextField,
} from '@mui/material';
import { Add, Delete, Chat, DeleteSweep } from '@mui/icons-material';
import type { ChatSession } from '../services/chatService';

interface ChatSessionListProps {
  sessions: ChatSession[];
  activeSessionId: string | null;
  onSessionSelect: (sessionId: string) => void;
  onNewChat: () => void;
  onDeleteSession: (sessionId: string) => void;
  onDeleteAllSessions: () => void;
  onRenameSession: (sessionId: string, newTitle: string) => void;
  loading?: boolean;
}

export function ChatSessionList({
  sessions,
  activeSessionId,
  onSessionSelect,
  onNewChat,
  onDeleteSession,
  onDeleteAllSessions,
  onRenameSession,
  loading = false,
}: ChatSessionListProps) {
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deleteAllDialogOpen, setDeleteAllDialogOpen] = useState(false);
  const [sessionToDelete, setSessionToDelete] = useState<string | null>(null);
  const [editingSessionId, setEditingSessionId] = useState<string | null>(null);
  const [editingTitle, setEditingTitle] = useState<string>('');

  const handleDeleteClick = (sessionId: string, event: React.MouseEvent) => {
    event.stopPropagation();
    setSessionToDelete(sessionId);
    setDeleteDialogOpen(true);
  };

  const handleDeleteConfirm = () => {
    if (sessionToDelete) {
      onDeleteSession(sessionToDelete);
      setDeleteDialogOpen(false);
      setSessionToDelete(null);
    }
  };

  const handleDeleteCancel = () => {
    setDeleteDialogOpen(false);
    setSessionToDelete(null);
  };

  const handleDeleteAllClick = () => {
    setDeleteAllDialogOpen(true);
  };

  const handleDeleteAllConfirm = () => {
    onDeleteAllSessions();
    setDeleteAllDialogOpen(false);
  };

  const handleDeleteAllCancel = () => {
    setDeleteAllDialogOpen(false);
  };

  const handleDoubleClick = (session: ChatSession, event: React.MouseEvent) => {
    event.stopPropagation();
    setEditingSessionId(session.id);
    setEditingTitle(session.title);
  };

  const handleSaveRename = () => {
    if (editingSessionId && editingTitle.trim() && editingTitle.length <= 100) {
      onRenameSession(editingSessionId, editingTitle.trim());
      setEditingSessionId(null);
      setEditingTitle('');
    }
  };

  const handleCancelEdit = () => {
    setEditingSessionId(null);
    setEditingTitle('');
  };

  const handleKeyDown = (event: React.KeyboardEvent) => {
    if (event.key === 'Enter') {
      event.preventDefault();
      handleSaveRename();
    } else if (event.key === 'Escape') {
      event.preventDefault();
      handleCancelEdit();
    }
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;
    return date.toLocaleDateString();
  };

  return (
    <Box
      sx={{
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        borderRight: '1px solid',
        borderColor: 'divider',
        backgroundColor: 'background.paper',
      }}
    >
      {/* Header with New Chat and Delete All Buttons */}
      <Box sx={{ p: 2, borderBottom: '1px solid', borderColor: 'divider' }}>
        <Box
          sx={{
            display: 'flex',
            gap: 1,
            alignItems: 'center',
            flexWrap: 'wrap',
          }}
        >
          <Button
            variant="contained"
            startIcon={<Add />}
            onClick={onNewChat}
            disabled={loading}
            sx={{
              textTransform: 'none',
              fontWeight: 600,
              flex: '2 1 auto',
              minWidth: 0,
              '@media (max-width: 600px)': {
                flex: '1 1 100%',
                mb: 1,
              },
            }}
          >
            New Chat
          </Button>
          {sessions.length > 0 && (
            <IconButton
              onClick={handleDeleteAllClick}
              disabled={loading}
              color="error"
              title="Delete All Chats"
              aria-label="Delete all chat sessions"
              sx={{
                border: '1px solid',
                borderColor: 'error.main',
                borderRadius: '8px', // Changed from default circular to curved square
                '&:hover': {
                  backgroundColor: 'error.light',
                  borderColor: 'error.main',
                },
              }}
            >
              <DeleteSweep />
            </IconButton>
          )}
        </Box>
      </Box>
      <Box
        sx={{
          flexGrow: 1,
          overflowY: 'auto',
          overflowX: 'hidden',
        }}
      >
        {sessions.length === 0 ? (
          <Box
            sx={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              height: '100%',
              p: 3,
              textAlign: 'center',
            }}
          >
            <Chat sx={{ fontSize: 48, color: 'text.disabled', mb: 2 }} />
            <Typography variant="body2" color="text.secondary">
              No chats yet
            </Typography>
            <Typography variant="caption" color="text.disabled">
              Click "New Chat" to start
            </Typography>
          </Box>
        ) : (
          <List sx={{ p: 0 }}>
            {sessions.map((session) => (
              <ListItem
                key={session.id}
                disablePadding
                secondaryAction={
                  <IconButton
                    edge="end"
                    size="small"
                    onClick={(e) => handleDeleteClick(session.id, e)}
                    sx={{
                      opacity: 0.6,
                      '&:hover': { opacity: 1, color: 'error.main' },
                    }}
                  >
                    <Delete fontSize="small" />
                  </IconButton>
                }
              >
                <ListItemButton
                  selected={session.id === activeSessionId}
                  onClick={() => onSessionSelect(session.id)}
                  onDoubleClick={(e) => handleDoubleClick(session, e)}
                  sx={{
                    py: 1.5,
                    px: 2,
                    borderBottom: '1px solid',
                    borderColor: 'divider',
                    '&.Mui-selected': {
                      backgroundColor: 'primary.light',
                      borderLeftWidth: 3,
                      borderLeftStyle: 'solid',
                      borderLeftColor: 'primary.main',
                    },
                  }}
                >
                  <ListItemText
                    primary={
                      editingSessionId === session.id ? (
                        <TextField
                          value={editingTitle}
                          onChange={(e) => setEditingTitle(e.target.value)}
                          onKeyDown={handleKeyDown}
                          onBlur={handleSaveRename}
                          autoFocus
                          fullWidth
                          size="small"
                          variant="standard"
                          inputProps={{ maxLength: 100 }}
                          sx={{
                            '& .MuiInputBase-input': {
                              fontSize: '0.875rem',
                              fontWeight: session.id === activeSessionId ? 600 : 400,
                            },
                          }}
                          onClick={(e) => e.stopPropagation()}
                        />
                      ) : (
                        <Typography
                          variant="body2"
                          sx={{
                            fontWeight: session.id === activeSessionId ? 600 : 400,
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            whiteSpace: 'nowrap',
                          }}
                        >
                          {session.title}
                        </Typography>
                      )
                    }
                    secondary={
                      <Typography
                        variant="caption"
                        color="text.secondary"
                        sx={{
                          fontSize: '0.75rem',
                          overflow: 'hidden',
                          textOverflow: 'ellipsis',
                          whiteSpace: 'nowrap',
                        }}
                      >
                        {formatDate(session.updatedAt)}
                      </Typography>
                    }
                  />
                </ListItemButton>
              </ListItem>
            ))}
          </List>
        )}
      </Box>

      {/* Delete Session Confirmation Dialog */}
      <Dialog
        open={deleteDialogOpen}
        onClose={handleDeleteCancel}
        aria-labelledby="delete-dialog-title"
        aria-describedby="delete-dialog-description"
      >
        <DialogTitle id="delete-dialog-title">Delete Chat Session</DialogTitle>
        <DialogContent>
          <DialogContentText id="delete-dialog-description">
            Are you sure you want to delete this chat session? This action cannot be undone.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleDeleteCancel} color="primary">
            Cancel
          </Button>
          <Button onClick={handleDeleteConfirm} color="error" variant="contained">
            Delete
          </Button>
        </DialogActions>
      </Dialog>

      {/* Delete All Sessions Confirmation Dialog */}
      <Dialog
        open={deleteAllDialogOpen}
        onClose={handleDeleteAllCancel}
        aria-labelledby="delete-all-dialog-title"
        aria-describedby="delete-all-dialog-description"
      >
        <DialogTitle id="delete-all-dialog-title">Delete All Chat Sessions</DialogTitle>
        <DialogContent>
          <DialogContentText id="delete-all-dialog-description">
            Are you sure you want to delete all chat sessions? This action cannot be undone and will remove all your chat history.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleDeleteAllCancel} color="primary">
            Cancel
          </Button>
          <Button onClick={handleDeleteAllConfirm} color="error" variant="contained">
            Delete All
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}