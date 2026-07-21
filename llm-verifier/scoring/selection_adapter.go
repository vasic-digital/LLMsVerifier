package scoring

// selection_adapter.go turns this package's model metadata and scores into
// selection.Candidate values, so credit-aware selection runs on the scoring
// engine's real output rather than on a parallel data set.
//
// # Why priceObserved is an explicit argument
//
// ModelData.InputTokenCost and ModelData.OutputTokenCost are bare float64.
// A genuinely free model and a model whose price was never fetched are both
// stored as 0. Selecting on that ambiguity would classify every price-less
// record as free and route paid traffic to it. These constructors therefore
// refuse to infer observation from the value: the caller states whether the
// numbers came from a real catalogue read, and when it cannot, the candidate
// is honestly built with an unknown price.

import "digital.vasic.llmsverifier/selection"

// CandidateFromModelData builds a selection candidate from models.dev-derived
// model data.
//
// strength is the caller's ranking metric (commonly a ComprehensiveScore's
// OverallScore); higher is stronger. priceObserved must be true only when the
// cost fields were actually populated from a catalogue or API response.
// currency labels the amounts and is informational.
func CandidateFromModelData(md ModelData, strength float64, priceObserved bool, currency string) selection.Candidate {
	price := selection.UnknownPrice()
	if priceObserved {
		price = selection.KnownPrice(md.InputTokenCost, md.OutputTokenCost, currency)
	}
	return selection.Candidate{
		ID:       md.ID,
		Provider: md.Provider,
		Strength: strength,
		Price:    price,
	}
}

// CandidateFromComprehensiveScore builds a candidate from a computed score,
// using OverallScore as the strength metric. The price is supplied separately
// because ComprehensiveScore carries no cost fields — its CostScore is a
// normalised 0..10 rating, not a price, and must never be mistaken for one.
func CandidateFromComprehensiveScore(score *ComprehensiveScore, provider string, price selection.TokenPrice) selection.Candidate {
	if score == nil {
		return selection.Candidate{}
	}
	return selection.Candidate{
		ID:       score.ModelID,
		Provider: provider,
		Strength: score.OverallScore,
		Price:    price,
	}
}

// CandidatesFromModelData maps a slice of model data to candidates.
//
// strengthOf supplies the ranking metric per model and priceObservedOf reports
// whether that model's cost fields were really observed. Both are required;
// when either is nil the corresponding input is treated as unavailable
// (strength 0, price unknown) rather than silently defaulted to a favourable
// value.
func CandidatesFromModelData(
	models []ModelData,
	strengthOf func(ModelData) float64,
	priceObservedOf func(ModelData) bool,
	currency string,
) []selection.Candidate {
	candidates := make([]selection.Candidate, 0, len(models))
	for _, md := range models {
		var strength float64
		if strengthOf != nil {
			strength = strengthOf(md)
		}
		observed := false
		if priceObservedOf != nil {
			observed = priceObservedOf(md)
		}
		candidates = append(candidates, CandidateFromModelData(md, strength, observed, currency))
	}
	return candidates
}
