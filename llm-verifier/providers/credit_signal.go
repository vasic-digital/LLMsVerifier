package providers

// credit_signal.go wires this package's EXISTING probe classification
// (ProbeVerdict, produced by ClassifyHTTPProbeOutcome / ClassifyProbeError)
// into the selection package's credit vocabulary.
//
// Without this file the two halves stay disconnected: providers/verdict.go
// already distinguishes a billing wall (ProbeVerdictQuotaExceeded, HTTP 402
// "Subscription usage cap exceeded. Please add balance to continue.") from a
// transient cap (ProbeVerdictRateLimited), but nothing turned that into an
// account-level credit determination a selector could act on.

import (
	"time"

	"digital.vasic.llmsverifier/selection"
)

// CreditStatusFromProbeOutcome infers an account's credit state from how a
// provider answered one real inference probe.
//
// probedAffordability is REQUIRED and is the reason this is not a one-line
// mapping. A successful probe only proves the account can pay when the model
// probed actually costs money: a 200 OK from a free model says nothing
// whatsoever about paid credit, and treating it as proof would route paid
// traffic on the strength of a free request. That case returns
// CreditUnknown, not CreditAvailable.
//
// Mapping:
//
//	quota_exceeded              -> exhausted (HTTP 402 / subscription cap)
//	ok  + probed model is paid  -> available (a billable request was served)
//	ok  + free or unknown model -> unknown   (proves nothing about credit)
//	rate_limited                -> unknown   (transient, not a credit fact)
//	failed                      -> unknown   (auth/invalid/server, not credit)
func CreditStatusFromProbeOutcome(outcome ProbeOutcome, probedAffordability selection.Affordability, observedAt time.Time) selection.CreditStatus {
	switch outcome.Verdict {
	case ProbeVerdictQuotaExceeded:
		return selection.CreditStatus{
			State:      selection.CreditExhausted,
			Signal:     selection.CreditSignalProbeResponse,
			Detail:     detailOr(outcome.Detail, string(outcome.Verdict)),
			ObservedAt: observedAt,
		}
	case ProbeVerdictOK:
		if probedAffordability != selection.AffordabilityPaid {
			return selection.UnknownCredit(
				"probe succeeded on a model that is not known to be paid; " +
					"no conclusion about paid credit")
		}
		return selection.CreditStatus{
			State:      selection.CreditAvailable,
			Signal:     selection.CreditSignalProbeResponse,
			Detail:     detailOr(outcome.Detail, "paid model probe succeeded"),
			ObservedAt: observedAt,
		}
	case ProbeVerdictRateLimited:
		return selection.UnknownCredit(
			detailOr(outcome.Detail, "rate limited; transient, not a credit determination"))
	default:
		return selection.UnknownCredit(
			detailOr(outcome.Detail, "probe failed for a non-credit reason"))
	}
}

func detailOr(detail, fallback string) string {
	if detail != "" {
		return detail
	}
	return fallback
}
