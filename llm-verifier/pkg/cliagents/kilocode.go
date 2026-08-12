// Package cliagents provides unified CLI agent configuration generation and validation.
package cliagents

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"digital.vasic.llmsverifier/pkg/helixendpoint"
)

// KiloCodeConfig represents the configuration for KiloCode VS Code extension
type KiloCodeConfig struct {
	Version             string                            `json:"version,omitempty"`
	Provider            KiloCodeProviderConfig            `json:"provider"`
	AdditionalProviders map[string]KiloCodeProviderConfig `json:"additionalProviders,omitempty"`
	Models              []KiloCodeModelDef                `json:"models,omitempty"`
	MCP                 map[string]KiloCodeMCP            `json:"mcpServers,omitempty"`
	Plugins             []string                          `json:"plugins,omitempty"`
	Agents              []KiloCodeAgent                   `json:"agents,omitempty"`
	Settings            KiloCodeSettings                  `json:"settings,omitempty"`
	Extensions          *HelixAgentExtensions             `json:"extensions,omitempty"`
	Formatters          FormattersConfig                  `json:"formatters,omitempty"`
}

// KiloCodeProviderConfig represents the provider configuration
type KiloCodeProviderConfig struct {
	Type      string `json:"type"`
	BaseURL   string `json:"baseUrl"`
	APIKey    string `json:"apiKey,omitempty"`
	APIKeyEnv string `json:"apiKeyEnv,omitempty"`
}

// KiloCodeModelDef represents a model definition
type KiloCodeModelDef struct {
	ID              string           `json:"id"`
	Name            string           `json:"name,omitempty"`
	MaxInputTokens  int              `json:"maxInputTokens,omitempty"`
	MaxOutputTokens int              `json:"maxOutputTokens,omitempty"`
	Capabilities    KiloCapabilities `json:"capabilities,omitempty"`
}

// KiloCapabilities represents model capabilities
type KiloCapabilities struct {
	Streaming     bool `json:"streaming,omitempty"`
	Vision        bool `json:"vision,omitempty"`
	FunctionCalls bool `json:"functionCalls,omitempty"`
	Embeddings    bool `json:"embeddings,omitempty"`
}

// KiloCodeMCP represents an MCP server configuration
type KiloCodeMCP struct {
	Transport   string            `json:"transport"` // stdio, sse
	Command     string            `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	URL         string            `json:"url,omitempty"`
	Environment map[string]string `json:"env,omitempty"`
}

// KiloCodeAgent represents an agent configuration
type KiloCodeAgent struct {
	Name      string `json:"name"`
	Model     string `json:"model"`
	Prompt    string `json:"prompt,omitempty"`
	MaxTokens int    `json:"maxTokens,omitempty"`
}

// KiloCodeSettings represents settings configuration
type KiloCodeSettings struct {
	AutoComplete     bool   `json:"autoComplete,omitempty"`
	InlineCompletion bool   `json:"inlineCompletion,omitempty"`
	StreamResponses  bool   `json:"streamResponses,omitempty"`
	ContextWindow    int    `json:"contextWindow,omitempty"`
	Theme            string `json:"theme,omitempty"`
}

// KiloCodeGenerator generates KiloCode configurations
type KiloCodeGenerator struct {
	schema *AgentSchema
}

// NewKiloCodeGenerator creates a new KiloCode configuration generator
func NewKiloCodeGenerator() *KiloCodeGenerator {
	homeDir, _ := os.UserHomeDir()
	return &KiloCodeGenerator{
		schema: &AgentSchema{
			AgentType:        AgentKiloCode,
			ConfigFileName:   "kilocode-settings.json",
			ConfigDirEnvVar:  "KILOCODE_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".vscode", "extensions", "kilocode"),
			SupportedFields: []string{
				"version", "provider", "models", "mcpServers", "agents", "settings",
			},
			RequiredFields: []string{"provider"},
			Description:    "KiloCode VS Code extension - AI-powered code completion and assistance",
		},
	}
}

// Generate generates a KiloCode configuration
func (g *KiloCodeGenerator) Generate(ctx context.Context, config *GeneratorConfig) (*GenerationResult, error) {
	result := &GenerationResult{
		AgentType:   AgentKiloCode,
		GeneratedAt: time.Now(),
	}

	// Build the configuration
	kiloConfig := &KiloCodeConfig{
		Version: "1.0",
	}

	// Configure provider
	baseURL := helixendpoint.JoinPath(
		helixendpoint.BaseURL(config.HelixAgentHost, config.HelixAgentPort), "v1")
	// Use real API key from environment for installed configs
	apiKey := os.Getenv("HELIXAGENT_API_KEY")
	if apiKey == "" {
		apiKey = "<YOUR_HELIXAGENT_API_KEY>"
	}

	// Single "Helix Agent" provider with TWO models:
	// - helix-debate: Helix AI Debate Ensemble (for coder, summarizer)
	// - helix-llm: Helix LLM (for task, title - with provider chain fallback)
	kiloConfig.Provider = KiloCodeProviderConfig{
		Type:    "openai-compatible",
		BaseURL: baseURL,
		APIKey:  apiKey,
	}

	// Configure models
	kiloConfig.Models = []KiloCodeModelDef{
		{
			ID:              "helix-llm",
			Name:            "Helix LLM",
			MaxInputTokens:  128000,
			MaxOutputTokens: 8192,
			Capabilities: KiloCapabilities{
				Streaming:     true,
				Vision:        true,
				FunctionCalls: true,
				Embeddings:    true,
			},
		},
		{
			ID:              "helix-debate",
			Name:            "Helix AI Debate Ensemble",
			MaxInputTokens:  128000,
			MaxOutputTokens: 16384,
			Capabilities: KiloCapabilities{
				Streaming:     true,
				Vision:        true,
				FunctionCalls: true,
				Embeddings:    true,
			},
		},
	}

	// Configure MCP servers (15+ out of the box)
	kiloConfig.MCP = make(map[string]KiloCodeMCP)
	for _, mcpServer := range config.MCPServers {
		mcp := KiloCodeMCP{}
		if mcpServer.Type == "remote" {
			mcp.Transport = "sse"
			mcp.URL = mcpServer.URL
		} else {
			mcp.Transport = "stdio"
			if len(mcpServer.Command) > 0 {
				mcp.Command = mcpServer.Command[0]
				if len(mcpServer.Command) > 1 {
					mcp.Args = mcpServer.Command[1:]
				}
			}
			mcp.Environment = mcpServer.Environment
		}
		kiloConfig.MCP[mcpServer.Name] = mcp
	}

	// Configure plugins
	kiloConfig.Plugins = DefaultPlugins()

	// Configure agents
	// Agent routing: task/title → helix-llm, coder/summarizer → helix-debate
	kiloConfig.Agents = []KiloCodeAgent{
		{
			Name:      "default",
			Model:     "helix-debate",
			Prompt:    "You are a helpful AI coding assistant integrated into VS Code.",
			MaxTokens: 8192,
		},
		{
			Name:      "coder",
			Model:     "helix-debate",
			Prompt:    "You are an expert software developer. Write clean, efficient, well-tested code.",
			MaxTokens: 16384,
		},
		{
			Name:      "summarizer",
			Model:     "helix-debate",
			Prompt:    "You are a summarization assistant. Create concise summaries of content.",
			MaxTokens: 4096,
		},
		{
			Name:      "completer",
			Model:     "helix-llm",
			Prompt:    "Complete the code following the cursor. Return only the completion.",
			MaxTokens: 1024,
		},
		{
			Name:      "explainer",
			Model:     "helix-debate",
			Prompt:    "Explain the selected code clearly and concisely.",
			MaxTokens: 4096,
		},
		{
			Name:      "task",
			Model:     "helix-llm",
			Prompt:    "Execute tasks efficiently with provider chain fallback for reliability.",
			MaxTokens: 4096,
		},
		{
			Name:      "title",
			Model:     "helix-llm",
			Prompt:    "Create concise, descriptive titles.",
			MaxTokens: 80,
		},
	}

	// Configure settings
	kiloConfig.Settings = KiloCodeSettings{
		AutoComplete:     true,
		InlineCompletion: true,
		StreamResponses:  true,
		ContextWindow:    128000,
		Theme:            "auto",
	}

	// Configure extensions (LSP, ACP, Embeddings, RAG, Skills)
	kiloConfig.Extensions = DefaultHelixAgentExtensions(config.HelixAgentHost, config.HelixAgentPort)

	// Configure formatters
	kiloConfig.Formatters = DefaultFormattersConfig(config.HelixAgentHost, config.HelixAgentPort)

	result.Config = kiloConfig
	result.Success = true

	return result, nil
}

// Validate validates a KiloCode configuration
func (g *KiloCodeGenerator) Validate(config any) (*ValidationResult, error) {
	result := &ValidationResult{Valid: true}

	kiloConfig, ok := config.(*KiloCodeConfig)
	if !ok {
		// Try to cast from map
		if configMap, ok := config.(map[string]interface{}); ok {
			return g.validateMap(configMap)
		}
		result.Valid = false
		result.Errors = append(result.Errors, "invalid configuration type: expected *KiloCodeConfig")
		return result, nil
	}

	// Validate required fields
	if kiloConfig.Provider.BaseURL == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "provider.baseUrl is required")
	}

	// Validate MCP servers
	for name, mcp := range kiloConfig.MCP {
		if mcp.Transport == "" {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("mcpServers.%s: transport is required", name))
		}
		if mcp.Transport == "sse" && mcp.URL == "" {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("mcpServers.%s: url is required for sse transport", name))
		}
		if mcp.Transport == "stdio" && mcp.Command == "" {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("mcpServers.%s: command is required for stdio transport", name))
		}
	}

	// Validate agents
	for _, agent := range kiloConfig.Agents {
		if agent.Name == "" {
			result.Valid = false
			result.Errors = append(result.Errors, "agent name is required")
		}
		if agent.Model == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("agent %s: model is not specified", agent.Name))
		}
	}

	return result, nil
}

// validateMap validates a configuration from a map
func (g *KiloCodeGenerator) validateMap(config map[string]interface{}) (*ValidationResult, error) {
	result := &ValidationResult{Valid: true}

	// Check for required provider section
	provider, hasProvider := config["provider"]
	if !hasProvider {
		result.Valid = false
		result.Errors = append(result.Errors, "provider section is required")
		return result, nil
	}

	// Validate provider has baseUrl
	providerMap, ok := provider.(map[string]interface{})
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, "provider must be an object")
		return result, nil
	}

	if _, hasBaseURL := providerMap["baseUrl"]; !hasBaseURL {
		result.Valid = false
		result.Errors = append(result.Errors, "provider.baseUrl is required")
	}

	// Validate MCP servers if present
	if mcp, hasMCP := config["mcpServers"]; hasMCP {
		mcpMap, ok := mcp.(map[string]interface{})
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

				transport, hasTransport := serverMap["transport"].(string)
				if !hasTransport {
					result.Valid = false
					result.Errors = append(result.Errors, fmt.Sprintf("mcpServers.%s: transport is required", name))
				}

				if transport == "sse" {
					if _, hasURL := serverMap["url"]; !hasURL {
						result.Valid = false
						result.Errors = append(result.Errors, fmt.Sprintf("mcpServers.%s: url is required for sse transport", name))
					}
				}
			}
		}
	}

	return result, nil
}

// GetSchema returns the KiloCode configuration schema
func (g *KiloCodeGenerator) GetSchema() *AgentSchema {
	return g.schema
}
