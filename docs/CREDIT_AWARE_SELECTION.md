# LLMsVerifier Credit-Aware Model Selection

The credit-aware selection system answers a question the scoring engine alone
cannot: **"which model should this account actually run, given what it can
afford right now?"**

The rule it implements is simple to state and easy to get subtly wrong:

> When the account has usable credit, run the strongest **paid** model.
> When it does not, run the strongest **free** model.
> When nobody knows, do not guess — do what the caller's policy says.

## Overview

LLMsVerifier already knew two things independently:

- What a model **costs** — `scoring.ModelData.InputTokenCost`,
  `scoring.ModelsDevModel.InputCostPer1M`, `enhanced.PricingInfo`.
- Whether a probe hit a **billing wall** — `providers.ProbeVerdictQuotaExceeded`
  (HTTP 402 *"Subscription usage cap exceeded. Please add balance to
  continue."*), distinct from the transient `ProbeVerdictRateLimited`.

Nothing joined them into a decision. The `selection` package is that join.

## Package Structure

```
LLMsVerifier/llm-verifier/
├── selection/                      # leaf package — stdlib + pkg/i18n only
│   ├── types.go                    # Affordability, TokenPrice, CreditStatus,
│   │                               #   Candidate, Policy, Decision, errors
│   ├── selector.go                 # Selector interface + CreditAwareSelector
│   ├── credit.go                   # CreditReporter, BalanceEndpointReporter,
│   │                               #   StaticCreditReporter
│   ├── i18n.go                     # CONST-046 translator seam
│   ├── selection_test.go           # unit matrix
│   ├── credit_endpoint_test.go     # real HTTP (httptest loopback)
│   ├── antivacuous_test.go         # anti-bluff guards with proven teeth
│   └── i18n_migration_test.go      # reason-rendering routes through i18n
├── providers/credit_signal.go      # ProbeOutcome  → selection.CreditStatus
└── scoring/selection_adapter.go    # ModelData/Score → selection.Candidate
```

`selection` is deliberately a **leaf**: it imports nothing from the rest of the
module. The adapters live in the packages that own the source types, so the
dependency arrow always points *into* `selection` and never out.

## The Two Tri-State Axes

Neither axis collapses silently. "Unknown" is a value, not a missing value.

| Axis | Values |
|------|--------|
| `Affordability` | `free` · `paid` · `unknown` |
| `CreditState` | `available` · `exhausted` · `unknown` |

### Affordability — why `TokenPrice` carries a `Known` flag

The rest of the module stores prices as bare `float64`. In those types a
genuinely free model and a model whose price was never fetched are **both
stored as `0`**. Selecting on that ambiguity would classify every price-less
record as free and route paid traffic to it.

`TokenPrice` refuses to guess. **The zero value of `TokenPrice` is UNKNOWN, not
free.**

```go
selection.UnknownPrice().Affordability()        // "unknown"
selection.KnownPrice(0, 0, "USD").Affordability() // "free"   — observed zero
selection.KnownPrice(3, 15, "USD").Affordability() // "paid"
selection.KnownPrice(-1, 5, "USD").Affordability() // "unknown" — corrupt, not a discount
```

### Credit state — and the signal that determined it

A `CreditStatus` records *how* it was decided, so a consumer can weigh a hard
balance reading differently from an inference off a single probe.

| `CreditSignal` | Meaning |
|----------------|---------|
| `balance_endpoint` | A provider balance/credits endpoint returned a number. |
| `probe_response` | Inferred from how the provider answered a real inference probe. |
| `operator_declared` | A human or configuration asserted it. Recorded distinctly because it is an assertion, not a measurement. |
| `none` | Nothing determined it. The only legal signal for `unknown`. |

`CreditStatus.Validate()` enforces the invariant **a decided state requires a
signal**. A status claiming `available` with signal `none` is a bluff — a
decision made without evidence — and the selector rejects it.

## Obtaining a Credit Status

### From a balance endpoint (real HTTP)

Every input is caller-supplied. The package ships **no provider list, no
default URL, no default field name** — a bundled default would embed one
consumer's provider set into shared infrastructure (CONST-069).

```go
reporter := &selection.BalanceEndpointReporter{
    Config: selection.BalanceEndpointConfig{
        URL:          "https://api.example.com/v1/credits",
        Headers:      map[string]string{"Authorization": "Bearer " + key},
        AmountPath:   "data/total_available",   // slash-separated JSON path
        CurrencyPath: "data/unit",
    },
}
status, err := reporter.CreditStatus(ctx, accountID)
```

Failure is never coerced into a decision. A transport error, a non-2xx status,
unparseable JSON, or a missing/non-numeric field all yield `CreditUnknown` with
the reason in `Detail`. Only an actual number read from the response produces
`available` (> 0) or `exhausted` (≤ 0). HTTP **402** is the one status that is
itself a positive statement of inability to pay, and maps to `exhausted`.

### From a probe response

`providers.CreditStatusFromProbeOutcome` wires the existing probe classifier
into the credit vocabulary. It takes the probed model's affordability, and that
argument is the whole point:

| Probe verdict | Probed model | Result |
|---------------|--------------|--------|
| `quota_exceeded` | any | `exhausted` |
| `ok` | **paid** | `available` — a billable request was served |
| `ok` | free or unknown | **`unknown`** — proves nothing about paid credit |
| `rate_limited` | any | `unknown` — transient, not a credit fact |
| `failed` | any | `unknown` — auth/invalid/server, not a credit fact |

A `200 OK` from a *free* model says nothing whatsoever about paid credit.
Treating it as proof would route paid traffic on the strength of a free
request.

## Selecting

```go
candidates := []selection.Candidate{
    {ID: "big-paid",  Strength: 9.5, Price: selection.KnownPrice(3, 15, "USD")},
    {ID: "big-free",  Strength: 7.0, Price: selection.KnownPrice(0, 0, "USD")},
    {ID: "unpriced",  Strength: 10.0, Price: selection.UnknownPrice()},
}

policy := selection.Policy{
    OnUnknownCredit:          selection.UnknownCreditPreferFree, // required
    TieBreak:                 selection.TieBreakCheapest,
    FallbackToFreeWhenNoPaid: true,
}

decision, err := selection.NewCreditAwareSelector().
    Select(ctx, candidates, status, policy)
```

### `Candidate.Strength` is the caller's

This package **never decides what "strongest" means**. Higher is stronger; the
number comes from whatever ranking the caller trusts — this module's scoring
engine, a benchmark table, an operator's ordering. That is what keeps the
selector reusable across consumers with incompatible ranking systems.

### `Policy` — everything the package refuses to decide for you

| Field | Effect |
|-------|--------|
| `OnUnknownCredit` | **Required.** `prefer_free` (conservative) · `prefer_paid` · `reject` (fail closed, returns `ErrCreditUnknownRejected`). Empty is rejected by `Validate()` — no default, so no consumer's money is spent by this package's guess. |
| `TieBreak` | `cheapest` or `none`. A final lexical-by-ID ordering is **always** applied after it, so selection is deterministic regardless of input order. |
| `AllowUnknownPriced` | Admits unpriced candidates as a **last resort** within the chosen branch: considered only when that branch holds no candidate of known price, and never outranking one that does, however strong. |
| `FallbackToFreeWhenNoPaid` | Paid branch may fall back to free. Downgrading costs nothing. |
| `FallbackToPaidWhenNoFree` | Free branch may fall back to paid. **This spends money the credit signal said was unavailable**, so it is off by default and must be opted into explicitly. |

The two fallback knobs are separate precisely so that asymmetry is the caller's
choice and not a hidden default.

### `Decision` — an auditable outcome

```go
type Decision struct {
    Chosen            *Candidate   // nil when nothing could be selected
    Branch            Branch       // the branch ATTEMPTED: paid | free | none
    FellBack          bool         // Chosen came from the opposite branch
    UsedUnknownPriced bool         // Chosen's real cost is not known
    CreditState       CreditState  // echoed so a stored decision explains itself
    CreditSignal      CreditSignal
    ReasonID          string       // stable i18n message ID
    Reason            string       // rendering under the active translator
    Considered, PaidAvailable, FreeAvailable, UnknownPriced int
}
```

Population counts are filled in **even on the error paths**, so a caller can log
why nothing was selected without re-deriving it.

### Errors

| Sentinel | Condition |
|----------|-----------|
| `ErrNoCandidates` | The candidate set was empty. |
| `ErrNoEligibleCandidate` | Candidates existed, none usable under the branch and policy. |
| `ErrCreditUnknownRejected` | Credit unknown, policy chose to fail closed. |
| `ErrPolicyIncomplete` | A required policy input was unset or unrecognised. |
| `ErrInvalidCreditStatus` | The status violated state-requires-signal. |

All usable with `errors.Is`, and deliberately distinguishable — a test asserts
they do not collapse into one another.

## Feeding It From This Module's Own Data

```go
candidates := scoring.CandidatesFromModelData(models,
    func(md scoring.ModelData) float64 { return strengthOf(md) },
    func(md scoring.ModelData) bool    { return catalogueWasRead(md) },
    "USD")
```

`priceObserved` is an explicit argument, not an inference from the value, for
exactly the reason described under **Affordability** above. When the caller
cannot say, the candidate is honestly built with an unknown price rather than
silently classified free.

`scoring.CandidateFromComprehensiveScore` takes the price separately because
`ComprehensiveScore` carries no cost fields — its `CostScore` is a normalised
0–10 rating, **not a price**, and must never be mistaken for one.

## Decoupling Contract (CONST-051(B) / CONST-069)

The package is consumer-agnostic by construction:

- It never decides what "strongest" means — the caller supplies `Strength`.
- It never decides what to do on unknown credit — the caller supplies
  `OnUnknownCredit`.
- It never discovers accounts, keys, endpoints, or config paths. Every input
  arrives as a function argument or a config struct field.
- `CreditReporter`'s `account` argument is opaque: never parsed, never turned
  into a filesystem path, never assumed to follow any naming scheme.
- It contains no consumer project name, path, command prefix, or convention.

## Testing

```bash
cd LLMsVerifier/llm-verifier

# The selection package (unit + real-HTTP integration + anti-vacuity guards)
GOMAXPROCS=2 nice -n 19 go test -count=1 -race -v ./selection/...

# The adapters, in the packages that own the source types
GOMAXPROCS=2 nice -n 19 go test -count=1 -race -v \
  -run 'TestCreditStatusFromProbeOutcome' ./providers/
GOMAXPROCS=2 nice -n 19 go test -count=1 -race -v -run 'TestCandidate' ./scoring/
```

**Integration layer, no fakes** (CONST-050(A)) — `BalanceEndpointReporter` is
exercised against a real `httptest` server over a real loopback socket. The
actual transport, request-building, status-handling and JSON-decoding paths
run; nothing is substituted for the code under test.

**Anti-vacuity guards with proven teeth** (CONST-035) — a suite a degenerate
selector could still pass is not evidence. `antivacuous_test.go` runs a
discrimination matrix that a correct selector must answer three different ways,
and rejects three failure shapes:

| Degenerate selector | Caught by |
|---------------------|-----------|
| Always returns the same model | distinctness check (`vacuous-constant`) |
| Never returns a model | per-scenario nil check (`vacuous-empty`) |
| Discriminates on credit but *inverted* | per-scenario correctness check |

Crucially the guard is itself proven falsifiable: three tests run it against
those deliberately broken selectors and **require it to fail them**. A guard
that never fails is the same bluff one level up.

## Related Documentation

- [Model Verification Guide](MODEL_VERIFICATION_GUIDE.md)
- [Capability Detection](CAPABILITY_DETECTION.md)
- [Scoring System Documentation](../SCORING_SYSTEM_DOCUMENTATION.md)
- [Architecture Overview](ARCHITECTURE_OVERVIEW.md)
