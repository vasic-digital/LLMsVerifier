# LLM Verifier Codebase Documentation

This document provides comprehensive documentation for the LLM Verifier codebase to assist future agents and developers in understanding the project structure, components, and key patterns.

## Table of Contents

- [Project Overview](#project-overview)
- [Directory Structure](#directory-structure)
- [Core Components](#core-components)
- [Configuration System](#configuration-system)
- [Database Layer](#database-layer)
- [LLM Integration](#llm-integration)
- [Scoring System](#scoring-system)
- [API Documentation](#api-documentation)
- [Testing Framework](#testing-framework)
- [Client Architecture](#client-architecture)
- [Key Commands](#key-commands)
- [Development Gotchas](#development-gotchas)

## Project Overview

The LLM Verifier is a Go-based application designed to verify, test, and benchmark Large Language Models (LLMs) based on their coding capabilities and other features. The application provides a comprehensive suite of tools to evaluate LLMs across multiple dimensions including code generation, tool use, embeddings, and more.

Key characteristics:
- Built with Go 1.25.2
- Uses Cobra for CLI interface
- Implements Gin framework for REST API
- Utilizes SQLite with SQL Cipher encryption for data storage
- Supports multiple client interfaces (TUI, Web, Desktop, Mobile)

## Directory Structure

```
llm-verifier/
├── cmd/
│   └── main.go - Application entry point with Cobra CLI setup
├── llmverifier/
│   ├── verifier.go - Core verification logic and model discovery
│   └── llm_client.go - LLM API client with OpenAI compatibility
├── database/
│   ├── database.go - Database initialization and schema definition
│   └── crud.go - CRUD operations for providers, models, and results
├── tests/
│   ├── test_helpers.go - Test utilities and mock server implementation
│   ├── unit_test.go - Unit tests
│   ├── integration_test.go - Integration tests
│   ├── e2e_test.go - End-to-end tests
│   ├── performance_test.go - Performance and benchmark tests
│   ├── security_test.go - Security tests
│   ├── automation_test.go - Automation tests
│   └── TEST_IMPLEMENTATION_GUIDE.md - Test coverage requirements
├── docs/
│   ├── API_DOCUMENTATION.md - REST API documentation
│   ├── SPECIFICATION.md - Feature requirements
│   └── IMPLEMENTATION_ROADMAP.md - 20-week implementation plan
├── go.mod - Go module file with dependencies
├── go.sum - Go checksum file
├── config.yaml - Configuration file with environment variable support
└── test_runner.sh - Script to run all test types
```

## Core Components

### Main Application (cmd/main.go)

The entry point of the application is `cmd/main.go` which sets up the Cobra CLI framework. The main function initializes the configuration, database, and starts the application.

Key functions:
- `main()` - Entry point that orchestrates application startup
- `LoadConfig()` - Loads configuration from config.yaml with environment variable substitution

### Verification Logic (llmverifier/verifier.go)

The core verification logic is implemented in `verifier.go`. This component is responsible for:
- Discovering available LLM models from configured endpoints
- Testing model capabilities through targeted API calls
- Calculating scores based on multiple dimensions
- Generating verification reports in Markdown and JSON formats

Key methods:
- `Verify()` - Main verification method that orchestrates the verification process
- `detectFeatures()` - Detects supported features by sending test requests
- `calculateScore()` - Calculates weighted score based on multiple dimensions

### LLM Client (llmverifier/llm_client.go)

The LLM client provides a unified interface to interact with different LLM providers. It implements OpenAI API compatibility and handles:
- Authentication with API keys
- Request/response formatting
- Error handling and retry logic
- Streaming responses when supported

### Database Layer (database/)

The database layer consists of two main files:

**database.go**: Handles database initialization and schema definition. Uses SQLite with SQL Cipher encryption for security.

**crud.go**: Implements comprehensive CRUD operations for:
- Providers (CreateProvider, GetProvider, UpdateProvider, DeleteProvider)
- Models (CreateModel, GetModel, UpdateModel, DeleteModel)
- Verification results (CreateVerificationResult, GetVerificationResults, etc.)

## Configuration System

### Configuration File (config.yaml)

The application uses a YAML-based configuration file with the following structure:

```yaml
global:
  base_url: "https://api.openai.com/v1"
  api_key: "${OPENAI_API_KEY}"
  max_retries: 3
  request_delay: 1s
  timeout: 30s
llms:  # Optional - if not specified, discover all models
  - name: "OpenAI GPT-4"
    endpoint: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4-turbo"
concurrency: 5
timeout: 60s
```

### Key Configuration Features

- **Environment Variable Substitution**: Supports `${VAR}` syntax for environment variables (e.g., `${OPENAI_API_KEY}`)
- **Automatic Model Discovery**: When no LLMs are specified in the config, the tool automatically discovers all available models from configured endpoints
- **Default Values**: The application provides sensible defaults for unspecified configuration options

## Database Layer

### Database Technology

- **Database**: SQLite with SQL Cipher encryption for security
- **Encryption**: All data is encrypted at rest using SQL Cipher
- **Connection**: Single connection with connection pooling for concurrent access

### Schema Design

The database schema consists of five main tables:

1. **providers** - Stores information about LLM providers (OpenAI, Anthropic, etc.)
   - id, name, endpoint, created_at, updated_at

2. **models** - Stores information about individual LLM models
   - id, provider_id, name, model_identifier, capabilities, created_at, updated_at
   - Foreign key relationship with providers table

3. **verification_results** - Stores results of verification runs
   - id, model_id, score, score_details, metadata, created_at
   - Foreign key relationship with models table
   - JSON field for score_details containing breakdown by dimension

4. **events** - Stores system events and notifications
   - id, event_type, payload, created_at
   - Used for real-time notifications and audit logging

5. **config_exports** - Stores exported configurations for other tools
   - id, format, content, created_at
   - Used for integration with other developer tools like OpenCode and Claude Code

### CRUD Operations

All database operations are implemented in `database/crud.go` with comprehensive methods for each entity type. Operations include:
- Create, Read, Update, Delete for providers and models
- Storage and retrieval of verification results
- Querying with filtering and sorting options
- Transaction support for operations requiring consistency

## LLM Integration

### API Compatibility

The application implements full OpenAI API compatibility, allowing it to work with any LLM provider that supports the OpenAI API format.

### Feature Detection

The system actively tests for feature support rather than relying on documentation. It verifies capabilities by sending test requests and analyzing responses.

Supported features detected:
- Tool/function calling
- Code generation and completion
- Code review and explanation
- Test generation
- Documentation abilities
- Refactoring capabilities
- Architecture understanding
- Security assessment
- Pattern recognition
- Embeddings functionality
- Reranking capabilities
- Audio, video, and image generation
- MCPs (Model Context Protocol)
- LSPs (Language Server Protocol)

### Feature Detection Methods

- **Tool Use**: Sends test requests with tool specifications and verifies the model uses them correctly
- **Code Generation**: Tests with specific programming tasks across multiple languages
- **Embeddings**: Makes embedding API calls with test text and verifies the response format
- **Streaming**: Sets stream=true parameter and verifies the response is streamed
- **Image Generation**: Sends image generation requests and verifies the response contains image data

## Scoring System

### Scoring Dimensions and Weights

The scoring system evaluates models across five dimensions with the following weights:

| Dimension | Weight | Description |
|-----------|--------|-------------|
| Code Capability | 40% | How well the model handles coding tasks across multiple programming languages |
| Responsiveness | 15% | Response time and throughput under load |
| Reliability | 15% | Availability and consistency of responses |
| Feature Richness | 20% | Number and quality of supported features and capabilities |
| Value Proposition | 10% | Overall value for coding tasks considering cost and performance |

### Scoring Calculation

The final score is calculated as a weighted average:

```
final_score = (code_capability * 0.4) + (responsiveness * 0.15) + (reliability * 0.15) + (feature_richness * 0.2) + (value_proposition * 0.1)
```

Each dimension is scored on a scale of 0-100, with detailed breakdowns stored in the score_details JSON field.

### Output Formats

Verification results are available in two formats:
- **Human-readable**: Markdown reports with detailed analysis and visualizations
- **Machine-readable**: JSON format for programmatic consumption

## API Documentation

### REST API Overview

The application provides a comprehensive REST API for programmatic access with authentication and rate limiting.

**Base URL**: `https://api.llmverifier.com/v1` or configured base_url from config.yaml

### Authentication

- **Method**: JWT-based authentication
- **Header**: `Authorization: Bearer <token>`
- **Token Generation**: Tokens are generated upon successful login and have a configurable expiration

### Rate Limiting

- **Authenticated Users**: 1000 requests per hour
- **Unauthenticated Users**: 100 requests per hour
- **Headers**: Rate limit information is included in response headers (X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset)

### Response Format

All API responses follow a consistent format:

```json
{
  "success": true,
  "data": { /* response data */ },
  "metadata": { /* pagination, timestamps, etc. */ }
}
```

### Key Endpoints

#### GET /models
List all verified models with filtering options

**Parameters**:
- `provider` (optional): Filter by provider (openai, anthropic, etc.)
- `capability` (optional): Filter by capability (code, embeddings, etc.)
- `min_score` (optional): Filter by minimum overall score
- `page` (optional): Page number for pagination
- `limit` (optional): Number of results per page (default: 20)

**Response**:
```json
{
  "success": true,
  "data": [
    {
      "id": "model-123",
      "name": "gpt-4-turbo",
      "provider": "openai",
      "score": 95.2,
      "capabilities": ["code", "tools", "images"],
      "created_at": "2025-01-15T10:00:00Z"
    }
  ],
  "metadata": {
    "total": 45,
    "page": 1,
    "pages": 3,
    "limit": 20
  }
}
```

#### POST /models/{id}/verify
Trigger verification of a specific model

**Path Parameters**:
- `id`: The ID of the model to verify

**Request Body**:
```json
{
  "test_suite": "comprehensive", // or "quick", "code-only", etc.
  "config": { /* optional configuration overrides */ }
}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "verification_id": "verify-456",
    "model_id": "model-123",
    "status": "started",
    "started_at": "2025-01-15T10:05:00Z"
  }
}
```

#### GET /verification-results
Get verification results with filtering and sorting

**Parameters**:
- `model_id` (optional): Filter by model ID
- `provider` (optional): Filter by provider
- `min_score` (optional): Filter by minimum score
- `start_date` (optional): Filter by start date
- `end_date` (optional): Filter by end date
- `sort` (optional): Sort field (score, created_at, etc.)
- `order` (optional): Sort order (asc, desc)
- `page` (optional): Page number
- `limit` (optional): Results per page

**Response**:
```json
{
  "success": true,
  "data": [
    {
      "id": "result-789",
      "model_id": "model-123",
      "score": 95.2,
      "score_details": {
        "code_capability": 98,
        "responsiveness": 92,
        "reliability": 95,
        "feature_richness": 96,
        "value_proposition": 90
      },
      "metadata": { /* test configuration, environment, etc. */ },
      "created_at": "2025-01-15T10:10:00Z"
    }
  ],
  "metadata": {
    "total": 150,
    "page": 1,
    "pages": 8,
    "limit": 20
  }
}
```

#### POST /config-exports
Create a configuration export for integration with other tools

**Request Body**:
```json
{
  "format": "opencode", // or "claude", "vscode", etc.
  "models": ["model-123", "model-456"], // optional: specific models to include
  "include_all_verified": false // whether to include all verified models
}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "export_id": "export-101",
    "format": "opencode",
    "content": "{\"models\": [...]}", // exported configuration
    "created_at": "2025-01-15T10:15:00Z"
  }
}
```

#### GET /health
Health check endpoint

**Response**:
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "version": "1.0.0",
    "database": "connected",
    "timestamp": "2025-01-15T10:20:00Z"
  }
}
```

## Testing Framework

### Test Types

The project implements six distinct types of tests with specific requirements:

1. **Unit Tests**
   - Test individual functions and methods in isolation
   - Use Go's built-in testing package
   - Mock dependencies using testify/mock
   - 100% line coverage required

2. **Integration Tests**
   - Test interactions between components
   - Use real database connections (SQLite in-memory)
   - Test API endpoints with httptest
   - 95% branch coverage required

3. **End-to-End (E2E) Tests**
   - Test complete workflows from CLI/API to database
   - Use temporary database files
   - Test error handling and edge cases
   - Cover all major user journeys

4. **Automation Tests**
   - Test CLI commands and automation scripts
   - Verify script behavior and output
   - Test configuration file parsing
   - Ensure backward compatibility

5. **Security Tests**
   - Test for common vulnerabilities (SQL injection, XSS, etc.)
   - Verify authentication and authorization
   - Test input validation and sanitization
   - Check for sensitive data exposure

6. **Performance/Benchmark Tests**
   - Measure response times under load
   - Test throughput and concurrency
   - Identify performance bottlenecks
   - Establish performance baselines

### Test Coverage Requirements

The project has stringent test coverage requirements:
- **Line Coverage**: 100% across all packages
- **Branch Coverage**: 95% minimum
- **Function Coverage**: 100% (every function must be tested)
- **Flaky Tests**: Zero tolerance - all tests must be deterministic

### Test Runner

The `test_runner.sh` script orchestrates all test types:

```bash
#!/bin/bash

# Run unit tests
go test ./tests/unit_test.go -v

# Run integration tests
go test ./tests/integration_test.go -v

# Run end-to-end tests
go test ./tests/e2e_test.go -v

# Run performance tests
go test ./tests/performance_test.go -bench=.

# Run security tests
go test ./tests/security_test.go -v

# Run automation tests
go test ./tests/automation_test.go -v

# Run all tests in all packages
go test ./tests/... -v
```

### Test Helper Utilities

The `test_helpers.go` file provides utilities for testing:
- `SetupTestServer()` - Creates a mock LLM API server for testing
- `CreateTestDB()` - Creates an in-memory SQLite database for integration tests
- `CreateTestConfig()` - Creates a test configuration with mock values
- `MockLLMResponse()` - Sets up mock responses for LLM API calls

## Client Architecture

### Multi-Client Design

The application supports multiple client interfaces to accommodate different user preferences:

1. **TUI (Terminal User Interface)**
   - Interactive terminal interface using tview or similar library
   - Real-time data browsing and filtering
   - Keyboard navigation and shortcuts
   - Color-coded output for easy interpretation

2. **REST API**
   - Programmatic access for automation and integration
   - Comprehensive Swagger documentation
   - Authentication and rate limiting
   - JSON responses for easy parsing

3. **Web Client**
   - Angular-based web interface
   - Data visualization with charts and graphs
   - Responsive design for different screen sizes
   - Export functionality for reports

4. **Desktop Application**
   - Electron-based cross-platform desktop app
   - Native look and feel on each platform
   - System tray integration
   - Offline capabilities with local storage

5. **Mobile Applications**
   - Flutter-based iOS and Android apps
   - Touch-optimized interface
   - Push notifications for verification completion
   - Camera integration for QR code scanning of reports

### Event System

The application implements a real-time event system using:
- **WebSocket**: For real-time updates in web and desktop clients
- **gRPC Streaming**: For high-performance streaming in programmatic clients
- **Server-Sent Events (SSE)**: For simple real-time updates in web clients

Events include:
- Verification progress updates
- Completion notifications
- Error alerts
- System status changes

### Notification System

The application integrates with multiple notification channels:
- **Slack**: Send verification results to Slack channels
- **Email**: Send detailed reports via email
- **Telegram**: Send notifications to Telegram bots
- **Matrix**: Integrate with Matrix rooms
- **WhatsApp**: Send alerts via WhatsApp

## Key Commands

### Build Commands

```bash
# Build the application
go build -o llm-verifier ./cmd/main.go

# Build with specific version
go build -ldflags "-X main.Version=1.0.0" -o llm-verifier ./cmd/main.go
```

### Run Commands

```bash
# Run the application
./llm-verifier

# Run with specific config file
./llm-verifier --config /path/to/config.yaml

# Run specific command (if supported)
./llm-verifier verify --model gpt-4
```

### Test Commands

```bash
# Run all tests using the test runner
./test_runner.sh

# Run specific test type
go test ./tests/unit_test.go -v
go test ./tests/integration_test.go -v

# Run all tests in all packages
go test ./tests/... -v

# Run benchmark tests
go test ./tests/performance_test.go -bench=.

# Generate coverage report
go test ./tests/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Database Commands

```bash
# Initialize database
./llm-verifier db init

# Migrate database (if migrations are implemented)
./llm-verifier db migrate

# Backup database
cp llmverifier.db llmverifier.db.backup

# Restore database
cp llmverifier.db.backup llmverifier.db
```

### Configuration Commands

```bash
# Generate default config file
./llm-verifier config init

# Validate config file
./llm-verifier config validate

# List current config
./llm-verifier config list
```

## Development Gotchas

### Automatic Model Discovery

When no LLMs are specified in the config file, the tool automatically discovers all available models from the configured API endpoints. This behavior can be surprising if you expect only specific models to be tested.

**Tip**: To test only specific models, explicitly list them in the `llms` section of the config file.

### Comprehensive Testing Requirements

The project has extremely high test coverage requirements:
- 100% line coverage
- 95% branch coverage
- 100% function coverage

**Tip**: Use `go test -cover` to check coverage before committing. Focus on edge cases and error paths to achieve full coverage.

### Security Focus

The application includes multiple security features that must be considered:
- **SQL Cipher Encryption**: All database data is encrypted at rest
- **JWT Authentication**: API endpoints require authentication
- **API Key Protection**: API keys are never logged and are stored securely

**Tip**: When debugging, avoid logging sensitive information. Use debug levels that exclude sensitive data.

### Configuration Export

The system can export configurations for integration with other developer tools. The export format may affect how models are represented in external tools.

**Tip**: Test configuration exports with the target tool to ensure compatibility.

### Feature Detection Approach

The system actively tests for feature support rather than relying on documentation. This means that even if a model claims to support a feature, it will be tested to verify actual capability.

**Tip**: When adding support for a new LLM provider, ensure that the feature detection tests are comprehensive and cover edge cases.

### Scoring Algorithm Details

The scoring algorithm uses specific weights that may not be immediately obvious:
- Code Capability: 40%
- Responsiveness: 15%
- Reliability: 15%
- Feature Richness: 20%
- Value Proposition: 10%

**Tip**: When interpreting scores, consider the weight distribution. A high feature richness score won't compensate for a low code capability score.

### API Compatibility Requirements

The LLM client expects OpenAI API compatibility. Even if a provider has a different API, it must be wrapped to match the OpenAI format.

**Tip**: When adding a new provider, create a wrapper that translates between the provider's API and the OpenAI format.

### Database Encryption

The SQLite database uses SQL Cipher encryption. This means that the database file cannot be opened with standard SQLite tools without the encryption key.

**Tip**: Use the application's built-in database commands for backup and restore operations to handle encryption properly.

### Event-Driven Architecture

The application uses an event-driven architecture with real-time notifications. This means that operations may complete asynchronously.

**Tip**: When testing workflows, account for potential delays between triggering an operation and receiving the completion event.
