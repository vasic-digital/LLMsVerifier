// Copyright 2026 Vasic Digital. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package recipes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"digital.vasic.llmsverifier/llmverifier"
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

// maxContextTestChars caps the test prompt size to avoid excessive API costs
const maxContextTestChars = 10000

func (t *contextWindowTest) Run(ctx context.Context, client *llmverifier.LLMClient) (llmverifier.TestResult, error) {
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

		// Generate a prompt with roughly minTokens/4 characters (1 token ~ 4 chars).
		// Cap at maxContextTestChars to avoid excessive API costs.
		targetChars := t.minTokens / 4
		if targetChars > maxContextTestChars {
			targetChars = maxContextTestChars
		}
		if targetChars < 100 {
			targetChars = 100
		}

		// Build a long context with a unique keyword planted at the beginning
		const secretWord = "ELEPHANTINE"
		filler := strings.Repeat("The quick brown fox jumps over the lazy dog. ", targetChars/46+1)
		if len(filler) > targetChars-200 {
			filler = filler[:targetChars-200]
		}

		prompt := fmt.Sprintf(
			"Remember the word: %s\n\n%s\n\nWhat was the word I asked you to remember at the very beginning of this message? Reply with just that single word.",
			secretWord, filler,
		)

		content, err := sendChat(ctx, c, modelName,
			"You have excellent memory. Answer precisely.",
			prompt,
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
		upper := strings.ToUpper(trimmed)
		score := 0.0
		passed := false

		if strings.Contains(upper, secretWord) {
			score = 1.0
			passed = true
		} else {
			// Proportional score: give partial credit if the model responded
			// coherently but couldn't recall the exact word
			if len(trimmed) > 0 {
				score = 0.25
			}
		}

		return llmverifier.TestResult{
			TestID:    t.id,
			Category:  t.category,
			Passed:    passed,
			Score:     score,
			Timestamp: time.Now(),
			Details: map[string]any{
				"test":          "context_window",
				"model":         modelName,
				"min_tokens":    t.minTokens,
				"prompt_chars":  len(prompt),
				"secret_word":   secretWord,
				"response":      trimmed,
				"word_recalled": strings.Contains(upper, secretWord),
			},
		}, nil
	})
}
