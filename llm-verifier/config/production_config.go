package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Environment represents the deployment environment
type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvStaging     Environment = "staging"
	EnvProduction  Environment = "production"
)

// ProductionConfig holds production-specific configuration
type ProductionConfig struct {
	// Core settings
	Environment Environment `yaml:"environment"`
	Debug       bool        `yaml:"debug"`
	LogLevel    string      `yaml:"log_level"`

	// Server configuration
	Server ProductionServerConfig `yaml:"server"`

	// Database configuration
	Database ProductionDatabaseConfig `yaml:"database"`

	// Security configuration
	Security ProductionSecurityConfig `yaml:"security"`

	// Performance configuration
	Performance ProductionPerformanceConfig `yaml:"performance"`

	// Monitoring configuration
	Monitoring ProductionMonitoringConfig `yaml:"monitoring"`

	// LLM provider configuration
	Providers ProvidersConfig `yaml:"providers"`

	// Enterprise features
	Enterprise EnterpriseConfig `yaml:"enterprise"`
}

// ProductionServerConfig holds server configuration
type ProductionServerConfig struct {
	Host         string              `yaml:"host"`
	Port         int                 `yaml:"port"`
	ReadTimeout  time.Duration       `yaml:"read_timeout"`
	WriteTimeout time.Duration       `yaml:"write_timeout"`
	IdleTimeout  time.Duration       `yaml:"idle_timeout"`
	TLS          ProductionTLSConfig `yaml:"tls"`
}

// ProductionTLSConfig holds TLS configuration
type ProductionTLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// ProductionDatabaseConfig holds production database configuration
type ProductionDatabaseConfig struct {
	Driver          string        `yaml:"driver"`
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	Database        string        `yaml:"database"`
	Username        string        `yaml:"username"`
	Password        string        `yaml:"password"`
	SSLMode         string        `yaml:"ssl_mode"`
	MaxConnections  int           `yaml:"max_connections"`
	MaxIdleTime     time.Duration `yaml:"max_idle_time"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// ProductionSecurityConfig holds production security configuration
type ProductionSecurityConfig struct {
	JWTSecret          string                    `yaml:"jwt_secret"`
	TokenExpiry        time.Duration             `yaml:"token_expiry"`
	RefreshTokenExpiry time.Duration             `yaml:"refresh_token_expiry"`
	MaxLoginAttempts   int                       `yaml:"max_login_attempts"`
	LockoutDuration    time.Duration             `yaml:"lockout_duration"`
	SessionTimeout     time.Duration             `yaml:"session_timeout"`
	RequireHTTPS       bool                      `yaml:"require_https"`
	RateLimiting       ProductionRateLimitConfig `yaml:"rate_limiting"`
	CORS               ProductionCORSConfig      `yaml:"cors"`
}

// ProductionRateLimitConfig holds rate limiting configuration
type ProductionRateLimitConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Requests int           `yaml:"requests"`
	Window   time.Duration `yaml:"window"`
}

// ProductionCORSConfig holds CORS configuration
type ProductionCORSConfig struct {
	Enabled          bool     `yaml:"enabled"`
	AllowedOrigins   []string `yaml:"allowed_origins"`
	AllowedMethods   []string `yaml:"allowed_methods"`
	AllowedHeaders   []string `yaml:"allowed_headers"`
	ExposedHeaders   []string `yaml:"exposed_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
	MaxAge           int      `yaml:"max_age"`
}

// ProductionPerformanceConfig holds performance configuration
type ProductionPerformanceConfig struct {
	MaxWorkers      int           `yaml:"max_workers"`
	WorkerTimeout   time.Duration `yaml:"worker_timeout"`
	QueueSize       int           `yaml:"queue_size"`
	EnableProfiling bool          `yaml:"enable_profiling"`
	EnableMetrics   bool          `yaml:"enable_metrics"`
	MemoryLimit     int64         `yaml:"memory_limit"`
	CPUQuota        int           `yaml:"cpu_quota"`
}

// ProductionMonitoringConfig holds monitoring configuration
type ProductionMonitoringConfig struct {
	Enabled    bool                       `yaml:"enabled"`
	Prometheus ProductionPrometheusConfig `yaml:"prometheus"`
	Tracing    ProductionTracingConfig    `yaml:"tracing"`
	Logging    ProductionLoggingConfig    `yaml:"logging"`
}

// ProductionPrometheusConfig holds Prometheus configuration
type ProductionPrometheusConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Port      int    `yaml:"port"`
	Path      string `yaml:"path"`
	Namespace string `yaml:"namespace"`
}

// ProductionTracingConfig holds distributed tracing configuration
type ProductionTracingConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Provider string `yaml:"provider"`
	Endpoint string `yaml:"endpoint"`
	Service  string `yaml:"service"`
}

// ProductionLoggingConfig holds production logging configuration
type ProductionLoggingConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	Output     string `yaml:"output"`
	FilePath   string `yaml:"file_path"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
	Compress   bool   `yaml:"compress"`
}

// ProvidersConfig holds LLM provider configurations
type ProvidersConfig struct {
	OpenAI    ProviderConfig `yaml:"openai"`
	Anthropic ProviderConfig `yaml:"anthropic"`
	Google    ProviderConfig `yaml:"google"`
	Meta      ProviderConfig `yaml:"meta"`
	DeepSeek  ProviderConfig `yaml:"deepseek"`
}

// ProviderConfig holds individual provider configuration
type ProviderConfig struct {
	Enabled   bool              `yaml:"enabled"`
	APIKey    string            `yaml:"api_key"`
	BaseURL   string            `yaml:"base_url"`
	Models    []string          `yaml:"models"`
	Timeout   time.Duration     `yaml:"timeout"`
	RateLimit ProviderRateLimit `yaml:"rate_limit"`
}

// ProviderRateLimit holds provider-specific rate limiting
type ProviderRateLimit struct {
	Requests int           `yaml:"requests"`
	Window   time.Duration `yaml:"window"`
}

// EnterpriseConfig holds enterprise features configuration
type EnterpriseConfig struct {
	RBAC             RBACConfig             `yaml:"rbac"`
	MultiTenancy     MultiTenancyConfig     `yaml:"multi_tenancy"`
	SSO              SSOConfig              `yaml:"sso"`
	AuditLogging     AuditLoggingConfig     `yaml:"audit_logging"`
	Compliance       ComplianceConfig       `yaml:"compliance"`
	AdvancedFeatures AdvancedFeaturesConfig `yaml:"advanced_features"`
}

// RBACConfig holds Role-Based Access Control configuration
type RBACConfig struct {
	Enabled  bool                    `yaml:"enabled"`
	Roles    map[string][]string     `yaml:"roles"`
	Policies map[string]PolicyConfig `yaml:"policies"`
}

// PolicyConfig holds policy configuration
type PolicyConfig struct {
	Resources []string `yaml:"resources"`
	Actions   []string `yaml:"actions"`
	Effect    string   `yaml:"effect"`
}

// MultiTenancyConfig holds multi-tenancy configuration
type MultiTenancyConfig struct {
	Enabled       bool     `yaml:"enabled"`
	IsolationType string   `yaml:"isolation_type"`
	TenantHeader  string   `yaml:"tenant_header"`
	DefaultTenant string   `yaml:"default_tenant"`
	AdminTenants  []string `yaml:"admin_tenants"`
}

// SSOConfig holds Single Sign-On configuration
type SSOConfig struct {
	Enabled   bool     `yaml:"enabled"`
	Providers []string `yaml:"providers"`
	Metadata  string   `yaml:"metadata"`
	Callback  string   `yaml:"callback"`
}

// AuditLoggingConfig holds audit logging configuration
type AuditLoggingConfig struct {
	Enabled    bool   `yaml:"enabled"`
	LogLevel   string `yaml:"log_level"`
	OutputPath string `yaml:"output_path"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
	Compress   bool   `yaml:"compress"`
}

// ComplianceConfig holds compliance configuration
type ComplianceConfig struct {
	Enabled       bool     `yaml:"enabled"`
	Standards     []string `yaml:"standards"`
	DataRetention int      `yaml:"data_retention"`
	Encryption    bool     `yaml:"encryption"`
	AccessControl bool     `yaml:"access_control"`
}

// AdvancedFeaturesConfig holds advanced features configuration
type AdvancedFeaturesConfig struct {
	Analytics      bool `yaml:"analytics"`
	ContextManager bool `yaml:"context_manager"`
	Checkpointing  bool `yaml:"checkpointing"`
	VectorStore    bool `yaml:"vector_store"`
	MLPipeline     bool `yaml:"ml_pipeline"`
}

// LoadProductionConfig loads production configuration from file
func LoadProductionConfig(configPath string) (*ProductionConfig, error) {
	config := &ProductionConfig{
		Environment: EnvProduction,
		Debug:       false,
		LogLevel:    "info",
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return getDefaultProductionConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// getDefaultProductionConfig returns default production configuration
func getDefaultProductionConfig() *ProductionConfig {
	return &ProductionConfig{
		Environment: EnvProduction,
		Debug:       false,
		LogLevel:    "info",
		Server: ProductionServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
			TLS: ProductionTLSConfig{
				Enabled:  false,
				CertFile: "/etc/ssl/certs/server.crt",
				KeyFile:  "/etc/ssl/private/server.key",
			},
		},
		Database: ProductionDatabaseConfig{
			Driver:          "postgres",
			Host:            "localhost",
			Port:            5432,
			Database:        "llm_verifier",
			Username:        "llm_verifier",
			Password:        "",
			SSLMode:         "require",
			MaxConnections:  100,
			MaxIdleTime:     30 * time.Minute,
			ConnMaxLifetime: 24 * time.Hour,
		},
		Security: ProductionSecurityConfig{
			JWTSecret:          "",
			TokenExpiry:        24 * time.Hour,
			RefreshTokenExpiry: 168 * time.Hour, // 7 days
			MaxLoginAttempts:   5,
			LockoutDuration:    15 * time.Minute,
			SessionTimeout:     8 * time.Hour,
			RequireHTTPS:       true,
			RateLimiting: ProductionRateLimitConfig{
				Enabled:  true,
				Requests: 1000,
				Window:   time.Minute,
			},
			CORS: ProductionCORSConfig{
				Enabled:          true,
				AllowedOrigins:   []string{"https://llm-verifier.com"},
				AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowedHeaders:   []string{"*"},
				ExposedHeaders:   []string{"X-Total-Count"},
				AllowCredentials: true,
				MaxAge:           86400,
			},
		},
		Performance: ProductionPerformanceConfig{
			MaxWorkers:      50,
			WorkerTimeout:   30 * time.Second,
			QueueSize:       1000,
			EnableProfiling: false,
			EnableMetrics:   true,
			MemoryLimit:     4 * 1024 * 1024 * 1024, // 4GB
			CPUQuota:        200,                    // 200% CPU
		},
		Monitoring: ProductionMonitoringConfig{
			Enabled: true,
			Prometheus: ProductionPrometheusConfig{
				Enabled:   true,
				Port:      9090,
				Path:      "/metrics",
				Namespace: "llm_verifier",
			},
			Tracing: ProductionTracingConfig{
				Enabled:  true,
				Provider: "jaeger",
				Endpoint: "http://localhost:14268/api/traces",
				Service:  "llm-verifier",
			},
			Logging: ProductionLoggingConfig{
				Level:      "info",
				Format:     "json",
				Output:     "file",
				FilePath:   "/app/logs/llm-verifier.log",
				MaxSize:    100,
				MaxBackups: 10,
				MaxAge:     30,
				Compress:   true,
			},
		},
		Providers: ProvidersConfig{
			OpenAI: ProviderConfig{
				Enabled: true,
				APIKey:  "",
				BaseURL: "https://api.openai.com/v1",
				Models:  []string{"gpt-4", "gpt-3.5-turbo"},
				Timeout: 30 * time.Second,
				RateLimit: ProviderRateLimit{
					Requests: 100,
					Window:   time.Minute,
				},
			},
			Anthropic: ProviderConfig{
				Enabled: true,
				APIKey:  "",
				BaseURL: "https://api.anthropic.com",
				Models:  []string{"claude-3-opus-20240229"},
				Timeout: 30 * time.Second,
				RateLimit: ProviderRateLimit{
					Requests: 50,
					Window:   time.Minute,
				},
			},
			Google: ProviderConfig{
				Enabled: true,
				APIKey:  "",
				BaseURL: "https://generativelanguage.googleapis.com",
				Models:  []string{"gemini-pro"},
				Timeout: 30 * time.Second,
				RateLimit: ProviderRateLimit{
					Requests: 60,
					Window:   time.Minute,
				},
			},
		},
		Enterprise: EnterpriseConfig{
			RBAC: RBACConfig{
				Enabled: true,
				Roles: map[string][]string{
					"admin": {"*"},
					"user":  {"read", "write"},
					"guest": {"read"},
				},
				Policies: map[string]PolicyConfig{
					"admin-policy": {
						Resources: []string{"*"},
						Actions:   []string{"*"},
						Effect:    "allow",
					},
				},
			},
			MultiTenancy: MultiTenancyConfig{
				Enabled:       true,
				IsolationType: "database",
				TenantHeader:  "X-Tenant-ID",
				DefaultTenant: "default",
				AdminTenants:  []string{"system"},
			},
			SSO: SSOConfig{
				Enabled:   false,
				Providers: []string{"oidc", "saml"},
				Metadata:  "/etc/sso/metadata.xml",
				Callback:  "/auth/callback",
			},
			AuditLogging: AuditLoggingConfig{
				Enabled:    true,
				LogLevel:   "info",
				OutputPath: "/app/logs/audit.log",
				MaxSize:    100,
				MaxBackups: 30,
				MaxAge:     90,
				Compress:   true,
			},
			Compliance: ComplianceConfig{
				Enabled:       true,
				Standards:     []string{"SOC2", "GDPR", "HIPAA"},
				DataRetention: 2555, // 7 years
				Encryption:    true,
				AccessControl: true,
			},
			AdvancedFeatures: AdvancedFeaturesConfig{
				Analytics:      true,
				ContextManager: true,
				Checkpointing:  true,
				VectorStore:    true,
				MLPipeline:     true,
			},
		},
	}
}

// GetProductionConfigPath returns the path to production config file
func GetProductionConfigPath() string {
	if configPath := os.Getenv("CONFIG_PATH"); configPath != "" {
		return configPath
	}

	return "/app/config/production.yaml"
}

// ValidateProductionConfig validates production configuration
func ValidateProductionConfig(config *ProductionConfig) error {
	if config == nil {
		return fmt.Errorf("production config is nil")
	}

	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	if config.Security.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required in production")
	}

	if config.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if config.Database.Password == "" {
		return fmt.Errorf("database password is required")
	}

	return nil
}

// LoadProductionConfigFromEnv loads production configuration from environment variables
func LoadProductionConfigFromEnv() *ProductionConfig {
	config := getDefaultProductionConfig()

	if host := os.Getenv("SERVER_HOST"); host != "" {
		config.Server.Host = host
	}

	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := fmt.Sscanf(port, "%d", &config.Server.Port); err == nil && p == 1 {
			// Successfully parsed port
		}
	}

	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		config.Security.JWTSecret = jwtSecret
	}

	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		config.Database.Host = dbHost
	}

	if dbPort := os.Getenv("DB_PORT"); dbPort != "" {
		if p, err := fmt.Sscanf(dbPort, "%d", &config.Database.Port); err == nil && p == 1 {
			// Successfully parsed port
		}
	}

	if dbPassword := os.Getenv("DB_PASSWORD"); dbPassword != "" {
		config.Database.Password = dbPassword
	}

	if openAIKey := os.Getenv("OPENAI_API_KEY"); openAIKey != "" {
		config.Providers.OpenAI.APIKey = openAIKey
	}

	if anthropicKey := os.Getenv("ANTHROPIC_API_KEY"); anthropicKey != "" {
		config.Providers.Anthropic.APIKey = anthropicKey
	}

	if googleKey := os.Getenv("GOOGLE_API_KEY"); googleKey != "" {
		config.Providers.Google.APIKey = googleKey
	}

	return config
}

// SaveProductionConfig saves production configuration to file
func SaveProductionConfig(config *ProductionConfig, configPath string) error {
	if err := ValidateProductionConfig(config); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// MergeProductionConfigs merges two production configurations
func MergeProductionConfigs(base, override *ProductionConfig) *ProductionConfig {
	result := *base // Deep copy

	if override.Environment != "" {
		result.Environment = override.Environment
	}

	if override.Debug {
		result.Debug = override.Debug
	}

	if override.LogLevel != "" {
		result.LogLevel = override.LogLevel
	}

	// Merge server config
	if override.Server.Host != "" {
		result.Server.Host = override.Server.Host
	}
	if override.Server.Port > 0 {
		result.Server.Port = override.Server.Port
	}
	if override.Server.ReadTimeout > 0 {
		result.Server.ReadTimeout = override.Server.ReadTimeout
	}
	if override.Server.WriteTimeout > 0 {
		result.Server.WriteTimeout = override.Server.WriteTimeout
	}
	if override.Server.IdleTimeout > 0 {
		result.Server.IdleTimeout = override.Server.IdleTimeout
	}

	// Merge database config
	if override.Database.Driver != "" {
		result.Database.Driver = override.Database.Driver
	}
	if override.Database.Host != "" {
		result.Database.Host = override.Database.Host
	}
	if override.Database.Port > 0 {
		result.Database.Port = override.Database.Port
	}
	if override.Database.Database != "" {
		result.Database.Database = override.Database.Database
	}
	if override.Database.Username != "" {
		result.Database.Username = override.Database.Username
	}
	if override.Database.Password != "" {
		result.Database.Password = override.Database.Password
	}

	// Merge security config
	if override.Security.JWTSecret != "" {
		result.Security.JWTSecret = override.Security.JWTSecret
	}
	if override.Security.TokenExpiry > 0 {
		result.Security.TokenExpiry = override.Security.TokenExpiry
	}
	if override.Security.MaxLoginAttempts > 0 {
		result.Security.MaxLoginAttempts = override.Security.MaxLoginAttempts
	}
	if override.Security.RequireHTTPS {
		result.Security.RequireHTTPS = override.Security.RequireHTTPS
	}

	// Merge monitoring config
	if override.Monitoring.Enabled {
		result.Monitoring.Enabled = override.Monitoring.Enabled
	}

	return &result
}

// GetProductionEnvVars returns environment variables for production deployment
func GetProductionEnvVars(config *ProductionConfig) map[string]string {
	envVars := make(map[string]string)

	envVars["GIN_MODE"] = "release"
	envVars["PORT"] = fmt.Sprintf("%d", config.Server.Port)
	envVars["LOG_LEVEL"] = config.LogLevel
	envVars["JWT_SECRET"] = config.Security.JWTSecret
	envVars["DB_HOST"] = config.Database.Host
	envVars["DB_PORT"] = fmt.Sprintf("%d", config.Database.Port)
	envVars["DB_NAME"] = config.Database.Database
	envVars["DB_USER"] = config.Database.Username
	envVars["DB_PASSWORD"] = config.Database.Password
	envVars["OPENAI_API_KEY"] = config.Providers.OpenAI.APIKey
	envVars["ANTHROPIC_API_KEY"] = config.Providers.Anthropic.APIKey
	envVars["GOOGLE_API_KEY"] = config.Providers.Google.APIKey

	// Monitoring and observability
	envVars["PROMETHEUS_ENABLED"] = fmt.Sprintf("%t", config.Monitoring.Prometheus.Enabled)
	envVars["PROMETHEUS_PORT"] = fmt.Sprintf("%d", config.Monitoring.Prometheus.Port)
	envVars["TRACING_ENABLED"] = fmt.Sprintf("%t", config.Monitoring.Tracing.Enabled)

	// Security
	envVars["REQUIRE_HTTPS"] = fmt.Sprintf("%t", config.Security.RequireHTTPS)
	envVars["RATE_LIMIT_ENABLED"] = fmt.Sprintf("%t", config.Security.RateLimiting.Enabled)
	envVars["RATE_LIMIT_REQUESTS"] = fmt.Sprintf("%d", config.Security.RateLimiting.Requests)

	// Enterprise features
	envVars["ENTERPRISE_ENABLED"] = "true"
	envVars["RBAC_ENABLED"] = fmt.Sprintf("%t", config.Enterprise.RBAC.Enabled)
	envVars["MULTI_TENANCY_ENABLED"] = fmt.Sprintf("%t", config.Enterprise.MultiTenancy.Enabled)
	envVars["AUDIT_LOGGING_ENABLED"] = fmt.Sprintf("%t", config.Enterprise.AuditLogging.Enabled)

	return envVars
}

// IsProductionEnvironment checks if running in production environment
func IsProductionEnvironment() bool {
	env := strings.ToLower(os.Getenv("ENVIRONMENT"))
	return env == "production" || env == "prod"
}

// GetProductionDatabaseURL returns production database URL
func GetProductionDatabaseURL(config *ProductionConfig) string {
	switch config.Database.Driver {
	case "postgres":
		return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			config.Database.Username,
			config.Database.Password,
			config.Database.Host,
			config.Database.Port,
			config.Database.Database,
			config.Database.SSLMode,
		)
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			config.Database.Username,
			config.Database.Password,
			config.Database.Host,
			config.Database.Port,
			config.Database.Database,
		)
	default:
		return ""
	}
}
