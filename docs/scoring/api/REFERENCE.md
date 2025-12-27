# 🚀 LLM Verifier Scoring System - API Reference

## 📋 Overview

This document provides comprehensive API reference for the LLM Verifier Scoring System. All endpoints follow RESTful conventions and return JSON responses.

## 🔗 Base URL

```
Development: http://localhost:8080/api/v1
Production: https://api.llm-verifier.com/api/v1
```

## 🔐 Authentication

Most endpoints require authentication via Bearer token:

```http
Authorization: Bearer <your-api-token>
```

## 📊 Endpoints

### 1. Score Calculation

#### Calculate Model Score
```http
POST /models/{model_id}/score/calculate
```

Calculate a comprehensive score for a specific model.

**Parameters:**
- `model_id` (path, required): The unique identifier of the model

**Request Body:**
```json
{
  "configuration": {
    "weights": {
      "response_speed": 0.25,
      "model_efficiency": 0.20,
      "cost_effectiveness": 0.25,
      "capability": 0.20,
      "recency": 0.10
    }
  },
  "force_recalculation": false
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "model_id": "gpt-4",
    "model_name": "GPT-4",
    "overall_score": 8.5,
    "score_suffix": "(SC:8.5)",
    "components": {
      "speed_score": 8.0,
      "efficiency_score": 9.0,
      "cost_score": 7.5,
      "capability_score": 9.0,
      "recency_score": 8.5
    },
    "last_calculated": "2025-12-27T18:16:00Z",
    "data_source": "models.dev",
    "calculation_hash": "abc123def456",
    "valid_until": "2025-12-28T18:16:00Z"
  },
  "metadata": {
    "request_id": "req_123456",
    "processing_time_ms": 145,
    "timestamp": "2025-12-27T18:16:00Z"
  }
}
```

**Error Responses:**
- `400 Bad Request`: Invalid request parameters
- `404 Not Found`: Model not found
- `500 Internal Server Error`: Server error

#### Get Current Model Score
```http
GET /models/{model_id}/score
```

Retrieve the current score for a model.

**Parameters:**
- `model_id` (path, required): The unique identifier of the model

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "model_id": "gpt-4",
    "model_name": "GPT-4",
    "overall_score": 8.5,
    "score_suffix": "(SC:8.5)",
    "components": {
      "speed_score": 8.0,
      "efficiency_score": 9.0,
      "cost_score": 7.5,
      "capability_score": 9.0,
      "recency_score": 8.5
    },
    "last_calculated": "2025-12-27T18:16:00Z",
    "data_source": "models.dev"
  },
  "metadata": {
    "request_id": "req_123457",
    "timestamp": "2025-12-27T18:16:00Z"
  }
}
```

#### Force Score Recalculation
```http
PUT /models/{model_id}/score/recalculate
```

Force recalculation of an existing score.

**Parameters:**
- `model_id` (path, required): The unique identifier of the model

**Request Body:**
```json
{
  "reason": "Model capabilities updated",
  "configuration": {
    "weights": {...}
  }
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "message": "Score recalculated successfully",
    "model_id": "gpt-4",
    "overall_score": 8.7,
    "score_suffix": "(SC:8.7)",
    "components": {...},
    "last_calculated": "2025-12-27T18:20:00Z",
    "recalc_reason": "Model capabilities updated"
  },
  "metadata": {
    "request_id": "req_123458",
    "timestamp": "2025-12-27T18:20:00Z"
  }
}
```

### 2. Batch Operations

#### Batch Calculate Scores
```http
POST /models/scores/batch
```

Calculate scores for multiple models in a single request.

**Request Body:**
```json
{
  "model_ids": ["gpt-4", "claude-3-sonnet", "llama-2-70b"],
  "configuration": {
    "weights": {
      "response_speed": 0.25,
      "model_efficiency": 0.20,
      "cost_effectiveness": 0.25,
      "capability": 0.20,
      "recency": 0.10
    }
  },
  "async": false
}
```

**Response (200 OK) - Synchronous:**
```json
{
  "success": true,
  "data": {
    "message": "Batch score calculation completed",
    "batch_id": "batch_1234567890",
    "status": "completed",
    "results": [
      {
        "model_id": "gpt-4",
        "model_name": "GPT-4",
        "overall_score": 8.5,
        "score_suffix": "(SC:8.5)",
        "components": {...},
        "success": true
      },
      {
        "model_id": "claude-3-sonnet",
        "model_name": "Claude 3 Sonnet",
        "overall_score": 7.8,
        "score_suffix": "(SC:7.8)",
        "components": {...},
        "success": true
      }
    ],
    "model_count": 3,
    "success_count": 3,
    "failure_count": 0
  },
  "metadata": {
    "request_id": "req_123459",
    "processing_time_ms": 892,
    "timestamp": "2025-12-27T18:16:00Z"
  }
}
```

**Response (202 Accepted) - Asynchronous:**
```json
{
  "success": true,
  "data": {
    "message": "Batch score calculation started",
    "batch_id": "batch_1234567890",
    "status": "processing",
    "model_count": 3
  },
  "metadata": {
    "request_id": "req_123460",
    "timestamp": "2025-12-27T18:16:00Z"
  }
}
```

### 3. Model Naming

#### Add Score Suffix to Model Name
```http
POST /models/naming/add-suffix
```

Add a score suffix to a model name.

**Request Body:**
```json
{
  "model_name": "GPT-4",
  "score": 8.5
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "original_name": "GPT-4",
    "updated_name": "GPT-4 (SC:8.5)",
    "score": 8.5,
    "score_suffix": "(SC:8.5)"
  },
  "metadata": {
    "request_id": "req_123461",
    "timestamp": "2025-12-27T18:16:00Z"
  }
}
```

#### Batch Update Model Names with Scores
```http
POST /models/naming/batch-update
```

Update multiple model names with their scores.

**Request Body:**
```json
{
  "model_scores": {
    "GPT-4": 8.5,
    "Claude-3": 7.8,
    "Llama-2": 6.9
  }
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "message": "Model names updated successfully",
    "results": {
      "GPT-4": "GPT-4 (SC:8.5)",
      "Claude-3": "Claude-3 (SC:7.8)",
      "Llama-2": "Llama-2 (SC:6.9)"
    },
    "count": 3
  },
  "metadata": {
    "request_id": "req_123462",
    "timestamp": "2025-12-27T18:16:00Z"
  }
}
```

### 4. Score Comparison

#### Compare Multiple Models
```http
GET /models/scores/compare?models=gpt-4,claude-3,llama-2
```

Compare scores between multiple models.

**Query Parameters:**
- `models` (query, required): Comma-separated list of model IDs

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "comparison": {
      "models": ["gpt-4", "claude-3", "llama-2"],
      "results": [
        {
          "model_id": "gpt-4",
          "model_name": "GPT-4",
          "overall_score": 8.5,
          "rank": 1
        },
        {
          "model_id": "claude-3",
          "model_name": "Claude 3",
          "overall_score": 7.8,
          "rank": 2
        },
        {
          "model_id": "llama-2",
          "model_name": "Llama 2",
          "overall_score": 6.9,
          "rank": 3
        }
      ],
      "best_model": "gpt-4",
      "analysis": "GPT-4 has the highest overall score with strong performance across all components"
    }
  },
  "metadata": {
    "request_id": "req_123463",
    "timestamp": "2025-12-27T18:16:00Z"
  }
}
```

### 5. Score Rankings

#### Get Model Rankings
```http
GET /models/scores/ranking?category=overall&limit=10&min_score=7.0&max_score=10.0
```

Get ranked list of models by score.

**Query Parameters:**
- `category` (query, optional): Scoring category (overall, speed, efficiency, cost, capability, recency)
- `limit` (query, optional): Maximum number of results (default: 50)
- `min_score` (query, optional): Minimum score filter
- `max_score` (query, optional): Maximum score filter

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "category": "overall",
    "rankings": [
      {
        "rank": 1,
        "model_id": "gpt-4",
        "model_name": "GPT-4 (SC:8.5)",
        "overall_score": 8.5,
        "score_suffix": "(SC:8.5)",
        "category_score": 8.5,
        "last_updated": "2025-12-27T18:16:00Z"
      },
      {
        "rank": 2,
        "model_id": "claude-3",
        "model_name": "Claude 3 (SC:7.8)",
        "overall_score": 7.8,
        "score_suffix": "(SC:7.8)",
        "category_score": 7.8,
        "last_updated": "2025-12-27T18:16:00Z"
      }
    ]
  },
  "metadata": {
    "request_id": "req_123464",
    "timestamp": "2025-12-27T18:16:00Z",
    "total_models": 25,
    "filtered_models": 10
  }
}
```

### 6. Configuration Management

#### Get Scoring Configuration
```http
GET /scoring/configuration?config=default
```

Get scoring configuration details.

**Query Parameters:**
- `config` (query, optional): Configuration name (default: "default")

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "config_name": "default",
    "weights": {
      "response_speed": 0.25,
      "model_efficiency": 0.20,
      "cost_effectiveness": 0.25,
      "capability": 0.20,
      "recency": 0.10
    },
    "thresholds": {
      "min_score": 0.0,
      "max_score": 10.0
    },
    "enabled": true,
    "last_updated": "2025-12-27T18:16:00Z"
  },
  "metadata": {
    "request_id": "req_123465",
    "timestamp": "2025-12-27T18:16:00Z"
  }
}
```

### 7. Score Validation

#### Validate Score Calculation
```http
POST /scoring/validate
```

Validate a score calculation result.

**Request Body:**
```json
{
  "model_id": "gpt-4",
  "score": 8.5,
  "method": "comprehensive_scoring",
  "expected_range": {
    "min": 7.0,
    "max": 10.0
  }
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "model_id": "gpt-4",
    "score": 8.5,
    "method": "comprehensive_scoring",
    "is_valid": true,
    "validation_result": {
      "within_expected_range": true,
      "component_scores_valid": true,
      "calculation_hash_verified": true
    },
    "message": "Score validation completed successfully"
  },
  "metadata": {
    "request_id": "req_123466",
    "timestamp": "2025-12-27T18:16:00Z"
  }
}
```

## 🔄 WebSocket Support

### Real-time Score Updates
```javascript
// Connect to WebSocket
const ws = new WebSocket('ws://localhost:8080/api/v1/ws/scoring');

// Subscribe to model score updates
ws.send(JSON.stringify({
    action: 'subscribe',
    model_ids: ['gpt-4', 'claude-3']
}));

// Listen for updates
ws.onmessage = function(event) {
    const update = JSON.parse(event.data);
    console.log('Score update:', update);
};
```

**WebSocket Message Format:**
```json
{
  "type": "score_update",
  "data": {
    "model_id": "gpt-4",
    "overall_score": 8.6,
    "score_suffix": "(SC:8.6)",
    "components": {...},
    "last_calculated": "2025-12-27T18:16:00Z"
  },
  "timestamp": "2025-12-27T18:16:00Z"
}
```

## 📊 Rate Limiting

API endpoints are rate-limited to ensure fair usage:

| Endpoint | Rate Limit | Window |
|----------|------------|---------|
| Score Calculation | 100 requests/minute | 60 seconds |
| Batch Operations | 10 requests/minute | 60 seconds |
| Model Naming | 200 requests/minute | 60 seconds |
| Score Comparison | 50 requests/minute | 60 seconds |

**Rate Limit Headers:**
```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640620800
```

## 🚨 Error Handling

### Standard Error Response
```json
{
  "success": false,
  "error": {
    "code": "MODEL_NOT_FOUND",
    "message": "Model with ID 'invalid-model' not found",
    "details": {
      "model_id": "invalid-model",
      "suggestion": "Please check the model ID and try again"
    }
  },
  "metadata": {
    "request_id": "req_123467",
    "timestamp": "2025-12-27T18:16:00Z"
  }
}
```

### Error Codes

| Code | Description | HTTP Status |
|------|-------------|-------------|
| `MODEL_NOT_FOUND` | Model not found in database | 404 |
| `INVALID_REQUEST` | Invalid request parameters | 400 |
| `SCORING_FAILED` | Score calculation failed | 500 |
| `API_TIMEOUT` | Models.dev API timeout | 504 |
| `DATABASE_ERROR` | Database operation failed | 500 |
| `RATE_LIMIT_EXCEEDED` | Rate limit exceeded | 429 |
| `AUTHENTICATION_FAILED` | Authentication failed | 401 |
| `AUTHORIZATION_FAILED` | Insufficient permissions | 403 |

## 📈 Performance Metrics

### Response Times

| Endpoint | Average Response Time | 95th Percentile |
|----------|----------------------|-----------------|
| Single Score Calculation | 150ms | 300ms |
| Batch Score Calculation (10 models) | 800ms | 1500ms |
| Model Name Update | 50ms | 100ms |
| Score Comparison | 200ms | 400ms |

### Throughput

- **Single Model**: 1000+ calculations/second
- **Batch Operations**: 100+ models/second
- **Concurrent Requests**: 1000+ simultaneous

## 🔍 Testing

### API Testing with curl

```bash
# Test score calculation
curl -X POST http://localhost:8080/api/v1/models/gpt-4/score/calculate \
  -H "Content-Type: application/json" \
  -d '{"configuration": {"weights": {"response_speed": 0.25, "model_efficiency": 0.20, "cost_effectiveness": 0.25, "capability": 0.20, "recency": 0.10}}}'

# Test batch calculation
curl -X POST http://localhost:8080/api/v1/models/scores/batch \
  -H "Content-Type: application/json" \
  -d '{"model_ids": ["gpt-4", "claude-3"], "async": false}'

# Test model naming
curl -X POST http://localhost:8080/api/v1/models/naming/add-suffix \
  -H "Content-Type: application/json" \
  -d '{"model_name": "GPT-4", "score": 8.5}'
```

### Load Testing

```bash
# Install hey (HTTP load generator)
go install github.com/rakyll/hey@latest

# Load test score calculation
hey -n 1000 -c 50 -m POST \
  -H "Content-Type: application/json" \
  -d '{"configuration": {"weights": {"response_speed": 0.25, "model_efficiency": 0.20, "cost_effectiveness": 0.25, "capability": 0.20, "recency": 0.10}}}' \
  http://localhost:8080/api/v1/models/gpt-4/score/calculate
```

## 🔧 SDK Examples

### Go SDK

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/your-org/llm-verifier-sdk-go"
)

func main() {
    // Initialize client
    client := llmverifier.NewClient("https://api.llm-verifier.com", "your-api-token")
    
    // Calculate score
    score, err := client.CalculateScore(context.Background(), "gpt-4", nil)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Score: %.1f %s\n", score.OverallScore, score.ScoreSuffix)
}
```

### Python SDK

```python
from llm_verifier_sdk import LLMVerifierClient

# Initialize client
client = LLMVerifierClient(
    base_url="https://api.llm-verifier.com",
    api_token="your-api-token"
)

# Calculate score
score = client.calculate_score("gpt-4")
print(f"Score: {score.overall_score} {score.score_suffix}")
```

### JavaScript SDK

```javascript
import { LLMVerifierClient } from 'llm-verifier-sdk';

// Initialize client
const client = new LLMVerifierClient({
    baseURL: 'https://api.llm-verifier.com',
    apiToken: 'your-api-token'
});

// Calculate score
const score = await client.calculateScore('gpt-4');
console.log(`Score: ${score.overallScore} ${score.scoreSuffix}`);
```

---

## 📚 Additional API Resources

- [API Examples](./EXAMPLES.md)
- [Error Handling](./ERROR_HANDLING.md)
- [Rate Limiting](./RATE_LIMITING.md)
- [WebSocket Guide](./WEBSOCKET.md)
- [SDK Documentation](./SDK.md)

---

*API Documentation Version: 1.0.0*  
*Last Updated: 2025-12-27*  
*Status: ✅ PRODUCTION READY*