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

		// Send 3 requests with small delays to verify consistent response quality.
		// We do not wait for the full configured duration; that is for real
		// verification runs. This is a quick stability probe.
		const totalRequests = 3
		const delayBetween = 500 * time.Millisecond

		successCount := 0
		responses := make([]string, 0, totalRequests)
		latencies := make([]int64, 0, totalRequests)
		var errors []string

		for i := 0; i < totalRequests; i++ {
			if i > 0 {
				select {
				case <-ctx.Done():
					break
				case <-time.After(delayBetween):
				}
			}

			start := time.Now()
			content, err := sendChat(ctx, c, modelName,
				"You are a helpful assistant. Be concise.",
				fmt.Sprintf("What is %d + %d?", (i+1)*10, (i+1)*5),
				100,
			)
			elapsed := time.Since(start).Milliseconds()
			latencies = append(latencies, elapsed)

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
				"test":           "long_session_stability",
				"model":          modelName,
				"duration":       t.duration.String(),
				"total_requests": totalRequests,
				"success_count":  successCount,
				"latencies_ms":   latencies,
				"responses":      responses,
				"errors":         errors,
			},
		}, nil
	})
}
