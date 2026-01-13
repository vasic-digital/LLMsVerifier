// Package cliagents provides unified CLI agent configuration generation and validation.
// This is the central authority for all CLI agent configuration generation in HelixAgent.
// All 16+ CLI agents should have their configurations generated and validated through this package.
package cliagents

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SupportedAgents lists all CLI agents supported by LLMsVerifier
var SupportedAgents = []string{
	"opencode",    // OpenCode.ai CLI
	"crush",       // Charm Land Crush CLI
	"kilocode",    // KiloCode VS Code extension
	"helixcode",   // HelixCode CLI
	"aider",       // Aider CLI
	"continue",    // Continue.dev extension
	"cursor",      // Cursor editor
	"cline",       // Cline CLI
	"windsurf",    // Windsurf editor
	"zed",         // Zed editor
	"neovim-ai",   // Neovim AI plugins
	"vscode-ai",   // VS Code AI extensions
	"intellij-ai", // IntelliJ AI plugins
	"claude-code", // Claude Code CLI
	"qwen-code",   // Qwen Code CLI
	"github-copilot", // GitHub Copilot
}

// AgentType represents the type of CLI agent
type AgentType string

const (
	AgentOpenCode    AgentType = "opencode"
	AgentCrush       AgentType = "crush"
	AgentKiloCode    AgentType = "kilocode"
	AgentHelixCode   AgentType = "helixcode"
	AgentAider       AgentType = "aider"
	AgentContinue    AgentType = "continue"
	AgentCursor      AgentType = "cursor"
	AgentCline       AgentType = "cline"
	AgentWindsurf    AgentType = "windsurf"
	AgentZed         AgentType = "zed"
	AgentNeovimAI    AgentType = "neovim-ai"
	AgentVSCodeAI    AgentType = "vscode-ai"
	AgentIntelliJAI  AgentType = "intellij-ai"
	AgentClaudeCode  AgentType = "claude-code"
	AgentQwenCode    AgentType = "qwen-code"
	AgentCopilot     AgentType = "github-copilot"
)

// GeneratorConfig contains configuration for CLI agent config generation
type GeneratorConfig struct {
	// HelixAgent endpoint configuration
	HelixAgentHost string
	HelixAgentPort int

	// Output directory for generated configs
	OutputDir string

	// Provider configurations
	Providers []ProviderConfig

	// MCP server configurations
	MCPServers []MCPServerConfig

	// Agent-specific configurations
	AgentConfigs map[AgentType]AgentSpecificConfig

	// Whether to include debug information
	Debug bool

	// Include verification scores in output
	IncludeScores bool
}

// ProviderConfig represents a provider configuration
type ProviderConfig struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	BaseURL      string            `json:"base_url,omitempty"`
	APIKey       string            `json:"api_key,omitempty"`
	APIKeyEnvVar string            `json:"api_key_env_var,omitempty"`
	Model        string            `json:"model,omitempty"`
	Models       []ModelConfig     `json:"models,omitempty"`
	Options      map[string]any    `json:"options,omitempty"`
	Score        float64           `json:"score,omitempty"`
	Verified     bool              `json:"verified,omitempty"`
}

// ModelConfig represents a model configuration
type ModelConfig struct {
	ID          string   `json:"id"`
	Name        string   `json:"name,omitempty"`
	MaxTokens   int      `json:"max_tokens,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Score       float64  `json:"score,omitempty"`
}

// MCPServerConfig represents an MCP server configuration
type MCPServerConfig struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"` // local, remote, stdio
	Command     []string          `json:"command,omitempty"`
	URL         string            `json:"url,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
}

// AgentSpecificConfig holds agent-specific settings
type AgentSpecificConfig struct {
	Enabled     bool           `json:"enabled"`
	Agents      []AgentDef     `json:"agents,omitempty"`
	Tools       map[string]any `json:"tools,omitempty"`
	Settings    map[string]any `json:"settings,omitempty"`
	Permissions map[string]any `json:"permissions,omitempty"`
}

// AgentDef represents an agent definition
type AgentDef struct {
	Name        string `json:"name"`
	Model       string `json:"model,omitempty"`
	Prompt      string `json:"prompt,omitempty"`
	SystemPrompt string `json:"system_prompt,omitempty"`
}

// GenerationResult contains the result of configuration generation
type GenerationResult struct {
	AgentType    AgentType         `json:"agent_type"`
	Success      bool              `json:"success"`
	ConfigPath   string            `json:"config_path,omitempty"`
	Config       any               `json:"config,omitempty"`
	Errors       []string          `json:"errors,omitempty"`
	Warnings     []string          `json:"warnings,omitempty"`
	GeneratedAt  time.Time         `json:"generated_at"`
	ValidationResult *ValidationResult `json:"validation,omitempty"`
}

// ValidationResult contains validation results
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// UnifiedGenerator generates configurations for all CLI agents
type UnifiedGenerator struct {
	config     *GeneratorConfig
	generators map[AgentType]AgentGenerator
}

// AgentGenerator is the interface for agent-specific generators
type AgentGenerator interface {
	Generate(ctx context.Context, config *GeneratorConfig) (*GenerationResult, error)
	Validate(config any) (*ValidationResult, error)
	GetSchema() *AgentSchema
}

// AgentSchema describes the configuration schema for an agent
type AgentSchema struct {
	AgentType        AgentType         `json:"agent_type"`
	SchemaURL        string            `json:"schema_url,omitempty"`
	ConfigFileName   string            `json:"config_file_name"`
	ConfigDirEnvVar  string            `json:"config_dir_env_var,omitempty"`
	DefaultConfigDir string            `json:"default_config_dir"`
	SupportedFields  []string          `json:"supported_fields"`
	RequiredFields   []string          `json:"required_fields"`
	Description      string            `json:"description"`
}

// NewUnifiedGenerator creates a new unified configuration generator
func NewUnifiedGenerator(config *GeneratorConfig) *UnifiedGenerator {
	if config == nil {
		config = DefaultGeneratorConfig()
	}

	ug := &UnifiedGenerator{
		config:     config,
		generators: make(map[AgentType]AgentGenerator),
	}

	// Register all supported generators
	ug.registerGenerators()

	return ug
}

// DefaultGeneratorConfig returns default generator configuration
func DefaultGeneratorConfig() *GeneratorConfig {
	homeDir, _ := os.UserHomeDir()
	return &GeneratorConfig{
		HelixAgentHost: "localhost",
		HelixAgentPort: 7061,
		OutputDir:      filepath.Join(homeDir, "Downloads"),
		IncludeScores:  true,
		MCPServers:     DefaultMCPServers(),
	}
}

// DefaultMCPServers returns default MCP server configurations
func DefaultMCPServers() []MCPServerConfig {
	return []MCPServerConfig{
		// HelixAgent MCP endpoints
		{Name: "helixagent-mcp", Type: "remote", URL: "http://localhost:7061/v1/mcp"},
		{Name: "helixagent-acp", Type: "remote", URL: "http://localhost:7061/v1/acp"},
		{Name: "helixagent-lsp", Type: "remote", URL: "http://localhost:7061/v1/lsp"},
		{Name: "helixagent-embeddings", Type: "remote", URL: "http://localhost:7061/v1/embeddings"},
		{Name: "helixagent-vision", Type: "remote", URL: "http://localhost:7061/v1/vision"},
		{Name: "helixagent-cognee", Type: "remote", URL: "http://localhost:7061/v1/cognee"},
		// Standard MCP servers
		{Name: "filesystem", Type: "local", Command: []string{"npx", "-y", "@modelcontextprotocol/server-filesystem"}},
		{Name: "github", Type: "local", Command: []string{"npx", "-y", "@modelcontextprotocol/server-github"}},
		{Name: "memory", Type: "local", Command: []string{"npx", "-y", "@modelcontextprotocol/server-memory"}},
		{Name: "fetch", Type: "local", Command: []string{"npx", "-y", "mcp-fetch"}},
		{Name: "puppeteer", Type: "local", Command: []string{"npx", "-y", "@modelcontextprotocol/server-puppeteer"}},
		{Name: "sqlite", Type: "local", Command: []string{"npx", "-y", "mcp-server-sqlite"}},
	}
}

// registerGenerators registers all agent-specific generators
func (ug *UnifiedGenerator) registerGenerators() {
	// Register generators for primary CLI agents (custom implementations)
	ug.generators[AgentOpenCode] = NewOpenCodeGenerator()
	ug.generators[AgentCrush] = NewCrushGenerator()
	ug.generators[AgentKiloCode] = NewKiloCodeGenerator()
	ug.generators[AgentHelixCode] = NewHelixCodeGenerator()

	// Register generators for additional CLI agents (generic implementations)
	ug.generators[AgentAider] = NewAiderGenerator()
	ug.generators[AgentContinue] = NewContinueGenerator()
	ug.generators[AgentCursor] = NewCursorGenerator()
	ug.generators[AgentCline] = NewClineGenerator()
	ug.generators[AgentWindsurf] = NewWindsurfGenerator()
	ug.generators[AgentZed] = NewZedGenerator()
	ug.generators[AgentNeovimAI] = NewNeovimAIGenerator()
	ug.generators[AgentVSCodeAI] = NewVSCodeAIGenerator()
	ug.generators[AgentIntelliJAI] = NewIntelliJAIGenerator()
	ug.generators[AgentClaudeCode] = NewClaudeCodeGenerator()
	ug.generators[AgentQwenCode] = NewQwenCodeGenerator()
	ug.generators[AgentCopilot] = NewCopilotGenerator()
}

// Generate generates configuration for a specific agent
func (ug *UnifiedGenerator) Generate(ctx context.Context, agentType AgentType) (*GenerationResult, error) {
	generator, ok := ug.generators[agentType]
	if !ok {
		return nil, fmt.Errorf("unsupported agent type: %s", agentType)
	}

	result, err := generator.Generate(ctx, ug.config)
	if err != nil {
		return nil, err
	}

	// Validate the generated config
	if result.Config != nil {
		validationResult, _ := generator.Validate(result.Config)
		result.ValidationResult = validationResult
	}

	return result, nil
}

// GenerateAll generates configurations for all supported agents
func (ug *UnifiedGenerator) GenerateAll(ctx context.Context) ([]*GenerationResult, error) {
	var results []*GenerationResult

	for agentType := range ug.generators {
		result, err := ug.Generate(ctx, agentType)
		if err != nil {
			results = append(results, &GenerationResult{
				AgentType: agentType,
				Success:   false,
				Errors:    []string{err.Error()},
			})
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

// Validate validates a configuration for a specific agent
func (ug *UnifiedGenerator) Validate(agentType AgentType, config any) (*ValidationResult, error) {
	generator, ok := ug.generators[agentType]
	if !ok {
		return nil, fmt.Errorf("unsupported agent type: %s", agentType)
	}

	return generator.Validate(config)
}

// GetSchema returns the configuration schema for a specific agent
func (ug *UnifiedGenerator) GetSchema(agentType AgentType) (*AgentSchema, error) {
	generator, ok := ug.generators[agentType]
	if !ok {
		return nil, fmt.Errorf("unsupported agent type: %s", agentType)
	}

	return generator.GetSchema(), nil
}

// GetAllSchemas returns configuration schemas for all supported agents
func (ug *UnifiedGenerator) GetAllSchemas() map[AgentType]*AgentSchema {
	schemas := make(map[AgentType]*AgentSchema)
	for agentType, generator := range ug.generators {
		schemas[agentType] = generator.GetSchema()
	}
	return schemas
}

// SaveConfig saves a generated configuration to file
func (ug *UnifiedGenerator) SaveConfig(result *GenerationResult) error {
	if result.Config == nil {
		return fmt.Errorf("no configuration to save")
	}

	data, err := json.MarshalIndent(result.Config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Determine output path
	generator := ug.generators[result.AgentType]
	schema := generator.GetSchema()

	outputPath := filepath.Join(ug.config.OutputDir, schema.ConfigFileName)
	result.ConfigPath = outputPath

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ListSupportedAgents returns list of all supported CLI agents
func (ug *UnifiedGenerator) ListSupportedAgents() []AgentType {
	var agents []AgentType
	for agentType := range ug.generators {
		agents = append(agents, agentType)
	}
	return agents
}
