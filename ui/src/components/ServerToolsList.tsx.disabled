import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  List,
  ListItem,
  ListItemText,
  CircularProgress,
  Alert,
  Chip,
  Divider,
  Tabs,
  Tab,
  Link,
} from '@mui/material';
import {
  Build as ToolIcon,
  Storage as ResourceIcon,
  Chat as PromptIcon,
} from '@mui/icons-material';
import {
  mcpServerService,
  type MCPTool,
  type MCPResource,
  type MCPPrompt,
} from '../services/mcpServerService';

interface ServerToolsListProps {
  serverName: string;
}

export const ServerToolsList: React.FC<ServerToolsListProps> = ({ serverName }) => {
  const [tools, setTools] = useState<MCPTool[]>([]);
  const [resources, setResources] = useState<MCPResource[]>([]);
  const [prompts, setPrompts] = useState<MCPPrompt[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>('');
  const [currentTab, setCurrentTab] = useState(0);

  useEffect(() => {
    const loadServerDetails = async () => {
      setLoading(true);
      setError('');
      try {
        const response = await mcpServerService.getServerDetails(serverName);
        setTools(response.server.tools || []);
        setResources(response.server.resources || []);
        setPrompts(response.server.prompts || []);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load server details');
      } finally {
        setLoading(false);
      }
    };

    loadServerDetails();
  }, [serverName]);

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 3 }}>
        <CircularProgress size={24} />
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="error" sx={{ my: 2 }}>
        {error}
      </Alert>
    );
  }

  const hasContent = tools.length > 0 || resources.length > 0 || prompts.length > 0;

  if (!hasContent) {
    return (
      <Box sx={{ py: 3, textAlign: 'center' }}>
        <Typography variant="body2" color="text.secondary">
          No tools, resources, or prompts discovered from this server
        </Typography>
      </Box>
    );
  }

  const renderToolsList = () => (
    <List dense>
      {tools.map((tool, index) => (
        <React.Fragment key={tool.name}>
          {index > 0 && <Divider component="li" />}
          <ListItem sx={{ alignItems: 'flex-start', px: 0 }}>
            <ListItemText
              primary={
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <Typography
                    variant="body2"
                    component="span"
                    sx={{ fontFamily: 'monospace', fontWeight: 'medium' }}
                  >
                    {tool.name}
                  </Typography>
                  {tool.inputSchema && (
                    <Chip
                      label={`${Object.keys(tool.inputSchema.properties || {}).length} params`}
                      size="small"
                      variant="outlined"
                      sx={{ height: 20, fontSize: '0.7rem' }}
                    />
                  )}
                </Box>
              }
              secondary={
                <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
                  {tool.description || 'No description available'}
                </Typography>
              }
            />
          </ListItem>
        </React.Fragment>
      ))}
    </List>
  );

  const renderResourcesList = () => (
    <List dense>
      {resources.map((resource, index) => (
        <React.Fragment key={resource.uri}>
          {index > 0 && <Divider component="li" />}
          <ListItem sx={{ alignItems: 'flex-start', px: 0 }}>
            <ListItemText
              primary={
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
                  <Typography
                    variant="body2"
                    component="span"
                    sx={{ fontFamily: 'monospace', fontWeight: 'medium' }}
                  >
                    {resource.name}
                  </Typography>
                  {resource.mimeType && (
                    <Chip
                      label={resource.mimeType}
                      size="small"
                      variant="outlined"
                      sx={{ height: 20, fontSize: '0.7rem' }}
                    />
                  )}
                </Box>
              }
              secondary={
                <Box sx={{ mt: 0.5 }}>
                  <Typography variant="body2" color="text.secondary">
                    {resource.description || 'No description available'}
                  </Typography>
                  <Link
                    href={resource.uri}
                    target={resource.uri.startsWith('http') ? '_blank' : undefined}
                    rel={resource.uri.startsWith('http') ? 'noopener noreferrer' : undefined}
                    variant="body2"
                    sx={{
                      fontFamily: 'monospace',
                      fontSize: '0.75rem',
                      display: 'block',
                      mt: 0.5,
                      wordBreak: 'break-all',
                    }}
                  >
                    {resource.uri}
                  </Link>
                </Box>
              }
            />
          </ListItem>
        </React.Fragment>
      ))}
    </List>
  );

  const renderPromptsList = () => (
    <List dense>
      {prompts.map((prompt, index) => (
        <React.Fragment key={prompt.name}>
          {index > 0 && <Divider component="li" />}
          <ListItem sx={{ alignItems: 'flex-start', px: 0 }}>
            <ListItemText
              primary={
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <Typography
                    variant="body2"
                    component="span"
                    sx={{ fontFamily: 'monospace', fontWeight: 'medium' }}
                  >
                    {prompt.name}
                  </Typography>
                  {prompt.arguments && prompt.arguments.length > 0 && (
                    <Chip
                      label={`${prompt.arguments.length} arg${
                        prompt.arguments.length !== 1 ? 's' : ''
                      }`}
                      size="small"
                      variant="outlined"
                      sx={{ height: 20, fontSize: '0.7rem' }}
                    />
                  )}
                </Box>
              }
              secondary={
                <Box sx={{ mt: 0.5 }}>
                  <Typography variant="body2" color="text.secondary">
                    {prompt.description || 'No description available'}
                  </Typography>
                  {prompt.arguments && prompt.arguments.length > 0 && (
                    <Box sx={{ mt: 1 }}>
                      <Typography
                        variant="caption"
                        color="text.secondary"
                        sx={{ fontWeight: 'medium' }}
                      >
                        Arguments:
                      </Typography>
                      <List dense disablePadding sx={{ ml: 2 }}>
                        {prompt.arguments.map((arg) => (
                          <ListItem key={arg.name} sx={{ py: 0.5, px: 0 }}>
                            <Typography
                              variant="caption"
                              sx={{ fontFamily: 'monospace' }}
                            >
                              {arg.name}
                              {arg.required && (
                                <Chip
                                  label="required"
                                  size="small"
                                  color="warning"
                                  sx={{ ml: 1, height: 16, fontSize: '0.65rem' }}
                                />
                              )}
                              {arg.description && (
                                <Typography
                                  variant="caption"
                                  color="text.secondary"
                                  sx={{ ml: 1 }}
                                >
                                  - {arg.description}
                                </Typography>
                              )}
                            </Typography>
                          </ListItem>
                        ))}
                      </List>
                    </Box>
                  )}
                </Box>
              }
            />
          </ListItem>
        </React.Fragment>
      ))}
    </List>
  );

  return (
    <Box sx={{ py: 2 }}>
      <Tabs
        value={currentTab}
        onChange={(_, newValue) => setCurrentTab(newValue)}
        sx={{ borderBottom: 1, borderColor: 'divider', mb: 2 }}
      >
        <Tab
          icon={<ToolIcon fontSize="small" />}
          iconPosition="start"
          label={`Tools (${tools.length})`}
          disabled={tools.length === 0}
          sx={{ textTransform: 'none', minHeight: 48 }}
        />
        <Tab
          icon={<ResourceIcon fontSize="small" />}
          iconPosition="start"
          label={`Resources (${resources.length})`}
          disabled={resources.length === 0}
          sx={{ textTransform: 'none', minHeight: 48 }}
        />
        <Tab
          icon={<PromptIcon fontSize="small" />}
          iconPosition="start"
          label={`Prompts (${prompts.length})`}
          disabled={prompts.length === 0}
          sx={{ textTransform: 'none', minHeight: 48 }}
        />
      </Tabs>

      {currentTab === 0 && tools.length > 0 && renderToolsList()}
      {currentTab === 1 && resources.length > 0 && renderResourcesList()}
      {currentTab === 2 && prompts.length > 0 && renderPromptsList()}
    </Box>
  );
};
