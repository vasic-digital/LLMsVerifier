# LLM Verifier Python SDK

Official Python SDK for the LLM Verifier platform. Provides easy access to model verification, scoring, and management capabilities.

## Installation

```bash
pip install llm-verifier-sdk
```

Or install from source:

```bash
git clone https://github.com/llm-verifier/llm-verifier.git
cd llm-verifier/sdk/python
pip install -e .
```

## Quick Start

```python
from llm_verifier_sdk import LLMVerifierClient

# Initialize client
client = LLMVerifierClient(
    base_url="http://localhost:8080",
    api_key="your-api-key"
)

# Get all verified models
models = client.get_models()
for model in models:
    print(f"{model.name} (SC:{model.score})")

# Verify a specific model
result = client.verify_model("gpt-4o", prompt="Hello, can you see this code?")
print(f"Verification: {'PASSED' if result.success else 'FAILED'}")
print(f"Score: {result.score}")
```

## Features

### Model Discovery

```python
# Get all models
models = client.get_models()

# Get models by provider
openai_models = client.get_models_by_provider("openai")

# Get models by score range
top_models = client.get_models_by_score(min_score=8.0, limit=10)

# Get a specific model
model = client.get_model("gpt-4o")
print(f"Capabilities: {model.capabilities}")
```

### Model Verification

```python
# Verify a single model
result = client.verify_model(
    model_id="gpt-4o",
    prompt="Can you analyze this code snippet?"
)

# Batch verification
results = client.batch_verify([
    {"model_id": "gpt-4o", "prompt": "Test prompt 1"},
    {"model_id": "claude-3-opus", "prompt": "Test prompt 2"}
])

# Check verification status
status = client.get_verification_status(verification_id)
print(f"Status: {status.state}")
```

### Scoring

```python
# Get model score breakdown
score = client.get_model_score("gpt-4o")
print(f"Overall: {score.overall}")
print(f"Speed: {score.speed}")
print(f"Capability: {score.capability}")
print(f"Cost: {score.cost}")
```

### Configuration Export

```python
# Export verified configuration
config = client.export_config(
    format="opencode",
    providers=["openai", "anthropic"]
)

# Save to file
with open("opencode-config.json", "w") as f:
    f.write(config)
```

### Provider Management

```python
# List providers
providers = client.get_providers()

# Check provider health
health = client.check_provider_health("openai")
print(f"Status: {health.status}")

# Discover models from provider
discovered = client.discover_models("anthropic")
```

## Configuration

### Environment Variables

```bash
export LLM_VERIFIER_API_KEY="your-api-key"
export LLM_VERIFIER_BASE_URL="http://localhost:8080"
```

### Client Options

```python
client = LLMVerifierClient(
    base_url="http://localhost:8080",
    api_key="your-api-key",
    timeout=30,           # Request timeout in seconds
    max_retries=3,        # Maximum retry attempts
    verify_ssl=True       # SSL certificate verification
)
```

## Error Handling

```python
from llm_verifier_sdk import (
    LLMVerifierClient,
    AuthenticationError,
    RateLimitError,
    ProviderError,
    ValidationError
)

try:
    result = client.verify_model("gpt-4o")
except AuthenticationError:
    print("Invalid API key")
except RateLimitError as e:
    print(f"Rate limited. Retry after {e.retry_after} seconds")
except ProviderError as e:
    print(f"Provider error: {e.message}")
except ValidationError as e:
    print(f"Validation error: {e.details}")
```

## Async Support

```python
import asyncio
from llm_verifier_sdk import AsyncLLMVerifierClient

async def main():
    client = AsyncLLMVerifierClient(api_key="your-api-key")

    # Async operations
    models = await client.get_models()
    result = await client.verify_model("gpt-4o")

    await client.close()

asyncio.run(main())
```

## Data Classes

### Model

```python
@dataclass
class Model:
    id: str
    name: str
    provider: str
    verified: bool
    score: float
    features: Dict[str, Any]
    capabilities: List[str]
    metadata: Dict[str, Any]
```

### VerificationResult

```python
@dataclass
class VerificationResult:
    model_id: str
    success: bool
    score: float
    capabilities: List[str]
    timestamp: datetime
    details: Dict[str, Any]
```

## Requirements

- Python 3.8+
- requests
- dataclasses (Python 3.7)

## License

MIT License - see LICENSE file for details.

## Links

- [Documentation](https://llm-verifier.dev/docs)
- [API Reference](https://llm-verifier.dev/docs/api-reference)
- [GitHub](https://github.com/llm-verifier/llm-verifier)
