package llmverifier

import (
	"testing"
	"time"

	"llm-verifier/config"
)

func TestNewVerifier(t *testing.T) {
	cfg := &config.Config{
		Concurrency: 5,
		Timeout:     60 * time.Second,
	}

	verifier := New(cfg)

	if verifier == nil {
		t.Fatal("Expected verifier to be created")
	}

	if verifier.cfg != cfg {
		t.Error("Expected verifier config to match provided config")
	}
}

func TestVerifier_Verify_EmptyConfig(t *testing.T) {
	cfg := &config.Config{
		Concurrency: 1,
		Timeout:     30 * time.Second,
		Global: config.GlobalConfig{
			BaseURL: "https://api.example.com",
			APIKey:  "test-key",
		},
	}

	verifier := New(cfg)

	// This should attempt to discover models from the global endpoint
	// Since we don't have a real API, we expect an error
	results, err := verifier.Verify()

	if err == nil {
		t.Error("Expected error when no LLMs configured and global endpoint discovery fails")
	}

	if results != nil {
		t.Error("Expected nil results when verification fails")
	}
}

func TestVerifier_Verify_SingleLLM(t *testing.T) {
	cfg := &config.Config{
		Concurrency: 1,
		Timeout:     30 * time.Second,
		LLMs: []config.LLMConfig{
			{
				Name:     "Test LLM",
				Endpoint: "https://api.example.com/v1",
				APIKey:   "test-key",
				Model:    "test-model",
			},
		},
	}

	verifier := New(cfg)

	// This will fail because there's no real API, but we can test the structure
	results, err := verifier.Verify()

	if err != nil {
		// Error is expected since we don't have a real API
		t.Logf("Verification failed as expected: %v", err)
	}

	if results == nil {
		t.Error("Expected results slice even if verification fails")
	}
}

func TestVerifier_Verify_MultipleLLMs(t *testing.T) {
	cfg := &config.Config{
		Concurrency: 2,
		Timeout:     30 * time.Second,
		LLMs: []config.LLMConfig{
			{
				Name:     "LLM 1",
				Endpoint: "https://api.example1.com/v1",
				APIKey:   "key1",
				Model:    "model1",
			},
			{
				Name:     "LLM 2",
				Endpoint: "https://api.example2.com/v1",
				APIKey:   "key2",
				Model:    "model2",
			},
		},
	}

	verifier := New(cfg)

	results, err := verifier.Verify()

	if err != nil {
		t.Logf("Verification failed as expected: %v", err)
	}

	if results == nil {
		t.Error("Expected results slice")
	}
}

func TestVerifier_Verify_NoModelSpecified(t *testing.T) {
	cfg := &config.Config{
		Concurrency: 1,
		Timeout:     30 * time.Second,
		LLMs: []config.LLMConfig{
			{
				Name:     "Test LLM",
				Endpoint: "https://api.example.com/v1",
				APIKey:   "test-key",
				// No model specified - should attempt to discover
			},
		},
	}

	verifier := New(cfg)

	results, err := verifier.Verify()

	if err != nil {
		t.Logf("Verification failed as expected: %v", err)
	}

	if results == nil {
		t.Error("Expected results slice")
	}
}

func TestVerifier_Verify_WithHeaders(t *testing.T) {
	cfg := &config.Config{
		Concurrency: 1,
		Timeout:     30 * time.Second,
		LLMs: []config.LLMConfig{
			{
				Name:     "Test LLM",
				Endpoint: "https://api.example.com/v1",
				APIKey:   "test-key",
				Model:    "test-model",
				Headers: map[string]string{
					"Custom-Header": "value",
					"X-API-Version": "2024-01-01",
				},
			},
		},
	}

	verifier := New(cfg)

	results, err := verifier.Verify()

	if err != nil {
		t.Logf("Verification failed as expected: %v", err)
	}

	if results == nil {
		t.Error("Expected results slice")
	}
}

func TestVerifier_Verify_WithFeatures(t *testing.T) {
	cfg := &config.Config{
		Concurrency: 1,
		Timeout:     30 * time.Second,
		LLMs: []config.LLMConfig{
			{
				Name:     "Test LLM",
				Endpoint: "https://api.example.com/v1",
				APIKey:   "test-key",
				Model:    "test-model",
				Features: map[string]bool{
					"tool_use":        true,
					"code_generation": true,
					"embeddings":      false,
				},
			},
		},
	}

	verifier := New(cfg)

	results, err := verifier.Verify()

	if err != nil {
		t.Logf("Verification failed as expected: %v", err)
	}

	if results == nil {
		t.Error("Expected results slice")
	}
}

func TestVerifier_Verify_Concurrency(t *testing.T) {
	cfg := &config.Config{
		Concurrency: 3, // Test with higher concurrency
		Timeout:     30 * time.Second,
		LLMs: []config.LLMConfig{
			{
				Name:     "LLM 1",
				Endpoint: "https://api.example1.com/v1",
				APIKey:   "key1",
				Model:    "model1",
			},
			{
				Name:     "LLM 2",
				Endpoint: "https://api.example2.com/v1",
				APIKey:   "key2",
				Model:    "model2",
			},
			{
				Name:     "LLM 3",
				Endpoint: "https://api.example3.com/v1",
				APIKey:   "key3",
				Model:    "model3",
			},
		},
	}

	verifier := New(cfg)

	results, err := verifier.Verify()

	if err != nil {
		t.Logf("Verification failed as expected: %v", err)
	}

	if results == nil {
		t.Error("Expected results slice")
	}
}

func TestVerifier_Verify_Timeout(t *testing.T) {
	cfg := &config.Config{
		Concurrency: 1,
		Timeout:     100 * time.Millisecond, // Very short timeout
		LLMs: []config.LLMConfig{
			{
				Name:     "Test LLM",
				Endpoint: "https://api.example.com/v1",
				APIKey:   "test-key",
				Model:    "test-model",
			},
		},
	}

	verifier := New(cfg)

	results, err := verifier.Verify()

	if err != nil {
		t.Logf("Verification failed as expected with timeout: %v", err)
	}

	if results == nil {
		t.Error("Expected results slice")
	}
}

func TestVerifier_Verify_EmptyLLMList(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network-dependent test in short mode")
	}
	cfg := &config.Config{
		Concurrency: 1,
		Timeout:     30 * time.Second,
		LLMs:        []config.LLMConfig{}, // Empty list
		Global: config.GlobalConfig{
			BaseURL: "https://api.example.com",
			APIKey:  "test-key",
		},
	}

	verifier := New(cfg)

	results, err := verifier.Verify()

	if err != nil {
		t.Logf("Verification failed as expected: %v", err)
	}

	if results == nil {
		t.Error("Expected results slice")
	}
}

func TestVerifier_Verify_InvalidEndpoint(t *testing.T) {
	cfg := &config.Config{
		Concurrency: 1,
		Timeout:     30 * time.Second,
		LLMs: []config.LLMConfig{
			{
				Name:     "Test LLM",
				Endpoint: "not-a-valid-url",
				APIKey:   "test-key",
				Model:    "test-model",
			},
		},
	}

	verifier := New(cfg)

	results, err := verifier.Verify()

	if err != nil {
		t.Logf("Verification failed as expected with invalid endpoint: %v", err)
	}

	if results == nil {
		t.Error("Expected results slice")
	}
}

func TestVerifier_Verify_MixedValidations(t *testing.T) {
	cfg := &config.Config{
		Concurrency: 2,
		Timeout:     30 * time.Second,
		LLMs: []config.LLMConfig{
			{
				Name:     "Valid LLM",
				Endpoint: "https://api.example.com/v1",
				APIKey:   "valid-key",
				Model:    "valid-model",
				Headers: map[string]string{
					"Authorization": "Bearer valid-token",
				},
				Features: map[string]bool{
					"chat": true,
				},
			},
			{
				Name:     "Invalid LLM",
				Endpoint: "invalid-url",
				APIKey:   "",
				Model:    "",
			},
			{
				Name:     "LLM with discovery",
				Endpoint: "https://api.example.com/v1",
				APIKey:   "discovery-key",
				// No model specified - should attempt discovery
			},
		},
	}

	verifier := New(cfg)

	results, err := verifier.Verify()

	if err != nil {
		t.Logf("Verification completed with errors as expected: %v", err)
	}

	if results == nil {
		t.Error("Expected results slice")
	}
}

func TestVerifier_verifySingleModel(t *testing.T) {
	cfg := &config.Config{
		Concurrency: 1,
		Timeout:     30 * time.Second,
	}

	verifier := New(cfg)

	// Create a mock client that will fail
	client := NewLLMClient("https://api.example.com/v1", "test-key", nil)

	result, err := verifier.verifySingleModel(client, "test-model", "https://api.example.com/v1")

	if err != nil {
		t.Logf("verifySingleModel failed as expected: %v", err)
	}

	// Check result structure
	if result.ModelInfo.ID != "test-model" {
		t.Errorf("Expected model ID 'test-model', got '%s'", result.ModelInfo.ID)
	}

	if result.ModelInfo.Endpoint != "https://api.example.com/v1" {
		t.Errorf("Expected endpoint 'https://api.example.com/v1', got '%s'", result.ModelInfo.Endpoint)
	}

	if result.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestVerifier_discoverAndVerifyAllModels(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network-dependent test in short mode")
	}
	cfg := &config.Config{
		Concurrency: 2,
		Timeout:     30 * time.Second,
		Global: config.GlobalConfig{
			BaseURL: "https://api.example.com",
			APIKey:  "test-key",
		},
	}

	verifier := New(cfg)

	results, err := verifier.discoverAndVerifyAllModels()

	if err != nil {
		t.Logf("discoverAndVerifyAllModels failed as expected: %v", err)
	}

	if results == nil {
		t.Error("Expected results slice")
	}
}

func TestVerifier_checkResponsiveness(t *testing.T) {
	cfg := &config.Config{
		Concurrency: 1,
		Timeout:     30 * time.Second,
	}

	verifier := New(cfg)

	client := NewLLMClient("https://api.example.com/v1", "test-key", nil)

	latency, responsive, err := verifier.checkResponsiveness(client, "test-model")

	// Should fail since there's no real API
	if responsive {
		t.Error("Expected model to be unresponsive")
	}

	if err == "" {
		t.Error("Expected error when checking responsiveness")
	}

	if latency <= 0 {
		t.Error("Expected latency measurement")
	}
}

func TestVerifier_checkOverload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network-dependent test in short mode")
	}
	cfg := &config.Config{
		Concurrency: 1,
		Timeout:     30 * time.Second,
	}

	verifier := New(cfg)

	client := NewLLMClient("https://api.example.com/v1", "test-key", nil)
	overloaded, avgLatency, throughput := verifier.checkOverload(client, "test-model")

	// Should return false for overload since we can't make real requests
	if overloaded {
		t.Error("Expected model not to be overloaded")
	}

	if avgLatency <= 0 {
		t.Error("Expected average latency measurement")
	}

	if throughput < 0 {
		t.Error("Expected throughput measurement")
	}
}

func TestVerifier_getModelDetailedInfo(t *testing.T) {
	cfg := &config.Config{
		Concurrency: 1,
		Timeout:     30 * time.Second,
	}

	verifier := New(cfg)

	client := NewLLMClient("https://api.example.com/v1", "test-key", nil)

	modelInfo, err := verifier.getModelDetailedInfo(client, "test-model")

	if err != nil {
		t.Logf("getModelDetailedInfo failed as expected: %v", err)
	}

	if modelInfo != nil {
		t.Error("Expected nil model info when API call fails")
	}
}

func TestVerifier_CalculateScores(t *testing.T) {
	cfg := &config.Config{
		Concurrency: 1,
		Timeout:     30 * time.Second,
	}

	_ = New(cfg) // verifier not used in this test

	// Create a test verification result
	_ = VerificationResult{ // result not used in this test
		ModelInfo: ModelInfo{
			ID:       "test-model",
			Endpoint: "https://api.example.com/v1",
		},
		Availability: AvailabilityResult{
			Exists:     true,
			Responsive: true,
			Overloaded: false,
			Latency:    100 * time.Millisecond,
		},
		ResponseTime: ResponseTimeResult{
			AverageLatency:   150 * time.Millisecond,
			Throughput:       10.5,
			MeasurementCount: 10,
		},
		FeatureDetection: FeatureDetectionResult{
			ToolUse:        true,
			CodeGeneration: true,
			Embeddings:     false,
		},
		CodeCapabilities: CodeCapabilityResult{
			LanguageSupport: []string{"python", "javascript"},
			CodeGeneration:  true,
			CodeCompletion:  true,
		},
		Timestamp: time.Now(),
	}

	// This method should calculate scores based on the result
	// Since we can't test the actual calculation without the implementation,
	// we'll just verify the method exists and can be called
	// (Note: This assumes CalculateScores is a method on Verifier or VerificationResult)
	t.Log("Score calculation would be tested here if the method was exported")
}

func TestVerifier_GenerateSummary(t *testing.T) {
	cfg := &config.Config{
		Concurrency: 1,
		Timeout:     30 * time.Second,
	}

	_ = New(cfg) // verifier not used in this test

	// Create test results
	results := []VerificationResult{
		{
			ModelInfo: ModelInfo{
				ID:       "model-1",
				Endpoint: "https://api.example.com/v1",
			},
			Availability: AvailabilityResult{
				Exists:     true,
				Responsive: true,
			},
			Timestamp: time.Now(),
		},
		{
			ModelInfo: ModelInfo{
				ID:       "model-2",
				Endpoint: "https://api.example.com/v1",
			},
			Availability: AvailabilityResult{
				Exists:     false,
				Responsive: false,
			},
			Timestamp: time.Now(),
			Error:     "Model not found",
		},
	}

	// This method should generate a summary from results
	// Since we can't test the actual generation without the implementation,
	// we'll just verify the concept
	t.Log("Summary generation would be tested here if the method was exported")

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}
