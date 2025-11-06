import { useState } from 'react';
import { Alert, AlertTitle, Button, Collapse, IconButton, Box } from '@mui/material';
import { Close as CloseIcon, Warning as WarningIcon } from '@mui/icons-material';

/**
 * Deprecation banner to inform users that this UI is deprecated
 * and they should migrate to UI2.
 *
 * Features:
 * - Dismissible (stores preference in localStorage)
 * - Links to DEPRECATED.md for migration guide
 * - Non-intrusive but visible warning
 */
export function DeprecationBanner() {
  const STORAGE_KEY = 'hyperion-ui-deprecation-banner-dismissed';

  const [open, setOpen] = useState(() => {
    // Check if user has previously dismissed the banner
    const dismissed = localStorage.getItem(STORAGE_KEY);
    return dismissed !== 'true';
  });

  const handleDismiss = () => {
    setOpen(false);
    localStorage.setItem(STORAGE_KEY, 'true');
  };

  const handleLearnMore = () => {
    window.open('https://github.com/yourusername/hyperion/blob/main/ui/DEPRECATED.md', '_blank');
  };

  return (
    <Collapse in={open}>
      <Alert
        severity="warning"
        icon={<WarningIcon />}
        sx={{
          mb: 0,
          borderRadius: 0,
          borderBottom: '1px solid',
          borderColor: 'warning.light',
          backgroundColor: 'warning.light',
          color: 'warning.dark',
          '& .MuiAlert-icon': {
            color: 'warning.main',
          },
        }}
        action={
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Button
              color="warning"
              size="small"
              onClick={handleLearnMore}
              sx={{
                fontWeight: 600,
                textTransform: 'none',
                minHeight: 32,
              }}
            >
              Learn More
            </Button>
            <IconButton
              aria-label="Close deprecation notice"
              color="inherit"
              size="small"
              onClick={handleDismiss}
            >
              <CloseIcon fontSize="small" />
            </IconButton>
          </Box>
        }
      >
        <AlertTitle sx={{ fontWeight: 700, mb: 0.5 }}>
          ⚠️ This UI is Deprecated
        </AlertTitle>
        <Box sx={{ fontSize: '0.875rem' }}>
          <strong>UI2 is now available!</strong> This version will be removed in <strong>May 2026</strong>.
          Please migrate to <code style={{
            backgroundColor: 'rgba(0, 0, 0, 0.1)',
            padding: '2px 6px',
            borderRadius: '4px',
            fontFamily: 'monospace',
          }}>/ui2</code> for better performance and new features.
        </Box>
      </Alert>
    </Collapse>
  );
}
