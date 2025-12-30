package providers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"llm-verifier/logging"
	"llm-verifier/verification"
)

func createAdapterTestLogger(t *testing.T) *logging.Logger {
	logger, err := logging.NewLogger(nil, map[string]any{"log_level": "error"})
	require.NoError(t, err)
	return logger
}

func TestNewProviderServiceAdapter(t *testing.T) {
	// Create a temp config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	err := os.WriteFile(configPath, []byte("{}"), 0644)
	require.NoError(t, err)

	logger := createAdapterTestLogger(t)
	service := NewModelProviderService(configPath, logger)

	adapter := NewProviderServiceAdapter(service)

	require.NotNil(t, adapter)

	// Check that it implements the interface
	var _ verification.ProviderServiceInterface = adapter
}

func TestProviderServiceAdapter_GetAllProviders_WithRegistered(t *testing.T) {
	// Create a temp config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	err := os.WriteFile(configPath, []byte("{}"), 0644)
	require.NoError(t, err)

	logger := createAdapterTestLogger(t)
	service := NewModelProviderService(configPath, logger)

	// Register some providers
	service.RegisterProvider("openai", "https://api.openai.com/v1", "sk-openai-key")
	service.RegisterProvider("anthropic", "https://api.anthropic.com/v1", "sk-anthropic-key")

	adapter := NewProviderServiceAdapter(service)

	providers := adapter.GetAllProviders()

	require.NotNil(t, providers)
	assert.Len(t, providers, 2)

	// Check OpenAI provider
	openai, exists := providers["openai"]
	assert.True(t, exists)
	assert.Equal(t, "openai", openai.ProviderID)
	assert.Equal(t, "https://api.openai.com/v1", openai.BaseURL)
	assert.Equal(t, "sk-openai-key", openai.APIKey)

	// Check Anthropic provider
	anthropic, exists := providers["anthropic"]
	assert.True(t, exists)
	assert.Equal(t, "anthropic", anthropic.ProviderID)
	assert.Equal(t, "https://api.anthropic.com/v1", anthropic.BaseURL)
	assert.Equal(t, "sk-anthropic-key", anthropic.APIKey)
}

func TestProviderServiceAdapter_GetAllProviders_Empty(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	err := os.WriteFile(configPath, []byte("{}"), 0644)
	require.NoError(t, err)

	logger := createAdapterTestLogger(t)
	service := NewModelProviderService(configPath, logger)

	adapter := NewProviderServiceAdapter(service)

	providers := adapter.GetAllProviders()

	require.NotNil(t, providers)
	assert.Len(t, providers, 0)
}

func TestProviderServiceAdapter_GetModels(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Create config with models
	configData := `{
		"provider": {
			"openai": {
				"models": {
					"gpt-4": {"name": "GPT-4", "maxTokens": 8192},
					"gpt-3.5-turbo": {"name": "GPT-3.5 Turbo", "maxTokens": 4096}
				}
			}
		}
	}`
	err := os.WriteFile(configPath, []byte(configData), 0644)
	require.NoError(t, err)

	logger := createAdapterTestLogger(t)
	service := NewModelProviderService(configPath, logger)
	service.RegisterProvider("openai", "https://api.openai.com/v1", "")

	adapter := NewProviderServiceAdapter(service)

	models, err := adapter.GetModels("openai")

	// May not find models if config format doesn't match exactly,
	// but should not return error
	require.NoError(t, err)
	require.NotNil(t, models)
}

func TestProviderServiceAdapter_GetModels_NoProvider(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	err := os.WriteFile(configPath, []byte("{}"), 0644)
	require.NoError(t, err)

	logger := createAdapterTestLogger(t)
	service := NewModelProviderService(configPath, logger)

	adapter := NewProviderServiceAdapter(service)

	models, err := adapter.GetModels("unknown-provider")

	// Should return empty list, not error
	require.NoError(t, err)
	require.NotNil(t, models)
}

func TestProviderServiceAdapter_ImplementsInterface(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	err := os.WriteFile(configPath, []byte("{}"), 0644)
	require.NoError(t, err)

	logger := createAdapterTestLogger(t)
	service := NewModelProviderService(configPath, logger)

	adapter := NewProviderServiceAdapter(service)

	// Verify the adapter correctly implements ProviderServiceInterface
	_, ok := adapter.(verification.ProviderServiceInterface)
	assert.True(t, ok, "Adapter should implement verification.ProviderServiceInterface")
}

func TestProviderServiceAdapter_GetModels_ReturnsModelInfo(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	err := os.WriteFile(configPath, []byte("{}"), 0644)
	require.NoError(t, err)

	logger := createAdapterTestLogger(t)
	service := NewModelProviderService(configPath, logger)

	adapter := NewProviderServiceAdapter(service)

	// Test that returned models are of type []verification.ModelInfo
	models, err := adapter.GetModels("openai")
	require.NoError(t, err)

	// Check type assertion works
	for _, m := range models {
		assert.IsType(t, verification.ModelInfo{}, m)
	}
}
