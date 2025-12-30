# LLM Verifier - User Manual

<p align="center">
  <img src="images/Logo.jpeg" alt="LLMsVerifier Logo" width="150" height="150">
</p>

## Table of Contents

1. [Introduction](#introduction)
2. [Installation](#installation)
3. [Quick Start](#quick-start)
4. [Configuration](#configuration)
5. [Usage Guide](#usage-guide)
6. [Client Types](#client-types)
7. [Advanced Features](#advanced-features)
8. [Troubleshooting](#troubleshooting)
9. [Best Practices](#best-practices)
10. [FAQ](#faq)

## Introduction

The LLM Verifier is a comprehensive tool designed to verify, test, and benchmark Large Language Models (LLMs) for their coding capabilities and overall performance. It supports OpenAI-compatible APIs and provides detailed analysis of model capabilities across multiple dimensions.

### Key Features

- **Model Discovery**: Automatically discover all available models from API endpoints
- **Comprehensive Testing**: Test model existence, responsiveness, overload status, and capabilities
- **Feature Detection**: Identify supported features like tool calling, embeddings, code generation, etc.
- **Coding Assessment**: Evaluate coding capabilities across multiple programming languages
- **Performance Scoring**: Calculate detailed scores for code capability, responsiveness, reliability, and features
- **Multi-Client Support**: CLI, TUI, REST API, Web, Desktop, and Mobile clients
- **Database Storage**: Persistent storage with SQLite and SQL Cipher encryption
- **Scheduling**: Automated periodic re-testing with flexible scheduling
- **Event System**: Real-time notifications and event streaming
- **Export Capabilities**: Configuration exports for major AI CLI tools

## Installation

### Prerequisites

- Go 1.21 or later
- SQLite 3.x
- OpenAI API key (or compatible API)

### Binary Installation

```bash
# Download the latest release
wget https://github.com/your-org/llm-verifier/releases/latest/download/llm-verifier-linux-amd64
chmod +x llm-verifier-linux-amd64
sudo mv llm-verifier-linux-amd64 /usr/local/bin/llm-verifier
```

### Source Installation

```bash
# Clone the repository
git clone https://github.com/your-org/llm-verifier.git
cd llm-verifier

# Build the application
go build -o llm-verifier cmd/main.go

# Install globally
sudo cp llm-verifier /usr/local/bin/
```

### Docker Installation

```bash
# Pull the Docker image
docker pull your-org/llm-verifier:latest

# Run with Docker
docker run -v $(pwd)/config:/config -v $(pwd)/reports:/reports \
  -e OPENAI_API_KEY=your-api-key \
  your-org/llm-verifier:latest
```

## Quick Start

### Basic Usage

1. **Create a configuration file** (`config.yaml`):
```yaml
global:
  base_url: "https://api.openai.com/v1"
  api_key: "${OPENAI_API_KEY}"
  max_retries: 3
  request_delay: 1s
  timeout: 30s

concurrency: 5
timeout: 60s
```

2. **Run the verifier**:
```bash
# Using environment variable for API key
export OPENAI_API_KEY=your-api-key
llm-verifier

# Or specify config file explicitly
llm-verifier -c /path/to/config.yaml -o /path/to/output
```

3. **View results**:
```bash
# Check generated reports
ls -la reports/
cat reports/llm_verification_report.md
```

### Advanced Quick Start

For testing specific models:
```yaml
global:
  base_url: "https://api.openai.com/v1"
  api_key: "${OPENAI_API_KEY}"

llms:
  - name: "GPT-4 Turbo"
    endpoint: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4-turbo"
    features:
      tool_calling: true
      embeddings: false

  - name: "GPT-3.5 Turbo"
    endpoint: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-3.5-turbo"
    features:
      tool_calling: true
      embeddings: false

concurrency: 3
timeout: 45s
```

## Configuration

### Global Configuration

```yaml
global:
  base_url: "https://api.openai.com/v1"          # Base API URL
  api_key: "${OPENAI_API_KEY}"                   # API key (use environment variable)
  max_retries: 3                                 # Maximum retry attempts
  request_delay: 1s                              # Delay between requests
  timeout: 30s                                   # Request timeout
  custom_params:                                 # Custom API parameters
    organization: "your-org"
    project: "your-project"
```

### Provider-Specific Configuration

```yaml
llms:
  - name: "OpenAI GPT-4"                         # Display name
    endpoint: "https://api.openai.com/v1"       # API endpoint
    api_key: "${OPENAI_API_KEY}"                # API key
    model: "gpt-4-turbo"                        # Specific model (optional)
    headers:                                     # Custom headers
      Custom-Header: "value"
      Authorization: "Bearer ${OPENAI_API_KEY}"
    features:                                    # Expected features
      tool_calling: true
      embeddings: false
      streaming: true
      multimodal: false
```

### Advanced Configuration Options

```yaml
# Concurrency and performance
concurrency: 5                                   # Concurrent verifications
timeout: 60s                                    # Global timeout

# Database configuration
database:
  path: "llm_verifier.db"                       # Database file path
  encryption_key: "${DB_ENCRYPTION_KEY}"       # Encryption key
  max_connections: 25                           # Maximum connections

# Logging configuration
logging:
  level: "info"                                 # Log level
  format: "json"                                # Log format
  file: "llm_verifier.log"                     # Log file path
  max_size: 100                                 # Max size in MB
  max_backups: 5                                # Max backup files

# Scheduling configuration
scheduling:
  enabled: true                                 # Enable scheduling
  timezone: "UTC"                               # Timezone for schedules
  
# Event system configuration
events:
  enabled: true                                 # Enable event system
  webhooks:                                     # Webhook endpoints
    - url: "https://your-webhook.com/events"
      secret: "${WEBHOOK_SECRET}"
  
# Notification configuration  
notifications:
  slack:
    enabled: true
    webhook_url: "${SLACK_WEBHOOK_URL}"
    channel: "#llm-alerts"
  
  email:
    enabled: true
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    username: "${EMAIL_USERNAME}"
    password: "${EMAIL_PASSWORD}"
    from: "llm-verifier@your-domain.com"
    to: ["admin@your-domain.com"]
```

## Usage Guide

### Command Line Interface (CLI)

#### Basic Commands

```bash
# Run verification with default config
llm-verifier

# Run with custom config file
llm-verifier -c custom-config.yaml

# Run with custom output directory
llm-verifier -o /custom/output/path

# Run with verbose logging
llm-verifier -v

# Run with debug logging
llm-verifier -d

# Run with specific models only
llm-verifier --models gpt-4,gpt-3.5-turbo

# Run with dry-run mode (no actual API calls)
llm-verifier --dry-run
```

#### Advanced Commands

```bash
# Run with database storage
llm-verifier --database llm_verifier.db --encryption-key "${DB_KEY}"

# Run with scheduling
llm-verifier --schedule "0 2 * * *"  # Daily at 2 AM

# Run with event streaming
llm-verifier --events --webhook "https://your-webhook.com"

# Run with notifications
llm-verifier --notify slack,email

# Run with custom concurrency
llm-verifier --concurrency 10

# Run with specific timeout
llm-verifier --timeout 120s
```

### Terminal User Interface (TUI)

#### Launching TUI
```bash
# Start TUI mode
llm-verifier tui

# TUI with specific database
llm-verifier tui --database llm_verifier.db
```

#### TUI Navigation
- **Arrow Keys**: Navigate through menus and lists
- **Enter**: Select item or confirm action
- **Escape**: Go back or cancel
- **Tab**: Switch between panels
- **F1**: Help and documentation
- **F5**: Refresh data
- **Ctrl+Q**: Quit application

#### TUI Features
- **Dashboard**: Overview of all models and their status
- **Model Browser**: Browse and filter models
- **Verification Results**: View detailed verification results
- **Performance Charts**: Visual performance metrics
- **Issue Tracker**: View and manage model issues
- **Configuration Manager**: Manage configurations
- **Real-time Updates**: Live data refresh and event streaming

### REST API Usage

#### Starting the API Server

```bash
# Start API server with default config
llm-verifier server

# Start with custom config
llm-verifier server -c config.yaml

# Start with specific port
llm-verifier server --port 9090

# Start with TLS/HTTPS
llm-verifier server --tls-cert cert.pem --tls-key key.pem
```

#### API Authentication

```bash
# Login to get JWT token
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "password"}'

# Use token in subsequent requests
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  http://localhost:8080/api/v1/models
```

#### Common API Operations

```bash
# List all models
curl -H "Authorization: Bearer TOKEN" \
  http://localhost:8080/api/v1/models

# Get specific model
curl -H "Authorization: Bearer TOKEN" \
  http://localhost:8080/api/v1/models/1

# Trigger verification
curl -X POST -H "Authorization: Bearer TOKEN" \
  http://localhost:8080/api/v1/models/1/verify

# Generate report
curl -X POST -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"report_type": "summary", "format": "json"}' \
  http://localhost:8080/api/v1/reports/generate
```

### Web Client Usage

#### Accessing the Web Interface

1. Start the API server
2. Open browser to `http://localhost:8080`
3. Login with your credentials
4. Navigate through the dashboard

#### Web Client Features

- **Interactive Dashboard**: Real-time model status and metrics
- **Model Comparison**: Side-by-side model comparison
- **Report Generation**: Download reports in multiple formats
- **Configuration Management**: Edit settings through web interface
- **Event Monitoring**: Live event stream and notifications
- **Issue Management**: Track and resolve model issues

### Desktop Application

#### Installation

```bash
# Download desktop app for your platform
# Linux
wget https://github.com/your-org/llm-verifier/releases/download/v1.0.0/llm-verifier-desktop-linux.AppImage
chmod +x llm-verifier-desktop-linux.AppImage

# macOS
wget https://github.com/your-org/llm-verifier/releases/download/v1.0.0/llm-verifier-desktop-macos.dmg

# Windows
wget https://github.com/your-org/llm-verifier/releases/download/v1.0.0/llm-verifier-desktop-windows.exe
```

#### Desktop Features

- **Native Look and Feel**: Platform-specific UI components
- **System Tray Integration**: Background monitoring and notifications
- **Offline Capabilities**: Work with cached data when offline
- **Auto-updates**: Automatic application updates
- **Keyboard Shortcuts**: Efficient navigation and operations

### Mobile Applications

#### iOS/Android Installation

```bash
# iOS: Download from App Store
# Search for "LLM Verifier"

# Android: Download APK or from Play Store
wget https://github.com/your-org/llm-verifier/releases/download/v1.0.0/llm-verifier-mobile.apk
```

#### Mobile Features

- **Touch-Optimized Interface**: Designed for mobile interaction
- **Push Notifications**: Real-time alerts for verification results
- **QR Code Scanning**: Quick configuration import
- **Offline Viewing**: Access cached reports offline
- **Biometric Authentication**: Secure access with fingerprint/face ID
- **Scheduler**: Set up and manage scheduled verifications
- **Export Tool**: Export configurations for CLI tools

### REST API

#### Starting API Server
```bash
# Start REST API server
llm-verifier api --port 8080 --host 0.0.0.0

# With authentication
llm-verifier api --port 8080 --auth-enabled --jwt-secret "${JWT_SECRET}"

# With HTTPS
llm-verifier api --port 8443 --tls-cert cert.pem --tls-key key.pem
```

#### API Endpoints

##### Models
```http
# List all models
GET /api/v1/models

# Get specific model
GET /api/v1/models/{model_id}

# Search models
GET /api/v1/models?search=gpt&provider=openai&min_score=80

# Get model verification results
GET /api/v1/models/{model_id}/results

# Trigger model verification
POST /api/v1/models/{model_id}/verify
```

##### Providers
```http
# List all providers
GET /api/v1/providers

# Get specific provider
GET /api/v1/providers/{provider_id}

# Add new provider
POST /api/v1/providers

# Update provider
PUT /api/v1/providers/{provider_id}

# Delete provider
DELETE /api/v1/providers/{provider_id}
```

##### Verification Results
```http
# List verification results
GET /api/v1/verification-results

# Get specific result
GET /api/v1/verification-results/{result_id}

# Get latest results
GET /api/v1/verification-results/latest

# Search results
GET /api/v1/verification-results?model_id=123&from_date=2024-01-01
```

##### Configuration Exports
```http
# List export configurations
GET /api/v1/config-exports

# Create new export
POST /api/v1/config-exports

# Download export
GET /api/v1/config-exports/{export_id}/download

# Get export for specific tool
GET /api/v1/config-exports/opencode
```

##### Scheduling
```http
# List schedules
GET /api/v1/schedules

# Create new schedule
POST /api/v1/schedules

# Update schedule
PUT /api/v1/schedules/{schedule_id}

# Delete schedule
DELETE /api/v1/schedules/{schedule_id}

# Trigger schedule manually
POST /api/v1/schedules/{schedule_id}/run
```

##### Events and Notifications
```http
# List events
GET /api/v1/events

# Subscribe to events
POST /api/v1/events/subscribe

# Unsubscribe from events
DELETE /api/v1/events/subscribe/{subscription_id}

# Get notification settings
GET /api/v1/notifications/settings

# Update notification settings
PUT /api/v1/notifications/settings
```

#### API Authentication
```http
# Login
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "password"
}

# Use JWT token in subsequent requests
GET /api/v1/models
Authorization: Bearer {jwt_token}
```

## Client Types

### Web Client (Angular)

#### Features
- **Responsive Design**: Works on desktop, tablet, and mobile
- **Real-time Updates**: Live verification status and results
- **Interactive Charts**: Performance metrics visualization
- **Advanced Filtering**: Filter models by various criteria
- **Comparison Tool**: Compare multiple models side-by-side
- **Export Wizard**: Easy configuration export for CLI tools
- **User Management**: Multi-user support with roles
- **Dashboard Customization**: Customizable dashboard widgets

#### Access
```bash
# Access web interface
http://localhost:4200

# With authentication
http://localhost:4200/login
```

### Desktop Applications

#### Platforms Supported
- **Windows**: Native Windows application
- **macOS**: Native macOS application  
- **Linux**: Native Linux application

#### Features
- **System Integration**: Native OS integration
- **Offline Mode**: Work without internet connection
- **System Notifications**: Native desktop notifications
- **Auto-updates**: Automatic application updates
- **System Tray**: Background operation with system tray
- **File Associations**: Open configuration files directly

### Mobile Applications

#### Platforms Supported
- **iOS**: Native iOS application
- **Android**: Native Android application
- **Harmony OS**: Huawei Harmony OS support
- **Aurora OS**: Russian Aurora OS support

#### Features
- **Push Notifications**: Mobile push notifications
- **Offline Access**: Access cached data offline
- **Biometric Authentication**: Fingerprint/Face ID login
- **Mobile-optimized UI**: Touch-friendly interface
- **Background Sync**: Sync data in background
- **Quick Actions**: Quick verification from home screen

## Advanced Features

### Database Integration

#### Setting up Database
```bash
# Initialize database with encryption
llm-verifier db init --encryption-key "${DB_ENCRYPTION_KEY}"

# Migrate database
llm-verifier db migrate --to-version latest

# Backup database
llm-verifier db backup --output backup.sql

# Restore database
llm-verifier db restore --input backup.sql
```

#### Database Queries
```bash
# Query models by criteria
llm-verifier db query models --min-score 80 --supports-tool-use

# Query verification history
llm-verifier db query results --model-id 123 --from-date 2024-01-01

# Query issues
llm-verifier db query issues --severity high --unresolved

# Query events
llm-verifier db query events --type verification_completed --limit 100
```

### Scheduling System

#### Creating Schedules
```bash
# Daily verification at 2 AM
llm-verifier schedule create --name "daily-verification" \
  --cron "0 2 * * *" --target all-models

# Weekly verification on Sundays
llm-verifier schedule create --name "weekly-verification" \
  --cron "0 0 * * 0" --target providers:openai

# Hourly verification for specific models
llm-verifier schedule create --name "hourly-gpt4" \
  --cron "0 * * * *" --target models:gpt-4,gpt-4-turbo

# Custom interval (every 6 hours)
llm-verifier schedule create --name "6h-verification" \
  --interval "6h" --target all-models
```

#### Managing Schedules
```bash
# List all schedules
llm-verifier schedule list

# Enable/disable schedule
llm-verifier schedule enable daily-verification
llm-verifier schedule disable daily-verification

# Update schedule
llm-verifier schedule update daily-verification --cron "0 3 * * *"

# Delete schedule
llm-verifier schedule delete daily-verification

# Run schedule manually
llm-verifier schedule run daily-verification
```

### Event System

#### Event Types
- `verification_started`: Verification process started
- `verification_completed`: Verification process completed
- `score_changed`: Model score changed significantly
- `issue_detected`: New issue detected
- `model_added`: New model added to database
- `model_removed`: Model removed from database
- `provider_offline`: Provider became unavailable
- `provider_online`: Provider became available

#### Event Subscriptions
```bash
# Subscribe to events via WebSocket
llm-verifier events subscribe --websocket ws://localhost:8080/events

# Subscribe to specific event types
llm-verifier events subscribe --types verification_completed,issue_detected

# Subscribe to events for specific models
llm-verifier events subscribe --models gpt-4,gpt-3.5-turbo

# Subscribe to events via webhook
llm-verifier events subscribe --webhook https://your-webhook.com/events
```

### Notification System

#### Slack Integration
```bash
# Configure Slack notifications
llm-verifier config set notifications.slack.webhook_url "${SLACK_WEBHOOK_URL}"
llm-verifier config set notifications.slack.channel "#llm-alerts"
llm-verifier config set notifications.slack.enabled true

# Test Slack notification
llm-verifier notify test slack --message "Test notification from LLM Verifier"
```

#### Email Integration
```bash
# Configure email notifications
llm-verifier config set notifications.email.smtp_host "smtp.gmail.com"
llm-verifier config set notifications.email.smtp_port 587
llm-verifier config set notifications.email.username "${EMAIL_USERNAME}"
llm-verifier config set notifications.email.password "${EMAIL_PASSWORD}"
llm-verifier config set notifications.email.from "llm-verifier@domain.com"
llm-verifier config set notifications.email.to "admin@domain.com,user@domain.com"
llm-verifier config set notifications.email.enabled true

# Test email notification
llm-verifier notify test email --subject "Test Subject" --message "Test message"
```

#### Telegram Integration
```bash
# Configure Telegram notifications
llm-verifier config set notifications.telegram.bot_token "${TELEGRAM_BOT_TOKEN}"
llm-verifier config set notifications.telegram.chat_id "${TELEGRAM_CHAT_ID}"
llm-verifier config set notifications.telegram.enabled true

# Test Telegram notification
llm-verifier notify test telegram --message "Test notification from LLM Verifier"
```

### Configuration Export

#### Export for OpenCode
```bash
# Export configuration for OpenCode
llm-verifier export opencode --output opencode-config.json \
  --min-score 80 --supports-tool-use --supports-code-generation

# Export with custom filters
llm-verifier export opencode --output opencode-config.json \
  --filter "overall_score >= 80 AND supports_code_generation = true"
```

#### Export for Crush
```bash
# Export configuration for Crush
llm-verifier export crush --output crush-config.json \
  --providers openai,anthropic --min-score 75
```

#### Export for Claude Code
```bash
# Export configuration for Claude Code
llm-verifier export claude-code --output claude-config.json \
  --models gpt-4,gpt-4-turbo,gpt-3.5-turbo
```

#### Custom Export
```bash
# Create custom export template
llm-verifier export create-template --name "my-template" \
  --format json --fields "model_id,name,overall_score,supports_tool_use"

# Export using custom template
llm-verifier export custom --template "my-template" --output custom-config.json
```

## Troubleshooting

### Common Issues

#### API Key Issues
```bash
# Test API key validity
llm-verifier test api-key --key "${OPENAI_API_KEY}"

# Check API key permissions
llm-verifier test api-key --key "${OPENAI_API_KEY}" --permissions

# Rotate API key
llm-verifier config update global.api_key "${NEW_API_KEY}"
```

#### Network Issues
```bash
# Test network connectivity
llm-verifier test network --endpoint "https://api.openai.com/v1"

# Check proxy settings
llm-verifier config get proxy
llm-verifier config set proxy.url "http://proxy.company.com:8080"
```

#### Database Issues
```bash
# Check database integrity
llm-verifier db check-integrity

# Repair database
llm-verifier db repair

# Reset database (WARNING: This will delete all data)
llm-verifier db reset --confirm
```

#### Performance Issues
```bash
# Check system performance
llm-verifier system performance

# Optimize database
llm-verifier db optimize

# Clear cache
llm-verifier system clear-cache
```

### Error Messages

#### "Country, region, or territory not supported"
This error occurs when the API is accessed from an unsupported region. Solutions:
- Use a VPN to access from a supported region
- Use a different API provider
- Contact your API provider for region support

#### "Rate limit exceeded"
This error occurs when too many requests are made. Solutions:
- Increase request delay in configuration
- Reduce concurrency setting
- Upgrade to a higher tier API plan
- Implement request queuing

#### "Model not found"
This error occurs when a specified model doesn't exist. Solutions:
- Check model name spelling
- Verify model availability in your API plan
- Use model discovery to find available models
- Update to use current model names

### Debug Mode

#### Enable Debug Logging
```bash
# Run with debug logging
llm-verifier -d

# Debug specific component
llm-verifier --debug-database
llm-verifier --debug-api
llm-verifier --debug-scheduler
```

#### Verbose Output
```bash
# Run with verbose output
llm-verifier -v

# Very verbose output
llm-verifier -vv
```

#### Trace Execution
```bash
# Enable execution tracing
llm-verifier --trace --trace-output trace.log

# Profile performance
llm-verifier --profile --profile-output profile.pprof
```

## Best Practices

### Security

1. **API Key Management**
   - Always use environment variables for API keys
   - Never commit API keys to version control
   - Rotate API keys regularly
   - Use different API keys for different environments

2. **Database Encryption**
   - Always use encryption for production databases
   - Store encryption keys securely
   - Rotate encryption keys periodically
   - Backup encryption keys separately

3. **Network Security**
   - Use HTTPS for all API communications
   - Implement proper firewall rules
   - Use VPN for sensitive environments
   - Enable audit logging

### Performance

1. **Concurrency Settings**
   - Start with conservative concurrency (3-5)
   - Monitor API rate limits
   - Adjust based on API provider limits
   - Consider network bandwidth

2. **Database Optimization**
   - Regular database maintenance
   - Index optimization
   - Query performance monitoring
   - Connection pooling

3. **Caching Strategy**
   - Cache verification results appropriately
   - Implement smart cache invalidation
   - Monitor cache hit rates
   - Balance freshness vs. performance

### Reliability

1. **Error Handling**
   - Implement proper retry mechanisms
   - Use exponential backoff
   - Log all errors appropriately
   - Set up monitoring and alerts

2. **Backup Strategy**
   - Regular database backups
   - Test backup restoration
   - Store backups securely
   - Implement backup verification

3. **Monitoring**
   - Set up comprehensive monitoring
   - Monitor key performance metrics
   - Set up alerting for critical issues
   - Regular health checks

## FAQ

### General Questions

**Q: What LLM providers are supported?**
A: The LLM Verifier supports any OpenAI-compatible API provider, including OpenAI, Azure OpenAI, Anthropic (with compatibility layer), and self-hosted models.

**Q: How often should I run verifications?**
A: For production use, daily verifications are recommended. For development, weekly or as-needed verification is sufficient.

**Q: Can I verify models from multiple providers at once?**
A: Yes, you can configure multiple providers in your configuration file, and the tool will verify models from all configured providers.

**Q: How do I interpret the scores?**
A: Scores range from 0-100 and are weighted across multiple dimensions: Code Capability (40%), Responsiveness (20%), Reliability (20%), Feature Richness (15%), and Value Proposition (5%).

### Technical Questions

**Q: What are the system requirements?**
A: Minimum requirements: 2 CPU cores, 4GB RAM, 1GB disk space. Recommended: 4+ CPU cores, 8GB RAM, 10GB disk space.

**Q: Can I run this in a containerized environment?**
A: Yes, Docker and Kubernetes deployments are fully supported with official container images.

**Q: How do I handle API rate limits?**
A: Configure appropriate concurrency levels, request delays, and retry mechanisms in your configuration file.

**Q: Is the database encrypted?**
A: Yes, the database uses SQL Cipher for encryption at rest. Use the `--encryption-key` parameter to enable encryption.

### Configuration Questions

**Q: Can I use different API keys for different providers?**
A: Yes, each provider can have its own API key configured in the `llms` section of the configuration file.

**Q: How do I configure proxy settings?**
A: Set proxy configuration in the global section: `proxy.url`, `proxy.username`, `proxy.password`.

**Q: Can I exclude certain models from verification?**
A: Yes, use filters in your configuration or the `--exclude-models` command line option.

**Q: How do I set up notifications?**
A: Configure notification settings in the `notifications` section of your configuration file or use the command-line configuration commands.

### Troubleshooting Questions

**Q: Why am I getting "Country not supported" errors?**
A: This is typically due to regional restrictions. Use a VPN or contact your API provider for region support.

**Q: How do I resolve "Rate limit exceeded" errors?**
A: Reduce concurrency, increase request delays, or upgrade your API plan for higher rate limits.

**Q: What should I do if verification fails for all models?**
A: Check your API key validity, network connectivity, and API endpoint configuration. Run with debug logging to identify the specific issue.

**Q: How do I report bugs or request features?**
A: Visit our GitHub repository to report issues or submit feature requests: https://github.com/your-org/llm-verifier

This user manual provides comprehensive guidance for using the LLM Verifier effectively. For additional support, consult the troubleshooting section or contact our support team.