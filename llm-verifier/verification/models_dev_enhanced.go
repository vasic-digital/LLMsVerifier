package verification

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"llm-verifier/logging"
)

// EnhancedModelsDevClient is a production-ready client for models.dev API
type EnhancedModelsDevClient struct {
	httpClient     *http.Client
	baseURL        string
	logger         *logging.Logger
	cacheEnabled   bool
	lastFetchTime  time.Time
	cachedData     *ModelsDevEnhancedResponse
}

// ModelsDevEnhancedResponse represents the full models.dev API structure
type ModelsDevEnhancedResponse map[string]ProviderData

// ProviderData contains provider information and their models
type ProviderData struct {
	ID             string                  `json:"id"`
	Env            []string                `json:"env"`
	NPM            string                  `json:"npm"`
	API            string                  `json:"api,omitempty"`
	Name           string                  `json:"name"`
	Doc            string                  `json:"doc"`
	Models         map[string]ModelDetails `json:"models"`
	LogoURL        string                  `json:"-"` // Computed field
}

// ModelDetails contains comprehensive model information
type ModelDetails struct {
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	Family           string              `json:"family,omitempty"`
	Attachment       bool                `json:"attachment"`
	Reasoning        bool                `json:"reasoning"`
	ToolCall         bool                `json:"tool_call"`
	Temperature      bool                `json:"temperature"`
	Knowledge        string              `json:"knowledge,omitempty"`
	ReleaseDate      string              `json:"release_date"`
	LastUpdated      string              `json:"last_updated"`
	Modalities       ModelModalities     `json:"modalities"`
	OpenWeights      bool                `json:"open_weights"`
	Cost             ModelCost           `json:"cost"`
	Limits           ModelLimits         `json:"limit"`
	StructuredOutput bool                `json:"structured_output,omitempty"`
	Status           string              `json:"status,omitempty"`
	Interleaved      *InterleavedConfig  `json:"interleaved,omitempty"`
}

// ModelModalities defines input/output modalities
type ModelModalities struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

// ModelCost contains pricing information
type ModelCost struct {
	Input              float64 `json:"input"`               // Cost per 1M input tokens (USD)
	Output             float64 `json:"output"`              // Cost per 1M output tokens (USD)
	Reasoning          float64 `json:"reasoning,omitempty"` // Cost per 1M reasoning tokens (USD)
	CacheRead          float64 `json:"cache_read,omitempty"` // Cost per 1M cached read tokens (USD)
	CacheWrite         float64 `json:"cache_write,omitempty"` // Cost per 1M cached write tokens (USD)
	InputAudio         float64 `json:"input_audio,omitempty"` // Cost per 1M audio input tokens (USD)
	OutputAudio        float64 `json:"output_audio,omitempty"` // Cost per 1M audio output tokens (USD)
}

// ModelLimits contains token limit information
type ModelLimits struct {
	Context uint64 `json:"context"` // Maximum context window in tokens
	Input   uint64 `json:"input"`   // Maximum input tokens
	Output  uint64 `json:"output"`  // Maximum output tokens
}

// InterleavedConfig defines interleaved reasoning configuration
type InterleavedConfig struct {
	Field string `json:"field,omitempty"` // "reasoning_content" or "reasoning_details"
}

// NewEnhancedModelsDevClient creates a new enhanced client
func NewEnhancedModelsDevClient(logger *logging.Logger) *EnhancedModelsDevClient {
	return &EnhancedModelsDevClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		baseURL:      "https://models.dev",
		logger:       logger,
		cacheEnabled: false, // Explicitly disabled for clean calls
	}
}

// FetchAllProviders fetches all provider data from models.dev (no caching)
func (c *EnhancedModelsDevClient) FetchAllProviders(ctx context.Context) (ModelsDevEnhancedResponse, error) {
	// Force fresh fetch by ignoring cache
	return c.fetchProviders(ctx, true)
}

// fetchProviders is the internal fetch implementation
func (c *EnhancedModelsDevClient) fetchProviders(ctx context.Context, forceFresh bool) (ModelsDevEnhancedResponse, error) {
	// Check cache only if enabled and not forced fresh
	if c.cacheEnabled && !forceFresh && c.cachedData != nil {
		if time.Since(c.lastFetchTime) < 5*time.Minute {
			c.logger.Info("Using cached models.dev data")
			return *c.cachedData, nil
		}
	}

	c.logger.Info("Fetching fresh data from models.dev API")

	// Create request with no-cache headers
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api.json", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers to ensure fresh data
	req.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Expires", "0")
	req.Header.Set("User-Agent", "LLM-Verifier-Enhanced/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from models.dev: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("models.dev API returned status %d", resp.StatusCode)
	}

	var response ModelsDevEnhancedResponse
	decoder := json.NewDecoder(resp.Body)
	decoder.DisallowUnknownFields() // Strict parsing

	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Enrich with logo URLs
	for providerID, provider := range response {
		provider.LogoURL = fmt.Sprintf("%s/logos/%s.svg", c.baseURL, providerID)
		response[providerID] = provider
	}

	// Update cache if enabled
	if c.cacheEnabled {
		c.cachedData = &response
		c.lastFetchTime = time.Now()
	}

	c.logger.Infof("Successfully fetched %d providers from models.dev", len(response))
	return response, nil
}

// GetProviderByID retrieves a specific provider by ID
func (c *EnhancedModelsDevClient) GetProviderByID(ctx context.Context, providerID string) (*ProviderData, error) {
	providers, err := c.FetchAllProviders(ctx)
	if err != nil {
		return nil, err
	}

	provider, exists := providers[providerID]
	if !exists {
		return nil, fmt.Errorf("provider %s not found in models.dev", providerID)
	}

	return &provider, nil
}

// FindModel searches for a model across all providers
// It tries multiple matching strategies in order:
// 1. Exact match on model ID
// 2. Provider/model path match (e.g., "openai/gpt-4")
// 3. Fuzzy match on model name
func (c *EnhancedModelsDevClient) FindModel(ctx context.Context, searchQuery string) ([]ModelMatch, error) {
	providers, err := c.FetchAllProviders(ctx)
	if err != nil {
		return nil, err
	}

	searchQuery = strings.TrimSpace(strings.ToLower(searchQuery))
	var matches []ModelMatch
	matchMap := make(map[string]bool) // Track unique matches

	// Strategy 1: Check for provider/model path format
	if strings.Contains(searchQuery, "/") {
		parts := strings.SplitN(searchQuery, "/", 2)
		providerID := parts[0]
		modelID := parts[1]

		if provider, exists := providers[providerID]; exists {
			if model, modelExists := provider.Models[modelID]; modelExists {
				key := fmt.Sprintf("%s/%s", providerID, modelID)
				if !matchMap[key] {
					matches = append(matches, ModelMatch{
						ProviderID:   providerID,
						ProviderData: provider,
						ModelID:      modelID,
						ModelData:    model,
						MatchScore:   1.0, // Perfect match
					})
					matchMap[key] = true
				}
				return matches, nil
			}
		}
	}

	// Strategy 2: Search across all providers
	for providerID, provider := range providers {
		for modelID, model := range provider.Models {
			score := c.calculateMatchScore(searchQuery, providerID, provider, modelID, model)

			if score > 0.3 { // Minimum threshold
				key := fmt.Sprintf("%s/%s", providerID, modelID)
				if !matchMap[key] {
					matches = append(matches, ModelMatch{
						ProviderID:   providerID,
						ProviderData: provider,
						ModelID:      modelID,
						ModelData:    model,
						MatchScore:   score,
					})
					matchMap[key] = true
				}
			}
		}
	}

	// Sort by match score (descending)
	c.sortMatchesByScore(matches)

	if len(matches) == 0 {
		return nil, fmt.Errorf("no matches found for query: %s", searchQuery)
	}

	return matches, nil
}

// ModelMatch represents a found model with match metadata
type ModelMatch struct {
	ProviderID   string
	ProviderData ProviderData
	ModelID      string
	ModelData    ModelDetails
	MatchScore   float64
}

// calculateMatchScore calculates how well a model matches the search query
func (c *EnhancedModelsDevClient) calculateMatchScore(query, providerID string, provider ProviderData, modelID string, model ModelDetails) float64 {
	query = strings.ToLower(query)
	modelIDLower := strings.ToLower(modelID)
	modelNameLower := strings.ToLower(model.Name)
	providerIDLower := strings.ToLower(providerID)
	familyLower := strings.ToLower(model.Family)

	// Perfect matches
	if modelIDLower == query || modelNameLower == query {
		return 1.0
	}

	// Provider/Model path match
	pathMatch := fmt.Sprintf("%s/%s", providerIDLower, modelIDLower)
	if pathMatch == query {
		return 0.95
	}

	score := 0.0

	// Model ID contains query
	if strings.Contains(modelIDLower, query) {
		score += 0.6
	}

	// Model name contains query
	if strings.Contains(modelNameLower, query) {
		score += 0.5
	}

	// Family match
	if strings.Contains(familyLower, query) {
		score += 0.4
	}

	// Provider match (reduced weight)
	if strings.Contains(providerIDLower, query) {
		score += 0.2
	}

	// Token-based matching for multi-word queries
	queryTokens := strings.Fields(query)
	if len(queryTokens) > 1 {
		matchedTokens := 0
		for _, token := range queryTokens {
			if strings.Contains(modelIDLower, token) ||
				strings.Contains(modelNameLower, token) ||
				strings.Contains(familyLower, token) {
				matchedTokens++
			}
		}
		score += float64(matchedTokens) / float64(len(queryTokens)) * 0.3
	}

	// Boost score for recent models
	if model.LastUpdated != "" {
		if parseDate(model.LastUpdated).Year() >= time.Now().Year()-1 {
			score += 0.1
		}
	}

	return score
}

// sortMatchesByScore sorts matches by score descending
func (c *EnhancedModelsDevClient) sortMatchesByScore(matches []ModelMatch) {
	for i := 0; i < len(matches)-1; i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].MatchScore > matches[i].MatchScore {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}
}

// GetModelsByProviderID retrieves all models for a specific provider
func (c *EnhancedModelsDevClient) GetModelsByProviderID(ctx context.Context, providerID string) ([]ModelMatch, error) {
	provider, err := c.GetProviderByID(ctx, providerID)
	if err != nil {
		return nil, err
	}

	var matches []ModelMatch
	for modelID, model := range provider.Models {
		matches = append(matches, ModelMatch{
			ProviderID:   providerID,
			ProviderData: *provider,
			ModelID:      modelID,
			ModelData:    model,
			MatchScore:   1.0, // All models from provider are full matches
		})
	}

	return matches, nil
}

// GetProvidersByNPM finds providers by their NPM package
func (c *EnhancedModelsDevClient) GetProvidersByNPM(ctx context.Context, npmPackage string) []ProviderData {
	providers, err := c.FetchAllProviders(ctx)
	if err != nil {
		c.logger.Errorf("Failed to fetch providers: %v", err)
		return nil
	}

	var matches []ProviderData
	npmPackage = strings.ToLower(npmPackage)

	for _, provider := range providers {
		if strings.EqualFold(provider.NPM, npmPackage) {
			matches = append(matches, provider)
		}
	}

	return matches
}

// FilterModelsByFeature returns models that support specific features
func (c *EnhancedModelsDevClient) FilterModelsByFeature(ctx context.Context, feature string, minScore float64) ([]ModelMatch, error) {
	providers, err := c.FetchAllProviders(ctx)
	if err != nil {
		return nil, err
	}

	var matches []ModelMatch
	feature = strings.ToLower(feature)

	for providerID, provider := range providers {
		for modelID, model := range provider.Models {
			score := c.calculateFeatureScore(model, feature)
			if score >= minScore {
				matches = append(matches, ModelMatch{
					ProviderID:   providerID,
					ProviderData: provider,
					ModelID:      modelID,
					ModelData:    model,
					MatchScore:   score,
				})
			}
		}
	}

	c.sortMatchesByScore(matches)
	return matches, nil
}

// calculateFeatureScore calculates how well a model supports a feature
func (c *EnhancedModelsDevClient) calculateFeatureScore(model ModelDetails, feature string) float64 {
	switch feature {
	case "tool_call", "tools", "function_calling":
		if model.ToolCall {
			return 1.0
		}
	case "reasoning", "chain_of_thought":
		if model.Reasoning {
			return 1.0
		}
	case "attachments", "files":
		if model.Attachment {
			return 1.0
		}
	case "structured_output", "json":
		if model.StructuredOutput {
			return 1.0
		}
	case "multimodal", "image_input":
		for _, modality := range model.Modalities.Input {
			if modality != "text" {
				return 1.0
			}
		}
	case "open_weights", "open_source":
		if model.OpenWeights {
			return 1.0
		}
	}

	return 0.0
}

// GetTotalModelCount returns the total number of models across all providers
func (c *EnhancedModelsDevClient) GetTotalModelCount(ctx context.Context) (int, error) {
	providers, err := c.FetchAllProviders(ctx)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, provider := range providers {
		count += len(provider.Models)
	}

	return count, nil
}

// GetProviderStats returns statistics about providers and models
func (c *EnhancedModelsDevClient) GetProviderStats(ctx context.Context) (*ProviderStats, error) {
	providers, err := c.FetchAllProviders(ctx)
	if err != nil {
		return nil, err
	}

	stats := &ProviderStats{
		TotalProviders:      len(providers),
		TotalModels:         0,
		ProvidersByNPM:      make(map[string]int),
		ModelsByFeature:     make(map[string]int),
		ModelsByModality:    make(map[string]int),
		OpenWeightModels:    0,
		RecentUpdates:       0,
	}

	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -7)

	for _, provider := range providers {
		// Count by NPM package
		stats.ProvidersByNPM[provider.NPM]++

		// Process models
		stats.TotalModels += len(provider.Models)

		for _, model := range provider.Models {
			// Feature stats
			if model.ToolCall {
				stats.ModelsByFeature["tool_call"]++
			}
			if model.Reasoning {
				stats.ModelsByFeature["reasoning"]++
			}
			if model.Attachment {
				stats.ModelsByFeature["attachment"]++
			}
			if model.StructuredOutput {
				stats.ModelsByFeature["structured_output"]++
			}
			if model.OpenWeights {
				stats.OpenWeightModels++
			}

			// Modality stats
			for _, modality := range model.Modalities.Input {
				stats.ModelsByModality[fmt.Sprintf("input_%s", modality)]++
			}
			for _, modality := range model.Modalities.Output {
				stats.ModelsByModality[fmt.Sprintf("output_%s", modality)]++
			}

			// Recent updates
			if model.LastUpdated != "" {
				updateDate, err := time.Parse("2006-01-02", model.LastUpdated)
				if err == nil && updateDate.After(sevenDaysAgo) {
					stats.RecentUpdates++
				}
			}
		}
	}

	return stats, nil
}

// ProviderStats contains aggregated statistics
type ProviderStats struct {
	TotalProviders      int
	TotalModels         int
	ProvidersByNPM      map[string]int
	ModelsByFeature     map[string]int
	ModelsByModality    map[string]int
	OpenWeightModels    int
	RecentUpdates       int
}

// Helper function to parse dates
func parseDate(dateStr string) time.Time {
	layout := "2006-01-02"
	t, _ := time.Parse(layout, dateStr)
	return t
}
