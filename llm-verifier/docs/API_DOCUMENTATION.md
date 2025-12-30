# LLM Verifier REST API Documentation

<p align="center">
  <img src="images/Logo.jpeg" alt="LLMsVerifier Logo" width="150" height="150">
</p>

## Overview

The LLM Verifier REST API provides programmatic access to all functionality of the LLM Verifier system. It is built using the GinGonic framework and provides a comprehensive set of endpoints for managing models, providers, verification results, and system configuration.

### Features

- **Model Management**: CRUD operations for LLM models and providers
- **Verification**: Trigger and monitor model verification processes
- **Reporting**: Generate detailed reports in multiple formats
- **Configuration**: Runtime configuration management
- **Authentication**: JWT-based secure authentication
- **Rate Limiting**: Built-in protection against abuse
- **Brotli Compression Support**: Detect and report Brotli compression capabilities
- **Swagger Documentation**: Interactive API documentation at `/swagger/index.html`

### Architecture

The API follows RESTful principles with the following key components:

- **Authentication Middleware**: JWT token validation
- **Rate Limiting Middleware**: Request throttling
- **CORS Support**: Cross-origin resource sharing
- **Structured Error Responses**: Consistent error handling
- **Pagination**: Efficient data retrieval for large datasets
- **Validation**: Input sanitization and validation

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

The API supports JWT-based authentication for secure access.

### Login

```http
POST /auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "password"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 3600,
  "user": {
    "id": 1,
    "username": "admin",
    "role": "admin"
  }
}
```

### Using Authentication

Include the JWT token in the Authorization header for subsequent requests:

```http
GET /models
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

## Rate Limiting

The API implements rate limiting to prevent abuse:

- **Authenticated requests**: 1000 requests per hour
- **Unauthenticated requests**: 100 requests per hour

Rate limit information is included in response headers:
```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1640995200
```

## Response Format

All responses follow a consistent format:

### Success Response
```json
{
  "success": true,
  "data": {
    // Response data
  },
  "meta": {
    "total": 100,
    "page": 1,
    "per_page": 20,
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
```

### Error Response
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validation failed",
    "details": {
      "field": "email",
      "message": "Email is required"
    }
  },
  "meta": {
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
```

## API Endpoints

### Models

#### List Models
```http
GET /models
```

**Query Parameters:**
- `search` (string): Search term for model name or description
- `provider_id` (int): Filter by provider ID
- `min_score` (float): Minimum overall score
- `verification_status` (string): Filter by verification status
- `supports_tool_use` (bool): Filter by tool use support
- `supports_code_generation` (bool): Filter by code generation support
- `page` (int): Page number (default: 1)
- `per_page` (int): Items per page (default: 20)
- `sort` (string): Sort field (name, score, created_at)
- `order` (string): Sort order (asc, desc)

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "provider_id": 1,
      "model_id": "gpt-4-turbo",
      "name": "GPT-4 Turbo",
      "description": "OpenAI's most capable model",
      "version": "2024-04",
      "architecture": "transformer",
      "parameter_count": 170000000000,
      "context_window_tokens": 128000,
      "max_output_tokens": 4096,
      "is_multimodal": true,
      "supports_vision": true,
      "supports_audio": false,
      "supports_video": false,
      "supports_reasoning": true,
      "open_source": false,
      "deprecated": false,
      "tags": ["gpt", "openai", "turbo"],
      "language_support": ["en", "es", "fr", "de", "it", "pt", "nl", "ru", "ja", "ko", "zh"],
      "use_case": "general_purpose",
      "verification_status": "verified",
      "overall_score": 92.5,
      "code_capability_score": 95.0,
      "responsiveness_score": 88.0,
      "reliability_score": 94.0,
      "feature_richness_score": 91.0,
      "value_proposition_score": 89.0,
      "last_verified": "2024-01-01T00:00:00Z",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "meta": {
    "total": 50,
    "page": 1,
    "per_page": 20
  }
}
```

#### Get Model
```http
GET /models/{model_id}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": 1,
    "provider_id": 1,
    "model_id": "gpt-4-turbo",
    "name": "GPT-4 Turbo",
    "description": "OpenAI's most capable model",
    "version": "2024-04",
    "architecture": "transformer",
    "parameter_count": 170000000000,
    "context_window_tokens": 128000,
    "max_output_tokens": 4096,
    "is_multimodal": true,
    "supports_vision": true,
    "supports_audio": false,
    "supports_video": false,
    "supports_reasoning": true,
    "supports_brotli": true,
    "supports_brotli": true,
    "open_source": false,
    "deprecated": false,
    "tags": ["gpt", "openai", "turbo"],
    "language_support": ["en", "es", "fr", "de", "it", "pt", "nl", "ru", "ja", "ko", "zh"],
    "use_case": "general_purpose",
    "verification_status": "verified",
    "overall_score": 92.5,
    "code_capability_score": 95.0,
    "responsiveness_score": 88.0,
    "reliability_score": 94.0,
    "feature_richness_score": 91.0,
    "value_proposition_score": 89.0,
    "last_verified": "2024-01-01T00:00:00Z",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

#### Create Model
```http
POST /models
Content-Type: application/json

{
  "provider_id": 1,
  "model_id": "custom-model",
  "name": "Custom Model",
  "description": "A custom language model",
  "version": "1.0",
  "architecture": "transformer",
  "parameter_count": 7000000000,
  "context_window_tokens": 8192,
  "max_output_tokens": 2048,
  "is_multimodal": false,
  "supports_vision": false,
  "supports_audio": false,
  "supports_video": false,
  "supports_reasoning": false,
  "open_source": true,
  "deprecated": false,
  "tags": ["custom", "open-source"],
  "language_support": ["en"],
  "use_case": "specialized"
}
```

#### Update Model
```http
PUT /models/{model_id}
Content-Type: application/json

{
  "name": "Updated Model Name",
  "description": "Updated description",
  "deprecated": true
}
```

#### Delete Model
```http
DELETE /models/{model_id}
```

#### Trigger Model Verification
```http
POST /models/{model_id}/verify
```

**Response:**
```json
{
  "success": true,
  "data": {
    "verification_id": 123,
    "status": "started",
    "message": "Verification started for model"
  }
}
```

#### Get Model Verification Results
```http
GET /models/{model_id}/results
```

### Providers

#### List Providers
```http
GET /providers
```

**Query Parameters:**
- `search` (string): Search term for provider name
- `is_active` (bool): Filter by active status
- `page` (int): Page number
- `per_page` (int): Items per page

#### Get Provider
```http
GET /providers/{provider_id}
```

#### Create Provider
```http
POST /providers
Content-Type: application/json

{
  "name": "Custom Provider",
  "endpoint": "https://api.custom-provider.com/v1",
  "api_key_encrypted": "encrypted_api_key",
  "description": "A custom LLM provider",
  "website": "https://custom-provider.com",
  "support_email": "support@custom-provider.com",
  "documentation_url": "https://docs.custom-provider.com"
}
```

#### Update Provider
```http
PUT /providers/{provider_id}
Content-Type: application/json

{
  "name": "Updated Provider Name",
  "is_active": false
}
```

#### Delete Provider
```http
DELETE /providers/{provider_id}
```

### Verification Results

#### List Verification Results
```http
GET /verification-results
```

**Query Parameters:**
- `model_id` (int): Filter by model ID
- `status` (string): Filter by status (running, completed, failed)
- `from_date` (date): Filter results from this date
- `to_date` (date): Filter results to this date
- `min_score` (float): Minimum overall score
- `page` (int): Page number
- `per_page` (int): Items per page

#### Get Verification Result
```http
GET /verification-results/{result_id}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": 123,
    "model_id": 1,
    "verification_type": "full",
    "started_at": "2024-01-01T00:00:00Z",
    "completed_at": "2024-01-01T00:05:00Z",
    "status": "completed",
    "exists": true,
    "responsive": true,
    "overloaded": false,
    "latency_ms": 150,
    "supports_tool_use": true,
    "supports_function_calling": true,
    "supports_code_generation": true,
    "supports_code_completion": true,
    "supports_code_review": true,
    "supports_code_explanation": true,
    "supports_embeddings": true,
    "supports_reranking": false,
    "supports_image_generation": false,
    "supports_audio_generation": false,
    "supports_video_generation": false,
    "supports_mcps": true,
    "supports_lsps": true,
    "supports_multimodal": true,
    "supports_streaming": true,
    "supports_json_mode": true,
    "supports_structured_output": true,
    "supports_reasoning": true,
    "supports_parallel_tool_use": true,
    "max_parallel_calls": 10,
    "supports_batch_processing": false,
    "code_language_support": ["python", "javascript", "go", "java", "cpp"],
    "code_debugging": true,
    "code_optimization": true,
    "test_generation": true,
    "documentation_generation": true,
    "refactoring": true,
    "error_resolution": true,
    "architecture_design": true,
    "security_assessment": true,
    "pattern_recognition": true,
    "debugging_accuracy": 0.85,
    "max_handled_depth": 5,
    "code_quality_score": 0.92,
    "logic_correctness_score": 0.88,
    "runtime_efficiency_score": 0.85,
    "overall_score": 92.5,
    "code_capability_score": 95.0,
    "responsiveness_score": 88.0,
    "reliability_score": 94.0,
    "feature_richness_score": 91.0,
    "value_proposition_score": 89.0,
    "avg_latency_ms": 145,
    "p95_latency_ms": 200,
    "min_latency_ms": 100,
    "max_latency_ms": 300,
    "throughput_rps": 6.9,
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

#### Get Latest Verification Results
```http
GET /verification-results/latest
```

### Configuration Exports

#### List Configuration Exports
```http
GET /config-exports
```

#### Create Configuration Export
```http
POST /config-exports
Content-Type: application/json

{
  "export_type": "opencode",
  "name": "High Quality Models",
  "description": "Models with score >= 80",
  "target_models": [1, 2, 3],
  "filters": {
    "min_score": 80,
    "supports_code_generation": true,
    "supports_tool_use": true
  }
}
```

#### Download Configuration Export
```http
GET /config-exports/{export_id}/download
```

**Response:** Configuration file content (JSON format)

#### Get Export for Specific Tool
```http
GET /config-exports/opencode
```

**Query Parameters:**
- `min_score` (float): Minimum score filter
- `supports_tool_use` (bool): Tool use support filter
- `supports_code_generation` (bool): Code generation support filter

### Schedules

#### List Schedules
```http
GET /schedules
```

#### Create Schedule
```http
POST /schedules
Content-Type: application/json

{
  "name": "Daily Verification",
  "description": "Daily verification of all models",
  "schedule_type": "cron",
  "cron_expression": "0 2 * * *",
  "target_type": "all_models",
  "target_id": null,
  "is_active": true
}
```

#### Update Schedule
```http
PUT /schedules/{schedule_id}
Content-Type: application/json

{
  "name": "Updated Schedule Name",
  "is_active": false
}
```

#### Delete Schedule
```http
DELETE /schedules/{schedule_id}
```

#### Trigger Schedule Manually
```http
POST /schedules/{schedule_id}/run
```

### Events and Notifications

#### List Events
```http
GET /events
```

**Query Parameters:**
- `event_type` (string): Filter by event type
- `severity` (string): Filter by severity (debug, info, warning, error, critical)
- `model_id` (int): Filter by model ID
- `provider_id` (int): Filter by provider ID
- `from_date` (date): Filter events from this date
- `to_date` (date): Filter events to this date
- `page` (int): Page number
- `per_page` (int): Items per page

#### Subscribe to Events
```http
POST /events/subscribe
Content-Type: application/json

{
  "event_types": ["verification_completed", "issue_detected"],
  "model_ids": [1, 2, 3],
  "webhook_url": "https://your-webhook.com/events",
  "webhook_secret": "your-webhook-secret"
}
```

#### Unsubscribe from Events
```http
DELETE /events/subscribe/{subscription_id}
```

#### Get Notification Settings
```http
GET /notifications/settings
```

#### Update Notification Settings
```http
PUT /notifications/settings
Content-Type: application/json

{
  "slack": {
    "enabled": true,
    "webhook_url": "https://hooks.slack.com/services/...",
    "channel": "#llm-alerts"
  },
  "email": {
    "enabled": true,
    "smtp_host": "smtp.gmail.com",
    "smtp_port": 587,
    "username": "user@gmail.com",
    "password": "app-password",
    "from": "llm-verifier@domain.com",
    "to": ["admin@domain.com"]
  },
  "telegram": {
    "enabled": true,
    "bot_token": "bot-token",
    "chat_id": "chat-id"
  }
}
```

### Issues

#### List Issues
```http
GET /issues
```

**Query Parameters:**
- `model_id` (int): Filter by model ID
- `severity` (string): Filter by severity (critical, high, medium, low)
- `status` (string): Filter by status (open, resolved)
- `issue_type` (string): Filter by issue type

#### Get Issue
```http
GET /issues/{issue_id}
```

#### Create Issue
```http
POST /issues
Content-Type: application/json

{
  "model_id": 1,
  "issue_type": "performance",
  "severity": "high",
  "title": "High latency detected",
  "description": "Model is showing consistently high response times",
  "symptoms": "Response times > 5 seconds",
  "workarounds": "Retry requests with exponential backoff",
  "affected_features": ["streaming", "real_time"]
}
```

#### Update Issue
```http
PUT /issues/{issue_id}
Content-Type: application/json

{
  "severity": "critical",
  "resolved_at": "2024-01-01T00:00:00Z",
  "resolution_notes": "Issue resolved after API provider fix"
}
```

### System

#### Health Check
```http
GET /health
```

**Response:**
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "timestamp": "2024-01-01T00:00:00Z",
    "version": "1.0.0",
    "database": "connected",
    "uptime": 3600
  }
}
```

#### System Information
```http
GET /system/info
```

**Response:**
```json
{
  "success": true,
  "data": {
    "version": "1.0.0",
    "build_time": "2024-01-01T00:00:00Z",
    "go_version": "1.21.0",
    "git_commit": "abc123",
    "database_version": "3.40.0",
    "total_models": 150,
    "total_providers": 10,
    "total_verifications": 5000,
    "system_stats": {
      "cpu_usage": 15.2,
      "memory_usage": 45.8,
      "disk_usage": 23.1
    }
  }
}
```

#### Database Statistics
```http
GET /system/database-stats
```

**Response:**
```json
{
  "success": true,
  "data": {
    "total_size": "125.5 MB",
    "table_stats": [
      {
        "table": "models",
        "rows": 150,
        "size": "2.1 MB"
      },
      {
        "table": "verification_results",
        "rows": 5000,
        "size": "45.2 MB"
      },
      {
        "table": "events",
        "rows": 15000,
        "size": "78.2 MB"
      }
    ],
    "index_stats": [
      {
        "index": "idx_models_overall_score",
        "size": "0.8 MB"
      }
    ]
  }
}
```

## WebSocket API

### Connection

Connect to the WebSocket endpoint for real-time event streaming:

```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/events/stream');

ws.onopen = function(event) {
  console.log('Connected to event stream');
};

ws.onmessage = function(event) {
  const data = JSON.parse(event.data);
  console.log('Received event:', data);
};

ws.onerror = function(error) {
  console.error('WebSocket error:', error);
};
```

### Event Subscription

Subscribe to specific events by sending a subscription message:

```json
{
  "action": "subscribe",
  "event_types": ["verification_completed", "issue_detected"],
  "model_ids": [1, 2, 3]
}
```

### Event Format

Events are sent as JSON messages:

```json
{
  "event_type": "verification_completed",
  "severity": "info",
  "title": "Verification Completed",
  "message": "Model GPT-4 verification completed with score 92.5",
  "data": {
    "model_id": 1,
    "verification_result_id": 123,
    "score": 92.5
  },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

## Error Codes

The API uses standard HTTP status codes and provides detailed error information:

### HTTP Status Codes

- **200 OK**: Successful request
- **201 Created**: Resource created successfully
- **400 Bad Request**: Invalid request parameters
- **401 Unauthorized**: Authentication required or invalid
- **403 Forbidden**: Insufficient permissions
- **404 Not Found**: Resource not found
- **409 Conflict**: Resource conflict
- **422 Unprocessable Entity**: Validation error
- **429 Too Many Requests**: Rate limit exceeded
- **500 Internal Server Error**: Server error

### Error Codes

- `VALIDATION_ERROR`: Request validation failed
- `AUTHENTICATION_ERROR`: Authentication failed
- `AUTHORIZATION_ERROR`: Insufficient permissions
- `NOT_FOUND`: Resource not found
- `CONFLICT`: Resource conflict
- `RATE_LIMIT_EXCEEDED`: Rate limit exceeded
- `DATABASE_ERROR`: Database operation failed
- `EXTERNAL_API_ERROR`: External API call failed
- `INTERNAL_ERROR`: Internal server error

## SDK and Client Libraries

### Go Client

```go
package main

import (
    "fmt"
    "github.com/your-org/llm-verifier-go-client"
)

func main() {
    client := llmverifier.NewClient("http://localhost:8080/api/v1")
    client.SetAuthToken("your-jwt-token")
    
    // List models
    models, err := client.ListModels(nil)
    if err != nil {
        panic(err)
    }
    
    for _, model := range models {
        fmt.Printf("Model: %s, Score: %.1f\n", model.Name, model.OverallScore)
    }
}
```

### Python Client

```python
from llm_verifier_client import LLMVerifierClient

client = LLMVerifierClient("http://localhost:8080/api/v1")
client.set_auth_token("your-jwt-token")

# List models
models = client.list_models()
for model in models:
    print(f"Model: {model['name']}, Score: {model['overall_score']}")

# Get specific model
model = client.get_model(1)
print(f"Model details: {model}")
```

### JavaScript/Node.js Client

```javascript
const { LLMVerifierClient } = require('llm-verifier-client');

const client = new LLMVerifierClient('http://localhost:8080/api/v1');
client.setAuthToken('your-jwt-token');

// List models
const models = await client.listModels();
models.forEach(model => {
    console.log(`Model: ${model.name}, Score: ${model.overall_score}`);
});

// Get specific model
const model = await client.getModel(1);
console.log('Model details:', model);
```

## Pagination

List endpoints support pagination using the following parameters:

- `page` (int): Page number (default: 1)
- `per_page` (int): Items per page (default: 20, max: 100)

Pagination information is included in the response meta:

```json
{
  "success": true,
  "data": [...],
  "meta": {
    "total": 150,
    "page": 2,
    "per_page": 20,
    "total_pages": 8,
    "has_next": true,
    "has_prev": true
  }
}
```

## Filtering and Search

Most list endpoints support filtering and search:

### Search
Use the `search` parameter to search across text fields:
```http
GET /models?search=gpt
```

### Filtering
Use specific field parameters to filter results:
```http
GET /models?min_score=80&supports_tool_use=true&verification_status=verified
```

### Sorting
Use the `sort` and `order` parameters:
```http
GET /models?sort=overall_score&order=desc
```

## Versioning

The API uses URL versioning. The current version is v1:
```
/api/v1/...
```

When breaking changes are introduced, a new version will be released (e.g., `/api/v2/`). The previous version will be maintained for a deprecation period.

## Changelog

### Version 1.0.0 (Current)
- Initial API release
- Full CRUD operations for models, providers, and verification results
- Event streaming with WebSocket support
- Configuration export functionality
- Scheduling system
- Notification system integration
- Comprehensive filtering and search
- JWT authentication
- Rate limiting

This API documentation provides comprehensive information for integrating with the LLM Verifier REST API. For additional support or to report issues, please visit our GitHub repository.