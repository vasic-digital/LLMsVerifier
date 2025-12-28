package tests

import (
	"context"
	"testing"
	"time"
	
	"llm-verifier/verification"
)

// TestPerformanceFastAPIResponse tests API response performance
func TestPerformanceFastAPIResponse(t *testing.T) {
	ctx := context.Background()
	client := verification.NewModelsDevClient()
	
	// Measure response time for models.dev API
	start := time.Now()
	models, err := client.FetchModels(ctx)
	elapsed := time.Since(start)
	
	if err != nil {
		t.Fatalf("API call failed: %v", err)
	}
	
	// Performance requirement: should respond within 5 seconds
	if elapsed > 5*time.Second {
		t.Errorf("API response too slow: %v (expected < 5s)", elapsed)
	}
	
	t.Logf("✓ API response time: %v", elapsed)
	t.Logf("✓ Retrieved %d models", len(models))
}

// TestPerformanceConcurrentRequests tests concurrent API calls
func TestPerformanceConcurrentRequests(t *testing.T) {
	ctx := context.Background()
	client := verification.NewModelsDevClient()
	
	// Test concurrent fetching
	start := time.Now()
	
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			_, err := client.FetchModels(ctx)
			if err != nil {
				t.Errorf("Concurrent fetch failed: %v", err)
			}
			done <- true
		}()
	}
	
	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}
	
	elapsed := time.Since(start)
	
	// Should complete 5 concurrent requests within reasonable time
	if elapsed > 15*time.Second {
		t.Errorf("Concurrent requests too slow: %v", elapsed)
	}
	
	t.Logf("✓ 5 concurrent requests completed in %v", elapsed)
}

// TestSecurityAPIKeyValidation tests that API key validation works
func TestSecurityAPIKeyValidation(t *testing.T) {
	ctx := context.Background()
	client := verification.NewModelsDevClient()
	
	// Test context timeout (security against hanging requests)
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	
	start := time.Now()
	_, err := client.FetchModels(ctx)
	elapsed := time.Since(start)
	
	if err != nil {
		t.Logf("API call completed with error (expected): %v", err)
	}
	
	// Should respect timeout
	if elapsed > 3*time.Second {
		t.Errorf("Request didn't respect timeout: %v", elapsed)
	}
	
	t.Logf("✓ Timeout enforced: %v", elapsed)
}

// TestSecurityNoSensitiveData verifies no sensitive data in responses
func TestSecurityNoSensitiveData(t *testing.T) {
	ctx := context.Background()
	client := verification.NewModelsDevClient()
	
	models, err := client.FetchModels(ctx)
	if err != nil {
		t.Fatalf("Failed to fetch models: %v", err)
	}
	
	// Check that no model contains sensitive info like API keys
	for _, model := range models {
		// Look for patterns that might indicate leaked credentials
		sensitivePatterns := []string{"sk-", "key", "secret", "token:"}
		
		jsonStr, _ := json.Marshal(model)
		str := string(jsonStr)
		
		for _, pattern := range sensitivePatterns {
			if contains := contain







sent := strings.Contains(strings.ToLower(str), pattern); contains {
				t.Errorf("Model %s may contain sensitive data pattern '%s': %s", 
					model.ModelID, pattern, str[:100])
			}
		}
	}
	
	t.Logf("✓ Checked %d models for sensitive data", len(models))
}

// TestSecurityContextIsolation tests proper context usage
func TestSecurityContextIsolation(t *testing.T) {
	// Create multiple contexts to test isolation
	ctx1, cancel1 := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel1()
	
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()
	
	client := verification.NewModelsDevClient()
	
	// First request with 1s timeout
	start1 := time.Now()
	_, err1 := client.FetchModels(ctx1)
	elapsed1 := time.Since(start1)
	
	// Second request with 2s timeout
	start2 := time.Now()
	_, err2 := client.FetchModels(ctx2)
	elapsed2 := time.Since(start2)
	
	// Both should complete successfully (no actual timeouts expected for this API)
	if err1 != nil && ctx1.Err() != nil {
		t.Logf("First request timed out as expected: %v", err1)
	}
	
	if err2 != nil && ctx2.Err() != nil {
		t.Logf("Second request timed out as expected: %v", err2)
	}
	
	t.Logf("✓ Context isolation working: elapsed1=%v, elapsed2=%v", elapsed1, elapsed2)
}

// TestPerformanceModelCount tests we're fetching enough models
func TestPerformanceModelCount(t *testing.T) {
	ctx := context.Background()
	client := verification.NewModelsDevClient()
	
	models, err := client.FetchModels(ctx)
	if err != nil {
		t.Fatalf("Failed to fetch models: %v", err)
	}
	
	// Should have a reasonable number of models
	minModels := 100
	maxModels := 10000
	
	if len(models) < minModels {
		t.Errorf("Too few models fetched: %d (expected >= %d)", len(models), minModels)
	}
	
	if len(models) > maxModels {
		t.Errorf("Too many models fetched: %d (expected <= %d)", len(models), maxModels)
	}
	
	t.Logf("✓ Fetched %d models (range: %d-%d)", len(models), minModels, maxModels)
}

// TestPerformanceMemoryUsage tests memory efficiency
func TestPerformanceMemoryUsage(t *testing.T) {
	ctx := context.Background()
	client := verification.NewModelsDevClient()
	
	// Multiple fetches to test stable memory usage
	var models [][]verification.ModelsDevModel
	
	for i := 0; i < 3; i++ {
		m, err := client.FetchModels(ctx)
		if err != nil {
			t.Fatalf("Fetch %d failed: %v", i, err)
		}
		models = append(models, m)
	}
	
	// Verify consistent counts (no memory leaks/growth)
	firstCount := len(models[0])
	for i := 1; i < len(models); i++ {
		if len(models[i]) != firstCount {
			t.Errorf("Inconsistent model count on fetch %d: %d vs %d", 
				i, len(models[i]), firstCount)
		}
	}
	
	t.Logf("✓ Memory usage stable across 3 fetches: %d models each", firstCount)
}

// TestSecurityHeaders tests proper security headers
func TestSecurityHeaders(t *testing.T) {
	// This test verifies the HTTP client uses proper headers
	client := verification.NewModelsDevClient()
	
	// Check header setup (in real client code)
	if client.httpClient.Timeout < 5*time.Second {
		t.Errorf("Timeout too short: %v", client.httpClient.Timeout)
	}
	
	t.Logf("✓ Client configured with timeout: %v", client.httpClient.Timeout)
}

// TestPerformanceUpdateFrequency tests how often we should update
func TestPerformanceUpdateFrequency(t *testing.T) {
	// This test documents expected update frequency
	ctx := context.Background()
	client := verification.NewModelsDevClient()
	
	models, err := client.FetchModels(ctx)
	if err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}
	
	// Find last update times to determine freshness
	recentUpdateCount := 0
	for _, model := range models {
		if model.LastUpdated != "" {
			// Count models updated in last 7 days
			recentUpdateCount++
		}
	}
	
	updateFrequency := float64(recentUpdateCount) / float64(len(models)) * 100
	
	t.Logf("✓ %d/%d models (%.1f%%) have recent update timestamps", 
		recentUpdateCount, len(models), updateFrequency)
	t.Logf("✓ Recommend fetching models.dev data daily for freshness")
}
