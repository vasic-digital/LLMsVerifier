package opencode_verifier

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"llm-verifier/pkg/opencode/config"
	"llm-verifier/database"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOpenCodeVerifier(t *testing.T) {
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	verifier := NewOpenCodeVerifier(db, "test.json")
	assert.NotNil(t, verifier)
	assert.NotNil(t, verifier.validator)
	assert.Equal(t, "test.json", verifier.configPath)
}

func TestVerifyConfiguration(t *testing.T) {
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	tests := []struct {
		name        string
		config      map[string]interface{}
		expectValid bool
		expectError bool
	}{
		{
			name: "valid minimal config",
			config: map[string]interface{}{
				"provider": map[string]interface{}{
					"openai": map[string]interface{}{
						"options": map[string]interface{}{
							"api_key": "test-key",
						},
					},
				},
			},
			expectValid: true,
			expectError: false,
		},
		{
			name: "config with agent",
			config: map[string]interface{}{
				"provider": map[string]interface{}{
					"openai": map[string]interface{}{},
				},
				"agent": map[string]interface{}{
					"build": map[string]interface{}{
						"model": "gpt-4",
						"prompt": "You are a build agent",
					},
				},
			},
			expectValid: true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "opencode.json")

			configJSON, err := json.Marshal(tt.config)
			require.NoError(t, err)

			err = os.WriteFile(configPath, configJSON, 0644)
			require.NoError(t, err)

			// Create verifier and verify
			verifier := NewOpenCodeVerifier(db, configPath)
			result, err := verifier.VerifyConfiguration()
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectValid, result.Valid)
			}
		})
	}
}

func TestVerifyProvider(t *testing.T) {
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	verifier := NewOpenCodeVerifier(db, "test.json")

	tests := []struct {
		name     string
		provider config.ProviderConfig
		expected ProviderVerificationStatus
	}{
		{
			name: "provider with options",
			provider: config.ProviderConfig{
				Options: map[string]interface{}{
					"api_key": "test-key",
				},
			},
			expected: ProviderVerificationStatus{
				Configured: true,
				Score:      60.0,
			},
		},
		{
			name: "provider with model",
			provider: config.ProviderConfig{
				Model: "gpt-4",
			},
			expected: ProviderVerificationStatus{
				Configured: true,
				Score:      60.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := verifier.verifyProvider("test", &tt.provider)
			assert.Equal(t, tt.expected.Configured, status.Configured)
			assert.Equal(t, tt.expected.Score, status.Score)
		})
	}
}

func TestVerifyAgent(t *testing.T) {
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	verifier := NewOpenCodeVerifier(db, "test.json")

	tests := []struct {
		name   string
		agent  config.AgentConfig
		expected AgentVerificationStatus
	}{
		{
			name: "agent with model",
			agent: config.AgentConfig{
				Model: "gpt-4",
			},
			expected: AgentVerificationStatus{
				HasModel: true,
				Score:    70.0,
			},
		},
		{
			name: "agent with prompt",
			agent: config.AgentConfig{
				Prompt: "You are a helpful assistant",
			},
			expected: AgentVerificationStatus{
				HasPrompt: true,
				Score:     70.0,
			},
		},
		{
			name: "agent with tools",
			agent: config.AgentConfig{
				Tools: map[string]bool{
					"github": true,
					"docker": true,
				},
			},
			expected: AgentVerificationStatus{
				ToolsConfigured: 2,
				Score:           54.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := verifier.verifyAgent("test", &tt.agent)
			assert.Equal(t, tt.expected.HasModel, status.HasModel)
			assert.Equal(t, tt.expected.HasPrompt, status.HasPrompt)
			assert.Equal(t, tt.expected.ToolsConfigured, status.ToolsConfigured)
			assert.Equal(t, tt.expected.Score, status.Score)
		})
	}
}

func TestVerifyMcp(t *testing.T) {
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	verifier := NewOpenCodeVerifier(db, "test.json")

	trueVal := true
	falseVal := false
	
	tests := []struct {
		name   string
		mcp    config.McpConfig
		expected McpVerificationStatus
	}{
		{
			name: "local MCP server",
			mcp: config.McpConfig{
				Type:    "local",
				Command: []string{"npx", "@modelcontextprotocol/server-github"},
				Enabled: &trueVal,
			},
			expected: McpVerificationStatus{
				Type:    "local",
				Enabled: true,
				Score:   70.0,
			},
		},
		{
			name: "remote MCP server",
			mcp: config.McpConfig{
				Type:    "remote",
				URL:     "https://api.github.com/mcp",
				Enabled: &falseVal,
			},
			expected: McpVerificationStatus{
				Type:    "remote",
				Enabled: false,
				Score:   50.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := verifier.verifyMcp("test", &tt.mcp)
			assert.Equal(t, tt.expected.Type, status.Type)
			assert.Equal(t, tt.expected.Enabled, status.Enabled)
			assert.Equal(t, tt.expected.Score, status.Score)
		})
	}
}

func TestGetVerificationStatus(t *testing.T) {
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	status, err := GetVerificationStatus(nil)
	require.NoError(t, err)
	assert.NotNil(t, status)
	assert.Contains(t, status, "total_configs")
	assert.Contains(t, status, "valid_configs")
	assert.Contains(t, status, "average_score")
}

func TestCalculateOverallScore(t *testing.T) {
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	verifier := NewOpenCodeVerifier(db, "test.json")

	tests := []struct {
		name     string
		result   *VerificationResult
		expected float64
	}{
		{
			name: "empty result",
			result: &VerificationResult{
				ProviderStatus: map[string]ProviderVerificationStatus{},
				AgentStatus:    map[string]AgentVerificationStatus{},
				McpStatus:      map[string]McpVerificationStatus{},
			},
			expected: 0.0,
		},
		{
			name: "with providers",
			result: &VerificationResult{
				ProviderStatus: map[string]ProviderVerificationStatus{
					"openai": {
						Score: 80.0,
					},
				},
			},
			expected: 80.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := verifier.calculateOverallScore(tt.result)
			assert.Equal(t, tt.expected, score)
		})
	}
}

func TestVerifyAllConfigurations(t *testing.T) {
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	tmpDir := t.TempDir()

	// Create a test .opencode directory
	opencodeDir := filepath.Join(tmpDir, ".opencode")
	err = os.MkdirAll(opencodeDir, 0755)
	require.NoError(t, err)

	// Create test config
	configPath := filepath.Join(opencodeDir, "opencode.json")
	testConfig := map[string]interface{}{
		"provider": map[string]interface{}{
			"openai": map[string]interface{}{
				"model": "gpt-4",
			},
		},
	}

	configJSON, err := json.Marshal(testConfig)
	require.NoError(t, err)

	err = os.WriteFile(configPath, configJSON, 0644)
	require.NoError(t, err)

	// Test verification
	err = VerifyAllConfigurations(db, tmpDir)
	assert.NoError(t, err)
}