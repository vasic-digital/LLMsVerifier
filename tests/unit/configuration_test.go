package unit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	opencodeConfig "llm-verifier/pkg/opencode/config"
)

func TestConfiguration_LoadFromFile(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		expectError   bool
		validateFunc  func(t *testing.T, cfg *opencodeConfig.Config)
	}{
		{
			name: "Valid OpenCode configuration",
			configContent: `{
				"username": "testuser",
				"provider": {
					"openai": {
						"options": {
							"apiKey": "sk-test-key",
							"baseURL": "https://api.openai.com/v1"
						},
						"model": "gpt-4"
					}
				},
				"agent": {
					"default": {
						"model": "gpt-4",
						"prompt": "You are a helpful assistant"
					}
				},
				"mcp": {}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, cfg *opencodeConfig.Config) {
				assert.NotNil(t, cfg)
				assert.Equal(t, "testuser", cfg.Username)
				assert.Contains(t, cfg.Provider, "openai")
				assert.Equal(t, "gpt-4", cfg.Provider["openai"].Model)
			},
		},
		{
			name: "Configuration with MCP servers",
			configContent: `{
				"username": "testuser",
				"provider": {
					"anthropic": {
						"options": {
							"apiKey": "sk-ant-test-key"
						},
						"model": "claude-3-opus"
					}
				},
				"mcp": {
					"test-server": {
						"type": "stdio",
						"command": ["npx", "test-server"],
						"enabled": true
					}
				}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, cfg *opencodeConfig.Config) {
				assert.NotNil(t, cfg)
				assert.Contains(t, cfg.Mcp, "test-server")
				assert.Equal(t, "stdio", cfg.Mcp["test-server"].Type)
			},
		},
		{
			name: "Invalid JSON",
			configContent: `{
				"invalid": json content
			}`,
			expectError: true,
			validateFunc: func(t *testing.T, cfg *opencodeConfig.Config) {
				assert.Nil(t, cfg)
			},
		},
		{
			name: "Empty configuration",
			configContent: `{}`,
			expectError:   false,
			validateFunc: func(t *testing.T, cfg *opencodeConfig.Config) {
				assert.NotNil(t, cfg)
				assert.Empty(t, cfg.Username)
			},
		},
	}

	loader := &opencodeConfig.ConfigLoader{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "opencode.json")
			err := os.WriteFile(configPath, []byte(tt.configContent), 0644)
			require.NoError(t, err)

			// Load configuration
			cfg, err := loader.LoadFromFile(configPath)

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

func TestConfiguration_SaveToFile(t *testing.T) {
	config := &opencodeConfig.Config{
		Username: "testuser",
		Provider: map[string]opencodeConfig.ProviderConfig{
			"openai": {
				Options: map[string]interface{}{
					"apiKey":  "sk-test-key",
					"baseURL": "https://api.openai.com/v1",
				},
				Model: "gpt-4",
			},
		},
		Agent: map[string]opencodeConfig.AgentConfig{
			"default": {
				Model:  "gpt-4",
				Prompt: "You are a helpful assistant",
			},
		},
		Mcp: map[string]opencodeConfig.McpConfig{},
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "saved_config.json")

	loader := &opencodeConfig.ConfigLoader{}
	err := loader.SaveToFile(config, configPath)
	require.NoError(t, err)

	// Verify file was created and can be loaded back
	loadedConfig, err := loader.LoadFromFile(configPath)
	require.NoError(t, err)
	assert.NotNil(t, loadedConfig)
	assert.Equal(t, "testuser", loadedConfig.Username)
	assert.Len(t, loadedConfig.Provider, 1)
	assert.Equal(t, "gpt-4", loadedConfig.Provider["openai"].Model)
}

func TestConfiguration_Serialization(t *testing.T) {
	originalConfig := &opencodeConfig.Config{
		Username: "serialization-test",
		Provider: map[string]opencodeConfig.ProviderConfig{
			"test-provider": {
				Options: map[string]interface{}{
					"apiKey":  "sk-test-serialization",
					"baseURL": "https://api.test.com/v1",
				},
				Model: "test-model",
			},
		},
		Agent: map[string]opencodeConfig.AgentConfig{
			"test-agent": {
				Model:       "test-model",
				Prompt:      "Test prompt",
				Description: "Test agent description",
			},
		},
		Mcp: map[string]opencodeConfig.McpConfig{
			"test-server": {
				Type:    "stdio",
				Command: []string{"npx", "test-server"},
			},
		},
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(originalConfig)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Deserialize back
	var deserializedConfig opencodeConfig.Config
	err = json.Unmarshal(jsonData, &deserializedConfig)
	require.NoError(t, err)

	// Verify serialization round-trip
	assert.Equal(t, originalConfig.Username, deserializedConfig.Username)
	assert.Equal(t, originalConfig.Provider["test-provider"].Options["apiKey"],
		deserializedConfig.Provider["test-provider"].Options["apiKey"])
	assert.Equal(t, originalConfig.Provider["test-provider"].Model,
		deserializedConfig.Provider["test-provider"].Model)
}

func TestConfiguration_ProviderOptions(t *testing.T) {
	tests := []struct {
		name     string
		provider opencodeConfig.ProviderConfig
		validate func(t *testing.T, p opencodeConfig.ProviderConfig)
	}{
		{
			name: "Provider with API key",
			provider: opencodeConfig.ProviderConfig{
				Options: map[string]interface{}{
					"apiKey": "sk-test-key",
				},
				Model: "gpt-4",
			},
			validate: func(t *testing.T, p opencodeConfig.ProviderConfig) {
				assert.Equal(t, "sk-test-key", p.Options["apiKey"])
				assert.Equal(t, "gpt-4", p.Model)
			},
		},
		{
			name: "Provider with custom base URL",
			provider: opencodeConfig.ProviderConfig{
				Options: map[string]interface{}{
					"apiKey":  "sk-test-key",
					"baseURL": "https://custom.api.com/v1",
				},
				Model: "custom-model",
			},
			validate: func(t *testing.T, p opencodeConfig.ProviderConfig) {
				assert.Equal(t, "https://custom.api.com/v1", p.Options["baseURL"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.provider)
		})
	}
}

func TestConfiguration_AgentConfig(t *testing.T) {
	temp := 0.7
	topP := 0.9
	maxSteps := 10
	disable := false

	agent := opencodeConfig.AgentConfig{
		Model:       "gpt-4",
		Temperature: &temp,
		TopP:        &topP,
		Prompt:      "You are a coding assistant",
		Description: "Default coding agent",
		MaxSteps:    &maxSteps,
		Disable:     &disable,
		Tools: map[string]bool{
			"bash":   true,
			"edit":   true,
			"read":   true,
			"search": true,
		},
	}

	assert.Equal(t, "gpt-4", agent.Model)
	assert.Equal(t, 0.7, *agent.Temperature)
	assert.Equal(t, 0.9, *agent.TopP)
	assert.Equal(t, 10, *agent.MaxSteps)
	assert.False(t, *agent.Disable)
	assert.True(t, agent.Tools["bash"])
}

func TestConfiguration_McpConfig(t *testing.T) {
	enabled := true
	timeout := 30

	mcp := opencodeConfig.McpConfig{
		Type:    "stdio",
		Command: []string{"npx", "test-server"},
		Environment: map[string]string{
			"TEST_VAR": "test_value",
		},
		Enabled: &enabled,
		Timeout: &timeout,
	}

	assert.Equal(t, "stdio", mcp.Type)
	assert.Len(t, mcp.Command, 2)
	assert.Equal(t, "test_value", mcp.Environment["TEST_VAR"])
	assert.True(t, *mcp.Enabled)
	assert.Equal(t, 30, *mcp.Timeout)
}

func TestConfiguration_SSEMcp(t *testing.T) {
	mcp := opencodeConfig.McpConfig{
		Type: "sse",
		URL:  "http://localhost:8080/sse",
		Headers: map[string]string{
			"Authorization": "Bearer test-token",
		},
	}

	assert.Equal(t, "sse", mcp.Type)
	assert.Equal(t, "http://localhost:8080/sse", mcp.URL)
	assert.Equal(t, "Bearer test-token", mcp.Headers["Authorization"])
}

func TestConfiguration_KeybindsConfig(t *testing.T) {
	keybinds := opencodeConfig.KeybindsConfig{
		Leader:          "space",
		AppExit:         "ctrl+q",
		EditorOpen:      "ctrl+e",
		SessionNew:      "ctrl+n",
		InputSubmit:     "enter",
		InputClear:      "ctrl+u",
		HistoryPrevious: "up",
		HistoryNext:     "down",
	}

	assert.Equal(t, "space", keybinds.Leader)
	assert.Equal(t, "ctrl+q", keybinds.AppExit)
	assert.Equal(t, "enter", keybinds.InputSubmit)
}

func TestConfiguration_PermissionConfig(t *testing.T) {
	perm := opencodeConfig.PermissionConfig{
		Edit:              "ask",
		Bash:              "allow",
		Webfetch:          "allow",
		ExternalDirectory: "deny",
	}

	assert.Equal(t, "ask", perm.Edit)
	assert.Equal(t, "allow", perm.Bash)
	assert.Equal(t, "deny", perm.ExternalDirectory)
}

func TestConfiguration_CompactionConfig(t *testing.T) {
	auto := true
	prune := false

	compaction := opencodeConfig.CompactionConfig{
		Auto:  &auto,
		Prune: &prune,
	}

	assert.True(t, *compaction.Auto)
	assert.False(t, *compaction.Prune)
}

func TestConfiguration_FileNotFound(t *testing.T) {
	loader := &opencodeConfig.ConfigLoader{}
	_, err := loader.LoadFromFile("/nonexistent/path/config.json")
	assert.Error(t, err)
}

func TestConfiguration_CompleteConfig(t *testing.T) {
	temp := 0.7
	enabled := true

	config := &opencodeConfig.Config{
		Username:     "complete-test-user",
		Instructions: []string{"Be helpful", "Write clean code"},
		Provider: map[string]opencodeConfig.ProviderConfig{
			"openai": {
				Options: map[string]interface{}{
					"apiKey": "sk-test",
				},
				Model: "gpt-4",
			},
			"anthropic": {
				Options: map[string]interface{}{
					"apiKey": "sk-ant-test",
				},
				Model: "claude-3-opus",
			},
		},
		Agent: map[string]opencodeConfig.AgentConfig{
			"default": {
				Model:       "gpt-4",
				Temperature: &temp,
				Prompt:      "You are a helpful assistant",
			},
		},
		Mcp: map[string]opencodeConfig.McpConfig{
			"test-server": {
				Type:    "stdio",
				Command: []string{"npx", "test"},
				Enabled: &enabled,
			},
		},
		Command: map[string]opencodeConfig.CommandConfig{
			"test": {
				Template:    "echo test",
				Description: "Test command",
			},
		},
	}

	// Verify complete config structure
	assert.Equal(t, "complete-test-user", config.Username)
	assert.Len(t, config.Instructions, 2)
	assert.Len(t, config.Provider, 2)
	assert.Contains(t, config.Provider, "openai")
	assert.Contains(t, config.Provider, "anthropic")
	assert.Len(t, config.Agent, 1)
	assert.Len(t, config.Mcp, 1)
	assert.Len(t, config.Command, 1)

	// Test JSON round-trip
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "complete_config.json")

	loader := &opencodeConfig.ConfigLoader{}
	err := loader.SaveToFile(config, configPath)
	require.NoError(t, err)

	loadedConfig, err := loader.LoadFromFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, config.Username, loadedConfig.Username)
	assert.Len(t, loadedConfig.Provider, 2)
}
