package api

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// ContentFilter provides content filtering capabilities
type ContentFilter struct {
	bannedWords    map[string]bool
	bannedPatterns []*regexp.Regexp
	toxicityWords  map[string]bool
}

// NewContentFilter creates a new content filter
func NewContentFilter() *ContentFilter {
	cf := &ContentFilter{
		bannedWords:    make(map[string]bool),
		bannedPatterns: []*regexp.Regexp{},
		toxicityWords:  make(map[string]bool),
	}

	cf.initializeFilters()
	return cf
}

// initializeFilters sets up default banned words and patterns
func (cf *ContentFilter) initializeFilters() {
	// Banned words (basic list - would be more comprehensive in production)
	bannedWords := []string{
		"inappropriate", "offensive", "harmful", "dangerous",
		"illegal", "prohibited", "forbidden", "banned",
		// Add more as needed
	}

	for _, word := range bannedWords {
		cf.bannedWords[strings.ToLower(word)] = true
	}

	// Banned patterns
	patterns := []string{
		// Email-like patterns that might be spam
		`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,
		// Phone number patterns
		`\+?\d{1,3}?[-.\s]?\(?(\d{3})\)?[-.\s]?(\d{3})[-.\s]?(\d{4})`,
		// URLs
		`https?://[^\s]+`,
		// Potential injection patterns
		`<script[^>]*>.*?</script>`,
		`<iframe[^>]*>.*?</iframe>`,
		`javascript:[^\s"']*`,
	}

	for _, pattern := range patterns {
		if re, err := regexp.Compile(pattern); err == nil {
			cf.bannedPatterns = append(cf.bannedPatterns, re)
		}
	}

	// Toxicity words (basic set)
	toxicityWords := []string{
		"hate", "violence", "abuse", "threat", "harassment",
		"discrimination", "bigotry", "intolerance",
		// Add more as needed
	}

	for _, word := range toxicityWords {
		cf.toxicityWords[strings.ToLower(word)] = true
	}
}

// AddBannedWord adds a word to the banned list
func (cf *ContentFilter) AddBannedWord(word string) {
	cf.bannedWords[strings.ToLower(word)] = true
}

// AddBannedPattern adds a regex pattern to the banned list
func (cf *ContentFilter) AddBannedPattern(pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %w", err)
	}
	cf.bannedPatterns = append(cf.bannedPatterns, re)
	return nil
}

// FilterContent filters content and returns filtered version and any violations
func (cf *ContentFilter) FilterContent(content string) (*FilteredContent, error) {
	result := &FilteredContent{
		OriginalContent: content,
		IsAllowed:       true,
		Violations:      []ContentViolation{},
	}

	// Convert to lowercase for case-insensitive matching
	lowerContent := strings.ToLower(content)

	// Check for banned words
	words := strings.Fields(lowerContent)
	for _, word := range words {
		// Remove punctuation for word matching
		cleanWord := strings.TrimFunc(word, func(r rune) bool {
			return !unicode.IsLetter(r)
		})

		if cf.bannedWords[cleanWord] {
			result.IsAllowed = false
			result.Violations = append(result.Violations, ContentViolation{
				Type:        "banned_word",
				Description: fmt.Sprintf("Contains banned word: %s", cleanWord),
				Severity:    "high",
			})
		}

		if cf.toxicityWords[cleanWord] {
			result.IsAllowed = false
			result.Violations = append(result.Violations, ContentViolation{
				Type:        "toxicity",
				Description: fmt.Sprintf("Contains potentially toxic content: %s", cleanWord),
				Severity:    "medium",
			})
		}
	}

	// Check for banned patterns
	for _, pattern := range cf.bannedPatterns {
		if pattern.MatchString(content) {
			result.IsAllowed = false
			result.Violations = append(result.Violations, ContentViolation{
				Type:        "pattern_match",
				Description: fmt.Sprintf("Matches banned pattern: %s", pattern.String()),
				Severity:    "high",
			})
		}
	}

	// Apply content filtering (censorship)
	result.FilteredContent = cf.censorContent(content)

	// Calculate risk score
	result.RiskScore = cf.calculateRiskScore(result.Violations)

	return result, nil
}

// censorContent censors inappropriate content
func (cf *ContentFilter) censorContent(content string) string {
	filtered := content

	// Censor banned words
	for word := range cf.bannedWords {
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(word) + `\b`)
		filtered = re.ReplaceAllStringFunc(filtered, func(match string) string {
			return strings.Repeat("*", len(match))
		})
	}

	// Censor patterns
	for _, pattern := range cf.bannedPatterns {
		filtered = pattern.ReplaceAllString(filtered, "[FILTERED]")
	}

	return filtered
}

// calculateRiskScore calculates a risk score based on violations
func (cf *ContentFilter) calculateRiskScore(violations []ContentViolation) float64 {
	if len(violations) == 0 {
		return 0.0
	}

	score := 0.0
	for _, violation := range violations {
		switch violation.Severity {
		case "low":
			score += 0.2
		case "medium":
			score += 0.5
		case "high":
			score += 1.0
		case "critical":
			score += 2.0
		}
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// CheckToxicity checks for toxic content and returns a toxicity score
func (cf *ContentFilter) CheckToxicity(content string) *ToxicityResult {
	result := &ToxicityResult{
		Score:      0.0,
		IsToxic:    false,
		ToxicWords: []string{},
	}

	lowerContent := strings.ToLower(content)
	words := strings.Fields(lowerContent)

	for _, word := range words {
		cleanWord := strings.TrimFunc(word, func(r rune) bool {
			return !unicode.IsLetter(r)
		})

		if cf.toxicityWords[cleanWord] {
			result.IsToxic = true
			result.Score += 0.3
			result.ToxicWords = append(result.ToxicWords, cleanWord)
		}
	}

	// Additional checks for patterns
	if strings.Contains(lowerContent, "threat") || strings.Contains(lowerContent, "harm") {
		result.Score += 0.2
		result.IsToxic = true
	}

	// Cap score
	if result.Score > 1.0 {
		result.Score = 1.0
	}

	return result
}

// FilteredContent represents the result of content filtering
type FilteredContent struct {
	OriginalContent string             `json:"original_content"`
	FilteredContent string             `json:"filtered_content"`
	IsAllowed       bool               `json:"is_allowed"`
	Violations      []ContentViolation `json:"violations,omitempty"`
	RiskScore       float64            `json:"risk_score"`
}

// ContentViolation represents a content violation
type ContentViolation struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

// ToxicityResult represents the result of toxicity checking
type ToxicityResult struct {
	Score      float64  `json:"score"`
	IsToxic    bool     `json:"is_toxic"`
	ToxicWords []string `json:"toxic_words,omitempty"`
}
