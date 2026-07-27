package database

// ProviderScore is one provider's aggregated verification score.
type ProviderScore struct {
	// Provider is the provider's canonical name as stored in the providers
	// table (e.g. "openrouter", "chutes", "llamacpp"). Consumers key off this.
	Provider string
	// Total is the provider's aggregate score on a 0-100 scale.
	//
	// Scale note. models.overall_score is stored on a 0-1 scale: the verify
	// endpoint (api/handlers.go, `model.OverallScore = score`) persists the
	// VerificationScore returned by providers.ModelVerificationService
	// verbatim, and that value is 0-1 (see its 0.0 / 0.3 / 0.5 thresholds).
	// Consumers of the scoring API work on 0-100, so the value is normalised
	// here, once, rather than leaving every caller to guess the unit.
	//
	// Do not confuse this with verification/code_verification_integration.go,
	// which multiplies the same VerificationScore by 10 — that builds a
	// separate code-capability record on a 0-10 scale (alongside
	// ResponsivenessScore: 8.0 and friends) and never writes this column.
	// Reading that path as authoritative for models.overall_score produces a
	// silent 10x under-report, which is exactly what a first cut of this file
	// did until a real verification run (score 0.96) showed 9.6 published
	// where 96 was expected.
	Total float64
	// ModelCount is how many verified, scored models the average is drawn from.
	// Exposed so a consumer can weigh a provider scored from one model
	// differently from one scored across thirty.
	ModelCount int
}

// ProviderScores returns the aggregate verification score for every active
// provider that has at least one genuinely verified, scored model.
//
// Providers with no verified models are OMITTED rather than returned with a
// zero score. This is deliberate and load-bearing: a consumer that receives
// "provider X scores 0" will rank it worst and route away from it, which is a
// materially different — and wrong — claim from "this provider has not been
// verified yet". Absence means "no data"; it must not be reported as "bad".
//
// A fresh deployment whose verification has never run therefore returns an
// empty map, which lets a consumer keep its own defaults instead of adopting
// scores that were never measured.
func (d *Database) ProviderScores() ([]ProviderScore, error) {
	const query = `
		SELECT p.name,
		       AVG(m.overall_score) AS avg_score,
		       COUNT(m.id)          AS model_count
		FROM providers p
		JOIN models m ON m.provider_id = p.id
		WHERE p.is_active = 1
		  AND m.deprecated = 0
		  AND m.verification_status = 'verified'
		  AND m.overall_score > 0
		GROUP BY p.name
		ORDER BY avg_score DESC
	`

	rows, err := d.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scores []ProviderScore
	for rows.Next() {
		var s ProviderScore
		var avg float64
		if err := rows.Scan(&s.Provider, &avg, &s.ModelCount); err != nil {
			return nil, err
		}
		s.Total = avg * 100.0 // 0-1 (stored) -> 0-100 (published)
		scores = append(scores, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return scores, nil
}
