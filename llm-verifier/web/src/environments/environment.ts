export const environment = {
  production: false,
  // Local development API server
  apiUrl: 'http://localhost:8080',
  // WebSocket URL for real-time updates (uses relative path for same-origin)
  wsUrl: 'ws://localhost:8080',
  // Feature flags
  features: {
    websocket: true,
    darkMode: true,
    analytics: false
  }
};