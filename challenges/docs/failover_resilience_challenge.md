# Failover and Resilience Comprehensive Challenge

## Overview
This challenge validates multi-provider failover, circuit breaker, latency-based routing, and health checking mechanisms from OPTIMIZATIONS.md.

## Challenge Type
Integration Test + Resilience Test + Failover Test

## Test Scenarios

### 1. Circuit Breaker Challenge
**Objective**: Verify circuit breaker pattern works

**States**:
- Closed (normal)
- Open (failed, blocking requests)
- Half-Open (testing if recovered)

**Steps**:
1. Send normal requests (circuit closed)
2. Induce 5 failures in 60 seconds
3. Verify circuit opens
4. Wait for timeout
5. Verify circuit goes to half-open
6. Test successful request
7. Verify circuit closes again

**Expected Results**:
- Circuit opens after threshold
- Requests blocked when open
- Half-open state after timeout
- Circuit closes on success

**Test Code**:
```go
func TestCircuitBreaker(t *testing.T) {
    cb := NewCircuitBreaker(CircuitBreakerConfig{
        FailureThreshold: 5,
        Timeout:        60 * time.Second,
    })

    // Normal requests pass
    err := cb.Execute(func() error { return nil })
    assert.NoError(t, err)

    // Induce failures
    for i := 0; i < 5; i++ {
        cb.Execute(func() error { return errors.New("failure") })
    }

    // Circuit should be open
    assert.True(t, cb.IsOpen())

    // Request should fail immediately
    err = cb.Execute(func() error { return nil })
    assert.Error(t, err)
    assert.Equal(t, ErrCircuitOpen, err)
}
```

---

### 2. Multi-Provider Failover Challenge
**Objective**: Verify failover across multiple providers

**Providers**: OpenAI, Anthropic, DeepSeek, SiliconFlow, NVIDIA

**Steps**:
1. Configure provider order with weights
2. Send request to primary provider
3. Simulate primary provider failure
4. Verify automatic failover to backup
5. Verify request succeeds with backup
6. Verify primary marked as degraded

**Expected Results**:
- Primary provider tried first
- Failure triggers failover
- Backup provider used
- Primary marked degraded
- Request completes successfully

**Test Code**:
```go
func TestMultiProviderFailover(t *testing.T) {
    failover := NewFailoverProvider([]ProviderConfig{
        {Name: "openai", Weight: 0.7, Endpoint: "openai-api"},
        {Name: "anthropic", Weight: 0.3, Endpoint: "anthropic-api"},
        {Name: "deepseek", Weight: 0.2, Endpoint: "deepseek-api"},
    })

    // Simulate OpenAI failure
    failover.SetProviderStatus("openai", StatusDegraded)

    resp, err := failover.Request(Request{Model: "gpt-4", Prompt: "test"})
    assert.NoError(t, err)
    assert.NotEqual(t, "openai", resp.ProviderUsed)
}
```

---

### 3. Latency-Based Routing Challenge
**Objective**: Verify routing based on TTFT (Time to First Token)

**Steps**:
1. Measure TTFT for each provider
2. Configure latency threshold (2 seconds)
3. Route based on latency
4. Verify fast providers preferred
5. Verify failover if slow

**Expected Results**:
- TTFT measured accurately
- Fast providers preferred
- Slow providers used only if fast fail
- Threshold enforced

**Test Code**:
```go
func TestLatencyBasedRouting(t *testing.T) {
    router := NewLatencyRouter(LatencyConfig{
        TTFTThreshold: 2 * time.Second,
    })

    // Record provider latencies
    router.RecordTTFT("openai", 1*time.Second)
    router.RecordTTFT("anthropic", 1.5*time.Second)
    router.RecordTTFT("deepseek", 3*time.Second)

    // Should route to fastest
    provider := router.SelectProvider([]string{"openai", "anthropic", "deepseek"})
    assert.Equal(t, "openai", provider)
}
```

---

### 4. Weighted Routing Challenge
**Objective**: Verify weighted routing (70% cost-effective, 30% premium)

**Steps**:
1. Configure provider weights
2. Send many requests
3. Verify distribution matches weights
4. Verify all providers used

**Expected Results**:
- Distribution matches weights
- All providers used
- Requests balanced

**Test Code**:
```go
func TestWeightedRouting(t *testing.T) {
    router := NewWeightedRouter(map[string]float64{
        "openai":      0.7,
        "anthropic":    0.3,
    })

    requests := 100
    counts := make(map[string]int)

    for i := 0; i < requests; i++ {
        provider := router.SelectProvider()
        counts[provider]++
    }

    // OpenAI should have ~70% of requests
    openaiPct := float64(counts["openai"]) / float64(requests)
    assert.InDelta(t, 0.7, openaiPct, 0.1)
}
```

---

### 5. Health Probe Challenge
**Objective**: Verify health checking

**Steps**:
1. Configure health probes
2. Send periodic health checks
3. Verify provider status updates
4. Verify unhealthy providers marked degraded

**Expected Results**:
- Health checks sent periodically
- Unhealthy providers marked
- Recovery detected
- Status updates

**Test Code**:
```go
func TestHealthProbe(t *testing.T) {
    monitor := NewHealthMonitor(HealthConfig{
        Interval: 30 * time.Second,
        Providers: []string{"openai", "anthropic"},
    })

    // Simulate OpenAI failure
    monitor.SetProviderHealth("openai", false)

    // Should be marked degraded
    status := monitor.GetStatus("openai")
    assert.Equal(t, StatusDegraded, status)
}
```

---

### 6. Provider Recovery Challenge
**Objective**: Verify providers can recover from degraded state

**Steps**:
1. Mark provider as degraded
2. Wait for recovery timeout
3. Perform health probe
4. Verify provider recovered
5. Restore normal routing

**Expected Results**:
- Degraded state cleared
- Health probe successful
- Provider back in rotation
- Normal routing resumes

**Test Code**:
```go
func TestProviderRecovery(t *testing.T) {
    cb := NewCircuitBreaker(CircuitBreakerConfig{
        Timeout: 60 * time.Second,
    })

    // Open circuit
    for i := 0; i < 5; i++ {
        cb.Execute(func() error { return errors.New("failure") })
    }

    assert.True(t, cb.IsOpen())

    // Wait for timeout
    time.Sleep(61 * time.Second)

    // Successful request should close circuit
    cb.Execute(func() error { return nil })

    assert.False(t, cb.IsOpen())
}
```

---

### 7. Exponential Backoff Challenge
**Objective**: Verify retry with exponential backoff

**Steps**:
1. Configure retry with backoff
2. Send request that fails
3. Verify retries occur
4. Verify backoff increases exponentially
5. Verify eventual success or fail

**Expected Results**:
- Retries occur
- Backoff increases (2s, 4s, 8s)
- Random jitter added
- Max backoff capped

**Test Code**:
```go
func TestExponentialBackoff(t *testing.T) {
    retry := NewRetryPolicy(RetryConfig{
        MaxAttempts:     4,
        InitialBackoff:  2 * time.Second,
        MaxBackoff:     16 * time.Second,
        Jitter:         true,
    })

    attempts := 0
    backoffs := []time.Duration{}

    err := retry.Execute(func() error {
        attempts++
        if attempts < 4 {
            return errors.New("transient error")
        }
        return nil
    }, &backoffs)

    assert.NoError(t, err)
    assert.Equal(t, 4, attempts)
    assert.GreaterOrEqual(t, backoffs[1], backoffs[0]*2)
}
```

---

### 8. Timeout Handling Challenge
**Objective**: Verify timeout handling for different providers

**Provider Timeouts**:
- DeepSeek: 30-minute server timeout
- OpenAI: Standard timeout
- Anthropic: Standard timeout

**Steps**:
1. Configure per-provider timeouts
2. Send long-running request
3. Verify timeout enforced
4. Verify proper error message
5. Verify no partial responses

**Expected Results**:
- Timeouts enforced
- Proper error messages
- No partial responses
- Connection properly closed

**Test Code**:
```go
func TestTimeoutHandling(t *testing.T) {
    client := NewLLMClient(TimeoutConfig{
        Default:    60 * time.Second,
        PerProvider: map[string]time.Duration{
            "deepseek": 30 * time.Minute,
        },
    })

    // Should use deepseek timeout
    resp, err := client.Request("deepseek", "deepseek-chat", longPrompt)
    assert.Error(t, err)
    assert.Equal(t, ErrTimeout, err)
}
```

---

### 9. Concurrent Failover Challenge
**Objective**: Verify failover works under concurrent load

**Steps**:
1. Send 100 concurrent requests
2. Failover primary provider mid-stream
3. Verify all requests succeed
4. Verify backup providers used
5. Verify no requests lost

**Expected Results**:
- All requests succeed
- Failover handled
- Backup providers used
- No requests lost
- No race conditions

**Test Code**:
```go
func TestConcurrentFailover(t *testing.T) {
    failover := NewFailoverProvider(providers)

    results := make(chan error, 100)

    for i := 0; i < 100; i++ {
        go func(id int) {
            _, err := failover.Request(Request{ID: id})
            results <- err
        }(i)
    }

    // Failover primary
    time.Sleep(10 * time.Millisecond)
    failover.SetProviderStatus("openai", StatusDegraded)

    // Collect results
    successCount := 0
    for i := 0; i < 100; i++ {
        err := <-results
        if err == nil {
            successCount++
        }
    }

    assert.Equal(t, 100, successCount)
}
```

---

### 10. State Persistence Challenge
**Objective**: Verify failover state persists across restarts

**Steps**:
1. Create failover state
2. Save state to file
3. Simulate restart
4. Load state from file
5. Verify state matches

**Expected Results**:
- State saved
- State loaded
- State matches
- Degraded status persists

**Test Code**:
```go
func TestStatePersistence(t *testing.T) {
    failover := NewFailoverProvider(providers)
    failover.SetProviderStatus("openai", StatusDegraded)

    // Save state
    err := failover.SaveState("failover_state.json")
    assert.NoError(t, err)

    // Simulate restart
    failover2 := NewFailoverProvider(providers)
    err = failover2.LoadState("failover_state.json")
    assert.NoError(t, err)

    // Status should persist
    status := failover2.GetStatus("openai")
    assert.Equal(t, StatusDegraded, status)
}
```

---

## Success Criteria

### Functional Requirements
- [ ] Circuit breaker works
- [ ] Multi-provider failover works
- [ ] Latency-based routing works
- [ ] Weighted routing works
- [ ] Health probes work
- [ ] Provider recovery works
- [ ] Exponential backoff works
- [ ] Timeout handling works
- [ ] Concurrent failover works
- [ ] State persists

### Reliability Requirements
- [ ] Failover time < 2 seconds
- [ ] Circuit breaker opens within threshold
- [ ] Recovery within timeout window
- [ ] No requests lost during failover

### Performance Requirements
- [ ] Health check < 1 second
- [ ] Routing decision < 10ms
- [ ] Backoff calculation < 1ms

## Dependencies
- Multiple provider API keys
- Failover service running

## Cleanup
- Remove state files
- Clear circuit breaker state
