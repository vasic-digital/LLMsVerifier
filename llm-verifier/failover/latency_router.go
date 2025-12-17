package failover

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"llm-verifier/database"
)

var (
	// ErrNoProvidersAvailable is returned when no providers are available for a model
	ErrNoProvidersAvailable = errors.New("no providers available for model")

	// ErrNoHealthyProviders is returned when no healthy providers are available
	ErrNoHealthyProviders = errors.New("no healthy providers available")
)

// LatencyTracker tracks latency metrics for providers
type LatencyTracker struct {
	providerLatencies map[string]*ProviderLatency
	mu                sync.RWMutex
}

// ProviderLatency holds latency information for a provider
type ProviderLatency struct {
	ProviderID     string
	SampleCount    int
	AverageLatency time.Duration
	MinLatency     time.Duration
	MaxLatency     time.Duration
	LastUpdated    time.Time
}

// NewLatencyTracker creates a new latency tracker
func NewLatencyTracker() *LatencyTracker {
	return &LatencyTracker{
		providerLatencies: make(map[string]*ProviderLatency),
	}
}

// RecordLatency records a latency measurement for a provider
func (lt *LatencyTracker) RecordLatency(providerID string, latency time.Duration) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	latencyInfo, exists := lt.providerLatencies[providerID]
	if !exists {
		latencyInfo = &ProviderLatency{
			ProviderID:     providerID,
			SampleCount:    0,
			AverageLatency: 0,
			MinLatency:     time.Hour, // Initialize to a large value
			MaxLatency:     0,
			LastUpdated:    time.Now(),
		}
		lt.providerLatencies[providerID] = latencyInfo
	}

	// Update metrics using exponential moving average
	alpha := 0.1 // Smoothing factor
	if latencyInfo.SampleCount == 0 {
		latencyInfo.AverageLatency = latency
	} else {
		latencyInfo.AverageLatency = time.Duration(float64(latencyInfo.AverageLatency)*(1-alpha) + float64(latency)*alpha)
	}

	latencyInfo.SampleCount++
	if latency < latencyInfo.MinLatency {
		latencyInfo.MinLatency = latency
	}
	if latency > latencyInfo.MaxLatency {
		latencyInfo.MaxLatency = latency
	}
	latencyInfo.LastUpdated = time.Now()
}

// GetFastestProvider returns the provider with the lowest average latency
func (lt *LatencyTracker) GetFastestProvider(providerIDs []string) string {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	var candidates []ProviderLatency
	for _, id := range providerIDs {
		if latency, exists := lt.providerLatencies[id]; exists {
			candidates = append(candidates, *latency)
		}
	}

	if len(candidates) == 0 {
		return ""
	}

	// Sort by average latency (ascending)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].AverageLatency < candidates[j].AverageLatency
	})

	return candidates[0].ProviderID
}

// GetLatencyStats returns latency statistics for a provider
func (lt *LatencyTracker) GetLatencyStats(providerID string) *ProviderLatency {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	if latency, exists := lt.providerLatencies[providerID]; exists {
		// Return a copy
		copy := *latency
		return &copy
	}
	return nil
}

// GetAllLatencyStats returns latency statistics for all providers
func (lt *LatencyTracker) GetAllLatencyStats() map[string]*ProviderLatency {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	result := make(map[string]*ProviderLatency)
	for k, v := range lt.providerLatencies {
		copy := *v
		result[k] = &copy
	}
	return result
}

// LatencyBasedRouter routes requests based on provider latency
type LatencyBasedRouter struct {
	latencyTracker *LatencyTracker
	healthChecker  *HealthChecker
	db             *database.Database
}

// NewLatencyBasedRouter creates a new latency-based router
func NewLatencyBasedRouter(latencyTracker *LatencyTracker, healthChecker *HealthChecker, db *database.Database) *LatencyBasedRouter {
	return &LatencyBasedRouter{
		latencyTracker: latencyTracker,
		healthChecker:  healthChecker,
		db:             db,
	}
}

// RouteRequest routes a request to the fastest available provider
func (lbr *LatencyBasedRouter) RouteRequest(modelID int64) (string, error) {
	// Get all providers that support this model
	providers, err := lbr.getProvidersForModel(modelID)
	if err != nil {
		return "", err
	}

	if len(providers) == 0 {
		return "", ErrNoProvidersAvailable
	}

	// Filter to only healthy providers
	var healthyProviders []string
	for _, providerID := range providers {
		if lbr.healthChecker.GetCircuitBreaker(providerID).IsAvailable() {
			healthyProviders = append(healthyProviders, providerID)
		}
	}

	if len(healthyProviders) == 0 {
		return "", ErrNoHealthyProviders
	}

	// Get the fastest provider
	fastestProvider := lbr.latencyTracker.GetFastestProvider(healthyProviders)
	if fastestProvider == "" {
		// Fallback to first healthy provider if no latency data
		fastestProvider = healthyProviders[0]
	}

	return fastestProvider, nil
}

// getProvidersForModel returns all provider IDs that support a given model
func (lbr *LatencyBasedRouter) getProvidersForModel(modelID int64) ([]string, error) {
	model, err := lbr.db.GetModel(modelID)
	if err != nil {
		return nil, err
	}

	// For now, return the model's provider
	// In a real implementation, this might return multiple providers
	providerID := fmt.Sprintf("%d", model.ProviderID)
	return []string{providerID}, nil
}

// WeightedRouter implements weighted routing based on cost-effectiveness and premium features
type WeightedRouter struct {
	latencyTracker *LatencyTracker
	healthChecker  *HealthChecker
	costWeight     float64 // Weight for cost-effective routing (0-1)
	premiumWeight  float64 // Weight for premium routing (0-1)
}

// NewWeightedRouter creates a new weighted router
func NewWeightedRouter(latencyTracker *LatencyTracker, healthChecker *HealthChecker) *WeightedRouter {
	return &WeightedRouter{
		latencyTracker: latencyTracker,
		healthChecker:  healthChecker,
		costWeight:     0.7, // 70% cost-effective
		premiumWeight:  0.3, // 30% premium
	}
}

// RouteRequest routes a request using weighted distribution
func (wr *WeightedRouter) RouteRequest(modelID int64, db *database.Database) (string, error) {
	// Get all providers for the model
	model, err := db.GetModel(modelID)
	if err != nil {
		return "", err
	}

	providerID := fmt.Sprintf("%d", model.ProviderID)

	// Check if provider is healthy
	if !wr.healthChecker.GetCircuitBreaker(providerID).IsAvailable() {
		return "", ErrNoHealthyProviders
	}

	// For now, just return the provider
	// In a full implementation, this would balance between cost-effective and premium providers
	return providerID, nil
}

// CalculateProviderScore calculates a score for provider selection
func (wr *WeightedRouter) CalculateProviderScore(providerID string, db *database.Database) float64 {
	// Get latency score (lower latency = higher score)
	latencyStats := wr.latencyTracker.GetLatencyStats(providerID)
	latencyScore := 1.0
	if latencyStats != nil && latencyStats.AverageLatency > 0 {
		// Normalize latency (assuming 1 second = baseline)
		normalizedLatency := float64(latencyStats.AverageLatency) / float64(time.Second)
		latencyScore = 1.0 / math.Max(normalizedLatency, 0.1) // Avoid division by zero
	}

	// Get health score
	healthScore := 1.0
	if !wr.healthChecker.GetCircuitBreaker(providerID).IsAvailable() {
		healthScore = 0.0
	}

	// For now, just return a combination of latency and health
	// In a full implementation, this would include cost data
	return latencyScore*0.6 + healthScore*0.4
}
