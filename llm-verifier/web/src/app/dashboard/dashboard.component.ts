import { Component, OnInit, OnDestroy } from '@angular/core';
import { Observable, Subscription } from 'rxjs';
import { ApiService, DashboardStats, Model, Provider, VerificationResult } from '../api.service';
import { WebSocketService, RealtimeEvent } from '../websocket.service';

@Component({
  selector: 'app-dashboard',
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.scss']
})
export class DashboardComponent implements OnInit, OnDestroy {
  title = 'LLM Verifier Dashboard';
  dashboardStats$: Observable<DashboardStats>;
  models$: Observable<Model[]>;
  providers$: Observable<Provider[]>;
  recentVerifications$: Observable<VerificationResult[]>;
  
  loading = true;
  error: string | null = null;
  isWebSocketConnected = false;
  recentEvents: RealtimeEvent[] = [];
  
  private subscriptions: Subscription[] = [];

  constructor(
    private apiService: ApiService,
    private webSocketService: WebSocketService
  ) {}

  ngOnInit(): void {
    this.loadData();
    this.setupWebSocket();
  }

  ngOnDestroy(): void {
    this.cleanup();
  }

  private setupWebSocket(): void {
    // Connect to WebSocket for real-time updates
    const wsUrl = this.getWebSocketUrl();
    this.webSocketService.connect(wsUrl);

    // Subscribe to connection status
    const connectionSub = this.webSocketService.connected$.subscribe(connected => {
      this.isWebSocketConnected = connected;
    });

    // Subscribe to real-time events
    const eventSub = this.webSocketService.events$.subscribe(event => {
      this.handleRealtimeEvent(event);
    });

    this.subscriptions.push(connectionSub, eventSub);
  }

  private getWebSocketUrl(): string {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    return `${protocol}//${host}/api/v1/ws`;
  }

  private handleRealtimeEvent(event: RealtimeEvent): void {
    // Add to recent events (keep last 10)
    this.recentEvents.unshift(event);
    if (this.recentEvents.length > 10) {
      this.recentEvents = this.recentEvents.slice(0, 10);
    }

    // Handle specific event types
    switch (event.type) {
      case 'model.verified':
      case 'verification.completed':
        // Refresh dashboard data when verification completes
        this.loadData();
        break;
      case 'model.verification.failed':
        // Show notification for failed verification
        this.error = `Verification failed: ${event.message}`;
        setTimeout(() => this.error = null, 5000);
        break;
      case 'system.health.changed':
        // Refresh system status
        this.loadData();
        break;
    }
  }

  private cleanup(): void {
    // Cleanup WebSocket and subscriptions
    this.webSocketService.disconnect();
    this.subscriptions.forEach(sub => sub.unsubscribe());
    this.subscriptions = [];
  }

  loadData(): void {
    this.loading = true;
    this.error = null;

    // Load all dashboard data
    this.dashboardStats$ = this.apiService.getDashboardStats();
    this.models$ = this.apiService.getModels(10); // Get first 10 models
    this.providers$ = this.apiService.getProviders();
    this.recentVerifications$ = this.apiService.getVerificationResults(5);

    // Handle loading state
    this.dashboardStats$.subscribe({
      next: () => {
        this.loading = false;
      },
      error: (err) => {
        this.error = 'Failed to load dashboard data';
        this.loading = false;
        console.error('Dashboard data error:', err);
      }
    });
  }

  refreshData(): void {
    console.log('Refreshing dashboard data...');
    this.loadData();
  }

  verifyModel(modelId: number): void {
    this.apiService.verifyModel(modelId.toString()).subscribe({
      next: (result) => {
        console.log('Verification started:', result);
        // Refresh data after verification starts
        setTimeout(() => this.loadData(), 1000);
      },
      error: (err) => {
        console.error('Failed to start verification:', err);
        this.error = 'Failed to start model verification';
      }
    });
  }

  deleteModel(modelId: number): void {
    if (confirm('Are you sure you want to delete this model?')) {
      this.apiService.deleteModel(modelId).subscribe({
        next: () => {
          console.log('Model deleted successfully');
          this.loadData();
        },
        error: (err) => {
          console.error('Failed to delete model:', err);
          this.error = 'Failed to delete model';
        }
      });
    }
  }

  formatDate(dateString: string): string {
    return new Date(dateString).toLocaleString();
  }

  getVerificationStatusClass(status: string): string {
    switch (status.toLowerCase()) {
      case 'completed':
        return 'status-completed';
      case 'running':
        return 'status-running';
      case 'failed':
        return 'status-failed';
      case 'pending':
        return 'status-pending';
      default:
        return 'status-unknown';
    }
  }

  getScoreClass(score: number): string {
    if (score >= 90) return 'score-excellent';
    if (score >= 80) return 'score-good';
    if (score >= 70) return 'score-average';
    if (score >= 60) return 'score-below-average';
    return 'score-poor';
  }
}
}