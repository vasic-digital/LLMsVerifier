# LLMsVerifier Capability Detection System

The LLMsVerifier Capability Detection System provides dynamic capability detection for LLM providers and CLI coding agents. This is the single source of truth for querying LLM and CLI agent capabilities.

## Overview

The capability detection system was built by analyzing the source code of 18+ CLI coding agents to determine their actual capabilities. Key findings:

- **No CLI agent supports HTTP/3 or QUIC** - All agents use HTTP/1.1 or HTTP/2
- **Streaming is universal** but uses different mechanisms (SSE, AsyncGenerator, JSONL)
- **Compression is limited** - Only Amazon Q has gzip, Forge has semantic compression
- **Caching varies widely** - Anthropic cache_control, DashScope, prompt caching, LLMOps

## HelixAgent Multi-Streaming Support

**IMPORTANT**: HelixAgent supports ALL streaming mechanisms to ensure compatibility with every CLI agent:

| Streaming Type | HelixAgent Support | Endpoint |
|---------------|-------------------|----------|
| **SSE** | ✅ Full Support | `/v1/chat/completions` (default) |
| **WebSocket** | ✅ Full Support | `/v1/ws/chat` |
| **AsyncGenerator** | ✅ Full Support | Via SSE chunked responses |
| **JSONL** | ✅ Full Support | `/v1/chat/completions?format=jsonl` |
| **EventStream** | ✅ Full Support | Via SSE with AWS format |
| **Stdout** | ✅ Full Support | For CLI piping |

This ensures that regardless of which CLI agent connects to HelixAgent, the appropriate streaming format is used automatically based on the client's capabilities.

## Package Structure

```
LLMsVerifier/llm-verifier/capabilities/
├── types.go           # Core capability type definitions
├── registry.go        # Provider and CLI agent capability registry
├── detector.go        # Dynamic capability detection service
├── config_generator.go # Optimized CLI agent configuration generator
└── capabilities_test.go # Comprehensive unit tests
```

## Capability Types

### Streaming Types

| Type | Description | Agents Using |
|------|-------------|--------------|
| `sse` | Server-Sent Events | OpenCode, ClaudeCode, Plandex, Crush |
| `websocket` | WebSocket | ClaudeCode |
| `async_generator` | AsyncGenerator/yield | KiloCode, Cline, OllamaCode |
| `jsonl` | JSON Lines streaming | GeminiCLI |
| `mpsc_stream` | Rust multi-producer channel | Forge |
| `event_stream` | AWS EventStream | Amazon Q |
| `stdout` | Standard output streaming | Aider, GPT Engineer |
| `none` | No streaming | DeepSeek CLI |

### Compression Types

| Type | Description | Agents Using |
|------|-------------|--------------|
| `gzip` | Gzip compression | Amazon Q, Plandex, Kiro |
| `brotli` | Brotli compression | None discovered |
| `semantic` | Semantic context compression | Forge |
| `chat` | Chat history compression | OllamaCode, GeminiCLI |

### Caching Types

| Type | Description | Providers/Agents |
|------|-------------|------------------|
| `anthropic_cache_control` | Anthropic cache_control header | Claude, OpenCode, Cline, Plandex |
| `dashscope_cache` | DashScope X-DashScope-CacheControl | Qwen, QwenCode |
| `prompt_caching` | Generic prompt caching | Most agents |
| `semantic_caching` | Semantic similarity caching | HelixCode |
| `llmops_cache` | LangChain/SQLite cache | GPT Engineer |

### Protocol Types

| Type | Description |
|------|-------------|
| `mcp` | Model Context Protocol |
| `acp` | Agent Communication Protocol |
| `lsp` | Language Server Protocol |
| `grpc` | gRPC |
| `openai` | OpenAI-compatible API |
| `anthropic` | Anthropic API |
| `ollama` | Ollama local API |

### Authentication Types

| Type | Description |
|------|-------------|
| `api_key` | API key in query/header |
| `bearer` | Bearer token |
| `oauth2` | OAuth2 flow |
| `none` | No authentication |
| `aws_sig_v4` | AWS Signature V4 |

## CLI Agent Capabilities Matrix

| Agent | Language | Streaming | HTTP/2 | HTTP/3 | Compression | Caching | MCP | Providers |
|-------|----------|-----------|--------|--------|-------------|---------|-----|-----------|
| OpenCode | Go | SSE | Yes | **No** | No | Anthropic | Yes | 15+ |
| ClaudeCode | TS | SSE/WS | Yes | **No** | No | Anthropic | Yes | 1 |
| KiloCode | TS | AsyncGen | Yes | **No** | No | Prompt | Yes | 43+ |
| Cline | TS | AsyncGen | Yes | **No** | No | Anthropic | No | 41+ |
| Aider | Python | stdout | Yes | **No** | No | Prompt | No | 10+ |
| Amazon Q | Rust | EventStream | Yes | **No** | Gzip | No | Yes | 2 |
| Forge | Rust | MpscStream | Yes | **No** | Semantic | No | No | 8 |
| Plandex | Go | SSE | Yes | **No** | Gzip | Anthropic | No | 10+ |
| Kiro | Python | HTTP | Yes | **No** | Gzip | No | No | 12 |
| OllamaCode | TS | AsyncGen | No | **No** | Chat | No | No | 2 |
| GeminiCLI | TS | JSONL | Yes | **No** | Chat | No | No | 1 |
| QwenCode | TS | SSE | Yes | **No** | No | DashScope | No | 1 |
| HelixCode | Go | SSE | Yes | **No** | No | Prompt/Semantic | Yes | 18+ |

**Key Finding: No CLI agent supports HTTP/3 or QUIC.** This was verified by analyzing source code of all agents.

## Usage

### Query Provider Capabilities

```go
import "llm-verifier/capabilities"

detector := capabilities.NewDetector()
ctx := context.Background()

// Query if OpenAI supports SSE streaming
sseType := capabilities.StreamingTypeSSE
query := &capabilities.CapabilityQuery{
    Provider:         "openai",
    RequireStreaming: &sseType,
}

result, err := detector.Query(ctx, query)
if result.Matches {
    fmt.Println("OpenAI supports SSE streaming")
}

// Query for HTTP/3 (will fail for all providers)
query := &capabilities.CapabilityQuery{
    Provider:     "openai",
    RequireHTTP3: true,
}
result, _ := detector.Query(ctx, query)
// result.Matches == false
// result.Recommendations contains "HTTP/3 is not currently supported"
```

### Get Capability Matrix

```go
detector := capabilities.NewDetector()
matrix := detector.GetCapabilityMatrix()

// Access indexed lookups
sseProviders := matrix.ByStreaming[capabilities.StreamingTypeSSE]
mcpAgents := matrix.ByProtocol[capabilities.ProtocolMCP]
oauthProviders := matrix.ByAuth[capabilities.AuthOAuth2]
```

### Generate CLI Agent Configuration

```go
generator := capabilities.NewConfigGenerator("localhost", 7061)

// Generate config for OpenCode
config, err := generator.GenerateForAgent("opencode", nil)
fmt.Println("Enabled features:", config.EnabledFeatures)
// Output: [streaming, mcp, anthropic_caching]

// Generate for all agents
configs, _ := generator.GenerateAllConfigs("/output/dir", nil)
```

### Lookup Capabilities

```go
// Get providers with specific capability
streamingProviders := capabilities.GetProvidersWithCapability("streaming", nil)
oauthProviders := capabilities.GetProvidersWithCapability("oauth", nil)
http3Providers := capabilities.GetProvidersWithCapability("http3", nil)
// http3Providers is empty - no provider supports HTTP/3

// Get CLI agents with specific capability
mcpAgents := capabilities.GetCLIAgentsWithCapability("mcp")
checkpointAgents := capabilities.GetCLIAgentsWithCapability("checkpointing")
```

## Extended Features

### Plan/Act Modes
Agents that support separate planning and acting phases:
- KiloCode, Cline, Kiro, HelixCode

### Checkpointing
Agents with Git-based file checkpointing:
- KiloCode, Cline, OllamaCode, HelixCode

### Sandboxing
Agents with container sandbox support:
- OllamaCode (Docker, Podman, macOS Seatbelt)

### Branching
Agents with plan branching:
- Plandex (multiple branches per plan)

### Extended Thinking
Agents with thinking/reasoning support:
- Cline (10,000 token budget)
- GeminiCLI (24,000 token budget)

## Configuration Generation

The ConfigGenerator produces optimized configurations for each CLI agent with all supported features enabled:

### OpenCode Configuration
```json
{
  "$schema": "https://opencode.ai/config.json",
  "provider": {
    "name": "helixagent",
    "options": {
      "baseUrl": "http://localhost:7061/v1",
      "model": "helixagent-debate"
    }
  },
  "mcp": {
    "helixagent-mcp": {
      "type": "remote",
      "enabled": true,
      "timeout": 60000,
      "url": "http://localhost:7061/v1/mcp"
    }
  }
}
```

### Cline Configuration
```json
{
  "apiProvider": "openai-compatible",
  "apiBaseUrl": "http://localhost:7061/v1",
  "apiModelId": "helixagent-debate",
  "thinkingBudgetTokens": 10000,
  "planModeEnabled": true,
  "distributedLocking": true,
  "enablePromptCaching": true
}
```

### Amazon Q Configuration (with gzip)
```json
{
  "endpoint": "http://localhost:7061/v1",
  "model": "helixagent-debate",
  "compression": {
    "enabled": true,
    "type": "gzip",
    "request": true,
    "response": true
  },
  "mcp": {
    "enabled": true,
    "endpoint": "http://localhost:7061/v1/mcp"
  }
}
```

## Challenge Validation

Run the capability detection challenge to validate the system:

```bash
./challenges/scripts/capability_detection_challenge.sh
```

The challenge validates:
1. Package structure (types, registry, detector, generator)
2. All streaming types covered
3. HTTP/3 correctly marked as unsupported
4. 18+ CLI agents registered
5. 10+ LLM providers registered
6. All unit tests pass
7. Configuration generation works

## API Reference

### Types

```go
type ProviderCapabilities struct {
    Provider    string
    Model       string
    Verified    bool
    VerifiedAt  time.Time
    Streaming   StreamingCapability
    Network     NetworkCapability
    Compression CompressionCapability
    Caching     CachingCapability
    Protocols   []ProtocolType
    Auth        AuthCapability
    Model_      ModelCapability
    Extended    ExtendedCapabilities
    Custom      map[string]interface{}
}

type CLIAgentCapabilities struct {
    Name          string
    Language      string
    ConfigFormat  string
    ConfigPath    string
    Streaming     StreamingCapability
    Network       NetworkCapability
    Compression   CompressionCapability
    Caching       CachingCapability
    Protocols     []ProtocolType
    ProviderCount int
    Providers     []string
    ToolCount     int
    Tools         []string
    Extended      ExtendedCapabilities
    ConfigOptions map[string]ConfigOption
}
```

### Functions

```go
// Registry functions
func GetProviderBaseCapabilities(provider string) *ProviderCapabilities
func GetCLIAgentCapabilities(agent string) *CLIAgentCapabilities
func GetAllProviders() []string
func GetAllCLIAgents() []string
func GetProvidersWithCapability(capName string, capValue interface{}) []string
func GetCLIAgentsWithCapability(capName string) []string

// Detector methods
func NewDetector() *Detector
func (d *Detector) DetectProviderCapabilities(ctx context.Context, provider, apiKey string) (*ProviderCapabilities, error)
func (d *Detector) Query(ctx context.Context, query *CapabilityQuery) (*CapabilityQueryResult, error)
func (d *Detector) GetCapabilityMatrix() *CapabilityMatrix

// ConfigGenerator methods
func NewConfigGenerator(host string, port int) *ConfigGenerator
func (cg *ConfigGenerator) GenerateForAgent(agentName string, providerCaps *ProviderCapabilities) (*GeneratedConfig, error)
func (cg *ConfigGenerator) GenerateAllConfigs(outputDir string, providerCaps *ProviderCapabilities) ([]*GeneratedConfig, error)
func (cg *ConfigGenerator) SaveConfig(config *GeneratedConfig, outputPath string) error
```
