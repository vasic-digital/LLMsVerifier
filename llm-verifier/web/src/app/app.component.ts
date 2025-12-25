import { Component, OnInit, Renderer2, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule, Routes } from '@angular/router';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatSidenavModule } from '@angular/material/sidenav';
import { MatListModule } from '@angular/material/list';
import { MatListItemModule } from '@angular/material/list';
import { MatCardModule } from '@angular/material/card';
import { MatTableModule } from '@angular/material/table';
import { MatTabsModule } from '@angular/material/tabs';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatChipsModule } from '@angular/material/chips';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatSnackBarModule } from '@angular/material/snack-bar';
import { MatMenuModule } from '@angular/material/menu';
import { MatSelectModule } from '@angular/material/select';

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
    MatListItemModule,
    MatCardModule,
    MatTableModule,
    MatTabsModule,
    MatProgressSpinnerModule,
    MatProgressBarModule,
    MatChipsModule,
    MatTooltipModule,
    MatSnackBarModule,
    MatMenuModule,
    MatSelectModule
  ],
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss']
})
export class AppComponent implements OnInit {
  title = 'LLM Verifier';
  isDarkMode = false;
  selectedTab = 0;
  selectedProvider = 'openai';
  
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

  constructor(private renderer: Renderer2) {}

  ngOnInit() {
    this.checkSystemThemePreference();
    this.checkSavedTheme();
  }

  @HostListener('document:keydown', ['$event'])
  handleKeyboardEvent(event: KeyboardEvent) {
    if ((event.ctrlKey || event.metaKey) && event.shiftKey && event.key === 'D')) {
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
}
