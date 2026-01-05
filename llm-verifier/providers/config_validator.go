package providers

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ConfigValidator validates provider configurations
type ConfigValidator struct {
	strictMode bool
	rules      []ValidationRule
}

// ValidationRule represents a validation rule
type ValidationRule struct {
	Field       string
	Description string
	Validate    func(value interface{}) error
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for %s: %s", e.Field, e.Message)
}

// ValidationResult contains the results of validation
type ValidationResult struct {
	Valid    bool
	Errors   []ValidationError
	Warnings []string
}

// ConfigValidatorOption configures the validator
type ConfigValidatorOption func(*ConfigValidator)

// WithStrictMode enables strict validation mode
func WithStrictMode() ConfigValidatorOption {
	return func(v *ConfigValidator) {
		v.strictMode = true
	}
}

// WithCustomRule adds a custom validation rule
func WithCustomRule(rule ValidationRule) ConfigValidatorOption {
	return func(v *ConfigValidator) {
		v.rules = append(v.rules, rule)
	}
}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator(opts ...ConfigValidatorOption) *ConfigValidator {
	v := &ConfigValidator{
		strictMode: false,
		rules:      make([]ValidationRule, 0),
	}

	// Add default rules
	v.rules = append(v.rules, defaultValidationRules()...)

	// Apply options
	for _, opt := range opts {
		opt(v)
	}

	return v
}

// Validate validates a provider configuration
func (v *ConfigValidator) Validate(config *ProviderConfig) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]string, 0),
	}

	if config == nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "config",
			Message: "configuration cannot be nil",
		})
		return result
	}

	// Validate name
	if config.Name == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "name",
			Message: "provider name is required",
		})
	} else if !isValidProviderName(config.Name) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "name",
			Message: "provider name contains invalid characters",
		})
	}

	// Validate endpoint
	if config.Endpoint == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "endpoint",
			Message: "endpoint URL is required",
		})
	} else if !isValidURL(config.Endpoint) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "endpoint",
			Message: "endpoint must be a valid URL",
		})
	}

	// Validate auth type
	validAuthTypes := map[string]bool{
		"bearer":  true,
		"api_key": true,
		"oauth":   true,
		"basic":   true,
		"none":    true,
	}
	if config.AuthType != "" && !validAuthTypes[config.AuthType] {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "auth_type",
			Message: fmt.Sprintf("invalid auth type: %s", config.AuthType),
		})
	}

	// Validate streaming format
	validStreamingFormats := map[string]bool{
		"sse":       true,
		"websocket": true,
		"json":      true,
		"ndjson":    true,
		"":          true, // Allow empty
	}
	if !validStreamingFormats[config.StreamingFormat] {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "streaming_format",
			Message: fmt.Sprintf("invalid streaming format: %s", config.StreamingFormat),
		})
	}

	// Validate rate limits
	if config.RateLimits.RequestsPerMinute < 0 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "rate_limits.requests_per_minute",
			Message: "requests per minute cannot be negative",
		})
	}

	// Validate timeouts
	if config.Timeouts.RequestTimeout < 0 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "timeouts.request_timeout",
			Message: "request timeout cannot be negative",
		})
	}

	// Validate retry config
	if config.RetryConfig.MaxRetries < 0 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "retry_config.max_retries",
			Message: "max retries cannot be negative",
		})
	}

	if config.RetryConfig.BackoffFactor < 1.0 && config.RetryConfig.BackoffFactor != 0 {
		result.Warnings = append(result.Warnings, "backoff factor less than 1.0 may not be effective")
	}

	// Strict mode additional validations
	if v.strictMode {
		if config.DefaultModel == "" {
			result.Warnings = append(result.Warnings, "default model is not set")
		}

		if config.Timeouts.RequestTimeout == 0 {
			result.Warnings = append(result.Warnings, "request timeout is not set, using default")
		}

		if config.RetryConfig.MaxRetries == 0 {
			result.Warnings = append(result.Warnings, "retry configuration is disabled")
		}
	}

	// Apply custom rules
	for _, rule := range v.rules {
		if err := rule.Validate(config); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   rule.Field,
				Message: err.Error(),
			})
		}
	}

	return result
}

// ValidateAPIKey validates an API key format
func (v *ConfigValidator) ValidateAPIKey(key string) error {
	if key == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	if len(key) < 8 {
		return fmt.Errorf("API key is too short")
	}

	// Check for common test/placeholder keys
	testPatterns := []string{"test", "demo", "example", "xxx", "placeholder"}
	lowerKey := strings.ToLower(key)
	for _, pattern := range testPatterns {
		if strings.Contains(lowerKey, pattern) {
			return fmt.Errorf("API key appears to be a test/placeholder key")
		}
	}

	return nil
}

// ValidateEndpoint validates an endpoint URL
func (v *ConfigValidator) ValidateEndpoint(endpoint string) error {
	if endpoint == "" {
		return fmt.Errorf("endpoint cannot be empty")
	}

	if !isValidURL(endpoint) {
		return fmt.Errorf("endpoint must be a valid URL")
	}

	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("failed to parse endpoint URL: %w", err)
	}

	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return fmt.Errorf("endpoint must use http or https scheme")
	}

	if v.strictMode && parsedURL.Scheme != "https" {
		return fmt.Errorf("endpoint must use https in strict mode")
	}

	return nil
}

// ValidateTimeouts validates timeout configurations
func (v *ConfigValidator) ValidateTimeouts(timeouts TimeoutConfig) error {
	if timeouts.RequestTimeout < 0 {
		return fmt.Errorf("request timeout cannot be negative")
	}

	if timeouts.ConnectTimeout < 0 {
		return fmt.Errorf("connect timeout cannot be negative")
	}

	if timeouts.StreamTimeout < 0 {
		return fmt.Errorf("stream timeout cannot be negative")
	}

	// Check for reasonable values
	if timeouts.RequestTimeout > 5*time.Minute {
		return fmt.Errorf("request timeout exceeds maximum allowed (5 minutes)")
	}

	return nil
}

// Helper functions

func isValidProviderName(name string) bool {
	// Allow alphanumeric, hyphens, and underscores
	matched, _ := regexp.MatchString(`^[a-zA-Z][a-zA-Z0-9_-]*$`, name)
	return matched
}

func isValidURL(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	return parsedURL.Scheme != "" && parsedURL.Host != ""
}

func defaultValidationRules() []ValidationRule {
	return []ValidationRule{
		// Add default validation rules here if needed
	}
}
