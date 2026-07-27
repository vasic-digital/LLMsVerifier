package selection

import (
	"context"
	"errors"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Shared fixtures. Strengths are deliberately non-monotonic with price so a
// selector that sorted by price and got the right answer by luck would fail.
// ---------------------------------------------------------------------------

var (
	paidStrong = Candidate{ID: "paid-strong", Provider: "p1", Strength: 9.5, Price: KnownPrice(3, 15, "USD")}
	paidMid    = Candidate{ID: "paid-mid", Provider: "p1", Strength: 7.0, Price: KnownPrice(10, 30, "USD")}
	paidWeak   = Candidate{ID: "paid-weak", Provider: "p2", Strength: 4.0, Price: KnownPrice(1, 2, "USD")}
	freeStrong = Candidate{ID: "free-strong", Provider: "p3", Strength: 6.5, Price: KnownPrice(0, 0, "USD")}
	freeWeak   = Candidate{ID: "free-weak", Provider: "p3", Strength: 2.0, Price: KnownPrice(0, 0, "USD")}
	pricelessX = Candidate{ID: "unpriced-x", Provider: "p4", Strength: 10.0, Price: UnknownPrice()}
)

func mixedPool() []Candidate {
	return []Candidate{freeWeak, paidMid, pricelessX, freeStrong, paidWeak, paidStrong}
}

func creditAvailable() CreditStatus {
	return CreditFromBalance(42.5, "USD", time.Unix(1700000000, 0))
}

func creditExhausted() CreditStatus {
	return CreditFromBalance(0, "USD", time.Unix(1700000000, 0))
}

func basePolicy(onUnknown UnknownCreditPolicy) Policy {
	return Policy{OnUnknownCredit: onUnknown, TieBreak: TieBreakCheapest}
}

func mustSelect(t *testing.T, candidates []Candidate, credit CreditStatus, policy Policy) Decision {
	t.Helper()
	d, err := NewCreditAwareSelector().Select(context.Background(), candidates, credit, policy)
	if err != nil {
		t.Fatalf("Select returned unexpected error: %v (decision=%+v)", err, d)
	}
	if d.Chosen == nil {
		t.Fatalf("Select returned nil Chosen with no error (decision=%+v)", d)
	}
	return d
}

// ---------------------------------------------------------------------------
// TokenPrice / Affordability
// ---------------------------------------------------------------------------

func TestTokenPrice_Affordability(t *testing.T) {
	cases := []struct {
		name  string
		price TokenPrice
		want  Affordability
	}{
		{"zero value is unknown, NOT free", TokenPrice{}, AffordabilityUnknown},
		{"explicit unknown", UnknownPrice(), AffordabilityUnknown},
		{"observed zero is free", KnownPrice(0, 0, "USD"), AffordabilityFree},
		{"positive input only is paid", KnownPrice(0.5, 0, "USD"), AffordabilityPaid},
		{"positive output only is paid", KnownPrice(0, 0.5, "USD"), AffordabilityPaid},
		{"both positive is paid", KnownPrice(3, 15, "USD"), AffordabilityPaid},
		{"negative input is unknown, not a discount", KnownPrice(-1, 5, "USD"), AffordabilityUnknown},
		{"negative output is unknown", KnownPrice(5, -1, "USD"), AffordabilityUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.price.Affordability(); got != tc.want {
				t.Fatalf("Affordability() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestTokenPrice_UnobservedIsNotZero is the paired mutation for the whole
// design: if TokenPrice ever lost its Known flag and collapsed to bare
// float64 (the shape scoring.ModelData uses), an unobserved price would
// report a total of 0 and be indistinguishable from free.
func TestTokenPrice_UnobservedIsNotZero(t *testing.T) {
	if _, ok := UnknownPrice().Total(); ok {
		t.Fatal("unobserved price reported a usable total; free/unknown collapse reintroduced")
	}
	total, ok := KnownPrice(0, 0, "USD").Total()
	if !ok || total != 0 {
		t.Fatalf("observed free price: total=%v ok=%v, want 0/true", total, ok)
	}
}

// ---------------------------------------------------------------------------
// CreditStatus invariants
// ---------------------------------------------------------------------------

func TestCreditStatus_Validate(t *testing.T) {
	cases := []struct {
		name    string
		status  CreditStatus
		wantErr bool
	}{
		{"unknown with no signal", UnknownCredit("nothing checked"), false},
		{"available from balance", CreditFromBalance(5, "USD", time.Now()), false},
		{"exhausted from zero balance", CreditFromBalance(0, "USD", time.Now()), false},
		{"exhausted from negative balance", CreditFromBalance(-3, "USD", time.Now()), false},
		{"declared available", DeclaredCredit(CreditAvailable, "operator says so", time.Now()), false},
		{"available with no signal is a bluff", CreditStatus{State: CreditAvailable, Signal: CreditSignalNone}, true},
		{"exhausted with empty signal is a bluff", CreditStatus{State: CreditExhausted}, true},
		{"unknown carrying a signal is incoherent", CreditStatus{State: CreditUnknown, Signal: CreditSignalBalanceEndpoint}, true},
		{"unrecognised state", CreditStatus{State: "maybe", Signal: CreditSignalProbeResponse}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.status.Validate()
			if tc.wantErr && err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
			if tc.wantErr && !errors.Is(err, ErrInvalidCreditStatus) {
				t.Fatalf("error %v is not ErrInvalidCreditStatus", err)
			}
		})
	}
}

func TestCreditFromBalance_StateBoundary(t *testing.T) {
	cases := []struct {
		amount float64
		want   CreditState
	}{
		{100, CreditAvailable},
		{0.0001, CreditAvailable},
		{0, CreditExhausted},
		{-0.01, CreditExhausted},
	}
	for _, tc := range cases {
		got := CreditFromBalance(tc.amount, "USD", time.Now())
		if got.State != tc.want {
			t.Fatalf("balance %v -> state %q, want %q", tc.amount, got.State, tc.want)
		}
		if got.Signal != CreditSignalBalanceEndpoint {
			t.Fatalf("balance %v -> signal %q, want balance_endpoint", tc.amount, got.Signal)
		}
		if got.Balance == nil || got.Balance.Amount != tc.amount {
			t.Fatalf("balance %v not carried through: %+v", tc.amount, got.Balance)
		}
	}
}

// DeclaredCredit must not let "declared unknown" masquerade as a decided
// state carrying an operator signal.
func TestDeclaredCredit_UnknownStaysUnknown(t *testing.T) {
	got := DeclaredCredit(CreditUnknown, "operator unsure", time.Now())
	if got.State != CreditUnknown || got.Signal != CreditSignalNone {
		t.Fatalf("declared-unknown produced %+v, want unknown/none", got)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("declared-unknown failed validation: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Core branch behaviour — the operator's mandate
// ---------------------------------------------------------------------------

// Credit available => strongest PAID model, even though an unpriced candidate
// and a free candidate exist and the strongest paid one is not the cheapest.
func TestSelect_CreditAvailable_PicksStrongestPaid(t *testing.T) {
	d := mustSelect(t, mixedPool(), creditAvailable(), basePolicy(UnknownCreditPreferFree))

	if d.Chosen.ID != paidStrong.ID {
		t.Fatalf("chose %q, want %q", d.Chosen.ID, paidStrong.ID)
	}
	if d.Branch != BranchPaid {
		t.Fatalf("branch = %q, want paid", d.Branch)
	}
	if d.ReasonID != ReasonPaidCreditAvailable {
		t.Fatalf("reason = %q, want %q", d.ReasonID, ReasonPaidCreditAvailable)
	}
	if d.FellBack || d.UsedUnknownPriced {
		t.Fatalf("unexpected fallback flags: %+v", d)
	}
	if d.CreditState != CreditAvailable || d.CreditSignal != CreditSignalBalanceEndpoint {
		t.Fatalf("credit echo = %q/%q", d.CreditState, d.CreditSignal)
	}
	if d.Considered != 6 || d.PaidAvailable != 3 || d.FreeAvailable != 2 || d.UnknownPriced != 1 {
		t.Fatalf("population counts wrong: %+v", d)
	}
}

// No credit => strongest FREE model. The unpriced candidate is the strongest
// overall (10.0) and must NOT be picked: its cost is unknown.
func TestSelect_CreditExhausted_PicksStrongestFree(t *testing.T) {
	d := mustSelect(t, mixedPool(), creditExhausted(), basePolicy(UnknownCreditPreferFree))

	if d.Chosen.ID != freeStrong.ID {
		t.Fatalf("chose %q, want %q", d.Chosen.ID, freeStrong.ID)
	}
	if d.Branch != BranchFree {
		t.Fatalf("branch = %q, want free", d.Branch)
	}
	if d.ReasonID != ReasonFreeCreditExhausted {
		t.Fatalf("reason = %q, want %q", d.ReasonID, ReasonFreeCreditExhausted)
	}
	if d.UsedUnknownPriced {
		t.Fatal("unpriced candidate leaked into the no-credit branch")
	}
}

// Unknown credit is never coerced: each policy produces its own documented
// outcome, and the conservative one spends nothing.
func TestSelect_CreditUnknown_FollowsPolicy(t *testing.T) {
	t.Run("prefer_free is the conservative choice", func(t *testing.T) {
		d := mustSelect(t, mixedPool(), UnknownCredit("no balance endpoint"), basePolicy(UnknownCreditPreferFree))
		if d.Chosen.ID != freeStrong.ID {
			t.Fatalf("chose %q, want %q", d.Chosen.ID, freeStrong.ID)
		}
		if d.Branch != BranchFree || d.ReasonID != ReasonFreeUnknownCredit {
			t.Fatalf("branch=%q reason=%q", d.Branch, d.ReasonID)
		}
		if d.CreditState != CreditUnknown {
			t.Fatalf("credit state silently changed to %q", d.CreditState)
		}
	})

	t.Run("prefer_paid opts into spending", func(t *testing.T) {
		d := mustSelect(t, mixedPool(), UnknownCredit("no balance endpoint"), basePolicy(UnknownCreditPreferPaid))
		if d.Chosen.ID != paidStrong.ID {
			t.Fatalf("chose %q, want %q", d.Chosen.ID, paidStrong.ID)
		}
		if d.Branch != BranchPaid || d.ReasonID != ReasonPaidUnknownCredit {
			t.Fatalf("branch=%q reason=%q", d.Branch, d.ReasonID)
		}
	})

	t.Run("reject fails closed", func(t *testing.T) {
		d, err := NewCreditAwareSelector().Select(context.Background(), mixedPool(),
			UnknownCredit("no balance endpoint"), basePolicy(UnknownCreditReject))
		if !errors.Is(err, ErrCreditUnknownRejected) {
			t.Fatalf("err = %v, want ErrCreditUnknownRejected", err)
		}
		if d.Chosen != nil {
			t.Fatalf("fail-closed still returned a model: %+v", d.Chosen)
		}
		if d.ReasonID != ReasonUnknownCreditRejected {
			t.Fatalf("reason = %q", d.ReasonID)
		}
		if d.Considered != 6 {
			t.Fatalf("population counts lost on the error path: %+v", d)
		}
	})
}

// ---------------------------------------------------------------------------
// Degenerate candidate sets
// ---------------------------------------------------------------------------

func TestSelect_EmptyCandidateSet(t *testing.T) {
	for _, credit := range []CreditStatus{creditAvailable(), creditExhausted(), UnknownCredit("n/a")} {
		d, err := NewCreditAwareSelector().Select(context.Background(), nil, credit, basePolicy(UnknownCreditPreferFree))
		if !errors.Is(err, ErrNoCandidates) {
			t.Fatalf("credit %q: err = %v, want ErrNoCandidates", credit.State, err)
		}
		if d.Chosen != nil {
			t.Fatalf("credit %q: empty set produced a model", credit.State)
		}
		if d.ReasonID != ReasonNoCandidates {
			t.Fatalf("credit %q: reason = %q", credit.State, d.ReasonID)
		}
	}
}

func TestSelect_OnlyFreeCandidates(t *testing.T) {
	pool := []Candidate{freeWeak, freeStrong}

	t.Run("no credit picks the strongest free", func(t *testing.T) {
		d := mustSelect(t, pool, creditExhausted(), basePolicy(UnknownCreditPreferFree))
		if d.Chosen.ID != freeStrong.ID {
			t.Fatalf("chose %q, want %q", d.Chosen.ID, freeStrong.ID)
		}
	})

	t.Run("credit available with no paid model refuses by default", func(t *testing.T) {
		d, err := NewCreditAwareSelector().Select(context.Background(), pool, creditAvailable(), basePolicy(UnknownCreditPreferFree))
		if !errors.Is(err, ErrNoEligibleCandidate) {
			t.Fatalf("err = %v, want ErrNoEligibleCandidate", err)
		}
		if d.Chosen != nil {
			t.Fatal("fallback happened without the caller opting in")
		}
	})

	t.Run("credit available falls back to free when opted in", func(t *testing.T) {
		p := basePolicy(UnknownCreditPreferFree)
		p.FallbackToFreeWhenNoPaid = true
		d := mustSelect(t, pool, creditAvailable(), p)
		if d.Chosen.ID != freeStrong.ID {
			t.Fatalf("chose %q, want %q", d.Chosen.ID, freeStrong.ID)
		}
		if !d.FellBack || d.ReasonID != ReasonFellBackToFree {
			t.Fatalf("fallback not recorded: fellBack=%v reason=%q", d.FellBack, d.ReasonID)
		}
		if d.Branch != BranchPaid {
			t.Fatalf("branch should record the ATTEMPTED branch, got %q", d.Branch)
		}
	})
}

func TestSelect_OnlyPaidCandidates(t *testing.T) {
	pool := []Candidate{paidWeak, paidStrong, paidMid}

	t.Run("credit available picks the strongest paid", func(t *testing.T) {
		d := mustSelect(t, pool, creditAvailable(), basePolicy(UnknownCreditPreferFree))
		if d.Chosen.ID != paidStrong.ID {
			t.Fatalf("chose %q, want %q", d.Chosen.ID, paidStrong.ID)
		}
	})

	// The money-safety case: no credit, nothing free. Spending must NOT
	// happen unless the caller explicitly enabled it.
	t.Run("no credit refuses to spend by default", func(t *testing.T) {
		d, err := NewCreditAwareSelector().Select(context.Background(), pool, creditExhausted(), basePolicy(UnknownCreditPreferFree))
		if !errors.Is(err, ErrNoEligibleCandidate) {
			t.Fatalf("err = %v, want ErrNoEligibleCandidate", err)
		}
		if d.Chosen != nil {
			t.Fatalf("selector spent money with no credit: %+v", d.Chosen)
		}
		if d.FreeAvailable != 0 || d.PaidAvailable != 3 {
			t.Fatalf("population counts wrong: %+v", d)
		}
	})

	t.Run("no credit spends only when explicitly opted in", func(t *testing.T) {
		p := basePolicy(UnknownCreditPreferFree)
		p.FallbackToPaidWhenNoFree = true
		d := mustSelect(t, pool, creditExhausted(), p)
		if d.Chosen.ID != paidStrong.ID {
			t.Fatalf("chose %q, want %q", d.Chosen.ID, paidStrong.ID)
		}
		if !d.FellBack || d.ReasonID != ReasonFellBackToPaid {
			t.Fatalf("fallback not recorded: %+v", d)
		}
	})

	// The two fallback knobs must be independent — enabling the free-ward one
	// must not silently enable the money-spending one.
	t.Run("free fallback knob does not enable paid fallback", func(t *testing.T) {
		p := basePolicy(UnknownCreditPreferFree)
		p.FallbackToFreeWhenNoPaid = true
		_, err := NewCreditAwareSelector().Select(context.Background(), pool, creditExhausted(), p)
		if !errors.Is(err, ErrNoEligibleCandidate) {
			t.Fatalf("err = %v, want ErrNoEligibleCandidate; fallback knobs are coupled", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Unknown-priced candidates
// ---------------------------------------------------------------------------

func TestSelect_UnknownPricedCandidates(t *testing.T) {
	t.Run("excluded from both branches by default", func(t *testing.T) {
		pool := []Candidate{pricelessX}
		for _, credit := range []CreditStatus{creditAvailable(), creditExhausted()} {
			_, err := NewCreditAwareSelector().Select(context.Background(), pool, credit, basePolicy(UnknownCreditPreferFree))
			if !errors.Is(err, ErrNoEligibleCandidate) {
				t.Fatalf("credit %q: err = %v, want ErrNoEligibleCandidate", credit.State, err)
			}
		}
	})

	t.Run("admitted as a last resort when allowed", func(t *testing.T) {
		p := basePolicy(UnknownCreditPreferFree)
		p.AllowUnknownPriced = true
		d := mustSelect(t, []Candidate{pricelessX}, creditExhausted(), p)
		if d.Chosen.ID != pricelessX.ID {
			t.Fatalf("chose %q, want %q", d.Chosen.ID, pricelessX.ID)
		}
		if !d.UsedUnknownPriced {
			t.Fatal("UsedUnknownPriced not flagged; the cost of this choice is not known")
		}
	})

	// Even at strength 10.0 — the highest in the pool — an unpriced candidate
	// must never outrank a candidate whose price IS known within the branch.
	t.Run("never outranks a known-priced candidate", func(t *testing.T) {
		p := basePolicy(UnknownCreditPreferFree)
		p.AllowUnknownPriced = true
		d := mustSelect(t, mixedPool(), creditExhausted(), p)
		if d.Chosen.ID != freeStrong.ID {
			t.Fatalf("chose %q, want %q — unpriced candidate outranked a known-priced one", d.Chosen.ID, freeStrong.ID)
		}
		if d.UsedUnknownPriced {
			t.Fatal("UsedUnknownPriced flagged for a known-priced choice")
		}
	})
}

// ---------------------------------------------------------------------------
// Ties in capability ranking
// ---------------------------------------------------------------------------

func TestSelect_TiesAreDeterministic(t *testing.T) {
	a := Candidate{ID: "tie-b", Strength: 8, Price: KnownPrice(9, 9, "USD")}
	b := Candidate{ID: "tie-a", Strength: 8, Price: KnownPrice(2, 2, "USD")}
	c := Candidate{ID: "tie-c", Strength: 8, Price: KnownPrice(2, 2, "USD")}

	t.Run("cheapest wins the tie", func(t *testing.T) {
		p := basePolicy(UnknownCreditPreferFree)
		d := mustSelect(t, []Candidate{a, b, c}, creditAvailable(), p)
		if d.Chosen.ID != "tie-a" {
			t.Fatalf("chose %q, want tie-a (cheapest, then lexical)", d.Chosen.ID)
		}
	})

	t.Run("lexical ID breaks a full tie", func(t *testing.T) {
		p := basePolicy(UnknownCreditPreferFree)
		p.TieBreak = TieBreakNone
		d := mustSelect(t, []Candidate{a, c, b}, creditAvailable(), p)
		if d.Chosen.ID != "tie-a" {
			t.Fatalf("chose %q, want tie-a (lexical)", d.Chosen.ID)
		}
	})

	// Input order must not change the answer. Feeding every rotation of the
	// pool must yield one identical result.
	t.Run("independent of input order", func(t *testing.T) {
		pool := mixedPool()
		var first string
		for i := range pool {
			rotated := append(append([]Candidate{}, pool[i:]...), pool[:i]...)
			d := mustSelect(t, rotated, creditAvailable(), basePolicy(UnknownCreditPreferFree))
			if i == 0 {
				first = d.Chosen.ID
				continue
			}
			if d.Chosen.ID != first {
				t.Fatalf("rotation %d chose %q, first rotation chose %q — selection is order-dependent", i, d.Chosen.ID, first)
			}
		}
	})

	// An unobserved price must not win "cheapest" by counting as zero.
	t.Run("unobserved price loses the cheapest tie-break", func(t *testing.T) {
		unpricedTie := Candidate{ID: "aaa-unpriced", Strength: 8, Price: UnknownPrice()}
		pricedTie := Candidate{ID: "zzz-priced", Strength: 8, Price: KnownPrice(0, 0, "USD")}
		p := basePolicy(UnknownCreditPreferFree)
		p.AllowUnknownPriced = true
		d := mustSelect(t, []Candidate{unpricedTie, pricedTie}, creditExhausted(), p)
		if d.Chosen.ID != pricedTie.ID {
			t.Fatalf("chose %q, want %q — unobserved price won cheapest", d.Chosen.ID, pricedTie.ID)
		}
	})
}

// ---------------------------------------------------------------------------
// Policy / input validation
// ---------------------------------------------------------------------------

func TestSelect_RejectsIncompletePolicy(t *testing.T) {
	cases := []struct {
		name   string
		policy Policy
	}{
		{"missing OnUnknownCredit", Policy{TieBreak: TieBreakCheapest}},
		{"unrecognised OnUnknownCredit", Policy{OnUnknownCredit: "whatever"}},
		{"unrecognised TieBreak", Policy{OnUnknownCredit: UnknownCreditPreferFree, TieBreak: "vibes"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewCreditAwareSelector().Select(context.Background(), mixedPool(), creditAvailable(), tc.policy)
			if !errors.Is(err, ErrPolicyIncomplete) {
				t.Fatalf("err = %v, want ErrPolicyIncomplete", err)
			}
		})
	}
}

func TestSelect_RejectsInvalidCreditStatus(t *testing.T) {
	bogus := CreditStatus{State: CreditAvailable, Signal: CreditSignalNone}
	_, err := NewCreditAwareSelector().Select(context.Background(), mixedPool(), bogus, basePolicy(UnknownCreditPreferFree))
	if !errors.Is(err, ErrInvalidCreditStatus) {
		t.Fatalf("err = %v, want ErrInvalidCreditStatus", err)
	}
}

func TestSelect_HonoursCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewCreditAwareSelector().Select(ctx, mixedPool(), creditAvailable(), basePolicy(UnknownCreditPreferFree))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

// The selector must not mutate or reorder the caller's slice.
func TestSelect_DoesNotMutateInput(t *testing.T) {
	pool := mixedPool()
	before := make([]Candidate, len(pool))
	copy(before, pool)

	mustSelect(t, pool, creditAvailable(), basePolicy(UnknownCreditPreferFree))

	for i := range pool {
		if pool[i] != before[i] {
			t.Fatalf("input slice mutated at index %d: %+v -> %+v", i, before[i], pool[i])
		}
	}
}
