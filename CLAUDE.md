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
