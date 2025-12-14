package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"llm-verifier/config"
	"llm-verifier/database"
	"llm-verifier/llmverifier"
)

// Integration tests for the LLM verifier
// These tests check that the different components work together properly

func TestConfigLoading(t *testing.T) {
	// Create a temporary config file for testing
	tempDir := t.TempDir()
	tempConfig := `global:
  base_url: "https://api.openai.com/v1"
  api_key: "test-key"
  max_retries: 3
  request_delay: 1s
  timeout: 30s

llms:
  - name: "Test Model"
    endpoint: "https://api.openai.com/v1"
    api_key: "test-key"
    model: "gpt-3.5-turbo"

concurrency: 2
timeout: 60s
`

	configPath := filepath.Join(tempDir, "test-config.yaml")
	err := os.WriteFile(configPath, []byte(tempConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}

	cfg, err := llmverifier.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Global.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("Expected base_url to be 'https://api.openai.com/v1', got %s", cfg.Global.BaseURL)
	}

	if len(cfg.LLMs) != 1 {
		t.Errorf("Expected 1 LLM configuration, got %d", len(cfg.LLMs))
	}

	if cfg.LLMs[0].Model != "gpt-3.5-turbo" {
		t.Errorf("Expected model to be 'gpt-3.5-turbo', got %s", cfg.LLMs[0].Model)
	}

	if cfg.Concurrency != 2 {
		t.Errorf("Expected concurrency to be 2, got %d", cfg.Concurrency)
	}
}

func TestVerifierInitialization(t *testing.T) {
	helper, cleanup := SetupTestEnvironment(t)
	defer cleanup()
	
	// Create a basic config
	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:    helper.MockServer.URL + "/v1",
			APIKey:     "test-key",
			MaxRetries: 3,
			Timeout:    30,
		},
		LLMs: []config.LLMConfig{
			{
				Name:     "Test Model",
				Endpoint: helper.MockServer.URL + "/v1",
				APIKey:   "test-key",
				Model:    "gpt-3.5-turbo",
			},
		},
		Concurrency: 1,
		Timeout:     60,
	}

	verifier := llmverifier.New(cfg)

	if verifier == nil {
		t.Error("Expected verifier to be initialized, got nil")
	}
}

// Test that the data structures properly marshal to JSON
func TestJSONMarshaling(t *testing.T) {
	now := time.Now()
	result := llmverifier.VerificationResult{
		ModelInfo: llmverifier.ModelInfo{
			ID:      "gpt-4-test",
			Object:  "model",
			Created: now.Unix(),
		},
		FeatureDetection: llmverifier.FeatureDetectionResult{
			MCPs:            true,
			LSPs:            false,
			Reranking:       true,
			ImageGeneration: true,
			AudioGeneration: true,
			VideoGeneration: false,
		},
		CodeCapabilities: llmverifier.CodeCapabilityResult{
			CodeGeneration: true,
			CodeCompletion: true,
		},
		GenerativeCapabilities: llmverifier.GenerativeCapabilityResult{
			CreativeWriting: true,
			Storytelling:    true,
		},
		Timestamp: now,
		OverallScore: 85.5,
	}

	// Test JSON marshaling
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal VerificationResult to JSON: %v", err)
	}

	if len(data) == 0 {
		t.Error("Marshaled JSON should not be empty")
	}

	// Test JSON unmarshaling
	var unmarshaledResult llmverifier.VerificationResult
	err = json.Unmarshal(data, &unmarshaledResult)
	if err != nil {
		t.Fatalf("Failed to unmarshal VerificationResult from JSON: %v", err)
	}

	if unmarshaledResult.ModelInfo.ID != "gpt-4-test" {
		t.Errorf("Expected model ID 'gpt-4-test', got '%s'", unmarshaledResult.ModelInfo.ID)
	}
}

func TestVerifierWithMockedAPI(t *testing.T) {
	helper, cleanup := SetupTestEnvironment(t)
	defer cleanup()
	
	// Create test verifier with mocked API
	verifier := CreateTestVerifier(helper.Config)
	
	// Test verification workflow with mocked API
	results, err := verifier.Verify()
	AssertNoError(t, err)
	AssertTrue(t, results != nil, "Results should not be nil")
	AssertTrue(t, len(results) > 0, "Should have verification results")
	
	// Verify that results contain expected data
	for _, result := range results {
		AssertTrue(t, result.ModelInfo.ID != "", "Model should have ID")
		AssertTrue(t, result.OverallScore >= 0, "Score should be non-negative")
		AssertTrue(t, result.OverallScore <= 100, "Score should not exceed 100")
	}
}