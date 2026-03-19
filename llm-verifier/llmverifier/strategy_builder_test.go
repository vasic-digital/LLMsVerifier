// Copyright 2026 Vasic Digital. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package llmverifier

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStrategyBuilder_Build_MinimalValid(t *testing.T) {
	s, err := NewStrategyBuilder("test").
		WithDescription("Test strategy").
		WithWeights(WeightConfig{
			Responsiveness: 0.5,
			Reliability:    0.5,
		}).
		Build()
	require.NoError(t, err)
	assert.Equal(t, "test", s.Name())
	assert.Equal(t, "Test strategy", s.Description())
}

func TestStrategyBuilder_Build_InvalidWeights(t *testing.T) {
	_, err := NewStrategyBuilder("bad").
		WithWeights(WeightConfig{Responsiveness: 0.5}).
		Build()
	assert.Error(t, err)
}

func TestStrategyBuilder_Build_NoWeights(t *testing.T) {
	_, err := NewStrategyBuilder("bad").Build()
	assert.Error(t, err)
}

func TestStrategyBuilder_WithMinScore(t *testing.T) {
	s, err := NewStrategyBuilder("test").
		WithWeights(WeightConfig{Responsiveness: 1.0}).
		MinScore(0.7).
		Build()
	require.NoError(t, err)
	th := s.MinimumThresholds()
	assert.Equal(t, 0.7, th.MinOverallScore)
}

func TestStrategyBuilder_WithMaxLatency(t *testing.T) {
	s, err := NewStrategyBuilder("test").
		WithWeights(WeightConfig{Responsiveness: 1.0}).
		MaxLatency(5 * time.Second).
		Build()
	require.NoError(t, err)
	assert.Equal(t, 5*time.Second, s.MinimumThresholds().MaxLatency)
}

func TestStrategyBuilder_WithMinContextWindow(t *testing.T) {
	s, err := NewStrategyBuilder("test").
		WithWeights(WeightConfig{Responsiveness: 1.0}).
		MinContextWindow(100000).
		Build()
	require.NoError(t, err)
	assert.Equal(t, 100000, s.MinimumThresholds().MinContextWindow)
}

func TestStrategyBuilder_RequireCapability(t *testing.T) {
	s, err := NewStrategyBuilder("test").
		WithWeights(WeightConfig{Responsiveness: 1.0}).
		RequireCapability("vision").
		RequireCapability("streaming").
		Build()
	require.NoError(t, err)
	th := s.MinimumThresholds()
	assert.Contains(t, th.RequiredCapabilities, "vision")
	assert.Contains(t, th.RequiredCapabilities, "streaming")
}

func TestStrategyBuilder_FilterModels_ByVisionCapability(t *testing.T) {
	s, err := NewStrategyBuilder("test").
		WithWeights(WeightConfig{Responsiveness: 1.0}).
		RequireCapability("vision").
		Build()
	require.NoError(t, err)
	models := []ModelInfo{
		{ID: "m1", SupportsVision: true},
		{ID: "m2", SupportsVision: false},
		{ID: "m3", SupportsVision: true},
	}
	filtered := s.FilterModels(models)
	assert.Len(t, filtered, 2)
	assert.Equal(t, "m1", filtered[0].ID)
	assert.Equal(t, "m3", filtered[1].ID)
}

func TestStrategyBuilder_FilterModels_NoRequirements(t *testing.T) {
	s, err := NewStrategyBuilder("test").
		WithWeights(WeightConfig{Responsiveness: 1.0}).
		Build()
	require.NoError(t, err)
	models := []ModelInfo{{ID: "m1"}, {ID: "m2"}}
	filtered := s.FilterModels(models)
	assert.Len(t, filtered, 2)
}

func TestStrategyBuilder_WithTest(t *testing.T) {
	mock := &mockVerificationTest{}
	s, err := NewStrategyBuilder("test").
		WithWeights(WeightConfig{Responsiveness: 1.0}).
		WithTest(mock).
		Build()
	require.NoError(t, err)
	tests := s.CustomTests()
	assert.Len(t, tests, 1)
	assert.Equal(t, "mock-test", tests[0].ID())
}

func TestStrategyBuilder_WithRecipe(t *testing.T) {
	recipe := &mockRecipe{testToAdd: &mockVerificationTest{}}
	s, err := NewStrategyBuilder("test").
		WithWeights(WeightConfig{Responsiveness: 1.0}).
		WithRecipe(recipe).
		Build()
	require.NoError(t, err)
	assert.Len(t, s.CustomTests(), 1)
}

func TestStrategyBuilder_ScoreModel_UsesWeights(t *testing.T) {
	s, err := NewStrategyBuilder("qa").
		WithWeights(WeightConfig{
			VisionCapability:     0.60,
			InstructionFollowing: 0.40,
		}).
		Build()
	require.NoError(t, err)
	results := []TestResult{
		{Category: TestCategoryVision, Score: 1.0},
		{Category: TestCategoryInstruction, Score: 0.5},
	}
	score, err := s.ScoreModel(context.Background(), ModelInfo{}, results)
	require.NoError(t, err)
	// 1.0*0.60 + 0.5*0.40 = 0.80
	assert.InDelta(t, 0.80, score.Overall, 0.001)
}

func TestStrategyBuilder_ScoreModel_MinScoreFilter(t *testing.T) {
	s, err := NewStrategyBuilder("strict").
		WithWeights(WeightConfig{Responsiveness: 1.0}).
		MinScore(0.8).
		Build()
	require.NoError(t, err)
	results := []TestResult{
		{Category: TestCategoryResponsiveness, Score: 0.5},
	}
	score, err := s.ScoreModel(context.Background(), ModelInfo{}, results)
	require.NoError(t, err)
	assert.False(t, score.Passed) // 0.5 < 0.8 threshold
}

func TestStrategyBuilder_Fluent_Chaining(t *testing.T) {
	s, err := NewStrategyBuilder("full").
		WithDescription("Full QA strategy").
		WithWeights(WeightConfig{
			Responsiveness:       0.15,
			CodeCapability:       0.10,
			FeatureRichness:      0.15,
			Reliability:          0.20,
			VisionCapability:     0.25,
			InstructionFollowing: 0.15,
		}).
		RequireCapability("vision").
		RequireCapability("streaming").
		MinScore(0.6).
		MaxLatency(10 * time.Second).
		MinContextWindow(50000).
		Build()
	require.NoError(t, err)
	assert.Equal(t, "full", s.Name())
	assert.Equal(t, "Full QA strategy", s.Description())
	th := s.MinimumThresholds()
	assert.Equal(t, 0.6, th.MinOverallScore)
	assert.Equal(t, 10*time.Second, th.MaxLatency)
	assert.Equal(t, 50000, th.MinContextWindow)
	assert.Len(t, th.RequiredCapabilities, 2)
}

// mockRecipe for testing WithRecipe
type mockRecipe struct {
	testToAdd VerificationTest
}

func (m *mockRecipe) Apply(builder *StrategyBuilder) *StrategyBuilder {
	return builder.WithTest(m.testToAdd)
}
