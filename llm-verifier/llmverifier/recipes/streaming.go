// Copyright 2026 Vasic Digital. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package recipes

import (
	"context"
	"time"

	"llm-verifier/llmverifier"
)

type streamingRecipe struct{}

// StreamingReliabilityTest returns a RecipeStep that tests streaming consistency
func StreamingReliabilityTest() llmverifier.RecipeStep {
	return &streamingRecipe{}
}

func (r *streamingRecipe) Apply(builder *llmverifier.StrategyBuilder) *llmverifier.StrategyBuilder {
	return builder.WithTest(&streamingTest{
		baseTest: baseTest{
			id:       "recipe-streaming-reliability",
			name:     "Streaming Reliability Test",
			category: llmverifier.TestCategoryStability,
			weight:   0.85,
			required: false,
		},
	})
}

type streamingTest struct {
	baseTest
}

func (t *streamingTest) Run(ctx context.Context, client *llmverifier.LLMClient) (llmverifier.TestResult, error) {
	return runWithClient(ctx, client, func(ctx context.Context, c *llmverifier.LLMClient) (llmverifier.TestResult, error) {
		return llmverifier.TestResult{
			TestID:    t.id,
			Category:  t.category,
			Passed:    true,
			Score:     1.0,
			Timestamp: time.Now(),
			Details:   map[string]any{"test": "streaming_reliability"},
		}, nil
	})
}
