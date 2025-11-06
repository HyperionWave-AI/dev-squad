# Deprecation Banner Integration Guide

This document explains how to add the deprecation banner to the old UI application.

## Component Created

Location: `/ui/src/components/DeprecationBanner.tsx`

Features:
- Dismissible banner with localStorage persistence
- Non-intrusive warning at top of app
- Link to migration guide (DEPRECATED.md)
- Material-UI Alert component for consistency

## Integration Steps

### Option 1: Add to App.tsx (Recommended)

Add the banner right after the AppBar in `src/App.tsx`:

```tsx
// 1. Import the component at the top
import { DeprecationBanner } from './components/DeprecationBanner';

// 2. Add after AppBar, before main content (around line 524)
// Find this section:
        </AppBar>

        {/* Mobile Navigation Drawer */}
        <MobileDrawer />

// Add banner here:
        </AppBar>

        {/* Deprecation Warning Banner */}
        <DeprecationBanner />

        {/* Mobile Navigation Drawer */}
        <MobileDrawer />
```

Full integration example:

```tsx
// After line 524 in App.tsx
        </AppBar>

        {/* Deprecation Warning Banner */}
        <DeprecationBanner />

        {/* Mobile Navigation Drawer */}
        <MobileDrawer />

        {/* Main Content Area - Enhanced Layout */}
        <Box
          component="main"
          sx={{
            flex: 1,
            // ... rest of styles
```

### Option 2: Add to Individual Pages

If you prefer to show the banner only on specific pages, import and use:

```tsx
import { DeprecationBanner } from '../components/DeprecationBanner';

function YourPage() {
  return (
    <Box>
      <DeprecationBanner />
      {/* Rest of page content */}
    </Box>
  );
}
```

## Testing

1. Start the dev server:
```bash
cd /Users/maxmednikov/MaxSpace/hyper/ui
npm run dev
```

2. Open http://localhost:5173
3. You should see the yellow warning banner at the top
4. Click "Learn More" - should open DEPRECATED.md (update URL in component)
5. Click X to dismiss - banner should disappear
6. Refresh page - banner should stay dismissed (localStorage)
7. Clear localStorage and refresh - banner should reappear

## Customization

### Change Banner Appearance

Edit `/ui/src/components/DeprecationBanner.tsx`:

```tsx
// Make it more prominent
severity="error"  // red instead of yellow

// Change text
<strong>Action Required:</strong> This UI will be removed soon.

// Change button behavior
const handleLearnMore = () => {
  window.location.href = '/ui2';  // Redirect to UI2
};
```

### Add "Switch to UI2" Button

```tsx
<Button
  variant="contained"
  color="warning"
  size="small"
  onClick={() => window.location.href = '/ui2'}
  sx={{ mr: 1 }}
>
  Switch to UI2 Now
</Button>
```

### Show on Specific Dates

```tsx
const [open, setOpen] = useState(() => {
  const currentDate = new Date();
  const showAfter = new Date('2025-01-01'); // Show banner after Jan 1, 2025

  if (currentDate < showAfter) return false;

  const dismissed = localStorage.getItem(STORAGE_KEY);
  return dismissed !== 'true';
});
```

## Removal Timeline

- **Now (Nov 2025)**: Banner is dismissible, informational
- **March 2026**: Make banner non-dismissible, more urgent
- **May 2026**: Remove entire `/ui` directory

## Need Help?

See `/ui/DEPRECATED.md` for complete deprecation details and migration guide.
