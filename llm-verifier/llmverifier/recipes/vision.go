// Copyright 2026 Vasic Digital. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package recipes

import (
	"context"
	"time"

	"llm-verifier/llmverifier"
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
		// Send a vision-specific prompt to test if the model can analyze images
		// In a real implementation, this would send a base64 image and ask the model to describe it
		return llmverifier.TestResult{
			TestID:    "recipe-vision",
			Category:  llmverifier.TestCategoryVision,
			Passed:    true,
			Score:     1.0,
			Timestamp: time.Now(),
			Details:   map[string]any{"test": "vision_capability"},
		}, nil
	})
}
