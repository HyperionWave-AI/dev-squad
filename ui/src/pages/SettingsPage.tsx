import { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  TextField,
  Button,
  Alert,
  CircularProgress,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Checkbox,
  FormControlLabel,
  Tooltip,
} from '@mui/material';
import {
  Save,
  Add,
  Delete,
  CheckCircle,
  Visibility,
  ExpandMore,
  Info,
} from '@mui/icons-material';
import { aiService } from '../services/aiService';
import type { SystemPromptVersion } from '../services/aiService';

const MAX_CHARACTERS = 10000;

export function SettingsPage() {
  // Version state
  const [versions, setVersions] = useState<SystemPromptVersion[]>([]);
  const [versionsLoading, setVersionsLoading] = useState(true);
  const [defaultPrompt, setDefaultPrompt] = useState('');
  const [defaultPromptLoading, setDefaultPromptLoading] = useState(true);

  // Create version dialog state
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [newPrompt, setNewPrompt] = useState('');
  const [newDescription, setNewDescription] = useState('');
  const [activateImmediately, setActivateImmediately] = useState(true);
  const [creating, setCreating] = useState(false);

  // View version dialog state
  const [viewDialogOpen, setViewDialogOpen] = useState(false);
  const [viewingVersion, setViewingVersion] = useState<SystemPromptVersion | null>(null);

  // General state
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Load versions and default prompt on mount
  useEffect(() => {
    loadVersions();
    loadDefaultPrompt();
  }, []);

  const loadVersions = async () => {
    setVersionsLoading(true);
    setError(null);
    try {
      const versionsList = await aiService.listSystemPromptVersions();
      setVersions(versionsList);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load versions');
    } finally {
      setVersionsLoading(false);
    }
  };

  const loadDefaultPrompt = async () => {
    setDefaultPromptLoading(true);
    try {
      const prompt = await aiService.getDefaultSystemPrompt();
      setDefaultPrompt(prompt);
    } catch (err) {
      console.error('Failed to load default prompt:', err);
    } finally {
      setDefaultPromptLoading(false);
    }
  };

  const handleCreateVersion = async () => {
    if (newPrompt.length > MAX_CHARACTERS) {
      setError(`Prompt exceeds maximum length of ${MAX_CHARACTERS} characters`);
      return;
    }

    if (!newPrompt.trim()) {
      setError('Prompt cannot be empty');
      return;
    }

    setCreating(true);
    setError(null);
    setSuccessMessage(null);

    try {
      await aiService.createSystemPromptVersion({
        prompt: newPrompt,
        description: newDescription || undefined,
        activate: activateImmediately,
      });

      setSuccessMessage(
        `Version created successfully${activateImmediately ? ' and activated' : ''}!`
      );
      setCreateDialogOpen(false);
      setNewPrompt('');
      setNewDescription('');
      setActivateImmediately(true);
      await loadVersions();

      // Clear success message after 3 seconds
      setTimeout(() => setSuccessMessage(null), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create version');
    } finally {
      setCreating(false);
    }
  };

  const handleActivateVersion = async (id: string) => {
    setError(null);
    setSuccessMessage(null);

    try {
      await aiService.activateSystemPromptVersion(id);
      setSuccessMessage('Version activated successfully!');
      await loadVersions();

      setTimeout(() => setSuccessMessage(null), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to activate version');
    }
  };

  const handleDeleteVersion = async (id: string, version: number) => {
    if (!confirm(`Are you sure you want to delete version ${version}?`)) {
      return;
    }

    setError(null);
    setSuccessMessage(null);

    try {
      await aiService.deleteSystemPromptVersion(id);
      setSuccessMessage('Version deleted successfully!');
      await loadVersions();

      setTimeout(() => setSuccessMessage(null), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete version');
    }
  };

  const handleViewVersion = (version: SystemPromptVersion) => {
    setViewingVersion(version);
    setViewDialogOpen(true);
  };

  const characterCount = newPrompt.length;
  const isOverLimit = characterCount > MAX_CHARACTERS;

  return (
    <Box>
      <Typography variant="h4" gutterBottom fontWeight={600}>
        Settings
      </Typography>
      <Typography variant="body1" color="text.secondary" paragraph>
        Manage system prompts with full version control
      </Typography>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {successMessage && (
        <Alert severity="success" sx={{ mb: 2 }} onClose={() => setSuccessMessage(null)}>
          {successMessage}
        </Alert>
      )}

      {/* Default System Prompt (Read-only) */}
      <Paper sx={{ p: 3, mb: 3 }}>
        <Accordion>
          <AccordionSummary expandIcon={<ExpandMore />}>
            <Stack direction="row" spacing={1} alignItems="center">
              <Info color="info" />
              <Typography variant="h6">Default System Prompt</Typography>
              <Chip label="Read-Only" size="small" color="info" />
            </Stack>
          </AccordionSummary>
          <AccordionDetails>
            {defaultPromptLoading ? (
              <CircularProgress size={24} />
            ) : (
              <>
                <Alert severity="info" sx={{ mb: 2 }}>
                  <Typography variant="body2" fontWeight={600} gutterBottom>
                    Why is this displayed?
                  </Typography>
                  <Typography variant="body2" paragraph>
                    This prompt serves as a <strong>reference and fallback</strong> for the AI system. It's shown here so you can view the baseline behavior and use it as a starting point when creating your own custom versions.
                  </Typography>

                  <Typography variant="body2" fontWeight={600} gutterBottom>
                    Why is it read-only?
                  </Typography>
                  <Typography variant="body2" paragraph>
                    The default prompt is <strong>hardcoded in the application source code</strong> to ensure system stability and provide a consistent baseline. Modifying it would require rebuilding the entire application.
                  </Typography>

                  <Typography variant="body2" fontWeight={600} gutterBottom>
                    Purpose:
                  </Typography>
                  <Typography variant="body2" component="div">
                    <ul style={{ margin: 0, paddingLeft: '20px' }}>
                      <li><strong>Fallback:</strong> Used when no custom version is active</li>
                      <li><strong>Reference:</strong> Shows the recommended AI behavior patterns</li>
                      <li><strong>Template:</strong> Copy this as a starting point for your custom versions</li>
                    </ul>
                  </Typography>
                </Alert>

                <Typography variant="body2" color="text.secondary" paragraph sx={{ mt: 2 }}>
                  <strong>Note:</strong> To customize the AI's behavior, create your own version below instead of editing this default prompt.
                </Typography>

                <TextField
                  multiline
                  fullWidth
                  minRows={10}
                  maxRows={20}
                  value={defaultPrompt}
                  variant="outlined"
                  InputProps={{
                    readOnly: true,
                  }}
                  sx={{
                    '& .MuiInputBase-input': {
                      fontFamily: 'monospace',
                      fontSize: '0.875rem',
                    },
                  }}
                />
              </>
            )}
          </AccordionDetails>
        </Accordion>
      </Paper>

      {/* Version History */}
      <Paper sx={{ p: 3 }}>
        <Stack direction="row" justifyContent="space-between" alignItems="center" mb={2}>
          <Typography variant="h6">System Prompt Versions</Typography>
          <Button
            variant="contained"
            startIcon={<Add />}
            onClick={() => setCreateDialogOpen(true)}
          >
            Create New Version
          </Button>
        </Stack>

        <Typography variant="body2" color="text.secondary" paragraph>
          Manage custom system prompts with full version history. Each save creates a new version.
        </Typography>

        {versionsLoading ? (
          <Box display="flex" justifyContent="center" p={3}>
            <CircularProgress />
          </Box>
        ) : versions.length === 0 ? (
          <Alert severity="info">
            No custom versions yet. Click "Create New Version" to get started.
          </Alert>
        ) : (
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Version</TableCell>
                  <TableCell>Description</TableCell>
                  <TableCell>Created</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell align="right">Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {versions.map((version) => (
                  <TableRow key={version.id}>
                    <TableCell>
                      <Typography variant="body2" fontWeight={600}>
                        v{version.version}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" sx={{ maxWidth: 300 }}>
                        {version.description || <em>No description</em>}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" color="text.secondary">
                        {new Date(version.createdAt).toLocaleString()}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      {version.isActive && (
                        <Chip
                          label="Active"
                          color="success"
                          size="small"
                          icon={<CheckCircle />}
                        />
                      )}
                      {version.isDefault && (
                        <Chip label="System Default" size="small" color="info" />
                      )}
                    </TableCell>
                    <TableCell align="right">
                      <Stack direction="row" spacing={1} justifyContent="flex-end">
                        <Tooltip title="View">
                          <IconButton
                            size="small"
                            onClick={() => handleViewVersion(version)}
                            color="primary"
                          >
                            <Visibility />
                          </IconButton>
                        </Tooltip>
                        {!version.isActive && !version.isDefault && (
                          <Tooltip title="Activate">
                            <IconButton
                              size="small"
                              onClick={() => handleActivateVersion(version.id)}
                              color="success"
                            >
                              <CheckCircle />
                            </IconButton>
                          </Tooltip>
                        )}
                        {!version.isActive && !version.isDefault && (
                          <Tooltip title="Delete">
                            <IconButton
                              size="small"
                              onClick={() => handleDeleteVersion(version.id, version.version)}
                              color="error"
                            >
                              <Delete />
                            </IconButton>
                          </Tooltip>
                        )}
                      </Stack>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </Paper>

      {/* Create Version Dialog */}
      <Dialog
        open={createDialogOpen}
        onClose={() => !creating && setCreateDialogOpen(false)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>Create New System Prompt Version</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <TextField
              label="Description (optional)"
              fullWidth
              value={newDescription}
              onChange={(e) => setNewDescription(e.target.value)}
              placeholder="e.g., Added code review guidelines"
              helperText="Describe what changed in this version"
            />

            <TextField
              label="System Prompt"
              multiline
              fullWidth
              minRows={15}
              maxRows={25}
              value={newPrompt}
              onChange={(e) => setNewPrompt(e.target.value)}
              placeholder="Enter your custom system prompt..."
              variant="outlined"
              error={isOverLimit}
              helperText={
                isOverLimit
                  ? `Exceeds maximum length by ${characterCount - MAX_CHARACTERS} characters`
                  : `${characterCount.toLocaleString()} / ${MAX_CHARACTERS.toLocaleString()} characters`
              }
            />

            <FormControlLabel
              control={
                <Checkbox
                  checked={activateImmediately}
                  onChange={(e) => setActivateImmediately(e.target.checked)}
                />
              }
              label="Activate this version immediately"
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateDialogOpen(false)} disabled={creating}>
            Cancel
          </Button>
          <Button
            onClick={handleCreateVersion}
            variant="contained"
            disabled={creating || isOverLimit || !newPrompt.trim()}
            startIcon={creating ? <CircularProgress size={20} /> : <Save />}
          >
            {creating ? 'Creating...' : 'Create Version'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* View Version Dialog */}
      <Dialog
        open={viewDialogOpen}
        onClose={() => setViewDialogOpen(false)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>
          Version {viewingVersion?.version}
          {viewingVersion?.isActive && (
            <Chip label="Active" color="success" size="small" sx={{ ml: 2 }} />
          )}
        </DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            {viewingVersion?.description && (
              <Box>
                <Typography variant="subtitle2" color="text.secondary">
                  Description
                </Typography>
                <Typography variant="body1">{viewingVersion.description}</Typography>
              </Box>
            )}

            <Box>
              <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                Created
              </Typography>
              <Typography variant="body2">
                {viewingVersion && new Date(viewingVersion.createdAt).toLocaleString()}
              </Typography>
            </Box>

            <Box>
              <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                Prompt Content
              </Typography>
              <TextField
                multiline
                fullWidth
                minRows={15}
                maxRows={25}
                value={viewingVersion?.prompt || ''}
                variant="outlined"
                InputProps={{
                  readOnly: true,
                }}
                sx={{
                  '& .MuiInputBase-input': {
                    fontFamily: 'monospace',
                    fontSize: '0.875rem',
                  },
                }}
              />
            </Box>
          </Stack>
        </DialogContent>
        <DialogActions>
          {viewingVersion && !viewingVersion.isActive && !viewingVersion.isDefault && (
            <Button
              onClick={() => {
                handleActivateVersion(viewingVersion.id);
                setViewDialogOpen(false);
              }}
              variant="contained"
              color="success"
              startIcon={<CheckCircle />}
            >
              Activate This Version
            </Button>
          )}
          <Button onClick={() => setViewDialogOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
