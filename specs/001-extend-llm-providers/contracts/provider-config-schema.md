# Provider Configuration Schema

## Overview
Schema for configuring new LLM providers in LLM Verifier.

## Provider Config Structure

```json
{
  "$schema": "https://llm-verifier/schemas/provider-config.json",
  "providers": {
    "groq": {
      "name": "Groq",
      "endpoint": "https://api.groq.com/openai/v1",
      "api_key": "${GROQ_API_KEY}",
      "models": ["llama2-70b-4096", "llama2-7b-2048"],
      "pricing": {
        "input_price_per_token": 0.00001,
        "output_price_per_token": 0.00002,
        "currency": "USD"
      },
      "limits": {
        "requests_per_minute": 30,
        "tokens_per_minute": 10000
      }
    },
    "togetherai": {
      "name": "Together AI",
      "endpoint": "https://api.together.xyz/v1",
      "api_key": "${TOGETHER_API_KEY}",
      "models": ["meta-llama/Llama-2-70b-chat-hf"],
      "pricing": {
        "input_price_per_token": 0.00002,
        "output_price_per_token": 0.00002,
        "currency": "USD"
      },
      "limits": {
        "requests_per_minute": 10,
        "tokens_per_minute": 5000
      }
    },
    "fireworks": {
      "name": "Fireworks AI",
      "endpoint": "https://api.fireworks.ai/inference/v1",
      "api_key": "${FIREWORKS_API_KEY}",
      "models": ["accounts/fireworks/models/llama-v2-7b-chat"],
      "pricing": {
        "input_price_per_token": 0.00001,
        "output_price_per_token": 0.00001,
        "currency": "USD"
      },
      "limits": {
        "requests_per_minute": 100,
        "tokens_per_minute": 20000
      }
    },
    "poe": {
      "name": "Poe",
      "endpoint": "https://api.poe.com/v1",
      "api_key": "${POE_API_KEY}",
      "models": ["GPT-4", "Claude-3-Opus"],
      "pricing": {
        "input_price_per_token": 0.00003,
        "output_price_per_token": 0.00006,
        "currency": "USD"
      },
      "limits": {
        "requests_per_minute": 60,
        "tokens_per_minute": 15000
      }
    },
    "navigator": {
      "name": "NaviGator AI",
      "endpoint": "https://api.ai.it.ufl.edu/v1",
      "api_key": "${NAVIGATOR_API_KEY}",
      "models": ["mistral-small-3.1"],
      "pricing": {
        "input_price_per_token": 0.000005,
        "output_price_per_token": 0.00001,
        "currency": "USD"
      },
      "limits": {
        "requests_per_minute": 20,
        "tokens_per_minute": 10000
      }
    }
  }
}
```

## Validation Rules

- `name`: String, required, unique
- `endpoint`: String, required, valid URL
- `api_key`: String, required, environment variable reference
- `models`: Array of strings, required, non-empty
- `pricing`: Object with input_price, output_price (floats >= 0), currency
- `limits`: Object with requests_per_minute, tokens_per_minute (integers > 0)