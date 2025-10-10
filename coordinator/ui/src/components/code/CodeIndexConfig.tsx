import React, { useState, useEffect } from 'react';
import {
  Card,
  CardContent,
  Typography,
  Button,
  List,
  ListItem,
  ListItemText,
  ListItemSecondaryAction,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  FormGroup,
  FormControlLabel,
  Checkbox,
  Box,
  CircularProgress,
} from '@mui/material';
import { Add, Delete, FolderOpen, Refresh } from '@mui/icons-material';
import { codeClient } from '../../services/codeClient';
import type { IndexStatus } from '../../types/codeIndex';

export const CodeIndexConfig: React.FC = () => {
  const [status, setStatus] = useState<IndexStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [folderPath, setFolderPath] = useState('');
  const [selectedPatterns, setSelectedPatterns] = useState<string[]>([
    '*.go',
    '*.ts',
    '*.tsx',
    '*.js',
  ]);
  const [excludePatterns, setExcludePatterns] = useState('node_modules,dist,build,.git');

  const FILE_PATTERNS = [
    { label: 'Go (*.go)', value: '*.go' },
    { label: 'TypeScript (*.ts)', value: '*.ts' },
    { label: 'TSX (*.tsx)', value: '*.tsx' },
    { label: 'JavaScript (*.js)', value: '*.js' },
    { label: 'Python (*.py)', value: '*.py' },
    { label: 'Java (*.java)', value: '*.java' },
  ];

  const loadStatus = async () => {
    try {
      setLoading(true);
      const data = await codeClient.getStatus();
      setStatus(data);
    } catch (err) {
      console.error('Failed to load status:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadStatus();
    // Refresh every 5 seconds
    const interval = setInterval(loadStatus, 5000);
    return () => clearInterval(interval);
  }, []);

  const handleAddFolder = async () => {
    if (!folderPath.trim()) return;

    try {
      setLoading(true);
      const excludeArray = excludePatterns
        .split(',')
        .map((p) => p.trim())
        .filter(Boolean);

      await codeClient.addFolder({
        folderPath: folderPath.trim(),
        filePatterns: selectedPatterns,
        excludePatterns: excludeArray,
      });

      // Trigger scan for new folder
      await codeClient.scan();

      // Refresh status
      await loadStatus();

      // Reset form
      setFolderPath('');
      setSelectedPatterns(['*.go', '*.ts', '*.tsx', '*.js']);
      setExcludePatterns('node_modules,dist,build,.git');
      setDialogOpen(false);
    } catch (err) {
      console.error('Failed to add folder:', err);
      alert(err instanceof Error ? err.message : 'Failed to add folder');
    } finally {
      setLoading(false);
    }
  };

  const handleRemoveFolder = async (configId: string) => {
    if (!confirm('Are you sure you want to remove this folder from the index?')) {
      return;
    }

    try {
      setLoading(true);
      await codeClient.removeFolder(configId);
      await loadStatus();
    } catch (err) {
      console.error('Failed to remove folder:', err);
      alert(err instanceof Error ? err.message : 'Failed to remove folder');
    } finally {
      setLoading(false);
    }
  };

  const handlePatternToggle = (pattern: string) => {
    setSelectedPatterns((prev) =>
      prev.includes(pattern)
        ? prev.filter((p) => p !== pattern)
        : [...prev, pattern]
    );
  };

  const handleScan = async () => {
    try {
      setLoading(true);
      await codeClient.scan();
      await loadStatus();
    } catch (err) {
      console.error('Failed to trigger scan:', err);
      alert(err instanceof Error ? err.message : 'Failed to trigger scan');
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <Card>
        <CardContent>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
            <Typography variant="h6" sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <FolderOpen />
              Folder Configuration
            </Typography>
            <IconButton onClick={handleScan} disabled={loading} size="small">
              <Refresh />
            </IconButton>
          </Box>

          {loading && !status && (
            <Box display="flex" justifyContent="center" py={4}>
              <CircularProgress />
            </Box>
          )}

          {status && (
            <>
              {status.folders?.length > 0 ? (
                <List>
                  {status.folders?.map((folder, index) => (
                    <ListItem key={index}>
                      <ListItemText
                        primary={folder.folderPath}
                        secondary={`${folder.fileCount} files indexed`}
                        primaryTypographyProps={{ variant: 'body2', noWrap: true }}
                        secondaryTypographyProps={{ variant: 'caption' }}
                      />
                      <ListItemSecondaryAction>
                        <IconButton
                          edge="end"
                          onClick={() => handleRemoveFolder(folder.folderPath)}
                          disabled={loading}
                          size="small"
                        >
                          <Delete />
                        </IconButton>
                      </ListItemSecondaryAction>
                    </ListItem>
                  ))}
                </List>
              ) : (
                <Typography variant="body2" color="text.secondary" sx={{ fontStyle: 'italic', py: 2 }}>
                  No folders indexed. Click "Add Folder" to get started.
                </Typography>
              )}
            </>
          )}

          <Button
            variant="outlined"
            startIcon={<Add />}
            onClick={() => setDialogOpen(true)}
            disabled={loading}
            fullWidth
            sx={{ mt: 2 }}
          >
            Add Folder
          </Button>
        </CardContent>
      </Card>

      {/* Add Folder Dialog */}
      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Add Folder to Index</DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, pt: 1 }}>
            <TextField
              fullWidth
              label="Folder Path"
              placeholder="/path/to/your/code"
              value={folderPath}
              onChange={(e) => setFolderPath(e.target.value)}
              helperText="Absolute path to the folder you want to index"
            />

            <Box>
              <Typography variant="subtitle2" gutterBottom>
                File Patterns
              </Typography>
              <FormGroup>
                {FILE_PATTERNS.map((pattern) => (
                  <FormControlLabel
                    key={pattern.value}
                    control={
                      <Checkbox
                        checked={selectedPatterns.includes(pattern.value)}
                        onChange={() => handlePatternToggle(pattern.value)}
                      />
                    }
                    label={pattern.label}
                  />
                ))}
              </FormGroup>
            </Box>

            <TextField
              fullWidth
              label="Exclude Patterns"
              placeholder="node_modules,dist,build,.git"
              value={excludePatterns}
              onChange={(e) => setExcludePatterns(e.target.value)}
              helperText="Comma-separated list of patterns to exclude"
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDialogOpen(false)} disabled={loading}>
            Cancel
          </Button>
          <Button onClick={handleAddFolder} variant="contained" disabled={!folderPath.trim() || loading}>
            {loading ? 'Adding...' : 'Add Folder'}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
};
