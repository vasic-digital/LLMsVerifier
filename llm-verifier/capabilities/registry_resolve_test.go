package capabilities

import (
	"testing"
	"time"

	"digital.vasic.llmsverifier/database"
)

// newResolverTestDB opens a real in-memory SQLite DB (schema + migrations run
// by database.New) and returns it. No mocks — this is the real persistence
// layer the resolver reads (CONST-050(A): non-unit paths use the real system).
func newResolverTestDB(t *testing.T) *database.Database {
	t.Helper()
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("database.New(:memory:) failed: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// seedModel creates a real provider + model row and returns the model PK.
func seedModel(t *testing.T, db *database.Database) int64 {
	t.Helper()
	provider := &database.Provider{
		Name:     "C3 Resolver Test Provider",
		Endpoint: "https://api.c3-resolver-test.example/v1",
		IsActive: true,
	}
	if err := db.CreateProvider(provider); err != nil {
		t.Fatalf("CreateProvider failed: %v", err)
	}
	model := &database.Model{
		ProviderID:  provider.ID,
		ModelID:     "c3-resolver-model",
		Name:        "c3-resolver-model",
		Description: "model under C3 fail-closed resolver test",
	}
	if err := db.CreateModel(model); err != nil {
		t.Fatalf("CreateModel failed: %v", err)
	}
	return model.ID
}

// persistFreshVerification writes a fresh (now) completed VerificationResult
// for modelID with the given RAG verdict, and returns nothing — the resolver
// reads it back via db.GetLatestVerificationResults.
func persistFreshVerification(t *testing.T, db *database.Database, modelID int64, ragSupported bool) {
	t.Helper()
	now := time.Now()
	completedAt := now.Add(2 * time.Second)
	vr := &database.VerificationResult{
		ModelID:          modelID,
		VerificationType: "comprehensive",
		StartedAt:        now,
		CompletedAt:      &completedAt,
		Status:           "completed",
		SupportsRAG:      ragSupported,
	}
	if err := db.CreateVerificationResult(vr); err != nil {
		t.Fatalf("CreateVerificationResult failed: %v", err)
	}
}

// TestC3Resolver_RuntimeSignature is the §11.4.108 machine-checkable runtime
// signature for C3: with a FRESH persisted VerificationResult(RAG=true) the
// resolver reports RAG verified with the probed value; with NO probe it
// FAIL-CLOSES to unverified — NEVER the seed's self-certified literal.
func TestC3Resolver_RuntimeSignature(t *testing.T) {
	db := newResolverTestDB(t)
	modelID := seedModel(t, db)

	// (1) FAIL-CLOSED: no probe persisted yet ⇒ unverified.
	value, verified, err := ResolveModelCapability(db, modelID, "rag")
	if err != nil {
		t.Fatalf("ResolveModelCapability(rag) unexpected error (no-probe case): %v", err)
	}
	if verified {
		t.Fatalf("no fresh probe present, resolver MUST fail-close to unverified; got verified=true value=%v", value)
	}
	if value {
		t.Fatalf("fail-closed verdict MUST report value=false absent a probe; got value=true")
	}

	// (2) VERIFIED: persist a fresh VerificationResult(RAG=true) ⇒ RAG verified.
	persistFreshVerification(t, db, modelID, true)
	value, verified, err = ResolveModelCapability(db, modelID, "rag")
	if err != nil {
		t.Fatalf("ResolveModelCapability(rag) unexpected error (fresh-probe case): %v", err)
	}
	if !verified {
		t.Fatalf("a fresh completed VerificationResult exists, resolver MUST report verified=true; got verified=false")
	}
	if !value {
		t.Fatalf("fresh probe has SupportsRAG=true, resolver MUST report value=true; got value=false")
	}
}

// TestC3Resolver_StaleProbeFailsClosed proves a probe OLDER than the CONST-037
// freshness window is NOT authoritative — the resolver fail-closes on a stale
// verification exactly as it does on no verification.
func TestC3Resolver_StaleProbeFailsClosed(t *testing.T) {
	db := newResolverTestDB(t)
	modelID := seedModel(t, db)

	old := time.Now().Add(-verificationFreshnessWindow - time.Hour)
	completedAt := old.Add(2 * time.Second)
	vr := &database.VerificationResult{
		ModelID:          modelID,
		VerificationType: "comprehensive",
		StartedAt:        old,
		CompletedAt:      &completedAt,
		Status:           "completed",
		SupportsRAG:      true, // even though the stale probe says true...
	}
	if err := db.CreateVerificationResult(vr); err != nil {
		t.Fatalf("CreateVerificationResult failed: %v", err)
	}

	value, verified, err := ResolveModelCapability(db, modelID, "rag")
	if err != nil {
		t.Fatalf("ResolveModelCapability(rag) unexpected error: %v", err)
	}
	if verified || value {
		t.Fatalf("a stale probe MUST fail-close to unverified; got verified=%v value=%v", verified, value)
	}
}

// TestC3Resolver_UnknownCapabilityErrors proves an unmapped capability name is
// an explicit ErrUnknownCapability, never a silent fail-closed false.
func TestC3Resolver_UnknownCapabilityErrors(t *testing.T) {
	db := newResolverTestDB(t)
	modelID := seedModel(t, db)
	if _, _, err := ResolveModelCapability(db, modelID, "totally_made_up_capability"); err != ErrUnknownCapability {
		t.Fatalf("expected ErrUnknownCapability for an unmapped name, got %v", err)
	}
}

// TestC3Resolver_NilDBFailsClosed proves a nil DB handle fail-closes cleanly
// (no probe source) rather than panicking.
func TestC3Resolver_NilDBFailsClosed(t *testing.T) {
	value, verified, err := ResolveModelCapability(nil, 1, "rag")
	if err != nil {
		t.Fatalf("nil-db case MUST NOT error; got %v", err)
	}
	if verified || value {
		t.Fatalf("nil db (no probe source) MUST fail-close to unverified; got verified=%v value=%v", verified, value)
	}
}
