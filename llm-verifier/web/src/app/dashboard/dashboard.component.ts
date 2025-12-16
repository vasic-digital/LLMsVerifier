import { Component, OnInit, OnDestroy } from '@angular/core';
import { Subject, interval, takeUntil } from 'rxjs';

interface DashboardStats {
  totalModels: number;
  totalProviders: number;
  verifiedModels: number;
  pendingModels: number;
  averageScore: number;
  lastVerification: Date;
  recentActivity: ActivityItem[];
}

interface ActivityItem {
  id: string;
  type: 'verification' | 'error' | 'success';
  message: string;
  timestamp: Date;
  details?: any;
}

@Component({
  selector: 'app-dashboard',
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.scss']
})
export class DashboardComponent implements OnInit, OnDestroy {
  private destroy$ = new Subject<void>();

  stats: DashboardStats = {
    totalModels: 0,
    totalProviders: 0,
    verifiedModels: 0,
    pendingModels: 0,
    averageScore: 0,
    lastVerification: new Date(),
    recentActivity: []
  };

  // Chart data
  scoreChartData = {
    labels: [],
    datasets: [{
      label: 'Model Scores',
      data: [],
      backgroundColor: 'rgba(54, 162, 235, 0.2)',
      borderColor: 'rgba(54, 162, 235, 1)',
      borderWidth: 1
    }]
  };

  verificationChartData = {
    labels: ['Verified', 'Pending', 'Failed'],
    datasets: [{
      data: [0, 0, 0],
      backgroundColor: ['#4CAF50', '#FFC107', '#F44336'],
      hoverBackgroundColor: ['#45a049', '#ffb300', '#da190b']
    }]
  };

  activityChartData = {
    labels: [],
    datasets: [{
      label: 'Activity',
      data: [],
      borderColor: 'rgba(75, 192, 192, 1)',
      backgroundColor: 'rgba(75, 192, 192, 0.2)',
      tension: 0.1
    }]
  };

  constructor() {}

  ngOnInit(): void {
    this.loadDashboardData();

    // Refresh data every 30 seconds
    interval(30000)
      .pipe(takeUntil(this.destroy$))
      .subscribe(() => {
        this.loadDashboardData();
      });
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  loadDashboardData(): void {
    // Mock data - in real implementation, this would call the API service
    this.stats = {
      totalModels: 25,
      totalProviders: 5,
      verifiedModels: 20,
      pendingModels: 5,
      averageScore: 87.5,
      lastVerification: new Date(),
      recentActivity: [
        {
          id: '1',
          type: 'verification',
          message: 'Model GPT-4 verification completed',
          timestamp: new Date(Date.now() - 5 * 60 * 1000),
          details: { score: 95.2, duration: '2.3s' }
        },
        {
          id: '2',
          type: 'success',
          message: 'Daily verification schedule executed',
          timestamp: new Date(Date.now() - 15 * 60 * 1000),
          details: { modelsProcessed: 25 }
        },
        {
          id: '3',
          type: 'error',
          message: 'Claude API rate limit exceeded',
          timestamp: new Date(Date.now() - 30 * 60 * 1000),
          details: { retryIn: '5 minutes' }
        }
      ]
    };

    // Update charts
    this.updateCharts();
  }

  private updateCharts(): void {
    // Score distribution chart
    this.scoreChartData = {
      labels: ['90-100', '80-89', '70-79', '60-69', '<60'],
      datasets: [{
        label: 'Model Scores',
        data: [8, 10, 5, 1, 1],
        backgroundColor: [
          'rgba(76, 175, 80, 0.8)',
          'rgba(139, 195, 74, 0.8)',
          'rgba(255, 193, 7, 0.8)',
          'rgba(255, 152, 0, 0.8)',
          'rgba(244, 67, 54, 0.8)'
        ],
        borderColor: [
          'rgba(76, 175, 80, 1)',
          'rgba(139, 195, 74, 1)',
          'rgba(255, 193, 7, 1)',
          'rgba(255, 152, 0, 1)',
          'rgba(244, 67, 54, 1)'
        ],
        borderWidth: 1
      }]
    };

    // Verification status chart
    this.verificationChartData = {
      labels: ['Verified', 'Pending', 'Failed'],
      datasets: [{
        data: [this.stats.verifiedModels, this.stats.pendingModels, 0],
        backgroundColor: ['#4CAF50', '#FFC107', '#F44336'],
        hoverBackgroundColor: ['#45a049', '#ffb300', '#da190b']
      }]
    };

    // Activity over time (mock data)
    const now = new Date();
    const labels = [];
    const data = [];
    for (let i = 23; i >= 0; i--) {
      const time = new Date(now.getTime() - i * 60 * 60 * 1000);
      labels.push(time.getHours() + ':00');
      data.push(Math.floor(Math.random() * 10) + 1);
    }

    this.activityChartData = {
      labels,
      datasets: [{
        label: 'Activity',
        data,
        borderColor: 'rgba(75, 192, 192, 1)',
        backgroundColor: 'rgba(75, 192, 192, 0.2)',
        tension: 0.1
      }]
    };
  }

  refreshData(): void {
    this.loadDashboardData();
  }

  getActivityIcon(activity: ActivityItem): string {
    switch (activity.type) {
      case 'verification':
        return '🔍';
      case 'success':
        return '✅';
      case 'error':
        return '❌';
      default:
        return 'ℹ️';
    }
  }

  getActivityColor(activity: ActivityItem): string {
    switch (activity.type) {
      case 'verification':
        return 'text-blue-600';
      case 'success':
        return 'text-green-600';
      case 'error':
        return 'text-red-600';
      default:
        return 'text-gray-600';
    }
  }

  formatTimeAgo(date: Date): string {
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / (1000 * 60));
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60));

    if (diffMins < 1) {
      return 'Just now';
    } else if (diffMins < 60) {
      return `${diffMins} minutes ago`;
    } else if (diffHours < 24) {
      return `${diffHours} hours ago`;
    } else {
      return date.toLocaleDateString();
    }
  }
}