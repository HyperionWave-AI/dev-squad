import React, { useState, useEffect } from 'react';
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
import { mcpServerService, type UpdateMCPServerRequest, type MCPServer } from '../services/mcpServerService';

interface EditMCPServerDialogProps {
  open: boolean;
  server: MCPServer | null;
  onClose: () => void;
  onSuccess: () => void;
  onError: (message: string) => void;
}

export const EditMCPServerDialog: React.FC<EditMCPServerDialogProps> = ({
  open,
  server,
  onClose,
  onSuccess,
  onError,
}) => {
  const [formData, setFormData] = useState<UpdateMCPServerRequest>({
    serverUrl: '',
    description: '',
  });
  const [headersJson, setHeadersJson] = useState<string>('');
  const [loading, setLoading] = useState(false);
  const [validationError, setValidationError] = useState<string>('');

  // Populate form when server prop changes
  useEffect(() => {
    if (server) {
      setFormData({
        serverUrl: server.serverUrl,
        description: server.description || '',
      });
      setHeadersJson(server.headers ? JSON.stringify(server.headers, null, 2) : '');
    }
  }, [server]);

  const handleClose = () => {
    if (!loading) {
      setFormData({ serverUrl: '', description: '' });
      setHeadersJson('');
      setValidationError('');
      onClose();
    }
  };

  const validateForm = (): boolean => {
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

    // Validate headers JSON (optional)
    if (headersJson.trim()) {
      try {
        const parsed = JSON.parse(headersJson);
        if (typeof parsed !== 'object' || Array.isArray(parsed)) {
          setValidationError('Headers must be a valid JSON object');
          return false;
        }
      } catch {
        setValidationError('Invalid JSON format for headers');
        return false;
      }
    }

    setValidationError('');
    return true;
  };

  const handleSubmit = async () => {
    if (!server || !validateForm()) {
      return;
    }

    setLoading(true);
    try {
      const request: UpdateMCPServerRequest = {
        serverUrl: formData.serverUrl.trim(),
        description: formData.description?.trim() || '',
      };

      // Parse and include headers if provided
      if (headersJson.trim()) {
        request.headers = JSON.parse(headersJson);
      }

      await mcpServerService.updateMCPServer(server.serverName, request);
      handleClose();
      onSuccess();
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to update MCP server';
      onError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const handleFieldChange = (field: keyof UpdateMCPServerRequest) => (
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
      <DialogTitle>Edit MCP Server</DialogTitle>
      <DialogContent>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
          {validationError && (
            <Alert severity="error" sx={{ mb: 1 }}>
              {validationError}
            </Alert>
          )}

          <TextField
            label="Server Name"
            value={server?.serverName || ''}
            fullWidth
            disabled
            helperText="Server name cannot be changed"
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

          <TextField
            label="Headers (JSON)"
            value={headersJson}
            onChange={(e) => {
              setHeadersJson(e.target.value);
              if (validationError) {
                setValidationError('');
              }
            }}
            fullWidth
            disabled={loading}
            multiline
            rows={4}
            helperText='Optional HTTP headers as JSON object. Example: {"Authorization": "Bearer token"}'
            placeholder='{"Authorization": "Bearer token", "X-Custom-Header": "value"}'
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
          disabled={loading || !formData.serverUrl}
          startIcon={loading ? <CircularProgress size={16} /> : null}
        >
          {loading ? 'Updating...' : 'Update Server'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};
