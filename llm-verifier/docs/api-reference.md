# LLM Verifier API Documentation

## Overview

The LLM Verifier provides a comprehensive REST API for managing LLM providers, models, and verification processes.

## Base URL
```
http://localhost:8080/api
```

## Authentication
All API endpoints require authentication via JWT token in the Authorization header:
```
Authorization: Bearer <jwt_token>
```

## Endpoints

### Health Check

#### GET /api/health
Returns the health status of the service.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-12-25T20:52:37Z",
  "version": "1.0.0"
}
```

### Models

#### GET /api/models
List all available models across all providers.

**Response:**
```json
{
  "models": [
    {
      "id": "gpt-4",
      "name": "GPT-4",
      "provider": "openai",
      "capabilities": ["text-generation", "chat"]
    }
  ]
}
```

#### GET /api/models/{id}
Get details for a specific model.

**Parameters:**
- `id` (path): Model ID

**Response:**
```json
{
  "id": "gpt-4",
  "name": "GPT-4",
  "provider": "openai",
  "max_tokens": 8192,
  "capabilities": ["text-generation", "chat"],
  "pricing": {
    "input_per_token": 0.00003,
    "output_per_token": 0.00006
  }
}
```

#### POST /api/models/{id}/verify
Run verification on a specific model.

**Parameters:**
- `id` (path): Model ID

**Request Body:**
```json
{
  "test_prompt": "Write a short poem about AI",
  "test_type": "creativity"
}
```

**Response:**
```json
{
  "model_id": "gpt-4",
  "score": 95.5,
  "status": "completed",
  "results": {
    "creativity": 9.2,
    "coherence": 9.8,
    "relevance": 9.5
  }
}
```

### Providers

#### GET /api/providers
List all configured providers.

**Response:**
```json
{
  "providers": [
    {
      "id": 1,
      "name": "openai",
      "endpoint": "https://api.openai.com/v1",
      "status": "active"
    }
  ]
}
```

#### POST /api/providers
Add a new provider.

**Request Body:**
```json
{
  "name": "groq",
  "endpoint": "https://api.groq.com/openai/v1",
  "api_key": "your-api-key",
  "models": ["llama2-70b-4096"]
}
```

**Response:**
```json
{
  "id": 2,
  "name": "groq",
  "status": "active"
}
```

## Error Responses

All errors follow this format:
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid API key provided",
    "details": {}
  }
}
```

## Rate Limiting

- 100 requests per minute per IP
- Burst limit: 200 requests

Rate limit headers are included in responses:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640995200
```

## Supported Providers

The API supports verification across 17+ LLM providers:
- OpenAI, Anthropic, Google, Cohere, Meta
- Groq, Together AI, Fireworks AI, Poe, NaviGator AI
- Mistral, xAI, SiliconFlow
- And extensible framework for additional providers