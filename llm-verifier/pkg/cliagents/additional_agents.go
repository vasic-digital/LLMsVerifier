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
	Version    string                            `json:"version,omitempty"`
	Provider   GenericProviderConfig             `json:"provider"`
	Providers  map[string]GenericProviderConfig  `json:"providers,omitempty"`
	Models     []GenericModelDef                 `json:"models,omitempty"`
	MCP        map[string]GenericMCP             `json:"mcp,omitempty"`
	Plugins    []string                          `json:"plugins,omitempty"`
	Extensions *HelixAgentExtensions             `json:"extensions,omitempty"`
	Settings   map[string]interface{}            `json:"settings,omitempty"`
	Formatters FormattersConfig                  `json:"formatters,omitempty"`
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

// ============================================================================
// Remaining Original 18 Agents
// ============================================================================

// NewKiroGenerator creates a generator for Kiro AI assistant
func NewKiroGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentKiro,
		schema: &AgentSchema{
			AgentType:        AgentKiro,
			ConfigFileName:   "kiro.json",
			ConfigDirEnvVar:  "KIRO_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "kiro"),
			SupportedFields:  []string{"provider", "model", "apiKey", "baseUrl", "temperature"},
			RequiredFields:   []string{"provider"},
			Description:      "Kiro - AI coding assistant",
		},
	}
}

// NewCodenameGooseGenerator creates a generator for Codename Goose
func NewCodenameGooseGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentCodenameGoose,
		schema: &AgentSchema{
			AgentType:        AgentCodenameGoose,
			ConfigFileName:   "goose.yaml",
			ConfigDirEnvVar:  "GOOSE_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "goose"),
			SupportedFields:  []string{"provider", "model", "mcp_servers", "tools", "extensions"},
			RequiredFields:   []string{"provider"},
			Description:      "Codename Goose - Block's AI coding agent",
		},
	}
}

// NewDeepSeekCLIGenerator creates a generator for DeepSeek CLI
func NewDeepSeekCLIGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentDeepSeekCLI,
		schema: &AgentSchema{
			AgentType:        AgentDeepSeekCLI,
			ConfigFileName:   "deepseek.json",
			ConfigDirEnvVar:  "DEEPSEEK_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "deepseek"),
			SupportedFields:  []string{"apiKey", "model", "temperature", "maxTokens"},
			RequiredFields:   []string{"apiKey"},
			Description:      "DeepSeek CLI - DeepSeek AI coding assistant",
		},
	}
}

// NewForgeGenerator creates a generator for Forge CLI
func NewForgeGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentForge,
		schema: &AgentSchema{
			AgentType:        AgentForge,
			ConfigFileName:   "forge.yaml",
			ConfigDirEnvVar:  "FORGE_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "forge"),
			SupportedFields:  []string{"provider", "model", "templates", "plugins"},
			RequiredFields:   []string{"provider"},
			Description:      "Forge - AI-powered project scaffolding",
		},
	}
}

// NewGeminiCLIGenerator creates a generator for Gemini CLI
func NewGeminiCLIGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentGeminiCLI,
		schema: &AgentSchema{
			AgentType:        AgentGeminiCLI,
			ConfigFileName:   "gemini.json",
			ConfigDirEnvVar:  "GEMINI_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "gemini"),
			SupportedFields:  []string{"apiKey", "model", "temperature", "maxOutputTokens", "safety"},
			RequiredFields:   []string{"apiKey"},
			Description:      "Gemini CLI - Google's AI coding assistant",
		},
	}
}

// NewGPTEngineerGenerator creates a generator for GPT Engineer
func NewGPTEngineerGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentGPTEngineer,
		schema: &AgentSchema{
			AgentType:        AgentGPTEngineer,
			ConfigFileName:   "gpt-engineer.toml",
			ConfigDirEnvVar:  "GPT_ENGINEER_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "gpt-engineer"),
			SupportedFields:  []string{"model", "api_key", "project_path", "steps"},
			RequiredFields:   []string{"model"},
			Description:      "GPT Engineer - AI code generation from prompts",
		},
	}
}

// NewMistralCodeGenerator creates a generator for Mistral Code
func NewMistralCodeGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentMistralCode,
		schema: &AgentSchema{
			AgentType:        AgentMistralCode,
			ConfigFileName:   "mistral.json",
			ConfigDirEnvVar:  "MISTRAL_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "mistral"),
			SupportedFields:  []string{"apiKey", "model", "temperature", "maxTokens", "safePrompt"},
			RequiredFields:   []string{"apiKey"},
			Description:      "Mistral Code - Mistral AI coding assistant",
		},
	}
}

// NewOllamaCodeGenerator creates a generator for Ollama Code
func NewOllamaCodeGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentOllamaCode,
		schema: &AgentSchema{
			AgentType:        AgentOllamaCode,
			ConfigFileName:   "ollama.json",
			ConfigDirEnvVar:  "OLLAMA_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "ollama"),
			SupportedFields:  []string{"host", "model", "keepAlive", "numCtx", "numGpu"},
			RequiredFields:   []string{"model"},
			Description:      "Ollama Code - Local LLM coding assistant (DEPRECATED)",
		},
	}
}

// NewPlandexGenerator creates a generator for Plandex CLI
func NewPlandexGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentPlandex,
		schema: &AgentSchema{
			AgentType:        AgentPlandex,
			ConfigFileName:   "plandex.json",
			ConfigDirEnvVar:  "PLANDEX_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".plandex"),
			SupportedFields:  []string{"apiKey", "model", "planStorage", "autoCommit"},
			RequiredFields:   []string{"apiKey"},
			Description:      "Plandex - AI-powered development planning",
		},
	}
}

// NewAmazonQGenerator creates a generator for Amazon Q
func NewAmazonQGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentAmazonQ,
		schema: &AgentSchema{
			AgentType:        AgentAmazonQ,
			ConfigFileName:   "amazon-q.json",
			ConfigDirEnvVar:  "AMAZON_Q_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".aws", "amazon-q"),
			SupportedFields:  []string{"profile", "region", "features", "telemetry"},
			RequiredFields:   []string{"profile"},
			Description:      "Amazon Q - AWS AI coding assistant",
		},
	}
}

// ============================================================================
// New 30 Agents
// ============================================================================

// NewAgentDeckGenerator creates a generator for Agent-Deck
func NewAgentDeckGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentAgentDeck,
		schema: &AgentSchema{
			AgentType:        AgentAgentDeck,
			ConfigFileName:   "agent-deck.json",
			ConfigDirEnvVar:  "AGENT_DECK_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "agent-deck"),
			SupportedFields:  []string{"providers", "models", "agents", "orchestration", "mcp"},
			RequiredFields:   []string{"providers"},
			Description:      "Agent-Deck - Multi-agent orchestration platform",
		},
	}
}

// NewBridleGenerator creates a generator for Bridle
func NewBridleGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentBridle,
		schema: &AgentSchema{
			AgentType:        AgentBridle,
			ConfigFileName:   "bridle.yaml",
			ConfigDirEnvVar:  "BRIDLE_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "bridle"),
			SupportedFields:  []string{"model", "provider", "tools", "permissions", "sandbox"},
			RequiredFields:   []string{"model"},
			Description:      "Bridle - Constrained AI agent framework",
		},
	}
}

// NewCheshireCatGenerator creates a generator for Cheshire Cat AI
func NewCheshireCatGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentCheshireCat,
		schema: &AgentSchema{
			AgentType:        AgentCheshireCat,
			ConfigFileName:   "cheshire-cat.json",
			ConfigDirEnvVar:  "CHESHIRE_CAT_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "cheshire-cat"),
			SupportedFields:  []string{"llm", "embedder", "memory", "plugins", "hooks"},
			RequiredFields:   []string{"llm"},
			Description:      "Cheshire Cat AI - Customizable AI assistant framework",
		},
	}
}

// NewClaudePluginsGenerator creates a generator for Claude Plugins
func NewClaudePluginsGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentClaudePlugins,
		schema: &AgentSchema{
			AgentType:        AgentClaudePlugins,
			ConfigFileName:   "plugins.json",
			ConfigDirEnvVar:  "CLAUDE_PLUGINS_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".claude", "plugins"),
			SupportedFields:  []string{"hooks", "commands", "permissions", "skills"},
			RequiredFields:   []string{},
			Description:      "Claude Code Plugins - Extensions for Claude Code",
		},
	}
}

// NewClaudeSquadGenerator creates a generator for Claude Squad
func NewClaudeSquadGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentClaudeSquad,
		schema: &AgentSchema{
			AgentType:        AgentClaudeSquad,
			ConfigFileName:   "claude-squad.yaml",
			ConfigDirEnvVar:  "CLAUDE_SQUAD_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "claude-squad"),
			SupportedFields:  []string{"agents", "coordinator", "tasks", "workflows", "mcp"},
			RequiredFields:   []string{"agents"},
			Description:      "Claude Squad - Multi-agent Claude orchestration",
		},
	}
}

// NewCodaiGenerator creates a generator for Codai
func NewCodaiGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentCodai,
		schema: &AgentSchema{
			AgentType:        AgentCodai,
			ConfigFileName:   "codai.json",
			ConfigDirEnvVar:  "CODAI_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "codai"),
			SupportedFields:  []string{"provider", "model", "context", "codeActions"},
			RequiredFields:   []string{"provider"},
			Description:      "Codai - AI code assistant CLI",
		},
	}
}

// NewCodexGenerator creates a generator for Codex CLI
func NewCodexGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentCodex,
		schema: &AgentSchema{
			AgentType:        AgentCodex,
			ConfigFileName:   "codex.json",
			ConfigDirEnvVar:  "CODEX_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "codex"),
			SupportedFields:  []string{"model", "tools", "workspace", "permissions", "mcp"},
			RequiredFields:   []string{"model"},
			Description:      "Codex - OpenAI Codex-powered CLI",
		},
	}
}

// NewCodexSkillsGenerator creates a generator for Codex Skills
func NewCodexSkillsGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentCodexSkills,
		schema: &AgentSchema{
			AgentType:        AgentCodexSkills,
			ConfigFileName:   "codex-skills.json",
			ConfigDirEnvVar:  "CODEX_SKILLS_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "codex-skills"),
			SupportedFields:  []string{"skills", "triggers", "templates", "actions"},
			RequiredFields:   []string{"skills"},
			Description:      "Codex Skills - Custom skill definitions for Codex",
		},
	}
}

// NewConduitGenerator creates a generator for Conduit
func NewConduitGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentConduit,
		schema: &AgentSchema{
			AgentType:        AgentConduit,
			ConfigFileName:   "conduit.json",
			ConfigDirEnvVar:  "CONDUIT_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "conduit"),
			SupportedFields:  []string{"provider", "model", "pipelines", "connectors"},
			RequiredFields:   []string{"provider"},
			Description:      "Conduit - AI data pipeline assistant",
		},
	}
}

// NewEmdashGenerator creates a generator for Emdash
func NewEmdashGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentEmdash,
		schema: &AgentSchema{
			AgentType:        AgentEmdash,
			ConfigFileName:   "emdash.json",
			ConfigDirEnvVar:  "EMDASH_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "emdash"),
			SupportedFields:  []string{"provider", "model", "shortcuts", "templates"},
			RequiredFields:   []string{"provider"},
			Description:      "Emdash - AI-powered text editing CLI",
		},
	}
}

// NewFauxPilotGenerator creates a generator for FauxPilot
func NewFauxPilotGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentFauxPilot,
		schema: &AgentSchema{
			AgentType:        AgentFauxPilot,
			ConfigFileName:   "fauxpilot.yaml",
			ConfigDirEnvVar:  "FAUXPILOT_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "fauxpilot"),
			SupportedFields:  []string{"server", "model", "backend", "triton"},
			RequiredFields:   []string{"server"},
			Description:      "FauxPilot - Self-hosted Copilot alternative",
		},
	}
}

// NewGetShitDoneGenerator creates a generator for Get Shit Done
func NewGetShitDoneGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentGetShitDone,
		schema: &AgentSchema{
			AgentType:        AgentGetShitDone,
			ConfigFileName:   "gsd.json",
			ConfigDirEnvVar:  "GSD_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "gsd"),
			SupportedFields:  []string{"provider", "model", "tasks", "automation"},
			RequiredFields:   []string{"provider"},
			Description:      "Get Shit Done - Task-focused AI assistant",
		},
	}
}

// NewGitHubCopilotCLIGenerator creates a generator for GitHub Copilot CLI
func NewGitHubCopilotCLIGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentGitHubCopilotCLI,
		schema: &AgentSchema{
			AgentType:        AgentGitHubCopilotCLI,
			ConfigFileName:   "copilot-cli.json",
			ConfigDirEnvVar:  "GITHUB_COPILOT_CLI_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "github-copilot-cli"),
			SupportedFields:  []string{"auth", "aliases", "shell", "telemetry"},
			RequiredFields:   []string{"auth"},
			Description:      "GitHub Copilot CLI - Terminal command suggestions",
		},
	}
}

// NewGitHubSpecKitGenerator creates a generator for GitHub Spec Kit
func NewGitHubSpecKitGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentGitHubSpecKit,
		schema: &AgentSchema{
			AgentType:        AgentGitHubSpecKit,
			ConfigFileName:   "spec-kit.json",
			ConfigDirEnvVar:  "GITHUB_SPEC_KIT_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "github-spec-kit"),
			SupportedFields:  []string{"repository", "specs", "templates", "validation"},
			RequiredFields:   []string{"repository"},
			Description:      "GitHub Spec Kit - AI specification generator",
		},
	}
}

// NewGitMCPGenerator creates a generator for GitMCP
func NewGitMCPGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentGitMCP,
		schema: &AgentSchema{
			AgentType:        AgentGitMCP,
			ConfigFileName:   "gitmcp.json",
			ConfigDirEnvVar:  "GITMCP_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "gitmcp"),
			SupportedFields:  []string{"servers", "transports", "tools", "resources"},
			RequiredFields:   []string{"servers"},
			Description:      "GitMCP - Git-based MCP server management",
		},
	}
}

// NewGPTMEGenerator creates a generator for GPTME
func NewGPTMEGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentGPTME,
		schema: &AgentSchema{
			AgentType:        AgentGPTME,
			ConfigFileName:   "gptme.toml",
			ConfigDirEnvVar:  "GPTME_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "gptme"),
			SupportedFields:  []string{"model", "provider", "tools", "context", "history"},
			RequiredFields:   []string{"model"},
			Description:      "GPTME - Personal AI assistant in terminal",
		},
	}
}

// NewMobileAgentGenerator creates a generator for Mobile Agent
func NewMobileAgentGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentMobileAgent,
		schema: &AgentSchema{
			AgentType:        AgentMobileAgent,
			ConfigFileName:   "mobile-agent.json",
			ConfigDirEnvVar:  "MOBILE_AGENT_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "mobile-agent"),
			SupportedFields:  []string{"provider", "model", "device", "actions", "vision"},
			RequiredFields:   []string{"provider"},
			Description:      "Mobile Agent - AI mobile device automation",
		},
	}
}

// NewMultiagentCodingGenerator creates a generator for Multiagent Coding
func NewMultiagentCodingGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentMultiagentCoding,
		schema: &AgentSchema{
			AgentType:        AgentMultiagentCoding,
			ConfigFileName:   "multiagent.yaml",
			ConfigDirEnvVar:  "MULTIAGENT_CODING_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "multiagent-coding"),
			SupportedFields:  []string{"agents", "roles", "coordination", "workspace"},
			RequiredFields:   []string{"agents"},
			Description:      "Multiagent Coding - Coordinated multi-agent development",
		},
	}
}

// NewNanocoderGenerator creates a generator for Nanocoder
func NewNanocoderGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentNanocoder,
		schema: &AgentSchema{
			AgentType:        AgentNanocoder,
			ConfigFileName:   "nanocoder.json",
			ConfigDirEnvVar:  "NANOCODER_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "nanocoder"),
			SupportedFields:  []string{"provider", "model", "snippets", "templates"},
			RequiredFields:   []string{"provider"},
			Description:      "Nanocoder - Lightweight AI code generator",
		},
	}
}

// NewNoiGenerator creates a generator for Noi
func NewNoiGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentNoi,
		schema: &AgentSchema{
			AgentType:        AgentNoi,
			ConfigFileName:   "noi.json",
			ConfigDirEnvVar:  "NOI_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "noi"),
			SupportedFields:  []string{"providers", "models", "prompts", "themes"},
			RequiredFields:   []string{"providers"},
			Description:      "Noi - Cross-platform AI chat interface",
		},
	}
}

// NewOctogenGenerator creates a generator for Octogen
func NewOctogenGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentOctogen,
		schema: &AgentSchema{
			AgentType:        AgentOctogen,
			ConfigFileName:   "octogen.yaml",
			ConfigDirEnvVar:  "OCTOGEN_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "octogen"),
			SupportedFields:  []string{"provider", "model", "kernel", "workspace"},
			RequiredFields:   []string{"provider"},
			Description:      "Octogen - AI code interpreter and executor",
		},
	}
}

// NewOpenHandsGenerator creates a generator for OpenHands
func NewOpenHandsGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentOpenHands,
		schema: &AgentSchema{
			AgentType:        AgentOpenHands,
			ConfigFileName:   "openhands.toml",
			ConfigDirEnvVar:  "OPENHANDS_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "openhands"),
			SupportedFields:  []string{"runtime", "model", "tools", "workspace", "mcp"},
			RequiredFields:   []string{"model"},
			Description:      "OpenHands - Open-source AI software engineer",
		},
	}
}

// NewPostgresMCPGenerator creates a generator for PostgresMCP
func NewPostgresMCPGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentPostgresMCP,
		schema: &AgentSchema{
			AgentType:        AgentPostgresMCP,
			ConfigFileName:   "postgres-mcp.json",
			ConfigDirEnvVar:  "POSTGRES_MCP_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "postgres-mcp"),
			SupportedFields:  []string{"connection", "queries", "tools", "schema"},
			RequiredFields:   []string{"connection"},
			Description:      "PostgresMCP - MCP server for PostgreSQL",
		},
	}
}

// NewShaiGenerator creates a generator for Shai
func NewShaiGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentShai,
		schema: &AgentSchema{
			AgentType:        AgentShai,
			ConfigFileName:   "shai.json",
			ConfigDirEnvVar:  "SHAI_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "shai"),
			SupportedFields:  []string{"provider", "model", "shell", "aliases"},
			RequiredFields:   []string{"provider"},
			Description:      "Shai - Shell AI assistant",
		},
	}
}

// NewSnowCLIGenerator creates a generator for SnowCLI
func NewSnowCLIGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentSnowCLI,
		schema: &AgentSchema{
			AgentType:        AgentSnowCLI,
			ConfigFileName:   "snowcli.yaml",
			ConfigDirEnvVar:  "SNOWCLI_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "snowcli"),
			SupportedFields:  []string{"connection", "warehouse", "database", "schema"},
			RequiredFields:   []string{"connection"},
			Description:      "SnowCLI - Snowflake AI-assisted CLI",
		},
	}
}

// NewTaskWeaverGenerator creates a generator for TaskWeaver
func NewTaskWeaverGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentTaskWeaver,
		schema: &AgentSchema{
			AgentType:        AgentTaskWeaver,
			ConfigFileName:   "taskweaver.yaml",
			ConfigDirEnvVar:  "TASKWEAVER_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "taskweaver"),
			SupportedFields:  []string{"model", "plugins", "workspace", "code_interpreter"},
			RequiredFields:   []string{"model"},
			Description:      "TaskWeaver - Microsoft's code-first AI agent",
		},
	}
}

// NewUIUXProMaxGenerator creates a generator for UI/UX Pro Max
func NewUIUXProMaxGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentUIUXProMax,
		schema: &AgentSchema{
			AgentType:        AgentUIUXProMax,
			ConfigFileName:   "uiux-pro-max.json",
			ConfigDirEnvVar:  "UIUX_PRO_MAX_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "uiux-pro-max"),
			SupportedFields:  []string{"provider", "model", "frameworks", "components"},
			RequiredFields:   []string{"provider"},
			Description:      "UI/UX Pro Max - AI UI/UX design assistant",
		},
	}
}

// NewVTCodeGenerator creates a generator for VTCode
func NewVTCodeGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentVTCode,
		schema: &AgentSchema{
			AgentType:        AgentVTCode,
			ConfigFileName:   "vtcode.json",
			ConfigDirEnvVar:  "VTCODE_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "vtcode"),
			SupportedFields:  []string{"provider", "model", "editor", "keybindings"},
			RequiredFields:   []string{"provider"},
			Description:      "VTCode - Visual Terminal Code AI assistant",
		},
	}
}

// NewWarpGenerator creates a generator for Warp terminal
func NewWarpGenerator() *GenericAgentGenerator {
	homeDir, _ := os.UserHomeDir()
	return &GenericAgentGenerator{
		agentType: AgentWarp,
		schema: &AgentSchema{
			AgentType:        AgentWarp,
			ConfigFileName:   "warp.yaml",
			ConfigDirEnvVar:  "WARP_CONFIG_DIR",
			DefaultConfigDir: filepath.Join(homeDir, ".config", "warp"),
			SupportedFields:  []string{"ai", "theme", "keybindings", "workflows"},
			RequiredFields:   []string{},
			Description:      "Warp - AI-powered terminal",
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
	// Use real API key from environment for installed configs;
	// env var references are NOT supported by most CLI agents
	apiKey := os.Getenv("HELIXAGENT_API_KEY")
	if apiKey == "" {
		apiKey = "<YOUR_HELIXAGENT_API_KEY>"
	}
	agentConfig.Provider = GenericProviderConfig{
		Type:    "openai-compatible",
		Name:    "helixagent",
		BaseURL: baseURL,
		APIKey:  apiKey,
	}

	// Add HelixLLM as additional provider (direct LLM access with RAG/agents)
	helixLLMHost := config.HelixLLMHost
	if helixLLMHost == "" {
		helixLLMHost = "localhost"
	}
	helixLLMPort := config.HelixLLMPort
	if helixLLMPort == 0 {
		helixLLMPort = 8443
	}
	helixLLMBaseURL := fmt.Sprintf("https://%s:%d/v1", helixLLMHost, helixLLMPort)
	agentConfig.Providers = map[string]GenericProviderConfig{
		"helixagent": {
			Type:    "openai-compatible",
			Name:    "helixagent",
			BaseURL: baseURL,
			APIKey:  apiKey,
		},
		"helixllm": {
			Type:    "openai-compatible",
			Name:    "helixllm",
			BaseURL: helixLLMBaseURL,
			APIKey:  config.HelixLLMAPIKey,
		},
	}

	// Configure models
	agentConfig.Models = []GenericModelDef{
		{
			ID:        "helixagent-debate",
			Name:      "HelixAgent AI Debate Ensemble",
			MaxTokens: 128000,
			Capabilities: []string{
				"vision", "streaming", "function_calls", "embeddings",
				"mcp", "acp", "lsp", "code_execution", "rag",
			},
		},
		{
			ID:        "helixllm-default",
			Name:      "HelixLLM Default",
			MaxTokens: 128000,
			Capabilities: []string{
				"vision", "streaming", "function_calls", "embeddings",
				"rag", "agents", "knowledge",
			},
		},
	}

	// Configure MCP servers (15+ out of the box)
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

	// Configure plugins
	agentConfig.Plugins = DefaultPlugins()

	// Configure extensions (LSP, ACP, Embeddings, RAG, Skills)
	agentConfig.Extensions = DefaultHelixAgentExtensions(config.HelixAgentHost, config.HelixAgentPort)

	// Agent-specific settings
	agentConfig.Settings = g.getAgentSpecificSettings()

	// Configure formatters
	agentConfig.Formatters = DefaultFormattersConfig(config.HelixAgentHost, config.HelixAgentPort)

	result.Config = agentConfig
	result.Success = true

	return result, nil
}

// getAgentSpecificSettings returns agent-specific default settings
func (g *GenericAgentGenerator) getAgentSpecificSettings() map[string]interface{} {
	switch g.agentType {
	// Original 18 agents
	case AgentAider:
		return map[string]interface{}{
			"auto_commits": true,
			"edit_format":  "diff",
			"stream":       true,
			"show_diffs":   true,
		}
	case AgentClaudeCode:
		return map[string]interface{}{
			"autoApprove": false,
			"theme":       "auto",
		}
	case AgentCline:
		return map[string]interface{}{
			"autoApprove":    false,
			"streamResponse": true,
			"showDiff":       true,
		}
	case AgentQwenCode:
		return map[string]interface{}{
			"language": "en",
			"stream":   true,
		}
	case AgentKiro:
		return map[string]interface{}{
			"autoComplete": true,
			"streaming":    true,
		}
	case AgentCodenameGoose:
		return map[string]interface{}{
			"streaming":  true,
			"extensions": []string{},
		}
	case AgentDeepSeekCLI:
		return map[string]interface{}{
			"stream":      true,
			"temperature": 0.7,
		}
	case AgentForge:
		return map[string]interface{}{
			"interactive": true,
			"templates":   []string{},
		}
	case AgentGeminiCLI:
		return map[string]interface{}{
			"stream": true,
			"safety": "balanced",
		}
	case AgentGPTEngineer:
		return map[string]interface{}{
			"steps":     []string{"clarify", "generate", "run"},
			"streaming": true,
		}
	case AgentMistralCode:
		return map[string]interface{}{
			"stream":     true,
			"safePrompt": true,
		}
	case AgentOllamaCode:
		return map[string]interface{}{
			"keepAlive": "5m",
			"numCtx":    4096,
		}
	case AgentPlandex:
		return map[string]interface{}{
			"autoCommit":  true,
			"interactive": true,
		}
	case AgentAmazonQ:
		return map[string]interface{}{
			"telemetry": false,
			"features":  []string{"code", "chat"},
		}
	// New 30 agents
	case AgentAgentDeck:
		return map[string]interface{}{
			"orchestration": "parallel",
			"maxAgents":     5,
		}
	case AgentBridle:
		return map[string]interface{}{
			"sandbox":     true,
			"permissions": []string{"read", "write"},
		}
	case AgentCheshireCat:
		return map[string]interface{}{
			"memory":  "persistent",
			"plugins": []string{},
		}
	case AgentClaudePlugins:
		return map[string]interface{}{
			"enabled": true,
			"skills":  []string{},
		}
	case AgentClaudeSquad:
		return map[string]interface{}{
			"coordination": "sequential",
			"maxAgents":    3,
		}
	case AgentCodai:
		return map[string]interface{}{
			"contextWindow": 8192,
			"streaming":     true,
		}
	case AgentCodex:
		return map[string]interface{}{
			"permissions": []string{"read", "write", "execute"},
			"streaming":   true,
		}
	case AgentCodexSkills:
		return map[string]interface{}{
			"autoTrigger": true,
			"skills":      []string{},
		}
	case AgentConduit:
		return map[string]interface{}{
			"pipelines":  []string{},
			"autoCommit": false,
		}
	case AgentContinue:
		return map[string]interface{}{
			"allowAnonymousTelemetry": false,
			"tabAutocomplete":         true,
			"systemMessage":           "You are a helpful AI coding assistant powered by HelixAgent.",
		}
	case AgentEmdash:
		return map[string]interface{}{
			"shortcuts": map[string]string{},
			"theme":     "auto",
		}
	case AgentFauxPilot:
		return map[string]interface{}{
			"backend":   "triton",
			"streaming": true,
		}
	case AgentGetShitDone:
		return map[string]interface{}{
			"autoExecute": false,
			"verbose":     true,
		}
	case AgentGitHubCopilotCLI:
		return map[string]interface{}{
			"telemetry": false,
			"aliases":   map[string]string{},
		}
	case AgentGitHubSpecKit:
		return map[string]interface{}{
			"validation": true,
			"templates":  []string{},
		}
	case AgentGitMCP:
		return map[string]interface{}{
			"autoSync":   true,
			"transports": []string{"stdio", "sse"},
		}
	case AgentGPTME:
		return map[string]interface{}{
			"history":   true,
			"tools":     []string{"shell", "python", "browser"},
			"streaming": true,
		}
	case AgentMobileAgent:
		return map[string]interface{}{
			"vision":    true,
			"actions":   []string{},
			"recording": false,
		}
	case AgentMultiagentCoding:
		return map[string]interface{}{
			"coordination": "hierarchical",
			"roles":        []string{"architect", "developer", "reviewer"},
		}
	case AgentNanocoder:
		return map[string]interface{}{
			"snippets":  []string{},
			"streaming": true,
		}
	case AgentNoi:
		return map[string]interface{}{
			"theme":  "system",
			"prompts": map[string]string{},
		}
	case AgentOctogen:
		return map[string]interface{}{
			"kernel":    "python",
			"workspace": "~/octogen-workspace",
		}
	case AgentOpenHands:
		return map[string]interface{}{
			"runtime":   "docker",
			"workspace": "~/openhands-workspace",
			"streaming": true,
		}
	case AgentPostgresMCP:
		return map[string]interface{}{
			"readOnly": true,
			"timeout":  30,
		}
	case AgentShai:
		return map[string]interface{}{
			"shell":   "bash",
			"aliases": map[string]string{},
		}
	case AgentSnowCLI:
		return map[string]interface{}{
			"warehouse": "COMPUTE_WH",
			"timeout":   60,
		}
	case AgentTaskWeaver:
		return map[string]interface{}{
			"codeInterpreter": true,
			"plugins":         []string{},
		}
	case AgentUIUXProMax:
		return map[string]interface{}{
			"frameworks": []string{"react", "vue", "svelte"},
			"components": []string{},
		}
	case AgentVTCode:
		return map[string]interface{}{
			"theme":       "dark",
			"keybindings": "vim",
		}
	case AgentWarp:
		return map[string]interface{}{
			"ai":        true,
			"workflows": []string{},
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
