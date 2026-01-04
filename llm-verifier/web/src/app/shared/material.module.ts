import { NgModule } from '@angular/core';

// Angular Material Modules - imported only once and re-exported
// This reduces bundle size by avoiding duplicate imports across components
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatSidenavModule } from '@angular/material/sidenav';
import { MatListModule } from '@angular/material/list';
import { MatCardModule } from '@angular/material/card';
import { MatTableModule } from '@angular/material/table';
import { MatTabsModule } from '@angular/material/tabs';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatChipsModule } from '@angular/material/chips';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatSnackBarModule } from '@angular/material/snack-bar';
import { MatMenuModule } from '@angular/material/menu';
import { MatSelectModule } from '@angular/material/select';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatDividerModule } from '@angular/material/divider';
import { MatExpansionModule } from '@angular/material/expansion';
import { MatDialogModule } from '@angular/material/dialog';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatRadioModule } from '@angular/material/radio';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatBadgeModule } from '@angular/material/badge';
import { MatPaginatorModule } from '@angular/material/paginator';
import { MatSortModule } from '@angular/material/sort';

/**
 * Shared Material Module
 *
 * Centralizes all Angular Material imports to:
 * 1. Reduce bundle size by avoiding duplicate imports
 * 2. Ensure consistent Material component availability
 * 3. Simplify maintenance of Material dependencies
 *
 * Usage: Import MaterialModule in feature modules that need Material components
 */
@NgModule({
  exports: [
    // Layout
    MatToolbarModule,
    MatSidenavModule,
    MatDividerModule,
    MatExpansionModule,

    // Navigation
    MatMenuModule,
    MatListModule,
    MatTabsModule,

    // Forms
    MatButtonModule,
    MatIconModule,
    MatInputModule,
    MatFormFieldModule,
    MatSelectModule,
    MatCheckboxModule,
    MatRadioModule,
    MatSlideToggleModule,

    // Data Display
    MatCardModule,
    MatTableModule,
    MatChipsModule,
    MatBadgeModule,
    MatPaginatorModule,
    MatSortModule,

    // Feedback
    MatProgressSpinnerModule,
    MatProgressBarModule,
    MatTooltipModule,
    MatSnackBarModule,
    MatDialogModule
  ]
})
export class MaterialModule { }
