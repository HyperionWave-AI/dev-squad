import { useState } from 'react';
import { Routes, Route, Navigate, useLocation, useNavigate } from 'react-router-dom';
import {
  ThemeProvider,
  CssBaseline,
  AppBar,
  Toolbar,
  Typography,
  Button,
  Container,
  Box,
  IconButton,
} from '@mui/material';
import { Dashboard, Psychology, Refresh, Code, Chat, Build, Settings, SmartToy } from '@mui/icons-material';
import { theme } from './theme';
import { KanbanBoard } from './components/KanbanBoard';
import { KnowledgeBrowser } from './components/KnowledgeBrowser';
import { CodeSearchPage } from './pages/CodeSearchPage';
import { CodeChatPage } from './pages/CodeChatPage';
import { HTTPToolsPage } from './pages/HTTPToolsPage';
import { SettingsPage } from './pages/SettingsPage';
import { SubagentsPage } from './pages/SubagentsPage';

function App() {
  const [refreshKey, setRefreshKey] = useState(0);
  const location = useLocation();
  const navigate = useNavigate();

  const handleRefresh = () => {
    setRefreshKey((prev) => prev + 1);
  };

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Box sx={{ display: 'flex', flexDirection: 'column', minHeight: '100vh' }}>
        {/* AppBar Header */}
        <AppBar
          position="sticky"
          elevation={1}
          sx={{
            backgroundColor: 'white',
            color: 'text.primary',
            borderBottom: '1px solid',
            borderColor: 'divider',
          }}
        >
          <Toolbar sx={{ gap: 2 }}>
            {/* Logo and Title */}
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexGrow: 1 }}>
              <Typography
                variant="h1"
                sx={{
                  fontSize: '1.5rem',
                  fontWeight: 700,
                  background: 'linear-gradient(135deg, #2563eb 0%, #9333ea 100%)',
                  backgroundClip: 'text',
                  WebkitBackgroundClip: 'text',
                  WebkitTextFillColor: 'transparent',
                }}
              >
                🚀 Hyperion Coordinator
              </Typography>
              <Typography
                variant="body2"
                sx={{
                  color: 'text.secondary',
                  display: { xs: 'none', sm: 'block' },
                }}
              >
                • Task & Knowledge Management
              </Typography>
            </Box>

            {/* Navigation Buttons */}
            <Box sx={{ display: 'flex', gap: 1 }}>
              <Button
                variant={location.pathname === '/chat' ? 'contained' : 'outlined'}
                startIcon={<Chat />}
                onClick={() => navigate('/chat')}
                sx={{
                  textTransform: 'none',
                  fontWeight: 500,
                }}
              >
                Chat
              </Button>
              <Button
                variant={location.pathname === '/tasks' ? 'contained' : 'outlined'}
                startIcon={<Dashboard />}
                onClick={() => navigate('/tasks')}
                sx={{
                  textTransform: 'none',
                  fontWeight: 500,
                }}
              >
                Tasks
              </Button>
              <Button
                variant={location.pathname === '/knowledge' ? 'contained' : 'outlined'}
                startIcon={<Psychology />}
                onClick={() => navigate('/knowledge')}
                sx={{
                  textTransform: 'none',
                  fontWeight: 500,
                }}
              >
                Knowledge
              </Button>
              <Button
                variant={location.pathname === '/code' ? 'contained' : 'outlined'}
                startIcon={<Code />}
                onClick={() => navigate('/code')}
                sx={{
                  textTransform: 'none',
                  fontWeight: 500,
                }}
              >
                Code
              </Button>
              <Button
                variant={location.pathname === '/tools' ? 'contained' : 'outlined'}
                startIcon={<Build />}
                onClick={() => navigate('/tools')}
                sx={{
                  textTransform: 'none',
                  fontWeight: 500,
                }}
              >
                Tools
              </Button>
              <Button
                variant={location.pathname === '/subagents' ? 'contained' : 'outlined'}
                startIcon={<SmartToy />}
                onClick={() => navigate('/subagents')}
                sx={{
                  textTransform: 'none',
                  fontWeight: 500,
                }}
              >
                Subagents
              </Button>
              <Button
                variant={location.pathname === '/settings' ? 'contained' : 'outlined'}
                startIcon={<Settings />}
                onClick={() => navigate('/settings')}
                sx={{
                  textTransform: 'none',
                  fontWeight: 500,
                }}
              >
                Settings
              </Button>
              <IconButton
                onClick={handleRefresh}
                color="primary"
                sx={{
                  ml: 1,
                }}
              >
                <Refresh />
              </IconButton>
            </Box>
          </Toolbar>
        </AppBar>

        {/* Main Content */}
        <Box
          component="main"
          sx={{
            flexGrow: 1,
            backgroundColor: 'background.default',
            py: 3,
          }}
        >
          <Container maxWidth="xl">
            <Routes>
              <Route path="/" element={<Navigate to="/tasks" replace />} />
              <Route path="/chat" element={<CodeChatPage key={refreshKey} />} />
              <Route path="/tasks" element={<KanbanBoard key={refreshKey} />} />
              <Route path="/knowledge" element={<KnowledgeBrowser key={refreshKey} />} />
              <Route path="/code" element={<CodeSearchPage key={refreshKey} />} />
              <Route path="/tools" element={<HTTPToolsPage key={refreshKey} />} />
              <Route path="/subagents" element={<SubagentsPage key={refreshKey} />} />
              <Route path="/settings" element={<SettingsPage key={refreshKey} />} />
            </Routes>
          </Container>
        </Box>

        {/* Footer */}
        <Box
          component="footer"
          sx={{
            py: 2,
            px: 2,
            mt: 'auto',
            backgroundColor: 'white',
            borderTop: '1px solid',
            borderColor: 'divider',
          }}
        >
          <Container maxWidth="xl">
            <Typography
              variant="body2"
              color="text.secondary"
              align="center"
            >
              Hyperion AI Platform • Coordinator MCP • Local Dev Environment
            </Typography>
          </Container>
        </Box>
      </Box>
    </ThemeProvider>
  );
}

export default App;