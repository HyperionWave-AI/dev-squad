import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  IconButton,
  CircularProgress,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  DialogContentText,
  Chip,
  Snackbar,
  Alert,
  Tooltip,
} from '@mui/material';
import {
  Add as AddIcon,
  Delete as DeleteIcon,
  Refresh as RefreshIcon,
  CloudQueue as ServerIcon,
} from '@mui/icons-material';
import { mcpServerService, type MCPServer } from '../services/mcpServerService';
import { AddMCPServerDialog } from '../components/AddMCPServerDialog';

export const MCPServersPage: React.FC = () => {
  // State
  const [servers, setServers] = useState<MCPServer[]>([]);
  const [loading, setLoading] = useState(true);

  // Dialog state
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedServer, setSelectedServer] = useState<MCPServer | null>(null);
  const [refreshingServer, setRefreshingServer] = useState<string | null>(null);

  // Snackbar state
  const [snackbar, setSnackbar] = useState<{
    open: boolean;
    message: string;
    severity: 'success' | 'error';
  }>({
    open: false,
    message: '',
    severity: 'success',
  });

  // Load servers
  const loadServers = async () => {
    setLoading(true);
    try {
      const response = await mcpServerService.listMCPServers();
      setServers(response.servers);
    } catch (error) {
      setSnackbar({
        open: true,
        message: error instanceof Error ? error.message : 'Failed to load MCP servers',
        severity: 'error',
      });
    } finally {
      setLoading(false);
    }
  };

  // Load servers on mount
  useEffect(() => {
    loadServers();
  }, []);

  // Handle add server success
  const handleAddSuccess = () => {
    setSnackbar({
      open: true,
      message: 'MCP server added successfully',
      severity: 'success',
    });
    loadServers();
  };

  // Handle add server error
  const handleAddError = (message: string) => {
    setSnackbar({
      open: true,
      message,
      severity: 'error',
    });
  };

  // Handle delete server click
  const handleDeleteClick = (server: MCPServer) => {
    setSelectedServer(server);
    setDeleteDialogOpen(true);
  };

  // Handle delete server confirm
  const handleDeleteConfirm = async () => {
    if (!selectedServer) return;

    try {
      await mcpServerService.removeMCPServer(selectedServer.serverName);
      setSnackbar({
        open: true,
        message: `MCP server '${selectedServer.serverName}' removed successfully`,
        severity: 'success',
      });
      setDeleteDialogOpen(false);
      setSelectedServer(null);
      loadServers();
    } catch (error) {
      setSnackbar({
        open: true,
        message: error instanceof Error ? error.message : 'Failed to remove MCP server',
        severity: 'error',
      });
    }
  };

  // Handle rediscover server
  const handleRediscover = async (serverName: string) => {
    setRefreshingServer(serverName);
    try {
      await mcpServerService.rediscoverMCPServer(serverName);
      setSnackbar({
        open: true,
        message: `Tools from '${serverName}' rediscovered successfully`,
        severity: 'success',
      });
      loadServers();
    } catch (error) {
      setSnackbar({
        open: true,
        message: error instanceof Error ? error.message : 'Failed to rediscover server tools',
        severity: 'error',
      });
    } finally {
      setRefreshingServer(null);
    }
  };

  // Format date
  const formatDate = (dateString: string): string => {
    try {
      return new Date(dateString).toLocaleString();
    } catch {
      return dateString;
    }
  };

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Box>
          <Typography variant="h4" gutterBottom>
            MCP Servers
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Manage external MCP servers and their tools
          </Typography>
        </Box>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => setAddDialogOpen(true)}
        >
          Add Server
        </Button>
      </Box>

      {/* Server count */}
      {!loading && (
        <Box sx={{ mb: 2 }}>
          <Chip
            icon={<ServerIcon />}
            label={`${servers.length} server${servers.length !== 1 ? 's' : ''}`}
            color="primary"
            variant="outlined"
          />
        </Box>
      )}

      {/* Servers table */}
      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Server Name</TableCell>
              <TableCell>Server URL</TableCell>
              <TableCell>Description</TableCell>
              <TableCell align="center">Tools</TableCell>
              <TableCell>Created</TableCell>
              <TableCell>Updated</TableCell>
              <TableCell align="center">Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={7} align="center" sx={{ py: 4 }}>
                  <CircularProgress />
                </TableCell>
              </TableRow>
            ) : servers.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} align="center" sx={{ py: 4 }}>
                  <Typography variant="body2" color="text.secondary">
                    No MCP servers registered. Click "Add Server" to get started.
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              servers.map((server) => (
                <TableRow key={server.serverName} hover>
                  <TableCell>
                    <Typography variant="body2" fontWeight="medium">
                      {server.serverName}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2" sx={{ fontFamily: 'monospace', fontSize: '0.875rem' }}>
                      {server.serverUrl}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2" color="text.secondary">
                      {server.description || '-'}
                    </Typography>
                  </TableCell>
                  <TableCell align="center">
                    <Chip
                      label={server.toolCount}
                      size="small"
                      color={server.toolCount > 0 ? 'success' : 'default'}
                    />
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2" color="text.secondary">
                      {formatDate(server.createdAt)}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2" color="text.secondary">
                      {formatDate(server.updatedAt)}
                    </Typography>
                  </TableCell>
                  <TableCell align="center">
                    <Box sx={{ display: 'flex', gap: 1, justifyContent: 'center' }}>
                      <Tooltip title="Rediscover tools from this server">
                        <IconButton
                          size="small"
                          onClick={() => handleRediscover(server.serverName)}
                          disabled={refreshingServer === server.serverName}
                          color="primary"
                        >
                          {refreshingServer === server.serverName ? (
                            <CircularProgress size={20} />
                          ) : (
                            <RefreshIcon />
                          )}
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="Remove this server">
                        <IconButton
                          size="small"
                          onClick={() => handleDeleteClick(server)}
                          color="error"
                        >
                          <DeleteIcon />
                        </IconButton>
                      </Tooltip>
                    </Box>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>

      {/* Add Server Dialog */}
      <AddMCPServerDialog
        open={addDialogOpen}
        onClose={() => setAddDialogOpen(false)}
        onSuccess={handleAddSuccess}
        onError={handleAddError}
      />

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteDialogOpen} onClose={() => setDeleteDialogOpen(false)}>
        <DialogTitle>Confirm Delete</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Are you sure you want to remove the MCP server <strong>{selectedServer?.serverName}</strong>?
            <br />
            <br />
            This will also remove all {selectedServer?.toolCount || 0} tool(s) associated with this server.
            <br />
            <br />
            This action cannot be undone.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteDialogOpen(false)}>Cancel</Button>
          <Button onClick={handleDeleteConfirm} color="error" variant="contained">
            Delete
          </Button>
        </DialogActions>
      </Dialog>

      {/* Snackbar */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={6000}
        onClose={() => setSnackbar({ ...snackbar, open: false })}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
      >
        <Alert
          onClose={() => setSnackbar({ ...snackbar, open: false })}
          severity={snackbar.severity}
          variant="filled"
        >
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Box>
  );
};
