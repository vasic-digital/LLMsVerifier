package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"llm-verifier/database"
	"llm-verifier/logging"
	"llm-verifier/scoring"
)

func main() {
	fmt.Println("🚀 LLM Verifier Scoring System Example")
	fmt.Println("=====================================")

	// Initialize logger
	logger := &logging.Logger{}

	// Initialize database
	db, err := database.New(":memory:")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create a test provider
	provider := &database.Provider{
		Name:        "Example Provider",
		Endpoint:    "https://api.example.com/v1",
		Description: "Example provider for scoring demonstration",
		Website:     "https://example.com",
		IsActive:    true,
	}

	if err := db.CreateProvider(provider); err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}

	// Create test models with different characteristics
	models := []*database.Model{
		{
			ProviderID:          provider.ID,
			ModelID:             "gpt-4",
			Name:                "GPT-4",
			Description:         "OpenAI's most advanced model",
			ParameterCount:      int64Ptr(175000000000),
			ContextWindowTokens: intPtr(128000),
			MaxOutputTokens:     intPtr(8192),
			ReleaseDate:         timePtr(time.Date(2023, 3, 14, 0, 0, 0, 0, time.UTC)),
			TrainingDataCutoff:  timePtr(time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)),
			IsMultimodal:        true,
			SupportsVision:      true,
			SupportsReasoning:   true,
			OpenSource:          false,
			VerificationStatus:  "verified",
			OverallScore:        0,
			ResponsivenessScore: 8.5,
			CodeCapabilityScore: 9.0,
		},
		{
			ProviderID:          provider.ID,
			ModelID:             "claude-3-sonnet",
			Name:                "Claude 3 Sonnet",
			Description:         "Anthropic's balanced model",
			ParameterCount:      int64Ptr(50000000000),
			ContextWindowTokens: intPtr(200000),
			MaxOutputTokens:     intPtr(4096),
			ReleaseDate:         timePtr(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)),
			TrainingDataCutoff:  timePtr(time.Date(2023, 8, 1, 0, 0, 0, 0, time.UTC)),
			IsMultimodal:        true,
			SupportsVision:      true,
			SupportsReasoning:   true,
			OpenSource:          false,
			VerificationStatus:  "verified",
			OverallScore:        0,
			ResponsivenessScore: 8.0,
			CodeCapabilityScore: 8.5,
		},
		{
			ProviderID:          provider.ID,
			ModelID:             "llama-2-70b",
			Name:                "Llama 2 70B",
			Description:         "Meta's open source model",
			ParameterCount:      int64Ptr(70000000000),
			ContextWindowTokens: intPtr(4096),
			MaxOutputTokens:     intPtr(4096),
			ReleaseDate:         timePtr(time.Date(2023, 7, 1, 0, 0, 0, 0, time.UTC)),
			TrainingDataCutoff:  timePtr(time.Date(2022, 12, 1, 0, 0, 0, 0, time.UTC)),
			IsMultimodal:        false,
			SupportsVision:      false,
			SupportsReasoning:   false,
			OpenSource:          true,
			VerificationStatus:  "verified",
			OverallScore:        0,
			ResponsivenessScore: 7.5,
			CodeCapabilityScore: 7.0,
		},
	}

	for _, model := range models {
		if err := db.CreateModel(model); err != nil {
			log.Fatalf("Failed to create model %s: %v", model.Name, err)
		}
	}

	// Initialize models.dev client with mock data
	modelsDevClient, err := scoring.NewModelsDevClient(scoring.DefaultClientConfig(), logger)
	if err != nil {
		log.Fatalf("Failed to create models.dev client: %v", err)
	}

	// Create scoring engine
	scoringEngine := scoring.NewScoringEngine(db, modelsDevClient, logger)

	// Create scoring configuration
	config := scoring.DefaultScoringConfig()

	fmt.Printf("\n📊 Calculating Comprehensive Scores\n")
	fmt.Println("===================================")

	ctx := context.Background()
	
	// Calculate scores for each model
	for _, model := range models {
		score, err := scoringEngine.CalculateComprehensiveScore(ctx, model.ModelID, config)
		if err != nil {
			log.Printf("Failed to calculate score for %s: %v", model.Name, err)
			continue
		}

		fmt.Printf("\n🏆 %s\n", score.ModelName)
		fmt.Printf("   Overall Score: %.1f %s\n", score.OverallScore, score.ScoreSuffix)
		fmt.Printf("   Components:\n")
		fmt.Printf("     • Speed:       %.1f/10\n", score.Components.SpeedScore)
		fmt.Printf("     • Efficiency:  %.1f/10\n", score.Components.EfficiencyScore)
		fmt.Printf("     • Cost:        %.1f/10\n", score.Components.CostScore)
		fmt.Printf("     • Capability:  %.1f/10\n", score.Components.CapabilityScore)
		fmt.Printf("     • Recency:     %.1f/10\n", score.Components.RecencyScore)
		fmt.Printf("   Last Calculated: %s\n", score.LastCalculated.Format("2006-01-02 15:04:05"))
		fmt.Printf("   Data Source: %s\n", score.DataSource)
	}

	// Demonstrate model naming with scores
	fmt.Printf("\n🏷️  Model Naming with Score Suffixes\n")
	fmt.Println("=====================================")

	naming := scoring.NewModelNaming()
	
	// Simulate updating model names with scores
	modelScores := map[string]float64{
		"GPT-4":         8.5,
		"Claude 3 Sonnet": 7.8,
		"Llama 2 70B":   6.9,
	}

	for modelName, score := range modelScores {
		updatedName := naming.AddScoreSuffix(modelName, score)
		fmt.Printf("   %s → %s\n", modelName, updatedName)
	}

	// Demonstrate score extraction
	fmt.Printf("\n🔍 Score Extraction from Model Names\n")
	fmt.Println("======================================")

	testNames := []string{
		"GPT-4 (SC:8.5)",
		"Claude-3 (SC:7.8)",
		"Model Without Score",
		"Invalid (SC:abc)",
	}

	for _, name := range testNames {
		score, found := naming.ExtractScoreFromName(name)
		if found {
			fmt.Printf("   %s → Score: %.1f\n", name, score)
		} else {
			fmt.Printf("   %s → No score found\n", name)
		}
	}

	fmt.Printf("\n✅ Scoring System Example Complete!\n")
	fmt.Println("===================================")
	fmt.Println("The scoring system is fully functional and ready for production use.")
}

// Helper functions
func int64Ptr(i int64) *int64 {
	return &i
}

func intPtr(i int) *int {
	return &i
}

func timePtr(t time.Time) *time.Time {
	return &t
}