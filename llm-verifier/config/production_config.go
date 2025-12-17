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
	Server ServerConfig `yaml:"server"`

	// Database configuration
	Database DatabaseConfig `yaml:"database"`

	// Security configuration
	Security SecurityConfig `yaml:"security"`

	// Performance configuration
	Performance PerformanceConfig `yaml:"performance"`

	// Monitoring configuration
	Monitoring MonitoringConfig `yaml:"monitoring"`

	// LLM provider configuration
	Providers ProvidersConfig `yaml:"providers"`

	// Enterprise features
	Enterprise EnterpriseConfig `yaml:"enterprise"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
	TLS          TLSConfig     `yaml:"tls"`
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
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

// SecurityConfig holds security configuration
type SecurityConfig struct {
	JWTSecret          string          `yaml:"jwt_secret"`
	TokenExpiry        time.Duration   `yaml:"token_expiry"`
	RefreshTokenExpiry time.Duration   `yaml:"refresh_token_expiry"`
	MaxLoginAttempts   int             `yaml:"max_login_attempts"`
	LockoutDuration    time.Duration   `yaml:"lockout_duration"`
	SessionTimeout     time.Duration   `yaml:"session_timeout"`
	RequireHTTPS       bool            `yaml:"require_https"`
	RateLimiting       RateLimitConfig `yaml:"rate_limiting"`
	CORS               CORSConfig      `yaml:"cors"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Requests int           `yaml:"requests"`
	Window   time.Duration `yaml:"window"`
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	Enabled          bool     `yaml:"enabled"`
	AllowedOrigins   []string `yaml:"allowed_origins"`
	AllowedMethods   []string `yaml:"allowed_methods"`
	AllowedHeaders   []string `yaml:"allowed_headers"`
	ExposedHeaders   []string `yaml:"exposed_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
	MaxAge           int      `yaml:"max_age"`
}

// PerformanceConfig holds performance configuration
type PerformanceConfig struct {
	MaxWorkers      int           `yaml:"max_workers"`
	WorkerTimeout   time.Duration `yaml:"worker_timeout"`
	QueueSize       int           `yaml:"queue_size"`
	EnableProfiling bool          `yaml:"enable_profiling"`
	EnableMetrics   bool          `yaml:"enable_metrics"`
	MemoryLimit     int64         `yaml:"memory_limit"`
	CPUQuota        int           `yaml:"cpu_quota"`
}

// MonitoringConfig holds monitoring configuration
type MonitoringConfig struct {
	Enabled    bool             `yaml:"enabled"`
	Prometheus PrometheusConfig `yaml:"prometheus"`
	Tracing    TracingConfig    `yaml:"tracing"`
	Logging    LoggingConfig    `yaml:"logging"`
}

// PrometheusConfig holds Prometheus configuration
type PrometheusConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Port      int    `yaml:"port"`
	Path      string `yaml:"path"`
	Namespace string `yaml:"namespace"`
}

// TracingConfig holds distributed tracing configuration
type TracingConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Provider string `yaml:"provider"`
	Endpoint string `yaml:"endpoint"`
	Service  string `yaml:"service"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	Output     string `yaml:"output"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
	Compress   bool   `yaml:"compress"`
}

// ProvidersConfig holds LLM provider configurations
type ProvidersConfig struct {
	DefaultProvider string                 `yaml:"default_provider"`
	OpenAI          OpenAIConfig           `yaml:"openai"`
	Azure           AzureConfig            `yaml:"azure"`
	Anthropic       AnthropicConfig        `yaml:"anthropic"`
	Google          GoogleConfig           `yaml:"google"`
	Local           map[string]interface{} `yaml:"local"`
}

// OpenAIConfig holds OpenAI configuration
type OpenAIConfig struct {
	APIKey      string        `yaml:"api_key"`
	BaseURL     string        `yaml:"base_url"`
	Timeout     time.Duration `yaml:"timeout"`
	MaxTokens   int           `yaml:"max_tokens"`
	Temperature float64       `yaml:"temperature"`
}

// AzureConfig holds Azure OpenAI configuration
type AzureConfig struct {
	APIKey         string        `yaml:"api_key"`
	Endpoint       string        `yaml:"endpoint"`
	DeploymentName string        `yaml:"deployment_name"`
	APIVersion     string        `yaml:"api_version"`
	Timeout        time.Duration `yaml:"timeout"`
	MaxTokens      int           `yaml:"max_tokens"`
	Temperature    float64       `yaml:"temperature"`
}

// AnthropicConfig holds Anthropic configuration
type AnthropicConfig struct {
	APIKey      string        `yaml:"api_key"`
	BaseURL     string        `yaml:"base_url"`
	Timeout     time.Duration `yaml:"timeout"`
	MaxTokens   int           `yaml:"max_tokens"`
	Temperature float64       `yaml:"temperature"`
}

// GoogleConfig holds Google AI configuration
type GoogleConfig struct {
	APIKey      string        `yaml:"api_key"`
	ProjectID   string        `yaml:"project_id"`
	Timeout     time.Duration `yaml:"timeout"`
	MaxTokens   int           `yaml:"max_tokens"`
	Temperature float64       `yaml:"temperature"`
}

// EnterpriseConfig holds enterprise features configuration
type EnterpriseConfig struct {
	RBAC         RBACConfig         `yaml:"rbac"`
	MultiTenant  MultiTenantConfig  `yaml:"multi_tenant"`
	LDAP         LDAPConfig         `yaml:"ldap"`
	SAML         SAMLConfig         `yaml:"saml"`
	AuditLogging AuditLoggingConfig `yaml:"audit_logging"`
}

// RBACConfig holds RBAC configuration
type RBACConfig struct {
	Enabled        bool   `yaml:"enabled"`
	DefaultRole    string `yaml:"default_role"`
	AdminRole      string `yaml:"admin_role"`
	SuperAdminRole string `yaml:"super_admin_role"`
}

// MultiTenantConfig holds multi-tenant configuration
type MultiTenantConfig struct {
	Enabled       bool   `yaml:"enabled"`
	DefaultTenant string `yaml:"default_tenant"`
	TenantHeader  string `yaml:"tenant_header"`
	IsolationMode string `yaml:"isolation_mode"`
}

// LDAPConfig holds LDAP configuration
type LDAPConfig struct {
	Enabled      bool     `yaml:"enabled"`
	Host         string   `yaml:"host"`
	Port         int      `yaml:"port"`
	BaseDN       string   `yaml:"base_dn"`
	BindUser     string   `yaml:"bind_user"`
	BindPassword string   `yaml:"bind_password"`
	UserFilter   string   `yaml:"user_filter"`
	GroupFilter  string   `yaml:"group_filter"`
	Attributes   []string `yaml:"attributes"`
	UseSSL       bool     `yaml:"use_ssl"`
}

// SAMLConfig holds SAML configuration
type SAMLConfig struct {
	Enabled             bool   `yaml:"enabled"`
	EntityID            string `yaml:"entity_id"`
	SSOURL              string `yaml:"sso_url"`
	IdentityProviderURL string `yaml:"identity_provider_url"`
	CertificateFile     string `yaml:"certificate_file"`
	KeyFile             string `yaml:"key_file"`
}

// AuditLoggingConfig holds audit logging configuration
type AuditLoggingConfig struct {
	Enabled     bool          `yaml:"enabled"`
	Storage     string        `yaml:"storage"`
	Retention   time.Duration `yaml:"retention"`
	Compression bool          `yaml:"compression"`
}

// ConfigManager manages configuration loading and validation
type ConfigManager struct {
	config *ProductionConfig
	env    Environment
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		env: getEnvironment(),
	}
}

// LoadConfiguration loads configuration from files and environment
func (cm *ConfigManager) LoadConfiguration() (*ProductionConfig, error) {
	// Load base configuration
	config, err := cm.loadBaseConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load base config: %w", err)
	}

	// Override with environment-specific configuration
	envConfig, err := cm.loadEnvironmentConfig(cm.env)
	if err != nil {
		return nil, fmt.Errorf("failed to load environment config: %w", err)
	}

	// Merge configurations
	config = cm.mergeConfigs(config, envConfig)

	// Override with environment variables
	config = cm.overrideWithEnvVars(config)

	// Set environment
	config.Environment = cm.env

	// Validate configuration
	if err := cm.validateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	cm.config = config
	return config, nil
}

// GetConfiguration returns the loaded configuration
func (cm *ConfigManager) GetConfiguration() *ProductionConfig {
	return cm.config
}

// getEnvironment determines the current environment
func getEnvironment() Environment {
	env := strings.ToLower(os.Getenv("ENVIRONMENT"))
	if env == "" {
		env = strings.ToLower(os.Getenv("GO_ENV"))
	}
	if env == "" {
		env = strings.ToLower(os.Getenv("ENV"))
	}
	if env == "" {
		return EnvDevelopment
	}

	switch env {
	case "dev", "development":
		return EnvDevelopment
	case "staging", "stage":
		return EnvStaging
	case "prod", "production":
		return EnvProduction
	default:
		return EnvDevelopment
	}
}

// loadBaseConfig loads the base configuration file
func (cm *ConfigManager) loadBaseConfig() (*ProductionConfig, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/production.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var config ProductionConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// loadEnvironmentConfig loads environment-specific configuration
func (cm *ConfigManager) loadEnvironmentConfig(env Environment) (*ProductionConfig, error) {
	configPath := filepath.Join("config", fmt.Sprintf("%s.yaml", env))

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &ProductionConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read environment config file %s: %w", configPath, err)
	}

	var config ProductionConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse environment config file: %w", err)
	}

	return &config, nil
}

// mergeConfigs merges two configurations
func (cm *ConfigManager) mergeConfigs(base, override *ProductionConfig) *ProductionConfig {
	// Simple merge - in production, you'd use a more sophisticated merge
	if override.Server.Host != "" {
		base.Server.Host = override.Server.Host
	}
	if override.Server.Port != 0 {
		base.Server.Port = override.Server.Port
	}
	if override.Database.Host != "" {
		base.Database.Host = override.Database.Host
	}
	if override.Database.Port != 0 {
		base.Database.Port = override.Database.Port
	}

	return base
}

// overrideWithEnvVars overrides configuration with environment variables
func (cm *ConfigManager) overrideWithEnvVars(config *ProductionConfig) *ProductionConfig {
	if host := os.Getenv("SERVER_HOST"); host != "" {
		config.Server.Host = host
	}
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := parseInt(port); err == nil {
			config.Server.Port = p
		}
	}
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		config.Database.Host = dbHost
	}
	if dbPort := os.Getenv("DB_PORT"); dbPort != "" {
		if p, err := parseInt(dbPort); err == nil {
			config.Database.Port = p
		}
	}
	if dbPassword := os.Getenv("DB_PASSWORD"); dbPassword != "" {
		config.Database.Password = dbPassword
	}
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		config.Security.JWTSecret = jwtSecret
	}

	return config
}

// validateConfig validates the configuration
func (cm *ConfigManager) validateConfig(config *ProductionConfig) error {
	if config.Server.Host == "" {
		return fmt.Errorf("server host is required")
	}
	if config.Server.Port == 0 {
		return fmt.Errorf("server port is required")
	}
	if config.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if config.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}
	if config.Security.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required")
	}

	if cm.env == EnvProduction {
		if config.Security.RequireHTTPS {
			if !config.Server.TLS.Enabled {
				return fmt.Errorf("HTTPS is required in production but TLS is not enabled")
			}
		}
	}

	return nil
}

// parseInt parses an integer from string
func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// IsProduction checks if the current environment is production
func (cm *ConfigManager) IsProduction() bool {
	return cm.env == EnvProduction
}

// IsDevelopment checks if the current environment is development
func (cm *ConfigManager) IsDevelopment() bool {
	return cm.env == EnvDevelopment
}

// ReloadConfiguration reloads the configuration
func (cm *ConfigManager) ReloadConfiguration() error {
	config, err := cm.LoadConfiguration()
	if err != nil {
		return err
	}
	cm.config = config
	return nil
}
