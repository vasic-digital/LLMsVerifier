import { Component, OnInit } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiService, Provider } from '../api.service';

@Component({
  selector: 'app-providers',
  templateUrl: './providers.component.html',
  styleUrls: ['./providers.component.scss']
})
export class ProvidersComponent implements OnInit {
  providers$: Observable<Provider[]>;
  loading = true;
  error: string | null = null;
  selectedProvider: Provider | null = null;
  showCreateForm = false;
  showEditForm = false;

  constructor(private apiService: ApiService) {}

  ngOnInit(): void {
    this.loadData();
  }

  loadData(): void {
    this.loading = true;
    this.error = null;

    this.providers$ = this.apiService.getProviders();

    this.providers$.subscribe({
      next: () => {
        this.loading = false;
      },
      error: (err) => {
        this.error = 'Failed to load providers';
        this.loading = false;
        console.error('Providers load error:', err);
      }
    });
  }

  refreshData(): void {
    this.loadData();
  }

  selectProvider(provider: Provider): void {
    this.selectedProvider = provider;
    this.showEditForm = true;
    this.showCreateForm = false;
  }

  createNewProvider(): void {
    this.selectedProvider = null;
    this.showCreateForm = true;
    this.showEditForm = false;
  }

  deleteProvider(providerId: number): void {
    if (confirm('Are you sure you want to delete this provider? This will also delete all models associated with this provider.')) {
      this.apiService.deleteProvider(providerId).subscribe({
        next: () => {
          console.log('Provider deleted successfully');
          this.refreshData();
        },
        error: (err) => {
          console.error('Failed to delete provider:', err);
          this.error = 'Failed to delete provider';
        }
      });
    }
  }

  onProviderCreated(provider: Provider): void {
    this.showCreateForm = false;
    this.refreshData();
  }

  onProviderUpdated(provider: Provider): void {
    this.showEditForm = false;
    this.selectedProvider = null;
    this.refreshData();
  }

  cancelForm(): void {
    this.showCreateForm = false;
    this.showEditForm = false;
    this.selectedProvider = null;
  }

  formatDate(dateString: string): string {
    return new Date(dateString).toLocaleString();
  }

  getStatusClass(status: string): string {
    switch (status.toLowerCase()) {
      case 'active':
        return 'status-active';
      case 'inactive':
        return 'status-inactive';
      default:
        return 'status-unknown';
    }
  }

  getReliabilityClass(score: number): string {
    if (score >= 90) return 'reliability-excellent';
    if (score >= 80) return 'reliability-good';
    if (score >= 70) return 'reliability-average';
    if (score >= 60) return 'reliability-below-average';
    return 'reliability-poor';
  }
}