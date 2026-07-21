package providers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"digital.vasic.llmsverifier/selection"
)

func TestCreditStatusFromProbeOutcome_Mapping(t *testing.T) {
	at := time.Unix(1700000000, 0).UTC()

	cases := []struct {
		name       string
		outcome    ProbeOutcome
		probed     selection.Affordability
		wantState  selection.CreditState
		wantSignal selection.CreditSignal
	}{
		{
			name:       "quota exceeded is a positive exhausted signal",
			outcome:    ProbeOutcome{Verdict: ProbeVerdictQuotaExceeded, Detail: "402 add balance"},
			probed:     selection.AffordabilityPaid,
			wantState:  selection.CreditExhausted,
			wantSignal: selection.CreditSignalProbeResponse,
		},
		{
			name:       "quota exceeded on a free model still means exhausted",
			outcome:    ProbeOutcome{Verdict: ProbeVerdictQuotaExceeded},
			probed:     selection.AffordabilityFree,
			wantState:  selection.CreditExhausted,
			wantSignal: selection.CreditSignalProbeResponse,
		},
		{
			name:       "ok on a paid model proves credit",
			outcome:    ProbeOutcome{Verdict: ProbeVerdictOK},
			probed:     selection.AffordabilityPaid,
			wantState:  selection.CreditAvailable,
			wantSignal: selection.CreditSignalProbeResponse,
		},
		{
			name:       "ok on a FREE model proves nothing about paid credit",
			outcome:    ProbeOutcome{Verdict: ProbeVerdictOK},
			probed:     selection.AffordabilityFree,
			wantState:  selection.CreditUnknown,
			wantSignal: selection.CreditSignalNone,
		},
		{
			name:       "ok on an unpriced model proves nothing",
			outcome:    ProbeOutcome{Verdict: ProbeVerdictOK},
			probed:     selection.AffordabilityUnknown,
			wantState:  selection.CreditUnknown,
			wantSignal: selection.CreditSignalNone,
		},
		{
			name:       "rate limited is transient, not a credit fact",
			outcome:    ProbeOutcome{Verdict: ProbeVerdictRateLimited, RetryAfterSeconds: 30},
			probed:     selection.AffordabilityPaid,
			wantState:  selection.CreditUnknown,
			wantSignal: selection.CreditSignalNone,
		},
		{
			name:       "generic failure is not a credit fact",
			outcome:    ProbeOutcome{Verdict: ProbeVerdictFailed, Detail: "500 upstream"},
			probed:     selection.AffordabilityPaid,
			wantState:  selection.CreditUnknown,
			wantSignal: selection.CreditSignalNone,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CreditStatusFromProbeOutcome(tc.outcome, tc.probed, at)
			if got.State != tc.wantState {
				t.Fatalf("state = %q, want %q", got.State, tc.wantState)
			}
			if got.Signal != tc.wantSignal {
				t.Fatalf("signal = %q, want %q", got.Signal, tc.wantSignal)
			}
			if err := got.Validate(); err != nil {
				t.Fatalf("produced status violates its own invariant: %v", err)
			}
			if got.Detail == "" {
				t.Fatal("produced status carries no explanation")
			}
		})
	}
}

// TestCreditStatusFromProbeOutcome_WiresRealClassifier drives the mapping from
// a REAL http.Response through the pre-existing ClassifyHTTPProbeOutcome
// classifier — the actual production path — rather than from a hand-built
// ProbeOutcome. A live 402 body captured in verdict.go's docs is replayed by a
// real loopback server.
func TestCreditStatusFromProbeOutcome_WiresRealClassifier(t *testing.T) {
	body := `{"detail":"Subscription usage cap exceeded. Please add balance to continue."}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("probe request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	outcome := ClassifyHTTPProbeOutcome("test-provider", resp, []byte(body))
	if outcome.Verdict != ProbeVerdictQuotaExceeded {
		t.Fatalf("classifier verdict = %q, want quota_exceeded", outcome.Verdict)
	}

	status := CreditStatusFromProbeOutcome(outcome, selection.AffordabilityPaid, time.Now())
	if status.State != selection.CreditExhausted {
		t.Fatalf("state = %q, want exhausted", status.State)
	}
	if status.Signal != selection.CreditSignalProbeResponse {
		t.Fatalf("signal = %q, want probe_response", status.Signal)
	}
}

// TestCreditStatusFromProbeOutcome_DrivesSelection closes the loop: a real 402
// response steers a real selector onto the free branch.
func TestCreditStatusFromProbeOutcome_DrivesSelection(t *testing.T) {
	candidates := []selection.Candidate{
		{ID: "paid-top", Strength: 9, Price: selection.KnownPrice(3, 15, "USD")},
		{ID: "free-top", Strength: 6, Price: selection.KnownPrice(0, 0, "USD")},
	}
	policy := selection.Policy{
		OnUnknownCredit: selection.UnknownCreditPreferFree,
		TieBreak:        selection.TieBreakCheapest,
	}

	exhausted := CreditStatusFromProbeOutcome(
		ProbeOutcome{Verdict: ProbeVerdictQuotaExceeded, Detail: "402"},
		selection.AffordabilityPaid, time.Now())
	d, err := selection.NewCreditAwareSelector().Select(t.Context(), candidates, exhausted, policy)
	if err != nil {
		t.Fatalf("selection error: %v", err)
	}
	if d.Chosen == nil || d.Chosen.ID != "free-top" {
		t.Fatalf("chose %+v, want free-top", d.Chosen)
	}

	available := CreditStatusFromProbeOutcome(
		ProbeOutcome{Verdict: ProbeVerdictOK}, selection.AffordabilityPaid, time.Now())
	d, err = selection.NewCreditAwareSelector().Select(t.Context(), candidates, available, policy)
	if err != nil {
		t.Fatalf("selection error: %v", err)
	}
	if d.Chosen == nil || d.Chosen.ID != "paid-top" {
		t.Fatalf("chose %+v, want paid-top", d.Chosen)
	}
}
