# LLM Verifier - Complete User Manual

## Table of Contents

1. [Introduction](#introduction)
2. [Quick Start](#quick-start)
3. [Installation](#installation)
4. [Configuration](#configuration)
5. [Using the CLI](#using-the-cli)
6. [Using the TUI](#using-the-tui)
7. [Using the Web Interface](#using-the-web-interface)
8. [Using the REST API](#using-the-rest-api)
9. [Client SDKs](#client-sdks)
10. [Advanced Features](#advanced-features)
11. [Troubleshooting](#troubleshooting)
12. [Best Practices](#best-practices)
13. [API Reference](#api-reference)

## Introduction

The LLM Verifier is a comprehensive platform for verifying, benchmarking, and managing Large Language Models (LLMs) from multiple providers. It provides enterprise-grade features for model evaluation, performance monitoring, and configuration management.

### Key Features

- **Multi-Provider Support**: OpenAI, Anthropic, Google, Cohere, and more
- **Comprehensive Verification**: Code generation, reasoning, tool use, and multimodal capabilities
- **Performance Benchmarking**: Automated scoring and comparative analysis
- **Multi-Client Architecture**: CLI, TUI, Web, and REST API interfaces
- **Enterprise Security**: RBAC, audit trails, and encrypted credential storage
- **Event-Driven Architecture**: Real-time notifications and monitoring
- **Scheduling System**: Automated periodic verification and reporting
- **Export Capabilities**: Configuration export for AI CLI tools

## Quick Start

### Prerequisites

- Go 1.21+
- SQLite3
- Node.js 18+ (for web interface)
- Docker (optional)

### Basic Setup

1. **Clone and build:**
```bash
git clone https://github.com/your-org/llm-verifier.git
cd llm-verifier
go build -o llm-verifier cmd/main.go
```

2. **Configure providers:**
```yaml
# config.yaml
llms:
  - name: "gpt-4"
    endpoint: "https://api.openai.com/v1"
    api_key: "your-openai-key"
    model: "gpt-4"

  - name: "claude-3"
    endpoint: "https://api.anthropic.com"
    api_key: "your-anthropic-key"
    model: "claude-3-sonnet-20240229"
```

3. **Start the server:**
```bash
./llm-verifier server --port 8080
```

4. **Verify a model:**
```bash
curl -X POST http://localhost:8080/api/v1/models/gpt-4/verify \
  -H "Authorization: Bearer your-token"
```

## Installation

### From Source

```bash
# Clone repository
git clone https://github.com/your-org/llm-verifier.git
cd llm-verifier

# Build
go build -o llm-verifier cmd/main.go

# Install (optional)
sudo cp llm-verifier /usr/local/bin/
```

### Using Docker

```bash
# Build image
docker build -t llm-verifier .

# Run container
docker run -p 8080:8080 -v $(pwd)/data:/app/data llm-verifier
```

### Using Docker Compose

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f
```

## Configuration

### Main Configuration File

```yaml
# config.yaml
profile: "production"

database:
  path: "/app/data/llm-verifier.db"
  max_connections: 10

api:
  port: 8080
  enable_cors: true
  jwt_secret: "${JWT_SECRET:-your-secret-key}"
  rate_limit: 100
  burst_limit: 200

logging:
  level: "info"
  format: "json"
  file_path: "logs/llm-verifier.log"

monitoring:
  enable_metrics: true
  metrics_port: 9090

notifications:
  slack:
    enabled: true
    webhook_url: "https://hooks.slack.com/services/..."
  email:
    enabled: true
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    username: "your-email@gmail.com"
    password: "your-app-password"
    default_recipient: "admin@company.com"

llms:
  - name: "gpt-4-turbo"
    endpoint: "https://api.openai.com/v1"
    api_key: "sk-..."
    model: "gpt-4-turbo-preview"
    features:
      code_generation: true
      tool_use: true

  - name: "claude-3-sonnet"
    endpoint: "https://api.anthropic.com"
    api_key: "sk-ant-..."
    model: "claude-3-sonnet-20240229"
    features:
      reasoning: true
      multimodal: true
```

### Environment Variables

```bash
# Database
export LLM_DB_PATH="/path/to/database.db"

# API
export LLM_API_PORT="8080"
export LLM_JWT_SECRET="your-secret"

# Logging
export LLM_LOG_LEVEL="debug"
export LLM_LOG_FILE="/path/to/logs.log"

# Notifications
export LLM_SLACK_WEBHOOK="https://hooks.slack.com/..."
export LLM_SMTP_HOST="smtp.gmail.com"
export LLM_SMTP_USER="your-email@gmail.com"
export LLM_SMTP_PASS="your-password"
```

## Using the CLI

### Basic Commands

```bash
# Start server
llm-verifier server --port 8080

# Start TUI
llm-verifier tui --server-url http://localhost:8080

# Validate system
llm-verifier validate

# Export configurations
llm-verifier export opencode --output ./exports/
```

### Model Management

```bash
# List all models
curl http://localhost:8080/api/v1/models

# Get specific model
curl http://localhost:8080/api/v1/models/1

# Verify model
curl -X POST http://localhost:8080/api/v1/models/1/verify

# Create new model
curl -X POST http://localhost:8080/api/v1/models \
  -H "Content-Type: application/json" \
  -d '{"model_id": "gpt-4", "name": "GPT-4", "provider_id": 1}'
```

### Verification Results

```bash
# Get verification results
curl http://localhost:8080/api/v1/verification-results

# Get specific result
curl http://localhost:8080/api/v1/verification-results/1

# Filter by date range
curl "http://localhost:8080/api/v1/verification-results?start_date=2024-01-01&end_date=2024-01-31"
```

## Using the TUI

The Terminal User Interface provides an interactive way to manage models and view results.

### Navigation

- `1-4`: Jump to screens (Dashboard, Models, Providers, Verification)
- `←/→` or `h/l`: Navigate between screens
- `q` or `Ctrl+C`: Quit

### Dashboard Screen

- View system statistics
- Monitor verification progress
- See recent activity
- Real-time updates every 30 seconds

### Models Screen

- Browse all models with filtering
- View verification status and scores
- Trigger verification runs
- Search by name, provider, or capabilities

### Keyboard Shortcuts

```
Dashboard:
  r - Refresh data

Models:
  ↑/↓ or k/j - Navigate
  Enter/Space - Verify model
  f - Filter/Search
  r - Refresh

Global:
  1-4 - Switch screens
  q - Quit
```

## Using the Web Interface

### Accessing the Web UI

1. Start the server: `./llm-verifier server --port 8080`
2. Open browser: `http://localhost:8080`
3. Navigate through the Angular-based interface

### Dashboard Features

- **Real-time Metrics**: Live updates of system statistics
- **Interactive Charts**: Verification status, score distributions, activity graphs
- **Recent Activity Feed**: Latest system events and verification results
- **Quick Actions**: Refresh data, trigger verifications

### Model Management

- **Model Browser**: Filter and search through all models
- **Detailed Views**: Comprehensive model information and capabilities
- **Verification History**: Past verification results and trends
- **Bulk Operations**: Verify multiple models simultaneously

### Provider Management

- **Provider Overview**: All configured providers and their status
- **Configuration**: Update provider settings and credentials
- **Health Monitoring**: Provider API status and response times

## Using the REST API

### Authentication

```bash
# Login to get JWT token
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "password"}'

# Use token in subsequent requests
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  http://localhost:8080/api/v1/models
```

### Endpoints

#### Models
```
GET    /api/v1/models              # List models
GET    /api/v1/models/{id}         # Get model details
POST   /api/v1/models              # Create model
PUT    /api/v1/models/{id}         # Update model
DELETE /api/v1/models/{id}         # Delete model
POST   /api/v1/models/{id}/verify  # Verify model
```

#### Providers
```
GET    /api/v1/providers           # List providers
GET    /api/v1/providers/{id}      # Get provider details
POST   /api/v1/providers           # Create provider
PUT    /api/v1/providers/{id}      # Update provider
DELETE /api/v1/providers/{id}      # Delete provider
```

#### Verification Results
```
GET    /api/v1/verification-results           # List results
GET    /api/v1/verification-results/{id}      # Get result details
POST   /api/v1/verification-results           # Create result
PUT    /api/v1/verification-results/{id}      # Update result
DELETE /api/v1/verification-results/{id}      # Delete result
```

#### System
```
GET    /health                          # Health check
GET    /health/detailed                 # Detailed health
GET    /api/v1/system/info              # System info
GET    /api/v1/system/database-stats    # Database stats
```

## Client SDKs

### Go SDK

```go
import "github.com/your-org/llm-verifier/sdk/go"

client := llmverifier.NewLLMVerifierClient("http://localhost:8080", "your-token")

// Get models
models, err := client.GetModels(10, 0, "")
if err != nil {
    log.Fatal(err)
}

// Verify model
result, err := client.VerifyModel("gpt-4")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Score: %.2f\n", result.Score)
```

### Python SDK

```python
from llm_verifier_sdk import LLMVerifierClient

client = LLMVerifierClient("http://localhost:8080", api_key="your-token")

# Login
auth = client.login("admin", "password")
print(f"Logged in as: {auth['user']['username']}")

# Get models
models = client.get_models(limit=10)
print(f"Found {len(models)} models")

# Verify model
result = client.verify_model("gpt-4")
print(f"Score: {result['score']}")
```

### JavaScript/TypeScript SDK

```javascript
import { LLMVerifierClient } from 'llm-verifier-sdk';

const client = new LLMVerifierClient('http://localhost:8080', 'your-token');

// Get models
const models = await client.getModels({ limit: 10 });
console.log(`Found ${models.length} models`);

// Verify model
const result = await client.verifyModel('gpt-4');
console.log(`Score: ${result.score}`);
```

## Advanced Features

### Event System

The LLM Verifier uses an event-driven architecture for real-time notifications:

```go
// Subscribe to events
subscriber := &MySubscriber{}
eventBus.Subscribe(subscriber)

// Events are published automatically for:
// - Model verification completion
// - Score changes
// - System health changes
// - Schedule execution
// - Error occurrences
```

### Notification System

Configure notifications for different channels:

```yaml
notifications:
  slack:
    enabled: true
    webhook_url: "https://hooks.slack.com/services/..."
  email:
    enabled: true
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    username: "alerts@company.com"
```

### Scheduling System

Automate periodic tasks:

```go
// Create daily verification schedule
schedule := scheduler.CreateDailyVerificationSchedule("daily-check", []string{"all"})
scheduler.CreateSchedule(schedule)

// Cron expressions supported:
// "0 2 * * *"     - Daily at 2 AM
// "*/30 * * * *"  - Every 30 minutes
// "0 */2 * * *"   - Every 2 hours
```

### Security Features

#### Credential Management
```go
// Secure credential storage
cm := security.NewCredentialManager("master-key", store)
err := cm.StoreCredential("openai", "api_key", "sk-...")
key, err := cm.RetrieveCredential("openai", "api_key")
```

#### API Key Masking
```go
masker := security.NewAPIKeyMasker()
safeLog := masker.MaskAPIKeys(logEntry) // Masks sk-... keys
```

#### Role-Based Access Control
```go
rbac := security.NewRBACManager()
rbac.AddRole(security.Role{
    ID: "admin",
    Permissions: []string{"models.*", "providers.*"},
})
allowed := rbac.CheckPermission(userID, "models", "create", nil)
```

### Performance & Scalability

#### Caching
```go
cache := performance.NewCacheManager(
    performance.NewMemoryCacheBackend(),
    30*time.Minute,
)
cache.Set("models:list", models, 10*time.Minute)
```

#### Load Balancing
```go
lb := performance.NewLoadBalancer([]string{
    "http://server1:8080",
    "http://server2:8080",
})
instance := lb.NextInstance()
```

#### Database Optimization
```go
optimizer := performance.NewDatabaseOptimizer()
slowQueries := optimizer.GetSlowQueries(time.Second, 5)
suggestions := optimizer.SuggestIndexes()
```

## Troubleshooting

### Common Issues

#### Database Connection Issues
```bash
# Check database file permissions
ls -la data/llm-verifier.db

# Reset database
rm data/llm-verifier.db
./llm-verifier server --port 8080  # Will recreate
```

#### API Key Problems
```bash
# Verify API key format
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:8080/api/v1/models

# Check logs for authentication errors
tail -f logs/llm-verifier.log | grep "auth"
```

#### High Memory Usage
```bash
# Enable garbage collection tuning
export GOGC=50  # Lower GC threshold

# Monitor memory usage
go tool pprof http://localhost:8080/debug/pprof/heap
```

#### Slow Performance
```bash
# Check database query performance
curl http://localhost:8080/api/v1/system/database-stats

# Enable query logging
export LLM_LOG_LEVEL=debug
```

### Debug Mode

```bash
# Enable debug logging
export LLM_LOG_LEVEL=debug

# Start with verbose output
./llm-verifier server --port 8080 --verbose

# Check health endpoints
curl http://localhost:8080/health/detailed
```

### Log Analysis

```bash
# Search for errors
grep "ERROR" logs/llm-verifier.log

# Find slow queries
grep "slow.*query" logs/llm-verifier.log

# Check API response times
grep "response.*time" logs/llm-verifier.log
```

## Best Practices

### Configuration Management

1. **Use environment-specific configs**
2. **Never commit secrets to version control**
3. **Use strong, unique JWT secrets**
4. **Regularly rotate API keys**

#### LLM Configuration Tools

The LLM Verifier provides specialized tools for managing configurations across different platforms:

**Crush Configuration Generator:**
```bash
# Generate Crush config from latest discovery
go run crush_config_converter.go challenges/results/provider_models_discovery/.../results/providers_crush.json

# The output will be a valid Crush config with:
# - Streaming flags set for compatible models
# - Accurate cost estimates
# - Provider-specific settings
```

**OpenCode Configuration:**
- Located in `test_exports/export_claude_code.json`
- Automatically includes streaming support for all compatible models
- Verified model capabilities and settings

**Best Practices:**
- Run configuration generation after each discovery challenge
- Verify streaming flags are correctly set
- Test configurations before deployment

### Performance Optimization

1. **Enable caching for frequently accessed data**
2. **Use database indexes for common queries**
3. **Configure appropriate connection pool sizes**
4. **Monitor memory usage and GC performance**

### Security Practices

1. **Implement least privilege access**
2. **Enable audit logging for sensitive operations**
3. **Regularly update dependencies**
4. **Use HTTPS in production**
5. **Implement rate limiting**

### Monitoring & Alerting

1. **Set up alerts for system health issues**
2. **Monitor API response times**
3. **Track verification success rates**
4. **Set up log aggregation and analysis**

### Backup & Recovery

1. **Regular database backups**
2. **Test restore procedures**
3. **Document disaster recovery process**
4. **Keep multiple backup generations**

## API Reference

### Authentication Endpoints

#### POST /auth/login
Authenticate user and return JWT token.

**Request:**
```json
{
  "username": "string",
  "password": "string"
}
```

**Response:**
```json
{
  "token": "jwt_token_string",
  "expires_at": "2024-12-31T23:59:59Z",
  "user": {
    "id": 1,
    "username": "admin",
    "email": "admin@example.com",
    "role": "admin"
  }
}
```

#### POST /auth/refresh
Refresh JWT token.

### Model Endpoints

#### GET /api/v1/models
List models with optional filtering.

**Query Parameters:**
- `limit` (integer): Maximum results (default: 50)
- `offset` (integer): Pagination offset (default: 0)
- `provider` (string): Filter by provider name
- `status` (string): Filter by verification status

**Response:**
```json
[
  {
    "id": 1,
    "provider_id": 1,
    "model_id": "gpt-4",
    "name": "GPT-4",
    "overall_score": 95.2,
    "verification_status": "verified",
    "created_at": "2024-01-01T00:00:00Z"
  }
]
```

#### GET /api/v1/models/{id}
Get detailed model information.

#### POST /api/v1/models
Create a new model (admin only).

#### PUT /api/v1/models/{id}
Update model information (admin only).

#### DELETE /api/v1/models/{id}
Delete a model (admin only).

#### POST /api/v1/models/{id}/verify
Trigger verification for a model.

### Provider Endpoints

#### GET /api/v1/providers
List all providers.

#### GET /api/v1/providers/{id}
Get provider details.

#### POST /api/v1/providers
Create provider (admin only).

#### PUT /api/v1/providers/{id}
Update provider (admin only).

#### DELETE /api/v1/providers/{id}
Delete provider (admin only).

### Verification Results Endpoints

#### GET /api/v1/verification-results
List verification results.

#### GET /api/v1/verification-results/{id}
Get verification result details.

### System Endpoints

#### GET /health
Basic health check.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T12:00:00Z",
  "uptime": "24h30m45s",
  "version": "1.0.0"
}
```

#### GET /health/detailed
Detailed health information.

#### GET /health/ready
Readiness probe for Kubernetes.

#### GET /health/live
Liveness probe for Kubernetes.

#### GET /metrics
Prometheus metrics.

#### GET /api/v1/system/info
System information and statistics.

#### GET /api/v1/system/database-stats
Database performance statistics.

---

## Support

### Getting Help

1. **Documentation**: Check this manual first
2. **Logs**: Enable debug logging and check logs
3. **Health Checks**: Use `/health/detailed` for diagnostics
4. **Community**: GitHub issues and discussions

### Reporting Issues

When reporting issues, please include:

1. **Version information**: `./llm-verifier --version`
2. **Configuration**: Redacted config file
3. **Logs**: Relevant log entries
4. **Steps to reproduce**: Detailed reproduction steps
5. **Environment**: OS, Go version, system specs

### Feature Requests

Feature requests should include:

1. **Use case**: What problem are you trying to solve?
2. **Proposed solution**: How should it work?
3. **Alternatives**: Other approaches considered
4. **Impact**: How it affects existing functionality

---

*This manual is continuously updated. Check for the latest version at the project repository.*