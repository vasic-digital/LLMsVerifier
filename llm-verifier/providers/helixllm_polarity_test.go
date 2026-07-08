package providers

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHelixLLM_ProviderRegistered_Polarity is the §11.4.115 RED/GREEN
// polarity guard proving HelixLLM is a TRACKED provider config record in
// this module's ProviderRegistry (config.go) — Phase A of the
// providers-coverage plan
// (docs/research/07.2026/06_providers_coverage/EXPANSION_PLAN_v2.md §3 Phase
// A). CONST-036/037/039: LLMsVerifier is the single source of truth; the
// in-repo HelixLLM local coder must be discoverable/verifiable exactly like
// every other provider.
//
// Deliberately isolated in its OWN minimal file, referencing ONLY
// NewProviderRegistry/GetConfig (symbols that already exist on the pre-fix
// artifact) — so this test compiles + runs cleanly against BOTH the pre-fix
// artifact (helixllm absent — RED_MODE=1 reproduces this) and the fixed
// artifact (helixllm present), with the RED assertion failing at the TEST
// level (never a build break) per §11.4.115. The rest of this change's
// tests (helixllm_test.go) reference the new config.go/registry.go symbols
// directly and are GREEN-only (post-fix) checks, mirroring the existing
// TestExtendedProviders_* precedent in extended_providers_test.go.
//
//	RED_MODE=1 (reproduce-on-broken-artifact): asserts the record is ABSENT —
//	           PASSes ONLY on a pre-fix build and FAILs on the current fixed
//	           build (record present).
//	RED_MODE=0 (default, standing regression guard): asserts the record IS
//	           present.
func TestHelixLLM_ProviderRegistered_Polarity(t *testing.T) {
	pr := NewProviderRegistry()
	_, ok := pr.GetConfig("helixllm")

	redMode := os.Getenv("RED_MODE")
	if redMode == "" {
		redMode = "0"
	}

	switch redMode {
	case "1":
		if ok {
			t.Fatalf("RED_MODE=1: expected the helixllm provider record to be ABSENT " +
				"(reproducing the pre-fix defect), but it is present")
		}
		t.Log("RED_MODE=1 PASS: reproduced absence of the helixllm provider record (pre-fix artifact)")
	case "0":
		require.True(t, ok, "GREEN guard: helixllm must be registered in the provider registry (CONST-036/039)")
		t.Log("RED_MODE=0 PASS: helixllm provider record registered")
	default:
		t.Fatalf("unknown RED_MODE=%q (expected 0 or 1)", redMode)
	}
}

// TestHelixLLM_ConfigShape asserts the registered row follows this module's
// established convention exactly: Endpoint carries the /v1 segment (matches
// this module's Endpoint+"/models" reachability convention — LIVE-CONFIRMED
// this session), Bearer AuthType (universal convention, even though the
// local coder needs no credential), openai_compatible + local_only markers,
// and CONST-036 no-hardcoded-model-list (empty DefaultModel + empty
// supported_models — discovered live instead).
//
// Deliberately kept in THIS minimal-dependency file (references only
// NewProviderRegistry/GetConfig — no reference to this change's new
// config.go helper symbols) so the whole file, including
// TestHelixLLM_ProviderRegistered_Polarity, compiles cleanly against both
// the pre-fix and fixed artifacts; the "HELIX_LLM_LOCAL_OPENAI_ENDPOINT"
// literal below is a TEST-OWNED expectation (not a production hardcode —
// the production side's identical literal lives in config.go's
// helixLLMLocalOpenAIEndpointEnv const, exercised directly by
// TestHelixLLM_EndpointResolution_EnvOverride in helixllm_test.go).
func TestHelixLLM_ConfigShape(t *testing.T) {
	pr := NewProviderRegistry()
	cfg, ok := pr.GetConfig("helixllm")
	require.True(t, ok)

	require.True(t, strings.HasSuffix(cfg.Endpoint, "/v1"),
		"helixllm endpoint must include the /v1 segment (matches this module's Endpoint+\"/models\" convention); got %q", cfg.Endpoint)
	require.Equal(t, "bearer", cfg.AuthType)
	require.Equal(t, "sse", cfg.StreamingFormat)
	require.Equal(t, true, cfg.Features["openai_compatible"])
	require.Equal(t, true, cfg.Features["local_only"])
	require.Equal(t, "HELIX_LLM_LOCAL_OPENAI_ENDPOINT", cfg.Features["env_var"])
	require.NotEmpty(t, cfg.Features["doc_url"])

	require.Empty(t, cfg.DefaultModel, "CONST-036: no hardcoded default model")
	models, ok := cfg.Features["supported_models"].([]string)
	require.True(t, ok, "supported_models must be []string")
	require.Empty(t, models, "CONST-036: no hardcoded model list")
}
