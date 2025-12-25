import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';

const routes: Routes = [
  {
    path: '',
    redirectTo: '/dashboard',
    pathMatch: 'full'
  },
  {
    path: 'dashboard',
    loadChildren: () => import('./pages/dashboard/dashboard.module').then(m => m.DashboardModule)
  },
  {
    path: 'providers',
    loadChildren: () => import('./pages/providers/providers.module').then(m => m.ProvidersModule)
  },
  {
    path: 'verification',
    loadChildren: () => import('./pages/verification/verification.module').then(m => m.VerificationModule)
  },
  {
    path: 'brotli',
    loadChildren: () => import('./pages/brotli/brotli.module').then(m => m.BrotliModule)
  },
  {
    path: 'monitoring',
    loadChildren: () => import('./pages/monitoring/monitoring.module').then(m => m.MonitoringModule)
  },
  {
    path: 'docs',
    loadChildren: () => import('./pages/docs/docs.module').then(m => m.DocsModule)
  }
];

@NgModule({
  imports: [RouterModule.forRoot(routes, {
    scrollPositionRestoration: 'enabled'
  })],
  exports: [RouterModule]
})
export class AppRoutingModule { }
