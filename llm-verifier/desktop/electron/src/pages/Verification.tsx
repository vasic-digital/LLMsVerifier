import React from 'react';
import { Box, Typography, Paper, Button } from '@mui/material';
import { PlayArrow as PlayIcon } from '@mui/icons-material';

const Verification: React.FC = () => {
  return (
    <Box sx={{ flexGrow: 1, p: 3 }}>
      <Typography variant="h4" gutterBottom>
        Model Verification
      </Typography>
      <Paper sx={{ p: 3 }}>
        <Typography variant="body1" sx={{ mb: 2 }}>
          Start model verification to test and benchmark LLM capabilities.
          Verification includes code generation, reasoning, multimodal features, and performance metrics.
        </Typography>
        <Button variant="contained" startIcon={<PlayIcon />} size="large">
          Start Verification
        </Button>
      </Paper>
    </Box>
  );
};

export default Verification;