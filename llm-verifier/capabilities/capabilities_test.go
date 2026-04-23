package capabilities

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Type Tests
// =============================================================================

func TestStreamingTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant StreamingType
		expected string
	}{
		{"SSE", StreamingTypeSSE, "sse"},
		{"WebSocket", StreamingTypeWebSocket, "websocket"},
		{"AsyncGenerator", StreamingTypeAsyncGen, "async_generator"},
		{"JSONL", StreamingTypeJSONL, "jsonl"},
		{"MpscStream", StreamingTypeMpscStream, "mpsc_stream"},
		{"EventStream", StreamingTypeEventStream, "event_stream"},
		{"Stdout", StreamingTypeStdout, "stdout"},
		{"None", StreamingTypeNone, "none"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.constant))
		})
	}
}

func TestHTTPVersionConstants(t *testing.T) {
	assert.Equal(t, "http/1.1", string(HTTPVersion1_1))
	assert.Equal(t, "http/2", string(HTTPVersion2))
	assert.Equal(t, "http/3", string(HTTPVersion3))
}

func TestCompressionTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant CompressionType
		expected string
	}{
		{"Gzip", CompressionGzip, "gzip"},
		{"Brotli", CompressionBrotli, "brotli"},
		{"Deflate", CompressionDeflate, "deflate"},
		{"Zstd", CompressionZstd, "zstd"},
		{"Semantic", CompressionSemantic, "semantic"},
		{"Chat", CompressionChat, "chat"},
		{"None", CompressionNone, "none"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.constant))
		})
	}
}

func TestCachingTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant CachingType
		expected string
	}{
		{"Anthropic", CachingAnthropic, "anthropic_cache_control"},
		{"DashScope", CachingDashScope, "dashscope_cache"},
		{"Prompt", CachingPrompt, "prompt_caching"},
		{"Semantic", CachingSemantic, "semantic_caching"},
		{"LLMOps", CachingLLMOps, "llmops_cache"},
		{"None", CachingNone, "none"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.constant))
		})
	}
}

func TestProtocolTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant ProtocolType
		expected string
	}{
		{"MCP", ProtocolMCP, "mcp"},
		{"ACP", ProtocolACP, "acp"},
		{"LSP", ProtocolLSP, "lsp"},
		{"gRPC", ProtocolGRPC, "grpc"},
		{"OpenAI", ProtocolOpenAI, "openai"},
		{"Anthropic", ProtocolAnthropic, "anthropic"},
		{"Ollama", ProtocolOllama, "ollama"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.constant))
		})
	}
}

func TestAuthTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant AuthType
		expected string
	}{
		{"APIKey", AuthAPIKey, "api_key"},
		{"Bearer", AuthBearer, "bearer"},
		{"OAuth2", AuthOAuth2, "oauth2"},
		{"None", AuthNone, "none"},
		{"AWSSigV4", AuthAWSSigV4, "aws_sig_v4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.constant))
		})
	}
}

// =============================================================================
// Registry Tests
// =============================================================================

func TestGetAllProviders(t *testing.T) {
	providers := GetAllProviders()

	// Should have at least the core providers
	assert.GreaterOrEqual(t, len(providers), 8, "Should have at least 8 providers")

	// Check for specific providers
	providerSet := make(map[string]bool)
	for _, p := range providers {
		providerSet[p] = true
	}

	expectedProviders := []string{"openai", "anthropic", "deepseek", "gemini", "qwen", "groq", "mistral", "zen"}
	for _, expected := range expectedProviders {
		assert.True(t, providerSet[expected], "Should have provider: %s", expected)
	}
}

func TestGetAllCLIAgents(t *testing.T) {
	agents := GetAllCLIAgents()

	// Should have at least 16 CLI agents
	assert.GreaterOrEqual(t, len(agents), 16, "Should have at least 16 CLI agents")

	// Check for specific agents
	agentSet := make(map[string]bool)
	for _, a := range agents {
		agentSet[a] = true
	}

	expectedAgents := []string{
		"opencode", "claudecode", "kilocode", "cline", "aider",
		"amazonq", "forge", "plandex", "kiro", "crush", "helixcode",
	}
	for _, expected := range expectedAgents {
		assert.True(t, agentSet[expected], "Should have CLI agent: %s", expected)
	}
}

func TestGetProviderBaseCapabilities(t *testing.T) {
	tests := []struct {
		provider        string
		expectStreaming bool
		expectHTTP2     bool
		expectHTTP3     bool // Currently all providers should be false
	}{
		{"openai", true, true, false},
		{"anthropic", true, true, false},
		{"deepseek", true, true, false},
		{"gemini", true, true, false},
		{"qwen", true, true, false},
		{"groq", true, true, false},
		{"mistral", true, true, false},
		{"zen", true, true, false},
		{"ollama", true, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			caps := GetProviderBaseCapabilities(tt.provider)
			require.NotNil(t, caps, "Provider %s should have capabilities", tt.provider)

			assert.Equal(t, tt.expectStreaming, caps.Streaming.Supported, "Streaming support mismatch")
			assert.Equal(t, tt.expectHTTP2, caps.Network.HTTP2Supported, "HTTP/2 support mismatch")
			assert.Equal(t, tt.expectHTTP3, caps.Network.HTTP3Supported, "HTTP/3 should be false for all providers")
		})
	}
}

func TestGetProviderBaseCapabilities_Unknown(t *testing.T) {
	caps := GetProviderBaseCapabilities("unknown_provider")
	assert.Nil(t, caps, "Unknown provider should return nil")
}

func TestGetCLIAgentCapabilities(t *testing.T) {
	tests := []struct {
		agent           string
		language        string
		configFormat    string
		expectStreaming bool
		expectHTTP3     bool // Currently all agents should be false
	}{
		{"opencode", "Go", "json", true, false},
		{"claudecode", "TypeScript", "json", true, false},
		{"kilocode", "TypeScript", "json", true, false},
		{"cline", "TypeScript", "json", true, false},
		{"aider", "Python", "yaml", true, false},
		{"amazonq", "Rust", "json", true, false},
		{"forge", "Rust", "yaml", true, false},
		{"plandex", "Go", "json", true, false},
		{"kiro", "Python", "yaml", true, false},
		{"ollamacode", "TypeScript", "json", true, false},
		{"deepseekcli", "TypeScript", "env", false, false}, // DeepSeek CLI doesn't support streaming
	}

	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			caps := GetCLIAgentCapabilities(tt.agent)
			require.NotNil(t, caps, "CLI agent %s should have capabilities", tt.agent)

			assert.Equal(t, tt.language, caps.Language, "Language mismatch")
			assert.Equal(t, tt.configFormat, caps.ConfigFormat, "Config format mismatch")
			assert.Equal(t, tt.expectStreaming, caps.Streaming.Supported, "Streaming support mismatch")
			assert.Equal(t, tt.expectHTTP3, caps.Network.HTTP3Supported, "HTTP/3 should be false for all CLI agents")
		})
	}
}

func TestGetCLIAgentCapabilities_Unknown(t *testing.T) {
	caps := GetCLIAgentCapabilities("unknown_agent")
	assert.Nil(t, caps, "Unknown agent should return nil")
}

// =============================================================================
// HTTP/3 Support Tests (Critical: All should be false)
// =============================================================================

func TestNoHTTP3Support_Providers(t *testing.T) {
	t.Log("CRITICAL TEST: Verifying no provider supports HTTP/3 (as discovered in source code analysis)")

	for _, provider := range GetAllProviders() {
		caps := GetProviderBaseCapabilities(provider)
		if caps != nil {
			assert.False(t, caps.Network.HTTP3Supported,
				"Provider %s should NOT support HTTP/3 (no provider currently does)", provider)
			assert.False(t, caps.Network.QUICSupported,
				"Provider %s should NOT support QUIC (no provider currently does)", provider)
		}
	}
}

func TestNoHTTP3Support_CLIAgents(t *testing.T) {
	t.Log("CRITICAL TEST: Verifying no CLI agent supports HTTP/3 (as discovered in source code analysis)")

	for _, agent := range GetAllCLIAgents() {
		caps := GetCLIAgentCapabilities(agent)
		if caps != nil {
			assert.False(t, caps.Network.HTTP3Supported,
				"CLI agent %s should NOT support HTTP/3 (no agent currently does)", agent)
			assert.False(t, caps.Network.QUICSupported,
				"CLI agent %s should NOT support QUIC (no agent currently does)", agent)
		}
	}
}

// =============================================================================
// Streaming Type Tests
// =============================================================================

func TestStreamingTypes_ByAgent(t *testing.T) {
	expectedStreamTypes := map[string][]StreamingType{
		"opencode":    {StreamingTypeSSE},
		"claudecode":  {StreamingTypeSSE, StreamingTypeWebSocket},
		"kilocode":    {StreamingTypeAsyncGen},
		"cline":       {StreamingTypeAsyncGen},
		"aider":       {StreamingTypeStdout},
		"amazonq":     {StreamingTypeEventStream},
		"forge":       {StreamingTypeMpscStream},
		"plandex":     {StreamingTypeSSE},
		"geminicli":   {StreamingTypeJSONL},
		"deepseekcli": {StreamingTypeNone},
	}

	for agent, expectedTypes := range expectedStreamTypes {
		t.Run(agent, func(t *testing.T) {
			caps := GetCLIAgentCapabilities(agent)
			require.NotNil(t, caps)

			for _, expectedType := range expectedTypes {
				found := false
				for _, actualType := range caps.Streaming.Types {
					if actualType == expectedType {
						found = true
						break
					}
				}
				assert.True(t, found, "Agent %s should support streaming type: %s", agent, expectedType)
			}
		})
	}
}

// =============================================================================
// Compression Tests
// =============================================================================

func TestCompressionSupport(t *testing.T) {
	// Only specific agents support compression
	compressionAgents := map[string]CompressionType{
		"amazonq":    CompressionGzip,
		"forge":      CompressionSemantic,
		"plandex":    CompressionGzip,
		"ollamacode": CompressionChat,
		"geminicli":  CompressionChat,
		"kiro":       CompressionGzip,
	}

	for agent, expectedType := range compressionAgents {
		t.Run(agent, func(t *testing.T) {
			caps := GetCLIAgentCapabilities(agent)
			require.NotNil(t, caps)

			assert.True(t, caps.Compression.Supported, "Agent %s should support compression", agent)
			assert.Contains(t, caps.Compression.Types, expectedType,
				"Agent %s should support %s compression", agent, expectedType)
		})
	}
}

func TestNoCompressionSupport(t *testing.T) {
	// Agents that don't support compression
	noCompressionAgents := []string{"opencode", "claudecode", "kilocode", "cline", "crush"}

	for _, agent := range noCompressionAgents {
		t.Run(agent, func(t *testing.T) {
			caps := GetCLIAgentCapabilities(agent)
			require.NotNil(t, caps)
			assert.False(t, caps.Compression.Supported, "Agent %s should NOT support compression", agent)
		})
	}
}

// =============================================================================
// Caching Tests
// =============================================================================

func TestCachingSupport(t *testing.T) {
	cachingAgents := map[string][]CachingType{
		"opencode":    {CachingAnthropic, CachingPrompt},
		"claudecode":  {CachingAnthropic},
		"kilocode":    {CachingPrompt},
		"cline":       {CachingAnthropic},
		"aider":       {CachingPrompt},
		"plandex":     {CachingAnthropic, CachingPrompt},
		"qwencode":    {CachingDashScope},
		"gptengineer": {CachingLLMOps},
		"helixcode":   {CachingPrompt, CachingSemantic},
	}

	for agent, expectedTypes := range cachingAgents {
		t.Run(agent, func(t *testing.T) {
			caps := GetCLIAgentCapabilities(agent)
			require.NotNil(t, caps)

			assert.True(t, caps.Caching.Supported, "Agent %s should support caching", agent)
			for _, expectedType := range expectedTypes {
				assert.Contains(t, caps.Caching.Types, expectedType,
					"Agent %s should support %s caching", agent, expectedType)
			}
		})
	}
}

// =============================================================================
// Protocol Tests
// =============================================================================

func TestMCPSupport(t *testing.T) {
	mcpAgents := []string{"opencode", "claudecode", "amazonq", "helixcode"}

	for _, agent := range mcpAgents {
		t.Run(agent, func(t *testing.T) {
			caps := GetCLIAgentCapabilities(agent)
			require.NotNil(t, caps)
			assert.Contains(t, caps.Protocols, ProtocolMCP, "Agent %s should support MCP", agent)
		})
	}
}

func TestOAuthSupport_Providers(t *testing.T) {
	oauthProviders := []string{"anthropic", "gemini", "qwen"}

	for _, provider := range oauthProviders {
		t.Run(provider, func(t *testing.T) {
			caps := GetProviderBaseCapabilities(provider)
			require.NotNil(t, caps)
			assert.True(t, caps.Auth.OAuthSupported, "Provider %s should support OAuth", provider)
		})
	}
}

// =============================================================================
// Extended Features Tests
// =============================================================================

func TestExtendedFeatures_PlanActModes(t *testing.T) {
	planActAgents := []string{"kilocode", "cline", "kiro", "helixcode"}

	for _, agent := range planActAgents {
		t.Run(agent, func(t *testing.T) {
			caps := GetCLIAgentCapabilities(agent)
			require.NotNil(t, caps)
			assert.True(t, caps.Extended.PlanActModes, "Agent %s should support plan/act modes", agent)
		})
	}
}

func TestExtendedFeatures_Checkpointing(t *testing.T) {
	checkpointAgents := []string{"kilocode", "cline", "ollamacode", "helixcode"}

	for _, agent := range checkpointAgents {
		t.Run(agent, func(t *testing.T) {
			caps := GetCLIAgentCapabilities(agent)
			require.NotNil(t, caps)
			assert.True(t, caps.Extended.Checkpointing, "Agent %s should support checkpointing", agent)
		})
	}
}

func TestExtendedFeatures_Sandboxing(t *testing.T) {
	caps := GetCLIAgentCapabilities("ollamacode")
	require.NotNil(t, caps)
	assert.True(t, caps.Extended.Sandboxing, "OllamaCode should support sandboxing")
	assert.Contains(t, caps.Extended.SandboxTypes, "docker")
	assert.Contains(t, caps.Extended.SandboxTypes, "podman")
	assert.Contains(t, caps.Extended.SandboxTypes, "seatbelt")
}

func TestExtendedFeatures_Branching(t *testing.T) {
	caps := GetCLIAgentCapabilities("plandex")
	require.NotNil(t, caps)
	assert.True(t, caps.Extended.Branching, "Plandex should support branching")
}

// =============================================================================
// Provider Count Tests
// =============================================================================

func TestProviderCounts(t *testing.T) {
	tests := []struct {
		agent         string
		minProviders  int
	}{
		{"kilocode", 40},   // 43+ providers
		{"cline", 40},      // 41+ providers
		{"opencode", 10},   // 15+ providers
		{"helixcode", 15},  // 18+ providers
		{"aider", 5},       // 10+ providers
	}

	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			caps := GetCLIAgentCapabilities(tt.agent)
			require.NotNil(t, caps)
			assert.GreaterOrEqual(t, caps.ProviderCount, tt.minProviders,
				"Agent %s should support at least %d providers", tt.agent, tt.minProviders)
		})
	}
}

// =============================================================================
// Tool Count Tests
// =============================================================================

func TestToolCounts(t *testing.T) {
	tests := []struct {
		agent     string
		minTools  int
	}{
		{"kilocode", 25},   // 28 tools
		{"helixcode", 20},  // 21 tools (all tools)
		{"opencode", 10},   // 21 tools
		{"claudecode", 10}, // 21 tools
		{"cline", 10},      // 15+ tools
	}

	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			caps := GetCLIAgentCapabilities(tt.agent)
			require.NotNil(t, caps)
			assert.GreaterOrEqual(t, caps.ToolCount, tt.minTools,
				"Agent %s should support at least %d tools", tt.agent, tt.minTools)
		})
	}
}

// =============================================================================
// Detector Tests
// =============================================================================

func TestNewDetector(t *testing.T) {
	detector := NewDetector()
	require.NotNil(t, detector)
	assert.NotNil(t, detector.httpClient)
	assert.NotNil(t, detector.cache)
}

func TestDetector_Query_Provider(t *testing.T) {
	detector := NewDetector()
	ctx := context.Background()

	// Query OpenAI capabilities
	sseType := StreamingTypeSSE
	query := &CapabilityQuery{
		Provider:          "openai",
		RequireStreaming:  &sseType,
	}

	result, err := detector.Query(ctx, query)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.Matches, "OpenAI should match SSE streaming requirement")
	assert.NotNil(t, result.Provider)
	assert.Equal(t, 1.0, result.PartialMatch)
}

func TestDetector_Query_HTTP3Requirement(t *testing.T) {
	detector := NewDetector()
	ctx := context.Background()

	// Query for HTTP/3 (should fail for all providers)
	query := &CapabilityQuery{
		Provider:     "openai",
		RequireHTTP3: true,
	}

	result, err := detector.Query(ctx, query)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.False(t, result.Matches, "No provider supports HTTP/3")
	assert.Contains(t, result.MissingCaps, "http3")
	assert.Contains(t, result.Recommendations[0], "HTTP/3 is not currently supported")
}

func TestDetector_Query_UnknownProvider(t *testing.T) {
	detector := NewDetector()
	ctx := context.Background()

	query := &CapabilityQuery{
		Provider: "unknown_provider",
	}

	result, err := detector.Query(ctx, query)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.False(t, result.Matches)
	assert.Contains(t, result.MissingCaps, "provider_not_found")
}

func TestDetector_GetCapabilityMatrix(t *testing.T) {
	detector := NewDetector()
	matrix := detector.GetCapabilityMatrix()

	require.NotNil(t, matrix)
	assert.False(t, matrix.GeneratedAt.IsZero())
	assert.NotEmpty(t, matrix.Providers)
	assert.NotEmpty(t, matrix.CLIAgents)

	// Check indexed lookups
	assert.NotEmpty(t, matrix.ByStreaming)
	assert.NotEmpty(t, matrix.ByProtocol)
	assert.NotEmpty(t, matrix.ByAuth)
}

// =============================================================================
// Capability Lookup Tests
// =============================================================================

func TestGetProvidersWithCapability(t *testing.T) {
	// Test streaming capability
	streamingProviders := GetProvidersWithCapability("streaming", nil)
	assert.NotEmpty(t, streamingProviders)
	assert.Contains(t, streamingProviders, "openai")
	assert.Contains(t, streamingProviders, "anthropic")

	// Test OAuth capability
	oauthProviders := GetProvidersWithCapability("oauth", nil)
	assert.NotEmpty(t, oauthProviders)
	assert.Contains(t, oauthProviders, "anthropic")
	assert.Contains(t, oauthProviders, "qwen")

	// Test vision capability
	visionProviders := GetProvidersWithCapability("vision", nil)
	assert.NotEmpty(t, visionProviders)
	assert.Contains(t, visionProviders, "openai")
	assert.Contains(t, visionProviders, "gemini")

	// Test HTTP/3 capability (should be empty)
	http3Providers := GetProvidersWithCapability("http3", nil)
	assert.Empty(t, http3Providers, "No providers should support HTTP/3")
}

func TestGetCLIAgentsWithCapability(t *testing.T) {
	// Test streaming capability
	streamingAgents := GetCLIAgentsWithCapability("streaming")
	assert.NotEmpty(t, streamingAgents)
	assert.Contains(t, streamingAgents, "opencode")
	assert.Contains(t, streamingAgents, "claudecode")

	// Test MCP capability
	mcpAgents := GetCLIAgentsWithCapability("mcp")
	assert.NotEmpty(t, mcpAgents)
	assert.Contains(t, mcpAgents, "opencode")
	assert.Contains(t, mcpAgents, "claudecode")

	// Test checkpointing capability
	checkpointAgents := GetCLIAgentsWithCapability("checkpointing")
	assert.NotEmpty(t, checkpointAgents)
	assert.Contains(t, checkpointAgents, "kilocode")
	assert.Contains(t, checkpointAgents, "ollamacode")

	// Test HTTP/3 capability (should be empty)
	http3Agents := GetCLIAgentsWithCapability("http3")
	assert.Empty(t, http3Agents, "No CLI agents should support HTTP/3")
}

// =============================================================================
// Config Generator Tests
// =============================================================================

func TestNewConfigGenerator(t *testing.T) {
	generator := NewConfigGenerator("localhost", 8100)
	require.NotNil(t, generator)
	assert.Equal(t, "localhost", generator.baseHost)
	assert.Equal(t, 8100, generator.basePort)
}

func TestConfigGenerator_GenerateForAgent(t *testing.T) {
	generator := NewConfigGenerator("localhost", 8100)

	agents := []string{
		"opencode", "claudecode", "kilocode", "cline", "crush",
		"helixcode", "aider", "plandex", "kiro", "forge",
		"amazonq", "ollamacode", "geminicli", "qwencode",
	}

	for _, agent := range agents {
		t.Run(agent, func(t *testing.T) {
			config, err := generator.GenerateForAgent(agent, nil)
			require.NoError(t, err)
			require.NotNil(t, config)

			assert.Equal(t, agent, config.Agent)
			assert.False(t, config.GeneratedAt.IsZero())
			assert.NotEmpty(t, config.ConfigPath)
			assert.NotEmpty(t, config.Format)
			assert.NotEmpty(t, config.Content)
		})
	}
}

func TestConfigGenerator_GenerateForAgent_Unknown(t *testing.T) {
	generator := NewConfigGenerator("localhost", 8100)

	_, err := generator.GenerateForAgent("unknown_agent", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown CLI agent")
}

func TestConfigGenerator_OpenCodeConfig(t *testing.T) {
	generator := NewConfigGenerator("localhost", 8100)
	config, err := generator.GenerateForAgent("opencode", nil)
	require.NoError(t, err)

	// Check required OpenCode fields
	assert.Contains(t, config.Content, "provider")
	assert.Contains(t, config.EnabledFeatures, "streaming")

	provider, ok := config.Content["provider"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "helixagent", provider["name"])

	options, ok := provider["options"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, options["baseUrl"], "localhost:8100")
	assert.Equal(t, "helixagent-debate", options["model"])
}

func TestConfigGenerator_ClineConfig(t *testing.T) {
	generator := NewConfigGenerator("localhost", 8100)
	config, err := generator.GenerateForAgent("cline", nil)
	require.NoError(t, err)

	// Check Cline-specific fields
	assert.Contains(t, config.Content, "thinkingBudgetTokens")
	assert.Contains(t, config.EnabledFeatures, "extended_thinking")
	assert.Contains(t, config.EnabledFeatures, "plan_act_modes")
	assert.Contains(t, config.EnabledFeatures, "distributed_locking")
}

func TestConfigGenerator_AmazonQConfig(t *testing.T) {
	generator := NewConfigGenerator("localhost", 8100)
	config, err := generator.GenerateForAgent("amazonq", nil)
	require.NoError(t, err)

	// Check compression is enabled (Amazon Q supports gzip)
	assert.Contains(t, config.EnabledFeatures, "gzip_compression")
	assert.Contains(t, config.EnabledFeatures, "mcp")
}

func TestConfigGenerator_OllamaCodeConfig(t *testing.T) {
	generator := NewConfigGenerator("localhost", 8100)
	config, err := generator.GenerateForAgent("ollamacode", nil)
	require.NoError(t, err)

	// Check sandboxing is enabled
	assert.Contains(t, config.EnabledFeatures, "sandboxing")
	assert.Contains(t, config.EnabledFeatures, "chat_compression")
	assert.Contains(t, config.EnabledFeatures, "checkpointing")
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestContainsStreamingType(t *testing.T) {
	types := []StreamingType{StreamingTypeSSE, StreamingTypeWebSocket}

	assert.True(t, containsStreamingType(types, StreamingTypeSSE))
	assert.True(t, containsStreamingType(types, StreamingTypeWebSocket))
	assert.False(t, containsStreamingType(types, StreamingTypeJSONL))
}

func TestContainsCompressionType(t *testing.T) {
	types := []CompressionType{CompressionGzip, CompressionBrotli}

	assert.True(t, containsCompressionType(types, CompressionGzip))
	assert.True(t, containsCompressionType(types, CompressionBrotli))
	assert.False(t, containsCompressionType(types, CompressionSemantic))
}

func TestContainsCachingType(t *testing.T) {
	types := []CachingType{CachingAnthropic, CachingPrompt}

	assert.True(t, containsCachingType(types, CachingAnthropic))
	assert.True(t, containsCachingType(types, CachingPrompt))
	assert.False(t, containsCachingType(types, CachingDashScope))
}

func TestContainsProtocol(t *testing.T) {
	protocols := []ProtocolType{ProtocolMCP, ProtocolOpenAI}

	assert.True(t, containsProtocol(protocols, ProtocolMCP))
	assert.True(t, containsProtocol(protocols, ProtocolOpenAI))
	assert.False(t, containsProtocol(protocols, ProtocolACP))
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkGetProviderBaseCapabilities(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetProviderBaseCapabilities("openai")
	}
}

func BenchmarkGetCLIAgentCapabilities(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetCLIAgentCapabilities("opencode")
	}
}

func BenchmarkDetector_Query(b *testing.B) {
	detector := NewDetector()
	ctx := context.Background()
	sseType := StreamingTypeSSE
	query := &CapabilityQuery{
		Provider:         "openai",
		RequireStreaming: &sseType,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.Query(ctx, query)
	}
}

func BenchmarkDetector_GetCapabilityMatrix(b *testing.B) {
	detector := NewDetector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = detector.GetCapabilityMatrix()
	}
}

func BenchmarkConfigGenerator_GenerateForAgent(b *testing.B) {
	generator := NewConfigGenerator("localhost", 8100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = generator.GenerateForAgent("opencode", nil)
	}
}
