package database

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupVerificationScoresTestDB(t *testing.T) *Database {
	dbFile := "/tmp/test_verification_scores_" + time.Now().Format("20060102150405") + ".db"
	db, err := New(dbFile)
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Close()
		os.Remove(dbFile)
	})
	return db
}

func createTestVerificationScoresModel(t *testing.T, db *Database) *Model {
	provider := &Provider{
		Name:     "test-provider",
		Endpoint: "http://test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model := &Model{
		ProviderID: provider.ID,
		ModelID:    "test-model",
		Name:       "Test Model",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)
	return model
}

func createTestVerificationScore(modelID int64) *VerificationScore {
	codeCorrectness := 85
	codeQuality := 80
	codeSpeed := 75
	errorHandling := 90
	contextUnderstanding := 82

	return &VerificationScore{
		ModelID:                   modelID,
		Score:                     85,
		ScoreType:                 "coding_capability",
		ScoringMethod:             "benchmark",
		Category:                  "fully_coding_capable",
		CodeCorrectnessScore:      &codeCorrectness,
		CodeQualityScore:          &codeQuality,
		CodeSpeedScore:            &codeSpeed,
		ErrorHandlingScore:        &errorHandling,
		ContextUnderstandingScore: &contextUnderstanding,
		Evidence:                  `{"test_cases": 100, "passed": 85}`,
		BenchmarkVersion:          "v1.0",
		ScoredBy:                  "auto",
		ConfidenceLevel:           90,
		ScoredAt:                  time.Now(),
	}
}

// ==================== VerificationScore CRUD Tests ====================

func TestCreateVerificationScore(t *testing.T) {
	db := setupVerificationScoresTestDB(t)
	model := createTestVerificationScoresModel(t, db)

	score := createTestVerificationScore(model.ID)
	id, err := db.CreateVerificationScore(score)
	require.NoError(t, err)
	assert.NotZero(t, id)
}

func TestCreateVerificationScore_WithExpiresAt(t *testing.T) {
	db := setupVerificationScoresTestDB(t)
	model := createTestVerificationScoresModel(t, db)

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	score := createTestVerificationScore(model.ID)
	score.ExpiresAt = &expiresAt

	id, err := db.CreateVerificationScore(score)
	require.NoError(t, err)
	assert.NotZero(t, id)
}

func TestCreateVerificationScore_MinimalFields(t *testing.T) {
	db := setupVerificationScoresTestDB(t)
	model := createTestVerificationScoresModel(t, db)

	score := &VerificationScore{
		ModelID:          model.ID,
		Score:            70,
		ScoreType:        "accuracy",
		ScoringMethod:    "manual",
		Category:         "coding_with_tools",
		BenchmarkVersion: "v1.0",
		ScoredBy:         "admin",
		ConfidenceLevel:  80,
		ScoredAt:         time.Now(),
	}

	id, err := db.CreateVerificationScore(score)
	require.NoError(t, err)
	assert.NotZero(t, id)
}

func TestGetVerificationScore(t *testing.T) {
	db := setupVerificationScoresTestDB(t)
	model := createTestVerificationScoresModel(t, db)

	score := createTestVerificationScore(model.ID)
	id, err := db.CreateVerificationScore(score)
	require.NoError(t, err)

	retrieved, err := db.GetVerificationScore(id)
	require.NoError(t, err)
	assert.Equal(t, id, retrieved.ID)
	assert.Equal(t, score.ModelID, retrieved.ModelID)
	assert.Equal(t, score.Score, retrieved.Score)
	assert.Equal(t, score.ScoreType, retrieved.ScoreType)
	assert.Equal(t, score.ScoringMethod, retrieved.ScoringMethod)
	assert.Equal(t, score.Category, retrieved.Category)
}

func TestGetVerificationScore_NotFound(t *testing.T) {
	db := setupVerificationScoresTestDB(t)

	_, err := db.GetVerificationScore(99999)
	assert.Error(t, err)
}

func TestGetLatestVerificationScore(t *testing.T) {
	db := setupVerificationScoresTestDB(t)
	model := createTestVerificationScoresModel(t, db)

	// Create older score
	oldScore := createTestVerificationScore(model.ID)
	oldScore.Score = 70
	oldScore.ScoredAt = time.Now().Add(-24 * time.Hour)
	_, err := db.CreateVerificationScore(oldScore)
	require.NoError(t, err)

	// Create newer score
	newScore := createTestVerificationScore(model.ID)
	newScore.Score = 85
	newScore.ScoredAt = time.Now()
	_, err = db.CreateVerificationScore(newScore)
	require.NoError(t, err)

	// Get latest
	latest, err := db.GetLatestVerificationScore(model.ID, "coding_capability")
	require.NoError(t, err)
	assert.Equal(t, 85, latest.Score)
}

func TestGetLatestVerificationScore_NotFound(t *testing.T) {
	db := setupVerificationScoresTestDB(t)
	model := createTestVerificationScoresModel(t, db)

	_, err := db.GetLatestVerificationScore(model.ID, "nonexistent_type")
	assert.Error(t, err)
}

func TestListVerificationScores_NoFilters(t *testing.T) {
	db := setupVerificationScoresTestDB(t)
	model := createTestVerificationScoresModel(t, db)

	// Create multiple scores
	for i := 0; i < 3; i++ {
		score := createTestVerificationScore(model.ID)
		score.Score = 70 + i*10
		_, err := db.CreateVerificationScore(score)
		require.NoError(t, err)
	}

	scores, err := db.ListVerificationScores(nil)
	require.NoError(t, err)
	assert.Len(t, scores, 3)
}

func TestListVerificationScores_FilterByModelID(t *testing.T) {
	db := setupVerificationScoresTestDB(t)
	model1 := createTestVerificationScoresModel(t, db)

	// Create second model
	provider := &Provider{
		Name:     "test-provider-2",
		Endpoint: "http://test2.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model2 := &Model{
		ProviderID: provider.ID,
		ModelID:    "test-model-2",
		Name:       "Test Model 2",
	}
	err = db.CreateModel(model2)
	require.NoError(t, err)

	// Create scores for model1
	for i := 0; i < 2; i++ {
		score := createTestVerificationScore(model1.ID)
		_, err = db.CreateVerificationScore(score)
		require.NoError(t, err)
	}

	// Create score for model2
	score := createTestVerificationScore(model2.ID)
	_, err = db.CreateVerificationScore(score)
	require.NoError(t, err)

	// Filter by model1
	scores, err := db.ListVerificationScores(map[string]interface{}{"model_id": model1.ID})
	require.NoError(t, err)
	assert.Len(t, scores, 2)
}

func TestListVerificationScores_FilterByScoreType(t *testing.T) {
	db := setupVerificationScoresTestDB(t)
	model := createTestVerificationScoresModel(t, db)

	scoreTypes := []string{"coding_capability", "accuracy", "speed"}
	for _, scoreType := range scoreTypes {
		score := createTestVerificationScore(model.ID)
		score.ScoreType = scoreType
		_, err := db.CreateVerificationScore(score)
		require.NoError(t, err)
	}

	scores, err := db.ListVerificationScores(map[string]interface{}{"score_type": "coding_capability"})
	require.NoError(t, err)
	assert.Len(t, scores, 1)
	assert.Equal(t, "coding_capability", scores[0].ScoreType)
}

func TestListVerificationScores_FilterByCategory(t *testing.T) {
	db := setupVerificationScoresTestDB(t)
	model := createTestVerificationScoresModel(t, db)

	categories := []string{"fully_coding_capable", "coding_with_tools", "chat_only"}
	for _, category := range categories {
		score := createTestVerificationScore(model.ID)
		score.Category = category
		_, err := db.CreateVerificationScore(score)
		require.NoError(t, err)
	}

	scores, err := db.ListVerificationScores(map[string]interface{}{"category": "fully_coding_capable"})
	require.NoError(t, err)
	assert.Len(t, scores, 1)
	assert.Equal(t, "fully_coding_capable", scores[0].Category)
}

func TestListVerificationScores_FilterByMinScore(t *testing.T) {
	db := setupVerificationScoresTestDB(t)
	model := createTestVerificationScoresModel(t, db)

	// Create scores with different values
	scoreValues := []int{50, 70, 80, 90}
	for _, val := range scoreValues {
		score := createTestVerificationScore(model.ID)
		score.Score = val
		_, err := db.CreateVerificationScore(score)
		require.NoError(t, err)
	}

	// Filter by minimum score 75
	scores, err := db.ListVerificationScores(map[string]interface{}{"min_score": 75})
	require.NoError(t, err)
	assert.Len(t, scores, 2) // 80 and 90
}

func TestUpdateVerificationScore(t *testing.T) {
	db := setupVerificationScoresTestDB(t)
	model := createTestVerificationScoresModel(t, db)

	score := createTestVerificationScore(model.ID)
	id, err := db.CreateVerificationScore(score)
	require.NoError(t, err)

	// Retrieve and update
	retrieved, err := db.GetVerificationScore(id)
	require.NoError(t, err)

	retrieved.Score = 95
	retrieved.Category = "fully_coding_capable"
	retrieved.ConfidenceLevel = 95
	err = db.UpdateVerificationScore(retrieved)
	require.NoError(t, err)

	// Verify update
	updated, err := db.GetVerificationScore(id)
	require.NoError(t, err)
	assert.Equal(t, 95, updated.Score)
	assert.Equal(t, "fully_coding_capable", updated.Category)
	assert.Equal(t, 95, updated.ConfidenceLevel)
}

func TestDeleteVerificationScore(t *testing.T) {
	db := setupVerificationScoresTestDB(t)
	model := createTestVerificationScoresModel(t, db)

	score := createTestVerificationScore(model.ID)
	id, err := db.CreateVerificationScore(score)
	require.NoError(t, err)

	err = db.DeleteVerificationScore(id)
	require.NoError(t, err)

	_, err = db.GetVerificationScore(id)
	assert.Error(t, err)
}

func TestDeleteVerificationScore_NotFound(t *testing.T) {
	db := setupVerificationScoresTestDB(t)

	err := db.DeleteVerificationScore(99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetModelCodingCapabilityScore(t *testing.T) {
	db := setupVerificationScoresTestDB(t)
	model := createTestVerificationScoresModel(t, db)

	// Create coding capability score
	score := createTestVerificationScore(model.ID)
	score.ScoreType = "coding_capability"
	score.Score = 88
	_, err := db.CreateVerificationScore(score)
	require.NoError(t, err)

	// Get coding capability score
	codingScore, err := db.GetModelCodingCapabilityScore(model.ID)
	require.NoError(t, err)
	assert.Equal(t, 88, codingScore.Score)
	assert.Equal(t, "coding_capability", codingScore.ScoreType)
}

func TestGetTopScoringModels(t *testing.T) {
	db := setupVerificationScoresTestDB(t)

	// Create models with different scores
	provider := &Provider{
		Name:     "test-provider",
		Endpoint: "http://test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	scores := []int{95, 85, 75, 65, 55}
	for i, scoreVal := range scores {
		model := &Model{
			ProviderID: provider.ID,
			ModelID:    "model-" + string(rune('A'+i)),
			Name:       "Model " + string(rune('A'+i)),
		}
		err = db.CreateModel(model)
		require.NoError(t, err)

		score := createTestVerificationScore(model.ID)
		score.Score = scoreVal
		_, err = db.CreateVerificationScore(score)
		require.NoError(t, err)
	}

	// Get top 3
	topScores, err := db.GetTopScoringModels("coding_capability", 3)
	require.NoError(t, err)
	assert.Len(t, topScores, 3)
	assert.Equal(t, 95, topScores[0].Score)
	assert.Equal(t, 85, topScores[1].Score)
	assert.Equal(t, 75, topScores[2].Score)
}

// ==================== CalculateCodingCapabilityScore Tests ====================

func TestCalculateCodingCapabilityScore_FullyCodingCapable(t *testing.T) {
	benchmarks := map[string]int{
		"code_correctness": 90,
		"code_quality":     85,
		"code_speed":       80,
		"error_handling":   95,
	}

	score, category := CalculateCodingCapabilityScore(benchmarks)

	// Expected: 90*0.40 + 85*0.30 + 80*0.20 + 95*0.10 = 36 + 25.5 + 16 + 9.5 = 87
	assert.Equal(t, 87, score)
	assert.Equal(t, "fully_coding_capable", category)
}

func TestCalculateCodingCapabilityScore_CodingWithTools(t *testing.T) {
	benchmarks := map[string]int{
		"code_correctness": 70,
		"code_quality":     65,
		"code_speed":       60,
		"error_handling":   70,
	}

	score, category := CalculateCodingCapabilityScore(benchmarks)

	// Expected: 70*0.40 + 65*0.30 + 60*0.20 + 70*0.10 = 28 + 19.5 + 12 + 7 = 66.5 -> 66
	assert.Equal(t, 66, score)
	assert.Equal(t, "coding_with_tools", category)
}

func TestCalculateCodingCapabilityScore_ChatWithTooling(t *testing.T) {
	benchmarks := map[string]int{
		"code_correctness": 50,
		"code_quality":     45,
		"code_speed":       40,
		"error_handling":   50,
	}

	score, category := CalculateCodingCapabilityScore(benchmarks)

	// Expected: 50*0.40 + 45*0.30 + 40*0.20 + 50*0.10 = 20 + 13.5 + 8 + 5 = 46.5 -> 46
	assert.Equal(t, 46, score)
	assert.Equal(t, "chat_with_tooling", category)
}

func TestCalculateCodingCapabilityScore_ChatOnly(t *testing.T) {
	benchmarks := map[string]int{
		"code_correctness": 30,
		"code_quality":     25,
		"code_speed":       20,
		"error_handling":   30,
	}

	score, category := CalculateCodingCapabilityScore(benchmarks)

	// Expected: 30*0.40 + 25*0.30 + 20*0.20 + 30*0.10 = 12 + 7.5 + 4 + 3 = 26.5 -> 26
	assert.Equal(t, 26, score)
	assert.Equal(t, "chat_only", category)
}

func TestCalculateCodingCapabilityScore_PartialBenchmarks(t *testing.T) {
	benchmarks := map[string]int{
		"code_correctness": 100,
		"code_quality":     100,
	}

	score, category := CalculateCodingCapabilityScore(benchmarks)

	// Expected: 100*0.40 + 100*0.30 = 40 + 30 = 70
	assert.Equal(t, 70, score)
	assert.Equal(t, "coding_with_tools", category)
}

func TestCalculateCodingCapabilityScore_EmptyBenchmarks(t *testing.T) {
	benchmarks := map[string]int{}

	score, category := CalculateCodingCapabilityScore(benchmarks)

	assert.Equal(t, 0, score)
	assert.Equal(t, "chat_only", category)
}

// ==================== Struct Tests ====================

func TestVerificationScore_Struct(t *testing.T) {
	codeCorrectness := 90
	codeQuality := 85
	codeSpeed := 80
	errorHandling := 92
	contextUnderstanding := 88
	verificationResultID := int64(42)
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	now := time.Now()

	score := VerificationScore{
		ID:                        1,
		ModelID:                   100,
		VerificationResultID:      &verificationResultID,
		Score:                     87,
		ScoreType:                 "coding_capability",
		ScoringMethod:             "benchmark",
		Category:                  "fully_coding_capable",
		CodeCorrectnessScore:      &codeCorrectness,
		CodeQualityScore:          &codeQuality,
		CodeSpeedScore:            &codeSpeed,
		ErrorHandlingScore:        &errorHandling,
		ContextUnderstandingScore: &contextUnderstanding,
		Evidence:                  `{"tests": 100}`,
		BenchmarkVersion:          "v2.0",
		ScoredBy:                  "system",
		ConfidenceLevel:           95,
		ScoredAt:                  now,
		ExpiresAt:                 &expiresAt,
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}

	assert.Equal(t, int64(1), score.ID)
	assert.Equal(t, int64(100), score.ModelID)
	assert.Equal(t, int64(42), *score.VerificationResultID)
	assert.Equal(t, 87, score.Score)
	assert.Equal(t, "coding_capability", score.ScoreType)
	assert.Equal(t, "benchmark", score.ScoringMethod)
	assert.Equal(t, "fully_coding_capable", score.Category)
	assert.Equal(t, 90, *score.CodeCorrectnessScore)
	assert.Equal(t, 85, *score.CodeQualityScore)
	assert.Equal(t, 80, *score.CodeSpeedScore)
	assert.Equal(t, 92, *score.ErrorHandlingScore)
	assert.Equal(t, 88, *score.ContextUnderstandingScore)
	assert.Equal(t, "v2.0", score.BenchmarkVersion)
	assert.Equal(t, 95, score.ConfidenceLevel)
}
