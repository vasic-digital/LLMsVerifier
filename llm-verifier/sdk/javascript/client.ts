/**
 * LLM Verifier JavaScript/TypeScript SDK
 * A client library for interacting with the LLM Verifier REST API
 */

interface AuthResponse {
  token: string;
  expires_at: string;
  user: User;
}

interface User {
  id: number;
  username: string;
  email: string;
  role: string;
}

interface Model {
  id: number;
  provider_id: number;
  model_id: string;
  name: string;
  description?: string;
  architecture?: string;
  parameter_count?: number;
  context_window_tokens?: number;
  max_output_tokens?: number;
  overall_score?: number;
  verification_status: string;
  created_at: string;
  updated_at: string;
}

interface VerificationResult {
  id: number;
  model_id: number;
  status: string;
  overall_score: number;
  code_capability_score: number;
  responsiveness_score: number;
  reliability_score: number;
  feature_richness_score: number;
  value_proposition_score: number;
  started_at: string;
  completed_at?: string;
}

interface Provider {
  id: number;
  name: string;
  endpoint: string;
  description?: string;
  status: string;
  created_at: string;
}

interface HealthStatus {
  status: string;
  timestamp: string;
  uptime: string;
  version: string;
  services?: Record<string, string>;
}

interface SystemInfo {
  version: string;
  go_version: string;
  build_time: string;
  database_size: number;
  models_count: number;
  providers_count: number;
  uptime: string;
}

export class LLMVerifierClient {
  private baseURL: string;
  private apiKey?: string;

  constructor(baseURL: string, apiKey?: string) {
    this.baseURL = baseURL.replace(/\/$/, ''); // Remove trailing slash
    this.apiKey = apiKey;
  }

  /**
   * Authenticate user and get JWT token
   */
  async login(username: string, password: string): Promise<AuthResponse> {
    const response = await this.post('/auth/login', {
      username,
      password
    });

    this.apiKey = response.token;
    return response;
  }

  /**
   * Refresh JWT token
   */
  async refreshToken(refreshToken: string): Promise<AuthResponse> {
    const response = await this.post('/auth/refresh', {
      refresh_token: refreshToken
    });

    this.apiKey = response.token;
    return response;
  }

  /**
   * Get all models with optional filtering
   */
  async getModels(options: {
    limit?: number;
    offset?: number;
    provider?: string;
  } = {}): Promise<Model[]> {
    const params = new URLSearchParams();
    if (options.limit) params.append('limit', options.limit.toString());
    if (options.offset) params.append('offset', options.offset.toString());
    if (options.provider) params.append('provider', options.provider);

    return this.get(`/api/v1/models?${params.toString()}`);
  }

  /**
   * Get a specific model by ID
   */
  async getModel(id: number): Promise<Model> {
    return this.get(`/api/v1/models/${id}`);
  }

  /**
   * Create a new model (admin only)
   */
  async createModel(model: {
    model_id: string;
    name: string;
    provider_id: number;
    description?: string;
    architecture?: string;
    parameter_count?: number;
    context_window_tokens?: number;
    max_output_tokens?: number;
  }): Promise<Model> {
    return this.post('/api/v1/models', model);
  }

  /**
   * Update an existing model (admin only)
   */
  async updateModel(id: number, updates: Partial<Model>): Promise<Model> {
    return this.put(`/api/v1/models/${id}`, updates);
  }

  /**
   * Delete a model (admin only)
   */
  async deleteModel(id: number): Promise<void> {
    return this.delete(`/api/v1/models/${id}`);
  }

  /**
   * Trigger verification for a specific model
   */
  async verifyModel(modelId: string): Promise<VerificationResult> {
    return this.post(`/api/v1/models/${modelId}/verify`, {
      model_id: modelId
    });
  }

  /**
   * Get verification results
   */
  async getVerificationResults(options: {
    limit?: number;
    offset?: number;
  } = {}): Promise<VerificationResult[]> {
    const params = new URLSearchParams();
    if (options.limit) params.append('limit', options.limit.toString());
    if (options.offset) params.append('offset', options.offset.toString());

    return this.get(`/api/v1/verification-results?${params.toString()}`);
  }

  /**
   * Get a specific verification result
   */
  async getVerificationResult(id: number): Promise<VerificationResult> {
    return this.get(`/api/v1/verification-results/${id}`);
  }

  /**
   * Get all providers
   */
  async getProviders(): Promise<Provider[]> {
    return this.get('/api/v1/providers');
  }

  /**
   * Get a specific provider
   */
  async getProvider(id: number): Promise<Provider> {
    return this.get(`/api/v1/providers/${id}`);
  }

  /**
   * Get system health status
   */
  async getHealth(): Promise<HealthStatus> {
    return this.get('/health');
  }

  /**
   * Get detailed health status
   */
  async getHealthDetailed(): Promise<any> {
    return this.get('/health/detailed');
  }

  /**
   * Get system information
   */
  async getSystemInfo(): Promise<SystemInfo> {
    return this.get('/api/v1/system/info');
  }

  /**
   * Get database statistics
   */
  async getDatabaseStats(): Promise<any> {
    return this.get('/api/v1/system/database-stats');
  }

  // HTTP helper methods

  private async get<T>(endpoint: string): Promise<T> {
    const response = await fetch(`${this.baseURL}${endpoint}`, {
      method: 'GET',
      headers: this.getHeaders()
    });

    return this.handleResponse<T>(response);
  }

  private async post<T>(endpoint: string, data: any): Promise<T> {
    const response = await fetch(`${this.baseURL}${endpoint}`, {
      method: 'POST',
      headers: this.getHeaders(),
      body: JSON.stringify(data)
    });

    return this.handleResponse<T>(response);
  }

  private async put<T>(endpoint: string, data: any): Promise<T> {
    const response = await fetch(`${this.baseURL}${endpoint}`, {
      method: 'PUT',
      headers: this.getHeaders(),
      body: JSON.stringify(data)
    });

    return this.handleResponse<T>(response);
  }

  private async delete(endpoint: string): Promise<void> {
    const response = await fetch(`${this.baseURL}${endpoint}`, {
      method: 'DELETE',
      headers: this.getHeaders()
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }
  }

  private getHeaders(): Record<string, string> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json'
    };

    if (this.apiKey) {
      headers['Authorization'] = `Bearer ${this.apiKey}`;
    }

    return headers;
  }

  private async handleResponse<T>(response: Response): Promise<T> {
    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`HTTP ${response.status}: ${errorText}`);
    }

    return response.json();
  }
}

// Example usage:
//
// const client = new LLMVerifierClient('http://localhost:8080');
//
// try {
//   const auth = await client.login('admin', 'password');
//   console.log('Logged in as:', auth.user.username);
//
//   const models = await client.getModels({ limit: 10 });
//   console.log(`Found ${models.length} models`);
//
//   const health = await client.getHealth();
//   console.log('System status:', health.status);
// } catch (error) {
//   console.error('Error:', error);
// }

export default LLMVerifierClient;