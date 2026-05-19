# LLMsVerifier — Test Coverage Matrix (round-296)

This document tracks the per-symbol test coverage of the
LLMsVerifier crush configuration validation surface — the gate
through which consuming projects (HelixCode, HelixAgent, etc.)
trust LLMsVerifier as the constitutional single source of truth
for provider/model metadata (CONST-036 / CONST-037 / CONST-038 /
CONST-039 / CONST-040 in consuming projects).

A row that lists captured runtime evidence is a real-coverage row.
A row whose evidence cell is empty is a documented gap — never a
silent bluff.

## Anti-bluff posture

LLMsVerifier sits at the centre of every consuming project's
"which providers / models can users actually call?" decision.
A failure mode where LLMsVerifier silently accepts unverified
provider configurations would translate, downstream, into a user-
facing "the documented model is unreachable" bug — exactly the
class of defect the 2026-04-28 operator mandate forbids.

The round-296 Challenge runner (`challenges/runner/main.go`)
exercises five real surfaces on the `crush_config` package per
locale fixture, with one mutation hook (invariant 3) paired to a
wrapper that rewrites mutation-detected runs to exit 99.

## Symbol → test coverage (crush_config package)

| Symbol | Test type | Test file / surface | Evidence captured |
|--------|-----------|---------------------|-------------------|
| `crush_config.NewSchemaValidator()` | unit | `llm-verifier/pkg/crush/config/validator_test.go::TestNewSchemaValidator` | `ok  digital.vasic.llmsverifier/pkg/crush/config  1.015s` |
| `crush_config.NewSchemaValidator()` | challenge | `challenges/runner/main.go` invariant 1 | `PASS  validator.constructor.{de,en,es,ja,sr}  nonNil=true` |
| `(*SchemaValidator).ValidateFromReader(valid)` | unit | `llm-verifier/pkg/crush/config/validator_test.go::TestValidateFromReader` | `ok` |
| `(*SchemaValidator).ValidateFromReader(valid)` | challenge | `challenges/runner/main.go` invariant 2 | `PASS  validator.valid_minimal_config.{5 locales}  Valid=true` |
| `(*SchemaValidator).ValidateFromReader(no-providers)` | challenge | `challenges/runner/main.go` invariant 3 | `PASS  validator.rejects_no_providers.{5 locales}  errors=1` |
| `(*SchemaValidator).ValidateFromReader(bad-JSON)` | challenge | `challenges/runner/main.go` invariant 4 | `PASS  validator.invalid_json_error.{5 locales}  err="invalid JSON: ..."` |
| `(*ConfigLoader).SaveToFile + LoadFromFile` | unit | `llm-verifier/pkg/crush/config/types_test.go::TestConfigLoader_SaveToFile, TestConfigLoader_LoadFromFile` | `ok` |
| `(*ConfigLoader).SaveToFile + LoadFromFile` | challenge | `challenges/runner/main.go` invariant 5 | `PASS  loader.roundtrip_preserves_provider.{5 locales}` |
| `(*SchemaValidator).validateStructure` | unit | indirect via `TestValidateFromReader` table cases | `ok` |
| `(*SchemaValidator).validateProviders` | unit | indirect via `TestValidateFromReader` provider-shape cases | `ok` |
| `(*SchemaValidator).validateModel` | unit | indirect via `TestValidateFromReader` model-shape cases | `ok` |
| `(*SchemaValidator).validateLSPs` | unit | indirect via `TestValidateFromReader` LSP-shape cases | `ok` |
| `Config / SelectedModel / ProviderConfig` JSON round-trip | unit | `types_test.go::TestConfig_JSONRoundTrip, TestModel_JSONRoundTrip` | `ok` |

## Paired-mutation invariant (CONST-050(A), §1.1)

Invariant 3 (`validator.rejects_no_providers`) is paired. The
wrapper (`challenges/llmsverifier_describe_challenge.sh mutate`)
sets `LLMSVERIFIER_MUTATE_RUNNER=1`, which inverts the polarity
in the runner — PASSing only when the validator silently accepts
a config WITHOUT providers (which it must NOT). Captured evidence:

```
$ bash challenges/llmsverifier_describe_challenge.sh normal
... Summary: PASS=25 FAIL=0 ...
=== Describe Challenge: PASSED ===
exit 0

$ bash challenges/llmsverifier_describe_challenge.sh mutate
... Summary: PASS=20 FAIL=5 ...
=== Describe Challenge: MUTATION DETECTED (runner rc=1 -> exit 99) ===
exit 99
```

Exit code 99 from the mutation wrapper is the contract that
proves the runner actually checks the invariant it claims to
check, not a metadata-only PASS.

## Test types matrix (CONST-050(B))

| Test type | Status | Path |
|-----------|--------|------|
| unit | covered | `llm-verifier/pkg/**/*_test.go` |
| integration | covered | `tests/integration/`, `llm-verifier/e2e_test.go` |
| e2e | covered | `tests/e2e/` |
| security | covered | `tests/security/` |
| performance | covered | `tests/performance/`, `llm-verifier/scoring/*_test.go` |
| challenge (round-296) | covered | `challenges/runner/main.go` + `challenges/llmsverifier_describe_challenge.sh` (paired) |
| challenge (CONST-033 host power) | covered | `challenges/scripts/no_suspend_calls_challenge.sh`, `host_no_auto_suspend_challenge.sh` |
| challenge (operations) | covered | `challenges/scripts/chaos_failure_injection_challenge.sh`, `ddos_health_flood_challenge.sh`, `scaling_horizontal_challenge.sh`, `stress_sustained_load_challenge.sh`, `ui_terminal_interaction_challenge.sh`, `ux_end_to_end_flow_challenge.sh` |
| benchmarking | covered | `llm-verifier/scoring/*_test.go`, `benchmark.sh` |

## Decoupling guarantee (CONST-051(B))

The round-296 runner imports ONLY
`digital.vasic.llmsverifier/pkg/crush/config` plus Go standard
library. No consuming-project namespace is referenced. The
fixtures use 5 locales (en/sr/de/es/ja) but the runner is
project-not-aware — any consumer can drop in additional fixtures
without changing the runner.

## Verbatim 2026-04-28 / 2026-05-19 operator mandate

> "all existing tests and Challenges do work in anti-bluff manner -
> they MUST confirm that all tested codebase really works as
> expected! We had been in position that all tests do execute with
> success and all Challenges as well, but in reality the most of
> the features does not work and can't be used! This MUST NOT be
> the case and execution of tests and Challenges MUST guarantee the
> quality, the completition and full usability by end users of the
> product!"

Operative rule: every PASS in this matrix MUST carry pasted
runtime evidence captured during the same session as the change.
