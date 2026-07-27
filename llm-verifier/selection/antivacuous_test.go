package selection

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// antivacuous_test.go implements the CONST-035 anti-bluff requirement for
// this capability: a test suite that a DEGENERATE selector could still pass
// is not evidence of anything.
//
// Two degenerate shapes are guarded against:
//
//	1. a selector that always returns the same model regardless of credit
//	   state (it would satisfy every "did it return something" assertion);
//	2. a selector that returns nothing (it would satisfy every "did it avoid
//	   spending money" assertion).
//
// The guard below rejects both. Crucially, the guard is itself proven to have
// teeth: TestAntiVacuous_GuardRejectsConstantSelector and
// TestAntiVacuous_GuardRejectsEmptySelector run it against deliberately
// broken selectors and REQUIRE it to fail them. A guard that never fails is
// the same bluff one level up.

// vacuityScenario is one row of the discrimination matrix: a credit state
// that must steer the selector to a specific, different answer.
type vacuityScenario struct {
	name       string
	credit     CreditStatus
	policy     Policy
	wantChosen string
}

// discriminationMatrix is built so that a correct selector MUST return three
// different models across the rows. Any selector collapsing to one answer, or
// to no answer, fails.
func discriminationMatrix() []vacuityScenario {
	freeOnly := basePolicy(UnknownCreditPreferFree)

	preferPaid := basePolicy(UnknownCreditPreferPaid)

	paidFallback := basePolicy(UnknownCreditPreferFree)
	paidFallback.FallbackToPaidWhenNoFree = true

	return []vacuityScenario{
		{
			name:       "credit available selects strongest paid",
			credit:     creditAvailable(),
			policy:     freeOnly,
			wantChosen: paidStrong.ID,
		},
		{
			name:       "credit exhausted selects strongest free",
			credit:     creditExhausted(),
			policy:     freeOnly,
			wantChosen: freeStrong.ID,
		},
		{
			name:       "unknown credit with prefer_paid selects paid",
			credit:     UnknownCredit("nothing checked"),
			policy:     preferPaid,
			wantChosen: paidStrong.ID,
		},
		{
			name:       "unknown credit with prefer_free selects free",
			credit:     UnknownCredit("nothing checked"),
			policy:     freeOnly,
			wantChosen: freeStrong.ID,
		},
	}
}

// guardSelectorDiscriminates runs the matrix against a selector and returns an
// error describing the first way it proved vacuous. It returns an error rather
// than calling t.Fatal so the sabotage tests below can assert that it fires.
//
// Three independent checks:
//
//	(a) every scenario must yield a model — an always-empty selector fails;
//	(b) every scenario must yield the CORRECT model;
//	(c) the set of distinct answers must have more than one member — a
//	    constant selector fails even if it happens to be right on one row.
func guardSelectorDiscriminates(s Selector) error {
	distinct := map[string]struct{}{}

	for _, sc := range discriminationMatrix() {
		d, err := s.Select(context.Background(), mixedPool(), sc.credit, sc.policy)
		if err != nil {
			return fmt.Errorf("scenario %q: unexpected error %w", sc.name, err)
		}
		if d.Chosen == nil {
			return fmt.Errorf("scenario %q: selector returned no model (vacuous-empty)", sc.name)
		}
		if d.Chosen.ID != sc.wantChosen {
			return fmt.Errorf("scenario %q: chose %q, want %q", sc.name, d.Chosen.ID, sc.wantChosen)
		}
		distinct[d.Chosen.ID] = struct{}{}
	}

	if len(distinct) < 2 {
		ids := make([]string, 0, len(distinct))
		for id := range distinct {
			ids = append(ids, id)
		}
		return fmt.Errorf("selector returned only %d distinct model(s) %v across %d scenarios "+
			"(vacuous-constant: credit state had no effect)", len(distinct), ids, len(discriminationMatrix()))
	}
	return nil
}

// TestAntiVacuous_RealSelectorPassesGuard proves the shipped selector actually
// discriminates on credit state.
func TestAntiVacuous_RealSelectorPassesGuard(t *testing.T) {
	if err := guardSelectorDiscriminates(NewCreditAwareSelector()); err != nil {
		t.Fatalf("production selector failed the anti-vacuity guard: %v", err)
	}
}

// constantSelector always returns the same model. Permitted here because this
// is a unit-test source (CONST-050(A)); it exists solely to prove the guard
// has teeth and is never reachable from production code.
type constantSelector struct{ id string }

func (c constantSelector) Select(_ context.Context, candidates []Candidate, _ CreditStatus, _ Policy) (Decision, error) {
	for i := range candidates {
		if candidates[i].ID == c.id {
			chosen := candidates[i]
			return Decision{Chosen: &chosen, Branch: BranchPaid}, nil
		}
	}
	return Decision{Branch: BranchNone}, ErrNoEligibleCandidate
}

// emptySelector never returns a model.
type emptySelector struct{}

func (emptySelector) Select(_ context.Context, _ []Candidate, _ CreditStatus, _ Policy) (Decision, error) {
	return Decision{Branch: BranchNone}, nil
}

// TestAntiVacuous_GuardRejectsConstantSelector proves the guard fails a
// selector that ignores credit state. Without this, TestAntiVacuous_Real...
// would be unfalsifiable.
func TestAntiVacuous_GuardRejectsConstantSelector(t *testing.T) {
	err := guardSelectorDiscriminates(constantSelector{id: paidStrong.ID})
	if err == nil {
		t.Fatal("guard accepted a constant selector; the anti-vacuity guard has no teeth")
	}
	if !strings.Contains(err.Error(), "want") && !strings.Contains(err.Error(), "vacuous-constant") {
		t.Fatalf("guard failed for an unrelated reason: %v", err)
	}
	t.Logf("guard correctly rejected constant selector: %v", err)
}

// TestAntiVacuous_GuardRejectsEmptySelector proves the guard fails a selector
// that returns nothing at all.
func TestAntiVacuous_GuardRejectsEmptySelector(t *testing.T) {
	err := guardSelectorDiscriminates(emptySelector{})
	if err == nil {
		t.Fatal("guard accepted an always-empty selector; the anti-vacuity guard has no teeth")
	}
	if !strings.Contains(err.Error(), "vacuous-empty") {
		t.Fatalf("guard failed for an unrelated reason: %v", err)
	}
	t.Logf("guard correctly rejected empty selector: %v", err)
}

// TestAntiVacuous_GuardRejectsInvertedSelector proves the guard also catches a
// selector that discriminates on credit state but the WRONG way round —
// spending when there is no credit and economising when there is. Distinctness
// alone would not catch this; correctness per row does.
type invertedSelector struct{}

func (invertedSelector) Select(ctx context.Context, candidates []Candidate, credit CreditStatus, policy Policy) (Decision, error) {
	flipped := credit
	switch credit.State {
	case CreditAvailable:
		flipped = CreditFromBalance(0, "USD", credit.ObservedAt)
	case CreditExhausted:
		flipped = CreditFromBalance(1, "USD", credit.ObservedAt)
	}
	return NewCreditAwareSelector().Select(ctx, candidates, flipped, policy)
}

func TestAntiVacuous_GuardRejectsInvertedSelector(t *testing.T) {
	err := guardSelectorDiscriminates(invertedSelector{})
	if err == nil {
		t.Fatal("guard accepted a credit-inverted selector; it only checks distinctness, not correctness")
	}
	t.Logf("guard correctly rejected inverted selector: %v", err)
}

// TestAntiVacuous_ErrorPathsAreDistinguishable proves the failure sentinels
// are not one collapsed error: a caller must be able to tell "you gave me
// nothing" from "nothing was affordable" from "resolve your credit first".
func TestAntiVacuous_ErrorPathsAreDistinguishable(t *testing.T) {
	sel := NewCreditAwareSelector()
	ctx := context.Background()

	_, errEmpty := sel.Select(ctx, nil, creditAvailable(), basePolicy(UnknownCreditPreferFree))
	_, errNone := sel.Select(ctx, []Candidate{paidStrong}, creditExhausted(), basePolicy(UnknownCreditPreferFree))
	_, errUnknown := sel.Select(ctx, mixedPool(), UnknownCredit("x"), basePolicy(UnknownCreditReject))

	if errors.Is(errEmpty, ErrNoEligibleCandidate) || errors.Is(errNone, ErrNoCandidates) {
		t.Fatal("ErrNoCandidates and ErrNoEligibleCandidate are not distinguishable")
	}
	if errors.Is(errUnknown, ErrNoCandidates) || errors.Is(errUnknown, ErrNoEligibleCandidate) {
		t.Fatal("ErrCreditUnknownRejected collapses into another sentinel")
	}
	for name, err := range map[string]error{"empty": errEmpty, "none": errNone, "unknown": errUnknown} {
		if err == nil {
			t.Fatalf("%s path returned nil error", name)
		}
	}
}
