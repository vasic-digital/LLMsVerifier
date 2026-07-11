package providers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newExtendedProviderKeys are the 11 provider config records ADDED in this
// coverage pass (config.go). xiaomi/novita were already registered before and
// hyperbolic is pre-existing, so they are not in this "added" set.
var newExtendedProviderKeys = []string{
	"poe", "perplexity", "sakana", "hunyuan", "xai",
	"moonshot", "zai", "fireworks", "deepinfra", "ai21", "reka",
}

// expectedExtendedBaseURLs pins the LATEST-doc-verified base URL for each newly
// added record (PROVIDER_COVERAGE.md §1.1, verified 2026-07-06). Dropping or
// mangling a record makes this table-driven test FAIL (the §1.1 mutation guard).
var expectedExtendedBaseURLs = map[string]string{
	"poe":        "https://api.poe.com/v1",
	"perplexity": "https://api.perplexity.ai",
	"sakana":     "https://api.sakana.ai/v1",
	"hunyuan":    "https://api.hunyuan.cloud.tencent.com/v1",
	"xai":        "https://api.x.ai/v1",
	"moonshot":   "https://api.moonshot.ai/v1",
	"zai":        "https://api.z.ai/api/paas/v4",
	"fireworks":  "https://api.fireworks.ai/inference/v1",
	"deepinfra":  "https://api.deepinfra.com/v1/openai",
	"ai21":       "https://api.ai21.com/studio/v1",
	"reka":       "https://api.reka.ai/v1",
}

// TestExtendedProviders_RecordsRegistered_Polarity is the §11.4.115 RED/GREEN
// polarity guard for the newly-added extended provider config records.
//
//	RED_MODE=1 (reproduce-on-broken-artifact): asserts the records are ABSENT —
//	           it PASSes ONLY on a pre-fix build (records not yet added) and
//	           FAILs on the current fixed build (records present). This is the
//	           honest reproduction proof, not a green-only test.
//	RED_MODE=0 (default, standing regression guard): asserts every added record
//	           IS present in the registry — dropping any record FAILs the gate.
func TestExtendedProviders_RecordsRegistered_Polarity(t *testing.T) {
	pr := NewProviderRegistry()

	var missing []string
	for _, name := range newExtendedProviderKeys {
		if _, ok := pr.GetConfig(name); !ok {
			missing = append(missing, name)
		}
	}

	if os.Getenv("RED_MODE") == "1" {
		require.NotEmpty(t, missing,
			"RED_MODE=1 reproduces the pre-fix defect: expected the extended provider records to be ABSENT, "+
				"but they are present (%d/%d). Run RED_MODE=1 against the pre-fix build to see it PASS.",
			len(newExtendedProviderKeys)-len(missing), len(newExtendedProviderKeys))
		t.Logf("RED_MODE=1: reproduced absence of %d extended records (pre-fix artifact)", len(missing))
		return
	}

	require.Empty(t, missing,
		"GREEN guard: extended provider records missing from the registry: %v", missing)
	t.Logf("GREEN guard: all %d extended provider records registered", len(newExtendedProviderKeys))
}

// TestExtendedProviders_BaseURLsAndAuth asserts each newly-added record carries
// the exact LATEST-doc-verified base URL, Bearer auth, and OpenAI-compat marker.
func TestExtendedProviders_BaseURLsAndAuth(t *testing.T) {
	pr := NewProviderRegistry()

	for _, name := range newExtendedProviderKeys {
		name := name
		t.Run(name, func(t *testing.T) {
			cfg, ok := pr.GetConfig(name)
			require.True(t, ok, "provider %q must be registered", name)
			assert.Equal(t, expectedExtendedBaseURLs[name], cfg.Endpoint,
				"provider %q base URL must match PROVIDER_COVERAGE.md §1.1", name)
			assert.Equal(t, "bearer", cfg.AuthType, "provider %q must use Bearer auth (§11.4.10)", name)
			assert.Equal(t, true, cfg.Features["openai_compatible"],
				"provider %q must be marked openai_compatible", name)
			// Every record must document which env var supplies its key.
			assert.NotEmpty(t, cfg.Features["env_var"], "provider %q must declare its env_var", name)
			// Every record must cite its verified doc URL (§11.4.99).
			assert.NotEmpty(t, cfg.Features["doc_url"], "provider %q must cite its verified doc_url", name)
		})
	}
}

// TestExtendedProviders_NoHardcodedModelList enforces CONST-036: the added
// records MUST NOT ship a hardcoded model list — the live set is discovered from
// each provider's own /v1/models. supported_models must be an EMPTY slice and
// DefaultModel must be empty (discovery-driven).
func TestExtendedProviders_NoHardcodedModelList(t *testing.T) {
	pr := NewProviderRegistry()

	for _, name := range newExtendedProviderKeys {
		name := name
		t.Run(name, func(t *testing.T) {
			cfg, ok := pr.GetConfig(name)
			require.True(t, ok)

			assert.Empty(t, cfg.DefaultModel,
				"CONST-036: provider %q must not hardcode a DefaultModel", name)

			raw, present := cfg.Features["supported_models"]
			require.True(t, present, "provider %q must declare supported_models (empty per CONST-036)", name)
			models, ok := raw.([]string)
			require.True(t, ok, "provider %q supported_models must be []string", name)
			assert.Empty(t, models,
				"CONST-036: provider %q must not hardcode a model list; got %v", name, models)
		})
	}
}

// TestExtendedProviders_ConfigParseRoundTrip proves each added record is a valid
// data record that JSON-marshals and unmarshals losslessly (register + parse).
func TestExtendedProviders_ConfigParseRoundTrip(t *testing.T) {
	pr := NewProviderRegistry()

	for _, name := range newExtendedProviderKeys {
		name := name
		t.Run(name, func(t *testing.T) {
			cfg, ok := pr.GetConfig(name)
			require.True(t, ok)

			data, err := json.Marshal(cfg)
			require.NoError(t, err, "provider %q config must marshal", name)

			var parsed ProviderConfig
			require.NoError(t, json.Unmarshal(data, &parsed), "provider %q config must unmarshal", name)

			assert.Equal(t, cfg.Name, parsed.Name)
			assert.Equal(t, cfg.Endpoint, parsed.Endpoint)
			assert.Equal(t, cfg.AuthType, parsed.AuthType)
		})
	}
}

// TestExtendedProviders_DocumentedPendingNotActivated verifies the UNVERIFIED /
// BLOCKED-until-GA providers are recorded as documented-pending DATA and are
// NOT registered as active endpoints (§11.4.6 — no endpoint invented).
func TestExtendedProviders_DocumentedPendingNotActivated(t *testing.T) {
	pr := NewProviderRegistry()
	pending := DocumentedPendingProviders()

	byName := make(map[string]PendingProviderRow, len(pending))
	for _, row := range pending {
		byName[row.Name] = row
		// A pending provider MUST carry a reason + enumerated unblock choices.
		assert.NotEmpty(t, row.Reason, "pending provider %q must state a reason", row.Name)
		assert.NotEmpty(t, row.UnblockChoices, "pending provider %q must enumerate unblock choices", row.Name)
	}

	// Baseten: documented-pending (per-deployment base URL, §1.2).
	baseten, ok := byName["baseten"]
	require.True(t, ok, "baseten must be recorded as documented-pending")
	assert.Equal(t, StatusDocumentedPending, baseten.Status)
	assert.False(t, pr.IsProviderSupported("baseten"),
		"baseten must NOT be activated as a provider (UNVERIFIED, §1.2)")

	// Subquadratic: blocked-until-GA (operator C4, §1.3).
	subq, ok := byName["subquadratic"]
	require.True(t, ok, "subquadratic must be recorded as blocked-until-ga")
	assert.Equal(t, StatusBlockedUntilGA, subq.Status)
	assert.False(t, pr.IsProviderSupported("subquadratic"),
		"subquadratic must NOT be activated (BLOCKED-until-GA, operator C4)")
}

// TestExtendedProviders_LiveReachability is the REAL reachability proof
// (§11.4.108). For every confirmed OpenAI-compatible extended provider it reads
// the provider's Bearer key from env/.env and, ONLY when present, makes a REAL
// GET <base>/models against the live endpoint, asserting HTTP success and at
// least one REAL model id (CONST-036 — discovery, never hardcoded). A missing
// key is an honest SKIP-with-reason `credential-absent` (§11.4.3), NEVER a faked
// PASS and NEVER a hardcoded response. The key value is never logged (§11.4.10).
//
// Evidence: each verdict is logged (captured in -v output) and, when
// PROVIDER_REACHABILITY_EVIDENCE points at a file, appended there as JSONL
// (sink-side artefact, §11.4.69).
func TestExtendedProviders_LiveReachability(t *testing.T) {
	if testing.Short() {
		t.Skip("SKIP-OK: live reachability skipped in -short mode")
	}
	loadDotEnvForTest(t)

	pr := NewProviderRegistry()
	evidencePath := os.Getenv("PROVIDER_REACHABILITY_EVIDENCE")

	type verdict struct {
		Provider  string `json:"provider"`
		EnvVar    string `json:"env_var"`
		Endpoint  string `json:"endpoint"`
		Result    string `json:"result"` // PASS | SKIP
		Reason    string `json:"reason,omitempty"`
		ModelID   string `json:"model_id,omitempty"`
		ModelsLen int    `json:"models_len,omitempty"`
		DocURL    string `json:"doc_url"`
	}

	var verdicts []verdict

	client := createHTTPClient()

	for _, meta := range ExtendedOpenAICompatProviders() {
		meta := meta
		t.Run(meta.RegistryKey, func(t *testing.T) {
			cfg, ok := pr.GetConfig(meta.RegistryKey)
			require.True(t, ok, "provider %q must be registered", meta.RegistryKey)

			key := strings.TrimSpace(os.Getenv(meta.EnvVar))
			if key == "" {
				reason := "credential-absent: " + meta.EnvVar + " not set"
				verdicts = append(verdicts, verdict{
					Provider: meta.RegistryKey, EnvVar: meta.EnvVar, Endpoint: cfg.Endpoint,
					Result: "SKIP", Reason: reason, DocURL: meta.DocURL,
				})
				t.Logf("SKIP %s: %s", meta.RegistryKey, reason)
				t.Skipf("SKIP-OK: %s", reason)
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			res := ProbeProviderReachability(ctx, client, cfg.Endpoint, key)

			// Transport-level failure (DNS / refused / timeout / malformed 200
			// body) is a genuine reachability/base-URL defect → FAIL.
			require.NoError(t, res.TransportErr,
				"live GET %s/models transport error for %s", cfg.Endpoint, meta.RegistryKey)

			// Account-state HTTP status: the endpoint IS reachable and the base
			// URL is correct, but the account/credential state blocks the
			// listing. Honest account-unavailable SKIP (§11.4.3) — never a code
			// FAIL, never a faked PASS. The response body is NOT surfaced
			// (may contain account identifiers, §11.4.10) — only the status code.
			if res.StatusCode != http.StatusOK {
				if AccountUnavailableStatus(res.StatusCode) {
					reason := fmt.Sprintf("credential-present-but-account-unavailable: HTTP %d "+
						"(base URL reachable; account auth/billing/quota state blocks /v1/models)", res.StatusCode)
					verdicts = append(verdicts, verdict{
						Provider: meta.RegistryKey, EnvVar: meta.EnvVar, Endpoint: cfg.Endpoint,
						Result: "SKIP", Reason: reason, DocURL: meta.DocURL,
					})
					t.Logf("SKIP %s: %s", meta.RegistryKey, reason)
					t.Skipf("SKIP-OK: %s", reason)
					return
				}
				// 5xx or any other unexpected status → genuine defect.
				t.Fatalf("live GET %s/models for %s returned unexpected HTTP %d",
					cfg.Endpoint, meta.RegistryKey, res.StatusCode)
			}

			require.NotNil(t, res.Models)
			require.NotEmpty(t, res.Models.Data,
				"provider %q returned no models (CONST-036 discovery expected ≥1)", meta.RegistryKey)

			modelID := res.Models.Data[0].ID
			require.NotEmpty(t, modelID, "provider %q first model id must be non-empty", meta.RegistryKey)

			verdicts = append(verdicts, verdict{
				Provider: meta.RegistryKey, EnvVar: meta.EnvVar, Endpoint: cfg.Endpoint,
				Result: "PASS", ModelID: modelID, ModelsLen: len(res.Models.Data), DocURL: meta.DocURL,
			})
			t.Logf("PASS %s: live /v1/models returned %d models, e.g. %q", meta.RegistryKey, len(res.Models.Data), modelID)
		})
	}

	// Persist a sink-side evidence artefact if a destination was provided.
	if evidencePath != "" {
		f, err := os.OpenFile(evidencePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			t.Logf("evidence write skipped: %v", err)
			return
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		for _, v := range verdicts {
			_ = enc.Encode(v)
		}
		t.Logf("wrote %d reachability verdicts to %s", len(verdicts), evidencePath)
	}
}

// loadDotEnvForTest loads KEY=VALUE pairs from a .env file (module or submodule
// root) into the environment IF the variable is not already set. Test-only
// convenience so a key placed in .env drives the live proof. Values are never
// logged (§11.4.10). Missing .env is a no-op (keys stay absent → honest SKIP).
func loadDotEnvForTest(t *testing.T) {
	t.Helper()
	candidates := []string{".env", "../.env", "../../.env", "../../../.env"}
	for _, p := range candidates {
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		f, err := os.Open(abs)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			line = strings.TrimPrefix(line, "export ")
			eq := strings.IndexByte(line, '=')
			if eq <= 0 {
				continue
			}
			k := strings.TrimSpace(line[:eq])
			v := strings.Trim(strings.TrimSpace(line[eq+1:]), `"'`)
			if k != "" && os.Getenv(k) == "" {
				_ = os.Setenv(k, v)
			}
		}
		f.Close()
	}
}
