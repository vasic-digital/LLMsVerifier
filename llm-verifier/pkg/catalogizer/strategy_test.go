// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package catalogizer

import (
	"context"
	"os"
	"testing"
	"time"

	"digital.vasic.llmsverifier/pkg/strategy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogizerStrategy_Name(t *testing.T) {
	s := NewCatalogizerStrategy()
	assert.Equal(t, "catalogizer", s.Name())
}

func TestCatalogizerStrategy_Description(t *testing.T) {
	s := NewCatalogizerStrategy()
	desc := s.Description()
	assert.Contains(t, desc, "quality")
	assert.Contains(t, desc, "cost")
	assert.Contains(t, desc, "context")
	assert.Contains(t, desc, "reliability")
	assert.Contains(t, desc, "capability")
}

func TestCatalogizerStrategy_Score(t *testing.T) {
	ctx := context.Background()
	s := NewCatalogizerStrategy()

	model := strategy.ModelInfo{
		ID:               "test-model",
		Name:             "Test Model",
		Provider:         "test",
		Model:            "test-1",
		QualityScore:     0.9,
		ReliabilityScore: 0.95,
		ContextWindow:    128000,
		AvgLatencyMs:     500,
		InputCostPer1k:   0.003,
		OutputCostPer1k:  0.015,
		Verified:         true,
		LastVerified:     time.Now(),
		Capabilities:     []string{"text", "code", "reasoning"},
	}

	score, err := s.Score(ctx, model)
	require.NoError(t, err)

	assert.Equal(t, "test-model", score.ModelID)
	assert.Equal(t, "catalogizer", score.StrategyName)
	assert.GreaterOrEqual(t, score.Overall, 0.0)
	assert.LessOrEqual(t, score.Overall, 1.0)
	assert.NotEmpty(t, score.Reasoning)
	assert.NotEmpty(t, score.DimensionScores)
	assert.Contains(t, score.DimensionScores, strategy.DimensionQuality)
	assert.Contains(t, score.DimensionScores, strategy.DimensionCost)
	assert.Contains(t, score.DimensionScores, strategy.DimensionContext)
	assert.Contains(t, score.DimensionScores, strategy.DimensionReliability)
	assert.Contains(t, score.DimensionScores, strategy.DimensionCapability)
}

func TestCatalogizerStrategy_Score_VerifiedModel(t *testing.T) {
	ctx := context.Background()
	s := NewCatalogizerStrategy()

	verifiedModel := strategy.ModelInfo{
		ID:               "verified",
		QualityScore:     0.8,
		ReliabilityScore: 0.9,
		ContextWindow:    128000,
		Verified:         true,
		LastVerified:     time.Now(),
	}

	unverifiedModel := strategy.ModelInfo{
		ID:               "unverified",
		QualityScore:     0.8,
		ReliabilityScore: 0.9,
		ContextWindow:    128000,
		Verified:         false,
	}

	verifiedScore, _ := s.Score(ctx, verifiedModel)
	unverifiedScore, _ := s.Score(ctx, unverifiedModel)

	assert.Greater(t, verifiedScore.Confidence, unverifiedScore.Confidence)
}

func TestCatalogizerStrategy_Score_Caching(t *testing.T) {
	ctx := context.Background()
	s := NewCatalogizerStrategy()

	model := strategy.ModelInfo{
		ID:               "cached-model",
		QualityScore:     0.8,
		ReliabilityScore: 0.9,
		ContextWindow:    128000,
	}

	score1, err := s.Score(ctx, model)
	require.NoError(t, err)

	score2, err := s.Score(ctx, model)
	require.NoError(t, err)

	assert.Equal(t, score1.Timestamp, score2.Timestamp)
}

func TestCatalogizerStrategy_Score_LargeContextBonus(t *testing.T) {
	ctx := context.Background()
	s := NewCatalogizerStrategy()

	largeContext := strategy.ModelInfo{
		ID:               "large-ctx",
		QualityScore:     0.85,
		ReliabilityScore: 0.90,
		ContextWindow:    200000,
	}

	smallContext := strategy.ModelInfo{
		ID:               "small-ctx",
		QualityScore:     0.85,
		ReliabilityScore: 0.90,
		ContextWindow:    32000,
	}

	largeScore, _ := s.Score(ctx, largeContext)
	smallScore, _ := s.Score(ctx, smallContext)

	// Large context model gets a bonus
	assert.Greater(t, largeScore.Overall, smallScore.Overall)
}

func TestCatalogizerStrategy_Score_FreeModelCostAdvantage(t *testing.T) {
	ctx := context.Background()
	s := NewCatalogizerStrategy()

	freeModel := strategy.ModelInfo{
		ID:               "free",
		QualityScore:     0.80,
		ReliabilityScore: 0.85,
		ContextWindow:    128000,
		InputCostPer1k:   0.0,
		OutputCostPer1k:  0.0,
	}

	expensiveModel := strategy.ModelInfo{
		ID:               "expensive",
		QualityScore:     0.80,
		ReliabilityScore: 0.85,
		ContextWindow:    128000,
		InputCostPer1k:   0.01,
		OutputCostPer1k:  0.03,
	}

	freeScore, _ := s.Score(ctx, freeModel)
	expensiveScore, _ := s.Score(ctx, expensiveModel)

	assert.Greater(t, freeScore.DimensionScores[strategy.DimensionCost],
		expensiveScore.DimensionScores[strategy.DimensionCost])
}

func TestCatalogizerStrategy_Validate(t *testing.T) {
	ctx := context.Background()
	s := NewCatalogizerStrategy()

	model := strategy.ModelInfo{
		ID:               "valid-model",
		QualityScore:     0.9,
		ReliabilityScore: 0.95,
		ContextWindow:    128000,
		Verified:         true,
	}

	result := s.Validate(ctx, model)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestCatalogizerStrategy_Validate_LowQuality(t *testing.T) {
	ctx := context.Background()
	s := NewCatalogizerStrategy()

	model := strategy.ModelInfo{
		ID:               "low-quality",
		QualityScore:     0.5,
		ReliabilityScore: 0.95,
		ContextWindow:    128000,
	}

	result := s.Validate(ctx, model)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

func TestCatalogizerStrategy_Validate_Warnings(t *testing.T) {
	ctx := context.Background()
	s := NewCatalogizerStrategy()

	model := strategy.ModelInfo{
		ID:               "warning-model",
		QualityScore:     0.75,
		ReliabilityScore: 0.70,
		ContextWindow:    16000,
	}

	result := s.Validate(ctx, model)
	assert.True(t, result.Valid)
	assert.Len(t, result.Warnings, 2) // low reliability + small context
}

func TestCatalogizerStrategy_Rank(t *testing.T) {
	ctx := context.Background()
	s := NewCatalogizerStrategy()

	models := []strategy.ModelInfo{
		{ID: "low", Name: "Low Quality", QualityScore: 0.5, ReliabilityScore: 0.8, ContextWindow: 32000},
		{ID: "high", Name: "High Quality", QualityScore: 0.95, ReliabilityScore: 0.98, ContextWindow: 200000},
		{ID: "mid", Name: "Mid Quality", QualityScore: 0.75, ReliabilityScore: 0.9, ContextWindow: 128000},
	}

	ranked, err := s.Rank(ctx, models)
	require.NoError(t, err)

	assert.Len(t, ranked, 3)
	assert.Equal(t, 1, ranked[0].Rank)
	assert.Equal(t, "high", ranked[0].Model.ID)
	assert.Equal(t, 2, ranked[1].Rank)
	assert.Equal(t, "mid", ranked[1].Model.ID)
	assert.Equal(t, 3, ranked[2].Rank)
	assert.Equal(t, "low", ranked[2].Model.ID)
}

func TestCatalogizerStrategy_Rank_Tiers(t *testing.T) {
	ctx := context.Background()
	s := NewCatalogizerStrategy()

	models := []strategy.ModelInfo{
		{ID: "tier1", QualityScore: 0.95, ReliabilityScore: 0.98, ContextWindow: 200000},
		{ID: "tier2", QualityScore: 0.60, ReliabilityScore: 0.75, ContextWindow: 32000},
		{ID: "tier3", QualityScore: 0.30, ReliabilityScore: 0.50, ContextWindow: 2048},
	}

	ranked, _ := s.Rank(ctx, models)

	assert.Equal(t, strategy.Tier1, ranked[0].Tier)
	assert.Equal(t, strategy.Tier2, ranked[1].Tier)
	assert.Equal(t, strategy.Tier3, ranked[2].Tier)
}

func TestCatalogizerStrategy_Rank_SelectionProbability(t *testing.T) {
	ctx := context.Background()
	s := NewCatalogizerStrategy()

	models := []strategy.ModelInfo{
		{ID: "a", QualityScore: 0.9, ReliabilityScore: 0.95, ContextWindow: 128000},
		{ID: "b", QualityScore: 0.7, ReliabilityScore: 0.85, ContextWindow: 64000},
	}

	ranked, _ := s.Rank(ctx, models)

	totalProb := 0.0
	for _, r := range ranked {
		totalProb += r.SelectionProbability
	}
	assert.InDelta(t, 1.0, totalProb, 0.001)
}

func TestCatalogizerStrategy_Select(t *testing.T) {
	ctx := context.Background()
	s := NewCatalogizerStrategy()

	ranked := []strategy.RankedModel{
		{
			Rank: 1,
			Model: strategy.ModelInfo{
				ID:               "best",
				QualityScore:     0.95,
				ReliabilityScore: 0.98,
				ContextWindow:    200000,
			},
		},
		{
			Rank: 2,
			Model: strategy.ModelInfo{
				ID:               "second",
				QualityScore:     0.8,
				ReliabilityScore: 0.9,
				ContextWindow:    128000,
			},
		},
	}

	req := strategy.Requirements{}
	selected, err := s.Select(ctx, ranked, req)
	require.NoError(t, err)
	assert.Equal(t, "best", selected.ID)
}

func TestCatalogizerStrategy_Select_PreferredProvider(t *testing.T) {
	ctx := context.Background()
	s := NewCatalogizerStrategy()

	ranked := []strategy.RankedModel{
		{
			Rank: 1,
			Model: strategy.ModelInfo{
				ID:               "openai-best",
				Provider:         "openai",
				QualityScore:     0.95,
				ReliabilityScore: 0.98,
				ContextWindow:    128000,
			},
		},
		{
			Rank: 2,
			Model: strategy.ModelInfo{
				ID:               "anthropic-good",
				Provider:         "anthropic",
				QualityScore:     0.9,
				ReliabilityScore: 0.95,
				ContextWindow:    200000,
			},
		},
	}

	req := strategy.Requirements{
		PreferredProvider: "anthropic",
	}

	selected, err := s.Select(ctx, ranked, req)
	require.NoError(t, err)
	assert.Equal(t, "anthropic", selected.Provider)
}

func TestCatalogizerStrategy_Select_NoMatchingRequirements(t *testing.T) {
	ctx := context.Background()
	s := NewCatalogizerStrategy()

	ranked := []strategy.RankedModel{
		{
			Model: strategy.ModelInfo{
				ID:               "low-quality",
				QualityScore:     0.5,
				ReliabilityScore: 0.95,
				ContextWindow:    128000,
			},
		},
	}

	req := strategy.Requirements{
		MinQualityScore: 0.9,
	}

	_, err := s.Select(ctx, ranked, req)
	assert.Error(t, err)
}

func TestCatalogizerStrategy_Select_EmptyRankedList(t *testing.T) {
	ctx := context.Background()
	s := NewCatalogizerStrategy()

	_, err := s.Select(ctx, []strategy.RankedModel{}, strategy.Requirements{})
	assert.Error(t, err)
}

func TestCatalogizerStrategy_WithCustomConfig(t *testing.T) {
	s := NewCatalogizerStrategy(func(cfg *CatalogizerStrategyConfig) {
		cfg.QualityWeight = 0.50
		cfg.CostWeight = 0.10
	})

	assert.Equal(t, 0.50, s.qualityWeight)
	assert.Equal(t, 0.10, s.costWeight)
}

func TestCatalogizerStrategy_SetWeights(t *testing.T) {
	s := NewCatalogizerStrategy()
	s.SetWeights(map[string]float64{
		strategy.DimensionQuality: 0.5,
		strategy.DimensionContext: 0.3,
	})

	assert.Equal(t, 0.5, s.qualityWeight)
	assert.Equal(t, 0.3, s.contextWeight)
}

func TestCatalogizerStrategy_ClearCache(t *testing.T) {
	ctx := context.Background()
	s := NewCatalogizerStrategy()

	model := strategy.ModelInfo{
		ID:               "cache-test",
		QualityScore:     0.8,
		ReliabilityScore: 0.9,
		ContextWindow:    128000,
	}

	score1, _ := s.Score(ctx, model)
	s.ClearCache()
	score2, _ := s.Score(ctx, model)

	// After cache clear, timestamps should differ
	assert.NotEqual(t, score1.Timestamp, score2.Timestamp)
}

func TestCatalogizerStrategy_FilterByRequirements_ExcludedProvider(t *testing.T) {
	s := NewCatalogizerStrategy()

	ranked := []strategy.RankedModel{
		{Model: strategy.ModelInfo{ID: "1", Provider: "openai", QualityScore: 0.9, ReliabilityScore: 0.95, ContextWindow: 128000}},
		{Model: strategy.ModelInfo{ID: "2", Provider: "anthropic", QualityScore: 0.9, ReliabilityScore: 0.95, ContextWindow: 128000}},
		{Model: strategy.ModelInfo{ID: "3", Provider: "deepseek", QualityScore: 0.9, ReliabilityScore: 0.95, ContextWindow: 128000}},
	}

	req := strategy.Requirements{
		ExcludedProviders: []string{"anthropic"},
	}

	filtered := s.filterByRequirements(ranked, req)
	assert.Len(t, filtered, 2)
	for _, f := range filtered {
		assert.NotEqual(t, "anthropic", f.Model.Provider)
	}
}

func TestCatalogizerStrategy_FilterByRequirements_RequiredCapabilities(t *testing.T) {
	s := NewCatalogizerStrategy()

	ranked := []strategy.RankedModel{
		{Model: strategy.ModelInfo{ID: "1", QualityScore: 0.9, ReliabilityScore: 0.95, ContextWindow: 128000, Capabilities: []string{"text", "code", "reasoning"}}},
		{Model: strategy.ModelInfo{ID: "2", QualityScore: 0.9, ReliabilityScore: 0.95, ContextWindow: 128000, Capabilities: []string{"text"}}},
		{Model: strategy.ModelInfo{ID: "3", QualityScore: 0.9, ReliabilityScore: 0.95, ContextWindow: 128000, Capabilities: []string{"text", "code", "reasoning", "multilingual"}}},
	}

	req := strategy.Requirements{
		RequiredCapabilities: []string{"code", "reasoning"},
	}

	filtered := s.filterByRequirements(ranked, req)
	assert.Len(t, filtered, 2)
	assert.Equal(t, "1", filtered[0].Model.ID)
	assert.Equal(t, "3", filtered[1].Model.ID)
}

// --- Model Registry Tests ---

func TestModelRegistry_NotEmpty(t *testing.T) {
	models := ModelRegistry()
	assert.NotEmpty(t, models)
	assert.GreaterOrEqual(t, len(models), 8)
}

func TestModelRegistry_AllHaveRequiredFields(t *testing.T) {
	for _, m := range ModelRegistry() {
		assert.NotEmpty(t, m.ID, "model ID must not be empty")
		assert.NotEmpty(t, m.Name, "model name must not be empty for %s", m.ID)
		assert.NotEmpty(t, m.Provider, "provider must not be empty for %s", m.ID)
		assert.NotEmpty(t, m.Model, "model identifier must not be empty for %s", m.ID)
		assert.Greater(t, m.ContextWindow, 0, "context window must be positive for %s", m.ID)
		assert.Greater(t, m.MaxOutputTokens, 0, "max output tokens must be positive for %s", m.ID)
		assert.Greater(t, m.QualityScore, 0.0, "quality score must be positive for %s", m.ID)
		assert.Greater(t, m.ReliabilityScore, 0.0, "reliability score must be positive for %s", m.ID)
		assert.NotEmpty(t, m.Capabilities, "capabilities must not be empty for %s", m.ID)
	}
}

func TestModelRegistry_QualityDescending(t *testing.T) {
	models := ModelRegistry()
	for i := 1; i < len(models); i++ {
		assert.GreaterOrEqual(t, models[i-1].QualityScore, models[i].QualityScore,
			"models should be ordered by quality descending: %s (%.2f) before %s (%.2f)",
			models[i-1].ID, models[i-1].QualityScore, models[i].ID, models[i].QualityScore)
	}
}

func TestModelRegistry_ContainsExpectedProviders(t *testing.T) {
	providers := make(map[string]bool)
	for _, m := range ModelRegistry() {
		providers[m.Provider] = true
	}

	expected := []string{"anthropic", "openai", "google", "kimi", "deepseek", "groq", "cerebras", "ollama"}
	for _, p := range expected {
		assert.True(t, providers[p], "registry should contain provider %s", p)
	}
}

func TestModelRegistry_ContainsKimi(t *testing.T) {
	found := false
	for _, m := range ModelRegistry() {
		if m.Provider == "kimi" {
			found = true
			assert.Equal(t, "kimi-k2.5", m.Model)
			assert.True(t, m.SupportsVision)
			assert.True(t, m.SupportsStreaming)
			assert.NotNil(t, m.Metadata)
			assert.Equal(t, "MIT", m.Metadata["license"])
			break
		}
	}
	assert.True(t, found, "kimi provider should be in registry")
}

func TestCatalogizerStrategy_ScoresAllRegistryModels(t *testing.T) {
	ctx := context.Background()
	s := NewCatalogizerStrategy()

	for _, m := range ModelRegistry() {
		score, err := s.Score(ctx, m)
		require.NoError(t, err, "scoring should not fail for %s", m.ID)
		assert.Greater(t, score.Overall, 0.0, "score should be positive for %s", m.ID)
		assert.LessOrEqual(t, score.Overall, 1.0, "score should be <= 1 for %s", m.ID)
		assert.NotEmpty(t, score.Reasoning, "reasoning should not be empty for %s", m.ID)
	}
}

func TestCatalogizerStrategy_RanksAllRegistryModels(t *testing.T) {
	ctx := context.Background()
	s := NewCatalogizerStrategy()

	ranked, err := s.Rank(ctx, ModelRegistry())
	require.NoError(t, err)
	assert.Equal(t, len(ModelRegistry()), len(ranked))

	for _, r := range ranked {
		assert.NotEqual(t, "", r.Tier, "tier should be set for %s", r.Model.ID)
	}
}

// --- SelectBest / SelectBudget Tests ---

func TestSelectBest_ReturnsOllama_WhenNoKeysSet(t *testing.T) {
	// Clear all API key env vars for this test
	keys := []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GOOGLE_API_KEY",
		"KIMI_API_KEY", "DEEPSEEK_API_KEY", "GROQ_API_KEY", "CEREBRAS_API_KEY"}
	saved := make(map[string]string)
	for _, k := range keys {
		saved[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	defer func() {
		for k, v := range saved {
			if v != "" {
				os.Setenv(k, v)
			}
		}
	}()

	best := SelectBest()
	require.NotNil(t, best)
	assert.Equal(t, "ollama", best.Provider)
}

func TestSelectBest_ReturnsHighestQuality_WhenKeySet(t *testing.T) {
	saved := os.Getenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	defer func() {
		if saved != "" {
			os.Setenv("ANTHROPIC_API_KEY", saved)
		} else {
			os.Unsetenv("ANTHROPIC_API_KEY")
		}
	}()

	best := SelectBest()
	require.NotNil(t, best)
	assert.Equal(t, "anthropic", best.Provider)
}

func TestSelectBudget_ReturnsCheapModel(t *testing.T) {
	saved := os.Getenv("GROQ_API_KEY")
	os.Setenv("GROQ_API_KEY", "test-key")
	defer func() {
		if saved != "" {
			os.Setenv("GROQ_API_KEY", saved)
		} else {
			os.Unsetenv("GROQ_API_KEY")
		}
	}()

	budget := SelectBudget(0.001)
	require.NotNil(t, budget)
	totalCost := budget.InputCostPer1k + budget.OutputCostPer1k
	assert.LessOrEqual(t, totalCost, 0.001)
}

func TestSelectBudget_ReturnsNil_WhenNoBudgetMatch(t *testing.T) {
	// Clear all keys
	keys := []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GOOGLE_API_KEY",
		"KIMI_API_KEY", "DEEPSEEK_API_KEY", "GROQ_API_KEY", "CEREBRAS_API_KEY"}
	saved := make(map[string]string)
	for _, k := range keys {
		saved[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	defer func() {
		for k, v := range saved {
			if v != "" {
				os.Setenv(k, v)
			}
		}
	}()

	// Budget of -1 should match nothing
	budget := SelectBudget(-1)
	assert.Nil(t, budget)
}

func TestSelectByCapability(t *testing.T) {
	// Clear all keys so only ollama is available
	keys := []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GOOGLE_API_KEY",
		"KIMI_API_KEY", "DEEPSEEK_API_KEY", "GROQ_API_KEY", "CEREBRAS_API_KEY"}
	saved := make(map[string]string)
	for _, k := range keys {
		saved[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	defer func() {
		for k, v := range saved {
			if v != "" {
				os.Setenv(k, v)
			}
		}
	}()

	// Ollama only has "text" capability
	model := SelectByCapability([]string{"text"})
	require.NotNil(t, model)
	assert.Equal(t, "ollama", model.Provider)

	// No available model has "multilingual"
	model = SelectByCapability([]string{"multilingual"})
	assert.Nil(t, model)
}

func TestAvailableModels_AlwaysIncludesOllama(t *testing.T) {
	// Clear all keys
	keys := []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GOOGLE_API_KEY",
		"KIMI_API_KEY", "DEEPSEEK_API_KEY", "GROQ_API_KEY", "CEREBRAS_API_KEY"}
	saved := make(map[string]string)
	for _, k := range keys {
		saved[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	defer func() {
		for k, v := range saved {
			if v != "" {
				os.Setenv(k, v)
			}
		}
	}()

	available := AvailableModels()
	require.NotEmpty(t, available)

	hasOllama := false
	for _, m := range available {
		if m.Provider == "ollama" {
			hasOllama = true
			break
		}
	}
	assert.True(t, hasOllama, "ollama should always be available")
}

func TestDefaultCatalogizerStrategyConfig(t *testing.T) {
	cfg := DefaultCatalogizerStrategyConfig()
	assert.Equal(t, 0.35, cfg.QualityWeight)
	assert.Equal(t, 0.20, cfg.CostWeight)
	assert.Equal(t, 0.15, cfg.ContextWeight)
	assert.Equal(t, 0.20, cfg.ReliabilityWeight)
	assert.Equal(t, 0.10, cfg.CapabilityWeight)
	assert.Equal(t, 5*time.Minute, cfg.CacheTTL)

	// Weights should sum to 1.0
	total := cfg.QualityWeight + cfg.CostWeight + cfg.ContextWeight +
		cfg.ReliabilityWeight + cfg.CapabilityWeight
	assert.InDelta(t, 1.0, total, 0.001)
}
