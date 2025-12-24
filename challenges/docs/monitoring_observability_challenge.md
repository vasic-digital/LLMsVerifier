# Monitoring and Observability Comprehensive Challenge

## Overview
This challenge validates monitoring and observability stack including Prometheus metrics, Grafana dashboards, Jaeger tracing, and alerting.

## Challenge Type
Integration Test + Monitoring Test + Observability Test

## Test Scenarios

### 1. Prometheus Metrics Challenge
**Objective**: Verify Prometheus metrics are exposed correctly

**Metrics**:
- Time to First Token (TTFT) per provider
- Request duration per provider
- Error rates per provider
- Circuit breaker state changes
- Checkpoint success/failure rate
- Token consumption

**Steps**:
1. Query /metrics endpoint
2. Verify all metrics present
3. Verify metric formats
4. Verify labels
5. Verify values

**Expected Results**:
- All metrics exposed
- Correct Prometheus format
- Proper labels
- Values are valid

**Test Code**:
```go
func TestPrometheusMetrics(t *testing.T) {
    resp, err := http.Get("http://localhost:8080/metrics")
    assert.NoError(t, err)
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    metrics := string(body)

    // Verify key metrics
    assert.Contains(t, metrics, "llm_time_to_first_token_seconds")
    assert.Contains(t, metrics, "llm_request_duration_seconds")
    assert.Contains(t, metrics, "llm_request_errors_total")
    assert.Contains(t, metrics, "llm_circuit_breaker_state")
}
```

---

### 2. Grafana Dashboard Challenge
**Objective**: Verify Grafana dashboards display correctly

**Dashboards**:
- TTFT by Provider (p50, p95)
- Request Duration and Errors
- Last Checkpoint Age
- Provider Health Status

**Steps**:
1. Access Grafana
2. Open LLM Verifier dashboard
3. Verify all panels load
4. Verify data populates
5. Verify real-time updates

**Expected Results**:
- All panels display
- Data is accurate
- Real-time updates work
- Dashboard loads quickly

**Test Code**:
```go
func TestGrafanaDashboard(t *testing.T) {
    client := NewGrafanaClient("http://localhost:3000", "admin", "admin")

    dashboard, err := client.GetDashboard("llm-verifier")
    assert.NoError(t, err)

    // Verify panels exist
    panels := dashboard.Panels
    assert.Greater(t, len(panels), 0)

    // Verify key panels
    assert.True(t, hasPanel(panels, "TTFT by Provider"))
    assert.True(t, hasPanel(panels, "Request Duration"))
    assert.True(t, hasPanel(panels, "Circuit Breaker Status"))
}
```

---

### 3. Jaeger Distributed Tracing Challenge
**Objective**: Verify Jaeger tracing works

**Steps**:
1. Send request with trace ID
2. Query Jaeger for trace
3. Verify span data
4. Verify service graph
5. Verify parent-child relationships

**Expected Results**:
- Traces captured
- Spans detailed
- Service graph accurate
- Parent-child relationships correct

**Test Code**:
```go
func TestJaegerTracing(t *testing.T) {
    tracer := NewJaegerTracer("localhost:4318")

    ctx, span := tracer.StartSpan(context.Background(), "llm-request")
    defer span.End()

    // Simulate request
    result := MakeLLMRequest(ctx, "openai", "gpt-4", "test")

    // Query Jaeger
    trace, err := tracer.GetTrace(span.Context().TraceID())
    assert.NoError(t, err)
    assert.Greater(t, len(trace.Spans), 0)
}
```

---

### 4. Alerting Challenge
**Objective**: Verify alerting works correctly

**Critical Alerts** (Page immediately):
- 5 consecutive failures across all providers
- Checkpoint system failure
- Memory usage > 90% for 5 minutes
- TTFT > 10 seconds for 10 consecutive requests

**Warning Alerts** (Notify within 1 hour):
- Single provider degraded > 15 minutes
- Token consumption > 80% of daily quota

**Steps**:
1. Trigger critical alert
2. Verify page sent
3. Trigger warning alert
4. Verify notification sent
5. Check AlertManager

**Expected Results**:
- Critical alerts paged immediately
- Warning alerts sent within 1 hour
- AlertManager shows alerts
- Alerts resolved properly

**Test Code**:
```go
func TestAlerting(t *testing.T) {
    alertManager := NewAlertManager("localhost:9093")

    // Trigger critical alert
    triggerCriticalAlert("checkpoint_failure")
    alert := <-alertManager.CriticalAlerts

    assert.Equal(t, "critical", alert.Severity)
    assert.Equal(t, "checkpoint_failure", alert.Labels["alertname"])
}
```

---

### 5. Metric Collection Challenge
**Objective**: Verify metrics are collected for all operations

**Operations**:
- Model discovery
- Model verification
- Configuration export
- Event emission
- Database queries

**Steps**:
1. Perform various operations
2. Query metrics
3. Verify metrics for each operation
4. Verify counters increment
5. Verify histograms record

**Expected Results**:
- All operations have metrics
- Counters increment
- Histograms record
- Gauges update

**Test Code**:
```go
func TestMetricCollection(t *testing.T) {
    client := NewLLMVerifierClient()

    // Perform operations
    client.DiscoverModels("openai")
    client.VerifyModel("gpt-4")
    client.ExportConfig("opencode")

    // Query metrics
    metrics := queryPrometheusMetrics()

    assert.Greater(t, metrics["llm_discovery_duration_seconds"].Count, 0)
    assert.Greater(t, metrics["llm_verification_duration_seconds"].Count, 0)
}
```

---

### 6. Dashboard Panel Configuration Challenge
**Objective**: Verify dashboard panels are configured correctly

**Panel Types**:
- Graph: TTFT over time
- Stat: Last checkpoint age
- Graph: Request duration and errors
- Singlestat: Success rate

**Steps**:
1. Verify each panel configuration
2. Verify queries
3. Verify thresholds
4. Verify legends
5. Verify colors

**Expected Results**:
- All panels configured
- Queries correct
- Thresholds set
- Legends clear

**Test Code**:
```go
func TestDashboardPanelConfig(t *testing.T) {
    dashboard := loadGrafanaDashboard()

    // Check TTFT panel
    ttftPanel := findPanel(dashboard, "TTFT by Provider")
    assert.Equal(t, "graph", ttftPanel.Type)
    assert.Contains(t, ttftPanel.Targets[0].Expr,
        "histogram_quantile(0.5, llm_time_to_first_token_seconds_bucket)")
}
```

---

### 7. Health Endpoint Challenge
**Objective**: Verify health endpoints work

**Endpoints**:
- /health - Basic health
- /health/ready - Readiness probe
- /health/live - Liveness probe
- /health/providers - Provider health

**Steps**:
1. Query /health
2. Query /health/ready
3. Query /health/live
4. Query /health/providers
5. Verify responses

**Expected Results**:
- All endpoints respond
- Status codes correct
- Health status accurate
- Provider status accurate

**Test Code**:
```go
func TestHealthEndpoints(t *testing.T) {
    client := http.Client{}

    // Basic health
    resp, _ := client.Get("http://localhost:8080/health")
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    // Readiness
    resp, _ = client.Get("http://localhost:8080/health/ready")
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    // Provider health
    resp, _ = client.Get("http://localhost:8080/health/providers")
    var health map[string]string
    json.NewDecoder(resp.Body).Decode(&health)
    assert.Contains(t, health, "openai")
}
```

---

### 8. Log Aggregation Challenge
**Objective**: Verify logs are aggregated and searchable

**Steps**:
1. Generate logs at different levels
2. Query logs by level
3. Query logs by time range
4. Query logs by service
5. Verify log metadata

**Expected Results**:
- Logs aggregated
- Queries work
- Metadata preserved

**Test Code**:
```go
func TestLogAggregation(t *testing.T) {
    logger := NewStructuredLogger()

    logger.Info("Info message", map[string]interface{}{"key": "value"})
    logger.Error("Error message", nil)
    logger.Warn("Warning message", nil)

    // Query logs
    logs := queryLogs("level = 'ERROR'")
    assert.Equal(t, 1, len(logs))
    assert.Contains(t, logs[0].Message, "Error message")
}
```

---

### 9. Performance Monitoring Challenge
**Objective**: Verify performance metrics are monitored

**Metrics**:
- Memory usage over time
- CPU usage over time
- Network latency
- Request throughput
- Error rate

**Steps**:
1. Load test application
2. Monitor performance metrics
3. Verify alerts trigger
4. Verify dashboard updates
5. Analyze trends

**Expected Results**:
- Performance metrics collected
- Thresholds enforced
- Dashboard shows trends
- Alerts fire appropriately

**Test Code**:
```go
func TestPerformanceMonitoring(t *testing.T) {
    monitor := NewPerformanceMonitor()

    // Run load test
    results := runLoadTest(1000, 100) // 1000 requests, 100 concurrent

    // Check metrics
    assert.Greater(t, monitor.GetMemoryUsage(), 0)
    assert.Greater(t, monitor.GetCPUUsage(), 0)
    assert.Less(t, monitor.GetErrorRate(), 0.05) // Less than 5%
}
```

---

### 10. Observability Stack Integration Challenge
**Objective**: Verify all components work together

**Components**:
- Prometheus (metrics)
- Grafana (visualization)
- Jaeger (tracing)
- AlertManager (alerts)
- Loki (logs)

**Steps**:
1. Send test request
2. Check Prometheus for metrics
3. Check Grafana for dashboard
4. Check Jaeger for trace
5. Check AlertManager for alerts

**Expected Results**:
- All components working
- Data flows correctly
- Correlated traces and metrics
- Complete observability

**Test Code**:
```go
func TestObservabilityIntegration(t *testing.T) {
    traceID := uuid.New()

    // Send request with trace ID
    sendRequestWithTrace(traceID)

    // Check all components
    metrics := queryPrometheus("trace_id=\"" + traceID.String() + "\"")
    assert.Greater(t, len(metrics), 0)

    trace := queryJaeger(traceID)
    assert.NotNil(t, trace)

    logs := queryLoki("trace_id=\"" + traceID.String() + "\"")
    assert.Greater(t, len(logs), 0)
}
```

---

## Success Criteria

### Functional Requirements
- [ ] Prometheus metrics exposed
- [ ] Grafana dashboards work
- [ ] Jaeger tracing works
- [ ] Alerting works
- [ ] Metrics collected
- [ ] Dashboards configured
- [ ] Health endpoints work
- [ ] Logs aggregated
- [ ] Performance monitored
- [ ] Stack integrated

### Monitoring Requirements
- [ ] TTFT measured per provider
- [ ] Error rates tracked
- [ ] Circuit breaker state monitored
- [ ] Checkpoint success tracked
- [ ] Token consumption tracked

### Alerting Requirements
- [ ] Critical alerts paged immediately
- [ ] Warning alerts sent within 1 hour
- [ ] Alerts resolved
- [ ] Alert deduplication works

## Dependencies
- Prometheus running
- Grafana running
- Jaeger running
- AlertManager running
- Loki (optional)

## Cleanup
- Clear test metrics
- Remove test alerts
- Clear test traces
