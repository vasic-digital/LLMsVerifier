package selection

import (
	"context"
	"sort"
)

// Message IDs for Decision.ReasonID. Declared as constants so tests assert
// identifiers rather than English literals (CONST-035 anti-bluff: a test
// asserting English would still pass if the call site bypassed i18n).
const (
	ReasonPaidCreditAvailable   = "llmsverifier_selection_reason_paid_credit_available"
	ReasonFreeCreditExhausted   = "llmsverifier_selection_reason_free_credit_exhausted"
	ReasonFreeUnknownCredit     = "llmsverifier_selection_reason_free_unknown_credit"
	ReasonPaidUnknownCredit     = "llmsverifier_selection_reason_paid_unknown_credit"
	ReasonFellBackToFree        = "llmsverifier_selection_reason_fell_back_to_free"
	ReasonFellBackToPaid        = "llmsverifier_selection_reason_fell_back_to_paid"
	ReasonNoCandidates          = "llmsverifier_selection_reason_no_candidates"
	ReasonNoEligibleCandidate   = "llmsverifier_selection_reason_no_eligible_candidate"
	ReasonUnknownCreditRejected = "llmsverifier_selection_reason_unknown_credit_rejected"
)

// Selector chooses one model from a candidate set given a credit status and a
// policy. It is an interface so consumers can substitute their own strategy
// while keeping this package's vocabulary.
type Selector interface {
	Select(ctx context.Context, candidates []Candidate, credit CreditStatus, policy Policy) (Decision, error)
}

// CreditAwareSelector implements the free/paid split:
//
//	credit available -> strongest PAID model
//	credit exhausted -> strongest FREE model
//	credit unknown   -> whatever policy.OnUnknownCredit says, never a guess
//
// It is stateless and safe for concurrent use.
type CreditAwareSelector struct{}

// NewCreditAwareSelector returns the default credit-aware selector.
func NewCreditAwareSelector() *CreditAwareSelector { return &CreditAwareSelector{} }

// compile-time interface satisfaction check.
var _ Selector = (*CreditAwareSelector)(nil)

// Select returns the strongest affordable candidate.
//
// On error the returned Decision is still populated with the branch, credit
// echo, population counts and a ReasonID, so a caller can log WHY nothing was
// selected without re-deriving it.
func (s *CreditAwareSelector) Select(ctx context.Context, candidates []Candidate, credit CreditStatus, policy Policy) (Decision, error) {
	if err := ctx.Err(); err != nil {
		return Decision{Branch: BranchNone}, err
	}
	if err := policy.Validate(); err != nil {
		return Decision{Branch: BranchNone}, err
	}
	if err := credit.Validate(); err != nil {
		return Decision{Branch: BranchNone}, err
	}

	paid, free, unknown := partition(candidates)

	decision := Decision{
		Branch:        BranchNone,
		CreditState:   credit.State,
		CreditSignal:  credit.Signal,
		Considered:    len(candidates),
		PaidAvailable: len(paid),
		FreeAvailable: len(free),
		UnknownPriced: len(unknown),
	}
	if decision.CreditSignal == "" {
		decision.CreditSignal = CreditSignalNone
	}

	if len(candidates) == 0 {
		return withReason(decision, ReasonNoCandidates), ErrNoCandidates
	}

	branch, reasonID, err := resolveBranch(credit.State, policy.OnUnknownCredit)
	if err != nil {
		return withReason(decision, reasonID), err
	}
	decision.Branch = branch

	primary, secondary := paid, free
	fallbackAllowed := policy.FallbackToFreeWhenNoPaid
	fallbackReason := ReasonFellBackToFree
	if branch == BranchFree {
		primary, secondary = free, paid
		fallbackAllowed = policy.FallbackToPaidWhenNoFree
		fallbackReason = ReasonFellBackToPaid
	}

	// Primary branch, known prices first.
	if best, ok := strongest(primary, policy.TieBreak); ok {
		decision.Chosen = &best
		return withReason(decision, reasonID), nil
	}

	// Primary branch, unknown-priced last resort (opt-in). Admitted only
	// because the branch held no candidate of known price.
	if policy.AllowUnknownPriced {
		if best, ok := strongest(unknown, policy.TieBreak); ok {
			decision.Chosen = &best
			decision.UsedUnknownPriced = true
			return withReason(decision, reasonID), nil
		}
	}

	// Opposite branch, only when the caller opted into that direction.
	if fallbackAllowed {
		if best, ok := strongest(secondary, policy.TieBreak); ok {
			decision.Chosen = &best
			decision.FellBack = true
			return withReason(decision, fallbackReason), nil
		}
	}

	return withReason(decision, ReasonNoEligibleCandidate), ErrNoEligibleCandidate
}

// resolveBranch maps a credit state plus the caller's unknown-credit policy to
// the branch to search. It is the only place credit state influences the
// outcome, and it never invents a state that was not supplied.
func resolveBranch(state CreditState, onUnknown UnknownCreditPolicy) (Branch, string, error) {
	switch state {
	case CreditAvailable:
		return BranchPaid, ReasonPaidCreditAvailable, nil
	case CreditExhausted:
		return BranchFree, ReasonFreeCreditExhausted, nil
	case CreditUnknown:
		switch onUnknown {
		case UnknownCreditPreferFree:
			return BranchFree, ReasonFreeUnknownCredit, nil
		case UnknownCreditPreferPaid:
			return BranchPaid, ReasonPaidUnknownCredit, nil
		case UnknownCreditReject:
			return BranchNone, ReasonUnknownCreditRejected, ErrCreditUnknownRejected
		}
	}
	// Unreachable while CreditStatus.Validate and Policy.Validate run first;
	// kept as a fail-closed guard rather than a silent default branch.
	return BranchNone, ReasonUnknownCreditRejected, ErrInvalidCreditStatus
}

// partition splits candidates by affordability class. Ordering within each
// bucket is irrelevant — strongest() imposes a total order.
func partition(candidates []Candidate) (paid, free, unknown []Candidate) {
	for _, c := range candidates {
		switch c.Affordability() {
		case AffordabilityPaid:
			paid = append(paid, c)
		case AffordabilityFree:
			free = append(free, c)
		default:
			unknown = append(unknown, c)
		}
	}
	return paid, free, unknown
}

// strongest returns the highest-Strength candidate under a total order that is
// independent of input ordering:
//
//  1. Strength, descending.
//  2. The policy tie-break.
//  3. ID, ascending — always applied last so the result is deterministic even
//     for candidates identical in every other respect.
func strongest(pool []Candidate, tb TieBreak) (Candidate, bool) {
	if len(pool) == 0 {
		return Candidate{}, false
	}
	ranked := make([]Candidate, len(pool))
	copy(ranked, pool)

	sort.SliceStable(ranked, func(i, j int) bool {
		a, b := ranked[i], ranked[j]
		if a.Strength != b.Strength {
			return a.Strength > b.Strength
		}
		if tb == TieBreakCheapest {
			at, aok := a.Price.Total()
			bt, bok := b.Price.Total()
			switch {
			case aok && bok && at != bt:
				return at < bt
			case aok && !bok:
				// A known price beats an unobserved one; an unobserved price
				// must never win "cheapest" by counting as zero.
				return true
			case !aok && bok:
				return false
			}
		}
		return a.ID < b.ID
	})
	return ranked[0], true
}

// withReason stamps the message ID and its rendering onto a decision.
func withReason(d Decision, reasonID string) Decision {
	d.ReasonID = reasonID
	d.Reason = tr(reasonID)
	return d
}
