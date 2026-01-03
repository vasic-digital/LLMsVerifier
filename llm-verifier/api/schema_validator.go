package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

// ValidationError represents a schema validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// ValidationResult contains the result of schema validation
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// SchemaValidator provides JSON schema validation
type SchemaValidator struct {
	schemas      map[string]map[string]interface{}
	compiledPatterns map[string]*regexp.Regexp
	mu           sync.RWMutex
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{
		schemas:          make(map[string]map[string]interface{}),
		compiledPatterns: make(map[string]*regexp.Regexp),
	}
}

// RegisterSchema registers a JSON schema for a given name
func (sv *SchemaValidator) RegisterSchema(name string, schema interface{}) error {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	schemaMap, ok := schema.(map[string]interface{})
	if !ok {
		// Convert to map
		schemaBytes, err := json.Marshal(schema)
		if err != nil {
			return fmt.Errorf("failed to marshal schema: %w", err)
		}
		if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
			return fmt.Errorf("failed to unmarshal schema: %w", err)
		}
	}

	sv.schemas[name] = schemaMap

	// Pre-compile any regex patterns in the schema
	sv.compilePatterns(schemaMap)

	return nil
}

// compilePatterns pre-compiles regex patterns in the schema
func (sv *SchemaValidator) compilePatterns(schema map[string]interface{}) {
	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		for fieldName, fieldSchema := range properties {
			if fs, ok := fieldSchema.(map[string]interface{}); ok {
				if pattern, ok := fs["pattern"].(string); ok {
					key := fmt.Sprintf("%p:%s", schema, fieldName)
					if compiled, err := regexp.Compile(pattern); err == nil {
						sv.compiledPatterns[key] = compiled
					}
				}
			}
		}
	}
}

// Validate validates data against a registered schema
func (sv *SchemaValidator) Validate(schemaName string, data interface{}) error {
	sv.mu.RLock()
	schema, exists := sv.schemas[schemaName]
	sv.mu.RUnlock()

	if !exists {
		return fmt.Errorf("schema '%s' not found", schemaName)
	}

	result := sv.ValidateWithResult(schema, data, "")
	if !result.Valid {
		// Return first error
		if len(result.Errors) > 0 {
			return fmt.Errorf("%s: %s", result.Errors[0].Field, result.Errors[0].Message)
		}
		return fmt.Errorf("validation failed")
	}
	return nil
}

// ValidateWithResult validates data and returns detailed results
func (sv *SchemaValidator) ValidateWithResult(schema map[string]interface{}, data interface{}, path string) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Convert data to map if it's a struct
	dataMap, ok := sv.toMap(data)
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   path,
			Message: "expected object",
			Value:   data,
		})
		return result
	}

	// Validate required fields
	sv.validateRequired(schema, dataMap, path, result)

	// Validate properties
	sv.validateProperties(schema, dataMap, path, result)

	// Validate additional properties
	sv.validateAdditionalProperties(schema, dataMap, path, result)

	return result
}

// toMap converts data to map[string]interface{}
func (sv *SchemaValidator) toMap(data interface{}) (map[string]interface{}, bool) {
	if m, ok := data.(map[string]interface{}); ok {
		return m, true
	}

	// Try JSON conversion for structs
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, false
	}

	var m map[string]interface{}
	if err := json.Unmarshal(dataBytes, &m); err != nil {
		return nil, false
	}

	return m, true
}

// validateRequired checks required fields
func (sv *SchemaValidator) validateRequired(schema map[string]interface{}, data map[string]interface{}, path string, result *ValidationResult) {
	required, ok := schema["required"]
	if !ok {
		return
	}

	requiredList, ok := required.([]interface{})
	if !ok {
		// Try string slice
		if strList, ok := required.([]string); ok {
			for _, field := range strList {
				if _, exists := data[field]; !exists {
					result.Valid = false
					result.Errors = append(result.Errors, ValidationError{
						Field:   sv.joinPath(path, field),
						Message: "required field is missing",
					})
				}
			}
		}
		return
	}

	for _, field := range requiredList {
		fieldName, ok := field.(string)
		if !ok {
			continue
		}
		if _, exists := data[fieldName]; !exists {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   sv.joinPath(path, fieldName),
				Message: "required field is missing",
			})
		}
	}
}

// validateProperties validates each property against its schema
func (sv *SchemaValidator) validateProperties(schema map[string]interface{}, data map[string]interface{}, path string, result *ValidationResult) {
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return
	}

	for fieldName, fieldSchema := range properties {
		fieldSchemaMap, ok := fieldSchema.(map[string]interface{})
		if !ok {
			continue
		}

		value, exists := data[fieldName]
		if !exists {
			continue // Optional field not present
		}

		fieldPath := sv.joinPath(path, fieldName)

		// Validate type
		if expectedType, ok := fieldSchemaMap["type"].(string); ok {
			if err := sv.validateType(value, expectedType, fieldPath); err != nil {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Field:   fieldPath,
					Message: err.Error(),
					Value:   value,
				})
				continue
			}
		}

		// Validate string constraints
		if str, ok := value.(string); ok {
			sv.validateStringConstraints(str, fieldSchemaMap, fieldPath, result)
		}

		// Validate number constraints
		if num, ok := sv.toNumber(value); ok {
			sv.validateNumberConstraints(num, fieldSchemaMap, fieldPath, result)
		}

		// Validate enum
		if enum, ok := fieldSchemaMap["enum"].([]interface{}); ok {
			sv.validateEnum(value, enum, fieldPath, result)
		}

		// Validate pattern
		if pattern, ok := fieldSchemaMap["pattern"].(string); ok {
			if str, ok := value.(string); ok {
				sv.validatePattern(str, pattern, fieldPath, result)
			}
		}

		// Validate nested object
		if expectedType, ok := fieldSchemaMap["type"].(string); ok && expectedType == "object" {
			if nestedData, ok := value.(map[string]interface{}); ok {
				nestedResult := sv.ValidateWithResult(fieldSchemaMap, nestedData, fieldPath)
				if !nestedResult.Valid {
					result.Valid = false
					result.Errors = append(result.Errors, nestedResult.Errors...)
				}
			}
		}

		// Validate array items
		if expectedType, ok := fieldSchemaMap["type"].(string); ok && expectedType == "array" {
			if arr, ok := value.([]interface{}); ok {
				sv.validateArray(arr, fieldSchemaMap, fieldPath, result)
			}
		}
	}
}

// validateStringConstraints validates string-specific constraints
func (sv *SchemaValidator) validateStringConstraints(str string, schema map[string]interface{}, path string, result *ValidationResult) {
	if minLength, ok := sv.toInt(schema["minLength"]); ok {
		if len(str) < minLength {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   path,
				Message: fmt.Sprintf("string length %d is less than minimum %d", len(str), minLength),
				Value:   str,
			})
		}
	}

	if maxLength, ok := sv.toInt(schema["maxLength"]); ok {
		if len(str) > maxLength {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   path,
				Message: fmt.Sprintf("string length %d exceeds maximum %d", len(str), maxLength),
				Value:   str,
			})
		}
	}

	// Validate format
	if format, ok := schema["format"].(string); ok {
		if err := sv.validateFormat(str, format); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   path,
				Message: err.Error(),
				Value:   str,
			})
		}
	}
}

// validateNumberConstraints validates number-specific constraints
func (sv *SchemaValidator) validateNumberConstraints(num float64, schema map[string]interface{}, path string, result *ValidationResult) {
	if minimum, ok := sv.toNumber(schema["minimum"]); ok {
		exclusive := false
		if exc, ok := schema["exclusiveMinimum"].(bool); ok {
			exclusive = exc
		}
		if exclusive && num <= minimum {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   path,
				Message: fmt.Sprintf("value %v must be greater than %v", num, minimum),
				Value:   num,
			})
		} else if !exclusive && num < minimum {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   path,
				Message: fmt.Sprintf("value %v is less than minimum %v", num, minimum),
				Value:   num,
			})
		}
	}

	if maximum, ok := sv.toNumber(schema["maximum"]); ok {
		exclusive := false
		if exc, ok := schema["exclusiveMaximum"].(bool); ok {
			exclusive = exc
		}
		if exclusive && num >= maximum {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   path,
				Message: fmt.Sprintf("value %v must be less than %v", num, maximum),
				Value:   num,
			})
		} else if !exclusive && num > maximum {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   path,
				Message: fmt.Sprintf("value %v exceeds maximum %v", num, maximum),
				Value:   num,
			})
		}
	}

	if multipleOf, ok := sv.toNumber(schema["multipleOf"]); ok {
		if multipleOf != 0 {
			remainder := num / multipleOf
			if remainder != float64(int64(remainder)) {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Field:   path,
					Message: fmt.Sprintf("value %v is not a multiple of %v", num, multipleOf),
					Value:   num,
				})
			}
		}
	}
}

// validateEnum validates that a value is one of the allowed values
func (sv *SchemaValidator) validateEnum(value interface{}, enum []interface{}, path string, result *ValidationResult) {
	for _, allowed := range enum {
		if fmt.Sprintf("%v", value) == fmt.Sprintf("%v", allowed) {
			return
		}
	}
	result.Valid = false
	result.Errors = append(result.Errors, ValidationError{
		Field:   path,
		Message: fmt.Sprintf("value must be one of: %v", enum),
		Value:   value,
	})
}

// validatePattern validates a string against a regex pattern
func (sv *SchemaValidator) validatePattern(str, pattern, path string, result *ValidationResult) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   path,
			Message: fmt.Sprintf("invalid pattern: %v", err),
		})
		return
	}

	if !re.MatchString(str) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   path,
			Message: fmt.Sprintf("value does not match pattern: %s", pattern),
			Value:   str,
		})
	}
}

// validateFormat validates common string formats
func (sv *SchemaValidator) validateFormat(str, format string) error {
	switch format {
	case "email":
		// Basic email validation
		if !strings.Contains(str, "@") || !strings.Contains(str, ".") {
			return fmt.Errorf("invalid email format")
		}
	case "uri", "url":
		if !strings.HasPrefix(str, "http://") && !strings.HasPrefix(str, "https://") {
			return fmt.Errorf("invalid URI format")
		}
	case "date":
		// Basic date format YYYY-MM-DD
		matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, str)
		if !matched {
			return fmt.Errorf("invalid date format (expected YYYY-MM-DD)")
		}
	case "date-time":
		// Basic ISO 8601 date-time
		matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`, str)
		if !matched {
			return fmt.Errorf("invalid date-time format")
		}
	case "uuid":
		matched, _ := regexp.MatchString(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`, str)
		if !matched {
			return fmt.Errorf("invalid UUID format")
		}
	case "ipv4":
		matched, _ := regexp.MatchString(`^(\d{1,3}\.){3}\d{1,3}$`, str)
		if !matched {
			return fmt.Errorf("invalid IPv4 format")
		}
	case "ipv6":
		matched, _ := regexp.MatchString(`^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$`, str)
		if !matched {
			return fmt.Errorf("invalid IPv6 format")
		}
	}
	return nil
}

// validateArray validates array elements
func (sv *SchemaValidator) validateArray(arr []interface{}, schema map[string]interface{}, path string, result *ValidationResult) {
	// Validate minItems
	if minItems, ok := sv.toInt(schema["minItems"]); ok {
		if len(arr) < minItems {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   path,
				Message: fmt.Sprintf("array has %d items, minimum is %d", len(arr), minItems),
			})
		}
	}

	// Validate maxItems
	if maxItems, ok := sv.toInt(schema["maxItems"]); ok {
		if len(arr) > maxItems {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   path,
				Message: fmt.Sprintf("array has %d items, maximum is %d", len(arr), maxItems),
			})
		}
	}

	// Validate uniqueItems
	if uniqueItems, ok := schema["uniqueItems"].(bool); ok && uniqueItems {
		seen := make(map[string]bool)
		for i, item := range arr {
			key := fmt.Sprintf("%v", item)
			if seen[key] {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Field:   fmt.Sprintf("%s[%d]", path, i),
					Message: "duplicate item in array",
					Value:   item,
				})
			}
			seen[key] = true
		}
	}

	// Validate items schema
	if itemsSchema, ok := schema["items"].(map[string]interface{}); ok {
		for i, item := range arr {
			itemPath := fmt.Sprintf("%s[%d]", path, i)

			if expectedType, ok := itemsSchema["type"].(string); ok {
				if err := sv.validateType(item, expectedType, itemPath); err != nil {
					result.Valid = false
					result.Errors = append(result.Errors, ValidationError{
						Field:   itemPath,
						Message: err.Error(),
						Value:   item,
					})
				}
			}

			// Validate nested object items
			if itemMap, ok := item.(map[string]interface{}); ok {
				if expectedType, _ := itemsSchema["type"].(string); expectedType == "object" {
					nestedResult := sv.ValidateWithResult(itemsSchema, itemMap, itemPath)
					if !nestedResult.Valid {
						result.Valid = false
						result.Errors = append(result.Errors, nestedResult.Errors...)
					}
				}
			}
		}
	}
}

// validateAdditionalProperties validates extra properties not in schema
func (sv *SchemaValidator) validateAdditionalProperties(schema map[string]interface{}, data map[string]interface{}, path string, result *ValidationResult) {
	additionalProperties, hasAdditional := schema["additionalProperties"]
	if !hasAdditional {
		return // Default is to allow additional properties
	}

	// If additionalProperties is false, reject unknown properties
	if allow, ok := additionalProperties.(bool); ok && !allow {
		properties, _ := schema["properties"].(map[string]interface{})
		for key := range data {
			if properties != nil {
				if _, exists := properties[key]; exists {
					continue
				}
			}
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   sv.joinPath(path, key),
				Message: "additional property not allowed",
			})
		}
	}
}

// validateType validates the type of a value
func (sv *SchemaValidator) validateType(value interface{}, expectedType, path string) error {
	if value == nil {
		return fmt.Errorf("value is null")
	}

	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "number":
		switch value.(type) {
		case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			// Valid
		default:
			return fmt.Errorf("expected number, got %T", value)
		}
	case "integer":
		switch v := value.(type) {
		case float64:
			if v != float64(int64(v)) {
				return fmt.Errorf("expected integer, got float")
			}
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			// Valid
		default:
			return fmt.Errorf("expected integer, got %T", value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("expected object, got %T", value)
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return fmt.Errorf("expected array, got %T", value)
		}
	case "null":
		if value != nil {
			return fmt.Errorf("expected null, got %T", value)
		}
	}

	return nil
}

// toNumber converts a value to float64
func (sv *SchemaValidator) toNumber(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	default:
		return 0, false
	}
}

// toInt converts a value to int
func (sv *SchemaValidator) toInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// joinPath joins path segments
func (sv *SchemaValidator) joinPath(base, field string) string {
	if base == "" {
		return field
	}
	return base + "." + field
}

// ValidateRequest validates an HTTP request body against a schema
func (sv *SchemaValidator) ValidateRequest(schemaName string, r *http.Request, target interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to decode request body: %w", err)
	}

	return sv.Validate(schemaName, target)
}

// ValidateResponse validates a response against a schema
func (sv *SchemaValidator) ValidateResponse(schemaName string, response interface{}) error {
	return sv.Validate(schemaName, response)
}

// ValidateJSON validates a JSON string against a schema
func (sv *SchemaValidator) ValidateJSON(schemaName string, jsonStr string) error {
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return sv.Validate(schemaName, data)
}

// GetSchemaNames returns all registered schema names
func (sv *SchemaValidator) GetSchemaNames() []string {
	sv.mu.RLock()
	defer sv.mu.RUnlock()

	names := make([]string, 0, len(sv.schemas))
	for name := range sv.schemas {
		names = append(names, name)
	}
	return names
}

// Predefined schemas
var (
	// ModelRequestSchema for model creation/update requests
	ModelRequestSchema = map[string]interface{}{
		"type":     "object",
		"required": []interface{}{"name", "provider_id"},
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":      "string",
				"minLength": 2,
				"maxLength": 100,
			},
			"provider_id": map[string]interface{}{
				"type":    "integer",
				"minimum": 1,
			},
			"description": map[string]interface{}{
				"type":      "string",
				"maxLength": 1000,
			},
			"model_id": map[string]interface{}{
				"type":      "string",
				"minLength": 1,
				"maxLength": 100,
			},
		},
	}

	// VerificationRequestSchema for verification requests
	VerificationRequestSchema = map[string]interface{}{
		"type":     "object",
		"required": []interface{}{"model_id"},
		"properties": map[string]interface{}{
			"model_id": map[string]interface{}{
				"type":    "integer",
				"minimum": 1,
			},
			"verification_type": map[string]interface{}{
				"type": "string",
				"enum": []interface{}{"comprehensive", "quick", "targeted"},
			},
			"timeout": map[string]interface{}{
				"type":    "integer",
				"minimum": 1,
				"maximum": 3600,
			},
		},
	}

	// ExportRequestSchema for export requests
	ExportRequestSchema = map[string]interface{}{
		"type":     "object",
		"required": []interface{}{"export_type"},
		"properties": map[string]interface{}{
			"export_type": map[string]interface{}{
				"type": "string",
				"enum": []interface{}{"opencode", "crush", "claude-code"},
			},
			"model_ids": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type":    "integer",
					"minimum": 1,
				},
			},
			"include_api_keys": map[string]interface{}{
				"type": "boolean",
			},
		},
	}
)
