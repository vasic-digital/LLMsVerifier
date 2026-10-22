// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package selector

import (
	"testing"

	"digital.vasic.llmsverifier/pkg/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noEnv is an EnvChecker that always returns false (no API keys).
func noEnv(key string) bool { return false }

// allEnv is an EnvChecker that always returns true (all keys set).
func allEnv(key string) bool { return true }

// selectiveEnv returns an EnvChecker that returns true only for
// the specified keys.
func selectiveEnv(keys ...string) EnvChecker {
	set := make(map[string]bool)
	for _, k := range keys {
		set[k] = true
	}
	return func(key string) bool {
		return set[key]
	}
}

// --- ModelType constants ---

func TestModelType_Values(t *testing.T) {
	assert.Equal(t, ModelType("vision"), ModelTypeVision)
	assert.Equal(t, ModelType("chat"), ModelTypeChat)
}

// --- SelectAll tests ---

func TestSelectAll_Vision_NoKeys(t *testing.T) {
	candidates := SelectAll(ModelTypeVision, noEnv)
	assert.NotEmpty(t, candidates, "should return candidates even without keys")

	for _, c := range candidates {
		assert.Equal(t, ModelTypeVision, c.Type)
	}

	// Ollama should always be available.
	hasOllama := false
	for _, c := range candidates {
		if c.Provider == "ollama" {
			hasOllama = true
			assert.True(t, c.Available, "ollama should always be available")
		}
	}
	assert.True(t, hasOllama, "ollama should be in vision candidates")
}

func TestSelectAll_Chat_NoKeys(t *testing.T) {
	candidates := SelectAll(ModelTypeChat, noEnv)
	assert.NotEmpty(t, candidates)

	for _, c := range candidates {
		assert.Equal(t, ModelTypeChat, c.Type)
	}

	// Ollama should always be available.
	hasOllama := false
	for _, c := range candidates {
		if c.Provider == "ollama" {
			hasOllama = true
			assert.True(t, c.Available)
		}
	}
	assert.True(t, hasOllama)
}

func TestSelectAll_Vision_AllKeys(t *testing.T) {
	candidates := SelectAll(ModelTypeVision, allEnv)
	assert.NotEmpty(t, candidates)

	// With all keys set, every candidate should be available.
	for _, c := range candidates {
		assert.True(t, c.Available,
			"provider %s should be available when all keys are set", c.Provider)
	}
}

func TestSelectAll_Chat_AllKeys(t *testing.T) {
	candidates := SelectAll(ModelTypeChat, allEnv)
	assert.NotEmpty(t, candidates)

	availableCount := 0
	for _, c := range candidates {
		if c.Available {
			availableCount++
		}
	}
	assert.Equal(t, len(candidates), availableCount)
}

func TestSelectAll_SortedByScore(t *testing.T) {
	candidates := SelectAll(ModelTypeVision, allEnv)
	require.NotEmpty(t, candidates)

	// All are available (allEnv). Within the available group,
	// candidates should be sorted by score descending.
	for i := 1; i < len(candidates); i++ {
		if candidates[i-1].Available == candidates[i].Available {
			assert.GreaterOrEqual(t, candidates[i-1].Score, candidates[i].Score,
				"candidates should be sorted by score descending: %s (%.3f) before %s (%.3f)",
				candidates[i-1].ID, candidates[i-1].Score,
				candidates[i].ID, candidates[i].Score)
		}
	}
}

func TestSelectAll_AvailableBeforeUnavailable(t *testing.T) {
	// Only set Anthropic key — some providers available, some not.
	checker := selectiveEnv("ANTHROPIC_API_KEY")
	candidates := SelectAll(ModelTypeVision, checker)

	// Find transition point from available to unavailable.
	seenUnavailable := false
	for _, c := range candidates {
		if !c.Available {
			seenUnavailable = true
		}
		if seenUnavailable && c.Available {
			// Local (ollama) might appear after unavailable
			// cloud providers — that's acceptable since ollama
			// may have a lower score.
			if c.Provider != "ollama" {
				t.Errorf("available model %s appeared after unavailable models "+
					"(non-ollama)", c.ID)
			}
		}
	}
}

// --- SelectBest tests ---

func TestSelectBest_Vision_NoKeys(t *testing.T) {
	best := SelectBest(ModelTypeVision, noEnv)
	// Should still return ollama as fallback.
	require.NotNil(t, best)
	assert.Equal(t, "ollama", best.Provider)
	assert.True(t, best.Available)
}

func TestSelectBest_Chat_NoKeys(t *testing.T) {
	best := SelectBest(ModelTypeChat, noEnv)
	require.NotNil(t, best)
	assert.Equal(t, "ollama", best.Provider)
	assert.True(t, best.Available)
}

func TestSelectBest_Vision_WithAnthropicKey(t *testing.T) {
	checker := selectiveEnv("ANTHROPIC_API_KEY")
	best := SelectBest(ModelTypeVision, checker)
	require.NotNil(t, best)
	assert.True(t, best.Available)
	// Should pick a cloud provider over ollama.
	assert.Equal(t, "anthropic", best.Provider)
}

func TestSelectBest_Chat_WithAnthropicKey(t *testing.T) {
	checker := selectiveEnv("ANTHROPIC_API_KEY")
	best := SelectBest(ModelTypeChat, checker)
	require.NotNil(t, best)
	assert.True(t, best.Available)
	assert.Equal(t, "anthropic", best.Provider)
}

func TestSelectBest_Vision_WithAllKeys(t *testing.T) {
	best := SelectBest(ModelTypeVision, allEnv)
	require.NotNil(t, best)
	assert.True(t, best.Available)
	// Score should be high — one of the top-tier providers.
	assert.Greater(t, best.Score, 0.5)
}

func TestSelectBest_Chat_WithAllKeys(t *testing.T) {
	best := SelectBest(ModelTypeChat, allEnv)
	require.NotNil(t, best)
	assert.True(t, best.Available)
	assert.Greater(t, best.Score, 0.5)
}

// --- SelectAllWithLocalModels tests ---

func TestSelectAllWithLocalModels_Vision(t *testing.T) {
	localModels := []local.LocalModelInfo{
		{
			Name:           "llava:13b",
			Provider:       "ollama",
			Host:           "thinker.local",
			SupportsVision: true,
			ParameterCount: 13_000_000_000,
			RAMRequired:    8_000_000_000,
			GPURequired:    8_000_000_000,
		},
		{
			Name:           "llama3.1:8b",
			Provider:       "ollama",
			Host:           "thinker.local",
			SupportsVision: false,
			ParameterCount: 8_000_000_000,
		},
	}

	candidates := SelectAllWithLocalModels(
		ModelTypeVision, noEnv, localModels,
	)

	// Only the vision local model should be included (plus
	// the registry ollama entry).
	localCount := 0
	for _, c := range candidates {
		if c.IsLocal && c.Host == "thinker.local" {
			localCount++
			assert.True(t, c.Available)
			assert.Equal(t, "llava:13b", c.Model)
		}
	}
	assert.Equal(t, 1, localCount,
		"only vision-capable local model should be included for vision type")
}

func TestSelectAllWithLocalModels_Chat(t *testing.T) {
	localModels := []local.LocalModelInfo{
		{
			Name:           "llava:13b",
			Provider:       "ollama",
			Host:           "thinker.local",
			SupportsVision: true,
			ParameterCount: 13_000_000_000,
		},
		{
			Name:           "llama3.1:8b",
			Provider:       "ollama",
			Host:           "thinker.local",
			SupportsVision: false,
			ParameterCount: 8_000_000_000,
		},
	}

	candidates := SelectAllWithLocalModels(
		ModelTypeChat, noEnv, localModels,
	)

	// Both local models should be included for chat type.
	localCount := 0
	for _, c := range candidates {
		if c.IsLocal && c.Host == "thinker.local" {
			localCount++
		}
	}
	assert.Equal(t, 2, localCount,
		"all local models should be included for chat type")
}

func TestSelectAllWithLocalModels_LocalModelsAvailable(t *testing.T) {
	localModels := []local.LocalModelInfo{
		{
			Name:           "llava:7b",
			Provider:       "ollama",
			Host:           "gpu-box.local",
			SupportsVision: true,
			ParameterCount: 7_000_000_000,
		},
	}

	candidates := SelectAllWithLocalModels(
		ModelTypeVision, noEnv, localModels,
	)

	for _, c := range candidates {
		if c.Host == "gpu-box.local" {
			assert.True(t, c.Available,
				"probed local models should be marked available")
			assert.True(t, c.IsLocal)
			assert.Equal(t, ModelTypeVision, c.Type)
		}
	}
}

func TestSelectAllWithLocalModels_NilLocalModels(t *testing.T) {
	candidates := SelectAllWithLocalModels(
		ModelTypeVision, allEnv, nil,
	)
	assert.NotEmpty(t, candidates,
		"should still return cloud candidates with nil local models")
}

// --- Scoring consistency tests ---

func TestSelectAll_ScoresNonZero(t *testing.T) {
	for _, mt := range []ModelType{ModelTypeVision, ModelTypeChat} {
		candidates := SelectAll(mt, allEnv)
		for _, c := range candidates {
			assert.Greater(t, c.Score, 0.0,
				"score should be positive for %s (%s)", c.ID, mt)
			assert.LessOrEqual(t, c.Score, 1.0,
				"score should be <= 1 for %s (%s)", c.ID, mt)
		}
	}
}

func TestSelectAll_AllHaveRequiredFields(t *testing.T) {
	for _, mt := range []ModelType{ModelTypeVision, ModelTypeChat} {
		candidates := SelectAll(mt, allEnv)
		for _, c := range candidates {
			assert.NotEmpty(t, c.ID, "ID must not be empty")
			assert.NotEmpty(t, c.Provider, "Provider must not be empty for %s", c.ID)
			assert.NotEmpty(t, c.Model, "Model must not be empty for %s", c.ID)
			assert.Equal(t, mt, c.Type, "Type must match for %s", c.ID)
		}
	}
}

// --- isAvailable tests ---

func TestIsAvailable_OllamaAlwaysAvailable(t *testing.T) {
	assert.True(t, isAvailable("ollama", noEnv))
}

func TestIsAvailable_CloudRequiresKey(t *testing.T) {
	assert.False(t, isAvailable("anthropic", noEnv))
	assert.True(t, isAvailable("anthropic", allEnv))
}

func TestIsAvailable_UnknownProvider(t *testing.T) {
	assert.False(t, isAvailable("unknown-provider", allEnv))
}

func TestIsAvailable_SelectiveKeys(t *testing.T) {
	checker := selectiveEnv("OPENAI_API_KEY")
	assert.True(t, isAvailable("openai", checker))
	assert.False(t, isAvailable("anthropic", checker))
	assert.True(t, isAvailable("ollama", checker))
}
