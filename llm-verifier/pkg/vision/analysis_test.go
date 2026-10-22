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

func TestAnalysisStrategy_Name(t *testing.T) {
	s := NewAnalysisStrategy()
	assert.Equal(t, "analysis", s.Name())
}

func TestAnalysisStrategy_Description(t *testing.T) {
	s := NewAnalysisStrategy()
	desc := s.Description()
	assert.Contains(t, desc, "description_quality")
	assert.Contains(t, desc, "ocr")
	assert.Contains(t, desc, "object_detection")
	assert.Contains(t, desc, "comprehensiveness")
	assert.Contains(t, desc, "cost")
}

// --- Score: Astica scores HIGHEST ---

func TestAnalysisStrategy_AsticaScoresHighest(t *testing.T) {
	ctx := context.Background()
	s := NewAnalysisStrategy()

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

	score, err := s.Score(ctx, astica)
	require.NoError(t, err)
	assert.Greater(t, score.Overall, 0.9,
		"Astica should score > 0.9 for analysis (all vision capabilities)")
	assert.Equal(t, "analysis", score.StrategyName)
}

// --- Score: Gemini scores ~0.7 ---

func TestAnalysisStrategy_GeminiScoresMediumHigh(t *testing.T) {
	ctx := context.Background()
	s := NewAnalysisStrategy()

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
		// No ocr, object_detection, face_detection,
		// content_moderation.
	}

	score, err := s.Score(ctx, gemini)
	require.NoError(t, err)
	assert.Greater(t, score.Overall, 0.4,
		"Gemini should score reasonably for analysis")
	assert.Less(t, score.Overall, 0.85,
		"Gemini should score below Astica for analysis")
}

// --- Score: Astica beats Gemini for analysis ---

func TestAnalysisStrategy_AsticaBeatsGemini(t *testing.T) {
	ctx := context.Background()
	s := NewAnalysisStrategy()

	astica := strategy.ModelInfo{
		ID: "astica", Provider: "astica",
		SupportsVision: true, QualityScore: 0.97,
		AvgLatencyMs:    800,
		InputCostPer1k:  0.0005,
		OutputCostPer1k: 0.0005,
		Verified:        true,
		LastVerified:    time.Now(),
		Capabilities: []string{
			"vision", "ocr", "object_detection",
			"face_detection", "content_moderation",
			"gui_analysis",
		},
	}

	gemini := strategy.ModelInfo{
		ID: "gemini", Provider: "google",
		SupportsVision: true, QualityScore: 0.88,
		AvgLatencyMs:    800,
		InputCostPer1k:  0.0001,
		OutputCostPer1k: 0.0004,
		Verified:        true,
		LastVerified:    time.Now(),
		Capabilities: []string{
			"vision", "gui_analysis", "json_output",
		},
	}

	asticaScore, _ := s.Score(ctx, astica)
	geminiScore, _ := s.Score(ctx, gemini)

	assert.Greater(t, asticaScore.Overall, geminiScore.Overall,
		"Astica must beat Gemini for analysis (rich vision capabilities)")
}

// --- Score: Non-vision model excluded ---

func TestAnalysisStrategy_NonVisionModelExcluded(t *testing.T) {
	ctx := context.Background()
	s := NewAnalysisStrategy()

	textOnly := strategy.ModelInfo{
		ID:             "text-only",
		Provider:       "openai",
		SupportsVision: false,
		QualityScore:   0.95,
	}

	score, err := s.Score(ctx, textOnly)
	require.NoError(t, err)
	assert.Equal(t, 0.0, score.Overall,
		"Non-vision models should score 0 for analysis")
}

// --- Score: OCR capability bonus ---

func TestAnalysisStrategy_OCRCapabilityBonus(t *testing.T) {
	ctx := context.Background()
	s := NewAnalysisStrategy()

	withOCR := strategy.ModelInfo{
		ID: "with-ocr", SupportsVision: true,
		QualityScore: 0.85,
		Capabilities: []string{"vision", "ocr"},
	}

	withoutOCR := strategy.ModelInfo{
		ID: "without-ocr", SupportsVision: true,
		QualityScore: 0.85,
		Capabilities: []string{"vision"},
	}

	ocrScore, _ := s.Score(ctx, withOCR)
	noOcrScore, _ := s.Score(ctx, withoutOCR)

	assert.Greater(t, ocrScore.Overall, noOcrScore.Overall,
		"Model with OCR should score higher for analysis")
}

// --- Score: Object detection bonus ---

func TestAnalysisStrategy_ObjectDetectionBonus(t *testing.T) {
	ctx := context.Background()
	s := NewAnalysisStrategy()

	withObj := strategy.ModelInfo{
		ID: "with-obj", SupportsVision: true,
		QualityScore: 0.85,
		Capabilities: []string{"vision", "object_detection"},
	}

	withoutObj := strategy.ModelInfo{
		ID: "without-obj", SupportsVision: true,
		QualityScore: 0.85,
		Capabilities: []string{"vision"},
	}

	objScore, _ := s.Score(ctx, withObj)
	noObjScore, _ := s.Score(ctx, withoutObj)

	assert.Greater(t, objScore.Overall, noObjScore.Overall,
		"Model with object detection should score higher for analysis")
}

// --- Score: Comprehensiveness bonus ---

func TestAnalysisStrategy_ComprehensivenessBonus(t *testing.T) {
	ctx := context.Background()
	s := NewAnalysisStrategy()

	comprehensive := strategy.ModelInfo{
		ID: "comprehensive", SupportsVision: true,
		QualityScore: 0.85,
		Capabilities: []string{
			"vision", "ocr", "object_detection",
			"face_detection", "content_moderation",
			"gui_analysis",
		},
	}

	minimal := strategy.ModelInfo{
		ID: "minimal", SupportsVision: true,
		QualityScore: 0.85,
		Capabilities: []string{"vision"},
	}

	compScore, _ := s.Score(ctx, comprehensive)
	minScore, _ := s.Score(ctx, minimal)

	assert.Greater(t, compScore.Overall, minScore.Overall,
		"Model with more vision capabilities should score higher")
}

// --- Validate: vision required ---

func TestAnalysisStrategy_Validate_NonVision(t *testing.T) {
	ctx := context.Background()
	s := NewAnalysisStrategy()

	model := strategy.ModelInfo{
		ID: "no-vision", SupportsVision: false,
		QualityScore: 0.95,
	}

	result := s.Validate(ctx, model)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

// --- Validate: warning for missing OCR ---

func TestAnalysisStrategy_Validate_MissingOCR(t *testing.T) {
	ctx := context.Background()
	s := NewAnalysisStrategy()

	model := strategy.ModelInfo{
		ID: "no-ocr", SupportsVision: true,
		QualityScore: 0.85,
		Capabilities: []string{"vision"},
	}

	result := s.Validate(ctx, model)
	assert.True(t, result.Valid)
	assert.NotEmpty(t, result.Warnings,
		"Should warn about missing OCR capability")
}

// --- Rank: Astica first ---

func TestAnalysisStrategy_RankingOrder(t *testing.T) {
	ctx := context.Background()
	s := NewAnalysisStrategy()

	models := []strategy.ModelInfo{
		{
			ID: "gemini", Provider: "google",
			SupportsVision: true, QualityScore: 0.88,
			Capabilities: []string{"vision", "gui_analysis"},
		},
		{
			ID: "astica", Provider: "astica",
			SupportsVision: true, QualityScore: 0.97,
			Capabilities: []string{
				"vision", "ocr", "object_detection",
				"face_detection", "content_moderation",
				"gui_analysis",
			},
		},
		{
			ID: "text-only", Provider: "openai",
			SupportsVision: false, QualityScore: 0.95,
		},
	}

	ranked, err := s.Rank(ctx, models)
	require.NoError(t, err)
	require.Len(t, ranked, 3)

	assert.Equal(t, "astica", ranked[0].Model.ID,
		"Astica should rank first for analysis")
	assert.Equal(t, "text-only", ranked[len(ranked)-1].Model.ID,
		"Text-only should rank last")
}

// --- Select: best from ranked ---

func TestAnalysisStrategy_Select_Best(t *testing.T) {
	ctx := context.Background()
	s := NewAnalysisStrategy()

	ranked := []strategy.RankedModel{
		{
			Rank:  1,
			Score: strategy.StrategyScore{Overall: 0.95},
			Model: strategy.ModelInfo{
				ID: "best", SupportsVision: true,
			},
		},
	}

	selected, err := s.Select(ctx, ranked, strategy.Requirements{})
	require.NoError(t, err)
	assert.Equal(t, "best", selected.ID)
}

// --- Select: empty list ---

func TestAnalysisStrategy_Select_EmptyList(t *testing.T) {
	ctx := context.Background()
	s := NewAnalysisStrategy()

	_, err := s.Select(ctx, nil, strategy.Requirements{})
	assert.Error(t, err)
}

// --- Interface compliance ---

func TestAnalysisStrategy_ImplementsInterface(t *testing.T) {
	var _ strategy.VerificationStrategy = (*AnalysisStrategy)(nil)
}

// --- DefaultAnalysisStrategyConfig ---

func TestDefaultAnalysisStrategyConfig(t *testing.T) {
	cfg := DefaultAnalysisStrategyConfig()
	assert.Equal(t, 0.35, cfg.DescriptionQualityWeight)
	assert.Equal(t, 0.20, cfg.OCRAccuracyWeight)
	assert.Equal(t, 0.20, cfg.ObjectDetectionWeight)
	assert.Equal(t, 0.15, cfg.ComprehensivenessWeight)
	assert.Equal(t, 0.10, cfg.CostWeight)

	total := cfg.DescriptionQualityWeight + cfg.OCRAccuracyWeight +
		cfg.ObjectDetectionWeight + cfg.ComprehensivenessWeight +
		cfg.CostWeight
	assert.InDelta(t, 1.0, total, 0.001)
}

// --- Caching ---

func TestAnalysisStrategy_Score_Caching(t *testing.T) {
	ctx := context.Background()
	s := NewAnalysisStrategy()

	model := strategy.ModelInfo{
		ID: "cached", SupportsVision: true,
		QualityScore: 0.85,
		Capabilities: []string{"vision", "ocr"},
	}

	score1, _ := s.Score(ctx, model)
	score2, _ := s.Score(ctx, model)

	assert.Equal(t, score1.Timestamp, score2.Timestamp,
		"Second call should return cached score")
}

// --- Free model cost bonus ---

func TestAnalysisStrategy_FreeCostBonus(t *testing.T) {
	ctx := context.Background()
	s := NewAnalysisStrategy()

	free := strategy.ModelInfo{
		ID: "free", SupportsVision: true,
		QualityScore: 0.70,
		Capabilities: []string{"vision"},
	}

	paid := strategy.ModelInfo{
		ID: "paid", SupportsVision: true,
		QualityScore:    0.70,
		InputCostPer1k:  0.005,
		OutputCostPer1k: 0.015,
		Capabilities:    []string{"vision"},
	}

	freeScore, _ := s.Score(ctx, free)
	paidScore, _ := s.Score(ctx, paid)

	assert.Greater(t,
		freeScore.DimensionScores[strategy.DimensionCost],
		paidScore.DimensionScores[strategy.DimensionCost],
		"Free model should have higher cost dimension score")
}
