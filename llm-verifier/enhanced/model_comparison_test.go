package enhanced

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"llm-verifier/database"
)

func TestNewModelComparisonEngine(t *testing.T) {
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	engine := NewModelComparisonEngine(db)
	require.NotNil(t, engine)
	assert.NotNil(t, engine.db)
}

func TestModelComparisonEngine_CompareModels_NotEnoughModels(t *testing.T) {
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	engine := NewModelComparisonEngine(db)

	t.Run("no models", func(t *testing.T) {
		_, err := engine.CompareModels([]string{}, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least 2 models")
	})

	t.Run("one model", func(t *testing.T) {
		_, err := engine.CompareModels([]string{"model-1"}, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least 2 models")
	})
}

func TestModelComparisonEngine_CompareModels_Success(t *testing.T) {
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create test provider
	provider := &database.Provider{
		Name:     "Test Provider",
		Endpoint: "https://test.example.com",
	}
	err = db.CreateProvider(provider)
	require.NoError(t, err)

	// Create test models
	contextWindow1 := 8192
	contextWindow2 := 32768
	params1 := int64(7000000000)
	params2 := int64(70000000000)
	releaseDate := time.Now()

	model1 := &database.Model{
		ProviderID:          provider.ID,
		ModelID:             "model-1",
		Name:                "Test Model 1",
		ContextWindowTokens: &contextWindow1,
		ParameterCount:      &params1,
		ReleaseDate:         &releaseDate,
		VerificationStatus:  "verified",
		OverallScore:        85.0,
	}
	err = db.CreateModel(model1)
	require.NoError(t, err)

	model2 := &database.Model{
		ProviderID:          provider.ID,
		ModelID:             "model-2",
		Name:                "Test Model 2",
		ContextWindowTokens: &contextWindow2,
		ParameterCount:      &params2,
		ReleaseDate:         &releaseDate,
		VerificationStatus:  "verified",
		OverallScore:        90.0,
	}
	err = db.CreateModel(model2)
	require.NoError(t, err)

	engine := NewModelComparisonEngine(db)

	result, err := engine.CompareModels([]string{"model-1", "model-2"}, false)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check basic structure
	assert.Len(t, result.Models, 2)
	assert.NotEmpty(t, result.Metrics)
	assert.NotEmpty(t, result.Summary)

	// Check context window metric
	if contextMetric, exists := result.Metrics["context_window"]; exists {
		assert.Equal(t, "Context Window", contextMetric.MetricName)
		assert.Len(t, contextMetric.Values, 2)
		assert.Equal(t, float64(contextWindow2), contextMetric.BestValue)
	}

	// Check parameters metric
	if paramMetric, exists := result.Metrics["parameters"]; exists {
		assert.Equal(t, "Parameters", paramMetric.MetricName)
		assert.Len(t, paramMetric.Values, 2)
	}
}

func TestModelComparisonEngine_CompareModels_WithVerificationResults(t *testing.T) {
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create test provider
	provider := &database.Provider{
		Name:     "Test Provider",
		Endpoint: "https://test.example.com",
	}
	err = db.CreateProvider(provider)
	require.NoError(t, err)

	// Create test models
	model1 := &database.Model{
		ProviderID:         provider.ID,
		ModelID:            "perf-model-1",
		Name:               "Performance Model 1",
		VerificationStatus: "verified",
	}
	err = db.CreateModel(model1)
	require.NoError(t, err)

	model2 := &database.Model{
		ProviderID:         provider.ID,
		ModelID:            "perf-model-2",
		Name:               "Performance Model 2",
		VerificationStatus: "verified",
	}
	err = db.CreateModel(model2)
	require.NoError(t, err)

	// Create verification results
	vr1 := &database.VerificationResult{
		ModelID:             model1.ID,
		VerificationType:    "full",
		Status:              "completed",
		OverallScore:        85.0,
		CodeCapabilityScore: 80.0,
		ResponsivenessScore: 90.0,
	}
	err = db.CreateVerificationResult(vr1)
	require.NoError(t, err)

	vr2 := &database.VerificationResult{
		ModelID:             model2.ID,
		VerificationType:    "full",
		Status:              "completed",
		OverallScore:        92.0,
		CodeCapabilityScore: 95.0,
		ResponsivenessScore: 85.0,
	}
	err = db.CreateVerificationResult(vr2)
	require.NoError(t, err)

	engine := NewModelComparisonEngine(db)

	result, err := engine.CompareModels([]string{"perf-model-1", "perf-model-2"}, true)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have performance metrics
	assert.NotEmpty(t, result.Metrics)
}

func TestModelComparisonEngine_createMetricComparison(t *testing.T) {
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	engine := NewModelComparisonEngine(db)

	t.Run("empty values", func(t *testing.T) {
		result := engine.createMetricComparison("Test", "Description", map[string]float64{}, true)
		assert.Equal(t, "Test", result.MetricName)
		assert.Equal(t, "Description", result.Description)
	})

	t.Run("higher is better", func(t *testing.T) {
		values := map[string]float64{
			"model-a": 100.0,
			"model-b": 50.0,
			"model-c": 75.0,
		}
		result := engine.createMetricComparison("Score", "Test score", values, true)

		assert.Equal(t, 100.0, result.BestValue)
		assert.Equal(t, 50.0, result.WorstValue)
		assert.Equal(t, 75.0, result.AvgValue)
		assert.Equal(t, "model-a", result.Ranking[0]) // Best should be first
		assert.Equal(t, "model-b", result.Ranking[2]) // Worst should be last
	})

	t.Run("lower is better", func(t *testing.T) {
		values := map[string]float64{
			"model-a": 100.0,
			"model-b": 50.0,
			"model-c": 75.0,
		}
		result := engine.createMetricComparison("Latency", "Response time", values, false)

		assert.Equal(t, 50.0, result.BestValue) // Lower is better, so best is 50
		assert.Equal(t, "model-b", result.Ranking[0]) // Best should be first
	})
}

func TestModelComparisonEngine_calculatePercentile(t *testing.T) {
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	engine := NewModelComparisonEngine(db)

	t.Run("single score", func(t *testing.T) {
		scores := map[string]float64{"model-a": 90.0}
		percentile := engine.calculatePercentile(90.0, scores)
		assert.Equal(t, 100.0, percentile)
	})

	t.Run("best score", func(t *testing.T) {
		scores := map[string]float64{
			"model-a": 100.0,
			"model-b": 80.0,
			"model-c": 60.0,
		}
		percentile := engine.calculatePercentile(100.0, scores)
		assert.Equal(t, 100.0, percentile) // Best score = 100th percentile
	})

	t.Run("middle score", func(t *testing.T) {
		scores := map[string]float64{
			"model-a": 100.0,
			"model-b": 80.0,
			"model-c": 60.0,
		}
		percentile := engine.calculatePercentile(80.0, scores)
		// 1 score is better (100), so percentile = (1 - 1/3) * 100 = 66.67
		assert.InDelta(t, 66.67, percentile, 0.1)
	})

	t.Run("worst score", func(t *testing.T) {
		scores := map[string]float64{
			"model-a": 100.0,
			"model-b": 80.0,
			"model-c": 60.0,
		}
		percentile := engine.calculatePercentile(60.0, scores)
		// 2 scores are better, so percentile = (1 - 2/3) * 100 = 33.33
		assert.InDelta(t, 33.33, percentile, 0.1)
	})
}

func TestModelComparisonEngine_generateSummary(t *testing.T) {
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	engine := NewModelComparisonEngine(db)

	t.Run("no models", func(t *testing.T) {
		result := &ComparisonResult{
			Models: []*database.Model{},
		}
		summary := engine.generateSummary(result)
		assert.Equal(t, "No models to compare", summary)
	})

	t.Run("with models and rankings", func(t *testing.T) {
		result := &ComparisonResult{
			Models: []*database.Model{
				{ModelID: "model-1"},
				{ModelID: "model-2"},
			},
			Rankings: map[string][]ModelRanking{
				"composite": {
					{ModelID: "model-1", Score: 90.0, Rank: 1},
					{ModelID: "model-2", Score: 80.0, Rank: 2},
				},
			},
			Metrics: map[string]MetricComparison{},
		}
		summary := engine.generateSummary(result)
		assert.Contains(t, summary, "Comparison of 2 models")
		assert.Contains(t, summary, "Best performer: model-1")
	})

	t.Run("with significant differences", func(t *testing.T) {
		result := &ComparisonResult{
			Models: []*database.Model{
				{ModelID: "model-1"},
				{ModelID: "model-2"},
			},
			Rankings: map[string][]ModelRanking{
				"composite": {
					{ModelID: "model-1", Score: 90.0, Rank: 1},
				},
			},
			Metrics: map[string]MetricComparison{
				"context_window": {
					Values:     map[string]float64{"model-1": 8192, "model-2": 32768},
					BestValue:  32768,
					WorstValue: 8192,
				},
				"parameters": {
					Values:     map[string]float64{"model-1": 7e9, "model-2": 70e9},
					BestValue:  70e9,
					WorstValue: 7e9,
				},
			},
		}
		summary := engine.generateSummary(result)
		assert.Contains(t, summary, "context window sizes")
		assert.Contains(t, summary, "model sizes")
	})
}

func TestModelComparisonEngine_generateRecommendations(t *testing.T) {
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	engine := NewModelComparisonEngine(db)

	t.Run("with rankings", func(t *testing.T) {
		result := &ComparisonResult{
			Rankings: map[string][]ModelRanking{
				"composite": {
					{ModelID: "best-model", Score: 95.0, Rank: 1},
					{ModelID: "other-model", Score: 80.0, Rank: 2},
				},
			},
			Metrics: map[string]MetricComparison{
				"context_window": {
					Ranking: []string{"other-model", "best-model"}, // Different best for context
				},
				"code_capability": {
					Ranking: []string{"other-model", "best-model"}, // Different best for code
				},
			},
			Recommendations: []string{},
		}

		engine.generateRecommendations(result)

		assert.NotEmpty(t, result.Recommendations)
		assert.Contains(t, result.Recommendations[0], "Best overall model: best-model")
	})
}

func TestMetricComparison_Structure(t *testing.T) {
	mc := MetricComparison{
		MetricName:  "Test Metric",
		Description: "A test metric for comparison",
		Values:      map[string]float64{"model-1": 100.0, "model-2": 80.0},
		Ranking:     []string{"model-1", "model-2"},
		BestValue:   100.0,
		WorstValue:  80.0,
		AvgValue:    90.0,
	}

	assert.Equal(t, "Test Metric", mc.MetricName)
	assert.Equal(t, 2, len(mc.Values))
	assert.Equal(t, 2, len(mc.Ranking))
	assert.Equal(t, 100.0, mc.BestValue)
}

func TestModelRanking_Structure(t *testing.T) {
	mr := ModelRanking{
		ModelID:    "test-model",
		Rank:       1,
		Score:      95.5,
		Percentile: 99.0,
	}

	assert.Equal(t, "test-model", mr.ModelID)
	assert.Equal(t, 1, mr.Rank)
	assert.Equal(t, 95.5, mr.Score)
	assert.Equal(t, 99.0, mr.Percentile)
}

func TestComparisonResult_Structure(t *testing.T) {
	cr := &ComparisonResult{
		Models:          []*database.Model{{ModelID: "test"}},
		Metrics:         map[string]MetricComparison{},
		Rankings:        map[string][]ModelRanking{},
		Recommendations: []string{"Test recommendation"},
		Summary:         "Test summary",
	}

	assert.Len(t, cr.Models, 1)
	assert.NotNil(t, cr.Metrics)
	assert.NotNil(t, cr.Rankings)
	assert.Len(t, cr.Recommendations, 1)
	assert.Equal(t, "Test summary", cr.Summary)
}
