package config

import (
	"crypto/rand"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (ve ValidationError) Error() string {
	return trData("config.validation.error_for_field", map[string]any{
		"field":   ve.Field,
		"message": ve.Message,
	})
}

// generateSecureJWTSecret generates a cryptographically secure JWT secret
func generateSecureJWTSecret() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	secretLength := 32

	bytes := make([]byte, secretLength)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate secure random bytes: %w", err)
	}

	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}

	return string(bytes), nil
}

// setDefaults sets default values for configuration
func setDefaults(cfg *Config) {
	// Set default timeout if not specified
	if cfg.Global.Timeout <= 0 {
		cfg.Global.Timeout = 30 * time.Second
	}

	// Set default top-level timeout if not specified
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}

	// Set default retry count
	if cfg.Global.MaxRetries <= 0 {
		cfg.Global.MaxRetries = 3
	}

	// Set default model if not specified
	if cfg.Global.DefaultModel == "" {
		cfg.Global.DefaultModel = "gpt-3.5-turbo"
	}

	// Set default concurrency
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 10
	}

	// Set default headers for LLM configs
	for i := range cfg.LLMs {
		if cfg.LLMs[i].Headers == nil {
			cfg.LLMs[i].Headers = make(map[string]string)
		}
		// Add default User-Agent if not present (check both cases)
		if _, exists := cfg.LLMs[i].Headers["User-Agent"]; !exists {
			if _, existsLower := cfg.LLMs[i].Headers["user-agent"]; !existsLower {
				cfg.LLMs[i].Headers["User-Agent"] = "LLM-Verifier/1.0"
			}
		}
	}

	// Set default API config
	if cfg.API.Port == "" {
		cfg.API.Port = "8080"
	}
	// Only set default JWT secret if none provided (validation will catch short secrets)
	if cfg.API.JWTSecret == "" {
		// Check if JWT secret is provided via environment variable
		if envSecret := os.Getenv("LLM_VERIFIER_JWT_SECRET"); envSecret != "" {
			cfg.API.JWTSecret = envSecret
		} else {
			// Generate a secure random secret as fallback
			secret, err := generateSecureJWTSecret()
			if err != nil {
				// Log error but continue with a fallback
				cfg.API.JWTSecret = "emergency-fallback-" + fmt.Sprintf("%d", time.Now().Unix())
			} else {
				cfg.API.JWTSecret = secret
			}
		}
	}

	// Set default logging config
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "json"
	}
	if cfg.Logging.Output == "" {
		cfg.Logging.Output = "stdout"
	}
}

// ValidationResult contains the result of configuration validation
type ValidationResult struct {
	Valid  bool
	Errors []ValidationError
}

// ValidateConfig validates the entire configuration
func ValidateConfig(cfg *Config) *ValidationResult {
	// Set defaults first
	setDefaults(cfg)

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

	// Validate logging config
	result.merge(validateLoggingConfig(&cfg.Logging))

	return result
}

// validateGlobalConfig validates global configuration
func validateGlobalConfig(global *GlobalConfig) *ValidationResult {
	result := &ValidationResult{Valid: true, Errors: make([]ValidationError, 0)}

	// Validate BaseURL
	if global.BaseURL != "" {
		if parsedURL, err := url.Parse(global.BaseURL); err != nil {
			result.addError("global.base_url", tr("config.validation.invalid_url_format"))
		} else if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			result.addError("global.base_url", tr("config.validation.url_scheme_required"))
		}
	}

	// Validate DefaultModel
	if global.DefaultModel == "" {
		result.addError("global.default_model", tr("config.validation.default_model_empty"))
	}

	// Validate timeout
	if global.Timeout <= 0 {
		result.addError("global.timeout", tr("config.validation.timeout_positive"))
	}

	return result
}

// validateLLMConfig validates a single LLM configuration
func validateLLMConfig(llm LLMConfig, index int) *ValidationResult {
	result := &ValidationResult{Valid: true, Errors: make([]ValidationError, 0)}
	fieldPrefix := fmt.Sprintf("llms[%d]", index)

	// Validate name
	if llm.Name == "" {
		result.addError(fieldPrefix+".name", tr("config.validation.llm_name_empty"))
	}

	// Validate endpoint
	if llm.Endpoint == "" {
		result.addError(fieldPrefix+".endpoint", tr("config.validation.endpoint_empty"))
	} else {
		if parsedURL, err := url.Parse(llm.Endpoint); err != nil {
			result.addError(fieldPrefix+".endpoint", tr("config.validation.invalid_url_format"))
		} else if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			result.addError(fieldPrefix+".endpoint", tr("config.validation.endpoint_scheme_required"))
		}
	}

	// Validate API key (optional but should be present for most providers)
	if llm.APIKey == "" && !isWellKnownProvider(llm.Endpoint) {
		result.addError(fieldPrefix+".api_key", tr("config.validation.api_key_required"))
	}

	return result
}

// validateDatabaseConfig validates database configuration
func validateDatabaseConfig(db *DatabaseConfig) *ValidationResult {
	result := &ValidationResult{Valid: true, Errors: make([]ValidationError, 0)}

	// Validate path
	if db.Path == "" {
		result.addError("database.path", tr("config.validation.database_path_empty"))
	}

	return result
}

// validateAPIConfig validates API configuration
func validateAPIConfig(api *APIConfig) *ValidationResult {
	result := &ValidationResult{Valid: true, Errors: make([]ValidationError, 0)}

	// Validate port
	if api.Port == "" {
		result.addError("api.port", tr("config.validation.api_port_empty"))
	} else {
		// Parse and validate port number
		portNum, err := strconv.Atoi(api.Port)
		if err != nil {
			result.addError("api.port", tr("config.validation.port_not_a_number"))
		} else if portNum < 1 || portNum > 65535 {
			result.addError("api.port", tr("config.validation.port_out_of_range"))
		}
	}

	// Validate JWT secret
	if api.JWTSecret == "" {
		result.addError("api.jwt_secret", tr("config.validation.jwt_secret_empty"))
	} else if len(api.JWTSecret) < 16 {
		result.addError("api.jwt_secret", tr("config.validation.jwt_secret_too_short"))
	}

	// Validate rate limits
	if api.RateLimit < 0 {
		result.addError("api.rate_limit", tr("config.validation.rate_limit_negative"))
	}

	return result
}

// validateMonitoringConfig validates monitoring configuration
func validateMonitoringConfig(monitoring *MonitoringConfig) *ValidationResult {
	result := &ValidationResult{Valid: true, Errors: make([]ValidationError, 0)}

	// Validate Prometheus port
	if monitoring.EnableMetrics && monitoring.MetricsPort == "" {
		result.addError("monitoring.metrics_port", tr("config.validation.metrics_port_required"))
	}

	// Validate health port
	if monitoring.EnableHealth && monitoring.HealthPort == "" {
		result.addError("monitoring.health_port", tr("config.validation.health_port_required"))
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

	return trData("config.validation.failed_summary", map[string]any{
		"messages": strings.Join(messages, "\n"),
	})
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
				fmt.Println(trData("config.validation.fixed_default_model", map[string]any{"value": "gpt-3.5-turbo"}))
			}
		case "api.port":
			if cfg.API.Port == "" {
				cfg.API.Port = "8080"
				fmt.Println(trData("config.validation.fixed_api_port", map[string]any{"value": "8080"}))
			}
		case "database.path":
			if cfg.Database.Path == "" {
				cfg.Database.Path = "./llm-verifier.db"
				fmt.Println(trData("config.validation.fixed_database_path", map[string]any{"value": "./llm-verifier.db"}))
			}
		}
	}

	// Re-validate after fixes
	if len(result.Errors) > 0 {
		return ValidateConfig(cfg)
	}

	return result
}

// validateLoggingConfig validates logging configuration
func validateLoggingConfig(logging *LoggingConfig) *ValidationResult {
	result := &ValidationResult{Valid: true, Errors: make([]ValidationError, 0)}

	// Only validate if fields are set
	if logging.Level != "" {
		validLevels := []string{"debug", "info", "warn", "error"}
		if !contains(validLevels, logging.Level) {
			result.addError("logging.level", tr("config.validation.log_level_invalid"))
		}
	}

	if logging.Format != "" {
		validFormats := []string{"json", "text"}
		if !contains(validFormats, logging.Format) {
			result.addError("logging.format", tr("config.validation.log_format_invalid"))
		}
	}

	if logging.Output != "" {
		validOutputs := []string{"stdout", "stderr", "file"}
		if !contains(validOutputs, logging.Output) {
			result.addError("logging.output", tr("config.validation.log_output_invalid"))
		}
	}

	if logging.Output == "file" && logging.FilePath == "" {
		result.addError("logging.file_path", tr("config.validation.log_file_path_required"))
	}

	if logging.MaxSize < 0 {
		result.addError("logging.max_size", tr("config.validation.log_max_size_positive"))
	}

	if logging.MaxAge < 0 {
		result.addError("logging.max_age", tr("config.validation.log_max_age_non_negative"))
	}

	return result
}
