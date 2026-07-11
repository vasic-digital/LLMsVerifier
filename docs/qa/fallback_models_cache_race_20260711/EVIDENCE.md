# §11.4.169 coverage gap-fill — `providers` package `modelCache` concurrency/race test

**Scope**: `submodules/llms_verifier` (module `digital.vasic.llmsverifier`, `llm-verifier/`), file-scope-limited per this
session's mandate. No other repo/submodule touched.

**Date**: 2026-07-11 (session `(T1/feature/helixllm-full-extension - claude3)`).

**Task**: §11.4.169 test-type-coverage improvement for the LLMsVerifier owned submodule (§11.4.51 equal-codebase) —
survey existing coverage, fill a *genuine* gap (race-condition/concurrency test type), never manufacture busywork.

## 1. Baseline survey

`cd llm-verifier && go test ./... -count=1` (full baseline, captured in-session):

```
58 "ok" packages
3 FAIL lines, all in ONE package: digital.vasic.llmsverifier/tests
  --- FAIL: TestCommandFlagValidation (1.30s)
  --- FAIL: TestOutputFormats (1.31s)
  FAIL  digital.vasic.llmsverifier/tests  22.716s
```

`tests/automation_test.go:TestCommandFlagValidation` / `TestOutputFormats` spawn `go run ../cmd ...` subprocesses
against a live LLM-Verifier API server; they fail in this sandbox because no compatible server is reachable
(pre-existing environment gate, `serverUnavailable(...)` skip-path exists in the same file but these two cases hit an
unexpected-error branch rather than the skip branch). This is **pre-existing, unrelated to this session's change**,
outside the assigned scope (race/concurrency coverage in `providers`/`capabilities`), and untouched by this commit.

`go vet ./...` — clean, zero findings.

### Existing race/concurrency coverage (already present, confirmed via grep before starting)

24 files across the tree already contain `t.Parallel()` / `TestXxxRace` / `TestXxxConcurrent` patterns, including
`providers/cache_expiry_race_red_test.go` — a RED-test (git-stash-verified per its own commit message) covering a
**different** shared cache: `ModelProviderService.cache` (a read-lock-protected map **write** during expired-entry
eviction — the classic "delete under RLock" bug, already fixed).

### Candidates surveyed and ruled out

- `capabilities/detector.go` (`Detector.cache`/`cacheExpiry` maps, `sync.RWMutex`): 13 existing detector-focused unit
  tests (`capabilities/capabilities_test.go`), **zero** concurrent-access test. Inspected the lock discipline in
  detail: `DetectProviderCapabilities` correctly `RLock`s the cache-hit check, `RUnlock`s before any network call, and
  only mutates `d.cache`/`d.cacheExpiry` under `d.mu.Lock()`. `GetProviderBaseCapabilities` returns a **shallow copy**
  of the seed (`seedCopy := *caps; return &seedCopy`), which isolates every value field including `Verified`; the one
  in-place `append` onto a copied slice (`caps.Streaming.Types = append(...)` in `detectStreaming`) only reuses the
  seed's backing array if the seed's composite-literal slice has spare capacity, which the registry's literals do not
  (`cap == len` for every `[]StreamingType{...}` seed literal surveyed) — so no live aliasing race exists today.
  **Verdict: lock-correct, not a genuine gap** (documented here per the honesty requirement — no test added for this
  candidate to avoid manufacturing busywork around a already-safe path).
- `providers/fallback_models.go` (`modelCache` package-level singleton, `sync.RWMutex`): `GetCachedModels`/
  `SetCachedModels`/`GetFallbackModels` are all exercised only by `TestModelCache_SetAndGet` /
  `TestModelCache_NotFound` — both single-threaded. This IS a genuine gap: `modelCache` is a **process-wide shared
  singleton** consulted by every provider's model-discovery fallback path, and the sibling
  `ModelProviderService.GetAllModels` (same package) fans discovery out across one goroutine per configured provider
  (`var wg sync.WaitGroup` + per-provider `go func(...)`) — exactly the concurrent-access shape this cache is
  production-exposed to, with zero `-race` coverage before this change.

**Chosen gap**: `providers/fallback_models.go`'s `modelCache` singleton — thinnest genuine gap, real production
concurrent-access shape, zero existing coverage of that shape.

## 2. Test added

New file: `llm-verifier/providers/fallback_models_cache_concurrent_race_test.go`
Function: `TestModelCache_ConcurrentGetSet_NoRace`

- 12 synthetic provider IDs, 6 concurrent writers (`SetCachedModels`) + 6 concurrent readers (`GetCachedModels`) per
  ID, plus concurrent `GetFallbackModels` calls on both a cached and an always-static-fallback provider ID (`openai`)
  — exercising the real production call path, not just the two cache primitives in isolation.
- Non-bluff sanity assertion after the goroutine storm: `race-provider-0` is genuinely retrievable with the expected
  2 models — a cache that silently dropped writes under contention would pass a bare `-race` scan but fail this
  functional check.

Test type: **race-condition / concurrency-atomicity** (§11.4.169 item 10/11), unit-layer (package `providers`, no
mocks used — the code under test is the real `modelCache` singleton, not a fake).

## 3. Real captured `go test -race` output

### 3a. Baseline (current, correct source) — first run

```
$ go test -race -count=1 -v -run TestModelCache_ConcurrentGetSet_NoRace ./providers/...
=== RUN   TestModelCache_ConcurrentGetSet_NoRace
--- PASS: TestModelCache_ConcurrentGetSet_NoRace (0.00s)
PASS
ok  	digital.vasic.llmsverifier/providers	1.035s
```

### 3b. Determinism (§11.4.50) — `-count=3` consecutive

```
$ go test -race -count=3 -v -run TestModelCache_ConcurrentGetSet_NoRace ./providers/...
=== RUN   TestModelCache_ConcurrentGetSet_NoRace
--- PASS: TestModelCache_ConcurrentGetSet_NoRace (0.00s)
=== RUN   TestModelCache_ConcurrentGetSet_NoRace
--- PASS: TestModelCache_ConcurrentGetSet_NoRace (0.00s)
=== RUN   TestModelCache_ConcurrentGetSet_NoRace
--- PASS: TestModelCache_ConcurrentGetSet_NoRace (0.00s)
PASS
ok  	digital.vasic.llmsverifier/providers	1.026s
```

### 3c. Full `providers` package regression under `-race` (no collateral breakage from the new file)

```
$ go test -race -count=1 ./providers/...
ok  	digital.vasic.llmsverifier/providers	51.731s
```

## 4. §1.1 paired mutation (load-bearing self-validation)

Mutation applied **in this session, then reverted before commit** (verified via `git diff` = empty after revert,
pasted below): `SetCachedModels` in `providers/fallback_models.go` had its `modelCache.mu.Lock()` /
`defer modelCache.mu.Unlock()` pair removed (replaced with a `// MUTATED for paired §1.1 self-validation` comment),
leaving the two map writes (`modelCache.models[...]`, `modelCache.timestamps[...]`) unguarded while
`GetCachedModels` still `RLock`s its own reads of the same maps.

Re-ran the identical test against the mutated source:

```
$ go test -race -count=1 -run TestModelCache_ConcurrentGetSet_NoRace ./providers/...
==================
WARNING: DATA RACE
Write at 0x00c0001fd8c0 by goroutine 10:
  runtime.mapassign_faststr()
  digital.vasic.llmsverifier/providers.SetCachedModels()
      .../providers/fallback_models.go:45 +0xf4
  ...
Previous write at 0x00c0001fd8c0 by goroutine 12:
  ...
==================
[... multiple further WARNING: DATA RACE blocks, then ...]
fatal error: concurrent map read and map write

goroutine 104 [running]:
digital.vasic.llmsverifier/providers.GetCachedModels(...)
      .../providers/fallback_models.go:32 +0x133
digital.vasic.llmsverifier/providers.TestModelCache_ConcurrentGetSet_NoRace.func2(...)
      .../providers/fallback_models_cache_concurrent_race_test.go:74 +0x85
...
FAIL	digital.vasic.llmsverifier/providers	0.035s
FAIL
$ echo "EXIT_CODE=$?"
EXIT_CODE=1
```

The mutation is **load-bearing**: the guard does not just fail under `-race`, it triggers a genuine Go runtime crash
(`fatal error: concurrent map read and map write`) — proof the test exercises a real, previously-uncovered hazard
class, not a vacuous assertion.

### Revert verification (before commit)

```
$ git diff --stat -- llm-verifier/providers/fallback_models.go
(empty)
$ git diff -- llm-verifier/providers/fallback_models.go
(empty)
$ grep -n "MUTATED for paired\|// always pass\|_mutated_" \
    llm-verifier/providers/fallback_models.go \
    llm-verifier/providers/fallback_models_cache_concurrent_race_test.go
clean
```

Post-revert, the guard was re-run twice more (§3a/§3b above are the POST-revert runs) confirming PASS×3 on the
restored, byte-identical original source. §11.4.84 working-tree quiescence confirmed: `git status --short` in the
submodule shows only the new test file as untracked; no residue.

## 5. Honest scope note

This closes ONE genuine thin spot. It does not claim the `providers`/`capabilities` packages are now exhaustively
race-covered — `client/rate_limiter.go`, `client/client_manager.go`, `providers/model_verification_service.go`, and
several `monitoring/*` files also carry bare mutex+map combinations with only single-threaded test coverage today;
those are candidates for a follow-up pass, not claimed as closed by this change.
