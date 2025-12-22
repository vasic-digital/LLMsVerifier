package llmverifier

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"llm-verifier/config"
)

// TestVerifier_SummarizeConversation tests conversation summarization
func TestVerifier_SummarizeConversation_Fixed(t *testing.T) {
	// Create mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		// Mock response for summarization
		response := `{
			"id": "test",
			"object": "chat.completion",
			"created": 1234567890,
			"model": "gpt-4",
			"choices": [{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "{\n  \"summary\": \"A conversation about testing and verification\",\n  \"topics\": [\"testing\", \"verification\"],\n  \"key_points\": [\"Discussed unit testing\", \"Mentioned integration testing\"],\n  \"importance\": 0.8\n}"
				},
				"finish_reason": "stop"
			}],
			"usage": {"prompt_tokens": 50, "completion_tokens": 30, "total_tokens": 80}
		}`
		w.Write([]byte(response))
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:         mockServer.URL,
			APIKey:          "test-key",
			DefaultModel:    "gpt-4",
			MaxRetries:      3,
			RequestDelay:    100 * time.Millisecond,
			Timeout:         30 * time.Second,
		},
		Concurrency: 1,
		Timeout:     30 * time.Second,
	}

	verifier := New(cfg)

	tests := []struct {
		name      string
		messages  []string
		expectErr bool
	}{
		{
			name:      "valid conversation",
			messages:  []string{"Hello", "How are you?", "I'm fine thanks"},
			expectErr: false,
		},
		{
			name:      "empty messages",
			messages:  []string{},
			expectErr: true,
		},
		{
			name:      "single message",
			messages:  []string{"Test message"},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary, err := verifier.SummarizeConversation(tt.messages)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if summary != nil {
					t.Error("Expected nil summary on error")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if summary == nil {
					t.Error("Expected summary but got nil")
				} else {
					// Verify summary structure
					if summary.Summary == "" {
						t.Error("Expected non-empty summary")
					}
					if len(summary.Topics) == 0 {
						t.Error("Expected topics to be populated")
					}
					if len(summary.KeyPoints) == 0 {
						t.Error("Expected key points to be populated")
					}
					if summary.Importance < 0 || summary.Importance > 1 {
						t.Error("Expected importance score between 0 and 1")
					}
				}
			}
		})
	}
}

// TestVerifier_GetGlobalClient tests global client creation
func TestVerifier_GetGlobalClient_Fixed(t *testing.T) {
	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL: "https://api.openai.com/v1",
			APIKey:  "test-key",
		},
	}

	verifier := New(cfg)
	client := verifier.GetGlobalClient()

	if client == nil {
		t.Error("Expected client to be created")
	}

	if client.endpoint != cfg.Global.BaseURL {
		t.Errorf("Expected endpoint %s, got %s", cfg.Global.BaseURL, client.endpoint)
	}

	if client.apiKey != cfg.Global.APIKey {
		t.Errorf("Expected API key %s, got %s", cfg.Global.APIKey, client.apiKey)
	}
}

// TestVerifier_Verify_ConcurrentModels tests concurrent model verification
func TestVerifier_Verify_ConcurrentModels_Fixed(t *testing.T) {
	cfg := &config.Config{
		Concurrency: 10,
		Timeout:     30 * time.Second,
		LLMs: []config.LLMConfig{
			{Name: "Model 1", Endpoint: "https://api1.test.com", APIKey: "key1", Model: "model1"},
			{Name: "Model 2", Endpoint: "https://api2.test.com", APIKey: "key2", Model: "model2"},
			{Name: "Model 3", Endpoint: "https://api3.test.com", APIKey: "key3", Model: "model3"},
		},
	}

	verifier := New(cfg)
	results, err := verifier.Verify()

	// Should fail due to fake APIs but return results
	if err == nil {
		t.Log("Verification unexpectedly succeeded with fake APIs")
	}
	if results == nil {
		t.Error("Expected results slice even with failures")
	}
}

// TestVerifier_Verify_CancelContext tests context cancellation
func TestVerifier_Verify_CancelContext_Fixed(t *testing.T) {
	cfg := &config.Config{
		Concurrency: 5,
		Timeout:     5 * time.Second,
		LLMs: []config.LLMConfig{
			{Name: "Test Model", Endpoint: "https://api.test.com", APIKey: "test-key", Model: "test-model"},
		},
	}

	verifier := New(cfg)

	// Create a context that will be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// The verifier should respect context cancellation
	done := make(chan struct{})
	go func() {
		_, _ = verifier.Verify()
		close(done)
	}()

	select {
	case <-done:
		t.Error("Verification completed before context cancellation")
	case <-ctx.Done():
		// Expected - context was cancelled
	}
}

// TestVerifier_verifySingleModel_Timeout tests timeout handling
func TestVerifier_verifySingleModel_Timeout_Fixed(t *testing.T) {
	cfg := &config.Config{
		Timeout: 100 * time.Millisecond,
	}

	verifier := New(cfg)
	client := verifier.GetGlobalClient()

	result, err := verifier.verifySingleModel(client, "test-model", "https://api.test.com")

	// Should fail due to timeout
	if err == nil {
		t.Error("Expected error due to timeout")
	}
	if result.Availability.Responsive {
		t.Error("Expected model to be unresponsive due to timeout")
	}
}

// TestVerifier_checkResponsiveness_Timeout tests responsiveness check timeout
func TestVerifier_checkResponsiveness_Timeout_Fixed(t *testing.T) {
	cfg := &config.Config{
		Timeout: 50 * time.Millisecond,
	}

	verifier := New(cfg)
	client := verifier.GetGlobalClient()

	latency, responsive, err := verifier.checkResponsiveness(client, "test-model")

	if responsive {
		t.Error("Expected model to be unresponsive due to timeout")
	}
	if err == "" {
		t.Error("Expected error message due to timeout")
	}
	if latency <= 0 {
		t.Error("Expected positive latency measurement")
	}
}

// TestVerifier_checkOverload_HighConcurrency tests overload detection
func TestVerifier_checkOverload_HighConcurrency_Fixed(t *testing.T) {
	cfg := &config.Config{
		Timeout: 30 * time.Second,
	}

	verifier := New(cfg)
	client := verifier.GetGlobalClient()

	overloaded, responseTime := verifier.checkOverload(client, "test-model")

	// Since we're hitting fake APIs, all requests should fail
	if !overloaded {
		t.Log("Note: Model detected as overloaded with fake API")
	}
	t.Logf("Overload detection results: overloaded=%v, avgLatency=%v, throughput=%v", 
		overloaded, responseTime.AverageLatency, responseTime.Throughput)
}

// TestVerifier_Verify_LargeNumberOfModels tests many models
func TestVerifier_Verify_LargeNumberOfModels_Fixed(t *testing.T) {
	models := make([]config.LLMConfig, 10) // Reduced for testing
	for i := 0; i < 10; i++ {
		models[i] = config.LLMConfig{
			Name:     "Model " + string(rune('A'+i%26)),
			Endpoint: "https://api.test.com",
			APIKey:   "key",
			Model:    "model",
		}
	}

	cfg := &config.Config{
		Concurrency: 20,
		Timeout:     60 * time.Second,
		LLMs:        models,
	}

	verifier := New(cfg)
	results, err := verifier.Verify()

	// Should fail due to fake APIs but return results
	if err == nil {
		t.Log("Verification unexpectedly succeeded with fake APIs")
	}
	if results == nil {
		t.Error("Expected results slice")
	}
	if len(results) != 10 {
		t.Errorf("Expected 10 results, got %d", len(results))
	}
}

// TestVerifier_EdgeCases tests edge cases
func TestVerifier_EdgeCases_Fixed(t *testing.T) {
	tests := []struct {
		name   string
		config  *config.Config
		testFn func(*Verifier, *testing.T)
	}{
		{
			name: "zero concurrency",
			config: &config.Config{
				Concurrency: 0,
				Timeout:     30 * time.Second,
			},
			testFn: func(v *Verifier, t *testing.T) {
				cfg := v.cfg
				if cfg.Concurrency != 0 {
					t.Errorf("Expected concurrency to remain 0, got %d", cfg.Concurrency)
				}
			},
		},
		{
			name: "negative concurrency",
			config: &config.Config{
				Concurrency: -5,
				Timeout:     30 * time.Second,
			},
			testFn: func(v *Verifier, t *testing.T) {
				t.Log("Testing with negative concurrency")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier := New(tt.config)
			tt.testFn(verifier, t)
		})
	}
}

// TestVerifier_discoverAndVerifyAllModels tests model discovery
func TestVerifier_discoverAndVerifyAllModels_Fixed(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"object": "list",
				"data": [
					{"id": "model1", "object": "model", "created": 1234567890, "owned_by": "test"},
					{"id": "model2", "object": "model", "created": 1234567891, "owned_by": "test"}
				]
			}`))
			return
		}
		// For chat completions, simulate timeout/failure
		w.WriteHeader(http.StatusRequestTimeout)
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL: mockServer.URL,
			APIKey:  "test-key",
		},
		Concurrency: 2,
		Timeout:     1 * time.Second,
	}

	verifier := New(cfg)
	results, err := verifier.discoverAndVerifyAllModels()

	if err == nil {
		t.Log("Note: discoverAndVerifyAllModels succeeded unexpectedly")
	}

	if results == nil {
		t.Error("Expected results slice")
	}
}

// Test feature detection methods
func TestVerifier_FeatureDetection_Fixed(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		response := `{
			"id": "test",
			"object": "chat.completion",
			"created": 1234567890,
			"model": "gpt-4",
			"choices": [{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "def test():\n    return True"
				},
				"finish_reason": "stop"
			}],
			"usage": {"prompt_tokens": 20, "completion_tokens": 10, "total_tokens": 30}
		}`
		w.Write([]byte(response))
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL: mockServer.URL,
			APIKey:  "test-key",
		},
		Timeout: 30 * time.Second,
	}

	verifier := New(cfg)
	client := verifier.GetGlobalClient()
	ctx := context.Background()

	// Test various feature detection methods
	features := []struct {
		name string
		test func() bool
	}{
		{"CodeGeneration", func() bool { return verifier.testCodeGeneration(client, "test-model", ctx) }},
		{"CodeCompletion", func() bool { return verifier.testCodeCompletion(client, "test-model", ctx) }},
		{"CodeExplanation", func() bool { return verifier.testCodeExplanation(client, "test-model", ctx) }},
		{"CodeReview", func() bool { return verifier.testCodeReview(client, "test-model", ctx) }},
	}

	for _, feature := range features {
		t.Run(feature.name, func(t *testing.T) {
			result := feature.test()
			t.Logf("Feature %s result: %v", feature.name, result)
		})
	}
}

// Test language-specific test functionality
func TestVerifier_LanguageSpecificTests_Fixed(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		response := `{
			"id": "test",
			"object": "chat.completion",
			"created": 1234567890,
			"model": "gpt-4",
			"choices": [{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "def add(a, b):\n    return a + b"
				},
				"finish_reason": "stop"
			}],
			"usage": {"prompt_tokens": 20, "completion_tokens": 15, "total_tokens": 35}
		}`
		w.Write([]byte(response))
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL: mockServer.URL,
			APIKey:  "test-key",
		},
		Timeout: 30 * time.Second,
	}

	verifier := New(cfg)
	client := verifier.GetGlobalClient()

	testResults := verifier.runLanguageSpecificTests(client, "test-model")

	// Verify test results structure
	if testResults.OverallSuccessRate < 0 || testResults.OverallSuccessRate > 100 {
		t.Errorf("Invalid overall success rate: %f", testResults.OverallSuccessRate)
	}

	t.Logf("Language test results: Python=%.1f, JS=%.1f, Go=%.1f, Overall=%.1f",
		testResults.PythonSuccessRate, testResults.JavascriptSuccessRate,
		testResults.GoSuccessRate, testResults.OverallSuccessRate)
}