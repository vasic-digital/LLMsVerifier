# LLM Verifier Java SDK

Official Java SDK for the LLM Verifier platform. Provides comprehensive access to model verification, scoring, and management capabilities.

## Requirements

- Java 11 or higher
- Jackson Databind
- HTTP Client (java.net.http)

## Installation

### Maven

```xml
<dependency>
    <groupId>com.llmverifier</groupId>
    <artifactId>llm-verifier-sdk</artifactId>
    <version>1.0.0</version>
</dependency>
```

### Gradle

```groovy
implementation 'com.llmverifier:llm-verifier-sdk:1.0.0'
```

## Quick Start

```java
import com.llmverifier.sdk.LLMVerifierClient;
import com.llmverifier.sdk.Model;
import com.llmverifier.sdk.VerificationResult;

public class Example {
    public static void main(String[] args) {
        // Initialize client
        LLMVerifierClient client = new LLMVerifierClient("your-api-key");

        // Get all verified models
        List<Model> models = client.getModels().join();
        for (Model model : models) {
            System.out.printf("%s (SC:%.1f)%n", model.getName(), model.getScore());
        }

        // Verify a model
        VerificationResult result = client.verifyModel("gpt-4o", "Hello, can you see this code?").join();
        System.out.printf("Verification: %s%n", result.isSuccess() ? "PASSED" : "FAILED");
        System.out.printf("Score: %.1f%n", result.getScore());
    }
}
```

## Features

### Model Discovery

```java
// Get all models
CompletableFuture<List<Model>> modelsFuture = client.getModels();
List<Model> models = modelsFuture.join();

// Get models by provider
List<Model> openaiModels = client.getModelsByProvider("openai").join();

// Get models by score range
List<Model> topModels = client.getModelsByScore(8.0, 10.0, 10).join();

// Get a specific model
Model model = client.getModel("gpt-4o").join();
System.out.println("Capabilities: " + model.getCapabilities());
```

### Model Verification

```java
// Verify a single model
VerificationResult result = client.verifyModel(
    "gpt-4o",
    "Can you analyze this code snippet?"
).join();

// Batch verification
List<BatchVerificationRequest> requests = Arrays.asList(
    new BatchVerificationRequest("gpt-4o", "Test prompt 1"),
    new BatchVerificationRequest("claude-3-opus", "Test prompt 2")
);
List<VerificationResult> results = client.batchVerify(requests).join();

// Check verification status
VerificationStatus status = client.getVerificationStatus(verificationId).join();
System.out.println("Status: " + status.getState());
```

### Scoring

```java
// Get model score breakdown
ScoreBreakdown score = client.getModelScore("gpt-4o").join();
System.out.printf("Overall: %.1f%n", score.getOverall());
System.out.printf("Speed: %.1f%n", score.getSpeed());
System.out.printf("Capability: %.1f%n", score.getCapability());
System.out.printf("Cost: %.1f%n", score.getCost());
```

### Configuration Export

```java
// Export verified configuration
String config = client.exportConfig(
    "opencode",
    Arrays.asList("openai", "anthropic")
).join();

// Save to file
Files.writeString(Path.of("opencode-config.json"), config);
```

### Provider Management

```java
// List providers
List<Provider> providers = client.getProviders().join();

// Check provider health
HealthStatus health = client.checkProviderHealth("openai").join();
System.out.println("Status: " + health.getStatus());

// Discover models from provider
List<DiscoveredModel> discovered = client.discoverModels("anthropic").join();
```

## Configuration

### Constructor Options

```java
// Default configuration
LLMVerifierClient client = new LLMVerifierClient("your-api-key");

// Custom base URL
LLMVerifierClient client = new LLMVerifierClient(
    "https://api.llmverifier.com",
    "your-api-key"
);

// With builder
LLMVerifierClient client = LLMVerifierClient.builder()
    .baseUrl("https://api.llmverifier.com")
    .apiKey("your-api-key")
    .timeout(Duration.ofSeconds(60))
    .maxRetries(3)
    .build();
```

## Error Handling

```java
import com.llmverifier.sdk.exceptions.*;

try {
    VerificationResult result = client.verifyModel("gpt-4o", "test").join();
} catch (CompletionException e) {
    Throwable cause = e.getCause();

    if (cause instanceof AuthenticationException) {
        System.out.println("Invalid API key");
    } else if (cause instanceof RateLimitException) {
        RateLimitException rle = (RateLimitException) cause;
        System.out.printf("Rate limited. Retry after %d seconds%n", rle.getRetryAfter());
    } else if (cause instanceof ProviderException) {
        ProviderException pe = (ProviderException) cause;
        System.out.println("Provider error: " + pe.getMessage());
    } else if (cause instanceof ValidationException) {
        ValidationException ve = (ValidationException) cause;
        System.out.println("Validation error: " + ve.getDetails());
    }
}
```

## Async Operations

All client methods return `CompletableFuture` for non-blocking operations:

```java
// Chain async operations
client.getModels()
    .thenApply(models -> models.stream()
        .filter(m -> m.getScore() >= 8.0)
        .collect(Collectors.toList()))
    .thenAccept(topModels -> {
        for (Model model : topModels) {
            System.out.println(model.getName());
        }
    })
    .exceptionally(e -> {
        System.err.println("Error: " + e.getMessage());
        return null;
    });

// Wait for multiple operations
CompletableFuture.allOf(
    client.verifyModel("gpt-4o", "test1"),
    client.verifyModel("claude-3-opus", "test2")
).join();
```

## Model Classes

### Model

```java
public class Model {
    private String id;
    private String name;
    private String provider;
    private boolean verified;
    private double score;
    private Map<String, Object> features;
    private List<String> capabilities;
    private Map<String, Object> metadata;

    // Getters and setters...
}
```

### VerificationResult

```java
public class VerificationResult {
    private String modelId;
    private boolean success;
    private double score;
    private List<String> capabilities;
    private Instant timestamp;
    private Map<String, Object> details;

    // Getters and setters...
}
```

### ScoreBreakdown

```java
public class ScoreBreakdown {
    private double overall;
    private double speed;
    private double efficiency;
    private double cost;
    private double capability;
    private double recency;

    // Getters and setters...
}
```

## Thread Safety

The `LLMVerifierClient` is thread-safe and can be shared across multiple threads. The underlying HTTP client uses connection pooling for efficient resource usage.

```java
// Recommended: Create single instance and reuse
private static final LLMVerifierClient client = new LLMVerifierClient("api-key");
```

## Cleanup

```java
// Close client when done (releases resources)
client.close();

// Or use try-with-resources
try (LLMVerifierClient client = new LLMVerifierClient("api-key")) {
    // Use client
}
```

## License

MIT License - see LICENSE file for details.

## Links

- [Documentation](https://llm-verifier.dev/docs)
- [API Reference](https://llm-verifier.dev/docs/api-reference)
- [GitHub](https://github.com/llm-verifier/llm-verifier)
