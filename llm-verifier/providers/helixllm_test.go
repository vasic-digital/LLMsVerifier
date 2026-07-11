package providers

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"digital.vasic.llmsverifier/capabilities"
	"digital.vasic.llmsverifier/config"
	"digital.vasic.llmsverifier/database"
	"digital.vasic.llmsverifier/llmverifier"
	"digital.vasic.llmsverifier/verification"

	"github.com/stretchr/testify/require"
)

// TestHelixLLM_EndpointResolution_EnvOverride proves the
// HELIX_LLM_LOCAL_OPENAI_ENDPOINT override is honoured, WITH or WITHOUT a
// trailing /v1 in the operator-supplied value — the same normalisation
// contract the sibling HelixAgent HelixLLM provider adapter documents
// (submodules/helix_agent/internal/llm/providers/helixllm/provider.go,
// normalizeBase/resolveEndpoint).
func TestHelixLLM_EndpointResolution_EnvOverride(t *testing.T) {
	t.Run("default (env unset)", func(t *testing.T) {
		t.Setenv(helixLLMLocalOpenAIEndpointEnv, "")
		require.Equal(t, "http://localhost:18434/v1", helixLLMEndpoint())
	})
	t.Run("env WITHOUT trailing /v1", func(t *testing.T) {
		t.Setenv(helixLLMLocalOpenAIEndpointEnv, "http://example-host:9999")
		require.Equal(t, "http://example-host:9999/v1", helixLLMEndpoint())
	})
	t.Run("env WITH trailing /v1", func(t *testing.T) {
		t.Setenv(helixLLMLocalOpenAIEndpointEnv, "http://example-host:9999/v1")
		require.Equal(t, "http://example-host:9999/v1", helixLLMEndpoint())
	})
	t.Run("env WITH trailing slash", func(t *testing.T) {
		t.Setenv(helixLLMLocalOpenAIEndpointEnv, "http://example-host:9999/v1/")
		require.Equal(t, "http://example-host:9999/v1", helixLLMEndpoint())
	})
}

// TestHelixLLM_LiveDiscovery_RealCoder is the §11.4.5/§11.4.69 LIVE proof: a
// real GET against the in-repo HelixLLM coder's OpenAI-compatible
// /v1/models endpoint (ProbeProviderReachability — the SAME reachability
// probe c696c5db proved against the 13 cloud extended-provider rows).
// Honest SKIP (never a faked PASS) when the coder is not reachable
// (§11.4.3) — this session it IS live at http://localhost:18434.
func TestHelixLLM_LiveDiscovery_RealCoder(t *testing.T) {
	if testing.Short() {
		t.Skip("SKIP-OK: live helixllm coder discovery skipped in -short mode")
	}

	pr := NewProviderRegistry()
	cfg, ok := pr.GetConfig("helixllm")
	require.True(t, ok)

	client := createHTTPClient()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	res := ProbeProviderReachability(ctx, client, cfg.Endpoint, "" /* local coder needs no credential */)
	if res.TransportErr != nil {
		t.Skipf("SKIP-OK: helixllm-not-reachable: %v (endpoint %s)", res.TransportErr, cfg.Endpoint)
	}
	require.Equal(t, 200, res.StatusCode, "live GET %s/models unexpected status", cfg.Endpoint)
	require.NotNil(t, res.Models)
	require.NotEmpty(t, res.Models.Data, "helixllm coder returned no models (CONST-036 discovery expected >=1)")

	modelID := res.Models.Data[0].ID
	require.NotEmpty(t, modelID, "helixllm coder's first model id must be non-empty")
	t.Logf("PASS live helixllm discovery: %s/models -> %d model(s), first id=%q", cfg.Endpoint, len(res.Models.Data), modelID)

	if evidencePath := os.Getenv("HELIXLLM_LIVE_EVIDENCE"); evidencePath != "" {
		payload, _ := json.Marshal(map[string]interface{}{
			"provider": "helixllm", "endpoint": cfg.Endpoint, "status": res.StatusCode,
			"model_id": modelID, "models_count": len(res.Models.Data),
		})
		_ = os.WriteFile(evidencePath, payload, 0o600)
	}
}

// TestHelixLLM_LiveVerification_RealProbe is the full LIVE proof tying
// HelixLLM's Phase-A provider registration into the landed C3/C4/C5
// capability chain (EXPANSION_PLAN_v2.md §0.1): it discovers the coder's
// REAL model id via GET /v1/models (CONST-036 — no hardcoded model id),
// dispatches the REAL per-capability probe battery
// (llmverifier.Verifier.DetectModelFeatures — real chat-completion HTTP
// round-trips against the live coder, no simulation), persists the composed
// database.VerificationResult (C5), and confirms the C3 fail-closed resolver
// (capabilities.ResolveModelCapability) now reports the capability VERIFIED
// with the REAL probed value — never the seed's hand-authored literal.
// Honest SKIP (§11.4.3), never a faked PASS, when the coder is not
// reachable.
func TestHelixLLM_LiveVerification_RealProbe(t *testing.T) {
	if testing.Short() {
		t.Skip("SKIP-OK: live helixllm verification skipped in -short mode")
	}

	pr := NewProviderRegistry()
	cfg, ok := pr.GetConfig("helixllm")
	require.True(t, ok)

	// (1) Real discovery: GET <endpoint>/models against the live coder.
	httpClient := createHTTPClient()
	discoverCtx, cancelDiscover := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelDiscover()
	disc := ProbeProviderReachability(discoverCtx, httpClient, cfg.Endpoint, "")
	if disc.TransportErr != nil {
		t.Skipf("SKIP-OK: helixllm-not-reachable: %v (endpoint %s)", disc.TransportErr, cfg.Endpoint)
	}
	require.Equal(t, 200, disc.StatusCode)
	require.NotNil(t, disc.Models)
	require.NotEmpty(t, disc.Models.Data, "helixllm coder returned no models")
	modelID := disc.Models.Data[0].ID
	require.NotEmpty(t, modelID)
	t.Logf("live discovery: %s/models -> model_id=%q", cfg.Endpoint, modelID)

	// (2) Real in-memory SQLite DB (no mocks — CONST-050(A)) + a real
	// provider+model row seeded with the JUST-discovered model id (mirrors
	// capabilities/registry_resolve_test.go's seedModel helper pattern).
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	dbProvider := &database.Provider{Name: "helixllm", Endpoint: cfg.Endpoint, IsActive: true}
	require.NoError(t, db.CreateProvider(dbProvider))
	dbModel := &database.Model{
		ProviderID:  dbProvider.ID,
		ModelID:     modelID,
		Name:        modelID,
		Description: "HelixLLM live coder model (Phase A registration proof, §11.4.5/§11.4.69)",
	}
	require.NoError(t, db.CreateModel(dbModel))

	// (3) Real C4/C5 probe dispatch: llmverifier.Verifier built against the
	// LIVE coder endpoint, no simulated client, no faked response.
	prober := llmverifier.New(&config.Config{
		Global: config.GlobalConfig{
			BaseURL:      cfg.Endpoint,
			APIKey:       "",
			DefaultModel: modelID,
			Timeout:      60 * time.Second,
		},
		Timeout: 5 * time.Minute, // overall context for the full probe battery
	})
	verifier := verification.NewVerifierWithProber(db, prober)

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	result, err := verifier.Verify(ctx, &verification.Request{
		ModelID: modelID,
		Prompt:  "Write a Go function that returns the sum of two integers.",
	})
	require.NoError(t, err, "real C4/C5 verification against the live helixllm coder must succeed")
	require.NotNil(t, result)
	require.Equal(t, "completed", result.Status)
	require.NotNil(t, result.RawRequest)
	require.NotNil(t, result.RawResponse)
	t.Logf("PASS live helixllm verification: Streaming=%v ToolUse=%v FunctionCalling=%v CodeGeneration=%v CodeCompletion=%v",
		result.SupportsStreaming, result.SupportsToolUse, result.SupportsFunctionCalling,
		result.SupportsCodeGeneration, result.SupportsCodeCompletion)

	// (4) Confirm the C3 fail-closed resolver now reports VERIFIED for a
	// capability with the REAL probed value — never registry.go's helixllm
	// seed literal (Streaming Supported=true is only a bootstrap default;
	// the resolver must report verified=true backed by THIS fresh persisted
	// probe, not the seed).
	value, verified, err := capabilities.ResolveModelCapability(db, dbModel.ID, "streaming")
	require.NoError(t, err)
	require.True(t, verified, "C3 resolver must report VERIFIED after a fresh persisted probe")
	require.Equal(t, result.SupportsStreaming, value, "resolver value must match the persisted real probe outcome")

	if evidencePath := os.Getenv("HELIXLLM_LIVE_EVIDENCE"); evidencePath != "" {
		payload, _ := json.Marshal(map[string]interface{}{
			"provider": "helixllm", "endpoint": cfg.Endpoint, "model_id": modelID,
			"verification_status":       result.Status,
			"supports_streaming":        result.SupportsStreaming,
			"supports_tool_use":         result.SupportsToolUse,
			"supports_function_calling": result.SupportsFunctionCalling,
			"supports_code_generation":  result.SupportsCodeGeneration,
			"resolver_verified":         verified,
			"resolver_value":            value,
		})
		_ = os.WriteFile(evidencePath, payload, 0o600)
	}
}
