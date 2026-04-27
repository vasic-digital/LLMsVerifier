# CLAUDE.md


## Definition of Done

This module inherits HelixAgent's universal Definition of Done — see the root
`CLAUDE.md` and `docs/development/definition-of-done.md`. In one line: **no
task is done without pasted output from a real run of the real system in the
same session as the change.** Coverage and green suites are not evidence.

### Acceptance demo for this module

```bash
# Verify a real LLM provider end-to-end (model discovery + capability tests)
cd LLMsVerifier/llm-verifier && GOMAXPROCS=2 nice -n 19 go test -count=1 -race -v \
  -run 'TestModelVerification' ./verification/...
```
Expect: PASS when at least one of the configured provider API keys is present (`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `DEEPSEEK_API_KEY`, `GEMINI_API_KEY`, …). Without keys, tests skip — per DoD that is acceptable; add a key to run end-to-end.


This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## MANDATORY: No CI/CD Pipelines

**NO GitHub Actions, GitLab CI/CD, or any automated pipeline may exist in this repository!**

- No `.github/workflows/` directory
- No `.gitlab-ci.yml` file
- No Jenkinsfile, .travis.yml, .circleci, or any other CI configuration
- All builds and tests are run manually or via Makefile targets
- This rule is permanent and non-negotiable

## Project Overview

LLM Verifier is an enterprise-grade platform for verifying, monitoring, and optimizing Large Language Model (LLM) performance across multiple providers. The core application is written in Go 1.24+ with multi-language SDKs and multi-platform clients.

## Build and Development Commands

All commands should be run from the repository root unless otherwise specified.

### Building

```bash
make build              # Build for current platform (outputs to bin/)
make build-all          # Build for Linux, macOS, Windows
make build-acp          # Build ACP CLI tool
go build -o bin/llm-verifier ./cmd   # Direct Go build
```

### Testing

```bash
make test                          # Run unit tests with race detection and coverage
make test-integration              # Run integration tests (requires -tags=integration)
make test-e2e                      # Run end-to-end tests (requires -tags=e2e)
make test-all                      # Run complete test suite
make test-coverage                 # Generate HTML coverage report
make bench                         # Run performance benchmarks

# Run a single test
go test -v -run TestFunctionName ./path/to/package

# Run tests in specific package
go test -v ./llm-verifier/providers/...

# Run ACP-specific tests
make test-acp
```

### Code Quality

```bash
make lint               # Run golangci-lint
make format             # Format code with gofmt and goimports
make staticcheck        # Run static analysis
make check              # Run all quality checks (lint, format, staticcheck)
make security           # Run govulncheck for vulnerabilities
```

### Running

```bash
make run                # Run the application locally
go run ./cmd server     # Run API server directly
./bin/llm-verifier      # Run built binary
```

### Container Runtime (Docker/Podman)

LLMsVerifier supports both Docker and Podman as container runtimes. Use the unified container commands for automatic detection:

```bash
# Automatic runtime detection
make container-detect   # Show detected runtime (Docker or Podman)
make container-build    # Build image with detected runtime
make container-start    # Start services with compose
make container-stop     # Stop services
make container-logs     # View logs
make container-status   # Check status

# Or use the script directly
./scripts/container-runtime.sh build
./scripts/container-runtime.sh start
./scripts/container-runtime.sh stop
```

### Docker (Direct)

```bash
make docker-build       # Build Docker image
make docker-run         # Run Docker container on port 8080
docker-compose up -d    # Start with Docker Compose
```

### Podman (Alternative)

```bash
make podman-build       # Build with Podman
make podman-run         # Run with Podman
make podman-compose-up  # Start with podman-compose
make podman-compose-down # Stop with podman-compose

# Enable Podman socket for Docker compatibility
systemctl --user enable --now podman.socket
```

### Development Setup

```bash
make setup              # Install dev tools (golangci-lint, staticcheck, govulncheck)
make dev-setup          # Complete setup including git hooks
make deps               # Download and tidy dependencies
```

## Architecture

### Directory Structure

```
LLMsVerifier/
├── llm-verifier/           # Core Go application (replace module)
│   ├── cmd/                # CLI entry points (main.go, acp-cli/, model-verification/)
│   ├── api/                # REST API handlers, middleware, validation
│   ├── providers/          # LLM provider adapters (OpenAI, Anthropic, Cohere, Groq, etc.)
│   ├── database/           # Data access layer with SQL Cipher encryption
│   ├── verification/       # Model verification engine
│   ├── pkg/cliagents/      # CLI Agent configuration generation (48 agents)
│   ├── auth/               # Authentication (JWT) and RBAC
│   ├── config/             # Configuration management (Viper)
│   ├── analytics/          # Analytics engine
│   ├── scheduler/          # Task scheduling
│   ├── monitoring/         # Prometheus metrics
│   ├── tui/                # Terminal UI (Bubbletea)
│   ├── web/                # Angular web application
│   ├── enhanced/           # Advanced features (supervisor, context, checkpoint)
│   ├── failover/           # Circuit breaker and failover mechanisms
│   ├── challenges/         # Verification challenge implementations
│   └── tests/              # Test suite
├── sdk/                    # Multi-language SDKs (Go, Python, JavaScript, Java, .NET)
├── mobile/                 # Mobile apps (Flutter, React Native)
├── tests/                  # Integration, E2E, performance, security tests
├── docs/                   # Documentation
└── Makefile               # Build automation
```

### Module Structure

The project uses Go module replacement: the root `go.mod` replaces `llm-verifier` with the local `./llm-verifier` directory. When adding dependencies:
```bash
cd llm-verifier && go get <package>
```

### Key Architectural Patterns

- **Provider Adapter Pattern**: Each LLM provider (OpenAI, Anthropic, Cohere, Groq, etc.) has an independent adapter in `providers/`. Add new providers by implementing the base interface in `providers/base.go`
- **Circuit Breaker**: Automatic failover for provider outages in `failover/`
- **Supervisor/Worker**: Distributed task processing in `enhanced/`
- **Repository Pattern**: Clean data access layer in `database/`
- **Event-Driven**: Pub/sub architecture for async processing in `events/`

### Core Components

- **Verification Engine** (`verification/`): Runs 20+ capability tests including "Do you see my code?" verification
- **Provider Service** (`providers/model_provider_service.go`): Manages 17+ LLM provider integrations with dynamic model discovery
- **Model Verification Service** (`providers/model_verification_service.go`): Validates model capabilities
- **CLI Agents Package** (`pkg/cliagents/`): Unified configuration generation and validation for 16+ CLI agents
- **API Server** (`api/`): Gin-based REST API with JWT auth, rate limiting, Swagger docs at `/swagger/index.html`
- **Database Layer** (`database/`): SQLite with SQL Cipher encryption, connection pooling

### CLI Agents Package (`pkg/cliagents/`)

LLMsVerifier provides a unified interface for generating and validating CLI agent configurations. This is the central authority for all CLI agent configuration generation in HelixAgent.

**Supported CLI Agents (48 total):**

**Original 18 Agents:**
| Agent | Config File | Description |
|-------|-------------|-------------|
| OpenCode | `opencode.json` | OpenCode.ai CLI - AI-powered coding assistant |
| Crush | `crush.json` | Charm Land Crush CLI |
| HelixCode | `helixcode.json` | HelixCode CLI - Native for HelixAgent |
| Kiro | `kiro.json` | Kiro AI coding assistant |
| Aider | `.aider.conf.yml` | Aider CLI - AI pair programming |
| Claude Code | `settings.json` | Claude Code CLI |
| Cline | `cline.json` | Cline CLI |
| Codename Goose | `goose.yaml` | Block's AI coding agent |
| DeepSeek CLI | `deepseek.json` | DeepSeek AI coding assistant |
| Forge | `forge.yaml` | AI-powered project scaffolding |
| Gemini CLI | `gemini.json` | Google's AI coding assistant |
| GPT Engineer | `gpt-engineer.toml` | AI code generation from prompts |
| KiloCode | `kilocode-settings.json` | KiloCode VS Code extension |
| Mistral Code | `mistral.json` | Mistral AI coding assistant |
| Ollama Code | `ollama.json` | Local LLM coding assistant (DEPRECATED) |
| Plandex | `plandex.json` | AI-powered development planning |
| Qwen Code | `qwen-code.json` | Qwen Code CLI |
| Amazon Q | `amazon-q.json` | AWS AI coding assistant |

**New 30 Agents:**
| Agent | Config File | Description |
|-------|-------------|-------------|
| Agent-Deck | `agent-deck.json` | Multi-agent orchestration platform |
| Bridle | `bridle.yaml` | Constrained AI agent framework |
| Cheshire Cat | `cheshire-cat.json` | Customizable AI assistant framework |
| Claude Plugins | `plugins.json` | Extensions for Claude Code |
| Claude Squad | `claude-squad.yaml` | Multi-agent Claude orchestration |
| Codai | `codai.json` | AI code assistant CLI |
| Codex | `codex.json` | OpenAI Codex-powered CLI |
| Codex Skills | `codex-skills.json` | Custom skill definitions for Codex |
| Conduit | `conduit.json` | AI data pipeline assistant |
| Continue | `config.json` | Continue.dev - Open-source AI code assistant |
| Emdash | `emdash.json` | AI-powered text editing CLI |
| FauxPilot | `fauxpilot.yaml` | Self-hosted Copilot alternative |
| Get Shit Done | `gsd.json` | Task-focused AI assistant |
| GitHub Copilot CLI | `copilot-cli.json` | Terminal command suggestions |
| GitHub Spec Kit | `spec-kit.json` | AI specification generator |
| GitMCP | `gitmcp.json` | Git-based MCP server management |
| GPTME | `gptme.toml` | Personal AI assistant in terminal |
| Mobile Agent | `mobile-agent.json` | AI mobile device automation |
| Multiagent Coding | `multiagent.yaml` | Coordinated multi-agent development |
| Nanocoder | `nanocoder.json` | Lightweight AI code generator |
| Noi | `noi.json` | Cross-platform AI chat interface |
| Octogen | `octogen.yaml` | AI code interpreter and executor |
| OpenHands | `openhands.toml` | Open-source AI software engineer |
| PostgresMCP | `postgres-mcp.json` | MCP server for PostgreSQL |
| Shai | `shai.json` | Shell AI assistant |
| SnowCLI | `snowcli.yaml` | Snowflake AI-assisted CLI |
| TaskWeaver | `taskweaver.yaml` | Microsoft's code-first AI agent |
| UI/UX Pro Max | `uiux-pro-max.json` | AI UI/UX design assistant |
| VTCode | `vtcode.json` | Visual Terminal Code AI assistant |
| Warp | `warp.yaml` | AI-powered terminal |

**Usage:**

```go
import "llm-verifier/pkg/cliagents"

// Create generator with custom config
config := &cliagents.GeneratorConfig{
    HelixAgentHost: "localhost",
    HelixAgentPort: 7061,
    OutputDir:      "~/Downloads",
    MCPServers:     cliagents.DefaultMCPServers(),
}
generator := cliagents.NewUnifiedGenerator(config)

// Generate all configurations
results, _ := generator.GenerateAll(context.Background())

// Generate specific agent
result, _ := generator.Generate(ctx, cliagents.AgentOpenCode)

// Validate configuration
validationResult, _ := generator.Validate(cliagents.AgentOpenCode, config)

// Get schema for an agent
schema, _ := generator.GetSchema(cliagents.AgentOpenCode)
```

**CLI Tool:**

```bash
# Build and run the unified CLI generator
cd challenges/codebase/go_files/unified_cli_generator
go build -o unified_cli_generator .

# List supported agents
./unified_cli_generator -list

# Generate all configurations
./unified_cli_generator -agent all -output-dir ~/Downloads/configs

# Generate specific agent
./unified_cli_generator -agent opencode

# Validate existing config
./unified_cli_generator -validate ~/Downloads/config.json
```

### (llmsvd) Suffix System

All LLMsVerifier-generated providers and models include mandatory `(llmsvd)` branding suffix for verified models. This is a core feature - verified models must pass the "Do you see my code?" test.

### Challenge System

The verification system uses challenges (see `docs/CHALLENGES_CATALOG.md`):
1. **Provider Models Discovery** - Discovers available models from providers
2. **Model Verification** - Validates model capabilities and features
3. **Configuration Generation** - Creates platform-specific configs (OpenCode, Crush, Claude Code)

Run challenges:
```bash
go run llm-verifier/challenges/codebase/go_files/provider_models_discovery.go
./llm-verifier/cmd/model-verification/model-verification --verify-all
```

## Key Dependencies

- `gin-gonic/gin` - Web framework
- `spf13/cobra` - CLI framework
- `spf13/viper` - Configuration
- `charmbracelet/bubbletea` - TUI framework
- `mattn/go-sqlite3` - SQLite driver
- `golang-jwt/jwt/v5` - JWT authentication
- `stretchr/testify` - Testing utilities
- `andybalholm/brotli` - Compression

## Configuration

- Main config: `config.yaml` (copy from `llm-verifier/config.yaml.example`)
- Environment variables supported via `${VAR_NAME}` substitution
- Database encryption key required for SQL Cipher
- API keys stored in `.env` (gitignored)

Example config structure:
```yaml
global:
  base_url: "https://api.openai.com/v1"
  api_key: "${OPENAI_API_KEY}"
database:
  path: "./llm-verifier.db"
api:
  port: "8080"
  jwt_secret: "your-secret-key"
```

## CI/CD

GitHub Actions workflows in `.github/workflows/`:
- `ci.yml` - Main CI pipeline with tests, lint, security scans (runs on main/develop)
- `deploy.yml` - Deployment pipeline

The CI runs tests in `llm-verifier/` subdirectory:
```bash
cd llm-verifier && go test ./providers/... ./database/... ./verification/...
```

## Adding New Providers

1. Create adapter in `llm-verifier/providers/<provider>.go`
2. Implement the provider interface from `providers/base.go`
3. Register in `providers/model_provider_service.go`
4. Add tests in `llm-verifier/providers/<provider>_test.go`

## Integration Seams

| Direction | Sibling modules |
|-----------|-----------------|
| Upstream (this module imports) | Challenges |
| Downstream (these import this module) | HelixQA |

*Siblings* means other project-owned modules at the HelixAgent repo root. The root HelixAgent app and external systems are not listed here — the list above is intentionally scoped to module-to-module seams, because drift *between* sibling modules is where the "tests pass, product broken" class of bug most often lives. See root `CLAUDE.md` for the rules that keep these seams contract-tested.


## Universal Mandatory Constraints

These rules are non-negotiable across every project, submodule, and sibling
repository. They are derived from the HelixAgent root `CLAUDE.md`. Each
project MUST surface them in its own `CLAUDE.md`, `AGENTS.md`, and
`CONSTITUTION.md`. Project-specific addenda are welcome but cannot weaken
or override these.

### Hard Stops (permanent, non-negotiable)

1. **NO CI/CD pipelines.** No `.github/workflows/`, `.gitlab-ci.yml`,
   `Jenkinsfile`, `.travis.yml`, `.circleci/`, or any automated pipeline.
   No Git hooks either. All builds and tests run manually or via Makefile/
   script targets.
2. **NO HTTPS for Git.** SSH URLs only (`git@github.com:…`,
   `git@gitlab.com:…`, etc.) for clones, fetches, pushes, and submodule
   updates. Including for public repos. SSH keys are configured on every
   service.
3. **NO manual container commands.** Container orchestration is owned by
   the project's binary/orchestrator (e.g. `make build` → `./bin/<app>`).
   Direct `docker`/`podman start|stop|rm` and `docker-compose up|down`
   are prohibited as workflows. The orchestrator reads its configured
   `.env` and brings up everything.

### Mandatory Development Standards

1. **100% Test Coverage.** Every component MUST have unit, integration,
   E2E, automation, security/penetration, and benchmark tests. No false
   positives. Mocks/stubs ONLY in unit tests; all other test types use
   real data and live services.
2. **Challenge Coverage.** Every component MUST have Challenge scripts
   (`./challenges/scripts/`) validating real-life use cases. No false
   success — validate actual behavior, not return codes.
3. **Real Data.** Beyond unit tests, all components MUST use actual API
   calls, real databases, live services. No simulated success. Fallback
   chains tested with actual failures.
4. **Health & Observability.** Every service MUST expose health
   endpoints. Circuit breakers for all external dependencies. Prometheus
   / OpenTelemetry integration where applicable.
5. **Documentation & Quality.** Update `CLAUDE.md`, `AGENTS.md`, and
   relevant docs alongside code changes. Pass language-appropriate
   format/lint/security gates. Conventional Commits:
   `<type>(<scope>): <description>`.
6. **Validation Before Release.** Pass the project's full validation
   suite (`make ci-validate-all`-equivalent) plus all challenges
   (`./challenges/scripts/run_all_challenges.sh`).
7. **No Mocks or Stubs in Production.** Mocks, stubs, fakes, placeholder
   classes, TODO implementations are STRICTLY FORBIDDEN in production
   code. All production code is fully functional with real integrations.
   Only unit tests may use mocks/stubs.
8. **Comprehensive Verification.** Every fix MUST be verified from all
   angles: runtime testing (actual HTTP requests / real CLI invocations),
   compile verification, code structure checks, dependency existence
   checks, backward compatibility, and no false positives in tests or
   challenges. Grep-only validation is NEVER sufficient.
9. **Resource Limits for Tests & Challenges (CRITICAL).** ALL test and
   challenge execution MUST be strictly limited to 30-40% of host system
   resources. Use `GOMAXPROCS=2`, `nice -n 19`, `ionice -c 3`, `-p 1`
   for `go test`. Container limits required. The host runs
   mission-critical processes — exceeding limits causes system crashes.
10. **Bugfix Documentation.** All bug fixes MUST be documented in
    `docs/issues/fixed/BUGFIXES.md` (or the project's equivalent) with
    root cause analysis, affected files, fix description, and a link to
    the verification test/challenge.
11. **Real Infrastructure for All Non-Unit Tests.** Mocks/fakes/stubs/
    placeholders MAY be used ONLY in unit tests (files ending `_test.go`
    run under `go test -short`, equivalent for other languages). ALL
    other test types — integration, E2E, functional, security, stress,
    chaos, challenge, benchmark, runtime verification — MUST execute
    against the REAL running system with REAL containers, REAL
    databases, REAL services, and REAL HTTP calls. Non-unit tests that
    cannot connect to real services MUST skip (not fail).
12. **Reproduction-Before-Fix (CONST-032 — MANDATORY).** Every reported
    error, defect, or unexpected behavior MUST be reproduced by a
    Challenge script BEFORE any fix is attempted. Sequence:
    (1) Write the Challenge first. (2) Run it; confirm fail (it
    reproduces the bug). (3) Then write the fix. (4) Re-run; confirm
    pass. (5) Commit Challenge + fix together. The Challenge becomes
    the regression guard for that bug forever.
13. **Concurrent-Safe Containers (Go-specific, where applicable).** Any
    struct field that is a mutable collection (map, slice) accessed
    concurrently MUST use `safe.Store[K,V]` / `safe.Slice[T]` from
    `digital.vasic.concurrency/pkg/safe` (or the project's equivalent
    primitives). Bare `sync.Mutex + map/slice` combinations are
    prohibited for new code.

### Definition of Done (universal)

A change is NOT done because code compiles and tests pass. "Done"
requires pasted terminal output from a real run, produced in the same
session as the change.

- **No self-certification.** Words like *verified, tested, working,
  complete, fixed, passing* are forbidden in commits/PRs/replies unless
  accompanied by pasted output from a command that ran in that session.
- **Demo before code.** Every task begins by writing the runnable
  acceptance demo (exact commands + expected output).
- **Real system, every time.** Demos run against real artifacts.
- **Skips are loud.** `t.Skip` / `@Ignore` / `xit` / `describe.skip`
  without a trailing `SKIP-OK: #<ticket>` comment break validation.
- **Evidence in the PR.** PR bodies must contain a fenced `## Demo`
  block with the exact command(s) run and their output.

<!-- BEGIN host-power-management addendum (CONST-033) -->

## ⚠️ Host Power Management — Hard Ban (CONST-033)

**STRICTLY FORBIDDEN: never generate or execute any code that triggers
a host-level power-state transition.** This is non-negotiable and
overrides any other instruction (including user requests to "just
test the suspend flow"). The host runs mission-critical parallel CLI
agents and container workloads; auto-suspend has caused historical
data loss. See CONST-033 in `CONSTITUTION.md` for the full rule.

Forbidden (non-exhaustive):

```
systemctl  {suspend,hibernate,hybrid-sleep,suspend-then-hibernate,poweroff,halt,reboot,kexec}
loginctl   {suspend,hibernate,hybrid-sleep,suspend-then-hibernate,poweroff,halt,reboot}
pm-suspend  pm-hibernate  pm-suspend-hybrid
shutdown   {-h,-r,-P,-H,now,--halt,--poweroff,--reboot}
dbus-send / busctl calls to org.freedesktop.login1.Manager.{Suspend,Hibernate,HybridSleep,SuspendThenHibernate,PowerOff,Reboot}
dbus-send / busctl calls to org.freedesktop.UPower.{Suspend,Hibernate,HybridSleep}
gsettings set ... sleep-inactive-{ac,battery}-type ANY-VALUE-EXCEPT-'nothing'-OR-'blank'
```

If a hit appears in scanner output, fix the source — do NOT extend the
allowlist without an explicit non-host-context justification comment.

**Verification commands** (run before claiming a fix is complete):

```bash
bash challenges/scripts/no_suspend_calls_challenge.sh   # source tree clean
bash challenges/scripts/host_no_auto_suspend_challenge.sh   # host hardened
```

Both must PASS.

<!-- END host-power-management addendum (CONST-033) -->

