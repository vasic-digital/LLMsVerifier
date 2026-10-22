// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"context"
	"testing"
	"time"

	"digital.vasic.llmsverifier/pkg/strategy"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Name / Description ---

func TestPlanningStrategy_Name(t *testing.T) {
	s := NewPlanningStrategy()
	assert.Equal(t, "planning", s.Name())
}

func TestPlanningStrategy_Description(t *testing.T) {
	s := NewPlanningStrategy()
	desc := s.Description()
	assert.Contains(t, desc, "reasoning")
	assert.Contains(t, desc, "context")
	assert.Contains(t, desc, "structured_output")
	assert.Contains(t, desc, "speed")
	assert.Contains(t, desc, "cost")
}

// --- Score: Claude scores highest (reasoning + large context) ---

func TestPlanningStrategy_ClaudeScoresHighest(t *testing.T) {
	ctx := context.Background()
	s := NewPlanningStrategy()

	claude := strategy.ModelInfo{
		ID:               "claude-opus",
		Provider:         "anthropic",
		SupportsVision:   true,
		ContextWindow:    1_000_000,
		AvgLatencyMs:     2000,
		InputCostPer1k:   0.015,
		OutputCostPer1k:  0.075,
		QualityScore:     0.95,
		ReliabilityScore: 0.98,
		Verified:         true,
		LastVerified:     time.Now(),
		Capabilities:     []string{"reasoning", "code", "json_output", "structured_output", "vision"},
	}

	score, err := s.Score(ctx, claude)
	require.NoError(t, err)
	assert.Greater(t, score.Overall, 0.7,
		"Claude should score high for planning (reasoning + large context)")
	assert.Equal(t, "planning", score.StrategyName)
}

// --- Score: Gemini close second (1M context, fast, cheap) ---

func TestPlanningStrategy_GeminiScoresHigh(t *testing.T) {
	ctx := context.Background()
	s := NewPlanningStrategy()

	gemini := strategy.ModelInfo{
		ID:               "gemini-flash",
		Provider:         "google",
		SupportsVision:   true,
		ContextWindow:    1_000_000,
		AvgLatencyMs:     800,
		InputCostPer1k:   0.0001,
		OutputCostPer1k:  0.0004,
		QualityScore:     0.88,
		ReliabilityScore: 0.96,
		Verified:         true,
		LastVerified:     time.Now(),
		Capabilities:     []string{"reasoning", "code", "json_output", "structured_output", "vision"},
	}

	score, err := s.Score(ctx, gemini)
	require.NoError(t, err)
	assert.Greater(t, score.Overall, 0.7,
		"Gemini should score high for planning (1M context + fast + cheap)")
}

// --- Score: Ollama scores low (small context, weaker reasoning) ---

func TestPlanningStrategy_OllamaScoresLow(t *testing.T) {
	ctx := context.Background()
	s := NewPlanningStrategy()

	ollama := strategy.ModelInfo{
		ID:               "ollama-llama",
		Provider:         "ollama",
		SupportsVision:   false,
		ContextWindow:    8192,
		AvgLatencyMs:     3000,
		InputCostPer1k:   0.0,
		OutputCostPer1k:  0.0,
		QualityScore:     0.55,
		ReliabilityScore: 0.80,
		Verified:         true,
		LastVerified:     time.Now(),
		Capabilities:     []string{"local"},
	}

	score, err := s.Score(ctx, ollama)
	require.NoError(t, err)
	assert.Less(t, score.Overall, 0.6,
		"Ollama should score low for planning (small context, no reasoning capability)")
}

// --- Score: Free model cost bonus ---

func TestPlanningStrategy_FreeCostBonus(t *testing.T) {
	ctx := context.Background()
	s := NewPlanningStrategy()

	free := strategy.ModelInfo{
		ID: "free-model", ContextWindow: 32000,
		QualityScore: 0.70,
		Capabilities: []string{"reasoning"},
	}

	paid := strategy.ModelInfo{
		ID: "paid-model", ContextWindow: 32000,
		QualityScore:    0.70,
		InputCostPer1k:  0.005,
		OutputCostPer1k: 0.015,
		Capabilities:    []string{"reasoning"},
	}

	freeScore, _ := s.Score(ctx, free)
	paidScore, _ := s.Score(ctx, paid)

	assert.Greater(t,
		freeScore.DimensionScores[strategy.DimensionCost],
		paidScore.DimensionScores[strategy.DimensionCost],
		"Free model should have higher cost dimension score")
}

// --- Score: Reasoning capability bonus ---

func TestPlanningStrategy_ReasoningCapabilityBonus(t *testing.T) {
	ctx := context.Background()
	s := NewPlanningStrategy()

	withReasoning := strategy.ModelInfo{
		ID: "with-reasoning", ContextWindow: 128000,
		QualityScore: 0.85,
		Capabilities: []string{"reasoning", "code"},
	}

	withoutReasoning := strategy.ModelInfo{
		ID: "without-reasoning", ContextWindow: 128000,
		QualityScore: 0.85,
		Capabilities: []string{"code"},
	}

	rScore, _ := s.Score(ctx, withReasoning)
	noRScore, _ := s.Score(ctx, withoutReasoning)

	assert.Greater(t, rScore.Overall, noRScore.Overall,
		"Model with reasoning capability should score higher for planning")
}

// --- Score: Large context window bonus ---

func TestPlanningStrategy_LargeContextBonus(t *testing.T) {
	ctx := context.Background()
	s := NewPlanningStrategy()

	largeCtx := strategy.ModelInfo{
		ID: "large-ctx", ContextWindow: 1_000_000,
		QualityScore: 0.85,
		Capabilities: []string{"reasoning"},
	}

	smallCtx := strategy.ModelInfo{
		ID: "small-ctx", ContextWindow: 8192,
		QualityScore: 0.85,
		Capabilities: []string{"reasoning"},
	}

	largeScore, _ := s.Score(ctx, largeCtx)
	smallScore, _ := s.Score(ctx, smallCtx)

	assert.Greater(t, largeScore.Overall, smallScore.Overall,
		"Model with larger context window should score higher for planning")
}

// --- Ranking: Claude/Gemini above Ollama ---

func TestPlanningStrategy_RankingOrder(t *testing.T) {
	ctx := context.Background()
	s := NewPlanningStrategy()

	models := []strategy.ModelInfo{
		{
			ID: "ollama", Provider: "ollama",
			ContextWindow: 8192, QualityScore: 0.55,
			Capabilities: []string{"local"},
		},
		{
			ID: "claude", Provider: "anthropic",
			ContextWindow: 1_000_000, QualityScore: 0.95,
			AvgLatencyMs: 2000,
			Capabilities: []string{"reasoning", "code", "json_output"},
		},
		{
			ID: "gemini", Provider: "google",
			ContextWindow: 1_000_000, QualityScore: 0.88,
			AvgLatencyMs: 800,
			Capabilities: []string{"reasoning", "code", "json_output"},
		},
	}

	ranked, err := s.Rank(ctx, models)
	require.NoError(t, err)
	require.Len(t, ranked, 3)

	// Claude or Gemini should be first; Ollama last.
	assert.NotEqual(t, "ollama", ranked[0].Model.ID,
		"Ollama should not rank first for planning")
	assert.Equal(t, "ollama", ranked[len(ranked)-1].Model.ID,
		"Ollama should rank last for planning")
}

// --- Validate: low quality warning ---

func TestPlanningStrategy_Validate_LowQuality(t *testing.T) {
	ctx := context.Background()
	s := NewPlanningStrategy()

	model := strategy.ModelInfo{
		ID: "low-quality", QualityScore: 0.4,
		ContextWindow: 32000,
		Capabilities:  []string{"reasoning"},
	}

	result := s.Validate(ctx, model)
	assert.True(t, result.Valid)
	assert.NotEmpty(t, result.Warnings,
		"Should warn about low quality score")
}

// --- Validate: small context warning ---

func TestPlanningStrategy_Validate_SmallContext(t *testing.T) {
	ctx := context.Background()
	s := NewPlanningStrategy()

	model := strategy.ModelInfo{
		ID: "small-ctx", QualityScore: 0.85,
		ContextWindow: 4096,
		Capabilities:  []string{"reasoning"},
	}

	result := s.Validate(ctx, model)
	assert.True(t, result.Valid)
	assert.NotEmpty(t, result.Warnings,
		"Should warn about small context window")
}

// --- Select: best from ranked ---

func TestPlanningStrategy_Select_Best(t *testing.T) {
	ctx := context.Background()
	s := NewPlanningStrategy()

	ranked := []strategy.RankedModel{
		{
			Rank:  1,
			Score: strategy.StrategyScore{Overall: 0.9},
			Model: strategy.ModelInfo{
				ID: "best", QualityScore: 0.95,
			},
		},
		{
			Rank:  2,
			Score: strategy.StrategyScore{Overall: 0.5},
			Model: strategy.ModelInfo{
				ID: "second", QualityScore: 0.70,
			},
		},
	}

	selected, err := s.Select(ctx, ranked, strategy.Requirements{})
	require.NoError(t, err)
	assert.Equal(t, "best", selected.ID)
}

// --- Select: empty list ---

func TestPlanningStrategy_Select_EmptyList(t *testing.T) {
	ctx := context.Background()
	s := NewPlanningStrategy()

	_, err := s.Select(ctx, nil, strategy.Requirements{})
	assert.Error(t, err)
}

// --- Select: preferred provider ---

func TestPlanningStrategy_Select_PreferredProvider(t *testing.T) {
	ctx := context.Background()
	s := NewPlanningStrategy()

	ranked := []strategy.RankedModel{
		{
			Rank:  1,
			Score: strategy.StrategyScore{Overall: 0.9},
			Model: strategy.ModelInfo{
				ID: "google", Provider: "google",
			},
		},
		{
			Rank:  2,
			Score: strategy.StrategyScore{Overall: 0.85},
			Model: strategy.ModelInfo{
				ID: "anthropic", Provider: "anthropic",
			},
		},
	}

	req := strategy.Requirements{
		PreferredProvider: "anthropic",
	}

	selected, err := s.Select(ctx, ranked, req)
	require.NoError(t, err)
	assert.Equal(t, "anthropic", selected.Provider)
}

// --- Interface compliance ---

func TestPlanningStrategy_ImplementsInterface(t *testing.T) {
	var _ strategy.VerificationStrategy = (*PlanningStrategy)(nil)
}

// --- DefaultPlanningStrategyConfig ---

func TestDefaultPlanningStrategyConfig(t *testing.T) {
	cfg := DefaultPlanningStrategyConfig()
	assert.Equal(t, 0.35, cfg.ReasoningWeight)
	assert.Equal(t, 0.25, cfg.ContextWeight)
	assert.Equal(t, 0.20, cfg.StructuredOutputWeight)
	assert.Equal(t, 0.10, cfg.SpeedWeight)
	assert.Equal(t, 0.10, cfg.CostWeight)

	total := cfg.ReasoningWeight + cfg.ContextWeight +
		cfg.StructuredOutputWeight + cfg.SpeedWeight + cfg.CostWeight
	assert.InDelta(t, 1.0, total, 0.001)
}

// --- Caching ---

func TestPlanningStrategy_Score_Caching(t *testing.T) {
	ctx := context.Background()
	s := NewPlanningStrategy()

	model := strategy.ModelInfo{
		ID: "cached", QualityScore: 0.85,
		ContextWindow: 128000,
		Capabilities:  []string{"reasoning"},
	}

	score1, _ := s.Score(ctx, model)
	score2, _ := s.Score(ctx, model)

	assert.Equal(t, score1.Timestamp, score2.Timestamp,
		"Second call should return cached score")
}

// --- Rank: selection probability sums to ~1.0 ---

func TestPlanningStrategy_Rank_SelectionProbability(t *testing.T) {
	ctx := context.Background()
	s := NewPlanningStrategy()

	models := []strategy.ModelInfo{
		{
			ID: "a", QualityScore: 0.90,
			ContextWindow: 128000,
			Capabilities:  []string{"reasoning"},
		},
		{
			ID: "b", QualityScore: 0.70,
			ContextWindow: 32000,
		},
	}

	ranked, _ := s.Rank(ctx, models)
	require.Len(t, ranked, 2)

	totalProb := 0.0
	for _, r := range ranked {
		totalProb += r.SelectionProbability
	}
	assert.InDelta(t, 1.0, totalProb, 0.01,
		"Selection probabilities should sum to ~1.0")
}
