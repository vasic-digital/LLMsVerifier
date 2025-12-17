import { Component, OnInit } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiService, Model, Provider } from '../api.service';

@Component({
  selector: 'app-models',
  templateUrl: './models.component.html',
  styleUrls: ['./models.component.scss']
})
export class ModelsComponent implements OnInit {
  models$: Observable<Model[]>;
  providers$: Observable<Provider[]>;
  
  loading = true;
  error: string | null = null;
  selectedModel: Model | null = null;
  showCreateForm = false;
  showEditForm = false;

  constructor(private apiService: ApiService) {}

  ngOnInit(): void {
    this.loadData();
  }

  loadData(): void {
    this.loading = true;
    this.error = null;

    this.models$ = this.apiService.getModels();
    this.providers$ = this.apiService.getProviders();

    this.models$.subscribe({
      next: () => {
        this.loading = false;
      },
      error: (err) => {
        this.error = 'Failed to load models';
        this.loading = false;
        console.error('Models load error:', err);
      }
    });
  }

  refreshData(): void {
    this.loadData();
  }

  selectModel(model: Model): void {
    this.selectedModel = model;
    this.showEditForm = true;
    this.showCreateForm = false;
  }

  createNewModel(): void {
    this.selectedModel = null;
    this.showCreateForm = true;
    this.showEditForm = false;
  }

  deleteModel(modelId: number): void {
    if (confirm('Are you sure you want to delete this model?')) {
      this.apiService.deleteModel(modelId).subscribe({
        next: () => {
          console.log('Model deleted successfully');
          this.refreshData();
        },
        error: (err) => {
          console.error('Failed to delete model:', err);
          this.error = 'Failed to delete model';
        }
      });
    }
  }

  verifyModel(modelId: number): void {
    this.apiService.verifyModel(modelId.toString()).subscribe({
      next: (result) => {
        console.log('Verification started:', result);
        this.refreshData();
      },
      error: (err) => {
        console.error('Failed to start verification:', err);
        this.error = 'Failed to start model verification';
      }
    });
  }

  onModelCreated(model: Model): void {
    this.showCreateForm = false;
    this.refreshData();
  }

  onModelUpdated(model: Model): void {
    this.showEditForm = false;
    this.selectedModel = null;
    this.refreshData();
  }

  cancelForm(): void {
    this.showCreateForm = false;
    this.showEditForm = false;
    this.selectedModel = null;
  }

  formatDate(dateString: string): string {
    return new Date(dateString).toLocaleString();
  }

  getVerificationStatusClass(status: string): string {
    switch (status.toLowerCase()) {
      case 'verified':
        return 'status-verified';
      case 'pending':
        return 'status-pending';
      case 'failed':
        return 'status-failed';
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

  getProviderName(providerId: number, providers: Provider[]): string {
    const provider = providers.find(p => p.id === providerId);
    return provider ? provider.name : `Provider ${providerId}`;
  }
}