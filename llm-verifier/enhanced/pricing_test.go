package enhanced

import (
	"testing"
)

func TestNewPricingDetector(t *testing.T) {
	detector := NewPricingDetector()
	if detector == nil {
		t.Fatal("Expected PricingDetector to be created, got nil")
	}
	if detector.httpClient == nil {
		t.Error("Expected httpClient to be initialized")
	}
	if detector.httpClient.Timeout.Seconds() != 30 {
		t.Errorf("Expected timeout to be 30 seconds, got %v", detector.httpClient.Timeout)
	}
}

func TestDetectOpenAIPricing(t *testing.T) {
	detector := NewPricingDetector()

	tests := []struct {
		name     string
		modelID  string
		expected float64 // Expected input token cost
	}{
		{
			name:     "GPT-4 Turbo",
			modelID:  "gpt-4-turbo",
			expected: 10.0,
		},
		{
			name:     "GPT-4",
			modelID:  "gpt-4",
			expected: 30.0,
		},
		{
			name:     "GPT-3.5 Turbo",
			modelID:  "gpt-3.5-turbo",
			expected: 0.5,
		},
		{
			name:     "Text Embedding 3 Small",
			modelID:  "text-embedding-3-small",
			expected: 0.02,
		},
		{
			name:     "Unknown OpenAI model",
			modelID:  "gpt-5-unknown",
			expected: 10.0, // Default conservative pricing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricing, err := detector.detectOpenAIPricing(tt.modelID)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if pricing.InputTokenCost != tt.expected {
				t.Errorf("Expected input token cost %v, got %v", tt.expected, pricing.InputTokenCost)
			}
			if pricing.Currency != "USD" {
				t.Errorf("Expected currency USD, got %s", pricing.Currency)
			}
			if pricing.PricingModel != "per_token" {
				t.Errorf("Expected pricing model per_token, got %s", pricing.PricingModel)
			}
		})
	}
}

func TestDetectAnthropicPricing(t *testing.T) {
	detector := NewPricingDetector()

	tests := []struct {
		name     string
		modelID  string
		expected float64 // Expected input token cost
	}{
		{
			name:     "Claude 3 Opus",
			modelID:  "claude-3-opus-20240229",
			expected: 15.0,
		},
		{
			name:     "Claude 3 Sonnet",
			modelID:  "claude-3-sonnet-20240229",
			expected: 3.0,
		},
		{
			name:     "Claude 3 Haiku",
			modelID:  "claude-3-haiku-20240307",
			expected: 0.25,
		},
		{
			name:     "Unknown Anthropic model",
			modelID:  "claude-4-unknown",
			expected: 8.0, // Default conservative pricing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricing, err := detector.detectAnthropicPricing(tt.modelID)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if pricing.InputTokenCost != tt.expected {
				t.Errorf("Expected input token cost %v, got %v", tt.expected, pricing.InputTokenCost)
			}
			if pricing.Currency != "USD" {
				t.Errorf("Expected currency USD, got %s", pricing.Currency)
			}
			if pricing.PricingModel != "per_token" {
				t.Errorf("Expected pricing model per_token, got %s", pricing.PricingModel)
			}
		})
	}
}

func TestDetectGooglePricing(t *testing.T) {
	detector := NewPricingDetector()

	tests := []struct {
		name     string
		modelID  string
		expected float64 // Expected input token cost
	}{
		{
			name:     "Gemini Pro",
			modelID:  "gemini-pro",
			expected: 0.5,
		},
		{
			name:     "Text Embedding Gecko",
			modelID:  "textembedding-gecko",
			expected: 0.1,
		},
		{
			name:     "Unknown Google model",
			modelID:  "gemini-ultra-unknown",
			expected: 0.5, // Default conservative pricing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricing, err := detector.detectGooglePricing(tt.modelID)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if pricing.InputTokenCost != tt.expected {
				t.Errorf("Expected input token cost %v, got %v", tt.expected, pricing.InputTokenCost)
			}
			if pricing.Currency != "USD" {
				t.Errorf("Expected currency USD, got %s", pricing.Currency)
			}
		})
	}
}

func TestDetectCoherePricing(t *testing.T) {
	detector := NewPricingDetector()

	tests := []struct {
		name     string
		modelID  string
		expected float64 // Expected input token cost
	}{
		{
			name:     "Command",
			modelID:  "command",
			expected: 15.0,
		},
		{
			name:     "Command Light",
			modelID:  "command-light",
			expected: 0.3,
		},
		{
			name:     "Embed English v3.0",
			modelID:  "embed-english-v3.0",
			expected: 0.1,
		},
		{
			name:     "Unknown Cohere model",
			modelID:  "command-xxl-unknown",
			expected: 15.0, // Default conservative pricing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricing, err := detector.detectCoherePricing(tt.modelID)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if pricing.InputTokenCost != tt.expected {
				t.Errorf("Expected input token cost %v, got %v", tt.expected, pricing.InputTokenCost)
			}
			if pricing.Currency != "USD" {
				t.Errorf("Expected currency USD, got %s", pricing.Currency)
			}
			if pricing.PricingModel != "per_token" {
				t.Errorf("Expected pricing model per_token, got %s", pricing.PricingModel)
			}
		})
	}
}

func TestDetectGenericPricing(t *testing.T) {
	detector := NewPricingDetector()

	tests := []struct {
		name         string
		providerName string
		modelID      string
		expectedMin  float64 // Minimum expected input token cost
		expectedMax  float64 // Maximum expected input token cost
	}{
		{
			name:         "Large model pattern",
			providerName: "unknown",
			modelID:      "model-large-v1",
			expectedMin:  15.0,
			expectedMax:  25.0,
		},
		{
			name:         "Small model pattern",
			providerName: "unknown",
			modelID:      "model-small-v1",
			expectedMin:  0.5,
			expectedMax:  2.0,
		},
		{
			name:         "Embedding model pattern",
			providerName: "unknown",
			modelID:      "embedding-model-v2",
			expectedMin:  0.05,
			expectedMax:  0.2,
		},
		{
			name:         "Default model",
			providerName: "unknown",
			modelID:      "regular-model",
			expectedMin:  5.0,
			expectedMax:  15.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricing, err := detector.detectGenericPricing(tt.providerName, tt.modelID)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if pricing.InputTokenCost < tt.expectedMin || pricing.InputTokenCost > tt.expectedMax {
				t.Errorf("Expected input token cost between %v and %v, got %v", tt.expectedMin, tt.expectedMax, pricing.InputTokenCost)
			}
			if pricing.Currency != "USD" {
				t.Errorf("Expected currency USD, got %s", pricing.Currency)
			}
			if pricing.PricingModel != "per_token" {
				t.Errorf("Expected pricing model per_token, got %s", pricing.PricingModel)
			}
		})
	}
}

func TestDetectPricing(t *testing.T) {
	detector := NewPricingDetector()

	tests := []struct {
		name         string
		providerName string
		modelID      string
		shouldError  bool
	}{
		{
			name:         "OpenAI provider",
			providerName: "openai",
			modelID:      "gpt-4-turbo",
			shouldError:  false,
		},
		{
			name:         "Anthropic provider",
			providerName: "anthropic",
			modelID:      "claude-3-sonnet-20240229",
			shouldError:  false,
		},
		{
			name:         "Google provider",
			providerName: "google",
			modelID:      "gemini-pro",
			shouldError:  false,
		},
		{
			name:         "Cohere provider",
			providerName: "cohere",
			modelID:      "command",
			shouldError:  false,
		},
		{
			name:         "Azure provider",
			providerName: "azure",
			modelID:      "gpt-4",
			shouldError:  false,
		},
		{
			name:         "Unknown provider",
			providerName: "unknown-provider",
			modelID:      "some-model",
			shouldError:  false, // Should not error, should return generic pricing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricing, err := detector.DetectPricing(tt.providerName, tt.modelID)
			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if pricing == nil {
					t.Error("Expected pricing info but got nil")
				}
				if pricing.Currency == "" {
					t.Error("Expected currency to be set")
				}
				if pricing.PricingModel == "" {
					t.Error("Expected pricing model to be set")
				}
			}
		})
	}
}

func TestCalculateCostEstimate(t *testing.T) {
	tests := []struct {
		name         string
		pricing      *PricingInfo
		inputTokens  int64
		outputTokens int64
		requestCount int
		expectedMin  float64
		expectedMax  float64
	}{
		{
			name: "Per token pricing",
			pricing: &PricingInfo{
				InputTokenCost:  10.0,
				OutputTokenCost: 30.0,
				PricingModel:    "per_token",
			},
			inputTokens:  1000000, // 1M tokens
			outputTokens: 500000,  // 0.5M tokens
			requestCount: 100,
			expectedMin:  25.0, // 10 + 15
			expectedMax:  26.0,
		},
		{
			name: "Per request pricing",
			pricing: &PricingInfo{
				RequestCost:  0.01,
				PricingModel: "per_request",
			},
			inputTokens:  1000000,
			outputTokens: 500000,
			requestCount: 100,
			expectedMin:  0.99, // 100 * 0.01
			expectedMax:  1.01,
		},
		{
			name: "Per character pricing",
			pricing: &PricingInfo{
				InputTokenCost:  0.5,
				OutputTokenCost: 0.5,
				PricingModel:    "per_character",
			},
			inputTokens:  1000, // 1K characters
			outputTokens: 500,  // 0.5K characters
			requestCount: 10,
			expectedMin:  0.74, // (1 * 0.5) + (0.5 * 0.5)
			expectedMax:  0.76,
		},
		{
			name: "Per image pricing",
			pricing: &PricingInfo{
				InputTokenCost: 4.0,
				PricingModel:   "per_image",
			},
			inputTokens:  0,
			outputTokens: 0,
			requestCount: 10,
			expectedMin:  39.9, // 10 * 4.0
			expectedMax:  40.1,
		},
		{
			name: "Unknown pricing model defaults to per_token",
			pricing: &PricingInfo{
				InputTokenCost:  5.0,
				OutputTokenCost: 15.0,
				PricingModel:    "unknown_model",
			},
			inputTokens:  1000000,
			outputTokens: 500000,
			requestCount: 100,
			expectedMin:  12.4, // 5 + 7.5
			expectedMax:  12.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := CalculateCostEstimate(tt.pricing, tt.inputTokens, tt.outputTokens, tt.requestCount)
			if cost < tt.expectedMin || cost > tt.expectedMax {
				t.Errorf("Expected cost between %v and %v, got %v", tt.expectedMin, tt.expectedMax, cost)
			}
		})
	}
}

func TestValidatePricing(t *testing.T) {
	tests := []struct {
		name        string
		pricing     *PricingInfo
		shouldError bool
	}{
		{
			name: "Valid pricing",
			pricing: &PricingInfo{
				InputTokenCost:  10.0,
				OutputTokenCost: 30.0,
				Currency:        "USD",
				PricingModel:    "per_token",
			},
			shouldError: false,
		},
		{
			name: "Negative input cost",
			pricing: &PricingInfo{
				InputTokenCost:  -1.0,
				OutputTokenCost: 30.0,
				Currency:        "USD",
				PricingModel:    "per_token",
			},
			shouldError: true,
		},
		{
			name: "Negative output cost",
			pricing: &PricingInfo{
				InputTokenCost:  10.0,
				OutputTokenCost: -1.0,
				Currency:        "USD",
				PricingModel:    "per_token",
			},
			shouldError: true,
		},
		{
			name: "Missing currency",
			pricing: &PricingInfo{
				InputTokenCost:  10.0,
				OutputTokenCost: 30.0,
				Currency:        "",
				PricingModel:    "per_token",
			},
			shouldError: true,
		},
		{
			name: "Missing pricing model",
			pricing: &PricingInfo{
				InputTokenCost:  10.0,
				OutputTokenCost: 30.0,
				Currency:        "USD",
				PricingModel:    "",
			},
			shouldError: true,
		},
		{
			name: "Invalid pricing model",
			pricing: &PricingInfo{
				InputTokenCost:  10.0,
				OutputTokenCost: 30.0,
				Currency:        "USD",
				PricingModel:    "invalid_model",
			},
			shouldError: true,
		},
		{
			name: "Valid per_request model",
			pricing: &PricingInfo{
				RequestCost:  0.01,
				Currency:     "USD",
				PricingModel: "per_request",
			},
			shouldError: false,
		},
		{
			name: "Valid per_character model",
			pricing: &PricingInfo{
				InputTokenCost:  0.5,
				OutputTokenCost: 0.5,
				Currency:        "USD",
				PricingModel:    "per_character",
			},
			shouldError: false,
		},
		{
			name: "Valid per_image model",
			pricing: &PricingInfo{
				InputTokenCost: 4.0,
				Currency:       "USD",
				PricingModel:   "per_image",
			},
			shouldError: false,
		},
		{
			name: "Valid per_minute model",
			pricing: &PricingInfo{
				InputTokenCost: 0.006,
				Currency:       "USD",
				PricingModel:   "per_minute",
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePricing(tt.pricing)
			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestPricingInfoStructure(t *testing.T) {
	// Test that PricingInfo struct fields are properly defined
	pricing := &PricingInfo{
		InputTokenCost:       10.0,
		OutputTokenCost:      30.0,
		CachedInputTokenCost: 5.0,
		StorageCost:          0.1,
		RequestCost:          0.01,
		Currency:             "USD",
		PricingModel:         "per_token",
		EffectiveFrom:        "2024-01-01",
		EffectiveTo:          "2024-12-31",
	}

	if pricing.InputTokenCost != 10.0 {
		t.Errorf("Expected InputTokenCost 10.0, got %v", pricing.InputTokenCost)
	}
	if pricing.OutputTokenCost != 30.0 {
		t.Errorf("Expected OutputTokenCost 30.0, got %v", pricing.OutputTokenCost)
	}
	if pricing.CachedInputTokenCost != 5.0 {
		t.Errorf("Expected CachedInputTokenCost 5.0, got %v", pricing.CachedInputTokenCost)
	}
	if pricing.StorageCost != 0.1 {
		t.Errorf("Expected StorageCost 0.1, got %v", pricing.StorageCost)
	}
	if pricing.RequestCost != 0.01 {
		t.Errorf("Expected RequestCost 0.01, got %v", pricing.RequestCost)
	}
	if pricing.Currency != "USD" {
		t.Errorf("Expected Currency USD, got %s", pricing.Currency)
	}
	if pricing.PricingModel != "per_token" {
		t.Errorf("Expected PricingModel per_token, got %s", pricing.PricingModel)
	}
	if pricing.EffectiveFrom != "2024-01-01" {
		t.Errorf("Expected EffectiveFrom 2024-01-01, got %s", pricing.EffectiveFrom)
	}
	if pricing.EffectiveTo != "2024-12-31" {
		t.Errorf("Expected EffectiveTo 2024-12-31, got %s", pricing.EffectiveTo)
	}
}
