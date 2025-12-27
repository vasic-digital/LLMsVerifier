# ACP (AI Coding Protocol) Implementation Guide

## Overview

This guide provides comprehensive documentation for implementing and testing ACP (AI Coding Protocol) support in the LLM Verifier project. ACP is an open protocol that standardizes communication between code editors and AI coding agents through JSON-RPC over stdio.

## What is ACP?

ACP (AI Coding Protocol) enables:
- **Editor Integration**: Seamless integration with editors like Zed, JetBrains IDEs, Avante.nvim, CodeCompanion.nvim
- **Tool Support**: Built-in tools, custom tools, and slash commands
- **MCP Compatibility**: Works with MCP servers configured in OpenCode
- **Context Management**: Project-specific rules and conversation history
- **Code Assistance**: Real-time code generation, completion, and error detection

## Implementation Architecture

### 1. Core Components

#### ACP Capability Detection (`testACPs` function)
```go
func (v *Verifier) testACPs(client *LLMClient, modelName string, ctx context.Context) bool
```

Tests five key ACP capabilities:
1. **JSON-RPC Protocol Comprehension**: Understanding JSON-RPC format
2. **Tool Calling Capability**: Ability to handle tool requests
3. **Context Management**: Multi-turn conversation retention
4. **Code Assistance**: Code generation and completion
5. **Error Detection**: Diagnostic and error resolution capabilities

#### Data Model Integration
```go
type FeatureDetectionResult struct {
    // ... existing fields ...
    MCPs             bool `json:"mcps"`
    LSPs             bool `json:"lsps"`
    ACPs             bool `json:"acps"`  // NEW FIELD
    Multimodal       bool `json:"multimodal"`
    // ... rest of fields ...
}
```

#### Database Schema Updates
```sql
ALTER TABLE verification_results ADD COLUMN supports_acps BOOLEAN DEFAULT 0;
ALTER TABLE models ADD COLUMN supports_acps BOOLEAN DEFAULT 0;
```

### 2. Provider Configuration

Each provider configuration includes ACP support:
```go
Features: map[string]interface{}{
    "supports_streaming": true,
    "supports_functions": true,
    "supports_vision":    true,
    "supports_acp":       true,  // ACP support flag
    "max_context_length": 128000,
    "supported_models":   []string{"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"},
}
```

### 3. Scoring Integration

ACP support contributes to experimental features scoring:
```go
totalExperimentalFeatures := 6 // MCPs, LSPs, ACPs, reranking, image generation, audio generation
if features.ACPs {
    experimentalFeatures++
}
breakdown.ExperimentalFeaturesScore = float64(experimentalFeatures) / float64(totalExperimentalFeatures) * 100
```

## Test Scenarios

### Test 1: JSON-RPC Protocol Comprehension
**Purpose**: Verify model understands JSON-RPC format
**Test**: Send JSON-RPC completion request
**Success Criteria**: Response contains JSON-RPC structure elements
```json
{
  "jsonrpc": "2.0",
  "method": "textDocument/completion",
  "params": {"textDocument": {"uri": "file:///test.py"}, "position": {"line": 0, "character": 10}},
  "id": 1
}
```

### Test 2: Tool Calling Capability
**Purpose**: Test tool integration abilities
**Test**: Request tool usage demonstration
**Success Criteria**: Response mentions tools or functions
```
Please demonstrate how you would call the "file_read" tool...
```

### Test 3: Context Management
**Purpose**: Test multi-turn conversation retention
**Test**: Multi-message conversation with context
**Success Criteria**: Model remembers previous context
```
Message 1: "Remember this project structure..."
Message 2: "Based on this structure, where should I..."
```

### Test 4: Code Assistance
**Purpose**: Test code generation capabilities
**Test**: Request code with specific requirements
**Success Criteria**: Generated code meets requirements
```
Write a Python function that:
1. Takes a list of user dictionaries
2. Validates email format
3. Includes error handling
4. Has type hints and docstring
```

### Test 5: Error Detection
**Purpose**: Test diagnostic capabilities
**Test**: Provide code with errors
**Success Criteria**: Model identifies and explains errors
```python
def process_user_data(users):
    valid_users = []
    for user in users:
        if user['email'].contains('@'):  # Error here
            valid_users.append(user)
    return valid_users
```

## Configuration Options

### Provider-Level Configuration
```json
{
  "name": "openai",
  "features": {
    "supports_acp": true,
    "acp_config": {
      "protocol_version": "2.0",
      "max_tool_calls": 10,
      "context_window_size": 128000,
      "supports_code_actions": true,
      "supports_diagnostics": true,
      "supports_completion": true
    }
  }
}
```

### Model-Level Configuration
```json
{
  "model_id": "gpt-4",
  "capabilities": {
    "acp": {
      "enabled": true,
      "features": ["jsonrpc", "tool_calling", "context_management", "code_assistance"]
    }
  }
}
```

## API Integration

### Validation Request
```json
POST /api/validate
{
  "model_name": "gpt-4",
  "supports_mcps": true,
  "supports_lsps": true,
  "supports_acps": true,  // ACP support
  "acp_features": {
    "jsonrpc_compliant": true,
    "tool_calling": true,
    "context_management": true,
    "code_assistance": true,
    "error_detection": true
  }
}
```

### Response Format
```json
{
  "model_name": "gpt-4",
  "verification_result": {
    "feature_detection": {
      "mcps": true,
      "lsps": true,
      "acps": true,  // ACP support confirmed
      "acp_score": 85.5
    },
    "overall_score": 92.3,
    "reliability": "high"
  }
}
```

## Testing Framework

### Unit Tests
```go
func TestACPsDetection(t *testing.T) {
    // Test ACP function signature and basic behavior
}

func TestACPsScoringIntegration(t *testing.T) {
    // Test ACP impact on scoring system
}
```

### Integration Tests
```go
func TestACPsWithRealProviders(t *testing.T) {
    // Test ACP detection with real LLM providers
}

func TestACPsDatabaseOperations(t *testing.T) {
    // Test ACP field persistence in database
}
```

### End-to-End Tests
```go
func TestACPsCompleteWorkflow(t *testing.T) {
    // Test complete ACP verification workflow
}

func TestACPsChallengeFramework(t *testing.T) {
    // Test ACP integration with challenge framework
}
```

### Performance Tests
```go
func TestACPsPerformanceBenchmark(t *testing.T) {
    // Benchmark ACP detection performance
}
```

### Security Tests
```go
func TestACPsSecurityValidation(t *testing.T) {
    // Test ACP input validation and security
}
```

## Performance Considerations

### Detection Time
- **Target**: < 5 seconds per model
- **Current**: ~2-3 seconds per model
- **Optimization**: Parallel testing of ACP capabilities

### Memory Usage
- **Baseline**: Minimal additional memory
- **Optimization**: Stream processing for large contexts

### Network Efficiency
- **Batch Processing**: Group ACP tests when possible
- **Caching**: Cache ACP results for identical models

## Security Considerations

### Input Validation
- Sanitize JSON-RPC requests
- Validate tool parameters
- Prevent code injection

### Rate Limiting
- Respect provider rate limits
- Implement request throttling
- Handle rate limit responses

### Error Handling
- Graceful degradation
- Secure error messages
- Audit logging

## Troubleshooting

### Common Issues

#### ACP Detection Returns False
**Symptoms**: Model shows ACP capabilities but test returns false
**Causes**: 
- Network timeouts
- Response parsing issues
- Strict success criteria

**Solutions**:
- Increase timeout values
- Check response format
- Adjust success thresholds

#### Database Errors
**Symptoms**: ACP fields not persisted correctly
**Causes**:
- Schema mismatch
- Data type issues
- Connection problems

**Solutions**:
- Verify database schema
- Check field mappings
- Test database connectivity

#### Performance Issues
**Symptoms**: ACP detection takes too long
**Causes**:
- Network latency
- Complex test scenarios
- Sequential processing

**Solutions**:
- Optimize test scenarios
- Implement parallel testing
- Add caching mechanisms

### Debug Mode
Enable debug logging for detailed ACP testing information:
```go
// Enable ACP debug logging
verifier.SetDebugMode(true)
verifier.SetACPDebugLevel("detailed")
```

## Best Practices

### Implementation
1. **Progressive Enhancement**: Start with basic ACP support, add advanced features gradually
2. **Provider Agnostic**: Design tests that work across different providers
3. **Error Resilience**: Handle failures gracefully without affecting other features
4. **Performance First**: Optimize for speed and efficiency

### Testing
1. **Comprehensive Coverage**: Test all ACP capabilities
2. **Real-world Scenarios**: Use realistic test cases
3. **Edge Cases**: Handle boundary conditions
4. **Regression Testing**: Ensure changes don't break existing functionality

### Maintenance
1. **Regular Updates**: Keep ACP tests current with protocol changes
2. **Provider Changes**: Monitor provider API updates
3. **Performance Monitoring**: Track ACP detection performance
4. **Documentation**: Keep documentation synchronized with implementation

## Future Enhancements

### Advanced Features
- **ACP 2.0 Support**: Next-generation protocol features
- **Custom Tools**: Provider-specific tool integration
- **Workspace Management**: Multi-project context handling
- **Collaborative Editing**: Real-time collaboration features

### Integration Improvements
- **IDE Plugins**: Direct editor integration
- **CI/CD Pipeline**: Automated ACP validation
- **Monitoring Dashboard**: Real-time ACP metrics
- **Analytics**: Usage patterns and insights

## Conclusion

The ACP implementation provides comprehensive support for AI Coding Protocol capabilities across the LLM Verifier ecosystem. With robust testing, proper configuration, and thorough documentation, this implementation enables accurate detection and scoring of ACP support in large language models.

For additional support or questions, refer to the project documentation or contact the development team.