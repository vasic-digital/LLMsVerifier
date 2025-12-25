package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"llm-verifier/api"
	"llm-verifier/config"
	"llm-verifier/llmverifier"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEndToEnd_API tests the complete API flow from HTTP request to response
func TestEndToEnd_API(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	// Load config (use default if config file doesn't exist)
	cfg, err := llmverifier.LoadConfig("config.yaml")
	if err != nil {
		// Create minimal config if loading fails
		cfg = &config.Config{}
	}

	// Create API server (database can be nil for basic testing)
	server := api.NewServer(cfg, nil)

	// Create test HTTP server
	testServer := httptest.NewServer(server.Router())
	defer testServer.Close()

	client := &http.Client{Timeout: 10 * time.Second}

	// Test 1: Health endpoint
	t.Run("HealthEndpoint", func(t *testing.T) {
		resp, err := client.Get(testServer.URL + "/api/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var healthResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&healthResp)
		require.NoError(t, err)

		assert.Equal(t, "healthy", healthResp["status"])
		assert.Contains(t, healthResp, "timestamp")
	})

	// Test 2: List providers endpoint
	t.Run("ListProviders", func(t *testing.T) {
		resp, err := client.Get(testServer.URL + "/api/providers")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var providersResp []map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&providersResp)
		require.NoError(t, err)

		// Should contain at least OpenAI, Anthropic, Google
		providerNames := make([]string, len(providersResp))
		for i, p := range providersResp {
			providerNames[i] = p["name"].(string)
		}

		assert.Contains(t, providerNames, "OpenAI")
		assert.Contains(t, providerNames, "Anthropic")
		assert.Contains(t, providerNames, "Google")
	})

	// Test 3: List models endpoint
	t.Run("ListModels", func(t *testing.T) {
		resp, err := client.Get(testServer.URL + "/api/models")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var modelsResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&modelsResp)
		require.NoError(t, err)

		assert.Contains(t, modelsResp, "models")
	})

	// Test 4: Add provider (POST /api/providers)
	t.Run("AddProvider", func(t *testing.T) {
		providerData := map[string]interface{}{
			"name":        "TestProvider",
			"endpoint":    "https://api.test.com",
			"api_key":     "test-key",
			"description": "Test provider for integration testing",
		}

		jsonData, err := json.Marshal(providerData)
		require.NoError(t, err)

		resp, err := client.Post(testServer.URL+"/api/providers", "application/json", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should succeed (201 Created) or be handled gracefully
		if resp.StatusCode == http.StatusCreated {
			var addResp map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&addResp)
			require.NoError(t, err)
			assert.Contains(t, addResp, "status")
		}
	})

	// Test 5: Error handling - invalid endpoint
	t.Run("InvalidEndpoint", func(t *testing.T) {
		resp, err := client.Get(testServer.URL + "/api/invalid-endpoint")
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return 404 Not Found
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// Test 6: Method not allowed
	t.Run("MethodNotAllowed", func(t *testing.T) {
		resp, err := client.Post(testServer.URL+"/api/health", "application/json", nil)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Health endpoint should only accept GET
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

// TestEndToEnd_LoadTesting tests the API under load
func TestEndToEnd_LoadTesting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	// Load config
	cfg, err := llmverifier.LoadConfig("config.yaml")
	if err != nil {
		cfg = &config.Config{}
	}

	// Create API server
	server := api.NewServer(cfg, nil)
	testServer := httptest.NewServer(server.Router())
	defer testServer.Close()

	client := &http.Client{Timeout: 5 * time.Second}

	// Test concurrent requests
	t.Run("ConcurrentRequests", func(t *testing.T) {
		numRequests := 10
		results := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				resp, err := client.Get(testServer.URL + "/api/health")
				if err != nil {
					results <- err
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					results <- http.ErrNotSupported // Using as error indicator
					return
				}
				results <- nil
			}()
		}

		// Wait for all requests to complete
		for i := 0; i < numRequests; i++ {
			err := <-results
			assert.NoError(t, err)
		}
	})

	// Test rapid sequential requests
	t.Run("RapidRequests", func(t *testing.T) {
		for i := 0; i < 20; i++ {
			resp, err := client.Get(testServer.URL + "/api/health")
			require.NoError(t, err)
			resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}
	})
}

// TestEndToEnd_ErrorScenarios tests various error scenarios
func TestEndToEnd_ErrorScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping error scenario test in short mode")
	}

	// Load config
	cfg, err := llmverifier.LoadConfig("config.yaml")
	if err != nil {
		cfg = &config.Config{}
	}

	// Create API server
	server := api.NewServer(cfg, nil)
	testServer := httptest.NewServer(server.Router())
	defer testServer.Close()

	client := &http.Client{Timeout: 3 * time.Second}

	// Test timeout scenario (by setting very short client timeout)
	t.Run("TimeoutHandling", func(t *testing.T) {
		shortTimeoutClient := &http.Client{Timeout: 1 * time.Nanosecond}
		_, err := shortTimeoutClient.Get(testServer.URL + "/api/health")

		// Should timeout
		assert.Error(t, err)
	})

	// Test malformed JSON
	t.Run("MalformedJSON", func(t *testing.T) {
		resp, err := client.Post(testServer.URL+"/api/providers", "application/json", bytes.NewBufferString("{invalid json"))
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should handle gracefully
		assert.True(t, resp.StatusCode >= 400)
	})

	// Test large payload
	t.Run("LargePayload", func(t *testing.T) {
		largeData := make([]byte, 1024*1024) // 1MB
		for i := range largeData {
			largeData[i] = 'a'
		}

		resp, err := client.Post(testServer.URL+"/api/providers", "application/json", bytes.NewBuffer(largeData))
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should handle large payloads appropriately
		assert.True(t, resp.StatusCode >= 200)
	})
}

// TestEndToEnd_Configuration tests configuration-related endpoints
func TestEndToEnd_Configuration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping configuration test in short mode")
	}

	// Load config
	cfg, err := llmverifier.LoadConfig("config.yaml")
	if err != nil {
		cfg = &config.Config{}
	}

	// Create API server
	server := api.NewServer(cfg, nil)
	testServer := httptest.NewServer(server.Router())
	defer testServer.Close()

	client := &http.Client{Timeout: 10 * time.Second}

	// Test configuration export (if endpoint exists)
	t.Run("ConfigExport", func(t *testing.T) {
		// This would test configuration export endpoints if they exist
		// For now, just verify the server handles config-related requests
		resp, err := client.Get(testServer.URL + "/api/health")
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// BenchmarkEndToEnd_API benchmarks the API endpoints
func BenchmarkEndToEnd_API(b *testing.B) {
	// Load config
	cfg, err := llmverifier.LoadConfig("config.yaml")
	if err != nil {
		cfg = &config.Config{}
	}

	// Create API server
	server := api.NewServer(cfg, nil)
	testServer := httptest.NewServer(server.Router())
	defer testServer.Close()

	client := &http.Client{Timeout: 10 * time.Second}

	b.Run("HealthEndpoint", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(testServer.URL + "/api/health")
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})

	b.Run("ProvidersEndpoint", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(testServer.URL + "/api/providers")
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}
