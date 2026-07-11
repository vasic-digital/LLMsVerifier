# P4 Extended OpenAI-compatible provider coverage — captured evidence

**Run-id:** phase4_extended_providers_20260707
**Branch:** feature/helixllm-full-extension
**Design:** `docs/research/07.2026/00_master/PROVIDER_COVERAGE.md` (parent repo, commit 1e6f3347)
**Scope:** 11 new OpenAI-compatible provider config records + real reachability proof (§11.4.108 / §11.4.5, anti-bluff).

All secrets redacted (§11.4.10). API keys are NEVER logged/committed; the evidence
records only provider name, endpoint, discovered model id, and doc URL.

---

## 1. Providers added (config-only reuse, 0 net-new wire adapters)

`llm-verifier/providers/config.go` — 11 new `ProviderConfig` records, base URLs
from PROVIDER_COVERAGE.md §1.1 (LATEST-doc-verified 2026-07-06). CONST-036:
`supported_models` EMPTY + `DefaultModel` empty (discovery via live `/v1/models`).
Bearer key from `<UPPER>_API_KEY` env var (§11.4.10).

| key | base URL |
|-----|----------|
| poe | https://api.poe.com/v1 |
| perplexity | https://api.perplexity.ai |
| sakana | https://api.sakana.ai/v1 |
| hunyuan | https://api.hunyuan.cloud.tencent.com/v1 |
| xai | https://api.x.ai/v1 |
| moonshot | https://api.moonshot.ai/v1 |
| zai | https://api.z.ai/api/paas/v4 |
| fireworks | https://api.fireworks.ai/inference/v1 |
| deepinfra | https://api.deepinfra.com/v1/openai |
| ai21 | https://api.ai21.com/studio/v1 |
| reka | https://api.reka.ai/v1 |

Already registered before this pass (§1.1 confirmed, base URL matches doc; NOT
re-added): `xiaomi`, `novita`. Pre-existing (untouched, §11.4.122): `hyperbolic`.

Documented-pending (NOT activated — `providers/extended_providers.go`
`DocumentedPendingProviders()`): `baseten` (documented-pending, per-deployment
base URL, §1.2), `subquadratic` (blocked-until-ga, operator C4, §1.3). No
endpoint invented (§11.4.6).

## 2. Build (EXIT 0)

```
$ GOMAXPROCS=2 nice -n 19 go build ./...
BUILD_EXIT=0
$ go vet ./providers/    # clean
```

## 3. Real reachability proof (§11.4.108) — live GET /v1/models

Keys present in the repo `.env` for a subset; absent keys → honest SKIP
(`credential-absent`, §11.4.3); reachable-but-account-blocked → honest SKIP
(`credential-present-but-account-unavailable`, HTTP status only, body not
surfaced). Raw JSONL: `reachability_evidence.jsonl`.

```
--- PASS  poe:     live /v1/models returned 341 models, e.g. "assistant"
--- SKIP  perplexity: credential-absent: PERPLEXITY_API_KEY not set
--- SKIP  sakana:     credential-absent: SAKANA_API_KEY not set
--- SKIP  hunyuan:    credential-absent: HUNYUAN_API_KEY not set
--- SKIP  xai:        credential-absent: XAI_API_KEY not set
--- SKIP  moonshot:   credential-absent: MOONSHOT_API_KEY not set
--- PASS  zai:      live /v1/models returned 8 models, e.g. "glm-4.5"
--- SKIP  fireworks:  credential-present-but-account-unavailable: HTTP 412
--- SKIP  deepinfra:  credential-absent: DEEPINFRA_API_KEY not set
--- SKIP  ai21:       credential-absent: AI21_API_KEY not set
--- SKIP  reka:       credential-absent: REKA_API_KEY not set
--- SKIP  xiaomi:     credential-absent: XIAOMI_API_KEY not set
--- PASS  novita:   live /v1/models returned 143 models, e.g. "tencent/hy3"
--- PASS: TestExtendedProviders_LiveReachability (3.20s)
ok  	digital.vasic.llmsverifier/providers	3.203s
```

LIVE-PROVEN (real endpoint answered with real discovered model ids, CONST-036):
`poe`, `zai`, `novita`. `fireworks` base URL confirmed reachable (real provider
HTTP 412), account billing-suspended → honest SKIP, not a code failure.

## 4. RED-first polarity proof (§11.4.115)

`RED_MODE=1` reproduces the pre-fix defect (records ABSENT). On the FIXED build it
correctly FAILs (records now present) — proving the guard is real, not green-only:

```
$ RED_MODE=1 go test -run TestExtendedProviders_RecordsRegistered_Polarity ./providers/
--- FAIL: TestExtendedProviders_RecordsRegistered_Polarity
    RED_MODE=1 reproduces the pre-fix defect: expected the extended provider
    records to be ABSENT, but they are present (11/11).
FAIL
```

`RED_MODE=0` (default GREEN guard) PASSes; dropping any record → FAILs (the §1.1
mutation guard).

## 5. Full touched-package suite (regression — GREEN)

```
$ GOMAXPROCS=2 nice -n 19 go test -count=1 ./providers/
ok  	digital.vasic.llmsverifier/providers	14.063s
```

New tests: `TestExtendedProviders_RecordsRegistered_Polarity`,
`_BaseURLsAndAuth`, `_NoHardcodedModelList` (CONST-036), `_ConfigParseRoundTrip`,
`_DocumentedPendingNotActivated`, `_LiveReachability` — all PASS/SKIP-honest.

## Anchors

§11.4.108 (runtime signature: real `/v1/models` round-trip on a clean build) ·
§11.4.115 (RED-on-broken polarity) · §11.4.3 (honest SKIP-with-reason) ·
§11.4.5/§11.4.69 (captured sink-side evidence artefact) · §11.4.10 (keys never
logged/committed) · §11.4.28 / CONST-045 / CONST-046 (provider is data) ·
CONST-036 (no hardcoded model list — live discovery) · §11.4.74 (reuse the
existing OpenAI-compatible adapter, 0 net-new wire adapters) · §11.4.6 (no
endpoint invented for baseten/subquadratic).
