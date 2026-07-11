package verification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"digital.vasic.llmsverifier/database"
	"digital.vasic.llmsverifier/llmverifier"
)

// Request represents a verification request
type Request struct {
	ModelID string `json:"model_id"`
	Prompt  string `json:"prompt"`
}

// Verifier handles model verification
type Verifier struct {
	db     *database.Database
	prober *llmverifier.Verifier
}

// NewVerifier creates a new verifier without a probe engine. Verify on such a
// verifier surfaces ErrVerifierNotConfigured rather than fabricating a result —
// use NewVerifierWithProber to obtain a verifier that dispatches real probes.
func NewVerifier(db *database.Database) *Verifier {
	return &Verifier{
		db: db,
	}
}

// NewVerifierWithProber wires a real llmverifier.Verifier probe engine into the
// VerificationService (change C5, 10b_code_exact_change_spec.md §3 C5). The
// prober carries the provider endpoint + API key via its injected config; Verify
// dispatches the per-capability probes (C4) through it and composes + persists a
// database.VerificationResult from the REAL outcomes.
func NewVerifierWithProber(db *database.Database, prober *llmverifier.Verifier) *Verifier {
	return &Verifier{
		db:     db,
		prober: prober,
	}
}

// Verify performs real model verification: it resolves the model, dispatches the
// C4 per-capability probes via the wired llmverifier.Verifier, composes a
// database.VerificationResult from the actual probe outcomes (including the
// CONST-040 RAG/Skills/Plugins capabilities), and persists it. The composed +
// persisted result is what the C3 fail-closed resolver later reads as the fresh,
// probe-sourced source of truth.
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

	// A verifier with no database + probe engine cannot verify anything. Surface
	// the gap loudly instead of fabricating a result — the previous hardcoded
	// all-capabilities-true return was a §11.4 / CONST-036/037 PASS-bluff.
	if v.db == nil || v.prober == nil {
		return nil, ErrVerifierNotConfigured
	}

	// (1) Resolve req.ModelID (the provider model identifier string) to the
	// int64 primary key via the models table.
	models, err := v.db.ListModels(map[string]interface{}{"model_id": req.ModelID})
	if err != nil {
		return nil, fmt.Errorf("failed to resolve model %q: %w", req.ModelID, err)
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("%w: %q", ErrModelNotFound, req.ModelID)
	}
	model := models[0]

	// (2) Dispatch the real per-capability probes (C4) through the prober. Probe
	// transport errors yield clean false verdicts inside detectFeatures; a
	// non-nil error here means the probe engine itself could not run.
	startedAt := time.Now()
	features, err := v.prober.DetectModelFeatures(req.ModelID)
	if err != nil {
		return nil, fmt.Errorf("feature detection failed for %q: %w", req.ModelID, err)
	}
	completedAt := time.Now()

	// (3) Compose a database.VerificationResult from the REAL probe outcomes,
	// including the CONST-040 RAG/Skills/Plugins flags sourced from the C4 probes.
	rawReq, _ := json.Marshal(map[string]interface{}{
		"model_id":          req.ModelID,
		"prompt":            req.Prompt,
		"verification_type": "capability",
	})
	rawResp, _ := json.Marshal(features)
	rawReqStr := string(rawReq)
	rawRespStr := string(rawResp)

	result := &database.VerificationResult{
		ModelID:                  model.ID,
		VerificationType:         "capability",
		StartedAt:                startedAt,
		CompletedAt:              &completedAt,
		Status:                   "completed",
		SupportsToolUse:          features.ToolUse,
		SupportsFunctionCalling:  features.FunctionCalling,
		SupportsCodeGeneration:   features.CodeGeneration,
		SupportsCodeCompletion:   features.CodeCompletion,
		SupportsCodeReview:       features.CodeReview,
		SupportsCodeExplanation:  features.CodeExplanation,
		SupportsEmbeddings:       features.Embeddings,
		SupportsReranking:        features.Reranking,
		SupportsImageGeneration:  features.ImageGeneration,
		SupportsAudioGeneration:  features.AudioGeneration,
		SupportsVideoGeneration:  features.VideoGeneration,
		SupportsMCPs:             features.MCPs,
		SupportsLSPs:             features.LSPs,
		SupportsACPs:             features.ACPs,
		SupportsRAG:              features.RAG,
		SupportsSkills:           features.Skills,
		SupportsPlugins:          features.Plugins,
		SupportsMultimodal:       features.Multimodal,
		SupportsStreaming:        features.Streaming,
		SupportsJSONMode:         features.JSONMode,
		SupportsStructuredOutput: features.StructuredOutput,
		SupportsReasoning:        features.Reasoning,
		SupportsParallelToolUse:  features.ParallelToolUse,
		MaxParallelCalls:         features.MaxParallelCalls,
		SupportsBatchProcessing:  features.BatchProcessing,
		SupportsBrotli:           features.SupportsBrotli,
		CodeLanguageSupport:      []string{},
		RawRequest:               &rawReqStr,
		RawResponse:              &rawRespStr,
		CreatedAt:                time.Now(),
	}

	// (4) Persist the composed result so the C3 resolver can read it.
	if err := v.db.CreateVerificationResult(result); err != nil {
		return nil, fmt.Errorf("failed to persist verification result for %q: %w", req.ModelID, err)
	}

	return result, nil
}

// ErrVerifierNotConfigured is returned by Verify when the verifier has no
// database + probe engine wired (e.g. constructed via NewVerifier(nil)). It
// keeps the honest contract that Verify never fabricates a result: the previous
// hardcoded all-capabilities-true return was a §11.4 / CONST-036/037 PASS-bluff
// and is removed — construct via NewVerifierWithProber(db, llmverifier.New(cfg))
// to obtain a verifier that dispatches real probes and persists real results.
var ErrVerifierNotConfigured = errors.New("verification.Verify: verifier is not configured with a database + probe engine (the previous hardcoded-all-capabilities-true return was a §11.4 / CONST-036/037 PASS-bluff); construct via NewVerifierWithProber(db, llmverifier.New(cfg))")

// ErrModelNotFound is returned by Verify when req.ModelID does not resolve to a
// row in the models table.
var ErrModelNotFound = errors.New("verification.Verify: model not found in the models table")

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
