import { Component, OnInit, OnDestroy } from '@angular/core';
import { Observable, Subscription, interval } from 'rxjs';
import { ApiService, DashboardStats, Model, VerificationResult } from '../../api.service';
import { ChartComponent, ChartData, ChartType } from '../chart/chart.component';

interface MetricsData {
  verificationTrends: ChartData;
  providerDistribution: ChartData;
  scoreDistribution: ChartData;
  performanceMetrics: ChartData;
}

@Component({
  selector: 'app-dashboard-metrics',
  templateUrl: './dashboard-metrics.component.html',
  styleUrls: ['./dashboard-metrics.component.scss']
})
export class DashboardMetricsComponent implements OnInit, OnDestroy {
  dashboardStats$!: Observable<DashboardStats>;
  models$!: Observable<Model[]>;
  recentVerifications$!: Observable<VerificationResult[]>;
  
  metricsData: MetricsData = {
    verificationTrends: { labels: [], datasets: [] },
    providerDistribution: { labels: [], datasets: [] },
    scoreDistribution: { labels: [], datasets: [] },
    performanceMetrics: { labels: [], datasets: [] }
  };
  
  loading = true;
  autoRefresh = true;
  refreshInterval = 30000; // 30 seconds
  
  private subscriptions: Subscription[] = [];

  constructor(private apiService: ApiService) {}

  ngOnInit(): void {
    this.loadMetrics();
    this.setupAutoRefresh();
  }

  ngOnDestroy(): void {
    this.subscriptions.forEach(sub => sub.unsubscribe());
  }

  private setupAutoRefresh(): void {
    const refreshSub = interval(this.refreshInterval).subscribe(() => {
      if (this.autoRefresh) {
        this.loadMetrics();
      }
    });
    this.subscriptions.push(refreshSub);
  }

  loadMetrics(): void {
    this.loading = true;
    
    this.dashboardStats$ = this.apiService.getDashboardStats();
    this.models$ = this.apiService.getModels(50);
    this.recentVerifications$ = this.apiService.getVerificationResults(100);

    // Subscribe to all observables to aggregate data
    const combinedSub = this.models$.subscribe(models => {
      this.recentVerifications$.subscribe(verifications => {
        this.dashboardStats$.subscribe(stats => {
          this.generateMetricsData(models, verifications, stats);
          this.loading = false;
        });
      });
    });

    this.subscriptions.push(combinedSub);
  }

  private generateMetricsData(
    models: Model[],
    verifications: VerificationResult[],
    stats: DashboardStats
  ): void {
    // Generate verification trends chart
    this.metricsData.verificationTrends = this.generateVerificationTrends(verifications);
    
    // Generate provider distribution chart
    this.metricsData.providerDistribution = this.generateProviderDistribution(models);
    
    // Generate score distribution chart
    this.metricsData.scoreDistribution = this.generateScoreDistribution(verifications);
    
    // Generate performance metrics chart
    this.metricsData.performanceMetrics = this.generatePerformanceMetrics(verifications);
  }

  private generateVerificationTrends(verifications: VerificationResult[]): ChartData {
    // Group verifications by day
    const dailyData = this.groupByDay(verifications);
    const labels = Object.keys(dailyData).sort();
    const data = labels.map(label => dailyData[label].length);

    return ChartComponent.createLineChart(labels, [{
      label: 'Daily Verifications',
      data,
      color: '#2196F3'
    }]);
  }

  private generateProviderDistribution(models: Model[]): ChartData {
    // Group models by provider
    const providerCounts = this.groupByProvider(models);
    const labels = Object.keys(providerCounts);
    const data = labels.map(label => providerCounts[label]);

    const colors = this.generateColors(labels.length);

    return ChartComponent.createPieChart(labels, data, colors);
  }

  private generateScoreDistribution(verifications: VerificationResult[]): ChartData {
    // Create score ranges
    const ranges = ['0-20', '21-40', '41-60', '61-80', '81-100'];
    const counts = [0, 0, 0, 0, 0];

    verifications.forEach(verification => {
      const score = verification.overall_score;
      if (score >= 0 && score <= 20) counts[0]++;
      else if (score <= 40) counts[1]++;
      else if (score <= 60) counts[2]++;
      else if (score <= 80) counts[3]++;
      else counts[4]++;
    });

    const colors = ['#F44336', '#FF9800', '#FFEB3B', '#8BC34A', '#4CAF50'];

    return ChartComponent.createBarChart(ranges, {
      label: 'Score Distribution',
      data: counts,
      colors
    });
  }

  private generatePerformanceMetrics(verifications: VerificationResult[]): ChartData {
    // Calculate average latency by provider
    const providerLatency = this.calculateProviderLatency(verifications);
    const labels = Object.keys(providerLatency);
    const data = labels.map(label => providerLatency[label]);

    return ChartComponent.createBarChart(labels, {
      label: 'Average Latency (ms)',
      data,
      colors: labels.map(() => '#607D8B')
    });
  }

  private groupByDay(verifications: VerificationResult[]): { [key: string]: VerificationResult[] } {
    const grouped: { [key: string]: VerificationResult[] } = {};
    
    verifications.forEach(verification => {
      const date = new Date(verification.started_at).toISOString().split('T')[0];
      if (!grouped[date]) {
        grouped[date] = [];
      }
      grouped[date].push(verification);
    });
    
    return grouped;
  }

  private groupByProvider(models: Model[]): { [key: string]: number } {
    const grouped: { [key: string]: number } = {};
    
    models.forEach(model => {
      const provider = model.provider_id.toString();
      grouped[provider] = (grouped[provider] || 0) + 1;
    });
    
    return grouped;
  }

  private calculateProviderLatency(verifications: VerificationResult[]): { [key: string]: number } {
    const latencies: { [key: string]: number[] } = {};
    
    verifications.forEach(verification => {
      const provider = verification.model_id.toString(); // Using model_id as provider proxy
      if (!latencies[provider]) {
        latencies[provider] = [];
      }
      
      // Calculate latency (mock calculation)
      const start = new Date(verification.started_at).getTime();
      const end = verification.completed_at ? 
        new Date(verification.completed_at).getTime() : 
        Date.now();
      const latency = end - start;
      
      latencies[provider].push(latency);
    });
    
    const averages: { [key: string]: number } = {};
    Object.keys(latencies).forEach(provider => {
      const avg = latencies[provider].reduce((a, b) => a + b, 0) / latencies[provider].length;
      averages[provider] = Math.round(avg / 1000); // Convert to seconds
    });
    
    return averages;
  }

  private generateColors(count: number): string[] {
    const colors = [
      '#FF6384', '#36A2EB', '#FFCE56', '#4BC0C0', '#9966FF',
      '#FF9F40', '#FF6384', '#C9CBCF', '#7C7C7C', '#4BC0C0'
    ];
    
    // If we need more colors than available, generate random ones
    if (count > colors.length) {
      const additional = Array.from({ length: count - colors.length }, () => 
        '#' + Math.floor(Math.random() * 16777215).toString(16)
      );
      return [...colors, ...additional];
    }
    
    return colors.slice(0, count);
  }

  toggleAutoRefresh(): void {
    this.autoRefresh = !this.autoRefresh;
  }

  refreshMetrics(): void {
    this.loadMetrics();
  }
}