# LLM Verifier

LLM Verifier is a comprehensive tool to verify, test, and benchmark LLMs based on their coding capabilities and other features. It supports OpenAI-compatible APIs and provides detailed analysis of model capabilities.

## Features

- **Model Discovery**: Automatically discover all available models from API endpoints
- **Comprehensive Testing**: Test model existence, responsiveness, overload status, and capabilities
- **Feature Detection**: Identify supported features like tool calling, embeddings, code generation, etc.
- **Coding Assessment**: Evaluate coding capabilities across multiple programming languages
- **Performance Scoring**: Calculate detailed scores for code capability, responsiveness, reliability, and feature richness
- **Reporting**: Generate both human-readable markdown reports and machine-readable JSON reports
- **Rankings**: Sort models by various criteria (strength, speed, reliability, etc.)

## Installation

```bash
# Clone the repository
git clone <repository-url>
cd llm-verifier

# Build the application
go build -o llm-verifier cmd/main.go
```

## Usage

### Basic Usage

```bash
./llm-verifier
```

This will use the default configuration file `config.yaml` and output reports to the `./reports` directory.

### With Custom Configuration

```bash
./llm-verifier -c /path/to/config.yaml -o /path/to/output
```

## Configuration

The tool uses a YAML configuration file. See `config.yaml` for a sample configuration:

```yaml
global:
  base_url: "https://api.openai.com/v1"
  api_key: "${OPENAI_API_KEY}" # Use environment variable
  max_retries: 3
  request_delay: 1s
  timeout: 30s

# If no LLMs are specified, the tool will automatically discover all available models
# llms: # Uncomment this section to specify specific LLMs to test
#   - name: "OpenAI GPT-4"
#     endpoint: "https://api.openai.com/v1"
#     api_key: "${OPENAI_API_KEY}"
#     model: "gpt-4-turbo"
#     headers:
#       Custom-Header: "value"
#     features:
#       tool_calling: true
#       embeddings: false
#
#   - name: "OpenAI GPT-3.5"
#     endpoint: "https://api.openai.com/v1"
#     api_key: "${OPENAI_API_KEY}"
#     model: "gpt-3.5-turbo"
#     features:
#       tool_calling: true
#       embeddings: false

concurrency: 5
timeout: 60s
```

## Output

The tool generates two types of reports:

1. **Markdown Report** (`llm_verification_report.md`): Human-readable report with detailed analysis
2. **JSON Report** (`llm_verification_report.json`): Machine-readable report for programmatic use

## Test Suite

The project includes comprehensive tests:

- Unit tests: Test individual functions and scoring algorithms
- Integration tests: Test component interactions
- End-to-end tests: Test complete workflows
- Performance tests: Benchmark critical functions
- Security tests: Verify secure handling of sensitive data
- Automation tests: Test automated workflows

Run all tests with:

```bash
go test ./tests/... -v
```

## Architecture

The tool is organized into several packages:

- `cmd/`: Main application entry point
- `llmverifier/`: Core verification logic
- `config/`: Configuration structures and loading
- `reports/`: Report generation
- `tests/`: Comprehensive test suite

## Capabilities Tested

The tool evaluates models across several dimensions:

- **Code Generation**: Ability to write code in multiple languages
- **Code Completion**: Ability to complete partial code
- **Code Debugging**: Ability to identify and fix code issues
- **Code Review**: Ability to review and improve code
- **Code Explanation**: Ability to explain code functionality
- **Test Generation**: Ability to create unit tests
- **Documentation**: Ability to create documentation
- **Refactoring**: Ability to improve code structure
- **Architecture Understanding**: Ability to design system architectures
- **Security Assessment**: Ability to identify security issues
- **Pattern Recognition**: Ability to implement design patterns
- **Language-Specific Tests**: Performance across multiple programming languages

## Scoring System

Models are scored across multiple dimensions:

- **Code Capability (40%)**: How well the model handles coding tasks
- **Responsiveness (20%)**: Response time and throughput
- **Reliability (20%)**: Availability and consistency
- **Feature Richness (15%)**: Supported features and capabilities
- **Value Proposition (5%)**: Overall value for coding tasks

## License

MIT