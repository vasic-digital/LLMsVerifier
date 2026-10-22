// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

// Package strategy provides provider usage categorization for fine-grained
// control over which LLM providers are used for specific purposes.
//
// This allows expensive vision-capable providers (like Gemini) to be
// reserved exclusively for vision tasks while cheaper providers handle
// general LLM chat tasks.
package strategy

// ProviderUsageType defines how a provider should be used.
type ProviderUsageType string

const (
	// UsageTypeAny means the provider can be used for any purpose.
	// This is the default behavior for most providers.
	UsageTypeAny ProviderUsageType = "any"

	// UsageTypeVisionOnly means the provider should ONLY be used for
	// vision-related tasks (screenshot analysis, image understanding).
	// Example: Gemini (expensive, excellent vision)
	UsageTypeVisionOnly ProviderUsageType = "vision_only"

	// UsageTypeLLMOnly means the provider should ONLY be used for
	// general LLM chat tasks (text generation, reasoning, planning).
	// These providers may lack vision capabilities or have cost
	// structures optimized for text.
	UsageTypeLLMOnly ProviderUsageType = "llm_only"
)

// IsValid checks if the usage type is valid.
func (t ProviderUsageType) IsValid() bool {
	switch t {
	case UsageTypeAny, UsageTypeVisionOnly, UsageTypeLLMOnly:
		return true
	}
	return false
}

// ProviderUsageConfig defines how providers should be categorized.
type ProviderUsageConfig struct {
	// ProviderUsages maps provider names to their usage types.
	// If a provider is not listed, it defaults to UsageTypeAny.
	//
	// Examples:
	//   "google":  UsageTypeVisionOnly  // Gemini - expensive, use only for vision
	//   "groq":    UsageTypeLLMOnly     // Fast, cheap text models
	//   "ollama":  UsageTypeAny         // Local, can do everything
	ProviderUsages map[string]ProviderUsageType `json:"provider_usages" yaml:"provider_usages"`

	// DefaultUsage is the fallback usage type for unlisted providers.
	// Defaults to UsageTypeAny.
	DefaultUsage ProviderUsageType `json:"default_usage" yaml:"default_usage"`

	// StrictMode when true will reject providers that don't have an
	// explicit usage type defined. When false (default), unlisted
	// providers use DefaultUsage.
	StrictMode bool `json:"strict_mode" yaml:"strict_mode"`
}

// DefaultProviderUsageConfig returns a sensible default configuration.
// By default:
//   - Gemini (Google) is vision-only due to cost
//   - All other providers are "any" (can be used for everything)
func DefaultProviderUsageConfig() *ProviderUsageConfig {
	return &ProviderUsageConfig{
		ProviderUsages: map[string]ProviderUsageType{
			// Gemini is expensive - reserve for vision tasks only
			"google": UsageTypeVisionOnly,
			"gemini": UsageTypeVisionOnly,

			// These providers are optimized for text/LLM tasks
			"groq":     UsageTypeLLMOnly,
			"cerebras": UsageTypeLLMOnly,
			"deepseek": UsageTypeLLMOnly,

			// Local providers can do everything
			"ollama":  UsageTypeAny,
			"ui-tars": UsageTypeAny,

			// These are good general-purpose providers
			"anthropic":  UsageTypeAny,
			"openai":     UsageTypeAny,
			"openrouter": UsageTypeAny,
		},
		DefaultUsage: UsageTypeAny,
		StrictMode:   false,
	}
}

// GetUsage returns the usage type for a provider.
func (c *ProviderUsageConfig) GetUsage(provider string) ProviderUsageType {
	if c == nil {
		return UsageTypeAny
	}

	// Normalize provider name (handle aliases)
	provider = normalizeProviderName(provider)

	// Check direct mapping
	if usage, ok := c.ProviderUsages[provider]; ok {
		return usage
	}

	// Check alternate names
	for name, usage := range c.ProviderUsages {
		if normalizeProviderName(name) == provider {
			return usage
		}
	}

	return c.DefaultUsage
}

// CanUseForVision checks if a provider can be used for vision tasks.
func (c *ProviderUsageConfig) CanUseForVision(provider string, supportsVision bool) bool {
	usage := c.GetUsage(provider)

	// Vision-only providers can do vision
	// Any-type providers can do vision if they support it
	switch usage {
	case UsageTypeVisionOnly:
		return true
	case UsageTypeAny:
		return supportsVision
	case UsageTypeLLMOnly:
		return false
	}
	return false
}

// CanUseForLLM checks if a provider can be used for general LLM tasks.
func (c *ProviderUsageConfig) CanUseForLLM(provider string) bool {
	usage := c.GetUsage(provider)

	// LLM-only providers can do LLM
	// Any-type providers can do LLM
	switch usage {
	case UsageTypeLLMOnly, UsageTypeAny:
		return true
	case UsageTypeVisionOnly:
		return false
	}
	return false
}

// GetVisionProviders returns a list of providers configured for vision use.
func (c *ProviderUsageConfig) GetVisionProviders() []string {
	var providers []string
	for name, usage := range c.ProviderUsages {
		if usage == UsageTypeVisionOnly || usage == UsageTypeAny {
			providers = append(providers, name)
		}
	}
	return providers
}

// GetLLMProviders returns a list of providers configured for LLM use.
func (c *ProviderUsageConfig) GetLLMProviders() []string {
	var providers []string
	for name, usage := range c.ProviderUsages {
		if usage == UsageTypeLLMOnly || usage == UsageTypeAny {
			providers = append(providers, name)
		}
	}
	return providers
}

// normalizeProviderName normalizes provider names for comparison.
func normalizeProviderName(name string) string {
	// Handle common aliases
	switch name {
	case "google", "gemini", "vertex", "vertexai":
		return "google"
	case "anthropic", "claude":
		return "anthropic"
	case "openai", "gpt", "chatgpt":
		return "openai"
	case "ollama", "local":
		return "ollama"
	}
	return name
}

// ProviderUsageRequirements extends Requirements with usage constraints.
type ProviderUsageRequirements struct {
	Requirements

	// RequireVisionProvider requires a provider specifically designated
	// for vision tasks (UsageTypeVisionOnly or UsageTypeAny with vision).
	RequireVisionProvider bool `json:"require_vision_provider" yaml:"require_vision_provider"`

	// RequireLLMProvider requires a provider specifically designated
	// for LLM tasks (UsageTypeLLMOnly or UsageTypeAny).
	RequireLLMProvider bool `json:"require_llm_provider" yaml:"require_llm_provider"`

	// UsageConfig is the provider usage configuration.
	// If nil, uses DefaultProviderUsageConfig().
	UsageConfig *ProviderUsageConfig `json:"usage_config" yaml:"usage_config"`
}

// GetUsageConfig returns the usage config or default.
func (r *ProviderUsageRequirements) GetUsageConfig() *ProviderUsageConfig {
	if r.UsageConfig != nil {
		return r.UsageConfig
	}
	return DefaultProviderUsageConfig()
}
