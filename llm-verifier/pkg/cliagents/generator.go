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

// SupportedAgents lists all 48 CLI agents supported by LLMsVerifier
var SupportedAgents = []string{
	// Original 18 agents
	"opencode",       // OpenCode.ai CLI
	"crush",          // Charm Land Crush CLI
	"helixcode",      // HelixCode CLI
	"kiro",           // Kiro AI assistant
	"aider",          // Aider CLI
	"claude-code",    // Claude Code CLI
	"cline",          // Cline CLI
	"codename-goose", // Codename Goose
	"deepseek-cli",   // DeepSeek CLI
	"forge",          // Forge CLI
	"gemini-cli",     // Gemini CLI
	"gpt-engineer",   // GPT Engineer
	"kilocode",       // KiloCode VS Code extension
	"mistral-code",   // Mistral Code
	"ollama-code",    // Ollama Code
	"plandex",        // Plandex CLI
	"qwen-code",      // Qwen Code CLI
	"amazon-q",       // Amazon Q
	// New 30 agents
	"agent-deck",         // Agent-Deck multi-agent orchestration
	"bridle",             // Bridle CLI
	"cheshire-cat",       // Cheshire Cat AI
	"claude-plugins",     // Claude Code Plugins
	"claude-squad",       // Claude Squad
	"codai",              // Codai CLI
	"codex",              // Codex CLI
	"codex-skills",       // Codex Skills
	"conduit",            // Conduit CLI
	"continue",           // Continue.dev extension
	"emdash",             // Emdash CLI
	"fauxpilot",          // FauxPilot
	"get-shit-done",      // Get Shit Done CLI
	"github-copilot-cli", // GitHub Copilot CLI
	"github-spec-kit",    // GitHub Spec Kit
	"git-mcp",            // GitMCP
	"gptme",              // GPTME CLI
	"mobile-agent",       // Mobile Agent
	"multiagent-coding",  // Multiagent Coding System
	"nanocoder",          // Nanocoder
	"noi",                // Noi CLI
	"octogen",            // Octogen
	"openhands",          // OpenHands
	"postgres-mcp",       // PostgresMCP
	"shai",               // Shai CLI
	"snow-cli",           // SnowCLI
	"task-weaver",        // TaskWeaver
	"ui-ux-pro-max",      // UI/UX Pro Max
	"vtcode",             // VTCode
	"warp",               // Warp terminal
}

// AgentType represents the type of CLI agent
type AgentType string

const (
	// Original 18 agents
	AgentOpenCode      AgentType = "opencode"
	AgentCrush         AgentType = "crush"
	AgentHelixCode     AgentType = "helixcode"
	AgentKiro          AgentType = "kiro"
	AgentAider         AgentType = "aider"
	AgentClaudeCode    AgentType = "claude-code"
	AgentCline         AgentType = "cline"
	AgentCodenameGoose AgentType = "codename-goose"
	AgentDeepSeekCLI   AgentType = "deepseek-cli"
	AgentForge         AgentType = "forge"
	AgentGeminiCLI     AgentType = "gemini-cli"
	AgentGPTEngineer   AgentType = "gpt-engineer"
	AgentKiloCode      AgentType = "kilocode"
	AgentMistralCode   AgentType = "mistral-code"
	AgentOllamaCode    AgentType = "ollama-code"
	AgentPlandex       AgentType = "plandex"
	AgentQwenCode      AgentType = "qwen-code"
	AgentAmazonQ       AgentType = "amazon-q"
	// New 30 agents
	AgentAgentDeck        AgentType = "agent-deck"
	AgentBridle           AgentType = "bridle"
	AgentCheshireCat      AgentType = "cheshire-cat"
	AgentClaudePlugins    AgentType = "claude-plugins"
	AgentClaudeSquad      AgentType = "claude-squad"
	AgentCodai            AgentType = "codai"
	AgentCodex            AgentType = "codex"
	AgentCodexSkills      AgentType = "codex-skills"
	AgentConduit          AgentType = "conduit"
	AgentContinue         AgentType = "continue"
	AgentEmdash           AgentType = "emdash"
	AgentFauxPilot        AgentType = "fauxpilot"
	AgentGetShitDone      AgentType = "get-shit-done"
	AgentGitHubCopilotCLI AgentType = "github-copilot-cli"
	AgentGitHubSpecKit    AgentType = "github-spec-kit"
	AgentGitMCP           AgentType = "git-mcp"
	AgentGPTME            AgentType = "gptme"
	AgentMobileAgent      AgentType = "mobile-agent"
	AgentMultiagentCoding AgentType = "multiagent-coding"
	AgentNanocoder        AgentType = "nanocoder"
	AgentNoi              AgentType = "noi"
	AgentOctogen          AgentType = "octogen"
	AgentOpenHands        AgentType = "openhands"
	AgentPostgresMCP      AgentType = "postgres-mcp"
	AgentShai             AgentType = "shai"
	AgentSnowCLI          AgentType = "snow-cli"
	AgentTaskWeaver       AgentType = "task-weaver"
	AgentUIUXProMax       AgentType = "ui-ux-pro-max"
	AgentVTCode           AgentType = "vtcode"
	AgentWarp             AgentType = "warp"
)

// GeneratorConfig contains configuration for CLI agent config generation
type GeneratorConfig struct {
	// HelixAgent endpoint configuration
	HelixAgentHost string
	HelixAgentPort int

	// HelixLLM endpoint configuration (direct LLM access with RAG/agents)
	HelixLLMHost   string
	HelixLLMPort   int
	HelixLLMAPIKey string

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
	Name         string         `json:"name"`
	Type         string         `json:"type"`
	BaseURL      string         `json:"base_url,omitempty"`
	APIKey       string         `json:"api_key,omitempty"`
	APIKeyEnvVar string         `json:"api_key_env_var,omitempty"`
	Model        string         `json:"model,omitempty"`
	Models       []ModelConfig  `json:"models,omitempty"`
	Options      map[string]any `json:"options,omitempty"`
	Score        float64        `json:"score,omitempty"`
	Verified     bool           `json:"verified,omitempty"`
}

// ModelConfig represents a model configuration
type ModelConfig struct {
	ID           string   `json:"id"`
	Name         string   `json:"name,omitempty"`
	MaxTokens    int      `json:"max_tokens,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Score        float64  `json:"score,omitempty"`
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
	Name         string `json:"name"`
	Model        string `json:"model,omitempty"`
	Prompt       string `json:"prompt,omitempty"`
	SystemPrompt string `json:"system_prompt,omitempty"`
}

// GenerationResult contains the result of configuration generation
type GenerationResult struct {
	AgentType        AgentType         `json:"agent_type"`
	Success          bool              `json:"success"`
	ConfigPath       string            `json:"config_path,omitempty"`
	Config           any               `json:"config,omitempty"`
	Errors           []string          `json:"errors,omitempty"`
	Warnings         []string          `json:"warnings,omitempty"`
	GeneratedAt      time.Time         `json:"generated_at"`
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
	AgentType        AgentType `json:"agent_type"`
	SchemaURL        string    `json:"schema_url,omitempty"`
	ConfigFileName   string    `json:"config_file_name"`
	ConfigDirEnvVar  string    `json:"config_dir_env_var,omitempty"`
	DefaultConfigDir string    `json:"default_config_dir"`
	SupportedFields  []string  `json:"supported_fields"`
	RequiredFields   []string  `json:"required_fields"`
	Description      string    `json:"description"`
}

// HelixAgentExtensions contains all HelixAgent-provided extensions and capabilities
// that every CLI agent config should include
type HelixAgentExtensions struct {
	// Plugins available for the CLI agent
	Plugins []string `json:"plugins,omitempty"`

	// LSP endpoints
	LSP *LSPConfig `json:"lsp,omitempty"`

	// ACP (Agent Communication Protocol) endpoints
	ACP *ACPConfig `json:"acp,omitempty"`

	// Embeddings configuration
	Embeddings *EmbeddingsConfig `json:"embeddings,omitempty"`

	// RAG (Retrieval-Augmented Generation) configuration
	RAG *RAGConfig `json:"rag,omitempty"`

	// Skills available for the CLI agent
	Skills []SkillConfig `json:"skills,omitempty"`
}

// LSPConfig represents Language Server Protocol configuration
type LSPConfig struct {
	Enabled  bool   `json:"enabled"`
	Endpoint string `json:"endpoint"`
}

// ACPConfig represents Agent Communication Protocol configuration
type ACPConfig struct {
	Enabled  bool   `json:"enabled"`
	Endpoint string `json:"endpoint"`
}

// EmbeddingsConfig represents embeddings service configuration
type EmbeddingsConfig struct {
	Enabled  bool   `json:"enabled"`
	Endpoint string `json:"endpoint"`
	Model    string `json:"model,omitempty"`
}

// RAGConfig represents Retrieval-Augmented Generation configuration
type RAGConfig struct {
	Enabled     bool   `json:"enabled"`
	Endpoint    string `json:"endpoint"`
	VectorStore string `json:"vector_store,omitempty"`
}

// SkillConfig represents a skill available to the CLI agent
type SkillConfig struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Endpoint    string `json:"endpoint,omitempty"`
	Enabled     bool   `json:"enabled"`
}

// DefaultHelixAgentExtensions returns the default set of extensions for a given host/port
func DefaultHelixAgentExtensions(host string, port int) *HelixAgentExtensions {
	baseURL := fmt.Sprintf("http://%s:%d", host, port)
	return &HelixAgentExtensions{
		Plugins: DefaultPlugins(),
		LSP: &LSPConfig{
			Enabled:  true,
			Endpoint: baseURL + "/v1/lsp",
		},
		ACP: &ACPConfig{
			Enabled:  true,
			Endpoint: baseURL + "/v1/acp",
		},
		Embeddings: &EmbeddingsConfig{
			Enabled:  true,
			Endpoint: baseURL + "/v1/embeddings",
			Model:    "helix-llm",
		},
		RAG: &RAGConfig{
			Enabled:     true,
			Endpoint:    baseURL + "/v1/rag",
			VectorStore: "chromadb",
		},
		Skills: DefaultSkills(host, port),
	}
}

// DefaultPlugins returns the default set of plugins for HelixAgent-powered CLI agents
func DefaultPlugins() []string {
	return []string{
		"helixagent-mcp",
		"helixagent-lsp",
		"helixagent-acp",
		"helixagent-embeddings",
		"helixagent-vision",
		"helixagent-rag",
		"helixagent-formatters",
		"helixagent-debate",
		"helixagent-memory",
		"helixagent-monitoring",
	}
}

// DefaultSkills returns the default set of skills for HelixAgent-powered CLI agents
func DefaultSkills(host string, port int) []SkillConfig {
	baseURL := fmt.Sprintf("http://%s:%d", host, port)
	return []SkillConfig{
		{Name: "code-review", Description: "AI-powered code review with multi-LLM debate", Endpoint: baseURL + "/v1/debate", Enabled: true},
		{Name: "code-format", Description: "Format code using 32+ formatters", Endpoint: baseURL + "/v1/format", Enabled: true},
		{Name: "semantic-search", Description: "Semantic code search using embeddings", Endpoint: baseURL + "/v1/embeddings", Enabled: true},
		{Name: "vision-analysis", Description: "Analyze images and screenshots", Endpoint: baseURL + "/v1/vision", Enabled: true},
		{Name: "memory-recall", Description: "Persistent memory across sessions", Endpoint: baseURL + "/v1/cognee", Enabled: true},
		{Name: "rag-retrieval", Description: "Retrieve context from knowledge base", Endpoint: baseURL + "/v1/rag", Enabled: true},
		{Name: "lsp-diagnostics", Description: "Language server diagnostics and completions", Endpoint: baseURL + "/v1/lsp", Enabled: true},
		{Name: "agent-communication", Description: "Inter-agent communication via ACP", Endpoint: baseURL + "/v1/acp", Enabled: true},
	}
}

// DefaultPluginsCount returns the number of default plugins
func DefaultPluginsCount() int {
	return len(DefaultPlugins())
}

// DefaultSkillsCount returns the number of default skills
func DefaultSkillsCount() int {
	return 8
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

// DefaultMCPServers returns default MCP server configurations (15+ servers)
// Includes: HelixAgent remote endpoints + npx-based local servers + free remote MCPs
// All agents MUST ship with 15+ MCP servers out of the box
func DefaultMCPServers() []MCPServerConfig {
	return DefaultMCPServersForHost("localhost", 7061)
}

// DefaultMCPServersForHost returns default MCP servers for a given host and port
func DefaultMCPServersForHost(host string, port int) []MCPServerConfig {
	baseURL := fmt.Sprintf("http://%s:%d", host, port)
	return []MCPServerConfig{
		// ============================================
		// HelixAgent Remote Endpoints (6)
		// Connect to running HelixAgent server
		// NO npm/npx dependencies - pure HTTP connections
		// ============================================
		{Name: "helixagent-mcp", Type: "remote", URL: baseURL + "/v1/mcp"},
		{Name: "helixagent-acp", Type: "remote", URL: baseURL + "/v1/acp"},
		{Name: "helixagent-lsp", Type: "remote", URL: baseURL + "/v1/lsp"},
		{Name: "helixagent-embeddings", Type: "remote", URL: baseURL + "/v1/embeddings"},
		{Name: "helixagent-vision", Type: "remote", URL: baseURL + "/v1/vision"},
		{Name: "helixagent-cognee", Type: "remote", URL: baseURL + "/v1/cognee"},

		// ============================================
		// HelixAgent Extended Services (3)
		// RAG, Formatters, and Monitoring endpoints
		// ============================================
		{Name: "helixagent-rag", Type: "remote", URL: baseURL + "/v1/rag"},
		{Name: "helixagent-formatters", Type: "remote", URL: baseURL + "/v1/formatters"},
		{Name: "helixagent-monitoring", Type: "remote", URL: baseURL + "/v1/monitoring"},

		// ============================================
		// Local npx-based MCP Servers (6)
		// Started on-demand by the CLI agent
		// Require Node.js/npm installed
		// ============================================
		{Name: "filesystem", Type: "local", Command: []string{"npx", "-y", "@modelcontextprotocol/server-filesystem", "."}},
		{Name: "memory", Type: "local", Command: []string{"npx", "-y", "@modelcontextprotocol/server-memory"}},
		{Name: "sequential-thinking", Type: "local", Command: []string{"npx", "-y", "@modelcontextprotocol/server-sequential-thinking"}},
		{Name: "everything", Type: "local", Command: []string{"npx", "-y", "@modelcontextprotocol/server-everything"}},
		{Name: "puppeteer", Type: "local", Command: []string{"npx", "-y", "@modelcontextprotocol/server-puppeteer"}},
		{Name: "sqlite", Type: "local", Command: []string{"npx", "-y", "mcp-server-sqlite-npx"}},

		// ============================================
		// Free Remote MCP Servers (3)
		// No authentication required
		// ============================================
		{Name: "context7", Type: "remote", URL: "https://mcp.context7.com/mcp"},
		{Name: "deepwiki", Type: "remote", URL: "https://mcp.deepwiki.com/mcp"},
		{Name: "cloudflare-docs", Type: "remote", URL: "https://docs.mcp.cloudflare.com/sse"},
	}
}

// DefaultMCPServersCount returns the total number of default MCP servers
func DefaultMCPServersCount() int {
	return 18 // 6 HelixAgent + 3 extended + 6 npx local + 3 free remote
}

// HelixLLMMCPServers returns MCP server entries for HelixLLM endpoints.
// NOTE: HelixLLM exposes REST API endpoints, NOT MCP protocol servers.
// These endpoints do not speak JSON-RPC over SSE, so they MUST NOT be
// registered as remote MCPs in CLI agents like OpenCode — doing so causes
// the agent to hang on startup waiting for an MCP handshake that never comes.
// HelixLLM is accessed via the "helixllm" provider entry instead.
// This function returns an empty slice; kept for backward compatibility.
func HelixLLMMCPServers(host string, port int) []MCPServerConfig {
	return nil
}

// ContainerizedMCPServers returns MCP servers running as Docker containers
// ZERO npm/npx dependencies - all MCPs run as Docker containers
// Use with: docker-compose -f docker/mcp/docker-compose.mcp-full.yml up -d
func ContainerizedMCPServers(host string) []MCPServerConfig {
	if host == "" {
		host = "localhost"
	}

	return []MCPServerConfig{
		// ============================================
		// HelixAgent Remote Endpoints (6)
		// ============================================
		{Name: "helixagent-mcp", Type: "remote", URL: "http://localhost:7061/v1/mcp"},
		{Name: "helixagent-acp", Type: "remote", URL: "http://localhost:7061/v1/acp"},
		{Name: "helixagent-lsp", Type: "remote", URL: "http://localhost:7061/v1/lsp"},
		{Name: "helixagent-embeddings", Type: "remote", URL: "http://localhost:7061/v1/embeddings"},
		{Name: "helixagent-vision", Type: "remote", URL: "http://localhost:7061/v1/vision"},
		{Name: "helixagent-cognee", Type: "remote", URL: "http://localhost:7061/v1/cognee"},

		// ============================================
		// Core MCP Servers (10) - Ports 9101-9110
		// ============================================
		{Name: "fetch", Type: "remote", URL: "http://" + host + ":9101/sse"},
		{Name: "git", Type: "remote", URL: "http://" + host + ":9102/sse"},
		{Name: "time", Type: "remote", URL: "http://" + host + ":9103/sse"},
		{Name: "filesystem", Type: "remote", URL: "http://" + host + ":9104/sse"},
		{Name: "memory", Type: "remote", URL: "http://" + host + ":9105/sse"},
		{Name: "everything", Type: "remote", URL: "http://" + host + ":9106/sse"},
		{Name: "sequential-thinking", Type: "remote", URL: "http://" + host + ":9107/sse"},
		{Name: "sqlite", Type: "remote", URL: "http://" + host + ":9108/sse"},
		{Name: "puppeteer", Type: "remote", URL: "http://" + host + ":9109/sse"},
		{Name: "postgres", Type: "remote", URL: "http://" + host + ":9110/sse"},

		// ============================================
		// Database MCP Servers (5) - Ports 9201-9205
		// ============================================
		{Name: "mongodb", Type: "remote", URL: "http://" + host + ":9201/sse"},
		{Name: "redis", Type: "remote", URL: "http://" + host + ":9202/sse"},
		{Name: "mysql", Type: "remote", URL: "http://" + host + ":9203/sse"},
		{Name: "elasticsearch", Type: "remote", URL: "http://" + host + ":9204/sse"},
		{Name: "supabase", Type: "remote", URL: "http://" + host + ":9205/sse"},

		// ============================================
		// Vector DB MCP Servers (4) - Ports 9301-9304
		// ============================================
		{Name: "qdrant", Type: "remote", URL: "http://" + host + ":9301/sse"},
		{Name: "chroma", Type: "remote", URL: "http://" + host + ":9302/sse"},
		{Name: "pinecone", Type: "remote", URL: "http://" + host + ":9303/sse"},
		{Name: "weaviate", Type: "remote", URL: "http://" + host + ":9304/sse"},

		// ============================================
		// DevOps MCP Servers (14) - Ports 9401-9414
		// ============================================
		{Name: "github", Type: "remote", URL: "http://" + host + ":9401/sse"},
		{Name: "gitlab", Type: "remote", URL: "http://" + host + ":9402/sse"},
		{Name: "sentry", Type: "remote", URL: "http://" + host + ":9403/sse"},
		{Name: "kubernetes", Type: "remote", URL: "http://" + host + ":9404/sse"},
		{Name: "docker", Type: "remote", URL: "http://" + host + ":9405/sse"},
		{Name: "ansible", Type: "remote", URL: "http://" + host + ":9406/sse"},
		{Name: "aws", Type: "remote", URL: "http://" + host + ":9407/sse"},
		{Name: "gcp", Type: "remote", URL: "http://" + host + ":9408/sse"},
		{Name: "heroku", Type: "remote", URL: "http://" + host + ":9409/sse"},
		{Name: "cloudflare", Type: "remote", URL: "http://" + host + ":9410/sse"},
		{Name: "vercel", Type: "remote", URL: "http://" + host + ":9411/sse"},
		{Name: "workers", Type: "remote", URL: "http://" + host + ":9412/sse"},
		{Name: "jetbrains", Type: "remote", URL: "http://" + host + ":9413/sse"},

		// ============================================
		// Browser MCP Servers (4) - Ports 9501-9504
		// ============================================
		{Name: "playwright", Type: "remote", URL: "http://" + host + ":9501/sse"},
		{Name: "browserbase", Type: "remote", URL: "http://" + host + ":9502/sse"},
		{Name: "firecrawl", Type: "remote", URL: "http://" + host + ":9503/sse"},
		{Name: "crawl4ai", Type: "remote", URL: "http://" + host + ":9504/sse"},

		// ============================================
		// Communication MCP Servers (3) - Ports 9601-9603
		// ============================================
		{Name: "slack", Type: "remote", URL: "http://" + host + ":9601/sse"},
		{Name: "discord", Type: "remote", URL: "http://" + host + ":9602/sse"},
		{Name: "telegram", Type: "remote", URL: "http://" + host + ":9603/sse"},

		// ============================================
		// Productivity MCP Servers (10) - Ports 9701-9710
		// ============================================
		{Name: "notion", Type: "remote", URL: "http://" + host + ":9701/sse"},
		{Name: "linear", Type: "remote", URL: "http://" + host + ":9702/sse"},
		{Name: "jira", Type: "remote", URL: "http://" + host + ":9703/sse"},
		{Name: "asana", Type: "remote", URL: "http://" + host + ":9704/sse"},
		{Name: "trello", Type: "remote", URL: "http://" + host + ":9705/sse"},
		{Name: "todoist", Type: "remote", URL: "http://" + host + ":9706/sse"},
		{Name: "monday", Type: "remote", URL: "http://" + host + ":9707/sse"},
		{Name: "airtable", Type: "remote", URL: "http://" + host + ":9708/sse"},
		{Name: "obsidian", Type: "remote", URL: "http://" + host + ":9709/sse"},
		{Name: "atlassian", Type: "remote", URL: "http://" + host + ":9710/sse"},

		// ============================================
		// Search/AI MCP Servers (10) - Ports 9801-9810
		// ============================================
		{Name: "brave-search", Type: "remote", URL: "http://" + host + ":9801/sse"},
		{Name: "exa", Type: "remote", URL: "http://" + host + ":9802/sse"},
		{Name: "tavily", Type: "remote", URL: "http://" + host + ":9803/sse"},
		{Name: "perplexity", Type: "remote", URL: "http://" + host + ":9804/sse"},
		{Name: "kagi", Type: "remote", URL: "http://" + host + ":9805/sse"},
		{Name: "omnisearch", Type: "remote", URL: "http://" + host + ":9806/sse"},
		{Name: "context7", Type: "remote", URL: "http://" + host + ":9807/sse"},
		{Name: "llamaindex", Type: "remote", URL: "http://" + host + ":9808/sse"},
		{Name: "langchain", Type: "remote", URL: "http://" + host + ":9809/sse"},
		{Name: "openai", Type: "remote", URL: "http://" + host + ":9810/sse"},

		// ============================================
		// Google MCP Servers (5) - Ports 9901-9905
		// ============================================
		{Name: "google-drive", Type: "remote", URL: "http://" + host + ":9901/sse"},
		{Name: "google-calendar", Type: "remote", URL: "http://" + host + ":9902/sse"},
		{Name: "google-maps", Type: "remote", URL: "http://" + host + ":9903/sse"},
		{Name: "youtube", Type: "remote", URL: "http://" + host + ":9904/sse"},
		{Name: "gmail", Type: "remote", URL: "http://" + host + ":9905/sse"},

		// ============================================
		// Monitoring MCP Servers (3) - Ports 9921-9923
		// ============================================
		{Name: "datadog", Type: "remote", URL: "http://" + host + ":9921/sse"},
		{Name: "grafana", Type: "remote", URL: "http://" + host + ":9922/sse"},
		{Name: "prometheus", Type: "remote", URL: "http://" + host + ":9923/sse"},

		// ============================================
		// Finance MCP Servers (3) - Ports 9941-9943
		// ============================================
		{Name: "stripe", Type: "remote", URL: "http://" + host + ":9941/sse"},
		{Name: "hubspot", Type: "remote", URL: "http://" + host + ":9942/sse"},
		{Name: "zendesk", Type: "remote", URL: "http://" + host + ":9943/sse"},

		// ============================================
		// Design MCP Servers (1) - Port 9961
		// ============================================
		{Name: "figma", Type: "remote", URL: "http://" + host + ":9961/sse"},
	}
}

// ContainerizedMCPServersCount returns the total number of containerized MCPs
func ContainerizedMCPServersCount() int {
	return len(ContainerizedMCPServers("localhost"))
}

// registerGenerators registers all 48 agent-specific generators
func (ug *UnifiedGenerator) registerGenerators() {
	// Register generators for primary CLI agents (custom implementations)
	ug.generators[AgentOpenCode] = NewOpenCodeGenerator()
	ug.generators[AgentCrush] = NewCrushGenerator()
	ug.generators[AgentKiloCode] = NewKiloCodeGenerator()
	ug.generators[AgentHelixCode] = NewHelixCodeGenerator()

	// Register generators for Original 18 agents (remaining)
	ug.generators[AgentKiro] = NewKiroGenerator()
	ug.generators[AgentAider] = NewAiderGenerator()
	ug.generators[AgentClaudeCode] = NewClaudeCodeGenerator()
	ug.generators[AgentCline] = NewClineGenerator()
	ug.generators[AgentCodenameGoose] = NewCodenameGooseGenerator()
	ug.generators[AgentDeepSeekCLI] = NewDeepSeekCLIGenerator()
	ug.generators[AgentForge] = NewForgeGenerator()
	ug.generators[AgentGeminiCLI] = NewGeminiCLIGenerator()
	ug.generators[AgentGPTEngineer] = NewGPTEngineerGenerator()
	ug.generators[AgentMistralCode] = NewMistralCodeGenerator()
	ug.generators[AgentOllamaCode] = NewOllamaCodeGenerator()
	ug.generators[AgentPlandex] = NewPlandexGenerator()
	ug.generators[AgentQwenCode] = NewQwenCodeGenerator()
	ug.generators[AgentAmazonQ] = NewAmazonQGenerator()

	// Register generators for New 30 agents
	ug.generators[AgentAgentDeck] = NewAgentDeckGenerator()
	ug.generators[AgentBridle] = NewBridleGenerator()
	ug.generators[AgentCheshireCat] = NewCheshireCatGenerator()
	ug.generators[AgentClaudePlugins] = NewClaudePluginsGenerator()
	ug.generators[AgentClaudeSquad] = NewClaudeSquadGenerator()
	ug.generators[AgentCodai] = NewCodaiGenerator()
	ug.generators[AgentCodex] = NewCodexGenerator()
	ug.generators[AgentCodexSkills] = NewCodexSkillsGenerator()
	ug.generators[AgentConduit] = NewConduitGenerator()
	ug.generators[AgentContinue] = NewContinueGenerator()
	ug.generators[AgentEmdash] = NewEmdashGenerator()
	ug.generators[AgentFauxPilot] = NewFauxPilotGenerator()
	ug.generators[AgentGetShitDone] = NewGetShitDoneGenerator()
	ug.generators[AgentGitHubCopilotCLI] = NewGitHubCopilotCLIGenerator()
	ug.generators[AgentGitHubSpecKit] = NewGitHubSpecKitGenerator()
	ug.generators[AgentGitMCP] = NewGitMCPGenerator()
	ug.generators[AgentGPTME] = NewGPTMEGenerator()
	ug.generators[AgentMobileAgent] = NewMobileAgentGenerator()
	ug.generators[AgentMultiagentCoding] = NewMultiagentCodingGenerator()
	ug.generators[AgentNanocoder] = NewNanocoderGenerator()
	ug.generators[AgentNoi] = NewNoiGenerator()
	ug.generators[AgentOctogen] = NewOctogenGenerator()
	ug.generators[AgentOpenHands] = NewOpenHandsGenerator()
	ug.generators[AgentPostgresMCP] = NewPostgresMCPGenerator()
	ug.generators[AgentShai] = NewShaiGenerator()
	ug.generators[AgentSnowCLI] = NewSnowCLIGenerator()
	ug.generators[AgentTaskWeaver] = NewTaskWeaverGenerator()
	ug.generators[AgentUIUXProMax] = NewUIUXProMaxGenerator()
	ug.generators[AgentVTCode] = NewVTCodeGenerator()
	ug.generators[AgentWarp] = NewWarpGenerator()
}

// Generate generates configuration for a specific agent
func (ug *UnifiedGenerator) Generate(ctx context.Context, agentType AgentType) (*GenerationResult, error) {
	// DEBUG: First line - confirm function entry
	fmt.Fprintf(os.Stderr, "[DEBUG-STDERR] Generate ENTERED: agentType=%q\n", agentType)

	generator, ok := ug.generators[agentType]
	if !ok {
		fmt.Fprintf(os.Stderr, "[DEBUG-STDERR] Generator NOT FOUND for: %q\n", agentType)
		return nil, fmt.Errorf("unsupported agent type: %s", agentType)
	}

	// DEBUG: Log which generator is being used
	fmt.Fprintf(os.Stderr, "[DEBUG-STDERR] Generate: agentType=%s, generator_type=%T\n", agentType, generator)

	result, err := generator.Generate(ctx, ug.config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[DEBUG-STDERR] Generator.Generate FAILED: %v\n", err)
		return nil, err
	}

	// DEBUG: Log the result type
	fmt.Fprintf(os.Stderr, "[DEBUG-STDERR] Result.Config type: %T\n", result.Config)

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
