// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package helixqa

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVisionModelRegistry_ContainsAstica(t *testing.T) {
	found := false
	for _, m := range VisionModelRegistry() {
		if m.Provider == "astica" {
			found = true
			assert.Equal(t, "astica-vision-25", m.ID)
			assert.Equal(t, "Astica Vision 2.5", m.Name)
			assert.Equal(t, "2.5_full", m.Model)
			assert.True(t, m.SupportsVision)
			assert.False(t, m.SupportsStreaming)
			assert.Equal(t, 0.97, m.QualityScore)
			assert.Equal(t, 0.95, m.ReliabilityScore)
			assert.Equal(t, 800, m.AvgLatencyMs)
			assert.Contains(t, m.Capabilities, "vision")
			assert.Contains(t, m.Capabilities, "ocr")
			assert.Contains(t, m.Capabilities, "object_detection")
			assert.Contains(t, m.Capabilities, "face_detection")
			assert.Contains(t, m.Capabilities, "content_moderation")
			assert.Contains(t, m.Capabilities, "gui_analysis")
			assert.NotNil(t, m.Metadata)
			assert.Equal(t, "comprehensive image understanding", m.Metadata["specialization"])
			break
		}
	}
	assert.True(t, found, "astica provider should be in registry")
}

func TestAstica_IsHighestQualityScore(t *testing.T) {
	models := VisionModelRegistry()
	var asticaScore float64
	for _, m := range models {
		if m.Provider == "astica" {
			asticaScore = m.QualityScore
			break
		}
	}
	require.Greater(t, asticaScore, 0.0, "astica must be in registry")

	for _, m := range models {
		if m.Provider == "astica" {
			continue
		}
		assert.GreaterOrEqual(t, asticaScore, m.QualityScore,
			"astica quality score (%.2f) must be >= %s quality score (%.2f)",
			asticaScore, m.ID, m.QualityScore)
	}
}

func TestAstica_InBudgetVisionModels(t *testing.T) {
	found := false
	for _, m := range BudgetVisionModels() {
		if m.Provider == "astica" {
			found = true
			break
		}
	}
	assert.True(t, found, "astica should be in budget models (low cost)")
}

func TestQAStrategy_ScoresAstica(t *testing.T) {
	ctx := context.Background()
	s := NewQAStrategy()

	for _, m := range VisionModelRegistry() {
		if m.Provider == "astica" {
			score, err := s.Score(ctx, m)
			require.NoError(t, err)
			assert.Greater(t, score.Overall, 0.0, "astica score should be positive")
			assert.LessOrEqual(t, score.Overall, 1.0, "astica score should be <= 1")
			assert.NotEmpty(t, score.Reasoning)
			return
		}
	}
	t.Fatal("astica not found in registry")
}

func TestQAStrategy_AsticaRanksFirst(t *testing.T) {
	ctx := context.Background()
	s := NewQAStrategy()

	ranked, err := s.Rank(ctx, VisionModelRegistry())
	require.NoError(t, err)
	require.NotEmpty(t, ranked)

	// Astica should be rank 1 (highest score among vision models).
	assert.Equal(t, "astica", ranked[0].Model.Provider,
		"astica should be the top-ranked vision model, got %s at rank 1",
		ranked[0].Model.Provider)
}
