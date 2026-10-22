// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

// Package vision provides a specialized strategy for selecting the
// best vision model. It prioritizes image understanding quality, OCR
// accuracy, GUI element detection, and screen analysis capability.
// This strategy is NOT the default -- it is used when vision model
// selection is specifically needed (e.g., HelixQA Execute/Curiosity
// phases, screenshot analysis, UI element detection).
//
// This strategy respects ProviderUsageConfig and will prefer providers
// designated as UsageTypeVisionOnly (e.g., Gemini) or UsageTypeAny
// with vision capabilities. Providers marked as UsageTypeLLMOnly are
// excluded from vision selection.
package vision

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"digital.vasic.llmsverifier/pkg/strategy"
)

// VisionStrategy is a specialized strategy for selecting the best
// vision model. It weighs image understanding quality, GUI element
// detection, OCR accuracy, speed, and cost efficiency. Local models
// (Ollama) get a cost=0 bonus but may score lower on quality.
// Cloud models like Astica score highest due to specialised vision
// capabilities.
//
// This strategy respects ProviderUsageConfig to ensure expensive
// vision providers (like Gemini) are used appropriately while
// excluding providers designated for LLM-only use.
type VisionStrategy struct {
	mu sync.RWMutex

	// imageQualityWeight weights image understanding ability.
	imageQualityWeight float64

	// guiDetectionWeight weights UI element detection ability.
	guiDetectionWeight float64

	// ocrAccuracyWeight weights text recognition accuracy.
	ocrAccuracyWeight float64

	// speedWeight weights response speed.
	speedWeight float64

	// costWeight weights cost efficiency.
	costWeight float64

	// scoreCache caches computed scores.
	scoreCache map[string]cachedVisionScore

	// usageConfig defines how providers should be used.
	// If nil, uses DefaultProviderUsageConfig() which marks
	// Gemini as vision-only and others as appropriate.
	usageConfig *strategy.ProviderUsageConfig
}

type cachedVisionScore struct {
	score     strategy.StrategyScore
	expiresAt time.Time
}

// VisionStrategyConfig holds configuration for the Vision strategy.
type VisionStrategyConfig struct {
	// ImageQualityWeight for image understanding (default: 0.35)
	ImageQualityWeight float64

	// GUIDetectionWeight for UI element detection (default: 0.25)
	GUIDetectionWeight float64

	// OCRAccuracyWeight for text recognition (default: 0.15)
	OCRAccuracyWeight float64

	// SpeedWeight for response speed (default: 0.15)
	SpeedWeight float64

	// CostWeight for cost efficiency (default: 0.10)
	CostWeight float64

	// UsageConfig defines provider categorization. If nil, uses
	// DefaultProviderUsageConfig() which marks Gemini as vision-only.
	UsageConfig *strategy.ProviderUsageConfig

	// PreferVisionOnlyProviders when true (default), prioritizes
	// providers marked as UsageTypeVisionOnly (e.g., Gemini) for
	// vision tasks. These are typically expensive but excellent
	// at vision tasks.
	PreferVisionOnlyProviders bool
}

// DefaultVisionStrategyConfig returns the default Vision strategy
// configuration.
func DefaultVisionStrategyConfig() *VisionStrategyConfig {
	return &VisionStrategyConfig{
		ImageQualityWeight:        0.35,
		GUIDetectionWeight:        0.25,
		OCRAccuracyWeight:         0.15,
		SpeedWeight:               0.15,
		CostWeight:                0.10,
		UsageConfig:               strategy.DefaultProviderUsageConfig(),
		PreferVisionOnlyProviders: true,
	}
}

// NewVisionStrategy creates a new vision-optimised strategy.
func NewVisionStrategy(
	opts ...func(*VisionStrategyConfig),
) *VisionStrategy {
	cfg := DefaultVisionStrategyConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return &VisionStrategy{
		imageQualityWeight: cfg.ImageQualityWeight,
		guiDetectionWeight: cfg.GUIDetectionWeight,
		ocrAccuracyWeight:  cfg.OCRAccuracyWeight,
		speedWeight:        cfg.SpeedWeight,
		costWeight:         cfg.CostWeight,
		scoreCache:         make(map[string]cachedVisionScore),
		usageConfig:        cfg.UsageConfig,
	}
}

// Name returns the strategy identifier.
func (s *VisionStrategy) Name() string {
	return "vision"
}

// Description returns a human-readable description.
func (s *VisionStrategy) Description() string {
	return fmt.Sprintf(
		"Vision strategy: image_quality(%.0f%%), gui_detection(%.0f%%), "+
			"ocr(%.0f%%), speed(%.0f%%), cost(%.0f%%)",
		s.imageQualityWeight*100,
		s.guiDetectionWeight*100,
		s.ocrAccuracyWeight*100,
		s.speedWeight*100,
		s.costWeight*100,
	)
}

// SetWeights sets custom dimension weights. Recognised keys:
//
//	quality  -> imageQualityWeight
//	vision   -> guiDetectionWeight
//	context  -> ocrAccuracyWeight (overloaded)
//	speed    -> speedWeight
//	cost     -> costWeight
func (s *VisionStrategy) SetWeights(weights map[string]float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if w, ok := weights[strategy.DimensionQuality]; ok {
		s.imageQualityWeight = w
	}
	if w, ok := weights[strategy.DimensionVision]; ok {
		s.guiDetectionWeight = w
	}
	if w, ok := weights[strategy.DimensionContext]; ok {
		s.ocrAccuracyWeight = w
	}
	if w, ok := weights[strategy.DimensionSpeed]; ok {
		s.speedWeight = w
	}
	if w, ok := weights[strategy.DimensionCost]; ok {
		s.costWeight = w
	}
}

// SetConstraints is a no-op for vision strategy (satisfies the
// interface expected by WithConstraints functional option).
func (s *VisionStrategy) SetConstraints(_ []strategy.Constraint) {}

// SetFallbacks is a no-op for vision strategy (satisfies the
// interface expected by WithFallbacks functional option).
func (s *VisionStrategy) SetFallbacks(_ []strategy.FallbackRule) {}

// Score evaluates a model with vision-specific scoring dimensions.
//
// Dimensions:
//   - Image quality: derived from QualityScore with bonus for
//     vision-specific capabilities (ocr, object_detection,
//     gui_analysis, face_detection).
//   - GUI detection: capability check for gui_analysis,
//     gui_grounding, gui_navigation, element_detection.
//   - OCR accuracy: capability check for ocr + base quality.
//   - Speed: normalised latency score.
//   - Cost: free models get 1.0; expensive models score lower.
//
// Models that do not support vision get an overall score of 0.
//
// Provider Usage Rules:
// getUsageConfig returns the usage config or default.
func (s *VisionStrategy) getUsageConfig() *strategy.ProviderUsageConfig {
	if s.usageConfig != nil {
		return s.usageConfig
	}
	return strategy.DefaultProviderUsageConfig()
}

// SetUsageConfig updates the provider usage configuration.
func (s *VisionStrategy) SetUsageConfig(config *strategy.ProviderUsageConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.usageConfig = config
}

//   - UsageTypeVisionOnly providers (e.g., Gemini) get a bonus for
//     vision tasks as they are designated for this purpose.
//   - UsageTypeLLMOnly providers are excluded (score = 0).
//   - UsageTypeAny providers are scored normally.
func (s *VisionStrategy) Score(
	ctx context.Context,
	model strategy.ModelInfo,
) (strategy.StrategyScore, error) {
	// Check if provider can be used for vision
	usage := s.getUsageConfig().GetUsage(model.Provider)
	if usage == strategy.UsageTypeLLMOnly {
		// LLM-only providers cannot do vision
		return strategy.StrategyScore{
			Overall:      0,
			Confidence:   0.1,
			Reasoning:    fmt.Sprintf("Provider %s is marked as LLM-only, cannot use for vision", model.Provider),
			Timestamp:    time.Now(),
			ModelID:      model.ID,
			StrategyName: s.Name(),
		}, nil
	}
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

	// --- Image quality dimension ---
	imageQuality := 0.0
	if model.SupportsVision {
		imageQuality = model.QualityScore

		// Bonus for specialised vision capabilities.
		visionCaps := countCapabilities(model.Capabilities,
			"vision", "ocr", "object_detection",
			"face_detection", "content_moderation",
		)
		if visionCaps >= 3 {
			imageQuality = math.Min(1.0, imageQuality+0.05)
		}

		// Bonus for vision-only providers (they're designated for this)
		if usage == strategy.UsageTypeVisionOnly {
			imageQuality = math.Min(1.0, imageQuality+0.10)
		}
	}
	scores[strategy.DimensionQuality] = imageQuality * s.imageQualityWeight

	// --- GUI detection dimension ---
	guiScore := 0.0
	if model.SupportsVision {
		guiCaps := countCapabilities(model.Capabilities,
			"gui_analysis", "gui_grounding",
			"gui_navigation", "element_detection",
		)
		if guiCaps > 0 {
			guiScore = float64(guiCaps) / 4.0
		} else {
			// General vision models can still do basic GUI
			// analysis, just not as well.
			guiScore = 0.4 * model.QualityScore
		}
	}
	scores[strategy.DimensionVision] = guiScore * s.guiDetectionWeight

	// --- OCR accuracy dimension ---
	ocrScore := 0.0
	if model.SupportsVision {
		if hasCapability(model.Capabilities, "ocr") {
			ocrScore = 0.9
		} else {
			// General vision models have some OCR ability.
			ocrScore = 0.5 * model.QualityScore
		}
	}
	scores[strategy.DimensionContext] = ocrScore * s.ocrAccuracyWeight

	// --- Speed dimension ---
	speedScore := 0.5
	if model.AvgLatencyMs > 0 {
		speedScore = 1.0 - float64(model.AvgLatencyMs)/6000.0
		if speedScore < 0 {
			speedScore = 0
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

	totalWeight := s.imageQualityWeight + s.guiDetectionWeight +
		s.ocrAccuracyWeight + s.speedWeight + s.costWeight
	if totalWeight > 0 {
		overall = overall / totalWeight
	}

	// Non-vision models get zero.
	if !model.SupportsVision {
		overall = 0
	}

	// Apply usage-based adjustments
	if usage == strategy.UsageTypeVisionOnly {
		// Vision-only providers get a bonus as they're designated for this
		overall = math.Min(1.0, overall*1.15)
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
		Reasoning:       s.generateReasoning(model, scores, imageQuality, guiScore, ocrScore),
		Timestamp:       time.Now(),
		ModelID:         model.ID,
		StrategyName:    s.Name(),
	}

	s.mu.Lock()
	s.scoreCache[cacheKey] = cachedVisionScore{
		score:     result,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	s.mu.Unlock()

	return result, nil
}

func (s *VisionStrategy) generateReasoning(
	model strategy.ModelInfo,
	scores map[string]float64,
	imageQuality, guiScore, ocrScore float64,
) string {
	visionStatus := "no vision"
	if model.SupportsVision {
		visionStatus = "vision capable"
	}

	return fmt.Sprintf(
		"Vision score %.2f: %s, image_quality %.2f, "+
			"gui_detection %.2f, ocr %.2f, provider %s",
		sumScores(scores),
		visionStatus,
		imageQuality,
		guiScore,
		ocrScore,
		model.Provider,
	)
}

func sumScores(scores map[string]float64) float64 {
	var sum float64
	for _, v := range scores {
		sum += v
	}
	return sum
}

// Validate checks if a model meets vision requirements.
func (s *VisionStrategy) Validate(
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
			"vision capability is required for vision strategy")
	}

	if model.QualityScore < 0.5 {
		result.Warnings = append(result.Warnings,
			"quality score below 0.5 may produce poor image analysis")
	}

	if model.AvgLatencyMs > 5000 {
		result.Warnings = append(result.Warnings,
			"high latency may impact interactive vision workflows")
	}

	score, _ := s.Score(ctx, model)
	result.Score = score.Overall
	result.Details["score"] = score
	result.Details["vision_capable"] = model.SupportsVision

	return result
}

// Rank sorts models by vision-specific criteria. Vision-capable
// models always rank above non-vision models.
func (s *VisionStrategy) Rank(
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
			Tier:  s.determineVisionTier(score.Overall, model.SupportsVision),
		})
	}

	sortVisionModels(ranked)

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

func (s *VisionStrategy) determineVisionTier(
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

// sortVisionModels puts vision-capable models first, then sorts by
// score descending within each group.
func sortVisionModels(ranked []strategy.RankedModel) {
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

// Select chooses the best vision model from the ranked list.
func (s *VisionStrategy) Select(
	ctx context.Context,
	ranked []strategy.RankedModel,
	req strategy.Requirements,
) (strategy.ModelInfo, error) {
	if len(ranked) == 0 {
		return strategy.ModelInfo{},
			fmt.Errorf("no models available for vision selection")
	}

	// Enforce vision requirement by default.
	visionReq := req
	visionReq.NeedsVision = true

	filtered := s.filterByRequirements(ranked, visionReq)
	if len(filtered) == 0 {
		return strategy.ModelInfo{},
			fmt.Errorf("no models meet vision requirements")
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

func (s *VisionStrategy) filterByRequirements(
	ranked []strategy.RankedModel,
	req strategy.Requirements,
) []strategy.RankedModel {
	result := make([]strategy.RankedModel, 0)

	usageConfig := s.getUsageConfig()

	for _, r := range ranked {
		model := r.Model

		// Check provider usage type
		usage := usageConfig.GetUsage(model.Provider)
		if usage == strategy.UsageTypeLLMOnly {
			// LLM-only providers cannot be used for vision
			continue
		}

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
func (s *VisionStrategy) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scoreCache = make(map[string]cachedVisionScore)
}

// --- Capability helpers ---

// countCapabilities counts how many of the target capabilities
// appear in the model's capability list.
func countCapabilities(caps []string, targets ...string) int {
	count := 0
	for _, target := range targets {
		for _, c := range caps {
			if c == target {
				count++
				break
			}
		}
	}
	return count
}

// hasCapability checks if a specific capability is present.
func hasCapability(caps []string, target string) bool {
	for _, c := range caps {
		if c == target {
			return true
		}
	}
	return false
}
