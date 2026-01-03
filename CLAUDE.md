# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

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
- **API Server** (`api/`): Gin-based REST API with JWT auth, rate limiting, Swagger docs at `/swagger/index.html`
- **Database Layer** (`database/`): SQLite with SQL Cipher encryption, connection pooling

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
