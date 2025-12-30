# LLM Verifier API Reference Manual

<p align="center">
  <img src="images/Logo.jpeg" alt="LLMsVerifier Logo" width="150" height="150">
</p>

<p align="center">
  <strong>Verify. Monitor. Optimize.</strong>
</p>

---

## Overview

The LLM Verifier API provides comprehensive programmatic access to all verification capabilities. This reference manual covers all endpoints, request/response formats, authentication, and integration patterns.

## Base URL and Authentication

### Base URL
```
https://your-verifier-instance.com/api/v1
```

All API requests must include authentication via Bearer token:

```
Authorization: Bearer <your-jwt-token>
```

### Obtaining Tokens

#### Login Endpoint
```http
POST /auth/login
Content-Type: application/json

{
  "username": "your-username",
  "password": "your-password"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 86400,
  "token_type": "Bearer"
}
```

#### Refresh Token
```http
POST /auth/refresh
Authorization: Bearer <current-token>

{
  "refresh_token": "refresh-token-here"
}
```

## Core Resources

### Providers

#### List Providers
```http
GET /providers
Authorization: Bearer <token>
```

**Query Parameters:**
- `status` - Filter by status (active, inactive)
- `limit` - Maximum results (default: 50)
- `offset` - Pagination offset (default: 0)

**Response:**
```json
{
  "providers": [
    {
      "id": "openai",
      "name": "OpenAI",
      "status": "active",
      "endpoint": "https://api.openai.com/v1",
      "models_count": 15,
      "last_health_check": "2025-12-25T20:00:00Z",
      "response_time_ms": 245,
      "error_rate": 0.001
    }
  ],
  "total": 18,
  "limit": 50,
  "offset": 0
}
```

#### Get Provider Details
```http
GET /providers/{provider_id}
Authorization: Bearer <token>
```

**Response:**
```json
{
  "id": "openai",
  "name": "OpenAI",
  "status": "active",
  "endpoint": "https://api.openai.com/v1",
  "description": "Industry-leading GPT models",
  "website": "https://openai.com",
  "models": [
    {
      "id": "gpt-4",
      "name": "GPT-4",
      "capabilities": ["text-generation", "code", "reasoning"],
      "pricing": {
        "input_per_1k_tokens": 0.03,
        "output_per_1k_tokens": 0.06
      }
    }
  ],
  "limits": {
    "requests_per_minute": 100,
    "tokens_per_minute": 10000
  },
  "health_metrics": {
    "uptime_percentage": 99.9,
    "average_response_time": 245,
    "error_rate": 0.001
  }
}
```

#### Update Provider Configuration
```http
PUT /providers/{provider_id}
Authorization: Bearer <token>
Content-Type: application/json

{
  "api_key": "new-api-key",
  "status": "active",
  "limits": {
    "requests_per_minute": 200
  }
}
```

### Models

#### List Models
```http
GET /models
Authorization: Bearer <token>
```

**Query Parameters:**
- `provider` - Filter by provider ID
- `capability` - Filter by capability (code, text, reasoning)
- `min_score` - Minimum performance score (0-100)
- `sort` - Sort field (name, score, speed, cost)
- `order` - Sort order (asc, desc)
- `limit` - Results per page
- `offset` - Pagination offset

**Response:**
```json
{
  "models": [
    {
      "id": "gpt-4",
      "name": "GPT-4",
      "provider": "openai",
      "capabilities": ["text-generation", "code", "reasoning"],
      "performance": {
        "overall_score": 95.2,
        "speed_score": 88.5,
        "accuracy_score": 97.1,
        "cost_score": 75.3
      },
      "pricing": {
        "input_per_1k_tokens": 0.03,
        "output_per_1k_tokens": 0.06,
        "currency": "USD"
      },
      "limits": {
        "max_tokens": 8192,
        "max_context": 128000
      },
      "last_verified": "2025-12-25T19:30:00Z"
    }
  ],
  "total": 156,
  "filters_applied": {
    "provider": "openai",
    "min_score": 90
  }
}
```

#### Get Model Details
```http
GET /models/{model_id}
Authorization: Bearer <token>
```

**Response:** Detailed model information including:
- Full capability assessment
- Historical performance data
- Pricing breakdowns
- Usage statistics
- Verification history

#### Compare Models
```http
POST /models/compare
Authorization: Bearer <token>
Content-Type: application/json

{
  "model_ids": ["gpt-4", "claude-3-opus", "gemini-pro"],
  "criteria": ["performance", "cost", "speed"],
  "test_scenario": "code_generation"
}
```

**Response:**
```json
{
  "comparison": {
    "models": ["gpt-4", "claude-3-opus", "gemini-pro"],
    "criteria": ["performance", "cost", "speed"],
    "results": {
      "gpt-4": {
        "performance_score": 96.2,
        "cost_per_1k_tokens": 0.09,
        "average_response_time": 1250
      },
      "claude-3-opus": {
        "performance_score": 95.8,
        "cost_per_1k_tokens": 0.015,
        "average_response_time": 980
      },
      "gemini-pro": {
        "performance_score": 93.5,
        "cost_per_1k_tokens": 0.001,
        "average_response_time": 450
      }
    },
    "recommendations": {
      "best_performance": "gpt-4",
      "best_value": "gemini-pro",
      "best_speed": "gemini-pro"
    }
  }
}
```

### Verifications

#### Create Verification
```http
POST /verifications
Authorization: Bearer <token>
Content-Type: application/json

{
  "model_ids": ["gpt-4"],
  "test_types": ["basic", "performance"],
  "test_scenario": "code_generation",
  "parameters": {
    "iterations": 5,
    "timeout_seconds": 30,
    "custom_prompts": [
      "Write a Python function to calculate fibonacci numbers"
    ]
  },
  "schedule": {
    "type": "immediate"
  }
}
```

**Response:**
```json
{
  "verification_id": "ver_1234567890",
  "status": "queued",
  "estimated_duration": 120,
  "models": ["gpt-4"],
  "created_at": "2025-12-25T20:00:00Z"
}
```

#### Get Verification Status
```http
GET /verifications/{verification_id}
Authorization: Bearer <token>
```

**Response:**
```json
{
  "id": "ver_1234567890",
  "status": "running",
  "progress": {
    "completed": 3,
    "total": 5,
    "percentage": 60
  },
  "current_test": "performance_test",
  "results": {
    "gpt-4": {
      "basic_test": {
        "status": "completed",
        "score": 98.5,
        "response_time": 245
      },
      "performance_test": {
        "status": "running",
        "progress": 60
      }
    }
  },
  "started_at": "2025-12-25T20:00:00Z",
  "estimated_completion": "2025-12-25T20:02:00Z"
}
```

#### List Verifications
```http
GET /verifications
Authorization: Bearer <token>
```

**Query Parameters:**
- `status` - Filter by status (queued, running, completed, failed)
- `model_id` - Filter by model
- `date_from` - Start date (ISO 8601)
- `date_to` - End date (ISO 8601)
- `limit` - Results per page
- `offset` - Pagination offset

### Reports

#### Generate Report
```http
POST /reports
Authorization: Bearer <token>
Content-Type: application/json

{
  "type": "model_comparison",
  "parameters": {
    "model_ids": ["gpt-4", "claude-3-opus"],
    "date_range": {
      "from": "2025-12-01T00:00:00Z",
      "to": "2025-12-25T23:59:59Z"
    },
    "metrics": ["performance", "cost", "speed"],
    "format": "pdf"
  }
}
```

**Response:**
```json
{
  "report_id": "rpt_1234567890",
  "status": "generating",
  "estimated_completion": "2025-12-25T20:01:00Z",
  "download_url": "/reports/rpt_1234567890/download"
}
```

#### Download Report
```http
GET /reports/{report_id}/download
Authorization: Bearer <token>
```

**Headers:**
```
Accept: application/pdf  # or application/json, text/csv
```

## WebSocket Endpoints

### Real-time Verification Updates
```javascript
const ws = new WebSocket('wss://your-verifier-instance.com/api/v1/ws/verifications/{verification_id}');

ws.onmessage = (event) => {
  const update = JSON.parse(event.data);
  console.log('Verification update:', update);
};
```

**Message Format:**
```json
{
  "type": "verification_progress",
  "verification_id": "ver_1234567890",
  "progress": {
    "completed": 4,
    "total": 5,
    "percentage": 80
  },
  "current_results": {
    "gpt-4": {
      "score": 97.2,
      "response_time": 234
    }
  }
}
```

## Error Handling

### Standard Error Response
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid model ID provided",
    "details": {
      "field": "model_id",
      "provided": "invalid-model",
      "valid_options": ["gpt-4", "claude-3-opus", "gemini-pro"]
    },
    "timestamp": "2025-12-25T20:00:00Z",
    "request_id": "req_1234567890"
  }
}
```

### HTTP Status Codes
- `200 OK` - Success
- `201 Created` - Resource created
- `400 Bad Request` - Invalid request
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Resource not found
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error
- `503 Service Unavailable` - Service temporarily unavailable

### Rate Limiting

Rate limit headers are included in all responses:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640995200
X-RateLimit-Retry-After: 60
```

## SDKs and Libraries

### Official SDKs

#### Python SDK
```python
from llm_verifier import Client

client = Client(api_key="your-jwt-token")

# List models
models = client.models.list(provider="openai")

# Run verification
result = client.verifications.create(
    model_ids=["gpt-4"],
    test_types=["basic", "performance"]
)

# Get results
status = client.verifications.get(result.id)
```

#### JavaScript SDK
```javascript
import { LLMVerifier } from 'llm-verifier-sdk';

const client = new LLMVerifier({
  apiKey: 'your-jwt-token'
});

// List providers
const providers = await client.providers.list();

// Compare models
const comparison = await client.models.compare({
  modelIds: ['gpt-4', 'claude-3-opus'],
  criteria: ['performance', 'cost']
});
```

### Community Libraries
- **Go SDK**: `go get github.com/your-org/llm-verifier-go`
- **Java SDK**: Available on Maven Central
- **.NET SDK**: Available on NuGet

## Best Practices

### Authentication
- Store JWT tokens securely
- Implement token refresh logic
- Handle token expiration gracefully
- Use HTTPS for all requests

### Error Handling
- Implement exponential backoff for retries
- Handle rate limiting appropriately
- Log errors with sufficient context
- Provide meaningful error messages to users

### Performance
- Use WebSocket connections for real-time updates
- Implement request batching for bulk operations
- Cache frequently accessed data
- Monitor API usage and optimize queries

### Security
- Never expose API keys in client-side code
- Validate all input data
- Use HTTPS for all communications
- Implement proper authentication flows

This API reference provides comprehensive documentation for integrating with LLM Verifier. For additional support, visit our developer portal or contact the development team.