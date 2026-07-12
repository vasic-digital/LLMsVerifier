package capabilities

import (
	"time"
)

// providerCapabilitySeeds are hand-authored bootstrap DEFAULTS ONLY — they are
// NOT verified and MUST be overridden by a live probe / DB `VerificationResult`
// before being shown to any user, per CONST-036/037/040 (§2.5, 10b §3 C3).
// Every seed's `Verified` field is therefore `false` by construction: a seed is
// unverified until a fresh probe backs it. The fail-closed resolver
// (ResolveModelCapability, registry_resolve.go) prefers a fresh per-model
// `database.VerificationResult` and reports `unverified` when none exists —
// never a hand-authored self-certified literal.
var providerCapabilitySeeds = map[string]*ProviderCapabilities{
	"openai": {
		Provider: "openai",
		Verified: false, // seed: unverified by construction (C3, CONST-036/037/040) — a fresh probe MUST override
		Streaming: StreamingCapability{
			Supported:   true,
			Types:       []StreamingType{StreamingTypeSSE},
			DefaultType: StreamingTypeSSE,
			ChunkTypes:  []string{"text", "function_call", "tool_calls"},
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1, HTTPVersion2},
			HTTP2Supported: true,
			HTTP3Supported: false, // Not supported
			QUICSupported:  false,
			ProxySupported: true,
			TimeoutConnect: 10 * time.Second,
			TimeoutRequest: 60 * time.Second,
			TimeoutStream:  300 * time.Second,
			MaxRetries:     3,
		},
		Compression: CompressionCapability{
			Supported:    true,
			Types:        []CompressionType{CompressionGzip},
			DefaultType:  CompressionGzip,
			ResponseComp: true,
		},
		Caching: CachingCapability{
			Supported: false,
		},
		Protocols: []ProtocolType{ProtocolOpenAI},
		Auth: AuthCapability{
			Types:       []AuthType{AuthBearer},
			DefaultType: AuthBearer,
			EnvVarName:  "OPENAI_API_KEY",
		},
		Model_: ModelCapability{
			Vision:           true,
			ImageInput:       true,
			ImageOutput:      true,
			FunctionCalling:  true,
			ToolUse:          true,
			Embeddings:       true,
			Reasoning:        true,
			MaxContextTokens: 128000,
			MaxOutputTokens:  16384,
		},
		Extended: ExtendedCapabilities{
			ReasoningEffort: "medium",
		},
	},

	"anthropic": {
		Provider: "anthropic",
		Verified: false, // seed: unverified by construction (C3, CONST-036/037/040) — a fresh probe MUST override
		Streaming: StreamingCapability{
			Supported:   true,
			Types:       []StreamingType{StreamingTypeSSE},
			DefaultType: StreamingTypeSSE,
			ChunkTypes:  []string{"content_block_delta", "message_delta", "message_stop"},
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1, HTTPVersion2},
			HTTP2Supported: true,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: true,
			TimeoutConnect: 15 * time.Second,
			TimeoutRequest: 120 * time.Second,
			TimeoutStream:  600 * time.Second,
			MaxRetries:     3,
		},
		Compression: CompressionCapability{
			Supported:    true,
			Types:        []CompressionType{CompressionGzip},
			DefaultType:  CompressionGzip,
			ResponseComp: true,
		},
		Caching: CachingCapability{
			Supported: true,
			Types:     []CachingType{CachingAnthropic, CachingPrompt},
		},
		Protocols: []ProtocolType{ProtocolAnthropic, ProtocolMCP},
		Auth: AuthCapability{
			Types:          []AuthType{AuthBearer, AuthOAuth2},
			DefaultType:    AuthBearer,
			OAuthSupported: true,
			TokenRefresh:   true,
			EnvVarName:     "ANTHROPIC_API_KEY",
		},
		Model_: ModelCapability{
			Vision:           true,
			ImageInput:       true,
			PDF:              true,
			FunctionCalling:  false,
			ToolUse:          true,
			Embeddings:       false,
			Reasoning:        true,
			MaxContextTokens: 200000,
			MaxOutputTokens:  8192,
		},
		Extended: ExtendedCapabilities{
			ReasoningEffort:  "medium",
			ThinkingBudget:   10000,
			ThoughtsIncluded: true,
		},
	},

	"deepseek": {
		Provider: "deepseek",
		Verified: false, // seed: unverified by construction (C3, CONST-036/037/040) — a fresh probe MUST override
		Streaming: StreamingCapability{
			Supported:   true,
			Types:       []StreamingType{StreamingTypeSSE},
			DefaultType: StreamingTypeSSE,
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1, HTTPVersion2},
			HTTP2Supported: true,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: true,
			TimeoutConnect: 10 * time.Second,
			TimeoutRequest: 60 * time.Second,
			TimeoutStream:  300 * time.Second,
			MaxRetries:     3,
		},
		Compression: CompressionCapability{
			Supported: false,
		},
		Caching: CachingCapability{
			Supported: false,
		},
		Protocols: []ProtocolType{ProtocolOpenAI},
		Auth: AuthCapability{
			Types:       []AuthType{AuthBearer},
			DefaultType: AuthBearer,
			EnvVarName:  "DEEPSEEK_API_KEY",
		},
		Model_: ModelCapability{
			FunctionCalling:  true,
			ToolUse:          true,
			Reasoning:        true,
			MaxContextTokens: 128000,
			MaxOutputTokens:  8192,
		},
		Extended: ExtendedCapabilities{
			ReasoningEffort:  "high",
			ThoughtsIncluded: true,
		},
	},

	"gemini": {
		Provider: "gemini",
		Verified: false, // seed: unverified by construction (C3, CONST-036/037/040) — a fresh probe MUST override
		Streaming: StreamingCapability{
			Supported:   true,
			Types:       []StreamingType{StreamingTypeJSONL, StreamingTypeSSE},
			DefaultType: StreamingTypeJSONL,
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1, HTTPVersion2},
			HTTP2Supported: true,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: true,
			TimeoutConnect: 10 * time.Second,
			TimeoutRequest: 60 * time.Second,
			TimeoutStream:  300 * time.Second,
			MaxRetries:     3,
		},
		Compression: CompressionCapability{
			Supported:   true,
			Types:       []CompressionType{CompressionChat, CompressionSemantic},
			DefaultType: CompressionChat,
		},
		Caching: CachingCapability{
			Supported: true,
			Types:     []CachingType{CachingPrompt},
		},
		Protocols: []ProtocolType{ProtocolOpenAI},
		Auth: AuthCapability{
			Types:          []AuthType{AuthAPIKey, AuthOAuth2},
			DefaultType:    AuthAPIKey,
			OAuthSupported: true,
			TokenRefresh:   true,
			EnvVarName:     "GEMINI_API_KEY",
		},
		Model_: ModelCapability{
			Vision:           true,
			ImageInput:       true,
			ImageOutput:      true,
			Audio:            true,
			Video:            true,
			PDF:              true,
			FunctionCalling:  true,
			ToolUse:          true,
			Embeddings:       true,
			CodeExecution:    true,
			Reasoning:        true,
			MaxContextTokens: 1000000,
			MaxOutputTokens:  8192,
		},
		Extended: ExtendedCapabilities{
			ThinkingBudget:   24000,
			ThoughtsIncluded: true,
		},
	},

	"qwen": {
		Provider: "qwen",
		Verified: false, // seed: unverified by construction (C3, CONST-036/037/040) — a fresh probe MUST override
		Streaming: StreamingCapability{
			Supported:   true,
			Types:       []StreamingType{StreamingTypeSSE},
			DefaultType: StreamingTypeSSE,
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1, HTTPVersion2},
			HTTP2Supported: true,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: true,
			TimeoutConnect: 10 * time.Second,
			TimeoutRequest: 60 * time.Second,
			TimeoutStream:  300 * time.Second,
			MaxRetries:     3,
		},
		Compression: CompressionCapability{
			Supported: false,
		},
		Caching: CachingCapability{
			Supported: true,
			Types:     []CachingType{CachingDashScope, CachingPrompt},
		},
		Protocols: []ProtocolType{ProtocolOpenAI},
		Auth: AuthCapability{
			Types:          []AuthType{AuthBearer, AuthOAuth2},
			DefaultType:    AuthBearer,
			OAuthSupported: true,
			TokenRefresh:   true,
			EnvVarName:     "DASHSCOPE_API_KEY",
		},
		Model_: ModelCapability{
			Vision:           true,
			ImageInput:       true,
			FunctionCalling:  true,
			ToolUse:          true,
			Embeddings:       true,
			CodeExecution:    true,
			MaxContextTokens: 128000,
			MaxOutputTokens:  8192,
		},
	},

	"groq": {
		Provider: "groq",
		Verified: false, // seed: unverified by construction (C3, CONST-036/037/040) — a fresh probe MUST override
		Streaming: StreamingCapability{
			Supported:   true,
			Types:       []StreamingType{StreamingTypeSSE},
			DefaultType: StreamingTypeSSE,
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1, HTTPVersion2},
			HTTP2Supported: true,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: true,
			TimeoutConnect: 10 * time.Second,
			TimeoutRequest: 30 * time.Second,
			TimeoutStream:  120 * time.Second,
			MaxRetries:     3,
		},
		Compression: CompressionCapability{
			Supported: false,
		},
		Caching: CachingCapability{
			Supported: false,
		},
		Protocols: []ProtocolType{ProtocolOpenAI},
		Auth: AuthCapability{
			Types:       []AuthType{AuthBearer},
			DefaultType: AuthBearer,
			EnvVarName:  "GROQ_API_KEY",
		},
		Model_: ModelCapability{
			FunctionCalling:  true,
			ToolUse:          true,
			MaxContextTokens: 32768,
			MaxOutputTokens:  8192,
		},
	},

	"mistral": {
		Provider: "mistral",
		Verified: false, // seed: unverified by construction (C3, CONST-036/037/040) — a fresh probe MUST override
		Streaming: StreamingCapability{
			Supported:   true,
			Types:       []StreamingType{StreamingTypeSSE},
			DefaultType: StreamingTypeSSE,
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1, HTTPVersion2},
			HTTP2Supported: true,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: true,
			TimeoutConnect: 10 * time.Second,
			TimeoutRequest: 60 * time.Second,
			TimeoutStream:  300 * time.Second,
			MaxRetries:     3,
		},
		Compression: CompressionCapability{
			Supported: false,
		},
		Caching: CachingCapability{
			Supported: false,
		},
		Protocols: []ProtocolType{ProtocolOpenAI},
		Auth: AuthCapability{
			Types:       []AuthType{AuthBearer},
			DefaultType: AuthBearer,
			EnvVarName:  "MISTRAL_API_KEY",
		},
		Model_: ModelCapability{
			Vision:           true,
			ImageInput:       true,
			FunctionCalling:  true,
			ToolUse:          true,
			Embeddings:       true,
			MaxContextTokens: 128000,
			MaxOutputTokens:  8192,
		},
	},

	"zen": {
		Provider: "zen",
		Verified: false, // seed: unverified by construction (C3, CONST-036/037/040) — a fresh probe MUST override
		Streaming: StreamingCapability{
			Supported:   true,
			Types:       []StreamingType{StreamingTypeSSE},
			DefaultType: StreamingTypeSSE,
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1, HTTPVersion2},
			HTTP2Supported: true,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: true,
			TimeoutConnect: 10 * time.Second,
			TimeoutRequest: 60 * time.Second,
			TimeoutStream:  300 * time.Second,
			MaxRetries:     3,
		},
		Compression: CompressionCapability{
			Supported: false,
		},
		Caching: CachingCapability{
			Supported: false,
		},
		Protocols: []ProtocolType{ProtocolOpenAI},
		Auth: AuthCapability{
			Types:       []AuthType{AuthBearer, AuthNone},
			DefaultType: AuthBearer,
			EnvVarName:  "OPENCODE_API_KEY",
		},
		Model_: ModelCapability{
			FunctionCalling:  true,
			ToolUse:          true,
			MaxContextTokens: 128000,
			MaxOutputTokens:  8192,
		},
		Custom: map[string]interface{}{
			"free_tier": true,
		},
	},

	"ollama": {
		Provider: "ollama",
		Verified: false, // seed: unverified by construction (C3, CONST-036/037/040) — a fresh probe MUST override
		Streaming: StreamingCapability{
			Supported:   true,
			Types:       []StreamingType{StreamingTypeJSONL},
			DefaultType: StreamingTypeJSONL,
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1},
			HTTP2Supported: false,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: false,
			TimeoutConnect: 5 * time.Second,
			TimeoutRequest: 120 * time.Second,
			TimeoutStream:  600 * time.Second,
			MaxRetries:     2,
		},
		Compression: CompressionCapability{
			Supported: false,
		},
		Caching: CachingCapability{
			Supported: false,
		},
		Protocols: []ProtocolType{ProtocolOllama},
		Auth: AuthCapability{
			Types:       []AuthType{AuthNone},
			DefaultType: AuthNone,
		},
		Model_: ModelCapability{
			Vision:           true,
			ImageInput:       true,
			FunctionCalling:  false,
			MaxContextTokens: 32768,
			MaxOutputTokens:  4096,
		},
		Custom: map[string]interface{}{
			"local_only": true,
		},
	},

	// helixllm — in-repo HelixLLM local coder (Phase A, providers-coverage
	// EXPANSION_PLAN_v2.md §3 Phase A). Mirrors the "ollama" seed shape for a
	// local, no-credential OpenAI-compatible server. LIVE-CONFIRMED this
	// session: the coder's llama-server sidecar serves /v1/chat/completions +
	// /v1/models with no Authorization required. As with every seed in this
	// map, these are hand-authored bootstrap DEFAULTS ONLY (Verified: false);
	// the real capability flags come from the C4/C5 probe
	// (verification.Verifier.Verify), never this literal.
	"helixllm": {
		Provider: "helixllm",
		Verified: false, // seed: unverified by construction (C3, CONST-036/037/040) — a fresh probe MUST override
		Streaming: StreamingCapability{
			Supported:   true,
			Types:       []StreamingType{StreamingTypeSSE},
			DefaultType: StreamingTypeSSE,
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1},
			HTTP2Supported: false,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: false,
			TimeoutConnect: 5 * time.Second,
			TimeoutRequest: 120 * time.Second,
			TimeoutStream:  600 * time.Second,
			MaxRetries:     2,
		},
		Compression: CompressionCapability{
			Supported: false,
		},
		Caching: CachingCapability{
			Supported: false,
		},
		Protocols: []ProtocolType{ProtocolOpenAI},
		Auth: AuthCapability{
			Types:       []AuthType{AuthNone},
			DefaultType: AuthNone,
		},
		Model_: ModelCapability{
			FunctionCalling:  false, // CONST-040: real value sourced from the C4 probe, not hardcoded
			MaxContextTokens: 32768,
			MaxOutputTokens:  4096,
		},
		Custom: map[string]interface{}{
			"local_only": true,
		},
	},
}

// CLI Agent capability registry - based on source code exploration
var cliAgentCapabilities = map[string]*CLIAgentCapabilities{
	"opencode": {
		Name:         "opencode",
		Language:     "Go",
		ConfigFormat: "json",
		ConfigPath:   "~/.config/opencode/opencode.json",
		Streaming: StreamingCapability{
			Supported:   true,
			Types:       []StreamingType{StreamingTypeSSE},
			DefaultType: StreamingTypeSSE,
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1, HTTPVersion2},
			HTTP2Supported: true,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: true,
			TimeoutConnect: 10 * time.Second,
			TimeoutRequest: 30 * time.Second,
			TimeoutStream:  300 * time.Second,
			MaxRetries:     3,
		},
		Compression: CompressionCapability{
			Supported: false,
		},
		Caching: CachingCapability{
			Supported: true,
			Types:     []CachingType{CachingAnthropic, CachingPrompt},
		},
		Protocols:     []ProtocolType{ProtocolOpenAI, ProtocolMCP},
		ProviderCount: 15,
		Providers:     []string{"openai", "anthropic", "deepseek", "gemini", "groq", "mistral", "ollama", "zen"},
		ToolCount:     21,
		Tools:         []string{"Bash", "Read", "Write", "Edit", "Glob", "Grep", "Git", "Diff", "Task", "Test", "Lint"},
		Extended: ExtendedCapabilities{
			AutoApproval: true,
			AutoContinue: true,
		},
	},

	"claudecode": {
		Name:         "claudecode",
		Language:     "TypeScript",
		ConfigFormat: "json",
		ConfigPath:   "~/.claude/settings.json",
		Streaming: StreamingCapability{
			Supported:   true,
			Types:       []StreamingType{StreamingTypeSSE, StreamingTypeWebSocket},
			DefaultType: StreamingTypeSSE,
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1, HTTPVersion2},
			HTTP2Supported: true,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: true,
			TimeoutConnect: 10 * time.Second,
			TimeoutRequest: 120 * time.Second,
			TimeoutStream:  600 * time.Second,
			MaxRetries:     3,
		},
		Compression: CompressionCapability{
			Supported: false,
		},
		Caching: CachingCapability{
			Supported: true,
			Types:     []CachingType{CachingAnthropic},
		},
		Protocols:     []ProtocolType{ProtocolAnthropic, ProtocolMCP},
		ProviderCount: 1,
		Providers:     []string{"anthropic"},
		ToolCount:     21,
		Tools:         []string{"Bash", "Read", "Write", "Edit", "Glob", "Grep", "Git", "Task", "WebFetch", "WebSearch"},
		Extended: ExtendedCapabilities{
			ReasoningEffort: "medium",
			AutoApproval:    true,
		},
	},

	"kilocode": {
		Name:         "kilocode",
		Language:     "TypeScript",
		ConfigFormat: "json",
		ConfigPath:   "~/.kilocode/settings.json",
		Streaming: StreamingCapability{
			Supported:   true,
			Types:       []StreamingType{StreamingTypeAsyncGen},
			DefaultType: StreamingTypeAsyncGen,
			ChunkTypes:  []string{"text", "reasoning", "tool_call", "error", "usage", "status"},
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1, HTTPVersion2},
			HTTP2Supported: true,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: true,
			TimeoutConnect: 10 * time.Second,
			TimeoutRequest: 60 * time.Second,
			TimeoutStream:  300 * time.Second,
			MaxRetries:     3,
		},
		Compression: CompressionCapability{
			Supported: false,
		},
		Caching: CachingCapability{
			Supported:  true,
			Types:      []CachingType{CachingPrompt},
			TTLDefault: 24 * time.Hour,
		},
		Protocols:     []ProtocolType{ProtocolOpenAI, ProtocolAnthropic, ProtocolMCP},
		ProviderCount: 43,
		Providers:     []string{"openai", "anthropic", "deepseek", "gemini", "groq", "mistral", "ollama", "openrouter"},
		ToolCount:     28,
		Tools:         []string{"Bash", "Read", "Write", "Edit", "Glob", "Grep", "Git", "Diff", "Test", "Lint", "Symbols", "References", "Definition", "PR", "Issue", "Workflow"},
		Extended: ExtendedCapabilities{
			PlanActModes:  true,
			AutoApproval:  true,
			Checkpointing: true,
		},
	},

	"cline": {
		Name:         "cline",
		Language:     "TypeScript",
		ConfigFormat: "json",
		ConfigPath:   "~/.cline/settings.json",
		Streaming: StreamingCapability{
			Supported:   true,
			Types:       []StreamingType{StreamingTypeAsyncGen},
			DefaultType: StreamingTypeAsyncGen,
			ChunkTypes:  []string{"text", "usage", "thinking", "tool_calls"},
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1, HTTPVersion2},
			HTTP2Supported: true,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: true,
			TimeoutConnect: 10 * time.Second,
			TimeoutRequest: 60 * time.Second,
			TimeoutStream:  300 * time.Second,
			MaxRetries:     3,
		},
		Compression: CompressionCapability{
			Supported: false,
		},
		Caching: CachingCapability{
			Supported: true,
			Types:     []CachingType{CachingAnthropic},
		},
		Protocols:     []ProtocolType{ProtocolOpenAI, ProtocolAnthropic, ProtocolGRPC},
		ProviderCount: 41,
		Providers:     []string{"openai", "anthropic", "deepseek", "gemini", "groq", "ollama", "openrouter", "bedrock", "vertex"},
		ToolCount:     15,
		Tools:         []string{"Bash", "Read", "Write", "Edit", "Diff", "Browser"},
		Extended: ExtendedCapabilities{
			ReasoningEffort:    "medium",
			ThinkingBudget:     10000,
			PlanActModes:       true,
			Checkpointing:      true,
			DistributedLocking: true,
			InstanceRegistry:   true,
			AutoApproval:       true,
		},
	},

	"aider": {
		Name:         "aider",
		Language:     "Python",
		ConfigFormat: "yaml",
		ConfigPath:   "~/.aider.conf.yml",
		Streaming: StreamingCapability{
			Supported:   true,
			Types:       []StreamingType{StreamingTypeStdout},
			DefaultType: StreamingTypeStdout,
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1, HTTPVersion2},
			HTTP2Supported: true,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: true,
			TimeoutConnect: 10 * time.Second,
			TimeoutRequest: 60 * time.Second,
			TimeoutStream:  300 * time.Second,
			MaxRetries:     3,
		},
		Compression: CompressionCapability{
			Supported: false,
		},
		Caching: CachingCapability{
			Supported: true,
			Types:     []CachingType{CachingPrompt},
		},
		Protocols:     []ProtocolType{ProtocolOpenAI, ProtocolAnthropic, ProtocolOllama},
		ProviderCount: 10,
		Providers:     []string{"openai", "anthropic", "deepseek", "gemini", "ollama", "azure"},
		ToolCount:     8,
		Tools:         []string{"Edit", "Git", "Voice", "Browser"},
		Extended: ExtendedCapabilities{
			AutoCommit:   true,
			AutoContinue: true,
		},
	},

	"amazonq": {
		Name:         "amazonq",
		Language:     "Rust",
		ConfigFormat: "json",
		ConfigPath:   "~/.aws/amazonq/settings.json",
		Streaming: StreamingCapability{
			Supported:   true,
			Types:       []StreamingType{StreamingTypeEventStream},
			DefaultType: StreamingTypeEventStream,
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1, HTTPVersion2},
			HTTP2Supported: true,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: true,
			TimeoutConnect: 10 * time.Second,
			TimeoutRequest: 60 * time.Second,
			TimeoutStream:  300 * time.Second,
			MaxRetries:     3,
		},
		Compression: CompressionCapability{
			Supported:    true,
			Types:        []CompressionType{CompressionGzip},
			DefaultType:  CompressionGzip,
			RequestComp:  true,
			ResponseComp: true,
		},
		Caching: CachingCapability{
			Supported: false,
		},
		Protocols:     []ProtocolType{ProtocolOpenAI, ProtocolMCP},
		ProviderCount: 2,
		Providers:     []string{"bedrock", "amazonq"},
		ToolCount:     12,
		Tools:         []string{"Bash", "Read", "Write", "Edit", "Git", "AWS"},
		Extended: ExtendedCapabilities{
			AutoApproval: true,
		},
	},

	"forge": {
		Name:         "forge",
		Language:     "Rust",
		ConfigFormat: "yaml",
		ConfigPath:   "~/.config/forge/config.yaml",
		Streaming: StreamingCapability{
			Supported:   true,
			Types:       []StreamingType{StreamingTypeMpscStream},
			DefaultType: StreamingTypeMpscStream,
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1, HTTPVersion2},
			HTTP2Supported: true,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: true,
			TimeoutConnect: 10 * time.Second,
			TimeoutRequest: 60 * time.Second,
			TimeoutStream:  300 * time.Second,
			MaxRetries:     3,
		},
		Compression: CompressionCapability{
			Supported:   true,
			Types:       []CompressionType{CompressionSemantic},
			DefaultType: CompressionSemantic,
		},
		Caching: CachingCapability{
			Supported: false,
		},
		Protocols:     []ProtocolType{ProtocolOpenAI, ProtocolAnthropic, ProtocolOllama},
		ProviderCount: 8,
		Providers:     []string{"openai", "anthropic", "ollama", "groq"},
		ToolCount:     10,
		Tools:         []string{"Bash", "Read", "Write", "Edit", "Ripgrep"},
		Extended: ExtendedCapabilities{
			AutoApproval: true,
		},
	},

	"plandex": {
		Name:         "plandex",
		Language:     "Go",
		ConfigFormat: "json",
		ConfigPath:   "~/.plandex/config.json",
		Streaming: StreamingCapability{
			Supported:    true,
			Types:        []StreamingType{StreamingTypeSSE},
			DefaultType:  StreamingTypeSSE,
			HeartbeatSec: 5,
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1, HTTPVersion2},
			HTTP2Supported: true,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: true,
			TimeoutConnect: 10 * time.Second,
			TimeoutRequest: 300 * time.Second,
			TimeoutStream:  600 * time.Second,
			MaxRetries:     3,
		},
		Compression: CompressionCapability{
			Supported:   true,
			Types:       []CompressionType{CompressionGzip},
			DefaultType: CompressionGzip,
		},
		Caching: CachingCapability{
			Supported: true,
			Types:     []CachingType{CachingAnthropic, CachingPrompt},
		},
		Protocols:     []ProtocolType{ProtocolOpenAI},
		ProviderCount: 10,
		Providers:     []string{"openai", "anthropic", "azure", "bedrock"},
		ToolCount:     8,
		Tools:         []string{"Read", "Write", "Edit", "Browser"},
		Extended: ExtendedCapabilities{
			Branching:    true,
			AutoCommit:   true,
			AutoContinue: true,
		},
	},

	"kiro": {
		Name:         "kiro",
		Language:     "Python",
		ConfigFormat: "yaml",
		ConfigPath:   ".kiro/steering/",
		Streaming: StreamingCapability{
			Supported:   true,
			Types:       []StreamingType{StreamingTypeSSE},
			DefaultType: StreamingTypeSSE,
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1, HTTPVersion2},
			HTTP2Supported: true,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: true,
			TimeoutConnect: 10 * time.Second,
			TimeoutRequest: 60 * time.Second,
			TimeoutStream:  300 * time.Second,
			MaxRetries:     3,
		},
		Compression: CompressionCapability{
			Supported:    true,
			Types:        []CompressionType{CompressionGzip},
			DefaultType:  CompressionGzip,
			ResponseComp: true,
		},
		Caching: CachingCapability{
			Supported: false,
		},
		Protocols:     []ProtocolType{ProtocolOpenAI},
		ProviderCount: 12,
		Providers:     []string{"copilot", "claude", "gemini", "cursor", "qwen", "opencode", "codex", "windsurf", "kilocode", "ollama"},
		ToolCount:     5,
		Tools:         []string{"Specify", "Plan", "Tasks", "Implement"},
		Extended: ExtendedCapabilities{
			PlanActModes: true,
		},
	},

	"gptengineer": {
		Name:         "gptengineer",
		Language:     "Python",
		ConfigFormat: "toml",
		ConfigPath:   "gpt-engineer.toml",
		Streaming: StreamingCapability{
			Supported:   true,
			Types:       []StreamingType{StreamingTypeStdout},
			DefaultType: StreamingTypeStdout,
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1},
			HTTP2Supported: false,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: true,
			TimeoutConnect: 10 * time.Second,
			TimeoutRequest: 60 * time.Second,
			TimeoutStream:  300 * time.Second,
			MaxRetries:     3,
		},
		Compression: CompressionCapability{
			Supported: false,
		},
		Caching: CachingCapability{
			Supported: true,
			Types:     []CachingType{CachingLLMOps},
		},
		Protocols:     []ProtocolType{ProtocolOpenAI, ProtocolAnthropic},
		ProviderCount: 4,
		Providers:     []string{"openai", "anthropic", "azure"},
		ToolCount:     5,
		Tools:         []string{"Generate", "Improve", "Test"},
		Extended: ExtendedCapabilities{
			AutoContinue: true,
		},
	},

	"ollamacode": {
		Name:         "ollamacode",
		Language:     "TypeScript",
		ConfigFormat: "json",
		ConfigPath:   "~/.ollama/settings.json",
		Streaming: StreamingCapability{
			Supported:   true,
			Types:       []StreamingType{StreamingTypeAsyncGen},
			DefaultType: StreamingTypeAsyncGen,
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1},
			HTTP2Supported: false,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: false,
			TimeoutConnect: 5 * time.Second,
			TimeoutRequest: 120 * time.Second,
			TimeoutStream:  600 * time.Second,
			MaxRetries:     2,
		},
		Compression: CompressionCapability{
			Supported:   true,
			Types:       []CompressionType{CompressionChat},
			DefaultType: CompressionChat,
		},
		Caching: CachingCapability{
			Supported: false,
		},
		Protocols:     []ProtocolType{ProtocolOllama, ProtocolOpenAI},
		ProviderCount: 2,
		Providers:     []string{"ollama", "openai"},
		ToolCount:     8,
		Tools:         []string{"Bash", "Read", "Write", "Edit", "Memory"},
		Extended: ExtendedCapabilities{
			Sandboxing:    true,
			SandboxTypes:  []string{"docker", "podman", "seatbelt"},
			Checkpointing: true,
			AutoApproval:  true,
		},
	},

	"deepseekcli": {
		Name:         "deepseekcli",
		Language:     "TypeScript",
		ConfigFormat: "env",
		ConfigPath:   ".env",
		Streaming: StreamingCapability{
			Supported:   false,
			Types:       []StreamingType{StreamingTypeNone},
			DefaultType: StreamingTypeNone,
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1},
			HTTP2Supported: false,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: true,
			TimeoutConnect: 10 * time.Second,
			TimeoutRequest: 30 * time.Second,
			TimeoutStream:  120 * time.Second,
			MaxRetries:     3,
		},
		Compression: CompressionCapability{
			Supported: false,
		},
		Caching: CachingCapability{
			Supported: false,
		},
		Protocols:     []ProtocolType{ProtocolOpenAI, ProtocolOllama},
		ProviderCount: 2,
		Providers:     []string{"deepseek", "ollama"},
		ToolCount:     3,
		Tools:         []string{"Chat", "Setup", "Interactive"},
	},

	"geminicli": {
		Name:         "geminicli",
		Language:     "TypeScript",
		ConfigFormat: "json",
		ConfigPath:   "~/.config/gemini-cli/config.json",
		Streaming: StreamingCapability{
			Supported:   true,
			Types:       []StreamingType{StreamingTypeJSONL},
			DefaultType: StreamingTypeJSONL,
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1, HTTPVersion2},
			HTTP2Supported: true,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: true,
			TimeoutConnect: 10 * time.Second,
			TimeoutRequest: 60 * time.Second,
			TimeoutStream:  300 * time.Second,
			MaxRetries:     3,
		},
		Compression: CompressionCapability{
			Supported:   true,
			Types:       []CompressionType{CompressionChat},
			DefaultType: CompressionChat,
		},
		Caching: CachingCapability{
			Supported: false,
		},
		Protocols:     []ProtocolType{ProtocolOpenAI},
		ProviderCount: 1,
		Providers:     []string{"gemini"},
		ToolCount:     10,
		Tools:         []string{"Bash", "Read", "Write", "Edit", "Glob", "Grep", "Git"},
		Extended: ExtendedCapabilities{
			ThinkingBudget:   24000,
			ThoughtsIncluded: true,
		},
	},

	"qwencode": {
		Name:         "qwencode",
		Language:     "TypeScript",
		ConfigFormat: "json",
		ConfigPath:   "~/.qwen/oauth_creds.json",
		Streaming: StreamingCapability{
			Supported:   true,
			Types:       []StreamingType{StreamingTypeSSE},
			DefaultType: StreamingTypeSSE,
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1, HTTPVersion2},
			HTTP2Supported: true,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: true,
			TimeoutConnect: 10 * time.Second,
			TimeoutRequest: 60 * time.Second,
			TimeoutStream:  300 * time.Second,
			MaxRetries:     3,
		},
		Compression: CompressionCapability{
			Supported: false,
		},
		Caching: CachingCapability{
			Supported: true,
			Types:     []CachingType{CachingDashScope},
		},
		Protocols:     []ProtocolType{ProtocolOpenAI},
		ProviderCount: 1,
		Providers:     []string{"qwen"},
		ToolCount:     10,
		Tools:         []string{"Bash", "Read", "Write", "Edit", "Glob", "Grep"},
		Extended: ExtendedCapabilities{
			AutoApproval: true,
		},
	},

	"crush": {
		Name:         "crush",
		Language:     "TypeScript",
		ConfigFormat: "json",
		ConfigPath:   "~/.config/crush/crush.json",
		Streaming: StreamingCapability{
			Supported:   true,
			Types:       []StreamingType{StreamingTypeSSE},
			DefaultType: StreamingTypeSSE,
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1, HTTPVersion2},
			HTTP2Supported: true,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: true,
			TimeoutConnect: 10 * time.Second,
			TimeoutRequest: 60 * time.Second,
			TimeoutStream:  300 * time.Second,
			MaxRetries:     3,
		},
		Compression: CompressionCapability{
			Supported: false,
		},
		Caching: CachingCapability{
			Supported: false,
		},
		Protocols:     []ProtocolType{ProtocolOpenAI},
		ProviderCount: 10,
		Providers:     []string{"openai", "anthropic", "deepseek", "gemini", "ollama"},
		ToolCount:     12,
		Tools:         []string{"Bash", "Read", "Write", "Edit", "Glob", "Grep", "Git"},
	},

	"helixcode": {
		Name:         "helixcode",
		Language:     "Go",
		ConfigFormat: "json",
		ConfigPath:   "~/.config/helixcode/config.json",
		Streaming: StreamingCapability{
			Supported:   true,
			Types:       []StreamingType{StreamingTypeSSE},
			DefaultType: StreamingTypeSSE,
		},
		Network: NetworkCapability{
			HTTPVersions:   []HTTPVersion{HTTPVersion1_1, HTTPVersion2},
			HTTP2Supported: true,
			HTTP3Supported: false,
			QUICSupported:  false,
			ProxySupported: true,
			TimeoutConnect: 10 * time.Second,
			TimeoutRequest: 60 * time.Second,
			TimeoutStream:  300 * time.Second,
			MaxRetries:     3,
		},
		Compression: CompressionCapability{
			Supported: false,
		},
		Caching: CachingCapability{
			Supported: true,
			Types:     []CachingType{CachingPrompt, CachingSemantic},
		},
		Protocols:     []ProtocolType{ProtocolOpenAI, ProtocolMCP, ProtocolACP},
		ProviderCount: 18,
		Providers:     []string{"helixagent", "openai", "anthropic", "deepseek", "gemini", "qwen", "groq", "mistral", "ollama", "zen"},
		ToolCount:     21,
		Tools:         []string{"Bash", "Read", "Write", "Edit", "Glob", "Grep", "Git", "Diff", "Task", "Test", "Lint", "TreeView", "FileInfo", "Symbols", "References", "Definition", "PR", "Issue", "Workflow", "WebFetch", "WebSearch"},
		Extended: ExtendedCapabilities{
			PlanActModes:  true,
			AutoApproval:  true,
			AutoCommit:    true,
			AutoContinue:  true,
			Checkpointing: true,
		},
	},
}

// GetProviderBaseCapabilities returns the base (SEED) capabilities for a
// provider. The returned struct is a COPY of the seed so a caller mutating
// `.Verified` (e.g. a detector that flips it true after a probe) can NEVER
// write that state back into the shared registry seed — the seed stays
// `Verified:false` (unverified-by-construction, C3 / CONST-036/037/040).
// Callers that need a probe-backed, fail-closed verdict MUST use
// ResolveModelCapability (registry_resolve.go), which prefers a fresh
// database.VerificationResult and fail-closes to unverified when none exists.
func GetProviderBaseCapabilities(provider string) *ProviderCapabilities {
	if caps, ok := providerCapabilitySeeds[provider]; ok {
		seedCopy := *caps // shallow copy isolates the value fields (incl. Verified)
		return &seedCopy
	}
	return nil
}

// GetCLIAgentCapabilities returns capabilities for a CLI agent
func GetCLIAgentCapabilities(agent string) *CLIAgentCapabilities {
	if caps, ok := cliAgentCapabilities[agent]; ok {
		return caps
	}
	return nil
}

// GetAllProviders returns all registered provider names
func GetAllProviders() []string {
	providers := make([]string, 0, len(providerCapabilitySeeds))
	for name := range providerCapabilitySeeds {
		providers = append(providers, name)
	}
	return providers
}

// GetAllCLIAgents returns all registered CLI agent names
func GetAllCLIAgents() []string {
	agents := make([]string, 0, len(cliAgentCapabilities))
	for name := range cliAgentCapabilities {
		agents = append(agents, name)
	}
	return agents
}

// GetProvidersWithCapability returns providers that support a specific capability
func GetProvidersWithCapability(capName string, capValue interface{}) []string {
	var result []string

	for name, caps := range providerCapabilitySeeds {
		switch capName {
		case "streaming":
			if caps.Streaming.Supported {
				result = append(result, name)
			}
		case "http3":
			if caps.Network.HTTP3Supported {
				result = append(result, name)
			}
		case "http2":
			if caps.Network.HTTP2Supported {
				result = append(result, name)
			}
		case "oauth":
			if caps.Auth.OAuthSupported {
				result = append(result, name)
			}
		case "vision":
			if caps.Model_.Vision {
				result = append(result, name)
			}
		case "function_calling":
			if caps.Model_.FunctionCalling {
				result = append(result, name)
			}
		case "reasoning":
			if caps.Model_.Reasoning {
				result = append(result, name)
			}
		case "embeddings":
			if caps.Model_.Embeddings {
				result = append(result, name)
			}
		case "caching":
			if caps.Caching.Supported {
				result = append(result, name)
			}
		case "compression":
			if caps.Compression.Supported {
				result = append(result, name)
			}
		}
	}

	return result
}

// GetCLIAgentsWithCapability returns CLI agents that support a specific capability
func GetCLIAgentsWithCapability(capName string) []string {
	var result []string

	for name, caps := range cliAgentCapabilities {
		switch capName {
		case "streaming":
			if caps.Streaming.Supported {
				result = append(result, name)
			}
		case "http3":
			if caps.Network.HTTP3Supported {
				result = append(result, name)
			}
		case "http2":
			if caps.Network.HTTP2Supported {
				result = append(result, name)
			}
		case "compression":
			if caps.Compression.Supported {
				result = append(result, name)
			}
		case "caching":
			if caps.Caching.Supported {
				result = append(result, name)
			}
		case "mcp":
			for _, p := range caps.Protocols {
				if p == ProtocolMCP {
					result = append(result, name)
					break
				}
			}
		case "checkpointing":
			if caps.Extended.Checkpointing {
				result = append(result, name)
			}
		case "sandboxing":
			if caps.Extended.Sandboxing {
				result = append(result, name)
			}
		case "auto_approval":
			if caps.Extended.AutoApproval {
				result = append(result, name)
			}
		}
	}

	return result
}
