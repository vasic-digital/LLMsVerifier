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

// TestVerifyModelCodeVisibility_DenialMustNotBeVerified is the §11.4.115
// RED-on-broken-artifact regression guard for the verification-gate PASS-bluff:
//
//	A model that explicitly DENIES code visibility on EVERY sample
//	("No, I cannot see your code") is the canonical negative case the
//	verification gate exists to catch. The "relaxed verification" status
//	logic in VerifyModelCodeVisibility unconditionally sets Status="verified"
//	for any model that returned ANY response (lines 149-158), so a code-blind
//	model is certified "verified". That verdict is consumed by the gate's
//	non-strict IsModelVerified (model_verification_service.go:296 returns true
//	on VerificationStatus != "error"), so a model that cannot see code passes
//	the verification gate — a §11.4 / §11.4.1 PASS-bluff in the source-of-truth
//	gate itself.
//
// CONTRACT: a model whose code visibility was NOT established
// (CodeVisibility == false) MUST NOT carry Status == "verified".
func TestVerifyModelCodeVisibility_DenialMustNotBeVerified(t *testing.T) {
	// Mock server: model denies seeing the code for EVERY sample/request.
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

	// The model demonstrably cannot see the code: this much already holds today.
	assert.False(t, result.CodeVisibility,
		"a model that denies code visibility on every sample must have CodeVisibility=false")
	assert.False(t, result.AffirmativeConfirmation,
		"a denying model must not register an affirmative confirmation")

	// The bug: Status is certified "verified" for a code-blind model.
	// A model whose code visibility was not established MUST NOT be "verified",
	// otherwise the gate's non-strict IsModelVerified treats it as usable.
	assert.NotEqual(t, "verified", result.Status,
		"GATE BLUFF: a model that cannot see the code must NOT be Status=verified "+
			"(non-strict IsModelVerified certifies any non-error status as verified)")
}
