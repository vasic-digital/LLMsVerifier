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

### Docker

```bash
make docker-build       # Build Docker image
make docker-run         # Run Docker container on port 8080
docker-compose up -d    # Start with Docker Compose
```

## Architecture

### Directory Structure

```
LLMsVerifier/
├── llm-verifier/           # Core Go application
│   ├── cmd/                # CLI entry points and commands
│   ├── api/                # REST API handlers, middleware, validation
│   ├── providers/          # LLM provider adapters (OpenAI, Anthropic, Google, etc.)
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

### Key Architectural Patterns

- **Provider Adapter Pattern**: Each LLM provider (OpenAI, Anthropic, Google, etc.) has an independent adapter in `providers/`
- **Circuit Breaker**: Automatic failover for provider outages in `failover/`
- **Supervisor/Worker**: Distributed task processing in `enhanced/`
- **Repository Pattern**: Clean data access layer in `database/`
- **Event-Driven**: Pub/sub architecture for async processing in `events/`

### Core Components

- **Verification Engine** (`verification/`): Runs 20+ capability tests including "Do you see my code?" verification
- **Provider Service** (`providers/`): Manages 17+ LLM provider integrations with dynamic model discovery
- **API Server** (`api/`): Gin-based REST API with JWT auth, rate limiting, Swagger docs at `/swagger/index.html`
- **Database Layer** (`database/`): SQLite with SQL Cipher encryption, connection pooling

### (llmsvd) Suffix System

All LLMsVerifier-generated providers and models include mandatory `(llmsvd)` branding suffix for verified models. This is a core feature - verified models must pass the "Do you see my code?" test.

## Key Dependencies

- `gin-gonic/gin` - Web framework
- `spf13/cobra` - CLI framework
- `spf13/viper` - Configuration
- `charmbracelet/bubbletea` - TUI framework
- `mattn/go-sqlite3` - SQLite driver
- `golang-jwt/jwt/v5` - JWT authentication
- `stretchr/testify` - Testing utilities

## Configuration

- Main config: `config.yaml` (copy from `config.yaml.example`)
- Environment variables supported via `${VAR_NAME}` substitution
- Database encryption key required for SQL Cipher
- API keys stored in `.env` (gitignored)

## CI/CD

GitHub Actions workflows in `.github/workflows/`:
- `ci.yml` - Main CI pipeline with tests, lint, security scans
- `deploy.yml` - Deployment pipeline
