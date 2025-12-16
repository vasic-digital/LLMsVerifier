import React from 'react';
import { Box, Typography, Paper } from '@mui/material';

const Notifications: React.FC = () => {
  return (
    <Box sx={{ flexGrow: 1, p: 3 }}>
      <Typography variant="h4" gutterBottom>
        Notifications
      </Typography>
      <Paper sx={{ p: 3 }}>
        <Typography variant="body1">
          View and manage system notifications. Configure notification channels
          (Slack, Email, Telegram) and monitor notification delivery status.
        </Typography>
      </Paper>
    </Box>
  );
};

export default Notifications;