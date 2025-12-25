import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatGridListModule } from '@angular/material/grid-list';
import { MatIconModule } from '@angular/material/icon';
import { MatTabsModule } from '@angular/material/tabs';
import { MatChipsModule } from '@angular/material/chips';
import { MatButtonModule } from '@angular/material/button';
import { LottieModule } from 'ngx-lottie';
import { DashboardService } from '../../shared/servies/dashboard.service';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [
    CommonModule,
    MatCardModule,
    MatProgressBarModule,
    MatGridListModule,
    MatIconModule,
    MatTabsModule,
    MatChipsModule,
    MatButtonModule,
    LottieModule
  ],
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.scss']
})
export class DashboardComponent implements OnInit {
  isLoading = true;
  selectedTab = 0;
  
  overallStats = {
    totalProviders: 7,
    totalModels: 395,
    verifiedModels: 395,
    successRate: 96.8,
    brotliSupport: 312,
    brotliRate: 74.8,
    avgLatency: 152
  };

  providers = [
    { name: 'OpenAI', models: 98, verified: 98, brotli: 98, status: 'excellent', latency: 128, icon: 'openai' },
    { name: 'Anthropic', models: 15, verified: 15, brotli: 15, status: 'excellent', latency: 189, icon: 'anthropic' },
    { name: 'Google', models: 42, verified: 42, brotli: 42, status: 'excellent', latency: 143, icon: 'google' },
    { name: 'Meta', models: 28, verified: 28, brotli: 28, status: 'excellent', latency: 234, icon: 'meta' },
    { name: 'Cohere', models: 18, verified: 18, brotli: 18, status: 'excellent', latency: 167, icon: 'cohere' },
    { name: 'Azure', models: 96, verified: 96, brotli: 96, status: 'excellent', latency: 142, icon: 'azure' },
    { name: 'Amazon Bedrock', models: 35, verified: 35, brotli: 15, status: 'good', latency: 198, icon: 'aws' }
  ];

  recentVerifications = [
    { model: 'gpt-4', provider: 'OpenAI', status: 'success', latency: 145, timestamp: '2 min ago' },
    { model: 'claude-3-opus', provider: 'Anthropic', status: 'success', latency: 215, timestamp: '5 min ago' },
    { model: 'gemini-pro', provider: 'Google', status: 'success', latency: 135, timestamp: '8 min ago' },
    { model: 'llama-2-70b', provider: 'Meta', status: 'success', latency: 256, timestamp: '12 min ago' }
  ];

  performanceMetrics = {
    brotliTests: 417,
    brotliCacheHits: 334,
    brotliCacheMisses: 83,
    cacheHitRate: 80.1,
    avgDetectionTime: 0.485,
    bandwidthSaved: 68.5
  };

  alerts = [
    { type: 'success', message: 'All systems operational', timestamp: 'Just now' },
    { type: 'info', message: 'Brotli cache warming up - 83% efficiency', timestamp: '2 min ago' },
    { type: 'warning', message: 'High latency detected on Meta provider', timestamp: '5 min ago' }
  ];

  constructor(private dashboardService: DashboardService) {}

  ngOnInit() {
    this.loadDashboardData();
  }

  loadDashboardData() {
    this.dashboardService.getStats().subscribe(stats => {
      this.overallStats = stats;
    });
    
    this.dashboardService.getProviders().subscribe(providers => {
      this.providers = providers;
    });

    this.dashboardService.getRecentVerifications().subscribe(verifications => {
      this.recentVerifications = verifications;
    });

    this.dashboardService.getAlerts().subscribe(alerts => {
      this.alerts = alerts;
    });

    setTimeout(() => {
      this.isLoading = false;
    }, 1000);
  }

  getProviderStatusIcon(status: string): string {
    switch(status) {
      case 'excellent': return 'check_circle';
      case 'good': return 'check_circle';
      case 'fair': return 'warning';
      case 'poor': return 'error';
      default: return 'help';
    }
  }

  getProviderStatusColor(status: string): string {
    switch(status) {
      case 'excellent': return '#4CAF50';
      case 'good': return '#8BC34A';
      case 'fair': return '#FFC107';
      case 'poor': return '#F44336';
      default: return '#9E9E9E';
    }
  }

  refreshDashboard() {
    this.isLoading = true;
    this.loadDashboardData();
  }

  getStatusColor(status: string): string {
    switch(status) {
      case 'success': return '#4CAF50';
      case 'warning': return '#FFC107';
      case 'error': return '#F44336';
      case 'info': return '#2196F3';
      default: return '#9E9E9E';
    }
  }
}
