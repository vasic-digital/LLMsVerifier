package analytics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== ModelRecommender Tests ====================

func TestNewModelRecommender(t *testing.T) {
	recommender := NewModelRecommender(nil)
	require.NotNil(t, recommender)
	assert.Nil(t, recommender.db)
}

func TestModelRecommender_RecommendModel(t *testing.T) {
	recommender := NewModelRecommender(nil)

	t.Run("coding task", func(t *testing.T) {
		requirements := TaskRequirements{
			TaskType:         "coding",
			Complexity:       "complex",
			SpeedRequirement: "quality_first",
			BudgetLimit:      0.10,
			RequiredFeatures: []string{"function_calling"},
			ContextLength:    8000,
		}

		result, err := recommender.RecommendModel(requirements)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.Recommendations)
		assert.NotEmpty(t, result.BestChoice.ModelID)
		assert.Equal(t, requirements, result.Requirements)
		assert.Greater(t, result.AnalysisTime, 0.0)
	})

	t.Run("simple writing task", func(t *testing.T) {
		requirements := TaskRequirements{
			TaskType:         "writing",
			Complexity:       "simple",
			SpeedRequirement: "fast",
			BudgetLimit:      0.01,
		}

		result, err := recommender.RecommendModel(requirements)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.Recommendations)
	})

	t.Run("analysis task", func(t *testing.T) {
		requirements := TaskRequirements{
			TaskType:         "analysis",
			Complexity:       "medium",
			SpeedRequirement: "normal",
		}

		result, err := recommender.RecommendModel(requirements)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.Recommendations)
	})

	t.Run("large context requirement", func(t *testing.T) {
		requirements := TaskRequirements{
			TaskType:      "research",
			Complexity:    "complex",
			ContextLength: 100000,
		}

		result, err := recommender.RecommendModel(requirements)
		require.NoError(t, err)
		require.NotNil(t, result)
		// Models with large context windows should rank higher
		assert.NotEmpty(t, result.Recommendations)
	})
}

func TestModelRecommender_GetModelComparison(t *testing.T) {
	recommender := NewModelRecommender(nil)

	t.Run("compare two models", func(t *testing.T) {
		requirements := TaskRequirements{
			TaskType:   "coding",
			Complexity: "medium",
		}

		comparison, err := recommender.GetModelComparison(requirements, []string{"gpt-4", "gpt-3.5-turbo"})
		require.NoError(t, err)
		require.NotNil(t, comparison)
		assert.Len(t, comparison.Models, 2)
	})

	t.Run("compare multiple models", func(t *testing.T) {
		requirements := TaskRequirements{
			TaskType: "writing",
		}

		comparison, err := recommender.GetModelComparison(requirements, []string{"gpt-4", "claude-3-sonnet", "gemini-pro"})
		require.NoError(t, err)
		require.NotNil(t, comparison)
		assert.Len(t, comparison.Models, 3)
		// Should be sorted by score
		assert.GreaterOrEqual(t, comparison.Models[0].Score, comparison.Models[1].Score)
	})

	t.Run("no matching models", func(t *testing.T) {
		requirements := TaskRequirements{
			TaskType: "coding",
		}

		_, err := recommender.GetModelComparison(requirements, []string{"nonexistent-model"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no requested models found")
	})
}

func TestModelRecommender_GetUsageInsights(t *testing.T) {
	recommender := NewModelRecommender(nil)

	timeRange := TimeRange{}

	insights, err := recommender.GetUsageInsights(timeRange)
	require.NoError(t, err)
	require.NotNil(t, insights)
	assert.NotEmpty(t, insights.MostUsedModels)
	assert.NotEmpty(t, insights.UnderUtilizedModels)
	assert.NotEmpty(t, insights.CostDrivers)
	assert.NotEmpty(t, insights.Insights)
}

// ==================== Score Calculation Tests ====================

func TestModelRecommender_calculateTaskMatchScore(t *testing.T) {
	recommender := NewModelRecommender(nil)

	t.Run("perfect match", func(t *testing.T) {
		score := recommender.calculateTaskMatchScore([]string{"coding", "writing"}, "coding")
		assert.Equal(t, 30.0, score)
	})

	t.Run("partial match", func(t *testing.T) {
		score := recommender.calculateTaskMatchScore([]string{"code_review"}, "code")
		assert.Equal(t, 20.0, score)
	})

	t.Run("no match", func(t *testing.T) {
		score := recommender.calculateTaskMatchScore([]string{"writing", "analysis"}, "coding")
		assert.Equal(t, 5.0, score)
	})

	t.Run("case insensitive match", func(t *testing.T) {
		score := recommender.calculateTaskMatchScore([]string{"CODING", "Writing"}, "Coding")
		assert.Equal(t, 30.0, score)
	})
}

func TestModelRecommender_calculateComplexityScore(t *testing.T) {
	recommender := NewModelRecommender(nil)

	t.Run("simple complexity", func(t *testing.T) {
		score := recommender.calculateComplexityScore(80.0, "simple")
		assert.Equal(t, 20.0, score)
	})

	t.Run("medium with high performance", func(t *testing.T) {
		score := recommender.calculateComplexityScore(90.0, "medium")
		assert.Equal(t, 20.0, score)
	})

	t.Run("medium with medium performance", func(t *testing.T) {
		score := recommender.calculateComplexityScore(80.0, "medium")
		assert.Equal(t, 15.0, score)
	})

	t.Run("medium with low performance", func(t *testing.T) {
		score := recommender.calculateComplexityScore(70.0, "medium")
		assert.Equal(t, 10.0, score)
	})

	t.Run("complex with high performance", func(t *testing.T) {
		score := recommender.calculateComplexityScore(95.0, "complex")
		assert.Equal(t, 20.0, score)
	})

	t.Run("complex with medium performance", func(t *testing.T) {
		score := recommender.calculateComplexityScore(85.0, "complex")
		assert.Equal(t, 15.0, score)
	})

	t.Run("complex with low performance", func(t *testing.T) {
		score := recommender.calculateComplexityScore(70.0, "complex")
		assert.Equal(t, 5.0, score)
	})

	t.Run("unknown complexity", func(t *testing.T) {
		score := recommender.calculateComplexityScore(80.0, "unknown")
		assert.Equal(t, 15.0, score)
	})
}

func TestModelRecommender_calculateSpeedScore(t *testing.T) {
	recommender := NewModelRecommender(nil)

	cheapModel := ModelData{
		CostPerToken:     0.005,
		PerformanceScore: 85.0,
	}

	expensiveModel := ModelData{
		CostPerToken:     0.03,
		PerformanceScore: 95.0,
	}

	t.Run("fast requirement - cheap model", func(t *testing.T) {
		score := recommender.calculateSpeedScore(cheapModel, "fast")
		assert.Equal(t, 15.0, score)
	})

	t.Run("fast requirement - expensive model", func(t *testing.T) {
		score := recommender.calculateSpeedScore(expensiveModel, "fast")
		assert.Equal(t, 8.0, score)
	})

	t.Run("normal requirement - balanced model", func(t *testing.T) {
		score := recommender.calculateSpeedScore(cheapModel, "normal")
		assert.Equal(t, 15.0, score)
	})

	t.Run("quality_first - high performance", func(t *testing.T) {
		score := recommender.calculateSpeedScore(expensiveModel, "quality_first")
		assert.Equal(t, 15.0, score)
	})

	t.Run("quality_first - medium performance", func(t *testing.T) {
		mediumModel := ModelData{PerformanceScore: 87.0}
		score := recommender.calculateSpeedScore(mediumModel, "quality_first")
		assert.Equal(t, 12.0, score)
	})

	t.Run("unknown requirement", func(t *testing.T) {
		score := recommender.calculateSpeedScore(cheapModel, "unknown")
		assert.Equal(t, 12.0, score)
	})
}

func TestModelRecommender_calculateCostScore(t *testing.T) {
	recommender := NewModelRecommender(nil)

	t.Run("within budget", func(t *testing.T) {
		model := ModelData{CostPerToken: 0.001}
		requirements := TaskRequirements{
			BudgetLimit: 0.10,
			Complexity:  "simple",
		}
		score := recommender.calculateCostScore(model, requirements)
		assert.Greater(t, score, 10.0)
	})

	t.Run("over budget", func(t *testing.T) {
		model := ModelData{CostPerToken: 0.10}
		requirements := TaskRequirements{
			BudgetLimit: 0.001,
			Complexity:  "complex",
		}
		score := recommender.calculateCostScore(model, requirements)
		assert.Equal(t, 0.0, score)
	})

	t.Run("no budget limit - very cheap", func(t *testing.T) {
		model := ModelData{CostPerToken: 0.0005}
		requirements := TaskRequirements{}
		score := recommender.calculateCostScore(model, requirements)
		assert.Equal(t, 15.0, score)
	})

	t.Run("no budget limit - reasonably priced", func(t *testing.T) {
		model := ModelData{CostPerToken: 0.005}
		requirements := TaskRequirements{}
		score := recommender.calculateCostScore(model, requirements)
		assert.Equal(t, 12.0, score)
	})

	t.Run("no budget limit - expensive", func(t *testing.T) {
		model := ModelData{CostPerToken: 0.015}
		requirements := TaskRequirements{}
		score := recommender.calculateCostScore(model, requirements)
		assert.Equal(t, 8.0, score)
	})

	t.Run("no budget limit - very expensive", func(t *testing.T) {
		model := ModelData{CostPerToken: 0.05}
		requirements := TaskRequirements{}
		score := recommender.calculateCostScore(model, requirements)
		assert.Equal(t, 4.0, score)
	})
}

func TestModelRecommender_calculateFeatureScore(t *testing.T) {
	recommender := NewModelRecommender(nil)

	t.Run("no requirements", func(t *testing.T) {
		score := recommender.calculateFeatureScore([]string{"streaming", "json_mode"}, nil)
		assert.Equal(t, 10.0, score)
	})

	t.Run("empty requirements", func(t *testing.T) {
		score := recommender.calculateFeatureScore([]string{"streaming", "json_mode"}, []string{})
		assert.Equal(t, 10.0, score)
	})

	t.Run("all features matched", func(t *testing.T) {
		score := recommender.calculateFeatureScore(
			[]string{"streaming", "json_mode", "function_calling"},
			[]string{"streaming", "json_mode"},
		)
		assert.Equal(t, 10.0, score)
	})

	t.Run("partial features matched", func(t *testing.T) {
		score := recommender.calculateFeatureScore(
			[]string{"streaming"},
			[]string{"streaming", "json_mode"},
		)
		assert.Equal(t, 5.0, score)
	})

	t.Run("no features matched", func(t *testing.T) {
		score := recommender.calculateFeatureScore(
			[]string{"streaming"},
			[]string{"json_mode", "function_calling"},
		)
		assert.Equal(t, 0.0, score)
	})
}

func TestModelRecommender_calculateContextScore(t *testing.T) {
	recommender := NewModelRecommender(nil)

	t.Run("no requirement", func(t *testing.T) {
		score := recommender.calculateContextScore(8000, 0)
		assert.Equal(t, 5.0, score)
	})

	t.Run("sufficient context", func(t *testing.T) {
		score := recommender.calculateContextScore(8000, 4000)
		assert.Equal(t, 5.0, score)
	})

	t.Run("marginally sufficient", func(t *testing.T) {
		score := recommender.calculateContextScore(8000, 6000)
		assert.Equal(t, 5.0, score)
	})

	t.Run("limited context", func(t *testing.T) {
		score := recommender.calculateContextScore(8000, 12000)
		assert.Equal(t, 3.0, score)
	})

	t.Run("insufficient context", func(t *testing.T) {
		score := recommender.calculateContextScore(4000, 16000)
		assert.Equal(t, 0.0, score)
	})
}

func TestModelRecommender_estimateCost(t *testing.T) {
	recommender := NewModelRecommender(nil)

	model := ModelData{
		CostPerToken: 0.01,
	}

	t.Run("simple task", func(t *testing.T) {
		requirements := TaskRequirements{Complexity: "simple"}
		cost := recommender.estimateCost(model, requirements)
		// Simple: 500 input + 200 output = 700 tokens = 0.007
		assert.InDelta(t, 0.007, cost, 0.001)
	})

	t.Run("medium task", func(t *testing.T) {
		requirements := TaskRequirements{Complexity: "medium"}
		cost := recommender.estimateCost(model, requirements)
		// Medium: 1000 input + 500 output = 1500 tokens = 0.015
		assert.InDelta(t, 0.015, cost, 0.001)
	})

	t.Run("complex task", func(t *testing.T) {
		requirements := TaskRequirements{Complexity: "complex"}
		cost := recommender.estimateCost(model, requirements)
		// Complex: 2000 input + 1000 output = 3000 tokens = 0.03
		assert.InDelta(t, 0.03, cost, 0.001)
	})

	t.Run("with context multiplier", func(t *testing.T) {
		requirements := TaskRequirements{
			Complexity:    "simple",
			ContextLength: 8000, // 2x baseline
		}
		cost := recommender.estimateCost(model, requirements)
		// Adjusted: (500 * 2) + 200 = 1200 tokens = 0.012
		assert.InDelta(t, 0.012, cost, 0.001)
	})

	t.Run("large context capped at 3x", func(t *testing.T) {
		requirements := TaskRequirements{
			Complexity:    "simple",
			ContextLength: 20000, // 5x baseline but capped at 3x
		}
		cost := recommender.estimateCost(model, requirements)
		// Adjusted: (500 * 3) + 200 = 1700 tokens = 0.017
		assert.InDelta(t, 0.017, cost, 0.001)
	})
}

func TestModelRecommender_scoreModel(t *testing.T) {
	recommender := NewModelRecommender(nil)

	model := ModelData{
		ID:               "test-model",
		Name:             "Test Model",
		Provider:         "TestProvider",
		TaskTypes:        []string{"coding", "writing"},
		PerformanceScore: 90.0,
		ReliabilityScore: 95.0,
		CostPerToken:     0.01,
		ContextLength:    8000,
		Features:         []string{"streaming", "json_mode"},
	}

	requirements := TaskRequirements{
		TaskType:         "coding",
		Complexity:       "medium",
		SpeedRequirement: "normal",
		RequiredFeatures: []string{"streaming"},
		ContextLength:    4000,
	}

	recommendation := recommender.scoreModel(model, requirements)

	assert.Equal(t, "test-model", recommendation.ModelID)
	assert.Equal(t, "TestProvider", recommendation.Provider)
	assert.Greater(t, recommendation.Score, 0.0)
	assert.LessOrEqual(t, recommendation.Score, 100.0)
	assert.Greater(t, recommendation.CostEstimate, 0.0)
	assert.Equal(t, 90.0, recommendation.PerformanceScore)
	assert.Equal(t, 95.0, recommendation.ReliabilityScore)
	assert.NotEmpty(t, recommendation.Reasoning)
}

// ==================== Struct Tests ====================

func TestTaskRequirements_Struct(t *testing.T) {
	requirements := TaskRequirements{
		TaskType:         "coding",
		Complexity:       "complex",
		SpeedRequirement: "quality_first",
		BudgetLimit:      0.10,
		RequiredFeatures: []string{"function_calling", "json_mode"},
		ContextLength:    16000,
	}

	assert.Equal(t, "coding", requirements.TaskType)
	assert.Equal(t, "complex", requirements.Complexity)
	assert.Equal(t, "quality_first", requirements.SpeedRequirement)
	assert.Equal(t, 0.10, requirements.BudgetLimit)
	assert.Len(t, requirements.RequiredFeatures, 2)
	assert.Equal(t, 16000, requirements.ContextLength)
}

func TestModelRecommendation_Struct(t *testing.T) {
	recommendation := ModelRecommendation{
		ModelID:          "gpt-4",
		Provider:         "OpenAI",
		Score:            95.5,
		CostEstimate:     0.05,
		PerformanceScore: 95.0,
		ReliabilityScore: 98.0,
		Reasoning:        []string{"Best for coding", "High reliability"},
	}

	assert.Equal(t, "gpt-4", recommendation.ModelID)
	assert.Equal(t, "OpenAI", recommendation.Provider)
	assert.Equal(t, 95.5, recommendation.Score)
	assert.Equal(t, 0.05, recommendation.CostEstimate)
	assert.Len(t, recommendation.Reasoning, 2)
}

func TestRecommendationResult_Struct(t *testing.T) {
	result := RecommendationResult{
		Requirements: TaskRequirements{TaskType: "coding"},
		Recommendations: []ModelRecommendation{
			{ModelID: "gpt-4", Score: 95.0},
			{ModelID: "claude-3-sonnet", Score: 90.0},
		},
		BestChoice: ModelRecommendation{ModelID: "gpt-4", Score: 95.0},
		AlternativeOptions: []ModelRecommendation{
			{ModelID: "claude-3-sonnet", Score: 90.0},
		},
		AnalysisTime: 5.5,
	}

	assert.Equal(t, "coding", result.Requirements.TaskType)
	assert.Len(t, result.Recommendations, 2)
	assert.Equal(t, "gpt-4", result.BestChoice.ModelID)
	assert.Len(t, result.AlternativeOptions, 1)
	assert.Equal(t, 5.5, result.AnalysisTime)
}

func TestModelData_Struct(t *testing.T) {
	model := ModelData{
		ID:               "test-model",
		Name:             "Test Model",
		Provider:         "TestProvider",
		TaskTypes:        []string{"coding", "writing"},
		PerformanceScore: 90.0,
		ReliabilityScore: 95.0,
		CostPerToken:     0.01,
		ContextLength:    8000,
		Features:         []string{"streaming"},
		MaxTokens:        4096,
	}

	assert.Equal(t, "test-model", model.ID)
	assert.Equal(t, "Test Model", model.Name)
	assert.Equal(t, "TestProvider", model.Provider)
	assert.Len(t, model.TaskTypes, 2)
	assert.Equal(t, 90.0, model.PerformanceScore)
	assert.Equal(t, 8000, model.ContextLength)
	assert.Equal(t, 4096, model.MaxTokens)
}

func TestModelComparison_Struct(t *testing.T) {
	comparison := ModelComparison{
		Requirements: TaskRequirements{TaskType: "coding"},
		Models: []ModelRecommendation{
			{ModelID: "gpt-4", Score: 95.0},
			{ModelID: "claude-3-sonnet", Score: 90.0},
		},
	}

	assert.Equal(t, "coding", comparison.Requirements.TaskType)
	assert.Len(t, comparison.Models, 2)
}

func TestUsageInsights_Struct(t *testing.T) {
	insights := UsageInsights{
		TimeRange:           TimeRange{},
		MostUsedModels:      []string{"gpt-4", "claude-3-sonnet"},
		UnderUtilizedModels: []string{"codellama-34b"},
		CostDrivers:         []string{"gpt-4"},
		Insights:            []string{"Consider using cheaper models for simple tasks"},
	}

	assert.Len(t, insights.MostUsedModels, 2)
	assert.Len(t, insights.UnderUtilizedModels, 1)
	assert.Len(t, insights.CostDrivers, 1)
	assert.Len(t, insights.Insights, 1)
}

// ==================== Integration Tests ====================

func TestModelRecommender_EndToEnd(t *testing.T) {
	recommender := NewModelRecommender(nil)

	// Test a full recommendation flow
	requirements := TaskRequirements{
		TaskType:         "coding",
		Complexity:       "complex",
		SpeedRequirement: "quality_first",
		BudgetLimit:      0.20,
		RequiredFeatures: []string{"function_calling", "json_mode"},
		ContextLength:    8000,
	}

	// Get recommendations
	result, err := recommender.RecommendModel(requirements)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify best choice meets criteria
	assert.NotEmpty(t, result.BestChoice.ModelID)
	assert.Greater(t, result.BestChoice.Score, 0.0)
	assert.LessOrEqual(t, result.BestChoice.CostEstimate, requirements.BudgetLimit*2) // Allow some margin

	// Compare top recommendations
	if len(result.Recommendations) >= 2 {
		comparison, err := recommender.GetModelComparison(requirements, []string{
			result.Recommendations[0].ModelID,
			result.Recommendations[1].ModelID,
		})
		require.NoError(t, err)
		require.NotNil(t, comparison)
		assert.Len(t, comparison.Models, 2)
	}

	// Get usage insights
	insights, err := recommender.GetUsageInsights(TimeRange{})
	require.NoError(t, err)
	require.NotNil(t, insights)
	assert.NotEmpty(t, insights.Insights)
}
