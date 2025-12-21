package performance

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Simple load test without external dependencies
func TestLoad(t *testing.T) {
	// Test configuration
	targetURL := "http://localhost:8080/api/v1/health" // Health endpoint
	concurrentUsers := 5
	duration := 1 * time.Minute

	fmt.Printf("Starting simple load test against %s\n", targetURL)
	fmt.Printf("Concurrent users: %d\n", concurrentUsers)
	fmt.Printf("Test duration: %v\n", duration)

	// Metrics
	var totalRequests int64
	var successRequests int64
	var failedRequests int64
	var startTime = time.Now()

	var wg sync.WaitGroup

	// Test function
	testRequest := func(userID int) {
		defer wg.Done()

		requestCount := 0
		endTime := time.Now().Add(duration)

		for time.Now().Before(endTime) {
			// Create a simple GET request to health endpoint
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Simple HTTP request
			req, err := http.NewRequest("GET", targetURL, nil)
			if err != nil {
				t.Logf("Error creating request for user %d: %v", userID, err)
				continue
			}

			// Set timeout via context
			req = req.WithContext(ctx)

			// Make request
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Logf("Request failed for user %d: %v", userID, err)
				atomic.AddInt64(&failedRequests, 1)
			} else {
				if resp.StatusCode == 200 {
					atomic.AddInt64(&successRequests, 1)
				} else {
					atomic.AddInt64(&failedRequests, 1)
				}
				resp.Body.Close()
			}

			atomic.AddInt64(&totalRequests, 1)
			requestCount++

			// Small delay between requests
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Start all concurrent users
	for i := 0; i < concurrentUsers; i++ {
		wg.Add(1)
		go testRequest(i)
	}

	// Wait for completion
	wg.Wait()

	// Calculate metrics
	testDuration := time.Since(startTime)
	requestsPerSecond := float64(totalRequests) / testDuration.Seconds()
	successRate := float64(successRequests) / float64(totalRequests) * 100
	errorRate := float64(failedRequests) / float64(totalRequests) * 100

	// Output results
	fmt.Printf("\n=== LOAD TEST RESULTS ===\n")
	fmt.Printf("Test Duration: %v\n", testDuration)
	fmt.Printf("Total Requests: %d\n", totalRequests)
	fmt.Printf("Successful Requests: %d\n", successRequests)
	fmt.Printf("Failed Requests: %d\n", failedRequests)
	fmt.Printf("Requests Per Second: %.2f\n", requestsPerSecond)
	fmt.Printf("Success Rate: %.2f%%\n", successRate)
	fmt.Printf("Error Rate: %.2f%%\n", errorRate)

	// Simple performance assessment
	if requestsPerSecond > 50 {
		fmt.Printf("✅ EXCELLENT: High throughput (>50 req/s)\n")
	} else if requestsPerSecond > 20 {
		fmt.Printf("✅ GOOD: Good throughput (>20 req/s)\n")
	} else {
		fmt.Printf("⚠️  NEEDS IMPROVEMENT: Low throughput (<20 req/s)\n")
	}

	if successRate > 95 {
		fmt.Printf("✅ EXCELLENT: High reliability (>95%%)\n")
	} else if successRate > 90 {
		fmt.Printf("✅ GOOD: Good reliability (>90%%)\n")
	} else {
		fmt.Printf("⚠️  NEEDS IMPROVEMENT: Low reliability (<90%%)\n")
	}

	if errorRate < 1 {
		fmt.Printf("✅ EXCELLENT: Low error rate (<1%%)\n")
	} else if errorRate < 5 {
		fmt.Printf("✅ GOOD: Acceptable error rate (<5%%)\n")
	} else {
		fmt.Printf("⚠️  NEEDS IMPROVEMENT: High error rate (>5%%)\n")
	}

	fmt.Printf("========================\n")
}
