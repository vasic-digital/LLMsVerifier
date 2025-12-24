package main

import (
	"testing"
)

func TestConvertToCrushConfig(t *testing.T) {
	// Sample discovery data
	discovery := ChallengeResult{
		Providers: []ProviderInfo{
			{
				Name:        "OpenAI",
				APIEndpoint: "https://api.openai.com/v1",
				Models: []ModelInfo{
					{
						ID:           "gpt-4-turbo",
						Name:         "GPT-4 Turbo",
						Capabilities: []string{"text-generation", "chat", "reasoning"},
						FreeToUse:    false,
					},
				},
			},
		},
	}

	config := convertToCrushConfig(discovery)

	if config.Schema != "https://charm.land/crush.json" {
		t.Errorf("Expected schema https://charm.land/crush.json, got %s", config.Schema)
	}

	if len(config.Providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(config.Providers))
	}

	provider, exists := config.Providers["openai"]
	if !exists {
		t.Error("Expected openai provider")
	}

	if provider.Name != "OpenAI" {
		t.Errorf("Expected provider name OpenAI, got %s", provider.Name)
	}

	if len(provider.Models) != 1 {
		t.Errorf("Expected 1 model, got %d", len(provider.Models))
	}

	model := provider.Models[0]
	if model.ID != "gpt-4-turbo" {
		t.Errorf("Expected model ID gpt-4-turbo, got %s", model.ID)
	}

	if model.CanReason != true {
		t.Error("Expected CanReason to be true")
	}
}

func TestHasCapability(t *testing.T) {
	caps := []string{"text-generation", "chat", "reasoning"}

	if !hasCapability(caps, "reasoning") {
		t.Error("Expected to find reasoning capability")
	}

	if hasCapability(caps, "multimodal") {
		t.Error("Expected not to find multimodal capability")
	}
}

func TestGetContextWindow(t *testing.T) {
	if getContextWindow("gpt-4-turbo") != 128000 {
		t.Error("Expected 128000 for gpt-4")
	}

	if getContextWindow("claude-3-5-sonnet") != 200000 {
		t.Error("Expected 200000 for claude-3")
	}

	if getContextWindow("unknown") != 4096 {
		t.Error("Expected 4096 for unknown")
	}
}
