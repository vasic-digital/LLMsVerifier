/**
 * LLM Verifier Models
 * Central module for all TypeScript interfaces and types
 */

// Model represents an LLM model in the system
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
  verification_status: VerificationStatus;
  created_at: string;
  updated_at: string;
}

// Provider represents an LLM provider (e.g., OpenAI, Anthropic)
export interface Provider {
  id: number;
  name: string;
  endpoint: string;
  description?: string;
  status: ProviderStatus;
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

// VerificationResult represents the outcome of a model verification
export interface VerificationResult {
  id: number;
  model_id: number;
  status: VerificationRunStatus;
  overall_score: number;
  code_capability_score: number;
  responsiveness_score: number;
  reliability_score: number;
  feature_richness_score: number;
  value_proposition_score: number;
  started_at: string;
  completed_at?: string;
  error_message?: string;
  details?: VerificationDetails;
}

// VerificationDetails contains detailed verification information
export interface VerificationDetails {
  tests_passed: number;
  tests_failed: number;
  tests_skipped: number;
  test_results?: TestResult[];
}

// TestResult represents a single test outcome
export interface TestResult {
  name: string;
  passed: boolean;
  duration_ms: number;
  message?: string;
}

// HealthStatus represents the system health state
export interface HealthStatus {
  status: 'healthy' | 'degraded' | 'unhealthy';
  timestamp: string;
  uptime: string;
  version: string;
  services?: Record<string, ServiceStatus>;
}

// ServiceStatus represents the status of an individual service
export interface ServiceStatus {
  status: 'up' | 'down' | 'degraded';
  latency_ms?: number;
  last_check?: string;
}

// SystemInfo provides system-level information
export interface SystemInfo {
  version: string;
  go_version: string;
  build_time: string;
  database_size: number;
  models_count: number;
  providers_count: number;
  uptime: string;
}

// DashboardStats aggregates statistics for the dashboard
export interface DashboardStats {
  totalModels: number;
  totalProviders: number;
  verifiedModels: number;
  pendingModels: number;
  averageScore: number;
  lastVerification: Date;
  recentActivity: ActivityItem[];
}

// ActivityItem represents a recent activity entry
export interface ActivityItem {
  id: string;
  type: ActivityType;
  message: string;
  timestamp: Date;
  details?: Record<string, unknown>;
}

// Type definitions for status values
export type VerificationStatus = 'verified' | 'unverified' | 'pending' | 'failed';
export type ProviderStatus = 'active' | 'inactive' | 'maintenance';
export type VerificationRunStatus = 'pending' | 'running' | 'completed' | 'failed';
export type ActivityType = 'verification' | 'error' | 'success' | 'info' | 'warning';

// API Response types
export interface ApiResponse<T> {
  data: T;
  message?: string;
  status: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  limit: number;
  offset: number;
  has_more: boolean;
}

// Request types
export interface CreateModelRequest {
  provider_id: number;
  model_id: string;
  name: string;
  description?: string;
  architecture?: string;
  parameter_count?: number;
  context_window_tokens?: number;
  max_output_tokens?: number;
}

export interface UpdateModelRequest {
  name?: string;
  description?: string;
  architecture?: string;
  parameter_count?: number;
  context_window_tokens?: number;
  max_output_tokens?: number;
}

export interface CreateProviderRequest {
  name: string;
  endpoint: string;
  description?: string;
  api_key?: string;
  website?: string;
  support_email?: string;
  documentation_url?: string;
}

export interface UpdateProviderRequest {
  name?: string;
  endpoint?: string;
  description?: string;
  api_key?: string;
  website?: string;
  support_email?: string;
  documentation_url?: string;
  is_active?: boolean;
}

// Authentication types
export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  token_type: string;
}

export interface RefreshTokenRequest {
  refresh_token: string;
}

// WebSocket event types
export interface WebSocketEvent {
  type: WebSocketEventType;
  payload: unknown;
  timestamp: string;
}

export type WebSocketEventType =
  | 'model.verified'
  | 'model.verification.started'
  | 'model.verification.failed'
  | 'verification.completed'
  | 'verification.progress'
  | 'provider.added'
  | 'provider.removed'
  | 'provider.updated'
  | 'system.health.changed'
  | 'system.error';
