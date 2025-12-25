import { test, expect } from '@playwright/test';

test.describe('Chart Components', () => {
  test('should display chart components on dashboard', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Check for chart containers
    await expect(page.locator('app-chart')).toBeVisible();
    
    // Check chart headers
    await expect(page.locator('h3:has-text("Verification Trends")')).toBeVisible();
    await expect(page.locator('h3:has-text("Provider Distribution")')).toBeVisible();
    await expect(page.locator('h3:has-text("Score Distribution")')).toBeVisible();
    await expect(page.locator('h3:has-text("Performance Metrics")')).toBeVisible();
  });

  test('should handle chart interactions', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Check for chart action buttons
    const chartActions = page.locator('.chart-actions button');
    await expect(chartActions).toBeVisible();
    
    // Test export functionality
    const exportButtons = page.locator('button[title="Export as Image"]');
    await expect(exportButtons).toBeVisible();
    
    // Test refresh functionality
    const refreshButtons = page.locator('button[title="Reset View"]');
    await expect(refreshButtons).toBeVisible();
    
    // Click on chart actions (they should not cause errors)
    await exportButtons.first().click();
    await refreshButtons.first().click();
    
    // Ensure no errors occurred
    const consoleErrors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    
    await expect.poll(() => consoleErrors.length).toBe(0);
  });

  test('should display chart legends', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Check for chart legends
    const legends = page.locator('.chart-legend');
    await expect(legends).toBeVisible();
    
    // Check legend items
    const legendItems = page.locator('.legend-item');
    await expect(legendItems).toBeVisible();
    
    // Check legend colors
    const legendColors = page.locator('.legend-color');
    await expect(legendColors).toBeVisible();
    
    // Check legend labels
    const legendLabels = page.locator('.legend-label');
    await expect(legendLabels).toBeVisible();
  });

  test('should handle responsive chart sizing', async ({ page }) => {
    const viewports = [
      { width: 375, height: 667 },  // Mobile
      { width: 768, height: 1024 }, // Tablet
      { width: 1280, height: 720 }, // Desktop
    ];
    
    for (const viewport of viewports) {
      await page.setViewportSize(viewport);
      await page.goto('/dashboard');
      
      // Check chart containers are visible
      const charts = page.locator('app-chart');
      await expect(charts).toBeVisible();
      
      // Check chart sizing
      const chart = charts.first();
      const box = await chart.boundingBox();
      
      if (box) {
        expect(box.width).toBeGreaterThan(100);
        expect(box.height).toBeGreaterThan(100);
      }
      
      // Ensure no horizontal overflow
      const bodyWidth = await page.evaluate(() => document.body.scrollWidth);
      expect(bodyWidth).toBeLessThanOrEqual(viewport.width);
    }
  });

  test('should display chart fallbacks when no data', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Check for "no data" fallback states
    const noDataMessages = page.locator('.no-data');
    await expect(noDataMessages).toBeVisible();
    
    // Check fallback content
    await expect(page.locator('mat-icon:has-text("bar_chart")')).toBeVisible();
    await expect(page.locator('p:has-text("No data available")')).toBeVisible();
  });

  test('should handle chart accessibility', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Check for proper ARIA labels and roles
    const charts = page.locator('app-chart');
    
    const chart = charts.first();
    
    // Check if charts have accessible names
    const accessibility = await chart.evaluate(el => {
      const canvas = el.querySelector('canvas');
      return {
        hasCanvas: !!canvas,
        canvasAriaLabel: canvas?.getAttribute('aria-label'),
        canvasRole: canvas?.getAttribute('role')
      };
    });
    
    expect(accessibility.hasCanvas).toBe(true);
    
    // Check chart action accessibility
    const exportButtons = page.locator('button[title="Export as Image"]');
    await expect(exportButtons).toBeVisible();
    
    const refreshButtons = page.locator('button[title="Reset View"]');
    await expect(refreshButtons).toBeVisible();
  });
});