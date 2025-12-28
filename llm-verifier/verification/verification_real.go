package verification

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type ModelsDevClient struct {
	httpClient *http.Client
	baseURL    string
}

type ModelsDevModel struct {
	Provider            string `json:"provider"`
	Model               string `json:"model"`
	Family              string `json:"family"`
	ProviderID          string `json:"provider_id"`
	ModelID             string `json:"model_id"`
	ToolCall            bool   `json:"tool_call"`
	Reasoning           bool   `json:"reasoning"`
	StructuredOutput    bool   `json:"structured_output"`
	ContextLimit        int    `json:"context_limit"`
	InputLimit          int    `json:"input_limit"`
	OutputLimit         int    `json:"output_limit"`
	InputCostPer1M      float64 `json:"input_cost_per_1m_tokens"`
	OutputCostPer1M     float64 `json:"output_cost_per_1m_tokens"`
	ReleaseDate         string `json:"release_date"`
	LastUpdated         string `json:"last_updated"`
	APIEndpoint         string `json:"api_endpoint"`
}

type ModelsDevResponse struct {
	Models []ModelsDevModel `json:"models"`
}

func NewModelsDevClient() *ModelsDevClient {
	return &ModelsDevClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				DisableKeepAlives:   false, // Enable keep-alives for efficiency
			},
		},
		baseURL: "https://models.dev/api",
	}
}

// FetchModels makes a fresh API call to models.dev with no caching
func (c *ModelsDevClient) FetchModels(ctx context.Context) ([]ModelsDevModel, error) {
	// Build request with proper headers for fresh data
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+".json", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers to ensure fresh data (no caching)
	req.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Expires", "0")
	req.Header.Set("User-Agent", "LLM-Verifier/1.0")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from models.dev: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("models.dev API returned status %d", resp.StatusCode)
	}
	
	var response ModelsDevResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return response.Models, nil
}

// FindModel searches for a model across all providers
func (c *ModelsDevClient) FindModel(ctx context.Context, modelID string) (*ModelsDevModel, error) {
	models, err := c.FetchModels(ctx)
	if err != nil {
		return nil, err
	}
	
	// Search by exact match on model_id, provider_id, or model name
	modelID = strings.ToLower(modelID)
	for _, model := range models {
		if strings.EqualFold(model.ModelID, modelID) ||
			strings.EqualFold(model.ProviderID, modelID) ||
			strings.EqualFold(model.Model, modelID) {
			return &model, nil
		}
	}
	
	// Try fuzzy matching if no exact match
	for _, model := range models {
		if strings.Contains(strings.ToLower(model.Model), modelID) ||
			strings.Contains(strings.ToLower(model.ModelID), modelID) {
			return &model, nil
		}
	}
	
	return nil, fmt.Errorf("model %s not found in models.dev", modelID)
}

// GetModelsByProvider returns all models for a specific provider
func (c *ModelsDevClient) GetModelsByProvider(ctx context.Context, providerID string) ([]ModelsDevModel, error) {
	allModels, err := c.FetchModels(ctx)
	if err != nil {
		return nil, err
	}
	
	var providerModels []ModelsDevModel
	providerID = strings.ToLower(providerID)
	
	for _, model := range allModels {
		if strings.EqualFold(model.ProviderID, providerID) ||
			strings.EqualFold(model.Provider, providerID) {
			providerModels = append(providerModels, model)
		}
	}
	
	if len(providerModels) == 0 {
		return nil, fmt.Errorf("no models found for provider %s", providerID)
	}
	
	return providerModels, nil
}
