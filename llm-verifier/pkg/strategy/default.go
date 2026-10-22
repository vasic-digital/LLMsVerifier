// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package strategy

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// DefaultStrategy is the default verification strategy that balances
// quality, speed, and cost with configurable weights.
type DefaultStrategy struct {
	mu sync.RWMutex

	// weights for different dimensions
	weights map[string]float64

	// constraints to apply during verification
	constraints []Constraint

	// fallback rules for degraded operation
	fallbacks []FallbackRule

	// cache for scores
	scoreCache map[string]cachedScore
}

type cachedScore struct {
	score     StrategyScore
	expiresAt time.Time
}

// NewDefaultStrategy creates a new default strategy with balanced weights
func NewDefaultStrategy(opts ...StrategyOption) *DefaultStrategy {
	s := &DefaultStrategy{
		weights: map[string]float64{
			DimensionQuality:     0.35,
			DimensionSpeed:       0.25,
			DimensionCost:        0.20,
			DimensionReliability: 0.20,
		},
		constraints: make([]Constraint, 0),
		fallbacks:   make([]FallbackRule, 0),
		scoreCache:  make(map[string]cachedScore),
	}

	for _, opt := range opts {
		_ = opt(s)
	}

	return s
}

// Name returns the strategy name
func (s *DefaultStrategy) Name() string {
	return "default"
}

// Description returns the strategy description
func (s *DefaultStrategy) Description() string {
	return "Default strategy balancing quality (35%), speed (25%), cost (20%), and reliability (20%)"
}

// SetWeights sets the dimension weights
func (s *DefaultStrategy) SetWeights(weights map[string]float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for k, v := range weights {
		s.weights[k] = v
	}
}

// SetConstraints sets the verification constraints
func (s *DefaultStrategy) SetConstraints(constraints []Constraint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.constraints = constraints
}

// SetFallbacks sets the fallback rules
func (s *DefaultStrategy) SetFallbacks(fallbacks []FallbackRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fallbacks = fallbacks
}

// Score evaluates a model and returns a detailed score
func (s *DefaultStrategy) Score(ctx context.Context, model ModelInfo) (StrategyScore, error) {
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

	qualityWeight := s.weights[DimensionQuality]
	speedWeight := s.weights[DimensionSpeed]
	costWeight := s.weights[DimensionCost]
	reliabilityWeight := s.weights[DimensionReliability]

	scores[DimensionQuality] = model.QualityScore * qualityWeight

	if model.AvgLatencyMs > 0 {
		latencyScore := 1.0 - float64(model.AvgLatencyMs)/10000.0
		if latencyScore < 0 {
			latencyScore = 0
		}
		scores[DimensionSpeed] = latencyScore * speedWeight
	} else {
		scores[DimensionSpeed] = 0.5 * speedWeight
	}

	if model.InputCostPer1k > 0 || model.OutputCostPer1k > 0 {
		totalCost := model.InputCostPer1k + model.OutputCostPer1k
		costScore := 1.0 - (totalCost / 0.1)
		if costScore < 0 {
			costScore = 0
		}
		if costScore > 1 {
			costScore = 1
		}
		scores[DimensionCost] = costScore * costWeight
	} else {
		scores[DimensionCost] = 0.5 * costWeight
	}

	scores[DimensionReliability] = model.ReliabilityScore * reliabilityWeight

	var overall float64
	for _, v := range scores {
		overall += v
	}

	totalWeight := qualityWeight + speedWeight + costWeight + reliabilityWeight
	if totalWeight > 0 {
		overall = overall / totalWeight
	}

	confidence := 0.8
	if model.Verified {
		confidence = 0.95
	}
	if time.Since(model.LastVerified) > 24*time.Hour {
		confidence *= 0.9
	}

	result := StrategyScore{
		Overall:         overall,
		DimensionScores: scores,
		Confidence:      confidence,
		Reasoning:       s.generateReasoning(model, scores),
		Timestamp:       time.Now(),
		ModelID:         model.ID,
		StrategyName:    s.Name(),
	}

	s.mu.Lock()
	s.scoreCache[cacheKey] = cachedScore{
		score:     result,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	s.mu.Unlock()

	return result, nil
}

func (s *DefaultStrategy) generateReasoning(model ModelInfo, scores map[string]float64) string {
	return fmt.Sprintf(
		"Model %s scored %.2f (quality: %.2f, speed: %.2f, cost: %.2f, reliability: %.2f)",
		model.Name,
		sumScores(scores),
		scores[DimensionQuality]/s.weights[DimensionQuality],
		scores[DimensionSpeed]/s.weights[DimensionSpeed],
		scores[DimensionCost]/s.weights[DimensionCost],
		scores[DimensionReliability]/s.weights[DimensionReliability],
	)
}

func sumScores(scores map[string]float64) float64 {
	var sum float64
	for _, v := range scores {
		sum += v
	}
	return sum
}

// Validate checks if a model meets minimum requirements
func (s *DefaultStrategy) Validate(ctx context.Context, model ModelInfo) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
		Details:  make(map[string]any),
	}

	// Copy constraints under read lock, then release before
	// calling Score (which acquires a write lock for caching).
	s.mu.RLock()
	constraints := make([]Constraint, len(s.constraints))
	copy(constraints, s.constraints)
	s.mu.RUnlock()

	for _, c := range constraints {
		if !c.Required {
			continue
		}

		if !s.checkConstraint(model, c) {
			result.Valid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("constraint %s failed: %s", c.Name, c.Description))
		}
	}

	if model.QualityScore < 0.5 {
		result.Warnings = append(result.Warnings,
			"quality score below 0.5 may result in poor performance")
	}

	if model.ReliabilityScore < 0.8 {
		result.Warnings = append(result.Warnings,
			"reliability score below 0.8 may result in intermittent failures")
	}

	score, _ := s.Score(ctx, model)
	result.Score = score.Overall
	result.Details["score"] = score

	return result
}

func (s *DefaultStrategy) checkConstraint(model ModelInfo, c Constraint) bool {
	switch c.Name {
	case "min_quality":
		if minQ, ok := c.Value.(float64); ok {
			return model.QualityScore >= minQ
		}
	case "min_reliability":
		if minR, ok := c.Value.(float64); ok {
			return model.ReliabilityScore >= minR
		}
	case "max_latency_ms":
		if maxL, ok := c.Value.(int); ok {
			return model.AvgLatencyMs <= maxL
		}
	case "requires_vision":
		if req, ok := c.Value.(bool); ok && req {
			return model.SupportsVision
		}
	case "requires_streaming":
		if req, ok := c.Value.(bool); ok && req {
			return model.SupportsStreaming
		}
	case "min_context":
		if minC, ok := c.Value.(int); ok {
			return model.ContextWindow >= minC
		}
	}
	return true
}

// Rank sorts models by strategy-specific criteria
func (s *DefaultStrategy) Rank(ctx context.Context, models []ModelInfo) ([]RankedModel, error) {
	ranked := make([]RankedModel, 0, len(models))

	for _, model := range models {
		score, err := s.Score(ctx, model)
		if err != nil {
			continue
		}

		ranked = append(ranked, RankedModel{
			Model: model,
			Score: score,
			Tier:  s.determineTier(score.Overall),
		})
	}

	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].Score.Overall > ranked[j].Score.Overall
	})

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

func (s *DefaultStrategy) determineTier(score float64) string {
	if score >= 0.8 {
		return Tier1
	}
	if score >= 0.6 {
		return Tier2
	}
	return Tier3
}

// Select chooses the best model from the ranked list
func (s *DefaultStrategy) Select(ctx context.Context, ranked []RankedModel, req Requirements) (ModelInfo, error) {
	if len(ranked) == 0 {
		return ModelInfo{}, fmt.Errorf("no models available for selection")
	}

	filtered := s.filterByRequirements(ranked, req)

	if len(filtered) == 0 {
		return ModelInfo{}, fmt.Errorf("no models meet requirements")
	}

	if req.PreferredProvider != "" {
		for _, r := range filtered {
			if r.Model.Provider == req.PreferredProvider {
				return r.Model, nil
			}
		}
	}

	return filtered[0].Model, nil
}

func (s *DefaultStrategy) filterByRequirements(ranked []RankedModel, req Requirements) []RankedModel {
	result := make([]RankedModel, 0)

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
		if req.MinContextWindow > 0 && model.ContextWindow < req.MinContextWindow {
			continue
		}
		if req.MaxLatencyMs > 0 && model.AvgLatencyMs > req.MaxLatencyMs {
			continue
		}
		if req.MinQualityScore > 0 && model.QualityScore < req.MinQualityScore {
			continue
		}
		if req.MinReliabilityScore > 0 && model.ReliabilityScore < req.MinReliabilityScore {
			continue
		}
		if req.MaxInputCostPer1k > 0 && model.InputCostPer1k > req.MaxInputCostPer1k {
			continue
		}
		if req.MaxOutputCostPer1k > 0 && model.OutputCostPer1k > req.MaxOutputCostPer1k {
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
