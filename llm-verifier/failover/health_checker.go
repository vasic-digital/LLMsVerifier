package failover

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"llm-verifier/database"
)

// HealthChecker monitors provider health and updates circuit breakers
type HealthChecker struct {
	db              *database.Database
	circuitBreakers map[string]*CircuitBreaker
	httpClient      *http.Client
	checkInterval   time.Duration
	stopCh          chan struct{}
	wg              sync.WaitGroup
	mu              sync.RWMutex
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(db *database.Database) *HealthChecker {
	return &HealthChecker{
		db:              db,
		circuitBreakers: make(map[string]*CircuitBreaker),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		checkInterval: 30 * time.Second, // Check every 30 seconds
		stopCh:        make(chan struct{}),
	}
}

// AddProvider adds a provider to health monitoring
func (hc *HealthChecker) AddProvider(providerID string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.circuitBreakers[providerID] = NewCircuitBreaker(fmt.Sprintf("provider-%s", providerID))
}

// RemoveProvider removes a provider from health monitoring
func (hc *HealthChecker) RemoveProvider(providerID string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	delete(hc.circuitBreakers, providerID)
}

// Start begins health checking
func (hc *HealthChecker) Start() {
	hc.wg.Add(1)
	go hc.healthCheckLoop()
	log.Println("Health checker started")
}

// Stop stops health checking
func (hc *HealthChecker) Stop() {
	close(hc.stopCh)
	hc.wg.Wait()
	log.Println("Health checker stopped")
}

// healthCheckLoop runs the health checking loop
func (hc *HealthChecker) healthCheckLoop() {
	defer hc.wg.Done()

	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-hc.stopCh:
			return
		case <-ticker.C:
			hc.performHealthChecks()
		}
	}
}

// performHealthChecks checks all providers
func (hc *HealthChecker) performHealthChecks() {
	hc.mu.RLock()
	providers := make([]string, 0, len(hc.circuitBreakers))
	for providerID := range hc.circuitBreakers {
		providers = append(providers, providerID)
	}
	hc.mu.RUnlock()

	for _, providerID := range providers {
		go hc.checkProviderHealth(providerID)
	}
}

// checkProviderHealth checks the health of a specific provider
func (hc *HealthChecker) checkProviderHealth(providerID string) {
	hc.mu.RLock()
	cb := hc.circuitBreakers[providerID]
	hc.mu.RUnlock()

	if cb == nil {
		return
	}

	// Convert providerID to int64
	providerIDInt, err := strconv.ParseInt(providerID, 10, 64)
	if err != nil {
		log.Printf("Invalid provider ID %s: %v", providerID, err)
		return
	}

	// Get provider info from database
	provider, err := hc.db.GetProvider(providerIDInt)
	if err != nil {
		log.Printf("Failed to get provider %s: %v", providerID, err)
		return
	}

	// Perform health check
	healthy := hc.checkProviderEndpoint(provider.Endpoint)

	// Update circuit breaker
	err = cb.Call(func() error {
		if !healthy {
			return fmt.Errorf("provider %s is unhealthy", providerID)
		}
		return nil
	})

	if err != nil {
		log.Printf("Provider %s health check failed: %v", providerID, err)
	} else {
		log.Printf("Provider %s health check passed", providerID)
	}

	// Update provider status in database
	hc.updateProviderHealth(providerID, healthy)
}

// checkProviderEndpoint performs a basic health check on the provider endpoint
func (hc *HealthChecker) checkProviderEndpoint(endpoint string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"/health", nil)
	if err != nil {
		return false
	}

	resp, err := hc.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// updateProviderHealth updates the provider's health status in the database
func (hc *HealthChecker) updateProviderHealth(providerID string, healthy bool) {
	// This would update the provider's last_checked timestamp and reliability_score
	// For now, we'll just log it
	status := "healthy"
	if !healthy {
		status = "unhealthy"
	}
	log.Printf("Provider %s status updated to: %s", providerID, status)
}

// GetCircuitBreaker returns the circuit breaker for a provider
func (hc *HealthChecker) GetCircuitBreaker(providerID string) *CircuitBreaker {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.circuitBreakers[providerID]
}

// GetHealthyProviders returns a list of healthy provider IDs
func (hc *HealthChecker) GetHealthyProviders() []string {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	var healthy []string
	for providerID, cb := range hc.circuitBreakers {
		if cb.IsAvailable() {
			healthy = append(healthy, providerID)
		}
	}
	return healthy
}
