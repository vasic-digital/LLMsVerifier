package verification

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVerifyModelCodeVisibility_DenialMustNotInflateScore is the §11.4.115
// RED-on-broken-artifact regression guard for a VerificationScore-inflation
// PASS-bluff in VerifyModelCodeVisibility (verification/code_verification.go).
//
// ROOT CAUSE (independent-audit finding, 2026-07-10): the unconditional
//
//	result.VerificationScore = max(result.ResponseAnalysis.ConfidenceScore, 0.7) // Minimum 0.7 score
//
// floors VerificationScore at 0.7 for EVERY model, regardless of whether the
// model demonstrated ANY code visibility. A model that DENIES seeing the code
// on every sample ("No, I cannot see your code") correctly earns
// ConfidenceScore == 0.0 and CodeVisibility == false and Status == "failed"
// (the sibling guard in code_visibility_status_bluff_red_test.go already
// covers the Status axis) -- but the SAME code-blind model's persisted
// VerificationScore is forced up to 0.7, a "mostly confident" score.
//
// This is not merely cosmetic: CodeVerificationIntegration.storeVerificationResult
// (verification/code_verification_integration.go:275-282) propagates this
// VerificationScore, UNCONDITIONALLY and WITHOUT checking Status or
// CodeVisibility, into the persisted database.VerificationResult's
// OverallScore / CodeQualityScore / CodeCapabilityScore (each
// `result.VerificationScore * 10`). A code-blind, verification-FAILED model
// is therefore written to the database with OverallScore=7.0 (out of 10) --
// comfortably inside this codebase's own "decent tier" bands (e.g.
// scoring/database_integration.go's `case score >= 7.0`) -- exactly the
// "verifier that cannot actually catch a bad input" failure mode: the
// boolean gate (Status/CodeVisibility) catches the denial, but the numeric
// score the rest of the system ranks/filters/exports by does not.
//
// CONTRACT: a model whose code visibility was NOT established
// (CodeVisibility == false, ConfidenceScore == 0.0) MUST NOT receive a
// VerificationScore that is HIGHER than its own measured ConfidenceScore --
// the floor must never fire for a model that demonstrated zero visibility.
func TestVerifyModelCodeVisibility_DenialMustNotInflateScore(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": "No, I cannot see your code. Please paste the code again.",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	logger := createTestLogger()
	httpClient := createTestHTTPClient()
	cvs := NewCodeVerificationService(httpClient, logger)

	mockProvider := NewMockProviderClient(server.URL, "test-api-key", server.Client())

	result, err := cvs.VerifyModelCodeVisibility(context.Background(), "blind-model", "test-provider", mockProvider)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Precondition: the model really did demonstrate zero code visibility.
	require.False(t, result.CodeVisibility, "precondition: a denying model must have CodeVisibility=false")
	require.Equal(t, 0.0, result.ResponseAnalysis.ConfidenceScore,
		"precondition: a denying model's measured ConfidenceScore must be 0.0")

	// The bug: VerificationScore is floored to 0.7 even though ConfidenceScore is 0.0.
	assert.LessOrEqual(t, result.VerificationScore, result.ResponseAnalysis.ConfidenceScore+1e-9,
		"SCORE-FLOOR BLUFF: VerificationScore (%.2f) must not exceed the model's own measured "+
			"ConfidenceScore (%.2f) when code visibility was never established -- an inflated score "+
			"for a code-blind model corrupts every downstream ranking/filtering/export that reads "+
			"VerificationScore without re-checking Status/CodeVisibility",
		result.VerificationScore, result.ResponseAnalysis.ConfidenceScore)
}
