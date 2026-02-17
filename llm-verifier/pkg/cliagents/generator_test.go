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

	// Must have at least 15 MCP servers
	if len(servers) < 15 {
		t.Errorf("Expected at least 15 MCP servers, got %d", len(servers))
	}

	// Check for HelixAgent remote endpoints (at least 6)
	helixAgentCount := 0
	localCount := 0
	freeRemoteCount := 0
	for _, server := range servers {
		if server.Type == "remote" && len(server.URL) > 0 {
			if len(server.URL) > 20 && server.URL[:21] == "http://localhost:7061" {
				helixAgentCount++
			} else if server.URL[:5] == "https" {
				freeRemoteCount++
			}
		}
		if server.Type == "local" {
			localCount++
		}
	}
	if helixAgentCount < 6 {
		t.Errorf("Expected at least 6 HelixAgent MCP endpoints, got %d", helixAgentCount)
	}
	if localCount < 6 {
		t.Errorf("Expected at least 6 local npx MCP servers, got %d", localCount)
	}
	if freeRemoteCount < 3 {
		t.Errorf("Expected at least 3 free remote MCP servers, got %d", freeRemoteCount)
	}
}

func TestDefaultMCPServersForHost(t *testing.T) {
	servers := DefaultMCPServersForHost("myhost", 9999)
	found := false
	for _, s := range servers {
		if s.Name == "helixagent-mcp" && s.URL == "http://myhost:9999/v1/mcp" {
			found = true
			break
		}
	}
	if !found {
		t.Error("DefaultMCPServersForHost did not configure custom host/port correctly")
	}
}

func TestDefaultPlugins(t *testing.T) {
	plugins := DefaultPlugins()
	if len(plugins) < 10 {
		t.Errorf("Expected at least 10 default plugins, got %d", len(plugins))
	}
}

func TestDefaultSkills(t *testing.T) {
	skills := DefaultSkills("localhost", 7061)
	if len(skills) < 8 {
		t.Errorf("Expected at least 8 default skills, got %d", len(skills))
	}
	for _, skill := range skills {
		if skill.Name == "" {
			t.Error("Skill name is empty")
		}
		if skill.Description == "" {
			t.Error("Skill description is empty")
		}
		if skill.Endpoint == "" {
			t.Error("Skill endpoint is empty")
		}
		if !skill.Enabled {
			t.Errorf("Skill %s should be enabled", skill.Name)
		}
	}
}

func TestDefaultHelixAgentExtensions(t *testing.T) {
	ext := DefaultHelixAgentExtensions("localhost", 7061)
	if ext == nil {
		t.Fatal("DefaultHelixAgentExtensions returned nil")
	}
	if ext.LSP == nil || !ext.LSP.Enabled {
		t.Error("LSP should be enabled")
	}
	if ext.ACP == nil || !ext.ACP.Enabled {
		t.Error("ACP should be enabled")
	}
	if ext.Embeddings == nil || !ext.Embeddings.Enabled {
		t.Error("Embeddings should be enabled")
	}
	if ext.RAG == nil || !ext.RAG.Enabled {
		t.Error("RAG should be enabled")
	}
	if len(ext.Plugins) < 10 {
		t.Errorf("Expected at least 10 plugins, got %d", len(ext.Plugins))
	}
	if len(ext.Skills) < 8 {
		t.Errorf("Expected at least 8 skills, got %d", len(ext.Skills))
	}
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
	if len(openCodeConfig.MCPServers) < 15 {
		t.Errorf("Expected at least 15 MCP servers, got %d", len(openCodeConfig.MCPServers))
	}
	if len(openCodeConfig.Agent) == 0 {
		t.Error("No agents configured")
	}
	if len(openCodeConfig.Plugin) == 0 {
		t.Error("No plugins configured")
	}
	if openCodeConfig.Extensions == nil {
		t.Error("Extensions is nil")
	} else {
		if openCodeConfig.Extensions.LSP == nil || !openCodeConfig.Extensions.LSP.Enabled {
			t.Error("LSP extension should be enabled")
		}
		if openCodeConfig.Extensions.ACP == nil || !openCodeConfig.Extensions.ACP.Enabled {
			t.Error("ACP extension should be enabled")
		}
		if openCodeConfig.Extensions.Embeddings == nil || !openCodeConfig.Extensions.Embeddings.Enabled {
			t.Error("Embeddings extension should be enabled")
		}
		if openCodeConfig.Extensions.RAG == nil || !openCodeConfig.Extensions.RAG.Enabled {
			t.Error("RAG extension should be enabled")
		}
		if len(openCodeConfig.Extensions.Skills) < 8 {
			t.Errorf("Expected at least 8 skills, got %d", len(openCodeConfig.Extensions.Skills))
		}
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
	if len(crushConfig.MCP) < 15 {
		t.Errorf("Expected at least 15 MCP servers, got %d", len(crushConfig.MCP))
	}
	if len(crushConfig.Plugins) == 0 {
		t.Error("No plugins configured")
	}
	if crushConfig.Extensions == nil {
		t.Error("Extensions is nil")
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
	if len(kiloConfig.MCP) < 15 {
		t.Errorf("Expected at least 15 MCP servers, got %d", len(kiloConfig.MCP))
	}
	if len(kiloConfig.Plugins) == 0 {
		t.Error("No plugins configured")
	}
	if kiloConfig.Extensions == nil {
		t.Error("Extensions is nil")
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
	if len(helixConfig.MCP) < 15 {
		t.Errorf("Expected at least 15 MCP servers, got %d", len(helixConfig.MCP))
	}
	if len(helixConfig.Plugins) == 0 {
		t.Error("No plugins configured")
	}
	if helixConfig.Extensions == nil {
		t.Error("Extensions is nil")
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
		"mcpServers": map[string]interface{}{
			"test": map[string]interface{}{
				"type": "sse",
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
		"mcpServers": map[string]interface{}{},
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
	if _, ok := parsed["plugin"]; !ok {
		t.Error("plugin field missing in JSON")
	}
	if _, ok := parsed["extensions"]; !ok {
		t.Error("extensions field missing in JSON")
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
				t.Fatalf("Config is nil for %s", agentType)
			}

			// Check schema
			schema, err := gen.GetSchema(agentType)
			if err != nil {
				t.Fatalf("GetSchema failed for %s: %v", agentType, err)
			}
			if schema.Description == "" {
				t.Errorf("Description is empty for %s", agentType)
			}

			// Verify generic agent has 15+ MCPs, plugins, and extensions
			genericConfig, ok := result.Config.(*GenericAgentConfig)
			if !ok {
				t.Fatalf("Config is not *GenericAgentConfig for %s", agentType)
			}
			if len(genericConfig.MCP) < 15 {
				t.Errorf("Expected at least 15 MCP servers for %s, got %d", agentType, len(genericConfig.MCP))
			}
			if len(genericConfig.Plugins) == 0 {
				t.Errorf("No plugins configured for %s", agentType)
			}
			if genericConfig.Extensions == nil {
				t.Errorf("Extensions is nil for %s", agentType)
			} else {
				if genericConfig.Extensions.LSP == nil || !genericConfig.Extensions.LSP.Enabled {
					t.Errorf("LSP should be enabled for %s", agentType)
				}
				if genericConfig.Extensions.ACP == nil || !genericConfig.Extensions.ACP.Enabled {
					t.Errorf("ACP should be enabled for %s", agentType)
				}
				if genericConfig.Extensions.Embeddings == nil || !genericConfig.Extensions.Embeddings.Enabled {
					t.Errorf("Embeddings should be enabled for %s", agentType)
				}
				if genericConfig.Extensions.RAG == nil || !genericConfig.Extensions.RAG.Enabled {
					t.Errorf("RAG should be enabled for %s", agentType)
				}
				if len(genericConfig.Extensions.Skills) < 8 {
					t.Errorf("Expected at least 8 skills for %s, got %d", agentType, len(genericConfig.Extensions.Skills))
				}
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

// TestAllAgentsCompleteness validates that ALL 48 agents have the required
// features: 15+ MCPs, plugins, LSP, ACP, Embeddings, RAG, Skills
func TestAllAgentsCompleteness(t *testing.T) {
	gen := NewUnifiedGenerator(nil)
	ctx := context.Background()

	results, err := gen.GenerateAll(ctx)
	if err != nil {
		t.Fatalf("GenerateAll failed: %v", err)
	}

	for _, result := range results {
		t.Run(string(result.AgentType), func(t *testing.T) {
			if !result.Success {
				t.Fatalf("Generation failed for %s", result.AgentType)
			}

			// Serialize to JSON and parse back to check structure
			jsonData, err := json.Marshal(result.Config)
			if err != nil {
				t.Fatalf("JSON marshal failed for %s: %v", result.AgentType, err)
			}

			var parsed map[string]interface{}
			if err := json.Unmarshal(jsonData, &parsed); err != nil {
				t.Fatalf("JSON unmarshal failed for %s: %v", result.AgentType, err)
			}

			// Check MCP servers (various field names across agents)
			mcpKey := ""
			for _, key := range []string{"mcp", "mcpServers"} {
				if _, ok := parsed[key]; ok {
					mcpKey = key
					break
				}
			}
			if mcpKey == "" {
				t.Errorf("Agent %s: no MCP servers field found", result.AgentType)
			} else {
				mcpMap, ok := parsed[mcpKey].(map[string]interface{})
				if !ok {
					t.Errorf("Agent %s: MCP field is not a map", result.AgentType)
				} else if len(mcpMap) < 15 {
					t.Errorf("Agent %s: expected at least 15 MCP servers, got %d", result.AgentType, len(mcpMap))
				}
			}

			// Check plugins exist
			if plugins, ok := parsed["plugins"]; ok {
				pluginArr, ok := plugins.([]interface{})
				if !ok {
					t.Errorf("Agent %s: plugins field is not an array", result.AgentType)
				} else if len(pluginArr) == 0 {
					t.Errorf("Agent %s: plugins array is empty", result.AgentType)
				}
			} else {
				// OpenCode uses "plugin" key
				if plugin, ok := parsed["plugin"]; ok {
					pluginArr, ok := plugin.([]interface{})
					if !ok || len(pluginArr) == 0 {
						t.Errorf("Agent %s: plugin field is invalid or empty", result.AgentType)
					}
				} else {
					t.Errorf("Agent %s: no plugins field found", result.AgentType)
				}
			}

			// Check extensions exist
			extensions, hasExtensions := parsed["extensions"]
			if !hasExtensions {
				t.Errorf("Agent %s: no extensions field found", result.AgentType)
			} else {
				extMap, ok := extensions.(map[string]interface{})
				if !ok {
					t.Errorf("Agent %s: extensions is not a map", result.AgentType)
				} else {
					// Check LSP
					if lsp, ok := extMap["lsp"]; ok {
						lspMap, ok := lsp.(map[string]interface{})
						if !ok || lspMap["enabled"] != true {
							t.Errorf("Agent %s: LSP should be enabled", result.AgentType)
						}
					} else {
						t.Errorf("Agent %s: LSP extension missing", result.AgentType)
					}

					// Check ACP
					if acp, ok := extMap["acp"]; ok {
						acpMap, ok := acp.(map[string]interface{})
						if !ok || acpMap["enabled"] != true {
							t.Errorf("Agent %s: ACP should be enabled", result.AgentType)
						}
					} else {
						t.Errorf("Agent %s: ACP extension missing", result.AgentType)
					}

					// Check Embeddings
					if emb, ok := extMap["embeddings"]; ok {
						embMap, ok := emb.(map[string]interface{})
						if !ok || embMap["enabled"] != true {
							t.Errorf("Agent %s: Embeddings should be enabled", result.AgentType)
						}
					} else {
						t.Errorf("Agent %s: Embeddings extension missing", result.AgentType)
					}

					// Check RAG
					if rag, ok := extMap["rag"]; ok {
						ragMap, ok := rag.(map[string]interface{})
						if !ok || ragMap["enabled"] != true {
							t.Errorf("Agent %s: RAG should be enabled", result.AgentType)
						}
					} else {
						t.Errorf("Agent %s: RAG extension missing", result.AgentType)
					}

					// Check Skills
					if skills, ok := extMap["skills"]; ok {
						skillsArr, ok := skills.([]interface{})
						if !ok || len(skillsArr) < 8 {
							t.Errorf("Agent %s: expected at least 8 skills", result.AgentType)
						}
					} else {
						t.Errorf("Agent %s: Skills extension missing", result.AgentType)
					}
				}
			}

			// Check formatters exist
			if _, ok := parsed["formatters"]; !ok {
				t.Errorf("Agent %s: no formatters field found", result.AgentType)
			}
		})
	}
}
