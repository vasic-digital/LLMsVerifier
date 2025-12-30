package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValidateLLMConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  LLMConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid LLM config",
			config: LLMConfig{
				Name:     "Test LLM",
				Endpoint: "https://api.example.com",
				APIKey:   "test-key",
				Model:    "gpt-4",
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "Missing name",
			config: LLMConfig{
				Endpoint: "https://api.example.com",
				APIKey:   "test-key",
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "Empty name",
			config: LLMConfig{
				Name:     "",
				Endpoint: "https://api.example.com",
				APIKey:   "test-key",
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "Missing endpoint",
			config: LLMConfig{
				Name:   "Test LLM",
				APIKey: "test-key",
			},
			wantErr: true,
			errMsg:  "endpoint is required",
		},
		{
			name: "Empty endpoint",
			config: LLMConfig{
				Name:     "Test LLM",
				Endpoint: "",
				APIKey:   "test-key",
			},
			wantErr: true,
			errMsg:  "endpoint is required",
		},
		{
			name: "Invalid endpoint URL",
			config: LLMConfig{
				Name:     "Test LLM",
				Endpoint: "not-a-url",
				APIKey:   "test-key",
			},
			wantErr: true,
			errMsg:  "endpoint must be a valid URL",
		},
		{
			name: "Valid with all fields",
			config: LLMConfig{
				Name:     "Complete LLM",
				Endpoint: "https://api.complete.com",
				APIKey:   "complete-key",
				Model:    "claude-3-sonnet",
				Headers: map[string]string{
					"X-Custom": "value",
					"User-Agent": "LLM-Verifier/1.0",
				},
				Features: map[string]bool{
					"code_generation": true,
					"multimodal":      true,
					"streaming":      true,
				},
			},
			wantErr: false,
			errMsg:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLLMConfig(&tt.config)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateGlobalConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  GlobalConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid global config",
			config: GlobalConfig{
				BaseURL:      "https://api.example.com",
				APIKey:       "test-key",
				DefaultModel: "gpt-4",
				MaxRetries:   3,
				RequestDelay: 1 * time.Second,
				Timeout:      30 * time.Second,
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "Missing base URL",
			config: GlobalConfig{
				APIKey: "test-key",
			},
			wantErr: true,
			errMsg:  "base_url is required",
		},
		{
			name: "Invalid base URL",
			config: GlobalConfig{
				BaseURL: "not-a-url",
				APIKey:  "test-key",
			},
			wantErr: true,
			errMsg:  "base_url must be a valid URL",
		},
		{
			name: "Negative max retries",
			config: GlobalConfig{
				BaseURL:    "https://api.example.com",
				APIKey:     "test-key",
				MaxRetries: -1,
			},
			wantErr: true,
			errMsg:  "max_retries must be non-negative",
		},
		{
			name: "Too many max retries",
			config: GlobalConfig{
				BaseURL:    "https://api.example.com",
				APIKey:     "test-key",
				MaxRetries: 100,
			},
			wantErr: true,
			errMsg:  "max_retries must be less than or equal to 10",
		},
		{
			name: "Negative request delay",
			config: GlobalConfig{
				BaseURL:      "https://api.example.com",
				APIKey:       "test-key",
				RequestDelay: -1 * time.Second,
			},
			wantErr: true,
			errMsg:  "request_delay must be non-negative",
		},
		{
			name: "Zero timeout",
			config: GlobalConfig{
				BaseURL: "https://api.example.com",
				APIKey:  "test-key",
				Timeout: 0,
			},
			wantErr: true,
			errMsg:  "timeout must be positive",
		},
		{
			name: "Too short timeout",
			config: GlobalConfig{
				BaseURL: "https://api.example.com",
				APIKey:  "test-key",
				Timeout: 100 * time.Millisecond,
			},
			wantErr: true,
			errMsg:  "timeout must be at least 1 second",
		},
		{
			name: "Valid with all fields",
			config: GlobalConfig{
				BaseURL:      "https://api.complete.com",
				APIKey:       "complete-key",
				DefaultModel: "claude-3-opus",
				MaxRetries:   5,
				RequestDelay: 2 * time.Second,
				Timeout:      60 * time.Second,
				CustomParams: map[string]any{
					"temperature": 0.7,
					"max_tokens":   4096,
				},
			},
			wantErr: false,
			errMsg:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGlobalConfig(&tt.config)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateDatabaseConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  DatabaseConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid database config",
			config: DatabaseConfig{
				Path:          "./test.db",
				EncryptionKey: "test-encryption-key",
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "Missing path",
			config: DatabaseConfig{
				EncryptionKey: "test-encryption-key",
			},
			wantErr: true,
			errMsg:  "path is required",
		},
		{
			name: "Empty path",
			config: DatabaseConfig{
				Path:          "",
				EncryptionKey: "test-encryption-key",
			},
			wantErr: true,
			errMsg:  "path is required",
		},
		{
			name: "In-memory database",
			config: DatabaseConfig{
				Path: ":memory:",
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "Invalid database path",
			config: DatabaseConfig{
				Path:          "invalid/path/\x00/with/null",
				EncryptionKey: "test-encryption-key",
			},
			wantErr: true,
			errMsg:  "path contains invalid characters",
		},
		{
			name: "Valid with absolute path",
			config: DatabaseConfig{
				Path:          "/data/llm-verifier.db",
				EncryptionKey: "absolute-key",
			},
			wantErr: false,
			errMsg:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDatabaseConfig(&tt.config)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAPIConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  APIConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid API config",
			config: APIConfig{
				Port:       "8080",
				JWTSecret:  "test-jwt-secret",
				RateLimit:  100,
				BurstLimit: 50,
				EnableCORS: true,
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "Missing port",
			config: APIConfig{
				JWTSecret: "test-jwt-secret",
			},
			wantErr: true,
			errMsg:  "port is required",
		},
		{
			name: "Invalid port number",
			config: APIConfig{
				Port:      "invalid",
				JWTSecret: "test-jwt-secret",
			},
			wantErr: true,
			errMsg:  "port must be a valid number",
		},
		{
			name: "Port out of range",
			config: APIConfig{
				Port:      "99999",
				JWTSecret: "test-jwt-secret",
			},
			wantErr: true,
			errMsg:  "port must be between 1 and 65535",
		},
		{
			name: "Privileged port",
			config: APIConfig{
				Port:      "80",
				JWTSecret: "test-jwt-secret",
			},
			wantErr: false, // Allow privileged ports for development
			errMsg:  "",
		},
		{
			name: "Missing JWT secret",
			config: APIConfig{
				Port: "8080",
			},
			wantErr: true,
			errMsg:  "jwt_secret is required",
		},
		{
			name: "Short JWT secret",
			config: APIConfig{
				Port:      "8080",
				JWTSecret: "short",
			},
			wantErr: true,
			errMsg:  "jwt_secret must be at least 16 characters",
		},
		{
			name: "Negative rate limit",
			config: APIConfig{
				Port:       "8080",
				JWTSecret:  "test-jwt-secret",
				RateLimit:  -1,
			},
			wantErr: true,
			errMsg:  "rate_limit must be non-negative",
		},
		{
			name: "Zero rate limit",
			config: APIConfig{
				Port:       "8080",
				JWTSecret:  "test-jwt-secret",
				RateLimit:  0,
			},
			wantErr: false, // Allow zero rate limit (no rate limiting)
			errMsg:  "",
		},
		{
			name: "Invalid TLS configuration",
			config: APIConfig{
				Port:         "8080",
				JWTSecret:    "test-jwt-secret",
				EnableHTTPS:   true,
				TLSCertFile:  "",
				TLSKeyFile:   "",
			},
			wantErr: true,
			errMsg:  "tls_cert_file and tls_key_file are required when enable_https is true",
		},
		{
			name: "Valid TLS configuration",
			config: APIConfig{
				Port:        "8443",
				JWTSecret:   "test-jwt-secret",
				EnableHTTPS: true,
				TLSCertFile: "/certs/server.crt",
				TLSKeyFile:  "/certs/server.key",
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "Valid with all options",
			config: APIConfig{
				Port:              "8080",
				JWTSecret:         "very-secure-jwt-secret-key",
				RateLimit:         1000,
				BurstLimit:        200,
				RateLimitWindow:   60,
				EnableCORS:       true,
				TrustedProxies:    "127.0.0.1,192.168.1.1",
				CORSOrigins:       "https://example.com,https://test.com",
				EnableHTTPS:       false,
				ReadTimeout:       30,
				WriteTimeout:      30,
				MaxHeaderBytes:    1048576,
			},
			wantErr: false,
			errMsg:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAPIConfig(&tt.config)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateLoggingConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  LoggingConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid logging config",
			config: LoggingConfig{
				Level:      "info",
				Format:     "json",
				Output:     "stdout",
				MaxSize:    100,
				MaxBackups: 5,
				MaxAge:     30,
				Compress:   true,
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "Invalid log level",
			config: LoggingConfig{
				Level: "invalid",
			},
			wantErr: true,
			errMsg:  "level must be one of: debug, info, warn, error",
		},
		{
			name: "Invalid log format",
			config: LoggingConfig{
				Level:  "info",
				Format: "invalid",
			},
			wantErr: true,
			errMsg:  "format must be one of: json, text",
		},
		{
			name: "Invalid log output",
			config: LoggingConfig{
				Level:  "info",
				Format: "json",
				Output: "invalid",
			},
			wantErr: true,
			errMsg:  "output must be one of: stdout, stderr, file",
		},
		{
			name: "File output without file path",
			config: LoggingConfig{
				Level:  "info",
				Format: "json",
				Output: "file",
			},
			wantErr: true,
			errMsg:  "file_path is required when output is file",
		},
		{
			name: "Valid file output",
			config: LoggingConfig{
				Level:    "debug",
				Format:   "json",
				Output:   "file",
				FilePath: "./logs/app.log",
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "Negative max size",
			config: LoggingConfig{
				Level:   "info",
				MaxSize: -1,
			},
			wantErr: true,
			errMsg:  "max_size must be positive",
		},
		{
			name: "Negative max age",
			config: LoggingConfig{
				Level:  "info",
				MaxAge: -1,
			},
			wantErr: true,
			errMsg:  "max_age must be non-negative",
		},
		{
			name: "Valid with all options",
			config: LoggingConfig{
				Level:      "debug",
				Format:     "json",
				Output:     "file",
				FilePath:   "/var/log/llm-verifier/app.log",
				MaxSize:    500,
				MaxBackups: 10,
				MaxAge:     90,
				Compress:   true,
			},
			wantErr: false,
			errMsg:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLoggingConfig(&tt.config)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCompleteConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid complete config",
			config: Config{
				Profile:    "production",
				Concurrency: 10,
				Timeout:    60 * time.Second,
				Global: GlobalConfig{
					BaseURL:      "https://api.example.com",
					APIKey:       "test-key",
					DefaultModel: "gpt-4",
				},
				Database: DatabaseConfig{
					Path: "./production.db",
				},
				API: APIConfig{
					Port:      "8080",
					JWTSecret: "secure-jwt-secret",
				},
				LLMs: []LLMConfig{
					{
						Name:     "Test LLM",
						Endpoint: "https://api.test.com",
						APIKey:   "test-llm-key",
					},
				},
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "Config with empty LLMs",
			config: Config{
				Timeout: 30 * time.Second,
				Global: GlobalConfig{
					BaseURL: "https://api.example.com",
					APIKey:  "test-key",
				},
				Database: DatabaseConfig{
					Path: "./test.db",
				},
				LLMs: []LLMConfig{},
			},
			wantErr: false, // Empty LLMs is allowed (will discover models)
			errMsg:  "",
		},
		{
			name: "Config with invalid LLM",
			config: Config{
				Global: GlobalConfig{
					BaseURL: "https://api.example.com",
					APIKey:  "test-key",
				},
				Database: DatabaseConfig{
					Path: "./test.db",
				},
				LLMs: []LLMConfig{
					{
						Name: "Invalid LLM", // Missing endpoint and API key
					},
				},
			},
			wantErr: true,
			errMsg:  "endpoint is required",
		},
		{
			name: "Config with negative concurrency",
			config: Config{
				Concurrency: -1,
				Global: GlobalConfig{
					BaseURL: "https://api.example.com",
					APIKey:  "test-key",
				},
				Database: DatabaseConfig{
					Path: "./test.db",
				},
			},
			wantErr: true,
			errMsg:  "concurrency must be positive",
		},
		{
			name: "Config with zero timeout",
			config: Config{
				Timeout: -1,
				Global: GlobalConfig{
					BaseURL: "https://api.example.com",
					APIKey:  "test-key",
				},
				Database: DatabaseConfig{
					Path: "./test.db",
				},
			},
			wantErr: true,
			errMsg:  "timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCompleteConfig(&tt.config)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigMerge(t *testing.T) {
	// Test merging configs
	defaultConfig := Config{
		Global: GlobalConfig{
			BaseURL:      "https://default.api.com",
			DefaultModel: "gpt-3.5-turbo",
			MaxRetries:   3,
		},
		Database: DatabaseConfig{
			Path: "./default.db",
		},
		API: APIConfig{
			Port:      "8080",
			RateLimit: 100,
		},
	}

	overlayConfig := Config{
		Global: GlobalConfig{
			BaseURL: "https://override.api.com", // Override
			APIKey:  "override-key",          // New field
		},
		Database: DatabaseConfig{
			EncryptionKey: "override-key", // Add to existing
		},
		API: APIConfig{
			Port: "9090", // Override
		},
	}

	merged := MergeConfigs(defaultConfig, overlayConfig)

	assert.Equal(t, "https://override.api.com", merged.Global.BaseURL)
	assert.Equal(t, "gpt-3.5-turbo", merged.Global.DefaultModel) // Kept from default
	assert.Equal(t, 3, merged.Global.MaxRetries)                  // Kept from default
	assert.Equal(t, "override-key", merged.Global.APIKey)           // Added from overlay

	assert.Equal(t, "./default.db", merged.Database.Path)           // Kept from default
	assert.Equal(t, "override-key", merged.Database.EncryptionKey)   // Added from overlay

	assert.Equal(t, "9090", merged.API.Port)     // Overridden
	assert.Equal(t, 100, merged.API.RateLimit) // Kept from default
}

// Test ValidationError
func TestValidationError(t *testing.T) {
	ve := ValidationError{
		Field:   "api.port",
		Message: "port is required",
	}

	errStr := ve.Error()
	assert.Contains(t, errStr, "api.port")
	assert.Contains(t, errStr, "port is required")
}

// Test ValidateConfig (the function that takes *Config)
func TestValidateConfigFunc(t *testing.T) {
	t.Run("valid config with defaults", func(t *testing.T) {
		cfg := &Config{
			Database: DatabaseConfig{
				Path: "./test.db",
			},
			LLMs: []LLMConfig{
				{
					Name:     "Test LLM",
					Endpoint: "https://api.openai.com/v1",
					APIKey:   "test-key",
				},
			},
		}

		result := ValidateConfig(cfg)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)

		// Check that defaults were set
		assert.Equal(t, "gpt-3.5-turbo", cfg.Global.DefaultModel)
		assert.Equal(t, "8080", cfg.API.Port)
	})

	t.Run("config with validation errors", func(t *testing.T) {
		cfg := &Config{
			Database: DatabaseConfig{
				Path: "", // Invalid - empty path
			},
			API: APIConfig{
				Port:      "invalid",
				JWTSecret: "short", // Too short
			},
		}

		result := ValidateConfig(cfg)
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)
	})

	t.Run("config with invalid LLM endpoint", func(t *testing.T) {
		cfg := &Config{
			Database: DatabaseConfig{
				Path: "./test.db",
			},
			LLMs: []LLMConfig{
				{
					Name:     "Invalid LLM",
					Endpoint: "not-a-url",
					APIKey:   "test-key",
				},
			},
		}

		result := ValidateConfig(cfg)
		assert.False(t, result.Valid)
	})
}

// Test validateGlobalConfig internal function
func TestValidateGlobalConfigInternal(t *testing.T) {
	t.Run("valid global config", func(t *testing.T) {
		cfg := &GlobalConfig{
			DefaultModel: "gpt-4",
			Timeout:      30 * time.Second,
		}
		result := validateGlobalConfig(cfg)
		assert.True(t, result.Valid)
	})

	t.Run("invalid base URL scheme", func(t *testing.T) {
		cfg := &GlobalConfig{
			BaseURL:      "ftp://invalid.com",
			DefaultModel: "gpt-4",
			Timeout:      30 * time.Second,
		}
		result := validateGlobalConfig(cfg)
		assert.False(t, result.Valid)
	})
}

// Test validateLLMConfig internal function
func TestValidateLLMConfigInternal(t *testing.T) {
	t.Run("valid LLM config", func(t *testing.T) {
		cfg := LLMConfig{
			Name:     "Test",
			Endpoint: "https://api.test.com",
			APIKey:   "key",
		}
		result := validateLLMConfig(cfg, 0)
		assert.True(t, result.Valid)
	})

	t.Run("well-known provider without API key", func(t *testing.T) {
		cfg := LLMConfig{
			Name:     "OpenAI",
			Endpoint: "https://api.openai.com/v1",
			APIKey:   "", // Well-known provider allows empty key in config
		}
		result := validateLLMConfig(cfg, 0)
		assert.True(t, result.Valid) // Should pass for well-known provider
	})

	t.Run("custom endpoint without API key", func(t *testing.T) {
		cfg := LLMConfig{
			Name:     "Custom",
			Endpoint: "https://custom.api.com",
			APIKey:   "",
		}
		result := validateLLMConfig(cfg, 0)
		assert.False(t, result.Valid) // Should fail for custom endpoint
	})
}

// Test validateDatabaseConfig internal function
func TestValidateDatabaseConfigInternal(t *testing.T) {
	t.Run("valid database config", func(t *testing.T) {
		cfg := &DatabaseConfig{
			Path: "./test.db",
		}
		result := validateDatabaseConfig(cfg)
		assert.True(t, result.Valid)
	})

	t.Run("empty database path", func(t *testing.T) {
		cfg := &DatabaseConfig{
			Path: "",
		}
		result := validateDatabaseConfig(cfg)
		assert.False(t, result.Valid)
	})
}

// Test validateAPIConfig internal function
func TestValidateAPIConfigInternal(t *testing.T) {
	t.Run("valid API config", func(t *testing.T) {
		cfg := &APIConfig{
			Port:      "8080",
			JWTSecret: "a-very-secure-secret-key",
			RateLimit: 100,
		}
		result := validateAPIConfig(cfg)
		assert.True(t, result.Valid)
	})

	t.Run("invalid port format", func(t *testing.T) {
		cfg := &APIConfig{
			Port:      "not-a-number",
			JWTSecret: "a-very-secure-secret-key",
		}
		result := validateAPIConfig(cfg)
		assert.False(t, result.Valid)
	})

	t.Run("port out of range", func(t *testing.T) {
		cfg := &APIConfig{
			Port:      "99999",
			JWTSecret: "a-very-secure-secret-key",
		}
		result := validateAPIConfig(cfg)
		assert.False(t, result.Valid)
	})
}

// Test validateMonitoringConfig internal function
func TestValidateMonitoringConfigInternal(t *testing.T) {
	t.Run("metrics enabled without port", func(t *testing.T) {
		cfg := &MonitoringConfig{
			EnableMetrics: true,
			MetricsPort:   "",
		}
		result := validateMonitoringConfig(cfg)
		assert.False(t, result.Valid)
	})

	t.Run("health enabled without port", func(t *testing.T) {
		cfg := &MonitoringConfig{
			EnableHealth: true,
			HealthPort:   "",
		}
		result := validateMonitoringConfig(cfg)
		assert.False(t, result.Valid)
	})

	t.Run("valid monitoring config", func(t *testing.T) {
		cfg := &MonitoringConfig{
			EnableMetrics: true,
			MetricsPort:   "9090",
			EnableHealth:  true,
			HealthPort:    "9091",
		}
		result := validateMonitoringConfig(cfg)
		assert.True(t, result.Valid)
	})
}

// Test isWellKnownProvider
func TestIsWellKnownProvider(t *testing.T) {
	tests := []struct {
		endpoint string
		expected bool
	}{
		{"https://api.openai.com/v1", true},
		{"https://api.anthropic.com/v1", true},
		{"https://generativelanguage.googleapis.com/v1", true},
		{"https://api.replicate.com/v1", true},
		{"https://api-inference.huggingface.co/models", true},
		{"https://custom.api.com", false},
		{"https://my-local-llm.com", false},
		{"not-a-url", false},
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			result := isWellKnownProvider(tt.endpoint)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test ValidationResult methods
func TestValidationResult(t *testing.T) {
	t.Run("addError", func(t *testing.T) {
		result := &ValidationResult{Valid: true, Errors: []ValidationError{}}
		result.addError("field1", "error1")

		assert.False(t, result.Valid)
		assert.Len(t, result.Errors, 1)
		assert.Equal(t, "field1", result.Errors[0].Field)
	})

	t.Run("merge", func(t *testing.T) {
		result1 := &ValidationResult{Valid: true, Errors: []ValidationError{}}
		result2 := &ValidationResult{
			Valid: false,
			Errors: []ValidationError{
				{Field: "field1", Message: "error1"},
			},
		}

		result1.merge(result2)
		assert.False(t, result1.Valid)
		assert.Len(t, result1.Errors, 1)
	})

	t.Run("Error string", func(t *testing.T) {
		result := &ValidationResult{Valid: true}
		assert.Empty(t, result.Error())

		result.addError("field1", "error1")
		errStr := result.Error()
		assert.Contains(t, errStr, "validation failed")
		assert.Contains(t, errStr, "field1")
	})
}

// Test setDefaults
func TestSetDefaults(t *testing.T) {
	cfg := &Config{}
	setDefaults(cfg)

	assert.Equal(t, 30*time.Second, cfg.Global.Timeout)
	assert.Equal(t, 30*time.Second, cfg.Timeout)
	assert.Equal(t, 3, cfg.Global.MaxRetries)
	assert.Equal(t, "gpt-3.5-turbo", cfg.Global.DefaultModel)
	assert.Equal(t, 10, cfg.Concurrency)
	assert.Equal(t, "8080", cfg.API.Port)
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
	assert.Equal(t, "stdout", cfg.Logging.Output)
}

// Test validateLoggingConfig internal function
func TestValidateLoggingConfigInternal(t *testing.T) {
	t.Run("valid logging config", func(t *testing.T) {
		cfg := &LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		}
		result := validateLoggingConfig(cfg)
		assert.True(t, result.Valid)
	})

	t.Run("file output without path", func(t *testing.T) {
		cfg := &LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "file",
		}
		result := validateLoggingConfig(cfg)
		assert.False(t, result.Valid)
	})
}

// Test ValidateAndFixConfig
func TestValidateAndFixConfig(t *testing.T) {
	cfg := &Config{
		Global: GlobalConfig{
			DefaultModel: "", // Will be fixed
		},
		API: APIConfig{
			Port: "", // Will be fixed
		},
		Database: DatabaseConfig{
			Path: "", // Will be fixed
		},
	}

	result := ValidateAndFixConfig(cfg)

	// Should have set defaults
	assert.Equal(t, "gpt-3.5-turbo", cfg.Global.DefaultModel)
	assert.Equal(t, "8080", cfg.API.Port)
	assert.Equal(t, "./llm-verifier.db", cfg.Database.Path)

	// After fixes, should be valid
	assert.True(t, result.Valid)
}