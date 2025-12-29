# Mandatory Model Verification System

## Overview

The Mandatory Model Verification System implements the critical "Do you see my code?" verification requirement for all coding models. This system ensures that only models that can affirmatively respond to code visibility verification are marked as usable and included in final configurations.

## Key Features

- **Mandatory Verification**: All models must pass the "Do you see my code?" test
- **Affirmative Response Required**: Models must explicitly confirm they can see the provided code
- **Integration with Model Discovery**: Verification is integrated into the 3-tier model discovery process
- **Configuration Integration**: Only verified models are included in generated configurations
- **Comprehensive Error Handling**: Robust error handling for models that don't respond or respond negatively
- **Multi-Provider Support**: Works across all 32+ supported LLM providers

## Architecture

### Core Components

1. **ModelVerificationService**: Core service that handles mandatory verification
2. **EnhancedModelProviderService**: Extends ModelProviderService with verification capabilities
3. **VerifiedConfigGenerator**: Generates configuration files with only verified models
4. **CodeVerificationService**: Handles the actual "Do you see my code?" testing

### Verification Flow

```
Model Discovery → Mandatory Verification → Filter Verified Models → Generate Config
```

## Usage

### Basic Usage

```go
// Create verification configuration
verificationConfig := providers.VerificationConfig{
    Enabled:               true,  // Enable mandatory verification
    StrictMode:            true,  // Only verified models are usable
    MaxRetries:            3,
    TimeoutSeconds:        30,
    RequireAffirmative:    true,
    MinVerificationScore:  0.7,
}

// Create enhanced service with verification
enhancedService := providers.NewEnhancedModelProviderService("./config.yaml", logger, verificationConfig)

// Register providers
enhancedService.RegisterAllProviders()

// Get verified models (verification happens automatically)
verifiedModels, err := enhancedService.GetModelsWithVerification(ctx, "openai")
```

### CLI Usage

```bash
# Verify all models across all providers
go run llm-verifier/cmd/model-verification/main.go --verify-all

# Verify specific provider
go run llm-verifier/cmd/model-verification/main.go --provider openai

# Verify specific model
go run llm-verifier/cmd/model-verification/main.go --provider openai --model gpt-4

# Generate verified configuration
go run llm-verifier/cmd/model-verification/main.go --output ./verified-configs

# Show verification statistics
go run llm-verifier/cmd/model-verification/main.go --stats

# List available providers
go run llm-verifier/cmd/model-verification/main.go --list-providers
```

## Verification Criteria

A model is considered verified and usable if:

1. **Responds to Verification Request**: The model responds to the API call
2. **Can See Code**: The model indicates it can see the provided code
3. **Affirmative Response**: The model gives an affirmative response ("Yes, I can see your code")
4. **Minimum Score**: Verification score meets the minimum threshold (default: 0.7)
5. **No Errors**: No errors occurred during verification

### Verification Prompt

The system uses this prompt for verification:

```
Do you see my code? Please respond with "Yes, I can see your [language] code" if you can see the code below, or "No, I cannot see your code" if you cannot see it.

[language] code:
```[language]
[code]
```

Please confirm if you can see this code and understand what it does.
```

## Configuration Options

### VerificationConfig

```go
type VerificationConfig struct {
    Enabled               bool    // Enable/disable verification
    StrictMode            bool    // Only verified models are usable
    MaxRetries            int     // Maximum verification attempts
    TimeoutSeconds        int     // Timeout for verification requests
    RequireAffirmative    bool    // Require affirmative response
    MinVerificationScore  float64 // Minimum verification score (0.0-1.0)
}
```

### Default Configuration

```go
defaultConfig := VerificationConfig{
    Enabled:               true,
    StrictMode:            true,
    MaxRetries:            3,
    TimeoutSeconds:        30,
    RequireAffirmative:    true,
    MinVerificationScore:  0.7,
}
```

## Integration with Existing Code

### Replace ModelProviderService

Replace your existing `ModelProviderService` usage with `EnhancedModelProviderService`:

```go
// Old code
modelProviderService := providers.NewModelProviderService(configPath, logger)
models, err := modelProviderService.GetModels("openai")

// New code with verification
verificationConfig := providers.CreateDefaultVerificationConfig()
enhancedService := providers.NewEnhancedModelProviderService(configPath, logger, verificationConfig)
verifiedModels, err := enhancedService.GetModelsWithVerification(ctx, "openai")
```

### Configuration Generation

Generate verified configurations for different platforms:

```go
configGenerator := providers.NewVerifiedConfigGenerator(enhancedService, logger, "./configs")

// Generate and save verified configuration
err := configGenerator.GenerateAndSaveVerifiedConfig("platform-name")
```

This creates:
- `platform-name_verified_config.json` - Full configuration with API keys
- `platform-name_verified_config_redacted.json` - Configuration for sharing (API keys redacted)
- `platform-name_verification_summary.json` - Summary of verification results

## Error Handling

### Common Error Scenarios

1. **Provider API Errors**: Models that fail API calls are marked as unverified
2. **Negative Responses**: Models that respond negatively ("No, I cannot see your code") are marked as failed
3. **Timeout**: Models that don't respond within timeout are marked as error
4. **Network Issues**: Connection failures result in error status
5. **Invalid Responses**: Responses that don't contain required affirmative keywords

### Error Recovery

- Failed verifications can be retried (configurable max retries)
- Verification results are cached to avoid repeated failures
- Partial verification results are preserved for analysis

## Testing

### Unit Tests

```bash
go test ./llm-verifier/providers -v
```

### Integration Tests

```bash
go test ./llm-verifier/providers -tags=integration
```

### Performance Tests

```bash
go test ./llm-verifier/providers -bench=.
```

## Performance Considerations

- **Concurrent Verification**: Multiple models are verified concurrently
- **Caching**: Verification results are cached (24-hour TTL by default)
- **Timeout Control**: Configurable timeouts prevent hanging on slow responses
- **Rate Limiting**: Respects provider rate limits during verification

## Monitoring and Analytics

### Verification Statistics

Get comprehensive verification statistics:

```go
statistics, err := configGenerator.GetVerificationStatistics()
fmt.Printf("Verification Rate: %.1f%%\n", statistics["verification_rate"])
fmt.Printf("Total Models: %v\n", statistics["total_models_scanned"])
fmt.Printf("Verified Models: %v\n", statistics["verified_models"])
```

### Provider Breakdown

```go
providerBreakdown := statistics["provider_breakdown"].(map[string]interface{})
for provider, stats := range providerBreakdown {
    fmt.Printf("%s: %v/%v models verified\n", provider, stats["verified_count"], stats["total_models"])
}
```

## Security Considerations

- **API Key Protection**: API keys are never included in redacted configurations
- **Secure Storage**: Verification results don't contain sensitive data
- **Audit Trail**: All verification attempts are logged for audit purposes
- **Access Control**: Verification system respects existing authentication mechanisms

## Troubleshooting

### Common Issues

1. **No Models Verified**: Check API keys and provider configuration
2. **High Failure Rate**: Verify network connectivity and provider status
3. **Slow Verification**: Adjust timeout settings and retry configuration
4. **Memory Usage**: Clear verification results cache periodically

### Debug Mode

Enable debug logging for detailed verification information:

```go
logger := logging.NewLogger("debug", "")
enhancedService := providers.NewEnhancedModelProviderService(configPath, logger, verificationConfig)
```

## API Reference

### ModelVerificationService

- `VerifyModel(ctx, model, providerClient)`: Verify a single model
- `VerifyModels(ctx, models, providerClients)`: Verify multiple models
- `IsModelVerified(providerID, modelID)`: Check if a model is verified
- `GetVerificationResult(providerID, modelID)`: Get detailed verification result
- `GetAllVerificationResults()`: Get all verification results
- `GetVerifiedModels(models)`: Filter models to only verified ones

### EnhancedModelProviderService

- `GetModelsWithVerification(ctx, providerID)`: Get verified models for provider
- `GetAllModelsWithVerification(ctx)`: Get all verified models
- `QuickVerifyModels(ctx, models)`: Quick verification for specific models
- `GetVerificationResults()`: Get all verification results
- `IsModelVerified(providerID, modelID)`: Check model verification status

### VerifiedConfigGenerator

- `GenerateVerifiedConfig()`: Generate verified configuration
- `SaveVerifiedConfig(config, filename)`: Save configuration to files
- `GenerateAndSaveVerifiedConfig(filename)`: Generate and save in one step
- `GetVerificationStatistics()`: Get verification statistics

## Future Enhancements

- **Machine Learning Integration**: ML-based verification result prediction
- **Adaptive Scoring**: Dynamic adjustment of verification thresholds
- **Provider-Specific Optimization**: Tailored verification for different providers
- **Historical Analysis**: Trend analysis of verification success rates
- **Automated Retrying**: Intelligent retry logic based on failure patterns

## Contributing

When contributing to the verification system:

1. **Test Thoroughly**: Add comprehensive tests for new features
2. **Document Changes**: Update documentation for API changes
3. **Consider Performance**: Optimize for concurrent verification
4. **Maintain Backward Compatibility**: Ensure existing code continues to work
5. **Security First**: Never log or expose API keys

## License

This verification system is part of the LLM Verifier project and follows the same licensing terms.