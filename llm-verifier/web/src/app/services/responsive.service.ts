import { Injectable, OnDestroy } from '@angular/core';
import { BehaviorSubject, Observable } from 'rxjs';
import { distinctUntilChanged } from 'rxjs/operators';

export type ScreenSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl';

@Injectable({
  providedIn: 'root'
})
export class ResponsiveService implements OnDestroy {
  private screenSizeSubject = new BehaviorSubject<ScreenSize>('lg');
  public screenSize$: Observable<ScreenSize> = this.screenSizeSubject.asObservable().pipe(distinctUntilChanged());

  private breakpoints = {
    xs: 0,
    sm: 576,
    md: 768,
    lg: 992,
    xl: 1200
  };

  private resizeListener: () => void;

  constructor() {
    this.updateScreenSize();
    this.resizeListener = this.throttle(() => this.updateScreenSize(), 100);
    window.addEventListener('resize', this.resizeListener);
  }

  ngOnDestroy(): void {
    window.removeEventListener('resize', this.resizeListener);
  }

  private updateScreenSize(): void {
    const width = window.innerWidth;
    let size: ScreenSize = 'lg';

    if (width < this.breakpoints.sm) {
      size = 'xs';
    } else if (width < this.breakpoints.md) {
      size = 'sm';
    } else if (width < this.breakpoints.lg) {
      size = 'md';
    } else if (width < this.breakpoints.xl) {
      size = 'lg';
    } else {
      size = 'xl';
    }

    this.screenSizeSubject.next(size);
  }

  private throttle(func: () => void, delay: number): () => void {
    let timeoutId: ReturnType<typeof setTimeout> | null = null;
    return () => {
      if (!timeoutId) {
        timeoutId = setTimeout(() => {
          func();
          timeoutId = null;
        }, delay);
      }
    };
  }

  // Public methods to check screen size
  isMobile(): boolean {
    const size = this.screenSizeSubject.getValue();
    return size === 'xs' || size === 'sm';
  }

  isTablet(): boolean {
    return this.screenSizeSubject.getValue() === 'md';
  }

  isDesktop(): boolean {
    const size = this.screenSizeSubject.getValue();
    return size === 'lg' || size === 'xl';
  }

  isSmallScreen(): boolean {
    return this.isMobile() || this.isTablet();
  }

  // Get current screen size
  getScreenSize(): ScreenSize {
    return this.screenSizeSubject.getValue();
  }

  // Check if screen is larger than a specific breakpoint
  isLargerThan(size: ScreenSize): boolean {
    const currentSize = this.screenSizeSubject.getValue();
    const sizes: ScreenSize[] = ['xs', 'sm', 'md', 'lg', 'xl'];
    return sizes.indexOf(currentSize) > sizes.indexOf(size);
  }

  // Check if screen is smaller than a specific breakpoint
  isSmallerThan(size: ScreenSize): boolean {
    const currentSize = this.screenSizeSubject.getValue();
    const sizes: ScreenSize[] = ['xs', 'sm', 'md', 'lg', 'xl'];
    return sizes.indexOf(currentSize) < sizes.indexOf(size);
  }

  // Get responsive grid columns
  getGridColumns(defaultColumns: number): number {
    const size = this.screenSizeSubject.getValue();
    
    switch (size) {
      case 'xs': return 1;
      case 'sm': return Math.min(2, defaultColumns);
      case 'md': return Math.min(3, defaultColumns);
      case 'lg': return Math.min(4, defaultColumns);
      case 'xl': return defaultColumns;
      default: return defaultColumns;
    }
  }

  // Get responsive font size
  getFontSize(baseSize: number): number {
    const size = this.screenSizeSubject.getValue();
    
    switch (size) {
      case 'xs': return baseSize * 0.8;
      case 'sm': return baseSize * 0.9;
      case 'md': return baseSize;
      case 'lg': return baseSize * 1.1;
      case 'xl': return baseSize * 1.2;
      default: return baseSize;
    }
  }

  // Get responsive spacing
  getSpacing(baseSpacing: number): number {
    const size = this.screenSizeSubject.getValue();
    
    switch (size) {
      case 'xs': return baseSpacing * 0.7;
      case 'sm': return baseSpacing * 0.8;
      case 'md': return baseSpacing * 0.9;
      case 'lg': return baseSpacing;
      case 'xl': return baseSpacing * 1.1;
      default: return baseSpacing;
    }
  }
}