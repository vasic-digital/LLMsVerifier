package suffix

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Parser handles parsing of model names with suffixes
type Parser struct{}

// NewParser creates a new suffix parser
func NewParser() *Parser {
	return &Parser{}
}

// Parse extracts the base name and suffixes from a model name
func (p *Parser) Parse(input string) (string, []string) {
	if input == "" {
		return "", []string{}
	}

	suffixes := []string{}
	baseName := input

	// Find all suffix patterns (text in parentheses)
	re := regexp.MustCompile(`\s*\(([^)]+)\)`)
	matches := re.FindAllStringSubmatchIndex(input, -1)

	if len(matches) > 0 {
		// Get base name (everything before first suffix)
		baseName = strings.TrimSpace(input[:matches[0][0]])

		// Extract suffixes
		for _, match := range matches {
			suffix := input[match[2]:match[3]]
			suffixes = append(suffixes, suffix)
		}
	}

	return baseName, suffixes
}

// Generator handles generation of model names with suffixes
type Generator struct{}

// NewGenerator creates a new suffix generator
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate creates a model name with the given suffixes
func (g *Generator) Generate(baseName string, suffixes map[string]bool) string {
	if len(suffixes) == 0 {
		return baseName
	}

	// Collect enabled suffixes and sort them
	var enabledSuffixes []string
	for suffix, enabled := range suffixes {
		if enabled {
			enabledSuffixes = append(enabledSuffixes, suffix)
		}
	}

	// Sort with llmsvd first, then alphabetically
	sort.Slice(enabledSuffixes, func(i, j int) bool {
		if enabledSuffixes[i] == "llmsvd" {
			return true
		}
		if enabledSuffixes[j] == "llmsvd" {
			return false
		}
		return enabledSuffixes[i] < enabledSuffixes[j]
	})

	// Build the output string
	result := baseName
	for _, suffix := range enabledSuffixes {
		result += " (" + suffix + ")"
	}

	return result
}

// Validator validates suffixes
type Validator struct{}

// NewValidator creates a new suffix validator
func NewValidator() *Validator {
	return &Validator{}
}

// Validate checks if a suffix is valid
func (v *Validator) Validate(suffix string) (bool, error) {
	suffix = strings.TrimSpace(suffix)
	if suffix == "" {
		return false, fmt.Errorf("empty suffix")
	}

	// Check for invalid characters
	invalidChars := regexp.MustCompile(`[@#$%^&*=+\[\]{}|\\<>]`)
	if invalidChars.MatchString(suffix) {
		return false, fmt.Errorf("invalid characters in suffix")
	}

	return true, nil
}

// Manager manages standard suffixes
type Manager struct {
	standardSuffixes map[string]suffixInfo
}

type suffixInfo struct {
	description string
	category    string
}

// NewManager creates a new suffix manager
func NewManager() *Manager {
	return &Manager{
		standardSuffixes: map[string]suffixInfo{
			"llmsvd":      {"LLM Suffix Verification Done", "verification"},
			"brotli":      {"Brotli compression support", "compression"},
			"http3":       {"HTTP/3 protocol support", "protocol"},
			"free to use": {"Free tier model", "pricing"},
			"open source": {"Open source model", "license"},
			"SC:8.5":      {"Score 8.5", "scoring"},
			"SC:9.0":      {"Score 9.0", "scoring"},
			"SC:7.5":      {"Score 7.5", "scoring"},
		},
	}
}

// IsStandardSuffix checks if a suffix is a standard suffix
func (m *Manager) IsStandardSuffix(suffix string) bool {
	// Handle scoring suffixes dynamically
	if strings.HasPrefix(suffix, "SC:") {
		return true
	}
	_, ok := m.standardSuffixes[suffix]
	return ok
}

// GetDescription returns the description of a suffix
func (m *Manager) GetDescription(suffix string) string {
	if strings.HasPrefix(suffix, "SC:") {
		return fmt.Sprintf("Score %s", strings.TrimPrefix(suffix, "SC:"))
	}
	if info, ok := m.standardSuffixes[suffix]; ok {
		return info.description
	}
	return ""
}

// GetCategory returns the category of a suffix
func (m *Manager) GetCategory(suffix string) string {
	if strings.HasPrefix(suffix, "SC:") {
		return "scoring"
	}
	if info, ok := m.standardSuffixes[suffix]; ok {
		return info.category
	}
	return ""
}

// Processor handles suffix operations
type Processor struct {
	parser    *Parser
	generator *Generator
}

// NewProcessor creates a new suffix processor
func NewProcessor() *Processor {
	return &Processor{
		parser:    NewParser(),
		generator: NewGenerator(),
	}
}

// Parse extracts base name and suffixes from input
func (p *Processor) Parse(input string) (string, []string) {
	return p.parser.Parse(input)
}

// AddSuffix adds a suffix to the model name
func (p *Processor) AddSuffix(input, suffix string) string {
	baseName, existingSuffixes := p.parser.Parse(input)

	// Create suffix map with existing suffixes
	suffixMap := make(map[string]bool)
	for _, s := range existingSuffixes {
		suffixMap[s] = true
	}

	// Add new suffix
	suffixMap[suffix] = true

	return p.generator.Generate(baseName, suffixMap)
}

// RemoveSuffix removes a suffix from the model name
func (p *Processor) RemoveSuffix(input, suffix string) string {
	baseName, existingSuffixes := p.parser.Parse(input)

	// Create suffix map without the removed suffix
	suffixMap := make(map[string]bool)
	for _, s := range existingSuffixes {
		if s != suffix {
			suffixMap[s] = true
		}
	}

	return p.generator.Generate(baseName, suffixMap)
}

// UpdateScore updates the score suffix in the model name
func (p *Processor) UpdateScore(input string, newScore float64) string {
	baseName, existingSuffixes := p.parser.Parse(input)

	// Create suffix map, replacing score suffix
	suffixMap := make(map[string]bool)
	for _, s := range existingSuffixes {
		if !strings.HasPrefix(s, "SC:") {
			suffixMap[s] = true
		}
	}

	// Add new score
	suffixMap[fmt.Sprintf("SC:%.1f", newScore)] = true

	return p.generator.Generate(baseName, suffixMap)
}

// Deduplicate removes duplicate suffixes
func (p *Processor) Deduplicate(input string) string {
	baseName, existingSuffixes := p.parser.Parse(input)

	// Use map to deduplicate
	suffixMap := make(map[string]bool)
	for _, s := range existingSuffixes {
		suffixMap[s] = true
	}

	return p.generator.Generate(baseName, suffixMap)
}

// Normalize normalizes the model name format
func (p *Processor) Normalize(input string) string {
	// Handle malformed parentheses by fixing nested patterns
	// Replace "( X (" with "(" and ")" with ") ("
	normalized := regexp.MustCompile(`\(\s*([^()]+)\s*\(`).ReplaceAllString(input, "($1) (")

	baseName, existingSuffixes := p.parser.Parse(normalized)

	suffixMap := make(map[string]bool)
	for _, s := range existingSuffixes {
		suffixMap[strings.TrimSpace(s)] = true
	}

	return p.generator.Generate(baseName, suffixMap)
}
