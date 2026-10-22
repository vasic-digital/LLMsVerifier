// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package strategy

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultStrategy_Name(t *testing.T) {
	s := NewDefaultStrategy()
	assert.Equal(t, "default", s.Name())
}

func TestDefaultStrategy_Description(t *testing.T) {
	s := NewDefaultStrategy()
	assert.Contains(t, s.Description(), "quality")
	assert.Contains(t, s.Description(), "speed")
	assert.Contains(t, NewDefaultStrategy().Description(), "cost")
}

func TestDefaultStrategy_Score(t *testing.T) {
	ctx := context.Background()
	s := NewDefaultStrategy()

	model := ModelInfo{
		ID:               "test-model-1",
		Name:             "Test Model",
		Provider:         "test",
		Model:            "test-1",
		QualityScore:     0.9,
		ReliabilityScore: 0.95,
		AvgLatencyMs:     500,
		InputCostPer1k:   0.01,
		OutputCostPer1k:  0.02,
		Verified:         true,
		LastVerified:     time.Now(),
	}

	score, err := s.Score(ctx, model)
	require.NoError(t, err)

	assert.Equal(t, "test-model-1", score.ModelID)
	assert.Equal(t, "default", score.StrategyName)
	assert.GreaterOrEqual(t, score.Overall, 0.0)
	assert.LessOrEqual(t, score.Overall, 1.0)
	assert.NotEmpty(t, score.Reasoning)
	assert.NotEmpty(t, score.DimensionScores)
	assert.Contains(t, score.DimensionScores, DimensionQuality)
	assert.Contains(t, score.DimensionScores, DimensionSpeed)
	assert.Contains(t, score.DimensionScores, DimensionCost)
	assert.Contains(t, score.DimensionScores, DimensionReliability)
}

func TestDefaultStrategy_Score_VerifiedModel(t *testing.T) {
	ctx := context.Background()
	s := NewDefaultStrategy()

	verifiedModel := ModelInfo{
		ID:               "verified",
		QualityScore:     0.8,
		ReliabilityScore: 0.9,
		Verified:         true,
		LastVerified:     time.Now(),
	}

	unverifiedModel := ModelInfo{
		ID:               "unverified",
		QualityScore:     0.8,
		ReliabilityScore: 0.9,
		Verified:         false,
	}

	verifiedScore, _ := s.Score(ctx, verifiedModel)
	unverifiedScore, _ := s.Score(ctx, unverifiedModel)

	assert.Greater(t, verifiedScore.Confidence, unverifiedScore.Confidence)
}

func TestDefaultStrategy_Score_Caching(t *testing.T) {
	ctx := context.Background()
	s := NewDefaultStrategy()

	model := ModelInfo{
		ID:               "cached-model",
		QualityScore:     0.8,
		ReliabilityScore: 0.9,
	}

	score1, err := s.Score(ctx, model)
	require.NoError(t, err)

	score2, err := s.Score(ctx, model)
	require.NoError(t, err)

	assert.Equal(t, score1.Timestamp, score2.Timestamp)
}

func TestDefaultStrategy_Validate(t *testing.T) {
	ctx := context.Background()
	s := NewDefaultStrategy()

	model := ModelInfo{
		ID:               "valid-model",
		QualityScore:     0.9,
		ReliabilityScore: 0.95,
		Verified:         true,
	}

	result := s.Validate(ctx, model)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestDefaultStrategy_Validate_WithConstraints(t *testing.T) {
	ctx := context.Background()
	s := NewDefaultStrategy(WithConstraints([]Constraint{
		{
			Name:        "min_quality",
			Type:        "range",
			Value:       0.8,
			Required:    true,
			Description: "minimum quality score",
		},
		{
			Name:        "requires_vision",
			Type:        "bool",
			Value:       true,
			Required:    true,
			Description: "vision capability required",
		},
	}))

	model := ModelInfo{
		ID:               "no-vision",
		QualityScore:     0.9,
		ReliabilityScore: 0.95,
		SupportsVision:   false,
	}

	result := s.Validate(ctx, model)

	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

func TestDefaultStrategy_Rank(t *testing.T) {
	ctx := context.Background()
	s := NewDefaultStrategy()

	models := []ModelInfo{
		{ID: "low", Name: "Low Quality", QualityScore: 0.5, ReliabilityScore: 0.8},
		{ID: "high", Name: "High Quality", QualityScore: 0.95, ReliabilityScore: 0.98},
		{ID: "mid", Name: "Mid Quality", QualityScore: 0.75, ReliabilityScore: 0.9},
	}

	ranked, err := s.Rank(ctx, models)
	require.NoError(t, err)

	assert.Len(t, ranked, 3)
	assert.Equal(t, 1, ranked[0].Rank)
	assert.Equal(t, "high", ranked[0].Model.ID)
	assert.Equal(t, 2, ranked[1].Rank)
	assert.Equal(t, "mid", ranked[1].Model.ID)
	assert.Equal(t, 3, ranked[2].Rank)
	assert.Equal(t, "low", ranked[2].Model.ID)
}

func TestDefaultStrategy_Rank_Tiers(t *testing.T) {
	ctx := context.Background()
	s := NewDefaultStrategy()

	// Models need realistic latency and cost values for tier
	// boundaries to work correctly. Without them, speed and cost
	// dimensions default to 0.5, pulling the overall score down.
	models := []ModelInfo{
		{ID: "tier1-model", QualityScore: 0.95, ReliabilityScore: 0.98, AvgLatencyMs: 500, InputCostPer1k: 0.001, OutputCostPer1k: 0.002},
		{ID: "tier2-model", QualityScore: 0.7, ReliabilityScore: 0.85, AvgLatencyMs: 3000, InputCostPer1k: 0.01, OutputCostPer1k: 0.02},
		{ID: "tier3-model", QualityScore: 0.4, ReliabilityScore: 0.7, AvgLatencyMs: 8000, InputCostPer1k: 0.04, OutputCostPer1k: 0.05},
	}

	ranked, _ := s.Rank(ctx, models)

	assert.Equal(t, Tier1, ranked[0].Tier)
	assert.Equal(t, Tier2, ranked[1].Tier)
	assert.Equal(t, Tier3, ranked[2].Tier)
}

func TestDefaultStrategy_Select(t *testing.T) {
	ctx := context.Background()
	s := NewDefaultStrategy()

	ranked := []RankedModel{
		{
			Rank: 1,
			Model: ModelInfo{
				ID:               "best",
				QualityScore:     0.95,
				ReliabilityScore: 0.98,
				SupportsVision:   true,
			},
		},
		{
			Rank: 2,
			Model: ModelInfo{
				ID:               "second",
				QualityScore:     0.8,
				ReliabilityScore: 0.9,
				SupportsVision:   false,
			},
		},
	}

	req := Requirements{
		NeedsVision: true,
	}

	selected, err := s.Select(ctx, ranked, req)
	require.NoError(t, err)
	assert.Equal(t, "best", selected.ID)
}

func TestDefaultStrategy_Select_PreferredProvider(t *testing.T) {
	ctx := context.Background()
	s := NewDefaultStrategy()

	ranked := []RankedModel{
		{
			Rank: 1,
			Model: ModelInfo{
				ID:               "openai-best",
				Provider:         "openai",
				QualityScore:     0.95,
				ReliabilityScore: 0.98,
			},
		},
		{
			Rank: 2,
			Model: ModelInfo{
				ID:               "anthropic-good",
				Provider:         "anthropic",
				QualityScore:     0.9,
				ReliabilityScore: 0.95,
			},
		},
	}

	req := Requirements{
		PreferredProvider: "anthropic",
	}

	selected, err := s.Select(ctx, ranked, req)
	require.NoError(t, err)
	assert.Equal(t, "anthropic", selected.Provider)
}

func TestDefaultStrategy_Select_NoMatchingRequirements(t *testing.T) {
	ctx := context.Background()
	s := NewDefaultStrategy()

	ranked := []RankedModel{
		{
			Model: ModelInfo{
				ID:               "no-vision",
				SupportsVision:   false,
				QualityScore:     0.9,
				ReliabilityScore: 0.95,
			},
		},
	}

	req := Requirements{
		NeedsVision: true,
	}

	_, err := s.Select(ctx, ranked, req)
	assert.Error(t, err)
}

func TestDefaultStrategy_Select_EmptyRankedList(t *testing.T) {
	ctx := context.Background()
	s := NewDefaultStrategy()

	req := Requirements{}

	_, err := s.Select(ctx, []RankedModel{}, req)
	assert.Error(t, err)
}

func TestDefaultStrategy_WithWeights(t *testing.T) {
	s := NewDefaultStrategy(WithWeights(map[string]float64{
		DimensionQuality: 0.5,
		DimensionSpeed:   0.5,
	}))

	assert.Equal(t, 0.5, s.weights[DimensionQuality])
	assert.Equal(t, 0.5, s.weights[DimensionSpeed])
}

func TestDefaultStrategy_SetWeights(t *testing.T) {
	s := NewDefaultStrategy()
	s.SetWeights(map[string]float64{
		DimensionVision: 0.4,
	})

	assert.Equal(t, 0.4, s.weights[DimensionVision])
}

func TestDefaultStrategy_FilterByRequirements(t *testing.T) {
	s := NewDefaultStrategy()

	ranked := []RankedModel{
		{Model: ModelInfo{ID: "1", SupportsVision: true, ContextWindow: 128000}},
		{Model: ModelInfo{ID: "2", SupportsVision: false, ContextWindow: 4000}},
		{Model: ModelInfo{ID: "3", SupportsVision: true, ContextWindow: 32000}},
	}

	req := Requirements{
		NeedsVision:      true,
		MinContextWindow: 30000,
	}

	filtered := s.filterByRequirements(ranked, req)

	assert.Len(t, filtered, 2)
	assert.Equal(t, "1", filtered[0].Model.ID)
	assert.Equal(t, "3", filtered[1].Model.ID)
}

func TestDefaultStrategy_FilterByRequirements_ExcludedProvider(t *testing.T) {
	s := NewDefaultStrategy()

	ranked := []RankedModel{
		{Model: ModelInfo{ID: "1", Provider: "openai"}},
		{Model: ModelInfo{ID: "2", Provider: "anthropic"}},
		{Model: ModelInfo{ID: "3", Provider: "google"}},
	}

	req := Requirements{
		ExcludedProviders: []string{"anthropic"},
	}

	filtered := s.filterByRequirements(ranked, req)

	assert.Len(t, filtered, 2)
	for _, f := range filtered {
		assert.NotEqual(t, "anthropic", f.Model.Provider)
	}
}

func TestDefaultStrategy_FilterByRequirements_RequiredCapabilities(t *testing.T) {
	s := NewDefaultStrategy()

	ranked := []RankedModel{
		{Model: ModelInfo{ID: "1", Capabilities: []string{"code", "math"}}},
		{Model: ModelInfo{ID: "2", Capabilities: []string{"chat"}}},
		{Model: ModelInfo{ID: "3", Capabilities: []string{"code", "vision"}}},
	}

	req := Requirements{
		RequiredCapabilities: []string{"code"},
	}

	filtered := s.filterByRequirements(ranked, req)

	assert.Len(t, filtered, 2)
}
