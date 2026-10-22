// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package recipe

import (
	"testing"
	"time"

	"digital.vasic.llmsverifier/pkg/strategy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecipeBuilder_Build(t *testing.T) {
	s := strategy.NewDefaultStrategy()

	recipe, err := NewRecipeBuilder().
		WithName("test-recipe").
		WithDescription("Test recipe").
		WithStrategy(s).
		Build()

	require.NoError(t, err)
	assert.NotEmpty(t, recipe.ID)
	assert.Equal(t, "test-recipe", recipe.Name)
	assert.Equal(t, "default", recipe.StrategyName)
}

func TestRecipeBuilder_Build_MissingStrategy(t *testing.T) {
	_, err := NewRecipeBuilder().
		WithName("test-recipe").
		Build()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "strategy is required")
}

func TestRecipeBuilder_Build_MissingName(t *testing.T) {
	s := strategy.NewDefaultStrategy()

	_, err := NewRecipeBuilder().
		WithStrategy(s).
		Build()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestRecipeBuilder_WithWeights(t *testing.T) {
	s := strategy.NewDefaultStrategy()

	recipe, err := NewRecipeBuilder().
		WithName("weighted").
		WithStrategy(s).
		WithWeight(strategy.DimensionQuality, 0.5).
		WithWeight(strategy.DimensionSpeed, 0.3).
		WithWeight(strategy.DimensionCost, 0.2).
		Build()

	require.NoError(t, err)
	assert.Equal(t, 0.5, recipe.Weights[strategy.DimensionQuality])
	assert.Equal(t, 0.3, recipe.Weights[strategy.DimensionSpeed])
	assert.Equal(t, 0.2, recipe.Weights[strategy.DimensionCost])
}

func TestRecipeBuilder_WithConstraints(t *testing.T) {
	s := strategy.NewDefaultStrategy()

	recipe, err := NewRecipeBuilder().
		WithName("constrained").
		WithStrategy(s).
		WithMinQuality(0.8).
		WithMinReliability(0.95).
		WithMaxLatency(3000).
		WithVisionRequired(true).
		Build()

	require.NoError(t, err)
	assert.Len(t, recipe.Constraints, 4)
}

func TestRecipeBuilder_WithFallbacks(t *testing.T) {
	s := strategy.NewDefaultStrategy()

	recipe, err := NewRecipeBuilder().
		WithName("with-fallback").
		WithStrategy(s).
		WithFallback(strategy.FallbackRule{
			Name:                "fallback-to-speed",
			Condition:           "quality_below_0.8",
			AlternativeStrategy: "speed",
			Priority:            1,
		}).
		Build()

	require.NoError(t, err)
	assert.Len(t, recipe.Fallbacks, 1)
}

func TestRecipeBuilder_WithTimeout(t *testing.T) {
	s := strategy.NewDefaultStrategy()

	recipe, err := NewRecipeBuilder().
		WithName("timed").
		WithStrategy(s).
		WithTimeout(10 * time.Minute).
		Build()

	require.NoError(t, err)
	assert.Equal(t, 10*time.Minute, recipe.Timeout)
}

func TestRecipeBuilder_WithTags(t *testing.T) {
	s := strategy.NewDefaultStrategy()

	recipe, err := NewRecipeBuilder().
		WithName("tagged").
		WithStrategy(s).
		WithTag("production").
		WithTag("high-priority").
		WithTags([]string{"verified", "fast"}).
		Build()

	require.NoError(t, err)
	assert.Contains(t, recipe.Tags, "production")
	assert.Contains(t, recipe.Tags, "high-priority")
	assert.Contains(t, recipe.Tags, "verified")
	assert.Contains(t, recipe.Tags, "fast")
}

func TestRecipe_Validate(t *testing.T) {
	s := strategy.NewDefaultStrategy()

	recipe := &Recipe{
		Name:     "valid",
		Strategy: s,
		Timeout:  5 * time.Minute,
	}

	err := recipe.Validate()
	assert.NoError(t, err)
}

func TestRecipe_Validate_MissingName(t *testing.T) {
	recipe := &Recipe{
		Strategy: strategy.NewDefaultStrategy(),
		Timeout:  5 * time.Minute,
	}

	err := recipe.Validate()
	assert.Error(t, err)
}

func TestRecipe_Clone(t *testing.T) {
	s := strategy.NewDefaultStrategy()

	original, _ := NewRecipeBuilder().
		WithName("original").
		WithStrategy(s).
		WithWeight(strategy.DimensionQuality, 0.6).
		WithTag("test").
		Build()

	clone := original.Clone()

	assert.Equal(t, original.Name, clone.Name)
	assert.Equal(t, original.Weights[strategy.DimensionQuality], clone.Weights[strategy.DimensionQuality])
	assert.Contains(t, clone.Tags, "test")

	clone.Tags[0] = "modified"
	assert.NotEqual(t, original.Tags[0], clone.Tags[0])
}

func TestRecipe_Apply(t *testing.T) {
	s := strategy.NewDefaultStrategy()

	recipe, _ := NewRecipeBuilder().
		WithName("apply-test").
		WithStrategy(s).
		WithWeight(strategy.DimensionQuality, 0.7).
		WithMinQuality(0.8).
		Build()

	err := recipe.Apply(s)
	assert.NoError(t, err)
}

func TestRecipeBuilder_FluentChaining(t *testing.T) {
	s := strategy.NewDefaultStrategy()

	recipe := NewRecipeBuilder().
		WithName("fluent").
		WithDescription("Fluent API test").
		WithStrategy(s).
		WithWeights(map[string]float64{
			strategy.DimensionQuality: 0.4,
			strategy.DimensionSpeed:   0.3,
			strategy.DimensionCost:    0.3,
		}).
		WithMinQuality(0.8).
		WithMinReliability(0.9).
		WithMaxLatency(2000).
		WithVisionRequired(true).
		WithStreamingRequired(true).
		WithMinContext(32000).
		WithTimeout(10 * time.Minute).
		WithMaxRetries(5).
		WithCacheTTL(10 * time.Minute).
		WithTag("production").
		WithVersion("2.0.0").
		BuildOrPanic()

	assert.Equal(t, "fluent", recipe.Name)
	assert.Equal(t, "Fluent API test", recipe.Description)
	assert.Len(t, recipe.Constraints, 6)
	assert.Equal(t, 5, recipe.MaxRetries)
	assert.Equal(t, "2.0.0", recipe.Version)
}

func TestRecipeBuilder_InvalidWeight(t *testing.T) {
	s := strategy.NewDefaultStrategy()

	_, err := NewRecipeBuilder().
		WithName("invalid-weight").
		WithStrategy(s).
		WithWeight(strategy.DimensionQuality, 1.5).
		Build()

	assert.Error(t, err)
}

func TestRecipeBuilder_NilStrategy(t *testing.T) {
	_, err := NewRecipeBuilder().
		WithName("nil-strategy").
		WithStrategy(nil).
		Build()

	assert.Error(t, err)
}

func TestRecipeBuilder_InvalidTimeout(t *testing.T) {
	s := strategy.NewDefaultStrategy()

	_, err := NewRecipeBuilder().
		WithName("invalid-timeout").
		WithStrategy(s).
		WithTimeout(-1 * time.Minute).
		Build()

	assert.Error(t, err)
}

func TestRecipeBuilder_InvalidMaxRetries(t *testing.T) {
	s := strategy.NewDefaultStrategy()

	_, err := NewRecipeBuilder().
		WithName("invalid-retries").
		WithStrategy(s).
		WithMaxRetries(-5).
		Build()

	assert.Error(t, err)
}
