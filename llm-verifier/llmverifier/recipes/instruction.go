// Copyright 2026 Vasic Digital. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package recipes

import (
	"context"
	"strings"
	"time"

	"digital.vasic.llmsverifier/llmverifier"
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
		modelName, err := discoverModelName(ctx, c)
		if err != nil {
			return llmverifier.TestResult{
				TestID:    t.id,
				Category:  t.category,
				Passed:    false,
				Score:     0,
				Timestamp: time.Now(),
				Details:   map[string]any{"error": err.Error(), "phase": "model_discovery"},
			}, nil
		}

		const expectedPhrase = "INSTRUCTION_TEST_PASSED"

		content, err := sendChat(ctx, c, modelName,
			"You are a precise instruction-following assistant. Follow instructions exactly.",
			"Reply with exactly the text: INSTRUCTION_TEST_PASSED",
			50,
		)
		if err != nil {
			return llmverifier.TestResult{
				TestID:    t.id,
				Category:  t.category,
				Passed:    false,
				Score:     0,
				Timestamp: time.Now(),
				Details:   map[string]any{"error": err.Error(), "phase": "chat_completion"},
			}, nil
		}

		trimmed := strings.TrimSpace(content)
		score := 0.0
		passed := false

		if trimmed == expectedPhrase {
			// Exact match
			score = 1.0
			passed = true
		} else if strings.Contains(trimmed, expectedPhrase) {
			// Contains the phrase but has extra text
			score = 0.5
			passed = false
		}
		// Otherwise score stays 0.0

		return llmverifier.TestResult{
			TestID:    t.id,
			Category:  t.category,
			Passed:    passed,
			Score:     score,
			Timestamp: time.Now(),
			Details: map[string]any{
				"test":           "instruction_following",
				"model":          modelName,
				"expected":       expectedPhrase,
				"actual":         trimmed,
				"exact_match":    trimmed == expectedPhrase,
				"contains_match": strings.Contains(trimmed, expectedPhrase),
			},
		}, nil
	})
}
