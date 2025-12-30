# LLM Verifier Developer Manual

<p align="center">
  <img src="images/Logo.jpeg" alt="LLMsVerifier Logo" width="150" height="150">
</p>

<p align="center">
  <strong>Verify. Monitor. Optimize.</strong>
</p>

---

## Introduction

This manual provides comprehensive guidance for developers working on the LLM Verifier codebase. It covers development workflows, architecture patterns, testing strategies, and contribution guidelines.

## Development Environment Setup

### Prerequisites

```bash
# Required tools
go version # 1.21+
docker --version # 20.10+
git --version # 2.25+
make --version # 4.0+

# Install development dependencies
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/cosmtrek/air@latest
go install github.com/vektra/mockery/v2@latest
```

### Local Development Setup

```bash
# Clone repository
git clone https://github.com/your-org/llm-verifier.git
cd llm-verifier

# Install dependencies
go mod download

# Copy configuration template
cp config/production_config.yaml config/local_config.yaml

# Edit local configuration
vim config/local_config.yaml
# Set development database path, test API keys, etc.

# Initialize development database
go run ./cmd init-db --config config/local_config.yaml

# Start development server with hot reload
air
```

### IDE Configuration

#### VS Code
```json
{
  "go.useLanguageServer": true,
  "go.formatTool": "goimports",
  "go.lintTool": "golangci-lint",
  "go.testFlags": ["-v", "-race"],
  "go.testTimeout": "10m",
  "go.coverOnSingleTest": true,
  "go.coverOnSave": true
}
```

#### GoLand
- Enable Go modules
- Set GOPROXY=https://proxy.golang.org
- Configure test coverage display
- Set up run configurations for different test suites

## Architecture Overview

### Core Components

```
┌─────────────────────────────────────────────────┐
│                LLM Verifier                      │
├─────────────────────────────────────────────────┤
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ │
│  │   API       │ │   Engine    │ │   Database  │ │
│  │   Layer     │ │   Layer     │ │   Layer     │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ │
├─────────────────────────────────────────────────┤
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ │
│  │ Providers   │ │ Tests       │ │ Monitoring  │ │
│  │             │ │ Framework   │ │             │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ │
└─────────────────────────────────────────────────┘
```

### Provider Architecture

Each LLM provider implements a standard adapter interface:

```go
type ProviderAdapter interface {
    // Core functionality
    Name() string
    Endpoint() string
    SupportsModel(modelID string) bool

    // Model operations
    DiscoverModels(ctx context.Context, apiKey string) ([]*database.Model, error)
    VerifyModel(ctx context.Context, apiKey, modelID, prompt string) (*database.VerificationResult, error)

    // Health and configuration
    ValidateConfig(config map[string]interface{}) error
    IsHealthy(ctx context.Context, apiKey string) bool
    GetRateLimits() (int, int) // requests/min, tokens/min
    GetPricing() (float64, float64) // input/1M, output/1M
}
```

## Development Workflows

### Adding a New Provider

#### Step 1: Create Provider Adapter

```bash
# Create provider directory and files
mkdir llm-verifier/providers/newprovider
touch llm-verifier/providers/newprovider/adapter.go
touch llm-verifier/providers/newprovider/adapter_test.go
```

#### Step 2: Implement Adapter

```go
package newprovider

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"

    "llm-verifier/database"
    "llm-verifier/providers"
)

type NewProviderAdapter struct {
    *providers.BaseAdapter
}

func NewNewProviderAdapter(client *http.Client, endpoint, apiKey string) *NewProviderAdapter {
    return &NewProviderAdapter{
        BaseAdapter: &providers.BaseAdapter{
            Client:   client,
            Endpoint: endpoint,
            APIKey:   apiKey,
            Headers: map[string]string{
                "Content-Type":  "application/json",
                "Authorization": fmt.Sprintf("Bearer %s", apiKey),
            },
        },
    }
}

func (n *NewProviderAdapter) Name() string {
    return "newprovider"
}

func (n *NewProviderAdapter) Endpoint() string {
    return n.BaseAdapter.Endpoint
}

// Implement remaining interface methods...
```

#### Step 3: Update Core Systems

```go
// Update llm-verifier/client/http_client.go
providerEndpoints := map[string]string{
    // ... existing providers
    "newprovider": "https://api.newprovider.com/v1",
}

// Update llm-verifier/enhanced/limits.go
case "newprovider":
    return ld.detectNewProviderLimits(headers)

// Update llm-verifier/enhanced/pricing.go
case "newprovider":
    return pd.detectNewProviderPricing(modelID)

// Update llm-verifier/llmverifier/config_export.go
case "NewProvider":
    // Add OpenCode, Crush, and Claude Code export logic

// Update llm-verifier/config/production_config.go
NewProvider ProviderConfig `yaml:"newprovider"`
```

#### Step 4: Add Comprehensive Tests

```go
func TestNewProviderAdapter(t *testing.T) {
    // Unit tests for adapter functionality
}

func TestNewProviderIntegration(t *testing.T) {
    // Integration tests with mock API
}

func TestNewProviderE2E(t *testing.T) {
    // End-to-end verification tests
}
```

### Testing Strategy

#### Unit Testing

```go
func TestProviderAdapter(t *testing.T) {
    adapter := NewTestAdapter()

    t.Run("Name", func(t *testing.T) {
        assert.Equal(t, "testprovider", adapter.Name())
    })

    t.Run("SupportsModel", func(t *testing.T) {
        assert.True(t, adapter.SupportsModel("test-model"))
        assert.False(t, adapter.SupportsModel("unsupported"))
    })

    t.Run("ValidateConfig", func(t *testing.T) {
        validConfig := map[string]interface{}{
            "api_key": "test-key",
        }
        assert.NoError(t, adapter.ValidateConfig(validConfig))

        invalidConfig := map[string]interface{}{}
        assert.Error(t, adapter.ValidateConfig(invalidConfig))
    })
}
```

#### Integration Testing

```go
func TestProviderIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // Setup test server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Mock API responses
        if r.URL.Path == "/v1/models" {
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusOK)
            json.NewEncoder(w).Encode(map[string]interface{}{
                "data": []map[string]interface{}{
                    {"id": "test-model", "object": "model"},
                },
            })
        }
    }))
    defer server.Close()

    adapter := NewTestAdapter(server.Client(), server.URL, "test-key")

    t.Run("DiscoverModels", func(t *testing.T) {
        models, err := adapter.DiscoverModels(context.Background(), "test-key")
        assert.NoError(t, err)
        assert.Len(t, models, 1)
        assert.Equal(t, "test-model", models[0].ModelID)
    })
}
```

#### End-to-End Testing

```go
func TestProviderE2E(t *testing.T) {
    if os.Getenv("E2E_TEST") != "true" {
        t.Skip("Skipping E2E test - set E2E_TEST=true to run")
    }

    // Full workflow test with real API
    adapter := NewProviderAdapter(http.DefaultClient, "https://api.provider.com", os.Getenv("PROVIDER_API_KEY"))

    // Test complete verification workflow
    result, err := adapter.VerifyModel(context.Background(), os.Getenv("PROVIDER_API_KEY"), "test-model", "Test prompt")
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Greater(t, result.Score, 0.0)
}
```

#### Performance Testing

```go
func BenchmarkProviderVerification(b *testing.B) {
    adapter := NewProviderAdapter(http.DefaultClient, "https://api.provider.com", "test-key")

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := adapter.VerifyModel(context.Background(), "test-key", "test-model", "Benchmark test")
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### Code Quality Standards

#### Linting

```bash
# Run linter
golangci-lint run

# Auto-fix issues
golangci-lint run --fix
```

#### Code Coverage

```bash
# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Minimum coverage requirement: 90%
go test ./... -cover | grep "coverage:" | awk '{sum += $5} END {print "Total coverage:", sum/NR "%"}'
```

#### Security Scanning

```bash
# Run security checks
go install github.com/securecodewarrior/govulncheck@latest
govulncheck ./...

# Check for secrets
go install github.com/zricethezav/gitleaks@latest
gitleaks detect --verbose
```

## API Integration Patterns

### REST API Integration

```go
func (a *Adapter) makeRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
    var reqBody io.Reader
    if body != nil {
        jsonData, err := json.Marshal(body)
        if err != nil {
            return nil, fmt.Errorf("failed to marshal request: %w", err)
        }
        reqBody = bytes.NewReader(jsonData)
    }

    url := fmt.Sprintf("%s%s", a.Endpoint, path)
    req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    // Set headers
    for key, value := range a.Headers {
        req.Header.Set(key, value)
    }

    return a.Client.Do(req)
}
```

### Streaming Response Handling

```go
func (a *Adapter) handleStreamingResponse(resp *http.Response, callback func(string)) error {
    scanner := bufio.NewScanner(resp.Body)
    for scanner.Scan() {
        line := scanner.Text()
        if strings.HasPrefix(line, "data: ") {
            data := strings.TrimPrefix(line, "data: ")
            if data == "[DONE]" {
                break
            }

            var streamResp StreamResponse
            if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
                continue // Skip malformed lines
            }

            if callback != nil {
                callback(streamResp.Choices[0].Delta.Content)
            }
        }
    }
    return scanner.Err()
}
```

### Error Handling

```go
func (a *Adapter) handleAPIError(resp *http.Response) error {
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return fmt.Errorf("failed to read error response: %w", err)
    }

    var apiError struct {
        Error struct {
            Type    string `json:"type"`
            Message string `json:"message"`
            Code    string `json:"code"`
        } `json:"error"`
    }

    if err := json.Unmarshal(body, &apiError); err != nil {
        return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
    }

    return fmt.Errorf("API error %s: %s", apiError.Error.Type, apiError.Error.Message)
}
```

## Database Operations

### Repository Pattern

```go
type ProviderRepository interface {
    Create(ctx context.Context, provider *database.Provider) error
    GetByID(ctx context.Context, id int64) (*database.Provider, error)
    Update(ctx context.Context, provider *database.Provider) error
    Delete(ctx context.Context, id int64) error
    List(ctx context.Context, limit, offset int) ([]*database.Provider, error)
}

type providerRepository struct {
    db *database.Database
}

func NewProviderRepository(db *database.Database) ProviderRepository {
    return &providerRepository{db: db}
}

func (r *providerRepository) Create(ctx context.Context, provider *database.Provider) error {
    query := `
        INSERT INTO providers (name, endpoint, api_key_encrypted, description, website, is_active, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `

    result, err := r.db.ExecContext(ctx, query,
        provider.Name, provider.Endpoint, provider.APIKeyEncrypted,
        provider.Description, provider.Website, provider.IsActive,
        provider.CreatedAt, provider.UpdatedAt)
    if err != nil {
        return fmt.Errorf("failed to create provider: %w", err)
    }

    id, err := result.LastInsertId()
    if err != nil {
        return fmt.Errorf("failed to get inserted ID: %w", err)
    }

    provider.ID = id
    return nil
}
```

### Transaction Management

```go
func (r *providerRepository) CreateWithModels(ctx context.Context, provider *database.Provider, models []*database.Model) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Insert provider
    if err := r.createProviderTx(ctx, tx, provider); err != nil {
        return err
    }

    // Insert models
    for _, model := range models {
        model.ProviderID = provider.ID
        if err := r.createModelTx(ctx, tx, model); err != nil {
            return err
        }
    }

    return tx.Commit()
}
```

## Configuration Management

### Environment-Based Configuration

```go
type Config struct {
    Database struct {
        Path    string `env:"DB_PATH" default:":memory:"`
        Key     string `env:"DB_ENCRYPTION_KEY"`
    } `yaml:"database"`

    API struct {
        Port       int    `env:"PORT" default:"8080"`
        JWTSecret  string `env:"JWT_SECRET" required:"true"`
        RateLimit  int    `env:"RATE_LIMIT" default:"100"`
    } `yaml:"api"`

    Providers map[string]ProviderConfig `yaml:"providers"`
}

func LoadConfig(path string) (*Config, error) {
    cfg := &Config{}

    // Load from file
    if err := cleanenv.ReadConfig(path, cfg); err != nil {
        return nil, fmt.Errorf("failed to load config from %s: %w", path, err)
    }

    // Override with environment variables
    if err := cleanenv.ReadEnv(cfg); err != nil {
        return nil, fmt.Errorf("failed to read environment: %w", err)
    }

    // Validate required fields
    if err := validateConfig(cfg); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }

    return cfg, nil
}
```

## Monitoring and Logging

### Structured Logging

```go
import "go.uber.org/zap"

type Logger struct {
    *zap.Logger
}

func NewLogger() *Logger {
    config := zap.NewProductionConfig()
    config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

    logger, err := config.Build()
    if err != nil {
        panic(fmt.Sprintf("failed to build logger: %v", err))
    }

    return &Logger{Logger: logger}
}

func (l *Logger) LogProviderOperation(provider, operation string, duration time.Duration, err error) {
    fields := []zap.Field{
        zap.String("provider", provider),
        zap.String("operation", operation),
        zap.Duration("duration", duration),
    }

    if err != nil {
        l.Error("Provider operation failed", append(fields, zap.Error(err))...)
    } else {
        l.Info("Provider operation completed", fields...)
    }
}
```

### Metrics Collection

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    verificationDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "llm_verifier_verification_duration_seconds",
            Help:    "Time taken to complete verification",
            Buckets: prometheus.DefBuckets,
        },
        []string{"provider", "model"},
    )

    providerRequests = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "llm_verifier_provider_requests_total",
            Help: "Total number of requests to providers",
        },
        []string{"provider", "status"},
    )
)

func init() {
    prometheus.MustRegister(verificationDuration)
    prometheus.MustRegister(providerRequests)
}

func RecordVerification(provider, model string, duration time.Duration) {
    verificationDuration.WithLabelValues(provider, model).Observe(duration.Seconds())
}

func RecordProviderRequest(provider, status string) {
    providerRequests.WithLabelValues(provider, status).Inc()
}
```

## Contributing Guidelines

### Commit Messages

```bash
# Format: type(scope): description
feat(provider): add support for Groq API
fix(test): resolve race condition in verification
docs(api): update endpoint documentation
refactor(engine): simplify verification logic
```

### Pull Request Process

1. **Create Feature Branch**
   ```bash
   git checkout -b feature/add-new-provider
   ```

2. **Write Tests First**
   ```bash
   go test ./... -v
   # Ensure tests pass before implementation
   ```

3. **Implement Feature**
   ```bash
   # Follow established patterns
   # Update documentation
   # Add comprehensive tests
   ```

4. **Code Review**
   ```bash
   golangci-lint run
   go test ./... -race -cover
   go mod tidy
   ```

5. **Submit PR**
   - Title: `feat: add support for NewProvider`
   - Description: Detailed explanation of changes
   - Tests: Coverage report
   - Breaking changes: Documented if any

### Code Review Checklist

- [ ] Tests pass with 100% coverage for new code
- [ ] No linting errors
- [ ] Documentation updated
- [ ] Security review completed
- [ ] Performance impact assessed
- [ ] Breaking changes documented
- [ ] Migration guide provided if needed

This developer manual provides the foundation for consistent, high-quality contributions to the LLM Verifier project. All developers should familiarize themselves with these patterns and guidelines.