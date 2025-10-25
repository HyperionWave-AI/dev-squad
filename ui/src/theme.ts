import { createTheme, type ThemeOptions } from '@mui/material/styles';

// Base theme configuration shared between light and dark modes
const baseTheme: ThemeOptions = {
  typography: {
    fontFamily: [
      'Inter',
      'system-ui',
      '-apple-system',
      'BlinkMacSystemFont',
      '"Segoe UI"',
      'Roboto',
      '"Helvetica Neue"',
      'Arial',
      'sans-serif',
    ].join(','),
    h1: {
      fontSize: '2.25rem',
      fontWeight: 700,
      lineHeight: 1.2,
    },
    h2: {
      fontSize: '1.875rem',
      fontWeight: 700,
      lineHeight: 1.3,
    },
    h3: {
      fontSize: '1.5rem',
      fontWeight: 600,
      lineHeight: 1.4,
    },
    h4: {
      fontSize: '1.25rem',
      fontWeight: 600,
      lineHeight: 1.4,
    },
    h5: {
      fontSize: '1.125rem',
      fontWeight: 600,
      lineHeight: 1.5,
    },
    h6: {
      fontSize: '1rem',
      fontWeight: 600,
      lineHeight: 1.5,
    },
    body1: {
      fontSize: '0.875rem',
      lineHeight: 1.5,
    },
    body2: {
      fontSize: '0.75rem',
      lineHeight: 1.5,
    },
  },
  shape: {
    borderRadius: 8,
  },
  components: {
    MuiCard: {
      styleOverrides: {
        root: ({ theme }) => ({
          boxShadow: theme.palette.mode === 'light' 
            ? '0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1)'
            : '0 1px 3px 0 rgb(0 0 0 / 0.3), 0 1px 2px -1px rgb(0 0 0 / 0.3)',
          '&:hover': {
            boxShadow: theme.palette.mode === 'light'
              ? '0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)'
              : '0 4px 6px -1px rgb(0 0 0 / 0.3), 0 2px 4px -2px rgb(0 0 0 / 0.3)',
          },
        }),
      },
    },
    MuiButton: {
      styleOverrides: {
        root: {
          textTransform: 'none',
          fontWeight: 500,
        },
      },
    },
    MuiChip: {
      styleOverrides: {
        root: {
          fontWeight: 500,
        },
      },
    },
    MuiAppBar: {
      styleOverrides: {
        root: ({ theme }) => ({
          backgroundColor: theme.palette.mode === 'light' 
            ? 'rgba(255, 255, 255, 0.95)' 
            : 'rgba(17, 24, 39, 0.95)',
          backdropFilter: 'blur(8px)',
          WebkitBackdropFilter: 'blur(8px)',
          borderBottom: `1px solid ${theme.palette.divider}`,
          boxShadow: theme.palette.mode === 'light'
            ? '0 1px 3px rgba(0, 0, 0, 0.1)'
            : '0 1px 3px rgba(0, 0, 0, 0.3)',
        }),
      },
    },
    MuiDrawer: {
      styleOverrides: {
        paper: ({ theme }) => ({
          backgroundColor: theme.palette.background.paper,
          borderRight: `1px solid ${theme.palette.divider}`,
        }),
      },
    },
    MuiListItemButton: {
      styleOverrides: {
        root: ({ theme }) => ({
          borderRadius: 8,
          margin: '0 8px 4px',
          '&.Mui-selected': {
            backgroundColor: theme.palette.primary.main,
            color: theme.palette.primary.contrastText,
            '& .MuiListItemIcon-root': {
              color: theme.palette.primary.contrastText,
            },
            '&:hover': {
              backgroundColor: theme.palette.primary.dark,
            },
          },
          '&:hover': {
            backgroundColor: theme.palette.mode === 'light'
              ? 'rgba(0, 0, 0, 0.04)'
              : 'rgba(255, 255, 255, 0.05)',
          },
        }),
      },
    },
  },
};

// Light theme configuration
const lightTheme = createTheme({
  ...baseTheme,
  palette: {
    mode: 'light',
    primary: {
      main: '#2563eb', // Blue-600
      light: '#60a5fa', // Blue-400
      dark: '#1e40af', // Blue-700
      contrastText: '#ffffff',
      50: '#eff6ff',
      100: '#dbeafe',
      500: '#3b82f6',
      600: '#2563eb',
      700: '#1d4ed8',
    },
    secondary: {
      main: '#9333ea', // Purple-600
      light: '#c084fc', // Purple-400
      dark: '#7e22ce', // Purple-700
      contrastText: '#ffffff',
    },
    success: {
      main: '#16a34a', // Green-600
      light: '#4ade80', // Green-400
      dark: '#15803d', // Green-700
    },
    warning: {
      main: '#ea580c', // Orange-600
      light: '#fb923c', // Orange-400
      dark: '#c2410c', // Orange-700
    },
    error: {
      main: '#dc2626', // Red-600
      light: '#f87171', // Red-400
      dark: '#b91c1c', // Red-700
    },
    info: {
      main: '#0891b2', // Cyan-600
      light: '#22d3ee', // Cyan-400
      dark: '#0e7490', // Cyan-700
    },
    background: {
      default: '#f8fafc', // Slate-50
      paper: '#ffffff',
    },
    text: {
      primary: '#1e293b', // Slate-800
      secondary: '#64748b', // Slate-500
    },
    divider: '#e2e8f0', // Slate-200
    action: {
      hover: 'rgba(0, 0, 0, 0.04)',
      selected: 'rgba(59, 130, 246, 0.1)',
    },
  },
});

// Dark theme configuration
const darkTheme = createTheme({
  ...baseTheme,
  palette: {
    mode: 'dark',
    primary: {
      main: '#60a5fa', // Blue-400 (lighter for dark mode)
      light: '#93c5fd', // Blue-300
      dark: '#3b82f6', // Blue-500
      contrastText: '#111827', // Dark text on light blue
      50: '#eff6ff',
      100: '#dbeafe',
      500: '#3b82f6',
      600: '#2563eb',
      700: '#1d4ed8',
    },
    secondary: {
      main: '#c084fc', // Purple-400
      light: '#d8b4fe', // Purple-300
      dark: '#a855f7', // Purple-500
      contrastText: '#111827',
    },
    success: {
      main: '#4ade80', // Green-400
      light: '#6ee7b7', // Green-300
      dark: '#16a34a', // Green-600
    },
    warning: {
      main: '#fb923c', // Orange-400
      light: '#fdba74', // Orange-300
      dark: '#ea580c', // Orange-600
    },
    error: {
      main: '#f87171', // Red-400
      light: '#fca5a5', // Red-300
      dark: '#dc2626', // Red-600
    },
    info: {
      main: '#22d3ee', // Cyan-400
      light: '#67e8f9', // Cyan-300
      dark: '#0891b2', // Cyan-600
    },
    background: {
      default: '#0f172a', // Slate-900
      paper: '#1e293b', // Slate-800
    },
    text: {
      primary: '#f1f5f9', // Slate-100
      secondary: '#cbd5e1', // Slate-300
    },
    divider: 'rgba(148, 163, 184, 0.2)', // Slate-400 with opacity
    action: {
      hover: 'rgba(255, 255, 255, 0.05)',
      selected: 'rgba(96, 165, 250, 0.1)',
    },
  },
});

// Function to get theme based on mode preference
export const getTheme = (mode: 'light' | 'dark' = 'light') => {
  return mode === 'dark' ? darkTheme : lightTheme;
};

// Default theme (light mode)
export const theme = lightTheme;

// Export both themes for direct access
export { lightTheme, darkTheme };

// Theme mode detection utility
export const getPreferredTheme = (): 'light' | 'dark' => {
  if (typeof window !== 'undefined') {
    // Check localStorage first
    const stored = localStorage.getItem('theme-mode');
    if (stored === 'light' || stored === 'dark') {
      return stored;
    }
    
    // Fall back to system preference
    if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
      return 'dark';
    }
  }
  
  return 'light';
};

// Theme persistence utility
export const setThemePreference = (mode: 'light' | 'dark') => {
  if (typeof window !== 'undefined') {
    localStorage.setItem('theme-mode', mode);
  }
};