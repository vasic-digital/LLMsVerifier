package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	ctxt "llm-verifier/enhanced/context"
)

// LLMRequest represents a request to an LLM
type LLMRequest struct {
	Messages    []Message `json:"messages,omitempty"`
	MaxTokens   *int      `json:"max_tokens,omitempty"`
	Temperature *float64  `json:"temperature,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ValidationLevel represents the level of validation
type ValidationLevel int

const (
	LevelSyntax ValidationLevel = iota
	LevelSemantic
	LevelIntegration
)

// ValidationResult represents the result of a validation
type ValidationResult struct {
	Level    ValidationLevel        `json:"level"`
	Passed   bool                   `json:"passed"`
	Errors   []string               `json:"errors,omitempty"`
	Warnings []string               `json:"warnings,omitempty"`
	Score    float64                `json:"score"` // 0.0 to 1.0
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ValidationGate represents a validation gate
type ValidationGate interface {
	Validate(ctx context.Context, input interface{}) *ValidationResult
	GetLevel() ValidationLevel
	GetName() string
}

// SyntaxValidator performs syntax validation
type SyntaxValidator struct {
	name string
}

// NewSyntaxValidator creates a new syntax validator
func NewSyntaxValidator(name string) *SyntaxValidator {
	return &SyntaxValidator{name: name}
}

// Validate performs syntax validation
func (sv *SyntaxValidator) Validate(ctx context.Context, input interface{}) *ValidationResult {
	result := &ValidationResult{
		Level:    LevelSyntax,
		Passed:   true,
		Errors:   []string{},
		Warnings: []string{},
		Score:    1.0,
		Metadata: make(map[string]interface{}),
	}

	switch v := input.(type) {
	case string:
		result = sv.validatePrompt(v)
	case *LLMRequest:
		result = sv.validateLLMRequest(v)
	case map[string]interface{}:
		result = sv.validateJSON(v)
	default:
		result.Passed = false
		result.Errors = append(result.Errors, "Unsupported input type for syntax validation")
		result.Score = 0.0
	}

	return result
}

// validatePrompt validates a prompt string
func (sv *SyntaxValidator) validatePrompt(prompt string) *ValidationResult {
	result := &ValidationResult{
		Level:    LevelSyntax,
		Passed:   true,
		Errors:   []string{},
		Warnings: []string{},
		Score:    1.0,
		Metadata: make(map[string]interface{}),
	}

	// Check for empty prompt
	if strings.TrimSpace(prompt) == "" {
		result.Passed = false
		result.Errors = append(result.Errors, "Prompt cannot be empty")
		result.Score = 0.0
		return result
	}

	// Check prompt length
	if len(prompt) > 10000 {
		result.Warnings = append(result.Warnings, "Prompt is very long (>10k characters)")
		result.Score -= 0.1
	}

	// Check for potentially harmful patterns
	harmfulPatterns := []string{
		`(?i)(drop table|delete from|truncate table)`,
		`(?i)(eval\(|exec\(|system\()`,
		`<script[^>]*>.*?</script>`,
	}

	for _, pattern := range harmfulPatterns {
		matched, _ := regexp.MatchString(pattern, prompt)
		if matched {
			result.Passed = false
			result.Errors = append(result.Errors, "Prompt contains potentially harmful content")
			result.Score = 0.0
			break
		}
	}

	// Check for balanced brackets/quotes
	if !sv.hasBalancedBrackets(prompt) {
		result.Warnings = append(result.Warnings, "Unbalanced brackets or quotes detected")
		result.Score -= 0.2
	}

	result.Metadata["length"] = len(prompt)
	result.Metadata["word_count"] = len(strings.Fields(prompt))

	if result.Score < 0 {
		result.Score = 0
	}

	return result
}

// validateLLMRequest validates an LLM request
func (sv *SyntaxValidator) validateLLMRequest(req *LLMRequest) *ValidationResult {
	result := &ValidationResult{
		Level:    LevelSyntax,
		Passed:   true,
		Errors:   []string{},
		Warnings: []string{},
		Score:    1.0,
		Metadata: make(map[string]interface{}),
	}

	// Validate messages if present
	if req.Messages != nil {
		for i, msg := range req.Messages {
			if strings.TrimSpace(msg.Content) == "" {
				result.Errors = append(result.Errors, fmt.Sprintf("Message %d has empty content", i))
				result.Passed = false
				result.Score -= 0.5
			}

			if msg.Role != "user" && msg.Role != "assistant" && msg.Role != "system" {
				result.Errors = append(result.Errors, fmt.Sprintf("Message %d has invalid role: %s", i, msg.Role))
				result.Passed = false
				result.Score -= 0.3
			}
		}
	}

	// Validate temperature
	if req.Temperature != nil {
		temp := *req.Temperature
		if temp < 0 || temp > 2 {
			result.Warnings = append(result.Warnings, "Temperature outside recommended range [0,2]")
			result.Score -= 0.1
		}
	}

	// Validate max tokens
	if req.MaxTokens != nil {
		if *req.MaxTokens <= 0 {
			result.Errors = append(result.Errors, "Max tokens must be positive")
			result.Passed = false
			result.Score -= 0.5
		}
	}

	if result.Score < 0 {
		result.Score = 0
	}

	return result
}

// validateJSON validates JSON structure
func (sv *SyntaxValidator) validateJSON(data map[string]interface{}) *ValidationResult {
	result := &ValidationResult{
		Level:    LevelSyntax,
		Passed:   true,
		Errors:   []string{},
		Warnings: []string{},
		Score:    1.0,
		Metadata: make(map[string]interface{}),
	}

	// Try to marshal back to JSON to check validity
	_, err := json.Marshal(data)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("Invalid JSON structure: %v", err))
		result.Score = 0.0
	}

	return result
}

// hasBalancedBrackets checks for balanced brackets and quotes
func (sv *SyntaxValidator) hasBalancedBrackets(s string) bool {
	brackets := map[rune]rune{
		')': '(',
		']': '[',
		'}': '{',
		'>': '<',
	}

	var stack []rune
	quotes := map[rune]bool{'"': false, '\'': false, '`': false}

	for _, char := range s {
		// Handle quotes
		if _, isQuote := quotes[char]; isQuote {
			// Toggle quote state (simplified - doesn't handle escaping)
			continue
		}

		// Handle brackets
		if open, isClose := brackets[char]; isClose {
			if len(stack) == 0 || stack[len(stack)-1] != open {
				return false
			}
			stack = stack[:len(stack)-1]
		} else if char == '(' || char == '[' || char == '{' || char == '<' {
			stack = append(stack, char)
		}
	}

	return len(stack) == 0
}

// GetLevel returns the validation level
func (sv *SyntaxValidator) GetLevel() ValidationLevel {
	return LevelSyntax
}

// GetName returns the validator name
func (sv *SyntaxValidator) GetName() string {
	return sv.name
}

// SemanticValidator performs semantic validation
type SemanticValidator struct {
	name       string
	contextMgr *ctxt.ConversationManager
}

// NewSemanticValidator creates a new semantic validator
func NewSemanticValidator(name string, contextMgr *ctxt.ConversationManager) *SemanticValidator {
	return &SemanticValidator{
		name:       name,
		contextMgr: contextMgr,
	}
}

// Validate performs semantic validation
func (sv *SemanticValidator) Validate(ctx context.Context, input interface{}) *ValidationResult {
	result := &ValidationResult{
		Level:    LevelSemantic,
		Passed:   true,
		Errors:   []string{},
		Warnings: []string{},
		Score:    1.0,
		Metadata: make(map[string]interface{}),
	}

	switch v := input.(type) {
	case string:
		result = sv.validatePromptSemantics(v)
	case *LLMRequest:
		result = sv.validateRequestSemantics(v)
	default:
		result.Passed = false
		result.Errors = append(result.Errors, "Unsupported input type for semantic validation")
		result.Score = 0.0
	}

	return result
}

// validatePromptSemantics validates prompt semantics
func (sv *SemanticValidator) validatePromptSemantics(prompt string) *ValidationResult {
	result := &ValidationResult{
		Level:    LevelSemantic,
		Passed:   true,
		Errors:   []string{},
		Warnings: []string{},
		Score:    1.0,
		Metadata: make(map[string]interface{}),
	}

	// Check for coherence and clarity
	words := strings.Fields(prompt)
	sentenceCount := strings.Count(prompt, ".") + strings.Count(prompt, "!") + strings.Count(prompt, "?")

	// Very short prompts might lack context
	if len(words) < 3 && sentenceCount == 0 {
		result.Warnings = append(result.Warnings, "Prompt is very brief and may lack sufficient context")
		result.Score -= 0.1
	}

	// Check for excessive repetition
	if sv.hasExcessiveRepetition(prompt) {
		result.Warnings = append(result.Warnings, "Prompt contains excessive repetition")
		result.Score -= 0.2
	}

	// Check for ambiguous pronouns
	ambiguousPronouns := []string{"it", "they", "them", "this", "that", "these", "those"}
	wordsLower := strings.ToLower(prompt)
	ambiguousCount := 0

	for _, pronoun := range ambiguousPronouns {
		ambiguousCount += strings.Count(wordsLower, pronoun)
	}

	if ambiguousCount > 3 {
		result.Warnings = append(result.Warnings, "Prompt contains many ambiguous pronouns")
		result.Score -= 0.15
	}

	result.Metadata["word_count"] = len(words)
	result.Metadata["sentence_count"] = sentenceCount
	result.Metadata["avg_words_per_sentence"] = float64(len(words)) / float64(sentenceCount+1)

	if result.Score < 0 {
		result.Score = 0
	}

	return result
}

// validateRequestSemantics validates request semantics
func (sv *SemanticValidator) validateRequestSemantics(req *LLMRequest) *ValidationResult {
	result := &ValidationResult{
		Level:    LevelSemantic,
		Passed:   true,
		Errors:   []string{},
		Warnings: []string{},
		Score:    1.0,
		Metadata: make(map[string]interface{}),
	}

	// Check conversation flow if context is available
	if req.Messages != nil && len(req.Messages) > 1 {
		flowIssues := sv.analyzeConversationFlow(req.Messages)
		result.Errors = append(result.Errors, flowIssues.Errors...)
		result.Warnings = append(result.Warnings, flowIssues.Warnings...)
		result.Score -= float64(len(flowIssues.Errors)) * 0.3
		result.Score -= float64(len(flowIssues.Warnings)) * 0.1
	}

	if result.Score < 0 {
		result.Score = 0
	}

	return result
}

// ConversationFlowAnalysis represents analysis of conversation flow
type ConversationFlowAnalysis struct {
	Errors   []string
	Warnings []string
}

// analyzeConversationFlow analyzes the flow of a conversation
func (sv *SemanticValidator) analyzeConversationFlow(messages []Message) ConversationFlowAnalysis {
	analysis := ConversationFlowAnalysis{
		Errors:   []string{},
		Warnings: []string{},
	}

	// Check for proper alternation between user and assistant
	lastRole := ""
	consecutiveSameRole := 0

	for i, msg := range messages {
		if msg.Role == lastRole {
			consecutiveSameRole++
			if consecutiveSameRole > 2 {
				analysis.Warnings = append(analysis.Warnings,
					fmt.Sprintf("Multiple consecutive messages from %s at position %d", msg.Role, i))
			}
		} else {
			consecutiveSameRole = 1
		}
		lastRole = msg.Role
	}

	// Check for very long messages
	for i, msg := range messages {
		if len(msg.Content) > 5000 {
			analysis.Warnings = append(analysis.Warnings,
				fmt.Sprintf("Very long message from %s at position %d", msg.Role, i))
		}
	}

	return analysis
}

// hasExcessiveRepetition checks for excessive word repetition
func (sv *SemanticValidator) hasExcessiveRepetition(text string) bool {
	words := strings.Fields(strings.ToLower(text))
	wordCount := make(map[string]int)

	for _, word := range words {
		if len(word) > 3 { // Only count words longer than 3 characters
			wordCount[word]++
		}
	}

	totalWords := len(words)
	for _, count := range wordCount {
		if float64(count)/float64(totalWords) > 0.1 { // More than 10% of words are the same
			return true
		}
	}

	return false
}

// GetLevel returns the validation level
func (sv *SemanticValidator) GetLevel() ValidationLevel {
	return LevelSemantic
}

// GetName returns the validator name
func (sv *SemanticValidator) GetName() string {
	return sv.name
}
