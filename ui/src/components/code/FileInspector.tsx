import React, { useEffect, useState } from 'react';
import {
  Drawer,
  Box,
  Typography,
  IconButton,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Alert,
  Divider,
} from '@mui/material';
import { Close, InsertDriveFile } from '@mui/icons-material';
import type { FileDetails, FileChunkDetails } from '../../types/codeIndex';
import { restCodeClient } from '../../services/restCodeClient';
import { ChunkList } from './ChunkList';

interface FileInspectorProps {
  open: boolean;
  onClose: () => void;
  fileId: string | null;
}

export const FileInspector: React.FC<FileInspectorProps> = ({ open, onClose, fileId }) => {
  const [fileDetails, setFileDetails] = useState<FileDetails | null>(null);
  const [chunks, setChunks] = useState<FileChunkDetails[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!fileId || !open) {
      return;
    }

    const fetchFileData = async () => {
      setLoading(true);
      setError(null);
      try {
        const [file, fileChunks] = await Promise.all([
          restCodeClient.getFile(fileId),
          restCodeClient.getFileChunks(fileId),
        ]);
        setFileDetails(file);
        setChunks(fileChunks);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load file details');
        console.error('Failed to fetch file data:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchFileData();
  }, [fileId, open]);

  const formatFileSize = (bytes: number): string => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  const formatDate = (dateString: string): string => {
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  return (
    <Drawer
      anchor="right"
      open={open}
      onClose={onClose}
      sx={{
        '& .MuiDrawer-paper': {
          width: 500,
          maxWidth: '90vw',
        },
      }}
    >
      <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
        {/* Header */}
        <Box
          sx={{
            p: 2,
            display: 'flex',
            alignItems: 'center',
            gap: 1,
            borderBottom: '1px solid',
            borderColor: 'divider',
          }}
        >
          <InsertDriveFile color="primary" />
          <Typography variant="h6" sx={{ flexGrow: 1 }}>
            File Inspector
          </Typography>
          <IconButton onClick={onClose} size="small">
            <Close />
          </IconButton>
        </Box>

        {/* Content */}
        <Box sx={{ flexGrow: 1, overflow: 'auto', p: 2 }}>
          {loading ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', py: 4 }}>
              <CircularProgress />
            </Box>
          ) : error ? (
            <Alert severity="error" sx={{ mb: 2 }}>
              {error}
            </Alert>
          ) : fileDetails ? (
            <>
              {/* File Metadata Card */}
              <Card sx={{ mb: 2 }}>
                <CardContent>
                  <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                    File Path
                  </Typography>
                  <Typography
                    variant="body2"
                    sx={{
                      fontFamily: 'monospace',
                      mb: 2,
                      wordBreak: 'break-all',
                    }}
                  >
                    {fileDetails.relativePath}
                  </Typography>

                  <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', mb: 2 }}>
                    <Chip
                      label={fileDetails.language.toUpperCase()}
                      color="primary"
                      size="small"
                    />
                    <Chip
                      label={formatFileSize(fileDetails.size)}
                      variant="outlined"
                      size="small"
                    />
                    <Chip
                      label={`${fileDetails.lineCount} lines`}
                      variant="outlined"
                      size="small"
                    />
                    <Chip
                      label={`${fileDetails.chunkCount} chunks`}
                      variant="outlined"
                      size="small"
                    />
                  </Box>

                  <Divider sx={{ my: 1 }} />

                  <Typography variant="caption" color="text.secondary">
                    Indexed: {formatDate(fileDetails.indexedAt)}
                  </Typography>
                </CardContent>
              </Card>

              {/* Chunks Section */}
              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: 600 }}>
                  Chunks ({chunks.length})
                </Typography>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                  File has been split into {chunks.length} chunks for indexing
                </Typography>
              </Box>

              {/* Chunk List */}
              <ChunkList
                chunks={chunks}
                fileId={fileId || ''}
                language={fileDetails.language}
              />
            </>
          ) : (
            <Typography variant="body2" color="text.secondary">
              No file selected
            </Typography>
          )}
        </Box>
      </Box>
    </Drawer>
  );
};
