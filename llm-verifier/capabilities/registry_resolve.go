package capabilities

import (
	"errors"
	"strings"
	"time"

	"digital.vasic.llmsverifier/database"
)

// C3 — fail-closed capability resolver.
//
// The static provider seed map (registry.go) is a hand-authored bootstrap
// default ONLY; it is `Verified:false` by construction (CONST-036/037/040).
// A capability the verifier has NOT confirmed MUST be reported as unverified —
// NEVER a hand-authored literal presented as authoritative.
//
// ResolveModelCapability prefers a FRESH per-model database.VerificationResult
// (produced + persisted by C4/C5 — see verification.Verify) and FAIL-CLOSES to
// `unverified` when no fresh probe exists. It is the single source of truth for
// "is capability X verified for model M": the seed is used only as the value
// hint, never as the verification verdict.
//
// Design decision (DB-handle acquisition): the change spec (10b §3 C3) routes
// accessors "through a resolver that prefers a fresh database.VerificationResult
// (per model)" but does NOT pin how the existing provider-keyed, DB-handle-less
// accessors (GetProviderBaseCapabilities/GetProvidersWithCapability) acquire a
// *database.Database or a per-model identity. Per §11.4.6 (no-guessing) the
// least-breaking choice is taken: ADD a new verification-aware resolver variant
// that takes the DB handle + model PK explicitly, leaving the existing
// provider-keyed accessors intact for back-compat (their seed `.Verified` is now
// honestly `false`). `database` does not import `capabilities`, so importing
// `database` here introduces no cycle.

// verificationFreshnessWindow is the maximum age at which a persisted
// VerificationResult is treated as authoritative — CONST-037's 24h
// re-verification window (the outer bound). CONST-038's 60s real-time-status
// window is a tighter poll cadence layered on top by consumers; the resolver's
// "is there a fresh probe at all" gate uses the 24h outer bound.
const verificationFreshnessWindow = 24 * time.Hour

// ErrUnknownCapability is returned when capName is not a resolvable capability.
// Returning an error (rather than a silent `false`) keeps the resolver honest:
// an unmapped capability is a programming error, never a fail-closed "no".
var ErrUnknownCapability = errors.New("capabilities: unknown capability name")

// ResolveModelCapability resolves whether model `modelID` (the models-table PK)
// supports capability `capName`, PREFERRING a fresh persisted probe and
// FAIL-CLOSING to unverified when none exists.
//
// Returns:
//   - value:    the capability's boolean value from the fresh probe (only
//     meaningful when verified==true; false when fail-closed).
//   - verified: true ONLY when a fresh (within CONST-037 24h) completed
//     VerificationResult backs the value; false when no probe / stale probe.
//   - err:      ErrUnknownCapability for an unmapped capName, or a wrapped DB
//     error. A nil db is treated as "no probe available" → fail-closed
//     (false,false,nil), never a panic.
//
// Fail-closed contract: absent a fresh probe the resolver reports
// (false, false, nil) — unverified — NEVER the seed's hand-authored literal.
func ResolveModelCapability(db *database.Database, modelID int64, capName string) (value bool, verified bool, err error) {
	// Validate the capability name up front so an unmapped name is an explicit
	// error rather than a silent fail-closed "no".
	if _, ok := capabilityIsMappable(capName); !ok {
		return false, false, ErrUnknownCapability
	}

	// No DB handle ⇒ no probe source ⇒ fail-closed to unverified.
	if db == nil {
		return false, false, nil
	}

	results, err := db.GetLatestVerificationResults([]int64{modelID})
	if err != nil {
		return false, false, err
	}
	if len(results) == 0 {
		// No probe has ever completed for this model ⇒ fail-closed.
		return false, false, nil
	}

	latest := results[0]
	if !verificationResultFresh(latest) {
		// A probe exists but is stale (or not completed) ⇒ fail-closed:
		// a stale verification is NOT authoritative (CONST-037/038).
		return false, false, nil
	}

	// Fresh, completed probe backs the value.
	return probedCapabilityValue(latest, capName), true, nil
}

// verificationResultFresh reports whether vr is a completed verification within
// the CONST-037 freshness window. A running/failed/cancelled result, or one
// with no completion timestamp, or one older than the window, is NOT fresh.
func verificationResultFresh(vr *database.VerificationResult) bool {
	if vr == nil {
		return false
	}
	if !strings.EqualFold(vr.Status, "completed") {
		return false
	}
	// Prefer CompletedAt (when the probe finished); fall back to StartedAt.
	stamp := vr.StartedAt
	if vr.CompletedAt != nil {
		stamp = *vr.CompletedAt
	}
	if stamp.IsZero() {
		return false
	}
	return time.Since(stamp) <= verificationFreshnessWindow
}

// capabilityIsMappable reports whether capName maps to a known VerificationResult
// capability field. The bool is the accessor into a *database.VerificationResult;
// the closed vocabulary is the CONST-040 capability set plus the core probe flags.
func capabilityIsMappable(capName string) (func(*database.VerificationResult) bool, bool) {
	fn, ok := verificationCapabilityAccessors[normalizeCapabilityName(capName)]
	return fn, ok
}

// probedCapabilityValue reads capName's boolean from a fresh VerificationResult.
// Callers MUST have validated capName via capabilityIsMappable first.
func probedCapabilityValue(vr *database.VerificationResult, capName string) bool {
	if fn, ok := verificationCapabilityAccessors[normalizeCapabilityName(capName)]; ok {
		return fn(vr)
	}
	return false
}

// normalizeCapabilityName lower-cases + trims a capability name so callers can
// pass "RAG" / "rag" / "tool_use" / "ToolUse" interchangeably.
func normalizeCapabilityName(capName string) string {
	return strings.ReplaceAll(strings.TrimSpace(strings.ToLower(capName)), "-", "_")
}

// verificationCapabilityAccessors maps a normalized capability name to the
// corresponding boolean on a database.VerificationResult. This is the closed
// vocabulary of resolvable capabilities (CONST-040 protocol/capability set +
// the core probe flags). Extending it is additive (new probe → new entry).
var verificationCapabilityAccessors = map[string]func(*database.VerificationResult) bool{
	// CONST-040 capability set.
	"rag":     func(vr *database.VerificationResult) bool { return vr.SupportsRAG },
	"skills":  func(vr *database.VerificationResult) bool { return vr.SupportsSkills },
	"plugins": func(vr *database.VerificationResult) bool { return vr.SupportsPlugins },
	"mcp":     func(vr *database.VerificationResult) bool { return vr.SupportsMCPs },
	"mcps":    func(vr *database.VerificationResult) bool { return vr.SupportsMCPs },
	"lsp":     func(vr *database.VerificationResult) bool { return vr.SupportsLSPs },
	"lsps":    func(vr *database.VerificationResult) bool { return vr.SupportsLSPs },
	"acp":     func(vr *database.VerificationResult) bool { return vr.SupportsACPs },
	"acps":    func(vr *database.VerificationResult) bool { return vr.SupportsACPs },

	// Core probe flags.
	"tool_use":          func(vr *database.VerificationResult) bool { return vr.SupportsToolUse },
	"function_calling":  func(vr *database.VerificationResult) bool { return vr.SupportsFunctionCalling },
	"parallel_tool_use": func(vr *database.VerificationResult) bool { return vr.SupportsParallelToolUse },
	"embeddings":        func(vr *database.VerificationResult) bool { return vr.SupportsEmbeddings },
	"reranking":         func(vr *database.VerificationResult) bool { return vr.SupportsReranking },
	"reasoning":         func(vr *database.VerificationResult) bool { return vr.SupportsReasoning },
	"multimodal":        func(vr *database.VerificationResult) bool { return vr.SupportsMultimodal },
	"streaming":         func(vr *database.VerificationResult) bool { return vr.SupportsStreaming },
	"json_mode":         func(vr *database.VerificationResult) bool { return vr.SupportsJSONMode },
	"structured_output": func(vr *database.VerificationResult) bool { return vr.SupportsStructuredOutput },
	"batch_processing":  func(vr *database.VerificationResult) bool { return vr.SupportsBatchProcessing },
	"brotli":            func(vr *database.VerificationResult) bool { return vr.SupportsBrotli },
}
