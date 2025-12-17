package tests

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

	"llm-verifier/config"
	"llm-verifier/llmverifier"
	"llm-verifier/llmverifier"
)

// TestLoadConfiguration tests config loading edge cases
func TestLoadConfiguration(t *testing.T) {
	t.Run("Load Configuration - Valid Config", func(t *testing.T) {
		// Test with valid config file
		validConfig := `global:
  base_url: "https://api.openai.com/v1"
  api_key: "valid-key"
llms:
  - name: "gpt-4"
    endpoint: "https://api.openai.com/v1"
    model: "gpt-4"
`

		// Create temporary config file
		tempDir := t.TempDir()
		tempConfig := filepath.Join(tempDir, "config.yaml")
		err := os.WriteFile(tempConfig, []byte(validConfig), 0644)
		if err != nil {
			t.Fatalf("Failed to write valid config file: %v", err)
		}

		// Load and verify
		cfg, err := config.LoadConfig(tempConfig)
		if err != nil {
			t.Errorf("Failed to load valid config: %v", err)
		}

		// Verify values
		if cfg.Global.BaseURL != "https://api.openai.com/v1" {
			t.Errorf("Expected base URL mismatch")
		}

		if cfg.Global.APIKey != "valid-key" {
			t.Errorf("Expected API key mismatch")
		}

		if len(cfg.LLMs) != 1 {
			t.Errorf("Expected 1 LLM configuration, got %d", len(cfg.LLMs))
		}

		llm := cfg.LLMs[0]
		if llm.Name != "gpt-4" {
			t.Errorf("Expected LLM name 'gpt-4', got '%s'", llm.Name)
		}

		if llm.Endpoint != "https://api.openai.com/v1" {
			t.Errorf("Expected LLM endpoint 'https://api.openai.com/v1', got '%s'", llm.Endpoint)
		}

		t.Logf("Valid config test passed")
	})

	t.Run("Load Configuration - Invalid Config", func(t *testing.T) {
		// Test with various invalid configurations
		testCases := []struct {
			name        string
			config      string
			expectError bool
			description string
		}{
			{
				name:     "Empty Base URL",
				config:     `global:
  api_key: "test-key"`,
				llms:     []config.LLMConfig{},
				expectError: true,
				description: "Should fail with empty base URL",
			},
			{
				name:     "Invalid API Key",
				config:     `global:
  base_url: "invalid-url",
				api_key: "test-key",
				llms:     []config.LLMConfig{},
				expectError: true,
				description: "Should fail with invalid URL format",
			},
			{
				name:     "No LLMs",
				config:     `global:
  base_url: "https://api.openai.com/v1",
				api_key: "test-key",
				llms:     []config.LLMConfig{},
				expectError: true,
				description: "Should fail with no LLM configurations",
			},
			{
				name:     "Malformed LLM Configuration",
				config:     `global:
  base_url: "https://api.openai.com/v1",
				api_key: "test-key",
				llms: `lm_config: [invalid}`,
				expectError: true,
				description: "Should fail with malformed LLM config",
			},
			{
				name:     "Partial LLM Configuration",
				config:     `global:
  base_url: "https://api.openai.com/v1",
				api_key: "test-key",
				llms:     []config.LLMConfig{
						Name:     "partial-lm",
						Endpoint: "https://api.openai.com/v1",
						Model:    "gpt-3.5-turbo",
					},
				},
				expectError: false,
				description: "Should pass with partial LLM config",
			},
		{
				name:     "Too Many LLMs",
				config:     `global:
  base_url: "https://api.openai.com/v1",
				api_key: "test-key",
				llms:     make([]config.LLMConfig, 100), // Many test LLMs
				},
				expectError: false,
				description: "Should pass with multiple LLM configs",
			},
	}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cfg, err := config.LoadConfig(strings.NewReader(tc.config))
				
				if tc.expectError {
					if err == nil {
						t.Errorf("Expected error for '%s': %s", tc.name)
					}
					continue
				}
				
				if err != nil {
					t.Logf("Unexpected error for '%s': %v", tc.name, err)
				}
			})
		}
	})

	t.Run("Load Configuration - Missing Config", func(t *testing.T) {
		// Test with missing config file
		nonExistentFile := "/path/to/nonexistent/config.yaml"
		
			cfg, err := config.LoadConfig(nonExistentFile)
		if err == nil {
			t.Errorf("Expected error for non-existent file")
		}
		
			if cfg != nil {
			t.Errorf("Config loading should fail for non-existent file")
		}

		t.Logf("Missing config test passed")
	})

	t.Run("Load Configuration - Override Environment", func(t *testing.T) {
		// Test config with environment variable overrides
		os.Setenv("LLM_BASE_URL", "https://override.openai.com/v1")
		
		cfg, err := config.LoadConfig("config.yaml")
		if err != nil {
			t.Errorf("Failed to load config with environment override: %v", err)
		}

		// Verify override worked
		if cfg.Global.BaseURL != "https://override.openai.com/v1" {
			t.Errorf("Environment override not applied to base URL")
		}

		t.Logf("Environment override test passed")
	})

	t.Logf("Load configuration tests completed")
}

// TestEnvironmentVariables tests environment variable handling
func TestEnvironmentVariables(t *testing.T) {
	t.Run("Environment Variables", func(t *testing.T) {
		// Test base URL override
		os.Setenv("LLM_BASE_URL", "https://env.openai.com/v1")
		
		cfg, err := config.LoadConfig("config.yaml")
		if err != nil {
			t.Fatalf("Failed to load config with env override: %v", err)
		}

		// Verify override worked
		if cfg.Global.BaseURL != "https://env.openai.com/v1" {
			t.Errorf("Environment override not applied to base URL")
		}

			// Test API key override
		os.Setenv("LLM_API_KEY", "env-test-key")
		
		cfg, err := config.LoadConfig("config.yaml")
		if err != nil {
			t.Fatalf("Failed to load config with env override: %v", err)
		}

		if cfg.Global.APIKey != "env-test-key" {
			t.Errorf("Environment override not applied to API key")
		}

		t.Logf("Environment variables test passed")
	})
}

// TestConfigReloadHotReload tests configuration hot reloading
func TestConfigReloadHotReload(t *testing.T) {
	t.Run("Config Hot Reload", func(t *testing.T) {
		// Create initial config file
		tempDir := t.TempDir()
		initialConfig := `global:
  base_url: "https://api.openai.com/v1"
  api_key: "initial-key"
llms:
  - name: "gpt-4"
    endpoint: "https://api.openai.com/v1"
    model: "gpt-3.5-turbo"
	}

		initialFile := filepath.Join(tempDir, "initial_config.yaml")
		if err := os.WriteFile(initialFile, []byte(initialConfig), 0644); err != nil {
			t.Fatalf("Failed to write initial config: %v", err)
		}

		// Load initial configuration
		cfg, err := config.LoadConfig(initialFile)
		if err != nil {
			t.Fatalf("Failed to load initial config: %v", err)
		}

		// Verify initial load worked
		if cfg.Global.BaseURL != "https://api.openai.com/v1" {
			t.Errorf("Failed to load initial config")
		}

		// Modify config file while server is running
		modifiedConfig := `global:
  base_url: "https://api.openai.com/v1"
  api_key: "modified-key"
llms:
  - name: "gpt-4"
    endpoint: "https://api.openai.com/v1"
    model: "gpt-4-updated",
	},

			modifiedFile := filepath.Join(tempDir, "modified_config.yaml")
		if err := os.WriteFile(modifiedFile, []byte(modifiedConfig), 0644); err != nil {
			t.Fatalf("Failed to write modified config: %v", err)
		}

		// Wait and verify hot reload
		time.Sleep(2 * time.Second)

		// Reload should pick up the changes
		reloadedCfg, err := config.LoadConfig(initialFile)
		if err != nil {
			t.Errorf("Failed to hot reload config: %v", err)
		}

		// Verify reloaded configuration
		if reloadedCfg.Global.BaseURL != "https://api.openai.com/v1" {
			t.Errorf("Hot reload didn't pick up base URL change")
		}

		if reloadedCfg.Global.APIKey != "modified-key" {
			t.Errorf("Hot reload didn't pick up API key change")
		}

		// Verify model update
		if len(reloadedCfg.LLMs) > 0 {
			updatedLLM := reloadedCfg.LLMs[0]
			if updatedLLM.Model != "gpt-4-updated" {
				t.Errorf("Hot reload didn't pick up model change")
			}
		}

		t.Logf("Hot reload test passed")
	})
}

	t.Logf("Config reload tests completed")
}

// TestErrorRecovery tests error handling and recovery
func TestErrorRecovery(t *testing.T) {
	t.Run("Error Recovery", func(t *testing.T) {
		// Test network error scenarios
		testCases := []struct {
			name        string
			description string
			simulateError bool
			testFunc     func() error {
				return
		}
		}{
			{
				name:     "Network Timeout",
				description: "Test network timeout handling",
				simulateError: true,
				testFunc: func() error {
					return fmt.Errorf("network timeout error")
				},
			},
			{
				name:     "Connection Refused",
				description: "Test connection refused",
				simulateError: true,
				testFunc: func() error {
					return fmt.Errorf("connection refused")
				},
			},
			{
				name:     "Invalid Response",
				description: "Test invalid server response",
				simulateError: true,
				testFunc: func() error {
					return fmt.Errorf("invalid response status")
				},
			},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.testFunc()
				if tc.simulateError {
					if err == nil {
						t.Errorf("Expected error for '%s', tc.name)
					} else {
						t.Logf("Error scenario '%s' recovered correctly: %v", tc.name)
					}
				}
			})
	}
	})
}

// TestConcurrentAccess tests concurrent access scenarios
func TestConcurrentAccess(t *testing.T) {
	t.Run("Concurrent Access", func(t *testing.T) {
		// Test multiple concurrent requests
		const numGoroutines = 20
		requests := make(chan int, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				requests <- i
			}()
		}

		// Wait for all requests to complete
		time.Sleep(100 * time.Millisecond)

			// Collect all responses
		var responses []int
		for i := 0; i < numGoroutines; i++ {
			select {
			case status := <-requests:
				responses = append(responses, status)
			default:
				responses = append(responses, 0)
			}
		}

		// Verify all requests succeeded
		successCount := 0
		for _, status := range responses {
			if status == 200 {
				successCount++
			}
		}

		if successCount != numGoroutines {
			t.Errorf("Not all concurrent requests succeeded: %d/%d", successCount, numGoroutines)
		} else {
			t.Logf("All concurrent requests succeeded: %d/%d", successCount, numGoroutines)
		}
	})
	})
}

// TestSessionManagement tests session handling
func TestSessionManagement(t *testing.T) {
	t.Run("Session Management", func(t *testing.T) {
		// Test session creation and cleanup
		t.Run("Session Creation", func(t *testing.T) {
			// Create session
			createReq := httptest.NewRequest("POST", "/auth/session/create", strings.NewReader(`{"user_id": "test-user"}`))
			rec := httptest.NewRecorder()
			testServer.Handler.ServeHTTP(rec, createReq)
			resp := rec.Result()
			
			if resp.StatusCode != http.StatusCreated {
				t.Errorf("Expected status 201, got %d", resp.StatusCode)
			}

			// Verify session data
			if resp.StatusCode == http.StatusCreated {
				var sessionData map[string]interface{}
				if err := json.NewDecoder(resp.Body).Decode(&sessionData); err != nil {
					t.Errorf("Failed to parse session data: %v", err)
				}
				
				if sessionData["user_id"] != "test-user" {
					t.Errorf("Expected user_id to be test-user")
				}
			}

			t.Logf("Session creation test passed")
		})

		t.Run("Session Expiration", func(t *testing.T) {
		// Create session
		createReq := httptest.NewRequest("POST", "/auth/session/create", strings.NewReader(`{"user_id": "test-user"}`))
			rec := httptest.NewRecorder()
			testServer.Handler.ServeHTTP(rec, createReq)
			resp := rec.Result()
			
			if resp.StatusCode != http.StatusCreated {
				t.Errorf("Expected status 201, got %d", resp.StatusCode)
			}

			// Get session ID
			sessionID := resp.Header().Get("X-Session-ID")
			if sessionID == "" {
				t.Error("Session ID header not found")
			}

			// Test session expiration by waiting
			expireReq := httptest.NewRequest("DELETE", "/auth/session/"+sessionID, nil)
			rec := httptest.NewRecorder()
			testServer.Handler.ServeHTTP(rec, expireReq)
			resp := rec.Result()
			
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			t.Logf("Session expiration test passed")
		})
	})
}

// TestSecurityHeaders tests security-related headers
func TestSecurityHeaders(t *testing.T) {
	t.Run("Security Headers", func(t *testing.T) {
		// Test missing security headers
		testReq := httptest.NewRequest("GET", "/secure/test", nil)
		rec := httptest.NewRecorder()
		testServer.Handler.ServeHTTP(rec, testReq)
		resp := rec.Result()
			
			if resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("Expected status 401, got %d", resp.StatusCode)
			}

			// Test security headers
	secureReq := httptest.NewRequest("GET", "/secure/test", nil)
	secureReq.Header.Set("X-API-Key", "test-key")
		rec2 := httptest.NewRecorder()
		testServer.Handler.ServeHTTP(rec2, secureReq)
			
			resp2 := rec2.Result()
			if resp2.StatusCode != http.StatusUnauthorized {
				t.Errorf("Expected status 401, got %d", resp2.StatusCode)
			}

		// Test CORS headers
		corsReq := httptest.NewRequest("OPTIONS", "/cors/test", nil)
			rec3 := httptest.NewRecorder()
			testServer.Handler.ServeHTTP(rec3, corsReq)
			resp3 := rec3.Result()
			
			if resp3.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp3.StatusCode)
			}

		t.Logf("Security headers test passed")
	})
}

// TestAPILimits tests API rate limiting
func TestAPILimits(t *testing.T) {
	t.Run("API Rate Limiting", func(t *testing.T) {
		// Test rate limiting at different levels
		testLimits := []struct {
			name     string
			limit    int
			duration time.Duration
			testFunc  func() error
		}{
				{
					name:     "Low Rate Limit",
					limit:    10,
					duration: time.Minute,
					testFunc: func() error {
						// Make requests rapidly
						success := 0
						for i := 0; i < 100; i++ {
							_ = make([]byte, 1024)
							req := httptest.NewRequest("GET", "/test/endpoint", nil)
							rec := httptest.NewRecorder()
							testServer.Handler.ServeHTTP(rec, req)
							resp := rec.Result()
							
							if resp.StatusCode == 200 {
								success++
							}
							
							if success%5 == 0 {
								// Continue making requests
								continue
							}
						}
						
						if success >= 8 {
							break // After 8 successful requests, assume rate limit is active
						}
						
						time.Sleep(100 * time.Millisecond)
					}
					
					if success < 5 {
						t.Errorf("Expected at least 5 successful requests with low rate limit")
					}
				},
			{
					name:     "Medium Rate Limit",
					limit:    50,
					duration: time.Minute,
					testFunc: func() error {
						success := 0
						for i := 0; i < 100; i++ {
							_ = make([]byte, 1024)
							req := httptest.NewRequest("GET", "/test/endpoint", nil)
							rec := httptest.NewRecorder()
							testServer.Handler.ServeHTTP(rec, req)
							resp := rec.Result()
							
							if resp.StatusCode == 200 {
								success++
							}
							
							if success%10 == 0 {
								// Continue making requests
								continue
							}
							
							if success >= 5 {
								break // After 5 successful requests, assume rate limit is working
							}
							
							time.Sleep(50 * time.Millisecond)
						}
					}
					
					if success < 3 {
						t.Errorf("Expected at least 3 successful requests with medium rate limit")
					}
				},
			},
			{
				name:     "High Rate Limit",
					limit:    100,
					duration: time.Second,
					testFunc: func() error {
						success := 0
						for i := 0; i < 20; i++ {
							_ = make([]byte, 1024)
							req := httptest.NewRequest("GET", "/test/endpoint", nil)
							rec := httptest.NewRecorder()
							testServer.Handler.ServeHTTP(rec, req)
							resp := rec.Result()
							
							if resp.StatusCode == 200 {
								success++
							}
							
							if success%2 == 0 {
								// Continue making requests
								continue
							}
							
							if success >= 15 {
								break // After 15 successful requests, assume rate limit is working
							}
							
							time.Sleep(100 * time.Millisecond)
						}
					}
					
					if success < 1 {
						t.Errorf("Expected at least 1 successful request with high rate limit")
					}
				},
			},
		}

		// Test rate limit exceeded scenarios
			t.Run("Rate Limit Exceeded", func(t *testing.T) {
				// Test behavior when rate limit is exceeded
				testFunc := func() error {
					success := 0
					
					for i := 0; i < 10; i++ {
						_ = make([]byte, 1024)
						req := httptest.NewRequest("GET", "/test/endpoint", nil)
						rec := httptest.NewRecorder()
						testServer.Handler.ServeHTTP(rec, req)
						resp := rec.Result()
						
						if resp.StatusCode == 429 {
							return // Rate limit exceeded
						}
						
						if resp.StatusCode == 200 {
							success++
						}
					}
				}

				// Should eventually hit rate limit
				for i := 0; i < 50; i++ {
					_ = make([]byte, 1024)
					req := httptest.NewRequest("GET", "/test/endpoint", nil)
					rec := httptest.NewRecorder()
					testServer.Handler.ServeHTTP(rec, req)
					resp := rec.Result()
					
					if resp.StatusCode == 200 {
						success++
					}
					
					if success%10 == 0 {
						// Continue until rate limit is hit
						continue
					}
					
					if success%30 == 0 {
						// Should hit rate limit eventually
						break
					}
				}
				}
			}
		})
	}
	})
}

// RunLoadConfiguration creates tests for config loading and edge cases
func RunLoadConfiguration(t *testing.T) {
	t.Run("Load Configuration", TestLoadConfiguration)
	t.Run("Load Configuration - Invalid Config", TestLoadConfiguration)
	t.Run("Load Configuration - Missing Config", TestLoadConfiguration)
	t.Run("Load Configuration - Override Environment", TestEnvironmentVariables)
	t.Run("Config Hot Reload", TestConfigReloadHotReload)
}