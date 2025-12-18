import React, { useState, useEffect } from 'react';
import {
  Box,
  Grid,
  Card,
  CardContent,
  Typography,
  Button,
  Chip,
  LinearProgress,
  Alert,
  Paper,
} from '@mui/material';
import {
  PlayArrow as PlayIcon,
  Stop as StopIcon,
  Assessment as AssessmentIcon,
  CheckCircle as SuccessIcon,
  Error as ErrorIcon,
  Schedule as ScheduleIcon,
} from '@mui/icons-material';

interface BackendStatus {
  running: boolean;
  port: string;
  host: string;
}

interface SystemStats {
  totalModels: number;
  verifiedModels: number;
  failedModels: number;
  activeJobs: number;
  pendingNotifications: number;
}

const Dashboard: React.FC = () => {
  const [backendStatus, setBackendStatus] = useState<BackendStatus | null>(null);
  const [systemStats, setSystemStats] = useState<SystemStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadDashboardData();
  }, []);

  const loadDashboardData = async () => {
    try {
      // Get backend status
      const status = await window.electronAPI.getBackendStatus();
      setBackendStatus(status);

      // Load mock system stats for now
      setSystemStats({
        totalModels: 15,
        verifiedModels: 12,
        failedModels: 1,
        activeJobs: 2,
        pendingNotifications: 3,
      });
    } catch (error) {
      console.error('Failed to load dashboard data:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleStartVerification = async () => {
    try {
      await window.electronAPI.startVerification();
      setSystemStats(prev => prev ? { ...prev, activeJobs: prev.activeJobs + 1 } : null);
      setTimeout(loadDashboardData, 1000);
      console.log('Verification started successfully');
    } catch (error) {
      console.error('Failed to start verification:', error);
    }
  };

  const handleStopVerification = async () => {
    try {
      await window.electronAPI.stopVerification();
      setSystemStats(prev => prev ? { ...prev, activeJobs: Math.max(0, prev.activeJobs - 1) } : null);
      setTimeout(loadDashboardData, 1000);
      console.log('Verification stopped successfully');
    } catch (error) {
      console.error('Failed to stop verification:', error);
    }
  };

  const handleStartBackend = async () => {
    try {
      await window.electronAPI.startBackend();
      setTimeout(loadDashboardData, 2000);
    } catch (error) {
      console.error('Failed to start backend:', error);
    }
  };

  const handleStopBackend = async () => {
    try {
      await window.electronAPI.stopBackend();
      setTimeout(loadDashboardData, 1000);
    } catch (error) {
      console.error('Failed to stop backend:', error);
    }
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <LinearProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ flexGrow: 1, p: 3 }}>
      <Typography variant="h4" gutterBottom>
        LLM Verifier Dashboard
      </Typography>

      {/* Backend Status Alert */}
      {!backendStatus?.running && (
        <Alert
          severity="warning"
          action={
            <Button color="inherit" size="small" onClick={handleStartBackend}>
              Start Backend
            </Button>
          }
          sx={{ mb: 3 }}
        >
          Backend is not running. Start the backend to enable full functionality.
        </Alert>
      )}

      {/* Stats Cards */}
      <Grid container spacing={3} sx={{ mb: 3 }}>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="textSecondary" gutterBottom>
                Total Models
              </Typography>
              <Typography variant="h4">
                {systemStats?.totalModels || 0}
              </Typography>
              <AssessmentIcon color="primary" />
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="textSecondary" gutterBottom>
                Verified Models
              </Typography>
              <Typography variant="h4" color="success.main">
                {systemStats?.verifiedModels || 0}
              </Typography>
              <SuccessIcon color="success" />
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="textSecondary" gutterBottom>
                Failed Models
              </Typography>
              <Typography variant="h4" color="error.main">
                {systemStats?.failedModels || 0}
              </Typography>
              <ErrorIcon color="error" />
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="textSecondary" gutterBottom>
                Active Jobs
              </Typography>
              <Typography variant="h4" color="info.main">
                {systemStats?.activeJobs || 0}
              </Typography>
              <ScheduleIcon color="info" />
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Quick Actions */}
      <Paper sx={{ p: 3, mb: 3 }}>
        <Typography variant="h6" gutterBottom>
          Quick Actions
        </Typography>
        <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap' }}>
          <Button
            variant="contained"
            startIcon={<PlayIcon />}
            onClick={handleStartVerification}
            disabled={!backendStatus?.running}
          >
            Start Verification
          </Button>
          <Button
            variant="outlined"
            startIcon={<StopIcon />}
            onClick={handleStopVerification}
            disabled={!backendStatus?.running}
          >
            Stop Verification
          </Button>
          <Button variant="outlined">
            Export Results
          </Button>
          <Button variant="outlined">
            View Reports
          </Button>
        </Box>
      </Paper>

      {/* System Status */}
      <Grid container spacing={3}>
        <Grid item xs={12} md={6}>
          <Paper sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom>
              Backend Status
            </Typography>
            <Box sx={{ mb: 2 }}>
              <Typography variant="body2" color="textSecondary">
                Status: <Chip
                  label={backendStatus?.running ? 'Running' : 'Stopped'}
                  color={backendStatus?.running ? 'success' : 'error'}
                  size="small"
                />
              </Typography>
              {backendStatus?.running && (
                <>
                  <Typography variant="body2" color="textSecondary">
                    Host: {backendStatus.host}:{backendStatus.port}
                  </Typography>
                  <Typography variant="body2" color="textSecondary">
                    API: http://{backendStatus.host}:{backendStatus.port}/api/v1
                  </Typography>
                </>
              )}
            </Box>
            <Box sx={{ display: 'flex', gap: 1 }}>
              {!backendStatus?.running ? (
                <Button
                  variant="contained"
                  color="success"
                  size="small"
                  onClick={handleStartBackend}
                >
                  Start Backend
                </Button>
              ) : (
                <Button
                  variant="contained"
                  color="error"
                  size="small"
                  onClick={handleStopBackend}
                >
                  Stop Backend
                </Button>
              )}
            </Box>
          </Paper>
        </Grid>

        <Grid item xs={12} md={6}>
          <Paper sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom>
              Recent Activity
            </Typography>
            <Box sx={{ maxHeight: 200, overflowY: 'auto' }}>
              <Typography variant="body2" color="textSecondary" sx={{ mb: 1 }}>
                • Verification completed for GPT-4-Turbo (Score: 92.5)
              </Typography>
              <Typography variant="body2" color="textSecondary" sx={{ mb: 1 }}>
                • Scheduled export job completed
              </Typography>
              <Typography variant="body2" color="textSecondary" sx={{ mb: 1 }}>
                • New model Claude-3.5-Sonnet detected
              </Typography>
              <Typography variant="body2" color="textSecondary" sx={{ mb: 1 }}>
                • Score changed notification sent
              </Typography>
            </Box>
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
};

export default Dashboard;