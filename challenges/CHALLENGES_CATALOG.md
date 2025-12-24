# LLM Verifier - Challenges Catalog

**Last Updated**: 2025-12-24 21:00 UTC

---

## 📋 What Are Challenges?

Challenges are **test scenarios** that prove if the LLM Verifier platform works correctly. Each challenge tests specific functionality with **clear validation criteria** and produces **evidence** you can use to verify system behavior.

Think of challenges like **experiments** you can run to prove the platform does what it claims to do.

---

## 🎯 Why Use Challenges?

### For Developers
- **Verify Features Work**: After adding new code, run a challenge to prove it works
- **Find Bugs Early**: Challenges reveal issues before users encounter them
- **Test Edge Cases**: Explore what happens with unusual inputs or failures
- **Document Behavior**: Challenges create evidence that you can show stakeholders

### For Everyday Users (Like You)
- **Trust the Platform**: Verify all providers are actually working, not just configured
- **Compare Performance**: See which models are fastest, cheapest, or most reliable
- **Test Before Deployment**: Run challenges before critical production use
- **Troubleshoot Issues**: Use challenge results to diagnose why something isn't working

### Practical Everyday Examples

**Example 1: Choosing an LLM for Coding**
```
Your Problem: "Which LLM should I use for coding tasks?"

How to Use Challenges:
1. Run Model Discovery Challenge
   → Finds all available coding-capable models
2. Run Model Verification Challenge
   → Tests each model for actual API calls
3. Check Scoring Results
   → See which models have highest coding capability scores
4. Check Test Results
   → Verify which models pass actual coding tests

Result: "I'll use deepseek-chat - Score 95, verified for coding"
```

**Example 2: Diagnosing Why a Model Isn't Working**
```
Your Problem: "OpenAI GPT-4 isn't responding"

How to Use Challenges:
1. Run Model Existence Verification
   → Sends actual HTTP HEAD request to check if endpoint is reachable
2. Run Model Responsiveness Test
   → Makes actual HTTP POST request with test prompt
3. Check Challenge Logs
   → See HTTP status codes and error messages

Result: "Model exists but returning 503 (Service Unavailable)"
```

**Example 3: Testing New Provider Setup**
```
Your Problem: "I just added a new API key, does it work?"

How to Use Challenges:
1. Run Provider Discovery Challenge
   → Tests connection with your API key
2. Run Model Verification Challenge
   → Tests each discovered model with actual API calls
3. Review Test Results
   → See if any models returned errors

Result: "API key valid, but provider rate-limited model discovery"
```

---

## 📚 All Challenges

### ✅ Completed Challenges

| Challenge | Description | Everyday Use Case | Validation Criteria | Status |
|-----------|-------------|-------------------|--------|
| **Provider Models Discovery** | Tests provider APIs, discovers available models | Checks if API key works, lists models | ✅ Complete |
| **Model Verification** | Tests each model for existence, responsiveness, features | **Must make actual HTTP requests to verify models**, measure latency, test streaming, test function calling | ✅ **Needs Fix** |
| **CLI Platform Challenge** | Tests command-line interface commands | Commands execute correctly, help text works | ⏳ Documented |
| **TUI Platform Challenge** | Tests terminal user interface | Interface renders, navigation works, model selection | ⏳ Documented |
| **REST API Platform Challenge** | Tests HTTP API endpoints | API responds correctly, health checks work | ⏳ Documented |
| **Web Platform Challenge** | Tests web dashboard | Dashboard loads, displays metrics, controls work | ⏳ Documented |
| **Mobile Platform Challenge** | Tests mobile applications | App runs, receives notifications, offline mode | ⏳ Documented |
| **Desktop Platform Challenge** | Tests desktop applications | Native app launches, system integration | ⏳ Documented |
| **Scoring & Usability Challenge** | Tests model scoring and user experience | Models ranked by coding capability, recommendations work | ⏳ Documented |
| **Limits & Pricing Challenge** | Tests rate limits, costs, quotas | Limits detected, costs calculated | ⏳ Documented |
| **Database Challenge** | Tests data storage and retrieval | CRUD operations work, queries execute | ✅ 40% (Pricing CRUD added) |
| **Configuration Export Challenge** | Tests exporting to OpenCode/Crush formats | Configs generated, valid JSON | ✅ Complete |
| **Event System Challenge** | Tests event publishing and subscription | Events can be published, subscribers notified | ⏳ Documented |
| **Scheduling Challenge** | Tests task scheduling and automation | Scheduled tasks execute | ⏳ Documented |
| **Failover & Resilience Challenge** | Tests automatic fallback and error recovery | Failover switches work, downtime handled | ⏳ Documented |
| **Context Checkpointing Challenge** | Tests conversation state saving | Sessions can be saved, resumed | ⏳ Documented |
| **Monitoring & Observability Challenge** | Tests metrics collection and dashboards | Metrics collected, dashboards work | ⏳ Documented |
| **Security & Authentication Challenge** | Tests login, permissions, API key security | Auth works, keys encrypted | ⏳ Documented |

---

## 📋 Model Verification - Validation Criteria

### ✅ Required Tests

Each model MUST pass these validation checks to be considered "verified":

#### 1. Existence Test
**What It Does**: Verifies the model is actually available on the provider's API
**Validation Criteria**:
- HTTP HEAD request to model endpoint returns 200 OK
- OR HTTP GET to /models endpoint includes model in the list
- Response includes valid model_id and model_name
- **Pass**: Model exists and is accessible

**Current Implementation**: ❌ **FAILS** - Only checks configuration file, no real API calls

#### 2. Responsiveness Test
**What It Does**: Measures if the model responds to requests within acceptable time limits
**Validation Criteria**:
- HTTP POST request with test prompt completes successfully
- Time to First Token (TTFT) < 10 seconds
- Total response time < 60 seconds
- No timeout errors
- **Pass**: Model responds reliably

**Current Implementation**: ❌ **FAILS** - No actual HTTP requests made

#### 3. Latency Measurement
**What It Does**: Measures actual response times for performance tracking
**Validation Criteria**:
- TTFT (Time to First Token) is measured and recorded
- Total response time is measured and recorded
- Average latency calculated from multiple requests
- **Pass**: Latency data collected

**Current Implementation**: ❌ **FAILS** - No actual HTTP requests made

#### 4. Feature Testing
**What It Does**: Validates that claimed features actually work
**Validation Criteria**:
- **Streaming**: At least 1 chunk received
- **Function Calling**: Tool call is successfully parsed and executed
- **Vision**: Image input is accepted and processed
- **Embeddings**: Text embedding vector is returned
- **Pass**: Feature is tested and works

**Current Implementation**: ❌ **FAILS** - Only checks configuration file flags

#### 5. Scoring & Coding Capability
**What It Does**: Evaluates model's effectiveness for development tasks
**Validation Criteria**:
- **Score Value**: Integer 0-100 (higher = better for coding)
- **Scoring Method**: Tests actual model performance on coding benchmarks
- **Coding Capability Categories**:
  - Fully Coding Capable (score 80-100): Excellent for production code
  - Coding with Tools (score 60-79): Good for automation
  - Chat with Tooling (score 40-59): Basic code assistance
  - Chat Only (score 0-39): Not suitable for coding
- **Evidence**: Actual test results from coding benchmarks
- **Pass**: Model has verified score with supporting evidence

**Current Implementation**: ❌ **FAILS** - No scoring system implemented

#### 6. Error Detection
**What It Does**: Identifies and classifies API errors
**Validation Criteria**:
- HTTP 4xx errors are detected and logged
- HTTP 429 (rate limits) is detected
- HTTP 401 (unauthorized) is detected
- HTTP 404 (not found) is detected
- Timeouts are detected
- Connection errors are detected
- **Pass**: Errors are caught and reported with details

**Current Implementation**: ❌ **FAILS** - No error detection

---

## 📖 Model Verification - Required Fixes

### High Priority - Add Real API Testing

**Problem**: Current implementation only checks configuration files, no actual API calls

**Required Changes**:

1. **Add HTTP Client** (`llm-verifier/client/http_client.go`):
```go
type HTTPClient struct {
    client  *http.Client
    timeout time.Duration
}

func NewHTTPClient(timeout time.Duration) *HTTPClient {
    return &HTTPClient{
        client: &http.Client{Timeout: timeout},
        timeout: timeout,
    }
}
```

2. **Make Real API Requests** (`llm-verifier/challenges/codebase/go_files/run_model_verification.go`):
```go
// Test model existence
func testModelExists(client *HTTPClient, provider string, apiKey string, modelID string) error {
    endpoint := getEndpoint(provider, modelID)
    req, _ := http.NewRequest("HEAD", endpoint, nil)
    req.Header.Set("Authorization", "Bearer " + apiKey)
    
    resp, err := client.client.Do(req)
    if err != nil {
        return err
    }
    
    if resp.StatusCode == 200 {
        return nil
    }
    return fmt.Errorf("model not found (status %d)", resp.StatusCode)
}

// Test responsiveness
func testResponsiveness(client *HTTPClient, provider string, apiKey string, modelID string) (time.Duration, error) {
    endpoint := getEndpoint(provider, modelID)
    req, _ := http.NewRequest("POST", endpoint, nil)
    req.Header.Set("Authorization", "Bearer " + apiKey)
    req.Header.Set("Content-Type", "application/json")
    req.Body = strings.NewReader(`{"prompt": "test"}`)
    
    start := time.Now()
    resp, err := client.client.Do(req)
    if err != nil {
        return time.Duration(0), err
    }
    
    return time.Since(start), nil
}

// Test streaming
func testStreaming(client *HTTPClient, provider string, apiKey string, modelID string) (bool, error) {
    // Actual streaming test with chunked response
    // Returns true if at least one chunk received
    
    return false, fmt.Errorf("streaming test not implemented")
}
```

3. **Update Database Schema** (`llm-verifier/database/schema.sql`):
```sql
-- Add verification results table
CREATE TABLE verification_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    model_id TEXT NOT NULL,
    provider_name TEXT NOT NULL,
    test_type TEXT NOT NULL, -- 'existence', 'responsiveness', 'latency', 'feature_test'
    status TEXT NOT NULL, -- 'passed', 'failed'
    status_code INTEGER,
    http_status INTEGER,
    latency_ms INTEGER,
    response_time_ms INTEGER,
    test_timestamp DATETIME,
    error_message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Add verification scores
CREATE TABLE verification_scores (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    model_id TEXT NOT NULL,
    provider_name TEXT NOT NULL,
    score INTEGER NOT NULL CHECK (score >= 0 AND score <= 100),
    scoring_method TEXT, -- 'manual', 'benchmark', 'auto'
    category TEXT, -- 'fully_coding_capable', 'coding_with_tools', 'chat_with_tooling', 'chat_only'
    coding_benchmark_score INTEGER,
    evidence TEXT, -- JSON with test results
    scored_at DATETIME,
    updated_at DATETIME
);
```

4. **Add Scoring Logic** (`llm-verifier/scoring/scoring.go`):
```go
package scoring

type CodingCapability int

const (
    NotSuitable CodingCapability = iota
    ChatOnly
    ChatWithTooling
    CodingWithTools
    FullyCodingCapable
)

func CalculateCodingScore(benchmarkResults []BenchmarkResult) int {
    // Score based on coding benchmark performance
    // Weight factors:
    // - Code correctness: 40%
    // - Code quality: 30%
    // - Code speed: 20%
    // - Error handling: 10%
    
    totalScore := 0
    for _, result := range benchmarkResults {
        totalScore += result.CodeCorrectness * 40
        totalScore += result.CodeQuality * 30
        totalScore += result.CodeSpeed * 20
        totalScore += result.ErrorHandling * 10
    }
    
    return totalScore / len(benchmarkResults)
}

func ClassifyCodingCapability(score int) CodingCapability {
    switch {
    case score >= 80: return FullyCodingCapable
    case score >= 60: return CodingWithTools
    case score >= 40: return ChatWithTooling
    default: return ChatOnly
    }
}
```

---

## 🚀 Quick Start for Model Verification

### Step 1: Test Model Existence
```bash
# Test if a model exists
go run llm-verifier/challenges/codebase/go_files/run_model_verification.go --test-type=existence
```

### Step 2: Test Responsiveness
```bash
# Measure response time
go run llm-verifier/challenges/codebase/go_files/run_model_verification.go --test-type=responsiveness
```

### Step 3: Verify Results
```bash
# Check test results
cat challenges/model_verification/*/results/verification_results.json
```

---

## 📊 Progress Tracking

| Area | Completion | Next Steps |
|-------|------------|-------------|
| Model Discovery | 100% ✅ | None |
| Model Verification | 30% 🟡 | Add real API testing, scoring system, update verification results to database |
| Configuration Export | 100% ✅ | None |
| Core Platforms | 100% ✅ | All interfaces implemented |
| Database | 60% 🟡 | Add verification_results table, implement scoring system |
| Multi-Client Architecture | 25% 🟠 | Add authentication, usage tracking |
| Event System | 0% 🔴 | Design schema, implement publishing |
| Test Framework | 60% 🟡 | Add integration tests, security tests |
| Failover & Resilience | 100% ✅ | Complete |
| Context Checkpointing | 0% 🔴 | Not started |
| Monitoring & Observability | 100% ✅ | Complete |
| Security & Authentication | 0% 🔴 | Not started |
| Scheduling | 0% 🔴 | Not started |

---

## 💡 Tips for Using Challenges Effectively

### 1. Always Start with Real Tests
- Don't assume configuration is correct
- Make actual API calls to verify
- Collect real evidence (HTTP status codes, latency data)

### 2. Interpret Results Carefully
- A "passed" verification with config-only checking means nothing
- Look at latency data to understand real performance
- Check error messages to understand failures

### 3. Run Tests Regularly
```bash
# Schedule daily verification checks
0 3 * * * * cd /path/to/LLMVerifier && go run challenges/run_model_verification.go
```

### 4. Keep Test History
```bash
# Archive old results
tar -czf challenges-backup-$(date +%Y%m%d).tar.gz challenges/*/
```

---

## 📖 References

- [Model Verification Challenge Docs](challenges/docs/model_verification_challenge.md)
- [Complete System Documentation](llm-verifier/docs/COMPLETE_SYSTEM_DOCUMENTATION.md)
- [API Documentation](llm-verifier/docs/API_DOCUMENTATION.md)
- [SPECIFICATION.md](SPEIFICATION.md)
- [OPTIMIZATIONS.md](OPTIMIZATIONS.md)

---

**Last Updated**: 2025-12-24 21:00 UTC
**Total Challenges**: 17 (1 complete, 16 pending/in-progress)
**Total Scenarios**: 170+ test cases documented
