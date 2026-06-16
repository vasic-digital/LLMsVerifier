package seed

import (
	"testing"

	"digital.vasic.llmsverifier/config"
	"digital.vasic.llmsverifier/database"
)

// newTestDB opens a real in-memory SQLite database with the production schema.
// Per CONST-050 this is the REAL database layer (real sqlite3 driver, real
// schema, real SQL) — not a fake — exercised in a _test.go unit/integration test.
func newTestDB(t *testing.T) *database.Database {
	t.Helper()
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func cfgWith(llms ...config.LLMConfig) *config.Config {
	return &config.Config{LLMs: llms}
}

// TestFromConfig_SeedsModelsFromConfig is the §11.4.43/§11.4.115 RED-baseline
// test: BEFORE the seed path existed, a freshly-booted verifier left the model
// DB empty even though config.yaml listed `llms:` — so ListModels returned 0 and
// /api/models reported count:0 (the deployed-verifier defect). It asserts the
// fix: FromConfig populates the DB from config so ListModels returns the
// configured models. Remove FromConfig's CreateModel call and this FAILs.
func TestFromConfig_SeedsModelsFromConfig(t *testing.T) {
	db := newTestDB(t)

	cfg := cfgWith(
		config.LLMConfig{Name: "deepseek", Endpoint: "https://api.deepseek.com/v1", Model: "deepseek-chat"},
		config.LLMConfig{Name: "groq", Endpoint: "https://api.groq.com/openai/v1", Model: "llama-3.3-70b-versatile"},
	)

	// Pre-condition (the broken/baseline state): empty DB.
	before, err := db.ListModels(map[string]any{})
	if err != nil {
		t.Fatalf("list models (before): %v", err)
	}
	if len(before) != 0 {
		t.Fatalf("baseline expected 0 models, got %d", len(before))
	}

	res, err := FromConfig(cfg, db)
	if err != nil {
		t.Fatalf("FromConfig: %v", err)
	}
	if res.ModelsCreated != 2 {
		t.Fatalf("expected 2 models created, got %d", res.ModelsCreated)
	}
	if res.ProvidersCreated != 2 {
		t.Fatalf("expected 2 providers created, got %d", res.ProvidersCreated)
	}

	// Post-condition: the real DB now serves the configured models — exactly
	// what /api/models reads via ListModels.
	after, err := db.ListModels(map[string]any{})
	if err != nil {
		t.Fatalf("list models (after): %v", err)
	}
	if len(after) != 2 {
		t.Fatalf("expected 2 models in DB after seed, got %d", len(after))
	}

	got := map[string]string{}
	for _, m := range after {
		got[m.ModelID] = m.Name
	}
	if got["deepseek-chat"] != "deepseek" {
		t.Errorf("missing deepseek-chat model, got map=%v", got)
	}
	if got["llama-3.3-70b-versatile"] != "groq" {
		t.Errorf("missing groq model, got map=%v", got)
	}
}

// TestFromConfig_Idempotent proves running the seed twice does not duplicate
// rows — the boot-seed-every-start safety property.
func TestFromConfig_Idempotent(t *testing.T) {
	db := newTestDB(t)
	cfg := cfgWith(
		config.LLMConfig{Name: "deepseek", Endpoint: "https://api.deepseek.com/v1", Model: "deepseek-chat"},
	)

	if _, err := FromConfig(cfg, db); err != nil {
		t.Fatalf("FromConfig (first): %v", err)
	}
	res2, err := FromConfig(cfg, db)
	if err != nil {
		t.Fatalf("FromConfig (second): %v", err)
	}
	if res2.ModelsCreated != 0 || res2.ProvidersCreated != 0 {
		t.Fatalf("second seed created rows: providers=%d models=%d (must be 0)",
			res2.ProvidersCreated, res2.ModelsCreated)
	}
	if res2.ModelsReused != 1 || res2.ProvidersReused != 1 {
		t.Fatalf("second seed reuse counts wrong: providers=%d models=%d (want 1/1)",
			res2.ProvidersReused, res2.ModelsReused)
	}

	all, err := db.ListModels(map[string]any{})
	if err != nil {
		t.Fatalf("list models: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("idempotency broken: expected 1 model after two seeds, got %d", len(all))
	}
}

// TestFromConfig_SkipsEmptyName proves a malformed entry is skipped with a
// recorded reason rather than producing a bad row (§11.4.6 honesty).
func TestFromConfig_SkipsEmptyName(t *testing.T) {
	db := newTestDB(t)
	cfg := cfgWith(
		config.LLMConfig{Name: "", Endpoint: "https://example.test/v1", Model: "x"},
		config.LLMConfig{Name: "groq", Endpoint: "https://api.groq.com/openai/v1", Model: "llama-3.3-70b-versatile"},
	)

	res, err := FromConfig(cfg, db)
	if err != nil {
		t.Fatalf("FromConfig: %v", err)
	}
	if len(res.Skipped) != 1 {
		t.Fatalf("expected 1 skipped entry, got %d (%v)", len(res.Skipped), res.Skipped)
	}
	if res.ModelsCreated != 1 {
		t.Fatalf("expected 1 model created (groq only), got %d", res.ModelsCreated)
	}
}

// TestFromConfig_ModelIDFallsBackToName proves model_id derives from Name when
// the Model field is empty (so a model is still seeded + discoverable).
func TestFromConfig_ModelIDFallsBackToName(t *testing.T) {
	db := newTestDB(t)
	cfg := cfgWith(config.LLMConfig{Name: "ollama-local", Endpoint: "http://localhost:11434/v1"})

	if _, err := FromConfig(cfg, db); err != nil {
		t.Fatalf("FromConfig: %v", err)
	}
	models, err := db.ListModels(map[string]any{"model_id": "ollama-local"})
	if err != nil {
		t.Fatalf("list models: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected model_id to fall back to name, got %d models", len(models))
	}
}

// TestFromConfig_NilArgs proves defensive guards.
func TestFromConfig_NilArgs(t *testing.T) {
	if _, err := FromConfig(nil, newTestDB(t)); err == nil {
		t.Error("expected error on nil config")
	}
	if _, err := FromConfig(cfgWith(), nil); err == nil {
		t.Error("expected error on nil database")
	}
}
