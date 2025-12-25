import { Component, OnInit, OnDestroy, HostListener } from '@angular/core';
import { Router } from '@angular/router';
import { ResponsiveService, ScreenSize } from '../../services/responsive.service';
import { Subscription } from 'rxjs';

interface NavItem {
  label: string;
  icon: string;
  route: string;
  badge?: number;
}

@Component({
  selector: 'app-mobile-nav',
  templateUrl: './mobile-nav.component.html',
  styleUrls: ['./mobile-nav.component.scss']
})
export class MobileNavComponent implements OnInit, OnDestroy {
  isOpen = false;
  screenSize: ScreenSize = 'lg';
  private subscriptions: Subscription[] = [];

  navItems: NavItem[] = [
    { label: 'Dashboard', icon: 'dashboard', route: '/dashboard' },
    { label: 'Models', icon: 'smart_toy', route: '/models' },
    { label: 'Providers', icon: 'dns', route: '/providers' },
    { label: 'Verification', icon: 'verified', route: '/verification', badge: 5 },
    { label: 'Monitoring', icon: 'monitor_heart', route: '/monitoring' },
    { label: 'Settings', icon: 'settings', route: '/settings' }
  ];

  constructor(
    private router: Router,
    private responsiveService: ResponsiveService
  ) {}

  ngOnInit(): void {
    // Subscribe to screen size changes
    const screenSizeSub = this.responsiveService.screenSize$.subscribe(size => {
      this.screenSize = size;
      // Auto-close nav when switching to larger screens
      if (!this.responsiveService.isMobile() && this.isOpen) {
        this.closeNav();
      }
    });

    this.subscriptions.push(screenSizeSub);
  }

  ngOnDestroy(): void {
    this.subscriptions.forEach(sub => sub.unsubscribe());
  }

  @HostListener('document:keydown.escape', ['$event'])
  handleEscape(event: KeyboardEvent): void {
    if (this.isOpen) {
      this.closeNav();
      event.preventDefault();
    }
  }

  @HostListener('document:click', ['$event'])
  handleClickOutside(event: Event): void {
    const target = event.target as HTMLElement;
    if (this.isOpen && !target.closest('.mobile-nav') && !target.closest('.mobile-nav-toggle')) {
      this.closeNav();
    }
  }

  toggleNav(): void {
    this.isOpen = !this.isOpen;
    if (this.isOpen) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }
  }

  closeNav(): void {
    this.isOpen = false;
    document.body.style.overflow = '';
  }

  navigateTo(route: string): void {
    this.router.navigate([route]);
    this.closeNav();
  }

  // Check if current route matches nav item
  isActiveRoute(route: string): boolean {
    return this.router.url === route || this.router.url.startsWith(route + '/');
  }

  // Get mobile-specific classes
  getNavClasses(): string {
    const classes = ['mobile-nav'];
    
    if (this.isOpen) {
      classes.push('nav-open');
    }
    
    if (this.screenSize === 'xs') {
      classes.push('nav-xs');
    } else if (this.screenSize === 'sm') {
      classes.push('nav-sm');
    }
    
    return classes.join(' ');
  }

  // Get backdrop classes
  getBackdropClasses(): string {
    const classes = ['mobile-nav-backdrop'];
    
    if (this.isOpen) {
      classes.push('backdrop-open');
    }
    
    return classes.join(' ');
  }
}