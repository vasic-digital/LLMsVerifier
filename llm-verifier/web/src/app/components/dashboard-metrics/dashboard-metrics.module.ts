import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatChipsModule } from '@angular/material/chips';
import { MatTooltipModule } from '@angular/material/tooltip';
import { DashboardMetricsComponent } from './dashboard-metrics.component';
import { ChartModule } from '../chart/chart.module';

@NgModule({
  declarations: [DashboardMetricsComponent],
  imports: [
    CommonModule,
    MatIconModule,
    MatButtonModule,
    MatProgressSpinnerModule,
    MatChipsModule,
    MatTooltipModule,
    ChartModule
  ],
  exports: [DashboardMetricsComponent]
})
export class DashboardMetricsModule { }