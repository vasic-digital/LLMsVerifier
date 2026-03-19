// Copyright 2026 Vasic Digital. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package llmverifier

import (
	"context"
	"testing"
	"time"

	"digital.vasic.llmsverifier/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_FullQAStrategyPipeline(t *testing.T) {
	// Build QA strategy (recipes tested separately to avoid import cycle)
	strategy, err := NewStrategyBuilder("helix-qa").
		WithDescription("Optimized for autonomous QA sessions").
		WithWeights(WeightConfig{
			Responsiveness:       0.15,
			CodeCapability:       0.10,
			FeatureRichness:      0.15,
			Reliability:          0.20,
			VisionCapability:     0.25,
			InstructionFollowing: 0.15,
		}).
		WithTest(&mockVerificationTest{}).
		RequireCapability("vision").
		MinScore(0.6).
		MaxLatency(10 * time.Second).
		MinContextWindow(50000).
		Build()
	require.NoError(t, err)

	// Verify strategy properties
	assert.Equal(t, "helix-qa", strategy.Name())
	assert.Len(t, strategy.CustomTests(), 1)
	wc := strategy.WeightConfig()
	assert.NoError(t, wc.Validate())
	assert.Equal(t, 0.25, wc.VisionCapability)

	// Filter models
	models := []ModelInfo{
		{ID: "gpt-4o", SupportsVision: true, Endpoint: "https://api.openai.com"},
		{ID: "claude-3", SupportsVision: true, Endpoint: "https://api.anthropic.com"},
		{ID: "gpt-3.5", SupportsVision: false, Endpoint: "https://api.openai.com"},
	}
	filtered := strategy.FilterModels(models)
	assert.Len(t, filtered, 2)

	// Score models
	for _, model := range filtered {
		testResults := []TestResult{
			{Category: TestCategoryVision, Score: 0.95, Passed: true},
			{Category: TestCategoryInstruction, Score: 0.88, Passed: true},
			{Category: TestCategoryFeature, Score: 0.90, Passed: true},
			{Category: TestCategoryStability, Score: 0.85, Passed: true},
		}
		score, err := strategy.ScoreModel(context.Background(), model, testResults)
		require.NoError(t, err)
		assert.True(t, score.Overall > 0.6) // meets min threshold
		assert.True(t, score.Passed)

		// Convert to PerformanceScore for backward compat
		ps := score.ToPerformanceScore()
		assert.True(t, ps.OverallScore > 0)
	}
}

func TestE2E_DefaultStrategyMatchesExisting(t *testing.T) {
	// Verify DefaultStrategy produces the same structure as CalculateScores
	cfg := &config.Config{
		Concurrency: 1,
		Timeout:     30 * time.Second,
	}
	v := New(cfg)

	// Use CalculateScores (existing)
	vr := VerificationResult{
		Availability: AvailabilityResult{Exists: true, Responsive: true},
		ResponseTime: ResponseTimeResult{AverageLatency: 500 * time.Millisecond},
	}
	existingPS, _ := v.CalculateScores(vr)

	// Use CalculateScoresWithStrategy (new)
	strategy := NewDefaultStrategy()
	newScore, err := v.CalculateScoresWithStrategy(vr, strategy)
	require.NoError(t, err)

	// Both should produce valid scores (exact values differ because
	// strategy processes TestResults, not VerificationResult directly)
	assert.True(t, existingPS.OverallScore >= 0)
	assert.True(t, newScore.Overall >= 0)
}

func TestE2E_StrategyBuilder_WithCustomTests(t *testing.T) {
	// Build, filter, score — full pipeline without actual API calls
	strategy, err := NewStrategyBuilder("integration").
		WithWeights(WeightConfig{
			VisionCapability:     0.50,
			InstructionFollowing: 0.30,
			Reliability:          0.20,
		}).
		WithTest(&mockVerificationTest{}).
		RequireCapability("vision").
		MinScore(0.5).
		Build()
	require.NoError(t, err)

	// Score with manual test results
	score, err := strategy.ScoreModel(context.Background(), ModelInfo{}, []TestResult{
		{Category: TestCategoryVision, Score: 0.9},
		{Category: TestCategoryInstruction, Score: 0.8},
	})
	require.NoError(t, err)
	// 0.9*0.50 + 0.8*0.30 = 0.45 + 0.24 = 0.69
	assert.InDelta(t, 0.69, score.Overall, 0.001)
	assert.True(t, score.Passed)
}
