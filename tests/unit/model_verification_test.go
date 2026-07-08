package unit

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.llmsverifier/providers"
	"digital.vasic.llmsverifier/verification"
)

// TestModelVerification_ValidModel tests verification of a valid model
//
// Round-17 converted VerifyModel to sentinel-return per §11.4/CONST-036/037 anti-bluff fix.
// Round-84 tightened this test from asserting-fabricated-values to asserting-sentinel-error.
// The previous body asserted result.OverallScore > 0 and "completed" status against a
// hardcoded-all-capabilities-true fabrication — that was a PASS-bluff.
//
// §11.4.114/§11.4.120 reconciliation (2026-07-08, against commit 28e6625a): commit
// 28e6625a (C5, §11.4.115 RED->GREEN) wired verification.Verify to compose+persist a
// real database.VerificationResult from real C4 probes via NewVerifierWithProber(db,
// prober), and renamed the unwired-sentinel ErrVerificationNotWired to
// ErrVerifierNotConfigured (same honest-no-fabrication contract; a Verifier
// constructed via NewModelVerifier(nil)/NewVerifier(nil) — no database, no probe
// engine — still returns the sentinel loudly instead of fabricating a result, exactly
// as this unit test exercises). Real end-to-end verification against a wired
// NewVerifierWithProber is exercised by the integration suite in ./llmverifier/
// (TestVerifier_*), not by this unit test.
func TestModelVerification_ValidModel(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier)

	ctx := context.Background()
	req := &verification.Request{
		ModelID: "test-model-123",
		Prompt:  "Test prompt for verification",
	}

	result, err := verifier.Verify(ctx, req)
	require.Error(t, err)
	assert.True(t, errors.Is(err, verification.ErrVerifierNotConfigured),
		"expected ErrVerifierNotConfigured sentinel, got: %v", err)
	assert.Nil(t, result, "sentinel-return contract: nil result alongside sentinel error")
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
//
// Round-17 converted VerifyModel to sentinel-return per §11.4/CONST-036/037 anti-bluff fix.
// Round-84 tightened this test from asserting-fabricated-values to asserting-sentinel-error.
// The previous body asserted NoError + non-nil result with a 5s context — that "succeeded"
// only because the bluff returned fabricated values without ever doing work. §11.4.114/
// §11.4.120 reconciliation (2026-07-08, commit 28e6625a): the sentinel was renamed
// ErrVerificationNotWired -> ErrVerifierNotConfigured; the unconfigured-Verifier case this
// test exercises (NewModelVerifier(nil)) still surfaces the sentinel, never a fabricated
// result. Once a NewVerifierWithProber(db, prober) instance is exercised end-to-end (the
// ./llmverifier/ integration suite), this test class re-tightens to a true timeout
// assertion (deadline-exceeded via httptest server delay).
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
	require.Error(t, err)
	assert.True(t, errors.Is(err, verification.ErrVerifierNotConfigured),
		"expected ErrVerifierNotConfigured sentinel, got: %v", err)
	assert.Nil(t, result, "sentinel-return contract: nil result alongside sentinel error")
}

// TestModelVerification_EdgeCases tests various edge cases
//
// Round-17 converted VerifyModel to sentinel-return per §11.4/CONST-036/037 anti-bluff fix.
// Round-84 tightened this test from asserting-fabricated-values to asserting-sentinel-error.
// The previous body asserted NoError + non-nil result for the four "well-formed input" cases
// — that was a PASS-bluff (the inputs were valid in shape, but the production code fabricated
// the result rather than verifying anything). For well-formed inputs that pass the nil/empty
// validation, the current contract (§11.4.114/§11.4.120 reconciled 2026-07-08 against commit
// 28e6625a: sentinel renamed ErrVerificationNotWired -> ErrVerifierNotConfigured) is
// sentinel-error against an unconfigured Verifier. Input-validation rejections (nil/empty)
// remain covered by TestModelVerification_InvalidModel which is unchanged (it tests the
// pre-sentinel validation gate that still works).
func TestModelVerification_EdgeCases(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier)

	testCases := []struct {
		name    string
		modelID string
		prompt  string
	}{
		{
			name:    "Valid model ID and prompt",
			modelID: "model-123",
			prompt:  "Test prompt",
		},
		{
			name:    "Model ID with special characters",
			modelID: "model/with-special_chars.v1",
			prompt:  "Test prompt",
		},
		{
			name:    "Very long prompt",
			modelID: "model-long-prompt",
			prompt:  string(make([]byte, 10000)),
		},
		{
			name:    "Unicode in prompt",
			modelID: "model-unicode",
			prompt:  "Test prompt with unicode: 你好世界 🚀",
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
			require.Error(t, err)
			assert.True(t, errors.Is(err, verification.ErrVerifierNotConfigured),
				"expected ErrVerifierNotConfigured sentinel, got: %v", err)
			assert.Nil(t, result, "sentinel-return contract: nil result alongside sentinel error")
		})
	}
}

// TestScoringSystem_CalculateScore tests the scoring calculation
//
// Round-17 converted VerifyModel to sentinel-return per §11.4/CONST-036/037 anti-bluff fix.
// Round-84 tightened this test from asserting-fabricated-values to asserting-sentinel-error.
// The previous body asserted OverallScore > 0 and a swarm of score-field positivity assertions
// against a hardcoded `8.5` fabrication — the highest blast-radius PASS-bluff in the suite,
// since it certified the SCORING ENGINE worked when no scoring engine had executed. Real
// scoring now runs when a Verifier is constructed via NewVerifierWithProber(db, prober)
// (commit 28e6625a, C5); this unit test intentionally constructs an unconfigured Verifier
// (no database, no probe engine) so the only honest assertion remains the (renamed)
// ErrVerifierNotConfigured sentinel — §11.4.114/§11.4.120 reconciliation, 2026-07-08.
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

	require.Error(t, err)
	assert.True(t, errors.Is(err, verification.ErrVerifierNotConfigured),
		"expected ErrVerifierNotConfigured sentinel, got: %v", err)
	assert.Nil(t, result, "sentinel-return contract: nil result alongside sentinel error")
}

// TestScoringSystem_GetScoreExplanation tests score explanation retrieval
//
// Round-17 converted VerifyModel to sentinel-return per §11.4/CONST-036/037 anti-bluff fix.
// Round-84 tightened this test from asserting-fabricated-values to asserting-sentinel-error.
// The previous body asserted result.ScoreDetails contained "performance" — that string was
// hardcoded into the fabricated return, not generated by any real explanation engine.
// §11.4.114/§11.4.120 reconciliation (2026-07-08, commit 28e6625a): sentinel renamed
// ErrVerificationNotWired -> ErrVerifierNotConfigured; this unconfigured-Verifier case
// still surfaces it rather than fabricating a result. Once a wired Verifier's real
// explanation structure is exercised end-to-end, this test class re-tightens accordingly.
func TestScoringSystem_GetScoreExplanation(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier)

	ctx := context.Background()
	result, err := verifier.Verify(ctx, &verification.Request{
		ModelID: "explanation-test-model",
		Prompt:  "Get score explanation test",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, verification.ErrVerifierNotConfigured),
		"expected ErrVerifierNotConfigured sentinel, got: %v", err)
	assert.Nil(t, result, "sentinel-return contract: nil result alongside sentinel error")
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
//
// Round-17 converted VerifyModel to sentinel-return per §11.4/CONST-036/037 anti-bluff fix.
// Round-84 tightened this test from asserting-fabricated-values to asserting-sentinel-error.
// The previous body asserted that after a validation-rejection (nil request), a well-formed
// follow-up call returned NoError + "completed" status — but "completed" was a hardcoded
// fabrication. §11.4.114/§11.4.120 reconciliation (2026-07-08, commit 28e6625a): the sentinel
// was renamed ErrVerificationNotWired -> ErrVerifierNotConfigured; the real recovery semantics
// today: validation-rejection returns a different error than the (renamed) sentinel;
// well-formed input against an unconfigured Verifier returns the sentinel. Both behaviors
// are deterministic post-error, which IS what "recovery" means at this layer.
func TestErrorHandling_Recovery(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier)

	ctx := context.Background()

	// Test rejection from invalid input (validation-layer error, NOT the sentinel)
	_, err := verifier.Verify(ctx, nil)
	require.Error(t, err)
	assert.False(t, errors.Is(err, verification.ErrVerifierNotConfigured),
		"nil-request should be rejected at validation gate, not by sentinel")

	// Verify the verifier is still functional after error — well-formed input
	// deterministically yields the sentinel, proving the verifier state survived.
	result, err := verifier.Verify(ctx, &verification.Request{
		ModelID: "recovery-test",
		Prompt:  "Test recovery after error",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, verification.ErrVerifierNotConfigured),
		"expected ErrVerifierNotConfigured sentinel post-recovery, got: %v", err)
	assert.Nil(t, result, "sentinel-return contract: nil result alongside sentinel error")
}

// TestConcurrentVerification tests concurrent verification requests
//
// Round-17 converted VerifyModel to sentinel-return per §11.4/CONST-036/037 anti-bluff fix.
// Round-84 tightened this test from asserting-fabricated-values to asserting-sentinel-error.
// The previous body launched 10 goroutines and asserted ALL returned NoError + "completed"
// — that "succeeded" only because the bluff was data-race-free by virtue of returning
// constant fabricated values. §11.4.114/§11.4.120 reconciliation (2026-07-08, commit
// 28e6625a): the sentinel was renamed ErrVerificationNotWired -> ErrVerifierNotConfigured.
// The honest concurrency assertion today: every concurrent call against an unconfigured
// Verifier deterministically yields the (renamed) sentinel, proving the package-level
// sentinel value is safe to read concurrently and the verifier struct survives parallel
// invocation without state corruption. The "channel-named-errors-shadows-imported-errors-pkg"
// hazard is avoided by naming the local channel errCh.
func TestConcurrentVerification(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier)

	ctx := context.Background()
	numRequests := 10
	resultsCh := make(chan *verification.Result, numRequests)
	errCh := make(chan error, numRequests)

	// Launch concurrent verification requests
	for i := 0; i < numRequests; i++ {
		go func(id int) {
			req := &verification.Request{
				ModelID: "concurrent-test-model",
				Prompt:  "Concurrent verification test",
			}
			result, err := verifier.Verify(ctx, req)
			if err != nil {
				errCh <- err
			} else {
				resultsCh <- result
			}
		}(i)
	}

	// Collect results — every goroutine MUST return the sentinel error.
	sentinelCount := 0
	unexpectedCount := 0

	for i := 0; i < numRequests; i++ {
		select {
		case result := <-resultsCh:
			t.Errorf("expected sentinel error from every concurrent call, got non-nil result: %+v", result)
			unexpectedCount++
		case err := <-errCh:
			assert.True(t, errors.Is(err, verification.ErrVerifierNotConfigured),
				"expected ErrVerifierNotConfigured sentinel from concurrent call, got: %v", err)
			sentinelCount++
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for concurrent results")
		}
	}

	assert.Equal(t, numRequests, sentinelCount, "every concurrent call must return the sentinel")
	assert.Equal(t, 0, unexpectedCount, "no concurrent call should return a non-nil result")
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
