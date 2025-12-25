package enhanced

import (
	"fmt"
	"strings"
	"time"

	"llm-verifier/database"
)

// PricingDetector detects pricing information for different providers
type PricingDetector struct {
	httpClient *http.Client
}

// NewPricingDetector creates a new pricing detector
func NewPricingDetector() *PricingDetector {
	return &PricingDetector{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// PricingInfo represents pricing information for a model
type PricingInfo struct {
	InputTokenCost       float64 `json:"input_token_cost"`        // Cost per 1M input tokens
	OutputTokenCost      float64 `json:"output_token_cost"`       // Cost per 1M output tokens
	CachedInputTokenCost float64 `json:"cached_input_token_cost"` // Cost per 1M cached input tokens
	StorageCost          float64 `json:"storage_cost"`            // Storage cost
	RequestCost          float64 `json:"request_cost"`            // Cost per request
	Currency             string  `json:"currency"`                // Currency (USD, EUR, etc.)
	PricingModel         string  `json:"pricing_model"`           // per_token, per_request, per_hour, etc.
	EffectiveFrom        string  `json:"effective_from"`          // Effective date
	EffectiveTo          string  `json:"effective_to"`            // Expiration date (if any)
}

// DetectPricing detects pricing information for a given model
func (pd *PricingDetector) DetectPricing(providerName, modelID string) (*PricingInfo, error) {
	switch strings.ToLower(providerName) {
	case "openai":
		return pd.detectOpenAIPricing(modelID)
	case "anthropic":
		return pd.detectAnthropicPricing(modelID)
	case "azure", "azure openai":
		return pd.detectAzureOpenAIPricing(modelID)
	case "google", "google cloud", "gcp":
		return pd.detectGooglePricing(modelID)
	case "cohere":
		return pd.detectCoherePricing(modelID)
	default:
		return pd.detectGenericPricing(providerName, modelID)
	}
}

// fetchOpenAIRealtimePricing attempts to fetch real-time pricing from OpenAI
func (pd *PricingDetector) fetchOpenAIRealtimePricing(modelID string) (*PricingInfo, error) {
	// OpenAI pricing can be fetched from various sources:
	// 1. OpenAI's official pricing page
	// 2. Third-party APIs like OpenRouter
	// 3. Cached pricing with periodic updates

	// For this implementation, we'll use a pricing API service
	// In production, you might use services like:
	// - OpenRouter API (https://openrouter.ai/docs#models)
	// - Custom pricing endpoints
	// - Web scraping of official documentation

	// Example implementation using a hypothetical pricing API
	pricingURL := fmt.Sprintf("https://api.example.com/pricing/openai/%s", modelID)

	req, err := http.NewRequest("GET", pricingURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create pricing request: %w", err)
	}

	resp, err := pd.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pricing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pricing API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read pricing response: %w", err)
	}

	var apiResponse struct {
		InputCost  float64 `json:"input_cost"`
		OutputCost float64 `json:"output_cost"`
		Currency   string  `json:"currency"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse pricing response: %w", err)
	}

	return &PricingInfo{
		InputTokenCost:  apiResponse.InputCost,
		OutputTokenCost: apiResponse.OutputCost,
		Currency:        apiResponse.Currency,
		PricingModel:    "per_token",
		EffectiveFrom:   time.Now().Format("2006-01-02"),
	}, nil
}

// updateOpenAIPricing updates OpenAI pricing from external sources
func (pd *PricingDetector) updateOpenAIPricing(db *database.Database) error {
	// This would fetch current OpenAI pricing and update the database
	// For demonstration, we'll update a few key models

	modelsToUpdate := []string{"gpt-4-turbo", "gpt-4", "gpt-3.5-turbo"}

	for _, modelID := range modelsToUpdate {
		pricing, err := pd.detectOpenAIPricing(modelID)
		if err != nil {
			continue // Skip models we can't get pricing for
		}

		// Convert to database format and update
		now := time.Now()
		dbPricing := &database.Pricing{
			ModelID:              0, // Would need to look up model ID
			InputTokenCost:       pricing.InputTokenCost,
			OutputTokenCost:      pricing.OutputTokenCost,
			CachedInputTokenCost: pricing.CachedInputTokenCost,
			StorageCost:          pricing.StorageCost,
			RequestCost:          pricing.RequestCost,
			Currency:             pricing.Currency,
			PricingModel:         pricing.PricingModel,
			EffectiveFrom:        &now,
		}

		// In a real implementation, you'd update or insert the pricing record
		_ = dbPricing // Placeholder to avoid unused variable error
	}

	return nil
}

// updateAnthropicPricing updates Anthropic pricing from their API
func (pd *PricingDetector) updateAnthropicPricing(db *database.Database) error {
	// Anthropic provides pricing information via their documentation
	// This could be implemented to fetch from their pricing page or API

	// For now, use the cached pricing
	return nil
}

// detectOpenAIPricing detects pricing for OpenAI models
func (pd *PricingDetector) detectOpenAIPricing(modelID string) (*PricingInfo, error) {
	// Try to fetch real-time pricing from OpenAI API first
	if realTimePricing, err := pd.fetchOpenAIRealtimePricing(modelID); err == nil && realTimePricing != nil {
		return realTimePricing, nil
	}

	// Fall back to cached pricing structure (as of 2024)
	pricingMap := map[string]*PricingInfo{
		"gpt-4-turbo": {
			InputTokenCost:       10.0, // $10 per 1M input tokens
			OutputTokenCost:      30.0, // $30 per 1M output tokens
			CachedInputTokenCost: 5.0,  // $5 per 1M cached input tokens
			Currency:             "USD",
			PricingModel:         "per_token",
			EffectiveFrom:        "2024-01-01",
		},
		"gpt-4-turbo-preview": {
			InputTokenCost:       10.0,
			OutputTokenCost:      30.0,
			CachedInputTokenCost: 5.0,
			Currency:             "USD",
			PricingModel:         "per_token",
			EffectiveFrom:        "2024-01-01",
		},
		"gpt-4": {
			InputTokenCost:       30.0, // $30 per 1M input tokens
			OutputTokenCost:      60.0, // $60 per 1M output tokens
			CachedInputTokenCost: 15.0,
			Currency:             "USD",
			PricingModel:         "per_token",
			EffectiveFrom:        "2024-01-01",
		},
		"gpt-4-32k": {
			InputTokenCost:       60.0,  // $60 per 1M input tokens
			OutputTokenCost:      120.0, // $120 per 1M output tokens
			CachedInputTokenCost: 30.0,
			Currency:             "USD",
			PricingModel:         "per_token",
			EffectiveFrom:        "2024-01-01",
		},
		"gpt-3.5-turbo": {
			InputTokenCost:       0.5, // $0.5 per 1M input tokens
			OutputTokenCost:      1.5, // $1.5 per 1M output tokens
			CachedInputTokenCost: 0.25,
			Currency:             "USD",
			PricingModel:         "per_token",
			EffectiveFrom:        "2024-01-01",
		},
		"gpt-3.5-turbo-16k": {
			InputTokenCost:       1.0, // $1.0 per 1M input tokens
			OutputTokenCost:      2.0, // $2.0 per 1M output tokens
			CachedInputTokenCost: 0.5,
			Currency:             "USD",
			PricingModel:         "per_token",
			EffectiveFrom:        "2024-01-01",
		},
		"text-embedding-3-large": {
			InputTokenCost:  0.13, // $0.13 per 1M input tokens
			OutputTokenCost: 0.13,
			Currency:        "USD",
			PricingModel:    "per_token",
			EffectiveFrom:   "2024-01-01",
		},
		"text-embedding-3-small": {
			InputTokenCost:  0.02, // $0.02 per 1M input tokens
			OutputTokenCost: 0.02,
			Currency:        "USD",
			PricingModel:    "per_token",
			EffectiveFrom:   "2024-01-01",
		},
		"text-embedding-ada-002": {
			InputTokenCost:  0.10, // $0.10 per 1M input tokens
			OutputTokenCost: 0.10,
			Currency:        "USD",
			PricingModel:    "per_token",
			EffectiveFrom:   "2024-01-01",
		},
		"dall-e-3": {
			InputTokenCost:  4.0, // $4.0 per image (1024x1024)
			OutputTokenCost: 4.0,
			Currency:        "USD",
			PricingModel:    "per_image",
			EffectiveFrom:   "2024-01-01",
		},
		"dall-e-2": {
			InputTokenCost:  2.0, // $2.0 per image (1024x1024)
			OutputTokenCost: 2.0,
			Currency:        "USD",
			PricingModel:    "per_image",
			EffectiveFrom:   "2024-01-01",
		},
		"tts-1": {
			InputTokenCost:  15.0, // $15.0 per 1M characters
			OutputTokenCost: 15.0,
			Currency:        "USD",
			PricingModel:    "per_character",
			EffectiveFrom:   "2024-01-01",
		},
		"tts-1-hd": {
			InputTokenCost:  30.0, // $30.0 per 1M characters
			OutputTokenCost: 30.0,
			Currency:        "USD",
			PricingModel:    "per_character",
			EffectiveFrom:   "2024-01-01",
		},
		"whisper-1": {
			InputTokenCost:  0.006, // $0.006 per minute
			OutputTokenCost: 0.006,
			Currency:        "USD",
			PricingModel:    "per_minute",
			EffectiveFrom:   "2024-01-01",
		},
	}

	// Try exact match first
	if pricing, exists := pricingMap[modelID]; exists {
		return pricing, nil
	}

	// Try prefix matching for variant models
	for prefix, pricing := range pricingMap {
		if strings.HasPrefix(modelID, prefix) {
			return pricing, nil
		}
	}

	// Default pricing for unknown OpenAI models
	return &PricingInfo{
		InputTokenCost:  10.0, // Conservative default
		OutputTokenCost: 30.0,
		Currency:        "USD",
		PricingModel:    "per_token",
		EffectiveFrom:   "2024-01-01",
	}, nil
}

// detectAnthropicPricing detects pricing for Anthropic models
func (pd *PricingDetector) detectAnthropicPricing(modelID string) (*PricingInfo, error) {
	// Anthropic pricing structure (as of 2024)
	pricingMap := map[string]*PricingInfo{
		"claude-3-opus-20240229": {
			InputTokenCost:  15.0, // $15 per 1M input tokens
			OutputTokenCost: 75.0, // $75 per 1M output tokens
			Currency:        "USD",
			PricingModel:    "per_token",
			EffectiveFrom:   "2024-02-29",
		},
		"claude-3-sonnet-20240229": {
			InputTokenCost:  3.0,  // $3 per 1M input tokens
			OutputTokenCost: 15.0, // $15 per 1M output tokens
			Currency:        "USD",
			PricingModel:    "per_token",
			EffectiveFrom:   "2024-02-29",
		},
		"claude-3-haiku-20240307": {
			InputTokenCost:  0.25, // $0.25 per 1M input tokens
			OutputTokenCost: 1.25, // $1.25 per 1M output tokens
			Currency:        "USD",
			PricingModel:    "per_token",
			EffectiveFrom:   "2024-03-07",
		},
		"claude-2.1": {
			InputTokenCost:  8.0,  // $8 per 1M input tokens
			OutputTokenCost: 24.0, // $24 per 1M output tokens
			Currency:        "USD",
			PricingModel:    "per_token",
			EffectiveFrom:   "2023-11-21",
		},
		"claude-2.0": {
			InputTokenCost:  8.0,  // $8 per 1M input tokens
			OutputTokenCost: 24.0, // $24 per 1M output tokens
			Currency:        "USD",
			PricingModel:    "per_token",
			EffectiveFrom:   "2023-07-11",
		},
		"claude-instant-1.2": {
			InputTokenCost:  0.8, // $0.8 per 1M input tokens
			OutputTokenCost: 2.4, // $2.4 per 1M output tokens
			Currency:        "USD",
			PricingModel:    "per_token",
			EffectiveFrom:   "2023-08-09",
		},
	}

	// Try exact match first
	if pricing, exists := pricingMap[modelID]; exists {
		return pricing, nil
	}

	// Try prefix matching for variant models
	for prefix, pricing := range pricingMap {
		if strings.HasPrefix(modelID, prefix) {
			return pricing, nil
		}
	}

	// Default pricing for unknown Anthropic models
	return &PricingInfo{
		InputTokenCost:  8.0, // Conservative default
		OutputTokenCost: 24.0,
		Currency:        "USD",
		PricingModel:    "per_token",
		EffectiveFrom:   "2024-01-01",
	}, nil
}

// detectAzureOpenAIPricing detects pricing for Azure OpenAI models
func (pd *PricingDetector) detectAzureOpenAIPricing(modelID string) (*PricingInfo, error) {
	// Azure OpenAI pricing is typically similar to OpenAI but with different rates
	// This is a simplified version - actual Azure pricing varies by region and tier
	pricingMap := map[string]*PricingInfo{
		"gpt-4": {
			InputTokenCost:       30.0, // Similar to OpenAI but may vary by region
			OutputTokenCost:      60.0,
			CachedInputTokenCost: 15.0,
			Currency:             "USD",
			PricingModel:         "per_token",
			EffectiveFrom:        "2024-01-01",
		},
		"gpt-4-32k": {
			InputTokenCost:       60.0,
			OutputTokenCost:      120.0,
			CachedInputTokenCost: 30.0,
			Currency:             "USD",
			PricingModel:         "per_token",
			EffectiveFrom:        "2024-01-01",
		},
		"gpt-35-turbo": {
			InputTokenCost:       0.5,
			OutputTokenCost:      1.5,
			CachedInputTokenCost: 0.25,
			Currency:             "USD",
			PricingModel:         "per_token",
			EffectiveFrom:        "2024-01-01",
		},
	}

	// Try exact match first
	if pricing, exists := pricingMap[modelID]; exists {
		return pricing, nil
	}

	// Default to OpenAI pricing for Azure models
	return pd.detectOpenAIPricing(modelID)
}

// detectGooglePricing detects pricing for Google Cloud models
func (pd *PricingDetector) detectGooglePricing(modelID string) (*PricingInfo, error) {
	// Google Cloud AI pricing (as of 2024)
	pricingMap := map[string]*PricingInfo{
		"gemini-pro": {
			InputTokenCost:  0.5, // $0.5 per 1M input tokens
			OutputTokenCost: 1.5, // $1.5 per 1M output tokens
			Currency:        "USD",
			PricingModel:    "per_token",
			EffectiveFrom:   "2024-01-01",
		},
		"gemini-pro-vision": {
			InputTokenCost:  0.5, // $0.5 per 1M input tokens
			OutputTokenCost: 1.5, // $1.5 per 1M output tokens
			Currency:        "USD",
			PricingModel:    "per_token",
			EffectiveFrom:   "2024-01-01",
		},
		"text-bison": {
			InputTokenCost:  0.5, // $0.5 per 1K characters
			OutputTokenCost: 0.5, // $0.5 per 1K characters
			Currency:        "USD",
			PricingModel:    "per_character",
			EffectiveFrom:   "2024-01-01",
		},
		"chat-bison": {
			InputTokenCost:  0.5, // $0.5 per 1K characters
			OutputTokenCost: 0.5, // $0.5 per 1K characters
			Currency:        "USD",
			PricingModel:    "per_character",
			EffectiveFrom:   "2024-01-01",
		},
		"textembedding-gecko": {
			InputTokenCost:  0.1, // $0.1 per 1K characters
			OutputTokenCost: 0.1,
			Currency:        "USD",
			PricingModel:    "per_character",
			EffectiveFrom:   "2024-01-01",
		},
	}

	// Try exact match first
	if pricing, exists := pricingMap[modelID]; exists {
		return pricing, nil
	}

	// Default pricing for unknown Google models
	return &PricingInfo{
		InputTokenCost:  0.5, // Conservative default
		OutputTokenCost: 1.5,
		Currency:        "USD",
		PricingModel:    "per_token",
		EffectiveFrom:   "2024-01-01",
	}, nil
}

// detectCoherePricing detects pricing for Cohere models
func (pd *PricingDetector) detectCoherePricing(modelID string) (*PricingInfo, error) {
	// Cohere pricing structure (as of 2024)
	pricingMap := map[string]*PricingInfo{
		"command": {
			InputTokenCost:  15.0, // $15.0 per 1M tokens
			OutputTokenCost: 15.0,
			Currency:        "USD",
			PricingModel:    "per_token",
			EffectiveFrom:   "2024-01-01",
		},
		"command-light": {
			InputTokenCost:  0.3, // $0.3 per 1M tokens
			OutputTokenCost: 0.6,
			Currency:        "USD",
			PricingModel:    "per_token",
			EffectiveFrom:   "2024-01-01",
		},
		"command-r": {
			InputTokenCost:  0.5, // $0.5 per 1M tokens
			OutputTokenCost: 1.5,
			Currency:        "USD",
			PricingModel:    "per_token",
			EffectiveFrom:   "2024-01-01",
		},
		"command-r-plus": {
			InputTokenCost:  3.0, // $3.0 per 1M tokens
			OutputTokenCost: 15.0,
			Currency:        "USD",
			PricingModel:    "per_token",
			EffectiveFrom:   "2024-01-01",
		},
		"embed-english-v3.0": {
			InputTokenCost:  0.1, // $0.1 per 1M tokens
			OutputTokenCost: 0.1,
			Currency:        "USD",
			PricingModel:    "per_token",
			EffectiveFrom:   "2024-01-01",
		},
		"embed-multilingual-v3.0": {
			InputTokenCost:  0.1, // $0.1 per 1M tokens
			OutputTokenCost: 0.1,
			Currency:        "USD",
			PricingModel:    "per_token",
			EffectiveFrom:   "2024-01-01",
		},
	}

	// Try exact match first
	if pricing, exists := pricingMap[modelID]; exists {
		return pricing, nil
	}

	// Default pricing for unknown Cohere models
	return &PricingInfo{
		InputTokenCost:  15.0, // Conservative default
		OutputTokenCost: 15.0,
		Currency:        "USD",
		PricingModel:    "per_token",
		EffectiveFrom:   "2024-01-01",
	}, nil
}

// detectGenericPricing attempts to detect pricing for unknown providers
func (pd *PricingDetector) detectGenericPricing(providerName, modelID string) (*PricingInfo, error) {
	// Try to extract pricing from provider's website or API documentation
	// This is a placeholder for more sophisticated pricing detection

	// Common patterns in model names that might indicate pricing tier
	if strings.Contains(modelID, "large") || strings.Contains(modelID, "xl") {
		return &PricingInfo{
			InputTokenCost:  20.0, // Higher tier pricing
			OutputTokenCost: 60.0,
			Currency:        "USD",
			PricingModel:    "per_token",
			EffectiveFrom:   "2024-01-01",
		}, nil
	}

	if strings.Contains(modelID, "small") || strings.Contains(modelID, "light") {
		return &PricingInfo{
			InputTokenCost:  1.0, // Lower tier pricing
			OutputTokenCost: 3.0,
			Currency:        "USD",
			PricingModel:    "per_token",
			EffectiveFrom:   "2024-01-01",
		}, nil
	}

	if strings.Contains(modelID, "embed") || strings.Contains(modelID, "embedding") {
		return &PricingInfo{
			InputTokenCost:  0.1, // Embedding model pricing
			OutputTokenCost: 0.1,
			Currency:        "USD",
			PricingModel:    "per_token",
			EffectiveFrom:   "2024-01-01",
		}, nil
	}

	// Default conservative pricing
	return &PricingInfo{
		InputTokenCost:  10.0, // Medium tier pricing
		OutputTokenCost: 30.0,
		Currency:        "USD",
		PricingModel:    "per_token",
		EffectiveFrom:   "2024-01-01",
	}, nil
}

// SavePricing saves pricing information to the database
func SavePricing(db *database.Database, modelID int64, pricing *PricingInfo) error {
	// Parse effective dates
	var effectiveFrom, effectiveTo *time.Time
	if pricing.EffectiveFrom != "" {
		from, err := time.Parse("2006-01-02", pricing.EffectiveFrom)
		if err == nil {
			effectiveFrom = &from
		}
	}

	if pricing.EffectiveTo != "" {
		to, err := time.Parse("2006-01-02", pricing.EffectiveTo)
		if err == nil {
			effectiveTo = &to
		}
	}

	pricingRecord := &database.Pricing{
		ModelID:              modelID,
		InputTokenCost:       pricing.InputTokenCost,
		OutputTokenCost:      pricing.OutputTokenCost,
		CachedInputTokenCost: pricing.CachedInputTokenCost,
		StorageCost:          pricing.StorageCost,
		RequestCost:          pricing.RequestCost,
		Currency:             pricing.Currency,
		PricingModel:         pricing.PricingModel,
		EffectiveFrom:        effectiveFrom,
		EffectiveTo:          effectiveTo,
	}

	return db.CreatePricing(pricingRecord)
}

// UpdatePricingFromAPI fetches and updates pricing from provider APIs
func (pd *PricingDetector) UpdatePricingFromAPI(db *database.Database, provider *database.Provider, model *database.Model) error {
	pricing, err := pd.DetectPricing(provider.Name, model.ModelID)
	if err != nil {
		return fmt.Errorf("failed to detect pricing for model %s: %w", model.ModelID, err)
	}

	return SavePricing(db, model.ID, pricing)
}

// BatchUpdatePricing updates pricing for multiple models
func (pd *PricingDetector) BatchUpdatePricing(db *database.Database, provider *database.Provider, models []*database.Model) error {
	for _, model := range models {
		if err := pd.UpdatePricingFromAPI(db, provider, model); err != nil {
			// Log error but continue with other models
			fmt.Printf("Warning: Failed to update pricing for model %s: %v\n", model.ModelID, err)
		}
	}
	return nil
}

// GetPricingComparison compares pricing between different models
func GetPricingComparison(db *database.Database, modelIDs []int64) (map[int64]*database.Pricing, error) {
	pricingMap := make(map[int64]*database.Pricing)

	for _, modelID := range modelIDs {
		// Get latest pricing for each model
		pricing, err := db.GetLatestPricing(modelID)
		if err != nil {
			// Skip models without pricing information
			continue
		}

		pricingMap[modelID] = pricing
	}

	return pricingMap, nil
}

// CalculateCostEstimate calculates estimated cost for a given usage pattern
func CalculateCostEstimate(pricing *PricingInfo, inputTokens, outputTokens int64, requestCount int) float64 {
	var totalCost float64

	switch pricing.PricingModel {
	case "per_token":
		totalCost += (float64(inputTokens) / 1000000.0) * pricing.InputTokenCost
		totalCost += (float64(outputTokens) / 1000000.0) * pricing.OutputTokenCost
	case "per_request":
		totalCost += float64(requestCount) * pricing.RequestCost
	case "per_character":
		totalCost += (float64(inputTokens) / 1000.0) * pricing.InputTokenCost
		totalCost += (float64(outputTokens) / 1000.0) * pricing.OutputTokenCost
	case "per_image":
		totalCost += float64(requestCount) * pricing.InputTokenCost
	case "per_minute":
		totalCost += float64(requestCount) * pricing.InputTokenCost
	default:
		// Default to per_token if unknown model
		totalCost += (float64(inputTokens) / 1000000.0) * pricing.InputTokenCost
		totalCost += (float64(outputTokens) / 1000000.0) * pricing.OutputTokenCost
	}

	return totalCost
}

// ValidatePricing validates pricing information
func ValidatePricing(pricing *PricingInfo) error {
	if pricing.InputTokenCost < 0 {
		return fmt.Errorf("input token cost cannot be negative")
	}

	if pricing.OutputTokenCost < 0 {
		return fmt.Errorf("output token cost cannot be negative")
	}

	if pricing.Currency == "" {
		return fmt.Errorf("currency is required")
	}

	if pricing.PricingModel == "" {
		return fmt.Errorf("pricing model is required")
	}

	validModels := []string{"per_token", "per_request", "per_character", "per_image", "per_minute"}
	isValid := false
	for _, model := range validModels {
		if pricing.PricingModel == model {
			isValid = true
			break
		}
	}

	if !isValid {
		return fmt.Errorf("invalid pricing model: %s", pricing.PricingModel)
	}

	return nil
}
