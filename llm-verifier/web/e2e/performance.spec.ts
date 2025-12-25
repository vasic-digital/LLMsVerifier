import { test, expect } from '@playwright/test';

test.describe('Performance Testing', () => {
  test('should load within performance budget', async ({ page }) => {
    // Start performance monitoring
    await page.goto('/');
    
    // Wait for page to be fully loaded
    await page.waitForLoadState('networkidle');
    
    // Get performance metrics
    const performanceMetrics = await page.evaluate(() => {
      const navigationTiming = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming;
      const paintTiming = performance.getEntriesByType('paint')[0] as PerformancePaintTiming;
      
      return {
        loadTime: navigationTiming.loadEventEnd - navigationTiming.navigationStart,
        firstContentfulPaint: paintTiming.startTime,
        domContentLoaded: navigationTiming.domContentLoadedEventEnd - navigationTiming.navigationStart,
        timeToInteractive: performance.now()
      };
    });
    
    console.log('Performance Metrics:', performanceMetrics);
    
    // Performance assertions
    expect(performanceMetrics.loadTime).toBeLessThan(3000); // Load time < 3s
    expect(performanceMetrics.firstContentfulPaint).toBeLessThan(1500); // FCP < 1.5s
    expect(performanceMetrics.domContentLoaded).toBeLessThan(2000); // DCL < 2s
  });

  test('should maintain good performance on mobile', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    const mobileMetrics = await page.evaluate(() => {
      const navigationTiming = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming;
      return {
        loadTime: navigationTiming.loadEventEnd - navigationTiming.navigationStart,
        firstContentfulPaint: performance.getEntriesByType('paint')[0].startTime
      };
    });
    
    console.log('Mobile Performance Metrics:', mobileMetrics);
    
    // Mobile performance assertions
    expect(mobileMetrics.loadTime).toBeLessThan(4000); // Mobile load time < 4s
    expect(mobileMetrics.firstContentfulPaint).toBeLessThan(2000); // Mobile FCP < 2s
  });

  test('should handle navigation efficiently', async ({ page }) => {
    await page.goto('/');
    
    // Measure navigation performance
    const startTime = await page.evaluate(() => performance.now());
    
    await page.click('a[routerlink="/dashboard"]');
    await page.waitForURL('**/dashboard');
    await page.waitForLoadState('networkidle');
    
    const navigationTime = await page.evaluate((start) => performance.now() - start, startTime);
    
    console.log('Navigation Time:', navigationTime);
    
    expect(navigationTime).toBeLessThan(1000); // Navigation should be < 1s
  });

  test('should have optimized bundle size', async ({ page }) => {
    await page.goto('/');
    
    // Get resource sizes
    const resourceSizes = await page.evaluate(() => {
      const resources = performance.getEntriesByType('resource') as PerformanceResourceTiming[];
      return resources.map(resource => ({
        name: resource.name,
        size: resource.transferSize || resource.encodedBodySize || 0,
        duration: resource.responseEnd - resource.requestStart
      }));
    });
    
    const totalSize = resourceSizes.reduce((sum, resource) => sum + resource.size, 0);
    const jsResources = resourceSizes.filter(r => r.name.includes('.js'));
    const jsSize = jsResources.reduce((sum, resource) => sum + resource.size, 0);
    
    console.log('Total Resource Size:', totalSize, 'bytes');
    console.log('JavaScript Bundle Size:', jsSize, 'bytes');
    console.log('Number of JS Resources:', jsResources.length);
    
    // Bundle size assertions
    expect(totalSize).toBeLessThan(500000); // Total size < 500KB
    expect(jsSize).toBeLessThan(300000); // JS size < 300KB
    expect(jsResources.length).toBeLessThan(10); // < 10 JS resources
  });
});