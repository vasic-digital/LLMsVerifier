# LLM Verifier REST API Documentation - v2.0 Updated

## Overview

The LLMsVerifier v2.0 REST API provides programmatic access to all functionality including the new **mandatory model verification system** and **LLMSVD suffix branding**. Built using the GinGonic framework, it offers comprehensive endpoints for managing verified models, providers, verification results, and system configuration.

### New in v2.0
- **Model Verification Endpoints**: Complete verification workflow management
- **Verified Model Filtering**: Query only models that pass verification
- **LLMSVD Suffix Integration**: All responses include mandatory branding
- **Enhanced Configuration Export**: Export only verified configurations
- **Advanced Analytics**: Verification metrics and statistics

## 🔑 Authentication

### JWT Token Authentication
All API requests require JWT authentication (except health check and login).

#### Login
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "password"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600,
    "user": {
      "id": 1,
      "username": "admin",
      "role": "admin"
    }
  }
}
```

#### Using Authentication
Include the JWT token in the Authorization header:
```http
GET /api/v1/models
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

## 🆕 New v2.0 Endpoints

### Model Verification

#### Trigger Model Verification
```http
POST /api/v1/models/{model_id}/verify
Authorization: Bearer {token}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "verification_id": 123,
    "status": "started",
    "message": "Verification started for model GPT-4 (llmsvd)",
    "estimated_completion": "2025-12-28T15:30:00Z"
  }
}
```

#### Get Verification Status
```http
GET /api/v1/models/{model_id}/verification-status
Authorization: Bearer {token}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "model_id": 1,
    "model_name": "GPT-4 (llmsvd)",
    "verification_status": "verified",
    "verification_score": 0.85,
    "can_see_code": true,
    "affirmative_response": true,
    "last_verified": "2025-12-28T14:30:00Z",
    "next_verification_due": "2025-12-29T14:30:00Z",
    "verification_history": [
      {
        "verification_id": 123,
        "score": 0.85,
        "status": "passed",
        "timestamp": "2025-12-28T14:30:00Z"
      }
    ]
  }
}
```

#### Get Detailed Verification Results
```http
GET /api/v1/models/{model_id}/verification-results
Authorization: Bearer {token}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "results": [
      {
        "id": 123,
        "model_id": 1,
        "verification_type": "code_visibility",
        "started_at": "2025-12-28T14:25:00Z",
        "completed_at": "2025-12-28T14:30:00Z",
        "status": "completed",
        "score": 0.85,
        "can_see_code": true,
        "affirmative_response": true,
        "response_text": "Yes, I can see your Python code. It's a function that calculates fibonacci numbers.",
        "verification_prompt": "Do you see my code? Please respond with 'Yes, I can see your Python code' if you can see the code below...",
        "test_code": "def fibonacci(n): if n <= 1: return n; return fibonacci(n-1) + fibonacci(n-2)",
        "retry_count": 0,
        "error_message": null
      }
    ],
    "statistics": {
      "total_verifications": 5,
      "passed": 4,
      "failed": 1,
      "average_score": 0.82,
      "last_verification": "2025-12-28T14:30:00Z"
    }
  }
}
```

### Verified Model Management

#### List Verified Models Only
```http
GET /api/v1/models?verification_status=verified
Authorization: Bearer {token}
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "provider_id": 1,
      "model_id": "gpt-4",
      "name": "GPT-4 (brotli) (http3) (llmsvd)",  # Note: (llmsvd) suffix
      "description": "OpenAI's most capable model with verification",
      "verification_status": "verified",
      "verification_score": 0.85,
      "can_see_code": true,
      "overall_score": 92.5,
      "features": {
        "supports_tool_use": true,
        "supports_code_generation": true,
        "supports_brotli": true,
        "supports_http3": true
      },
      "last_verified": "2025-12-28T14:30:00Z"
    }
  ],
  "meta": {
    "total": 25,
    "verified": 23,
    "verification_rate": 92.0
  }
}
```

#### Filter by Verification Score
```http
GET /api/v1/models?min_verification_score=0.8&verification_status=verified
Authorization: Bearer {token}
```

#### Bulk Verification Operations
```http
POST /api/v1/models/verify-bulk
Authorization: Bearer {token}
Content-Type: application/json

{
  "model_ids": [1, 2, 3, 4, 5],
  "verification_config": {
    "strict_mode": true,
    "timeout_seconds": 30,
    "concurrent_limit": 5
  }
}
```

### Enhanced Configuration Export

#### Export Verified OpenCode Configuration
```http
POST /api/v1/config-exports/opencode
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "Verified Models Config",
  "description": "OpenCode configuration with only verified models",
  "filters": {
    "verification_status": "verified",
    "min_score": 80,
    "supports_code_generation": true,
    "providers": ["openai", "anthropic"]
  },
  "include_api_keys": false,
  "format_options": {
    "include_costs": true,
    "include_features": true
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "export_id": "export_123",
    "download_url": "/api/v1/config-exports/export_123/download",
    "config_preview": {
      "$schema": "https://opencodelabs.io/schema/model-config.json",
      "provider": {
        "OpenAI (llmsvd)": {
          "options": {
            "apiKey": ""
          },
          "models": {
            "GPT-4 (brotli) (http3) (llmsvd)": {
              "model": "gpt-4",
              "cost": { "input": 0.03, "output": 0.06 }
            },
            "GPT-4 Turbo (brotli) (http3) (llmsvd)": {
              "model": "gpt-4-turbo",
              "cost": { "input": 0.01, "output": 0.03 }
            }
          }
        }
      }
    },
    "statistics": {
      "total_models": 15,
      "verified_models": 15,
      "verification_rate": 100.0,
      "providers_included": ["openai", "anthropic"]
    }
  }
}
```

#### Export Verified Crush Configuration
```http
POST /api/v1/config-exports/crush
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "Verified Crush Config",
  "description": "Crush configuration with verified models only",
  "filters": {
    "verification_status": "verified",
    "min_verification_score": 0.7,
    "supports_streaming": true
  },
  "format_options": {
    "include_costs": true,
    "include_capabilities": true
  }
}
```

### Verification Analytics

#### Get Verification Statistics
```http
GET /api/v1/analytics/verification
Authorization: Bearer {token}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "overview": {
      "total_models": 150,
      "verified_models": 135,
      "verification_rate": 90.0,
      "average_verification_score": 0.82,
      "verification_failures": 15
    },
    "by_provider": {
      "openai": {
        "total": 25,
        "verified": 23,
        "rate": 92.0,
        "average_score": 0.85
      },
      "anthropic": {
        "total": 20,
        "verified": 18,
        "rate": 90.0,
        "average_score": 0.83
      }
    },
    "trends": {
      "daily_verification_rate": [
        {"date": "2025-12-27", "rate": 88.5},
        {"date": "2025-12-28", "rate": 90.0}
      ],
      "score_distribution": {
        "0.9-1.0": 45,
        "0.8-0.9": 60,
        "0.7-0.8": 30,
        "<0.7": 15
      }
    }
  }
}
```

#### Get Verification Logs
```http
GET /api/v1/analytics/verification-logs?from_date=2025-12-27&to_date=2025-12-28
Authorization: Bearer {token}
```

## 📊 Enhanced Model Endpoints

### List Models with Verification Filters
```http
GET /api/v1/models?verification_status=verified&min_verification_score=0.8&supports_code_generation=true
Authorization: Bearer {token}
```

**Enhanced Response with Verification Data:**
```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "provider_id": 1,
      "provider_name": "OpenAI (llmsvd)",  # Note: (llmsvd) suffix
      "model_id": "gpt-4",
      "name": "GPT-4 (brotli) (http3) (llmsvd)",  # Note: (llmsvd) suffix
      "description": "OpenAI's most capable model with verification",
      "verification_status": "verified",
      "verification_score": 0.85,
      "can_see_code": true,
      "affirmative_response": true,
      "overall_score": 92.5,
      "code_capability_score": 95.0,
      "responsiveness_score": 88.0,
      "reliability_score": 94.0,
      "feature_richness_score": 91.0,
      "features": {
        "supports_tool_use": true,
        "supports_function_calling": true,
        "supports_code_generation": true,
        "supports_brotli": true,
        "supports_http3": true,
        "supports_streaming": true,
        "supports_json_mode": true
      },
      "last_verified": "2025-12-28T14:30:00Z",
      "verification_history": [
        {
          "date": "2025-12-28T14:30:00Z",
          "score": 0.85,
          "status": "passed"
        }
      ]
    }
  ],
  "meta": {
    "total": 25,
    "page": 1,
    "per_page": 20,
    "filters_applied": {
      "verification_status": "verified",
      "min_verification_score": 0.8,
      "supports_code_generation": true
    },
    "summary": {
      "verified_models": 23,
      "verification_rate": 92.0,
      "average_verification_score": 0.84
    }
  }
}
```

### Create Model with Verification
```http
POST /api/v1/models
Authorization: Bearer {token}
Content-Type: application/json

{
  "provider_id": 1,
  "model_id": "custom-model",
  "name": "Custom Model (llmsvd)",  # Must include (llmsvd) suffix
  "description": "A custom model with verification",
  "verification_config": {
    "auto_verify": true,
    "verification_priority": "high"
  },
  "features": {
    "supports_tool_use": true,
    "supports_code_generation": true,
    "supports_brotli": true
  }
}
```

## 🏢 Enhanced Provider Endpoints

### List Providers with Verification Data
```http
GET /api/v1/providers?include_verification_stats=true
Authorization: Bearer {token}
```

**Response with Verification Statistics:**
```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "name": "OpenAI (llmsvd)",  # Note: (llmsvd) suffix
      "endpoint": "https://api.openai.com/v1",
      "description": "OpenAI API with LLMsVerifier verification",
      "is_active": true,
      "has_llmsvd_suffix": true,
      "verification_statistics": {
        "total_models": 25,
        "verified_models": 23,
        "verification_rate": 92.0,
        "average_verification_score": 0.84,
        "last_verification_batch": "2025-12-28T14:30:00Z"
      },
      "models": [
        {
          "id": 1,
          "name": "GPT-4 (brotli) (http3) (llmsvd)",
          "verification_status": "verified",
          "verification_score": 0.85
        }
      ]
    }
  ],
  "meta": {
    "total": 10,
    "with_llmsvd_suffix": 10,
    "verification_summary": {
      "total_models": 250,
      "verified_models": 225,
      "overall_verification_rate": 90.0
    }
  }
}
```

## 📈 Analytics and Reporting

### Get Comprehensive Analytics
```http
GET /api/v1/analytics/comprehensive?from_date=2025-12-01&to_date=2025-12-28
Authorization: Bearer {token}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "system_overview": {
      "total_models": 150,
      "verified_models": 135,
      "total_providers": 10,
      "verification_rate": 90.0,
      "system_uptime": 99.9
    },
    "verification_analytics": {
      "total_verifications": 1250,
      "successful_verifications": 1125,
      "failed_verifications": 125,
      "average_verification_score": 0.82,
      "verification_trend": [
        {"date": "2025-12-20", "rate": 88.5},
        {"date": "2025-12-21", "rate": 89.2},
        {"date": "2025-12-22", "rate": 90.0}
      ]
    },
    "performance_metrics": {
      "average_api_response_time": 145,
      "verification_average_time": 3200,
      "database_query_average_time": 12,
      "memory_usage": {
        "current": 512,
        "peak": 768,
        "average": 456
      }
    },
    "branding_compliance": {
      "models_with_llmsvd_suffix": 150,
      "providers_with_llmsvd_suffix": 10,
      "branding_compliance_rate": 100.0
    }
  }
}
```

### Generate Verification Report
```http
POST /api/v1/reports/verification
Authorization: Bearer {token}
Content-Type: application/json

{
  "report_type": "comprehensive",
  "date_range": {
    "from": "2025-12-01",
    "to": "2025-12-28"
  },
  "filters": {
    "providers": ["openai", "anthropic"],
    "min_verification_score": 0.7
  },
  "format": "pdf",
  "include_recommendations": true
}
```

## 🔧 System Management

### Get System Configuration
```http
GET /api/v1/system/configuration
Authorization: Bearer {token}
```

**Response (v2.0 Configuration):**
```json
{
  "success": true,
  "data": {
    "version": "2.0.0",
    "configuration": {
      "model_verification": {
        "enabled": true,
        "strict_mode": true,
        "require_affirmative": true,
        "max_retries": 3,
        "timeout_seconds": 30,
        "min_verification_score": 0.7
      },
      "branding": {
        "enabled": true,
        "suffix": "(llmsvd)",
        "position": "final"
      },
      "api": {
        "port": 8080,
        "rate_limit": 1000,
        "enable_cors": true
      },
      "monitoring": {
        "enabled": true,
        "prometheus": {
          "enabled": true,
          "port": 9090
        }
      }
    }
  }
}
```

### Update System Configuration
```http
PUT /api/v1/system/configuration
Authorization: Bearer {token}
Content-Type: application/json

{
  "model_verification": {
    "enabled": true,
    "strict_mode": false,
    "min_verification_score": 0.6
  },
  "branding": {
    "enabled": true,
    "suffix": "(llmsvd)"
  }
}
```

## 🚨 Error Handling

### New Error Codes for v2.0
```json
{
  "VERIFICATION_FAILED": "Model failed verification requirements",
  "VERIFICATION_TIMEOUT": "Model verification timed out",
  "INSUFFICIENT_VERIFICATION_SCORE": "Model verification score below threshold",
  "BRANDING_REQUIRED": "LLMSVD suffix required but missing",
  "VERIFICATION_DISABLED": "Model verification is disabled",
  "VERIFICATION_IN_PROGRESS": "Verification already in progress for this model"
}
```

### Error Response Example
```json
{
  "success": false,
  "error": {
    "code": "VERIFICATION_FAILED",
    "message": "Model GPT-4 (llmsvd) failed verification - cannot see provided code",
    "details": {
      "model_id": 1,
      "verification_score": 0.45,
      "can_see_code": false,
      "affirmative_response": false,
      "suggestion": "Try increasing timeout or check provider status"
    }
  },
  "meta": {
    "timestamp": "2025-12-28T14:30:00Z",
    "request_id": "req_123"
  }
}
```

## 📚 SDK Examples

### Go SDK (v2.0)
```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/vasic-digital/LLMsVerifier/sdk/go/v2"
)

func main() {
    // Create v2 client
    client := llmverifier.NewClient("http://localhost:8080", "your-api-key")
    
    // Get only verified models
    verifiedModels, err := client.GetVerifiedModels(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d verified models:\n", len(verifiedModels))
    for _, model := range verifiedModels {
        fmt.Printf("- %s (Score: %.2f, Verified: %v)\n", 
            model.Name, model.VerificationScore, model.VerificationStatus)
    }
    
    // Trigger verification for specific model
    verification, err := client.VerifyModel(context.Background(), "gpt-4", "print('hello')")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Verification Result: Score=%.2f, CanSeeCode=%v\n", 
        verification.Score, verification.CanSeeCode)
    
    // Export verified configuration
    export, err := client.ExportVerifiedConfiguration(context.Background(), "opencode")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Exported %d verified models to OpenCode format\n", 
        len(export.Models))
}
```

### Python SDK (v2.0)
```python
from llm_verifier_client import LLMVerifierClient

client = LLMVerifierClient("http://localhost:8080", "your-api-key")

# Get only verified models
verified_models = client.get_verified_models()
print(f"Found {len(verified_models)} verified models:")
for model in verified_models:
    print(f"- {model['name']} (Score: {model['verification_score']:.2f})")

# Verify specific model
verification = client.verify_model("gpt-4", "print('hello')")
print(f"Verification: Score={verification['score']}, CanSeeCode={verification['can_see_code']}")

# Export verified configuration
export = client.export_verified_configuration(format="opencode")
print(f"Exported {len(export['models'])} verified models")
```

### JavaScript SDK (v2.0)
```javascript
const { LLMVerifierClient } = require('@llm-verifier/sdk');

const client = new LLMVerifierClient('http://localhost:8080', 'your-api-key');

async function main() {
    try {
        // Get verified models only
        const verifiedModels = await client.getVerifiedModels();
        console.log(`Found ${verifiedModels.length} verified models:`);
        verifiedModels.forEach(model => {
            console.log(`- ${model.name} (Score: ${model.verificationScore.toFixed(2)})`);
        });
        
        // Verify specific model
        const verification = await client.verifyModel('gpt-4', 'print("hello")');
        console.log(`Verification: Score=${verification.score}, CanSeeCode=${verification.canSeeCode}`);
        
        // Export verified configuration
        const export = await client.exportVerifiedConfiguration('opencode');
        console.log(`Exported ${export.models.length} verified models`);
        
    } catch (error) {
        console.error('Error:', error.message);
    }
}

main();
```

## 📊 Rate Limiting and Performance

### Enhanced Rate Limiting (v2.0)
```http
# Rate limit headers in response
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1640995200
X-RateLimit-Reset-After: 3600

# Verification-specific limits
X-Verification-RateLimit-Limit: 100
X-Verification-RateLimit-Remaining: 95
```

### Performance Optimization Tips
1. **Use Bulk Operations**: For multiple verifications, use bulk endpoints
2. **Implement Caching**: Cache verification results client-side
3. **Filter Efficiently**: Use query parameters to reduce response size
4. **Monitor Limits**: Watch rate limit headers to avoid throttling

## 🔍 WebSocket Support

### Real-time Verification Updates
```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/events/verification');

ws.onopen = function() {
    // Subscribe to verification events
    ws.send(JSON.stringify({
        action: 'subscribe',
        event_types: ['verification_started', 'verification_completed'],
        model_ids': [1, 2, 3]
    }));
};

ws.onmessage = function(event) {
    const data = JSON.parse(event.data);
    console.log('Verification event:', data);
};
```

## 🔄 API Versioning

### Version Headers
```http
# Current version (v2.0)
GET /api/v2/models
Accept: application/vnd.llm-verifier.v2+json

# Legacy version (v1.0) - Deprecated
GET /api/v1/models
Accept: application/vnd.llm-verifier.v1+json
```

### Deprecation Notices
- **v1.0 APIs**: Deprecated as of v2.0, will be removed in v3.0
- **Migration Timeline**: 6-month deprecation period
- **Breaking Changes**: v2.0 introduces breaking changes from v1.0

---

**The LLMsVerifier v2.0 API provides comprehensive access to the enhanced verification system and LLMSVD branding, ensuring secure, efficient, and scalable integration with your applications.**