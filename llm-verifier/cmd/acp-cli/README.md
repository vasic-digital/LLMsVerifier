# ACP CLI - AI Coding Protocol Testing Tool

A powerful command-line tool for testing AI Coding Protocol (ACP) support in Large Language Models.

## Installation

### Build from source
```bash
# Install dependencies
make setup

# Build ACP CLI
make build-acp

# The binary will be available at bin/acp-cli
```

### Install system-wide
```bash
make install-acp
# Installs to /usr/local/bin/acp-cli
```

### Build for multiple platforms
```bash
make build-acp-all
# Creates binaries for Linux, macOS, and Windows
```

## Usage

### Verify ACP support for a single model
```bash
acp-cli verify --model gpt-4 --provider openai

# With custom timeout
acp-cli verify --model gpt-4 --provider openai --timeout 60

# JSON output
acp-cli verify --model gpt-4 --provider openai --output json
```

### Batch verify multiple models
```bash
# Comma-separated list
acp-cli batch --models gpt-4,gpt-3.5-turbo,claude-3-opus

# From file (one model per line)
echo -e "gpt-4\ngpt-3.5-turbo\nclaude-3-opus" > models.txt
acp-cli batch --models-file models.txt

# Concurrent testing (faster for many models)
acp-cli batch --models gpt-4,gpt-3.5-turbo,claude-3-opus --concurrent 5

# Custom timeout
acp-cli batch --models gpt-4,gpt-3.5-turbo,claude-3-opus --timeout 45
```

### List supported providers
```bash
acp-cli providers

# JSON output for scripting
acp-cli providers --output json
```

### Monitor ACP support over time
```bash
# Continuous monitoring for 1 hour, checking every 5 minutes
acp-cli monitor --models gpt-4,claude-3-opus --interval 300 --duration 3600

# With alert threshold (alert if score drops below 0.7)
acp-cli monitor --models gpt-4,claude-3-opus --alert-threshold 0.7
```

## Command Reference

### `verify` - Test single model
```bash
acp-cli verify --model MODEL --provider PROVIDER [flags]

Flags:
  -m, --model string       Model name to test (required)
  -p, --provider string    Provider name (required)
  -t, --timeout int        Timeout in seconds (default 30)
  -o, --output string      Output format: table, json (default "table")
  -v, --verbose            Verbose output
```

### `batch` - Test multiple models
```bash
acp-cli batch [flags]

Flags:
      --models string        Comma-separated list of models to test
      --models-file string   File containing models to test (one per line)
  -c, --concurrent int       Number of concurrent tests (default 5)
  -t, --timeout int          Timeout in seconds (default 30)
  -o, --output string        Output format: table, json (default "table")
```

### `providers` - List supported providers
```bash
acp-cli providers [flags]

Flags:
  -o, --output string        Output format: table, json (default "table")
```

### `monitor` - Monitor ACP support
```bash
acp-cli monitor --models MODELS [flags]

Flags:
      --models string        Comma-separated list of models to monitor
  -i, --interval int         Monitoring interval in seconds (default 300)
  -d, --duration int         Total monitoring duration in seconds (default 3600)
      --alert-threshold      ACP score threshold for alerts (default 0.7)
```

### `config` - Manage configuration
```bash
# Initialize configuration
acp-cli config init

# Show current configuration
acp-cli config show

# Validate configuration
acp-cli config validate
```

## Output Formats

### Table Format (default)
```
ACP Verification Results:
--------------------------------------------------
Model:    gpt-4
Provider: openai
Result:   ✅ SUPPORTED
--------------------------------------------------

ACP Features:
  ✓ JSON-RPC Protocol Comprehension
  ✓ Tool Calling Capability
  ✓ Context Management
  ✓ Code Assistance
  ✓ Error Detection

This model is fully compatible with ACP-enabled editors!
Timestamp: 2025-12-27T21:29:18Z
```

### JSON Format
```json
{
  "model": "gpt-4",
  "provider": "openai",
  "supported": true,
  "timestamp": "2025-12-27T21:29:18Z"
}
```

## Configuration

### Environment Variables
```bash
# OpenAI Configuration
export OPENAI_API_KEY="sk-your-api-key"

# Anthropic Configuration
export ANTHROPIC_API_KEY="sk-ant-your-api-key"

# General Configuration
export ACP_VERBOSE="true"
export ACP_CONCURRENT="10"
```

### Configuration File
Create `acp-config.json`:
```json
{
  "acp": {
    "enabled": true,
    "timeout": 30,
    "max_concurrent": 5,
    "retry_attempts": 3,
    "providers": ["openai", "anthropic", "deepseek", "google"]
  }
}
```

## Examples

### Continuous Integration
```bash
#!/bin/bash
# verify-acp-models.sh

# Test critical models
acp-cli batch --models gpt-4,claude-3-opus,deepseek-chat \
  --output json > acp-results.json

# Check if all models passed
if jq -e '.[] | select(.supported == false)' acp-results.json > /dev/null; then
  echo "❌ Some models failed ACP verification"
  exit 1
fi

echo "✅ All models passed ACP verification"
```

### Monitoring ACP Health
```bash
#!/bin/bash
# monitor-acp-health.sh

acp-cli monitor --models gpt-4,claude-3-opus,deepseek-chat \
  --interval 300 \
  --duration 7200 \
  --alert-threshold 0.6 \
  --output json | tee acp-monitor.log

# Send alerts if issues found
grep "ALERT" acp-monitor.log | mail -s "ACP Health Alert" admin@example.com
```

### Generating Reports
```bash
# Run batch test
acp-cli batch --models-file all-models.txt --output json > acp-report.json

# Generate summary
jq -r '.[] | "\(.model): \(.supported | if . then \"✅\" else \"❌\" end)"' acp-report.json

# Count supported models
cat acp-report.json | jq '[.[] | select(.supported == true)] | length'
```

## Testing

### Run ACP CLI tests
```bash
# Run all ACP tests
make test-acp

# Run quick test
make run-acp-test

# Run batch test
make run-acp-batch
```

## Development

### Building from source
```bash
cd llm-verifier/cmd/acp-cli
go build -o ../../../bin/acp-cli .
```

### Running tests
```bash
cd llm-verifier/cmd/acp-cli
go test ./...
```

## Troubleshooting

### Common Issues

**"Provider not found" error**
- Ensure provider name is correct (openai, anthropic, deepseek, google)
- Check that provider is configured in the system
- Verify configuration file is loaded correctly

**"Model failed ACP verification"**
- Check if model actually supports conversational context
- Verify the model responds to JSON-RPC format
- Try increasing timeout for slower models
- Check API key and rate limits

**Connection timeouts**
- Increase timeout with `--timeout` flag
- Check network connectivity to provider
- Verify API endpoints are accessible
- Consider using a proxy if behind firewall

### Debug Mode
Run with `--verbose` flag for detailed output:
```bash
acp-cli verify --model gpt-4 --provider openai --verbose
```

## Integration with CI/CD

### GitHub Actions
```yaml
- name: Test ACP Support
  run: |
    make build-acp
    ./bin/acp-cli batch --models ${{ env.CRITICAL_MODELS }} --output json > acp-results.json
    
- name: Check ACP Results
  run: |
    unsupported=$(jq '[.[] | select(.supported == false)] | length' acp-results.json)
    if [ "$unsupported" -gt 0 ]; then
      echo "❌ Some models failed ACP verification"
      exit 1
    fi
```

### GitLab CI
```yaml
test:acp:
  script:
    - make build-acp
    - ./bin/acp-cli batch --models $CRITICAL_MODELS --output json > acp-results.json
    - |
      if [ "$(jq '[.[] | select(.supported == false)] | length' acp-results.json)" -gt 0 ]; then
        echo "ACP verification failed"
        exit 1
      fi
```

## License

This tool is part of the LLM Verifier project and follows the same license terms.