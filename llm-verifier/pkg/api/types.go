// Package api defines the canonical data types for LLMsVerifier.
// These types are the single source of truth for model verification
// metadata shared across the Helix ecosystem.
package api

import "time"

// Model represents a verified LLM model.
type Model struct {
	ID                   string          `json:"id"`
	ProviderID           string          `json:"provider_id"`
	Name                 string          `json:"name"`
	VerificationStatus   string          `json:"verification_status"`
	CanSeeCode           bool            `json:"can_see_code"`
	AffirmativeResponse  bool            `json:"affirmative_response"`
	OverallScore         float64         `json:"overall_score"`
	ResponsivenessScore  float64         `json:"responsiveness_score"`
	CodeCapabilityScore  float64         `json:"code_capability_score"`
	FeatureRichnessScore float64         `json:"feature_richness_score"`
	ReliabilityScore     float64         `json:"reliability_score"`
	Capabilities         map[string]bool `json:"capabilities"`
	Pricing              PricingInfo     `json:"pricing"`
	LastVerifiedAt       time.Time       `json:"last_verified_at"`
}

// PricingInfo holds cost metadata for a model.
type PricingInfo struct {
	InputTokenCost  float64 `json:"input_token_cost"`
	OutputTokenCost float64 `json:"output_token_cost"`
	Currency        string  `json:"currency"`
}
