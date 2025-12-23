# Challenge Runner Guide

**Complete Guide to Using the Generic Challenge Runner**

---

## 📖 Table of Contents

1. [What is Challenge Runner?](#what-is-the-challenge-runner)
2. [Basic Usage](#basic-usage)
3. [Advanced Usage](#advanced-usage)
4. [Command Reference](#command-reference)
5. [Platform Options](#platform-options)
6. [Examples](#examples)
7. [Best Practices](#best-practices)

---

## What is the Challenge Runner?

The **Challenge Runner** (`run_challenges.sh`) is a generic script that can:

1. **List all available challenges** - Browse the Challenge Bank
2. **Run specific challenges** - Execute one or more challenges by name
3. **Run all challenges** - Execute every challenge in the bank
4. **Support multiple platforms** - Test on CLI, REST API, TUI, Desktop, Mobile, Web
5. **Create proper structure** - Generate timestamped result directories
6. **Log everything** - Capture all commands and activities

### How It Works

```
User Input
     ↓
Challenge Bank (JSON)
     ↓
Challenge Runner Script
     ↓
Challenge Script (Specific)
     ↓
Platform Binary (llm-verifier, llm-verifier-api, etc.)
     ↓
Challenge Results (Timestamped Directory)
     ↓
Git Versioning (Results only)
```

---

## Basic Usage

### List All Challenges

View all available challenges with details:

```bash
cd challenges
./run_challenges.sh --list
```

**Output:**
```
provider_models_discovery
  Name:         Provider Models Discovery
  Description:  Discover all available providers, their models, and verify features
  Platforms:    cli, rest-api, tui, desktop
  Duration:     5-10 minutes
  Outputs:      providers_opencode.json, providers_crush.json

model_verification
  Name:         Model Verification
  Description:  Verify each model's actual capabilities
  Platforms:    cli, rest-api
  Duration:     15-30 minutes
  Outputs:      models_verification_opencode.json

[... more challenges]
```

### Run Single Challenge

Execute a specific challenge using default platform (CLI):

```bash
cd challenges
./run_challenges.sh provider_models_discovery
```

**What Happens:**
1. Challenge runner reads Challenge Bank JSON
2. Finds challenge by ID `provider_models_discovery`
3. Checks binary availability (`llm-verifier`)
4. Creates timestamped directory structure
5. Executes challenge script
6. Challenge script runs binary with config
7. Logs all commands and activities
8. Saves results to `results/` subdirectory
9. Completes with summary

**Output Directory:**
```
challenges/provider_models_discovery/2025/12/23/1766500000/
├── config.yaml
├── logs/
│   ├── challenge.log
│   └── commands.log
└── results/
    ├── providers_opencode.json
    └── providers_crush.json
```

---

## Advanced Usage

### Run Multiple Challenges

Execute multiple specific challenges:

```bash
cd challenges
./run_challenges.sh provider_models_discovery model_verification
./run_challenges.sh provider_models_discovery model_verification feature_integration
```

**Execution Order:**
1. `provider_models_discovery` - First
2. `model_verification` - Second
3. `feature_integration` - Third (if specified)

### Run All Challenges

Execute every challenge in the Challenge Bank:

```bash
cd challenges
./run_challenges.sh --all
```

**Benefits:**
- Complete system validation
- All features tested
- All platforms validated
- Comprehensive test coverage

**Note:** May take 1-2 hours depending on challenge count.

### Specify Platform

Run challenge on specific platform:

```bash
cd challenges

# CLI (default)
./run_challenges.sh provider_models_discovery --platform cli

# REST API
./run_challenges.sh provider_models_discovery --platform rest-api

# TUI
./run_challenges.sh provider_models_discovery --platform tui

# Desktop
./run_challenges.sh provider_models_discovery --platform desktop

# Mobile
./run_challenges.sh provider_models_discovery --platform mobile

# Web
./run_challenges.sh provider_models_discovery --platform web
```

**Platform-Specific Behavior:**
- Different binary used per platform
- Different configuration templates
- Platform-specific test scenarios
- Platform-specific result formats

### Combine Options

Use multiple options together:

```bash
# Run specific challenge on REST API
./run_challenges.sh provider_models_discovery --platform rest-api

# Run all challenges on TUI
./run_challenges.sh --all --platform tui

# List platforms
./run_challenges.sh --platforms
```

---

## Command Reference

### Global Options

| Option | Description | Example |
|---------|-------------|---------|
| `--list` | List all available challenges | `./run_challenges.sh --list` |
| `--all` | Run all challenges | `./run_challenges.sh --all` |
| `--platform PLATFORM` | Use specific platform | `./run_challenges.sh provider_models_discovery --platform cli` |
| `--platforms` | List available platforms | `./run_challenges.sh --platforms` |
| `--help`, `-h` | Show help message | `./run_challenges.sh --help` |

### Challenge IDs

| ID | Name | Description |
|-----|-------|-------------|
| `provider_models_discovery` | Provider Models Discovery | Discover providers and models |
| `model_verification` | Model Verification | Verify model capabilities |
| `feature_integration` | Feature Integration | Test multi-provider features |
| `performance_benchmark` | Performance Benchmark | Benchmark system performance |
| `rest_api_testing` | REST API Testing | Test REST API endpoints |
| `tui_functionality` | TUI Functionality | Test TUI features |
| `desktop_app_testing` | Desktop App Testing | Test desktop application |
| `mobile_app_testing` | Mobile App Testing | Test mobile application |
| `config_export_import` | Config Export/Import | Test config management |

---

## Platform Options

### CLI (Command Line Interface)

**Binary:** `llm-verifier`

**Description:** Command-line interface for all LLM Verifier functionality

**Supported Challenges:** All

**Advantages:**
- Fast execution
- Scriptable
- No UI dependencies
- Suitable for automation

**Example:**
```bash
./run_challenges.sh provider_models_discovery --platform cli
```

### REST API

**Binary:** `llm-verifier-api`

**Description:** REST API server for programmatic access

**Supported Challenges:** All

**Advantages:**
- Remote access
- Multi-client support
- API testing
- Integration ready

**Example:**
```bash
# Start API server (if not running)
../llm-verifier-api --port 8080 &

# Run challenge
./run_challenges.sh provider_models_discovery --platform rest-api
```

### TUI (Terminal User Interface)

**Binary:** `llm-verifier-tui`

**Description:** Terminal-based user interface

**Supported Challenges:** Most (CLI-based challenges)

**Advantages:**
- Interactive
- Visual feedback
- Keyboard navigation
- Terminal-friendly

**Example:**
```bash
./run_challenges.sh tui_functionality --platform tui
```

### Desktop

**Binary:** `llm-verifier-desktop`

**Description:** Desktop application (Electron-based)

**Supported Challenges:** UI-based challenges

**Advantages:**
- Graphical interface
- Desktop integration
- System tray
- Drag-and-drop

**Example:**
```bash
./run_challenges.sh desktop_app_testing --platform desktop
```

### Mobile

**Binary:** `llm-verifier-mobile`

**Description:** Mobile application (Flutter-based)

**Supported Challenges:** UI-based challenges

**Advantages:**
- Touch interface
- Mobile-optimized
- Notifications
- Offline capable

**Example:**
```bash
./run_challenges.sh mobile_app_testing --platform mobile
```

### Web

**Binary:** `llm-verifier-web`

**Description:** Web application (Next.js-based)

**Supported Challenges:** UI-based challenges

**Advantages:**
- Browser-based
- No installation
- Cross-platform
- Shareable links

**Example:**
```bash
# Start web server (if not running)
../llm-verifier-web --port 3000 &

# Run challenge
./run_challenges.sh provider_models_discovery --platform web
```

---

## Examples

### Example 1: Quick Test (CLI)

Run a quick test using CLI:

```bash
cd challenges
./run_challenges.sh provider_models_discovery
```

**Timeline:**
- 0:00 - Start challenge
- 0:01 - Create config
- 0:02 - Run binary
- 0:05 - Complete challenge
- 0:05 - View results

### Example 2: Full Validation (CLI)

Run multiple dependent challenges:

```bash
cd challenges
./run_challenges.sh provider_models_discovery model_verification
```

**Timeline:**
- 0:00 - Start provider discovery
- 0:05 - Complete provider discovery
- 0:05 - Start model verification
- 0:20 - Complete model verification
- 0:20 - View all results

### Example 3: All Platforms Test

Run same challenge on all platforms:

```bash
cd challenges

# CLI
./run_challenges.sh provider_models_discovery --platform cli

# REST API
./run_challenges.sh provider_models_discovery --platform rest-api

# TUI
./run_challenges.sh provider_models_discovery --platform tui

# Desktop
./run_challenges.sh provider_models_discovery --platform desktop
```

**Purpose:** Verify challenge works on all platforms

### Example 4: Nightly Regression Test

Run all challenges every night:

```bash
#!/bin/bash
# nightly_test.sh
cd /path/to/llm-verifier/challenges
./run_challenges.sh --all

# Configure cron (run every night at 2 AM)
# crontab -e
# 0 2 * * * /path/to/nightly_test.sh > /var/log/llm-verifier-nightly.log 2>&1
```

### Example 5: CI/CD Integration

Integrate challenge runner in CI/CD:

```yaml
# .github/workflows/challenge_tests.yml
name: Challenge Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Build binaries
        run: |
          go build -o llm-verifier cmd/main.go
      - name: Run challenges
        run: |
          cd challenges
          ./run_challenges.sh --all
      - name: Upload results
        uses: actions/upload-artifact@v2
        with:
          name: challenge-results
          path: challenges/*/20**/*/
```

---

## Best Practices

### 1. Always Run on Multiple Platforms

**Why:** Challenge might work on CLI but fail on Desktop.

**Practice:**
```bash
# Test on CLI first
./run_challenges.sh my_challenge --platform cli

# Then test on other platforms
./run_challenges.sh my_challenge --platform rest-api
./run_challenges.sh my_challenge --platform tui
```

### 2. Review Logs After Execution

**Why:** Logs contain valuable debug information.

**Practice:**
```bash
cd challenges/<challenge_name>/YYYY/MM/DD/timestamp/logs

# Check for errors
grep -i "error\|failed\|exception" challenge.log

# Review commands executed
cat commands.log
```

### 3. Validate JSON Results

**Why:** Malformed JSON indicates issues.

**Practice:**
```bash
cd challenges/<challenge_name>/YYYY/MM/DD/timestamp/results

# Validate JSON
jq . providers_opencode.json
jq . providers_crush.json
```

### 4. Keep Challenge Results Versioned

**Why:** Track changes over time, enable rollback.

**Practice:**
```bash
cd challenges
git add <challenge_name>/20**/*/
git commit -m "Challenge results: <challenge_name> on YYYY-MM-DD"
git push
```

**Note:** API keys are loaded from environment variables (ApiKey_*), never store them in files or commit them!

### 5. Clean Up Old Results

**Why:** Save disk space, keep relevant results.

**Practice:**
```bash
# Keep only last 30 days
find challenges -type d -mtime +30 -exec rm -rf {} \;

# Or archive old results
tar -czf challenges_archive_YYYY-MM-DD.tar.gz challenges/*/2025/11/*
```

### 6. Use Meaningful Timestamps

**Why:** Easy to identify test runs.

**Practice:**
```bash
# Good: Clear timestamp
TIMESTAMP=$(date +%s)
# Result: challenges/provider_discovery/2025/12/23/1766505525/

# Avoid: Multiple runs in same minute
# Result: challenges/provider_discovery/2025/12/23/1766505525/
#         challenges/provider_discovery/2025/12/23/1766505526/
```

### 7. Run Challenges in Dependency Order

**Why:** Ensure prerequisites are met.

**Practice:**
```bash
# Wrong order (will fail)
./run_challenges.sh model_verification provider_models_discovery

# Correct order (will succeed)
./run_challenges.sh provider_models_discovery model_verification
```

---

## Summary

The Challenge Runner provides:

✅ **Generic execution** - Run any challenge from bank
✅ **Multi-platform support** - Test on all application derivatives
✅ **Automatic directory creation** - Versioned results structure
✅ **Complete logging** - All commands and activities captured
✅ **Dependency checking** - Validates challenge order
✅ **Flexible configuration** - Platform-specific, data-driven

**By mastering the Challenge Runner, you can:**
- Quickly test any feature
- Validate changes across all platforms
- Automate testing workflows
- Maintain comprehensive test coverage

---

**Next:** [Creating Challenges](05_CREATING_CHALLENGES.md) - Learn to write your own challenges
