package llmverifier

import (
	"encoding/json"
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

	config, err := createOfficialOpenCodeConfig(results, options)
	if err != nil {
		t.Fatalf("Failed to create OpenCode config: %v", err)
	}

	// Verify structure
	if config.Schema != "https://opencode.ai/config.json" {
		t.Errorf("Expected schema 'https://opencode.ai/config.json', got '%s'", config.Schema)
	}

	if len(config.Provider) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(config.Provider))
	}

	// Check OpenAI provider
	openaiProvider, exists := config.Provider["openai"]
	if !exists {
		t.Error("Expected 'openai' provider to exist")
	}
	if openaiProvider.Options.APIKey != "$OPENAI_API_KEY" {
		t.Errorf("Expected API key '$OPENAI_API_KEY', got '%s'", openaiProvider.Options.APIKey)
	}
	if len(openaiProvider.Models) != 0 {
		t.Errorf("Expected models to be empty, got %d models", len(openaiProvider.Models))
	}

	// Check Anthropic provider
	anthropicProvider, exists := config.Provider["anthropic"]
	if !exists {
		t.Error("Expected 'anthropic' provider to exist")
	}
	if anthropicProvider.Options.APIKey != "$ANTHROPIC_API_KEY" {
		t.Errorf("Expected API key '$ANTHROPIC_API_KEY', got '%s'", anthropicProvider.Options.APIKey)
	}

	// Test JSON marshaling
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Test unmarshaling
	var unmarshaled OpenCodeConfig
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if unmarshaled.Schema != config.Schema {
		t.Error("Schema not preserved during marshal/unmarshal")
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
