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
} from '@mui/material';
import { Save, Clear } from '@mui/icons-material';
import { aiService } from '../services/aiService';

const MAX_CHARACTERS = 10000;

export function SettingsPage() {
  const [systemPrompt, setSystemPrompt] = useState('');
  const [originalPrompt, setOriginalPrompt] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Load system prompt on mount
  useEffect(() => {
    loadSystemPrompt();
  }, []);

  const loadSystemPrompt = async () => {
    setLoading(true);
    setError(null);
    try {
      const prompt = await aiService.getSystemPrompt();
      setSystemPrompt(prompt);
      setOriginalPrompt(prompt);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load system prompt');
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    if (systemPrompt.length > MAX_CHARACTERS) {
      setError(`System prompt exceeds maximum length of ${MAX_CHARACTERS} characters`);
      return;
    }

    setSaving(true);
    setError(null);
    setSuccessMessage(null);

    try {
      await aiService.updateSystemPrompt(systemPrompt);
      setOriginalPrompt(systemPrompt);
      setSuccessMessage('System prompt saved successfully!');

      // Clear success message after 3 seconds
      setTimeout(() => setSuccessMessage(null), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save system prompt');
    } finally {
      setSaving(false);
    }
  };

  const handleClear = () => {
    setSystemPrompt(originalPrompt);
    setError(null);
    setSuccessMessage(null);
  };

  const characterCount = systemPrompt.length;
  const isModified = systemPrompt !== originalPrompt;
  const isOverLimit = characterCount > MAX_CHARACTERS;

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      <Typography variant="h4" gutterBottom fontWeight={600}>
        Settings
      </Typography>
      <Typography variant="body1" color="text.secondary" paragraph>
        Configure AI behavior and customize system prompts
      </Typography>

      <Paper sx={{ p: 3, mt: 3 }}>
        <Typography variant="h6" gutterBottom>
          System Prompt
        </Typography>
        <Typography variant="body2" color="text.secondary" paragraph>
          Customize the AI's behavior by defining a system prompt. This will be applied to all chat interactions.
        </Typography>

        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {successMessage && (
          <Alert severity="success" sx={{ mb: 2 }}>
            {successMessage}
          </Alert>
        )}

        <TextField
          multiline
          fullWidth
          minRows={10}
          maxRows={20}
          value={systemPrompt}
          onChange={(e) => setSystemPrompt(e.target.value)}
          placeholder="Enter system prompt..."
          variant="outlined"
          sx={{ mb: 2 }}
          error={isOverLimit}
        />

        <Stack direction="row" spacing={2} alignItems="center" justifyContent="space-between">
          <Typography
            variant="body2"
            color={isOverLimit ? 'error' : 'text.secondary'}
          >
            Characters: {characterCount.toLocaleString()} / {MAX_CHARACTERS.toLocaleString()}
          </Typography>

          <Stack direction="row" spacing={2}>
            <Button
              variant="outlined"
              startIcon={<Clear />}
              onClick={handleClear}
              disabled={!isModified || saving}
            >
              Reset
            </Button>
            <Button
              variant="contained"
              startIcon={saving ? <CircularProgress size={20} /> : <Save />}
              onClick={handleSave}
              disabled={!isModified || saving || isOverLimit}
            >
              {saving ? 'Saving...' : 'Save'}
            </Button>
          </Stack>
        </Stack>
      </Paper>
    </Box>
  );
}
