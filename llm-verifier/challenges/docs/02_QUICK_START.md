# Quick Start Guide

**Get Started with the Challenge Framework in 5 Minutes**

---

## 📖 Prerequisites

Before you begin, ensure you have:

- ✅ **Project Built** - All binaries compiled (CLI, REST API, TUI, Desktop, Mobile, Web)
- ✅ **Git Installed** - For versioning challenge results
- ✅ **Bash Shell** - For running challenge scripts
- ✅ **API Keys Available** - Set environment variables: ApiKey_HuggingFace, ApiKey_Nvidia, ApiKey_Chutes, ApiKey_SiliconFlow, ApiKey_Kimi, ApiKey_Gemini, ApiKey_OpenRouter, ApiKey_Z_AI, ApiKey_DeepSeek

### Verify Binaries

```bash
cd /path/to/llm-verifier
ls -lh llm-verifier*

# Expected output:
# llm-verifier         (38M) - CLI binary
# llm-verifier-api     (37M) - REST API binary
# llm-verifier-tui      (37M) - TUI binary
# llm-verifier-desktop  (37M) - Desktop binary
# llm-verifier-mobile    (37M) - Mobile binary
```

---

## 🚀 Step 1: List Available Challenges

View all available challenges in the Challenge Bank:

```bash
cd challenges
./run_challenges.sh --list
```

**Expected Output:**
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
  Outputs:      models_verification_opencode.json, ...

[... and more challenges]
```

---

## 🚀 Step 2: Run Your First Challenge

Execute the "Provider Models Discovery" challenge:

```bash
cd challenges
./run_challenges.sh provider_models_discovery
```

**What Happens:**
1. Challenge script creates timestamped directory
2. Configuration file is generated (`config.yaml`)
3. Platform binary is executed with configuration
4. Results are saved in `results/` subdirectory
5. All commands are logged in `logs/commands.log`
6. All activities are logged in `logs/challenge.log`

**Expected Output:**
```
[2025-12-23 19:16:00] ========================================
[2025-12-23 19:16:00] PROVIDER MODELS DISCOVERY CHALLENGE
[2025-12-23 19:16:00] ========================================
[2025-12-23 19:16:00] 
[2025-12-23 19:16:00] Challenge Directory: challenges/provider_models_discovery/2025/12/23/1766500000
[2025-12-23 19:16:00] 
[2025-12-23 19:16:00] Creating configuration file...
[2025-12-23 19:16:00] Configuration file created: challenges/provider_models_discovery/2025/12/23/1766500000/config.yaml
[2025-12-23 19:16:00] 
[2025-12-23 19:16:00] Executing binary...
[2025-12-23 19:16:00] COMMAND: ./llm-verifier -c config.yaml -o results/
[2025-12-23 19:16:00] 
[2025-12-23 19:16:00] [Binary output...]
[2025-12-23 19:16:00] 
[2025-12-23 19:16:00] ========================================
[2025-12-23 19:16:00] CHALLENGE COMPLETE
[2025-12-23 19:16:00] ========================================
[2025-12-23 19:16:00] 
[2025-12-23 19:16:00] Summary:
[2025-12-23 19:16:00]   Total Providers: 9
[2025-12-23 19:16:00]   Verified: 9 (100%)
[2025-12-23 19:16:00]   Total Models: 26
```

---

## 🚀 Step 3: View Results

Navigate to the challenge results directory:

```bash
cd challenges/provider_models_discovery/2025/12/23/<timestamp>/results

# List results
ls -la

# Expected output:
# providers_opencode.json  - Provider configuration
# providers_crush.json    - Full challenge results
```

### View Results

**providers_opencode.json:**
```json
{
  "challenge_name": "provider_models_discovery",
  "date": "2025-12-23",
  "binary": "./llm-verifier",
  "summary": {
    "total_providers": 9,
    "success_count": 9,
    "total_models": 26
  },
  "providers": [...]
}
```

**providers_crush.json:**
```json
{
  "challenge_name": "provider_models_discovery",
  "challenge_type": "binary_verification",
  "start_time": "2025-12-23 19:16:00",
  "binary_used": "./llm-verifier",
  "config_file": "config.yaml",
  "commands_executed": [...],
  "summary": {...},
  "providers_verified": [...]
}
```

---

## 🚀 Step 4: Review Logs

All challenge activities are logged:

```bash
cd challenges/provider_models_discovery/2025/12/23/<timestamp>/logs

# List logs
ls -la

# Expected output:
# challenge.log  - Verbose execution log
# commands.log   - All binary commands executed
```

### View Challenge Log

```bash
cat challenge.log | less
```

**Contains:**
- Challenge start/end times
- Configuration creation details
- Binary command execution
- All error messages (if any)
- Complete activity trail

### View Commands Log

```bash
cat commands.log
```

**Contains:**
```
[2025-12-23 19:16:00] COMMAND: ./llm-verifier -c config.yaml -o results/
[2025-12-23 19:16:00] COMMAND: ./llm-verifier ai-config export --format opencode --output results/
```

---

## 🎯 Common Quick Start Scenarios

### Scenario 1: Run Single Challenge on CLI

```bash
cd challenges
./run_challenges.sh provider_models_discovery --platform cli
```

### Scenario 2: Run Multiple Challenges

```bash
cd challenges
./run_challenges.sh provider_models_discovery model_verification --platform cli
```

### Scenario 3: Run All Challenges

```bash
cd challenges
./run_challenges.sh --all --platform cli
```

### Scenario 4: Run Challenge on Different Platform

```bash
# REST API
./run_challenges.sh provider_models_discovery --platform rest-api

# TUI
./run_challenges.sh provider_models_discovery --platform tui

# Desktop
./run_challenges.sh provider_models_discovery --platform desktop
```

---

## 🔍 Verifying Results

### Check Results Directory Structure

```bash
# Verify structure
cd challenges/provider_models_discovery/2025/12/23/<timestamp>
tree -L 2

# Expected:
# .
# ├── config.yaml
# ├── logs/
# │   ├── challenge.log
# │   └── commands.log
# └── results/
#     ├── providers_opencode.json
#     └── providers_crush.json
```

### Validate JSON Results

```bash
# Validate JSON format
cd results
jq . providers_opencode.json
jq . providers_crush.json

# Expected: Parsed JSON (no errors)
```

### Check Logs for Errors

```bash
# Search for errors in logs
cd logs
grep -i "error\|failed\|exception" challenge.log

# Expected: No errors (or specific error messages for debugging)
```

---

## ✅ Success Criteria

You have successfully run your first challenge if:

✅ **Challenge directory created** - `challenges/<name>/YYYY/MM/DD/timestamp/`
✅ **Logs directory exists** - `logs/` with `challenge.log` and `commands.log`
✅ **Results directory exists** - `results/` with JSON files
✅ **Commands logged** - `commands.log` contains all binary commands
✅ **JSON files valid** - Can parse `providers_opencode.json` and `providers_crush.json`
✅ **No critical errors** - Logs don't contain fatal errors

---

## 📚 Next Steps

1. [Read All Challenges](#read-all-challenges) - Explore available challenges
2. [Create Custom Challenge](#create-custom-challenge) - Write your own challenges
3. [Run Multiple Platforms](#run-multiple-platforms) - Test on all platforms
4. [Automate Challenges](#automate-challenges) - Schedule automatic runs

### Read All Challenges

```bash
cd challenges
./run_challenges.sh --list
```

### Create Custom Challenge

```bash
cd challenges
mkdir -p challenge_runners/my_custom_challenge
# Edit challenges_bank.json to add your challenge
# Run your challenge
./run_challenges.sh my_custom_challenge
```

### Run Multiple Platforms

```bash
cd challenges

# Run on CLI
./run_challenges.sh provider_models_discovery --platform cli

# Run on REST API
./run_challenges.sh provider_models_discovery --platform rest-api

# Run on TUI
./run_challenges.sh provider_models_discovery --platform tui
```

---

## 🆘 Troubleshooting

### Challenge Not Found

**Error:** `Challenge not found: provider_models_discovery`

**Solution:**
```bash
# List available challenges
./run_challenges.sh --list

# Verify correct challenge name
cat challenges_bank.json | grep '"id"'
```

### Binary Not Found

**Error:** `Binary not found: llm-verifier`

**Solution:**
```bash
# Build the project first
cd ..
go build -o llm-verifier cmd/main.go

# Verify binary exists
ls -la llm-verifier*

# Return to challenges directory
cd challenges
```

### Permission Denied

**Error:** `./llm-verifier: Permission denied`

**Solution:**
```bash
# Make binary executable
chmod +x ../llm-verifier
```

### JSON Parse Error

**Error:** `providers_opencode.json: parse error`

**Solution:**
```bash
# Validate JSON
jq . providers_opencode.json

# If error, check logs
cat logs/challenge.log | tail -50

# Re-run challenge
./run_challenges.sh provider_models_discovery
```

---

## 🎉 Congratulations

You have successfully:

✅ Listed all available challenges  
✅ Run your first challenge  
✅ Viewed challenge results  
✅ Reviewed challenge logs  
✅ Verified challenge output  

**You are now ready to:**
- Run multiple challenges
- Test on different platforms
- Create custom challenges
- Automate challenge execution

---

**Next:** [Challenge Runner Guide](04_CHALLENGE_RUNNER_GUIDE.md) - Learn advanced usage
