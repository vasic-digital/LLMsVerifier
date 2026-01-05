package unit

import (
	"context"
	"testing"
	"time"

	"llm-verifier/verification"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModelVerification_ValidModel tests verification of a valid model
func TestModelVerification_ValidModel(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier, "verifier should not be nil")

	ctx := context.Background()
	req := &verification.Request{
		ModelID: "gpt-4",
		Prompt:  "Test prompt for verification",
	}

	result, err := verifier.Verify(ctx, req)
	require.NoError(t, err, "verification should succeed for valid model")
	require.NotNil(t, result, "result should not be nil")

	assert.Equal(t, "completed", result.Status, "status should be completed")
	assert.NotNil(t, result.ModelExists, "ModelExists should not be nil")
	assert.True(t, *result.ModelExists, "model should exist")
}

// TestModelVerification_InvalidModel tests verification with invalid model ID
func TestModelVerification_InvalidModel(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier, "verifier should not be nil")

	ctx := context.Background()
	req := &verification.Request{
		ModelID: "",
		Prompt:  "Test prompt",
	}

	result, err := verifier.Verify(ctx, req)
	assert.Error(t, err, "should error for empty model ID")
	assert.Nil(t, result, "result should be nil on error")
	assert.Contains(t, err.Error(), "model ID is required")
}

// TestModelVerification_Timeout tests verification timeout handling
func TestModelVerification_Timeout(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier, "verifier should not be nil")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req := &verification.Request{
		ModelID: "gpt-4",
		Prompt:  "Test prompt",
	}

	// Verify should complete before timeout for this implementation
	result, err := verifier.Verify(ctx, req)
	require.NoError(t, err, "verification should complete before timeout")
	require.NotNil(t, result, "result should not be nil")
}

// TestModelVerification_EdgeCases tests various edge cases
func TestModelVerification_EdgeCases(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier)

	testCases := []struct {
		name      string
		req       *verification.Request
		expectErr bool
		errMsg    string
	}{
		{
			name:      "nil request",
			req:       nil,
			expectErr: true,
			errMsg:    "cannot be nil",
		},
		{
			name:      "empty prompt",
			req:       &verification.Request{ModelID: "gpt-4", Prompt: ""},
			expectErr: true,
			errMsg:    "prompt is required",
		},
		{
			name:      "empty model ID",
			req:       &verification.Request{ModelID: "", Prompt: "test"},
			expectErr: true,
			errMsg:    "model ID is required",
		},
		{
			name:      "valid request with special characters",
			req:       &verification.Request{ModelID: "model-v2-beta", Prompt: "Test with special chars: @#$%"},
			expectErr: false,
		},
		{
			name:      "very long prompt",
			req:       &verification.Request{ModelID: "gpt-4", Prompt: string(make([]byte, 10000))},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := verifier.Verify(context.Background(), tc.req)
			if tc.expectErr {
				assert.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// TestScoringSystem_CalculateScore tests the scoring system
func TestScoringSystem_CalculateScore(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier)

	ctx := context.Background()
	req := &verification.Request{
		ModelID: "gpt-4",
		Prompt:  "Test scoring",
	}

	result, err := verifier.Verify(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Validate score fields
	assert.GreaterOrEqual(t, result.OverallScore, 0.0, "overall score should be >= 0")
	assert.LessOrEqual(t, result.OverallScore, 10.0, "overall score should be <= 10")
	assert.GreaterOrEqual(t, result.CodeQualityScore, 0.0)
	assert.GreaterOrEqual(t, result.LogicCorrectnessScore, 0.0)
	assert.GreaterOrEqual(t, result.RuntimeEfficiencyScore, 0.0)
}

// TestScoringSystem_GetScoreExplanation tests score explanation generation
func TestScoringSystem_GetScoreExplanation(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier)

	ctx := context.Background()
	req := &verification.Request{
		ModelID: "gpt-4",
		Prompt:  "Test explanation",
	}

	result, err := verifier.Verify(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check that score details are provided
	assert.NotEmpty(t, result.ScoreDetails, "score details should not be empty")
}

// TestHTTPClient_ProviderRequests tests HTTP client behavior
func TestHTTPClient_ProviderRequests(t *testing.T) {
	// Test that the verifier properly handles requests
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier)

	// Multiple sequential requests should work
	for i := 0; i < 5; i++ {
		ctx := context.Background()
		req := &verification.Request{
			ModelID: "gpt-4",
			Prompt:  "Request " + string(rune('0'+i)),
		}

		result, err := verifier.Verify(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)
	}
}

// TestConfiguration_Validation tests configuration validation
func TestConfiguration_Validation(t *testing.T) {
	// Test verifier creation with nil database (should work for basic cases)
	verifier := verification.NewModelVerifier(nil)
	assert.NotNil(t, verifier, "verifier should be created even with nil db")

	// NewVerifier should also work
	verifier2 := verification.NewVerifier(nil)
	assert.NotNil(t, verifier2, "NewVerifier should also work")
}

// TestErrorHandling_Recovery tests error handling and recovery
func TestErrorHandling_Recovery(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier)

	// After an error, subsequent requests should still work
	ctx := context.Background()

	// First request with error
	_, err := verifier.Verify(ctx, &verification.Request{ModelID: "", Prompt: "test"})
	assert.Error(t, err)

	// Second request should succeed
	result, err := verifier.Verify(ctx, &verification.Request{ModelID: "gpt-4", Prompt: "test"})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// TestConcurrentVerification tests concurrent verification requests
func TestConcurrentVerification(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier)

	const numGoroutines = 10
	results := make(chan *verification.Result, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			ctx := context.Background()
			req := &verification.Request{
				ModelID: "gpt-4",
				Prompt:  "Concurrent test",
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

	for i := 0; i < numGoroutines; i++ {
		select {
		case <-results:
			successCount++
		case <-errors:
			errorCount++
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for goroutines")
		}
	}

	assert.Equal(t, numGoroutines, successCount, "all requests should succeed")
	assert.Equal(t, 0, errorCount, "no errors should occur")
}

// TestVerificationResult_Fields tests that all result fields are populated
func TestVerificationResult_Fields(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier)

	ctx := context.Background()
	req := &verification.Request{
		ModelID: "gpt-4",
		Prompt:  "Test all fields",
	}

	result, err := verifier.Verify(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check capability fields
	assert.True(t, result.SupportsToolUse, "should support tool use")
	assert.True(t, result.SupportsFunctionCalling, "should support function calling")
	assert.True(t, result.SupportsCodeGeneration, "should support code generation")
	assert.True(t, result.SupportsStreaming, "should support streaming")
	assert.True(t, result.SupportsJSONMode, "should support JSON mode")

	// Check latency fields
	assert.NotNil(t, result.LatencyMs, "latency should not be nil")
	assert.Greater(t, int(result.AvgLatencyMs), 0, "avg latency should be positive")

	// Check code language support
	assert.NotEmpty(t, result.CodeLanguageSupport, "should support some languages")
}
