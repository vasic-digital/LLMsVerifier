package verification

import (
	"context"
	"fmt"

	"digital.vasic.llmsverifier/database"
)

// Request represents a verification request
type Request struct {
	ModelID string `json:"model_id"`
	Prompt  string `json:"prompt"`
}

// Verifier handles model verification
type Verifier struct {
	db *database.Database
}

// NewVerifier creates a new verifier
func NewVerifier(db *database.Database) *Verifier {
	return &Verifier{
		db: db,
	}
}

// Verify performs model verification
func (v *Verifier) Verify(ctx context.Context, req *Request) (*database.VerificationResult, error) {
	if req == nil {
		return nil, fmt.Errorf("verification request cannot be nil")
	}

	if req.ModelID == "" {
		return nil, fmt.Errorf("model ID is required")
	}

	if req.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	// Real verification requires: (1) resolve req.ModelID against the
	// models table to get the int64 PK; (2) dispatch the appropriate
	// per-feature test via the llmverifier.Verifier against the
	// configured provider endpoint; (3) compose a VerificationResult
	// from the actual outcomes.
	//
	// Previously this function returned a hardcoded
	// VerificationResult with EVERY capability flag set to true and
	// EVERY score set to 8.5, regardless of which model was queried.
	// That is a CONST-036/037 violation at the verification
	// entrypoint — the single source of truth was lying to its
	// consumers about every model's capabilities. §11.4 PASS-bluff
	// at the highest blast-radius layer.
	//
	// Fix: return ErrVerificationNotWired so callers see the gap
	// loudly. The canonical implementation requires wiring the
	// existing llmverifier.Verifier (already present at
	// ./llmverifier/verifier.go with TestCapabilities, TestModel,
	// etc.) into this VerificationService — work tracked as
	// round-17 deferral.
	return nil, ErrVerificationNotWired
}

// ErrVerificationNotWired is returned by VerifyModel until the
// llmverifier.Verifier is plumbed into VerificationService. The
// previous fabricated-all-capabilities-true return was a §11.4 /
// CONST-036/037 PASS-bluff and is now removed.
var ErrVerificationNotWired = fmt.Errorf("verification.VerifyModel: real verifier dispatch is not wired (model resolution + per-feature test + score composition required); the previous hardcoded-all-capabilities-true return was a §11.4 / CONST-036/037 PASS-bluff and is now removed — wire llmverifier.Verifier into VerificationService to restore")

// Result is an alias for VerificationResult for backward compatibility
type Result = database.VerificationResult

// ModelVerifier is an alias for Verifier for backward compatibility
type ModelVerifier = Verifier

// NewModelVerifier creates a new model verifier (alias for NewVerifier)
func NewModelVerifier(db *database.Database) *ModelVerifier {
	return NewVerifier(db)
}

// Helper functions
func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}
