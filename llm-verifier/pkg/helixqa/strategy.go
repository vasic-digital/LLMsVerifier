// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

// Package helixqa provides HelixQA-specific strategy implementations
// optimized for autonomous QA testing scenarios.
package helixqa

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"digital.vasic.llmsverifier/pkg/strategy"
)

// QAStrategy is a specialized strategy optimized for autonomous QA testing.
// It heavily weighs vision capabilities, response speed for interactive testing,
// and quality for accurate bug detection.
type QAStrategy struct {
	mu sync.RWMutex

	// baseStrategy is the underlying default strategy
	baseStrategy strategy.VerificationStrategy

	// visionWeight is the weight for vision capability
	visionWeight float64

	// speedWeight is the weight for response speed
	speedWeight float64

	// qualityWeight is the weight for response quality
	qualityWeight float64

	// costWeight is the weight for cost efficiency
	costWeight float64

	// reliabilityWeight is the weight for reliability
	reliabilityWeight float64

	// scoreCache for caching computed scores
	scoreCache map[string]cachedQAScore

	// testContextWeight allows adjusting weights based on test context
	testContextWeight map[string]float64
}

type cachedQAScore struct {
	score     strategy.StrategyScore
	expiresAt time.Time
}

// QAStrategyConfig holds configuration for the QA strategy
type QAStrategyConfig struct {
	// VisionWeight for vision capability (default: 0.25)
	VisionWeight float64

	// SpeedWeight for response speed (default: 0.25)
	SpeedWeight float64

	// QualityWeight for response quality (default: 0.30)
	QualityWeight float64

	// CostWeight for cost efficiency (default: 0.10)
	CostWeight float64

	// ReliabilityWeight for reliability (default: 0.10)
	ReliabilityWeight float64

	// CacheTTL for score caching (default: 5 minutes)
	CacheTTL time.Duration
}

// DefaultQAStrategyConfig returns the default QA strategy configuration
func DefaultQAStrategyConfig() *QAStrategyConfig {
	return &QAStrategyConfig{
		VisionWeight:      0.25,
		SpeedWeight:       0.25,
		QualityWeight:     0.30,
		CostWeight:        0.10,
		ReliabilityWeight: 0.10,
		CacheTTL:          5 * time.Minute,
	}
}

// NewQAStrategy creates a new QA-optimized strategy
func NewQAStrategy(opts ...func(*QAStrategyConfig)) *QAStrategy {
	cfg := DefaultQAStrategyConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return &QAStrategy{
		baseStrategy:      strategy.NewDefaultStrategy(),
		visionWeight:      cfg.VisionWeight,
		speedWeight:       cfg.SpeedWeight,
		qualityWeight:     cfg.QualityWeight,
		costWeight:        cfg.CostWeight,
		reliabilityWeight: cfg.ReliabilityWeight,
		scoreCache:        make(map[string]cachedQAScore),
		testContextWeight: make(map[string]float64),
	}
}

// Name returns the strategy name
func (s *QAStrategy) Name() string {
	return "qa"
}

// Description returns the strategy description
func (s *QAStrategy) Description() string {
	return fmt.Sprintf("QA-optimized strategy: vision(%.0f%%), speed(%.0f%%), quality(%.0f%%), cost(%.0f%%), reliability(%.0f%%)",
		s.visionWeight*100, s.speedWeight*100, s.qualityWeight*100,
		s.costWeight*100, s.reliabilityWeight*100)
}

// SetWeights sets custom dimension weights
func (s *QAStrategy) SetWeights(weights map[string]float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if w, ok := weights[strategy.DimensionVision]; ok {
		s.visionWeight = w
	}
	if w, ok := weights[strategy.DimensionSpeed]; ok {
		s.speedWeight = w
	}
	if w, ok := weights[strategy.DimensionQuality]; ok {
		s.qualityWeight = w
	}
	if w, ok := weights[strategy.DimensionCost]; ok {
		s.costWeight = w
	}
	if w, ok := weights[strategy.DimensionReliability]; ok {
		s.reliabilityWeight = w
	}
}

// SetConstraints sets verification constraints (delegates to base strategy)
func (s *QAStrategy) SetConstraints(constraints []strategy.Constraint) {
	if cs, ok := s.baseStrategy.(interface{ SetConstraints([]strategy.Constraint) }); ok {
		cs.SetConstraints(constraints)
	}
}

// SetFallbacks sets fallback rules (delegates to base strategy)
func (s *QAStrategy) SetFallbacks(fallbacks []strategy.FallbackRule) {
	if fs, ok := s.baseStrategy.(interface{ SetFallbacks([]strategy.FallbackRule) }); ok {
		fs.SetFallbacks(fallbacks)
	}
}

// SetTestContext adjusts weights based on test context
func (s *QAStrategy) SetTestContext(context string, weightAdjustment float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.testContextWeight[context] = weightAdjustment
}

// Score evaluates a model with QA-specific scoring
func (s *QAStrategy) Score(ctx context.Context, model strategy.ModelInfo) (strategy.StrategyScore, error) {
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

	visionScore := 0.0
	if model.SupportsVision {
		visionScore = 1.0
	}
	scores[strategy.DimensionVision] = visionScore * s.visionWeight

	speedScore := 0.5
	if model.AvgLatencyMs > 0 {
		latencyScore := 1.0 - float64(model.AvgLatencyMs)/5000.0
		if latencyScore < 0 {
			latencyScore = 0
		}
		speedScore = latencyScore
	}
	scores[strategy.DimensionSpeed] = speedScore * s.speedWeight

	qualityScore := model.QualityScore
	scores[strategy.DimensionQuality] = qualityScore * s.qualityWeight

	costScore := 0.5
	if model.InputCostPer1k > 0 || model.OutputCostPer1k > 0 {
		totalCost := model.InputCostPer1k + model.OutputCostPer1k
		costScore = 1.0 - (totalCost / 0.08)
		if costScore < 0 {
			costScore = 0
		}
		if costScore > 1 {
			costScore = 1
		}
	}
	scores[strategy.DimensionCost] = costScore * s.costWeight

	reliabilityScore := model.ReliabilityScore
	scores[strategy.DimensionReliability] = reliabilityScore * s.reliabilityWeight

	var overall float64
	for _, v := range scores {
		overall += v
	}

	totalWeight := s.visionWeight + s.speedWeight + s.qualityWeight + s.costWeight + s.reliabilityWeight
	if totalWeight > 0 {
		overall = overall / totalWeight
	}

	if model.SupportsVision {
		overall = math.Min(1.0, overall+0.1)
	}

	if model.AvgLatencyMs > 5000 {
		overall = math.Max(0.0, overall-0.1)
	}

	confidence := 0.75
	if model.Verified {
		confidence = 0.95
	}
	if !model.SupportsVision {
		confidence *= 0.8
	}
	if time.Since(model.LastVerified) > 24*time.Hour {
		confidence *= 0.9
	}

	result := strategy.StrategyScore{
		Overall:         overall,
		DimensionScores: scores,
		Confidence:      confidence,
		Reasoning:       s.generateQAReasoning(model, scores, visionScore),
		Timestamp:       time.Now(),
		ModelID:         model.ID,
		StrategyName:    s.Name(),
	}

	s.mu.Lock()
	s.scoreCache[cacheKey] = cachedQAScore{
		score:     result,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	s.mu.Unlock()

	return result, nil
}

func (s *QAStrategy) generateQAReasoning(model strategy.ModelInfo, scores map[string]float64, visionScore float64) string {
	visionStatus := "no vision"
	if model.SupportsVision {
		visionStatus = "vision capable"
	}

	latencyStatus := "unknown latency"
	if model.AvgLatencyMs > 0 {
		latencyStatus = fmt.Sprintf("%dms latency", model.AvgLatencyMs)
	}

	return fmt.Sprintf(
		"QA score %.2f: %s, %s, quality %.2f, reliability %.2f",
		sumQAScores(scores),
		visionStatus,
		latencyStatus,
		model.QualityScore,
		model.ReliabilityScore,
	)
}

func sumQAScores(scores map[string]float64) float64 {
	var sum float64
	for _, v := range scores {
		sum += v
	}
	return sum
}

// Validate checks if a model meets QA requirements
func (s *QAStrategy) Validate(ctx context.Context, model strategy.ModelInfo) strategy.ValidationResult {
	result := strategy.ValidationResult{
		Valid:    true,
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
		Details:  make(map[string]any),
	}

	if !model.SupportsVision {
		result.Warnings = append(result.Warnings,
			"vision capability recommended for autonomous QA testing")
	}

	if model.AvgLatencyMs > 5000 {
		result.Warnings = append(result.Warnings,
			"high latency may impact interactive testing performance")
	}

	if model.QualityScore < 0.6 {
		result.Valid = false
		result.Errors = append(result.Errors,
			"quality score below 0.6 may result in poor issue detection")
	}

	if model.ReliabilityScore < 0.8 {
		result.Warnings = append(result.Warnings,
			"reliability score below 0.8 may result in interrupted sessions")
	}

	score, _ := s.Score(ctx, model)
	result.Score = score.Overall
	result.Details["score"] = score
	result.Details["vision_capable"] = model.SupportsVision

	return result
}

// Rank sorts models by QA-specific criteria
func (s *QAStrategy) Rank(ctx context.Context, models []strategy.ModelInfo) ([]strategy.RankedModel, error) {
	ranked := make([]strategy.RankedModel, 0, len(models))

	for _, model := range models {
		score, err := s.Score(ctx, model)
		if err != nil {
			continue
		}

		ranked = append(ranked, strategy.RankedModel{
			Model: model,
			Score: score,
			Tier:  s.determineQATier(score.Overall, model.SupportsVision),
		})
	}

	sortQAModels(ranked)

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

func (s *QAStrategy) determineQATier(score float64, hasVision bool) string {
	if hasVision {
		if score >= 0.7 {
			return strategy.Tier1
		}
		if score >= 0.5 {
			return strategy.Tier2
		}
	} else {
		if score >= 0.85 {
			return strategy.Tier1
		}
		if score >= 0.65 {
			return strategy.Tier2
		}
	}
	return strategy.Tier3
}

func sortQAModels(ranked []strategy.RankedModel) {
	for i := 0; i < len(ranked)-1; i++ {
		for j := i + 1; j < len(ranked); j++ {
			iScore := ranked[i].Score.Overall
			jScore := ranked[j].Score.Overall
			iVision := ranked[i].Model.SupportsVision
			jVision := ranked[j].Model.SupportsVision

			if jVision && !iVision {
				ranked[i], ranked[j] = ranked[j], ranked[i]
				continue
			}

			if (iVision == jVision) && jScore > iScore {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}
}

// Select chooses the best model for QA testing
func (s *QAStrategy) Select(ctx context.Context, ranked []strategy.RankedModel, req strategy.Requirements) (strategy.ModelInfo, error) {
	if len(ranked) == 0 {
		return strategy.ModelInfo{}, fmt.Errorf("no models available for QA testing")
	}

	qaReq := s.enhanceRequirements(req)
	filtered := s.filterByQARequirements(ranked, qaReq)

	if len(filtered) == 0 {
		return strategy.ModelInfo{}, fmt.Errorf("no models meet QA requirements (vision: %v)", req.NeedsVision)
	}

	if req.PreferredProvider != "" {
		for _, r := range filtered {
			if r.Model.Provider == req.PreferredProvider {
				return r.Model, nil
			}
		}
	}

	for _, r := range filtered {
		if r.Model.SupportsVision {
			return r.Model, nil
		}
	}

	return filtered[0].Model, nil
}

func (s *QAStrategy) enhanceRequirements(req strategy.Requirements) strategy.Requirements {
	if req.MinQualityScore == 0 {
		req.MinQualityScore = 0.6
	}
	if req.MinReliabilityScore == 0 {
		req.MinReliabilityScore = 0.8
	}
	if req.MaxLatencyMs == 0 {
		req.MaxLatencyMs = 5000
	}
	return req
}

func (s *QAStrategy) filterByQARequirements(ranked []strategy.RankedModel, req strategy.Requirements) []strategy.RankedModel {
	result := make([]strategy.RankedModel, 0)

	for _, r := range ranked {
		model := r.Model

		if req.NeedsVision && !model.SupportsVision {
			continue
		}
		if req.NeedsStreaming && !model.SupportsStreaming {
			continue
		}
		if req.MinQualityScore > 0 && model.QualityScore < req.MinQualityScore {
			continue
		}
		if req.MinReliabilityScore > 0 && model.ReliabilityScore < req.MinReliabilityScore {
			continue
		}
		if req.MaxLatencyMs > 0 && model.AvgLatencyMs > req.MaxLatencyMs {
			continue
		}
		if req.MinContextWindow > 0 && model.ContextWindow < req.MinContextWindow {
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

		result = append(result, r)
	}

	return result
}

// ClearCache clears the score cache
func (s *QAStrategy) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scoreCache = make(map[string]cachedQAScore)
}
