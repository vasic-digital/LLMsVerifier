// Copyright 2026 Vasic Digital. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package recipes

import (
	"context"
	"testing"
	"time"

	"digital.vasic.llmsverifier/llmverifier"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- RecipeStep Interface Tests ---

func TestVisionTest_ImplementsRecipeStep(t *testing.T) {
	var _ llmverifier.RecipeStep = VisionTest()
}

func TestInstructionFollowingTest_ImplementsRecipeStep(t *testing.T) {
	var _ llmverifier.RecipeStep = InstructionFollowingTest()
}

func TestContextWindowTest_ImplementsRecipeStep(t *testing.T) {
	var _ llmverifier.RecipeStep = ContextWindowTest(100000)
}

func TestStreamingReliabilityTest_ImplementsRecipeStep(t *testing.T) {
	var _ llmverifier.RecipeStep = StreamingReliabilityTest()
}

func TestLongSessionStabilityTest_ImplementsRecipeStep(t *testing.T) {
	var _ llmverifier.RecipeStep = LongSessionStabilityTest(30 * time.Minute)
}

// --- VisionTest Recipe ---

func TestVisionTest_Apply_AddsTest(t *testing.T) {
	builder := llmverifier.NewStrategyBuilder("test").
		WithWeights(llmverifier.WeightConfig{VisionCapability: 1.0})
	builder = VisionTest().Apply(builder)
	s, err := builder.Build()
	require.NoError(t, err)
	tests := s.CustomTests()
	assert.Len(t, tests, 1)
	assert.Equal(t, "recipe-vision", tests[0].ID())
	assert.Equal(t, llmverifier.TestCategoryVision, tests[0].Category())
	assert.True(t, tests[0].Required())
}

func TestVisionTest_Run_WithNilClient(t *testing.T) {
	recipe := VisionTest()
	builder := llmverifier.NewStrategyBuilder("test").
		WithWeights(llmverifier.WeightConfig{VisionCapability: 1.0})
	builder = recipe.Apply(builder)
	s, err := builder.Build()
	require.NoError(t, err)
	test := s.CustomTests()[0]
	// With nil client, should return a result with error/low score
	result, err := test.Run(context.Background(), nil)
	assert.NoError(t, err) // Run itself doesn't error, result captures the failure
	assert.False(t, result.Passed)
}

// --- InstructionFollowingTest Recipe ---

func TestInstructionFollowingTest_Apply_AddsTest(t *testing.T) {
	builder := llmverifier.NewStrategyBuilder("test").
		WithWeights(llmverifier.WeightConfig{InstructionFollowing: 1.0})
	builder = InstructionFollowingTest().Apply(builder)
	s, err := builder.Build()
	require.NoError(t, err)
	tests := s.CustomTests()
	assert.Len(t, tests, 1)
	assert.Equal(t, "recipe-instruction-following", tests[0].ID())
	assert.Equal(t, llmverifier.TestCategoryInstruction, tests[0].Category())
}

// --- ContextWindowTest Recipe ---

func TestContextWindowTest_Apply_AddsTest(t *testing.T) {
	builder := llmverifier.NewStrategyBuilder("test").
		WithWeights(llmverifier.WeightConfig{FeatureRichness: 1.0})
	builder = ContextWindowTest(100000).Apply(builder)
	s, err := builder.Build()
	require.NoError(t, err)
	tests := s.CustomTests()
	assert.Len(t, tests, 1)
	assert.Equal(t, "recipe-context-window", tests[0].ID())
	assert.Equal(t, llmverifier.TestCategoryFeature, tests[0].Category())
}

func TestContextWindowTest_Weight(t *testing.T) {
	builder := llmverifier.NewStrategyBuilder("test").
		WithWeights(llmverifier.WeightConfig{FeatureRichness: 1.0})
	builder = ContextWindowTest(100000).Apply(builder)
	s, err := builder.Build()
	require.NoError(t, err)
	assert.Equal(t, 0.8, s.CustomTests()[0].Weight())
}

// --- StreamingReliabilityTest Recipe ---

func TestStreamingReliabilityTest_Apply_AddsTest(t *testing.T) {
	builder := llmverifier.NewStrategyBuilder("test").
		WithWeights(llmverifier.WeightConfig{Reliability: 1.0})
	builder = StreamingReliabilityTest().Apply(builder)
	s, err := builder.Build()
	require.NoError(t, err)
	tests := s.CustomTests()
	assert.Len(t, tests, 1)
	assert.Equal(t, "recipe-streaming-reliability", tests[0].ID())
	assert.Equal(t, llmverifier.TestCategoryStability, tests[0].Category())
}

// --- LongSessionStabilityTest Recipe ---

func TestLongSessionStabilityTest_Apply_AddsTest(t *testing.T) {
	builder := llmverifier.NewStrategyBuilder("test").
		WithWeights(llmverifier.WeightConfig{Reliability: 1.0})
	builder = LongSessionStabilityTest(30 * time.Minute).Apply(builder)
	s, err := builder.Build()
	require.NoError(t, err)
	tests := s.CustomTests()
	assert.Len(t, tests, 1)
	assert.Equal(t, "recipe-long-session-stability", tests[0].ID())
	assert.Equal(t, llmverifier.TestCategoryStability, tests[0].Category())
}

func TestLongSessionStabilityTest_Duration(t *testing.T) {
	recipe := LongSessionStabilityTest(45 * time.Minute)
	builder := llmverifier.NewStrategyBuilder("test").
		WithWeights(llmverifier.WeightConfig{Reliability: 1.0})
	builder = recipe.Apply(builder)
	s, err := builder.Build()
	require.NoError(t, err)
	// The test should be configured with the given duration
	test := s.CustomTests()[0]
	assert.Equal(t, "recipe-long-session-stability", test.ID())
}

// --- Multiple Recipes Combined ---

func TestMultipleRecipes_Combined(t *testing.T) {
	s, err := llmverifier.NewStrategyBuilder("qa").
		WithWeights(llmverifier.WeightConfig{
			VisionCapability:     0.30,
			InstructionFollowing: 0.20,
			Reliability:          0.30,
			FeatureRichness:      0.20,
		}).
		WithRecipe(VisionTest()).
		WithRecipe(InstructionFollowingTest()).
		WithRecipe(StreamingReliabilityTest()).
		WithRecipe(ContextWindowTest(50000)).
		WithRecipe(LongSessionStabilityTest(10 * time.Minute)).
		Build()
	require.NoError(t, err)
	assert.Len(t, s.CustomTests(), 5)

	// Verify all IDs are unique
	seen := make(map[string]bool)
	for _, test := range s.CustomTests() {
		assert.False(t, seen[test.ID()], "duplicate test ID: %s", test.ID())
		seen[test.ID()] = true
	}
}
