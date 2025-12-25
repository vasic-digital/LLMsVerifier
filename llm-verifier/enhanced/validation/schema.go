package validation

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// SchemaValidator provides JSON schema validation for API requests and responses
type SchemaValidator struct {
	validate *validator.Validate
	schemas  map[string]interface{}
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator() *SchemaValidator {
	validate := validator.New()

	// Register custom validation tags
	validate.RegisterValidation("llm_model", validateLLMModel)
	validate.RegisterValidation("provider_name", validateProviderName)
	validate.RegisterValidation("safe_prompt", validateSafePrompt)

	return &SchemaValidator{
		validate: validate,
		schemas:  make(map[string]interface{}),
	}
}

// RegisterSchema registers a JSON schema for validation
func (sv *SchemaValidator) RegisterSchema(name string, schema interface{}) {
	sv.schemas[name] = schema
}

// ValidateStruct validates a struct using struct tags
func (sv *SchemaValidator) ValidateStruct(s interface{}) error {
	return sv.validate.Struct(s)
}

// ValidateJSON validates JSON data against a registered schema
func (sv *SchemaValidator) ValidateJSON(data []byte, schemaName string) error {
	schema, exists := sv.schemas[schemaName]
	if !exists {
		return fmt.Errorf("schema not found: %s", schemaName)
	}

	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	return sv.validateSchema(jsonData, schema)
}

// validateSchema validates data against a schema (simplified implementation)
func (sv *SchemaValidator) validateSchema(data, schema interface{}) error {
	// This is a simplified schema validation
	// In a full implementation, you would use a proper JSON Schema validator

	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("data must be a JSON object")
	}

	schemaMap, ok := schema.(map[string]interface{})
	if !ok {
		return fmt.Errorf("schema must be a JSON object")
	}

	// Check required fields
	if required, ok := schemaMap["required"].([]interface{}); ok {
		for _, req := range required {
			if reqStr, ok := req.(string); ok {
				if _, exists := dataMap[reqStr]; !exists {
					return fmt.Errorf("required field missing: %s", reqStr)
				}
			}
		}
	}

	// Check field types and constraints
	if properties, ok := schemaMap["properties"].(map[string]interface{}); ok {
		for fieldName, fieldSchema := range properties {
			if fieldData, exists := dataMap[fieldName]; exists {
				if err := sv.validateField(fieldData, fieldSchema); err != nil {
					return fmt.Errorf("field %s validation failed: %w", fieldName, err)
				}
			}
		}
	}

	return nil
}

// validateField validates a single field against its schema
func (sv *SchemaValidator) validateField(data interface{}, schema interface{}) error {
	schemaMap, ok := schema.(map[string]interface{})
	if !ok {
		return nil // No constraints
	}

	// Check type
	if expectedType, ok := schemaMap["type"].(string); ok {
		actualType := sv.getJSONType(data)
		if actualType != expectedType {
			return fmt.Errorf("expected type %s, got %s", expectedType, actualType)
		}
	}

	// Check string constraints
	if strData, ok := data.(string); ok {
		if minLength, ok := schemaMap["minLength"].(float64); ok {
			if len(strData) < int(minLength) {
				return fmt.Errorf("string too short: minimum %d characters", int(minLength))
			}
		}
		if maxLength, ok := schemaMap["maxLength"].(float64); ok {
			if len(strData) > int(maxLength) {
				return fmt.Errorf("string too long: maximum %d characters", int(maxLength))
			}
		}
		if pattern, ok := schemaMap["pattern"].(string); ok {
			matched, err := regexp.MatchString(pattern, strData)
			if err != nil {
				return fmt.Errorf("invalid pattern: %w", err)
			}
			if !matched {
				return fmt.Errorf("string does not match pattern: %s", pattern)
			}
		}
	}

	// Check numeric constraints
	if numData, ok := data.(float64); ok {
		if minimum, ok := schemaMap["minimum"].(float64); ok {
			if numData < minimum {
				return fmt.Errorf("value below minimum: %f", minimum)
			}
		}
		if maximum, ok := schemaMap["maximum"].(float64); ok {
			if numData > maximum {
				return fmt.Errorf("value above maximum: %f", maximum)
			}
		}
	}

	return nil
}

// getJSONType returns the JSON type of a value
func (sv *SchemaValidator) getJSONType(data interface{}) string {
	switch data.(type) {
	case bool:
		return "boolean"
	case float64, int, int64:
		return "number"
	case string:
		return "string"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "unknown"
	}
}

// APIRequestSchema represents a schema for API requests
type APIRequestSchema struct {
	Name      string      `json:"name"`
	Version   string      `json:"version"`
	Schema    interface{} `json:"schema"`
	Endpoints []string    `json:"endpoints"`
}

// APIResponseSchema represents a schema for API responses
type APIResponseSchema struct {
	Name       string      `json:"name"`
	StatusCode int         `json:"status_code"`
	Schema     interface{} `json:"schema"`
}

// SchemaEnforcer enforces API schemas
type SchemaEnforcer struct {
	requestSchemas  map[string]*APIRequestSchema
	responseSchemas map[string]*APIResponseSchema
	validator       *SchemaValidator
	strictMode      bool
}

// NewSchemaEnforcer creates a new schema enforcer
func NewSchemaEnforcer(strictMode bool) *SchemaEnforcer {
	return &SchemaEnforcer{
		requestSchemas:  make(map[string]*APIRequestSchema),
		responseSchemas: make(map[string]*APIResponseSchema),
		validator:       NewSchemaValidator(),
		strictMode:      strictMode,
	}
}

// RegisterRequestSchema registers a schema for API requests
func (se *SchemaEnforcer) RegisterRequestSchema(schema *APIRequestSchema) {
	se.requestSchemas[schema.Name] = schema
	for _, endpoint := range schema.Endpoints {
		se.validator.RegisterSchema(endpoint+"_request", schema.Schema)
	}
}

// RegisterResponseSchema registers a schema for API responses
func (se *SchemaEnforcer) RegisterResponseSchema(schema *APIResponseSchema) {
	key := fmt.Sprintf("%s_%d", schema.Name, schema.StatusCode)
	se.responseSchemas[key] = schema
	se.validator.RegisterSchema(key+"_response", schema.Schema)
}

// ValidateRequest validates an API request
func (se *SchemaEnforcer) ValidateRequest(endpoint string, data []byte) error {
	schemaName := endpoint + "_request"
	return se.validator.ValidateJSON(data, schemaName)
}

// ValidateResponse validates an API response
func (se *SchemaEnforcer) ValidateResponse(endpoint string, statusCode int, data []byte) error {
	schemaName := fmt.Sprintf("%s_%d_response", endpoint, statusCode)
	return se.validator.ValidateJSON(data, schemaName)
}

// ValidateStruct validates a struct
func (se *SchemaEnforcer) ValidateStruct(s interface{}) error {
	return se.validator.ValidateStruct(s)
}

// SetupDefaultSchemas sets up default schemas for the LLM Verifier API
func (se *SchemaEnforcer) SetupDefaultSchemas() {
	// LLM Request Schema
	llmRequestSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"prompt": map[string]interface{}{
				"type":      "string",
				"minLength": 1,
				"maxLength": 10000,
			},
			"messages": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"role": map[string]interface{}{
							"type": "string",
							"enum": []string{"user", "assistant", "system"},
						},
						"content": map[string]interface{}{
							"type":      "string",
							"minLength": 1,
							"maxLength": 5000,
						},
					},
					"required": []string{"role", "content"},
				},
			},
			"max_tokens": map[string]interface{}{
				"type":    "integer",
				"minimum": 1,
				"maximum": 4096,
			},
			"temperature": map[string]interface{}{
				"type":    "number",
				"minimum": 0.0,
				"maximum": 2.0,
			},
			"stream": map[string]interface{}{
				"type": "boolean",
			},
		},
		"oneOf": []map[string]interface{}{
			{"required": []string{"prompt"}},
			{"required": []string{"messages"}},
		},
	}

	// Verification Result Schema
	verificationResultSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type": "integer",
			},
			"model_id": map[string]interface{}{
				"type": "integer",
			},
			"verification_type": map[string]interface{}{
				"type": "string",
				"enum": []string{"full", "quick", "scheduled", "manual"},
			},
			"status": map[string]interface{}{
				"type": "string",
				"enum": []string{"running", "completed", "failed", "cancelled"},
			},
			"overall_score": map[string]interface{}{
				"type":    "number",
				"minimum": 0.0,
				"maximum": 100.0,
			},
			"code_capability_score": map[string]interface{}{
				"type":    "number",
				"minimum": 0.0,
				"maximum": 100.0,
			},
			"responsiveness_score": map[string]interface{}{
				"type":    "number",
				"minimum": 0.0,
				"maximum": 100.0,
			},
			"reliability_score": map[string]interface{}{
				"type":    "number",
				"minimum": 0.0,
				"maximum": 100.0,
			},
		},
		"required": []string{"id", "model_id", "verification_type", "status"},
	}

	// Register schemas
	se.RegisterRequestSchema(&APIRequestSchema{
		Name:    "llm_request",
		Version: "1.0",
		Schema:  llmRequestSchema,
		Endpoints: []string{
			"/api/v1/verify",
			"/api/v1/chat",
		},
	})

	se.RegisterResponseSchema(&APIResponseSchema{
		Name:       "verification_result",
		StatusCode: 200,
		Schema:     verificationResultSchema,
	})
}

// Custom validation functions

// validateLLMModel validates LLM model names
func validateLLMModel(fl validator.FieldLevel) bool {
	model := fl.Field().String()
	// Basic validation - should contain provider/model pattern
	return strings.Contains(model, "/") && len(model) > 3
}

// validateProviderName validates provider names
func validateProviderName(fl validator.FieldLevel) bool {
	provider := strings.ToLower(fl.Field().String())
	validProviders := []string{
		"openai", "anthropic", "google", "cohere", "meta", "mistral",
		"azure", "aws", "huggingface", "replicate",
		"groq", "togetherai", "fireworks", "poe", "navigator",
	}

	for _, valid := range validProviders {
		if provider == valid {
			return true
		}
	}
	return false
}

// validateSafePrompt validates that prompts don't contain harmful content
func validateSafePrompt(fl validator.FieldLevel) bool {
	prompt := strings.ToLower(fl.Field().String())

	// Check for harmful patterns
	harmfulPatterns := []string{
		"drop table", "delete from", "truncate table",
		"eval(", "exec(", "system(",
		"<script", "javascript:",
		"rm -rf", "format c:",
	}

	for _, pattern := range harmfulPatterns {
		if strings.Contains(prompt, pattern) {
			return false
		}
	}

	return true
}

// CrossProviderValidator validates consistency across providers
type CrossProviderValidator struct {
	providerResults map[string]interface{}
	mu              sync.RWMutex
}

// NewCrossProviderValidator creates a new cross-provider validator
func NewCrossProviderValidator() *CrossProviderValidator {
	return &CrossProviderValidator{
		providerResults: make(map[string]interface{}),
	}
}

// AddProviderResult adds a result from a provider
func (cpv *CrossProviderValidator) AddProviderResult(provider string, result interface{}) {
	cpv.mu.Lock()
	defer cpv.mu.Unlock()
	cpv.providerResults[provider] = result
}

// ValidateConsistency validates consistency across providers
func (cpv *CrossProviderValidator) ValidateConsistency() *ValidationResult {
	cpv.mu.RLock()
	defer cpv.mu.RUnlock()

	result := &ValidationResult{
		Level:    LevelIntegration,
		Passed:   true,
		Errors:   []string{},
		Warnings: []string{},
		Score:    1.0,
		Metadata: make(map[string]interface{}),
	}

	if len(cpv.providerResults) < 2 {
		result.Warnings = append(result.Warnings, "Need at least 2 providers for cross-validation")
		result.Score = 0.8
		return result
	}

	// Compare scores across providers (simplified)
	scores := make(map[string]float64)
	for provider, res := range cpv.providerResults {
		// Extract score from result (this would depend on the actual result structure)
		if score := cpv.extractScore(res); score >= 0 {
			scores[provider] = score
		}
	}

	if len(scores) < 2 {
		result.Warnings = append(result.Warnings, "Could not extract scores from provider results")
		result.Score = 0.7
		return result
	}

	// Check for significant score differences
	avgScore := cpv.calculateAverage(scores)
	maxDeviation := cpv.calculateMaxDeviation(scores, avgScore)

	if maxDeviation > 20.0 { // More than 20 points difference
		result.Warnings = append(result.Warnings, fmt.Sprintf("Large score variation across providers: %.1f points", maxDeviation))
		result.Score -= 0.2
	}

	result.Metadata["provider_count"] = len(cpv.providerResults)
	result.Metadata["average_score"] = avgScore
	result.Metadata["max_deviation"] = maxDeviation

	return result
}

// extractScore extracts a score from a provider result (simplified)
func (cpv *CrossProviderValidator) extractScore(result interface{}) float64 {
	// This is a placeholder - actual implementation would depend on result structure
	if res, ok := result.(map[string]interface{}); ok {
		if score, ok := res["overall_score"].(float64); ok {
			return score
		}
	}
	return -1
}

// calculateAverage calculates the average of scores
func (cpv *CrossProviderValidator) calculateAverage(scores map[string]float64) float64 {
	sum := 0.0
	for _, score := range scores {
		sum += score
	}
	return sum / float64(len(scores))
}

// calculateMaxDeviation calculates the maximum deviation from average
func (cpv *CrossProviderValidator) calculateMaxDeviation(scores map[string]float64, avg float64) float64 {
	maxDev := 0.0
	for _, score := range scores {
		dev := math.Abs(score - avg)
		if dev > maxDev {
			maxDev = dev
		}
	}
	return maxDev
}

// ContextAwareValidator provides context-aware validation rules
type ContextAwareValidator struct {
	contextRules map[string][]ValidationRule
	history      []ValidationResult
	maxHistory   int
}

// ValidationRule represents a context-aware validation rule
type ValidationRule struct {
	Name        string
	Description string
	Condition   func(context map[string]interface{}) bool
	Action      func(result *ValidationResult, context map[string]interface{})
	Priority    int // Higher priority rules are checked first
}

// NewContextAwareValidator creates a new context-aware validator
func NewContextAwareValidator(maxHistory int) *ContextAwareValidator {
	return &ContextAwareValidator{
		contextRules: make(map[string][]ValidationRule),
		history:      make([]ValidationResult, 0),
		maxHistory:   maxHistory,
	}
}

// AddRule adds a validation rule for a specific context type
func (cav *ContextAwareValidator) AddRule(contextType string, rule ValidationRule) {
	cav.contextRules[contextType] = append(cav.contextRules[contextType], rule)

	// Sort rules by priority (highest first)
	rules := cav.contextRules[contextType]
	for i := 0; i < len(rules)-1; i++ {
		for j := i + 1; j < len(rules); j++ {
			if rules[i].Priority < rules[j].Priority {
				rules[i], rules[j] = rules[j], rules[i]
			}
		}
	}
}

// ValidateWithContext validates input with context awareness
func (cav *ContextAwareValidator) ValidateWithContext(input interface{}, contextType string, context map[string]interface{}) *ValidationResult {
	// Start with basic validation
	baseResult := &ValidationResult{
		Level:    LevelSemantic,
		Passed:   true,
		Errors:   []string{},
		Warnings: []string{},
		Score:    1.0,
		Metadata: make(map[string]interface{}),
	}

	// Apply context-aware rules
	if rules, exists := cav.contextRules[contextType]; exists {
		for _, rule := range rules {
			if rule.Condition(context) {
				rule.Action(baseResult, context)
			}
		}
	}

	// Add historical context
	cav.addHistoricalContext(baseResult, contextType, context)

	// Store result in history
	cav.addToHistory(*baseResult)

	return baseResult
}

// addHistoricalContext adds insights from historical validation results
func (cav *ContextAwareValidator) addHistoricalContext(result *ValidationResult, contextType string, context map[string]interface{}) {
	if len(cav.history) == 0 {
		return
	}

	// Analyze recent history for patterns
	recentResults := cav.getRecentResults(10) // Last 10 results

	// Check for error patterns
	errorPatterns := cav.analyzeErrorPatterns(recentResults, contextType)
	for _, pattern := range errorPatterns {
		result.Warnings = append(result.Warnings, pattern)
		result.Score -= 0.1
	}

	// Check for performance trends
	performanceTrend := cav.analyzePerformanceTrend(recentResults)
	if performanceTrend < 0 {
		result.Warnings = append(result.Warnings, "Validation scores trending downward")
		result.Score -= 0.05
	}

	result.Metadata["historical_analysis"] = true
	result.Metadata["recent_results_count"] = len(recentResults)
	result.Metadata["performance_trend"] = performanceTrend
}

// analyzeErrorPatterns analyzes patterns in recent validation errors
func (cav *ContextAwareValidator) analyzeErrorPatterns(results []ValidationResult, contextType string) []string {
	errorCounts := make(map[string]int)

	for _, result := range results {
		for _, err := range result.Errors {
			errorCounts[err]++
		}
	}

	var patterns []string
	for errorMsg, count := range errorCounts {
		if count >= 3 { // Error occurred 3+ times recently
			patterns = append(patterns, fmt.Sprintf("Recurring error pattern: %s (%d occurrences)", errorMsg, count))
		}
	}

	return patterns
}

// analyzePerformanceTrend analyzes the trend in validation scores
func (cav *ContextAwareValidator) analyzePerformanceTrend(results []ValidationResult) float64 {
	if len(results) < 2 {
		return 0
	}

	// Calculate trend using simple linear regression
	n := float64(len(results))
	sumX := n * (n - 1) / 2
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0

	for i, result := range results {
		x := float64(i)
		y := result.Score
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	return slope
}

// getRecentResults returns the most recent validation results
func (cav *ContextAwareValidator) getRecentResults(count int) []ValidationResult {
	if len(cav.history) <= count {
		return cav.history
	}
	return cav.history[len(cav.history)-count:]
}

// addToHistory adds a result to the validation history
func (cav *ContextAwareValidator) addToHistory(result ValidationResult) {
	cav.history = append(cav.history, result)

	// Maintain maximum history size
	if len(cav.history) > cav.maxHistory {
		cav.history = cav.history[len(cav.history)-cav.maxHistory:]
	}
}

// SetupDefaultRules sets up default context-aware validation rules
func (cav *ContextAwareValidator) SetupDefaultRules() {
	// LLM Request Context Rules
	cav.AddRule("llm_request", ValidationRule{
		Name:        "high_frequency_user",
		Description: "Check for unusually high request frequency from user",
		Condition: func(context map[string]interface{}) bool {
			if frequency, ok := context["requests_per_minute"].(float64); ok {
				return frequency > 10.0 // More than 10 requests per minute
			}
			return false
		},
		Action: func(result *ValidationResult, context map[string]interface{}) {
			result.Warnings = append(result.Warnings, "High request frequency detected - may indicate abuse")
			result.Score -= 0.2
			result.Metadata["high_frequency_detected"] = true
		},
		Priority: 10,
	})

	cav.AddRule("llm_request", ValidationRule{
		Name:        "large_context_window",
		Description: "Check for very large context windows that may impact performance",
		Condition: func(context map[string]interface{}) bool {
			if messages, ok := context["message_count"].(int); ok {
				return messages > 20 // More than 20 messages
			}
			return false
		},
		Action: func(result *ValidationResult, context map[string]interface{}) {
			result.Warnings = append(result.Warnings, "Large context window may impact response quality and latency")
			result.Score -= 0.1
			result.Metadata["large_context"] = true
		},
		Priority: 8,
	})

	cav.AddRule("llm_request", ValidationRule{
		Name:        "suspicious_prompt_patterns",
		Description: "Check for suspicious prompt patterns that may indicate jailbreak attempts",
		Condition: func(context map[string]interface{}) bool {
			if prompt, ok := context["prompt"].(string); ok {
				suspiciousPatterns := []string{
					"ignore previous instructions",
					"override safety",
					"jailbreak",
					"uncensored mode",
				}
				promptLower := strings.ToLower(prompt)
				for _, pattern := range suspiciousPatterns {
					if strings.Contains(promptLower, pattern) {
						return true
					}
				}
			}
			return false
		},
		Action: func(result *ValidationResult, context map[string]interface{}) {
			result.Errors = append(result.Errors, "Prompt contains suspicious patterns that may violate usage policies")
			result.Passed = false
			result.Score = 0.0
			result.Metadata["suspicious_content"] = true
		},
		Priority: 15, // High priority - security related
	})

	// Verification Context Rules
	cav.AddRule("verification", ValidationRule{
		Name:        "model_performance_drift",
		Description: "Check for significant changes in model performance over time",
		Condition: func(context map[string]interface{}) bool {
			if baselineScore, ok := context["baseline_score"].(float64); ok {
				if currentScore, ok := context["current_score"].(float64); ok {
					deviation := math.Abs(currentScore - baselineScore)
					return deviation > 15.0 // More than 15 points deviation
				}
			}
			return false
		},
		Action: func(result *ValidationResult, context map[string]interface{}) {
			result.Warnings = append(result.Warnings, "Significant model performance drift detected")
			result.Score -= 0.3
			result.Metadata["performance_drift"] = true
		},
		Priority: 12,
	})

	cav.AddRule("verification", ValidationRule{
		Name:        "provider_outage_pattern",
		Description: "Check for patterns indicating provider outages",
		Condition: func(context map[string]interface{}) bool {
			if recentErrors, ok := context["recent_errors"].(int); ok {
				if totalRequests, ok := context["total_requests"].(int); ok {
					errorRate := float64(recentErrors) / float64(totalRequests)
					return errorRate > 0.5 // More than 50% error rate
				}
			}
			return false
		},
		Action: func(result *ValidationResult, context map[string]interface{}) {
			result.Warnings = append(result.Warnings, "High error rate may indicate provider outage or configuration issues")
			result.Score -= 0.4
			result.Metadata["potential_outage"] = true
		},
		Priority: 14,
	})
}

// GetValidationHistory returns the validation history
func (cav *ContextAwareValidator) GetValidationHistory() []ValidationResult {
	history := make([]ValidationResult, len(cav.history))
	copy(history, cav.history)
	return history
}

// ClearHistory clears the validation history
func (cav *ContextAwareValidator) ClearHistory() {
	cav.history = make([]ValidationResult, 0)
}
