package cliagents

import (
	"context"
	"encoding/json"
	"testing"
)

func TestNewUnifiedGenerator(t *testing.T) {
	// Test with nil config
	gen := NewUnifiedGenerator(nil)
	if gen == nil {
		t.Fatal("NewUnifiedGenerator returned nil")
	}
	if gen.config == nil {
		t.Error("Generator config is nil")
	}

	// Test with custom config
	config := &GeneratorConfig{
		HelixAgentHost: "custom-host",
		HelixAgentPort: 8080,
	}
	gen = NewUnifiedGenerator(config)
	if gen.config.HelixAgentHost != "custom-host" {
		t.Errorf("Expected host 'custom-host', got '%s'", gen.config.HelixAgentHost)
	}
}

func TestDefaultGeneratorConfig(t *testing.T) {
	config := DefaultGeneratorConfig()
	if config == nil {
		t.Fatal("DefaultGeneratorConfig returned nil")
	}
	if config.HelixAgentHost != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", config.HelixAgentHost)
	}
	if config.HelixAgentPort != 7061 {
		t.Errorf("Expected port 7061, got %d", config.HelixAgentPort)
	}
	if !config.IncludeScores {
		t.Error("Expected IncludeScores to be true")
	}
}

func TestDefaultMCPServers(t *testing.T) {
	servers := DefaultMCPServers()
	if len(servers) == 0 {
		t.Fatal("DefaultMCPServers returned empty slice")
	}

	// Check for HelixAgent MCP endpoints
	helixAgentCount := 0
	for _, server := range servers {
		if server.Type == "remote" {
			helixAgentCount++
		}
	}
	if helixAgentCount < 6 {
		t.Errorf("Expected at least 6 HelixAgent MCP endpoints, got %d", helixAgentCount)
	}

	// DefaultMCPServers returns only HelixAgent remote endpoints (no local npx servers)
	// Local MCP servers are available via ContainerizedMCPServers() instead
}

func TestListSupportedAgents(t *testing.T) {
	gen := NewUnifiedGenerator(nil)
	agents := gen.ListSupportedAgents()

	// Should have 48 agents registered
	if len(agents) != 48 {
		t.Errorf("Expected 48 supported agents, got %d", len(agents))
	}

	// Check that key agents are registered
	expectedAgents := []AgentType{
		AgentOpenCode, AgentCrush, AgentKiloCode, AgentHelixCode,
		AgentAider, AgentContinue, AgentClaudeCode, AgentCline,
		AgentKiro, AgentCodenameGoose, AgentCodex, AgentOpenHands,
	}
	agentMap := make(map[AgentType]bool)
	for _, a := range agents {
		agentMap[a] = true
	}
	for _, expected := range expectedAgents {
		if !agentMap[expected] {
			t.Errorf("Expected agent %s not found", expected)
		}
	}
}

func TestGenerateOpenCode(t *testing.T) {
	gen := NewUnifiedGenerator(nil)
	ctx := context.Background()

	result, err := gen.Generate(ctx, AgentOpenCode)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}
	if result.Config == nil {
		t.Fatal("Config is nil")
	}

	// Validate the generated config
	openCodeConfig, ok := result.Config.(*OpenCodeConfig)
	if !ok {
		t.Fatal("Config is not *OpenCodeConfig")
	}

	if openCodeConfig.Provider.Options.BaseURL == "" {
		t.Error("Provider BaseURL is empty")
	}
	if len(openCodeConfig.Provider.Options.Models) == 0 {
		t.Error("No models configured")
	}
	if len(openCodeConfig.MCPServers) == 0 {
		t.Error("No MCP servers configured")
	}
	if len(openCodeConfig.Agent) == 0 {
		t.Error("No agents configured")
	}

	// Check validation result
	if result.ValidationResult == nil {
		t.Error("ValidationResult is nil")
	} else if !result.ValidationResult.Valid {
		t.Errorf("Config validation failed: %v", result.ValidationResult.Errors)
	}
}

func TestGenerateCrush(t *testing.T) {
	gen := NewUnifiedGenerator(nil)
	ctx := context.Background()

	result, err := gen.Generate(ctx, AgentCrush)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}
	if result.Config == nil {
		t.Fatal("Config is nil")
	}

	crushConfig, ok := result.Config.(*CrushConfig)
	if !ok {
		t.Fatal("Config is not *CrushConfig")
	}

	if crushConfig.Provider.BaseURL == "" {
		t.Error("Provider BaseURL is empty")
	}
}

func TestGenerateKiloCode(t *testing.T) {
	gen := NewUnifiedGenerator(nil)
	ctx := context.Background()

	result, err := gen.Generate(ctx, AgentKiloCode)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}
	if result.Config == nil {
		t.Fatal("Config is nil")
	}

	kiloConfig, ok := result.Config.(*KiloCodeConfig)
	if !ok {
		t.Fatal("Config is not *KiloCodeConfig")
	}

	if kiloConfig.Provider.BaseURL == "" {
		t.Error("Provider BaseURL is empty")
	}
}

func TestGenerateHelixCode(t *testing.T) {
	gen := NewUnifiedGenerator(nil)
	ctx := context.Background()

	result, err := gen.Generate(ctx, AgentHelixCode)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}
	if result.Config == nil {
		t.Fatal("Config is nil")
	}

	helixConfig, ok := result.Config.(*HelixCodeConfig)
	if !ok {
		t.Fatal("Config is not *HelixCodeConfig")
	}

	if helixConfig.Provider.BaseURL == "" {
		t.Error("Provider BaseURL is empty")
	}
	if len(helixConfig.Agents) == 0 {
		t.Error("No agents configured")
	}
}

func TestGenerateAll(t *testing.T) {
	gen := NewUnifiedGenerator(nil)
	ctx := context.Background()

	results, err := gen.GenerateAll(ctx)
	if err != nil {
		t.Fatalf("GenerateAll failed: %v", err)
	}

	if len(results) != 48 {
		t.Errorf("Expected 48 results, got %d", len(results))
	}

	// Check all succeeded
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}
	if successCount != 48 {
		t.Errorf("Expected all 48 to succeed, got %d", successCount)
	}
}

func TestGenerateUnsupportedAgent(t *testing.T) {
	gen := NewUnifiedGenerator(nil)
	ctx := context.Background()

	_, err := gen.Generate(ctx, AgentType("unknown-agent"))
	if err == nil {
		t.Error("Expected error for unsupported agent")
	}
}

func TestGetSchema(t *testing.T) {
	gen := NewUnifiedGenerator(nil)

	schema, err := gen.GetSchema(AgentOpenCode)
	if err != nil {
		t.Fatalf("GetSchema failed: %v", err)
	}

	if schema == nil {
		t.Fatal("Schema is nil")
	}
	if schema.AgentType != AgentOpenCode {
		t.Errorf("Expected AgentType 'opencode', got '%s'", schema.AgentType)
	}
	if schema.ConfigFileName == "" {
		t.Error("ConfigFileName is empty")
	}
	if len(schema.RequiredFields) == 0 {
		t.Error("RequiredFields is empty")
	}
}

func TestGetAllSchemas(t *testing.T) {
	gen := NewUnifiedGenerator(nil)
	schemas := gen.GetAllSchemas()

	if len(schemas) != 48 {
		t.Errorf("Expected 48 schemas, got %d", len(schemas))
	}

	// Check that each schema has required fields
	for agentType, schema := range schemas {
		if schema.ConfigFileName == "" {
			t.Errorf("Agent %s has empty ConfigFileName", agentType)
		}
		if schema.Description == "" {
			t.Errorf("Agent %s has empty Description", agentType)
		}
	}
}

func TestValidateOpenCodeConfig(t *testing.T) {
	gen := NewUnifiedGenerator(nil)

	// Valid config
	validConfig := &OpenCodeConfig{
		Provider: OpenCodeProviderConfig{
			Options: OpenCodeProviderOptions{
				BaseURL: "http://localhost:7061/v1",
			},
		},
		MCPServers: map[string]OpenCodeMCPServer{
			"test": {
				Type: "sse",
				URL:  "http://localhost:7061/mcp",
			},
		},
	}

	result, err := gen.Validate(AgentOpenCode, validConfig)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if !result.Valid {
		t.Errorf("Expected valid config, got errors: %v", result.Errors)
	}

	// Invalid config - missing MCP URL for remote type
	invalidConfig := &OpenCodeConfig{
		Provider: OpenCodeProviderConfig{
			Options: OpenCodeProviderOptions{
				BaseURL: "http://localhost:7061/v1",
			},
		},
		MCPServers: map[string]OpenCodeMCPServer{
			"test": {
				Type: "sse",
				// Missing URL
			},
		},
	}

	result, err = gen.Validate(AgentOpenCode, invalidConfig)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if result.Valid {
		t.Error("Expected invalid config")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected validation errors")
	}
}

func TestValidateFromMap(t *testing.T) {
	gen := NewUnifiedGenerator(nil)

	// Valid config as map
	validConfig := map[string]interface{}{
		"provider": map[string]interface{}{
			"options": map[string]interface{}{
				"baseURL": "http://localhost:7061/v1",
			},
		},
		"mcp": map[string]interface{}{
			"test": map[string]interface{}{
				"type": "remote",
				"url":  "http://localhost:7061/mcp",
			},
		},
	}

	result, err := gen.Validate(AgentOpenCode, validConfig)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if !result.Valid {
		t.Errorf("Expected valid config, got errors: %v", result.Errors)
	}

	// Invalid config - missing provider
	invalidConfig := map[string]interface{}{
		"mcp": map[string]interface{}{},
	}

	result, err = gen.Validate(AgentOpenCode, invalidConfig)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if result.Valid {
		t.Error("Expected invalid config")
	}
}

func TestConfigJSONSerialization(t *testing.T) {
	gen := NewUnifiedGenerator(nil)
	ctx := context.Background()

	result, err := gen.Generate(ctx, AgentOpenCode)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(result.Config, "", "  ")
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("JSON output is empty")
	}

	// Parse back
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	// Verify key fields exist
	if _, ok := parsed["provider"]; !ok {
		t.Error("provider field missing in JSON")
	}
	if _, ok := parsed["mcpServers"]; !ok {
		t.Error("mcpServers field missing in JSON")
	}
}

func TestGenericAgentGenerator(t *testing.T) {
	// Test all generic agents generate successfully
	genericAgents := []AgentType{
		// Original 18 (excluding 4 custom: OpenCode, Crush, KiloCode, HelixCode)
		AgentKiro, AgentAider, AgentClaudeCode, AgentCline, AgentCodenameGoose,
		AgentDeepSeekCLI, AgentForge, AgentGeminiCLI, AgentGPTEngineer,
		AgentMistralCode, AgentOllamaCode, AgentPlandex, AgentQwenCode, AgentAmazonQ,
		// New 30
		AgentAgentDeck, AgentBridle, AgentCheshireCat, AgentClaudePlugins,
		AgentClaudeSquad, AgentCodai, AgentCodex, AgentCodexSkills, AgentConduit,
		AgentContinue, AgentEmdash, AgentFauxPilot, AgentGetShitDone,
		AgentGitHubCopilotCLI, AgentGitHubSpecKit, AgentGitMCP, AgentGPTME,
		AgentMobileAgent, AgentMultiagentCoding, AgentNanocoder, AgentNoi,
		AgentOctogen, AgentOpenHands, AgentPostgresMCP, AgentShai, AgentSnowCLI,
		AgentTaskWeaver, AgentUIUXProMax, AgentVTCode, AgentWarp,
	}

	gen := NewUnifiedGenerator(nil)
	ctx := context.Background()

	for _, agentType := range genericAgents {
		t.Run(string(agentType), func(t *testing.T) {
			result, err := gen.Generate(ctx, agentType)
			if err != nil {
				t.Fatalf("Generate failed for %s: %v", agentType, err)
			}
			if !result.Success {
				t.Errorf("Expected Success for %s", agentType)
			}
			if result.Config == nil {
				t.Errorf("Config is nil for %s", agentType)
			}

			// Check schema
			schema, err := gen.GetSchema(agentType)
			if err != nil {
				t.Fatalf("GetSchema failed for %s: %v", agentType, err)
			}
			if schema.Description == "" {
				t.Errorf("Description is empty for %s", agentType)
			}
		})
	}
}

func TestAgentSpecificSettings(t *testing.T) {
	gen := NewUnifiedGenerator(nil)
	ctx := context.Background()

	// Test Aider specific settings
	result, _ := gen.Generate(ctx, AgentAider)
	config := result.Config.(*GenericAgentConfig)
	if _, ok := config.Settings["auto_commits"]; !ok {
		t.Error("Aider should have auto_commits setting")
	}

	// Test Continue specific settings
	result, _ = gen.Generate(ctx, AgentContinue)
	config = result.Config.(*GenericAgentConfig)
	if _, ok := config.Settings["tabAutocomplete"]; !ok {
		t.Error("Continue should have tabAutocomplete setting")
	}
}

func TestSupportedAgentsList(t *testing.T) {
	expected := []string{
		// Original 18
		"opencode", "crush", "helixcode", "kiro", "aider", "claude-code", "cline",
		"codename-goose", "deepseek-cli", "forge", "gemini-cli", "gpt-engineer",
		"kilocode", "mistral-code", "ollama-code", "plandex", "qwen-code", "amazon-q",
		// New 30
		"agent-deck", "bridle", "cheshire-cat", "claude-plugins", "claude-squad",
		"codai", "codex", "codex-skills", "conduit", "continue", "emdash",
		"fauxpilot", "get-shit-done", "github-copilot-cli", "github-spec-kit",
		"git-mcp", "gptme", "mobile-agent", "multiagent-coding", "nanocoder",
		"noi", "octogen", "openhands", "postgres-mcp", "shai", "snow-cli",
		"task-weaver", "ui-ux-pro-max", "vtcode", "warp",
	}

	if len(SupportedAgents) != 48 {
		t.Errorf("Expected 48 supported agents, got %d", len(SupportedAgents))
	}

	for _, exp := range expected {
		found := false
		for _, agent := range SupportedAgents {
			if agent == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected agent %s not found in SupportedAgents", exp)
		}
	}
}
