package providers

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVerifiedConfigGenerator(t *testing.T) {
	logger := NewTestLogger()
	config := CreateDefaultVerificationConfig()
	emps := NewEnhancedModelProviderService("", logger, config)

	vcg := NewVerifiedConfigGenerator(emps, logger, "/tmp/test-output")

	require.NotNil(t, vcg)
	assert.Equal(t, "/tmp/test-output", vcg.outputDir)
	assert.Equal(t, emps, vcg.enhancedService)
	assert.Equal(t, logger, vcg.logger)
}

func TestVerifiedConfigGenerator_CreateRedactedConfig(t *testing.T) {
	logger := NewTestLogger()
	config := CreateDefaultVerificationConfig()
	emps := NewEnhancedModelProviderService("", logger, config)
	vcg := NewVerifiedConfigGenerator(emps, logger, "/tmp/test-output")

	originalConfig := &VerifiedConfig{
		GeneratedAt:         time.Now(),
		VerificationEnabled: true,
		StrictMode:          true,
		TotalModels:         10,
		VerifiedModels:      8,
		Providers: map[string]VerifiedProviderConfig{
			"openai": {
				ProviderID:   "openai",
				ProviderName: "OpenAI",
				BaseURL:      "https://api.openai.com/v1",
				ModelCount:   5,
				VerifiedModels: []VerifiedModelConfig{
					{ModelID: "gpt-4", ModelName: "GPT-4"},
				},
			},
			"anthropic": {
				ProviderID:   "anthropic",
				ProviderName: "Anthropic",
				BaseURL:      "https://api.anthropic.com/v1",
				ModelCount:   5,
				VerifiedModels: []VerifiedModelConfig{
					{ModelID: "claude-3", ModelName: "Claude 3"},
				},
			},
		},
	}

	redacted := vcg.createRedactedConfig(originalConfig)

	require.NotNil(t, redacted)
	assert.Equal(t, originalConfig.TotalModels, redacted.TotalModels)
	assert.Equal(t, originalConfig.VerifiedModels, redacted.VerifiedModels)
	assert.Equal(t, originalConfig.VerificationEnabled, redacted.VerificationEnabled)
	assert.Equal(t, originalConfig.StrictMode, redacted.StrictMode)

	// Check that BaseURLs are redacted
	for _, provider := range redacted.Providers {
		assert.Equal(t, "REDACTED", provider.BaseURL)
	}

	// Verify original is not modified
	assert.Equal(t, "https://api.openai.com/v1", originalConfig.Providers["openai"].BaseURL)
}

func TestVerifiedConfigGenerator_SaveConfigToFile(t *testing.T) {
	logger := NewTestLogger()
	config := CreateDefaultVerificationConfig()
	emps := NewEnhancedModelProviderService("", logger, config)

	tmpDir, err := os.MkdirTemp("", "vcg-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	vcg := NewVerifiedConfigGenerator(emps, logger, tmpDir)

	testConfig := &VerifiedConfig{
		GeneratedAt:         time.Now(),
		VerificationEnabled: true,
		TotalModels:         5,
		VerifiedModels:      3,
	}

	filePath := filepath.Join(tmpDir, "test_config.json")
	err = vcg.saveConfigToFile(testConfig, filePath)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(filePath)
	assert.NoError(t, err)

	// Verify content
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "\"total_models\": 5")
	assert.Contains(t, string(content), "\"verified_models\": 3")
}

func TestVerifiedConfigGenerator_SaveVerificationSummary(t *testing.T) {
	logger := NewTestLogger()
	config := CreateDefaultVerificationConfig()
	emps := NewEnhancedModelProviderService("", logger, config)

	tmpDir, err := os.MkdirTemp("", "vcg-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	vcg := NewVerifiedConfigGenerator(emps, logger, tmpDir)

	testConfig := &VerifiedConfig{
		GeneratedAt:         time.Now(),
		VerificationEnabled: true,
		StrictMode:          false,
		TotalModels:         10,
		VerifiedModels:      7,
		Providers: map[string]VerifiedProviderConfig{
			"openai": {
				ProviderID:   "openai",
				ModelCount:   6,
				VerifiedModels: []VerifiedModelConfig{
					{ModelID: "gpt-4"},
					{ModelID: "gpt-4-turbo"},
					{ModelID: "gpt-3.5"},
				},
			},
			"anthropic": {
				ProviderID:   "anthropic",
				ModelCount:   4,
				VerifiedModels: []VerifiedModelConfig{
					{ModelID: "claude-3-opus"},
					{ModelID: "claude-3-sonnet"},
				},
			},
		},
	}

	summaryPath := filepath.Join(tmpDir, "summary.json")
	err = vcg.saveVerificationSummary(testConfig, summaryPath)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(summaryPath)
	assert.NoError(t, err)

	// Verify content
	content, err := os.ReadFile(summaryPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "\"total_models\": 10")
	assert.Contains(t, string(content), "\"verified_models\": 7")
	assert.Contains(t, string(content), "\"verification_rate\":")
	assert.Contains(t, string(content), "\"provider_count\": 2")
}

func TestVerifiedConfig_Structure(t *testing.T) {
	vc := VerifiedConfig{
		GeneratedAt:         time.Now(),
		VerificationEnabled: true,
		StrictMode:          true,
		TotalModels:         20,
		VerifiedModels:      15,
		Providers:           make(map[string]VerifiedProviderConfig),
	}

	assert.True(t, vc.VerificationEnabled)
	assert.True(t, vc.StrictMode)
	assert.Equal(t, 20, vc.TotalModels)
	assert.Equal(t, 15, vc.VerifiedModels)
	assert.NotNil(t, vc.Providers)
}

func TestVerifiedProviderConfig_Structure(t *testing.T) {
	vpc := VerifiedProviderConfig{
		ProviderID:   "openai",
		ProviderName: "OpenAI",
		BaseURL:      "https://api.openai.com/v1",
		ModelCount:   10,
		VerifiedModels: []VerifiedModelConfig{
			{ModelID: "gpt-4", ModelName: "GPT-4"},
		},
	}

	assert.Equal(t, "openai", vpc.ProviderID)
	assert.Equal(t, "OpenAI", vpc.ProviderName)
	assert.Equal(t, "https://api.openai.com/v1", vpc.BaseURL)
	assert.Equal(t, 10, vpc.ModelCount)
	assert.Len(t, vpc.VerifiedModels, 1)
}

func TestVerifiedModelConfig_Structure(t *testing.T) {
	now := time.Now()
	vmc := VerifiedModelConfig{
		ModelID:             "gpt-4",
		ModelName:           "GPT-4",
		DisplayName:         "GPT-4 (Latest)",
		Features:            map[string]interface{}{"streaming": true},
		MaxTokens:           8192,
		CostPer1MInput:      30.0,
		CostPer1MOutput:     60.0,
		VerificationScore:   0.95,
		CanSeeCode:          true,
		AffirmativeResponse: true,
		LastVerifiedAt:      now,
	}

	assert.Equal(t, "gpt-4", vmc.ModelID)
	assert.Equal(t, "GPT-4", vmc.ModelName)
	assert.Equal(t, "GPT-4 (Latest)", vmc.DisplayName)
	assert.Equal(t, 8192, vmc.MaxTokens)
	assert.Equal(t, 30.0, vmc.CostPer1MInput)
	assert.Equal(t, 60.0, vmc.CostPer1MOutput)
	assert.Equal(t, 0.95, vmc.VerificationScore)
	assert.True(t, vmc.CanSeeCode)
	assert.True(t, vmc.AffirmativeResponse)
	assert.Equal(t, now, vmc.LastVerifiedAt)
	assert.True(t, vmc.Features["streaming"].(bool))
}

func TestVerifiedConfigGenerator_SaveVerifiedConfig(t *testing.T) {
	logger := NewTestLogger()
	config := CreateDefaultVerificationConfig()
	emps := NewEnhancedModelProviderService("", logger, config)

	tmpDir, err := os.MkdirTemp("", "vcg-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	vcg := NewVerifiedConfigGenerator(emps, logger, tmpDir)

	testConfig := &VerifiedConfig{
		GeneratedAt:         time.Now(),
		VerificationEnabled: true,
		StrictMode:          true,
		TotalModels:         10,
		VerifiedModels:      8,
		Providers: map[string]VerifiedProviderConfig{
			"openai": {
				ProviderID:   "openai",
				ProviderName: "OpenAI",
				BaseURL:      "https://api.openai.com/v1",
				ModelCount:   10,
				VerifiedModels: []VerifiedModelConfig{
					{ModelID: "gpt-4", ModelName: "GPT-4"},
				},
			},
		},
	}

	err = vcg.SaveVerifiedConfig(testConfig, "test")
	require.NoError(t, err)

	// Verify all three files were created
	_, err = os.Stat(filepath.Join(tmpDir, "test_verified_config.json"))
	assert.NoError(t, err)

	_, err = os.Stat(filepath.Join(tmpDir, "test_verified_config_redacted.json"))
	assert.NoError(t, err)

	_, err = os.Stat(filepath.Join(tmpDir, "test_verification_summary.json"))
	assert.NoError(t, err)
}

func TestVerifiedConfigGenerator_SaveVerifiedConfig_CreatesDirectory(t *testing.T) {
	logger := NewTestLogger()
	config := CreateDefaultVerificationConfig()
	emps := NewEnhancedModelProviderService("", logger, config)

	tmpBase, err := os.MkdirTemp("", "vcg-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpBase)

	newDir := filepath.Join(tmpBase, "new", "nested", "dir")
	vcg := NewVerifiedConfigGenerator(emps, logger, newDir)

	testConfig := &VerifiedConfig{
		GeneratedAt:         time.Now(),
		VerificationEnabled: true,
		TotalModels:         5,
		VerifiedModels:      3,
		Providers:           make(map[string]VerifiedProviderConfig),
	}

	err = vcg.SaveVerifiedConfig(testConfig, "test")
	require.NoError(t, err)

	// Verify directory was created
	info, err := os.Stat(newDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}
