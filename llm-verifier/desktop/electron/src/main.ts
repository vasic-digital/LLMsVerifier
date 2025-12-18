import { app, BrowserWindow, ipcMain, dialog, shell, Menu } from 'electron';
import { join } from 'path';
import { spawn } from 'child_process';
import * as fs from 'fs';
import * as path from 'path';

const isDev = process.env.NODE_ENV === 'development';

// Keep a global reference of the window object
let mainWindow: BrowserWindow | null = null;
let backendProcess: any = null;

// Application configuration
const config = {
  backendPort: process.env.BACKEND_PORT || '8080',
  backendHost: process.env.BACKEND_HOST || 'localhost',
  backendPath: process.env.BACKEND_PATH || path.join(__dirname, '..', '..', '..', 'llm-verifier'),
};

function createWindow(): void {
  // Create the browser window
  mainWindow = new BrowserWindow({
    width: 1400,
    height: 900,
    minWidth: 1000,
    minHeight: 700,
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
      enableRemoteModule: false,
      preload: join(__dirname, 'preload.js'),
    },
    icon: path.join(__dirname, 'assets', 'icon.png'),
    titleBarStyle: process.platform === 'darwin' ? 'hiddenInset' : 'default',
    show: false, // Don't show until ready-to-show
  });

  // Load the app
  const startUrl = isDev
    ? 'http://localhost:3000'
    : `file://${join(__dirname, '../renderer/index.html')}`;

  mainWindow.loadURL(startUrl);

  // Show window when ready to prevent visual flash
  mainWindow.once('ready-to-show', () => {
    if (mainWindow) {
      mainWindow.show();
    }
  });

  // Open DevTools in development
  if (isDev) {
    mainWindow.webContents.openDevTools();
  }

  // Emitted when the window is closed
  mainWindow.on('closed', () => {
    mainWindow = null;
  });

  // Handle external links
  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    shell.openExternal(url);
    return { action: 'deny' };
  });
}

// Start the backend process
function startBackend(): void {
  if (backendProcess) {
    return; // Already running
  }

  const backendExecutable = process.platform === 'win32'
    ? 'llm-verifier.exe'
    : 'llm-verifier';

  const backendFullPath = path.join(config.backendPath, backendExecutable);

  console.log('Starting backend:', backendFullPath);

  try {
    backendProcess = spawn(backendFullPath, ['api', '--port', config.backendPort], {
      cwd: config.backendPath,
      stdio: ['ignore', 'pipe', 'pipe'],
      detached: false,
    });

    backendProcess.stdout?.on('data', (data: Buffer) => {
      console.log('Backend stdout:', data.toString());
    });

    backendProcess.stderr?.on('data', (data: Buffer) => {
      console.error('Backend stderr:', data.toString());
    });

    backendProcess.on('close', (code: number) => {
      console.log(`Backend process exited with code ${code}`);
      backendProcess = null;
    });

    backendProcess.on('error', (error: Error) => {
      console.error('Failed to start backend:', error);
      backendProcess = null;
    });

  } catch (error) {
    console.error('Error starting backend:', error);
  }
}

// Stop the backend process
function stopBackend(): void {
  if (backendProcess) {
    if (process.platform === 'win32') {
      spawn('taskkill', ['/pid', backendProcess.pid.toString(), '/t', '/f']);
    } else {
      backendProcess.kill('SIGTERM');
    }
    backendProcess = null;
  }
}

// IPC handlers
function setupIpcHandlers(): void {
  // Backend management
  ipcMain.handle('start-backend', async () => {
    try {
      startBackend();
      return { success: true };
    } catch (error) {
      return { success: false, error: (error as Error).message };
    }
  });

  ipcMain.handle('stop-backend', async () => {
    try {
      stopBackend();
      return { success: true };
    } catch (error) {
      return { success: false, error: (error as Error).message };
    }
  });

  ipcMain.handle('get-backend-status', async () => {
    return {
      running: backendProcess !== null,
      port: config.backendPort,
      host: config.backendHost,
    };
  });

  // File system operations
  ipcMain.handle('select-directory', async () => {
    if (!mainWindow) return null;

    const result = await dialog.showOpenDialog(mainWindow, {
      properties: ['openDirectory'],
    });

    if (result.canceled) {
      return null;
    }

    return result.filePaths[0];
  });

  ipcMain.handle('select-file', async (event, options: any) => {
    if (!mainWindow) return null;

    const result = await dialog.showOpenDialog(mainWindow, {
      properties: options.properties || ['openFile'],
      filters: options.filters || [],
    });

    if (result.canceled) {
      return null;
    }

    return result.filePaths[0];
  });

  ipcMain.handle('save-file', async (event, options: any) => {
    if (!mainWindow) return null;

    const result = await dialog.showSaveDialog(mainWindow, {
      filters: options.filters || [],
      defaultPath: options.defaultPath,
    });

    if (result.canceled) {
      return null;
    }

    return result.filePath;
  });

  // System information
  ipcMain.handle('get-system-info', async () => {
    return {
      platform: process.platform,
      arch: process.arch,
      version: app.getVersion(),
      electron: process.versions.electron,
      node: process.versions.node,
      chrome: process.versions.chrome,
    };
  });

  // Configuration management
  ipcMain.handle('load-config', async () => {
    try {
      const configPath = path.join(app.getPath('userData'), 'config.json');
      if (fs.existsSync(configPath)) {
        const configData = fs.readFileSync(configPath, 'utf-8');
        return JSON.parse(configData);
      }
      return {};
    } catch (error) {
      console.error('Failed to load config:', error);
      return {};
    }
  });

  ipcMain.handle('save-config', async (event, configData) => {
    try {
      const configPath = path.join(app.getPath('userData'), 'config.json');
      fs.writeFileSync(configPath, JSON.stringify(configData, null, 2));
      return { success: true };
    } catch (error) {
      console.error('Failed to save config:', error);
      return { success: false, error: (error as Error).message };
    }
  });

  // Verification management
  ipcMain.handle('start-verification', async () => {
    try {
      // Send HTTP request to backend to start verification
      const response = await fetch(`http://${config.backendHost}:${config.backendPort}/api/v1/verification/start`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          models: ['all'], // Start verification for all models
          priority: 'normal',
        }),
      });
      
      if (response.ok) {
        return { success: true, data: await response.json() };
      } else {
        return { success: false, error: 'Failed to start verification' };
      }
    } catch (error) {
      console.error('Failed to start verification:', error);
      return { success: false, error: (error as Error).message };
    }
  });

  ipcMain.handle('stop-verification', async () => {
    try {
      // Send HTTP request to backend to stop verification
      const response = await fetch(`http://${config.backendHost}:${config.backendPort}/api/v1/verification/stop`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      });
      
      if (response.ok) {
        return { success: true, data: await response.json() };
      } else {
        return { success: false, error: 'Failed to stop verification' };
      }
    } catch (error) {
      console.error('Failed to stop verification:', error);
      return { success: false, error: (error as Error).message };
    }
  });
}

// Create application menu
function createMenu(): void {
  const template: any[] = [
    {
      label: 'File',
      submenu: [
        {
          label: 'New Verification',
          accelerator: 'CmdOrCtrl+N',
          click: () => {
            mainWindow?.webContents.send('menu-action', 'new-verification');
          },
        },
        { type: 'separator' },
        {
          label: 'Import Configuration',
          click: () => {
            mainWindow?.webContents.send('menu-action', 'import-config');
          },
        },
        {
          label: 'Export Results',
          click: () => {
            mainWindow?.webContents.send('menu-action', 'export-results');
          },
        },
        { type: 'separator' },
        {
          label: 'Quit',
          accelerator: process.platform === 'darwin' ? 'Cmd+Q' : 'Ctrl+Q',
          click: () => {
            app.quit();
          },
        },
      ],
    },
    {
      label: 'Edit',
      submenu: [
        { role: 'undo' },
        { role: 'redo' },
        { type: 'separator' },
        { role: 'cut' },
        { role: 'copy' },
        { role: 'paste' },
        { role: 'selectall' },
      ],
    },
    {
      label: 'View',
      submenu: [
        { role: 'reload' },
        { role: 'forcereload' },
        { role: 'toggledevtools' },
        { type: 'separator' },
        { role: 'resetzoom' },
        { role: 'zoomin' },
        { role: 'zoomout' },
        { type: 'separator' },
        { role: 'togglefullscreen' },
      ],
    },
    {
      label: 'Window',
      submenu: [
        { role: 'minimize' },
        { role: 'close' },
      ],
    },
    {
      label: 'Help',
      submenu: [
        {
          label: 'Documentation',
          click: () => {
            shell.openExternal('https://llm-verifier.ai/docs');
          },
        },
        {
          label: 'Report Issue',
          click: () => {
            shell.openExternal('https://github.com/llm-verifier/llm-verifier/issues');
          },
        },
        { type: 'separator' },
        {
          label: 'About LLM Verifier',
          click: () => {
            mainWindow?.webContents.send('menu-action', 'show-about');
          },
        },
      ],
    },
  ];

  // macOS specific menu adjustments
  if (process.platform === 'darwin') {
    template.unshift({
      label: app.getName(),
      submenu: [
        { role: 'about' },
        { type: 'separator' },
        { role: 'services', submenu: [] },
        { type: 'separator' },
        { role: 'hide' },
        { role: 'hideothers' },
        { role: 'unhide' },
        { type: 'separator' },
        { role: 'quit' },
      ],
    });

    // Window menu
    template[4].submenu = [
      { role: 'close' },
      { role: 'minimize' },
      { role: 'zoom' },
      { type: 'separator' },
      { role: 'front' },
    ];
  }

  const menu = Menu.buildFromTemplate(template);
  Menu.setApplicationMenu(menu);
}

// App event handlers
app.whenReady().then(() => {
  createMenu();
  setupIpcHandlers();
  createWindow();

  // On macOS, re-create window when dock icon is clicked
  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});

// Quit when all windows are closed, except on macOS
app.on('window-all-closed', () => {
  stopBackend();
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

// Handle app shutdown
app.on('before-quit', () => {
  stopBackend();
});

// Security: Prevent navigation to external websites
app.on('web-contents-created', (event, contents) => {
  contents.on('will-navigate', (event, navigationUrl) => {
    const parsedUrl = new URL(navigationUrl);

    if (parsedUrl.origin !== 'http://localhost:3000' && parsedUrl.origin !== 'file://') {
      event.preventDefault();
    }
  });
});