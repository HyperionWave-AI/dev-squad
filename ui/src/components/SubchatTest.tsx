/**
 * SubchatTest Component
 *
 * Test component for subchat functionality with Material-UI styling.
 * Demonstrates subchat patterns and provides testing interface.
 * Responsive design optimized for mobile, tablet, and desktop viewports.
 */

import React, { useState } from 'react';
import {
  Box,
  Button,
  Typography,
  CircularProgress,
  Paper,
  Container,
  Card,
  CardContent,
  CardActions,
  Divider,
  Chip,
} from '@mui/material';
import { 
  PlayArrow as PlayIcon,
  Stop as StopIcon,
  Refresh as RefreshIcon,
  CheckCircle as CheckIcon,
  Error as ErrorIcon,
} from '@mui/icons-material';

interface TestResult {
  id: string;
  name: string;
  status: 'pending' | 'running' | 'passed' | 'failed';
  message?: string;
  duration?: number;
}

interface SubchatTestProps {
  className?: string;
}

export const SubchatTest: React.FC<SubchatTestProps> = ({ className }) => {

  const [tests, setTests] = useState<TestResult[]>([
    { id: '1', name: 'Subchat Creation', status: 'pending' },
    { id: '2', name: 'Message Handling', status: 'pending' },
    { id: '3', name: 'Status Updates', status: 'pending' },
    { id: '4', name: 'Error Handling', status: 'pending' },
    { id: '5', name: 'Cleanup Process', status: 'pending' },
  ]);
  const [isRunning, setIsRunning] = useState(false);
  const [overallStatus, setOverallStatus] = useState<'idle' | 'running' | 'completed' | 'error'>('idle');

  // Responsive typography scaling
  const getResponsiveTypography = () => ({
    h4: {
      fontSize: {
        xs: '1.5rem', // 24px on mobile
        sm: '1.75rem', // 28px on tablet
        md: '2rem', // 32px on desktop
      },
      lineHeight: {
        xs: 1.3,
        sm: 1.4,
        md: 1.5,
      },
    },
    h6: {
      fontSize: {
        xs: '1rem', // 16px on mobile
        sm: '1.125rem', // 18px on tablet
        md: '1.25rem', // 20px on desktop
      },
    },
    body1: {
      fontSize: {
        xs: '0.875rem', // 14px on mobile
        sm: '1rem', // 16px on tablet+
      },
    },
    body2: {
      fontSize: {
        xs: '0.75rem', // 12px on mobile
        sm: '0.875rem', // 14px on tablet+
      },
    },
  });

  // Responsive spacing and sizing
  const getResponsiveSpacing = () => ({
    containerPadding: {
      xs: 2, // 16px on mobile
      sm: 3, // 24px on tablet
      md: 4, // 32px on desktop
    },
    cardPadding: {
      xs: 2, // 16px on mobile
      sm: 2.5, // 20px on tablet
      md: 3, // 24px on desktop
    },
    buttonMinHeight: {
      xs: 44, // Touch-friendly 44px minimum
      sm: 40,
      md: 36,
    },
    gridSpacing: {
      xs: 2, // 16px on mobile
      sm: 2.5, // 20px on tablet
      md: 3, // 24px on desktop
    },
  });

  const responsiveStyles = getResponsiveTypography();
  const spacing = getResponsiveSpacing();

  const runTest = async (testId: string): Promise<void> => {
    return new Promise((resolve) => {
      setTests(prev => prev.map(test => 
        test.id === testId 
          ? { ...test, status: 'running' as const }
          : test
      ));

      // Simulate test execution
      setTimeout(() => {
        const success = Math.random() > 0.2; // 80% success rate
        const duration = Math.floor(Math.random() * 2000) + 500; // 500-2500ms

        setTests(prev => prev.map(test => 
          test.id === testId 
            ? { 
                ...test, 
                status: success ? 'passed' as const : 'failed' as const,
                message: success ? 'Test completed successfully' : 'Test failed with error',
                duration
              }
            : test
        ));
        resolve();
      }, Math.floor(Math.random() * 2000) + 1000);
    });
  };

  const runAllTests = async () => {
    setIsRunning(true);
    setOverallStatus('running');
    
    // Reset all tests
    setTests(prev => prev.map(test => ({ 
      ...test, 
      status: 'pending' as const, 
      message: undefined, 
      duration: undefined 
    })));

    try {
      // Run tests sequentially
      for (const test of tests) {
        await runTest(test.id);
      }
      setOverallStatus('completed');
    } catch (error) {
      setOverallStatus('error');
    } finally {
      setIsRunning(false);
    }
  };

  const resetTests = () => {
    setTests(prev => prev.map(test => ({ 
      ...test, 
      status: 'pending' as const, 
      message: undefined, 
      duration: undefined 
    })));
    setOverallStatus('idle');
    setIsRunning(false);
  };

  const getStatusIcon = (status: TestResult['status']) => {
    switch (status) {
      case 'running':
        return <CircularProgress size={20} />;
      case 'passed':
        return <CheckIcon color="success" />;
      case 'failed':
        return <ErrorIcon color="error" />;
      default:
        return null;
    }
  };

  const getStatusColor = (status: TestResult['status']) => {
    switch (status) {
      case 'running':
        return 'primary';
      case 'passed':
        return 'success';
      case 'failed':
        return 'error';
      default:
        return 'default';
    }
  };

  const passedTests = tests.filter(t => t.status === 'passed').length;
  const failedTests = tests.filter(t => t.status === 'failed').length;
  const totalTests = tests.length;

  return (
    <Container 
      maxWidth="lg" 
      className={className}
      sx={{ 
        py: spacing.containerPadding,
        px: { xs: 2, sm: 3 }
      }}
    >
      <Box sx={{ 
        backgroundColor: 'background.paper', 
        borderRadius: { xs: 1, sm: 2 }, 
        boxShadow: 1, 
        p: spacing.cardPadding 
      }}>
        {/* Header */}
        <Box
          display="flex"
          justifyContent="space-between"
          alignItems="center"
          mb={{ xs: 3, sm: 4 }}
          sx={{
            flexDirection: { xs: 'column', sm: 'row' },
            gap: { xs: 2, sm: 0 },
            alignItems: { xs: 'stretch', sm: 'center' },
          }}
        >
          <Box sx={{ textAlign: { xs: 'center', sm: 'left' } }}>
            <Typography 
              variant="h4" 
              component="h1" 
              sx={{ 
                fontWeight: 600, 
                mb: 1,
                fontSize: responsiveStyles.h4.fontSize,
                lineHeight: responsiveStyles.h4.lineHeight,
              }}
            >
              Subchat Test Suite
            </Typography>
            <Typography 
              variant="body2" 
              color="text.secondary"
              sx={{ fontSize: responsiveStyles.body2.fontSize }}
            >
              Test subchat functionality and components
            </Typography>
          </Box>
          
          <Box sx={{ 
            display: 'flex', 
            gap: 1,
            flexDirection: { xs: 'column', sm: 'row' },
            width: { xs: '100%', sm: 'auto' }
          }}>
            <Button
              variant="contained"
              startIcon={isRunning ? <StopIcon /> : <PlayIcon />}
              onClick={isRunning ? resetTests : runAllTests}
              disabled={isRunning && overallStatus === 'running'}
              sx={{
                minWidth: { xs: '100%', sm: 'auto' },
                minHeight: spacing.buttonMinHeight,
                py: { xs: 1.5, sm: 1.25 },
                px: { xs: 3, sm: 2.5, md: 3 },
                borderRadius: { xs: 1.5, sm: 2 },
                textTransform: 'none',
                fontWeight: 600,
                fontSize: { xs: '0.875rem', sm: '1rem' },
              }}
            >
              {isRunning ? 'Stop Tests' : 'Run All Tests'}
            </Button>
            
            <Button
              variant="outlined"
              startIcon={<RefreshIcon />}
              onClick={resetTests}
              disabled={isRunning}
              sx={{
                minWidth: { xs: '100%', sm: 'auto' },
                minHeight: spacing.buttonMinHeight,
                py: { xs: 1.5, sm: 1.25 },
                px: { xs: 3, sm: 2.5, md: 3 },
                borderRadius: { xs: 1.5, sm: 2 },
                textTransform: 'none',
                fontWeight: 600,
                fontSize: { xs: '0.875rem', sm: '1rem' },
              }}
            >
              Reset
            </Button>
          </Box>
        </Box>

        {/* Test Results Summary */}
        {overallStatus !== 'idle' && (
          <Paper 
            sx={{ 
              p: spacing.cardPadding, 
              mb: 3, 
              backgroundColor: 'background.default',
              borderRadius: { xs: 1, sm: 2 }
            }}
          >
            <Typography 
              variant="h6" 
              sx={{ 
                mb: 2,
                fontSize: responsiveStyles.h6.fontSize 
              }}
            >
              Test Results
            </Typography>
            <Box sx={{ 
              display: 'flex', 
              gap: 2, 
              flexWrap: 'wrap',
              justifyContent: { xs: 'center', sm: 'flex-start' }
            }}>
              <Chip 
                label={`${passedTests}/${totalTests} Passed`} 
                color="success" 
                variant={passedTests > 0 ? 'filled' : 'outlined'}
              />
              <Chip 
                label={`${failedTests} Failed`} 
                color="error" 
                variant={failedTests > 0 ? 'filled' : 'outlined'}
              />
              <Chip 
                label={`Status: ${overallStatus}`} 
                color={overallStatus === 'completed' ? 'success' : 'primary'}
              />
            </Box>
          </Paper>
        )}

        {/* Test Cases */}
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: spacing.gridSpacing }}>
          {tests.map((test, index) => (
            <Card 
              key={test.id}
              sx={{ 
                borderRadius: { xs: 1, sm: 2 },
                boxShadow: 1,
                transition: 'box-shadow 0.2s ease-in-out',
                '&:hover': {
                  boxShadow: 2,
                },
              }}
            >
              <CardContent sx={{ p: spacing.cardPadding }}>
                <Box sx={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  flexDirection: { xs: 'column', sm: 'row' },
                  gap: { xs: 2, sm: 1 },
                  alignItems: { xs: 'stretch', sm: 'center' }
                }}>
                  <Box sx={{ 
                    display: 'flex', 
                    alignItems: 'center', 
                    gap: 2,
                    width: { xs: '100%', sm: 'auto' }
                  }}>
                    <Typography 
                      variant="body1" 
                      sx={{ 
                        fontWeight: 500,
                        fontSize: responsiveStyles.body1.fontSize,
                        textAlign: { xs: 'center', sm: 'left' },
                        flex: 1
                      }}
                    >
                      {index + 1}. {test.name}
                    </Typography>
                  </Box>
                  
                  <Box sx={{ 
                    display: 'flex', 
                    alignItems: 'center', 
                    gap: 2,
                    justifyContent: { xs: 'center', sm: 'flex-end' }
                  }}>
                    {getStatusIcon(test.status)}
                    <Chip 
                      label={test.status} 
                      color={getStatusColor(test.status)}
                      size="small"
                      sx={{ minWidth: 80 }}
                    />
                  </Box>
                </Box>
                
                {test.message && (
                  <>
                    <Divider sx={{ my: 2 }} />
                    <Typography 
                      variant="body2" 
                      color="text.secondary"
                      sx={{ 
                        fontSize: responsiveStyles.body2.fontSize,
                        textAlign: { xs: 'center', sm: 'left' }
                      }}
                    >
                      {test.message}
                      {test.duration && ` (${test.duration}ms)`}
                    </Typography>
                  </>
                )}
              </CardContent>
              
              <CardActions sx={{ px: spacing.cardPadding, pb: spacing.cardPadding }}>
                <Button
                  size="small"
                  onClick={() => runTest(test.id)}
                  disabled={isRunning || test.status === 'running'}
                  sx={{
                    textTransform: 'none',
                    fontSize: responsiveStyles.body2.fontSize,
                    width: { xs: '100%', sm: 'auto' }
                  }}
                >
                  Run Test
                </Button>
              </CardActions>
            </Card>
          ))}
        </Box>
      </Box>
    </Container>
  );
};

export default SubchatTest;