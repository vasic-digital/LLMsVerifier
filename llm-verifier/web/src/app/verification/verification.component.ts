import { Component, OnInit } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiService, VerificationResult, Model, Provider } from '../api.service';

@Component({
  selector: 'app-verification',
  templateUrl: './verification.component.html',
  styleUrls: ['./verification.component.scss']
})
export class VerificationComponent implements OnInit {
  verificationResults$: Observable<VerificationResult[]>;
  models$: Observable<Model[]>;
  providers$: Observable<Provider[]>;
  
  loading = true;
  error: string | null = null;
  selectedResult: VerificationResult | null = null;
  showDetails = false;
  showFilters = false;
  
  // Filters
  filters = {
    model_id: '',
    status: '',
    limit: 50,
    offset: 0
  };

  constructor(private apiService: ApiService) {}

  ngOnInit(): void {
    this.loadData();
  }

  loadData(): void {
    this.loading = true;
    this.error = null;

    this.verificationResults$ = this.apiService.getVerificationResults(this.filters.limit, this.filters.offset);
    this.models$ = this.apiService.getModels();
    this.providers$ = this.apiService.getProviders();

    this.verificationResults$.subscribe({
      next: () => {
        this.loading = false;
      },
      error: (err) => {
        this.error = 'Failed to load verification results';
        this.loading = false;
        console.error('Verification results load error:', err);
      }
    });
  }

  refreshData(): void {
    this.loadData();
  }

  applyFilters(): void {
    this.filters.offset = 0; // Reset to first page
    this.loadData();
  }

  resetFilters(): void {
    this.filters = {
      model_id: '',
      status: '',
      limit: 50,
      offset: 0
    };
    this.loadData();
  }

  showResultDetails(result: VerificationResult): void {
    this.selectedResult = result;
    this.showDetails = true;
  }

  closeDetails(): void {
    this.showDetails = false;
    this.selectedResult = null;
  }

  reverifyResult(result: VerificationResult): void {
    this.apiService.verifyModel(result.model_id.toString()).subscribe({
      next: (verification) => {
        console.log('Re-verification started:', verification);
        this.refreshData();
      },
      error: (err) => {
        console.error('Failed to start re-verification:', err);
        this.error = 'Failed to start re-verification';
      }
    });
  }

  deleteResult(resultId: number): void {
    if (confirm('Are you sure you want to delete this verification result?')) {
      this.apiService.deleteVerificationResult(resultId).subscribe({
        next: () => {
          console.log('Verification result deleted successfully');
          this.refreshData();
        },
        error: (err) => {
          console.error('Failed to delete verification result:', err);
          this.error = 'Failed to delete verification result';
        }
      });
    }
  }

  loadMore(): void {
    this.filters.offset += this.filters.limit;
    this.apiService.getVerificationResults(this.filters.limit, this.filters.offset).subscribe({
      next: (newResults) => {
        // In a real implementation, you would append these to existing results
        console.log('Loaded more results:', newResults);
      },
      error: (err) => {
        console.error('Failed to load more results:', err);
        this.error = 'Failed to load more results';
      }
    });
  }

  formatDate(dateString: string): string {
    return new Date(dateString).toLocaleString();
  }

  getDuration(startTime: string, endTime?: string): string {
    if (!endTime) return 'In progress';
    
    const start = new Date(startTime);
    const end = new Date(endTime);
    const duration = end.getTime() - start.getTime();
    
    if (duration < 1000) {
      return `${duration}ms`;
    } else if (duration < 60000) {
      return `${Math.round(duration / 1000)}s`;
    } else {
      return `${Math.round(duration / 60000)}m ${Math.round((duration % 60000) / 1000)}s`;
    }
  }

  getStatusClass(status: string): string {
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

  getModelName(modelId: number, models: Model[]): string {
    const model = models.find(m => m.id === modelId);
    return model ? model.name : `Model ${modelId}`;
  }

  getProviderName(providerId: number, providers: Provider[]): string {
    const provider = providers.find(p => p.id === providerId);
    return provider ? provider.name : `Provider ${providerId}`;
  }
}