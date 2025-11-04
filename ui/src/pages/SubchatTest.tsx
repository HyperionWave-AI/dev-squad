import { Box, Button, Typography, Stack, Container } from '@mui/material';

export default function SubchatTest() {
  const handleButton1Click = () => {
    console.log('Button 1 clicked');
  };

  const handleButton2Click = () => {
    console.log('Button 2 clicked');
  };

  const handleButton3Click = () => {
    console.log('Button 3 clicked');
  };

  return (
    <Container maxWidth="md" sx={{ py: 6 }}>
      <Box sx={{ maxWidth: 600, mx: 'auto' }}>
        <Typography variant="h4" component="h1" fontWeight="bold" textAlign="center" mb={8}>
          Subchat Test Page
        </Typography>

        <Stack spacing={4} alignItems="center">
          <Button
            onClick={handleButton1Click}
            variant="contained"
            sx={{ width: 192, height: 48 }}
          >
            Test Button 1
          </Button>

          <Button
            onClick={handleButton2Click}
            variant="outlined"
            sx={{ width: 192, height: 48 }}
          >
            Test Button 2
          </Button>

          <Button
            onClick={handleButton3Click}
            variant="contained"
            color="secondary"
            sx={{ width: 192, height: 48 }}
          >
            Test Button 3
          </Button>
        </Stack>
      </Box>
    </Container>
  );
}
