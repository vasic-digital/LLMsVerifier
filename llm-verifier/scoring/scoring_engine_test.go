package scoring

import (
	"context"
	"math"
	"testing"
	"time"

	"llm-verifier/database"
)

// MockModelsDevClient is a mock implementation of ModelsDevClient for testing
type MockModelsDevClient struct {
	models map[string]ModelsDevModel
}

func NewMockModelsDevClient() *MockModelsDevClient {
	return &MockModelsDevClient{
		models: make(map[string]ModelsDevModel),
	}
}

func (m *MockModelsDevClient) FetchAllModels(ctx context.Context) (*ModelsDevAPIResponse, error) {
	var response ModelsDevAPIResponse
	for _, model := range m.models {
		response.Models = append(response.Models, model)
	}
	return &response, nil
}

func (m *MockModelsDevClient) FetchModelByID(ctx context.Context, modelID string) (*ModelsDevModel, error) {
	model, exists := m.models[modelID]
	if !exists {
		return nil, fmt.Errorf("model %s not found", modelID)
	}
	return &model, nil
}

func (m *MockModelsDevClient) AddMockModel(model ModelsDevModel) {
	m.models[model.ModelID] = model
}

// TestScoringEngineBasic tests basic scoring engine functionality
func TestScoringEngineBasic(t *testing.T) {
	// Setup
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	mockClient := NewMockModelsDevClient()
	logger := setupTestLogger()
	
	engine := NewScoringEngine(db, mockClient, logger)
	
	// Add a test model
	testModel := createTestModel()
	err := db.CreateModel(testModel)
	if err != nil {
		t.Fatalf("Failed to create test model: %v", err)
	}

	// Add mock models.dev data
	mockDevModel := createTestModelsDevModel()
	mockClient.AddMockModel(mockDevModel)

	// Test score calculation
	ctx := context.Background()
	config := DefaultScoringConfig()
	
	score, err := engine.CalculateComprehensiveScore(ctx, "gpt-4", config)
	if err != nil {
		t.Fatalf("Failed to calculate score: %v", err)
	}

	// Verify score structure
	if score == nil {
		t.Fatal("Score should not be nil")
	}

	if score.ModelID != "gpt-4" {
		t.Errorf("Expected model ID gpt-4, got %s", score.ModelID)
	}

	if score.OverallScore < 0 || score.OverallScore > 10 {
		t.Errorf("Score should be between 0 and 10, got %f", score.OverallScore)
	}

	// Verify components
	if score.Components.SpeedScore < 0 || score.Components.SpeedScore > 10 {
		t.Errorf("Speed score should be between 0 and 10, got %f", score.Components.SpeedScore)
	}

	if score.Components.EfficiencyScore < 0 || score.Components.EfficiencyScore > 10 {
		t.Errorf("Efficiency score should be between 0 and 10, got %f", score.Components.EfficiencyScore)
	}

	if score.Components.CostScore < 0 || score.Components.CostScore > 10 {
		t.Errorf("Cost score should be between 0 and 10, got %f", score.Components.CostScore)
	}

	if score.Components.CapabilityScore < 0 || score.Components.CapabilityScore > 10 {
		t.Errorf("Capability score should be between 0 and 10, got %f", score.Components.CapabilityScore)
	}

	if score.Components.RecencyScore < 0 || score.Components.RecencyScore > 10 {
		t.Errorf("Recency score should be between 0 and 10, got %f", score.Components.RecencyScore)
	}

	// Verify score suffix
	expectedSuffix := fmt.Sprintf("(SC:%.1f)", score.OverallScore)
	if score.ScoreSuffix != expectedSuffix {
		t.Errorf("Expected score suffix %s, got %s", expectedSuffix, score.ScoreSuffix)
	}
}

// TestScoreComponents tests individual score components
func TestScoreComponents(t *testing.T) {
	// Setup
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	mockClient := NewMockModelsDevClient()
	logger := setupTestLogger()
	
	engine := NewScoringEngine(db, mockClient, logger)
	
	// Test different model configurations
	testCases := []struct {
		name           string
		model          database.Model
		devModel       ModelsDevModel
		expectedHigh   []string // Components expected to have high scores
		expectedLow    []string // Components expected to have low scores
	}{
		{
			name: "Fast Expensive Model",
			model: database.Model{
				ModelID:        "fast-model",
				Name:           "Fast Model",
				ParameterCount: int64Ptr(1000000000), // 1B parameters
			},
			devModel: ModelsDevModel{
				ModelID:             "fast-model",
				InputCostPer1M:      10.0,  // Expensive
				OutputCostPer1M:     30.0,  // Expensive
				ContextLimit:        128000,
				ToolCall:            true,
				Reasoning:           false,
				SupportsStructuredOutput: true,
				ReleaseDate:         "2024-01-01",
				LastUpdated:         "2024-01-15",
			},
			expectedHigh: []string{"capability"}, // Should have good capability score
			expectedLow:  []string{"cost"},        // Should have poor cost score
		},
		{
			name: "Slow Cheap Model",
			model: database.Model{
				ModelID:        "slow-model",
				Name:           "Slow Model",
				ParameterCount: int64Ptr(10000000000), // 10B parameters
			},
			devModel: ModelsDevModel{
				ModelID:             "slow-model",
				InputCostPer1M:      0.1,   // Cheap
				OutputCostPer1M:     0.3,   // Cheap
				ContextLimit:        32000, // Smaller context
				ToolCall:            false,
				Reasoning:           false,
				SupportsStructuredOutput: false,
				ReleaseDate:         "2023-01-01", // Older
				LastUpdated:         "2023-06-01",
			},
			expectedHigh: []string{"cost"},        // Should have good cost score
			expectedLow:  []string{"capability"},  // Should have poor capability score
		},
		{
			name: "Efficient Small Model",
			model: database.Model{
				ModelID:        "efficient-model",
				Name:           "Efficient Model",
				ParameterCount: int64Ptr(100000000), // 100M parameters
			},
			devModel: ModelsDevModel{
				ModelID:             "efficient-model",
				InputCostPer1M:      0.5,   // Moderate cost
				OutputCostPer1M:     1.5,   // Moderate cost
				ContextLimit:        64000, // Good context
				ToolCall:            true,
				Reasoning:           true,
				SupportsStructuredOutput: true,
				ReleaseDate:         "2024-06-01", // Recent
				LastUpdated:         "2024-06-15",
			},
			expectedHigh: []string{"efficiency", "recency"}, // Should have good efficiency and recency
			expectedLow:  []string{},                         // Should be balanced
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test model
			err := db.CreateModel(&tc.model)
			if err != nil {
				t.Fatalf("Failed to create test model: %v", err)
			}

			// Add mock data
			mockClient.AddMockModel(tc.devModel)

			// Calculate score
			ctx := context.Background()
			config := DefaultScoringConfig()
			
			score, err := engine.CalculateComprehensiveScore(ctx, tc.model.ModelID, config)
			if err != nil {
				t.Fatalf("Failed to calculate score: %v", err)
			}

			// Check expected high scores
			for _, component := range tc.expectedHigh {
				switch component {
				case "speed":
					if score.Components.SpeedScore < 6.0 {
						t.Errorf("Expected high speed score, got %f", score.Components.SpeedScore)
					}
				case "efficiency":
					if score.Components.EfficiencyScore < 6.0 {
						t.Errorf("Expected high efficiency score, got %f", score.Components.EfficiencyScore)
					}
				case "cost":
					if score.Components.CostScore < 6.0 {
						t.Errorf("Expected high cost score, got %f", score.Components.CostScore)
					}
				case "capability":
					if score.Components.CapabilityScore < 6.0 {
						t.Errorf("Expected high capability score, got %f", score.Components.CapabilityScore)
					}
				case "recency":
					if score.Components.RecencyScore < 6.0 {
						t.Errorf("Expected high recency score, got %f", score.Components.RecencyScore)
					}
				}
			}

			// Check expected low scores
			for _, component := range tc.expectedLow {
				switch component {
				case "speed":
					if score.Components.SpeedScore > 4.0 {
						t.Errorf("Expected low speed score, got %f", score.Components.SpeedScore)
					}
				case "efficiency":
					if score.Components.EfficiencyScore > 4.0 {
						t.Errorf("Expected low efficiency score, got %f", score.Components.EfficiencyScore)
					}
				case "cost":
					if score.Components.CostScore > 4.0 {
						t.Errorf("Expected low cost score, got %f", score.Components.CostScore)
					}
				case "capability":
					if score.Components.CapabilityScore > 4.0 {
						t.Errorf("Expected low capability score, got %f", score.Components.CapabilityScore)
					}
				case "recency":
					if score.Components.RecencyScore > 4.0 {
						t.Errorf("Expected low recency score, got %f", score.Components.RecencyScore)
					}
				}
			}
		})
	}
}

// TestScoreNormalization tests score normalization functions
func TestScoreNormalization(t *testing.T) {
	engine := &ScoringEngine{}

	testCases := []struct {
		name     string
		input    interface{}
		expected float64
		minBound float64
		maxBound float64
	}{
		{
			name:     "Response Time Fast",
			input:    100 * time.Millisecond,
			expected: 9.5, // Should be high score for fast response
			minBound: 8.0,
			maxBound: 10.0,
		},
		{
			name:     "Response Time Slow",
			input:    5000 * time.Millisecond,
			expected: 3.0, // Should be low score for slow response
			minBound: 2.0,
			maxBound: 4.0,
		},
		{
			name:     "High Throughput",
			input:    10.0, // 10 requests per second
			expected: 9.0,  // Should be high score
			minBound: 8.0,
			maxBound: 10.0,
		},
		{
			name:     "Low Throughput",
			input:    0.1, // 0.1 requests per second
			expected: 1.0, // Should be low score
			minBound: 0.5,
			maxBound: 2.0,
		},
		{
			name:     "Low Error Rate",
			input:    0.01, // 1% error rate
			expected: 9.9,  // Should be high score
			minBound: 9.5,
			maxBound: 10.0,
		},
		{
			name:     "High Error Rate",
			input:    0.1, // 10% error rate
			expected: 9.0, // Should be lower score
			minBound: 8.5,
			maxBound: 9.5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var score float64
			
			switch v := tc.input.(type) {
			case time.Duration:
				score = engine.normalizeResponseTime(v)
			case float64:
				if tc.name == "High Throughput" || tc.name == "Low Throughput" {
					score = engine.normalizeThroughput(v)
				} else {
					score = engine.normalizeErrorRate(v)
				}
			}

			if score < tc.minBound || score > tc.maxBound {
				t.Errorf("Expected score between %f and %f, got %f", tc.minBound, tc.maxBound, score)
			}
		})
	}
}

// TestScoringWeights tests different scoring weight configurations
func TestScoringWeights(t *testing.T) {
	// Setup
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	mockClient := NewMockModelsDevClient()
	logger := setupTestLogger()
	
	engine := NewScoringEngine(db, mockClient, logger)
	
	// Create test model
	testModel := createTestModel()
	err := db.CreateModel(testModel)
	if err != nil {
		t.Fatalf("Failed to create test model: %v", err)
	}

	mockClient.AddMockModel(createTestModelsDevModel())

	// Test different weight configurations
	weightConfigs := []struct {
		name    string
		weights struct {
			speed      float64
			efficiency float64
			cost       float64
			capability float64
			recency    float64
		}
		expectedBias string
	}{
		{
			name: "Speed Focused",
			weights: struct {
				speed      float64
				efficiency float64
				cost       float64
				capability float64
				recency    float64
			}{0.5, 0.1, 0.1, 0.2, 0.1},
			expectedBias: "speed",
		},
		{
			name: "Cost Focused",
			weights: struct {
				speed      float64
				efficiency float64
				cost       float64
				capability float64
				recency    float64
			}{0.1, 0.1, 0.6, 0.1, 0.1},
			expectedBias: "cost",
		},
		{
			name: "Capability Focused",
			weights: struct {
				speed      float64
				efficiency float64
				cost       float64
				capability float64
				recency    float64
			}{0.1, 0.1, 0.1, 0.6, 0.1},
			expectedBias: "capability",
		},
	}

	for _, wc := range weightConfigs {
		t.Run(wc.name, func(t *testing.T) {
			config := ScoringConfig{
				Weights: struct {
					Speed      float64 `json:"speed"`
					Efficiency float64 `json:"efficiency"`
					Cost       float64 `json:"cost"`
					Capability float64 `json:"capability"`
					Recency    float64 `json:"recency"`
				}{
					Speed:      wc.weights.speed,
					Efficiency: wc.weights.efficiency,
					Cost:       wc.weights.cost,
					Capability: wc.weights.capability,
					Recency:    wc.weights.recency,
				},
			}

			ctx := context.Background()
			score, err := engine.CalculateComprehensiveScore(ctx, "gpt-4", config)
			if err != nil {
				t.Fatalf("Failed to calculate score: %v", err)
			}

			// Verify weights are applied correctly
			// This is a simplified check - in a real implementation, you'd test with
			// models that have known characteristics for each component
			if score.OverallScore < 0 || score.OverallScore > 10 {
				t.Errorf("Overall score should be between 0 and 10, got %f", score.OverallScore)
			}
		})
	}
}

// TestModelNaming tests the model naming functionality
func TestModelNaming(t *testing.T) {
	naming := NewModelNaming()

	testCases := []struct {
		name           string
		modelName      string
		score          float64
		expectedResult string
		shouldContain  string
	}{
		{
			name:           "Add score suffix to clean name",
			modelName:      "GPT-4",
			score:          8.5,
			expectedResult: "GPT-4 (SC:8.5)",
			shouldContain:  "(SC:8.5)",
		},
		{
			name:           "Update existing score suffix",
			modelName:      "Claude-3 (SC:7.2)",
			score:          8.9,
			expectedResult: "Claude-3 (SC:8.9)",
			shouldContain:  "(SC:8.9)",
		},
		{
			name:           "Handle name with extra spaces",
			modelName:      "  Gemini Pro  ",
			score:          7.3,
			expectedResult: "Gemini Pro (SC:7.3)",
			shouldContain:  "(SC:7.3)",
		},
		{
			name:           "Score with one decimal precision",
			modelName:      "Llama-2",
			score:          6.789,
			expectedResult: "Llama-2 (SC:6.8)",
			shouldContain:  "(SC:6.8)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := naming.AddScoreSuffix(tc.modelName, tc.score)
			
			if result != tc.expectedResult {
				t.Errorf("Expected result '%s', got '%s'", tc.expectedResult, result)
			}
			
			if !contains(result, tc.shouldContain) {
				t.Errorf("Result should contain '%s', got '%s'", tc.shouldContain, result)
			}
		})
	}
}

// TestScoreExtraction tests extracting scores from model names
func TestScoreExtraction(t *testing.T) {
	naming := NewModelNaming()

	testCases := []struct {
		name          string
		modelName     string
		expectedScore float64
		expectedFound bool
	}{
		{
			name:          "Extract score from suffix",
			modelName:     "GPT-4 (SC:8.5)",
			expectedScore: 8.5,
			expectedFound: true,
		},
		{
			name:          "Extract score with spaces",
			modelName:     "Claude-3   (SC:7.2)  ",
			expectedScore: 7.2,
			expectedFound: true,
		},
		{
			name:          "No score suffix",
			modelName:     "Gemini Pro",
			expectedScore: 0,
			expectedFound: false,
		},
		{
			name:          "Invalid score format",
			modelName:     "Model (SC:invalid)",
			expectedScore: 0,
			expectedFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score, found := naming.ExtractScoreFromName(tc.modelName)
			
			if found != tc.expectedFound {
				t.Errorf("Expected found=%v, got %v", tc.expectedFound, found)
			}
			
			if found && math.Abs(score-tc.expectedScore) > 0.01 {
				t.Errorf("Expected score %f, got %f", tc.expectedScore, score)
			}
		})
	}
}

// TestBatchModelNaming tests batch operations for model naming
func TestBatchModelNaming(t *testing.T) {
	naming := NewModelNaming()

	modelScores := map[string]float64{
		"GPT-4":       8.5,
		"Claude-3":    7.8,
		"Gemini Pro":  7.2,
		"Llama-2":     6.9,
	}

	results := naming.BatchUpdateModelNames(modelScores)

	if len(results) != len(modelScores) {
		t.Errorf("Expected %d results, got %d", len(modelScores), len(results))
	}

	for originalName, expectedScore := range modelScores {
		updatedName, exists := results[originalName]
		if !exists {
			t.Errorf("Missing result for model %s", originalName)
			continue
		}

		expectedSuffix := fmt.Sprintf("(SC:%.1f)", expectedScore)
		if !contains(updatedName, expectedSuffix) {
			t.Errorf("Expected updated name to contain '%s', got '%s'", expectedSuffix, updatedName)
		}
	}
}

// TestScoreSuffixFormatter tests the score suffix formatter
func TestScoreSuffixFormatter(t *testing.T) {
	formatter := NewScoreSuffixFormatter()

	testCases := []struct {
		name                string
		score               float64
		includeDescription  bool
		shouldContain       []string
	}{
		{
			name:               "High score with description",
			score:              9.2,
			includeDescription: true,
			shouldContain:      []string{"(SC:9.2)", "Exceptional"},
		},
		{
			name:               "Medium score with description",
			score:              6.5,
			includeDescription: true,
			shouldContain:      []string{"(SC:6.5)", "Good"},
		},
		{
			name:               "Low score with description",
			score:              3.1,
			includeDescription: true,
			shouldContain:      []string{"(SC:3.1)", "Poor"},
		},
		{
			name:               "Score without description",
			score:              7.8,
			includeDescription: false,
			shouldContain:      []string{"(SC:7.8)"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatter.FormatForDisplay(tc.score, tc.includeDescription)
			
			for _, expected := range tc.shouldContain {
				if !contains(result, expected) {
					t.Errorf("Expected result to contain '%s', got '%s'", expected, result)
				}
			}
		})
	}
}

// TestScoreValidation tests score validation
func TestScoreValidation(t *testing.T) {
	naming := NewModelNaming()

	testCases := []struct {
		name     string
		suffix   string
		expected bool
	}{
		{
			name:     "Valid score suffix",
			suffix:   "(SC:8.5)",
			expected: true,
		},
		{
			name:     "Valid score suffix with spaces",
			suffix:   " (SC:7.2) ",
			expected: true,
		},
		{
			name:     "Invalid format",
			suffix:   "SC:8.5",
			expected: false,
		},
		{
			name:     "Invalid score value",
			suffix:   "(SC:invalid)",
			expected: false,
		},
		{
			name:     "Missing parentheses",
			suffix:   "SC:8.5",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := naming.ValidateScoreSuffix(tc.suffix)
			if result != tc.expected {
				t.Errorf("Expected validation result %v, got %v", tc.expected, result)
			}
		})
	}
}

// BenchmarkScoreCalculation benchmarks the score calculation performance
func BenchmarkScoreCalculation(b *testing.B) {
	// Setup
	db := setupBenchmarkDatabase(b)
	defer cleanupBenchmarkDatabase(b, db)

	mockClient := NewMockModelsDevClient()
	logger := setupBenchmarkLogger()
	
	engine := NewScoringEngine(db, mockClient, logger)
	
	// Create test model
	testModel := createTestModel()
	err := db.CreateModel(testModel)
	if err != nil {
		b.Fatalf("Failed to create test model: %v", err)
	}

	mockClient.AddMockModel(createTestModelsDevModel())

	ctx := context.Background()
	config := DefaultScoringConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.CalculateComprehensiveScore(ctx, "gpt-4", config)
		if err != nil {
			b.Fatalf("Failed to calculate score: %v", err)
		}
	}
}

// BenchmarkModelNaming benchmarks model naming operations
func BenchmarkModelNaming(b *testing.B) {
	naming := NewModelNaming()
	modelName := "GPT-4 Turbo"
	score := 8.7

	b.Run("AddScoreSuffix", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = naming.AddScoreSuffix(modelName, score)
		}
	})

	b.Run("ExtractScoreFromName", func(b *testing.B) {
		nameWithScore := naming.AddScoreSuffix(modelName, score)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = naming.ExtractScoreFromName(nameWithScore)
		}
	})

	b.Run("RemoveScoreSuffix", func(b *testing.B) {
		nameWithScore := naming.AddScoreSuffix(modelName, score)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = naming.RemoveScoreSuffix(nameWithScore)
		}
	})
}

// Helper functions for tests

func setupTestDatabase(t *testing.T) *database.Database {
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	return db
}

func cleanupTestDatabase(t *testing.T, db *database.Database) {
	if err := db.Close(); err != nil {
		t.Errorf("Failed to close test database: %v", err)
	}
}

func setupBenchmarkDatabase(b *testing.B) *database.Database {
	db, err := database.New(":memory:")
	if err != nil {
		b.Fatalf("Failed to create benchmark database: %v", err)
	}
	return db
}

func cleanupBenchmarkDatabase(b *testing.B, db *database.Database) {
	if err := db.Close(); err != nil {
		b.Errorf("Failed to close benchmark database: %v", err)
	}
}

func setupTestLogger() *logging.Logger {
	// Return a mock logger or test logger implementation
	return &logging.Logger{}
}

func setupBenchmarkLogger() *logging.Logger {
	// Return a mock logger or benchmark logger implementation
	return &logging.Logger{}
}

func createTestModel() *database.Model {
	return &database.Model{
		ProviderID:          1,
		ModelID:             "gpt-4",
		Name:                "GPT-4",
		Description:         "OpenAI GPT-4 model",
		ParameterCount:      int64Ptr(175000000000), // 175B parameters
		ContextWindowTokens: intPtr(128000),
		MaxOutputTokens:     intPtr(8192),
		ReleaseDate:         timePtr(time.Date(2023, 3, 14, 0, 0, 0, 0, time.UTC)),
		TrainingDataCutoff:  timePtr(time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)),
		IsMultimodal:        true,
		SupportsVision:      true,
		OpenSource:          false,
		Deprecated:          false,
		Tags:                []string{"text", "vision", "reasoning"},
		LanguageSupport:     []string{"en", "es", "fr", "de", "it", "pt", "nl", "ru", "ja", "ko", "zh"},
		UseCase:             "general-purpose",
		VerificationStatus:  "verified",
		OverallScore:        0, // Will be calculated
	}
}

func createTestModelsDevModel() ModelsDevModel {
	return ModelsDevModel{
		Provider:            "OpenAI",
		Model:               "GPT-4",
		Family:              "gpt-4",
		ProviderID:          "openai",
		ModelID:             "gpt-4",
		ToolCall:            true,
		Reasoning:           true,
		Input:               0.03,
		Output:              0.06,
		InputCostPer1M:      30.0,
		OutputCostPer1M:     60.0,
		ReasoningCostPer1M:  60.0,
		CacheReadCostPer1M:  3.0,
		CacheWriteCostPer1M: 7.5,
		ContextLimit:        128000,
		InputLimit:          128000,
		OutputLimit:         8192,
		StructuredOutput:    true,
		Temperature:         true,
		Weights:             "175B",
		Knowledge:           "2023-12",
		ReleaseDate:         "2023-03-14",
		LastUpdated:         "2024-01-15",
		AdditionalData: ModelsDevAdditionalData{
			ParameterCount:     175000000000,
			Architecture:       "transformer",
			TrainingDataCutoff: "2023-12",
			OpenWeights:        false,
			Multimodal:         true,
			Vision:             true,
			Audio:              false,
			Video:              false,
			Languages:          []string{"en", "es", "fr", "de", "it", "pt", "nl", "ru", "ja", "ko", "zh"},
			Tags:               []string{"text", "vision", "reasoning"},
			License:            "proprietary",
			DocumentationURL:   "https://platform.openai.com/docs/models/gpt-4",
			APIEndpoint:        "https://api.openai.com/v1",
		},
	}
}

func int64Ptr(i int64) *int64 {
	return &i
}

func intPtr(i int) *int {
	return &i
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr || 
		   len(s) > len(substr) && s[:len(substr)] == substr ||
		   len(substr) < len(s) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}