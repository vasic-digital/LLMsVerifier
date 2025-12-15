package config

import "time"

// Config represents the main configuration for the LLM verifier
type Config struct {
	LLMs        []LLMConfig    `mapstructure:"llms"`
	Global      GlobalConfig   `mapstructure:"global"`
	Database    DatabaseConfig `mapstructure:"database"`
	API         APIConfig      `mapstructure:"api"`
	Concurrency int            `mapstructure:"concurrency"`
	Timeout     time.Duration  `mapstructure:"timeout"`
}

// LLMConfig represents configuration for a single LLM endpoint
type LLMConfig struct {
	Name     string            `mapstructure:"name"`               // Name of the LLM service
	Endpoint string            `mapstructure:"endpoint"`           // API endpoint URL
	APIKey   string            `mapstructure:"api_key"`            // API key for authentication
	Model    string            `mapstructure:"model,omitempty"`    // Specific model to test (optional if auto-discovery)
	Headers  map[string]string `mapstructure:"headers,omitempty"`  // Additional headers to send with requests
	Features map[string]bool   `mapstructure:"features,omitempty"` // Expected features of the model
}

// GlobalConfig holds global configuration options
type GlobalConfig struct {
	BaseURL      string                 `mapstructure:"base_url"`      // Base URL for the API
	APIKey       string                 `mapstructure:"api_key"`       // Global API key
	DefaultModel string                 `mapstructure:"default_model"` // Default model name
	MaxRetries   int                    `mapstructure:"max_retries"`   // Maximum number of retries for failed requests
	RequestDelay time.Duration          `mapstructure:"request_delay"` // Delay between requests
	Timeout      time.Duration          `mapstructure:"timeout"`       // Request timeout
	CustomParams map[string]interface{} `mapstructure:"custom_params"` // Custom parameters for API calls
}

// DatabaseConfig holds database configuration options
type DatabaseConfig struct {
	Path          string `mapstructure:"path"`           // Path to the database file
	EncryptionKey string `mapstructure:"encryption_key"` // Encryption key for SQL Cipher
}

// APIConfig holds REST API configuration options
type APIConfig struct {
	Port              string `mapstructure:"port"`                  // Port to run the API server on
	JWTSecret         string `mapstructure:"jwt_secret"`            // Secret key for JWT token signing
	RateLimit         int    `mapstructure:"rate_limit"`            // Global rate limit (requests per minute)
	BurstLimit        int    `mapstructure:"burst_limit"`           // Burst limit for short periods
	RateLimitWindow   int    `mapstructure:"rate_limit_window"`     // Rate limit window in seconds
	EnableCORS        bool   `mapstructure:"enable_cors"`           // Enable CORS headers
	TrustedProxies    string `mapstructure:"trusted_proxies"`       // Comma-separated list of trusted proxy IPs
	RateLimitByAPIKey bool   `mapstructure:"rate_limit_by_api_key"` // Rate limit by API key instead of IP
}
