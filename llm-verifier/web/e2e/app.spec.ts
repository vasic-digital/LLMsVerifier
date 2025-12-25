import { test, expect } from '@playwright/test';

test.describe('LLM Verifier Web Application', () => {
  test('should display the main dashboard', async ({ page }) => {
    await page.goto('/');
    
    // Check if the app loads successfully
    await expect(page.locator('app-root')).toBeVisible();
    
    // Check for main elements
    await expect(page.locator('h1:has-text("LLM Verifier")')).toBeVisible();
    await expect(page.locator('.tagline:has-text("Automated LLM Verification & Testing Platform")')).toBeVisible();
    
    // Check for stats cards
    await expect(page.locator('.stat-card')).toHaveCount(6);
    await expect(page.locator('.stat-value')).toHaveCount(6);
  });

  test('should navigate to dashboard', async ({ page }) => {
    await page.goto('/');
    
    // Navigate to dashboard
    await page.click('a[routerlink="/dashboard"]');
    await page.waitForURL('**/dashboard');
    
    // Check dashboard content
    await expect(page.locator('h1:has-text("LLM Verifier Dashboard")')).toBeVisible();
    await expect(page.locator('button:has-text("Refresh")')).toBeVisible();
    
    // Check for stats grid
    await expect(page.locator('.stats-grid')).toBeVisible();
    await expect(page.locator('.stats-grid mat-card')).toHaveCount(6);
  });

  test('should display dashboard metrics', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Check for advanced metrics dashboard
    await expect(page.locator('app-dashboard-metrics')).toBeVisible();
    await expect(page.locator('h2:has-text("Advanced Metrics Dashboard")')).toBeVisible();
    
    // Check for metrics grid
    await expect(page.locator('.metrics-grid')).toBeVisible();
    await expect(page.locator('.metric-card')).toHaveCountAtLeast(4);
  });

  test('should handle refresh functionality', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Click refresh button
    const refreshButton = page.locator('button:has-text("Refresh")');
    await refreshButton.click();
    
    // Check if loading state appears
    await expect(page.locator('mat-spinner')).toBeVisible({ timeout: 1000 });
    
    // Wait for loading to complete
    await expect(page.locator('mat-spinner')).toBeHidden({ timeout: 5000 });
  });

  test('should toggle dark mode', async ({ page }) => {
    await page.goto('/');
    
    // Check initial theme
    const body = page.locator('body');
    const initialTheme = await body.getAttribute('class');
    
    // Toggle dark mode
    await page.click('button:has-text("Dark Mode")');
    
    // Check if theme changed
    await expect(body).toHaveClass(/dark-theme/);
    
    // Toggle back to light mode
    await page.click('button:has-text("Light Mode")');
    await expect(body).toHaveClass(/light-theme/);
  });

  test('should handle keyboard shortcuts', async ({ page }) => {
    await page.goto('/');
    
    // Test dark mode keyboard shortcut
    await page.keyboard.press('Control+Shift+D');
    await expect(page.locator('body')).toHaveClass(/dark-theme/);
    
    // Test escape key for mobile nav (if visible)
    await page.keyboard.press('Escape');
  });

  test('should display responsive design on mobile', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/');
    
    // Check mobile-specific elements
    await expect(page.locator('.mobile-nav-toggle')).toBeVisible();
    await expect(page.locator('.mat-sidenav')).toBeHidden();
    
    // Check responsive layout
    await expect(page.locator('.header-content')).toBeVisible();
    await expect(page.locator('.stats-section')).toBeVisible();
    
    // Check touch-friendly sizing
    const buttons = page.locator('.mat-button, .mat-raised-button');
    const count = await buttons.count();
    
    for (let i = 0; i < count; i++) {
      const button = buttons.nth(i);
      const box = await button.boundingBox();
      
      if (box) {
        expect(box.width).toBeGreaterThanOrEqual(44);
        expect(box.height).toBeGreaterThanOrEqual(44);
      }
    }
  });

  test('should handle mobile navigation', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/');
    
    // Open mobile navigation
    await page.click('.mobile-nav-toggle');
    await expect(page.locator('.mobile-nav')).toHaveClass(/nav-open/);
    await expect(page.locator('.mobile-nav-backdrop')).toHaveClass(/backdrop-open/);
    
    // Navigate using mobile nav
    await page.click('.mobile-nav .nav-item:has-text("Dashboard")');
    await page.waitForURL('**/dashboard');
    await expect(page.locator('.mobile-nav')).not.toHaveClass(/nav-open/);
  });

  test('should display real-time events', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Check for real-time events section
    await expect(page.locator('h2:has-text("Real-time Events")')).toBeVisible();
    await expect(page.locator('.events-section')).toBeVisible();
    
    // Check WebSocket connection status
    const wsStatus = page.locator('.status-online, .status-disconnected');
    await expect(wsStatus).toBeVisible();
  });

  test('should handle error states gracefully', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Simulate API error by blocking requests
    await page.route('**/api/**', route => {
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Server error' })
      });
    });
    
    // Trigger refresh to cause error
    await page.click('button:has-text("Refresh")');
    
    // Check error handling
    await expect(page.locator('.error')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('.error')).toContainText('Failed to load');
  });

  test('should maintain state across navigation', async ({ page }) => {
    await page.goto('/');
    
    // Set dark mode
    await page.click('button:has-text("Dark Mode")');
    
    // Navigate away and back
    await page.click('a[routerlink="/dashboard"]');
    await page.waitForURL('**/dashboard');
    
    await page.goBack();
    await page.waitForURL('**/');
    
    // Check if theme persists
    await expect(page.locator('body')).toHaveClass(/dark-theme/);
  });

  test('should handle different screen sizes', async ({ page }) => {
    const breakpoints = [
      { width: 320, height: 568 },  // iPhone SE
      { width: 375, height: 667 },  // iPhone 8
      { width: 414, height: 896 },  // iPhone XR
      { width: 768, height: 1024 }, // iPad
      { width: 1024, height: 768 }, // iPad landscape
      { width: 1280, height: 720 }, // Desktop
      { width: 1920, height: 1080 }, // Large desktop
    ];
    
    for (const viewport of breakpoints) {
      await page.setViewportSize(viewport);
      await page.goto('/dashboard');
      
      // Basic functionality checks
      await expect(page.locator('h1')).toBeVisible();
      await expect(page.locator('.stats-grid')).toBeVisible();
      
      // Ensure no horizontal scrolling
      const bodyWidth = await page.evaluate(() => document.body.scrollWidth);
      expect(bodyWidth).toBeLessThanOrEqual(viewport.width);
      
      // Check responsive classes
      const body = page.locator('body');
      const bodyClass = await body.getAttribute('class');
      expect(bodyClass).toMatch(/screen-(xs|sm|md|lg|xl)/);
    }
  });

  test('should display loading states properly', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Slow down network to see loading states
    await page.route('**/api/**', route => {
      // Add delay to simulate slow network
      setTimeout(() => route.continue(), 2000);
    });
    
    // Trigger refresh
    await page.click('button:has-text("Refresh")');
    
    // Check loading indicator
    await expect(page.locator('mat-spinner')).toBeVisible();
    await expect(page.locator('.loading')).toBeVisible();
    
    // Wait for loading to complete
    await expect(page.locator('mat-spinner')).toBeHidden({ timeout: 10000 });
  });
});