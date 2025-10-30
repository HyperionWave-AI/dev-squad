import { useState, useEffect } from 'react';
import { Routes, Route, Navigate, useLocation, useNavigate } from 'react-router-dom';
import {
  ThemeProvider,
  CssBaseline,
  AppBar,
  Toolbar,
  Typography,
  Button,
  Box,
  IconButton,
  Drawer,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Divider,
  Chip,
} from '@mui/material';
import {
  Dashboard,
  Psychology,
  Refresh,
  Code,
  Chat,
  Build,
  Settings,
  SmartToy,
  Menu as MenuIcon,
  Close as CloseIcon,
  LightMode,
  DarkMode,
  Hub
} from '@mui/icons-material';
import { getTheme, getPreferredTheme, setThemePreference } from './theme';
import { KanbanBoard } from './components/KanbanBoard';
import { KnowledgeBrowser } from './components/KnowledgeBrowser';
import { CodeSearchPage } from './pages/CodeSearchPage';
import { CodeChatPage } from './pages/CodeChatPage';
import { HTTPToolsPage } from './pages/HTTPToolsPage';
import { SettingsPage } from './pages/SettingsPage';
import { SubagentsPage } from './pages/SubagentsPage';
import { MCPServersPage } from './pages/MCPServersPage';
import { ConversationModeProvider } from './contexts/ConversationModeContext';

function App() {
  const [refreshKey, setRefreshKey] = useState(0);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [themeMode, setThemeMode] = useState<'light' | 'dark'>('light');
  const location = useLocation();
  const navigate = useNavigate();
  // Enhanced responsive breakpoints with better mobile-first approach
  // const muiTheme = useTheme();
  // const isMobile = useMediaQuery(muiTheme.breakpoints.down('sm')); // <600px
  // const isTablet = useMediaQuery(muiTheme.breakpoints.between('sm', 'md')); // 600px - 900px

  // Initialize theme mode on component mount
  useEffect(() => {
    const preferredTheme = getPreferredTheme();
    setThemeMode(preferredTheme);
  }, []);

  // Toggle theme mode
  const toggleTheme = () => {
    const newMode = themeMode === 'light' ? 'dark' : 'light';
    setThemeMode(newMode);
    setThemePreference(newMode);
  };

  const handleRefresh = () => {
    setRefreshKey((prev) => prev + 1);
  };

  const handleMobileMenuToggle = () => {
    setMobileMenuOpen(!mobileMenuOpen);
  };

  const handleNavigate = (path: string) => {
    navigate(path);
    setMobileMenuOpen(false); // Close mobile menu after navigation
  };

  // Enhanced navigation items with better organization
  const navigationItems = [
    { path: '/chat', label: 'Chat', icon: <Chat />, priority: 'high' },
    { path: '/tasks', label: 'Tasks', icon: <Dashboard />, priority: 'high' },
    { path: '/knowledge', label: 'Knowledge', icon: <Psychology />, priority: 'medium' },
    { path: '/code', label: 'Code', icon: <Code />, priority: 'medium' },
    { path: '/mcp-servers', label: 'MCP Servers', icon: <Hub />, priority: 'medium' },
    { path: '/tools', label: 'Tools', icon: <Build />, priority: 'low' },
    { path: '/subagents', label: 'Subagents', icon: <SmartToy />, priority: 'low' },
    { path: '/settings', label: 'Settings', icon: <Settings />, priority: 'low' },
  ];

  // Get current page info for better UX
  const currentPage = navigationItems.find(item => item.path === location.pathname);

  // Enhanced Mobile Navigation Drawer with better accessibility
  const MobileDrawer = () => (
    <Drawer
      anchor="left"
      open={mobileMenuOpen}
      onClose={() => setMobileMenuOpen(false)}
      sx={{
        '& .MuiDrawer-paper': {
          width: { xs: '85vw', sm: 320 },
          maxWidth: 400,
          boxSizing: 'border-box',
          backgroundColor: 'background.paper',
          borderRight: '1px solid',
          borderColor: 'divider',
        },
      }}
      // Enhanced accessibility
      ModalProps={{
        keepMounted: true, // Better mobile performance
      }}
    >
      <Box sx={{ 
        p: 2, 
        display: 'flex', 
        alignItems: 'center', 
        justifyContent: 'space-between',
        minHeight: { xs: 56, sm: 64 }, // Match header height
        borderBottom: '1px solid',
        borderColor: 'divider',
        backgroundColor: 'background.paper',
      }}>
        <Typography variant="h6" sx={{ 
          fontWeight: 700,
          background: 'linear-gradient(135deg, #2563eb 0%, #9333ea 100%)',
          backgroundClip: 'text',
          WebkitBackgroundClip: 'text',
          WebkitTextFillColor: 'transparent',
          fontSize: { xs: '1.1rem', sm: '1.25rem' },
        }}>
          🚀 Hyperion
        </Typography>
        <IconButton 
          onClick={() => setMobileMenuOpen(false)}
          aria-label="Close navigation menu"
          size="small"
          sx={{
            '&:hover': {
              backgroundColor: 'action.hover',
            },
          }}
        >
          <CloseIcon />
        </IconButton>
      </Box>
      
      <List sx={{ pt: 1, flex: 1 }}>
        {navigationItems.map((item) => (
          <ListItem key={item.path} disablePadding>
            <ListItemButton
              selected={location.pathname === item.path}
              onClick={() => handleNavigate(item.path)}
              sx={{
                minHeight: 56, // Touch-friendly height
                px: 3,
                py: 1.5,
                '&.Mui-selected': {
                  backgroundColor: 'primary.main',
                  color: 'primary.contrastText',
                  '& .MuiListItemIcon-root': {
                    color: 'primary.contrastText',
                  },
                  '&:hover': {
                    backgroundColor: 'primary.dark',
                  },
                },
                '&:hover': {
                  backgroundColor: location.pathname === item.path 
                    ? 'primary.dark' 
                    : 'action.hover',
                },
                borderRadius: 1,
                mx: 1,
                mb: 0.5,
              }}
            >
              <ListItemIcon sx={{ minWidth: 48 }}>
                {item.icon}
              </ListItemIcon>
              <ListItemText 
                primary={item.label}
                primaryTypographyProps={{
                  fontSize: '1rem',
                  fontWeight: location.pathname === item.path ? 600 : 400,
                }}
              />
            </ListItemButton>
          </ListItem>
        ))}
      </List>
      
      <Divider sx={{ my: 1 }} />
      
      <List>
        <ListItem disablePadding>
          <ListItemButton 
            onClick={handleRefresh}
            sx={{ 
              minHeight: 56,
              px: 3,
              py: 1.5,
              '&:hover': {
                backgroundColor: 'action.hover',
              },
              borderRadius: 1,
              mx: 1,
              mb: 1,
            }}
          >
            <ListItemIcon sx={{ minWidth: 48 }}>
              <Refresh />
            </ListItemIcon>
            <ListItemText 
              primary="Refresh"
              primaryTypographyProps={{
                fontSize: '1rem',
              }}
            />
          </ListItemButton>
        </ListItem>
        
        {/* Theme Toggle in Mobile Menu */}
        <ListItem disablePadding>
          <ListItemButton 
            onClick={toggleTheme}
            sx={{ 
              minHeight: 56,
              px: 3,
              py: 1.5,
              '&:hover': {
                backgroundColor: 'action.hover',
              },
              borderRadius: 1,
              mx: 1,
              mb: 1,
            }}
          >
            <ListItemIcon sx={{ minWidth: 48 }}>
              {themeMode === 'light' ? <DarkMode /> : <LightMode />}
            </ListItemIcon>
            <ListItemText 
              primary={themeMode === 'light' ? 'Dark Mode' : 'Light Mode'}
              primaryTypographyProps={{
                fontSize: '1rem',
              }}
            />
          </ListItemButton>
        </ListItem>
      </List>
    </Drawer>
  );

  return (
    <ThemeProvider theme={getTheme(themeMode)}>
      <ConversationModeProvider>
        <CssBaseline />
        <Box sx={{
          display: 'flex',
          flexDirection: 'column',
          minHeight: '100vh',
          width: '100vw',
          overflow: 'hidden',
          backgroundColor: 'background.default',
        }}>
          {/* Enhanced AppBar Header - Mobile-First Responsive Design */}
          <AppBar
            position="sticky"
            elevation={0}
            sx={{
              backgroundColor: 'background.paper',
              borderBottom: '1px solid',
              borderColor: 'divider',
              color: 'text.primary',
              zIndex: (theme) => theme.zIndex.drawer + 1,
              minHeight: { xs: 56, sm: 64 },
            }}
          >
            <Toolbar
              sx={{
                minHeight: { xs: '56px !important', sm: '64px !important' },
                px: { xs: 2, sm: 3 },
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
              }}
            >
              {/* Left section - Mobile menu + Logo */}
              <Box sx={{ 
                display: 'flex', 
                alignItems: 'center', 
                gap: { xs: 1, sm: 2 },
                flex: { xs: 1, sm: 'none' },
              }}>
                {/* Mobile Menu Button */}
                <IconButton
                  edge="start"
                  color="inherit"
                  aria-label="Open navigation menu"
                  onClick={handleMobileMenuToggle}
                  sx={{
                    display: { xs: 'flex', md: 'none' },
                    mr: { xs: 1, sm: 2 },
                    '&:hover': {
                      backgroundColor: 'action.hover',
                    },
                  }}
                >
                  <MenuIcon />
                </IconButton>

                {/* Logo/Brand */}
                <Typography
                  variant="h6"
                  component="div"
                  sx={{
                    fontWeight: 700,
                    background: 'linear-gradient(135deg, #2563eb 0%, #9333ea 100%)',
                    backgroundClip: 'text',
                    WebkitBackgroundClip: 'text',
                    WebkitTextFillColor: 'transparent',
                    fontSize: { xs: '1.1rem', sm: '1.25rem', md: '1.5rem' },
                    cursor: 'pointer',
                    userSelect: 'none',
                    display: { xs: 'block', sm: 'block' },
                  }}
                  onClick={() => handleNavigate('/tasks')}
                >
                  🚀 Hyperion
                </Typography>
              </Box>

              {/* Center section - Desktop Navigation (hidden on mobile/tablet) */}
              <Box sx={{ 
                display: { xs: 'none', md: 'flex' }, 
                gap: 1,
                flex: 1,
                justifyContent: 'center',
                maxWidth: 600,
              }}>
                {navigationItems.filter(item => item.priority === 'high' || item.priority === 'medium').map((item) => (
                  <Button
                    key={item.path}
                    onClick={() => handleNavigate(item.path)}
                    startIcon={item.icon}
                    variant={location.pathname === item.path ? 'contained' : 'text'}
                    sx={{
                      textTransform: 'none',
                      fontWeight: location.pathname === item.path ? 600 : 400,
                      px: 2,
                      py: 1,
                      borderRadius: 2,
                      minWidth: 'auto',
                      '&:hover': {
                        backgroundColor: location.pathname === item.path 
                          ? 'primary.dark' 
                          : 'action.hover',
                      },
                    }}
                  >
                    {item.label}
                  </Button>
                ))}
              </Box>

              {/* Right section - Actions */}
              <Box sx={{ 
                display: 'flex', 
                alignItems: 'center', 
                gap: 1,
                flex: { xs: 'none', sm: 'none' },
              }}>
                {/* Current Page Indicator (Mobile/Tablet only) */}
                {currentPage && (
                  <Chip
                    icon={currentPage.icon}
                    label={currentPage.label}
                    size="small"
                    sx={{
                      display: { xs: 'flex', md: 'none' },
                      backgroundColor: 'primary.main',
                      color: 'primary.contrastText',
                      fontWeight: 600,
                      '& .MuiChip-icon': {
                        color: 'primary.contrastText',
                      },
                    }}
                  />
                )}

                {/* Theme Toggle (Desktop only) */}
                <IconButton
                  onClick={toggleTheme}
                  color="inherit"
                  aria-label={`Switch to ${themeMode === 'light' ? 'dark' : 'light'} mode`}
                  sx={{
                    display: { xs: 'none', md: 'flex' },
                    '&:hover': {
                      backgroundColor: 'action.hover',
                    },
                  }}
                >
                  {themeMode === 'light' ? <DarkMode /> : <LightMode />}
                </IconButton>

                {/* Refresh Button (Desktop only) */}
                <IconButton
                  onClick={handleRefresh}
                  color="inherit"
                  aria-label="Refresh page"
                  sx={{
                    display: { xs: 'none', md: 'flex' },
                    '&:hover': {
                      backgroundColor: 'action.hover',
                    },
                  }}
                >
                  <Refresh />
                </IconButton>
              </Box>
            </Toolbar>
          </AppBar>

          {/* Mobile Navigation Drawer */}
          <MobileDrawer />

          {/* Main Content Area with Enhanced Responsive Layout */}
          <Box
            component="main"
            sx={{
              flex: 1,
              display: 'flex',
              flexDirection: 'column',
              overflow: 'hidden',
              backgroundColor: 'background.default',
              position: 'relative',
            }}
          >
            <Routes>
              <Route path="/" element={<Navigate to="/tasks" replace />} />
              <Route path="/tasks" element={<KanbanBoard key={refreshKey} />} />
              <Route path="/knowledge" element={<KnowledgeBrowser key={refreshKey} />} />
              <Route path="/code" element={<CodeSearchPage />} />
              <Route path="/chat" element={<CodeChatPage />} />
              <Route path="/mcp-servers" element={<MCPServersPage />} />
              <Route path="/tools" element={<HTTPToolsPage />} />
              <Route path="/subagents" element={<SubagentsPage />} />
              <Route path="/settings" element={<SettingsPage />} />
            </Routes>
          </Box>
        </Box>
      </ConversationModeProvider>
    </ThemeProvider>
  );
}

export default App;