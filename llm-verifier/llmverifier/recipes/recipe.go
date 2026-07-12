// Copyright 2026 Vasic Digital. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package recipes provides composable RecipeStep building blocks for StrategyBuilder.
// Each recipe adds a VerificationTest that evaluates a specific LLM capability.
package recipes

import (
	"context"
	"fmt"
	"time"

	"digital.vasic.llmsverifier/llmverifier"
)

// baseTest provides common fields for all recipe verification tests
type baseTest struct {
	id       string
	name     string
	category llmverifier.TestCategory
	weight   float64
	required bool
}

func (b *baseTest) ID() string                         { return b.id }
func (b *baseTest) Name() string                       { return b.name }
func (b *baseTest) Category() llmverifier.TestCategory { return b.category }
func (b *baseTest) Weight() float64                    { return b.weight }
func (b *baseTest) Required() bool                     { return b.required }

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

// discoverModelName retrieves the first available model name from the endpoint.
// This is used by recipes that need to send chat completion requests.
func discoverModelName(ctx context.Context, client *llmverifier.LLMClient) (string, error) {
	models, err := client.ListModels(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list models: %w", err)
	}
	if len(models) == 0 {
		return "", fmt.Errorf("no models available at endpoint")
	}
	return models[0].ID, nil
}

// sendChat sends a simple chat completion request and returns the response content.
// It discovers the model name automatically.
func sendChat(ctx context.Context, client *llmverifier.LLMClient, modelName, systemPrompt, userPrompt string, maxTokens int) (string, error) {
	req := llmverifier.ChatCompletionRequest{
		Model: modelName,
		Messages: []llmverifier.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   intPtr(maxTokens),
		Temperature: floatPtr(0.3),
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	return resp.Choices[0].Message.Content, nil
}

func intPtr(i int) *int           { return &i }
func floatPtr(f float64) *float64 { return &f }
