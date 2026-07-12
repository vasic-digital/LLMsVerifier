package providers

// errors_402_quota_red_test.go — PWU-2 §11.4.115 RED-baseline test.
//
// This file exercises ONLY the pre-existing providers/errors.go
// ClassifyError (no new symbols from PWU-2's fix) against a REAL captured
// HTTP 402 payload (§11.4.6 no-guessing — not a synthetic/invented body):
//
//	qa-results/multitrack/logs/T4_iter2_20260706T223846Z.log:
//	"API Error: 402 Error from provider(chutes,zai-org/GLM-5.2-TEE: 402):
//	 {"detail":"Subscription usage cap exceeded. Please add balance to
//	 continue."}"
//
// BEFORE the PWU-2 fix (providers/errors.go has no 402 case in any
// per-provider classifier nor in ClassifyError's top-level dispatch),
// this test FAILS: ClassifyError returns Code=UNKNOWN_ERROR /
// Type=ErrorTypeUnknown for a 402 response — the exact "opaque failed"
// collapse the incorporation plan
// (docs/research/llmsverifier_incorporation_20260707/ANALYSIS_AND_PLAN.md
// PART A.3 / PART C.3 / PART F PWU-2) identifies as the gap. Captured RED
// evidence: qa-results/llmsverifier_pwu2/red_402_classify_error.log.
//
// AFTER the fix this test PASSES unchanged — it is the GREEN regression
// guard for the same assertion (§11.4.115 polarity: one source, two
// roles). Captured GREEN evidence:
// qa-results/llmsverifier_pwu2/green_402_classify_error.log.

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyError_402QuotaGap_RealCapturedPayload(t *testing.T) {
	// "chutes" is the real provider name from the captured log line; the
	// provider-agnostic 402 override must fire for it exactly as for any
	// other provider name (402 is not part of chutes's own vocabulary —
	// it falls through classifyGenericError, which is what makes this a
	// provider-agnostic override rather than a per-provider addition).
	classifier := NewErrorClassifier("chutes")

	resp := &http.Response{StatusCode: http.StatusPaymentRequired, Header: http.Header{}}
	body := []byte(`{"detail":"Subscription usage cap exceeded. Please add balance to continue."}`)

	result := classifier.ClassifyError(resp, body)

	assert.Equal(t, "QUOTA_EXCEEDED", result.Code,
		"HTTP 402 (captured live, qa-results/multitrack/logs/T4_iter2_20260706T223846Z.log) MUST classify as QUOTA_EXCEEDED, not opaque UNKNOWN_ERROR")
	assert.Equal(t, ErrorTypeQuota, result.Type,
		"HTTP 402 MUST map to ErrorTypeQuota so a wired probe result can emit a distinct quota_exceeded verdict, never an opaque failed")
	assert.False(t, result.Retryable,
		"a subscription/usage-cap condition will not clear on a short retry — MUST NOT be marked Retryable")
}
