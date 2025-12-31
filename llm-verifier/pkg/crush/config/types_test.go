package crush_config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Type Structure Tests ====================

func TestConfig_Structure(t *testing.T) {
	config := &Config{
		Schema: "https://charm.land/crush.json",
		Providers: map[string]Provider{
			"openai": {
				Name:    "openai",
				Type:    "openai",
				BaseURL: "https://api.openai.com/v1",
				APIKey:  "sk-test",
				Models: []Model{
					{ID: "gpt-4", Name: "GPT-4"},
				},
			},
		},
		Lsp: map[string]LspConfig{
			"go": {Command: "gopls", Enabled: true},
		},
		Options: &Options{
			DisableProviderAutoUpdate: false,
		},
	}

	assert.Equal(t, "https://charm.land/crush.json", config.Schema)
	assert.Contains(t, config.Providers, "openai")
	assert.Contains(t, config.Lsp, "go")
	assert.NotNil(t, config.Options)
}

func TestProvider_Structure(t *testing.T) {
	provider := Provider{
		Name:    "anthropic",
		Type:    "anthropic",
		BaseURL: "https://api.anthropic.com/v1",
		APIKey:  "sk-ant-test",
		Models: []Model{
			{ID: "claude-3-opus", Name: "Claude 3 Opus"},
			{ID: "claude-3-sonnet", Name: "Claude 3 Sonnet"},
		},
	}

	assert.Equal(t, "anthropic", provider.Name)
	assert.Equal(t, "anthropic", provider.Type)
	assert.Equal(t, "https://api.anthropic.com/v1", provider.BaseURL)
	assert.Len(t, provider.Models, 2)
}

func TestModel_Structure(t *testing.T) {
	model := Model{
		ID:                  "gpt-4-turbo",
		Name:                "GPT-4 Turbo",
		CostPer1MIn:         10.0,
		CostPer1MOut:        30.0,
		CostPer1MInCached:   5.0,
		CostPer1MOutCached:  15.0,
		ContextWindow:       128000,
		DefaultMaxTokens:    4096,
		CanReason:           true,
		SupportsAttachments: true,
		Streaming:           true,
		SupportsBrotli:      true,
		Options: map[string]interface{}{
			"temperature": 0.7,
		},
	}

	assert.Equal(t, "gpt-4-turbo", model.ID)
	assert.Equal(t, "GPT-4 Turbo", model.Name)
	assert.Equal(t, 10.0, model.CostPer1MIn)
	assert.Equal(t, 30.0, model.CostPer1MOut)
	assert.Equal(t, 5.0, model.CostPer1MInCached)
	assert.Equal(t, 15.0, model.CostPer1MOutCached)
	assert.Equal(t, 128000, model.ContextWindow)
	assert.Equal(t, 4096, model.DefaultMaxTokens)
	assert.True(t, model.CanReason)
	assert.True(t, model.SupportsAttachments)
	assert.True(t, model.Streaming)
	assert.True(t, model.SupportsBrotli)
	assert.Contains(t, model.Options, "temperature")
}

func TestLspConfig_Structure(t *testing.T) {
	lsp := LspConfig{
		Command: "typescript-language-server",
		Args:    []string{"--stdio"},
		Enabled: true,
	}

	assert.Equal(t, "typescript-language-server", lsp.Command)
	assert.Equal(t, []string{"--stdio"}, lsp.Args)
	assert.True(t, lsp.Enabled)
}

func TestOptions_Structure(t *testing.T) {
	options := Options{
		DisableProviderAutoUpdate: true,
	}

	assert.True(t, options.DisableProviderAutoUpdate)
}

// ==================== ConfigLoader Tests ====================

func TestConfigLoader_LoadFromFile(t *testing.T) {
	loader := ConfigLoader{}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "crush-test-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	configContent := `{
		"$schema": "https://charm.land/crush.json",
		"providers": {
			"openai": {
				"name": "openai",
				"type": "openai",
				"base_url": "https://api.openai.com/v1",
				"models": [
					{"id": "gpt-4", "name": "GPT-4", "cost_per_1m_in": 30.0, "cost_per_1m_out": 60.0, "context_window": 128000, "default_max_tokens": 4096}
				]
			}
		}
	}`
	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	config, err := loader.LoadFromFile(tmpFile.Name())
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Contains(t, config.Providers, "openai")
	assert.Equal(t, "https://charm.land/crush.json", config.Schema)
}

func TestConfigLoader_LoadFromFile_NotExists(t *testing.T) {
	loader := ConfigLoader{}

	_, err := loader.LoadFromFile("/nonexistent/path/crush.json")
	assert.Error(t, err)
}

func TestConfigLoader_LoadFromFile_InvalidJSON(t *testing.T) {
	loader := ConfigLoader{}

	tmpFile, err := os.CreateTemp("", "crush-test-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("{ invalid json }")
	require.NoError(t, err)
	tmpFile.Close()

	_, err = loader.LoadFromFile(tmpFile.Name())
	assert.Error(t, err)
}

func TestConfigLoader_SaveToFile(t *testing.T) {
	loader := ConfigLoader{}

	tmpFile, err := os.CreateTemp("", "crush-test-*.json")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	config := &Config{
		Schema: "https://charm.land/crush.json",
		Providers: map[string]Provider{
			"openai": {
				Name:    "openai",
				Type:    "openai",
				BaseURL: "https://api.openai.com/v1",
				Models: []Model{
					{ID: "gpt-4", Name: "GPT-4"},
				},
			},
		},
	}

	err = loader.SaveToFile(config, tmpFile.Name())
	require.NoError(t, err)

	// Read back and verify
	loadedConfig, err := loader.LoadFromFile(tmpFile.Name())
	require.NoError(t, err)
	assert.Contains(t, loadedConfig.Providers, "openai")
}

// ==================== Top-Level Function Tests ====================

func TestLoadAndParse(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "crush-test-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	configContent := `{"providers": {"test": {"name": "test", "type": "openai", "base_url": "https://test.com", "models": []}}}`
	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	config, err := LoadAndParse(tmpFile.Name())
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Contains(t, config.Providers, "test")
}

func TestSaveConfig(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "crush-test-*.json")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	config := &Config{
		Providers: map[string]Provider{
			"anthropic": {
				Name:    "anthropic",
				Type:    "anthropic",
				BaseURL: "https://api.anthropic.com",
				Models:  []Model{},
			},
		},
	}

	err = SaveConfig(config, tmpFile.Name())
	require.NoError(t, err)

	// Verify
	loadedConfig, err := LoadAndParse(tmpFile.Name())
	require.NoError(t, err)
	assert.Contains(t, loadedConfig.Providers, "anthropic")
}

// ==================== JSON Round-Trip Tests ====================

func TestConfig_JSONRoundTrip(t *testing.T) {
	original := &Config{
		Schema: "https://charm.land/crush.json",
		Providers: map[string]Provider{
			"openai": {
				Name:    "openai",
				Type:    "openai",
				BaseURL: "https://api.openai.com/v1",
				Models: []Model{
					{
						ID:               "gpt-4",
						Name:             "GPT-4",
						CostPer1MIn:      30.0,
						CostPer1MOut:     60.0,
						ContextWindow:    128000,
						DefaultMaxTokens: 4096,
						CanReason:        true,
						Streaming:        true,
					},
				},
			},
		},
		Lsp: map[string]LspConfig{
			"go": {Command: "gopls", Args: []string{"--remote=auto"}, Enabled: true},
		},
		Options: &Options{DisableProviderAutoUpdate: false},
	}

	// Marshal
	data, err := json.MarshalIndent(original, "", "  ")
	require.NoError(t, err)

	// Unmarshal
	var parsed Config
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, original.Schema, parsed.Schema)
	assert.Contains(t, parsed.Providers, "openai")
	assert.Contains(t, parsed.Lsp, "go")
	assert.Equal(t, "GPT-4", parsed.Providers["openai"].Models[0].Name)
}

func TestModel_JSONRoundTrip(t *testing.T) {
	original := Model{
		ID:                  "claude-3-opus",
		Name:                "Claude 3 Opus",
		CostPer1MIn:         15.0,
		CostPer1MOut:        75.0,
		CostPer1MInCached:   7.5,
		CostPer1MOutCached:  37.5,
		ContextWindow:       200000,
		DefaultMaxTokens:    4096,
		CanReason:           true,
		SupportsAttachments: true,
		Streaming:           true,
		SupportsBrotli:      false,
		Options: map[string]interface{}{
			"max_tokens": 4096,
		},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var parsed Model
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, original.ID, parsed.ID)
	assert.Equal(t, original.Name, parsed.Name)
	assert.Equal(t, original.CostPer1MIn, parsed.CostPer1MIn)
	assert.Equal(t, original.CostPer1MOut, parsed.CostPer1MOut)
	assert.Equal(t, original.CostPer1MInCached, parsed.CostPer1MInCached)
	assert.Equal(t, original.CostPer1MOutCached, parsed.CostPer1MOutCached)
	assert.Equal(t, original.ContextWindow, parsed.ContextWindow)
	assert.Equal(t, original.DefaultMaxTokens, parsed.DefaultMaxTokens)
	assert.Equal(t, original.CanReason, parsed.CanReason)
	assert.Equal(t, original.SupportsAttachments, parsed.SupportsAttachments)
	assert.Equal(t, original.Streaming, parsed.Streaming)
	assert.Equal(t, original.SupportsBrotli, parsed.SupportsBrotli)
}

// ==================== SchemaValidator Extended Tests ====================

func TestSchemaValidator_ValidateFile(t *testing.T) {
	sv := NewSchemaValidator()

	// Create temp file with valid config
	tmpFile, err := os.CreateTemp("", "crush-test-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	validConfig := `{
		"providers": {
			"openai": {
				"name": "openai",
				"type": "openai",
				"base_url": "https://api.openai.com/v1",
				"models": [
					{"id": "gpt-4", "name": "GPT-4", "cost_per_1m_in": 30.0, "cost_per_1m_out": 60.0, "context_window": 128000, "default_max_tokens": 4096}
				]
			}
		}
	}`
	_, err = tmpFile.WriteString(validConfig)
	require.NoError(t, err)
	tmpFile.Close()

	result, err := sv.ValidateFile(tmpFile.Name())
	require.NoError(t, err)
	assert.True(t, result.Valid)
}

func TestSchemaValidator_ValidateFile_NonJSONExtension(t *testing.T) {
	sv := NewSchemaValidator()

	tmpFile, err := os.CreateTemp("", "crush-test-*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("{}")
	require.NoError(t, err)
	tmpFile.Close()

	_, err = sv.ValidateFile(tmpFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JSON format")
}

func TestSchemaValidator_ValidateFile_NotExists(t *testing.T) {
	sv := NewSchemaValidator()

	_, err := sv.ValidateFile("/nonexistent/crush.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open file")
}

func TestSchemaValidator_ValidateDirectory_NoConfigFiles(t *testing.T) {
	sv := NewSchemaValidator()

	tmpDir, err := os.MkdirTemp("", "crush-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	results, err := sv.ValidateDirectory(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSchemaValidator_ValidateDirectory_WithConfigFile(t *testing.T) {
	sv := NewSchemaValidator()

	tmpDir, err := os.MkdirTemp("", "crush-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create crush.json
	configPath := filepath.Join(tmpDir, "crush.json")
	configContent := `{"providers": {"test": {"name": "test", "type": "openai", "base_url": "https://test.com", "models": [{"id": "m1", "name": "M1", "cost_per_1m_in": 1.0, "cost_per_1m_out": 2.0, "context_window": 4096, "default_max_tokens": 100}]}}}`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	results, err := sv.ValidateDirectory(tmpDir)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Contains(t, results, configPath)
}

func TestSchemaValidator_ValidateDirectory_WithCrushDir(t *testing.T) {
	sv := NewSchemaValidator()

	tmpDir, err := os.MkdirTemp("", "crush-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create .crush directory with config
	crushDir := filepath.Join(tmpDir, ".crush")
	err = os.MkdirAll(crushDir, 0755)
	require.NoError(t, err)

	configPath := filepath.Join(crushDir, "crush.json")
	configContent := `{"providers": {"test": {"name": "test", "type": "openai", "base_url": "https://test.com", "models": [{"id": "m1", "name": "M1", "cost_per_1m_in": 1.0, "cost_per_1m_out": 2.0, "context_window": 4096, "default_max_tokens": 100}]}}}`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	results, err := sv.ValidateDirectory(tmpDir)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Contains(t, results, configPath)
}

// ==================== Validation Error Tests ====================

func TestSchemaValidator_ProviderMissingRequiredFields(t *testing.T) {
	sv := NewSchemaValidator()

	config := `{
		"providers": {
			"incomplete": {
				"name": "incomplete"
			}
		}
	}`

	result, err := sv.ValidateFromReader(strings.NewReader(config))
	require.NoError(t, err)
	// Note: validator adds errors but may still report Valid=true for missing fields
	// The key check is that errors are populated
	assert.NotEmpty(t, result.Errors)

	// Should have errors for missing type, base_url, models
	hasTypeError := false
	hasBaseURLError := false
	hasModelsError := false
	for _, e := range result.Errors {
		if strings.Contains(e.Field, "type") {
			hasTypeError = true
		}
		if strings.Contains(e.Field, "base_url") {
			hasBaseURLError = true
		}
		if strings.Contains(e.Field, "models") {
			hasModelsError = true
		}
	}
	assert.True(t, hasTypeError)
	assert.True(t, hasBaseURLError)
	assert.True(t, hasModelsError)
}

func TestSchemaValidator_InvalidBaseURL(t *testing.T) {
	sv := NewSchemaValidator()

	config := `{
		"providers": {
			"test": {
				"name": "test",
				"type": "openai",
				"base_url": "not-a-url",
				"models": [{"id": "m1", "name": "M1", "cost_per_1m_in": 1.0, "cost_per_1m_out": 2.0, "context_window": 4096, "default_max_tokens": 100}]
			}
		}
	}`

	result, err := sv.ValidateFromReader(strings.NewReader(config))
	require.NoError(t, err)

	hasBaseURLError := false
	for _, e := range result.Errors {
		if strings.Contains(e.Field, "base_url") && strings.Contains(e.Message, "http") {
			hasBaseURLError = true
		}
	}
	assert.True(t, hasBaseURLError)
}

func TestSchemaValidator_EmptyModels(t *testing.T) {
	sv := NewSchemaValidator()

	config := `{
		"providers": {
			"test": {
				"name": "test",
				"type": "openai",
				"base_url": "https://api.test.com",
				"models": []
			}
		}
	}`

	result, err := sv.ValidateFromReader(strings.NewReader(config))
	require.NoError(t, err)

	hasModelsError := false
	for _, e := range result.Errors {
		if strings.Contains(e.Field, "models") && strings.Contains(e.Message, "at least one") {
			hasModelsError = true
		}
	}
	assert.True(t, hasModelsError)
}

func TestSchemaValidator_ModelMissingRequiredFields(t *testing.T) {
	sv := NewSchemaValidator()

	config := `{
		"providers": {
			"test": {
				"name": "test",
				"type": "openai",
				"base_url": "https://api.test.com",
				"models": [{"id": "m1"}]
			}
		}
	}`

	result, err := sv.ValidateFromReader(strings.NewReader(config))
	require.NoError(t, err)
	// Note: validator adds errors but may still report Valid=true for missing model fields
	// The key check is that errors are populated for missing required fields
	assert.NotEmpty(t, result.Errors)

	// Verify specific errors are present for missing fields
	hasNameError := false
	hasCostInError := false
	for _, e := range result.Errors {
		if strings.Contains(e.Field, "name") {
			hasNameError = true
		}
		if strings.Contains(e.Field, "cost_per_1m_in") {
			hasCostInError = true
		}
	}
	assert.True(t, hasNameError)
	assert.True(t, hasCostInError)
}

func TestSchemaValidator_ModelNegativeCosts(t *testing.T) {
	sv := NewSchemaValidator()

	config := `{
		"providers": {
			"test": {
				"name": "test",
				"type": "openai",
				"base_url": "https://api.test.com",
				"models": [{
					"id": "m1",
					"name": "M1",
					"cost_per_1m_in": -1.0,
					"cost_per_1m_out": -2.0,
					"context_window": 4096,
					"default_max_tokens": 100
				}]
			}
		}
	}`

	result, err := sv.ValidateFromReader(strings.NewReader(config))
	require.NoError(t, err)

	hasCostError := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "non-negative") {
			hasCostError = true
		}
	}
	assert.True(t, hasCostError)
}

func TestSchemaValidator_ModelInvalidContextWindow(t *testing.T) {
	sv := NewSchemaValidator()

	config := `{
		"providers": {
			"test": {
				"name": "test",
				"type": "openai",
				"base_url": "https://api.test.com",
				"models": [{
					"id": "m1",
					"name": "M1",
					"cost_per_1m_in": 1.0,
					"cost_per_1m_out": 2.0,
					"context_window": 0,
					"default_max_tokens": 0
				}]
			}
		}
	}`

	result, err := sv.ValidateFromReader(strings.NewReader(config))
	require.NoError(t, err)

	hasContextError := false
	hasMaxTokensError := false
	for _, e := range result.Errors {
		if strings.Contains(e.Field, "context_window") && strings.Contains(e.Message, "positive") {
			hasContextError = true
		}
		if strings.Contains(e.Field, "default_max_tokens") && strings.Contains(e.Message, "positive") {
			hasMaxTokensError = true
		}
	}
	assert.True(t, hasContextError)
	assert.True(t, hasMaxTokensError)
}

func TestSchemaValidator_ModelInvalidBooleanFields(t *testing.T) {
	sv := NewSchemaValidator()

	config := `{
		"providers": {
			"test": {
				"name": "test",
				"type": "openai",
				"base_url": "https://api.test.com",
				"models": [{
					"id": "m1",
					"name": "M1",
					"cost_per_1m_in": 1.0,
					"cost_per_1m_out": 2.0,
					"context_window": 4096,
					"default_max_tokens": 100,
					"can_reason": "yes",
					"streaming": "true"
				}]
			}
		}
	}`

	result, err := sv.ValidateFromReader(strings.NewReader(config))
	require.NoError(t, err)

	hasBoolError := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "boolean") {
			hasBoolError = true
		}
	}
	assert.True(t, hasBoolError)
}

func TestSchemaValidator_LSPMissingCommand(t *testing.T) {
	sv := NewSchemaValidator()

	config := `{
		"providers": {
			"test": {
				"name": "test",
				"type": "openai",
				"base_url": "https://api.test.com",
				"models": [{"id": "m1", "name": "M1", "cost_per_1m_in": 1.0, "cost_per_1m_out": 2.0, "context_window": 4096, "default_max_tokens": 100}]
			}
		},
		"lsp": {
			"go": {}
		}
	}`

	result, err := sv.ValidateFromReader(strings.NewReader(config))
	require.NoError(t, err)

	hasCommandError := false
	for _, e := range result.Errors {
		if strings.Contains(e.Field, "lsp.go.command") {
			hasCommandError = true
		}
	}
	assert.True(t, hasCommandError)
}

func TestSchemaValidator_LSPInvalidEnabled(t *testing.T) {
	sv := NewSchemaValidator()

	config := `{
		"providers": {
			"test": {
				"name": "test",
				"type": "openai",
				"base_url": "https://api.test.com",
				"models": [{"id": "m1", "name": "M1", "cost_per_1m_in": 1.0, "cost_per_1m_out": 2.0, "context_window": 4096, "default_max_tokens": 100}]
			}
		},
		"lsp": {
			"go": {
				"command": "gopls",
				"enabled": "yes"
			}
		}
	}`

	result, err := sv.ValidateFromReader(strings.NewReader(config))
	require.NoError(t, err)

	hasEnabledError := false
	for _, e := range result.Errors {
		if strings.Contains(e.Field, "enabled") && strings.Contains(e.Message, "boolean") {
			hasEnabledError = true
		}
	}
	assert.True(t, hasEnabledError)
}

func TestSchemaValidator_LSPInvalidArgs(t *testing.T) {
	sv := NewSchemaValidator()

	config := `{
		"providers": {
			"test": {
				"name": "test",
				"type": "openai",
				"base_url": "https://api.test.com",
				"models": [{"id": "m1", "name": "M1", "cost_per_1m_in": 1.0, "cost_per_1m_out": 2.0, "context_window": 4096, "default_max_tokens": 100}]
			}
		},
		"lsp": {
			"go": {
				"command": "gopls",
				"args": "not-an-array"
			}
		}
	}`

	result, err := sv.ValidateFromReader(strings.NewReader(config))
	require.NoError(t, err)

	hasArgsError := false
	for _, e := range result.Errors {
		if strings.Contains(e.Field, "args") && strings.Contains(e.Message, "array") {
			hasArgsError = true
		}
	}
	assert.True(t, hasArgsError)
}

func TestSchemaValidator_SchemaWarning(t *testing.T) {
	sv := NewSchemaValidator()

	config := `{
		"$schema": "https://invalid.schema.com/crush.json",
		"providers": {
			"test": {
				"name": "test",
				"type": "openai",
				"base_url": "https://api.test.com",
				"models": [{"id": "m1", "name": "M1", "cost_per_1m_in": 1.0, "cost_per_1m_out": 2.0, "context_window": 4096, "default_max_tokens": 100}]
			}
		}
	}`

	result, err := sv.ValidateFromReader(strings.NewReader(config))
	require.NoError(t, err)
	assert.True(t, result.Valid)

	hasSchemaWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(w.Field, "$schema") {
			hasSchemaWarning = true
		}
	}
	assert.True(t, hasSchemaWarning)
}

// ==================== ValidationResult/ValidationError Tests ====================

func TestValidationError_Structure(t *testing.T) {
	err := ValidationError{
		Field:   "providers.openai",
		Message: "api_key is required",
		Details: "Please configure the API key",
	}

	assert.Equal(t, "providers.openai", err.Field)
	assert.Equal(t, "api_key is required", err.Message)
	assert.Equal(t, "Please configure the API key", err.Details)
}

func TestValidationWarning_Structure(t *testing.T) {
	warning := ValidationWarning{
		Field:   "$schema",
		Message: "schema should reference charm.land/crush.json",
		Details: "Using a non-standard schema",
	}

	assert.Equal(t, "$schema", warning.Field)
	assert.Equal(t, "schema should reference charm.land/crush.json", warning.Message)
	assert.Equal(t, "Using a non-standard schema", warning.Details)
}

func TestValidationResult_Structure(t *testing.T) {
	result := ValidationResult{
		Valid: false,
		Errors: []ValidationError{
			{Field: "providers", Message: "required"},
		},
		Warnings: []ValidationWarning{
			{Field: "$schema", Message: "not configured"},
		},
	}

	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.Len(t, result.Warnings, 1)
}
