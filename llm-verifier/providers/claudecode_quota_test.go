package providers

// claudecode_quota_test.go — PWU-2 unit tests for the bridge's
// quota-vs-rate-limit disambiguation (isQuotaSignal checked BEFORE
// isRateLimitSignal at every ChatCompletion error site; see
// providers/claudecode.go). Mocks/no-exec pure-function tests only, per
// §11.4.27 (unit tests exercise the closed-set string classifiers
// directly -- the real `claude -p` exec path is covered by
// claudecode_integration_test.go behind //go:build integration).

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- Class 3: native weekly-limit text (operator-cited literal phrase) ---

func TestIsQuotaSignal_WeeklyLimit_OperatorCitedPhrase(t *testing.T) {
	// UNCONFIRMED (§11.4.6): the exact live claude -p stderr/exit text for
	// this class was not re-captured during PWU-2 (would burn the quota
	// this bridge protects). "weekly limit" is both the operator-cited
	// literal phrase for this PWU AND an existing captured-FACT
	// quota-exhausted marker
	// (constitution/scripts/multitrack/multitrack_fallback_monitor.sh
	// MT_FBMON_QUOTA_DEFAULT).
	assert.True(t, isQuotaSignal("You've hit your weekly limit · resets Monday"),
		"the native Claude-Code-CLI weekly-limit UX message MUST classify as a quota signal")
}

func TestIsQuotaSignal_SessionLimit(t *testing.T) {
	assert.True(t, isQuotaSignal("Your session limit has been reached"))
}

func TestIsQuotaSignal_402CapturedDetailText(t *testing.T) {
	// Real captured 402 body text (qa-results/multitrack/logs/
	// T4_iter2_20260706T223846Z.log) -- proves the SAME phrase that HTTP
	// classifies as QUOTA_EXCEEDED (errors.go) is ALSO recognised by the
	// bridge's own text classifier when it appears in claude -p
	// stdout/stderr rather than an HTTP response body.
	assert.True(t, isQuotaSignal(`{"detail":"Subscription usage cap exceeded. Please add balance to continue."}`))
}

func TestIsQuotaSignal_QuotaPrecedesRateLimit_NoFalseRateLimitMatch(t *testing.T) {
	// The weekly-limit phrase must NOT also match the rate-limit signal
	// set -- if it did, ChatCompletion's isQuotaSignal-checked-first
	// ordering would still be correct, but a caller relying on
	// isRateLimitSignal alone (pre-PWU-2 code path) would have
	// mis-classified a permanent quota condition as a transient one.
	weeklyLimitText := "You've hit your weekly limit · resets Monday"
	assert.True(t, isQuotaSignal(weeklyLimitText))
	assert.False(t, isRateLimitSignal(weeklyLimitText),
		"a weekly-limit message must not ALSO satisfy the transient rate-limit signal set")
}

func TestIsQuotaSignal_EmptyString_False(t *testing.T) {
	assert.False(t, isQuotaSignal(""))
}

func TestIsQuotaSignal_UnrelatedText_False(t *testing.T) {
	assert.False(t, isQuotaSignal("connection refused"))
}

// --- Rate-limit signal set is unaffected by the PWU-2 addition -----------

func TestIsRateLimitSignal_StillMatchesFairUsage429(t *testing.T) {
	// Real captured 429 fixture text
	// (scripts/multitrack/test_provider_autorotate.sh FIX_429), reused
	// here to prove PWU-2's quota-signal addition did not regress the
	// pre-existing PWU-1 rate-limit classification.
	assert.True(t, isRateLimitSignal(`API Error: Request rejected (429) · Error from provider(zai-coding-plan,glm-5.2: 429): {"error":{"code":"1313","message":"Your account's current usage pattern does not comply with the Fair Usage Policy, and your request frequency has been limited."}}`))
}
