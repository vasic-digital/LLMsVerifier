# Mandatory Model Code Verification System

## Overview

The Mandatory Model Code Verification System ensures that coding models can actually see and process code with tooling support before being marked as usable. This system sends "Do you see my code?" test requests to each coding model and analyzes their responses to confirm code visibility.

## Features

- **Automated Code Visibility Testing**: Sends test code samples to models and verifies they can see the code
- **Multi-Language Support**: Tests code visibility across Python, JavaScript, Go, Java, and C#
- **Response Analysis**: Analyzes model responses for affirmative confirmation of code visibility
- **Verification Status Tracking**: Maintains verification status for each model in the database
- **Integration with Model Discovery**: Seamlessly integrates with the existing model provider service
- **Comprehensive Reporting**: Generates detailed reports in JSON, CSV, and Markdown formats

## Architecture

### Core Components

1. **CodeVerificationService** (`llm-verifier/verification/code_verification.go`)
   - Handles the core verification logic
   - Sends test requests to models
   - Analyzes responses for code visibility confirmation
   - Calculates confidence scores

2. **CodeVerificationIntegration** (`llm-verifier/verification/code_verification_integration.go`)
   - Integrates verification with the model discovery process
   - Manages verification status in the database
   - Provides APIs for querying verification results

3. **CLI Tool** (`llm-verifier/cmd/code-verification/main.go`)
   - Command-line interface for running verifications
   - Supports filtering by providers and models
   - Generates comprehensive reports

### Verification Process

1. **Model Selection**: Identifies models that support code generation or have coding-related features
2. **Test Request**: Sends "Do you see my code?" prompt with sample code in multiple languages
3. **Response Analysis**: Analyzes responses for affirmative confirmation and code understanding
4. **Status Update**: Updates model metadata with verification status and scores
5. **Report Generation**: Creates detailed verification reports

## Installation

### Prerequisites

- Go 1.21 or higher
- SQLite database
- API keys for model providers (configured in `.env` file)

### Build the Verification Tool

```bash
cd llm-verifier/cmd/code-verification
go build -o code-verification main.go
```

### Configuration

Create a configuration file `code_verification_config.json`:

```json
{
  "provider_filter": [],
  "model_filter": [],
  "max_concurrency": 5,
  "timeout_seconds": 60,
  "output_format": "json"
}
```

## Usage

### Basic Usage

```bash
# Verify all models from all providers
./code-verification

# Verify specific providers
./code-verification -providers openai,anthropic

# Verify specific models
./code-verification -models gpt-4,claude-3.5-sonnet

# Custom output format and directory
./code-verification -output ./results -format markdown
```

### Command Line Options

```
-config string       Path to configuration file
-output string       Output directory for results (default "verification_results")
-providers string    Comma-separated list of providers to verify
-models string       Comma-separated list of models to verify
-concurrency int     Maximum number of concurrent verifications (default 5)
-timeout int         Timeout in seconds for each verification (default 60)
-format string       Output format: json, csv, markdown (default "json")
-db string           Database path (default "../llm-verifier.db")
-help                Show help information
```

## Verification Logic

### Test Code Samples

The system uses representative code samples in multiple languages:

1. **Python**: Fibonacci function with recursion
2. **JavaScript**: QuickSort implementation
3. **Go**: Basic package and main function
4. **Java**: Calculator class with static methods
5. **C#**: Program class with string interpolation

### Response Analysis

The system analyzes model responses for:

- **Affirmative Keywords**: "yes", "i can see", "i see", "visible", "can see"
- **Negative Keywords**: "no", "cannot see", "can't see", "not visible", "do not see"
- **Code References**: Mentions of functions, classes, variables, etc.
- **Language Detection**: Recognition of the programming language

### Scoring Algorithm

Models receive verification scores based on:

- Affirmative response confirmation (50% weight)
- Absence of negative responses (20% weight)
- Code reference detection (10% weight)
- Language understanding level (30% weight)

### Verification Status

Models are marked with one of these statuses:

- **verified**: Successfully confirmed code visibility
- **failed**: Could not confirm code visibility
- **error**: Technical error during verification
- **not_verified**: Has not been verified yet

## Database Schema

### Verification Results Table

The system stores verification results in the `verification_results` table with:

- Model identification and metadata
- Verification status and scores
- Code capability flags
- Response analysis data
- Timestamps and error messages

### Model Metadata Updates

Verified models receive updated metadata:

- `code_visibility_verified`: Boolean indicating successful verification
- `tool_support_verified`: Boolean indicating tooling support
- `verification_score`: Numerical confidence score (0-1)
- `verification_status`: Current verification status
- `last_verified`: Timestamp of last verification

## API Integration

### Provider Service Integration

The system integrates with the existing `ModelProviderService`:

```go
// Create verification service
verificationService := verification.NewCodeVerificationService(httpClient, logger, providerService)

// Create integration
integration := verification.NewCodeVerificationIntegration(verificationService, db, logger, providerService)

// Run verification
results, err := integration.VerifyAllModelsWithCodeSupport(ctx)
```

### Querying Verification Status

```go
// Get verification status for a specific model
status, err := integration.GetVerificationStatus(modelID, providerID)

// Get all verified models
verifiedModels, err := integration.GetAllVerifiedModels()
```

## Output Formats

### JSON Report

```json
{
  "timestamp": "2025-01-01T12:00:00Z",
  "total_models": 150,
  "verified_models": 120,
  "failed_models": 25,
  "error_models": 5,
  "average_score": 8.5,
  "results": [...],
  "summary": {...}
}
```

### CSV Report

```csv
Provider,Model,Status,VerificationScore,CodeVisibility,ToolSupport,VerifiedAt
openai,gpt-4,verified,9.2,true,true,2025-01-01T12:00:00Z
anthropic,claude-3.5-sonnet,verified,8.8,true,true,2025-01-01T12:00:01Z
```

### Markdown Report

```markdown
# Code Verification Report

**Generated:** 2025-01-01T12:00:00Z
**Total Models:** 150
**Verified Models:** 120
**Average Score:** 8.5

## Summary by Provider

| Provider | Total | Verified | Failed | Average Score |
|----------|-------|----------|--------|---------------|
| openai   | 50    | 45       | 5      | 9.2           |
```

## Testing

### Run Tests

```bash
# Run the test script
./test_code_verification.sh

# Test individual components
go test ./llm-verifier/verification/...
```

### Test Coverage

The system includes comprehensive tests for:

- Code verification logic
- Response analysis algorithms
- Database integration
- CLI functionality
- Report generation

## Performance

### Concurrency

- Supports configurable concurrent verifications (default: 5)
- Thread-safe database operations
- Efficient HTTP client pooling

### Caching

- Verification results cached for 24 hours
- Provider model lists cached to reduce API calls
- Database query optimization

## Security

### API Key Management

- API keys stored encrypted in database
- Environment variable support
- Secure HTTP client configuration

### Rate Limiting

- Configurable request timeouts
- Provider-specific rate limit handling
- Automatic retry with exponential backoff

## Monitoring

### Logging

- Structured logging with JSON output
- Configurable log levels
- Request/response logging for debugging

### Metrics

- Verification success/failure rates
- Average response times
- Provider performance metrics

## Troubleshooting

### Common Issues

1. **API Key Errors**: Ensure API keys are properly configured in `.env` file
2. **Timeout Issues**: Increase timeout value for slow providers
3. **Database Errors**: Check database path and permissions
4. **Rate Limiting**: Reduce concurrency for rate-limited providers

### Debug Mode

Enable debug logging:

```bash
export LOG_LEVEL=debug
./code-verification -providers openai
```

## Future Enhancements

### Planned Features

- Support for additional programming languages
- Advanced code understanding tests
- Visual code verification for multimodal models
- Integration with continuous verification pipelines
- Machine learning-based response analysis

### API Extensions

- REST API for verification management
- Webhook notifications for verification completion
- Real-time verification status updates
- Verification scheduling and automation

## Contributing

### Development Setup

1. Clone the repository
2. Install Go 1.21+
3. Set up API keys in `.env` file
4. Run tests: `go test ./...`
5. Build tools: `go build ./...`

### Code Style

- Follow Go best practices
- Use structured logging
- Include comprehensive tests
- Document public APIs

## License

This project is part of the LLM Verifier system and follows the same licensing terms.

## Support

For issues and questions:

1. Check the troubleshooting section
2. Review logs for error details
3. Submit issues to the project repository
4. Contact the development team

---

**Note**: This verification system is critical for ensuring that models can actually process code before being used in production environments. Always run verification before deploying new models or providers.