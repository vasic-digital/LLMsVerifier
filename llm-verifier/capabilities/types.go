// Package capabilities provides dynamic capability detection for LLM providers and CLI agents.
// This is the single source of truth for querying LLM and CLI agent capabilities.
package capabilities

import (
	"time"
)

// StreamingType represents the type of streaming support
type StreamingType string

const (
	StreamingTypeSSE         StreamingType = "sse"             // Server-Sent Events
	StreamingTypeWebSocket   StreamingType = "websocket"       // WebSocket
	StreamingTypeAsyncGen    StreamingType = "async_generator" // AsyncGenerator/yield
	StreamingTypeJSONL       StreamingType = "jsonl"           // JSON Lines streaming
	StreamingTypeMpscStream  StreamingType = "mpsc_stream"     // Rust multi-producer channel
	StreamingTypeEventStream StreamingType = "event_stream"    // AWS EventStream
	StreamingTypeStdout      StreamingType = "stdout"          // Standard output streaming
	StreamingTypeNone        StreamingType = "none"            // No streaming support
)

// HTTPVersion represents supported HTTP versions
type HTTPVersion string

const (
	HTTPVersion1_1 HTTPVersion = "http/1.1"
	HTTPVersion2   HTTPVersion = "http/2"
	HTTPVersion3   HTTPVersion = "http/3" // Note: Currently no CLI agent supports HTTP/3
)

// CompressionType represents compression algorithm support
type CompressionType string

const (
	CompressionGzip     CompressionType = "gzip"
	CompressionBrotli   CompressionType = "brotli"
	CompressionDeflate  CompressionType = "deflate"
	CompressionZstd     CompressionType = "zstd"
	CompressionSemantic CompressionType = "semantic" // Context/semantic compression
	CompressionChat     CompressionType = "chat"     // Chat history compression
	CompressionNone     CompressionType = "none"
)

// CachingType represents caching mechanism support
type CachingType string

const (
	CachingAnthropic CachingType = "anthropic_cache_control" // Anthropic cache_control header
	CachingDashScope CachingType = "dashscope_cache"         // Qwen DashScope X-DashScope-CacheControl
	CachingPrompt    CachingType = "prompt_caching"          // Generic prompt caching
	CachingSemantic  CachingType = "semantic_caching"        // Semantic similarity caching
	CachingLLMOps    CachingType = "llmops_cache"            // LangChain/SQLite cache
	CachingNone      CachingType = "none"
)

// ProtocolType represents supported protocols
type ProtocolType string

const (
	ProtocolMCP       ProtocolType = "mcp"       // Model Context Protocol
	ProtocolACP       ProtocolType = "acp"       // Agent Communication Protocol
	ProtocolLSP       ProtocolType = "lsp"       // Language Server Protocol
	ProtocolGRPC      ProtocolType = "grpc"      // gRPC
	ProtocolOpenAI    ProtocolType = "openai"    // OpenAI API compatible
	ProtocolAnthropic ProtocolType = "anthropic" // Anthropic API
	ProtocolOllama    ProtocolType = "ollama"    // Ollama local API
)

// AuthType represents authentication mechanism
type AuthType string

const (
	AuthAPIKey   AuthType = "api_key"
	AuthBearer   AuthType = "bearer"
	AuthOAuth2   AuthType = "oauth2"
	AuthNone     AuthType = "none" // Free/anonymous access
	AuthAWSSigV4 AuthType = "aws_sig_v4"
)

// ProviderCapabilities represents the full capability set of an LLM provider
type ProviderCapabilities struct {
	Provider   string    `json:"provider"`
	Model      string    `json:"model,omitempty"`
	Verified   bool      `json:"verified"`
	VerifiedAt time.Time `json:"verified_at,omitempty"`

	// Streaming capabilities
	Streaming StreamingCapability `json:"streaming"`

	// HTTP/Network capabilities
	Network NetworkCapability `json:"network"`

	// Compression capabilities
	Compression CompressionCapability `json:"compression"`

	// Caching capabilities
	Caching CachingCapability `json:"caching"`

	// Protocol support
	Protocols []ProtocolType `json:"protocols"`

	// Authentication
	Auth AuthCapability `json:"auth"`

	// Model capabilities
	Model_ ModelCapability `json:"model_capabilities"`

	// Extended features
	Extended ExtendedCapabilities `json:"extended"`

	// Provider-specific features (dynamic)
	Custom map[string]interface{} `json:"custom,omitempty"`

	// CapabilityEvidence maps a capability name (e.g. "tool_use", "rag",
	// "skills", "plugins") to the captured probe artefact that proves its
	// verdict — CONST-040 anti-bluff seam (§11.4.5 / §11.4.69). Storage only;
	// probe wiring lands in a later change.
	CapabilityEvidence map[string]CapabilityEvidenceEntry `json:"capability_evidence,omitempty"`
}

// StreamingCapability details streaming support
type StreamingCapability struct {
	Supported    bool            `json:"supported"`
	Types        []StreamingType `json:"types"`
	DefaultType  StreamingType   `json:"default_type"`
	ChunkTypes   []string        `json:"chunk_types,omitempty"`   // Supported chunk types
	BufferSize   int             `json:"buffer_size,omitempty"`   // Buffer size in bytes
	HeartbeatSec int             `json:"heartbeat_sec,omitempty"` // Heartbeat interval
	MaxDuration  time.Duration   `json:"max_duration,omitempty"`  // Max stream duration
}

// NetworkCapability details network/HTTP support
type NetworkCapability struct {
	HTTPVersions   []HTTPVersion `json:"http_versions"`
	HTTP2Supported bool          `json:"http2_supported"`
	HTTP3Supported bool          `json:"http3_supported"` // Note: Currently false for all CLI agents
	QUICSupported  bool          `json:"quic_supported"`  // Note: Currently false for all CLI agents
	ProxySupported bool          `json:"proxy_supported"`
	TLSVersion     string        `json:"tls_version,omitempty"`
	TimeoutConnect time.Duration `json:"timeout_connect"`
	TimeoutRequest time.Duration `json:"timeout_request"`
	TimeoutStream  time.Duration `json:"timeout_stream"`
	MaxRetries     int           `json:"max_retries"`
}

// CompressionCapability details compression support
type CompressionCapability struct {
	Supported    bool              `json:"supported"`
	Types        []CompressionType `json:"types"`
	DefaultType  CompressionType   `json:"default_type"`
	RequestComp  bool              `json:"request_compression"`  // Compress requests
	ResponseComp bool              `json:"response_compression"` // Handle compressed responses
}

// CachingCapability details caching support
type CachingCapability struct {
	Supported    bool          `json:"supported"`
	Types        []CachingType `json:"types"`
	TTLDefault   time.Duration `json:"ttl_default,omitempty"`
	MaxCacheSize int64         `json:"max_cache_size,omitempty"`
}

// AuthCapability details authentication support
type AuthCapability struct {
	Types          []AuthType `json:"types"`
	DefaultType    AuthType   `json:"default_type"`
	OAuthSupported bool       `json:"oauth_supported"`
	TokenRefresh   bool       `json:"token_refresh"` // Supports automatic token refresh
	EnvVarName     string     `json:"env_var_name,omitempty"`
}

// ModelCapability details model-specific capabilities
type ModelCapability struct {
	Vision           bool `json:"vision"`
	ImageInput       bool `json:"image_input"`
	ImageOutput      bool `json:"image_output"`
	Audio            bool `json:"audio"`
	Video            bool `json:"video"`
	PDF              bool `json:"pdf"`
	OCR              bool `json:"ocr"`
	FunctionCalling  bool `json:"function_calling"`
	ToolUse          bool `json:"tool_use"`
	Embeddings       bool `json:"embeddings"`
	CodeExecution    bool `json:"code_execution"`
	WebBrowsing      bool `json:"web_browsing"`
	Reasoning        bool `json:"reasoning"`        // Extended thinking/reasoning
	SupportsRAG      bool `json:"supports_rag"`     // CONST-040: retrieval-augmented generation
	SupportsSkills   bool `json:"supports_skills"`  // CONST-040: agent skills/tools capability
	SupportsPlugins  bool `json:"supports_plugins"` // CONST-040: plugin invocation capability
	MaxContextTokens int  `json:"max_context_tokens"`
	MaxOutputTokens  int  `json:"max_output_tokens"`
}

// CapabilityEvidenceEntry holds a captured wire artefact proving a single
// capability's verification verdict — the CONST-040 / §11.4.5 / §11.4.69
// anti-bluff seam: every capability flag traces to a real probe request+response.
type CapabilityEvidenceEntry struct {
	Capability  string    `json:"capability"`             // e.g. "tool_use", "rag", "skills", "plugins"
	RawRequest  string    `json:"raw_request,omitempty"`  // captured probe request payload
	RawResponse string    `json:"raw_response,omitempty"` // captured probe response payload
	Verdict     bool      `json:"verdict"`                // true iff the probe confirmed the capability
	Method      string    `json:"method,omitempty"`       // probe method identifier
	VerifiedAt  time.Time `json:"verified_at,omitempty"`  // when the probe ran (CONST-037/038 freshness)
}

// ExtendedCapabilities details additional advanced features
type ExtendedCapabilities struct {
	// Thinking/Reasoning features
	ReasoningEffort  string `json:"reasoning_effort,omitempty"` // low, medium, high
	ThinkingBudget   int    `json:"thinking_budget,omitempty"`  // Token budget for thinking
	ThoughtsIncluded bool   `json:"thoughts_included"`          // Include thought process

	// Multi-mode features
	PlanActModes  bool `json:"plan_act_modes"` // Separate planning/acting
	Checkpointing bool `json:"checkpointing"`  // Git-based checkpoints
	Branching     bool `json:"branching"`      // Plan branching (Plandex)

	// Security features
	Sandboxing   bool     `json:"sandboxing"`              // Container sandbox support
	SandboxTypes []string `json:"sandbox_types,omitempty"` // docker, podman, seatbelt

	// Collaboration features
	DistributedLocking bool `json:"distributed_locking"` // Multi-instance coordination
	InstanceRegistry   bool `json:"instance_registry"`   // Instance discovery

	// Auto features
	AutoApproval bool `json:"auto_approval"` // Automatic action approval
	AutoCommit   bool `json:"auto_commit"`   // Automatic git commits
	AutoContinue bool `json:"auto_continue"` // Continue after success

	// Provider-specific
	Quirks []string `json:"quirks,omitempty"` // Provider-specific quirks
}

// CLIAgentCapabilities represents the capability set of a CLI agent
type CLIAgentCapabilities struct {
	Name         string `json:"name"`
	Language     string `json:"language"`      // Go, TypeScript, Python, Rust
	ConfigFormat string `json:"config_format"` // json, yaml, toml, env
	ConfigPath   string `json:"config_path"`   // Default config file path

	// Streaming
	Streaming StreamingCapability `json:"streaming"`

	// Network
	Network NetworkCapability `json:"network"`

	// Compression
	Compression CompressionCapability `json:"compression"`

	// Caching
	Caching CachingCapability `json:"caching"`

	// Protocols
	Protocols []ProtocolType `json:"protocols"`

	// Providers supported
	ProviderCount int      `json:"provider_count"`
	Providers     []string `json:"providers,omitempty"`

	// Tools supported
	ToolCount int      `json:"tool_count"`
	Tools     []string `json:"tools,omitempty"`

	// Extended features
	Extended ExtendedCapabilities `json:"extended"`

	// Configuration options that can be enabled
	ConfigOptions map[string]ConfigOption `json:"config_options,omitempty"`
}

// ConfigOption represents a configurable option for CLI agents
type ConfigOption struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"` // bool, string, int, array
	Default     interface{} `json:"default"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	EnvVar      string      `json:"env_var,omitempty"`
	ConfigKey   string      `json:"config_key"`
	EnabledBy   string      `json:"enabled_by,omitempty"` // Capability that enables this option
}

// CapabilityQuery represents a query for specific capabilities
type CapabilityQuery struct {
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	CLIAgent string `json:"cli_agent,omitempty"`

	// Query specific capabilities
	RequireStreaming   *StreamingType   `json:"require_streaming,omitempty"`
	RequireHTTP3       bool             `json:"require_http3,omitempty"`
	RequireCompression *CompressionType `json:"require_compression,omitempty"`
	RequireCaching     *CachingType     `json:"require_caching,omitempty"`
	RequireProtocol    *ProtocolType    `json:"require_protocol,omitempty"`
	RequireOAuth       bool             `json:"require_oauth,omitempty"`
	RequireVision      bool             `json:"require_vision,omitempty"`
	RequireFunctions   bool             `json:"require_functions,omitempty"`
	RequireReasoning   bool             `json:"require_reasoning,omitempty"`
	MinContextTokens   int              `json:"min_context_tokens,omitempty"`
}

// CapabilityQueryResult represents the result of a capability query
type CapabilityQueryResult struct {
	Query           *CapabilityQuery      `json:"query"`
	Matches         bool                  `json:"matches"`
	Provider        *ProviderCapabilities `json:"provider,omitempty"`
	CLIAgent        *CLIAgentCapabilities `json:"cli_agent,omitempty"`
	MissingCaps     []string              `json:"missing_capabilities,omitempty"`
	PartialMatch    float64               `json:"partial_match_score"` // 0.0 - 1.0
	Recommendations []string              `json:"recommendations,omitempty"`
}

// CapabilityMatrix represents capability support across all providers/agents
type CapabilityMatrix struct {
	GeneratedAt time.Time                        `json:"generated_at"`
	Providers   map[string]*ProviderCapabilities `json:"providers"`
	CLIAgents   map[string]*CLIAgentCapabilities `json:"cli_agents"`

	// Indexed lookups
	ByStreaming   map[StreamingType][]string   `json:"by_streaming"`
	ByProtocol    map[ProtocolType][]string    `json:"by_protocol"`
	ByCompression map[CompressionType][]string `json:"by_compression"`
	ByCaching     map[CachingType][]string     `json:"by_caching"`
	ByAuth        map[AuthType][]string        `json:"by_auth"`
}
