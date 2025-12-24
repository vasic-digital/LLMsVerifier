# CLI Platform Comprehensive Challenge

## Overview
This challenge validates the complete functionality of the LLM Verifier CLI client, ensuring all features work correctly from the command line interface.

## Challenge Type
E2E (End-to-End) + Integration Test

## Platforms Covered
- CLI (Command Line Interface) - Main Implementation

## Test Scenarios

### 1. Basic Model Discovery CLI Challenge
**Objective**: Verify CLI can discover models from providers

**Steps**:
1. Run CLI with `llm-verifier discover` command
2. Verify output displays discovered models
3. Check that results are saved to database
4. Validate JSON report generation

**Expected Results**:
- All configured providers are queried
- Models are discovered and listed
- Database is updated with discovered models
- JSON report is generated

**Verification Commands**:
```bash
llm-verifier discover --providers openai,anthropic --output-file discover_results.json
cat discover_results.json | jq '.providers | length'
```

---

### 2. Model Verification CLI Challenge
**Objective**: Verify CLI can test and verify model capabilities

**Steps**:
1. Run CLI with `llm-verifier verify` command
2. Specify specific models or all models
3. Verify feature detection (streaming, function calling, etc.)
4. Check scoring system (0-100% usability score)

**Expected Results**:
- Models are verified for capabilities
- Usability scores are calculated (0-100%)
- Feature matrix is generated
- Markdown report is created

**Verification Commands**:
```bash
llm-verifier verify --model gpt-4 --features streaming,function_calling,vision
llm-verifier verify --all --output-file verification_results.md
grep "Usability Score" verification_results.md
```

---

### 3. Database Query CLI Challenge
**Objective**: Verify CLI can query and filter database

**Steps**:
1. Query database for top-rated models
2. Filter by features (streaming, vision, etc.)
3. Sort by score, speed, reliability
4. Export results to CSV/JSON

**Expected Results**:
- Database queries return correct results
- Filtering works correctly
- Sorting options work
- Export formats are valid

**Verification Commands**:
```bash
llm-verifier query --sort-by score --top 10
llm-verifier query --filter streaming=true --output models.json
llm-verifier query --provider openai --sort-by speed --format csv
```

---

### 4. Limits and Pricing CLI Challenge
**Objective**: Verify CLI can detect and display limits and pricing

**Steps**:
1. Query model limits (requests per day/week/month)
2. Check remaining limits
3. Display pricing information
4. Sort/filter by cost

**Expected Results**:
- Limits are displayed correctly
- Remaining limits are accurate
- Pricing is fetched and displayed
- Cost-based sorting works

**Verification Commands**:
```bash
llm-verifier limits --model gpt-4 --provider openai
llm-verifier pricing --all-providers --sort-by cost_per_1m_tokens
llm-verifier limits --check-remaining
```

---

### 5. Configuration Export CLI Challenge
**Objective**: Verify CLI can export configurations for other AI agents

**Steps**:
1. Export configuration for OpenCode
2. Export configuration for Crush
3. Export configuration for Claude Code
4. Verify generated configurations are valid

**Expected Results**:
- Configurations are generated in correct format
- API keys are properly included or redacted
- Models are prioritized by score
- Configurations are valid JSON/YAML

**Verification Commands**:
```bash
llm-verifier export --target opencode --output opencode_config.json
llm-verifier export --target crush --output crush_config.yaml
llm-verifier export --target claude_code --output claude_config.json --redact-keys
llm-verifier export --all-targets --directory configs/
```

---

### 6. Event Subscription CLI Challenge
**Objective**: Verify CLI can subscribe to events and receive notifications

**Steps**:
1. Register event subscriber via CLI
2. Trigger score change event
3. Receive notification
4. Unsubscribe from events

**Expected Results**:
- Event subscription is registered
- Events are received when triggered
- Notifications are displayed
- Unsubscription works

**Verification Commands**:
```bash
llm-verifier events subscribe --type score_change --channel stdout
llm-verifier events subscribe --type model_detected --channel websocket
llm-verifier events list
llm-verifier events unsubscribe --subscription-id <id>
```

---

### 7. Scheduled Re-test CLI Challenge
**Objective**: Verify CLI can schedule periodic re-tests

**Steps**:
1. Schedule daily re-test for all models
2. Schedule hourly re-test for specific provider
3. List scheduled tasks
4. Cancel scheduled task

**Expected Results**:
- Scheduled tasks are created
- Scheduling works (hourly, daily, weekly, monthly)
- Tasks are listed correctly
- Cancellation works

**Verification Commands**:
```bash
llm-verifier schedule --create --interval daily --all-models --time "02:00"
llm-verifier schedule --create --interval hourly --provider openai --models gpt-4
llm-verifier schedule list
llm-verifier schedule cancel --task-id <id>
```

---

### 8. Multi-provider Failover CLI Challenge
**Objective**: Verify CLI can handle provider failover

**Steps**:
1. Run verification with multiple providers
2. Simulate provider failure
3. Verify automatic failover
4. Check circuit breaker state

**Expected Results**:
- Failover works automatically
- Circuit breaker is triggered on failures
- Requests are routed to backup providers
- Provider health is monitored

**Verification Commands**:
```bash
llm-verifier verify --providers openai,anthropic,deepseek --failover enabled
llm-verifier health-check --all-providers
llm-verifier circuit-breaker --status
```

---

### 9. Context Management CLI Challenge
**Objective**: Verify CLI manages context and conversation history

**Steps**:
1. Start verification session with context tracking
2. Generate conversation summary
3. Retrieve relevant context
4. Manage conversation history

**Expected Results**:
- Context is tracked across requests
- Summaries are generated periodically
- History is trimmed when needed
- Memory integration works

**Verification Commands**:
```bash
llm-verifier verify --context-tracking enabled --summary-interval 10
llm-verifier context summary --session-id <id>
llm-verifier context retrieve --query "GPT-4 performance"
```

---

### 10. Report Generation CLI Challenge
**Objective**: Verify CLI generates comprehensive reports

**Steps**:
1. Generate Markdown report
2. Generate JSON report
3. Generate HTML report
4. Generate CSV export

**Expected Results**:
- All report formats are generated
- Reports contain all required information
- Reports are valid in their respective formats
- Sorting/filtering is applied in reports

**Verification Commands**:
```bash
llm-verifier report --format markdown --output report.md --all-models
llm-verifier report --format json --output report.json
llm-verifier report --format html --output report.html --provider openai
llm-verifier report --format csv --output models.csv --sort-by score
```

---

## Success Criteria

### Functional Requirements
- [ ] All CLI commands execute without errors
- [ ] Help documentation is complete and accurate
- [ ] Output formats are valid and readable
- [ ] Database operations complete successfully
- [ ] Event subscriptions work correctly
- [ ] Scheduling operations work correctly
- [ ] Failover mechanisms work as expected
- [ ] Context management is functional

### Non-Functional Requirements
- [ ] CLI completes within acceptable time limits
- [ ] Memory usage is within limits
- [ ] Error messages are clear and actionable
- [ ] Progress indicators are displayed for long operations
- [ ] Exit codes are appropriate for success/failure

## Dependencies
- Database must be initialized
- Provider API keys must be configured
- Configuration file must exist

## Test Data Requirements
- At least 2 different providers configured
- At least 3 models per provider
- Valid API keys for all providers

## Cleanup
- Remove generated reports
- Clear test event subscriptions
- Cancel scheduled tasks
