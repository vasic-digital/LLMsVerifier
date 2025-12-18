import { contextBridge, ipcRenderer } from 'electron';

// Expose protected methods that allow the renderer process to use
// the ipcRenderer without exposing the entire object
contextBridge.exposeInMainWorld('electronAPI', {
  // Backend management
  startBackend: () => ipcRenderer.invoke('start-backend'),
  stopBackend: () => ipcRenderer.invoke('stop-backend'),
  getBackendStatus: () => ipcRenderer.invoke('get-backend-status'),

  // File system operations
  selectDirectory: () => ipcRenderer.invoke('select-directory'),
  selectFile: (options: any) => ipcRenderer.invoke('select-file', options),
  saveFile: (options: any) => ipcRenderer.invoke('save-file', options),

  // System information
  getSystemInfo: () => ipcRenderer.invoke('get-system-info'),

  // Configuration management
  loadConfig: () => ipcRenderer.invoke('load-config'),
  saveConfig: (config: any) => ipcRenderer.invoke('save-config', config),

  // Verification management
  startVerification: () => ipcRenderer.invoke('start-verification'),
  stopVerification: () => ipcRenderer.invoke('stop-verification'),

  // Menu actions
  onMenuAction: (callback: (action: string) => void) => {
    ipcRenderer.on('menu-action', (event, action) => callback(action));
  },

  // Remove listeners
  removeAllListeners: (event: string) => {
    ipcRenderer.removeAllListeners(event);
  },
});

// Also expose a simple API for development
if (process.env.NODE_ENV === 'development') {
  contextBridge.exposeInMainWorld('dev', {
    platform: process.platform,
    versions: process.versions,
  });
}