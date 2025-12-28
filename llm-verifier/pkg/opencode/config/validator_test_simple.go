package opencode_config

import (
	"testing"
	"path/filepath"
	"os"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigLoaderLoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.json")

	testConfig := &Config{
		Provider: map[string]ProviderConfig{
			"openai": {
				Options: map[string]interface{}{
					"api_key": "test-key",
				},
			},
		},
	}

	loader := ConfigLoader{}
	err := loader.SaveToFile(testConfig, configPath)
	require.NoError(t, err)

	loadedConfig, err := loader.LoadFromFile(configPath)
	require.NoError(t, err)

	assert.Equal(t, 
		testConfig.Provider["openai"].Options["api_key"],
		loadedConfig.Provider["openai"].Options["api_key"],
	)
}

func TestCreateDefaultConfig(t *testing.T) {
	config := CreateDefaultConfig()
	
	assert.NotNil(t, config)
	assert.NotNil(t, config.Provider)
	assert.NotNil(t, config.Agent)
	
	assert.Contains(t, config.Provider, "openai")
	assert.Contains(t, config.Agent, "build")
}

func TestStripJSONCComments(t *testing.T) {
	input := `{
		"provider": {} // Test comment
	}`
	result := stripJSONCComments(input)
	assert.NotEmpty(t, result)
}