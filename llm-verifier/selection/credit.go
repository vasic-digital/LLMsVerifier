package selection

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// CreditReporter determines an account's credit status. Implementations are
// free to read a balance endpoint, interpret a probe, or consult a cache.
//
// The account argument is an opaque, caller-defined identifier. This package
// never parses it, never derives a filesystem path from it, and never assumes
// any naming scheme — that is what keeps the interface reusable.
type CreditReporter interface {
	CreditStatus(ctx context.Context, account string) (CreditStatus, error)
}

// StaticCreditReporter reports a fixed status. Its purpose in production is
// the operator-declared path: a consumer that already knows the answer (from
// its own configuration) can present it through the same interface, with the
// provenance honestly recorded as CreditSignalOperatorDeclared.
type StaticCreditReporter struct {
	Status CreditStatus
}

// CreditStatus returns the configured status, validating the
// state-requires-signal invariant so a malformed declaration cannot enter the
// selector.
func (r StaticCreditReporter) CreditStatus(_ context.Context, _ string) (CreditStatus, error) {
	if err := r.Status.Validate(); err != nil {
		return UnknownCredit("invalid declared status"), err
	}
	return r.Status, nil
}

// BalanceEndpointConfig describes a provider's balance/credits endpoint.
//
// Every field is supplied by the caller. This package ships no provider list,
// no default URL, and no default header — a bundled default would embed one
// consumer's provider set into shared infrastructure (CONST-069).
type BalanceEndpointConfig struct {
	// URL is the full balance endpoint URL. Required.
	URL string
	// Method defaults to GET when empty.
	Method string
	// Headers are sent verbatim (authorisation, API version, …).
	Headers map[string]string
	// AmountPath is a slash-separated path into the JSON response naming the
	// numeric balance, e.g. "data/total_available" or "credit_balance".
	// Required — this package does not guess field names.
	AmountPath string
	// CurrencyPath optionally names a string field holding the currency.
	CurrencyPath string
	// Timeout defaults to 15s when zero.
	Timeout time.Duration
}

// Validate reports whether the config carries the inputs this package refuses
// to invent.
func (c BalanceEndpointConfig) Validate() error {
	if strings.TrimSpace(c.URL) == "" {
		return fmt.Errorf("%w: balance endpoint URL is required", ErrPolicyIncomplete)
	}
	if strings.TrimSpace(c.AmountPath) == "" {
		return fmt.Errorf("%w: balance endpoint AmountPath is required", ErrPolicyIncomplete)
	}
	return nil
}

// BalanceEndpointReporter reads a real balance endpoint over HTTP.
//
// Failure is never coerced into a decision: a transport error, a non-2xx
// status, unparseable JSON, or a missing/non-numeric amount field all yield
// CreditUnknown with the reason in Detail. Only an actual number read from the
// response produces CreditAvailable or CreditExhausted.
type BalanceEndpointReporter struct {
	Config BalanceEndpointConfig
	// Client is optional; a timeout-bounded default is used when nil.
	Client *http.Client
	// Now is optional; time.Now is used when nil. Present so tests can pin
	// ObservedAt without touching the wall clock.
	Now func() time.Time
}

var _ CreditReporter = (*BalanceEndpointReporter)(nil)
var _ CreditReporter = StaticCreditReporter{}

// CreditStatus performs the balance request and interprets the response.
func (r *BalanceEndpointReporter) CreditStatus(ctx context.Context, account string) (CreditStatus, error) {
	if err := r.Config.Validate(); err != nil {
		return UnknownCredit("balance endpoint misconfigured"), err
	}

	method := r.Config.Method
	if method == "" {
		method = http.MethodGet
	}
	req, err := http.NewRequestWithContext(ctx, method, r.Config.URL, nil)
	if err != nil {
		return UnknownCredit(fmt.Sprintf("balance request build failed: %v", err)), err
	}
	for k, v := range r.Config.Headers {
		req.Header.Set(k, v)
	}

	client := r.Client
	if client == nil {
		timeout := r.Config.Timeout
		if timeout <= 0 {
			timeout = 15 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}

	resp, err := client.Do(req)
	if err != nil {
		// Transport failure says nothing about the account's credit.
		return UnknownCredit(fmt.Sprintf("balance request failed: %v", err)), nil
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return UnknownCredit(fmt.Sprintf("balance response unreadable: %v", err)), nil
	}

	// HTTP 402 on a balance endpoint is itself a positive statement that the
	// account cannot pay — the same signal class providers.ProbeVerdictQuotaExceeded
	// carries on an inference endpoint.
	if resp.StatusCode == http.StatusPaymentRequired {
		return CreditStatus{
			State:      CreditExhausted,
			Signal:     CreditSignalBalanceEndpoint,
			Detail:     fmt.Sprintf("balance endpoint returned 402 for account %q", account),
			ObservedAt: r.now(),
		}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return UnknownCredit(fmt.Sprintf("balance endpoint status %d", resp.StatusCode)), nil
	}

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return UnknownCredit(fmt.Sprintf("balance response not JSON: %v", err)), nil
	}

	amount, ok := lookupNumber(payload, r.Config.AmountPath)
	if !ok {
		return UnknownCredit(fmt.Sprintf("no numeric value at %q", r.Config.AmountPath)), nil
	}
	currency, _ := lookupString(payload, r.Config.CurrencyPath)

	return CreditFromBalance(amount, currency, r.now()), nil
}

func (r *BalanceEndpointReporter) now() time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now()
}

// lookupPath walks a slash-separated path through decoded JSON objects.
func lookupPath(doc any, path string) (any, bool) {
	if strings.TrimSpace(path) == "" {
		return nil, false
	}
	cur := doc
	for _, segment := range strings.Split(path, "/") {
		if segment == "" {
			continue
		}
		obj, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = obj[segment]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

// lookupNumber resolves a path to a JSON number. A numeric string is accepted
// because several providers return balances as strings; anything else fails
// rather than defaulting to zero, since zero means "exhausted" here.
func lookupNumber(doc any, path string) (float64, bool) {
	v, ok := lookupPath(doc, path)
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return n, true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case string:
		// strconv (not Sscanf) so a trailing-garbage string like "12.5abc"
		// is rejected outright rather than silently truncated to a balance.
		if f, err := strconv.ParseFloat(strings.TrimSpace(n), 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// lookupString resolves a path to a JSON string.
func lookupString(doc any, path string) (string, bool) {
	v, ok := lookupPath(doc, path)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}
