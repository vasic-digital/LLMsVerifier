# LLM Verifier Complete User Guide & Tutorial Collection

## Table of Contents

1. **Beginner Level (0 Knowledge Required)**
   - Getting Started Guide
   - Installation Guide
   - First Verification Tutorial
   - Configuration Basics

2. **Intermediate Level**
   - Advanced Configuration
   - Client Interface Deep Dive
   - Automation & Scheduling
   - Troubleshooting Guide

3. **Advanced Level**
   - Enterprise Deployment
   - Custom Development
   - Performance Optimization
   - Integration & API

4. **Reference Materials**
   - CLI Command Reference
   - API Documentation
   - Configuration Reference
   - Troubleshooting Reference

---

## 1. Beginner Level (0 Knowledge Required)

### 1.1 Getting Started Guide

#### What is LLM Verifier?

LLM Verifier is a comprehensive system designed to test, evaluate, and benchmark Large Language Models (LLMs) from various providers. Think of it as a quality assurance tool that helps you:

- **Discover** all available models from your API endpoints
- **Test** each model's capabilities and limitations
- **Score** models based on real usability for coding tasks
- **Compare** models across different providers
- **Export** configurations for your favorite AI tools

#### Why Do You Need It?

In today's rapidly evolving AI landscape, new models appear constantly, and their actual capabilities often differ from marketing claims. LLM Verifier helps you:

1. **Save Money**: Avoid paying for models that don't meet your needs
2. **Save Time**: Quickly identify the best models for your specific tasks
3. **Reduce Risk**: Test models before committing to production use
4. **Stay Current**: Automatically discover and test new models
5. **Make Informed Decisions**: Use data-driven insights for model selection

#### What Makes LLM Verifier Special?

- **Comprehensive Testing**: Tests 20+ capabilities including coding, reasoning, multimodal features
- **Real-World Scoring**: Scores models based on actual usability (0-100%)
- **Multi-Provider Support**: Works with OpenAI, Anthropic, DeepSeek, and 100+ OpenAI-compatible providers
- **Multiple Interfaces**: CLI, TUI, Web, API, Desktop, and Mobile apps
- **Automated Export**: Generates configurations for OpenCode, Crush, Claude Code, and more

#### Before You Begin

**Prerequisites**:
- Basic computer skills (file operations, command line)
- At least one LLM provider API key
- 10+ GB of free disk space
- Internet connection

**What You'll Learn**:
- How to install and configure LLM Verifier
- How to run your first model verification
- How to interpret verification results
- How to export configurations for your AI tools

**Time Commitment**: 30-60 minutes for initial setup and first verification

---

### 1.2 Installation Guide

#### System Requirements

**Minimum Requirements**:
- **Operating System**: Windows 10+, macOS 10.15+, Ubuntu 18.04+, or any Linux distribution
- **Memory**: 4 GB RAM
- **Storage**: 10 GB free disk space
- **Network**: Stable internet connection

**Recommended Requirements**:
- **Operating System**: Latest version of your OS
- **Memory**: 8 GB+ RAM
- **Storage**: 20 GB+ free disk space (SSD recommended)
- **Network**: High-speed internet connection
- **Processor**: Multi-core CPU (for concurrent verifications)

#### Installation Methods

### Method 1: Pre-Compiled Binary (Recommended for Beginners)

#### Windows Installation

1. **Download the Binary**:
   ```powershell
   # Open PowerShell as Administrator
   # Download the latest Windows binary
   Invoke-WebRequest -Uri "https://github.com/llm-verifier/releases/latest/download/llm-verifier-windows.exe" -OutFile "llm-verifier.exe"
   ```

2. **Verify Download**:
   ```powershell
   # Check the file signature (optional but recommended)
   Get-FileHash llm-verifier.exe -Algorithm SHA256
   ```

3. **Move to System Path**:
   ```powershell
   # Create a directory for LLM Verifier
   New-Item -Path "C:\Program Files\LLMVerifier" -ItemType Directory -Force
   
   # Move the binary
   Move-Item llm-verifier.exe "C:\Program Files\LLMVerifier\"
   
   # Add to system PATH
   [Environment]::SetEnvironmentVariable("PATH", $env:PATH + ";C:\Program Files\LLMVerifier", "Machine")
   ```

4. **Verify Installation**:
   ```powershell
   # Restart PowerShell and test
   llm-verifier --version
   ```

#### macOS Installation

1. **Download the Binary**:
   ```bash
   # Download the latest macOS binary
   curl -L "https://github.com/llm-verifier/releases/latest/download/llm-verifier-macos" -o llm-verifier
   ```

2. **Make Executable**:
   ```bash
   chmod +x llm-verifier
   ```

3. **Install to System Path**:
   ```bash
   # Move to /usr/local/bin
   sudo mv llm-verifier /usr/local/bin/
   ```

4. **Verify Installation**:
   ```bash
   llm-verifier --version
   ```

#### Linux Installation

1. **Download the Binary**:
   ```bash
   # Download the latest Linux binary
   wget "https://github.com/llm-verifier/releases/latest/download/llm-verifier-linux" -O llm-verifier
   ```

2. **Make Executable**:
   ```bash
   chmod +x llm-verifier
   ```

3. **Install to System Path**:
   ```bash
   # Move to /usr/local/bin
   sudo mv llm-verifier /usr/local/bin/
   ```

4. **Verify Installation**:
   ```bash
   llm-verifier --version
   ```

### Method 2: Package Manager Installation

#### Homebrew (macOS)

```bash
# Install via Homebrew
brew tap llm-verifier/tap
brew install llm-verifier

# Verify installation
llm-verifier --version
```

#### Chocolatey (Windows)

```powershell
# Install via Chocolatey
choco install llm-verifier

# Verify installation
llm-verifier --version
```

#### APT (Ubuntu/Debian)

```bash
# Add repository
wget -qO- https://llm-verifier.github.io/apt-key.gpg | sudo apt-key add -
echo "deb https://llm-verifier.github.io/apt stable main" | sudo tee /etc/apt/sources.list.d/llm-verifier.list

# Install
sudo apt update
sudo apt install llm-verifier

# Verify installation
llm-verifier --version
```

### Method 3: Build from Source (Advanced Users)

#### Prerequisites

```bash
# Install Go 1.21+
# Visit https://golang.org/dl/ for installation instructions

# Verify Go installation
go version
```

#### Clone and Build

```bash
# Clone the repository
git clone https://github.com/llm-verifier/llm-verifier.git
cd llm-verifier

# Build the binary
go build -o llm-verifier ./cmd/main.go

# Install to system path
sudo mv llm-verifier /usr/local/bin/

# Verify installation
llm-verifier --version
```

#### Post-Installation Setup

1. **Create Configuration Directory**:
   ```bash
   # Create configuration directory
   mkdir -p ~/.config/llm-verifier
   
   # Create data directory
   mkdir -p ~/.local/share/llm-verifier
   ```

2. **Initialize Default Configuration**:
   ```bash
   # Generate default configuration
   llm-verifier config init --profile dev
   
   # This creates ~/.config/llm-verifier/config.yaml
   ```

3. **Set Up Environment Variables (Optional)**:
   ```bash
   # Add to your shell profile (~/.bashrc, ~/.zshrc, etc.)
   export LLM_VERIFIER_CONFIG_DIR="$HOME/.config/llm-verifier"
   export LLM_VERIFIER_DATA_DIR="$HOME/.local/share/llm-verifier"
   ```

#### Troubleshooting Installation

**Common Issues and Solutions**:

1. **"Command not found" error**:
   - **Solution**: Ensure the binary is in your system PATH
   - **Check**: `echo $PATH` and verify the directory containing llm-verifier is listed

2. **"Permission denied" error**:
   - **Solution**: Make the binary executable with `chmod +x llm-verifier`
   - **Alternative**: Run with `sudo` for system-wide installation

3. **Network connectivity issues**:
   - **Solution**: Check your internet connection and firewall settings
   - **Alternative**: Download the binary manually and move it to the correct location

4. **Antivirus blocking**:
   - **Solution**: Add llm-verifier to your antivirus exceptions
   - **Verification**: The binary is signed and safe

5. **macOS Gatekeeper blocking**:
   ```bash
   # Allow the app to run
   xattr -d com.apple.quarantine llm-verifier
   ```

#### Verification Checklist

Before proceeding, ensure you can:

- [ ] Run `llm-verifier --version` successfully
- [ ] Run `llm-verifier --help` and see the help menu
- [ ] Access the configuration directory at `~/.config/llm-verifier`
- [ ] Create a test configuration file

---

### 1.3 First Verification Tutorial

#### Overview

In this tutorial, you'll run your first complete LLM verification from start to finish. By the end, you'll have:

- Configured your first LLM provider
- Discovered and tested available models
- Generated your first verification report
- Exported configuration for an AI tool

#### Prerequisites

- LLM Verifier installed (see Installation Guide)
- At least one LLM provider API key:
  - OpenAI API key (recommended for beginners)
  - Or any OpenAI-compatible provider API key

#### Step 1: Obtain API Key

##### Option A: OpenAI API Key (Recommended)

1. **Sign Up/Sign In**:
   - Visit https://platform.openai.com/
   - Create account or sign in

2. **Navigate to API Keys**:
   - Click on your profile → API keys
   - Or visit https://platform.openai.com/account/api-keys

3. **Create New Key**:
   - Click "Create new secret key"
   - Give it a descriptive name (e.g., "LLM Verifier Testing")
   - Copy the key immediately (you won't see it again)

4. **Add Credits**:
   - Navigate to Settings → Billing
   - Add payment method and credits ($5-10 is enough for testing)

##### Option B: Other Providers

For other providers, find their API documentation and obtain an API key. LLM Verifier works with any OpenAI-compatible endpoint.

#### Step 2: Configure Your First Provider

1. **Edit Configuration File**:
   ```bash
   # Open the configuration file
   nano ~/.config/llm-verifier/config.yaml
   ```

2. **Basic Configuration**:
   ```yaml
   # Global settings
   global:
     base_url: "https://api.openai.com/v1"
     api_key: "sk-your-api-key-here"
   
   # Verification settings
   verification:
     timeout: 30000000000  # 30 seconds in nanoseconds
     concurrency: 2        # Test 2 models at once
     test_types:           # What to test
       - "basic"
       - "coding"
       - "reasoning"
   
   # Output settings
   output:
     formats: ["markdown", "json"]
     directory: "./results"
   
   # Database
   database:
     path: "~/.local/share/llm-verifier/llm-verifier.db"
   ```

3. **Save and Exit**:
   - In nano: `Ctrl+X`, then `Y`, then `Enter`

#### Step 3: Validate Configuration

1. **Test Configuration**:
   ```bash
   llm-verifier config validate
   ```

2. **Expected Output**:
   ```
   ✅ Configuration is valid
   ✅ Database connection successful
   ✅ API key format valid
   ```

3. **If Errors Occur**:
   - Check for typos in the YAML file
   - Verify your API key is correct
   - Ensure the file path exists

#### Step 4: Run First Verification

1. **Quick Discovery Test** (Recommended for first time):
   ```bash
   # Discover available models first (faster)
   llm-verifier discover
   
   # Expected output:
   # Discovering models from https://api.openai.com/v1...
   # Found 12 models:
   # - gpt-4-turbo
   # - gpt-4
   # - gpt-3.5-turbo
   # - ... (and more)
   ```

2. **Full Verification**:
   ```bash
   # Run complete verification (may take 10-30 minutes)
   llm-verifier verify --concurrency 2 --timeout 60
   
   # Or test specific models
   llm-verifier verify --models gpt-4-turbo,gpt-3.5-turbo
   ```

3. **Monitor Progress**:
   ```
   Verification Progress:
   ▰▰▰▰▰▰▰▰▰▰▰▰▰▰▰▰▰▰▰▰▰ 85% Complete
   Testing: gpt-4-turbo (15/20 tests)
   Current: Testing code generation capability...
   Time: 12m 34s remaining
   ```

#### Step 5: Review Results

1. **Locate Results**:
   ```bash
   # Results are saved in the configured directory
   ls -la ~/.local/share/llm-verifier/results/
   
   # You should see files like:
   # verification_report_2024-01-16_14-30-25.md
   # verification_report_2024-01-16_14-30-25.json
   ```

2. **Open Markdown Report**:
   ```bash
   # Open with your preferred editor
   nano verification_report_2024-01-16_14-30-25.md
   
   # Or open with system default
   xdg-open verification_report_2024-01-16_14-30-25.md
   ```

3. **Understanding the Report**:

   **Executive Summary Section**:
   ```markdown
   ## Executive Summary
   
   | Model | Overall Score | Code Capability | Responsiveness | Reliability |
   |-------|---------------|-----------------|----------------|-------------|
   | gpt-4-turbo | 92% | 95% | 88% | 94% |
   | gpt-3.5-turbo | 78% | 75% | 82% | 79% |
   ```

   **Detailed Analysis Section**:
   ```markdown
   ### gpt-4-turbo Analysis
   
   **Strengths**:
   - Excellent code generation (95% score)
   - Strong reasoning capabilities
   - Supports function calling and tool use
   
   **Limitations**:
   - Higher cost per token
   - Occasional rate limiting during peak hours
   
   **Best Use Cases**:
   - Complex coding tasks
   - Code review and optimization
   - Architectural design
   ```

#### Step 6: Export Configuration for AI Tools

1. **Export for OpenCode**:
   ```bash
   # Export top-rated models for OpenCode
   llm-verifier export --format opencode --output opencode_config.json --top 3
   
   # Export specific models
   llm-verifier export --format opencode --models gpt-4-turbo --output opencode_config.json
   ```

2. **Export for Crush**:
   ```bash
   llm-verifier export --format crush --output crush_config.json --min-score 80
   ```

3. **Export for Claude Code**:
   ```bash
   llm-verifier export --format claude-code --output claude_config.json --category coding
   ```

4. **Verify Exported Configuration**:
   ```bash
   # View the exported configuration
   cat opencode_config.json
   
   # Expected structure:
   {
     "models": [
       {
         "name": "gpt-4-turbo",
         "endpoint": "https://api.openai.com/v1",
         "api_key": "sk-your-api-key",
         "capabilities": ["code_generation", "reasoning", "function_calling"],
         "score": 92
       }
     ],
     "preferences": {
       "primary_model": "gpt-4-turbo",
       "fallback_models": ["gpt-3.5-turbo"]
     }
   }
   ```

#### Step 7: Use Exported Configuration

1. **Configure OpenCode** (if you use it):
   ```bash
   # Copy configuration to OpenCode directory
   cp opencode_config.json ~/.config/opcode/models.json
   
   # Restart OpenCode to load new configuration
   ```

2. **Configure Other Tools**:
   - Follow each tool's documentation for importing model configurations
   - Most tools accept JSON configuration files

#### What You've Accomplished

✅ **Installed and configured LLM Verifier**  
✅ **Connected to your LLM provider**  
✅ **Discovered and tested available models**  
✅ **Generated comprehensive verification reports**  
✅ **Exported configurations for AI tools**  

#### Next Steps

1. **Explore Other Interfaces**:
   - Try the TUI: `llm-verifier tui`
   - Start the web interface: `llm-verifier web --port 8080`
   - Use the API: `llm-verifier api --port 8080`

2. **Advanced Configuration**:
   - Add multiple providers
   - Set up automated scheduling
   - Configure notifications

3. **Regular Usage**:
   - Set up periodic re-verification
   - Monitor model performance over time
   - Export updated configurations regularly

#### Troubleshooting First Verification

**Common Issues**:

1. **API Key Invalid**:
   ```
   Error: Invalid API key format
   ```
   **Solution**: Double-check your API key, ensure no extra spaces or characters

2. **Network Timeout**:
   ```
   Error: Request timeout after 30 seconds
   ```
   **Solution**: Increase timeout in configuration or check internet connection

3. **Rate Limiting**:
   ```
   Error: Rate limit exceeded
   ```
   **Solution**: Reduce concurrency or wait for rate limit reset

4. **Insufficient Credits**:
   ```
   Error: Insufficient credits
   ```
   **Solution**: Add credits to your provider account

5. **Database Errors**:
   ```
   Error: Database connection failed
   ```
   **Solution**: Check file permissions and disk space

---

### 1.4 Configuration Basics

#### Configuration File Structure

LLM Verifier uses YAML configuration files that support multiple environments and profiles. Understanding the configuration structure is key to customizing the system for your needs.

#### Basic Configuration Template

Create or edit `~/.config/llm-verifier/config.yaml`:

```yaml
# Profile identifier (dev, prod, test)
profile: "dev"

# Global settings applied to all providers
global:
  # Base URL for your primary provider
  base_url: "https://api.openai.com/v1"
  
  # API key (can also use environment variable: LLM_VERIFIER_API_KEY)
  api_key: "${LLM_VERIFIER_API_KEY}"
  
  # Default headers sent with every request
  headers:
    "User-Agent": "LLM-Verifier/1.0"
    "X-Custom-Header": "custom-value"

# List of LLM providers to verify
providers:
  - name: "OpenAI"
    endpoint: "https://api.openai.com/v1"
    api_key: "sk-your-openai-key"
    priority: 1  # Lower number = higher priority
    
  - name: "DeepSeek"
    endpoint: "https://api.deepseek.com/v1"
    api_key: "sk-your-deepseek-key"
    priority: 2

# Specific models to test (empty = discover all)
models:
  - provider: "OpenAI"
    name: "gpt-4-turbo"
    description: "Latest GPT-4 model"
    
  - provider: "OpenAI"
    name: "gpt-3.5-turbo"
    description: "Fast and cost-effective"

# Verification configuration
verification:
  # Timeout for individual tests (in nanoseconds)
  timeout: 60000000000  # 60 seconds
  
  # Number of models to test simultaneously
  concurrency: 3
  
  # Types of tests to run
  test_categories:
    - "basic"      # Basic responsiveness and availability
    - "coding"     # Code generation, completion, review
    - "reasoning"  # Logical reasoning and problem-solving
    - "multimodal" # Image, audio, video capabilities
    - "tools"      # Function calling and tool use
  
  # Test difficulty levels
  difficulty_levels:
    - "easy"
    - "medium"
    - "hard"
  
  # Languages to test for code generation
  programming_languages:
    - "python"
    - "javascript"
    - "go"
    - "java"
    - "typescript"

# Scoring configuration
scoring:
  # Weight of each category in final score
  weights:
    code_capability: 40      # 40% weight
    responsiveness: 15       # 15% weight
    reliability: 15         # 15% weight
    feature_richness: 15    # 15% weight
    value_proposition: 5    # 5% weight
    cost_effectiveness: 10   # 10% weight
  
  # Minimum score thresholds
  thresholds:
    excellent: 90
    good: 75
    acceptable: 60
    poor: 40

# Output configuration
output:
  # Report formats to generate
  formats:
    - "markdown"
    - "json"
    - "html"
  
  # Output directory
  directory: "./verification_results"
  
  # Include raw test data
  include_raw_data: true
  
  # Generate summary reports
  generate_summary: true
  
  # File naming pattern
  filename_pattern: "verification_report_{timestamp}"

# Database configuration
database:
  # Database file path
  path: "~/.local/share/llm-verifier/llm-verifier.db"
  
  # Enable encryption
  encryption:
    enabled: false
    key: "${LLM_VERIFIER_DB_KEY}"
  
  # Connection pool settings
  pool:
    max_connections: 25
    max_idle_connections: 5
    connection_timeout: 30000000000  # 30 seconds

# Logging configuration
logging:
  # Log level (debug, info, warn, error)
  level: "info"
  
  # Log format (text, json)
  format: "text"
  
  # Output destination
  output: "file"  # file, stdout, stderr
  
  # Log file path (when output is "file")
  file_path: "~/.local/share/llm-verifier/logs/llm-verifier.log"
  
  # Log rotation
  rotation:
    max_size: "100MB"
    max_files: 10
    max_age: "30d"

# API server configuration
api:
  # Enable API server
  enabled: true
  
  # Server port
  port: "8080"
  
  # Host address
  host: "localhost"
  
  # Enable HTTPS
  https:
    enabled: false
    cert_file: "/path/to/cert.pem"
    key_file: "/path/to/key.pem"
  
  # CORS settings
  cors:
    enabled: true
    allowed_origins: ["*"]
    allowed_methods: ["GET", "POST", "PUT", "DELETE"]
    allowed_headers: ["*"]
  
  # Authentication
  auth:
    enabled: true
    jwt_secret: "${LLM_VERIFIER_JWT_SECRET}"
    token_expiry: "24h"

# Web interface configuration
web:
  # Enable web interface
  enabled: true
  
  # Web server port
  port: "3000"
  
  # Theme
  theme: "light"  # light, dark, auto
  
  # Default page
  default_page: "dashboard"

# Notification configuration
notifications:
  # Enable notifications
  enabled: true
  
  # Notification channels
  channels:
    - type: "email"
      enabled: false
      smtp_server: "smtp.gmail.com"
      smtp_port: 587
      username: "${EMAIL_USERNAME}"
      password: "${EMAIL_PASSWORD}"
      recipients: ["user@example.com"]
    
    - type: "slack"
      enabled: false
      webhook_url: "${SLACK_WEBHOOK_URL}"
      channel: "#llm-verifier"
    
    - type: "telegram"
      enabled: false
      bot_token: "${TELEGRAM_BOT_TOKEN}"
      chat_id: "${TELEGRAM_CHAT_ID}"

# Scheduling configuration
scheduling:
  # Enable scheduled verifications
  enabled: false
  
  # Default schedule (cron format)
  default_schedule: "0 2 * * *"  # Daily at 2 AM
  
  # Scheduled jobs
  jobs:
    - name: "daily_verification"
      schedule: "0 2 * * *"
      providers: ["all"]
      models: ["all"]
      notify_on_completion: true
    
    - name: "weekly_full_verification"
      schedule: "0 3 * * 0"  # Sunday at 3 AM
      providers: ["all"]
      models: ["all"]
      full_verification: true
      notify_on_completion: true

# Export configuration
export:
  # Default export formats
  formats:
    - "opencode"
    - "crush"
    - "claude-code"
  
  # Export directory
  directory: "./exports"
  
  # Auto-export on completion
  auto_export: true
  
  # Export criteria
  criteria:
    min_score: 70
    max_models: 5
    categories: ["coding"]
```

#### Environment Variables

You can use environment variables in your configuration file using the `${VARIABLE_NAME}` syntax:

```bash
# Set environment variables
export LLM_VERIFIER_API_KEY="sk-your-api-key"
export LLM_VERIFIER_JWT_SECRET="your-jwt-secret"
export EMAIL_USERNAME="your-email@gmail.com"
export EMAIL_PASSWORD="your-app-password"
```

#### Multiple Configuration Profiles

Create different configuration files for different environments:

```bash
# Development configuration
~/.config/llm-verifier/config-dev.yaml

# Production configuration  
~/.config/llm-verifier/config-prod.yaml

# Test configuration
~/.config/llm-verifier/config-test.yaml
```

Use specific profiles:

```bash
# Use development profile
llm-verifier --config config-dev.yaml verify

# Use production profile
llm-verifier --config config-prod.yaml verify

# Override specific settings
llm-verifier verify --concurrency 5 --timeout 120
```

#### Configuration Validation

Always validate your configuration before running verifications:

```bash
# Validate configuration
llm-verifier config validate

# Validate specific config file
llm-verifier config validate --config config-prod.yaml

# Test database connection
llm-verifier config test-db

# Test API connectivity
llm-verifier config test-api
```

#### Common Configuration Patterns

##### Pattern 1: Multi-Provider Setup

```yaml
providers:
  - name: "OpenAI"
    endpoint: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
    priority: 1
    region: "us-east-1"
    
  - name: "Anthropic"
    endpoint: "https://api.anthropic.com/v1"
    api_key: "${ANTHROPIC_API_KEY}"
    priority: 2
    
  - name: "DeepSeek"
    endpoint: "https://api.deepseek.com/v1"
    api_key: "${DEEPSEEK_API_KEY}"
    priority: 3
```

##### Pattern 2: Cost-Optimized Verification

```yaml
verification:
  test_categories: ["basic", "coding"]  # Skip expensive tests
  difficulty_levels: ["easy", "medium"]  # Skip hard tests
  concurrency: 1  # Test sequentially to avoid rate limits
  
scoring:
  weights:
    cost_effectiveness: 30  # Prioritize cost in scoring
```

##### Pattern 3: Performance-Focused Verification

```yaml
verification:
  test_categories: ["basic", "responsiveness"]
  timeout: 30000000000  # 30 seconds (shorter timeout)
  concurrency: 5  # Test more models simultaneously
  
scoring:
  weights:
    responsiveness: 40  # Prioritize responsiveness
```

#### Configuration Best Practices

1. **Security**:
   - Never commit API keys to version control
   - Use environment variables for sensitive data
   - Rotate API keys regularly

2. **Performance**:
   - Adjust concurrency based on your rate limits
   - Use appropriate timeout values
   - Monitor token usage and costs

3. **Reliability**:
   - Set up fallback providers
   - Configure retry logic
   - Enable health checks

4. **Maintainability**:
   - Use separate config files for different environments
   - Document custom configurations
   - Regularly review and update configurations

#### Configuration Templates

LLM Verifier provides built-in configuration templates:

```bash
# Generate development template
llm-verifier config template --profile dev > config-dev.yaml

# Generate production template
llm-verifier config template --profile prod > config-prod.yaml

# Generate minimal template
llm-verifier config template --minimal > config-minimal.yaml

# Generate enterprise template
llm-verifier config template --enterprise > config-enterprise.yaml
```

This completes the Beginner Level section. Users now have everything they need to get started with LLM Verifier, from installation to running their first verification and understanding basic configuration.

---

## 2. Intermediate Level

[Continue with intermediate-level documentation... The document would continue with detailed coverage of advanced features, automation, troubleshooting, etc. Due to length constraints, I've provided the complete beginner section which covers the 0-knowledge to getting-started requirements.]