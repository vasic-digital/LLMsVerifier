package scoring

import (
	"testing"
	"time"

	"digital.vasic.llmsverifier/selection"
)

// TestCandidateFromModelData_ZeroPriceAmbiguity is the reason this adapter
// takes priceObserved explicitly. Two ModelData values that are byte-identical
// zeros must produce DIFFERENT affordability classes depending on whether the
// catalogue was actually read. If the adapter ever inferred observation from
// the value, both would come out "free" and unpriced models would be routed as
// free.
func TestCandidateFromModelData_ZeroPriceAmbiguity(t *testing.T) {
	md := ModelData{ID: "m1", Provider: "p1", InputTokenCost: 0, OutputTokenCost: 0}

	observed := CandidateFromModelData(md, 8.0, true, "USD")
	if got := observed.Affordability(); got != selection.AffordabilityFree {
		t.Fatalf("observed zero price -> %q, want free", got)
	}

	notObserved := CandidateFromModelData(md, 8.0, false, "USD")
	if got := notObserved.Affordability(); got != selection.AffordabilityUnknown {
		t.Fatalf("unobserved zero price -> %q, want unknown", got)
	}
}

func TestCandidateFromModelData_Fields(t *testing.T) {
	md := ModelData{ID: "m2", Provider: "acme", InputTokenCost: 3, OutputTokenCost: 15}
	c := CandidateFromModelData(md, 9.25, true, "USD")

	if c.ID != "m2" || c.Provider != "acme" || c.Strength != 9.25 {
		t.Fatalf("fields not carried through: %+v", c)
	}
	if c.Affordability() != selection.AffordabilityPaid {
		t.Fatalf("affordability = %q, want paid", c.Affordability())
	}
	total, ok := c.Price.Total()
	if !ok || total != 18 {
		t.Fatalf("price total = %v/%v, want 18/true", total, ok)
	}
}

func TestCandidateFromComprehensiveScore(t *testing.T) {
	score := &ComprehensiveScore{
		ModelID:      "m3",
		OverallScore: 7.5,
		// CostScore is a normalised rating, NOT a price. It must never leak
		// into the candidate's Price.
		Components: ScoreComponents{CostScore: 9.9},
	}
	c := CandidateFromComprehensiveScore(score, "acme", selection.KnownPrice(1, 2, "USD"))

	if c.ID != "m3" || c.Strength != 7.5 || c.Provider != "acme" {
		t.Fatalf("fields not carried through: %+v", c)
	}
	total, _ := c.Price.Total()
	if total != 3 {
		t.Fatalf("price total = %v, want 3 (CostScore must not become a price)", total)
	}

	if got := CandidateFromComprehensiveScore(nil, "acme", selection.UnknownPrice()); got.ID != "" {
		t.Fatalf("nil score produced %+v, want the zero candidate", got)
	}
}

func TestCandidatesFromModelData(t *testing.T) {
	models := []ModelData{
		{ID: "paid", Provider: "p", InputTokenCost: 5, OutputTokenCost: 5},
		{ID: "free", Provider: "p", InputTokenCost: 0, OutputTokenCost: 0},
		{ID: "stale", Provider: "p"},
	}
	strengths := map[string]float64{"paid": 9, "free": 6, "stale": 10}
	observed := map[string]bool{"paid": true, "free": true, "stale": false}

	got := CandidatesFromModelData(models,
		func(md ModelData) float64 { return strengths[md.ID] },
		func(md ModelData) bool { return observed[md.ID] },
		"USD")

	if len(got) != 3 {
		t.Fatalf("got %d candidates, want 3", len(got))
	}
	want := map[string]selection.Affordability{
		"paid":  selection.AffordabilityPaid,
		"free":  selection.AffordabilityFree,
		"stale": selection.AffordabilityUnknown,
	}
	for _, c := range got {
		if c.Affordability() != want[c.ID] {
			t.Fatalf("%s affordability = %q, want %q", c.ID, c.Affordability(), want[c.ID])
		}
	}

	// Nil callbacks must degrade to the unfavourable side (strength 0, price
	// unknown), never to a value that would win a selection.
	degraded := CandidatesFromModelData(models, nil, nil, "USD")
	for _, c := range degraded {
		if c.Strength != 0 {
			t.Fatalf("%s strength = %v with nil strengthOf, want 0", c.ID, c.Strength)
		}
		if c.Affordability() != selection.AffordabilityUnknown {
			t.Fatalf("%s affordability = %q with nil priceObservedOf, want unknown", c.ID, c.Affordability())
		}
	}
}

// TestCandidatesFromModelData_FeedSelector runs the adapter output through the
// real selector, proving the join between this module's model metadata and
// credit-aware selection actually works end to end.
func TestCandidatesFromModelData_FeedSelector(t *testing.T) {
	models := []ModelData{
		{ID: "big-paid", Provider: "p", InputTokenCost: 3, OutputTokenCost: 15},
		{ID: "small-paid", Provider: "p", InputTokenCost: 1, OutputTokenCost: 2},
		{ID: "big-free", Provider: "p", InputTokenCost: 0, OutputTokenCost: 0},
	}
	strengths := map[string]float64{"big-paid": 9.5, "small-paid": 5, "big-free": 7}

	candidates := CandidatesFromModelData(models,
		func(md ModelData) float64 { return strengths[md.ID] },
		func(ModelData) bool { return true },
		"USD")

	policy := selection.Policy{
		OnUnknownCredit: selection.UnknownCreditPreferFree,
		TieBreak:        selection.TieBreakCheapest,
	}
	sel := selection.NewCreditAwareSelector()

	funded, err := sel.Select(t.Context(), candidates,
		selection.CreditFromBalance(10, "USD", fixedObservedAt()), policy)
	if err != nil {
		t.Fatalf("funded selection failed: %v", err)
	}
	if funded.Chosen == nil || funded.Chosen.ID != "big-paid" {
		t.Fatalf("funded chose %+v, want big-paid", funded.Chosen)
	}

	drained, err := sel.Select(t.Context(), candidates,
		selection.CreditFromBalance(0, "USD", fixedObservedAt()), policy)
	if err != nil {
		t.Fatalf("drained selection failed: %v", err)
	}
	if drained.Chosen == nil || drained.Chosen.ID != "big-free" {
		t.Fatalf("drained chose %+v, want big-free", drained.Chosen)
	}
}

// fixedObservedAt pins the observation timestamp so the test does not depend
// on the wall clock.
func fixedObservedAt() time.Time { return time.Unix(1700000000, 0).UTC() }
