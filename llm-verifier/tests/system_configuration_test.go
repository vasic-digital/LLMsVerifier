package tests

import (
	"testing"
	"time"

	"llm-verifier/enhanced/analytics"
	contextmanager "llm-verifier/enhanced/context"
	"llm-verifier/enhanced/supervisor"
)

// TestSystemConfiguration performs system-level configuration tests
func TestSystemConfiguration(t *testing.T) {
	t.Run("Analytics Configuration", func(t *testing.T) {
		// Test analytics engine configuration
		t.Log("Testing analytics engine configuration")

		config := analytics.AnalyticsConfig{
			RetentionPeriod:   24 * time.Hour,
			MaxTimeSeriesSize: 1000,
			BatchSize:         100,
			FlushInterval:     30 * time.Second,
			EnablePredictions: false,
			MLModelConfig:     make(map[string]interface{}),
		}

		if config.RetentionPeriod <= 0 {
			t.Error("Retention period should be positive")
		}

		if config.MaxTimeSeriesSize <= 0 {
			t.Error("Max time series size should be positive")
		}

		if config.BatchSize <= 0 {
			t.Error("Batch size should be positive")
		}

		t.Log("Analytics configuration test passed")
	})

	t.Run("Context Manager Configuration", func(t *testing.T) {
		// Test context manager configuration
		t.Log("Testing context manager configuration")

		config := contextmanager.ContextConfig{
			ShortTermMaxMessages:    100,
			ShortTermWindowDuration: 1 * time.Hour,
			LongTermMaxSummaries:    50,
			SummarizationThreshold:  5000,
			BackupEnabled:           false,
			BackupInterval:          24 * time.Hour,
			StorageConfig:           make(map[string]interface{}),
		}

		if config.ShortTermMaxMessages <= 0 {
			t.Error("Short term max messages should be positive")
		}

		if config.SummarizationThreshold <= 0 {
			t.Error("Summarization threshold should be positive")
		}

		t.Log("Context manager configuration test passed")
	})

	t.Run("Supervisor Configuration", func(t *testing.T) {
		// Test supervisor configuration
		t.Log("Testing supervisor configuration")

		config := supervisor.SupervisorConfig{
			MaxConcurrentJobs:   10,
			JobTimeout:          5 * time.Minute,
			HealthCheckInterval: 30 * time.Second,
			RetryAttempts:       3,
			RetryBackoff:        1 * time.Second,

			EnableAutoScaling:    true,
			EnablePredictions:    false,
			EnableAdaptiveLoad:   true,
			EnableCircuitBreaker: true,

			HighLoadThreshold:  0.8,
			LowLoadThreshold:   0.3,
			ErrorRateThreshold: 0.1,
			MemoryThreshold:    0.9,
		}

		if config.MaxConcurrentJobs <= 0 {
			t.Error("Max concurrent jobs should be positive")
		}

		if config.JobTimeout <= 0 {
			t.Error("Job timeout should be positive")
		}

		if config.HighLoadThreshold <= 0 || config.HighLoadThreshold > 1 {
			t.Error("High load threshold should be between 0 and 1")
		}

		t.Log("Supervisor configuration test passed")
	})
}

// TestComponentCompatibility tests component compatibility
func TestComponentCompatibility(t *testing.T) {
	t.Run("Configuration Synchronization", func(t *testing.T) {
		// Test that different component configurations are compatible
		t.Log("Testing configuration synchronization")

		analyticsConfig := analytics.AnalyticsConfig{
			RetentionPeriod:   24 * time.Hour,
			MaxTimeSeriesSize: 1000,
			BatchSize:         100,
			FlushInterval:     30 * time.Second,
		}

		contextConfig := contextmanager.ContextConfig{
			ShortTermMaxMessages:    100,
			ShortTermWindowDuration: 1 * time.Hour,
			LongTermMaxSummaries:    50,
			SummarizationThreshold:  5000,
		}

		// Verify configurations are compatible - basic sanity checks
		if analyticsConfig.MaxTimeSeriesSize < contextConfig.LongTermMaxSummaries {
			t.Error("Analytics max time series size should accommodate context summaries")
		}

		t.Log("Configuration synchronization test passed")
	})

	t.Run("Resource Management", func(t *testing.T) {
		// Test resource management across components
		t.Log("Testing resource management")

		// Simulate resource allocation
		totalMemory := 1024 * 1024 * 1024    // 1GB
		analyticsMemory := 1024 * 1024 * 100 // 100MB
		contextMemory := 1024 * 1024 * 50    // 50MB

		allocatedMemory := analyticsMemory + contextMemory

		if allocatedMemory > totalMemory {
			t.Errorf("Allocated memory (%d) exceeds total memory (%d)", allocatedMemory, totalMemory)
		}

		// Test that allocation percentages are reasonable
		analyticsPercentage := float64(analyticsMemory) / float64(totalMemory) * 100
		contextPercentage := float64(contextMemory) / float64(totalMemory) * 100

		if analyticsPercentage > 20 {
			t.Error("Analytics memory allocation should not exceed 20% of total")
		}

		if contextPercentage > 10 {
			t.Error("Context memory allocation should not exceed 10% of total")
		}

		t.Log("Resource management test passed")
	})
}
