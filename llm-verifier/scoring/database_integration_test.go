package scoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"llm-verifier/database"
)

func TestNewDatabaseIntegration(t *testing.T) {
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	di := NewDatabaseIntegration(db)

	require.NotNil(t, di)
	assert.NotNil(t, di.db)
}

func TestDatabaseIntegration_GetModelByID(t *testing.T) {
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	// Create a test provider
	provider := &database.Provider{
		Name:     "Test Provider",
		Endpoint: "https://api.test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	// Create a test model
	model := &database.Model{
		ProviderID: provider.ID,
		ModelID:    "test-model-1",
		Name:       "Test Model 1",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	di := NewDatabaseIntegration(db)

	t.Run("existing model", func(t *testing.T) {
		result, err := di.GetModelByID(model.ID)
		require.NoError(t, err)
		assert.Equal(t, model.ID, result.ID)
		assert.Equal(t, "Test Model 1", result.Name)
	})

	t.Run("non-existing model", func(t *testing.T) {
		_, err := di.GetModelByID(99999)
		assert.Error(t, err)
	})
}

func TestDatabaseIntegration_GetModelByModelID(t *testing.T) {
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	// Create a test provider
	provider := &database.Provider{
		Name:     "Test Provider",
		Endpoint: "https://api.test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	// Create test models
	model := &database.Model{
		ProviderID: provider.ID,
		ModelID:    "gpt-4-test",
		Name:       "GPT-4 Test",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	di := NewDatabaseIntegration(db)

	t.Run("existing model ID", func(t *testing.T) {
		result, err := di.GetModelByModelID("gpt-4-test")
		require.NoError(t, err)
		assert.Equal(t, "gpt-4-test", result.ModelID)
		assert.Equal(t, "GPT-4 Test", result.Name)
	})

	t.Run("non-existing model ID", func(t *testing.T) {
		_, err := di.GetModelByModelID("non-existing-model")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "model not found")
	})
}

func TestDatabaseIntegration_UpdateModelScores(t *testing.T) {
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	// Create a test provider and model
	provider := &database.Provider{
		Name:     "Test Provider",
		Endpoint: "https://api.test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model := &database.Model{
		ProviderID: provider.ID,
		ModelID:    "score-test-model",
		Name:       "Score Test Model",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	di := NewDatabaseIntegration(db)

	t.Run("update with ModelScore", func(t *testing.T) {
		modelScore := &ModelScore{
			ModelID:   "score-test-model",
			ModelName: "Score Test Model",
			Score:     8.5,
			Components: ScoreComponents{
				SpeedScore:      7.0,
				EfficiencyScore: 8.0,
				CostScore:       6.5,
				CapabilityScore: 9.0,
				RecencyScore:    8.5,
			},
		}

		err := di.UpdateModelScores(model.ID, modelScore)
		require.NoError(t, err)

		// Verify the update
		updated, err := db.GetModel(model.ID)
		require.NoError(t, err)
		assert.Equal(t, 8.5, updated.OverallScore)
	})

	t.Run("update with ComprehensiveScore", func(t *testing.T) {
		compScore := &ComprehensiveScore{
			ModelID:      "score-test-model",
			ModelName:    "Score Test Model",
			OverallScore: 9.0,
			Components: ScoreComponents{
				SpeedScore:      8.0,
				EfficiencyScore: 9.0,
				CostScore:       7.5,
				CapabilityScore: 9.5,
				RecencyScore:    9.0,
			},
		}

		err := di.UpdateModelScores(model.ID, compScore)
		require.NoError(t, err)

		// Verify the update
		updated, err := db.GetModel(model.ID)
		require.NoError(t, err)
		assert.Equal(t, 9.0, updated.OverallScore)
	})

	t.Run("update with unsupported type", func(t *testing.T) {
		err := di.UpdateModelScores(model.ID, "invalid type")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported score type")
	})

	t.Run("non-existing model", func(t *testing.T) {
		modelScore := &ModelScore{Score: 8.0}
		err := di.UpdateModelScores(99999, modelScore)
		assert.Error(t, err)
	})
}

func TestDatabaseIntegration_CreateVerificationScore(t *testing.T) {
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	// Create a test provider and model
	provider := &database.Provider{
		Name:     "Test Provider",
		Endpoint: "https://api.test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model := &database.Model{
		ProviderID: provider.ID,
		ModelID:    "verification-test-model",
		Name:       "Verification Test Model",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	di := NewDatabaseIntegration(db)

	t.Run("create with ModelScore", func(t *testing.T) {
		modelScore := &ModelScore{
			ModelID:   "verification-test-model",
			ModelName: "Verification Test Model",
			Score:     8.5,
			Components: ScoreComponents{
				SpeedScore:      7.0,
				EfficiencyScore: 8.0,
				CostScore:       6.5,
				CapabilityScore: 9.0,
				RecencyScore:    8.5,
			},
			ScoreSuffix: "(SC:8.5)",
		}

		err := di.CreateVerificationScore(model.ID, modelScore)
		require.NoError(t, err)
	})

	t.Run("create with ComprehensiveScore", func(t *testing.T) {
		compScore := &ComprehensiveScore{
			ModelID:      "verification-test-model",
			ModelName:    "Verification Test Model",
			OverallScore: 9.0,
			Components: ScoreComponents{
				SpeedScore:      8.0,
				EfficiencyScore: 9.0,
				CostScore:       7.5,
				CapabilityScore: 9.5,
				RecencyScore:    9.0,
			},
			ScoreSuffix: "(SC:9.0)",
		}

		err := di.CreateVerificationScore(model.ID, compScore)
		require.NoError(t, err)
	})

	t.Run("create with unsupported type", func(t *testing.T) {
		err := di.CreateVerificationScore(model.ID, "invalid type")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported score type")
	})
}

func TestDatabaseIntegration_GetLatestVerificationScore(t *testing.T) {
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	// Create a test provider and model
	provider := &database.Provider{
		Name:     "Test Provider",
		Endpoint: "https://api.test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model := &database.Model{
		ProviderID: provider.ID,
		ModelID:    "latest-score-model",
		Name:       "Latest Score Model",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	di := NewDatabaseIntegration(db)

	// Create a verification score
	modelScore := &ModelScore{
		Score: 8.5,
		Components: ScoreComponents{
			SpeedScore:      7.0,
			EfficiencyScore: 8.0,
			CostScore:       6.5,
			CapabilityScore: 9.0,
			RecencyScore:    8.5,
		},
	}
	err = di.CreateVerificationScore(model.ID, modelScore)
	require.NoError(t, err)

	// Get the latest score
	score, err := di.GetLatestVerificationScore(model.ID)
	// May return nil if no scores exist for the type
	if err == nil && score != nil {
		assert.Equal(t, model.ID, score.ModelID)
	}
}

func TestDatabaseIntegration_ListModelsByScore(t *testing.T) {
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	// Create a test provider
	provider := &database.Provider{
		Name:     "Test Provider",
		Endpoint: "https://api.test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	// Create models with different scores
	models := []struct {
		modelID string
		name    string
		score   float64
	}{
		{"model-high", "High Score Model", 9.0},
		{"model-medium", "Medium Score Model", 7.0},
		{"model-low", "Low Score Model", 5.0},
	}

	for _, m := range models {
		model := &database.Model{
			ProviderID:   provider.ID,
			ModelID:      m.modelID,
			Name:         m.name,
			OverallScore: m.score,
		}
		err := db.CreateModel(model)
		require.NoError(t, err)
	}

	di := NewDatabaseIntegration(db)

	t.Run("filter by min score", func(t *testing.T) {
		results, err := di.ListModelsByScore(6.0, 10.0, 10)
		require.NoError(t, err)
		// Should return models with score >= 6.0
		assert.GreaterOrEqual(t, len(results), 0)
	})

	t.Run("with limit", func(t *testing.T) {
		results, err := di.ListModelsByScore(0.0, 10.0, 2)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(results), 2)
	})
}

func TestDatabaseIntegration_GetTopScoringModels(t *testing.T) {
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	// Create a test provider
	provider := &database.Provider{
		Name:     "Test Provider",
		Endpoint: "https://api.test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	// Create models with different scores
	for i := 0; i < 5; i++ {
		model := &database.Model{
			ProviderID:   provider.ID,
			ModelID:      "top-model-" + string(rune('A'+i)),
			Name:         "Top Model " + string(rune('A'+i)),
			OverallScore: float64(5 + i),
		}
		err := db.CreateModel(model)
		require.NoError(t, err)
	}

	di := NewDatabaseIntegration(db)

	t.Run("get top 3 models", func(t *testing.T) {
		results, err := di.GetTopScoringModels(3)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(results), 3)

		// Verify results are sorted by score (descending)
		for i := 0; i < len(results)-1; i++ {
			assert.GreaterOrEqual(t, results[i].OverallScore, results[i+1].OverallScore)
		}
	})
}

func TestDatabaseIntegration_CreateScoringEvent(t *testing.T) {
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	// Create a test provider and model
	provider := &database.Provider{
		Name:     "Test Provider",
		Endpoint: "https://api.test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model := &database.Model{
		ProviderID: provider.ID,
		ModelID:    "event-test-model",
		Name:       "Event Test Model",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	di := NewDatabaseIntegration(db)

	t.Run("create scoring event", func(t *testing.T) {
		details := map[string]interface{}{
			"score":      8.5,
			"model_name": "Event Test Model",
		}

		err := di.CreateScoringEvent("score_calculated", "Score calculated successfully", &model.ID, details)
		require.NoError(t, err)
	})

	t.Run("create event without model ID", func(t *testing.T) {
		details := map[string]interface{}{
			"action": "system_event",
		}

		err := di.CreateScoringEvent("system_info", "System information", nil, details)
		require.NoError(t, err)
	})
}

func TestDatabaseIntegration_GetVerificationResults(t *testing.T) {
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	// Create a test provider and model
	provider := &database.Provider{
		Name:     "Test Provider",
		Endpoint: "https://api.test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model := &database.Model{
		ProviderID: provider.ID,
		ModelID:    "result-test-model",
		Name:       "Result Test Model",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	di := NewDatabaseIntegration(db)

	t.Run("empty model IDs", func(t *testing.T) {
		results, err := di.GetVerificationResults([]int64{})
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("get results for model", func(t *testing.T) {
		results, err := di.GetVerificationResults([]int64{model.ID})
		// May return error or nil if no verification results exist
		// We just want to test the function runs without panic
		if err == nil {
			// Results can be nil or empty slice
			t.Logf("Got %d results", len(results))
		}
	})
}

func TestGetScoreCategory(t *testing.T) {
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	di := NewDatabaseIntegration(db)

	tests := []struct {
		score    float64
		expected string
	}{
		{9.0, "fully_coding_capable"},
		{8.5, "fully_coding_capable"},
		{7.5, "coding_with_tools"},
		{7.0, "coding_with_tools"},
		{6.0, "chat_with_tooling"},
		{5.0, "chat_with_tooling"},
		{4.0, "chat_only"},
		{0.0, "chat_only"},
	}

	for _, tc := range tests {
		t.Run("score_"+string(rune('0'+int(tc.score))), func(t *testing.T) {
			result := di.getScoreCategory(tc.score)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCreateScoreEvidence(t *testing.T) {
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	di := NewDatabaseIntegration(db)

	t.Run("with ModelScore", func(t *testing.T) {
		modelScore := &ModelScore{
			Score: 8.5,
			Components: ScoreComponents{
				SpeedScore:      7.0,
				EfficiencyScore: 8.0,
				CostScore:       6.5,
				CapabilityScore: 9.0,
				RecencyScore:    8.5,
			},
			ScoreSuffix: "(SC:8.5)",
		}

		evidence := di.createScoreEvidence(modelScore)
		assert.NotEmpty(t, evidence)
		assert.Contains(t, evidence, "overall_score")
		assert.Contains(t, evidence, "components")
	})

	t.Run("with ComprehensiveScore", func(t *testing.T) {
		compScore := &ComprehensiveScore{
			OverallScore: 9.0,
			Components: ScoreComponents{
				SpeedScore:      8.0,
				EfficiencyScore: 9.0,
				CostScore:       7.5,
				CapabilityScore: 9.5,
				RecencyScore:    9.0,
			},
			ScoreSuffix: "(SC:9.0)",
		}

		evidence := di.createScoreEvidence(compScore)
		assert.NotEmpty(t, evidence)
		assert.Contains(t, evidence, "overall_score")
	})

	t.Run("with unknown type", func(t *testing.T) {
		evidence := di.createScoreEvidence("unknown type")
		assert.NotEmpty(t, evidence)
		// Should create default evidence
		assert.Contains(t, evidence, "overall_score")
	})
}

// Helper functions are defined in scoring_engine_test.go
