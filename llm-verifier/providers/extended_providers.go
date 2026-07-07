package providers

// Extended OpenAI-compatible provider coverage (P4 provider-coverage phase).
//
// This file carries the *data* that accompanies the extended provider config
// records registered in config.go: the per-provider credential env-var name and
// the LATEST-doc-verified reference URL, plus the documented-pending providers
// that MUST NOT be activated yet.
//
// Source of truth for every base URL + doc URL:
//   docs/research/07.2026/00_master/PROVIDER_COVERAGE.md (verified 2026-07-06).
//
// A provider is DATA, not code (§11.4.28 decoupling, CONST-045 no-hardcoded-host,
// CONST-046 no-hardcoded-content). Base URLs are NOT duplicated here — the
// reachability layer reads the endpoint from the ProviderRegistry so there is a
// single source of truth for each endpoint.

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// ExtendedProviderMeta is the per-provider metadata needed to perform a REAL
// reachability probe: which registry key it maps to, which env var holds its
// Bearer key (§11.4.10 — never hardcoded/logged), and the authoritative doc URL
// the base URL was verified against (§11.4.99).
type ExtendedProviderMeta struct {
	// RegistryKey is the ProviderRegistry key (config.go) whose Endpoint is used.
	RegistryKey string
	// EnvVar is the environment variable holding the provider's API key.
	EnvVar string
	// DocURL is the LATEST authoritative doc the base URL was verified against.
	DocURL string
	// AlreadyRegisteredBefore is true for confirmed §1.1 providers that were
	// already present in the registry prior to this coverage pass (xiaomi,
	// novita) — recorded honestly so the "added" count is not overstated.
	AlreadyRegisteredBefore bool
}

// ExtendedOpenAICompatProviders returns the confirmed OpenAI-compatible hosted
// providers from PROVIDER_COVERAGE.md §1.1 (13 rows). Two of them (xiaomi,
// novita) were already registered before this pass and are flagged accordingly.
// The other 11 are newly added as config records in config.go.
//
// This slice is the driver for the live reachability proof: for each row, the
// harness reads EnvVar from the environment and — only when a key is present —
// makes a REAL GET <endpoint>/models against the live base URL. Absent key ⇒
// honest SKIP-with-reason `credential-absent` (§11.4.3), never a faked PASS.
func ExtendedOpenAICompatProviders() []ExtendedProviderMeta {
	return []ExtendedProviderMeta{
		{RegistryKey: "poe", EnvVar: "POE_API_KEY", DocURL: "https://creator.poe.com/docs/external-applications/openai-compatible-api"},
		{RegistryKey: "perplexity", EnvVar: "PERPLEXITY_API_KEY", DocURL: "https://docs.perplexity.ai/getting-started/overview"},
		{RegistryKey: "sakana", EnvVar: "SAKANA_API_KEY", DocURL: "https://console.sakana.ai/get-started"},
		{RegistryKey: "hunyuan", EnvVar: "HUNYUAN_API_KEY", DocURL: "https://cloud.tencent.com/document/product/1729/111007"},
		{RegistryKey: "xai", EnvVar: "XAI_API_KEY", DocURL: "https://docs.x.ai/docs/overview"},
		{RegistryKey: "moonshot", EnvVar: "MOONSHOT_API_KEY", DocURL: "https://platform.kimi.ai/docs/api/chat"},
		{RegistryKey: "zai", EnvVar: "ZAI_API_KEY", DocURL: "https://docs.z.ai/guides/overview/quick-start"},
		{RegistryKey: "fireworks", EnvVar: "FIREWORKS_API_KEY", DocURL: "https://docs.fireworks.ai/tools-sdks/openai-compatibility"},
		{RegistryKey: "deepinfra", EnvVar: "DEEPINFRA_API_KEY", DocURL: "https://docs.deepinfra.com/chat/overview"},
		{RegistryKey: "ai21", EnvVar: "AI21_API_KEY", DocURL: "https://docs.ai21.com/reference/jamba-1-6-api-ref"},
		{RegistryKey: "reka", EnvVar: "REKA_API_KEY", DocURL: "https://docs.reka.ai/chat/overview"},
		// Already registered before this pass (§1.1 confirmed, base URL matches doc):
		{RegistryKey: "xiaomi", EnvVar: "XIAOMI_API_KEY", DocURL: "https://platform.xiaomimimo.com/docs/en-US/api/chat/openai-api", AlreadyRegisteredBefore: true},
		{RegistryKey: "novita", EnvVar: "NOVITA_API_KEY", DocURL: "https://novita.ai/docs/guides/llm-api", AlreadyRegisteredBefore: true},
	}
}

// PendingProviderStatus is the closed set of non-active provider states.
type PendingProviderStatus string

const (
	// StatusDocumentedPending: base URL / auth NOT verified from an authoritative
	// page this pass — recorded, NOT activated, no endpoint invented (§11.4.6).
	StatusDocumentedPending PendingProviderStatus = "documented-pending"
	// StatusBlockedUntilGA: no public API exists yet — OPERATOR-BLOCKED (§11.4.21).
	StatusBlockedUntilGA PendingProviderStatus = "blocked-until-ga"
)

// PendingProviderRow records a provider that is deliberately NOT activated.
type PendingProviderRow struct {
	Name           string
	Status         PendingProviderStatus
	Reason         string
	UnblockChoices string // enumerated unblock choices (§11.4.148 D3 / §11.4.21)
	DocSection     string
}

// DocumentedPendingProviders returns the extended providers that MUST NOT be
// activated in this pass — recorded as data so they are tracked, never invented
// as endpoints (§11.4.6 no-guessing). Per PROVIDER_COVERAGE.md §1.2 / §1.3.
//
// Hyperbolic is NOT listed here: an active "hyperbolic" record already exists in
// the registry from prior work and is left untouched (§11.4.122 no-silent-removal);
// its base URL is nevertheless marked UNVERIFIED by PROVIDER_COVERAGE.md §1.2 and
// should be re-verified at build before being trusted.
func DocumentedPendingProviders() []PendingProviderRow {
	return []PendingProviderRow{
		{
			Name:           "baseten",
			Status:         StatusDocumentedPending,
			Reason:         "Baseten exposes per-model/per-deployment Model API hosts, so there is no single global base URL to hardcode (PROVIDER_COVERAGE.md §1.2). OpenAI-compatible per deployment.",
			UnblockChoices: "[A] inject a concrete per-deployment base URL as config (CONST-045) then reuse the OpenAI adapter; [B] wait for a documented global base URL",
			DocSection:     "PROVIDER_COVERAGE.md §1.2",
		},
		{
			Name:           "subquadratic",
			Status:         StatusBlockedUntilGA,
			Reason:         "No public base URL/auth — private beta, no adapter work possible until a public API or operator-supplied beta credentials exist (operator C4, PROVIDER_COVERAGE.md §1.3).",
			UnblockChoices: "[A] operator supplies beta credentials + base URL → promote to active; [B] wait for public GA; [C] drop from scope if the beta closes",
			DocSection:     "PROVIDER_COVERAGE.md §1.3 (operator C4)",
		},
	}
}

// ReachabilityResult is the outcome of a REAL GET <base>/models probe. It
// exposes the HTTP status code so the caller can classify honestly:
//   - 200 + ≥1 model     → the endpoint answered with real discovered models
//     (CONST-036), a genuine PASS.
//   - 401/402/403/407/412/429 → the endpoint IS reachable and the base URL is
//     correct (the request reached the provider and got a provider-shaped
//     auth/billing/quota/suspension response), but the account state blocks
//     listing — an honest account-unavailable SKIP (§11.4.3), NOT a code defect
//     and NOT a faked PASS.
//   - transport error / 5xx / 200-with-0-models → a genuine reachability or
//     base-URL defect (FAIL).
//
// The response BODY is deliberately NOT retained (it can contain account
// identifiers); only the status code + the parsed models list are returned so
// nothing sensitive is ever logged/committed (§11.4.10).
type ReachabilityResult struct {
	StatusCode int
	Models     *OpenAIModelsResponse
	// TransportErr is set only for connection-level failures (DNS, refused,
	// timeout) or a malformed 200 body — i.e. genuine defects, never an
	// account-state HTTP status.
	TransportErr error
}

// AccountUnavailableStatus reports whether an HTTP status means the endpoint is
// reachable + the base URL is correct but the account/credential state blocks
// the /v1/models listing (auth/payment/forbidden/proxy-auth/precondition/quota).
func AccountUnavailableStatus(status int) bool {
	switch status {
	case http.StatusUnauthorized, // 401
		http.StatusPaymentRequired,    // 402
		http.StatusForbidden,          // 403
		http.StatusProxyAuthRequired,  // 407
		http.StatusPreconditionFailed, // 412 (e.g. suspended account)
		http.StatusTooManyRequests:    // 429 (quota)
		return true
	}
	return false
}

// ProbeProviderReachability performs a REAL reachability probe against a
// provider's live GET <endpoint>/models over the OpenAI-compatible surface
// (§11.4.74 — same wire shape the GroqAdapter uses). The apiKey MUST come from
// the environment (§11.4.10); it is set as a Bearer header and never logged.
func ProbeProviderReachability(ctx context.Context, client *http.Client, endpoint, apiKey string) ReachabilityResult {
	if client == nil {
		client = &http.Client{}
	}
	url := strings.TrimSuffix(endpoint, "/") + "/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ReachabilityResult{TransportErr: err}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return ReachabilityResult{TransportErr: err}
	}
	defer resp.Body.Close()

	res := ReachabilityResult{StatusCode: resp.StatusCode}
	if resp.StatusCode == http.StatusOK {
		var models OpenAIModelsResponse
		if derr := json.NewDecoder(resp.Body).Decode(&models); derr != nil {
			res.TransportErr = derr
			return res
		}
		res.Models = &models
	}
	return res
}
