import React from 'react';
import { Box, Typography, Paper } from '@mui/material';

const Export: React.FC = () => {
  return (
    <Box sx={{ flexGrow: 1, p: 3 }}>
      <Typography variant="h4" gutterBottom>
        Export Configurations
      </Typography>
      <Paper sx={{ p: 3 }}>
        <Typography variant="body1">
          Export model configurations for AI CLI tools like OpenCode, Crush, and Claude Code.
          Select models, apply filters, and generate optimized configurations.
        </Typography>
      </Paper>
    </Box>
  );
};

export default Export;