export const environment = {
  production: true,
  // API URL is configured at build time or runtime via environment variables
  // For Docker/Podman deployments, this defaults to the container's backend service
  // Override with LLM_VERIFIER_API_URL environment variable at build time
  apiUrl: (typeof window !== 'undefined' && (window as any).__env?.apiUrl) || '/api',
  // WebSocket URL for real-time updates
  wsUrl: (typeof window !== 'undefined' && (window as any).__env?.wsUrl) || '',
  // Feature flags
  features: {
    websocket: true,
    darkMode: true,
    analytics: true
  }
};