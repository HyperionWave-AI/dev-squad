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
  Radio,
  RadioGroup,
  FormControl,
  FormLabel,
} from '@mui/material';
import { Add, Delete, FolderOpen, Refresh, PlayArrow, Stop, Sync } from '@mui/icons-material';
import { restCodeClient } from '../../services/restCodeClient';
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
  const [customIncludePatterns, setCustomIncludePatterns] = useState('');
  const [excludePatterns, setExcludePatterns] = useState('node_modules,dist,build,.git');
  const [chunkSize, setChunkSize] = useState<'xs' | 's' | 'm' | 'l' | 'xl'>('m');
  const [actionLoading, setActionLoading] = useState(false);

  const FILE_PATTERNS = [
    { label: 'Go (*.go)', value: '*.go' },
    { label: 'TypeScript (*.ts)', value: '*.ts' },
    { label: 'TSX (*.tsx)', value: '*.tsx' },
    { label: 'JavaScript (*.js)', value: '*.js' },
    { label: 'Python (*.py)', value: '*.py' },
    { label: 'Java (*.java)', value: '*.java' },
  ];

  const CHUNK_SIZE_OPTIONS = [
    { value: 'xs', label: 'Extra Small (100 lines)', description: 'Best for fine-grained searches' },
    { value: 's', label: 'Small (250 lines)', description: 'Good for focused code sections' },
    { value: 'm', label: 'Medium (500 lines)', description: 'Balanced for most use cases' },
    { value: 'l', label: 'Large (1000 lines)', description: 'Good for larger context' },
    { value: 'xl', label: 'Extra Large (2000 lines)', description: 'Maximum context per chunk' },
  ];

  const loadStatus = async () => {
    try {
      setLoading(true);
      const data = await restCodeClient.getStatus();
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

      // Combine selected patterns with custom patterns
      const customPatterns = customIncludePatterns
        .split(',')
        .map(p => p.trim())
        .filter(p => p.length > 0);

      const allIncludePatterns = [...selectedPatterns, ...customPatterns];

      // Parse exclude patterns
      const excludePatternsArray = excludePatterns
        .split(',')
        .map(p => p.trim())
        .filter(p => p.length > 0);

      await restCodeClient.addFolder({
        folderPath: folderPath.trim(),
        includePatterns: allIncludePatterns.length > 0 ? allIncludePatterns : undefined,
        excludePatterns: excludePatternsArray.length > 0 ? excludePatternsArray : undefined,
        chunkSize: chunkSize,
      });

      // Trigger scan for new folder
      await restCodeClient.scan(folderPath.trim());

      // Refresh status
      await loadStatus();

      // Reset form
      setFolderPath('');
      setSelectedPatterns(['*.go', '*.ts', '*.tsx', '*.js']);
      setCustomIncludePatterns('');
      setExcludePatterns('node_modules,dist,build,.git');
      setChunkSize('m');
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
      await restCodeClient.removeFolder(configId);
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

  const handleRefresh = async () => {
    await loadStatus();
  };

  const handleEnableWatcher = async () => {
    try {
      setActionLoading(true);
      const result = await restCodeClient.enableWatcher();
      alert(result.message || 'File watcher enabled successfully');
      await loadStatus();
    } catch (err) {
      console.error('Failed to enable watcher:', err);
      alert(err instanceof Error ? err.message : 'Failed to enable file watcher');
    } finally {
      setActionLoading(false);
    }
  };

  const handleDisableWatcher = async () => {
    try {
      setActionLoading(true);
      const result = await restCodeClient.disableWatcher();
      alert(result.message || 'File watcher disabled successfully');
      await loadStatus();
    } catch (err) {
      console.error('Failed to disable watcher:', err);
      alert(err instanceof Error ? err.message : 'Failed to disable file watcher');
    } finally {
      setActionLoading(false);
    }
  };

  const handleReindexAll = async () => {
    if (!confirm('This will reindex all files in all folders. This may take some time. Continue?')) {
      return;
    }

    try {
      setActionLoading(true);
      const result = await restCodeClient.reindexAll();
      alert(
        result.message ||
        `Reindexed ${result.foldersReindexed} folders, ${result.totalFilesIndexed} files`
      );
      await loadStatus();
    } catch (err) {
      console.error('Failed to reindex all:', err);
      alert(err instanceof Error ? err.message : 'Failed to reindex all files');
    } finally {
      setActionLoading(false);
    }
  };

  const watcherRunning = status?.watcherStatus === 'running';

  return (
    <>
      <Card>
        <CardContent>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
            <Typography variant="h6" sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <FolderOpen />
              Folder Configuration
            </Typography>
            <IconButton onClick={handleRefresh} disabled={loading} size="small">
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
                          onClick={() => handleRemoveFolder(folder.configId)}
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

          {/* File Watcher Controls */}
          <Box sx={{ mt: 2, display: 'flex', gap: 1 }}>
            {watcherRunning ? (
              <Button
                variant="outlined"
                color="warning"
                startIcon={<Stop />}
                onClick={handleDisableWatcher}
                disabled={actionLoading || loading}
                fullWidth
                size="small"
              >
                Disable Indexing
              </Button>
            ) : (
              <Button
                variant="outlined"
                color="success"
                startIcon={<PlayArrow />}
                onClick={handleEnableWatcher}
                disabled={actionLoading || loading}
                fullWidth
                size="small"
              >
                Enable Indexing
              </Button>
            )}
          </Box>

          {/* Reindex All Button */}
          <Button
            variant="outlined"
            color="primary"
            startIcon={<Sync />}
            onClick={handleReindexAll}
            disabled={actionLoading || loading || !status?.folders?.length}
            fullWidth
            sx={{ mt: 1 }}
            size="small"
          >
            {actionLoading ? 'Reindexing...' : 'Reindex All Files'}
          </Button>
        </CardContent>
      </Card>

      {/* Add Folder Dialog */}
      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="md" fullWidth>
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
              label="Custom Include Patterns"
              placeholder="*.json,*.yaml,*.md"
              value={customIncludePatterns}
              onChange={(e) => setCustomIncludePatterns(e.target.value)}
              helperText="Comma-separated list of additional file patterns to include"
            />

            <TextField
              fullWidth
              label="Exclude Patterns"
              placeholder="node_modules,dist,build,.git"
              value={excludePatterns}
              onChange={(e) => setExcludePatterns(e.target.value)}
              helperText="Comma-separated list of patterns to exclude"
            />

            <FormControl component="fieldset">
              <FormLabel component="legend">Chunk Size</FormLabel>
              <RadioGroup
                value={chunkSize}
                onChange={(e) => setChunkSize(e.target.value as 'xs' | 's' | 'm' | 'l' | 'xl')}
              >
                {CHUNK_SIZE_OPTIONS.map((option) => (
                  <FormControlLabel
                    key={option.value}
                    value={option.value}
                    control={<Radio />}
                    label={
                      <Box>
                        <Typography variant="body2">{option.label}</Typography>
                        <Typography variant="caption" color="text.secondary">
                          {option.description}
                        </Typography>
                      </Box>
                    }
                  />
                ))}
              </RadioGroup>
            </FormControl>
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
