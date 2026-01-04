import { NgModule } from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { HttpClientModule } from '@angular/common/http';
import { AppRoutingModule } from './app.routes';
import { AppComponent } from './app.component';

// Shared Material Module - reduces bundle size by centralizing Material imports
import { MaterialModule } from './shared/material.module';

// Custom Components
import { DashboardMetricsModule } from './components/dashboard-metrics/dashboard-metrics.module';
import { ChartModule } from './components/chart/chart.module';

@NgModule({
  declarations: [],
  imports: [
    BrowserModule,
    BrowserAnimationsModule,
    HttpClientModule,
    AppRoutingModule,
    // Use shared MaterialModule instead of individual Material imports
    MaterialModule,
    DashboardMetricsModule,
    ChartModule
  ],
  providers: [],
  bootstrap: [AppComponent]
})
export class AppModule { }
