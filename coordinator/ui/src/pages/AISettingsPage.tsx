/**
 * AI Settings Page Component
 *
 * Provides interface for managing system prompts and subagents.
 * Features: system prompt editor, subagent CRUD operations, optimistic UI updates.
 */

import { useState, useEffect } from 'react';
import {
  Box,
  Container,
  Paper,
  TextField,
  Button,
  Typography,
  List,
  Card,
  CardContent,
  CardActions,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Snackbar,
  Alert,
  Skeleton,
  Divider,
} from '@mui/material';
import { Save, Cancel, Add, Edit, Delete, Settings } from '@mui/icons-material';
import * as aiSettingsService from '../services/aiSettingsService';
import type { Subagent, SubagentCreate } from '../services/aiSettingsService';

export function AISettingsPage() {
  // System Prompt State
  const [systemPrompt, setSystemPrompt] = useState('');
  const [originalPrompt, setOriginalPrompt] = useState('');
  const [promptLoading, setPromptLoading] = useState(true);
  const [promptSaving, setPromptSaving] = useState(false);

  // Subagents State
  const [subagents, setSubagents] = useState<Subagent[]>([]);
  const [subagentsLoading, setSubagentsLoading] = useState(true);

  // Dialog State
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingSubagent, setEditingSubagent] = useState<Subagent | null>(null);
  const [dialogForm, setDialogForm] = useState<SubagentCreate>({
    name: '',
    description: '',
    systemPrompt: '',
  });

  // Delete Confirmation State
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [subagentToDelete, setSubagentToDelete] = useState<Subagent | null>(null);

  // Snackbar State
  const [snackbar, setSnackbar] = useState<{
    open: boolean;
    message: string;
    severity: 'success' | 'error';
  }>({
    open: false,
    message: '',
    severity: 'success',
  });

  // Load system prompt on mount
  useEffect(() => {
    loadSystemPrompt();
  }, []);

  // Load subagents on mount
  useEffect(() => {
    loadSubagents();
  }, []);

  const loadSystemPrompt = async () => {
    try {
      setPromptLoading(true);
      const prompt = await aiSettingsService.getSystemPrompt();
      setSystemPrompt(prompt);
      setOriginalPrompt(prompt);
    } catch (error) {
      showSnackbar(
        `Failed to load system prompt: ${error instanceof Error ? error.message : 'Unknown error'}`,
        'error'
      );
    } finally {
      setPromptLoading(false);
    }
  };

  const loadSubagents = async () => {
    try {
      setSubagentsLoading(true);
      const data = await aiSettingsService.listSubagents();
      setSubagents(data);
    } catch (error) {
      showSnackbar(
        `Failed to load subagents: ${error instanceof Error ? error.message : 'Unknown error'}`,
        'error'
      );
    } finally {
      setSubagentsLoading(false);
    }
  };

  const handleSavePrompt = async () => {
    try {
      setPromptSaving(true);
      await aiSettingsService.updateSystemPrompt(systemPrompt);
      setOriginalPrompt(systemPrompt);
      showSnackbar('System prompt saved successfully', 'success');
    } catch (error) {
      showSnackbar(
        `Failed to save system prompt: ${error instanceof Error ? error.message : 'Unknown error'}`,
        'error'
      );
    } finally {
      setPromptSaving(false);
    }
  };

  const handleCancelPrompt = () => {
    setSystemPrompt(originalPrompt);
  };

  const handleNewSubagent = () => {
    setEditingSubagent(null);
    setDialogForm({
      name: '',
      description: '',
      systemPrompt: '',
    });
    setDialogOpen(true);
  };

  const handleEditSubagent = (subagent: Subagent) => {
    setEditingSubagent(subagent);
    setDialogForm({
      name: subagent.name,
      description: subagent.description,
      systemPrompt: subagent.systemPrompt,
    });
    setDialogOpen(true);
  };

  const handleDeleteClick = (subagent: Subagent) => {
    setSubagentToDelete(subagent);
    setDeleteDialogOpen(true);
  };

  const handleDialogClose = () => {
    setDialogOpen(false);
    setEditingSubagent(null);
    setDialogForm({
      name: '',
      description: '',
      systemPrompt: '',
    });
  };

  const handleDialogSave = async () => {
    // Validate required fields
    if (!dialogForm.name.trim()) {
      showSnackbar('Name is required', 'error');
      return;
    }
    if (!dialogForm.description.trim()) {
      showSnackbar('Description is required', 'error');
      return;
    }
    if (!dialogForm.systemPrompt.trim()) {
      showSnackbar('System prompt is required', 'error');
      return;
    }

    try {
      if (editingSubagent) {
        // Update existing subagent
        const updated = await aiSettingsService.updateSubagent(editingSubagent.id, dialogForm);
        // Optimistic update
        setSubagents((prev) =>
          prev.map((s) => (s.id === updated.id ? updated : s))
        );
        showSnackbar('Subagent updated successfully', 'success');
      } else {
        // Create new subagent
        const created = await aiSettingsService.createSubagent(dialogForm);
        // Optimistic update
        setSubagents((prev) => [...prev, created]);
        showSnackbar('Subagent created successfully', 'success');
      }
      handleDialogClose();
    } catch (error) {
      showSnackbar(
        `Failed to save subagent: ${error instanceof Error ? error.message : 'Unknown error'}`,
        'error'
      );
    }
  };

  const handleDeleteConfirm = async () => {
    if (!subagentToDelete) return;

    try {
      await aiSettingsService.deleteSubagent(subagentToDelete.id);
      // Optimistic update
      setSubagents((prev) => prev.filter((s) => s.id !== subagentToDelete.id));
      showSnackbar('Subagent deleted successfully', 'success');
      setDeleteDialogOpen(false);
      setSubagentToDelete(null);
    } catch (error) {
      showSnackbar(
        `Failed to delete subagent: ${error instanceof Error ? error.message : 'Unknown error'}`,
        'error'
      );
    }
  };

  const showSnackbar = (message: string, severity: 'success' | 'error') => {
    setSnackbar({ open: true, message, severity });
  };

  const handleSnackbarClose = () => {
    setSnackbar((prev) => ({ ...prev, open: false }));
  };

  const isPromptDirty = systemPrompt !== originalPrompt;

  return (
    <Container maxWidth="lg" sx={{ py: 4 }}>
      {/* Header */}
      <Box sx={{ mb: 4, display: 'flex', alignItems: 'center', gap: 2 }}>
        <Settings sx={{ fontSize: 32, color: 'primary.main' }} />
        <Typography variant="h4" component="h1" fontWeight={600}>
          AI Settings
        </Typography>
      </Box>

      {/* System Prompt Section */}
      <Paper sx={{ p: 3, mb: 4 }}>
        <Typography variant="h6" gutterBottom fontWeight={600}>
          System Prompt
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          Configure the default system prompt for AI interactions
        </Typography>

        {promptLoading ? (
          <Skeleton variant="rectangular" height={150} sx={{ borderRadius: 1 }} />
        ) : (
          <>
            <TextField
              fullWidth
              multiline
              minRows={6}
              maxRows={12}
              value={systemPrompt}
              onChange={(e) => setSystemPrompt(e.target.value)}
              placeholder="Enter system prompt..."
              sx={{
                mb: 2,
                '& .MuiOutlinedInput-root': {
                  fontFamily: 'monospace',
                  fontSize: '0.9rem',
                },
              }}
            />

            <Box sx={{ display: 'flex', gap: 1, justifyContent: 'flex-end' }}>
              <Button
                variant="outlined"
                startIcon={<Cancel />}
                onClick={handleCancelPrompt}
                disabled={!isPromptDirty || promptSaving}
              >
                Cancel
              </Button>
              <Button
                variant="contained"
                startIcon={<Save />}
                onClick={handleSavePrompt}
                disabled={!isPromptDirty || promptSaving}
              >
                {promptSaving ? 'Saving...' : 'Save'}
              </Button>
            </Box>
          </>
        )}
      </Paper>

      {/* Subagents Section */}
      <Paper sx={{ p: 3 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
          <div>
            <Typography variant="h6" fontWeight={600}>
              Subagents
            </Typography>
            <Typography variant="body2" color="text.secondary">
              Manage AI subagents with custom system prompts
            </Typography>
          </div>
          <Button
            variant="contained"
            startIcon={<Add />}
            onClick={handleNewSubagent}
            disabled={subagentsLoading}
          >
            New Subagent
          </Button>
        </Box>

        <Divider sx={{ mb: 2 }} />

        {subagentsLoading ? (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <Skeleton variant="rectangular" height={120} sx={{ borderRadius: 1 }} />
            <Skeleton variant="rectangular" height={120} sx={{ borderRadius: 1 }} />
          </Box>
        ) : subagents.length === 0 ? (
          <Box
            sx={{
              textAlign: 'center',
              py: 6,
              color: 'text.secondary',
            }}
          >
            <Typography variant="body1" gutterBottom>
              No subagents configured yet
            </Typography>
            <Typography variant="body2">
              Click "New Subagent" to create one
            </Typography>
          </Box>
        ) : (
          <List sx={{ p: 0 }}>
            {subagents.map((subagent) => (
              <Card key={subagent.id} sx={{ mb: 2 }}>
                <CardContent>
                  <Typography variant="h6" gutterBottom>
                    {subagent.name}
                  </Typography>
                  <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                    {subagent.description}
                  </Typography>
                  <Typography
                    variant="caption"
                    sx={{
                      display: 'block',
                      color: 'text.disabled',
                      fontFamily: 'monospace',
                      whiteSpace: 'nowrap',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                    }}
                  >
                    Prompt: {subagent.systemPrompt}
                  </Typography>
                </CardContent>
                <CardActions sx={{ justifyContent: 'flex-end', px: 2, pb: 2 }}>
                  <IconButton
                    size="small"
                    onClick={() => handleEditSubagent(subagent)}
                    sx={{ color: 'primary.main' }}
                  >
                    <Edit fontSize="small" />
                  </IconButton>
                  <IconButton
                    size="small"
                    onClick={() => handleDeleteClick(subagent)}
                    sx={{ color: 'error.main' }}
                  >
                    <Delete fontSize="small" />
                  </IconButton>
                </CardActions>
              </Card>
            ))}
          </List>
        )}
      </Paper>

      {/* Subagent Create/Edit Dialog */}
      <Dialog open={dialogOpen} onClose={handleDialogClose} maxWidth="sm" fullWidth>
        <DialogTitle>
          {editingSubagent ? 'Edit Subagent' : 'New Subagent'}
        </DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
            <TextField
              label="Name"
              fullWidth
              required
              value={dialogForm.name}
              onChange={(e) => setDialogForm((prev) => ({ ...prev, name: e.target.value }))}
              placeholder="e.g., Code Reviewer"
            />
            <TextField
              label="Description"
              fullWidth
              required
              value={dialogForm.description}
              onChange={(e) => setDialogForm((prev) => ({ ...prev, description: e.target.value }))}
              placeholder="e.g., Reviews code for quality and best practices"
            />
            <TextField
              label="System Prompt"
              fullWidth
              required
              multiline
              minRows={4}
              maxRows={8}
              value={dialogForm.systemPrompt}
              onChange={(e) => setDialogForm((prev) => ({ ...prev, systemPrompt: e.target.value }))}
              placeholder="Enter the system prompt for this subagent..."
              sx={{
                '& .MuiOutlinedInput-root': {
                  fontFamily: 'monospace',
                  fontSize: '0.9rem',
                },
              }}
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleDialogClose} color="inherit">
            Cancel
          </Button>
          <Button onClick={handleDialogSave} variant="contained">
            {editingSubagent ? 'Update' : 'Create'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteDialogOpen} onClose={() => setDeleteDialogOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle>Delete Subagent?</DialogTitle>
        <DialogContent>
          <Typography>
            Are you sure you want to delete "{subagentToDelete?.name}"?
            This action cannot be undone.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteDialogOpen(false)} color="inherit">
            Cancel
          </Button>
          <Button onClick={handleDeleteConfirm} color="error" variant="contained">
            Delete
          </Button>
        </DialogActions>
      </Dialog>

      {/* Snackbar Notifications */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={4000}
        onClose={handleSnackbarClose}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
      >
        <Alert onClose={handleSnackbarClose} severity={snackbar.severity} sx={{ width: '100%' }}>
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Container>
  );
}
