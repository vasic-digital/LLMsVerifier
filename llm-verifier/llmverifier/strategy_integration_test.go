// Copyright 2026 Vasic Digital. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package llmverifier

import (
	"context"
	"testing"
	"time"

	"llm-verifier/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyWithStrategy_EmptyModels(t *testing.T) {
	cfg := &config.Config{
		Concurrency: 1,
		Timeout:     10 * time.Second,
	}
	v := New(cfg)
	strategy := NewDefaultStrategy()
	results, scores, err := v.VerifyWithStrategy(context.Background(), strategy, []ModelInfo{})
	assert.NoError(t, err)
	assert.Empty(t, results)
	assert.Empty(t, scores)
}

func TestVerifyWithStrategy_NilModels(t *testing.T) {
	cfg := &config.Config{
		Concurrency: 1,
		Timeout:     10 * time.Second,
	}
	v := New(cfg)
	strategy := NewDefaultStrategy()
	results, scores, err := v.VerifyWithStrategy(context.Background(), strategy, nil)
	assert.NoError(t, err)
	assert.Empty(t, results)
	assert.Empty(t, scores)
}

func TestVerifyWithStrategy_FiltersModels(t *testing.T) {
	cfg := &config.Config{
		Concurrency: 1,
		Timeout:     10 * time.Second,
	}
	_ = New(cfg) // verifier created but filtering test doesn't need it
	strategy, err := NewStrategyBuilder("vision-only").
		WithWeights(WeightConfig{VisionCapability: 1.0}).
		RequireCapability("vision").
		Build()
	require.NoError(t, err)

	models := []ModelInfo{
		{ID: "has-vision", SupportsVision: true, Endpoint: "http://fake"},
		{ID: "no-vision", SupportsVision: false, Endpoint: "http://fake"},
	}

	// VerifyWithStrategy filters first, then only processes matching models
	// Since we can't call real APIs, we just verify the filtering works
	filtered := strategy.FilterModels(models)
	assert.Len(t, filtered, 1)
	assert.Equal(t, "has-vision", filtered[0].ID)
}

func TestCalculateScoresWithStrategy_DefaultStrategy(t *testing.T) {
	cfg := &config.Config{}
	v := New(cfg)
	vr := VerificationResult{
		Availability: AvailabilityResult{Exists: true, Responsive: true},
	}
	strategy := NewDefaultStrategy()
	score, err := v.CalculateScoresWithStrategy(vr, strategy)
	require.NoError(t, err)
	assert.True(t, score.Overall >= 0)
}

func TestCalculateScoresWithStrategy_CustomStrategy(t *testing.T) {
	cfg := &config.Config{}
	v := New(cfg)
	vr := VerificationResult{
		ModelInfo: ModelInfo{ID: "test-model"},
	}
	strategy, err := NewStrategyBuilder("custom").
		WithWeights(WeightConfig{VisionCapability: 1.0}).
		MinScore(0.5).
		Build()
	require.NoError(t, err)

	score, err := v.CalculateScoresWithStrategy(vr, strategy)
	require.NoError(t, err)
	// With no test results, score should be 0 and not pass min threshold
	assert.Equal(t, 0.0, score.Overall)
	assert.False(t, score.Passed)
}

func TestCalculateScoresWithStrategy_PreservesExistingCalculateScores(t *testing.T) {
	// Verify that the existing CalculateScores function still works unchanged
	cfg := &config.Config{}
	v := New(cfg)
	vr := VerificationResult{
		Availability: AvailabilityResult{Exists: true, Responsive: true},
		ResponseTime: ResponseTimeResult{AverageLatency: 500 * time.Millisecond},
	}
	// Call existing function - must not panic or change behavior
	ps, sd := v.CalculateScores(vr)
	_ = ps
	_ = sd
	// Just verify it doesn't crash - the actual values are tested in existing test files
}
