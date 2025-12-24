package performance

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"llm-verifier/client"
)

// BrotliBenchmarkResult represents the result of a Brotli performance benchmark
type BrotliBenchmarkResult struct {
	Provider         string        `json:"provider"`
	ModelID          string        `json:"model_id"`
	APICalls         int           `json:"api_calls"`
	SuccessRate      float64       `json:"success_rate"`
	AvgDetectionTime time.Duration `json:"avg_detection_time"`
	CacheHitRate     float64       `json:"cache_hit_rate"`
	TotalTime        time.Duration `json:"total_time"`
	Errors           int           `json:"errors"`
	BrotliSupported  bool          `json:"brotli_supported"`
}

// BrotliBenchmark runs performance benchmarks for Brotli detection
func BrotliBenchmark(httpClient *client.HTTPClient, provider, apiKey, modelID string, iterations int) (*BrotliBenchmarkResult, error) {
	if iterations <= 0 {
		iterations = 10
	}

	var (
		successCount int
		errorCount   int
		totalTime    time.Duration
		cacheHits    int
		cacheMisses  int
	)

	var brotliSupported bool

	// Run iterations
	for i := 0; i < iterations; i++ {
		startTime := time.Now()

		supportsBrotli, err := httpClient.TestBrotliSupport(context.Background(), provider, apiKey, modelID)

		duration := time.Since(startTime)
		totalTime += duration

		if err != nil {
			errorCount++
			log.Printf("Brotli benchmark iteration %d failed: %v", i+1, err)
			continue
		}

		successCount++

		// Track cache hits/misses
		if i == 0 {
			cacheMisses++
		} else {
			cacheHits++
		}

		// Record the Brotli support result
		brotliSupported = supportsBrotli
	}

	// Calculate metrics
	successRate := float64(successCount) / float64(iterations) * 100
	avgDetectionTime := totalTime / time.Duration(successCount)

	cacheHitRate := 0.0
	if cacheHits+cacheMisses > 0 {
		cacheHitRate = float64(cacheHits) / float64(cacheHits+cacheMisses) * 100
	}

	return &BrotliBenchmarkResult{
		Provider:         provider,
		ModelID:          modelID,
		APICalls:         iterations,
		SuccessRate:      successRate,
		AvgDetectionTime: avgDetectionTime,
		CacheHitRate:     cacheHitRate,
		TotalTime:        totalTime,
		Errors:           errorCount,
		BrotliSupported:  brotliSupported,
	}, nil
}

// ConcurrentBrotliBenchmark runs Brotli benchmarks concurrently for multiple providers
type ConcurrentBenchmarkResult struct {
	ProviderResults map[string]*BrotliBenchmarkResult `json:"provider_results"`
	TotalTime       time.Duration                     `json:"total_time"`
	Concurrency     int                               `json:"concurrency"`
}

func ConcurrentBrotliBenchmark(httpClient *client.HTTPClient, providers map[string]string, modelID string, iterations int, concurrency int) (*ConcurrentBenchmarkResult, error) {
	if concurrency <= 0 {
		concurrency = 3
	}

	var wg sync.WaitGroup
	results := make(map[string]*BrotliBenchmarkResult)
	var mu sync.Mutex

	semaphore := make(chan struct{}, concurrency)

	startTime := time.Now()

	for provider, apiKey := range providers {
		wg.Add(1)

		go func(p, key string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result, err := BrotliBenchmark(httpClient, p, key, modelID, iterations)
			if err != nil {
				log.Printf("Benchmark failed for provider %s: %v", p, err)
				return
			}

			mu.Lock()
			results[p] = result
			mu.Unlock()
		}(provider, apiKey)
	}

	wg.Wait()
	totalTime := time.Since(startTime)

	return &ConcurrentBenchmarkResult{
		ProviderResults: results,
		TotalTime:       totalTime,
		Concurrency:     concurrency,
	}, nil
}

// PrintBenchmarkResults prints benchmark results in a readable format
func PrintBenchmarkResults(result *BrotliBenchmarkResult) {
	fmt.Printf("\n=== Brotli Benchmark Results ===\n")
	fmt.Printf("Provider: %s\n", result.Provider)
	fmt.Printf("Model: %s\n", result.ModelID)
	fmt.Printf("API Calls: %d\n", result.APICalls)
	fmt.Printf("Success Rate: %.2f%%\n", result.SuccessRate)
	fmt.Printf("Average Detection Time: %s\n", result.AvgDetectionTime)
	fmt.Printf("Cache Hit Rate: %.2f%%\n", result.CacheHitRate)
	fmt.Printf("Total Time: %s\n", result.TotalTime)
	fmt.Printf("Errors: %d\n", result.Errors)
	fmt.Printf("Brotli Supported: %t\n", result.BrotliSupported)
}

// PrintConcurrentBenchmarkResults prints concurrent benchmark results
func PrintConcurrentBenchmarkResults(result *ConcurrentBenchmarkResult) {
	fmt.Printf("\n=== Concurrent Brotli Benchmark Results ===\n")
	fmt.Printf("Concurrency: %d\n", result.Concurrency)
	fmt.Printf("Total Time: %s\n", result.TotalTime)
	fmt.Printf("\nProvider Results:\n")

	for provider, providerResult := range result.ProviderResults {
		fmt.Printf("\n--- %s ---\n", provider)
		fmt.Printf("  Model: %s\n", providerResult.ModelID)
		fmt.Printf("  Success Rate: %.2f%%\n", providerResult.SuccessRate)
		fmt.Printf("  Avg Detection Time: %s\n", providerResult.AvgDetectionTime)
		fmt.Printf("  Cache Hit Rate: %.2f%%\n", providerResult.CacheHitRate)
		fmt.Printf("  Brotli Supported: %t\n", providerResult.BrotliSupported)
	}
}
