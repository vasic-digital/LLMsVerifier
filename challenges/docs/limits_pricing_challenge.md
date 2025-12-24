# Limits and Pricing Comprehensive Challenge

## Overview
This challenge validates the system's ability to detect and report rate limits, quotas, remaining limits, and pricing information for all models.

## Challenge Type
Functional Test + API Integration Test + Data Validation Test

## Test Scenarios

### 1. Rate Limits Detection Challenge
**Objective**: Verify system can detect rate limits

**Types of Limits**:
- Requests Per Minute (RPM)
- Requests Per Day (RPD)
- Requests Per Week (RPW)
- Requests Per Month (RPMonth)
- Tokens Per Minute (TPM)
- Concurrent requests

**Steps**:
1. Send requests at increasing rates
2. Monitor for rate limit errors
3. Parse rate limit headers
4. Determine exact limits

**Expected Results**:
- Rate limits are detected accurately
- Different limit types are identified
- Headers are parsed correctly

**Test Code**:
```go
func TestRateLimitDetection(t *testing.T) {
    detector := NewLimitDetector()

    limits, err := detector.DetectRateLimits("openai", "gpt-4")
    assert.NoError(t, err)

    assert.Greater(t, limits.RPM, 0)
    assert.Greater(t, limits.TPM, 0)
    assert.Greater(t, limits.RPD, 0)
}
```

---

### 2. Remaining Limits Calculation Challenge
**Objective**: Verify system tracks remaining limits

**Steps**:
1. Get total limits from provider
2. Track usage
3. Calculate remaining
4. Verify accuracy

**Expected Results**:
- Remaining limits are accurate
- Real-time tracking works
- Resets are handled correctly

**Test Code**:
```go
func TestRemainingLimits(t *testing.T) {
    tracker := NewLimitTracker()

    // Set initial limits
    tracker.SetLimits("openai", "gpt-4", RateLimits{
        RPM: 100,
        TPM: 100000,
        RPD: 10000,
    })

    // Record usage
    tracker.RecordUsage("openai", "gpt-4", Usage{
        Requests: 10,
        Tokens:   5000,
    })

    remaining := tracker.GetRemaining("openai", "gpt-4")
    assert.Equal(t, 90, remaining.RPM)
    assert.Equal(t, 95000, remaining.TPM)
    assert.Equal(t, 9990, remaining.RPD)
}
```

---

### 3. Quota Reset Detection Challenge
**Objective**: Verify system detects when quotas reset

**Steps**:
1. Track quota usage
2. Monitor for quota reset
3. Handle time zone differences
4. Verify reset timing

**Expected Results**:
- Resets detected at correct time
- Time zones handled correctly
- Usage resets accurately

**Test Code**:
```go
func TestQuotaReset(t *testing.T) {
    tracker := NewLimitTracker()

    tracker.SetResetSchedule("openai", "gpt-4", ResetSchedule{
        RPM:  "@every 1m",
        RPD:  "@daily",
        TPM:  "@every 1m",
    })

    // Simulate time passing
    tracker.AdvanceTime(1 * time.Minute)

    assert.Equal(t, 100, tracker.GetRemaining("openai", "gpt-4").RPM)
}
```

---

### 4. Pricing Detection Challenge
**Objective**: Verify system fetches pricing information

**Pricing Types**:
- Input tokens per 1M
- Output tokens per 1M
- Per request pricing
- Tiered pricing
- Per-second pricing

**Steps**:
1. Query provider pricing API
2. Parse pricing data
3. Store pricing information
4. Verify accuracy

**Expected Results**:
- Pricing is fetched correctly
- Different pricing models handled
- Currency conversion works

**Test Code**:
```go
func TestPricingDetection(t *testing.T) {
    detector := NewPricingDetector()

    pricing, err := detector.DetectPricing("openai", "gpt-4")
    assert.NoError(t, err)

    assert.Greater(t, pricing.InputTokensPer1M, 0.0)
    assert.Greater(t, pricing.OutputTokensPer1M, 0.0)
    assert.Equal(t, "USD", pricing.Currency)
}
```

---

### 5. Cost Estimation Challenge
**Objective**: Verify system can estimate costs

**Steps**:
1. Provide usage metrics
2. Calculate cost
3. Compare across models
4. Verify accuracy

**Expected Results**:
- Cost estimates are accurate
- Multiple pricing models handled
- Estimates are in correct currency

**Test Code**:
```go
func TestCostEstimation(t *testing.T) {
    estimator := NewCostEstimator()

    pricing := ModelPricing{
        InputTokensPer1M:  30.0,
        OutputTokensPer1M: 60.0,
    }

    usage := Usage{
        InputTokens:  1000,
        OutputTokens: 500,
    }

    cost := estimator.EstimateCost(pricing, usage)
    expected := (1000/1000000.0)*30.0 + (500/1000000.0)*60.0
    assert.InDelta(t, expected, cost, 0.0001)
}
```

---

### 6. Provider-Specific Limit Handling Challenge
**Objective**: Verify different provider limit formats

**Providers**:
- OpenAI: Headers with rate limit info
- Anthropic: Rate limits in response
- AWS Bedrock: Quotas via API
- Google: Different limits per tier

**Steps**:
1. Test OpenAI limits
2. Test Anthropic limits
3. Test AWS Bedrock limits
4. Test Google limits

**Expected Results**:
- Each provider's format handled
- Limits extracted correctly
- Errors handled gracefully

**Test Code**:
```go
func TestProviderSpecificLimits(t *testing.T) {
    detector := NewLimitDetector()

    providers := []string{"openai", "anthropic", "aws-bedrock", "google"}

    for _, provider := range providers {
        limits, err := detector.DetectRateLimits(provider, "test-model")
        assert.NoError(t, err)
        assert.NotZero(t, limits.RPM)
    }
}
```

---

### 7. Limit Exceeded Handling Challenge
**Objective**: Verify system handles limit exceeded scenarios

**Steps**:
1. Exceed rate limit
2. Catch 429 errors
3. Implement backoff
4. Resume after reset

**Expected Results**:
- 429 errors caught
- Backoff implemented
- Requests resume after reset
- No requests lost

**Test Code**:
```go
func TestLimitExceededHandling(t *testing.T) {
    client := NewLLMVerifier(config)

    // Exceed limit
    for i := 0; i < 200; i++ {
        _, err := client.SendRequest("openai", "gpt-4", testPrompt)
        if err != nil && IsRateLimitError(err) {
            break
        }
    }

    // Should back off and wait
    time.Sleep(5 * time.Second)

    // Should resume
    resp, err := client.SendRequest("openai", "gpt-4", testPrompt)
    assert.NoError(t, err)
    assert.NotNil(t, resp)
}
```

---

### 8. Pricing Comparison Challenge
**Objective**: Verify pricing can be compared across models

**Steps**:
1. Get pricing for multiple models
2. Compare costs
3. Rank by cost
4. Identify best value

**Expected Results**:
- Pricing comparison works
- Cost ranking accurate
- Best value identified

**Test Code**:
```go
func TestPricingComparison(t *testing.T) {
    comparator := NewPricingComparator()

    pricing := map[string]ModelPricing{
        "gpt-4":      {InputTokensPer1M: 30, OutputTokensPer1M: 60},
        "gpt-3.5":    {InputTokensPer1M: 0.5, OutputTokensPer1M: 1.5},
        "claude-3":    {InputTokensPer1M: 15, OutputTokensPer1M: 75},
    }

    ranked := comparator.RankByCost(pricing, Usage{InputTokens: 1000, OutputTokens: 500})
    assert.Equal(t, "gpt-3.5", ranked[0])
    assert.Equal(t, "claude-3", ranked[1])
    assert.Equal(t, "gpt-4", ranked[2])
}
```

---

### 9. Limits and Pricing Export Challenge
**Objective**: Verify limits and pricing can be exported

**Steps**:
1. Export limits to JSON
2. Export pricing to CSV
3. Include in reports
4. Verify format

**Expected Results**:
- Export formats are valid
- Data is complete
- Reports include information

**Test Code**:
```go
func TestLimitsPricingExport(t *testing.T) {
    exporter := NewLimitsExporter()

    limits := map[string]RateLimits{
        "gpt-4": {RPM: 100, TPM: 100000},
    }

    jsonExport := exporter.ExportJSON(limits)
    assert.JSONEq(t, `{"gpt-4":{"rpm":100,"tpm":100000}}`, jsonExport)
}
```

---

### 10. Real-Time Limit Monitoring Challenge
**Objective**: Verify limits are monitored in real-time

**Steps**:
1. Send requests continuously
2. Monitor limits in real-time
3. Trigger alerts near limits
4. Prevent exceeding limits

**Expected Results**:
- Real-time monitoring works
- Alerts triggered appropriately
- Limits not exceeded
- Requests throttled if needed

**Test Code**:
```go
func TestRealTimeLimitMonitoring(t *testing.T) {
    monitor := NewLimitMonitor()

    monitor.OnNearLimit(func(model string, remaining RateLimits) {
        if remaining.RPM < 10 {
            // Trigger alert
            fmt.Printf("ALERT: %s near limit (RPM: %d)\n", model, remaining.RPM)
        }
    })

    monitor.StartMonitoring("openai", "gpt-4")

    // Send requests
    for i := 0; i < 95; i++ {
        monitor.RecordUsage("openai", "gpt-4", Usage{Requests: 1})
    }

    // Should have triggered alert
}
```

---

## Success Criteria

### Functional Requirements
- [ ] Rate limits detected accurately
- [ ] Remaining limits calculated correctly
- [ ] Quota resets detected
- [ ] Pricing fetched correctly
- [ ] Costs estimated accurately
- [ ] Provider-specific formats handled
- [ ] Limit exceeded scenarios handled
- [ ] Pricing comparison works
- [ ] Exports work correctly
- [ ] Real-time monitoring works

### Accuracy Requirements
- [ ] Limit detection accuracy > 99%
- [ ] Cost estimation accuracy > 99%
- [ ] Reset timing accuracy within 1 second

### Performance Requirements
- [ ] Limit detection < 1 second
- [ ] Remaining calculation < 100ms
- [ ] Cost estimation < 10ms

## Dependencies
- Valid API keys
- Network access to provider APIs

## Cleanup
- No cleanup needed
