import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, BehaviorSubject } from 'rxjs';
import { environment } from '../environments/environment';

export interface Model {
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

export interface Provider {
  id: number;
  name: string;
  endpoint: string;
  description?: string;
  status: string;
  created_at: string;
  updated_at?: string;
  api_key_encrypted?: string;
  website?: string;
  support_email?: string;
  documentation_url?: string;
  is_active?: boolean;
  reliability_score?: number;
  average_response_time_ms?: number;
}

export interface VerificationResult {
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

export interface HealthStatus {
  status: string;
  timestamp: string;
  uptime: string;
  version: string;
  services?: Record<string, string>;
}

export interface SystemInfo {
  version: string;
  go_version: string;
  build_time: string;
  database_size: number;
  models_count: number;
  providers_count: number;
  uptime: string;
}

export interface DashboardStats {
  totalModels: number;
  totalProviders: number;
  verifiedModels: number;
  pendingModels: number;
  averageScore: number;
  lastVerification: Date;
  recentActivity: ActivityItem[];
}

export interface ActivityItem {
  id: string;
  type: 'verification' | 'error' | 'success';
  message: string;
  timestamp: Date;
  details?: any;
}

@Injectable({
  providedIn: 'root'
})
export class ApiService {
  private baseUrl = environment.apiUrl;
  private dashboardStatsSubject = new BehaviorSubject<DashboardStats | null>(null);
  public dashboardStats$ = this.dashboardStatsSubject.asObservable();

  constructor(private http: HttpClient) {}

  // Authentication
  login(username: string, password: string): Observable<any> {
    return this.http.post(`${this.baseUrl}/auth/login`, { username, password });
  }

  refreshToken(refreshToken: string): Observable<any> {
    return this.http.post(`${this.baseUrl}/auth/refresh`, { refresh_token: refreshToken });
  }

  // Models
  getModels(limit?: number, offset?: number, provider?: string): Observable<Model[]> {
    let params = new HttpParams();
    if (limit) params = params.set('limit', limit.toString());
    if (offset) params = params.set('offset', offset.toString());
    if (provider) params = params.set('provider', provider);

    return this.http.get<Model[]>(`${this.baseUrl}/api/v1/models`, { params });
  }

  getModel(id: number): Observable<Model> {
    return this.http.get<Model>(`${this.baseUrl}/api/v1/models/${id}`);
  }

  createModel(model: Partial<Model>): Observable<Model> {
    return this.http.post<Model>(`${this.baseUrl}/api/v1/models`, model);
  }

  updateModel(id: number, updates: Partial<Model>): Observable<Model> {
    return this.http.put<Model>(`${this.baseUrl}/api/v1/models/${id}`, updates);
  }

  deleteModel(id: number): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/api/v1/models/${id}`);
  }

  deleteProvider(id: number): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/api/v1/providers/${id}`);
  }

  deleteVerificationResult(id: number): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/api/v1/verification-results/${id}`);
  }

  verifyModel(modelId: string): Observable<VerificationResult> {
    return this.http.post<VerificationResult>(`${this.baseUrl}/api/v1/models/${modelId}/verify`, {
      model_id: modelId
    });
  }

  // Providers
  getProviders(): Observable<Provider[]> {
    return this.http.get<Provider[]>(`${this.baseUrl}/api/v1/providers`);
  }

  getProvider(id: number): Observable<Provider> {
    return this.http.get<Provider>(`${this.baseUrl}/api/v1/providers/${id}`);
  }

  // Verification Results
  getVerificationResults(limit?: number, offset?: number): Observable<VerificationResult[]> {
    let params = new HttpParams();
    if (limit) params = params.set('limit', limit.toString());
    if (offset) params = params.set('offset', offset.toString());

    return this.http.get<VerificationResult[]>(`${this.baseUrl}/api/v1/verification-results`, { params });
  }

  // System
  getHealth(): Observable<HealthStatus> {
    return this.http.get<HealthStatus>(`${this.baseUrl}/health`);
  }

  getSystemInfo(): Observable<SystemInfo> {
    return this.http.get<SystemInfo>(`${this.baseUrl}/api/v1/system/info`);
  }

  getDatabaseStats(): Observable<any> {
    return this.http.get(`${this.baseUrl}/api/v1/system/database-stats`);
  }

  // Dashboard data aggregation
  getDashboardStats(): Observable<DashboardStats> {
    // In a real implementation, this might be a dedicated dashboard endpoint
    // For now, we'll aggregate data from multiple endpoints
    return new Observable(observer => {
      const stats: DashboardStats = {
        totalModels: 0,
        totalProviders: 0,
        verifiedModels: 0,
        pendingModels: 0,
        averageScore: 0,
        lastVerification: new Date(),
        recentActivity: []
      };

      // Get models and providers in parallel
      const models$ = this.getModels();
      const providers$ = this.getProviders();
      const results$ = this.getVerificationResults(50);

      let completed = 0;
      const total = 3;

      const checkComplete = () => {
        completed++;
        if (completed === total) {
          // Calculate derived stats
          stats.pendingModels = stats.totalModels - stats.verifiedModels;
          if (stats.verifiedModels > 0) {
            // This would need to be calculated from actual verification results
            stats.averageScore = 87.5; // Mock for now
          }

          observer.next(stats);
          observer.complete();
        }
      };

      models$.subscribe({
        next: (models) => {
          stats.totalModels = models.length;
          stats.verifiedModels = models.filter(m => m.verification_status === 'verified').length;
          checkComplete();
        },
        error: (err) => {
          console.error('Failed to load models:', err);
          checkComplete();
        }
      });

      providers$.subscribe({
        next: (providers) => {
          stats.totalProviders = providers.length;
          checkComplete();
        },
        error: (err) => {
          console.error('Failed to load providers:', err);
          checkComplete();
        }
      });

      results$.subscribe({
        next: (results) => {
          if (results.length > 0) {
            // Create activity items from recent results
            stats.recentActivity = results.slice(0, 5).map(result => ({
              id: result.id.toString(),
              type: result.status === 'completed' ? 'success' as const : 'verification' as const,
              message: `Model verification ${result.status}`,
              timestamp: new Date(result.started_at),
              details: {
                score: result.overall_score,
                duration: result.completed_at ?
                  `${Math.round((new Date(result.completed_at).getTime() - new Date(result.started_at).getTime()) / 1000)}s` :
                  'In progress'
              }
            }));
          }
          checkComplete();
        },
        error: (err) => {
          console.error('Failed to load verification results:', err);
          checkComplete();
        }
      });
    });
  }

  // Update dashboard stats
  updateDashboardStats(stats: DashboardStats): void {
    this.dashboardStatsSubject.next(stats);
  }
}