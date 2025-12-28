package crush_config

import (
	"bytes"
	"encoding/json"
	"testing"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSchemaValidator(t *testing.T) {
	validator := NewSchemaValidator()
	assert.NotNil(t, validator)
	assert.NotNil(t, validator.loader)
}

func TestValidateFromReader(t *testing.T) {
	tests := []struct {
		name        string
		config      map[string]interface{}
		expectValid bool
		errorCount  int
	}{
		{
			name: "minimal valid config",
			config: map[string]interface{}{
				"providers": map[string]interface{}{
					"openai": map[string]interface{}{
						"name":     "openai",
						"type":     "openai",
						"base_url": "https://api.openai.com/v1",
						"models": []interface{}{
							map[string]interface{}{
								"id":                   "gpt-4",
								"name":                 "GPT-4",
								"cost_per_1m_in":       30.0,
								"cost_per_1m_out":      60.0,
								"context_window":       128000,
								"default_max_tokens":   4096,
								"can_reason":          true,
								"supports_attachments": false,
								"streaming":           true,
							},
						},
					},
				},
			},
			expectValid: true,
			errorCount:  0,
		},
		{
			name: "config without providers",
			config: map[string]interface{}{
				"lsp": map[string]interface{}{},
			},
			expectValid: false,
			errorCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configJSON, err := json.Marshal(tt.config)
			require.NoError(t, err)

			validator := NewSchemaValidator()
			result, err := validator.ValidateFromReader(bytes.NewReader(configJSON))

			require.NoError(t, err)
			assert.Equal(t, tt.expectValid, result.Valid)
			assert.Equal(t, tt.errorCount, len(result.Errors))
		})
	}
}

func TestCreateDefaultConfig(t *testing.T) {
	config := CreateDefaultConfig()
	
	assert.NotNil(t, config)
	assert.NotNil(t, config.Providers)
	assert.NotNil(t, config.Options)
	
	assert.Contains(t, config.Providers, "openai")
	assert.Equal(t, "https://charm.land/crush.json", config.Schema)
}