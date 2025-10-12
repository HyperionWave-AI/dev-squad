import { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Button,
  TextField,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  IconButton,
  Card,
  CardContent,
  CardActions,
  Alert,
  CircularProgress,
  Chip,
  Stack,
  List,
  ListItem,
  ListItemText,
  Checkbox,
} from '@mui/material';
import {
  Add,
  Edit,
  Delete,
  SmartToy,
  Search,
  Close,
  Download,
  Upload,
} from '@mui/icons-material';
import { aiService, type Subagent, type CreateSubagentParams, type ClaudeAgent } from '../services/aiService';

const MAX_NAME_LENGTH = 50;
const MAX_DESCRIPTION_LENGTH = 200;
const MAX_PROMPT_LENGTH = 10000;

export function SubagentsPage() {
  const [subagents, setSubagents] = useState<Subagent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [dialogOpen, setDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [importDialogOpen, setImportDialogOpen] = useState(false);
  const [editingSubagent, setEditingSubagent] = useState<Subagent | null>(null);
  const [deletingSubagent, setDeletingSubagent] = useState<Subagent | null>(null);
  const [formData, setFormData] = useState<CreateSubagentParams>({
    name: '',
    description: '',
    systemPrompt: '',
  });
  const [formErrors, setFormErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);
  const [claudeAgents, setClaudeAgents] = useState<ClaudeAgent[]>([]);
  const [selectedAgents, setSelectedAgents] = useState<string[]>([]);
  const [loadingClaudeAgents, setLoadingClaudeAgents] = useState(false);
  const [importing, setImporting] = useState(false);
  const [importingAll, setImportingAll] = useState(false);

  // Load subagents on mount
  useEffect(() => {
    loadSubagents();
  }, []);

  // Load Claude agents when import dialog opens
  useEffect(() => {
    if (importDialogOpen) {
      loadClaudeAgents();
    }
  }, [importDialogOpen]);

  const loadSubagents = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await aiService.listSubagents();
      setSubagents(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load subagents');
    } finally {
      setLoading(false);
    }
  };

  const loadClaudeAgents = async () => {
    setLoadingClaudeAgents(true);
    setError(null);
    try {
      const data = await aiService.listClaudeAgents();
      setClaudeAgents(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load Claude agents');
    } finally {
      setLoadingClaudeAgents(false);
    }
  };

  const validateForm = (): boolean => {
    const errors: Record<string, string> = {};

    if (!formData.name.trim()) {
      errors.name = 'Name is required';
    } else if (formData.name.length < 3) {
      errors.name = 'Name must be at least 3 characters';
    } else if (formData.name.length > MAX_NAME_LENGTH) {
      errors.name = `Name must not exceed ${MAX_NAME_LENGTH} characters`;
    }

    if (formData.description && formData.description.length > MAX_DESCRIPTION_LENGTH) {
      errors.description = `Description must not exceed ${MAX_DESCRIPTION_LENGTH} characters`;
    }

    if (!formData.systemPrompt.trim()) {
      errors.systemPrompt = 'System prompt is required';
    } else if (formData.systemPrompt.length > MAX_PROMPT_LENGTH) {
      errors.systemPrompt = `System prompt must not exceed ${MAX_PROMPT_LENGTH} characters`;
    }

    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleOpenDialog = (subagent?: Subagent) => {
    if (subagent) {
      setEditingSubagent(subagent);
      setFormData({
        name: subagent.name,
        description: subagent.description || '',
        systemPrompt: subagent.systemPrompt,
      });
    } else {
      setEditingSubagent(null);
      setFormData({
        name: '',
        description: '',
        systemPrompt: '',
      });
    }
    setFormErrors({});
    setDialogOpen(true);
  };

  const handleCloseDialog = () => {
    setDialogOpen(false);
    setEditingSubagent(null);
    setFormData({
      name: '',
      description: '',
      systemPrompt: '',
    });
    setFormErrors({});
  };

  const handleSubmit = async () => {
    if (!validateForm()) {
      return;
    }

    setSubmitting(true);
    setError(null);

    try {
      if (editingSubagent) {
        // Update existing subagent
        await aiService.updateSubagent(editingSubagent.id, formData);
      } else {
        // Create new subagent
        await aiService.createSubagent(formData);
      }

      await loadSubagents();
      handleCloseDialog();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save subagent');
    } finally {
      setSubmitting(false);
    }
  };

  const handleOpenDeleteDialog = (subagent: Subagent) => {
    setDeletingSubagent(subagent);
    setDeleteDialogOpen(true);
  };

  const handleCloseDeleteDialog = () => {
    setDeleteDialogOpen(false);
    setDeletingSubagent(null);
  };

  const handleDelete = async () => {
    if (!deletingSubagent) return;

    setSubmitting(true);
    setError(null);

    try {
      await aiService.deleteSubagent(deletingSubagent.id);
      await loadSubagents();
      handleCloseDeleteDialog();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete subagent');
    } finally {
      setSubmitting(false);
    }
  };

  const handleImportClaudeAgents = async () => {
    setImporting(true);
    setError(null);

    try {
      const result = await aiService.importClaudeAgents(selectedAgents);

      if (result.success) {
        setError(null);
        // Show success message by creating a temporary success alert
        const successMsg = `Successfully imported ${result.imported} agent${result.imported !== 1 ? 's' : ''}`;
        setError(successMsg);

        // Refresh subagents list
        await loadSubagents();

        // Close dialog and reset
        setImportDialogOpen(false);
        setSelectedAgents([]);
        setClaudeAgents([]);
      }

      if (result.errors.length > 0) {
        const errorMsg = `Import completed with warnings: ${result.errors.join(', ')}`;
        setError(errorMsg);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to import Claude agents');
    } finally {
      setImporting(false);
    }
  };

  const handleImportAllClaudeAgents = async () => {
    setImportingAll(true);
    setError(null);

    try {
      const result = await aiService.importAllClaudeAgents();

      if (result.success) {
        setError(null);
        // Show success message
        const successMsg = `Successfully imported ${result.imported} agent${result.imported !== 1 ? 's' : ''}`;
        setError(successMsg);

        // Refresh subagents list
        await loadSubagents();
      }

      if (result.errors.length > 0) {
        const errorMsg = `Import completed with warnings: ${result.errors.join(', ')}`;
        setError(errorMsg);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to import all Claude agents');
    } finally {
      setImportingAll(false);
    }
  };

  const handleToggleAgent = (agentName: string) => {
    setSelectedAgents((prev) =>
      prev.includes(agentName)
        ? prev.filter((name) => name !== agentName)
        : [...prev, agentName]
    );
  };

  const filteredSubagents = subagents.filter((subagent) => {
    const query = searchQuery.toLowerCase();
    return (
      subagent.name.toLowerCase().includes(query) ||
      (subagent.description || '').toLowerCase().includes(query)
    );
  });

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      {/* Header */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" mb={3}>
        <Box>
          <Typography variant="h4" fontWeight={600}>
            Subagents
          </Typography>
          <Typography variant="body1" color="text.secondary">
            Manage AI subagents with custom system prompts
          </Typography>
        </Box>
        <Stack direction="row" spacing={2}>
          <Button
            variant="outlined"
            startIcon={<Download />}
            onClick={() => setImportDialogOpen(true)}
            size="large"
          >
            Import from Claude
          </Button>
          <Button
            variant="outlined"
            startIcon={importingAll ? <CircularProgress size={20} /> : <Upload />}
            onClick={handleImportAllClaudeAgents}
            disabled={importingAll || loading}
            size="large"
          >
            {importingAll ? 'Importing...' : 'Import All'}
          </Button>
          <Button
            variant="contained"
            startIcon={<Add />}
            onClick={() => handleOpenDialog()}
            size="large"
          >
            Create Subagent
          </Button>
        </Stack>
      </Stack>

      {error && (
        <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {/* Search Bar */}
      <Paper sx={{ p: 2, mb: 3 }}>
        <TextField
          fullWidth
          placeholder="Search subagents..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          InputProps={{
            startAdornment: <Search sx={{ mr: 1, color: 'text.secondary' }} />,
          }}
        />
      </Paper>

      {/* Subagents Grid */}
      {filteredSubagents.length === 0 ? (
        <Paper sx={{ p: 4, textAlign: 'center' }}>
          <SmartToy sx={{ fontSize: 64, color: 'text.secondary', mb: 2 }} />
          <Typography variant="h6" color="text.secondary">
            {searchQuery ? 'No subagents found' : 'No subagents yet'}
          </Typography>
          <Typography variant="body2" color="text.secondary" paragraph>
            {searchQuery ? 'Try a different search query' : 'Create your first subagent to get started'}
          </Typography>
          {!searchQuery && (
            <Button
              variant="contained"
              startIcon={<Add />}
              onClick={() => handleOpenDialog()}
            >
              Create Subagent
            </Button>
          )}
        </Paper>
      ) : (
        <Box
          sx={{
            display: 'grid',
            gridTemplateColumns: {
              xs: '1fr',
              md: 'repeat(2, 1fr)',
              lg: 'repeat(3, 1fr)',
            },
            gap: 3,
          }}
        >
          {filteredSubagents.map((subagent) => (
            <Card key={subagent.id} sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
              <CardContent sx={{ flexGrow: 1 }}>
                <Stack direction="row" alignItems="center" spacing={1} mb={1}>
                  <SmartToy color="primary" />
                  <Typography variant="h6" fontWeight={600}>
                    {subagent.name}
                  </Typography>
                </Stack>
                {subagent.description && (
                  <Typography variant="body2" color="text.secondary" paragraph>
                    {subagent.description}
                  </Typography>
                )}
                <Chip
                  label={`${subagent.systemPrompt.length} chars`}
                  size="small"
                  variant="outlined"
                />
              </CardContent>
              <CardActions>
                <Button
                  size="small"
                  startIcon={<Edit />}
                  onClick={() => handleOpenDialog(subagent)}
                >
                  Edit
                </Button>
                <Button
                  size="small"
                  color="error"
                  startIcon={<Delete />}
                  onClick={() => handleOpenDeleteDialog(subagent)}
                >
                  Delete
                </Button>
              </CardActions>
            </Card>
          ))}
        </Box>
      )}

      {/* Create/Edit Dialog */}
      <Dialog
        open={dialogOpen}
        onClose={handleCloseDialog}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>
          <Stack direction="row" justifyContent="space-between" alignItems="center">
            <Typography variant="h6">
              {editingSubagent ? 'Edit Subagent' : 'Create Subagent'}
            </Typography>
            <IconButton onClick={handleCloseDialog} size="small">
              <Close />
            </IconButton>
          </Stack>
        </DialogTitle>
        <DialogContent>
          <Stack spacing={3} sx={{ mt: 1 }}>
            <TextField
              label="Name"
              required
              fullWidth
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              error={!!formErrors.name}
              helperText={formErrors.name || `${formData.name.length}/${MAX_NAME_LENGTH} characters`}
            />

            <TextField
              label="Description"
              fullWidth
              multiline
              rows={2}
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              error={!!formErrors.description}
              helperText={formErrors.description || `${formData.description?.length || 0}/${MAX_DESCRIPTION_LENGTH} characters`}
            />

            <TextField
              label="System Prompt"
              required
              fullWidth
              multiline
              rows={10}
              value={formData.systemPrompt}
              onChange={(e) => setFormData({ ...formData, systemPrompt: e.target.value })}
              error={!!formErrors.systemPrompt}
              helperText={formErrors.systemPrompt || `${formData.systemPrompt.length}/${MAX_PROMPT_LENGTH} characters`}
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDialog} disabled={submitting}>
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={handleSubmit}
            disabled={submitting}
            startIcon={submitting ? <CircularProgress size={20} /> : undefined}
          >
            {submitting ? 'Saving...' : editingSubagent ? 'Update' : 'Create'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog
        open={deleteDialogOpen}
        onClose={handleCloseDeleteDialog}
      >
        <DialogTitle>Delete Subagent</DialogTitle>
        <DialogContent>
          <Typography>
            Are you sure you want to delete <strong>{deletingSubagent?.name}</strong>?
            This action cannot be undone.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDeleteDialog} disabled={submitting}>
            Cancel
          </Button>
          <Button
            variant="contained"
            color="error"
            onClick={handleDelete}
            disabled={submitting}
            startIcon={submitting ? <CircularProgress size={20} /> : <Delete />}
          >
            {submitting ? 'Deleting...' : 'Delete'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Import from Claude Dialog */}
      <Dialog
        open={importDialogOpen}
        onClose={() => setImportDialogOpen(false)}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle>
          <Stack direction="row" justifyContent="space-between" alignItems="center">
            <Typography variant="h6">Import from Claude</Typography>
            <IconButton onClick={() => setImportDialogOpen(false)} size="small">
              <Close />
            </IconButton>
          </Stack>
        </DialogTitle>
        <DialogContent>
          {loadingClaudeAgents ? (
            <Box display="flex" justifyContent="center" alignItems="center" minHeight="200px">
              <CircularProgress />
            </Box>
          ) : claudeAgents.length === 0 ? (
            <Alert severity="info">
              No Claude agents found in .claude/agents directory
            </Alert>
          ) : (
            <List>
              {claudeAgents.map((agent) => (
                <ListItem
                  key={agent.name}
                  dense
                  button
                  onClick={() => handleToggleAgent(agent.name)}
                >
                  <Checkbox
                    edge="start"
                    checked={selectedAgents.includes(agent.name)}
                    tabIndex={-1}
                    disableRipple
                  />
                  <ListItemText
                    primary={agent.name}
                    secondary={agent.description}
                  />
                </ListItem>
              ))}
            </List>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setImportDialogOpen(false)} disabled={importing}>
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={handleImportClaudeAgents}
            disabled={selectedAgents.length === 0 || importing}
            startIcon={importing ? <CircularProgress size={20} /> : <Download />}
          >
            {importing ? 'Importing...' : `Import (${selectedAgents.length})`}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
