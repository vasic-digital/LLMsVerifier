# ACP (AI Coding Protocol) Examples and Usage Demonstrations

## Overview

This document provides comprehensive examples and demonstrations of ACP (AI Coding Protocol) implementation in the LLM Verifier project, including practical use cases, configuration examples, and integration patterns.

## Table of Contents

1. [Basic ACP Detection Examples](#basic-acp-detection-examples)
2. [Configuration Examples](#configuration-examples)
3. [API Usage Examples](#api-usage-examples)
4. [Integration Examples](#integration-examples)
5. [Test Examples](#test-examples)
6. [Performance Examples](#performance-examples)
7. [Real-world Use Cases](#real-world-use-cases)
8. [Troubleshooting Examples](#troubleshooting-examples)

---

## Basic ACP Detection Examples

### Example 1: Simple ACP Detection

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/llmverifier/llmverifier"
    "github.com/llmverifier/llmverifier/config"
    "github.com/llmverifier/llmverifier/providers"
)

func main() {
    // Setup configuration
    cfg := &config.Config{
        GlobalTimeout: 30 * time.Second,
        MaxRetries:    3,
    }

    // Create verifier
    verifier := llmverifier.New(cfg)

    // Create client for OpenAI
    registry := providers.NewProviderRegistry()
    providerConfig, _ := registry.GetConfig("openai")
    
    client, err := providers.NewClient(providerConfig, "sk-your-api-key")
    if err != nil {
        log.Fatal("Failed to create client:", err)
    }

    // Test ACP support
    ctx := context.Background()
    modelName := "gpt-4"
    
    supportsACP := verifier.TestACPs(client, modelName, ctx)
    
    fmt.Printf("Model %s ACP support: %t\n", modelName, supportsACP)
    
    if supportsACP {
        fmt.Println("✓ This model supports ACP capabilities")
    } else {
        fmt.Println("✗ This model does not support ACP capabilities")
    }
}
```

### Example 2: Batch ACP Detection

```go
func batchACPDetection() {
    models := []string{
        "gpt-4",
        "gpt-3.5-turbo",
        "claude-3-opus",
        "claude-3-sonnet",
        "deepseek-chat",
    }

    results := make(map[string]bool)
    
    for _, model := range models {
        ctx := context.Background()
        supportsACP := verifier.TestACPs(client, model, ctx)
        results[model] = supportsACP
        
        fmt.Printf("%-20s: %t\n", model, supportsACP)
    }
    
    // Summary statistics
    supportedCount := 0
    for _, supported := range results {
        if supported {
            supportedCount++
        }
    }
    
    fmt.Printf("\nSummary: %d/%d models support ACP\n", supportedCount, len(models))
}
```

### Example 3: Detailed ACP Analysis

```go
func detailedACPAnalysis() {
    // Run complete verification including ACP
    results, err := verifier.Verify()
    if err != nil {
        log.Fatal("Verification failed:", err)
    }
    
    // Analyze ACP results
    fmt.Println("=== ACP Analysis Report ===")
    fmt.Printf("Total models tested: %d\n", len(results))
    
    var acpSupported []string
    var acpNotSupported []string
    
    for _, result := range results {
        if result.FeatureDetection.ACPs {
            acpSupported = append(acpSupported, result.ModelInfo.ID)
        } else {
            acpNotSupported = append(acpNotSupported, result.ModelInfo.ID)
        }
    }
    
    fmt.Printf("ACP Supported: %d models\n", len(acpSupported))
    for _, model := range acpSupported {
        fmt.Printf("  ✓ %s\n", model)
    }
    
    fmt.Printf("ACP Not Supported: %d models\n", len(acpNotSupported))
    for _, model := range acpNotSupported {
        fmt.Printf("  ✗ %s\n", model)
    }
    
    // Score analysis
    fmt.Println("\n=== Score Analysis ===")
    for _, result := range results {
        acpStatus := "No"
        if result.FeatureDetection.ACPs {
            acpStatus = "Yes"
        }
        fmt.Printf("%-20s | ACP: %-3s | Overall Score: %.1f\n",
            result.ModelInfo.ID,
            acpStatus,
            result.PerformanceScores.OverallScore)
    }
}
```

---

## Configuration Examples

### Provider Configuration with ACP Support

```json
{
  "name": "openai",
  "endpoint": "https://api.openai.com/v1",
  "auth_type": "bearer",
  "streaming_format": "sse",
  "default_model": "gpt-4",
  "rate_limits": {
    "requests_per_minute": 60,
    "requests_per_hour": 1000,
    "burst_limit": 10
  },
  "features": {
    "supports_streaming": true,
    "supports_functions": true,
    "supports_vision": true,
    "supports_acp": true,
    "max_context_length": 128000,
    "supported_models": ["gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"],
    "acp_config": {
      "protocol_version": "2.0",
      "max_tool_calls": 10,
      "context_window_size": 128000,
      "supported_methods": [
        "textDocument/completion",
        "textDocument/hover",
        "textDocument/definition"
      ]
    }
  }
}
```

### Model-Specific ACP Configuration

```yaml
models:
  - id: "gpt-4"
    provider: "openai"
    capabilities:
      acp:
        enabled: true
        features:
          - jsonrpc_compliance
          - tool_calling
          - context_management
          - code_assistance
          - error_detection
        timeout: 30
        retry_config:
          max_retries: 3
          backoff_factor: 2.0

  - id: "claude-3-opus"
    provider: "anthropic"
    capabilities:
      acp:
        enabled: true
        features:
          - jsonrpc_compliance
          - context_management
          - code_assistance
        timeout: 45
        retry_config:
          max_retries: 5
          backoff_factor: 1.5
```

### Environment-Specific ACP Settings

```go
func getACPConfigForEnvironment(env string) map[string]interface{} {
    switch env {
    case "development":
        return map[string]interface{}{
            "enabled": true,
            "timeout": 60,
            "max_tool_calls": 20,
            "debug_mode": true,
        }
    case "staging":
        return map[string]interface{}{
            "enabled": true,
            "timeout": 45,
            "max_tool_calls": 15,
            "debug_mode": false,
        }
    case "production":
        return map[string]interface{}{
            "enabled": true,
            "timeout": 30,
            "max_tool_calls": 10,
            "debug_mode": false,
        }
    default:
        return map[string]interface{}{
            "enabled": false,
        }
    }
}
```

---

## API Usage Examples

### REST API Examples

#### Verify ACP Support

```bash
# Test ACP support for a specific model
curl -X POST http://localhost:8080/api/verify/acp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model_name": "gpt-4",
    "provider": "openai",
    "endpoint": "https://api.openai.com/v1",
    "api_key": "sk-your-api-key",
    "test_scenarios": [
      {"name": "jsonrpc_compliance", "enabled": true},
      {"name": "tool_calling", "enabled": true},
      {"name": "context_management", "enabled": true},
      {"name": "code_assistance", "enabled": true},
      {"name": "error_detection", "enabled": true}
    ],
    "timeout": 30
  }'
```

#### Get ACP Test Results

```bash
# Get detailed ACP test results
curl -X GET http://localhost:8080/api/results/acp/gpt-4 \
  -H "Authorization: Bearer YOUR_API_KEY"
```

#### List Models with ACP Support

```bash
# List all models that support ACP
curl -X GET "http://localhost:8080/api/models/acp?min_score=0.7&limit=50" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Go SDK Examples

#### Basic ACP Verification

```go
package main

import (
    "fmt"
    "log"
    "github.com/llmverifier/sdk-go"
)

func main() {
    // Create SDK client
    client := llmverifier.NewClient("YOUR_API_KEY")
    
    // Verify ACP support
    result, err := client.VerifyACP("gpt-4", "openai")
    if err != nil {
        log.Fatal("ACP verification failed:", err)
    }
    
    fmt.Printf("ACP Support: %t\n", result.Supported)
    fmt.Printf("Overall Score: %.2f\n", result.Score)
    fmt.Printf("Confidence: %.2f\n", result.Confidence)
    
    // Print capability details
    for capability, detail := range result.Capabilities {
        fmt.Printf("%s: %t (score: %.2f)\n", 
            capability, detail.Supported, detail.Score)
    }
}
```

#### Batch ACP Verification

```go
func batchACPVerification() {
    client := llmverifier.NewClient("YOUR_API_KEY")
    
    models := []struct {
        ModelID  string
        Provider string
    }{
        {"gpt-4", "openai"},
        {"gpt-3.5-turbo", "openai"},
        {"claude-3-opus", "anthropic"},
        {"claude-3-sonnet", "anthropic"},
        {"deepseek-chat", "deepseek"},
    }
    
    results, err := client.VerifyACPBatch(models)
    if err != nil {
        log.Fatal("Batch verification failed:", err)
    }
    
    for _, result := range results {
        fmt.Printf("%-20s | ACP: %t | Score: %.2f\n",
            result.ModelID, result.Supported, result.Score)
    }
}
```

### Python SDK Examples

```python
import llmverifier

# Create client
client = llmverifier.Client("YOUR_API_KEY")

# Verify single model
result = client.verify_acp("gpt-4", "openai")
print(f"ACP Support: {result.supported}")
print(f"Overall Score: {result.score:.2f}")

# Verify multiple models
models = [
    {"model_id": "gpt-4", "provider": "openai"},
    {"model_id": "claude-3-opus", "provider": "anthropic"},
    {"model_id": "deepseek-chat", "provider": "deepseek"}
]

results = client.verify_acp_batch(models)
for result in results:
    print(f"{result.model_id}: ACP={result.supported}, Score={result.score:.2f}")
```

---

## Integration Examples

### CI/CD Integration

#### GitHub Actions Workflow

```yaml
name: ACP Verification

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  acp-verification:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.21
    
    - name: Run ACP Verification
      env:
        OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
        ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
      run: |
        go run cmd/acp-verification/main.go \
          --models gpt-4,gpt-3.5-turbo,claude-3-opus,claude-3-sonnet \
          --output acp-results.json \
          --format json
    
    - name: Upload Results
      uses: actions/upload-artifact@v3
      with:
        name: acp-results
        path: acp-results.json
    
    - name: Comment PR
      if: github.event_name == 'pull_request'
      uses: actions/github-script@v6
      with:
        script: |
          const fs = require('fs');
          const results = JSON.parse(fs.readFileSync('acp-results.json', 'utf8'));
          
          let comment = '## ACP Verification Results\n\n';
          comment += '| Model | ACP Support | Score |\n';
          comment += '|-------|-------------|-------|\n';
          
          for (const result of results) {
            comment += `| ${result.model} | ${result.acp_support ? '✅' : '❌'} | ${result.score.toFixed(2)} |\n`;
          }
          
          github.rest.issues.createComment({
            issue_number: context.issue.number,
            owner: context.repo.owner,
            repo: context.repo.repo,
            body: comment
          });
```

#### Pre-commit Hook

```bash
#!/bin/bash
# .git/hooks/pre-commit

echo "Running ACP verification on changed models..."

# Get changed model configurations
changed_models=$(git diff --cached --name-only | grep "models/" | xargs)

if [ -n "$changed_models" ]; then
    # Run ACP verification
    go run cmd/acp-verification/main.go \
        --config-files $changed_models \
        --fail-on-missing-acp
    
    if [ $? -ne 0 ]; then
        echo "❌ ACP verification failed for some models"
        exit 1
    fi
fi

echo "✅ ACP verification passed"
```

### Monitoring Integration

#### Prometheus Metrics

```go
package monitoring

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    acpVerificationDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "acp_verification_duration_seconds",
            Help: "Duration of ACP verification tests",
            Buckets: prometheus.DefBuckets,
        },
        []string{"provider", "model", "result"},
    )
    
    acpSupportRate = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "acp_support_rate",
            Help: "Rate of ACP support by provider",
        },
        []string{"provider"},
    )
    
    acpTestFailures = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "acp_test_failures_total",
            Help: "Total number of ACP test failures",
        },
        []string{"provider", "model", "reason"},
    )
)

func RecordACPVerification(provider, model string, duration float64, supported bool) {
    result := "unsupported"
    if supported {
        result = "supported"
    }
    
    acpVerificationDuration.WithLabelValues(provider, model, result).Observe(duration)
}
```

#### Grafana Dashboard

```json
{
  "dashboard": {
    "title": "ACP Verification Dashboard",
    "panels": [
      {
        "title": "ACP Support Rate by Provider",
        "type": "stat",
        "targets": [
          {
            "expr": "acp_support_rate",
            "legendFormat": "{{provider}}"
          }
        ]
      },
      {
        "title": "ACP Verification Duration",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, acp_verification_duration_seconds)",
            "legendFormat": "95th percentile"
          }
        ]
      },
      {
        "title": "ACP Test Failures",
        "type": "table",
        "targets": [
          {
            "expr": "increase(acp_test_failures_total[1h])",
            "legendFormat": "{{provider}} - {{model}}"
          }
        ]
      }
    ]
  }
}
```

---

## Test Examples

### Unit Test Examples

```go
package tests

import (
    "testing"
    "github.com/llmverifier/llmverifier"
)

func TestACPsDetection(t *testing.T) {
    // Create mock client with ACP support
    mockClient := &MockLLMClient{
        Responses: map[string]string{
            "jsonrpc": `{"jsonrpc":"2.0","result":{"items":[{"label":"print"}]}}`,
            "tool":    `I'll use the file_read tool to analyze the file`,
            "context": `Based on your project structure, I recommend...`,
            "code":    `def validate_users(users: List[Dict]) -> List[Dict]:`,
            "error":   `Line 3: KeyError - missing 'email' key`,
        },
    }
    
    verifier := llmverifier.New(&config.Config{})
    ctx := context.Background()
    
    supportsACP := verifier.TestACPs(mockClient, "test-model", ctx)
    
    if !supportsACP {
        t.Error("Expected ACP support to be detected")
    }
}

func TestACPsWithVariousResponses(t *testing.T) {
    testCases := []struct {
        name           string
        responses      map[string]string
        expectedResult bool
    }{
        {
            name: "Full ACP Support",
            responses: map[string]string{
                "jsonrpc": `{"jsonrpc":"2.0","result":{}}`,
                "tool":    `Using file_read tool`,
                "context": `Remembering project structure`,
                "code":    `def function(): pass`,
                "error":   `Error on line 3`,
            },
            expectedResult: true,
        },
        {
            name: "Partial ACP Support",
            responses: map[string]string{
                "jsonrpc": `I understand JSON-RPC`,
                "tool":    `I can use tools`,
                "context": `I remember context`,
                "code":    ``,  // Empty response
                "error":   ``,  // Empty response
            },
            expectedResult: true,
        },
        {
            name: "No ACP Support",
            responses: map[string]string{
                "jsonrpc": `I don't understand`,
                "tool":    `What tools?`,
                "context": `What context?`,
                "code":    `I can't code`,
                "error":   `I don't see errors`,
            },
            expectedResult: false,
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            mockClient := &MockLLMClient{Responses: tc.responses}
            verifier := llmverifier.New(&config.Config{})
            
            supportsACP := verifier.TestACPs(mockClient, "test-model", context.Background())
            
            if supportsACP != tc.expectedResult {
                t.Errorf("Expected %t, got %t", tc.expectedResult, supportsACP)
            }
        })
    }
}
```

### Integration Test Examples

```go
func TestACPsIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // Setup test environment
    cfg := loadIntegrationTestConfig()
    verifier := llmverifier.New(cfg)
    
    // Test with real providers
    providers := []string{"openai", "anthropic"}
    
    for _, providerName := range providers {
        t.Run(providerName, func(t *testing.T) {
            // This would use real API keys in a secure test environment
            result := testACPWithProvider(t, verifier, providerName)
            
            if !result.Supported {
                t.Logf("Provider %s does not support ACP (this may be expected)", providerName)
            }
            
            // Verify result is reasonable
            if result.Score < 0 || result.Score > 1 {
                t.Errorf("Invalid ACP score: %f", result.Score)
            }
        })
    }
}
```

---

## Performance Examples

### Performance Benchmarking

```go
package main

import (
    "fmt"
    "time"
    "github.com/llmverifier/llmverifier"
)

func benchmarkACPDetection() {
    models := []string{
        "gpt-3.5-turbo",
        "gpt-4",
        "claude-3-haiku",
        "claude-3-sonnet",
        "claude-3-opus",
    }
    
    fmt.Println("=== ACP Detection Performance Benchmark ===")
    fmt.Printf("%-20s | %-10s | %-10s | %-10s\n", "Model", "Duration", "Supported", "Score")
    fmt.Println(string(make([]byte, 60, 60)))
    
    for _, model := range models {
        start := time.Now()
        
        // Run ACP detection
        supportsACP := verifier.TestACPs(client, model, ctx)
        
        duration := time.Since(start)
        
        fmt.Printf("%-20s | %-10s | %-10t | %-10s\n",
            model,
            duration.Round(time.Millisecond),
            supportsACP,
            "N/A") // Score would come from detailed results
    }
}
```

### Concurrent ACP Testing

```go
func concurrentACPTesting(models []string) {
    fmt.Println("=== Concurrent ACP Testing ===")
    
    start := time.Now()
    
    // Create channels for results
    results := make(chan struct {
        model    string
        supported bool
        duration time.Duration
    }, len(models))
    
    // Launch concurrent tests
    for _, model := range models {
        go func(m string) {
            testStart := time.Now()
            supported := verifier.TestACPs(client, m, ctx)
            duration := time.Since(testStart)
            
            results <- struct {
                model    string
                supported bool
                duration time.Duration
            }{m, supported, duration}
        }(model)
    }
    
    // Collect results
    for i := 0; i < len(models); i++ {
        result := <-results
        fmt.Printf("%-20s | %-8t | %s\n", 
            result.model, result.supported, 
            result.duration.Round(time.Millisecond))
    }
    
    totalDuration := time.Since(start)
    fmt.Printf("\nTotal concurrent testing time: %s\n", totalDuration.Round(time.Millisecond))
}
```

---

## Real-world Use Cases

### Use Case 1: IDE Plugin Development

```javascript
// VS Code extension integration
const vscode = require('vscode');
const llmverifier = require('llmverifier');

class ACPCodeLensProvider {
    constructor(client) {
        this.client = client;
    }
    
    async provideCodeLenses(document, token) {
        const codeLenses = [];
        
        // Get current model configuration
        const config = vscode.workspace.getConfiguration('acp');
        const model = config.get('model', 'gpt-4');
        const provider = config.get('provider', 'openai');
        
        // Verify ACP support
        const acpResult = await this.client.verifyACP(model, provider);
        
        if (acpResult.supported) {
            // Add code lens for ACP-powered features
            const lens = new vscode.CodeLens(
                new vscode.Range(0, 0, 0, 0),
                {
                    title: `ACP: ${(acpResult.score * 100).toFixed(0)}%`,
                    command: 'acp.showCapabilities',
                    arguments: [acpResult]
                }
            );
            codeLenses.push(lens);
        }
        
        return codeLenses;
    }
}
```

### Use Case 2: Continuous Integration

```yaml
# GitLab CI configuration
stages:
  - validate
  - test
  - deploy

acp-validation:
  stage: validate
  image: golang:1.21
  script:
    - |
      go run cmd/acp-validator/main.go \
        --models "$MODELS" \
        --providers "$PROVIDERS" \
        --output acp-report.json \
        --format json \
        --fail-threshold 0.7
    
    - |
      if [ -f acp-report.json ]; then
        echo "## ACP Validation Results"
        cat acp-report.json | jq -r '.results[] | "- \(.model): \(if .supported then "✅" else "❌" end) (score: \(.score))"'
      fi
  variables:
    MODELS: "gpt-4,gpt-3.5-turbo,claude-3-opus"
    PROVIDERS: "openai,anthropic"
  artifacts:
    reports:
      json: acp-report.json
    expire_in: 1 week
```

### Use Case 3: Model Selection Dashboard

```python
# Flask web application for model selection
from flask import Flask, render_template, jsonify
import llmverifier

app = Flask(__name__)
client = llmverifier.Client("YOUR_API_KEY")

@app.route('/api/models/acp')
def get_acp_models():
    """Get models with ACP support"""
    models = [
        {"model_id": "gpt-4", "provider": "openai"},
        {"model_id": "claude-3-opus", "provider": "anthropic"},
        {"model_id": "deepseek-chat", "provider": "deepseek"},
    ]
    
    results = client.verify_acp_batch(models)
    
    return jsonify({
        "models": [
            {
                "model_id": r.model_id,
                "provider": r.provider,
                "acp_supported": r.supported,
                "acp_score": r.score,
                "recommendation": "recommended" if r.score > 0.8 else "acceptable"
            }
            for r in results
        ]
    })

@app.route('/dashboard')
def dashboard():
    """Render ACP dashboard"""
    return render_template('acp_dashboard.html')
```

---

## Troubleshooting Examples

### Common Issues and Solutions

#### Issue 1: ACP Detection Returns False for Supported Models

**Problem**: Model shows ACP capabilities but detection returns false

**Diagnostic Script**:
```go
func debugACPDetection() {
    model := "gpt-4"
    
    fmt.Println("=== ACP Detection Debug ===")
    
    // Test each capability individually
    tests := []struct {
        name string
        test func() bool
    }{
        {
            name: "JSON-RPC Test",
            test: func() bool {
                req := createJSONRPCRequest()
                resp, err := client.ChatCompletion(ctx, req)
                if err != nil {
                    fmt.Printf("JSON-RPC Error: %v\n", err)
                    return false
                }
                fmt.Printf("JSON-RPC Response: %s\n", resp.Choices[0].Message.Content)
                return evaluateJSONRPCResponse(resp.Choices[0].Message.Content)
            },
        },
        {
            name: "Tool Calling Test",
            test: func() bool {
                req := createToolCallingRequest()
                resp, err := client.ChatCompletion(ctx, req)
                if err != nil {
                    fmt.Printf("Tool Calling Error: %v\n", err)
                    return false
                }
                fmt.Printf("Tool Calling Response: %s\n", resp.Choices[0].Message.Content)
                return evaluateToolCallingResponse(resp.Choices[0].Message.Content)
            },
        },
        // Add more tests...
    }
    
    for _, test := range tests {
        fmt.Printf("\n%s:\n", test.name)
        result := test.test()
        fmt.Printf("Result: %t\n", result)
    }
}
```

#### Issue 2: Performance Issues

**Problem**: ACP detection takes too long

**Performance Analysis**:
```go
func analyzeACPPerformance() {
    model := "gpt-4"
    
    fmt.Println("=== ACP Performance Analysis ===")
    
    // Measure individual test times
    testTimes := make(map[string]time.Duration)
    
    tests := []struct {
        name string
        fn   func()
    }{
        {
            name: "JSON-RPC Test",
            fn: func() {
                req := createJSONRPCRequest()
                client.ChatCompletion(ctx, req)
            },
        },
        {
            name: "Tool Calling Test",
            fn: func() {
                req := createToolCallingRequest()
                client.ChatCompletion(ctx, req)
            },
        },
        // Add more tests...
    }
    
    for _, test := range tests {
        start := time.Now()
        test.fn()
        duration := time.Since(start)
        testTimes[test.name] = duration
        fmt.Printf("%s: %s\n", test.name, duration.Round(time.Millisecond))
    }
    
    // Identify bottlenecks
    totalTime := time.Duration(0)
    for _, duration := range testTimes {
        totalTime += duration
    }
    
    fmt.Printf("\nTotal time: %s\n", totalTime.Round(time.Millisecond))
    
    // Suggest optimizations
    fmt.Println("\nOptimization Suggestions:")
    for name, duration := range testTimes {
        if duration > 2*time.Second {
            fmt.Printf("- %s is slow (%s), consider optimization\n", name, duration)
        }
    }
}
```

#### Issue 3: Configuration Problems

**Problem**: ACP configuration not working as expected

**Configuration Validator**:
```go
func validateACPConfiguration() {
    fmt.Println("=== ACP Configuration Validation ===")
    
    // Check provider configurations
    registry := providers.NewProviderRegistry()
    providers := []string{"openai", "anthropic", "deepseek", "google"}
    
    for _, providerName := range providers {
        config, exists := registry.GetConfig(providerName)
        if !exists {
            fmt.Printf("❌ Provider %s not found in registry\n", providerName)
            continue
        }
        
        fmt.Printf("\nProvider: %s\n", providerName)
        
        // Check ACP feature
        if acpSupport, ok := config.Features["supports_acp"]; ok {
            fmt.Printf("  ACP Support: %t\n", acpSupport.(bool))
        } else {
            fmt.Printf("  ACP Support: not configured\n")
        }
        
        // Check ACP-specific configuration
        if acpConfig, ok := config.Features["acp_config"]; ok {
            fmt.Printf("  ACP Config: present\n")
            config := acpConfig.(map[string]interface{})
            fmt.Printf("    Protocol Version: %s\n", config["protocol_version"])
            fmt.Printf("    Max Tool Calls: %v\n", config["max_tool_calls"])
            fmt.Printf("    Context Window: %v\n", config["context_window_size"])
        } else {
            fmt.Printf("  ACP Config: missing\n")
        }
        
        // Validate configuration
        validationErrors := validateProviderACPConfig(config)
        if len(validationErrors) > 0 {
            fmt.Printf("  Validation Errors:\n")
            for _, error := range validationErrors {
                fmt.Printf("    - %s\n", error)
            }
        } else {
            fmt.Printf("  Validation: ✅ passed\n")
        }
    }
}

func validateProviderACPConfig(config *providers.ProviderConfig) []string {
    var errors []string
    
    // Check required fields
    if _, ok := config.Features["supports_acp"]; !ok {
        errors = append(errors, "supports_acp feature not defined")
    }
    
    if acpConfig, ok := config.Features["acp_config"]; ok {
        acpMap := acpConfig.(map[string]interface{})
        
        if _, ok := acpMap["protocol_version"]; !ok {
            errors = append(errors, "acp_config.protocol_version not defined")
        }
        
        if _, ok := acpMap["max_tool_calls"]; !ok {
            errors = append(errors, "acp_config.max_tool_calls not defined")
        }
    }
    
    return errors
}
```

---

This comprehensive examples document provides practical demonstrations of ACP implementation, from basic detection to advanced integrations, helping developers understand and implement ACP support effectively in their projects.