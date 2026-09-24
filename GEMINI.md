## INHERITED FROM the Helix Constitution

> Base agent rules live in the Helix Constitution. **READ THOSE FIRST.**
> The constitution's `GEMINI.md` and the universal `Constitution.md` it
> references are authoritative for any topic not covered here. Locate the
> constitution from any nested depth via its `find_constitution.sh` helper —
> do NOT hardcode a path (this module stays fully decoupled and
> project-agnostic per §11.4.28). Module-specific rules below extend the
> universal clauses; they never weaken them.

Critical universal rules every CLI agent (Claude Code, Cursor, Aider,
Codex, Gemini CLI, Qwen Code) MUST honour while working in this module:

- **No bluffing.** Every PASS carries positive captured evidence. §11.4.
- **Mutation-paired gates.** Every new gate has a paired mutation proving
  it catches regressions. §1.1.
- **No guessing language** (`likely`, `probably`, `maybe`, `seems`). §11.4.6.
- **Credentials never tracked.** `.env` patterns git-ignored; runtime-load
  only. §11.4.10.
- **Never force-push.** Force-push is forbidden; integrate by merging onto
  the latest main. §9 / §11.4.113.
- **60% RAM cap.** Heavy work wrapped in a bounded execution scope. §12.6.

Canonical reference: https://github.com/HelixDevelopment/HelixConstitution

---

# GEMINI.md — LLMsVerifier Authoritative Agent Guide

## INHERITED FROM constitution/GEMINI.md

All rules in `constitution/GEMINI.md` (and the `constitution/Constitution.md` it references) apply unconditionally. This file's rules below extend them — they MUST NOT weaken any inherited rule. Use `constitution/find_constitution.sh` from the parent project root to resolve the absolute path of the submodule from any nested location.

**Scope**: All AI agents, human contributors, and automated processes
working on the `digital.vasic.llmsverifier` module.

## Agent Identity & Purpose

You are an AI agent working on **LLMsVerifier**, an enterprise-grade
platform for verifying, monitoring, and optimizing Large Language Model
(LLM) performance across multiple providers. Consuming Helix-family
projects treat it as their single source of truth for provider/model
metadata, verification status, and scoring data — your work here directly
impacts every downstream system that depends on it.

**Your mandate**: write real, working, tested code. No simulations, no
placeholders, no "for now" implementations. Every feature you implement
MUST actually work when a user invokes it.

## Module Overview

LLMsVerifier is written in Go. The **root module** (`llmsverifier`, `go
1.25.3`) is a thin governance + dependency wrapper; the real application
lives in the `llm-verifier/` subdirectory (module `digital.vasic.llmsverifier`,
wired in via `replace digital.vasic.llmsverifier => ./llm-verifier`). A
sibling own-org dependency, `digital.vasic.challenges`, is consumed by
reference at `../challenges` relative to this repo root (`replace
digital.vasic.challenges => ../challenges`) — this submodule expects
Challenges to be checked out as a sibling directory, never nested.

### Key capabilities (per `README.md` — ground truth)

- **Mandatory model verification** — every model must pass a "Do you see
  my code?" verification check before it is considered usable.
- **12 provider adapters** — OpenAI, Anthropic, Cohere, Groq, Together AI,
  Mistral, xAI, Replicate, DeepSeek, Cerebras, Cloudflare Workers AI,
  SiliconFlow.
- **Real-time health monitoring** with intelligent failover (circuit
  breaker pattern).
- **Enterprise features** — LDAP/SSO (SAML/OIDC), SQLCipher
  database-level encryption, Splunk/DataDog/New Relic/ELK monitoring
  integrations.
- **Multi-platform clients** — CLI, TUI, Web, Desktop, Mobile.
- **`(llmsvd)` branding suffix** — every LLMsVerifier-generated
  provider/model configuration carries this mandatory suffix; only
  verified models are included in exported configurations.
- **SDKs** — Python and JavaScript, full API coverage.

### Technology stack

- **Messaging**: RabbitMQ (`amqp091-go`), Kafka (`segmentio/kafka-go`).
- **HTTP / API**: Gin, gRPC, WebSocket (`gorilla/websocket`).
- **AuthN/Z**: `golang-jwt/jwt/v5`, LDAP (`go-ldap/ldap/v3`), NTLM
  (`Azure/go-ntlmssp`).
- **Persistence**: SQLite / SQLCipher (`mattn/go-sqlite3`) for
  database-level encryption of sensitive verification data.
- **Config / CLI**: Viper, `spf13/pflag`, `spf13/cast`.
- **Logging**: `sirupsen/logrus`. **Testing**: `stretchr/testify`.

## Repository Layout

```
llms_verifier/                            # ← repo root (this submodule)
├── (agent manuals: this file + its siblings) / CONSTITUTION.md
├── README.md                             # ground-truth project description
├── go.mod                                # module llmsverifier, go 1.25.3
├── llm-verifier/                         # ← the real Go application (module digital.vasic.llmsverifier)
│   ├── cmd/                              # model-verification, crush-config-converter, tui, testsuite, …
│   ├── internal/  pkg/                   # api, cliagents, crush, helixqa, i18n, opencode, …
│   ├── config/                           # config.yaml(.example), config_full.yaml
│   ├── mobile/  desktop/                 # client surfaces
│   └── docs/                             # COMPLETE_USER_MANUAL.md, API_DOCUMENTATION.md, …
├── docs/                                 # ARCHITECTURE_OVERVIEW.md, MODEL_VERIFICATION_GUIDE.md, …
├── challenges/                           # this submodule's own Challenge bank
│   ├── runner/main.go                    # in-process Go binary exercising real crush_config surfaces
│   ├── fixtures/{de,en,es,ja,sr}.yaml    # 5-locale fixtures
│   └── llmsverifier_describe_challenge.sh
├── configs/  Assets/  sdk/  mobile/  monitoring/  helm/  k8s/  website/
└── scripts/                               # create_complete_*.py, build_and_test_fixed.sh, …
```

**Cardinal rule:** the real application code lives under `llm-verifier/`,
not the repo root's `internal`/`cmd` (the root only carries a thin
governance module + top-level docs). "Edit the API layer" means
`llm-verifier/internal/api`, not a root-level `internal/`.

## Build & Test Commands

```bash
# Docker (recommended quick start)
docker-compose up -d

# Local development — the real application lives in llm-verifier/
cd llm-verifier
go mod download
go run cmd/main.go

# Run all tests
go test ./...
./run_comprehensive_tests.sh

# Model verification
./llm-verifier/cmd/model-verification/model-verification --verify-all

# This submodule's own Challenge bank (round-296)
bash challenges/llmsverifier_describe_challenge.sh normal    # exit 0
bash challenges/llmsverifier_describe_challenge.sh mutate    # paired §1.1 mutation — must exit 99
```

**Anti-bluff smoke check** (must always pass):
```bash
grep -rn "simulated\|for now\|TODO implement\|placeholder" \
  llm-verifier/internal llm-verifier/cmd && echo "BLUFF FOUND" || echo "clean"
```

## Code Patterns for Agents

- **Interface-driven design** — define a contract (e.g. `Provider` with
  `Generate`/`GetModels`/`HealthCheck`) and implement it with REAL
  behaviour (real HTTP calls, real process execution) — never a
  simulation.
- **Manager pattern** — mutex-guarded state persisted to a REAL database
  before being cached in memory; return wrapped errors
  (`fmt.Errorf("...: %w", err)`), never swallow them.
- **Testing (unit)** — table-driven tests with `testify/mock` are
  permitted only here (§11.4.27).
- **Testing (integration — NO MOCKS)** — open a REAL SQLite/SQLCipher
  database in `t.TempDir()`, exercise the REAL service (e.g.
  `VerifyModel` against a real provider call), and verify the REAL
  persisted result on read-back.

### Common anti-patterns to avoid

- **The simulation trap** — returning a canned string instead of making
  the real generation call.
- **The hardcoded list** — returning a literal slice of models instead of
  querying the provider's real `/api/tags`-class endpoint.
- **The stub interface** — a method that returns `nil` / `// TODO:
  implement` instead of really dialing, verifying, and installing.

## Anti-Bluff Checklist for Every Task

Before marking any task complete, verify:

- [ ] **No simulation** — code doesn't contain "simulate", "for now",
      "TODO implement", "placeholder".
- [ ] **Real HTTP calls** — API clients make actual HTTP requests with
      real bodies.
- [ ] **Real database operations** — real queries, not in-memory maps
      (unless explicitly caching).
- [ ] **Real process execution** — `os/exec`, not `fmt.Printf` +
      `time.Sleep`.
- [ ] **Real file operations** — `os.ReadFile`/`os.WriteFile`, not mock
      in-memory buffers.
- [ ] **Test validates reality** — checks actual behaviour, not just call
      counts.
- [ ] **Challenge validates end-to-end** — exercises the complete user
      workflow.
- [ ] **Documentation example works** — a README example executes
      successfully when copy-pasted.
- [ ] **No bare skips** — every `t.Skip()` carries a `SKIP-OK: #<ticket>`
      marker.
- [ ] **Evidence pasted** — the commit/PR contains actual terminal output
      from a real execution.

## LLMsVerifier Constitutional Mandates

- **Single source of truth** — LLMsVerifier is the sole authoritative
  source, for every consuming project, of model metadata, provider
  metadata, verification status, and scoring data. No hardcoded model
  lists, no simulated discovery.
- **Model provider anti-bluff guarantee** — every model displayed to
  users MUST be verified within 24h; integration tests MUST use real
  provider data, not mocks.
- **Real-time model status accuracy** — model status MUST reflect
  verifier state within 60s (poll interval ≤ 60s if push unavailable).
- **All-providers integration mandate** — MUST integrate with all 12
  supported providers listed above.
- **Capability integration mandate** — MCP, LSP, ACP, Embedding, RAG,
  Skills, and Plugins capability flags MUST be sourced from this
  project's own `VerificationResult`, never hardcoded.

## Working with Submodules

This submodule depends on the Challenges submodule (`digital.vasic.challenges`,
resolved via `../challenges`) to run its own Challenge suites. When working
with submodule dependencies: (1) check governance — does the dependency
carry the full governance-carrier set; (2) add what's missing, referencing
the constitution; (3) verify this submodule actually compiles against the
sibling layout; (4) confirm this submodule's own Challenges actually run.

## Emergency Procedures

**If you discover a bluff**: stop working on dependent features, document
it under `docs/issues/BLUFFS.md`, write a Challenge that reproduces it,
fix it, verify the Challenge now passes, and update documentation to
reflect reality.

**If a test passes but the feature doesn't work**: the test is a bluff —
tighten it, add assertions on actual output quality, add anti-bluff checks
(no "simulated" in responses), run it against real infrastructure, verify
it FAILS on the broken code, then fix the code.

## Governance pointers

| Document | Authority |
|---|---|
| [`CONSTITUTION.md`](CONSTITUTION.md) | Submodule-scoped anchors inherited from the Helix Constitution |
| [`GEMINI.md`](GEMINI.md) (this file's siblings: one per supported CLI) | AI-agent operating manuals — same inheritance pointer + anti-bluff posture. A governance edit lands in every one of them or in none — cascade any rule change across all of them so an agent reading any single carrier gets the same binding rules. |
| [`helix-deps.yaml`](helix-deps.yaml) | Own-org dependency manifest |
| `docs/ARCHITECTURE_OVERVIEW.md` / `docs/MODEL_VERIFICATION_GUIDE.md` | Architecture + verification-flow detail |
| `llm-verifier/docs/COMPLETE_USER_MANUAL.md` / `llm-verifier/docs/API_DOCUMENTATION.md` | End-user + API reference |

## Resources

- `README.md` — overview, install, usage.
- `docs/` — architecture, model-verification guide, LLMSVD-suffix guide,
  configuration-migration guide, capability detection, test-suite guide.
- `challenges/` — this submodule's own Challenge bank.

---

*Remember: your code will be used by real people. Write code that
actually works.*
