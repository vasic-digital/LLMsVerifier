package scoring

import (
	"fmt"
	"strings"
)

// ModelDisplayName handles formatting model names with feature suffixes
type ModelDisplayName struct {
	naming *ModelNaming
}

// NewModelDisplayName creates a new ModelDisplayName formatter
func NewModelDisplayName() *ModelDisplayName {
	return &ModelDisplayName{
		naming: NewModelNaming(),
	}
}

// FormatWithFeatureSuffixes formats a model name with feature-based suffixes
// Adds: (brotli), (http3), (toon), (free to use), (open source), etc.
func (md *ModelDisplayName) FormatWithFeatureSuffixes(modelName string, modelInfo interface{}) string {
	return md.FormatWithFeatureSuffixesAndLLMsVerifier(modelName, modelInfo)
}

// FormatWithFeatureSuffixesAndLLMsVerifier formats a model name with feature-based suffixes and mandatory LLMsVerifier suffix
// Adds: (brotli), (http3), (toon), (free to use), (open source), etc. + (llmsvd) as the final suffix
func (md *ModelDisplayName) FormatWithFeatureSuffixesAndLLMsVerifier(modelName string, modelInfo interface{}) string {
	// Remove any existing suffixes first
	cleanName := md.naming.RemoveScoreSuffix(modelName)
	cleanName = md.removeFeatureSuffixes(cleanName)
	
	// Build suffix list
	var suffixes []string
	
	// Extract features based on modelInfo type (map or struct)
	features := md.extractFeatures(modelInfo)
	
	// Add feature-based suffixes in consistent order
	// Order: technical features, cost, licensing, performance
	if features.supportsBrotli {
		suffixes = append(suffixes, "(brotli)")
	}
	
	if features.supportsHTTP3 {
		suffixes = append(suffixes, "(http3)")
	}
	
	if features.supportsToon {
		suffixes = append(suffixes, "(toon)")
	}
	
	if features.isOpenSource {
		suffixes = append(suffixes, "(open source)")
	}
	
	if features.isFree {
		suffixes = append(suffixes, "(free to use)")
	}
	
	if features.isFast {
		suffixes = append(suffixes, "(fast)")
	}
	
	// Add mandatory LLMsVerifier suffix as the final suffix
	suffixes = append(suffixes, "(llmsvd)")
	
	// Build final name
	return fmt.Sprintf("%s %s", cleanName, strings.Join(suffixes, " "))
}

// ModelFeatures contains feature flags for a model
type ModelFeatures struct {
	supportsBrotli  bool
	supportsHTTP3   bool
	supportsToon    bool
	isFree          bool
	isOpenSource    bool
	isFast          bool
	responseTime    float64 // in milliseconds
	costPer1MInput  float64
	costPer1MOutput float64
}

// extractFeatures extracts feature information from model data
func (md *ModelDisplayName) extractFeatures(modelInfo interface{}) ModelFeatures {
	features := ModelFeatures{}
	
	// Handle map[string]interface{} type
	if modelMap, ok := modelInfo.(map[string]interface{}); ok {
		// Check for boolean flags
		if val, exists := modelMap["supports_brotli"]; exists {
			if boolVal, ok := val.(bool); ok {
				features.supportsBrotli = boolVal
			}
		}
		
		if val, exists := modelMap["supports_http3"]; exists {
			if boolVal, ok := val.(bool); ok {
				features.supportsHTTP3 = boolVal
			}
		}
		
		if val, exists := modelMap["supports_toon"]; exists {
			if boolVal, ok := val.(bool); ok {
				features.supportsToon = boolVal
			}
		}
		
		if val, exists := modelMap["open_weights"]; exists {
			if boolVal, ok := val.(bool); ok {
				features.isOpenSource = boolVal
			}
		}
		
		// Check for cost information
		if cost, exists := modelMap["cost"]; exists {
			if costMap, ok := cost.(map[string]interface{}); ok {
				if input, exists := costMap["input"]; exists {
					if inputFloat, ok := input.(float64); ok {
						features.costPer1MInput = inputFloat
						// Consider free if both input and output are 0
						if output, exists := costMap["output"]; exists {
							if outputFloat, ok := output.(float64); ok {
								features.costPer1MOutput = outputFloat
								features.isFree = inputFloat == 0 && outputFloat == 0
							}
						}
					}
				}
			}
		}
		
		// Check response time for "fast" flag
		if val, exists := modelMap["response_time_ms"]; exists {
			if responseTime, ok := val.(float64); ok {
				features.responseTime = responseTime
				features.isFast = responseTime < 1000 // < 1 second
			}
		}
	}
	
	// Handle struct type (if needed in future)
	// Add struct field extraction here
	
	return features
}

// removeFeatureSuffixes removes known feature suffixes from model names
func (md *ModelDisplayName) removeFeatureSuffixes(name string) string {
	// List of suffix patterns to remove
	suffixes := []string{
		"(brotli)", "(http3)", "(toon)", "(free to use)",
		"(open source)", "(fast)", "(optimized)", "(premium)",
		"(experimental)", "(beta)", "(prod)", "(stable)",
		"(deprecated)", "(legacy)", "(llmsvd)",
	}
	
	for _, suffix := range suffixes {
		name = strings.ReplaceAll(name, " "+suffix, "")
		// Also handle case where it's the only suffix
		if strings.HasSuffix(name, suffix) {
			name = strings.TrimSuffix(name, suffix)
			name = strings.TrimSpace(name)
		}
	}
	
	return name
}

// GetAllFeatureSuffixes returns all possible feature suffixes
func GetAllFeatureSuffixes() []string {
	return []string{
		"(brotli)",
		"(http3)",
		"(toon)",
		"(free to use)",
		"(open source)",
		"(fast)",
		"(optimized)",
		"(premium)",
		"(experimental)",
		"(beta)",
		"(prod)",
		"(stable)",
		"(deprecated)",
		"(legacy)",
		"(llmsvd)",
	}
}

// FormatModelNameWithScoreAndFeatures formats a model name with both score and feature suffixes
func (md *ModelDisplayName) FormatModelNameWithScoreAndFeatures(
	modelName string, 
	score float64, 
	modelInfo interface{},
	includeScore bool,
) string {
	// Start with feature formatting
	nameWithFeatures := md.FormatWithFeatureSuffixes(modelName, modelInfo)
	
	// Add score suffix if requested
	if includeScore {
		return md.naming.AddScoreSuffix(nameWithFeatures, score)
	}
	
	return nameWithFeatures
}

// ParseFeatureSuffixes extracts feature suffixes from a model name
func (md *ModelDisplayName) ParseFeatureSuffixes(modelName string) []string {
	var found []string
	allSuffixes := GetAllFeatureSuffixes()
	
	for _, suffix := range allSuffixes {
		if strings.Contains(modelName, suffix) {
			found = append(found, suffix)
		}
	}
	
	return found
}

// HasFeatureSuffix checks if a model name has specific feature suffixes
func (md *ModelDisplayName) HasFeatureSuffix(modelName string, suffixes ...string) bool {
	allFeatures := md.ParseFeatureSuffixes(modelName)
	
	for _, toCheck := range suffixes {
		for _, found := range allFeatures {
			if found == toCheck {
				return true
			}
		}
	}
	
	return false
}

// ValidateFeatureSuffix validates that a suffix is in the allowed list
func (md *ModelDisplayName) ValidateFeatureSuffix(suffix string) (valid bool, suggestion string) {
	allSuffixes := GetAllFeatureSuffixes()
	
	for _, validSuffix := range allSuffixes {
		if suffix == validSuffix {
			return true, ""
		}
	}
	
	// If not found, return false with suggested format
	return false, "Unknown suffix. Use one of: " + strings.Join(allSuffixes, ", ")
}