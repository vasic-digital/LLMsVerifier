// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

// Package catalogizer provides a Catalogizer-specific strategy for selecting
// the best available LLM model for general catalog-api tasks such as metadata
// enrichment, content analysis, and recommendations. Unlike the HelixQA
// strategy which prioritizes vision/screenshot analysis, this strategy
// optimizes for text understanding, reasoning, and multilingual support.
package catalogizer

import (
	"context"
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	"digital.vasic.llmsverifier/pkg/strategy"
)

// CatalogizerStrategy selects the best available LLM model for general
// catalog-api tasks (metadata enrichment, content analysis, recommendations).
type CatalogizerStrategy struct {
	mu sync.RWMutex

	// baseStrategy is the underlying default strategy
	baseStrategy strategy.VerificationStrategy

	// qualityWeight is the weight for response quality
	qualityWeight float64

	// costWeight is the weight for cost efficiency
	costWeight float64

	// contextWeight is the weight for context window size
	contextWeight float64

	// reliabilityWeight is the weight for reliability
	reliabilityWeight float64

	// capabilityWeight is the weight for required capabilities
	capabilityWeight float64

	// scoreCache for caching computed scores
	scoreCache map[string]cachedCatalogizerScore

	// usageConfig defines provider categorization for this strategy.
	// If nil, uses DefaultProviderUsageConfig() which marks Gemini
	// as vision-only to preserve it for vision tasks.
	usageConfig *strategy.ProviderUsageConfig
}

type cachedCatalogizerScore struct {
	score     strategy.StrategyScore
	expiresAt time.Time
}

// CatalogizerStrategyConfig holds configuration for the Catalogizer strategy
type CatalogizerStrategyConfig struct {
	// QualityWeight for response quality (default: 0.35)
	QualityWeight float64

	// CostWeight for cost efficiency (default: 0.20)
	CostWeight float64

	// ContextWeight for context window size (default: 0.15)
	ContextWeight float64

	// ReliabilityWeight for reliability (default: 0.20)
	ReliabilityWeight float64

	// CapabilityWeight for required capabilities (default: 0.10)
	CapabilityWeight float64

	// CacheTTL for score caching (default: 5 minutes)
	CacheTTL time.Duration

	// UsageConfig defines provider categorization. If nil, uses
	// DefaultProviderUsageConfig() which marks Gemini as vision-only
	// to preserve it for vision tasks.
	UsageConfig *strategy.ProviderUsageConfig
}

// DefaultCatalogizerStrategyConfig returns the default Catalogizer strategy configuration
func DefaultCatalogizerStrategyConfig() *CatalogizerStrategyConfig {
	return &CatalogizerStrategyConfig{
		QualityWeight:     0.35,
		CostWeight:        0.20,
		ContextWeight:     0.15,
		ReliabilityWeight: 0.20,
		CapabilityWeight:  0.10,
		CacheTTL:          5 * time.Minute,
	}
}

// NewCatalogizerStrategy creates a new Catalogizer-optimized strategy
func NewCatalogizerStrategy(opts ...func(*CatalogizerStrategyConfig)) *CatalogizerStrategy {
	cfg := DefaultCatalogizerStrategyConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return &CatalogizerStrategy{
		baseStrategy:      strategy.NewDefaultStrategy(),
		qualityWeight:     cfg.QualityWeight,
		costWeight:        cfg.CostWeight,
		contextWeight:     cfg.ContextWeight,
		reliabilityWeight: cfg.ReliabilityWeight,
		capabilityWeight:  cfg.CapabilityWeight,
		scoreCache:        make(map[string]cachedCatalogizerScore),
		usageConfig:       cfg.UsageConfig,
	}
}

// getUsageConfig returns the usage config or default
func (s *CatalogizerStrategy) getUsageConfig() *strategy.ProviderUsageConfig {
	if s.usageConfig != nil {
		return s.usageConfig
	}
	return strategy.DefaultProviderUsageConfig()
}

// SetUsageConfig updates the provider usage configuration
func (s *CatalogizerStrategy) SetUsageConfig(config *strategy.ProviderUsageConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.usageConfig = config
}

// Name returns the strategy name
func (s *CatalogizerStrategy) Name() string {
	return "catalogizer"
}

// Description returns the strategy description
func (s *CatalogizerStrategy) Description() string {
	return fmt.Sprintf(
		"Catalogizer strategy: quality(%.0f%%), cost(%.0f%%), context(%.0f%%), reliability(%.0f%%), capability(%.0f%%)",
		s.qualityWeight*100, s.costWeight*100, s.contextWeight*100,
		s.reliabilityWeight*100, s.capabilityWeight*100,
	)
}

// SetWeights sets custom dimension weights
func (s *CatalogizerStrategy) SetWeights(weights map[string]float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if w, ok := weights[strategy.DimensionQuality]; ok {
		s.qualityWeight = w
	}
	if w, ok := weights[strategy.DimensionCost]; ok {
		s.costWeight = w
	}
	if w, ok := weights[strategy.DimensionContext]; ok {
		s.contextWeight = w
	}
	if w, ok := weights[strategy.DimensionReliability]; ok {
		s.reliabilityWeight = w
	}
	if w, ok := weights[strategy.DimensionCapability]; ok {
		s.capabilityWeight = w
	}
}

// SetConstraints sets verification constraints (delegates to base strategy)
func (s *CatalogizerStrategy) SetConstraints(constraints []strategy.Constraint) {
	if cs, ok := s.baseStrategy.(interface{ SetConstraints([]strategy.Constraint) }); ok {
		cs.SetConstraints(constraints)
	}
}

// SetFallbacks sets fallback rules (delegates to base strategy)
func (s *CatalogizerStrategy) SetFallbacks(fallbacks []strategy.FallbackRule) {
	if fs, ok := s.baseStrategy.(interface{ SetFallbacks([]strategy.FallbackRule) }); ok {
		fs.SetFallbacks(fallbacks)
	}
}

// Score evaluates a model with Catalogizer-specific scoring
func (s *CatalogizerStrategy) Score(ctx context.Context, model strategy.ModelInfo) (strategy.StrategyScore, error) {
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

	// Quality: highest weight -- accurate metadata enrichment and content analysis
	scores[strategy.DimensionQuality] = model.QualityScore * s.qualityWeight

	// Cost: important for high-volume catalog operations
	costScore := 0.5
	if model.InputCostPer1k > 0 || model.OutputCostPer1k > 0 {
		totalCost := model.InputCostPer1k + model.OutputCostPer1k
		costScore = 1.0 - (totalCost / 0.05)
		if costScore < 0 {
			costScore = 0
		}
		if costScore > 1 {
			costScore = 1
		}
	} else {
		// Free models get full cost score
		costScore = 1.0
	}
	scores[strategy.DimensionCost] = costScore * s.costWeight

	// Context: larger context windows handle richer metadata payloads
	contextScore := 0.5
	if model.ContextWindow > 0 {
		// Normalize: 128K = 1.0, scale logarithmically
		contextScore = math.Log2(float64(model.ContextWindow)) / math.Log2(128000.0)
		if contextScore > 1 {
			contextScore = 1
		}
		if contextScore < 0 {
			contextScore = 0
		}
	}
	scores[strategy.DimensionContext] = contextScore * s.contextWeight

	// Reliability: catalog operations must complete consistently
	scores[strategy.DimensionReliability] = model.ReliabilityScore * s.reliabilityWeight

	// Capability: bonus for text, code, reasoning, multilingual
	capScore := 0.0
	targetCaps := []string{"text", "code", "reasoning", "multilingual"}
	if len(model.Capabilities) > 0 {
		matches := 0
		for _, target := range targetCaps {
			for _, mc := range model.Capabilities {
				if mc == target {
					matches++
					break
				}
			}
		}
		capScore = float64(matches) / float64(len(targetCaps))
	}
	scores[strategy.DimensionCapability] = capScore * s.capabilityWeight

	// Compute overall weighted score
	var overall float64
	for _, v := range scores {
		overall += v
	}

	totalWeight := s.qualityWeight + s.costWeight + s.contextWeight + s.reliabilityWeight + s.capabilityWeight
	if totalWeight > 0 {
		overall = overall / totalWeight
	}

	// Bonus: models with large context windows (>= 200K) get a small boost
	// for handling large catalog metadata payloads
	if model.ContextWindow >= 200000 {
		overall = math.Min(1.0, overall+0.05)
	}

	// Confidence calculation
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
		Reasoning:       s.generateReasoning(model, scores, capScore),
		Timestamp:       time.Now(),
		ModelID:         model.ID,
		StrategyName:    s.Name(),
	}

	s.mu.Lock()
	s.scoreCache[cacheKey] = cachedCatalogizerScore{
		score:     result,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	s.mu.Unlock()

	return result, nil
}

func (s *CatalogizerStrategy) generateReasoning(model strategy.ModelInfo, scores map[string]float64, capScore float64) string {
	return fmt.Sprintf(
		"Catalogizer score %.2f: quality %.2f, cost %.2f, context %dK, reliability %.2f, capabilities %.0f%%",
		sumCatalogizerScores(scores),
		model.QualityScore,
		model.InputCostPer1k+model.OutputCostPer1k,
		model.ContextWindow/1000,
		model.ReliabilityScore,
		capScore*100,
	)
}

func sumCatalogizerScores(scores map[string]float64) float64 {
	var sum float64
	for _, v := range scores {
		sum += v
	}
	return sum
}

// Validate checks if a model meets Catalogizer requirements
func (s *CatalogizerStrategy) Validate(ctx context.Context, model strategy.ModelInfo) strategy.ValidationResult {
	result := strategy.ValidationResult{
		Valid:    true,
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
		Details:  make(map[string]any),
	}

	if model.QualityScore < 0.7 {
		result.Valid = false
		result.Errors = append(result.Errors,
			"quality score below 0.7 insufficient for accurate metadata enrichment")
	}

	if model.ReliabilityScore < 0.8 {
		result.Warnings = append(result.Warnings,
			"reliability score below 0.8 may cause inconsistent catalog operations")
	}

	if model.ContextWindow < 32000 {
		result.Warnings = append(result.Warnings,
			"context window below 32K may limit metadata payload handling")
	}

	score, _ := s.Score(ctx, model)
	result.Score = score.Overall
	result.Details["score"] = score

	return result
}

// Rank sorts models by Catalogizer-specific criteria
func (s *CatalogizerStrategy) Rank(ctx context.Context, models []strategy.ModelInfo) ([]strategy.RankedModel, error) {
	ranked := make([]strategy.RankedModel, 0, len(models))

	for _, model := range models {
		score, err := s.Score(ctx, model)
		if err != nil {
			continue
		}

		ranked = append(ranked, strategy.RankedModel{
			Model: model,
			Score: score,
			Tier:  s.determineTier(score.Overall),
		})
	}

	sortCatalogizerModels(ranked)

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

func (s *CatalogizerStrategy) determineTier(score float64) string {
	if score >= 0.75 {
		return strategy.Tier1
	}
	if score >= 0.55 {
		return strategy.Tier2
	}
	return strategy.Tier3
}

func sortCatalogizerModels(ranked []strategy.RankedModel) {
	for i := 0; i < len(ranked)-1; i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].Score.Overall > ranked[i].Score.Overall {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}
}

// Select chooses the best model for Catalogizer tasks
func (s *CatalogizerStrategy) Select(ctx context.Context, ranked []strategy.RankedModel, req strategy.Requirements) (strategy.ModelInfo, error) {
	if len(ranked) == 0 {
		return strategy.ModelInfo{}, fmt.Errorf("no models available for Catalogizer tasks")
	}

	catReq := s.enhanceRequirements(req)
	filtered := s.filterByRequirements(ranked, catReq)

	if len(filtered) == 0 {
		return strategy.ModelInfo{}, fmt.Errorf("no models meet Catalogizer requirements")
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

func (s *CatalogizerStrategy) enhanceRequirements(req strategy.Requirements) strategy.Requirements {
	if req.MinQualityScore == 0 {
		req.MinQualityScore = 0.7
	}
	if req.MinReliabilityScore == 0 {
		req.MinReliabilityScore = 0.8
	}
	if req.MinContextWindow == 0 {
		req.MinContextWindow = 32000
	}
	return req
}

func (s *CatalogizerStrategy) filterByRequirements(ranked []strategy.RankedModel, req strategy.Requirements) []strategy.RankedModel {
	result := make([]strategy.RankedModel, 0)
	usageConfig := s.getUsageConfig()

	for _, r := range ranked {
		model := r.Model

		// Check provider usage type
		usage := usageConfig.GetUsage(model.Provider)

		// For general LLM tasks, exclude vision-only providers
		// to preserve them for vision-specific work
		if usage == strategy.UsageTypeVisionOnly && !req.NeedsVision {
			// Skip vision-only providers for general LLM tasks
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
		if req.MinQualityScore > 0 && model.QualityScore < req.MinQualityScore {
			continue
		}
		if req.MinReliabilityScore > 0 && model.ReliabilityScore < req.MinReliabilityScore {
			continue
		}
		if req.MinContextWindow > 0 && model.ContextWindow < req.MinContextWindow {
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

// ClearCache clears the score cache
func (s *CatalogizerStrategy) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scoreCache = make(map[string]cachedCatalogizerScore)
}

// --- Model Registry ---

// providerEnvKeys maps provider names to their API key environment variables
var providerEnvKeys = map[string]string{
	"anthropic": "ANTHROPIC_API_KEY",
	"openai":    "OPENAI_API_KEY",
	"google":    "GOOGLE_API_KEY",
	"kimi":      "KIMI_API_KEY",
	"deepseek":  "DEEPSEEK_API_KEY",
	"groq":      "GROQ_API_KEY",
	"cerebras":  "CEREBRAS_API_KEY",
	"ollama":    "", // Local, no key needed
}

// ModelRegistry returns all known models for Catalogizer tasks.
// These entries feed into the CatalogizerStrategy scoring and ranking
// pipeline for metadata enrichment, content analysis, and recommendations.
func ModelRegistry() []strategy.ModelInfo {
	return []strategy.ModelInfo{
		// --- Tier 1: Premium (best quality) ---
		{
			ID:                      "anthropic-sonnet",
			Name:                    "Claude Sonnet 4",
			Provider:                "anthropic",
			Model:                   "claude-sonnet-4-20250514",
			SupportsVision:          true,
			SupportsStreaming:       true,
			SupportsFunctionCalling: true,
			ContextWindow:           200000,
			MaxOutputTokens:         8192,
			AvgLatencyMs:            1500,
			InputCostPer1k:          0.003,
			OutputCostPer1k:         0.015,
			QualityScore:            0.95,
			ReliabilityScore:        0.97,
			Capabilities:            []string{"text", "code", "reasoning"},
		},
		{
			ID:                      "openai-gpt4o",
			Name:                    "GPT-4o",
			Provider:                "openai",
			Model:                   "gpt-4o",
			SupportsVision:          true,
			SupportsStreaming:       true,
			SupportsFunctionCalling: true,
			ContextWindow:           128000,
			MaxOutputTokens:         4096,
			AvgLatencyMs:            1200,
			InputCostPer1k:          0.005,
			OutputCostPer1k:         0.015,
			QualityScore:            0.93,
			ReliabilityScore:        0.98,
			Capabilities:            []string{"text", "code", "reasoning"},
		},
		{
			ID:                      "gemini-flash",
			Name:                    "Gemini 2.0 Flash",
			Provider:                "google",
			Model:                   "gemini-2.5-flash",
			SupportsVision:          true,
			SupportsStreaming:       true,
			SupportsFunctionCalling: true,
			ContextWindow:           1000000,
			MaxOutputTokens:         8192,
			AvgLatencyMs:            800,
			InputCostPer1k:          0.0001,
			OutputCostPer1k:         0.0004,
			QualityScore:            0.90,
			ReliabilityScore:        0.96,
			Capabilities:            []string{"text", "code", "reasoning", "multilingual"},
		},

		// --- Tier 2: Value (good quality, low cost) ---
		{
			ID:                      "kimi-k25",
			Name:                    "Kimi K2.5",
			Provider:                "kimi",
			Model:                   "kimi-k2.5",
			SupportsVision:          true,
			SupportsStreaming:       true,
			SupportsFunctionCalling: true,
			ContextWindow:           256000,
			MaxOutputTokens:         8192,
			AvgLatencyMs:            1000,
			InputCostPer1k:          0.0006,
			OutputCostPer1k:         0.0006,
			QualityScore:            0.88,
			ReliabilityScore:        0.90,
			Capabilities:            []string{"text", "code", "reasoning", "multilingual"},
			Metadata: map[string]any{
				"architecture": "1T MoE",
				"license":      "MIT",
				"api_compat":   "openai",
			},
		},
		{
			ID:                      "deepseek-chat",
			Name:                    "DeepSeek Chat",
			Provider:                "deepseek",
			Model:                   "deepseek-chat",
			SupportsVision:          false,
			SupportsStreaming:       true,
			SupportsFunctionCalling: true,
			ContextWindow:           128000,
			MaxOutputTokens:         8192,
			AvgLatencyMs:            900,
			InputCostPer1k:          0.00014,
			OutputCostPer1k:         0.00028,
			QualityScore:            0.85,
			ReliabilityScore:        0.88,
			Capabilities:            []string{"text", "code", "reasoning"},
		},

		// --- Tier 3: Budget (acceptable quality, very cheap) ---
		{
			ID:                      "groq-llama70b",
			Name:                    "Llama 3.3 70B (Groq)",
			Provider:                "groq",
			Model:                   "llama-3.3-70b-versatile",
			SupportsVision:          false,
			SupportsStreaming:       true,
			SupportsFunctionCalling: false,
			ContextWindow:           128000,
			MaxOutputTokens:         8192,
			AvgLatencyMs:            300,
			InputCostPer1k:          0.00006,
			OutputCostPer1k:         0.00006,
			QualityScore:            0.80,
			ReliabilityScore:        0.85,
			Capabilities:            []string{"text", "reasoning"},
		},
		{
			ID:                      "cerebras-llama70b",
			Name:                    "Llama 3.3 70B (Cerebras)",
			Provider:                "cerebras",
			Model:                   "llama-3.3-70b",
			SupportsVision:          false,
			SupportsStreaming:       true,
			SupportsFunctionCalling: false,
			ContextWindow:           128000,
			MaxOutputTokens:         8192,
			AvgLatencyMs:            200,
			InputCostPer1k:          0.0,
			OutputCostPer1k:         0.0,
			QualityScore:            0.80,
			ReliabilityScore:        0.82,
			Capabilities:            []string{"text", "reasoning"},
		},

		// --- Tier 4: Local (free, always available) ---
		{
			ID:                      "ollama-llama31",
			Name:                    "Llama 3.1 8B (Ollama)",
			Provider:                "ollama",
			Model:                   "llama3.1:8b",
			SupportsVision:          false,
			SupportsStreaming:       true,
			SupportsFunctionCalling: false,
			ContextWindow:           128000,
			MaxOutputTokens:         4096,
			AvgLatencyMs:            2000,
			InputCostPer1k:          0.0,
			OutputCostPer1k:         0.0,
			QualityScore:            0.70,
			ReliabilityScore:        0.80,
			Capabilities:            []string{"text"},
		},
	}
}

// SelectBest returns the highest-quality model with a configured API key.
// It checks environment variables for each provider and returns the
// best available model, falling back to local Ollama if nothing else is available.
func SelectBest() *strategy.ModelInfo {
	for _, m := range ModelRegistry() {
		if isProviderAvailable(m.Provider) {
			model := m
			return &model
		}
	}
	return nil
}

// SelectBudget returns the best model under a cost threshold (per 1K tokens).
// Returns nil if no model meets the budget constraint.
func SelectBudget(maxCostPer1K float64) *strategy.ModelInfo {
	for _, m := range ModelRegistry() {
		totalCost := m.InputCostPer1k + m.OutputCostPer1k
		if totalCost <= maxCostPer1K && isProviderAvailable(m.Provider) {
			model := m
			return &model
		}
	}
	return nil
}

// SelectByCapability returns the best available model that has all
// the specified capabilities.
func SelectByCapability(required []string) *strategy.ModelInfo {
	for _, m := range ModelRegistry() {
		if !isProviderAvailable(m.Provider) {
			continue
		}
		hasAll := true
		for _, req := range required {
			found := false
			for _, cap := range m.Capabilities {
				if cap == req {
					found = true
					break
				}
			}
			if !found {
				hasAll = false
				break
			}
		}
		if hasAll {
			model := m
			return &model
		}
	}
	return nil
}

// AvailableModels returns all models whose provider has a configured API key.
func AvailableModels() []strategy.ModelInfo {
	var available []strategy.ModelInfo
	for _, m := range ModelRegistry() {
		if isProviderAvailable(m.Provider) {
			available = append(available, m)
		}
	}
	return available
}

// isProviderAvailable checks if a provider has a configured API key.
// Ollama (local) is always considered available.
func isProviderAvailable(provider string) bool {
	envKey, ok := providerEnvKeys[provider]
	if !ok {
		return false
	}
	// Local providers are always available
	if envKey == "" {
		return true
	}
	return os.Getenv(envKey) != ""
}
