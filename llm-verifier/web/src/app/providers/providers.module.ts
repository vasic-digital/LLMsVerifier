import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule, Routes } from '@angular/router';

import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';

import { ProvidersComponent } from './providers.component';

const routes: Routes = [
  { path: '', component: ProvidersComponent }
];

@NgModule({
  declarations: [ProvidersComponent],
  imports: [
    CommonModule,
    RouterModule.forChild(routes),
    MatCardModule,
    MatButtonModule
  ]
})
export class ProvidersModule { }