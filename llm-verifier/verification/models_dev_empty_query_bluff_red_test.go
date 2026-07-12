package verification

import "testing"

// TestCalculateMatchScore_EmptyQueryMustNotMatch is the §11.4.115
// RED-on-broken-artifact regression guard for an empty-input acceptance bug
// in EnhancedModelsDevClient.calculateMatchScore (verification/models_dev_enhanced.go).
//
// ROOT CAUSE (independent-audit finding, 2026-07-10): calculateMatchScore
// scores a candidate model against a search query using strings.Contains
// substring checks:
//
//	if strings.Contains(modelIDLower, query)   { score += 0.6 }
//	if strings.Contains(modelNameLower, query) { score += 0.5 }
//	if strings.Contains(familyLower, query)    { score += 0.4 }
//	if strings.Contains(providerIDLower, query){ score += 0.2 }
//
// strings.Contains(s, "") is ALWAYS true for any s (the empty string is a
// substring of every string) -- so an EMPTY query scores >= 1.7 against
// EVERY model in the catalogue, comfortably clearing FindModel's `score >
// 0.3` match threshold (models_dev_enhanced.go:252). A malformed/empty
// model ID (a plausible real-world payload -- see
// providers/model_provider_service.go:437 `mps.modelsDevClient.FindModel(ctx,
// data.ID)`, where data.ID is decoded straight from a provider's external
// `/v1/models` JSON response and is NOT validated non-empty first) therefore
// does not fail to match -- it matches EVERY model in the entire models.dev
// catalogue. FindModel then takes matches[0] after a same-score-ties-not-
// reordered sort over a Go map-iteration-order (non-deterministic across
// runs) traversal, so the caller enriches the malformed-ID model with an
// ARBITRARY unrelated model's name/cost/context-limit/capability flags
// (providers/model_provider_service.go:496-507 enhanceFromModelsDevMatch) --
// a "verifier"/matcher that is supposed to REJECT a non-match instead
// silently fabricates one from garbage input.
//
// CONTRACT: an empty (or whitespace-only) search query must score 0 against
// every candidate -- it must never be treated as a substring match.
func TestCalculateMatchScore_EmptyQueryMustNotMatch(t *testing.T) {
	c := NewEnhancedModelsDevClient(nil)

	provider := ProviderData{ID: "some-provider", Name: "Some Provider"}
	model := ModelDetails{
		ID:     "some-model-id",
		Name:   "Some Model",
		Family: "some-family",
	}

	score := c.calculateMatchScore("", "some-provider", provider, "some-model-id", model)

	const matchThreshold = 0.3 // FindModel's Strategy-2 acceptance threshold
	if score > matchThreshold {
		t.Fatalf("EMPTY-QUERY BLUFF: calculateMatchScore(\"\", ...) returned %.2f, which is ABOVE "+
			"FindModel's %.2f match threshold -- an empty/malformed model ID would match EVERY model "+
			"in the catalogue instead of matching none", score, matchThreshold)
	}

	// Same check against a SECOND, entirely different model -- an empty
	// query must not match this one either (proving the defect is "matches
	// everything", not merely "matches this one model by coincidence").
	otherProvider := ProviderData{ID: "totally-different-provider", Name: "Totally Different"}
	otherModel := ModelDetails{ID: "totally-different-model", Name: "Totally Different Model", Family: "other-family"}
	otherScore := c.calculateMatchScore("", "totally-different-provider", otherProvider, "totally-different-model", otherModel)
	if otherScore > matchThreshold {
		t.Fatalf("EMPTY-QUERY BLUFF: calculateMatchScore(\"\", ...) matched an UNRELATED model (score %.2f), "+
			"confirming an empty query fabricates matches against the entire catalogue", otherScore)
	}
}
