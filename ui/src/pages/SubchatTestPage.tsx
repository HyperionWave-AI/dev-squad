/**
 * SubchatTest Page Component
 * 
 * A placeholder page with three action buttons following the existing design system patterns.
 * Uses Material-UI components for consistency with SubchatList and SubchatCreationDialog.
 */

import { useState } from 'react';
import {
  Box,
  Button,
  Typography,
  Container,
  Paper,
  Stack,
} from '@mui/material';
import { 
  PlayArrow as GetStartedIcon,
  Settings as ConfigureIcon,
  Assessment as AnalyzeIcon,
} from '@mui/icons-material';

export const SubchatTestPage = () => {
  const [buttonStates, setButtonStates] = useState({
    started: false,
    configured: false,
    analyzed: false,
  });

  const handleGetStarted = () => {
    setButtonStates(prev => ({ ...prev, started: true }));
    console.log('Get Started clicked - ready to implement functionality');
  };

  const handleConfigure = () => {
    setButtonStates(prev => ({ ...prev, configured: true }));
    console.log('Configure clicked - ready to implement configuration');
  };

  const handleAnalyze = () => {
    setButtonStates(prev => ({ ...prev, analyzed: true }));
    console.log('Analyze clicked - ready to implement analysis');
  };

  return (
    <Container maxWidth="lg" sx={{ py: 4 }}>
      <Box sx={{ backgroundColor: 'background.paper', borderRadius: 2, boxShadow: 1, p: 3 }}>
        {/* Centered placeholder section */}
        <Paper
          variant="outlined"
          sx={{
            p: 6,
            textAlign: 'center',
            backgroundColor: 'background.default',
            borderRadius: 3,
            border: '2px dashed',
            borderColor: 'divider',
            minHeight: 400,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <Box sx={{ maxWidth: 600, mx: 'auto' }}>
            {/* Welcome heading */}
            <Typography 
              variant="h4" 
              component="h1"
              color="text.primary" 
              gutterBottom 
              sx={{ 
                fontWeight: 600,
                mb: 2,
              }}
            >
              Welcome to Subchat Test
            </Typography>
            
            {/* Description text */}
            <Typography 
              variant="body1" 
              color="text.secondary" 
              paragraph 
              sx={{ 
                mb: 4,
                fontSize: '1.1rem',
                lineHeight: 1.6,
              }}
            >
              This is your testing environment for subchat functionality. 
              Choose an action below to explore and test the subchat features with specialist agents.
            </Typography>
            
            {/* Status indicators */}
            {(buttonStates.started || buttonStates.configured || buttonStates.analyzed) && (
              <Box sx={{ mb: 3 }}>
                {buttonStates.started && (
                  <Typography 
                    variant="body2" 
                    color="success.main" 
                    sx={{ fontWeight: 500, mb: 0.5 }}
                  >
                    ✓ Testing environment started
                  </Typography>
                )}
                {buttonStates.configured && (
                  <Typography 
                    variant="body2" 
                    color="info.main" 
                    sx={{ fontWeight: 500, mb: 0.5 }}
                  >
                    ✓ Configuration completed
                  </Typography>
                )}
                {buttonStates.analyzed && (
                  <Typography 
                    variant="body2" 
                    color="warning.main" 
                    sx={{ fontWeight: 500, mb: 0.5 }}
                  >
                    ✓ Analysis initiated
                  </Typography>
                )}
              </Box>
            )}
            
            {/* Three action buttons */}
            <Stack 
              direction={{ xs: 'column', sm: 'row' }} 
              spacing={2} 
              justifyContent="center"
              sx={{ mb: 3 }}
            >
              {/* Get Started button */}
              <Button
                variant="contained"
                size="large"
                startIcon={<GetStartedIcon />}
                onClick={handleGetStarted}
                disabled={buttonStates.started}
                aria-label="Get started with subchat testing"
                sx={{
                  py: 1.5,
                  px: 3,
                  borderRadius: 2,
                  textTransform: 'none',
                  fontWeight: 600,
                  fontSize: '1rem',
                  minWidth: 140,
                  boxShadow: 2,
                  '&:hover': {
                    boxShadow: 4,
                    transform: 'translateY(-1px)',
                  },
                  '&:disabled': {
                    backgroundColor: 'success.main',
                    color: 'success.contrastText',
                    opacity: 0.8,
                  },
                  transition: 'all 0.2s ease-in-out',
                }}
              >
                {buttonStates.started ? 'Started' : 'Get Started'}
              </Button>

              {/* Configure button */}
              <Button
                variant="outlined"
                size="large"
                startIcon={<ConfigureIcon />}
                onClick={handleConfigure}
                disabled={buttonStates.configured}
                aria-label="Configure subchat settings"
                sx={{
                  py: 1.5,
                  px: 3,
                  borderRadius: 2,
                  textTransform: 'none',
                  fontWeight: 600,
                  fontSize: '1rem',
                  minWidth: 140,
                  borderWidth: 2,
                  '&:hover': {
                    borderWidth: 2,
                    transform: 'translateY(-1px)',
                  },
                  '&:disabled': {
                    borderColor: 'info.main',
                    color: 'info.main',
                    opacity: 0.8,
                  },
                  transition: 'all 0.2s ease-in-out',
                }}
              >
                {buttonStates.configured ? 'Configured' : 'Configure'}
              </Button>

              {/* Analyze button */}
              <Button
                variant="contained"
                size="large"
                startIcon={<AnalyzeIcon />}
                onClick={handleAnalyze}
                disabled={buttonStates.analyzed}
                aria-label="Analyze subchat performance"
                color="warning"
                sx={{
                  py: 1.5,
                  px: 3,
                  borderRadius: 2,
                  textTransform: 'none',
                  fontWeight: 600,
                  fontSize: '1rem',
                  minWidth: 140,
                  boxShadow: 2,
                  '&:hover': {
                    boxShadow: 4,
                    transform: 'translateY(-1px)',
                  },
                  '&:disabled': {
                    backgroundColor: 'warning.main',
                    color: 'warning.contrastText',
                    opacity: 0.8,
                  },
                  transition: 'all 0.2s ease-in-out',
                }}
              >
                {buttonStates.analyzed ? 'Analyzed' : 'Analyze'}
              </Button>
            </Stack>
            
            {/* Additional help text */}
            <Typography 
              variant="caption" 
              color="text.secondary" 
              display="block"
              sx={{ 
                mt: 3,
                fontStyle: 'italic',
              }}
            >
              Choose any of the actions above to begin your subchat testing journey
            </Typography>
          </Box>
        </Paper>
      </Box>
    </Container>
  );
};