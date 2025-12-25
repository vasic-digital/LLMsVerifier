import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { DashboardComponent } from './dashboard.component';

// Angular Material Modules
import { MatCardModule } from '@angular/material/card';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatGridListModule } from '@angular/material/grid-list';
import { MatIconModule } from '@angular/material/icon';
import { MatTabsModule } from '@angular/material/tabs';
import { MatChipsModule } from '@angular/material/chips';
import { MatButtonModule } from '@angular/material/button';
import { MatTableModule } from '@angular/material/table';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { LottieModule } from 'ngx-lottie';

@NgModule({
  declarations: [
    DashboardComponent
  ],
  imports: [
    CommonModule,
    RouterModule.forChild([
      {
        path: '',
        component: DashboardComponent
      }
    ]),
    MatCardModule,
    MatProgressBarModule,
    MatGridListModule,
    MatIconModule,
    MatTabsModule,
    MatChipsModule,
    MatButtonModule,
    MatTableModule,
    MatProgressSpinnerModule,
    LottieModule
  ]
})
export class DashboardModule { }
