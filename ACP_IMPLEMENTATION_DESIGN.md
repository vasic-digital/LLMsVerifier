# ACP (AI Coding Protocol) Implementation Design

## Overview
This document outlines the design for implementing ACP (AI Coding Protocol) support in the LLM Verifier project, following the same patterns as MCP (Model Context Protocol) implementation.

## Data Model Updates

### 1. FeatureDetectionResult Structure
Add ACP support field to the existing FeatureDetectionResult struct:

```go
type FeatureDetectionResult struct {
    // ... existing fields ...
    MCPs             bool                 `json:"mcps"`
    LSPs             bool                 `json:"lsps"`
    ACPs             bool                 `json:"acps"`  // NEW FIELD
    Multimodal       bool                 `json:"multimodal"`
    // ... rest of fields ...
}
```

### 2. Database Schema Updates
Add ACP support columns to verification_results table:

```sql
ALTER TABLE verification_results ADD COLUMN supports_acps BOOLEAN DEFAULT 0;
ALTER TABLE models ADD COLUMN supports_acps BOOLEAN DEFAULT 0;
```

### 3. API Validation Updates
Add ACP fields to validation structures:

```go
type ValidationRequest struct {
    // ... existing fields ...
    SupportsMCPs             bool       `json:"supports_mcps"`
    SupportsACPs             bool       `json:"supports_acps"`  // NEW FIELD
    // ... rest of fields ...
}
```

## ACP Capability Detection Framework

### 1. Core ACP Test Function
Implement comprehensive ACP testing function:

```go
func (v *Verifier) testACPs(client *LLMClient, modelName string, ctx context.Context) bool {
    // Test ACP protocol compliance
    // Test JSON-RPC comprehension
    // Test tool calling capabilities
    // Test context management
    // Test code assistance features
}
```

### 2. ACP Test Scenarios

#### Test 1: JSON-RPC Protocol Comprehension
```go
// Test if model understands JSON-RPC format
req1 := ChatCompletionRequest{
    Model: modelName,
    Messages: []Message{
        {
            Role: "user",
            Content: `You are an ACP-compatible AI coding agent. Please respond to this JSON-RPC request:
{"jsonrpc":"2.0","method":"textDocument/completion","params":{"textDocument":{"uri":"file:///test.py"},"position":{"line":0,"character":10}},"id":1}

What would be an appropriate response for a code completion request?`,
        },
    },
}
```

#### Test 2: Tool Calling Capability
```go
// Test if model can handle tool calling
req2 := ChatCompletionRequest{
    Model: modelName,
    Messages: []Message{
        {
            Role: "user",
            Content: `As an ACP agent, you have access to tools. Please call the "file_read" tool to read the content of a Python file and then suggest improvements.`,
        },
    },
}
```

#### Test 3: Context Management
```go
// Test context retention for multi-turn conversations
req3 := ChatCompletionRequest{
    Model: modelName,
    Messages: []Message{
        {
            Role: "user",
            Content: "I'm working on a Python project with the following structure: src/main.py, tests/test_main.py, requirements.txt. Remember this structure.",
        },
        {
            Role: "assistant",
            Content: "I've noted your Python project structure: src/main.py, tests/test_main.py, requirements.txt.",
        },
        {
            Role: "user",
            Content: "Based on this structure, where should I add a new utility module?",
        },
    },
}
```

#### Test 4: Code Assistance
```go
// Test code generation and assistance capabilities
req4 := ChatCompletionRequest{
    Model: modelName,
    Messages: []Message{
        {
            Role: "user",
            Content: `As an ACP coding agent, help me write a Python function that:
1. Takes a list of numbers
2. Returns the sum of all even numbers
3. Includes proper error handling
4. Has type hints and docstring`,
        },
    },
}
```

### 3. Scoring Integration
Update the scoring system to include ACP support:

```go
// In verifier.go scoring section
totalExperimentalFeatures := 6 // MCPs, LSPs, ACPs, reranking, image generation, audio generation
if features.ACPs {
    experimentalFeatures++
}
breakdown.ExperimentalFeaturesScore = float64(experimentalFeatures) / float64(totalExperimentalFeatures) * 100
```

## Provider Configuration Updates

### 1. Add ACP Features to Provider Configs
Update provider configurations to include ACP support:

```go
Features: map[string]interface{}{
    "supports_streaming": true,
    "supports_functions": true,
    "supports_vision":    true,
    "supports_acp":       true,  // NEW FIELD
    "max_context_length": 128000,
    "supported_models":   []string{"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"},
},
```

### 2. ACP-Specific Configuration
Add ACP-specific configuration options:

```go
type ACPConfig struct {
    Enabled              bool   `json:"enabled"`
    ProtocolVersion      string `json:"protocol_version"`
    MaxToolCalls         int    `json:"max_tool_calls"`
    ContextWindowSize    int    `json:"context_window_size"`
    SupportsCodeActions  bool   `json:"supports_code_actions"`
    SupportsDiagnostics  bool   `json:"supports_diagnostics"`
    SupportsCompletion   bool   `json:"supports_completion"`
}
```

## Testing Framework

### 1. Unit Tests
- Test ACP function signatures
- Test ACP configuration validation
- Test ACP scoring integration

### 2. Integration Tests
- Test ACP with real LLM providers
- Test ACP database operations
- Test ACP API endpoints

### 3. End-to-End Tests
- Test complete ACP verification workflow
- Test ACP reporting and analytics
- Test ACP challenge framework

### 4. Performance Tests
- Measure ACP detection performance
- Test ACP with large context windows
- Benchmark ACP tool calling

### 5. Security Tests
- Test ACP input validation
- Test ACP authentication
- Test ACP authorization

## Documentation and Examples

### 1. ACP Implementation Guide
- Step-by-step ACP implementation
- Configuration examples
- Best practices

### 2. ACP API Documentation
- API endpoint documentation
- Request/response examples
- Error handling

### 3. Video Course Content
- ACP overview and concepts
- Implementation walkthrough
- Real-world examples

### 4. Website Updates
- ACP feature page
- ACP compatibility matrix
- ACP usage examples

## Implementation Phases

### Phase 1: Core Implementation (Week 1)
1. Update data models
2. Implement ACP test function
3. Update database schema
4. Basic ACP detection

### Phase 2: Integration (Week 2)
1. Provider configuration updates
2. API validation updates
3. Scoring system integration
4. Reporting updates

### Phase 3: Testing (Week 3)
1. Unit tests
2. Integration tests
3. End-to-end tests
4. Performance tests

### Phase 4: Documentation (Week 4)
1. Implementation guides
2. API documentation
3. Video course content
4. Website updates

## Success Metrics
- ACP support detected for 95%+ of tested LLMs
- All test types passing (unit, integration, e2e, performance, security)
- Complete documentation coverage
- Video course materials produced
- Website fully updated
- Full automation workflows implemented