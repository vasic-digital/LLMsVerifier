# LLM Scoring System Documentation

## Overview

The LLM Scoring System provides comprehensive, automated scoring for Large Language Models based on multiple performance metrics including response speed, model efficiency, cost-effectiveness, capabilities, and recency. Each verified model receives a score from 0-10, displayed as a suffix in the model name (e.g., `GPT-4 (SC:8.5)`).

## Features

### Core Capabilities
- **Automated Scoring**: Calculates comprehensive scores using 5 key metrics
- **Real-time Updates**: Scores update automatically as conditions change
- **HTTP/3 & Brotli Support**: Fast API calls with modern protocols
- **models.dev Integration**: Fetches comprehensive model data from external sources
- **Batch Processing**: Score multiple models simultaneously
- **Background Sync**: Automatic synchronization with external data sources

### Scoring Components

#### 1. Response Speed Score (25% weight)
- **Average Response Time**: Measured across multiple test requests
- **P95 Latency**: 95th percentile response time
- **Throughput**: Requests per second capability
- **Consistency**: Standard deviation of response times

#### 2. Model Efficiency Score (20% weight)
- **Parameter Count**: Number of model parameters (smaller is better for efficiency)
- **Context Window**: How effectively the model uses its context
- **Memory Efficiency**: Tokens processed per parameter
- **Architecture Efficiency**: Model architecture optimizations

#### 3. Cost Effectiveness Score (25% weight)
- **Input Token Cost**: Price per million input tokens
- **Output Token Cost**: Price per million output tokens
- **Cache Pricing**: Cached token pricing
- **Additional Costs**: Audio, vision, reasoning costs
- **Value Ratio**: Performance per dollar spent

#### 4. Capability Score (20% weight)
- **Feature Support**: Tool calling, reasoning, multimodal capabilities
- **Code Capabilities**: Programming language support, debugging, optimization
- **Context Handling**: Maximum context length and utilization
- **Reliability**: Uptime and error rate

#### 5. Recency Score (10% weight)
- **Release Date**: How recent the model is
- **Update Frequency**: Frequency of updates and improvements
- **Maintenance Status**: Whether the model is actively maintained
- **Community Support**: Open source vs proprietary

## Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                    Scoring System                           │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │   API Layer     │  │  Scoring Engine │  │   Database   │ │
│  │                 │  │                 │  │              │ │
│  │ • REST Endpoints│  │ • Score Calc    │  │ • Model Data │ │
│  │ • Batch Ops     │  │ • Weighting     │  │ • Scores     │ │
│  │ • Config Mgmt   │  │ • Normalization │  │ • Metrics    │ │
│  └────────┬────────┘  └────────┬────────┘  └──────┬───────┘ │
│           │                    │                   │         │
│  ┌────────▼────────┐  ┌────────▼────────┐  ┌──────▼───────┐ │
│  │  models.dev     │  │  HTTP/3 Client  │  │  Background  │ │
│  │  Integration    │  │  Brotli Support │  │  Workers     │ │
│  │                 │  │                 │  │              │ │
│  │ • Model Data    │  │ • Fast API      │  │ • Auto Sync  │ │
│  │ • Pricing Info  │  │ • Compression   │  │ • Monitoring │ │
│  │ • Capabilities  │  │ • Fallback      │  │ • Recalc     │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Data Flow

1. **Model Discovery**: System discovers models through verification process
2. **Data Collection**: Fetches model data from models.dev API with HTTP/3 + Brotli
3. **Performance Monitoring**: Collects real-time performance metrics
4. **Score Calculation**: Applies weighted algorithm to calculate comprehensive score
5. **Name Update**: Updates model names with score suffix
6. **Storage**: Persists scores and metrics to database
7. **Monitoring**: Tracks changes and triggers recalculation when needed

## API Reference

### Model Scoring Endpoints

#### Calculate Model Score
```http
POST /api/v1/models/{model_id}/score/calculate
Content-Type: application/json

{
  "configuration": {
    "weights": {
      "speed": 0.25,
      "efficiency": 0.20,
      "cost": 0.25,
      "capability": 0.20,
      "recency": 0.10
    }
  },
  "force_recalculation": false
}
```

**Response:**
```json
{
  "message": "Score calculated successfully",
  "model_id": "gpt-4",
  "model_name": "GPT-4",
  "formatted_name": "GPT-4 (SC:8.5)",
  "overall_score": 8.5,
  "score_suffix": "(SC:8.5)",
  "components": {
    "speed_score": 8.2,
    "efficiency_score": 7.8,
    "cost_score": 6.5,
    "capability_score": 9.1,
    "recency_score": 8.9
  },
  "last_calculated": "2024-01-15T10:30:00Z",
  "calculation_hash": "a1b2c3d4"
}
```

#### Get Model Score
```http
GET /api/v1/models/{model_id}/score
```

#### Batch Calculate Scores
```http
POST /api/v1/models/scores/batch
Content-Type: application/json

{
  "model_ids": ["gpt-4", "claude-3", "gemini-pro"],
  "configuration": {...},
  "async": true
}
```

#### Compare Models
```http
GET /api/v1/models/scores/compare?models=gpt-4,claude-3,gemini-pro
```

#### Get Model Rankings
```http
GET /api/v1/models/scores/ranking?category=overall&limit=50
```

### Model Naming Endpoints

#### Add Score Suffix
```http
POST /api/v1/models/naming/add-suffix
Content-Type: application/json

{
  "model_name": "GPT-4 Turbo",
  "score": 8.7
}
```

#### Batch Update Model Names
```http
POST /api/v1/models/naming/batch-update
Content-Type: application/json

{
  "model_scores": {
    "GPT-4": 8.5,
    "Claude-3": 7.8,
    "Gemini Pro": 7.2
  }
}
```

### External Data Integration

#### Sync with models.dev
```http
POST /api/v1/scoring/sync-models-dev
Content-Type: application/json

{
  "provider_id": "openai",
  "model_id": "gpt-4",
  "force_sync": false
}
```

#### Fetch models.dev Data
```http
GET /api/v1/scoring/models-dev/{model_id}
```

## Configuration

### Default Scoring Weights
```json
{
  "weights": {
    "speed": 0.25,
    "efficiency": 0.20,
    "cost": 0.25,
    "capability": 0.20,
    "recency": 0.10
  },
  "normalization": {
    "min_score": 0.0,
    "max_score": 10.0
  },
  "cache_duration": "1h"
}
```

### System Configuration
```json
{
  "auto_sync_interval": "6h",
  "score_recalc_interval": "1h",
  "performance_window": "24h",
  "max_concurrent_calcs": 10,
  "enable_background_sync": true,
  "enable_score_monitoring": true,
  "score_change_threshold": 0.5
}
```

## Score Interpretation

### Score Ranges
- **SC:9.0-10.0**: Exceptional - Top-tier models with excellent performance across all metrics
- **SC:8.0-8.9**: Excellent - High-performing models with minor limitations
- **SC:7.0-7.9**: Very Good - Solid performers with some trade-offs
- **SC:6.0-6.9**: Good - Capable models with notable limitations
- **SC:5.0-5.9**: Average - Basic functionality with significant trade-offs
- **SC:4.0-4.9**: Below Average - Models with major limitations
- **SC:3.0-3.9**: Poor - Models with severe limitations
- **SC:0.0-2.9**: Unacceptable - Models not suitable for production use

### Component Scores
Each component (Speed, Efficiency, Cost, Capability, Recency) is scored individually on a 0-10 scale:
- **Speed**: Lower response times and higher throughput score better
- **Efficiency**: Smaller models with good performance score better
- **Cost**: Lower costs per token score better
- **Capability**: More features and better reliability score better
- **Recency**: Newer, well-maintained models score better

## Integration Guide

### Basic Integration
```go
import (
    "llm-verifier/scoring"
    "llm-verifier/database"
    "llm-verifier/logging"
)

// Initialize database
db, err := database.New("/path/to/database.db")
if err != nil {
    log.Fatal(err)
}

// Create logger
logger := logging.NewLogger()

// Configure scoring system
config := scoring.DefaultScoringSystemConfig()

// Initialize scoring system
scoringSystem, err := scoring.NewScoringSystem(db, logger, config)
if err != nil {
    log.Fatal(err)
}

// Start background processes
ctx := context.Background()
if err := scoringSystem.Start(ctx); err != nil {
    log.Fatal(err)
}

// Calculate score for a model
score, err := scoringSystem.CalculateModelScore(ctx, "gpt-4", nil)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Model: %s, Score: %s\n", score.ModelName, score.ScoreSuffix)
```

### Advanced Integration with Custom Weights
```go
// Custom scoring configuration
customConfig := scoring.ScoringConfig{
    Weights: struct {
        Speed      float64 `json:"speed"`
        Efficiency float64 `json:"efficiency"`
        Cost       float64 `json:"cost"`
        Capability float64 `json:"capability"`
        Recency    float64 `json:"recency"`
    }{
        Speed:      0.3,  // Prioritize speed
        Efficiency: 0.15,
        Cost:       0.2,
        Capability: 0.25,
        Recency:    0.1,
    },
}

// Calculate with custom weights
score, err := scoringSystem.CalculateModelScore(ctx, "gpt-4", &customConfig)
```

### Batch Processing
```go
modelIDs := []string{"gpt-4", "claude-3", "gemini-pro", "llama-2"}
scores, err := scoringSystem.BatchCalculateScores(ctx, modelIDs, nil)
if err != nil {
    log.Fatal(err)
}

for _, score := range scores {
    fmt.Printf("%s: %s\n", score.ModelName, score.ScoreSuffix)
}
```

### Model Naming Integration
```go
// Update model names with scores
scores := map[string]float64{
    "gpt-4":    8.5,
    "claude-3": 7.8,
}

err := scoringSystem.BatchUpdateModelNamesWithScores(scores)
if err != nil {
    log.Fatal(err)
}
```

## Performance Optimization

### HTTP/3 and Brotli Support
The system automatically uses HTTP/3 and Brotli compression when available:
- **HTTP/3**: Faster connection establishment and multiplexing
- **Brotli**: Better compression ratios than gzip
- **Fallback**: Automatically falls back to HTTP/2 and gzip

### Caching Strategy
- **Score Caching**: Scores are cached for 1 hour by default
- **Performance Metrics**: Cached for performance analysis
- **External Data**: models.dev data cached with TTL

### Concurrent Processing
- **Batch Operations**: Configurable concurrency limits
- **Background Workers**: Non-blocking background processes
- **Rate Limiting**: Respects API rate limits

## Monitoring and Alerting

### Score Change Monitoring
- **Threshold Alerts**: Notifies when scores change significantly (>0.5 points)
- **Trend Analysis**: Tracks score changes over time
- **Component Changes**: Identifies which components changed

### Performance Monitoring
- **API Response Times**: Monitors external API performance
- **Calculation Performance**: Tracks score calculation times
- **Error Rates**: Monitors system health

### Health Checks
- **Database Connectivity**: Ensures database availability
- **External API Status**: Monitors models.dev availability
- **Background Process Status**: Ensures workers are running

## Testing

### Unit Tests
```bash
go test ./scoring -v
```

### Integration Tests
```bash
go test ./scoring -tags=integration -v
```

### Performance Tests
```bash
go test ./scoring -bench=. -benchmem
```

### Test Coverage
```bash
go test ./scoring -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Troubleshooting

### Common Issues

#### Score Calculation Failures
- **Check Model Data**: Ensure model exists in database
- **Verify API Connectivity**: Check models.dev API status
- **Review Logs**: Look for error messages in logs

#### Model Name Updates
- **Check Permissions**: Ensure database write permissions
- **Verify Score Range**: Scores must be between 0-10
- **Check Existing Names**: May conflict with existing suffixes

#### Performance Issues
- **Database Optimization**: Ensure proper indexes
- **API Rate Limits**: Check external API rate limits
- **Memory Usage**: Monitor memory consumption

### Debug Mode
Enable debug logging to troubleshoot issues:
```go
logger := logging.NewLoggerWithLevel(logging.Debug)
scoringSystem, err := scoring.NewScoringSystem(db, logger, config)
```

### Log Analysis
Key log messages to watch for:
- `Score calculated successfully`: Normal operation
- `HTTP/3 request failed, falling back`: Network fallback
- `Failed to calculate model score`: Calculation errors
- `Background sync completed`: Sync status

## Security Considerations

### API Security
- **Authentication**: All endpoints require authentication
- **Rate Limiting**: Prevents abuse
- **Input Validation**: Sanitizes all inputs

### Data Protection
- **Encryption**: Database encryption for sensitive data
- **Access Control**: Role-based access control
- **Audit Logging**: Tracks all changes

### External API Security
- **HTTPS**: All external calls use HTTPS
- **Certificate Validation**: Validates SSL certificates
- **Timeout Handling**: Prevents hanging connections

## Future Enhancements

### Planned Features
- **Machine Learning Optimization**: Use ML to optimize scoring weights
- **User Preference Integration**: Allow custom scoring weights
- **A/B Testing Support**: Compare different scoring algorithms
- **Predictive Scoring**: Predict future performance based on trends
- **Multi-language Support**: Localized scoring explanations

### Performance Improvements
- **Distributed Processing**: Support for distributed score calculation
- **Advanced Caching**: Redis-based caching for better performance
- **Streaming Updates**: Real-time score updates via WebSocket
- **Edge Computing**: Process scores closer to data sources

### Integration Expansions
- **Additional Data Sources**: Integrate with more model databases
- **Custom Metrics**: Support for user-defined metrics
- **Third-party Integrations**: Slack, Discord, email notifications
- **Export Formats**: CSV, JSON, Excel exports

## Contributing

### Development Setup
```bash
git clone https://github.com/vasic-digital/LLMsVerifier.git
cd LLMsVerifier/llm-verifier
go mod download
go test ./scoring
```

### Code Style
- Follow Go best practices
- Use meaningful variable names
- Add comprehensive comments
- Write unit tests for new features

### Pull Request Process
1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass
5. Submit pull request with detailed description

## Support

### Documentation
- [API Documentation](llm-verifier/docs/API_DOCUMENTATION.md)
- [Integration Guide](llm-verifier/docs/INTEGRATION_GUIDE.md)
- [Troubleshooting Guide](llm-verifier/docs/TROUBLESHOOTING.md)

### Community
- [GitHub Issues](https://github.com/vasic-digital/LLMsVerifier/issues)
- [GitHub Discussions](https://github.com/vasic-digital/LLMsVerifier/discussions)
- [Discord Community](https://discord.gg/llm-verifier)

### Commercial Support
For enterprise support and custom integrations, contact: support@llm-verifier.com