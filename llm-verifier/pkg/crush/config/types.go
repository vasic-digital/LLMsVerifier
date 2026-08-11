package crush_config

import (
	"encoding/json"
	"os"

	"digital.vasic.llmsverifier/pkg/helixendpoint"
)

// Config represents the complete Crush configuration structure matching the official schema
// Schema: https://charm.land/crush.json
type Config struct {
	Schema      string                    `json:"$schema,omitempty"`
	Models      map[string]SelectedModel  `json:"models,omitempty"`
	Providers   map[string]ProviderConfig `json:"providers,omitempty"`
	MCP         MCPs                      `json:"mcp,omitempty"`
	LSP         LSPs                      `json:"lsp,omitempty"`
	Options     *Options                  `json:"options,omitempty"`
	Permissions *Permissions              `json:"permissions,omitempty"`
	Tools       *Tools                    `json:"tools,omitempty"`
}

// SelectedModel represents a selected model configuration
type SelectedModel struct {
	Model            string                 `json:"model"`
	Provider         string                 `json:"provider"`
	ReasoningEffort  string                 `json:"reasoning_effort,omitempty"` // low, medium, high
	Think            bool                   `json:"think,omitempty"`
	MaxTokens        int                    `json:"max_tokens,omitempty"`
	Temperature      float64                `json:"temperature,omitempty"`
	TopP             float64                `json:"top_p,omitempty"`
	TopK             int                    `json:"top_k,omitempty"`
	FrequencyPenalty float64                `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64                `json:"presence_penalty,omitempty"`
	ProviderOptions  map[string]interface{} `json:"provider_options,omitempty"`
}

// ProviderConfig represents an AI provider configuration
type ProviderConfig struct {
	ID                 string                 `json:"id,omitempty"`
	Name               string                 `json:"name,omitempty"`
	Type               string                 `json:"type"` // openai, openai-compat, anthropic, gemini, azure, vertexai
	BaseURL            string                 `json:"base_url,omitempty"`
	APIKey             string                 `json:"api_key,omitempty"`
	OAuth              *Token                 `json:"oauth,omitempty"`
	Disable            bool                   `json:"disable,omitempty"`
	SystemPromptPrefix string                 `json:"system_prompt_prefix,omitempty"`
	ExtraHeaders       map[string]string      `json:"extra_headers,omitempty"`
	ExtraBody          map[string]interface{} `json:"extra_body,omitempty"`
	ProviderOptions    map[string]interface{} `json:"provider_options,omitempty"`
	Models             []Model                `json:"models,omitempty"`
}

// Token represents an OAuth2 token
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	ExpiresAt    int    `json:"expires_at"`
}

// Model represents a model available from a provider
type Model struct {
	ID                     string        `json:"id"`
	Name                   string        `json:"name"`
	CostPer1MIn            float64       `json:"cost_per_1m_in"`
	CostPer1MOut           float64       `json:"cost_per_1m_out"`
	CostPer1MInCached      float64       `json:"cost_per_1m_in_cached,omitempty"`
	CostPer1MOutCached     float64       `json:"cost_per_1m_out_cached,omitempty"`
	ContextWindow          int           `json:"context_window"`
	DefaultMaxTokens       int           `json:"default_max_tokens"`
	CanReason              bool          `json:"can_reason"`
	ReasoningLevels        []string      `json:"reasoning_levels,omitempty"`
	DefaultReasoningEffort string        `json:"default_reasoning_effort,omitempty"`
	SupportsAttachments    bool          `json:"supports_attachments"`
	Options                *ModelOptions `json:"options,omitempty"`
}

// ModelOptions represents model-specific options
type ModelOptions struct {
	Temperature      float64                `json:"temperature,omitempty"`
	TopP             float64                `json:"top_p,omitempty"`
	TopK             int                    `json:"top_k,omitempty"`
	FrequencyPenalty float64                `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64                `json:"presence_penalty,omitempty"`
	ProviderOptions  map[string]interface{} `json:"provider_options,omitempty"`
}

// MCPs represents a collection of MCP server configurations
type MCPs map[string]MCPConfig

// MCPConfig represents an MCP server configuration
type MCPConfig struct {
	Type          string            `json:"type"`                     // stdio, sse, http
	Command       string            `json:"command,omitempty"`        // Command for stdio MCP servers
	Args          []string          `json:"args,omitempty"`           // Arguments for the command
	Env           map[string]string `json:"env,omitempty"`            // Environment variables
	URL           string            `json:"url,omitempty"`            // URL for HTTP/SSE MCP servers
	Disabled      bool              `json:"disabled,omitempty"`       // Whether this MCP server is disabled
	DisabledTools []string          `json:"disabled_tools,omitempty"` // Tools to disable from this MCP
	Timeout       int               `json:"timeout,omitempty"`        // Timeout in seconds (default: 15)
	Headers       map[string]string `json:"headers,omitempty"`        // HTTP headers for HTTP/SSE MCP servers
}

// LSPs represents a collection of LSP server configurations
type LSPs map[string]LSPConfig

// LSPConfig represents an LSP server configuration
type LSPConfig struct {
	Disabled    bool                   `json:"disabled,omitempty"`
	Command     string                 `json:"command,omitempty"`
	Args        []string               `json:"args,omitempty"`
	Env         map[string]string      `json:"env,omitempty"`
	Filetypes   []string               `json:"filetypes,omitempty"`
	RootMarkers []string               `json:"root_markers,omitempty"`
	InitOptions map[string]interface{} `json:"init_options,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
	Timeout     int                    `json:"timeout,omitempty"` // Default: 30
}

// Options represents general application options
type Options struct {
	ContextPaths              []string     `json:"context_paths,omitempty"`
	SkillsPaths               []string     `json:"skills_paths,omitempty"`
	TUI                       *TUIOptions  `json:"tui,omitempty"`
	Debug                     bool         `json:"debug,omitempty"`
	DebugLSP                  bool         `json:"debug_lsp,omitempty"`
	DisableAutoSummarize      bool         `json:"disable_auto_summarize,omitempty"`
	DataDirectory             string       `json:"data_directory,omitempty"`
	DisabledTools             []string     `json:"disabled_tools,omitempty"`
	DisableProviderAutoUpdate bool         `json:"disable_provider_auto_update,omitempty"`
	DisableDefaultProviders   bool         `json:"disable_default_providers,omitempty"`
	Attribution               *Attribution `json:"attribution,omitempty"`
	DisableMetrics            bool         `json:"disable_metrics,omitempty"`
	InitializeAs              string       `json:"initialize_as,omitempty"`
	AutoLSP                   bool         `json:"auto_lsp,omitempty"`
	Progress                  bool         `json:"progress,omitempty"`
}

// TUIOptions represents terminal user interface options
type TUIOptions struct {
	CompactMode bool         `json:"compact_mode,omitempty"`
	DiffMode    string       `json:"diff_mode,omitempty"` // unified, split
	Completions *Completions `json:"completions,omitempty"`
	Transparent bool         `json:"transparent,omitempty"`
}

// Completions represents completions UI options
type Completions struct {
	MaxDepth int `json:"max_depth,omitempty"`
	MaxItems int `json:"max_items,omitempty"`
}

// Attribution represents attribution settings for generated content
type Attribution struct {
	TrailerStyle  string `json:"trailer_style,omitempty"`  // none, co-authored-by, assisted-by
	CoAuthoredBy  bool   `json:"co_authored_by,omitempty"` // Deprecated
	GeneratedWith bool   `json:"generated_with,omitempty"`
}

// Permissions represents permission settings for tool usage
type Permissions struct {
	AllowedTools []string `json:"allowed_tools,omitempty"`
}

// Tools represents tool configurations
type Tools struct {
	LS   *ToolLS   `json:"ls"`
	Grep *ToolGrep `json:"grep"`
}

// ToolLS represents ls tool configuration
type ToolLS struct {
	MaxDepth int `json:"max_depth,omitempty"`
	MaxItems int `json:"max_items,omitempty"`
}

// ToolGrep represents grep tool configuration
type ToolGrep struct {
	Timeout int `json:"timeout,omitempty"`
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

// CreateDefaultConfig creates a default Crush configuration for HelixAgent.
//
// HXC-250: the provider base URL is RESOLVED from the consuming project's
// injected endpoint (see package helixendpoint), never hardcoded here. Callers
// holding an explicit endpoint should use CreateHelixAgentConfig, which
// overrides the resolved default.
func CreateDefaultConfig() *Config {
	return &Config{
		Schema: "https://charm.land/crush.json",
		Providers: map[string]ProviderConfig{
			"helixagent": {
				ID:      "helixagent",
				Name:    "HelixAgent",
				Type:    "openai-compat",
				BaseURL: helixendpoint.JoinPath(helixendpoint.DefaultBaseURL(), "v1"),
				Models: []Model{
					{
						ID:                  "helixagent-debate",
						Name:                "HelixAgent AI Debate Ensemble",
						CostPer1MIn:         0,
						CostPer1MOut:        0,
						CostPer1MInCached:   0,
						CostPer1MOutCached:  0,
						ContextWindow:       128000,
						DefaultMaxTokens:    8192,
						CanReason:           true,
						SupportsAttachments: true,
						Options: &ModelOptions{
							Temperature: 0.7,
						},
					},
				},
			},
		},
		Models: map[string]SelectedModel{
			"default": {
				Model:     "helixagent-debate",
				Provider:  "helixagent",
				MaxTokens: 8192,
			},
		},
		MCP: MCPs{},
		LSP: LSPs{},
		Tools: &Tools{
			LS: &ToolLS{
				MaxDepth: 0,
				MaxItems: 1000,
			},
			Grep: &ToolGrep{
				Timeout: 30,
			},
		},
		Options: &Options{
			AutoLSP:      true,
			Progress:     true,
			Debug:        false,
			InitializeAs: "AGENTS.md",
			Attribution: &Attribution{
				TrailerStyle:  "assisted-by",
				GeneratedWith: true,
			},
		},
		Permissions: &Permissions{
			AllowedTools: []string{"bash", "view", "edit", "glob", "grep"},
		},
	}
}

// CreateHelixAgentConfig creates a Crush configuration for HelixAgent with custom settings
func CreateHelixAgentConfig(apiKey, baseURL string, mcpServers MCPs, lspServers LSPs) *Config {
	config := CreateDefaultConfig()

	if baseURL != "" {
		provider := config.Providers["helixagent"]
		provider.BaseURL = baseURL
		config.Providers["helixagent"] = provider
	}

	if apiKey != "" {
		provider := config.Providers["helixagent"]
		provider.APIKey = apiKey
		config.Providers["helixagent"] = provider
	}

	if mcpServers != nil {
		config.MCP = mcpServers
	}

	if lspServers != nil {
		config.LSP = lspServers
	}

	return config
}
