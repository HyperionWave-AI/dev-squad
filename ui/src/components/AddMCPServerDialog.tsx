import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  Box,
  Alert,
  CircularProgress,
} from '@mui/material';
import { mcpServerService, type AddMCPServerRequest } from '../services/mcpServerService';

interface AddMCPServerDialogProps {
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
  onError: (message: string) => void;
}

export const AddMCPServerDialog: React.FC<AddMCPServerDialogProps> = ({
  open,
  onClose,
  onSuccess,
  onError,
}) => {
  const [formData, setFormData] = useState<AddMCPServerRequest>({
    serverName: '',
    serverUrl: '',
    description: '',
  });
  const [loading, setLoading] = useState(false);
  const [validationError, setValidationError] = useState<string>('');

  const handleClose = () => {
    if (!loading) {
      setFormData({ serverName: '', serverUrl: '', description: '' });
      setValidationError('');
      onClose();
    }
  };

  const validateForm = (): boolean => {
    // Validate server name (alphanumeric, dash, underscore only)
    const serverNameRegex = /^[a-zA-Z0-9_-]+$/;
    if (!formData.serverName.trim()) {
      setValidationError('Server name is required');
      return false;
    }
    if (!serverNameRegex.test(formData.serverName)) {
      setValidationError('Server name can only contain alphanumeric characters, dashes, and underscores');
      return false;
    }

    // Validate server URL
    if (!formData.serverUrl.trim()) {
      setValidationError('Server URL is required');
      return false;
    }

    try {
      new URL(formData.serverUrl);
    } catch {
      setValidationError('Invalid server URL format');
      return false;
    }

    setValidationError('');
    return true;
  };

  const handleSubmit = async () => {
    if (!validateForm()) {
      return;
    }

    setLoading(true);
    try {
      const request: AddMCPServerRequest = {
        serverName: formData.serverName.trim(),
        serverUrl: formData.serverUrl.trim(),
        description: formData.description?.trim() || '',
      };

      await mcpServerService.addMCPServer(request);
      handleClose();
      onSuccess();
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to add MCP server';
      onError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const handleFieldChange = (field: keyof AddMCPServerRequest) => (
    event: React.ChangeEvent<HTMLInputElement>
  ) => {
    setFormData((prev) => ({
      ...prev,
      [field]: event.target.value,
    }));
    // Clear validation error when user starts typing
    if (validationError) {
      setValidationError('');
    }
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>Add MCP Server</DialogTitle>
      <DialogContent>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
          {validationError && (
            <Alert severity="error" sx={{ mb: 1 }}>
              {validationError}
            </Alert>
          )}

          <TextField
            label="Server Name"
            value={formData.serverName}
            onChange={handleFieldChange('serverName')}
            required
            fullWidth
            disabled={loading}
            helperText="Unique name for this MCP server (alphanumeric, dash, underscore only)"
            placeholder="my-mcp-server"
          />

          <TextField
            label="Server URL"
            value={formData.serverUrl}
            onChange={handleFieldChange('serverUrl')}
            required
            fullWidth
            disabled={loading}
            helperText="Full URL to the MCP server endpoint"
            placeholder="http://localhost:3000/mcp"
          />

          <TextField
            label="Description"
            value={formData.description}
            onChange={handleFieldChange('description')}
            fullWidth
            disabled={loading}
            multiline
            rows={3}
            helperText="Optional description of what this server provides"
            placeholder="Describe the tools and capabilities this server provides..."
          />
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose} disabled={loading}>
          Cancel
        </Button>
        <Button
          onClick={handleSubmit}
          variant="contained"
          disabled={loading || !formData.serverName || !formData.serverUrl}
          startIcon={loading ? <CircularProgress size={16} /> : null}
        >
          {loading ? 'Adding...' : 'Add Server'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};
