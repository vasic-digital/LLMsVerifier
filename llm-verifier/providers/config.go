package providers

import (
	"time"
)

// ProviderConfig represents configuration for a specific provider
type ProviderConfig struct {
	Name            string                 `json:"name"`
	Endpoint        string                 `json:"endpoint"`
	AuthType        string                 `json:"auth_type"`        // "bearer", "api_key", "oauth"
	StreamingFormat string                 `json:"streaming_format"` // "sse", "websocket", "json"
	DefaultModel    string                 `json:"default_model"`
	RateLimits      RateLimitConfig        `json:"rate_limits"`
	Timeouts        TimeoutConfig          `json:"timeouts"`
	RetryConfig     RetryConfig            `json:"retry_config"`
	Features        map[string]interface{} `json:"features"`
}

// RateLimitConfig defines rate limiting settings
type RateLimitConfig struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	RequestsPerHour   int `json:"requests_per_hour"`
	BurstLimit        int `json:"burst_limit"`
}

// TimeoutConfig defines timeout settings
type TimeoutConfig struct {
	RequestTimeout time.Duration `json:"request_timeout"`
	StreamTimeout  time.Duration `json:"stream_timeout"`
	ConnectTimeout time.Duration `json:"connect_timeout"`
}

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxRetries      int           `json:"max_retries"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	RetryableErrors []string      `json:"retryable_errors"`
}

// ProviderRegistry manages provider configurations
type ProviderRegistry struct {
	providers map[string]*ProviderConfig
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	pr := &ProviderRegistry{
		providers: make(map[string]*ProviderConfig),
	}
	pr.registerDefaultProviders()
	return pr
}

// GetConfig returns configuration for a provider
func (pr *ProviderRegistry) GetConfig(providerName string) (*ProviderConfig, bool) {
	config, exists := pr.providers[providerName]
	return config, exists
}

// RegisterProvider registers a custom provider configuration
func (pr *ProviderRegistry) RegisterProvider(config *ProviderConfig) {
	pr.providers[config.Name] = config
}

// registerDefaultProviders registers built-in provider configurations
func (pr *ProviderRegistry) registerDefaultProviders() {
	// OpenAI configuration
	pr.providers["openai"] = &ProviderConfig{
		Name:            "openai",
		Endpoint:        "https://api.openai.com/v1",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "gpt-4",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 60,
			RequestsPerHour:   1000,
			BurstLimit:        10,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": true,
			"supports_vision":    true,
			"max_context_length": 128000,
			"supported_models":   []string{"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"},
		},
	}

	// DeepSeek configuration
	pr.providers["deepseek"] = &ProviderConfig{
		Name:            "deepseek",
		Endpoint:        "https://api.deepseek.com/v1",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "deepseek-chat",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 30,
			RequestsPerHour:   1000,
			BurstLimit:        5,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    false,
			"max_context_length": 32768,
			"supported_models":   []string{"deepseek-chat", "deepseek-coder"},
		},
	}

	// Anthropic configuration
	pr.providers["anthropic"] = &ProviderConfig{
		Name:            "anthropic",
		Endpoint:        "https://api.anthropic.com/v1",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "claude-3-opus-20240229",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 50,
			RequestsPerHour:   1000,
			BurstLimit:        8,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 120 * time.Second, // Claude can be slower
			StreamTimeout:  600 * time.Second,
			ConnectTimeout: 15 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    2 * time.Second,
			MaxDelay:        60 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504", "529"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    true,
			"max_context_length": 200000,
			"supported_models":   []string{"claude-3-opus", "claude-3-sonnet", "claude-3-haiku"},
		},
	}

	// Google AI configuration
	pr.providers["google"] = &ProviderConfig{
		Name:            "google",
		Endpoint:        "https://generativelanguage.googleapis.com/v1beta",
		AuthType:        "api_key",
		StreamingFormat: "sse",
		DefaultModel:    "gemini-pro",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 60,
			RequestsPerHour:   1000,
			BurstLimit:        10,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    true,
			"max_context_length": 32768,
			"supported_models":   []string{"gemini-pro", "gemini-pro-vision"},
		},
	}

	// Generic configuration for unknown providers
	pr.providers["generic"] = &ProviderConfig{
		Name:            "generic",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "unknown",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 30,
			RequestsPerHour:   500,
			BurstLimit:        5,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 30 * time.Second,
			StreamTimeout:  180 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      2,
			InitialDelay:    1 * time.Second,
			MaxDelay:        15 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    false,
			"max_context_length": 4096,
			"supported_models":   []string{},
		},
	}
}

// GetProviderNames returns all registered provider names
func (pr *ProviderRegistry) GetProviderNames() []string {
	names := make([]string, 0, len(pr.providers))
	for name := range pr.providers {
		names = append(names, name)
	}
	return names
}

// IsProviderSupported checks if a provider is supported
func (pr *ProviderRegistry) IsProviderSupported(providerName string) bool {
	_, exists := pr.providers[providerName]
	return exists
}

// GetDefaultConfig returns a default configuration for unknown providers
func (pr *ProviderRegistry) GetDefaultConfig() *ProviderConfig {
	config := *pr.providers["generic"] // Copy the generic config
	return &config
}
