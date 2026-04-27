# CLAUDE.md - LLM Verifier


## Definition of Done

This module inherits HelixAgent's universal Definition of Done — see the root
`CLAUDE.md` and `docs/development/definition-of-done.md`. In one line: **no
task is done without pasted output from a real run of the real system in the
same session as the change.** Coverage and green suites are not evidence.

### Acceptance demo for this module

```bash
# Provider adapters (OpenAI/Anthropic/Gemini/Groq/…) against their real APIs
cd LLMsVerifier/llm-verifier && GOMAXPROCS=2 nice -n 19 go test -count=1 -race -v \
  -run 'TestProvider' ./providers/...
```
Expect: PASS for adapters whose API key is set in the environment; `ProviderAdapter.DiscoverModels/TestModel/SupportsFeature` exercised against live endpoints. Adapters without keys skip cleanly.


## Module Overview

The LLM Verifier is a comprehensive tool for verifying, testing, and benchmarking Large Language Models based on their coding capabilities and feature support. It serves as the quality assurance and ranking system for HelixAgent's provider selection.

## Architecture

### Core Purpose

The LLM Verifier answers the question: "Which LLM provider should I use for this task?"

It does this by:
1. **Discovering** all available models from configured providers
2. **Testing** each model's actual capabilities (not just claimed features)
3. **Scoring** models across multiple dimensions
4. **Ranking** providers by overall quality and specific use cases
5. **Exporting** configurations for CLI agents

### System Context

```
┌─────────────────────────────────────────────────────────┐
│                    LLM Verifier                         │
├─────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │  Provider    │  │   Feature    │  │   Scoring    │  │
│  │  Discovery   │→ │   Testing    │→ │   Engine     │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
│         ↓                   ↓                ↓          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │   SQLite     │  │   Reports    │  │   Config     │  │
│  │   Database   │  │  (MD/JSON)   │  │   Exports    │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────┘
                           ↓
                    HelixAgent
               (Uses scores for
                provider selection)
```

## Code Organization

### Directory Structure

```
llm-verifier/
├── cmd/
│   └── main.go                 # CLI entry point
├── llmverifier/
│   ├── verifier.go             # Core verification logic
│   └── llm_client.go           # LLM API client
├── database/
│   ├── database.go             # DB initialization
│   └── crud.go                 # CRUD operations
├── tests/
│   ├── test_helpers.go         # Test utilities
│   ├── unit_test.go
│   ├── integration_test.go
│   ├── e2e_test.go
│   ├── performance_test.go
│   ├── security_test.go
│   └── automation_test.go
├── api/
│   └── server.go               # REST API server
├── ai/
│   └── scoring.go              # AI-assisted scoring
├── analytics/
│   └── reports.go              # Report generation
├── auth/
│   └── jwt.go                  # Authentication
├── challenges/
│   └── runner.go               # Challenge execution
├── client/
│   └── clients.go              # Multi-platform clients
├── cmd/
│   └── commands.go             # CLI commands
├── config/
│   └── config.go               # Configuration
└── docs/
    ├── API_DOCUMENTATION.md
    ├── SPECIFICATION.md
    └── IMPLEMENTATION_ROADMAP.md
```

### Key Components

**1. Provider Discovery (llmverifier/verifier.go)**

Discovers all models from API endpoints:

```go
func (v *Verifier) DiscoverProviders() ([]Provider, error) {
    // Query each provider's /v1/models endpoint
    // Parse model metadata
    // Store in database
}
```

**2. Feature Testing (llmverifier/verifier.go)**

Tests actual capabilities by making real API calls:

```go
func (v *Verifier) TestFeature(model Model, feature Feature) (bool, error) {
    // Send test request
    // Verify response format
    // Check for expected behavior
}
```

**3. Scoring Engine (ai/scoring.go)**

Calculates weighted scores:

```go
func CalculateScore(results TestResults) Score {
    codeCapability := testCodeCapability(results) * 0.40
    responsiveness := testResponsiveness(results) * 0.15
    reliability := testReliability(results) * 0.15
    featureRichness := testFeatureRichness(results) * 0.20
    valueProp := testValueProposition(results) * 0.10
    
    return codeCapability + responsiveness + reliability + 
           featureRichness + valueProp
}
```

**4. Database Layer (database/)**

SQLite with SQL Cipher encryption:

```go
type Database struct {
    db *sql.DB
}

// CRUD operations for providers, models, results
func (d *Database) CreateProvider(p Provider) error
func (d *Database) GetProvider(id string) (Provider, error)
func (d *Database) CreateVerificationResult(r Result) error
```

## Design Patterns

### Provider Adapter Pattern

Each LLM provider implements a common interface:

```go
type ProviderAdapter interface {
    DiscoverModels() ([]Model, error)
    TestModel(model Model) (TestResult, error)
    SupportsFeature(feature Feature) bool
}
```

Adapters exist for:
- OpenAI
- Anthropic
- DeepSeek
- Groq
- Together AI
- Mistral
- xAI
- Replicate
- Cohere
- Cerebras
- Cloudflare Workers AI
- SiliconFlow

### Feature Detection Strategy

Instead of trusting documentation, features are actively tested:

```go
var featureTests = map[Feature]TestFunc{
    FeatureToolCalling: testToolCalling,
    FeatureStreaming:   testStreaming,
    FeatureEmbeddings:  testEmbeddings,
    FeatureVision:      testVision,
    // ... etc
}

func testToolCalling(client LLMClient) bool {
    // Send request with tool definitions
    // Verify model calls tool correctly
    // Return true if successful
}
```

### Scoring Algorithm

Multi-dimensional weighted scoring:

| Dimension | Weight | Test Method |
|-----------|--------|-------------|
| Code Capability | 40% | Multi-language coding tasks |
| Responsiveness | 15% | Response time measurements |
| Reliability | 15% | Success rate under load |
| Feature Richness | 20% | Count of supported features |
| Value Proposition | 10% | Cost/performance ratio |

### Export Strategy

Multiple export formats for different clients:

```go
type ConfigExporter interface {
    Export(models []Model) (string, error)
}

type OpenCodeExporter struct{}
type ClaudeExporter struct{}
type VSCodeExporter struct{}
```

## Implementation Details

### Database Schema

**providers table:**
```sql
CREATE TABLE providers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    endpoint TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**models table:**
```sql
CREATE TABLE models (
    id TEXT PRIMARY KEY,
    provider_id TEXT REFERENCES providers(id),
    name TEXT NOT NULL,
    model_identifier TEXT NOT NULL,
    capabilities JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**verification_results table:**
```sql
CREATE TABLE verification_results (
    id TEXT PRIMARY KEY,
    model_id TEXT REFERENCES models(id),
    score REAL NOT NULL,
    score_details JSON,
    metadata JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Concurrency Model

Parallel testing of providers:

```go
func (v *Verifier) VerifyAll() error {
    var wg sync.WaitGroup
    semaphore := make(chan struct{}, v.concurrency)
    
    for _, provider := range v.providers {
        wg.Add(1)
        semaphore <- struct{}{}
        
        go func(p Provider) {
            defer wg.Done()
            defer func() { <-semaphore }()
            
            v.verifyProvider(p)
        }(provider)
    }
    
    wg.Wait()
    return nil
}
```

### Error Handling

Comprehensive error wrapping:

```go
func (v *Verifier) verifyModel(model Model) (Result, error) {
    result, err := v.tester.Test(model)
    if err != nil {
        return Result{}, fmt.Errorf("failed to test model %s: %w", model.ID, err)
    }
    
    if err := v.db.SaveResult(result); err != nil {
        return Result{}, fmt.Errorf("failed to save result for %s: %w", model.ID, err)
    }
    
    return result, nil
}
```

## Testing Strategy

### Test Coverage Requirements

- **Line Coverage**: 100%
- **Branch Coverage**: 95%
- **Function Coverage**: 100%

### Test Types

**Unit Tests:**
- Individual function testing
- Mock external dependencies
- Fast execution

**Integration Tests:**
- Database operations
- API endpoint testing
- Component interactions

**E2E Tests:**
- Full verification workflows
- Real API calls (optional)
- End-user scenarios

**Performance Tests:**
- Response time benchmarks
- Throughput testing
- Load testing

**Security Tests:**
- Input validation
- Authentication testing
- Data protection

**Automation Tests:**
- CLI command testing
- Script behavior
- Configuration parsing

## API Design

### REST Endpoints

**GET /models** - List all verified models
```json
{
  "success": true,
  "data": [
    {
      "id": "gpt-4",
      "name": "GPT-4",
      "provider": "openai",
      "score": 95.5,
      "capabilities": ["code", "tools", "streaming"]
    }
  ]
}
```

**POST /verify** - Trigger verification
```json
{
  "provider": "openai",
  "model": "gpt-4",
  "tests": ["code", "tools", "streaming"]
}
```

**GET /rankings** - Get provider rankings
```json
{
  "rankings": [
    {"provider": "openai", "avg_score": 94.2, "rank": 1},
    {"provider": "anthropic", "avg_score": 93.8, "rank": 2}
  ]
}
```

## Configuration

### config.yaml Structure

```yaml
global:
  base_url: "${OPENAI_API_KEY}"
  api_key: "${API_KEY}"
  max_retries: 3
  timeout: 30s

llms:
  - name: "OpenAI GPT-4"
    endpoint: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4-turbo"

concurrency: 5
timeout: 60s
```

### Environment Variables

- `OPENAI_API_KEY`
- `ANTHROPIC_API_KEY`
- `DEEPSEEK_API_KEY`
- `GROQ_API_KEY`
- And others for each provider

## Security Considerations

### Data Protection

- **Database Encryption**: SQL Cipher encrypts all data at rest
- **API Key Security**: Keys stored in environment variables only
- **No Logging**: API keys never logged
- **Secure Exports**: Exported configs protected with 600 permissions

### Input Validation

All inputs validated before processing:
- API key format validation
- URL validation for endpoints
- Timeout bounds checking
- Concurrency limits

## Performance Optimization

### Caching Strategy

Results cached to avoid redundant testing:
```go
type Cache struct {
    results map[string]VerificationResult
    mu      sync.RWMutex
}
```

### Rate Limiting

Respectful API usage:
```go
func (c *LLMClient) makeRequest(req Request) (*Response, error) {
    // Wait for rate limiter
    c.rateLimiter.Wait()
    
    // Make request
    resp, err := c.httpClient.Do(req)
    
    // Update rate limiter based on headers
    c.rateLimiter.Update(resp.Header)
    
    return resp, err
}
```

## Deployment

### Docker

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o llm-verifier ./cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/llm-verifier /usr/local/bin/
CMD ["llm-verifier"]
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: llm-verifier
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: llm-verifier
        image: helixagent/llm-verifier:latest
        env:
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: api-keys
              key: openai
```

## Monitoring

### Metrics

- Verification success rate
- Average response time per provider
- Feature detection accuracy
- Database query performance
- API error rates

### Health Checks

```go
func (s *Server) HealthCheck() HealthStatus {
    return HealthStatus{
        Database:   s.db.Ping() == nil,
        API:        s.apiClient.HealthCheck(),
        DiskSpace:  checkDiskSpace(),
        Timestamp:  time.Now(),
    }
}
```

## Related Documentation

- [API Documentation](docs/API_DOCUMENTATION.md)
- [Specification](docs/SPECIFICATION.md)
- [Implementation Roadmap](docs/IMPLEMENTATION_ROADMAP.md)
- [Security Guide](SECURITY_CONFIGURATION_EXPORT.md)

## Development Workflow

1. **Add Provider**: Implement ProviderAdapter interface
2. **Add Feature Test**: Create test function in featureTests map
3. **Update Scoring**: Adjust scoring weights if needed
4. **Add Tests**: Unit, integration, and e2e tests
5. **Update Docs**: Document new features and APIs
6. **Run Verification**: Test with real providers

## Maintenance

### Regular Tasks

- Update provider endpoints when changed
- Refresh model lists from providers
- Review and update feature tests
- Monitor verification success rates
- Update security patches

### Adding New Providers

1. Create adapter in `llmverifier/adapters/`
2. Implement ProviderAdapter interface
3. Add configuration in `config.yaml` template
4. Add tests
5. Document in README.md

## Future Enhancements

- **Real-time Monitoring**: Live dashboard of provider health
- **Predictive Scoring**: ML-based quality prediction
- **A/B Testing**: Compare provider performance
- **Custom Tests**: User-defined verification tests
- **Benchmark Suite**: Standardized coding challenges

## Integration Seams

| Direction | Sibling modules |
|-----------|-----------------|
| Upstream (this module imports) | Challenges (via parent) |
| Downstream (these import this module) | HelixQA (via parent) |

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
