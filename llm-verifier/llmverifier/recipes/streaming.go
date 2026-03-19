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

		// Send 3 consecutive requests and measure response consistency.
		const totalRequests = 3
		successCount := 0
		responses := make([]string, 0, totalRequests)
		var errors []string

		for i := 0; i < totalRequests; i++ {
			content, err := sendChat(ctx, c, modelName,
				"You are a helpful assistant.",
				fmt.Sprintf("Reply with a single sentence about the number %d.", i+1),
				100,
			)
			if err != nil {
				errors = append(errors, fmt.Sprintf("request_%d: %s", i+1, err.Error()))
				responses = append(responses, "")
				continue
			}

			trimmed := strings.TrimSpace(content)
			if len(trimmed) > 0 {
				successCount++
			}
			responses = append(responses, trimmed)
		}

		score := float64(successCount) / float64(totalRequests)
		passed := successCount == totalRequests

		return llmverifier.TestResult{
			TestID:    t.id,
			Category:  t.category,
			Passed:    passed,
			Score:     score,
			Timestamp: time.Now(),
			Details: map[string]any{
				"test":            "streaming_reliability",
				"model":           modelName,
				"total_requests":  totalRequests,
				"success_count":   successCount,
				"responses":       responses,
				"errors":          errors,
			},
		}, nil
	})
}
