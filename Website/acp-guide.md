# AI Coding Protocol (ACP) Guide

## Overview

The AI Coding Protocol (ACP) is an open protocol that standardizes communication between code editors and AI coding agents through JSON-RPC over stdio. LLM Verifier includes comprehensive ACP support detection to help you identify which models are compatible with ACP-enabled editors and development environments.

## What is ACP?

ACP enables seamless integration between your favorite code editors and AI coding assistants by providing:

- **Standardized Communication**: JSON-RPC protocol for reliable editor-LLM messaging
- **Tool Integration**: Support for built-in tools, custom tools, and slash commands  
- **Context Management**: Project-specific rules and conversation history retention
- **Real-time Assistance**: Live code completion, error detection, and refactoring suggestions
- **Multi-editor Support**: Works with Zed, JetBrains IDEs, Avante.nvim, CodeCompanion.nvim, and more

## Supported Editors

### Zed
```json
{
  "language_models": {
    "opencode": {
      "provider": {
        "type": "acp",
        "command": "opencode acp"
      }
    }
  }
}
```

### JetBrains IDEs
```json
{
  "acp.servers": [{
    "name": "OpenCode",
    "command": "opencode acp"
  }]
}
```

### Avante.nvim
```lua
{
  provider = "acp",
  acp = {
    command = "opencode acp"
  }
}
```

### CodeCompanion.nvim
```lua
{
  adapters = {
    acp = function()
      return require("codecompanion.adapters").use("acp", {
        command = "opencode acp"
      })
    end,
  }
}
```

## ACP Capability Detection

LLM Verifier automatically tests for ACP support using five key capabilities:

### 1. JSON-RPC Protocol Comprehension
Tests if the model understands and can respond to JSON-RPC format messages used for editor communication.

### 2. Tool Calling Capability  
Verifies the model can handle tool requests and integrate with editor tools like file operations and terminal commands.

### 3. Context Management
Ensures the model maintains conversation context across multiple turns, remembering project structure and previous interactions.

### 4. Code Assistance
Validates the model's ability to provide helpful code generation, completion, and suggestions within the editor context.

### 5. Error Detection
Tests the model's capability to identify and explain code errors, providing diagnostic information like a language server.

## Configuration

### Provider Configuration
Enable ACP support in your provider configuration:

```json
{
  "name": "openai",
  "features": {
    "supports_acp": true,
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

### Model-Specific Settings
Configure ACP features per model:

```yaml
models:
  - id: "gpt-4"
    capabilities:
      acp:
        enabled: true
        features:
          - jsonrpc_compliance
          - tool_calling
          - context_management
          - code_assistance
          - error_detection
```

## Testing ACP Support

### Using the Web Interface
1. Navigate to the model verification page
2. Select models to test
3. Enable "ACP Support" in test options
4. Run verification
5. View ACP compatibility results

### Using the API
```bash
curl -X POST http://localhost:8080/api/verify/acp \
  -H "Content-Type: application/json" \
  -d '{
    "model_name": "gpt-4",
    "provider": "openai",
    "test_scenarios": [
      {"name": "jsonrpc_compliance", "enabled": true},
      {"name": "tool_calling", "enabled": true},
      {"name": "context_management", "enabled": true},
      {"name": "code_assistance", "enabled": true},
      {"name": "error_detection", "enabled": true}
    ]
  }'
```

### Using the CLI
```bash
# Test ACP support for specific model
llm-verifier verify-acp --model gpt-4 --provider openai

# Test multiple models
llm-verifier verify-acp-batch --models gpt-4,claude-3-opus,deepseek-chat
```

## Interpreting Results

### ACP Score
Models receive an ACP score from 0.0 to 1.0 based on their performance across all five capability tests:

- **0.8-1.0**: Excellent ACP support - Full editor integration recommended
- **0.6-0.79**: Good ACP support - Basic editor integration possible  
- **0.4-0.59**: Limited ACP support - Partial functionality available
- **Below 0.4**: Minimal ACP support - Not recommended for editor integration

### Capability Breakdown
Each ACP capability is scored individually:

```json
{
  "acp_capabilities": {
    "jsonrpc_compliance": 0.9,
    "tool_calling": 0.8,
    "context_management": 0.85,
    "code_assistance": 0.9,
    "error_detection": 0.75
  },
  "overall_acp_score": 0.84
}
```

## Best Practices

### For Editor Integration
1. **Choose High-Scoring Models**: Select models with ACP scores above 0.7 for best experience
2. **Test in Your Environment**: Verify ACP support works with your specific editor setup
3. **Configure Timeouts**: Set appropriate timeouts for your use case (typically 30-60 seconds)
4. **Monitor Performance**: Track ACP test duration and optimize as needed

### For Model Selection
1. **Prioritize ACP Support**: When choosing between similar models, prefer those with better ACP scores
2. **Consider Context Requirements**: Models with strong context management are better for complex projects
3. **Evaluate Tool Integration**: If you need specific tools, verify tool calling capability scores
4. **Balance Features**: Consider ACP support alongside other capabilities like code generation and reasoning

### For Development Teams
1. **Standardize Testing**: Use consistent ACP testing across your model evaluation process
2. **Document Requirements**: Clearly specify ACP requirements for your editor integrations
3. **Version Control**: Track ACP scores across model versions to identify regressions
4. **Automate Monitoring**: Set up automated ACP verification in your CI/CD pipeline

## Integration Examples

### VS Code Extension
```javascript
// Check ACP support before enabling features
const acpResult = await llmVerifier.verifyACP(model, provider);
if (acpResult.supported && acpResult.score > 0.7) {
    enableACPFeatures();
} else {
    showFallbackFeatures();
}
```

### JetBrains Plugin
```kotlin
// Use ACP results for feature enablement
val acpResult = LLMVerifier.verifyACP(modelId, provider)
if (acpResult.acpSupported && acpResult.overallScore >= 0.7) {
    ACPIntegration.enable()
} else {
    showStandardCompletion()
}
```

### Vim/Neovim Plugin
```vim
" Check ACP compatibility
let acp_result = LLMVerifierVerifyACP(g:llm_model, g:llm_provider)
if acp_result.supported && acp_result.score > 0.7
    call EnableACPFeatures()
else
    call ShowWarning("ACP not fully supported")
endif
```

## Troubleshooting

### Common Issues

**ACP Detection Returns False**
- Check if the model actually supports conversational context
- Verify the model responds appropriately to JSON-RPC format
- Ensure sufficient context is provided in test messages
- Consider increasing timeout for slower models

**Low ACP Scores**
- Review individual capability scores to identify weak areas
- Test with different prompt variations
- Check if model version affects ACP capabilities
- Compare with similar models from the same provider

**Inconsistent Results**
- Run multiple tests and average results
- Check for rate limiting or temporary issues
- Verify API key permissions and rate limits
- Monitor for provider-specific behavior changes

### Performance Optimization

**Slow ACP Tests**
- Reduce test complexity while maintaining coverage
- Implement parallel testing for multiple capabilities
- Cache results for models that don't change frequently
- Use connection pooling for API requests

**High Resource Usage**
- Implement request batching where possible
- Use efficient JSON parsing libraries
- Monitor memory usage during large-scale testing
- Consider implementing test result caching

## Advanced Configuration

### Custom ACP Tests
Extend ACP testing with custom scenarios:

```go
func customACPTest(client *LLMClient, modelName string) bool {
    // Implement custom ACP capability test
    // Return true if model supports the custom capability
}
```

### Dynamic Thresholds
Adjust ACP score thresholds based on your requirements:

```json
{
  "acp_thresholds": {
    "minimum_score": 0.6,
    "recommended_score": 0.8,
    "excellent_score": 0.9
  }
}
```

### Weighted Scoring
Customize importance of different ACP capabilities:

```json
{
  "acp_weights": {
    "jsonrpc_compliance": 0.25,
    "tool_calling": 0.20,
    "context_management": 0.20,
    "code_assistance": 0.20,
    "error_detection": 0.15
  }
}
```

## Security Considerations

### Input Validation
- All ACP test inputs are sanitized and validated
- Malicious patterns are detected and handled safely
- Injection attempts are prevented through proper escaping

### Data Privacy
- Sensitive information is not included in test messages
- API keys and credentials are never exposed in responses
- Test results are stored securely with appropriate access controls

### Rate Limiting
- ACP tests respect provider rate limits
- Concurrent testing is controlled to prevent abuse
- Backoff strategies are implemented for rate limit scenarios

## API Reference

### Endpoints

#### Verify ACP Support
```
POST /api/verify/acp
```

#### Get ACP Results
```
GET /api/results/acp/{model_id}
```

#### List ACP-Compatible Models
```
GET /api/models/acp?min_score=0.7
```

### Response Format
```json
{
  "success": true,
  "model_id": "gpt-4",
  "acp_support": {
    "supported": true,
    "score": 0.85,
    "confidence": 0.9,
    "capabilities": {
      "jsonrpc_compliance": {"supported": true, "score": 0.9},
      "tool_calling": {"supported": true, "score": 0.8},
      "context_management": {"supported": true, "score": 0.85},
      "code_assistance": {"supported": true, "score": 0.9},
      "error_detection": {"supported": true, "score": 0.75}
    }
  }
}
```

## Support

For ACP-related support:
- **Documentation**: See the main [documentation portal](../llm-verifier/docs/)
- **Issues**: Report ACP-specific issues on [GitHub](https://github.com/vasic-digital/LLMsVerifier/issues)
- **Community**: Join discussions in [GitHub Discussions](https://github.com/vasic-digital/LLMsVerifier/discussions)
- **Enterprise**: Contact enterprise support for custom ACP integrations

---

**Next Steps**: 
- [Try ACP Testing](../llm-verifier/docs/end-user-manual.md#acp-testing)
- [Configure Your Editor](../llm-verifier/docs/administrator-manual.md#editor-integration)
- [API Integration Guide](../llm-verifier/docs/api-reference-manual.md#acp-api)