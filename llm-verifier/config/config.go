package config

import "time"

// Config represents the main configuration for the LLM verifier
type Config struct {
	LLMs        []LLMConfig `yaml:"llms"`
	Global      GlobalConfig `yaml:"global"`
	Concurrency int          `yaml:"concurrency"`
	Timeout     time.Duration `yaml:"timeout"`
}

// LLMConfig represents configuration for a single LLM endpoint
type LLMConfig struct {
	Name     string            `yaml:"name"`              // Name of the LLM service
	Endpoint string            `yaml:"endpoint"`          // API endpoint URL
	APIKey   string            `yaml:"api_key"`           // API key for authentication
	Model    string            `yaml:"model,omitempty"`   // Specific model to test (optional if auto-discovery)
	Headers  map[string]string `yaml:"headers,omitempty"` // Additional headers to send with requests
	Features map[string]bool   `yaml:"features,omitempty"` // Expected features of the model
}

// GlobalConfig holds global configuration options
type GlobalConfig struct {
	BaseURL      string            `yaml:"base_url"`         // Base URL for the API
	APIKey       string            `yaml:"api_key"`          // Global API key
	DefaultModel string            `yaml:"default_model"`    // Default model name
	MaxRetries   int               `yaml:"max_retries"`      // Maximum number of retries for failed requests
	RequestDelay time.Duration     `yaml:"request_delay"`    // Delay between requests
	Timeout      time.Duration     `yaml:"timeout"`          // Request timeout
	CustomParams map[string]interface{} `yaml:"custom_params"` // Custom parameters for API calls
}