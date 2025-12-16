import React from 'react';
import { Box, Typography, Paper } from '@mui/material';

const Models: React.FC = () => {
  return (
    <Box sx={{ flexGrow: 1, p: 3 }}>
      <Typography variant="h4" gutterBottom>
        Model Management
      </Typography>
      <Paper sx={{ p: 3 }}>
        <Typography variant="body1">
          Model management interface will be implemented here.
          This will show all discovered models, their verification status, scores, and allow manual re-verification.
        </Typography>
      </Paper>
    </Box>
  );
};

export default Models;