package crush_config

import (
	"encoding/json"
	"os"
)

// Config represents the Crush configuration structure
type Config struct {
	Schema   string                   `json:"$schema,omitempty"`
	Providers map[string]Provider     `json:"providers,omitempty"`
	Lsp       map[string]LspConfig    `json:"lsp,omitempty"`
	Options   *Options               `json:"options,omitempty"`
}

// Provider represents a provider configuration
type Provider struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	BaseURL  string      `json:"base_url"`
	APIKey   string      `json:"api_key,omitempty"`
	Models   []Model     `json:"models"`
}

// Model represents a model configuration
type Model struct {
	ID                    string                 `json:"id"`
	Name                  string                 `json:"name"`
	CostPer1MIn           float64                `json:"cost_per_1m_in"`
	CostPer1MOut          float64                `json:"cost_per_1m_out"`
	CostPer1MInCached     float64                `json:"cost_per_1m_in_cached,omitempty"`
	CostPer1MOutCached    float64                `json:"cost_per_1m_out_cached,omitempty"`
	ContextWindow         int                    `json:"context_window"`
	DefaultMaxTokens      int                    `json:"default_max_tokens"`
	CanReason            bool                   `json:"can_reason"`
	SupportsAttachments   bool                   `json:"supports_attachments"`
	Streaming            bool                   `json:"streaming"`
	SupportsBrotli       bool                   `json:"supports_brotli,omitempty"`
	Options              map[string]interface{} `json:"options,omitempty"`
}

// LspConfig represents Language Server Protocol configuration
type LspConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	Enabled bool     `json:"enabled"`
}

// Options represents global configuration options
type Options struct {
	DisableProviderAutoUpdate bool `json:"disable_provider_auto_update,omitempty"`
}

// ConfigLoader handles loading and saving Crush configurations
type ConfigLoader struct{}

// LoadFromFile loads a Crush configuration from a file
func (cl *ConfigLoader) LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	
	return &config, nil
}

// SaveToFile saves a Crush configuration to a file
func (cl *ConfigLoader) SaveToFile(config *Config, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(path, data, 0644)
}

// LoadAndParse loads and parses a Crush configuration file
func LoadAndParse(path string) (*Config, error) {
	loader := ConfigLoader{}
	return loader.LoadFromFile(path)
}

// SaveConfig saves a Crush configuration to a file
func SaveConfig(config *Config, path string) error {
	loader := ConfigLoader{}
	return loader.SaveToFile(config, path)
}

// CreateDefaultConfig creates a default Crush configuration
func CreateDefaultConfig() *Config {
	return &Config{
		Schema: "https://charm.land/crush.json",
		Providers: map[string]Provider{
			"openai": {
				Name:    "openai",
				Type:    "openai",
				BaseURL: "https://api.openai.com/v1",
				Models: []Model{
					{
						ID:                  "gpt-4",
						Name:                "GPT-4",
						CostPer1MIn:         30.0,
						CostPer1MOut:        60.0,
						ContextWindow:       128000,
						DefaultMaxTokens:    4096,
						CanReason:          true,
						SupportsAttachments: false,
						Streaming:          true,
						SupportsBrotli:     true,
						Options:            map[string]interface{}{},
					},
				},
			},
		},
		Lsp: map[string]LspConfig{},
		Options: &Options{
			DisableProviderAutoUpdate: false,
		},
	}
}