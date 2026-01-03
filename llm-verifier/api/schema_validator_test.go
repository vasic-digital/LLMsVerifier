package api

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSchemaValidator(t *testing.T) {
	validator := NewSchemaValidator()

	assert.NotNil(t, validator)
	assert.NotNil(t, validator.schemas)
	assert.Empty(t, validator.schemas)
}

func TestSchemaValidator_RegisterSchema(t *testing.T) {
	validator := NewSchemaValidator()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
		},
	}

	err := validator.RegisterSchema("test_schema", schema)

	require.NoError(t, err)
	assert.Contains(t, validator.schemas, "test_schema")
	assert.Equal(t, schema, validator.schemas["test_schema"])
}

func TestSchemaValidator_Validate_SchemaNotFound(t *testing.T) {
	validator := NewSchemaValidator()

	err := validator.Validate("nonexistent", map[string]interface{}{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schema 'nonexistent' not found")
}

func TestSchemaValidator_Validate_RequiredFields(t *testing.T) {
	validator := NewSchemaValidator()

	schema := map[string]interface{}{
		"type":     "object",
		"required": []interface{}{"name", "email"},
		"properties": map[string]interface{}{
			"name":  map[string]interface{}{"type": "string"},
			"email": map[string]interface{}{"type": "string"},
		},
	}

	err := validator.RegisterSchema("user", schema)
	require.NoError(t, err)

	t.Run("all required fields present", func(t *testing.T) {
		data := map[string]interface{}{
			"name":  "John",
			"email": "john@example.com",
		}
		err := validator.Validate("user", data)
		assert.NoError(t, err)
	})

	t.Run("missing required field", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "John",
		}
		err := validator.Validate("user", data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required field is missing")
	})
}

func TestSchemaValidator_Validate_TypeValidation(t *testing.T) {
	validator := NewSchemaValidator()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name":    map[string]interface{}{"type": "string"},
			"age":     map[string]interface{}{"type": "number"},
			"active":  map[string]interface{}{"type": "boolean"},
			"tags":    map[string]interface{}{"type": "array"},
			"address": map[string]interface{}{"type": "object"},
		},
	}

	err := validator.RegisterSchema("full_types", schema)
	require.NoError(t, err)

	t.Run("valid types", func(t *testing.T) {
		data := map[string]interface{}{
			"name":    "John",
			"age":     float64(30),
			"active":  true,
			"tags":    []interface{}{"admin", "user"},
			"address": map[string]interface{}{"city": "NYC"},
		}
		err := validator.Validate("full_types", data)
		assert.NoError(t, err)
	})

	t.Run("invalid string type", func(t *testing.T) {
		data := map[string]interface{}{
			"name": 123, // should be string
		}
		err := validator.Validate("full_types", data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected string")
	})

	t.Run("invalid number type", func(t *testing.T) {
		data := map[string]interface{}{
			"age": "thirty", // should be number
		}
		err := validator.Validate("full_types", data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected number")
	})

	t.Run("invalid boolean type", func(t *testing.T) {
		data := map[string]interface{}{
			"active": "yes", // should be boolean
		}
		err := validator.Validate("full_types", data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected boolean")
	})

	t.Run("invalid array type", func(t *testing.T) {
		data := map[string]interface{}{
			"tags": "tag1,tag2", // should be array
		}
		err := validator.Validate("full_types", data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected array")
	})

	t.Run("invalid object type", func(t *testing.T) {
		data := map[string]interface{}{
			"address": "123 Main St", // should be object
		}
		err := validator.Validate("full_types", data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected object")
	})
}

func TestSchemaValidator_ValidateType_Integer(t *testing.T) {
	validator := NewSchemaValidator()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"count": map[string]interface{}{"type": "integer"},
		},
	}

	err := validator.RegisterSchema("counter", schema)
	require.NoError(t, err)

	t.Run("integer value validates correctly", func(t *testing.T) {
		// The new validator handles integer type correctly
		data := map[string]interface{}{
			"count": 42,
		}
		err := validator.Validate("counter", data)
		// Integer values should pass validation
		assert.NoError(t, err)
	})

	t.Run("float that is whole number passes as integer", func(t *testing.T) {
		// After JSON marshal/unmarshal, integers become float64
		// but our validator checks if float64 is a whole number
		data := map[string]interface{}{
			"count": float64(42),
		}
		err := validator.Validate("counter", data)
		assert.NoError(t, err)
	})

	t.Run("invalid integer (string)", func(t *testing.T) {
		data := map[string]interface{}{
			"count": "not a number",
		}
		err := validator.Validate("counter", data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected integer")
	})

	t.Run("float with decimals fails as integer", func(t *testing.T) {
		data := map[string]interface{}{
			"count": 42.5,
		}
		err := validator.Validate("counter", data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected integer")
	})
}

func TestSchemaValidator_ValidateRequest(t *testing.T) {
	validator := NewSchemaValidator()

	schema := map[string]interface{}{
		"type":     "object",
		"required": []interface{}{"name"},
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
			"age":  map[string]interface{}{"type": "number"},
		},
	}

	err := validator.RegisterSchema("person", schema)
	require.NoError(t, err)

	t.Run("valid request", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name": "John", "age": 30}`)
		req := httptest.NewRequest("POST", "/test", body)

		var target map[string]interface{}
		err := validator.ValidateRequest("person", req, &target)

		assert.NoError(t, err)
		assert.Equal(t, "John", target["name"])
	})

	t.Run("invalid JSON", func(t *testing.T) {
		body := bytes.NewBufferString(`{invalid json}`)
		req := httptest.NewRequest("POST", "/test", body)

		var target map[string]interface{}
		err := validator.ValidateRequest("person", req, &target)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode request body")
	})

	t.Run("missing required field", func(t *testing.T) {
		body := bytes.NewBufferString(`{"age": 30}`)
		req := httptest.NewRequest("POST", "/test", body)

		var target map[string]interface{}
		err := validator.ValidateRequest("person", req, &target)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required field is missing")
	})
}

func TestSchemaValidator_ValidateResponse(t *testing.T) {
	validator := NewSchemaValidator()

	schema := map[string]interface{}{
		"type":     "object",
		"required": []interface{}{"status"},
		"properties": map[string]interface{}{
			"status": map[string]interface{}{"type": "string"},
		},
	}

	err := validator.RegisterSchema("response", schema)
	require.NoError(t, err)

	t.Run("valid response", func(t *testing.T) {
		response := map[string]interface{}{
			"status": "success",
		}
		err := validator.ValidateResponse("response", response)
		assert.NoError(t, err)
	})

	t.Run("invalid response", func(t *testing.T) {
		response := map[string]interface{}{
			"message": "success", // missing required 'status'
		}
		err := validator.ValidateResponse("response", response)
		assert.Error(t, err)
	})
}

func TestPredefinedSchemas(t *testing.T) {
	t.Run("ModelRequestSchema structure", func(t *testing.T) {
		assert.NotNil(t, ModelRequestSchema)
		assert.Equal(t, "object", ModelRequestSchema["type"])

		required := ModelRequestSchema["required"].([]interface{})
		assert.Contains(t, required, "name")
		assert.Contains(t, required, "provider_id")
	})

	t.Run("VerificationRequestSchema structure", func(t *testing.T) {
		assert.NotNil(t, VerificationRequestSchema)
		assert.Equal(t, "object", VerificationRequestSchema["type"])

		required := VerificationRequestSchema["required"].([]interface{})
		assert.Contains(t, required, "model_id")
	})

	t.Run("ExportRequestSchema structure", func(t *testing.T) {
		assert.NotNil(t, ExportRequestSchema)
		assert.Equal(t, "object", ExportRequestSchema["type"])

		required := ExportRequestSchema["required"].([]interface{})
		assert.Contains(t, required, "export_type")
	})
}

func TestSchemaValidator_ValidateModelRequest(t *testing.T) {
	validator := NewSchemaValidator()
	err := validator.RegisterSchema("model", ModelRequestSchema)
	require.NoError(t, err)

	t.Run("valid model request", func(t *testing.T) {
		data := map[string]interface{}{
			"name":        "GPT-4",
			"provider_id": 1,
			"description": "OpenAI GPT-4 model",
		}
		err := validator.Validate("model", data)
		// With proper integer validation, this now passes
		assert.NoError(t, err)
	})

	t.Run("missing required fields", func(t *testing.T) {
		data := map[string]interface{}{
			"description": "Test model",
		}
		err := validator.Validate("model", data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required field is missing")
	})

	t.Run("missing provider_id", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "GPT-4",
		}
		err := validator.Validate("model", data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required field is missing")
	})
}


func TestSchemaValidator_BasicValidation_EdgeCases(t *testing.T) {
	validator := NewSchemaValidator()

	t.Run("schema marshal error", func(t *testing.T) {
		// Channel cannot be marshaled to JSON
		badSchema := make(chan int)
		err := validator.RegisterSchema("bad", badSchema)

		// RegisterSchema now returns error for unmarshalable schema
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal schema")
	})

	t.Run("data marshal error", func(t *testing.T) {
		schema := map[string]interface{}{"type": "object"}
		err := validator.RegisterSchema("good", schema)
		require.NoError(t, err)

		// Channel cannot be marshaled - this is caught during validation
		err = validator.Validate("good", make(chan int))
		assert.Error(t, err)
		// The error comes from toMap which does JSON marshal
	})

	t.Run("unknown type accepted", func(t *testing.T) {
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"custom": map[string]interface{}{"type": "custom_type"},
			},
		}
		err := validator.RegisterSchema("custom", schema)
		require.NoError(t, err)

		data := map[string]interface{}{
			"custom": "any value",
		}
		err = validator.Validate("custom", data)
		assert.NoError(t, err) // Unknown types are accepted
	})
}

