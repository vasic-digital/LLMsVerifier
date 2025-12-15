import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule, Routes } from '@angular/router';

import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';

import { VerificationComponent } from './verification.component';

const routes: Routes = [
  { path: '', component: VerificationComponent }
];

@NgModule({
  declarations: [VerificationComponent],
  imports: [
    CommonModule,
    RouterModule.forChild(routes),
    MatCardModule,
    MatButtonModule
  ]
})
export class VerificationModule { }