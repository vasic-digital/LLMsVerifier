import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';

const routes: Routes = [
  { path: '', redirectTo: '/dashboard', pathMatch: 'full' },
  { path: 'dashboard', loadChildren: () => import('./dashboard/dashboard.module').then(m => m.DashboardModule) },
  { path: 'models', loadChildren: () => import('./models/models.module').then(m => m.ModelsModule) },
  { path: 'providers', loadChildren: () => import('./providers/providers.module').then(m => m.ProvidersModule) },
  { path: 'verification', loadChildren: () => import('./verification/verification.module').then(m => m.VerificationModule) },
];

@NgModule({
  imports: [RouterModule.forRoot(routes)],
  exports: [RouterModule]
})
export class AppRoutingModule { }