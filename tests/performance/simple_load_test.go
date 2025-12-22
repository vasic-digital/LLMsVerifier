package performance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

// Simple load test for Enhanced LLM Verifier
func TestSimpleLoad(t *testing.T) {
	// Test configuration
	targetURL := "http://localhost:8080/api/v1/verify"
	concurrentUsers := 10
	duration := 2 * time.Second

	var wg sync.WaitGroup
	totalRequests := 0
	successfulRequests := 0
	errors := make([]error, 0)

	startTime := time.Now()

	fmt.Printf("Starting load test against %s\n", targetURL)
	fmt.Printf("Concurrent users: %d\n", concurrentUsers)
	fmt.Printf("Test duration: %v\n", duration)

	// HTTP Client for testing
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Run load test
	for i := 0; i < concurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			endTime := time.Now().Add(duration)
			requestCount := 0

			for time.Now().Before(endTime) {
				payload := map[string]interface{}{
					"model":       "gpt-4",
					"provider":    "openai",
					"prompt":      fmt.Sprintf("Load test prompt %d-%d", userID, requestCount),
					"max_tokens":  1000,
					"temperature": 0.7,
				}

				jsonData, _ := json.Marshal(payload)

				req, err := http.NewRequest("POST", targetURL, bytes.NewReader(jsonData))
				if err != nil {
					errors = append(errors, fmt.Errorf("user %d: %v", userID, err))
					return
				}

				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer test-token")

				resp, err := client.Do(req)
				if err != nil {
					errors = append(errors, fmt.Errorf("user %d: %v", userID, err))
					return
				}

				resp.Body.Close()

				totalRequests++
				requestCount++

				if resp.StatusCode == 200 {
					successfulRequests++
				}

				// Small delay between requests
				time.Sleep(100 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	// Calculate metrics
	testDuration := time.Since(startTime)
	
	// Output results
	fmt.Printf("\n=== LOAD TEST RESULTS ===\n")
	fmt.Printf("Total Requests: %d\n", totalRequests)
	fmt.Printf("Successful Requests: %d\n", successfulRequests)
	fmt.Printf("Failed Requests: %d\n", totalRequests-successfulRequests)
	
	if totalRequests == 0 {
		fmt.Printf("No requests were made. Cannot calculate metrics.\n")
		fmt.Printf("Test Duration: %v\n", testDuration)
		return
	}
	
	requestsPerSecond := float64(totalRequests) / testDuration.Seconds()
	errorRate := float64(totalRequests-successfulRequests) / float64(totalRequests) * 100
	
	fmt.Printf("Success Rate: %.2f%%\n", float64(successfulRequests)/float64(totalRequests)*100)
	fmt.Printf("Requests Per Second: %.2f\n", requestsPerSecond)
	fmt.Printf("Average Response Time: %v\n", testDuration/time.Duration(totalRequests))
	fmt.Printf("Test Duration: %v\n", testDuration)
	fmt.Printf("Error Rate: %.2f%%\n", errorRate)

	if len(errors) > 0 {
		fmt.Printf("\nErrors:\n")
		for _, err := range errors {
			fmt.Printf("- %v\n", err)
		}
	}

	// Simple performance assessment
	if requestsPerSecond > 100 {
		fmt.Printf("✅ EXCELLENT: High throughput achieved\n")
	} else if requestsPerSecond > 50 {
		fmt.Printf("✅ GOOD: Acceptable throughput\n")
	} else {
		fmt.Printf("⚠️  NEEDS IMPROVEMENT: Low throughput\n")
	}

	if errorRate < 1.0 {
		fmt.Printf("✅ EXCELLENT: Low error rate\n")
	} else if errorRate < 5.0 {
		fmt.Printf("✅ GOOD: Acceptable error rate\n")
	} else {
		fmt.Printf("⚠️  NEEDS IMPROVEMENT: High error rate\n")
	}
}
