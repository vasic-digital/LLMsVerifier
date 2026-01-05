# LLMsVerifier Protocol Documentation

This directory contains implementation guides for the protocols supported by LLMsVerifier.

## Supported Protocols

### 1. AI Coding Protocol (ACP)

The AI Coding Protocol enables structured communication between AI coding assistants and development environments.

**Documentation:**
- [ACP Implementation Guide](./ACP_GUIDE.md) - Complete guide for ACP testing and verification
- [ACP CLI Reference](../../llm-verifier/cmd/acp-cli/README.md) - Command-line tool for ACP testing

**Key Features:**
- JSON-RPC protocol comprehension testing
- Tool calling capability verification
- Context management validation
- Code assistance evaluation
- Error detection testing

### 2. Model Context Protocol (MCP)

The Model Context Protocol provides a standardized way to manage context across LLM interactions.

**Key Capabilities Tested:**
- Context window management
- Multi-turn conversation handling
- Token limit awareness
- Context compression techniques

### 3. Language Server Protocol (LSP)

The Language Server Protocol integration enables code intelligence features in development environments.

**Key Capabilities Tested:**
- Code completion accuracy
- Diagnostic reporting
- Go-to-definition support
- Symbol resolution

## Protocol Verification

LLMsVerifier validates that models correctly support these protocols through the verification system:

```bash
# Verify ACP support
./bin/acp-cli verify --model gpt-4 --provider openai

# Batch verify multiple models
./bin/acp-cli batch --models gpt-4,claude-3-opus,deepseek-chat

# Monitor protocol support over time
./bin/acp-cli monitor --models gpt-4,claude-3-opus --interval 300
```

## Integration

These protocols are verified as part of the model verification process:

```yaml
model_verification:
  enabled: true
  protocols:
    acp:
      enabled: true
      strict_mode: true
    mcp:
      enabled: true
      context_window_check: true
    lsp:
      enabled: true
      code_intelligence_check: true
```

## Related Documentation

- [Model Verification Guide](../MODEL_VERIFICATION_GUIDE.md)
- [API Documentation](../API_DOCUMENTATION_UPDATED.md)
- [Complete System Documentation](../COMPLETE_SYSTEM_DOCUMENTATION.md)
