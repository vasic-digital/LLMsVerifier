# AI Coding Protocol (ACP) Implementation Guide

## Overview

The AI Coding Protocol (ACP) is a standardized protocol for testing and verifying AI coding assistant capabilities. LLMsVerifier provides comprehensive tools for ACP verification.

## What is ACP?

ACP defines how AI coding assistants should:
- Understand and process JSON-RPC formatted requests
- Execute tool calls in development environments
- Manage context across coding sessions
- Provide code assistance (completions, refactoring, explanations)
- Detect and report errors accurately

## Verification Features

### 1. JSON-RPC Protocol Comprehension

Tests the model's ability to understand and respond to JSON-RPC 2.0 formatted requests:

```json
{
  "jsonrpc": "2.0",
  "method": "textDocument/completion",
  "params": {
    "textDocument": {"uri": "file:///path/to/file.go"},
    "position": {"line": 10, "character": 5}
  },
  "id": 1
}
```

### 2. Tool Calling Capability

Verifies the model can correctly invoke development tools:
- File operations (read, write, search)
- Code analysis tools
- Build and test commands
- Git operations

### 3. Context Management

Tests the model's ability to:
- Maintain context across multiple requests
- Handle large codebases
- Track file modifications
- Preserve conversation history

### 4. Code Assistance

Evaluates code assistance quality:
- Completion accuracy
- Refactoring suggestions
- Code explanations
- Bug detection

### 5. Error Detection

Verifies error handling capabilities:
- Syntax error detection
- Type error identification
- Runtime error prediction
- Security vulnerability detection

## Using the ACP CLI

### Installation

```bash
# Build from source
make build-acp

# Install system-wide
make install-acp
```

### Basic Verification

```bash
# Verify single model
acp-cli verify --model gpt-4 --provider openai

# With verbose output
acp-cli verify --model gpt-4 --provider openai --verbose

# JSON output for automation
acp-cli verify --model gpt-4 --provider openai --output json
```

### Batch Verification

```bash
# Multiple models
acp-cli batch --models gpt-4,claude-3-opus,deepseek-chat

# From file
acp-cli batch --models-file models.txt --concurrent 5

# With timeout
acp-cli batch --models gpt-4,claude-3-opus --timeout 60
```

### Continuous Monitoring

```bash
# Monitor for 1 hour, check every 5 minutes
acp-cli monitor --models gpt-4,claude-3-opus \
  --interval 300 \
  --duration 3600 \
  --alert-threshold 0.7
```

## Verification Scoring

ACP verification produces a score from 0.0 to 1.0:

| Score Range | Rating | Description |
|-------------|--------|-------------|
| 0.9 - 1.0 | Excellent | Full ACP support, ready for production |
| 0.8 - 0.9 | Good | Minor issues, suitable for most use cases |
| 0.7 - 0.8 | Acceptable | Some limitations, use with caution |
| 0.0 - 0.7 | Failed | Insufficient ACP support |

### Scoring Components

1. **Protocol Comprehension (25%)**: JSON-RPC understanding
2. **Tool Calling (25%)**: Correct tool invocation
3. **Context Handling (20%)**: Context management accuracy
4. **Code Quality (20%)**: Assistance quality
5. **Error Handling (10%)**: Error detection accuracy

## Configuration

### Environment Variables

```bash
export OPENAI_API_KEY="sk-your-key"
export ANTHROPIC_API_KEY="sk-ant-your-key"
export ACP_VERBOSE="true"
export ACP_TIMEOUT="30"
```

### Configuration File

```yaml
acp:
  enabled: true
  timeout: 30
  max_concurrent: 5
  retry_attempts: 3
  providers:
    - openai
    - anthropic
    - deepseek
  verification:
    protocol_weight: 0.25
    tool_calling_weight: 0.25
    context_weight: 0.20
    code_quality_weight: 0.20
    error_handling_weight: 0.10
```

## Integration with CI/CD

### GitHub Actions

```yaml
- name: Verify ACP Support
  run: |
    make build-acp
    ./bin/acp-cli batch --models ${{ env.MODELS }} --output json > results.json

- name: Check Results
  run: |
    failed=$(jq '[.[] | select(.supported == false)] | length' results.json)
    if [ "$failed" -gt 0 ]; then
      echo "ACP verification failed for $failed models"
      exit 1
    fi
```

### Pre-deployment Verification

```bash
#!/bin/bash
# verify-before-deploy.sh

CRITICAL_MODELS="gpt-4,claude-3-opus"

./bin/acp-cli batch --models "$CRITICAL_MODELS" --output json > acp-results.json

if jq -e '.[] | select(.supported == false)' acp-results.json > /dev/null; then
  echo "FAILED: Some critical models failed ACP verification"
  exit 1
fi

echo "SUCCESS: All critical models passed ACP verification"
```

## Troubleshooting

### Common Issues

**Model fails verification unexpectedly:**
- Increase timeout (`--timeout 60`)
- Check API rate limits
- Verify API key permissions

**Inconsistent results:**
- Run multiple times with `--retry 3`
- Check provider status
- Review model version

**Timeout errors:**
- Increase timeout setting
- Check network connectivity
- Consider using a faster model for testing

### Debug Mode

```bash
# Enable verbose logging
acp-cli verify --model gpt-4 --provider openai --verbose

# Save debug output
acp-cli verify --model gpt-4 --provider openai --verbose 2>&1 | tee debug.log
```

## Best Practices

1. **Regular Verification**: Run ACP verification on critical models periodically
2. **Alert Thresholds**: Set appropriate alert thresholds for monitoring
3. **CI Integration**: Include ACP verification in CI/CD pipelines
4. **Model Selection**: Use verified models for production ACP integrations
5. **Documentation**: Keep records of verification results

## Related Documentation

- [Protocol Overview](./README.md)
- [Model Verification Guide](../MODEL_VERIFICATION_GUIDE.md)
- [ACP CLI Reference](../../llm-verifier/cmd/acp-cli/README.md)
