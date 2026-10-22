// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package recipe

import (
	"time"

	"digital.vasic.llmsverifier/pkg/chat"
	"digital.vasic.llmsverifier/pkg/strategy"
	"digital.vasic.llmsverifier/pkg/vision"
)

// Predefined recipes for common use cases

// QARecipe returns a recipe optimized for autonomous QA testing.
// Prioritizes vision, speed, and quality over cost.
func QARecipe() *Recipe {
	return NewRecipeBuilder().
		WithName("qa-autonomous").
		WithDescription("Optimized for autonomous QA testing with vision, speed, and quality").
		WithStrategy(strategy.NewDefaultStrategy()).
		WithWeights(map[string]float64{
			strategy.DimensionQuality:     0.35,
			strategy.DimensionSpeed:       0.30,
			strategy.DimensionVision:      0.20,
			strategy.DimensionReliability: 0.15,
		}).
		WithMinQuality(0.7).
		WithMinReliability(0.85).
		WithMaxLatency(3000).
		WithVisionRequired(true).
		WithStreamingRequired(true).
		WithMinContext(16000).
		WithTimeout(10 * time.Minute).
		WithMaxRetries(5).
		WithCacheTTL(10 * time.Minute).
		WithTag("qa").
		WithTag("autonomous").
		WithTag("vision").
		WithVersion("1.0.0").
		BuildOrPanic()
}

// SpeedRecipe returns a recipe optimized for fast responses.
// Prioritizes speed and cost over quality.
func SpeedRecipe() *Recipe {
	return NewRecipeBuilder().
		WithName("speed-optimized").
		WithDescription("Optimized for fast responses with acceptable quality").
		WithStrategy(strategy.NewDefaultStrategy()).
		WithWeights(map[string]float64{
			strategy.DimensionSpeed:       0.5,
			strategy.DimensionCost:        0.25,
			strategy.DimensionQuality:     0.15,
			strategy.DimensionReliability: 0.1,
		}).
		WithMinQuality(0.6).
		WithMinReliability(0.8).
		WithMaxLatency(1000).
		WithStreamingRequired(true).
		WithTimeout(3 * time.Minute).
		WithMaxRetries(3).
		WithTag("speed").
		WithTag("low-latency").
		WithVersion("1.0.0").
		BuildOrPanic()
}

// QualityRecipe returns a recipe optimized for highest quality.
// Prioritizes quality and reliability over speed and cost.
func QualityRecipe() *Recipe {
	return NewRecipeBuilder().
		WithName("quality-optimized").
		WithDescription("Optimized for highest quality responses").
		WithStrategy(strategy.NewDefaultStrategy()).
		WithWeights(map[string]float64{
			strategy.DimensionQuality:     0.5,
			strategy.DimensionReliability: 0.3,
			strategy.DimensionSpeed:       0.1,
			strategy.DimensionCost:        0.1,
		}).
		WithMinQuality(0.85).
		WithMinReliability(0.9).
		WithMaxLatency(10000).
		WithStreamingRequired(true).
		WithMinContext(32000).
		WithTimeout(15 * time.Minute).
		WithMaxRetries(5).
		WithTag("quality").
		WithTag("high-quality").
		WithVersion("1.0.0").
		BuildOrPanic()
}

// CostRecipe returns a recipe optimized for cost efficiency.
// Prioritizes cost over quality and speed.
func CostRecipe() *Recipe {
	return NewRecipeBuilder().
		WithName("cost-optimized").
		WithDescription("Optimized for cost efficiency with acceptable quality").
		WithStrategy(strategy.NewDefaultStrategy()).
		WithWeights(map[string]float64{
			strategy.DimensionCost:        0.4,
			strategy.DimensionQuality:     0.3,
			strategy.DimensionReliability: 0.2,
			strategy.DimensionSpeed:       0.1,
		}).
		WithMinQuality(0.5).
		WithMinReliability(0.8).
		WithMaxLatency(5000).
		WithTimeout(5 * time.Minute).
		WithMaxRetries(3).
		WithTag("cost").
		WithTag("budget").
		WithVersion("1.0.0").
		BuildOrPanic()
}

// CodeGenerationRecipe returns a recipe optimized for code generation.
// Prioritizes quality and context over speed.
func CodeGenerationRecipe() *Recipe {
	return NewRecipeBuilder().
		WithName("code-generation").
		WithDescription("Optimized for code generation with large context").
		WithStrategy(strategy.NewDefaultStrategy()).
		WithWeights(map[string]float64{
			strategy.DimensionQuality:     0.45,
			strategy.DimensionContext:     0.25,
			strategy.DimensionReliability: 0.2,
			strategy.DimensionSpeed:       0.1,
		}).
		WithMinQuality(0.8).
		WithMinReliability(0.9).
		WithMaxLatency(8000).
		WithStreamingRequired(true).
		WithMinContext(64000).
		WithTimeout(20 * time.Minute).
		WithMaxRetries(5).
		WithTag("code").
		WithTag("generation").
		WithTag("large-context").
		WithVersion("1.0.0").
		BuildOrPanic()
}

// VisionRecipe returns a recipe optimized for vision tasks.
// Requires vision capability with high quality. Uses the dedicated
// VisionStrategy which scores models on image understanding, GUI
// detection, OCR accuracy, speed, and cost.
func VisionRecipe() *Recipe {
	return NewRecipeBuilder().
		WithName("vision-optimized").
		WithDescription("Optimized for image analysis and vision tasks using VisionStrategy").
		WithStrategy(VisionPreset()).
		WithWeights(map[string]float64{
			strategy.DimensionQuality: 0.35,
			strategy.DimensionVision:  0.25,
			strategy.DimensionContext: 0.15,
			strategy.DimensionSpeed:   0.15,
			strategy.DimensionCost:    0.10,
		}).
		WithMinQuality(0.75).
		WithMinReliability(0.85).
		WithMaxLatency(5000).
		WithVisionRequired(true).
		WithStreamingRequired(true).
		WithMinContext(16000).
		WithTimeout(10 * time.Minute).
		WithMaxRetries(5).
		WithTag("vision").
		WithTag("multimodal").
		WithTag("image-analysis").
		WithVersion("1.1.0").
		BuildOrPanic()
}

// VisionPreset returns a VisionStrategy instance configured with
// default weights optimised for vision model selection. Use this
// when vision model selection is specifically needed (e.g., HelixQA
// Execute/Curiosity phases, screenshot analysis, UI navigation).
func VisionPreset() *vision.VisionStrategy {
	return vision.NewVisionStrategy()
}

// NavigationPreset returns a NavigationStrategy instance configured
// with default weights optimised for UI navigation tasks. It
// prioritises JSON compliance (structured action arrays), GUI
// understanding, speed, and cost. Use this for HelixQA Execute and
// Curiosity phases where the model must produce machine-parseable
// JSON actions from screenshots.
func NavigationPreset() *vision.NavigationStrategy {
	return vision.NewNavigationStrategy()
}

// AnalysisPreset returns an AnalysisStrategy instance configured
// with default weights optimised for screenshot analysis tasks. It
// prioritises description quality, OCR accuracy, object detection,
// comprehensiveness, and cost. Use this for HelixQA Analyze phase
// where the model must produce rich, detailed image descriptions.
func AnalysisPreset() *vision.AnalysisStrategy {
	return vision.NewAnalysisStrategy()
}

// PlanningPreset returns a PlanningStrategy instance configured
// with default weights optimised for test plan generation. It
// prioritises reasoning quality, context window size, structured
// output, speed, and cost. Use this for HelixQA Learn and Plan
// phases where the model must produce well-reasoned test cases.
func PlanningPreset() *chat.PlanningStrategy {
	return chat.NewPlanningStrategy()
}

// BalancedRecipe returns a balanced recipe for general use.
func BalancedRecipe() *Recipe {
	return NewRecipeBuilder().
		WithName("balanced").
		WithDescription("Balanced recipe for general-purpose use").
		WithStrategy(strategy.NewDefaultStrategy()).
		WithWeights(map[string]float64{
			strategy.DimensionQuality:     0.35,
			strategy.DimensionSpeed:       0.25,
			strategy.DimensionCost:        0.2,
			strategy.DimensionReliability: 0.2,
		}).
		WithMinQuality(0.7).
		WithMinReliability(0.85).
		WithMaxLatency(3000).
		WithStreamingRequired(true).
		WithMinContext(16000).
		WithTimeout(5 * time.Minute).
		WithMaxRetries(3).
		WithTag("balanced").
		WithTag("general").
		WithVersion("1.0.0").
		BuildOrPanic()
}

// GetRecipeByName returns a predefined recipe by name.
func GetRecipeByName(name string) *Recipe {
	switch name {
	case "qa", "qa-autonomous":
		return QARecipe()
	case "speed", "speed-optimized":
		return SpeedRecipe()
	case "quality", "quality-optimized":
		return QualityRecipe()
	case "cost", "cost-optimized":
		return CostRecipe()
	case "code", "code-generation":
		return CodeGenerationRecipe()
	case "vision", "vision-optimized":
		return VisionRecipe()
	case "balanced", "default":
		return BalancedRecipe()
	default:
		return nil
	}
}

// ListRecipes returns all predefined recipe names.
func ListRecipes() []string {
	return []string{
		"qa-autonomous",
		"speed-optimized",
		"quality-optimized",
		"cost-optimized",
		"code-generation",
		"vision-optimized",
		"balanced",
	}
}
