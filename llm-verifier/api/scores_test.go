package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"digital.vasic.llmsverifier/config"
	"digital.vasic.llmsverifier/database"
)

// newScoresTestServer builds a Server backed by a real, empty SQLite database
// in a temp dir. Real DB, not a fake: the endpoint's whole job is to report
// what the database actually contains, so a stubbed store would test nothing.
func newScoresTestServer(t *testing.T) (*Server, *database.Database) {
	t.Helper()

	db, err := database.New(filepath.Join(t.TempDir(), "scores_test.db"))
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return NewServer(&config.Config{}, db), db
}

func getScores(t *testing.T, s *Server) (int, scoresResponse) {
	t.Helper()

	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/scores", nil))

	var body scoresResponse
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode /api/scores response %q: %v", rec.Body.String(), err)
		}
	}
	return rec.Code, body
}

// seedProvider creates a provider and returns its id.
func seedProvider(t *testing.T, db *database.Database, name string) int64 {
	t.Helper()

	p := &database.Provider{
		Name:     name,
		Endpoint: "https://example.invalid/v1",
		IsActive: true,
	}
	if err := db.CreateProvider(p); err != nil {
		t.Fatalf("create provider %q: %v", name, err)
	}
	return p.ID
}

func seedModel(t *testing.T, db *database.Database, providerID int64, modelID, status string, score float64) {
	t.Helper()

	m := &database.Model{
		ProviderID:         providerID,
		ModelID:            modelID,
		Name:               modelID,
		VerificationStatus: status,
		OverallScore:       score,
	}
	if err := db.CreateModel(m); err != nil {
		t.Fatalf("create model %q: %v", modelID, err)
	}
}

// TestScoresHandler_EmptyDatabaseReportsNoScores pins the honest-empty
// contract. This is the case that matters most in practice: a fresh deployment
// that has never run verification.
//
// It MUST return 200 with an empty object, never fabricated numbers and never
// an error. Consumers distinguish "measured nothing" (keep your own defaults)
// from "measured these" (authoritative) purely by whether this map is empty --
// HelixLLM's gateway, for instance, replaces its entire static score table
// whenever this returns a non-empty map. Inventing placeholder scores here
// would silently destroy working defaults downstream.
func TestScoresHandler_EmptyDatabaseReportsNoScores(t *testing.T) {
	s, _ := newScoresTestServer(t)

	code, body := getScores(t, s)

	if code != http.StatusOK {
		t.Fatalf("status = %d, want %d", code, http.StatusOK)
	}
	if body.Scores == nil {
		t.Fatal("scores map is nil; want an empty (non-null) object so consumers decode a usable map")
	}
	if len(body.Scores) != 0 {
		t.Fatalf("scores = %v, want empty: an unverified deployment must not publish scores", body.Scores)
	}
}

// TestScoresHandler_PublishesVerifiedModelsOnScaleOf100 proves the aggregate
// and the unit conversion, both on real rows.
//
// overall_score is stored 0-1 (the verify endpoint persists the 0-1
// VerificationScore verbatim); the published contract is 0-100. A provider
// with models scoring 0.7 and 0.9 must therefore publish 80 — not 8 (the
// under-report a 0-10 reading of the stored scale produces) and not 0.8.
//
// The values here are deliberately in the range the live verify path actually
// writes: a real run against a local model stored 0.96 and must publish 96.
func TestScoresHandler_PublishesVerifiedModelsOnScaleOf100(t *testing.T) {
	s, db := newScoresTestServer(t)

	id := seedProvider(t, db, "openrouter")
	seedModel(t, db, id, "model-a", "verified", 0.7)
	seedModel(t, db, id, "model-b", "verified", 0.9)

	code, body := getScores(t, s)

	if code != http.StatusOK {
		t.Fatalf("status = %d, want %d", code, http.StatusOK)
	}

	got, ok := body.Scores["openrouter"]
	if !ok {
		t.Fatalf("provider openrouter missing from scores %v", body.Scores)
	}
	if got.Total != 80.0 {
		t.Errorf("total = %v, want 80 (mean of 0.7 and 0.9 stored on 0-1, published on 0-100)", got.Total)
	}
	if got.ModelCount != 2 {
		t.Errorf("model_count = %d, want 2", got.ModelCount)
	}
}

// TestScoresHandler_MatchesLiveVerifyPathScale pins the exact value observed
// from a real verification run, so a future change to the stored scale cannot
// silently shift what consumers receive.
//
// A live verify of a local model returned VerificationScore 0.96, which
// api/handlers.go persists verbatim to models.overall_score. The published
// score for that provider must be 96 — the value the consuming gateway ranks
// against its own 0-100 table.
func TestScoresHandler_MatchesLiveVerifyPathScale(t *testing.T) {
	s, db := newScoresTestServer(t)

	id := seedProvider(t, db, "llamacpp")
	seedModel(t, db, id, "/models/Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf", "verified", 0.96)

	code, body := getScores(t, s)

	if code != http.StatusOK {
		t.Fatalf("status = %d, want %d", code, http.StatusOK)
	}
	got, ok := body.Scores["llamacpp"]
	if !ok {
		t.Fatalf("provider llamacpp missing from scores %v", body.Scores)
	}
	if got.Total != 96.0 {
		t.Errorf("total = %v, want 96: a stored VerificationScore of 0.96 must publish as 96 "+
			"on the 0-100 consumer contract", got.Total)
	}
}

// TestScoresHandler_ExcludesUnverifiedAndOmitsUnscoredProviders is the
// anti-bluff case: it proves the endpoint reports only genuinely-verified
// measurements, and that a provider with nothing verified is ABSENT rather than
// present with a zero.
//
// The distinction is not cosmetic. "Provider scores 0" tells a consumer to rank
// it last and route away from it; "provider is absent" tells it there is no
// measurement. Reporting an unverified provider as 0 would actively mislead
// routing decisions.
func TestScoresHandler_ExcludesUnverifiedAndOmitsUnscoredProviders(t *testing.T) {
	s, db := newScoresTestServer(t)

	verified := seedProvider(t, db, "scored-provider")
	seedModel(t, db, verified, "good", "verified", 0.8)
	// Same provider, but these must not drag the average: not verified.
	seedModel(t, db, verified, "pending-model", "pending", 1.0)
	seedModel(t, db, verified, "failed-model", "failed", 1.0)

	// A provider whose only model was never verified must not appear at all.
	unscored := seedProvider(t, db, "unmeasured-provider")
	seedModel(t, db, unscored, "never-run", "pending", 0)

	code, body := getScores(t, s)

	if code != http.StatusOK {
		t.Fatalf("status = %d, want %d", code, http.StatusOK)
	}

	got, ok := body.Scores["scored-provider"]
	if !ok {
		t.Fatalf("scored-provider missing from scores %v", body.Scores)
	}
	if got.Total != 80.0 {
		t.Errorf("total = %v, want 80: only the 'verified' model (8.0) may count, "+
			"pending/failed rows must be excluded from the average", got.Total)
	}
	if got.ModelCount != 1 {
		t.Errorf("model_count = %d, want 1 (only the verified model)", got.ModelCount)
	}

	if entry, present := body.Scores["unmeasured-provider"]; present {
		t.Errorf("unmeasured-provider present with %+v; a provider with no verified model must be "+
			"OMITTED, not published as a zero score (zero reads as 'bad', absence reads as 'unknown')", entry)
	}
}

// TestScoresHandler_RejectsNonGET keeps the method contract explicit.
func TestScoresHandler_RejectsNonGET(t *testing.T) {
	s, _ := newScoresTestServer(t)

	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/scores", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /api/scores status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// TestRoutes_ServerAndTestRouterAgree guards the drift that made this endpoint
// easy to get wrong: Router() (tests) and Start() (production) used to declare
// separate route tables, so a route could exist in one and not the other. Both
// now build from routes(); this asserts the shared table actually serves the
// endpoints consumers depend on.
func TestRoutes_ServerAndTestRouterAgree(t *testing.T) {
	s, _ := newScoresTestServer(t)
	mux := s.routes()

	for _, path := range []string{"/api/health", "/api/models", "/api/providers", "/api/scores"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if h, pattern := mux.Handler(req); h == nil || pattern == "" {
			t.Errorf("no handler registered for %s (pattern=%q)", path, pattern)
		}
	}
}
