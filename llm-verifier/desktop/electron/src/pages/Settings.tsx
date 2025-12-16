import React from 'react';
import { Box, Typography, Paper } from '@mui/material';

const Settings: React.FC = () => {
  return (
    <Box sx={{ flexGrow: 1, p: 3 }}>
      <Typography variant="h4" gutterBottom>
        Settings
      </Typography>
      <Paper sx={{ p: 3 }}>
        <Typography variant="body1">
          Configure LLM Verifier settings including API keys, notification channels,
          scheduling preferences, and system behavior. Manage provider configurations
          and customize verification parameters.
        </Typography>
      </Paper>
    </Box>
  );
};

export default Settings;