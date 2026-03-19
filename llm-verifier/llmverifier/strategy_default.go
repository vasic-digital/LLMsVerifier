// Copyright 2026 Vasic Digital. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package llmverifier

import "context"

// defaultStrategy wraps the existing scoring behavior.
// Produces identical results to the hardcoded CalculateScores weights:
// Responsiveness=0.30, CodeCapability=0.25, FeatureRichness=0.25, Reliability=0.20
type defaultStrategy struct{}

// NewDefaultStrategy creates the default scoring strategy
func NewDefaultStrategy() ScoringStrategy {
	return &defaultStrategy{}
}

func (ds *defaultStrategy) Name() string {
	return "default"
}

func (ds *defaultStrategy) Description() string {
	return "Default scoring strategy matching existing CalculateScores behavior"
}

func (ds *defaultStrategy) WeightConfig() WeightConfig {
	return WeightConfig{
		Responsiveness:       0.30,
		CodeCapability:       0.25,
		FeatureRichness:      0.25,
		Reliability:          0.20,
		VisionCapability:     0.00,
		InstructionFollowing: 0.00,
	}
}

func (ds *defaultStrategy) CustomTests() []VerificationTest {
	return nil
}

func (ds *defaultStrategy) FilterModels(models []ModelInfo) []ModelInfo {
	return models
}

func (ds *defaultStrategy) MinimumThresholds() Thresholds {
	return Thresholds{}
}

func (ds *defaultStrategy) ScoreModel(_ context.Context, _ ModelInfo, results []TestResult) (StrategyScore, error) {
	wc := ds.WeightConfig()
	scores := map[string]float64{
		"responsiveness":   0,
		"code_capability":  0,
		"feature_richness": 0,
		"reliability":      0,
	}
	counts := map[string]int{}

	for _, r := range results {
		switch r.Category {
		case TestCategoryResponsiveness, TestCategoryLatency:
			scores["responsiveness"] += r.Score
			counts["responsiveness"]++
		case TestCategoryCode:
			scores["code_capability"] += r.Score
			counts["code_capability"]++
		case TestCategoryFeature:
			scores["feature_richness"] += r.Score
			counts["feature_richness"]++
		case TestCategoryExistence, TestCategoryStability:
			scores["reliability"] += r.Score
			counts["reliability"]++
		}
	}

	// Average per dimension
	for k, c := range counts {
		if c > 0 {
			scores[k] /= float64(c)
		}
	}

	overall := scores["responsiveness"]*wc.Responsiveness +
		scores["code_capability"]*wc.CodeCapability +
		scores["feature_richness"]*wc.FeatureRichness +
		scores["reliability"]*wc.Reliability

	return StrategyScore{
		Overall:   overall,
		Breakdown: scores,
		Passed:    true,
	}, nil
}
