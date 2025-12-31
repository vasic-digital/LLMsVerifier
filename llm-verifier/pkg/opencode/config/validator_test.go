package opencode_config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== SchemaValidator Tests ====================

func TestNewSchemaValidator(t *testing.T) {
	sv := NewSchemaValidator()
	require.NotNil(t, sv)
}

func TestSchemaValidator_ValidateFromReader_ValidConfig(t *testing.T) {
	sv := NewSchemaValidator()

	validJSON := `{
		"provider": {
			"openai": {
				"options": {
					"api_key": "sk-test123"
				}
			}
		}
	}`

	result, err := sv.ValidateFromReader(bytes.NewReader([]byte(validJSON)))
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestSchemaValidator_ValidateFromReader_MissingProvider(t *testing.T) {
	sv := NewSchemaValidator()

	invalidJSON := `{
		"agent": {
			"test": {
				"model": "gpt-4"
			}
		}
	}`

	result, err := sv.ValidateFromReader(bytes.NewReader([]byte(invalidJSON)))
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)

	// Check for provider error
	hasProviderError := false
	for _, e := range result.Errors {
		if e.Field == "provider" {
			hasProviderError = true
			break
		}
	}
	assert.True(t, hasProviderError)
}

func TestSchemaValidator_ValidateFromReader_InvalidJSON(t *testing.T) {
	sv := NewSchemaValidator()

	invalidJSON := `{ invalid json }`

	_, err := sv.ValidateFromReader(bytes.NewReader([]byte(invalidJSON)))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")
}

func TestSchemaValidator_ValidateFromReader_ProviderWithNoConfig(t *testing.T) {
	sv := NewSchemaValidator()

	json := `{
		"provider": {
			"test": {}
		}
	}`

	result, err := sv.ValidateFromReader(bytes.NewReader([]byte(json)))
	require.NoError(t, err)
	// Should have an error about missing options/model
	hasError := false
	for _, e := range result.Errors {
		if e.Field == "provider.test" {
			hasError = true
			break
		}
	}
	assert.True(t, hasError)
}

func TestSchemaValidator_ValidateFromReader_AgentWithNoModelOrPrompt(t *testing.T) {
	sv := NewSchemaValidator()

	json := `{
		"provider": {
			"openai": {
				"model": "gpt-4"
			}
		},
		"agent": {
			"test-agent": {}
		}
	}`

	result, err := sv.ValidateFromReader(bytes.NewReader([]byte(json)))
	require.NoError(t, err)
	// Should have an error about missing model/prompt
	hasError := false
	for _, e := range result.Errors {
		if e.Field == "agent.test-agent" {
			hasError = true
			break
		}
	}
	assert.True(t, hasError)
}

func TestSchemaValidator_ValidateFromReader_MCPWithInvalidType(t *testing.T) {
	sv := NewSchemaValidator()

	json := `{
		"provider": {
			"openai": {"model": "gpt-4"}
		},
		"mcp": {
			"test-mcp": {
				"type": "invalid-type"
			}
		}
	}`

	result, err := sv.ValidateFromReader(bytes.NewReader([]byte(json)))
	require.NoError(t, err)
	// Should have an error about invalid type
	hasError := false
	for _, e := range result.Errors {
		if e.Field == "mcp.test-mcp.type" {
			hasError = true
			break
		}
	}
	assert.True(t, hasError)
}

func TestSchemaValidator_ValidateFromReader_MCPLocalMissingCommand(t *testing.T) {
	sv := NewSchemaValidator()

	json := `{
		"provider": {
			"openai": {"model": "gpt-4"}
		},
		"mcp": {
			"test-mcp": {
				"type": "local"
			}
		}
	}`

	result, err := sv.ValidateFromReader(bytes.NewReader([]byte(json)))
	require.NoError(t, err)
	// Should have an error about missing command
	hasError := false
	for _, e := range result.Errors {
		if e.Field == "mcp.test-mcp.command" {
			hasError = true
			break
		}
	}
	assert.True(t, hasError)
}

func TestSchemaValidator_ValidateFromReader_MCPRemoteMissingURL(t *testing.T) {
	sv := NewSchemaValidator()

	json := `{
		"provider": {
			"openai": {"model": "gpt-4"}
		},
		"mcp": {
			"test-mcp": {
				"type": "remote"
			}
		}
	}`

	result, err := sv.ValidateFromReader(bytes.NewReader([]byte(json)))
	require.NoError(t, err)
	// Should have an error about missing URL
	hasError := false
	for _, e := range result.Errors {
		if e.Field == "mcp.test-mcp.url" {
			hasError = true
			break
		}
	}
	assert.True(t, hasError)
}

func TestSchemaValidator_ValidateFile(t *testing.T) {
	sv := NewSchemaValidator()

	// Create temp file
	tmpFile, err := os.CreateTemp("", "opencode-test-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	validConfig := `{
		"provider": {
			"openai": {"model": "gpt-4"}
		}
	}`
	_, err = tmpFile.WriteString(validConfig)
	require.NoError(t, err)
	tmpFile.Close()

	result, err := sv.ValidateFile(tmpFile.Name())
	require.NoError(t, err)
	assert.True(t, result.Valid)
}

func TestSchemaValidator_ValidateFile_NotExists(t *testing.T) {
	sv := NewSchemaValidator()

	_, err := sv.ValidateFile("/nonexistent/path/config.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open file")
}

func TestSchemaValidator_ValidateFile_WithJSONComments(t *testing.T) {
	sv := NewSchemaValidator()

	// Create temp file with JSONC comments
	tmpFile, err := os.CreateTemp("", "opencode-test-*.jsonc")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	jsoncConfig := `{
		// This is a comment
		"provider": {
			"openai": {"model": "gpt-4"} // inline comment
		}
	}`
	_, err = tmpFile.WriteString(jsoncConfig)
	require.NoError(t, err)
	tmpFile.Close()

	result, err := sv.ValidateFile(tmpFile.Name())
	require.NoError(t, err)
	assert.True(t, result.Valid)
}

func TestSchemaValidator_ValidateDirectory_NoConfigFiles(t *testing.T) {
	sv := NewSchemaValidator()

	tmpDir, err := os.MkdirTemp("", "opencode-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	results, err := sv.ValidateDirectory(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSchemaValidator_ValidateDirectory_WithConfigFile(t *testing.T) {
	sv := NewSchemaValidator()

	tmpDir, err := os.MkdirTemp("", "opencode-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create opencode.json
	configPath := filepath.Join(tmpDir, "opencode.json")
	configContent := `{"provider": {"test": {"model": "gpt-4"}}}`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	results, err := sv.ValidateDirectory(tmpDir)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Contains(t, results, configPath)
}

func TestSchemaValidator_ValidateDirectory_WithOpenCodeDir(t *testing.T) {
	sv := NewSchemaValidator()

	tmpDir, err := os.MkdirTemp("", "opencode-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create .opencode directory with config
	openCodeDir := filepath.Join(tmpDir, ".opencode")
	err = os.MkdirAll(openCodeDir, 0755)
	require.NoError(t, err)

	configPath := filepath.Join(openCodeDir, "opencode.json")
	configContent := `{"provider": {"test": {"model": "gpt-4"}}}`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	results, err := sv.ValidateDirectory(tmpDir)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Contains(t, results, configPath)
}

// ==================== ConfigLoader Tests ====================

func TestConfigLoader_LoadFromFile(t *testing.T) {
	loader := ConfigLoader{}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "opencode-test-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	configContent := `{
		"provider": {
			"openai": {"model": "gpt-4"}
		},
		"agent": {
			"build": {"model": "gpt-4-turbo"}
		}
	}`
	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	config, err := loader.LoadFromFile(tmpFile.Name())
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Contains(t, config.Provider, "openai")
	assert.Contains(t, config.Agent, "build")
}

func TestConfigLoader_LoadFromFile_NotExists(t *testing.T) {
	loader := ConfigLoader{}

	_, err := loader.LoadFromFile("/nonexistent/config.json")
	assert.Error(t, err)
}

func TestConfigLoader_LoadFromFile_InvalidJSON(t *testing.T) {
	loader := ConfigLoader{}

	tmpFile, err := os.CreateTemp("", "opencode-test-*.json")
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

	tmpFile, err := os.CreateTemp("", "opencode-test-*.json")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	config := &Config{
		Provider: map[string]ProviderConfig{
			"openai": {Model: "gpt-4"},
		},
	}

	err = loader.SaveToFile(config, tmpFile.Name())
	require.NoError(t, err)

	// Read back and verify
	loadedConfig, err := loader.LoadFromFile(tmpFile.Name())
	require.NoError(t, err)
	assert.Contains(t, loadedConfig.Provider, "openai")
	assert.Equal(t, "gpt-4", loadedConfig.Provider["openai"].Model)
}

// ==================== Helper Function Tests ====================

func TestStripJSONCComments_SingleLineComment(t *testing.T) {
	input := `{
		// comment
		"key": "value"
	}`

	result := stripJSONCComments(input)
	assert.NotContains(t, result, "// comment")
	assert.Contains(t, result, "\"key\": \"value\"")
}

func TestStripJSONCComments_InlineComment(t *testing.T) {
	input := `{"key": "value" // inline}`

	result := stripJSONCComments(input)
	assert.NotContains(t, result, "// inline")
	assert.Contains(t, result, "\"key\": \"value\"")
}

func TestStripJSONCComments_CommentInString(t *testing.T) {
	input := `{"url": "https://example.com/path"}`

	result := stripJSONCComments(input)
	// URL should be preserved
	assert.Contains(t, result, "https://example.com/path")
}

func TestStripJSONCComments_NoComments(t *testing.T) {
	input := `{"key": "value"}`

	result := stripJSONCComments(input)
	assert.Contains(t, result, "\"key\": \"value\"")
}

// ==================== Top-Level Function Tests ====================

func TestLoadAndParse(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "opencode-test-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	configContent := `{"provider": {"test": {"model": "gpt-4"}}}`
	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	config, err := LoadAndParse(tmpFile.Name())
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Contains(t, config.Provider, "test")
}

func TestSaveConfig(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "opencode-test-*.json")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	config := &Config{
		Provider: map[string]ProviderConfig{
			"anthropic": {Model: "claude-3"},
		},
	}

	err = SaveConfig(config, tmpFile.Name())
	require.NoError(t, err)

	// Verify
	loadedConfig, err := LoadAndParse(tmpFile.Name())
	require.NoError(t, err)
	assert.Contains(t, loadedConfig.Provider, "anthropic")
}

// ==================== ValidationResult/ValidationError Tests ====================

func TestValidationError_Structure(t *testing.T) {
	err := ValidationError{
		Field:   "provider.openai",
		Message: "api_key is required",
		Details: "Please configure the API key",
	}

	assert.Equal(t, "provider.openai", err.Field)
	assert.Equal(t, "api_key is required", err.Message)
	assert.Equal(t, "Please configure the API key", err.Details)
}

func TestValidationWarning_Structure(t *testing.T) {
	warning := ValidationWarning{
		Field:   "agent.test",
		Message: "temperature not set, using default",
		Details: "Default temperature is 0.7",
	}

	assert.Equal(t, "agent.test", warning.Field)
	assert.Equal(t, "temperature not set, using default", warning.Message)
	assert.Equal(t, "Default temperature is 0.7", warning.Details)
}

func TestValidationResult_Structure(t *testing.T) {
	result := ValidationResult{
		Valid: false,
		Errors: []ValidationError{
			{Field: "provider", Message: "required"},
		},
		Warnings: []ValidationWarning{
			{Field: "agent", Message: "not configured"},
		},
	}

	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.Len(t, result.Warnings, 1)
}
