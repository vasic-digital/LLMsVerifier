# LLMsVerifier Big Data Integration

LLMsVerifier includes Big Data integrations for scalable verification data storage, analytics, and pattern recognition.

## Overview

| Component | Technology | Use Case |
|-----------|------------|----------|
| Storage | MinIO | Verification results, model artifacts, logs |
| Vector DB | Qdrant | Similarity search on verification patterns |
| Streaming | Apache Flink | Real-time verification event processing |
| Analytics | Apache Iceberg | Historical verification analytics |

## Quick Start

### Using with HelixAgent Infrastructure

LLMsVerifier can use the same Big Data infrastructure as HelixAgent:

```bash
# Start from HelixAgent directory
cd /path/to/HelixAgent
docker-compose -f docker-compose.bigdata.yml up -d
```

### Standalone Configuration

```go
import (
    "llm-verifier/bigdata/storage"
    "llm-verifier/bigdata/vectordb"
    "llm-verifier/bigdata/streaming"
    "llm-verifier/bigdata/lakehouse"
)

func main() {
    ctx := context.Background()

    // MinIO for verification results
    minioConfig := storage.DefaultMinIOConfig()
    minioClient, _ := storage.NewMinIOClient(minioConfig)
    minioClient.Connect(ctx)

    // Qdrant for embedding search
    qdrantConfig := vectordb.DefaultQdrantConfig()
    qdrantClient, _ := vectordb.NewQdrantClient(qdrantConfig)
    qdrantClient.Connect(ctx)

    // Flink for streaming
    flinkConfig := streaming.DefaultFlinkConfig()
    flinkClient, _ := streaming.NewFlinkClient(flinkConfig)
    flinkClient.Connect(ctx)

    // Iceberg for analytics
    icebergConfig := lakehouse.DefaultIcebergConfig()
    icebergClient, _ := lakehouse.NewIcebergClient(icebergConfig)
    icebergClient.Connect(ctx)
}
```

## Storage (MinIO)

Store verification results, model artifacts, and logs in MinIO.

### Configuration

```go
config := storage.DefaultMinIOConfig()
config.Endpoint = "localhost:9000"
config.VerificationBucket = "llmsverifier-results"
config.ModelsBucket = "llmsverifier-models"
config.LogsBucket = "llmsverifier-logs"
```

### Usage

```go
// Store verification result
result := &storage.VerificationResult{
    ID:         uuid.New().String(),
    ProviderID: "openai",
    ModelID:    "gpt-4",
    Score:      0.95,
    Passed:     true,
    Timestamp:  time.Now(),
    Duration:   5 * time.Second,
}
err := client.StoreVerificationResult(ctx, result)

// Retrieve verification results
results, err := client.ListVerificationResults(ctx, "openai", "gpt-4", time.Now().Add(-24*time.Hour))

// Store model artifact
err = client.StoreModelArtifact(ctx, "openai", "gpt-4", "config.json", reader, size)
```

## Vector Database (Qdrant)

Enable similarity search on verification patterns and model capabilities.

### Configuration

```go
config := vectordb.DefaultQdrantConfig()
config.Host = "localhost"
config.HTTPPort = 6333
config.VerificationCollection = "verification_embeddings"
config.ModelEmbeddingsCollection = "model_embeddings"
config.VectorDimension = 1536
```

### Usage

```go
// Store verification embedding
embedding := &vectordb.VerificationEmbedding{
    ID:         uuid.New().String(),
    ProviderID: "openai",
    ModelID:    "gpt-4",
    Vector:     generateEmbedding(verificationResult),
    Score:      0.95,
    Timestamp:  time.Now(),
}
err := client.UpsertVerificationEmbedding(ctx, embedding)

// Find similar verifications
similar, err := client.SearchSimilarVerifications(ctx, queryVector, 10)

// Store model embedding for capability matching
modelEmb := &vectordb.ModelEmbedding{
    ID:           uuid.New().String(),
    ProviderID:   "openai",
    ModelID:      "gpt-4",
    ModelName:    "GPT-4 Turbo",
    Vector:       generateCapabilityEmbedding(model),
    Capabilities: model.Capabilities,
}
err = client.UpsertModelEmbedding(ctx, modelEmb)

// Find similar models
similarModels, err := client.SearchSimilarModels(ctx, capabilityVector, 5)
```

## Streaming (Flink)

Process verification events in real-time.

### Configuration

```go
config := streaming.DefaultFlinkConfig()
config.JobManagerHost = "localhost"
config.RESTURL = "http://localhost:8082"
```

### Usage

```go
// Get cluster status
overview, err := client.GetClusterOverview(ctx)
fmt.Printf("Running jobs: %d\n", overview.JobsRunning)

// Submit verification streaming job
streamConfig := streaming.DefaultVerificationStreamConfig()
streamConfig.InputTopic = "llmsverifier.verification.requests"
streamConfig.OutputTopic = "llmsverifier.verification.results"

jobID, err := client.SubmitVerificationJob(ctx, jarID, streamConfig)

// Monitor job
job, err := client.GetJob(ctx, jobID)
fmt.Printf("Job state: %s\n", job.State)
```

## Analytics (Iceberg)

Query historical verification data with time-travel capabilities.

### Configuration

```go
config := lakehouse.DefaultIcebergConfig()
config.CatalogURI = "http://localhost:8181"
config.Warehouse = "s3://llmsverifier-iceberg/warehouse"
config.Namespace = "llmsverifier"
```

### Usage

```go
// Ensure tables exist
err := client.EnsureVerificationTables(ctx)

// List tables
tables, err := client.ListTables(ctx)

// Get table info
info, err := client.GetTable(ctx, "verification_results")
fmt.Printf("Table location: %s\n", info.Location)

// Check table exists
exists, err := client.TableExists(ctx, "verification_results")
```

### Table Schemas

**verification_results:**
| Column | Type | Description |
|--------|------|-------------|
| id | string | Unique verification ID |
| provider_id | string | Provider identifier |
| model_id | string | Model identifier |
| score | double | Verification score |
| passed | boolean | Whether verification passed |
| duration_ms | long | Duration in milliseconds |
| error_message | string | Error message if failed |
| timestamp | timestamp | Verification timestamp |
| details_json | string | JSON details |

**model_metrics:**
| Column | Type | Description |
|--------|------|-------------|
| id | string | Unique metric ID |
| provider_id | string | Provider identifier |
| model_id | string | Model identifier |
| metric_name | string | Metric name |
| metric_value | double | Metric value |
| timestamp | timestamp | Collection timestamp |
| tags_json | string | JSON tags |

## Use Cases

### 1. Verification Result Storage and Search

```go
// Store result in MinIO
minioClient.StoreVerificationResult(ctx, result)

// Generate and store embedding
embedding := generateEmbedding(result)
qdrantClient.UpsertVerificationEmbedding(ctx, &VerificationEmbedding{
    Vector: embedding,
    // ...
})

// Write to Iceberg for analytics
icebergClient.AppendToTable(ctx, "verification_results", result)
```

### 2. Find Similar Verification Patterns

```go
// When a new verification fails, find similar past failures
queryVector := generateEmbedding(failedResult)
similar, _ := qdrantClient.SearchSimilarVerifications(ctx, queryVector, 10)

for _, s := range similar {
    if s.Score < 0.7 {
        fmt.Printf("Similar failure found: %s/%s (score: %.2f)\n",
            s.ProviderID, s.ModelID, s.Score)
    }
}
```

### 3. Model Capability Matching

```go
// Find models with similar capabilities to a reference model
refModel := getModelInfo("openai", "gpt-4")
capVector := generateCapabilityEmbedding(refModel)

similar, _ := qdrantClient.SearchSimilarModels(ctx, capVector, 5)
for _, m := range similar {
    fmt.Printf("Similar model: %s (similarity: %.2f)\n",
        m.ModelName, m.Similarity)
}
```

## Running Tests

```bash
cd llm-verifier
go test -v ./bigdata/...
```

## Integration with HelixAgent

LLMsVerifier Big Data integration is designed to work seamlessly with HelixAgent:

1. **Shared Infrastructure**: Use the same MinIO, Qdrant, Flink, and Iceberg instances
2. **Compatible Schemas**: Data formats are compatible for cross-system queries
3. **Unified Configuration**: Configure once in HelixAgent, use in both systems

```yaml
# configs/bigdata.yaml (shared)
minio:
  endpoint: "localhost:9000"
  # LLMsVerifier buckets
  llmsverifier_results_bucket: "llmsverifier-results"
  llmsverifier_models_bucket: "llmsverifier-models"
  # HelixAgent buckets
  helixagent_debates_bucket: "helixagent-debates"
```
