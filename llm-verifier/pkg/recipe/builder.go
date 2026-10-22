// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

// Package recipe provides a builder pattern for constructing verification
// recipes that combine strategies, constraints, and fallback rules.
package recipe

import (
	"digital.vasic.llmsverifier/pkg/strategy"
	"fmt"
	"time"
)

// Recipe is a complete verification configuration that can be applied
// to select and verify LLMs for a specific use case.
type Recipe struct {
	// ID is the unique recipe identifier
	ID string `json:"id"`

	// Name is the human-readable recipe name
	Name string `json:"name"`

	// Description explains the recipe purpose
	Description string `json:"description"`

	// Strategy is the verification strategy to use
	Strategy strategy.VerificationStrategy `json:"-"`

	// StrategyName is the name of the strategy (for serialization)
	StrategyName string `json:"strategy_name"`

	// Constraints are the verification constraints to apply
	Constraints []strategy.Constraint `json:"constraints"`

	// Weights are custom dimension weights
	Weights map[string]float64 `json:"weights"`

	// Fallbacks are fallback rules for degraded operation
	Fallbacks []strategy.FallbackRule `json:"fallbacks"`

	// Timeout is the verification timeout
	Timeout time.Duration `json:"timeout"`

	// MaxRetries is the maximum number of retries
	MaxRetries int `json:"max_retries"`

	// CacheTTL is how long to cache results
	CacheTTL time.Duration `json:"cache_ttl"`

	// Tags for categorization
	Tags []string `json:"tags"`

	// Version of the recipe
	Version string `json:"version"`
}

// RecipeBuilder constructs verification recipes using a fluent interface.
type RecipeBuilder struct {
	recipe *Recipe
	errors []error
}

// NewRecipeBuilder creates a new recipe builder.
func NewRecipeBuilder() *RecipeBuilder {
	return &RecipeBuilder{
		recipe: &Recipe{
			Weights:     make(map[string]float64),
			Constraints: make([]strategy.Constraint, 0),
			Fallbacks:   make([]strategy.FallbackRule, 0),
			Tags:        make([]string, 0),
			Timeout:     5 * time.Minute,
			MaxRetries:  3,
			CacheTTL:    5 * time.Minute,
			Version:     "1.0.0",
		},
		errors: make([]error, 0),
	}
}

// WithName sets the recipe name.
func (b *RecipeBuilder) WithName(name string) *RecipeBuilder {
	b.recipe.Name = name
	return b
}

// WithDescription sets the recipe description.
func (b *RecipeBuilder) WithDescription(desc string) *RecipeBuilder {
	b.recipe.Description = desc
	return b
}

// WithStrategy sets the verification strategy.
func (b *RecipeBuilder) WithStrategy(s strategy.VerificationStrategy) *RecipeBuilder {
	if s == nil {
		b.errors = append(b.errors, fmt.Errorf("strategy cannot be nil"))
		return b
	}
	b.recipe.Strategy = s
	b.recipe.StrategyName = s.Name()
	return b
}

// WithWeight sets a dimension weight.
func (b *RecipeBuilder) WithWeight(dimension string, weight float64) *RecipeBuilder {
	if weight < 0 || weight > 1 {
		b.errors = append(b.errors, fmt.Errorf("weight for %s must be between 0 and 1", dimension))
		return b
	}
	b.recipe.Weights[dimension] = weight
	return b
}

// WithWeights sets multiple dimension weights.
func (b *RecipeBuilder) WithWeights(weights map[string]float64) *RecipeBuilder {
	for dim, weight := range weights {
		b.WithWeight(dim, weight)
	}
	return b
}

// WithConstraint adds a verification constraint.
func (b *RecipeBuilder) WithConstraint(c strategy.Constraint) *RecipeBuilder {
	b.recipe.Constraints = append(b.recipe.Constraints, c)
	return b
}

// WithConstraints adds multiple verification constraints.
func (b *RecipeBuilder) WithConstraints(constraints []strategy.Constraint) *RecipeBuilder {
	b.recipe.Constraints = append(b.recipe.Constraints, constraints...)
	return b
}

// WithMinQuality adds a minimum quality constraint.
func (b *RecipeBuilder) WithMinQuality(minQuality float64) *RecipeBuilder {
	return b.WithConstraint(strategy.Constraint{
		Name:        "min_quality",
		Type:        "range",
		Value:       minQuality,
		Weight:      1.0,
		Required:    true,
		Description: fmt.Sprintf("minimum quality score of %.2f", minQuality),
	})
}

// WithMinReliability adds a minimum reliability constraint.
func (b *RecipeBuilder) WithMinReliability(minReliability float64) *RecipeBuilder {
	return b.WithConstraint(strategy.Constraint{
		Name:        "min_reliability",
		Type:        "range",
		Value:       minReliability,
		Weight:      1.0,
		Required:    true,
		Description: fmt.Sprintf("minimum reliability score of %.2f", minReliability),
	})
}

// WithMaxLatency adds a maximum latency constraint.
func (b *RecipeBuilder) WithMaxLatency(maxMs int) *RecipeBuilder {
	return b.WithConstraint(strategy.Constraint{
		Name:        "max_latency_ms",
		Type:        "range",
		Value:       maxMs,
		Weight:      0.8,
		Required:    false,
		Description: fmt.Sprintf("maximum latency of %dms", maxMs),
	})
}

// WithVisionRequired adds a vision requirement constraint.
func (b *RecipeBuilder) WithVisionRequired(required bool) *RecipeBuilder {
	return b.WithConstraint(strategy.Constraint{
		Name:        "requires_vision",
		Type:        "bool",
		Value:       required,
		Weight:      1.0,
		Required:    required,
		Description: "vision capability required",
	})
}

// WithStreamingRequired adds a streaming requirement constraint.
func (b *RecipeBuilder) WithStreamingRequired(required bool) *RecipeBuilder {
	return b.WithConstraint(strategy.Constraint{
		Name:        "requires_streaming",
		Type:        "bool",
		Value:       required,
		Weight:      0.8,
		Required:    required,
		Description: "streaming capability required",
	})
}

// WithMinContext adds a minimum context window constraint.
func (b *RecipeBuilder) WithMinContext(minTokens int) *RecipeBuilder {
	return b.WithConstraint(strategy.Constraint{
		Name:        "min_context",
		Type:        "range",
		Value:       minTokens,
		Weight:      0.7,
		Required:    true,
		Description: fmt.Sprintf("minimum context window of %d tokens", minTokens),
	})
}

// WithFallback adds a fallback rule.
func (b *RecipeBuilder) WithFallback(rule strategy.FallbackRule) *RecipeBuilder {
	b.recipe.Fallbacks = append(b.recipe.Fallbacks, rule)
	return b
}

// WithFallbacks adds multiple fallback rules.
func (b *RecipeBuilder) WithFallbacks(rules []strategy.FallbackRule) *RecipeBuilder {
	b.recipe.Fallbacks = append(b.recipe.Fallbacks, rules...)
	return b
}

// WithTimeout sets the verification timeout.
func (b *RecipeBuilder) WithTimeout(d time.Duration) *RecipeBuilder {
	if d <= 0 {
		b.errors = append(b.errors, fmt.Errorf("timeout must be positive"))
		return b
	}
	b.recipe.Timeout = d
	return b
}

// WithMaxRetries sets the maximum retries.
func (b *RecipeBuilder) WithMaxRetries(max int) *RecipeBuilder {
	if max < 0 {
		b.errors = append(b.errors, fmt.Errorf("max retries cannot be negative"))
		return b
	}
	b.recipe.MaxRetries = max
	return b
}

// WithCacheTTL sets the cache TTL.
func (b *RecipeBuilder) WithCacheTTL(d time.Duration) *RecipeBuilder {
	b.recipe.CacheTTL = d
	return b
}

// WithTag adds a tag.
func (b *RecipeBuilder) WithTag(tag string) *RecipeBuilder {
	b.recipe.Tags = append(b.recipe.Tags, tag)
	return b
}

// WithTags adds multiple tags.
func (b *RecipeBuilder) WithTags(tags []string) *RecipeBuilder {
	b.recipe.Tags = append(b.recipe.Tags, tags...)
	return b
}

// WithVersion sets the recipe version.
func (b *RecipeBuilder) WithVersion(version string) *RecipeBuilder {
	b.recipe.Version = version
	return b
}

// Build constructs the recipe and validates it.
func (b *RecipeBuilder) Build() (*Recipe, error) {
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("recipe build errors: %v", b.errors)
	}

	if b.recipe.Strategy == nil {
		return nil, fmt.Errorf("strategy is required")
	}

	if b.recipe.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	b.recipe.ID = generateRecipeID(b.recipe.Name)

	if err := b.recipe.Validate(); err != nil {
		return nil, fmt.Errorf("recipe validation failed: %w", err)
	}

	return b.recipe, nil
}

// BuildOrPanic constructs the recipe or panics on error.
func (b *RecipeBuilder) BuildOrPanic() *Recipe {
	r, err := b.Build()
	if err != nil {
		panic(err)
	}
	return r
}

// Validate validates the recipe configuration.
func (r *Recipe) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}

	if r.Strategy == nil {
		return fmt.Errorf("strategy is required")
	}

	if r.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	if r.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}

	totalWeight := 0.0
	for _, w := range r.Weights {
		totalWeight += w
	}

	return nil
}

// Clone creates a copy of the recipe.
func (r *Recipe) Clone() *Recipe {
	clone := &Recipe{
		ID:           r.ID,
		Name:         r.Name,
		Description:  r.Description,
		Strategy:     r.Strategy,
		StrategyName: r.StrategyName,
		Timeout:      r.Timeout,
		MaxRetries:   r.MaxRetries,
		CacheTTL:     r.CacheTTL,
		Version:      r.Version,
		Weights:      make(map[string]float64),
		Constraints:  make([]strategy.Constraint, len(r.Constraints)),
		Fallbacks:    make([]strategy.FallbackRule, len(r.Fallbacks)),
		Tags:         make([]string, len(r.Tags)),
	}

	copy(clone.Constraints, r.Constraints)
	copy(clone.Fallbacks, r.Fallbacks)
	copy(clone.Tags, r.Tags)

	for k, v := range r.Weights {
		clone.Weights[k] = v
	}

	return clone
}

// Apply applies the recipe to a strategy.
func (r *Recipe) Apply(s strategy.VerificationStrategy) error {
	if ws, ok := s.(interface{ SetWeights(map[string]float64) }); ok && len(r.Weights) > 0 {
		ws.SetWeights(r.Weights)
	}

	if cs, ok := s.(interface{ SetConstraints([]strategy.Constraint) }); ok && len(r.Constraints) > 0 {
		cs.SetConstraints(r.Constraints)
	}

	if fs, ok := s.(interface{ SetFallbacks([]strategy.FallbackRule) }); ok && len(r.Fallbacks) > 0 {
		fs.SetFallbacks(r.Fallbacks)
	}

	return nil
}

func generateRecipeID(name string) string {
	return fmt.Sprintf("recipe-%s-%d", name, time.Now().UnixNano())
}
