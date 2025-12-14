package llmverifier

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
	"llm-verifier/config"
)

// LoadConfig loads the configuration from a YAML file
func LoadConfig(filePath string) (*config.Config, error) {
	viper.SetConfigFile(filePath)
	viper.AutomaticEnv() // Allow environment variables to override config

	// Set default values
	viper.SetDefault("concurrency", 1)
	viper.SetDefault("timeout", 60*time.Second)

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg config.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Expand environment variables in the config
	cfg.Global.APIKey = os.ExpandEnv(cfg.Global.APIKey)

	for i := range cfg.LLMs {
		cfg.LLMs[i].APIKey = os.ExpandEnv(cfg.LLMs[i].APIKey)
		cfg.LLMs[i].Endpoint = os.ExpandEnv(cfg.LLMs[i].Endpoint)
	}

	return &cfg, nil
}