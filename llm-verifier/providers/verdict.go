package providers

// verdict.go — PWU-2 of the LLMsVerifier incorporation
// (docs/research/llmsverifier_incorporation_20260707/ANALYSIS_AND_PLAN.md
// PART C.3 + PART F PWU-2, ATMOSphere-Android-15 repo). Wires
// providers/errors.go's ClassifyError (previously a classifier with ZERO
// non-self callers -- PART A.3 line 31 of the incorporation plan) into an
// actual PROBE RESULT: a closed-set ProbeVerdict + retry_after that a
// caller (health probe, multitrack provider-rotation consumer) can act
// on, so a capped route surfaces as a DISTINCT rate_limited/quota_exceeded
// verdict instead of collapsing into an opaque generic failure.
//
// Three real failure classes wired here, all captured live (§11.4.6 --
// no guessing):
//
//  1. HTTP 429 -- qa-results/multitrack/logs/*.log (this repo, T1..T4
//     provider-rotation supervisor logs, 2026-07-06):
//     "API Error: Request rejected (429) . Error from
//     provider(zai-coding-plan,glm-5.2: 429): {"error":{"code":"1313",
//     "message":"Your account's current usage pattern does not comply
//     with the Fair Usage Policy..."}}"
//     Already correctly classified by the PRE-EXISTING ClassifyError
//     (Type=ErrorTypeRateLimit) -- the PWU-2 gap for this class is that
//     nothing ever WIRED that classification into a caller-visible
//     verdict. ClassifyHTTPProbeOutcome below is that wiring.
//
//  2. HTTP 402 -- qa-results/multitrack/logs/T4_iter2_20260706T223846Z.log
//     {"detail":"Subscription usage cap exceeded. Please add balance to
//     continue."}. providers/errors.go had NO 402 case anywhere (every
//     per-provider classifier falls through to UNKNOWN_ERROR) -- fixed in
//     this same PWU (see errors.go's http.StatusPaymentRequired branch),
//     proven RED->GREEN by
//     providers/errors_402_quota_red_test.go.
//
//  3. Native Claude-Code-CLI-bridge weekly-limit text -- the Claude Code
//     CLI's own weekly-usage-cap UX message ("You've hit your weekly
//     limit ..."), the same quota-exhausted class already recognised as a
//     captured-FACT sub-classification marker by
//     constitution/scripts/multitrack/multitrack_fallback_monitor.sh
//     (MT_FBMON_QUOTA_DEFAULT = "resets\nweekly limit\nsession limit").
//     UNCONFIRMED (§11.4.6): the exact live claude -p exit code/stderr
//     byte-for-byte text for this class was not re-captured during this
//     PWU (forcing it live would burn the very quota this bridge exists
//     to protect); the fixture used in claudecode_quota_test.go is the
//     operator-cited literal phrase plus the project's existing
//     quota-marker vocabulary. Classified in providers/claudecode.go via
//     ErrClaudeCodeQuotaExceeded (sibling of the pre-existing
//     ErrClaudeCodeRateLimited), wired into this file's ProbeOutcome via
//     ClassifyProbeError.
//
// NOT wired in this PWU (documented gap, not silently skipped per
// §11.4.6): the CLI's own `api_error_status` JSON field
// (claudecode.go's claudeCodeCLIResult.APIErrorStatus) is kept as
// json.RawMessage because its non-null shape on a genuine error response
// was never captured live (see claudecode.go's doc comment) -- feeding an
// UNCONFIRMED schema through ClassifyError would be a guess, not a wire.
// This is a tracked follow-up, not a silent omission.

import (
	"errors"
	"fmt"
	"net/http"
)

// ProbeVerdict is the closed-set outcome of a provider/model health or
// verification probe (§11.4.69 sink-side positive-evidence taxonomy).
// Never extend this set with a free-text value -- add a new named
// constant so every switch over ProbeVerdict stays exhaustive-checkable.
type ProbeVerdict string

const (
	// ProbeVerdictOK means the probe completed with a genuine content
	// determination and no infra condition was observed.
	ProbeVerdictOK ProbeVerdict = "ok"
	// ProbeVerdictRateLimited means the probe hit a TRANSIENT rate-limit
	// / Fair-Usage / overloaded condition (HTTP 429, or the bridge's
	// 429/"rate limit"/"overloaded"/"too many requests" text signals)
	// that is expected to clear on its own after RetryAfterSeconds (or a
	// short backoff when no Retry-After was supplied).
	ProbeVerdictRateLimited ProbeVerdict = "rate_limited"
	// ProbeVerdictQuotaExceeded means the probe hit a quota /
	// subscription / balance / weekly-or-session-limit condition (HTTP
	// 402, or the bridge's weekly/session-limit text signals) that will
	// NOT clear on a short retry -- distinct from ProbeVerdictRateLimited
	// so a caller (e.g. the multitrack provider-rotation consumer) can
	// apply a materially longer cooldown instead of a quick backoff.
	ProbeVerdictQuotaExceeded ProbeVerdict = "quota_exceeded"
	// ProbeVerdictFailed means the probe failed for a reason that is
	// neither rate-limit nor quota (auth failure, invalid request,
	// server error, CLI parse failure, etc).
	ProbeVerdictFailed ProbeVerdict = "failed"
)

// ProbeOutcome is the structured result PWU-2 wires providers/errors.go's
// ClassifyError (HTTP path) and the Claude-Code-CLI-bridge's sentinel
// errors (native path, providers/claudecode.go) into. This REPLACES the
// previous opaque-string-error collapse a caller had no way to
// distinguish rate_limited/quota_exceeded from a generic failure without
// re-parsing free text.
type ProbeOutcome struct {
	Verdict           ProbeVerdict `json:"verdict"`
	RetryAfterSeconds int          `json:"retry_after_seconds,omitempty"`
	Detail            string       `json:"detail,omitempty"`
}

// ClassifyHTTPProbeOutcome wires providers/errors.go's ClassifyError into
// a ProbeOutcome for any HTTP-based adapter (openai-compatible,
// anthropic, deepseek, google, generic). This is the class-1 (HTTP 429)
// and class-2 (HTTP 402) wiring PWU-2 adds: a caller that previously saw
// only an opaque "<provider> API error: <status> - <body>" string can
// instead call this function against the same *http.Response + body and
// receive a verdict it can act on (backoff vs long-cooldown vs generic
// failure) plus the parsed Retry-After duration when the server sent one.
func ClassifyHTTPProbeOutcome(provider string, resp *http.Response, body []byte) ProbeOutcome {
	if resp == nil {
		return ProbeOutcome{Verdict: ProbeVerdictFailed, Detail: "nil http response"}
	}

	pe := NewErrorClassifier(provider).ClassifyError(resp, body)

	outcome := ProbeOutcome{Detail: pe.Message}
	switch pe.Type {
	case ErrorTypeRateLimit:
		outcome.Verdict = ProbeVerdictRateLimited
	case ErrorTypeQuota:
		outcome.Verdict = ProbeVerdictQuotaExceeded
	default:
		outcome.Verdict = ProbeVerdictFailed
	}

	if pe.RetryAfter > 0 {
		outcome.RetryAfterSeconds = int(pe.RetryAfter.Seconds())
	}
	if outcome.Detail == "" {
		outcome.Detail = fmt.Sprintf("http status %d", resp.StatusCode)
	}
	return outcome
}

// ClassifyProbeError maps ANY probe error -- Claude-Code-CLI-bridge
// (providers/claudecode.go's ErrClaudeCodeRateLimited /
// ErrClaudeCodeQuotaExceeded sentinels) or any other errors.Is-wrapped
// sentinel a future adapter introduces -- into the same ProbeOutcome
// shape ClassifyHTTPProbeOutcome produces for HTTP adapters, so a caller
// (health probe, multitrack consumer) never has to know whether the
// underlying alias was HTTP or CLI-bridge. This is the class-3 (native
// weekly-limit text) wiring PWU-2 adds.
func ClassifyProbeError(err error) ProbeOutcome {
	if err == nil {
		return ProbeOutcome{Verdict: ProbeVerdictOK}
	}
	// Quota checked before rate-limit: ErrClaudeCodeQuotaExceeded is the
	// more specific / higher-cost-to-misclassify condition (it will not
	// clear on a short retry), and errors.Is against a wrapped sentinel
	// never matches both at once for the same error value, so ordering
	// here only matters for defence-in-depth against a future error that
	// (incorrectly) wraps both sentinels.
	if errors.Is(err, ErrClaudeCodeQuotaExceeded) {
		return ProbeOutcome{Verdict: ProbeVerdictQuotaExceeded, Detail: err.Error()}
	}
	if errors.Is(err, ErrClaudeCodeRateLimited) {
		return ProbeOutcome{Verdict: ProbeVerdictRateLimited, Detail: err.Error()}
	}
	return ProbeOutcome{Verdict: ProbeVerdictFailed, Detail: err.Error()}
}
