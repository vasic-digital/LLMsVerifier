// Copyright 2026 Vasic Digital. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package llmverifier

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Automation tests verify build correctness and interface compliance

func TestAutomation_AllStrategyTypes_ImplementInterface(t *testing.T) {
	// DefaultStrategy
	var _ ScoringStrategy = NewDefaultStrategy()

	// BuiltStrategy (via builder)
	s, err := NewStrategyBuilder("test").
		WithWeights(WeightConfig{Responsiveness: 1.0}).
		Build()
	assert.NoError(t, err)
	var _ ScoringStrategy = s
}

func TestAutomation_AllRecipeSteps_Composable(t *testing.T) {
	// Verify all recipe types can be combined in a single builder chain
	builder := NewStrategyBuilder("all-recipes").
		WithWeights(WeightConfig{
			Responsiveness:       0.20,
			CodeCapability:       0.20,
			FeatureRichness:      0.20,
			Reliability:          0.20,
			VisionCapability:     0.10,
			InstructionFollowing: 0.10,
		})

	// This must compile and not panic
	s, err := builder.Build()
	assert.NoError(t, err)
	assert.NotNil(t, s)
}

func TestAutomation_StrategyScore_AllFieldsAccessible(t *testing.T) {
	ss := StrategyScore{
		Overall:   0.85,
		Breakdown: map[string]float64{"test": 0.9},
		Passed:    true,
		Details:   map[string]any{"key": "value"},
	}
	assert.Equal(t, 0.85, ss.Overall)
	assert.Equal(t, 0.9, ss.Breakdown["test"])
	assert.True(t, ss.Passed)
	assert.Equal(t, "value", ss.Details["key"])
}

func TestAutomation_TestResult_AllFieldsAccessible(t *testing.T) {
	tr := TestResult{
		TestID:   "t1",
		Category: TestCategoryVision,
		Passed:   true,
		Score:    0.95,
	}
	assert.Equal(t, "t1", tr.TestID)
	assert.Equal(t, TestCategoryVision, tr.Category)
}

func TestAutomation_Thresholds_AllFieldsAccessible(t *testing.T) {
	th := Thresholds{
		MinOverallScore:      0.5,
		MinContextWindow:     50000,
		RequiredCapabilities: []string{"vision"},
	}
	assert.Equal(t, 0.5, th.MinOverallScore)
	assert.Equal(t, 50000, th.MinContextWindow)
}

func TestAutomation_WeightConfig_AllDimensions(t *testing.T) {
	wc := WeightConfig{
		Responsiveness:       0.1,
		CodeCapability:       0.1,
		FeatureRichness:      0.1,
		Reliability:          0.1,
		VisionCapability:     0.3,
		InstructionFollowing: 0.3,
	}
	assert.NoError(t, wc.Validate())
}

func TestAutomation_VerificationTest_InterfaceComplete(t *testing.T) {
	mock := &mockVerificationTest{}
	assert.Equal(t, "mock-test", mock.ID())
	assert.Equal(t, "Mock Test", mock.Name())
	assert.Equal(t, TestCategoryVision, mock.Category())
	assert.Equal(t, 1.0, mock.Weight())
	assert.False(t, mock.Required())

	result, err := mock.Run(context.Background(), nil)
	assert.NoError(t, err)
	assert.True(t, result.Passed)
}
