import { Component, OnInit } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiService, DashboardStats, Model, Provider, VerificationResult } from '../api.service';

@Component({
  selector: 'app-dashboard',
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.scss']
})
export class DashboardComponent implements OnInit {
  title = 'LLM Verifier Dashboard';
  dashboardStats$: Observable<DashboardStats>;
  models$: Observable<Model[]>;
  providers$: Observable<Provider[]>;
  recentVerifications$: Observable<VerificationResult[]>;
  
  loading = true;
  error: string | null = null;

  constructor(private apiService: ApiService) {}

  ngOnInit(): void {
    this.loadData();
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