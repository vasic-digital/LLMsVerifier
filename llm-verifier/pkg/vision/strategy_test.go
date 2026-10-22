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

func TestVisionStrategy_Name(t *testing.T) {
	s := NewVisionStrategy()
	assert.Equal(t, "vision", s.Name())
}

func TestVisionStrategy_Description(t *testing.T) {
	s := NewVisionStrategy()
	desc := s.Description()
	assert.Contains(t, desc, "image_quality")
	assert.Contains(t, desc, "gui_detection")
	assert.Contains(t, desc, "ocr")
	assert.Contains(t, desc, "speed")
	assert.Contains(t, desc, "cost")
}

// --- Score ---

func TestVisionStrategy_Score_VisionModel(t *testing.T) {
	ctx := context.Background()
	s := NewVisionStrategy()

	model := strategy.ModelInfo{
		ID:               "astica-vision",
		Name:             "Astica Vision",
		Provider:         "astica",
		Model:            "2.5_full",
		SupportsVision:   true,
		ContextWindow:    32768,
		AvgLatencyMs:     800,
		InputCostPer1k:   0.0005,
		OutputCostPer1k:  0.0005,
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

	score, err := s.Score(ctx, model)
	require.NoError(t, err)

	assert.Equal(t, "astica-vision", score.ModelID)
	assert.Equal(t, "vision", score.StrategyName)
	assert.Greater(t, score.Overall, 0.5)
	assert.LessOrEqual(t, score.Overall, 1.0)
	assert.NotEmpty(t, score.Reasoning)
	assert.Contains(t, score.DimensionScores, strategy.DimensionQuality)
	assert.Contains(t, score.DimensionScores, strategy.DimensionVision)
	assert.Contains(t, score.DimensionScores, strategy.DimensionSpeed)
	assert.Contains(t, score.DimensionScores, strategy.DimensionCost)
}

func TestVisionStrategy_Score_NonVisionModel(t *testing.T) {
	ctx := context.Background()
	s := NewVisionStrategy()

	model := strategy.ModelInfo{
		ID:               "text-only",
		Name:             "Text Only",
		Provider:         "openai",
		SupportsVision:   false,
		QualityScore:     0.95,
		ReliabilityScore: 0.98,
	}

	score, err := s.Score(ctx, model)
	require.NoError(t, err)

	assert.Equal(t, 0.0, score.Overall,
		"Non-vision models should score 0")
	assert.Less(t, score.Confidence, 0.2,
		"Non-vision models should have very low confidence")
}

func TestVisionStrategy_Score_AsticaHighest(t *testing.T) {
	ctx := context.Background()
	s := NewVisionStrategy()

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
	}

	asticaScore, _ := s.Score(ctx, astica)
	ollamaScore, _ := s.Score(ctx, ollama)

	assert.Greater(t, asticaScore.Overall, ollamaScore.Overall,
		"Astica should score higher than Ollama due to specialised vision capabilities")
}

func TestVisionStrategy_Score_LocalModelFreeCostBonus(t *testing.T) {
	ctx := context.Background()
	s := NewVisionStrategy()

	// Two models with same quality but different cost.
	free := strategy.ModelInfo{
		ID:             "free-model",
		Provider:       "ollama",
		SupportsVision: true,
		AvgLatencyMs:   2000,
		InputCostPer1k: 0.0, OutputCostPer1k: 0.0,
		QualityScore:     0.70,
		ReliabilityScore: 0.85,
		Capabilities:     []string{"vision"},
	}

	paid := strategy.ModelInfo{
		ID:             "paid-model",
		Provider:       "cloud",
		SupportsVision: true,
		AvgLatencyMs:   2000,
		InputCostPer1k: 0.005, OutputCostPer1k: 0.015,
		QualityScore:     0.70,
		ReliabilityScore: 0.85,
		Capabilities:     []string{"vision"},
	}

	freeScore, _ := s.Score(ctx, free)
	paidScore, _ := s.Score(ctx, paid)

	assert.Greater(t, freeScore.DimensionScores[strategy.DimensionCost],
		paidScore.DimensionScores[strategy.DimensionCost],
		"Free model should have higher cost dimension score")
}

func TestVisionStrategy_Score_GUISpecialisedBonus(t *testing.T) {
	ctx := context.Background()
	s := NewVisionStrategy()

	guiSpecialised := strategy.ModelInfo{
		ID:             "gui-model",
		Provider:       "stepfun",
		SupportsVision: true,
		AvgLatencyMs:   900,
		QualityScore:   0.82,
		Capabilities: []string{
			"vision", "gui_grounding",
			"gui_navigation", "element_detection",
		},
	}

	general := strategy.ModelInfo{
		ID:             "general-model",
		Provider:       "openai",
		SupportsVision: true,
		AvgLatencyMs:   900,
		QualityScore:   0.82,
		Capabilities:   []string{"vision"},
	}

	guiScore, _ := s.Score(ctx, guiSpecialised)
	generalScore, _ := s.Score(ctx, general)

	assert.Greater(t,
		guiScore.DimensionScores[strategy.DimensionVision],
		generalScore.DimensionScores[strategy.DimensionVision],
		"GUI-specialised model should score higher on gui_detection")
}

func TestVisionStrategy_Score_VerifiedModel(t *testing.T) {
	ctx := context.Background()
	s := NewVisionStrategy()

	verified := strategy.ModelInfo{
		ID:             "verified",
		SupportsVision: true,
		QualityScore:   0.85,
		Verified:       true,
		LastVerified:   time.Now(),
		Capabilities:   []string{"vision"},
	}

	unverified := strategy.ModelInfo{
		ID:             "unverified",
		SupportsVision: true,
		QualityScore:   0.85,
		Verified:       false,
		Capabilities:   []string{"vision"},
	}

	vScore, _ := s.Score(ctx, verified)
	uvScore, _ := s.Score(ctx, unverified)

	assert.Greater(t, vScore.Confidence, uvScore.Confidence,
		"Verified model should have higher confidence")
}

func TestVisionStrategy_Score_Caching(t *testing.T) {
	ctx := context.Background()
	s := NewVisionStrategy()

	model := strategy.ModelInfo{
		ID:             "cached",
		SupportsVision: true,
		QualityScore:   0.85,
		Capabilities:   []string{"vision"},
	}

	score1, _ := s.Score(ctx, model)
	score2, _ := s.Score(ctx, model)

	assert.Equal(t, score1.Timestamp, score2.Timestamp,
		"Second call should return cached score")
}

func TestVisionStrategy_Score_OCRCapabilityBonus(t *testing.T) {
	ctx := context.Background()
	s := NewVisionStrategy()

	withOCR := strategy.ModelInfo{
		ID:             "with-ocr",
		SupportsVision: true,
		QualityScore:   0.85,
		Capabilities:   []string{"vision", "ocr"},
	}

	withoutOCR := strategy.ModelInfo{
		ID:             "without-ocr",
		SupportsVision: true,
		QualityScore:   0.85,
		Capabilities:   []string{"vision"},
	}

	ocrScore, _ := s.Score(ctx, withOCR)
	noOcrScore, _ := s.Score(ctx, withoutOCR)

	assert.Greater(t,
		ocrScore.DimensionScores[strategy.DimensionContext],
		noOcrScore.DimensionScores[strategy.DimensionContext],
		"Model with OCR capability should score higher on OCR dimension")
}

// --- Validate ---

func TestVisionStrategy_Validate_VisionModel(t *testing.T) {
	ctx := context.Background()
	s := NewVisionStrategy()

	model := strategy.ModelInfo{
		ID:               "valid",
		SupportsVision:   true,
		QualityScore:     0.85,
		ReliabilityScore: 0.90,
		Capabilities:     []string{"vision"},
	}

	result := s.Validate(ctx, model)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestVisionStrategy_Validate_NonVisionModel(t *testing.T) {
	ctx := context.Background()
	s := NewVisionStrategy()

	model := strategy.ModelInfo{
		ID:               "no-vision",
		SupportsVision:   false,
		QualityScore:     0.95,
		ReliabilityScore: 0.98,
	}

	result := s.Validate(ctx, model)

	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

func TestVisionStrategy_Validate_LowQualityWarning(t *testing.T) {
	ctx := context.Background()
	s := NewVisionStrategy()

	model := strategy.ModelInfo{
		ID:             "low-quality",
		SupportsVision: true,
		QualityScore:   0.4,
		Capabilities:   []string{"vision"},
	}

	result := s.Validate(ctx, model)

	assert.True(t, result.Valid)
	assert.NotEmpty(t, result.Warnings)
}

func TestVisionStrategy_Validate_HighLatencyWarning(t *testing.T) {
	ctx := context.Background()
	s := NewVisionStrategy()

	model := strategy.ModelInfo{
		ID:             "slow",
		SupportsVision: true,
		QualityScore:   0.85,
		AvgLatencyMs:   6000,
		Capabilities:   []string{"vision"},
	}

	result := s.Validate(ctx, model)

	assert.True(t, result.Valid)
	assert.NotEmpty(t, result.Warnings)
}

// --- Rank ---

func TestVisionStrategy_Rank_VisionFirst(t *testing.T) {
	ctx := context.Background()
	s := NewVisionStrategy()

	models := []strategy.ModelInfo{
		{
			ID: "text-only", SupportsVision: false,
			QualityScore: 0.95, ReliabilityScore: 0.98,
		},
		{
			ID: "vision-low", SupportsVision: true,
			QualityScore: 0.60, ReliabilityScore: 0.80,
			Capabilities: []string{"vision"},
		},
		{
			ID: "vision-high", SupportsVision: true,
			QualityScore: 0.90, ReliabilityScore: 0.95,
			Capabilities: []string{"vision", "gui_analysis"},
		},
	}

	ranked, err := s.Rank(ctx, models)
	require.NoError(t, err)
	require.Len(t, ranked, 3)

	assert.Equal(t, "vision-high", ranked[0].Model.ID)
	assert.Equal(t, "vision-low", ranked[1].Model.ID)
	assert.Equal(t, "text-only", ranked[2].Model.ID)
}

func TestVisionStrategy_Rank_Tiers(t *testing.T) {
	ctx := context.Background()
	s := NewVisionStrategy()

	models := []strategy.ModelInfo{
		{
			ID: "tier1", SupportsVision: true,
			QualityScore: 0.97, ReliabilityScore: 0.95,
			AvgLatencyMs: 800,
			Capabilities: []string{
				"vision", "ocr", "object_detection",
				"gui_analysis",
			},
		},
		{
			ID: "tier2", SupportsVision: true,
			QualityScore: 0.70, ReliabilityScore: 0.85,
			Capabilities: []string{"vision"},
		},
		{
			ID: "tier3", SupportsVision: false,
			QualityScore: 0.95, ReliabilityScore: 0.98,
		},
	}

	ranked, _ := s.Rank(ctx, models)
	require.Len(t, ranked, 3)

	assert.Equal(t, strategy.Tier1, ranked[0].Tier)
	assert.Equal(t, strategy.Tier2, ranked[1].Tier)
	assert.Equal(t, strategy.Tier3, ranked[2].Tier)
}

func TestVisionStrategy_Rank_SelectionProbability(t *testing.T) {
	ctx := context.Background()
	s := NewVisionStrategy()

	models := []strategy.ModelInfo{
		{
			ID: "a", SupportsVision: true,
			QualityScore: 0.90, Capabilities: []string{"vision"},
		},
		{
			ID: "b", SupportsVision: true,
			QualityScore: 0.70, Capabilities: []string{"vision"},
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

func TestVisionStrategy_Rank_EmptyList(t *testing.T) {
	ctx := context.Background()
	s := NewVisionStrategy()

	ranked, err := s.Rank(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, ranked)
}

// --- Select ---

func TestVisionStrategy_Select_BestVision(t *testing.T) {
	ctx := context.Background()
	s := NewVisionStrategy()

	ranked := []strategy.RankedModel{
		{
			Rank:  1,
			Score: strategy.StrategyScore{Overall: 0.9},
			Model: strategy.ModelInfo{
				ID: "best", SupportsVision: true,
				QualityScore: 0.95,
			},
		},
		{
			Rank:  2,
			Score: strategy.StrategyScore{Overall: 0.7},
			Model: strategy.ModelInfo{
				ID: "second", SupportsVision: true,
				QualityScore: 0.80,
			},
		},
	}

	selected, err := s.Select(ctx, ranked, strategy.Requirements{})
	require.NoError(t, err)
	assert.Equal(t, "best", selected.ID)
}

func TestVisionStrategy_Select_EnforcesVision(t *testing.T) {
	ctx := context.Background()
	s := NewVisionStrategy()

	ranked := []strategy.RankedModel{
		{
			Rank: 1,
			Model: strategy.ModelInfo{
				ID: "no-vision", SupportsVision: false,
				QualityScore: 0.99,
			},
		},
	}

	_, err := s.Select(ctx, ranked, strategy.Requirements{})
	assert.Error(t, err,
		"Should reject non-vision models")
}

func TestVisionStrategy_Select_PreferredProvider(t *testing.T) {
	ctx := context.Background()
	s := NewVisionStrategy()

	ranked := []strategy.RankedModel{
		{
			Rank: 1,
			Model: strategy.ModelInfo{
				ID: "openai", Provider: "openai",
				SupportsVision: true, QualityScore: 0.95,
			},
		},
		{
			Rank: 2,
			Model: strategy.ModelInfo{
				ID: "ollama", Provider: "ollama",
				SupportsVision: true, QualityScore: 0.65,
			},
		},
	}

	req := strategy.Requirements{
		PreferredProvider: "ollama",
	}

	selected, err := s.Select(ctx, ranked, req)
	require.NoError(t, err)
	assert.Equal(t, "ollama", selected.Provider,
		"Preferred provider should be honoured")
}

func TestVisionStrategy_Select_EmptyList(t *testing.T) {
	ctx := context.Background()
	s := NewVisionStrategy()

	_, err := s.Select(ctx, nil, strategy.Requirements{})
	assert.Error(t, err)
}

func TestVisionStrategy_Select_ExcludedProvider(t *testing.T) {
	ctx := context.Background()
	s := NewVisionStrategy()

	ranked := []strategy.RankedModel{
		{
			Rank: 1,
			Model: strategy.ModelInfo{
				ID: "excluded", Provider: "openai",
				SupportsVision: true, QualityScore: 0.95,
			},
		},
		{
			Rank: 2,
			Model: strategy.ModelInfo{
				ID: "allowed", Provider: "google",
				SupportsVision: true, QualityScore: 0.88,
			},
		},
	}

	req := strategy.Requirements{
		ExcludedProviders: []string{"openai"},
	}

	selected, err := s.Select(ctx, ranked, req)
	require.NoError(t, err)
	assert.Equal(t, "google", selected.Provider)
}

// --- SetWeights ---

func TestVisionStrategy_SetWeights(t *testing.T) {
	s := NewVisionStrategy()
	s.SetWeights(map[string]float64{
		strategy.DimensionQuality: 0.50,
		strategy.DimensionVision:  0.30,
		strategy.DimensionCost:    0.20,
	})

	assert.Equal(t, 0.50, s.imageQualityWeight)
	assert.Equal(t, 0.30, s.guiDetectionWeight)
	assert.Equal(t, 0.20, s.costWeight)
}

func TestVisionStrategy_CustomConfig(t *testing.T) {
	s := NewVisionStrategy(func(cfg *VisionStrategyConfig) {
		cfg.ImageQualityWeight = 0.50
		cfg.GUIDetectionWeight = 0.20
		cfg.OCRAccuracyWeight = 0.10
		cfg.SpeedWeight = 0.10
		cfg.CostWeight = 0.10
	})

	assert.Equal(t, 0.50, s.imageQualityWeight)
	assert.Equal(t, 0.20, s.guiDetectionWeight)
	assert.Equal(t, 0.10, s.ocrAccuracyWeight)
}

// --- ClearCache ---

func TestVisionStrategy_ClearCache(t *testing.T) {
	ctx := context.Background()
	s := NewVisionStrategy()

	model := strategy.ModelInfo{
		ID:             "cache-test",
		SupportsVision: true,
		QualityScore:   0.85,
		Capabilities:   []string{"vision"},
	}

	score1, _ := s.Score(ctx, model)
	s.ClearCache()
	score2, _ := s.Score(ctx, model)

	assert.NotEqual(t, score1.Timestamp, score2.Timestamp,
		"After clearing cache, score should have a new timestamp")
}

// --- DefaultVisionStrategyConfig ---

func TestDefaultVisionStrategyConfig(t *testing.T) {
	cfg := DefaultVisionStrategyConfig()
	assert.Equal(t, 0.35, cfg.ImageQualityWeight)
	assert.Equal(t, 0.25, cfg.GUIDetectionWeight)
	assert.Equal(t, 0.15, cfg.OCRAccuracyWeight)
	assert.Equal(t, 0.15, cfg.SpeedWeight)
	assert.Equal(t, 0.10, cfg.CostWeight)

	// Weights should sum to 1.0.
	total := cfg.ImageQualityWeight + cfg.GUIDetectionWeight +
		cfg.OCRAccuracyWeight + cfg.SpeedWeight + cfg.CostWeight
	assert.InDelta(t, 1.0, total, 0.001)
}

// --- Capability helpers ---

func TestCountCapabilities(t *testing.T) {
	caps := []string{"vision", "ocr", "gui_analysis"}

	assert.Equal(t, 2,
		countCapabilities(caps, "vision", "ocr", "face_detection"))
	assert.Equal(t, 0,
		countCapabilities(caps, "code", "reasoning"))
	assert.Equal(t, 0,
		countCapabilities(nil, "vision"))
}

func TestHasCapability(t *testing.T) {
	caps := []string{"vision", "ocr"}

	assert.True(t, hasCapability(caps, "vision"))
	assert.True(t, hasCapability(caps, "ocr"))
	assert.False(t, hasCapability(caps, "code"))
	assert.False(t, hasCapability(nil, "vision"))
}

// --- Integration: Score against VisionModelRegistry ---

func TestVisionStrategy_ScoresRegistryModels(t *testing.T) {
	ctx := context.Background()
	s := NewVisionStrategy()

	// Representative models from the registry.
	models := []strategy.ModelInfo{
		{
			ID: "astica", Provider: "astica",
			SupportsVision: true, QualityScore: 0.97,
			ReliabilityScore: 0.95, AvgLatencyMs: 800,
			InputCostPer1k: 0.0005, OutputCostPer1k: 0.0005,
			Capabilities: []string{
				"vision", "ocr", "object_detection",
				"face_detection", "content_moderation",
				"gui_analysis",
			},
		},
		{
			ID: "openai-gpt4o", Provider: "openai",
			SupportsVision: true, QualityScore: 0.95,
			ReliabilityScore: 0.98, AvgLatencyMs: 1200,
			InputCostPer1k: 0.005, OutputCostPer1k: 0.015,
			Capabilities: []string{"vision", "code", "reasoning", "gui_analysis"},
		},
		{
			ID: "gemini-flash", Provider: "google",
			SupportsVision: true, QualityScore: 0.88,
			ReliabilityScore: 0.96, AvgLatencyMs: 800,
			InputCostPer1k: 0.0001, OutputCostPer1k: 0.0004,
			Capabilities: []string{"vision", "code", "reasoning", "gui_analysis"},
		},
		{
			ID: "ollama-llava", Provider: "ollama",
			SupportsVision: true, QualityScore: 0.65,
			ReliabilityScore: 0.80, AvgLatencyMs: 3000,
			InputCostPer1k: 0.0, OutputCostPer1k: 0.0,
			Capabilities: []string{"vision", "local"},
		},
	}

	ranked, err := s.Rank(ctx, models)
	require.NoError(t, err)
	require.Len(t, ranked, 4)

	// Gemini (vision-only designated) or Astica (specialised) should rank first.
	// With DefaultProviderUsageConfig, Google gets UsageTypeVisionOnly +15% bonus.
	topProvider := ranked[0].Model.Provider
	assert.Contains(t, []string{"google", "astica"}, topProvider,
		"Top vision model should be Gemini (vision-only bonus) or Astica (specialized)")

	// All should have positive scores.
	for _, r := range ranked {
		assert.Greater(t, r.Score.Overall, 0.0,
			"Vision model %s should have positive score", r.Model.ID)
	}

	// Ollama should be last (lowest quality but vision-capable).
	assert.Equal(t, "ollama-llava", ranked[len(ranked)-1].Model.ID,
		"Ollama should rank last among vision models")
}

// --- VerificationStrategy interface compliance ---

func TestVisionStrategy_ImplementsInterface(t *testing.T) {
	var _ strategy.VerificationStrategy = (*VisionStrategy)(nil)
}
