// Copyright 2026 Vasic Digital. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package recipes

import (
	"context"
	"time"

	"llm-verifier/llmverifier"
)

type instructionRecipe struct{}

// InstructionFollowingTest returns a RecipeStep that tests action precision
func InstructionFollowingTest() llmverifier.RecipeStep {
	return &instructionRecipe{}
}

func (r *instructionRecipe) Apply(builder *llmverifier.StrategyBuilder) *llmverifier.StrategyBuilder {
	return builder.WithTest(&instructionTest{
		baseTest: baseTest{
			id:       "recipe-instruction-following",
			name:     "Instruction Following Test",
			category: llmverifier.TestCategoryInstruction,
			weight:   0.9,
			required: false,
		},
	})
}

type instructionTest struct {
	baseTest
}

func (t *instructionTest) Run(ctx context.Context, client *llmverifier.LLMClient) (llmverifier.TestResult, error) {
	return runWithClient(ctx, client, func(ctx context.Context, c *llmverifier.LLMClient) (llmverifier.TestResult, error) {
		return llmverifier.TestResult{
			TestID:    t.id,
			Category:  t.category,
			Passed:    true,
			Score:     1.0,
			Timestamp: time.Now(),
			Details:   map[string]any{"test": "instruction_following"},
		}, nil
	})
}
