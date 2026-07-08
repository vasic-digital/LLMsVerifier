package unit

import (
	"context"
	"errors"
	"testing"
	"time"

	"digital.vasic.llmsverifier/verification"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Anti-bluff note (round-323 / HXV-001 — re-keyed from a §11.4 bluff test;
// §11.4.114/§11.4.120 reconciled 2026-07-08 against commit 28e6625a).
//
// These tests previously asserted that verification.Verify() returned a
// VerificationResult with EVERY capability flag true and EVERY score 8.5
// for any model — i.e. they only passed because production fabricated
// success. Round-17 (commit a6328629) correctly removed that
// hardcoded-all-true bluff from verification/verification.go and made
// Verify() return ErrVerificationNotWired until a real
// llmverifier.Verifier was plumbed in.
//
// Commit 28e6625a (C5, §11.4.115 RED->GREEN) wired that real dispatch:
// NewVerifierWithProber(db, prober) now resolves the model, dispatches the
// C4 per-capability probes, composes a real database.VerificationResult, and
// persists it. The unwired-sentinel ErrVerificationNotWired was RENAMED (not
// re-designed) to ErrVerifierNotConfigured, returned whenever a Verifier is
// constructed WITHOUT both a database AND a probe engine — exactly the case
// these unit tests exercise via NewModelVerifier(nil)/NewVerifier(nil). This
// file is reconciled per §11.4.120 (fix-breaks-its-own-gate: the `go vet`
// failure here was the correct signal 28e6625a landed, not a regression) +
// §11.4.114 (isolated against the last-known-good commit, 28e6625a itself,
// whose own message shows the sibling verification/verification_test.go was
// reconciled in the same commit but this tests/unit copy was missed).
//
// That correct production fix left these unit tests asserting the *old
// bluff behaviour*, so they began failing — discovered as HXV-001. The
// honest contract these unit tests now certify is:
//
//  1. Input validation (nil request / empty ModelID / empty Prompt) is
//     enforced BEFORE any dispatch and returns a descriptive error with
//     a nil result.
//  2. A well-formed request, against a Verifier constructed WITHOUT a
//     database + probe engine (the case in a unit test — no real infra),
//     returns ErrVerifierNotConfigured loudly rather than fabricating a
//     result.
//
// Real end-to-end verification against live provider endpoints is
// exercised by the integration suite in ./llmverifier/ (TestVerifier_*),
// which uses real config + real HTTP — not by these unit tests.

// TestModelVerification_ValidRequest_NotConfigured proves a well-formed
// request returns the loud ErrVerifierNotConfigured sentinel — never a
// fabricated result — when the Verifier has no database + probe engine.
func TestModelVerification_ValidRequest_NotConfigured(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier, "verifier should not be nil")

	ctx := context.Background()
	req := &verification.Request{
		ModelID: "gpt-4",
		Prompt:  "Test prompt for verification",
	}

	result, err := verifier.Verify(ctx, req)
	require.Error(t, err, "unconfigured verification must surface the gap loudly")
	require.Nil(t, result, "no fabricated result may be returned when unconfigured")
	assert.ErrorIs(t, err, verification.ErrVerifierNotConfigured,
		"error must be the ErrVerifierNotConfigured sentinel")
}

// TestModelVerification_InvalidModel tests verification with invalid model ID.
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
	assert.NotErrorIs(t, err, verification.ErrVerifierNotConfigured,
		"validation error must be distinct from the not-configured sentinel")
}

// TestModelVerification_Timeout tests that input validation and the
// not-configured dispatch both honour a deadline-bounded context.
func TestModelVerification_Timeout(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier, "verifier should not be nil")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req := &verification.Request{
		ModelID: "gpt-4",
		Prompt:  "Test prompt",
	}

	// Verify returns synchronously (validation + sentinel) so it
	// completes well within the deadline; it must still surface the
	// not-configured sentinel rather than a fabricated result.
	result, err := verifier.Verify(ctx, req)
	require.Error(t, err)
	require.Nil(t, result)
	assert.ErrorIs(t, err, verification.ErrVerifierNotConfigured)
}

// TestModelVerification_EdgeCases tests various edge cases of the
// validation + not-configured-dispatch contract.
func TestModelVerification_EdgeCases(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier)

	testCases := []struct {
		name string
		req  *verification.Request
		// errMsg is the substring expected in a *validation* error.
		// When empty, the request is well-formed and the call must
		// return the ErrVerifierNotConfigured sentinel instead.
		errMsg string
	}{
		{
			name:   "nil request",
			req:    nil,
			errMsg: "cannot be nil",
		},
		{
			name:   "empty prompt",
			req:    &verification.Request{ModelID: "gpt-4", Prompt: ""},
			errMsg: "prompt is required",
		},
		{
			name:   "empty model ID",
			req:    &verification.Request{ModelID: "", Prompt: "test"},
			errMsg: "model ID is required",
		},
		{
			name:   "valid request with special characters",
			req:    &verification.Request{ModelID: "model-v2-beta", Prompt: "Test with special chars: @#$%"},
			errMsg: "",
		},
		{
			name:   "very long prompt",
			req:    &verification.Request{ModelID: "gpt-4", Prompt: string(make([]byte, 10000))},
			errMsg: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := verifier.Verify(context.Background(), tc.req)
			require.Error(t, err)
			require.Nil(t, result, "no result may be returned on the error path")
			if tc.errMsg != "" {
				assert.Contains(t, err.Error(), tc.errMsg)
				assert.NotErrorIs(t, err, verification.ErrVerifierNotConfigured,
					"validation error must be distinct from not-configured sentinel")
			} else {
				assert.ErrorIs(t, err, verification.ErrVerifierNotConfigured,
					"well-formed-but-unconfigured request must surface the sentinel")
			}
		})
	}
}

// TestVerification_NotConfiguredSentinel_IsStable proves the sentinel error
// is a stable, identifiable value (so callers can branch on it) and that it
// names the missing configuration honestly — the anti-bluff guarantee.
func TestVerification_NotConfiguredSentinel_IsStable(t *testing.T) {
	require.NotNil(t, verification.ErrVerifierNotConfigured)
	msg := verification.ErrVerifierNotConfigured.Error()
	assert.Contains(t, msg, "not configured",
		"sentinel must honestly state the configuration gap")
	assert.Contains(t, msg, "PASS-bluff",
		"sentinel must reference the anti-bluff rationale")

	verifier := verification.NewModelVerifier(nil)
	_, err := verifier.Verify(context.Background(),
		&verification.Request{ModelID: "gpt-4", Prompt: "stable check"})
	assert.True(t, errors.Is(err, verification.ErrVerifierNotConfigured),
		"Verify must return the same sentinel instance on every call")
}

// TestConfiguration_Validation tests verifier construction.
func TestConfiguration_Validation(t *testing.T) {
	// Test verifier creation with nil database (construction must
	// succeed; the gap surfaces only at Verify() time).
	verifier := verification.NewModelVerifier(nil)
	assert.NotNil(t, verifier, "verifier should be created even with nil db")

	// NewVerifier should also work
	verifier2 := verification.NewVerifier(nil)
	assert.NotNil(t, verifier2, "NewVerifier should also work")
}

// TestErrorHandling_Recovery proves the verifier is stateless across
// calls: a validation error on one call does not corrupt the contract of
// the next call.
func TestErrorHandling_Recovery(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier)

	ctx := context.Background()

	// First request fails validation.
	r1, err1 := verifier.Verify(ctx, &verification.Request{ModelID: "", Prompt: "test"})
	require.Error(t, err1)
	require.Nil(t, r1)
	assert.Contains(t, err1.Error(), "model ID is required")

	// Second well-formed request still returns the not-configured sentinel
	// (no state leaked from the prior validation failure).
	r2, err2 := verifier.Verify(ctx, &verification.Request{ModelID: "gpt-4", Prompt: "test"})
	require.Error(t, err2)
	require.Nil(t, r2)
	assert.ErrorIs(t, err2, verification.ErrVerifierNotConfigured)
}

// TestConcurrentVerification proves the verifier contract is race-free:
// concurrent callers all observe the same deterministic outcome.
func TestConcurrentVerification(t *testing.T) {
	verifier := verification.NewModelVerifier(nil)
	require.NotNil(t, verifier)

	const numGoroutines = 10
	errCh := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			ctx := context.Background()
			req := &verification.Request{
				ModelID: "gpt-4",
				Prompt:  "Concurrent test",
			}
			_, err := verifier.Verify(ctx, req)
			errCh <- err
		}()
	}

	notConfiguredCount := 0
	for i := 0; i < numGoroutines; i++ {
		select {
		case err := <-errCh:
			require.Error(t, err, "every concurrent call must return an error")
			if errors.Is(err, verification.ErrVerifierNotConfigured) {
				notConfiguredCount++
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for goroutines")
		}
	}

	assert.Equal(t, numGoroutines, notConfiguredCount,
		"all concurrent well-formed requests must return ErrVerifierNotConfigured")
}
