// Package selection provides credit-aware model selection: given a set of
// candidate models, an account's credit status, and a caller-supplied policy,
// it returns the strongest model the account can actually pay for.
//
// # Why this package exists
//
// LLMsVerifier already knows two things independently: what a model COSTS
// (scoring.ModelData.InputTokenCost, scoring.ModelsDevModel.InputCostPer1M,
// enhanced.PricingInfo) and whether a probe hit a billing wall
// (providers.ProbeVerdictQuotaExceeded). Nothing joined them into a decision.
// This package is that join, expressed as a leaf vocabulary with zero
// dependencies on the rest of the module so any consumer can reuse it.
//
// # Decoupling contract (CONST-051(B) / CONST-069)
//
// This package is consumer-agnostic by construction:
//
//   - It never decides what "strongest" means. The caller supplies
//     Candidate.Strength from whatever ranking it trusts (this module's
//     scoring engine, a benchmark table, an operator's ordering).
//   - It never decides what to do when credit is unknown. The caller supplies
//     Policy.OnUnknownCredit.
//   - It never discovers accounts, keys, endpoints, or config paths. Every
//     input arrives as a function argument.
//   - It contains no consumer project name, path, command prefix, or
//     convention.
//
// # The three-state invariant
//
// Both axes are tri-state and neither collapses silently:
//
//	Affordability: free | paid | unknown
//	CreditState:   available | exhausted | unknown
//
// "Unknown" is a first-class value, never coerced. In particular a zero price
// is NOT free unless the price was explicitly observed — see TokenPrice.
package selection

import (
	"errors"
	"fmt"
	"time"
)

// Affordability classifies what it costs to run one request against a model.
type Affordability string

const (
	// AffordabilityFree means the model was OBSERVED to cost nothing per
	// token. It is never inferred from missing data.
	AffordabilityFree Affordability = "free"
	// AffordabilityPaid means the model was OBSERVED to have a positive
	// per-token cost.
	AffordabilityPaid Affordability = "paid"
	// AffordabilityUnknown means the price was not observed, or was observed
	// as nonsensical (negative). A candidate in this state is excluded from
	// both branches unless Policy.AllowUnknownPriced says otherwise, and even
	// then it ranks below every candidate whose price IS known.
	AffordabilityUnknown Affordability = "unknown"
)

// TokenPrice is an explicitly-optional per-million-token price.
//
// The Known flag exists because the rest of this module stores prices as bare
// float64 (scoring.ModelData.InputTokenCost, scoring.ModelsDevModel.
// InputCostPer1M). In those types a free model and a model whose price was
// never fetched are byte-identical zeros. Selecting on that ambiguity would
// route paid traffic to a model believed free. TokenPrice refuses to guess:
// the zero value of TokenPrice is UNKNOWN, not free.
type TokenPrice struct {
	// Known reports whether the prices below were actually observed.
	Known bool `json:"known"`
	// InputPerMillion is the cost of 1M input tokens in Currency.
	InputPerMillion float64 `json:"input_per_million"`
	// OutputPerMillion is the cost of 1M output tokens in Currency.
	OutputPerMillion float64 `json:"output_per_million"`
	// Currency is the ISO-4217-ish code the amounts are denominated in.
	// Informational only: Affordability never compares across currencies.
	Currency string `json:"currency,omitempty"`
}

// KnownPrice builds an observed price. Callers MUST only use this when the
// numbers came from a real catalogue or API response.
func KnownPrice(inputPerMillion, outputPerMillion float64, currency string) TokenPrice {
	return TokenPrice{
		Known:            true,
		InputPerMillion:  inputPerMillion,
		OutputPerMillion: outputPerMillion,
		Currency:         currency,
	}
}

// UnknownPrice builds an unobserved price. Equivalent to the zero value; it
// exists so call sites read as a deliberate statement of ignorance.
func UnknownPrice() TokenPrice { return TokenPrice{} }

// Affordability derives the free/paid/unknown class from the price.
//
// Rules, in order:
//   - price not observed  -> unknown
//   - any component < 0   -> unknown (corrupt data, not a discount)
//   - both components = 0 -> free
//   - otherwise           -> paid
func (p TokenPrice) Affordability() Affordability {
	if !p.Known {
		return AffordabilityUnknown
	}
	if p.InputPerMillion < 0 || p.OutputPerMillion < 0 {
		return AffordabilityUnknown
	}
	if p.InputPerMillion == 0 && p.OutputPerMillion == 0 {
		return AffordabilityFree
	}
	return AffordabilityPaid
}

// Total is the sum of the observed per-million components, used only as a
// tie-break. It returns ok=false when the price was never observed, so a
// caller can never mistake an unobserved price for a cost of zero.
func (p TokenPrice) Total() (total float64, ok bool) {
	if !p.Known {
		return 0, false
	}
	return p.InputPerMillion + p.OutputPerMillion, true
}

// CreditState is whether an account can currently pay for paid inference.
type CreditState string

const (
	// CreditAvailable means a signal positively showed spendable credit.
	CreditAvailable CreditState = "available"
	// CreditExhausted means a signal positively showed credit is gone or the
	// account is barred from paid inference (balance <= 0, HTTP 402, a
	// quota/subscription-cap verdict).
	CreditExhausted CreditState = "exhausted"
	// CreditUnknown means no signal determined the state — no balance
	// endpoint, a transient probe failure, or nothing was checked. It is
	// never silently treated as either of the other two.
	CreditUnknown CreditState = "unknown"
)

// CreditSignal records HOW a CreditState was determined, so a consumer can
// weigh a hard balance reading differently from an inference drawn off one
// probe response.
type CreditSignal string

const (
	// CreditSignalNone means nothing determined the state. It is the only
	// legal signal for CreditUnknown and is illegal for the other states.
	CreditSignalNone CreditSignal = "none"
	// CreditSignalBalanceEndpoint means a provider balance/credits endpoint
	// returned a number.
	CreditSignalBalanceEndpoint CreditSignal = "balance_endpoint"
	// CreditSignalProbeResponse means the state was inferred from how the
	// provider answered a real inference probe (e.g. HTTP 402 => exhausted).
	CreditSignalProbeResponse CreditSignal = "probe_response"
	// CreditSignalOperatorDeclared means a human or configuration asserted
	// the state. Recorded distinctly because it is an assertion, not a
	// measurement.
	CreditSignalOperatorDeclared CreditSignal = "operator_declared"
)

// Balance is an observed account balance.
type Balance struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency,omitempty"`
}

// CreditStatus is a credit determination plus its provenance.
//
// The invariant enforced by Validate is: a decided state REQUIRES a signal.
// A CreditStatus claiming CreditAvailable with CreditSignalNone is a bluff —
// something decided without evidence — and is rejected.
type CreditStatus struct {
	State  CreditState  `json:"state"`
	Signal CreditSignal `json:"signal"`
	// Balance is populated only when the signal actually carried a number.
	Balance *Balance `json:"balance,omitempty"`
	// Detail is free text describing the evidence (status line, endpoint,
	// verdict). Never parsed; for humans and audit records.
	Detail string `json:"detail,omitempty"`
	// ObservedAt is when the signal was taken. Zero means not recorded.
	ObservedAt time.Time `json:"observed_at,omitempty"`
}

// UnknownCredit builds the honest "nothing was determined" status.
func UnknownCredit(detail string) CreditStatus {
	return CreditStatus{State: CreditUnknown, Signal: CreditSignalNone, Detail: detail}
}

// CreditFromBalance builds a status from an observed balance reading.
// A strictly positive amount means available; zero or negative means
// exhausted. Both are decided states because the number was actually read.
func CreditFromBalance(amount float64, currency string, observedAt time.Time) CreditStatus {
	state := CreditExhausted
	if amount > 0 {
		state = CreditAvailable
	}
	return CreditStatus{
		State:      state,
		Signal:     CreditSignalBalanceEndpoint,
		Balance:    &Balance{Amount: amount, Currency: currency},
		Detail:     fmt.Sprintf("balance=%g %s", amount, currency),
		ObservedAt: observedAt,
	}
}

// DeclaredCredit builds a status from an operator/config assertion.
// state must be CreditAvailable or CreditExhausted; anything else yields an
// unknown status, because "the operator declared it unknown" carries no
// evidence and must not masquerade as a decided state.
func DeclaredCredit(state CreditState, detail string, observedAt time.Time) CreditStatus {
	if state != CreditAvailable && state != CreditExhausted {
		return UnknownCredit(detail)
	}
	return CreditStatus{
		State:      state,
		Signal:     CreditSignalOperatorDeclared,
		Detail:     detail,
		ObservedAt: observedAt,
	}
}

// Validate enforces the state/signal invariant.
func (c CreditStatus) Validate() error {
	switch c.State {
	case CreditUnknown:
		if c.Signal != CreditSignalNone && c.Signal != "" {
			return fmt.Errorf("%w: state=%s carries signal=%s", ErrInvalidCreditStatus, c.State, c.Signal)
		}
		return nil
	case CreditAvailable, CreditExhausted:
		if c.Signal == CreditSignalNone || c.Signal == "" {
			return fmt.Errorf("%w: state=%s decided with no signal", ErrInvalidCreditStatus, c.State)
		}
		return nil
	default:
		return fmt.Errorf("%w: unrecognised state %q", ErrInvalidCreditStatus, c.State)
	}
}

// Candidate is one model under consideration.
//
// Strength is supplied by the caller and is the ONLY notion of "stronger"
// this package has. Higher is stronger. This package deliberately does not
// compute, normalise, or second-guess it — that is what keeps the selector
// reusable across consumers with incompatible ranking systems.
type Candidate struct {
	ID       string     `json:"id"`
	Provider string     `json:"provider,omitempty"`
	Strength float64    `json:"strength"`
	Price    TokenPrice `json:"price"`
}

// Affordability is the candidate's free/paid/unknown class.
func (c Candidate) Affordability() Affordability { return c.Price.Affordability() }

// Branch is which side of the free/paid split a decision was taken on.
type Branch string

const (
	// BranchPaid means the selector looked for a paid model.
	BranchPaid Branch = "paid"
	// BranchFree means the selector looked for a free model.
	BranchFree Branch = "free"
	// BranchNone means no branch was reached (no candidates, or the policy
	// refused to proceed on unknown credit).
	BranchNone Branch = "none"
)

// TieBreak selects the secondary ordering applied when two candidates have
// identical Strength. A final lexical-by-ID ordering is ALWAYS applied after
// this, so selection is fully deterministic regardless of input order.
type TieBreak string

const (
	// TieBreakCheapest prefers the lower observed total price. Candidates
	// with unobserved prices lose this comparison rather than winning it by
	// counting as zero.
	TieBreakCheapest TieBreak = "cheapest"
	// TieBreakNone applies no secondary ordering; ties fall straight through
	// to the lexical-by-ID tiebreak.
	TieBreakNone TieBreak = "none"
)

// UnknownCreditPolicy is the caller's instruction for CreditUnknown. There is
// no default: the selector rejects an empty value so no consumer's money is
// spent, and no consumer's request is downgraded, by this package's guess.
type UnknownCreditPolicy string

const (
	// UnknownCreditPreferFree takes the free branch — conservative, spends
	// nothing that was not confirmed affordable.
	UnknownCreditPreferFree UnknownCreditPolicy = "prefer_free"
	// UnknownCreditPreferPaid takes the paid branch — for consumers that
	// would rather attempt the strongest model and handle a billing refusal.
	UnknownCreditPreferPaid UnknownCreditPolicy = "prefer_paid"
	// UnknownCreditReject refuses to select at all, returning
	// ErrCreditUnknownRejected — fail-closed for consumers that must resolve
	// credit before proceeding.
	UnknownCreditReject UnknownCreditPolicy = "reject"
)

// Policy carries every decision this package refuses to make on the caller's
// behalf.
type Policy struct {
	// OnUnknownCredit is required. Empty is rejected by Validate.
	OnUnknownCredit UnknownCreditPolicy `json:"on_unknown_credit"`

	// TieBreak orders candidates of equal Strength. Empty means TieBreakNone.
	TieBreak TieBreak `json:"tie_break,omitempty"`

	// AllowUnknownPriced admits candidates whose price was never observed as
	// a LAST RESORT within the chosen branch: they are considered only when
	// that branch holds no candidate of known price, and they never outrank
	// one that does, however strong they are.
	AllowUnknownPriced bool `json:"allow_unknown_priced"`

	// FallbackToFreeWhenNoPaid lets a paid-branch selection fall back to the
	// strongest free model when no paid candidate exists. Downgrading costs
	// nothing, so consumers commonly enable it.
	FallbackToFreeWhenNoPaid bool `json:"fallback_to_free_when_no_paid"`

	// FallbackToPaidWhenNoFree lets a free-branch selection fall back to a
	// paid model when no free candidate exists. This SPENDS MONEY the credit
	// signal said was unavailable, so it is off by default and must be opted
	// into explicitly. The two fallbacks are separate knobs precisely so this
	// asymmetry is the caller's choice and not a hidden default.
	FallbackToPaidWhenNoFree bool `json:"fallback_to_paid_when_no_free"`
}

// Validate reports whether the policy is complete enough to act on.
func (p Policy) Validate() error {
	switch p.OnUnknownCredit {
	case UnknownCreditPreferFree, UnknownCreditPreferPaid, UnknownCreditReject:
	case "":
		return fmt.Errorf("%w: OnUnknownCredit is required", ErrPolicyIncomplete)
	default:
		return fmt.Errorf("%w: unrecognised OnUnknownCredit %q", ErrPolicyIncomplete, p.OnUnknownCredit)
	}
	switch p.TieBreak {
	case TieBreakCheapest, TieBreakNone, "":
	default:
		return fmt.Errorf("%w: unrecognised TieBreak %q", ErrPolicyIncomplete, p.TieBreak)
	}
	return nil
}

// Decision is the full, auditable outcome of a selection.
type Decision struct {
	// Chosen is nil when no model could be selected.
	Chosen *Candidate `json:"chosen,omitempty"`
	// Branch is the side the selector committed to.
	Branch Branch `json:"branch"`
	// FellBack reports that Branch had no usable candidate and the opposite
	// branch supplied Chosen instead.
	FellBack bool `json:"fell_back"`
	// UsedUnknownPriced reports that Chosen's price was never observed, so
	// its real cost is not known.
	UsedUnknownPriced bool `json:"used_unknown_priced"`

	// CreditState / CreditSignal echo the inputs that drove the branch, so a
	// stored decision explains itself without the original CreditStatus.
	CreditState  CreditState  `json:"credit_state"`
	CreditSignal CreditSignal `json:"credit_signal"`

	// ReasonID is a stable i18n message ID; Reason is its rendering under the
	// active translator. Tests assert ReasonID, never English text.
	ReasonID string `json:"reason_id"`
	Reason   string `json:"reason"`

	// Population counts of the candidate set, for audit and for detecting a
	// selector that silently saw nothing.
	Considered    int `json:"considered"`
	PaidAvailable int `json:"paid_available"`
	FreeAvailable int `json:"free_available"`
	UnknownPriced int `json:"unknown_priced"`
}

// Selection errors. All are sentinels usable with errors.Is.
var (
	// ErrNoCandidates means the candidate set was empty.
	ErrNoCandidates = errors.New("selection: no candidates supplied")
	// ErrNoEligibleCandidate means candidates existed but none was usable
	// under the resolved branch and policy.
	ErrNoEligibleCandidate = errors.New("selection: no eligible candidate for the resolved branch")
	// ErrCreditUnknownRejected means credit was unknown and the policy chose
	// to fail closed rather than guess.
	ErrCreditUnknownRejected = errors.New("selection: credit state unknown and policy rejects unknown credit")
	// ErrPolicyIncomplete means the caller left a required policy input unset
	// or supplied an unrecognised value.
	ErrPolicyIncomplete = errors.New("selection: policy incomplete")
	// ErrInvalidCreditStatus means the credit status violated the
	// state-requires-signal invariant.
	ErrInvalidCreditStatus = errors.New("selection: invalid credit status")
)
