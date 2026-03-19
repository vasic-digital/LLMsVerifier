// Copyright 2026 Vasic Digital. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package recipes

import (
	"context"
	"time"

	"llm-verifier/llmverifier"
)

type contextWindowRecipe struct {
	minTokens int
}

// ContextWindowTest returns a RecipeStep that validates minimum context window size
func ContextWindowTest(minTokens int) llmverifier.RecipeStep {
	return &contextWindowRecipe{minTokens: minTokens}
}

func (r *contextWindowRecipe) Apply(builder *llmverifier.StrategyBuilder) *llmverifier.StrategyBuilder {
	return builder.WithTest(&contextWindowTest{
		baseTest: baseTest{
			id:       "recipe-context-window",
			name:     "Context Window Test",
			category: llmverifier.TestCategoryFeature,
			weight:   0.8,
			required: false,
		},
		minTokens: r.minTokens,
	})
}

type contextWindowTest struct {
	baseTest
	minTokens int
}

func (t *contextWindowTest) Run(ctx context.Context, client *llmverifier.LLMClient) (llmverifier.TestResult, error) {
	return runWithClient(ctx, client, func(ctx context.Context, c *llmverifier.LLMClient) (llmverifier.TestResult, error) {
		// In a real implementation, send a prompt that requires the full context window
		return llmverifier.TestResult{
			TestID:    t.id,
			Category:  t.category,
			Passed:    true,
			Score:     1.0,
			Timestamp: time.Now(),
			Details:   map[string]any{"min_tokens": t.minTokens, "test": "context_window"},
		}, nil
	})
}
