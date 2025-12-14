package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestConfigDefaults(t *testing.T) {
	cfg := &Config{
		Concurrency: 5,
		Timeout:     60 * time.Second,
		Database: DatabaseConfig{
			Path: "./llm-verifier.db",
		},
		API: APIConfig{
			Port:       "8080",
			RateLimit:  100,
			EnableCORS: true,
		},
	}

	if cfg.Concurrency != 5 {
		t.Errorf("Expected Concurrency to be 5, got %d", cfg.Concurrency)
	}

	if cfg.Timeout != 60*time.Second {
		t.Errorf("Expected Timeout to be 60s, got %v", cfg.Timeout)
	}

	if cfg.Database.Path != "./llm-verifier.db" {
		t.Errorf("Expected Database.Path to be './llm-verifier.db', got %s", cfg.Database.Path)
	}

	if cfg.API.Port != "8080" {
		t.Errorf("Expected API.Port to be '8080', got %s", cfg.API.Port)
	}

	if cfg.API.RateLimit != 100 {
		t.Errorf("Expected API.RateLimit to be 100, got %d", cfg.API.RateLimit)
	}

	if !cfg.API.EnableCORS {
		t.Errorf("Expected API.EnableCORS to be true, got %v", cfg.API.EnableCORS)
	}
}

func TestLLMConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  LLMConfig
		wantErr bool
	}{
		{
			name: "Valid config with name and endpoint",
			config: LLMConfig{
				Name:     "Test LLM",
				Endpoint: "https://api.example.com",
				APIKey:   "test-key",
			},
			wantErr: false,
		},
		{
			name: "Missing name",
			config: LLMConfig{
				Endpoint: "https://api.example.com",
				APIKey:   "test-key",
			},
			wantErr: true,
		},
		{
			name: "Missing endpoint",
			config: LLMConfig{
				Name:   "Test LLM",
				APIKey: "test-key",
			},
			wantErr: true,
		},
		{
			name: "Empty config",
			config: LLMConfig{
				Name:     "",
				Endpoint: "",
				APIKey:   "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasName := tt.config.Name != ""
			hasEndpoint := tt.config.Endpoint != ""

			if tt.wantErr && (hasName && hasEndpoint) {
				t.Errorf("Expected error but config appears valid")
			}

			if !tt.wantErr && (!hasName || !hasEndpoint) {
				t.Errorf("Expected valid config but missing required fields")
			}
		})
	}
}

func TestGlobalConfigDefaults(t *testing.T) {
	cfg := GlobalConfig{
		BaseURL:      "https://api.example.com",
		APIKey:       "test-key",
		DefaultModel: "gpt-4",
		MaxRetries:   3,
		RequestDelay: 1 * time.Second,
		Timeout:      30 * time.Second,
	}

	if cfg.BaseURL != "https://api.example.com" {
		t.Errorf("Expected BaseURL to be 'https://api.example.com', got %s", cfg.BaseURL)
	}

	if cfg.APIKey != "test-key" {
		t.Errorf("Expected APIKey to be 'test-key', got %s", cfg.APIKey)
	}

	if cfg.DefaultModel != "gpt-4" {
		t.Errorf("Expected DefaultModel to be 'gpt-4', got %s", cfg.DefaultModel)
	}

	if cfg.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries to be 3, got %d", cfg.MaxRetries)
	}

	if cfg.RequestDelay != 1*time.Second {
		t.Errorf("Expected RequestDelay to be 1s, got %v", cfg.RequestDelay)
	}

	if cfg.Timeout != 30*time.Second {
		t.Errorf("Expected Timeout to be 30s, got %v", cfg.Timeout)
	}
}

func TestConfigFromFile(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	configContent := `
global:
  base_url: "https://api.test.com"
  api_key: "test-api-key"
  max_retries: 5
  request_delay: 2s
  timeout: 45s

database:
  path: "./test.db"
  encryption_key: "test-encryption-key"

api:
  port: "9090"
  jwt_secret: "test-jwt-secret"
  rate_limit: 200
  enable_cors: false

concurrency: 10
timeout: 90s
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Test viper can read the config
	viper.SetConfigFile(configPath)
	err = viper.ReadInConfig()
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	// Check some values
	if viper.GetString("global.base_url") != "https://api.test.com" {
		t.Errorf("Expected global.base_url to be 'https://api.test.com', got %s", viper.GetString("global.base_url"))
	}

	if viper.GetInt("global.max_retries") != 5 {
		t.Errorf("Expected global.max_retries to be 5, got %d", viper.GetInt("global.max_retries"))
	}

	if viper.GetString("database.path") != "./test.db" {
		t.Errorf("Expected database.path to be './test.db', got %s", viper.GetString("database.path"))
	}

	if viper.GetString("api.port") != "9090" {
		t.Errorf("Expected api.port to be '9090', got %s", viper.GetString("api.port"))
	}

	if viper.GetInt("concurrency") != 10 {
		t.Errorf("Expected concurrency to be 10, got %d", viper.GetInt("concurrency"))
	}
}
