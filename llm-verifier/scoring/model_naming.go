package scoring

import (
	"fmt"
	"regexp"
	"strings"
)

// ModelNaming handles the addition and management of score suffixes in model names
type ModelNaming struct {
	scoreSuffixPattern *regexp.Regexp
}

// NewModelNaming creates a new ModelNaming instance
func NewModelNaming() *ModelNaming {
	// Pattern to match existing score suffixes like (SC:9.5), (SC:8.2), etc.
	pattern := regexp.MustCompile(`\s*\(SC:\d+\.\d+\)\s*$`)
	
	return &ModelNaming{
		scoreSuffixPattern: pattern,
	}
}

// AddScoreSuffix adds or updates the score suffix in a model name
func (mn *ModelNaming) AddScoreSuffix(modelName string, score float64) string {
	// Remove any existing score suffix
	cleanName := mn.RemoveScoreSuffix(modelName)
	
	// Generate new score suffix
	suffix := mn.GenerateScoreSuffix(score)
	
	// Combine clean name with new suffix
	return fmt.Sprintf("%s %s", strings.TrimSpace(cleanName), suffix)
}

// RemoveScoreSuffix removes any existing score suffix from a model name
func (mn *ModelNaming) RemoveScoreSuffix(modelName string) string {
	return mn.scoreSuffixPattern.ReplaceAllString(modelName, "")
}

// GenerateScoreSuffix generates a score suffix based on the given score
func (mn *ModelNaming) GenerateScoreSuffix(score float64) string {
	// Round to one decimal place as specified
	roundedScore := fmt.Sprintf("%.1f", score)
	return fmt.Sprintf("(SC:%s)", roundedScore)
}

// ExtractScoreFromName extracts the score from a model name if present
func (mn *ModelNaming) ExtractScoreFromName(modelName string) (float64, bool) {
	// Look for score suffix pattern
	matches := mn.scoreSuffixPattern.FindStringSubmatch(modelName)
	if len(matches) == 0 {
		return 0, false
	}
	
	// Extract the numeric score from the match
	scoreStr := strings.TrimSpace(matches[0])
	scoreStr = strings.TrimPrefix(scoreStr, "(SC:")
	scoreStr = strings.TrimSuffix(scoreStr, ")")
	scoreStr = strings.TrimSpace(scoreStr)
	
	var score float64
	_, err := fmt.Sscanf(scoreStr, "%f", &score)
	if err != nil {
		return 0, false
	}
	
	return score, true
}

// HasScoreSuffix checks if a model name already has a score suffix
func (mn *ModelNaming) HasScoreSuffix(modelName string) bool {
	return mn.scoreSuffixPattern.MatchString(modelName)
}

// UpdateModelNameWithScore updates a model name with a new score, handling existing suffixes
func (mn *ModelNaming) UpdateModelNameWithScore(modelName string, newScore float64) string {
	return mn.AddScoreSuffix(modelName, newScore)
}

// ValidateScoreSuffix validates that a score suffix is properly formatted
func (mn *ModelNaming) ValidateScoreSuffix(suffix string) bool {
	// Check if it matches our expected pattern
	pattern := regexp.MustCompile(`^\s*\(SC:\d+\.\d+\)\s*$`)
	return pattern.MatchString(suffix)
}

// GetScoreSuffixFormat returns the expected format for score suffixes
func (mn *ModelNaming) GetScoreSuffixFormat() string {
	return "(SC:X.X) where X.X is a score from 0.0 to 10.0"
}

// BatchUpdateModelNames updates multiple model names with their respective scores
func (mn *ModelNaming) BatchUpdateModelNames(modelScores map[string]float64) map[string]string {
	results := make(map[string]string)
	
	for modelName, score := range modelScores {
		updatedName := mn.UpdateModelNameWithScore(modelName, score)
		results[modelName] = updatedName
	}
	
	return results
}

// NormalizeModelName normalizes a model name by ensuring consistent formatting
func (mn *ModelNaming) NormalizeModelName(modelName string) string {
	// Remove extra whitespace
	modelName = strings.Join(strings.Fields(modelName), " ")
	
	// Ensure proper capitalization (optional, could be configurable)
	// modelName = strings.Title(strings.ToLower(modelName))
	
	return modelName
}

// CompareModelNames compares two model names, ignoring score suffixes
func (mn *ModelNaming) CompareModelNames(name1, name2 string) bool {
	cleanName1 := mn.RemoveScoreSuffix(name1)
	cleanName2 := mn.RemoveScoreSuffix(name2)
	
	return strings.EqualFold(cleanName1, cleanName2)
}

// ExtractBaseName extracts the base model name without any score suffix
func (mn *ModelNaming) ExtractBaseName(modelName string) string {
	return mn.RemoveScoreSuffix(modelName)
}

// GenerateScoreSuffixWithConfidence generates a score suffix with confidence indicator
func (mn *ModelNaming) GenerateScoreSuffixWithConfidence(score float64, confidence float64) string {
	baseSuffix := mn.GenerateScoreSuffix(score)
	
	// Add confidence indicator
	var confidenceIndicator string
	switch {
	case confidence >= 0.9:
		confidenceIndicator = "★" // High confidence
	case confidence >= 0.7:
		confidenceIndicator = "◆" // Medium confidence
	case confidence >= 0.5:
		confidenceIndicator = "▲" // Low confidence
	default:
		confidenceIndicator = "?" // Very low confidence
	}
	
	return fmt.Sprintf("%s%s", baseSuffix, confidenceIndicator)
}

// ParseScoreSuffix parses a score suffix and extracts the score and confidence
func (mn *ModelNaming) ParseScoreSuffix(suffix string) (score float64, confidence string, valid bool) {
	// Remove confidence indicator if present
	confidenceIndicators := []string{"★", "◆", "▲", "?"}
	baseSuffix := suffix
	
	for _, indicator := range confidenceIndicators {
		if strings.HasSuffix(suffix, indicator) {
			confidence = indicator
			baseSuffix = strings.TrimSuffix(suffix, indicator)
			break
		}
	}
	
	// Extract score from base suffix
	s, valid := mn.ExtractScoreFromName("model " + baseSuffix)
	if !valid {
		return 0, "", false
	}
	
	return s, confidence, true
}

// ScoreSuffixFormatter provides advanced formatting options for score suffixes
type ScoreSuffixFormatter struct {
	modelNaming *ModelNaming
}

// NewScoreSuffixFormatter creates a new formatter
func NewScoreSuffixFormatter() *ScoreSuffixFormatter {
	return &ScoreSuffixFormatter{
		modelNaming: NewModelNaming(),
	}
}

// FormatWithColor returns a colored score suffix based on score range
func (sf *ScoreSuffixFormatter) FormatWithColor(score float64) string {
	suffix := sf.modelNaming.GenerateScoreSuffix(score)
	
	// Add ANSI color codes based on score
	var colorCode string
	switch {
	case score >= 9.0:
		colorCode = "\033[32m" // Green
	case score >= 7.0:
		colorCode = "\033[33m" // Yellow
	case score >= 5.0:
		colorCode = "\033[35m" // Magenta
	default:
		colorCode = "\033[31m" // Red
	}
	
	resetCode := "\033[0m"
	return fmt.Sprintf("%s%s%s", colorCode, suffix, resetCode)
}

// FormatWithDescription returns a score suffix with descriptive text
func (sf *ScoreSuffixFormatter) FormatWithDescription(score float64) string {
	suffix := sf.modelNaming.GenerateScoreSuffix(score)
	
	var description string
	switch {
	case score >= 9.0:
		description = "Exceptional"
	case score >= 8.0:
		description = "Excellent"
	case score >= 7.0:
		description = "Very Good"
	case score >= 6.0:
		description = "Good"
	case score >= 5.0:
		description = "Average"
	case score >= 4.0:
		description = "Below Average"
	case score >= 3.0:
		description = "Poor"
	default:
		description = "Unacceptable"
	}
	
	return fmt.Sprintf("%s [%s]", suffix, description)
}

// FormatForDisplay returns a formatted score suffix for display purposes
func (sf *ScoreSuffixFormatter) FormatForDisplay(score float64, includeDescription bool) string {
	if includeDescription {
		return sf.FormatWithDescription(score)
	}
	return sf.modelNaming.GenerateScoreSuffix(score)
}