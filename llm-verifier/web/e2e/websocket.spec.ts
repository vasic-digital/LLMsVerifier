import { test, expect } from '@playwright/test';

test.describe('WebSocket Functionality', () => {
  test('should display WebSocket connection status', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Check WebSocket status display
    const wsStatus = page.locator('.status-online, .status-disconnected');
    await expect(wsStatus).toBeVisible();
    
    // Check status text
    const statusText = await wsStatus.textContent();
    expect(statusText).toMatch(/(Connected|Disconnected)/);
    
    // Check WebSocket connection controls
    const wsButtons = page.locator('button[mat-icon-button]');
    await expect(wsButtons).toBeVisible();
  });

  test('should handle WebSocket connection/disconnection', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Check initial state
    const connectButton = page.locator('button:has(mat-icon:has-text("link"))');
    const disconnectButton = page.locator('button:has(mat-icon:has-text("link_off"))');
    
    // One of these should be visible based on connection state
    const isConnected = await connectButton.isHidden();
    
    if (isConnected) {
      // Currently connected, test disconnection
      await disconnectButton.click();
      await expect(connectButton).toBeVisible({ timeout: 5000 });
    } else {
      // Currently disconnected, test connection
      await connectButton.click();
      await expect(disconnectButton).toBeVisible({ timeout: 5000 });
    }
  });

  test('should display real-time events', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Check real-time events section
    await expect(page.locator('h2:has-text("Real-time Events")')).toBeVisible();
    await expect(page.locator('.events-section')).toBeVisible();
    
    // Check events list
    const eventsList = page.locator('.events-list');
    await expect(eventsList).toBeVisible();
    
    // Check event items (they might be empty if no events)
    const eventItems = page.locator('.event-item');
    const count = await eventItems.count();
    
    if (count > 0) {
      // Check event structure
      await expect(eventItems.first().locator('.event-type')).toBeVisible();
      await expect(eventItems.first().locator('.event-time')).toBeVisible();
      await expect(eventItems.first().locator('.event-message')).toBeVisible();
    }
  });

  test('should handle WebSocket reconnection', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Simulate network disruption
    await page.route('**/ws**', route => {
      // Block WebSocket connections
      route.abort();
    });
    
    // Try to connect (should fail)
    const connectButton = page.locator('button:has(mat-icon:has-text("link"))');
    if (await connectButton.isVisible()) {
      await connectButton.click();
    }
    
    // Check error state
    await expect(page.locator('.status-disconnected')).toBeVisible({ timeout: 5000 });
    
    // Restore connection
    await page.unroute('**/ws**');
    
    // Try reconnection
    if (await connectButton.isVisible()) {
      await connectButton.click();
      await expect(page.locator('.status-online')).toBeVisible({ timeout: 10000 });
    }
  });

  test('should update dashboard on WebSocket events', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Mock WebSocket events
    await page.evaluate(() => {
      // Simulate WebSocket message
      const event = new CustomEvent('websocketMessage', {
        detail: {
          type: 'model.verified',
          message: 'Model verification completed',
          timestamp: new Date().toISOString(),
          severity: 'success'
        }
      });
      window.dispatchEvent(event);
    });
    
    // Check if event appears in the list
    await expect(page.locator('.event-item')).toBeVisible({ timeout: 5000 });
    
    // Check event content
    await expect(page.locator('.event-message:has-text("Model verification completed")')).toBeVisible();
  });

  test('should handle WebSocket heartbeat', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Mock heartbeat event
    await page.evaluate(() => {
      const event = new CustomEvent('websocketMessage', {
        detail: {
          type: 'heartbeat',
          timestamp: new Date().toISOString(),
          message: 'Heartbeat received'
        }
      });
      window.dispatchEvent(event);
    });
    
    // Heartbeat should not create visible events but maintain connection
    const wsStatus = page.locator('.status-online');
    await expect(wsStatus).toBeVisible({ timeout: 5000 });
  });

  test('should display WebSocket connection errors', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Mock connection error
    await page.evaluate(() => {
      const event = new CustomEvent('websocketError', {
        detail: {
          message: 'Connection failed',
          code: 'ECONNREFUSED'
        }
      });
      window.dispatchEvent(event);
    });
    
    // Check error display
    await expect(page.locator('.error')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('.error')).toContainText('Connection failed');
    
    // Error should auto-dismiss
    await expect(page.locator('.error')).toBeHidden({ timeout: 10000 });
  });

  test('should maintain WebSocket state across navigation', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Get initial connection state
    const initialStatus = await page.locator('.status-online, .status-disconnected').textContent();
    
    // Navigate away
    await page.click('a[routerlink="/"]');
    await page.waitForURL('**/');
    
    // Navigate back
    await page.click('a[routerlink="/dashboard"]');
    await page.waitForURL('**/dashboard');
    
    // Check if WebSocket state is maintained
    const finalStatus = await page.locator('.status-online, .status-disconnected').textContent();
    expect(finalStatus).toBe(initialStatus);
  });
});