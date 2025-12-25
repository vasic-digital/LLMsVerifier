import { Component, OnInit, OnDestroy, Renderer2, HostListener } from '@angular/core';
import { ResponsiveService, ScreenSize } from './services/responsive.service';
import { Subscription } from 'rxjs';
import { CommonModule } from '@angular/common';
import { RouterModule, Routes } from '@angular/router';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
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
import { MobileNavModule } from './components/mobile-nav/mobile-nav.module';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [
    CommonModule,
    RouterModule,
    BrowserAnimationsModule,
    MatToolbarModule,
    MatButtonModule,
    MatIconModule,
    MatSidenavModule,
    MatListModule,

    MatCardModule,
    MatTableModule,
    MatTabsModule,
    MatProgressSpinnerModule,
    MatProgressBarModule,
    MatChipsModule,
    MatTooltipModule,
    MatSnackBarModule,
    MatMenuModule,
    MatSelectModule,
    MobileNavModule
  ],
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss']
})
export class AppComponent implements OnInit, OnDestroy {
  title = 'LLM Verifier';
  isDarkMode = false;
  selectedTab = 0;
  selectedProvider = 'openai';
  isSidenavOpen = false;
  
  providers = [
    { id: 'openai', name: 'OpenAI', icon: 'openai', models: 98, status: '✅ Verified' },
    { id: 'anthropic', name: 'Anthropic', icon: 'anthropic', models: 15, status: '✅ Verified' },
    { id: 'google', name: 'Google', icon: 'google', models: 42, status: '✅ Verified' },
    { id: 'meta', name: 'Meta', icon: 'meta', models: 28, status: '✅ Verified' },
    { id: 'cohere', name: 'Cohere', icon: 'cohere', models: 18, status: '✅ Verified' },
    { id: 'azure', name: 'Azure', icon: 'azure', models: 96, status: '✅ Verified' },
    { id: 'bedrock', name: 'Amazon Bedrock', icon: 'aws', models: 35, status: '✅ Verified' }
  ];

  stats = {
    totalProviders: 7,
    totalModels: 395,
    verifiedModels: 395,
    successRate: 96.8,
    brotliSupport: 312,
    brotliRate: 74.8,
    avgLatency: '152ms'
  };

  verifiedModels = 395;
  screenSize: ScreenSize = 'lg';
  private subscriptions: Subscription[] = [];

  constructor(
    private renderer: Renderer2,
    private responsiveService: ResponsiveService
  ) {}

  ngOnInit() {
    this.checkSystemThemePreference();
    this.checkSavedTheme();
    this.setupResponsive();
  }

  @HostListener('document:keydown', ['$event'])
  handleKeyboardEvent(event: KeyboardEvent) {
    if ((event.ctrlKey || event.metaKey) && event.shiftKey && event.key === 'D') {
      event.preventDefault();
      this.toggleDarkMode();
    }
  }

  toggleSidenav() {
    this.isSidenavOpen = !this.isSidenavOpen;
  }

  selectTab(tab: number) {
    this.selectedTab = tab;
  }

  toggleDarkMode() {
    this.isDarkMode = !this.isDarkMode;
    this.applyTheme();
    localStorage.setItem('theme', this.isDarkMode ? 'dark' : 'light');
  }

  private applyTheme() {
    if (this.isDarkMode) {
      this.renderer.addClass(document.body, 'dark-theme');
      this.renderer.removeClass(document.body, 'light-theme');
    } else {
      this.renderer.addClass(document.body, 'light-theme');
      this.renderer.removeClass(document.body, 'dark-theme');
    }
  }

  private checkSavedTheme() {
    const savedTheme = localStorage.getItem('theme');
    this.isDarkMode = savedTheme === 'dark';
    this.applyTheme();
  }

  private checkSystemThemePreference() {
    if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
      if (!localStorage.getItem('theme')) {
        this.isDarkMode = true;
        this.applyTheme();
      }
    }
  }

  private listenForSystemThemeChanges() {
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
      if (!localStorage.getItem('theme')) {
        this.isDarkMode = e.matches;
        this.applyTheme();
      }
    });
  }

  getLottieAnimation(name: string): string {
    // Return empty string for now since Lottie is not available
    return '';
  }

  private setupResponsive(): void {
    const screenSizeSub = this.responsiveService.screenSize$.subscribe(size => {
      this.screenSize = size;
      this.updateResponsiveClasses();
    });
    this.subscriptions.push(screenSizeSub);
  }

  private updateResponsiveClasses(): void {
    // Remove existing responsive classes
    ['screen-xs', 'screen-sm', 'screen-md', 'screen-lg', 'screen-xl'].forEach(cls => {
      this.renderer.removeClass(document.body, cls);
    });

    // Add current screen size class
    this.renderer.addClass(document.body, `screen-${this.screenSize}`);

    // Add mobile/desktop classes
    if (this.responsiveService.isMobile()) {
      this.renderer.addClass(document.body, 'mobile-view');
      this.renderer.removeClass(document.body, 'desktop-view');
    } else {
      this.renderer.addClass(document.body, 'desktop-view');
      this.renderer.removeClass(document.body, 'mobile-view');
    }
  }

  ngOnDestroy(): void {
    this.subscriptions.forEach(sub => sub.unsubscribe());
  }
}
