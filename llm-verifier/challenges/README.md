# Challenge Tests

This directory contains end-to-end "Challenge" tests that verify the system's functionality using production-ready binaries, not source code.

## Challenge Types

### 1. Provider Discovery Challenge
Tests all LLM providers to:
- Discover available models
- Identify supported features (MCPs, LSPs, Embeddings, etc.)
- Verify real-world usability
- Generate configuration files

### 2. Model Verification Challenge
Tests individual models to verify:
- Chat completion functionality
- Streaming support
- Function calling
- Context handling
- Error scenarios

### 3. Feature Integration Challenge
Tests integration features:
- Multiple providers failover
- Load balancing
- Rate limiting
- Health monitoring
- Notification delivery

## Directory Structure

```
challenges/
├── .gitignore                    # Ensures API keys are NOT committed
├── provider_models_discovery.go  # Challenge runner
├── provider_models_discovery/
│   └── 2025/
│       └── 12/
│           └── 23/
│               └── [timestamp]/
│                   ├── results/
│                   │   ├── providers_opencode.json
│                   │   └── providers_crush.json
│                   └── logs/
│                       └── challenge.log
└── [challenge_name]/
    └── [year]/
        └── [month]/
            └── [day]/
                └── [timestamp]/
                    ├── results/
                    └── logs/
```

## Running Challenges

### Prerequisites
1. Build production binary: `go build -o llm-verifier ./cmd/server`
2. Create `challenges/api_keys.json` with API keys (NOT in git)
3. Run challenge: `go run challenges/provider_models_discovery.go`

### API Keys Format

Create `challenges/api_keys.json`:
```json
{
  "huggingface": "your-huggingface-key",
  "nvidia": "your-nvidia-key",
  "chutes": "your-chutes-key",
  "siliconflow": "your-siliconflow-key",
  "kimi": "your-kimi-key",
  "gemini": "your-gemini-key",
  "openrouter": "your-openrouter-key",
  "zai": "your-zai-key",
  "deepseek": "your-deepseek-key"
}
```

**IMPORTANT**: API keys file must be in `.gitignore` and NEVER committed!

## Challenge Results

Each challenge generates:
- `providers_opencode.json` - Configuration for providers
- `providers_crush.json` - Full test results with all details
- `challenge.log` - Verbose execution logs

All results must be:
- Non-empty
- Real data (no placeholders)
- Properly structured
- Git versioned (except API keys)

## Challenge Status

- ✅ Provider Models Discovery - Implemented
- ⏳ Model Verification - Planned
- ⏳ Feature Integration - Planned
