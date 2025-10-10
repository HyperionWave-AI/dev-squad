import React, { useEffect, useState } from 'react';
import {
  Card,
  CardContent,
  Typography,
  Chip,
  List,
  ListItem,
  ListItemText,
  Box,
  CircularProgress,
} from '@mui/material';
import { FolderOpen, CheckCircle, Error as ErrorIcon } from '@mui/icons-material';
import { codeClient } from '../../services/codeClient';
import type { IndexStatus as IIndexStatus } from '../../types/codeIndex';

export const IndexStatus: React.FC = () => {
  const [status, setStatus] = useState<IIndexStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadStatus = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await codeClient.getStatus();
      setStatus(data);
    } catch (err) {
      console.error('Failed to load index status:', err);
      setError(err instanceof Error ? err.message : 'Failed to load status');
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

  const formatSize = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
  };

  if (loading && !status) {
    return (
      <Card>
        <CardContent>
          <Box display="flex" justifyContent="center" alignItems="center" minHeight={200}>
            <CircularProgress />
          </Box>
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card>
        <CardContent>
          <Box display="flex" alignItems="center" gap={1} color="error.main">
            <ErrorIcon />
            <Typography variant="body2">{error}</Typography>
          </Box>
        </CardContent>
      </Card>
    );
  }

  if (!status) {
    return null;
  }

  const watcherColor = status.watcherStatus === 'running' ? 'success' : 'error';
  const watcherIcon = status.watcherStatus === 'running' ? <CheckCircle fontSize="small" /> : <ErrorIcon fontSize="small" />;

  return (
    <Card>
      <CardContent>
        <Typography variant="h6" gutterBottom sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <FolderOpen />
          Index Status
        </Typography>

        {/* Summary Stats */}
        <Box sx={{ mb: 2 }}>
          <Typography variant="body2" color="text.secondary">
            Total Folders: <strong>{status.totalFolders}</strong>
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Total Files: <strong>{status.totalFiles}</strong>
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Total Size: <strong>{formatSize(status.totalSize)}</strong>
          </Typography>
          <Box sx={{ mt: 1 }}>
            <Chip
              label={status.watcherStatus === 'running' ? 'Watcher Running' : 'Watcher Stopped'}
              color={watcherColor}
              size="small"
              icon={watcherIcon}
            />
          </Box>
        </Box>

        {/* Folder List */}
        {status.folders?.length > 0 && (
          <>
            <Typography variant="subtitle2" gutterBottom>
              Indexed Folders
            </Typography>
            <List dense>
              {status.folders?.map((folder, index) => (
                <ListItem key={index} sx={{ px: 0 }}>
                  <ListItemText
                    primary={folder.folderPath}
                    secondary={`${folder.fileCount} files`}
                    primaryTypographyProps={{ variant: 'body2', noWrap: true }}
                    secondaryTypographyProps={{ variant: 'caption' }}
                  />
                  {folder.enabled && (
                    <Chip label="Enabled" size="small" color="primary" variant="outlined" />
                  )}
                </ListItem>
              ))}
            </List>
          </>
        )}

        {(!status.folders || status.folders.length === 0) && (
          <Typography variant="body2" color="text.secondary" sx={{ fontStyle: 'italic' }}>
            No folders indexed yet
          </Typography>
        )}
      </CardContent>
    </Card>
  );
};
