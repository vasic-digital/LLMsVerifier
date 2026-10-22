// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

// Package selector provides unified model selection across both
// cloud and local providers for HelixQA. It combines the vision
// model registry (pkg/helixqa), the chat model registry
// (pkg/catalogizer), and locally discovered models (pkg/local)
// into a single ranked candidate list, enabling HelixQA to
// dynamically select the best model for each purpose (vision
// vs chat) without hardcoding preferences.
package selector

import (
	"context"
	"sort"

	"digital.vasic.llmsverifier/pkg/catalogizer"
	"digital.vasic.llmsverifier/pkg/helixqa"
	"digital.vasic.llmsverifier/pkg/local"
	"digital.vasic.llmsverifier/pkg/strategy"
)

// ModelType distinguishes vision vs chat models so the selector
// can filter and rank candidates for the appropriate purpose.
type ModelType string

const (
	// ModelTypeVision selects models suited for screenshot
	// analysis, UI navigation, and visual QA tasks.
	ModelTypeVision ModelType = "vision"

	// ModelTypeChat selects models suited for general purpose
	// reasoning, test planning, and report generation.
	ModelTypeChat ModelType = "chat"
)

// ModelCandidate wraps both cloud and local models with unified
// scoring metadata so they can be compared and ranked together.
type ModelCandidate struct {
	// ID is the unique model identifier.
	ID string `json:"id"`

	// Provider is the LLM provider name.
	Provider string `json:"provider"`

	// Model is the specific model identifier.
	Model string `json:"model"`

	// Type indicates whether this is a vision or chat model.
	Type ModelType `json:"type"`

	// IsLocal indicates the model runs on local infrastructure.
	IsLocal bool `json:"is_local"`

	// Score is the computed quality score (0-1).
	Score float64 `json:"score"`

	// Available indicates whether the provider has been verified
	// as reachable (API key set for cloud, host reachable for local).
	Available bool `json:"available"`

	// Host is the hostname for local models (empty for cloud).
	Host string `json:"host,omitempty"`

	// Info contains the full model information from the registry.
	Info strategy.ModelInfo `json:"info"`
}

// EnvChecker is a function that returns true if the given
// environment variable is set and non-empty. This allows callers
// to inject their own environment checking logic for testing.
type EnvChecker func(string) bool

// SelectBest returns the highest-scoring available model of the
// given type. Returns nil if no available models exist.
func SelectBest(modelType ModelType, envChecker EnvChecker) *ModelCandidate {
	candidates := SelectAll(modelType, envChecker)
	for _, c := range candidates {
		if c.Available {
			return &c
		}
	}
	return nil
}

// SelectAll returns all models of the given type sorted by score
// (highest first). Both available and unavailable models are
// included so callers can see the full landscape.
func SelectAll(modelType ModelType, envChecker EnvChecker) []ModelCandidate {
	return SelectAllWithLocalModels(modelType, envChecker, nil)
}

// SelectAllWithLocalModels is like SelectAll but also includes
// locally discovered models in the candidate pool.
func SelectAllWithLocalModels(
	modelType ModelType,
	envChecker EnvChecker,
	localModels []local.LocalModelInfo,
) []ModelCandidate {
	var candidates []ModelCandidate

	// Gather cloud models from the appropriate registry.
	switch modelType {
	case ModelTypeVision:
		candidates = append(candidates,
			visionCandidatesFromRegistry(envChecker)...)
	case ModelTypeChat:
		candidates = append(candidates,
			chatCandidatesFromRegistry(envChecker)...)
	}

	// Add local models that match the requested type.
	for _, lm := range localModels {
		if modelType == ModelTypeVision && !lm.SupportsVision {
			continue
		}
		info := local.ToModelInfo(lm)
		score := local.ScoreLocalModel(lm)

		candidates = append(candidates, ModelCandidate{
			ID:        info.ID,
			Provider:  lm.Provider,
			Model:     lm.Name,
			Type:      modelType,
			IsLocal:   true,
			Score:     score,
			Available: true, // probed models are available
			Host:      lm.Host,
			Info:      info,
		})
	}

	// Sort by score descending.
	sort.SliceStable(candidates, func(i, j int) bool {
		// Available models always rank above unavailable ones.
		if candidates[i].Available != candidates[j].Available {
			return candidates[i].Available
		}
		return candidates[i].Score > candidates[j].Score
	})

	return candidates
}

// providerEnvKeys maps provider names to their API key
// environment variables. Matches the keys used in
// pkg/catalogizer and pkg/helixqa registries.
var providerEnvKeys = map[string]string{
	"anthropic":    "ANTHROPIC_API_KEY",
	"openai":       "OPENAI_API_KEY",
	"google":       "GOOGLE_API_KEY",
	"kimi":         "KIMI_API_KEY",
	"deepseek":     "DEEPSEEK_API_KEY",
	"groq":         "GROQ_API_KEY",
	"cerebras":     "CEREBRAS_API_KEY",
	"stepfun":      "STEPFUN_API_KEY",
	"qwen":         "QWEN_API_KEY",
	"astica":       "ASTICA_API_KEY",
	"nvidia":       "NVIDIA_API_KEY",
	"githubmodels": "GITHUB_MODELS_API_KEY",
	"xai":          "XAI_API_KEY",
	"openrouter":   "OPENROUTER_API_KEY",
	"huggingface":  "HUGGINGFACE_API_KEY",
	"chutes":       "CHUTES_API_KEY",
	"siliconflow":  "SILICONFLOW_API_KEY",
	"replicate":    "REPLICATE_API_KEY",
	"zhipu":        "ZHIPU_API_KEY",
	"ollama":       "", // local, always available
}

// isAvailable checks if a provider's API key is configured.
func isAvailable(provider string, envChecker EnvChecker) bool {
	envKey, ok := providerEnvKeys[provider]
	if !ok {
		// Unknown provider — check if there might be an env key
		// following the PROVIDER_API_KEY convention.
		return false
	}
	if envKey == "" {
		// Local providers are always available.
		return true
	}
	return envChecker(envKey)
}

// visionCandidatesFromRegistry builds candidates from the
// VisionModelRegistry in pkg/helixqa.
func visionCandidatesFromRegistry(envChecker EnvChecker) []ModelCandidate {
	ctx := context.Background()
	scorer := helixqa.NewQAStrategy()
	models := helixqa.VisionModelRegistry()

	candidates := make([]ModelCandidate, 0, len(models))
	for _, m := range models {
		score, err := scorer.Score(ctx, m)
		overallScore := m.QualityScore // fallback
		if err == nil {
			overallScore = score.Overall
		}

		candidates = append(candidates, ModelCandidate{
			ID:        m.ID,
			Provider:  m.Provider,
			Model:     m.Model,
			Type:      ModelTypeVision,
			IsLocal:   m.Provider == "ollama",
			Score:     overallScore,
			Available: isAvailable(m.Provider, envChecker),
			Info:      m,
		})
	}
	return candidates
}

// chatCandidatesFromRegistry builds candidates from the
// ModelRegistry in pkg/catalogizer.
func chatCandidatesFromRegistry(envChecker EnvChecker) []ModelCandidate {
	ctx := context.Background()
	scorer := catalogizer.NewCatalogizerStrategy()
	models := catalogizer.ModelRegistry()

	candidates := make([]ModelCandidate, 0, len(models))
	for _, m := range models {
		score, err := scorer.Score(ctx, m)
		overallScore := m.QualityScore // fallback
		if err == nil {
			overallScore = score.Overall
		}

		candidates = append(candidates, ModelCandidate{
			ID:        m.ID,
			Provider:  m.Provider,
			Model:     m.Model,
			Type:      ModelTypeChat,
			IsLocal:   m.Provider == "ollama",
			Score:     overallScore,
			Available: isAvailable(m.Provider, envChecker),
			Info:      m,
		})
	}
	return candidates
}
