package scoring

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultScoringSystemConfig(t *testing.T) {
	config := DefaultScoringSystemConfig()

	assert.Equal(t, 6*time.Hour, config.AutoSyncInterval)
	assert.Equal(t, 1*time.Hour, config.ScoreRecalcInterval)
	assert.Equal(t, 10, config.MaxConcurrentCalcs)
	assert.True(t, config.EnableBackgroundSync)
	assert.True(t, config.EnableScoreMonitoring)
	assert.Equal(t, 0.5, config.ScoreChangeThreshold)
}

func TestScoreDistribution(t *testing.T) {
	dist := ScoreDistribution{
		TotalModels:  100,
		AverageScore: 7.5,
		MedianScore:  7.8,
		MinScore:     2.1,
		MaxScore:     9.8,
		ScoreRanges: []ScoreRange{
			{Min: 9.0, Max: 10.0, Count: 5, Percentage: 5.0},
			{Min: 8.0, Max: 8.9, Count: 15, Percentage: 15.0},
		},
	}

	assert.Equal(t, 100, dist.TotalModels)
	assert.Equal(t, 7.5, dist.AverageScore)
	assert.Equal(t, 7.8, dist.MedianScore)
	assert.Len(t, dist.ScoreRanges, 2)
}

func TestScoreRange(t *testing.T) {
	sr := ScoreRange{
		Min:        7.0,
		Max:        7.9,
		Count:      30,
		Percentage: 30.0,
	}

	assert.Equal(t, 7.0, sr.Min)
	assert.Equal(t, 7.9, sr.Max)
	assert.Equal(t, 30, sr.Count)
	assert.Equal(t, 30.0, sr.Percentage)
}

func TestModelRanking(t *testing.T) {
	ranking := ModelRanking{
		Rank:          1,
		ModelID:       "gpt-4",
		ModelName:     "GPT-4",
		OverallScore:  9.5,
		CategoryScore: 8.5,
		ScoreSuffix:   "(SC:9.5)",
		Category:      "capability",
	}

	assert.Equal(t, 1, ranking.Rank)
	assert.Equal(t, "gpt-4", ranking.ModelID)
	assert.Equal(t, "GPT-4", ranking.ModelName)
	assert.Equal(t, 9.5, ranking.OverallScore)
	assert.Equal(t, 8.5, ranking.CategoryScore)
	assert.Equal(t, "capability", ranking.Category)
}

// TestScoringSystemWithMockClient tests scoring system with mock dependencies
func TestScoringSystemWithMockClient(t *testing.T) {
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	logger := setupTestLogger()
	mockClient := NewMockModelsDevClient()

	// Create engine directly with mock client
	engine := NewScoringEngine(db, mockClient, logger)
	require.NotNil(t, engine)

	// Test model naming functionality
	naming := NewModelNaming()
	require.NotNil(t, naming)

	// Test score suffix operations
	name := naming.AddScoreSuffix("Test Model", 8.5)
	assert.Contains(t, name, "8.5")
}

// TestScoringSystemConfig tests the system config struct
func TestScoringSystemConfig(t *testing.T) {
	config := ScoringSystemConfig{
		MaxConcurrentCalcs:    5,
		EnableBackgroundSync:  false,
		EnableScoreMonitoring: false,
		ScoreChangeThreshold:  1.0,
	}

	assert.Equal(t, 5, config.MaxConcurrentCalcs)
	assert.False(t, config.EnableBackgroundSync)
	assert.False(t, config.EnableScoreMonitoring)
	assert.Equal(t, 1.0, config.ScoreChangeThreshold)
}
