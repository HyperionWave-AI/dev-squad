import { useState } from 'react';
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
import { Dashboard, Psychology, Refresh } from '@mui/icons-material';
import { theme } from './theme';
import { KanbanBoard } from './components/KanbanBoard';
import { KnowledgeBrowser } from './components/KnowledgeBrowser';

type View = 'dashboard' | 'knowledge';

function App() {
  const [currentView, setCurrentView] = useState<View>('dashboard');
  const [refreshKey, setRefreshKey] = useState(0);

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
                variant={currentView === 'dashboard' ? 'contained' : 'outlined'}
                startIcon={<Dashboard />}
                onClick={() => setCurrentView('dashboard')}
                sx={{
                  textTransform: 'none',
                  fontWeight: 500,
                }}
              >
                Dashboard
              </Button>
              <Button
                variant={currentView === 'knowledge' ? 'contained' : 'outlined'}
                startIcon={<Psychology />}
                onClick={() => setCurrentView('knowledge')}
                sx={{
                  textTransform: 'none',
                  fontWeight: 500,
                }}
              >
                Knowledge
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
            {currentView === 'dashboard' && <KanbanBoard key={refreshKey} />}
            {currentView === 'knowledge' && <KnowledgeBrowser key={refreshKey} />}
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