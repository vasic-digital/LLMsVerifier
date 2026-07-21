package selection

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// credit_endpoint_test.go exercises BalanceEndpointReporter against a REAL
// HTTP server over a real loopback socket (net/http/httptest) — the actual
// transport, request-building, status handling and JSON-decoding code paths
// run, with no HTTP client substituted. Per CONST-050(A) this keeps the
// non-unit surface free of fakes: the only thing standing in for a live
// provider is the server's canned payload, which is the response BODY, not a
// replacement for any of the code under test.

func newBalanceServer(t *testing.T, status int, body string, wantHeader map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range wantHeader {
			if got := r.Header.Get(k); got != v {
				t.Errorf("header %s = %q, want %q", k, got, v)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func fixedNow() func() time.Time {
	ts := time.Unix(1700000000, 0).UTC()
	return func() time.Time { return ts }
}

func TestBalanceEndpointReporter_PositiveBalanceIsAvailable(t *testing.T) {
	srv := newBalanceServer(t, http.StatusOK,
		`{"data":{"total_available":"37.25","unit":"USD"}}`,
		map[string]string{"Authorization": "Bearer test-token"})

	r := &BalanceEndpointReporter{
		Config: BalanceEndpointConfig{
			URL:          srv.URL,
			Headers:      map[string]string{"Authorization": "Bearer test-token"},
			AmountPath:   "data/total_available",
			CurrencyPath: "data/unit",
		},
		Now: fixedNow(),
	}

	got, err := r.CreditStatus(context.Background(), "acct-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.State != CreditAvailable {
		t.Fatalf("state = %q, want available", got.State)
	}
	if got.Signal != CreditSignalBalanceEndpoint {
		t.Fatalf("signal = %q, want balance_endpoint", got.Signal)
	}
	if got.Balance == nil || got.Balance.Amount != 37.25 || got.Balance.Currency != "USD" {
		t.Fatalf("balance = %+v, want 37.25 USD", got.Balance)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("reported status failed its own invariant: %v", err)
	}
}

func TestBalanceEndpointReporter_ZeroBalanceIsExhausted(t *testing.T) {
	srv := newBalanceServer(t, http.StatusOK, `{"credit_balance":0}`, nil)

	r := &BalanceEndpointReporter{
		Config: BalanceEndpointConfig{URL: srv.URL, AmountPath: "credit_balance"},
		Now:    fixedNow(),
	}
	got, err := r.CreditStatus(context.Background(), "acct-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.State != CreditExhausted {
		t.Fatalf("state = %q, want exhausted", got.State)
	}
}

// HTTP 402 on the balance endpoint is itself a positive statement of
// inability to pay — the same class providers.ProbeVerdictQuotaExceeded
// carries on an inference endpoint.
func TestBalanceEndpointReporter_PaymentRequiredIsExhausted(t *testing.T) {
	srv := newBalanceServer(t, http.StatusPaymentRequired,
		`{"detail":"Subscription usage cap exceeded. Please add balance to continue."}`, nil)

	r := &BalanceEndpointReporter{
		Config: BalanceEndpointConfig{URL: srv.URL, AmountPath: "credit_balance"},
		Now:    fixedNow(),
	}
	got, err := r.CreditStatus(context.Background(), "acct-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.State != CreditExhausted || got.Signal != CreditSignalBalanceEndpoint {
		t.Fatalf("402 produced %q/%q, want exhausted/balance_endpoint", got.State, got.Signal)
	}
}

// Everything that is NOT a successful reading must degrade to UNKNOWN — never
// to available (which would spend money on no evidence) and never to
// exhausted (which would needlessly downgrade every request).
func TestBalanceEndpointReporter_FailuresDegradeToUnknown(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		body       string
		amountPath string
	}{
		{"server error", http.StatusInternalServerError, `{"error":"boom"}`, "credit_balance"},
		{"unauthorised", http.StatusUnauthorized, `{"error":"bad key"}`, "credit_balance"},
		{"rate limited", http.StatusTooManyRequests, `{"error":"slow down"}`, "credit_balance"},
		{"not JSON", http.StatusOK, `<html>nope</html>`, "credit_balance"},
		{"field missing", http.StatusOK, `{"something_else":5}`, "credit_balance"},
		{"field not numeric", http.StatusOK, `{"credit_balance":"unlimited"}`, "credit_balance"},
		{"numeric string with garbage", http.StatusOK, `{"credit_balance":"12.5abc"}`, "credit_balance"},
		{"path into a non-object", http.StatusOK, `{"data":5}`, "data/total"},
		{"null value", http.StatusOK, `{"credit_balance":null}`, "credit_balance"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newBalanceServer(t, tc.status, tc.body, nil)
			r := &BalanceEndpointReporter{
				Config: BalanceEndpointConfig{URL: srv.URL, AmountPath: tc.amountPath},
				Now:    fixedNow(),
			}
			got, err := r.CreditStatus(context.Background(), "acct-1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.State != CreditUnknown {
				t.Fatalf("state = %q, want unknown (detail=%q)", got.State, got.Detail)
			}
			if got.Signal != CreditSignalNone {
				t.Fatalf("unknown state carried signal %q", got.Signal)
			}
			if got.Detail == "" {
				t.Fatal("unknown status carried no explanation")
			}
		})
	}
}

// A dead endpoint (connection refused) must also degrade to unknown, not to a
// decided state.
func TestBalanceEndpointReporter_TransportFailureIsUnknown(t *testing.T) {
	srv := newBalanceServer(t, http.StatusOK, `{"credit_balance":5}`, nil)
	dead := srv.URL
	srv.Close() // real closed socket, real dial failure

	r := &BalanceEndpointReporter{
		Config: BalanceEndpointConfig{URL: dead, AmountPath: "credit_balance", Timeout: 2 * time.Second},
		Now:    fixedNow(),
	}
	got, err := r.CreditStatus(context.Background(), "acct-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.State != CreditUnknown {
		t.Fatalf("state = %q, want unknown", got.State)
	}
}

func TestBalanceEndpointReporter_RejectsIncompleteConfig(t *testing.T) {
	cases := map[string]BalanceEndpointConfig{
		"no URL":           {AmountPath: "balance"},
		"no AmountPath":    {URL: "http://127.0.0.1:1"},
		"blank AmountPath": {URL: "http://127.0.0.1:1", AmountPath: "   "},
	}
	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			r := &BalanceEndpointReporter{Config: cfg}
			got, err := r.CreditStatus(context.Background(), "acct")
			if !errors.Is(err, ErrPolicyIncomplete) {
				t.Fatalf("err = %v, want ErrPolicyIncomplete", err)
			}
			if got.State != CreditUnknown {
				t.Fatalf("misconfigured reporter produced state %q", got.State)
			}
		})
	}
}

// End-to-end through the real reporter and the real selector: a live balance
// reading drives the branch, with no hand-built CreditStatus in between.
func TestBalanceEndpointReporter_DrivesSelection(t *testing.T) {
	t.Run("funded account gets the strongest paid model", func(t *testing.T) {
		srv := newBalanceServer(t, http.StatusOK, `{"credit_balance":250.0}`, nil)
		r := &BalanceEndpointReporter{
			Config: BalanceEndpointConfig{URL: srv.URL, AmountPath: "credit_balance"},
			Now:    fixedNow(),
		}
		status, err := r.CreditStatus(context.Background(), "acct")
		if err != nil {
			t.Fatalf("reporter error: %v", err)
		}
		d := mustSelect(t, mixedPool(), status, basePolicy(UnknownCreditPreferFree))
		if d.Chosen.ID != paidStrong.ID {
			t.Fatalf("chose %q, want %q", d.Chosen.ID, paidStrong.ID)
		}
	})

	t.Run("drained account gets the strongest free model", func(t *testing.T) {
		srv := newBalanceServer(t, http.StatusOK, `{"credit_balance":0.0}`, nil)
		r := &BalanceEndpointReporter{
			Config: BalanceEndpointConfig{URL: srv.URL, AmountPath: "credit_balance"},
			Now:    fixedNow(),
		}
		status, err := r.CreditStatus(context.Background(), "acct")
		if err != nil {
			t.Fatalf("reporter error: %v", err)
		}
		d := mustSelect(t, mixedPool(), status, basePolicy(UnknownCreditPreferFree))
		if d.Chosen.ID != freeStrong.ID {
			t.Fatalf("chose %q, want %q", d.Chosen.ID, freeStrong.ID)
		}
	})

	t.Run("unreachable endpoint takes the conservative branch", func(t *testing.T) {
		srv := newBalanceServer(t, http.StatusOK, `{"credit_balance":250.0}`, nil)
		dead := srv.URL
		srv.Close()
		r := &BalanceEndpointReporter{
			Config: BalanceEndpointConfig{URL: dead, AmountPath: "credit_balance", Timeout: 2 * time.Second},
			Now:    fixedNow(),
		}
		status, err := r.CreditStatus(context.Background(), "acct")
		if err != nil {
			t.Fatalf("reporter error: %v", err)
		}
		d := mustSelect(t, mixedPool(), status, basePolicy(UnknownCreditPreferFree))
		if d.Chosen.ID != freeStrong.ID {
			t.Fatalf("chose %q, want %q", d.Chosen.ID, freeStrong.ID)
		}
		if d.ReasonID != ReasonFreeUnknownCredit {
			t.Fatalf("reason = %q, want %q", d.ReasonID, ReasonFreeUnknownCredit)
		}
	})
}

func TestStaticCreditReporter(t *testing.T) {
	t.Run("valid declaration passes through", func(t *testing.T) {
		want := DeclaredCredit(CreditAvailable, "prepaid invoice #42", fixedNow()())
		r := StaticCreditReporter{Status: want}
		got, err := r.CreditStatus(context.Background(), "acct")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.State != CreditAvailable || got.Signal != CreditSignalOperatorDeclared {
			t.Fatalf("got %q/%q", got.State, got.Signal)
		}
	})

	t.Run("invalid declaration is refused", func(t *testing.T) {
		r := StaticCreditReporter{Status: CreditStatus{State: CreditAvailable, Signal: CreditSignalNone}}
		got, err := r.CreditStatus(context.Background(), "acct")
		if !errors.Is(err, ErrInvalidCreditStatus) {
			t.Fatalf("err = %v, want ErrInvalidCreditStatus", err)
		}
		if got.State != CreditUnknown {
			t.Fatalf("refused declaration produced state %q", got.State)
		}
	})
}
