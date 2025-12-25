// Package providers implements LLM provider integrations
package providers

import (
	"context"
	"fmt"
	"time"

	"llm-verifier/client"
	"llm-verifier/database"
)

// GroqProvider implements the Provider interface for Groq
type GroqProvider struct{}

// NewGroqProvider creates a new Groq provider instance
func NewGroqProvider() *GroqProvider {
	return &GroqProvider{}
}

// Name returns the provider name
func (p *GroqProvider) Name() string {
	return "groq"
}

// Endpoint returns the API endpoint
func (p *GroqProvider) Endpoint() string {
	return "https://api.groq.com/openai/v1"
}

// SupportsModel checks if a model is supported
func (p *GroqProvider) SupportsModel(modelID string) bool {
	supportedModels := []string{
		"llama2-70b-4096",
		"llama2-7b-2048",
		"mixtral-8x7b-32768",
	}
	for _, model := range supportedModels {
		if model == modelID {
			return true
		}
	}
	return false
}

// DiscoverModels discovers available models from Groq
func (p *GroqProvider) DiscoverModels(ctx context.Context, apiKey string) ([]*database.Model, error) {
	// Use HTTP client to fetch models
	httpClient := client.NewHTTPClient(30 * time.Second)
	models, err := httpClient.FetchModels(ctx, p.Name(), apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to discover Groq models: %w", err)
	}

	var dbModels []*database.Model
	for _, model := range models {
		dbModel := &database.Model{
			ProviderID:        0, // Set by caller
			ModelID:           model.ID,
			Name:              model.Name,
			Description:       fmt.Sprintf("Groq %s model", model.Name),
			MaxInputTokens:    model.MaxInputTokens,
			MaxOutputTokens:   model.MaxOutputTokens,
			SupportsStreaming: true,
		}
		dbModels = append(dbModels, dbModel)
	}

	return dbModels, nil
}

// ValidateConfig validates provider configuration
func (p *GroqProvider) ValidateConfig(config map[string]interface{}) error {
	if apiKey, ok := config["api_key"].(string); !ok || apiKey == "" {
		return fmt.Errorf("Groq API key is required")
	}
	return nil
}

// GetRateLimits returns rate limits for Groq
func (p *GroqProvider) GetRateLimits() (int, int) { // requests per minute, tokens per minute
	return 30, 10000
}

// GetPricing returns pricing information
func (p *GroqProvider) GetPricing() (float64, float64) { // input per 1M tokens, output per 1M tokens
	return 0.01, 0.02
}

// VerifyModel performs verification test on a Groq model
func (p *GroqProvider) VerifyModel(ctx context.Context, apiKey, modelID string, testPrompt string) (*database.VerificationResult, error) {
	httpClient := client.NewHTTPClient(30 * time.Second)

	// Send test request
	response, err := httpClient.SendRequest(ctx, p.Name(), apiKey, modelID, testPrompt)
	if err != nil {
		return nil, fmt.Errorf("Groq verification failed: %w", err)
	}

	// Parse and score response
	result := &database.VerificationResult{
		ProviderID: 0, // Set by caller
		ModelID:    modelID,
		TestType:   "basic_verification",
		Status:     "completed",
		Response:   response,
		Score:      95.0, // Placeholder scoring
		Metadata:   map[string]interface{}{"provider": "groq"},
	}

	return result, nil
}

// IsHealthy checks if Groq service is accessible
func (p *GroqProvider) IsHealthy(ctx context.Context, apiKey string) bool {
	// Simple health check by trying to list models
	_, err := p.DiscoverModels(ctx, apiKey)
	return err == nil
}
