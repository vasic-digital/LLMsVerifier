package unit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"llm-verifier/config"
	"llm-verifier/providers"
)

func TestConfiguration_LoadFromFile(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		expectError   bool
		validateFunc  func(t *testing.T, cfg *config.Config)
	}{
		{
			name: "Valid OpenCode configuration",
			configContent: `{
				"$schema": "https://opencode.sh/schema.json",
				"username": "testuser",
				"provider": {
					"openai": {
						"options": {
							"apiKey": "sk-test-key",
							"baseURL": "https://api.openai.com/v1"
						},
						"models": {
							"gpt-4": {
								"id": "gpt-4",
								"name": "GPT-4",
								"displayName": "GPT-4 (SC:9.0)",
								"provider": {
									"id": "openai",
									"npm": "@openai/sdk"
								},
								"maxTokens": 8192,
								"supportsHTTP3": true
							}
						}
					}
				},
				"agent": {
					"name": "test-agent"
				},
				"mcp": {
					"servers": []
				}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, cfg *config.Config) {
				assert.NotNil(t, cfg)
				assert.Equal(t, "testuser", cfg.Username)
				assert.Contains(t, cfg.Providers, "openai")
				assert.Contains(t, cfg.Providers["openai"].Models, "gpt-4")
			},
		},
		{
			name: "Valid Crush configuration",
			configContent: `{
				"$schema": "https://charm.land/crush.json",
				"providers": {
					"openai": {
						"name": "openai",
						"type": "openai",
						"base_url": "https://api.openai.com/v1",
						"api_key": "sk-test-key",
						"models": [
							{
								"id": "gpt-4",
								"name": "GPT-4",
								"cost_per_1m_in": 30.0,
								"cost_per_1m_out": 60.0,
								"context_window": 128000,
								"supports_brotli": true
							}
						]
					}
				}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, cfg *config.Config) {
				assert.NotNil(t, cfg)
				assert.Contains(t, cfg.Providers, "openai")
				assert.Len(t, cfg.Providers["openai"].Models, 1)
			},
		},
		{
			name: "Invalid JSON",
			configContent: `{
				"invalid": json content
			}`,
			expectError: true,
			validateFunc: func(t *testing.T, cfg *config.Config) {
				assert.Nil(t, cfg)
			},
		},
		{
			name: "Missing required fields",
			configContent: `{
				"provider": {}
			}`,
			expectError: true,
			validateFunc: func(t *testing.T, cfg *config.Config) {
				// Should have validation errors
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.json")
			err := os.WriteFile(configPath, []byte(tt.configContent), 0644)
			require.NoError(t, err)

			// Load configuration
			cfg, err := config.LoadFromFile(configPath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, cfg)
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, cfg)
			}
		})
	}
}

func TestConfiguration_EnvironmentVariableResolution(t *testing.T) {
	// Set environment variables
	os.Setenv("TEST_API_KEY", "sk-test-env-key")
	os.Setenv("TEST_BASE_URL", "https://api.test.com/v1")
	defer func() {
		os.Unsetenv("TEST_API_KEY")
		os.Unsetenv("TEST_BASE_URL")
	}()

	configContent := `{
		"$schema": "https://opencode.sh/schema.json",
		"username": "testuser",
		"provider": {
			"test": {
				"options": {
					"apiKey": "${TEST_API_KEY}",
					"baseURL": "${TEST_BASE_URL}"
				},
				"models": {
					"test-model": {
						"id": "test-model",
						"name": "Test Model",
						"displayName": "Test Model",
						"provider": {
							"id": "test",
							"npm": "@test/sdk"
						},
						"maxTokens": 4096,
						"supportsHTTP3": true
					}
				}
			}
		},
		"agent": {
			"name": "test-agent"
		},
		"mcp": {
			"servers": []
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify environment variables were resolved
	provider, exists := cfg.Providers["test"]
	assert.True(t, exists)
	assert.Equal(t, "sk-test-env-key", provider.Options["apiKey"])
	assert.Equal(t, "https://api.test.com/v1", provider.Options["baseURL"])
}

func TestConfiguration_DefaultValues(t *testing.T) {
	configContent := `{
		"$schema": "https://opencode.sh/schema.json",
		"username": "testuser",
		"provider": {
			"openai": {
				"options": {
					"apiKey": "sk-test-key",
					"baseURL": "https://api.openai.com/v1"
				},
				"models": {
					"gpt-4": {
						"id": "gpt-4",
						"name": "GPT-4",
						"displayName": "GPT-4",
						"provider": {
							"id": "openai",
							"npm": "@openai/sdk"
						},
						"maxTokens": 8192
					}
				}
			}
		},
		"agent": {
			"name": "test-agent"
		},
		"mcp": {
			"servers": []
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	// Verify default values are applied
	model := cfg.Providers["openai"].Models["gpt-4"]
	assert.Equal(t, "gpt-4", model.ID)
	assert.Equal(t, "GPT-4", model.Name)
	assert.Equal(t, "GPT-4", model.DisplayName)
	assert.Equal(t, 8192, model.MaxTokens)
	assert.False(t, model.SupportsHTTP3) // Default should be false
}

func TestConfiguration_Validation(t *testing.T) {
	tests := []struct {
		name          string
		config        map[string]interface{}
		expectValid   bool
		expectedError string
	}{
		{
			name: "Valid OpenCode config",
			config: map[string]interface{}{
				"$schema": "https://opencode.sh/schema.json",
				"username": "testuser",
				"provider": map[string]interface{}{
					"openai": map[string]interface{}{
						"options": map[string]interface{}{
							"apiKey": "sk-test-key",
							"baseURL": "https://api.openai.com/v1",
						},
						"models": map[string]interface{}{
							"gpt-4": map[string]interface{}{
								"id": "gpt-4",
								"name": "GPT-4",
								"displayName": "GPT-4",
								"provider": map[string]interface{}{
									"id": "openai",
									"npm": "@openai/sdk",
								},
								"maxTokens": 8192,
								"supportsHTTP3": true,
							},
						},
					},
				},
				"agent": map[string]interface{}{
					"name": "test-agent",
				},
				"mcp": map[string]interface{}{
					"servers": []interface{}{},
				},
			},
			expectValid: true,
		},
		{
			name: "Missing required schema",
			config: map[string]interface{}{
				"username": "testuser",
				"provider": map[string]interface{}{},
				"agent":    map[string]interface{}{},
				"mcp":      map[string]interface{}{},
			},
			expectValid:   false,
			expectedError: "missing $schema",
		},
		{
			name: "Invalid schema URL",
			config: map[string]interface{}{
				"$schema": "invalid-schema-url",
				"username": "testuser",
				"provider": map[string]interface{}{},
				"agent":    map[string]interface{}{},
				"mcp":      map[string]interface{}{},
			},
			expectValid:   false,
			expectedError: "invalid schema",
		},
		{
			name: "Empty provider section",
			config: map[string]interface{}{
				"$schema": "https://opencode.sh/schema.json",
				"username": "testuser",
				"provider": map[string]interface{}{},
				"agent":    map[string]interface{}{},
				"mcp":      map[string]interface{}{},
			},
			expectValid:   false,
			expectedError: "no providers configured",
		},
		{
			name: "Provider without API key",
			config: map[string]interface{}{
				"$schema": "https://opencode.sh/schema.json",
				"username": "testuser",
				"provider": map[string]interface{}{
					"openai": map[string]interface{}{
						"options": map[string]interface{}{
							"baseURL": "https://api.openai.com/v1",
						},
						"models": map[string]interface{}{},
					},
				},
				"agent": map[string]interface{}{},
				"mcp":   map[string]interface{}{},
			},
			expectValid:   false,
			expectedError: "missing API key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := config.NewValidator()
			err := validator.Validate(tt.config)

			if tt.expectValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			}
		})
	}
}

func TestConfiguration_SaveToFile(t *testing.T) {
	config := &config.Config{
		Schema:   "https://opencode.sh/schema.json",
		Username: "testuser",
		Providers: map[string]config.Provider{
			"openai": {
				Options: map[string]interface{}{
					"apiKey":  "sk-test-key",
					"baseURL": "https://api.openai.com/v1",
				},
				Models: map[string]config.Model{
					"gpt-4": {
						ID:          "gpt-4",
						Name:        "GPT-4",
						DisplayName: "GPT-4",
						Provider: config.ProviderInfo{
							ID:  "openai",
							NPM: "@openai/sdk",
						},
						MaxTokens:     8192,
						SupportsHTTP3: true,
					},
				},
			},
		},
		Agent: config.Agent{
			Name: "test-agent",
		},
		MCP: config.MCP{
			Servers: []interface{}{},
		},
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "saved_config.json")

	err := config.SaveToFile(config, configPath)
	require.NoError(t, err)

	// Verify file was created and can be loaded back
	loadedConfig, err := config.LoadFromFile(configPath)
	require.NoError(t, err)
	assert.NotNil(t, loadedConfig)
	assert.Equal(t, config.Schema, loadedConfig.Schema)
	assert.Equal(t, config.Username, loadedConfig.Username)
	assert.Len(t, loadedConfig.Providers, 1)
}

func TestConfiguration_MergeConfigurations(t *testing.T) {
	baseConfig := &config.Config{
		Schema:   "https://opencode.sh/schema.json",
		Username: "baseuser",
		Providers: map[string]config.Provider{
			"openai": {
				Options: map[string]interface{}{
					"apiKey":  "sk-base-key",
					"baseURL": "https://api.openai.com/v1",
				},
				Models: map[string]config.Model{
					"gpt-4": {
						ID:            "gpt-4",
						Name:          "GPT-4",
						DisplayName:   "GPT-4",
						MaxTokens:     8192,
						SupportsHTTP3: true,
					},
				},
			},
		},
	}

	overrideConfig := &config.Config{
		Username: "overrideuser",
		Providers: map[string]config.Provider{
			"openai": {
				Options: map[string]interface{}{
					"apiKey": "sk-override-key",
				},
				Models: map[string]config.Model{
					"gpt-4": {
						ID:          "gpt-4",
						Name:        "GPT-4-Override",
						DisplayName: "GPT-4 Override",
						MaxTokens:   16384,
					},
					"gpt-3.5-turbo": {
						ID:          "gpt-3.5-turbo",
						Name:        "GPT-3.5 Turbo",
						DisplayName: "GPT-3.5 Turbo",
						MaxTokens:   4096,
					},
				},
			},
		},
	}

	mergedConfig := config.Merge(baseConfig, overrideConfig)

	assert.Equal(t, "overrideuser", mergedConfig.Username)
	assert.Equal(t, "sk-override-key", mergedConfig.Providers["openai"].Options["apiKey"])
	assert.Equal(t, "GPT-4-Override", mergedConfig.Providers["openai"].Models["gpt-4"].Name)
	assert.Equal(t, 16384, mergedConfig.Providers["openai"].Models["gpt-4"].MaxTokens)
	assert.Contains(t, mergedConfig.Providers["openai"].Models, "gpt-3.5-turbo")
}

func TestConfiguration_Security(t *testing.T) {
	configContent := `{
		"$schema": "https://opencode.sh/schema.json",
		"username": "testuser",
		"provider": {
			"openai": {
				"options": {
					"apiKey": "sk-test-key-with-sensitive-data",
					"baseURL": "https://api.openai.com/v1"
				},
				"models": {
					"gpt-4": {
						"id": "gpt-4",
						"name": "GPT-4",
						"displayName": "GPT-4",
						"provider": {
							"id": "openai",
							"npm": "@openai/sdk"
						},
						"maxTokens": 8192,
						"supportsHTTP3": true
					}
				}
			}
		},
		"agent": {
			"name": "test-agent"
		},
		"mcp": {
			"servers": []
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "secure_config.json")
	err := os.WriteFile(configPath, []byte(configContent), 0600) // Restrictive permissions
	require.NoError(t, err)

	// Verify file permissions
	info, err := os.Stat(configPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	// Test secure loading
	cfg, err := config.LoadFromFileSecure(configPath)
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "sk-test-key-with-sensitive-data", cfg.Providers["openai"].Options["apiKey"])
}

func TestConfiguration_Serialization(t *testing.T) {
	originalConfig := &config.Config{
		Schema:   "https://opencode.sh/schema.json",
		Username: "serialization-test",
		Providers: map[string]config.Provider{
			"test-provider": {
				Options: map[string]interface{}{
					"apiKey":  "sk-test-serialization",
					"baseURL": "https://api.test.com/v1",
				},
				Models: map[string]config.Model{
					"test-model": {
						ID:            "test-model",
						Name:          "Test Model",
						DisplayName:   "Test Model (SC:8.5)",
						MaxTokens:     4096,
						SupportsHTTP3: true,
						Provider: config.ProviderInfo{
							ID:  "test-provider",
							NPM: "@test/sdk",
						},
					},
				},
			},
		},
		Agent: config.Agent{
			Name: "serialization-test-agent",
		},
		MCP: config.MCP{
			Servers: []interface{}{
				map[string]interface{}{
					"name": "test-server",
					"type": "test",
				},
			},
		},
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(originalConfig)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Deserialize back
	var deserializedConfig config.Config
	err = json.Unmarshal(jsonData, &deserializedConfig)
	require.NoError(t, err)

	// Verify serialization round-trip
	assert.Equal(t, originalConfig.Schema, deserializedConfig.Schema)
	assert.Equal(t, originalConfig.Username, deserializedConfig.Username)
	assert.Equal(t, originalConfig.Providers["test-provider"].Options["apiKey"], 
		deserializedConfig.Providers["test-provider"].Options["apiKey"])
	assert.Equal(t, originalConfig.Providers["test-provider"].Models["test-model"].MaxTokens,
		deserializedConfig.Providers["test-provider"].Models["test-model"].MaxTokens)
}

func TestConfiguration_ComplexScenarios(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
		test   func(t *testing.T, cfg *config.Config)
	}{
		{
			name: "Large configuration with many providers",
			config: &config.Config{
				Schema:   "https://opencode.sh/schema.json",
				Username: "large-config-test",
				Providers: generateLargeProviderSet(),
				Agent:    config.Agent{Name: "large-test-agent"},
				MCP:      config.MCP{Servers: []interface{}{}},
			},
			test: func(t *testing.T, cfg *config.Config) {
				assert.Len(t, cfg.Providers, 50) // 50 providers
				totalModels := 0
				for _, provider := range cfg.Providers {
					totalModels += len(provider.Models)
				}
				assert.Greater(t, totalModels, 100) // Should have many models
			},
		},
		{
			name: "Configuration with special characters",
			config: &config.Config{
				Schema:   "https://opencode.sh/schema.json",
				Username: "user_with-special.chars",
				Providers: map[string]config.Provider{
					"test-provider": {
						Options: map[string]interface{}{
							"apiKey":  "sk-test-key-with-special-chars!@#$%^&*()",
							"baseURL": "https://api.test.com/v1/with/path?param=value&other=test",
						},
						Models: map[string]config.Model{
							"test-model": {
								ID:          "test-model",
								Name:        "Test Model (SC:8.5) (brotli) (http3)",
								DisplayName: "Test Model (SC:8.5) (brotli) (http3) (free to use)",
								MaxTokens:   4096,
							},
						},
					},
				},
				Agent: config.Agent{Name: "special-chars-agent"},
				MCP:   config.MCP{Servers: []interface{}{}},
			},
			test: func(t *testing.T, cfg *config.Config) {
				assert.Contains(t, cfg.Providers["test-provider"].Options["apiKey"], "!")
				assert.Contains(t, cfg.Providers["test-provider"].Models["test-model"].Name, "(brotli)")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "complex_config.json")
			
			err := config.SaveToFile(tt.config, configPath)
			require.NoError(t, err)

			loadedConfig, err := config.LoadFromFile(configPath)
			require.NoError(t, err)

			tt.test(t, loadedConfig)
		})
	}
}

// Helper function to generate a large provider set for testing
func generateLargeProviderSet() map[string]config.Provider {
	providers := make(map[string]config.Provider)
	
	for i := 0; i < 50; i++ {
		providerName := fmt.Sprintf("provider-%d", i)
		models := make(map[string]config.Model)
		
		// Add 2-5 models per provider
		for j := 0; j < 2+(i%4); j++ {
			modelID := fmt.Sprintf("model-%d-%d", i, j)
			models[modelID] = config.Model{
				ID:          modelID,
				Name:        fmt.Sprintf("Model %d-%d", i, j),
				DisplayName: fmt.Sprintf("Model %d-%d (SC:%.1f)", i, j, 8.0+float64(j)*0.2),
				MaxTokens:   1000 + j*1000,
				Provider: config.ProviderInfo{
					ID:  providerName,
					NPM: fmt.Sprintf("@%s/sdk", providerName),
				},
			}
		}
		
		providers[providerName] = config.Provider{
			Options: map[string]interface{}{
				"apiKey":  fmt.Sprintf("sk-test-key-%d", i),
				"baseURL": fmt.Sprintf("https://api.%s.com/v1", providerName),
			},
			Models: models,
		}
	}
	
	return providers
}