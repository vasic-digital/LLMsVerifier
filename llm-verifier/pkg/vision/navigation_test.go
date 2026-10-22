// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package vision

import (
	"context"
	"testing"
	"time"

	"digital.vasic.llmsverifier/pkg/strategy"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Name / Description ---

func TestNavigationStrategy_Name(t *testing.T) {
	s := NewNavigationStrategy()
	assert.Equal(t, "navigation", s.Name())
}

func TestNavigationStrategy_Description(t *testing.T) {
	s := NewNavigationStrategy()
	desc := s.Description()
	assert.Contains(t, desc, "json_compliance")
	assert.Contains(t, desc, "gui_understand")
	assert.Contains(t, desc, "speed")
	assert.Contains(t, desc, "cost")
}

// --- Score: Astica scores LOW (no JSON output) ---

func TestNavigationStrategy_AsticaScoresLow(t *testing.T) {
	ctx := context.Background()
	s := NewNavigationStrategy()

	astica := strategy.ModelInfo{
		ID:             "astica-vision",
		Provider:       "astica",
		SupportsVision: true,
		AvgLatencyMs:   800,
		InputCostPer1k: 0.0005, OutputCostPer1k: 0.0005,
		QualityScore:     0.97,
		ReliabilityScore: 0.95,
		Verified:         true,
		LastVerified:     time.Now(),
		Capabilities: []string{
			"vision", "ocr", "object_detection",
			"face_detection", "content_moderation",
			"gui_analysis",
		},
		// NOTE: No "json_output" or "structured_output" capability.
	}

	score, err := s.Score(ctx, astica)
	require.NoError(t, err)
	assert.Less(t, score.Overall, 0.5,
		"Astica should score low for navigation (no JSON output)")
	assert.Equal(t, "navigation", score.StrategyName)
}

// --- Score: Gemini scores HIGH (JSON + fast) ---

func TestNavigationStrategy_GeminiScoresHigh(t *testing.T) {
	ctx := context.Background()
	s := NewNavigationStrategy()

	gemini := strategy.ModelInfo{
		ID:             "gemini-flash",
		Provider:       "google",
		SupportsVision: true,
		AvgLatencyMs:   800,
		InputCostPer1k: 0.0001, OutputCostPer1k: 0.0004,
		QualityScore:     0.88,
		ReliabilityScore: 0.96,
		Verified:         true,
		LastVerified:     time.Now(),
		Capabilities: []string{
			"vision", "json_output", "structured_output",
			"gui_analysis", "code", "reasoning",
		},
	}

	score, err := s.Score(ctx, gemini)
	require.NoError(t, err)
	assert.Greater(t, score.Overall, 0.7,
		"Gemini should score high for navigation (JSON + fast)")
}

// --- Score: Kimi scores HIGH (JSON + cheap) ---

func TestNavigationStrategy_KimiScoresHigh(t *testing.T) {
	ctx := context.Background()
	s := NewNavigationStrategy()

	kimi := strategy.ModelInfo{
		ID:             "kimi-vision",
		Provider:       "moonshot",
		SupportsVision: true,
		AvgLatencyMs:   1200,
		InputCostPer1k: 0.0, OutputCostPer1k: 0.0,
		QualityScore:     0.80,
		ReliabilityScore: 0.90,
		Verified:         true,
		LastVerified:     time.Now(),
		Capabilities: []string{
			"vision", "json_output", "gui_analysis",
		},
	}

	score, err := s.Score(ctx, kimi)
	require.NoError(t, err)
	assert.Greater(t, score.Overall, 0.7,
		"Kimi should score high for navigation (JSON + cheap)")
}

// --- Score: Ollama scores medium (local, no cost, slower) ---

func TestNavigationStrategy_OllamaScoresMedium(t *testing.T) {
	ctx := context.Background()
	s := NewNavigationStrategy()

	ollama := strategy.ModelInfo{
		ID:             "ollama-llava",
		Provider:       "ollama",
		SupportsVision: true,
		AvgLatencyMs:   3000,
		InputCostPer1k: 0.0, OutputCostPer1k: 0.0,
		QualityScore:     0.65,
		ReliabilityScore: 0.80,
		Verified:         true,
		LastVerified:     time.Now(),
		Capabilities:     []string{"vision", "local"},
		// No json_output or structured_output.
	}

	score, err := s.Score(ctx, ollama)
	require.NoError(t, err)
	// Ollama lacks JSON output so it gets penalised, but free cost
	// helps. Should be between Astica-low and Gemini-high.
	assert.Greater(t, score.Overall, 0.2,
		"Ollama should score above zero")
	assert.Less(t, score.Overall, 0.7,
		"Ollama should score below JSON-capable models")
}

// --- Score: Non-vision model excluded ---

func TestNavigationStrategy_NonVisionModelExcluded(t *testing.T) {
	ctx := context.Background()
	s := NewNavigationStrategy()

	textOnly := strategy.ModelInfo{
		ID:               "text-only",
		Provider:         "openai",
		SupportsVision:   false,
		QualityScore:     0.95,
		ReliabilityScore: 0.98,
		Capabilities:     []string{"code", "reasoning"},
	}

	score, err := s.Score(ctx, textOnly)
	require.NoError(t, err)
	assert.Equal(t, 0.0, score.Overall,
		"Non-vision models should score 0 for navigation")
	assert.Less(t, score.Confidence, 0.2,
		"Non-vision models should have very low confidence")
}

// --- Score: Gemini beats Astica for navigation ---

func TestNavigationStrategy_GeminiBeatAstica(t *testing.T) {
	ctx := context.Background()
	s := NewNavigationStrategy()

	astica := strategy.ModelInfo{
		ID:             "astica-vision",
		Provider:       "astica",
		SupportsVision: true,
		AvgLatencyMs:   800,
		InputCostPer1k: 0.0005, OutputCostPer1k: 0.0005,
		QualityScore:     0.97,
		ReliabilityScore: 0.95,
		Verified:         true,
		LastVerified:     time.Now(),
		Capabilities: []string{
			"vision", "ocr", "object_detection",
			"face_detection", "content_moderation",
			"gui_analysis",
		},
	}

	gemini := strategy.ModelInfo{
		ID:             "gemini-flash",
		Provider:       "google",
		SupportsVision: true,
		AvgLatencyMs:   800,
		InputCostPer1k: 0.0001, OutputCostPer1k: 0.0004,
		QualityScore:     0.88,
		ReliabilityScore: 0.96,
		Verified:         true,
		LastVerified:     time.Now(),
		Capabilities: []string{
			"vision", "json_output", "structured_output",
			"gui_analysis", "code", "reasoning",
		},
	}

	asticaScore, _ := s.Score(ctx, astica)
	geminiScore, _ := s.Score(ctx, gemini)

	assert.Greater(t, geminiScore.Overall, asticaScore.Overall,
		"Gemini must beat Astica for navigation (JSON output is critical)")
}

// --- Ranking order matches expectations ---

func TestNavigationStrategy_RankingOrder(t *testing.T) {
	ctx := context.Background()
	s := NewNavigationStrategy()

	models := []strategy.ModelInfo{
		{
			ID: "astica", Provider: "astica",
			SupportsVision: true, QualityScore: 0.97,
			AvgLatencyMs:    800,
			InputCostPer1k:  0.0005,
			OutputCostPer1k: 0.0005,
			Capabilities: []string{
				"vision", "ocr", "object_detection",
				"face_detection", "content_moderation",
				"gui_analysis",
			},
		},
		{
			ID: "gemini-flash", Provider: "google",
			SupportsVision: true, QualityScore: 0.88,
			AvgLatencyMs:    800,
			InputCostPer1k:  0.0001,
			OutputCostPer1k: 0.0004,
			Capabilities: []string{
				"vision", "json_output", "structured_output",
				"gui_analysis",
			},
		},
		{
			ID: "text-only", Provider: "openai",
			SupportsVision: false, QualityScore: 0.95,
		},
		{
			ID: "ollama-llava", Provider: "ollama",
			SupportsVision: true, QualityScore: 0.65,
			AvgLatencyMs: 3000,
			Capabilities: []string{"vision", "local"},
		},
	}

	ranked, err := s.Rank(ctx, models)
	require.NoError(t, err)
	require.Len(t, ranked, 4)

	// Gemini first (JSON capable), Astica and Ollama after, text-only
	// last.
	assert.Equal(t, "gemini-flash", ranked[0].Model.ID,
		"Gemini should rank first (JSON + fast)")
	assert.Equal(t, "text-only", ranked[len(ranked)-1].Model.ID,
		"Text-only should rank last")
}

// --- Validate: vision required ---

func TestNavigationStrategy_Validate_NonVision(t *testing.T) {
	ctx := context.Background()
	s := NewNavigationStrategy()

	model := strategy.ModelInfo{
		ID: "no-vision", SupportsVision: false,
		QualityScore: 0.95,
	}

	result := s.Validate(ctx, model)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

// --- Validate: warning for missing JSON ---

func TestNavigationStrategy_Validate_MissingJSON(t *testing.T) {
	ctx := context.Background()
	s := NewNavigationStrategy()

	model := strategy.ModelInfo{
		ID: "no-json", SupportsVision: true,
		QualityScore: 0.85,
		Capabilities: []string{"vision", "ocr"},
	}

	result := s.Validate(ctx, model)
	assert.True(t, result.Valid)
	assert.NotEmpty(t, result.Warnings,
		"Should warn about missing JSON output capability")
}

// --- Select: best from ranked ---

func TestNavigationStrategy_Select_Best(t *testing.T) {
	ctx := context.Background()
	s := NewNavigationStrategy()

	ranked := []strategy.RankedModel{
		{
			Rank:  1,
			Score: strategy.StrategyScore{Overall: 0.9},
			Model: strategy.ModelInfo{
				ID: "best", SupportsVision: true,
			},
		},
		{
			Rank:  2,
			Score: strategy.StrategyScore{Overall: 0.5},
			Model: strategy.ModelInfo{
				ID: "second", SupportsVision: true,
			},
		},
	}

	selected, err := s.Select(ctx, ranked, strategy.Requirements{})
	require.NoError(t, err)
	assert.Equal(t, "best", selected.ID)
}

// --- Select: empty list ---

func TestNavigationStrategy_Select_EmptyList(t *testing.T) {
	ctx := context.Background()
	s := NewNavigationStrategy()

	_, err := s.Select(ctx, nil, strategy.Requirements{})
	assert.Error(t, err)
}

// --- Select: enforces vision ---

func TestNavigationStrategy_Select_EnforcesVision(t *testing.T) {
	ctx := context.Background()
	s := NewNavigationStrategy()

	ranked := []strategy.RankedModel{
		{
			Rank: 1,
			Model: strategy.ModelInfo{
				ID: "no-vision", SupportsVision: false,
			},
		},
	}

	_, err := s.Select(ctx, ranked, strategy.Requirements{})
	assert.Error(t, err,
		"Should reject non-vision models")
}

// --- Select: preferred provider ---

func TestNavigationStrategy_Select_PreferredProvider(t *testing.T) {
	ctx := context.Background()
	s := NewNavigationStrategy()

	ranked := []strategy.RankedModel{
		{
			Rank: 1,
			Model: strategy.ModelInfo{
				ID: "google", Provider: "google",
				SupportsVision: true,
			},
		},
		{
			Rank: 2,
			Model: strategy.ModelInfo{
				ID: "moonshot", Provider: "moonshot",
				SupportsVision: true,
			},
		},
	}

	req := strategy.Requirements{
		PreferredProvider: "moonshot",
	}

	selected, err := s.Select(ctx, ranked, req)
	require.NoError(t, err)
	assert.Equal(t, "moonshot", selected.Provider)
}

// --- Interface compliance ---

func TestNavigationStrategy_ImplementsInterface(t *testing.T) {
	var _ strategy.VerificationStrategy = (*NavigationStrategy)(nil)
}

// --- Caching ---

func TestNavigationStrategy_Score_Caching(t *testing.T) {
	ctx := context.Background()
	s := NewNavigationStrategy()

	model := strategy.ModelInfo{
		ID: "cached", SupportsVision: true,
		QualityScore: 0.85,
		Capabilities: []string{"vision", "json_output"},
	}

	score1, _ := s.Score(ctx, model)
	score2, _ := s.Score(ctx, model)

	assert.Equal(t, score1.Timestamp, score2.Timestamp,
		"Second call should return cached score")
}

func TestNavigationStrategy_ClearCache(t *testing.T) {
	ctx := context.Background()
	s := NewNavigationStrategy()

	model := strategy.ModelInfo{
		ID: "cache-test", SupportsVision: true,
		QualityScore: 0.85,
		Capabilities: []string{"vision", "json_output"},
	}

	score1, _ := s.Score(ctx, model)
	s.ClearCache()
	score2, _ := s.Score(ctx, model)

	assert.NotEqual(t, score1.Timestamp, score2.Timestamp,
		"After clearing cache, score should have a new timestamp")
}

// --- DefaultNavigationStrategyConfig ---

func TestDefaultNavigationStrategyConfig(t *testing.T) {
	cfg := DefaultNavigationStrategyConfig()
	assert.Equal(t, 0.40, cfg.JSONComplianceWeight)
	assert.Equal(t, 0.25, cfg.GUIUnderstandWeight)
	assert.Equal(t, 0.20, cfg.SpeedWeight)
	assert.Equal(t, 0.15, cfg.CostWeight)

	total := cfg.JSONComplianceWeight + cfg.GUIUnderstandWeight +
		cfg.SpeedWeight + cfg.CostWeight
	assert.InDelta(t, 1.0, total, 0.001)
}

// --- Custom config ---

func TestNavigationStrategy_CustomConfig(t *testing.T) {
	s := NewNavigationStrategy(func(cfg *NavigationStrategyConfig) {
		cfg.JSONComplianceWeight = 0.50
		cfg.GUIUnderstandWeight = 0.20
		cfg.SpeedWeight = 0.20
		cfg.CostWeight = 0.10
	})

	assert.Equal(t, 0.50, s.jsonComplianceWeight)
	assert.Equal(t, 0.20, s.guiUnderstandWeight)
	assert.Equal(t, 0.20, s.speedWeight)
	assert.Equal(t, 0.10, s.costWeight)
}

// --- Rank: selection probability sums to ~1.0 ---

func TestNavigationStrategy_Rank_SelectionProbability(t *testing.T) {
	ctx := context.Background()
	s := NewNavigationStrategy()

	models := []strategy.ModelInfo{
		{
			ID: "a", SupportsVision: true,
			QualityScore: 0.90,
			Capabilities: []string{"vision", "json_output"},
		},
		{
			ID: "b", SupportsVision: true,
			QualityScore: 0.70,
			Capabilities: []string{"vision"},
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

// --- Rank: empty list ---

func TestNavigationStrategy_Rank_EmptyList(t *testing.T) {
	ctx := context.Background()
	s := NewNavigationStrategy()

	ranked, err := s.Rank(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, ranked)
}
