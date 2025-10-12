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
  TablePagination,
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
} from '@mui/material';
import {
  Add as AddIcon,
  Visibility as ViewIcon,
  Delete as DeleteIcon,
  Refresh as RefreshIcon,
} from '@mui/icons-material';
import { httpToolsService } from '../services/httpToolsService';
import type { HTTPToolDefinition } from '../services/httpToolsService';
import { AddHTTPToolDialog } from '../components/AddHTTPToolDialog';

export const HTTPToolsPage: React.FC = () => {
  // State
  const [tools, setTools] = useState<HTTPToolDefinition[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(20);
  const [total, setTotal] = useState(0);

  // Dialog state
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [viewDialogOpen, setViewDialogOpen] = useState(false);
  const [selectedTool, setSelectedTool] = useState<HTTPToolDefinition | null>(null);

  // Snackbar state
  const [snackbar, setSnackbar] = useState<{ open: boolean; message: string; severity: 'success' | 'error' }>({
    open: false,
    message: '',
    severity: 'success',
  });

  // Load tools
  const loadTools = async () => {
    setLoading(true);
    try {
      const response = await httpToolsService.listHTTPTools(page + 1, rowsPerPage);
      setTools(response.tools);
      setTotal(response.total);
    } catch (error) {
      setSnackbar({
        open: true,
        message: error instanceof Error ? error.message : 'Failed to load HTTP tools',
        severity: 'error',
      });
    } finally {
      setLoading(false);
    }
  };

  // Load tools on mount and when pagination changes
  useEffect(() => {
    loadTools();
  }, [page, rowsPerPage]);

  // Handle pagination
  const handleChangePage = (_event: unknown, newPage: number) => {
    setPage(newPage);
  };

  const handleChangeRowsPerPage = (event: React.ChangeEvent<HTMLInputElement>) => {
    setRowsPerPage(parseInt(event.target.value, 10));
    setPage(0);
  };

  // Handle view tool
  const handleView = (tool: HTTPToolDefinition) => {
    setSelectedTool(tool);
    setViewDialogOpen(true);
  };

  // Handle delete tool
  const handleDeleteClick = (tool: HTTPToolDefinition) => {
    setSelectedTool(tool);
    setDeleteDialogOpen(true);
  };

  const handleDeleteConfirm = async () => {
    if (!selectedTool?.id) return;

    try {
      await httpToolsService.deleteHTTPTool(selectedTool.id);
      setSnackbar({
        open: true,
        message: 'HTTP tool deleted successfully',
        severity: 'success',
      });
      setDeleteDialogOpen(false);
      setSelectedTool(null);
      loadTools();
    } catch (error) {
      setSnackbar({
        open: true,
        message: error instanceof Error ? error.message : 'Failed to delete HTTP tool',
        severity: 'error',
      });
    }
  };

  // Handle add tool success
  const handleAddSuccess = () => {
    loadTools();
  };

  // Get HTTP method color
  const getMethodColor = (method: string): 'default' | 'primary' | 'success' | 'warning' | 'error' => {
    switch (method) {
      case 'GET': return 'primary';
      case 'POST': return 'success';
      case 'PUT': return 'warning';
      case 'DELETE': return 'error';
      case 'PATCH': return 'default';
      default: return 'default';
    }
  };

  return (
    <Box>
      {/* Page Header */}
      <Box sx={{ mb: 4, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Box>
          <Typography variant="h4" sx={{ fontWeight: 700, mb: 1 }}>
            HTTP Tools
          </Typography>
          <Typography variant="body1" color="text.secondary">
            Manage external HTTP API tools for dynamic execution
          </Typography>
        </Box>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <IconButton onClick={loadTools} color="primary">
            <RefreshIcon />
          </IconButton>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => setAddDialogOpen(true)}
          >
            Add Tool
          </Button>
        </Box>
      </Box>

      {/* Tools Table */}
      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}>
          <CircularProgress />
        </Box>
      ) : tools.length === 0 ? (
        <Paper sx={{ p: 8, textAlign: 'center' }}>
          <Typography variant="h6" color="text.secondary" gutterBottom>
            No HTTP tools configured
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
            Create your first HTTP tool to start integrating external APIs
          </Typography>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => setAddDialogOpen(true)}
          >
            Add First Tool
          </Button>
        </Paper>
      ) : (
        <>
          <TableContainer component={Paper}>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell><strong>Tool Name</strong></TableCell>
                  <TableCell><strong>Description</strong></TableCell>
                  <TableCell><strong>Endpoint</strong></TableCell>
                  <TableCell><strong>Method</strong></TableCell>
                  <TableCell align="right"><strong>Actions</strong></TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {tools.map((tool) => (
                  <TableRow key={tool.id} hover>
                    <TableCell>
                      <Typography variant="body2" sx={{ fontFamily: 'monospace', fontWeight: 600 }}>
                        {tool.toolName}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" sx={{ maxWidth: 400, overflow: 'hidden', textOverflow: 'ellipsis' }}>
                        {tool.description}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" sx={{ fontFamily: 'monospace', fontSize: '0.85rem', color: 'text.secondary' }}>
                        {tool.endpoint}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={tool.httpMethod}
                        color={getMethodColor(tool.httpMethod)}
                        size="small"
                      />
                    </TableCell>
                    <TableCell align="right">
                      <IconButton
                        size="small"
                        color="primary"
                        onClick={() => handleView(tool)}
                        title="View details"
                      >
                        <ViewIcon />
                      </IconButton>
                      <IconButton
                        size="small"
                        color="error"
                        onClick={() => handleDeleteClick(tool)}
                        title="Delete tool"
                      >
                        <DeleteIcon />
                      </IconButton>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>

          {/* Pagination */}
          <TablePagination
            component="div"
            count={total}
            page={page}
            onPageChange={handleChangePage}
            rowsPerPage={rowsPerPage}
            onRowsPerPageChange={handleChangeRowsPerPage}
            rowsPerPageOptions={[10, 20, 50, 100]}
          />
        </>
      )}

      {/* Add Tool Dialog */}
      <AddHTTPToolDialog
        open={addDialogOpen}
        onClose={() => setAddDialogOpen(false)}
        onSuccess={handleAddSuccess}
      />

      {/* View Tool Dialog */}
      <Dialog open={viewDialogOpen} onClose={() => setViewDialogOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>Tool Details</DialogTitle>
        <DialogContent>
          {selectedTool && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
              <Box>
                <Typography variant="subtitle2" color="text.secondary">Tool Name</Typography>
                <Typography variant="body1" sx={{ fontFamily: 'monospace' }}>{selectedTool.toolName}</Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2" color="text.secondary">Description</Typography>
                <Typography variant="body1">{selectedTool.description}</Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2" color="text.secondary">Endpoint</Typography>
                <Typography variant="body1" sx={{ fontFamily: 'monospace', wordBreak: 'break-all' }}>
                  {selectedTool.endpoint}
                </Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2" color="text.secondary">HTTP Method</Typography>
                <Chip label={selectedTool.httpMethod} color={getMethodColor(selectedTool.httpMethod)} size="small" />
              </Box>
              {selectedTool.headers && selectedTool.headers.length > 0 && (
                <Box>
                  <Typography variant="subtitle2" color="text.secondary" gutterBottom>Headers</Typography>
                  {selectedTool.headers.map((header, index) => (
                    <Typography key={index} variant="body2" sx={{ fontFamily: 'monospace', fontSize: '0.85rem' }}>
                      {header.key}: {header.value}
                    </Typography>
                  ))}
                </Box>
              )}
              {selectedTool.parameters && selectedTool.parameters.length > 0 && (
                <Box>
                  <Typography variant="subtitle2" color="text.secondary" gutterBottom>Parameters</Typography>
                  {selectedTool.parameters.map((param, index) => (
                    <Box key={index} sx={{ mb: 1 }}>
                      <Typography variant="body2">
                        <strong>{param.name}</strong> ({param.type})
                        {param.required && <Chip label="Required" size="small" sx={{ ml: 1 }} />}
                      </Typography>
                      {param.description && (
                        <Typography variant="caption" color="text.secondary">
                          {param.description}
                        </Typography>
                      )}
                    </Box>
                  ))}
                </Box>
              )}
              {selectedTool.authType && selectedTool.authType !== 'none' && (
                <Box>
                  <Typography variant="subtitle2" color="text.secondary">Authentication</Typography>
                  <Typography variant="body1">{selectedTool.authType}</Typography>
                </Box>
              )}
              {selectedTool.createdAt && (
                <Box>
                  <Typography variant="subtitle2" color="text.secondary">Created</Typography>
                  <Typography variant="body2">{new Date(selectedTool.createdAt).toLocaleString()}</Typography>
                </Box>
              )}
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setViewDialogOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteDialogOpen} onClose={() => setDeleteDialogOpen(false)}>
        <DialogTitle>Confirm Delete</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Are you sure you want to delete the tool <strong>{selectedTool?.toolName}</strong>?
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
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert severity={snackbar.severity} onClose={() => setSnackbar({ ...snackbar, open: false })}>
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Box>
  );
};
