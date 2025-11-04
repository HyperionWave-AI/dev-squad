# Hyperion Coordinator UI - UI/UX Patterns

## Table of Contents

1. [Design System Overview](#design-system-overview)
2. [Material-UI Theme](#material-ui-theme)
3. [Tailwind CSS Configuration](#tailwind-css-configuration)
4. [Dark Mode](#dark-mode)
5. [Accessibility (WCAG 2.1 AA)](#accessibility-wcag-21-aa)
6. [Responsive Design](#responsive-design)
7. [Component Patterns](#component-patterns)
8. [Typography](#typography)
9. [Color System](#color-system)
10. [Spacing & Layout](#spacing--layout)
11. [Interactive States](#interactive-states)
12. [Best Practices](#best-practices)

---

## Design System Overview

The Hyperion Coordinator UI follows a **mobile-first, accessible-first** design approach with a dual styling system:

- **Material-UI (MUI)** for component library and theming
- **Tailwind CSS** for utility-first styling and responsive design

### Design Principles

1. **Consistency**: Unified visual language across all components
2. **Accessibility**: WCAG 2.1 AA compliance throughout
3. **Responsiveness**: Mobile-first design (375px → 1920px)
4. **Performance**: Optimized rendering and bundle size
5. **Maintainability**: Scalable design tokens and patterns

### Key Features

- 🎨 Custom color palette with brand colors
- 🌙 Dark mode with system preference detection
- 📱 Mobile-first responsive breakpoints
- ♿ WCAG 2.1 AA accessibility compliance
- 🎯 44px minimum touch targets
- 📐 8px grid system for spacing

---

## Material-UI Theme

### Theme Configuration (`src/theme.ts`)

The theme is configured with light and dark mode variants.

### Color Palette

#### Light Mode

```typescript
palette: {
  mode: 'light',
  primary: {
    main: '#2563eb',      // Blue-600
    light: '#60a5fa',     // Blue-400
    dark: '#1e40af',      // Blue-700
    contrastText: '#ffffff',
  },
  secondary: {
    main: '#9333ea',      // Purple-600
    light: '#c084fc',     // Purple-400
    dark: '#7e22ce',      // Purple-700
    contrastText: '#ffffff',
  },
  success: {
    main: '#16a34a',      // Green-600
    light: '#4ade80',     // Green-400
    dark: '#15803d',      // Green-700
  },
  warning: {
    main: '#ea580c',      // Orange-600
    light: '#fb923c',     // Orange-400
    dark: '#c2410c',      // Orange-700
  },
  error: {
    main: '#dc2626',      // Red-600
    light: '#f87171',     // Red-400
    dark: '#b91c1c',      // Red-700
  },
  background: {
    default: '#f8fafc',   // Slate-50
    paper: '#ffffff',
  },
  text: {
    primary: '#1e293b',   // Slate-800
    secondary: '#64748b', // Slate-500
  },
}
```

#### Dark Mode

```typescript
palette: {
  mode: 'dark',
  primary: {
    main: '#60a5fa',      // Blue-400 (lighter for dark mode)
    light: '#93c5fd',     // Blue-300
    dark: '#3b82f6',      // Blue-500
    contrastText: '#111827',
  },
  secondary: {
    main: '#c084fc',      // Purple-400
    light: '#d8b4fe',     // Purple-300
    dark: '#a855f7',      // Purple-500
    contrastText: '#111827',
  },
  background: {
    default: '#0f172a',   // Slate-900
    paper: '#1e293b',     // Slate-800
  },
  text: {
    primary: '#f1f5f9',   // Slate-100
    secondary: '#cbd5e1', // Slate-300
  },
}
```

### Typography

```typescript
typography: {
  fontFamily: [
    'Inter',
    'system-ui',
    '-apple-system',
    'BlinkMacSystemFont',
    '"Segoe UI"',
    'Roboto',
    'sans-serif',
  ].join(','),
  h1: {
    fontSize: '2.25rem',  // 36px
    fontWeight: 700,
    lineHeight: 1.2,
  },
  h2: {
    fontSize: '1.875rem', // 30px
    fontWeight: 700,
    lineHeight: 1.3,
  },
  h3: {
    fontSize: '1.5rem',   // 24px
    fontWeight: 600,
    lineHeight: 1.4,
  },
  h4: {
    fontSize: '1.25rem',  // 20px
    fontWeight: 600,
    lineHeight: 1.4,
  },
  h5: {
    fontSize: '1.125rem', // 18px
    fontWeight: 600,
    lineHeight: 1.5,
  },
  h6: {
    fontSize: '1rem',     // 16px
    fontWeight: 600,
    lineHeight: 1.5,
  },
  body1: {
    fontSize: '0.875rem', // 14px
    lineHeight: 1.5,
  },
  body2: {
    fontSize: '0.75rem',  // 12px
    lineHeight: 1.5,
  },
}
```

### Component Overrides

#### Card Component

```typescript
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
}
```

#### Button Component

```typescript
MuiButton: {
  styleOverrides: {
    root: {
      textTransform: 'none',  // No uppercase transformation
      fontWeight: 500,
    },
  },
}
```

#### AppBar Component

```typescript
MuiAppBar: {
  styleOverrides: {
    root: ({ theme }) => ({
      backgroundColor: theme.palette.mode === 'light'
        ? 'rgba(255, 255, 255, 0.95)'
        : 'rgba(17, 24, 39, 0.95)',
      backdropFilter: 'blur(8px)',
      WebkitBackdropFilter: 'blur(8px)',
      borderBottom: `1px solid ${theme.palette.divider}`,
    }),
  },
}
```

#### ListItemButton (Navigation)

```typescript
MuiListItemButton: {
  styleOverrides: {
    root: ({ theme }) => ({
      borderRadius: 8,
      margin: '0 8px 4px',
      '&.Mui-selected': {
        backgroundColor: theme.palette.primary.main,
        color: theme.palette.primary.contrastText,
        '&:hover': {
          backgroundColor: theme.palette.primary.dark,
        },
      },
    }),
  },
}
```

### Using the Theme

```typescript
import { getTheme, getPreferredTheme, setThemePreference } from '@/theme';
import { ThemeProvider } from '@mui/material/styles';

function App() {
  const [mode, setMode] = useState<'light' | 'dark'>(getPreferredTheme());
  const theme = getTheme(mode);

  const toggleTheme = () => {
    const newMode = mode === 'light' ? 'dark' : 'light';
    setMode(newMode);
    setThemePreference(newMode);
  };

  return (
    <ThemeProvider theme={theme}>
      <Button onClick={toggleTheme}>Toggle Theme</Button>
    </ThemeProvider>
  );
}
```

---

## Tailwind CSS Configuration

### Mobile-First Breakpoints

Aligned with Material-UI breakpoints:

```javascript
screens: {
  'xs': '0px',
  'sm': '600px',      // Material-UI sm
  'md': '900px',      // Material-UI md
  'lg': '1200px',     // Material-UI lg
  'xl': '1536px',     // Material-UI xl
  // Additional mobile breakpoints
  'mobile-sm': '375px',   // Small mobile devices
  'mobile-lg': '414px',   // Large mobile devices
  'tablet-sm': '768px',   // Small tablets
  'tablet-lg': '1024px',  // Large tablets
}
```

### Touch-Friendly Spacing

```javascript
spacing: {
  'touch-sm': '2.75rem',   // 44px - minimum touch target
  'touch-md': '3rem',      // 48px - comfortable touch target
  'touch-lg': '3.5rem',    // 56px - large touch target
}

minHeight: {
  'touch': '44px',           // iOS minimum
  'touch-comfortable': '48px',
  'touch-large': '56px',     // Material Design large
}

minWidth: {
  'touch': '44px',
  'touch-comfortable': '48px',
  'touch-large': '56px',
}
```

### Custom Colors

```javascript
colors: {
  primary: {
    50: '#eff6ff',
    100: '#dbeafe',
    // ... full scale
    600: '#2563eb',  // Main primary
    900: '#1e3a8a',
  },
  secondary: {
    50: '#faf5ff',
    // ... full scale
    600: '#9333ea',  // Main secondary
    900: '#581c87',
  },
}
```

### Custom Utilities

```javascript
// Touch-friendly interactive elements
'.touch-target': {
  minHeight: '44px',
  minWidth: '44px',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
}

// Mobile-safe text sizing (prevents zoom on iOS)
'.text-mobile-safe': {
  fontSize: '16px',
}

// Mobile-optimized scrolling
'.scroll-mobile': {
  '-webkit-overflow-scrolling': 'touch',
  overscrollBehavior: 'contain',
}

// Safe area padding for devices with notches
'.safe-area-padding': {
  paddingTop: 'env(safe-area-inset-top)',
  paddingRight: 'env(safe-area-inset-right)',
  paddingBottom: 'env(safe-area-inset-bottom)',
  paddingLeft: 'env(safe-area-inset-left)',
}
```

### Using Tailwind Classes

```typescript
// Responsive layout
<div className="flex flex-col md:flex-row gap-4">
  <div className="w-full md:w-1/2">Column 1</div>
  <div className="w-full md:w-1/2">Column 2</div>
</div>

// Touch-friendly button
<button className="touch-target bg-primary-600 text-white rounded-lg">
  Click Me
</button>

// Mobile-specific visibility
<div className="mobile-only">Visible only on mobile</div>
<div className="desktop-only">Visible only on desktop</div>
```

---

## Dark Mode

### System Preference Detection

```typescript
// src/theme.ts
export const getPreferredTheme = (): 'light' | 'dark' => {
  if (typeof window !== 'undefined') {
    // Check localStorage first
    const stored = localStorage.getItem('theme-mode');
    if (stored === 'light' || stored === 'dark') {
      return stored;
    }

    // Fall back to system preference
    if (window.matchMedia &&
        window.matchMedia('(prefers-color-scheme: dark)').matches) {
      return 'dark';
    }
  }

  return 'light';
};
```

### Theme Persistence

```typescript
export const setThemePreference = (mode: 'light' | 'dark') => {
  if (typeof window !== 'undefined') {
    localStorage.setItem('theme-mode', mode);
  }
};
```

### Theme Toggle Component

```typescript
import { IconButton } from '@mui/material';
import { Brightness4, Brightness7 } from '@mui/icons-material';

interface ThemeToggleProps {
  mode: 'light' | 'dark';
  onToggle: () => void;
}

const ThemeToggle: React.FC<ThemeToggleProps> = ({ mode, onToggle }) => {
  return (
    <IconButton onClick={onToggle} aria-label="Toggle theme">
      {mode === 'dark' ? <Brightness7 /> : <Brightness4 />}
    </IconButton>
  );
};
```

### Theme-Aware Components

```typescript
import { useTheme } from '@mui/material/styles';

const MyComponent: React.FC = () => {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';

  return (
    <Box
      sx={{
        backgroundColor: isDark ? 'rgba(255, 255, 255, 0.05)' : 'rgba(0, 0, 0, 0.02)',
        color: theme.palette.text.primary,
      }}
    >
      Content
    </Box>
  );
};
```

---

## Accessibility (WCAG 2.1 AA)

### WCAG 2.1 AA Requirements

#### Color Contrast

- **Normal text**: 4.5:1 contrast ratio
- **Large text**: 3:1 contrast ratio
- **UI components**: 3:1 contrast ratio

All colors in the theme meet these requirements.

### Semantic HTML

```typescript
// ✅ GOOD - Semantic structure
<main>
  <header>
    <h1>Page Title</h1>
  </header>
  <article>
    <h2>Section Title</h2>
    <p>Content...</p>
  </article>
</main>

// ❌ BAD - No semantic structure
<div>
  <div>
    <div>Page Title</div>
  </div>
  <div>
    <div>Section Title</div>
    <div>Content...</div>
  </div>
</div>
```

### ARIA Labels and Roles

```typescript
// Icon buttons
<IconButton aria-label="Delete task" onClick={handleDelete}>
  <Delete />
</IconButton>

// Navigation
<nav aria-label="Main navigation">
  <List>
    <ListItemButton aria-current={isActive ? 'page' : undefined}>
      Dashboard
    </ListItemButton>
  </List>
</nav>

// Live regions for dynamic content
<div role="status" aria-live="polite" aria-atomic="true">
  Task created successfully
</div>

// Dialog
<Dialog
  open={open}
  onClose={handleClose}
  aria-labelledby="dialog-title"
  aria-describedby="dialog-description"
>
  <DialogTitle id="dialog-title">Confirm Action</DialogTitle>
  <DialogContent id="dialog-description">
    Are you sure?
  </DialogContent>
</Dialog>
```

### Keyboard Navigation

All interactive elements must be keyboard accessible:

```typescript
// Custom interactive element
<div
  role="button"
  tabIndex={0}
  onClick={handleClick}
  onKeyDown={(e) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      handleClick();
    }
  }}
  aria-label="Custom button"
>
  Click Me
</div>

// Skip navigation link
<a href="#main-content" className="sr-only focus:not-sr-only">
  Skip to main content
</a>
```

### Focus Management

```typescript
import { useRef, useEffect } from 'react';

const Modal: React.FC<ModalProps> = ({ open, onClose }) => {
  const firstFocusableRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (open && firstFocusableRef.current) {
      firstFocusableRef.current.focus();
    }
  }, [open]);

  return (
    <Dialog open={open} onClose={onClose}>
      <DialogTitle>Modal Title</DialogTitle>
      <DialogActions>
        <Button ref={firstFocusableRef} onClick={onClose}>
          Close
        </Button>
      </DialogActions>
    </Dialog>
  );
};
```

### Screen Reader Support

```typescript
// Descriptive labels
<TextField
  label="Email address"
  aria-describedby="email-help"
  aria-required="true"
  aria-invalid={hasError}
/>
<Typography id="email-help" variant="caption">
  We'll never share your email
</Typography>

// Loading states
{loading && (
  <Box role="status" aria-live="polite">
    <CircularProgress />
    <span className="sr-only">Loading content...</span>
  </Box>
)}

// Error announcements
{error && (
  <Alert severity="error" role="alert">
    {error.message}
  </Alert>
)}
```

### Heading Hierarchy

```typescript
// ✅ GOOD - Proper hierarchy
<h1>Page Title</h1>
<h2>Section 1</h2>
<h3>Subsection 1.1</h3>
<h3>Subsection 1.2</h3>
<h2>Section 2</h2>

// ❌ BAD - Skipping levels
<h1>Page Title</h1>
<h3>Section 1</h3>  // Skipped h2
```

---

## Responsive Design

### Breakpoint Behaviors

#### Desktop (1200px+)

- 4-column Kanban layout
- Full navigation sidebar
- Expanded data tables
- Side-by-side content areas

```typescript
<Grid container spacing={3}>
  <Grid item xs={12} lg={3}>Column 1</Grid>
  <Grid item xs={12} lg={3}>Column 2</Grid>
  <Grid item xs={12} lg={3}>Column 3</Grid>
  <Grid item xs={12} lg={3}>Column 4</Grid>
</Grid>
```

#### Tablet (768px - 1199px)

- 2-column Kanban layout
- Collapsible navigation
- Responsive tables
- Stacked content areas

```typescript
<Grid container spacing={2}>
  <Grid item xs={12} md={6}>Column 1</Grid>
  <Grid item xs={12} md={6}>Column 2</Grid>
</Grid>
```

#### Mobile (375px - 767px)

- Single-column Kanban with horizontal scroll
- Hamburger menu navigation
- Mobile-optimized tables
- Stacked content
- 44px minimum touch targets

```typescript
<Grid container spacing={2}>
  <Grid item xs={12}>Full Width Column</Grid>
</Grid>
```

### Responsive Patterns

#### Responsive Container

```typescript
<Container maxWidth="lg" sx={{ px: { xs: 2, sm: 3, md: 4 } }}>
  <Typography variant="h4" sx={{ fontSize: { xs: '1.5rem', md: '2rem' } }}>
    Responsive Title
  </Typography>
</Container>
```

#### Responsive Grid

```typescript
<Grid container spacing={{ xs: 2, md: 3 }}>
  <Grid item xs={12} sm={6} md={4} lg={3}>
    <Card>Item</Card>
  </Grid>
</Grid>
```

#### Responsive Typography

```typescript
<Typography
  variant="h1"
  sx={{
    fontSize: {
      xs: '1.5rem',   // 24px on mobile
      sm: '2rem',     // 32px on tablet
      md: '2.5rem',   // 40px on desktop
    }
  }}
>
  Responsive Heading
</Typography>
```

### Touch Optimization

```typescript
// Touch-friendly buttons
<Button
  sx={{
    minHeight: 44,
    minWidth: 44,
    px: 3,
    py: 1.5,
  }}
>
  Touch Me
</Button>

// Touch-friendly spacing
<Stack spacing={{ xs: 2, md: 3 }}>
  <Box>Item 1</Box>
  <Box>Item 2</Box>
</Stack>
```

### Horizontal Scroll Handling

```typescript
// Mobile Kanban with horizontal scroll
<Box
  sx={{
    display: 'flex',
    gap: 2,
    overflowX: 'auto',
    pb: 2,
    '&::-webkit-scrollbar': {
      height: 8,
    },
    '&::-webkit-scrollbar-thumb': {
      backgroundColor: 'rgba(0,0,0,0.2)',
      borderRadius: 4,
    },
  }}
>
  <Box sx={{ minWidth: 300 }}>Column 1</Box>
  <Box sx={{ minWidth: 300 }}>Column 2</Box>
  <Box sx={{ minWidth: 300 }}>Column 3</Box>
</Box>
```

---

## Component Patterns

### Card Pattern

```typescript
import { Card, CardHeader, CardContent, CardActions, Button } from '@mui/material';

<Card>
  <CardHeader
    title="Task Title"
    subheader="Created 2 hours ago"
    action={
      <IconButton aria-label="More options">
        <MoreVert />
      </IconButton>
    }
  />
  <CardContent>
    <Typography variant="body2" color="text.secondary">
      Task description goes here
    </Typography>
  </CardContent>
  <CardActions>
    <Button size="small">Edit</Button>
    <Button size="small" color="error">Delete</Button>
  </CardActions>
</Card>
```

### Form Pattern

```typescript
import { TextField, Button, Stack } from '@mui/material';

<Stack component="form" spacing={2} onSubmit={handleSubmit}>
  <TextField
    label="Task Title"
    required
    fullWidth
    value={title}
    onChange={(e) => setTitle(e.target.value)}
    aria-label="Task title"
  />
  <TextField
    label="Description"
    multiline
    rows={4}
    fullWidth
    value={description}
    onChange={(e) => setDescription(e.target.value)}
  />
  <Button type="submit" variant="contained">
    Submit
  </Button>
</Stack>
```

### List Pattern

```typescript
import { List, ListItem, ListItemText, ListItemButton, Divider } from '@mui/material';

<List>
  {items.map((item, index) => (
    <React.Fragment key={item.id}>
      <ListItemButton onClick={() => handleClick(item.id)}>
        <ListItemText
          primary={item.title}
          secondary={item.description}
        />
      </ListItemButton>
      {index < items.length - 1 && <Divider />}
    </React.Fragment>
  ))}
</List>
```

---

## Typography

### Type Scale

| Variant | Size | Weight | Use Case |
|---------|------|--------|----------|
| h1 | 36px | 700 | Page titles |
| h2 | 30px | 700 | Section headers |
| h3 | 24px | 600 | Subsection headers |
| h4 | 20px | 600 | Card titles |
| h5 | 18px | 600 | Small headings |
| h6 | 16px | 600 | Smallest headings |
| body1 | 14px | 400 | Body text |
| body2 | 12px | 400 | Secondary text |

### Usage Examples

```typescript
<Typography variant="h1" component="h1" gutterBottom>
  Page Title
</Typography>

<Typography variant="body1" color="text.primary">
  Primary body text
</Typography>

<Typography variant="body2" color="text.secondary">
  Secondary body text
</Typography>
```

---

## Color System

### Semantic Colors

| Purpose | Light Mode | Dark Mode | Usage |
|---------|------------|-----------|-------|
| Primary | #2563eb | #60a5fa | Main actions, links |
| Secondary | #9333ea | #c084fc | Accent elements |
| Success | #16a34a | #4ade80 | Completed states |
| Warning | #ea580c | #fb923c | Warnings |
| Error | #dc2626 | #f87171 | Errors |
| Info | #0891b2 | #22d3ee | Information |

### Status Colors

```typescript
const statusColors = {
  pending: 'warning',
  in_progress: 'info',
  completed: 'success',
  blocked: 'error',
} as const;

<Chip label="In Progress" color={statusColors.in_progress} />
```

---

## Spacing & Layout

### 8px Grid System

All spacing follows an 8px grid:

```typescript
spacing: {
  0: '0px',
  1: '8px',
  2: '16px',
  3: '24px',
  4: '32px',
  5: '40px',
  6: '48px',
}
```

### Layout Patterns

```typescript
// Page container
<Box sx={{ p: { xs: 2, md: 3 } }}>
  <Stack spacing={3}>
    <PageHeader />
    <PageContent />
  </Stack>
</Box>

// Content grid
<Grid container spacing={3}>
  <Grid item xs={12} md={8}>Main Content</Grid>
  <Grid item xs={12} md={4}>Sidebar</Grid>
</Grid>
```

---

## Interactive States

### Hover States

```typescript
<Button
  sx={{
    '&:hover': {
      backgroundColor: 'primary.dark',
      boxShadow: 2,
    },
  }}
>
  Hover Me
</Button>
```

### Focus States

```typescript
<TextField
  sx={{
    '& .MuiOutlinedInput-root': {
      '&.Mui-focused fieldset': {
        borderColor: 'primary.main',
        borderWidth: 2,
      },
    },
  }}
/>
```

### Active States

```typescript
<Button
  sx={{
    '&:active': {
      transform: 'scale(0.98)',
    },
  }}
>
  Click Me
</Button>
```

---

## Best Practices

### 1. Use Theme Values

```typescript
// ✅ GOOD - Use theme
<Box sx={{ color: 'primary.main', p: 2 }}>

// ❌ BAD - Hardcoded values
<Box style={{ color: '#2563eb', padding: '16px' }}>
```

### 2. Responsive Design

```typescript
// ✅ GOOD - Mobile-first
<Box sx={{ width: '100%', md: { width: '50%' } }}>

// ❌ BAD - Desktop-first
<Box sx={{ width: '50%' }}>
```

### 3. Accessibility

```typescript
// ✅ GOOD - Accessible
<IconButton aria-label="Delete">
  <Delete />
</IconButton>

// ❌ BAD - No label
<IconButton>
  <Delete />
</IconButton>
```

### 4. Consistent Spacing

```typescript
// ✅ GOOD - Use Stack/Grid
<Stack spacing={2}>
  <Item />
  <Item />
</Stack>

// ❌ BAD - Manual margins
<div>
  <div style={{ marginBottom: 16 }}>Item</div>
  <div>Item</div>
</div>
```

---

## Related Documentation

- [Architecture Overview](./ARCHITECTURE.md) - System architecture
- [Component Catalog](./COMPONENTS.md) - Component reference
- [Developer Guide](./DEVELOPER_GUIDE.md) - Development setup
- [Testing Guide](./TESTING.md) - Testing strategies
- [Troubleshooting](./TROUBLESHOOTING.md) - Common issues

---

**Last Updated**: 2025-11-04
**Version**: ui2 (Hyperion Coordinator UI)
**Maintainer**: Hyperion Platform Team
