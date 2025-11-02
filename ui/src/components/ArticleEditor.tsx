import React, { useState } from 'react';
import {
  Box,
  Paper,
  TextField,
  Button,
  Typography,
  Divider,
  Tabs,
  Tab,
  Alert,
} from '@mui/material';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import type { KnowledgeEntry } from '../services/knowledgeService';

interface ArticleEditorProps {
  entry: KnowledgeEntry;
  onSave: (text: string, metadata: Record<string, any>) => Promise<void>;
  onCancel: () => void;
}

export const ArticleEditor: React.FC<ArticleEditorProps> = ({
  entry,
  onSave,
  onCancel,
}) => {
  const [text, setText] = useState(entry.text);
  const [metadataJson, setMetadataJson] = useState(
    JSON.stringify(entry.metadata || {}, null, 2)
  );
  const [activeTab, setActiveTab] = useState(0);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [metadataError, setMetadataError] = useState<string | null>(null);

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  const validateMetadata = (json: string): Record<string, any> | null => {
    try {
      const parsed = JSON.parse(json);
      if (typeof parsed !== 'object' || Array.isArray(parsed)) {
        setMetadataError('Metadata must be a JSON object');
        return null;
      }
      setMetadataError(null);
      return parsed;
    } catch (e) {
      setMetadataError('Invalid JSON syntax');
      return null;
    }
  };

  const handleMetadataChange = (value: string) => {
    setMetadataJson(value);
    validateMetadata(value);
  };

  const handleSave = async () => {
    if (!text.trim()) {
      setError('Content cannot be empty');
      return;
    }

    const metadata = validateMetadata(metadataJson);
    if (metadata === null) {
      return;
    }

    setSaving(true);
    setError(null);

    try {
      await onSave(text, metadata);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save entry');
      setSaving(false);
    }
  };

  return (
    <Box sx={{ height: '100%', overflow: 'auto', p: 3 }}>
      <Paper elevation={2} sx={{ p: 3 }}>
        {/* Header */}
        <Box sx={{ mb: 3 }}>
          <Typography variant="h6" gutterBottom>
            Edit Entry
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Collection: {entry.collection}
          </Typography>
        </Box>

        <Divider sx={{ mb: 3 }} />

        {/* Error display */}
        {error && (
          <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
            {error}
          </Alert>
        )}

        {/* Content editor with preview */}
        <Box sx={{ mb: 3 }}>
          <Typography variant="subtitle2" gutterBottom>
            Content
          </Typography>
          <Tabs value={activeTab} onChange={handleTabChange} sx={{ mb: 2 }}>
            <Tab label="Edit" />
            <Tab label="Preview" />
          </Tabs>

          {activeTab === 0 && (
            <TextField
              fullWidth
              multiline
              rows={20}
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder="Enter markdown content..."
              variant="outlined"
              sx={{
                '& .MuiInputBase-root': {
                  fontFamily: 'monospace',
                  fontSize: '0.875rem',
                },
              }}
            />
          )}

          {activeTab === 1 && (
            <Box
              sx={{
                border: 1,
                borderColor: 'divider',
                borderRadius: 1,
                p: 2,
                minHeight: 400,
                maxHeight: 600,
                overflow: 'auto',
                bgcolor: 'background.default',
                '& h1': { fontSize: '2rem', fontWeight: 600, mb: 2, mt: 3 },
                '& h2': { fontSize: '1.5rem', fontWeight: 600, mb: 2, mt: 2 },
                '& h3': { fontSize: '1.25rem', fontWeight: 600, mb: 1.5, mt: 2 },
                '& h4': { fontSize: '1.1rem', fontWeight: 600, mb: 1, mt: 1.5 },
                '& p': { mb: 2, lineHeight: 1.7 },
                '& code': {
                  bgcolor: 'grey.100',
                  color: 'error.dark',
                  px: 0.5,
                  py: 0.25,
                  borderRadius: 0.5,
                  fontFamily: 'monospace',
                  fontSize: '0.875rem',
                },
                '& pre': {
                  bgcolor: 'grey.100',
                  p: 2,
                  borderRadius: 1,
                  overflow: 'auto',
                  mb: 2,
                },
                '& pre code': {
                  bgcolor: 'transparent',
                  color: 'inherit',
                  p: 0,
                },
                '& ul, & ol': { mb: 2, pl: 3 },
                '& li': { mb: 0.5 },
                '& blockquote': {
                  borderLeft: 4,
                  borderColor: 'primary.main',
                  pl: 2,
                  ml: 0,
                  fontStyle: 'italic',
                  color: 'text.secondary',
                },
                '& table': {
                  borderCollapse: 'collapse',
                  width: '100%',
                  mb: 2,
                },
                '& th, & td': {
                  border: 1,
                  borderColor: 'divider',
                  p: 1,
                  textAlign: 'left',
                },
                '& th': {
                  bgcolor: 'grey.100',
                  fontWeight: 600,
                },
              }}
            >
              <ReactMarkdown remarkPlugins={[remarkGfm]}>
                {text || '*No content to preview*'}
              </ReactMarkdown>
            </Box>
          )}
        </Box>

        {/* Metadata editor */}
        <Box sx={{ mb: 3 }}>
          <Typography variant="subtitle2" gutterBottom>
            Metadata (JSON)
          </Typography>
          <TextField
            fullWidth
            multiline
            rows={6}
            value={metadataJson}
            onChange={(e) => handleMetadataChange(e.target.value)}
            placeholder='{"key": "value"}'
            variant="outlined"
            error={!!metadataError}
            helperText={metadataError}
            sx={{
              '& .MuiInputBase-root': {
                fontFamily: 'monospace',
                fontSize: '0.875rem',
              },
            }}
          />
        </Box>

        <Divider sx={{ mb: 3 }} />

        {/* Action buttons */}
        <Box sx={{ display: 'flex', gap: 2, justifyContent: 'flex-end' }}>
          <Button
            variant="outlined"
            onClick={onCancel}
            disabled={saving}
          >
            Cancel
          </Button>
          <Button
            variant="contained"
            color="primary"
            onClick={handleSave}
            disabled={saving || !!metadataError}
          >
            {saving ? 'Saving...' : 'Save'}
          </Button>
        </Box>
      </Paper>
    </Box>
  );
};
