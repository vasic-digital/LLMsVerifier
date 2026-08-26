package api

import (
	"encoding/json"
	"net/http"
)

// scoreEntry is one provider's published score.
type scoreEntry struct {
	Total float64 `json:"total"`
	// ModelCount lets a consumer distinguish a provider averaged over one
	// verified model from one averaged over many. Additive: a consumer that
	// only reads "total" is unaffected.
	ModelCount int `json:"model_count"`
}

// scoresResponse is the published shape of GET /api/scores:
//
//	{"scores": {"openrouter": {"total": 87.5, "model_count": 4}}}
//
// The nesting (rather than a flat provider->float map) is intentional: it
// leaves room to publish per-dimension scores later without breaking consumers
// that already read .total.
type scoresResponse struct {
	Scores map[string]scoreEntry `json:"scores"`
}

// ScoresHandler serves GET /api/scores — the aggregate verification score per
// provider, drawn from real verification results in the database.
//
// This exists because LLMsVerifier is the single source of truth for scoring
// data (CONST-036), and consumers were already asking for it: HelixLLM's
// gateway polls {verifier}/api/scores every few minutes and had been logging
// "verifier unreachable, using static scores" indefinitely, because no such
// endpoint existed on any port. The route is the missing half of a contract
// that one side had already implemented.
//
// Honest-empty contract: when no provider has a verified, scored model, this
// returns {"scores":{}} with 200 — a successful "we have measured nothing yet",
// not an error and not fabricated numbers. That distinction matters to the
// consumer: HelixLLM treats an empty map as "keep my own defaults" and a
// populated map as authoritative, replacing its static table wholesale. Serving
// a partial or invented map here would silently discard working defaults and
// degrade routing, so the endpoint publishes only what was genuinely measured.
func (s *Server) ScoresHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if s.database == nil {
		http.Error(w, "database not configured", http.StatusServiceUnavailable)
		return
	}

	rows, err := s.database.ProviderScores()
	if err != nil {
		http.Error(w, "failed to read provider scores", http.StatusInternalServerError)
		return
	}

	// Always a non-nil map so the JSON is {"scores":{}} rather than
	// {"scores":null} — a consumer decoding into a map gets an empty map either
	// way, but null reads as a malformed payload to anything stricter.
	resp := scoresResponse{Scores: make(map[string]scoreEntry, len(rows))}
	for _, row := range rows {
		resp.Scores[row.Provider] = scoreEntry{
			Total:      row.Total,
			ModelCount: row.ModelCount,
		}
	}

	_ = json.NewEncoder(w).Encode(resp)
}
