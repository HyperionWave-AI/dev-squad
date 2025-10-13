import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  IconButton,
  Box,
  Typography,
  Checkbox,
  FormControlLabel,
  Snackbar,
  Alert,
  Stack,
} from '@mui/material';
import { Add as AddIcon, Delete as DeleteIcon } from '@mui/icons-material';
import { httpToolsService } from '../services/httpToolsService';
import type { HTTPToolDefinition } from '../services/httpToolsService';

interface AddHTTPToolDialogProps {
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

interface HeaderEntry {
  key: string;
  value: string;
}

interface ParameterEntry {
  name: string;
  type: 'string' | 'number' | 'boolean' | 'object';
  required: boolean;
  description?: string;
}

export const AddHTTPToolDialog: React.FC<AddHTTPToolDialogProps> = ({
  open,
  onClose,
  onSuccess,
}) => {
  // Form state
  const [toolName, setToolName] = useState('');
  const [description, setDescription] = useState('');
  const [endpoint, setEndpoint] = useState('');
  const [httpMethod, setHttpMethod] = useState<'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'>('GET');
  const [headers, setHeaders] = useState<HeaderEntry[]>([]);
  const [parameters, setParameters] = useState<ParameterEntry[]>([]);
  const [authType, setAuthType] = useState<'none' | 'bearer' | 'apiKey' | 'basic'>('none');
  const [authConfig, setAuthConfig] = useState<Record<string, string>>({});

  // Validation errors
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);

  // Snackbar state
  const [snackbar, setSnackbar] = useState<{ open: boolean; message: string; severity: 'success' | 'error' }>({
    open: false,
    message: '',
    severity: 'success',
  });

  // Validation functions
  const validateToolName = (name: string): string | null => {
    if (!name.trim()) return 'Tool name is required';
    if (!/^[a-zA-Z0-9_]+$/.test(name)) return 'Tool name can only contain letters, numbers, and underscores';
    return null;
  };

  const validateDescription = (desc: string): string | null => {
    if (!desc.trim()) return 'Description is required';
    if (desc.length < 10) return 'Description must be at least 10 characters';
    return null;
  };

  const validateEndpoint = (url: string): string | null => {
    if (!url.trim()) return 'Endpoint URL is required';
    try {
      new URL(url);
      return null;
    } catch {
      return 'Invalid URL format';
    }
  };

  // Header management
  const addHeader = () => {
    setHeaders([...headers, { key: '', value: '' }]);
  };

  const updateHeader = (index: number, field: 'key' | 'value', value: string) => {
    const updated = [...headers];
    updated[index][field] = value;
    setHeaders(updated);
  };

  const removeHeader = (index: number) => {
    setHeaders(headers.filter((_, i) => i !== index));
  };

  // Parameter management
  const addParameter = () => {
    setParameters([...parameters, { name: '', type: 'string', required: false }]);
  };

  const updateParameter = (index: number, field: keyof ParameterEntry, value: any) => {
    const updated = [...parameters];
    updated[index] = { ...updated[index], [field]: value };
    setParameters(updated);
  };

  const removeParameter = (index: number) => {
    setParameters(parameters.filter((_, i) => i !== index));
  };

  // Auth config management
  const updateAuthConfig = (key: string, value: string) => {
    setAuthConfig({ ...authConfig, [key]: value });
  };

  // Form submission
  const handleSubmit = async () => {
    // Validate all fields
    const newErrors: Record<string, string> = {};

    const toolNameError = validateToolName(toolName);
    if (toolNameError) newErrors.toolName = toolNameError;

    const descriptionError = validateDescription(description);
    if (descriptionError) newErrors.description = descriptionError;

    const endpointError = validateEndpoint(endpoint);
    if (endpointError) newErrors.endpoint = endpointError;

    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors);
      return;
    }

    setSubmitting(true);
    setErrors({});

    try {
      const tool: HTTPToolDefinition = {
        toolName: toolName.trim(),
        description: description.trim(),
        endpoint: endpoint.trim(),
        httpMethod,
        headers: headers.filter(h => h.key.trim() && h.value.trim()),
        parameters: parameters.filter(p => p.name.trim()),
        authType: authType === 'none' ? undefined : authType,
        authConfig: authType === 'none' ? undefined : authConfig,
      };

      await httpToolsService.addHTTPTool(tool);

      setSnackbar({
        open: true,
        message: 'HTTP tool created successfully!',
        severity: 'success',
      });

      // Reset form
      resetForm();

      // Notify parent and close
      onSuccess();
      setTimeout(() => onClose(), 500);
    } catch (error) {
      setSnackbar({
        open: true,
        message: error instanceof Error ? error.message : 'Failed to create HTTP tool',
        severity: 'error',
      });
    } finally {
      setSubmitting(false);
    }
  };

  const resetForm = () => {
    setToolName('');
    setDescription('');
    setEndpoint('');
    setHttpMethod('GET');
    setHeaders([]);
    setParameters([]);
    setAuthType('none');
    setAuthConfig({});
    setErrors({});
  };

  const handleClose = () => {
    if (!submitting) {
      resetForm();
      onClose();
    }
  };

  return (
    <>
      <Dialog open={open} onClose={handleClose} maxWidth="md" fullWidth>
        <DialogTitle>Add HTTP Tool</DialogTitle>
        <DialogContent>
          <Stack spacing={3} sx={{ mt: 1 }}>
            {/* Tool Name */}
            <TextField
              label="Tool Name"
              value={toolName}
              onChange={(e) => setToolName(e.target.value)}
              error={!!errors.toolName}
              helperText={errors.toolName || 'Use alphanumeric characters and underscores only'}
              required
              fullWidth
            />

            {/* Description */}
            <TextField
              label="Description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              error={!!errors.description}
              helperText={errors.description || 'Describe what this tool does (used for semantic discovery)'}
              required
              multiline
              rows={3}
              fullWidth
            />

            {/* Endpoint */}
            <TextField
              label="Endpoint URL"
              value={endpoint}
              onChange={(e) => setEndpoint(e.target.value)}
              error={!!errors.endpoint}
              helperText={errors.endpoint || 'Full URL including protocol (e.g., https://api.example.com/data)'}
              required
              fullWidth
            />

            {/* HTTP Method */}
            <FormControl fullWidth required>
              <InputLabel>HTTP Method</InputLabel>
              <Select
                value={httpMethod}
                onChange={(e) => setHttpMethod(e.target.value as any)}
                label="HTTP Method"
              >
                <MenuItem value="GET">GET</MenuItem>
                <MenuItem value="POST">POST</MenuItem>
                <MenuItem value="PUT">PUT</MenuItem>
                <MenuItem value="DELETE">DELETE</MenuItem>
                <MenuItem value="PATCH">PATCH</MenuItem>
              </Select>
            </FormControl>

            {/* Headers */}
            <Box>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
                <Typography variant="subtitle2">Headers (Optional)</Typography>
                <Button startIcon={<AddIcon />} onClick={addHeader} size="small">
                  Add Header
                </Button>
              </Box>
              {headers.map((header, index) => (
                <Box key={index} sx={{ display: 'flex', gap: 1, mb: 1 }}>
                  <TextField
                    label="Key"
                    value={header.key}
                    onChange={(e) => updateHeader(index, 'key', e.target.value)}
                    size="small"
                    fullWidth
                  />
                  <TextField
                    label="Value"
                    value={header.value}
                    onChange={(e) => updateHeader(index, 'value', e.target.value)}
                    size="small"
                    fullWidth
                  />
                  <IconButton onClick={() => removeHeader(index)} size="small" color="error">
                    <DeleteIcon />
                  </IconButton>
                </Box>
              ))}
            </Box>

            {/* Parameters */}
            <Box>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
                <Typography variant="subtitle2">Parameters (Optional)</Typography>
                <Button startIcon={<AddIcon />} onClick={addParameter} size="small">
                  Add Parameter
                </Button>
              </Box>
              {parameters.map((param, index) => (
                <Box key={index} sx={{ display: 'flex', gap: 1, mb: 1, alignItems: 'flex-start' }}>
                  <TextField
                    label="Name"
                    value={param.name}
                    onChange={(e) => updateParameter(index, 'name', e.target.value)}
                    size="small"
                    sx={{ flex: 2 }}
                  />
                  <FormControl size="small" sx={{ flex: 1 }}>
                    <InputLabel>Type</InputLabel>
                    <Select
                      value={param.type}
                      onChange={(e) => updateParameter(index, 'type', e.target.value)}
                      label="Type"
                    >
                      <MenuItem value="string">String</MenuItem>
                      <MenuItem value="number">Number</MenuItem>
                      <MenuItem value="boolean">Boolean</MenuItem>
                      <MenuItem value="object">Object</MenuItem>
                    </Select>
                  </FormControl>
                  <FormControlLabel
                    control={
                      <Checkbox
                        checked={param.required}
                        onChange={(e) => updateParameter(index, 'required', e.target.checked)}
                      />
                    }
                    label="Required"
                  />
                  <IconButton onClick={() => removeParameter(index)} size="small" color="error">
                    <DeleteIcon />
                  </IconButton>
                </Box>
              ))}
            </Box>

            {/* Auth Type */}
            <FormControl fullWidth>
              <InputLabel>Authentication</InputLabel>
              <Select
                value={authType}
                onChange={(e) => setAuthType(e.target.value as any)}
                label="Authentication"
              >
                <MenuItem value="none">None</MenuItem>
                <MenuItem value="bearer">Bearer Token</MenuItem>
                <MenuItem value="apiKey">API Key</MenuItem>
                <MenuItem value="basic">Basic Auth</MenuItem>
              </Select>
            </FormControl>

            {/* Auth Config */}
            {authType === 'bearer' && (
              <TextField
                label="Bearer Token"
                value={authConfig.token || ''}
                onChange={(e) => updateAuthConfig('token', e.target.value)}
                fullWidth
                type="password"
              />
            )}
            {authType === 'apiKey' && (
              <>
                <TextField
                  label="API Key Header Name"
                  value={authConfig.headerName || ''}
                  onChange={(e) => updateAuthConfig('headerName', e.target.value)}
                  fullWidth
                  placeholder="e.g., X-API-Key"
                />
                <TextField
                  label="API Key Value"
                  value={authConfig.apiKey || ''}
                  onChange={(e) => updateAuthConfig('apiKey', e.target.value)}
                  fullWidth
                  type="password"
                />
              </>
            )}
            {authType === 'basic' && (
              <>
                <TextField
                  label="Username"
                  value={authConfig.username || ''}
                  onChange={(e) => updateAuthConfig('username', e.target.value)}
                  fullWidth
                />
                <TextField
                  label="Password"
                  value={authConfig.password || ''}
                  onChange={(e) => updateAuthConfig('password', e.target.value)}
                  fullWidth
                  type="password"
                />
              </>
            )}
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleClose} disabled={submitting}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            variant="contained"
            disabled={submitting}
          >
            {submitting ? 'Creating...' : 'Create Tool'}
          </Button>
        </DialogActions>
      </Dialog>

      <Snackbar
        open={snackbar.open}
        autoHideDuration={6000}
        onClose={() => setSnackbar({ ...snackbar, open: false })}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert severity={snackbar.severity} onClose={() => setSnackbar({ ...snackbar, open: false })}>
          {snackbar.message}
        </Alert>
      </Snackbar>
    </>
  );
};
