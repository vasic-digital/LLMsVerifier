// Package cliagents provides unified CLI agent configuration generation and validation.
package cliagents

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// HelixCodeConfig represents the configuration for HelixCode CLI
type HelixCodeConfig struct {
	Schema              string                             `json:"$schema,omitempty"`
	Version             string                             `json:"version,omitempty"`
	Provider            HelixCodeProviderConfig            `json:"provider"`
	AdditionalProviders map[string]HelixCodeProviderConfig `json:"additional_providers,omitempty"`
	Models              []HelixCodeModelDef                `json:"models,omitempty"`
	MCP                 map[string]HelixCodeMCP            `json:"mcp,omitempty"`
	Plugins             []string                           `json:"plugins,omitempty"`
	Agents              map[string]HelixCodeAgent          `json:"agents,omitempty"`
	Tools               HelixCodeTools                     `json:"tools,omitempty"`
	Permissions         HelixCodePermissions               `json:"permissions,omitempty"`
	Settings            HelixCodeSettings                  `json:"settings,omitempty"`
	Extensions          *HelixAgentExtensions              `json:"extensions,omitempty"`
	Formatters          FormattersConfig                   `json:"formatters,omitempty"`
	Auth                *HelixCodeAuth                     `json:"auth,omitempty"`
}

// HelixCodeAuth carries the JWT secret used by the HelixCode CLI to mint
// the bearer token it sends to HelixAgent. This MUST match the HelixAgent
// server's JWT_SECRET (sourced from HelixAgent's .env). Finding #24:
// generator previously omitted this block; users had to hand-patch every
// installed config or HelixCode would fail to start with a JWT
// validation error.
type HelixCodeAuth struct {
	JWTSecret string `json:"jwt_secret"`
}

// HelixCodeProviderConfig represents the provider configuration
type HelixCodeProviderConfig struct {
	Type      string                   `json:"type"`
	BaseURL   string                   `json:"base_url"`
	APIKey    string                   `json:"api_key,omitempty"`
	APIKeyEnv string                   `json:"api_key_env,omitempty"`
	Options   HelixCodeProviderOptions `json:"options,omitempty"`
}

// HelixCodeProviderOptions contains additional provider options
type HelixCodeProviderOptions struct {
	Timeout    int  `json:"timeout,omitempty"`
	MaxRetries int  `json:"max_retries,omitempty"`
	EnableSSE  bool `json:"enable_sse,omitempty"`
}

// HelixCodeModelDef represents a model definition
type HelixCodeModelDef struct {
	ID           string                `json:"id"`
	Name         string                `json:"name,omitempty"`
	MaxTokens    int                   `json:"max_tokens,omitempty"`
	Capabilities HelixCodeCapabilities `json:"capabilities,omitempty"`
	Protocols    []string              `json:"protocols,omitempty"`
}

// HelixCodeCapabilities represents model capabilities
type HelixCodeCapabilities struct {
	Vision        bool `json:"vision,omitempty"`
	Streaming     bool `json:"streaming,omitempty"`
	FunctionCalls bool `json:"function_calls,omitempty"`
	Embeddings    bool `json:"embeddings,omitempty"`
	CodeExec      bool `json:"code_execution,omitempty"`
	FileOps       bool `json:"file_operations,omitempty"`
	WebSearch     bool `json:"web_search,omitempty"`
}

// HelixCodeMCP represents an MCP server configuration
type HelixCodeMCP struct {
	Type        string            `json:"type"` // local, remote, stdio
	Command     []string          `json:"command,omitempty"`
	URL         string            `json:"url,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Environment map[string]string `json:"env,omitempty"`
	Enabled     bool              `json:"enabled,omitempty"`
}

// HelixCodeAgent represents an agent configuration
type HelixCodeAgent struct {
	Model        string   `json:"model"`
	SystemPrompt string   `json:"system_prompt,omitempty"`
	MaxTokens    int      `json:"max_tokens,omitempty"`
	Temperature  float64  `json:"temperature,omitempty"`
	Tools        []string `json:"tools,omitempty"`
}

// HelixCodeTools represents tool configurations
type HelixCodeTools struct {
	FileSystem bool `json:"filesystem,omitempty"`
	Terminal   bool `json:"terminal,omitempty"`
	Browser    bool `json:"browser,omitempty"`
	Search     bool `json:"search,omitempty"`
	Git        bool `json:"git,omitempty"`
	MCP        bool `json:"mcp,omitempty"`
	LSP        bool `json:"lsp,omitempty"`
	Embeddings bool `json:"embeddings,omitempty"`
	Vision     bool `json:"vision,omitempty"`
}

// HelixCodePermissions represents permissions configuration
type HelixCodePermissions struct {
	AllowRead    bool     `json:"allow_read,omitempty"`
	AllowWrite   bool     `json:"allow_write,omitempty"`
	AllowExec    bool     `json:"allow_exec,omitempty"`
	AllowNetwork bool     `json:"allow_network,omitempty"`
	AllowedPaths []string `json:"allowed_paths,omitempty"`
	DeniedPaths  []string `json:"denied_paths,omitempty"`
}

// HelixCodeSettings represents settings configuration
type HelixCodeSettings struct {
	Theme           string `json:"theme,omitempty"`
	AutoSave        bool   `json:"auto_save,omitempty"`
	StreamResponses bool   `json:"stream_responses,omitempty"`
	ShowProgress    bool   `json:"show_progress,omitempty"`
	ConfirmActions  bool   `json:"confirm_actions,omitempty"`
	DebugMode       bool   `json:"debug_mode,omitempty"`
	LogLevel        string `json:"log_level,omitempty"`
}

// HelixCodeGenerator generates HelixCode configurations
type HelixCodeGenerator struct {
	schema *AgentSchema
}

// NewHelixCodeGenerator creates a new HelixCode configuration generator
func NewHelixCodeGenerator() *HelixCodeGenerator {
	homeDir, _ := os.UserHomeDir()
	return &HelixCodeGenerator{
		schema: &AgentSchema{
			AgentType:        AgentHelixCode,
			ConfigFileName:   "helixcode.json",
			ConfigDirEnvVar:  "HELIXCODE_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "helixcode"),
			SupportedFields: []string{
				"$schema", "version", "provider", "models", "mcp", "agents",
				"tools", "permissions", "settings",
			},
			RequiredFields: []string{"provider"},
			Description:    "HelixCode CLI - Native CLI for HelixAgent with AI Debate Ensemble",
		},
	}
}

// Generate generates a HelixCode configuration
func (g *HelixCodeGenerator) Generate(ctx context.Context, config *GeneratorConfig) (*GenerationResult, error) {
	result := &GenerationResult{
		AgentType:   AgentHelixCode,
		GeneratedAt: time.Now(),
	}

	// Build the configuration
	helixConfig := &HelixCodeConfig{
		Version: "1.0",
	}

	// Configure provider
	baseURL := fmt.Sprintf("http://%s:%d/v1", config.HelixAgentHost, config.HelixAgentPort)
	// Use real API key from environment for installed configs
	apiKey := os.Getenv("HELIXAGENT_API_KEY")
	if apiKey == "" {
		apiKey = "<YOUR_HELIXAGENT_API_KEY>"
	}

	// Single "Helix Agent" provider with TWO models:
	// - helix-debate: Helix AI Debate Ensemble (for coder, summarizer)
	// - helix-llm: Helix LLM (for task, title - with provider chain fallback)
	helixConfig.Provider = HelixCodeProviderConfig{
		Type:    "helixagent",
		BaseURL: baseURL,
		APIKey:  apiKey,
		Options: HelixCodeProviderOptions{
			Timeout:    120,
			MaxRetries: 3,
			EnableSSE:  true,
		},
	}

	// Configure models
	helixConfig.Models = []HelixCodeModelDef{
		{
			ID:        "helix-llm",
			Name:      "Helix LLM",
			MaxTokens: 128000,
			Capabilities: HelixCodeCapabilities{
				Vision:        true,
				Streaming:     true,
				FunctionCalls: true,
				Embeddings:    true,
				CodeExec:      true,
				FileOps:       true,
				WebSearch:     true,
			},
			Protocols: []string{"openai-compatible"},
		},
		{
			ID:        "helix-debate",
			Name:      "Helix AI Debate Ensemble",
			MaxTokens: 128000,
			Capabilities: HelixCodeCapabilities{
				Vision:        true,
				Streaming:     true,
				FunctionCalls: true,
				Embeddings:    true,
				CodeExec:      true,
				FileOps:       true,
				WebSearch:     true,
			},
			Protocols: []string{"mcp", "acp", "lsp"},
		},
	}

	// Configure MCP servers (15+ out of the box)
	helixConfig.MCP = make(map[string]HelixCodeMCP)
	for _, mcpServer := range config.MCPServers {
		mcp := HelixCodeMCP{
			Type:    mcpServer.Type,
			Enabled: true,
		}
		if mcpServer.Type == "remote" {
			mcp.URL = mcpServer.URL
		} else {
			mcp.Command = mcpServer.Command
			mcp.Args = mcpServer.Args
			mcp.Environment = mcpServer.Environment
		}
		helixConfig.MCP[mcpServer.Name] = mcp
	}

	// Configure plugins
	helixConfig.Plugins = DefaultPlugins()

	// Configure agents
	// Agent routing: task/title → helix-llm, coder/summarizer → helix-debate
	helixConfig.Agents = map[string]HelixCodeAgent{
		"default": {
			Model:        "helix-debate",
			SystemPrompt: "You are HelixCode, an AI coding assistant powered by the HelixAgent AI Debate Ensemble. You have access to all HelixAgent capabilities including MCP, ACP, LSP protocols.",
			MaxTokens:    8192,
			Temperature:  0.7,
			Tools:        []string{"filesystem", "terminal", "mcp", "lsp", "embeddings", "vision"},
		},
		"coder": {
			Model:        "helix-debate",
			SystemPrompt: "You are an expert software developer. Write clean, efficient, well-tested code. Follow best practices and design patterns.",
			MaxTokens:    16384,
			Temperature:  0.3,
			Tools:        []string{"filesystem", "terminal", "git", "lsp"},
		},
		"reviewer": {
			Model:        "helix-debate",
			SystemPrompt: "You are a senior code reviewer. Analyze code for bugs, security vulnerabilities, performance issues, and adherence to best practices.",
			MaxTokens:    8192,
			Temperature:  0.5,
			Tools:        []string{"filesystem", "search", "lsp"},
		},
		"architect": {
			Model:        "helix-llm",
			SystemPrompt: "You are a software architect. Design scalable, maintainable systems. Consider trade-offs and provide clear architectural decisions.",
			MaxTokens:    16384,
			Temperature:  0.6,
			Tools:        []string{"filesystem", "search", "browser"},
		},
		"task": {
			Model:        "helix-llm",
			SystemPrompt: "You are a task assistant. Execute tasks efficiently with provider chain fallback for reliability.",
			MaxTokens:    4096,
			Temperature:  0.7,
			Tools:        []string{"filesystem", "terminal", "mcp"},
		},
		"title": {
			Model:        "helix-llm",
			SystemPrompt: "You are a title generator. Create concise, descriptive titles.",
			MaxTokens:    80,
			Temperature:  0.5,
			Tools:        []string{"filesystem"},
		},
	}

	// Configure tools
	helixConfig.Tools = HelixCodeTools{
		FileSystem: true,
		Terminal:   true,
		Browser:    true,
		Search:     true,
		Git:        true,
		MCP:        true,
		LSP:        true,
		Embeddings: true,
		Vision:     true,
	}

	// Configure permissions
	helixConfig.Permissions = HelixCodePermissions{
		AllowRead:    true,
		AllowWrite:   true,
		AllowExec:    true,
		AllowNetwork: true,
	}

	// Configure settings
	helixConfig.Settings = HelixCodeSettings{
		Theme:           "auto",
		AutoSave:        true,
		StreamResponses: true,
		ShowProgress:    true,
		ConfirmActions:  true,
		LogLevel:        "info",
	}

	// Configure extensions (LSP, ACP, Embeddings, RAG, Skills)
	helixConfig.Extensions = DefaultHelixAgentExtensions(config.HelixAgentHost, config.HelixAgentPort)

	// Configure formatters
	helixConfig.Formatters = DefaultFormattersConfig(config.HelixAgentHost, config.HelixAgentPort)

	// Configure auth: HelixCode mints JWTs that HelixAgent validates with
	// the same secret. Source from JWT_SECRET (HelixAgent's canonical env
	// var) with HELIXAGENT_JWT_SECRET as fallback. Finding #24: missing
	// this block caused every installed HelixCode to fail at startup with
	// a JWT validation error until manually patched.
	if jwt := os.Getenv("JWT_SECRET"); jwt != "" {
		helixConfig.Auth = &HelixCodeAuth{JWTSecret: jwt}
	} else if jwt := os.Getenv("HELIXAGENT_JWT_SECRET"); jwt != "" {
		helixConfig.Auth = &HelixCodeAuth{JWTSecret: jwt}
	} else {
		helixConfig.Auth = &HelixCodeAuth{JWTSecret: "<YOUR_JWT_SECRET>"}
	}

	result.Config = helixConfig
	result.Success = true

	return result, nil
}

// Validate validates a HelixCode configuration
func (g *HelixCodeGenerator) Validate(config any) (*ValidationResult, error) {
	result := &ValidationResult{Valid: true}

	helixConfig, ok := config.(*HelixCodeConfig)
	if !ok {
		// Try to cast from map
		if configMap, ok := config.(map[string]interface{}); ok {
			return g.validateMap(configMap)
		}
		result.Valid = false
		result.Errors = append(result.Errors, "invalid configuration type: expected *HelixCodeConfig")
		return result, nil
	}

	// Validate required fields
	if helixConfig.Provider.BaseURL == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "provider.base_url is required")
	}

	// Validate MCP servers
	for name, mcp := range helixConfig.MCP {
		if mcp.Type == "" {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("mcp.%s: type is required", name))
		}
		if mcp.Type == "remote" && mcp.URL == "" {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("mcp.%s: url is required for remote type", name))
		}
		if mcp.Type == "local" && len(mcp.Command) == 0 {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("mcp.%s: command is required for local type", name))
		}
	}

	// Validate agents
	for name, agent := range helixConfig.Agents {
		if agent.Model == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("agent.%s: model is not specified", name))
		}
	}

	return result, nil
}

// validateMap validates a configuration from a map
func (g *HelixCodeGenerator) validateMap(config map[string]interface{}) (*ValidationResult, error) {
	result := &ValidationResult{Valid: true}

	// Check for required provider section
	provider, hasProvider := config["provider"]
	if !hasProvider {
		result.Valid = false
		result.Errors = append(result.Errors, "provider section is required")
		return result, nil
	}

	// Validate provider has base_url
	providerMap, ok := provider.(map[string]interface{})
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, "provider must be an object")
		return result, nil
	}

	if _, hasBaseURL := providerMap["base_url"]; !hasBaseURL {
		result.Valid = false
		result.Errors = append(result.Errors, "provider.base_url is required")
	}

	// Validate MCP servers if present
	if mcp, hasMCP := config["mcp"]; hasMCP {
		mcpMap, ok := mcp.(map[string]interface{})
		if !ok {
			result.Valid = false
			result.Errors = append(result.Errors, "mcp must be an object")
		} else {
			for name, server := range mcpMap {
				serverMap, ok := server.(map[string]interface{})
				if !ok {
					result.Valid = false
					result.Errors = append(result.Errors, fmt.Sprintf("mcp.%s must be an object", name))
					continue
				}

				serverType, hasType := serverMap["type"].(string)
				if !hasType {
					result.Valid = false
					result.Errors = append(result.Errors, fmt.Sprintf("mcp.%s: type is required", name))
				}

				if serverType == "remote" {
					if _, hasURL := serverMap["url"]; !hasURL {
						result.Valid = false
						result.Errors = append(result.Errors, fmt.Sprintf("mcp.%s: url is required for remote type", name))
					}
				}
			}
		}
	}

	return result, nil
}

// GetSchema returns the HelixCode configuration schema
func (g *HelixCodeGenerator) GetSchema() *AgentSchema {
	return g.schema
}
