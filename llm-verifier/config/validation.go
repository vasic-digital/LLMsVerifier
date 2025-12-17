package config

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (ve ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", ve.Field, ve.Message)
}

// ValidationResult contains the result of configuration validation
type ValidationResult struct {
	Valid  bool
	Errors []ValidationError
}

// ValidateConfig validates the entire configuration
func ValidateConfig(cfg *Config) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: make([]ValidationError, 0),
	}

	// Validate global config
	result.merge(validateGlobalConfig(&cfg.Global))

	// Validate LLMs
	for i, llm := range cfg.LLMs {
		result.merge(validateLLMConfig(llm, i))
	}

	// Validate database config
	result.merge(validateDatabaseConfig(&cfg.Database))

	// Validate API config
	result.merge(validateAPIConfig(&cfg.API))

	// Validate monitoring config
	result.merge(validateMonitoringConfig(&cfg.Monitoring))

	return result
}

// validateGlobalConfig validates global configuration
func validateGlobalConfig(global *GlobalConfig) *ValidationResult {
	result := &ValidationResult{Valid: true, Errors: make([]ValidationError, 0)}

	// Validate BaseURL
	if global.BaseURL != "" {
		if parsedURL, err := url.Parse(global.BaseURL); err != nil {
			result.addError("global.base_url", "invalid URL format")
		} else if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			result.addError("global.base_url", "URL must use http or https scheme")
		}
	}

	// Validate DefaultModel
	if global.DefaultModel == "" {
		result.addError("global.default_model", "default model cannot be empty")
	}

	// Validate timeout
	if global.Timeout <= 0 {
		result.addError("global.timeout", "timeout must be greater than 0")
	}

	return result
}

// validateLLMConfig validates a single LLM configuration
func validateLLMConfig(llm LLMConfig, index int) *ValidationResult {
	result := &ValidationResult{Valid: true, Errors: make([]ValidationError, 0)}
	fieldPrefix := fmt.Sprintf("llms[%d]", index)

	// Validate name
	if llm.Name == "" {
		result.addError(fieldPrefix+".name", "LLM name cannot be empty")
	}

	// Validate endpoint
	if llm.Endpoint == "" {
		result.addError(fieldPrefix+".endpoint", "endpoint cannot be empty")
	} else {
		if parsedURL, err := url.Parse(llm.Endpoint); err != nil {
			result.addError(fieldPrefix+".endpoint", "invalid URL format")
		} else if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			result.addError(fieldPrefix+".endpoint", "endpoint must use http or https scheme")
		}
	}

	// Validate API key (optional but should be present for most providers)
	if llm.APIKey == "" && !isWellKnownProvider(llm.Endpoint) {
		result.addError(fieldPrefix+".api_key", "API key is required for custom endpoints")
	}

	return result
}

// validateDatabaseConfig validates database configuration
func validateDatabaseConfig(db *DatabaseConfig) *ValidationResult {
	result := &ValidationResult{Valid: true, Errors: make([]ValidationError, 0)}

	// Validate path
	if db.Path == "" {
		result.addError("database.path", "database path cannot be empty")
	}

	return result
}

// validateAPIConfig validates API configuration
func validateAPIConfig(api *APIConfig) *ValidationResult {
	result := &ValidationResult{Valid: true, Errors: make([]ValidationError, 0)}

	// Validate port
	if api.Port == "" {
		result.addError("api.port", "API port cannot be empty")
	}

	// Validate JWT secret
	if api.JWTSecret == "" {
		result.addError("api.jwt_secret", "JWT secret cannot be empty")
	} else if len(api.JWTSecret) < 32 {
		result.addError("api.jwt_secret", "JWT secret should be at least 32 characters long")
	}

	// Validate rate limits
	if api.RateLimit <= 0 {
		result.addError("api.rate_limit", "rate limit must be greater than 0")
	}

	return result
}

// validateMonitoringConfig validates monitoring configuration
func validateMonitoringConfig(monitoring *MonitoringConfig) *ValidationResult {
	result := &ValidationResult{Valid: true, Errors: make([]ValidationError, 0)}

	// Validate Prometheus port
	if monitoring.EnableMetrics && monitoring.MetricsPort == "" {
		result.addError("monitoring.metrics_port", "metrics port required when metrics are enabled")
	}

	// Validate health port
	if monitoring.EnableHealth && monitoring.HealthPort == "" {
		result.addError("monitoring.health_port", "health port required when health checks are enabled")
	}

	return result
}

// isWellKnownProvider checks if the endpoint is for a well-known provider that might not need an API key
func isWellKnownProvider(endpoint string) bool {
	wellKnownProviders := []string{
		"openai.com",
		"anthropic.com",
		"googleapis.com",
		"replicate.com",
		"huggingface.co",
	}

	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return false
	}

	for _, provider := range wellKnownProviders {
		if strings.Contains(parsedURL.Host, provider) {
			return true
		}
	}

	return false
}

// addError adds a validation error to the result
func (vr *ValidationResult) addError(field, message string) {
	vr.Errors = append(vr.Errors, ValidationError{
		Field:   field,
		Message: message,
	})
	vr.Valid = false
}

// merge merges another validation result into this one
func (vr *ValidationResult) merge(other *ValidationResult) {
	if !other.Valid {
		vr.Valid = false
		vr.Errors = append(vr.Errors, other.Errors...)
	}
}

// Error returns a string representation of all validation errors
func (vr *ValidationResult) Error() string {
	if vr.Valid {
		return ""
	}

	var messages []string
	for _, err := range vr.Errors {
		messages = append(messages, err.Error())
	}

	return fmt.Sprintf("configuration validation failed:\n%s", strings.Join(messages, "\n"))
}

// ValidateAndFixConfig validates the configuration and attempts to fix common issues
func ValidateAndFixConfig(cfg *Config) *ValidationResult {
	result := ValidateConfig(cfg)

	// Attempt to fix some common issues
	for _, err := range result.Errors {
		switch err.Field {
		case "global.default_model":
			if cfg.Global.DefaultModel == "" {
				cfg.Global.DefaultModel = "gpt-3.5-turbo"
				fmt.Printf("Fixed: Set default model to 'gpt-3.5-turbo'\n")
			}
		case "api.port":
			if cfg.API.Port == "" {
				cfg.API.Port = "8080"
				fmt.Printf("Fixed: Set API port to '8080'\n")
			}
		case "database.path":
			if cfg.Database.Path == "" {
				cfg.Database.Path = "./llm-verifier.db"
				fmt.Printf("Fixed: Set database path to './llm-verifier.db'\n")
			}
		}
	}

	// Re-validate after fixes
	if len(result.Errors) > 0 {
		return ValidateConfig(cfg)
	}

	return result
}
