import { Component, Input, OnInit, OnChanges, SimpleChanges } from '@angular/core';

export interface ChartData {
  labels: string[];
  datasets: ChartDataset[];
}

export interface ChartDataset {
  label: string;
  data: number[];
  backgroundColor?: string | string[];
  borderColor?: string | string[];
  borderWidth?: number;
}

export type ChartType = 'line' | 'bar' | 'pie' | 'doughnut' | 'radar';

export interface ChartOptions {
  responsive?: boolean;
  maintainAspectRatio?: boolean;
  scales?: {
    x?: {
      title?: string;
    };
    y?: {
      title?: string;
      beginAtZero?: boolean;
    };
  };
  plugins?: {
    legend?: {
      display?: boolean;
      position?: 'top' | 'bottom' | 'left' | 'right';
    };
    title?: {
      display?: boolean;
      text?: string;
    };
  };
}

@Component({
  selector: 'app-chart',
  templateUrl: './chart.component.html',
  styleUrls: ['./chart.component.scss']
})
export class ChartComponent implements OnInit, OnChanges {
  @Input() data: ChartData = { labels: [], datasets: [] };
  @Input() type: ChartType = 'bar';
  @Input() options: ChartOptions = {};
  @Input() title: string = '';
  @Input() height: string = '300px';

  chartId = `chart-${Math.random().toString(36).substr(2, 9)}`;
  chartInstance: any;

  ngOnInit(): void {
    this.initChart();
  }

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['data'] || changes['type'] || changes['options']) {
      this.initChart();
    }
  }

  private initChart(): void {
    // Use Angular Material's built-in charting capabilities
    // This is a placeholder for actual chart library integration
    // In a real implementation, you would use Chart.js, ng2-charts, or similar
    
    // For now, we'll create a simple SVG-based visualization
    this.createSimpleChart();
  }

  private createSimpleChart(): void {
    // Simple SVG chart implementation for demonstration
    // This would be replaced with a proper chart library
    const canvas = document.getElementById(this.chartId) as HTMLCanvasElement;
    if (!canvas) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    // Clear canvas
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    // Simple bar chart implementation
    if (this.data.datasets.length > 0 && this.data.labels.length > 0) {
      const dataset = this.data.datasets[0];
      const maxValue = Math.max(...dataset.data);
      const barWidth = canvas.width / dataset.data.length;
      
      dataset.data.forEach((value, index) => {
        const barHeight = (value / maxValue) * canvas.height * 0.8;
        const x = index * barWidth;
        const y = canvas.height - barHeight;

        ctx.fillStyle = dataset.backgroundColor as string || '#2196F3';
        ctx.fillRect(x + 5, y, barWidth - 10, barHeight);

        // Add label
        ctx.fillStyle = '#333';
        ctx.font = '12px Arial';
        ctx.textAlign = 'center';
        ctx.fillText(value.toString(), x + barWidth / 2, y - 5);
      });
    }
  }

  // Helper methods for common chart types
  static createLineChart(labels: string[], datasets: { label: string; data: number[]; color: string }[]): ChartData {
    return {
      labels,
      datasets: datasets.map(ds => ({
        label: ds.label,
        data: ds.data,
        borderColor: ds.color,
        backgroundColor: `${ds.color}20`,
        borderWidth: 2,
        fill: true
      }))
    };
  }

  static createBarChart(labels: string[], datasets: { label: string; data: number[]; colors: string[] }): ChartData {
    return {
      labels,
      datasets: [{
        label: datasets.label,
        data: datasets.data,
        backgroundColor: datasets.colors,
        borderColor: datasets.colors.map(color => color + 'CC'),
        borderWidth: 1
      }]
    };
  }

  static createPieChart(labels: string[], data: number[], colors: string[]): ChartData {
    return {
      labels,
      datasets: [{
        label: 'Distribution',
        data,
        backgroundColor: colors,
        borderColor: colors.map(color => color + 'CC'),
        borderWidth: 1
      }]
    };
  }

  // Export chart data
  exportAsImage(): void {
    const canvas = document.getElementById(this.chartId) as HTMLCanvasElement;
    if (!canvas) return;

    const link = document.createElement('a');
    link.download = `${this.title || 'chart'}.png`;
    link.href = canvas.toDataURL();
    link.click();
  }

  // Reset chart view
  resetView(): void {
    this.initChart();
  }
}