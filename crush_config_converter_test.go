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
						Features:     map[string]interface{}{"streaming": true},
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

	if model.Streaming != true {
		t.Error("Expected Streaming to be true")
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

func TestGetStreamingSupport(t *testing.T) {
	modelWithStreaming := ModelInfo{
		Features: map[string]interface{}{"streaming": true},
	}
	if !getStreamingSupport(modelWithStreaming) {
		t.Error("Expected streaming support to be true")
	}

	modelWithoutStreaming := ModelInfo{
		Features: map[string]interface{}{},
	}
	if getStreamingSupport(modelWithoutStreaming) {
		t.Error("Expected streaming support to be false")
	}

	modelWithFalseStreaming := ModelInfo{
		Features: map[string]interface{}{"streaming": false},
	}
	if getStreamingSupport(modelWithFalseStreaming) {
		t.Error("Expected streaming support to be false")
	}

	modelWithNonBoolStreaming := ModelInfo{
		Features: map[string]interface{}{"streaming": "yes"},
	}
	if getStreamingSupport(modelWithNonBoolStreaming) {
		t.Error("Expected streaming support to be false for non-bool value")
	}
}

func TestGetCostPer1MIn(t *testing.T) {
	if getCostPer1MIn("Anthropic", false) != 3 {
		t.Error("Expected 3 for Anthropic paid")
	}
	if getCostPer1MIn("Anthropic", true) != 0 {
		t.Error("Expected 0 for Anthropic free")
	}
	if getCostPer1MIn("Unknown", false) != 1 {
		t.Error("Expected 1 for unknown paid")
	}
}

func TestGetCostPer1MOut(t *testing.T) {
	if getCostPer1MOut("OpenAI", false) != 15 {
		t.Error("Expected 15 for OpenAI paid")
	}
	if getCostPer1MOut("OpenAI", true) != 0 {
		t.Error("Expected 0 for OpenAI free")
	}
	if getCostPer1MOut("Unknown", false) != 5 {
		t.Error("Expected 5 for unknown paid")
	}
}

func TestGetProviderType(t *testing.T) {
	if getProviderType("Anthropic") != "anthropic" {
		t.Error("Expected anthropic for Anthropic")
	}
	if getProviderType("OpenAI") != "openai" {
		t.Error("Expected openai for OpenAI")
	}
	if getProviderType("Unknown") != "openai" {
		t.Error("Expected openai default")
	}
}

func TestConvertToCrushConfigMultipleProviders(t *testing.T) {
	discovery := ChallengeResult{
		Providers: []ProviderInfo{
			{
				Name:        "OpenAI",
				APIEndpoint: "https://api.openai.com/v1",
				Models: []ModelInfo{
					{
						ID:           "gpt-4-turbo",
						Name:         "GPT-4 Turbo",
						Capabilities: []string{"text-generation", "chat"},
						Features:     map[string]interface{}{"streaming": true},
						FreeToUse:    false,
					},
				},
			},
			{
				Name:        "Anthropic",
				APIEndpoint: "https://api.anthropic.com/v1",
				Models: []ModelInfo{
					{
						ID:           "claude-3-5-sonnet",
						Name:         "Claude 3.5 Sonnet",
						Capabilities: []string{"text-generation", "chat", "reasoning"},
						Features:     map[string]interface{}{},
						FreeToUse:    false,
					},
				},
			},
		},
	}

	config := convertToCrushConfig(discovery)

	if len(config.Providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(config.Providers))
	}

	openai, exists := config.Providers["openai"]
	if !exists {
		t.Error("Expected openai provider")
	}
	if openai.Type != "openai" {
		t.Error("Expected openai type")
	}
	if len(openai.Models) != 1 {
		t.Error("Expected 1 model for openai")
	}
	if !openai.Models[0].Streaming {
		t.Error("Expected streaming true for gpt-4-turbo")
	}

	anthropic, exists := config.Providers["anthropic"]
	if !exists {
		t.Error("Expected anthropic provider")
	}
	if anthropic.Type != "anthropic" {
		t.Error("Expected anthropic type")
	}
	if !anthropic.Models[0].CanReason {
		t.Error("Expected CanReason true for claude")
	}
}
