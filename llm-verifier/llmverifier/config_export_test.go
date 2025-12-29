package llmverifier

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOpenCodeConfigExport(t *testing.T) {
	// Create mock verification results
	results := []VerificationResult{
		{
			ModelInfo: ModelInfo{
				ID:          "gpt-4-turbo",
				Description: "Latest GPT-4 model",
				Endpoint:    "https://api.openai.com/v1",
				Tags:        []string{"coding", "reasoning"},
			},
			PerformanceScores: PerformanceScore{
				OverallScore:     92.5,
				CodeCapability:   95.0,
				Responsiveness:   90.0,
				Reliability:      94.0,
				FeatureRichness:  93.0,
				ValueProposition: 85.0,
			},
		},
		{
			ModelInfo: ModelInfo{
				ID:          "claude-3-5-sonnet-20241022",
				Description: "Anthropic's most capable model",
				Endpoint:    "https://api.anthropic.com/v1",
				Tags:        []string{"reasoning", "coding"},
			},
			PerformanceScores: PerformanceScore{
				OverallScore:     89.7,
				CodeCapability:   88.0,
				Responsiveness:   87.0,
				Reliability:      92.0,
				FeatureRichness:  91.0,
				ValueProposition: 82.0,
			},
		},
	}

	// Test exporting with API keys
	options := &ExportOptions{
		IncludeAPIKey: true,
	}

	configMap, err := createCorrectOpenCodeConfig(results, options)
	if err != nil {
		t.Fatalf("Failed to create OpenCode config: %v", err)
	}

	// Verify schema
	if schema, ok := configMap["$schema"].(string); !ok || schema != "./opencode-schema.json" {
		t.Errorf("Expected schema './opencode-schema.json', got '%v'", configMap["$schema"])
	}

	// Check providers section
	providers, ok := configMap["providers"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected providers to be a map")
	}
	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}

	// Check OpenAI provider
	openaiProviderData, exists := providers["openai"]
	if !exists {
		t.Error("Expected 'openai' provider to exist")
	}
	openaiProvider, ok := openaiProviderData.(map[string]interface{})
	if !ok {
		t.Fatal("Expected openai provider to be a map")
	}
	if apiKey, ok := openaiProvider["apiKey"].(string); !ok || apiKey != "${OPENAI_API_KEY}" {
		t.Errorf("Expected API key '${OPENAI_API_KEY}', got '%v'", openaiProvider["apiKey"])
	}

	// Check Anthropic provider
	anthropicProviderData, exists := providers["anthropic"]
	if !exists {
		t.Error("Expected 'anthropic' provider to exist")
	}
	anthropicProvider, ok := anthropicProviderData.(map[string]interface{})
	if !ok {
		t.Fatal("Expected anthropic provider to be a map")
	}
	if apiKey, ok := anthropicProvider["apiKey"].(string); !ok || apiKey != "${ANTHROPIC_API_KEY}" {
		t.Errorf("Expected API key '${ANTHROPIC_API_KEY}', got '%v'", anthropicProvider["apiKey"])
	}

	// Check agents section
	agents, ok := configMap["agents"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected agents to be a map")
	}

	requiredAgents := []string{"coder", "task", "title"}
	for _, agentName := range requiredAgents {
		if _, exists := agents[agentName]; !exists {
			t.Errorf("Expected agent '%s' to exist", agentName)
		}
	}

	// Test JSON marshaling
	data, err := json.MarshalIndent(configMap, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Test unmarshaling back to map
	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if unmarshaled["$schema"] != configMap["$schema"] {
		t.Error("Schema not preserved during marshal/unmarshal")
	}

	// Verify the config matches OpenCode's expected structure
	// Check that it has all required sections for OpenCode
	requiredTopLevelKeys := []string{"$schema", "data", "providers", "agents", "tui", "shell", "autoCompact", "debug", "debugLSP"}
	for _, key := range requiredTopLevelKeys {
		if _, exists := configMap[key]; !exists {
			t.Errorf("Missing required top-level key: %s", key)
		}
	}

	// Verify data section has directory
	if data, ok := configMap["data"].(map[string]interface{}); ok {
		if _, hasDir := data["directory"]; !hasDir {
			t.Error("data section should have directory field")
		}
	} else {
		t.Error("data section should be an object")
	}

	// Verify agents have proper model references (provider.model format)
	if agents, ok := configMap["agents"].(map[string]interface{}); ok {
		for agentName, agentData := range agents {
			if agent, ok := agentData.(map[string]interface{}); ok {
				if model, hasModel := agent["model"]; hasModel {
					if modelStr, ok := model.(string); ok && !strings.Contains(modelStr, ".") {
						t.Errorf("Agent %s model should be in provider.model format, got: %s", agentName, modelStr)
					}
				}
			}
		}
	}
}

func TestOpenCodeConfigValidation(t *testing.T) {
	// Create a valid config
	config := &OpenCodeConfig{
		Schema: "https://opencode.ai/config.json",
		Provider: map[string]OpenCodeProvider{
			"openai": {
				Options: OpenCodeOptions{APIKey: "$OPENAI_API_KEY"},
				Models:  map[string]any{},
			},
		},
	}

	// Test valid config
	if err := validateOpenCodeConfigStructure(config); err != nil {
		t.Errorf("Valid config failed validation: %v", err)
	}

	// Test missing schema
	invalidConfig := &OpenCodeConfig{
		Schema:   "",
		Provider: config.Provider,
	}
	if err := validateOpenCodeConfigStructure(invalidConfig); err == nil {
		t.Error("Expected error for missing schema")
	}

	// Test invalid schema
	invalidConfig = &OpenCodeConfig{
		Schema:   "https://invalid.com/schema.json",
		Provider: config.Provider,
	}
	if err := validateOpenCodeConfigStructure(invalidConfig); err == nil {
		t.Error("Expected error for invalid schema")
	}

	// Test missing providers
	invalidConfig = &OpenCodeConfig{
		Schema:   "https://opencode.ai/config.json",
		Provider: nil,
	}
	if err := validateOpenCodeConfigStructure(invalidConfig); err == nil {
		t.Error("Expected error for missing providers")
	}

	// Test missing API key
	invalidConfig = &OpenCodeConfig{
		Schema: "https://opencode.ai/config.json",
		Provider: map[string]OpenCodeProvider{
			"openai": {
				Options: OpenCodeOptions{APIKey: ""},
				Models:  map[string]any{},
			},
		},
	}
	if err := validateOpenCodeConfigStructure(invalidConfig); err == nil {
		t.Error("Expected error for missing API key")
	}

	// Test non-empty models (should fail)
	invalidConfig = &OpenCodeConfig{
		Schema: "https://opencode.ai/config.json",
		Provider: map[string]OpenCodeProvider{
			"openai": {
				Options: OpenCodeOptions{APIKey: "$OPENAI_API_KEY"},
				Models:  map[string]any{"gpt-4": "invalid"},
			},
		},
	}
	if err := validateOpenCodeConfigStructure(invalidConfig); err == nil {
		t.Error("Expected error for non-empty models")
	}
}
