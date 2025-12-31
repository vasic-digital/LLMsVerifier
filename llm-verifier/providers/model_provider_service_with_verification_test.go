package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDefaultVerificationConfig(t *testing.T) {
	config := CreateDefaultVerificationConfig()

	assert.True(t, config.Enabled)
	assert.True(t, config.StrictMode)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 30, config.TimeoutSeconds)
	assert.True(t, config.RequireAffirmative)
	assert.Equal(t, 0.7, config.MinVerificationScore)
}

func TestEnhancedModelProviderService_GetVerificationResults(t *testing.T) {
	logger := NewTestLogger()
	config := CreateDefaultVerificationConfig()

	emps := NewEnhancedModelProviderService("", logger, config)

	results := emps.GetVerificationResults()
	assert.NotNil(t, results)
	assert.Empty(t, results)
}

func TestEnhancedModelProviderService_GetModelVerificationResult(t *testing.T) {
	logger := NewTestLogger()
	config := CreateDefaultVerificationConfig()

	emps := NewEnhancedModelProviderService("", logger, config)

	result := emps.GetModelVerificationResult("openai", "gpt-4")
	assert.Nil(t, result)
}

func TestEnhancedModelProviderService_IsModelVerified(t *testing.T) {
	logger := NewTestLogger()
	config := CreateDefaultVerificationConfig()

	emps := NewEnhancedModelProviderService("", logger, config)

	verified := emps.IsModelVerified("openai", "gpt-4")
	assert.False(t, verified)
}

func TestEnhancedModelProviderService_EnableVerification(t *testing.T) {
	logger := NewTestLogger()
	config := CreateDefaultVerificationConfig()

	emps := NewEnhancedModelProviderService("", logger, config)

	// Should not panic
	emps.EnableVerification(true)
	emps.EnableVerification(false)
}

func TestEnhancedModelProviderService_SetStrictMode(t *testing.T) {
	logger := NewTestLogger()
	config := CreateDefaultVerificationConfig()

	emps := NewEnhancedModelProviderService("", logger, config)

	// Should not panic
	emps.SetStrictMode(true)
	emps.SetStrictMode(false)
}

func TestEnhancedModelProviderService_ClearVerificationResults(t *testing.T) {
	logger := NewTestLogger()
	config := CreateDefaultVerificationConfig()

	emps := NewEnhancedModelProviderService("", logger, config)

	// Should not panic
	emps.ClearVerificationResults()

	results := emps.GetVerificationResults()
	assert.Empty(t, results)
}

func TestEnhancedModelProviderService_GetVerificationService(t *testing.T) {
	logger := NewTestLogger()
	config := CreateDefaultVerificationConfig()

	emps := NewEnhancedModelProviderService("", logger, config)

	vs := emps.GetVerificationService()
	require.NotNil(t, vs)
}

func TestEnhancedModelProviderService_FilterVerifiedModels_VerificationDisabled(t *testing.T) {
	logger := NewTestLogger()
	config := VerificationConfig{
		Enabled: false,
	}

	emps := NewEnhancedModelProviderService("", logger, config)

	models := []Model{
		{ID: "model1", Name: "Model 1", ProviderID: "openai"},
		{ID: "model2", Name: "Model 2", ProviderID: "openai"},
	}

	results := make(map[string]*ModelVerificationResult)

	// When verification is disabled, all models should be returned
	filtered := emps.filterVerifiedModels(models, results)
	assert.Len(t, filtered, 2)
}

func TestEnhancedModelProviderService_FilterVerifiedModels_WithResults(t *testing.T) {
	logger := NewTestLogger()
	config := CreateDefaultVerificationConfig()

	emps := NewEnhancedModelProviderService("", logger, config)

	models := []Model{
		{ID: "model1", Name: "Model 1", ProviderID: "openai"},
		{ID: "model2", Name: "Model 2", ProviderID: "openai"},
		{ID: "model3", Name: "Model 3", ProviderID: "openai"},
	}

	results := map[string]*ModelVerificationResult{
		"openai:model1": {
			VerificationStatus: "verified",
			CanSeeCode:         true,
		},
		"openai:model2": {
			VerificationStatus: "failed",
			CanSeeCode:         false,
		},
		// model3 has no result
	}

	filtered := emps.filterVerifiedModels(models, results)
	assert.Len(t, filtered, 1)
	assert.Equal(t, "model1", filtered[0].ID)
}

func TestEnhancedModelProviderService_FilterVerifiedModels_EmptyModels(t *testing.T) {
	logger := NewTestLogger()
	config := CreateDefaultVerificationConfig()

	emps := NewEnhancedModelProviderService("", logger, config)

	filtered := emps.filterVerifiedModels([]Model{}, nil)
	assert.Empty(t, filtered)
}

func TestEnhancedModelProviderService_FilterVerifiedModels_AllVerified(t *testing.T) {
	logger := NewTestLogger()
	config := CreateDefaultVerificationConfig()

	emps := NewEnhancedModelProviderService("", logger, config)

	models := []Model{
		{ID: "model1", Name: "Model 1", ProviderID: "openai"},
		{ID: "model2", Name: "Model 2", ProviderID: "anthropic"},
	}

	results := map[string]*ModelVerificationResult{
		"openai:model1": {
			VerificationStatus: "verified",
			CanSeeCode:         true,
		},
		"anthropic:model2": {
			VerificationStatus: "verified",
			CanSeeCode:         true,
		},
	}

	filtered := emps.filterVerifiedModels(models, results)
	assert.Len(t, filtered, 2)
}

func TestEnhancedModelProviderService_FilterVerifiedModels_NoneVerified(t *testing.T) {
	logger := NewTestLogger()
	config := CreateDefaultVerificationConfig()

	emps := NewEnhancedModelProviderService("", logger, config)

	models := []Model{
		{ID: "model1", Name: "Model 1", ProviderID: "openai"},
		{ID: "model2", Name: "Model 2", ProviderID: "anthropic"},
	}

	results := map[string]*ModelVerificationResult{
		"openai:model1": {
			VerificationStatus: "verified",
			CanSeeCode:         false, // Can't see code
		},
		"anthropic:model2": {
			VerificationStatus: "failed",
			CanSeeCode:         true,
		},
	}

	filtered := emps.filterVerifiedModels(models, results)
	assert.Empty(t, filtered)
}
