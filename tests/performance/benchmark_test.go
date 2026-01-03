package performance

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Benchmark model discovery performance
func BenchmarkModelDiscovery(b *testing.B) {
	// Setup mock server with realistic response time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate network latency
		time.Sleep(50 * time.Millisecond)
		
		response := map[string]interface{}{
			"data": generateMockModels(100), // 100 models per provider
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		client := &http.Client{Timeout: 10 * time.Second}
		
		for pb.Next() {
			resp, err := client.Get(fmt.Sprintf("%s/v1/models", server.URL))
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// Benchmark model verification performance
func BenchmarkModelVerification(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate model verification processing time
		time.Sleep(100 * time.Millisecond)
		
		response := map[string]interface{}{
			"success":     true,
			"score":       8.5,
			"modelId":     "test-model",
			"responseTime": 95.5,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		client := &http.Client{Timeout: 10 * time.Second}
		verificationData := map[string]interface{}{
			"modelId": "test-model",
			"prompt":  "Test verification prompt",
		}
		jsonData, _ := json.Marshal(verificationData)
		
		for pb.Next() {
			resp, err := client.Post(fmt.Sprintf("%s/v1/verify", server.URL), 
				"application/json", strings.NewReader(string(jsonData)))
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// Benchmark concurrent operations
func BenchmarkConcurrentOperations(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate processing time
		time.Sleep(25 * time.Millisecond)
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	}))
	defer server.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		client := &http.Client{Timeout: 10 * time.Second}
		
		var wg sync.WaitGroup
		for pb.Next() {
			wg.Add(1)
			go func() {
				defer wg.Done()
				resp, err := client.Get(fmt.Sprintf("%s/test", server.URL))
				if err != nil {
					return
				}
				resp.Body.Close()
			}()
		}
		wg.Wait()
	})
}

// Benchmark configuration loading
func BenchmarkConfigurationLoading(b *testing.B) {
	// Create large configuration
	largeConfig := generateLargeConfiguration(1000) // 1000 models
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Simulate configuration parsing
			var config map[string]interface{}
			json.Unmarshal([]byte(largeConfig), &config)
		}
	})
}

// Benchmark memory usage with large model sets
func BenchmarkMemoryUsage(b *testing.B) {
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		// Create large dataset
		models := make([]map[string]interface{}, 10000)
		for j := 0; j < 10000; j++ {
			models[j] = map[string]interface{}{
				"id":        fmt.Sprintf("model-%d", j),
				"name":      fmt.Sprintf("Model %d", j),
				"provider":  "test-provider",
				"maxTokens": 8192,
				"metadata": map[string]interface{}{
					"supportsBrotli": j%2 == 0,
					"supportsHTTP3":  j%3 == 0,
					"score":          float64(j%10) + 8.0,
				},
			}
		}
		_ = models // Prevent optimization
	}
}

// Benchmark database operations
func BenchmarkDatabaseOperations(b *testing.B) {
	b.Run("Insert", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Simulate database insert
			model := map[string]interface{}{
				"id":       fmt.Sprintf("model-%d", i),
				"name":     fmt.Sprintf("Model %d", i),
				"provider": "test-provider",
			}
			_ = model
		}
	})
	
	b.Run("Query", func(b *testing.B) {
		// Pre-populate data
		models := make([]map[string]interface{}, 1000)
		for i := 0; i < 1000; i++ {
			models[i] = map[string]interface{}{
				"id":   fmt.Sprintf("model-%d", i),
				"name": fmt.Sprintf("Model %d", i),
			}
		}
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// Simulate query
				for _, model := range models {
					if model["id"] == "model-500" {
						break
					}
				}
			}
		})
	})
}

// Test performance under load
func TestPerformanceUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Setup test server
	requestCount := 0
	var mu sync.Mutex
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()
		
		// Simulate processing time
		time.Sleep(10 * time.Millisecond)
		
		response := map[string]interface{}{
			"status": "ok",
			"count":  requestCount,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Test with increasing load
	concurrencyLevels := []int{1, 10, 50, 100}
	
	for _, concurrency := range concurrencyLevels {
		t.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(t *testing.T) {
			start := time.Now()
			requestCount = 0
			
			var wg sync.WaitGroup
			errors := 0
			
			for i := 0; i < concurrency; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					
					client := &http.Client{Timeout: 5 * time.Second}
					resp, err := client.Get(fmt.Sprintf("%s/test", server.URL))
					if err != nil {
						errors++
						return
					}
					defer resp.Body.Close()
					
					if resp.StatusCode != http.StatusOK {
						errors++
					}
				}()
			}
			
			wg.Wait()
			duration := time.Since(start)
			
			assert.Equal(t, 0, errors, "Should have no errors at concurrency level %d", concurrency)
			assert.Equal(t, concurrency, requestCount)
			assert.Less(t, duration, time.Duration(concurrency)*50*time.Millisecond, 
				"Performance degraded at concurrency level %d", concurrency)
			
			t.Logf("Concurrency %d: Completed %d requests in %v", concurrency, requestCount, duration)
		})
	}
}

// Test response time consistency
func TestResponseTimeConsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Consistent processing time
		time.Sleep(50 * time.Millisecond)
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	}))
	defer server.Close()

	responseTimes := make([]time.Duration, 100)
	
	for i := 0; i < 100; i++ {
		start := time.Now()
		resp, err := http.Get(fmt.Sprintf("%s/test", server.URL))
		require.NoError(t, err)
		resp.Body.Close()
		responseTimes[i] = time.Since(start)
	}
	
	// Calculate statistics
	var total time.Duration
	for _, rt := range responseTimes {
		total += rt
	}
	average := total / time.Duration(len(responseTimes))
	
	// Check consistency (standard deviation should be low)
	var variance float64
	for _, rt := range responseTimes {
		diff := float64(rt - average)
		variance += diff * diff
	}
	variance /= float64(len(responseTimes))
	// Take square root to get standard deviation (variance is in ns^2)
	stdDev := time.Duration(math.Sqrt(variance))

	assert.Less(t, stdDev, 20*time.Millisecond, "Response time varies too much")
	assert.Less(t, average, 100*time.Millisecond, "Average response time too high")
	assert.Greater(t, average, 40*time.Millisecond, "Average response time too low")
}

// Benchmark cache performance
func BenchmarkCachePerformance(b *testing.B) {
	cache := make(map[string]interface{})
	
	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("model-%d", i)
		cache[key] = map[string]interface{}{
			"id":   key,
			"name": fmt.Sprintf("Model %d", i),
		}
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Cache read
			if model, exists := cache["model-500"]; exists {
				_ = model
			}
		}
	})
}

// Benchmark JSON serialization/deserialization
func BenchmarkJSONSerialization(b *testing.B) {
	model := map[string]interface{}{
		"id":          "test-model",
		"name":        "Test Model",
		"provider":    "test-provider",
		"maxTokens":   8192,
		"contextWindow": 128000,
		"supportsBrotli": true,
		"supportsHTTP3": true,
		"score":       8.5,
		"metadata": map[string]interface{}{
			"cost": map[string]interface{}{
				"input":  0.01,
				"output": 0.02,
			},
			"latency": map[string]interface{}{
				"average": 100,
				"p95":     150,
				"p99":     200,
			},
		},
	}
	
	b.Run("Marshal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(model)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	
	b.Run("Unmarshal", func(b *testing.B) {
		jsonData, _ := json.Marshal(model)
		b.ReportAllocs()
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			var result map[string]interface{}
			err := json.Unmarshal(jsonData, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Test memory efficiency
func TestMemoryEfficiency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Test with different data sizes
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		t.Run(fmt.Sprintf("Size_%d", size), func(t *testing.T) {
			// Measure memory before
			var m1 runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&m1)
			
			// Create large dataset
			models := make([]map[string]interface{}, size)
			for i := 0; i < size; i++ {
				models[i] = map[string]interface{}{
					"id":       fmt.Sprintf("model-%d", i),
					"name":     fmt.Sprintf("Model %d", i),
					"provider": "test-provider",
					"metadata": generateLargeMetadata(),
				}
			}
			
			// Measure memory after - no GC to avoid reclaiming our test data
			var m2 runtime.MemStats
			runtime.ReadMemStats(&m2)

			// Handle potential underflow (GC may have run between measurements)
			var memoryUsed uint64
			if m2.Alloc >= m1.Alloc {
				memoryUsed = m2.Alloc - m1.Alloc
			} else {
				memoryUsed = 0 // GC reclaimed memory
			}

			// Keep reference to models to prevent GC from collecting them during test
			_ = len(models)

			expectedMemory := uint64(size) * 2048 // Rough estimate: 2KB per model with metadata

			assert.Less(t, memoryUsed, expectedMemory*2, "Memory usage too high for %d models", size)

			perModel := uint64(0)
			if memoryUsed > 0 {
				perModel = memoryUsed / uint64(size)
			}
			t.Logf("Size %d: Memory used: %d bytes, Per model: %d bytes",
				size, memoryUsed, perModel)
		})
	}
}

// Benchmark string operations
func BenchmarkStringOperations(b *testing.B) {
	modelName := "GPT-4 (llmsvd) (brotli) (http3) (SC:8.5)"
	
	b.Run("ParseSuffixes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			parts := strings.Split(modelName, " ")
			_ = parts
		}
	})
	
	b.Run("GenerateDisplayName", func(b *testing.B) {
		suffixes := []string{"llmsvd", "brotli", "http3", "SC:8.5"}
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			displayName := "GPT-4"
			for _, suffix := range suffixes {
				displayName += fmt.Sprintf(" (%s)", suffix)
			}
			_ = displayName
		}
	})
}

// Helper functions
func generateMockModels(count int) []map[string]interface{} {
	models := make([]map[string]interface{}, count)
	for i := 0; i < count; i++ {
		models[i] = map[string]interface{}{
			"id":       fmt.Sprintf("model-%d", i),
			"name":     fmt.Sprintf("Model %d", i),
			"provider": "test-provider",
			"created":  time.Now().Unix(),
		}
	}
	return models
}

func generateLargeConfiguration(modelCount int) string {
	config := map[string]interface{}{
		"$schema": "https://opencode.sh/schema.json",
		"username": "testuser",
		"provider": map[string]interface{}{
			"test-provider": map[string]interface{}{
				"options": map[string]interface{}{
					"apiKey":  "sk-test-key",
					"baseURL": "https://api.test.com/v1",
				},
				"models": generateMockModels(modelCount),
			},
		},
	}
	
	jsonData, _ := json.Marshal(config)
	return string(jsonData)
}

func generateLargeMetadata() map[string]interface{} {
	return map[string]interface{}{
		"supportsBrotli": true,
		"supportsHTTP3":  true,
		"score":          8.5,
		"cost": map[string]interface{}{
			"input":  0.01,
			"output": 0.02,
		},
		"performance": map[string]interface{}{
			"latency": map[string]interface{}{
				"avg":  100,
				"p95":  150,
				"p99":  200,
				"p999": 300,
			},
			"throughput": map[string]interface{}{
				"rps": 1000,
				"tpm": 60000,
			},
		},
		"features": []string{"streaming", "batching", "caching", "retry"},
		"capabilities": map[string]interface{}{
			"maxTokens":       8192,
			"contextWindow":   128000,
			"supportsImages":  true,
			"supportsAudio":   false,
			"supportsVideo":   false,
		},
	}
}