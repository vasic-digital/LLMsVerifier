package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Structures for reading the discovery JSON
type ModelInfo struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Capabilities []string               `json:"capabilities"`
	Features     map[string]interface{} `json:"features"`
	FreeToUse    bool                   `json:"free_to_use"`
}

type ProviderInfo struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	APIEndpoint string      `json:"api_endpoint"`
	Models      []ModelInfo `json:"models"`
	Status      string      `json:"status"`
	FreeToUse   bool        `json:"free_to_use"`
}

type ChallengeResult struct {
	Providers []ProviderInfo `json:"providers"`
}

// Crush config structures
type CrushModel struct {
	ID                     string            `json:"id"`
	Name                   string            `json:"name"`
	CostPer1MIn            float64           `json:"cost_per_1m_in"`
	CostPer1MOut           float64           `json:"cost_per_1m_out"`
	CostPer1MInCached      float64           `json:"cost_per_1m_in_cached"`
	CostPer1MOutCached     float64           `json:"cost_per_1m_out_cached"`
	ContextWindow          int               `json:"context_window"`
	DefaultMaxTokens       int               `json:"default_max_tokens"`
	CanReason              bool              `json:"can_reason"`
	ReasoningLevels        []string          `json:"reasoning_levels,omitempty"`
	DefaultReasoningEffort string            `json:"default_reasoning_effort,omitempty"`
	SupportsAttachments    bool              `json:"supports_attachments"`
	Streaming              bool              `json:"streaming,omitempty"`
	Options                CrushModelOptions `json:"options"`
}

type CrushModelOptions struct {
	Temperature      float64     `json:"temperature,omitempty"`
	TopP             float64     `json:"top_p,omitempty"`
	TopK             int         `json:"top_k,omitempty"`
	FrequencyPenalty float64     `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64     `json:"presence_penalty,omitempty"`
	ProviderOptions  interface{} `json:"provider_options,omitempty"`
}

type CrushProvider struct {
	Name    string       `json:"name"`
	Type    string       `json:"type"`
	BaseURL string       `json:"base_url"`
	Models  []CrushModel `json:"models"`
}

type CrushConfig struct {
	Schema    string                   `json:"$schema"`
	Providers map[string]CrushProvider `json:"providers"`
	LSP       map[string]interface{}   `json:"lsp"`
	Options   map[string]interface{}   `json:"options"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run converter.go <discovery_json_file>")
	}

	discoveryFile := os.Args[1]

	data, err := ioutil.ReadFile(discoveryFile)
	if err != nil {
		log.Fatalf("Failed to read discovery file: %v", err)
	}

	var result ChallengeResult
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	crushConfig := convertToCrushConfig(result)

	output, err := json.MarshalIndent(crushConfig, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal config: %v", err)
	}

	outputFile := strings.TrimSuffix(discoveryFile, filepath.Ext(discoveryFile)) + "_crush_config.json"
	if err := ioutil.WriteFile(outputFile, output, 0644); err != nil {
		log.Fatalf("Failed to write config: %v", err)
	}

	fmt.Printf("Crush config written to: %s\n", outputFile)
}

func convertToCrushConfig(result ChallengeResult) CrushConfig {
	providers := make(map[string]CrushProvider)

	for _, provider := range result.Providers {
		if len(provider.Models) == 0 {
			continue // Skip providers with no models
		}

		crushProvider := CrushProvider{
			Name:    provider.Name,
			Type:    getProviderType(provider.Name),
			BaseURL: provider.APIEndpoint,
			Models:  make([]CrushModel, 0, len(provider.Models)),
		}

		for _, model := range provider.Models {
			crushModel := CrushModel{
				ID:                  model.ID,
				Name:                model.Name,
				CostPer1MIn:         getCostPer1MIn(provider.Name, model.FreeToUse),
				CostPer1MOut:        getCostPer1MOut(provider.Name, model.FreeToUse),
				CostPer1MInCached:   getCostPer1MInCached(provider.Name, model.FreeToUse),
				CostPer1MOutCached:  getCostPer1MOutCached(provider.Name, model.FreeToUse),
				ContextWindow:       getContextWindow(model.ID),
				DefaultMaxTokens:    getDefaultMaxTokens(model.ID),
				CanReason:           hasCapability(model.Capabilities, "reasoning"),
				SupportsAttachments: hasCapability(model.Capabilities, "multimodal"),
				Streaming:           getStreamingSupport(model),
				Options:             CrushModelOptions{}, // Default empty options
			}

			if hasCapability(model.Capabilities, "multimodal") {
				crushModel.SupportsAttachments = true
			}

			crushProvider.Models = append(crushProvider.Models, crushModel)
		}

		providers[strings.ToLower(provider.Name)] = crushProvider
	}

	return CrushConfig{
		Schema:    "https://charm.land/crush.json",
		Providers: providers,
		LSP: map[string]interface{}{
			"go": map[string]interface{}{
				"command": "gopls",
				"enabled": true,
			},
			"typescript": map[string]interface{}{
				"command": "typescript-language-server",
				"args":    []string{"--stdio"},
				"enabled": true,
			},
		},
		Options: map[string]interface{}{
			"disable_provider_auto_update": true,
		},
	}
}

func getProviderType(name string) string {
	switch strings.ToLower(name) {
	case "anthropic":
		return "anthropic"
	case "openai":
		return "openai"
	default:
		return "openai" // Default to openai-compatible
	}
}

func getCostPer1MIn(provider string, free bool) float64 {
	if free {
		return 0
	}
	switch strings.ToLower(provider) {
	case "anthropic":
		return 3
	case "openai":
		return 3
	default:
		return 1
	}
}

func getCostPer1MOut(provider string, free bool) float64 {
	if free {
		return 0
	}
	switch strings.ToLower(provider) {
	case "anthropic":
		return 15
	case "openai":
		return 15
	default:
		return 5
	}
}

func getCostPer1MInCached(provider string, free bool) float64 {
	if free {
		return 0
	}
	return getCostPer1MIn(provider, free) * 0.5
}

func getCostPer1MOutCached(provider string, free bool) float64 {
	if free {
		return 0
	}
	return getCostPer1MOut(provider, free) * 0.5
}

func getContextWindow(modelID string) int {
	switch {
	case strings.Contains(modelID, "gpt-4"):
		return 128000
	case strings.Contains(modelID, "claude-3"):
		return 200000
	case strings.Contains(modelID, "llama-3"):
		return 8000
	default:
		return 4096
	}
}

func getDefaultMaxTokens(modelID string) int {
	switch {
	case strings.Contains(modelID, "gpt-4"):
		return 4096
	case strings.Contains(modelID, "claude-3"):
		return 8192
	default:
		return 4096
	}
}

func hasCapability(capabilities []string, cap string) bool {
	for _, c := range capabilities {
		if strings.Contains(strings.ToLower(c), strings.ToLower(cap)) {
			return true
		}
	}
	return false
}

func getStreamingSupport(model ModelInfo) bool {
	if streaming, ok := model.Features["streaming"]; ok {
		if b, ok := streaming.(bool); ok {
			return b
		}
	}
	return false
}
