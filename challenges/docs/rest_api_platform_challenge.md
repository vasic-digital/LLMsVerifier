# REST API Comprehensive Challenge

## Overview
This challenge validates the complete functionality of the LLM Verifier REST API, ensuring all endpoints work correctly with proper authentication, validation, and error handling.

## Challenge Type
Integration Test + API Contract Test + Load Test

## Platforms Covered
- REST API (GinGonic Framework)

## Test Scenarios

### 1. API Authentication Challenge
**Objective**: Verify API authentication mechanisms

**Steps**:
1. Test API key authentication
2. Test JWT token authentication
3. Test authentication failures
4. Test token refresh

**Expected Results**:
- API key authentication works
- JWT token authentication works
- Invalid credentials return 401
- Token refresh works correctly

**Test Requests**:
```bash
# API Key Authentication
curl -X GET http://localhost:8080/api/v1/models \
  -H "X-API-Key: your-api-key"

# JWT Authentication
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}'

curl -X GET http://localhost:8080/api/v1/models \
  -H "Authorization: Bearer <jwt-token>"

# Invalid Auth (should fail)
curl -X GET http://localhost:8080/api/v1/models \
  -H "X-API-Key: invalid-key"
```

---

### 2. Model Discovery API Challenge
**Objective**: Verify model discovery endpoints

**Steps**:
1. Discover models from all providers
2. Discover models from specific provider
3. Stream discovery results
4. Handle discovery errors

**Expected Results**:
- All providers are queried
- Specific provider discovery works
- Streaming returns results incrementally
- Errors are handled gracefully

**Test Requests**:
```bash
# Discover all models
curl -X POST http://localhost:8080/api/v1/discover \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"providers": ["openai","anthropic"]}'

# Discover specific provider
curl -X POST http://localhost:8080/api/v1/discover/openai \
  -H "Authorization: Bearer <token>"

# Stream results
curl -X GET http://localhost:8080/api/v1/discover/stream \
  -H "Authorization: Bearer <token>"
```

---

### 3. Model Verification API Challenge
**Objective**: Verify model verification endpoints

**Steps**:
1. Verify single model
2. Verify multiple models
3. Check verification status
4. Get verification results

**Expected Results**:
- Single model verification works
- Batch verification works
- Status endpoint returns progress
- Results are complete and accurate

**Test Requests**:
```bash
# Verify single model
curl -X POST http://localhost:8080/api/v1/verify \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"model_id":"gpt-4","provider":"openai","features":["streaming","function_calling"]}'

# Verify multiple models
curl -X POST http://localhost:8080/api/v1/verify/batch \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"models":[{"model_id":"gpt-4","provider":"openai"},{"model_id":"claude-3-opus","provider":"anthropic"}]}'

# Check status
curl -X GET http://localhost:8080/api/v1/verify/status/<job-id> \
  -H "Authorization: Bearer <token>"

# Get results
curl -X GET http://localhost:8080/api/v1/verify/results/<job-id> \
  -H "Authorization: Bearer <token>"
```

---

### 4. Database Query API Challenge
**Objective**: Verify database query endpoints

**Steps**:
1. Query all models
2. Filter models by criteria
3. Sort and paginate results
4. Export results

**Expected Results**:
- All models are returned
- Filters work correctly
- Sorting and pagination work
- Export formats are valid

**Test Requests**:
```bash
# Query all models
curl -X GET http://localhost:8080/api/v1/models \
  -H "Authorization: Bearer <token>"

# Filter models
curl -X GET "http://localhost:8080/api/v1/models?provider=openai&min_score=80&features=streaming" \
  -H "Authorization: Bearer <token>"

# Sort and paginate
curl -X GET "http://localhost:8080/api/v1/models?sort_by=score&order=desc&page=1&limit=20" \
  -H "Authorization: Bearer <token>"

# Export results
curl -X GET "http://localhost:8080/api/v1/models/export?format=json" \
  -H "Authorization: Bearer <token>" -o models.json
```

---

### 5. Limits and Pricing API Challenge
**Objective**: Verify limits and pricing endpoints

**Steps**:
1. Get model limits
2. Get remaining limits
3. Get pricing information
4. Get cost estimates

**Expected Results**:
- Limits are returned accurately
- Remaining limits are correct
- Pricing is up-to-date
- Cost estimates are calculated

**Test Requests**:
```bash
# Get limits
curl -X GET http://localhost:8080/api/v1/limits/gpt-4 \
  -H "Authorization: Bearer <token>"

# Get remaining limits
curl -X GET http://localhost:8080/api/v1/limits/remaining \
  -H "Authorization: Bearer <token>"

# Get pricing
curl -X GET http://localhost:8080/api/v1/pricing \
  -H "Authorization: Bearer <token>"

# Get cost estimate
curl -X POST http://localhost:8080/api/v1/pricing/estimate \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"model_id":"gpt-4","tokens":1000}'
```

---

### 6. Configuration Export API Challenge
**Objective**: Verify configuration export endpoints

**Steps**:
1. Export for OpenCode
2. Export for Crush
3. Export for Claude Code
4. Export for multiple targets

**Expected Results**:
- Exports are in correct format
- API keys are handled securely
- Models are prioritized
- Exports are valid

**Test Requests**:
```bash
# Export for OpenCode
curl -X POST http://localhost:8080/api/v1/export/opencode \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"redact_keys":false,"min_score":70}' -o opencode_config.json

# Export for Crush
curl -X POST http://localhost:8080/api/v1/export/crush \
  -H "Authorization: Bearer <token>" -o crush_config.yaml

# Export for Claude Code
curl -X POST http://localhost:8080/api/v1/export/claude-code \
  -H "Authorization: Bearer <token>" -o claude_config.json

# Export all
curl -X GET http://localhost:8080/api/v1/export/all \
  -H "Authorization: Bearer <token>" -o exports.zip
```

---

### 7. Event Subscription API Challenge
**Objective**: Verify event subscription and WebSocket endpoints

**Steps**:
1. Subscribe to events via HTTP
2. Subscribe via WebSocket
3. Receive live events
4. Unsubscribe from events

**Expected Results**:
- HTTP subscriptions work
- WebSocket connections work
- Events are received in real-time
- Unsubscription removes subscription

**Test Requests**:
```bash
# Subscribe via HTTP
curl -X POST http://localhost:8080/api/v1/events/subscribe \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"event_types":["score_change","model_detected"],"callback_url":"http://example.com/webhook"}'

# List subscriptions
curl -X GET http://localhost:8080/api/v1/events/subscriptions \
  -H "Authorization: Bearer <token>"

# WebSocket connection
wscat -c ws://localhost:8080/api/v1/events/ws?token=<jwt-token>

# Unsubscribe
curl -X DELETE http://localhost:8080/api/v1/events/subscribe/<subscription-id> \
  -H "Authorization: Bearer <token>"
```

---

### 8. Scheduling API Challenge
**Objective**: Verify scheduling endpoints

**Steps**:
1. Create scheduled task
2. List scheduled tasks
3. Get task details
4. Cancel task

**Expected Results**:
- Tasks are created correctly
- Task list is accurate
- Task details are complete
- Cancellation removes task

**Test Requests**:
```bash
# Create task
curl -X POST http://localhost:8080/api/v1/schedule \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Daily Verification","interval":"daily","time":"02:00","all_models":true}'

# List tasks
curl -X GET http://localhost:8080/api/v1/schedule \
  -H "Authorization: Bearer <token>"

# Get task details
curl -X GET http://localhost:8080/api/v1/schedule/<task-id> \
  -H "Authorization: Bearer <token>"

# Cancel task
curl -X DELETE http://localhost:8080/api/v1/schedule/<task-id> \
  -H "Authorization: Bearer <token>"
```

---

### 9. Health and Monitoring API Challenge
**Objective**: Verify health and monitoring endpoints

**Steps**:
1. Check API health
2. Get provider health status
3. Get metrics
4. Get circuit breaker status

**Expected Results**:
- Health endpoint returns status
- Provider health is accurate
- Metrics are comprehensive
- Circuit breaker state is current

**Test Requests**:
```bash
# Health check
curl -X GET http://localhost:8080/health

# Provider health
curl -X GET http://localhost:8080/api/v1/health/providers \
  -H "Authorization: Bearer <token>"

# Metrics (Prometheus format)
curl -X GET http://localhost:8080/metrics

# Circuit breaker status
curl -X GET http://localhost:8080/api/v1/health/circuit-breakers \
  -H "Authorization: Bearer <token>"
```

---

### 10. Reporting API Challenge
**Objective**: Verify report generation endpoints

**Steps**:
1. Generate Markdown report
2. Generate JSON report
3. Generate HTML report
4. Generate CSV export

**Expected Results**:
- Reports are generated in correct format
- Reports contain all information
- Reports are valid
- Content is accurate

**Test Requests**:
```bash
# Markdown report
curl -X POST http://localhost:8080/api/v1/reports \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"format":"markdown","all_models":true}' -o report.md

# JSON report
curl -X POST http://localhost:8080/api/v1/reports \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"format":"json","provider":"openai"}' -o report.json

# HTML report
curl -X POST http://localhost:8080/api/v1/reports \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"format":"html","min_score":70}' -o report.html
```

---

## Success Criteria

### Functional Requirements
- [ ] All endpoints respond correctly
- [ ] Authentication works for all protected endpoints
- [ ] Input validation prevents invalid requests
- [ ] Error responses have proper status codes and messages
- [ ] Pagination works for large datasets
- [ ] Filtering and sorting work correctly
- [ ] WebSocket connections are stable
- [ ] Rate limiting is enforced

### Performance Requirements
- [ ] Response time < 200ms for simple queries
- [ ] Response time < 2s for complex operations
- [ ] API can handle 1000 requests/minute
- [ ] WebSocket latency < 100ms
- [ ] Streaming responses start within 500ms

### Security Requirements
- [ ] API keys are validated
- [ ] JWT tokens are properly signed and verified
- [ ] Sensitive data is never leaked in responses
- [ ] CORS is properly configured
- [ ] Rate limiting prevents abuse
- [ ] SQL injection is prevented

## Dependencies
- API server must be running
- Database must be initialized
- Provider API keys must be configured

## Test Data Requirements
- Valid API key or JWT token
- At least 2 providers configured
- Test models available

## Cleanup
- Remove test subscriptions
- Cancel scheduled tasks
- Delete test reports
