package scoring

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestScoreComponents_Structure(t *testing.T) {
	sc := ScoreComponents{
		SpeedScore:      8.5,
		EfficiencyScore: 7.8,
		CostScore:       9.0,
		CapabilityScore: 8.0,
		RecencyScore:    6.5,
	}

	assert.Equal(t, 8.5, sc.SpeedScore)
	assert.Equal(t, 7.8, sc.EfficiencyScore)
	assert.Equal(t, 9.0, sc.CostScore)
	assert.Equal(t, 8.0, sc.CapabilityScore)
	assert.Equal(t, 6.5, sc.RecencyScore)
}

func TestComprehensiveScore_Structure(t *testing.T) {
	now := time.Now()
	cs := ComprehensiveScore{
		ModelID:      "gpt-4",
		ModelName:    "GPT-4",
		OverallScore: 8.5,
		ScoreSuffix:  "(SC:8.5)",
		Components: ScoreComponents{
			SpeedScore:      8.0,
			EfficiencyScore: 8.5,
			CostScore:       7.0,
			CapabilityScore: 9.0,
			RecencyScore:    8.5,
		},
		LastCalculated:  now,
		CalculationHash: "abc123",
		DataSource:      "models.dev",
	}

	assert.Equal(t, "gpt-4", cs.ModelID)
	assert.Equal(t, "GPT-4", cs.ModelName)
	assert.Equal(t, 8.5, cs.OverallScore)
	assert.Equal(t, "(SC:8.5)", cs.ScoreSuffix)
	assert.Equal(t, "abc123", cs.CalculationHash)
	assert.Equal(t, "models.dev", cs.DataSource)
}

func TestModelScore_Structure(t *testing.T) {
	now := time.Now()
	validUntil := now.Add(24 * time.Hour)

	ms := ModelScore{
		ID:                 1,
		ModelID:            "claude-3",
		ModelName:          "Claude 3",
		Score:              9.0,
		ScoreSuffix:        "(SC:9.0)",
		Components:         ScoreComponents{SpeedScore: 9.0},
		CalculationHash:    "def456",
		CalculationDetails: "Speed weighted highest",
		LastCalculated:     now,
		ValidUntil:         &validUntil,
		IsActive:           true,
		CreatedAt:          now,
		UpdatedAt:          now,
		DataSource:         "internal",
	}

	assert.Equal(t, int64(1), ms.ID)
	assert.Equal(t, "claude-3", ms.ModelID)
	assert.Equal(t, 9.0, ms.Score)
	assert.True(t, ms.IsActive)
	assert.NotNil(t, ms.ValidUntil)
}

func TestScoreWeights_Structure(t *testing.T) {
	sw := ScoreWeights{
		ResponseSpeed:     0.25,
		ModelEfficiency:   0.20,
		CostEffectiveness: 0.25,
		Capability:        0.20,
		Recency:           0.10,
	}

	assert.Equal(t, 0.25, sw.ResponseSpeed)
	assert.Equal(t, 0.20, sw.ModelEfficiency)
	assert.Equal(t, 0.25, sw.CostEffectiveness)
	assert.Equal(t, 0.20, sw.Capability)
	assert.Equal(t, 0.10, sw.Recency)

	// Verify weights sum to 1.0
	total := sw.ResponseSpeed + sw.ModelEfficiency + sw.CostEffectiveness + sw.Capability + sw.Recency
	assert.InDelta(t, 1.0, total, 0.0001)
}

func TestScoreThresholds_Structure(t *testing.T) {
	st := ScoreThresholds{
		MinScore: 0.0,
		MaxScore: 10.0,
	}

	assert.Equal(t, 0.0, st.MinScore)
	assert.Equal(t, 10.0, st.MaxScore)
}

func TestScoringConfig_Structure(t *testing.T) {
	now := time.Now()
	sc := ScoringConfig{
		ConfigName: "production",
		Weights: ScoreWeights{
			ResponseSpeed: 0.3,
		},
		Thresholds: ScoreThresholds{
			MinScore: 0.0,
			MaxScore: 10.0,
		},
		Enabled:     true,
		LastUpdated: now,
	}

	assert.Equal(t, "production", sc.ConfigName)
	assert.True(t, sc.Enabled)
	assert.Equal(t, 0.3, sc.Weights.ResponseSpeed)
}

func TestBatchScoreRequest_Structure(t *testing.T) {
	weights := &ScoreWeights{ResponseSpeed: 0.5}
	bsr := BatchScoreRequest{
		ModelIDs: []string{"gpt-4", "claude-3", "gemini-pro"},
		Weights:  weights,
	}

	assert.Len(t, bsr.ModelIDs, 3)
	assert.Contains(t, bsr.ModelIDs, "gpt-4")
	assert.NotNil(t, bsr.Weights)
}

func TestBatchScoreResponse_Structure(t *testing.T) {
	bsr := BatchScoreResponse{
		Scores: []*ComprehensiveScore{
			{ModelID: "gpt-4", OverallScore: 8.5},
			{ModelID: "claude-3", OverallScore: 9.0},
		},
		Processed:   2,
		Failed:      0,
		Total:       2,
		ProcessTime: 1.5,
	}

	assert.Len(t, bsr.Scores, 2)
	assert.Equal(t, 2, bsr.Processed)
	assert.Equal(t, 0, bsr.Failed)
	assert.Equal(t, 1.5, bsr.ProcessTime)
}

func TestScoreComparison_Structure(t *testing.T) {
	sc := ScoreComparison{
		ModelID1:    "gpt-4",
		ModelID2:    "claude-3",
		Score1:      8.5,
		Score2:      9.0,
		Difference:  0.5,
		BetterModel: "claude-3",
	}

	assert.Equal(t, "gpt-4", sc.ModelID1)
	assert.Equal(t, "claude-3", sc.ModelID2)
	assert.Equal(t, 0.5, sc.Difference)
	assert.Equal(t, "claude-3", sc.BetterModel)
}

func TestScoreHistory_Structure(t *testing.T) {
	now := time.Now()
	sh := ScoreHistory{
		ModelID:      "gpt-4",
		Scores:       []float64{8.0, 8.2, 8.5, 8.5, 8.7},
		Timestamps:   []time.Time{now.Add(-4 * time.Hour), now.Add(-3 * time.Hour), now.Add(-2 * time.Hour), now.Add(-1 * time.Hour), now},
		ScoreChanges: []float64{0.0, 0.2, 0.3, 0.0, 0.2},
	}

	assert.Equal(t, "gpt-4", sh.ModelID)
	assert.Len(t, sh.Scores, 5)
	assert.Len(t, sh.Timestamps, 5)
	assert.Len(t, sh.ScoreChanges, 5)
}

func TestScoreAnalytics_Structure(t *testing.T) {
	sa := ScoreAnalytics{
		AverageScore: 7.5,
		MedianScore:  7.8,
		MinScore:     5.0,
		MaxScore:     9.5,
		StdDev:       1.2,
		TotalModels:  100,
		ScoreDistribution: []ScoreDistribution{
			{TotalModels: 100, AverageScore: 7.5, MedianScore: 7.8, MinScore: 5.0, MaxScore: 9.5},
		},
	}

	assert.Equal(t, 7.5, sa.AverageScore)
	assert.Equal(t, 7.8, sa.MedianScore)
	assert.Equal(t, 100, sa.TotalModels)
	assert.Len(t, sa.ScoreDistribution, 1)
}

func TestModelData_Structure(t *testing.T) {
	now := time.Now()
	md := ModelData{
		ID:              "gpt-4-turbo",
		Name:            "GPT-4 Turbo",
		Provider:        "openai",
		Description:     "Most capable GPT-4 model",
		Capabilities:    []string{"chat", "code", "vision"},
		ContextWindow:   128000,
		MaxTokens:       4096,
		InputTokenCost:  0.01,
		OutputTokenCost: 0.03,
		ThroughputRPS:   100.0,
		LatencyMs:       500,
		ReleaseDate:     now.Add(-30 * 24 * time.Hour),
		TrainingCutoff:  now.Add(-90 * 24 * time.Hour),
		ParameterCount:  175000000000,
		OpenSource:      false,
		Multimodal:      true,
		Reasoning:       true,
		LastUpdated:     now,
	}

	assert.Equal(t, "gpt-4-turbo", md.ID)
	assert.Equal(t, "openai", md.Provider)
	assert.Equal(t, 128000, md.ContextWindow)
	assert.Len(t, md.Capabilities, 3)
	assert.False(t, md.OpenSource)
	assert.True(t, md.Multimodal)
}

func TestModelRanking_Structure(t *testing.T) {
	mr := ModelRanking{
		Rank:          1,
		ModelID:       "gpt-4",
		ModelName:     "GPT-4",
		OverallScore:  9.5,
		ScoreSuffix:   "(SC:9.5)",
		Category:      "coding",
		CategoryScore: 9.8,
		LastUpdated:   time.Now(),
	}

	assert.Equal(t, 1, mr.Rank)
	assert.Equal(t, "gpt-4", mr.ModelID)
	assert.Equal(t, 9.5, mr.OverallScore)
	assert.Equal(t, "coding", mr.Category)
}

func TestDefaultScoringConfig(t *testing.T) {
	config := DefaultScoringConfig()

	assert.Equal(t, "default", config.ConfigName)
	assert.True(t, config.Enabled)

	// Check default weights
	assert.Equal(t, 0.25, config.Weights.ResponseSpeed)
	assert.Equal(t, 0.20, config.Weights.ModelEfficiency)
	assert.Equal(t, 0.25, config.Weights.CostEffectiveness)
	assert.Equal(t, 0.20, config.Weights.Capability)
	assert.Equal(t, 0.10, config.Weights.Recency)

	// Check thresholds
	assert.Equal(t, 0.0, config.Thresholds.MinScore)
	assert.Equal(t, 10.0, config.Thresholds.MaxScore)

	// Verify weights sum to 1.0
	total := config.Weights.ResponseSpeed + config.Weights.ModelEfficiency +
		config.Weights.CostEffectiveness + config.Weights.Capability + config.Weights.Recency
	assert.InDelta(t, 1.0, total, 0.0001)
}

func TestDefaultScoringConfig_HasRecentTimestamp(t *testing.T) {
	config := DefaultScoringConfig()

	// The LastUpdated should be recent (within last second)
	assert.True(t, time.Since(config.LastUpdated) < time.Second)
}
