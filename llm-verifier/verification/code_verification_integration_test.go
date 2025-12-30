package verification

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"llm-verifier/database"
	"llm-verifier/logging"
)

// MockProviderService implements ProviderServiceInterface for testing
type MockProviderService struct {
	providers map[string]ProviderClientInfo
	models    map[string][]ModelInfo
	err       error
}

func (m *MockProviderService) GetAllProviders() map[string]ProviderClientInfo {
	return m.providers
}

func (m *MockProviderService) GetModels(providerID string) ([]ModelInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.models[providerID], nil
}

// setupTestLogger creates a test logger
func setupTestLogger(t *testing.T) *logging.Logger {
	dbFile := "/tmp/test_verify_integration_" + time.Now().Format("20060102150405") + ".db"
	db, err := database.New(dbFile)
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Close()
		os.Remove(dbFile)
	})

	config := map[string]any{
		"console_level": "error", // Suppress console output during tests
		"file_level":    "error",
	}

	logger, err := logging.NewLogger(db, config)
	require.NoError(t, err)

	t.Cleanup(func() {
		logger.Close()
	})

	return logger
}

// Test NewCodeVerificationIntegration
func TestNewCodeVerificationIntegration(t *testing.T) {
	verificationService := &CodeVerificationService{}
	logger := setupTestLogger(t)
	providerService := &MockProviderService{}

	integration := NewCodeVerificationIntegration(verificationService, nil, logger, providerService)

	assert.NotNil(t, integration)
	assert.Equal(t, verificationService, integration.verificationService)
	assert.Equal(t, logger, integration.logger)
	assert.Equal(t, providerService, integration.providerService)
	assert.Nil(t, integration.db)
}

func TestNewCodeVerificationIntegration_WithDatabase(t *testing.T) {
	verificationService := &CodeVerificationService{}
	logger := setupTestLogger(t)
	providerService := &MockProviderService{}

	// Create in-memory database for testing
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	integration := NewCodeVerificationIntegration(verificationService, db, logger, providerService)

	assert.NotNil(t, integration)
	assert.NotNil(t, integration.db)
}

// Test shouldVerifyModel
func TestShouldVerifyModel_WithCodeFeature(t *testing.T) {
	integration := &CodeVerificationIntegration{
		logger: setupTestLogger(t),
	}

	model := ModelInfo{
		ID:         "gpt-4",
		Name:       "GPT-4",
		ProviderID: "openai",
		Features: map[string]interface{}{
			"code": true,
		},
	}

	result := integration.shouldVerifyModel(model)
	assert.True(t, result)
}

func TestShouldVerifyModel_WithToolCallFeature(t *testing.T) {
	integration := &CodeVerificationIntegration{
		logger: setupTestLogger(t),
	}

	model := ModelInfo{
		ID:         "model-1",
		Name:       "Model One",
		ProviderID: "provider-1",
		Features: map[string]interface{}{
			"tool_call": true,
		},
	}

	result := integration.shouldVerifyModel(model)
	assert.True(t, result)
}

func TestShouldVerifyModel_WithReasoningFeature(t *testing.T) {
	integration := &CodeVerificationIntegration{
		logger: setupTestLogger(t),
	}

	model := ModelInfo{
		ID:         "model-1",
		Name:       "Model One",
		ProviderID: "provider-1",
		Features: map[string]interface{}{
			"reasoning": true,
		},
	}

	result := integration.shouldVerifyModel(model)
	assert.True(t, result)
}

func TestShouldVerifyModel_WithCodeKeyword(t *testing.T) {
	integration := &CodeVerificationIntegration{
		logger: setupTestLogger(t),
	}

	tests := []struct {
		name     string
		model    ModelInfo
		expected bool
	}{
		{
			name: "code in name",
			model: ModelInfo{
				ID:         "model-1",
				Name:       "Code Assistant",
				ProviderID: "provider-1",
			},
			expected: true,
		},
		{
			name: "coder in ID",
			model: ModelInfo{
				ID:         "deepseek-coder",
				Name:       "DeepSeek",
				ProviderID: "deepseek",
			},
			expected: true,
		},
		{
			name: "gpt-4 in name",
			model: ModelInfo{
				ID:         "some-id",
				Name:       "GPT-4 Model",
				ProviderID: "provider-1",
			},
			expected: true,
		},
		{
			name: "claude in name",
			model: ModelInfo{
				ID:         "claude-3",
				Name:       "Claude 3",
				ProviderID: "anthropic",
			},
			expected: true,
		},
		{
			name: "mistral in name",
			model: ModelInfo{
				ID:         "mistral-7b",
				Name:       "Mistral 7B",
				ProviderID: "mistral",
			},
			expected: true,
		},
		{
			name: "llama in name",
			model: ModelInfo{
				ID:         "llama-3.2",
				Name:       "Llama 3.2",
				ProviderID: "meta",
			},
			expected: true,
		},
		{
			name: "codestral in name",
			model: ModelInfo{
				ID:         "codestral-latest",
				Name:       "Codestral",
				ProviderID: "mistral",
			},
			expected: true,
		},
		{
			name: "programming in name",
			model: ModelInfo{
				ID:         "prog-model",
				Name:       "Programming Assistant",
				ProviderID: "provider-1",
			},
			expected: true,
		},
		{
			name: "development in name",
			model: ModelInfo{
				ID:         "dev-model",
				Name:       "Development Helper",
				ProviderID: "provider-1",
			},
			expected: true,
		},
		{
			name: "no code keywords",
			model: ModelInfo{
				ID:         "text-model",
				Name:       "Text Generator",
				ProviderID: "provider-1",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := integration.shouldVerifyModel(tt.model)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldVerifyModel_NoFeaturesNoKeywords(t *testing.T) {
	integration := &CodeVerificationIntegration{
		logger: setupTestLogger(t),
	}

	model := ModelInfo{
		ID:         "simple-model",
		Name:       "Simple Model",
		ProviderID: "provider-1",
		Features:   nil,
	}

	result := integration.shouldVerifyModel(model)
	assert.False(t, result)
}

func TestShouldVerifyModel_FeaturesFalse(t *testing.T) {
	integration := &CodeVerificationIntegration{
		logger: setupTestLogger(t),
	}

	model := ModelInfo{
		ID:         "simple-model",
		Name:       "Simple Model",
		ProviderID: "provider-1",
		Features: map[string]interface{}{
			"code":      false,
			"tool_call": false,
			"reasoning": false,
		},
	}

	result := integration.shouldVerifyModel(model)
	assert.False(t, result)
}

// Test VerifyAllModelsWithCodeSupport
func TestVerifyAllModelsWithCodeSupport_NoProviders(t *testing.T) {
	providerService := &MockProviderService{
		providers: map[string]ProviderClientInfo{},
		models:    map[string][]ModelInfo{},
	}

	integration := NewCodeVerificationIntegration(
		&CodeVerificationService{},
		nil,
		setupTestLogger(t),
		providerService,
	)

	ctx := context.Background()
	results, err := integration.VerifyAllModelsWithCodeSupport(ctx)

	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestVerifyAllModelsWithCodeSupport_GetModelsError(t *testing.T) {
	providerService := &MockProviderService{
		providers: map[string]ProviderClientInfo{
			"openai": {
				ProviderID: "openai",
				BaseURL:    "https://api.openai.com",
				APIKey:     "test-key",
			},
		},
		models: map[string][]ModelInfo{},
		err:    errors.New("failed to get models"),
	}

	integration := NewCodeVerificationIntegration(
		&CodeVerificationService{},
		nil,
		setupTestLogger(t),
		providerService,
	)

	ctx := context.Background()
	results, err := integration.VerifyAllModelsWithCodeSupport(ctx)

	assert.NoError(t, err) // Error is logged but not returned
	assert.Empty(t, results)
}

func TestVerifyAllModelsWithCodeSupport_NoCodeModels(t *testing.T) {
	providerService := &MockProviderService{
		providers: map[string]ProviderClientInfo{
			"provider-1": {
				ProviderID: "provider-1",
				BaseURL:    "https://api.example.com",
				APIKey:     "test-key",
			},
		},
		models: map[string][]ModelInfo{
			"provider-1": {
				{
					ID:         "text-model",
					Name:       "Text Generator",
					ProviderID: "provider-1",
				},
			},
		},
	}

	integration := NewCodeVerificationIntegration(
		&CodeVerificationService{},
		nil,
		setupTestLogger(t),
		providerService,
	)

	ctx := context.Background()
	results, err := integration.VerifyAllModelsWithCodeSupport(ctx)

	assert.NoError(t, err)
	assert.Empty(t, results) // No models should be verified as none support code
}

// Test verifyModel
func TestVerifyModel_ProviderNotFound(t *testing.T) {
	providerService := &MockProviderService{
		providers: map[string]ProviderClientInfo{},
		models:    map[string][]ModelInfo{},
	}

	integration := NewCodeVerificationIntegration(
		&CodeVerificationService{},
		nil,
		setupTestLogger(t),
		providerService,
	)

	model := ModelInfo{
		ID:         "gpt-4",
		Name:       "GPT-4",
		ProviderID: "openai",
	}

	ctx := context.Background()
	result, err := integration.verifyModel(ctx, model)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "error", result.Status)
	assert.Contains(t, result.ErrorMessage, "Provider openai not found")
}

// Test updateModelVerificationStatus
// Note: ModelInfo is passed by value, so modifications don't persist to caller
// This tests that the function doesn't error - modification happens on local copy
func TestUpdateModelVerificationStatus(t *testing.T) {
	integration := &CodeVerificationIntegration{
		logger: setupTestLogger(t),
	}

	model := ModelInfo{
		ID:         "gpt-4",
		Name:       "GPT-4",
		ProviderID: "openai",
		Features:   nil,
	}

	result := &CodeVerificationResult{
		CodeVisibility:    true,
		ToolSupport:       true,
		VerificationScore: 0.95,
		TestedAt:          time.Now(),
		VerificationID:    "ver-123",
		Status:            "verified",
	}

	// Function doesn't return error
	err := integration.updateModelVerificationStatus(model, result)
	assert.NoError(t, err)

	// Note: model.Features is nil because model is passed by value
	// The function modifies a local copy, not the original
	assert.Nil(t, model.Features)
}

func TestUpdateModelVerificationStatus_ExistingFeatures(t *testing.T) {
	integration := &CodeVerificationIntegration{
		logger: setupTestLogger(t),
	}

	// When Features map already exists, map is a reference type
	// so modifications DO persist through the reference
	model := ModelInfo{
		ID:         "gpt-4",
		Name:       "GPT-4",
		ProviderID: "openai",
		Features: map[string]interface{}{
			"existing_feature": true,
		},
	}

	result := &CodeVerificationResult{
		CodeVisibility:    false,
		ToolSupport:       false,
		VerificationScore: 0.5,
		TestedAt:          time.Now(),
		VerificationID:    "ver-456",
		Status:            "partial",
	}

	err := integration.updateModelVerificationStatus(model, result)

	assert.NoError(t, err)
	// Map modifications persist since map is a reference type
	assert.Equal(t, true, model.Features["existing_feature"]) // Preserved
	assert.Equal(t, false, model.Features["code_visibility_verified"])
	assert.Equal(t, false, model.Features["tool_support_verified"])
	assert.Equal(t, 0.5, model.Features["verification_score"])
}

// Test VerificationResult struct
func TestVerificationResult_Struct(t *testing.T) {
	now := time.Now()
	result := VerificationResult{
		ProviderID:        "openai",
		ModelID:           "gpt-4",
		VerificationID:    "ver-123",
		Status:            "verified",
		CodeVisibility:    true,
		ToolSupport:       true,
		VerificationScore: 0.95,
		VerifiedAt:        now,
		ErrorMessage:      "",
	}

	assert.Equal(t, "openai", result.ProviderID)
	assert.Equal(t, "gpt-4", result.ModelID)
	assert.Equal(t, "ver-123", result.VerificationID)
	assert.Equal(t, "verified", result.Status)
	assert.True(t, result.CodeVisibility)
	assert.True(t, result.ToolSupport)
	assert.Equal(t, 0.95, result.VerificationScore)
	assert.Equal(t, now, result.VerifiedAt)
	assert.Empty(t, result.ErrorMessage)
}

func TestVerificationResult_WithError(t *testing.T) {
	result := VerificationResult{
		ProviderID:   "openai",
		ModelID:      "gpt-4",
		Status:       "error",
		ErrorMessage: "connection timeout",
		VerifiedAt:   time.Now(),
	}

	assert.Equal(t, "error", result.Status)
	assert.Equal(t, "connection timeout", result.ErrorMessage)
	assert.False(t, result.CodeVisibility)
	assert.False(t, result.ToolSupport)
}

// Test helper functions
func TestPtrBool(t *testing.T) {
	truePtr := ptrBool(true)
	falsePtr := ptrBool(false)

	assert.NotNil(t, truePtr)
	assert.NotNil(t, falsePtr)
	assert.True(t, *truePtr)
	assert.False(t, *falsePtr)
}

func TestPtrInt(t *testing.T) {
	zeroPtr := ptrInt(0)
	positivePtr := ptrInt(100)
	negativePtr := ptrInt(-50)

	assert.NotNil(t, zeroPtr)
	assert.NotNil(t, positivePtr)
	assert.NotNil(t, negativePtr)
	assert.Equal(t, 0, *zeroPtr)
	assert.Equal(t, 100, *positivePtr)
	assert.Equal(t, -50, *negativePtr)
}

// Test GetVerificationStatus with database
func TestGetVerificationStatus_WithDatabase(t *testing.T) {
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	integration := NewCodeVerificationIntegration(
		&CodeVerificationService{},
		db,
		setupTestLogger(t),
		&MockProviderService{},
	)

	// Model not found should return error
	_, err = integration.GetVerificationStatus("nonexistent", "provider")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "model not found")
}

// Test GetAllVerifiedModels with database
func TestGetAllVerifiedModels_WithDatabase(t *testing.T) {
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	integration := NewCodeVerificationIntegration(
		&CodeVerificationService{},
		db,
		setupTestLogger(t),
		&MockProviderService{},
	)

	results, err := integration.GetAllVerifiedModels()
	assert.NoError(t, err)
	assert.Empty(t, results) // No models in fresh database
}

// Test concurrent verification
func TestVerifyAllModelsWithCodeSupport_Concurrent(t *testing.T) {
	providerService := &MockProviderService{
		providers: map[string]ProviderClientInfo{
			"provider-1": {ProviderID: "provider-1", BaseURL: "https://api1.example.com", APIKey: "key1"},
			"provider-2": {ProviderID: "provider-2", BaseURL: "https://api2.example.com", APIKey: "key2"},
			"provider-3": {ProviderID: "provider-3", BaseURL: "https://api3.example.com", APIKey: "key3"},
		},
		models: map[string][]ModelInfo{
			"provider-1": {
				{ID: "text-only", Name: "Text Only", ProviderID: "provider-1"},
			},
			"provider-2": {
				{ID: "text-only-2", Name: "Text Only 2", ProviderID: "provider-2"},
			},
			"provider-3": {
				{ID: "text-only-3", Name: "Text Only 3", ProviderID: "provider-3"},
			},
		},
	}

	integration := NewCodeVerificationIntegration(
		&CodeVerificationService{},
		nil,
		setupTestLogger(t),
		providerService,
	)

	ctx := context.Background()
	results, err := integration.VerifyAllModelsWithCodeSupport(ctx)

	assert.NoError(t, err)
	// All models are text-only so none should be verified
	assert.Empty(t, results)
}

// Test context cancellation
func TestVerifyAllModelsWithCodeSupport_ContextCancellation(t *testing.T) {
	providerService := &MockProviderService{
		providers: map[string]ProviderClientInfo{
			"openai": {ProviderID: "openai", BaseURL: "https://api.openai.com", APIKey: "key"},
		},
		models: map[string][]ModelInfo{
			"openai": {
				{ID: "gpt-4", Name: "GPT-4", ProviderID: "openai", Features: map[string]interface{}{"code": true}},
			},
		},
	}

	integration := NewCodeVerificationIntegration(
		&CodeVerificationService{},
		nil,
		setupTestLogger(t),
		providerService,
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Should still complete without panic even if context is cancelled
	results, err := integration.VerifyAllModelsWithCodeSupport(ctx)

	assert.NoError(t, err)
	// Result may or may not be empty depending on timing
	_ = results
}
