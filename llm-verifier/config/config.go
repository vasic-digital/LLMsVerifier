package config

import "time"

// Config represents the main configuration for the LLM verifier
type Config struct {
	LLMs        []LLMConfig `mapstructure:"llms"`
	Global      GlobalConfig `mapstructure:"global"`
	Concurrency int          `mapstructure:"concurrency"`
	Timeout     time.Duration `mapstructure:"timeout"`
}

// LLMConfig represents configuration for a single LLM endpoint
type LLMConfig struct {
	Name     string            `mapstructure:"name"`              // Name of the LLM service
	Endpoint string            `mapstructure:"endpoint"`          // API endpoint URL
	APIKey   string            `mapstructure:"api_key"`           // API key for authentication
	Model    string            `mapstructure:"model,omitempty"`   // Specific model to test (optional if auto-discovery)
	Headers  map[string]string `mapstructure:"headers,omitempty"` // Additional headers to send with requests
	Features map[string]bool   `mapstructure:"features,omitempty"` // Expected features of the model
}

// GlobalConfig holds global configuration options
type GlobalConfig struct {
	BaseURL      string            `mapstructure:"base_url"`         // Base URL for the API
	APIKey       string            `mapstructure:"api_key"`          // Global API key
	DefaultModel string            `mapstructure:"default_model"`    // Default model name
	MaxRetries   int               `mapstructure:"max_retries"`      // Maximum number of retries for failed requests
	RequestDelay time.Duration     `mapstructure:"request_delay"`    // Delay between requests
	Timeout      time.Duration     `mapstructure:"timeout"`          // Request timeout
	CustomParams map[string]interface{} `mapstructure:"custom_params"` // Custom parameters for API calls
}