package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// SchemaValidator provides JSON schema validation
type SchemaValidator struct {
	schemas map[string]interface{}
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{
		schemas: make(map[string]interface{}),
	}
}

// RegisterSchema registers a JSON schema for a given name
func (sv *SchemaValidator) RegisterSchema(name string, schema interface{}) {
	sv.schemas[name] = schema
}

// Validate validates data against a registered schema
func (sv *SchemaValidator) Validate(schemaName string, data interface{}) error {
	schema, exists := sv.schemas[schemaName]
	if !exists {
		return fmt.Errorf("schema %s not found", schemaName)
	}

	// For now, implement basic validation
	// In production, you'd use a proper JSON schema validator like gojsonschema
	return sv.basicValidation(schema, data)
}

// basicValidation performs basic validation (placeholder for full JSON schema validation)
func (sv *SchemaValidator) basicValidation(schema, data interface{}) error {
	// Convert to JSON for basic validation
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	var schemaMap map[string]interface{}
	var dataMap map[string]interface{}

	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		return fmt.Errorf("invalid schema format: %w", err)
	}

	if err := json.Unmarshal(dataBytes, &dataMap); err != nil {
		return fmt.Errorf("invalid data format: %w", err)
	}

	// Basic required field validation
	if required, ok := schemaMap["required"].([]interface{}); ok {
		for _, field := range required {
			if fieldName, ok := field.(string); ok {
				if _, exists := dataMap[fieldName]; !exists {
					return fmt.Errorf("required field '%s' is missing", fieldName)
				}
			}
		}
	}

	// Basic type validation
	if properties, ok := schemaMap["properties"].(map[string]interface{}); ok {
		for fieldName, fieldSchema := range properties {
			if fieldSchemaMap, ok := fieldSchema.(map[string]interface{}); ok {
				if expectedType, ok := fieldSchemaMap["type"].(string); ok {
					if actualValue, exists := dataMap[fieldName]; exists {
						if err := sv.validateType(actualValue, expectedType); err != nil {
							return fmt.Errorf("field '%s': %w", fieldName, err)
						}
					}
				}
			}
		}
	}

	return nil
}

// validateType validates the type of a value
func (sv *SchemaValidator) validateType(value interface{}, expectedType string) error {
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "number":
		switch value.(type) {
		case float64, int, int64:
			// Valid
		default:
			return fmt.Errorf("expected number, got %T", value)
		}
	case "integer":
		switch value.(type) {
		case int, int64:
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
	default:
		// Unknown type, accept it
	}

	return nil
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
