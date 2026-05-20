package verification

import (
	"context"
	"errors"
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

// TestVerifier_Verify_NotWiredContract certifies the honest round-17
// contract: a valid Request (non-nil, ModelID + Prompt populated) does
// NOT silently succeed with fabricated capabilities. The previous
// implementation returned a hardcoded VerificationResult with every
// capability flag true and every score 8.5 regardless of the model —
// a CONST-036/037 PASS-bluff at the single-source-of-truth layer.
// Until llmverifier.Verifier is plumbed into VerificationService, the
// entrypoint MUST surface the gap loudly via ErrVerificationNotWired.
// Closing this test by asserting fabricated success would re-introduce
// the bluff (HXV-002, round-348).
func TestVerifier_Verify_NotWiredContract(t *testing.T) {
	v := NewVerifier(nil)
	req := &Request{
		ModelID: "test-model",
		Prompt:  "test prompt",
	}

	result, err := v.Verify(context.Background(), req)

	require.Error(t, err, "valid request must surface the not-wired gap, never fabricate success")
	assert.Nil(t, result, "no VerificationResult may be returned while dispatch is un-wired")
	assert.True(t, errors.Is(err, ErrVerificationNotWired),
		"error must be the honest ErrVerificationNotWired sentinel, got: %v", err)
	assert.Contains(t, err.Error(), "PASS-bluff",
		"error message must name the removed bluff so the gap stays visible")
}

// TestVerifier_Verify_ResultScores certifies that the verifier never
// fabricates scores. The pre-honesty implementation returned an 8.5
// score for every dimension on every model; round-17 removed that
// fabrication. A valid request MUST return ErrVerificationNotWired
// rather than any score at all.
func TestVerifier_Verify_ResultScores(t *testing.T) {
	v := NewVerifier(nil)
	req := &Request{
		ModelID: "gpt-4",
		Prompt:  "What is the meaning of life?",
	}

	result, err := v.Verify(context.Background(), req)

	require.Error(t, err, "scores must never be fabricated; un-wired dispatch must error")
	assert.Nil(t, result, "no result (and therefore no score) may be returned un-wired")
	assert.True(t, errors.Is(err, ErrVerificationNotWired),
		"error must be ErrVerificationNotWired, got: %v", err)
}

// TestVerifier_Verify_LatencyMetrics certifies that latency metrics
// are never fabricated. The pre-honesty implementation populated
// AvgLatencyMs/P95LatencyMs with constant fabricated values without
// ever making a real API call. Un-wired dispatch MUST error.
func TestVerifier_Verify_LatencyMetrics(t *testing.T) {
	v := NewVerifier(nil)
	req := &Request{
		ModelID: "claude-3",
		Prompt:  "Explain quantum computing",
	}

	result, err := v.Verify(context.Background(), req)

	require.Error(t, err, "latency metrics must come from a real call, never be fabricated")
	assert.Nil(t, result, "no result (and therefore no latency metrics) may be returned un-wired")
	assert.True(t, errors.Is(err, ErrVerificationNotWired),
		"error must be ErrVerificationNotWired, got: %v", err)
}

// TestVerifier_Verify_CodeLanguageSupport certifies that
// per-language support flags are never fabricated. The pre-honesty
// implementation claimed python/go/javascript support for every
// model unconditionally. Un-wired dispatch MUST error.
func TestVerifier_Verify_CodeLanguageSupport(t *testing.T) {
	v := NewVerifier(nil)
	req := &Request{
		ModelID: "codex",
		Prompt:  "Write a function",
	}

	result, err := v.Verify(context.Background(), req)

	require.Error(t, err, "language support must be measured, never fabricated")
	assert.Nil(t, result, "no result (and therefore no language list) may be returned un-wired")
	assert.True(t, errors.Is(err, ErrVerificationNotWired),
		"error must be ErrVerificationNotWired, got: %v", err)
}

// TestVerifier_Verify_CodeCapabilities certifies that code-capability
// flags (debugging, optimization, test generation, etc.) are never
// fabricated. The pre-honesty implementation set every flag true for
// every model. Un-wired dispatch MUST error.
func TestVerifier_Verify_CodeCapabilities(t *testing.T) {
	v := NewVerifier(nil)
	req := &Request{
		ModelID: "code-model",
		Prompt:  "Debug this code",
	}

	result, err := v.Verify(context.Background(), req)

	require.Error(t, err, "code-capability flags must be tested, never fabricated all-true")
	assert.Nil(t, result, "no result (and therefore no capability flags) may be returned un-wired")
	assert.True(t, errors.Is(err, ErrVerificationNotWired),
		"error must be ErrVerificationNotWired, got: %v", err)
}

// TestVerifier_Verify_ModelStatusFlags certifies that model status
// flags (ModelExists, Responsive, Overloaded) are never fabricated.
// The pre-honesty implementation reported every model as existing and
// responsive without ever contacting the provider. Un-wired dispatch
// MUST error.
func TestVerifier_Verify_ModelStatusFlags(t *testing.T) {
	v := NewVerifier(nil)
	req := &Request{
		ModelID: "active-model",
		Prompt:  "Hello",
	}

	result, err := v.Verify(context.Background(), req)

	require.Error(t, err, "status flags must reflect a real probe, never be fabricated")
	assert.Nil(t, result, "no result (and therefore no status flags) may be returned un-wired")
	assert.True(t, errors.Is(err, ErrVerificationNotWired),
		"error must be ErrVerificationNotWired, got: %v", err)
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

// TestVerifier_Verify_ContextCancellation certifies that even with a
// cancelled context the verifier surfaces the honest not-wired gap
// rather than fabricating a completed result. (Once dispatch is
// wired, a cancelled context must produce a context error — never a
// fabricated success either way.)
func TestVerifier_Verify_ContextCancellation(t *testing.T) {
	v := NewVerifier(nil)
	req := &Request{
		ModelID: "test-model",
		Prompt:  "test prompt",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := v.Verify(ctx, req)

	require.Error(t, err, "un-wired dispatch must error regardless of context state")
	assert.Nil(t, result, "no fabricated result may be returned for a cancelled context")
	assert.True(t, errors.Is(err, ErrVerificationNotWired),
		"error must be ErrVerificationNotWired, got: %v", err)
}

// TestVerifier_Verify_MultipleRequests certifies that the honest
// not-wired contract holds across repeated invocations — no request
// in a batch silently succeeds with fabricated capabilities.
func TestVerifier_Verify_MultipleRequests(t *testing.T) {
	v := NewVerifier(nil)

	requests := []Request{
		{ModelID: "model-1", Prompt: "prompt 1"},
		{ModelID: "model-2", Prompt: "prompt 2"},
		{ModelID: "model-3", Prompt: "prompt 3"},
	}

	for _, req := range requests {
		req := req
		result, err := v.Verify(context.Background(), &req)

		require.Error(t, err, "every request must surface the not-wired gap, got nil err for %s", req.ModelID)
		assert.Nil(t, result, "no fabricated result may be returned for %s", req.ModelID)
		assert.True(t, errors.Is(err, ErrVerificationNotWired),
			"error must be ErrVerificationNotWired for %s, got: %v", req.ModelID, err)
	}
}
