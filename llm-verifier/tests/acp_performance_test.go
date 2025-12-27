package tests

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/llmverifier/llmverifier"
	"github.com/llmverifier/llmverifier/config"
)

// TestACPsPerformanceBaseline establishes performance baselines for ACP detection
func TestACPsPerformanceBaseline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cfg := &config.Config{
		GlobalTimeout: 60 * time.Second,
		MaxRetries:    3,
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
		{"Very Slow Model", "very_slow", 1000 * time.Millisecond},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &PerformanceMockClient{
				ModelType:     tc.modelType,
				ResponseDelay: tc.delay,
			}

			ctx := context.Background()
			
			// Run multiple iterations for statistical significance
			iterations := 10
			times := make([]time.Duration, iterations)
			
			for i := 0; i < iterations; i++ {
				start := time.Now()
				supportsACP := verifier.TestACPs(mockClient, tc.modelType, ctx)
				duration := time.Since(start)
				
				times[i] = duration
				
				// Verify result is reasonable
				if supportsACP != true {
					t.Errorf("Expected ACP support for mock client, got %t", supportsACP)
				}
			}
			
			// Calculate statistics
			avgTime, minTime, maxTime := calculateTimeStats(times)
			
			t.Logf("Performance Results for %s:", tc.name)
			t.Logf("  Average: %s", avgTime.Round(time.Millisecond))
			t.Logf("  Min: %s", minTime.Round(time.Millisecond))
			t.Logf("  Max: %s", maxTime.Round(time.Millisecond))
			t.Logf("  StdDev: %s", calculateStdDev(times).Round(time.Millisecond))
			
			// Performance assertions
			maxAcceptableTime := 10 * time.Second // Should complete within 10 seconds
			if avgTime > maxAcceptableTime {
				t.Errorf("Average time %s exceeds maximum %s", avgTime, maxAcceptableTime)
			}
			
			// Consistency check (max should not be more than 3x min)
			if maxTime > minTime*3 {
				t.Logf("Warning: High variance in performance (max/min ratio: %.2f)", 
					float64(maxTime)/float64(minTime))
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
		GlobalTimeout: 120 * time.Second,
		MaxRetries:    3,
	}
	verifier := llmverifier.New(cfg)

	concurrencyLevels := []int{1, 5, 10, 20, 50}
	modelsPerLevel := 10

	for _, concurrency := range concurrencyLevels {
		t.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(t *testing.T) {
			start := time.Now()
			
			// Create semaphore for concurrency control
			semaphore := make(chan struct{}, concurrency)
			var wg sync.WaitGroup
			
			// Results channel
			results := make(chan struct {
				model    string
				supported bool
				duration  time.Duration
			}, modelsPerLevel)
			
			// Launch concurrent tests
			for i := 0; i < modelsPerLevel; i++ {
				wg.Add(1)
				modelName := fmt.Sprintf("concurrent-model-%d", i)
				
				go func(m string) {
					defer wg.Done()
					
					// Acquire semaphore
					semaphore <- struct{}{}
					defer func() { <-semaphore }()
					
					mockClient := &ConcurrentMockClient{
						ModelName:     m,
						ResponseDelay: 100 * time.Millisecond,
					}
					
					testStart := time.Now()
					supported := verifier.TestACPs(mockClient, m, context.Background())
					duration := time.Since(testStart)
					
					results <- struct {
						model    string
						supported bool
						duration  time.Duration
					}{m, supported, duration}
				}(modelName)
			}
			
			// Wait for completion
			go func() {
				wg.Wait()
				close(results)
			}()
			
			// Collect results
			var totalDuration time.Duration
			var testDurations []time.Duration
			successCount := 0
			
			for result := range results {
				testDurations = append(testDurations, result.duration)
				totalDuration += result.duration
				if result.supported {
					successCount++
				}
			}
			
			elapsed := time.Since(start)
			
			// Calculate statistics
			avgTime, minTime, maxTime := calculateTimeStats(testDurations)
			
			t.Logf("Concurrent Testing Results (concurrency=%d):", concurrency)
			t.Logf("  Total elapsed time: %s", elapsed.Round(time.Millisecond))
			t.Logf("  Sum of individual times: %s", totalDuration.Round(time.Millisecond))
			t.Logf("  Average test time: %s", avgTime.Round(time.Millisecond))
			t.Logf("  Min test time: %s", minTime.Round(time.Millisecond))
			t.Logf("  Max test time: %s", maxTime.Round(time.Millisecond))
			t.Logf("  Success rate: %d/%d (%.1f%%)", 
				successCount, modelsPerLevel, 
				float64(successCount)/float64(modelsPerLevel)*100)
			
			// Performance assertions
			if elapsed > 30*time.Second {
				t.Errorf("Total elapsed time %s exceeds maximum 30s", elapsed)
			}
			
			// Efficiency check (concurrent should be faster than sequential)
			expectedSequentialTime := avgTime * time.Duration(modelsPerLevel)
			if elapsed >= expectedSequentialTime {
				t.Logf("Warning: Concurrent execution not efficient (elapsed: %s, expected sequential: %s)",
					elapsed, expectedSequentialTime)
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
		GlobalTimeout: 60 * time.Second,
		MaxRetries:    3,
	}
	verifier := llmverifier.New(cfg)

	// Measure baseline memory
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)
	baselineAlloc := m1.Alloc
	
	t.Logf("Baseline memory usage: %d KB", baselineAlloc/1024)
	
	// Run multiple ACP tests
	iterations := 100
	for i := 0; i < iterations; i++ {
		mockClient := &MemoryTestClient{
			Iteration: i,
			LargeResponse: generateLargeResponse(),
		}
		
		supportsACP := verifier.TestACPs(mockClient, fmt.Sprintf("memory-test-%d", i), context.Background())
		_ = supportsACP // Use the result to prevent optimization
		
		// Force GC every 10 iterations to simulate realistic usage
		if i%10 == 0 {
			runtime.GC()
		}
	}
	
	// Measure final memory
	var m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m2)
	finalAlloc := m2.Alloc
	
	memoryGrowth := finalAlloc - baselineAlloc
	avgMemoryPerTest := memoryGrowth / uint64(iterations)
	
	t.Logf("Final memory usage: %d KB", finalAlloc/1024)
	t.Logf("Memory growth: %d KB", memoryGrowth/1024)
	t.Logf("Average memory per test: %d KB", avgMemoryPerTest/1024)
	
	// Memory assertions
	maxAcceptableGrowth := uint64(10 * 1024 * 1024) // 10 MB total
	if memoryGrowth > maxAcceptableGrowth {
		t.Errorf("Memory growth %d KB exceeds maximum %d KB", 
			memoryGrowth/1024, maxAcceptableGrowth/1024)
	}
	
	maxAcceptablePerTest := uint64(100 * 1024) // 100 KB per test
	if avgMemoryPerTest > maxAcceptablePerTest {
		t.Errorf("Average memory per test %d KB exceeds maximum %d KB",
			avgMemoryPerTest/1024, maxAcceptablePerTest/1024)
	}
}

// TestACPsResourceLimits tests behavior under resource constraints
func TestACPsResourceLimits(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource limits test in short mode")
	}

	cfg := &config.Config{
		GlobalTimeout: 5 * time.Second, // Very short timeout
		MaxRetries:    1,
	}
	verifier := llmverifier.New(cfg)

	testCases := []struct {
		name        string
		client      llmverifier.LLMClient
		expectError bool
		description string
	}{
		{
			name:        "TimeoutClient",
			client:      &TimeoutMockClient{Delay: 10 * time.Second},
			expectError: true,
			description: "Client that always times out",
		},
		{
			name:        "MemoryIntensiveClient",
			client:      &MemoryIntensiveClient{DataSize: 100 * 1024 * 1024}, // 100MB
			expectError: false,
			description: "Client that returns very large responses",
		},
		{
			name:        "RateLimitedClient",
			client:      &RateLimitedMockClient{FailRate: 0.8},
			expectError: false,
			description: "Client that frequently fails",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			start := time.Now()
			
			supportsACP := verifier.TestACPs(tc.client, "resource-test-model", ctx)
			
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
		GlobalTimeout: 300 * time.Second, // 5 minutes for large scale test
		MaxRetries:    3,
	}
	verifier := llmverifier.New(cfg)

	// Test different scales
	scales := []struct {
		name        string
		modelCount  int
		concurrency int
	}{
		{"Small Scale", 10, 2},
		{"Medium Scale", 50, 5},
		{"Large Scale", 100, 10},
		{"Extra Large Scale", 200, 20},
	}

	for _, scale := range scales {
		t.Run(scale.name, func(t *testing.T) {
			start := time.Now()
			
			// Create semaphore for concurrency control
			semaphore := make(chan struct{}, scale.concurrency)
			var wg sync.WaitGroup
			
			// Results tracking
			results := make(chan struct {
				model    string
				supported bool
				duration  time.Duration
			}, scale.modelCount)
			
			// Launch tests
			for i := 0; i < scale.modelCount; i++ {
				wg.Add(1)
				modelName := fmt.Sprintf("scale-test-%d", i)
				
				go func(m string) {
					defer wg.Done()
					
					semaphore <- struct{}{}
					defer func() { <-semaphore }()
					
					mockClient := &ScalabilityMockClient{
						ModelName:     m,
						ResponseDelay: 50 * time.Millisecond,
					}
					
					testStart := time.Now()
					supported := verifier.TestACPs(mockClient, m, context.Background())
					duration := time.Since(testStart)
					
					results <- struct {
						model    string
						supported bool
						duration  time.Duration
					}{m, supported, duration}
				}(modelName)
			}
			
			// Wait for completion
			go func() {
				wg.Wait()
				close(results)
			}()
			
			// Collect results
			successCount := 0
			totalDuration := time.Duration(0)
			
			for result := range results {
				totalDuration += result.duration
				if result.supported {
					successCount++
				}
			}
			
			elapsed := time.Since(start)
			
			t.Logf("Scalability Test Results (%s):", scale.name)
			t.Logf("  Models tested: %d", scale.modelCount)
			t.Logf("  Concurrency: %d", scale.concurrency)
			t.Logf("  Total elapsed time: %s", elapsed.Round(time.Second))
			t.Logf("  Average time per model: %s", (totalDuration/time.Duration(scale.modelCount)).Round(time.Millisecond))
			t.Logf("  Success rate: %d/%d (%.1f%%)", 
				successCount, scale.modelCount, 
				float64(successCount)/float64(scale.modelCount)*100)
			
			// Scalability assertions
			maxAcceptableTime := time.Duration(scale.modelCount) * 2 * time.Second // 2s per model max
			if elapsed > maxAcceptableTime {
				t.Errorf("Elapsed time %s exceeds maximum %s", elapsed, maxAcceptableTime)
			}
			
			// Linear scalability check (should not grow exponentially)
			if scale.modelCount > 10 {
				// Check that time growth is sub-linear
				expectedLinearTime := time.Duration(scale.modelCount/10) * time.Second
				if elapsed > expectedLinearTime*3 {
					t.Logf("Warning: Scalability may be degrading (elapsed: %s, expected linear: %s)",
						elapsed, expectedLinearTime)
				}
			}
		})
	}
}

// Helper types for performance testing

type PerformanceMockClient struct {
	ModelType     string
	ResponseDelay time.Duration
}

func (c *PerformanceMockClient) ChatCompletion(ctx context.Context, request llmverifier.ChatCompletionRequest) (*llmverifier.ChatCompletionResponse, error) {
	select {
	case <-time.After(c.ResponseDelay):
		return generatePerformanceResponse(c.ModelType, request.Messages[0].Content), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type ConcurrentMockClient struct {
	ModelName     string
	ResponseDelay time.Duration
}

func (c *ConcurrentMockClient) ChatCompletion(ctx context.Context, request llmverifier.ChatCompletionRequest) (*llmverifier.ChatCompletionResponse, error) {
	select {
	case <-time.After(c.ResponseDelay):
		return generateConcurrentResponse(c.ModelName, request.Messages[0].Content), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type MemoryTestClient struct {
	Iteration     int
	LargeResponse string
}

func (c *MemoryTestClient) ChatCompletion(ctx context.Context, request llmverifier.ChatCompletionRequest) (*llmverifier.ChatCompletionResponse, error) {
	// Return large response to test memory handling
	return &llmverifier.ChatCompletionResponse{
		Choices: []llmverifier.Choice{
			{
				Message: llmverifier.Message{
					Role:    "assistant",
					Content: c.LargeResponse,
				},
			},
		},
	}, nil
}

type TimeoutMockClient struct {
	Delay time.Duration
}

func (c *TimeoutMockClient) ChatCompletion(ctx context.Context, request llmverifier.ChatCompletionRequest) (*llmverifier.ChatCompletionResponse, error) {
	select {
	case <-time.After(c.Delay):
		return &llmverifier.ChatCompletionResponse{
			Choices: []llmverifier.Choice{
				{
					Message: llmverifier.Message{
						Role:    "assistant",
						Content: "This response came after a delay",
					},
				},
			},
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type MemoryIntensiveClient struct {
	DataSize int
}

func (c *MemoryIntensiveClient) ChatCompletion(ctx context.Context, request llmverifier.ChatCompletionRequest) (*llmverifier.ChatCompletionResponse, error) {
	// Generate large response
	largeData := make([]byte, c.DataSize)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	
	return &llmverifier.ChatCompletionResponse{
		Choices: []llmverifier.Choice{
			{
				Message: llmverifier.Message{
					Role:    "assistant",
					Content: string(largeData),
				},
			},
		},
	}, nil
}

type RateLimitedMockClient struct {
	FailRate float64
}

func (c *RateLimitedMockClient) ChatCompletion(ctx context.Context, request llmverifier.ChatCompletionRequest) (*llmverifier.ChatCompletionResponse, error) {
	if float64(time.Now().UnixNano()%100)/100 < c.FailRate {
		return nil, fmt.Errorf("rate limited")
	}
	
	return &llmverifier.ChatCompletionResponse{
		Choices: []llmverifier.Choice{
			{
				Message: llmverifier.Message{
					Role:    "assistant",
					Content: "Success response",
				},
			},
		},
	}, nil
}

type ScalabilityMockClient struct {
	ModelName     string
	ResponseDelay time.Duration
}

func (c *ScalabilityMockClient) ChatCompletion(ctx context.Context, request llmverifier.ChatCompletionRequest) (*llmverifier.ChatCompletionResponse, error) {
	select {
	case <-time.After(c.ResponseDelay):
		return generateScalabilityResponse(c.ModelName, request.Messages[0].Content), nil
	case <-ctx.Done():
		return nil, ctx.Err()
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

func generatePerformanceResponse(modelType, content string) *llmverifier.ChatCompletionResponse {
	// Generate appropriate response based on request content
	responseText := "Performance test response"
	
	if contains(content, "jsonrpc") {
		responseText = `{"jsonrpc":"2.0","result":{"items":[{"label":"test","kind":"function"}]}}`
	} else if contains(content, "tool") {
		responseText = "Using tool: file_read with parameters"
	} else if contains(content, "context") {
		responseText = "Maintaining context across conversation"
	} else if contains(content, "function") {
		responseText = "func test() string { return \"performance\" }"
	} else if contains(content, "error") {
		responseText = "Error detected on line 5: syntax error"
	}
	
	return &llmverifier.ChatCompletionResponse{
		Choices: []llmverifier.Choice{
			{
				Message: llmverifier.Message{
					Role:    "assistant",
					Content: responseText,
				},
			},
		},
	}
}

func generateConcurrentResponse(modelName, content string) *llmverifier.ChatCompletionResponse {
	return &llmverifier.ChatCompletionResponse{
		Choices: []llmverifier.Choice{
			{
				Message: llmverifier.Message{
					Role:    "assistant",
					Content: fmt.Sprintf("Concurrent response for %s", modelName),
				},
			},
		},
	}
}

func generateScalabilityResponse(modelName, content string) *llmverifier.ChatCompletionResponse {
	return &llmverifier.ChatCompletionResponse{
		Choices: []llmverifier.Choice{
			{
				Message: llmverifier.Message{
					Role:    "assistant",
					Content: fmt.Sprintf("Scalability test response from %s", modelName),
				},
			},
		},
	}
}