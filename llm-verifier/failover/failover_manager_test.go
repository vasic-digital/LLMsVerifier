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

func setupTestFailoverDB(t *testing.T) *database.Database {
	dbFile := "/tmp/test_failover_" + time.Now().Format("20060102150405") + ".db"

	db, err := database.New(dbFile)
	require.NoError(t, err, "Failed to create test database")

	t.Cleanup(func() {
		os.Remove(dbFile)
	})

	return db
}

func TestNewFailoverManager(t *testing.T) {
	db := setupTestFailoverDB(t)

	fm := NewFailoverManager(db)

	assert.NotNil(t, fm, "FailoverManager should not be nil")
	assert.NotNil(t, fm.healthChecker, "HealthChecker should be initialized")
	assert.NotNil(t, fm.latencyTracker, "LatencyTracker should be initialized")
	assert.NotNil(t, fm.providers, "Providers map should be initialized")
	assert.NotNil(t, fm.models, "Models map should be initialized")
	assert.NotNil(t, fm.costWeights, "Cost weights map should be initialized")

	// Stop the manager
	fm.Stop()
}

func TestFailoverManagerStart(t *testing.T) {
	db := setupTestFailoverDB(t)
	fm := NewFailoverManager(db)

	// Start should not panic
	fm.Start()

	fm.Stop()
}

func TestFailoverManagerStop(t *testing.T) {
	db := setupTestFailoverDB(t)
	fm := NewFailoverManager(db)

	// Stop should not panic
	fm.Stop()
}

func TestSecureRandFloat64(t *testing.T) {
	randVal, err := secureRandFloat64()

	assert.NoError(t, err, "Should generate random value without error")
	assert.GreaterOrEqual(t, randVal, 0.0, "Random value should be >= 0")
	assert.Less(t, randVal, 1.0, "Random value should be < 1")
}

func TestSecureRandFloat64Multiple(t *testing.T) {
	values := make([]float64, 100)
	uniqueValues := make(map[float64]bool)

	for i := 0; i < 100; i++ {
		val, err := secureRandFloat64()
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, val, 0.0)
		assert.Less(t, val, 1.0)
		values[i] = val
		uniqueValues[val] = true
	}

	// Check we got some variety
	assert.Greater(t, len(uniqueValues), 50, "Should have variety in random values")
}

func TestSecureRandIntn(t *testing.T) {
	testCases := []int{1, 5, 10, 100, 1000}

	for _, n := range testCases {
		val := secureRandIntn(n)
		assert.GreaterOrEqual(t, val, 0, "Random value should be >= 0 for n=%d", n)
		assert.Less(t, val, n, "Random value should be < %d", n)
	}
}

func TestSecureRandIntnZero(t *testing.T) {
	val := secureRandIntn(0)
	assert.Equal(t, 0, val, "Random value with n=0 should be 0")
}

func TestSecureRandIntnNegative(t *testing.T) {
	val := secureRandIntn(-5)
	assert.Equal(t, 0, val, "Random value with negative n should be 0")
}

func TestSecureRandIntnDistribution(t *testing.T) {
	counts := make([]int, 10)
	n := 10

	// Generate 1000 random numbers
	for i := 0; i < 1000; i++ {
		val := secureRandIntn(n)
		counts[val]++
	}

	// Each value should appear roughly 100 times (with some variance)
	for i, count := range counts {
		assert.Greater(t, count, 50, "Value %d should appear more than 50 times", i)
		assert.Less(t, count, 200, "Value %d should appear less than 200 times", i)
	}
}

func TestFailoverManagerSelectProviderNoProviders(t *testing.T) {
	db := setupTestFailoverDB(t)
	fm := NewFailoverManager(db)
	defer fm.Stop()

	_, err := fm.SelectProvider("non-existent-model")
	assert.Error(t, err, "Should return error when model not found")
}

func TestFailoverManagerSelectProviderWithModel(t *testing.T) {
	db := setupTestFailoverDB(t)

	// Create test provider
	provider := &database.Provider{
		Name:        "test-provider",
		Endpoint:    "https://test.com/api",
		Description: "Test provider",
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	// Create test model
	model := &database.Model{
		ModelID:    "test-model",
		Name:       "Test Model",
		ProviderID: provider.ID,
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	// Create failover manager to load providers
	fm := NewFailoverManager(db)
	defer fm.Stop()

	// Select provider
	selected, err := fm.SelectProvider("test-model")
	assert.NoError(t, err, "Should select provider successfully")
	assert.NotNil(t, selected, "Provider should not be nil")
	assert.Equal(t, provider.ID, selected.ID, "Should return correct provider")
}

func TestFailoverManagerRecordLatency(t *testing.T) {
	db := setupTestFailoverDB(t)
	fm := NewFailoverManager(db)
	defer fm.Stop()

	providerID := int64(123)
	latency := 100 * time.Millisecond

	// Record latency - should not panic
	fm.RecordLatency(providerID, latency)
}

func TestFailoverManagerReportFailure(t *testing.T) {
	db := setupTestFailoverDB(t)
	fm := NewFailoverManager(db)
	defer fm.Stop()

	providerID := int64(123)

	// Report failure - should not panic
	fm.ReportFailure(providerID)
}

func TestFailoverManagerReportSuccess(t *testing.T) {
	db := setupTestFailoverDB(t)
	fm := NewFailoverManager(db)
	defer fm.Stop()

	providerID := int64(123)

	// Report success - should not panic
	fm.ReportSuccess(providerID)
}

func TestFailoverManagerGetProviderStatus(t *testing.T) {
	db := setupTestFailoverDB(t)
	fm := NewFailoverManager(db)
	defer fm.Stop()

	status := fm.GetProviderStatus()

	assert.NotNil(t, status, "Status should not be nil")
	assert.IsType(t, map[string]interface{}{}, status, "Status should be a map")
}

func TestFailoverManagerGetProviderStatusWithProviders(t *testing.T) {
	db := setupTestFailoverDB(t)

	// Create test providers
	provider1 := &database.Provider{
		Name:        "openai",
		Endpoint:    "https://openai.com/api",
		Description: "OpenAI provider",
	}
	err := db.CreateProvider(provider1)
	require.NoError(t, err)

	provider2 := &database.Provider{
		Name:        "mistral",
		Endpoint:    "https://mistral.com/api",
		Description: "Mistral provider",
	}
	err = db.CreateProvider(provider2)
	require.NoError(t, err)

	// Create failover manager
	fm := NewFailoverManager(db)
	defer fm.Stop()

	// Get status
	status := fm.GetProviderStatus()

	assert.NotNil(t, status)
	assert.Equal(t, 2, len(status), "Should have status for 2 providers")
}

func TestFailoverManagerCalculateCostWeight(t *testing.T) {
	providers := []struct {
		name   string
		weight float64
	}{
		{"openai", 0.8},
		{"anthropic", 0.7},
		{"google", 0.6},
		{"mistral", 0.7},
		{"meta", 0.3},
		{"default", 0.5},
	}

	for _, tc := range providers {
		db := setupTestFailoverDB(t)
		fm := NewFailoverManager(db)

		provider := &database.Provider{
			Name: tc.name,
		}
		weight := fm.calculateCostWeight(provider)

		assert.Equal(t, tc.weight, weight, "Cost weight for %s should match", tc.name)

		fm.Stop()
	}
}

func TestFailoverManagerSelectWeightedProvider(t *testing.T) {
	db := setupTestFailoverDB(t)

	// Create multiple providers
	providers := []*database.Provider{
		{ID: 1, Name: "provider-1"},
		{ID: 2, Name: "provider-2"},
		{ID: 3, Name: "provider-3"},
	}

	fm := NewFailoverManager(db)
	defer fm.Stop()

	// Single provider should return itself
	selected := fm.selectWeightedProvider(providers[:1])
	assert.Equal(t, providers[0].ID, selected.ID, "Single provider should be returned")

	// Multiple providers should return one
	selected = fm.selectWeightedProvider(providers)
	assert.NotNil(t, selected, "Should select a provider")
	assert.Contains(t, []int64{1, 2, 3}, selected.ID, "Should select from available providers")
}

func TestFailoverManagerLoadProvidersAndModels(t *testing.T) {
	db := setupTestFailoverDB(t)

	// Create provider
	provider := &database.Provider{
		Name:        "test-provider",
		Endpoint:    "https://test.com/api",
		Description: "Test provider",
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	// Create model
	model := &database.Model{
		ModelID:    "test-model",
		Name:       "Test Model",
		ProviderID: provider.ID,
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	// Create failover manager - should load providers and models
	fm := NewFailoverManager(db)
	defer fm.Stop()

	// Verify providers loaded
	fm.mu.RLock()
	loadedProvider, exists := fm.providers[provider.ID]
	fm.mu.RUnlock()

	assert.True(t, exists, "Provider should be loaded")
	assert.Equal(t, provider.ID, loadedProvider.ID, "Provider ID should match")

	// Verify models loaded
	fm.mu.RLock()
	loadedModels, exists := fm.models[model.ModelID]
	fm.mu.RUnlock()

	assert.True(t, exists, "Model should be loaded")
	assert.Len(t, loadedModels, 1, "Should have 1 model in group")
}

func TestFailoverManagerLoadMultipleModels(t *testing.T) {
	db := setupTestFailoverDB(t)

	// Create provider
	provider := &database.Provider{
		Name:        "test-provider",
		Endpoint:    "https://test.com/api",
		Description: "Test provider",
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	// Create multiple models for same model_id
	model1 := &database.Model{
		ModelID:    "gpt-4",
		Name:       "GPT-4",
		ProviderID: provider.ID,
	}
	err = db.CreateModel(model1)
	require.NoError(t, err)

	model2 := &database.Model{
		ModelID:    "gpt-4",
		Name:       "GPT-4 Turbo",
		ProviderID: provider.ID,
	}
	err = db.CreateModel(model2)
	require.NoError(t, err)

	// Create failover manager
	fm := NewFailoverManager(db)
	defer fm.Stop()

	// Verify models grouped
	fm.mu.RLock()
	loadedModels, exists := fm.models["gpt-4"]
	fm.mu.RUnlock()

	assert.True(t, exists, "Models should be loaded")
	assert.Len(t, loadedModels, 2, "Should have 2 models in group")
}

func TestFailoverManagerConcurrentSelectProvider(t *testing.T) {
	db := setupTestFailoverDB(t)

	// Create test provider
	provider := &database.Provider{
		Name:        "test-provider",
		Endpoint:    "https://test.com/api",
		Description: "Test provider",
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	// Create test model
	model := &database.Model{
		ModelID:    "test-model",
		Name:       "Test Model",
		ProviderID: provider.ID,
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	// Create failover manager
	fm := NewFailoverManager(db)
	defer fm.Stop()

	// Concurrent selects
	var wg sync.WaitGroup
	selectedProviders := make([]*database.Provider, 50)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			selected, _ := fm.SelectProvider("test-model")
			if selected != nil {
				selectedProviders[index] = selected
			}
		}(i)
	}

	wg.Wait()

	// All should get a provider
	nonNull := 0
	for _, p := range selectedProviders {
		if p != nil {
			nonNull++
		}
	}

	assert.Greater(t, nonNull, 40, "Most selections should succeed")
}

func TestFailoverManagerConcurrentLatencyRecording(t *testing.T) {
	db := setupTestFailoverDB(t)
	fm := NewFailoverManager(db)
	defer fm.Stop()

	var wg sync.WaitGroup
	providerID := int64(123)

	// Concurrent latency recording
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fm.RecordLatency(providerID, time.Duration(i)*time.Millisecond)
		}()
	}

	wg.Wait()
}

func TestFailoverManagerConcurrentFailureReporting(t *testing.T) {
	db := setupTestFailoverDB(t)
	fm := NewFailoverManager(db)
	defer fm.Stop()

	var wg sync.WaitGroup
	providerID := int64(123)

	// Concurrent failure reporting
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fm.ReportFailure(providerID)
		}()
	}

	wg.Wait()
}

func TestFailoverManagerConcurrentSuccessReporting(t *testing.T) {
	db := setupTestFailoverDB(t)
	fm := NewFailoverManager(db)
	defer fm.Stop()

	var wg sync.WaitGroup
	providerID := int64(123)

	// Concurrent success reporting
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fm.ReportSuccess(providerID)
		}()
	}

	wg.Wait()
}

func TestFailoverManagerStatusConcurrent(t *testing.T) {
	db := setupTestFailoverDB(t)
	fm := NewFailoverManager(db)
	defer fm.Stop()

	var wg sync.WaitGroup
	statuses := make([]map[string]interface{}, 10)

	// Concurrent status retrieval
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			statuses[index] = fm.GetProviderStatus()
		}(i)
	}

	wg.Wait()

	// All should return status
	for _, status := range statuses {
		assert.NotNil(t, status, "Status should not be nil")
	}
}

func TestFailoverManagerNilHealthChecker(t *testing.T) {
	fm := &FailoverManager{
		healthChecker:  nil,
		latencyTracker: nil,
		providers:      make(map[int64]*database.Provider),
		models:         make(map[string][]*database.Model),
		costWeights:    make(map[int64]float64),
	}

	// Should not panic
	fm.Stop()
}

func TestFailoverManagerLoadEmptyDatabase(t *testing.T) {
	db := setupTestFailoverDB(t)
	fm := NewFailoverManager(db)
	defer fm.Stop()

	fm.mu.RLock()
	providersCount := len(fm.providers)
	modelsCount := len(fm.models)
	fm.mu.RUnlock()

	assert.Equal(t, 0, providersCount, "Should have 0 providers from empty DB")
	assert.Equal(t, 0, modelsCount, "Should have 0 models from empty DB")
}

func TestFailoverManagerGetHealthyProvidersOnly(t *testing.T) {
	db := setupTestFailoverDB(t)

	// Create test provider
	provider := &database.Provider{
		Name:        "test-provider",
		Endpoint:    "https://test.com/api",
		Description: "Test provider",
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	// Create test model
	model := &database.Model{
		ModelID:    "test-model",
		Name:       "Test Model",
		ProviderID: provider.ID,
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	fm := NewFailoverManager(db)
	defer fm.Stop()

	// Select should work with healthy provider
	selected, err := fm.SelectProvider("test-model")
	assert.NoError(t, err)
	assert.NotNil(t, selected)
}
