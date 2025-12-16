// Electron API types
export interface ElectronAPI {
  // Backend management
  startBackend: () => Promise<{ success: boolean; error?: string }>;
  stopBackend: () => Promise<{ success: boolean; error?: string }>;
  getBackendStatus: () => Promise<{
    running: boolean;
    port: string;
    host: string;
  }>;

  // File system operations
  selectDirectory: () => Promise<string | null>;
  selectFile: (options: any) => Promise<string | null>;
  saveFile: (options: any) => Promise<string | null>;

  // System information
  getSystemInfo: () => Promise<{
    platform: string;
    arch: string;
    version: string;
    electron: string;
    node: string;
    chrome: string;
  }>;

  // Configuration management
  loadConfig: () => Promise<any>;
  saveConfig: (config: any) => Promise<{ success: boolean; error?: string }>;

  // Menu actions
  onMenuAction: (callback: (action: string) => void) => void;
  removeAllListeners: (event: string) => void;
}

// Extend Window interface to include our Electron API
declare global {
  interface Window {
    electronAPI: ElectronAPI;
    dev?: {
      platform: string;
      versions: any;
    };
  }
}

// Model types
export interface Model {
  id: string;
  name: string;
  provider: string;
  score: number;
  status: string;
  lastVerified: string;
  capabilities: string[];
}

// Verification result types
export interface VerificationResult {
  id: string;
  modelId: string;
  status: 'running' | 'completed' | 'failed';
  score: number;
  startedAt: string;
  completedAt?: string;
  error?: string;
}

// Notification types
export interface Notification {
  id: string;
  type: string;
  channel: string;
  priority: string;
  title: string;
  message: string;
  timestamp: string;
  sent: boolean;
}

// Schedule types
export interface ScheduledJob {
  id: string;
  name: string;
  type: string;
  schedule: string;
  enabled: boolean;
  lastRun?: string;
  nextRun?: string;
  status: string;
}

// Export types
export interface ExportOptions {
  format: string;
  top?: number;
  minScore?: number;
  categories?: string[];
  providers?: string[];
  models?: string[];
  includeAPIKey?: boolean;
}