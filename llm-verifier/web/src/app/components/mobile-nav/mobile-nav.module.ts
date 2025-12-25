import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { MatIconModule } from '@angular/material/icon';
import { MobileNavComponent } from './mobile-nav.component';

@NgModule({
  declarations: [MobileNavComponent],
  imports: [
    CommonModule,
    RouterModule,
    MatIconModule
  ],
  exports: [MobileNavComponent]
})
export class MobileNavModule { }