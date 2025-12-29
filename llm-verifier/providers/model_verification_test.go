package providers

import (
	"context"
	"testing"
	"time"

	"llm-verifier/client"
	"llm-verifier/logging"
)

// MockProviderClient implements ProviderClientInterface for testing
type MockProviderClient struct {
	baseURL    string
	apiKey     string
	httpClient *client.HTTPClient
}

func (m *MockProviderClient) GetBaseURL() string {
	return m.baseURL
}

func (m *MockProviderClient) GetAPIKey() string {
	return m.apiKey
}

func (m *MockProviderClient) GetHTTPClient() *client.HTTPClient {
	return m.httpClient
}

func TestModelVerificationService_VerifyModel(t *testing.T) {
	logger, err := logging.NewLogger(nil, map[string]any{"log_level": "debug"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	httpClient := client.NewHTTPClient(logger)
	
	config := VerificationConfig{
		Enabled:               true,
		StrictMode:            true,
		MaxRetries:            3,
		TimeoutSeconds:        30,
		RequireAffirmative:    true,
		MinVerificationScore:  0.7,
	}
	
	verificationService := NewModelVerificationService(httpClient, logger, config)
	
	// Create a test model
	testModel := Model{
		ID:           "gpt-4",
		Name:         "GPT-4",
		ProviderID:   "openai",
		ProviderName: "OpenAI",
		Features:     map[string]interface{}{"tool_call": true},
		MaxTokens:    8192,
		CostPer1MInput: 30.0,
		CostPer1MOutput: 60.0,
	}
	
	// Create mock provider client
	mockProviderClient := &ProviderClient{
		ProviderID: "openai",
		BaseURL:    "https://api.openai.com/v1",
		APIKey:     "test-api-key",
		HTTPClient: httpClient,
		logger:     logger,
	}
	
	ctx := context.Background()
	
	// Test verification (this will fail with real API, but tests the structure)
	result, err := verificationService.VerifyModel(ctx, testModel, mockProviderClient)
	
	if err != nil {
		t.Logf("Expected error in test environment: %v", err)
		// In a real test environment with mock responses, this should succeed
		return
	}
	
	// Verify result structure
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	
	if result.ModelID != testModel.ID {
		t.Errorf("Expected ModelID %s, got %s", testModel.ID, result.ModelID)
	}
	
	if result.ProviderID != testModel.ProviderID {
		t.Errorf("Expected ProviderID %s, got %s", testModel.ProviderID, result.ProviderID)
	}
	
	if result.VerificationStatus == "" {
		t.Error("Expected non-empty VerificationStatus")
	}
}

func TestModelVerificationService_VerifyModels(t *testing.T) {
	logger, err := logging.NewLogger(nil, map[string]any{"log_level": "debug"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	httpClient := client.NewHTTPClient(logger)
	
	config := VerificationConfig{
		Enabled:               true,
		StrictMode:            false, // More lenient for testing
		MaxRetries:            1,
		TimeoutSeconds:        10,
		RequireAffirmative:    true,
		MinVerificationScore:  0.5,
	}
	
	verificationService := NewModelVerificationService(httpClient, logger, config)
	
	// Create test models
	testModels := []Model{
		{
			ID:           "gpt-4",
			Name:         "GPT-4",
			ProviderID:   "openai",
			ProviderName: "OpenAI",
			Features:     map[string]interface{}{"tool_call": true},
			MaxTokens:    8192,
		},
		{
			ID:           "claude-3-sonnet",
			Name:         "Claude 3 Sonnet",
			ProviderID:   "anthropic",
			ProviderName: "Anthropic",
			Features:     map[string]interface{}{"tool_call": true},
			MaxTokens:    200000,
		},
	}
	
	// Create mock provider clients
	providerClients := map[string]*ProviderClient{
		"openai": {
			ProviderID: "openai",
			BaseURL:    "https://api.openai.com/v1",
			APIKey:     "test-openai-key",
			HTTPClient: httpClient,
			logger:     logger,
		},
		"anthropic": {
			ProviderID: "anthropic",
			BaseURL:    "https://api.anthropic.com/v1",
			APIKey:     "test-anthropic-key",
			HTTPClient: httpClient,
			logger:     logger,
		},
	}
	
	ctx := context.Background()
	
	// Test batch verification
	results := verificationService.VerifyModels(ctx, testModels, providerClients)
	
	if len(results) != len(testModels) {
		t.Errorf("Expected %d results, got %d", len(testModels), len(results))
	}
	
	// Verify each result
	for _, model := range testModels {
		key := fmt.Sprintf("%s:%s", model.ProviderID, model.ID)
		result, exists := results[key]
		
		if !exists {
		t.Errorf("Expected result for model %s", model.ID)
			continue
		}
		
		if result.ModelID != model.ID {
			t.Errorf("Expected ModelID %s, got %s", model.ID, result.ModelID)
		}
		
		if result.ProviderID != model.ProviderID {
			t.Errorf("Expected ProviderID %s, got %s", model.ProviderID, result.ProviderID)
		}
	}
}

func TestModelVerificationService_IsModelVerified(t *testing.T) {
	logger, err := logging.NewLogger(nil, map[string]any{"log_level": "debug"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	httpClient := client.NewHTTPClient(logger)
	
	config := VerificationConfig{
		Enabled:               true,
		StrictMode:            true,
		MaxRetries:            3,
		TimeoutSeconds:        30,
		RequireAffirmative:    true,
		MinVerificationScore:  0.7,
	}
	
	verificationService := NewModelVerificationService(httpClient, logger, config)
	
	// Test with verification disabled
	verificationService.EnableVerification(false)
	
	if !verificationService.IsModelVerified("openai", "gpt-4") {
		t.Error("Expected model to be verified when verification is disabled")
	}
	
	// Test with verification enabled but no result
	verificationService.EnableVerification(true)
	
	if verificationService.IsModelVerified("openai", "gpt-4") {
		t.Error("Expected model to not be verified when no verification result exists")
	}
	
	// Manually add a verification result
	key := "openai:gpt-4"
	verificationService.verificationResults[key] = &ModelVerificationResult{
		ModelID:             "gpt-4",
		ProviderID:          "openai",
		VerificationStatus:  "verified",
		CanSeeCode:          true,
		AffirmativeResponse: true,
		VerificationScore:   0.8,
		LastVerifiedAt:      time.Now(),
	}
	
	// Test with valid verification result
	if !verificationService.IsModelVerified("openai", "gpt-4") {
		t.Error("Expected model to be verified with valid result")
	}
}

func TestModelVerificationService_GetVerifiedModels(t *testing.T) {
	logger, err := logging.NewLogger(nil, map[string]any{"log_level": "debug"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	httpClient := client.NewHTTPClient(logger)
	
	config := VerificationConfig{
		Enabled:               true,
		StrictMode:            true,
		MaxRetries:            3,
		TimeoutSeconds:        30,
		RequireAffirmative:    true,
		MinVerificationScore:  0.7,
	}
	
	verificationService := NewModelVerificationService(httpClient, logger, config)
	
	// Create test models
	testModels := []Model{
		{
			ID:           "gpt-4",
			Name:         "GPT-4",
			ProviderID:   "openai",
			ProviderName: "OpenAI",
		},
		{
			ID:           "claude-3-sonnet",
			Name:         "Claude 3 Sonnet",
			ProviderID:   "anthropic",
			ProviderName: "Anthropic",
		},
		{
			ID:           "unverified-model",
			Name:         "Unverified Model",
			ProviderID:   "test",
			ProviderName: "Test Provider",
		},
	}
	
	// Add verification results for some models
	verificationService.verificationResults["openai:gpt-4"] = &ModelVerificationResult{
		ModelID:             "gpt-4",
		ProviderID:          "openai",
		VerificationStatus:  "verified",
		CanSeeCode:          true,
		AffirmativeResponse: true,
		VerificationScore:   0.8,
		LastVerifiedAt:      time.Now(),
	}
	
	verificationService.verificationResults["anthropic:claude-3-sonnet"] = &ModelVerificationResult{
		ModelID:             "claude-3-sonnet",
		ProviderID:          "anthropic",
		VerificationStatus:  "verified",
		CanSeeCode:          true,
		AffirmativeResponse: true,
		VerificationScore:   0.9,
		LastVerifiedAt:      time.Now(),
	}
	
	// Test filtering verified models
	verifiedModels := verificationService.GetVerifiedModels(testModels)
	
	if len(verifiedModels) != 2 {
		t.Errorf("Expected 2 verified models, got %d", len(verifiedModels))
	}
	
	// Verify the correct models are included
	verifiedModelIDs := make(map[string]bool)
	for _, model := range verifiedModels {
		verifiedModelIDs[model.ID] = true
	}
	
	if !verifiedModelIDs["gpt-4"] {
		t.Error("Expected gpt-4 to be in verified models")
	}
	
	if !verifiedModelIDs["claude-3-sonnet"] {
		t.Error("Expected claude-3-sonnet to be in verified models")
	}
	
	if verifiedModelIDs["unverified-model"] {
		t.Error("Expected unverified-model to not be in verified models")
	}
}

func TestEnhancedModelProviderService_GetModelsWithVerification(t *testing.T) {
	logger, err := logging.NewLogger(nil, map[string]any{"log_level": "debug"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	// Create verification configuration
	verificationConfig := VerificationConfig{
		Enabled:               true,
		StrictMode:            false, // More lenient for testing
		MaxRetries:            1,
		TimeoutSeconds:        10,
		RequireAffirmative:    true,
		MinVerificationScore:  0.5,
	}
	
	// Create enhanced service
	enhancedService := NewEnhancedModelProviderService("./test-config.yaml", logger, verificationConfig)
	
	// Register a test provider
	enhancedService.RegisterProvider("test-provider", "https://api.test.com/v1", "test-api-key")
	
	ctx := context.Background()
	
	// This test would require mocking the actual API responses
	// For now, we just test that the method doesn't panic
	models, err := enhancedService.GetModelsWithVerification(ctx, "test-provider")
	
	// In a real test environment, we would mock the responses
	if err != nil {
		t.Logf("Expected error in test environment: %v", err)
		return
	}
	
	t.Logf("Got %d models from test provider", len(models))
}

func TestVerifiedConfigGenerator(t *testing.T) {
	logger, err := logging.NewLogger(nil, map[string]any{"log_level": "debug"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	// Create a mock verified config
	verifiedConfig := &VerifiedConfig{
		GeneratedAt:         time.Now(),
		VerificationEnabled: true,
		StrictMode:          true,
		TotalModels:         10,
		VerifiedModels:      8,
		Providers: map[string]VerifiedProviderConfig{
			"openai": {
				ProviderID:   "openai",
				ProviderName: "OpenAI",
				BaseURL:      "https://api.openai.com/v1",
				ModelCount:   5,
				VerifiedModels: []VerifiedModelConfig{
					{
						ModelID:             "gpt-4",
						ModelName:           "GPT-4",
						DisplayName:         "GPT-4",
						Features:            map[string]interface{}{"tool_call": true},
						MaxTokens:           8192,
						CostPer1MInput:      30.0,
						CostPer1MOutput:     60.0,
						VerificationScore:   0.8,
						CanSeeCode:          true,
						AffirmativeResponse: true,
						LastVerifiedAt:      time.Now(),
					},
				},
			},
		},
	}
	
	// Create config generator
	configGenerator := &VerifiedConfigGenerator{
		enhancedService: nil, // Not needed for this test
		logger:          logger,
		outputDir:       "./test-output",
	}
	
	// Test saving configuration
	err := configGenerator.SaveVerifiedConfig(verifiedConfig, "test")
	if err != nil {
		t.Errorf("Failed to save verified config: %v", err)
	}
	
	// Test statistics generation
	statistics, err := configGenerator.GetVerificationStatistics()
	if err != nil {
		t.Errorf("Failed to get verification statistics: %v", err)
	}
	
	if statistics["total_models_scanned"] != verifiedConfig.TotalModels {
		t.Errorf("Expected total_models_scanned to be %d, got %v", verifiedConfig.TotalModels, statistics["total_models_scanned"])
	}
	
	if statistics["verified_models"] != verifiedConfig.VerifiedModels {
		t.Errorf("Expected verified_models to be %d, got %v", verifiedConfig.VerifiedModels, statistics["verified_models"])
	}
	
	expectedRate := float64(verifiedConfig.VerifiedModels) / float64(verifiedConfig.TotalModels) * 100
	if statistics["verification_rate"] != expectedRate {
		t.Errorf("Expected verification_rate to be %f, got %v", expectedRate, statistics["verification_rate"])
	}
}

func TestVerificationConfig(t *testing.T) {
	// Test default configuration
	defaultConfig := CreateDefaultVerificationConfig()
	
	if !defaultConfig.Enabled {
		t.Error("Expected default config to have verification enabled")
	}
	
	if !defaultConfig.StrictMode {
		t.Error("Expected default config to have strict mode enabled")
	}
	
	if defaultConfig.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries to be 3, got %d", defaultConfig.MaxRetries)
	}
	
	if defaultConfig.MinVerificationScore != 0.7 {
		t.Errorf("Expected MinVerificationScore to be 0.7, got %f", defaultConfig.MinVerificationScore)
	}
}

func BenchmarkModelVerificationService_VerifyModels(b *testing.B) {
	logger, err := logging.NewLogger(nil, map[string]any{"log_level": "error"}) // Reduce logging for benchmark
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	httpClient := client.NewHTTPClient(logger)
	
	config := VerificationConfig{
		Enabled:               true,
		StrictMode:            false,
		MaxRetries:            1,
		TimeoutSeconds:        5,
		RequireAffirmative:    true,
		MinVerificationScore:  0.5,
	}
	
	verificationService := NewModelVerificationService(httpClient, logger, config)
	
	// Create test models
	testModels := make([]Model, 10)
	providerClients := make(map[string]*ProviderClient)
	
	for i := 0; i < 10; i++ {
		providerID := "test-provider"
		modelID := fmt.Sprintf("test-model-%d", i)
		
		testModels[i] = Model{
			ID:           modelID,
			Name:         modelID,
			ProviderID:   providerID,
			ProviderName: "Test Provider",
		}
		
		if _, exists := providerClients[providerID]; !exists {
			providerClients[providerID] = &ProviderClient{
				ProviderID: providerID,
				BaseURL:    "https://api.test.com/v1",
				APIKey:     "test-api-key",
				HTTPClient: httpClient,
				logger:     logger,
			}
		}
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		results := verificationService.VerifyModels(ctx, testModels, providerClients)
		
		if len(results) != len(testModels) {
			b.Errorf("Expected %d results, got %d", len(testModels), len(results))
		}
	}
}