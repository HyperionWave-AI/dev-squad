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
  useTheme,
  useMediaQuery,
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
  Science
} from '@mui/icons-material';
import { getTheme, getPreferredTheme, setThemePreference } from './theme';
import { KanbanBoard } from './components/KanbanBoard';
import { KnowledgeBrowser } from './components/KnowledgeBrowser';
import { CodeSearchPage } from './pages/CodeSearchPage';
import { CodeChatPage } from './pages/CodeChatPage';
import { HTTPToolsPage } from './pages/HTTPToolsPage';
import { SettingsPage } from './pages/SettingsPage';
import { SubagentsPage } from './pages/SubagentsPage';
import { SubchatTestPage } from './pages/SubchatTestPage';
import { ConversationModeProvider } from './contexts/ConversationModeContext';

function App() {
  const [refreshKey, setRefreshKey] = useState(0);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [themeMode, setThemeMode] = useState<'light' | 'dark'>('light');
  const location = useLocation();
  const navigate = useNavigate();
  const muiTheme = useTheme();
  
  // Enhanced responsive breakpoints with better mobile-first approach
  const isMobile = useMediaQuery(muiTheme.breakpoints.down('sm')); // <600px
  const isTablet = useMediaQuery(muiTheme.breakpoints.between('sm', 'md')); // 600px - 900px

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
    { path: '/subchat-test', label: 'Subchat Test', icon: <Science />, priority: 'high' }, // TODO: Remove before commit
    { path: '/knowledge', label: 'Knowledge', icon: <Psychology />, priority: 'medium' },
    { path: '/code', label: 'Code', icon: <Code />, priority: 'medium' },
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
              color: 'text.primary',
              borderBottom: '1px solid',
              borderColor: 'divider',
              width: '100%',
              zIndex: (theme) => theme.zIndex.drawer + 1,
              boxShadow: themeMode === 'light'
                ? '0 1px 3px rgba(0, 0, 0, 0.1)'
                : '0 1px 3px rgba(0, 0, 0, 0.3)',
            }}
          >
            <Toolbar
              sx={{
                gap: { xs: 1, sm: 2 },
                px: { xs: 2, sm: 3, md: 4 },
                minHeight: { xs: 56, sm: 64 },
                width: '100%',
                justifyContent: 'space-between',
                alignItems: 'center',
              }}
            >
              {/* Left Section: Mobile Menu + Logo */}
              <Box sx={{
                display: 'flex',
                alignItems: 'center',
                gap: { xs: 1, sm: 2 },
                minWidth: 0,
                flex: '1 1 auto',
              }}>
                {/* Mobile Menu Button - Enhanced */}
                {isMobile && (
                  <IconButton
                    edge="start"
                    color="inherit"
                    aria-label="Open navigation menu"
                    onClick={handleMobileMenuToggle}
                    sx={{
                      mr: 1,
                      p: 1.5,
                      minWidth: 44,
                      minHeight: 44,
                      '&:hover': {
                        backgroundColor: 'action.hover',
                      },
                      '&:focus': {
                        outline: '2px solid',
                        outlineColor: 'primary.main',
                        outlineOffset: '2px',
                      },
                    }}
                  >
                    <MenuIcon />
                  </IconButton>
                )}

                {/* Logo and Title - Enhanced Responsive Design */}
                <Box sx={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: { xs: 1, sm: 2 },
                  minWidth: 0,
                  flex: '1 1 auto',
                }}>
                  <Typography
                    variant="h1"
                    component="h1"
                    sx={{
                      fontSize: {
                        xs: '1.1rem',
                        sm: '1.25rem',
                        md: '1.4rem',
                        lg: '1.5rem'
                      },
                      fontWeight: 700,
                      background: themeMode === 'light'
                        ? 'linear-gradient(135deg, #2563eb 0%, #9333ea 100%)'
                        : 'linear-gradient(135deg, #60a5fa 0%, #c084fc 100%)',
                      backgroundClip: 'text',
                      WebkitBackgroundClip: 'text',
                      WebkitTextFillColor: 'transparent',
                      whiteSpace: 'nowrap',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      lineHeight: 1.2,
                      letterSpacing: '-0.01em',
                    }}
                  >
                    {isMobile ? '🚀 Hyperion' : '🚀 Hyperion Coordinator'}
                  </Typography>

                  {/* Current Page Indicator - Enhanced */}
                  {currentPage && !isMobile && (
                    <Chip
                      icon={currentPage.icon}
                      label={currentPage.label}
                      size="small"
                      variant="outlined"
                      sx={{
                        ml: 2,
                        height: 28,
                        fontSize: '0.75rem',
                        fontWeight: 500,
                        borderColor: 'primary.main',
                        color: 'primary.main',
                        backgroundColor: themeMode === 'light' ? 'primary.50' : 'rgba(96, 165, 250, 0.1)',
                        '& .MuiChip-icon': {
                          color: 'primary.main',
                          fontSize: '1rem',
                        },
                      }}
                    />
                  )}
                </Box>
              </Box>

              {/* Desktop Navigation - Enhanced Responsive Design */}
              {!isMobile && (
                <Box sx={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: { sm: 1, md: 2 },
                  flex: '0 0 auto',
                }}>
                  {/* High Priority Navigation Items */}
                  {navigationItems
                    .filter(item => item.priority === 'high')
                    .map((item) => (
                      <Button
                        key={item.path}
                        startIcon={item.icon}
                        onClick={() => handleNavigate(item.path)}
                        variant={location.pathname === item.path ? 'contained' : 'text'}
                        size={isTablet ? 'small' : 'medium'}
                        sx={{
                          minWidth: { sm: 80, md: 100 },
                          minHeight: 40,
                          px: { sm: 1.5, md: 2 },
                          fontSize: { sm: '0.8rem', md: '0.875rem' },
                          fontWeight: 500,
                          textTransform: 'none',
                          borderRadius: 2,
                          '&:hover': {
                            backgroundColor: location.pathname === item.path
                              ? 'primary.dark'
                              : 'action.hover',
                          },
                          '&:focus': {
                            outline: '2px solid',
                            outlineColor: 'primary.main',
                            outlineOffset: '2px',
                          },
                        }}
                      >
                        {isTablet ? '' : item.label}
                      </Button>
                    ))}

                  {/* Medium Priority Items - Tablet+ */}
                  {!isTablet && navigationItems
                    .filter(item => item.priority === 'medium')
                    .map((item) => (
                      <Button
                        key={item.path}
                        startIcon={item.icon}
                        onClick={() => handleNavigate(item.path)}
                        variant={location.pathname === item.path ? 'contained' : 'text'}
                        size="medium"
                        sx={{
                          minWidth: 100,
                          minHeight: 40,
                          px: 2,
                          fontSize: '0.875rem',
                          fontWeight: 500,
                          textTransform: 'none',
                          borderRadius: 2,
                          '&:hover': {
                            backgroundColor: location.pathname === item.path
                              ? 'primary.dark'
                              : 'action.hover',
                          },
                          '&:focus': {
                            outline: '2px solid',
                            outlineColor: 'primary.main',
                            outlineOffset: '2px',
                          },
                        }}
                      >
                        {item.label}
                      </Button>
                    ))}

                  {/* Refresh Button - Enhanced */}
                  <IconButton
                    onClick={handleRefresh}
                    aria-label="Refresh data"
                    size={isTablet ? 'small' : 'medium'}
                    sx={{
                      ml: 1,
                      minWidth: 40,
                      minHeight: 40,
                      '&:hover': {
                        backgroundColor: 'action.hover',
                        transform: 'rotate(90deg)',
                      },
                      '&:focus': {
                        outline: '2px solid',
                        outlineColor: 'primary.main',
                        outlineOffset: '2px',
                      },
                      transition: 'all 0.2s ease-in-out',
                    }}
                  >
                    <Refresh />
                  </IconButton>

                  {/* Theme Toggle Button */}
                  <IconButton
                    onClick={toggleTheme}
                    aria-label={`Switch to ${themeMode === 'light' ? 'dark' : 'light'} theme`}
                    size={isTablet ? 'small' : 'medium'}
                    sx={{
                      ml: 1,
                      minWidth: 40,
                      minHeight: 40,
                      color: 'text.primary',
                      '&:hover': {
                        backgroundColor: 'action.hover',
                      },
                      '&:focus': {
                        outline: '2px solid',
                        outlineColor: 'primary.main',
                        outlineOffset: '2px',
                      },
                      transition: 'all 0.2s ease-in-out',
                    }}
                  >
                    {themeMode === 'light' ? <DarkMode /> : <LightMode />}
                  </IconButton>
                </Box>
              )}
            </Toolbar>
          </AppBar>

          {/* Mobile Navigation Drawer */}
          <MobileDrawer />

          {/* Main Content Area - Enhanced Layout */}
          <Box
            component="main"
            sx={{
              flex: 1,
              display: 'flex',
              flexDirection: 'column',
              width: '100%',
              minHeight: 0, // Allow flex shrinking
              backgroundColor: 'background.default',
              overflow: 'hidden',
            }}
          >
            <Routes>
              <Route path="/" element={<Navigate to="/chat" replace />} />
              <Route path="/chat" element={<CodeChatPage key={refreshKey} />} />
              <Route path="/tasks" element={<KanbanBoard key={refreshKey} />} />
              <Route path="/subchat-test" element={<SubchatTestPage key={refreshKey} />} /> {/* TODO: Remove before commit */}
              <Route path="/knowledge" element={<KnowledgeBrowser key={refreshKey} />} />
              <Route path="/code" element={<CodeSearchPage key={refreshKey} />} />
              <Route path="/tools" element={<HTTPToolsPage key={refreshKey} />} />
              <Route path="/subagents" element={<SubagentsPage key={refreshKey} />} />
              <Route path="/settings" element={<SettingsPage key={refreshKey} />} />
            </Routes>
          </Box>
        </Box>
      </ConversationModeProvider>
    </ThemeProvider>
  );
}

export default App;