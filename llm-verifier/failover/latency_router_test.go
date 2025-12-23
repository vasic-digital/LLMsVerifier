package failover

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"llm-verifier/database"
)

func TestNewLatencyTracker(t *testing.T) {
	lt := NewLatencyTracker()

	assert.NotNil(t, lt, "LatencyTracker should not be nil")
	assert.NotNil(t, lt.providerLatencies, "Provider latencies map should be initialized")
	assert.Equal(t, 0, len(lt.providerLatencies), "Should have 0 providers initially")
}

func TestLatencyTrackerRecordLatency(t *testing.T) {
	lt := NewLatencyTracker()

	providerID := "provider-1"
	latency := 100 * time.Millisecond

	lt.RecordLatency(providerID, latency)

	stats := lt.GetLatencyStats(providerID)
	assert.NotNil(t, stats, "Should return stats for provider")
	assert.Equal(t, providerID, stats.ProviderID, "Provider ID should match")
	assert.Equal(t, 1, stats.SampleCount, "Should have 1 sample")
	assert.Equal(t, latency, stats.AverageLatency, "Average latency should match")
	assert.Equal(t, latency, stats.MinLatency, "Min latency should match")
	assert.Equal(t, latency, stats.MaxLatency, "Max latency should match")
}

func TestLatencyTrackerRecordLatencyMultiple(t *testing.T) {
	lt := NewLatencyTracker()

	providerID := "provider-1"
	latencies := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		300 * time.Millisecond,
	}

	for _, latency := range latencies {
		lt.RecordLatency(providerID, latency)
	}

	stats := lt.GetLatencyStats(providerID)
	assert.NotNil(t, stats)
	assert.Equal(t, 3, stats.SampleCount, "Should have 3 samples")
	assert.Equal(t, 100*time.Millisecond, stats.MinLatency, "Min latency should be 100ms")
	assert.Equal(t, 300*time.Millisecond, stats.MaxLatency, "Max latency should be 300ms")
}

func TestLatencyTrackerRecordLatencyMultipleProviders(t *testing.T) {
	lt := NewLatencyTracker()

	providerIDs := []string{"provider-1", "provider-2", "provider-3"}

	for _, id := range providerIDs {
		lt.RecordLatency(id, 100*time.Millisecond)
	}

	for _, id := range providerIDs {
		stats := lt.GetLatencyStats(id)
		assert.NotNil(t, stats, "Should return stats for provider %s", id)
		assert.Equal(t, 1, stats.SampleCount, "Should have 1 sample")
	}
}

func TestLatencyTrackerGetLatencyStatsNonExistent(t *testing.T) {
	lt := NewLatencyTracker()

	stats := lt.GetLatencyStats("non-existent")
	assert.Nil(t, stats, "Should return nil for non-existent provider")
}

func TestLatencyTrackerGetAllLatencyStats(t *testing.T) {
	lt := NewLatencyTracker()

	// Record latencies for multiple providers
	lt.RecordLatency("provider-1", 100*time.Millisecond)
	lt.RecordLatency("provider-2", 200*time.Millisecond)
	lt.RecordLatency("provider-3", 150*time.Millisecond)

	allStats := lt.GetAllLatencyStats()

	assert.NotNil(t, allStats)
	assert.Equal(t, 3, len(allStats), "Should have stats for 3 providers")

	for providerID, stats := range allStats {
		assert.NotNil(t, stats, "Stats for %s should not be nil", providerID)
		assert.Equal(t, providerID, stats.ProviderID, "Provider ID should match")
	}
}

func TestLatencyTrackerGetAllLatencyStatsEmpty(t *testing.T) {
	lt := NewLatencyTracker()

	allStats := lt.GetAllLatencyStats()

	assert.NotNil(t, allStats)
	assert.Equal(t, 0, len(allStats), "Should have 0 stats when empty")
}

func TestLatencyTrackerGetFastestProvider(t *testing.T) {
	lt := NewLatencyTracker()

	// Record different latencies
	lt.RecordLatency("provider-1", 200*time.Millisecond)
	lt.RecordLatency("provider-2", 100*time.Millisecond)
	lt.RecordLatency("provider-3", 150*time.Millisecond)

	fastest := lt.GetFastestProvider([]string{"provider-1", "provider-2", "provider-3"})

	assert.Equal(t, "provider-2", fastest, "Should return fastest provider")
}

func TestLatencyTrackerGetFastestProviderSingle(t *testing.T) {
	lt := NewLatencyTracker()

	lt.RecordLatency("provider-1", 100*time.Millisecond)

	fastest := lt.GetFastestProvider([]string{"provider-1"})

	assert.Equal(t, "provider-1", fastest, "Should return only provider")
}

func TestLatencyTrackerGetFastestProviderNoLatencyData(t *testing.T) {
	lt := NewLatencyTracker()

	fastest := lt.GetFastestProvider([]string{"provider-1", "provider-2"})

	assert.Empty(t, fastest, "Should return empty string when no latency data")
}

func TestLatencyTrackerGetFastestProviderEmptyList(t *testing.T) {
	lt := NewLatencyTracker()

	fastest := lt.GetFastestProvider([]string{})

	assert.Empty(t, fastest, "Should return empty string for empty list")
}

func TestLatencyTrackerGetFastestProviderPartialData(t *testing.T) {
	lt := NewLatencyTracker()

	// Record latency only for provider-2
	lt.RecordLatency("provider-2", 100*time.Millisecond)

	fastest := lt.GetFastestProvider([]string{"provider-1", "provider-2", "provider-3"})

	assert.Equal(t, "provider-2", fastest, "Should return fastest from available data")
}

func TestLatencyTrackerExponentialMovingAverage(t *testing.T) {
	lt := NewLatencyTracker()

	providerID := "provider-1"

	// First latency - sets baseline
	lt.RecordLatency(providerID, 100*time.Millisecond)
	stats := lt.GetLatencyStats(providerID)
	assert.Equal(t, 100*time.Millisecond, stats.AverageLatency, "First latency should be average")

	// Second latency - EMA should update
	lt.RecordLatency(providerID, 200*time.Millisecond)
	stats = lt.GetLatencyStats(providerID)
	// EMA: 100 * 0.9 + 200 * 0.1 = 90 + 20 = 110
	expectedEMA := 110 * time.Millisecond
	// Allow for some rounding differences
	assert.InDelta(t, expectedEMA, stats.AverageLatency, float64(5*time.Millisecond), "EMA should be calculated correctly")
}

func TestLatencyTrackerMinMaxLatency(t *testing.T) {
	lt := NewLatencyTracker()

	providerID := "provider-1"

	// Record various latencies
	lt.RecordLatency(providerID, 150*time.Millisecond)
	lt.RecordLatency(providerID, 50*time.Millisecond)
	lt.RecordLatency(providerID, 200*time.Millisecond)
	lt.RecordLatency(providerID, 100*time.Millisecond)

	stats := lt.GetLatencyStats(providerID)

	assert.Equal(t, 50*time.Millisecond, stats.MinLatency, "Min latency should be 50ms")
	assert.Equal(t, 200*time.Millisecond, stats.MaxLatency, "Max latency should be 200ms")
}

func TestLatencyTrackerLastUpdated(t *testing.T) {
	lt := NewLatencyTracker()

	providerID := "provider-1"

	before := time.Now()
	lt.RecordLatency(providerID, 100*time.Millisecond)
	after := time.Now()

	stats := lt.GetLatencyStats(providerID)

	assert.True(t, stats.LastUpdated.After(before) || stats.LastUpdated.Equal(before), "Last updated should be after recording")
	assert.True(t, stats.LastUpdated.Before(after) || stats.LastUpdated.Equal(after), "Last updated should be before or at recording time")
}

func TestLatencyTrackerCopyReturned(t *testing.T) {
	lt := NewLatencyTracker()

	providerID := "provider-1"
	lt.RecordLatency(providerID, 100*time.Millisecond)

	// Get stats
	stats1 := lt.GetLatencyStats(providerID)

	// Get stats again
	stats2 := lt.GetLatencyStats(providerID)

	// Modify first copy
	stats1.AverageLatency = 999 * time.Millisecond

	// Second copy should not be affected
	assert.Equal(t, 100*time.Millisecond, stats2.AverageLatency, "Returned stats should be a copy")
}

func TestLatencyTrackerConcurrentRecordLatency(t *testing.T) {
	lt := NewLatencyTracker()

	var wg sync.WaitGroup
	providerID := "provider-1"

	// Concurrent recordings
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			latency := time.Duration(index) * time.Millisecond
			lt.RecordLatency(providerID, latency)
		}(i)
	}

	wg.Wait()

	stats := lt.GetLatencyStats(providerID)
	assert.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats.SampleCount, 90, "Should have recorded most samples")
}

func TestLatencyTrackerConcurrentGetStats(t *testing.T) {
	lt := NewLatencyTracker()

	providerID := "provider-1"
	lt.RecordLatency(providerID, 100*time.Millisecond)

	var wg sync.WaitGroup

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lt.GetLatencyStats(providerID)
		}()
	}

	wg.Wait()

	// Should not panic
	assert.True(t, true)
}

func TestNewLatencyBasedRouter(t *testing.T) {
	lt := NewLatencyTracker()
	hc := NewHealthChecker(nil)
	db := setupTestHealthCheckerDB(t)

	lbr := NewLatencyBasedRouter(lt, hc, db)

	assert.NotNil(t, lbr, "Router should not be nil")
	assert.NotNil(t, lbr.latencyTracker, "Latency tracker should be set")
	assert.NotNil(t, lbr.healthChecker, "Health checker should be set")
	assert.NotNil(t, lbr.db, "Database should be set")
}

func TestNewWeightedRouter(t *testing.T) {
	lt := NewLatencyTracker()
	hc := NewHealthChecker(nil)

	wr := NewWeightedRouter(lt, hc)

	assert.NotNil(t, wr, "Router should not be nil")
	assert.NotNil(t, wr.latencyTracker, "Latency tracker should be set")
	assert.NotNil(t, wr.healthChecker, "Health checker should be set")
	assert.Equal(t, 0.7, wr.costWeight, "Cost weight should be 0.7")
	assert.Equal(t, 0.3, wr.premiumWeight, "Premium weight should be 0.3")
}

func TestErrNoProvidersAvailable(t *testing.T) {
	assert.NotNil(t, ErrNoProvidersAvailable, "Error should not be nil")
	assert.Equal(t, "no providers available for model", ErrNoProvidersAvailable.Error())
}

func TestErrNoHealthyProviders(t *testing.T) {
	assert.NotNil(t, ErrNoHealthyProviders, "Error should not be nil")
	assert.Equal(t, "no healthy providers available", ErrNoHealthyProviders.Error())
}

func TestProviderLatencyStruct(t *testing.T) {
	now := time.Now()
	pl := ProviderLatency{
		ProviderID:     "provider-1",
		SampleCount:    10,
		AverageLatency: 100 * time.Millisecond,
		MinLatency:     50 * time.Millisecond,
		MaxLatency:     200 * time.Millisecond,
		LastUpdated:    now,
	}

	assert.Equal(t, "provider-1", pl.ProviderID)
	assert.Equal(t, 10, pl.SampleCount)
	assert.Equal(t, 100*time.Millisecond, pl.AverageLatency)
	assert.Equal(t, 50*time.Millisecond, pl.MinLatency)
	assert.Equal(t, 200*time.Millisecond, pl.MaxLatency)
	assert.Equal(t, now, pl.LastUpdated)
}

func TestLatencyTrackerZeroLatency(t *testing.T) {
	lt := NewLatencyTracker()

	providerID := "provider-1"

	lt.RecordLatency(providerID, 0)

	stats := lt.GetLatencyStats(providerID)
	assert.NotNil(t, stats)
	assert.Equal(t, time.Duration(0), stats.AverageLatency, "Should handle zero latency")
}

func TestLatencyTrackerHighLatency(t *testing.T) {
	lt := NewLatencyTracker()

	providerID := "provider-1"

	highLatency := 10 * time.Minute
	lt.RecordLatency(providerID, highLatency)

	stats := lt.GetLatencyStats(providerID)
	assert.NotNil(t, stats)
	assert.Equal(t, highLatency, stats.AverageLatency, "Should handle high latency")
}

func TestLatencyTrackerNanoSecondLatency(t *testing.T) {
	lt := NewLatencyTracker()

	providerID := "provider-1"

	lt.RecordLatency(providerID, 100*time.Nanosecond)

	stats := lt.GetLatencyStats(providerID)
	assert.NotNil(t, stats)
	assert.Equal(t, 100*time.Nanosecond, stats.AverageLatency, "Should handle nanosecond latency")
}

func TestWeightedRouterCalculateProviderScore(t *testing.T) {
	lt := NewLatencyTracker()
	hc := NewHealthChecker(nil)
	db := setupTestHealthCheckerDB(t)

	wr := NewWeightedRouter(lt, hc)
	hc.AddProvider("1")

	// Create provider in database
	provider := &database.Provider{
		Name:       "test-provider",
		Endpoint:   "http://localhost:9999",
		Description: "Test",
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	// Record some latency
	lt.RecordLatency("1", 100*time.Millisecond)

	score := wr.CalculateProviderScore("1", db)

	assert.Greater(t, score, 0.0, "Score should be positive")
	assert.Greater(t, score, 0.0, "Score should be positive")
}

func TestWeightedRouterCalculateProviderScoreNoLatency(t *testing.T) {
	lt := NewLatencyTracker()
	hc := NewHealthChecker(nil)
	db := setupTestHealthCheckerDB(t)

	wr := NewWeightedRouter(lt, hc)
	hc.AddProvider("1")

	// Create provider in database
	provider := &database.Provider{
		Name:       "test-provider",
		Endpoint:   "http://localhost:9999",
		Description: "Test",
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	score := wr.CalculateProviderScore("1", db)

	assert.Greater(t, score, 0.0, "Score should be positive even without latency")
}

func TestLatencyTrackerLargeSampleCount(t *testing.T) {
	lt := NewLatencyTracker()

	providerID := "provider-1"

	// Record many samples
	for i := 0; i < 1000; i++ {
		latency := time.Duration(100+i%100) * time.Millisecond
		lt.RecordLatency(providerID, latency)
	}

	stats := lt.GetLatencyStats(providerID)
	assert.Equal(t, 1000, stats.SampleCount, "Should have 1000 samples")
}

func TestLatencyTrackerDifferentLatencyPatterns(t *testing.T) {
	lt := NewLatencyTracker()

	providerID := "provider-1"

	// Increasing pattern
	for i := 1; i <= 10; i++ {
		lt.RecordLatency(providerID, time.Duration(i*10)*time.Millisecond)
	}

	stats := lt.GetLatencyStats(providerID)
	assert.Equal(t, 10*time.Millisecond, stats.MinLatency, "Min should be 10ms")
	assert.Equal(t, 100*time.Millisecond, stats.MaxLatency, "Max should be 100ms")
}

func TestLatencyTrackerConsistentLatency(t *testing.T) {
	lt := NewLatencyTracker()

	providerID := "provider-1"

	// Record same latency 10 times
	for i := 0; i < 10; i++ {
		lt.RecordLatency(providerID, 100*time.Millisecond)
	}

	stats := lt.GetLatencyStats(providerID)
	assert.Equal(t, 100*time.Millisecond, stats.MinLatency, "Min should be 100ms")
	assert.Equal(t, 100*time.Millisecond, stats.MaxLatency, "Max should be 100ms")
	assert.Equal(t, 100*time.Millisecond, stats.AverageLatency, "Average should be 100ms")
}

func TestLatencyTrackerAllStats(t *testing.T) {
	lt := NewLatencyTracker()

	providers := []string{"p1", "p2", "p3"}
	latencies := []time.Duration{
		100 * time.Millisecond,
		150 * time.Millisecond,
		200 * time.Millisecond,
	}

	for i, providerID := range providers {
		lt.RecordLatency(providerID, latencies[i])
	}

	allStats := lt.GetAllLatencyStats()
	assert.Equal(t, 3, len(allStats))

	// Verify each provider's stats
	for i, providerID := range providers {
		stats := allStats[providerID]
		assert.NotNil(t, stats)
		assert.Equal(t, providerID, stats.ProviderID)
		assert.Equal(t, latencies[i], stats.AverageLatency)
	}
}
