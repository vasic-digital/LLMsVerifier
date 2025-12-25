import { test, expect } from '@playwright/test';

test.describe('LLM Verifier Basic Functionality', () => {
  test('should load the main application', async ({ page }) => {
    await page.goto('/');
    
    // Check if the app loads successfully
    await expect(page.locator('app-root')).toBeVisible();
    
    // Check for main title
    await expect(page.locator('h1:has-text("LLM Verifier")')).toBeVisible();
    
    // Check for basic stats display
    await expect(page.locator('.stat-card')).toBeVisible();
    await expect(page.locator('.stat-value')).toBeVisible();
  });

  test('should navigate to dashboard page', async ({ page }) => {
    await page.goto('/');
    
    // Navigate to dashboard
    await page.click('a[routerlink="/dashboard"]');
    await page.waitForURL('**/dashboard');
    
    // Check dashboard loads
    await expect(page.locator('app-dashboard')).toBeVisible();
  });

  test('should toggle dark mode', async ({ page }) => {
    await page.goto('/');
    
    // Check initial theme
    const body = page.locator('body');
    const initialTheme = await body.getAttribute('class');
    
    // Toggle dark mode using keyboard shortcut
    await page.keyboard.press('Control+Shift+D');
    
    // Check if theme changed
    const newTheme = await body.getAttribute('class');
    expect(newTheme).not.toBe(initialTheme);
  });

  test('should display responsive design on mobile', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/');
    
    // Check responsive classes
    const body = page.locator('body');
    const bodyClass = await body.getAttribute('class');
    expect(bodyClass).toMatch(/screen-(xs|sm|md|lg|xl)/);
    
    // Check mobile navigation
    await expect(page.locator('.mobile-nav-toggle')).toBeVisible();
  });

  test('should handle basic navigation', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Check dashboard content
    await expect(page.locator('h1:has-text("LLM Verifier Dashboard")')).toBeVisible();
    
    // Navigate back to home
    await page.click('a[routerlink="/"]');
    await page.waitForURL('**/');
    
    // Check home page loads
    await expect(page.locator('h1:has-text("LLM Verifier")')).toBeVisible();
  });

  test('should display loading states', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Mock slow API response
    await page.route('**/api/**', route => {
      // Add delay to simulate slow network
      setTimeout(() => route.continue(), 2000);
    });
    
    // Trigger refresh
    await page.click('button:has-text("Refresh")');
    
    // Check loading indicator appears
    await expect(page.locator('mat-spinner')).toBeVisible({ timeout: 3000 });
  });

  test('should handle basic error states', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Mock API error
    await page.route('**/api/**', route => {
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Server error' })
      });
    });
    
    // Trigger refresh
    await page.click('button:has-text("Refresh")');
    
    // Check error handling
    await expect(page.locator('.error')).toBeVisible({ timeout: 5000 });
  });
});