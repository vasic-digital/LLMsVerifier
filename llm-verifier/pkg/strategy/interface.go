// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

// Package strategy provides the Strategy pattern for LLM verification,
// allowing different verification strategies to be plugged in based on
// use case requirements.
package strategy

import (
	"context"
	"time"
)

// VerificationStrategy defines how LLMs are scored, validated, and selected.
// Implementations can optimize for different criteria (quality, speed, cost, etc.)
type VerificationStrategy interface {
	// Name returns the strategy identifier
	Name() string

	// Description returns a human-readable description of the strategy
	Description() string

	// Score evaluates a model and returns a detailed score breakdown
	Score(ctx context.Context, model ModelInfo) (StrategyScore, error)

	// Validate checks if a model meets the strategy's minimum requirements
	Validate(ctx context.Context, model ModelInfo) ValidationResult

	// Rank sorts models by strategy-specific criteria
	Rank(ctx context.Context, models []ModelInfo) ([]RankedModel, error)

	// Select chooses the best model from the ranked list based on requirements
	Select(ctx context.Context, ranked []RankedModel, req Requirements) (ModelInfo, error)
}

// ModelInfo contains information about an LLM model
type ModelInfo struct {
	// ID is the unique identifier for the model
	ID string `json:"id"`

	// Name is the human-readable name
	Name string `json:"name"`

	// Provider is the LLM provider (openai, anthropic, google, etc.)
	Provider string `json:"provider"`

	// Model is the specific model identifier
	Model string `json:"model"`

	// SupportsVision indicates if the model can process images
	SupportsVision bool `json:"supports_vision"`

	// SupportsStreaming indicates if the model supports streaming responses
	SupportsStreaming bool `json:"supports_streaming"`

	// SupportsFunctionCalling indicates if the model supports function calling
	SupportsFunctionCalling bool `json:"supports_function_calling"`

	// ContextWindow is the maximum context size in tokens
	ContextWindow int `json:"context_window"`

	// MaxOutputTokens is the maximum output tokens
	MaxOutputTokens int `json:"max_output_tokens"`

	// AvgLatencyMs is the average response latency in milliseconds
	AvgLatencyMs int `json:"avg_latency_ms"`

	// InputCostPer1k is the cost per 1000 input tokens in USD
	InputCostPer1k float64 `json:"input_cost_per_1k"`

	// OutputCostPer1k is the cost per 1000 output tokens in USD
	OutputCostPer1k float64 `json:"output_cost_per_1k"`

	// QualityScore is a quality rating (0-1) from benchmarks
	QualityScore float64 `json:"quality_score"`

	// ReliabilityScore is the uptime/reliability rating (0-1)
	ReliabilityScore float64 `json:"reliability_score"`

	// Verified indicates if the model has passed verification
	Verified bool `json:"verified"`

	// LastVerified is when the model was last verified
	LastVerified time.Time `json:"last_verified"`

	// Capabilities lists additional model capabilities
	Capabilities []string `json:"capabilities"`

	// Metadata contains additional provider-specific information
	Metadata map[string]any `json:"metadata,omitempty"`
}

// StrategyScore contains detailed scoring breakdown
type StrategyScore struct {
	// Overall score (0-1)
	Overall float64 `json:"overall"`

	// DimensionScores breaks down the score by dimension
	DimensionScores map[string]float64 `json:"dimension_scores"`

	// Confidence indicates how confident the scoring is (0-1)
	Confidence float64 `json:"confidence"`

	// Reasoning explains the score
	Reasoning string `json:"reasoning"`

	// Timestamp when the score was calculated
	Timestamp time.Time `json:"timestamp"`

	// ModelID is the model that was scored
	ModelID string `json:"model_id"`

	// StrategyName is the strategy that produced this score
	StrategyName string `json:"strategy_name"`
}

// ValidationResult contains the result of model validation
type ValidationResult struct {
	// Valid indicates if the model passes validation
	Valid bool `json:"valid"`

	// Errors lists validation errors
	Errors []string `json:"errors,omitempty"`

	// Warnings lists validation warnings
	Warnings []string `json:"warnings,omitempty"`

	// Score is the validation score (0-1)
	Score float64 `json:"score"`

	// Details provides additional validation details
	Details map[string]any `json:"details,omitempty"`
}

// RankedModel is a model with its ranking information
type RankedModel struct {
	// Model is the model information
	Model ModelInfo `json:"model"`

	// Rank is the position in the ranked list (1 = best)
	Rank int `json:"rank"`

	// Score is the strategy score for this model
	Score StrategyScore `json:"score"`

	// SelectionProbability for probabilistic selection
	SelectionProbability float64 `json:"selection_probability,omitempty"`

	// Tier groups models by quality tier (tier1, tier2, tier3)
	Tier string `json:"tier,omitempty"`
}

// Requirements specifies what capabilities are needed for model selection
type Requirements struct {
	// NeedsVision indicates vision capability is required
	NeedsVision bool `json:"needs_vision"`

	// NeedsStreaming indicates streaming is required
	NeedsStreaming bool `json:"needs_streaming"`

	// NeedsFunctionCalling indicates function calling is required
	NeedsFunctionCalling bool `json:"needs_function_calling"`

	// MinContextWindow is the minimum context window size required
	MinContextWindow int `json:"min_context_window"`

	// MaxLatencyMs is the maximum acceptable latency
	MaxLatencyMs int `json:"max_latency_ms"`

	// MinQualityScore is the minimum quality score required
	MinQualityScore float64 `json:"min_quality_score"`

	// MinReliabilityScore is the minimum reliability required
	MinReliabilityScore float64 `json:"min_reliability_score"`

	// MaxInputCostPer1k is the maximum input cost per 1k tokens
	MaxInputCostPer1k float64 `json:"max_input_cost_per_1k"`

	// MaxOutputCostPer1k is the maximum output cost per 1k tokens
	MaxOutputCostPer1k float64 `json:"max_output_cost_per_1k"`

	// PreferredProvider is the preferred provider (optional)
	PreferredProvider string `json:"preferred_provider,omitempty"`

	// ExcludedProviders are providers to exclude
	ExcludedProviders []string `json:"excluded_providers,omitempty"`

	// RequiredCapabilities are capabilities that must be present
	RequiredCapabilities []string `json:"required_capabilities,omitempty"`

	// CustomConstraints are strategy-specific constraints
	CustomConstraints map[string]any `json:"custom_constraints,omitempty"`

	// UseCase describes the intended use case
	UseCase string `json:"use_case,omitempty"`
}

// Constraint represents a verification constraint
type Constraint struct {
	// Name is the constraint identifier
	Name string `json:"name"`

	// Type is the constraint type (range, enum, custom)
	Type string `json:"type"`

	// Value is the constraint value
	Value any `json:"value"`

	// Weight is the importance of this constraint (0-1)
	Weight float64 `json:"weight"`

	// Required indicates if this is a hard constraint
	Required bool `json:"required"`

	// Description explains the constraint
	Description string `json:"description"`
}

// FallbackRule defines when and how to fall back to alternative models
type FallbackRule struct {
	// Name is the rule identifier
	Name string `json:"name"`

	// Condition is the condition that triggers fallback
	Condition string `json:"condition"`

	// AlternativeStrategy is the strategy to use for fallback
	AlternativeStrategy string `json:"alternative_strategy,omitempty"`

	// AlternativeProvider is an alternative provider to try
	AlternativeProvider string `json:"alternative_provider,omitempty"`

	// AlternativeModel is a specific alternative model
	AlternativeModel string `json:"alternative_model,omitempty"`

	// Priority is the fallback priority (lower = higher priority)
	Priority int `json:"priority"`
}

// StrategyOption is a functional option for configuring strategies
type StrategyOption func(VerificationStrategy) error

// WithWeights sets custom dimension weights
func WithWeights(weights map[string]float64) StrategyOption {
	return func(s VerificationStrategy) error {
		if ws, ok := s.(interface{ SetWeights(map[string]float64) }); ok {
			ws.SetWeights(weights)
		}
		return nil
	}
}

// WithConstraints adds verification constraints
func WithConstraints(constraints []Constraint) StrategyOption {
	return func(s VerificationStrategy) error {
		if cs, ok := s.(interface{ SetConstraints([]Constraint) }); ok {
			cs.SetConstraints(constraints)
		}
		return nil
	}
}

// WithFallbacks sets fallback rules
func WithFallbacks(fallbacks []FallbackRule) StrategyOption {
	return func(s VerificationStrategy) error {
		if fs, ok := s.(interface{ SetFallbacks([]FallbackRule) }); ok {
			fs.SetFallbacks(fallbacks)
		}
		return nil
	}
}

// Dimension constants for scoring
const (
	DimensionQuality     = "quality"
	DimensionSpeed       = "speed"
	DimensionCost        = "cost"
	DimensionReliability = "reliability"
	DimensionVision      = "vision"
	DimensionContext     = "context"
	DimensionCapability  = "capability"
)

// Tier constants for model grouping
const (
	Tier1 = "tier1" // Top-tier models
	Tier2 = "tier2" // Mid-tier models
	Tier3 = "tier3" // Budget-tier models
)
