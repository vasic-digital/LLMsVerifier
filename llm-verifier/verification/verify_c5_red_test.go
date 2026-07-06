package verification

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"digital.vasic.llmsverifier/config"
	"digital.vasic.llmsverifier/database"
	"digital.vasic.llmsverifier/llmverifier"
)

// c5ProbePositiveContent is the assistant message a loopback endpoint returns so
// the CONST-040 C4 RAG probe reaches a positive verdict (it cites the injected
// sentinel "zorblax-7742"). It gives the C5 GREEN guard a real, probe-sourced
// runtime signature: the persisted VerificationResult.SupportsRAG MUST be true
// ONLY because a real wire call detected the capability.
const c5ProbePositiveContent = "Using the code_formatter skill and the weather_lookup plugin, " +
	"the grounded answer from the provided document is zorblax-7742."

// TestC5VerifyComposesAndPersists is the §11.4.115 RED-baseline-on-the-broken-
// artifact polarity guard for change C5 (10b_code_exact_change_spec.md §3 C5):
// wire VerificationService.Verify to dispatch the C4 probes, compose a
// database.VerificationResult from the REAL outcomes, and PERSIST it (so C3's
// fail-closed resolver can later read a fresh per-model VerificationResult).
//
// Defect (pre-C5, captured this session): Verify returned ErrVerificationNotWired
// unconditionally — no model resolution, no probe dispatch, no persisted result.
//
// Polarity switch (§11.4.115):
//
//	RED_MODE=1           — reproduce the defect: assert Verify does NOT persist a
//	    result (err != nil AND result == nil). On the FIXED artifact the probe
//	    dispatch + persist path runs, so this branch FAILs (defect no longer
//	    reproduces) — the polarity proof.
//	RED_MODE=0 (default) — standing GREEN regression guard: a wired verifier
//	    resolves the model, dispatches the real probes against a loopback
//	    endpoint, composes and PERSISTS a VerificationResult whose ModelID is the
//	    model PK and whose CONST-040 SupportsRAG is true because the real probe
//	    detected it, and the row is retrievable from the DB. Runs on every build
//	    per §11.4.135.
func TestC5VerifyComposesAndPersists(t *testing.T) {
	redMode := os.Getenv("RED_MODE")
	if redMode == "" {
		redMode = "0"
	}

	// Real in-memory DB with a real provider + model row (foreign keys ON, so the
	// model MUST exist for persistence to succeed).
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	prov := &database.Provider{Name: "c5-prov", Endpoint: "loopback"}
	if err := db.CreateProvider(prov); err != nil {
		t.Fatalf("create provider: %v", err)
	}
	model := &database.Model{ProviderID: prov.ID, ModelID: "gpt-4-c5", Name: "GPT-4 C5"}
	if err := db.CreateModel(model); err != nil {
		t.Fatalf("create model: %v", err)
	}

	// Real loopback endpoint returning a capability-positive answer for every
	// probe's chat/completions call — a REAL HTTP round-trip (no simulation).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"` +
			c5ProbePositiveContent + `"}}]}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		Timeout: 30 * time.Second,
		Global: config.GlobalConfig{
			BaseURL:      server.URL,
			APIKey:       "loopback-key",
			DefaultModel: "gpt-4-c5",
			Timeout:      30 * time.Second,
		},
	}
	prober := llmverifier.New(cfg)
	v := NewVerifierWithProber(db, prober)

	res, verr := v.Verify(context.Background(), &Request{ModelID: "gpt-4-c5", Prompt: "Do you see my code?"})

	switch redMode {
	case "1": // reproduce-and-assert-defect-present (FAILs on the FIXED artifact)
		if verr != nil && res == nil {
			t.Logf("RED_MODE=1 PASS: defect reproduced — Verify persisted no VerificationResult (err=%v).", verr)
			return
		}
		t.Fatalf("RED_MODE=1: expected the pre-C5 not-wired defect (err!=nil, result==nil), "+
			"but Verify composed+persisted a result (err=%v, result=%v) — C5 is wired, polarity intact", verr, res)
	case "0": // standing GREEN guard
		if verr != nil {
			t.Fatalf("RED_MODE=0: Verify must compose+persist a VerificationResult, got err=%v", verr)
		}
		if res == nil {
			t.Fatalf("RED_MODE=0: Verify returned nil result with no error")
		}
		if res.ModelID != model.ID {
			t.Fatalf("RED_MODE=0: persisted ModelID = %d, want model PK %d", res.ModelID, model.ID)
		}
		if res.ID == 0 {
			t.Fatalf("RED_MODE=0: persisted result has no DB id (was not written)")
		}
		// Runtime signature (§11.4.108): SupportsRAG is true ONLY because the C4
		// RAG probe made a real wire call to the loopback endpoint and detected
		// the injected sentinel — a probe-sourced, non-fabricated capability.
		if !res.SupportsRAG {
			t.Fatalf("RED_MODE=0: composed result SupportsRAG=false; the C4 RAG probe outcome " +
				"did not flow into the persisted VerificationResult")
		}
		if res.RawResponse == nil || *res.RawResponse == "" {
			t.Fatalf("RED_MODE=0: persisted result has empty RawResponse; per-capability evidence not captured")
		}
		// Prove it is genuinely PERSISTED and readable back (what C3 will read).
		got, gerr := db.GetVerificationResult(res.ID)
		if gerr != nil {
			t.Fatalf("RED_MODE=0: persisted result not readable back: %v", gerr)
		}
		if got.ModelID != model.ID || !got.SupportsRAG {
			t.Fatalf("RED_MODE=0: read-back mismatch: ModelID=%d SupportsRAG=%v", got.ModelID, got.SupportsRAG)
		}
		t.Logf("RED_MODE=0 PASS: Verify composed + persisted VerificationResult id=%d "+
			"(ModelID=%d, SupportsRAG=%v via real probe) — readable back for the C3 resolver.",
			res.ID, res.ModelID, res.SupportsRAG)
	default:
		t.Fatalf("unknown RED_MODE=%q (expected 0 or 1)", redMode)
	}
}
