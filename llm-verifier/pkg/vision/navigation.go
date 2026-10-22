// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package vision

import (
	"context"
	"fmt"
	"sync"
	"time"

	"digital.vasic.llmsverifier/pkg/strategy"
)

// NavigationStrategy is a specialised strategy for selecting the best
// model for UI navigation tasks. It prioritises JSON compliance (the
// ability to produce structured JSON action arrays from screenshots),
// GUI understanding, speed, and cost efficiency.
//
// Models that return descriptions instead of JSON (like Astica) score
// LOW because the navigation pipeline requires machine-parseable JSON
// action arrays, not natural-language captions.
type NavigationStrategy struct {
	mu sync.RWMutex

	// jsonComplianceWeight weights structured JSON output ability.
	jsonComplianceWeight float64

	// guiUnderstandWeight weights GUI element detection and focus
	// tracking ability.
	guiUnderstandWeight float64

	// speedWeight weights response speed (interactive navigation
	// needs sub-5s responses).
	speedWeight float64

	// costWeight weights cost efficiency for high-volume navigation
	// calls.
	costWeight float64

	// scoreCache caches computed scores.
	scoreCache map[string]cachedNavigationScore
}

type cachedNavigationScore struct {
	score     strategy.StrategyScore
	expiresAt time.Time
}

// NavigationStrategyConfig holds configuration for the Navigation
// strategy.
type NavigationStrategyConfig struct {
	// JSONComplianceWeight for structured JSON output (default: 0.40)
	JSONComplianceWeight float64

	// GUIUnderstandWeight for GUI element detection (default: 0.25)
	GUIUnderstandWeight float64

	// SpeedWeight for response speed (default: 0.20)
	SpeedWeight float64

	// CostWeight for cost efficiency (default: 0.15)
	CostWeight float64
}

// DefaultNavigationStrategyConfig returns the default Navigation
// strategy configuration.
func DefaultNavigationStrategyConfig() *NavigationStrategyConfig {
	return &NavigationStrategyConfig{
		JSONComplianceWeight: 0.40,
		GUIUnderstandWeight:  0.25,
		SpeedWeight:          0.20,
		CostWeight:           0.15,
	}
}

// NewNavigationStrategy creates a new navigation-optimised strategy.
func NewNavigationStrategy(
	opts ...func(*NavigationStrategyConfig),
) *NavigationStrategy {
	cfg := DefaultNavigationStrategyConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return &NavigationStrategy{
		jsonComplianceWeight: cfg.JSONComplianceWeight,
		guiUnderstandWeight:  cfg.GUIUnderstandWeight,
		speedWeight:          cfg.SpeedWeight,
		costWeight:           cfg.CostWeight,
		scoreCache:           make(map[string]cachedNavigationScore),
	}
}

// Name returns the strategy identifier.
func (s *NavigationStrategy) Name() string {
	return "navigation"
}

// Description returns a human-readable description.
func (s *NavigationStrategy) Description() string {
	return fmt.Sprintf(
		"Navigation strategy: json_compliance(%.0f%%), gui_understand(%.0f%%), "+
			"speed(%.0f%%), cost(%.0f%%)",
		s.jsonComplianceWeight*100,
		s.guiUnderstandWeight*100,
		s.speedWeight*100,
		s.costWeight*100,
	)
}

// SetWeights sets custom dimension weights. Recognised keys:
//
//	quality  -> jsonComplianceWeight
//	vision   -> guiUnderstandWeight
//	speed    -> speedWeight
//	cost     -> costWeight
func (s *NavigationStrategy) SetWeights(weights map[string]float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if w, ok := weights[strategy.DimensionQuality]; ok {
		s.jsonComplianceWeight = w
	}
	if w, ok := weights[strategy.DimensionVision]; ok {
		s.guiUnderstandWeight = w
	}
	if w, ok := weights[strategy.DimensionSpeed]; ok {
		s.speedWeight = w
	}
	if w, ok := weights[strategy.DimensionCost]; ok {
		s.costWeight = w
	}
}

// SetConstraints is a no-op for navigation strategy.
func (s *NavigationStrategy) SetConstraints(_ []strategy.Constraint) {}

// SetFallbacks is a no-op for navigation strategy.
func (s *NavigationStrategy) SetFallbacks(_ []strategy.FallbackRule) {}

// Score evaluates a model with navigation-specific scoring dimensions.
//
// Dimensions:
//   - JSON compliance (40%): models with json_output or
//     structured_output capability get 1.0. Models without get 0.2
//     (Astica returns descriptions, not JSON).
//   - GUI understanding (25%): gui_analysis, gui_grounding,
//     gui_navigation capabilities.
//   - Speed (20%): normalised against 5s ideal (lower latency =
//     higher score).
//   - Cost (15%): free models get 1.0, scale by cost.
//
// Models that do not support vision get an overall score of 0.
func (s *NavigationStrategy) Score(
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

	// --- JSON compliance dimension ---
	jsonScore := 0.0
	if model.SupportsVision {
		if hasCapability(model.Capabilities, "json_output") ||
			hasCapability(model.Capabilities, "structured_output") {
			jsonScore = 1.0
		} else {
			// Models without structured output produce descriptions,
			// not JSON action arrays. Heavily penalised.
			jsonScore = 0.2
		}
	}
	scores[strategy.DimensionQuality] = jsonScore * s.jsonComplianceWeight

	// --- GUI understanding dimension ---
	guiScore := 0.0
	if model.SupportsVision {
		guiCaps := countCapabilities(model.Capabilities,
			"gui_analysis", "gui_grounding",
			"gui_navigation", "element_detection",
		)
		if guiCaps > 0 {
			guiScore = float64(guiCaps) / 4.0
		} else {
			// General vision models can do basic GUI analysis at
			// reduced quality.
			guiScore = 0.3 * model.QualityScore
		}
	}
	scores[strategy.DimensionVision] = guiScore * s.guiUnderstandWeight

	// --- Speed dimension ---
	speedScore := 0.5
	if model.AvgLatencyMs > 0 {
		// Normalise against 5s ideal. Sub-1s models get close to 1.0.
		speedScore = 1.0 - float64(model.AvgLatencyMs)/5000.0
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

	totalWeight := s.jsonComplianceWeight + s.guiUnderstandWeight +
		s.speedWeight + s.costWeight
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
		Reasoning:       s.generateReasoning(model, scores, jsonScore, guiScore),
		Timestamp:       time.Now(),
		ModelID:         model.ID,
		StrategyName:    s.Name(),
	}

	s.mu.Lock()
	s.scoreCache[cacheKey] = cachedNavigationScore{
		score:     result,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	s.mu.Unlock()

	return result, nil
}

func (s *NavigationStrategy) generateReasoning(
	model strategy.ModelInfo,
	scores map[string]float64,
	jsonScore, guiScore float64,
) string {
	visionStatus := "no vision"
	if model.SupportsVision {
		visionStatus = "vision capable"
	}

	jsonStatus := "no JSON output"
	if hasCapability(model.Capabilities, "json_output") ||
		hasCapability(model.Capabilities, "structured_output") {
		jsonStatus = "JSON output capable"
	}

	return fmt.Sprintf(
		"Navigation score %.2f: %s, %s, json_compliance %.2f, "+
			"gui_understand %.2f, provider %s",
		sumScores(scores),
		visionStatus,
		jsonStatus,
		jsonScore,
		guiScore,
		model.Provider,
	)
}

// Validate checks if a model meets navigation requirements.
func (s *NavigationStrategy) Validate(
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
			"vision capability is required for navigation strategy")
	}

	if !hasCapability(model.Capabilities, "json_output") &&
		!hasCapability(model.Capabilities, "structured_output") {
		result.Warnings = append(result.Warnings,
			"model lacks json_output/structured_output capability; "+
				"navigation requires JSON action arrays")
	}

	if model.AvgLatencyMs > 5000 {
		result.Warnings = append(result.Warnings,
			"high latency may impact interactive navigation workflows")
	}

	score, _ := s.Score(ctx, model)
	result.Score = score.Overall
	result.Details["score"] = score
	result.Details["vision_capable"] = model.SupportsVision
	result.Details["json_capable"] = hasCapability(model.Capabilities, "json_output") ||
		hasCapability(model.Capabilities, "structured_output")

	return result
}

// Rank sorts models by navigation-specific criteria. JSON-capable
// vision models always rank above description-only models.
func (s *NavigationStrategy) Rank(
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
			Tier:  s.determineNavigationTier(score.Overall, model.SupportsVision),
		})
	}

	sortNavigationModels(ranked)

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

func (s *NavigationStrategy) determineNavigationTier(
	score float64,
	hasVision bool,
) string {
	if !hasVision {
		return strategy.Tier3
	}
	if score >= 0.70 {
		return strategy.Tier1
	}
	if score >= 0.45 {
		return strategy.Tier2
	}
	return strategy.Tier3
}

// sortNavigationModels puts vision-capable models first, then sorts by
// score descending within each group.
func sortNavigationModels(ranked []strategy.RankedModel) {
	for i := 0; i < len(ranked)-1; i++ {
		for j := i + 1; j < len(ranked); j++ {
			iVision := ranked[i].Model.SupportsVision
			jVision := ranked[j].Model.SupportsVision

			// Vision-capable always before non-vision.
			if jVision && !iVision {
				ranked[i], ranked[j] = ranked[j], ranked[i]
				continue
			}
			// Within the same group, higher score first.
			if iVision == jVision &&
				ranked[j].Score.Overall > ranked[i].Score.Overall {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}
}

// Select chooses the best navigation model from the ranked list.
func (s *NavigationStrategy) Select(
	ctx context.Context,
	ranked []strategy.RankedModel,
	req strategy.Requirements,
) (strategy.ModelInfo, error) {
	if len(ranked) == 0 {
		return strategy.ModelInfo{},
			fmt.Errorf("no models available for navigation selection")
	}

	// Enforce vision requirement by default.
	navReq := req
	navReq.NeedsVision = true

	filtered := s.filterByRequirements(ranked, navReq)
	if len(filtered) == 0 {
		return strategy.ModelInfo{},
			fmt.Errorf("no models meet navigation requirements")
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

func (s *NavigationStrategy) filterByRequirements(
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
func (s *NavigationStrategy) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scoreCache = make(map[string]cachedNavigationScore)
}
