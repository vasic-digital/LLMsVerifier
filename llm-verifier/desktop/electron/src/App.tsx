import React, { useState, useEffect } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { ThemeProvider, createTheme } from '@mui/material/styles';
import { CssBaseline, Box, AppBar, Toolbar, Typography, Button, IconButton, Drawer, List, ListItem, ListItemIcon, ListItemText, Divider } from '@mui/material';
import {
  Dashboard as DashboardIcon,
  Settings as SettingsIcon,
  Assessment as AssessmentIcon,
  Notifications as NotificationsIcon,
  Schedule as ScheduleIcon,
  GetApp as ExportIcon,
  Menu as MenuIcon,
  Close as CloseIcon
} from '@mui/icons-material';

// Components
import Dashboard from './pages/Dashboard';
import Models from './pages/Models';
import Verification from './pages/Verification';
import Export from './pages/Export';
import Settings from './pages/Settings';
import Notifications from './pages/Notifications';
import Scheduler from './pages/Scheduler';

// Types
interface BackendStatus {
  running: boolean;
  port: string;
  host: string;
}

// Theme
const theme = createTheme({
  palette: {
    mode: 'light',
    primary: {
      main: '#1976d2',
    },
    secondary: {
      main: '#dc004e',
    },
  },
  typography: {
    fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
  },
});

// Navigation items
const navigationItems = [
  { text: 'Dashboard', icon: <DashboardIcon />, path: '/dashboard' },
  { text: 'Models', icon: <AssessmentIcon />, path: '/models' },
  { text: 'Verification', icon: <AssessmentIcon />, path: '/verification' },
  { text: 'Export', icon: <ExportIcon />, path: '/export' },
  { text: 'Notifications', icon: <NotificationsIcon />, path: '/notifications' },
  { text: 'Scheduler', icon: <ScheduleIcon />, path: '/scheduler' },
  { text: 'Settings', icon: <SettingsIcon />, path: '/settings' },
];

function App() {
  const [backendStatus, setBackendStatus] = useState<BackendStatus | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [currentPage, setCurrentPage] = useState('Dashboard');

  // Initialize backend status
  useEffect(() => {
    checkBackendStatus();
  }, []);

  const checkBackendStatus = async () => {
    try {
      const status = await window.electronAPI.getBackendStatus();
      setBackendStatus(status);
    } catch (error) {
      console.error('Failed to get backend status:', error);
    }
  };

  const startBackend = async () => {
    try {
      await window.electronAPI.startBackend();
      setTimeout(checkBackendStatus, 2000); // Check status after 2 seconds
    } catch (error) {
      console.error('Failed to start backend:', error);
    }
  };

  const stopBackend = async () => {
    try {
      await window.electronAPI.stopBackend();
      setTimeout(checkBackendStatus, 1000);
    } catch (error) {
      console.error('Failed to stop backend:', error);
    }
  };

  const handleNavigation = (path: string, text: string) => {
    setCurrentPage(text);
    setDrawerOpen(false);
  };

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Box sx={{ display: 'flex' }}>
        {/* App Bar */}
        <AppBar position="fixed" sx={{ zIndex: theme.zIndex.drawer + 1 }}>
          <Toolbar>
            <IconButton
              color="inherit"
              aria-label="open drawer"
              edge="start"
              onClick={() => setDrawerOpen(!drawerOpen)}
              sx={{ mr: 2 }}
            >
              {drawerOpen ? <CloseIcon /> : <MenuIcon />}
            </IconButton>
            <Typography variant="h6" noWrap component="div" sx={{ flexGrow: 1 }}>
              LLM Verifier Desktop - {currentPage}
            </Typography>

            {/* Backend Status */}
            {backendStatus && (
              <Box sx={{ display: 'flex', alignItems: 'center', mr: 2 }}>
                <Typography variant="body2" sx={{ mr: 1 }}>
                  Backend: {backendStatus.running ? 'Running' : 'Stopped'}
                </Typography>
                {!backendStatus.running ? (
                  <Button
                    variant="contained"
                    color="success"
                    size="small"
                    onClick={startBackend}
                  >
                    Start
                  </Button>
                ) : (
                  <Button
                    variant="contained"
                    color="error"
                    size="small"
                    onClick={stopBackend}
                  >
                    Stop
                  </Button>
                )}
              </Box>
            )}
          </Toolbar>
        </AppBar>

        {/* Navigation Drawer */}
        <Drawer
          variant="temporary"
          open={drawerOpen}
          onClose={() => setDrawerOpen(false)}
          sx={{
            width: 240,
            flexShrink: 0,
            '& .MuiDrawer-paper': {
              width: 240,
              boxSizing: 'border-box',
            },
          }}
        >
          <Toolbar />
          <Box sx={{ overflow: 'auto' }}>
            <List>
              {navigationItems.map((item) => (
                <ListItem
                  button
                  key={item.path}
                  onClick={() => handleNavigation(item.path, item.text)}
                  component="a"
                  href={item.path}
                >
                  <ListItemIcon>
                    {item.icon}
                  </ListItemIcon>
                  <ListItemText primary={item.text} />
                </ListItem>
              ))}
            </List>
            <Divider />
          </Box>
        </Drawer>

        {/* Main Content */}
        <Box component="main" sx={{ flexGrow: 1, p: 3 }}>
          <Toolbar />
          <Routes>
            <Route path="/" element={<Navigate to="/dashboard" replace />} />
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/models" element={<Models />} />
            <Route path="/verification" element={<Verification />} />
            <Route path="/export" element={<Export />} />
            <Route path="/notifications" element={<Notifications />} />
            <Route path="/scheduler" element={<Scheduler />} />
            <Route path="/settings" element={<Settings />} />
          </Routes>
        </Box>
      </Box>
    </ThemeProvider>
  );
}

export default App;