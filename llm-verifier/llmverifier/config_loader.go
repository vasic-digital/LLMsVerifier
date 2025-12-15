package llmverifier

import (
	"crypto/rand"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
	"llm-verifier/config"
)

// LoadConfig loads the configuration from a YAML file with validation and advanced features
func LoadConfig(filePath string) (*config.Config, error) {
	viper.SetConfigFile(filePath)
	viper.AutomaticEnv() // Allow environment variables to override config

	// Set environment variable prefix
	viper.SetEnvPrefix("LLM_VERIFIER")

	// Set default values
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg config.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Expand environment variables in the config
	if err := expandEnvironmentVariables(&cfg); err != nil {
		return nil, fmt.Errorf("error expanding environment variables: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Set computed defaults
	setComputedDefaults(&cfg)

	return &cfg, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Global defaults
	viper.SetDefault("global.max_retries", 3)
	viper.SetDefault("global.request_delay", 1*time.Second)
	viper.SetDefault("global.timeout", 30*time.Second)

	// Database defaults
	viper.SetDefault("database.path", "llm_verifier.db")

	// API defaults
	viper.SetDefault("api.port", "8080")
	viper.SetDefault("api.rate_limit", 100)
	viper.SetDefault("api.burst_limit", 200)
	viper.SetDefault("api.rate_limit_window", 60)
	viper.SetDefault("api.enable_cors", false)

	// Application defaults
	viper.SetDefault("concurrency", 1)
	viper.SetDefault("timeout", 60*time.Second)
}

// expandEnvironmentVariables expands environment variables in configuration
func expandEnvironmentVariables(cfg *config.Config) error {
	// Expand global API key
	cfg.Global.APIKey = expandEnvVar(cfg.Global.APIKey)

	// Expand global base URL
	cfg.Global.BaseURL = expandEnvVar(cfg.Global.BaseURL)

	// Expand database encryption key
	cfg.Database.EncryptionKey = expandEnvVar(cfg.Database.EncryptionKey)

	// Expand API JWT secret
	cfg.API.JWTSecret = expandEnvVar(cfg.API.JWTSecret)

	// Expand LLM configurations
	for i := range cfg.LLMs {
		cfg.LLMs[i].Name = expandEnvVar(cfg.LLMs[i].Name)
		cfg.LLMs[i].Endpoint = expandEnvVar(cfg.LLMs[i].Endpoint)
		cfg.LLMs[i].APIKey = expandEnvVar(cfg.LLMs[i].APIKey)
		cfg.LLMs[i].Model = expandEnvVar(cfg.LLMs[i].Model)

		// Expand headers
		if cfg.LLMs[i].Headers != nil {
			for k, v := range cfg.LLMs[i].Headers {
				cfg.LLMs[i].Headers[k] = expandEnvVar(v)
			}
		}
	}

	return nil
}

// expandEnvVar expands a single environment variable with fallback
func expandEnvVar(value string) string {
	if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
		envVar := strings.TrimSuffix(strings.TrimPrefix(value, "${"), "}")
		if envValue := os.Getenv(envVar); envValue != "" {
			return envValue
		}
		// If environment variable is not set, return the original value
		return value
	}
	return os.ExpandEnv(value)
}

// validateConfig validates the configuration
func validateConfig(cfg *config.Config) error {
	// Validate concurrency
	if cfg.Concurrency < 1 || cfg.Concurrency > 100 {
		return fmt.Errorf("concurrency must be between 1 and 100, got %d", cfg.Concurrency)
	}

	// Validate timeout
	if cfg.Timeout < 1*time.Second || cfg.Timeout > 10*time.Minute {
		return fmt.Errorf("timeout must be between 1s and 10m, got %v", cfg.Timeout)
	}

	// Validate global config
	if err := validateGlobalConfig(&cfg.Global); err != nil {
		return fmt.Errorf("global config validation failed: %w", err)
	}

	// Validate database config
	if err := validateDatabaseConfig(&cfg.Database); err != nil {
		return fmt.Errorf("database config validation failed: %w", err)
	}

	// Validate API config
	if err := validateAPIConfig(&cfg.API); err != nil {
		return fmt.Errorf("API config validation failed: %w", err)
	}

	// Validate LLM configs
	for i, llm := range cfg.LLMs {
		if err := validateLLMConfig(&llm, i); err != nil {
			return fmt.Errorf("LLM config %d validation failed: %w", i, err)
		}
	}

	return nil
}

// validateGlobalConfig validates global configuration
func validateGlobalConfig(global *config.GlobalConfig) error {
	// Validate base URL if provided
	if global.BaseURL != "" && !strings.HasPrefix(global.BaseURL, "http") {
		return fmt.Errorf("base_url must start with http:// or https://")
	}

	// Validate max retries
	if global.MaxRetries < 0 || global.MaxRetries > 10 {
		return fmt.Errorf("max_retries must be between 0 and 10, got %d", global.MaxRetries)
	}

	// Validate request delay
	if global.RequestDelay < 0 || global.RequestDelay > 1*time.Minute {
		return fmt.Errorf("request_delay must be between 0 and 1m, got %v", global.RequestDelay)
	}

	// Validate timeout
	if global.Timeout < 1*time.Second || global.Timeout > 10*time.Minute {
		return fmt.Errorf("timeout must be between 1s and 10m, got %v", global.Timeout)
	}

	return nil
}

// validateDatabaseConfig validates database configuration
func validateDatabaseConfig(db *config.DatabaseConfig) error {
	// Validate database path
	if db.Path == "" {
		return fmt.Errorf("database path cannot be empty")
	}

	// Validate encryption key length if provided
	if db.EncryptionKey != "" && len(db.EncryptionKey) < 16 {
		return fmt.Errorf("encryption key must be at least 16 characters long")
	}

	return nil
}

// validateAPIConfig validates API configuration
func validateAPIConfig(api *config.APIConfig) error {
	// Validate port
	if port, err := strconv.Atoi(api.Port); err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("API port must be between 1 and 65535, got %s", api.Port)
	}

	// Validate rate limit
	if api.RateLimit < 1 || api.RateLimit > 10000 {
		return fmt.Errorf("rate_limit must be between 1 and 10000, got %d", api.RateLimit)
	}

	// Validate burst limit
	if api.BurstLimit < api.RateLimit {
		return fmt.Errorf("burst_limit (%d) must be greater than or equal to rate_limit (%d)", api.BurstLimit, api.RateLimit)
	}

	// Validate rate limit window
	if api.RateLimitWindow < 1 || api.RateLimitWindow > 3600 {
		return fmt.Errorf("rate_limit_window must be between 1 and 3600 seconds, got %d", api.RateLimitWindow)
	}

	return nil
}

// validateLLMConfig validates LLM configuration
func validateLLMConfig(llm *config.LLMConfig, index int) error {
	// Validate name
	if strings.TrimSpace(llm.Name) == "" {
		return fmt.Errorf("LLM name cannot be empty")
	}

	// Validate endpoint
	if llm.Endpoint == "" {
		return fmt.Errorf("LLM endpoint cannot be empty")
	}
	if !strings.HasPrefix(llm.Endpoint, "http") {
		return fmt.Errorf("LLM endpoint must start with http:// or https://")
	}

	// Validate API key (required for most providers)
	if llm.APIKey == "" && !strings.Contains(llm.Endpoint, "localhost") && !strings.Contains(llm.Endpoint, "127.0.0.1") {
		return fmt.Errorf("LLM API key is required for non-local endpoints")
	}

	return nil
}

// setComputedDefaults sets computed default values
func setComputedDefaults(cfg *config.Config) {
	// Set default model if not specified
	if cfg.Global.DefaultModel == "" && len(cfg.LLMs) > 0 {
		cfg.Global.DefaultModel = cfg.LLMs[0].Model
		if cfg.Global.DefaultModel == "" {
			cfg.Global.DefaultModel = cfg.LLMs[0].Name
		}
	}

	// Set default JWT secret if not provided
	if cfg.API.JWTSecret == "" {
		cfg.API.JWTSecret = generateSecureRandomString(32)
	}

	// Set default database encryption key if not provided
	if cfg.Database.EncryptionKey == "" {
		cfg.Database.EncryptionKey = generateSecureRandomString(32)
	}
}

// generateSecureRandomString generates a secure random string for secrets
func generateSecureRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	randomBytes := make([]byte, length)

	// Generate secure random bytes
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to simple generation if crypto/rand fails
		for i := range result {
			result[i] = charset[i%len(charset)]
		}
		return string(result)
	}

	// Map random bytes to charset
	for i := range result {
		result[i] = charset[int(randomBytes[i])%len(charset)]
	}

	return string(result)
}
