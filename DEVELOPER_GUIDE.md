# LLM Verifier Developer Guide

## Architecture Overview

LLM Verifier is a comprehensive Go application for verifying and benchmarking Large Language Models across multiple providers. The system provides automated testing, performance scoring, and configuration export capabilities.

### Core Components

```
llm-verifier/
├── cmd/                    # Command-line interface
├── llmverifier/           # Core business logic
│   ├── config_export.go   # Configuration export functionality
│   ├── verifier.go        # Model verification engine
│   ├── analytics.go       # Analytics and monitoring
│   └── migration.go       # Configuration migration tools
├── providers/             # Provider-specific implementations
├── database/              # Data persistence layer
├── logging/               # Structured logging
└── tests/                 # Comprehensive test suite
```

### Key Design Patterns

1. **Dependency Injection**: Services accept dependencies through interfaces
2. **Strategy Pattern**: Different verification strategies for different providers
3. **Observer Pattern**: Event-driven architecture for monitoring
4. **Factory Pattern**: Provider and service instantiation
5. **Repository Pattern**: Data access abstraction

---

## Getting Started with Development

### Prerequisites

- Go 1.21 or later
- Git
- Make (optional, for build automation)
- Docker (for integration testing)

### Development Setup

1. **Clone and setup:**
   ```bash
   git clone https://github.com/your-org/llm-verifier.git
   cd llm-verifier
   go mod download
   ```

2. **Run tests:**
   ```bash
   go test ./... -v
   ```

3. **Build the application:**
   ```bash
   go build ./cmd/main.go -o llm-verifier
   ```

4. **Run in development mode:**
   ```bash
   export LLM_VERIFIER_DEBUG=true
   ./llm-verifier --help
   ```

### Development Workflow

1. Create a feature branch: `git checkout -b feature/your-feature`
2. Make changes with tests
3. Run full test suite: `go test ./... -race -cover`
4. Update documentation if needed
5. Submit pull request

---

## Core Concepts

### Verification Process

The verification process consists of several stages:

1. **Discovery**: Identify available models from providers
2. **Preparation**: Set up test scenarios and prompts
3. **Execution**: Run tests against each model
4. **Scoring**: Calculate performance metrics
5. **Reporting**: Generate comprehensive reports

### Scoring Algorithm

Models are scored across multiple dimensions:

```go
type PerformanceScore struct {
    OverallScore      float64 // Weighted average of all metrics
    CodeCapability    float64 // Code generation and analysis ability
    Responsiveness    float64 // API response times
    Reliability       float64 // Error rates and consistency
    FeatureRichness   float64 // Advanced feature support
    ValueProposition  float64 // Cost vs. performance ratio
}
```

**Scoring Weights:**
- Code Capability: 25%
- Responsiveness: 20%
- Reliability: 20%
- Feature Richness: 20%
- Value Proposition: 15%

### Provider Architecture

Each provider implements the `Provider` interface:

```go
type Provider interface {
    SendMessages(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*ProviderResponse, error)
    StreamResponse(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent
    Model() models.Model
}
```

**Supported Providers:**
- OpenAI (GPT-3.5, GPT-4, GPT-4o)
- Anthropic (Claude models)
- Google (Gemini)
- Groq (Fast inference)
- Together AI
- Fireworks AI
- And 15+ more providers

---

## Adding New Providers

### Step 1: Define Provider Structure

Create a new provider file in `providers/`:

```go
// providers/custom_provider.go
package providers

type CustomProvider struct {
    apiKey     string
    baseURL    string
    model      models.Model
    httpClient *http.Client
}

func NewCustomProvider(apiKey, baseURL string, model models.Model) *CustomProvider {
    return &CustomProvider{
        apiKey:     apiKey,
        baseURL:    baseURL,
        model:      model,
        httpClient: &http.Client{Timeout: 30 * time.Second},
    }
}
```

### Step 2: Implement Provider Interface

```go
func (p *CustomProvider) SendMessages(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*ProviderResponse, error) {
    // Convert messages to provider format
    requestBody := p.convertMessages(messages)

    // Add tools if supported
    if len(tools) > 0 {
        requestBody.Tools = p.convertTools(tools)
    }

    // Make API request
    resp, err := p.makeRequest(ctx, "POST", "/chat/completions", requestBody)
    if err != nil {
        return nil, fmt.Errorf("Custom provider request failed: %w", err)
    }

    // Parse response
    return p.parseResponse(resp)
}

func (p *CustomProvider) StreamResponse(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent {
    // Implement streaming if supported
    ch := make(chan ProviderEvent)

    go func() {
        defer close(ch)
        // Streaming implementation
    }()

    return ch
}

func (p *CustomProvider) Model() models.Model {
    return p.model
}
```

### Step 3: Add to Provider Factory

Update `config_export.go` to include the new provider:

```go
func NewProvider(providerName models.ModelProvider, opts ...ProviderClientOption) (Provider, error) {
    // ... existing cases ...

    case models.ProviderCustom:
        return &baseProvider[CustomClient]{
            options: clientOptions,
            client:  newCustomClient(clientOptions),
        }, nil

    // ... rest of cases ...
}
```

### Step 4: Update Provider Detection

Add custom provider detection in `extractProvider()`:

```go
func extractProvider(endpoint string) string {
    // ... existing patterns ...

    if strings.Contains(endpoint, "custom-api.com") {
        return "custom"
    }

    // ... existing logic ...
}
```

### Step 5: Add Tests

Create comprehensive tests:

```go
// providers/custom_provider_test.go
func TestCustomProvider_SendMessages(t *testing.T) {
    // Test message sending
}

func TestCustomProvider_StreamResponse(t *testing.T) {
    // Test streaming functionality
}

func TestCustomProvider_ErrorHandling(t *testing.T) {
    // Test error scenarios
}
```

### Step 6: Update Documentation

Add provider documentation to user manual and API reference.

---

## Testing Strategy

### Test Types

1. **Unit Tests**: Individual function/component testing
2. **Integration Tests**: Component interaction testing
3. **End-to-End Tests**: Complete workflow testing
4. **Performance Tests**: Load and performance benchmarking
5. **Security Tests**: Vulnerability and sanitization testing

### Test Organization

```
tests/
├── unit/              # Unit tests (90%+ coverage)
├── integration/       # Integration tests
├── e2e/              # End-to-end tests
├── performance/      # Performance benchmarks
├── security/         # Security validation
└── compatibility/    # Cross-platform testing
```

### Running Tests

**Full test suite:**
```bash
go test ./... -v -race -cover
```

**Specific test categories:**
```bash
# Unit tests only
go test ./llmverifier -v -short

# Integration tests
go test ./tests/integration -v

# Performance benchmarks
go test -bench=. -benchmem ./...

# Coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Writing Tests

**Basic test structure:**
```go
func TestFunctionName(t *testing.T) {
    // Arrange
    setupTestData()

    // Act
    result, err := functionUnderTest(input)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, expectedResult, result)
}
```

**Table-driven tests:**
```go
func TestFunctionName(t *testing.T) {
    testCases := []struct {
        name     string
        input    TestInput
        expected TestOutput
    }{
        {"case1", input1, expected1},
        {"case2", input2, expected2},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result := functionUnderTest(tc.input)
            assert.Equal(t, tc.expected, result)
        })
    }
}
```

---

## API Reference

### Core Interfaces

#### Provider Interface
```go
type Provider interface {
    SendMessages(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*ProviderResponse, error)
    StreamResponse(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent
    Model() models.Model
}
```

#### ModelVerifier Interface
```go
type ModelVerifier interface {
    VerifyModel(ctx context.Context, model models.Model) (*VerificationResult, error)
    VerifyMultipleModels(ctx context.Context, models []models.Model) ([]VerificationResult, error)
    GetVerificationHistory(modelID string) ([]VerificationResult, error)
}
```

### Configuration Structures

#### ExportOptions
```go
type ExportOptions struct {
    IncludeAPIKey    bool     // Include API keys in export
    MinScore         float64  // Minimum score threshold
    Providers        []string // Specific providers to include
    OutputFormat     string   // Export format
    Compression      bool     // Compress output
}
```

#### VerificationResult
```go
type VerificationResult struct {
    ModelInfo         ModelInfo       `json:"model_info"`
    PerformanceScores PerformanceScore `json:"performance_scores"`
    Error             string           `json:"error,omitempty"`
    Timestamp         time.Time        `json:"timestamp"`
}
```

### Error Handling

LLM Verifier uses structured error handling:

```go
// Custom error types
type VerificationError struct {
    ModelID   string
    Provider  string
    ErrorType string
    Message   string
}

func (e *VerificationError) Error() string {
    return fmt.Sprintf("[%s] %s: %s", e.Provider, e.ModelID, e.Message)
}

// Error wrapping
if err := verifyModel(model); err != nil {
    return fmt.Errorf("model verification failed for %s: %w", model.ID, err)
}
```

---

## Performance Optimization

### Profiling

**CPU profiling:**
```bash
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof
```

**Memory profiling:**
```bash
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof
```

### Optimization Techniques

1. **Connection Pooling**: Reuse HTTP connections
2. **Request Batching**: Group multiple requests
3. **Caching**: Cache verification results
4. **Parallel Processing**: Concurrent verification
5. **Resource Limits**: Control memory and CPU usage

### Benchmarking

**Performance benchmarks:**
```go
func BenchmarkVerification(b *testing.B) {
    for i := 0; i < b.N; i++ {
        verifyModel(testModel)
    }
}
```

**Load testing:**
```go
func BenchmarkConcurrentVerification(b *testing.B) {
    // Test concurrent model verification
    sem := make(chan struct{}, 10) // Limit concurrency
    // ... benchmark implementation
}
```

---

## Security Implementation

### Input Validation

All inputs are validated and sanitized:

```go
func validateInput(input, inputType string) bool {
    switch inputType {
    case "model_id":
        return validateModelID(input)
    case "api_key":
        return validateAPIKey(input)
    case "endpoint":
        return validateEndpoint(input)
    default:
        return false
    }
}
```

### Secret Management

API keys and sensitive data are handled securely:

```go
// Mask sensitive data in logs
func maskAPIKey(apiKey string) string {
    if len(apiKey) <= 8 {
        return "***"
    }
    return apiKey[:4] + "***" + apiKey[len(apiKey)-4:]
}

// Secure configuration storage
func saveSecureConfig(config map[string]interface{}, filePath string) error {
    // Encrypt sensitive fields
    encrypted := encryptSensitiveFields(config)

    // Save with restrictive permissions
    return saveWithPermissions(encrypted, filePath, 0600)
}
```

### Rate Limiting

Prevent API abuse:

```go
type RateLimiter struct {
    requests map[string]*time.Ticker
    limits   map[string]int
}

func (rl *RateLimiter) Allow(provider string) bool {
    limit := rl.limits[provider]
    // Rate limiting logic
}
```

---

## Deployment and Operations

### Container Deployment

**Dockerfile:**
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o llm-verifier ./cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/llm-verifier /usr/local/bin/
EXPOSE 8080
CMD ["llm-verifier", "serve"]
```

**Docker Compose:**
```yaml
version: '3.8'
services:
  llm-verifier:
    build: .
    ports:
      - "8080:8080"
    environment:
      - LLM_VERIFIER_DATABASE_URL=postgres://...
    volumes:
      - ./config:/app/config
```

### Kubernetes Deployment

**Deployment manifest:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: llm-verifier
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: llm-verifier
        image: your-org/llm-verifier:latest
        ports:
        - containerPort: 8080
        env:
        - name: LLM_VERIFIER_DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: llm-verifier-secrets
              key: database-url
```

### Monitoring and Observability

**Metrics collection:**
```go
// Prometheus metrics
var (
    verificationDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "llm_verifier_duration_seconds",
            Help: "Time taken for model verification",
        },
        []string{"provider", "model"},
    )

    verificationErrors = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "llm_verifier_errors_total",
            Help: "Total number of verification errors",
        },
        []string{"provider", "error_type"},
    )
)
```

**Logging:**
```go
// Structured logging
logger := logrus.New()
logger.SetFormatter(&logrus.JSONFormatter{})
logger.WithFields(logrus.Fields{
    "provider": providerName,
    "model": modelID,
    "duration": duration,
}).Info("Model verification completed")
```

---

## Contributing Guidelines

### Code Standards

1. **Go Style**: Follow standard Go formatting (`gofmt`)
2. **Documentation**: Document all public APIs
3. **Testing**: 90%+ test coverage for new code
4. **Error Handling**: Use error wrapping and structured errors
5. **Logging**: Use structured logging with appropriate levels

### Commit Messages

```
feat: add support for new AI provider
fix: resolve ProviderInitError in OpenCode configs
docs: update user manual with troubleshooting guide
test: add comprehensive integration tests
refactor: optimize model verification performance
```

### Pull Request Process

1. **Branch naming**: `feature/description` or `fix/issue-number`
2. **Tests**: All tests pass, new tests added
3. **Documentation**: Updated if needed
4. **Review**: At least one maintainer review
5. **Merge**: Squash merge with descriptive commit message

### Issue Reporting

**Bug reports should include:**
- Go version and OS
- Full error messages and stack traces
- Steps to reproduce
- Expected vs. actual behavior
- Configuration files (with sensitive data removed)

---

## Troubleshooting Development Issues

### Common Development Problems

#### 1. Build Failures
```bash
# Clean and rebuild
go clean -cache
go mod tidy
go build ./...
```

#### 2. Test Failures
```bash
# Run tests with verbose output
go test -v -run TestFailingTest

# Debug with race detector
go test -race -run TestFailingTest
```

#### 3. Dependency Issues
```bash
# Update dependencies
go get -u ./...

# Clean module cache
go clean -modcache
```

#### 4. Performance Issues
```bash
# Profile application
go tool pprof http://localhost:8080/debug/pprof/profile
```

---

## Roadmap and Future Development

### Planned Features

1. **Q4 2024**: Advanced model comparison tools
2. **Q1 2025**: Real-time performance monitoring dashboard
3. **Q2 2025**: Custom verification test frameworks
4. **Q3 2025**: Multi-cloud provider optimization
5. **Q4 2025**: AI-powered test case generation

### Technology Evolution

1. **Go 1.22+ Migration**: Utilize new language features
2. **Performance Optimizations**: Further reduce latency
3. **Security Enhancements**: Advanced threat detection
4. **Scalability Improvements**: Support for 1000+ concurrent verifications

---

This developer guide provides comprehensive information for contributing to and extending LLM Verifier. For user-facing documentation, see the User Manual.