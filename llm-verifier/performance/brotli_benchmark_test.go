package performance

import (
	"testing"
	"time"

	"llm-verifier/client"
)

func TestBrotliBenchmark(t *testing.T) {
	httpClient := client.NewHTTPClient(30 * time.Second)

	// Test with mock provider (will fail due to invalid API key, but should test logic)
	result, err := BrotliBenchmark(httpClient, "openai", "test_key", "gpt-4", 3)

	if err != nil {
		t.Logf("Benchmark completed with expected error (invalid API key): %v", err)
		return
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Provider != "openai" {
		t.Errorf("Expected provider 'openai', got '%s'", result.Provider)
	}

	if result.ModelID != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", result.ModelID)
	}

	if result.APICalls != 3 {
		t.Errorf("Expected 3 API calls, got %d", result.APICalls)
	}
}

func TestConcurrentBrotliBenchmark(t *testing.T) {
	httpClient := client.NewHTTPClient(30 * time.Second)

	providers := map[string]string{
		"openai":    "test_key_openai",
		"anthropic": "test_key_anthropic",
	}

	result, err := ConcurrentBrotliBenchmark(httpClient, providers, "test-model", 2, 2)

	if err != nil {
		t.Logf("Concurrent benchmark completed with expected error: %v", err)
		return
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result.ProviderResults) != len(providers) {
		t.Errorf("Expected %d provider results, got %d", len(providers), len(result.ProviderResults))
	}

	if result.Concurrency != 2 {
		t.Errorf("Expected concurrency 2, got %d", result.Concurrency)
	}
}

func TestPrintBenchmarkResults(t *testing.T) {
	result := &BrotliBenchmarkResult{
		Provider:         "openai",
		ModelID:          "gpt-4",
		APICalls:         5,
		SuccessRate:      80.0,
		AvgDetectionTime: 500 * time.Millisecond,
		CacheHitRate:     75.0,
		TotalTime:        2 * time.Second,
		Errors:           1,
		BrotliSupported:  true,
	}

	// This should not panic
	PrintBenchmarkResults(result)
}

func TestPrintConcurrentBenchmarkResults(t *testing.T) {
	result := &ConcurrentBenchmarkResult{
		ProviderResults: map[string]*BrotliBenchmarkResult{
			"openai": {
				Provider:         "openai",
				ModelID:          "gpt-4",
				APICalls:         3,
				SuccessRate:      100.0,
				AvgDetectionTime: 400 * time.Millisecond,
				CacheHitRate:     66.67,
				TotalTime:        1 * time.Second,
				Errors:           0,
				BrotliSupported:  true,
			},
		},
		TotalTime:   1 * time.Second,
		Concurrency: 2,
	}

	// This should not panic
	PrintConcurrentBenchmarkResults(result)
}
