package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"llm-verifier/logging"
	"llm-verifier/verification"
)

// TestEnhancedModelsDevClient_Create tests client creation
func TestEnhancedModelsDevClient_Create(t *testing.T) {
	logger := logging.GetLogger()
	client := verification.NewEnhancedModelsDevClient(logger)

	if client == nil {
		t.Fatal("Expected client to be created, got nil")
	}

	if client.GetBaseURL() != "https://models.dev" {
		t.Errorf("Expected base URL https://models.dev, got %s", client.GetBaseURL())
	}

	if client.GetCacheEnabled() {
		t.Error("Cache should be disabled by default")
	}
}

// TestEnhancedModelsDevClient_FetchAllProviders tests fetching all providers
func TestEnhancedModelsDevClient_FetchAllProviders(t *testing.T) {
	// Create mock server
	mockData := createMockModelsDevResponse()
	server := createMockModelsDevServer(mockData)
	defer server.Close()

	logger := logging.GetLogger()
	client := verification.NewEnhancedModelsDevClient(logger)
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	providers, err := client.FetchAllProviders(ctx)

	if err != nil {
		t.Fatalf("Failed to fetch providers: %v", err)
	}

	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}

	// Check provider data
	openai, exists := providers["openai"]
	if !exists {
		t.Fatal("OpenAI provider not found")
	}

	if openai.Name != "OpenAI" {
		t.Errorf("Expected provider name 'OpenAI', got '%s'", openai.Name)
	}

	if openai.NPM != "@ai-sdk/openai" {
		t.Errorf("Expected NPM package '@ai-sdk/openai', got '%s'", openai.NPM)
	}

	if len(openai.Models) != 2 {
		t.Errorf("Expected 2 models for OpenAI, got %d", len(openai.Models))
	}
}

// TestEnhancedModelsDevClient_GetProviderByID tests getting specific provider
func TestEnhancedModelsDevClient_GetProviderByID(t *testing.T) {
	mockData := createMockModelsDevResponse()
	server := createMockModelsDevServer(mockData)
	defer server.Close()

	logger := logging.GetLogger()
	client := verification.NewEnhancedModelsDevClient(logger)
	client.SetBaseURL(server.URL)

	ctx := context.Background()

	// Test existing provider
	provider, err := client.GetProviderByID(ctx, "openai")
	if err != nil {
		t.Fatalf("Failed to get OpenAI provider: %v", err)
	}

	if provider.ID != "openai" {
		t.Errorf("Expected provider ID 'openai', got '%s'", provider.ID)
	}

	// Test non-existing provider
	_, err = client.GetProviderByID(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent provider, got nil")
	}

	expectedError := "provider nonexistent not found"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got '%s'", expectedError, err.Error())
	}
}

// TestEnhancedModelsDevClient_FindModel tests finding models
func TestEnhancedModelsDevClient_FindModel(t *testing.T) {
	mockData := createMockModelsDevResponse()
	server := createMockModelsDevServer(mockData)
	defer server.Close()

	logger := logging.GetLogger()
	client := verification.NewEnhancedModelsDevClient(logger)
	client.SetBaseURL(server.URL)

	ctx := context.Background()

	// Test exact match by model ID
	t.Run("ExactMatch", func(t *testing.T) {
		matches, err := client.FindModel(ctx, "gpt-4")
		if err != nil {
			t.Fatalf("Failed to find model: %v", err)
		}

		if len(matches) == 0 {
			t.Fatal("No matches found for gpt-4")
		}

		// Should have high match score
		if matches[0].MatchScore < 0.9 {
			t.Errorf("Expected high match score for exact match, got %f", matches[0].MatchScore)
		}
	})

	// Test provider/model path match
	t.Run("ProviderModelPath", func(t *testing.T) {
		matches, err := client.FindModel(ctx, "openai/gpt-4")
		if err != nil {
			t.Fatalf("Failed to find model: %v", err)
		}

		if len(matches) != 1 {
			t.Errorf("Expected 1 match for openai/gpt-4, got %d", len(matches))
		}

		if matches[0].ProviderID != "openai" || matches[0].ModelID != "gpt-4" {
			t.Error("Provider/model path match failed")
		}
	})

	// Test fuzzy matching
	t.Run("FuzzyMatch", func(t *testing.T) {
		matches, err := client.FindModel(ctx, "gpt3")
		if err != nil {
			// Fuzzy match might not find anything with this test data
			t.Logf("Fuzzy match returned error (may be expected): %v", err)
			return
		}

		if len(matches) > 0 {
			// Check that matches are sorted by score
			for i := 1; i < len(matches); i++ {
				if matches[i].MatchScore > matches[i-1].MatchScore {
					t.Errorf("Matches not sorted by score at index %d", i)
				}
			}
		}
	})

	// Test non-existent model
	t.Run("NonExistent", func(t *testing.T) {
		_, err := client.FindModel(ctx, "nonexistent-model-12345")
		if err == nil {
			t.Error("Expected error for non-existent model, got nil")
		}
	})
}

// TestEnhancedModelsDevClient_GetModelsByProviderID tests getting all models for provider
func TestEnhancedModelsDevClient_GetModelsByProviderID(t *testing.T) {
	mockData := createMockModelsDevResponse()
	server := createMockModelsDevServer(mockData)
	defer server.Close()

	logger := logging.GetLogger()
	client := verification.NewEnhancedModelsDevClient(logger)
	client.SetBaseURL(server.URL)

	ctx := context.Background()

	// Test existing provider
	matches, err := client.GetModelsByProviderID(ctx, "openai")
	if err != nil {
		t.Fatalf("Failed to get models: %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("Expected 2 models, got %d", len(matches))
	}

	// All matches should have perfect score for provider-wide query
	for _, match := range matches {
		if match.MatchScore != 1.0 {
			t.Errorf("Expected match score 1.0, got %f", match.MatchScore)
		}
	}

	// Test non-existent provider
	_, err = client.GetModelsByProviderID(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent provider")
	}
}

// TestEnhancedModelsDevClient_FilterModelsByFeature tests feature filtering
func TestEnhancedModelsDevClient_FilterModelsByFeature(t *testing.T) {
	mockData := createMockModelsDevResponse()
	server := createMockModelsDevServer(mockData)
	defer server.Close()

	logger := logging.GetLogger()
	client := verification.NewEnhancedModelsDevClient(logger)
	client.SetBaseURL(server.URL)

	ctx := context.Background()

	// Test tool_call feature
	t.Run("ToolCallFeature", func(t *testing.T) {
		matches, err := client.FilterModelsByFeature(ctx, "tool_call", 0.5)
		if err != nil {
			t.Fatalf("Failed to filter models: %v", err)
		}

		// gpt-4 and claude should support tool calls
		if len(matches) == 0 {
			t.Error("Expected models with tool_call support, got none")
		}

		// Verify all matches actually support tool calls
		for _, match := range matches {
			if !match.ModelData.ToolCall {
				t.Errorf("Model %s doesn't support tool_call but was returned", match.ModelID)
			}
		}
	})

	// Test reasoning feature
	t.Run("ReasoningFeature", func(t *testing.T) {
		matches, err := client.FilterModelsByFeature(ctx, "reasoning", 0.5)
		if err != nil {
			t.Fatalf("Failed to filter models: %v", err)
		}

		// reasoning-preview should support reasoning
		found := false
		for _, match := range matches {
			if match.ModelID == "reasoning-preview" {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected reasoning-preview in reasoning models")
		}
	})

	// Test with high threshold
	t.Run("HighThreshold", func(t *testing.T) {
		matches, err := client.FilterModelsByFeature(ctx, "tool_call", 1.0)
		if err != nil {
			t.Fatalf("Failed to filter models: %v", err)
		}

		// With threshold 1.0, only models with perfect score should be returned
		for _, match := range matches {
			if match.MatchScore < 1.0 {
				t.Errorf("Model %s returned with score %f, expected 1.0", 
					match.ModelID, match.MatchScore)
			}
		}
	})
}

// TestEnhancedModelsDevClient_GetTotalModelCount tests model counting
func TestEnhancedModelsDevClient_GetTotalModelCount(t *testing.T) {
	mockData := createMockModelsDevResponse()
	server := createMockModelsDevServer(mockData)
	defer server.Close()

	logger := logging.GetLogger()
	client := verification.NewEnhancedModelsDevClient(logger)
	client.SetBaseURL(server.URL)

	ctx := context.Background()

	count, err := client.GetTotalModelCount(ctx)
	if err != nil {
		t.Fatalf("Failed to get model count: %v", err)
	}

	// Our mock has 2 providers with 2 models each = 4 models
	if count != 4 {
		t.Errorf("Expected 4 models total, got %d", count)
	}
}

// TestEnhancedModelsDevClient_GetProviderStats tests statistics
func TestEnhancedModelsDevClient_GetProviderStats(t *testing.T) {
	mockData := createMockModelsDevResponse()
	server := createMockModelsDevServer(mockData)
	defer server.Close()

	logger := logging.GetLogger()
	client := verification.NewEnhancedModelsDevClient(logger)
	client.SetBaseURL(server.URL)

	ctx := context.Background()

	stats, err := client.GetProviderStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.TotalProviders != 2 {
		t.Errorf("Expected 2 providers, got %d", stats.TotalProviders)
	}

	if stats.TotalModels != 4 {
		t.Errorf("Expected 4 models, got %d", stats.TotalModels)
	}

	// Check NPM distribution
	if stats.ProvidersByNPM["@ai-sdk/openai"] != 1 {
		t.Error("NPM distribution stats incorrect")
	}

	// Check feature stats
	if stats.ModelsByFeature["tool_call"] != 3 { // gpt-4, claude, reasoning-preview
		t.Errorf("Expected 3 models with tool_call, got %d", stats.ModelsByFeature["tool_call"])
	}

	if stats.ModelsByFeature["reasoning"] != 1 { // reasoning-preview
		t.Errorf("Expected 1 model with reasoning, got %d", stats.ModelsByFeature["reasoning"])
	}

	// Check modality stats
	if stats.ModelsByModality["input_text"] != 4 {
		t.Errorf("Expected 4 models with text input, got %d", stats.ModelsByModality["input_text"])
	}

	if stats.OpenWeightModels != 1 { // reasoning-preview
		t.Errorf("Expected 1 open weight model, got %d", stats.OpenWeightModels)
	}
}

// TestEnhancedModelsDevClient_APIError tests error handling
func TestEnhancedModelsDevClient_APIError(t *testing.T) {
	// Create server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	logger := logging.GetLogger()
	client := verification.NewEnhancedModelsDevClient(logger)
	client.SetBaseURL(server.URL)

	ctx := context.Background()

	_, err := client.FetchAllProviders(ctx)
	if err == nil {
		t.Error("Expected error from API, got nil")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("Expected 500 error in message, got: %s", err.Error())
	}
}

// TestEnhancedModelsDevClient_NetworkError tests network error handling
func TestEnhancedModelsDevClient_NetworkError(t *testing.T) {
	logger := logging.GetLogger()
	client := verification.NewEnhancedModelsDevClient(logger)
	client.SetBaseURL("http://localhost:99999") // Invalid URL

	ctx := context.Background()

	_, err := client.FetchAllProviders(ctx)
	if err == nil {
		t.Error("Expected network error, got nil")
	}
}

// TestEnhancedModelsDevClient_Timeout tests timeout handling
func TestEnhancedModelsDevClient_Timeout(t *testing.T) {
	// Create slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	logger := logging.GetLogger()
	client := verification.NewEnhancedModelsDevClient(logger)
	client.SetBaseURL(server.URL)
	client.SetTimeout(100 * time.Millisecond)

	ctx := context.Background()

	_, err := client.FetchAllProviders(ctx)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

// Helper function to create mock models.dev response
func createMockModelsDevResponse() verification.ModelsDevEnhancedResponse {
	return verification.ModelsDevEnhancedResponse{
		"openai": verification.ProviderData{
			ID:   "openai",
			Env:  []string{"OPENAI_API_KEY"},
			NPM:  "@ai-sdk/openai",
			Name: "OpenAI",
			Doc:  "https://platform.openai.com/docs",
			Models: map[string]verification.ModelDetails{
				"gpt-4": {
					ID:          "gpt-4",
					Name:        "GPT-4",
					Family:      "gpt-4",
					Attachment:  true,
					Reasoning:   false,
					ToolCall:    true,
					Temperature: true,
					Knowledge:   "2024-04",
					ReleaseDate: "2024-05-13",
					LastUpdated: "2024-05-13",
					Modalities: verification.ModelModalities{
						Input:  []string{"text", "image"},
						Output: []string{"text"},
					},
					OpenWeights: false,
					Cost: verification.ModelCost{
						Input:  30.0,
						Output: 60.0,
					},
					Limits: verification.ModelLimits{
						Context: 128000,
						Input:   128000,
						Output:  4096,
					},
					StructuredOutput: true,
				},
				"reasoning-preview": {
					ID:          "reasoning-preview",
					Name:        "Reasoning Preview",
					Family:      "gpt-4",
					Attachment:  false,
					Reasoning:   true,
					ToolCall:    true,
					Temperature: true,
					Knowledge:   "2024-04",
					ReleaseDate: "2024-12-15",
					LastUpdated: "2024-12-15",
					Modalities: verification.ModelModalities{
						Input:  []string{"text"},
						Output: []string{"text"},
					},
					OpenWeights: true,
					Cost: verification.ModelCost{
						Input:  15.0,
						Output: 60.0,
					},
					Limits: verification.ModelLimits{
						Context: 32000,
						Input:   32000,
						Output:  8192,
					},
				},
			},
		},
		"anthropic": verification.ProviderData{
			ID:   "anthropic",
			Env:  []string{"ANTHROPIC_API_KEY"},
			NPM:  "@ai-sdk/anthropic",
			Name: "Anthropic",
			Doc:  "https://docs.anthropic.com",
			Models: map[string]verification.ModelDetails{
				"claude-3-5-sonnet": {
					ID:          "claude-3-5-sonnet",
					Name:        "Claude 3.5 Sonnet",
					Family:      "claude-3.5",
					Attachment:  true,
					Reasoning:   false,
					ToolCall:    true,
					Temperature: true,
					Knowledge:   "2024-04",
					ReleaseDate: "2024-10-22",
					LastUpdated: "2024-10-22",
					Modalities: verification.ModelModalities{
						Input:  []string{"text", "image"},
						Output: []string{"text"},
					},
					OpenWeights: false,
					Cost: verification.ModelCost{
						Input:  3.0,
						Output: 15.0,
					},
					Limits: verification.ModelLimits{
						Context: 200000,
						Input:   200000,
						Output:  8192,
					},
				},
			},
		},
	}
}

// Helper to create mock server
func createMockModelsDevServer(data verification.ModelsDevEnhancedResponse) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(data)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// Test helpers for client methods (needed since they're not exported)

func (c *verification.EnhancedModelsDevClient) GetBaseURL() string {
	return c.baseURL
}

func (c *verification.EnhancedModelsDevClient) GetCacheEnabled() bool {
	return c.cacheEnabled
}

func (c *verification.EnhancedModelsDevClient) SetBaseURL(url string) {
	c.baseURL = url
}

func (c *verification.EnhancedModelsDevClient) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

func (c *verification.EnhancedModelsDevClient) SetCacheEnabled(enabled bool) {
	c.cacheEnabled = enabled
}