package scoring

import (
	"testing"

	"github.com/stretchr/testify/require"
	"digital.vasic.llmsverifier/database"
)

// TestListModelsByScore_RespectsMaxScore is a reproduce-first RED test for the
// maxScore-ignored bug: ListModelsByScore accepts a maxScore bound but the
// underlying db.ListModels has no max_score filter, so models scoring ABOVE
// maxScore are wrongly returned (the range degenerates to "score >= minScore").
func TestListModelsByScore_RespectsMaxScore(t *testing.T) {
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	provider := &database.Provider{
		Name:     "Test Provider",
		Endpoint: "https://api.test.com",
		IsActive: true,
	}
	require.NoError(t, db.CreateProvider(provider))

	mk := func(id string, score float64) {
		m := &database.Model{
			ProviderID:   provider.ID,
			ModelID:      id,
			Name:         id,
			OverallScore: score,
		}
		require.NoError(t, db.CreateModel(m))
	}
	mk("low", 3.0)
	mk("mid", 6.0)
	mk("high", 9.0)

	di := NewDatabaseIntegration(db)

	// Request the range [4.0, 7.0]: only "mid" (6.0) qualifies.
	results, err := di.ListModelsByScore(4.0, 7.0, 0)
	require.NoError(t, err)

	for _, m := range results {
		if m.OverallScore < 4.0 || m.OverallScore > 7.0 {
			t.Fatalf("model %q with score %.1f is outside requested range [4.0, 7.0]", m.ModelID, m.OverallScore)
		}
	}
	require.Len(t, results, 1, "expected exactly one model within [4.0, 7.0]")
	require.Equal(t, "mid", results[0].ModelID)
}
