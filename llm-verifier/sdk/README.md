# LLM Verifier Client SDKs

This directory contains client SDKs for interacting with the LLM Verifier REST API from different programming languages.

## Available SDKs

### 1. Go SDK (`go/client.go`)

A comprehensive Go client for the LLM Verifier API with full type safety and error handling.

#### Installation

```bash
go get github.com/your-org/llm-verifier/sdk/go
```

#### Usage

```go
package main

import (
    "fmt"
    "log"

    client "github.com/your-org/llm-verifier/sdk/go"
)

func main() {
    // Create client
    c := client.NewLLMVerifierClient("http://localhost:8080", "")

    // Login
    auth, err := c.Login("admin", "password")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Logged in as: %s\n", auth.User.Username)

    // Get models
    models, err := c.GetModels(10, 0, "")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Found %d models\n", len(models))

    // Verify a model
    result, err := c.VerifyModel("gpt-4")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Verification score: %.2f\n", result.Score)
}
```

### 2. Python SDK (`python/client.py`)

A Python client with async support and comprehensive error handling.

#### Installation

```bash
pip install llm-verifier-sdk
```

#### Usage

```python
from llm_verifier_sdk import LLMVerifierClient

# Create client
client = LLMVerifierClient("http://localhost:8080")

# Login
auth = client.login("admin", "password")
print(f"Logged in as: {auth['user']['username']}")

# Get models
models = client.get_models(limit=10)
print(f"Found {len(models)} models")

# Get health status
health = client.get_health()
print(f"System status: {health['status']}")

# Verify a model
result = client.verify_model("gpt-4")
print(f"Verification score: {result['score']}")
```

### 3. JavaScript/TypeScript SDK (`javascript/client.ts`)

A modern TypeScript client with full type definitions and async/await support.

#### Installation

```bash
npm install llm-verifier-sdk
```

#### Usage

```typescript
import { LLMVerifierClient } from 'llm-verifier-sdk';

// Create client
const client = new LLMVerifierClient('http://localhost:8080');

// Login
const auth = await client.login('admin', 'password');
console.log(`Logged in as: ${auth.user.username}`);

// Get models
const models = await client.getModels({ limit: 10 });
console.log(`Found ${models.length} models`);

// Get health status
const health = await client.getHealth();
console.log(`System status: ${health.status}`);

// Verify a model
const result = await client.verifyModel('gpt-4');
console.log(`Verification score: ${result.score}`);
```

## API Endpoints Covered

All SDKs provide methods for the following API endpoints:

### Authentication
- `POST /auth/login` - User login
- `POST /auth/refresh` - Token refresh

### Models
- `GET /api/v1/models` - List models with filtering
- `GET /api/v1/models/{id}` - Get specific model
- `POST /api/v1/models` - Create model (admin)
- `PUT /api/v1/models/{id}` - Update model (admin)
- `DELETE /api/v1/models/{id}` - Delete model (admin)
- `POST /api/v1/models/{id}/verify` - Trigger verification

### Verification Results
- `GET /api/v1/verification-results` - List verification results
- `GET /api/v1/verification-results/{id}` - Get specific result

### Providers
- `GET /api/v1/providers` - List providers
- `GET /api/v1/providers/{id}` - Get specific provider

### System
- `GET /health` - Health check
- `GET /api/v1/system/info` - System information

## Authentication

All SDKs support JWT token authentication:

1. Call `login()` method to authenticate and receive a token
2. The token is automatically stored and included in subsequent requests
3. Use the token parameter in the constructor for pre-authenticated clients

## Error Handling

All SDKs include comprehensive error handling:

- **Go**: Returns `error` types with descriptive messages
- **Python**: Raises `Exception` with HTTP status and message
- **JavaScript**: Throws `Error` objects with detailed information

## Response Types

### Go SDK
- Uses strongly typed structs for all responses
- Full type safety and IntelliSense support

### Python SDK
- Returns dictionaries for maximum flexibility
- Easy to work with JSON-like data structures

### JavaScript/TypeScript SDK
- Full TypeScript type definitions
- IntelliSense support and compile-time type checking

## Development

### Building the SDKs

```bash
# Go SDK
cd sdk/go
go build ./...

# Python SDK (requires testing)
cd sdk/python
python -m pytest

# JavaScript SDK
cd sdk/javascript
npm run build
```

### Testing

Each SDK includes comprehensive tests and example usage. Run the tests to ensure compatibility with your LLM Verifier instance.

## Contributing

When contributing to the SDKs:

1. Maintain API compatibility across all languages
2. Include comprehensive error handling
3. Add examples and documentation
4. Ensure all SDKs implement the same functionality
5. Test against the actual API endpoints

## License

This project is licensed under the same license as the main LLM Verifier project.