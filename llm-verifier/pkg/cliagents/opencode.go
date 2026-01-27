// Package cliagents provides unified CLI agent configuration generation and validation.
package cliagents

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// OpenCodeConfig represents the configuration for OpenCode CLI
type OpenCodeConfig struct {
	Schema      string                       `json:"$schema,omitempty"`
	Provider    OpenCodeProviderConfig       `json:"provider"`
	MCPServers  map[string]OpenCodeMCPServer `json:"mcpServers,omitempty"`
	Agent       map[string]OpenCodeAgent     `json:"agent,omitempty"`
	Tools       map[string]bool              `json:"tools,omitempty"`
	Permissions OpenCodePermissions          `json:"permission,omitempty"`
	Settings    OpenCodeSettings             `json:"settings,omitempty"`
}

// OpenCodeProviderConfig represents the provider section
type OpenCodeProviderConfig struct {
	Options OpenCodeProviderOptions `json:"options"`
}

// OpenCodeProviderOptions contains provider options
type OpenCodeProviderOptions struct {
	BaseURL      string                      `json:"baseURL,omitempty"`
	APIKey       string                      `json:"apiKey,omitempty"`
	APIKeyEnvVar string                      `json:"apiKeyEnvVar,omitempty"`
	Model        string                      `json:"model,omitempty"`
	Models       []OpenCodeModelDef          `json:"models,omitempty"`
}

// OpenCodeModelDef represents a model definition
type OpenCodeModelDef struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name,omitempty"`
	MaxTokens    int                    `json:"maxTokens,omitempty"`
	Capabilities OpenCodeCapabilities   `json:"capabilities,omitempty"`
}

// OpenCodeCapabilities represents model capabilities
type OpenCodeCapabilities struct {
	Vision        bool `json:"vision,omitempty"`
	ImageInput    bool `json:"imageInput,omitempty"`
	ImageOutput   bool `json:"imageOutput,omitempty"`
	OCR           bool `json:"ocr,omitempty"`
	PDF           bool `json:"pdf,omitempty"`
	Audio         bool `json:"audio,omitempty"`
	Video         bool `json:"video,omitempty"`
	Streaming     bool `json:"streaming,omitempty"`
	FunctionCalls bool `json:"functionCalls,omitempty"`
	ToolUse       bool `json:"toolUse,omitempty"`
	Embeddings    bool `json:"embeddings,omitempty"`
	CodeExecution bool `json:"codeExecution,omitempty"`
	WebBrowsing   bool `json:"webBrowsing,omitempty"`
	FileUpload    bool `json:"fileUpload,omitempty"`
	NoFileLimit   bool `json:"noFileLimit,omitempty"`
	MCP           bool `json:"mcp,omitempty"`
	ACP           bool `json:"acp,omitempty"`
	LSP           bool `json:"lsp,omitempty"`
}

// OpenCodeMCPServer represents an MCP server configuration for OpenCode
// OpenCode uses two formats:
// - Local (stdio): {"command": "npx", "args": ["-y", "package"]}
// - Remote (SSE): {"type": "sse", "url": "http://..."}
type OpenCodeMCPServer struct {
	// For local/stdio MCP servers
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`

	// For remote/SSE MCP servers
	Type string `json:"type,omitempty"` // "sse" for remote servers
	URL  string `json:"url,omitempty"`

	// Environment variables
	Env map[string]string `json:"env,omitempty"`
}

// OpenCodeAgent represents an agent configuration
type OpenCodeAgent struct {
	Model  string `json:"model,omitempty"`
	Prompt string `json:"prompt,omitempty"`
}

// OpenCodePermissions represents permissions configuration
type OpenCodePermissions struct {
	AllowRead   bool `json:"allowRead,omitempty"`
	AllowWrite  bool `json:"allowWrite,omitempty"`
	AllowExec   bool `json:"allowExec,omitempty"`
	AllowNet    bool `json:"allowNet,omitempty"`
}

// OpenCodeSettings represents settings configuration
type OpenCodeSettings struct {
	Theme           string `json:"theme,omitempty"`
	AutoSave        bool   `json:"autoSave,omitempty"`
	ConfirmActions  bool   `json:"confirmActions,omitempty"`
}

// OpenCodeGenerator generates OpenCode configurations
type OpenCodeGenerator struct {
	schema *AgentSchema
}

// NewOpenCodeGenerator creates a new OpenCode configuration generator
func NewOpenCodeGenerator() *OpenCodeGenerator {
	homeDir, _ := os.UserHomeDir()
	return &OpenCodeGenerator{
		schema: &AgentSchema{
			AgentType:        AgentOpenCode,
			SchemaURL:        "https://opencode.ai/config.json",
			ConfigFileName:   "opencode.json",
			ConfigDirEnvVar:  "OPENCODE_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "opencode"),
			SupportedFields: []string{
				"$schema", "provider", "mcpServers", "agent", "tools",
				"permission", "settings", "keybinds", "command", "tui",
			},
			RequiredFields: []string{"provider"},
			Description:    "OpenCode.ai CLI - AI-powered coding assistant",
		},
	}
}

// Generate generates an OpenCode configuration
func (g *OpenCodeGenerator) Generate(ctx context.Context, config *GeneratorConfig) (*GenerationResult, error) {
	result := &GenerationResult{
		AgentType:   AgentOpenCode,
		GeneratedAt: time.Now(),
	}

	// Build the configuration
	openCodeConfig := &OpenCodeConfig{
		Schema: g.schema.SchemaURL,
	}

	// Configure provider
	baseURL := fmt.Sprintf("http://%s:%d/v1", config.HelixAgentHost, config.HelixAgentPort)
	openCodeConfig.Provider = OpenCodeProviderConfig{
		Options: OpenCodeProviderOptions{
			BaseURL:      baseURL,
			APIKeyEnvVar: "HELIXAGENT_API_KEY",
			Models: []OpenCodeModelDef{
				{
					ID:        "helixagent-debate",
					Name:      "HelixAgent AI Debate Ensemble",
					MaxTokens: 128000,
					Capabilities: OpenCodeCapabilities{
						Vision:        true,
						ImageInput:    true,
						ImageOutput:   true,
						OCR:           true,
						PDF:           true,
						Streaming:     true,
						FunctionCalls: true,
						ToolUse:       true,
						Embeddings:    true,
						FileUpload:    true,
						NoFileLimit:   true,
						MCP:           true,
						ACP:           true,
						LSP:           true,
					},
				},
			},
		},
	}

	// Configure MCP servers in OpenCode format
	// OpenCode uses two formats:
	// - Local (stdio): {"command": "npx", "args": ["-y", "package"]}
	// - Remote (SSE): {"type": "sse", "url": "http://..."}
	openCodeConfig.MCPServers = make(map[string]OpenCodeMCPServer)
	for _, mcpServer := range config.MCPServers {
		mcp := OpenCodeMCPServer{}
		if mcpServer.Type == "remote" {
			// Remote servers use SSE type
			mcp.Type = "sse"
			mcp.URL = mcpServer.URL
		} else {
			// Local servers use command + args format
			if len(mcpServer.Command) > 0 {
				mcp.Command = mcpServer.Command[0]
				if len(mcpServer.Command) > 1 {
					mcp.Args = mcpServer.Command[1:]
				}
			}
			if len(mcpServer.Environment) > 0 {
				mcp.Env = mcpServer.Environment
			}
		}
		openCodeConfig.MCPServers[mcpServer.Name] = mcp
	}

	// Configure agents
	openCodeConfig.Agent = map[string]OpenCodeAgent{
		"default": {
			Model:  "helixagent-debate",
			Prompt: "You are a helpful AI coding assistant powered by HelixAgent AI Debate Ensemble.",
		},
		"code-reviewer": {
			Model:  "helixagent-debate",
			Prompt: "You are an expert code reviewer. Analyze code for bugs, security issues, and improvements.",
		},
		"embeddings": {
			Model:  "helixagent-debate",
			Prompt: "Generate embeddings for semantic search and similarity matching.",
		},
		"vision": {
			Model:  "helixagent-debate",
			Prompt: "Analyze images and visual content with detailed descriptions.",
		},
	}

	// Configure tools
	openCodeConfig.Tools = map[string]bool{
		"file":       true,
		"search":     true,
		"terminal":   true,
		"browser":    true,
		"mcp":        true,
		"embeddings": true,
		"vision":     true,
		"lsp":        true,
	}

	// Configure permissions
	openCodeConfig.Permissions = OpenCodePermissions{
		AllowRead:  true,
		AllowWrite: true,
		AllowExec:  true,
		AllowNet:   true,
	}

	result.Config = openCodeConfig
	result.Success = true

	return result, nil
}

// Validate validates an OpenCode configuration
func (g *OpenCodeGenerator) Validate(config any) (*ValidationResult, error) {
	result := &ValidationResult{Valid: true}

	openCodeConfig, ok := config.(*OpenCodeConfig)
	if !ok {
		// Try to cast from map
		if configMap, ok := config.(map[string]interface{}); ok {
			return g.validateMap(configMap)
		}
		result.Valid = false
		result.Errors = append(result.Errors, "invalid configuration type: expected *OpenCodeConfig")
		return result, nil
	}

	// Validate required fields
	if openCodeConfig.Provider.Options.BaseURL == "" && openCodeConfig.Provider.Options.APIKeyEnvVar == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "provider.options must have baseURL or apiKeyEnvVar")
	}

	// Validate MCP servers
	for name, mcp := range openCodeConfig.MCPServers {
		// Remote servers must have type "sse" and URL
		if mcp.Type == "sse" {
			if mcp.URL == "" {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("mcpServers.%s: url is required for sse type", name))
			}
		} else {
			// Local servers must have a command
			if mcp.Command == "" {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("mcpServers.%s: command is required for local type", name))
			}
		}
	}

	// Validate agents
	for name, agent := range openCodeConfig.Agent {
		if agent.Model == "" && agent.Prompt == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("agent.%s: has neither model nor prompt", name))
		}
	}

	return result, nil
}

// validateMap validates a configuration from a map (e.g., from JSON)
func (g *OpenCodeGenerator) validateMap(config map[string]interface{}) (*ValidationResult, error) {
	result := &ValidationResult{Valid: true}

	// Check for valid top-level keys
	validKeys := map[string]bool{
		"$schema": true, "plugin": true, "enterprise": true, "instructions": true,
		"provider": true, "mcpServers": true, "tools": true, "agent": true,
		"command": true, "keybinds": true, "username": true, "share": true,
		"permission": true, "compaction": true, "sse": true, "mode": true, "autoshare": true,
		"tui": true,
	}

	for key := range config {
		if !validKeys[key] {
			result.Warnings = append(result.Warnings, fmt.Sprintf("unknown top-level key: %s", key))
		}
	}

	// Check for required provider section
	provider, hasProvider := config["provider"]
	if !hasProvider {
		result.Valid = false
		result.Errors = append(result.Errors, "provider section is required")
		return result, nil
	}

	// Validate provider has options
	providerMap, ok := provider.(map[string]interface{})
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, "provider must be an object")
		return result, nil
	}

	if _, hasOptions := providerMap["options"]; !hasOptions {
		result.Valid = false
		result.Errors = append(result.Errors, "provider.options is required")
	}

	// Validate MCP servers if present
	if mcpServers, hasMCP := config["mcpServers"]; hasMCP {
		mcpMap, ok := mcpServers.(map[string]interface{})
		if !ok {
			result.Valid = false
			result.Errors = append(result.Errors, "mcpServers must be an object")
		} else {
			for name, server := range mcpMap {
				serverMap, ok := server.(map[string]interface{})
				if !ok {
					result.Valid = false
					result.Errors = append(result.Errors, fmt.Sprintf("mcpServers.%s must be an object", name))
					continue
				}

				// Check for SSE type (remote) vs local (command-based)
				serverType, hasType := serverMap["type"].(string)
				if hasType && serverType == "sse" {
					// Remote SSE server requires URL
					if _, hasURL := serverMap["url"]; !hasURL {
						result.Valid = false
						result.Errors = append(result.Errors, fmt.Sprintf("mcpServers.%s: url is required for sse type", name))
					}
				} else {
					// Local server requires command
					if _, hasCommand := serverMap["command"]; !hasCommand {
						result.Valid = false
						result.Errors = append(result.Errors, fmt.Sprintf("mcpServers.%s: command is required for local type", name))
					}
				}
			}
		}
	}

	return result, nil
}

// GetSchema returns the OpenCode configuration schema
func (g *OpenCodeGenerator) GetSchema() *AgentSchema {
	return g.schema
}
