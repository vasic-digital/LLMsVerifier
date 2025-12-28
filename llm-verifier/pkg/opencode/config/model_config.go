package opencode_config

// ModelConfig represents configuration for a specific model
type ModelConfig struct {
	Name             string   `json:"name"`
	MaxTokens        int      `json:"maxTokens"`
	CostPer1MIn      float64  `json:"cost_per_1m_in,omitempty"`
	CostPer1MOut     float64  `json:"cost_per_1m_out,omitempty"`
	SupportsBrotli   bool     `json:"supports_brotli,omitempty"`
}

// ProviderConfig with models support
type ProviderConfigWithModels struct {
	Options map[string]interface{} `json:"options,omitempty"`
	Model   string                 `json:"model,omitempty"`
	Models  map[string]ModelConfig `json:"models,omitempty"`
}