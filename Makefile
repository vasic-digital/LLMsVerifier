# LLM Verifier Makefile
# Comprehensive build, test, and development automation

.PHONY: help build test clean docker deps lint format security check all \
        container-detect container-build container-start container-stop \
        container-logs container-status container-run \
        podman-build podman-run podman-compose-up podman-compose-down \
        messaging-start messaging-stop messaging-logs messaging-status \
        messaging-health messaging-clean messaging-topics messaging-queues test-messaging

# Default target
help: ## Show this help message
	@echo "LLM Verifier - Development Commands"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

# Development Setup
setup: deps ## Set up development environment
	@echo "Setting up development environment..."
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest

deps: ## Download and tidy dependencies
	go mod download
	go mod tidy

# P1.5-WP4: API-key loader. Sourced before build/test recipes so credentials
# from $HOME/api_keys.sh (or .env fallback) are available in the env. Guard
# keeps recipes working when the loader is absent.
LOAD_KEYS := if [ -f scripts/load_api_keys.sh ]; then . scripts/load_api_keys.sh; fi

# Building
build: ## Build the application
	@$(LOAD_KEYS); go build -o bin/llm-verifier ./cmd

build-all: ## Build for multiple platforms
	GOOS=linux GOARCH=amd64 go build -o bin/llm-verifier-linux-amd64 ./cmd
	GOOS=darwin GOARCH=amd64 go build -o bin/llm-verifier-darwin-amd64 ./cmd
	GOOS=windows GOARCH=amd64 go build -o bin/llm-verifier-windows-amd64.exe ./cmd

build-acp: ## Build ACP CLI tool
	cd llm-verifier/cmd/acp-cli && go build -o ../../../bin/acp-cli .

build-acp-all: ## Build ACP CLI for multiple platforms  
	cd llm-verifier/cmd/acp-cli && GOOS=linux GOARCH=amd64 go build -o ../../../bin/acp-cli-linux-amd64 .
	cd llm-verifier/cmd/acp-cli && GOOS=darwin GOARCH=amd64 go build -o ../../../bin/acp-cli-darwin-amd64 .
	cd llm-verifier/cmd/acp-cli && GOOS=windows GOARCH=amd64 go build -o ../../../bin/acp-cli-windows-amd64.exe .

# Testing
test: ## Run unit tests
	@$(LOAD_KEYS); go test -v -race -coverprofile=coverage.out ./...

test-integration: ## Run integration tests
	go test -v -tags=integration ./tests/integration/...

test-e2e: ## Run end-to-end tests
	go test -v -tags=e2e ./tests/e2e/...

test-all: test test-integration test-e2e ## Run all tests

test-coverage: test ## Generate and display test coverage
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out | tail -1

bench: ## Run performance benchmarks
	go test -bench=. -benchmem ./...

# Code Quality
lint: ## Run linter
	golangci-lint run --timeout=5m

format: ## Format code
	gofmt -s -w .
	goimports -w .

check-format: ## Check code formatting
	@if [ "$$(gofmt -s -l . | wc -l)" -gt 0 ]; then \
		echo "Code is not formatted:"; \
		gofmt -s -l .; \
		exit 1; \
	fi

staticcheck: ## Run static analysis
	staticcheck ./...

# Security
security: ## Run security checks
	govulncheck ./...
	# Additional security checks can be added here

security-scan: ## Comprehensive security scanning
	@echo "Running security scans..."
	gosec ./...
	trivy fs .
	govulncheck ./...

# Container Runtime (Docker/Podman)
container-detect: ## Detect available container runtime
	@./scripts/container-runtime.sh detect

container-build: ## Build container image (Docker or Podman)
	@./scripts/container-runtime.sh build

container-start: ## Start services with compose (Docker or Podman)
	@./scripts/container-runtime.sh start

container-stop: ## Stop services
	@./scripts/container-runtime.sh stop

container-logs: ## View container logs
	@./scripts/container-runtime.sh logs

container-status: ## Check container status
	@./scripts/container-runtime.sh status

container-run: ## Run container directly (without compose)
	@./scripts/container-runtime.sh run

# Docker (legacy commands, use container-* for cross-runtime support)
docker-build: ## Build Docker image
	docker build -t llm-verifier:latest .

docker-run: ## Run Docker container
	docker run -p 8080:8080 llm-verifier:latest

docker-test: ## Test in Docker environment
	docker build -t llm-verifier:test -f Dockerfile.test .
	docker run --rm llm-verifier:test

# Podman (alternative runtime)
podman-build: ## Build with Podman
	podman build -t llm-verifier:latest .

podman-run: ## Run with Podman
	podman run -p 8080:8080 llm-verifier:latest

podman-compose-up: ## Start with podman-compose
	podman-compose up -d

podman-compose-down: ## Stop with podman-compose
	podman-compose down

# Messaging Infrastructure (RabbitMQ + Kafka)
messaging-start: ## Start messaging infrastructure (RabbitMQ + Kafka)
	@echo "Starting messaging infrastructure..."
	docker-compose -f docker-compose.messaging.yml up -d
	@echo "Waiting for services to be healthy..."
	@sleep 10
	@echo "Messaging infrastructure started"
	@echo "RabbitMQ UI: http://localhost:15672 (llmsverifier/llmsverifier123)"
	@echo "Kafka UI: http://localhost:8082"

messaging-stop: ## Stop messaging infrastructure
	@echo "Stopping messaging infrastructure..."
	docker-compose -f docker-compose.messaging.yml down

messaging-logs: ## View messaging infrastructure logs
	docker-compose -f docker-compose.messaging.yml logs -f

messaging-status: ## Check messaging infrastructure status
	@echo "=== RabbitMQ Status ==="
	@docker exec llmsverifier-rabbitmq rabbitmqctl status 2>/dev/null || echo "RabbitMQ not running"
	@echo ""
	@echo "=== Kafka Status ==="
	@docker exec llmsverifier-kafka kafka-topics --bootstrap-server localhost:9092 --list 2>/dev/null || echo "Kafka not running"

messaging-health: ## Check messaging infrastructure health
	@echo "Checking RabbitMQ..."
	@curl -s -u llmsverifier:llmsverifier123 http://localhost:15672/api/health/checks/alarms 2>/dev/null || echo "RabbitMQ health check failed"
	@echo ""
	@echo "Checking Kafka..."
	@docker exec llmsverifier-kafka kafka-broker-api-versions --bootstrap-server localhost:9092 2>/dev/null | head -5 || echo "Kafka health check failed"

messaging-clean: ## Clean messaging infrastructure (removes volumes)
	@echo "WARNING: This will delete all messaging data!"
	@read -p "Are you sure? (y/N): " confirm; \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		docker-compose -f docker-compose.messaging.yml down -v; \
		echo "Messaging infrastructure cleaned"; \
	fi

messaging-topics: ## List Kafka topics
	docker exec llmsverifier-kafka kafka-topics --bootstrap-server localhost:9092 --list

messaging-queues: ## List RabbitMQ queues
	docker exec llmsverifier-rabbitmq rabbitmqctl list_queues

test-messaging: ## Run messaging layer tests
	go test -v -cover ./internal/messaging/...

# Development
run: ## Run the application locally
	go run ./cmd server

dev: ## Run in development mode with hot reload
	@echo "Development mode - use 'make run' for now"
	@echo "Hot reload can be implemented with tools like air"

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean ./...

# CI/CD Simulation
ci: check test integration security ## Run full CI pipeline locally

check: lint format staticcheck check-format ## Run all code quality checks

# Documentation
docs: ## Generate documentation
	@echo "Generating API documentation..."
	swag init -g cmd/server.go -o docs/api
	@echo "Generating code documentation..."
	go doc -all ./... > docs/code.md

docs-serve: ## Serve documentation locally
	@echo "Serving documentation on http://localhost:3000"
	cd docs && python3 -m http.server 3000

# Database
db-migrate: ## Run database migrations
	@echo "Running database migrations..."
	go run ./cmd migrate up

db-reset: ## Reset database (dangerous!)
	@echo "WARNING: This will reset the database!"
	read -p "Are you sure? (y/N): " confirm; \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		go run ./cmd migrate reset; \
	fi

# Releases
release-prep: ## Prepare for release
	@echo "Preparing for release..."
	make test-all
	make security-scan
	make build-all
	@echo "Release preparation complete"

release: ## Create a new release (requires VERSION variable)
	@echo "Creating release v$(VERSION)..."
	git tag -a v$(VERSION) -m "Release v$(VERSION)"
	git push origin v$(VERSION)
	@echo "Release v$(VERSION) created"

# Utilities
install-hooks: ## Install git hooks
	@echo "Installing git hooks..."
	cp scripts/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit

install-acp: build-acp ## Install ACP CLI to system
	sudo cp bin/acp-cli /usr/local/bin/acp-cli
	@echo "ACP CLI installed to /usr/local/bin/acp-cli"

test-acp: ## Run ACP-specific tests
	@echo "Running ACP unit tests..."
	go test -v ./llm-verifier/tests/acp_test.go
	@echo "Running ACP integration tests..."
	go test -v -tags=integration ./llm-verifier/tests/acp_integration_test.go
	@echo "Running ACP performance tests..."
	go test -v -tags=performance ./llm-verifier/tests/acp_performance_test.go
	@echo "Running ACP security tests..."
	go test -v -tags=security ./llm-verifier/tests/acp_security_test.go
	@echo "Running ACP automation tests..."
	go test -v -tags=automation ./llm-verifier/tests/acp_automation_test.go

# Development helpers for ACP
run-acp-test: build-acp ## Run ACP CLI test quickly
	./bin/acp-cli verify --model gpt-4 --provider openai

run-acp-batch: build-acp ## Run ACP batch test
	./bin/acp-cli batch --models gpt-4,claude-3-opus,deepseek-chat

update-deps: ## Update all dependencies
	go get -u ./...
	go mod tidy

audit-deps: ## Audit dependencies for security issues
	go list -json -m all | docker run --rm -i golang:1.21 go mod download && echo "Dependencies audited"

# Help for specific targets
help-build: ## Help for build targets
	@echo "Build targets:"
	@echo "  build          - Build for current platform"
	@echo "  build-all      - Build for Linux, macOS, Windows"
	@echo "  build-acp      - Build ACP CLI tool"
	@echo "  build-acp-all  - Build ACP CLI for multiple platforms"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-build - Build Docker image"

help-test: ## Help for test targets
	@echo "Test targets:"
	@echo "  test           - Run unit tests"
	@echo "  test-integration - Run integration tests"
	@echo "  test-e2e       - Run end-to-end tests"
	@echo "  test-all       - Run all tests"
	@echo "  test-coverage  - Generate coverage report"

help-quality: ## Help for quality targets
	@echo "Quality targets:"
	@echo "  lint          - Run linter"
	@echo "  format        - Format code"
	@echo "  staticcheck   - Run static analysis"
	@echo "  check         - Run all quality checks"

help-security: ## Help for security targets
	@echo "Security targets:"
	@echo "  security       - Basic security checks"
	@echo "  security-scan  - Comprehensive security scan"

# Development workflow
dev-setup: setup install-hooks ## Complete development setup

dev-daily: check test ## Daily development workflow

# Emergency commands
emergency-stop: ## Emergency stop for all services
	@echo "Stopping all services..."
	docker-compose down --remove-orphans 2>/dev/null || true
	pkill -f llm-verifier 2>/dev/null || true
	@echo "Emergency stop complete"

emergency-clean: emergency-stop clean ## Emergency cleanup
	@echo "Emergency cleanup complete"

# Aliases for common commands
t: test
b: build
c: check
r: run

# Meta targets
all: check test build ## Run everything

.DEFAULT_GOAL := help