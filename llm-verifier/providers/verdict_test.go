package providers

// verdict_test.go — PWU-2 unit tests for the wired ProbeOutcome result
// (§11.4.115: these tests did not compile before this PWU -- verdict.go,
// ProbeOutcome, ProbeVerdict, ClassifyHTTPProbeOutcome, ClassifyProbeError,
// and ErrClaudeCodeQuotaExceeded did not exist; captured structural-RED
// evidence: qa-results/llmsverifier_pwu2/red_structural_gap.log, an empty
// grep for those symbols against the pre-PWU-2 tree).

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- Class 1: HTTP 429 (real captured Fair-Usage fixture) -----------------

func TestClassifyHTTPProbeOutcome_429_RealCapturedFairUsageFixture(t *testing.T) {
	// Real captured 429 body (scripts/multitrack/test_provider_autorotate.sh
	// FIX_429 fixture, itself captured live from a zai-coding-plan route,
	// 2026-07-xx).
	body := []byte(`{"error":{"code":"1313","message":"Your account's current usage pattern does not comply with the Fair Usage Policy, and your request frequency has been limited. For details, please refer to the Subscription Service Agreement. To restore access, please submit a request."}}`)
	header := http.Header{}
	header.Set("Retry-After", "30")
	resp := &http.Response{StatusCode: http.StatusTooManyRequests, Header: header}

	outcome := ClassifyHTTPProbeOutcome("zai-coding-plan", resp, body)

	assert.Equal(t, ProbeVerdictRateLimited, outcome.Verdict,
		"HTTP 429 (real captured Fair-Usage fixture) MUST classify as rate_limited, not opaque failed")
	assert.Equal(t, 30, outcome.RetryAfterSeconds,
		"Retry-After header MUST be surfaced as retry_after_seconds")
	assert.NotEmpty(t, outcome.Detail)
}

// --- Class 2: HTTP 402 (real captured qa-results fixture) ------------------

func TestClassifyHTTPProbeOutcome_402_RealCapturedQuotaFixture(t *testing.T) {
	// Real captured 402 body (qa-results/multitrack/logs/
	// T4_iter2_20260706T223846Z.log, this repo, 2026-07-06).
	body := []byte(`{"detail":"Subscription usage cap exceeded. Please add balance to continue."}`)
	resp := &http.Response{StatusCode: http.StatusPaymentRequired, Header: http.Header{}}

	outcome := ClassifyHTTPProbeOutcome("chutes", resp, body)

	assert.Equal(t, ProbeVerdictQuotaExceeded, outcome.Verdict,
		"HTTP 402 (real captured qa-results fixture) MUST classify as quota_exceeded, not opaque failed")
	assert.Equal(t, 0, outcome.RetryAfterSeconds,
		"a quota condition carries no short-retry budget")
}

func TestClassifyHTTPProbeOutcome_NilResponse_IsFailedNotPanic(t *testing.T) {
	outcome := ClassifyHTTPProbeOutcome("openai", nil, nil)
	assert.Equal(t, ProbeVerdictFailed, outcome.Verdict)
}

func TestClassifyHTTPProbeOutcome_ServerError_IsFailedNotRateLimitedOrQuota(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusInternalServerError, Header: http.Header{}}
	outcome := ClassifyHTTPProbeOutcome("openai", resp, []byte(`{}`))
	assert.Equal(t, ProbeVerdictFailed, outcome.Verdict,
		"a 500 is neither rate_limited nor quota_exceeded -- must not be misclassified")
}

// --- Class 3: native Claude-Code-CLI-bridge (quota vs rate-limit) ---------

func TestClassifyProbeError_ClaudeCodeQuotaExceeded(t *testing.T) {
	err := fmt.Errorf("%w: subtype=error_max_turns terminal_reason=quota", ErrClaudeCodeQuotaExceeded)
	outcome := ClassifyProbeError(err)
	assert.Equal(t, ProbeVerdictQuotaExceeded, outcome.Verdict)
}

func TestClassifyProbeError_ClaudeCodeRateLimited(t *testing.T) {
	err := fmt.Errorf("%w: 429 too many requests", ErrClaudeCodeRateLimited)
	outcome := ClassifyProbeError(err)
	assert.Equal(t, ProbeVerdictRateLimited, outcome.Verdict)
}

func TestClassifyProbeError_NilIsOK(t *testing.T) {
	outcome := ClassifyProbeError(nil)
	assert.Equal(t, ProbeVerdictOK, outcome.Verdict)
}

func TestClassifyProbeError_GenericErrorIsFailed(t *testing.T) {
	outcome := ClassifyProbeError(errors.New("some unrelated failure"))
	assert.Equal(t, ProbeVerdictFailed, outcome.Verdict)
}
