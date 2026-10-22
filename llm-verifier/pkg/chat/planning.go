// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

// Package chat provides strategies for selecting the best chat/reasoning
// model. Unlike the vision package which targets image understanding,
// chat strategies optimise for text-based reasoning, planning, and
// structured output generation.
package chat

import (
	"context"
	"fmt"
	"sync"
	"time"

	"digital.vasic.llmsverifier/pkg/strategy"
)

// PlanningStrategy is a specialised strategy for selecting the best
// model for test plan generation. It prioritises reasoning quality,
// context window size (for large knowledge bases), structured output
// ability, speed, and cost efficiency.
//
// Claude and GPT-4 score highest due to strong reasoning and large
// context windows. Gemini scores close second (1M context). Local
// Ollama models score lower (small context, weaker reasoning).
type PlanningStrategy struct {
	mu sync.RWMutex

	// reasoningWeight weights multi-step reasoning ability.
	reasoningWeight float64

	// contextWeight weights context window size.
	contextWeight float64

	// structuredOutputWeight weights ability to produce organised
	// test cases.
	structuredOutputWeight float64

	// speedWeight weights response speed.
	speedWeight float64

	// costWeight weights cost efficiency.
	costWeight float64

	// scoreCache caches computed scores.
	scoreCache map[string]cachedPlanningScore
}

type cachedPlanningScore struct {
	score     strategy.StrategyScore
	expiresAt time.Time
}

// PlanningStrategyConfig holds configuration for the Planning
// strategy.
type PlanningStrategyConfig struct {
	// ReasoningWeight for multi-step reasoning (default: 0.35)
	ReasoningWeight float64

	// ContextWeight for context window size (default: 0.25)
	ContextWeight float64

	// StructuredOutputWeight for organised output (default: 0.20)
	StructuredOutputWeight float64

	// SpeedWeight for response speed (default: 0.10)
	SpeedWeight float64

	// CostWeight for cost efficiency (default: 0.10)
	CostWeight float64
}

// DefaultPlanningStrategyConfig returns the default Planning strategy
// configuration.
func DefaultPlanningStrategyConfig() *PlanningStrategyConfig {
	return &PlanningStrategyConfig{
		ReasoningWeight:        0.35,
		ContextWeight:          0.25,
		StructuredOutputWeight: 0.20,
		SpeedWeight:            0.10,
		CostWeight:             0.10,
	}
}

// NewPlanningStrategy creates a new planning-optimised strategy.
func NewPlanningStrategy(
	opts ...func(*PlanningStrategyConfig),
) *PlanningStrategy {
	cfg := DefaultPlanningStrategyConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return &PlanningStrategy{
		reasoningWeight:        cfg.ReasoningWeight,
		contextWeight:          cfg.ContextWeight,
		structuredOutputWeight: cfg.StructuredOutputWeight,
		speedWeight:            cfg.SpeedWeight,
		costWeight:             cfg.CostWeight,
		scoreCache:             make(map[string]cachedPlanningScore),
	}
}

// Name returns the strategy identifier.
func (s *PlanningStrategy) Name() string {
	return "planning"
}

// Description returns a human-readable description.
func (s *PlanningStrategy) Description() string {
	return fmt.Sprintf(
		"Planning strategy: reasoning(%.0f%%), context(%.0f%%), "+
			"structured_output(%.0f%%), speed(%.0f%%), cost(%.0f%%)",
		s.reasoningWeight*100,
		s.contextWeight*100,
		s.structuredOutputWeight*100,
		s.speedWeight*100,
		s.costWeight*100,
	)
}

// SetWeights sets custom dimension weights. Recognised keys:
//
//	quality    -> reasoningWeight
//	context    -> contextWeight
//	capability -> structuredOutputWeight
//	speed      -> speedWeight
//	cost       -> costWeight
func (s *PlanningStrategy) SetWeights(weights map[string]float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if w, ok := weights[strategy.DimensionQuality]; ok {
		s.reasoningWeight = w
	}
	if w, ok := weights[strategy.DimensionContext]; ok {
		s.contextWeight = w
	}
	if w, ok := weights[strategy.DimensionCapability]; ok {
		s.structuredOutputWeight = w
	}
	if w, ok := weights[strategy.DimensionSpeed]; ok {
		s.speedWeight = w
	}
	if w, ok := weights[strategy.DimensionCost]; ok {
		s.costWeight = w
	}
}

// SetConstraints is a no-op for planning strategy.
func (s *PlanningStrategy) SetConstraints(_ []strategy.Constraint) {}

// SetFallbacks is a no-op for planning strategy.
func (s *PlanningStrategy) SetFallbacks(_ []strategy.FallbackRule) {}

// Score evaluates a model with planning-specific scoring dimensions.
//
// Dimensions:
//   - Reasoning quality (35%): reasoning capability + high
//     QualityScore. Models with the "reasoning" capability get a
//     bonus.
//   - Context window (25%): normalised against 1M tokens (larger =
//     better for knowledge base + test plans).
//   - Structured output (20%): json_output, structured_output, or
//     code capability.
//   - Speed (10%): normalised against 10s (planning is one-time per
//     session, so latency is less critical).
//   - Cost (10%): free models get 1.0, scale by cost.
func (s *PlanningStrategy) Score(
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

	// --- Reasoning quality dimension ---
	reasoningScore := model.QualityScore
	if hasCapability(model.Capabilities, "reasoning") {
		reasoningScore += 0.10
		if reasoningScore > 1.0 {
			reasoningScore = 1.0
		}
	}
	scores[strategy.DimensionQuality] = reasoningScore * s.reasoningWeight

	// --- Context window dimension ---
	contextScore := 0.0
	if model.ContextWindow > 0 {
		// Normalise against 1M tokens.
		contextScore = float64(model.ContextWindow) / 1_000_000.0
		if contextScore > 1.0 {
			contextScore = 1.0
		}
	}
	scores[strategy.DimensionContext] = contextScore * s.contextWeight

	// --- Structured output dimension ---
	structuredScore := 0.3 // baseline
	if hasCapability(model.Capabilities, "json_output") ||
		hasCapability(model.Capabilities, "structured_output") {
		structuredScore = 0.9
	} else if hasCapability(model.Capabilities, "code") {
		// Code-capable models can produce structured output
		// reasonably well.
		structuredScore = 0.7
	}
	scores[strategy.DimensionCapability] = structuredScore * s.structuredOutputWeight

	// --- Speed dimension ---
	speedScore := 0.5
	if model.AvgLatencyMs > 0 {
		// Normalise against 10s (planning tolerates higher latency).
		speedScore = 1.0 - float64(model.AvgLatencyMs)/10000.0
		if speedScore < 0 {
			speedScore = 0
		}
		if speedScore > 1 {
			speedScore = 1
		}
	}
	scores[strategy.DimensionSpeed] = speedScore * s.speedWeight

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

	totalWeight := s.reasoningWeight + s.contextWeight +
		s.structuredOutputWeight + s.speedWeight + s.costWeight
	if totalWeight > 0 {
		overall = overall / totalWeight
	}

	// Confidence calculation.
	confidence := 0.75
	if model.Verified {
		confidence = 0.95
	}
	if time.Since(model.LastVerified) > 24*time.Hour {
		confidence *= 0.9
	}

	result := strategy.StrategyScore{
		Overall:         overall,
		DimensionScores: scores,
		Confidence:      confidence,
		Reasoning:       s.generatePlanningReasoning(model, scores, reasoningScore, contextScore, structuredScore),
		Timestamp:       time.Now(),
		ModelID:         model.ID,
		StrategyName:    s.Name(),
	}

	s.mu.Lock()
	s.scoreCache[cacheKey] = cachedPlanningScore{
		score:     result,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	s.mu.Unlock()

	return result, nil
}

func (s *PlanningStrategy) generatePlanningReasoning(
	model strategy.ModelInfo,
	scores map[string]float64,
	reasoningScore, contextScore, structuredScore float64,
) string {
	reasoningStatus := "no reasoning capability"
	if hasCapability(model.Capabilities, "reasoning") {
		reasoningStatus = "reasoning capable"
	}

	return fmt.Sprintf(
		"Planning score %.2f: %s, reasoning %.2f, "+
			"context_window %.2f, structured_output %.2f, provider %s",
		sumScores(scores),
		reasoningStatus,
		reasoningScore,
		contextScore,
		structuredScore,
		model.Provider,
	)
}

// Validate checks if a model meets planning requirements.
func (s *PlanningStrategy) Validate(
	ctx context.Context,
	model strategy.ModelInfo,
) strategy.ValidationResult {
	result := strategy.ValidationResult{
		Valid:    true,
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
		Details:  make(map[string]any),
	}

	if model.QualityScore < 0.5 {
		result.Warnings = append(result.Warnings,
			"quality score below 0.5 may produce poor test plans")
	}

	if model.ContextWindow < 16000 {
		result.Warnings = append(result.Warnings,
			"context window below 16K tokens may be insufficient for test planning")
	}

	if !hasCapability(model.Capabilities, "reasoning") {
		result.Warnings = append(result.Warnings,
			"model lacks reasoning capability; complex test plans may be lower quality")
	}

	score, _ := s.Score(ctx, model)
	result.Score = score.Overall
	result.Details["score"] = score
	result.Details["has_reasoning"] = hasCapability(model.Capabilities, "reasoning")
	result.Details["context_window"] = model.ContextWindow

	return result
}

// Rank sorts models by planning-specific criteria.
func (s *PlanningStrategy) Rank(
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
			Tier:  s.determinePlanningTier(score.Overall),
		})
	}

	sortPlanningModels(ranked)

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

func (s *PlanningStrategy) determinePlanningTier(score float64) string {
	if score >= 0.70 {
		return strategy.Tier1
	}
	if score >= 0.50 {
		return strategy.Tier2
	}
	return strategy.Tier3
}

// sortPlanningModels sorts by score descending.
func sortPlanningModels(ranked []strategy.RankedModel) {
	for i := 0; i < len(ranked)-1; i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].Score.Overall > ranked[i].Score.Overall {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}
}

// Select chooses the best planning model from the ranked list.
func (s *PlanningStrategy) Select(
	ctx context.Context,
	ranked []strategy.RankedModel,
	req strategy.Requirements,
) (strategy.ModelInfo, error) {
	if len(ranked) == 0 {
		return strategy.ModelInfo{},
			fmt.Errorf("no models available for planning selection")
	}

	filtered := s.filterByRequirements(ranked, req)
	if len(filtered) == 0 {
		return strategy.ModelInfo{},
			fmt.Errorf("no models meet planning requirements")
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

func (s *PlanningStrategy) filterByRequirements(
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
func (s *PlanningStrategy) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scoreCache = make(map[string]cachedPlanningScore)
}

// --- Capability helpers (local to chat package) ---

func hasCapability(caps []string, target string) bool {
	for _, c := range caps {
		if c == target {
			return true
		}
	}
	return false
}

func sumScores(scores map[string]float64) float64 {
	var sum float64
	for _, v := range scores {
		sum += v
	}
	return sum
}
