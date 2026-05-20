package capabilities

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ConfigGenerator generates optimized CLI agent configurations
type ConfigGenerator struct {
	detector *Detector
	baseHost string
	basePort int
}

// NewConfigGenerator creates a new configuration generator
func NewConfigGenerator(host string, port int) *ConfigGenerator {
	return &ConfigGenerator{
		detector: NewDetector(),
		baseHost: host,
		basePort: port,
	}
}

// GeneratedConfig represents a generated CLI agent configuration
type GeneratedConfig struct {
	Agent            string                 `json:"agent"`
	GeneratedAt      time.Time              `json:"generated_at"`
	ConfigPath       string                 `json:"config_path"`
	Format           string                 `json:"format"`
	Content          map[string]interface{} `json:"content"`
	EnabledFeatures  []string               `json:"enabled_features"`
	Recommendations  []string               `json:"recommendations,omitempty"`
}

// GenerateForAgent generates an optimized configuration for a specific CLI agent
func (cg *ConfigGenerator) GenerateForAgent(agentName string, providerCaps *ProviderCapabilities) (*GeneratedConfig, error) {
	agentCaps := GetCLIAgentCapabilities(agentName)
	if agentCaps == nil {
		return nil, fmt.Errorf("unknown CLI agent: %s", agentName)
	}

	config := &GeneratedConfig{
		Agent:       agentName,
		GeneratedAt: time.Now(),
		ConfigPath:  agentCaps.ConfigPath,
		Format:      agentCaps.ConfigFormat,
		Content:     make(map[string]interface{}),
		EnabledFeatures: []string{},
	}

	// Generate agent-specific configuration
	switch agentName {
	case "opencode":
		cg.generateOpenCodeConfig(config, agentCaps, providerCaps)
	case "claudecode":
		cg.generateClaudeCodeConfig(config, agentCaps, providerCaps)
	case "kilocode":
		cg.generateKiloCodeConfig(config, agentCaps, providerCaps)
	case "cline":
		cg.generateClineConfig(config, agentCaps, providerCaps)
	case "crush":
		cg.generateCrushConfig(config, agentCaps, providerCaps)
	case "helixcode":
		cg.generateHelixCodeConfig(config, agentCaps, providerCaps)
	case "aider":
		cg.generateAiderConfig(config, agentCaps, providerCaps)
	case "plandex":
		cg.generatePlandexConfig(config, agentCaps, providerCaps)
	case "kiro":
		cg.generateKiroConfig(config, agentCaps, providerCaps)
	case "forge":
		cg.generateForgeConfig(config, agentCaps, providerCaps)
	case "amazonq":
		cg.generateAmazonQConfig(config, agentCaps, providerCaps)
	case "ollamacode":
		cg.generateOllamaCodeConfig(config, agentCaps, providerCaps)
	case "geminicli":
		cg.generateGeminiCLIConfig(config, agentCaps, providerCaps)
	case "qwencode":
		cg.generateQwenCodeConfig(config, agentCaps, providerCaps)
	case "gptengineer":
		cg.generateGPTEngineerConfig(config, agentCaps, providerCaps)
	case "deepseekcli":
		cg.generateDeepSeekCLIConfig(config, agentCaps, providerCaps)
	default:
		cg.generateGenericConfig(config, agentCaps, providerCaps)
	}

	// Add recommendations for HTTP/3
	if !agentCaps.Network.HTTP3Supported {
		config.Recommendations = append(config.Recommendations,
			tr("llmsverifier_capabilities_rec_http3_unsupported_agent"))
	}

	return config, nil
}

// generateOpenCodeConfig generates configuration for OpenCode CLI
func (cg *ConfigGenerator) generateOpenCodeConfig(config *GeneratedConfig, agent *CLIAgentCapabilities, provider *ProviderCapabilities) {
	baseURL := fmt.Sprintf("http://%s:%d", cg.baseHost, cg.basePort)

	content := map[string]interface{}{
		"$schema": "https://opencode.ai/config.json",
		"provider": map[string]interface{}{
			"name": "helixagent",
			"options": map[string]interface{}{
				"baseUrl": baseURL + "/v1",
				"model":   "helixagent-debate",
			},
		},
	}

	// Add MCP servers if supported
	mcpServers := map[string]interface{}{}
	for _, protocol := range agent.Protocols {
		if protocol == ProtocolMCP {
			mcpServers["helixagent-mcp"] = map[string]interface{}{
				"type":    "remote",
				"enabled": true,
				"timeout": 60000,
				"url":     baseURL + "/v1/mcp",
			}
			config.EnabledFeatures = append(config.EnabledFeatures, "mcp")
			break
		}
	}
	if len(mcpServers) > 0 {
		content["mcp"] = mcpServers
	}

	// Enable streaming if supported
	if agent.Streaming.Supported {
		config.EnabledFeatures = append(config.EnabledFeatures, "streaming")
	}

	// Enable caching if supported
	if agent.Caching.Supported {
		for _, ct := range agent.Caching.Types {
			config.EnabledFeatures = append(config.EnabledFeatures, fmt.Sprintf("caching:%s", ct))
		}
	}

	config.Content = content
}

// generateClaudeCodeConfig generates configuration for Claude Code CLI
func (cg *ConfigGenerator) generateClaudeCodeConfig(config *GeneratedConfig, agent *CLIAgentCapabilities, provider *ProviderCapabilities) {
	content := map[string]interface{}{
		"apiEndpoint": fmt.Sprintf("http://%s:%d/v1/messages", cg.baseHost, cg.basePort),
		"model":       "helixagent-debate",
	}

	// Enable streaming (always supported)
	if agent.Streaming.Supported {
		content["streaming"] = true
		config.EnabledFeatures = append(config.EnabledFeatures, "streaming")
	}

	// Enable OAuth if available
	if provider != nil && provider.Auth.OAuthSupported {
		content["useOAuth"] = true
		config.EnabledFeatures = append(config.EnabledFeatures, "oauth")
	}

	// Enable caching
	if agent.Caching.Supported {
		content["enableCaching"] = true
		config.EnabledFeatures = append(config.EnabledFeatures, "anthropic_caching")
	}

	// Enable extended thinking if provider supports it
	if provider != nil && provider.Model_.Reasoning {
		content["reasoningEffort"] = provider.Extended.ReasoningEffort
		config.EnabledFeatures = append(config.EnabledFeatures, "reasoning")
	}

	config.Content = content
}

// generateKiloCodeConfig generates configuration for KiloCode
func (cg *ConfigGenerator) generateKiloCodeConfig(config *GeneratedConfig, agent *CLIAgentCapabilities, provider *ProviderCapabilities) {
	baseURL := fmt.Sprintf("http://%s:%d", cg.baseHost, cg.basePort)

	content := map[string]interface{}{
		"provider": map[string]interface{}{
			"type":    "openai-compatible",
			"baseUrl": baseURL + "/v1",
			"model":   "helixagent-debate",
		},
		"streaming": map[string]interface{}{
			"enabled":    agent.Streaming.Supported,
			"chunkTypes": agent.Streaming.ChunkTypes,
		},
	}

	// Enable all 28 tools
	content["tools"] = map[string]interface{}{
		"enabled": agent.Tools,
	}
	config.EnabledFeatures = append(config.EnabledFeatures, fmt.Sprintf("tools:%d", agent.ToolCount))

	// Enable plan/act modes
	if agent.Extended.PlanActModes {
		content["modes"] = map[string]interface{}{
			"planAct": true,
		}
		config.EnabledFeatures = append(config.EnabledFeatures, "plan_act_modes")
	}

	// Enable checkpointing
	if agent.Extended.Checkpointing {
		content["checkpointing"] = map[string]interface{}{
			"enabled": true,
		}
		config.EnabledFeatures = append(config.EnabledFeatures, "checkpointing")
	}

	// Enable auto-approval
	if agent.Extended.AutoApproval {
		content["autoApproval"] = map[string]interface{}{
			"enabled": true,
		}
		config.EnabledFeatures = append(config.EnabledFeatures, "auto_approval")
	}

	config.Content = content
}

// generateClineConfig generates configuration for Cline
func (cg *ConfigGenerator) generateClineConfig(config *GeneratedConfig, agent *CLIAgentCapabilities, provider *ProviderCapabilities) {
	baseURL := fmt.Sprintf("http://%s:%d", cg.baseHost, cg.basePort)

	content := map[string]interface{}{
		"apiProvider": "openai-compatible",
		"apiBaseUrl":  baseURL + "/v1",
		"apiModelId":  "helixagent-debate",
	}

	// Enable extended thinking
	if agent.Extended.ThinkingBudget > 0 {
		content["thinkingBudgetTokens"] = agent.Extended.ThinkingBudget
		config.EnabledFeatures = append(config.EnabledFeatures, "extended_thinking")
	}

	// Enable reasoning effort
	if agent.Extended.ReasoningEffort != "" {
		content["reasoningEffort"] = agent.Extended.ReasoningEffort
		config.EnabledFeatures = append(config.EnabledFeatures, "reasoning")
	}

	// Enable caching
	if agent.Caching.Supported {
		content["enablePromptCaching"] = true
		config.EnabledFeatures = append(config.EnabledFeatures, "prompt_caching")
	}

	// Enable plan/act modes
	if agent.Extended.PlanActModes {
		content["planModeEnabled"] = true
		config.EnabledFeatures = append(config.EnabledFeatures, "plan_act_modes")
	}

	// Enable distributed locking
	if agent.Extended.DistributedLocking {
		content["distributedLocking"] = true
		config.EnabledFeatures = append(config.EnabledFeatures, "distributed_locking")
	}

	// Enable auto-approval settings
	if agent.Extended.AutoApproval {
		content["autoApprovalSettings"] = map[string]interface{}{
			"actions": map[string]interface{}{
				"read-files":  true,
				"browse-web":  true,
				"write-files": false,
			},
		}
		config.EnabledFeatures = append(config.EnabledFeatures, "auto_approval")
	}

	config.Content = content
}

// generateCrushConfig generates configuration for Crush CLI
func (cg *ConfigGenerator) generateCrushConfig(config *GeneratedConfig, agent *CLIAgentCapabilities, provider *ProviderCapabilities) {
	baseURL := fmt.Sprintf("http://%s:%d", cg.baseHost, cg.basePort)

	content := map[string]interface{}{
		"provider": map[string]interface{}{
			"baseUrl": baseURL + "/v1",
			"model":   "helixagent-debate",
		},
		"streaming": agent.Streaming.Supported,
	}

	if agent.Streaming.Supported {
		config.EnabledFeatures = append(config.EnabledFeatures, "streaming")
	}

	config.Content = content
}

// generateHelixCodeConfig generates configuration for HelixCode CLI
func (cg *ConfigGenerator) generateHelixCodeConfig(config *GeneratedConfig, agent *CLIAgentCapabilities, provider *ProviderCapabilities) {
	baseURL := fmt.Sprintf("http://%s:%d", cg.baseHost, cg.basePort)

	content := map[string]interface{}{
		"helixagent": map[string]interface{}{
			"endpoint": baseURL,
			"model":    "helixagent-debate",
		},
		"streaming": map[string]interface{}{
			"enabled": agent.Streaming.Supported,
			"type":    agent.Streaming.DefaultType,
		},
		"protocols": map[string]interface{}{
			"mcp": containsProtocolType(agent.Protocols, ProtocolMCP),
			"acp": containsProtocolType(agent.Protocols, ProtocolACP),
		},
		"tools": map[string]interface{}{
			"enabled": agent.Tools,
			"count":   agent.ToolCount,
		},
	}

	// Enable all extended features
	if agent.Extended.PlanActModes {
		content["planActModes"] = true
		config.EnabledFeatures = append(config.EnabledFeatures, "plan_act_modes")
	}

	if agent.Extended.AutoApproval {
		content["autoApproval"] = true
		config.EnabledFeatures = append(config.EnabledFeatures, "auto_approval")
	}

	if agent.Extended.AutoCommit {
		content["autoCommit"] = true
		config.EnabledFeatures = append(config.EnabledFeatures, "auto_commit")
	}

	if agent.Extended.AutoContinue {
		content["autoContinue"] = true
		config.EnabledFeatures = append(config.EnabledFeatures, "auto_continue")
	}

	if agent.Extended.Checkpointing {
		content["checkpointing"] = true
		config.EnabledFeatures = append(config.EnabledFeatures, "checkpointing")
	}

	// Enable caching
	if agent.Caching.Supported {
		cachingConfig := map[string]interface{}{
			"enabled": true,
			"types":   agent.Caching.Types,
		}
		content["caching"] = cachingConfig
		config.EnabledFeatures = append(config.EnabledFeatures, "caching")
	}

	config.Content = content
}

// generateAiderConfig generates configuration for Aider
func (cg *ConfigGenerator) generateAiderConfig(config *GeneratedConfig, agent *CLIAgentCapabilities, provider *ProviderCapabilities) {
	baseURL := fmt.Sprintf("http://%s:%d/v1", cg.baseHost, cg.basePort)

	// Aider uses YAML format
	content := map[string]interface{}{
		"openai-api-base": baseURL,
		"model":           "openai/helixagent-debate",
		"stream":          agent.Streaming.Supported,
	}

	// Enable auto-commit
	if agent.Extended.AutoCommit {
		content["auto-commits"] = true
		config.EnabledFeatures = append(config.EnabledFeatures, "auto_commit")
	}

	// Enable auto-continue
	if agent.Extended.AutoContinue {
		content["auto-lint"] = true
		content["auto-test"] = true
		config.EnabledFeatures = append(config.EnabledFeatures, "auto_continue")
	}

	// Enable caching
	if agent.Caching.Supported {
		content["cache-prompts"] = true
		config.EnabledFeatures = append(config.EnabledFeatures, "prompt_caching")
	}

	config.Content = content
}

// generatePlandexConfig generates configuration for Plandex
func (cg *ConfigGenerator) generatePlandexConfig(config *GeneratedConfig, agent *CLIAgentCapabilities, provider *ProviderCapabilities) {
	baseURL := fmt.Sprintf("http://%s:%d/v1", cg.baseHost, cg.basePort)

	content := map[string]interface{}{
		"baseUrl": baseURL,
		"model":   "helixagent-debate",
		"streaming": map[string]interface{}{
			"enabled":      agent.Streaming.Supported,
			"heartbeatSec": agent.Streaming.HeartbeatSec,
		},
	}

	// Enable branching
	if agent.Extended.Branching {
		content["branching"] = map[string]interface{}{
			"enabled": true,
		}
		config.EnabledFeatures = append(config.EnabledFeatures, "branching")
	}

	// Enable auto-commit
	if agent.Extended.AutoCommit {
		content["autoCommit"] = true
		config.EnabledFeatures = append(config.EnabledFeatures, "auto_commit")
	}

	// Enable auto-continue
	if agent.Extended.AutoContinue {
		content["autoContinue"] = true
		config.EnabledFeatures = append(config.EnabledFeatures, "auto_continue")
	}

	// Enable caching
	if agent.Caching.Supported {
		content["caching"] = map[string]interface{}{
			"enabled": true,
			"types":   agent.Caching.Types,
		}
		config.EnabledFeatures = append(config.EnabledFeatures, "caching")
	}

	// Enable compression
	if agent.Compression.Supported {
		content["compression"] = map[string]interface{}{
			"enabled": true,
			"type":    agent.Compression.DefaultType,
		}
		config.EnabledFeatures = append(config.EnabledFeatures, "compression")
	}

	config.Content = content
}

// generateKiroConfig generates configuration for Kiro
func (cg *ConfigGenerator) generateKiroConfig(config *GeneratedConfig, agent *CLIAgentCapabilities, provider *ProviderCapabilities) {
	content := map[string]interface{}{
		"ai_provider": "opencode",
		"helixagent_endpoint": fmt.Sprintf("http://%s:%d/v1", cg.baseHost, cg.basePort),
	}

	// Enable plan/act modes (3-phase methodology)
	if agent.Extended.PlanActModes {
		content["three_phase_enabled"] = true
		config.EnabledFeatures = append(config.EnabledFeatures, "three_phase_methodology")
	}

	// Enable compression
	if agent.Compression.Supported {
		content["compression"] = true
		config.EnabledFeatures = append(config.EnabledFeatures, "compression")
	}

	config.Content = content
}

// generateForgeConfig generates configuration for Forge
func (cg *ConfigGenerator) generateForgeConfig(config *GeneratedConfig, agent *CLIAgentCapabilities, provider *ProviderCapabilities) {
	baseURL := fmt.Sprintf("http://%s:%d/v1", cg.baseHost, cg.basePort)

	content := map[string]interface{}{
		"provider": map[string]interface{}{
			"type":    "openai",
			"baseUrl": baseURL,
			"model":   "helixagent-debate",
		},
		"streaming": map[string]interface{}{
			"enabled": agent.Streaming.Supported,
			"type":    string(agent.Streaming.DefaultType),
		},
	}

	// Enable semantic compression
	if agent.Compression.Supported && containsCompressionType(agent.Compression.Types, CompressionSemantic) {
		content["compression"] = map[string]interface{}{
			"enabled": true,
			"type":    "semantic",
		}
		config.EnabledFeatures = append(config.EnabledFeatures, "semantic_compression")
	}

	// Enable auto-approval
	if agent.Extended.AutoApproval {
		content["autoApproval"] = true
		config.EnabledFeatures = append(config.EnabledFeatures, "auto_approval")
	}

	config.Content = content
}

// generateAmazonQConfig generates configuration for Amazon Q
func (cg *ConfigGenerator) generateAmazonQConfig(config *GeneratedConfig, agent *CLIAgentCapabilities, provider *ProviderCapabilities) {
	baseURL := fmt.Sprintf("http://%s:%d/v1", cg.baseHost, cg.basePort)

	content := map[string]interface{}{
		"endpoint": baseURL,
		"model":    "helixagent-debate",
		"streaming": map[string]interface{}{
			"enabled": agent.Streaming.Supported,
			"type":    string(agent.Streaming.DefaultType),
		},
	}

	// Enable gzip compression (Amazon Q supports it)
	if agent.Compression.Supported && containsCompressionType(agent.Compression.Types, CompressionGzip) {
		content["compression"] = map[string]interface{}{
			"enabled":  true,
			"type":     "gzip",
			"request":  agent.Compression.RequestComp,
			"response": agent.Compression.ResponseComp,
		}
		config.EnabledFeatures = append(config.EnabledFeatures, "gzip_compression")
	}

	// Enable MCP
	if containsProtocolType(agent.Protocols, ProtocolMCP) {
		content["mcp"] = map[string]interface{}{
			"enabled":  true,
			"endpoint": baseURL + "/mcp",
		}
		config.EnabledFeatures = append(config.EnabledFeatures, "mcp")
	}

	// Enable auto-approval
	if agent.Extended.AutoApproval {
		content["autoApproval"] = true
		config.EnabledFeatures = append(config.EnabledFeatures, "auto_approval")
	}

	config.Content = content
}

// generateOllamaCodeConfig generates configuration for Ollama Code
func (cg *ConfigGenerator) generateOllamaCodeConfig(config *GeneratedConfig, agent *CLIAgentCapabilities, provider *ProviderCapabilities) {
	content := map[string]interface{}{
		"ollama": map[string]interface{}{
			"baseUrl": fmt.Sprintf("http://%s:%d/v1", cg.baseHost, cg.basePort),
			"model":   "helixagent-debate",
		},
		"streaming": map[string]interface{}{
			"enabled": agent.Streaming.Supported,
		},
	}

	// Enable chat compression
	if agent.Compression.Supported && containsCompressionType(agent.Compression.Types, CompressionChat) {
		content["compression"] = map[string]interface{}{
			"enabled": true,
			"type":    "chat",
		}
		config.EnabledFeatures = append(config.EnabledFeatures, "chat_compression")
	}

	// Enable sandboxing
	if agent.Extended.Sandboxing {
		content["sandbox"] = map[string]interface{}{
			"enabled": true,
			"types":   agent.Extended.SandboxTypes,
		}
		config.EnabledFeatures = append(config.EnabledFeatures, "sandboxing")
	}

	// Enable checkpointing
	if agent.Extended.Checkpointing {
		content["checkpointing"] = map[string]interface{}{
			"enabled": true,
		}
		config.EnabledFeatures = append(config.EnabledFeatures, "checkpointing")
	}

	// Enable auto-approval
	if agent.Extended.AutoApproval {
		content["autoApproval"] = true
		config.EnabledFeatures = append(config.EnabledFeatures, "auto_approval")
	}

	config.Content = content
}

// generateGeminiCLIConfig generates configuration for Gemini CLI
func (cg *ConfigGenerator) generateGeminiCLIConfig(config *GeneratedConfig, agent *CLIAgentCapabilities, provider *ProviderCapabilities) {
	content := map[string]interface{}{
		"provider": map[string]interface{}{
			"type":    "openai-compatible",
			"baseUrl": fmt.Sprintf("http://%s:%d/v1", cg.baseHost, cg.basePort),
			"model":   "helixagent-debate",
		},
		"streaming": map[string]interface{}{
			"enabled": agent.Streaming.Supported,
			"type":    string(agent.Streaming.DefaultType),
		},
	}

	// Enable chat compression
	if agent.Compression.Supported {
		content["compression"] = map[string]interface{}{
			"enabled": true,
			"type":    string(agent.Compression.DefaultType),
		}
		config.EnabledFeatures = append(config.EnabledFeatures, "chat_compression")
	}

	// Enable extended thinking
	if agent.Extended.ThinkingBudget > 0 {
		content["thinking"] = map[string]interface{}{
			"enabled":         true,
			"budget":          agent.Extended.ThinkingBudget,
			"includeThoughts": agent.Extended.ThoughtsIncluded,
		}
		config.EnabledFeatures = append(config.EnabledFeatures, "extended_thinking")
	}

	config.Content = content
}

// generateQwenCodeConfig generates configuration for Qwen Code
func (cg *ConfigGenerator) generateQwenCodeConfig(config *GeneratedConfig, agent *CLIAgentCapabilities, provider *ProviderCapabilities) {
	content := map[string]interface{}{
		"endpoint": fmt.Sprintf("http://%s:%d/v1", cg.baseHost, cg.basePort),
		"model":    "helixagent-debate",
		"streaming": map[string]interface{}{
			"enabled": agent.Streaming.Supported,
		},
	}

	// Enable DashScope caching
	if agent.Caching.Supported && containsCachingType(agent.Caching.Types, CachingDashScope) {
		content["caching"] = map[string]interface{}{
			"enabled": true,
			"type":    "dashscope",
		}
		config.EnabledFeatures = append(config.EnabledFeatures, "dashscope_caching")
	}

	// Enable auto-approval
	if agent.Extended.AutoApproval {
		content["autoApproval"] = true
		config.EnabledFeatures = append(config.EnabledFeatures, "auto_approval")
	}

	config.Content = content
}

// generateGPTEngineerConfig generates configuration for GPT Engineer
func (cg *ConfigGenerator) generateGPTEngineerConfig(config *GeneratedConfig, agent *CLIAgentCapabilities, provider *ProviderCapabilities) {
	content := map[string]interface{}{
		"run": map[string]interface{}{
			"build": "make build",
			"test":  "make test",
		},
		"paths": map[string]interface{}{
			"base": ".",
		},
	}

	// Enable LLMOps caching
	if agent.Caching.Supported {
		content["caching"] = map[string]interface{}{
			"enabled": true,
			"type":    "sqlite",
		}
		config.EnabledFeatures = append(config.EnabledFeatures, "llmops_caching")
	}

	// Enable auto-continue
	if agent.Extended.AutoContinue {
		content["auto_continue"] = true
		config.EnabledFeatures = append(config.EnabledFeatures, "auto_continue")
	}

	config.Content = content
}

// generateDeepSeekCLIConfig generates configuration for DeepSeek CLI
func (cg *ConfigGenerator) generateDeepSeekCLIConfig(config *GeneratedConfig, agent *CLIAgentCapabilities, provider *ProviderCapabilities) {
	// DeepSeek CLI uses environment variables
	content := map[string]interface{}{
		"DEEPSEEK_USE_LOCAL": "false",
		"DEEPSEEK_API_KEY":   "${DEEPSEEK_API_KEY}",
		"DEEPSEEK_MODEL":     "deepseek-chat",
	}

	// Note: DeepSeek CLI does not support streaming
	config.Recommendations = append(config.Recommendations,
		tr("llmsverifier_capabilities_rec_deepseek_no_streaming"))

	config.Content = content
}

// generateGenericConfig generates a generic configuration
func (cg *ConfigGenerator) generateGenericConfig(config *GeneratedConfig, agent *CLIAgentCapabilities, provider *ProviderCapabilities) {
	baseURL := fmt.Sprintf("http://%s:%d/v1", cg.baseHost, cg.basePort)

	content := map[string]interface{}{
		"endpoint": baseURL,
		"model":    "helixagent-debate",
	}

	// Add streaming if supported
	if agent.Streaming.Supported {
		content["streaming"] = true
		config.EnabledFeatures = append(config.EnabledFeatures, "streaming")
	}

	config.Content = content
}

// Helper function
func containsProtocolType(protocols []ProtocolType, p ProtocolType) bool {
	for _, proto := range protocols {
		if proto == p {
			return true
		}
	}
	return false
}

// SaveConfig saves a generated configuration to disk
func (cg *ConfigGenerator) SaveConfig(config *GeneratedConfig, outputPath string) error {
	var data []byte
	var err error

	switch config.Format {
	case "json":
		data, err = json.MarshalIndent(config.Content, "", "  ")
	case "yaml":
		// For YAML we output JSON but note it should be converted
		data, err = json.MarshalIndent(config.Content, "", "  ")
	default:
		data, err = json.MarshalIndent(config.Content, "", "  ")
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(outputPath, data, 0644)
}

// GenerateAllConfigs generates configurations for all CLI agents
func (cg *ConfigGenerator) GenerateAllConfigs(outputDir string, providerCaps *ProviderCapabilities) ([]*GeneratedConfig, error) {
	agents := GetAllCLIAgents()
	configs := make([]*GeneratedConfig, 0, len(agents))

	for _, agent := range agents {
		config, err := cg.GenerateForAgent(agent, providerCaps)
		if err != nil {
			continue // Skip agents that fail
		}

		// Save the config
		outputPath := filepath.Join(outputDir, fmt.Sprintf("%s-config.json", agent))
		if err := cg.SaveConfig(config, outputPath); err != nil {
			config.Recommendations = append(config.Recommendations,
				fmt.Sprintf(tr("llmsverifier_capabilities_rec_save_config_failed"), err))
		}

		configs = append(configs, config)
	}

	return configs, nil
}
