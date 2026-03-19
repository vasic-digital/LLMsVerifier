// Copyright 2026 Vasic Digital. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package recipes

import (
	"context"
	"strings"
	"time"

	"digital.vasic.llmsverifier/llmverifier"
)

// visionRecipe adds a vision capability test
type visionRecipe struct{}

// VisionTest returns a RecipeStep that tests screenshot analysis capability
func VisionTest() llmverifier.RecipeStep {
	return &visionRecipe{}
}

func (r *visionRecipe) Apply(builder *llmverifier.StrategyBuilder) *llmverifier.StrategyBuilder {
	return builder.WithTest(&visionTest{
		baseTest: baseTest{
			id:       "recipe-vision",
			name:     "Vision Capability Test",
			category: llmverifier.TestCategoryVision,
			weight:   1.0,
			required: true,
		},
	})
}

type visionTest struct {
	baseTest
}

func (t *visionTest) Run(ctx context.Context, client *llmverifier.LLMClient) (llmverifier.TestResult, error) {
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

		// Ask the model about its vision capabilities using a text-based check.
		// Since we may not have a real image to send, we probe whether the model
		// understands and claims vision/image analysis ability.
		content, err := sendChat(ctx, c, modelName,
			"You are a helpful assistant. Answer concisely.",
			"Can you analyze screenshots and describe UI elements in images? "+
				"Reply starting with YES or NO, then briefly explain your capabilities.",
			200,
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

		upper := strings.ToUpper(strings.TrimSpace(content))
		score := 0.0
		passed := false

		if strings.HasPrefix(upper, "YES") {
			// Model claims vision capability
			score = 1.0
			passed = true
		} else if strings.Contains(upper, "YES") || strings.Contains(upper, "IMAGE") || strings.Contains(upper, "VISION") || strings.Contains(upper, "SCREENSHOT") {
			// Model mentions vision-related terms but doesn't lead with YES
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
				"test":         "vision_capability",
				"model":        modelName,
				"response":     content,
				"response_len": len(content),
			},
		}, nil
	})
}
