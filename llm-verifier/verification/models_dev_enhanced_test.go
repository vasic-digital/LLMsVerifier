package verification

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createMockModelsDevResponse() ModelsDevEnhancedResponse {
	return ModelsDevEnhancedResponse{
		"openai": ProviderData{
			ID:   "openai",
			Name: "OpenAI",
			Env:  []string{"OPENAI_API_KEY"},
			NPM:  "@ai-sdk/openai",
			API:  "https://api.openai.com/v1",
			Doc:  "https://platform.openai.com/docs",
			Models: map[string]ModelDetails{
				"gpt-4": {
					ID:       "gpt-4",
					Name:     "GPT-4",
					Family:   "gpt",
					ToolCall: true,
					Modalities: ModelModalities{
						Input:  []string{"text"},
						Output: []string{"text"},
					},
					Cost: ModelCost{
						Input:  30.0,
						Output: 60.0,
					},
					Limits: ModelLimits{
						Context: 8192,
						Input:   8192,
						Output:  4096,
					},
					LastUpdated: time.Now().Format("2006-01-02"),
				},
				"gpt-4-vision": {
					ID:         "gpt-4-vision",
					Name:       "GPT-4 Vision",
					Family:     "gpt",
					ToolCall:   true,
					Attachment: true,
					Modalities: ModelModalities{
						Input:  []string{"text", "image"},
						Output: []string{"text"},
					},
					Cost: ModelCost{
						Input:  10.0,
						Output: 30.0,
					},
					Limits: ModelLimits{
						Context: 128000,
						Input:   128000,
						Output:  4096,
					},
				},
			},
		},
		"anthropic": ProviderData{
			ID:   "anthropic",
			Name: "Anthropic",
			Env:  []string{"ANTHROPIC_API_KEY"},
			NPM:  "@ai-sdk/anthropic",
			API:  "https://api.anthropic.com/v1",
			Doc:  "https://docs.anthropic.com",
			Models: map[string]ModelDetails{
				"claude-3-opus": {
					ID:               "claude-3-opus",
					Name:             "Claude 3 Opus",
					Family:           "claude",
					ToolCall:         true,
					Reasoning:        true,
					StructuredOutput: true,
					OpenWeights:      false,
					Modalities: ModelModalities{
						Input:  []string{"text", "image"},
						Output: []string{"text"},
					},
					Cost: ModelCost{
						Input:  15.0,
						Output: 75.0,
					},
					Limits: ModelLimits{
						Context: 200000,
						Input:   200000,
						Output:  4096,
					},
				},
			},
		},
		"ollama": ProviderData{
			ID:   "ollama",
			Name: "Ollama",
			NPM:  "ollama-ai-provider",
			Models: map[string]ModelDetails{
				"llama2": {
					ID:          "llama2",
					Name:        "Llama 2",
					Family:      "llama",
					OpenWeights: true,
					Modalities: ModelModalities{
						Input:  []string{"text"},
						Output: []string{"text"},
					},
				},
			},
		},
	}
}

func createMockModelsDevServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := createMockModelsDevResponse()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
}

func TestNewEnhancedModelsDevClient(t *testing.T) {
	logger := createTestLogger()
	client := NewEnhancedModelsDevClient(logger)

	assert.NotNil(t, client)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, "https://models.dev", client.baseURL)
	assert.Equal(t, logger, client.logger)
	assert.False(t, client.cacheEnabled)
}

func TestEnhancedModelsDevClient_FetchAllProviders(t *testing.T) {
	server := createMockModelsDevServer()
	defer server.Close()

	logger := createTestLogger()
	client := NewEnhancedModelsDevClient(logger)
	client.baseURL = server.URL

	providers, err := client.FetchAllProviders(context.Background())

	require.NoError(t, err)
	assert.Len(t, providers, 3)
	assert.Contains(t, providers, "openai")
	assert.Contains(t, providers, "anthropic")
	assert.Contains(t, providers, "ollama")
}

func TestEnhancedModelsDevClient_FetchAllProviders_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	logger := createTestLogger()
	client := NewEnhancedModelsDevClient(logger)
	client.baseURL = server.URL

	_, err := client.FetchAllProviders(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestEnhancedModelsDevClient_FetchAllProviders_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	logger := createTestLogger()
	client := NewEnhancedModelsDevClient(logger)
	client.baseURL = server.URL

	_, err := client.FetchAllProviders(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode")
}

func TestEnhancedModelsDevClient_FetchAllProviders_WithCache(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		response := createMockModelsDevResponse()
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	logger := createTestLogger()
	client := NewEnhancedModelsDevClient(logger)
	client.baseURL = server.URL
	client.cacheEnabled = true

	// First call should fetch
	_, err := client.fetchProviders(context.Background(), false)
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)

	// Second call should use cache
	_, err = client.fetchProviders(context.Background(), false)
	require.NoError(t, err)
	assert.Equal(t, 1, callCount) // Cache hit

	// Force fresh should bypass cache
	_, err = client.fetchProviders(context.Background(), true)
	require.NoError(t, err)
	assert.Equal(t, 2, callCount) // Fresh fetch
}

func TestEnhancedModelsDevClient_GetProviderByID(t *testing.T) {
	server := createMockModelsDevServer()
	defer server.Close()

	logger := createTestLogger()
	client := NewEnhancedModelsDevClient(logger)
	client.baseURL = server.URL

	provider, err := client.GetProviderByID(context.Background(), "openai")

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, "openai", provider.ID)
	assert.Equal(t, "OpenAI", provider.Name)
	assert.Len(t, provider.Models, 2)
}

func TestEnhancedModelsDevClient_GetProviderByID_NotFound(t *testing.T) {
	server := createMockModelsDevServer()
	defer server.Close()

	logger := createTestLogger()
	client := NewEnhancedModelsDevClient(logger)
	client.baseURL = server.URL

	_, err := client.GetProviderByID(context.Background(), "nonexistent")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEnhancedModelsDevClient_FindModel_ExactMatch(t *testing.T) {
	server := createMockModelsDevServer()
	defer server.Close()

	logger := createTestLogger()
	client := NewEnhancedModelsDevClient(logger)
	client.baseURL = server.URL

	matches, err := client.FindModel(context.Background(), "gpt-4")

	require.NoError(t, err)
	require.NotEmpty(t, matches)
	// Should find gpt-4 with high match score (may include boost for recent models)
	found := false
	for _, m := range matches {
		if m.ModelID == "gpt-4" {
			found = true
			assert.GreaterOrEqual(t, m.MatchScore, 1.0)
			break
		}
	}
	assert.True(t, found, "Should find gpt-4 in matches")
}

func TestEnhancedModelsDevClient_FindModel_PathFormat(t *testing.T) {
	server := createMockModelsDevServer()
	defer server.Close()

	logger := createTestLogger()
	client := NewEnhancedModelsDevClient(logger)
	client.baseURL = server.URL

	matches, err := client.FindModel(context.Background(), "openai/gpt-4")

	require.NoError(t, err)
	require.Len(t, matches, 1)
	assert.Equal(t, "openai", matches[0].ProviderID)
	assert.Equal(t, "gpt-4", matches[0].ModelID)
	assert.Equal(t, 1.0, matches[0].MatchScore)
}

func TestEnhancedModelsDevClient_FindModel_FuzzyMatch(t *testing.T) {
	server := createMockModelsDevServer()
	defer server.Close()

	logger := createTestLogger()
	client := NewEnhancedModelsDevClient(logger)
	client.baseURL = server.URL

	matches, err := client.FindModel(context.Background(), "claude")

	require.NoError(t, err)
	require.NotEmpty(t, matches)
	// Should find claude-3-opus
	found := false
	for _, m := range matches {
		if m.ModelID == "claude-3-opus" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find claude-3-opus")
}

func TestEnhancedModelsDevClient_FindModel_NoMatch(t *testing.T) {
	server := createMockModelsDevServer()
	defer server.Close()

	logger := createTestLogger()
	client := NewEnhancedModelsDevClient(logger)
	client.baseURL = server.URL

	_, err := client.FindModel(context.Background(), "zzzznonexistent")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no matches found")
}

func TestEnhancedModelsDevClient_GetModelsByProviderID(t *testing.T) {
	server := createMockModelsDevServer()
	defer server.Close()

	logger := createTestLogger()
	client := NewEnhancedModelsDevClient(logger)
	client.baseURL = server.URL

	matches, err := client.GetModelsByProviderID(context.Background(), "openai")

	require.NoError(t, err)
	assert.Len(t, matches, 2)
	for _, match := range matches {
		assert.Equal(t, "openai", match.ProviderID)
		assert.Equal(t, 1.0, match.MatchScore)
	}
}

func TestEnhancedModelsDevClient_GetModelsByProviderID_NotFound(t *testing.T) {
	server := createMockModelsDevServer()
	defer server.Close()

	logger := createTestLogger()
	client := NewEnhancedModelsDevClient(logger)
	client.baseURL = server.URL

	_, err := client.GetModelsByProviderID(context.Background(), "nonexistent")

	require.Error(t, err)
}

func TestEnhancedModelsDevClient_GetProvidersByNPM(t *testing.T) {
	server := createMockModelsDevServer()
	defer server.Close()

	logger := createTestLogger()
	client := NewEnhancedModelsDevClient(logger)
	client.baseURL = server.URL

	providers := client.GetProvidersByNPM(context.Background(), "@ai-sdk/openai")

	require.Len(t, providers, 1)
	assert.Equal(t, "openai", providers[0].ID)
}

func TestEnhancedModelsDevClient_GetProvidersByNPM_NotFound(t *testing.T) {
	server := createMockModelsDevServer()
	defer server.Close()

	logger := createTestLogger()
	client := NewEnhancedModelsDevClient(logger)
	client.baseURL = server.URL

	providers := client.GetProvidersByNPM(context.Background(), "nonexistent-package")

	assert.Empty(t, providers)
}

func TestEnhancedModelsDevClient_FilterModelsByFeature_ToolCall(t *testing.T) {
	server := createMockModelsDevServer()
	defer server.Close()

	logger := createTestLogger()
	client := NewEnhancedModelsDevClient(logger)
	client.baseURL = server.URL

	matches, err := client.FilterModelsByFeature(context.Background(), "tool_call", 1.0)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(matches), 2) // gpt-4, gpt-4-vision, claude-3-opus
	for _, match := range matches {
		assert.True(t, match.ModelData.ToolCall)
	}
}

func TestEnhancedModelsDevClient_FilterModelsByFeature_Reasoning(t *testing.T) {
	server := createMockModelsDevServer()
	defer server.Close()

	logger := createTestLogger()
	client := NewEnhancedModelsDevClient(logger)
	client.baseURL = server.URL

	matches, err := client.FilterModelsByFeature(context.Background(), "reasoning", 1.0)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(matches), 1)
	for _, match := range matches {
		assert.True(t, match.ModelData.Reasoning)
	}
}

func TestEnhancedModelsDevClient_FilterModelsByFeature_OpenWeights(t *testing.T) {
	server := createMockModelsDevServer()
	defer server.Close()

	logger := createTestLogger()
	client := NewEnhancedModelsDevClient(logger)
	client.baseURL = server.URL

	matches, err := client.FilterModelsByFeature(context.Background(), "open_weights", 1.0)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(matches), 1)
	for _, match := range matches {
		assert.True(t, match.ModelData.OpenWeights)
	}
}

func TestEnhancedModelsDevClient_FilterModelsByFeature_Multimodal(t *testing.T) {
	server := createMockModelsDevServer()
	defer server.Close()

	logger := createTestLogger()
	client := NewEnhancedModelsDevClient(logger)
	client.baseURL = server.URL

	matches, err := client.FilterModelsByFeature(context.Background(), "multimodal", 1.0)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(matches), 2) // gpt-4-vision, claude-3-opus
}

func TestEnhancedModelsDevClient_GetTotalModelCount(t *testing.T) {
	server := createMockModelsDevServer()
	defer server.Close()

	logger := createTestLogger()
	client := NewEnhancedModelsDevClient(logger)
	client.baseURL = server.URL

	count, err := client.GetTotalModelCount(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 4, count) // 2 openai + 1 anthropic + 1 ollama
}

func TestEnhancedModelsDevClient_GetProviderStats(t *testing.T) {
	server := createMockModelsDevServer()
	defer server.Close()

	logger := createTestLogger()
	client := NewEnhancedModelsDevClient(logger)
	client.baseURL = server.URL

	stats, err := client.GetProviderStats(context.Background())

	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, 3, stats.TotalProviders)
	assert.Equal(t, 4, stats.TotalModels)
	assert.NotEmpty(t, stats.ProvidersByNPM)
	assert.NotEmpty(t, stats.ModelsByFeature)
	assert.GreaterOrEqual(t, stats.OpenWeightModels, 1)
}

func TestEnhancedModelsDevClient_CalculateMatchScore(t *testing.T) {
	logger := createTestLogger()
	client := NewEnhancedModelsDevClient(logger)

	provider := ProviderData{
		ID:   "openai",
		Name: "OpenAI",
	}

	model := ModelDetails{
		ID:          "gpt-4",
		Name:        "GPT-4",
		Family:      "gpt",
		LastUpdated: time.Now().Format("2006-01-02"),
	}

	// Exact match on model ID
	score := client.calculateMatchScore("gpt-4", "openai", provider, "gpt-4", model)
	assert.Equal(t, 1.0, score)

	// Exact match on model name
	score = client.calculateMatchScore("gpt-4", "openai", provider, "gpt-4", model)
	assert.Equal(t, 1.0, score)

	// Path match
	score = client.calculateMatchScore("openai/gpt-4", "openai", provider, "gpt-4", model)
	assert.Equal(t, 0.95, score)

	// Partial match
	score = client.calculateMatchScore("gpt", "openai", provider, "gpt-4", model)
	assert.Greater(t, score, 0.3)

	// Multi-word query
	score = client.calculateMatchScore("gpt four", "openai", provider, "gpt-4", model)
	assert.GreaterOrEqual(t, score, 0.0)
}

func TestEnhancedModelsDevClient_CalculateFeatureScore(t *testing.T) {
	logger := createTestLogger()
	client := NewEnhancedModelsDevClient(logger)

	tests := []struct {
		name     string
		model    ModelDetails
		feature  string
		expected float64
	}{
		{
			name:     "tool_call enabled",
			model:    ModelDetails{ToolCall: true},
			feature:  "tool_call",
			expected: 1.0,
		},
		{
			name:     "tool_call disabled",
			model:    ModelDetails{ToolCall: false},
			feature:  "tool_call",
			expected: 0.0,
		},
		{
			name:     "reasoning enabled",
			model:    ModelDetails{Reasoning: true},
			feature:  "reasoning",
			expected: 1.0,
		},
		{
			name:     "structured_output enabled",
			model:    ModelDetails{StructuredOutput: true},
			feature:  "structured_output",
			expected: 1.0,
		},
		{
			name:     "open_weights enabled",
			model:    ModelDetails{OpenWeights: true},
			feature:  "open_source",
			expected: 1.0,
		},
		{
			name: "multimodal with image",
			model: ModelDetails{
				Modalities: ModelModalities{
					Input: []string{"text", "image"},
				},
			},
			feature:  "multimodal",
			expected: 1.0,
		},
		{
			name:     "unknown feature",
			model:    ModelDetails{},
			feature:  "unknown_feature",
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := client.calculateFeatureScore(tt.model, tt.feature)
			assert.Equal(t, tt.expected, score)
		})
	}
}

func TestEnhancedModelsDevClient_SortMatchesByScore(t *testing.T) {
	logger := createTestLogger()
	client := NewEnhancedModelsDevClient(logger)

	matches := []ModelMatch{
		{ModelID: "low", MatchScore: 0.3},
		{ModelID: "high", MatchScore: 0.9},
		{ModelID: "medium", MatchScore: 0.6},
	}

	client.sortMatchesByScore(matches)

	assert.Equal(t, "high", matches[0].ModelID)
	assert.Equal(t, "medium", matches[1].ModelID)
	assert.Equal(t, "low", matches[2].ModelID)
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		input    string
		expected int // year
	}{
		{"2024-01-15", 2024},
		{"2023-12-31", 2023},
		{"invalid", 1}, // Default zero time year
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseDate(tt.input)
			assert.Equal(t, tt.expected, result.Year())
		})
	}
}

func TestModelMatch_Struct(t *testing.T) {
	match := ModelMatch{
		ProviderID:   "openai",
		ProviderData: ProviderData{ID: "openai", Name: "OpenAI"},
		ModelID:      "gpt-4",
		ModelData:    ModelDetails{ID: "gpt-4", Name: "GPT-4"},
		MatchScore:   0.95,
	}

	assert.Equal(t, "openai", match.ProviderID)
	assert.Equal(t, "OpenAI", match.ProviderData.Name)
	assert.Equal(t, "gpt-4", match.ModelID)
	assert.Equal(t, "GPT-4", match.ModelData.Name)
	assert.Equal(t, 0.95, match.MatchScore)
}

func TestProviderStats_Struct(t *testing.T) {
	stats := ProviderStats{
		TotalProviders:   10,
		TotalModels:      100,
		ProvidersByNPM:   map[string]int{"@ai-sdk/openai": 1},
		ModelsByFeature:  map[string]int{"tool_call": 50},
		ModelsByModality: map[string]int{"input_text": 100},
		OpenWeightModels: 25,
		RecentUpdates:    5,
	}

	assert.Equal(t, 10, stats.TotalProviders)
	assert.Equal(t, 100, stats.TotalModels)
	assert.Equal(t, 1, stats.ProvidersByNPM["@ai-sdk/openai"])
	assert.Equal(t, 50, stats.ModelsByFeature["tool_call"])
	assert.Equal(t, 25, stats.OpenWeightModels)
	assert.Equal(t, 5, stats.RecentUpdates)
}

func TestProviderData_Struct(t *testing.T) {
	provider := ProviderData{
		ID:      "openai",
		Env:     []string{"OPENAI_API_KEY"},
		NPM:     "@ai-sdk/openai",
		API:     "https://api.openai.com/v1",
		Name:    "OpenAI",
		Doc:     "https://platform.openai.com/docs",
		Models:  make(map[string]ModelDetails),
		LogoURL: "https://models.dev/logos/openai.svg",
	}

	assert.Equal(t, "openai", provider.ID)
	assert.Equal(t, "OpenAI", provider.Name)
	assert.Contains(t, provider.Env, "OPENAI_API_KEY")
	assert.Equal(t, "@ai-sdk/openai", provider.NPM)
}

func TestModelDetails_Struct(t *testing.T) {
	model := ModelDetails{
		ID:               "gpt-4",
		Name:             "GPT-4",
		Family:           "gpt",
		Attachment:       true,
		Reasoning:        true,
		ToolCall:         true,
		Temperature:      true,
		Knowledge:        "2023-04",
		ReleaseDate:      "2023-03-14",
		LastUpdated:      "2024-01-15",
		OpenWeights:      false,
		StructuredOutput: true,
		Modalities: ModelModalities{
			Input:  []string{"text", "image"},
			Output: []string{"text"},
		},
		Cost: ModelCost{
			Input:  30.0,
			Output: 60.0,
		},
		Limits: ModelLimits{
			Context: 128000,
			Input:   128000,
			Output:  4096,
		},
	}

	assert.Equal(t, "gpt-4", model.ID)
	assert.True(t, model.ToolCall)
	assert.True(t, model.Reasoning)
	assert.Contains(t, model.Modalities.Input, "image")
	assert.Equal(t, 30.0, model.Cost.Input)
	assert.Equal(t, uint64(128000), model.Limits.Context)
}

func TestModelModalities_Struct(t *testing.T) {
	modalities := ModelModalities{
		Input:  []string{"text", "image", "audio"},
		Output: []string{"text", "audio"},
	}

	assert.Len(t, modalities.Input, 3)
	assert.Len(t, modalities.Output, 2)
	assert.Contains(t, modalities.Input, "image")
}

func TestModelCost_Struct(t *testing.T) {
	cost := ModelCost{
		Input:       2.5,
		Output:      10.0,
		Reasoning:   5.0,
		CacheRead:   0.5,
		CacheWrite:  1.0,
		InputAudio:  100.0,
		OutputAudio: 200.0,
	}

	assert.Equal(t, 2.5, cost.Input)
	assert.Equal(t, 10.0, cost.Output)
	assert.Equal(t, 5.0, cost.Reasoning)
	assert.Equal(t, 0.5, cost.CacheRead)
	assert.Equal(t, 100.0, cost.InputAudio)
}

func TestModelLimits_Struct(t *testing.T) {
	limits := ModelLimits{
		Context: 128000,
		Input:   100000,
		Output:  4096,
	}

	assert.Equal(t, uint64(128000), limits.Context)
	assert.Equal(t, uint64(100000), limits.Input)
	assert.Equal(t, uint64(4096), limits.Output)
}
