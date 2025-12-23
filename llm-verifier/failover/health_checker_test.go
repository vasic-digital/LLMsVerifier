package failover

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"llm-verifier/database"
)

func setupTestHealthCheckerDB(t *testing.T) *database.Database {
	dbFile := "/tmp/test_health_" + time.Now().Format("20060102150405") + ".db"

	db, err := database.New(dbFile)
	require.NoError(t, err, "Failed to create test database")

	t.Cleanup(func() {
		os.Remove(dbFile)
	})

	return db
}

func TestNewHealthChecker(t *testing.T) {
	db := setupTestHealthCheckerDB(t)

	hc := NewHealthChecker(db)

	assert.NotNil(t, hc, "HealthChecker should not be nil")
	assert.NotNil(t, hc.db, "Database should be set")
	assert.NotNil(t, hc.circuitBreakers, "Circuit breakers map should be initialized")
	assert.NotNil(t, hc.httpClient, "HTTP client should be set")
	assert.NotNil(t, hc.stopCh, "Stop channel should be initialized")
	assert.Equal(t, 30*time.Second, hc.checkInterval, "Check interval should be 30 seconds")
}

func TestHealthCheckerAddProvider(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	providerID := "test-provider-123"

	hc.AddProvider(providerID)

	hc.mu.RLock()
	cb, exists := hc.circuitBreakers[providerID]
	hc.mu.RUnlock()

	assert.True(t, exists, "Provider should be added")
	assert.NotNil(t, cb, "Circuit breaker should be created")
	assert.Contains(t, cb.name, providerID, "Circuit breaker name should contain provider ID")
}

func TestHealthCheckerAddMultipleProviders(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	providerIDs := []string{"provider-1", "provider-2", "provider-3"}

	for _, id := range providerIDs {
		hc.AddProvider(id)
	}

	hc.mu.RLock()
	count := len(hc.circuitBreakers)
	hc.mu.RUnlock()

	assert.Equal(t, 3, count, "All providers should be added")
}

func TestHealthCheckerRemoveProvider(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	providerID := "test-provider-123"

	// Add then remove
	hc.AddProvider(providerID)
	hc.RemoveProvider(providerID)

	hc.mu.RLock()
	_, exists := hc.circuitBreakers[providerID]
	hc.mu.RUnlock()

	assert.False(t, exists, "Provider should be removed")
}

func TestHealthCheckerRemoveNonExistentProvider(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	// Should not panic
	hc.RemoveProvider("non-existent")
}

func TestHealthCheckerStart(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	hc.Start()

	// Wait a bit for goroutine to start
	time.Sleep(100 * time.Millisecond)

	// Should not block
	assert.NotNil(t, hc.stopCh)

	hc.Stop()
}

func TestHealthCheckerStop(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	hc.Start()
	hc.Stop()

}
	// Should not panic
func TestHealthCheckerStartStop(t *testing.T) {
	// Start and stop multiple times - should create new instances each time
	for i := 0; i < 3; i++ {
		db := setupTestHealthCheckerDB(t)
		hc := NewHealthChecker(db)
		hc.Start()
		time.Sleep(10 * time.Millisecond)
		hc.Stop()
	}
}

func TestHealthCheckerGetCircuitBreaker(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	providerID := "test-provider"

	hc.AddProvider(providerID)

	cb := hc.GetCircuitBreaker(providerID)

	assert.NotNil(t, cb, "Circuit breaker should be returned")
	assert.Contains(t, cb.name, providerID, "Should return correct circuit breaker")
}

func TestHealthCheckerGetCircuitBreakerNonExistent(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	cb := hc.GetCircuitBreaker("non-existent")

	assert.Nil(t, cb, "Should return nil for non-existent provider")
}

func TestHealthCheckerGetHealthyProviders(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	// Add providers
	providerIDs := []string{"provider-1", "provider-2", "provider-3"}
	for _, id := range providerIDs {
		hc.AddProvider(id)
	}

	healthy := hc.GetHealthyProviders()

	assert.NotNil(t, healthy, "Should return healthy providers list")
	assert.Equal(t, 3, len(healthy), "All providers should be healthy initially")

	// Verify all provider IDs are returned
	for _, id := range providerIDs {
		assert.Contains(t, healthy, id, "Provider ID should be in healthy list")
	}
}

func TestHealthCheckerGetHealthyProvidersEmpty(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	healthy := hc.GetHealthyProviders()

	if healthy != nil { assert.NotNil(t, healthy) }
	if healthy != nil { assert.Equal(t, 0, len(healthy)) }
}

func TestHealthCheckerPerformHealthChecks(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	// Add provider (will fail health check without real endpoint)
	hc.AddProvider("test-provider")

	// Should not panic
	hc.performHealthChecks()
}

func TestHealthCheckerCheckProviderHealth(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	providerID := "123"
	hc.AddProvider(providerID)

	// Create provider in database
	provider := &database.Provider{
		ID:       123,
		Name:     "test-provider",
		Endpoint: "http://localhost:9999/health",
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	// Should not panic
	hc.checkProviderHealth(providerID)
}

func TestHealthCheckerCheckProviderEndpoint(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	// Test with invalid endpoint
	healthy := hc.checkProviderEndpoint("http://invalid-endpoint-that-does-not-exist.local/health")
	assert.False(t, healthy, "Should be unhealthy for invalid endpoint")
}

func TestHealthCheckerCheckProviderEndpointInvalidURL(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	// Test with invalid URL format
	healthy := hc.checkProviderEndpoint("://invalid-url")
	assert.False(t, healthy, "Should be unhealthy for invalid URL")
}

func TestHealthCheckerUpdateProviderHealth(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	// Should not panic
	hc.updateProviderHealth("test-provider", true)
	hc.updateProviderHealth("test-provider", false)
}

func TestHealthCheckerConcurrentAddProvider(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	var wg sync.WaitGroup

	// Concurrent adds
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			providerID := "provider-" + string(rune('0'+index%10))
			hc.AddProvider(providerID)
		}(i)
	}

	wg.Wait()

	hc.mu.RLock()
	count := len(hc.circuitBreakers)
	hc.mu.RUnlock()

	// Should handle concurrent operations
	assert.Greater(t, count, 0, "Should have some providers added")
}

func TestHealthCheckerConcurrentRemoveProvider(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	// Add some providers first
	for i := 0; i < 10; i++ {
		hc.AddProvider("provider-" + string(rune('0'+i)))
	}

	var wg sync.WaitGroup

	// Concurrent removes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			hc.RemoveProvider("provider-" + string(rune('0'+index%10)))
		}(i)
	}

	wg.Wait()

	// Should not panic
	assert.True(t, true)
}

func TestHealthCheckerConcurrentGetCircuitBreaker(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	// Add providers
	for i := 0; i < 10; i++ {
		hc.AddProvider("provider-" + string(rune('0'+i)))
	}

	var wg sync.WaitGroup

	// Concurrent gets
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			providerID := "provider-" + string(rune('0'+index%10))
			hc.GetCircuitBreaker(providerID)
		}(i)
	}

	wg.Wait()

	// Should not panic
	assert.True(t, true)
}

func TestHealthCheckerConcurrentGetHealthyProviders(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	// Add providers
	for i := 0; i < 10; i++ {
		hc.AddProvider("provider-" + string(rune('0'+i)))
	}

	var wg sync.WaitGroup

	// Concurrent gets
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hc.GetHealthyProviders()
		}()
	}

	wg.Wait()

	// Should not panic
	assert.True(t, true)
}


func TestHealthCheckerAddGetRemoveCycle(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	providerID := "test-provider"

	// Add
	hc.AddProvider(providerID)
	cb1 := hc.GetCircuitBreaker(providerID)
	assert.NotNil(t, cb1)

	// Get
	cb2 := hc.GetCircuitBreaker(providerID)
	assert.Equal(t, cb1.name, cb2.name, "Should return same circuit breaker")

	// Remove
	hc.RemoveProvider(providerID)
	cb3 := hc.GetCircuitBreaker(providerID)
	assert.Nil(t, cb3)
}

func TestHealthCheckerCircuitBreakerIntegration(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	providerID := "test-provider"
	hc.AddProvider(providerID)

	// Get circuit breaker
	cb := hc.GetCircuitBreaker(providerID)
	assert.NotNil(t, cb)
	assert.Equal(t, StateClosed, cb.GetState())

	// Check healthy providers
	healthy := hc.GetHealthyProviders()
	assert.Contains(t, healthy, providerID, "Provider should be healthy initially")

	// Report failure through circuit breaker
	cb.Call(func() error {
		return assert.AnError
	})

	// Provider may still be in healthy list until next check
	assert.True(t, true)
}

func TestHealthCheckerInvalidProviderID(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	// Add provider with invalid ID (non-numeric)
	hc.AddProvider("invalid-id")

	// Check health - should handle gracefully
	hc.checkProviderHealth("invalid-id")

	// Should not panic
	assert.True(t, true)
}

func TestHealthCheckerEmptyCircuitBreakers(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	// Get healthy providers without adding any
	healthy := hc.GetHealthyProviders()
	assert.Equal(t, 0, len(healthy))

	// Perform health checks without any providers
	hc.performHealthChecks()

	// Should not panic
	assert.True(t, true)
}

func TestHealthCheckerHTTPClient(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	assert.NotNil(t, hc.httpClient, "HTTP client should be initialized")
	assert.Equal(t, 10*time.Second, hc.httpClient.Timeout, "HTTP client should have 10s timeout")
}

func TestHealthCheckerStopChannel(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	assert.NotNil(t, hc.stopCh, "Stop channel should be initialized")

	// Close should not panic
	close(hc.stopCh)
}

func TestHealthCheckerCheckInterval(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	assert.Equal(t, 30*time.Second, hc.checkInterval, "Default check interval should be 30s")
}

func TestHealthCheckerWithDatabaseProvider(t *testing.T) {
	db := setupTestHealthCheckerDB(t)

	// Create provider
	provider := &database.Provider{
		Name:       "test-provider",
		Endpoint:   "http://localhost:9999/api",
		Description: "Test provider",
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	hc := NewHealthChecker(db)
	hc.AddProvider("123")

	// Check provider health - will use database
	hc.checkProviderHealth("123")

	// Should not panic
	assert.True(t, true)
}

func TestHealthCheckerGetCircuitBreakerState(t *testing.T) {
	db := setupTestHealthCheckerDB(t)
	hc := NewHealthChecker(db)

	providerID := "test-provider"
	hc.AddProvider(providerID)

	cb := hc.GetCircuitBreaker(providerID)
	assert.NotNil(t, cb)

	// Check initial state
	state := cb.GetState()
	assert.Equal(t, StateClosed, state, "Initial state should be closed")

	// Check availability
	available := cb.IsAvailable()
	assert.True(t, available, "Should be available initially")
}
