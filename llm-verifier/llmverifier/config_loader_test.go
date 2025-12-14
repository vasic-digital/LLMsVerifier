package llmverifier

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yaml")
	configContent := `
llms:
  - name: "test-llm"
    endpoint: "https://api.test.com/v1"
    api_key: "test-api-key"
    model: "test-model"
    headers:
      Custom-Header: "value"
    features:
      tool_use: true
      code_generation: true

global:
  base_url: "https://api.test.com"
  api_key: "global-api-key"
  default_model: "default-model"
  max_retries: 3
  request_delay: 1s
  timeout: 30s
  custom_params:
    temperature: 0.7

database:
  path: "test.db"
  encryption_key: "test-encryption-key"

api:
  port: "8080"
  jwt_secret: "test-jwt-secret"
  rate_limit: 100
  enable_cors: true

concurrency: 5
timeout: 120s
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Test loading the config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify LLM configuration
	if len(cfg.LLMs) != 1 {
		t.Errorf("Expected 1 LLM, got %d", len(cfg.LLMs))
	}

	llm := cfg.LLMs[0]
	if llm.Name != "test-llm" {
		t.Errorf("Expected LLM name 'test-llm', got '%s'", llm.Name)
	}
	if llm.Endpoint != "https://api.test.com/v1" {
		t.Errorf("Expected endpoint 'https://api.test.com/v1', got '%s'", llm.Endpoint)
	}
	if llm.APIKey != "test-api-key" {
		t.Errorf("Expected API key 'test-api-key', got '%s'", llm.APIKey)
	}
	if llm.Model != "test-model" {
		t.Errorf("Expected model 'test-model', got '%s'", llm.Model)
	}
	if llm.Headers["Custom-Header"] != "value" {
		t.Errorf("Expected custom header 'value', got '%s'", llm.Headers["Custom-Header"])
	}
	if !llm.Features["tool_use"] {
		t.Error("Expected tool_use feature to be true")
	}
	if !llm.Features["code_generation"] {
		t.Error("Expected code_generation feature to be true")
	}

	// Verify global configuration
	if cfg.Global.BaseURL != "https://api.test.com" {
		t.Errorf("Expected base URL 'https://api.test.com', got '%s'", cfg.Global.BaseURL)
	}
	if cfg.Global.APIKey != "global-api-key" {
		t.Errorf("Expected global API key 'global-api-key', got '%s'", cfg.Global.APIKey)
	}
	if cfg.Global.DefaultModel != "default-model" {
		t.Errorf("Expected default model 'default-model', got '%s'", cfg.Global.DefaultModel)
	}
	if cfg.Global.MaxRetries != 3 {
		t.Errorf("Expected max retries 3, got %d", cfg.Global.MaxRetries)
	}
	if cfg.Global.RequestDelay != time.Second {
		t.Errorf("Expected request delay 1s, got %v", cfg.Global.RequestDelay)
	}
	if cfg.Global.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", cfg.Global.Timeout)
	}
	if temp, ok := cfg.Global.CustomParams["temperature"].(float64); !ok || temp != 0.7 {
		t.Errorf("Expected custom param temperature 0.7, got %v", cfg.Global.CustomParams["temperature"])
	}

	// Verify database configuration
	if cfg.Database.Path != "test.db" {
		t.Errorf("Expected database path 'test.db', got '%s'", cfg.Database.Path)
	}
	if cfg.Database.EncryptionKey != "test-encryption-key" {
		t.Errorf("Expected encryption key 'test-encryption-key', got '%s'", cfg.Database.EncryptionKey)
	}

	// Verify API configuration
	if cfg.API.Port != "8080" {
		t.Errorf("Expected API port '8080', got '%s'", cfg.API.Port)
	}
	if cfg.API.JWTSecret != "test-jwt-secret" {
		t.Errorf("Expected JWT secret 'test-jwt-secret', got '%s'", cfg.API.JWTSecret)
	}
	if cfg.API.RateLimit != 100 {
		t.Errorf("Expected rate limit 100, got %d", cfg.API.RateLimit)
	}
	if !cfg.API.EnableCORS {
		t.Error("Expected CORS to be enabled")
	}

	// Verify top-level configuration
	if cfg.Concurrency != 5 {
		t.Errorf("Expected concurrency 5, got %d", cfg.Concurrency)
	}
	if cfg.Timeout != 120*time.Second {
		t.Errorf("Expected timeout 120s, got %v", cfg.Timeout)
	}
}

func TestLoadConfig_EnvironmentVariables(t *testing.T) {
	// Create a temporary config file with environment variables
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_env_config.yaml")
	configContent := `
llms:
  - name: "test-llm"
    endpoint: "${TEST_ENDPOINT}"
    api_key: "${TEST_API_KEY}"

global:
  api_key: "${GLOBAL_API_KEY}"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Set environment variables
	os.Setenv("TEST_ENDPOINT", "https://env.test.com/v1")
	os.Setenv("TEST_API_KEY", "env-api-key")
	os.Setenv("GLOBAL_API_KEY", "env-global-key")
	defer func() {
		os.Unsetenv("TEST_ENDPOINT")
		os.Unsetenv("TEST_API_KEY")
		os.Unsetenv("GLOBAL_API_KEY")
	}()

	// Test loading the config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify environment variable expansion
	if len(cfg.LLMs) != 1 {
		t.Errorf("Expected 1 LLM, got %d", len(cfg.LLMs))
	}

	llm := cfg.LLMs[0]
	if llm.Endpoint != "https://env.test.com/v1" {
		t.Errorf("Expected expanded endpoint 'https://env.test.com/v1', got '%s'", llm.Endpoint)
	}
	if llm.APIKey != "env-api-key" {
		t.Errorf("Expected expanded API key 'env-api-key', got '%s'", llm.APIKey)
	}
	if cfg.Global.APIKey != "env-global-key" {
		t.Errorf("Expected expanded global API key 'env-global-key', got '%s'", cfg.Global.APIKey)
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	// Create a minimal config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_minimal_config.yaml")
	configContent := `
llms:
  - name: "test-llm"
    endpoint: "https://api.test.com/v1"
    api_key: "test-api-key"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Test loading the config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify default values
	if cfg.Concurrency != 1 {
		t.Errorf("Expected default concurrency 1, got %d", cfg.Concurrency)
	}
	if cfg.Timeout != 60*time.Second {
		t.Errorf("Expected default timeout 60s, got %v", cfg.Timeout)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error for non-existent config file")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	// Create a temporary config file with invalid YAML
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_invalid_config.yaml")
	configContent := `invalid: yaml: content: [`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestLoadConfig_EmptyConfig(t *testing.T) {
	// Create an empty config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_empty_config.yaml")
	configContent := ``

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Should load with defaults
	if cfg == nil {
		t.Error("Expected config to be loaded")
	}
	if len(cfg.LLMs) != 0 {
		t.Errorf("Expected 0 LLMs, got %d", len(cfg.LLMs))
	}
}
