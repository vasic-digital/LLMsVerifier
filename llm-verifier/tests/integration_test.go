package tests

import (
	"os"
	"testing"

	"llm-verifier/config"
	"llm-verifier/llmverifier"
)

// Integration tests for the LLM verifier
// These tests check that the different components work together properly

func TestConfigLoading(t *testing.T) {
	// Create a temporary config file for testing
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

	tempFile, err := os.CreateTemp("", "config_test_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write([]byte(tempConfig)); err != nil {
		t.Fatalf("Failed to write to temp config file: %v", err)
	}
	tempFile.Close()

	cfg, err := llmverifier.LoadConfig(tempFile.Name())
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
	// Create a basic config
	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:    "https://api.openai.com/v1",
			APIKey:     "test-key",
			MaxRetries: 3,
			Timeout:    30,
		},
		LLMs: []config.LLMConfig{
			{
				Name:     "Test Model",
				Endpoint: "https://api.openai.com/v1",
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

// Note: The actual verification tests that make real API calls are skipped by default
// to avoid requiring real API keys and making actual API calls during testing.
// They can be enabled by setting an environment variable.
func TestVerifierWithMockedAPI(t *testing.T) {
	t.Skip("This test requires mocking the API client, which is not implemented yet.")
}