package tests

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"llm-verifier/config"
	"llm-verifier/llmverifier"
)

// TestACPsPerformanceBaseline establishes performance baselines for ACP detection
func TestACPsPerformanceBaseline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cfg := &config.Config{
		Timeout: 60 * time.Second,
		Global: config.GlobalConfig{
			MaxRetries: 3,
		},
	}
	verifier := llmverifier.New(cfg)

	// Test models with different characteristics
	testCases := []struct {
		name      string
		modelType string
		delay     time.Duration
	}{
		{"Fast Model", "fast", 50 * time.Millisecond},
		{"Medium Model", "medium", 200 * time.Millisecond},
		{"Slow Model", "slow", 500 * time.Millisecond},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a real LLMClient pointing to a non-existent endpoint
			// This tests error handling performance
			mockClient := llmverifier.NewLLMClient("http://localhost:9999", "test-key", nil)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			start := time.Now()
			supportsACP := verifier.TestACPs(mockClient, tc.modelType, ctx)
			duration := time.Since(start)

			t.Logf("Performance Results for %s:", tc.name)
			t.Logf("  Duration: %s", duration.Round(time.Millisecond))
			t.Logf("  Result: %t", supportsACP)

			// Performance assertions
			maxAcceptableTime := 10 * time.Second
			if duration > maxAcceptableTime {
				t.Errorf("Duration %s exceeds maximum %s", duration, maxAcceptableTime)
			}
		})
	}
}

// TestACPsConcurrentPerformance tests ACP detection under concurrent load
func TestACPsConcurrentPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent performance test in short mode")
	}

	cfg := &config.Config{
		Timeout: 120 * time.Second,
		Global: config.GlobalConfig{
			MaxRetries: 3,
		},
	}
	verifier := llmverifier.New(cfg)

	concurrencyLevels := []int{1, 5, 10}
	modelsPerLevel := 5

	for _, concurrency := range concurrencyLevels {
		t.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(t *testing.T) {
			start := time.Now()

			// Create semaphore for concurrency control
			semaphore := make(chan struct{}, concurrency)
			var wg sync.WaitGroup

			// Results tracking
			successCount := 0
			var mu sync.Mutex

			// Launch concurrent tests
			for i := 0; i < modelsPerLevel; i++ {
				wg.Add(1)
				modelName := fmt.Sprintf("concurrent-model-%d", i)

				go func(m string) {
					defer wg.Done()

					// Acquire semaphore
					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					mockClient := llmverifier.NewLLMClient("http://localhost:9999", "test-key", nil)

					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()

					supported := verifier.TestACPs(mockClient, m, ctx)

					mu.Lock()
					if supported {
						successCount++
					}
					mu.Unlock()
				}(modelName)
			}

			// Wait for completion
			wg.Wait()

			elapsed := time.Since(start)

			t.Logf("Concurrent Testing Results (concurrency=%d):", concurrency)
			t.Logf("  Total elapsed time: %s", elapsed.Round(time.Millisecond))
			t.Logf("  Success rate: %d/%d", successCount, modelsPerLevel)

			// Performance assertions
			if elapsed > 30*time.Second {
				t.Errorf("Total elapsed time %s exceeds maximum 30s", elapsed)
			}
		})
	}
}

// TestACPsMemoryUsage tests memory consumption during ACP detection
func TestACPsMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory usage test in short mode")
	}

	cfg := &config.Config{
		Timeout: 60 * time.Second,
		Global: config.GlobalConfig{
			MaxRetries: 3,
		},
	}
	verifier := llmverifier.New(cfg)

	// Measure baseline memory
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)
	baselineAlloc := m1.Alloc

	t.Logf("Baseline memory usage: %d KB", baselineAlloc/1024)

	// Run multiple ACP tests
	iterations := 10
	for i := 0; i < iterations; i++ {
		mockClient := llmverifier.NewLLMClient("http://localhost:9999", "test-key", nil)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		supportsACP := verifier.TestACPs(mockClient, fmt.Sprintf("memory-test-%d", i), ctx)
		cancel()
		_ = supportsACP

		// Force GC every 5 iterations
		if i%5 == 0 {
			runtime.GC()
		}
	}

	// Measure final memory
	var m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m2)
	finalAlloc := m2.Alloc

	memoryGrowth := int64(finalAlloc) - int64(baselineAlloc)

	t.Logf("Final memory usage: %d KB", finalAlloc/1024)
	t.Logf("Memory growth: %d KB", memoryGrowth/1024)

	// Memory assertions
	maxAcceptableGrowth := int64(50 * 1024 * 1024) // 50 MB total
	if memoryGrowth > maxAcceptableGrowth {
		t.Errorf("Memory growth %d KB exceeds maximum %d KB",
			memoryGrowth/1024, maxAcceptableGrowth/1024)
	}
}

// TestACPsResourceLimits tests behavior under resource constraints
func TestACPsResourceLimits(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource limits test in short mode")
	}

	cfg := &config.Config{
		Timeout: 5 * time.Second, // Very short timeout
		Global: config.GlobalConfig{
			MaxRetries: 1,
		},
	}
	verifier := llmverifier.New(cfg)

	testCases := []struct {
		name        string
		description string
	}{
		{
			name:        "TimeoutClient",
			description: "Client that always times out",
		},
		{
			name:        "NonExistentEndpoint",
			description: "Client with non-existent endpoint",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := llmverifier.NewLLMClient("http://localhost:9999", "test-key", nil)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			start := time.Now()
			supportsACP := verifier.TestACPs(mockClient, "resource-test-model", ctx)
			duration := time.Since(start)

			t.Logf("Resource limit test '%s':", tc.description)
			t.Logf("  Result: %t", supportsACP)
			t.Logf("  Duration: %s", duration.Round(time.Millisecond))

			// Verify reasonable behavior under constraints
			maxAcceptableDuration := 10 * time.Second
			if duration > maxAcceptableDuration {
				t.Errorf("Duration %s exceeds maximum %s", duration, maxAcceptableDuration)
			}
		})
	}
}

// TestACPsScalability tests ACP detection scalability
func TestACPsScalability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scalability test in short mode")
	}

	cfg := &config.Config{
		Timeout: 300 * time.Second, // 5 minutes for large scale test
		Global: config.GlobalConfig{
			MaxRetries: 3,
		},
	}
	verifier := llmverifier.New(cfg)

	// Test different scales
	scales := []struct {
		name        string
		modelCount  int
		concurrency int
	}{
		{"Small Scale", 5, 2},
		{"Medium Scale", 10, 5},
	}

	for _, scale := range scales {
		t.Run(scale.name, func(t *testing.T) {
			start := time.Now()

			// Create semaphore for concurrency control
			semaphore := make(chan struct{}, scale.concurrency)
			var wg sync.WaitGroup

			// Results tracking
			successCount := 0
			var mu sync.Mutex

			// Launch tests
			for i := 0; i < scale.modelCount; i++ {
				wg.Add(1)
				modelName := fmt.Sprintf("scale-test-%d", i)

				go func(m string) {
					defer wg.Done()

					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					mockClient := llmverifier.NewLLMClient("http://localhost:9999", "test-key", nil)

					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					defer cancel()

					supported := verifier.TestACPs(mockClient, m, ctx)

					mu.Lock()
					if supported {
						successCount++
					}
					mu.Unlock()
				}(modelName)
			}

			// Wait for completion
			wg.Wait()

			elapsed := time.Since(start)

			t.Logf("Scalability Test Results (%s):", scale.name)
			t.Logf("  Models tested: %d", scale.modelCount)
			t.Logf("  Concurrency: %d", scale.concurrency)
			t.Logf("  Total elapsed time: %s", elapsed.Round(time.Second))
			t.Logf("  Success rate: %d/%d (%.1f%%)",
				successCount, scale.modelCount,
				float64(successCount)/float64(scale.modelCount)*100)

			// Scalability assertions
			maxAcceptableTime := time.Duration(scale.modelCount) * 5 * time.Second
			if elapsed > maxAcceptableTime {
				t.Errorf("Elapsed time %s exceeds maximum %s", elapsed, maxAcceptableTime)
			}
		})
	}
}

// Helper functions
func calculateTimeStats(times []time.Duration) (avg, min, max time.Duration) {
	if len(times) == 0 {
		return 0, 0, 0
	}

	min = times[0]
	max = times[0]
	total := time.Duration(0)

	for _, d := range times {
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
		total += d
	}

	avg = total / time.Duration(len(times))
	return avg, min, max
}

func calculateStdDev(times []time.Duration) time.Duration {
	if len(times) <= 1 {
		return 0
	}

	avg, _, _ := calculateTimeStats(times)

	var sumSquares float64
	for _, d := range times {
		diff := float64(d - avg)
		sumSquares += diff * diff
	}

	variance := sumSquares / float64(len(times)-1)
	stdDev := time.Duration(float64(time.Nanosecond) * sqrt(variance))

	return stdDev
}

func sqrt(x float64) float64 {
	if x == 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

func generateLargeResponse() string {
	// Generate a large response for memory testing
	var response strings.Builder
	response.Grow(1024 * 1024) // 1MB

	for i := 0; i < 10000; i++ {
		response.WriteString(fmt.Sprintf("Generated code block %d: \n", i))
		response.WriteString("```go\n")
		response.WriteString("func example() string {\n")
		response.WriteString("    return \"large response\"\n")
		response.WriteString("}\n")
		response.WriteString("```\n\n")
	}

	return response.String()
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
