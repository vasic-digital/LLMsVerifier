// Package cliagents provides unified CLI agent configuration generation and validation.
package cliagents

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// GenericAgentConfig represents a generic configuration for various CLI agents
type GenericAgentConfig struct {
	Version     string                     `json:"version,omitempty"`
	Provider    GenericProviderConfig      `json:"provider"`
	Models      []GenericModelDef          `json:"models,omitempty"`
	MCP         map[string]GenericMCP      `json:"mcp,omitempty"`
	Settings    map[string]interface{}     `json:"settings,omitempty"`
}

// GenericProviderConfig represents a generic provider configuration
type GenericProviderConfig struct {
	Type      string `json:"type,omitempty"`
	Name      string `json:"name,omitempty"`
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key,omitempty"`
	APIKeyEnv string `json:"api_key_env,omitempty"`
}

// GenericModelDef represents a generic model definition
type GenericModelDef struct {
	ID           string   `json:"id"`
	Name         string   `json:"name,omitempty"`
	MaxTokens    int      `json:"max_tokens,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// GenericMCP represents a generic MCP server configuration
type GenericMCP struct {
	Type        string            `json:"type"`
	Command     []string          `json:"command,omitempty"`
	URL         string            `json:"url,omitempty"`
	Environment map[string]string `json:"env,omitempty"`
}

// GenericAgentGenerator generates configurations for various CLI agents
type GenericAgentGenerator struct {
	agentType AgentType
	schema    *AgentSchema
}

// NewAiderGenerator creates a generator for Aider CLI
func NewAiderGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentAider,
		schema: &AgentSchema{
			AgentType:        AgentAider,
			ConfigFileName:   ".aider.conf.yml",
			DefaultConfigDir: homeDir,
			SupportedFields:  []string{"model", "openai-api-base", "openai-api-key", "auto-commits", "edit-format"},
			RequiredFields:   []string{"model"},
			Description:      "Aider CLI - AI pair programming in the terminal",
		},
	}
}

// NewContinueGenerator creates a generator for Continue.dev extension
func NewContinueGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentContinue,
		schema: &AgentSchema{
			AgentType:        AgentContinue,
			ConfigFileName:   "config.json",
			ConfigDirEnvVar:  "CONTINUE_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".continue"),
			SupportedFields:  []string{"models", "tabAutocompleteModel", "embeddingsProvider", "contextProviders", "slashCommands"},
			RequiredFields:   []string{"models"},
			Description:      "Continue.dev - Open-source AI code assistant",
		},
	}
}

// NewCursorGenerator creates a generator for Cursor editor
func NewCursorGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentCursor,
		schema: &AgentSchema{
			AgentType:        AgentCursor,
			ConfigFileName:   "settings.json",
			DefaultConfigDir: filepath.Join(homeDir, ".cursor"),
			SupportedFields:  []string{"ai.provider", "ai.model", "ai.apiKey", "ai.baseUrl"},
			RequiredFields:   []string{"ai.provider"},
			Description:      "Cursor - AI-first code editor",
		},
	}
}

// NewClineGenerator creates a generator for Cline CLI
func NewClineGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentCline,
		schema: &AgentSchema{
			AgentType:        AgentCline,
			ConfigFileName:   "cline.json",
			ConfigDirEnvVar:  "CLINE_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "cline"),
			SupportedFields:  []string{"provider", "model", "apiKey", "baseUrl", "maxTokens"},
			RequiredFields:   []string{"provider"},
			Description:      "Cline - AI coding assistant CLI",
		},
	}
}

// NewWindsurfGenerator creates a generator for Windsurf editor
func NewWindsurfGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentWindsurf,
		schema: &AgentSchema{
			AgentType:        AgentWindsurf,
			ConfigFileName:   "windsurf.json",
			DefaultConfigDir: filepath.Join(homeDir, ".windsurf"),
			SupportedFields:  []string{"provider", "models", "settings", "keybindings"},
			RequiredFields:   []string{"provider"},
			Description:      "Windsurf - AI-powered code editor",
		},
	}
}

// NewZedGenerator creates a generator for Zed editor
func NewZedGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentZed,
		schema: &AgentSchema{
			AgentType:        AgentZed,
			ConfigFileName:   "settings.json",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "zed"),
			SupportedFields:  []string{"assistant", "language_models", "inline_completions"},
			RequiredFields:   []string{"assistant"},
			Description:      "Zed - High-performance code editor with AI",
		},
	}
}

// NewNeovimAIGenerator creates a generator for Neovim AI plugins
func NewNeovimAIGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentNeovimAI,
		schema: &AgentSchema{
			AgentType:        AgentNeovimAI,
			ConfigFileName:   "ai.lua",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "nvim", "lua"),
			SupportedFields:  []string{"provider", "model", "api_key", "endpoint", "keymaps"},
			RequiredFields:   []string{"provider"},
			Description:      "Neovim AI plugins (ChatGPT.nvim, copilot.lua, etc.)",
		},
	}
}

// NewVSCodeAIGenerator creates a generator for VS Code AI extensions
func NewVSCodeAIGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentVSCodeAI,
		schema: &AgentSchema{
			AgentType:        AgentVSCodeAI,
			ConfigFileName:   "settings.json",
			DefaultConfigDir: filepath.Join(homeDir, ".vscode"),
			SupportedFields:  []string{"ai.provider", "ai.model", "ai.apiKey", "ai.endpoint"},
			RequiredFields:   []string{"ai.provider"},
			Description:      "VS Code AI extensions configuration",
		},
	}
}

// NewIntelliJAIGenerator creates a generator for IntelliJ AI plugins
func NewIntelliJAIGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentIntelliJAI,
		schema: &AgentSchema{
			AgentType:        AgentIntelliJAI,
			ConfigFileName:   "ai-settings.xml",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "JetBrains"),
			SupportedFields:  []string{"provider", "model", "apiKey", "endpoint", "enabled"},
			RequiredFields:   []string{"provider"},
			Description:      "IntelliJ AI Assistant and plugins",
		},
	}
}

// NewClaudeCodeGenerator creates a generator for Claude Code CLI
func NewClaudeCodeGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentClaudeCode,
		schema: &AgentSchema{
			AgentType:        AgentClaudeCode,
			ConfigFileName:   "settings.json",
			ConfigDirEnvVar:  "CLAUDE_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".claude"),
			SupportedFields:  []string{"apiKey", "model", "maxTokens", "temperature", "permissions"},
			RequiredFields:   []string{"apiKey"},
			Description:      "Claude Code - Anthropic's CLI for Claude",
		},
	}
}

// NewQwenCodeGenerator creates a generator for Qwen Code CLI
func NewQwenCodeGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentQwenCode,
		schema: &AgentSchema{
			AgentType:        AgentQwenCode,
			ConfigFileName:   "qwen-code.json",
			ConfigDirEnvVar:  "QWEN_CODE_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "qwen-code"),
			SupportedFields:  []string{"apiKey", "model", "baseUrl", "maxTokens", "settings"},
			RequiredFields:   []string{"apiKey"},
			Description:      "Qwen Code - Alibaba's AI coding assistant CLI",
		},
	}
}

// NewCopilotGenerator creates a generator for GitHub Copilot
func NewCopilotGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentCopilot,
		schema: &AgentSchema{
			AgentType:        AgentCopilot,
			ConfigFileName:   "hosts.json",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "github-copilot"),
			SupportedFields:  []string{"github.com", "oauth_token", "user"},
			RequiredFields:   []string{"github.com"},
			Description:      "GitHub Copilot - AI pair programmer",
		},
	}
}

// Generate generates a configuration for the generic agent
func (g *GenericAgentGenerator) Generate(ctx context.Context, config *GeneratorConfig) (*GenerationResult, error) {
	result := &GenerationResult{
		AgentType:   g.agentType,
		GeneratedAt: time.Now(),
	}

	// Build the configuration
	agentConfig := &GenericAgentConfig{
		Version: "1.0",
	}

	// Configure provider
	baseURL := fmt.Sprintf("http://%s:%d/v1", config.HelixAgentHost, config.HelixAgentPort)
	agentConfig.Provider = GenericProviderConfig{
		Type:      "openai-compatible",
		Name:      "helixagent",
		BaseURL:   baseURL,
		APIKeyEnv: "HELIXAGENT_API_KEY",
	}

	// Configure models
	agentConfig.Models = []GenericModelDef{
		{
			ID:        "helixagent-debate",
			Name:      "HelixAgent AI Debate Ensemble",
			MaxTokens: 128000,
			Capabilities: []string{
				"vision", "streaming", "function_calls", "embeddings",
				"mcp", "acp", "lsp",
			},
		},
	}

	// Configure MCP servers
	agentConfig.MCP = make(map[string]GenericMCP)
	for _, mcpServer := range config.MCPServers {
		mcp := GenericMCP{
			Type: mcpServer.Type,
		}
		if mcpServer.Type == "remote" {
			mcp.URL = mcpServer.URL
		} else {
			mcp.Command = mcpServer.Command
			mcp.Environment = mcpServer.Environment
		}
		agentConfig.MCP[mcpServer.Name] = mcp
	}

	// Agent-specific settings
	agentConfig.Settings = g.getAgentSpecificSettings()

	result.Config = agentConfig
	result.Success = true

	return result, nil
}

// getAgentSpecificSettings returns agent-specific default settings
func (g *GenericAgentGenerator) getAgentSpecificSettings() map[string]interface{} {
	switch g.agentType {
	case AgentAider:
		return map[string]interface{}{
			"auto_commits":  true,
			"edit_format":   "diff",
			"stream":        true,
			"show_diffs":    true,
		}
	case AgentContinue:
		return map[string]interface{}{
			"allowAnonymousTelemetry": false,
			"tabAutocomplete":         true,
			"systemMessage":           "You are a helpful AI coding assistant powered by HelixAgent.",
		}
	case AgentCursor:
		return map[string]interface{}{
			"enableAI":          true,
			"suggestionsDelay":  100,
			"maxSuggestions":    3,
		}
	case AgentCline:
		return map[string]interface{}{
			"autoApprove":    false,
			"streamResponse": true,
			"showDiff":       true,
		}
	case AgentWindsurf:
		return map[string]interface{}{
			"theme":           "auto",
			"inlineComplete":  true,
			"chatEnabled":     true,
		}
	case AgentZed:
		return map[string]interface{}{
			"assistant": map[string]interface{}{
				"enabled": true,
				"button":  true,
			},
		}
	case AgentNeovimAI:
		return map[string]interface{}{
			"show_popup":    true,
			"auto_trigger":  false,
			"timeout":       30000,
		}
	case AgentVSCodeAI:
		return map[string]interface{}{
			"inlineSuggest.enabled": true,
			"chat.enabled":          true,
		}
	case AgentIntelliJAI:
		return map[string]interface{}{
			"enabled":      true,
			"autoComplete": true,
		}
	case AgentClaudeCode:
		return map[string]interface{}{
			"autoApprove": false,
			"theme":       "auto",
		}
	case AgentQwenCode:
		return map[string]interface{}{
			"language": "en",
			"stream":   true,
		}
	case AgentCopilot:
		return map[string]interface{}{
			"enabled": true,
		}
	default:
		return map[string]interface{}{}
	}
}

// Validate validates a configuration for the generic agent
func (g *GenericAgentGenerator) Validate(config any) (*ValidationResult, error) {
	result := &ValidationResult{Valid: true}

	genericConfig, ok := config.(*GenericAgentConfig)
	if !ok {
		// Try to cast from map
		if configMap, ok := config.(map[string]interface{}); ok {
			return g.validateMap(configMap)
		}
		result.Valid = false
		result.Errors = append(result.Errors, "invalid configuration type")
		return result, nil
	}

	// Validate required fields
	if genericConfig.Provider.BaseURL == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "provider.base_url is required")
	}

	// Validate MCP servers
	for name, mcp := range genericConfig.MCP {
		if mcp.Type == "" {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("mcp.%s: type is required", name))
		}
	}

	return result, nil
}

// validateMap validates a configuration from a map
func (g *GenericAgentGenerator) validateMap(config map[string]interface{}) (*ValidationResult, error) {
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

	return result, nil
}

// GetSchema returns the configuration schema for this agent
func (g *GenericAgentGenerator) GetSchema() *AgentSchema {
	return g.schema
}
