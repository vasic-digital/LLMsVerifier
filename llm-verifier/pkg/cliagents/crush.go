// Package cliagents provides unified CLI agent configuration generation and validation.
package cliagents

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	crush_config "digital.vasic.llmsverifier/pkg/crush/config"
)

// CrushConfig represents the configuration for Crush CLI with HelixAgent extensions
type CrushConfig struct {
	*crush_config.Config
	Plugins    []string              `json:"plugins,omitempty"`
	Extensions *HelixAgentExtensions `json:"extensions,omitempty"`
	Formatters FormattersConfig      `json:"formatters,omitempty"`
}

// CrushGenerator generates Crush configurations matching the official schema
// Schema: https://charm.land/crush.json
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
				"$schema", "providers", "models", "mcp", "lsp", "options", "permissions", "tools",
			},
			RequiredFields: []string{"providers", "tools"},
			Description:    "Charm Land Crush CLI - AI coding assistant from Charm",
		},
	}
}

// Generate generates a Crush configuration matching the official schema
func (g *CrushGenerator) Generate(ctx context.Context, config *GeneratorConfig) (*GenerationResult, error) {
	result := &GenerationResult{
		AgentType:   AgentCrush,
		GeneratedAt: time.Now(),
	}

	// Build the configuration using the wrapper struct with HelixAgent extensions
	crushConfig := &CrushConfig{
		Config: &crush_config.Config{
			Schema: "https://charm.land/crush.json",
		},
		Plugins:    DefaultPlugins(),
		Extensions: DefaultHelixAgentExtensions(config.HelixAgentHost, config.HelixAgentPort),
		Formatters: DefaultFormattersConfig(config.HelixAgentHost, config.HelixAgentPort),
	}

	// Get API key from environment for installed configs
	// CLI agents do NOT support env var references like "$HELIXAGENT_API_KEY"
	apiKey := os.Getenv("HELIXAGENT_API_KEY")
	if apiKey == "" {
		apiKey = "<YOUR_HELIXAGENT_API_KEY>"
	}

	baseURL := fmt.Sprintf("http://%s:%d/v1", config.HelixAgentHost, config.HelixAgentPort)

	// Configure HelixLLM endpoint
	helixLLMHost := config.HelixLLMHost
	if helixLLMHost == "" {
		helixLLMHost = "localhost"
	}
	helixLLMPort := config.HelixLLMPort
	if helixLLMPort == 0 {
		helixLLMPort = 8443
	}
	helixLLMBaseURL := fmt.Sprintf("https://%s:%d/v1", helixLLMHost, helixLLMPort)

	// Configure providers (map by provider ID)
	crushConfig.Providers = map[string]crush_config.ProviderConfig{
		"helixagent": {
			ID:      "helixagent",
			Name:    "HelixAgent",
			Type:    "openai-compat",
			BaseURL: baseURL,
			APIKey:  apiKey,
			Models: []crush_config.Model{
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
					Options: &crush_config.ModelOptions{
						Temperature: 0.7,
					},
				},
			},
		},
		"helixllm": {
			ID:      "helixllm",
			Name:    "HelixLLM",
			Type:    "openai-compat",
			BaseURL: helixLLMBaseURL,
			APIKey:  config.HelixLLMAPIKey,
			Models: []crush_config.Model{
				{
					ID:                  "deepseek-chat",
					Name:                "HelixLLM",
					CostPer1MIn:         0,
					CostPer1MOut:        0,
					CostPer1MInCached:   0,
					CostPer1MOutCached:  0,
					ContextWindow:       128000,
					DefaultMaxTokens:    8192,
					CanReason:           true,
					SupportsAttachments: true,
					Options: &crush_config.ModelOptions{
						Temperature: 0.7,
					},
				},
			},
		},
	}

	// Configure models (map by model type/role to SelectedModel)
	crushConfig.Models = map[string]crush_config.SelectedModel{
		"default": {
			Model:       "helixagent-debate",
			Provider:    "helixagent",
			MaxTokens:   8192,
			Temperature: 0.7,
		},
		"large": {
			Model:     "helixagent-debate",
			Provider:  "helixagent",
			MaxTokens: 32768,
		},
		"reasoning": {
			Model:           "helixagent-debate",
			Provider:        "helixagent",
			MaxTokens:       16384,
			ReasoningEffort: "high",
		},
	}

	// Configure MCP servers (15+ out of the box)
	crushConfig.MCP = make(crush_config.MCPs)
	for _, mcpServer := range config.MCPServers {
		mcp := crush_config.MCPConfig{
			Timeout: 30,
		}

		switch mcpServer.Type {
		case "remote", "sse", "http":
			mcp.Type = "sse"
			mcp.URL = mcpServer.URL
		default:
			mcp.Type = "stdio"
			mcp.Command = mcpServer.Command[0]
			if len(mcpServer.Command) > 1 {
				mcp.Args = mcpServer.Command[1:]
			}
			if mcpServer.Args != nil {
				mcp.Args = append(mcp.Args, mcpServer.Args...)
			}
			mcp.Env = mcpServer.Environment
		}

		crushConfig.MCP[mcpServer.Name] = mcp
	}

	// Configure LSP servers
	crushConfig.LSP = make(crush_config.LSPs)
	lspConfigs := map[string]crush_config.LSPConfig{
		"go": {
			Command:     "gopls",
			Filetypes:   []string{"go"},
			RootMarkers: []string{"go.mod", "go.sum"},
			Timeout:     30,
		},
		"typescript": {
			Command:     "typescript-language-server",
			Args:        []string{"--stdio"},
			Filetypes:   []string{"typescript", "javascript", "typescriptreact", "javascriptreact"},
			RootMarkers: []string{"package.json", "tsconfig.json"},
			Timeout:     30,
		},
		"python": {
			Command:     "pylsp",
			Filetypes:   []string{"python"},
			RootMarkers: []string{"pyproject.toml", "setup.py", "requirements.txt"},
			Timeout:     30,
		},
		"rust": {
			Command:     "rust-analyzer",
			Filetypes:   []string{"rust"},
			RootMarkers: []string{"Cargo.toml"},
			Timeout:     30,
		},
	}
	for name, lsp := range lspConfigs {
		crushConfig.LSP[name] = lsp
	}

	// Configure tools (required field)
	crushConfig.Tools = &crush_config.Tools{
		LS: &crush_config.ToolLS{
			MaxDepth: 10,
			MaxItems: 1000,
		},
		Grep: &crush_config.ToolGrep{
			Timeout: 60,
		},
	}

	// Configure options
	crushConfig.Options = &crush_config.Options{
		AutoLSP:       true,
		Progress:      true,
		Debug:         false,
		DataDirectory: ".crush",
		InitializeAs:  "AGENTS.md",
		ContextPaths:  []string{"AGENTS.md", "CLAUDE.md", ".cursorrules"},
		SkillsPaths:   []string{"~/.config/crush/skills"},
		Attribution: &crush_config.Attribution{
			TrailerStyle:  "assisted-by",
			GeneratedWith: true,
		},
		DisableMetrics: false,
		TUI: &crush_config.TUIOptions{
			CompactMode: false,
			DiffMode:    "unified",
			Completions: &crush_config.Completions{
				MaxDepth: 10,
				MaxItems: 1000,
			},
		},
	}

	// Configure permissions
	crushConfig.Permissions = &crush_config.Permissions{
		AllowedTools: []string{
			"bash", "view", "edit", "glob", "grep", "ls",
			"sourcegraph", "web_search", "web_fetch",
		},
	}

	result.Config = crushConfig
	result.Success = true

	return result, nil
}

// Validate validates a Crush configuration
func (g *CrushGenerator) Validate(config any) (*ValidationResult, error) {
	result := &ValidationResult{Valid: true}

	var crushConfig *crush_config.Config

	switch c := config.(type) {
	case *CrushConfig:
		crushConfig = c.Config
	case *crush_config.Config:
		crushConfig = c
	default:
		// Try to cast from map
		if configMap, ok := config.(map[string]interface{}); ok {
			return g.validateMap(configMap)
		}
		result.Valid = false
		result.Errors = append(result.Errors, "invalid configuration type: expected *CrushConfig or *crush_config.Config")
		return result, nil
	}

	// Validate required providers
	if len(crushConfig.Providers) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "providers section is required and must have at least one provider")
	}

	// Validate each provider has required fields
	for id, provider := range crushConfig.Providers {
		if provider.Type == "" {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("providers.%s: type is required", id))
		}
		if provider.BaseURL == "" && provider.Type != "openai" {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("providers.%s: base_url is required for non-openai providers", id))
		}
	}

	// Validate required tools section
	if crushConfig.Tools == nil {
		result.Valid = false
		result.Errors = append(result.Errors, "tools section is required")
	} else {
		if crushConfig.Tools.LS == nil {
			result.Valid = false
			result.Errors = append(result.Errors, "tools.ls is required")
		}
		if crushConfig.Tools.Grep == nil {
			result.Valid = false
			result.Errors = append(result.Errors, "tools.grep is required")
		}
	}

	// Validate models reference existing providers
	for name, model := range crushConfig.Models {
		if model.Provider == "" {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("models.%s: provider is required", name))
		} else if _, exists := crushConfig.Providers[model.Provider]; !exists {
			result.Warnings = append(result.Warnings, fmt.Sprintf("models.%s: provider '%s' not found in providers section", name, model.Provider))
		}
		if model.Model == "" {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("models.%s: model is required", name))
		}
	}

	// Validate MCP servers
	for name, mcp := range crushConfig.MCP {
		if mcp.Type == "" {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("mcp.%s: type is required", name))
		}
		validTypes := map[string]bool{"stdio": true, "sse": true, "http": true}
		if !validTypes[mcp.Type] {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("mcp.%s: type must be 'stdio', 'sse', or 'http'", name))
		}
		if mcp.Type == "stdio" && mcp.Command == "" {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("mcp.%s: command is required for stdio type", name))
		}
		if (mcp.Type == "sse" || mcp.Type == "http") && mcp.URL == "" {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("mcp.%s: url is required for sse/http type", name))
		}
	}

	// Validate LSP servers
	for name, lsp := range crushConfig.LSP {
		if lsp.Command == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("lsp.%s: command is empty", name))
		}
	}

	return result, nil
}

// validateMap validates a configuration from a map
func (g *CrushGenerator) validateMap(config map[string]interface{}) (*ValidationResult, error) {
	result := &ValidationResult{Valid: true}

	// Check for required providers section
	providers, hasProviders := config["providers"]
	if !hasProviders {
		result.Valid = false
		result.Errors = append(result.Errors, "providers section is required")
		return result, nil
	}

	providersMap, ok := providers.(map[string]interface{})
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, "providers must be an object")
		return result, nil
	}

	if len(providersMap) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "providers must have at least one provider")
	}

	// Validate each provider
	for id, provider := range providersMap {
		providerMap, ok := provider.(map[string]interface{})
		if !ok {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("providers.%s must be an object", id))
			continue
		}

		if providerType, hasType := providerMap["type"]; !hasType {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("providers.%s: type is required", id))
		} else if _, ok := providerType.(string); !ok {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("providers.%s: type must be a string", id))
		}
	}

	// Check for required tools section
	tools, hasTools := config["tools"]
	if !hasTools {
		result.Valid = false
		result.Errors = append(result.Errors, "tools section is required")
		return result, nil
	}

	toolsMap, ok := tools.(map[string]interface{})
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, "tools must be an object")
		return result, nil
	}

	if _, hasLS := toolsMap["ls"]; !hasLS {
		result.Valid = false
		result.Errors = append(result.Errors, "tools.ls is required")
	}
	if _, hasGrep := toolsMap["grep"]; !hasGrep {
		result.Valid = false
		result.Errors = append(result.Errors, "tools.grep is required")
	}

	// Validate models if present
	if models, hasModels := config["models"]; hasModels {
		modelsMap, ok := models.(map[string]interface{})
		if !ok {
			result.Valid = false
			result.Errors = append(result.Errors, "models must be an object")
		} else {
			for name, model := range modelsMap {
				modelMap, ok := model.(map[string]interface{})
				if !ok {
					result.Valid = false
					result.Errors = append(result.Errors, fmt.Sprintf("models.%s must be an object", name))
					continue
				}
				if _, hasModel := modelMap["model"]; !hasModel {
					result.Valid = false
					result.Errors = append(result.Errors, fmt.Sprintf("models.%s: model is required", name))
				}
				if _, hasProvider := modelMap["provider"]; !hasProvider {
					result.Valid = false
					result.Errors = append(result.Errors, fmt.Sprintf("models.%s: provider is required", name))
				}
			}
		}
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

				validTypes := map[string]bool{"stdio": true, "sse": true, "http": true}
				if !validTypes[serverType] {
					result.Valid = false
					result.Errors = append(result.Errors, fmt.Sprintf("mcp.%s: type must be 'stdio', 'sse', or 'http'", name))
				}

				if serverType == "stdio" {
					if _, hasCommand := serverMap["command"]; !hasCommand {
						result.Valid = false
						result.Errors = append(result.Errors, fmt.Sprintf("mcp.%s: command is required for stdio type", name))
					}
				} else if serverType == "sse" || serverType == "http" {
					if _, hasURL := serverMap["url"]; !hasURL {
						result.Valid = false
						result.Errors = append(result.Errors, fmt.Sprintf("mcp.%s: url is required for sse/http type", name))
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
