package providers

import (
	"net/http"
	"testing"
	"time"

	"llm-verifier/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelVerificationService_GetVerificationResult(t *testing.T) {
	logger := NewTestLogger()
	httpClient := client.NewHTTPClient(30 * time.Second)
	config := CreateDefaultVerificationConfig()

	mvs := NewModelVerificationService(httpClient, logger, config)

	// Should return nil for non-existent result
	result := mvs.GetVerificationResult("openai", "gpt-4")
	assert.Nil(t, result)
}

func TestModelVerificationService_GetAllVerificationResults_Empty(t *testing.T) {
	logger := NewTestLogger()
	httpClient := client.NewHTTPClient(30 * time.Second)
	config := CreateDefaultVerificationConfig()

	mvs := NewModelVerificationService(httpClient, logger, config)

	results := mvs.GetAllVerificationResults()
	assert.NotNil(t, results)
	assert.Empty(t, results)
}

func TestModelVerificationService_SetStrictMode(t *testing.T) {
	logger := NewTestLogger()
	httpClient := client.NewHTTPClient(30 * time.Second)
	config := CreateDefaultVerificationConfig()

	mvs := NewModelVerificationService(httpClient, logger, config)

	// Should not panic
	mvs.SetStrictMode(true)
	mvs.SetStrictMode(false)
}

func TestModelVerificationService_ClearVerificationResults(t *testing.T) {
	logger := NewTestLogger()
	httpClient := client.NewHTTPClient(30 * time.Second)
	config := CreateDefaultVerificationConfig()

	mvs := NewModelVerificationService(httpClient, logger, config)

	// Store a result
	mvs.storeVerificationResult("openai:gpt-4", &ModelVerificationResult{
		VerificationStatus: "verified",
	})

	// Verify it exists
	results := mvs.GetAllVerificationResults()
	assert.Len(t, results, 1)

	// Clear and verify it's gone
	mvs.ClearVerificationResults()
	results = mvs.GetAllVerificationResults()
	assert.Empty(t, results)
}

func TestModelVerificationService_StoreAndRetrieveResult(t *testing.T) {
	logger := NewTestLogger()
	httpClient := client.NewHTTPClient(30 * time.Second)
	config := CreateDefaultVerificationConfig()

	mvs := NewModelVerificationService(httpClient, logger, config)

	// Store a result
	testResult := &ModelVerificationResult{
		VerificationStatus:  "verified",
		CanSeeCode:          true,
		AffirmativeResponse: true,
		VerificationScore:   0.95,
	}
	mvs.storeVerificationResult("anthropic:claude-3", testResult)

	// Retrieve it
	result := mvs.GetVerificationResult("anthropic", "claude-3")
	require.NotNil(t, result)
	assert.Equal(t, "verified", result.VerificationStatus)
	assert.True(t, result.CanSeeCode)
	assert.True(t, result.AffirmativeResponse)
	assert.Equal(t, 0.95, result.VerificationScore)
}

func TestModelVerificationService_IsModelVerified_VerificationDisabled(t *testing.T) {
	logger := NewTestLogger()
	httpClient := client.NewHTTPClient(30 * time.Second)
	config := VerificationConfig{
		Enabled: false,
	}

	mvs := NewModelVerificationService(httpClient, logger, config)
	mvs.EnableVerification(false)

	// When verification is disabled, all models should be considered verified
	verified := mvs.IsModelVerified("openai", "gpt-4")
	assert.True(t, verified)
}

func TestModelVerificationService_IsModelVerified_NoResult(t *testing.T) {
	logger := NewTestLogger()
	httpClient := client.NewHTTPClient(30 * time.Second)
	config := CreateDefaultVerificationConfig()

	mvs := NewModelVerificationService(httpClient, logger, config)
	// Strict mode by default

	// No verification result, should not be verified
	verified := mvs.IsModelVerified("openai", "gpt-4")
	assert.False(t, verified)
}

func TestModelVerificationService_IsModelVerified_StrictMode_Success(t *testing.T) {
	logger := NewTestLogger()
	httpClient := client.NewHTTPClient(30 * time.Second)
	config := CreateDefaultVerificationConfig()

	mvs := NewModelVerificationService(httpClient, logger, config)
	mvs.SetStrictMode(true)

	// Store a fully verified result
	mvs.storeVerificationResult("openai:gpt-4", &ModelVerificationResult{
		VerificationStatus:  "verified",
		CanSeeCode:          true,
		AffirmativeResponse: true,
		VerificationScore:   0.85,
	})

	verified := mvs.IsModelVerified("openai", "gpt-4")
	assert.True(t, verified)
}

func TestModelVerificationService_IsModelVerified_StrictMode_LowScore(t *testing.T) {
	logger := NewTestLogger()
	httpClient := client.NewHTTPClient(30 * time.Second)
	config := CreateDefaultVerificationConfig()

	mvs := NewModelVerificationService(httpClient, logger, config)
	mvs.SetStrictMode(true)

	// Store a result with low score
	mvs.storeVerificationResult("openai:gpt-4", &ModelVerificationResult{
		VerificationStatus:  "verified",
		CanSeeCode:          true,
		AffirmativeResponse: true,
		VerificationScore:   0.5, // Below 0.7 threshold
	})

	verified := mvs.IsModelVerified("openai", "gpt-4")
	assert.False(t, verified)
}

func TestModelVerificationService_IsModelVerified_NonStrictMode(t *testing.T) {
	logger := NewTestLogger()
	httpClient := client.NewHTTPClient(30 * time.Second)
	config := CreateDefaultVerificationConfig()

	mvs := NewModelVerificationService(httpClient, logger, config)
	mvs.SetStrictMode(false)

	// Store any result (not error)
	mvs.storeVerificationResult("openai:gpt-4", &ModelVerificationResult{
		VerificationStatus: "partial",
	})

	verified := mvs.IsModelVerified("openai", "gpt-4")
	assert.True(t, verified)
}

func TestModelVerificationService_IsModelVerified_NonStrictMode_Error(t *testing.T) {
	logger := NewTestLogger()
	httpClient := client.NewHTTPClient(30 * time.Second)
	config := CreateDefaultVerificationConfig()

	mvs := NewModelVerificationService(httpClient, logger, config)
	mvs.SetStrictMode(false)

	// Store error result
	mvs.storeVerificationResult("openai:gpt-4", &ModelVerificationResult{
		VerificationStatus: "error",
	})

	verified := mvs.IsModelVerified("openai", "gpt-4")
	assert.False(t, verified)
}

func TestModelVerificationService_GetVerifiedModels_VerificationDisabled(t *testing.T) {
	logger := NewTestLogger()
	httpClient := client.NewHTTPClient(30 * time.Second)
	config := CreateDefaultVerificationConfig()

	mvs := NewModelVerificationService(httpClient, logger, config)
	mvs.EnableVerification(false)

	models := []Model{
		{ID: "model1", Name: "Model 1", ProviderID: "openai"},
		{ID: "model2", Name: "Model 2", ProviderID: "anthropic"},
	}

	verified := mvs.GetVerifiedModels(models)
	assert.Len(t, verified, 2)
}

func TestModelVerificationService_GetVerifiedModels_FilterVerified(t *testing.T) {
	logger := NewTestLogger()
	httpClient := client.NewHTTPClient(30 * time.Second)
	config := CreateDefaultVerificationConfig()

	mvs := NewModelVerificationService(httpClient, logger, config)
	mvs.SetStrictMode(true)

	// Store results for some models
	mvs.storeVerificationResult("openai:model1", &ModelVerificationResult{
		VerificationStatus:  "verified",
		CanSeeCode:          true,
		AffirmativeResponse: true,
		VerificationScore:   0.85,
	})

	// model2 has no result

	models := []Model{
		{ID: "model1", Name: "Model 1", ProviderID: "openai"},
		{ID: "model2", Name: "Model 2", ProviderID: "openai"},
	}

	verified := mvs.GetVerifiedModels(models)
	assert.Len(t, verified, 1)
	assert.Equal(t, "model1", verified[0].ID)
}

// Test verificationProviderClient
func TestVerificationProviderClient_GetBaseURL(t *testing.T) {
	vpc := &verificationProviderClient{
		baseURL: "https://api.openai.com/v1",
	}

	assert.Equal(t, "https://api.openai.com/v1", vpc.GetBaseURL())
}

func TestVerificationProviderClient_GetAPIKey(t *testing.T) {
	vpc := &verificationProviderClient{
		apiKey: "sk-test-key",
	}

	assert.Equal(t, "sk-test-key", vpc.GetAPIKey())
}

func TestVerificationProviderClient_GetHTTPClient(t *testing.T) {
	httpClient := &http.Client{Timeout: 60 * time.Second}
	vpc := &verificationProviderClient{
		httpClient: httpClient,
	}

	assert.Equal(t, httpClient, vpc.GetHTTPClient())
}

func TestVerificationProviderClient_AllFields(t *testing.T) {
	httpClient := &http.Client{Timeout: 60 * time.Second}
	vpc := &verificationProviderClient{
		baseURL:    "https://api.example.com/v1",
		apiKey:     "sk-secret-123",
		httpClient: httpClient,
	}

	assert.Equal(t, "https://api.example.com/v1", vpc.GetBaseURL())
	assert.Equal(t, "sk-secret-123", vpc.GetAPIKey())
	assert.Equal(t, httpClient, vpc.GetHTTPClient())
}
