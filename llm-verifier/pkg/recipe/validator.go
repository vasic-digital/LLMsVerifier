// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package recipe

import (
	"fmt"
)

// RecipeValidator validates verification recipes
type RecipeValidator struct{}

// NewRecipeValidator creates a new recipe validator
func NewRecipeValidator() *RecipeValidator {
	return &RecipeValidator{}
}

// ValidateRecipe validates an entire recipe
func (v *RecipeValidator) ValidateRecipe(recipe *Recipe) error {
	if recipe == nil {
		return fmt.Errorf("recipe is nil")
	}

	if recipe.Name == "" {
		return fmt.Errorf("recipe name is required")
	}

	if recipe.Strategy == nil {
		return fmt.Errorf("recipe strategy is required")
	}

	if recipe.Timeout <= 0 {
		return fmt.Errorf("recipe timeout must be positive")
	}

	if recipe.MaxRetries < 0 {
		return fmt.Errorf("recipe max retries cannot be negative")
	}

	totalWeight := 0.0
	for _, w := range recipe.Weights {
		totalWeight += w
	}

	if totalWeight > 0 && (totalWeight < 0.99 || totalWeight > 1.01) {
		return fmt.Errorf("recipe weights must sum to 1.0 (got %.2f)", totalWeight)
	}

	return nil
}
