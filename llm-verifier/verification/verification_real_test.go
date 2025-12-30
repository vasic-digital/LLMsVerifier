package verification

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createMockModelsDevAPIResponse() ModelsDevResponse {
	return ModelsDevResponse{
		Models: []ModelsDevModel{
			{
				Provider:         "OpenAI",
				Model:            "GPT-4",
				Family:           "gpt",
				ProviderID:       "openai",
				ModelID:          "gpt-4",
				ToolCall:         true,
				Reasoning:        false,
				StructuredOutput: true,
				ContextLimit:     8192,
				InputLimit:       8192,
				OutputLimit:      4096,
				InputCostPer1M:   30.0,
				OutputCostPer1M:  60.0,
				ReleaseDate:      "2023-03-14",
				LastUpdated:      "2024-01-15",
				APIEndpoint:      "https://api.openai.com/v1/chat/completions",
			},
			{
				Provider:         "OpenAI",
				Model:            "GPT-3.5 Turbo",
				Family:           "gpt",
				ProviderID:       "openai",
				ModelID:          "gpt-3.5-turbo",
				ToolCall:         true,
				Reasoning:        false,
				StructuredOutput: true,
				ContextLimit:     16384,
				InputLimit:       16384,
				OutputLimit:      4096,
				InputCostPer1M:   0.5,
				OutputCostPer1M:  1.5,
				ReleaseDate:      "2023-03-01",
				LastUpdated:      "2024-01-10",
				APIEndpoint:      "https://api.openai.com/v1/chat/completions",
			},
			{
				Provider:         "Anthropic",
				Model:            "Claude 3 Opus",
				Family:           "claude",
				ProviderID:       "anthropic",
				ModelID:          "claude-3-opus-20240229",
				ToolCall:         true,
				Reasoning:        true,
				StructuredOutput: true,
				ContextLimit:     200000,
				InputLimit:       200000,
				OutputLimit:      4096,
				InputCostPer1M:   15.0,
				OutputCostPer1M:  75.0,
				ReleaseDate:      "2024-02-29",
				LastUpdated:      "2024-02-29",
				APIEndpoint:      "https://api.anthropic.com/v1/messages",
			},
		},
	}
}

func createMockModelsDevAPIServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify no-cache headers were sent
		assert.Equal(nil, "no-cache, no-store, must-revalidate", r.Header.Get("Cache-Control"))

		response := createMockModelsDevAPIResponse()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
}

func TestNewModelsDevClient(t *testing.T) {
	client := NewModelsDevClient()

	assert.NotNil(t, client)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, "https://models.dev/api", client.baseURL)
}

func TestModelsDevClient_FetchModels_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The endpoint is baseURL + ".json" so path should end with .json
		assert.Contains(t, r.URL.Path, ".json")
		response := createMockModelsDevAPIResponse()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewModelsDevClient()
	// The client appends ".json" to baseURL, so we set baseURL to server.URL + "/api"
	// which becomes server.URL + "/api.json" when fetching
	client.baseURL = server.URL + "/api"

	models, err := client.FetchModels(context.Background())

	require.NoError(t, err)
	assert.Len(t, models, 3)
	assert.Equal(t, "gpt-4", models[0].ModelID)
	assert.Equal(t, "gpt-3.5-turbo", models[1].ModelID)
	assert.Equal(t, "claude-3-opus-20240229", models[2].ModelID)
}

func TestModelsDevClient_FetchModels_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewModelsDevClient()
	client.baseURL = server.URL + "/api"

	_, err := client.FetchModels(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestModelsDevClient_FetchModels_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewModelsDevClient()
	client.baseURL = server.URL + "/api"

	_, err := client.FetchModels(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode")
}

func TestModelsDevClient_FindModel_ExactMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := createMockModelsDevAPIResponse()
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewModelsDevClient()
	client.baseURL = server.URL + "/api"

	model, err := client.FindModel(context.Background(), "gpt-4")

	require.NoError(t, err)
	require.NotNil(t, model)
	assert.Equal(t, "gpt-4", model.ModelID)
	assert.Equal(t, "GPT-4", model.Model)
	assert.Equal(t, "openai", model.ProviderID)
}

func TestModelsDevClient_FindModel_CaseInsensitive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := createMockModelsDevAPIResponse()
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewModelsDevClient()
	client.baseURL = server.URL + "/api"

	model, err := client.FindModel(context.Background(), "GPT-4")

	require.NoError(t, err)
	require.NotNil(t, model)
	assert.Equal(t, "gpt-4", model.ModelID)
}

func TestModelsDevClient_FindModel_FuzzyMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := createMockModelsDevAPIResponse()
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewModelsDevClient()
	client.baseURL = server.URL + "/api"

	model, err := client.FindModel(context.Background(), "claude-3")

	require.NoError(t, err)
	require.NotNil(t, model)
	assert.Contains(t, model.ModelID, "claude-3")
}

func TestModelsDevClient_FindModel_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := createMockModelsDevAPIResponse()
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewModelsDevClient()
	client.baseURL = server.URL + "/api"

	_, err := client.FindModel(context.Background(), "nonexistent-model-xyz")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestModelsDevClient_FindModel_ByProviderID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := createMockModelsDevAPIResponse()
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewModelsDevClient()
	client.baseURL = server.URL + "/api"

	model, err := client.FindModel(context.Background(), "openai")

	require.NoError(t, err)
	require.NotNil(t, model)
	assert.Equal(t, "openai", model.ProviderID)
}

func TestModelsDevClient_GetModelsByProvider_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := createMockModelsDevAPIResponse()
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewModelsDevClient()
	client.baseURL = server.URL + "/api"

	models, err := client.GetModelsByProvider(context.Background(), "openai")

	require.NoError(t, err)
	assert.Len(t, models, 2) // gpt-4 and gpt-3.5-turbo
	for _, model := range models {
		assert.Equal(t, "openai", model.ProviderID)
	}
}

func TestModelsDevClient_GetModelsByProvider_CaseInsensitive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := createMockModelsDevAPIResponse()
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewModelsDevClient()
	client.baseURL = server.URL + "/api"

	models, err := client.GetModelsByProvider(context.Background(), "OPENAI")

	require.NoError(t, err)
	assert.Len(t, models, 2)
}

func TestModelsDevClient_GetModelsByProvider_ByProviderName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := createMockModelsDevAPIResponse()
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewModelsDevClient()
	client.baseURL = server.URL + "/api"

	models, err := client.GetModelsByProvider(context.Background(), "Anthropic")

	require.NoError(t, err)
	assert.Len(t, models, 1)
	assert.Equal(t, "claude-3-opus-20240229", models[0].ModelID)
}

func TestModelsDevClient_GetModelsByProvider_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := createMockModelsDevAPIResponse()
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewModelsDevClient()
	client.baseURL = server.URL + "/api"

	_, err := client.GetModelsByProvider(context.Background(), "nonexistent-provider")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no models found")
}

func TestModelsDevModel_Struct(t *testing.T) {
	model := ModelsDevModel{
		Provider:         "OpenAI",
		Model:            "GPT-4",
		Family:           "gpt",
		ProviderID:       "openai",
		ModelID:          "gpt-4",
		ToolCall:         true,
		Reasoning:        true,
		StructuredOutput: true,
		ContextLimit:     8192,
		InputLimit:       8192,
		OutputLimit:      4096,
		InputCostPer1M:   30.0,
		OutputCostPer1M:  60.0,
		ReleaseDate:      "2023-03-14",
		LastUpdated:      "2024-01-15",
		APIEndpoint:      "https://api.openai.com/v1/chat/completions",
	}

	assert.Equal(t, "OpenAI", model.Provider)
	assert.Equal(t, "GPT-4", model.Model)
	assert.Equal(t, "gpt", model.Family)
	assert.Equal(t, "openai", model.ProviderID)
	assert.Equal(t, "gpt-4", model.ModelID)
	assert.True(t, model.ToolCall)
	assert.True(t, model.Reasoning)
	assert.True(t, model.StructuredOutput)
	assert.Equal(t, 8192, model.ContextLimit)
	assert.Equal(t, 4096, model.OutputLimit)
	assert.Equal(t, 30.0, model.InputCostPer1M)
	assert.Equal(t, 60.0, model.OutputCostPer1M)
}

func TestModelsDevResponse_Struct(t *testing.T) {
	response := ModelsDevResponse{
		Models: []ModelsDevModel{
			{ModelID: "gpt-4"},
			{ModelID: "gpt-3.5-turbo"},
		},
	}

	assert.Len(t, response.Models, 2)
	assert.Equal(t, "gpt-4", response.Models[0].ModelID)
	assert.Equal(t, "gpt-3.5-turbo", response.Models[1].ModelID)
}
