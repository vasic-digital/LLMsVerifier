package verification

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVerifier(t *testing.T) {
	v := NewVerifier(nil)

	assert.NotNil(t, v)
	assert.Nil(t, v.db)
}

func TestVerifier_Verify_NilRequest(t *testing.T) {
	v := NewVerifier(nil)

	result, err := v.Verify(context.Background(), nil)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestVerifier_Verify_EmptyModelID(t *testing.T) {
	v := NewVerifier(nil)
	req := &Request{
		ModelID: "",
		Prompt:  "test prompt",
	}

	result, err := v.Verify(context.Background(), req)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "model ID is required")
}

func TestVerifier_Verify_EmptyPrompt(t *testing.T) {
	v := NewVerifier(nil)
	req := &Request{
		ModelID: "test-model",
		Prompt:  "",
	}

	result, err := v.Verify(context.Background(), req)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "prompt is required")
}

func TestVerifier_Verify_Success(t *testing.T) {
	v := NewVerifier(nil)
	req := &Request{
		ModelID: "test-model",
		Prompt:  "test prompt",
	}

	result, err := v.Verify(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify result fields
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, "model_verification", result.VerificationType)
	assert.NotZero(t, result.ID)
	assert.NotNil(t, result.StartedAt)
	assert.NotNil(t, result.CompletedAt)

	// Verify capability flags
	assert.True(t, result.SupportsToolUse)
	assert.True(t, result.SupportsFunctionCalling)
	assert.True(t, result.SupportsCodeGeneration)
	assert.True(t, result.SupportsStreaming)
	assert.True(t, result.SupportsJSONMode)

	// Verify scores
	assert.Greater(t, result.OverallScore, float64(0))
	assert.Greater(t, result.CodeCapabilityScore, float64(0))
	assert.Greater(t, result.ResponsivenessScore, float64(0))
}

func TestVerifier_Verify_ResultScores(t *testing.T) {
	v := NewVerifier(nil)
	req := &Request{
		ModelID: "gpt-4",
		Prompt:  "What is the meaning of life?",
	}

	result, err := v.Verify(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, result)

	// All scores should be valid (0-10 range)
	assert.GreaterOrEqual(t, result.OverallScore, float64(0))
	assert.LessOrEqual(t, result.OverallScore, float64(10))

	assert.GreaterOrEqual(t, result.CodeCapabilityScore, float64(0))
	assert.LessOrEqual(t, result.CodeCapabilityScore, float64(10))

	assert.GreaterOrEqual(t, result.ReliabilityScore, float64(0))
	assert.LessOrEqual(t, result.ReliabilityScore, float64(10))
}

func TestVerifier_Verify_LatencyMetrics(t *testing.T) {
	v := NewVerifier(nil)
	req := &Request{
		ModelID: "claude-3",
		Prompt:  "Explain quantum computing",
	}

	result, err := v.Verify(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Latency metrics should be populated
	assert.NotNil(t, result.LatencyMs)
	assert.Greater(t, result.AvgLatencyMs, 0)
	assert.Greater(t, result.P95LatencyMs, 0)
	assert.GreaterOrEqual(t, result.MaxLatencyMs, result.MinLatencyMs)
	assert.GreaterOrEqual(t, result.P95LatencyMs, result.AvgLatencyMs)
}

func TestVerifier_Verify_CodeLanguageSupport(t *testing.T) {
	v := NewVerifier(nil)
	req := &Request{
		ModelID: "codex",
		Prompt:  "Write a function",
	}

	result, err := v.Verify(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Should support multiple languages
	assert.NotEmpty(t, result.CodeLanguageSupport)
	assert.Contains(t, result.CodeLanguageSupport, "python")
	assert.Contains(t, result.CodeLanguageSupport, "go")
	assert.Contains(t, result.CodeLanguageSupport, "javascript")
}

func TestVerifier_Verify_CodeCapabilities(t *testing.T) {
	v := NewVerifier(nil)
	req := &Request{
		ModelID: "code-model",
		Prompt:  "Debug this code",
	}

	result, err := v.Verify(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Code-related capabilities
	assert.True(t, result.CodeDebugging)
	assert.True(t, result.CodeOptimization)
	assert.True(t, result.TestGeneration)
	assert.True(t, result.DocumentationGeneration)
	assert.True(t, result.Refactoring)
	assert.True(t, result.ErrorResolution)
}

func TestVerifier_Verify_ModelStatusFlags(t *testing.T) {
	v := NewVerifier(nil)
	req := &Request{
		ModelID: "active-model",
		Prompt:  "Hello",
	}

	result, err := v.Verify(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Model should exist and be responsive
	assert.NotNil(t, result.ModelExists)
	assert.True(t, *result.ModelExists)

	assert.NotNil(t, result.Responsive)
	assert.True(t, *result.Responsive)

	assert.NotNil(t, result.Overloaded)
	assert.False(t, *result.Overloaded)
}

func TestRequest_Struct(t *testing.T) {
	req := Request{
		ModelID: "test-model-id",
		Prompt:  "test prompt content",
	}

	assert.Equal(t, "test-model-id", req.ModelID)
	assert.Equal(t, "test prompt content", req.Prompt)
}

func TestBoolPtr(t *testing.T) {
	truePtr := boolPtr(true)
	falsePtr := boolPtr(false)

	assert.NotNil(t, truePtr)
	assert.True(t, *truePtr)

	assert.NotNil(t, falsePtr)
	assert.False(t, *falsePtr)
}

func TestIntPtr(t *testing.T) {
	ptr42 := intPtr(42)
	ptr0 := intPtr(0)
	ptrNeg := intPtr(-100)

	assert.NotNil(t, ptr42)
	assert.Equal(t, 42, *ptr42)

	assert.NotNil(t, ptr0)
	assert.Equal(t, 0, *ptr0)

	assert.NotNil(t, ptrNeg)
	assert.Equal(t, -100, *ptrNeg)
}

func TestVerifier_Verify_ContextCancellation(t *testing.T) {
	v := NewVerifier(nil)
	req := &Request{
		ModelID: "test-model",
		Prompt:  "test prompt",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Should still work since current implementation doesn't check context
	result, err := v.Verify(ctx, req)

	// Note: Current implementation doesn't actually check context
	// This test documents expected behavior
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestVerifier_Verify_MultipleRequests(t *testing.T) {
	v := NewVerifier(nil)

	requests := []Request{
		{ModelID: "model-1", Prompt: "prompt 1"},
		{ModelID: "model-2", Prompt: "prompt 2"},
		{ModelID: "model-3", Prompt: "prompt 3"},
	}

	for _, req := range requests {
		result, err := v.Verify(context.Background(), &req)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "completed", result.Status)
	}
}
