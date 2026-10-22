// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package helixqa

import (
	"digital.vasic.llmsverifier/pkg/recipe"
	"digital.vasic.llmsverifier/pkg/strategy"
)

// QARecipe returns a recipe optimized for HelixQA autonomous testing
func QARecipe() *recipe.Recipe {
	return recipe.NewRecipeBuilder().
		WithName("helixqa-autonomous").
		WithDescription("Optimized for HelixQA autonomous QA sessions with vision, speed, and quality").
		WithStrategy(NewQAStrategy()).
		WithWeights(map[string]float64{
			strategy.DimensionVision:      0.25,
			strategy.DimensionSpeed:       0.25,
			strategy.DimensionQuality:     0.30,
			strategy.DimensionCost:        0.10,
			strategy.DimensionReliability: 0.10,
		}).
		WithMinQuality(0.7).
		WithMinReliability(0.85).
		WithMaxLatency(3000).
		WithVisionRequired(true).
		WithStreamingRequired(true).
		WithMinContext(16000).
		WithFallback(strategy.FallbackRule{
			Name:                "fallback-no-vision",
			Condition:           "no_vision_models_available",
			AlternativeStrategy: "default",
			Priority:            1,
		}).
		WithFallback(strategy.FallbackRule{
			Name:                "fallback-speed",
			Condition:           "latency_exceeded",
			AlternativeProvider: "groq",
			Priority:            2,
		}).
		WithTag("helixqa").
		WithTag("autonomous").
		WithTag("vision").
		WithTag("testing").
		WithVersion("1.0.0").
		BuildOrPanic()
}

// QAVisionOnlyRecipe returns a strict recipe that requires vision
func QAVisionOnlyRecipe() *recipe.Recipe {
	return recipe.NewRecipeBuilder().
		WithName("helixqa-vision-only").
		WithDescription("Strict vision-only recipe for screenshot analysis").
		WithStrategy(NewQAStrategy(func(cfg *QAStrategyConfig) {
			cfg.VisionWeight = 0.4
			cfg.QualityWeight = 0.35
			cfg.SpeedWeight = 0.15
			cfg.CostWeight = 0.05
			cfg.ReliabilityWeight = 0.05
		})).
		WithMinQuality(0.8).
		WithMinReliability(0.9).
		WithMaxLatency(2000).
		WithVisionRequired(true).
		WithMinContext(32000).
		WithTag("helixqa").
		WithTag("vision-only").
		WithTag("strict").
		WithVersion("1.0.0").
		BuildOrPanic()
}

// QAFastRecipe returns a recipe optimized for fast interactive testing
func QAFastRecipe() *recipe.Recipe {
	return recipe.NewRecipeBuilder().
		WithName("helixqa-fast").
		WithDescription("Optimized for fast interactive QA testing").
		WithStrategy(NewQAStrategy(func(cfg *QAStrategyConfig) {
			cfg.VisionWeight = 0.2
			cfg.SpeedWeight = 0.4
			cfg.QualityWeight = 0.25
			cfg.CostWeight = 0.1
			cfg.ReliabilityWeight = 0.05
		})).
		WithMinQuality(0.65).
		WithMinReliability(0.8).
		WithMaxLatency(1000).
		WithVisionRequired(false).
		WithStreamingRequired(true).
		WithTag("helixqa").
		WithTag("fast").
		WithTag("interactive").
		WithVersion("1.0.0").
		BuildOrPanic()
}

// QAComprehensiveRecipe returns a recipe for comprehensive testing
func QAComprehensiveRecipe() *recipe.Recipe {
	return recipe.NewRecipeBuilder().
		WithName("helixqa-comprehensive").
		WithDescription("Comprehensive recipe for thorough QA testing with high quality").
		WithStrategy(NewQAStrategy(func(cfg *QAStrategyConfig) {
			cfg.VisionWeight = 0.25
			cfg.QualityWeight = 0.4
			cfg.ReliabilityWeight = 0.2
			cfg.SpeedWeight = 0.1
			cfg.CostWeight = 0.05
		})).
		WithMinQuality(0.85).
		WithMinReliability(0.95).
		WithMaxLatency(5000).
		WithVisionRequired(true).
		WithStreamingRequired(true).
		WithMinContext(64000).
		WithTag("helixqa").
		WithTag("comprehensive").
		WithTag("high-quality").
		WithVersion("1.0.0").
		BuildOrPanic()
}

// GetQARecipe returns a QA recipe by name
func GetQARecipe(name string) *recipe.Recipe {
	switch name {
	case "default", "autonomous":
		return QARecipe()
	case "vision-only":
		return QAVisionOnlyRecipe()
	case "fast":
		return QAFastRecipe()
	case "comprehensive":
		return QAComprehensiveRecipe()
	default:
		return nil
	}
}

// ListQARecipes returns all available QA recipe names
func ListQARecipes() []string {
	return []string{
		"autonomous",
		"vision-only",
		"fast",
		"comprehensive",
	}
}
