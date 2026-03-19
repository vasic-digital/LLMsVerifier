// Copyright 2026 Vasic Digital. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package recipes provides composable RecipeStep building blocks for StrategyBuilder.
// Each recipe adds a VerificationTest that evaluates a specific LLM capability.
package recipes

import (
	"context"
	"time"

	"llm-verifier/llmverifier"
)

// baseTest provides common fields for all recipe verification tests
type baseTest struct {
	id       string
	name     string
	category llmverifier.TestCategory
	weight   float64
	required bool
}

func (b *baseTest) ID() string                        { return b.id }
func (b *baseTest) Name() string                      { return b.name }
func (b *baseTest) Category() llmverifier.TestCategory { return b.category }
func (b *baseTest) Weight() float64                   { return b.weight }
func (b *baseTest) Required() bool                    { return b.required }

// runWithClient is a helper that checks for nil client before running
func runWithClient(ctx context.Context, client *llmverifier.LLMClient, fn func(context.Context, *llmverifier.LLMClient) (llmverifier.TestResult, error)) (llmverifier.TestResult, error) {
	if client == nil {
		return llmverifier.TestResult{
			Passed:    false,
			Score:     0,
			Timestamp: time.Now(),
			Details:   map[string]any{"error": "nil client"},
		}, nil
	}
	return fn(ctx, client)
}
