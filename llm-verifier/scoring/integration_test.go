package scoring

import (
	"context"
	"fmt"
	"testing"
	"time"

	"llm-verifier/database"
)

// TestScoringSystemIntegration tests the complete scoring system integration
func TestScoringSystemIntegration(t *testing.T) {
	// Setup
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	// Create a test provider
	testProvider := &database.Provider{
		Name:        "Integration Test Provider",
		Endpoint:    "https://api.test.com/v1",
		Description: "Provider for integration testing",
		Website:     "https://test.com",
		IsActive:    true,
	}
	err := db.CreateProvider(testProvider)
	if err != nil {
		t.Fatalf("Failed to create test provider: %v", err)
	}

	// Create a comprehensive test model
	testModel := &database.Model{
		ProviderID:          testProvider.ID,
		ModelID:             "integration-test-model",
		Name:                "Integration Test Model",
		Description:         "A comprehensive test model for integration testing",
		ParameterCount:      int64Ptr(5000000000), // 5B parameters
		ContextWindowTokens: intPtr(64000),
		MaxOutputTokens:     intPtr(4096),
		ReleaseDate:         timePtr(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)),
		TrainingDataCutoff:  timePtr(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		IsMultimodal:        true,
		SupportsVision:      true,
		SupportsReasoning:   true,
		OpenSource:          false,
		Deprecated:          false,
		Tags:                []string{"text", "vision", "reasoning", "integration"},
		LanguageSupport:     []string{"en", "es", "fr", "de"},
		UseCase:             "integration-testing",
		VerificationStatus:  "verified",
		OverallScore:        0, // Will be calculated
		ResponsivenessScore: 7.5,
		CodeCapabilityScore: 8.0,
	}

	err = db.CreateModel(testModel)
	if err != nil {
		t.Fatalf("Failed to create test model: %v", err)
	}

	// Create mock models.dev client
	mockClient := NewMockModelsDevClient()
	logger := setupTestLogger()
	
	// Create comprehensive models.dev model data
	integrationDevModel := ModelsDevModel{
		Provider:            "Integration Test Provider",
		Model:               "Integration Test Model",
		Family:              "integration-test",
		ProviderID:          "integration-test-provider",
		ModelID:             "integration-test-model",
		ToolCall:            true,
		Reasoning:           true,
		Input:               0.01,
		Output:              0.03,
		InputCostPer1M:      10.0,
		OutputCostPer1M:     30.0,
		ReasoningCostPer1M:  30.0,
		CacheReadCostPer1M:  3.0,
		CacheWriteCostPer1M: 7.5,
		ContextLimit:        64000,
		InputLimit:          64000,
		OutputLimit:         4096,
		StructuredOutput:    true,
		Temperature:         true,
		Weights:             "5B",
		Knowledge:           "2024-01",
		ReleaseDate:         "2024-06-15",
		LastUpdated:         "2024-06-20",
		AdditionalData: ModelsDevAdditionalData{
			ParameterCount:     5000000000,
			Architecture:       "transformer",
			TrainingDataCutoff: "2024-01",
			OpenWeights:        false,
			Multimodal:         true,
			Vision:             true,
			Audio:              false,
			Video:              false,
			Languages:          []string{"en", "es", "fr", "de"},
			Tags:               []string{"text", "vision", "reasoning", "integration"},
			License:            "proprietary",
			DocumentationURL:   "https://test.com/docs",
			APIEndpoint:        "https://api.test.com/v1",
		},
	}

	mockClient.AddMockModel(integrationDevModel)

	// Create scoring engine
	engine := NewScoringEngine(db, mockClient, logger)

	// Test comprehensive score calculation
	ctx := context.Background()
	config := DefaultScoringConfig()

	score, err := engine.CalculateComprehensiveScore(ctx, "integration-test-model", config)
	if err != nil {
		t.Fatalf("Failed to calculate comprehensive score: %v", err)
	}

	// Verify score structure and values
	if score == nil {
		t.Fatal("Score should not be nil")
	}

	if score.ModelID != "integration-test-model" {
		t.Errorf("Expected model ID 'integration-test-model', got %s", score.ModelID)
	}

	if score.ModelName != "Integration Test Model" {
		t.Errorf("Expected model name 'Integration Test Model', got %s", score.ModelName)
	}

	if score.OverallScore < 0 || score.OverallScore > 10 {
		t.Errorf("Overall score should be between 0 and 10, got %f", score.OverallScore)
	}

	// Verify score suffix format
	expectedSuffix := fmt.Sprintf("(SC:%.1f)", score.OverallScore)
	if score.ScoreSuffix != expectedSuffix {
		t.Errorf("Expected score suffix %s, got %s", expectedSuffix, score.ScoreSuffix)
	}

	// Verify individual components are reasonable
	components := []struct {
		name  string
		score float64
		min   float64
		max   float64
	}{
		{"Speed", score.Components.SpeedScore, 3.0, 9.0},
		{"Efficiency", score.Components.EfficiencyScore, 4.0, 9.0},
		{"Cost", score.Components.CostScore, 3.0, 8.0},
		{"Capability", score.Components.CapabilityScore, 6.0, 9.0},
		{"Recency", score.Components.RecencyScore, 6.0, 9.0},
	}

	for _, comp := range components {
		if comp.score < comp.min || comp.score > comp.max {
			t.Errorf("%s score should be between %.1f and %.1f, got %.1f", 
				comp.name, comp.min, comp.max, comp.score)
		}
	}

	// Verify model naming integration
	naming := NewModelNaming()
	formattedName := naming.AddScoreSuffix(score.ModelName, score.OverallScore)
	expectedFormattedName := fmt.Sprintf("%s (SC:%.1f)", score.ModelName, score.OverallScore)
	if formattedName != expectedFormattedName {
		t.Errorf("Expected formatted name '%s', got '%s'", expectedFormattedName, formattedName)
	}

	// Log the comprehensive results for debugging
	t.Logf("Integration Test Results:")
	t.Logf("Model: %s", score.ModelName)
	t.Logf("Overall Score: %.1f %s", score.OverallScore, score.ScoreSuffix)
	t.Logf("Components: Speed=%.1f, Efficiency=%.1f, Cost=%.1f, Capability=%.1f, Recency=%.1f",
		score.Components.SpeedScore,
		score.Components.EfficiencyScore,
		score.Components.CostScore,
		score.Components.CapabilityScore,
		score.Components.RecencyScore)
	t.Logf("Last Calculated: %s", score.LastCalculated.Format(time.RFC3339))
}

// TestScoringSystemWithDifferentConfigurations tests scoring with different weight configurations
func TestScoringSystemWithDifferentConfigurations(t *testing.T) {
	// Setup
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	// Create a test provider and model
	testProvider := &database.Provider{
		Name:     "Config Test Provider",
		Endpoint: "https://config.test.com/v1",
		IsActive: true,
	}
	err := db.CreateProvider(testProvider)
	if err != nil {
		t.Fatalf("Failed to create test provider: %v", err)
	}

	testModel := &database.Model{
		ProviderID:       testProvider.ID,
		ModelID:          "config-test-model",
		Name:             "Config Test Model",
		ParameterCount:   int64Ptr(1000000000), // 1B parameters
		IsMultimodal:     true,
		SupportsReasoning: true,
		ReleaseDate:      timePtr(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		OpenSource:       false,
	}

	err = db.CreateModel(testModel)
	if err != nil {
		t.Fatalf("Failed to create test model: %v", err)
	}

	// Create mock client and engine
	mockClient := NewMockModelsDevClient()
	mockClient.AddMockModel(ModelsDevModel{
		ModelID:        "config-test-model",
		InputCostPer1M: 5.0, // Moderate cost
	})
	logger := setupTestLogger()
	engine := NewScoringEngine(db, mockClient, logger)

	ctx := context.Background()

	// Test different weight configurations
	configs := []struct {
		name   string
		config ScoringConfig
	}{
		{
			name: "Speed Focused",
			config: ScoringConfig{
				Weights: ScoreWeights{
					ResponseSpeed:     0.6,
					ModelEfficiency:   0.1,
					CostEffectiveness: 0.1,
					Capability:        0.1,
					Recency:           0.1,
				},
			},
		},
		{
			name: "Cost Focused",
			config: ScoringConfig{
				Weights: ScoreWeights{
					ResponseSpeed:     0.1,
					ModelEfficiency:   0.1,
					CostEffectiveness: 0.6,
					Capability:        0.1,
					Recency:           0.1,
				},
			},
		},
		{
			name: "Capability Focused",
			config: ScoringConfig{
				Weights: ScoreWeights{
					ResponseSpeed:     0.1,
					ModelEfficiency:   0.1,
					CostEffectiveness: 0.1,
					Capability:        0.6,
					Recency:           0.1,
				},
			},
		},
	}

	var previousScore float64
	for i, tc := range configs {
		t.Run(tc.name, func(t *testing.T) {
			score, err := engine.CalculateComprehensiveScore(ctx, "config-test-model", tc.config)
			if err != nil {
				t.Fatalf("Failed to calculate score with %s config: %v", tc.name, err)
			}

			if score == nil {
				t.Fatal("Score should not be nil")
			}

			if score.OverallScore < 0 || score.OverallScore > 10 {
				t.Errorf("Overall score should be between 0 and 10, got %f", score.OverallScore)
			}

			// Verify that different configurations produce different scores
			if i > 0 && score.OverallScore == previousScore {
				t.Errorf("%s configuration produced the same score as previous configuration: %f", 
					tc.name, score.OverallScore)
			}
			previousScore = score.OverallScore

			t.Logf("%s Configuration: Overall Score = %.1f", tc.name, score.OverallScore)
		})
	}
}