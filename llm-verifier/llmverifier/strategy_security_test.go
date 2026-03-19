// Copyright 2026 Vasic Digital. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package llmverifier

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecurity_WeightConfig_NegativeWeightsRejected(t *testing.T) {
	wc := WeightConfig{Responsiveness: -1.0, Reliability: 2.0}
	assert.Error(t, wc.Validate())
}

func TestSecurity_WeightConfig_ExtremeValues(t *testing.T) {
	wc := WeightConfig{Responsiveness: 1e308}
	assert.Error(t, wc.Validate())
}

func TestSecurity_StrategyBuilder_EmptyName(t *testing.T) {
	s, err := NewStrategyBuilder("").
		WithWeights(WeightConfig{Responsiveness: 1.0}).
		Build()
	// Empty name is allowed (no security issue) but we verify it works
	assert.NoError(t, err)
	assert.Equal(t, "", s.Name())
}

func TestSecurity_StrategyBuilder_LongName(t *testing.T) {
	longName := strings.Repeat("a", 10000)
	s, err := NewStrategyBuilder(longName).
		WithWeights(WeightConfig{Responsiveness: 1.0}).
		Build()
	assert.NoError(t, err)
	assert.Len(t, s.Name(), 10000)
}

func TestSecurity_StrategyBuilder_SpecialCharactersInCapability(t *testing.T) {
	s, err := NewStrategyBuilder("test").
		WithWeights(WeightConfig{Responsiveness: 1.0}).
		RequireCapability("'; DROP TABLE models; --").
		Build()
	assert.NoError(t, err)
	// Capability filtering is string matching, not SQL — no injection risk
	models := []ModelInfo{{ID: "m1", Tags: []string{"safe"}}}
	filtered := s.FilterModels(models)
	assert.Empty(t, filtered) // model doesn't have the capability
}

func TestSecurity_StrategyScore_NilBreakdown(t *testing.T) {
	ss := StrategyScore{Overall: 0.5, Breakdown: nil}
	ps := ss.ToPerformanceScore()
	assert.Equal(t, 0.5, ps.OverallScore)
	assert.Equal(t, 0.0, ps.Responsiveness) // nil map returns zero value
}

func TestSecurity_Thresholds_NegativeValues(t *testing.T) {
	th := Thresholds{
		MinOverallScore:  -1.0,
		MinContextWindow: -100,
	}
	// Negative thresholds should effectively disable the check
	assert.True(t, th.Check(0.0, 0, 0, nil))
}

func TestSecurity_FilterModels_NilSlice(t *testing.T) {
	ds := NewDefaultStrategy()
	filtered := ds.FilterModels(nil)
	assert.Nil(t, filtered)
}

func TestSecurity_ScoreModel_VeryLargeResultSet(t *testing.T) {
	ds := NewDefaultStrategy()
	var results []TestResult
	for i := 0; i < 10000; i++ {
		results = append(results, TestResult{Category: TestCategoryResponsiveness, Score: 0.5})
	}
	score, err := ds.ScoreModel(nil, ModelInfo{}, results)
	assert.NoError(t, err)
	assert.InDelta(t, 0.5*0.30, score.Overall, 0.001)
}
