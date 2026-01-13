// Package cliagents provides unified CLI agent configuration generation and validation.
package cliagents

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CrushConfig represents the configuration for Crush CLI
type CrushConfig struct {
	Version     string                    `json:"version,omitempty"`
	Provider    CrushProviderConfig       `json:"provider"`
	Models      []CrushModelDef           `json:"models,omitempty"`
	MCP         map[string]CrushMCP       `json:"mcp,omitempty"`
	Agents      map[string]CrushAgent     `json:"agents,omitempty"`
	Settings    CrushSettings             `json:"settings,omitempty"`
}

// CrushProviderConfig represents the provider configuration
type CrushProviderConfig struct {
	Name        string            `json:"name"`
	BaseURL     string            `json:"base_url"`
	APIKey      string            `json:"api_key,omitempty"`
	APIKeyEnv   string            `json:"api_key_env,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// CrushModelDef represents a model definition
type CrushModelDef struct {
	ID          string             `json:"id"`
	Name        string             `json:"name,omitempty"`
	MaxTokens   int                `json:"max_tokens,omitempty"`
	Capabilities []string          `json:"capabilities,omitempty"`
}

// CrushMCP represents an MCP server configuration
type CrushMCP struct {
	Type        string            `json:"type"`
	Command     []string          `json:"command,omitempty"`
	URL         string            `json:"url,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Environment map[string]string `json:"env,omitempty"`
}

// CrushAgent represents an agent configuration
type CrushAgent struct {
	Model       string `json:"model"`
	System      string `json:"system,omitempty"`
	MaxTokens   int    `json:"max_tokens,omitempty"`
}

// CrushSettings represents settings configuration
type CrushSettings struct {
	Theme           string `json:"theme,omitempty"`
	StreamResponses bool   `json:"stream_responses,omitempty"`
	ShowProgress    bool   `json:"show_progress,omitempty"`
	ConfirmCommands bool   `json:"confirm_commands,omitempty"`
}

// CrushGenerator generates Crush configurations
type CrushGenerator struct {
	schema *AgentSchema
}

// NewCrushGenerator creates a new Crush configuration generator
func NewCrushGenerator() *CrushGenerator {
	homeDir, _ := os.UserHomeDir()
	return &CrushGenerator{
		schema: &AgentSchema{
			AgentType:        AgentCrush,
			ConfigFileName:   "crush.json",
			ConfigDirEnvVar:  "CRUSH_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "crush"),
			SupportedFields: []string{
				"version", "provider", "models", "mcp", "agents", "settings",
			},
			RequiredFields: []string{"provider"},
			Description:    "Charm Land Crush CLI - AI coding assistant from Charm",
		},
	}
}

// Generate generates a Crush configuration
func (g *CrushGenerator) Generate(ctx context.Context, config *GeneratorConfig) (*GenerationResult, error) {
	result := &GenerationResult{
		AgentType:   AgentCrush,
		GeneratedAt: time.Now(),
	}

	// Build the configuration
	crushConfig := &CrushConfig{
		Version: "1.0",
	}

	// Configure provider
	baseURL := fmt.Sprintf("http://%s:%d/v1", config.HelixAgentHost, config.HelixAgentPort)
	crushConfig.Provider = CrushProviderConfig{
		Name:      "helixagent",
		BaseURL:   baseURL,
		APIKeyEnv: "HELIXAGENT_API_KEY",
	}

	// Configure models
	crushConfig.Models = []CrushModelDef{
		{
			ID:        "helixagent-debate",
			Name:      "HelixAgent AI Debate Ensemble",
			MaxTokens: 128000,
			Capabilities: []string{
				"vision", "streaming", "function_calls", "embeddings",
				"mcp", "acp", "lsp", "code_execution",
			},
		},
	}

	// Configure MCP servers
	crushConfig.MCP = make(map[string]CrushMCP)
	for _, mcpServer := range config.MCPServers {
		mcp := CrushMCP{
			Type: mcpServer.Type,
		}
		if mcpServer.Type == "remote" {
			mcp.URL = mcpServer.URL
		} else {
			mcp.Command = mcpServer.Command
			mcp.Args = mcpServer.Args
			mcp.Environment = mcpServer.Environment
		}
		crushConfig.MCP[mcpServer.Name] = mcp
	}

	// Configure agents
	crushConfig.Agents = map[string]CrushAgent{
		"default": {
			Model:     "helixagent-debate",
			System:    "You are a helpful AI coding assistant powered by HelixAgent AI Debate Ensemble.",
			MaxTokens: 8192,
		},
		"coder": {
			Model:     "helixagent-debate",
			System:    "You are an expert software developer. Write clean, efficient, and well-documented code.",
			MaxTokens: 16384,
		},
		"reviewer": {
			Model:     "helixagent-debate",
			System:    "You are a code reviewer. Analyze code for bugs, security issues, and best practices.",
			MaxTokens: 8192,
		},
	}

	// Configure settings
	crushConfig.Settings = CrushSettings{
		Theme:           "auto",
		StreamResponses: true,
		ShowProgress:    true,
		ConfirmCommands: true,
	}

	result.Config = crushConfig
	result.Success = true

	return result, nil
}

// Validate validates a Crush configuration
func (g *CrushGenerator) Validate(config any) (*ValidationResult, error) {
	result := &ValidationResult{Valid: true}

	crushConfig, ok := config.(*CrushConfig)
	if !ok {
		// Try to cast from map
		if configMap, ok := config.(map[string]interface{}); ok {
			return g.validateMap(configMap)
		}
		result.Valid = false
		result.Errors = append(result.Errors, "invalid configuration type: expected *CrushConfig")
		return result, nil
	}

	// Validate required fields
	if crushConfig.Provider.BaseURL == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "provider.base_url is required")
	}

	// Validate MCP servers
	for name, mcp := range crushConfig.MCP {
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
	for name, agent := range crushConfig.Agents {
		if agent.Model == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("agent.%s: model is not specified", name))
		}
	}

	return result, nil
}

// validateMap validates a configuration from a map
func (g *CrushGenerator) validateMap(config map[string]interface{}) (*ValidationResult, error) {
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

// GetSchema returns the Crush configuration schema
func (g *CrushGenerator) GetSchema() *AgentSchema {
	return g.schema
}
