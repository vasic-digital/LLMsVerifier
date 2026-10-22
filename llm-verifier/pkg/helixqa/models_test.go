// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package helixqa

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVisionModelRegistry_NotEmpty(t *testing.T) {
	models := VisionModelRegistry()
	assert.NotEmpty(t, models)
	assert.GreaterOrEqual(t, len(models), 7)
}

func TestVisionModelRegistry_AllHaveVision(t *testing.T) {
	for _, m := range VisionModelRegistry() {
		assert.True(t, m.SupportsVision, "model %s should support vision", m.ID)
	}
}

func TestVisionModelRegistry_AllHaveRequiredFields(t *testing.T) {
	for _, m := range VisionModelRegistry() {
		assert.NotEmpty(t, m.ID, "model ID must not be empty")
		assert.NotEmpty(t, m.Name, "model name must not be empty for %s", m.ID)
		assert.NotEmpty(t, m.Provider, "provider must not be empty for %s", m.ID)
		assert.NotEmpty(t, m.Model, "model identifier must not be empty for %s", m.ID)
		assert.Greater(t, m.ContextWindow, 0, "context window must be positive for %s", m.ID)
		assert.Greater(t, m.MaxOutputTokens, 0, "max output tokens must be positive for %s", m.ID)
		assert.Greater(t, m.QualityScore, 0.0, "quality score must be positive for %s", m.ID)
		assert.Greater(t, m.ReliabilityScore, 0.0, "reliability score must be positive for %s", m.ID)
	}
}

func TestVisionModelRegistry_ContainsKimi(t *testing.T) {
	found := false
	for _, m := range VisionModelRegistry() {
		if m.Provider == "kimi" {
			found = true
			assert.Equal(t, "kimi-k2.5", m.Model)
			assert.True(t, m.SupportsVision)
			assert.True(t, m.SupportsStreaming)
			assert.NotNil(t, m.Metadata)
			assert.Equal(t, "MIT", m.Metadata["license"])
			break
		}
	}
	assert.True(t, found, "kimi provider should be in registry")
}

func TestVisionModelRegistry_ContainsStepGUI(t *testing.T) {
	found := false
	for _, m := range VisionModelRegistry() {
		if m.Provider == "stepfun" {
			found = true
			assert.Equal(t, "step-1.5v-mini", m.Model)
			assert.True(t, m.SupportsVision)
			assert.Contains(t, m.Capabilities, "gui_grounding")
			break
		}
	}
	assert.True(t, found, "stepfun provider should be in registry")
}

func TestVisionModelRegistry_ContainsQwen(t *testing.T) {
	found := false
	for _, m := range VisionModelRegistry() {
		if m.Provider == "qwen" {
			found = true
			assert.True(t, m.SupportsVision)
			break
		}
	}
	assert.True(t, found, "qwen provider should be in registry")
}

func TestBudgetVisionModels_OnlyLowCost(t *testing.T) {
	budget := BudgetVisionModels()
	assert.NotEmpty(t, budget)
	for _, m := range budget {
		totalCost := m.InputCostPer1k + m.OutputCostPer1k
		assert.Less(t, totalCost, 0.002, "model %s should be budget (< $2/1M tokens)", m.ID)
	}
}

func TestBudgetVisionModels_ContainsKimi(t *testing.T) {
	found := false
	for _, m := range BudgetVisionModels() {
		if m.Provider == "kimi" {
			found = true
			break
		}
	}
	assert.True(t, found, "kimi should be in budget models")
}

func TestGUISpecializedModels_NotEmpty(t *testing.T) {
	gui := GUISpecializedModels()
	assert.NotEmpty(t, gui)
}

func TestGUISpecializedModels_ContainsStepGUI(t *testing.T) {
	found := false
	for _, m := range GUISpecializedModels() {
		if m.Provider == "stepfun" {
			found = true
			break
		}
	}
	assert.True(t, found, "stepfun should be in GUI specialized models")
}

func TestGUISpecializedModels_ContainsQwen(t *testing.T) {
	found := false
	for _, m := range GUISpecializedModels() {
		if m.Provider == "qwen" {
			found = true
			break
		}
	}
	assert.True(t, found, "qwen should be in GUI specialized models")
}

func TestQAStrategy_ScoresNewProviders(t *testing.T) {
	ctx := context.Background()
	s := NewQAStrategy()

	for _, m := range VisionModelRegistry() {
		score, err := s.Score(ctx, m)
		require.NoError(t, err, "scoring should not fail for %s", m.ID)
		assert.Greater(t, score.Overall, 0.0, "score should be positive for %s", m.ID)
		assert.LessOrEqual(t, score.Overall, 1.0, "score should be <= 1 for %s", m.ID)
		assert.NotEmpty(t, score.Reasoning, "reasoning should not be empty for %s", m.ID)
	}
}

func TestQAStrategy_RanksNewProviders(t *testing.T) {
	ctx := context.Background()
	s := NewQAStrategy()

	ranked, err := s.Rank(ctx, VisionModelRegistry())
	require.NoError(t, err)
	assert.Equal(t, len(VisionModelRegistry()), len(ranked))

	// All vision models should be tier1 or tier2
	for _, r := range ranked {
		assert.NotEqual(t, "", r.Tier, "tier should be set for %s", r.Model.ID)
	}
}
