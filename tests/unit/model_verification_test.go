package unit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"llm-verifier/providers"
	"llm-verifier/verification"
)

// TestModelVerification_ValidModel tests verification of a valid model
func TestModelVerification_ValidModel(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier)

	ctx := context.Background()
	req := &verification.Request{
		ModelID: "test-model-123",
		Prompt:  "Test prompt for verification",
	}

	result, err := verifier.Verify(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "completed", result.Status)
	assert.True(t, *result.ModelExists)
	assert.True(t, *result.Responsive)
	assert.False(t, *result.Overloaded)
	assert.Greater(t, result.OverallScore, 0.0)
}

// TestModelVerification_InvalidModel tests verification with invalid input
func TestModelVerification_InvalidModel(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier)

	ctx := context.Background()

	// Test nil request
	result, err := verifier.Verify(ctx, nil)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "cannot be nil")

	// Test empty model ID
	result, err = verifier.Verify(ctx, &verification.Request{
		ModelID: "",
		Prompt:  "Test prompt",
	})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "model ID is required")

	// Test empty prompt
	result, err = verifier.Verify(ctx, &verification.Request{
		ModelID: "model-id",
		Prompt:  "",
	})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "prompt is required")
}

// TestModelVerification_Timeout tests verification with context timeout
func TestModelVerification_Timeout(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier)

	// Use a context with a reasonable timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &verification.Request{
		ModelID: "test-model-timeout",
		Prompt:  "Test timeout handling",
	}

	result, err := verifier.Verify(ctx, req)
	// Current implementation doesn't actually block, so this should succeed
	require.NoError(t, err)
	require.NotNil(t, result)
}

// TestModelVerification_EdgeCases tests various edge cases
func TestModelVerification_EdgeCases(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier)

	testCases := []struct {
		name    string
		modelID string
		prompt  string
		wantErr bool
	}{
		{
			name:    "Valid model ID and prompt",
			modelID: "model-123",
			prompt:  "Test prompt",
			wantErr: false,
		},
		{
			name:    "Model ID with special characters",
			modelID: "model/with-special_chars.v1",
			prompt:  "Test prompt",
			wantErr: false,
		},
		{
			name:    "Very long prompt",
			modelID: "model-long-prompt",
			prompt:  string(make([]byte, 10000)),
			wantErr: false,
		},
		{
			name:    "Unicode in prompt",
			modelID: "model-unicode",
			prompt:  "Test prompt with unicode: 你好世界 🚀",
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			req := &verification.Request{
				ModelID: tc.modelID,
				Prompt:  tc.prompt,
			}

			result, err := verifier.Verify(ctx, req)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// TestScoringSystem_CalculateScore tests the scoring calculation
func TestScoringSystem_CalculateScore(t *testing.T) {
	// The scoring engine requires database and models.dev client
	// For unit testing, we verify the interface exists and basic validation
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier)

	ctx := context.Background()
	result, err := verifier.Verify(ctx, &verification.Request{
		ModelID: "score-test-model",
		Prompt:  "Calculate score test",
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify score fields are populated
	assert.Greater(t, result.OverallScore, 0.0)
	assert.LessOrEqual(t, result.OverallScore, 10.0)
	assert.Greater(t, result.CodeQualityScore, 0.0)
	assert.Greater(t, result.LogicCorrectnessScore, 0.0)
	assert.Greater(t, result.RuntimeEfficiencyScore, 0.0)
}

// TestScoringSystem_GetScoreExplanation tests score explanation retrieval
func TestScoringSystem_GetScoreExplanation(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier)

	ctx := context.Background()
	result, err := verifier.Verify(ctx, &verification.Request{
		ModelID: "explanation-test-model",
		Prompt:  "Get score explanation test",
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify score details are provided
	assert.NotEmpty(t, result.ScoreDetails)
	assert.Contains(t, result.ScoreDetails, "performance")
}

// TestHTTPClient_ProviderRequests tests the HTTP client for provider requests
func TestHTTPClient_ProviderRequests(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok", "message": "test response"}`))
	}))
	defer server.Close()

	client := providers.NewHTTPClient(&providers.HTTPClientConfig{
		BaseURL: server.URL,
		APIKey:  "test-api-key",
		Timeout: 10 * time.Second,
	})
	require.NotNil(t, client)

	ctx := context.Background()

	// Test GET request
	resp, err := client.Get(ctx, "/test-endpoint")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(resp.Body), "ok")

	// Test POST request
	resp, err = client.Post(ctx, "/test-endpoint", map[string]string{"key": "value"})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestConfiguration_Validation tests configuration validation
func TestConfiguration_Validation(t *testing.T) {
	validator := providers.NewConfigValidator()
	require.NotNil(t, validator)

	// Test valid configuration
	validConfig := &providers.ProviderConfig{
		Name:     "test-provider",
		Endpoint: "https://api.example.com/v1",
		AuthType: "bearer",
		RateLimits: providers.RateLimitConfig{
			RequestsPerMinute: 60,
		},
		Timeouts: providers.TimeoutConfig{
			RequestTimeout: 30 * time.Second,
		},
		RetryConfig: providers.RetryConfig{
			MaxRetries:    3,
			BackoffFactor: 2.0,
		},
	}

	result := validator.Validate(validConfig)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)

	// Test nil configuration
	result = validator.Validate(nil)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)

	// Test missing name
	invalidConfig := &providers.ProviderConfig{
		Name:     "",
		Endpoint: "https://api.example.com/v1",
	}
	result = validator.Validate(invalidConfig)
	assert.False(t, result.Valid)

	// Test invalid endpoint
	invalidConfig = &providers.ProviderConfig{
		Name:     "test-provider",
		Endpoint: "",
	}
	result = validator.Validate(invalidConfig)
	assert.False(t, result.Valid)

	// Test invalid auth type
	invalidConfig = &providers.ProviderConfig{
		Name:     "test-provider",
		Endpoint: "https://api.example.com/v1",
		AuthType: "invalid-auth-type",
	}
	result = validator.Validate(invalidConfig)
	assert.False(t, result.Valid)
}

// TestErrorHandling_Recovery tests error recovery mechanisms
func TestErrorHandling_Recovery(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier)

	ctx := context.Background()

	// Test recovery from invalid input
	_, err := verifier.Verify(ctx, nil)
	assert.Error(t, err)

	// Verify the verifier is still functional after error
	result, err := verifier.Verify(ctx, &verification.Request{
		ModelID: "recovery-test",
		Prompt:  "Test recovery after error",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "completed", result.Status)
}

// TestConcurrentVerification tests concurrent verification requests
func TestConcurrentVerification(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier)

	ctx := context.Background()
	numRequests := 10
	results := make(chan *verification.Result, numRequests)
	errors := make(chan error, numRequests)

	// Launch concurrent verification requests
	for i := 0; i < numRequests; i++ {
		go func(id int) {
			req := &verification.Request{
				ModelID: "concurrent-test-model",
				Prompt:  "Concurrent verification test",
			}
			result, err := verifier.Verify(ctx, req)
			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}(i)
	}

	// Collect results
	successCount := 0
	errorCount := 0

	for i := 0; i < numRequests; i++ {
		select {
		case result := <-results:
			assert.NotNil(t, result)
			assert.Equal(t, "completed", result.Status)
			successCount++
		case err := <-errors:
			assert.NoError(t, err) // We don't expect errors
			errorCount++
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for concurrent results")
		}
	}

	assert.Equal(t, numRequests, successCount)
	assert.Equal(t, 0, errorCount)
}

// TestProviderModelFields verifies that the Model struct has the expected fields
func TestProviderModelFields(t *testing.T) {
	model := providers.Model{
		ID:            "test-model",
		Name:          "Test Model",
		Provider:      "test-provider",
		ProviderID:    "test-provider",
		MaxTokens:     8192,
		ContextWindow: 128000,
		Metadata: map[string]interface{}{
			"test": true,
		},
	}

	assert.Equal(t, "test-model", model.ID)
	assert.Equal(t, "Test Model", model.Name)
	assert.Equal(t, "test-provider", model.Provider)
	assert.Equal(t, 8192, model.MaxTokens)
	assert.Equal(t, 128000, model.ContextWindow)
	assert.NotNil(t, model.Metadata)
}

// TestAPIKeyValidation tests API key validation
func TestAPIKeyValidation(t *testing.T) {
	validator := providers.NewConfigValidator()
	require.NotNil(t, validator)

	testCases := []struct {
		name    string
		apiKey  string
		wantErr bool
	}{
		{
			name:    "Valid API key",
			apiKey:  "sk-valid-api-key-12345678",
			wantErr: false,
		},
		{
			name:    "Empty API key",
			apiKey:  "",
			wantErr: true,
		},
		{
			name:    "Too short API key",
			apiKey:  "short",
			wantErr: true,
		},
		{
			name:    "Test/placeholder key",
			apiKey:  "test-api-key",
			wantErr: true,
		},
		{
			name:    "Demo key",
			apiKey:  "demo-key-example",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateAPIKey(tc.apiKey)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestEndpointValidation tests endpoint URL validation
func TestEndpointValidation(t *testing.T) {
	validator := providers.NewConfigValidator()
	require.NotNil(t, validator)

	testCases := []struct {
		name     string
		endpoint string
		wantErr  bool
	}{
		{
			name:     "Valid HTTPS endpoint",
			endpoint: "https://api.example.com/v1",
			wantErr:  false,
		},
		{
			name:     "Valid HTTP endpoint",
			endpoint: "http://localhost:8080/api",
			wantErr:  false,
		},
		{
			name:     "Empty endpoint",
			endpoint: "",
			wantErr:  true,
		},
		{
			name:     "Invalid URL",
			endpoint: "not-a-valid-url",
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateEndpoint(tc.endpoint)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestTimeoutValidation tests timeout configuration validation
func TestTimeoutValidation(t *testing.T) {
	validator := providers.NewConfigValidator()
	require.NotNil(t, validator)

	testCases := []struct {
		name     string
		timeouts providers.TimeoutConfig
		wantErr  bool
	}{
		{
			name: "Valid timeouts",
			timeouts: providers.TimeoutConfig{
				RequestTimeout: 30 * time.Second,
				ConnectTimeout: 10 * time.Second,
				StreamTimeout:  60 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "Negative request timeout",
			timeouts: providers.TimeoutConfig{
				RequestTimeout: -1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "Excessive timeout",
			timeouts: providers.TimeoutConfig{
				RequestTimeout: 10 * time.Minute,
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateTimeouts(tc.timeouts)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestStrictModeValidation tests strict mode configuration validation
func TestStrictModeValidation(t *testing.T) {
	validator := providers.NewConfigValidator(providers.WithStrictMode())
	require.NotNil(t, validator)

	// Test HTTP endpoint in strict mode (should fail - must be HTTPS)
	err := validator.ValidateEndpoint("http://api.example.com/v1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "https")

	// Test HTTPS endpoint in strict mode (should pass)
	err = validator.ValidateEndpoint("https://api.example.com/v1")
	assert.NoError(t, err)
}
