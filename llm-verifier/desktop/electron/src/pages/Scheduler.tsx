import React from 'react';
import { Box, Typography, Paper } from '@mui/material';

const Scheduler: React.FC = () => {
  return (
    <Box sx={{ flexGrow: 1, p: 3 }}>
      <Typography variant="h4" gutterBottom>
        Job Scheduler
      </Typography>
      <Paper sx={{ p: 3 }}>
        <Typography variant="body1">
          Manage scheduled jobs for automated verification, exports, and maintenance tasks.
          Configure cron schedules and monitor job execution history.
        </Typography>
      </Paper>
    </Box>
  );
};

export default Scheduler;