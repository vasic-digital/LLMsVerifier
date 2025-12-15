package llmverifier

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"llm-verifier/config"
)

// ExportConfig exports configuration to various formats
func ExportConfig(cfg *config.Config, format, outputPath string) error {
	var data []byte
	var err error

	switch strings.ToLower(format) {
	case "json":
		data, err = json.MarshalIndent(cfg, "", "  ")
	case "yaml", "yml":
		data, err = yaml.Marshal(cfg)
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure output directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ImportConfig imports configuration from a file
func ImportConfig(inputPath string) (*config.Config, error) {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg config.Config

	// Determine format from file extension
	ext := strings.ToLower(filepath.Ext(inputPath))
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON config: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal YAML config: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file format: %s", ext)
	}

	return &cfg, nil
}

// ValidateConfigFile validates a configuration file without loading it
func ValidateConfigFile(filePath string) error {
	cfg, err := ImportConfig(filePath)
	if err != nil {
		return err
	}

	// Validate the imported config
	if err := validateConfig(cfg); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	return nil
}

// MigrateConfig migrates configuration from old format to new format
func MigrateConfig(oldPath, newPath string) error {
	// Import old config
	oldCfg, err := ImportConfig(oldPath)
	if err != nil {
		return fmt.Errorf("failed to import old config: %w", err)
	}

	// Apply migration transformations
	newCfg := migrateConfigFields(oldCfg)

	// Export new config
	if err := ExportConfig(newCfg, "yaml", newPath); err != nil {
		return fmt.Errorf("failed to export migrated config: %w", err)
	}

	return nil
}

// migrateConfigFields applies migration transformations
func migrateConfigFields(oldCfg *config.Config) *config.Config {
	newCfg := *oldCfg // Copy all fields

	// Add default values for new fields
	if newCfg.Logging.Level == "" {
		newCfg.Logging.Level = "info"
	}
	if newCfg.Logging.Format == "" {
		newCfg.Logging.Format = "text"
	}
	if newCfg.Logging.Output == "" {
		newCfg.Logging.Output = "stdout"
	}

	// Set monitoring defaults
	if !newCfg.Monitoring.EnableHealth {
		newCfg.Monitoring.EnableHealth = true
		newCfg.Monitoring.HealthPort = "8086"
	}

	// Set security defaults
	if !newCfg.Security.EnableRateLimiting {
		newCfg.Security.EnableRateLimiting = true
	}

	return &newCfg
}

// GenerateConfigTemplate generates a configuration template
func GenerateConfigTemplate(profile string) (*config.Config, error) {
	cfg := &config.Config{}

	// Apply profile-specific template
	switch strings.ToLower(profile) {
	case "dev", "development":
		cfg.Profile = "dev"
		cfg.Logging.Level = "debug"
		cfg.Logging.Output = "stdout"
		cfg.API.Port = "8080"
		cfg.API.EnableCORS = true
		cfg.Database.Path = "llm_verifier_dev.db"
		cfg.Concurrency = 2
		cfg.Timeout = 30 * 60 * 1000000000 // 30 minutes in nanoseconds

	case "prod", "production":
		cfg.Profile = "prod"
		cfg.Logging.Level = "info"
		cfg.Logging.Format = "json"
		cfg.Logging.Output = "file"
		cfg.Logging.FilePath = "/var/log/llm-verifier.log"
		cfg.API.Port = "8080"
		cfg.API.EnableHTTPS = true
		cfg.Database.Path = "/var/lib/llm-verifier/llm_verifier.db"
		cfg.Concurrency = 10
		cfg.Timeout = 60 * 60 * 1000000000 // 1 hour in nanoseconds
		cfg.Monitoring.EnableMetrics = true
		cfg.Monitoring.EnableTracing = true
		cfg.Security.EnableRateLimiting = true

	case "test", "testing":
		cfg.Profile = "test"
		cfg.Logging.Level = "error"
		cfg.Logging.Output = "stdout"
		cfg.Database.Path = ":memory:"
		cfg.API.Port = "8081"
		cfg.Concurrency = 1
		cfg.Timeout = 5 * 1000000000 // 5 seconds in nanoseconds

	default:
		return nil, fmt.Errorf("unknown profile: %s", profile)
	}

	// Set computed defaults
	setComputedDefaults(cfg)

	return cfg, nil
}
