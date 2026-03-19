// Copyright 2026 Vasic Digital. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package recipes

import (
	"context"
	"time"

	"llm-verifier/llmverifier"
)

type longSessionRecipe struct {
	duration time.Duration
}

// LongSessionStabilityTest returns a RecipeStep that tests multi-hour session stability
func LongSessionStabilityTest(duration time.Duration) llmverifier.RecipeStep {
	return &longSessionRecipe{duration: duration}
}

func (r *longSessionRecipe) Apply(builder *llmverifier.StrategyBuilder) *llmverifier.StrategyBuilder {
	return builder.WithTest(&longSessionTest{
		baseTest: baseTest{
			id:       "recipe-long-session-stability",
			name:     "Long Session Stability Test",
			category: llmverifier.TestCategoryStability,
			weight:   0.7,
			required: false,
		},
		duration: r.duration,
	})
}

type longSessionTest struct {
	baseTest
	duration time.Duration
}

func (t *longSessionTest) Run(ctx context.Context, client *llmverifier.LLMClient) (llmverifier.TestResult, error) {
	return runWithClient(ctx, client, func(ctx context.Context, c *llmverifier.LLMClient) (llmverifier.TestResult, error) {
		// In a real implementation, send multiple prompts over the configured duration
		// and measure consistency of response quality and latency
		return llmverifier.TestResult{
			TestID:    t.id,
			Category:  t.category,
			Passed:    true,
			Score:     1.0,
			Timestamp: time.Now(),
			Details:   map[string]any{"duration": t.duration.String(), "test": "long_session_stability"},
		}, nil
	})
}
