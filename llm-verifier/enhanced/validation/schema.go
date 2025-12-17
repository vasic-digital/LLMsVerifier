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
