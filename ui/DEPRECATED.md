# ⚠️ DEPRECATED - This UI is no longer maintained

**NOTICE**: This UI implementation (`/ui`) has been **deprecated** and replaced by **UI2** (`/ui2`).

## Status

- **Deprecated Date**: November 5, 2025
- **Planned Removal**: May 2026 (6 months from deprecation)
- **Current Status**: Read-only maintenance mode (critical security fixes only)
- **Replacement**: UI2 (`/ui2`) is production-ready and actively maintained

## Why Was This Deprecated?

The original UI (`/ui`) served us well but has been superseded by UI2 with significant improvements:

### Technology Stack Comparison

| Feature | Old UI (`/ui`) | New UI2 (`/ui2`) |
|---------|---------------|-----------------|
| React | 19.1.1 | 19.1.1 |
| TypeScript | 5.8.3 | 5.8.3 |
| UI Framework | Material-UI (@mui/material) | Radix UI (headless components) |
| Styling | Tailwind CSS + MUI styles | Tailwind CSS only |
| Component Architecture | Mixed patterns | Atomic Design (atoms/molecules/organisms) |
| Design System | MUI theme customization | Custom design system with CVA |
| Icons | @mui/icons-material + @heroicons/react | lucide-react |
| Animation | Basic MUI transitions | Framer Motion |
| Bundle Size | Larger (MUI overhead) | Smaller (headless components) |
| Accessibility | MUI defaults | Full ARIA + keyboard nav custom impl |
| Dark Mode | MUI ThemeProvider | Custom theme system |
| Developer Experience | MUI abstractions | Direct control with Radix primitives |

### Key Improvements in UI2

1. **Modern Component Architecture**
   - Atomic Design pattern (atoms → molecules → organisms → templates → pages)
   - Smaller, more composable components
   - Better separation of concerns

2. **Better Performance**
   - Smaller bundle size (no Material-UI dependency)
   - Faster initial load
   - Optimized re-renders with Radix UI primitives

3. **Enhanced Developer Experience**
   - Full TypeScript coverage with better types
   - CVA (class-variance-authority) for type-safe variants
   - Better component discoverability
   - Consistent API patterns

4. **Superior Design System**
   - Custom design tokens
   - Tailwind-first approach (no style conflicts)
   - More flexible theming
   - Better dark mode support

5. **Improved Accessibility**
   - Full keyboard navigation
   - Complete ARIA implementation
   - Screen reader optimized
   - Focus management

6. **Better Testing**
   - Playwright E2E tests
   - Component unit tests with Vitest
   - Accessibility testing with @axe-core/playwright

## Migration Guide

### For Developers

#### 1. Update Your Development Workflow

```bash
# Old UI (deprecated)
cd /Users/maxmednikov/MaxSpace/hyper/ui
npm run dev  # ❌ Don't use

# New UI2 (use this)
cd /Users/maxmednikov/MaxSpace/hyper/ui2
npm run dev  # ✅ Use this instead
```

#### 2. Build Process

```bash
# Old UI
cd ui && npm run build  # ❌ Deprecated

# New UI2
cd ui2 && npm run build  # ✅ Use this
```

#### 3. Component Migration Mapping

| Old UI (Material-UI) | New UI2 (Radix UI) | Notes |
|---------------------|-------------------|-------|
| `@mui/material/Button` | `@/components/ui/button` | Custom Button with CVA variants |
| `@mui/material/Card` | `@/components/ui/card` | Simpler API, more flexible |
| `@mui/material/Dialog` | `@radix-ui/react-dialog` | More control over behavior |
| `@mui/material/Select` | `@radix-ui/react-select` | Headless, fully customizable |
| `@mui/material/AppBar` | Custom header components | Tailwind-based layout |
| `@mui/icons-material/*` | `lucide-react` | Modern icon set |
| `@heroicons/react` | `lucide-react` | Unified icon library |

#### 4. Styling Migration

```tsx
// Old UI - Material-UI sx prop
<Box sx={{
  p: 2,
  display: 'flex',
  gap: 2,
  backgroundColor: 'background.paper'
}}>

// New UI2 - Tailwind classes
<div className="p-4 flex gap-4 bg-white dark:bg-gray-800">
```

#### 5. Theme Migration

```tsx
// Old UI - MUI ThemeProvider
import { ThemeProvider } from '@mui/material';
import { getTheme } from './theme';

<ThemeProvider theme={getTheme('light')}>

// New UI2 - Custom theme context
import { ThemeProvider } from '@/contexts/ThemeContext';

<ThemeProvider defaultTheme="light">
```

### Key File Location Changes

| Component Type | Old UI Path | New UI2 Path |
|---------------|-------------|--------------|
| Pages | `/ui/src/pages/` | `/ui2/src/pages/` |
| Components | `/ui/src/components/` | `/ui2/src/components/{atoms,molecules,organisms}/` |
| Styles | `/ui/src/App.css`, inline sx | `/ui2/src/styles/` + Tailwind |
| Types | `/ui/src/types/` | `/ui2/src/types/` |
| Services | `/ui/src/services/` | `/ui2/src/lib/` |
| Theme | `/ui/src/theme.ts` | `/ui2/src/contexts/ThemeContext.tsx` |

### Common Migration Patterns

#### 1. Page Component Migration

```tsx
// Old UI - Material-UI based
import { Box, Typography, Button } from '@mui/material';

function TasksPage() {
  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4">Tasks</Typography>
      <Button variant="contained">Create Task</Button>
    </Box>
  );
}

// New UI2 - Tailwind + Radix UI
import { Button } from '@/components/ui/button';

function TasksPage() {
  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold">Tasks</h1>
      <Button variant="default">Create Task</Button>
    </div>
  );
}
```

#### 2. Form Migration

```tsx
// Old UI
import { TextField, Select, MenuItem } from '@mui/material';

<TextField
  label="Task Name"
  variant="outlined"
  fullWidth
/>
<Select>
  <MenuItem value="high">High</MenuItem>
</Select>

// New UI2
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem } from '@/components/ui/select';

<div className="space-y-2">
  <label>Task Name</label>
  <Input placeholder="Enter task name" />
</div>
<Select>
  <SelectContent>
    <SelectItem value="high">High</SelectItem>
  </SelectContent>
</Select>
```

## Feature Parity Status

✅ **Complete**: All features from old UI implemented in UI2
✅ **Task Dashboard**: Kanban board with drag-and-drop
✅ **Knowledge Browser**: Semantic search and collections
✅ **Code Search**: Semantic code search with syntax highlighting
✅ **Reflection System**: Decision tracking and lessons learned
✅ **MCP Servers**: Server management UI
✅ **Subagents**: Agent configuration and monitoring
✅ **Settings**: Configuration management
✅ **Dark Mode**: Enhanced dark theme support
✅ **Responsive Design**: Mobile, tablet, desktop optimized
✅ **Accessibility**: WCAG 2.1 AA compliant

## Timeline

- **October 2025**: UI2 development completed
- **November 5, 2025**: Old UI officially deprecated
- **November-December 2025**: Parallel operation (both UIs available)
- **January 2026**: Old UI marked for removal in documentation
- **March 2026**: Warning banners added to old UI
- **May 2026**: Old UI removed from repository

## Getting Help

If you encounter issues migrating to UI2:

1. **Check UI2 Documentation**: See `/ui2/README.md` for setup and usage
2. **Component Examples**: Browse `/ui2/src/components/` for component usage
3. **Atomic Design Guide**: Understand the new component architecture
4. **Ask the Team**: Reach out to the AI & Experience Squad

## For Users

If you're still accessing the old UI:

1. **Switch to UI2 immediately** for the best experience
2. **Bookmark the new URL**: Use UI2's deployment URL
3. **Report any missing features**: Help us ensure feature parity
4. **Enjoy improved performance**: UI2 is faster and more responsive

## What Happens to the Old UI?

1. **No New Features**: No new features will be added to the old UI
2. **Critical Security Only**: Only critical security patches will be applied
3. **Final Removal**: Complete removal planned for May 2026
4. **Archive**: Code will be archived for reference but removed from active codebase

## Why Not Keep Both?

Maintaining two UI implementations:
- Doubles maintenance burden
- Creates inconsistent user experience
- Wastes development resources
- Causes confusion for new developers
- Prevents us from innovating on the modern UI

## Questions?

- **Why Material-UI → Radix UI?** Better performance, smaller bundle, more control
- **Will bookmarks break?** Update bookmarks to UI2 URLs
- **Can I still use old UI?** Only until May 2026
- **Is UI2 stable?** Yes, production-ready and actively maintained
- **What if I find a bug?** Report it! We're committed to making UI2 better

---

**TLDR**: Use `/ui2` for all new work. Old `/ui` will be removed in May 2026.

**Documentation**: See [UI2 README](/ui2/README.md) for complete setup and usage guide.

**Last Updated**: November 5, 2025
