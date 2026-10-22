// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package vision

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"digital.vasic.llmsverifier/pkg/strategy"
)

// AnalysisStrategy is a specialised strategy for selecting the best
// model for screenshot analysis tasks. It prioritises description
// quality, OCR accuracy, object detection, comprehensiveness, and cost
// efficiency.
//
// Models like Astica that specialise in rich image descriptions, OCR,
// and object detection score HIGHEST. General-purpose vision models
// (Gemini, GPT-4o) score lower because their descriptions are less
// detailed for analysis tasks.
type AnalysisStrategy struct {
	mu sync.RWMutex

	// descriptionQualityWeight weights rich image description ability.
	descriptionQualityWeight float64

	// ocrAccuracyWeight weights text recognition accuracy.
	ocrAccuracyWeight float64

	// objectDetectionWeight weights UI element and object
	// identification.
	objectDetectionWeight float64

	// comprehensivenessWeight weights breadth of visual analysis
	// capabilities.
	comprehensivenessWeight float64

	// costWeight weights cost efficiency.
	costWeight float64

	// scoreCache caches computed scores.
	scoreCache map[string]cachedAnalysisScore
}

type cachedAnalysisScore struct {
	score     strategy.StrategyScore
	expiresAt time.Time
}

// AnalysisStrategyConfig holds configuration for the Analysis
// strategy.
type AnalysisStrategyConfig struct {
	// DescriptionQualityWeight for image descriptions (default: 0.35)
	DescriptionQualityWeight float64

	// OCRAccuracyWeight for text recognition (default: 0.20)
	OCRAccuracyWeight float64

	// ObjectDetectionWeight for object identification (default: 0.20)
	ObjectDetectionWeight float64

	// ComprehensivenessWeight for breadth of capabilities
	// (default: 0.15)
	ComprehensivenessWeight float64

	// CostWeight for cost efficiency (default: 0.10)
	CostWeight float64
}

// DefaultAnalysisStrategyConfig returns the default Analysis strategy
// configuration.
func DefaultAnalysisStrategyConfig() *AnalysisStrategyConfig {
	return &AnalysisStrategyConfig{
		DescriptionQualityWeight: 0.35,
		OCRAccuracyWeight:        0.20,
		ObjectDetectionWeight:    0.20,
		ComprehensivenessWeight:  0.15,
		CostWeight:               0.10,
	}
}

// NewAnalysisStrategy creates a new analysis-optimised strategy.
func NewAnalysisStrategy(
	opts ...func(*AnalysisStrategyConfig),
) *AnalysisStrategy {
	cfg := DefaultAnalysisStrategyConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return &AnalysisStrategy{
		descriptionQualityWeight: cfg.DescriptionQualityWeight,
		ocrAccuracyWeight:        cfg.OCRAccuracyWeight,
		objectDetectionWeight:    cfg.ObjectDetectionWeight,
		comprehensivenessWeight:  cfg.ComprehensivenessWeight,
		costWeight:               cfg.CostWeight,
		scoreCache:               make(map[string]cachedAnalysisScore),
	}
}

// Name returns the strategy identifier.
func (s *AnalysisStrategy) Name() string {
	return "analysis"
}

// Description returns a human-readable description.
func (s *AnalysisStrategy) Description() string {
	return fmt.Sprintf(
		"Analysis strategy: description_quality(%.0f%%), ocr(%.0f%%), "+
			"object_detection(%.0f%%), comprehensiveness(%.0f%%), cost(%.0f%%)",
		s.descriptionQualityWeight*100,
		s.ocrAccuracyWeight*100,
		s.objectDetectionWeight*100,
		s.comprehensivenessWeight*100,
		s.costWeight*100,
	)
}

// SetWeights sets custom dimension weights. Recognised keys:
//
//	quality  -> descriptionQualityWeight
//	context  -> ocrAccuracyWeight
//	vision   -> objectDetectionWeight
//	capability -> comprehensivenessWeight
//	cost     -> costWeight
func (s *AnalysisStrategy) SetWeights(weights map[string]float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if w, ok := weights[strategy.DimensionQuality]; ok {
		s.descriptionQualityWeight = w
	}
	if w, ok := weights[strategy.DimensionContext]; ok {
		s.ocrAccuracyWeight = w
	}
	if w, ok := weights[strategy.DimensionVision]; ok {
		s.objectDetectionWeight = w
	}
	if w, ok := weights[strategy.DimensionCapability]; ok {
		s.comprehensivenessWeight = w
	}
	if w, ok := weights[strategy.DimensionCost]; ok {
		s.costWeight = w
	}
}

// SetConstraints is a no-op for analysis strategy.
func (s *AnalysisStrategy) SetConstraints(_ []strategy.Constraint) {}

// SetFallbacks is a no-op for analysis strategy.
func (s *AnalysisStrategy) SetFallbacks(_ []strategy.FallbackRule) {}

// Score evaluates a model with analysis-specific scoring dimensions.
//
// Dimensions:
//   - Description quality (35%): derived from QualityScore with
//     bonuses for vision-specific capabilities (ocr,
//     object_detection, face_detection, content_moderation).
//   - OCR accuracy (20%): ocr capability check.
//   - Object detection (20%): object_detection capability check.
//   - Comprehensiveness (15%): count of vision-related capabilities
//     normalised against a maximum of 6.
//   - Cost (10%): free models get 1.0; expensive models score lower.
//
// Models that do not support vision get an overall score of 0.
func (s *AnalysisStrategy) Score(
	ctx context.Context,
	model strategy.ModelInfo,
) (strategy.StrategyScore, error) {
	s.mu.RLock()
	cacheKey := model.ID
	if cached, ok := s.scoreCache[cacheKey]; ok {
		if time.Now().Before(cached.expiresAt) {
			s.mu.RUnlock()
			return cached.score, nil
		}
	}
	s.mu.RUnlock()

	scores := make(map[string]float64)

	// --- Description quality dimension ---
	descScore := 0.0
	if model.SupportsVision {
		descScore = model.QualityScore
		// Bonus for specialised vision capabilities that improve
		// description richness.
		visionCaps := countCapabilities(model.Capabilities,
			"vision", "ocr", "object_detection",
			"face_detection", "content_moderation",
		)
		if visionCaps >= 4 {
			descScore = math.Min(1.0, descScore+0.10)
		} else if visionCaps >= 3 {
			descScore = math.Min(1.0, descScore+0.05)
		}
	}
	scores[strategy.DimensionQuality] = descScore * s.descriptionQualityWeight

	// --- OCR accuracy dimension ---
	ocrScore := 0.0
	if model.SupportsVision {
		if hasCapability(model.Capabilities, "ocr") {
			ocrScore = 0.95
		} else {
			// General vision models have some OCR ability.
			ocrScore = 0.4 * model.QualityScore
		}
	}
	scores[strategy.DimensionContext] = ocrScore * s.ocrAccuracyWeight

	// --- Object detection dimension ---
	objScore := 0.0
	if model.SupportsVision {
		if hasCapability(model.Capabilities, "object_detection") {
			objScore = 0.95
		} else {
			// General vision models have some detection ability.
			objScore = 0.35 * model.QualityScore
		}
	}
	scores[strategy.DimensionVision] = objScore * s.objectDetectionWeight

	// --- Comprehensiveness dimension ---
	compScore := 0.0
	if model.SupportsVision {
		analysisCaps := countCapabilities(model.Capabilities,
			"vision", "ocr", "object_detection",
			"face_detection", "content_moderation",
			"gui_analysis",
		)
		compScore = float64(analysisCaps) / 6.0
		if compScore > 1.0 {
			compScore = 1.0
		}
	}
	scores[strategy.DimensionCapability] = compScore * s.comprehensivenessWeight

	// --- Cost dimension ---
	costScore := 0.5
	if model.InputCostPer1k == 0 && model.OutputCostPer1k == 0 {
		costScore = 1.0
	} else {
		totalCost := model.InputCostPer1k + model.OutputCostPer1k
		costScore = 1.0 - (totalCost / 0.04)
		if costScore < 0 {
			costScore = 0
		}
		if costScore > 1 {
			costScore = 1
		}
	}
	scores[strategy.DimensionCost] = costScore * s.costWeight

	// --- Overall ---
	var overall float64
	for _, v := range scores {
		overall += v
	}

	totalWeight := s.descriptionQualityWeight + s.ocrAccuracyWeight +
		s.objectDetectionWeight + s.comprehensivenessWeight + s.costWeight
	if totalWeight > 0 {
		overall = overall / totalWeight
	}

	// Non-vision models get zero.
	if !model.SupportsVision {
		overall = 0
	}

	// Confidence calculation.
	confidence := 0.75
	if model.Verified {
		confidence = 0.95
	}
	if !model.SupportsVision {
		confidence = 0.1
	}
	if time.Since(model.LastVerified) > 24*time.Hour {
		confidence *= 0.9
	}

	result := strategy.StrategyScore{
		Overall:         overall,
		DimensionScores: scores,
		Confidence:      confidence,
		Reasoning:       s.generateAnalysisReasoning(model, scores, descScore, ocrScore, objScore, compScore),
		Timestamp:       time.Now(),
		ModelID:         model.ID,
		StrategyName:    s.Name(),
	}

	s.mu.Lock()
	s.scoreCache[cacheKey] = cachedAnalysisScore{
		score:     result,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	s.mu.Unlock()

	return result, nil
}

func (s *AnalysisStrategy) generateAnalysisReasoning(
	model strategy.ModelInfo,
	scores map[string]float64,
	descScore, ocrScore, objScore, compScore float64,
) string {
	visionStatus := "no vision"
	if model.SupportsVision {
		visionStatus = "vision capable"
	}

	return fmt.Sprintf(
		"Analysis score %.2f: %s, description_quality %.2f, "+
			"ocr %.2f, object_detection %.2f, comprehensiveness %.2f, provider %s",
		sumScores(scores),
		visionStatus,
		descScore,
		ocrScore,
		objScore,
		compScore,
		model.Provider,
	)
}

// Validate checks if a model meets analysis requirements.
func (s *AnalysisStrategy) Validate(
	ctx context.Context,
	model strategy.ModelInfo,
) strategy.ValidationResult {
	result := strategy.ValidationResult{
		Valid:    true,
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
		Details:  make(map[string]any),
	}

	if !model.SupportsVision {
		result.Valid = false
		result.Errors = append(result.Errors,
			"vision capability is required for analysis strategy")
	}

	if model.QualityScore < 0.5 {
		result.Warnings = append(result.Warnings,
			"quality score below 0.5 may produce poor image analysis")
	}

	if !hasCapability(model.Capabilities, "ocr") {
		result.Warnings = append(result.Warnings,
			"model lacks OCR capability; text recognition may be limited")
	}

	score, _ := s.Score(ctx, model)
	result.Score = score.Overall
	result.Details["score"] = score
	result.Details["vision_capable"] = model.SupportsVision
	result.Details["has_ocr"] = hasCapability(model.Capabilities, "ocr")
	result.Details["has_object_detection"] = hasCapability(model.Capabilities, "object_detection")

	return result
}

// Rank sorts models by analysis-specific criteria. Vision-capable
// models with rich analysis capabilities rank highest.
func (s *AnalysisStrategy) Rank(
	ctx context.Context,
	models []strategy.ModelInfo,
) ([]strategy.RankedModel, error) {
	ranked := make([]strategy.RankedModel, 0, len(models))

	for _, model := range models {
		score, err := s.Score(ctx, model)
		if err != nil {
			continue
		}

		ranked = append(ranked, strategy.RankedModel{
			Model: model,
			Score: score,
			Tier:  s.determineAnalysisTier(score.Overall, model.SupportsVision),
		})
	}

	sortAnalysisModels(ranked)

	for i := range ranked {
		ranked[i].Rank = i + 1
	}

	totalScore := 0.0
	for _, r := range ranked {
		totalScore += r.Score.Overall
	}
	if totalScore > 0 {
		for i := range ranked {
			ranked[i].SelectionProbability = ranked[i].Score.Overall / totalScore
		}
	}

	return ranked, nil
}

func (s *AnalysisStrategy) determineAnalysisTier(
	score float64,
	hasVision bool,
) string {
	if !hasVision {
		return strategy.Tier3
	}
	if score >= 0.70 {
		return strategy.Tier1
	}
	if score >= 0.50 {
		return strategy.Tier2
	}
	return strategy.Tier3
}

// sortAnalysisModels puts vision-capable models first, then sorts by
// score descending within each group.
func sortAnalysisModels(ranked []strategy.RankedModel) {
	for i := 0; i < len(ranked)-1; i++ {
		for j := i + 1; j < len(ranked); j++ {
			iVision := ranked[i].Model.SupportsVision
			jVision := ranked[j].Model.SupportsVision

			if jVision && !iVision {
				ranked[i], ranked[j] = ranked[j], ranked[i]
				continue
			}
			if iVision == jVision &&
				ranked[j].Score.Overall > ranked[i].Score.Overall {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}
}

// Select chooses the best analysis model from the ranked list.
func (s *AnalysisStrategy) Select(
	ctx context.Context,
	ranked []strategy.RankedModel,
	req strategy.Requirements,
) (strategy.ModelInfo, error) {
	if len(ranked) == 0 {
		return strategy.ModelInfo{},
			fmt.Errorf("no models available for analysis selection")
	}

	// Enforce vision requirement by default.
	analysisReq := req
	analysisReq.NeedsVision = true

	filtered := s.filterByRequirements(ranked, analysisReq)
	if len(filtered) == 0 {
		return strategy.ModelInfo{},
			fmt.Errorf("no models meet analysis requirements")
	}

	// Honour preferred provider if set.
	if req.PreferredProvider != "" {
		for _, r := range filtered {
			if r.Model.Provider == req.PreferredProvider {
				return r.Model, nil
			}
		}
	}

	return filtered[0].Model, nil
}

func (s *AnalysisStrategy) filterByRequirements(
	ranked []strategy.RankedModel,
	req strategy.Requirements,
) []strategy.RankedModel {
	result := make([]strategy.RankedModel, 0)

	for _, r := range ranked {
		model := r.Model

		if req.NeedsVision && !model.SupportsVision {
			continue
		}
		if req.NeedsStreaming && !model.SupportsStreaming {
			continue
		}
		if req.NeedsFunctionCalling && !model.SupportsFunctionCalling {
			continue
		}
		if req.MinContextWindow > 0 &&
			model.ContextWindow < req.MinContextWindow {
			continue
		}
		if req.MaxLatencyMs > 0 &&
			model.AvgLatencyMs > req.MaxLatencyMs {
			continue
		}
		if req.MinQualityScore > 0 &&
			model.QualityScore < req.MinQualityScore {
			continue
		}
		if req.MinReliabilityScore > 0 &&
			model.ReliabilityScore < req.MinReliabilityScore {
			continue
		}
		if req.MaxInputCostPer1k > 0 &&
			model.InputCostPer1k > req.MaxInputCostPer1k {
			continue
		}
		if req.MaxOutputCostPer1k > 0 &&
			model.OutputCostPer1k > req.MaxOutputCostPer1k {
			continue
		}

		excluded := false
		for _, ep := range req.ExcludedProviders {
			if model.Provider == ep {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		hasAllCaps := true
		for _, cap := range req.RequiredCapabilities {
			found := false
			for _, mc := range model.Capabilities {
				if mc == cap {
					found = true
					break
				}
			}
			if !found {
				hasAllCaps = false
				break
			}
		}
		if !hasAllCaps {
			continue
		}

		result = append(result, r)
	}

	return result
}

// ClearCache clears the score cache.
func (s *AnalysisStrategy) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scoreCache = make(map[string]cachedAnalysisScore)
}
