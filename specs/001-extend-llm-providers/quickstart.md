# Quickstart: Extend LLM Providers Support

## Overview
This guide helps you add support for 5 new LLM providers to your LLM Verifier installation.

## Prerequisites
- LLM Verifier installed and running
- API keys for desired providers
- Go 1.21+ development environment

## Step 1: Obtain API Keys

Get API keys from each provider:

### Groq
1. Visit https://console.groq.com
2. Sign up for an account
3. Generate an API key
4. Set environment variable: `export GROQ_API_KEY=your_key_here`

### Together AI
1. Visit https://api.together.xyz
2. Create account and get API key
3. Set: `export TOGETHER_API_KEY=your_key_here`

### Fireworks AI
1. Visit https://fireworks.ai
2. Sign up and get API key
3. Set: `export FIREWORKS_API_KEY=your_key_here`

### Poe
1. Visit https://poe.com/developer
2. Get OpenAI-compatible API key
3. Set: `export POE_API_KEY=your_key_here`

### NaviGator AI
1. Visit https://ai.it.ufl.edu
2. Obtain API access
3. Set: `export NAVIGATOR_API_KEY=your_key_here`

## Step 2: Update Configuration

Add new providers to your `config.yaml`:

```yaml
llms:
  # Existing providers...
  
  # New providers
  - name: "groq-llama2-70b"
    provider: "groq"
    api_key: "${GROQ_API_KEY}"
    model: "llama2-70b-4096"
    enabled: true
    
  - name: "together-llama2-70b"
    provider: "togetherai"
    api_key: "${TOGETHER_API_KEY}"
    model: "meta-llama/Llama-2-70b-chat-hf"
    enabled: true
    
  - name: "fireworks-llama2-7b"
    provider: "fireworks"
    api_key: "${FIREWORKS_API_KEY}"
    model: "accounts/fireworks/models/llama-v2-7b-chat"
    enabled: true
    
  - name: "poe-gpt4"
    provider: "poe"
    api_key: "${POE_API_KEY}"
    model: "GPT-4"
    enabled: true
    
  - name: "navigator-mistral"
    provider: "navigator"
    api_key: "${NAVIGATOR_API_KEY}"
    model: "mistral-small-3.1"
    enabled: true
```

## Step 3: Deploy and Test

1. Restart LLM Verifier with new configuration
2. Run verification tests:
   ```bash
   go test ./tests/... -v -run "TestProvider"
   ```
3. Verify models are discovered:
   ```bash
   curl http://localhost:8080/api/v1/models
   ```
4. Run full verification:
   ```bash
   go run cmd/main.go verify --all
   ```

## Step 4: Export Configurations

Generate configs for external tools:

```bash
# OpenCode config
go run cmd/main.go export --format opencode --providers groq,togetherai,fireworks,poe,navigator

# Crush config  
go run cmd/main.go export --format crush --providers groq,togetherai,fireworks,poe,navigator

# Claude Code config
go run cmd/main.go export --format claude-code --providers groq,togetherai,fireworks,poe,navigator
```

## Troubleshooting

### Provider Not Found
- Check API key is set correctly
- Verify endpoint URLs in provider code
- Check provider service status

### Verification Fails
- Review API rate limits
- Check model availability
- Validate API key permissions

### Export Fails
- Ensure provider is fully implemented
- Check config export logic for new providers

## Next Steps
- Monitor provider performance
- Update documentation with new providers
- Consider adding more providers from the research documents