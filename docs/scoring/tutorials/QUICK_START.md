# 🚀 LLM Verifier Scoring System - Quick Start Guide

Get up and running with the LLM Verifier Scoring System in under 10 minutes!

## 📋 Prerequisites

- Go 1.21+ installed
- Basic Go programming knowledge
- Internet connection (for models.dev API)

## ⚡ 5-Minute Setup

### 1. Install & Build

```bash
# Clone the repository
git clone https://github.com/your-org/llm-verifier.git
cd llm-verifier

# Download dependencies
go mod download

# Build the project
go build ./...

# Verify build
go test ./llm-verifier/scoring/... -v -run TestScoringEngineBasic
```

### 2. Your First Score Calculation

Create a file `main.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "llm-verifier/database"
    "llm-verifier/logging"
    "llm-verifier/scoring"
)

func main() {
    fmt.Println("🚀 LLM Verifier Scoring - Quick Start")
    
    // Initialize database (in-memory for demo)
    db, err := database.New(":memory:")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Create sample provider
    provider := &database.Provider{
        Name:     "Demo Provider",
        Endpoint: "https://api.demo.com",
        IsActive: true,
    }
    db.CreateProvider(provider)

    // Create sample model
    model := &database.Model{
        ProviderID:          provider.ID,
        ModelID:             "demo-model",
        Name:                "Demo Model",
        ParameterCount:      int64Ptr(1000000000),
        IsMultimodal:        true,
        SupportsReasoning:   true,
        ReleaseDate:         timePtr(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
        VerificationStatus:  "verified",
        OverallScore:        0,
        ResponsivenessScore: 8.0,
    }
    db.CreateModel(model)

    // Initialize scoring system
    logger := &logging.Logger{}
    client, _ := scoring.NewModelsDevClient(scoring.DefaultClientConfig(), logger)
    engine := scoring.NewScoringEngine(db, client, logger)

    // Calculate score!
    ctx := context.Background()
    config := scoring.DefaultScoringConfig()
    
    score, err := engine.CalculateComprehensiveScore(ctx, "demo-model", config)
    if err != nil {
        log.Fatal(err)
    }

    // Display results
    fmt.Printf("\n📊 Results:\n")
    fmt.Printf("Model: %s\n", score.ModelName)
    fmt.Printf("Overall Score: %.1f %s\n", score.OverallScore, score.ScoreSuffix)
    fmt.Printf("\n📈 Components:\n")
    fmt.Printf("  Speed:       %.1f/10\n", score.Components.SpeedScore)
    fmt.Printf("  Efficiency:  %.1f/10\n", score.Components.EfficiencyScore)
    fmt.Printf("  Cost:        %.1f/10\n", score.Components.CostScore)
    fmt.Printf("  Capability:  %.1f/10\n", score.Components.CapabilityScore)
    fmt.Printf("  Recency:     %.1f/10\n", score.Components.RecencyScore)
}

// Helper functions
func int64Ptr(i int64) *int64 { return &i }
func timePtr(t time.Time) *time.Time { return &t }
```

### 3. Run It!

```bash
go run main.go
```

**Expected Output:**
```
🚀 LLM Verifier Scoring - Quick Start

📊 Results:
Model: Demo Model
Overall Score: 7.2 (SC:7.2)

📈 Components:
  Speed:       8.0/10
  Efficiency:  9.0/10
  Cost:        6.0/10
  Capability:  7.5/10
  Recency:     8.0/10
```

## 🎯 Next Steps (2 minutes each)

### Try Different Configurations

```go
// Speed-focused configuration
speedConfig := scoring.ScoringConfig{
    Weights: scoring.ScoreWeights{
        ResponseSpeed:     0.6,  // Emphasize speed
        ModelEfficiency:   0.1,
        CostEffectiveness: 0.1,
        Capability:        0.1,
        Recency:           0.1,
    },
}

speedScore, _ := engine.CalculateComprehensiveScore(ctx, "demo-model", speedConfig)
fmt.Printf("Speed-focused score: %.1f\n", speedScore.OverallScore)
```

### Add Model Naming

```go
// Initialize model naming
naming := scoring.NewModelNaming()

// Add score suffix
updatedName := naming.AddScoreSuffix("GPT-4", 8.5)
fmt.Printf("Updated name: %s\n", updatedName) // "GPT-4 (SC:8.5)"

// Extract score
score, found := naming.ExtractScoreFromName("GPT-4 (SC:8.5)")
if found {
    fmt.Printf("Extracted score: %.1f\n", score)
}
```

### Batch Operations

```go
// Batch calculate scores
modelIDs := []string{"demo-model-1", "demo-model-2", "demo-model-3"}
scores, _ := engine.CalculateBatchScores(ctx, modelIDs, &config.Weights)

fmt.Println("\n📋 Batch Results:")
for _, score := range scores {
    fmt.Printf("%s: %.1f %s\n", score.ModelName, score.OverallScore, score.ScoreSuffix)
}
```

## 🏗️ Architecture Overview

```mermaid
graph TD
    A[Your Code] --> B[Scoring Engine]
    B --> C[Models.dev Client]
    B --> D[Database Integration]
    C --> E[HTTP/3 + Brotli]
    D --> F[SQLite Database]
    B --> G[Score Calculation]
    G --> H[5 Components]
    H --> I[Overall Score]
    I --> J[(SC:X.X) Format]
```

## ⚙️ Configuration Options

### Default Weights
```json
{
  "response_speed": 0.25,
  "model_efficiency": 0.20, 
  "cost_effectiveness": 0.25,
  "capability": 0.20,
  "recency": 0.10
}
```

### Custom Configurations
- **Speed Focused**: Response Speed = 0.6
- **Cost Focused**: Cost Effectiveness = 0.6  
- **Capability Focused**: Capability = 0.6
- **Balanced**: All components = 0.2

## 🚀 Advanced Usage (Next 5 minutes)

### 1. Real-time Monitoring

```go
// Enable performance metrics
metrics := scoring.NewPerformanceMetrics()
metrics.StartTimer("calculation")

score, _ := engine.CalculateComprehensiveScore(ctx, "demo-model", config)

metrics.EndTimer("calculation")
fmt.Printf("Calculation took: %v\n", metrics.GetDuration("calculation"))
```

### 2. Score Analytics

```go
// Get score distribution
analytics, _ := scoringSystem.GetScoreAnalytics()
fmt.Printf("Average Score: %.1f\n", analytics.AverageScore)
fmt.Printf("Score Range: %.1f - %.1f\n", analytics.MinScore, analytics.MaxScore)
```

### 3. Custom Scoring Logic

```go
// Extend scoring with custom logic
type CustomScorer struct {
    base *scoring.ScoringEngine
}

func (cs *CustomScorer) CalculateWithBonus(modelID string) (*scoring.ComprehensiveScore, error) {
    score, err := cs.base.CalculateComprehensiveScore(context.Background(), modelID, scoring.DefaultScoringConfig())
    if err != nil {
        return nil, err
    }
    
    // Add custom bonus
    if score.Components.CapabilityScore > 8.0 {
        score.OverallScore += 0.5
        score.ScoreSuffix = fmt.Sprintf("(SC:%.1f)", score.OverallScore)
    }
    
    return score, nil
}
```

## 🔍 Debugging

### Enable Debug Logging

```go
logger := logging.NewLogger(logging.DEBUG)
engine := scoring.NewScoringEngine(db, client, logger)
```

### Common Issues

1. **Build Errors**
```bash
go clean -cache
go mod tidy
go build ./...
```

2. **Database Issues**
```bash
# Check permissions
ls -la data/

# Verify encryption key
export LLM_ENCRYPTION_KEY="your-key"
```

3. **API Connection Issues**
```bash
# Test connectivity
curl -H "Accept-Encoding: br, gzip" https://models.dev/api.json
```

## 📊 Performance Benchmarks

| Operation | Average Time | 95th Percentile |
|-----------|-------------|-----------------|
| Single Score | 150ms | 300ms |
| Batch (10 models) | 800ms | 1500ms |
| Name Update | 50ms | 100ms |

## 🎯 Next Steps

1. **Production Setup**: See [Deployment Guide](../guides/DEPLOYMENT.md)
2. **API Integration**: Check [API Reference](./REFERENCE.md)
3. **Advanced Features**: Explore [Advanced Usage](../guides/ADVANCED.md)
4. **Performance Tuning**: Read [Performance Guide](../guides/PERFORMANCE.md)

## 🤝 Getting Help

- **Documentation**: Check complete docs at `/docs/scoring/`
- **Examples**: See `/examples/scoring/`
- **Issues**: Report on GitHub Issues
- **Community**: Join our Discord server

---

**🎉 Congratulations! You're now ready to use the LLM Verifier Scoring System in production!**

*Next: [Complete Documentation](../COMPLETE_DOCUMENTATION.md) →*