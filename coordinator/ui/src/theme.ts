import { createTheme } from '@mui/material/styles';

// Custom theme for Hyperion Coordinator
export const theme = createTheme({
  palette: {
    mode: 'light',
    primary: {
      main: '#2563eb', // Blue-600
      light: '#60a5fa', // Blue-400
      dark: '#1e40af', // Blue-700
      contrastText: '#ffffff',
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
  },
  typography: {
    fontFamily: [
      'Inter',
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
        root: {
          boxShadow: '0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1)',
          '&:hover': {
            boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)',
          },
        },
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
  },
});