import { test, expect } from '@playwright/test';

test.describe('Mobile Responsiveness', () => {
  test('should display mobile navigation on small screens', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    
    await page.goto('/');
    
    // Check if mobile navigation toggle is visible
    const mobileToggle = page.locator('.mobile-nav-toggle');
    await expect(mobileToggle).toBeVisible();
    
    // Check if desktop sidenav is hidden
    const desktopSidenav = page.locator('.mat-sidenav');
    await expect(desktopSidenav).toBeHidden();
  });

  test('should toggle mobile navigation', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/');
    
    const mobileToggle = page.locator('.mobile-nav-toggle');
    const mobileNav = page.locator('.mobile-nav');
    const backdrop = page.locator('.mobile-nav-backdrop');
    
    // Initially closed
    await expect(mobileNav).not.toHaveClass(/nav-open/);
    await expect(backdrop).not.toHaveClass(/backdrop-open/);
    
    // Open navigation
    await mobileToggle.click();
    await expect(mobileNav).toHaveClass(/nav-open/);
    await expect(backdrop).toHaveClass(/backdrop-open/);
    
    // Close via backdrop
    await backdrop.click();
    await expect(mobileNav).not.toHaveClass(/nav-open/);
    await expect(backdrop).not.toHaveClass(/backdrop-open/);
  });

  test('should adapt layout for tablet screens', async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 1024 });
    await page.goto('/dashboard');
    
    // Check responsive grid layout
    const statsGrid = page.locator('.stats-grid');
    await expect(statsGrid).toBeVisible();
    
    // Check if metrics grid adapts
    const metricsGrid = page.locator('.metrics-grid');
    await expect(metricsGrid).toBeVisible();
  });

  test('should display touch-friendly buttons on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/dashboard');
    
    // Check button sizes
    const buttons = page.locator('.mat-button, .mat-raised-button');
    const count = await buttons.count();
    
    for (let i = 0; i < count; i++) {
      const button = buttons.nth(i);
      const box = await button.boundingBox();
      
      if (box) {
        // Minimum touch target size
        expect(box.width).toBeGreaterThanOrEqual(44);
        expect(box.height).toBeGreaterThanOrEqual(44);
      }
    }
  });

  test('should handle keyboard navigation on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/');
    
    // Open mobile nav
    await page.locator('.mobile-nav-toggle').click();
    
    // Test escape key to close
    await page.keyboard.press('Escape');
    await expect(page.locator('.mobile-nav')).not.toHaveClass(/nav-open/);
  });

  test('should maintain functionality across breakpoints', async ({ page }) => {
    const breakpoints = [
      { width: 320, height: 568 },  // iPhone SE
      { width: 375, height: 667 },  // iPhone 8
      { width: 414, height: 896 },  // iPhone XR
      { width: 768, height: 1024 }, // iPad
      { width: 1024, height: 768 }, // iPad landscape
      { width: 1280, height: 720 }, // Desktop
    ];
    
    for (const viewport of breakpoints) {
      await page.setViewportSize(viewport);
      await page.goto('/dashboard');
      
      // Basic functionality checks
      await expect(page.locator('h1')).toBeVisible();
      await expect(page.locator('.stats-grid')).toBeVisible();
      
      // Refresh functionality
      const refreshButton = page.locator('button:has-text("Refresh"), button:has(svg[data-icon="refresh"])').first();
      await expect(refreshButton).toBeVisible();
      
      // Ensure no horizontal scrolling
      const bodyWidth = await page.evaluate(() => document.body.scrollWidth);
      const viewportWidth = viewport.width;
      expect(bodyWidth).toBeLessThanOrEqual(viewportWidth);
    }
  });
});