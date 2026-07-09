// Copyright 2026 Vasic Digital. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package llmverifier

import (
	"context"
	"fmt"
	"time"
)

// NewStrategyBuilder creates a new StrategyBuilder with the given name
func NewStrategyBuilder(name string) *StrategyBuilder {
	return &StrategyBuilder{name: name}
}

// WithDescription sets the strategy description
func (b *StrategyBuilder) WithDescription(desc string) *StrategyBuilder {
	b.description = desc
	return b
}

// WithWeights sets the scoring weights
func (b *StrategyBuilder) WithWeights(wc WeightConfig) *StrategyBuilder {
	b.weights = wc
	return b
}

// WithTest adds a custom verification test
func (b *StrategyBuilder) WithTest(test VerificationTest) *StrategyBuilder {
	b.tests = append(b.tests, test)
	return b
}

// WithRecipe applies a recipe step to the builder
func (b *StrategyBuilder) WithRecipe(recipe RecipeStep) *StrategyBuilder {
	return recipe.Apply(b)
}

// RequireCapability adds a required capability for model filtering
func (b *StrategyBuilder) RequireCapability(cap string) *StrategyBuilder {
	b.capabilities = append(b.capabilities, cap)
	return b
}

// MinScore sets the minimum overall score threshold
func (b *StrategyBuilder) MinScore(score float64) *StrategyBuilder {
	b.minScore = score
	return b
}

// MaxLatency sets the maximum acceptable latency
func (b *StrategyBuilder) MaxLatency(d time.Duration) *StrategyBuilder {
	b.maxLatency = d
	return b
}

// MinContextWindow sets the minimum context window size
func (b *StrategyBuilder) MinContextWindow(tokens int) *StrategyBuilder {
	b.minContext = tokens
	return b
}

// Build validates configuration and returns a ScoringStrategy
func (b *StrategyBuilder) Build() (ScoringStrategy, error) {
	if err := b.weights.Validate(); err != nil {
		return nil, fmt.Errorf("invalid weights: %w", err)
	}

	tests := make([]VerificationTest, len(b.tests))
	copy(tests, b.tests)

	caps := make([]string, len(b.capabilities))
	copy(caps, b.capabilities)

	return &builtStrategy{
		name:        b.name,
		description: b.description,
		weights:     b.weights,
		tests:       tests,
		thresholds: Thresholds{
			MinOverallScore:      b.minScore,
			MaxLatency:           b.maxLatency,
			MinContextWindow:     b.minContext,
			RequiredCapabilities: caps,
		},
	}, nil
}

// builtStrategy implements ScoringStrategy from builder configuration
type builtStrategy struct {
	name        string
	description string
	weights     WeightConfig
	tests       []VerificationTest
	thresholds  Thresholds
}

func (s *builtStrategy) Name() string                    { return s.name }
func (s *builtStrategy) Description() string             { return s.description }
func (s *builtStrategy) WeightConfig() WeightConfig      { return s.weights }
func (s *builtStrategy) CustomTests() []VerificationTest { return s.tests }
func (s *builtStrategy) MinimumThresholds() Thresholds   { return s.thresholds }

func (s *builtStrategy) FilterModels(models []ModelInfo) []ModelInfo {
	if len(s.thresholds.RequiredCapabilities) == 0 {
		return models
	}

	var filtered []ModelInfo
	for _, m := range models {
		if s.modelMeetsCapabilities(m) {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

func (s *builtStrategy) modelMeetsCapabilities(m ModelInfo) bool {
	for _, cap := range s.thresholds.RequiredCapabilities {
		switch cap {
		case "vision":
			if !m.SupportsVision {
				return false
			}
		case "audio":
			if !m.SupportsAudio {
				return false
			}
		case "video":
			if !m.SupportsVideo {
				return false
			}
		case "reasoning":
			if !m.SupportsReasoning {
				return false
			}
		case "streaming":
			// Streaming is generally available; check capabilities struct if needed
			continue
		default:
			// Unknown capability — check tags
			found := false
			for _, tag := range m.Tags {
				if tag == cap {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}

func (s *builtStrategy) ScoreModel(_ context.Context, _ ModelInfo, results []TestResult) (StrategyScore, error) {
	wc := s.weights

	// Map categories to weight dimensions
	dimensionMap := map[TestCategory]string{
		TestCategoryResponsiveness: "responsiveness",
		TestCategoryLatency:        "responsiveness",
		TestCategoryCode:           "code_capability",
		TestCategoryFeature:        "feature_richness",
		TestCategoryExistence:      "reliability",
		TestCategoryStability:      "reliability",
		TestCategoryVision:         "vision_capability",
		TestCategoryInstruction:    "instruction_following",
	}

	weightMap := map[string]float64{
		"responsiveness":        wc.Responsiveness,
		"code_capability":       wc.CodeCapability,
		"feature_richness":      wc.FeatureRichness,
		"reliability":           wc.Reliability,
		"vision_capability":     wc.VisionCapability,
		"instruction_following": wc.InstructionFollowing,
	}

	scores := make(map[string]float64)
	counts := make(map[string]int)

	for _, r := range results {
		dim, ok := dimensionMap[r.Category]
		if !ok {
			continue
		}
		scores[dim] += r.Score
		counts[dim]++
	}

	// Average per dimension
	for k, c := range counts {
		if c > 0 {
			scores[k] /= float64(c)
		}
	}

	// Weighted overall
	var overall float64
	for dim, weight := range weightMap {
		overall += scores[dim] * weight
	}

	passed := overall >= s.thresholds.MinOverallScore

	return StrategyScore{
		Overall:   overall,
		Breakdown: scores,
		Passed:    passed,
	}, nil
}
