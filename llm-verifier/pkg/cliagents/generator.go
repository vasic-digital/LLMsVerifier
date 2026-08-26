// Package cliagents provides unified CLI agent configuration generation and validation.
// This is the central authority for all CLI agent configuration generation in HelixAgent.
// All 16+ CLI agents should have their configurations generated and validated through this package.
package cliagents

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"digital.vasic.llmsverifier/pkg/helixendpoint"
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
	baseURL := helixendpoint.BaseURL(host, port)
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
	baseURL := helixendpoint.BaseURL(host, port)
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

// NewUnifiedGenerator creates a new unified configuration generator.
//
// Ownership / mutation semantics (HXC-250): config is retained BY REFERENCE
// (unchanged from before HXC-250) and its MCPServers field is REASSIGNED to a
// slice whose HelixAgent entries target HelixAgentHost:HelixAgentPort, so a
// caller re-reading config.MCPServers after construction observes the
// retargeted entries. The caller's original backing array is never written
// through (RetargetHelixAgentMCPServers returns a copy) and the operation is
// idempotent. Callers who need their slice left untouched should pass a copy.
func NewUnifiedGenerator(config *GeneratorConfig) *UnifiedGenerator {
	if config == nil {
		config = DefaultGeneratorConfig()
	}

	ug := &UnifiedGenerator{
		config:     config,
		generators: make(map[AgentType]AgentGenerator),
	}

	// HXC-250: make HelixAgentHost/HelixAgentPort authoritative for the
	// HelixAgent MCP endpoints. Without this a consumer doing the natural
	// thing — DefaultGeneratorConfig() then setting Host/Port — kept the
	// MCP URLs built at DefaultGeneratorConfig() time, so every generated
	// agent config pointed at the placeholder endpoint instead of the
	// injected one. Consumer-supplied non-HelixAgent MCP entries are
	// untouched.
	//
	// Side effect, deliberate and documented: config is caller-owned and is
	// retained by reference (unchanged from before HXC-250), so the caller
	// observes its MCPServers field pointing at the retargeted slice. The
	// caller's original backing array is NOT written through — Retarget
	// returns a copy — and the operation is idempotent, so re-constructing a
	// generator from the same config, or from one already retargeted to a
	// different endpoint, yields the same result as a fresh derivation.
	ug.config.MCPServers = RetargetHelixAgentMCPServers(
		ug.config.MCPServers, ug.config.HelixAgentHost, ug.config.HelixAgentPort)

	// Register all supported generators
	ug.registerGenerators()

	return ug
}

// helixAgentMCPNamePrefix marks the MCP entries whose endpoint is owned by the
// HelixAgent service (and therefore follows HelixAgentHost/HelixAgentPort).
const helixAgentMCPNamePrefix = "helixagent-"

// RetargetHelixAgentMCPServers returns servers with every HelixAgent-owned
// remote entry re-pointed at host:port, preserving each entry's path and query.
// Entries that are not HelixAgent-owned (npx-local servers, third-party remote
// MCPs, anything the consumer added) are returned unchanged.
//
// A nil/empty input yields the full default set for host:port, so a consumer
// that leaves MCPServers unset still ships the documented MCP servers pointed
// at its own endpoint.
func RetargetHelixAgentMCPServers(servers []MCPServerConfig, host string, port int) []MCPServerConfig {
	if len(servers) == 0 {
		return DefaultMCPServersForHost(host, port)
	}

	baseURL := helixendpoint.BaseURL(host, port)
	out := make([]MCPServerConfig, len(servers))
	copy(out, servers)

	for i := range out {
		if out[i].Type != "remote" || !strings.HasPrefix(out[i].Name, helixAgentMCPNamePrefix) {
			continue
		}
		parsed, err := url.Parse(out[i].URL)
		// A scheme-less or opaque URL ("localhost:8100/v1/mcp", "mailto:x")
		// parses without error but carries no recoverable host/path, so
		// treat it as unusable alongside a hard parse error rather than
		// silently emitting a bare base URL with the path dropped.
		if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.Opaque != "" {
			// Unusable URL: rebuild from the entry name so the endpoint is
			// still injected rather than silently left stale (§11.4.6 — do not
			// guess that a broken URL was already correct).
			out[i].URL = helixendpoint.JoinPath(baseURL,
				"v1/"+strings.TrimPrefix(out[i].Name, helixAgentMCPNamePrefix))
			continue
		}
		suffix := parsed.EscapedPath()
		if parsed.RawQuery != "" {
			suffix += "?" + parsed.RawQuery
		}
		if parsed.Fragment != "" {
			suffix += "#" + parsed.EscapedFragment()
		}
		out[i].URL = helixendpoint.JoinPath(baseURL, suffix)
	}

	return out
}

// DefaultGeneratorConfig returns default generator configuration.
//
// The HelixAgent endpoint is RESOLVED, not hardcoded: it comes from the
// consuming project's injected environment when present and from the
// documented placeholder otherwise. See package helixendpoint.
func DefaultGeneratorConfig() *GeneratorConfig {
	homeDir, _ := os.UserHomeDir()
	host := helixendpoint.Host()
	port := helixendpoint.Port()
	return &GeneratorConfig{
		HelixAgentHost: host,
		HelixAgentPort: port,
		OutputDir:      filepath.Join(homeDir, "Downloads"),
		IncludeScores:  true,
		MCPServers:     DefaultMCPServersForHost(host, port),
	}
}

// DefaultMCPServers returns default MCP server configurations (15+ servers)
// Includes: HelixAgent remote endpoints + npx-based local servers + free remote MCPs
// All agents MUST ship with 15+ MCP servers out of the box.
//
// The HelixAgent endpoint is resolved from the consuming project's injected
// configuration (see package helixendpoint), never hardcoded here.
func DefaultMCPServers() []MCPServerConfig {
	return DefaultMCPServersForHost(helixendpoint.Host(), helixendpoint.Port())
}

// DefaultMCPServersForHost returns default MCP servers for a given host and port
func DefaultMCPServersForHost(host string, port int) []MCPServerConfig {
	baseURL := helixendpoint.BaseURL(host, port)
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
	return ContainerizedMCPServersForHostPort(host, helixendpoint.Port())
}

// ContainerizedMCPServersForHostPort returns the containerized MCP servers with
// the HelixAgent endpoint injected explicitly.
//
// HXC-250: the six HelixAgent entries previously ignored the caller's host and
// pinned a literal endpoint, so a consumer generating configs for a non-local
// deployment still shipped URLs pointing at the generating machine.
func ContainerizedMCPServersForHostPort(host string, port int) []MCPServerConfig {
	if strings.TrimSpace(host) == "" {
		host = helixendpoint.Host()
	}

	helixBase := helixendpoint.BaseURL(host, port)
	// sse builds the containerized-MCP URLs, IPv6-safe (an unbracketed literal
	// would otherwise produce "http://::1:9101/sse").
	sse := func(p int) string { return helixendpoint.JoinPath(helixendpoint.BaseURL(host, p), "sse") }

	return []MCPServerConfig{
		// ============================================
		// HelixAgent Remote Endpoints (6)
		// ============================================
		{Name: "helixagent-mcp", Type: "remote", URL: helixendpoint.JoinPath(helixBase, "v1/mcp")},
		{Name: "helixagent-acp", Type: "remote", URL: helixendpoint.JoinPath(helixBase, "v1/acp")},
		{Name: "helixagent-lsp", Type: "remote", URL: helixendpoint.JoinPath(helixBase, "v1/lsp")},
		{Name: "helixagent-embeddings", Type: "remote", URL: helixendpoint.JoinPath(helixBase, "v1/embeddings")},
		{Name: "helixagent-vision", Type: "remote", URL: helixendpoint.JoinPath(helixBase, "v1/vision")},
		{Name: "helixagent-cognee", Type: "remote", URL: helixendpoint.JoinPath(helixBase, "v1/cognee")},

		// ============================================
		// Core MCP Servers (10) - Ports 9101-9110
		// ============================================
		{Name: "fetch", Type: "remote", URL: sse(9101)},
		{Name: "git", Type: "remote", URL: sse(9102)},
		{Name: "time", Type: "remote", URL: sse(9103)},
		{Name: "filesystem", Type: "remote", URL: sse(9104)},
		{Name: "memory", Type: "remote", URL: sse(9105)},
		{Name: "everything", Type: "remote", URL: sse(9106)},
		{Name: "sequential-thinking", Type: "remote", URL: sse(9107)},
		{Name: "sqlite", Type: "remote", URL: sse(9108)},
		{Name: "puppeteer", Type: "remote", URL: sse(9109)},
		{Name: "postgres", Type: "remote", URL: sse(9110)},

		// ============================================
		// Database MCP Servers (5) - Ports 9201-9205
		// ============================================
		{Name: "mongodb", Type: "remote", URL: sse(9201)},
		{Name: "redis", Type: "remote", URL: sse(9202)},
		{Name: "mysql", Type: "remote", URL: sse(9203)},
		{Name: "elasticsearch", Type: "remote", URL: sse(9204)},
		{Name: "supabase", Type: "remote", URL: sse(9205)},

		// ============================================
		// Vector DB MCP Servers (4) - Ports 9301-9304
		// ============================================
		{Name: "qdrant", Type: "remote", URL: sse(9301)},
		{Name: "chroma", Type: "remote", URL: sse(9302)},
		{Name: "pinecone", Type: "remote", URL: sse(9303)},
		{Name: "weaviate", Type: "remote", URL: sse(9304)},

		// ============================================
		// DevOps MCP Servers (14) - Ports 9401-9414
		// ============================================
		{Name: "github", Type: "remote", URL: sse(9401)},
		{Name: "gitlab", Type: "remote", URL: sse(9402)},
		{Name: "sentry", Type: "remote", URL: sse(9403)},
		{Name: "kubernetes", Type: "remote", URL: sse(9404)},
		{Name: "docker", Type: "remote", URL: sse(9405)},
		{Name: "ansible", Type: "remote", URL: sse(9406)},
		{Name: "aws", Type: "remote", URL: sse(9407)},
		{Name: "gcp", Type: "remote", URL: sse(9408)},
		{Name: "heroku", Type: "remote", URL: sse(9409)},
		{Name: "cloudflare", Type: "remote", URL: sse(9410)},
		{Name: "vercel", Type: "remote", URL: sse(9411)},
		{Name: "workers", Type: "remote", URL: sse(9412)},
		{Name: "jetbrains", Type: "remote", URL: sse(9413)},

		// ============================================
		// Browser MCP Servers (4) - Ports 9501-9504
		// ============================================
		{Name: "playwright", Type: "remote", URL: sse(9501)},
		{Name: "browserbase", Type: "remote", URL: sse(9502)},
		{Name: "firecrawl", Type: "remote", URL: sse(9503)},
		{Name: "crawl4ai", Type: "remote", URL: sse(9504)},

		// ============================================
		// Communication MCP Servers (3) - Ports 9601-9603
		// ============================================
		{Name: "slack", Type: "remote", URL: sse(9601)},
		{Name: "discord", Type: "remote", URL: sse(9602)},
		{Name: "telegram", Type: "remote", URL: sse(9603)},

		// ============================================
		// Productivity MCP Servers (10) - Ports 9701-9710
		// ============================================
		{Name: "notion", Type: "remote", URL: sse(9701)},
		{Name: "linear", Type: "remote", URL: sse(9702)},
		{Name: "jira", Type: "remote", URL: sse(9703)},
		{Name: "asana", Type: "remote", URL: sse(9704)},
		{Name: "trello", Type: "remote", URL: sse(9705)},
		{Name: "todoist", Type: "remote", URL: sse(9706)},
		{Name: "monday", Type: "remote", URL: sse(9707)},
		{Name: "airtable", Type: "remote", URL: sse(9708)},
		{Name: "obsidian", Type: "remote", URL: sse(9709)},
		{Name: "atlassian", Type: "remote", URL: sse(9710)},

		// ============================================
		// Search/AI MCP Servers (10) - Ports 9801-9810
		// ============================================
		{Name: "brave-search", Type: "remote", URL: sse(9801)},
		{Name: "exa", Type: "remote", URL: sse(9802)},
		{Name: "tavily", Type: "remote", URL: sse(9803)},
		{Name: "perplexity", Type: "remote", URL: sse(9804)},
		{Name: "kagi", Type: "remote", URL: sse(9805)},
		{Name: "omnisearch", Type: "remote", URL: sse(9806)},
		{Name: "context7", Type: "remote", URL: sse(9807)},
		{Name: "llamaindex", Type: "remote", URL: sse(9808)},
		{Name: "langchain", Type: "remote", URL: sse(9809)},
		{Name: "openai", Type: "remote", URL: sse(9810)},

		// ============================================
		// Google MCP Servers (5) - Ports 9901-9905
		// ============================================
		{Name: "google-drive", Type: "remote", URL: sse(9901)},
		{Name: "google-calendar", Type: "remote", URL: sse(9902)},
		{Name: "google-maps", Type: "remote", URL: sse(9903)},
		{Name: "youtube", Type: "remote", URL: sse(9904)},
		{Name: "gmail", Type: "remote", URL: sse(9905)},

		// ============================================
		// Monitoring MCP Servers (3) - Ports 9921-9923
		// ============================================
		{Name: "datadog", Type: "remote", URL: sse(9921)},
		{Name: "grafana", Type: "remote", URL: sse(9922)},
		{Name: "prometheus", Type: "remote", URL: sse(9923)},

		// ============================================
		// Finance MCP Servers (3) - Ports 9941-9943
		// ============================================
		{Name: "stripe", Type: "remote", URL: sse(9941)},
		{Name: "hubspot", Type: "remote", URL: sse(9942)},
		{Name: "zendesk", Type: "remote", URL: sse(9943)},

		// ============================================
		// Design MCP Servers (1) - Port 9961
		// ============================================
		{Name: "figma", Type: "remote", URL: sse(9961)},
	}
}

// ContainerizedMCPServersCount returns the total number of containerized MCPs
func ContainerizedMCPServersCount() int {
	// The count is host-independent; "" selects the resolved default host
	// rather than pinning a literal here (HXC-250).
	return len(ContainerizedMCPServers(""))
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
