# Provider Endpoint Research - API URLs from Documentation

## Objective
Find OpenAI-compatible API endpoint URLs for NEW providers in .env file.

## New Providers Identified

### 1. Sarvam_AI_India
**API Base URL Investigation:**

Check documentation:
```bash
curl -s https://docs.sarvam.ai | grep -i "api"
```

Expected pattern: `https://api.sarvam.ai/v1` or similar

**Research Status:** 🔍 TODO

### 2. Vulavula
**API Base URL Investigation:**

Check documentation:
```bash
curl -s https://docs.vulavula.com | grep -i "api endpoint"
```

Expected pattern: `https://api.vulavula.com/v1` or similar

**Research Status:** 🔍 TODO

### 3. Vercel_Ai_Gateway
**API Base URL Known:**
From .env comment: Available
URL: `https://api.vercel.com/v1` (based on Vercel patterns)

**Status:** ✅ KNOWN (from .env)

### 4. Modal_Token_Secret
**API Base URL:**
URL: `https://api.modal.com/v1` (from .env curl examples)

**Status:** ✅ KNOWN (from .env)

## Research Commands

```bash
# For each new provider, run:
# 1. Check official documentation
# 2. Look for API endpoint in docs
# 3. Verify OpenAI compatibility
# 4. Test with curl

# Example for Sarvam:
curl --location 'https://api.sarvam.ai/v1/chat/completions' \
  --header 'Authorization: Bearer $API_KEY' \
  --header 'Content-Type: application/json' \
  --data '{
    "model": "your-model",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

## Expected API Patterns
- Sarvam: `https://api.sarvam.ai/v1` (inferred)
- Vulavula: `https://api.vulavula.com/v1` (inferred)
- Vercel: `https://api.vercel.com/v1` (inferred)

All should be OpenAI-compatible endpoints.