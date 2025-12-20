package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_FromFile(t *testing.T) {
	// Create temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	configContent := `
profile: "test"
concurrency: 10
timeout: 90s

global:
  base_url: "https://api.test.com"
  api_key: "test-api-key"
  default_model: "gpt-4"
  max_retries: 5
  request_delay: 2s
  timeout: 45s

database:
  path: "./test.db"
  encryption_key: "test-encryption-key"

api:
  port: "9090"
  jwt_secret: "test-jwt-secret"
  rate_limit: 200
  burst_limit: 50
  rate_limit_window: 60
  enable_cors: false
  trusted_proxies: "127.0.0.1,192.168.1.1"
  cors_origins: "https://example.com,https://test.com"
  enable_https: true
  tls_cert_file: "./cert.pem"
  tls_key_file: "./key.pem"
  read_timeout: 30
  write_timeout: 30
  max_header_bytes: 1048576

logging:
  level: "debug"
  format: "json"
  output: "file"
  file_path: "./logs/test.log"
  max_size: 100
  max_backups: 5
  max_age: 30
  compress: true

monitoring:
  enable_metrics: true
  metrics_port: "9091"
  enable_health: true
  health_port: "8081"
  enable_tracing: true
  tracing_endpoint: "http://jaeger:14268/api/traces"
  enable_profiling: true
  profiling_port: "6060"

security:
  enable_rate_limiting: true
  enable_ip_whitelist: true
  ip_whitelist: ["127.0.0.1", "192.168.1.0/24"]
  enable_request_logging: true
  sensitive_headers: ["authorization", "cookie"]
  enable_csrf_protection: true
  csrf_token_length: 32
  session_timeout: 120

notifications:
  slack:
    enabled: true
    webhook_url: "https://hooks.slack.com/test"
  email:
    enabled: true
    smtp_host: "smtp.example.com"
    smtp_port: 587
    username: "test@example.com"
    password: "test-password"
    default_recipient: "admin@example.com"
  telegram:
    enabled: false
    bot_token: ""
    chat_id: ""
  matrix:
    enabled: false
    homeserver_url: ""
    access_token: ""
    room_id: ""
  whatsapp:
    enabled: false
    api_key: ""
    phone_number_id: ""
    default_recipient: ""

llms:
  - name: "Test LLM 1"
    endpoint: "https://api.test1.com"
    api_key: "test-key-1"
    model: "gpt-3.5-turbo"
    headers:
      X-Custom-Header: "test-value"
    features:
      code_generation: true
      multimodal: false
  - name: "Test LLM 2"
    endpoint: "https://api.test2.com"
    api_key: "test-key-2"
    model: "claude-3"
    features:
      code_generation: false
      multimodal: true
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load config
	viper.SetConfigFile(configPath)
	err = viper.ReadInConfig()
	require.NoError(t, err)

	var cfg Config
	err = viper.Unmarshal(&cfg)
	require.NoError(t, err)

	// Verify basic config
	assert.Equal(t, "test", cfg.Profile)
	assert.Equal(t, 10, cfg.Concurrency)
	assert.Equal(t, 90*time.Second, cfg.Timeout)

	// Verify global config
	assert.Equal(t, "https://api.test.com", cfg.Global.BaseURL)
	assert.Equal(t, "test-api-key", cfg.Global.APIKey)
	assert.Equal(t, "gpt-4", cfg.Global.DefaultModel)
	assert.Equal(t, 5, cfg.Global.MaxRetries)
	assert.Equal(t, 2*time.Second, cfg.Global.RequestDelay)
	assert.Equal(t, 45*time.Second, cfg.Global.Timeout)

	// Verify database config
	assert.Equal(t, "./test.db", cfg.Database.Path)
	assert.Equal(t, "test-encryption-key", cfg.Database.EncryptionKey)

	// Verify API config
	assert.Equal(t, "9090", cfg.API.Port)
	assert.Equal(t, "test-jwt-secret", cfg.API.JWTSecret)
	assert.Equal(t, 200, cfg.API.RateLimit)
	assert.Equal(t, 50, cfg.API.BurstLimit)
	assert.Equal(t, 60, cfg.API.RateLimitWindow)
	assert.False(t, cfg.API.EnableCORS)
	assert.Equal(t, "127.0.0.1,192.168.1.1", cfg.API.TrustedProxies)
	assert.Equal(t, "https://example.com,https://test.com", cfg.API.CORSOrigins)
	assert.True(t, cfg.API.EnableHTTPS)
	assert.Equal(t, "./cert.pem", cfg.API.TLSCertFile)
	assert.Equal(t, "./key.pem", cfg.API.TLSKeyFile)
	assert.Equal(t, 30, cfg.API.ReadTimeout)
	assert.Equal(t, 30, cfg.API.WriteTimeout)
	assert.Equal(t, 1048576, cfg.API.MaxHeaderBytes)

	// Verify logging config
	assert.Equal(t, "debug", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
	assert.Equal(t, "file", cfg.Logging.Output)
	assert.Equal(t, "./logs/test.log", cfg.Logging.FilePath)
	assert.Equal(t, 100, cfg.Logging.MaxSize)
	assert.Equal(t, 5, cfg.Logging.MaxBackups)
	assert.Equal(t, 30, cfg.Logging.MaxAge)
	assert.True(t, cfg.Logging.Compress)

	// Verify monitoring config
	assert.True(t, cfg.Monitoring.EnableMetrics)
	assert.Equal(t, "9091", cfg.Monitoring.MetricsPort)
	assert.True(t, cfg.Monitoring.EnableHealth)
	assert.Equal(t, "8081", cfg.Monitoring.HealthPort)
	assert.True(t, cfg.Monitoring.EnableTracing)
	assert.Equal(t, "http://jaeger:14268/api/traces", cfg.Monitoring.TracingEndpoint)
	assert.True(t, cfg.Monitoring.EnableProfiling)
	assert.Equal(t, "6060", cfg.Monitoring.ProfilingPort)

	// Verify security config
	assert.True(t, cfg.Security.EnableRateLimiting)
	assert.True(t, cfg.Security.EnableIPWhitelist)
	assert.Equal(t, []string{"127.0.0.1", "192.168.1.0/24"}, cfg.Security.IPWhitelist)
	assert.True(t, cfg.Security.EnableRequestLogging)
	assert.Equal(t, []string{"authorization", "cookie"}, cfg.Security.SensitiveHeaders)
	assert.True(t, cfg.Security.EnableCSRFProtection)
	assert.Equal(t, 32, cfg.Security.CSRFTokenLength)
	assert.Equal(t, 120, cfg.Security.SessionTimeout)

	// Verify notifications config
	assert.True(t, cfg.Notifications.Slack.Enabled)
	assert.Equal(t, "https://hooks.slack.com/test", cfg.Notifications.Slack.WebhookURL)
	assert.True(t, cfg.Notifications.Email.Enabled)
	assert.Equal(t, "smtp.example.com", cfg.Notifications.Email.SMTPHost)
	assert.Equal(t, 587, cfg.Notifications.Email.SMTPPort)
	assert.Equal(t, "test@example.com", cfg.Notifications.Email.Username)
	assert.Equal(t, "test-password", cfg.Notifications.Email.Password)
	assert.Equal(t, "admin@example.com", cfg.Notifications.Email.DefaultRecipient)
	assert.False(t, cfg.Notifications.Telegram.Enabled)
	assert.False(t, cfg.Notifications.Matrix.Enabled)
	assert.False(t, cfg.Notifications.WhatsApp.Enabled)

	// Verify LLM configs
	assert.Len(t, cfg.LLMs, 2)
	
	llm1 := cfg.LLMs[0]
	assert.Equal(t, "Test LLM 1", llm1.Name)
	assert.Equal(t, "https://api.test1.com", llm1.Endpoint)
	assert.Equal(t, "test-key-1", llm1.APIKey)
	assert.Equal(t, "gpt-3.5-turbo", llm1.Model)
	assert.Equal(t, map[string]string{"X-Custom-Header": "test-value", "User-Agent": "LLM-Verifier/1.0"}, llm1.Headers)
	assert.True(t, llm1.Features["code_generation"])
	assert.False(t, llm1.Features["multimodal"])

	llm2 := cfg.LLMs[1]
	assert.Equal(t, "Test LLM 2", llm2.Name)
	assert.Equal(t, "https://api.test2.com", llm2.Endpoint)
	assert.Equal(t, "test-key-2", llm2.APIKey)
	assert.Equal(t, "claude-3", llm2.Model)
	assert.False(t, llm2.Features["code_generation"])
	assert.True(t, llm2.Features["multimodal"])
}

func TestLoadConfig_EnvOverrides(t *testing.T) {
	// Set environment variables
	os.Setenv("LLM_VERIFIER_PROFILE", "prod")
	os.Setenv("LLM_VERIFIER_CONCURRENCY", "20")
	os.Setenv("LLM_VERIFIER_API_PORT", "8080")
	os.Setenv("LLM_VERIFIER_DATABASE_PATH", "/data/production.db")
	os.Setenv("LLM_VERIFIER_GLOBAL_API_KEY", "prod-api-key")
	os.Setenv("LLM_VERIFIER_LOGGING_LEVEL", "info")
	defer func() {
		os.Unsetenv("LLM_VERIFIER_PROFILE")
		os.Unsetenv("LLM_VERIFIER_CONCURRENCY")
		os.Unsetenv("LLM_VERIFIER_API_PORT")
		os.Unsetenv("LLM_VERIFIER_DATABASE_PATH")
		os.Unsetenv("LLM_VERIFIER_GLOBAL_API_KEY")
		os.Unsetenv("LLM_VERIFIER_LOGGING_LEVEL")
	}()

	// Configure viper to read from environment
	viper.SetEnvPrefix("LLM_VERIFIER")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var cfg Config
	err := viper.Unmarshal(&cfg)
	require.NoError(t, err)

	// Verify environment overrides
	assert.Equal(t, "prod", cfg.Profile)
	assert.Equal(t, 20, cfg.Concurrency)
	assert.Equal(t, "8080", cfg.API.Port)
	assert.Equal(t, "/data/production.db", cfg.Database.Path)
	assert.Equal(t, "prod-api-key", cfg.Global.APIKey)
	assert.Equal(t, "info", cfg.Logging.Level)
}

func TestLoadConfig_DefaultValues(t *testing.T) {
	// Reset viper to default state
	viper.Reset()

	var cfg Config
	err := viper.Unmarshal(&cfg)
	require.NoError(t, err)

	// Verify default values are set
	assert.Equal(t, "", cfg.Profile)
	assert.Equal(t, 0, cfg.Concurrency) // Default 0
	assert.Equal(t, time.Duration(0), cfg.Timeout) // Default 0

	// Global defaults
	assert.Equal(t, "", cfg.Global.BaseURL)
	assert.Equal(t, "", cfg.Global.APIKey)
	assert.Equal(t, "", cfg.Global.DefaultModel)
	assert.Equal(t, 0, cfg.Global.MaxRetries)
	assert.Equal(t, time.Duration(0), cfg.Global.RequestDelay)
	assert.Equal(t, time.Duration(0), cfg.Global.Timeout)

	// Database defaults
	assert.Equal(t, "", cfg.Database.Path)
	assert.Equal(t, "", cfg.Database.EncryptionKey)

	// API defaults
	assert.Equal(t, "", cfg.API.Port)
	assert.Equal(t, "", cfg.API.JWTSecret)
	assert.Equal(t, 0, cfg.API.RateLimit)
	assert.False(t, cfg.API.EnableCORS)

	// Logging defaults
	assert.Equal(t, "", cfg.Logging.Level)
	assert.Equal(t, "", cfg.Logging.Format)
	assert.Equal(t, "", cfg.Logging.Output)
	assert.Equal(t, 0, cfg.Logging.MaxSize)
	assert.False(t, cfg.Logging.Compress)

	// Monitoring defaults
	assert.False(t, cfg.Monitoring.EnableMetrics)
	assert.False(t, cfg.Monitoring.EnableHealth)
	assert.False(t, cfg.Monitoring.EnableTracing)
	assert.False(t, cfg.Monitoring.EnableProfiling)

	// Security defaults
	assert.False(t, cfg.Security.EnableRateLimiting)
	assert.False(t, cfg.Security.EnableIPWhitelist)
	assert.Empty(t, cfg.Security.IPWhitelist)
	assert.False(t, cfg.Security.EnableCSRFProtection)

	// Notifications defaults
	assert.False(t, cfg.Notifications.Slack.Enabled)
	assert.False(t, cfg.Notifications.Email.Enabled)
	assert.False(t, cfg.Notifications.Telegram.Enabled)
	assert.False(t, cfg.Notifications.Matrix.Enabled)
	assert.False(t, cfg.Notifications.WhatsApp.Enabled)
}

func TestLoadConfig_MissingFile(t *testing.T) {
	// Try to load non-existent file
	viper.SetConfigFile("/non/existent/config.yaml")
	err := viper.ReadInConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	// Create temporary config file with invalid YAML
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid-config.yaml")

	configContent := `
profile: "test"
global:
  base_url: "https://api.test.com"
  api_key: "test-api-key"
invalid yaml: [unclosed bracket
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Try to load invalid config
	viper.SetConfigFile(configPath)
	err = viper.ReadInConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "yaml")
}

func TestLoadConfig_PartialConfig(t *testing.T) {
	// Create config with only some sections
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "partial-config.yaml")

	configContent := `
global:
  base_url: "https://api.test.com"
  api_key: "test-api-key"

database:
  path: "./test.db"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load config
	viper.SetConfigFile(configPath)
	err = viper.ReadInConfig()
	require.NoError(t, err)

	var cfg Config
	err = viper.Unmarshal(&cfg)
	require.NoError(t, err)

	// Verify loaded sections
	assert.Equal(t, "https://api.test.com", cfg.Global.BaseURL)
	assert.Equal(t, "test-api-key", cfg.Global.APIKey)
	assert.Equal(t, "./test.db", cfg.Database.Path)

	// Verify other sections have defaults
	assert.Empty(t, cfg.LLMs)
	assert.Equal(t, "", cfg.API.Port)
	assert.Empty(t, cfg.Logging.Level)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		cfg    Config
		valid  bool
		errMsg string
	}{
		{
			name: "Valid minimal config",
			cfg: Config{
				Global: GlobalConfig{
					BaseURL: "https://api.test.com",
					APIKey:  "test-key",
				},
				Database: DatabaseConfig{
					Path: "./test.db",
				},
			},
			valid:  true,
			errMsg: "",
		},
		{
			name: "Missing global base URL",
			cfg: Config{
				Global: GlobalConfig{
					APIKey: "test-key",
				},
				Database: DatabaseConfig{
					Path: "./test.db",
				},
			},
			valid:  false,
			errMsg: "global.base_url is required",
		},
		{
			name: "Missing global API key",
			cfg: Config{
				Global: GlobalConfig{
					BaseURL: "https://api.test.com",
				},
				Database: DatabaseConfig{
					Path: "./test.db",
				},
			},
			valid:  false,
			errMsg: "global.api_key is required",
		},
		{
			name: "Missing database path",
			cfg: Config{
				Global: GlobalConfig{
					BaseURL: "https://api.test.com",
					APIKey:  "test-key",
				},
				Database: DatabaseConfig{},
			},
			valid:  false,
			errMsg: "database.path is required",
		},
		{
			name: "Invalid API port",
			cfg: Config{
				Global: GlobalConfig{
					BaseURL: "https://api.test.com",
					APIKey:  "test-key",
				},
				Database: DatabaseConfig{
					Path: "./test.db",
				},
				API: APIConfig{
					Port: "invalid",
				},
			},
			valid:  false,
			errMsg: "api.port must be a valid port number",
		},
		{
			name: "Negative concurrency",
			cfg: Config{
				Global: GlobalConfig{
					BaseURL: "https://api.test.com",
					APIKey:  "test-key",
				},
				Database: DatabaseConfig{
					Path: "./test.db",
				},
				Concurrency: -1,
			},
			valid:  false,
			errMsg: "concurrency must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCompleteConfig(&tt.cfg)
			
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			}
		})
	}
}