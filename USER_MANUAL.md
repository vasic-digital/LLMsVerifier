# LLM Verifier User Manual

## Getting Started with LLM Verifier

Welcome to LLM Verifier! This comprehensive guide will help you get started with model verification, configuration export, and integration with various AI platforms.

### Table of Contents
1. [Installation and Setup](#installation-and-setup)
2. [First Model Verification](#first-model-verification)
3. [Understanding Results](#understanding-results)
4. [Configuration Export](#configuration-export)
5. [OpenCode Integration](#opencode-integration)
6. [Troubleshooting](#troubleshooting)
7. [Advanced Features](#advanced-features)

---

## Installation and Setup

### Prerequisites

**System Requirements:**
- Go 1.21 or later
- Linux/macOS/Windows
- 2GB RAM minimum, 4GB recommended
- Internet connection for API verification

**API Keys Required:**
- OpenAI API key (for GPT models)
- Anthropic API key (for Claude models)
- Google AI API key (for Gemini models)
- Other provider keys as needed

### Installation Steps

1. **Clone the repository:**
   ```bash
   git clone https://github.com/your-org/llm-verifier.git
   cd llm-verifier
   ```

2. **Build the application:**
   ```bash
   go build ./cmd/main.go -o llm-verifier
   ```

3. **Set up environment variables:**
   ```bash
   # Create .env file
   cat > .env << EOF
   OPENAI_API_KEY=sk-your-openai-key-here
   ANTHROPIC_API_KEY=sk-ant-your-anthropic-key-here
   GOOGLE_API_KEY=your-google-ai-key-here
   EOF
   ```

4. **Verify installation:**
   ```bash
   ./llm-verifier --version
   ```

---

## First Model Verification

### Basic Verification Command

Run your first model verification:

```bash
./llm-verifier verify \
  --provider openai \
  --model gpt-4o \
  --endpoint https://api.openai.com/v1
```

### Batch Verification

Verify multiple models at once:

```bash
./llm-verifier verify \
  --config config.yaml \
  --parallel 3 \
  --timeout 30s
```

### Configuration File Example

Create a `config.yaml` file:

```yaml
providers:
  - name: openai
    api_key: ${OPENAI_API_KEY}
    models:
      - gpt-4o
      - gpt-4-turbo
      - gpt-3.5-turbo

  - name: anthropic
    api_key: ${ANTHROPIC_API_KEY}
    models:
      - claude-3-5-sonnet-20241022
      - claude-3-opus-20240229

verification:
  parallel_requests: 3
  timeout: 30s
  retries: 3
```

---

## Understanding Results

### Verification Report Structure

After verification, LLM Verifier generates a comprehensive report:

```
📊 VERIFICATION RESULTS SUMMARY
═══════════════════════════════════════════════

Total Models Tested: 5
Successful Verifications: 4
Failed Verifications: 1
Average Response Time: 2.3s

📈 PERFORMANCE SCORES
═══════════════════════════════════════════════

1. gpt-4o (OpenAI)
   ├── Overall Score: 95.2/100
   ├── Code Capability: 96.0/100
   ├── Responsiveness: 94.0/100
   ├── Reliability: 97.0/100
   └── Feature Richness: 93.0/100

2. claude-3-5-sonnet (Anthropic)
   ├── Overall Score: 92.8/100
   └── ...
```

### Score Interpretation

- **90-100**: Excellent - Production ready
- **80-89**: Good - Suitable for most use cases
- **70-79**: Fair - May have limitations
- **60-69**: Poor - Not recommended for production
- **<60**: Critical issues - Avoid use

### Detailed Metrics

**Code Capability**: Ability to generate, analyze, and debug code
**Responsiveness**: API response times and latency
**Reliability**: Consistency of responses and error rates
**Feature Richness**: Support for advanced features (streaming, tools, etc.)

---

## Configuration Export

### Export to OpenCode (Recommended)

Export verified models to OpenCode format:

```bash
./llm-verifier export-config opencode \
  --output opencode-config.json \
  --include-api-keys \
  --min-score 80
```

### Export to Other Formats

```bash
# Export to Crush format
./llm-verifier export-config crush \
  --output crush-config.json

# Export to Claude Code format
./llm-verifier export-config claude-code \
  --output claude-config.json
```

### Export Options

- `--include-api-keys`: Include API keys in export (⚠️  secure handling required)
- `--min-score <score>`: Only export models with score above threshold
- `--format <format>`: Export format (opencode, crush, claude-code)
- `--output <file>`: Output file path

---

## OpenCode Integration

### What is OpenCode?

OpenCode is a modern AI coding assistant that supports multiple LLM providers. LLM Verifier generates optimized configurations for seamless OpenCode integration.

### Integration Steps

1. **Export OpenCode configuration:**
   ```bash
   ./llm-verifier export-config opencode \
     --output ~/.opencode/config.json \
     --include-api-keys
   ```

2. **Secure the configuration:**
   ```bash
   chmod 600 ~/.opencode/config.json
   ```

3. **Launch OpenCode:**
   ```bash
   opencode
   ```

### ProviderInitError Resolution

If you encounter ProviderInitError with OpenCode:

1. **Check configuration schema:**
   ```json
   {
     "$schema": "./opencode-schema.json",
     "providers": {
       "openai": {
         "apiKey": "sk-...",
         "disabled": false,
         "provider": "openai"
       }
     }
   }
   ```

2. **Verify no old format elements:**
   - ❌ No `"provider"` (singular) section
   - ❌ No npm package references
   - ✅ `"providers"` (plural) section exists
   - ✅ `"agents"` section exists

3. **Use migration tool for old configs:**
   ```bash
   ./llm-verifier migrate-config \
     --input old-config.json \
     --output new-config.json
   ```

---

## Troubleshooting

### Common Issues and Solutions

#### 1. API Key Authentication Errors

**Problem:** `401 Unauthorized` or `403 Forbidden`

**Solutions:**
- Verify API key is correct and active
- Check API key format (usually starts with `sk-`)
- Ensure API key has sufficient permissions
- Check API quota/limits

#### 2. Network Connectivity Issues

**Problem:** `connection timeout` or `network unreachable`

**Solutions:**
- Check internet connectivity
- Verify API endpoints are accessible
- Try different network/proxy settings
- Check firewall settings

#### 3. ProviderInitError in OpenCode

**Problem:** OpenCode fails to initialize providers

**Solutions:**
- Export fresh configuration with LLM Verifier
- Verify configuration uses correct schema
- Check that providers section exists
- Ensure API keys are properly formatted

#### 4. Performance Issues

**Problem:** Verification takes too long or fails

**Solutions:**
- Reduce parallel requests: `--parallel 1`
- Increase timeout: `--timeout 60s`
- Check system resources (CPU, memory)
- Try during off-peak hours

#### 5. Configuration Export Errors

**Problem:** Export fails or generates invalid config

**Solutions:**
- Ensure write permissions on output directory
- Check available disk space
- Verify JSON syntax if using custom templates
- Try without `--include-api-keys` first

### Debug Mode

Enable detailed logging:

```bash
export LLM_VERIFIER_DEBUG=true
./llm-verifier verify --debug
```

### Getting Help

- **Documentation:** Check this manual and API docs
- **Logs:** Review application logs for error details
- **GitHub Issues:** Report bugs with full error logs
- **Community:** Join our Discord/Slack for support

---

## Advanced Features

### Custom Verification Tests

Create custom verification scenarios:

```go
// Custom test example
func customCodeGenerationTest(model Model) VerificationResult {
    prompt := "Write a function to calculate fibonacci numbers in Go"

    response := model.SendMessage(prompt)
    score := evaluateCodeQuality(response)

    return VerificationResult{
        ModelInfo: model.Info,
        PerformanceScores: PerformanceScore{
            CodeCapability: score,
        },
    }
}
```

### Batch Processing

Process large numbers of models:

```bash
# Process 100 models with optimized settings
./llm-verifier verify \
  --batch-size 10 \
  --parallel 5 \
  --timeout 45s \
  --retries 5
```

### Performance Monitoring

Monitor verification performance:

```bash
./llm-verifier verify \
  --metrics \
  --output-metrics metrics.json
```

### Integration with CI/CD

Automate verification in CI pipelines:

```yaml
# GitHub Actions example
- name: Verify LLM Models
  run: |
    ./llm-verifier verify \
      --config ci-config.yaml \
      --junit-report results.xml

- name: Upload Results
  uses: actions/upload-artifact@v3
  with:
    name: verification-results
    path: results.xml
```

---

## Security Best Practices

### API Key Management

1. **Never commit API keys to version control**
2. **Use environment variables or secure vaults**
3. **Rotate keys regularly**
4. **Limit key permissions to minimum required**

### Configuration Security

1. **Set restrictive file permissions on configs:**
   ```bash
   chmod 600 config.json
   ```

2. **Use secure storage for sensitive configurations**
3. **Avoid sharing configurations with API keys**

### Network Security

1. **Use HTTPS for all API communications**
2. **Implement rate limiting**
3. **Monitor for unusual API usage patterns**
4. **Use VPNs for sensitive operations**

---

## Performance Optimization

### Hardware Requirements

- **CPU**: 4+ cores recommended for parallel verification
- **RAM**: 8GB+ for large model batches
- **Storage**: 10GB+ for logs and results
- **Network**: Stable high-speed internet

### Optimization Tips

1. **Parallel Processing:**
   ```bash
   ./llm-verifier verify --parallel $(nproc)
   ```

2. **Batch Processing:**
   ```bash
   ./llm-verifier verify --batch-size 50
   ```

3. **Selective Verification:**
   ```bash
   ./llm-verifier verify --providers openai,anthropic
   ```

4. **Caching:**
   ```bash
   ./llm-verifier verify --cache-results
   ```

---

## API Reference

### Command Line Interface

#### Global Options
- `--debug`: Enable debug logging
- `--verbose`: Verbose output
- `--config <file>`: Configuration file path
- `--output <file>`: Output file path

#### Verification Commands
```bash
llm-verifier verify [options]
llm-verifier export-config <format> [options]
llm-verifier migrate-config [options]
llm-verifier analytics [options]
```

#### Configuration Commands
```bash
llm-verifier config validate [file]
llm-verifier config migrate [input] [output]
llm-verifier config backup [destination]
```

---

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

### Testing

Run the full test suite:

```bash
go test ./... -v -race -cover
```

### Code Standards

- Follow Go naming conventions
- Add documentation for public APIs
- Write comprehensive tests
- Use meaningful commit messages

---

## Changelog

### Version 2.0.0 (Current)
- ✅ **ProviderInitError Resolution**: Fixed OpenCode configuration compatibility
- ✅ **Enhanced Provider Detection**: Support for 20+ AI providers
- ✅ **Intelligent Model Selection**: Automated agent model assignment
- ✅ **Migration Tools**: Convert old configurations to new format
- ✅ **Analytics & Monitoring**: Comprehensive usage tracking
- ✅ **Comprehensive Testing**: 100% test coverage for core features

### Version 1.5.0
- Basic model verification functionality
- Support for major AI providers
- Configuration export capabilities

---

## Support and Community

- **Documentation**: [docs.llm-verifier.dev](https://docs.llm-verifier.dev)
- **GitHub**: [github.com/your-org/llm-verifier](https://github.com/your-org/llm-verifier)
- **Discord**: [discord.gg/llm-verifier](https://discord.gg/llm-verifier)
- **Twitter**: [@LLMVerifier](https://twitter.com/LLMVerifier)

---

**LLM Verifier v2.0.0** - Making AI model verification simple, reliable, and production-ready.