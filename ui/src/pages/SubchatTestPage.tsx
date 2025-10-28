/**
 * SubchatTest Page Component
 * 
 * A responsive placeholder page with three action buttons following mobile-first design patterns.
 * Uses Material-UI components with responsive breakpoints and mobile-optimized touch targets.
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
    <Container 
      maxWidth="lg" 
      sx={{ 
        py: { xs: 2, sm: 3, md: 4 },
        px: { xs: 1, sm: 2 }
      }}
    >
      <Box 
        sx={{ 
          backgroundColor: 'background.paper', 
          borderRadius: { xs: 1, sm: 2 }, 
          boxShadow: { xs: 0, sm: 1 }, 
          p: { xs: 1.5, sm: 2, md: 3 }
        }}
      >
        {/* Responsive centered placeholder section */}
        <Paper
          variant="outlined"
          sx={{
            p: { xs: 3, sm: 4, md: 6 },
            textAlign: 'center',
            backgroundColor: 'background.default',
            borderRadius: { xs: 2, sm: 3 },
            border: '2px dashed',
            borderColor: 'divider',
            minHeight: { xs: 'auto', sm: 350, md: 400 },
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <Box 
            sx={{ 
              maxWidth: { xs: '100%', sm: 500, md: 600 }, 
              mx: 'auto',
              width: '100%'
            }}
          >
            {/* Responsive welcome heading */}
            <Typography 
              variant="h4" 
              component="h1"
              color="text.primary" 
              gutterBottom 
              sx={{ 
                fontWeight: 600,
                mb: { xs: 1.5, sm: 2 },
                fontSize: { 
                  xs: '1.75rem', 
                  sm: '2.125rem', 
                  md: '2.5rem' 
                },
              }}
            >
              Welcome to Subchat Test
            </Typography>
            
            {/* Responsive description text */}
            <Typography 
              variant="body1" 
              color="text.secondary" 
              paragraph 
              sx={{ 
                mb: { xs: 3, sm: 4 },
                fontSize: { xs: '1rem', sm: '1.05rem', md: '1.1rem' },
                lineHeight: { xs: 1.5, sm: 1.6 },
                px: { xs: 0, sm: 1 }
              }}
            >
              This is your testing environment for subchat functionality. 
              Choose an action below to explore and test the subchat features with specialist agents.
            </Typography>
            
            {/* Responsive status indicators */}
            {(buttonStates.started || buttonStates.configured || buttonStates.analyzed) && (
              <Box sx={{ mb: { xs: 2, sm: 3 } }}>
                {buttonStates.started && (
                  <Typography 
                    variant="body2" 
                    color="success.main" 
                    sx={{ 
                      fontWeight: 500, 
                      mb: 0.5,
                      fontSize: { xs: '0.875rem', sm: '0.875rem' }
                    }}
                  >
                    ✓ Testing environment started
                  </Typography>
                )}
                {buttonStates.configured && (
                  <Typography 
                    variant="body2" 
                    color="info.main" 
                    sx={{ 
                      fontWeight: 500, 
                      mb: 0.5,
                      fontSize: { xs: '0.875rem', sm: '0.875rem' }
                    }}
                  >
                    ✓ Configuration completed
                  </Typography>
                )}
                {buttonStates.analyzed && (
                  <Typography 
                    variant="body2" 
                    color="warning.main" 
                    sx={{ 
                      fontWeight: 500, 
                      mb: 0.5,
                      fontSize: { xs: '0.875rem', sm: '0.875rem' }
                    }}
                  >
                    ✓ Analysis initiated
                  </Typography>
                )}
              </Box>
            )}
            
            {/* Responsive action buttons with mobile-first design */}
            <Stack 
              direction={{ xs: 'column', sm: 'row' }} 
              spacing={{ xs: 2, sm: 2 }} 
              justifyContent="center"
              sx={{ 
                mb: { xs: 2, sm: 3 },
                alignItems: { xs: 'stretch', sm: 'center' }
              }}
            >
              {/* Get Started button - mobile optimized */}
              <Button
                variant="contained"
                size="large"
                startIcon={<GetStartedIcon />}
                onClick={handleGetStarted}
                disabled={buttonStates.started}
                aria-label="Get started with subchat testing"
                sx={{
                  py: { xs: 1.75, sm: 1.5 },
                  px: { xs: 2, sm: 3 },
                  borderRadius: 2,
                  textTransform: 'none',
                  fontWeight: 600,
                  fontSize: { xs: '1rem', sm: '1rem' },
                  minWidth: { xs: 'auto', sm: 140 },
                  minHeight: { xs: 48, sm: 'auto' }, // 48px minimum touch target
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

              {/* Configure button - mobile optimized */}
              <Button
                variant="outlined"
                size="large"
                startIcon={<ConfigureIcon />}
                onClick={handleConfigure}
                disabled={buttonStates.configured}
                aria-label="Configure subchat settings"
                sx={{
                  py: { xs: 1.75, sm: 1.5 },
                  px: { xs: 2, sm: 3 },
                  borderRadius: 2,
                  textTransform: 'none',
                  fontWeight: 600,
                  fontSize: { xs: '1rem', sm: '1rem' },
                  minWidth: { xs: 'auto', sm: 140 },
                  minHeight: { xs: 48, sm: 'auto' }, // 48px minimum touch target
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

              {/* Analyze button - mobile optimized */}
              <Button
                variant="contained"
                size="large"
                startIcon={<AnalyzeIcon />}
                onClick={handleAnalyze}
                disabled={buttonStates.analyzed}
                aria-label="Analyze subchat performance"
                color="warning"
                sx={{
                  py: { xs: 1.75, sm: 1.5 },
                  px: { xs: 2, sm: 3 },
                  borderRadius: 2,
                  textTransform: 'none',
                  fontWeight: 600,
                  fontSize: { xs: '1rem', sm: '1rem' },
                  minWidth: { xs: 'auto', sm: 140 },
                  minHeight: { xs: 48, sm: 'auto' }, // 48px minimum touch target
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
            
            {/* Responsive help text */}
            <Typography 
              variant="caption" 
              color="text.secondary" 
              display="block"
              sx={{ 
                mt: { xs: 2, sm: 3 },
                fontStyle: 'italic',
                fontSize: { xs: '0.75rem', sm: '0.75rem' },
                px: { xs: 1, sm: 0 }
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