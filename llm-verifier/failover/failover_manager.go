package failover

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"sort"
	"strconv"
	"sync"
	"time"

	"llm-verifier/database"
)

// FailoverManager coordinates circuit breakers, health checking, and routing
type FailoverManager struct {
	healthChecker  *HealthChecker
	latencyTracker *LatencyTracker
	providers      map[int64]*database.Provider
	models         map[string][]*database.Model // model_id -> providers
	costWeights    map[int64]float64            // provider_id -> cost weight (0.0-1.0)
	mu             sync.RWMutex
}

// NewFailoverManager creates a new failover manager
func NewFailoverManager(db *database.Database) *FailoverManager {
	healthChecker := NewHealthChecker(db)
	latencyTracker := NewLatencyTracker()

	fm := &FailoverManager{
		healthChecker:  healthChecker,
		latencyTracker: latencyTracker,
		providers:      make(map[int64]*database.Provider),
		models:         make(map[string][]*database.Model),
		costWeights:    make(map[int64]float64),
	}

	// Start health checking
	healthChecker.Start()

	// Load initial providers and models
	fm.loadProvidersAndModels(db)

	return fm
}

// Start begins the failover manager
func (fm *FailoverManager) Start() {
	log.Println("Failover manager started")
}

// Stop stops the failover manager
func (fm *FailoverManager) Stop() {
	if fm.healthChecker != nil {
		fm.healthChecker.Stop()
	}
	log.Println("Failover manager stopped")
}

// secureRandFloat64 generates a cryptographically secure random float64 in [0,1)
func secureRandFloat64() (float64, error) {
	// Generate 64-bit random number and convert to float64 [0,1)
	bytes := make([]byte, 8)
	_, err := rand.Read(bytes)
	if err != nil {
		return 0, err
	}

	// Convert bytes to uint64, then to float64 in [0,1)
	randInt := new(big.Int).SetBytes(bytes).Uint64()
	return float64(randInt) / float64(1<<64), nil
}

// secureRandIntn generates a cryptographically secure random int in [0,n)
func secureRandIntn(n int) int {
	if n <= 0 {
		return 0
	}

	// Generate random number and take modulo
	bytes := make([]byte, 4)
	_, err := rand.Read(bytes)
	if err != nil {
		// Fallback to time-based random (not ideal but functional)
		return int(time.Now().UnixNano()) % n
	}

	randInt := int(new(big.Int).SetBytes(bytes).Int64())
	if randInt < 0 {
		randInt = -randInt
	}
	return randInt % n
}

// SelectProvider selects the best provider for a model using failover logic
func (fm *FailoverManager) SelectProvider(modelID string) (*database.Provider, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	models, exists := fm.models[modelID]
	if !exists || len(models) == 0 {
		return nil, fmt.Errorf("no providers available for model %s", modelID)
	}

	// Get healthy providers for this model
	healthyProviders := fm.healthChecker.GetHealthyProviders()
	var availableProviders []*database.Provider

	for _, model := range models {
		provider, exists := fm.providers[model.ProviderID]
		if !exists {
			continue
		}

		// Check if provider is in healthy list
		for _, healthyID := range healthyProviders {
			if strconv.FormatInt(model.ProviderID, 10) == healthyID {
				availableProviders = append(availableProviders, provider)
				break
			}
		}
	}

	if len(availableProviders) == 0 {
		return nil, ErrNoHealthyProviders
	}

	// Apply weighted routing (70% cost-effective, 30% premium)
	return fm.selectWeightedProvider(availableProviders), nil
}

// selectWeightedProvider selects a provider based on cost weights and latency
func (fm *FailoverManager) selectWeightedProvider(providers []*database.Provider) *database.Provider {
	if len(providers) == 1 {
		return providers[0]
	}

	// Sort by latency (lowest first)
	sort.Slice(providers, func(i, j int) bool {
		statsI := fm.latencyTracker.GetLatencyStats(strconv.FormatInt(providers[i].ID, 10))
		statsJ := fm.latencyTracker.GetLatencyStats(strconv.FormatInt(providers[j].ID, 10))

		var latencyI, latencyJ time.Duration
		if statsI != nil {
			latencyI = statsI.AverageLatency
		}
		if statsJ != nil {
			latencyJ = statsJ.AverageLatency
		}
		return latencyI < latencyJ
	})

	// 70% chance for cost-effective (first 70% of sorted list)
	// 30% chance for premium (remaining providers)
	costEffectiveCount := int(float64(len(providers)) * 0.7)
	if costEffectiveCount < 1 {
		costEffectiveCount = 1
	}

	// Use secure random for provider selection
	r, err := secureRandFloat64()
	if err != nil {
		log.Printf("Failed to generate secure random, using timestamp as fallback: %v", err)
		r = float64(time.Now().UnixNano()%1000) / 1000.0
	}

	if r < 0.7 && costEffectiveCount > 0 {
		// Select from cost-effective providers (lower latency)
		return providers[secureRandIntn(costEffectiveCount)]
	} else {
		// Select from premium providers (higher latency, potentially better quality)
		premiumStart := costEffectiveCount
		if premiumStart >= len(providers) {
			premiumStart = 0
		}
		return providers[premiumStart+secureRandIntn(len(providers)-premiumStart)]
	}
}

// RecordLatency records latency for a provider
func (fm *FailoverManager) RecordLatency(providerID int64, latency time.Duration) {
	fm.latencyTracker.RecordLatency(strconv.FormatInt(providerID, 10), latency)
}

// ReportFailure reports a failure for a provider (triggers circuit breaker)
func (fm *FailoverManager) ReportFailure(providerID int64) {
	cb := fm.healthChecker.GetCircuitBreaker(strconv.FormatInt(providerID, 10))
	if cb != nil {
		cb.recordResult(false)
	}
}

// ReportSuccess reports a success for a provider
func (fm *FailoverManager) ReportSuccess(providerID int64) {
	cb := fm.healthChecker.GetCircuitBreaker(strconv.FormatInt(providerID, 10))
	if cb != nil {
		cb.recordResult(true)
	}
}

// GetProviderStatus returns the status of all providers
func (fm *FailoverManager) GetProviderStatus() map[string]interface{} {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	status := make(map[string]interface{})

	for providerID, provider := range fm.providers {
		cb := fm.healthChecker.GetCircuitBreaker(strconv.FormatInt(providerID, 10))
		circuitState := "unknown"
		if cb != nil {
			switch cb.GetState() {
			case StateClosed:
				circuitState = "closed"
			case StateOpen:
				circuitState = "open"
			case StateHalfOpen:
				circuitState = "half-open"
			}
		}

		stats := fm.latencyTracker.GetLatencyStats(strconv.FormatInt(providerID, 10))
		avgLatency := time.Duration(0)
		if stats != nil {
			avgLatency = stats.AverageLatency
		}

		healthy := false
		if cb != nil {
			healthy = cb.IsAvailable()
		}

		status[strconv.FormatInt(providerID, 10)] = map[string]interface{}{
			"name":            provider.Name,
			"healthy":         healthy,
			"average_latency": avgLatency.String(),
			"circuit_state":   circuitState,
		}
	}

	return status
}

// loadProvidersAndModels loads providers and models from database
func (fm *FailoverManager) loadProvidersAndModels(db *database.Database) {
	// Load providers
	providers, err := db.ListProviders(map[string]interface{}{})
	if err != nil {
		log.Printf("Failed to load providers: %v", err)
		return
	}

	for _, provider := range providers {
		fm.providers[provider.ID] = provider
		fm.healthChecker.AddProvider(strconv.FormatInt(provider.ID, 10))
		fm.costWeights[provider.ID] = fm.calculateCostWeight(provider)
	}

	// Load models and group by model_id
	models, err := db.ListModels(map[string]interface{}{})
	if err != nil {
		log.Printf("Failed to load models: %v", err)
		return
	}

	for _, model := range models {
		fm.models[model.ModelID] = append(fm.models[model.ModelID], model)
	}

	log.Printf("Loaded %d providers and %d model groups for failover management", len(fm.providers), len(fm.models))
}

// calculateCostWeight calculates a cost weight for a provider (0.0-1.0, lower is cheaper)
func (fm *FailoverManager) calculateCostWeight(provider *database.Provider) float64 {
	// Simple cost calculation based on provider name
	// In a real implementation, this would use pricing data
	switch provider.Name {
	case "openai":
		return 0.8 // Premium
	case "anthropic":
		return 0.7 // Premium
	case "google":
		return 0.6 // Balanced
	case "mistral":
		return 0.7 // Established provider
	case "meta":
		return 0.3 // Cost-effective
	case "groq":
		return 0.8 // High performance
	case "togetherai":
		return 0.6 // Balanced
	case "fireworks":
		return 0.6 // Balanced
	case "poe":
		return 0.5 // Standard
	case "navigator":
		return 0.4 // Research-focused
	default:
		return 0.5 // Default
	}
}
