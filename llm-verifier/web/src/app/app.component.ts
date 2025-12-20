import { Component, OnInit, Renderer2, HostListener } from '@angular/core';

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss']
})
export class AppComponent implements OnInit {
  title = 'LLM Verifier Web';
  isDarkMode = false;

  constructor(private renderer: Renderer2) {}

  ngOnInit() {
    // Check for saved theme preference
    const savedTheme = localStorage.getItem('theme');
    this.isDarkMode = savedTheme === 'dark';
    this.applyTheme();

    // Check system preference
    this.checkSystemThemePreference();
  }

  @HostListener('document:keydown', ['$event'])
  handleKeyboardEvent(event: KeyboardEvent) {
    // Toggle dark mode with Ctrl/Cmd + Shift + D
    if ((event.ctrlKey || event.metaKey) && event.shiftKey && event.key === 'D') {
      event.preventDefault();
      this.toggleDarkMode();
    }
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

  private checkSystemThemePreference() {
    // Check if user has system dark mode preference
    if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
      // Only apply if no saved preference
      if (!localStorage.getItem('theme')) {
        this.isDarkMode = true;
        this.applyTheme();
      }
    }

    // Listen for system theme changes
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
      if (!localStorage.getItem('theme')) {
        this.isDarkMode = e.matches;
        this.applyTheme();
      }
    });
  }
}