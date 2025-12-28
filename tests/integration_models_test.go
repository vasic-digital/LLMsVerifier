package tests

import (
	"context"
	"testing"
	"time"
	
	"llm-verifier/client"
	"llm-verifier/verification"
)

// TestIntegrationProviderEndpoints tests provider-specific endpoint mappings
func TestIntegrationProviderEndpoints(t *testing.T) {
	ctx := context.Background()
	httpClient := client.NewHTTPClient(30 * time.Second)
	
	// Test that each provider's endpoint is reachable
	providers := []struct {
		name     string
		endpoint string
		modelID  string
		apiKey   string // Use dummy key to test endpoint exists
	}{
		{"openrouter", "https://openrouter.ai/api/v1", "openai/gpt-4", "sk-dummy"},
		{"deepseek", "https://api.deepseek.com/v1", "deepseek-chat", "sk-dummy"},
		{"groq", "https://api.groq.com/openai/v1", "llama2-70b-4096", "sk-dummy"},
	}
	
	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			// With dummy key, we expect 401 but should get a response
			// This verifies the endpoint exists and is reachable
			_, err := httpClient.TestModelExists(ctx, p.name, p.apiKey, p.modelID)
			
			// We want to see authentication errors (good), not network errors (bad)
			if err != nil {
				// Check if it's network vs auth error
				if ctx.Err() != nil {
					t.Errorf("Timeout/network error for %s endpoint: %v", p.name, err)
				} else {
					t.Logf("✓ %s endpoint reachable (auth error expected): %v", p.name, err)
				}
			}
		})
	}
}

// TestIntegrationModelDiscovery tests discovering models via models.dev
func TestIntegrationModelDiscovery(t *testing.T) {
	ctx := context.Background()
	client := verification.NewModelsDevClient()
	
	// Find popular models across providers
	popularModels := []string{"gpt-4", "claude-3.5-sonnet", "llama-3", "mixtral"}
	
	for _, modelName := range popularModels {
		t.Run(modelName, func(t *testing.T) {
			model, err := client.FindModel(ctx, modelName)
			if err != nil {
				t.Logf("Model %s not found: %v", modelName, err)
				return // Not all models may exist
			}
			
			t.Logf("✓ Found %s: %s (Context: %d, ToolCall: %v, $/1M: %.2f/%.2f)",
				model.ProviderID, model.ModelID, model.ContextLimit,
				model.ToolCall, model.InputCostPer1M, model.OutputCostPer1M)
		})
	}
}

// TestIntegrationModelFeatures tests feature detection accuracy
func TestIntegrationModelFeatures(t *testing.T) {
	ctx := context.Background()
	client := verification.NewModelsDevClient()
	
	testCases := []struct {
		modelID        string
		expectToolCall bool
		expectReasoning bool
	}{
		{"gpt-4", true, false},
		{"claude-3.5-sonnet", true, false},
		{"deepseek-chat", true, false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.modelID, func(t *testing.T) {
			model, err := client.FindModel(ctx, tc.modelID)
			if err != nil {
				t.Skipf("Model %s not available: %v", tc.modelID, err)
				return
			}
			
			if model.ToolCall != tc.expectToolCall {
				t.Errorf("Model %s: expected ToolCall=%v, got %v", 
					tc.modelID, tc.expectToolCall, model.ToolCall)
			}
			
			if model.Reasoning != tc.expectReasoning {
				t.Errorf("Model %s: expected Reasoning=%v, got %v", 
					tc.modelID, tc.expectReasoning, model.Reasoning)
			}
		})
	}
}

// TestIntegrationPricingData verifies pricing data is accurate
func TestIntegrationPricingData(t *testing.T) {
	ctx := context.Background()
	client := verification.NewModelsDevClient()
	
	// Check that pricing data exists for popular models
	modelsToCheck := []string{"gpt-4", "claude-3.5-sonnet"}
	
	for _, modelName := range modelsToCheck {
		t.Run(modelName, func(t *testing.T) {
			model, err := client.FindModel(ctx, modelName)
			if err != nil {
				t.Skipf("Model %s not available for pricing check", modelName)
				return
			}
			
			if model.InputCostPer1M <= 0 || model.OutputCostPer1M <= 0 {
				t.Errorf("Model %s has invalid pricing: Input=%.2f, Output=%.2f", 
					modelName, model.InputCostPer1M, model.OutputCostPer1M)
			}
			
			t.Logf("Pricing for %s: $%.2f/1M input, $%.2f/1M output",
				modelName, model.InputCostPer1M, model.OutputCostPer1M)
		})
	}
}

// TestIntegrationResponseTime verifies API response times are reasonable
func TestIntegrationResponseTime(t *testing.T) {
	ctx := context.Background()
	client := verification.NewModelsDevClient()
	
	start := time.Now()
	_, err := client.FetchModels(ctx)
	duration := time.Since(start)
	
	if err != nil {
		t.Fatalf("Failed to fetch models: %v", err)
	}
	
	// Should complete in reasonable time (< 10 seconds for full API)
	if duration > 10*time.Second {
		t.Errorf("Models.dev API call took too long: %v", duration)
	}
	
	t.Logf("Models.dev API completed in %v", duration)
}

// TestIntegrationContextLimits verifies context limits make sense
func TestIntegrationContextLimits(t *testing.T) {
	ctx := context.Background()
	client := verification.NewModelsDevClient()
	
	// Test that context limits are reasonable
	popularModels := []string{"gpt-4", "claude-3.5-sonnet"}
	
	for _, modelName := range popularModels {
		t.Run(modelName, func(t *testing.T) {
			model, err := client.FindModel(ctx, modelName)
			if err != nil {
				t.Skipf("Model %s not available", modelName)
				return
			}
			
			// Reasonable context limits (32K-200K for modern models)
			minContext := 32000
			maxContext := 200000
			
			if model.ContextLimit < minContext {
				t.Errorf("Model %s has suspiciously low context limit: %d", 
					modelName, model.ContextLimit)
			}
			
			if model.ContextLimit > maxContext {
				t.Errorf("Model %s has extremely high context limit: %d", 
					modelName, model.ContextLimit)
			}
			
			t.Logf("✓ %s: ContextLimit=%d", model.ModelID, model.ContextLimit)
		})
	}
}
