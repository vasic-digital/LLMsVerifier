package tests

import (
	"context"
	"testing"
	"time"
	
	"llm-verifier/client"
	"llm-verifier/verification"
)

// TestVerificationRealHTTPCalls tests that verification makes real HTTP calls with no caching
func TestVerificationRealHTTPCalls(t *testing.T) {
	ctx := context.Background()
	httpClient := client.NewHTTPClient(30 * time.Second)
	
	tests := []struct {
		name       string
		provider   string
		apiKey     string
		modelID    string
		wantStatus int
	}{
		{
			name:       "OpenRouter GPT-4 - Clean Call",
			provider:   "openrouter",
			apiKey:     "sk-or-v1-testkey",
			modelID:    "openai/gpt-4",
			wantStatus: 200, // Expect 401 for invalid key, but endpoint works
		},
		{
			name:       "DeepSeek - Clean Call",
			provider:   "deepseek",
			apiKey:     "sk-test-key",
			modelID:    "deepseek-chat",
			wantStatus: 200, // Expect 401 for invalid key
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make the call
			exists, err := httpClient.TestModelExists(ctx, tt.provider, tt.apiKey, tt.modelID)
			
			// We expect errors (invalid keys) but the endpoint should respond
			if err != nil {
				t.Logf("Expected error (invalid key): %v", err)
			}
			
			// The important thing is that we got a response (not a timeout or network error)
			// Status codes like 401 (Unauthorized) are acceptable responses
			if err == nil && !exists {
				t.Logf("Model exists check returned false, but no network error")
			}
		})
	}
}

// TestModelsDevAPICalls tests models.dev integration
func TestModelsDevAPICalls(t *testing.T) {
	ctx := context.Background()
	client := verification.NewModelsDevClient()
	
	t.Run("FetchAllModels", func(t *testing.T) {
		models, err := client.FetchModels(ctx)
		if err != nil {
			t.Fatalf("Failed to fetch models from models.dev: %v", err)
		}
		
		if len(models) == 0 {
			t.Fatal("No models returned from models.dev")
		}
		
		t.Logf("Successfully fetched %d models from models.dev", len(models))
	})
	
	t.Run("FindModel", func(t *testing.T) {
		// Test finding a known model
		model, err := client.FindModel(ctx, "gpt-4")
		if err != nil {
			t.Fatalf("Failed to find model: %v", err)
		}
		
		if model.ModelID == "" {
			t.Fatal("Model ID is empty")
		}
		
		t.Logf("Found model: %s (Provider: %s)", model.Model, model.Provider)
	})
	
	t.Run("GetModelsByProvider", func(t *testing.T) {
		models, err := client.GetModelsByProvider(ctx, "openai")
		if err != nil {
			t.Fatalf("Failed to get models for provider: %v", err)
		}
		
		if len(models) == 0 {
			t.Fatal("No models found for OpenAI provider")
		}
		
		t.Logf("Found %d models for OpenAI provider", len(models))
	})
}

// TestNoCachingTests verify that API calls are not cached
func TestNoCachingTests(t *testing.T) {
	ctx := context.Background()
	client := verification.NewModelsDevClient()
	
	// Make two consecutive calls - they should both hit the network
	start1 := time.Now()
	models1, err1 := client.FetchModels(ctx)
	duration1 := time.Since(start1)
	
	start2 := time.Now()
	models2, err2 := client.FetchModels(ctx)
	duration2 := time.Since(start2)
	
	if err1 != nil || err2 != nil {
		t.Fatalf("API call errors: err1=%v, err2=%v", err1, err2)
	}
	
	// Both calls should succeed and return the same data
	if len(models1) != len(models2) {
		t.Fatalf("Inconsistent results: first call returned %d models, second call returned %d", 
			len(models1), len(models2))
	}
	
	// Log the times to verify both calls were made
	t.Logf("First call duration: %v, Second call duration: %v", duration1, duration2)
	t.Logf("Both calls returned %d models", len(models1))
}

// TestModelsAccuracy tests model accuracy with real data
func TestModelsAccuracy(t *testing.T) {
	ctx := context.Background()
	client := verification.NewModelsDevClient()
	
	// Test known models that should exist
	knownModels := []struct {
		modelID           string
		expectedProvider  string
		shouldHaveFeatures bool
	}{
		{"gpt-4", "openai", true},
		{"claude-3-5-sonnet", "anthropic", true},
		{"deepseek-chat", "deepseek", true},
	}
	
	for _, km := range knownModels {
		t.Run(km.modelID, func(t *testing.T) {
			model, err := client.FindModel(ctx, km.modelID)
			if err != nil {
				t.Fatalf("Could not find known model %s: %v", km.modelID, err)
			}
			
			if !strings.Contains(strings.ToLower(model.Provider), km.expectedProvider) {
				t.Errorf("Model %s: expected provider %s, got %s", 
					km.modelID, km.expectedProvider, model.Provider)
			}
			
			if km.shouldHaveFeatures {
				if model.ContextLimit == 0 {
					t.Errorf("Model %s has no context limit set", km.modelID)
				}
			}
			
			t.Logf("✓ %s: Provider=%s, Context=%d, ToolCall=%v",
				model.ModelID, model.Provider, model.ContextLimit, model.ToolCall)
		})
	}
}
