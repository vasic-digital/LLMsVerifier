package config

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ValidateLLMConfig validates a single LLM configuration
func ValidateLLMConfig(cfg *LLMConfig) error {
	if cfg.Name == "" {
		return fmt.Errorf("name is required")
	}

	if cfg.Endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}

	// Validate endpoint URL
	if _, err := url.ParseRequestURI(cfg.Endpoint); err != nil {
		return fmt.Errorf("endpoint must be a valid URL: %w", err)
	}

	return nil
}

// ValidateGlobalConfig validates global configuration
func ValidateGlobalConfig(cfg *GlobalConfig) error {
	if cfg.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}

	// Validate base URL
	if _, err := url.ParseRequestURI(cfg.BaseURL); err != nil {
		return fmt.Errorf("base_url must be a valid URL: %w", err)
	}

	if cfg.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be non-negative")
	}

	if cfg.MaxRetries > 10 {
		return fmt.Errorf("max_retries must be less than or equal to 10")
	}

	if cfg.RequestDelay < 0 {
		return fmt.Errorf("request_delay must be non-negative")
	}

	if cfg.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	if cfg.Timeout < 1000000000 { // Less than 1 second
		return fmt.Errorf("timeout must be at least 1 second")
	}

	return nil
}

// ValidateDatabaseConfig validates database configuration
func ValidateDatabaseConfig(cfg *DatabaseConfig) error {
	if cfg.Path == "" {
		return fmt.Errorf("path is required")
	}

	// Check for invalid characters in path
	if strings.ContainsAny(cfg.Path, "\x00") {
		return fmt.Errorf("path contains invalid characters")
	}

	return nil
}

// ValidateAPIConfig validates API configuration
func ValidateAPIConfig(cfg *APIConfig) error {
	if cfg.Port == "" {
		return fmt.Errorf("port is required")
	}

	// Validate port number
	port, err := strconv.Atoi(cfg.Port)
	if err != nil {
		return fmt.Errorf("port must be a valid number: %w", err)
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	if cfg.JWTSecret == "" {
		return fmt.Errorf("jwt_secret is required")
	}

	if len(cfg.JWTSecret) < 16 {
		return fmt.Errorf("jwt_secret must be at least 16 characters")
	}

	if cfg.RateLimit < 0 {
		return fmt.Errorf("rate_limit must be non-negative")
	}

	if cfg.EnableHTTPS {
		if cfg.TLSCertFile == "" || cfg.TLSKeyFile == "" {
			return fmt.Errorf("tls_cert_file and tls_key_file are required when enable_https is true")
		}
	}

	return nil
}

// ValidateLoggingConfig validates logging configuration
func ValidateLoggingConfig(cfg *LoggingConfig) error {
	validLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLevels, cfg.Level) {
		return fmt.Errorf("level must be one of: %s", strings.Join(validLevels, ", "))
	}

	validFormats := []string{"json", "text"}
	if !contains(validFormats, cfg.Format) {
		return fmt.Errorf("format must be one of: %s", strings.Join(validFormats, ", "))
	}

	validOutputs := []string{"stdout", "stderr", "file"}
	if !contains(validOutputs, cfg.Output) {
		return fmt.Errorf("output must be one of: %s", strings.Join(validOutputs, ", "))
	}

	if cfg.Output == "file" && cfg.FilePath == "" {
		return fmt.Errorf("file_path is required when output is file")
	}

	if cfg.MaxSize < 0 {
		return fmt.Errorf("max_size must be positive")
	}

	if cfg.MaxAge < 0 {
		return fmt.Errorf("max_age must be non-negative")
	}

	return nil
}

// ValidateCompleteConfig validates the complete configuration
func ValidateCompleteConfig(cfg *Config) error {
	// Set defaults first
	setDefaults(cfg)

	// Validate global config
	if err := ValidateGlobalConfig(&cfg.Global); err != nil {
		return fmt.Errorf("global.%s", err.Error())
	}

	// Validate database config
	if err := ValidateDatabaseConfig(&cfg.Database); err != nil {
		return fmt.Errorf("database.%s", err.Error())
	}

	// Validate API config
	if err := ValidateAPIConfig(&cfg.API); err != nil {
		return fmt.Errorf("api.%s", err.Error())
	}

	// Validate logging config if set
	if cfg.Logging.Level != "" || cfg.Logging.Format != "" {
		if err := ValidateLoggingConfig(&cfg.Logging); err != nil {
			return fmt.Errorf("logging.%s", err.Error())
		}
	}

	// Validate LLM configs
	for i, llm := range cfg.LLMs {
		if err := ValidateLLMConfig(&llm); err != nil {
			return fmt.Errorf("llms[%d].%s", i, err.Error())
		}
	}

	// Validate general settings
	if cfg.Concurrency < 0 {
		return fmt.Errorf("concurrency must be positive")
	}

	if cfg.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	return nil
}

// MergeConfigs merges two configurations, with overlay taking precedence
func MergeConfigs(base, overlay Config) Config {
	result := base

	// Overlay non-empty values
	if overlay.Profile != "" {
		result.Profile = overlay.Profile
	}

	if overlay.Concurrency != 0 {
		result.Concurrency = overlay.Concurrency
	}

	if overlay.Timeout != 0 {
		result.Timeout = overlay.Timeout
	}

	// Merge global config
	if overlay.Global.BaseURL != "" {
		result.Global.BaseURL = overlay.Global.BaseURL
	}
	if overlay.Global.APIKey != "" {
		result.Global.APIKey = overlay.Global.APIKey
	}
	if overlay.Global.DefaultModel != "" {
		result.Global.DefaultModel = overlay.Global.DefaultModel
	}
	if overlay.Global.MaxRetries != 0 {
		result.Global.MaxRetries = overlay.Global.MaxRetries
	}
	if overlay.Global.RequestDelay != 0 {
		result.Global.RequestDelay = overlay.Global.RequestDelay
	}
	if overlay.Global.Timeout != 0 {
		result.Global.Timeout = overlay.Global.Timeout
	}

	// Merge database config
	if overlay.Database.Path != "" {
		result.Database.Path = overlay.Database.Path
	}
	if overlay.Database.EncryptionKey != "" {
		result.Database.EncryptionKey = overlay.Database.EncryptionKey
	}

	// Merge API config
	if overlay.API.Port != "" {
		result.API.Port = overlay.API.Port
	}
	if overlay.API.JWTSecret != "" {
		result.API.JWTSecret = overlay.API.JWTSecret
	}
	if overlay.API.RateLimit != 0 {
		result.API.RateLimit = overlay.API.RateLimit
	}
	if overlay.API.BurstLimit != 0 {
		result.API.BurstLimit = overlay.API.BurstLimit
	}
	if overlay.API.RateLimitWindow != 0 {
		result.API.RateLimitWindow = overlay.API.RateLimitWindow
	}

	// Replace LLMs if provided
	if len(overlay.LLMs) > 0 {
		result.LLMs = overlay.LLMs
	}

	return result
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
