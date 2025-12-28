package crush_verifier

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"llm-verifier/database"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCrushVerifier(t *testing.T) {
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	verifier := NewCrushVerifier(db, "test.json")
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
				"providers": map[string]interface{}{
					"openai": map[string]interface{}{
						"name":     "openai",
						"type":     "openai",
						"base_url": "https://api.openai.com/v1",
						"api_key":  "sk-test-key",
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
								"supports_brotli":     true,
							},
						},
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
			configPath := filepath.Join(tmpDir, "crush.json")

			configJSON, err := json.Marshal(tt.config)
			require.NoError(t, err)

			err = os.WriteFile(configPath, configJSON, 0644)
			require.NoError(t, err)

			// Create verifier and verify
			verifier := NewCrushVerifier(db, configPath)
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

func TestVerifyAllConfigurations(t *testing.T) {
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	tmpDir := t.TempDir()

	// Create a test crush.json
	configPath := filepath.Join(tmpDir, "crush.json")
	testConfig := map[string]interface{}{
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
	}

	configJSON, err := json.Marshal(testConfig)
	require.NoError(t, err)

	err = os.WriteFile(configPath, configJSON, 0644)
	require.NoError(t, err)

	// Test verification
	err = VerifyAllConfigurations(db, tmpDir)
	assert.NoError(t, err)
}

func TestCalculateOverallScore(t *testing.T) {
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	verifier := NewCrushVerifier(db, "test.json")

	tests := []struct {
		name     string
		result   *VerificationResult
		expected float64
	}{
		{
			name: "empty result",
			result: &VerificationResult{
				ProviderStatus: map[string]ProviderVerificationStatus{},
				ModelStatus:    map[string]map[string]ModelVerificationStatus{},
				LspStatus:      map[string]LspVerificationStatus{},
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
			expected: 85.0, // 80 + 5 bonus for no errors/warnings
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := verifier.calculateOverallScore(tt.result)
			if tt.expected == 0.0 {
				assert.Equal(t, tt.expected, score)
			} else {
				assert.Equal(t, tt.expected, score)
			}
		})
	}
}