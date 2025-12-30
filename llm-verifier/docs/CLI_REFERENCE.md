# LLM Verifier CLI Reference

<p align="center">
  <img src="images/Logo.jpeg" alt="LLMsVerifier Logo" width="150" height="150">
</p>

<p align="center">
  <strong>Verify. Monitor. Optimize.</strong>
</p>

---

## Overview

The LLM Verifier CLI provides comprehensive command-line interface for managing, verifying, and benchmarking Large Language Models (LLMs). It supports all core functionality including model management, verification workflows, reporting, and system administration.

## Global Flags

These flags apply to all commands:

- `-c, --config string` - Configuration file path (default: `config.yaml`)
- `-s, --server string` - API server URL (default: `http://localhost:8080`)
- `-u, --username string` - Username for authentication
- `-p, --password string` - Password for authentication
- `-o, --output string` - Output directory for reports (default: `./reports`)
- `-h, --help` - Show help for command

## Commands

### Core Verification

**`llm-verifier`** - Run verification with default configuration
```bash
llm-verifier -c config.yaml -o ./reports
```

### Server Management

**`llm-verifier server`** - Start the REST API server
```bash
llm-verifier server --port 8080
```

Flags:
- `--port string` - Port to listen on (default: `8080`)
- `--config string` - Configuration file path

### Model Management

**`llm-verifier models list`** - List all models
```bash
llm-verifier models list --format table
llm-verifier models list --format json --filter "provider=openai"
```

Flags:
- `--filter string` - Filter models by criteria
- `--format string` - Output format: `table`, `json`, `yaml` (default: `table`)
- `--limit int` - Maximum number of results

**`llm-verifier models get <id>`** - Get model details
```bash
llm-verifier models get gpt-4-turbo
```

**`llm-verifier models create`** - Create a new model
```bash
llm-verifier models create --name "GPT-4" --provider openai --model-id gpt-4
```

**`llm-verifier models update <id>`** - Update model
**`llm-verifier models delete <id>`** - Delete model

### Provider Management

**`llm-verifier providers list`** - List all providers
```bash
llm-verifier providers list --format json
```

**`llm-verifier providers get <id>`** - Get provider details
**`llm-verifier providers create`** - Create new provider
**`llm-verifier providers update <id>`** - Update provider
**`llm-verifier providers delete <id>`** - Delete provider

### Verification Results

**`llm-verifier results list`** - List verification results
```bash
llm-verifier results list --filter "model=gpt-4" --limit 10
```

**`llm-verifier results get <id>`** - Get result details
**`llm-verifier results export`** - Export results to file

### Configuration Export

**`llm-verifier ai-config export`** - Export AI CLI agent configurations
```bash
llm-verifier ai-config export --format opencode --output ./exports
```

Flags:
- `--format string` - Export format: `opencode`, `crush`, `claude-code`, `json`, `yaml`
- `--output string` - Output directory
- `--include-api-key` - Include API key placeholder

**`llm-verifier ai-config validate`** - Validate exported configuration

### Scheduling

**`llm-verifier schedules list`** - List scheduled jobs
**`llm-verifier schedules create`** - Create new schedule
**`llm-verifier schedules delete <id>`** - Delete schedule

### Event Management

**`llm-verifier events list`** - List system events
**`llm-verifier events subscribe`** - Subscribe to real-time events

### Issue Tracking

**`llm-verifier issues list`** - List detected issues
**`llm-verifier issues resolve <id>`** - Mark issue as resolved

### Rate Limits

**`llm-verifier limits list`** - List rate limits
**`llm-verifier limits update`** - Update limit configuration

### Pricing Management

**`llm-verifier pricing list`** - List pricing plans
**`llm-verifier pricing update`** - Update pricing information

### Batch Operations

**`llm-verifier batch verify`** - Batch verification of multiple models
```bash
llm-verifier batch verify --models gpt-4,claude-3 --parallel 3
```

**`llm-verifier batch export`** - Batch export configurations

### Terminal User Interface (TUI)

**`llm-verifier tui`** - Start the Terminal User Interface
```bash
llm-verifier tui --server http://localhost:8080
```

### Configuration Management

**`llm-verifier config show`** - Show current configuration
**`llm-verifier config export`** - Export configuration to file
**`llm-verifier config validate`** - Validate configuration file

### Log Management

**`llm-verifier logs list`** - List system logs
**`llm-verifier logs tail`** - Tail live logs

## TUI Navigation

The Terminal User Interface provides an interactive interface with keyboard navigation:

### Screen Navigation
- `1` - Dashboard (main screen with statistics)
- `2` - Models (browse and manage models)
- `3` - Providers (manage LLM providers)
- `4` - Verification (run and monitor verifications)

### Global Shortcuts
- `q` or `Ctrl+C` - Quit application
- `r` - Refresh current screen data
- `h` - Show help overlay

### Dashboard Features
- Real-time statistics updated every 30 seconds
- Verification progress visualization
- Quick action buttons with keyboard shortcuts
- Color-coded score indicators

### Model Browser
- Filter models by provider, capability, score
- Sort by different columns
- View detailed model information
- Trigger verification directly from TUI

### Verification Screen
- Select models for verification
- Configure verification parameters
- Monitor real-time progress
- View results immediately upon completion

## Examples

### Basic Verification Workflow
```bash
# Start API server
llm-verifier server --port 8080 &

# List available models
llm-verifier models list

# Run verification
llm-verifier --config production.yaml --output ./results

# Export configuration for AI tools
llm-verifier ai-config export --format crush --output ./ai-configs
```

### Automated Testing Pipeline
```bash
# Batch verify all models
llm-verifier batch verify --all --parallel 5

# Export results in JSON format
llm-verifier results list --format json > results.json

# Generate markdown report
llm-verifier --output ./reports
```

### Integration with CI/CD
```bash
# Validate configuration before deployment
llm-verifier config validate --config ci-config.yaml

# Run verification with specific models
llm-verifier --config ci-config.yaml --output ./artifacts

# Export configuration for deployment
llm-verifier ai-config export --format opencode --output ./artifacts
```

## Troubleshooting

### Common Issues

**Connection refused errors**: Ensure the API server is running: `llm-verifier server --port 8080`

**Authentication errors**: Provide valid credentials: `llm-verifier --username admin --password secret`

**Configuration errors**: Validate configuration file: `llm-verifier config validate --config your-config.yaml`

**Export format not supported**: Check available formats with `llm-verifier ai-config export --help`

### Getting Help

Use the `--help` flag with any command for detailed usage information:
```bash
llm-verifier --help
llm-verifier models --help
llm-verifier ai-config export --help
```

## See Also

- [User Manual](USER_MANUAL.md) - Comprehensive user guide
- [API Documentation](API_DOCUMENTATION.md) - REST API reference
- [Deployment Guide](DEPLOYMENT_GUIDE.md) - Production deployment instructions
- [Environment Variables](ENVIRONMENT_VARIABLES.md) - Configuration via environment variables